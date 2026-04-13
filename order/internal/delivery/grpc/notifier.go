package grpc

import (
	"sync"
)

type Subscriber chan OrderUpdate

type OrderUpdate struct {
	OrderID   string
	OldStatus string
	NewStatus string
	Message   string
}

type Notifier struct {
	subscribers map[string][]Subscriber
	mu          sync.RWMutex
}

func NewNotifier() *Notifier {
	return &Notifier{
		subscribers: make(map[string][]Subscriber),
	}
}

func (n *Notifier) Subscribe(orderID string) Subscriber {
	n.mu.Lock()
	defer n.mu.Unlock()

	ch := make(Subscriber, 1)
	n.subscribers[orderID] = append(n.subscribers[orderID], ch)

	return ch
}

func (n *Notifier) Notify(update OrderUpdate) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	for _, ch := range n.subscribers[update.OrderID] {
		ch <- update
	}
}
