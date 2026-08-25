package vision

import "testing"

func TestBatchShouldYield(t *testing.T) {
	if !batchShouldYield(2, 100) {
		t.Fatal("batch should yield while interactives wait inside window")
	}
	if batchShouldYield(0, 100) {
		t.Fatal("batch must not yield with no interactive waiters")
	}
	old := BatchStarvationMs
	BatchStarvationMs = 300
	defer func() { BatchStarvationMs = old }()
	if batchShouldYield(3, 400) {
		t.Fatal("starvation window must release batch")
	}
}

func TestPrioName(t *testing.T) {
	if prioName(true) != "interactive" || prioName(false) != "batch" {
		t.Fatal("prioName broken")
	}
}
