package pool

type Worker struct {
	worker func()
}

func NewWorker(worker func()) *Worker {
	return &Worker{
		worker: worker,
	}
}

func (obj *Worker) Run() {
	obj.worker()
}

type WorkerPool struct {
	jobWorker   chan *Worker
	stopChannel chan bool
}

func NewWorkerPool(cap int) *WorkerPool {
	return &WorkerPool{
		jobWorker:   make(chan *Worker, cap),
		stopChannel: make(chan bool),
	}
}

func (obj *WorkerPool) AddWorker(worker *Worker) {
	obj.jobWorker <- worker
}

func (obj *WorkerPool) Run() {
	go func() {
		for {
			select {
			case <-obj.stopChannel:
				return
			case worker := <-obj.jobWorker:
				worker.Run()
			}
		}
	}()
}

func (obj *WorkerPool) Stop() {
	obj.stopChannel <- true
}
