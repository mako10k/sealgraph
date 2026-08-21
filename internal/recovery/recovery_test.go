package recovery

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPreparedCommittedCanonicalRoundTripAndUnsafeDirectoryRefusal(t *testing.T) {
	repositoryDir := t.TempDir()
	store := NewStore(repositoryDir)
	record, err := store.Prepare("seal", []Transition{{REF: "root", Before: nil, After: []byte("manifest")}})
	if err != nil || record.State != Prepared {
		t.Fatalf("prepared=%+v err=%v", record, err)
	}
	loaded, err := store.Load(record.ID)
	if err != nil || loaded.State != Prepared {
		t.Fatalf("loaded=%+v err=%v", loaded, err)
	}
	committed, err := store.Commit(record)
	if err != nil || committed.State != Committed {
		t.Fatalf("committed=%+v err=%v", committed, err)
	}

	unsafeRoot := t.TempDir()
	if err := os.Symlink(t.TempDir(), filepath.Join(unsafeRoot, "logs")); err != nil {
		t.Fatal(err)
	}
	if _, err := NewStore(unsafeRoot).Prepare("seal", []Transition{{REF: "root", After: []byte("manifest")}}); err == nil {
		t.Fatal("recovery store followed a symlink directory")
	}
}

func TestLoadRequiresExactFullIDAndCanonicalBytes(t *testing.T) {
	repositoryDir := t.TempDir()
	store := NewStore(repositoryDir)
	record, err := store.Prepare("seal", []Transition{{REF: "root", After: []byte("manifest")}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Load(record.ID[:12]); err == nil {
		t.Fatal("operation ID prefix was accepted")
	}
	path := filepath.Join(repositoryDir, "logs", "recovery", record.ID+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append([]byte(" "), data...), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Load(record.ID); err == nil {
		t.Fatal("noncanonical recovery bytes were accepted")
	}
}

func TestPrepareRejectsInvalidREFKindShapeAndManifestLimit(t *testing.T) {
	store := NewStore(t.TempDir())
	tests := []struct {
		name        string
		kind        string
		transitions []Transition
	}{
		{"invalid REF", "seal", []Transition{{REF: "bad..ref", After: []byte("manifest")}}},
		{"seal deletion", "seal", []Transition{{REF: "root", Before: []byte("manifest")}}},
		{"tag absent", "tag", []Transition{{REF: "root", After: []byte("manifest")}}},
		{"move single", "mv", []Transition{{REF: "root", Before: []byte("manifest")}}},
		{"oversized", "seal", []Transition{{REF: "root", After: make([]byte, MaxManifestBytes+1)}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := store.Prepare(test.kind, test.transitions); err == nil {
				t.Fatal("invalid recovery record was prepared")
			}
		})
	}
}
