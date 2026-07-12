package utils

import (
	"fmt"
	"pplx2api/config"
)

func searchShowDetails(index int, title, url, snippet string) string {
	return fmt.Sprintf("<details>\n<summary>[%d] %s</summary>\n\n%s\n\n[Link](%s)\n\n</details>", index, title, snippet, url)
}

func searchShowCompatible(index int, title, url, snippet string) string {
	return fmt.Sprintf("[%d] [%s](%s):\n%s\n", index, title, url, snippet)
}

// SearchShow 格式化单条搜索结果。
// #7 fix: 快照 SearchResultCompatible 加 RLock，避免与 Reload 并发时竪态。
func SearchShow(index int, title, url, snippet string) string {
	index++
	if len([]rune(snippet)) > 150 {
		runeSnippet := []rune(snippet)
		snippet = fmt.Sprintf("%s ……", string(runeSnippet[:150]))
	}
	config.ConfigInstance.RwMutex.RLock()
	compatible := config.ConfigInstance.SearchResultCompatible
	config.ConfigInstance.RwMutex.RUnlock()
	if compatible {
		return searchShowCompatible(index, title, url, snippet)
	}
	return searchShowDetails(index, title, url, snippet)
}
