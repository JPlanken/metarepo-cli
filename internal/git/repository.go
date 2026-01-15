package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// RepoInfo holds information about a git repository
type RepoInfo struct {
	Name        string
	Path        string
	AbsPath     string
	URL         string
	Branch      string
	LastCommit  CommitInfo
	HasChanges  bool
	IsDetached  bool
	HasRemote   bool
}

// CommitInfo holds information about a commit
type CommitInfo struct {
	Hash    string
	Author  string
	Date    time.Time
	Message string
}

// IsGitRepo checks if a directory is a git repository
func IsGitRepo(path string) bool {
	gitDir := filepath.Join(path, ".git")
	info, err := os.Stat(gitDir)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// GetRepoInfo returns information about a git repository
func GetRepoInfo(repoPath string) (*RepoInfo, error) {
	if !IsGitRepo(repoPath) {
		return nil, fmt.Errorf("not a git repository: %s", repoPath)
	}

	absPath, err := filepath.Abs(repoPath)
	if err != nil {
		return nil, err
	}

	info := &RepoInfo{
		Name:    filepath.Base(absPath),
		Path:    repoPath,
		AbsPath: absPath,
	}

	// Get remote URL
	if url, err := runGitCommand(repoPath, "remote", "get-url", "origin"); err == nil {
		info.URL = strings.TrimSpace(url)
		info.HasRemote = true
	}

	// Get current branch
	if branch, err := runGitCommand(repoPath, "rev-parse", "--abbrev-ref", "HEAD"); err == nil {
		info.Branch = strings.TrimSpace(branch)
		if info.Branch == "HEAD" {
			info.IsDetached = true
		}
	}

	// Get last commit info
	if hash, err := runGitCommand(repoPath, "rev-parse", "--short", "HEAD"); err == nil {
		info.LastCommit.Hash = strings.TrimSpace(hash)
	}

	if author, err := runGitCommand(repoPath, "log", "-1", "--format=%an"); err == nil {
		info.LastCommit.Author = strings.TrimSpace(author)
	}

	if dateStr, err := runGitCommand(repoPath, "log", "-1", "--format=%ci"); err == nil {
		dateStr = strings.TrimSpace(dateStr)
		if t, err := time.Parse("2006-01-02 15:04:05 -0700", dateStr); err == nil {
			info.LastCommit.Date = t
		}
	}

	if msg, err := runGitCommand(repoPath, "log", "-1", "--format=%s"); err == nil {
		info.LastCommit.Message = strings.TrimSpace(msg)
	}

	// Check for changes
	if status, err := runGitCommand(repoPath, "status", "--porcelain"); err == nil {
		info.HasChanges = strings.TrimSpace(status) != ""
	}

	return info, nil
}

// ScanForRepos finds all git repositories in a directory
func ScanForRepos(rootPath string) ([]*RepoInfo, error) {
	var repos []*RepoInfo

	absRoot, err := filepath.Abs(rootPath)
	if err != nil {
		return nil, err
	}

	err = filepath.Walk(absRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		// Skip hidden directories except .git
		if info.IsDir() && strings.HasPrefix(info.Name(), ".") && info.Name() != ".git" {
			return filepath.SkipDir
		}

		// Skip common non-repo directories
		skipDirs := []string{"node_modules", "vendor", ".venv", "venv", "__pycache__", "dist", "build"}
		for _, skip := range skipDirs {
			if info.Name() == skip {
				return filepath.SkipDir
			}
		}

		// Check if this is a git repo
		if info.IsDir() && IsGitRepo(path) {
			if repoInfo, err := GetRepoInfo(path); err == nil {
				// Calculate relative path from root
				relPath, _ := filepath.Rel(absRoot, path)
				repoInfo.Path = relPath
				repos = append(repos, repoInfo)
			}
			// Don't descend into nested git repos
			return filepath.SkipDir
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return repos, nil
}

// Pull performs a git pull on the repository
func Pull(repoPath string) error {
	_, err := runGitCommand(repoPath, "pull")
	return err
}

// Push performs a git push on the repository
func Push(repoPath string) error {
	_, err := runGitCommand(repoPath, "push")
	return err
}

// Clone clones a repository
func Clone(url, destPath string) error {
	cmd := exec.Command("git", "clone", url, destPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// runGitCommand runs a git command in the specified directory
func runGitCommand(repoPath string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}
