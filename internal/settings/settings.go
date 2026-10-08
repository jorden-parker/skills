// Package settings edits the skillOverrides key in Claude's settings.json
// without disturbing anything else in the file.
package settings

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	"github.com/jorden-parker/skills/internal/atomicfile"
)

const key = "skillOverrides"

// ReadOverrides returns the skillOverrides map. A missing or unreadable file,
// or one that is not a JSON object, yields an empty map.
func ReadOverrides(path string) map[string]string {
	out := map[string]string{}
	b, err := os.ReadFile(path)
	if err != nil {
		return out
	}
	var doc map[string]any
	if json.Unmarshal(b, &doc) != nil {
		return out
	}
	ov, _ := doc[key].(map[string]any)
	for k, v := range ov {
		if s, ok := v.(string); ok {
			out[k] = s
		}
	}
	return out
}

// WriteOverrides merges states into skillOverrides. Entries that are on or
// off are removed, and the key itself goes when nothing partial remains.
// The parent directory is created, a missing file starts as {}, a copy of
// the previous file is left at path.bak, and the new file is written
// atomically with every other key in its original order.
func WriteOverrides(path string, states map[string]string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		b = []byte("{}\n")
		if err := os.WriteFile(path, b, 0o644); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	if err := os.WriteFile(path+".bak", b, 0o644); err != nil {
		return err
	}
	var doc map[string]any
	if err := json.Unmarshal(b, &doc); err != nil {
		return err
	}
	if doc == nil {
		doc = map[string]any{}
	}
	order := map[string][]string{}
	_ = scanOrder(json.NewDecoder(bytes.NewReader(b)), "", order)

	ov, _ := doc[key].(map[string]any)
	if ov == nil {
		ov = map[string]any{}
	}
	for k, v := range states {
		ov[k] = v
	}
	for k, v := range ov {
		if s, ok := v.(string); !ok || s == "on" || s == "off" {
			delete(ov, k)
		}
	}
	if len(ov) == 0 {
		delete(doc, key)
	} else {
		doc[key] = ov
	}
	var buf bytes.Buffer
	if err := encode(&buf, doc, "", "", order); err != nil {
		return err
	}
	buf.WriteByte('\n')
	return atomicfile.Write(path, buf.Bytes())
}

// scanOrder records the key order of every object in a JSON document.
func scanOrder(dec *json.Decoder, path string, order map[string][]string) error {
	t, err := dec.Token()
	if err != nil {
		return err
	}
	switch t {
	case json.Delim('{'):
		for dec.More() {
			k, err := dec.Token()
			if err != nil {
				return err
			}
			key, _ := k.(string)
			order[path] = append(order[path], key)
			if err := scanOrder(dec, child(path, key), order); err != nil {
				return err
			}
		}
		_, err = dec.Token()
	case json.Delim('['):
		for i := 0; dec.More(); i++ {
			if err := scanOrder(dec, child(path, strconv.Itoa(i)), order); err != nil {
				return err
			}
		}
		_, err = dec.Token()
	}
	return err
}

func child(path, key string) string { return path + "\x00" + key }

// encode writes v the way json.MarshalIndent does, except that an object's
// keys keep the order they were read in. Keys added since go last, sorted.
func encode(b *bytes.Buffer, v any, path, indent string, order map[string][]string) error {
	switch x := v.(type) {
	case map[string]any:
		if len(x) == 0 {
			b.WriteString("{}")
			return nil
		}
		var keys, added []string
		seen := map[string]bool{}
		for _, k := range order[path] {
			if _, ok := x[k]; ok && !seen[k] {
				keys, seen[k] = append(keys, k), true
			}
		}
		for k := range x {
			if !seen[k] {
				added = append(added, k)
			}
		}
		sort.Strings(added)
		keys = append(keys, added...)
		b.WriteString("{\n")
		for i, k := range keys {
			name, err := json.Marshal(k)
			if err != nil {
				return err
			}
			b.WriteString(indent + "  ")
			b.Write(name)
			b.WriteString(": ")
			if err := encode(b, x[k], child(path, k), indent+"  ", order); err != nil {
				return err
			}
			if i < len(keys)-1 {
				b.WriteByte(',')
			}
			b.WriteByte('\n')
		}
		b.WriteString(indent + "}")
	case []any:
		if len(x) == 0 {
			b.WriteString("[]")
			return nil
		}
		b.WriteString("[\n")
		for i, item := range x {
			b.WriteString(indent + "  ")
			if err := encode(b, item, child(path, strconv.Itoa(i)), indent+"  ", order); err != nil {
				return err
			}
			if i < len(x)-1 {
				b.WriteByte(',')
			}
			b.WriteByte('\n')
		}
		b.WriteString(indent + "]")
	default:
		out, err := json.MarshalIndent(v, indent, "  ")
		if err != nil {
			return err
		}
		b.Write(out)
	}
	return nil
}
