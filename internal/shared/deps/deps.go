// Package deps provides a small generic functional-options helper shared by
// the bootstrap code: options validate at apply time and return an error
// instead of failing later in the runtime.
package deps

// Option configures a value of type T, reporting an error when the supplied
// configuration is invalid.
type Option[T any] func(*T) error

// Apply runs every option against t in order, stopping at the first error.
func Apply[T any](t *T, opts ...Option[T]) error {
	for _, o := range opts {
		if err := o(t); err != nil {
			return err
		}
	}
	return nil
}

// Build allocates a zero T, applies the options to it and returns the result,
// or the first option error.
func Build[T any](opts ...Option[T]) (*T, error) {
	t := new(T)
	if err := Apply(t, opts...); err != nil {
		return nil, err
	}
	return t, nil
}
