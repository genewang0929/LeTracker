package entity

import (
	"time"
)

// Question 對應資料庫的 questions 表
type Question struct {
	ID            string    `json:"id"` // UUID
	LeetcodeID    int       `json:"leetcode_id"`
	Title         string    `json:"title"`
	Slug          string    `json:"slug"`
	Difficulty    string    `json:"difficulty"`
	Category      string    `json:"category"`
	IsNeetcode150 bool      `json:"is_neetcode_150"`
	CreatedAt     time.Time `json:"created_at"`
}

// SubmissionLog 對應資料庫的 study_logs 表
// 這是每一次練習的流水帳
type SubmissionLog struct {
	ID               string    `json:"id"`
	UserID           string    `json:"user_id"`
	QuestionID       string    `json:"question_id"`
	Status           string    `json:"status"`        // "SOLVED", "FAILED"
	MasteryLevel     int       `json:"mastery_level"` // 0-3
	TimeTakenSeconds int       `json:"time_taken_seconds"`
	Notes            string    `json:"notes"`
	Date             time.Time `json:"attempted_at"`
}

// UserQuestionStats 對應資料庫的 user_question_stats 表
// 這是演算法計算後的當前狀態
type UserQuestionStats struct {
	ID             string    `json:"id"`
	UserID         string    `json:"user_id"`
	QuestionID     string    `json:"question_id"`
	Streak         int       `json:"streak"`
	EaseFactor     float64   `json:"ease_factor"`
	IntervalDays   int       `json:"interval_days"`
	Status         string    `json:"status"` // "NEW", "LEARNING", "REVIEW", "MASTERED"
	NextReviewAt   time.Time `json:"next_review_at"`
	LastReviewedAt time.Time `json:"last_reviewed_at"`
}

type QuestionTask struct {
	QuestionID    string    `json:"question_id"`
	Title         string    `json:"title"`
	Slug          string    `json:"slug"`
	Difficulty    string    `json:"difficulty"`
	Status        string    `json:"status"` // "NEW", "REVIEW"
	NextReviewAt  time.Time `json:"next_review_at"`
	OverdueByDays float64   `json:"overdue_by_days"` // 用來顯示「逾期多久」
}
