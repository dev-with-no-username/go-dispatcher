package dispatcher

import (
	"os"
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestDispatcher(t *testing.T) {
	t.Run("simpleTest", simpleTest)
	t.Run("overloadTest", overloadTest)
}

func simpleTest(t *testing.T) {
	os.Setenv("MAX_WORKERS", "10")

	// Initialize the JobQueue
	JobQueue = make(chan Job, 10)

	// Number of workers and jobs
	maxWorkers := 10
	numJobs := 2000

	// Create the dispatcher with workers
	dispatcher := NewDispatcher(maxWorkers)

	// Run the dispatcher
	dispatcher.Run()

	// WaitGroup to wait for all jobs to finish processing
	var wg sync.WaitGroup
	wg.Add(numJobs)

	// Submit jobs to the JobQueue
	for i := 0; i < numJobs; i++ {
		go func() {
			t.Logf("numJob %d", i)
			job := Job{
				Payload: Payload{
					Name: "job_" + strconv.Itoa(i),
				},
			}
			JobQueue <- job

			time.Sleep(time.Second)
			wg.Done()
		}()
	}

	wg.Wait()

	// Close channels
	close(JobQueue)

	// Ensure all jobs were processed
	if len(JobQueue) != 0 {
		t.Errorf("Expected JobQueue to be empty, but got %d jobs left", len(JobQueue))
	}
}

func overloadTest(t *testing.T) {
	os.Setenv("MAX_WORKERS", "100")

	// Initialize the JobQueue
	JobQueue = make(chan Job, 100)

	// Number of workers and jobs
	maxWorkers := 100
	numJobs := 2000000

	// Create the dispatcher with workers
	dispatcher := NewDispatcher(maxWorkers)

	// Run the dispatcher
	dispatcher.Run()

	// WaitGroup to wait for all jobs to finish processing
	var wg sync.WaitGroup
	wg.Add(numJobs)

	// Submit jobs to the JobQueue
	for i := 0; i < numJobs; i++ {
		go func() {
			t.Logf("numJob %d", i)
			job := Job{
				Payload: Payload{
					Name: "job_" + strconv.Itoa(i),
				},
			}
			JobQueue <- job

			time.Sleep(time.Second)
			wg.Done()
		}()
	}

	wg.Wait()

	// Close channels
	close(JobQueue)

	// Ensure all jobs were processed
	if len(JobQueue) != 0 {
		t.Errorf("Expected JobQueue to be empty, but got %d jobs left", len(JobQueue))
	}
}
