# 🚀 Codesprint - Complete Deployment Checklist

## ✅ Pre-Deployment Checklist

Before deploying to production, ensure all these items are completed:

### Local Testing
- [ ] Run `./setup.sh` successfully
- [ ] Application starts without errors
- [ ] Can sign up and log in
- [ ] Made yourself admin via SQL
- [ ] Can create contests (admin only)
- [ ] Can create problems and testcases
- [ ] Can submit code successfully
- [ ] Submissions return correct verdicts (not runtime errors)
- [ ] Leaderboard updates properly
- [ ] Judge0 is responding

---

## 🌐 Deployment Options

### Option A: Render.com (Recommended for Beginners)

**Pros:**
- Free tier available
- Easy setup
- Automatic HTTPS
- Good documentation

**Cons:**
- Judge0 needs separate hosting
- Database limited to 90 days free

**Steps:**

1. **Push to GitHub**
```bash
git init
git add .
git commit -m "Initial commit - Codesprint"
git remote add origin https://github.com/YOUR_USERNAME/codesprint.git
git push -u origin main
```

2. **Create Web Service**
- Go to render.com → New → Web Service
- Connect GitHub repo
- Name: `codesprint-backend`
- Build: `go build -o main`
- Start: `./main`
- Instance: Free

3. **Add Environment Variables**
```
PORT=8080
JWT_SECRET=<run: openssl rand -hex 32>
DB_HOST=<from postgres service>
DB_PORT=5432
DB_USER=<from postgres service>
DB_PASSWORD=<from postgres service>
DB_NAME=codesprint
JUDGE0_URL=https://judge0-ce.p.rapidapi.com
RAPIDAPI_KEY=your_rapidapi_key_here
```

4. **Create PostgreSQL Database**
- New → PostgreSQL
- Name: `codesprint-db`
- Free tier
- Copy connection details to web service env vars

5. **Run Migrations**
```bash
# Connect using External Database URL
psql <EXTERNAL_URL>

# Run
\i schema.sql
\i migrations.sql
```

6. **Make Yourself Admin**
```sql
UPDATE users SET is_admin = TRUE, is_main_manager = TRUE 
WHERE email = 'your@email.com';
```

7. **Deploy Judge0 Separately**
- Use ngrok for development
- Or deploy to Fly.io (see Option B)

---

### Option B: Fly.io (Best for Full Stack)

**Pros:**
- Completely free (within limits)
- Docker support
- Can host Judge0 alongside app
- Global CDN

**Cons:**
- Slightly more complex setup
- Command-line based

**Steps:**

1. **Install flyctl**
```bash
curl -L https://fly.io/install.sh | sh
flyctl auth login
```

2. **Create fly.toml**
```toml
app = "codesprint"

[build]
  dockerfile = "Dockerfile"

[[services]]
  internal_port = 8080
  protocol = "tcp"

  [[services.ports]]
    port = 80
    handlers = ["http"]
  
  [[services.ports]]
    port = 443
    handlers = ["tls", "http"]

[env]
  PORT = "8080"
  JUDGE0_URL = "https://judge0-ce.p.rapidapi.com"
  RAPIDAPI_KEY = "your_rapidapi_key_here"
```

3. **Create PostgreSQL**
```bash
flyctl postgres create --name codesprint-db
flyctl postgres attach codesprint-db
```

4. **Set Secrets**
```bash
flyctl secrets set JWT_SECRET="$(openssl rand -hex 32)"
```

5. **Deploy**
```bash
flyctl deploy
```

6. **Run Migrations**
```bash
flyctl postgres connect -a codesprint-db
# Then run schema.sql and migrations.sql
```

7. **Make Yourself Admin** (same SQL as above)

---

### Option C: Railway.app

**Pros:**
- Easiest deployment
- Auto-detects Go
- Free tier
- Built-in PostgreSQL

**Cons:**
- Limited free tier
- Judge0 hosting tricky

**Steps:**

1. **Push to GitHub** (same as Option A)

2. **Create Railway Project**
- Go to railway.app
- New Project → Deploy from GitHub
- Select your repo

3. **Add PostgreSQL**
- Add → Database → PostgreSQL
- Railway auto-configures connection

4. **Set Variables**
```
JWT_SECRET=<generate>
JUDGE0_URL=https://judge0-ce.p.rapidapi.com
RAPIDAPI_KEY=your_rapidapi_key_here
```

5. **Deploy** (automatic after push)

6. **Run Migrations** via Railway dashboard → Database → Query

7. **Make Yourself Admin** (same SQL)

---

## 🔒 Production Security Checklist

Before going live:

### Environment Variables
- [ ] JWT_SECRET is strong (min 32 random characters)
- [ ] Database password is strong
- [ ] No default passwords in use

### Code Changes
- [ ] Remove OTP logging (search for `fmt.Printf.*OTP`)
- [ ] Set up real SMTP service for emails
- [ ] Add rate limiting
- [ ] Restrict CORS to your domain

### Database
- [ ] Backups configured
- [ ] Connection pooling optimized
- [ ] Indexes created (check schema.sql)

### Monitoring
- [ ] Application logs monitored
- [ ] Error alerts set up
- [ ] Database metrics tracked
- [ ] Judge0 health checks

---

## 📊 Post-Deployment Testing

After deployment, test these scenarios:

### 1. User Flow
```bash
# Sign up
curl -X POST https://your-app.com/api/signup \
  -H "Content-Type: application/json" \
  -d '{"name":"Test","email":"test@test.com","password":"pass123"}'
  
# Login
curl -X POST https://your-app.com/api/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@test.com","password":"pass123"}'
```

### 2. Admin Access
- Make user admin via SQL
- Login again
- Verify Admin button shows
- Create test contest
- Create test problem

### 3. Submission
- Add testcase to problem
- Submit code via UI
- Verify it compiles and runs
- Check verdict is correct

### 4. Leaderboard
- Submit solutions as multiple users
- Check leaderboard updates
- Verify correct ranking
- Confirm polling is 30 seconds

---

## 🐛 Common Deployment Issues

### "Database connection failed"
**Fix:**
- Verify DATABASE_URL is correct
- Check database is in same region
- Ensure SSL mode matches (often `sslmode=require` in production)

### "Judge0 not responding"
**Fix:**
- If using ngrok, ensure it's running
- Check JUDGE0_URL environment variable
- Consider deploying Judge0 to same platform

### "Admin button not showing"
**Fix:**
- Clear browser cache
- Check is_admin in database
- Verify JWT token is valid
- Check browser console for errors

### "Submissions always pending"
**Fix:**
- Check Judge0 is accessible
- Verify Judge0_URL is correct
- Check Judge0 logs for errors
- Test Judge0 directly: `curl JUDGE0_URL/about`

---

## 📈 Scaling Considerations

### When you outgrow free tier:

1. **Upgrade Database**
   - More connections
   - Better performance
   - Automated backups

2. **Dedicated Judge0**
   - Separate server
   - More workers
   - Resource limits increased

3. **Add Caching**
   - Redis for leaderboard
   - CDN for static files
   - Database query caching

4. **Load Balancing**
   - Multiple app instances
   - Session management
   - Sticky sessions if needed

---

## 🎯 Success Criteria

Your deployment is successful when:

- [ ] Application loads without errors
- [ ] Users can sign up and log in
- [ ] Admins can create contests
- [ ] Code submissions work correctly
- [ ] Leaderboard updates properly
- [ ] No runtime errors in logs
- [ ] Response times < 500ms
- [ ] Judge0 verdicts are accurate

---

## 📞 Getting Help

If you encounter issues:

1. Check logs first:
   - Render: Dashboard → Logs
   - Fly.io: `flyctl logs`
   - Railway: Dashboard → Logs

2. Verify environment variables are set

3. Test each component individually:
   - Database connection
   - Judge0 connectivity
   - API endpoints

4. Common commands:
```bash
# Render
# View logs in dashboard

# Fly.io  
flyctl logs
flyctl ssh console
flyctl postgres connect

# Railway
# View logs in dashboard
```

---

**Ready to deploy? Choose your platform and follow the steps above!**

**Estimated deployment time:**
- Render.com: 20-30 minutes
- Fly.io: 15-20 minutes
- Railway.app: 10-15 minutes
