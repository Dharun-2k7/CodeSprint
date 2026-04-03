package services

import (
	"codesprint/database"
	"codesprint/judge"
	"codesprint/models"
	"fmt"
	"time"
)

type SubmissionTestResult struct {
	TestcaseID int    `json:"testcase_id"`
	Status     string `json:"status"` // accepted / wrong_answer / time_limit_exceeded
	Stdout     string `json:"stdout"`
	Expected   string `json:"expected"`
	Matches    bool   `json:"matches"`
	RuntimeMS  int    `json:"runtime_ms"`
}

type SubmitResult struct {
	Success     bool                    `json:"success"`
	SubmissionID int                   `json:"submission_id"`
	Verdict     string                 `json:"verdict"` // AC / WA / TLE
	Score       int                    `json:"score"`
	RuntimeMS   int                    `json:"runtime_ms"`
	TestResults []SubmissionTestResult `json:"test_results"`
}

// SubmitAndJudge runs against all testcases for a problem, stores final results in DB, and returns verdict synchronously.
func SubmitAndJudge(userID, problemID, contestID int, language, code string) (*SubmitResult, error) {
	// Fetch all testcases
	rows, err := database.DB.Query(
		"SELECT id, input, expected_output FROM testcases WHERE problem_id = $1 ORDER BY id",
		problemID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch testcases: %w", err)
	}
	defer rows.Close()

	var testcases []models.Testcase
	for rows.Next() {
		var tc models.Testcase
		if err := rows.Scan(&tc.ID, &tc.Input, &tc.ExpectedOutput); err != nil {
			continue
		}
		testcases = append(testcases, tc)
	}

	if len(testcases) == 0 {
		return nil, fmt.Errorf("no testcases found for this problem")
	}

	// Create submission record
	var submissionID int
	err = database.DB.QueryRow(
		"INSERT INTO submissions (user_id, problem_id, contest_id, language, code, status) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id",
		userID, problemID, contestID, language, code, "pending",
	).Scan(&submissionID)
	if err != nil {
		return nil, fmt.Errorf("failed to create submission: %w", err)
	}

	languageID := judge.GetLanguageID(language)

	overallVerdict := "AC"
	runtimeMS := 0
	allPassed := true
	// Canonical DB status to set on submissions row.
	finalDBStatus := "accepted"

	results := make([]SubmissionTestResult, 0, len(testcases))

	for _, tc := range testcases {
		result, err := judge.SubmitCode(code, languageID, tc.Input)
		if err != nil {
			// Treat judge submission failures as TLE.
			results = append(results, SubmissionTestResult{
				TestcaseID: tc.ID,
				Status:     "time_limit_exceeded",
				Stdout:     "",
				Expected:   tc.ExpectedOutput,
				Matches:    false,
				RuntimeMS:  0,
			})
			allPassed = false
			overallVerdict = "TLE"
			finalDBStatus = "time_limit_exceeded"
			continue
		}

		pollResult, err := judge.PollSubmissionResult(result.Token, 30, time.Second*2)
		if err != nil || pollResult == nil || pollResult.Status == nil {
			results = append(results, SubmissionTestResult{
				TestcaseID: tc.ID,
				Status:     "time_limit_exceeded",
				Stdout:     "",
				Expected:   tc.ExpectedOutput,
				Matches:    false,
				RuntimeMS:  0,
			})
			allPassed = false
			overallVerdict = "TLE"
			finalDBStatus = "time_limit_exceeded"
			continue
		}

		internalStatus := judge.MapJudge0StatusToInternal(pollResult.Status.ID)

		// Parse runtime (Judge0 returns time in seconds as string)
		execRuntimeMS := 0
		if pollResult.Time != "" {
			var runtimeSeconds float64
			if _, err := fmt.Sscanf(pollResult.Time, "%f", &runtimeSeconds); err == nil {
				execRuntimeMS = int(runtimeSeconds * 1000)
			}
		}
		if execRuntimeMS > runtimeMS {
			runtimeMS = execRuntimeMS
		}

		output := trimWhitespace(pollResult.Stdout)
		expected := trimWhitespace(tc.ExpectedOutput)

		matches := false
		finalStatusForThisTC := internalStatus
		if internalStatus == "accepted" {
			matches = output == expected
			if !matches {
				finalStatusForThisTC = "wrong_answer"
			}
		} else {
			// Anything non-accepted becomes non-passing.
			matches = false
		}

		// Verdict aggregation with priority: TLE > WA > AC.
		if finalStatusForThisTC == "wrong_answer" && overallVerdict != "TLE" {
			overallVerdict = "WA"
		}
		if finalStatusForThisTC == "time_limit_exceeded" || finalStatusForThisTC == "compilation_error" || finalStatusForThisTC == "runtime_error" || finalStatusForThisTC == "memory_limit_exceeded" {
			overallVerdict = "TLE"
			finalDBStatus = "time_limit_exceeded"
			allPassed = false
		}
		if finalStatusForThisTC != "accepted" {
			allPassed = false
		}

		results = append(results, SubmissionTestResult{
			TestcaseID: tc.ID,
			Status:     finalStatusForThisTC,
			Stdout:     pollResult.Stdout,
			Expected:   tc.ExpectedOutput,
			Matches:    matches,
			RuntimeMS:  execRuntimeMS,
		})

		// Update last seen token for debugging (best effort).
		_, _ = database.DB.Exec("UPDATE submissions SET judge0_token = $1 WHERE id = $2", result.Token, submissionID)
	}

	if allPassed && overallVerdict == "AC" {
		finalDBStatus = "accepted"
	} else if overallVerdict == "WA" {
		finalDBStatus = "wrong_answer"
	} else {
		finalDBStatus = "time_limit_exceeded"
	}

	score := 0
	if finalDBStatus == "accepted" {
		score = 100
	}

	_, err = database.DB.Exec(
		"UPDATE submissions SET status = $1, score = $2, runtime = $3 WHERE id = $4",
		finalDBStatus, score, runtimeMS, submissionID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update submission: %w", err)
	}

	// Update leaderboard cache
	updateLeaderboardCache(submissionID)

	return &SubmitResult{
		Success:      true,
		SubmissionID: submissionID,
		Verdict:      overallVerdict,
		Score:        score,
		RuntimeMS:    runtimeMS,
		TestResults:  results,
	}, nil
}

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
		return
	}

	// Get current leaderboard entry
	var solvedCount, penalty int
	var lastSubmissionTime time.Time
	err = database.DB.QueryRow(
		"SELECT solved_count, penalty, last_submission_time FROM leaderboard_cache WHERE contest_id = $1 AND user_id = $2",
		submission.ContestID, submission.UserID,
	).Scan(&solvedCount, &penalty, &lastSubmissionTime)

	contestStart, _ := getContestStartTime(submission.ContestID)
	penaltyMinutes := int(submission.CreatedAt.Sub(contestStart).Minutes())

	if err != nil {
		database.DB.Exec(
			"INSERT INTO leaderboard_cache (contest_id, user_id, solved_count, penalty, last_submission_time) VALUES ($1, $2, $3, $4, $5)",
			submission.ContestID, submission.UserID, 1, penaltyMinutes, submission.CreatedAt,
		)
	} else {
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

