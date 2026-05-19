package worker

import (
	"fmt"
	"time"
)

type JobStatus string

const (
	StatusPending    JobStatus = "pending"
	StatusProcessing JobStatus = "processing"
	StatusCompleted  JobStatus = "completed"
	StatusFailed     JobStatus = "failed"
	StatusRetry      JobStatus = "retry"
)

type Job struct {
	ID          string
	EventID     string
	OrderID     string
	To          string
	Subject     string
	Body        string
	Status      JobStatus
	RetryCount  int
	MaxRetries  int // Будет 5
	NextRetryAt time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Error       string
}

func NewJob(eventID, orderID, to, subject, body string, maxRetries int) *Job {
	now := time.Now()
	return &Job{
		ID:          eventID,
		EventID:     eventID,
		OrderID:     orderID,
		To:          to,
		Subject:     subject,
		Body:        body,
		Status:      StatusPending,
		RetryCount:  0,
		MaxRetries:  maxRetries, // Будет 5
		NextRetryAt: now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

func (j *Job) CanRetry() bool {
	return j.RetryCount < j.MaxRetries
}

// Exponential backoff: 2s, 4s, 8s, 16s, 32s
func (j *Job) CalculateNextBackoff() time.Duration {
	baseDelay := 2 * time.Second
	backoff := baseDelay * (1 << j.RetryCount) // 2^retryCount

	// Максимум 1 минута
	maxDelay := 1 * time.Minute
	if backoff > maxDelay {
		backoff = maxDelay
	}

	return backoff
}

func (j *Job) UpdateForRetry(err error) {
	j.RetryCount++
	j.Status = StatusRetry
	j.Error = err.Error()
	j.UpdatedAt = time.Now()

	backoff := j.CalculateNextBackoff()
	j.NextRetryAt = time.Now().Add(backoff)

	// Логируем информацию о ретрае
	fmt.Printf("[Job] Retry %d/%d scheduled in %v for job %s\n",
		j.RetryCount, j.MaxRetries, backoff, j.ID)
}

func (j *Job) MarkCompleted() {
	j.Status = StatusCompleted
	j.UpdatedAt = time.Now()
	j.Error = ""
	fmt.Printf("[Job] ✅ Job %s completed successfully after %d retries\n", j.ID, j.RetryCount)
}

func (j *Job) MarkFailed(err error) {
	j.Status = StatusFailed
	j.Error = err.Error()
	j.UpdatedAt = time.Now()
	fmt.Printf("[Job] ❌ Job %s permanently failed after %d/%d retries\n",
		j.ID, j.RetryCount, j.MaxRetries)
}

func (j *Job) String() string {
	return fmt.Sprintf("Job{ID=%s, OrderID=%s, Status=%s, Retries=%d/%d}",
		j.ID, j.OrderID, j.Status, j.RetryCount, j.MaxRetries)
}
