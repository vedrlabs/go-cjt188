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

// ForSend 发送时取当前序列号，并自增刷新（模 256）
func (s *SER) ForSend() byte {
	s.lock.Lock()
	defer s.lock.Unlock()
	v := s.value
	s.value++
	return v
}

// OnReceive 接收时用对端序列号刷新本地
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
