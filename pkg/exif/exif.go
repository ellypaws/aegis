// Package exif writes UTF-8 text key/values into PNG metadata while encoding.
// It wraps image/png and inserts iTXt chunks immediately after IHDR.
package exif

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"hash/crc32"
	"image"
	"image/png"
	"io"
	"unicode/utf8"
)

const pngSig = "\x89PNG\r\n\x1a\n"

type Encoder[T any] struct {
	PNG png.Encoder
	// Key is the iTXt keyword used for the JSON payload.
	// Must be ASCII printable and <= 79 bytes. Default is "JSON" if empty.
	Key  string
	Data T
}

func NewEncoder[T any](data T) *Encoder[T] {
	return &Encoder[T]{PNG: png.Encoder{}, Data: data}
}

func (e *Encoder[T]) Put(data T) {
	e.Data = data
}

func (e *Encoder[T]) EncodeReader(w io.Writer, r io.Reader) error {
	m, _, err := image.Decode(r)
	if err != nil {
		return err
	}
	return e.Encode(w, m)
}

func (e *Encoder[T]) Encode(w io.Writer, m image.Image) error {
	var buf bytes.Buffer
	if err := e.PNG.Encode(&buf, m); err != nil {
		return err
	}

	// Marshal the generic payload to JSON.
	var payload []byte
	var err error
	payload, err = json.Marshal(e.Data)
	if err != nil {
		return err
	}
	if len(payload) == 0 || string(payload) == "null" {
		_, err := io.Copy(w, &buf)
		return err
	}

	// If no metadata key provided, use a sane default.
	key := e.Key
	if key == "" {
		key = "JSON"
	}

	// If the key is invalid or payload is not valid UTF-8 (should be), just copy through.
	if !isASCIIPrintable(key) || len(key) > 79 || !utf8.Valid(payload) {
		_, err := io.Copy(w, &buf)
		return err
	}

	return injectITXtAfterIHDRJSON(&buf, w, key, string(payload))
}

func injectITXtAfterIHDRJSON(src *bytes.Buffer, dst io.Writer, key, jsonText string) error {
	data := src.Bytes()
	if len(data) < 8 || string(data[:8]) != pngSig {
		_, err := dst.Write(data)
		return err
	}
	if _, err := dst.Write(data[:8]); err != nil {
		return err
	}
	off := 8

	if off+8 > len(data) {
		_, err := dst.Write(data[8:])
		return err
	}
	clen := int(binary.BigEndian.Uint32(data[off : off+4]))
	end := off + 8 + clen + 4
	if end > len(data) {
		_, err := dst.Write(data[8:])
		return err
	}
	if _, err := dst.Write(data[off:end]); err != nil {
		return err
	}
	off = end

	if ch, ok := buildITXtChunk(key, jsonText); ok {
		if _, err := dst.Write(ch); err != nil {
			return err
		}
	}

	_, err := dst.Write(data[off:])
	return err
}

// iTXt (uncompressed) layout:
// keyword (1-79 bytes Latin-1) + 0
// compression_flag (1 byte = 0)
// compression_method (1 byte = 0)
// language_tag (zero bytes, then 0)
// translated_keyword (zero bytes, then 0)
// text (UTF-8 bytes)
func buildITXtChunk(key, text string) ([]byte, bool) {
	if key == "" || len(key) > 79 {
		return nil, false
	}
	if !isASCIIPrintable(key) {
		return nil, false
	}
	if !utf8.ValidString(text) {
		return nil, false
	}

	var payload bytes.Buffer
	payload.WriteString(key)
	payload.WriteByte(0)
	payload.WriteByte(0)
	payload.WriteByte(0)
	payload.WriteByte(0)
	payload.WriteByte(0)
	payload.WriteString(text)

	return makeChunk("iTXt", payload.Bytes()), true
}

func makeChunk(kind string, payload []byte) []byte {
	var b bytes.Buffer
	var lenbuf [4]byte
	binary.BigEndian.PutUint32(lenbuf[:], uint32(len(payload)))
	b.Write(lenbuf[:])

	typ := []byte(kind)
	b.Write(typ)
	b.Write(payload)

	crc := crc32.ChecksumIEEE(append(typ, payload...))
	binary.BigEndian.PutUint32(lenbuf[:], crc)
	b.Write(lenbuf[:])

	return b.Bytes()
}

func isASCIIPrintable(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c < 32 || c > 126 {
			return false
		}
	}
	return true
}
