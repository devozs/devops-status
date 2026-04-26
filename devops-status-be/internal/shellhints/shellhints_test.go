package shellhints

import (
	"strings"
	"testing"
)

func TestParseProbeEnvLines(t *testing.T) {
	m, err := ParseProbeEnvLines("FOO=bar\n#c\n\nBAZ=quux")
	if err != nil {
		t.Fatal(err)
	}
	if m["FOO"] != "bar" || m["BAZ"] != "quux" {
		t.Fatalf("got %#v", m)
	}
	_, err = ParseProbeEnvLines("bad line")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseNodeSelectorLines(t *testing.T) {
	m, err := ParseNodeSelectorLines("topology.kubernetes.io/zone=us-east-1a\n")
	if err != nil {
		t.Fatal(err)
	}
	if len(m) != 1 {
		t.Fatalf("got %#v", m)
	}
	_, err = ParseNodeSelectorLines("noequals")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestMapsEqual(t *testing.T) {
	if !MapsEqual(map[string]string{"a": "1"}, map[string]string{"a": "1"}) {
		t.Fatal()
	}
	if MapsEqual(map[string]string{"a": "1"}, map[string]string{"a": "2"}) {
		t.Fatal()
	}
}

func TestValidateHintBodyMaxSize(t *testing.T) {
	large := strings.Repeat("a", MaxHintBodyBytes+1)
	if err := ValidateHintBody(KindProbeShell, large); err == nil {
		t.Fatal("expected size error")
	}
}

func TestValidateHintBodyUnknownKind(t *testing.T) {
	if err := ValidateHintBody("nope", "x"); err == nil {
		t.Fatal("expected error")
	}
}
