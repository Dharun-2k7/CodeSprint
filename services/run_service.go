package services

import (
	"codesprint/database"
	"codesprint/models"
	"fmt"
)

type TestExecutionResult struct {
	TestcaseID int    `json:"testcase_id"`
	Status     string `json:"status"` // accepted / wrong_answer / time_limit_exceeded (re-mapped to runtime_error for piston timeout)
	Stdout     string `json:"stdout"`
	Expected   string `json:"expected"`
	Matches    bool   `json:"matches"`
	RuntimeMS  int    `json:"runtime_ms"`
}

type RunResult struct {
	Success     bool                  `json:"success"`
	Verdict     string                `json:"verdict"` // AC / WA / RE
	RuntimeMS   int                   `json:"runtime_ms"`
	TestResults []TestExecutionResult `json:"test_results"`
	SampleCount int                   `json:"sample_count"`
}

func RunSamples(problemID int, language string, code string) (*RunResult, error) {
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

	results := make([]TestExecutionResult, 0, len(testcases))
	overallVerdict := "AC"
	maxRuntimeMS := 0

	for _, tc := range testcases {
		stdout, stderr, status, latency, err := ExecuteCode(language, code, tc.Input)
		if err != nil {
			// API execution error or system timeout
			results = append(results, TestExecutionResult{
				TestcaseID: tc.ID,
				Status:     "runtime_error", // Mapped generically in Piston logic
				Stdout:     stderr,
				Expected:   tc.ExpectedOutput,
				Matches:    false,
				RuntimeMS:  latency,
			})
			overallVerdict = "RE"
			if latency > maxRuntimeMS { maxRuntimeMS = latency }
			continue
		}

		if latency > maxRuntimeMS {
			maxRuntimeMS = latency
		}

		output := trimWhitespace(stdout)
		expected := trimWhitespace(tc.ExpectedOutput)
		matches := false
		finalStatus := status

		if status != "runtime_error" {
			matches = output == expected
			if matches {
				finalStatus = "accepted"
			} else {
				finalStatus = "wrong_answer"
			}
		}

		// Verdict priority RE > WA > AC
		if finalStatus == "wrong_answer" && overallVerdict != "RE" {
			overallVerdict = "WA"
		} else if finalStatus == "runtime_error" {
			overallVerdict = "RE"
		}

		results = append(results, TestExecutionResult{
			TestcaseID: tc.ID,
			Status:     finalStatus,
			Stdout:     stdout,
			Expected:   tc.ExpectedOutput,
			Matches:    matches,
			RuntimeMS:  latency,
		})
	}

	return &RunResult{
		Success:     true,
		Verdict:     overallVerdict,
		RuntimeMS:   maxRuntimeMS,
		TestResults: results,
		SampleCount: len(testcases),
	}, nil
}
