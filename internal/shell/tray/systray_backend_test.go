package tray

import "testing"

func TestBackendStopsWhenReadyCallbackStopsDuringStartup(t *testing.T) {
	originalRun := runWithExternalLoop
	originalSetOnTapped := setOnTapped
	defer func() {
		runWithExternalLoop = originalRun
		setOnTapped = originalSetOnTapped
	}()

	var startCalls, endCalls int
	runWithExternalLoop = func(onReady, _ func()) (func(), func()) {
		onReady()
		return func() { startCalls++ }, func() { endCalls++ }
	}
	setOnTapped = func(func()) {}

	backend := &systrayBackend{}
	if err := backend.Start(BackendCallbacks{Ready: backend.Stop}); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if startCalls != 1 {
		t.Fatalf("start calls = %d, want 1", startCalls)
	}
	if endCalls != 1 {
		t.Fatalf("end calls = %d, want 1", endCalls)
	}
}
