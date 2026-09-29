package sop

import "strings"

func splitLines(content string) []string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.TrimSuffix(content, "\n")
	if content == "" {
		return nil
	}
	return strings.Split(content, "\n")
}

// diffLines returns a line-oriented diff of old against new.
func diffLines(oldLines, newLines []string) []DiffLine {
	n, m := len(oldLines), len(newLines)
	dp := make([][]int, n+1)
	for i := range dp {
		dp[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if oldLines[i] == newLines[j] {
				dp[i][j] = dp[i+1][j+1] + 1
			} else if dp[i+1][j] >= dp[i][j+1] {
				dp[i][j] = dp[i+1][j]
			} else {
				dp[i][j] = dp[i][j+1]
			}
		}
	}

	var lines []DiffLine
	i, j := 0, 0
	for i < n && j < m {
		if oldLines[i] == newLines[j] {
			lines = append(lines, DiffLine{Kind: "context", Text: oldLines[i]})
			i++
			j++
			continue
		}
		if dp[i+1][j] >= dp[i][j+1] {
			lines = append(lines, DiffLine{Kind: "del", Text: oldLines[i]})
			i++
			continue
		}
		lines = append(lines, DiffLine{Kind: "add", Text: newLines[j]})
		j++
	}
	for i < n {
		lines = append(lines, DiffLine{Kind: "del", Text: oldLines[i]})
		i++
	}
	for j < m {
		lines = append(lines, DiffLine{Kind: "add", Text: newLines[j]})
		j++
	}
	if lines == nil {
		lines = []DiffLine{}
	}
	return lines
}
