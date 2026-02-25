package storage

import (
	"context"
	"fmt"

	"github.com/umputun/tg-spam/app/storage/engine"
)

func (s *StorageTestSuite) TestNewChannelSettings() {
	ctx := context.Background()
	for _, dbt := range s.getTestDB() {
		db := dbt.DB
		s.Run(fmt.Sprintf("with %s", db.Type()), func() {
			cs, err := NewChannelSettings(ctx, db)
			s.Require().NoError(err)
			defer db.Exec("DROP TABLE channel_settings")

			s.Require().NotNil(cs)

			if db.Type() == engine.Sqlite {
				var cols []struct {
					CID       int     `db:"cid"`
					Name      string  `db:"name"`
					Type      string  `db:"type"`
					NotNull   bool    `db:"notnull"`
					DfltValue *string `db:"dflt_value"`
					PK        bool    `db:"pk"`
				}
				err = db.Select(&cols, "PRAGMA table_info(channel_settings)")
				s.Require().NoError(err)

				colMap := make(map[string]string)
				for _, col := range cols {
					colMap[col.Name] = col.Type
				}

				s.Equal("TEXT", colMap["gid"])
				s.Equal("REAL", colMap["similarity_threshold"])
				s.Equal("INTEGER", colMap["min_msg_len"])
			}
		})
	}
}

func (s *StorageTestSuite) TestChannelSettingsCRUD() {
	ctx := context.Background()
	for _, dbt := range s.getTestDB() {
		db := dbt.DB
		s.Run(fmt.Sprintf("with %s", db.Type()), func() {
			cs, err := NewChannelSettings(ctx, db)
			s.Require().NoError(err)
			defer db.Exec("DROP TABLE channel_settings")

			// create with defaults
			id, err := cs.Create(ctx, "test-gid")
			s.Require().NoError(err)
			s.Require().Positive(id)

			// get and verify defaults
			settings, err := cs.Get(ctx, "test-gid")
			s.Require().NoError(err)
			s.Require().NotNil(settings)
			s.Equal("test-gid", settings.GID)
			s.InDelta(0.5, settings.SimilarityThreshold, 0.001)
			s.Equal(50, settings.MinMsgLen)
			s.Equal(2, settings.MaxEmoji)
			s.InDelta(50.0, settings.MinSpamProbability, 0.001)
			s.Equal(1, settings.FirstMessagesCount)
			s.False(settings.ParanoidMode)
			s.True(settings.CASEnabled)
			s.False(settings.OpenAIEnabled)
			s.Equal("gpt-4o-mini", settings.OpenAIModel)
			s.Equal(-1, settings.MetaLinksLimit)
			s.Equal(0, settings.DuplicatesThreshold)
			s.Equal("1h", settings.DuplicatesWindow)
			s.Equal(100, settings.AggressiveCleanupLim)

			// get non-existent
			notFound, err := cs.Get(ctx, "nonexistent")
			s.Require().NoError(err)
			s.Nil(notFound)

			// update
			settings.SimilarityThreshold = 0.7
			settings.MinMsgLen = 100
			settings.CASEnabled = false
			settings.OpenAIEnabled = true
			settings.ParanoidMode = true
			settings.MetaLinksLimit = 3
			settings.DuplicatesThreshold = 5
			err = cs.Update(ctx, *settings)
			s.Require().NoError(err)

			updated, err := cs.Get(ctx, "test-gid")
			s.Require().NoError(err)
			s.InDelta(0.7, updated.SimilarityThreshold, 0.001)
			s.Equal(100, updated.MinMsgLen)
			s.False(updated.CASEnabled)
			s.True(updated.OpenAIEnabled)
			s.True(updated.ParanoidMode)
			s.Equal(3, updated.MetaLinksLimit)
			s.Equal(5, updated.DuplicatesThreshold)

			// delete
			err = cs.Delete(ctx, "test-gid")
			s.Require().NoError(err)

			deleted, err := cs.Get(ctx, "test-gid")
			s.Require().NoError(err)
			s.Nil(deleted)
		})
	}
}

func (s *StorageTestSuite) TestChannelSettingsNilDB() {
	ctx := context.Background()
	_, err := NewChannelSettings(ctx, nil)
	s.Require().Error(err)
	s.Contains(err.Error(), "db connection is nil")
}
