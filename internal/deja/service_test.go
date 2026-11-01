package deja

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestServiceSnapshotCachesOnce(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	svc := NewService("/workspace", func(_ context.Context, dir string, args ...string) (string, error) {
		calls.Add(1)
		require.Equal(t, "/workspace", dir)
		require.Equal(t, []string{"orient"}, args)
		return "project convention: use go test", nil
	})

	first := svc.Snapshot(context.Background())
	second := svc.Snapshot(context.Background())

	require.Contains(t, first, MemoryInstruction)
	require.Contains(t, first, "project convention: use go test")
	require.Equal(t, first, second)
	require.Equal(t, int32(1), calls.Load())
}

func TestServiceRecallSkipsTrivialAndCachesTopic(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	svc := NewService("/workspace", func(_ context.Context, _ string, args ...string) (string, error) {
		calls.Add(1)
		require.Equal(t, "recall", args[0])
		return "matched session 01a00feb: fixed timeout in coordinator.go", nil
	})

	require.Empty(t, svc.Recall(context.Background(), "ok"))
	require.Empty(t, svc.Recall(context.Background(), "ship it"))
	require.Zero(t, calls.Load())

	got1 := svc.Recall(context.Background(), "where is the coordinator timeout configured?")
	got2 := svc.Recall(context.Background(), "where is the coordinator timeout configured?")
	require.Contains(t, got1, "matched session 01a00feb")
	require.Equal(t, got1, got2)
	require.Equal(t, int32(1), calls.Load())

	got3 := svc.Recall(context.Background(), "how does the SQLite migration lock work?")
	require.Contains(t, got3, "matched session 01a00feb")
	require.Equal(t, int32(2), calls.Load())
}

func TestServiceDegradesCleanlyOnError(t *testing.T) {
	t.Parallel()

	svc := NewService("/workspace", func(_ context.Context, _ string, _ ...string) (string, error) {
		return "", errors.New("deja not indexed")
	})

	require.Empty(t, svc.Snapshot(context.Background()))
	require.Empty(t, svc.Recall(context.Background(), "investigate database lock contention"))
}

func TestServiceClampsValidUTF8(t *testing.T) {
	t.Parallel()

	long := strings.Repeat("абвг", 500)
	svc := NewService("/workspace", func(_ context.Context, _ string, _ ...string) (string, error) {
		return long, nil
	})

	snap := svc.Snapshot(context.Background())
	require.True(t, strings.HasSuffix(snap, "..."))
}
