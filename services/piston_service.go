package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const pistonAPIHost = "https://emkc.org/api/v2/piston/execute"

type pistonFile struct {
	Content string `json:"content"`
}

type pistonRequest struct {
	Language string       `json:"language"`
	Version  string       `json:"version"`
	Files    []pistonFile `json:"files"`
	Stdin    string       `json:"stdin"`
}

type pistonResponseRun struct {
	Stdout string `json:"stdout"`
	Stderr string `json:"stderr"`
	Code   int    `json:"code"`
	Signal string `json:"signal"`
	Output string `json:"output"`
}

type pistonResponse struct {
	Language string            `json:"language"`
	Version  string            `json:"version"`
	Run      pistonResponseRun `json:"run"`
	Message  string            `json:"message"`
}

var httpClient = &http.Client{
	Timeout: 10 * time.Second, // Timeout to ensure requests bound logically.
}

// ExecuteCode sends synchronous requests to the Piston execution API
func ExecuteCode(language string, code string, input string) (output string, errOut string, status string, runtimeMs int, err error) {
	// Standardize language matching natively to Piston version arrays bounds.
	mappedLang := language
	if language == "c" {
		mappedLang = "c"
	} else if language == "cpp" || language == "c++" {
		mappedLang = "cpp"
	} else if language == "python" || language == "python3" {
		mappedLang = "python"
	}

	reqBody := pistonRequest{
		Language: mappedLang,
		Version:  "*", // Select latest matching compiler native fallback.
		Files: []pistonFile{
			{Content: code},
		},
		Stdin: input,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", "", "RE", 0, fmt.Errorf("failed to encode piston bounds payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, pistonAPIHost, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", "", "RE", 0, fmt.Errorf("failed creating HTTP execution instance: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	startTimer := time.Now()
	resp, err := httpClient.Do(req)
	latency := int(time.Since(startTimer).Milliseconds())

	if err != nil {
		return "", "", "RE", latency, fmt.Errorf("Piston executing loop error mapping: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", "RE", latency, fmt.Errorf("Piston response bounded %d", resp.StatusCode)
	}

	var res pistonResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", "", "RE", latency, fmt.Errorf("failed formatting json payload responses native bound")
	}

	if res.Message != "" {
		return "", res.Message, "RE", latency, fmt.Errorf(res.Message)
	}

	outputStatus := "AC"
	if res.Run.Code != 0 || res.Run.Stderr != "" {
		outputStatus = "runtime_error"
	}

	return res.Run.Stdout, res.Run.Stderr, outputStatus, latency, nil
}
