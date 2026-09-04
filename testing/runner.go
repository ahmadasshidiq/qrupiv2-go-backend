package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"time"
)

type Config struct {
	BaseURL      string
	Email        string
	Password     string
	ProvinceCode string
	RegencyCode  string
}

func ConfigFromEnv() Config {
	return Config{
		BaseURL:      env("TEST_API_BASE_URL", "http://localhost:3000/api/v1"),
		Email:        env("TEST_EMAIL", env("SEED_EMAIL", "admin@admin.com")),
		Password:     env("TEST_PASSWORD", env("SEED_PASS", "Admin11234")),
		ProvinceCode: env("TEST_PROVINCE_CODE", "13"),
		RegencyCode:  env("TEST_REGENCY_CODE", "13.71"),
	}
}

type Result struct {
	Name       string `json:"name"`
	Method     string `json:"method"`
	Path       string `json:"path"`
	Status     string `json:"status"`
	HTTPStatus int    `json:"http_status,omitempty"`
	DurationMS int64  `json:"duration_ms"`
	Message    string `json:"message,omitempty"`
	Response   string `json:"response,omitempty"`
}

type Runner struct {
	Config    Config
	Client    *http.Client
	Token     string
	StartedAt time.Time
	Results   []Result
}

func NewRunner(config Config) *Runner {
	return &Runner{Config: config, Client: &http.Client{Timeout: 30 * time.Second}, StartedAt: time.Now()}
}

func (r *Runner) JSON(name, method, path string, body any, expected ...int) map[string]any {
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			r.failLocal(name, method, path, err)
			return nil
		}
		reader = bytes.NewReader(encoded)
	}
	return r.request(name, method, path, reader, "application/json", expected)
}

func (r *Runner) Form(name, method, path string, fields map[string][]string, expected ...int) map[string]any {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for key, values := range fields {
		for _, value := range values {
			_ = writer.WriteField(key, value)
		}
	}
	_ = writer.Close()
	return r.request(name, method, path, &body, writer.FormDataContentType(), expected)
}

func (r *Runner) request(name, method, path string, body io.Reader, contentType string, expected []int) map[string]any {
	started := time.Now()
	request, err := http.NewRequest(method, strings.TrimRight(r.Config.BaseURL, "/")+path, body)
	if err != nil {
		r.failLocal(name, method, path, err)
		return nil
	}
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}
	if r.Token != "" {
		request.Header.Set("Authorization", "Bearer "+r.Token)
	}
	response, err := r.Client.Do(request)
	if err != nil {
		r.Results = append(r.Results, Result{Name: name, Method: method, Path: path, Status: "FAIL", DurationMS: time.Since(started).Milliseconds(), Message: err.Error()})
		fmt.Printf("FAIL %-45s %s\n", name, err)
		return nil
	}
	defer response.Body.Close()
	raw, readErr := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if readErr != nil {
		raw = []byte(readErr.Error())
	}
	status := "PASS"
	if !containsStatus(expected, response.StatusCode) {
		status = "FAIL"
	}
	result := Result{Name: name, Method: method, Path: path, Status: status, HTTPStatus: response.StatusCode, DurationMS: time.Since(started).Milliseconds(), Response: truncate(string(raw), 4000)}
	if status == "FAIL" {
		result.Message = fmt.Sprintf("expected HTTP %v, received %d", expected, response.StatusCode)
	}
	r.Results = append(r.Results, result)
	fmt.Printf("%-4s %-45s HTTP %d (%d ms)\n", status, name, response.StatusCode, result.DurationMS)

	var decoded map[string]any
	if json.Unmarshal(raw, &decoded) != nil {
		return nil
	}
	return decoded
}

func (r *Runner) Assert(name string, passed bool, message string) {
	status := "PASS"
	if !passed {
		status = "FAIL"
	}
	r.Results = append(r.Results, Result{Name: name, Method: "ASSERT", Status: status, Message: message})
	fmt.Printf("%-4s %s\n", status, name)
}

func (r *Runner) Skip(name, reason string) {
	r.Results = append(r.Results, Result{Name: name, Status: "SKIP", Message: reason})
	fmt.Printf("SKIP %-45s %s\n", name, reason)
}

func (r *Runner) Failed() bool {
	for _, result := range r.Results {
		if result.Status == "FAIL" {
			return true
		}
	}
	return false
}

func (r *Runner) failLocal(name, method, path string, err error) {
	r.Results = append(r.Results, Result{Name: name, Method: method, Path: path, Status: "FAIL", Message: err.Error()})
}

func containsStatus(values []int, status int) bool {
	for _, value := range values {
		if value == status {
			return true
		}
	}
	return false
}

func dataString(response map[string]any) string {
	value, _ := response["data"].(string)
	return value
}

func nestedString(response map[string]any, keys ...string) string {
	var current any = response
	for _, key := range keys {
		object, ok := current.(map[string]any)
		if !ok {
			return ""
		}
		current = object[key]
	}
	value, _ := current.(string)
	return value
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func truncate(value string, max int) string {
	if len(value) <= max {
		return value
	}
	return value[:max] + "..."
}
