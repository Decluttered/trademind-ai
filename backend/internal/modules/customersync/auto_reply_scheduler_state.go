package customersync

import "sync"

var (
	autoReplySchedulerMu      sync.Mutex
	autoReplySchedulerRunning bool
)

func setAutoReplyPollingSchedulerRunning(v bool) {
	autoReplySchedulerMu.Lock()
	autoReplySchedulerRunning = v
	autoReplySchedulerMu.Unlock()
}

// AutoReplyPollingSchedulerRunning reports whether this process owns the polling loop.
func AutoReplyPollingSchedulerRunning() bool {
	autoReplySchedulerMu.Lock()
	defer autoReplySchedulerMu.Unlock()
	return autoReplySchedulerRunning
}
