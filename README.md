# go-cjt188

CJ/T 188-2018《户用计量仪表数据传输技术条件》Go 实现。  
Go implementation of **CJ/T 188-2018** (Technical requirements of utility meters data transmission).

- 中文说明见下方 [中文](#中文)
- English docs: [English](#english)

---

## 中文

### 简介

本库用于解析与组帧 CJ/T 188-2018 报文，覆盖：

- 帧编解码（前导字 / `68H` … `CS` / `16H`）
- 仪表类型、地址域、控制码
- 读 / 写数据操作
- SM4-CBC 密文传输（可选）
- 写数据 payload 拼接工具（BCD + 单位代号）


### 设计说明

建议保留**一个长期 Frame** 做接收解析，发送时用 `Copy()` 出一份：

| 共享字段 | 每次独立字段 |
|---------|-------------|
| `guide` / `ser` / `key` | `address` / `control` / `data` / `meterType` |

`SER` 在会话内共享：发送 `ForSend()` 自增，接收结果方法里 `OnReceive()` 刷新。

### 快速开始：接收

```go
frame := cjt188.BuildCJT1882018Frame([]byte{0xFE, 0xFE}, cjt188.MeterTypeColdWater)
_ = frame.BindDecryptKey(key) // 密文时需要 16 字节 SM4 密钥

err := frame.Decode(bufio.NewReader(conn))
if err != nil {
    // handle
}

mt := frame.MeterType()
addr := frame.Address()
ctrl := frame.Control()

if frame.IsReadData() && frame.IsNormalResponse() {
    di0, di1, data, ok := frame.ReadResult(true) // true=去掉单位代号
    _, _, _ = di0, di1, data
    _ = ok
}

if frame.IsWriteData() && frame.IsNormalResponse() {
    di0, di1, data, ok := frame.WriteResult(false)
    _, _, _, _ = di0, di1, data, ok
}

if frame.IsAbnormalResponse() {
    ser, st, ok := frame.AbnormalResult()
    _, _, _ = ser, st, ok
}
```

### 快速开始：发送读

```go
// 先保证 frame 上已有地址（通常来自上次 Decode，或自行绑定）
req, err := frame.Copy().
    BindMeterType(cjt188.MeterTypeColdWater).
    Read(0x90, 0x1F, false) // DI0 DI1，encrypt=false 明文
_ = err
_ = req // 完整报文：前导字 + 68H ... 16H
```

### 快速开始：发送写

```go
item, err := cjt188.EncodeWriteData(3.96, 4, true, cjt188.UnitM3) // XXXXXX.XX + 单位
if err != nil {
    // handle
}
valve, _ := cjt188.EncodeWriteData(byte(0x55), 1, false, 0) // 开阀 55H
payload := cjt188.JoinWritePayload(item, valve)

req, err := frame.Copy().Write(0xA0, 0x17, payload, false)
_ = err
_ = req
```

`XXXX.XXXX` 格式可用：

```go
cjt188.EncodeWriteDataWithDecimals(7.1180, 4, 4, true, cjt188.UnitM3H)
```

### 密文传输

1. `BindDecryptKey(16字节密钥)`
2. 读/写时 `encrypt=true`（控制码 D3=1）
3. 算法：SM4-CBC；IV = `T + A0~A6 + SER×8`
4. 仅加密 DATA 中去掉 DI、SER 后的部分（前置 6 字节时间戳 + PKCS7）

### 主要 API

| API | 说明 |
|-----|------|
| `BuildCJT1882018Frame` | 创建会话 Frame |
| `Copy` | 复制发送用 Frame（共享 ser/key/guide） |
| `Decode` / `Encode` | 解析 / 组帧 |
| `Read` / `Write` | 生成读/写请求报文 |
| `ReadResult` / `WriteResult` | 正常应答取 DI 与业务数据 |
| `AbnormalResult` | 异常应答取 SER 与状态 ST |
| `EncodeWriteData` / `JoinWritePayload` | 拼写数据 payload |
| `MeterType` / `Address` / `Control` | 获取字段 |

### 依赖

- Go 1.21+
- `github.com/tjfoc/gmsm`（SM4）

---

## English

### Introduction

This library encodes and decodes **CJ/T 188-2018** frames for utility meters (water / heat / gas), including:

- Frame codec (preamble / `0x68` … checksum / `0x16`)
- Meter type, address field, control code
- Read / write operations
- Optional SM4-CBC ciphertext mode
- Helpers to build write payloads (BCD + unit code)

> Use the actual module path from `go.mod` of this repo.

### Design

Keep **one long-lived Frame** for receiving. For sending, call `Copy()`:

| Shared | Per request |
|--------|-------------|
| `guide` / `ser` / `key` | `address` / `control` / `data` / `meterType` |

`SER` is shared in the session: `ForSend()` increments on transmit; result helpers call `OnReceive()` on success.

### Receive

```go
frame := cjt188.BuildCJT1882018Frame([]byte{0xFE, 0xFE}, cjt188.MeterTypeColdWater)
_ = frame.BindDecryptKey(key) // 16-byte SM4 key when using ciphertext

err := frame.Decode(bufio.NewReader(conn))
if err != nil {
    // handle
}

if frame.IsReadData() && frame.IsNormalResponse() {
    di0, di1, data, ok := frame.ReadResult(true) // true = strip unit codes
    _, _, _, _ = di0, di1, data, ok
}
```

### Send read

```go
req, err := frame.Copy().
    BindMeterType(cjt188.MeterTypeColdWater).
    Read(0x90, 0x1F, false) // DI0, DI1, encrypt=false
_ = err
_ = req
```

### Send write

```go
item, err := cjt188.EncodeWriteData(3.96, 4, true, cjt188.UnitM3)
if err != nil {
    // handle
}
payload := cjt188.JoinWritePayload(item)

req, err := frame.Copy().Write(0xA0, 0x17, payload, false)
_ = err
_ = req
```

For `XXXX.XXXX` (4 decimal places):

```go
cjt188.EncodeWriteDataWithDecimals(7.1180, 4, 4, true, cjt188.UnitM3H)
```

### Ciphertext mode

1. `BindDecryptKey(16-byte key)`
2. Pass `encrypt=true` on `Read` / `Write` (control bit D3=1)
3. Algorithm: SM4-CBC; IV = `T + A0~A6 + SER×8`
4. Only the DATA part after DI/SER is encrypted (6-byte timestamp prefix + PKCS7)

### Main API

| API | Description |
|-----|-------------|
| `BuildCJT1882018Frame` | Create a session frame |
| `Copy` | Clone for sending (shares ser/key/guide) |
| `Decode` / `Encode` | Parse / build frames |
| `Read` / `Write` | Build read/write request frames |
| `ReadResult` / `WriteResult` | Extract DI + payload from normal responses |
| `AbnormalResult` | Extract SER + status from abnormal responses |
| `EncodeWriteData` / `JoinWritePayload` | Build write payloads |
| `MeterType` / `Address` / `Control` | Field accessors |

### Dependencies

- Go 1.21+
- `github.com/tjfoc/gmsm` (SM4)

### License

See repository license file if present.
