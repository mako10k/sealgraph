package repository

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/mako10k/sealgraph/internal/domain"
	"github.com/mako10k/sealgraph/internal/revision"
)

const revisionCacheSchema = "sealgraph/revision-cache/v1"

var errRevisionCacheObservationMismatch = errors.New("revision cache REF-head observation does not match")

type revisionCacheRecord struct {
	Seal   domain.ObjectID  `json:"seal"`
	Parent *domain.ObjectID `json:"parent_revision"`
}

type revisionCachePayload struct {
	Schema           string                `json:"schema"`
	RepositoryFormat int                   `json:"repository_format"`
	Observation      string                `json:"observation_sha256"`
	Revisions        []revisionCacheRecord `json:"revisions"`
}

type revisionCacheDocument struct {
	Schema           string                `json:"schema"`
	RepositoryFormat int                   `json:"repository_format"`
	Observation      string                `json:"observation_sha256"`
	Revisions        []revisionCacheRecord `json:"revisions"`
	Checksum         string                `json:"checksum_sha256"`
}

func (r *Repository) revisionIndex(ctx context.Context, observation headObservation, bypass bool) (*revision.Index, string, error) {
	var cacheWarning string
	if !bypass {
		cached, err := r.readRevisionCache(observation)
		if err == nil && cached != nil {
			index, restoreErr := revision.Restore(ctx, observation.revisionHeads(), cached, revision.LoadSealFunc(r.LoadSeal))
			if restoreErr == nil {
				return index, "", nil
			}
		} else if err != nil && !errors.Is(err, os.ErrNotExist) && !errors.Is(err, errRevisionCacheObservationMismatch) {
			cacheWarning = fmt.Sprintf("ignored invalid revision cache: %v", err)
		}
	}
	index, err := revision.Build(ctx, observation.revisionHeads(), revision.LoadSealFunc(r.LoadSeal))
	if err != nil {
		return nil, "", err
	}
	if err := r.writeRevisionCache(observation, index.Records()); err != nil {
		writeWarning := fmt.Sprintf("could not refresh disposable revision cache: %v", err)
		if cacheWarning == "" {
			cacheWarning = writeWarning
		} else {
			cacheWarning += "; " + writeWarning
		}
	}
	return index, cacheWarning, nil
}

func (r *Repository) readRevisionCache(observation headObservation) ([]revision.Record, error) {
	path := r.revisionCachePath()
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return nil, errors.New("revision cache is not a regular non-symlink file")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var document revisionCacheDocument
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&document); err != nil {
		return nil, fmt.Errorf("decode cache: %w", err)
	}
	if err := requireCacheEOF(decoder); err != nil {
		return nil, err
	}
	if document.Schema != revisionCacheSchema || document.RepositoryFormat != 4 {
		return nil, errors.New("cache schema or repository format does not match")
	}
	if document.Observation != observation.digest() {
		return nil, errRevisionCacheObservationMismatch
	}
	if err := parseObservationDigest(document.Checksum); err != nil {
		return nil, fmt.Errorf("invalid cache checksum: %w", err)
	}
	payload := revisionCachePayload{Schema: document.Schema, RepositoryFormat: document.RepositoryFormat, Observation: document.Observation, Revisions: document.Revisions}
	payloadBytes, _ := json.Marshal(payload)
	wanted := fmt.Sprintf("%x", sha256.Sum256(payloadBytes))
	if document.Checksum != wanted {
		return nil, errors.New("cache checksum does not match its derived records")
	}
	result := make([]revision.Record, len(document.Revisions))
	for i, record := range document.Revisions {
		result[i] = revision.Record{ID: record.Seal, Parent: record.Parent}
	}
	return result, nil
}

func requireCacheEOF(decoder *json.Decoder) error {
	var trailing any
	if err := decoder.Decode(&trailing); errors.Is(err, io.EOF) {
		return nil
	} else if err != nil {
		return fmt.Errorf("decode trailing cache data: %w", err)
	}
	return errors.New("cache contains more than one JSON value")
}

func (r *Repository) writeRevisionCache(observation headObservation, records []revision.Record) error {
	wires := make([]revisionCacheRecord, len(records))
	for i, record := range records {
		wires[i] = revisionCacheRecord{Seal: record.ID, Parent: record.Parent}
	}
	payload := revisionCachePayload{Schema: revisionCacheSchema, RepositoryFormat: 4, Observation: observation.digest(), Revisions: wires}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	document := revisionCacheDocument{
		Schema: payload.Schema, RepositoryFormat: payload.RepositoryFormat, Observation: payload.Observation,
		Revisions: payload.Revisions, Checksum: fmt.Sprintf("%x", sha256.Sum256(payloadBytes)),
	}
	data, err := json.Marshal(document)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return writeAtomicCacheFile(r.revisionCachePath(), data)
}

func writeAtomicCacheFile(path string, data []byte) error {
	dir := filepath.Dir(path)
	info, err := os.Lstat(dir)
	if errors.Is(err, os.ErrNotExist) {
		if err := os.Mkdir(dir, 0o755); err != nil {
			return fmt.Errorf("create cache directory: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("inspect cache directory: %w", err)
	} else if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return errors.New("cache path is not a real directory")
	}
	temp, err := os.CreateTemp(dir, ".tmp-revision-cache-")
	if err != nil {
		return fmt.Errorf("create cache temporary file: %w", err)
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if _, err = temp.Write(data); err == nil {
		err = temp.Sync()
	}
	closeErr := temp.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return fmt.Errorf("write cache: %w", err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("publish cache atomically: %w", err)
	}
	return nil
}

func (r *Repository) revisionCachePath() string {
	return filepath.Join(r.dir, "cache", "revision-v1.json")
}
