package services

import (
	"codesprint/database"
	"codesprint/models"
	"fmt"
	"time"
)

type SubmissionTestResult struct {
	TestcaseID int    `json:"testcase_id"`
	Status     string `json:"status"` // accepted / wrong_answer / runtime_error
	Stdout     string `json:"stdout"`
	Expected   string `json:"expected"`
	Matches    bool   `json:"matches"`
	RuntimeMS  int    `json:"runtime_ms"`
}

type SubmitResult struct {
	Success      bool                   `json:"success"`
	SubmissionID int                    `json:"submission_id"`
	Verdict      string                 `json:"verdict"` // AC / WA / RE
	Score        int                    `json:"score"`
	RuntimeMS    int                    `json:"runtime_ms"`
	TestResults  []SubmissionTestResult `json:"test_results"`
}

// SubmitAndJudge runs against all testcases explicitly using synchronous piston endpoints.
func SubmitAndJudge(userID, problemID, contestID int, language, code string) (*SubmitResult, error) {
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

	var submissionID int
	err = database.DB.QueryRow(
		"INSERT INTO submissions (user_id, problem_id, contest_id, language, code, status) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id",
		userID, problemID, contestID, language, code, "pending",
	).Scan(&submissionID)
	if err != nil {
		return nil, fmt.Errorf("failed to create submission: %w", err)
	}

	overallVerdict := "AC"
	maxRuntimeMS := 0
	allPassed := true
	finalDBStatus := "accepted"

	results := make([]SubmissionTestResult, 0, len(testcases))

	for _, tc := range testcases {
		stdout, stderr, status, latency, err := ExecuteCode(language, code, tc.Input)

		if latency > maxRuntimeMS {
			maxRuntimeMS = latency
		}

		if err != nil || status == "runtime_error" {
			results = append(results, SubmissionTestResult{
				TestcaseID: tc.ID,
				Status:     "runtime_error",
				Stdout:     stderr,
				Expected:   tc.ExpectedOutput,
				Matches:    false,
				RuntimeMS:  latency,
			})
			allPassed = false
			overallVerdict = "RE"
			finalDBStatus = "runtime_error"
			continue
		}

		output := trimWhitespace(stdout)
		expected := trimWhitespace(tc.ExpectedOutput)

		matches := false
		finalStatusForThisTC := "accepted"
		
		matches = output == expected
		if !matches {
			finalStatusForThisTC = "wrong_answer"
		}

		// Verdict Prioritisation RE > WA > AC.
		if finalStatusForThisTC == "wrong_answer" && overallVerdict != "RE" {
			overallVerdict = "WA"
            finalDBStatus  = "wrong_answer"
		}
		if finalStatusForThisTC != "accepted" {
			allPassed = false
		}

		results = append(results, SubmissionTestResult{
			TestcaseID: tc.ID,
			Status:     finalStatusForThisTC,
			Stdout:     stdout,
			Expected:   tc.ExpectedOutput,
			Matches:    matches,
			RuntimeMS:  latency,
		})
	}

	if allPassed && overallVerdict == "AC" {
		finalDBStatus = "accepted"
	}

	score := 0
	if finalDBStatus == "accepted" {
		score = 100
	}

	_, err = database.DB.Exec(
		"UPDATE submissions SET status = $1, score = $2, runtime = $3 WHERE id = $4",
		finalDBStatus, score, maxRuntimeMS, submissionID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update submission: %w", err)
	}

	updateLeaderboardCache(submissionID)

	return &SubmitResult{
		Success:      true,
		SubmissionID: submissionID,
		Verdict:      overallVerdict,
		Score:        score,
		RuntimeMS:    maxRuntimeMS,
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

	if submission.Status != "accepted" {
		return
	}

	var existingCount int
	err = database.DB.QueryRow(
		"SELECT COUNT(*) FROM submissions WHERE user_id = $1 AND contest_id = $2 AND problem_id = $3 AND status = 'accepted'",
		submission.UserID, submission.ContestID, submission.ProblemID,
	).Scan(&existingCount)
	if err != nil || existingCount > 1 {
		return
	}

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
