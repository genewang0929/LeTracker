package srs

import (
	"math"
	"math/rand"
	"time"
)

// ReviewInput 前端/資料庫/回放邏輯傳入的參數
type ReviewInput struct {
	CurrentInterval int     // 上一次設定的間隔 (天)
	CurrentEF       float64 // 目前的難度係數 (預設 2.5)
	Repetitions     int     // 連續答對次數 (Streak)
	Grade           int     // 0: Again, 1: Hard, 2: Good, 3: Easy

	// ActualDays 是本次練習距離上次練習的「實際天數」
	// - 在「一般刷題模式」下，通常不需要傳 (或是傳 0)，我們會忽略它。
	// - 在「歷史回放模式」下，這是關鍵參數，用來判斷是否給予 "Long-term Retention Bonus"。
	ActualDays float64
}

// ReviewOutput 計算結果
type ReviewOutput struct {
	NextReviewAt time.Time
	Interval     int
	EaseFactor   float64
	Repetitions  int
}

// CalculateNextReview 執行 LeTracker 專屬改良演算法 (含回放邏輯)
func CalculateNextReview(input ReviewInput) ReviewOutput {
	// 初始化隨機數種子 (建議在 main init 做，這裡為了安全起見保留)
	// rand.Seed(time.Now().UnixNano())

	// ---------------------------------------------------------
	// 邏輯 1: 處理 "Again" (重做)
	// ---------------------------------------------------------
	if input.Grade == 0 {
		return ReviewOutput{
			NextReviewAt: time.Now().AddDate(0, 0, 1), // 明天立刻做
			Interval:     1,
			EaseFactor:   math.Max(1.3, input.CurrentEF-0.2), // 懲罰 EF 但設底限
			Repetitions:  0,                                  // 重置 streak
		}
	}

	// ---------------------------------------------------------
	// 邏輯 2: 計算新的 Ease Factor (EF)
	// ---------------------------------------------------------
	// 公式：EF' = EF + (0.1 - (3-Grade) * (0.08 + (3-Grade) * 0.02))
	newEF := input.CurrentEF + (0.1 - float64(3-input.Grade)*(0.08+float64(3-input.Grade)*0.02))
	if newEF < 1.3 {
		newEF = 1.3 // EF 底限
	}

	// ---------------------------------------------------------
	// 邏輯 3: 計算新的 Interval (天數)
	// ---------------------------------------------------------
	var newInterval int
	newRepetitions := input.Repetitions + 1

	if newRepetitions == 1 {
		newInterval = 1
	} else if newRepetitions == 2 {
		// [早期階段細緻化]
		switch input.Grade {
		case 1: // Hard
			newInterval = 3 // 3天後再見
		case 2: // Good
			newInterval = 5
		case 3: // Easy
			newInterval = 7
		default:
			newInterval = 4
		}
	} else {
		// [後期階段計算]

		// A. 基礎計算 (Interval * EF)
		baseInterval := float64(input.CurrentInterval) * newEF

		// B. Hard 懲罰 & Easy 獎勵
		modifier := 1.0
		if input.Grade == 1 {
			modifier = 0.8 // Hard 打 8 折
		} else if input.Grade == 3 {
			modifier = 1.1 // Easy 給 1.1 倍
		}

		calculatedDays := baseInterval * modifier

		// C. [回放模式專屬]: 長期記憶獎勵 (Retention Bonus)
		// 如果使用者在歷史紀錄中，實際隔了很久(ActualDays)才做且做對了，
		// 代表他記憶很深，我們應該大幅拉長下一次間隔。
		if input.ActualDays > 0 && input.Grade >= 2 {
			// 如果 實際間隔 > 預定間隔 的 1.5 倍
			if input.CurrentInterval > 0 && input.ActualDays > float64(input.CurrentInterval)*1.5 {
				// 給予 1.5 倍的額外獎勵 (這是一個激進但合理的策略)
				calculatedDays = math.Max(calculatedDays, input.ActualDays*1.5)

				// 並且因為表現優異，稍微提升 EF
				newEF += 0.15
			}
		}

		newInterval = int(math.Round(calculatedDays))
	}

	// ---------------------------------------------------------
	// 邏輯 4: Fuzzing (模糊化) - 防止題目堆積
	// ---------------------------------------------------------
	// 當間隔大於 10 天時，加入 ±5% 的隨機波動
	if newInterval > 10 {
		fuzzFactor := 0.95 + rand.Float64()*0.1
		newInterval = int(math.Round(float64(newInterval) * fuzzFactor))
	}

	// 確保至少間隔 1 天
	if newInterval < 1 {
		newInterval = 1
	}

	return ReviewOutput{
		NextReviewAt: time.Now().AddDate(0, 0, newInterval),
		Interval:     newInterval,
		EaseFactor:   newEF,
		Repetitions:  newRepetitions,
	}
}
