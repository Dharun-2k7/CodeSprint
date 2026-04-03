package judge

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

const rapidAPIHost = "judge0-ce.p.rapidapi.com"

var judge0URL = getJudge0URL()

func getJudge0URL() string {
	url := os.Getenv("JUDGE0_URL")
	if url == "" {
		return "https://judge0-ce.p.rapidapi.com"
	}
	return url
}

func rapidAPIKey() (string, error) {
	key := os.Getenv("RAPIDAPI_KEY")
	if key == "" {
		return "", fmt.Errorf("RAPIDAPI_KEY is not set")
	}
	return key, nil
}

var httpClient = &http.Client{
	Timeout: 30 * time.Second,
}

func applyRapidHeaders(req *http.Request) error {
	key, err := rapidAPIKey()
	if err != nil {
		return err
	}
	req.Header.Set("X-RapidAPI-Key", key)
	req.Header.Set("X-RapidAPI-Host", rapidAPIHost)
	return nil
}

// Language IDs for Judge0
const (
	LanguageC      = 50 // C (GCC 9.2.0)
	LanguageCPP    = 54 // C++ (GCC 9.2.0)
	LanguagePython = 71 // Python (3.8.1)
)

// Judge0Submission represents a submission to Judge0
type Judge0Submission struct {
	SourceCode string `json:"source_code"`
	LanguageID int    `json:"language_id"`
	Stdin      string `json:"stdin,omitempty"`
}

// Judge0Response represents a response from Judge0
type Judge0Response struct {
	Token         string        `json:"token"`
	Status        *Judge0Status `json:"status,omitempty"`
	Stdout        string        `json:"stdout,omitempty"`
	Stderr        string        `json:"stderr,omitempty"`
	Time          string        `json:"time,omitempty"`
	Memory        int           `json:"memory,omitempty"`
	CompileOutput string        `json:"compile_output,omitempty"`
}

// Judge0Status represents the status of a submission
type Judge0Status struct {
	ID          int    `json:"id"`
	Description string `json:"description"`
}

// SubmitCode submits code to Judge0
func SubmitCode(code string, languageID int, input string) (*Judge0Response, error) {
	submission := Judge0Submission{
		SourceCode: code,
		LanguageID: languageID,
		Stdin:      input,
	}

	jsonData, err := json.Marshal(submission)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal submission: %w", err)
	}

	url := fmt.Sprintf("%s/submissions?base64_encoded=false&wait=false", judge0URL)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create submission request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if err := applyRapidHeaders(req); err != nil {
		return nil, err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to submit to Judge0: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Judge0 returned status %d", resp.StatusCode)
	}

	var result Judge0Response
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// GetSubmissionResult retrieves the result of a submission from Judge0
func GetSubmissionResult(token string) (*Judge0Response, error) {
	url := fmt.Sprintf("%s/submissions/%s?base64_encoded=false", judge0URL, token)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create result request: %w", err)
	}
	if err := applyRapidHeaders(req); err != nil {
		return nil, err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get result from Judge0: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Judge0 returned status %d", resp.StatusCode)
	}

	var result Judge0Response
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// PollSubmissionResult polls Judge0 until the submission is complete
func PollSubmissionResult(token string, maxAttempts int, delay time.Duration) (*Judge0Response, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(maxAttempts)*delay)
	defer cancel()

	for i := 0; i < maxAttempts; i++ {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("submission timed out after %d attempts: %w", maxAttempts, ctx.Err())
		default:
		}

		result, err := GetSubmissionResult(token)
		if err != nil {
			return nil, err
		}

		// Status ID 1-2 means in queue or processing, 3 means completed
		if result.Status != nil && result.Status.ID >= 3 {
			return result, nil
		}

		t := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			t.Stop()
			return nil, fmt.Errorf("submission timed out after %d attempts: %w", maxAttempts, ctx.Err())
		case <-t.C:
		}
	}

	return nil, fmt.Errorf("submission timed out after %d attempts", maxAttempts)
}

// MapJudge0StatusToInternal maps Judge0 status to internal status
func MapJudge0StatusToInternal(judge0StatusID int) string {
	switch judge0StatusID {
	case 3: // Accepted
		return "accepted"
	case 4: // Wrong Answer
		return "wrong_answer"
	case 5: // Time Limit Exceeded
		return "time_limit_exceeded"
	case 6: // Compilation Error
		return "compilation_error"
	case 7: // Runtime Error
		return "runtime_error"
	case 8: // Memory Limit Exceeded
		return "memory_limit_exceeded"
	default:
		return "pending"
	}
}

// GetLanguageID maps language string to Judge0 language ID
func GetLanguageID(language string) int {
	switch language {
	case "c", "C":
		return LanguageC
	}

	// Tolerant mapping for frontend values like `python3`, `c++`.
	switch language {
	case "cpp", "c++", "C++", "CXX":
		return LanguageCPP
	case "python", "python3", "Python", "PYTHON3":
		return LanguagePython
	case "c", "C":
		return LanguageC
	}

	switch language {
	case "c":
		return LanguageC
	case "cpp", "c++":
		return LanguageCPP
	case "python", "python3":
		return LanguagePython
	default:
		return LanguageC // default to C
	}
}
