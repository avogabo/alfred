package winston

import (
	"path/filepath"
	"strings"
)

func (p *ImportProcessor) buildRelativePath(sourceNZB string) string {
	base := filepath.Base(sourceNZB)
	name := strings.TrimSuffix(base, filepath.Ext(base))

	sourceRoot := filepath.Clean(p.cfg.SourceRoot)
	dir := filepath.Dir(sourceNZB)
	relDir, err := filepath.Rel(sourceRoot, dir)
	if err != nil || relDir == "." {
		relDir = ""
	}

	switch p.cfg.DefaultMode {
	case "preserve":
		if relDir == "" {
			return filepath.ToSlash(name)
		}
		return filepath.ToSlash(filepath.Join(relDir, name))
	case "template":
		if relDir == "" {
			return filepath.ToSlash(filepath.Join("imports", name))
		}
		return filepath.ToSlash(filepath.Join("imports", relDir, name))
	default:
		if relDir == "" {
			return filepath.ToSlash(name)
		}
		return filepath.ToSlash(filepath.Join(relDir, name))
	}
}
