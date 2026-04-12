package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
)

var completionRe = regexp.MustCompile(`\s+✅\s+\d{4}-\d{2}-\d{2}\s*$`)

// toggleTask flips the checkbox at lineNum in filePath.
// complete=true: [ ] → [x] ✅ today
// complete=false: [x] ✅ date → [ ]
func toggleTask(filePath string, lineNum int, complete bool) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	lines := strings.Split(string(data), "\n")
	if lineNum >= len(lines) {
		return fmt.Errorf("line %d out of range (file has %d lines)", lineNum, len(lines))
	}

	line := lines[lineNum]

	if complete {
		line = strings.Replace(line, "- [ ]", "- [x]", 1)
		line = completionRe.ReplaceAllString(line, "")
		line = line + " ✅ " + time.Now().Format("2006-01-02")
	} else {
		line = strings.Replace(line, "- [x]", "- [ ]", 1)
		line = completionRe.ReplaceAllString(line, "")
	}

	lines[lineNum] = line
	return writeLines(filePath, lines)
}

func writeLines(path string, lines []string) error {
	content := strings.Join(lines, "\n")
	// ensure single trailing newline
	content = strings.TrimRight(content, "\n") + "\n"

	tmp := path + ".classtrack.tmp"
	if err := os.WriteFile(tmp, []byte(content), 0644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
