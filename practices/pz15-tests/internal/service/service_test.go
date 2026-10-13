package service_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/Pondaaaaaa/GO/practices/pz15-tests/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This test-local implementation records calls but needs no external resources.
type stubRepo struct {
	users map[string]service.User
	err   error
	calls []string
}

func (r *stubRepo) ByEmail(email string) (service.User, error) {
	r.calls = append(r.calls, email)
	if r.err != nil {
		return service.User{ID: 999}, r.err
	}
	u, ok := r.users[email]
	if !ok {
		return service.User{}, service.ErrNotFound
	}
	return u, nil
}

var _ service.UserRepo = (*stubRepo)(nil)

func TestService_FindIDByEmail(t *testing.T) {
	storageErr := errors.New("storage unavailable")
	wrappedNotFound := fmt.Errorf("lookup: %w", service.ErrNotFound)
	cases := []struct {
		name, email string
		repoErr     error
		wantID      int64
		wantErr     error
	}{
		{"found", "reader@example.com", nil, 42, nil},
		{"not_found", "missing@example.com", nil, 0, service.ErrNotFound},
		{"empty_email_forwarded", "", nil, 0, service.ErrNotFound},
		{"repository_error", "reader@example.com", storageErr, 0, storageErr},
		{"wrapped_not_found", "reader@example.com", wrappedNotFound, 0, service.ErrNotFound},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &stubRepo{users: map[string]service.User{
				"reader@example.com": {ID: 42, Email: "reader@example.com"},
			}, err: tc.repoErr}
			svc := service.New(repo)
			id, err := svc.FindIDByEmail(tc.email)
			if tc.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tc.wantErr)
				if tc.repoErr != nil {
					assert.Same(t, tc.repoErr, err, "repository error must be returned unchanged")
				}
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tc.wantID, id)
			assert.Equal(t, []string{tc.email}, repo.calls, "forward the exact email once")
		})
	}
}
