package customerchat

import "sync"

var (
	autoReplyWorkerStateMu sync.Mutex
	autoReplyWorkerCount   int
)

func markAutoReplyWorkerStarted() {
	autoReplyWorkerStateMu.Lock()
	autoReplyWorkerCount++
	autoReplyWorkerStateMu.Unlock()
}

func markAutoReplyWorkerStopped() {
	autoReplyWorkerStateMu.Lock()
	if autoReplyWorkerCount > 0 {
		autoReplyWorkerCount--
	}
	autoReplyWorkerStateMu.Unlock()
}

// AutoReplyWorkersRunning reports whether this process currently has an active consumer.
func AutoReplyWorkersRunning() bool {
	autoReplyWorkerStateMu.Lock()
	defer autoReplyWorkerStateMu.Unlock()
	return autoReplyWorkerCount > 0
}
