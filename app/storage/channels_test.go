package storage

import (
	"context"
	"fmt"

	"github.com/umputun/tg-spam/app/storage/engine"
)

func (s *StorageTestSuite) TestNewChannels() {
	ctx := context.Background()
	for _, dbt := range s.getTestDB() {
		db := dbt.DB
		s.Run(fmt.Sprintf("with %s", db.Type()), func() {
			ch, err := NewChannels(ctx, db)
			s.Require().NoError(err)
			defer db.Exec("DROP TABLE channels")

			s.Require().NotNil(ch)

			if db.Type() == engine.Sqlite {
				var cols []struct {
					CID       int     `db:"cid"`
					Name      string  `db:"name"`
					Type      string  `db:"type"`
					NotNull   bool    `db:"notnull"`
					DfltValue *string `db:"dflt_value"`
					PK        bool    `db:"pk"`
				}
				err = db.Select(&cols, "PRAGMA table_info(channels)")
				s.Require().NoError(err)

				colMap := make(map[string]string)
				for _, col := range cols {
					colMap[col.Name] = col.Type
				}

				s.Equal("TEXT", colMap["gid"])
				s.Equal("INTEGER", colMap["telegram_id"])
				s.Equal("TEXT", colMap["name"])
			}
		})
	}
}

func (s *StorageTestSuite) TestChannelsCRUD() {
	ctx := context.Background()
	for _, dbt := range s.getTestDB() {
		db := dbt.DB
		s.Run(fmt.Sprintf("with %s", db.Type()), func() {
			ch, err := NewChannels(ctx, db)
			s.Require().NoError(err)
			defer db.Exec("DROP TABLE channels")

			// create channel
			channel := ChannelInfo{
				GID:        "test-channel",
				TelegramID: -1001234567890,
				Name:       "Test Channel",
				Username:   "testchannel",
				Active:     true,
			}
			id, err := ch.Create(ctx, channel)
			s.Require().NoError(err)
			s.Require().Positive(id)

			// find by gid
			found, err := ch.FindByGID(ctx, "test-channel")
			s.Require().NoError(err)
			s.Require().NotNil(found)
			s.Equal("test-channel", found.GID)
			s.Equal(int64(-1001234567890), found.TelegramID)
			s.Equal("Test Channel", found.Name)
			s.Equal("testchannel", found.Username)
			s.True(found.Active)

			// find by telegram id
			foundByTG, err := ch.FindByTelegramID(ctx, -1001234567890)
			s.Require().NoError(err)
			s.Require().NotNil(foundByTG)
			s.Equal("test-channel", foundByTG.GID)

			// find non-existent
			notFound, err := ch.FindByGID(ctx, "nonexistent")
			s.Require().NoError(err)
			s.Nil(notFound)

			// create second channel
			channel2 := ChannelInfo{
				GID:        "test-channel-2",
				TelegramID: -1009876543210,
				Name:       "Test Channel 2",
				Active:     false,
			}
			_, err = ch.Create(ctx, channel2)
			s.Require().NoError(err)

			// list all
			channels, err := ch.List(ctx)
			s.Require().NoError(err)
			s.Len(channels, 2)

			// list active only
			active, err := ch.ListActive(ctx)
			s.Require().NoError(err)
			s.Len(active, 1)
			s.Equal("test-channel", active[0].GID)

			// update
			found.Name = "Updated Channel"
			found.Active = false
			err = ch.Update(ctx, *found)
			s.Require().NoError(err)

			updated, err := ch.FindByGID(ctx, "test-channel")
			s.Require().NoError(err)
			s.Equal("Updated Channel", updated.Name)
			s.False(updated.Active)

			// delete
			err = ch.Delete(ctx, id)
			s.Require().NoError(err)

			deleted, err := ch.FindByGID(ctx, "test-channel")
			s.Require().NoError(err)
			s.Nil(deleted)
		})
	}
}

func (s *StorageTestSuite) TestChannelsNilDB() {
	ctx := context.Background()
	_, err := NewChannels(ctx, nil)
	s.Require().Error(err)
	s.Contains(err.Error(), "db connection is nil")
}
