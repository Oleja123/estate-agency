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
	"github.com/Oleja123/estate-agency/internal/infrastructure/testdb"
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
		GoosePath: "/home/oleg/go/bin/goose",
	}

	// try to ensure singleton test container and run migrations from code; if docker unavailable, fail fast
	tdb, err := testdb.EnsureStarted(testCtx, testLogger)
	if err != nil {
		testLogger.Error("Failed to start test DB container", "error", err)
		os.Exit(1)
	}
	defer tdb.Terminate()
	// update config to point to the container
	testConfig.DbConfig.Host = tdb.Host
	testConfig.DbConfig.Port = tdb.Port

	testClient, _ = postgresqlclient.NewClient(context.Background(), *testLogger, testConfig)
	testRepo = New(testClient, testLogger)

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
			name: "successful_creation",
			user: func() user.User {
				v := "+1234567890"
				return user.User{
					Email:        "success@example.com",
					PasswordHash: "hashed_password",
					FirstName:    "John",
					LastName:     "Doe",
					PhoneNumber:  &v,
					UserRole:     "client",
				}
			}(),
			wantErr: false,
			validate: func(t *testing.T, userID int) {
				createdUser, err := testRepo.GetByID(testCtx, userID)
				require.NoError(t, err)
				assert.Equal(t, "success@example.com", createdUser.Email)
				assert.True(t, createdUser.IsActive)
			},
		},
		{
			name: "duplicate_email",
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
			name: "user_with_all_fields",
			user: func() user.User {
				v := "+1111111111"
				return user.User{
					Email:        "full@example.com",
					PasswordHash: "full_hash",
					FirstName:    "Full",
					LastName:     "User",
					PhoneNumber:  &v,
					UserRole:     "admin",
				}
			}(),
			wantErr: false,
			validate: func(t *testing.T, userID int) {
				createdUser, err := testRepo.GetByID(testCtx, userID)
				require.NoError(t, err)
				assert.Equal(t, "full@example.com", createdUser.Email)
				assert.Equal(t, "Full", createdUser.FirstName)
				assert.Equal(t, "User", createdUser.LastName)
				if assert.NotNil(t, createdUser.PhoneNumber) {
					assert.Equal(t, "+1111111111", *createdUser.PhoneNumber)
				}
				assert.Equal(t, user.RoleAdmin, createdUser.UserRole)
			},
		},
		{
			name: "user_without_phone",
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
				assert.Nil(t, createdUser.PhoneNumber)
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
			name:    "successful_get_by_id",
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
			name:    "not_found",
			id:      99999,
			wantErr: true,
			errType: basedb.IsNotFound,
		},
		{
			name:    "zero_id",
			id:      0,
			wantErr: true,
			errType: basedb.IsNotFound,
		},
		{
			name:    "negative_id",
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
			name:    "successful_get_by_email",
			email:   "getbyemail_test@example.com",
			wantErr: false,
			validate: func(t *testing.T, foundUser user.User) {
				assert.Equal(t, "getbyemail_test@example.com", foundUser.Email)
				assert.Equal(t, "Email", foundUser.FirstName)
				assert.Equal(t, user.RoleAdmin, foundUser.UserRole)
			},
		},
		{
			name:    "email_not_found",
			email:   "nonexistent@example.com",
			wantErr: true,
			errType: basedb.IsNotFound,
		},
		{
			name:    "empty_email",
			email:   "",
			wantErr: true,
			errType: basedb.IsNotFound,
		},
		{
			name:    "case_sensitive_email",
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
			name: "successful_update_all_fields",
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
			updateUser: func() user.User {
				v := "+9999999999"
				return user.User{
					Email:       "update_all@example.com",
					FirstName:   "Updated",
					LastName:    "Name",
					PhoneNumber: &v,
					UserRole:    "admin",
					IsActive:    false,
				}
			}(),
			wantErr: false,
			validate: func(t *testing.T, userID int) {
				updated, err := testRepo.GetByID(testCtx, userID)
				require.NoError(t, err)
				assert.Equal(t, "Updated", updated.FirstName)
				assert.Equal(t, "Name", updated.LastName)
				if assert.NotNil(t, updated.PhoneNumber) {
					assert.Equal(t, "+9999999999", *updated.PhoneNumber)
				}
				assert.Equal(t, user.RoleAdmin, updated.UserRole)
				assert.False(t, updated.IsActive)
			},
		},
		{
			name: "update_only_first_name",
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
			name: "update_non_existent_user",
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
			name: "activate_user",
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

func TestDeleteUserTableDriven(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() int
		deleteID int
		wantErr  bool
		errType  func(error) bool
		validate func(t *testing.T, deletedID int)
	}{
		{
			name: "successful_delete",
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
			name: "delete_non_existent_user",
			setup: func() int {
				return 99999
			},
			deleteID: 99999,
			wantErr:  true,
			errType:  basedb.IsNotFound,
		},
		{
			name: "delete_already_deleted_user",
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
			name: "delete_with_zero_id",
			setup: func() int {
				return 0
			},
			deleteID: 0,
			wantErr:  true,
			errType:  basedb.IsNotFound,
		},
		{
			name: "delete_with_negative_id",
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

func TestListUsersTableDriven(t *testing.T) {
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
		name      string
		request   user.ListRequest
		wantLen   int // expected number of items in the returned slice
		wantTotal int // expected total matching rows regardless of pagination
		validate  func(t *testing.T, users []user.User)
	}{
		{
			name: "list_all_users_with_pagination",
			request: user.ListRequest{
				Limit:  3,
				Offset: 0,
			},
			wantLen:   3,
			wantTotal: 5,
			validate: func(t *testing.T, users []user.User) {
				require.Len(t, users, 3)
			},
		},
		{
			name: "second_page_with_pagination",
			request: user.ListRequest{
				Limit:  3,
				Offset: 3,
			},
			wantLen:   2,
			wantTotal: 5,
		},
		{
			name: "filter_by_client_role",
			request: user.ListRequest{
				Filter: user.Filter{
					UserRole: user.RoleClient,
				},
				Limit: 10,
			},
			wantLen:   2,
			wantTotal: 2,
			validate: func(t *testing.T, users []user.User) {
				for _, u := range users {
					assert.Equal(t, user.RoleClient, u.UserRole)
				}
			},
		},
		{
			name: "filter_by_admin_role",
			request: user.ListRequest{
				Filter: user.Filter{
					UserRole: user.RoleAdmin,
				},
				Limit: 10,
			},
			wantLen:   3,
			wantTotal: 3,
			validate: func(t *testing.T, users []user.User) {
				for _, u := range users {
					assert.Equal(t, user.RoleAdmin, u.UserRole)
				}
			},
		},
		{
			name: "search_by_first_name",
			request: user.ListRequest{
				Filter: user.Filter{
					Search: "john",
				},
				Limit: 10,
			},
			wantLen:   1,
			wantTotal: 1,
			validate: func(t *testing.T, users []user.User) {
				assert.Equal(t, "client1@example.com", users[0].Email)
			},
		},
		{
			name: "search_by_last_name",
			request: user.ListRequest{
				Filter: user.Filter{
					Search: "smith",
				},
				Limit: 10,
			},
			wantLen:   1,
			wantTotal: 1,
			validate: func(t *testing.T, users []user.User) {
				assert.Equal(t, "client2@example.com", users[0].Email)
			},
		},
		{
			name: "search_by_email",
			request: user.ListRequest{
				Filter: user.Filter{
					Search: "admin1",
				},
				Limit: 10,
			},
			wantLen:   1,
			wantTotal: 1,
			validate: func(t *testing.T, users []user.User) {
				assert.Equal(t, "admin1@example.com", users[0].Email)
			},
		},
		{
			name: "search_with_multiple_results",
			request: user.ListRequest{
				Filter: user.Filter{
					Search: "admin",
				},
				Limit: 10,
			},
			wantLen:   2,
			wantTotal: 2,
		},
		{
			name: "empty_result_for_non_matching_search",
			request: user.ListRequest{
				Filter: user.Filter{
					Search: "nonexistent",
				},
				Limit: 10,
			},
			wantLen:   0,
			wantTotal: 0,
		},
		{
			name: "large_limit_returns_all",
			request: user.ListRequest{
				Limit: 100,
			},
			wantLen:   5,
			wantTotal: 5,
		},
		{
			name: "zero_limit_returns_all",
			request: user.ListRequest{
				Limit: 0,
			},
			// treat limit==0 as "no limit" (return all matching rows)
			wantLen:   0,
			wantTotal: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, truncateTables())
			setupTestUsers()

			users, total, err := testRepo.List(testCtx, tt.request)

			require.NoError(t, err)
			// page length must match expected
			require.Len(t, users, tt.wantLen)
			// total must equal expected matching rows regardless of pagination
			assert.Equal(t, tt.wantTotal, total)

			if tt.validate != nil {
				tt.validate(t, users)
			}
		})
	}
}

func TestListUsersEmptyDatabase(t *testing.T) {
	tests := []struct {
		name    string
		request user.ListRequest
		wantLen int
	}{
		{
			name: "empty_database_with_default_request",
			request: user.ListRequest{
				Limit:  10,
				Offset: 0,
			},
			wantLen: 0,
		},
		{
			name: "empty_database_with_filter",
			request: user.ListRequest{
				Filter: user.Filter{
					UserRole: "client",
				},
				Limit: 10,
			},
			wantLen: 0,
		},
		{
			name: "empty_database_with_search",
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

			users, total, err := testRepo.List(testCtx, tt.request)

			require.NoError(t, err)
			require.Empty(t, users)
			require.Len(t, users, tt.wantLen)
			assert.Equal(t, tt.wantLen, total)
		})
	}
}
