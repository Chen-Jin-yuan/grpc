package monitor

type Counter interface {
	IncrementOfValue(value interface{})
	GetCountOfValue(value interface{}) int
	GetData() map[interface{}]int
}
