package stats

import "sync/atomic"

type CoreSnapshot struct {
	GeneratedDocs      uint64
	SuccessfulRequests uint64
	SuccessfulDocs     uint64
	RetryAttempts      uint64
	TerminalFailures   uint64
	DroppedBatches     uint64
}

type CoreStats struct {
	generatedDocs      atomic.Uint64
	successfulRequests atomic.Uint64
	successfulDocs     atomic.Uint64
	retryAttempts      atomic.Uint64
	terminalFailures   atomic.Uint64
	droppedBatches     atomic.Uint64
}

func NewCoreStats() *CoreStats {
	return &CoreStats{}
}

func (c *CoreStats) IncGeneratedDocs(count int) {
	c.generatedDocs.Add(uint64(count))
}

func (c *CoreStats) IncSuccessfulRequest(docCount int) {
	c.successfulRequests.Add(1)
	c.successfulDocs.Add(uint64(docCount))
}

func (c *CoreStats) IncRetryAttempt() {
	c.retryAttempts.Add(1)
}

func (c *CoreStats) IncTerminalFailure() {
	c.terminalFailures.Add(1)
}

func (c *CoreStats) IncDroppedBatch() {
	c.droppedBatches.Add(1)
}

func (c *CoreStats) Snapshot() CoreSnapshot {
	return CoreSnapshot{
		GeneratedDocs:      c.generatedDocs.Load(),
		SuccessfulRequests: c.successfulRequests.Load(),
		SuccessfulDocs:     c.successfulDocs.Load(),
		RetryAttempts:      c.retryAttempts.Load(),
		TerminalFailures:   c.terminalFailures.Load(),
		DroppedBatches:     c.droppedBatches.Load(),
	}
}
