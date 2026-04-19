package rotolog

import "errors"

// ErrClosed is returned when Write is called after the rotator has been closed.
var ErrClosed = errors.New("rotolog: rotator is closed")
