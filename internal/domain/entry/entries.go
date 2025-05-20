package entry // entiryの配列としてのentriesのメソッドとして単純なsumの計算．メソッドは今後増やす可能性あるよな

import "fmt"

type Entries []Entry

func (e Entries) SumAll() (float64, error) {
	var sum float64
	for _, entry := range e {
		val, ok := entry.Data["amount"]
		if !ok {
			return 0, fmt.Errorf("amount field not found in entry data for entry ID %s", entry.EntryID)
		}
		switch v := val.(type) {
		case float64:
			sum += v
		case int:
			sum += float64(v)
		case int32:
			sum += float64(v)
		case int64:
			sum += float64(v)
		default:
		}
	}
	return sum, nil
}
