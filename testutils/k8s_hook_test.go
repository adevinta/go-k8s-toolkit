package testutils

import (
	"github.com/cenkalti/backoff"
)

var (
	originalNewBackoff = newBackOff
)

func ResetHooks() {
	newBackOff = originalNewBackoff
}

func SetNewBackOff(f func() backoff.BackOff) {
	newBackOff = f
}
