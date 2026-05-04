package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	slugdomain "simple-license-server/internal/domain/slug"
)

const (
	DefaultOfflineTokenLifetimeHours   = 24
	DefaultOfflineTokenLifetimeSeconds = DefaultOfflineTokenLifetimeHours * 3600
)

type SlugRecord struct {
	ID                          int64
	Name                        string
	MaxActivations              int
	ExpirationType              string
	ExpirationDays              *int
	FixedExpiresAt              *time.Time
	OfflineEnabled              bool
	OfflineTokenLifetimeSeconds int
	IsDefault                   bool
	DeletedAt                   *time.Time
	CreatedAt                   time.Time
	UpdatedAt                   time.Time
}

type CreateSlugParams struct {
	Name                        string
	MaxActivations              int
	ExpirationType              string
	ExpirationDays              *int
	FixedExpiresAt              *time.Time
	OfflineEnabled              bool
	OfflineTokenLifetimeSeconds int
}

type UpdateSlugParams struct {
	Name                        *string
	MaxActivations              *int
	ExpirationType              *string
	ExpirationDays              *int
	FixedExpiresAt              **time.Time
	OfflineEnabled              *bool
	OfflineTokenLifetimeSeconds *int
}

func (s *Store) ListSlugs(ctx context.Context, includeArchived bool) ([]SlugRecord, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id,
		       name,
		       max_activations,
		       expiration_type,
		       expiration_days,
		       fixed_expires_at,
		       offline_enabled,
		       offline_token_lifetime_seconds,
		       is_default,
		       deleted_at,
		       created_at,
		       updated_at
		FROM slugs
		WHERE ($1 OR deleted_at IS NULL)
		ORDER BY created_at ASC
	`, includeArchived)
	if err != nil {
		return nil, fmt.Errorf("list slugs: %w", err)
	}
	defer rows.Close()

	items := make([]SlugRecord, 0)
	for rows.Next() {
		record, err := scanSlugRecord(rows)
		if err != nil {
			return nil, err
		}

		items = append(items, record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate slugs: %w", err)
	}

	return items, nil
}

func (s *Store) GetSlugByName(ctx context.Context, name string) (SlugRecord, error) {
	slugName, err := slugdomain.ParseName(name)
	if err != nil {
		return SlugRecord{}, fmt.Errorf("invalid slug name: %w", err)
	}

	row := s.db.QueryRow(ctx, `
		SELECT id,
		       name,
		       max_activations,
		       expiration_type,
		       expiration_days,
		       fixed_expires_at,
		       offline_enabled,
		       offline_token_lifetime_seconds,
		       is_default,
		       deleted_at,
		       created_at,
		       updated_at
		FROM slugs
		WHERE name = $1
		  AND deleted_at IS NULL
	`, slugName.String())

	record, err := scanSlugRecord(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return SlugRecord{}, ErrNotFound
		}
		return SlugRecord{}, err
	}

	return record, nil
}

func (s *Store) CreateSlug(ctx context.Context, params CreateSlugParams) (SlugRecord, error) {
	slugName, err := slugdomain.ParseName(params.Name)
	if err != nil {
		return SlugRecord{}, fmt.Errorf("invalid slug name: %w", err)
	}

	policy, err := slugdomain.NewPolicy(params.MaxActivations, params.ExpirationType, params.ExpirationDays, params.FixedExpiresAt)
	if err != nil {
		return SlugRecord{}, fmt.Errorf("invalid slug policy: %w", err)
	}

	expirationDays := sql.NullInt32{}
	if days := policy.ExpirationDays(); days != nil {
		expirationDays = sql.NullInt32{Int32: int32(*days), Valid: true}
	}

	fixedExpiresAt := sql.NullTime{}
	if fixed := policy.FixedExpiresAt(); fixed != nil {
		fixedExpiresAt = sql.NullTime{Time: fixed.UTC(), Valid: true}
	}

	offlineTokenLifetimeSeconds := params.OfflineTokenLifetimeSeconds
	if offlineTokenLifetimeSeconds <= 0 {
		offlineTokenLifetimeSeconds = DefaultOfflineTokenLifetimeSeconds
	}

	row := s.db.QueryRow(ctx, `
		INSERT INTO slugs (name, max_activations, expiration_type, expiration_days, fixed_expires_at, offline_enabled, offline_token_lifetime_seconds, is_default)
		VALUES ($1, $2, $3, $4, $5, $6, $7, FALSE)
		RETURNING id,
		          name,
		          max_activations,
		          expiration_type,
		          expiration_days,
		          fixed_expires_at,
		          offline_enabled,
		          offline_token_lifetime_seconds,
		          is_default,
		          deleted_at,
		          created_at,
		          updated_at
	`,
		slugName.String(),
		policy.MaxActivations(),
		policy.ExpirationType(),
		expirationDays,
		fixedExpiresAt,
		params.OfflineEnabled,
		offlineTokenLifetimeSeconds,
	)

	record, err := scanSlugRecord(row)
	if err != nil {
		if isUniqueViolation(err) {
			return SlugRecord{}, ErrConflict
		}
		return SlugRecord{}, fmt.Errorf("create slug: %w", err)
	}

	return record, nil
}

func (s *Store) UpdateSlugByName(ctx context.Context, currentName string, params UpdateSlugParams) (SlugRecord, error) {
	current, err := s.GetSlugByName(ctx, currentName)
	if err != nil {
		return SlugRecord{}, err
	}

	name := current.Name
	if params.Name != nil {
		name = strings.TrimSpace(*params.Name)
	}

	maxActivations := current.MaxActivations
	if params.MaxActivations != nil {
		maxActivations = *params.MaxActivations
	}

	expirationType := current.ExpirationType
	if params.ExpirationType != nil {
		expirationType = strings.TrimSpace(*params.ExpirationType)
	}

	offlineEnabled := current.OfflineEnabled
	if params.OfflineEnabled != nil {
		offlineEnabled = *params.OfflineEnabled
	}

	offlineTokenLifetimeSeconds := current.OfflineTokenLifetimeSeconds
	if params.OfflineTokenLifetimeSeconds != nil {
		offlineTokenLifetimeSeconds = *params.OfflineTokenLifetimeSeconds
	}
	if offlineTokenLifetimeSeconds <= 0 {
		return SlugRecord{}, fmt.Errorf("offline_token_lifetime_seconds must be greater than 0")
	}

	var expirationDays sql.NullInt32
	if params.ExpirationDays != nil {
		expirationDays = sql.NullInt32{Int32: int32(*params.ExpirationDays), Valid: true}
	} else if current.ExpirationDays != nil {
		expirationDays = sql.NullInt32{Int32: int32(*current.ExpirationDays), Valid: true}
	}

	var fixedExpiresAt sql.NullTime
	if params.FixedExpiresAt != nil {
		if *params.FixedExpiresAt != nil {
			fixedExpiresAt = sql.NullTime{Time: (*params.FixedExpiresAt).UTC(), Valid: true}
		}
	} else if current.FixedExpiresAt != nil {
		fixedExpiresAt = sql.NullTime{Time: current.FixedExpiresAt.UTC(), Valid: true}
	}

	var expirationDaysPtr *int
	if expirationDays.Valid {
		v := int(expirationDays.Int32)
		expirationDaysPtr = &v
	}

	var fixedExpiresAtPtr *time.Time
	if fixedExpiresAt.Valid {
		v := fixedExpiresAt.Time.UTC()
		fixedExpiresAtPtr = &v
	}

	validatedName, err := slugdomain.ParseName(name)
	if err != nil {
		return SlugRecord{}, fmt.Errorf("invalid slug name: %w", err)
	}

	policy, err := slugdomain.NewPolicy(maxActivations, expirationType, expirationDaysPtr, fixedExpiresAtPtr)
	if err != nil {
		return SlugRecord{}, fmt.Errorf("invalid slug policy: %w", err)
	}

	expirationDays = sql.NullInt32{}
	if days := policy.ExpirationDays(); days != nil {
		expirationDays = sql.NullInt32{Int32: int32(*days), Valid: true}
	}

	fixedExpiresAt = sql.NullTime{}
	if fixed := policy.FixedExpiresAt(); fixed != nil {
		fixedExpiresAt = sql.NullTime{Time: fixed.UTC(), Valid: true}
	}

	row := s.db.QueryRow(ctx, `
		UPDATE slugs
		SET name = $1,
		    max_activations = $2,
		    expiration_type = $3,
		    expiration_days = $4,
		    fixed_expires_at = $5,
		    offline_enabled = $6,
		    offline_token_lifetime_seconds = $7,
		    updated_at = NOW()
		WHERE id = $8
		RETURNING id,
		          name,
		          max_activations,
		          expiration_type,
		          expiration_days,
		          fixed_expires_at,
		          offline_enabled,
		          offline_token_lifetime_seconds,
		          is_default,
		          deleted_at,
		          created_at,
		          updated_at
	`,
		validatedName.String(),
		policy.MaxActivations(),
		policy.ExpirationType(),
		expirationDays,
		fixedExpiresAt,
		offlineEnabled,
		offlineTokenLifetimeSeconds,
		current.ID,
	)

	record, err := scanSlugRecord(row)
	if err != nil {
		if isUniqueViolation(err) {
			return SlugRecord{}, ErrConflict
		}
		if errors.Is(err, pgx.ErrNoRows) {
			return SlugRecord{}, ErrNotFound
		}
		return SlugRecord{}, fmt.Errorf("update slug: %w", err)
	}

	return record, nil
}

func (s *Store) DeleteSlugByName(ctx context.Context, name string) error {
	slugName, err := slugdomain.ParseName(name)
	if err != nil {
		return fmt.Errorf("invalid slug name: %w", err)
	}

	name = slugName.String()

	var (
		slugID    int64
		isDefault bool
	)

	err = s.db.QueryRow(ctx, `
		SELECT id, is_default
		FROM slugs
		WHERE name = $1
		  AND deleted_at IS NULL
	`, name).Scan(&slugID, &isDefault)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("query slug before delete: %w", err)
	}

	if isDefault {
		return ErrConflict
	}

	var licenseCount int
	err = s.db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM licenses
		WHERE slug_id = $1
		  AND status <> 'revoked'
		  AND (expires_at IS NULL OR expires_at > NOW())
	`, slugID).Scan(&licenseCount)
	if err != nil {
		return fmt.Errorf("count slug licenses: %w", err)
	}

	if licenseCount > 0 {
		return ErrConflict
	}

	result, err := s.db.Exec(ctx, `
		UPDATE slugs
		SET deleted_at = NOW(),
		    updated_at = NOW()
		WHERE id = $1
		  AND deleted_at IS NULL
	`, slugID)
	if err != nil {
		return fmt.Errorf("delete slug: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

type slugScanner interface {
	Scan(dest ...any) error
}

func scanSlugRecord(row slugScanner) (SlugRecord, error) {
	var (
		record         SlugRecord
		expirationDays sql.NullInt32
		fixedExpiresAt sql.NullTime
		deletedAt      sql.NullTime
	)

	err := row.Scan(
		&record.ID,
		&record.Name,
		&record.MaxActivations,
		&record.ExpirationType,
		&expirationDays,
		&fixedExpiresAt,
		&record.OfflineEnabled,
		&record.OfflineTokenLifetimeSeconds,
		&record.IsDefault,
		&deletedAt,
		&record.CreatedAt,
		&record.UpdatedAt,
	)
	if err != nil {
		return SlugRecord{}, err
	}

	if expirationDays.Valid {
		v := int(expirationDays.Int32)
		record.ExpirationDays = &v
	}

	if fixedExpiresAt.Valid {
		v := fixedExpiresAt.Time.UTC()
		record.FixedExpiresAt = &v
	}

	if deletedAt.Valid {
		v := deletedAt.Time.UTC()
		record.DeletedAt = &v
	}

	record.CreatedAt = record.CreatedAt.UTC()
	record.UpdatedAt = record.UpdatedAt.UTC()

	return record, nil
}
