package pool

import (
	"testing"
	"fmt"
)

func TestNewWorker(t *testing.T) {
	workPool := NewWorkerPool(10)

	workPool.Run()

	workPool.AddWorker(NewWorker(func() {
		fmt.Println("hello")
	}))

	workPool.AddWorker(NewWorker(func() {
		fmt.Println("hello1")
	}))

	workPool.WaitStop()
}
