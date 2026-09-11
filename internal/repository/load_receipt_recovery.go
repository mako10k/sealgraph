package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/mako10k/sealgraph/internal/canonical"
)

var receiptObjectKeyOrder = map[string][]string{
	"$":                     {"schema", "source_document_sha256", "seal_mappings", "collapse_groups", "typed_objects", "unchanged_objects", "refs", "tags", "semantic_projection", "excluded_objects", "excluded_state", "published_format", "repository_digest"},
	"$.seal_mappings[]":     {"old_seal", "new_seal", "material", "provenance"},
	"$.collapse_groups[]":   {"new_seal", "old_seals"},
	"$.typed_objects[]":     {"kind", "id"},
	"$.refs[]":              {"name", "old_head", "new_head"},
	"$.tags[]":              {"ref", "name", "old_target", "new_target"},
	"$.semantic_projection": {"materialized_parent_assertions", "unobserved_parent_assertions", "collapsed_revision_assertions", "merged_cause_links"},
	"$.semantic_projection.materialized_parent_assertions[]": {"old_observer", "old_child", "old_parent", "new_observer", "new_target", "new_previous"},
	"$.semantic_projection.unobserved_parent_assertions[]":   {"old_child", "old_parent", "new_child", "new_parent"},
	"$.semantic_projection.collapsed_revision_assertions[]":  {"old_observer", "old_child", "old_parent", "new_observer", "new_collapsed_seal"},
	"$.semantic_projection.merged_cause_links[]":             {"old_observer", "old_targets", "new_observer", "new_target"},
}

func RecoverUniversalLoadReceipt(ctx context.Context, workDir, sourceDigest string) ([]byte, error) {
	if err := parseObservationDigest(sourceDigest); err != nil {
		return nil, fmt.Errorf("invalid --source-document-sha256: %w", err)
	}
	repo, err := OpenStandalone(workDir)
	if err != nil {
		return nil, err
	}
	path := filepath.Join(repo.dir, "logs", "migration", sourceDigest+".receipt")
	info, err := os.Lstat(path)
	if err != nil {
		return nil, fmt.Errorf("read migration receipt %s: %w", sourceDigest, err)
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("migration receipt %s is not a regular non-symlink file", sourceDigest)
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("read migration receipt %s: %w", sourceDigest, err)
	}
	opened, err := file.Stat()
	if err != nil || !opened.Mode().IsRegular() || !os.SameFile(info, opened) {
		_ = file.Close()
		return nil, fmt.Errorf("migration receipt %s changed while it was opened", sourceDigest)
	}
	data, err := io.ReadAll(file)
	closeErr := file.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return nil, fmt.Errorf("read migration receipt %s: %w", sourceDigest, err)
	}
	receipt, err := decodeCanonicalUniversalLoadReceipt(data)
	if err != nil {
		return nil, fmt.Errorf("validate migration receipt %s: %w", sourceDigest, err)
	}
	if receipt.SourceDigest != sourceDigest {
		return nil, fmt.Errorf("migration receipt source digest is %s, expected %s", receipt.SourceDigest, sourceDigest)
	}
	if _, err := repo.Fsck(ctx); err != nil {
		return nil, fmt.Errorf("validate repository before receipt recovery: %w", err)
	}
	if err := validateReceiptAgainstRepository(ctx, repo, receipt); err != nil {
		return nil, fmt.Errorf("migration receipt is stale: repository no longer matches receipt: %w", err)
	}
	digest, err := repositoryDigest(ctx, repo)
	if err != nil {
		return nil, fmt.Errorf("derive repository digest before receipt recovery: %w", err)
	}
	if digest != receipt.RepositoryDigest {
		return nil, fmt.Errorf("migration receipt is stale: repository digest is %s, receipt requires %s", digest, receipt.RepositoryDigest)
	}
	return data, nil
}

func decodeCanonicalUniversalLoadReceipt(data []byte) (universalLoadReceiptValue, error) {
	if len(data) < 2 || data[len(data)-1] != '\n' || data[len(data)-2] == '\n' {
		return universalLoadReceiptValue{}, errors.New("receipt must be exactly one canonical JSON document followed by one LF")
	}
	payload := data[:len(data)-1]
	canonicalBytes, err := canonicalizeReceiptJSON(payload)
	if err != nil {
		return universalLoadReceiptValue{}, err
	}
	if !bytes.Equal(payload, canonicalBytes) {
		return universalLoadReceiptValue{}, errors.New("receipt JSON is not canonical")
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	var receipt universalLoadReceiptValue
	if err := decoder.Decode(&receipt); err != nil {
		return universalLoadReceiptValue{}, err
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return universalLoadReceiptValue{}, errors.New("receipt has trailing JSON data")
	}
	if err := validateUniversalLoadReceiptValue(receipt); err != nil {
		return universalLoadReceiptValue{}, err
	}
	return receipt, nil
}

func canonicalizeReceiptJSON(data []byte) ([]byte, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	result, err := appendCanonicalReceiptValue(nil, decoder, "$")
	if err != nil {
		return nil, err
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return nil, errors.New("receipt has trailing JSON data")
	}
	return result, nil
}

func appendCanonicalReceiptValue(dst []byte, decoder *json.Decoder, path string) ([]byte, error) {
	token, err := decoder.Token()
	if err != nil {
		return nil, err
	}
	switch value := token.(type) {
	case json.Delim:
		switch value {
		case '{':
			return appendCanonicalReceiptObject(dst, decoder, path)
		case '[':
			return appendCanonicalReceiptArray(dst, decoder, path)
		default:
			return nil, fmt.Errorf("unexpected JSON delimiter %q", value)
		}
	case string:
		return appendCanonicalJSONString(dst, value)
	case json.Number:
		return append(dst, value.String()...), nil
	case bool:
		if value {
			return append(dst, "true"...), nil
		}
		return append(dst, "false"...), nil
	case nil:
		return append(dst, "null"...), nil
	default:
		return nil, fmt.Errorf("unsupported JSON token %T", token)
	}
}

func appendCanonicalReceiptObject(dst []byte, decoder *json.Decoder, path string) ([]byte, error) {
	expected, ok := receiptObjectKeyOrder[path]
	if !ok {
		return nil, fmt.Errorf("unexpected receipt object at %s", path)
	}
	dst = append(dst, '{')
	seen := make(map[string]bool, len(expected))
	keys := make([]string, 0, len(expected))
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return nil, err
		}
		key, ok := token.(string)
		if !ok || seen[key] {
			return nil, fmt.Errorf("receipt object %s has a non-string or duplicate key", path)
		}
		seen[key] = true
		keys = append(keys, key)
		if len(keys) > 1 {
			dst = append(dst, ',')
		}
		dst, err = appendCanonicalJSONString(dst, key)
		if err != nil {
			return nil, err
		}
		dst = append(dst, ':')
		dst, err = appendCanonicalReceiptValue(dst, decoder, path+"."+key)
		if err != nil {
			return nil, err
		}
	}
	if _, err := decoder.Token(); err != nil {
		return nil, err
	}
	if !equalStrings(keys, expected) {
		return nil, fmt.Errorf("receipt object %s has fields out of order, missing, or extra", path)
	}
	return append(dst, '}'), nil
}

func appendCanonicalReceiptArray(dst []byte, decoder *json.Decoder, path string) ([]byte, error) {
	dst = append(dst, '[')
	index := 0
	for decoder.More() {
		if index > 0 {
			dst = append(dst, ',')
		}
		var err error
		dst, err = appendCanonicalReceiptValue(dst, decoder, path+"[]")
		if err != nil {
			return nil, err
		}
		index++
	}
	if _, err := decoder.Token(); err != nil {
		return nil, err
	}
	return append(dst, ']'), nil
}

func appendCanonicalJSONString(dst []byte, value string) ([]byte, error) {
	return canonical.AppendString(dst, value)
}
