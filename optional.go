package studio

import "encoding/json"

// Optional is a patch field that distinguishes an absent field from an explicit null. Its zero
// value is absent, Some sets a value, and Null sends null.
type Optional[T any] struct {
	value *T
	set   bool
}

// Some returns an Optional that sends value.
func Some[T any](value T) Optional[T] {
	return Optional[T]{value: &value, set: true}
}

// Null returns an Optional that sends an explicit null.
func Null[T any]() Optional[T] {
	return Optional[T]{set: true}
}

// IsZero reports whether the Optional is absent. encoding/json's omitzero uses it.
func (o Optional[T]) IsZero() bool {
	return !o.set
}

// MarshalJSON sends the set value, or null for an explicit null.
func (o Optional[T]) MarshalJSON() ([]byte, error) {
	if o.value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(*o.value)
}

// UnmarshalJSON records the field as set, keeping an explicit null distinct from a value.
func (o *Optional[T]) UnmarshalJSON(data []byte) error {
	o.set = true
	if string(data) == "null" {
		o.value = nil
		return nil
	}
	var value T
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	o.value = &value
	return nil
}
