package agent

import (
	"sync"

	"github.com/fantods/yaah/internal/message"
)

type QueueMode int

const (
	QueueModeFollowUp QueueMode = iota
	QueueModeSteering
)

type pendingEntry struct {
	mode QueueMode
	msg  message.Message
}

type PendingMessageQueue struct {
	mu      sync.Mutex
	entries []pendingEntry
}

func NewPendingMessageQueue() *PendingMessageQueue {
	return &PendingMessageQueue{
		entries: []pendingEntry{},
	}
}

func (q *PendingMessageQueue) Enqueue(mode QueueMode, msg message.Message) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.entries = append(q.entries, pendingEntry{mode: mode, msg: msg})
}

func (q *PendingMessageQueue) Dequeue() (QueueMode, message.Message, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.entries) == 0 {
		return QueueModeFollowUp, nil, false
	}
	entry := q.entries[0]
	q.entries = q.entries[1:]
	return entry.mode, entry.msg, true
}

func (q *PendingMessageQueue) DequeueByMode(mode QueueMode) (message.Message, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	for i, entry := range q.entries {
		if entry.mode == mode {
			q.entries = append(q.entries[:i], q.entries[i+1:]...)
			return entry.msg, true
		}
	}
	return nil, false
}

func (q *PendingMessageQueue) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.entries)
}

func (q *PendingMessageQueue) Clear() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.entries = []pendingEntry{}
}
