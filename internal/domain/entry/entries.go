package entry

type Entries []Entry

// Summation は Entries の合計値を計算します。
func (e Entries) Summation() int64 {
	var total int64
	for _, entry := range e {
		// ここでは仮に Data["value"] が数値であると仮定します。
		// 実際のロジックに合わせて修正してください。
		if value, ok := entry.Data["value"].(float64); ok {
			total += int64(value)
		}
	}
	return total
}

// DiscountSummation は Entries の割引合計値を計算します。
func (e Entries) DiscountSummation() int64 {
	var total int64
	for _, entry := range e {
		// ここでは仮に Data["value"] と Data["discount"] が数値であると仮定します。
		// 実際のロジックに合わせて修正してください。
		value, valueOk := entry.Data["value"].(float64)
		discount, discountOk := entry.Data["discount"].(float64)
		if valueOk && discountOk {
			total += int64(value - discount)
		} else if valueOk {
			total += int64(value)
		}
	}
	return total
}
