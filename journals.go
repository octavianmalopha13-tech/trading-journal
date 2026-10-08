package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

type JournalRequest struct {
	Setup    string `json:"setup"`
	Emotions string `json:"emotions"`
	Mistakes string `json:"mistakes"`
	Lessons  string `json:"lessons"`
	Notes    string `json:"notes"`
	Rating   int    `json:"rating"`
}

func handleCreateJournal(w http.ResponseWriter, r *http.Request, db *sql.DB) {
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

	var count int

	err = db.QueryRow(`SELECT COUNT(*) FROM trades t JOIN accounts a ON t.account_id=a.id WHERE t.id=? AND a.user_id=?`, tradeID, userID).Scan(&count)

	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}

	if count == 0 {
		http.Error(w, "trade not found or not yours", http.StatusForbidden)
		return
	}

	var req JournalRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "rating must be between 0 and 5", http.StatusBadRequest)
		return
	}

	now := time.Now().Unix()

	result, err := db.Exec(`
  INSERT INTO journals(
   trade_id,
   setup,
   emotions,
   mistakes,
   lessons,
   notes,
   rating,
   created_at,
   updated_at
   )
   VALUES(?,?,?,?,?,?,?,?,?)
  `,
		tradeID,
		req.Setup,
		req.Emotions,
		req.Mistakes,
		req.Lessons,
		req.Notes,
		req.Rating,
		now,
		now,
	)

	if err != nil {
		log.Printf("❌ JOURNAL INSERT ERROR: %v", err) // ← ADD THIS
		if strings.Contains(strings.ToUpper(err.Error()), "UNIQUE") {
			http.Error(w, "journal already exists for this trade", http.StatusConflict)
			return
		}
		http.Error(w, "failed to create journal", http.StatusInternalServerError)
		return
	}

	if err != nil {
		if strings.Contains(strings.ToUpper(err.Error()), "UNIQUE") {
			http.Error(w, "journal already exists for this trade", http.StatusConflict)
			return
		}
		http.Error(w, "failed to create journal", http.StatusInternalServerError)
		return
	}

	journalID, err := result.LastInsertId()

	if err != nil {
		http.Error(w, "failed to get journal ID", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "ok",
		"id":     journalID,
	})
}

func handleGetJournal(w http.ResponseWriter, r *http.Request, db *sql.DB) {
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

	var journal Journal

	err = db.QueryRow(`
		SELECT
			j.id,
			j.trade_id,
			j.setup,
			j.emotions,
			j.mistakes,
			j.lessons,
			j.notes,
			j.rating,
			j.created_at,
			j.updated_at
		FROM journals j
		JOIN trades t ON j.trade_id = t.id
		JOIN accounts a ON t.account_id = a.id
		WHERE j.trade_id = ?
		  AND a.user_id = ?
	`,
		tradeID,
		userID,
	).Scan(
		&journal.ID,
		&journal.TradeID,
		&journal.Setup,
		&journal.Emotions,
		&journal.Mistakes,
		&journal.Lessons,
		&journal.Notes,
		&journal.Rating,
		&journal.CreatedAt,
		&journal.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "journal not found", http.StatusNotFound)
			return
		}

		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(journal)
}

func handleUpdateJournal(w http.ResponseWriter, r *http.Request, db *sql.DB) {
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

	var req JournalRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Rating < 0 || req.Rating > 5 {
		http.Error(w, "rating must be between 0 and 5", http.StatusBadRequest)
		return
	}

	result, err := db.Exec(`
		UPDATE journals
		SET
			setup = ?,
			emotions = ?,
			mistakes = ?,
			lessons = ?,
			notes = ?,
			rating = ?,
			updated_at = ?
		WHERE trade_id = ?
		  AND trade_id IN (
			  SELECT t.id
			  FROM trades t
			  JOIN accounts a ON t.account_id = a.id
			  WHERE a.user_id = ?
		  )
	`,
		req.Setup,
		req.Emotions,
		req.Mistakes,
		req.Lessons,
		req.Notes,
		req.Rating,
		time.Now().Unix(),
		tradeID,
		userID,
	)

	if err != nil {
		http.Error(
			w,
			"failed to update journal",
			http.StatusInternalServerError,
		)
		return
	}

	rowsAffected, err := result.RowsAffected()

	if err != nil {
		http.Error(
			w,
			"failed to check update",
			http.StatusInternalServerError,
		)
		return
	}

	if rowsAffected == 0 {
		http.Error(w, "journal not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(map[string]string{
		"status": "updated",
	})
}

func handleDeleteJournal(w http.ResponseWriter, r *http.Request, db *sql.DB) {
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
		DELETE FROM journals
		WHERE trade_id = ?
		  AND trade_id IN (
			  SELECT t.id
			  FROM trades t
			  JOIN accounts a ON t.account_id = a.id
			  WHERE a.user_id = ?
		  )
	`,
		tradeID,
		userID,
	)

	if err != nil {
		http.Error(
			w,
			"failed to delete journal",
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
		http.Error(w, "journal not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(map[string]string{
		"status": "deleted",
	})
}
