package mdgen

import (
	"os"
	"path/filepath"
	"testing"
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
