package util

import (
	"regexp"
	"strings"
)

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

// IsEmptyOrInvisible returns true if the string is empty or contains only ANSI escape sequences.
func IsEmptyOrInvisible(s string) bool {
	return strings.TrimSpace(ansiRe.ReplaceAllString(s, "")) == ""
}

// TrimEmptyLine trims leading and trailing lines that are empty or contain only ANSI codes.
func TrimEmptyLine(content string) string {
	// Trim leading empty lines
	for {
		index := strings.Index(content, "\n")
		if index == -1 {
			break
		}
		if IsEmptyOrInvisible(content[:index]) {
			content = content[index+1:]
		} else {
			break
		}
	}
	// Trim trailing empty lines
	for {
		index := strings.LastIndex(content, "\n")
		if index == -1 {
			break
		}
		if IsEmptyOrInvisible(content[index+1:]) {
			content = content[:index]
		} else {
			break
		}
	}
	return content
}
