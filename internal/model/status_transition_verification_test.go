package model

import (
	"sync"
	"testing"
)

func TestProcessingCannotCloseDirectlyUnderConcurrentRequests(t *testing.T) {
	start := make(chan struct{}); var wg sync.WaitGroup; accepted := make(chan bool, 2)
	for i:=0; i<2; i++ { wg.Add(1); go func(){ defer wg.Done(); <-start; accepted <- LegalTransition(StatusProcessing, StatusClosed) }() }
	close(start); wg.Wait(); close(accepted)
	for ok := range accepted { if ok { t.Fatal("processing status bypassed completion and closed directly") } }
}

func TestCompletedStatusCanCloseThroughValidTransition(t *testing.T) {
	if !LegalTransition(StatusDone, StatusClosed) {
		t.Fatal("completed status should be allowed to close")
	}
}
