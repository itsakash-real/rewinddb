package snapshot

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"

	"github.com/itsakash-real/nimbi/internal/timeline"
)

// magicCompressed is a 4-byte header that identifies gzip-compressed objects.
// Any object starting with these bytes is decompressed before JSON decoding.
// Raw JSON objects will never start with 0x1f8b (gzip magic) [web:40].
var magicCompressed = []byte{0x1f, 0x8b, 0x08, 0x00}

// marshalSnapshot serialises a Snapshot to gzip-compressed JSON.
// The stored bytes begin with the gzip magic header (0x1f8b).
func marshalSnapshot(snap *timeline.Snapshot) ([]byte, error) {
	jsonBytes, err := json.Marshal(snap)
	if err != nil {
		return nil, fmt.Errorf("marshalSnapshot: json: %w", err)
	}

	var buf bytes.Buffer
	gz, err := gzip.NewWriterLevel(&buf, gzip.BestSpeed)
	if err != nil {
		return nil, fmt.Errorf("marshalSnapshot: gzip writer: %w", err)
	}
	if _, err := gz.Write(jsonBytes); err != nil {
		gz.Close()
		return nil, fmt.Errorf("marshalSnapshot: gzip write: %w", err)
	}
	if err := gz.Close(); err != nil {
		return nil, fmt.Errorf("marshalSnapshot: gzip close: %w", err)
	}
	return buf.Bytes(), nil
}

// unmarshalSnapshot decodes a Snapshot from either:
//   - gzip-compressed JSON (magic header present)
//   - raw uncompressed JSON (backward-compatible with existing stores)
func unmarshalSnapshot(data []byte) (*timeline.Snapshot, error) {
	reader := io.Reader(bytes.NewReader(data))

	if isGzipCompressed(data) {
		gz, err := gzip.NewReader(bytes.NewReader(data))
		if err != nil {
			return nil, fmt.Errorf("unmarshalSnapshot: gzip reader: %w", err)
		}
		defer gz.Close()
		reader = gz
	}

	var snap timeline.Snapshot
	if err := json.NewDecoder(reader).Decode(&snap); err != nil {
		return nil, fmt.Errorf("unmarshalSnapshot: decode: %w", err)
	}
	return &snap, nil
}

// isGzipCompressed checks the leading bytes for the gzip magic number.
func isGzipCompressed(data []byte) bool {
	return len(data) >= 2 && data[0] == 0x1f && data[1] == 0x8b
}
