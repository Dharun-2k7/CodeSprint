package handlers

import (
	"codesprint/database"
	"codesprint/models"
	"codesprint/utils"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
)

// CreateContest handles contest creation (admin only)
func CreateContest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := utils.GetUserIDFromRequest(r)
	if userID == 0 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Check if user is admin
	var isAdmin bool
	err := database.DB.QueryRow("SELECT is_admin FROM users WHERE id = $1", userID).Scan(&isAdmin)
	if err != nil || !isAdmin {
		http.Error(w, "Admin access required", http.StatusForbidden)
		return
	}

	var req models.CreateContestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate input
	if req.Title == "" {
		http.Error(w, "Title is required", http.StatusBadRequest)
		return
	}

	if req.EndTime.Before(req.StartTime) {
		http.Error(w, "End time must be after start time", http.StatusBadRequest)
		return
	}

	// Create contest
	var contestID int
	writerName := sql.NullString{String: req.WriterName, Valid: req.WriterName != ""}
	err = database.DB.QueryRow(
		"INSERT INTO contests (title, start_time, end_time, created_by, writer_name, standings_frozen) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id",
		req.Title, req.StartTime, req.EndTime, userID, writerName, false,
	).Scan(&contestID)
	if err != nil {
		http.Error(w, "Failed to create contest", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":         contestID,
		"title":      req.Title,
		"start_time": req.StartTime,
		"end_time":   req.EndTime,
	})
}

// GetContests returns all contests
func GetContests(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	rows, err := database.DB.Query(`
		SELECT id, title, start_time, end_time, created_by, writer_name, standings_frozen, freeze_time, created_at 
		FROM contests 
		ORDER BY created_at DESC
	`)
	if err != nil {
		http.Error(w, "Failed to fetch contests", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var contests []models.Contest
	for rows.Next() {
		var contest models.Contest
		var writerName sql.NullString
		var freezeTime sql.NullTime
		err := rows.Scan(&contest.ID, &contest.Title, &contest.StartTime, &contest.EndTime, &contest.CreatedBy, &writerName, &contest.StandingsFrozen, &freezeTime, &contest.CreatedAt)
		if err != nil {
			http.Error(w, "Failed to scan contest", http.StatusInternalServerError)
			return
		}
		if writerName.Valid {
			contest.WriterName = writerName.String
		}
		if freezeTime.Valid {
			contest.FreezeTime = freezeTime.Time
		}
		contests = append(contests, contest)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(contests)
}

// GetContest returns a specific contest
func GetContest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get contest ID from URL path (mux variable)
	vars := mux.Vars(r)
	contestIDStr := vars["id"]
	if contestIDStr == "" {
		contestIDStr = r.URL.Query().Get("id")
	}
	contestID, err := strconv.Atoi(contestIDStr)
	if err != nil {
		http.Error(w, "Invalid contest ID", http.StatusBadRequest)
		return
	}

	var contest models.Contest
	var writerName sql.NullString
	var freezeTime sql.NullTime
	err = database.DB.QueryRow(
		"SELECT id, title, start_time, end_time, created_by, writer_name, standings_frozen, freeze_time, created_at FROM contests WHERE id = $1",
		contestID,
	).Scan(&contest.ID, &contest.Title, &contest.StartTime, &contest.EndTime, &contest.CreatedBy, &writerName, &contest.StandingsFrozen, &freezeTime, &contest.CreatedAt)
	if err != nil {
		http.Error(w, "Contest not found", http.StatusNotFound)
		return
	}
	if writerName.Valid {
		contest.WriterName = writerName.String
	}
	if freezeTime.Valid {
		contest.FreezeTime = freezeTime.Time
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(contest)
}

// UpdateContest handles updating contest details (admin only)
func UpdateContest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := utils.GetUserIDFromRequest(r)
	if userID == 0 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Check if user is admin
	var isAdmin bool
	err := database.DB.QueryRow("SELECT is_admin FROM users WHERE id = $1", userID).Scan(&isAdmin)
	if err != nil || !isAdmin {
		http.Error(w, "Admin access required", http.StatusForbidden)
		return
	}

	vars := mux.Vars(r)
	contestIDStr := vars["id"]
	contestID, err := strconv.Atoi(contestIDStr)
	if err != nil {
		http.Error(w, "Invalid contest ID", http.StatusBadRequest)
		return
	}

	var req models.UpdateContestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Build update query dynamically
	updates := []string{}
	args := []interface{}{}
	argIndex := 1

	if req.Title != "" {
		updates = append(updates, fmt.Sprintf("title = $%d", argIndex))
		args = append(args, req.Title)
		argIndex++
	}
	if !req.StartTime.IsZero() {
		updates = append(updates, fmt.Sprintf("start_time = $%d", argIndex))
		args = append(args, req.StartTime)
		argIndex++
	}
	if !req.EndTime.IsZero() {
		updates = append(updates, fmt.Sprintf("end_time = $%d", argIndex))
		args = append(args, req.EndTime)
		argIndex++
	}
	if req.WriterName != "" {
		updates = append(updates, fmt.Sprintf("writer_name = $%d", argIndex))
		args = append(args, req.WriterName)
		argIndex++
	}
	updates = append(updates, fmt.Sprintf("standings_frozen = $%d", argIndex))
	args = append(args, req.StandingsFrozen)
	argIndex++
	if !req.FreezeTime.IsZero() {
		updates = append(updates, fmt.Sprintf("freeze_time = $%d", argIndex))
		args = append(args, req.FreezeTime)
		argIndex++
	}

	if len(updates) == 0 {
		http.Error(w, "No fields to update", http.StatusBadRequest)
		return
	}

	args = append(args, contestID)
	query := fmt.Sprintf("UPDATE contests SET %s WHERE id = $%d", strings.Join(updates, ", "), argIndex)
	_, err = database.DB.Exec(query, args...)
	if err != nil {
		http.Error(w, "Failed to update contest", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Contest updated successfully",
	})
}
