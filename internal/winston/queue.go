package winston

import (
	"context"
	"errors"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"alfred/internal/config"
)

type QueueRunner struct {
	cfg  config.Config
	proc *ImportProcessor
}

func NewQueueRunner(cfg config.Config, proc *ImportProcessor) *QueueRunner {
	return &QueueRunner{cfg: cfg, proc: proc}
}

func (q *QueueRunner) Run(ctx context.Context) error {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		if err := q.runOnce(ctx); err != nil && !errors.Is(err, context.Canceled) {
			log.Printf("winston: queue pass error: %v", err)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (q *QueueRunner) runOnce(ctx context.Context) error {
	items, err := q.listNZBs()
	if err != nil {
		return err
	}
	if len(items) == 0 {
		return nil
	}

	for _, nzb := range items {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if err := q.proc.ImportOne(ctx, nzb); err != nil {
			log.Printf("winston: import item error source=%s err=%v", nzb, err)
			if q.proc.state != nil {
				_ = q.proc.state.Put(nzb, ImportedRecord{RelativePath: "", Status: "error", State: StateFailed, Confidence: ConfidenceLow, Metadata: ItemMetadata{}, Preview: &ItemPreview{SourceNZBPath: nzb, State: StateFailed, Confidence: ConfidenceLow, ProposedPath: "", Reason: err.Error(), ResolverError: err.Error()}})
			}
			continue
		}
	}
	return nil
}

func (q *QueueRunner) ListNZBs() ([]string, error) {
	return q.listNZBs()
}

func (q *QueueRunner) listNZBs() ([]string, error) {
	var out []string
	err := filepath.WalkDir(q.cfg.SourceRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.EqualFold(filepath.Ext(d.Name()), ".nzb") {
			out = append(out, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(out)
	return out, nil
}
