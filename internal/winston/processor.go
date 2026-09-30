package winston

import (
	"context"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"alfred/internal/config"
	"alfred/internal/plex"
)

type ImportProcessor struct {
	cfg     config.Config
	alt     *AltMountClient
	plex    *plex.Client
	state   *StateStore
	matcher *Matcher
	filebot *FileBotClient
}

func NewImportProcessor(cfg config.Config, alt *AltMountClient, plexClient *plex.Client, state *StateStore) *ImportProcessor {
	return &ImportProcessor{cfg: cfg, alt: alt, plex: plexClient, state: state, matcher: NewMatcher(), filebot: NewFileBotClient(cfg)}
}

func (p *ImportProcessor) Run(ctx context.Context) error {
	log.Printf("winston: bootstrap processor ready, source_root=%s", p.cfg.SourceRoot)
	<-ctx.Done()
	return nil
}

func (p *ImportProcessor) ImportOne(ctx context.Context, sourceNZB string) error {
	return p.importOneInternal(ctx, sourceNZB, false)
}

func (p *ImportProcessor) RetryImport(ctx context.Context, sourceNZB string) error {
	return p.importOneInternal(ctx, sourceNZB, true)
}

func (p *ImportProcessor) importOneInternal(ctx context.Context, sourceNZB string, force bool) error {
	if p.state != nil && !force {
		if rec, ok := p.state.Data.Imported[sourceNZB]; ok {
			if rec.Status == "submitted" || rec.Status == "completed" || rec.Status == "error" || rec.State == StateImported || rec.State == StateImporting || rec.State == StateFailed {
				log.Printf("winston: skipping already seen nzb: %s status=%s state=%s", sourceNZB, rec.Status, rec.State)
				return nil
			}
		}
	}

	preview := p.BuildPreview(sourceNZB, ItemMetadata{})
	if p.state != nil {
		if rec, ok := p.state.Data.Imported[sourceNZB]; ok && rec.Metadata != (ItemMetadata{}) {
			preview = p.BuildPreview(sourceNZB, rec.Metadata)
		}
	}
	relativePath := preview.ProposedPath
	if normalizeKind(preview.Metadata.Kind) == "series" && preview.Metadata.Season > 0 && preview.Metadata.Episode == 0 {
		relativePath = filepath.ToSlash(filepath.Dir(relativePath))
	}

	if !force && preview.State == StateNeedsReview && !(preview.Confidence == ConfidenceMedium && p.cfg.AutoImportMedium) {
		log.Printf("winston: review required for %s proposed=%s reason=%s", sourceNZB, preview.ProposedPath, preview.Reason)
		if p.state != nil {
			_ = p.state.Put(sourceNZB, ImportedRecord{RelativePath: preview.ProposedPath, Status: "review", State: preview.State, Confidence: preview.Confidence, Metadata: preview.Metadata, Preview: preview})
		}
		return nil
	}

	if p.alt == nil || p.alt.baseURL == "" {
		return nil
	}
	resp, err := p.alt.ImportFile(ctx, ManualImportRequest{
		FilePath:     p.stageAltMountNZB(sourceNZB),
		RelativePath: func() *string { if strings.TrimSpace(relativePath) == "" { return nil }; v := strings.TrimSpace(relativePath); return &v }(),
	})
	if err != nil {
		if p.state != nil {
			preview.State = StateFailed
			preview.Reason = err.Error()
			preview.ResolverError = err.Error()
			_ = p.state.Put(sourceNZB, ImportedRecord{
				RelativePath: relativePath,
				Status:       "error",
				State:        StateFailed,
				Confidence:   preview.Confidence,
				Metadata:     preview.Metadata,
				Preview:      preview,
				Reason:       err.Error(),
			})
		}
		return err
	}
	if resp.QueueID <= 0 {
		log.Printf("winston: altmount accepted source=%s with queue_id=%d and no queue tracking, relative_path=%s", sourceNZB, resp.QueueID, relativePath)
		if p.state != nil {
			_ = p.state.Put(sourceNZB, ImportedRecord{QueueID: resp.QueueID, RelativePath: relativePath, Status: "submitted", State: StateImporting, Confidence: preview.Confidence, Metadata: preview.Metadata, Preview: preview})
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(p.cfg.SleepBetweenImports):
			return nil
		}
	}
	log.Printf("winston: imported source=%s queue_id=%d relative_path=%s", sourceNZB, resp.QueueID, relativePath)
	if p.state != nil {
		_ = p.state.Put(sourceNZB, ImportedRecord{QueueID: resp.QueueID, RelativePath: relativePath, Status: "submitted", State: StateImporting, Confidence: preview.Confidence, Metadata: preview.Metadata, Preview: preview})
	}

	item, err := p.waitForQueueReady(ctx, resp.QueueID)
	if err != nil {
		if p.state != nil {
			_ = p.state.Put(sourceNZB, ImportedRecord{QueueID: resp.QueueID, RelativePath: relativePath, Status: "error", State: StateFailed, Confidence: preview.Confidence, Metadata: preview.Metadata, Preview: preview, Reason: err.Error()})
		}
		return err
	}
	if p.state != nil {
		_ = p.state.Put(sourceNZB, ImportedRecord{QueueID: resp.QueueID, RelativePath: item.TargetPath, Status: item.Status, State: StateImported, Confidence: preview.Confidence, Metadata: preview.Metadata, Preview: preview})
	}
	if p.plex != nil {
		_ = p.plex.RefreshPath(item.TargetPath)
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(p.cfg.SleepBetweenImports):
		return nil
	}
}

func (p *ImportProcessor) EnsurePreview(sourceNZB string) (*ItemPreview, error) {
	preview := p.BuildPreview(sourceNZB, ItemMetadata{})
	if p.state != nil {
		if rec, ok := p.state.Data.Imported[sourceNZB]; ok && rec.Metadata != (ItemMetadata{}) {
			preview = p.BuildPreview(sourceNZB, rec.Metadata)
		}
		rec := ImportedRecord{}
		if existing, ok := p.state.Data.Imported[sourceNZB]; ok {
			rec = existing
		}
		rec.RelativePath = preview.ProposedPath
		rec.State = preview.State
		rec.Confidence = preview.Confidence
		rec.Metadata = preview.Metadata
		rec.Preview = preview
		if rec.Status == "" {
			rec.Status = "review"
		}
		if err := p.state.Put(sourceNZB, rec); err != nil {
			return nil, err
		}
	}
	return preview, nil
}

func (p *ImportProcessor) altMountFilePath(sourceNZB string) string {
	from := strings.Trim(strings.TrimSpace(p.cfg.AltMountPathFrom), "\"'")
	to := strings.Trim(strings.TrimSpace(p.cfg.AltMountPathTo), "\"'")
	if to == "" {
		to = "/config/.nzbs"
	}
	if from == "" {
		from = strings.Trim(strings.TrimSpace(p.cfg.SourceRoot), "\"'")
	}
	if from == "" && to == "" {
		return sourceNZB
	}

	cleanSource := filepath.Clean(strings.Trim(strings.TrimSpace(sourceNZB), "\"'"))
	cleanFrom := filepath.Clean(from)
	cleanTo := filepath.Clean(to)

	if cleanSource == cleanFrom {
		return cleanTo
	}
	if from != "" {
		if rel, err := filepath.Rel(cleanFrom, cleanSource); err == nil && rel != "." && !strings.HasPrefix(rel, "..") {
			return filepath.ToSlash(filepath.Join(cleanTo, rel))
		}
	}
	return filepath.ToSlash(filepath.Join(cleanTo, filepath.Base(cleanSource)))
}

func (p *ImportProcessor) stageAltMountNZB(sourceNZB string) string {
	staging := strings.TrimSpace(p.cfg.AltMountStagingDir)
	if staging == "" {
		return p.altMountFilePath(sourceNZB)
	}
	baseName := filepath.Base(sourceNZB)
	target := filepath.Join(staging, baseName)
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		log.Printf("winston: altmount staging mkdir failed source=%s target=%s err=%v", sourceNZB, target, err)
		return p.altMountFilePath(sourceNZB)
	}
	src, err := os.Open(sourceNZB)
	if err != nil {
		log.Printf("winston: altmount staging open source failed source=%s err=%v", sourceNZB, err)
		return p.altMountFilePath(sourceNZB)
	}
	defer src.Close()
	dst, err := os.Create(target)
	if err != nil {
		log.Printf("winston: altmount staging create target failed source=%s target=%s err=%v", sourceNZB, target, err)
		return p.altMountFilePath(sourceNZB)
	}
	if _, err := io.Copy(dst, src); err != nil {
		dst.Close()
		log.Printf("winston: altmount staging copy failed source=%s target=%s err=%v", sourceNZB, target, err)
		return p.altMountFilePath(sourceNZB)
	}
	if err := dst.Close(); err != nil {
		log.Printf("winston: altmount staging close target failed source=%s target=%s err=%v", sourceNZB, target, err)
	}
	stagingPath := strings.TrimSpace(p.cfg.AltMountStagingPath)
	if stagingPath != "" {
		return filepath.ToSlash(filepath.Join(stagingPath, baseName))
	}
	return p.altMountFilePath(target)
}
