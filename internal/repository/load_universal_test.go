package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mako10k/sealgraph/internal/canonical"
	"github.com/mako10k/sealgraph/internal/domain"
	"github.com/mako10k/sealgraph/internal/migration"
)

func universalRootDump(t *testing.T, content []byte) []byte {
	return universalRootDumpWithDraft(t, content, false)
}

func universalRootDumpWithDraft(t *testing.T, content []byte, draft bool) []byte {
	t.Helper()
	contentID := domain.ComputeNativeBlobID(content)
	payload := domain.SealPayload{
		Schema: domain.SealSchema, Content: domain.ContentRef{Store: domain.NativeStore, Type: domain.BlobType, ID: contentID},
		Attachments: []domain.Attachment{}, Links: []domain.Link{}, Root: true, Draft: draft,
	}
	payloadBytes, err := canonical.EncodeSeal(payload)
	if err != nil {
		t.Fatal(err)
	}
	oldID := domain.ComputeNativeBlobID(payloadBytes)
	document, err := migration.EncodeUniversalBlobV1(migration.UniversalBlobV1{
		Objects: []migration.ObjectRecord{{ID: contentID, Data: content}},
		Seals:   []migration.UniversalSealRecord{{ID: oldID, Payload: payload, PayloadBytes: payloadBytes}},
		REFs:    []migration.RefRecord{{Name: "ROOT", Head: oldID}}, Tags: []migration.TagRecord{},
		ExcludedObjects: []domain.ObjectID{}, ExcludedState: append([]string{}, migration.UniversalExcludedState...),
	})
	if err != nil {
		t.Fatal(err)
	}
	return document
}

func TestLoadUniversalBlobV1PublishesFormat5AndReceipt(t *testing.T) {
	dir := t.TempDir()
	input := universalRootDump(t, []byte("migrated\x00material"))
	result, err := LoadUniversalBlobV1(context.Background(), dir, input)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Warnings) != 0 || !bytes.HasSuffix(result.Receipt, []byte("\n")) || !bytes.Contains(result.Receipt, []byte(`"schema":"sealgraph/universal-blob-load-receipt/v1"`)) {
		t.Fatalf("result=%q warnings=%v", result.Receipt, result.Warnings)
	}
	config, err := os.ReadFile(filepath.Join(dir, ".sealgraph", "config"))
	if err != nil {
		t.Fatal(err)
	}
	if string(config) != configBytes {
		t.Fatalf("config=%q", config)
	}
	repo, err := OpenStandalone(dir)
	if err != nil {
		t.Fatal(err)
	}
	requireMigratedRoot(t, repo)
	requireRecoverableLoadReceipt(t, dir, repo, input, result.Receipt)
}

func requireMigratedRoot(t *testing.T, repo *Repository) {
	t.Helper()
	shown, err := repo.Show(context.Background(), "ROOT")
	if err != nil {
		t.Fatal(err)
	}
	if string(shown.Content) != "migrated\x00material" || shown.Resolved.Seal.Schema != "sealgraph/seal/v5" {
		t.Fatalf("shown=%+v content=%q", shown.Resolved, shown.Content)
	}
}

func requireRecoverableLoadReceipt(t *testing.T, dir string, repo *Repository, input, receipt []byte) {
	t.Helper()
	digest := strings.TrimSuffix(filepath.Base(migrationReceiptPath(filepath.Join(dir, ".sealgraph"), input)), ".receipt")
	stored, err := os.ReadFile(filepath.Join(dir, ".sealgraph", "logs", "migration", digest+".receipt"))
	if err != nil || !bytes.Equal(stored, receipt) {
		t.Fatalf("stored receipt err=%v equal=%t", err, bytes.Equal(stored, receipt))
	}
	recovered, err := RecoverUniversalLoadReceipt(context.Background(), dir, digest)
	if err != nil || !bytes.Equal(recovered, receipt) {
		t.Fatalf("recovered receipt err=%v equal=%t", err, bytes.Equal(recovered, receipt))
	}
	if err := os.Chmod(filepath.Join(dir, ".sealgraph", "logs", "migration", digest+".receipt"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := verifyUniversalLoadModes(filepath.Join(dir, ".sealgraph")); err == nil || !strings.Contains(err.Error(), "mode is 0644") {
		t.Fatalf("loader mode verification error=%v", err)
	}
	recovered, err = RecoverUniversalLoadReceipt(context.Background(), dir, digest)
	if err != nil || !bytes.Equal(recovered, receipt) {
		t.Fatalf("mode-neutral recovered receipt err=%v equal=%t", err, bytes.Equal(recovered, receipt))
	}
	if _, err := repo.objects.WriteBlob(context.Background(), []byte("later unreferenced state")); err != nil {
		t.Fatal(err)
	}
	if _, err := RecoverUniversalLoadReceipt(context.Background(), dir, digest); err == nil || !strings.Contains(err.Error(), "stale") {
		t.Fatalf("stale recovery error=%v", err)
	}
}

func TestRecoverUniversalLoadReceiptRejectsNoncanonicalStoredBytes(t *testing.T) {
	dir, digest, path, stored := loadedReceiptFixture(t)
	noncanonical := bytes.Replace(stored, []byte(`"seal_mappings":[`), []byte(`"seal_mappings": [`), 1)
	if err := os.WriteFile(path, noncanonical, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := RecoverUniversalLoadReceipt(context.Background(), dir, digest); err == nil || !strings.Contains(err.Error(), "not canonical") {
		t.Fatalf("recovery error=%v", err)
	}
}

func TestRecoverUniversalLoadReceiptRejectsAlternateNumberSpelling(t *testing.T) {
	dir, digest, path, stored := loadedReceiptFixture(t)
	noncanonical := bytes.Replace(stored, []byte(`"published_format":5`), []byte(`"published_format":5.0`), 1)
	if bytes.Equal(noncanonical, stored) {
		t.Fatal("published_format number fixture did not mutate receipt")
	}
	if err := os.WriteFile(path, noncanonical, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := RecoverUniversalLoadReceipt(context.Background(), dir, digest); err == nil {
		t.Fatal("receipt with alternate JSON number spelling was accepted")
	}
}

func loadedReceiptFixture(t *testing.T) (dir, digest, path string, stored []byte) {
	t.Helper()
	dir = t.TempDir()
	input := universalRootDump(t, []byte("material"))
	if _, err := LoadUniversalBlobV1(context.Background(), dir, input); err != nil {
		t.Fatal(err)
	}
	digest = strings.TrimSuffix(filepath.Base(migrationReceiptPath(filepath.Join(dir, ".sealgraph"), input)), ".receipt")
	path = filepath.Join(dir, ".sealgraph", "logs", "migration", digest+".receipt")
	var err error
	stored, err = os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return dir, digest, path, stored
}

func TestRecoverUniversalLoadReceiptRejectsCanonicalButInconsistentStoredValues(t *testing.T) {
	dir := t.TempDir()
	input := universalRootDump(t, []byte("material"))
	result, err := LoadUniversalBlobV1(context.Background(), dir, input)
	if err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		SealMappings []struct {
			OldSeal string `json:"old_seal"`
		} `json:"seal_mappings"`
	}
	if err := json.Unmarshal(result.Receipt, &decoded); err != nil {
		t.Fatal(err)
	}
	digest := strings.TrimSuffix(filepath.Base(migrationReceiptPath(filepath.Join(dir, ".sealgraph"), input)), ".receipt")
	path := filepath.Join(dir, ".sealgraph", "logs", "migration", digest+".receipt")
	old := decoded.SealMappings[0].OldSeal
	replacement := strings.Repeat("a", 64)
	if replacement == old {
		replacement = strings.Repeat("b", 64)
	}
	mutated := bytes.Replace(result.Receipt, []byte(`"old_seal":"`+old+`"`), []byte(`"old_seal":"`+replacement+`"`), 1)
	if bytes.Equal(mutated, result.Receipt) {
		t.Fatal("receipt mutation did not match")
	}
	if err := os.WriteFile(path, mutated, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := RecoverUniversalLoadReceipt(context.Background(), dir, digest); err == nil || !strings.Contains(err.Error(), "does not match seal_mappings") {
		t.Fatalf("recovery error=%v", err)
	}
}

func TestUniversalLoadReceiptInventoriesTypedRoleCollisionFromUnchangedBlob(t *testing.T) {
	content := []byte(`{"schema":"sealgraph/provenance/v1","root":true,"draft":false,"cause_links":[]}`)
	dir := t.TempDir()
	input := universalRootDump(t, content)
	result, err := LoadUniversalBlobV1(context.Background(), dir, input)
	if err != nil {
		t.Fatal(err)
	}
	contentID := domain.ComputeNativeBlobID(content).String()
	if !bytes.Contains(result.Receipt, []byte(`{"kind":"provenance","id":"`+contentID+`"}`)) {
		t.Fatalf("receipt omits unchanged Blob's Provenance role: %s", result.Receipt)
	}
	digest := strings.TrimSuffix(filepath.Base(migrationReceiptPath(filepath.Join(dir, ".sealgraph"), input)), ".receipt")
	if _, err := RecoverUniversalLoadReceipt(context.Background(), dir, digest); err != nil {
		t.Fatalf("recover typed-role-collision receipt: %v", err)
	}
}

func TestUniversalLoadReceiptIsRevalidatedAgainstExactLoadInputs(t *testing.T) {
	dir := t.TempDir()
	input := universalRootDump(t, []byte("material"))
	result, err := LoadUniversalBlobV1(context.Background(), dir, input)
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := decodeCanonicalUniversalLoadReceipt(result.Receipt)
	if err != nil {
		t.Fatal(err)
	}
	dump, err := migration.DecodeUniversalBlobV1(input)
	if err != nil {
		t.Fatal(err)
	}
	projection, err := projectUniversalDump(dump)
	if err != nil {
		t.Fatal(err)
	}
	receipt.SourceDigest = strings.Repeat("a", 64)
	if err := validateReceiptAgainstLoadInputs(input, dump, projection, receipt.RepositoryDigest, receipt); err == nil || !strings.Contains(err.Error(), "does not exactly match") {
		t.Fatalf("receipt/input validation error=%v", err)
	}
}

func TestLoadUniversalBlobV1RejectsNoncanonicalInputBeforeTargetCreation(t *testing.T) {
	dir := t.TempDir()
	input := bytes.TrimSuffix(universalRootDump(t, []byte("material")), []byte("\n"))
	if _, err := LoadUniversalBlobV1(context.Background(), dir, input); err == nil || !strings.Contains(err.Error(), "PRE_PUBLICATION_FAILURE") {
		t.Fatalf("err=%v", err)
	}
	if _, err := os.Lstat(filepath.Join(dir, ".sealgraph")); !os.IsNotExist(err) {
		t.Fatalf("target exists: %v", err)
	}
}

func TestLoadUniversalBlobV1RejectsAlternateNumberSpellingsBeforeTargetCreation(t *testing.T) {
	canonicalInput := universalRootDump(t, []byte("material"))
	for _, spelling := range []string{"+4", "04", "4.0", "4.00", "4e0", "4E+0", "4E-0"} {
		t.Run(spelling, func(t *testing.T) {
			dir := t.TempDir()
			input := bytes.Replace(canonicalInput, []byte(`"format":4`), []byte(`"format":`+spelling), 1)
			if bytes.Equal(input, canonicalInput) {
				t.Fatal("number fixture did not mutate source_repository.format")
			}
			if _, err := LoadUniversalBlobV1(context.Background(), dir, input); err == nil || !strings.Contains(err.Error(), "PRE_PUBLICATION_FAILURE") {
				t.Fatalf("spelling=%q err=%v", spelling, err)
			}
			if _, err := os.Lstat(filepath.Join(dir, ".sealgraph")); !os.IsNotExist(err) {
				t.Fatalf("target exists: %v", err)
			}
		})
	}
}

func TestLoadUniversalBlobV1RejectsNonzeroBase64PadBitsBeforeTargetCreation(t *testing.T) {
	objectInput := universalRootDump(t, []byte("f"))
	payloadInput := universalRootDump(t, []byte("payload"))
	if _, ok := mutateBase64PadBits(payloadInput, "payload_base64"); !ok {
		payloadInput = universalRootDumpWithDraft(t, []byte("payload"), true)
	}
	tests := []struct {
		name  string
		field string
		input []byte
	}{
		{name: "object bytes", field: "bytes_base64", input: objectInput},
		{name: "Seal payload", field: "payload_base64", input: payloadInput},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			input, ok := mutateBase64PadBits(test.input, test.field)
			if !ok {
				t.Fatalf("%s fixture has no padded Base64 value", test.field)
			}
			if _, err := LoadUniversalBlobV1(context.Background(), dir, input); err == nil || !strings.Contains(err.Error(), "PRE_PUBLICATION_FAILURE") {
				t.Fatalf("err=%v", err)
			}
			if _, err := os.Lstat(filepath.Join(dir, ".sealgraph")); !os.IsNotExist(err) {
				t.Fatalf("target exists: %v", err)
			}
		})
	}
}

func mutateBase64PadBits(input []byte, field string) ([]byte, bool) {
	marker := []byte(`"` + field + `":"`)
	start := bytes.Index(input, marker)
	if start < 0 {
		return nil, false
	}
	start += len(marker)
	endOffset := bytes.IndexByte(input[start:], '"')
	if endOffset < 0 {
		return nil, false
	}
	end := start + endOffset
	encoded := input[start:end]
	dataIndex, mask := -1, byte(0)
	switch {
	case bytes.HasSuffix(encoded, []byte("==")):
		dataIndex, mask = len(encoded)-3, 0x0f
	case bytes.HasSuffix(encoded, []byte("=")):
		dataIndex, mask = len(encoded)-2, 0x03
	default:
		return nil, false
	}
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	value := strings.IndexByte(alphabet, encoded[dataIndex])
	if value < 0 || byte(value)&mask != 0 {
		return nil, false
	}
	mutated := append([]byte(nil), input...)
	mutated[start+dataIndex] = alphabet[value|1]
	return mutated, true
}

func TestLoadUniversalBlobV1NeverReplacesExistingTarget(t *testing.T) {
	dir := t.TempDir()
	if _, err := InitStandalone(dir); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(filepath.Join(dir, ".sealgraph", "config"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := LoadUniversalBlobV1(context.Background(), dir, universalRootDump(t, []byte("material"))); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("err=%v", err)
	}
	after, err := os.ReadFile(filepath.Join(dir, ".sealgraph", "config"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("existing target changed")
	}
}

func TestLoadDirectoryDurabilityUsesNoFollowHandleAndFinalMode(t *testing.T) {
	dir := t.TempDir()
	staging := filepath.Join(dir, "staging")
	if err := os.Mkdir(staging, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := finalizeStagingDirectory(staging); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(staging)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o755 {
		t.Fatalf("staging mode=%v", info.Mode())
	}
	link := filepath.Join(dir, "staging-link")
	if err := os.Symlink(staging, link); err != nil {
		t.Fatal(err)
	}
	if err := syncDirectoryForLoad(link); err == nil {
		t.Fatal("directory sync followed a symbolic link")
	}
}
