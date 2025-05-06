package main

import (
	"context"
	"fmt"
	"log"
	"path/filepath"

	"mcp-changelog-promoter/pkg/changelog"
	"mcp-changelog-promoter/pkg/changelog/gitchglog"
	"mcp-changelog-promoter/pkg/changelog/releaseit"
	"mcp-changelog-promoter/pkg/promoter"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	// 创建 MCP 服务器
	s := server.NewMCPServer(
		"Changelog Promoter",
		"1.0.0",
	)

	// 定义通用参数
	// 注意：mcp.WithString 返回的是 ToolOption
	commonParams := []mcp.ToolOption{
		mcp.WithString("repo_path",
			mcp.Required(),
			mcp.Description("本地 Git 仓库的路径"),
		),
		mcp.WithString("version",
			mcp.Description("指定要处理的版本（Git 标签），默认处理最新版本"),
		),
	}

	// changelog_promoter_gitchglog 工具
	// 将通用参数直接展开传入 NewTool
	gitChglogToolOptions := []mcp.ToolOption{
		mcp.WithDescription("将 git-chglog 生成的 Changelog 转换为宣传文案"),
	}
	gitChglogToolOptions = append(gitChglogToolOptions, commonParams...)
	gitChglogTool := mcp.NewTool("changelog_promoter_gitchglog", gitChglogToolOptions...)
	s.AddTool(gitChglogTool, handleGitChglog)

	// changelog_promoter_releaseit 工具
	// 将通用参数直接展开传入 NewTool
	releaseItToolOptions := []mcp.ToolOption{
		mcp.WithDescription("将 release-it (conventional-changelog) 生成的 CHANGELOG.md 转换为宣传文案"),
	}
	releaseItToolOptions = append(releaseItToolOptions, commonParams...)
	releaseItTool := mcp.NewTool("changelog_promoter_releaseit", releaseItToolOptions...)
	s.AddTool(releaseItTool, handleReleaseIt)

	// 启动 stdio 服务器
	if err := server.ServeStdio(s); err != nil {
		log.Fatalf("服务器错误: %v\n", err)
	}
}

// handleGitChglog 处理 git-chglog 工具请求
func handleGitChglog(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	repoPath, version, err := parseCommonParams(request)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	adapter := gitchglog.NewAdapter()
	var entry *changelog.VersionEntry

	if version == "" {
		entry, err = adapter.GetLatestEntry(repoPath)
	} else {
		entry, err = adapter.GetVersionEntry(repoPath, version)
	}

	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("获取 changelog 失败 (git-chglog): %v", err)), nil
	}

	return createAndFormatResult(entry)
}

// handleReleaseIt 处理 release-it 工具请求
func handleReleaseIt(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	repoPath, version, err := parseCommonParams(request)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	adapter := releaseit.NewAdapter()
	var entry *changelog.VersionEntry

	if version == "" {
		entry, err = adapter.GetLatestEntry(repoPath)
	} else {
		entry, err = adapter.GetVersionEntry(repoPath, version)
	}

	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("获取 changelog 失败 (release-it/CHANGELOG.md): %v", err)), nil
	}

	return createAndFormatResult(entry)
}

// parseCommonParams 解析通用的 repo_path 和 version 参数
func parseCommonParams(request mcp.CallToolRequest) (repoPath string, version string, err error) {
	rp, ok := request.Params.Arguments["repo_path"].(string)
	if !ok {
		err = fmt.Errorf("repo_path 必须是字符串")
		return
	}
	// 获取绝对路径
	repoPath, err = filepath.Abs(rp)
	if err != nil {
		err = fmt.Errorf("无法获取仓库绝对路径：%w", err)
		return
	}

	version, _ = request.Params.Arguments["version"].(string)
	return
}

// createAndFormatResult 调用 promoter 并格式化结果
func createAndFormatResult(entry *changelog.VersionEntry) (*mcp.CallToolResult, error) {
	// 创建宣传任务（包含提示词和上下文）
	task, err := promoter.CreatePromotionTask(entry)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("创建宣传任务失败: %v", err)), nil
	}

	// 构建包含多个具体 Content 类型的切片
	var finalContentSlice []mcp.Content // 切片类型是接口类型

	// 添加 Prompt
	finalContentSlice = append(finalContentSlice, mcp.TextContent{
		Type: "text", // 必须是 "text"
		Text: task.Prompt,
	})

	// 添加 Changelog Content
	if content, ok := task.Context["description"]; ok && content != "" {
		finalContentSlice = append(finalContentSlice, mcp.TextContent{
			Type: "text", // 必须是 "text"
			Text: content,
		})
	}
	// 添加 Code Diff
	if content, ok := task.Context["code_diff"]; ok && content != "" {
		finalContentSlice = append(finalContentSlice, mcp.TextContent{
			Type: "text",  // 必须是 "text"
			Text: content, // 内容本身包含 diff 格式
		})
	}
	// 添加 Readme Content
	if content, ok := task.Context["readme"]; ok && content != "" {
		finalContentSlice = append(finalContentSlice, mcp.TextContent{
			Type: "text",  // 必须是 "text"
			Text: content, // 内容本身是 Markdown
		})
	}

	// 检查是否至少有一个 Content Block
	if len(finalContentSlice) == 0 {
		return mcp.NewToolResultError("未能生成任何内容块"), nil
	}

	// 构建并返回 CallToolResult
	return &mcp.CallToolResult{
		Content: finalContentSlice,
	}, nil
}
