package main

import (
	"log"
	"net/http"
	"os"

	"codesprint/database"
	"codesprint/handlers"
	"codesprint/middleware"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables if .env exists
	if _, err := os.Stat(".env"); err == nil {
		_ = godotenv.Load()
	}

	// Initialize database
	if err := database.InitDB(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.CloseDB()

	// Initialize optional Redis cache
	database.InitRedis()

	// Router
	r := mux.NewRouter()

	// Global CORS middleware
	r.Use(corsMiddleware)

	// API routes
	api := r.PathPrefix("/api").Subrouter()

	// Auth routes
	api.HandleFunc("/signup", handlers.Signup).Methods("POST")
	api.HandleFunc("/login", handlers.Login).Methods("POST")
	api.HandleFunc("/forgot-password", handlers.ForgotPassword).Methods("POST")
	api.HandleFunc("/reset-password", handlers.ResetPassword).Methods("POST")
	api.HandleFunc("/request-2fa-otp", handlers.Request2FAOTP).Methods("POST")
	api.HandleFunc("/login-2fa", handlers.LoginWith2FA).Methods("POST")
	api.Handle(
		"/enable-2fa",
		middleware.AuthMiddleware(handlers.Enable2FA),
	).Methods("POST")

	// Contest routes
	api.HandleFunc("/contests", handlers.GetContests).Methods("GET")
	api.HandleFunc("/contest/{id:[0-9]+}", handlers.GetContest).Methods("GET")
	api.Handle(
		"/contests",
		middleware.AuthMiddleware(handlers.CreateContest),
	).Methods("POST")
	api.Handle(
		"/contest/{id:[0-9]+}",
		middleware.AdminMiddleware(handlers.UpdateContest),
	).Methods("PUT")

	// Problem routes
	api.Handle(
		"/problems",
		middleware.AuthMiddleware(handlers.GetContestProblems),
	).Methods("GET")
	api.HandleFunc("/problem/{id:[0-9]+}", handlers.GetProblem).Methods("GET")
	api.Handle(
		"/problems",
		middleware.AuthMiddleware(handlers.CreateProblem),
	).Methods("POST")

	// Testcase routes
	api.Handle(
		"/testcases",
		middleware.AuthMiddleware(handlers.CreateTestcase),
	).Methods("POST")
	api.HandleFunc("/testcases", handlers.GetTestcases).Methods("GET")

	// Submission routes
	api.Handle(
		"/run",
		middleware.AuthMiddleware(handlers.RunCode),
	).Methods("POST")
	api.Handle(
		"/submission",
		middleware.AuthMiddleware(handlers.SubmitCode),
	).Methods("POST")
	api.HandleFunc("/submission/{id:[0-9]+}", handlers.GetSubmission).Methods("GET")
	api.Handle(
		"/submissions",
		middleware.AuthMiddleware(handlers.GetUserSubmissions),
	).Methods("GET")

	// Leaderboard
	api.HandleFunc(
		"/leaderboard/{contest_id:[0-9]+}",
		handlers.GetLeaderboard,
	).Methods("GET")

	// Serve frontend (must be last)
	r.PathPrefix("/").Handler(
		http.StripPrefix(
			"/",
			http.FileServer(http.Dir("./frontend/")),
		),
	)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server running on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}

// CORS middleware
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

