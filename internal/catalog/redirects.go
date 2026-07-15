package catalog

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

func (s *Store) RedirectTarget(ctx context.Context, entityType, mbid string) (string, error) {
	var newMBID string
	err := s.DB().QueryRowContext(ctx,
		`SELECT new_mbid FROM gid_redirects WHERE entity_type = ? AND old_mbid = ?`,
		entityType, mbid).Scan(&newMBID)
	if errors.Is(err, sql.ErrNoRows) || isMissingRedirectsTable(err) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("gid redirect %s %s: %w", entityType, mbid, err)
	}
	return newMBID, nil
}

func isMissingRedirectsTable(err error) bool {
	return err != nil && strings.Contains(err.Error(), "no such table")
}
