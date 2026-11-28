package go_cjt188

func buildControl() *Control {
	return &Control{}
}

// BuildControl 按字段组成控制码
func BuildControl(direction Direction, abnormal Abnormal, mode TransferMode, function Function) *Control {
	return &Control{
		direction: direction,
		abnormal:  abnormal,
		mode:      mode,
		function:  function,
	}
}

// Direction 传送方向（D7）
type Direction byte

const (
	DirectionMaster Direction = 0 // 由主站发出的控制帧
	DirectionSlave  Direction = 1 // 由从站发出的应答帧
)

// String 返回传送方向说明
func (d Direction) String() string {
	if d == DirectionSlave {
		return "由从站发出的应答帧"
	}
	return "由主站发出的控制帧"
}

// IsDirectionMaster 是否由主站发出的控制帧
func (d Direction) IsDirectionMaster() bool { return d == DirectionMaster }

// IsDirectionSlave 是否由从站发出的应答帧
func (d Direction) IsDirectionSlave() bool { return d == DirectionSlave }

// Abnormal 通信异常标志（D6）
type Abnormal byte

const (
	AbnormalNo  Abnormal = 0 // 通信正常
	AbnormalYes Abnormal = 1 // 通信异常
)

// String 返回通信状态说明
func (a Abnormal) String() string {
	if a == AbnormalYes {
		return "通信异常"
	}
	return "通信正常"
}

// IsAbnormalNo 是否通信正常
func (a Abnormal) IsAbnormalNo() bool { return a == AbnormalNo }

// IsAbnormalYes 是否通信异常
func (a Abnormal) IsAbnormalYes() bool { return a == AbnormalYes }

// TransferMode 传输模式（D3）
type TransferMode byte

const (
	TransferModePlain  TransferMode = 0 // 明文传输
	TransferModeCipher TransferMode = 1 // 密文传输
)

// String 返回传输模式说明
func (m TransferMode) String() string {
	if m == TransferModeCipher {
		return "密文传输"
	}
	return "明文传输"
}

// IsTransferModePlain 是否明文传输
func (m TransferMode) IsTransferModePlain() bool { return m == TransferModePlain }

// IsTransferModeCipher 是否密文传输
func (m TransferMode) IsTransferModeCipher() bool { return m == TransferModeCipher }

// Function 功能定义（D5~D0，匹配时忽略 D3）
type Function byte

const (
	FunctionReserved     Function = 0x00 // 保留
	FunctionReadData     Function = 0x01 // 读数据
	FunctionWriteData    Function = 0x04 // 写数据
	FunctionReadAddress  Function = 0x03 // 读地址(表号)
	FunctionWriteAddress Function = 0x15 // 写地址(表号)
	FunctionWriteSync    Function = 0x16 // 写机电同步数(置表底数)
)

// String 返回功能定义说明
func (f Function) String() string {
	if f.IsFunctionManufacturer() {
		return "厂商自定义"
	}
	switch f {
	case FunctionReserved:
		return "保留"
	case FunctionReadData:
		return "读数据"
	case FunctionWriteData:
		return "写数据"
	case FunctionReadAddress:
		return "读地址(表号)"
	case FunctionWriteAddress:
		return "写地址(表号)"
	case FunctionWriteSync:
		return "写机电同步数(置表底数)"
	default:
		return "未知功能"
	}
}

// IsFunctionReserved 是否保留
func (f Function) IsFunctionReserved() bool { return f == FunctionReserved }

// IsFunctionReadData 是否读数据
func (f Function) IsFunctionReadData() bool { return f == FunctionReadData }

// IsFunctionWriteData 是否写数据
func (f Function) IsFunctionWriteData() bool { return f == FunctionWriteData }

// IsFunctionReadAddress 是否读地址(表号)
func (f Function) IsFunctionReadAddress() bool { return f == FunctionReadAddress }

// IsFunctionWriteAddress 是否写地址(表号)
func (f Function) IsFunctionWriteAddress() bool { return f == FunctionWriteAddress }

// IsFunctionWriteSync 是否写机电同步数(置表底数)
func (f Function) IsFunctionWriteSync() bool { return f == FunctionWriteSync }

// IsFunctionManufacturer 是否厂商自定义
func (f Function) IsFunctionManufacturer() bool { return f&0x20 != 0 }

// Control 控制码
type Control struct {
	direction Direction    // D7 传送方向
	abnormal  Abnormal     // D6 通信异常标志
	mode      TransferMode // D3 传输模式
	function  Function     // D5~D0 功能定义（已忽略 D3）
}

// decode 解码
func (t *Control) decode(data byte) error {
	t.direction = Direction((data >> 7) & 0x01)
	t.abnormal = Abnormal((data >> 6) & 0x01)
	t.mode = TransferMode((data >> 3) & 0x01)
	// D5~D0 功能定义，匹配时忽略 D3（掩码 0x37 = 0011 0111）
	t.function = Function(data & 0x37)
	return nil
}

// encode 编码为 1 字节控制码
func (t *Control) encode() byte {
	return byte(t.direction)<<7 |
		byte(t.abnormal)<<6 |
		byte(t.mode)<<3 |
		(byte(t.function) & 0x37)
}

// Values 一次性获取控制码全部字段
func (t *Control) Values() (direction Direction, abnormal Abnormal, mode TransferMode, function Function) {
	return t.direction, t.abnormal, t.mode, t.function
}

// IsDirectionMaster 是否由主站发出的控制帧
func (t *Control) IsDirectionMaster() bool { return t.direction.IsDirectionMaster() }

// IsDirectionSlave 是否由从站发出的应答帧
func (t *Control) IsDirectionSlave() bool { return t.direction.IsDirectionSlave() }

// IsAbnormalNo 是否通信正常
func (t *Control) IsAbnormalNo() bool { return t.abnormal.IsAbnormalNo() }

// IsAbnormalYes 是否通信异常
func (t *Control) IsAbnormalYes() bool { return t.abnormal.IsAbnormalYes() }

// IsTransferModePlain 是否明文传输
func (t *Control) IsTransferModePlain() bool { return t.mode.IsTransferModePlain() }

// IsTransferModeCipher 是否密文传输
func (t *Control) IsTransferModeCipher() bool { return t.mode.IsTransferModeCipher() }

// IsFunctionReserved 是否保留
func (t *Control) IsFunctionReserved() bool { return t.function.IsFunctionReserved() }

// IsFunctionReadData 是否读数据
func (t *Control) IsFunctionReadData() bool { return t.function.IsFunctionReadData() }

// IsFunctionWriteData 是否写数据
func (t *Control) IsFunctionWriteData() bool { return t.function.IsFunctionWriteData() }

// IsFunctionReadAddress 是否读地址(表号)
func (t *Control) IsFunctionReadAddress() bool { return t.function.IsFunctionReadAddress() }

// IsFunctionWriteAddress 是否写地址(表号)
func (t *Control) IsFunctionWriteAddress() bool { return t.function.IsFunctionWriteAddress() }

// IsFunctionWriteSync 是否写机电同步数(置表底数)
func (t *Control) IsFunctionWriteSync() bool { return t.function.IsFunctionWriteSync() }

// IsFunctionManufacturer 是否厂商自定义
func (t *Control) IsFunctionManufacturer() bool { return t.function.IsFunctionManufacturer() }
