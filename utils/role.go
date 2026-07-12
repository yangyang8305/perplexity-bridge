package utils

import (
	"pplx2api/config"
)

// GetRolePrefix 返回对应角色的提示前缀。
// #7 fix: 快照 NoRolePrefix 加 RLock，避免与 Reload 并发时竪态。
func GetRolePrefix(role string) string {
	config.ConfigInstance.RwMutex.RLock()
	noPrefix := config.ConfigInstance.NoRolePrefix
	config.ConfigInstance.RwMutex.RUnlock()
	if noPrefix {
		return ""
	}
	switch role {
	case "system":
		return "System: "
	case "user":
		return "Human: "
	case "assistant":
		return "Assistant: "
	default:
		return "Unknown: "
	}
}
