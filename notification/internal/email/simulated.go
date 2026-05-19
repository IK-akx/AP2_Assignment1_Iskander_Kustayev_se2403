package email

import (
	"fmt"
	"log"
	"math/rand"
	"time"
)

type SimulatedEmailSender struct {
	delayMs     int
	failureRate float64
}

func NewSimulatedEmailSender(delayMs int, failureRate float64) *SimulatedEmailSender {
	if failureRate < 0 {
		failureRate = 0
	}
	if failureRate > 1 {
		failureRate = 1
	}

	return &SimulatedEmailSender{
		delayMs:     delayMs,
		failureRate: failureRate,
	}
}

func (s *SimulatedEmailSender) Send(to, subject, body string) error {
	log.Printf("[Simulated] Attempting to send email to %s", to)
	log.Printf("[Simulated] Subject: %s", subject)

	if s.delayMs > 0 {
		log.Printf("[Simulated] Simulating network delay: %d ms", s.delayMs)
		time.Sleep(time.Duration(s.delayMs) * time.Millisecond)
	}

	// 80% failure rate
	if s.failureRate > 0 {
		rand.Seed(time.Now().UnixNano())
		randomValue := rand.Float64()

		if randomValue < s.failureRate {
			log.Printf("[Simulated] ❌ FAILURE! (failure rate: %.0f%%)", s.failureRate*100)
			log.Printf("[Simulated] Error: random network failure - connection timeout")
			return fmt.Errorf("simulated provider error: random network failure (rate: %.0f%%)", s.failureRate*100)
		}
	}

	// Success (20%)
	log.Printf("[Simulated] ✅ SUCCESS! Email sent to %s", to)
	return nil
}

func (s *SimulatedEmailSender) GetProviderName() string {
	return fmt.Sprintf("Simulated(delay=%dms,failureRate=%.0f%%)", s.delayMs, s.failureRate*100)
}
