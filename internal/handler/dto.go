// internal/handler/dto.go
package handler

type SubmitReviewRequest struct {
	QuestionID string `json:"question_id" binding:"required"`
	// 0: Again, 1: Hard, 2: Good, 3: Easy
	Grade int `json:"grade" binding:"min=0,max=3"`
}

type SubmitReviewResponse struct {
	NextReviewAt string `json:"next_review_at"`
	IntervalDays int    `json:"interval_days"`
	Message      string `json:"message"`
}
