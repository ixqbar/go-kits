package pool

import (
	"fmt"
	"testing"
)

func TestNewWorker(t *testing.T) {
	workPool := NewWorkerPool(10)

	workPool.Run()

	workPool.AddWorker(NewWorker(func(params interface{}) {
		fmt.Println("hello")
	}, nil))

	workPool.AddWorker(NewWorker(func(params interface{}) {
		fmt.Println("hello1")
		fmt.Println(params)
	}, "todo"))

	workPool.WaitStop()
}
