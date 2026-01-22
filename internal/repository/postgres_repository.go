package repository

import (
	"context"
	"database/sql"
	"errors"
	"letracker/internal/entity"
	"log"
)

type postgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository 是建構函式
func NewPostgresRepository(db *sql.DB) Repository {
	return &postgresRepository{db: db}
}

// -------------------------------------------------------
// Question 實作
// -------------------------------------------------------

func (r *postgresRepository) GetQuestionBySlug(ctx context.Context, slug string) (*entity.Question, error) {
	query := `SELECT id, title, slug FROM questions WHERE slug = $1`

	var q entity.Question
	err := r.db.QueryRowContext(ctx, query, slug).Scan(&q.ID, &q.Title, &q.Slug)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("question not found")
		}
		return nil, err
	}
	return &q, nil
}

func (r *postgresRepository) CreateQuestion(ctx context.Context, q entity.Question) (string, error) {
	// 這裡使用 RETURNING id 讓 Postgres 回傳生成的 UUID
	query := `
		INSERT INTO questions (title, slug, is_neetcode_150)
		VALUES ($1, $2, $3)
		RETURNING id
	`
	var id string
	err := r.db.QueryRowContext(ctx, query, q.Title, q.Slug, q.IsNeetcode150).Scan(&id)
	log.Println(err)
	return id, err
}

// -------------------------------------------------------
// Stats 實作
// -------------------------------------------------------

func (r *postgresRepository) GetUserStats(ctx context.Context, userID, questionID string) (*entity.UserQuestionStats, error) {
	query := `
		SELECT id, streak, ease_factor, interval_days, next_review_at
		FROM user_question_stats
		WHERE user_id = $1 AND question_id = $2
	`
	var stats entity.UserQuestionStats
	// 記得掃描進去時要小心 NULL 值，這裡假設 DB 欄位都有 NOT NULL 或 Default
	err := r.db.QueryRowContext(ctx, query, userID, questionID).Scan(
		&stats.ID, &stats.Streak, &stats.EaseFactor, &stats.IntervalDays, &stats.NextReviewAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			// 如果沒找到，回傳 nil 讓 Service 層決定給預設值
			return nil, nil
		}
		return nil, err
	}
	return &stats, nil
}

func (r *postgresRepository) UpsertUserStats(ctx context.Context, stats entity.UserQuestionStats) error {
	// PostgreSQL 強大的 "ON CONFLICT" 語法
	// 如果 (user_id, question_id) 已經存在，就 Update，否則 Insert
	query := `
		INSERT INTO user_question_stats (
			user_id, question_id, streak, ease_factor, interval_days, next_review_at, last_reviewed_at, status
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (user_id, question_id) DO UPDATE SET
			streak = EXCLUDED.streak,
			ease_factor = EXCLUDED.ease_factor,
			interval_days = EXCLUDED.interval_days,
			next_review_at = EXCLUDED.next_review_at,
			last_reviewed_at = EXCLUDED.last_reviewed_at,
			status = EXCLUDED.status
	`
	_, err := r.db.ExecContext(ctx, query,
		stats.UserID, stats.QuestionID, stats.Streak, stats.EaseFactor,
		stats.IntervalDays, stats.NextReviewAt, stats.LastReviewedAt, stats.Status,
	)
	return err
}

// -------------------------------------------------------
// Logs 實作
// -------------------------------------------------------

func (r *postgresRepository) CreateLog(ctx context.Context, log entity.SubmissionLog) error {
	query := `
		INSERT INTO study_logs (user_id, question_id, status, mastery_level, attempted_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := r.db.ExecContext(ctx, query, log.UserID, log.QuestionID, log.Status, log.MasteryLevel, log.Date)
	return err
}

func (r *postgresRepository) BatchCreateLogs(ctx context.Context, logs []entity.SubmissionLog) error {
	// 這裡示範使用 Transaction 進行批次寫入
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO study_logs (user_id, question_id, status, attempted_at)
		VALUES ($1, $2, $3, $4)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, log := range logs {
		if _, err := stmt.ExecContext(ctx, log.UserID, log.QuestionID, log.Status, log.Date); err != nil {
			tx.Rollback() // 有一筆失敗就全部回滾
			return err
		}
	}

	return tx.Commit()
}

func (r *postgresRepository) GetDailyTasks(ctx context.Context, userID string, limit int) ([]entity.QuestionTask, error) {
	// 邏輯解說：
	// 1. 找出所有已經到期的 (next_review_at <= NOW()) 或是 全新的 (status = 'NEW')
	// 2. 計算 priority：
	//    - 如果是 NEW 或 interval=0，給予極高權重 (1000)，確保新題也會出現
	//    - 否則計算 (Now - NextReview) / Interval
	// 3. 取前 limit 筆 (例如 3 筆)

	query := `
		SELECT
			q.id, q.title, q.slug, q.difficulty, s.status, s.next_review_at,
			EXTRACT(EPOCH FROM (NOW() - s.next_review_at)) / 86400.0 as overdue_days
		FROM user_question_stats s
		JOIN questions q ON s.question_id = q.id
		WHERE s.user_id = $1
		  AND (s.next_review_at <= NOW() OR s.status = 'NEW')
		ORDER BY
			CASE
				WHEN s.interval_days = 0 THEN 1000.0
				ELSE EXTRACT(EPOCH FROM (NOW() - s.next_review_at)) / (s.interval_days * 86400)
			END DESC
		LIMIT $2
	`

	rows, err := r.db.QueryContext(ctx, query, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []entity.QuestionTask
	for rows.Next() {
		var t entity.QuestionTask
		// 掃描資料
		if err := rows.Scan(&t.QuestionID, &t.Title, &t.Slug, &t.Difficulty, &t.Status, &t.NextReviewAt, &t.OverdueByDays); err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}

	return tasks, nil
}
