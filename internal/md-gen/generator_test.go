package mdgen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	jsonschemaparser "github.com/sighupio/md-gen/internal/json-schema-parser"
)

func TestResolveFileRef(t *testing.T) {
	dir := t.TempDir()

	shared := `{"type": "string", "$defs": {"Types.Quantity": {"type": "string", "pattern": "^[0-9]+$"}}}`
	if err := os.WriteFile(filepath.Join(dir, "shared.json"), []byte(shared), 0o600); err != nil {
		t.Fatal(err)
	}

	g := NewBaseGenerator("", nil, dir)

	whole, src, err := g.resolveFileRef("./shared.json")
	if err != nil || whole.Type[0] != "string" || src != "./shared.json" {
		t.Fatalf("whole file: got %v, %q, %v", whole, src, err)
	}

	def, src, err := g.resolveFileRef("./shared.json#/$defs/Types.Quantity")
	if err != nil || def.Pattern != "^[0-9]+$" || src != "./shared.json" {
		t.Fatalf("definition: got %v, %q, %v", def, src, err)
	}

	for _, ref := range []string{"./shared.json#/$defs/Missing", "./shared.json#/properties/x"} {
		if _, _, err := g.resolveFileRef(ref); err == nil {
			t.Errorf("%s: expected an error", ref)
		}
	}
}

func TestGeneratePrefersFieldDescriptionOverRefDescription(t *testing.T) {
	dir := t.TempDir()

	root := `{
  "type": "object",
  "properties": {
    "size": {"$ref": "#/$defs/Quantity", "description": "The size of each disk."},
    "sizes": {"type": "array", "description": "The size of each volume.", "items": {"$ref": "#/$defs/Quantity"}},
    "limit": {"$ref": "#/$defs/Quantity"}
  },
  "$defs": {"Quantity": {"type": "string", "description": "A Kubernetes quantity."}}
}`

	path := filepath.Join(dir, "root.json")
	if err := os.WriteFile(path, []byte(root), 0o600); err != nil {
		t.Fatal(err)
	}

	schema, err := jsonschemaparser.NewBaseParser(path).Parse()
	if err != nil {
		t.Fatal(err)
	}

	out, err := NewBaseGenerator("", schema, dir).Generate()
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{"The size of each disk.", "The size of each volume.", "A Kubernetes quantity."} {
		if !strings.Contains(string(out), want) {
			t.Errorf("output does not contain %q:\n%s", want, out)
		}
	}

	// the type description stays only where no field description is closer
	if n := strings.Count(string(out), "A Kubernetes quantity."); n != 1 {
		t.Errorf("type description appears %d times, want 1:\n%s", n, out)
	}
}
