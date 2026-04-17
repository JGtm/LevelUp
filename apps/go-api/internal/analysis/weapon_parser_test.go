package analysis

import "testing"

func TestFindChunkAtTime_Empty(t *testing.T) {
	got := FindChunkAtTime(nil, nil, 1000)
	if got != 0 {
		t.Errorf("FindChunkAtTime(nil) = %d, want 0", got)
	}
}

func TestFindChunkAtTime_SingleChunk(t *testing.T) {
	chunks := []int{0}
	timing := []ChunkTiming{{StartMS: 0, EndMS: 5000}}
	got := FindChunkAtTime(chunks, timing, 2500)
	if got != 0 {
		t.Errorf("FindChunkAtTime = %d, want 0", got)
	}
}

func TestFindChunkAtTime_MultipleChunks(t *testing.T) {
	chunks := []int{0, 1, 2}
	timing := []ChunkTiming{
		{StartMS: 0, EndMS: 5000},
		{StartMS: 5000, EndMS: 10000},
		{StartMS: 10000, EndMS: 15000},
	}
	got := FindChunkAtTime(chunks, timing, 7500)
	if got != 1 {
		t.Errorf("FindChunkAtTime = %d, want 1", got)
	}
}

func TestFindChunkAtTime_BeyondRange(t *testing.T) {
	chunks := []int{0, 1}
	timing := []ChunkTiming{
		{StartMS: 0, EndMS: 5000},
		{StartMS: 5000, EndMS: 10000},
	}
	got := FindChunkAtTime(chunks, timing, 20000)
	if got != 1 {
		t.Errorf("FindChunkAtTime beyond range = %d, want last chunk 1", got)
	}
}

func TestBuildWeaponTimelines_Empty(t *testing.T) {
	wt, indices := BuildWeaponTimelines(nil)
	if len(indices) != 0 {
		t.Errorf("expected 0 indices, got %d", len(indices))
	}
	if len(wt.Timeline) != 0 {
		t.Errorf("expected empty timeline, got %d", len(wt.Timeline))
	}
}

func TestBuildWeaponTimelines_EmptyChunk(t *testing.T) {
	chunks := map[int]ChunkData{
		0: {Data: []byte{}, StartMS: 0, DurationMS: 5000},
	}
	wt, indices := BuildWeaponTimelines(chunks)
	if len(indices) != 1 {
		t.Errorf("expected 1 index, got %d", len(indices))
	}
	if len(wt.Timing) != 1 {
		t.Errorf("expected 1 timing entry, got %d", len(wt.Timing))
	}
}

func TestScanFireEventsAll_Empty(t *testing.T) {
	events := ScanFireEventsAll(nil, 0, 5000)
	if len(events) != 0 {
		t.Errorf("expected 0 events, got %d", len(events))
	}
}

func TestScanFireEvents_Empty(t *testing.T) {
	events := ScanFireEvents(nil, 0, 0, 5000)
	if len(events) != 0 {
		t.Errorf("expected 0 events, got %d", len(events))
	}
}
