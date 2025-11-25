package propertytypesvc

import (
	"context"
	"log/slog"
	"os"
	"testing"

	dto "github.com/Oleja123/estate-agency/internal/application/property_type/dto"
	apperrors "github.com/Oleja123/estate-agency/internal/application/property_type/errors"
	propertytype "github.com/Oleja123/estate-agency/internal/domain/property_type"
	basedberrors "github.com/Oleja123/estate-agency/internal/infrastructure/basedb/basedberrors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockRepo is a lightweight mock implementing propertytype.Repository for tests.
type mockRepo struct {
	GetByNameFunc func(ctx context.Context, name string) (propertytype.PropertyType, error)
	CreateFunc    func(ctx context.Context, pt propertytype.PropertyType) (int, error)
	UpdateFunc    func(ctx context.Context, pt propertytype.PropertyType) error
}

func (f *mockRepo) Create(ctx context.Context, propertyType propertytype.PropertyType) (int, error) {
	if f.CreateFunc != nil {
		return f.CreateFunc(ctx, propertyType)
	}
	return 0, nil
}

func (f *mockRepo) GetByID(ctx context.Context, id int) (propertytype.PropertyType, error) {
	return propertytype.PropertyType{}, basedberrors.NewErrNotFound("property_type", id)
}

func (f *mockRepo) GetByName(ctx context.Context, name string) (propertytype.PropertyType, error) {
	if f.GetByNameFunc != nil {
		return f.GetByNameFunc(ctx, name)
	}
	return propertytype.PropertyType{}, basedberrors.NewErrNotFound("property_type", name)
}

func (f *mockRepo) Update(ctx context.Context, propertyType propertytype.PropertyType) error {
	if f.UpdateFunc != nil {
		return f.UpdateFunc(ctx, propertyType)
	}
	return nil
}

func (f *mockRepo) Delete(ctx context.Context, id int) error { return nil }
func (f *mockRepo) List(ctx context.Context, req propertytype.ListRequest) ([]propertytype.PropertyType, error) {
	return nil, nil
}

func makeLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestService_Create(t *testing.T) {
	ctx := context.Background()
	logger := makeLogger()
	tests := []struct {
		name        string
		repoFactory func() *mockRepo
		req         dto.CreateRequest
		wantID      int
		wantErr     bool
		wantErrType string // "invalid" | "already"
	}{
		{
			name: "success",
			repoFactory: func() *mockRepo {
				return &mockRepo{
					GetByNameFunc: func(ctx context.Context, name string) (propertytype.PropertyType, error) {
						return propertytype.PropertyType{}, basedberrors.NewErrNotFound("property_type", name)
					},
					CreateFunc: func(ctx context.Context, pt propertytype.PropertyType) (int, error) {
						return 42, nil
					},
				}
			},
			req:     dto.CreateRequest{Name: "apartment"},
			wantID:  42,
			wantErr: false,
		},
		{
			name: "empty_name",
			repoFactory: func() *mockRepo {
				return &mockRepo{}
			},
			req:         dto.CreateRequest{Name: ""},
			wantErr:     true,
			wantErrType: "invalid",
		},
		{
			name: "already_exists",
			repoFactory: func() *mockRepo {
				return &mockRepo{
					GetByNameFunc: func(ctx context.Context, name string) (propertytype.PropertyType, error) {
						return propertytype.PropertyType{Id: 7, Name: name}, nil
					},
				}
			},
			req:         dto.CreateRequest{Name: "house"},
			wantErr:     true,
			wantErrType: "already",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := tc.repoFactory()
			svc := New(repo, logger)

			id, err := svc.Create(ctx, tc.req)
			if tc.wantErr {
				require.Error(t, err)
				switch tc.wantErrType {
				case "invalid":
					_, ok := err.(apperrors.ErrInvalidInput)
					assert.True(t, ok)
				case "already":
					_, ok := err.(apperrors.ErrAlreadyExists)
					assert.True(t, ok)
				default:
					t.Fatalf("unknown wantErrType %s", tc.wantErrType)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.wantID, id)
			}
		})
	}
}

func TestService_Update(t *testing.T) {
	ctx := context.Background()
	logger := makeLogger()
	tests := []struct {
		name        string
		repoFactory func() *mockRepo
		req         dto.UpdateRequest
		wantErr     bool
		wantErrType string // "invalid" | "already"
	}{
		{
			name: "success",
			repoFactory: func() *mockRepo {
				return &mockRepo{
					GetByNameFunc: func(ctx context.Context, name string) (propertytype.PropertyType, error) {
						return propertytype.PropertyType{}, basedberrors.NewErrNotFound("property_type", name)
					},
					UpdateFunc: func(ctx context.Context, pt propertytype.PropertyType) error {
						return nil
					},
				}
			},
			req:     dto.UpdateRequest{Id: 5, Name: "updated"},
			wantErr: false,
		},
		{
			name:        "empty_name",
			repoFactory: func() *mockRepo { return &mockRepo{} },
			req:         dto.UpdateRequest{Id: 5, Name: ""},
			wantErr:     true,
			wantErrType: "invalid",
		},
		{
			name: "name_conflict",
			repoFactory: func() *mockRepo {
				return &mockRepo{
					GetByNameFunc: func(ctx context.Context, name string) (propertytype.PropertyType, error) {
						return propertytype.PropertyType{Id: 9, Name: name}, nil
					},
				}
			},
			req:         dto.UpdateRequest{Id: 5, Name: "duplicate"},
			wantErr:     true,
			wantErrType: "already",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := tc.repoFactory()
			svc := New(repo, logger)

			err := svc.Update(ctx, tc.req)
			if tc.wantErr {
				require.Error(t, err)
				switch tc.wantErrType {
				case "invalid":
					_, ok := err.(apperrors.ErrInvalidInput)
					assert.True(t, ok)
				case "already":
					_, ok := err.(apperrors.ErrAlreadyExists)
					assert.True(t, ok)
				default:
					t.Fatalf("unknown wantErrType %s", tc.wantErrType)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}
