package format4extract

import (
	"bytes"
	"compress/zlib"
	"crypto/sha256"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/mako10k/sealgraph/internal/domain"
)

func objectIDFromPath(path string) (domain.ObjectID, error) {
	parts := strings.Split(path, "/")
	if len(parts) != 2 || len(parts[0]) != 2 || len(parts[1]) != 62 {
		return domain.ObjectID{}, fmt.Errorf("unexpected object path %q; expected 2/62 lower-hex fanout", path)
	}
	id, err := domain.ParseObjectID(parts[0] + parts[1])
	if err != nil {
		return domain.ObjectID{}, fmt.Errorf("unexpected object path %q: %w", path, err)
	}
	return id, nil
}

func validateObjectDirectoryPath(path string) error {
	if len(path) != 2 || strings.Contains(path, "/") {
		return fmt.Errorf("unexpected object directory %q; expected one two-lower-hex fanout", path)
	}
	for _, character := range path {
		if !((character >= '0' && character <= '9') || (character >= 'a' && character <= 'f')) {
			return fmt.Errorf("unexpected object directory %q; expected one two-lower-hex fanout", path)
		}
	}
	return nil
}

func decodeLooseObject(id domain.ObjectID, physical []byte) ([]byte, error) {
	source := bytes.NewReader(physical)
	zr, err := zlib.NewReader(source)
	if err != nil {
		return nil, fmt.Errorf("object %s has invalid zlib envelope: %w", id, err)
	}
	uncompressed, readErr := io.ReadAll(zr)
	closeErr := zr.Close()
	if readErr != nil {
		return nil, fmt.Errorf("decompress object %s: %w", id, readErr)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("verify object %s compression checksum: %w", id, closeErr)
	}
	if source.Len() != 0 {
		return nil, fmt.Errorf("object %s has %d trailing compressed bytes", id, source.Len())
	}
	digest := sha256.Sum256(uncompressed)
	if fmt.Sprintf("%x", digest) != id.String() {
		return nil, fmt.Errorf("object %s hash mismatch", id)
	}
	return parseBlobEnvelope(uncompressed)
}

func parseBlobEnvelope(data []byte) ([]byte, error) {
	nul := bytes.IndexByte(data, 0)
	if nul < 0 {
		return nil, fmt.Errorf("object envelope has no NUL terminator")
	}
	payload := data[nul+1:]
	expected := domain.BlobType + " " + strconv.Itoa(len(payload))
	if string(data[:nul]) != expected {
		return nil, fmt.Errorf("invalid object header %q; expected exactly %q", data[:nul], expected)
	}
	return payload, nil
}
