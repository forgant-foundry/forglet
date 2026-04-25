package project

import "github.com/forgant-foundry/eventing"

// EventStream holds the per-file ordered event sequences that will be replayed
// into aggregates during synthesis. Position in each file's slice determines
// apply order — seq numbers are re-normalized on Events() so plugins never need
// to manage them.
//
// Contributors that write to cross-cutting files (files not owned by the active
// synthesizer) also call SetFormat to register how the file should be rendered.
// Any contributor may call SetFormat for any file — there is no ownership.
type EventStream struct {
	files   map[string][]eventing.Event
	formats map[string]FileFormat
}

// NewEventStream creates an EventStream from a per-file event map.
// Each file's slice is copied so mutations do not affect the original.
// Exported so plugin authors can create streams in their own tests.
func NewEventStream(events map[string][]eventing.Event) *EventStream {
	files := make(map[string][]eventing.Event, len(events))
	for file, evts := range events {
		cp := make([]eventing.Event, len(evts))
		copy(cp, evts)
		files[file] = cp
	}
	return &EventStream{files: files, formats: map[string]FileFormat{}}
}

// SetFormat registers the render format for a cross-cutting file. Call this
// alongside any Append to a file that the active synthesizer does not manage.
// Multiple contributors may call SetFormat for the same file; the last call wins.
func (s *EventStream) SetFormat(file string, format FileFormat) {
	s.formats[file] = format
}

// Formats returns the registered render formats keyed by filename.
func (s *EventStream) Formats() map[string]FileFormat {
	out := make(map[string]FileFormat, len(s.formats))
	for k, v := range s.formats {
		out[k] = v
	}
	return out
}

// Append adds events to the end of the named file's stream.
func (s *EventStream) Append(file string, events ...eventing.Event) {
	s.files[file] = append(s.files[file], events...)
}

// InsertAfter inserts events immediately after the last event of the given
// type in the named file's stream. Falls back to Append if no match is found.
func (s *EventStream) InsertAfter(file, eventType string, events ...eventing.Event) {
	slice := s.files[file]
	idx := lastIndex(slice, eventType)
	if idx < 0 {
		s.files[file] = append(slice, events...)
		return
	}
	s.files[file] = insertAt(slice, idx+1, events...)
}

// InsertBefore inserts events immediately before the first event of the given
// type in the named file's stream. Falls back to prepending if no match is found.
func (s *EventStream) InsertBefore(file, eventType string, events ...eventing.Event) {
	slice := s.files[file]
	idx := firstIndex(slice, eventType)
	if idx < 0 {
		s.files[file] = append(events, slice...)
		return
	}
	s.files[file] = insertAt(slice, idx, events...)
}

// Replace removes all events of the given type from the named file's stream
// and inserts the replacement events at the position of the first match.
// No-op if no event of that type exists.
func (s *EventStream) Replace(file, eventType string, events ...eventing.Event) {
	slice := s.files[file]
	firstMatch := firstIndex(slice, eventType)
	if firstMatch < 0 {
		return
	}
	without := make([]eventing.Event, 0, len(slice))
	for _, e := range slice {
		if e.Type != eventType {
			without = append(without, e)
		}
	}
	// clamp insertion point since removals may have shrunk the slice
	at := firstMatch
	if at > len(without) {
		at = len(without)
	}
	s.files[file] = insertAt(without, at, events...)
}

// Remove removes all events of the given type from the named file's stream.
func (s *EventStream) Remove(file, eventType string) {
	slice := s.files[file]
	result := make([]eventing.Event, 0, len(slice))
	for _, e := range slice {
		if e.Type != eventType {
			result = append(result, e)
		}
	}
	s.files[file] = result
}

// Events returns the final per-file event map with seq numbers re-normalized
// to reflect final position (position 0 → seq 1, position 1 → seq 2, …).
// EventID and EventType are preserved so aggregate node provenance remains intact.
func (s *EventStream) Events() map[string][]eventing.Event {
	out := make(map[string][]eventing.Event, len(s.files))
	for file, evts := range s.files {
		normalized := make([]eventing.Event, len(evts))
		for i, e := range evts {
			e.Seq = int64(i + 1)
			normalized[i] = e
		}
		out[file] = normalized
	}
	return out
}

func firstIndex(events []eventing.Event, eventType string) int {
	for i, e := range events {
		if e.Type == eventType {
			return i
		}
	}
	return -1
}

func lastIndex(events []eventing.Event, eventType string) int {
	for i := len(events) - 1; i >= 0; i-- {
		if events[i].Type == eventType {
			return i
		}
	}
	return -1
}

func insertAt(slice []eventing.Event, at int, events ...eventing.Event) []eventing.Event {
	result := make([]eventing.Event, 0, len(slice)+len(events))
	result = append(result, slice[:at]...)
	result = append(result, events...)
	result = append(result, slice[at:]...)
	return result
}
