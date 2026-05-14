package markdown

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// FrontmatterFile is a markdown file that begins with a YAML frontmatter
// block (`---\n...\n---\n`) followed by the markdown body. It is the
// storage format for backlog items and users.
//
// Why typed-Get/Set on yaml.Node rather than a Go struct:
//   - field set varies by file kind (item / user / ...)
//   - users may add extra keys; round-tripping must preserve them
//   - ordering and comments survive
//
// Use the typed helpers (GetString, GetStringSlice, GetTimeline, ...) at
// call sites; the struct keeps the parsed YAML as a *yaml.Node so any
// caller that needs it can manipulate the tree directly.
type FrontmatterFile struct {
	path    string
	root    *yaml.Node // mapping node, top-level frontmatter
	body    string
	dirty   bool
	hasFM   bool
}

// LoadFrontmatter reads `path`. If the file is absent, returns a fresh
// FrontmatterFile bound to that path with empty frontmatter and body.
func LoadFrontmatter(path string) (*FrontmatterFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &FrontmatterFile{path: path, root: emptyMapping(), hasFM: true}, nil
		}
		return nil, err
	}
	f, err := ParseFrontmatter(string(data))
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	f.path = path
	return f, nil
}

// ParseFrontmatter parses raw markdown source.
func ParseFrontmatter(src string) (*FrontmatterFile, error) {
	f := &FrontmatterFile{root: emptyMapping(), hasFM: true}

	src = strings.TrimPrefix(src, "\ufeff")
	if !strings.HasPrefix(src, "---\n") && !strings.HasPrefix(src, "---\r\n") {
		// no frontmatter; treat all as body
		f.body = src
		return f, nil
	}

	// find closing fence: a line that is exactly `---`
	rest := src[strings.Index(src, "\n")+1:]
	end := -1
	idx := 0
	for {
		nl := strings.Index(rest[idx:], "\n")
		var line string
		if nl < 0 {
			line = rest[idx:]
		} else {
			line = rest[idx : idx+nl]
		}
		trim := strings.TrimRight(line, "\r")
		if trim == "---" || trim == "..." {
			end = idx + len(line)
			break
		}
		if nl < 0 {
			break
		}
		idx += nl + 1
	}
	if end < 0 {
		return nil, fmt.Errorf("frontmatter opener `---` has no closer")
	}
	yamlBlock := rest[:end-len("---")]
	bodyStart := end
	if bodyStart < len(rest) && rest[bodyStart] == '\n' {
		bodyStart++
	}
	if bodyStart < len(rest) && rest[bodyStart] == '\n' {
		bodyStart++
	}
	body := ""
	if bodyStart <= len(rest) {
		body = rest[bodyStart:]
	}

	var root yaml.Node
	if strings.TrimSpace(yamlBlock) != "" {
		if err := yaml.Unmarshal([]byte(yamlBlock), &root); err != nil {
			return nil, fmt.Errorf("yaml: %w", err)
		}
	}
	if root.Kind == yaml.DocumentNode && len(root.Content) > 0 {
		f.root = root.Content[0]
	}
	if f.root == nil || f.root.Kind != yaml.MappingNode {
		f.root = emptyMapping()
	}
	f.body = body
	return f, nil
}

func emptyMapping() *yaml.Node {
	return &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
}

// Path returns the bound file path.
func (f *FrontmatterFile) Path() string { return f.path }

// SetPath rebinds the file path (used when items move between dirs).
func (f *FrontmatterFile) SetPath(p string) { f.path = p }

// Dirty returns true when the in-memory state diverges from disk.
func (f *FrontmatterFile) Dirty() bool { return f.dirty }

// Body returns the markdown body (everything after the closing `---`).
func (f *FrontmatterFile) Body() string { return f.body }

// SetBody replaces the body. Adds a trailing newline if missing.
func (f *FrontmatterFile) SetBody(body string) {
	if body != "" && !strings.HasSuffix(body, "\n") {
		body += "\n"
	}
	if f.body != body {
		f.body = body
		f.dirty = true
	}
}

// Bytes returns the serialized file: frontmatter + body.
func (f *FrontmatterFile) Bytes() []byte {
	var buf bytes.Buffer
	buf.WriteString("---\n")
	if f.root != nil && len(f.root.Content) > 0 {
		enc := yaml.NewEncoder(&buf)
		enc.SetIndent(2)
		_ = enc.Encode(f.root)
		_ = enc.Close()
	}
	buf.WriteString("---\n")
	if f.body != "" {
		if !strings.HasPrefix(f.body, "\n") {
			buf.WriteString("\n")
		}
		buf.WriteString(f.body)
		if !strings.HasSuffix(f.body, "\n") {
			buf.WriteString("\n")
		}
	}
	return buf.Bytes()
}

// Save writes Bytes() to Path() if dirty. No-op when path is empty.
func (f *FrontmatterFile) Save() error {
	if f.path == "" || !f.dirty {
		return nil
	}
	if err := os.WriteFile(f.path, f.Bytes(), 0644); err != nil {
		return err
	}
	f.dirty = false
	return nil
}

// MarkDirty forces the next Save to write even when no setter was called.
func (f *FrontmatterFile) MarkDirty() { f.dirty = true }

// findKey returns the mapping pair (key, value) yaml.Nodes for `key`,
// or (nil,nil) if missing.
func (f *FrontmatterFile) findKey(key string) (*yaml.Node, *yaml.Node) {
	if f.root == nil {
		return nil, nil
	}
	for i := 0; i+1 < len(f.root.Content); i += 2 {
		k := f.root.Content[i]
		if k.Value == key {
			return k, f.root.Content[i+1]
		}
	}
	return nil, nil
}

// setNode upserts `key` with the given value node, preserving order on update.
func (f *FrontmatterFile) setNode(key string, value *yaml.Node) {
	if f.root == nil {
		f.root = emptyMapping()
	}
	for i := 0; i+1 < len(f.root.Content); i += 2 {
		if f.root.Content[i].Value == key {
			f.root.Content[i+1] = value
			f.dirty = true
			return
		}
	}
	f.root.Content = append(f.root.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
		value,
	)
	f.dirty = true
}

// removeKey deletes `key` from the frontmatter.
func (f *FrontmatterFile) removeKey(key string) {
	if f.root == nil {
		return
	}
	for i := 0; i+1 < len(f.root.Content); i += 2 {
		if f.root.Content[i].Value == key {
			f.root.Content = append(f.root.Content[:i], f.root.Content[i+2:]...)
			f.dirty = true
			return
		}
	}
}

// GetString returns the scalar string value at `key`, or "" if missing.
func (f *FrontmatterFile) GetString(key string) string {
	_, v := f.findKey(key)
	if v == nil {
		return ""
	}
	if v.Kind == yaml.ScalarNode {
		return v.Value
	}
	return ""
}

// SetString sets `key` to a scalar string. Removes the key if value is "".
func (f *FrontmatterFile) SetString(key, value string) {
	if value == "" {
		f.removeKey(key)
		return
	}
	f.setNode(key, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value})
}

// GetStringSlice returns a list of strings at `key`, accepting either a
// YAML sequence or a single scalar.
func (f *FrontmatterFile) GetStringSlice(key string) []string {
	_, v := f.findKey(key)
	if v == nil {
		return nil
	}
	if v.Kind == yaml.ScalarNode {
		s := strings.TrimSpace(v.Value)
		if s == "" {
			return nil
		}
		return []string{s}
	}
	if v.Kind == yaml.SequenceNode {
		out := make([]string, 0, len(v.Content))
		for _, n := range v.Content {
			if n.Kind == yaml.ScalarNode {
				out = append(out, n.Value)
			}
		}
		return out
	}
	return nil
}

// SetStringSlice writes a YAML flow sequence (`[a, b, c]`) at `key`.
// Removes the key when items is empty.
func (f *FrontmatterFile) SetStringSlice(key string, items []string) {
	if len(items) == 0 {
		f.removeKey(key)
		return
	}
	seq := &yaml.Node{Kind: yaml.SequenceNode, Style: yaml.FlowStyle, Tag: "!!seq"}
	for _, item := range items {
		seq.Content = append(seq.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: item})
	}
	f.setNode(key, seq)
}

// GetBool returns the boolean value at `key`, or false if missing/non-bool.
func (f *FrontmatterFile) GetBool(key string) bool {
	_, v := f.findKey(key)
	if v == nil || v.Kind != yaml.ScalarNode {
		return false
	}
	switch strings.ToLower(v.Value) {
	case "true", "yes", "on", "1":
		return true
	}
	return false
}

// SetBool writes a boolean at `key`. False removes the key.
func (f *FrontmatterFile) SetBool(key string, value bool) {
	if !value {
		f.removeKey(key)
		return
	}
	f.setNode(key, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!bool", Value: "true"})
}

// GetMap returns a string->string view of the mapping value at `key`.
// Returns nil when the key is missing or not a mapping. Order preserved.
func (f *FrontmatterFile) GetMap(key string) []KV {
	_, v := f.findKey(key)
	if v == nil || v.Kind != yaml.MappingNode {
		return nil
	}
	out := make([]KV, 0, len(v.Content)/2)
	for i := 0; i+1 < len(v.Content); i += 2 {
		k := v.Content[i].Value
		val := ""
		if v.Content[i+1].Kind == yaml.ScalarNode {
			val = v.Content[i+1].Value
		}
		out = append(out, KV{Key: k, Value: val})
	}
	return out
}

// SetMap upserts a mapping value at `key`. Empty entries delete the key.
func (f *FrontmatterFile) SetMap(key string, entries []KV) {
	if len(entries) == 0 {
		f.removeKey(key)
		return
	}
	m := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	for _, kv := range entries {
		m.Content = append(m.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: kv.Key},
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: kv.Value},
		)
	}
	f.setNode(key, m)
}

// KV is a frontmatter mapping entry.
type KV struct {
	Key   string
	Value string
}

// HasKey reports whether the frontmatter contains `key`.
func (f *FrontmatterFile) HasKey(key string) bool {
	k, _ := f.findKey(key)
	return k != nil
}

// Remove deletes a key.
func (f *FrontmatterFile) Remove(key string) {
	f.removeKey(key)
}

// Keys returns all top-level frontmatter keys in document order.
func (f *FrontmatterFile) Keys() []string {
	if f.root == nil {
		return nil
	}
	out := make([]string, 0, len(f.root.Content)/2)
	for i := 0; i+1 < len(f.root.Content); i += 2 {
		out = append(out, f.root.Content[i].Value)
	}
	return out
}
