package devtools

import (
	"encoding/json"
	"errors"
)

// errStateRedactionFailed replaces a panic from Options.Redact. The panic value is discarded because it may contain the state being redacted.
var errStateRedactionFailed = errors.New("state redaction failed")

// Run application callbacks before taking Inspector.mu; redactors may inspect the inspector. Never retain callback errors, which may themselves contain sensitive input.
func (i *Inspector) redactEvent(event Event) Event {
	if i.options.RedactMetadata == nil {
		return event
	}
	metadata := i.redactMetadata(event.StoreID, Metadata{Name: event.Name, Key: event.Key, Error: event.Error})
	event.Name, event.Key, event.Error = metadata.Name, metadata.Key, metadata.Error
	return event
}

func (i *Inspector) redactMetadata(id string, original Metadata) (result Metadata) {
	result = Metadata{Error: "metadata redaction failed"}
	defer func() {
		if recover() != nil {
			result = Metadata{Error: "metadata redaction failed"}
		}
	}()
	if filtered, err := i.options.RedactMetadata(id, original); err == nil {
		result = filtered
	}
	return result
}

// redactState runs Options.Redact on the notifying goroutine and contains its panics, so a faulty redactor becomes an inspection error instead of escaping into the application's Store write or Attach.
func (i *Inspector) redactState(id string, raw json.RawMessage) (result json.RawMessage, err error) {
	defer func() {
		if recover() != nil {
			result, err = nil, errStateRedactionFailed
		}
	}()
	return i.options.Redact(id, raw)
}
