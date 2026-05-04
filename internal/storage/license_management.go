package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
)

func (s *Store) ListLicenses(ctx context.Context, params LicenseListParams) (LicenseListResult, error) {
	page := params.Page
	if page <= 0 {
		page = 1
	}

	pageSize := params.PageSize
	if pageSize <= 0 {
		pageSize = 10
	}

	searchPattern := ""
	if search := strings.TrimSpace(params.Search); search != "" {
		searchPattern = "%" + escapeLikePattern(search) + "%"
	}

	status := strings.TrimSpace(params.Status)
	limit := pageSize
	offset := (page - 1) * pageSize

	total, err := s.countLicenses(ctx, searchPattern, status)
	if err != nil {
		return LicenseListResult{}, err
	}

	counts, err := s.countLicenseStatuses(ctx)
	if err != nil {
		return LicenseListResult{}, err
	}

	rows, err := s.db.Query(ctx, `
		SELECT l.id,
		       l.key,
		       l.status,
		       l.expires_at,
		       l.created_at,
		       l.activated_at,
		       (
		           SELECT MAX(a.last_validated_at)
		           FROM activations a
		           WHERE a.license_id = l.id
		       ) AS last_validated_at,
		       l.revoked_at,
		       l.metadata,
		       s.name,
		       s.offline_enabled,
		       s.offline_token_lifetime_seconds,
		       l.max_activations,
		       (
		           SELECT COUNT(*)
		           FROM activations a
		           WHERE a.license_id = l.id
		             AND a.deactivated_at IS NULL
		       ) AS active_seats
		FROM licenses l
		JOIN slugs s ON s.id = l.slug_id
		WHERE ($1 = '' OR l.key ILIKE $1 ESCAPE '\' OR s.name ILIKE $1 ESCAPE '\' OR l.metadata::text ILIKE $1 ESCAPE '\')
		  AND ($2 = '' OR (
		      CASE
		          WHEN l.status <> 'revoked' AND l.expires_at IS NOT NULL AND l.expires_at <= NOW() THEN 'expired'
		          ELSE l.status
		      END
		  ) = $2)
		ORDER BY l.created_at DESC, l.id DESC
		LIMIT $3 OFFSET $4
	`, searchPattern, status, limit, offset)
	if err != nil {
		return LicenseListResult{}, fmt.Errorf("list licenses: %w", err)
	}
	defer rows.Close()

	licenses := make([]LicenseRow, 0)
	for rows.Next() {
		license, err := scanLicenseListRow(rows)
		if err != nil {
			return LicenseListResult{}, err
		}

		licenses = append(licenses, license)
	}

	if err := rows.Err(); err != nil {
		return LicenseListResult{}, fmt.Errorf("iterate licenses: %w", err)
	}

	return LicenseListResult{
		Licenses: licenses,
		Total:    total,
		Counts:   counts,
	}, nil
}

func (s *Store) countLicenses(ctx context.Context, searchPattern, status string) (int, error) {
	var total int
	err := s.db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM licenses l
		JOIN slugs s ON s.id = l.slug_id
		WHERE ($1 = '' OR l.key ILIKE $1 ESCAPE '\' OR s.name ILIKE $1 ESCAPE '\' OR l.metadata::text ILIKE $1 ESCAPE '\')
		  AND ($2 = '' OR (
		      CASE
		          WHEN l.status <> 'revoked' AND l.expires_at IS NOT NULL AND l.expires_at <= NOW() THEN 'expired'
		          ELSE l.status
		      END
		  ) = $2)
	`, searchPattern, status).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("count licenses: %w", err)
	}

	return total, nil
}

func (s *Store) countLicenseStatuses(ctx context.Context) (LicenseStatusCounts, error) {
	var counts LicenseStatusCounts
	err := s.db.QueryRow(ctx, `
		SELECT COUNT(*),
		       COUNT(*) FILTER (WHERE status = 'active' AND NOT expired),
		       COUNT(*) FILTER (WHERE status = 'inactive' AND NOT expired),
		       COUNT(*) FILTER (WHERE status = 'revoked'),
		       COUNT(*) FILTER (WHERE expired)
		FROM (
		    SELECT status,
		           status <> 'revoked' AND expires_at IS NOT NULL AND expires_at <= NOW() AS expired
		    FROM licenses
		) license_statuses
	`).Scan(&counts.Total, &counts.Active, &counts.Inactive, &counts.Revoked, &counts.Expired)
	if err != nil {
		return LicenseStatusCounts{}, fmt.Errorf("count license statuses: %w", err)
	}

	return counts, nil
}

type licenseListScanner interface {
	Scan(dest ...any) error
}

func scanLicenseListRow(row licenseListScanner) (LicenseRow, error) {
	var (
		license         LicenseRow
		expiresAt       sql.NullTime
		activatedAt     sql.NullTime
		lastValidatedAt sql.NullTime
		revokedAt       sql.NullTime
		metadataJSON    []byte
	)

	err := row.Scan(
		&license.ID,
		&license.Key,
		&license.Status,
		&expiresAt,
		&license.CreatedAt,
		&activatedAt,
		&lastValidatedAt,
		&revokedAt,
		&metadataJSON,
		&license.SlugName,
		&license.OfflineEnabled,
		&license.OfflineTokenLifetimeSeconds,
		&license.MaxActivations,
		&license.ActiveSeats,
	)
	if err != nil {
		return LicenseRow{}, fmt.Errorf("scan license row: %w", err)
	}

	if expiresAt.Valid {
		v := expiresAt.Time.UTC()
		license.ExpiresAt = &v
	}
	if activatedAt.Valid {
		v := activatedAt.Time.UTC()
		license.ActivatedAt = &v
	}
	if lastValidatedAt.Valid {
		v := lastValidatedAt.Time.UTC()
		license.LastValidatedAt = &v
	}
	if revokedAt.Valid {
		v := revokedAt.Time.UTC()
		license.RevokedAt = &v
	}

	license.CreatedAt = license.CreatedAt.UTC()
	if len(metadataJSON) == 0 {
		license.Metadata = map[string]any{}
		return license, nil
	}

	if err := json.Unmarshal(metadataJSON, &license.Metadata); err != nil {
		return LicenseRow{}, fmt.Errorf("decode license metadata: %w", err)
	}

	return license, nil
}

func escapeLikePattern(value string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return replacer.Replace(value)
}
