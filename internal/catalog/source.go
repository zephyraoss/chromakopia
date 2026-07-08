package catalog

import "context"

type RecordingSummary struct {
	Title      string
	Artist     string
	Album      string
	DurationMs int64
}

type RecordingSource interface {
	RecordingSummary(ctx context.Context, mbid string) (*RecordingSummary, error)
}

func (s *Store) RecordingSummary(ctx context.Context, mbid string) (*RecordingSummary, error) {
	rec, err := s.Recording(ctx, mbid)
	if rec == nil || err != nil {
		return nil, err
	}
	return summarizeRecording(rec), nil
}

func summarizeRecording(rec *Recording) *RecordingSummary {
	sum := &RecordingSummary{
		Title:      rec.Title,
		Artist:     rec.ArtistCredit,
		DurationMs: rec.LengthMs,
	}
	if len(rec.Releases) > 0 {
		first := rec.Releases[0]
		if first.ReleaseGroup != nil && first.ReleaseGroup.Title != "" {
			sum.Album = first.ReleaseGroup.Title
		} else {
			sum.Album = first.Title
		}
	}
	return sum
}
