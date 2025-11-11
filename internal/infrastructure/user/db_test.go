package userdb

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/Oleja123/estate-agency/internal/domain/user"
	"github.com/Oleja123/estate-agency/internal/infrastructure/basedb"
	"github.com/Oleja123/estate-agency/internal/infrastructure/client"
	postgresqlclient "github.com/Oleja123/estate-agency/internal/infrastructure/client/postgresql"
	"github.com/Oleja123/estate-agency/internal/infrastructure/config"
	"github.com/Oleja123/estate-agency/internal/infrastructure/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	testClient client.Client
	testRepo   *Repository
	testLogger *slog.Logger
	testCtx    context.Context
)

func TestMain(m *testing.M) {
	testCtx = context.Background()

	testLogger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	testConfig := config.Config{
		DbConfig: config.DatabaseConfig{
			Username:         "root",
			Password:         "root",
			Host:             "localhost",
			Port:             "5432",
			Database:         "test",
			MaxAttempts:      5,
			SecondsToConnect: 5,
		},
	}

	testClient, _ = postgresqlclient.NewClient(context.Background(), *testLogger, testConfig)
	testRepo = New(testClient, testLogger)

	if err := utils.RunGooseMigrations(testLogger); err != nil {
		testLogger.Error("Не удалось запустить миграции goose", "ошибка", err)
		os.Exit(1)
	}

	code := m.Run()
	os.Exit(code)
}

func truncateTables() error {
	_, err := testClient.Exec(context.Background(), "TRUNCATE TABLE users RESTART IDENTITY CASCADE")
	return err
}

func TestCreateUser(t *testing.T) {
	tests := []struct {
		name     string
		user     user.User
		wantErr  bool
		errType  func(error) bool
		setup    func()
		validate func(t *testing.T, userID int)
	}{
		{
			name: "successful creation",
			user: user.User{
				Email:        "success@example.com",
				PasswordHash: "hashed_password",
				FirstName:    "John",
				LastName:     "Doe",
				PhoneNumber:  "+1234567890",
				UserRole:     "client",
			},
			wantErr: false,
			validate: func(t *testing.T, userID int) {
				createdUser, err := testRepo.GetByID(testCtx, userID)
				require.NoError(t, err)
				assert.Equal(t, "success@example.com", createdUser.Email)
				assert.True(t, createdUser.IsActive)
			},
		},
		{
			name: "duplicate email",
			user: user.User{
				Email:        "duplicate@example.com",
				PasswordHash: "hash2",
				FirstName:    "Second",
				LastName:     "User",
				UserRole:     "client",
			},
			wantErr: true,
			errType: basedb.IsAlreadyExists,
			setup: func() {
				firstUser := user.User{
					Email:        "duplicate@example.com",
					PasswordHash: "hash1",
					FirstName:    "First",
					LastName:     "User",
					UserRole:     "client",
				}
				_, err := testRepo.Create(testCtx, firstUser)
				require.NoError(t, err)
			},
		},
		{
			name: "user with all fields",
			user: user.User{
				Email:        "full@example.com",
				PasswordHash: "full_hash",
				FirstName:    "Full",
				LastName:     "User",
				PhoneNumber:  "+1111111111",
				UserRole:     "admin",
			},
			wantErr: false,
			validate: func(t *testing.T, userID int) {
				createdUser, err := testRepo.GetByID(testCtx, userID)
				require.NoError(t, err)
				assert.Equal(t, "full@example.com", createdUser.Email)
				assert.Equal(t, "Full", createdUser.FirstName)
				assert.Equal(t, "User", createdUser.LastName)
				assert.Equal(t, "+1111111111", createdUser.PhoneNumber)
				assert.Equal(t, user.RoleAdmin, createdUser.UserRole)
			},
		},
		{
			name: "user without phone",
			user: user.User{
				Email:        "nophone@example.com",
				PasswordHash: "hash",
				FirstName:    "No",
				LastName:     "Phone",
				UserRole:     "admin",
			},
			wantErr: false,
			validate: func(t *testing.T, userID int) {
				createdUser, err := testRepo.GetByID(testCtx, userID)
				require.NoError(t, err)
				assert.Equal(t, "nophone@example.com", createdUser.Email)
				assert.Empty(t, createdUser.PhoneNumber)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, truncateTables())
			if tt.setup != nil {
				tt.setup()
			}

			userID, err := testRepo.Create(testCtx, tt.user)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errType != nil {
					assert.True(t, tt.errType(err))
				}
			} else {
				require.NoError(t, err)
				assert.Greater(t, userID, 0)

				if tt.validate != nil {
					tt.validate(t, userID)
				}
			}
		})
	}
}

func TestGetById(t *testing.T) {
	testUser := user.User{
		Email:        "getbyid_test@example.com",
		PasswordHash: "hash",
		FirstName:    "Test",
		LastName:     "User",
		UserRole:     "client",
	}

	userID, err := testRepo.Create(testCtx, testUser)
	require.NoError(t, err)

	tests := []struct {
		name     string
		id       int
		wantErr  bool
		errType  func(error) bool
		validate func(t *testing.T, foundUser user.User)
	}{
		{
			name:    "successful get by id",
			id:      userID,
			wantErr: false,
			validate: func(t *testing.T, foundUser user.User) {
				assert.Equal(t, userID, foundUser.Id)
				assert.Equal(t, "getbyid_test@example.com", foundUser.Email)
				assert.Equal(t, "Test", foundUser.FirstName)
				assert.True(t, foundUser.IsActive)
			},
		},
		{
			name:    "not found",
			id:      99999,
			wantErr: true,
			errType: basedb.IsNotFound,
		},
		{
			name:    "zero id",
			id:      0,
			wantErr: true,
			errType: basedb.IsNotFound,
		},
		{
			name:    "negative id",
			id:      -1,
			wantErr: true,
			errType: basedb.IsNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			foundUser, err := testRepo.GetByID(testCtx, tt.id)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errType != nil {
					assert.True(t, tt.errType(err))
				}
			} else {
				require.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, foundUser)
				}
			}
		})
	}
}

func TestGetByEmail(t *testing.T) {
	testUser := user.User{
		Email:        "getbyemail_test@example.com",
		PasswordHash: "hash",
		FirstName:    "Email",
		LastName:     "Test",
		UserRole:     "admin",
	}

	_, err := testRepo.Create(testCtx, testUser)
	require.NoError(t, err)

	tests := []struct {
		name     string
		email    string
		wantErr  bool
		errType  func(error) bool
		validate func(t *testing.T, foundUser user.User)
	}{
		{
			name:    "successful get by email",
			email:   "getbyemail_test@example.com",
			wantErr: false,
			validate: func(t *testing.T, foundUser user.User) {
				assert.Equal(t, "getbyemail_test@example.com", foundUser.Email)
				assert.Equal(t, "Email", foundUser.FirstName)
				assert.Equal(t, user.RoleAdmin, foundUser.UserRole)
			},
		},
		{
			name:    "email not found",
			email:   "nonexistent@example.com",
			wantErr: true,
			errType: basedb.IsNotFound,
		},
		{
			name:    "empty email",
			email:   "",
			wantErr: true,
			errType: basedb.IsNotFound,
		},
		{
			name:    "case sensitive email",
			email:   "GETBYEMAIL_TEST@EXAMPLE.COM",
			wantErr: true,
			errType: basedb.IsNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			foundUser, err := testRepo.GetByEmail(testCtx, tt.email)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errType != nil {
					assert.True(t, tt.errType(err))
				}
			} else {
				require.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, foundUser)
				}
			}
		})
	}
}

func TestUpdateUser(t *testing.T) {
	tests := []struct {
		name       string
		setup      func() int
		updateUser user.User
		wantErr    bool
		errType    func(error) bool
		validate   func(t *testing.T, userID int)
	}{
		{
			name: "successful update all fields",
			setup: func() int {
				u := user.User{
					Email:        "update_all@example.com",
					PasswordHash: "original_hash",
					FirstName:    "Original",
					LastName:     "User",
					UserRole:     "client",
				}
				id, err := testRepo.Create(testCtx, u)
				require.NoError(t, err)
				return id
			},
			updateUser: user.User{
				Email:       "update_all@example.com",
				FirstName:   "Updated",
				LastName:    "Name",
				PhoneNumber: "+9999999999",
				UserRole:    "admin",
				IsActive:    false,
			},
			wantErr: false,
			validate: func(t *testing.T, userID int) {
				updated, err := testRepo.GetByID(testCtx, userID)
				require.NoError(t, err)
				assert.Equal(t, "Updated", updated.FirstName)
				assert.Equal(t, "Name", updated.LastName)
				assert.Equal(t, "+9999999999", updated.PhoneNumber)
				assert.Equal(t, user.RoleAdmin, updated.UserRole)
				assert.False(t, updated.IsActive)
			},
		},
		{
			name: "update only first name",
			setup: func() int {
				u := user.User{
					Email:        "update_partial@example.com",
					PasswordHash: "hash",
					FirstName:    "Old",
					LastName:     "User",
					UserRole:     user.RoleClient,
				}
				id, err := testRepo.Create(testCtx, u)
				require.NoError(t, err)
				return id
			},
			updateUser: user.User{
				Email:     "update_partial@example.com",
				FirstName: "New",
				LastName:  "User",
				UserRole:  user.RoleClient,
			},
			wantErr: false,
			validate: func(t *testing.T, userID int) {
				updated, err := testRepo.GetByID(testCtx, userID)
				require.NoError(t, err)
				assert.Equal(t, "New", updated.FirstName)
				assert.Equal(t, "User", updated.LastName)
				assert.Equal(t, user.RoleClient, updated.UserRole)
			},
		},
		{
			name: "update non-existent user",
			setup: func() int {
				return 99999
			},
			updateUser: user.User{
				Id:        99999,
				Email:     "nonexistent@example.com",
				FirstName: "Nonexistent",
				LastName:  "User",
				UserRole:  user.RoleClient,
			},
			wantErr: true,
			errType: basedb.IsNotFound,
		},
		{
			name: "activate user",
			setup: func() int {
				u := user.User{
					Email:        "activate@example.com",
					PasswordHash: "hash",
					FirstName:    "Inactive",
					LastName:     "User",
					UserRole:     user.RoleClient,
				}
				id, err := testRepo.Create(testCtx, u)
				require.NoError(t, err)
				return id
			},
			updateUser: user.User{
				Email:     "activate@example.com",
				FirstName: "Active",
				LastName:  "User",
				UserRole:  user.RoleClient,
				IsActive:  true,
			},
			wantErr: false,
			validate: func(t *testing.T, userID int) {
				updated, err := testRepo.GetByID(testCtx, userID)
				require.NoError(t, err)
				assert.True(t, updated.IsActive)
				assert.Equal(t, "Active", updated.FirstName)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, truncateTables())
			userID := tt.setup()

			tt.updateUser.Id = userID
			err := testRepo.Update(testCtx, tt.updateUser)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errType != nil {
					assert.True(t, tt.errType(err))
				}
			} else {
				require.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, userID)
				}
			}
		})
	}
}

func TestDeleteUser_TableDriven(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() int
		deleteID int
		wantErr  bool
		errType  func(error) bool
		validate func(t *testing.T, deletedID int)
	}{
		{
			name: "successful delete",
			setup: func() int {
				u := user.User{
					Email:        "delete_success@example.com",
					PasswordHash: "hash",
					FirstName:    "Delete",
					LastName:     "Me",
					UserRole:     "client",
				}
				id, err := testRepo.Create(testCtx, u)
				require.NoError(t, err)
				return id
			},
			deleteID: 0,
			wantErr:  false,
			validate: func(t *testing.T, deletedID int) {
				_, err := testRepo.GetByID(testCtx, deletedID)
				assert.Error(t, err)
				assert.True(t, basedb.IsNotFound(err))
			},
		},
		{
			name: "delete non-existent user",
			setup: func() int {
				return 99999
			},
			deleteID: 99999,
			wantErr:  true,
			errType:  basedb.IsNotFound,
		},
		{
			name: "delete already deleted user",
			setup: func() int {
				u := user.User{
					Email:        "delete_twice@example.com",
					PasswordHash: "hash",
					FirstName:    "Delete",
					LastName:     "Twice",
					UserRole:     "client",
				}
				id, err := testRepo.Create(testCtx, u)
				require.NoError(t, err)

				_, err = testRepo.Delete(testCtx, id)
				require.NoError(t, err)

				return id
			},
			deleteID: 0,
			wantErr:  true,
			errType:  basedb.IsNotFound,
		},
		{
			name: "delete with zero id",
			setup: func() int {
				return 0
			},
			deleteID: 0,
			wantErr:  true,
			errType:  basedb.IsNotFound,
		},
		{
			name: "delete with negative id",
			setup: func() int {
				return -1
			},
			deleteID: -1,
			wantErr:  true,
			errType:  basedb.IsNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, truncateTables())
			userID := tt.setup()

			deleteID := tt.deleteID
			if deleteID == 0 {
				deleteID = userID
			}

			deletedID, err := testRepo.Delete(testCtx, deleteID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errType != nil {
					assert.True(t, tt.errType(err))
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, deleteID, deletedID)
				if tt.validate != nil {
					tt.validate(t, deletedID)
				}
			}
		})
	}
}

func TestListUsers_TableDriven(t *testing.T) {
	setupTestUsers := func() {
		users := []user.User{
			{
				Email:        "client1@example.com",
				PasswordHash: "hash1",
				FirstName:    "John",
				LastName:     "Doe",
				UserRole:     user.RoleClient,
			},
			{
				Email:        "client2@example.com",
				PasswordHash: "hash2",
				FirstName:    "Jane",
				LastName:     "Smith",
				UserRole:     user.RoleClient,
			},
			{
				Email:        "admin1@example.com",
				PasswordHash: "hash3",
				FirstName:    "Admin",
				LastName:     "One",
				UserRole:     user.RoleAdmin,
			},
			{
				Email:        "admin2@example.com",
				PasswordHash: "hash4",
				FirstName:    "Admin",
				LastName:     "Two",
				UserRole:     user.RoleAdmin,
			},
			{
				Email:        "realtor1@example.com",
				PasswordHash: "hash5",
				FirstName:    "Realtor",
				LastName:     "One",
				UserRole:     user.RoleAdmin,
			},
		}

		for i := range users {
			_, err := testRepo.Create(testCtx, users[i])
			require.NoError(t, err)
		}
	}

	tests := []struct {
		name     string
		request  user.ListRequest
		wantLen  int
		validate func(t *testing.T, users []user.User)
	}{
		{
			name: "list all users with pagination",
			request: user.ListRequest{
				Limit:  3,
				Offset: 0,
			},
			wantLen: 3,
			validate: func(t *testing.T, users []user.User) {
				assert.Len(t, users, 3)
			},
		},
		{
			name: "second page with pagination",
			request: user.ListRequest{
				Limit:  3,
				Offset: 3,
			},
			wantLen: 2,
		},
		{
			name: "filter by client role",
			request: user.ListRequest{
				Filter: user.Filter{
					UserRole: user.RoleClient,
				},
				Limit: 10,
			},
			wantLen: 2,
			validate: func(t *testing.T, users []user.User) {
				for _, u := range users {
					assert.Equal(t, user.RoleClient, u.UserRole)
				}
			},
		},
		{
			name: "filter by admin role",
			request: user.ListRequest{
				Filter: user.Filter{
					UserRole: user.RoleAdmin,
				},
				Limit: 10,
			},
			wantLen: 3,
			validate: func(t *testing.T, users []user.User) {
				for _, u := range users {
					assert.Equal(t, user.RoleAdmin, u.UserRole)
				}
			},
		},
		{
			name: "search by first name",
			request: user.ListRequest{
				Filter: user.Filter{
					Search: "john",
				},
				Limit: 10,
			},
			wantLen: 1,
			validate: func(t *testing.T, users []user.User) {
				assert.Equal(t, "client1@example.com", users[0].Email)
			},
		},
		{
			name: "search by last name",
			request: user.ListRequest{
				Filter: user.Filter{
					Search: "smith",
				},
				Limit: 10,
			},
			wantLen: 1,
			validate: func(t *testing.T, users []user.User) {
				assert.Equal(t, "client2@example.com", users[0].Email)
			},
		},
		{
			name: "search by email",
			request: user.ListRequest{
				Filter: user.Filter{
					Search: "admin1",
				},
				Limit: 10,
			},
			wantLen: 1,
			validate: func(t *testing.T, users []user.User) {
				assert.Equal(t, "admin1@example.com", users[0].Email)
			},
		},
		{
			name: "search with multiple results",
			request: user.ListRequest{
				Filter: user.Filter{
					Search: "admin",
				},
				Limit: 10,
			},
			wantLen: 2,
		},
		{
			name: "empty result for non-matching search",
			request: user.ListRequest{
				Filter: user.Filter{
					Search: "nonexistent",
				},
				Limit: 10,
			},
			wantLen: 0,
		},
		{
			name: "large limit returns all",
			request: user.ListRequest{
				Limit: 100,
			},
			wantLen: 5,
		},
		{
			name: "zero limit returns empty",
			request: user.ListRequest{
				Limit: 0,
			},
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, truncateTables())
			setupTestUsers()

			users, err := testRepo.List(testCtx, tt.request)

			require.NoError(t, err)
			assert.Len(t, users, tt.wantLen)

			if tt.validate != nil {
				tt.validate(t, users)
			}
		})
	}
}

func TestListUsers_EmptyDatabase(t *testing.T) {
	tests := []struct {
		name    string
		request user.ListRequest
		wantLen int
	}{
		{
			name: "empty database with default request",
			request: user.ListRequest{
				Limit:  10,
				Offset: 0,
			},
			wantLen: 0,
		},
		{
			name: "empty database with filter",
			request: user.ListRequest{
				Filter: user.Filter{
					UserRole: "client",
				},
				Limit: 10,
			},
			wantLen: 0,
		},
		{
			name: "empty database with search",
			request: user.ListRequest{
				Filter: user.Filter{
					Search: "anything",
				},
				Limit: 10,
			},
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, truncateTables())

			users, err := testRepo.List(testCtx, tt.request)

			require.NoError(t, err)
			assert.Empty(t, users)
			assert.Len(t, users, tt.wantLen)
		})
	}
}
