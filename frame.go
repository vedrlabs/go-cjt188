package go_cjt188

import (
	"bufio"
	"crypto/cipher"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/tjfoc/gmsm/sm4"
)

const (
	StartChar byte = 0x68 //起始字符
	guideChar byte = 0xFF //前导字
	EndChar   byte = 0x16
)

// BuildCJT1882018Frame 创建一个CJT188帧报文解析
func BuildCJT1882018Frame(guide []byte, meterType MeterType) *CJT1882018Frame {
	return &CJT1882018Frame{
		guide:     guide,
		meterType: meterType,
		ser:       NewSER(),
	}
}

// CJT1882018Frame CJT188帧报文
type CJT1882018Frame struct {
	guide     []byte
	meterType MeterType //仪表类型
	address   *Address  //地址
	control   *Control  //控制域
	data      []byte    //数据
	key       []byte    //SM4密钥（密文传输时由 BindDecryptKey 绑定）
	ser       *SER
}

// Copy copy一个
func (f *CJT1882018Frame) Copy() *CJT1882018Frame {
	return &CJT1882018Frame{
		guide:     f.guide,
		meterType: f.meterType,
		ser:       f.ser,
		key:       f.key,
		address:   f.address,
		control:   f.control,
	}
}

// BindMeterType 绑定表类型
func (f *CJT1882018Frame) BindMeterType(meterType MeterType) *CJT1882018Frame {
	f.meterType = meterType
	return f
}

// BindAddress 绑定地址域（原始值 A0~A6，共 7 字节）
func (f *CJT1882018Frame) BindAddress(raw []byte) error {
	if len(raw) != 7 {
		return fmt.Errorf("address length error: need 7 bytes, got %d", len(raw))
	}
	addr := buildAddress()
	if err := addr.decode(raw); err != nil {
		return err
	}
	f.address = addr
	return nil
}

// BindAddressHex 绑定地址域（正序十六进制表号 5 字节、厂家编码 2 字节，内部反转为线序）
func (f *CJT1882018Frame) BindAddressHex(addrHex, manufacturerHex string) error {
	addr := buildAddress()
	if err := addr.decodeHex(addrHex, manufacturerHex); err != nil {
		return err
	}
	f.address = addr
	return nil
}

// BindControl 绑定控制域（原始值，1 字节控制码）
func (f *CJT1882018Frame) BindControl(raw byte) error {
	ctrl := buildControl()
	if err := ctrl.decode(raw); err != nil {
		return err
	}
	f.control = ctrl
	return nil
}

// BindControlFields 按字段绑定控制域
func (f *CJT1882018Frame) BindControlFields(direction Direction, abnormal Abnormal, mode TransferMode, function Function) *CJT1882018Frame {
	f.control = BuildControl(direction, abnormal, mode, function)
	return f
}

// MeterType 获取仪表类型
func (f *CJT1882018Frame) MeterType() MeterType {
	return f.meterType
}

// Address 获取地址域
func (f *CJT1882018Frame) Address() *Address {
	return f.address
}

// Control 获取控制域
func (f *CJT1882018Frame) Control() *Control {
	return f.control
}

// BindDecryptKey 绑定解密密钥（16字节 SM4 密钥）
func (f *CJT1882018Frame) BindDecryptKey(key []byte) error {
	if len(key) != 16 {
		return fmt.Errorf("SM4 key length error: need 16 bytes, got %d", len(key))
	}
	f.key = append([]byte(nil), key...)
	return nil
}

func (f *CJT1882018Frame) Decode(buf *bufio.Reader) (err error) {
	var startChar byte
	//先解析起始字符
	for {
		if err = binary.Read(buf, binary.BigEndian, &startChar); err != nil {
			return
		}
		//过滤掉前导字
		if startChar == guideChar {
			continue
		}
		if startChar == StartChar {
			break
		}
		continue
	}
	csFrame := []byte{StartChar}
	//解析仪表类型
	var meterType byte
	if err = binary.Read(buf, binary.BigEndian, &meterType); err != nil {
		return
	}
	csFrame = append(csFrame, meterType)
	f.meterType = MeterType(meterType)
	//解析地址域
	f.address = buildAddress()
	address := make([]byte, f.address.TheoreticalLength())
	if err = binary.Read(buf, binary.BigEndian, &address); err != nil {
		return
	}
	csFrame = append(csFrame, address...)
	if err = f.address.decode(address); err != nil {
		return
	}
	//解码控制域
	f.control = buildControl()
	var control byte
	if err = binary.Read(buf, binary.BigEndian, &control); err != nil {
		return
	}
	csFrame = append(csFrame, control)
	if err = f.control.decode(control); err != nil {
		return
	}
	//获取数据长度域
	var length byte
	if err = binary.Read(buf, binary.BigEndian, &length); err != nil {
		return
	}
	csFrame = append(csFrame, length)
	//获取数据
	data := make([]byte, length)
	if err = binary.Read(buf, binary.BigEndian, &data); err != nil {
		return
	}
	csFrame = append(csFrame, data...)
	//获取cs
	var cs byte
	if err = binary.Read(buf, binary.BigEndian, &cs); err != nil {
		return
	}
	//计算cs
	frameCs := f.cs(csFrame)
	if frameCs != cs {
		err = fmt.Errorf("frame cs not equal, need:%d but:%d", cs, frameCs)
		return
	}
	var endChar byte
	if err = binary.Read(buf, binary.BigEndian, &endChar); err != nil {
		return
	}
	if endChar != EndChar {
		err = fmt.Errorf("end char not equal, need:%d but:%d", EndChar, endChar)
		return
	}
	//判断是不是加密的
	if f.control.IsTransferModeCipher() {
		f.data, err = f.decrypt(data)
		return
	}
	f.data = data
	return
}

// decrypt 解密数据域：DI(2)+SER(1)保持明文，其余为 SM4-CBC 密文；
// 解密后去掉 PKCS7 填充和前置 6 字节时间戳，返回 DI|SER|业务数据。
func (f *CJT1882018Frame) decrypt(data []byte) (decrypted []byte, err error) {
	if len(f.key) != 16 {
		return nil, fmt.Errorf("decrypt key not bound, call BindDecryptKey first")
	}
	// DATA = DI0 DI1 SER | 密文
	const diLen, serLen, tsLen = 2, 1, 6
	if len(data) < diLen+serLen+sm4.BlockSize {
		return nil, fmt.Errorf("ciphertext data field too short: %d", len(data))
	}
	di := data[:diLen]
	ser := data[diLen]
	ciphertext := data[diLen+serLen:]
	if len(ciphertext)%sm4.BlockSize != 0 {
		return nil, fmt.Errorf("ciphertext length is not a multiple of %d: %d", sm4.BlockSize, len(ciphertext))
	}

	block, err := sm4.NewCipher(f.key)
	if err != nil {
		return nil, fmt.Errorf("create SM4 cipher failed: %w", err)
	}
	plain := make([]byte, len(ciphertext))
	cipher.NewCBCDecrypter(block, f.iv(ser)).CryptBlocks(plain, ciphertext)

	plain, err = pkcs7Unpad(plain, sm4.BlockSize)
	if err != nil {
		return nil, err
	}
	if len(plain) < tsLen {
		return nil, fmt.Errorf("decrypted data shorter than timestamp: %d", len(plain))
	}
	// 去掉前置 6 字节 BCD 时间戳 ss mm hh DD MM YY
	payload := plain[tsLen:]

	decrypted = make([]byte, 0, diLen+serLen+len(payload))
	decrypted = append(decrypted, di...)
	decrypted = append(decrypted, ser)
	decrypted = append(decrypted, payload...)
	return decrypted, nil
}

// iv 按表4构造 CBC 初始化向量：T + A0~A6 + SER×8
// ser 取自当前报文 DATA，不能用会话 SER 的当前值替代
func (f *CJT1882018Frame) iv(ser byte) []byte {
	iv := make([]byte, 16)
	iv[0] = byte(f.meterType)
	copy(iv[1:8], f.address.bytes())
	copy(iv[8:16], repeatByte(ser, 8))
	return iv
}

// pkcs7Unpad 去除 PKCS7 填充
func pkcs7Unpad(data []byte, blockSize int) ([]byte, error) {
	if len(data) == 0 || len(data)%blockSize != 0 {
		return nil, fmt.Errorf("invalid PKCS7 padded length: %d", len(data))
	}
	pad := int(data[len(data)-1])
	if pad == 0 || pad > blockSize || pad > len(data) {
		return nil, fmt.Errorf("invalid PKCS7 padding value: %d", pad)
	}
	for i := 0; i < pad; i++ {
		if data[len(data)-1-i] != byte(pad) {
			return nil, fmt.Errorf("PKCS7 padding check failed")
		}
	}
	return data[:len(data)-pad], nil
}

// cs 计算校验码：从帧起始符到校验码之前所有字节二进制算术累加，不计超过 FFH 的溢出值
func (f *CJT1882018Frame) cs(frame []byte) byte {
	var sum byte
	for _, b := range frame {
		sum += b
	}
	return sum
}

// Encode 编码为完整帧：前导字 + 68H | T | A0~A6 | C | L | DATA | CS | 16H
func (f *CJT1882018Frame) Encode() ([]byte, error) {
	if f.address == nil {
		return nil, fmt.Errorf("address is nil")
	}
	if f.control == nil {
		return nil, fmt.Errorf("control is nil")
	}

	data := f.data
	if f.control.IsTransferModeCipher() {
		var err error
		data, err = f.encrypt(f.data)
		if err != nil {
			return nil, err
		}
	}
	if len(data) > 0xFF {
		return nil, fmt.Errorf("data field too long: %d", len(data))
	}

	// 先组协议帧（不含前导码），L/CS 只相对协议帧计算
	frame := make([]byte, 0, 12+len(data))
	frame = append(frame, StartChar)
	frame = append(frame, byte(f.meterType))
	frame = append(frame, f.address.bytes()...)
	frame = append(frame, f.control.encode())
	frame = append(frame, byte(len(data)))
	frame = append(frame, data...)
	frame = append(frame, f.cs(frame))
	frame = append(frame, EndChar)

	// 前导码仅贴在最前面，不参与长度与校验
	if len(f.guide) == 0 {
		return frame, nil
	}
	out := make([]byte, 0, len(f.guide)+len(frame))
	out = append(out, f.guide...)
	out = append(out, frame...)
	return out, nil
}

// encrypt 加密数据域：DI(2)+SER(1)保持明文，其余加时间戳和 PKCS7 后 SM4-CBC 加密
func (f *CJT1882018Frame) encrypt(data []byte) ([]byte, error) {
	if len(f.key) != 16 {
		return nil, fmt.Errorf("encrypt key not bound, call BindDecryptKey first")
	}
	const diLen, serLen = 2, 1
	if len(data) < diLen+serLen {
		return nil, fmt.Errorf("data field too short: %d", len(data))
	}
	di := data[:diLen]
	ser := data[diLen]
	payload := data[diLen+serLen:]

	plain := append(timestampBCD(), payload...)
	plain = pkcs7Pad(plain, sm4.BlockSize)

	block, err := sm4.NewCipher(f.key)
	if err != nil {
		return nil, fmt.Errorf("create SM4 cipher failed: %w", err)
	}
	ciphertext := make([]byte, len(plain))
	cipher.NewCBCEncrypter(block, f.iv(ser)).CryptBlocks(ciphertext, plain)

	out := make([]byte, 0, diLen+serLen+len(ciphertext))
	out = append(out, di...)
	out = append(out, ser)
	out = append(out, ciphertext...)
	return out, nil
}

// timestampBCD 当前时间戳 ss mm hh DD MM YY（BCD）
func timestampBCD() []byte {
	now := time.Now()
	return []byte{
		binToBCD(now.Second()),
		binToBCD(now.Minute()),
		binToBCD(now.Hour()),
		binToBCD(now.Day()),
		binToBCD(int(now.Month())),
		binToBCD(now.Year() % 100),
	}
}

func binToBCD(v int) byte {
	return byte((v/10)<<4 | (v % 10))
}

// pkcs7Pad PKCS7 填充到 blockSize 整数倍
func pkcs7Pad(data []byte, blockSize int) []byte {
	pad := blockSize - len(data)%blockSize
	if pad == 0 {
		pad = blockSize
	}
	out := make([]byte, len(data)+pad)
	copy(out, data)
	for i := len(data); i < len(out); i++ {
		out[i] = byte(pad)
	}
	return out
}

// IsReadData 是否为读数据功能
func (f *CJT1882018Frame) IsReadData() bool {
	return f.control != nil && f.control.IsFunctionReadData()
}

// IsWriteData 是否为写数据功能
func (f *CJT1882018Frame) IsWriteData() bool {
	return f.control != nil && f.control.IsFunctionWriteData()
}

// IsNormalResponse 是否为从站正常应答
func (f *CJT1882018Frame) IsNormalResponse() bool {
	return f.control != nil && f.control.IsDirectionSlave() && f.control.IsAbnormalNo()
}

// IsAbnormalResponse 是否为从站异常应答
func (f *CJT1882018Frame) IsAbnormalResponse() bool {
	return f.control != nil && f.control.IsDirectionSlave() && f.control.IsAbnormalYes()
}

// ReadResult 读数据正常应答时返回 DI0、DI1 与业务数据（不含 SER）；否则 ok=false
// hasUnit 为 true 时，按表19去掉各数据项末尾的单位代号（每项 5 字节：4 字节数值 + 1 字节单位）
func (f *CJT1882018Frame) ReadResult(hasUnit bool) (di0, di1 byte, data []byte, ok bool) {
	if !f.IsReadData() || !f.IsNormalResponse() {
		return 0, 0, nil, false
	}
	di0, di1, data, ok = f.normalResult()
	if !ok {
		return 0, 0, nil, false
	}
	if hasUnit {
		data = stripUnitCodes(data)
	}
	return di0, di1, data, true
}

// WriteResult 写数据正常应答时返回 DI0、DI1 与业务数据（不含 SER）；否则 ok=false
// hasUnit 为 true 时，按表19去掉各数据项末尾的单位代号
func (f *CJT1882018Frame) WriteResult(hasUnit bool) (di0, di1 byte, data []byte, ok bool) {
	if !f.IsWriteData() || !f.IsNormalResponse() {
		return 0, 0, nil, false
	}
	di0, di1, data, ok = f.normalResult()
	if !ok {
		return 0, 0, nil, false
	}
	if hasUnit {
		data = stripUnitCodes(data)
	}
	return di0, di1, data, true
}

// AbnormalResult 异常应答时返回 SER 与状态 ST；否则 ok=false
// 异常应答数据域：SER | ST（L=03H 时 ST 通常为 2 字节）
func (f *CJT1882018Frame) AbnormalResult() (ser byte, st []byte, ok bool) {
	if !f.IsAbnormalResponse() {
		return 0, nil, false
	}
	if len(f.data) < 1 {
		return 0, nil, false
	}
	ser = f.data[0]
	if f.ser != nil {
		f.ser.OnReceive(ser)
	}
	st = append([]byte(nil), f.data[1:]...)
	return ser, st, true
}

// normalResult 解析正常应答数据域：DI0 DI1 SER | N1...Nm
func (f *CJT1882018Frame) normalResult() (di0, di1 byte, data []byte, ok bool) {
	if len(f.data) < 3 {
		return 0, 0, nil, false
	}
	di0, di1 = f.data[0], f.data[1]
	if f.ser != nil {
		f.ser.OnReceive(f.data[2])
	}
	data = append([]byte(nil), f.data[3:]...)
	return di0, di1, data, true
}

// stripUnitCodes 去掉带单位代号数据项的末尾 1 字节单位（表19：总长 5 = 数值 4 + 单位 1）
// 按 5 字节一组剥离；不足一组的尾部原样保留（如时间、状态字等）
func stripUnitCodes(data []byte) []byte {
	const itemLen, valueLen = 5, 4
	n := len(data) / itemLen
	out := make([]byte, 0, n*valueLen+len(data)%itemLen)
	for i := 0; i < n; i++ {
		off := i * itemLen
		out = append(out, data[off:off+valueLen]...)
	}
	out = append(out, data[n*itemLen:]...)
	return out
}
