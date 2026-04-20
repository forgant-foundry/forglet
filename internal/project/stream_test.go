package project_test

import (
	"testing"

	"github.com/forgant-foundry/eventing"
	"github.com/forgant-foundry/forglet/internal/project"
)

func makeEvt(id, typ string) eventing.Event {
	return eventing.Event{ID: id, Type: typ, Seq: 99}
}

func types(events []eventing.Event) []string {
	out := make([]string, len(events))
	for i, e := range events {
		out[i] = e.Type
	}
	return out
}

func TestEventStream_Append(t *testing.T) {
	s := project.NewEventStream(map[string][]eventing.Event{
		"f": {makeEvt("e1", "init")},
	})
	s.Append("f", makeEvt("e2", "scripts.added"))

	got := types(s.Events()["f"])
	want := []string{"init", "scripts.added"}
	assertOrder(t, got, want)
}

func TestEventStream_InsertAfter_Match(t *testing.T) {
	s := project.NewEventStream(map[string][]eventing.Event{
		"f": {makeEvt("e1", "init"), makeEvt("e2", "scripts.added"), makeEvt("e3", "dep.added")},
	})
	s.InsertAfter("f", "init", makeEvt("eX", "plugin.after.init"))

	got := types(s.Events()["f"])
	assertOrder(t, got, []string{"init", "plugin.after.init", "scripts.added", "dep.added"})
}

func TestEventStream_InsertAfter_LastOccurrence(t *testing.T) {
	s := project.NewEventStream(map[string][]eventing.Event{
		"f": {makeEvt("e1", "dep.added"), makeEvt("e2", "dep.added"), makeEvt("e3", "scripts.added")},
	})
	s.InsertAfter("f", "dep.added", makeEvt("eX", "plugin.added"))

	got := types(s.Events()["f"])
	assertOrder(t, got, []string{"dep.added", "dep.added", "plugin.added", "scripts.added"})
}

func TestEventStream_InsertAfter_NoMatch_Appends(t *testing.T) {
	s := project.NewEventStream(map[string][]eventing.Event{
		"f": {makeEvt("e1", "init")},
	})
	s.InsertAfter("f", "nonexistent", makeEvt("eX", "plugin.added"))

	got := types(s.Events()["f"])
	assertOrder(t, got, []string{"init", "plugin.added"})
}

func TestEventStream_InsertBefore_Match(t *testing.T) {
	s := project.NewEventStream(map[string][]eventing.Event{
		"f": {makeEvt("e1", "init"), makeEvt("e2", "scripts.added")},
	})
	s.InsertBefore("f", "scripts.added", makeEvt("eX", "plugin.before.scripts"))

	got := types(s.Events()["f"])
	assertOrder(t, got, []string{"init", "plugin.before.scripts", "scripts.added"})
}

func TestEventStream_InsertBefore_FirstOccurrence(t *testing.T) {
	s := project.NewEventStream(map[string][]eventing.Event{
		"f": {makeEvt("e1", "dep.added"), makeEvt("e2", "dep.added")},
	})
	s.InsertBefore("f", "dep.added", makeEvt("eX", "plugin.added"))

	got := types(s.Events()["f"])
	assertOrder(t, got, []string{"plugin.added", "dep.added", "dep.added"})
}

func TestEventStream_InsertBefore_NoMatch_Prepends(t *testing.T) {
	s := project.NewEventStream(map[string][]eventing.Event{
		"f": {makeEvt("e1", "init")},
	})
	s.InsertBefore("f", "nonexistent", makeEvt("eX", "plugin.added"))

	got := types(s.Events()["f"])
	assertOrder(t, got, []string{"plugin.added", "init"})
}

func TestEventStream_Replace(t *testing.T) {
	s := project.NewEventStream(map[string][]eventing.Event{
		"f": {makeEvt("e1", "init"), makeEvt("e2", "dep.added"), makeEvt("e3", "scripts.added")},
	})
	s.Replace("f", "dep.added", makeEvt("eX", "plugin.dep"))

	got := types(s.Events()["f"])
	assertOrder(t, got, []string{"init", "plugin.dep", "scripts.added"})
}

func TestEventStream_Replace_AllOccurrencesRemovedReplacementAtFirst(t *testing.T) {
	s := project.NewEventStream(map[string][]eventing.Event{
		"f": {makeEvt("e1", "dep.added"), makeEvt("e2", "scripts.added"), makeEvt("e3", "dep.added")},
	})
	s.Replace("f", "dep.added", makeEvt("eX", "plugin.dep"))

	got := types(s.Events()["f"])
	assertOrder(t, got, []string{"plugin.dep", "scripts.added"})
}

func TestEventStream_Replace_NoMatch_Noop(t *testing.T) {
	s := project.NewEventStream(map[string][]eventing.Event{
		"f": {makeEvt("e1", "init")},
	})
	s.Replace("f", "nonexistent", makeEvt("eX", "plugin.added"))

	got := types(s.Events()["f"])
	assertOrder(t, got, []string{"init"})
}

func TestEventStream_Remove(t *testing.T) {
	s := project.NewEventStream(map[string][]eventing.Event{
		"f": {makeEvt("e1", "init"), makeEvt("e2", "dep.added"), makeEvt("e3", "scripts.added")},
	})
	s.Remove("f", "dep.added")

	got := types(s.Events()["f"])
	assertOrder(t, got, []string{"init", "scripts.added"})
}

func TestEventStream_Remove_AllOccurrences(t *testing.T) {
	s := project.NewEventStream(map[string][]eventing.Event{
		"f": {makeEvt("e1", "dep.added"), makeEvt("e2", "scripts.added"), makeEvt("e3", "dep.added")},
	})
	s.Remove("f", "dep.added")

	got := types(s.Events()["f"])
	assertOrder(t, got, []string{"scripts.added"})
}

func TestEventStream_SeqNormalized(t *testing.T) {
	s := project.NewEventStream(map[string][]eventing.Event{
		"f": {makeEvt("e1", "init"), makeEvt("e2", "scripts.added"), makeEvt("e3", "dep.added")},
	})
	s.InsertAfter("f", "init", makeEvt("eX", "plugin.added"))

	for i, e := range s.Events()["f"] {
		wantSeq := int64(i + 1)
		if e.Seq != wantSeq {
			t.Errorf("event[%d] (%s): seq = %d, want %d", i, e.Type, e.Seq, wantSeq)
		}
	}
}

func TestEventStream_SeqNormalized_PreservesIDAndType(t *testing.T) {
	s := project.NewEventStream(map[string][]eventing.Event{
		"f": {makeEvt("my-id", "my-type")},
	})
	events := s.Events()["f"]
	if events[0].ID != "my-id" {
		t.Errorf("ID = %q, want %q", events[0].ID, "my-id")
	}
	if events[0].Type != "my-type" {
		t.Errorf("Type = %q, want %q", events[0].Type, "my-type")
	}
}

func TestEventStream_NewEventStream_DoesNotMutateOriginal(t *testing.T) {
	original := map[string][]eventing.Event{
		"f": {makeEvt("e1", "init")},
	}
	s := project.NewEventStream(original)
	s.Append("f", makeEvt("e2", "extra"))

	if len(original["f"]) != 1 {
		t.Error("NewEventStream mutated the original event map")
	}
}

func assertOrder(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("event order: got %v, want %v", got, want)
		return
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("event[%d]: got %q, want %q (full order: %v)", i, got[i], want[i], got)
			return
		}
	}
}
