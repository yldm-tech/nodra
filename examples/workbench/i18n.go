package main

const (
	languageEnglish = "en"
	languageChinese = "zh-Hans"
)

var languageOptions = []string{languageEnglish, languageChinese}

type translations struct {
	title             string
	brand             string
	tagline           string
	preferences       string
	folder            string
	folderPlaceholder string
	includeHidden     string
	changesApply      string
	saveSettings      string
	cancelEdits       string
	saving            string
	saveFailed        string
	saved             string
	searchPlaceholder string
	search            string
	cancelSearch      string
	searching         string
	matches           string
	truncated         string
	resultsFor        string
	typeToBegin       string
	language          string
}

func normalizeLanguage(language string) string {
	if language == languageChinese {
		return languageChinese
	}
	return languageEnglish
}

func translationsFor(language string) translations {
	if normalizeLanguage(language) == languageChinese {
		return translations{
			title:             "工作区搜索",
			brand:             "Nodra + MyGo",
			tagline:           "搜索本地文件，将草稿与已保存的设置分开。",
			preferences:       "搜索设置",
			folder:            "文件夹",
			folderPlaceholder: "文件夹路径",
			includeHidden:     "包含隐藏文件",
			changesApply:      "成功保存后应用更改。",
			saveSettings:      "保存设置",
			cancelEdits:       "取消编辑",
			saving:            "正在保存…",
			saveFailed:        "保存失败：",
			saved:             "设置已保存",
			searchPlaceholder: "搜索文件名…",
			search:            "搜索",
			cancelSearch:      "取消搜索",
			searching:         "正在搜索…",
			matches:           "%d 个匹配 · 已检查 %d 个文件",
			truncated:         "结果受数量限制，请缩小搜索词或文件夹范围。",
			resultsFor:        "搜索结果：%q",
			typeToBegin:       "输入文件名开始搜索。",
			language:          "语言",
		}
	}
	return translations{
		title:             "Workspace search",
		brand:             "Nodra + MyGo",
		tagline:           "Find local files. Keep drafts separate from saved preferences.",
		preferences:       "Search preferences",
		folder:            "Folder",
		folderPlaceholder: "Folder path",
		includeHidden:     "Include hidden files",
		changesApply:      "Changes apply after a successful save.",
		saveSettings:      "Save settings",
		cancelEdits:       "Cancel edits",
		saving:            "Saving settings…",
		saveFailed:        "Save failed: ",
		saved:             "Settings saved",
		searchPlaceholder: "Search file names…",
		search:            "Search",
		cancelSearch:      "Cancel search",
		searching:         "Searching…",
		matches:           "%d matches · %d files checked",
		truncated:         "Showing a bounded result set. Narrow the query or folder.",
		resultsFor:        "Results for %q",
		typeToBegin:       "Type a file name to begin.",
		language:          "Language",
	}
}
