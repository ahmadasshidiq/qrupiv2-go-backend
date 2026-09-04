package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"time"
)

type Report struct {
	StartedAt time.Time `json:"started_at"`
	EndedAt   time.Time `json:"ended_at"`
	BaseURL   string    `json:"base_url"`
	Passed    int       `json:"passed"`
	Failed    int       `json:"failed"`
	Skipped   int       `json:"skipped"`
	Results   []Result  `json:"results"`
}

func (r *Runner) WriteReports(directory string) (string, string, error) {
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return "", "", err
	}
	report := Report{StartedAt: r.StartedAt, EndedAt: time.Now(), BaseURL: r.Config.BaseURL, Results: r.Results}
	for _, result := range report.Results {
		switch result.Status {
		case "PASS":
			report.Passed++
		case "FAIL":
			report.Failed++
		case "SKIP":
			report.Skipped++
		}
	}
	timestamp := time.Now().Format("20060102-150405")
	jsonPath := filepath.Join(directory, "report-"+timestamp+".json")
	htmlPath := filepath.Join(directory, "report-"+timestamp+".html")
	encoded, _ := json.MarshalIndent(report, "", "  ")
	if err := os.WriteFile(jsonPath, encoded, 0o644); err != nil {
		return "", "", err
	}
	if err := writeHTMLReport(htmlPath, report); err != nil {
		return "", "", err
	}
	_ = os.WriteFile(filepath.Join(directory, "latest.json"), encoded, 0o644)
	_ = writeHTMLReport(filepath.Join(directory, "latest.html"), report)
	return jsonPath, htmlPath, nil
}

func writeHTMLReport(path string, report Report) error {
	const page = `<!doctype html><html><head><meta charset="utf-8"><title>QRUPI API Test Report</title><style>
body{font:14px system-ui;margin:32px;color:#1f2937}table{border-collapse:collapse;width:100%}th,td{padding:9px;border:1px solid #ddd;text-align:left}.PASS{color:#087f23}.FAIL{color:#c62828}.SKIP{color:#996515}pre{white-space:pre-wrap;max-width:700px}h1{margin-bottom:4px}</style></head><body>
<h1>QRUPI API Integration Report</h1><p>{{.BaseURL}} · {{.StartedAt.Format "02 Jan 2006 15:04:05"}}</p>
<h3>PASS {{.Passed}} · FAIL {{.Failed}} · SKIP {{.Skipped}}</h3><table><tr><th>Status</th><th>Test</th><th>Request</th><th>HTTP</th><th>Duration</th><th>Details</th></tr>
{{range .Results}}<tr><td class="{{.Status}}">{{.Status}}</td><td>{{.Name}}</td><td>{{.Method}} {{.Path}}</td><td>{{.HTTPStatus}}</td><td>{{.DurationMS}} ms</td><td><pre>{{if .Message}}{{.Message}}{{else}}{{.Response}}{{end}}</pre></td></tr>{{end}}</table></body></html>`
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	tmpl, err := template.New("report").Parse(page)
	if err != nil {
		return err
	}
	if err := tmpl.Execute(file, report); err != nil {
		return fmt.Errorf("render HTML report: %w", err)
	}
	return nil
}
