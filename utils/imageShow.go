package utils

import "fmt"

// ImageShow returns a markdown image tag for the given URL.
// B6 fix: index parameter was incremented but the result was never used in the format string.
func ImageShow(index int, modelName, url string) string {
	_ = index // kept for API compatibility; not rendered in markdown image syntax
	return fmt.Sprintf("![%s](%s)", modelName, url)
}
