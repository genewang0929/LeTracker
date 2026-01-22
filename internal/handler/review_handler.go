// internal/handler/review_handler.go
package handler

import (
	"letracker/internal/service"
	"net/http"

	"github.com/gin-gonic/gin"
)

type ReviewHandler struct {
	svc service.ReviewService
}

// 建構子注入 Service
func NewReviewHandler(svc service.ReviewService) *ReviewHandler {
	return &ReviewHandler{svc: svc}
}

// HandleSubmitReview 處理 POST /api/v1/reviews
func (h *ReviewHandler) HandleSubmitReview(c *gin.Context) {
	var req SubmitReviewRequest

	// 1. 綁定並驗證 JSON
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 2. 從 Middleware 獲取 User ID (假設你有做 JWT Auth)
	// userID := c.MustGet("userID").(string)
	userID := "test-user-id" // 暫時寫死方便測試

	// 3. 呼叫 Service
	serviceReq := service.ReviewRequest{
		QuestionID: req.QuestionID,
		Grade:      req.Grade,
	}

	result, err := h.svc.ProcessReview(c.Request.Context(), userID, serviceReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process review"})
		return
	}

	// 4. 回傳結果
	c.JSON(http.StatusOK, SubmitReviewResponse{
		NextReviewAt: result.NextReviewAt.Format("2006-01-02 15:04:05"),
		IntervalDays: result.Interval,
		Message:      "Review recorded successfully. Keep it up!",
	})
}

func (h *ReviewHandler) HandleImportHistory(c *gin.Context) {
	var req service.ImportSubmissionRequest

	// 1. 解析 JSON
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON format: " + err.Error()})
		return
	}

	// 2. 假裝取得 UserID (之後接 Auth Middleware)
	userID := "00000000-0000-0000-0000-000000000000"

	// 3. 呼叫 Service

	if err := h.svc.ImportHistory(c.Request.Context(), userID, req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Import failed: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "History imported successfully",
		"count":   len(req.History),
	})
}

func (h *ReviewHandler) HandleGetDailyTasks(c *gin.Context) {
	// 假設從 Middleware 拿到 UserID
	// userID := c.MustGet("userID").(string)
	userID := "test-user-id"

	tasks, err := h.svc.GetTodayTasks(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch tasks"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"date":  "2026-01-18", // 可以回傳今天的日期
		"tasks": tasks,
	})
}
