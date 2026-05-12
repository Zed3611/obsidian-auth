package testutils

import "testing"

func SkipShort(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping functional test in short mode")
	}
}
