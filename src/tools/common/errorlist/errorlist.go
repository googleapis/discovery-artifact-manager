// Package errorlist maintains a list of errors that clients can
// populate and later retrieve, in order to present multiple errors to
// the user at once (as opposed to causing piecemeal incremental
// failures as the user fixes one error and re-runs).
package errorlist

import (
	"fmt"
	"strings"
)

// Errors accumulates a list of errors and returns it as a single error.
type Errors []string

// Add adds 'err' to the Errors.
func (el *Errors) Add(err error) {
	*el = append(*el, err.Error())
}

// Error returns an error consisting of all the previously accumulated
// errors, or nil of no errors were encountered.
func (el Errors) Error() error {
	if len(el) == 0 {
		return nil
	}
	return fmt.Errorf("%s", strings.Join(el, "\n\n"))
}

// Clear forgets about all previously accumulated errors.
func (el *Errors) Clear() {
	*el = nil
}
