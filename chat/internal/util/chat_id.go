package util

import (
	"crypto/md5"
	"fmt"
	"sort"
)

// GetPrivateConversationID 生成私聊会话的确定性 ID
func GetPrivateConversationID(user1, user2 string) string {
	// 1. 排序，保证 A+B 和 B+A 生成同一个 ID
	ids := []string{user1, user2}
	sort.Strings(ids)

	// 2. 拼接并 Hash
	raw := ids[0] + ":" + ids[1]
	hash := md5.Sum([]byte(raw))
	return fmt.Sprintf("%x", hash)
}
