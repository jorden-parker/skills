package settings

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestWriteCreatesFileAndBackup(t *testing.T) {
	path := filepath.Join(t.TempDir(), "claude", "settings.json")
	if err := WriteOverrides(path, map[string]string{"alpha": "name-only", "beta": "on"}); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(path)
	want := "{\n  \"skillOverrides\": {\n    \"alpha\": \"name-only\"\n  }\n}\n"
	if string(got) != want {
		t.Fatalf("got %q", got)
	}
	bak, _ := os.ReadFile(path + ".bak")
	if string(bak) != "{}\n" {
		t.Fatalf("backup = %q", bak)
	}
	if ov := ReadOverrides(path); !reflect.DeepEqual(ov, map[string]string{"alpha": "name-only"}) {
		t.Fatalf("overrides = %v", ov)
	}
}

func TestWriteDropsOnOffAndEmptyKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := WriteOverrides(path, map[string]string{"alpha": "name-only", "beta": "user-invocable-only"}); err != nil {
		t.Fatal(err)
	}
	if err := WriteOverrides(path, map[string]string{"alpha": "off", "beta": "on"}); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(path)
	if string(got) != "{}\n" {
		t.Fatalf("got %q", got)
	}
	bak, _ := os.ReadFile(path + ".bak")
	if string(bak) == "{}\n" {
		t.Fatal("backup is not the previous file")
	}
}

func TestWritePreservesOtherKeysAndOrder(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	body := "{\n  \"zeta\": [1, {\"b\": 1, \"a\": 2}],\n  \"skillOverrides\": {\"old\": \"name-only\"},\n  \"alpha\": \"x\"\n}\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := WriteOverrides(path, map[string]string{"new": "user-invocable-only"}); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(path)
	want := "{\n  \"zeta\": [\n    1,\n    {\n      \"b\": 1,\n      \"a\": 2\n    }\n  ],\n  \"skillOverrides\": {\n    \"old\": \"name-only\",\n    \"new\": \"user-invocable-only\"\n  },\n  \"alpha\": \"x\"\n}\n"
	if string(got) != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestReadInvalidJSONIsEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if ov := ReadOverrides(path); len(ov) != 0 {
		t.Fatalf("overrides = %v", ov)
	}
	if ov := ReadOverrides(filepath.Join(t.TempDir(), "missing.json")); len(ov) != 0 {
		t.Fatalf("overrides = %v", ov)
	}
	if err := WriteOverrides(path, map[string]string{"a": "name-only"}); err == nil {
		t.Fatal("invalid JSON was overwritten")
	}
}
