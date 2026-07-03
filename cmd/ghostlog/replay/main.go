package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
)

type fileEntry struct {
	Path         string `json:"path"`
	LinesAdded   int    `json:"lines_added"`
	LinesRemoved int    `json:"lines_removed"`
	IsTest       bool   `json:"is_test"`
	IsGenerated  bool   `json:"is_generated"`
	FuncCount    int    `json:"func_count,omitempty"`
}

type commitRecord struct {
	Timestamp    time.Time   `json:"timestamp"`
	CommitHash   string      `json:"commit_hash"`
	FilesChanged []fileEntry `json:"files_changed"`
}

func main() {
	logPath := flag.String("log", ".ghostlog.jsonl", "path to the JSONL log file")
	flag.Parse()

	f, err := os.Open(*logPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "replay: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	var records []commitRecord
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var r commitRecord
		if err := json.Unmarshal([]byte(line), &r); err != nil {
			fmt.Fprintf(os.Stderr, "replay: malformed line: %v\n", err)
			continue
		}
		records = append(records, r)
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "replay: read error: %v\n", err)
		os.Exit(1)
	}

	if len(records) == 0 {
		fmt.Fprintln(os.Stderr, "replay: no records found")
		return
	}

	fmt.Printf("ghostlog session — %d commits\n", len(records))
	fmt.Println(strings.Repeat("─", 60))

	for i, r := range records {
		totalAdded, totalRemoved := 0, 0
		for _, fe := range r.FilesChanged {
			totalAdded += fe.LinesAdded
			totalRemoved += fe.LinesRemoved
		}

		fmt.Printf("\n[%02d] %s  %s\n", i+1, r.CommitHash, r.Timestamp.Local().Format("2006-01-02 15:04:05"))
		fmt.Printf("     +%-4d -%d across %d file(s)\n", totalAdded, totalRemoved, len(r.FilesChanged))

		for _, fe := range r.FilesChanged {
			tags := fileTags(fe)
			if tags != "" {
				fmt.Printf("     %-48s %s\n", fe.Path, tags)
			} else {
				fmt.Printf("     %s\n", fe.Path)
			}
		}
	}

	fmt.Printf("\n%s\n", strings.Repeat("─", 60))
	if len(records) > 1 {
		span := records[len(records)-1].Timestamp.Sub(records[0].Timestamp).Round(time.Second)
		fmt.Printf("session duration: %s\n", span)
	}
}

func fileTags(fe fileEntry) string {
	var tags []string
	if fe.IsTest {
		tags = append(tags, "[test]")
	}
	if fe.IsGenerated {
		tags = append(tags, "[generated]")
	}
	if fe.FuncCount > 0 {
		tags = append(tags, fmt.Sprintf("[funcs:%d]", fe.FuncCount))
	}
	return strings.Join(tags, " ")
}
