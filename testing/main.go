package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), "QRUPI API integration test runner")
		fmt.Fprintln(flag.CommandLine.Output(), "\nUsage:\n  go run ./testing")
		fmt.Fprintln(flag.CommandLine.Output(), "\nConfiguration is read from .env; see testing/README.md.")
	}
	flag.Parse()
	_ = godotenv.Load()
	runner := NewRunner(ConfigFromEnv())
	runSuite(runner)

	jsonPath, htmlPath, err := runner.WriteReports("testing/reports")
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to write reports:", err)
		os.Exit(1)
	}
	fmt.Printf("\nJSON report : %s\nHTML report : %s\n", jsonPath, htmlPath)
	if runner.Failed() {
		os.Exit(1)
	}
}
