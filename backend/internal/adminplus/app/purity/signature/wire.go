package signature

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

const (
	maxEncodedLength = 128 * 1024
	maxDecodedLength = 96 * 1024
	maxFields        = 512
	maxDepth         = 4
)

var (
	errSignatureEmpty      = errors.New("signature is empty")
	errSignatureTooLarge   = errors.New("signature exceeds size limit")
	errInvalidWireMessage  = errors.New("invalid protobuf wire message")
	errWireDepthExceeded   = errors.New("protobuf nesting exceeds limit")
	errWireFieldsExceeded  = errors.New("protobuf field count exceeds limit")
	errInvalidFieldNumber  = errors.New("protobuf field number must be positive")
	errUnsupportedWireType = errors.New("unsupported protobuf wire type")
)

type Fingerprint struct {
	DecodedLengthBucket string            `json:"decoded_length_bucket"`
	TopLevelFields      []int             `json:"top_level_fields"`
	EnvelopeFields      []int             `json:"envelope_fields,omitempty"`
	MetadataFields      []int             `json:"metadata_fields,omitempty"`
	MetadataValueTypes  map[string]string `json:"metadata_value_types,omitempty"`
	DedupHash           string            `json:"-"`
}

type wireMessage struct {
	fields []wireField
}

type wireField struct {
	number   int
	wireType int
	value    []byte
	child    *wireMessage
}

func Analyze(encoded string) (Fingerprint, error) {
	raw, err := decode(encoded)
	if err != nil {
		return Fingerprint{}, err
	}
	message, err := parseMessage(raw, 0)
	if err != nil {
		return Fingerprint{}, err
	}
	fingerprint := buildFingerprint(message, len(raw))
	hash := sha256.Sum256(raw)
	fingerprint.DedupHash = hex.EncodeToString(hash[:6])
	return fingerprint, nil
}

func decode(encoded string) ([]byte, error) {
	value := strings.TrimSpace(encoded)
	if value == "" {
		return nil, errSignatureEmpty
	}
	if len(value) > maxEncodedLength {
		return nil, errSignatureTooLarge
	}
	encodings := []*base64.Encoding{
		base64.StdEncoding.Strict(),
		base64.RawStdEncoding.Strict(),
		base64.URLEncoding.Strict(),
		base64.RawURLEncoding.Strict(),
	}
	var lastErr error
	for _, encoding := range encodings {
		decoded, err := encoding.DecodeString(value)
		if err != nil {
			lastErr = err
			continue
		}
		if len(decoded) == 0 {
			return nil, errSignatureEmpty
		}
		if len(decoded) > maxDecodedLength {
			return nil, errSignatureTooLarge
		}
		return decoded, nil
	}
	return nil, fmt.Errorf("decode signature: %w", lastErr)
}

func parseMessage(data []byte, depth int) (*wireMessage, error) {
	if depth > maxDepth {
		return nil, errWireDepthExceeded
	}
	if len(data) == 0 {
		return nil, errInvalidWireMessage
	}
	message := &wireMessage{fields: make([]wireField, 0, 8)}
	for offset := 0; offset < len(data); {
		if len(message.fields) >= maxFields {
			return nil, errWireFieldsExceeded
		}
		key, consumed, err := readVarint(data[offset:])
		if err != nil {
			return nil, err
		}
		offset += consumed
		number := int(key >> 3)
		if number <= 0 {
			return nil, errInvalidFieldNumber
		}
		wireType := int(key & 0x7)
		field := wireField{number: number, wireType: wireType}
		switch wireType {
		case 0:
			_, consumed, err = readVarint(data[offset:])
			if err != nil {
				return nil, err
			}
			offset += consumed
		case 1:
			if len(data)-offset < 8 {
				return nil, errInvalidWireMessage
			}
			offset += 8
		case 2:
			length, lengthBytes, lengthErr := readVarint(data[offset:])
			if lengthErr != nil {
				return nil, lengthErr
			}
			offset += lengthBytes
			if length > uint64(len(data)-offset) {
				return nil, errInvalidWireMessage
			}
			end := offset + int(length)
			field.value = data[offset:end]
			if depth < maxDepth && len(field.value) > 0 {
				if child, childErr := parseMessage(field.value, depth+1); childErr == nil {
					field.child = child
				}
			}
			offset = end
		case 5:
			if len(data)-offset < 4 {
				return nil, errInvalidWireMessage
			}
			offset += 4
		default:
			return nil, errUnsupportedWireType
		}
		message.fields = append(message.fields, field)
	}
	if len(message.fields) == 0 {
		return nil, errInvalidWireMessage
	}
	return message, nil
}

func readVarint(data []byte) (uint64, int, error) {
	var value uint64
	for index := 0; index < 10; index++ {
		if index >= len(data) {
			return 0, 0, errInvalidWireMessage
		}
		current := data[index]
		if index == 9 && current > 1 {
			return 0, 0, errInvalidWireMessage
		}
		value |= uint64(current&0x7f) << (7 * index)
		if current < 0x80 {
			return value, index + 1, nil
		}
	}
	return 0, 0, errInvalidWireMessage
}

func buildFingerprint(root *wireMessage, decodedLength int) Fingerprint {
	envelope := bestChild(root, envelopeScore)
	metadata := bestDescendant(envelope, metadataScore)
	if envelope == nil {
		envelope = root
	}
	if metadata == nil {
		metadata = envelope
	}
	return Fingerprint{
		DecodedLengthBucket: lengthBucket(decodedLength),
		TopLevelFields:      uniqueFields(root),
		EnvelopeFields:      uniqueFields(envelope),
		MetadataFields:      uniqueFields(metadata),
		MetadataValueTypes:  valueTypes(metadata),
	}
}

func bestChild(message *wireMessage, score func(*wireMessage) int) *wireMessage {
	if message == nil {
		return nil
	}
	var best *wireMessage
	bestScore := -1
	for _, field := range message.fields {
		if field.child == nil {
			continue
		}
		if current := score(field.child); current > bestScore {
			best = field.child
			bestScore = current
		}
	}
	return best
}

func bestDescendant(message *wireMessage, score func(*wireMessage) int) *wireMessage {
	if message == nil {
		return nil
	}
	var best *wireMessage
	bestScore := -1
	var walk func(*wireMessage, int)
	walk = func(current *wireMessage, depth int) {
		if current == nil || depth > maxDepth {
			return
		}
		if depth > 0 {
			if currentScore := score(current); currentScore > bestScore {
				best = current
				bestScore = currentScore
			}
		}
		for _, field := range current.fields {
			walk(field.child, depth+1)
		}
	}
	walk(message, 0)
	return best
}

func envelopeScore(message *wireMessage) int {
	fields := uniqueFields(message)
	return overlapScore(fields, []int{1, 2, 3, 4, 5})*10 + len(fields)
}

func metadataScore(message *wireMessage) int {
	fields := uniqueFields(message)
	score := overlapScore(fields, []int{1, 2, 3, 5, 6, 7, 8, 11})*10 + len(fields)
	for _, field := range message.fields {
		text := strings.ToLower(string(field.value))
		if strings.Contains(text, "claude") {
			score += 25
		}
		if text == "thinking" || text == "redacted_thinking" {
			score += 25
		}
	}
	return score
}

func overlapScore(actual []int, expected []int) int {
	set := make(map[int]struct{}, len(actual))
	for _, value := range actual {
		set[value] = struct{}{}
	}
	count := 0
	for _, value := range expected {
		if _, ok := set[value]; ok {
			count++
		}
	}
	return count
}

func uniqueFields(message *wireMessage) []int {
	if message == nil {
		return nil
	}
	seen := make(map[int]struct{}, len(message.fields))
	for _, field := range message.fields {
		seen[field.number] = struct{}{}
	}
	out := make([]int, 0, len(seen))
	for number := range seen {
		out = append(out, number)
	}
	sort.Ints(out)
	return out
}

func valueTypes(message *wireMessage) map[string]string {
	if message == nil {
		return nil
	}
	out := make(map[string]string)
	for _, field := range message.fields {
		key := strconv.Itoa(field.number)
		valueType := wireValueType(field)
		if previous, ok := out[key]; ok && previous != valueType {
			out[key] = "mixed"
			continue
		}
		out[key] = valueType
	}
	return out
}

func wireValueType(field wireField) string {
	switch field.wireType {
	case 0:
		return "varint"
	case 1:
		return "fixed64"
	case 5:
		return "fixed32"
	case 2:
		if field.child != nil {
			return "message"
		}
		if utf8.Valid(field.value) && printableText(field.value) {
			text := strings.ToLower(strings.TrimSpace(string(field.value)))
			if strings.Contains(text, "claude") {
				return "text:model"
			}
			if text == "thinking" || text == "redacted_thinking" {
				return "text:block_type"
			}
			return "text"
		}
		return "bytes:" + lengthBucket(len(field.value))
	default:
		return "unknown"
	}
}

func printableText(value []byte) bool {
	for _, current := range string(value) {
		if current < 0x20 && current != '\n' && current != '\r' && current != '\t' {
			return false
		}
	}
	return true
}

func lengthBucket(length int) string {
	switch {
	case length < 32:
		return "0-31"
	case length < 64:
		return "32-63"
	case length < 128:
		return "64-127"
	case length < 256:
		return "128-255"
	case length < 512:
		return "256-511"
	case length < 1024:
		return "512-1023"
	default:
		return "1024+"
	}
}
