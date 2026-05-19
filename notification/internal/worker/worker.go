package worker

import (
	"log"
	"sync"
	"time"

	"notification/internal/email"
	"notification/internal/idempotency"
)

type WorkerPool struct {
	numWorkers    int
	jobQueue      *JobQueue
	emailSender   email.EmailSender
	idempotency   *idempotency.RedisStore
	stopCh        chan struct{}
	wg            sync.WaitGroup
	activeWorkers map[int]bool
	mu            sync.RWMutex
}

func NewWorkerPool(
	numWorkers int,
	jobQueue *JobQueue,
	emailSender email.EmailSender,
	idempotencyStore *idempotency.RedisStore,
) *WorkerPool {
	return &WorkerPool{
		numWorkers:    numWorkers,
		jobQueue:      jobQueue,
		emailSender:   emailSender,
		idempotency:   idempotencyStore,
		stopCh:        make(chan struct{}),
		activeWorkers: make(map[int]bool),
	}
}

func (wp *WorkerPool) Start() {
	log.Printf("[WorkerPool] 🚀 Starting %d workers (parallel processing)...", wp.numWorkers)

	for i := 0; i < wp.numWorkers; i++ {
		wp.wg.Add(1)
		go wp.worker(i)
	}

	go wp.monitorWorkers()

	log.Printf("[WorkerPool] ✅ All %d workers started and ready", wp.numWorkers)
}

func (wp *WorkerPool) monitorWorkers() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		wp.mu.RLock()
		activeCount := len(wp.activeWorkers)
		queueSize := wp.jobQueue.Len()
		wp.mu.RUnlock()

		if activeCount > 0 || queueSize > 0 {
			log.Printf("[WorkerPool] 📊 Status: %d active workers, %d jobs in queue (max=%d)",
				activeCount, queueSize, wp.jobQueue.Cap())
		}
	}
}

func (wp *WorkerPool) worker(id int) {
	defer wp.wg.Done()

	log.Printf("[Worker %d] 🔧 Started (ready to process jobs)", id)

	for {
		select {
		case <-wp.stopCh:
			log.Printf("[Worker %d] 🛑 Stopping", id)
			wp.markWorkerInactive(id)
			return
		default:
			wp.markWorkerActive(id)

			// Get next job (blocks until job is ready)
			job := wp.jobQueue.Pop()

			log.Printf("[Worker %d] 📨 Processing job: %s (queue size: %d)",
				id, job.ID, wp.jobQueue.Len())

			// Process the job
			wp.processJob(job, id)
		}
	}
}

func (wp *WorkerPool) markWorkerActive(id int) {
	wp.mu.Lock()
	defer wp.mu.Unlock()
	wp.activeWorkers[id] = true
}

func (wp *WorkerPool) markWorkerInactive(id int) {
	wp.mu.Lock()
	defer wp.mu.Unlock()
	delete(wp.activeWorkers, id)
}

func (wp *WorkerPool) processJob(job *Job, workerID int) {
	startTime := time.Now()

	// Check idempotency
	if wp.idempotency != nil {
		processed, err := wp.idempotency.IsProcessed(job.EventID)
		if err != nil {
			log.Printf("[Worker %d] ⚠️ Failed to check idempotency for %s: %v", workerID, job.ID, err)
			job.UpdateForRetry(err)
			wp.jobQueue.Push(job)
			return
		}

		if processed {
			log.Printf("[Worker %d] 🔄 Job %s already processed (idempotency), skipping", workerID, job.ID)
			wp.jobQueue.Remove(job.ID)
			return
		}
	}

	// Attempt to send email
	log.Printf("[Worker %d] 📧 Sending email for job %s to %s (attempt %d/%d)",
		workerID, job.ID, job.To, job.RetryCount+1, job.MaxRetries)

	err := wp.emailSender.Send(job.To, job.Subject, job.Body)

	duration := time.Since(startTime)

	if err != nil {
		log.Printf("[Worker %d] ❌ Failed to send email for job %s: %v (took %v)",
			workerID, job.ID, err, duration)

		if job.CanRetry() {
			job.UpdateForRetry(err)
			log.Printf("[Worker %d] 🔁 Job %s scheduled for retry %d/%d in %v",
				workerID, job.ID, job.RetryCount, job.MaxRetries,
				time.Until(job.NextRetryAt))

			wp.jobQueue.Push(job)
		} else {
			job.MarkFailed(err)
			log.Printf("[Worker %d] 💀 Job %s permanently failed after %d retries",
				workerID, job.ID, job.RetryCount)

			wp.jobQueue.Remove(job.ID)
			wp.logDeadLetter(job)
		}
		return
	}

	// Success!
	log.Printf("[Worker %d] ✅ SUCCESS! Email sent for job %s (took %v, after %d retries)",
		workerID, job.ID, duration, job.RetryCount)

	if wp.idempotency != nil {
		if err := wp.idempotency.MarkProcessed(job.EventID); err != nil {
			log.Printf("[Worker %d] ⚠️ Failed to mark job %s as processed: %v", workerID, job.ID, err)
		}
	}

	job.MarkCompleted()
	wp.jobQueue.Remove(job.ID)
}

func (wp *WorkerPool) logDeadLetter(job *Job) {
	log.Printf("[DeadLetter] Permanently failed job:")
	log.Printf("  ID: %s", job.ID)
	log.Printf("  OrderID: %s", job.OrderID)
	log.Printf("  To: %s", job.To)
	log.Printf("  Retries: %d/%d", job.RetryCount, job.MaxRetries)
	log.Printf("  Last Error: %s", job.Error)
	log.Printf("  Created: %v", job.CreatedAt)
	log.Printf("  Failed: %v", job.UpdatedAt)
}

func (wp *WorkerPool) Stop() {
	log.Printf("[WorkerPool] 🛑 Stopping all workers...")
	close(wp.stopCh)
	wp.wg.Wait()
	log.Printf("[WorkerPool] ✅ All workers stopped")
}
