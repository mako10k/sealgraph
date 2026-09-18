package repository

import (
	"bytes"
	"fmt"

	canonicalv7 "github.com/mako10k/sealgraph/internal/canonical/v7"
	"github.com/mako10k/sealgraph/internal/domain"
	domainv7 "github.com/mako10k/sealgraph/internal/domain/v7"
)

// originClosure validates the immutable typed children of one format-7
// provenance against the exact Material content. It does not traverse Cause
// or revision edges, which belong to the separate Seal graph.
func originClosure(contentID domain.ObjectID, content []byte, originID domain.ObjectID, read func(domain.ObjectID) ([]byte, error)) (domainv7.OriginMap, map[string]domainv7.SourceSnapshot, error) {
	mapBytes, err := read(originID)
	if err != nil {
		return domainv7.OriginMap{}, nil, fmt.Errorf("read OriginMap %s: %w", originID, err)
	}
	origin, err := canonicalv7.DecodeOriginMap(mapBytes)
	if err != nil {
		return domainv7.OriginMap{}, nil, fmt.Errorf("OriginMap %s is not canonical: %w", originID, err)
	}
	if !origin.Content.Equal(contentID) {
		return domainv7.OriginMap{}, nil, fmt.Errorf("OriginMap %s content %s differs from Material content %s", originID, origin.Content, contentID)
	}
	snapshots := make(map[string]domainv7.SourceSnapshot)
	sources := make(map[string][]byte)
	var offset uint64
	for index, run := range origin.Runs {
		if run.Length > uint64(len(content))-offset {
			return domainv7.OriginMap{}, nil, fmt.Errorf("OriginMap %s run %d exceeds content byte length", originID, index)
		}
		if run.Kind == "external" {
			snapshot, ok := snapshots[run.Snapshot.String()]
			if !ok {
				snapshotBytes, readErr := read(run.Snapshot)
				if readErr != nil {
					return domainv7.OriginMap{}, nil, fmt.Errorf("read SourceSnapshot %s: %w", run.Snapshot, readErr)
				}
				snapshot, err = canonicalv7.DecodeSourceSnapshot(snapshotBytes)
				if err != nil {
					return domainv7.OriginMap{}, nil, fmt.Errorf("SourceSnapshot %s is not canonical: %w", run.Snapshot, err)
				}
				snapshots[run.Snapshot.String()] = snapshot
			}
			source, ok := sources[snapshot.Content.String()]
			if !ok {
				source, err = read(snapshot.Content)
				if err != nil {
					return domainv7.OriginMap{}, nil, fmt.Errorf("read full source Blob %s: %w", snapshot.Content, err)
				}
				sources[snapshot.Content.String()] = source
			}
			if run.SourceStart > uint64(len(source)) || run.Length > uint64(len(source))-run.SourceStart {
				return domainv7.OriginMap{}, nil, fmt.Errorf("OriginMap %s run %d exceeds full source Blob %s", originID, index, snapshot.Content)
			}
			if !bytes.Equal(content[int(offset):int(offset+run.Length)], source[int(run.SourceStart):int(run.SourceStart+run.Length)]) {
				return domainv7.OriginMap{}, nil, fmt.Errorf("OriginMap %s run %d copied bytes differ from SourceSnapshot %s", originID, index, run.Snapshot)
			}
		}
		offset += run.Length
	}
	if offset != uint64(len(content)) {
		return domainv7.OriginMap{}, nil, fmt.Errorf("OriginMap %s runs cover %d bytes, content has %d", originID, offset, len(content))
	}
	return origin, snapshots, nil
}
