package go_cjt188

// MeterType 仪表类型
type MeterType byte

const (
	// 10H~19H: 水表
	MeterTypeColdWater        MeterType = 0x10 // 冷水水表
	MeterTypeDomesticHotWater MeterType = 0x11 // 生活热水水表
	MeterTypeDirectDrinking   MeterType = 0x12 // 直饮水水表
	MeterTypeReclaimedWater   MeterType = 0x13 // 中水水表

	// 20H~29H: 热量表
	MeterTypeHeat           MeterType = 0x20 // 热量表(计热量)
	MeterTypeCooling        MeterType = 0x21 // 热量表(计冷量)
	MeterTypeHeatAndCooling MeterType = 0x22 // 热量表(计热量和冷量)

	// 30H~39H: 燃气表
	MeterTypeGas MeterType = 0x30 // 燃气表

	// 40H~49H: 自定义仪表
)

// String 返回仪表类型说明
func (t MeterType) String() string {
	switch t {
	case MeterTypeColdWater:
		return "冷水水表"
	case MeterTypeDomesticHotWater:
		return "生活热水水表"
	case MeterTypeDirectDrinking:
		return "直饮水水表"
	case MeterTypeReclaimedWater:
		return "中水水表"
	case MeterTypeHeat:
		return "热量表(计热量)"
	case MeterTypeCooling:
		return "热量表(计冷量)"
	case MeterTypeHeatAndCooling:
		return "热量表(计热量和冷量)"
	case MeterTypeGas:
		return "燃气表"
	default:
		if t >= 0x10 && t <= 0x19 {
			return "水表"
		}
		if t >= 0x20 && t <= 0x29 {
			return "热量表"
		}
		if t >= 0x30 && t <= 0x39 {
			return "燃气表"
		}
		if t >= 0x40 && t <= 0x49 {
			return "自定义仪表"
		}
		return "未知仪表类型"
	}
}
