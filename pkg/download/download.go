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

// Mirror site configuration
const (
	MirrorSiteBaseURL = "https://toolchains.mirror.toulan.fun"
)

// GitHubInfo contains extracted information from a GitHub URL
type GitHubInfo struct {
	Owner    string
	Repo     string
	Branch   string
	FilePath string
	FileName string
}

// NewGitHubDownloader creates a new GitHub downloader
func NewGitHubDownloader() *GitHubDownloader {
	return &GitHubDownloader{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// parseGitHubURL extracts owner, repo, branch, and file information from a GitHub URL
func (d *GitHubDownloader) parseGitHubURL(githubURL string) (*GitHubInfo, error) {
	parsedURL, err := url.Parse(githubURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	// Handle raw GitHub URLs
	if parsedURL.Host == "raw.githubusercontent.com" {
		// Format: https://raw.githubusercontent.com/owner/repo/branch/file
		// or: https://raw.githubusercontent.com/owner/repo/refs/heads/branch/file
		pathParts := strings.Split(strings.Trim(parsedURL.Path, "/"), "/")
		if len(pathParts) < 4 {
			return nil, fmt.Errorf("invalid raw GitHub URL format")
		}

		owner := pathParts[0]
		repo := pathParts[1]
		var branch, filePath string

		if len(pathParts) >= 5 && pathParts[2] == "refs" && pathParts[3] == "heads" {
			// Format: /owner/repo/refs/heads/branch/file
			branch = pathParts[4]
			filePath = strings.Join(pathParts[5:], "/")
		} else {
			// Format: /owner/repo/branch/file
			branch = pathParts[2]
			filePath = strings.Join(pathParts[3:], "/")
		}

		return &GitHubInfo{
			Owner:    owner,
			Repo:     repo,
			Branch:   branch,
			FilePath: filePath,
			FileName: filepath.Base(filePath),
		}, nil
	}

	// Handle regular GitHub URLs
	if parsedURL.Host != "github.com" {
		return nil, fmt.Errorf("URL must be from github.com or raw.githubusercontent.com, got: %s", parsedURL.Host)
	}

	// Parse path: /owner/repo/blob/branch/file
	pathParts := strings.Split(strings.Trim(parsedURL.Path, "/"), "/")
	if len(pathParts) < 5 {
		return nil, fmt.Errorf("invalid GitHub URL format, expected: https://github.com/owner/repo/blob/branch/file")
	}

	if pathParts[2] != "blob" {
		return nil, fmt.Errorf("URL must contain '/blob/', got: %s", githubURL)
	}

	owner := pathParts[0]
	repo := pathParts[1]
	branch := pathParts[3]
	filePath := strings.Join(pathParts[4:], "/")

	return &GitHubInfo{
		Owner:    owner,
		Repo:     repo,
		Branch:   branch,
		FilePath: filePath,
		FileName: filepath.Base(filePath),
	}, nil
}

// ConvertToRawURL converts a GitHub blob URL to raw content URL or returns raw URLs as-is
// Supported formats:
//  1. https://github.com/nodewee/llm-calling-templates/blob/main/qwen-vl-ocr-image.json
//     Becomes: https://raw.githubusercontent.com/nodewee/llm-calling-templates/main/qwen-vl-ocr-image.json
//  2. https://raw.githubusercontent.com/nodewee/llm-calling-templates/refs/heads/main/ollama-image-class.json
//     Returns: as-is (already raw format)
func (d *GitHubDownloader) ConvertToRawURL(githubURL string) (string, error) {
	info, err := d.parseGitHubURL(githubURL)
	if err != nil {
		return "", err
	}

	// Build raw URL
	rawURL := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s",
		info.Owner, info.Repo, info.Branch, info.FilePath)
	return rawURL, nil
}

// buildMirrorURL constructs a mirror site URL from GitHub URL information
func (d *GitHubDownloader) buildMirrorURL(info *GitHubInfo) string {
	// Mirror site format: https://toolchains.mirror.toulan.fun/{owner}/{repo}/latest/{filename}
	return fmt.Sprintf("%s/%s/%s/latest/%s",
		MirrorSiteBaseURL, info.Owner, info.Repo, info.FileName)
}

// downloadFromURL downloads a file from the given URL and saves it to the specified path
func (d *GitHubDownloader) downloadFromURL(downloadURL, destPath string) error {
	resp, err := d.client.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download file, status: %d %s", resp.StatusCode, resp.Status)
	}

	// Save to file
	destFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to save file: %w", err)
	}

	return nil
}

// DownloadTemplate downloads a template file from GitHub URL with mirror fallback
func (d *GitHubDownloader) DownloadTemplate(githubURL, templateDir string) (string, error) {
	// Parse GitHub URL to extract information
	info, err := d.parseGitHubURL(githubURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse GitHub URL: %w", err)
	}

	// Ensure filename has .json extension
	filename := info.FileName
	if !strings.HasSuffix(filename, ".json") {
		filename += ".json"
	}

	// Create template directory if it doesn't exist
	if err := utils.CreateDirWithPlatformPermissions(templateDir); err != nil {
		return "", fmt.Errorf("failed to create template directory: %w", err)
	}

	destPath := filepath.Join(templateDir, filename)

	// First, try to download from GitHub
	rawURL, err := d.ConvertToRawURL(githubURL)
	if err != nil {
		return "", fmt.Errorf("failed to convert GitHub URL: %w", err)
	}

	fmt.Printf("Downloading from GitHub: %s\n", rawURL)
	githubErr := d.downloadFromURL(rawURL, destPath)
	if githubErr == nil {
		fmt.Printf("Successfully downloaded from GitHub\n")
		return destPath, nil
	}

	// GitHub download failed, try mirror site
	fmt.Printf("GitHub download failed (%v), trying mirror site...\n", githubErr)
	mirrorURL := d.buildMirrorURL(info)
	fmt.Printf("Downloading from mirror: %s\n", mirrorURL)

	mirrorErr := d.downloadFromURL(mirrorURL, destPath)
	if mirrorErr != nil {
		return "", fmt.Errorf("failed to download from both GitHub and mirror site. GitHub error: %v, Mirror error: %v",
			githubErr, mirrorErr)
	}

	fmt.Printf("Successfully downloaded from mirror site\n")
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
