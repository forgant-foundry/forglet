package eventing

import (
	"encoding/json"
	"time"
)

// Event is a domain event represented as a JSON transfer object.
// Seq is a monotonic sequence number that orders events within an aggregate.
// For synthetic sequencing (e.g. template synthesis), use 1, 2, 3...
// For time-ordered domains, use time.Now().UnixNano() and recover via Time().
type Event struct {
	ID      string          `json:"id"`
	Type    string          `json:"type"`
	Seq     int64           `json:"seq"`
	Payload json.RawMessage `json:"payload"`
}

// Time interprets Seq as Unix nanoseconds and returns the corresponding time.
// Only meaningful when Seq was populated from time.Time.UnixNano().
func (e *Event) Time() time.Time {
	return time.Unix(0, e.Seq)
}
