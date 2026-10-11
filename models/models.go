package models

import (
	"time"

	"gorm.io/gorm"
)

// User 用户模型
type User struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Username  string         `gorm:"size:50;uniqueIndex;not null" json:"username"`
	Password  string         `gorm:"size:255;not null" json:"-"`
	Email     string         `gorm:"size:100" json:"email"`
	Avatar    string         `gorm:"size:255" json:"avatar"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	Notes     []Note         `gorm:"foreignKey:UserID" json:"-"`
	Tags      []Tag          `gorm:"foreignKey:UserID" json:"-"`
}

// Note 笔记模型
type Note struct {
	ID         uint           `gorm:"primaryKey" json:"id"`
	UserID     uint           `gorm:"index;not null" json:"user_id"`
	Title      string         `gorm:"size:200;not null" json:"title"`
	Content    string         `gorm:"type:text" json:"content"`
	Summary    string         `gorm:"size:500" json:"summary"`
	IsPublic   bool           `gorm:"default:false;index" json:"is_public"`
	ShareToken string         `gorm:"size:64;uniqueIndex" json:"share_token"`
	ViewCount  int            `gorm:"default:0" json:"view_count"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
	Tags       []Tag          `gorm:"many2many:note_tags;" json:"tags,omitempty"`
	User       User           `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// Tag 标签模型
type Tag struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	UserID    uint           `gorm:"index;not null" json:"user_id"`
	Name      string         `gorm:"size:50;not null" json:"name"`
	Color     string         `gorm:"size:20;default:#3b82f6" json:"color"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	Notes     []Note         `gorm:"many2many:note_tags;" json:"-"`
}

// NoteTag 笔记标签关联表
type NoteTag struct {
	NoteID uint `gorm:"primaryKey"`
	TagID  uint `gorm:"primaryKey"`
}

// UploadedFile 文件上传记录
type UploadedFile struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	UserID    uint           `gorm:"index;not null" json:"user_id"`
	FileName  string         `gorm:"size:255;not null" json:"file_name"`
	FilePath  string         `gorm:"size:500;not null" json:"file_path"`
	FileSize  int64          `json:"file_size"`
	FileType  string         `gorm:"size:50" json:"file_type"`
	IsPublic  bool           `gorm:"default:false" json:"is_public"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName 指定表名
func (NoteTag) TableName() string {
	return "note_tags"
}
