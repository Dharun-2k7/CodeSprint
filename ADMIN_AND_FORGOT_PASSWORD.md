# Admin and Forgot Password Fixes

## 1. Make yourself the only admin (no need to clear DB)

**Option A – One-time SQL (replace with your email):**
```bash
docker exec -i codesprint_db psql -U codesprint -d codesprint -c "UPDATE users SET is_admin = false; UPDATE users SET is_admin = true WHERE email = 'YOUR_EMAIL@example.com';"
```

**Option B – Automatic on every backend start**  
Add to `docker-compose.yml` under `backend` → `environment`:
```yaml
- ADMIN_EMAIL=your@email.com
```
Then the backend must apply it on startup. The `database/db.go` in this project is read-only; add this logic to your copy of `db.go` in `InitDB()` after `RunMigrations()`:

```go
// If ADMIN_EMAIL is set, ensure only that user is admin
if adminEmail := os.Getenv("ADMIN_EMAIL"); adminEmail != "" {
    DB.Exec("UPDATE users SET is_admin = false")
    DB.Exec("UPDATE users SET is_admin = true WHERE email = $1", adminEmail)
}
```

And run migrations in `InitDB()` (same file) after `InitSchema()`:
```go
// Run migrations (adds is_admin, 2FA, etc.)
migrations, _ := os.ReadFile("database/migrations.sql")
if migrations != nil {
    DB.Exec(string(migrations))
}
```

---

## 2. Forgot password

- **Backend:** Forgot/Reset password handlers now return JSON errors (`{"error": "..."}`) so the frontend can show them instead of breaking on `response.json()`.
- **No real email:** The app does not send real email; it only logs the OTP and, in the API response, returns the OTP in the `otp` field (for development). So “forgot password” works like this:
  1. Enter your email → Submit.
  2. Backend returns success and the OTP in the response.
  3. In the alert you should see: “OTP (use this to reset): 123456”.
  4. Enter that OTP and your new password in the reset form and submit.

If you still don’t see the OTP, open DevTools (F12) → Network, trigger “Send Reset OTP”, and check the JSON response for the `otp` field.

---

## 3. Quick check in the database

See who is admin:
```bash
docker exec -i codesprint_db psql -U codesprint -d codesprint -c "SELECT id, email, is_admin FROM users;"
```

If you need to fix file permissions to edit code:
```bash
chmod u+w database/db.go docker-compose.yml frontend/app.js
```
