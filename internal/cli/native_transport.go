package cli

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math"
	"path/filepath"
	"strconv"

	"github.com/mako10k/sealgraph/internal/fsread"
	"github.com/mako10k/sealgraph/internal/repository"
)

type nativeLoadReceipt struct {
	Schema           string `json:"schema"`
	Result           string `json:"result"`
	SnapshotSHA256   string `json:"snapshot_sha256"`
	RepositoryFormat int    `json:"repository_format"`
	Blobs            int    `json:"blobs"`
	Refs             int    `json:"refs"`
	Candidates       int    `json:"candidates"`
}

func runDump(ctx context.Context, workDir string, args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("dump", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var format singleString
	flags.Var(&format, "format", "required native snapshot format")
	if err := flags.Parse(args); err != nil {
		return flagUsageError(stderr, "dump", err)
	}
	if flags.NArg() != 0 {
		return usageError(stderr, "dump accepts no positional arguments; unexpected argument %q", flags.Arg(0))
	}
	if !format.set {
		return usageError(stderr, "dump requires --format native-blobs-v1")
	}
	if format.value != "native-blobs-v1" {
		return usageError(stderr, "dump format %q is unsupported; expected native-blobs-v1", format.value)
	}
	repo, err := repository.OpenStandalone(workDir)
	if err != nil {
		return commandError(stderr, "dump", err)
	}
	if repo.Format() != 7 {
		return commandError(stderr, "dump", fmt.Errorf("native-blobs-v1 dump requires repository format 7"))
	}
	snapshot, err := repo.DumpNativeSnapshotV1(ctx)
	if err != nil {
		return commandError(stderr, "dump", err)
	}
	if written, err := stdout.Write(snapshot); err != nil || written != len(snapshot) {
		if err == nil {
			err = io.ErrShortWrite
		}
		return commandError(stderr, "dump", fmt.Errorf("write native-blobs-v1 snapshot: %w", err))
	}
	fmt.Fprintln(stderr, "NOTICE: dump includes all retained Blobs, including opaque orphan Blobs")
	return 0
}

func runNativeLoad(ctx context.Context, workDir, file string, maxBytes int64, stdout, stderr io.Writer) int {
	input, err := readNativeSnapshotInput(workDir, file, maxBytes)
	if err != nil {
		return commandError(stderr, "load", err)
	}
	receipt, err := repository.LoadNativeSnapshotV1WithPrePublishCheck(ctx, workDir, input, func() error {
		latest, readErr := readNativeSnapshotInput(workDir, file, maxBytes)
		if readErr != nil {
			return readErr
		}
		if !bytes.Equal(input, latest) {
			return fmt.Errorf("CHANGED_DURING_READ: native snapshot file %q changed before publication", file)
		}
		return nil
	})
	if err != nil {
		return commandError(stderr, "load", err)
	}
	var loaded nativeLoadReceipt
	if err := json.Unmarshal(receipt, &loaded); err != nil {
		return nativeLoadCommittedFailure(stderr, fmt.Errorf("decode native load receipt: %w", err))
	}
	digest := sha256.Sum256(input)
	if loaded.Schema != "sealgraph/native-load/v1" || loaded.Result != "LOADED" || loaded.RepositoryFormat != 7 || loaded.SnapshotSHA256 != fmt.Sprintf("%x", digest) {
		return nativeLoadCommittedFailure(stderr, fmt.Errorf("native load receipt is inconsistent with exact input"))
	}
	if len(receipt) == 0 || receipt[len(receipt)-1] != '\n' {
		return nativeLoadCommittedFailure(stderr, fmt.Errorf("native load receipt is not canonical"))
	}
	if written, err := stdout.Write(receipt); err != nil || written != len(receipt) {
		if err == nil {
			err = io.ErrShortWrite
		}
		fmt.Fprintf(stderr, "error: sealgraph load: LOAD_COMMITTED_OUTPUT_UNDELIVERED: destination is already published; do not retry load; verify with fsck and inventory: %v\n", err)
		return 3
	}
	return 0
}

func nativeLoadCommittedFailure(stderr io.Writer, err error) int {
	fmt.Fprintf(stderr, "error: sealgraph load: LOAD_COMMITTED_OUTPUT_UNDELIVERED: destination is already published; do not retry load; verify with fsck and inventory: %v\n", err)
	return 3
}

func readNativeSnapshotInput(workDir, file string, maxBytes int64) ([]byte, error) {
	path := file
	if !filepath.IsAbs(path) {
		path = filepath.Join(workDir, path)
	}
	return readNativeSnapshotPass(path, file, maxBytes)
}

func readNativeSnapshotPass(path, displayPath string, maxBytes int64) ([]byte, error) {
	inputFile, err := fsread.OpenFileNoFollow(path)
	if err != nil {
		return nil, fmt.Errorf("open native snapshot file %q: %w", displayPath, err)
	}
	defer inputFile.Close()
	opened, err := inputFile.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat native snapshot file %q: %w", displayPath, err)
	}
	if !opened.Mode().IsRegular() {
		return nil, fmt.Errorf("native snapshot file %q is not a regular non-symlink file", displayPath)
	}
	limit := maxBytes
	if maxBytes < math.MaxInt64 {
		limit++
	}
	data, err := io.ReadAll(io.LimitReader(inputFile, limit))
	if err != nil {
		return nil, fmt.Errorf("read native snapshot file %q: %w", displayPath, err)
	}
	if int64(len(data)) > maxBytes {
		return nil, fmt.Errorf("native snapshot input exceeds --max-input-bytes %d", maxBytes)
	}
	final, err := inputFile.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat native snapshot file %q after read: %w", displayPath, err)
	}
	if opened.Size() != final.Size() || !opened.ModTime().Equal(final.ModTime()) || opened.Mode() != final.Mode() {
		return nil, fmt.Errorf("CHANGED_DURING_READ: native snapshot file %q changed while it was read", displayPath)
	}
	return data, nil
}

func parseNativeMaxInput(value string) (int64, error) {
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("--max-input-bytes requires a positive integer")
	}
	return parsed, nil
}
