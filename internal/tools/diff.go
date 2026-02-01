package tools

import (
	"fmt"
	"strings"
)

func GenerateUnifiedDiff(oldContent, newContent, filename string) string {
	oldLines := strings.Split(oldContent, "\n")
	newLines := strings.Split(newContent, "\n")

	var b strings.Builder
	b.WriteString(fmt.Sprintf("--- %s\n", filename))
	b.WriteString(fmt.Sprintf("+++ %s\n", filename))

	// Find changed regions and display with context
	i, j := 0, 0
	for i < len(oldLines) || j < len(newLines) {
		if i < len(oldLines) && j < len(newLines) && oldLines[i] == newLines[j] {
			// Context line - only show if near a change
			if isNearChange(oldLines, newLines, i, j, 3) {
				b.WriteString(fmt.Sprintf(" %s\n", oldLines[i]))
			}
			i++
			j++
		} else {
			// Find divergence
			oldEnd, newEnd := findMatchPoint(oldLines, newLines, i, j)

			for k := i; k < oldEnd && k < len(oldLines); k++ {
				b.WriteString(fmt.Sprintf("-%s\n", oldLines[k]))
			}
			for k := j; k < newEnd && k < len(newLines); k++ {
				b.WriteString(fmt.Sprintf("+%s\n", newLines[k]))
			}

			i = oldEnd
			j = newEnd
		}
	}

	return b.String()
}

func FormatNewFile(path, content string) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("+++ %s (new file)\n", path))
	for _, line := range strings.Split(content, "\n") {
		b.WriteString(fmt.Sprintf("+%s\n", line))
	}
	return b.String()
}

func FormatDeleteFile(path, content string) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("--- %s (deleted)\n", path))
	for _, line := range strings.Split(content, "\n") {
		b.WriteString(fmt.Sprintf("-%s\n", line))
	}
	return b.String()
}

func FormatBashCommand(command string) string {
	return fmt.Sprintf("$ %s", command)
}

func isNearChange(oldLines, newLines []string, oi, ni, radius int) bool {
	for d := 1; d <= radius; d++ {
		// Check if there's a mismatch within radius lines
		if oi+d < len(oldLines) && ni+d < len(newLines) && oldLines[oi+d] != newLines[ni+d] {
			return true
		}
		if oi-d >= 0 && ni-d >= 0 && oldLines[oi-d] != newLines[ni-d] {
			return true
		}
	}
	return false
}

func findMatchPoint(oldLines, newLines []string, oi, ni int) (int, int) {
	// Simple approach: advance through non-matching lines until we find a match
	maxLook := 50

	for d := 1; d < maxLook; d++ {
		// Try advancing old
		if oi+d < len(oldLines) && ni < len(newLines) {
			if oldLines[oi+d] == newLines[ni] {
				return oi + d, ni
			}
		}
		// Try advancing new
		if ni+d < len(newLines) && oi < len(oldLines) {
			if newLines[ni+d] == oldLines[oi] {
				return oi, ni + d
			}
		}
		// Try advancing both
		if oi+d < len(oldLines) && ni+d < len(newLines) {
			if oldLines[oi+d] == newLines[ni+d] {
				return oi + d, ni + d
			}
		}
	}

	// No match found, consume remaining
	return len(oldLines), len(newLines)
}
