package metrices

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func resetStatsForTest() {
	stats = make(map[string]*Stats)
	globalStats = &Stats{MinTime: 999999999}
}

func TestUpdateStats(t *testing.T) {
	resetStatsForTest()

	updateStats("/api", 500000)   // 0.5s
	updateStats("/api", 2000000)  // 2s, slow request
	updateStats("/other", 100000) // 0.1s

	apiStats, ok := stats["/api"]
	if !ok {
		t.Fatalf("expected stats for /api path")
	}

	if apiStats.Count != 2 {
		t.Fatalf("expected /api count 2, got %d", apiStats.Count)
	}
	if apiStats.MaxTime != 2000000 {
		t.Fatalf("expected max 2000000, got %d", apiStats.MaxTime)
	}
	if apiStats.MinTime != 500000 {
		t.Fatalf("expected min 500000, got %d", apiStats.MinTime)
	}
	if apiStats.SlowRequests != 1 {
		t.Fatalf("expected 1 slow request, got %d", apiStats.SlowRequests)
	}

	if globalStats.Count != 3 {
		t.Fatalf("expected global count 3, got %d", globalStats.Count)
	}
	if globalStats.TotalTime != int64(2600000) {
		t.Fatalf("expected global total 2600000, got %d", globalStats.TotalTime)
	}
	if globalStats.SlowRequests != 1 {
		t.Fatalf("expected global slow 1, got %d", globalStats.SlowRequests)
	}
}

func TestPrintStats(t *testing.T) {
	resetStatsForTest()

	stats["/a"] = &Stats{
		Count:     2,
		TotalTime: 600000,
		MaxTime:   400000,
		MinTime:   200000,
	}
	stats["/b"] = &Stats{
		Count:     1,
		TotalTime: 200000,
		MaxTime:   200000,
		MinTime:   200000,
	}
	globalStats = &Stats{
		Count:     3,
		TotalTime: 800000,
		MaxTime:   400000,
		MinTime:   200000,
		// SlowRequests default 0
	}

	output := captureOutput(printStats)

	if !strings.Contains(output, "全体: リクエスト数=3") {
		t.Fatalf("expected global stats in output, got: %s", output)
	}
	if !strings.Contains(output, "/a: 回数=2") {
		t.Fatalf("expected /a stats in output, got: %s", output)
	}
	if len(stats) != 0 {
		t.Fatalf("expected stats reset, got %d entries", len(stats))
	}
	if globalStats.Count != 0 || globalStats.MinTime != 999999999 {
		t.Fatalf("expected global stats reset, got %+v", globalStats)
	}
}

func captureOutput(fn func()) string {
	origStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		panic(err)
	}
	os.Stdout = w

	done := make(chan struct{})
	var buf bytes.Buffer
	go func() {
		_, _ = io.Copy(&buf, r)
		close(done)
	}()

	fn()

	_ = w.Close()
	os.Stdout = origStdout
	<-done
	return buf.String()
}
