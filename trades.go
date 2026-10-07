package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	//    "time"

	"github.com/go-chi/chi/v5"
)

type CreateTradeRequest struct {
	AccountID       int64    `json:"account_id"`
	Ticket          string   `json:"ticket"`
	Symbol          string   `json:"symbol"`
	Direction       string   `json:"direction"`
	LotSize         float64  `json:"lot_size"`
	EntryPrice      float64  `json:"entry_price"`
	ExitPrice       *float64 `json:"exit_price"`
	SL              *float64 `json:"sl"`
	TP              *float64 `json:"tp"`
	OpenTime        int64    `json:"open_time"`
	CloseTime       *int64   `json:"close_time"`
	PnL             *float64 `json:"pnl"`
	Commission      float64  `json:"commission"`
	Swap            float64  `json:"swap"`
	DurationSeconds *int64   `json:"duration_seconds"`
}

func handleCreateTrade(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	userID := getUserID(r)

	if userID == 0 {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req CreateTradeRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	var count int

	err := db.QueryRow("SELECT COUNT(*) FROM accounts WHERE id=? AND user_id=?", req.AccountID, userID).Scan(&count)

	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}

	if count == 0 {
		http.Error(w, "account not found or not yours", http.StatusForbidden)
		return
	}

	if req.Ticket == "" || req.Symbol == "" || req.Direction == "" {
		http.Error(w, "ticket, symbol and direction are required", http.StatusBadRequest)
		return
	}

	req.Direction = strings.ToLower(req.Direction)

	if req.Direction != "buy" && req.Direction != "sell" {
		http.Error(w, "direction must be buy or sell", http.StatusBadRequest)
		return
	}

	result, err := db.Exec(`
   INSERT INTO trades (
     account_id,
     ticket,
     symbol,
     direction,
     lot_size,
     entry_price,
     exit_price,
     sl,
     tp,
     open_time,
     close_time,
     pnl,
     commission,
     swap,
     duration_seconds
   )
   VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
  `,
		req.AccountID,
		req.Ticket,
		req.Symbol,
		req.Direction,
		req.LotSize,
		req.EntryPrice,
		req.ExitPrice,
		req.SL,
		req.TP,
		req.OpenTime,
		req.CloseTime,
		req.PnL,
		req.Commission,
		req.Swap,
		req.DurationSeconds,
	)

	if err != nil {
		if strings.Contains(strings.ToUpper(err.Error()), "UNIQUE") {
			http.Error(w, "trade already exists", http.StatusConflict)
			return
		}

		http.Error(w, "failed to create trade", http.StatusInternalServerError)
		return
	}

	tradeID, err := result.LastInsertId()

	if err != nil {
		http.Error(w, "failed to get trade ID", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "ok",
		"id":     tradeID,
	})
}

func handleListTrades(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	userID := getUserID(r)

	if userID == 0 {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	accountIDStr := r.URL.Query().Get("account_id")

	var rows *sql.Rows
	var err error

	if accountIDStr == "" {

		rows, err = db.Query(`
			SELECT
				t.id,
				t.account_id,
				t.ticket,
				t.symbol,
				t.direction,
				t.lot_size,
				t.entry_price,
				t.exit_price,
				t.sl,
				t.tp,
				t.open_time,
				t.close_time,
				t.pnl,
				t.commission,
				t.swap,
				t.duration_seconds
			FROM trades t
			JOIN accounts a ON t.account_id = a.id
			WHERE a.user_id = ?
			ORDER BY t.open_time DESC
		`, userID)

	} else {

		accountID, parseErr := strconv.ParseInt(accountIDStr, 10, 64)

		if parseErr != nil {
			http.Error(w, "invalid account_id", http.StatusBadRequest)
			return
		}

		rows, err = db.Query(`
			SELECT
				t.id,
				t.account_id,
				t.ticket,
				t.symbol,
				t.direction,
				t.lot_size,
				t.entry_price,
				t.exit_price,
				t.sl,
				t.tp,
				t.open_time,
				t.close_time,
				t.pnl,
				t.commission,
				t.swap,
				t.duration_seconds
			FROM trades t
			JOIN accounts a ON t.account_id = a.id
			WHERE a.user_id = ?
			  AND t.account_id = ?
			ORDER BY t.open_time DESC
		`, userID, accountID)
	}

	if err != nil {
		http.Error(
			w,
			"failed to fetch trades",
			http.StatusInternalServerError,
		)
		return
	}

	defer rows.Close()

	trades := []Trade{}

	for rows.Next() {
		var trade Trade

		err := rows.Scan(
			&trade.ID,
			&trade.AccountID,
			&trade.Ticket,
			&trade.Symbol,
			&trade.Direction,
			&trade.LotSize,
			&trade.EntryPrice,
			&trade.ExitPrice,
			&trade.SL,
			&trade.TP,
			&trade.OpenTime,
			&trade.CloseTime,
			&trade.PnL,
			&trade.Commission,
			&trade.Swap,
			&trade.DurationSeconds,
		)

		if err != nil {
			http.Error(
				w,
				"failed to read trade",
				http.StatusInternalServerError,
			)
			return
		}

		trades = append(trades, trade)
	}

	if err := rows.Err(); err != nil {
		http.Error(
			w,
			"failed to read trades",
			http.StatusInternalServerError,
		)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(trades)
}

func handleDeleteTrade(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	userID := getUserID(r)

	if userID == 0 {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")

	tradeID, err := strconv.ParseInt(idStr, 10, 64)

	if err != nil {
		http.Error(w, "invalid trade ID", http.StatusBadRequest)
		return
	}

	result, err := db.Exec(`
		DELETE FROM trades
		WHERE id = ?
		  AND account_id IN (
			  SELECT id
			  FROM accounts
			  WHERE user_id = ?
		  )
	`, tradeID, userID)

	if err != nil {
		http.Error(
			w,
			"failed to delete trade",
			http.StatusInternalServerError,
		)
		return
	}

	rowsAffected, err := result.RowsAffected()

	if err != nil {
		http.Error(
			w,
			"failed to check deletion",
			http.StatusInternalServerError,
		)
		return
	}

	if rowsAffected == 0 {
		http.Error(
			w,
			"trade not found",
			http.StatusNotFound,
		)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(map[string]string{
		"status": "deleted",
	})
}
