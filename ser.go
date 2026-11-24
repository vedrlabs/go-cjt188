package go_cjt188

import "sync"

// SER 序列号，占用 1 字节，需在发送/接收时刷新
type SER struct {
	value byte
	lock  sync.Mutex
}

// NewSER 创建序列号对象，初始值为 0
func NewSER() *SER {
	return &SER{}
}

// Value 返回当前序列号
func (s *SER) Value() byte {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.value
}

// ForSend 发送前按模 256 加 1 后产生序列号（标准：每次通讯前 +1）
// 初始为 0 时，第一次发送得到 1；OnReceive 同步为应答 SER 后，下次发送会再 +1，不会重复
func (s *SER) ForSend() byte {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.value++
	return s.value
}

// OnReceive 接收时用对端序列号刷新本地（与请求 SER 一致）；下次 ForSend 会在此基础上 +1
func (s *SER) OnReceive(v byte) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.value = v
}

// repeatByte 将字节 b 重复 n 次（用于 CBC IV 中 SER×8）
func repeatByte(b byte, n int) []byte {
	out := make([]byte, n)
	for i := range out {
		out[i] = b
	}
	return out
}
