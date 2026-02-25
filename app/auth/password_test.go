package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashAndCheckPassword(t *testing.T) {
	t.Run("correct password", func(t *testing.T) {
		hash, err := HashPassword("my-secret-pass")
		require.NoError(t, err)
		assert.NotEmpty(t, hash)
		assert.True(t, CheckPassword("my-secret-pass", hash))
	})

	t.Run("wrong password", func(t *testing.T) {
		hash, err := HashPassword("my-secret-pass")
		require.NoError(t, err)
		assert.False(t, CheckPassword("wrong-password", hash))
	})

	t.Run("different hashes for same password", func(t *testing.T) {
		hash1, err := HashPassword("same-password")
		require.NoError(t, err)
		hash2, err := HashPassword("same-password")
		require.NoError(t, err)
		assert.NotEqual(t, hash1, hash2) // bcrypt produces different hashes each time
		assert.True(t, CheckPassword("same-password", hash1))
		assert.True(t, CheckPassword("same-password", hash2))
	})
}
