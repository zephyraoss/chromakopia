package dataset

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	chroma "github.com/zephyraoss/libchroma"
)

type Dataset struct {
	d     *chroma.Dataset
	stats chroma.DatasetStats
}

type Resolved struct {
	MBID       uuid.UUID
	DurationMs uint32
	Embedded   *chroma.TrackMetadata
}

func Open(prefix string) (*Dataset, error) {
	d, err := chroma.Open(prefix)
	if err != nil {
		return nil, err
	}
	stats := d.Stats()
	if !stats.HasPostingIndex {
		d.Close()
		return nil, fmt.Errorf("%w: %s.cki", chroma.ErrNoPostingIndex, prefix)
	}
	return &Dataset{d: d, stats: stats}, nil
}

func (d *Dataset) Close() error {
	return d.d.Close()
}

func (d *Dataset) QueryFull(values []uint32, opts *chroma.PostingQueryOptions) ([]chroma.PostingHit, error) {
	return d.d.QueryFull(values, opts)
}

func (d *Dataset) Stride() int {
	return int(d.stats.PostingIndexTuning.Stride)
}

func (d *Dataset) RecordCount() uint64 {
	return d.stats.RecordCount
}

func (d *Dataset) HasMetadataMap() bool {
	return d.stats.HasMetadata
}

func (d *Dataset) Resolve(fingerprintID uint32) (*Resolved, error) {
	fp, err := d.d.Lookup(fingerprintID)
	if err != nil {
		return nil, fmt.Errorf("datastore lookup %d: %w", fingerprintID, err)
	}
	res := &Resolved{DurationMs: fp.DurationMs}

	meta, mbid, err := d.d.LookupMetadata(fingerprintID)
	if err != nil && mbid == nil {
		if errors.Is(err, chroma.ErrRecordNotFound) {
			return res, nil
		}
		return nil, fmt.Errorf("metadata lookup %d: %w", fingerprintID, err)
	}
	if mbid != nil {
		res.MBID = *mbid
	}
	res.Embedded = meta
	return res, nil
}
