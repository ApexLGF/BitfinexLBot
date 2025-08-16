package bitfinex

import (
	"strconv"
	"sync"
	"time"
)

// NonceManager 统一的nonce管理器，确保所有API调用使用严格递增的nonce
type NonceManager struct {
	mutex     sync.Mutex
	lastNonce int64
}

// NewNonceManager 创建新的nonce管理器
func NewNonceManager() *NonceManager {
	// 使用当前时间戳的微秒值乘以1000作为起始值，确保足够大的间隔
	now := time.Now().UnixNano() / 1000 * 1000
	return &NonceManager{
		lastNonce: now,
	}
}

// GetNextNonce 获取下一个nonce值，确保严格递增
func (nm *NonceManager) GetNextNonce() string {
	nm.mutex.Lock()
	defer nm.mutex.Unlock()

	// 使用微秒时间戳乘以1000作为基础（根据Bitfinex社区建议）
	now := time.Now().UnixNano() / 1000 * 1000

	// 确保nonce严格递增，如果当前时间戳不大于上次的nonce，则在上次基础上增加10000
	// 使用更大的间隔以避免高并发时的网络延迟导致的顺序问题
	if now <= nm.lastNonce {
		now = nm.lastNonce + 10000
	}

	nm.lastNonce = now
	return strconv.FormatInt(now, 10)
}

// GetCurrentNonce 获取当前nonce值（用于调试）
func (nm *NonceManager) GetCurrentNonce() int64 {
	nm.mutex.Lock()
	defer nm.mutex.Unlock()
	return nm.lastNonce
}