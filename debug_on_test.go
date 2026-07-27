//go:build wasmedge_debug

package wasmedge

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

const (
	debugLeakHelperModeEnv = "WASMEDGE_GO_DEBUG_LEAK_HELPER_MODE"
	debugLeakHelperLogEnv  = "WASMEDGE_GO_DEBUG_LEAK_HELPER_LOG"

	debugStatisticsLeakLine = "wasmedge: leak: Statistics garbage-collected without Close"
	debugStoreLeakLine      = "wasmedge: leak: Store garbage-collected without Close"

	debugLeakCount    = 4
	debugClosedCount  = 8
	debugCopyCount    = 8
	debugControlCount = 4
)

// TestDebugLeakDiagnostics uses subprocesses because its positive controls
// deliberately abandon native resources. Subprocess isolation also prevents
// asynchronous diagnostics from contaminating unrelated tests.
func TestDebugLeakDiagnostics(t *testing.T) {
	for _, mode := range []string{"leak", "closed", "copied"} {
		t.Run(mode, func(t *testing.T) {
			log := runDebugLeakSubprocess(t, mode)
			switch mode {
			case "leak":
				assertDebugLeakLog(t, log, debugLeakCount, 0)
			case "closed":
				assertDebugLeakLog(t, log, 0, debugControlCount)
			case "copied":
				assertDebugLeakLog(t, log, 0, 2*debugControlCount)
			}
		})
	}
}

func runDebugLeakSubprocess(t *testing.T, mode string) string {
	t.Helper()

	logPath := filepath.Join(t.TempDir(), "stderr.log")
	logFile, err := os.Create(logPath)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(
		ctx,
		os.Args[0],
		"-test.run=^TestDebugLeakHelper$",
		"-test.count=1",
	)
	cmd.Env = append(
		os.Environ(),
		debugLeakHelperModeEnv+"="+mode,
		debugLeakHelperLogEnv+"="+logPath,
	)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = logFile
	runErr := cmd.Run()
	closeErr := logFile.Close()

	logBytes, readErr := os.ReadFile(logPath)
	if ctx.Err() != nil {
		t.Fatalf(
			"debug leak helper %q timed out: %v\nstdout:\n%s\nstderr:\n%s",
			mode, ctx.Err(), stdout.String(), logBytes,
		)
	}
	if runErr != nil {
		t.Fatalf(
			"debug leak helper %q failed: %v\nstdout:\n%s\nstderr:\n%s",
			mode, runErr, stdout.String(), logBytes,
		)
	}
	if closeErr != nil {
		t.Fatalf("close debug leak log for %q: %v", mode, closeErr)
	}
	if readErr != nil {
		t.Fatalf("read debug leak log for %q: %v", mode, readErr)
	}
	return string(logBytes)
}

// TestDebugLeakHelper is selected directly by TestDebugLeakDiagnostics.
// Running it normally skips so the parent test cannot recurse accidentally.
func TestDebugLeakHelper(t *testing.T) {
	mode := os.Getenv(debugLeakHelperModeEnv)
	if mode == "" {
		t.Skip("subprocess helper")
	}
	logPath := os.Getenv(debugLeakHelperLogEnv)
	if logPath == "" {
		t.Fatal("debug leak helper log path is empty")
	}

	switch mode {
	case "leak":
		for range debugLeakCount {
			abandonDebugStatistics()
		}
		waitForDebugLeakCount(
			t, logPath, debugStatisticsLeakLine, debugLeakCount,
		)
		runDebugCleanupProbeRounds(t, 2)
		assertNoExtraDebugLeaks(
			t, logPath, debugStatisticsLeakLine, debugLeakCount,
		)
	case "closed":
		for range debugClosedCount {
			if err := closeDebugStatistics(); err != nil {
				t.Fatal(err)
			}
		}
		for range debugControlCount {
			abandonDebugStore()
		}
		waitForDebugLeakCount(t, logPath, debugStoreLeakLine, debugControlCount)
		runDebugCleanupProbeRounds(t, 3)
		assertNoDebugLeaks(t, logPath, debugStatisticsLeakLine)
	case "copied":
		copies := make([]Statistics, debugCopyCount)
		for i := range copies {
			copies[i] = makeDebugStatisticsCopy()
		}

		for range debugControlCount {
			abandonDebugStore()
		}
		waitForDebugLeakCount(t, logPath, debugStoreLeakLine, debugControlCount)
		runDebugCleanupProbeRounds(t, 3)
		assertNoDebugLeaks(t, logPath, debugStatisticsLeakLine)
		for i := range copies {
			_ = copies[i].InstrCount()
		}
		// A local variable may become unreachable after its last ordinary use.
		// Keep every copied wrapper alive across all preceding forced GCs.
		runtime.KeepAlive(copies)

		for i := range copies {
			if err := copies[i].Close(); err != nil {
				t.Fatal(err)
			}
		}
		copies = nil

		for range debugControlCount {
			abandonDebugStore()
		}
		waitForDebugLeakCount(
			t, logPath, debugStoreLeakLine, 2*debugControlCount,
		)
		runDebugCleanupProbeRounds(t, 3)
		assertNoDebugLeaks(t, logPath, debugStatisticsLeakLine)
	default:
		t.Fatalf("unknown debug leak helper mode %q", mode)
	}
}

//go:noinline
func abandonDebugStatistics() {
	statistics := NewStatistics()
	runtime.KeepAlive(statistics)
}

//go:noinline
func closeDebugStatistics() error {
	statistics := NewStatistics()
	err := statistics.Close()
	runtime.KeepAlive(statistics)
	return err
}

//go:noinline
func makeDebugStatisticsCopy() Statistics {
	statistics := NewStatistics()
	copied := *statistics
	runtime.KeepAlive(statistics)
	return copied
}

//go:noinline
func abandonDebugStore() {
	store := NewStore()
	runtime.KeepAlive(store)
}

type debugCleanupProbe struct {
	reachable *byte
	padding   [32]byte
}

func signalDebugCleanupProbe(done chan struct{}) {
	done <- struct{}{}
}

//go:noinline
func abandonDebugCleanupProbe(done chan struct{}) {
	probe := &debugCleanupProbe{reachable: new(byte)}
	runtime.AddCleanup(probe, signalDebugCleanupProbe, done)
	runtime.KeepAlive(probe)
}

func runDebugCleanupProbeRounds(t *testing.T, rounds int) {
	t.Helper()
	for range rounds {
		done := make(chan struct{}, 1)
		abandonDebugCleanupProbe(done)
		deadline := time.Now().Add(10 * time.Second)
		for {
			runtime.GC()
			select {
			case <-done:
				goto nextRound
			default:
			}
			if time.Now().After(deadline) {
				t.Fatal("timed out waiting for runtime cleanup probe")
			}
			runtime.Gosched()
			time.Sleep(10 * time.Millisecond)
		}
	nextRound:
	}
}

func waitForDebugLeakCount(
	t *testing.T,
	logPath, line string,
	want int,
) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for {
		runtime.GC()
		log := readDebugLeakLog(t, logPath)
		if strings.Count(log, line) >= want {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf(
				"timed out waiting for %d %q diagnostics; log:\n%s",
				want, line, log,
			)
		}
		runtime.Gosched()
		time.Sleep(10 * time.Millisecond)
	}
}

func assertNoDebugLeaks(t *testing.T, logPath, line string) {
	t.Helper()
	log := readDebugLeakLog(t, logPath)
	if got := strings.Count(log, line); got != 0 {
		t.Fatalf("got %d unexpected %q diagnostics; log:\n%s", got, line, log)
	}
}

func assertNoExtraDebugLeaks(
	t *testing.T,
	logPath, line string,
	want int,
) {
	t.Helper()
	log := readDebugLeakLog(t, logPath)
	if got := strings.Count(log, line); got != want {
		t.Fatalf("got %d %q diagnostics, want %d; log:\n%s", got, line, want, log)
	}
}

func readDebugLeakLog(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read debug leak log: %v", err)
	}
	return string(data)
}

func assertDebugLeakLog(
	t *testing.T,
	log string,
	wantStatistics, wantStores int,
) {
	t.Helper()
	gotStatistics := 0
	gotStores := 0
	for _, line := range strings.Split(strings.TrimSpace(log), "\n") {
		switch line {
		case "":
		case debugStatisticsLeakLine:
			gotStatistics++
		case debugStoreLeakLine:
			gotStores++
		default:
			t.Fatalf("unexpected debug helper stderr line %q; full log:\n%s", line, log)
		}
	}
	if gotStatistics != wantStatistics || gotStores != wantStores {
		t.Fatalf(
			"debug leak counts: Statistics=%d Store=%d, want Statistics=%d Store=%d; log:\n%s",
			gotStatistics, gotStores, wantStatistics, wantStores, log,
		)
	}
}
