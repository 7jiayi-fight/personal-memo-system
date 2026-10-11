package handlers

import (
	"fmt"
	"memo-system/models"
	"memo-system/utils"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

const maxUploadSize = 10 * 1024 * 1024 // 10MB

var allowedImageTypes = map[string]bool{
	"image/jpeg":    true,
	"image/png":     true,
	"image/gif":     true,
	"image/webp":    true,
	"image/svg+xml": true,
	"image/bmp":     true,
}

// UploadImage 上传图片（用于富文本编辑器）
func UploadImage(c *gin.Context) {
	userID := c.GetUint("userID")

	// 限制上传大小
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUploadSize)

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "获取文件失败: " + err.Error()})
		return
	}

	// 检查文件大小
	if file.Size > maxUploadSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件大小不能超过10MB"})
		return
	}

	// 检查文件类型
	ext := filepath.Ext(file.Filename)
	contentType := file.Header.Get("Content-Type")
	if !allowedImageTypes[contentType] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不支持的图片格式"})
		return
	}

	// 生成文件名
	timestamp := time.Now().UnixNano()
	newFilename := fmt.Sprintf("user_%d_%d%s", userID, timestamp, ext)
	savePath := filepath.Join("static/uploads", newFilename)

	// 保存文件
	if err := c.SaveUploadedFile(file, savePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存文件失败: " + err.Error()})
		return
	}

	// 记录文件信息
	uploadedFile := models.UploadedFile{
		UserID:   userID,
		FileName: file.Filename,
		FilePath: "/uploads/" + newFilename,
		FileSize: file.Size,
		FileType: contentType,
		IsPublic: false,
	}

	if err := utils.DB.Create(&uploadedFile).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "记录文件信息失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "上传成功",
		"url":     "/uploads/" + newFilename,
		"file_id": uploadedFile.ID,
	})
}

// ListFiles 获取用户上传的文件列表
func ListFiles(c *gin.Context) {
	userID := c.GetUint("userID")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	query := utils.DB.Model(&models.UploadedFile{}).Where("user_id = ?", userID)

	var total int64
	query.Count(&total)

	var files []models.UploadedFile
	offset := (page - 1) * pageSize
	err := query.Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&files).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取文件列表失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"files": files,
		"pagination": gin.H{
			"page":       page,
			"page_size":  pageSize,
			"total":      total,
			"total_page": (total + int64(pageSize) - 1) / int64(pageSize),
		},
	})
}

// DeleteFile 删除文件
func DeleteFile(c *gin.Context) {
	userID := c.GetUint("userID")
	id := c.Param("id")

	var file models.UploadedFile
	if err := utils.DB.First(&file, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "文件不存在"})
		return
	}

	if file.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权删除此文件"})
		return
	}

	// 软删除
	if err := utils.DB.Delete(&file).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除文件失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "文件已删除"})
}
