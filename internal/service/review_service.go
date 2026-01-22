package service

import (
	"context"
	"sort"
	"time"

	"letracker/internal/entity"
	"letracker/internal/repository"
	"letracker/pkg/srs"
)

// ReviewService 定義所有與複習相關的業務邏輯
type ReviewService interface {
	// ProcessReview 處理使用者當下的練習提交 (單題)
	ProcessReview(ctx context.Context, userID string, req ReviewRequest) (*srs.ReviewOutput, error)

	// ImportHistory 處理從 Extension 抓來的整包歷史紀錄 (批次)
	ImportHistory(ctx context.Context, userID string, req ImportSubmissionRequest) error

	GetTodayTasks(ctx context.Context, userID string) ([]entity.QuestionTask, error)
}

type reviewServiceImpl struct {
	repo repository.Repository
}

// NewReviewService 建構子
func NewReviewService(repo repository.Repository) ReviewService {
	return &reviewServiceImpl{repo: repo}
}

// =========================================================
// DTOs (Data Transfer Objects)
// =========================================================

type ReviewRequest struct {
	QuestionID string
	Grade      int // 0-3
}

type ImportSubmissionRequest struct {
	History []HistoryItem `json:"history"`
}

type HistoryItem struct {
	Title     string `json:"title"`
	Slug      string `json:"slug"`
	Status    string `json:"status"`    // "Accepted", "Wrong Answer" ...
	Timestamp int64  `json:"timestamp"` // Unix timestamp
}

// =========================================================
// 1. ProcessReview (單題即時處理)
// =========================================================

func (s *reviewServiceImpl) ProcessReview(ctx context.Context, userID string, req ReviewRequest) (*srs.ReviewOutput, error) {
	// 1. 取得目前狀態 (如果沒有則初始化)
	currentStats, err := s.repo.GetUserStats(ctx, userID, req.QuestionID)
	if err != nil {
		return nil, err
	}

	// 處理第一次練習的情況
	if currentStats == nil {
		currentStats = &entity.UserQuestionStats{
			IntervalDays: 0,
			EaseFactor:   2.5,
			Streak:       0,
		}
	}

	// 2. 執行 SRS 演算法
	algoInput := srs.ReviewInput{
		CurrentInterval: currentStats.IntervalDays,
		CurrentEF:       currentStats.EaseFactor,
		Repetitions:     currentStats.Streak,
		Grade:           req.Grade,
		ActualDays:      0, // 即時練習通常不需要算 Retention Bonus
	}

	result := srs.CalculateNextReview(algoInput)

	// 3. 更新 DB (Stats)
	newStats := entity.UserQuestionStats{
		UserID:         userID,
		QuestionID:     req.QuestionID,
		Streak:         result.Repetitions,
		EaseFactor:     result.EaseFactor,
		IntervalDays:   result.Interval,
		NextReviewAt:   result.NextReviewAt,
		LastReviewedAt: time.Now(),
		Status:         determineStatus(result.Repetitions),
	}

	if err := s.repo.UpsertUserStats(ctx, newStats); err != nil {
		return nil, err
	}

	// 4. 寫入 Log (流水帳)
	log := entity.SubmissionLog{
		UserID:       userID,
		QuestionID:   req.QuestionID,
		Status:       "SOLVED", // 這裡簡化，假設 ProcessReview 是做對了才呼叫，或需擴充 Request
		MasteryLevel: req.Grade,
		Date:         time.Now(),
	}
	// 如果 Grade 是 0，視為 Failed
	if req.Grade == 0 {
		log.Status = "FAILED"
	}

	if err := s.repo.CreateLog(ctx, log); err != nil {
		return nil, err
	}

	return &result, nil
}

// =========================================================
// 2. ImportHistory (歷史回放與匯入)
// =========================================================
//
// // 定義一個內部使用的 struct，專門給 Replay 邏輯用
type replayItem struct {
	Timestamp time.Time
	Status    string
	Title     string
}

func (s *reviewServiceImpl) ImportHistory(ctx context.Context, userID string, req ImportSubmissionRequest) error {
	// 1. 資料前處理：按時間排序 (從舊到新)
	sort.Slice(req.History, func(i, j int) bool {
		return req.History[i].Timestamp < req.History[j].Timestamp
	})

	// 用 Map 分組： Key=Slug, Value=List of items
	historyBySlug := make(map[string][]replayItem)

	for _, item := range req.History {
		historyBySlug[item.Slug] = append(historyBySlug[item.Slug], replayItem{
			Timestamp: time.Unix(item.Timestamp, 0),
			Status:    item.Status,
			Title:     item.Title,
		})
	}

	// 2. 逐題處理
	for slug, items := range historyBySlug {
		// A. 確保題目存在 (Lazy Loading)
		// 取第一筆紀錄的 Title 來當作題目名稱
		questionID, err := s.ensureQuestionExists(ctx, slug, items[0].Title)
		if err != nil {
			continue
		}

		// B. 執行回放演算法 (Replay) 計算最終狀態
		finalStats, logsToInsert := s.replayHistory(userID, questionID, items)

		// C. 寫入最終狀態
		if err := s.repo.UpsertUserStats(ctx, finalStats); err != nil {
			return err
		}

		// D. 批次寫入 Logs
		if len(logsToInsert) > 0 {
			if err := s.repo.BatchCreateLogs(ctx, logsToInsert); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *reviewServiceImpl) GetTodayTasks(ctx context.Context, userID string) ([]entity.QuestionTask, error) {
	// 設定 limit 為 3 (根據你的需求，也可以做成參數傳入)
	return s.repo.GetDailyTasks(ctx, userID, 3)
}

// Helper: 確保題目存在，不存在則建立
func (s *reviewServiceImpl) ensureQuestionExists(ctx context.Context, slug string, title string) (string, error) {
	// 1. 查 DB
	q, err := s.repo.GetQuestionBySlug(ctx, slug)
	if err == nil {
		return q.ID, nil
	}

	// 2. 沒找到 -> 建立
	newQ := entity.Question{
		Slug:          slug,
		Title:         title,
		IsNeetcode150: true, // 匯入的預設為 true
	}
	// log.Println(newQ)
	return s.repo.CreateQuestion(ctx, newQ)
}

// Helper: 核心回放邏輯
func (s *reviewServiceImpl) replayHistory(userID, questionID string, items []replayItem) (entity.UserQuestionStats, []entity.SubmissionLog) {
	// 初始化狀態
	currentStats := entity.UserQuestionStats{
		UserID:       userID,
		QuestionID:   questionID,
		IntervalDays: 0,
		EaseFactor:   2.5,
		Streak:       0,
		Status:       "NEW",
	}

	var logs []entity.SubmissionLog
	var lastReviewDate time.Time

	for i, item := range items {
		// 1. 準備 Log 物件
		mastery := 2 // 預設 Accepted = Good
		if item.Status != "Accepted" {
			mastery = 0 // Failed
		}

		log := entity.SubmissionLog{
			UserID:       userID,
			QuestionID:   questionID,
			Status:       "SOLVED",
			MasteryLevel: mastery,
			Date:         item.Timestamp,
		}
		if mastery == 0 {
			log.Status = "FAILED"
		}
		logs = append(logs, log)

		// 2. 計算 ActualDays (距離上次的時間)
		actualDays := 0.0
		if i > 0 {
			actualDays = item.Timestamp.Sub(lastReviewDate).Hours() / 24.0
		}

		// 設定第一筆的時間錨點
		if i == 0 {
			lastReviewDate = item.Timestamp
		}

		// [過濾機制]：如果同一天刷多次 (間隔 < 12小時)，跳過 SRS 計算，但 Log 照記
		if i > 0 && actualDays < 0.5 {
			continue
		}

		// 3. 呼叫 SRS
		srsInput := srs.ReviewInput{
			CurrentInterval: currentStats.IntervalDays,
			CurrentEF:       currentStats.EaseFactor,
			Repetitions:     currentStats.Streak,
			Grade:           mastery,
			ActualDays:      actualDays, // 傳入實際天數以觸發 Bonus
		}

		srsOutput := srs.CalculateNextReview(srsInput)

		// 4. 更新狀態
		currentStats.IntervalDays = srsOutput.Interval
		currentStats.EaseFactor = srsOutput.EaseFactor
		currentStats.Streak = srsOutput.Repetitions
		currentStats.Status = determineStatus(srsOutput.Repetitions)

		lastReviewDate = item.Timestamp
	}

	// 計算最終的 NextReviewAt
	currentStats.LastReviewedAt = lastReviewDate
	currentStats.NextReviewAt = lastReviewDate.AddDate(0, 0, currentStats.IntervalDays)

	return currentStats, logs
}

func determineStatus(streak int) string {
	if streak == 0 {
		return "LEARNING"
	} else if streak > 5 {
		return "MASTERED"
	}
	return "REVIEW"
}
