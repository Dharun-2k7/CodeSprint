package services

import (
	"codesprint/database"
	"codesprint/judge"
	"codesprint/models"
	"fmt"
	"time"
)

type TestExecutionResult struct {
	TestcaseID int    `json:"testcase_id"`
	Status      string `json:"status"` // accepted / wrong_answer / time_limit_exceeded
	Stdout      string `json:"stdout"`
	Expected    string `json:"expected"`
	Matches     bool   `json:"matches"`
	RuntimeMS   int    `json:"runtime_ms"`
}

type RunResult struct {
	Success      bool                   `json:"success"`
	Verdict      string                 `json:"verdict"` // AC / WA / TLE
	RuntimeMS    int                    `json:"runtime_ms"`
	TestResults  []TestExecutionResult `json:"test_results"`
	SampleCount  int                    `json:"sample_count"`
}

func RunSamples(problemID int, language string, code string) (*RunResult, error) {
	// Fetch sample testcases
	rows, err := database.DB.Query(
		"SELECT id, input, expected_output FROM testcases WHERE problem_id = $1 AND is_sample = true ORDER BY id",
		problemID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch sample testcases: %w", err)
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
		return nil, fmt.Errorf("no sample testcases found")
	}

	languageID := judge.GetLanguageID(language)

	results := make([]TestExecutionResult, 0, len(testcases))
	overallVerdict := "AC"
	runtimeMS := 0

	for _, tc := range testcases {
		result, err := judge.SubmitCode(code, languageID, tc.Input)
		if err != nil {
			// Treat judge submission failures as TLE for verdict purposes.
			results = append(results, TestExecutionResult{
				TestcaseID: tc.ID,
				Status:     "time_limit_exceeded",
				Stdout:     "",
				Expected:   tc.ExpectedOutput,
				Matches:    false,
				RuntimeMS:  0,
			})
			overallVerdict = "TLE"
			continue
		}

		pollResult, err := judge.PollSubmissionResult(result.Token, 30, time.Second*2)
		if err != nil || pollResult == nil || pollResult.Status == nil {
			results = append(results, TestExecutionResult{
				TestcaseID: tc.ID,
				Status:     "time_limit_exceeded",
				Stdout:     "",
				Expected:   tc.ExpectedOutput,
				Matches:    false,
				RuntimeMS:  0,
			})
			overallVerdict = "TLE"
			continue
		}

		internalStatus := judge.MapJudge0StatusToInternal(pollResult.Status.ID)

		// Parse runtime (Judge0 returns time in seconds as a string like "0.001")
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
		finalStatus := internalStatus

		if internalStatus == "accepted" {
			matches = output == expected
			if !matches {
				finalStatus = "wrong_answer"
			}
		}

		// Decide per-sample verdict contribution
		if finalStatus == "wrong_answer" && overallVerdict != "TLE" {
			overallVerdict = "WA"
		}
		if finalStatus == "time_limit_exceeded" {
			overallVerdict = "TLE"
		}

		results = append(results, TestExecutionResult{
			TestcaseID: tc.ID,
			Status:     finalStatus,
			Stdout:     pollResult.Stdout,
			Expected:   tc.ExpectedOutput,
			Matches:    matches,
			RuntimeMS:  execRuntimeMS,
		})
	}

	return &RunResult{
		Success:     true,
		Verdict:     overallVerdict,
		RuntimeMS:   runtimeMS,
		TestResults: results,
		SampleCount: len(testcases),
	}, nil
}

