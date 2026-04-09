package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/user/leetcode-cli/internal/config"
)

var langExtension = map[string]string{
	"golang":     "go",
	"go":         "go",
	"python":     "py",
	"python3":    "py",
	"javascript": "js",
	"typescript": "ts",
	"java":       "java",
	"cpp":        "cpp",
	"c":          "c",
	"rust":       "rs",
}

func SolutionPath(cfg *config.Config, questionNum, titleSlug, lang string) (string, error) {
	dir, err := cfg.SolutionsDirResolved()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("cannot create solutions dir %s: %w", dir, err)
	}

	ext, ok := langExtension[strings.ToLower(lang)]
	if !ok {
		ext = lang
	}

	filename := fmt.Sprintf("%s-%s.%s", questionNum, titleSlug, ext)
	return filepath.Join(dir, filename), nil
}

func SaveSolution(cfg *config.Config, questionNum, titleSlug, lang, code string) (string, error) {
	path, err := SolutionPath(cfg, questionNum, titleSlug, lang)
	if err != nil {
		return "", err
	}
	return path, os.WriteFile(path, []byte(code), 0644)
}

func LoadSolution(cfg *config.Config, questionNum, titleSlug, lang string) (string, error) {
	path, err := SolutionPath(cfg, questionNum, titleSlug, lang)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func SolutionExists(cfg *config.Config, questionNum, titleSlug, lang string) bool {
	path, err := SolutionPath(cfg, questionNum, titleSlug, lang)
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}

func ReadFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("cannot read file %s: %w", path, err)
	}
	return string(data), nil
}
