//go:build darwin || linux

package lock

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestReadObservedRejectsNamedPipesWithoutBlocking(t *testing.T) {
	dir := t.TempDir()
	pipePath := filepath.Join(dir, "lock-pipe")
	if err := syscall.Mkfifo(pipePath, 0o600); err != nil {
		t.Fatal(err)
	}
	pipeLink := filepath.Join(dir, "lock-pipe-link")
	if err := os.Symlink(pipePath, pipeLink); err != nil {
		t.Fatal(err)
	}

	for _, path := range []string{pipePath, pipeLink} {
		t.Run(filepath.Base(path), func(t *testing.T) {
			type result struct {
				observed *Lock
				err      error
			}
			readResult := make(chan result, 1)
			go func() {
				observed, err := ReadObserved(path)
				readResult <- result{observed: observed, err: err}
			}()

			select {
			case got := <-readResult:
				if got.err == nil || !strings.Contains(got.err.Error(), "not a regular file") || got.observed != nil {
					t.Fatalf("ReadObserved(named pipe) = (%#v, %v), want non-regular-file error", got.observed, got.err)
				}
			case <-time.After(time.Second):
				writer, err := os.OpenFile(path, os.O_WRONLY, 0)
				if err != nil {
					t.Fatal(err)
				}
				if err := writer.Close(); err != nil {
					t.Fatal(err)
				}
				<-readResult
				t.Fatal("ReadObserved(named pipe) blocked while opening a non-regular file")
			}
		})
	}
}
