package pool

import "sync"

type Worker struct {
	worker func(params interface{})
	params interface{}
}

func NewWorker(worker func(params interface{}), params interface{}) *Worker {
	return &Worker{
		worker: worker,
		params: params,
	}
}

func (obj *Worker) Run() {
	obj.worker(obj.params)
}

type WorkerPool struct {
	waitGroup   sync.WaitGroup
	jobWorkers  chan *Worker
	stopChannel chan bool
	willStop    bool
}

func NewWorkerPool(cap int) *WorkerPool {
	return &WorkerPool{
		jobWorkers:  make(chan *Worker, cap),
		stopChannel: make(chan bool),
	}
}

func (obj *WorkerPool) AddWorker(worker *Worker) {
	if obj.willStop {
		return
	}

	obj.waitGroup.Add(1)
	obj.jobWorkers <- worker
}

func (obj *WorkerPool) Run() {
	go func() {
		for {
			select {
			case <-obj.stopChannel:
				return
			case worker := <-obj.jobWorkers:
				worker.Run()
				obj.waitGroup.Done()
			}
		}
	}()
}

func (obj *WorkerPool) Wait() {
	obj.willStop = true
	obj.waitGroup.Wait()
}

func (obj *WorkerPool) Stop() {
	obj.willStop = true
	obj.stopChannel <- true
}

func (obj *WorkerPool) WaitStop() {
	obj.Wait()
	obj.Stop()
}
