// Package recovery stores strict non-canonical local REF operation records.
package recovery

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mako10k/sealgraph/internal/domain"
)

const (
	Schema           = "sealgraph/recovery/v1"
	MaxManifestBytes = 16 << 20
	MaxRecordBytes   = 64 << 20
)

type State string

const (
	Prepared  State = "PREPARED"
	Committed State = "COMMITTED"
)

type Transition struct {
	REF    string `json:"ref"`
	Before []byte `json:"before"`
	After  []byte `json:"after"`
}

type Record struct {
	Schema      string       `json:"schema"`
	ID          string       `json:"operation_id"`
	State       State        `json:"state"`
	Kind        string       `json:"kind"`
	Transitions []Transition `json:"transitions"`
}

type Entry struct {
	ID     string
	Record *Record
	Err    error
}

type Store struct {
	root string
	dir  string
}

func NewStore(repositoryDir string) *Store {
	return &Store{root: repositoryDir, dir: filepath.Join(repositoryDir, "logs", "recovery")}
}

func (s *Store) Prepare(kind string, transitions []Transition) (Record, error) {
	id, err := newID()
	if err != nil {
		return Record{}, err
	}
	record := Record{Schema: Schema, ID: id, State: Prepared, Kind: kind, Transitions: transitions}
	if err := validateRecord(record); err != nil {
		return Record{}, err
	}
	data, err := encode(record)
	if err != nil {
		return Record{}, err
	}
	if err := s.ensureDirectory(); err != nil {
		return Record{}, fmt.Errorf("create recovery journal directory: %w", err)
	}
	path := s.path(id)
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return Record{}, fmt.Errorf("create PREPARED recovery record: %w", err)
	}
	writeErr := writeSyncClose(file, data)
	if writeErr != nil {
		_ = os.Remove(path)
		return Record{}, fmt.Errorf("publish PREPARED recovery record: %w", writeErr)
	}
	if err := syncDirectory(s.dir); err != nil {
		return Record{}, fmt.Errorf("PREPARED recovery record %s may be visible but directory durability failed: %w", id, err)
	}
	return record, nil
}

func (s *Store) Commit(prepared Record) (Record, error) {
	if err := s.requireDirectory(); err != nil {
		return Record{}, err
	}
	if prepared.State != Prepared {
		return Record{}, fmt.Errorf("operation %s is not PREPARED", prepared.ID)
	}
	current, err := s.Load(prepared.ID)
	if err != nil {
		return Record{}, err
	}
	if !equalRecord(current, prepared) {
		return Record{}, fmt.Errorf("PREPARED recovery record %s changed before COMMITTED publication", prepared.ID)
	}
	committed := prepared
	committed.State = Committed
	data, err := encode(committed)
	if err != nil {
		return Record{}, err
	}
	temp, err := os.CreateTemp(s.dir, ".tmp-recovery-")
	if err != nil {
		return Record{}, err
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if err := writeSyncClose(temp, data); err != nil {
		return Record{}, err
	}
	if err := os.Rename(tempPath, s.path(prepared.ID)); err != nil {
		return Record{}, fmt.Errorf("publish COMMITTED recovery record: %w", err)
	}
	if err := syncDirectory(s.dir); err != nil {
		return Record{}, fmt.Errorf("operation %s committed but journal directory durability failed: %w", prepared.ID, err)
	}
	return committed, nil
}

func (s *Store) Load(id string) (Record, error) {
	if !validID(id) {
		return Record{}, fmt.Errorf("operation ID must be exactly 32 lower-case hexadecimal characters")
	}
	if err := s.requireDirectory(); err != nil {
		return Record{}, err
	}
	info, err := os.Lstat(s.path(id))
	if err != nil {
		return Record{}, err
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return Record{}, fmt.Errorf("recovery record %s is not a regular non-symlink file", id)
	}
	if info.Size() > MaxRecordBytes {
		return Record{}, fmt.Errorf("recovery record %s exceeds %d bytes", id, MaxRecordBytes)
	}
	data, err := os.ReadFile(s.path(id))
	if err != nil {
		return Record{}, err
	}
	return decode(data, id)
}

func (s *Store) List() ([]Entry, error) {
	if err := s.requireDirectory(); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	entries, err := os.ReadDir(s.dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	result := make([]Entry, 0, len(entries))
	for _, entry := range entries {
		id := strings.TrimSuffix(entry.Name(), ".json")
		if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 || !validID(id) || entry.Name() != id+".json" {
			result = append(result, Entry{ID: entry.Name(), Err: fmt.Errorf("invalid recovery journal entry")})
			continue
		}
		record, loadErr := s.Load(id)
		if loadErr != nil {
			result = append(result, Entry{ID: id, Err: loadErr})
		} else {
			copy := record
			result = append(result, Entry{ID: id, Record: &copy})
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}

func (s *Store) path(id string) string { return filepath.Join(s.dir, id+".json") }

func (s *Store) ensureDirectory() error {
	current := s.root
	for _, name := range []string{"logs", "recovery"} {
		current = filepath.Join(current, name)
		info, err := os.Lstat(current)
		if errors.Is(err, os.ErrNotExist) {
			if err := os.Mkdir(current, 0o700); err != nil {
				return err
			}
			continue
		}
		if err != nil {
			return err
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("recovery journal path %s is not a real directory", current)
		}
	}
	return nil
}

func (s *Store) requireDirectory() error {
	current := s.root
	for _, name := range []string{"logs", "recovery"} {
		current = filepath.Join(current, name)
		info, err := os.Lstat(current)
		if err != nil {
			return err
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("recovery journal path %s is not a real directory", current)
		}
	}
	return nil
}

func encode(record Record) ([]byte, error) {
	if err := validateRecord(record); err != nil {
		return nil, err
	}
	data, err := json.Marshal(record)
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func decode(data []byte, expectedID string) (Record, error) {
	var record Record
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&record); err != nil {
		return Record{}, err
	}
	if record.ID != expectedID {
		return Record{}, fmt.Errorf("record operation ID %q does not match filename", record.ID)
	}
	canonical, err := encode(record)
	if err != nil {
		return Record{}, err
	}
	if !bytes.Equal(data, canonical) {
		return Record{}, fmt.Errorf("recovery record is not canonical")
	}
	return record, nil
}

func validateRecord(record Record) error {
	if record.Schema != Schema || !validID(record.ID) || (record.State != Prepared && record.State != Committed) {
		return fmt.Errorf("invalid recovery record header")
	}
	if record.Kind != "seal" && record.Kind != "tag" && record.Kind != "mv" && record.Kind != "ref-drop" {
		return fmt.Errorf("unsupported recovery operation kind %q", record.Kind)
	}
	if len(record.Transitions) == 0 || len(record.Transitions) > 2 {
		return fmt.Errorf("recovery record requires one or two transitions")
	}
	for i, transition := range record.Transitions {
		if err := domain.ValidateREF(transition.REF); err != nil {
			return fmt.Errorf("invalid recovery transition REF %q: %w", transition.REF, err)
		}
		if bytes.Equal(transition.Before, transition.After) {
			return fmt.Errorf("invalid recovery transition for %q", transition.REF)
		}
		if len(transition.Before) > MaxManifestBytes || len(transition.After) > MaxManifestBytes {
			return fmt.Errorf("recovery transition for %q exceeds %d manifest bytes", transition.REF, MaxManifestBytes)
		}
		if i > 0 && record.Transitions[i-1].REF >= transition.REF {
			return fmt.Errorf("recovery transitions are not strictly REF-sorted")
		}
	}
	if err := validateKindShape(record); err != nil {
		return err
	}
	return nil
}

func validateKindShape(record Record) error {
	switch record.Kind {
	case "seal":
		if len(record.Transitions) != 1 || record.Transitions[0].After == nil {
			return fmt.Errorf("seal recovery requires one transition to a present manifest")
		}
		return nil
	case "tag":
		return validateSingleShape(record.Transitions, true, true, "tag recovery requires one present-to-present transition")
	case "ref-drop":
		return validateSingleShape(record.Transitions, true, false, "ref-drop recovery requires one present-to-absent transition")
	case "mv":
		return validateMoveShape(record.Transitions)
	}
	return nil
}

func validateSingleShape(transitions []Transition, beforePresent, afterPresent bool, message string) error {
	if len(transitions) != 1 || (transitions[0].Before != nil) != beforePresent || (transitions[0].After != nil) != afterPresent {
		return errors.New(message)
	}
	return nil
}

func validateMoveShape(transitions []Transition) error {
	if len(transitions) != 2 {
		return fmt.Errorf("mv recovery requires exactly two transitions")
	}
	oldCount, newCount := 0, 0
	for _, transition := range transitions {
		if transition.Before != nil && transition.After == nil {
			oldCount++
		}
		if transition.Before == nil && transition.After != nil {
			newCount++
		}
	}
	if oldCount != 1 || newCount != 1 {
		return fmt.Errorf("mv recovery requires one present-to-absent and one absent-to-present transition")
	}
	return nil
}

func newID() (string, error) {
	data := make([]byte, 16)
	if _, err := rand.Read(data); err != nil {
		return "", err
	}
	return hex.EncodeToString(data), nil
}

func validID(value string) bool {
	if len(value) != 32 {
		return false
	}
	decoded, err := hex.DecodeString(value)
	return err == nil && hex.EncodeToString(decoded) == value
}

func writeSyncClose(file *os.File, data []byte) error {
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}

func syncDirectory(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}

func equalRecord(left, right Record) bool {
	a, _ := encode(left)
	b, _ := encode(right)
	return bytes.Equal(a, b)
}
