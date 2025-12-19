package easyslip

import "errors"

var ErrDupeSlip = errors.New("This slip is dupe")
var ErrInternal = errors.New("internal error")