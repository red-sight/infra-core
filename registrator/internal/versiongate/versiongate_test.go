package versiongate

import "testing"

func TestContractHash(t *testing.T) {
	base := map[string]interface{}{
		"info":  map[string]interface{}{"version": "1.0.0", "title": "X"},
		"paths": map[string]interface{}{"/a": map[string]interface{}{"get": map[string]interface{}{}}},
	}
	// Same contract, different info.version → same hash (version excluded).
	bumped := map[string]interface{}{
		"info":  map[string]interface{}{"version": "2.0.0", "title": "X"},
		"paths": map[string]interface{}{"/a": map[string]interface{}{"get": map[string]interface{}{}}},
	}
	if ContractHash(base) != ContractHash(bumped) {
		t.Error("hash must ignore info.version")
	}
	// Different paths → different hash.
	changed := map[string]interface{}{
		"info":  map[string]interface{}{"version": "1.0.0"},
		"paths": map[string]interface{}{"/a": map[string]interface{}{}, "/b": map[string]interface{}{}},
	}
	if ContractHash(base) == ContractHash(changed) {
		t.Error("hash must change when paths change")
	}
	// Schema change is detected too.
	withSchema := map[string]interface{}{
		"paths":      base["paths"],
		"components": map[string]interface{}{"schemas": map[string]interface{}{"Foo": map[string]interface{}{"type": "object"}}},
	}
	if ContractHash(base) == ContractHash(withSchema) {
		t.Error("hash must change when schemas change")
	}
	// Deterministic across calls.
	if ContractHash(base) != ContractHash(base) {
		t.Error("hash must be stable")
	}
}

func TestCompareVersions(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"1.0.0", "1.0.0", 0},
		{"1.0", "1.0.0", 0}, // missing component == 0
		{"v1", "v2", -1},    // leading v stripped
		{"2.0.0", "1.9.9", 1},
		{"1.10.0", "1.2.0", 1}, // numeric, not lexical
		{"1.2.0", "1.10.0", -1},
		{"v1.4.2", "v1.4.1", 1},
	}
	for _, c := range cases {
		if got := compareVersions(c.a, c.b); got != c.want {
			t.Errorf("compareVersions(%q,%q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestCheck(t *testing.T) {
	last := map[string]Digest{
		"svc-a": {Version: "1.0.0", ContractHash: "aaa"},
		"svc-b": {Version: "2.1.0", ContractHash: "bbb"},
	}

	t.Run("new service is not gated", func(t *testing.T) {
		cur := map[string]Digest{"svc-new": {Version: "0.1.0", ContractHash: "zzz"}}
		if v := Check(cur, last); len(v) != 0 {
			t.Errorf("new service should not be gated, got %v", v)
		}
	})

	t.Run("unchanged contract passes", func(t *testing.T) {
		cur := map[string]Digest{"svc-a": {Version: "1.0.0", ContractHash: "aaa"}}
		if v := Check(cur, last); len(v) != 0 {
			t.Errorf("unchanged contract should pass, got %v", v)
		}
	})

	t.Run("contract changed with version bump passes", func(t *testing.T) {
		cur := map[string]Digest{"svc-a": {Version: "1.1.0", ContractHash: "ccc"}}
		if v := Check(cur, last); len(v) != 0 {
			t.Errorf("bumped version should pass, got %v", v)
		}
	})

	t.Run("contract changed without version bump is a violation", func(t *testing.T) {
		cur := map[string]Digest{"svc-a": {Version: "1.0.0", ContractHash: "ccc"}}
		v := Check(cur, last)
		if len(v) != 1 || v[0].Service != "svc-a" {
			t.Fatalf("expected one violation for svc-a, got %v", v)
		}
	})

	t.Run("contract changed with version decrease is a violation", func(t *testing.T) {
		cur := map[string]Digest{"svc-b": {Version: "2.0.0", ContractHash: "ddd"}}
		if v := Check(cur, last); len(v) != 1 {
			t.Errorf("version decrease with contract change must be a violation, got %v", v)
		}
	})
}
