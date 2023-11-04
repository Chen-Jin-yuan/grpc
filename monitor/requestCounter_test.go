package monitor

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"
)

func TestRequestCounter(t *testing.T) {
	rc := NewRequestCounters()
	rc.GetCounter("request-type").IncrementOfValue("v1")
	rc.GetCounter("request-type").IncrementOfValue("v1")
	rc.GetCounter("request-type").IncrementOfValue("v2")
	rc.GetCounter("request-type").IncrementOfValue("v3")
	rc.GetCounter(10001).IncrementOfValue(3)
	rc.GetCounter(10001).IncrementOfValue(2)
	rc.GetCounter(10001).IncrementOfValue(3)
	rc.GetCounter("hello").IncrementOfValue(3)

	data, err := rc.ToJSON()
	if err != nil {
		fmt.Printf("ToJSON err: %v\n", err)
	}
	fmt.Printf("data: %+v\n", data)

	jsonData, err := json.Marshal(&data)
	if err != nil {
		fmt.Printf("marshal err: %v\n", err)
		return
	}
	err = os.WriteFile("exam_counter.json", jsonData, 0644)
	if err != nil {
		fmt.Printf("writeFile err: %v\n", err)
		return
	}
}
func TestRequestCounterMultiThread(t *testing.T) {
	var wg sync.WaitGroup
	rc := NewRequestCounters()
	go write(rc, "request-type", 1, &wg)
	wg.Add(1)
	go write(rc, "request-type", 2, &wg)
	wg.Add(1)
	go write(rc, "request-type", 3, &wg)
	wg.Add(1)
	go write(rc, "request-type", 4, &wg)
	wg.Add(1)

	go get(rc)

	wg.Wait()
}
func write(rc *RequestCounters, key string, value int, wg *sync.WaitGroup) {
	for i := 0; i < 10; i++ {
		rc.GetCounter(key).IncrementOfValue(value)
		time.Sleep(time.Duration(value * 5e8))
	}
	wg.Done()
}
func get(rc *RequestCounters) {
	for {
		data, err := rc.ToJSON()
		if err != nil {
			fmt.Printf("ToJSON err: %v\n", err)
		}
		fmt.Printf("data: %+v\n", data)
		time.Sleep(1e9)
	}
}
