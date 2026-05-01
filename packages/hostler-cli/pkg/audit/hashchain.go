// Package audit — Audit Events Hash Chain.
//
// Each audit_event record carries prev_hash + event_hash; chain integrity
// provides tamper evidence. hash = SHA256(prev_hash || canonical_json(event)).
// canonical_json serializes id/timestamp/event_type/entity_type/entity_id/
// actor/details/session_id in a fixed order.
package audit

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/ports"
)

// CanonicalEventJSON produces deterministic JSON for hash computation.
// Unlike the JSON spec it sorts keys, strips whitespace, and orders by
// UTF-8 byte sequence. The details map is sorted alphabetically by key,
// and nested maps are sorted recursively.
//
// trac: TRC-OP004
func CanonicalEventJSON(e ports.AuditEvent) string {
	var sb strings.Builder
	sb.WriteString(`{"id":`)
	sb.WriteString(fmt.Sprintf("%d", e.ID))
	sb.WriteString(`,"timestamp":`)
	sb.WriteString(quote(e.Timestamp))
	sb.WriteString(`,"event_type":`)
	sb.WriteString(quote(e.EventType))
	sb.WriteString(`,"entity_type":`)
	sb.WriteString(quote(e.EntityType))
	sb.WriteString(`,"entity_id":`)
	sb.WriteString(quote(e.EntityID))
	sb.WriteString(`,"actor":`)
	sb.WriteString(quote(e.ActorID))
	sb.WriteString(`,"session_id":`)
	sb.WriteString(quote(e.SessionID))
	sb.WriteString(`,"details":`)
	sb.WriteString(canonicalValue(e.Details))
	sb.WriteString(`}`)
	return sb.String()
}

// CalculateHash combines prev_hash and the event into a SHA256 hex string.
// For the first event (prev_hash == "") only the canonical event payload
// is hashed.
//
// trac: TRC-OP004
func CalculateHash(prevHash string, e ports.AuditEvent) string {
	h := sha256.New()
	if prevHash != "" {
		h.Write([]byte(prevHash))
	}
	h.Write([]byte(CanonicalEventJSON(e)))
	return hex.EncodeToString(h.Sum(nil))
}

func quote(s string) string {
	var sb strings.Builder
	sb.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			sb.WriteString(`\"`)
		case '\\':
			sb.WriteString(`\\`)
		case '\n':
			sb.WriteString(`\n`)
		case '\r':
			sb.WriteString(`\r`)
		case '\t':
			sb.WriteString(`\t`)
		default:
			if r < 0x20 {
				sb.WriteString(fmt.Sprintf(`\u%04x`, r))
			} else {
				sb.WriteRune(r)
			}
		}
	}
	sb.WriteByte('"')
	return sb.String()
}

func canonicalMap(m map[string]any) string {
	if m == nil {
		return "null"
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var sb strings.Builder
	sb.WriteByte('{')
	for i, k := range keys {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString(quote(k))
		sb.WriteByte(':')
		sb.WriteString(canonicalValue(m[k]))
	}
	sb.WriteByte('}')
	return sb.String()
}

func canonicalValue(v any) string {
	switch x := v.(type) {
	case nil:
		return "null"
	case string:
		return quote(x)
	case bool:
		if x {
			return "true"
		}
		return "false"
	case int, int32, int64:
		return fmt.Sprintf("%d", x)
	case float32, float64:
		return fmt.Sprintf("%v", x)
	case map[string]any:
		return canonicalMap(x)
	case []any:
		var sb strings.Builder
		sb.WriteByte('[')
		for i, it := range x {
			if i > 0 {
				sb.WriteByte(',')
			}
			sb.WriteString(canonicalValue(it))
		}
		sb.WriteByte(']')
		return sb.String()
	default:
		return quote(fmt.Sprintf("%v", v))
	}
}
