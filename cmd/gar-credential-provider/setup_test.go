package main

import (
	"context"
	"testing"
	"time"

	"github.com/mondu-ai/gar-credential-provider/internal/nodesetup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _ nodesetup.Installer = (*mockInstaller)(nil)

type mockInstaller struct {
	installResult nodesetup.InstallResult
	installErr    error
	restartCalled bool
	restartErr    error
}

func (m *mockInstaller) Install() (nodesetup.InstallResult, error) {
	return m.installResult, m.installErr
}

func (m *mockInstaller) RestartKubelet() error {
	m.restartCalled = true
	return m.restartErr
}

func TestRunSetupWithInstaller(t *testing.T) {
	tests := []struct {
		name        string
		result      nodesetup.InstallResult
		wantRestart bool
	}{
		{
			name: "changes made triggers kubelet restart",
			result: nodesetup.InstallResult{
				Changed: true,
				Actions: []string{"Copied binary"},
			},
			wantRestart: true,
		},
		{
			name: "no changes skips kubelet restart",
			result: nodesetup.InstallResult{
				Changed: false,
			},
			wantRestart: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockInstaller{installResult: tt.result}

			ctx, cancel := context.WithCancel(context.Background())

			done := make(chan struct{})
			go func() {
				runSetupWithInstaller(ctx, mock)
				close(done)
			}()

			// Give it a moment to reach the signal wait
			time.Sleep(50 * time.Millisecond)
			cancel()

			select {
			case <-done:
			case <-time.After(2 * time.Second):
				t.Fatal("runSetupWithInstaller did not exit after context cancellation")
			}

			assert.Equal(t, tt.wantRestart, mock.restartCalled)
		})
	}
}

func TestRunSetupWithInstaller_BlocksUntilSignal(t *testing.T) {
	mock := &mockInstaller{
		installResult: nodesetup.InstallResult{Changed: false},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		runSetupWithInstaller(ctx, mock)
		close(done)
	}()

	// Verify it's still blocking after setup
	select {
	case <-done:
		t.Fatal("runSetupWithInstaller returned before context was canceled")
	case <-time.After(100 * time.Millisecond):
		// Expected: still running
	}

	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		require.Fail(t, "runSetupWithInstaller did not exit after context cancellation")
	}
}
