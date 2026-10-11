package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"memo-system/models"
	"memo-system/utils"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// NoteCreateRequest 创建笔记请求
type NoteCreateRequest struct {
	Title    string `json:"title" binding:"required,max=200"`
	Content  string `json:"content"`
	TagIDs   []uint `json:"tag_ids"`
	IsPublic bool   `json:"is_public"`
}

// NoteUpdateRequest 更新笔记请求
type NoteUpdateRequest struct {
	Title    string `json:"title" binding:"max=200"`
	Content  string `json:"content"`
	TagIDs   []uint `json:"tag_ids"`
	IsPublic *bool  `json:"is_public"`
}

// generateShareToken 生成分享令牌
func generateShareToken() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// stripHTML 去除HTML标签用于生成摘要
func stripHTML(html string) string {
	// 简单的HTML标签去除
	result := html
	// 移除script和style内容
	for {
		start := strings.Index(strings.ToLower(result), "<script")
		if start == -1 {
			break
		}
		end := strings.Index(strings.ToLower(result[start:]), "</script>")
		if end == -1 {
			break
		}
		end += start + len("</script>")
		result = result[:start] + result[end:]
	}
	for {
		start := strings.Index(strings.ToLower(result), "<style")
		if start == -1 {
			break
		}
		end := strings.Index(strings.ToLower(result[start:]), "</style>")
		if end == -1 {
			break
		}
		end += start + len("</style>")
		result = result[:start] + result[end:]
	}
	// 移除所有HTML标签
	for {
		start := strings.Index(result, "<")
		if start == -1 {
			break
		}
		end := strings.Index(result[start:], ">")
		if end == -1 {
			break
		}
		end += start + 1
		result = result[:start] + " " + result[end:]
	}
	result = strings.TrimSpace(result)
	// 将多个空格替换为单个空格
	for strings.Contains(result, "  ") {
		result = strings.ReplaceAll(result, "  ", " ")
	}
	return result
}

// CreateNote 创建笔记
func CreateNote(c *gin.Context) {
	userID := c.GetUint("userID")

	var req NoteCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误: " + err.Error()})
		return
	}

	// 生成摘要
	plainText := stripHTML(req.Content)
	summary := plainText
	if len(summary) > 200 {
		summary = summary[:200] + "..."
	}

	note := models.Note{
		UserID:     userID,
		Title:      req.Title,
		Content:    req.Content,
		Summary:    summary,
		IsPublic:   req.IsPublic,
		ShareToken: generateShareToken(),
	}

	// 处理标签
	if len(req.TagIDs) > 0 {
		var tags []models.Tag
		utils.DB.Where("id IN ? AND user_id = ?", req.TagIDs, userID).Find(&tags)
		note.Tags = tags
	}

	if err := utils.DB.Create(&note).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建笔记失败: " + err.Error()})
		return
	}

	// 预加载标签
	utils.DB.Preload("Tags").First(&note, note.ID)

	c.JSON(http.StatusCreated, gin.H{
		"message": "创建成功",
		"note":    note,
	})
}

// GetNote 获取单条笔记
func GetNote(c *gin.Context) {
	userID := c.GetUint("userID")
	id := c.Param("id")

	var note models.Note
	if err := utils.DB.Preload("Tags").First(&note, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "笔记不存在"})
		return
	}

	// 权限检查：只能查看自己的笔记
	if note.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权访问此笔记"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"note": note})
}

// ListNotes 获取笔记列表
func ListNotes(c *gin.Context) {
	userID := c.GetUint("userID")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	keyword := c.Query("keyword")
	tagID := c.Query("tag_id")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	query := utils.DB.Model(&models.Note{}).Where("user_id = ?", userID)

	// 关键词搜索
	if keyword != "" {
		query = query.Where("title LIKE ? OR content LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}

	// 标签筛选
	if tagID != "" {
		query = query.Joins("JOIN note_tags ON note_tags.note_id = notes.id").
			Where("note_tags.tag_id = ?", tagID)
	}

	var total int64
	query.Count(&total)

	var notes []models.Note
	offset := (page - 1) * pageSize
	err := query.Preload("Tags").
		Order("updated_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&notes).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取笔记列表失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"notes": notes,
		"pagination": gin.H{
			"page":       page,
			"page_size":  pageSize,
			"total":      total,
			"total_page": (total + int64(pageSize) - 1) / int64(pageSize),
		},
	})
}

// UpdateNote 更新笔记
func UpdateNote(c *gin.Context) {
	userID := c.GetUint("userID")
	id := c.Param("id")

	var note models.Note
	if err := utils.DB.First(&note, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "笔记不存在"})
		return
	}

	if note.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权修改此笔记"})
		return
	}

	var req NoteUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误: " + err.Error()})
		return
	}

	updates := make(map[string]interface{})
	if req.Title != "" {
		updates["title"] = req.Title
	}
	if req.Content != "" || (req.Content == "" && strings.TrimSpace(stripHTML(req.Content)) == "") {
		updates["content"] = req.Content
		plainText := stripHTML(req.Content)
		summary := plainText
		if len(summary) > 200 {
			summary = summary[:200] + "..."
		}
		updates["summary"] = summary
	}
	if req.IsPublic != nil {
		updates["is_public"] = *req.IsPublic
	}

	if err := utils.DB.Model(&note).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新笔记失败"})
		return
	}

	// 更新标签关联
	if req.TagIDs != nil {
		var tags []models.Tag
		if len(req.TagIDs) > 0 {
			utils.DB.Where("id IN ? AND user_id = ?", req.TagIDs, userID).Find(&tags)
		}
		utils.DB.Model(&note).Association("Tags").Replace(tags)
	}

	// 重新加载笔记
	utils.DB.Preload("Tags").First(&note, note.ID)

	c.JSON(http.StatusOK, gin.H{
		"message": "更新成功",
		"note":    note,
	})
}

// DeleteNote 软删除笔记（放入回收站）
func DeleteNote(c *gin.Context) {
	userID := c.GetUint("userID")
	id := c.Param("id")

	var note models.Note
	if err := utils.DB.First(&note, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "笔记不存在"})
		return
	}

	if note.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权删除此笔记"})
		return
	}

	if err := utils.DB.Delete(&note).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "笔记已移至回收站"})
}

// GetTrashList 获取回收站列表
func GetTrashList(c *gin.Context) {
	userID := c.GetUint("userID")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	query := utils.DB.Unscoped().Model(&models.Note{}).
		Where("user_id = ? AND deleted_at IS NOT NULL", userID)

	var total int64
	query.Count(&total)

	var notes []models.Note
	offset := (page - 1) * pageSize
	err := query.Preload("Tags").
		Order("deleted_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&notes).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取回收站列表失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"notes": notes,
		"pagination": gin.H{
			"page":       page,
			"page_size":  pageSize,
			"total":      total,
			"total_page": (total + int64(pageSize) - 1) / int64(pageSize),
		},
	})
}

// RestoreNote 恢复回收站中的笔记
func RestoreNote(c *gin.Context) {
	userID := c.GetUint("userID")
	id := c.Param("id")

	var note models.Note
	if err := utils.DB.Unscoped().First(&note, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "笔记不存在"})
		return
	}

	if note.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权操作此笔记"})
		return
	}

	// 恢复笔记（清除deleted_at）
	if err := utils.DB.Unscoped().Model(&note).Update("deleted_at", gorm.Expr("NULL")).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "恢复失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "笔记已恢复"})
}

// PermanentDelete 永久删除笔记
func PermanentDelete(c *gin.Context) {
	userID := c.GetUint("userID")
	id := c.Param("id")

	var note models.Note
	if err := utils.DB.Unscoped().First(&note, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "笔记不存在"})
		return
	}

	if note.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权操作此笔记"})
		return
	}

	// 先清除标签关联
	utils.DB.Model(&note).Association("Tags").Clear()

	// 永久删除
	if err := utils.DB.Unscoped().Delete(&note).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "永久删除失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "笔记已永久删除"})
}

// GetPublicNote 通过分享令牌获取公开笔记
func GetPublicNote(c *gin.Context) {
	token := c.Param("token")

	var note models.Note
	if err := utils.DB.Preload("Tags").Preload("User").
		Where("share_token = ? AND is_public = ?", token, true).
		First(&note).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "笔记不存在或未公开分享"})
		return
	}

	// 增加浏览量
	utils.DB.Model(&note).UpdateColumn("view_count", gorm.Expr("view_count + 1"))

	c.JSON(http.StatusOK, gin.H{
		"note": gin.H{
			"id":         note.ID,
			"title":      note.Title,
			"content":    note.Content,
			"summary":    note.Summary,
			"view_count": note.ViewCount + 1,
			"created_at": note.CreatedAt,
			"updated_at": note.UpdatedAt,
			"tags":       note.Tags,
			"author":     note.User.Username,
		},
	})
}

// TogglePublic 切换笔记公开状态
func TogglePublic(c *gin.Context) {
	userID := c.GetUint("userID")
	id := c.Param("id")

	var note models.Note
	if err := utils.DB.First(&note, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "笔记不存在"})
		return
	}

	if note.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权操作此笔记"})
		return
	}

	note.IsPublic = !note.IsPublic
	utils.DB.Save(&note)

	c.JSON(http.StatusOK, gin.H{
		"message":     "状态已更新",
		"is_public":   note.IsPublic,
		"share_token": note.ShareToken,
		"share_url":   "/share/" + note.ShareToken,
	})
}
