package main

import (
	"log"
	"memo-system/handlers"
	"memo-system/middleware"
	"memo-system/utils"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

func main() {
	// 确保数据目录存在
	os.MkdirAll("data", 0755)
	os.MkdirAll("static/uploads", 0755)

	// 初始化数据库
	utils.InitDB("data/memo.db")

	// 设置Gin模式
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// 中间件
	r.Use(middleware.CORSMiddleware())

	// 静态文件
	r.Static("/static", "./static")
	r.Static("/uploads", "./static/uploads")

	// HTML模板
	r.LoadHTMLGlob("templates/*")

	// 页面路由
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", nil)
	})
	r.GET("/login", func(c *gin.Context) {
		c.HTML(http.StatusOK, "login.html", nil)
	})
	r.GET("/register", func(c *gin.Context) {
		c.HTML(http.StatusOK, "register.html", nil)
	})
	r.GET("/share/:token", func(c *gin.Context) {
		c.HTML(http.StatusOK, "share.html", gin.H{"token": c.Param("token")})
	})

	// API路由组 - 无需认证
	api := r.Group("/api")
	{
		api.POST("/register", handlers.Register)
		api.POST("/login", handlers.Login)
		api.GET("/share/:token", handlers.GetPublicNote)
	}

	// API路由组 - 需要认证
	authAPI := r.Group("/api")
	authAPI.Use(middleware.JWTAuth())
	{
		// 用户
		authAPI.POST("/logout", handlers.Logout)
		authAPI.GET("/user", handlers.GetCurrentUser)

		// 笔记
		authAPI.POST("/notes", handlers.CreateNote)
		authAPI.GET("/notes", handlers.ListNotes)
		authAPI.GET("/notes/:id", handlers.GetNote)
		authAPI.PUT("/notes/:id", handlers.UpdateNote)
		authAPI.DELETE("/notes/:id", handlers.DeleteNote)

		// 回收站
		authAPI.GET("/trash", handlers.GetTrashList)
		authAPI.POST("/trash/:id/restore", handlers.RestoreNote)
		authAPI.DELETE("/trash/:id", handlers.PermanentDelete)

		// 分享
		authAPI.POST("/notes/:id/toggle-public", handlers.TogglePublic)

		// 标签
		authAPI.POST("/tags", handlers.CreateTag)
		authAPI.GET("/tags", handlers.ListTags)
		authAPI.PUT("/tags/:id", handlers.UpdateTag)
		authAPI.DELETE("/tags/:id", handlers.DeleteTag)

		// 文件上传
		authAPI.POST("/upload/image", handlers.UploadImage)
		authAPI.GET("/files", handlers.ListFiles)
		authAPI.DELETE("/files/:id", handlers.DeleteFile)
	}

	// 处理404
	r.NoRoute(func(c *gin.Context) {
		// 接受HTML请求的，返回首页
		if c.Request.Header.Get("Accept")[:5] == "text/" {
			c.File("templates/index.html")
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": "Not Found"})
	})

	// 获取绝对路径
	absPath, _ := filepath.Abs(".")
	log.Printf("Server starting on http://localhost:8080")
	log.Printf("Working directory: %s", absPath)
	log.Fatal(r.Run(":8080"))
}
