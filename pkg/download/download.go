package download

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nodewee/llm-caller/pkg/utils"
)

// GitHubDownloader handles downloading files from GitHub URLs
type GitHubDownloader struct {
	client *http.Client
}

// NewGitHubDownloader creates a new GitHub downloader
func NewGitHubDownloader() *GitHubDownloader {
	return &GitHubDownloader{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ConvertToRawURL converts a GitHub blob URL to raw content URL
// Example: https://github.com/nodewee/llm-calling-templates/blob/main/qwen-vl-ocr-image.json
// Becomes: https://raw.githubusercontent.com/nodewee/llm-calling-templates/main/qwen-vl-ocr-image.json
func (d *GitHubDownloader) ConvertToRawURL(githubURL string) (string, error) {
	parsedURL, err := url.Parse(githubURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}

	if parsedURL.Host != "github.com" {
		return "", fmt.Errorf("URL must be from github.com, got: %s", parsedURL.Host)
	}

	// Parse path: /owner/repo/blob/branch/file
	pathParts := strings.Split(strings.Trim(parsedURL.Path, "/"), "/")
	if len(pathParts) < 5 {
		return "", fmt.Errorf("invalid GitHub URL format, expected: https://github.com/owner/repo/blob/branch/file")
	}

	if pathParts[2] != "blob" {
		return "", fmt.Errorf("URL must contain '/blob/', got: %s", githubURL)
	}

	owner := pathParts[0]
	repo := pathParts[1]
	branch := pathParts[3]
	filePath := strings.Join(pathParts[4:], "/")

	rawURL := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s", owner, repo, branch, filePath)
	return rawURL, nil
}

// DownloadTemplate downloads a template file from GitHub URL
func (d *GitHubDownloader) DownloadTemplate(githubURL, templateDir string) (string, error) {
	// Convert to raw URL
	rawURL, err := d.ConvertToRawURL(githubURL)
	if err != nil {
		return "", fmt.Errorf("failed to convert GitHub URL: %w", err)
	}

	// Extract filename from URL
	parsedURL, _ := url.Parse(rawURL)
	filename := filepath.Base(parsedURL.Path)
	if filename == "" || filename == "." {
		return "", fmt.Errorf("could not extract filename from URL: %s", githubURL)
	}

	// Ensure filename has .json extension
	if !strings.HasSuffix(filename, ".json") {
		filename += ".json"
	}

	// Create template directory if it doesn't exist
	if err := utils.CreateDirWithPlatformPermissions(templateDir); err != nil {
		return "", fmt.Errorf("failed to create template directory: %w", err)
	}

	// Download the file
	resp, err := d.client.Get(rawURL)
	if err != nil {
		return "", fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download file, status: %d %s", resp.StatusCode, resp.Status)
	}

	// Save to file
	destPath := filepath.Join(templateDir, filename)
	destFile, err := os.Create(destPath)
	if err != nil {
		return "", fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to save file: %w", err)
	}

	return destPath, nil
}

// ValidateTemplateFile validates that the downloaded file is a valid JSON template
func (d *GitHubDownloader) ValidateTemplateFile(filePath string) error {
	// Read the file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Basic validation - check if it's valid JSON and contains required fields
	if len(data) == 0 {
		return fmt.Errorf("file is empty")
	}

	// Check if it looks like JSON
	content := strings.TrimSpace(string(data))
	if !strings.HasPrefix(content, "{") || !strings.HasSuffix(content, "}") {
		return fmt.Errorf("file does not appear to be a JSON file")
	}

	return nil
}
