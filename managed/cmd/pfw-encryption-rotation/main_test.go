package main

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/percona/pmm/managed/utils/dbsecret"
)

// The parsing itself is covered by managed/utils/dbsecret. What matters here is
// that an omitted or empty --postgres-password still reaches the resolver: the
// documented invocation in data_encryption.md passes no password at all, and a
// flag default would have quietly reintroduced the shipped constant.
func TestSetupParamsResolvesPassword(t *testing.T) {
	t.Parallel()

	t.Run("empty password is resolved, not left empty", func(t *testing.T) {
		t.Parallel()
		// No secret store on a test runner, so this is dbsecret's fallback --
		// the point is that it is never the empty string.
		assert.Equal(t, dbsecret.ManagedDBPassword(), setupParams(flags{}).Password)
		assert.NotEmpty(t, setupParams(flags{}).Password)
	})

	t.Run("an explicit password wins", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "explicit", setupParams(flags{DBPassword: "explicit"}).Password)
	})
}
