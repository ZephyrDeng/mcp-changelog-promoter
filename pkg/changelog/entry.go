package changelog

// VersionEntry 表示从任何来源获取的通用版本信息
type VersionEntry struct {
	ProjectName   string // 项目名称
	Version       string
	Date          string // YYYY-MM-DD 格式，可能为空
	Description   string // 与该版本相关的描述性内容 (通常是 Markdown)
	CodeDiff      string // 代码变更差异（当 Description 不够详细时）
	Readme        string // 项目 README 内容（当 Description 不够详细时）
	SourceAdapter string // 生成此条目的适配器名称 (例如 "release-it", "git-chglog")
}
