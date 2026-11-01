package logging

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
)

// A shared Pretty logger must serialize writes even when the configured writer
// is not safe for concurrent use, as is the case with bytes.Buffer.
func TestPrettyConcurrentWritesAreAtomic(t *testing.T) {
	const workers = 48
	const messages = 25

	var output bytes.Buffer
	logger := NewPretty(&output, false)
	var wg sync.WaitGroup
	for worker := 0; worker < workers; worker++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for i := 0; i < messages; i++ {
				logger.Log(context.Background(), LevelInfo, "RaceTest", fmt.Sprintf("%d-%d", id, i))
			}
		}(worker)
	}
	wg.Wait()

	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != workers*messages {
		t.Fatalf("got %d complete lines, expected %d", len(lines), workers*messages)
	}
	seen := make(map[string]bool, workers*messages)
	for _, line := range lines {
		const prefix = "[RaceTest] "
		index := strings.Index(line, prefix)
		if index < 0 {
			t.Fatalf("corrupted log line %q", line)
		}
		message := line[index+len(prefix):]
		if seen[message] {
			t.Fatalf("duplicated log message %q", message)
		}
		seen[message] = true
	}
	if len(seen) != workers*messages {
		t.Fatalf("observed %d unique messages, expected %d", len(seen), workers*messages)
	}
}
