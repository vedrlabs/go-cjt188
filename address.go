package go_cjt188

import (
	"encoding/hex"
	"fmt"
)

func buildAddress() *Address {
	return &Address{}
}

// Address 地址域 A0~A6
// 由 7 字节组成，每字节为 2 位 BCD 码，地址长度为 14 位十进制数；
// 低地址在前，高地址在后；A5、A6 为厂商代码。
type Address struct {
	addr             [5]byte // 表地址（A0~A4）
	manufacturerCode [2]byte // 厂家编码（A5~A6）
}

// TheoreticalLength 理论长度
func (a *Address) TheoreticalLength() byte {
	return 7
}

// decode 解码
func (a *Address) decode(data []byte) error {
	copy(a.addr[:], data[0:5])
	copy(a.manufacturerCode[:], data[5:7])
	return nil
}

// decodeHex 用正序十六进制表号、厂家编码解码（内部反转为线序低地址在前）
// addrHex 须解出 5 字节，manufacturerHex 须解出 2 字节。
func (a *Address) decodeHex(addrHex, manufacturerHex string) error {
	addr, err := hex.DecodeString(addrHex)
	if err != nil {
		return fmt.Errorf("meter address hex decode failed: %w", err)
	}
	if len(addr) != 5 {
		return fmt.Errorf("meter address length error: need 5 bytes, got %d", len(addr))
	}
	mfr, err := hex.DecodeString(manufacturerHex)
	if err != nil {
		return fmt.Errorf("manufacturer code hex decode failed: %w", err)
	}
	if len(mfr) != 2 {
		return fmt.Errorf("manufacturer code length error: need 2 bytes, got %d", len(mfr))
	}
	copy(a.addr[:], a.reverse(addr))
	copy(a.manufacturerCode[:], a.reverse(mfr))
	return nil
}

// AddrString 返回表地址十进制字符串（高位在前）
func (a *Address) AddrString() string {
	return hex.EncodeToString(a.reverse(a.addr[:]))
}

// ManufacturerCodeString 返回厂家编码十进制字符串（高位在前）
func (a *Address) ManufacturerCodeString() string {
	return hex.EncodeToString(a.reverse(a.manufacturerCode[:]))
}

// reverse 反转字节数组
func (a *Address) reverse(data []byte) []byte {
	reversed := make([]byte, len(data))
	for i := range data {
		reversed[i] = data[len(data)-1-i]
	}
	return reversed
}

// bytes 返回完整地址域 A0~A6
func (a *Address) bytes() []byte {
	out := make([]byte, 7)
	copy(out[0:5], a.addr[:])
	copy(out[5:7], a.manufacturerCode[:])
	return out
}

// IsAAHBroadcast 当地址为全AAH或部分为AAH时，按AAH定义的字节广播，从站如应答应返回实际地址
func (a *Address) IsAAHBroadcast() bool {
	for _, b := range a.bytes() {
		if b == 0xAA {
			return true
		}
	}
	return false
}

// Is99HBroadcast 当地址为全99H时为广播地址，只针对特殊命令有效，不要求从站应答
func (a *Address) Is99HBroadcast() bool {
	for _, b := range a.bytes() {
		if b != 0x99 {
			return false
		}
	}
	return true
}
