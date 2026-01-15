package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// RuntimeInfo holds information about detected runtimes/tools in a repo
type RuntimeInfo struct {
	Language string // python, node, go, rust, etc.
	Version  string // version if detectable
	Files    []string // config files found
}

// DetectRuntimes scans a repository for runtime/tool configurations
func DetectRuntimes(repoPath string) []RuntimeInfo {
	var runtimes []RuntimeInfo

	// Python detection
	if rt := detectPython(repoPath); rt != nil {
		runtimes = append(runtimes, *rt)
	}

	// Node.js detection
	if rt := detectNode(repoPath); rt != nil {
		runtimes = append(runtimes, *rt)
	}

	// Go detection
	if rt := detectGo(repoPath); rt != nil {
		runtimes = append(runtimes, *rt)
	}

	// Rust detection
	if rt := detectRust(repoPath); rt != nil {
		runtimes = append(runtimes, *rt)
	}

	return runtimes
}

func detectPython(repoPath string) *RuntimeInfo {
	var files []string
	version := ""

	// Check for Python indicators
	indicators := []string{
		"requirements.txt",
		"pyproject.toml",
		"setup.py",
		"Pipfile",
		".python-version",
	}

	for _, indicator := range indicators {
		path := filepath.Join(repoPath, indicator)
		if _, err := os.Stat(path); err == nil {
			files = append(files, indicator)

			// Try to extract version from .python-version
			if indicator == ".python-version" {
				if data, err := os.ReadFile(path); err == nil {
					version = strings.TrimSpace(string(data))
				}
			}

			// Try to extract version from pyproject.toml
			if indicator == "pyproject.toml" && version == "" {
				if data, err := os.ReadFile(path); err == nil {
					re := regexp.MustCompile(`python\s*=\s*"([^"]+)"`)
					if matches := re.FindStringSubmatch(string(data)); len(matches) > 1 {
						version = matches[1]
					}
				}
			}
		}
	}

	// Check for venv
	venvPaths := []string{".venv", "venv"}
	for _, venv := range venvPaths {
		if _, err := os.Stat(filepath.Join(repoPath, venv)); err == nil {
			files = append(files, venv+"/")
		}
	}

	if len(files) == 0 {
		return nil
	}

	// Try to get Python version from venv if not found
	if version == "" {
		for _, venv := range venvPaths {
			pythonPath := filepath.Join(repoPath, venv, "bin", "python")
			if _, err := os.Stat(pythonPath); err == nil {
				cmd := exec.Command(pythonPath, "--version")
				if output, err := cmd.Output(); err == nil {
					parts := strings.Fields(string(output))
					if len(parts) >= 2 {
						version = parts[1]
					}
				}
				break
			}
		}
	}

	return &RuntimeInfo{
		Language: "python",
		Version:  version,
		Files:    files,
	}
}

func detectNode(repoPath string) *RuntimeInfo {
	var files []string
	version := ""

	indicators := []string{
		"package.json",
		"package-lock.json",
		"yarn.lock",
		"pnpm-lock.yaml",
		".nvmrc",
		".node-version",
	}

	for _, indicator := range indicators {
		path := filepath.Join(repoPath, indicator)
		if _, err := os.Stat(path); err == nil {
			files = append(files, indicator)

			// Extract version from .nvmrc or .node-version
			if indicator == ".nvmrc" || indicator == ".node-version" {
				if data, err := os.ReadFile(path); err == nil {
					version = strings.TrimSpace(string(data))
				}
			}
		}
	}

	if len(files) == 0 {
		return nil
	}

	return &RuntimeInfo{
		Language: "node",
		Version:  version,
		Files:    files,
	}
}

func detectGo(repoPath string) *RuntimeInfo {
	var files []string
	version := ""

	goMod := filepath.Join(repoPath, "go.mod")
	if _, err := os.Stat(goMod); err == nil {
		files = append(files, "go.mod")

		// Extract Go version from go.mod
		if data, err := os.ReadFile(goMod); err == nil {
			re := regexp.MustCompile(`go\s+(\d+\.\d+)`)
			if matches := re.FindStringSubmatch(string(data)); len(matches) > 1 {
				version = matches[1]
			}
		}
	}

	if _, err := os.Stat(filepath.Join(repoPath, "go.sum")); err == nil {
		files = append(files, "go.sum")
	}

	if len(files) == 0 {
		return nil
	}

	return &RuntimeInfo{
		Language: "go",
		Version:  version,
		Files:    files,
	}
}

func detectRust(repoPath string) *RuntimeInfo {
	var files []string
	version := ""

	cargoToml := filepath.Join(repoPath, "Cargo.toml")
	if _, err := os.Stat(cargoToml); err == nil {
		files = append(files, "Cargo.toml")

		// Check rust-toolchain.toml for version
		toolchainPath := filepath.Join(repoPath, "rust-toolchain.toml")
		if data, err := os.ReadFile(toolchainPath); err == nil {
			files = append(files, "rust-toolchain.toml")
			re := regexp.MustCompile(`channel\s*=\s*"([^"]+)"`)
			if matches := re.FindStringSubmatch(string(data)); len(matches) > 1 {
				version = matches[1]
			}
		}
	}

	if _, err := os.Stat(filepath.Join(repoPath, "Cargo.lock")); err == nil {
		files = append(files, "Cargo.lock")
	}

	if len(files) == 0 {
		return nil
	}

	return &RuntimeInfo{
		Language: "rust",
		Version:  version,
		Files:    files,
	}
}
