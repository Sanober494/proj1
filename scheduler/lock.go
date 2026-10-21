package scheduler

import (
	"runtime"
	"sync/atomic"
)

// TASLock is a test-and-set spin lock.
type TASLock struct {
	state int32 // 0 = unlocked, 1 = locked
}

// Lock spins until the lock has been acquired.
func (l *TASLock) Lock() {
	for atomic.SwapInt32(&l.state, 1) == 1 {
		runtime.Gosched() // let the lock holder run if threads are oversubscribed
	}
}

// Unlock releases the lock.
func (l *TASLock) Unlock() {
	atomic.StoreInt32(&l.state, 0)
}
