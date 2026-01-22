package repository

import (
	"context"
	"letracker/internal/entity"
)

// Repository 定義了所有資料庫操作的方法
// 這樣做的好處是方便未來寫單元測試 (Mocking)
type Repository interface {
	// Question 相關
	GetQuestionBySlug(ctx context.Context, slug string) (*entity.Question, error)
	CreateQuestion(ctx context.Context, q entity.Question) (string, error) // 回傳 ID

	// Stats (SRS 狀態) 相關
	// 取得某使用者對某題的狀態
	GetUserStats(ctx context.Context, userID, questionID string) (*entity.UserQuestionStats, error)
	// 更新或插入狀態 (Upsert)
	UpsertUserStats(ctx context.Context, stats entity.UserQuestionStats) error

	// Logs (流水帳) 相關
	CreateLog(ctx context.Context, log entity.SubmissionLog) error
	// 批次寫入 Logs (給匯入歷史紀錄用)
	BatchCreateLogs(ctx context.Context, logs []entity.SubmissionLog) error
	// GetDailyTasks: 撈出今天需要做的題目 (含題目詳細資訊)
	GetDailyTasks(ctx context.Context, userID string, limit int) ([]entity.QuestionTask, error)
}
