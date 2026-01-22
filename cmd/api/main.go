package main

import (
	"database/sql"
	"log"
	"os"

	"letracker/internal/handler"
	"letracker/internal/repository"
	"letracker/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq" // PostgreSQL Driver
)

func main() {
	// 1. 連線資料庫
	// 請將此字串換成你的 Supabase Connection String
	// 建議格式: "postgres://postgres:[PASSWORD]@db.[REF].supabase.co:5432/postgres?sslmode=disable"
	// connStr := os.Getenv("DB_DSN")
	// if connStr == "" {
	// 	// 方便測試，你可以暫時把字串寫死在這裡，但不要 commit 到 git
	// 	log.Fatal("DB_DSN environment variable is not set")
	// }
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	connStr := os.Getenv("DB_DSN")
	if connStr == "" {
		log.Fatal("DB_DSN environment variable is not set")
	}

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Failed to connect to DB:", err)
	}
	defer db.Close()

	// ✅ 加入這段：強制送出 Ping 請求來測試真實連線
	// 這會在啟動時稍微卡住一下下，直到連線成功或 Timeout
	// if err := db.Ping(); err != nil {
	// 	log.Fatal("❌ Failed to establish connection to DB (Ping failed):", err)
	// }

	// log.Println("✅ Successfully connected to Database!")

	// 2. 初始化依賴注入 (Dependency Injection)
	repo := repository.NewPostgresRepository(db)
	svc := service.NewReviewService(repo)
	h := handler.NewReviewHandler(svc)

	// 3. 設定 Gin Router (API Endpoints 就在這裡！)
	r := gin.Default()

	// 建立一個 group，方便版本管理
	api := r.Group("/api/v1")
	{
		// 測試連線用
		api.GET("/ping", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "pong"})
		})

		// [核心功能路由]
		// 1. 匯入歷史紀錄 (Chrome Extension 會打這支)
		api.POST("/history", h.HandleImportHistory)

		// 2. 獲取每日任務 (Web App 首頁會打這支)
		api.GET("/tasks", h.HandleGetDailyTasks)
		// (註：HandleGetDailyTasks 的程式碼在上一則對話中)

		// 3. 提交練習結果 (做完題目後打這支)
		api.POST("/reviews", h.HandleSubmitReview)
	}

	// 4. 啟動伺服器
	log.Println("Server starting on port 8080...")
	if err := r.Run(":8080"); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}
