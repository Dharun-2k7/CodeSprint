package handlers

import (
	"codesprint/database"
	"codesprint/models"
	"codesprint/services"
	"codesprint/utils"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

// RunCode handles running code against sample testcases
func RunCode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Method not allowed"})
		return
	}

	userID := utils.GetUserIDFromRequest(r)
	if userID == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Unauthorized"})
		return
	}

	var req models.RunCodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Invalid request body"})
		return
	}

	// Validate input
	if req.Code == "" || req.Language == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Code and language are required"})
		return
	}

	runRes, err := services.RunSamples(req.ProblemID, req.Language, req.Code)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(runRes)
}

// SubmitCode handles code submission
func SubmitCode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Method not allowed"})
		return
	}

	userID := utils.GetUserIDFromRequest(r)
	if userID == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Unauthorized"})
		return
	}

	var req models.SubmitCodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Invalid request body"})
		return
	}

	// Validate input
	if req.Code == "" || req.Language == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Code and language are required"})
		return
	}

	if req.ProblemID == 0 || req.ContestID == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "problem_id and contest_id are required"})
		return
	}

	subRes, err := services.SubmitAndJudge(userID, req.ProblemID, req.ContestID, req.Language, req.Code)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		// Service failures could be judge issues or DB issues; treat as 500 unless it's clearly a bad request.
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(subRes)
}


// GetSubmission returns a submission by ID
func GetSubmission(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Method not allowed"})
		return
	}

	// Get submission ID from URL path (mux variable)
	vars := mux.Vars(r)
	submissionIDStr := vars["id"]
	if submissionIDStr == "" {
		submissionIDStr = r.URL.Query().Get("id")
	}
	submissionID, err := strconv.Atoi(submissionIDStr)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Invalid submission ID"})
		return
	}

	var submission models.Submission
	err = database.DB.QueryRow(
		"SELECT id, user_id, problem_id, contest_id, language, code, status, score, runtime, created_at FROM submissions WHERE id = $1",
		submissionID,
	).Scan(&submission.ID, &submission.UserID, &submission.ProblemID, &submission.ContestID, &submission.Language, &submission.Code, &submission.Status, &submission.Score, &submission.Runtime, &submission.CreatedAt)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Submission not found"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(submission)
}

// GetUserSubmissions returns all submissions for a user in a contest
func GetUserSubmissions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Method not allowed"})
		return
	}

	userID := utils.GetUserIDFromRequest(r)
	if userID == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Unauthorized"})
		return
	}

	contestIDStr := r.URL.Query().Get("contest_id")
	contestID, err := strconv.Atoi(contestIDStr)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Invalid contest ID"})
		return
	}

	rows, err := database.DB.Query(
		"SELECT id, user_id, problem_id, contest_id, language, status, score, runtime, created_at FROM submissions WHERE user_id = $1 AND contest_id = $2 ORDER BY created_at DESC",
		userID, contestID,
	)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Failed to fetch submissions"})
		return
	}
	defer rows.Close()

	var submissions []models.Submission
	for rows.Next() {
		var sub models.Submission
		err := rows.Scan(&sub.ID, &sub.UserID, &sub.ProblemID, &sub.ContestID, &sub.Language, &sub.Status, &sub.Score, &sub.Runtime, &sub.CreatedAt)
		if err != nil {
			continue
		}
		submissions = append(submissions, sub)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(submissions)
}

// updateLeaderboardCache updates the leaderboard cache for a submission
func updateLeaderboardCache(submissionID int) {
	var submission models.Submission
	err := database.DB.QueryRow(
		"SELECT user_id, contest_id, problem_id, status, created_at FROM submissions WHERE id = $1",
		submissionID,
	).Scan(&submission.UserID, &submission.ContestID, &submission.ProblemID, &submission.Status, &submission.CreatedAt)
	if err != nil {
		return
	}

	// Only count accepted submissions
	if submission.Status != "accepted" {
		return
	}

	// Check if this is the first accepted submission for this problem
	var existingCount int
	err = database.DB.QueryRow(
		"SELECT COUNT(*) FROM submissions WHERE user_id = $1 AND contest_id = $2 AND problem_id = $3 AND status = 'accepted'",
		submission.UserID, submission.ContestID, submission.ProblemID,
	).Scan(&existingCount)
	if err != nil || existingCount > 1 {
		// Not the first accepted, don't update
		return
	}

	// Get current leaderboard entry
	var solvedCount, penalty int
	var lastSubmissionTime time.Time
	err = database.DB.QueryRow(
		"SELECT solved_count, penalty, last_submission_time FROM leaderboard_cache WHERE contest_id = $1 AND user_id = $2",
		submission.ContestID, submission.UserID,
	).Scan(&solvedCount, &penalty, &lastSubmissionTime)

	if err != nil {
		// Create new entry
		contestStart, _ := getContestStartTime(submission.ContestID)
		penaltyMinutes := int(submission.CreatedAt.Sub(contestStart).Minutes())
		database.DB.Exec(
			"INSERT INTO leaderboard_cache (contest_id, user_id, solved_count, penalty, last_submission_time) VALUES ($1, $2, $3, $4, $5)",
			submission.ContestID, submission.UserID, 1, penaltyMinutes, submission.CreatedAt,
		)
	} else {
		// Update existing entry
		contestStart, _ := getContestStartTime(submission.ContestID)
		penaltyMinutes := int(submission.CreatedAt.Sub(contestStart).Minutes())
		database.DB.Exec(
			"UPDATE leaderboard_cache SET solved_count = solved_count + 1, penalty = penalty + $1, last_submission_time = $2 WHERE contest_id = $3 AND user_id = $4",
			penaltyMinutes, submission.CreatedAt, submission.ContestID, submission.UserID,
		)
	}
}

func getContestStartTime(contestID int) (time.Time, error) {
	var startTime time.Time
	err := database.DB.QueryRow(
		"SELECT start_time FROM contests WHERE id = $1",
		contestID,
	).Scan(&startTime)
	return startTime, err
}
