package gitchglog

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"mcp-changelog-promoter/pkg/changelog"
)

// Adapter interacts with the git-chglog command-line tool.
type Adapter struct{}

// NewAdapter creates a new git-chglog adapter.
func NewAdapter() *Adapter {
	return &Adapter{}
}

// checkGitChglogExists 检查 git-chglog 命令是否存在于 PATH 中，并能在指定目录下成功执行
func checkGitChglogExists(repoPath string) error {
	var stderrBuf bytes.Buffer
	cmd := exec.Command("git-chglog", "--version")
	cmd.Dir = repoPath
	cmd.Stderr = &stderrBuf

	if err := cmd.Run(); err != nil {
		stderrStr := stderrBuf.String()
		// 检查是否是 "command not found" 类型的错误
		if strings.Contains(err.Error(), "executable file not found") || strings.Contains(err.Error(), "no such file or directory") {
			return fmt.Errorf("'git-chglog' command not found in PATH. Please install git-chglog: https://github.com/git-chglog/git-chglog")
		}
		// 其他执行错误，包含 stderr 以获取更多信息
		return fmt.Errorf("检查 'git-chglog --version' 在目录 '%s' 执行失败: %w\nStderr:\n%s", repoPath, err, stderrStr)
	}
	return nil
}

// GetVersionEntry retrieves the changelog content, code diff, and README for a specific version (tag).
func (a *Adapter) GetVersionEntry(repoPath string, version string) (*changelog.VersionEntry, error) {
	// 检查 git-chglog 是否存在
	if err := checkGitChglogExists(repoPath); err != nil {
		return nil, err
	}

	// 1. Find the previous tag to define the range for git-chglog
	prevTagCmd := exec.Command("git", "describe", "--tags", "--abbrev=0", version+"^")
	prevTagCmd.Dir = repoPath
	prevTagOutput, err := prevTagCmd.Output()
	var tagRange string
	var prevTag string
	if err != nil {
		// First tag
		tagRange = version
	} else {
		prevTag = strings.TrimSpace(string(prevTagOutput))
		tagRange = fmt.Sprintf("%s..%s", prevTag, version)
	}

	// 2. Run git-chglog for the determined range or tag
	var chglogOutput []byte
	var stderrBuf bytes.Buffer
	chglogCmd := exec.Command("git-chglog", tagRange)
	chglogCmd.Dir = repoPath
	chglogCmd.Stderr = &stderrBuf
	chglogOutput, err = chglogCmd.Output()
	if err != nil {
		stderrBuf.Reset()
		chglogCmd = exec.Command("git-chglog", "--output", version) // Fallback
		chglogCmd.Dir = repoPath
		chglogCmd.Stderr = &stderrBuf
		chglogOutput, err = chglogCmd.Output()
		if err != nil {
			// 即使 git-chglog 失败，我们仍然尝试获取其他信息
			fmt.Fprintf(os.Stderr, "Warning: git-chglog failed (range '%s', '--output %s'): %v\nStderr: %s\n", tagRange, version, err, stderrBuf.String())
			chglogOutput = []byte("无法生成 Changelog 内容。") // 提供一个默认值
		}
	}
	changelogDesc := string(chglogOutput)

	// 3. Get the tag date
	dateCmd := exec.Command("git", "show", "-s", "--format=%cI", version)
	dateCmd.Dir = repoPath
	dateOutput, err := dateCmd.Output()
	var dateStr string
	if err == nil {
		tagTime, parseErr := time.Parse(time.RFC3339, strings.TrimSpace(string(dateOutput)))
		if parseErr == nil {
			dateStr = tagTime.Format("2006-01-02")
		}
	}

	// 4. 获取项目名称
	projectName := filepath.Base(repoPath)

	// 5. 无条件获取代码变更和 README
	codeDiff := ""
	readmeContent := ""

	// 5.1 获取代码变更 (--stat 摘要)
	var diffCmd *exec.Cmd
	if prevTag != "" {
		// Remove --stat and --summary for full diff
		diffCmd = exec.Command("git", "diff", prevTag, version)
	} else {
		// Remove --stat and --summary for full diff
		diffCmd = exec.Command("git", "show", version)
	}
	diffCmd.Dir = repoPath
	diffOutput, diffErr := diffCmd.Output()
	if diffErr == nil {
		codeDiff = string(diffOutput)
		// 如果摘要过长，尝试获取更简洁的文件列表
		if len(codeDiff) > 5000 {
			var nameStatusCmd *exec.Cmd
			// Fallback to name-status is less useful with full diff, maybe remove?
			// For now, keep it but the primary diff is now full.
			// Consider removing this fallback logic later if full diff is always preferred.
			// var nameStatusCmd *exec.Cmd // Removed redeclaration
			if prevTag != "" {
				nameStatusCmd = exec.Command("git", "diff", "--name-status", prevTag, version) // Keep name-status as a potential shorter fallback if needed
			} else {
				nameStatusCmd = exec.Command("git", "show", "--name-status", version) // Keep name-status as a potential shorter fallback if needed
			}
			nameStatusCmd.Dir = repoPath
			nameStatusOutput, nsErr := nameStatusCmd.Output()
			if nsErr == nil {
				codeDiff = string(nameStatusOutput)
			}
			// 如果仍然太长，则截断
			if len(codeDiff) > 5000 {
				codeDiff = codeDiff[:5000] + "\n...(diff 截断)"
			}
		}
	} else {
		fmt.Fprintf(os.Stderr, "Warning: Failed to get code diff for %s: %v\n", version, diffErr)
		codeDiff = "无法获取代码变更。"
	}

	// 5.2 读取 README.md
	readmePath := filepath.Join(repoPath, "README.md")
	readmeData, readErr := os.ReadFile(readmePath)
	if readErr == nil {
		readmeContent = string(readmeData)
		// 如果 README 太长，截断
		if len(readmeContent) > 2000 {
			readmeLines := strings.Split(readmeContent, "\n")
			var truncatedReadme strings.Builder
			lineCount := 0
			for _, line := range readmeLines {
				trimmedLine := strings.TrimSpace(line)
				if trimmedLine == "" {
					continue
				}
				truncatedReadme.WriteString(trimmedLine)
				truncatedReadme.WriteString("\n")
				lineCount++
				if lineCount >= 30 { // Limit to ~30 non-empty lines
					break
				}
			}
			readmeContent = truncatedReadme.String() + "\n...(README 截断)"
		}
	} else {
		fmt.Fprintf(os.Stderr, "Warning: Failed to read README.md in %s: %v\n", repoPath, readErr)
		readmeContent = "无法读取 README 内容。"
	}

	// 6. Create the entry
	entry := &changelog.VersionEntry{
		ProjectName:   projectName,
		Version:       version,
		Date:          dateStr,
		Description:   changelogDesc,
		CodeDiff:      codeDiff,
		Readme:        readmeContent,
		SourceAdapter: "git-chglog", // Indicate the source
	}

	return entry, nil
}

// isOnlyCommitIDs is kept for potential future use or inspection, but not currently used for gating.
func isOnlyCommitIDs(content string) bool {
	lines := strings.Split(content, "\n")
	commitIDPattern := regexp.MustCompile(`^[a-f0-9]{7,40}$`)
	commitIDLines := 0
	nonEmptyLines := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		nonEmptyLines++
		if commitIDPattern.MatchString(line) {
			commitIDLines++
		}
	}
	return nonEmptyLines > 0 && float64(commitIDLines)/float64(nonEmptyLines) > 0.5
}

// GetLatestEntry gets the changelog entry for the latest tag.
func (a *Adapter) GetLatestEntry(repoPath string) (*changelog.VersionEntry, error) {
	// 检查 git-chglog 是否存在
	if err := checkGitChglogExists(repoPath); err != nil {
		return nil, err
	}
	latestTagCmd := exec.Command("git", "describe", "--tags", "--abbrev=0")
	latestTagCmd.Dir = repoPath
	latestTagOutput, err := latestTagCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("获取最新 tag 失败：%w", err)
	}
	latestTag := strings.TrimSpace(string(latestTagOutput))
	if latestTag == "" {
		return nil, fmt.Errorf("未能找到任何 tag")
	}
	return a.GetVersionEntry(repoPath, latestTag)
}
