package domain

import "time"

type IdempotencyKey struct {
	Key       string    `gorm:"primaryKey;size:100"`
	OrderID   string    `gorm:"size:50"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

func (IdempotencyKey) TableName() string {
	return "idempotency_keys"
}
