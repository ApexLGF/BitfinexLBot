package bitfinex

import (
	"strconv"
	"sync/atomic"
	"time"
)

// CustomNonceGenerator 自定义nonce生成器，实现Bitfinex SDK的NonceGenerator接口
type CustomNonceGenerator struct {
	nonce uint64
	Step  uint64 // 递增步长，用于避免并发冲突（公开字段）
}

// GetNonce 获取下一个nonce值，使用原子操作确保线程安全
// 这个方法实现了Bitfinex SDK的NonceGenerator接口
func (g *CustomNonceGenerator) GetNonce() string {
	return strconv.FormatUint(atomic.AddUint64(&g.nonce, g.Step), 10)
}

// GetNonceUint64 获取下一个nonce值作为uint64
func (g *CustomNonceGenerator) GetNonceUint64() uint64 {
	return atomic.AddUint64(&g.nonce, g.Step)
}

// GetCurrentNonceValue 获取当前nonce值（不递增，用于调试）
func (g *CustomNonceGenerator) GetCurrentNonceValue() uint64 {
	return atomic.LoadUint64(&g.nonce)
}

// NewCustomNonceGenerator 创建新的自定义nonce生成器（默认步长10）
func NewCustomNonceGenerator() *CustomNonceGenerator {
	return NewCustomNonceGeneratorWithStep(10)
}

// NewCustomNonceGeneratorWithStep 创建自定义步长的nonce生成器
func NewCustomNonceGeneratorWithStep(step uint64) *CustomNonceGenerator {
	// 修复：使用正确的微秒级时间戳计算方式
	// time.Now().UnixNano() / 1000 = 微秒级时间戳
	initialNonce := uint64(time.Now().UnixNano()) / 1000
	
	if step == 0 {
		step = 1 // 确保步长至少为1
	}
	
	return &CustomNonceGenerator{
		nonce: initialNonce,
		Step:  step,
	}
}

// 全局nonce生成器实例
var globalNonce *CustomNonceGenerator

func init() {
	// 使用更大的步长（500）来避免高频API调用时的nonce冲突
	// 更大的步长可以确保即使在并发场景下也能避免nonce冲突
	globalNonce = NewCustomNonceGeneratorWithStep(500)
}

// GetGlobalNonce 获取全局nonce（字符串格式）
func GetGlobalNonce() string {
	return globalNonce.GetNonce()
}

// GetGlobalNonceUint64 获取全局nonce（uint64格式）
func GetGlobalNonceUint64() uint64 {
	return globalNonce.GetNonceUint64()
}

// GetGlobalNonceGenerator 获取全局nonce生成器实例
func GetGlobalNonceGenerator() *CustomNonceGenerator {
	return globalNonce
}