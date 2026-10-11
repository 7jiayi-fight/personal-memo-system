package handlers

import (
	"memo-system/models"
	"memo-system/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

// TagCreateRequest 创建标签请求
type TagCreateRequest struct {
	Name  string `json:"name" binding:"required,max=50"`
	Color string `json:"color"`
}

// TagUpdateRequest 更新标签请求
type TagUpdateRequest struct {
	Name  string `json:"name" binding:"max=50"`
	Color string `json:"color"`
}

// CreateTag 创建标签
func CreateTag(c *gin.Context) {
	userID := c.GetUint("userID")

	var req TagCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
		return
	}

	// 检查同名标签是否已存在
	var existingTag models.Tag
	if err := utils.DB.Where("user_id = ? AND name = ?", userID, req.Name).First(&existingTag).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "标签已存在"})
		return
	}

	color := req.Color
	if color == "" {
		color = "#3b82f6"
	}

	tag := models.Tag{
		UserID: userID,
		Name:   req.Name,
		Color:  color,
	}

	if err := utils.DB.Create(&tag).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建标签失败"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "创建成功",
		"tag":     tag,
	})
}

// ListTags 获取标签列表
func ListTags(c *gin.Context) {
	userID := c.GetUint("userID")

	var tags []models.Tag
	if err := utils.DB.Where("user_id = ?", userID).Order("created_at DESC").Find(&tags).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取标签列表失败"})
		return
	}

	// 统计每个标签的笔记数量
	type tagWithCount struct {
		models.Tag
		NoteCount int64 `json:"note_count"`
	}

	var result []tagWithCount
	for _, tag := range tags {
		var count int64
		utils.DB.Table("note_tags").Where("tag_id = ?", tag.ID).Count(&count)
		result = append(result, tagWithCount{Tag: tag, NoteCount: count})
	}

	c.JSON(http.StatusOK, gin.H{"tags": result})
}

// UpdateTag 更新标签
func UpdateTag(c *gin.Context) {
	userID := c.GetUint("userID")
	id := c.Param("id")

	var tag models.Tag
	if err := utils.DB.First(&tag, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "标签不存在"})
		return
	}

	if tag.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权修改此标签"})
		return
	}

	var req TagUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
		return
	}

	if req.Name != "" {
		tag.Name = req.Name
	}
	if req.Color != "" {
		tag.Color = req.Color
	}

	if err := utils.DB.Save(&tag).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新标签失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "更新成功",
		"tag":     tag,
	})
}

// DeleteTag 删除标签
func DeleteTag(c *gin.Context) {
	userID := c.GetUint("userID")
	id := c.Param("id")

	var tag models.Tag
	if err := utils.DB.First(&tag, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "标签不存在"})
		return
	}

	if tag.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权删除此标签"})
		return
	}

	// 清除标签与笔记的关联
	utils.DB.Exec("DELETE FROM note_tags WHERE tag_id = ?", id)

	if err := utils.DB.Delete(&tag).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除标签失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "标签已删除"})
}
