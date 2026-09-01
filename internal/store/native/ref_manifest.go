package native

import "github.com/mako10k/sealgraph/internal/store/refmanifest"

const refManifestSchema = refmanifest.Schema

type refManifest = refmanifest.Manifest

func encodeRefManifest(input refManifest) ([]byte, error) {
	return refmanifest.Encode(input)
}

func decodeRefManifest(data []byte) (refManifest, error) {
	return refmanifest.Decode(data)
}
