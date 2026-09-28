package winston

func (p *ImportProcessor) Approve(sourceNZB string) (*ItemPreview, error) {
	rec, ok := p.state.Data.Imported[sourceNZB]
	if !ok {
		rec = ImportedRecord{}
	}
	preview := p.BuildPreview(sourceNZB, rec.Metadata)
	preview.State = StateApproved
	rec.State = StateApproved
	rec.Status = "approved"
	rec.Confidence = preview.Confidence
	rec.RelativePath = preview.ProposedPath
	rec.Preview = preview
	if err := p.state.Put(sourceNZB, rec); err != nil {
		return nil, err
	}
	return preview, nil
}
