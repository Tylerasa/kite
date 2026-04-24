package services

import (
	"crypto/sha256"
	"fmt"
	"sync"
	"time"

	"github.com/kite/internal/domain/exceptions"
)

const lockExpiry = 30 * time.Second

// TransactionGuard prevents duplicate financial operations within a 30-second window.
// It is keyed by a SHA-256 fingerprint of the request parameters so that identical
// requests from double-clicks or network retries are rejected without hitting the DB.
// Expired locks are cleaned up on every check call (same pattern as Cinnamon).
type TransactionGuard struct {
	mu    sync.Mutex
	locks map[[32]byte]time.Time
}

func NewTransactionGuard() *TransactionGuard {
	return &TransactionGuard{
		locks: make(map[[32]byte]time.Time),
	}
}

// CheckAndLock hashes the provided fields into a fingerprint and checks whether
// an identical operation has been seen in the last 30 seconds.
// Returns ErrDuplicateTransaction if a lock exists; otherwise records the lock.
func (g *TransactionGuard) CheckAndLock(fields ...string) error {
	key := fingerprint(fields...)

	g.mu.Lock()
	defer g.mu.Unlock()

	g.cleanExpired()

	if _, exists := g.locks[key]; exists {
		return exceptions.ErrDuplicateTransaction
	}

	g.locks[key] = time.Now()
	return nil
}

func (g *TransactionGuard) cleanExpired() {
	for k, lockedAt := range g.locks {
		if time.Since(lockedAt) > lockExpiry {
			delete(g.locks, k)
		}
	}
}

func fingerprint(fields ...string) [32]byte {
	h := sha256.New()
	for i, f := range fields {
		if i > 0 {
			h.Write([]byte("|"))
		}
		h.Write([]byte(f))
	}
	var out [32]byte
	copy(out[:], h.Sum(nil))
	return out
}

// PayoutFingerprint builds the canonical key for a payout request.
func PayoutFingerprint(userID, currency string, amount int64, accountNumber, bankCode string) []string {
	return []string{userID, currency, fmt.Sprintf("%d", amount), accountNumber, bankCode}
}
