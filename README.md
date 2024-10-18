# go-dispatcher

dispatcher in go, obtained from [Handling 1 Million Requests per Minute with Go](http://marcio.io/2015/07/handling-1-million-requests-per-minute-with-golang/)

First of all you have to run the following command at the start of the app in which you want to use the dispatcher

```go
dispatcher := NewDispatcher(MaxWorker)
dispatcher.Run()
```

Then, let’s break down the code, focusing on how the `Dispatcher` and `Worker` work together to execute jobs.

## Overview

This code implements a **worker pool** pattern in Go. Here's how it works:

1. The `Dispatcher` manages a pool of workers.
2. The workers process jobs from a job queue (`JobQueue`).
3. When a job arrives, the dispatcher assigns it to an available worker.
4. Each worker registers itself in the `WorkerPool` to signal availability.

### Key Components

#### 1. **Dispatcher**

- The `Dispatcher` is responsible for managing the workers and assigning jobs to them.
- It has a `WorkerPool` channel, which holds the channels of idle workers (each channel in the pool is a `chan Job`, the worker's job channel).

##### Dispatcher Initialization

```go
func NewDispatcher(maxWorkers int) *Dispatcher {
    pool := make(chan chan Job, maxWorkers)  // Buffered channel
    return &Dispatcher{WorkerPool: pool}
}
```

- The `WorkerPool` is a **buffered channel** (`make(chan chan Job, maxWorkers)`). The buffer size is equal to the number of workers (`maxWorkers`), meaning the pool can hold up to `maxWorkers` idle workers at any time.

##### Dispatcher Run

```go
func (d *Dispatcher) Run() {
    maxWorkers, _ := strconv.Atoi(MaxWorker)
    // starting n number of workers
    for i := 0; i < maxWorkers; i++ {
        worker := NewWorker(d.WorkerPool)  // Create a worker
        worker.Start()                     // Start the worker (runs in a goroutine)
    }

    go d.dispatch()  // Run the dispatch logic in a goroutine
}
```

- This method creates `maxWorkers` number of workers. Each worker runs in a separate goroutine.
- The `dispatch()` method is also run in a separate goroutine, which listens for jobs in the `JobQueue`.

##### Dispatcher Job Dispatch

```go
func (d *Dispatcher) dispatch() {
    for {
        select {
        case job := <-JobQueue:  // A job is received
            go func(job Job) {
                jobChannel := <-d.WorkerPool  // Wait for an available worker
                jobChannel <- job             // Assign the job to the worker
            }(job)
        }
    }
}
```

- The dispatcher constantly listens for new jobs from the `JobQueue`.
- When a job is received:
  - The dispatcher waits (blocks) until an idle worker is available by receiving from `d.WorkerPool`. The worker pool holds the job channels of workers that are ready for work.
  - It then sends the job to the idle worker via its job channel (`jobChannel <- job`).
- Since this happens inside a goroutine (`go func(job Job)`), the dispatcher can handle multiple jobs concurrently, as each job will get dispatched in its own goroutine.

#### 2. **Worker**

- A `Worker` is responsible for executing jobs.
- Each worker has:
  - A `WorkerPool` channel to register itself as available for work.
  - A `JobChannel` where it listens for jobs to process.
  - A `quit` channel to signal when the worker should stop.

##### Worker Initialization and Start

```go
type Worker struct {
    WorkerPool chan chan Job
    JobChannel chan Job
    quit       chan bool
}

func NewWorker(workerPool chan chan Job) Worker {
    return Worker{
        WorkerPool: workerPool,
        JobChannel: make(chan Job),
        quit:       make(chan bool),
    }
}

func (w Worker) Start() {
    go func() {
        for {
            w.WorkerPool <- w.JobChannel  // Register this worker in the pool

            select {
            case job := <-w.JobChannel:
                // Job received, process it
                log.Debugf("Processing job %v", job)
                // Perform the job (example of job processing commented out)
                // job.Payload.UploadToS3()

            case <-w.quit:
                // Quit signal received, stop the worker
                return
            }
        }
    }()
}
```

- **Worker Registration**: Each worker continuously registers itself in the `WorkerPool` by sending its `JobChannel` into the pool (`w.WorkerPool <- w.JobChannel`). This signals that the worker is idle and ready to accept jobs.
- **Job Execution**: After registering itself in the pool, the worker waits in a `select` block:
  - If it receives a job on its `JobChannel`, it processes the job.
  - If it receives a quit signal on the `quit` channel, it stops the worker.

### Flow of Execution

1. **Dispatcher Starts**:
   - The `Run()` method of the `Dispatcher` starts multiple workers (`worker.Start()`) and launches the `dispatch()` method in a goroutine.

2. **Workers Register Themselves**:
   - Each worker, in its goroutine, registers itself in the `WorkerPool` by sending its `JobChannel` to `w.WorkerPool`. This makes it available to the dispatcher for job assignment.

3. **Job Arrival**:
   - When a job arrives (from `JobQueue`), the `dispatch()` method grabs an available worker’s job channel from `WorkerPool` (`jobChannel := <-d.WorkerPool`).
   - It then sends the job to the worker (`jobChannel <- job`).

4. **Job Processing**:
   - The worker, which is listening on its `JobChannel`, receives the job and processes it. Once done, the worker re-registers itself in the pool by sending its `JobChannel` back to `WorkerPool`, making it available for another job.

5. **Worker Continues**:
   - After processing the job, the worker goes back to the pool and waits for another job. This loop continues until a quit signal is received.

### Does `w.WorkerPool <- w.JobChannel` Block?

Now, based on the dispatcher’s `WorkerPool` being a **buffered channel**, here’s how the line `w.WorkerPool <- w.JobChannel` behaves:

- **Buffered Channel**: The `WorkerPool` is a buffered channel (`make(chan chan Job, maxWorkers)`). If the buffer is **not full**, the line `w.WorkerPool <- w.JobChannel` will **not block**. The worker can register itself immediately because there’s space in the pool for idle workers.
- However, if the **buffer is full** (i.e., all workers are currently idle and their channels are in the pool), the worker would block at this line until the dispatcher picks up a worker from the pool.

### Conclusion

- Workers register themselves in the `WorkerPool` via `w.WorkerPool <- w.JobChannel`. Whether this line blocks depends on the state of the buffered `WorkerPool`. If there’s space, it executes immediately; otherwise, it waits.
- The dispatcher assigns jobs to available workers by waiting for a worker’s job channel from `WorkerPool`, then dispatching the job to that worker.
- Workers continue processing jobs in a loop until they receive a quit signal.
