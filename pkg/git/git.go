package git

import (
	"os"
	"os/exec"
	"path/filepath"
)

// Git 处理 Git 相关的操作
type Git struct {
	repoPath string
}

// New 创建一个新的 Git 处理器
func New(repoPath string) *Git {
	return &Git{
		repoPath: repoPath,
	}
}

// GetCommitDiff 获取指定 commit 的变更内容
func (g *Git) GetCommitDiff(commitID string) (string, error) {
	cmd := exec.Command("git", "show", commitID)
	cmd.Dir = g.repoPath
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

// ReadReadme 读取仓库的 README.md 文件
func (g *Git) ReadReadme() (string, error) {
	readmePath := filepath.Join(g.repoPath, "README.md")
	content, err := os.ReadFile(readmePath)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// ValidateRepo 验证给定路径是否是有效的 Git 仓库
func (g *Git) ValidateRepo() error {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	cmd.Dir = g.repoPath
	return cmd.Run()
}
