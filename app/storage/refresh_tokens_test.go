package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/umputun/tg-spam/app/storage/engine"
)

func (s *StorageTestSuite) TestNewRefreshTokens() {
	ctx := context.Background()
	for _, dbt := range s.getTestDB() {
		db := dbt.DB
		s.Run(fmt.Sprintf("with %s", db.Type()), func() {
			rt, err := NewRefreshTokens(ctx, db)
			s.Require().NoError(err)
			defer db.Exec("DROP TABLE refresh_tokens")

			s.Require().NotNil(rt)

			if db.Type() == engine.Sqlite {
				var cols []struct {
					CID       int     `db:"cid"`
					Name      string  `db:"name"`
					Type      string  `db:"type"`
					NotNull   bool    `db:"notnull"`
					DfltValue *string `db:"dflt_value"`
					PK        bool    `db:"pk"`
				}
				err = db.Select(&cols, "PRAGMA table_info(refresh_tokens)")
				s.Require().NoError(err)

				colMap := make(map[string]string)
				for _, col := range cols {
					colMap[col.Name] = col.Type
				}

				s.Equal("INTEGER", colMap["user_id"])
				s.Equal("TEXT", colMap["token_hash"])
				s.Equal("DATETIME", colMap["expires_at"])
			}
		})
	}
}

func (s *StorageTestSuite) TestRefreshTokensCRUD() {
	ctx := context.Background()
	for _, dbt := range s.getTestDB() {
		db := dbt.DB
		s.Run(fmt.Sprintf("with %s", db.Type()), func() {
			rt, err := NewRefreshTokens(ctx, db)
			s.Require().NoError(err)
			defer db.Exec("DROP TABLE refresh_tokens")

			// create token
			token := RefreshTokenInfo{
				UserID:    1,
				TokenHash: "hash123",
				ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
			}
			id, err := rt.Create(ctx, token)
			s.Require().NoError(err)
			s.Require().Positive(id)

			// find by hash
			found, err := rt.FindByHash(ctx, "hash123")
			s.Require().NoError(err)
			s.Require().NotNil(found)
			s.Equal(int64(1), found.UserID)
			s.Equal("hash123", found.TokenHash)

			// find non-existent
			notFound, err := rt.FindByHash(ctx, "nonexistent")
			s.Require().NoError(err)
			s.Nil(notFound)

			// create second token for same user
			token2 := RefreshTokenInfo{
				UserID:    1,
				TokenHash: "hash456",
				ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
			}
			_, err = rt.Create(ctx, token2)
			s.Require().NoError(err)

			// delete by hash
			err = rt.DeleteByHash(ctx, "hash123")
			s.Require().NoError(err)

			deleted, err := rt.FindByHash(ctx, "hash123")
			s.Require().NoError(err)
			s.Nil(deleted)

			// second token should still exist
			stillExists, err := rt.FindByHash(ctx, "hash456")
			s.Require().NoError(err)
			s.Require().NotNil(stillExists)

			// delete all by user id
			err = rt.DeleteByUserID(ctx, 1)
			s.Require().NoError(err)

			alsoDeleted, err := rt.FindByHash(ctx, "hash456")
			s.Require().NoError(err)
			s.Nil(alsoDeleted)
		})
	}
}

func (s *StorageTestSuite) TestRefreshTokensCleanExpired() {
	ctx := context.Background()
	for _, dbt := range s.getTestDB() {
		db := dbt.DB
		s.Run(fmt.Sprintf("with %s", db.Type()), func() {
			rt, err := NewRefreshTokens(ctx, db)
			s.Require().NoError(err)
			defer db.Exec("DROP TABLE refresh_tokens")

			// create expired token
			expired := RefreshTokenInfo{
				UserID:    1,
				TokenHash: "expired_hash",
				ExpiresAt: time.Now().Add(-1 * time.Hour),
			}
			_, err = rt.Create(ctx, expired)
			s.Require().NoError(err)

			// create valid token
			valid := RefreshTokenInfo{
				UserID:    2,
				TokenHash: "valid_hash",
				ExpiresAt: time.Now().Add(24 * time.Hour),
			}
			_, err = rt.Create(ctx, valid)
			s.Require().NoError(err)

			// clean expired
			count, err := rt.CleanExpired(ctx)
			s.Require().NoError(err)
			s.Equal(int64(1), count)

			// expired should be gone
			gone, err := rt.FindByHash(ctx, "expired_hash")
			s.Require().NoError(err)
			s.Nil(gone)

			// valid should remain
			still, err := rt.FindByHash(ctx, "valid_hash")
			s.Require().NoError(err)
			s.Require().NotNil(still)
		})
	}
}

func (s *StorageTestSuite) TestRefreshTokensNilDB() {
	ctx := context.Background()
	_, err := NewRefreshTokens(ctx, nil)
	s.Require().Error(err)
	s.Contains(err.Error(), "db connection is nil")
}
