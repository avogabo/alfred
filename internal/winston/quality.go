package winston

import (
	"encoding/xml"
	"os"
	"regexp"
	"strings"
)

var qualityPatterns = []struct {
	label string
	re    *regexp.Regexp
}{
	{label: "2160", re: regexp.MustCompile(`(?i)(?:^|[^0-9])(2160p|2160|4k|uhd)(?:[^0-9]|$)`)},
	{label: "1080", re: regexp.MustCompile(`(?i)(?:^|[^0-9])(1080p|1080)(?:[^0-9]|$)`)},
	{label: "720", re: regexp.MustCompile(`(?i)(?:^|[^0-9])(720p|720)(?:[^0-9]|$)`)},
}

type nzb struct {
	Files []struct {
		Subject string `xml:"subject,attr"`
	} `xml:"file"`
}

func detectQualityForSource(sourceNZB string) string {
	if q := detectQualityFromText(sourceNZB); q != "" {
		return q
	}
	if q := detectQualityFromText(filepathBase(sourceNZB)); q != "" {
		return q
	}
	if q := detectQualityFromNZB(sourceNZB); q != "" {
		return q
	}
	return ""
}

func detectQualityFromNZB(path string) string {
	data, err := os.ReadFile(path)
	if err != nil || len(data) == 0 {
		return ""
	}
	var doc nzb
	if err := xml.Unmarshal(data, &doc); err != nil {
		return ""
	}
	for _, f := range doc.Files {
		if q := detectQualityFromText(f.Subject); q != "" {
			return q
		}
	}
	return ""
}

func detectQualityFromText(s string) string {
	low := strings.ToLower(s)
	for _, p := range qualityPatterns {
		if p.re.MatchString(low) {
			return p.label
		}
	}
	return ""
}

