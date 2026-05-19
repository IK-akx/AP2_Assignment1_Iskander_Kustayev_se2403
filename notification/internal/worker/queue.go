package worker

import (
	"container/heap"
	"log"
	"sync"
	"time"
)

type JobQueue struct {
	mu       sync.Mutex
	queue    priorityQueue
	cond     *sync.Cond
	maxSize  int // Size of queue
	sizeCond *sync.Cond
}

func NewJobQueue(maxSize int) *JobQueue {
	q := &JobQueue{
		queue:   make(priorityQueue, 0),
		maxSize: maxSize,
	}
	q.cond = sync.NewCond(&q.mu)
	q.sizeCond = sync.NewCond(&q.mu)
	heap.Init(&q.queue)
	return q
}

func (q *JobQueue) Push(job *Job) {
	q.mu.Lock()
	defer q.mu.Unlock()

	for q.queue.Len() >= q.maxSize {
		log.Printf("[Queue] Queue is full (size=%d, max=%d), waiting...", q.queue.Len(), q.maxSize)
		q.sizeCond.Wait()
	}

	heap.Push(&q.queue, job)
	log.Printf("[Queue] Job %s added to queue (size=%d/%d, status=%s, retry=%d/%d)",
		job.ID, q.queue.Len(), q.maxSize, job.Status, job.RetryCount, job.MaxRetries)

	q.cond.Signal()
}

func (q *JobQueue) Pop() *Job {
	q.mu.Lock()
	defer q.mu.Unlock()

	for {
		for q.queue.Len() == 0 {
			q.cond.Wait()
		}

		nextJob := q.queue[0]

		if time.Now().Before(nextJob.NextRetryAt) {
			waitDuration := time.Until(nextJob.NextRetryAt)
			log.Printf("[Queue] Job %s not ready, waiting %v", nextJob.ID, waitDuration)

			timer := time.NewTimer(waitDuration)
			q.mu.Unlock()
			<-timer.C
			q.mu.Lock()
			continue
		}

		job := heap.Pop(&q.queue).(*Job)

		q.sizeCond.Signal()

		return job
	}
}

func (q *JobQueue) Remove(jobID string) bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	for i, job := range q.queue {
		if job.ID == jobID {
			heap.Remove(&q.queue, i)
			log.Printf("[Queue] Job %s removed from queue (size=%d/%d)", jobID, q.queue.Len(), q.maxSize)
			q.sizeCond.Signal()
			return true
		}
	}
	return false
}

func (q *JobQueue) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.queue.Len()
}

func (q *JobQueue) Cap() int {
	return q.maxSize
}

// priorityQueue implements heap.Interface
type priorityQueue []*Job

func (pq priorityQueue) Len() int { return len(pq) }

func (pq priorityQueue) Less(i, j int) bool {
	return pq[i].NextRetryAt.Before(pq[j].NextRetryAt)
}

func (pq priorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
}

func (pq *priorityQueue) Push(x interface{}) {
	job := x.(*Job)
	*pq = append(*pq, job)
}

func (pq *priorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	job := old[n-1]
	old[n-1] = nil
	*pq = old[0 : n-1]
	return job
}
