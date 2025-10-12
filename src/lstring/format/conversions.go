package format

func toInt(val any) (int64, bool) {
	switch tval := val.(type) {
	case uint:
		return int64(tval), true
	case uint8:
		return int64(tval), true
	case uint16:
		return int64(tval), true
	case uint32:
		return int64(tval), true
	case int:
		return int64(tval), true
	case int8:
		return int64(tval), true
	case int16:
		return int64(tval), true
	case int32:
		return int64(tval), true
	case int64:
		return tval, true
	case float32:
		return int64(tval), true
	case float64:
		return int64(tval), true
	default:
		return 0, false
	}
}

func toFloat(val any) (float64, bool) {
	switch tval := val.(type) {
	case uint:
		return float64(tval), true
	case uint8:
		return float64(tval), true
	case uint16:
		return float64(tval), true
	case uint32:
		return float64(tval), true
	case int:
		return float64(tval), true
	case int8:
		return float64(tval), true
	case int16:
		return float64(tval), true
	case int32:
		return float64(tval), true
	case int64:
		return float64(tval), true
	case float32:
		return float64(tval), true
	case float64:
		return tval, true
	default:
		return 0, false
	}
}
