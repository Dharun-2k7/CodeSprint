-- Database schema for Codesprint 

-- Users table
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Contests table
CREATE TABLE IF NOT EXISTS contests (
    id SERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    start_time TIMESTAMP NOT NULL,
    end_time TIMESTAMP NOT NULL,
    created_by INTEGER REFERENCES users(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Problems table
CREATE TABLE IF NOT EXISTS problems (
    id SERIAL PRIMARY KEY,
    contest_id INTEGER REFERENCES contests(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    statement TEXT NOT NULL,
    time_limit INTEGER DEFAULT 1000, -- milliseconds
    memory_limit INTEGER DEFAULT 256, -- MB
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Testcases table
CREATE TABLE IF NOT EXISTS testcases (
    id SERIAL PRIMARY KEY,
    problem_id INTEGER REFERENCES problems(id) ON DELETE CASCADE,
    input TEXT NOT NULL,
    expected_output TEXT NOT NULL,
    is_sample BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Submissions table
CREATE TABLE IF NOT EXISTS submissions (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    problem_id INTEGER REFERENCES problems(id),
    contest_id INTEGER REFERENCES contests(id),
    language VARCHAR(50) NOT NULL,
    code TEXT NOT NULL,
    status VARCHAR(50) DEFAULT 'pending', -- pending, running, accepted, wrong_answer, time_limit_exceeded, runtime_error, compilation_error
    score INTEGER DEFAULT 0,
    runtime INTEGER DEFAULT 0, -- milliseconds
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Leaderboard cache table (optional, for performance)
CREATE TABLE IF NOT EXISTS leaderboard_cache (
    contest_id INTEGER REFERENCES contests(id) ON DELETE CASCADE,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    solved_count INTEGER DEFAULT 0,
    penalty INTEGER DEFAULT 0, -- total penalty in minutes
    last_submission_time TIMESTAMP,
    PRIMARY KEY (contest_id, user_id)
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_submissions_user_contest ON submissions(user_id, contest_id);
CREATE INDEX IF NOT EXISTS idx_submissions_problem ON submissions(problem_id);
CREATE INDEX IF NOT EXISTS idx_submissions_status ON submissions(status);
CREATE INDEX IF NOT EXISTS idx_leaderboard_contest ON leaderboard_cache(contest_id);
CREATE INDEX IF NOT EXISTS idx_problems_contest ON problems(contest_id);

