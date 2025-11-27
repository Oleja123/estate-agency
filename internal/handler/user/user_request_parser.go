package userhandler

import (
	"net/http"
	"strconv"
	"strings"

	dto "github.com/Oleja123/estate-agency/internal/application/user/dto"
	domain "github.com/Oleja123/estate-agency/internal/domain/user"
)

// parseListUsersRequest parses query parameters from the request into
// dto.ListUsersRequest. It is intentionally lenient: invalid values are
// ignored and defaults are used.
func parseListUsersRequest(r *http.Request) (dto.ListUsersRequest, error) {
	q := r.URL.Query()

	// pagination
	limit := 20
	if l := q.Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 {
			limit = v
		}
	}
	if limit > 500 {
		limit = 500
	}
	offset := 0
	if o := q.Get("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil && v >= 0 {
			offset = v
		}
	}

	filt := domain.Filter{}
	if em := q.Get("email"); em != "" {
		filt.Email = em
	}
	if role := q.Get("role"); role != "" {
		if rrole, err := domain.ParseRole(role); err == nil {
			filt.UserRole = rrole
		}
	}
	if sa := q.Get("search"); sa != "" {
		filt.Search = sa
	}
	if isAct := q.Get("is_active"); isAct != "" {
		if b, err := strconv.ParseBool(isAct); err == nil {
			filt.IsActive = &b
		}
	}
	if ids := q.Get("ids"); ids != "" {
		parts := strings.Split(ids, ",")
		for _, p := range parts {
			if p = strings.TrimSpace(p); p != "" {
				if v, err := strconv.Atoi(p); err == nil {
					filt.IDs = append(filt.IDs, v)
				}
			}
		}
	}

	return dto.ListUsersRequest{Filter: filt, Limit: limit, Offset: offset}, nil
}
