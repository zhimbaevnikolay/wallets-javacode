package herrors

import "errors"

var (
	ErrWorkerStop = errors.New("worker stopped")
)
