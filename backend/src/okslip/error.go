package okslip

import "errors"

// ErrTimeout wraps request/context timeouts so callers can detect them with
// errors.Is instead of fragile string matching.
var ErrTimeout = errors.New("okslip request timed out")