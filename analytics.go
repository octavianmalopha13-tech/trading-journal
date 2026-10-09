package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
)

type SummaryResponse struct {
	TotalTrades     int     `json:"total_trades"`
	WinningTrades   int     `json:"winning_trades"`
	LosingTrades    int     `json:"losing_trades"`
	WinRate         float64 `json:"win_rate"`
	TotalPnL        float64 `json:"total_pnl"`
	AvgWin          float64 `json:"avg_win"`
	AvgLoss         float64 `json:"avg_loss"`
	LargestWin      float64 `json:"largest_win"`
	LargestLoss     float64 `json:"largest_loss"`
	TotalCommission float64 `json:"total_commission"`
	TotalSwap       float64 `json:"total_swap"`
}

type SymbolStats struct {
	Symbol   string  `json:"symbol"`
	Trades   int     `json:"trades"`
	Wins     int     `json:"wins"`
	WinRate  float64 `json:"win_rate"`
	TotalPnL float64 `json:"total_pnl"`
	AvgPnL   float64 `json:"avg_pnl"`
}

type SetupStats struct {
	Setup    string  `json:"setup"`
	Trades   int     `json:"trades"`
	Wins     int     `json:"wins"`
	WinRate  float64 `json:"win_rate"`
	TotalPnL float64 `json:"total_pnl"`
	AvgPnL   float64 `json:"avg_pnl"`
}

type EquityPoint struct {
	Timestamp     int64   `json:"timestamp"`
	CumulativePnL float64 `json:"cumulative_pnl"`
}

func handleAnalyticsSummary(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	userID := getUserID(r)

	if userID == 0 {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var response SummaryResponse

	var totalPnLPositive, totalPnLNegative float64

	err := db.QueryRow(`
		SELECT
			COUNT(*) AS trades,
			COALESCE(SUM(CASE WHEN t.pnl > 0 THEN 1 ELSE 0 END) AS wins, 0),
			COALESCE(SUM(CASE WHEN t.pnl < 0 THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(t.pnl), 0) AS total_pnl,
			COALESCE(AVG(CASE WHEN t.pnl > 0 THEN t.pnl END), 0),
			COALESCE(AVG(CASE WHEN t.pnl < 0 THEN t.pnl END), 0),
			COALESCE(MAX(t.pnl), 0),
			COALESCE(MIN(t.pnl), 0),
			COALESCE(SUM(t.commission), 0),
			COALESCE(SUM(t.swap), 0),
			COALESCE(SUM(CASE WHEN t.pnl > 0 THEN t.pnl ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN t.pnl < 0 THEN t.pnl ELSE 0 END), 0)
		FROM trades t
		JOIN accounts a ON t.account_id = a.id
		WHERE a.user_id = ?
	`, userID).Scan(
		&response.TotalTrades,
		&response.WinningTrades,
		&response.LosingTrades,
		&response.TotalPnL,
		&response.AvgWin,
		&response.AvgLoss,
		&response.LargestWin,
		&response.LargestLoss,
		&response.TotalCommission,
		&response.TotalSwap,
		&totalPnLPositive,
		&totalPnLNegative,
	)

	if err != nil {
		http.Error(w, "failed to fetch analytics", http.StatusInternalServerError)
		return
	}

	if response.TotalTrades > 0 {
		response.WinRate = float64(response.WinningTrades) /
			float64(response.TotalTrades) * 100
	}

	// Keep AvgLoss negative, matching the sign of losing trades.
	_ = totalPnLPositive
	_ = totalPnLNegative

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleAnalyticsBySymbol(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	userID := getUserID(r)

	if userID == 0 {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	rows, err := db.Query(`
		SELECT
			t.symbol,
			COUNT(*) AS trades,
			SUM(CASE WHEN t.pnl > 0 THEN 1 ELSE 0 END) AS wins,
			COALESCE(SUM(t.pnl), 0) AS total_pnl,
			COALESCE(AVG(t.pnl), 0) AS avg_pnl
		FROM trades t
		JOIN accounts a ON t.account_id = a.id
		WHERE a.user_id = ?
		GROUP BY t.symbol
		ORDER BY total_pnl DESC
	`, userID)

	if err != nil {
		log.Printf("❌SYMBOL QUERY ERROR: %v", err)
		http.Error(w, "failed to fetch symbol analytics", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	stats := []SymbolStats{}

	for rows.Next() {
		var s SymbolStats

		if err := rows.Scan(
			&s.Symbol,
			&s.Trades,
			&s.Wins,
			&s.TotalPnL,
			&s.AvgPnL,
		); err != nil {
			http.Error(w, "failed to read symbol analytics", http.StatusInternalServerError)
			return
		}

		if s.Trades > 0 {
			s.WinRate = float64(s.Wins) / float64(s.Trades) * 100
		}

		stats = append(stats, s)
	}

	if err := rows.Err(); err != nil {
		http.Error(w, "failed to read symbol analytics", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func handleAnalyticsBySetup(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	userID := getUserID(r)

	if userID == 0 {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	rows, err := db.Query(`
		SELECT
			j.setup,
			COUNT(*) AS trades,
			SUM(CASE WHEN t.pnl > 0 THEN 1 ELSE 0 END) AS wins,
			COALESCE(SUM(t.pnl), 0) AS total_pnl,
			COALESCE(AVG(t.pnl), 0) AS avg_pnl
		FROM journals j
		JOIN trades t ON j.trade_id = t.id
		JOIN accounts a ON t.account_id = a.id
		WHERE a.user_id = ?
		  AND j.setup != ''
		GROUP BY j.setup
		ORDER BY total_pnl DESC
	`, userID)

	if err != nil {
		log.Printf("❌ SETUP QUERY ERROR: %v", err)
		http.Error(w, "failed to fetch setup analytics", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	stats := []SetupStats{}

	for rows.Next() {
		var s SetupStats

		if err := rows.Scan(
			&s.Setup,
			&s.Trades,
			&s.Wins,
			&s.TotalPnL,
			&s.AvgPnL,
		); err != nil {
			http.Error(w, "failed to read setup analytics", http.StatusInternalServerError)
			return
		}

		if s.Trades > 0 {
			s.WinRate = float64(s.Wins) / float64(s.Trades) * 100
		}

		stats = append(stats, s)
	}

	if err := rows.Err(); err != nil {
		http.Error(w, "failed to read setup analytics", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func handleAnalyticsEquityCurve(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	userID := getUserID(r)

	if userID == 0 {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	rows, err := db.Query(`
		SELECT t.close_time, t.pnl
		FROM trades t
		JOIN accounts a ON t.account_id = a.id
		WHERE a.user_id = ?
		  AND t.close_time IS NOT NULL
		  AND t.pnl IS NOT NULL
		ORDER BY t.close_time ASC, t.id ASC
	`, userID)

	if err != nil {
		http.Error(w, "failed to fetch equity curve", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	points := []EquityPoint{}
	var cumulative float64

	for rows.Next() {
		var closeTime int64
		var pnl float64

		if err := rows.Scan(&closeTime, &pnl); err != nil {
			http.Error(w, "failed to read equity curve", http.StatusInternalServerError)
			return
		}

		cumulative += pnl

		points = append(points, EquityPoint{
			Timestamp:     closeTime,
			CumulativePnL: cumulative,
		})
	}

	if err := rows.Err(); err != nil {
		http.Error(w, "failed to read equity curve", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(points)
}
