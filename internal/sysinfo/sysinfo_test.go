package sysinfo

import (
	"testing"
)

func TestSnapshot(t *testing.T) {
	snap, err := Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() failed: %v", err)
	}

	// CPU load averages should be non-negative
	if snap.LoadAvg1 < 0 || snap.LoadAvg5 < 0 || snap.LoadAvg15 < 0 {
		t.Errorf("load averages should be non-negative: %.2f %.2f %.2f",
			snap.LoadAvg1, snap.LoadAvg5, snap.LoadAvg15)
	}

	// Memory should be positive
	if snap.MemTotalBytes <= 0 {
		t.Errorf("expected positive total memory, got %d", snap.MemTotalBytes)
	}
	if snap.MemUsedBytes <= 0 {
		t.Errorf("expected positive used memory, got %d", snap.MemUsedBytes)
	}
	if snap.MemUsedBytes > snap.MemTotalBytes {
		t.Errorf("used memory (%d) exceeds total (%d)", snap.MemUsedBytes, snap.MemTotalBytes)
	}

	// Disk should be positive
	if snap.DiskTotalBytes <= 0 {
		t.Errorf("expected positive disk total, got %d", snap.DiskTotalBytes)
	}
	if snap.DiskUsedBytes <= 0 {
		t.Errorf("expected positive disk used, got %d", snap.DiskUsedBytes)
	}
	if snap.DiskAvailBytes <= 0 {
		t.Errorf("expected positive disk avail, got %d", snap.DiskAvailBytes)
	}
}
