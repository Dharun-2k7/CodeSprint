package handlers

import (
	"codesprint/database"
	"codesprint/email"
	"codesprint/models"
	"codesprint/utils"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// GetCurrentUser returns the current user's info
func GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	userID := utils.GetUserIDFromRequest(r)
	if userID == 0 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var user models.User
	err := database.DB.QueryRow(
		"SELECT id, name, email, is_admin, is_main_manager, two_factor_enabled, created_at FROM users WHERE id = $1",
		userID,
	).Scan(&user.ID, &user.Name, &user.Email, &user.IsAdmin, &user.IsMainManager, &user.TwoFactorEnabled, &user.CreatedAt)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":                  user.ID,
		"name":                user.Name,
		"email":               user.Email,
		"is_admin":            user.IsAdmin,
		"is_main_manager":     user.IsMainManager,
		"two_factor_enabled":  user.TwoFactorEnabled,
		"created_at":          user.CreatedAt,
	})
}

// Signup handles user registration
func Signup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.SignupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate input
	if req.Name == "" || req.Email == "" || req.Password == "" {
		http.Error(w, "Name, email, and password are required", http.StatusBadRequest)
		return
	}

	// Check if user already exists
	var existingID int
	err := database.DB.QueryRow("SELECT id FROM users WHERE email = $1", req.Email).Scan(&existingID)
	if err == nil {
		http.Error(w, "User already exists", http.StatusConflict)
		return
	}

	// Hash password
	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		http.Error(w, "Failed to hash password", http.StatusInternalServerError)
		return
	}

	// Create user
	var userID int
	err = database.DB.QueryRow(
		"INSERT INTO users (name, email, password_hash) VALUES ($1, $2, $3) RETURNING id",
		req.Name, req.Email, hashedPassword,
	).Scan(&userID)
	if err != nil {
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	// Generate JWT
	token, err := utils.GenerateJWT(userID, req.Email)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	// Get user with admin status
	var isAdmin, isMainManager bool
	database.DB.QueryRow("SELECT is_admin, is_main_manager FROM users WHERE id = $1", userID).Scan(&isAdmin, &isMainManager)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"token": token,
		"user": map[string]interface{}{
			"id":              userID,
			"name":            req.Name,
			"email":           req.Email,
			"is_admin":        isAdmin,
			"is_main_manager": isMainManager,
		},
	})
}

// Login handles user login
func Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get user from database
	var user models.User
	err := database.DB.QueryRow(
		"SELECT id, name, email, password_hash, is_admin, is_main_manager, two_factor_enabled FROM users WHERE email = $1",
		req.Email,
	).Scan(&user.ID, &user.Name, &user.Email, &user.PasswordHash, &user.IsAdmin, &user.IsMainManager, &user.TwoFactorEnabled)
	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Check password
	if !utils.CheckPasswordHash(req.Password, user.PasswordHash) {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// If 2FA is enabled, require OTP
	if user.TwoFactorEnabled {
		// Return response indicating 2FA is required
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"requires_2fa": true,
			"message":      "2FA is enabled. Please request and enter OTP",
		})
		return
	}

	// Generate JWT
	token, err := utils.GenerateJWT(user.ID, user.Email)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"token": token,
		"user": map[string]interface{}{
			"id":              user.ID,
			"name":            user.Name,
			"email":           user.Email,
			"is_admin":        user.IsAdmin,
			"is_main_manager": user.IsMainManager,
		},
	})
}

// ForgotPassword handles forgot password requests
func ForgotPassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.ForgotPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid request body"})
		return
	}

	// Check if user exists
	var userID int
	err := database.DB.QueryRow("SELECT id FROM users WHERE email = $1", req.Email).Scan(&userID)
	if err != nil {
		// Don't reveal if user exists or not (security best practice)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "If the email exists, a password reset OTP has been sent",
		})
		return
	}

	// Generate OTP
	otp, err := utils.GenerateOTP()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Failed to generate OTP"})
		return
	}

	// Store OTP in database (expires in 15 minutes)
	expiresAt := time.Now().Add(15 * time.Minute)
	_, err = database.DB.Exec(
		"INSERT INTO otp_codes (user_id, code, purpose, expires_at) VALUES ($1, $2, $3, $4)",
		userID, otp, "password_reset", expiresAt,
	)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Failed to store OTP"})
		return
	}

	// Send password reset OTP email via SMTP
	emailBody := fmt.Sprintf("Your password reset OTP is: %s\n\nThis code will expire in 15 minutes.", otp)
	if err := email.SendEmail(req.Email, "Password Reset OTP - Codesprint", emailBody); err != nil {
		fmt.Printf("Failed to send password reset email: %v\n", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Failed to send OTP email"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "If the email exists, a password reset OTP has been sent",
	})
}

// ResetPassword handles password reset with OTP
func ResetPassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.ResetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid request body"})
		return
	}

	// Validate OTP
	var userID int
	err := database.DB.QueryRow(
		"SELECT user_id FROM otp_codes WHERE code = $1 AND purpose = $2 AND expires_at > NOW() AND used = FALSE ORDER BY created_at DESC LIMIT 1",
		req.OTP, "password_reset",
	).Scan(&userID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid or expired OTP"})
		return
	}

	// Hash new password
	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Failed to hash password"})
		return
	}

	// Update password
	_, err = database.DB.Exec("UPDATE users SET password_hash = $1 WHERE id = $2", hashedPassword, userID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Failed to update password"})
		return
	}

	// Mark OTP as used
	database.DB.Exec("UPDATE otp_codes SET used = TRUE WHERE code = $1", req.OTP)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Password reset successfully",
	})
}

// LoginWith2FA handles login with 2FA OTP
func LoginWith2FA(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.TwoFactorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Check if user exists and has 2FA enabled
	var user models.User
	err := database.DB.QueryRow(
		"SELECT id, name, email, password_hash, is_admin, is_main_manager, two_factor_enabled FROM users WHERE email = $1",
		req.Email,
	).Scan(&user.ID, &user.Name, &user.Email, &user.PasswordHash, &user.IsAdmin, &user.IsMainManager, &user.TwoFactorEnabled)
	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	if !user.TwoFactorEnabled {
		http.Error(w, "2FA is not enabled for this account", http.StatusBadRequest)
		return
	}

	// Validate OTP
	var userID int
	err = database.DB.QueryRow(
		"SELECT user_id FROM otp_codes WHERE user_id = $1 AND code = $2 AND purpose = $3 AND expires_at > NOW() AND used = FALSE ORDER BY created_at DESC LIMIT 1",
		user.ID, req.OTP, "two_factor",
	).Scan(&userID)
	if err != nil {
		http.Error(w, "Invalid or expired OTP", http.StatusUnauthorized)
		return
	}

	// Mark OTP as used
	database.DB.Exec("UPDATE otp_codes SET used = TRUE WHERE code = $1 AND user_id = $2", req.OTP, user.ID)

	// Generate JWT
	token, err := utils.GenerateJWT(user.ID, user.Email)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"token": token,
		"user": map[string]interface{}{
			"id":              user.ID,
			"name":            user.Name,
			"email":           user.Email,
			"is_admin":        user.IsAdmin,
			"is_main_manager": user.IsMainManager,
		},
	})
}

// Request2FAOTP sends a 2FA OTP to the user
func Request2FAOTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get user and verify password
	var user models.User
	err := database.DB.QueryRow(
		"SELECT id, name, email, password_hash, two_factor_enabled FROM users WHERE email = $1",
		req.Email,
	).Scan(&user.ID, &user.Name, &user.Email, &user.PasswordHash, &user.TwoFactorEnabled)
	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	if !utils.CheckPasswordHash(req.Password, user.PasswordHash) {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	if !user.TwoFactorEnabled {
		http.Error(w, "2FA is not enabled for this account", http.StatusBadRequest)
		return
	}

	// Generate OTP
	otp, err := utils.GenerateOTP()
	if err != nil {
		http.Error(w, "Failed to generate OTP", http.StatusInternalServerError)
		return
	}

	// Store OTP in database (expires in 5 minutes)
	expiresAt := time.Now().Add(5 * time.Minute)
	_, err = database.DB.Exec(
		"INSERT INTO otp_codes (user_id, code, purpose, expires_at) VALUES ($1, $2, $3, $4)",
		user.ID, otp, "two_factor", expiresAt,
	)
	if err != nil {
		http.Error(w, "Failed to store OTP", http.StatusInternalServerError)
		return
	}

	// Send 2FA OTP email via SMTP
	emailBody := fmt.Sprintf("Your login OTP is: %s\n\nThis code will expire in 5 minutes.", otp)
	if err := email.SendEmail(req.Email, "Login OTP - Codesprint", emailBody); err != nil {
		fmt.Printf("Failed to send 2FA OTP email: %v\n", err)
		http.Error(w, "Failed to send OTP email", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "OTP has been sent to your email",
	})
}

// Enable2FA enables 2FA for a user
func Enable2FA(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := utils.GetUserIDFromRequest(r)
	if userID == 0 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req models.Enable2FARequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Generate and send initial OTP to verify email
	otp, err := utils.GenerateOTP()
	if err != nil {
		http.Error(w, "Failed to generate OTP", http.StatusInternalServerError)
		return
	}

	// Store OTP for verification
	expiresAt := time.Now().Add(10 * time.Minute)
	_, err = database.DB.Exec(
		"INSERT INTO otp_codes (user_id, code, purpose, expires_at) VALUES ($1, $2, $3, $4)",
		userID, otp, "two_factor_setup", expiresAt,
	)
	if err != nil {
		http.Error(w, "Failed to store OTP", http.StatusInternalServerError)
		return
	}

	// Get user email
	var email string
	database.DB.QueryRow("SELECT email FROM users WHERE id = $1", userID).Scan(&email)

	// Queue email
	emailBody := fmt.Sprintf("Your 2FA setup verification OTP is: %s\n\nEnter this code to enable 2FA. This code will expire in 10 minutes.", otp)
	database.DB.Exec(
		"INSERT INTO email_queue (recipient_email, subject, body) VALUES ($1, $2, $3)",
		email, "Enable 2FA - Codesprint", emailBody,
	)

	fmt.Printf("2FA Setup OTP for %s: %s\n", email, otp)

	// If OTP provided, verify and enable
	if req.OTP != "" {
		var verifiedUserID int
		err := database.DB.QueryRow(
			"SELECT user_id FROM otp_codes WHERE user_id = $1 AND code = $2 AND purpose = $3 AND expires_at > NOW() AND used = FALSE ORDER BY created_at DESC LIMIT 1",
			userID, req.OTP, "two_factor_setup",
		).Scan(&verifiedUserID)
		if err != nil {
			http.Error(w, "Invalid or expired OTP", http.StatusBadRequest)
			return
		}

		// Enable 2FA
		_, err = database.DB.Exec("UPDATE users SET two_factor_enabled = TRUE WHERE id = $1", userID)
		if err != nil {
			http.Error(w, "Failed to enable 2FA", http.StatusInternalServerError)
			return
		}

		// Mark OTP as used
		database.DB.Exec("UPDATE otp_codes SET used = TRUE WHERE code = $1", req.OTP)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "2FA enabled successfully",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "OTP sent. Verify with the OTP to enable 2FA",
		"otp":     otp, // Development only
	})
}
