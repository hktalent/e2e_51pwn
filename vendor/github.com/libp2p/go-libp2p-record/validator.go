package record

import (
	"errors"
	"fmt"
)

// ErrInvalidRecordType is returned if a DHTRecord keys prefix
// is not found in the Validator map of the DHT.
var ErrInvalidRecordType = errors.New("invalid record keytype")

// ErrBetterRecord is returned by a subsystem when it fails because it found a
// better record.
type ErrBetterRecord struct {
	// Key is the key associated with the record.
	Key string
	// Value is the best value that was found, according to the record's
	// validator.
	Value []byte
}

func (e *ErrBetterRecord) Error() string {
	return fmt.Sprintf("found better value for %q", e.Key)
}

// Validator is an interface that should be implemented by record validators.
type Validator interface {
	// Validate validates the given record, returning an error if it's
	// invalid (e.g., expired, signed by the wrong key, etc.).
	Validate(key string, value []byte) error

	// Select selects the best record from the set of records (e.g., the
	// newest).
	//
	// Decisions made by select should be stable.
	Select(key string, values [][]byte) (int, error)
}

// NamespacedValidator is a validator that delegates to sub-validators by
// namespace.
type NamespacedValidator map[string]Validator

// ValidatorByKey looks up the validator responsible for validating the given
// key.
func (v NamespacedValidator) ValidatorByKey(key string) Validator {
	ns, _, err := SplitKey(key)
	if err != nil {
		return nil
	}
	return v[ns]
}

// Validate conforms to the Validator interface.
func (v NamespacedValidator) Validate(key string, value []byte) error {
	vi := v.ValidatorByKey(key)
	if vi == nil {
		return ErrInvalidRecordType
	}
	return vi.Validate(key, value)
}

// Select conforms to the Validator interface.
func (v NamespacedValidator) Select(key string, values [][]byte) (int, error) {
	if len(values) == 0 {
		return 0, errors.New("can't select from no values")
	}
	vi := v.ValidatorByKey(key)
	if vi == nil {
		return 0, ErrInvalidRecordType
	}
	return vi.Select(key, values)
}

var _ Validator = NamespacedValidator{}
