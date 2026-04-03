package services

// trimWhitespace normalizes stdout/expected output by trimming:
// - leading/trailing whitespace
// - and normalizing line breaks
//
// Keeping logic similar to the previous handler implementation so verdicts stay consistent.
func trimWhitespace(s string) string {
	lines := []string{}
	currentLine := ""

	for _, char := range s {
		if char == '\n' || char == '\r' {
			if len(currentLine) > 0 {
				lines = append(lines, currentLine)
				currentLine = ""
			}
		} else {
			currentLine += string(char)
		}
	}

	if len(currentLine) > 0 {
		lines = append(lines, currentLine)
	}

	// Join lines with newline
	result := ""
	for i, line := range lines {
		if i > 0 {
			result += "\n"
		}
		result += line
	}

	return result
}

