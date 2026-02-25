package storage

import (
	"context"
	"fmt"
)

func (s *StorageTestSuite) TestNewBots() {
	ctx := context.Background()
	for _, dbt := range s.getTestDB() {
		db := dbt.DB
		s.Run(fmt.Sprintf("with %s", db.Type()), func() {
			b, err := NewBots(ctx, db)
			s.Require().NoError(err)
			defer db.Exec("DROP TABLE bots")

			s.Require().NotNil(b)
		})
	}
}

func (s *StorageTestSuite) TestBotsCRUD() {
	ctx := context.Background()
	for _, dbt := range s.getTestDB() {
		db := dbt.DB
		s.Run(fmt.Sprintf("with %s", db.Type()), func() {
			b, err := NewBots(ctx, db)
			s.Require().NoError(err)
			defer db.Exec("DROP TABLE bots")

			// create bot
			bot := BotInfo{
				Name:     "Test Bot",
				Token:    "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11",
				Username: "testbot",
				Active:   true,
			}
			id, err := b.Create(ctx, bot)
			s.Require().NoError(err)
			s.Require().Positive(id)

			// find by id
			found, err := b.FindByID(ctx, id)
			s.Require().NoError(err)
			s.Require().NotNil(found)
			s.Equal("Test Bot", found.Name)
			s.Equal("123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11", found.Token)
			s.Equal("testbot", found.Username)
			s.True(found.Active)

			// find non-existent
			notFound, err := b.FindByID(ctx, 999)
			s.Require().NoError(err)
			s.Nil(notFound)

			// create second bot (inactive)
			bot2 := BotInfo{
				Name:     "Test Bot 2",
				Token:    "654321:XYZ-ABC5678mnoPq-rst90U3v4w567xy22",
				Username: "testbot2",
				Active:   false,
			}
			_, err = b.Create(ctx, bot2)
			s.Require().NoError(err)

			// list all
			bots, err := b.List(ctx)
			s.Require().NoError(err)
			s.Len(bots, 2)

			// list active only
			active, err := b.ListActive(ctx)
			s.Require().NoError(err)
			s.Len(active, 1)
			s.Equal("Test Bot", active[0].Name)

			// update
			found.Name = "Updated Bot"
			found.Active = false
			err = b.Update(ctx, *found)
			s.Require().NoError(err)

			updated, err := b.FindByID(ctx, id)
			s.Require().NoError(err)
			s.Equal("Updated Bot", updated.Name)
			s.False(updated.Active)

			// delete
			err = b.Delete(ctx, id)
			s.Require().NoError(err)

			deleted, err := b.FindByID(ctx, id)
			s.Require().NoError(err)
			s.Nil(deleted)
		})
	}
}

func (s *StorageTestSuite) TestBotsNilDB() {
	ctx := context.Background()
	_, err := NewBots(ctx, nil)
	s.Require().Error(err)
	s.Contains(err.Error(), "db connection is nil")
}
