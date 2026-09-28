package winston

func (p *ImportProcessor) ApplyTMDBCorrection(sourceNZB string, tmdbID int) (*ItemPreview, error) {
	rec, ok := p.state.Data.Imported[sourceNZB]
	if !ok {
		rec = ImportedRecord{}
	}
	rec.Metadata.TMDBID = tmdbID
	rec.Metadata.TVDBID = 0
	rec.Metadata.IMDBID = ""
	preview := p.BuildPreview(sourceNZB, rec.Metadata)
	rec.State = StateCorrected
	rec.Confidence = preview.Confidence
	rec.RelativePath = preview.ProposedPath
	rec.Preview = preview
	if err := p.state.Put(sourceNZB, rec); err != nil {
		return nil, err
	}
	return preview, nil
}

func (p *ImportProcessor) ApplyRelativePathOverride(sourceNZB, relativePath string) (*ItemPreview, error) {
	rec, ok := p.state.Data.Imported[sourceNZB]
	if !ok {
		rec = ImportedRecord{}
	}
	rec.Metadata.RelativePathOverride = relativePath
	preview := p.BuildPreview(sourceNZB, rec.Metadata)
	rec.State = StateCorrected
	rec.Confidence = preview.Confidence
	rec.RelativePath = preview.ProposedPath
	rec.Preview = preview
	if err := p.state.Put(sourceNZB, rec); err != nil {
		return nil, err
	}
	return preview, nil
}
