package promoter

import (
	"fmt"

	"github.com/ZephyrDeng/mcp-changelog-promoter/pkg/changelog"
)

// PromotionTask 包含生成宣传文案所需的提示和上下文
type PromotionTask struct {
	Prompt  string            `json:"prompt"`
	Context map[string]string `json:"context"`
}

// CreatePromotionTask 根据 Changelog 版本条目创建宣传文案生成任务
func CreatePromotionTask(entry *changelog.VersionEntry) (*PromotionTask, error) {
	if entry == nil {
		return nil, fmt.Errorf("输入的版本条目为 nil")
	}

	// --- 根据适配器来源动态生成 Changelog 描述 ---
	var changelogSourceDescription string
	switch entry.SourceAdapter {
	case "git-chglog":
		changelogSourceDescription = "由 **git-chglog** 生成，通常包含该版本范围内的详细提交信息列表。你需要从中提炼关键的特性、修复和重大变更。"
	case "release-it":
		changelogSourceDescription = "由 **release-it (conventional-changelog)** 生成，通常已按类型（如 Features, Bug Fixes, BREAKING CHANGES）分类。请优先利用此结构。"
	default: // 未知或其他来源
		changelogSourceDescription = "来源未知或未指定。请仔细分析其内容结构。"
	}

	// --- 更新后的提示词 ---
	prompt := fmt.Sprintf(`
你是开源项目 {ProjectName: %s} 的技术营销专家。请为版本 v{%s}(发布日期: %s) 撰写一篇引人入胜的发布公告。目标受众是开发者。

## 可用信息源：
1.  **Changelog 内容 (来源: %s)**: %s
2.  **代码变更摘要 (Code Diff)**: 显示该版本与上一版本之间的代码统计变化（文件增删改）。
3.  **项目 README (截断)**: 提供项目整体概览和目标。

## 宣传文案要求：
1.  **整合信息**: 综合利用以上所有信息源。**请特别注意 Changelog 的来源 (%s)** 并据此调整解读策略。优先参考 Changelog，但如果其内容过于简单或与代码变更摘要明显不符，请侧重根据代码变更来推断核心功能和改进。使用 README 理解项目背景。
2.  **价值导向**: 将技术变更转化为清晰的用户价值和实际收益。解释新功能"为什么"重要，而不仅仅是"是什么"。
3.  **结构清晰**: 遵循下方模板，突出核心亮点。
4.  **风格专业热情**: 语气积极，适合开发者社区，可适度使用 Emoji。
5.  **篇幅控制**: 总字数控制在 300 字以内。

## 宣传稿模板（请按照此结构填写）：
"""
# 🎉 {ProjectName: %s} v{%s} 现已发布！

[开头段落：用 1-2 句话简要介绍此版本的主题或最重要的改进，激发读者兴趣。]

## ✨ 本次更新亮点：

*   **[核心功能/改进 1 + Emoji]**: [详细描述功能/改进，说明其解决的问题、带来的价值以及用户如何受益。可结合 Changelog (来源: %s) 和 Code Diff 信息。]
*   **[核心功能/改进 2 + Emoji]**: [同上，介绍第二个重要变更。]
*   **(可选)[核心功能/改进 3 + Emoji]**: [如果还有其他重要亮点，可在此添加。]

## 🛠️ 其他改进与修复：

[简要列出 2-4 个其他值得提及的优化、修复或次要功能。]

## 🚀 即刻体验！

我们强烈推荐所有用户升级到 v{%s}，体验这些强大的新功能和改进。

👉 [文档链接 或 GitHub Release 链接]
👉 [项目仓库链接]

感谢社区的支持！期待您的反馈！
"""

请根据以上模板和要求，基于下方提供的 Changelog、代码变更摘要和 README 内容，创建一份专业、吸引人且突出价值的宣传公告。填充模板时，请替换方括号中的内容，并确保最终文案流畅自然。

提供的上下文信息：
`, entry.ProjectName, entry.Version, entry.Date, // Header info
		entry.SourceAdapter, changelogSourceDescription, // Dynamic Changelog source info
		entry.SourceAdapter,              // Reminder in requirements
		entry.ProjectName, entry.Version, // Template Title
		entry.SourceAdapter, // Reminder in template highlights
		entry.Version)       // Template Call to action

	// --- 准备上下文映射 ---
	context := map[string]string{
		"description":    entry.Description,
		"code_diff":      entry.CodeDiff,
		"readme":         entry.Readme,
		"source_adapter": entry.SourceAdapter, // Add source adapter to context
	}

	return &PromotionTask{
		Prompt:  prompt,
		Context: context,
	}, nil
}
