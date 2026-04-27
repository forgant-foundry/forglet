package commands

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestRunHelp_NoArgs_PrintsOverview(t *testing.T) {
	out := captureStdout(t, func() { RunHelp(nil) })
	for _, want := range []string{"forglet", "new", "synth", "schema", "plugin", "domain"} {
		if !strings.Contains(out, want) {
			t.Errorf("overview missing %q", want)
		}
	}
}

func TestRunHelp_PluginTopic_HasKeyInterfaces(t *testing.T) {
	out := captureStdout(t, func() { RunHelp([]string{"plugin"}) })
	for _, want := range []string{"Weave", "Scaffolder", "Schemer", "RCSchema", "SetFormat", "RegisterPlugin"} {
		if !strings.Contains(out, want) {
			t.Errorf("plugin help missing %q", want)
		}
	}
}

func TestRunHelp_DomainTopic_HasKeyInterfaces(t *testing.T) {
	out := captureStdout(t, func() { RunHelp([]string{"domain"}) })
	for _, want := range []string{"Synthesizer", "InitializeEvents", "OverlayEvents", "PostSynthesizer", "Schemer", "RCSchema"} {
		if !strings.Contains(out, want) {
			t.Errorf("domain help missing %q", want)
		}
	}
}

func TestRunHelp_UnknownTopic_FallsBackToOverview(t *testing.T) {
	out := captureStdout(t, func() { RunHelp([]string{"notexist"}) })
	if !strings.Contains(out, "forglet") {
		t.Error("unknown topic should fall back to overview")
	}
}

func captureStdout(t *testing.T, f func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	f()
	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}
