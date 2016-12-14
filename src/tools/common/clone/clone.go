// Package clone implements a generic Clone() operation applicable to
// all types.
package clone

import (
	"bytes"
	"encoding/gob"
	"fmt"
)

// Clone clones the exported fields of 'src' into 'dst'. Empty maps
// and slices may be converted to nil pointers. Typical usage:
//
//   var a SomeType
//   var b SomeType
//   Clone(a, &b)
func Clone(src, dst interface{}) error {
	// Implementation is based on
	// https://groups.google.com/forum/#!topic/golang-nuts/vK6P0dmQI84

	buf := new(bytes.Buffer)
	enc := gob.NewEncoder(buf)
	dec := gob.NewDecoder(buf)
	if err := enc.Encode(src); err != nil {
		return fmt.Errorf("encoder error: %s\n   src: %#v\n   dst: %#v\n", err, src, dst)
	}
	if err := dec.Decode(dst); err != nil {
		return fmt.Errorf("decoder error: %s\n   src: %#v\n   dst: %#v\n", err, src, dst)
	}
	return nil
}
