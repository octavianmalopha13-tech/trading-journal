package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func handleMe(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	userID := getUserID(r)
	if userID == 0 {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"user_id": userID,
		"status":  "authenticated",
	})
}

func main() {
	// 1. Init DB
	db, err := initDB("trading.db")
	if err != nil {
		log.Fatalf("DB init failed: %v", err)
	}
	defer db.Close()
	log.Println("✅ Database ready")

	// 2. Chi router
	r := chi.NewRouter()

	// 3. Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// 4. Routes
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Trading Journal API")
	})

	// Public routes
	r.Post("/signup", func(w http.ResponseWriter, r *http.Request) {
		handleSignup(w, r, db)
	})
	r.Post("/login", func(w http.ResponseWriter, r *http.Request) {
		handleLogin(w, r, db)
	})

	// Protected route
	/*r.With(authMiddleware).Get("/me", func(w http.ResponseWriter, r *http.Request) {
		handleMe(w, r, db)
	})*/
	r.Get("/me", authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		handleMe(w, r, db)
	}))

	r.Route("/accounts", func(r chi.Router) {
		r.Use(func(next http.Handler) http.Handler {
			return authMiddleware(next.ServeHTTP)
		})
		r.Post("/", func(w http.ResponseWriter, r *http.Request) {
			handleCreateAccount(w, r, db)
		})
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			handleListAccounts(w, r, db)
		})
		r.Delete("/{id}", func(w http.ResponseWriter, r *http.Request) {
			handleDeleteAccount(w, r, db)
		})
	})

	r.Route("/trades", func(r chi.Router) {
		r.Use(func(next http.Handler) http.Handler {
			return authMiddleware(next.ServeHTTP)
		})
		r.Post("/", func(w http.ResponseWriter, r *http.Request) {
			handleCreateTrade(w, r, db)
		})
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			handleListTrades(w, r, db)
		})
		r.Delete("/{id}", func(w http.ResponseWriter, r *http.Request) {
			handleDeleteTrade(w, r, db)
		})

		// NEW – Journal routes (nested under a specific trade)
		r.Post("/{id}/journal", func(w http.ResponseWriter, r *http.Request) {
			handleCreateJournal(w, r, db)
		})
		r.Get("/{id}/journal", func(w http.ResponseWriter, r *http.Request) {
			handleGetJournal(w, r, db)
		})
		r.Put("/{id}/journal", func(w http.ResponseWriter, r *http.Request) {
			handleUpdateJournal(w, r, db)
		})
		r.Delete("/{id}/journal", func(w http.ResponseWriter, r *http.Request) {
			handleDeleteJournal(w, r, db)
		})
	})

	// 5. Start server
	port := ":8080"
	log.Printf("🚀 Server on http://localhost%s", port)
	if err := http.ListenAndServe(port, r); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
