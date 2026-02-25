package storage

import (
	"context"
	"fmt"

	"github.com/umputun/tg-spam/app/storage/engine"
)

func (s *StorageTestSuite) TestNewAdminUsers() {
	ctx := context.Background()
	for _, dbt := range s.getTestDB() {
		db := dbt.DB
		s.Run(fmt.Sprintf("with %s", db.Type()), func() {
			au, err := NewAdminUsers(ctx, db)
			s.Require().NoError(err)
			defer db.Exec("DROP TABLE admin_users")

			s.Require().NotNil(au)

			if db.Type() == engine.Sqlite {
				var cols []struct {
					CID       int     `db:"cid"`
					Name      string  `db:"name"`
					Type      string  `db:"type"`
					NotNull   bool    `db:"notnull"`
					DfltValue *string `db:"dflt_value"`
					PK        bool    `db:"pk"`
				}
				err = db.Select(&cols, "PRAGMA table_info(admin_users)")
				s.Require().NoError(err)

				colMap := make(map[string]string)
				for _, col := range cols {
					colMap[col.Name] = col.Type
				}

				s.Equal("TEXT", colMap["username"])
				s.Equal("TEXT", colMap["password_hash"])
				s.Equal("TEXT", colMap["role"])
			}
		})
	}
}

func (s *StorageTestSuite) TestAdminUsersCRUD() {
	ctx := context.Background()
	for _, dbt := range s.getTestDB() {
		db := dbt.DB
		s.Run(fmt.Sprintf("with %s", db.Type()), func() {
			au, err := NewAdminUsers(ctx, db)
			s.Require().NoError(err)
			defer db.Exec("DROP TABLE admin_users")

			// create user
			user := AdminUserInfo{
				Username:     "testadmin",
				PasswordHash: "$2a$10$hash",
				Role:         "superadmin",
				DisplayName:  "Test Admin",
				Active:       true,
			}
			id, err := au.Create(ctx, user)
			s.Require().NoError(err)
			s.Require().Positive(id)

			// find by username
			found, err := au.FindByUsername(ctx, "testadmin")
			s.Require().NoError(err)
			s.Require().NotNil(found)
			s.Equal("testadmin", found.Username)
			s.Equal("superadmin", found.Role)
			s.Equal("Test Admin", found.DisplayName)
			s.True(found.Active)

			// find by id
			foundByID, err := au.FindByID(ctx, id)
			s.Require().NoError(err)
			s.Require().NotNil(foundByID)
			s.Equal("testadmin", foundByID.Username)

			// find non-existent
			notFound, err := au.FindByUsername(ctx, "nonexistent")
			s.Require().NoError(err)
			s.Nil(notFound)

			// list
			users, err := au.List(ctx)
			s.Require().NoError(err)
			s.Len(users, 1)

			// update
			found.DisplayName = "Updated Admin"
			found.Role = "admin"
			err = au.Update(ctx, *found)
			s.Require().NoError(err)

			updated, err := au.FindByID(ctx, id)
			s.Require().NoError(err)
			s.Equal("Updated Admin", updated.DisplayName)
			s.Equal("admin", updated.Role)

			// update password
			err = au.UpdatePassword(ctx, id, "$2a$10$newhash")
			s.Require().NoError(err)

			withNewPass, err := au.FindByID(ctx, id)
			s.Require().NoError(err)
			s.Equal("$2a$10$newhash", withNewPass.PasswordHash)

			// update last login
			err = au.UpdateLastLogin(ctx, id)
			s.Require().NoError(err)

			// count
			count, err := au.Count(ctx)
			s.Require().NoError(err)
			s.Equal(1, count)

			// delete
			err = au.Delete(ctx, id)
			s.Require().NoError(err)

			deleted, err := au.FindByID(ctx, id)
			s.Require().NoError(err)
			s.Nil(deleted)

			count, err = au.Count(ctx)
			s.Require().NoError(err)
			s.Equal(0, count)
		})
	}
}

func (s *StorageTestSuite) TestAdminUsersNilDB() {
	ctx := context.Background()
	_, err := NewAdminUsers(ctx, nil)
	s.Require().Error(err)
	s.Contains(err.Error(), "db connection is nil")
}
