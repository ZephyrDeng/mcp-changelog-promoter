package releaseit

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testChangelogContent = `
# 更新日志

所有此项目的显著更改都将记录在此文件中。

格式基于 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.0.0/)，
并且此项目遵循 [Semantic Versioning](https://semver.org/lang/zh-CN/)。

## [1.0.1](https://github.com/ZephyrDeng/spreadsheet-mcp/compare/v1.0.0...v1.0.1) (2025-04-22)


### Bug Fixes

* remove JSON formatting for output in server tool ([a08afaa](https://github.com/ZephyrDeng/spreadsheet-mcp/commit/a08afaa5ed7962ee3f0e0db5747fcd9714ffee98))

# [1.0.0](https://github.com/ZephyrDeng/spreadsheet-mcp/compare/v0.1.2...v1.0.0) (2025-04-22)


### Bug Fixes

* Change output to JSON, fix tests, update docs ([5553045](https://github.com/ZephyrDeng/spreadsheet-mcp/commit/5553045ebe8cb0ca0aba7b5b75b53cb8e5a91116))


### Features

* **spreadsheet:** Change spreadsheet tool output format to JSON array ([20fe9a6](https://github.com/ZephyrDeng/spreadsheet-mcp/commit/20fe9a6ad97e1ffd11027edfe20a95aaa2a1d657))


### BREAKING CHANGES

* **spreadsheet:** The output format of view_spreadsheet, filter_spreadsheet, and sort_spreadsheet tools has changed from a Markdown string to a JSON string. Clients relying on the old format need to be updated.

## [0.1.2](https://github.com/ZephyrDeng/spreadsheet-mcp/compare/v0.1.1...v0.1.2) (2025-04-17)


### Bug Fixes

* Resolve TS errors and enhance spreadsheet cell parsing ([1197cbf](https://github.com/ZephyrDeng/spreadsheet-mcp/commit/1197cbf545b5d0c0a90e18e96f35a9f3dfe8d5b5))

## 0.1.1 (2025-04-16)


# 0.1.0 (2025-04-15)


### Features

* Allow view_spreadsheet to preview up to max rows ([8b93043](https://github.com/ZephyrDeng/spreadsheet-mcp/commit/8b93043a4a17b27272b9a13dc58e45ad77c31470))
* initialize project with TypeScript, core scripts, and spreadsheet utilities ([b167527](https://github.com/ZephyrDeng/spreadsheet-mcp/commit/b16752706c92b5cfd7b6d2b95693eda299103867))
* setup project for npm publishing with ci and release automation ([5f7f423](https://github.com/ZephyrDeng/spreadsheet-mcp/commit/5f7f4236b70d40071d7ee367ebc19f4bfeba2317))
`

// setupTestChangelog creates a temporary directory and CHANGELOG.md file for testing.
func setupTestChangelog(t *testing.T) (string, func()) {
	tmpDir, err := os.MkdirTemp("", "changelog_test")
	require.NoError(t, err, "Failed to create temp dir")

	changelogPath := filepath.Join(tmpDir, "CHANGELOG.md")
	err = os.WriteFile(changelogPath, []byte(testChangelogContent), 0644)
	require.NoError(t, err, "Failed to write temp changelog file")

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return changelogPath, cleanup
}

func TestParseSpecificVersionFromFile_Found(t *testing.T) {
	changelogPath, cleanup := setupTestChangelog(t)
	defer cleanup()

	version := "1.0.1"
	entry, err := parseSpecificVersionFromFile(changelogPath, version)

	require.NoError(t, err)
	require.NotNil(t, entry)

	assert.Equal(t, version, entry.Version)
	assert.Equal(t, "2025-04-22", entry.Date)
	assert.Contains(t, entry.Description, "Bug Fixes")
	assert.Contains(t, entry.Description, "remove JSON formatting")
	assert.NotContains(t, entry.Description, "## [1.0.1]") // Should not include the header itself
	assert.NotContains(t, entry.Description, "# [1.0.0]")  // Should not include next version's content
}

func TestParseSpecificVersionFromFile_FoundNoLinkDate(t *testing.T) {
	changelogPath, cleanup := setupTestChangelog(t)
	defer cleanup()

	version := "0.1.1"
	entry, err := parseSpecificVersionFromFile(changelogPath, version)

	require.NoError(t, err)
	require.NotNil(t, entry)

	assert.Equal(t, version, entry.Version)
	assert.Equal(t, "2025-04-16", entry.Date)
	assert.Equal(t, "", entry.Description) // Expect empty description as there's nothing between 0.1.1 and 0.1.0 headers
}

func TestParseSpecificVersionFromFile_NotFound(t *testing.T) {
	changelogPath, cleanup := setupTestChangelog(t)
	defer cleanup()

	version := "9.9.9"
	entry, err := parseSpecificVersionFromFile(changelogPath, version)

	require.Error(t, err)
	assert.Nil(t, entry)
	assert.Contains(t, err.Error(), "未在")
	assert.Contains(t, err.Error(), "中找到版本 9.9.9")
}

func TestFindLatestVersionInFile(t *testing.T) {
	changelogPath, cleanup := setupTestChangelog(t)
	defer cleanup()

	latestVersion, err := findLatestVersionInFile(changelogPath)

	require.NoError(t, err)
	assert.Equal(t, "1.0.1", latestVersion) // Should be the first version found
}
