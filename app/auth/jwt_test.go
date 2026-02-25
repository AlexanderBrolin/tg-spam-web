package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateAndValidateToken(t *testing.T) {
	secret := "test-secret-key"

	t.Run("valid token", func(t *testing.T) {
		token, err := GenerateToken(1, "admin", "superadmin", secret, 15*time.Minute)
		require.NoError(t, err)
		assert.NotEmpty(t, token)

		claims, err := ValidateToken(token, secret)
		require.NoError(t, err)
		assert.Equal(t, int64(1), claims.UserID)
		assert.Equal(t, "admin", claims.Username)
		assert.Equal(t, "superadmin", claims.Role)
		assert.Equal(t, "tg-spam-corp", claims.Issuer)
	})

	t.Run("expired token", func(t *testing.T) {
		token, err := GenerateToken(1, "admin", "superadmin", secret, -1*time.Minute)
		require.NoError(t, err)

		_, err = ValidateToken(token, secret)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid token")
	})

	t.Run("wrong secret", func(t *testing.T) {
		token, err := GenerateToken(1, "admin", "superadmin", secret, 15*time.Minute)
		require.NoError(t, err)

		_, err = ValidateToken(token, "wrong-secret")
		assert.Error(t, err)
	})

	t.Run("invalid token string", func(t *testing.T) {
		_, err := ValidateToken("not-a-valid-token", secret)
		assert.Error(t, err)
	})
}
