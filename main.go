package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"mimo-webui/internal/auth"
	"mimo-webui/internal/config"
	"mimo-webui/internal/db"
	"mimo-webui/internal/handlers"
	"mimo-webui/internal/middleware"
	"mimo-webui/internal/mimo"
)

//go:embed templates/*
var templateFS embed.FS

//go:embed static/*
var staticFS embed.FS

func main() {
	cfgPath := flag.String("config", "config.toml", "配置文件路径")
	flag.Parse()

	if v := os.Getenv("CONFIG_PATH"); v != "" {
		*cfgPath = v
	}

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	if cfg.MiMo.APIKey == "" {
		log.Fatal("MIMO_API_KEY is required (set in config.toml or environment)")
	}

	database, err := db.Open(cfg.Database.Path)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer database.Close()

	// Create default admin if no users exist
	if err := ensureDefaultAdmin(database, cfg.Auth.AdminPassword); err != nil {
		log.Fatalf("ensure default admin: %v", err)
	}

	mimoClient := mimo.NewClient(cfg.MiMo.BaseURL, cfg.MiMo.APIKey)

	// Ensure upload dir exists
	os.MkdirAll(cfg.Upload.TempDir, 0o755)

	// Start background cleanup
	middleware.StartCleanup(cfg.Upload.TempDir, cfg.Upload.CleanupIntervalMin, cfg.Upload.FileExpiryMin)

	r := gin.Default()

	// Load templates from embed FS
	tmpl := template.Must(template.ParseFS(templateFS, "templates/pages/*.html"))
	r.SetHTMLTemplate(tmpl)

	// Static files
	staticSub, _ := fs.Sub(staticFS, "static")
	r.StaticFS("/static", http.FS(staticSub))

	// CORS
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	setupRoutes(r, database, mimoClient, cfg)

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	log.Printf("MiMo WebUI starting on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("server: %v", err)
	}
}

func setupRoutes(r *gin.Engine, database *db.DB, client *mimo.Client, cfg *config.Config) {
	// Public routes
	r.GET("/login", handlers.LoginPage())
	r.GET("/register", handlers.RegisterPage())
	r.POST("/api/login", handlers.LoginHandler(database, cfg.Auth))
	r.POST("/api/register", handlers.RegisterHandler(database, cfg.Auth))

	// Redirect root to /chat
	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/chat")
	})

	// Authenticated page routes
	authMw := middleware.AuthMiddleware(cfg.Auth.JWTSecret)
	auth := r.Group("/", authMw)
	{
		auth.GET("/chat", handlers.ChatPage())
		auth.GET("/chat/:session_id", handlers.ChatSessionPage())
		auth.GET("/image", handlers.ImagePage())
		auth.GET("/audio", handlers.AudioPage())
		auth.GET("/video", handlers.VideoPage())
		auth.GET("/asr", handlers.ASRPage())
		auth.GET("/tts", handlers.TTSPage())
	}

	// Authenticated API routes
	api := r.Group("/api", authMw)
	{
		// Upload
		api.POST("/upload", handlers.UploadHandler(
			cfg.Upload.TempDir,
			cfg.Upload.MaxImageMB,
			cfg.Upload.MaxAudioMB,
			cfg.Upload.MaxVideoMB,
		))
		api.GET("/media/:file_id", handlers.MediaServeHandler(cfg.Upload.TempDir))

		// Multimodal understanding
		api.POST("/image", handlers.ImageUnderstandingHandler(client, cfg.Upload.TempDir))
		api.POST("/audio", handlers.AudioUnderstandingHandler(client, cfg.Upload.TempDir))
		api.POST("/video", handlers.VideoUnderstandingHandler(client, cfg.Upload.TempDir))
		api.POST("/asr", handlers.ASRHandler(client, cfg.Upload.TempDir))
		api.POST("/tts", handlers.TTSHandler(client, cfg.Upload.TempDir))

		// Chat sessions
		api.POST("/sessions", handlers.CreateSession(database))
		api.GET("/sessions", handlers.ListSessions(database))
		api.DELETE("/sessions/:session_id", handlers.DeleteSession(database))
		api.POST("/sessions/:session_id/messages", handlers.SendMessage(database, client, cfg.Upload.TempDir))
		api.GET("/sessions/:session_id/messages", handlers.ListMessages(database))
		api.POST("/sessions/:session_id/generate-title", handlers.GenerateTitle(database, client))

		// User info
		api.GET("/me", func(c *gin.Context) {
			user := middleware.GetAuthUser(c)
			c.JSON(http.StatusOK, gin.H{
				"user_id":  user.UserID,
				"username": user.Username,
				"role":     user.Role,
			})
		})
	}
}

func ensureDefaultAdmin(database *db.DB, adminPassword string) error {
	ctx := context.Background()
	count, err := db.CountUsers(ctx, database)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	if adminPassword == "" {
		adminPassword = "admin123"
		log.Println("[init] ⚠️  未配置 admin_password，使用默认密码 admin123")
	}

	hash, err := auth.HashPassword(adminPassword)
	if err != nil {
		return err
	}
	if _, err := db.CreateUser(ctx, database, "admin", hash, "admin"); err != nil {
		return err
	}
	log.Println("[init] Created default admin user: admin")
	if adminPassword == "admin123" {
		log.Println("[init] ⚠️  请登录后立即修改密码！")
	}
	return nil
}
