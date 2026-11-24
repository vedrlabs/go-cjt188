package go_cjt188

import (
	"fmt"
	"math"
	"strconv"
	"time"
)

// 常用单位代号（CJ/T 188）
const (
	UnitJ      byte = 0x01 // J
	UnitWh     byte = 0x02 // Wh
	UnitKWh    byte = 0x05 // kWh
	UnitMWh    byte = 0x08 // MWh
	UnitMWh100 byte = 0x0A // MWh×100
	UnitKJ     byte = 0x0B // kJ
	UnitMJ     byte = 0x0E // MJ
	UnitGJ     byte = 0x11 // GJ
	UnitGJ100  byte = 0x13 // GJ×100
	UnitW      byte = 0x14 // W
	UnitKW     byte = 0x17 // kW
	UnitMW     byte = 0x1A // MW
	UnitL      byte = 0x29 // L
	UnitM3     byte = 0x2C // m³
	UnitLH     byte = 0x32 // L/h
	UnitM3H    byte = 0x35 // m³/h
)

// EncodeWriteData 将写数据项编码为 payload 片段（数值 BCD 低字节在前；hasUnit 时末尾追加单位代号）
// length 为数值占用字节数（不含单位）；浮点数默认保留 2 位小数（对应 XXXXXX.XX）
func EncodeWriteData(value any, length int, hasUnit bool, unitCode byte) ([]byte, error) {
	return EncodeWriteDataWithDecimals(value, length, 2, hasUnit, unitCode)
}

// EncodeWriteDataWithDecimals 同 EncodeWriteData，可指定小数位数（如 XXXX.XXXX 传 decimals=4）
func EncodeWriteDataWithDecimals(value any, length, decimals int, hasUnit bool, unitCode byte) ([]byte, error) {
	if length <= 0 {
		return nil, fmt.Errorf("invalid length: %d", length)
	}
	raw, err := anyToBCD(value, length, decimals)
	if err != nil {
		return nil, err
	}
	if !hasUnit {
		return raw, nil
	}
	out := make([]byte, 0, length+1)
	out = append(out, raw...)
	out = append(out, unitCode)
	return out, nil
}

// JoinWritePayload 拼接多个写数据片段，得到 Write 的 payload
func JoinWritePayload(parts ...[]byte) []byte {
	n := 0
	for _, p := range parts {
		n += len(p)
	}
	out := make([]byte, 0, n)
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}

// anyToBCD 将任意值转为 length 字节 BCD（低字节在前）
func anyToBCD(value any, length, decimals int) ([]byte, error) {
	switch v := value.(type) {
	case []byte:
		if len(v) != length {
			return nil, fmt.Errorf("[]byte length mismatch: want %d got %d", length, len(v))
		}
		out := make([]byte, length)
		copy(out, v)
		return out, nil
	case byte:
		if length != 1 {
			return nil, fmt.Errorf("byte value requires length=1, got %d", length)
		}
		return []byte{v}, nil
	case int:
		return intToBCD(int64(v), length, 0)
	case int32:
		return intToBCD(int64(v), length, 0)
	case int64:
		return intToBCD(v, length, 0)
	case uint:
		return intToBCD(int64(v), length, 0)
	case uint32:
		return intToBCD(int64(v), length, 0)
	case uint64:
		return intToBCD(int64(v), length, 0)
	case float32:
		return floatToBCD(float64(v), length, decimals)
	case float64:
		return floatToBCD(v, length, decimals)
	case string:
		return stringToBCD(v, length, decimals)
	case time.Time:
		if length != 7 {
			return nil, fmt.Errorf("time.Time requires length=7, got %d", length)
		}
		// YYYYMMDDhhmmss，低字节在前：ss mm hh DD MM YY YY
		yy := v.Year() % 100
		yyyy := v.Year()
		return []byte{
			binToBCD(v.Second()),
			binToBCD(v.Minute()),
			binToBCD(v.Hour()),
			binToBCD(v.Day()),
			binToBCD(int(v.Month())),
			binToBCD(yy),
			binToBCD(yyyy / 100),
		}, nil
	default:
		return nil, fmt.Errorf("unsupported value type %T", value)
	}
}

func floatToBCD(v float64, length, decimals int) ([]byte, error) {
	if decimals < 0 {
		return nil, fmt.Errorf("invalid decimals: %d", decimals)
	}
	scale := math.Pow10(decimals)
	n := int64(math.Round(v * scale))
	return intToBCD(n, length, decimals)
}

func stringToBCD(s string, length, decimals int) ([]byte, error) {
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return floatToBCD(f, length, decimals)
	}
	if n, err := strconv.ParseInt(s, 10, 64); err == nil {
		return intToBCD(n, length, 0)
	}
	return nil, fmt.Errorf("invalid numeric string: %q", s)
}

// intToBCD 将整数编为 length 字节 BCD，低字节在前（已按小数位放大的整数）
func intToBCD(n int64, length, _ int) ([]byte, error) {
	neg := n < 0
	if neg {
		n = -n
	}
	maxDigits := length * 2
	out := make([]byte, length)
	for i := 0; i < length; i++ {
		lo := int(n % 10)
		n /= 10
		hi := int(n % 10)
		n /= 10
		out[i] = byte(hi<<4 | lo)
	}
	if n != 0 {
		return nil, fmt.Errorf("value exceeds %d BCD digits", maxDigits)
	}
	// 部分水表用最高半字节高位表示符号；此处保持原值，负值由调用方按厂商约定处理
	_ = neg
	return out, nil
}
