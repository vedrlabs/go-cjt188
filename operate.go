package go_cjt188

// Read 生成一个读操作的帧（主站请求读数据：DI0 DI1 SER）
// encrypt 为 true 时使用密文传输（控制码 D3=1）
func (f *CJT1882018Frame) Read(di0, di1 byte, encrypt bool) ([]byte, error) {
	if f.address != nil {
		addr := *f.address
		f.address = &addr
	}
	mode := TransferModePlain
	if encrypt {
		mode = TransferModeCipher
	}
	f.control = &Control{
		direction: DirectionMaster,
		abnormal:  AbnormalNo,
		mode:      mode,
		function:  FunctionReadData,
	}
	f.data = []byte{di0, di1, f.ser.ForSend()}
	return f.Encode()
}

// Write 生成一个写操作的帧（主站请求写数据：DI0 DI1 SER + 数据）
// encrypt 为 true 时使用密文传输（控制码 D3=1）
func (f *CJT1882018Frame) Write(di0, di1 byte, payload []byte, encrypt bool) ([]byte, error) {
	if f.address != nil {
		addr := *f.address
		f.address = &addr
	}
	mode := TransferModePlain
	if encrypt {
		mode = TransferModeCipher
	}
	f.control = &Control{
		direction: DirectionMaster,
		abnormal:  AbnormalNo,
		mode:      mode,
		function:  FunctionWriteData,
	}
	f.data = make([]byte, 0, 3+len(payload))
	f.data = append(f.data, di0, di1, f.ser.ForSend())
	f.data = append(f.data, payload...)
	return f.Encode()
}
