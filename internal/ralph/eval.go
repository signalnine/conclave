package ralph

import (
	"path/filepath"
	"regexp"
	"strings"
)

var filteredPrefixes = []string{
	"node_modules/", "vendor/", ".venv/", "venv/",
	"__pycache__/", ".git/",
}

var textExtensions = map[string]bool{
	".go": true, ".js": true, ".ts": true, ".jsx": true, ".tsx": true,
	".py": true, ".rb": true, ".rs": true, ".java": true, ".c": true,
	".h": true, ".cpp": true, ".cs": true, ".php": true, ".swift": true,
	".kt": true, ".scala": true, ".sh": true, ".bash": true, ".zsh": true,
	".json": true, ".yaml": true, ".yml": true, ".toml": true, ".xml": true,
	".html": true, ".css": true, ".scss": true, ".less": true, ".sql": true,
	".md": true, ".txt": true, ".cfg": true, ".ini": true, ".env": true,
	".graphql": true, ".proto": true, ".vue": true, ".svelte": true,
}

func ExtractFilePathsFromOutput(output string) []string {
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`\(([a-zA-Z0-9_./-]+\.[a-zA-Z]+):\d+(?::\d+)?\)`),
		regexp.MustCompile(`(?m)^FAIL\s+([a-zA-Z0-9_./-]+\.[a-zA-Z]+)`),
		regexp.MustCompile(`File "([a-zA-Z0-9_./-]+\.[a-zA-Z]+)", line \d+`),
		regexp.MustCompile(`(?m)^FAILED\s+([a-zA-Z0-9_./-]+\.[a-zA-Z]+)::`),
		regexp.MustCompile(`(?m)^\t([a-zA-Z0-9_./-]+\.go):\d+`),
	}

	seen := make(map[string]bool)
	var result []string

	for _, re := range patterns {
		for _, match := range re.FindAllStringSubmatch(output, -1) {
			path := match[1]
			if seen[path] || isFiltered(path) || strings.HasPrefix(path, "node:") {
				continue
			}
			seen[path] = true
			result = append(result, path)
		}
	}
	return result
}

func isFiltered(path string) bool {
	for _, prefix := range filteredPrefixes {
		if strings.HasPrefix(path, prefix) || strings.Contains(path, "/"+prefix) {
			return true
		}
	}
	return false
}

func isTextFile(path string) bool {
	ext := filepath.Ext(path)
	return textExtensions[ext]
}

func isInsideDir(baseDir, path string) bool {
	abs, err := filepath.Abs(filepath.Join(baseDir, path))
	if err != nil {
		return false
	}
	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		return false
	}
	return strings.HasPrefix(abs, absBase+string(filepath.Separator)) || abs == absBase
}
