package storage

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	licensedomain "simple-license-server/internal/domain/license"
	slugdomain "simple-license-server/internal/domain/slug"
)

var (
	ErrNotFound   = errors.New("not found")
	ErrConflict   = errors.New("conflict")
	ErrInProgress = errors.New("in progress")
)

//go:embed schema.sql
var schemaFS embed.FS

type Store struct {
	db *pgxpool.Pool
}

type queryable interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}

type GeneratedLicense struct {
	LicenseKey string
	Slug       string
	Status     string
	Metadata   map[string]any
	ExpiresAt  *time.Time
	CreatedAt  time.Time
}

type LicenseListParams struct {
	Page     int
	PageSize int
	Search   string
	Status   string
}

type LicenseStatusCounts struct {
	Total    int
	Active   int
	Inactive int
	Revoked  int
	Expired  int
}

type LicenseListResult struct {
	Licenses []LicenseRow
	Total    int
	Counts   LicenseStatusCounts
}

type RevokeResult struct {
	Valid      bool
	Status     string
	LicenseKey string
	RevokedAt  time.Time
}

type ActivationResult struct {
	Valid                       bool
	Status                      string
	LicenseID                   string
	LicenseKey                  string
	Slug                        string
	Fingerprint                 string
	ExpiresAt                   *time.Time
	OfflineEnabled              bool
	OfflineTokenLifetimeSeconds int
	Reason                      string
}

type ValidationResult struct {
	Valid                       bool
	Status                      string
	LicenseID                   string
	Slug                        string
	ExpiresAt                   *time.Time
	OfflineEnabled              bool
	OfflineTokenLifetimeSeconds int
	Reason                      string
}

type DeactivationResult struct {
	Valid          bool
	Released       bool
	Status         string
	ActiveSeats    int
	MaxActivations int
	ExpiresAt      *time.Time
}

type APIKeyRecord struct {
	ID        int64
	Name      string
	KeyType   string
	KeyHint   string
	CreatedAt time.Time
	RevokedAt *time.Time
}

type CreatedAPIKey struct {
	APIKey string
	Record APIKeyRecord
}

type CreateAPIKeyParams struct {
	Name    string
	KeyType string
}

type WebhookEndpoint struct {
	ID        int64
	Name      string
	URL       string
	Events    []string
	Enabled   bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

type CreateWebhookEndpointParams struct {
	Name    string
	URL     string
	Events  []string
	Enabled bool
}

type UpdateWebhookEndpointParams struct {
	Name    *string
	URL     *string
	Events  *[]string
	Enabled *bool
}

type WebhookDelivery struct {
	ID          int64
	EndpointURL string
	EventType   string
	Payload     map[string]any
	Attempts    int
	CreatedAt   time.Time
}

type WebhookDeliveryLog struct {
	ID                 int64
	EndpointID         int64
	EndpointName       string
	EndpointURL        string
	EventType          string
	Status             string
	Attempts           int
	LastResponseStatus *int
	LastError          *string
	NextAttemptAt      time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
	DeliveredAt        *time.Time
}

type LicenseRow struct {
	ID                          string
	Key                         string
	Status                      string
	ExpiresAt                   *time.Time
	CreatedAt                   time.Time
	ActivatedAt                 *time.Time
	LastValidatedAt             *time.Time
	RevokedAt                   *time.Time
	Metadata                    map[string]any
	SlugName                    string
	OfflineEnabled              bool
	OfflineTokenLifetimeSeconds int
	MaxActivations              int
	ActiveSeats                 int
}

type slugOptions struct {
	SlugID                      int64
	ExpirationType              string
	ExpirationDays              sql.NullInt32
	FixedExpiresAt              sql.NullTime
	MaxActivations              int
	OfflineEnabled              bool
	OfflineTokenLifetimeSeconds int
}

func New(ctx context.Context, databaseURL string) (*Store, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("connect postgres: %w", err)
	}

	store := &Store{db: pool}

	if err := store.Migrate(ctx); err != nil {
		pool.Close()
		return nil, err
	}

	return store, nil
}

func (s *Store) Close() {
	s.db.Close()
}

func (s *Store) Ping(ctx context.Context) error {
	return s.db.Ping(ctx)
}

func (s *Store) Migrate(ctx context.Context) error {
	schema, err := schemaFS.ReadFile("schema.sql")
	if err != nil {
		return fmt.Errorf("read schema: %w", err)
	}

	if _, err := s.db.Exec(ctx, string(schema)); err != nil {
		return fmt.Errorf("run schema migration: %w", err)
	}

	return nil
}

func (s *Store) GenerateLicenseIdempotent(ctx context.Context, endpoint, idemKey, requestHash, slugName string, metadata map[string]any) (GeneratedLicense, json.RawMessage, bool, error) {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return GeneratedLicense{}, nil, false, fmt.Errorf("begin idempotent generate tx: %w", err)
	}
	defer tx.Rollback(ctx)

	storedHash, storedResponse, err := getIdempotencyRecordTx(ctx, tx, endpoint, idemKey)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return GeneratedLicense{}, nil, false, err
	}

	if err == nil {
		if storedHash != requestHash {
			return GeneratedLicense{}, nil, false, ErrConflict
		}

		if len(storedResponse) > 0 {
			if err := tx.Commit(ctx); err != nil {
				return GeneratedLicense{}, nil, false, fmt.Errorf("commit idempotent read tx: %w", err)
			}

			return GeneratedLicense{}, storedResponse, true, nil
		}

		return GeneratedLicense{}, nil, false, ErrInProgress
	} else {
		if _, err := tx.Exec(ctx, `
			INSERT INTO idempotency_records (endpoint, idem_key, request_hash, response_body)
			VALUES ($1, $2, $3, NULL)
		`, endpoint, idemKey, requestHash); err != nil {
			if !isUniqueViolation(err) {
				return GeneratedLicense{}, nil, false, fmt.Errorf("create idempotency placeholder: %w", err)
			}

			storedHash, storedResponse, fetchErr := getIdempotencyRecordTx(ctx, tx, endpoint, idemKey)
			if fetchErr != nil {
				return GeneratedLicense{}, nil, false, fetchErr
			}

			if storedHash != requestHash {
				return GeneratedLicense{}, nil, false, ErrConflict
			}

			if len(storedResponse) > 0 {
				if err := tx.Commit(ctx); err != nil {
					return GeneratedLicense{}, nil, false, fmt.Errorf("commit idempotent read tx: %w", err)
				}

				return GeneratedLicense{}, storedResponse, true, nil
			}

			return GeneratedLicense{}, nil, false, ErrInProgress
		}
	}

	generated, err := generateLicenseWithQuerier(ctx, tx, slugName, metadata)
	if err != nil {
		return GeneratedLicense{}, nil, false, err
	}

	responseBody, err := marshalGenerateResponseBody(generated)
	if err != nil {
		return GeneratedLicense{}, nil, false, err
	}

	if _, err := tx.Exec(ctx, `
		UPDATE idempotency_records
		SET response_body = $1
		WHERE endpoint = $2 AND idem_key = $3
	`, responseBody, endpoint, idemKey); err != nil {
		return GeneratedLicense{}, nil, false, fmt.Errorf("persist idempotent response: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return GeneratedLicense{}, nil, false, fmt.Errorf("commit idempotent generate tx: %w", err)
	}

	return generated, responseBody, false, nil
}

func (s *Store) GenerateLicense(ctx context.Context, slugName string, metadata map[string]any) (GeneratedLicense, error) {
	return generateLicenseWithQuerier(ctx, s.db, slugName, metadata)
}

func generateLicenseWithQuerier(ctx context.Context, q queryable, slugName string, metadata map[string]any) (GeneratedLicense, error) {
	options, err := loadSlugOptionsByName(ctx, q, slugName)
	if err != nil {
		return GeneratedLicense{}, err
	}

	return createLicenseFromSlugOptions(ctx, q, slugName, options, metadata)
}

func loadSlugOptionsByName(ctx context.Context, q queryable, slugName string) (slugOptions, error) {
	var options slugOptions
	err := q.QueryRow(ctx, `
		SELECT id, expiration_type, expiration_days, fixed_expires_at, max_activations, offline_enabled, offline_token_lifetime_seconds
		FROM slugs
		WHERE name = $1
		  AND deleted_at IS NULL
	`, slugName).Scan(
		&options.SlugID,
		&options.ExpirationType,
		&options.ExpirationDays,
		&options.FixedExpiresAt,
		&options.MaxActivations,
		&options.OfflineEnabled,
		&options.OfflineTokenLifetimeSeconds,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return slugOptions{}, ErrNotFound
		}
		return slugOptions{}, fmt.Errorf("lookup slug options: %w", err)
	}

	var expirationDays *int
	if options.ExpirationDays.Valid {
		v := int(options.ExpirationDays.Int32)
		expirationDays = &v
	}

	var fixedExpiresAt *time.Time
	if options.FixedExpiresAt.Valid {
		v := options.FixedExpiresAt.Time.UTC()
		fixedExpiresAt = &v
	}

	if _, err := slugdomain.NewPolicy(options.MaxActivations, options.ExpirationType, expirationDays, fixedExpiresAt); err != nil {
		return slugOptions{}, fmt.Errorf("invalid slug policy in storage: %w", err)
	}

	return options, nil
}

func createLicenseFromSlugOptions(ctx context.Context, q queryable, slugName string, options slugOptions, metadata map[string]any) (GeneratedLicense, error) {
	expiresAt, err := resolveExpiration(options.ExpirationType, options.ExpirationDays, options.FixedExpiresAt)
	if err != nil {
		return GeneratedLicense{}, err
	}

	metadataJSON, err := metadataToJSON(metadata)
	if err != nil {
		return GeneratedLicense{}, err
	}

	var createdAt time.Time
	licenseKey := ""
	for i := 0; i < 8; i++ {
		licenseKey, err = generateLicenseKey()
		if err != nil {
			return GeneratedLicense{}, err
		}

		err = q.QueryRow(ctx, `
			INSERT INTO licenses (key, slug_id, status, metadata, expires_at, max_activations)
			VALUES ($1, $2, 'inactive', $3, $4, $5)
			RETURNING created_at
		`, licenseKey, options.SlugID, metadataJSON, expiresAt, options.MaxActivations).Scan(&createdAt)
		if err == nil {
			break
		}

		if !isUniqueViolation(err) {
			return GeneratedLicense{}, fmt.Errorf("insert license: %w", err)
		}
	}

	if err != nil {
		return GeneratedLicense{}, fmt.Errorf("failed to create unique license key")
	}

	return GeneratedLicense{
		LicenseKey: licenseKey,
		Slug:       slugName,
		Status:     "inactive",
		Metadata:   copyMetadata(metadata),
		ExpiresAt:  expiresAt,
		CreatedAt:  createdAt.UTC(),
	}, nil
}

func getIdempotencyRecordTx(ctx context.Context, tx pgx.Tx, endpoint, key string) (string, json.RawMessage, error) {
	var requestHash string
	var response []byte
	err := tx.QueryRow(ctx, `
		SELECT request_hash, response_body
		FROM idempotency_records
		WHERE endpoint = $1 AND idem_key = $2
		FOR UPDATE
	`, endpoint, key).Scan(&requestHash, &response)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil, ErrNotFound
		}
		return "", nil, fmt.Errorf("query idempotency record: %w", err)
	}

	return requestHash, json.RawMessage(response), nil
}

func marshalGenerateResponseBody(generated GeneratedLicense) ([]byte, error) {
	payload := struct {
		LicenseKey string         `json:"license_key"`
		Slug       string         `json:"slug"`
		Status     string         `json:"status"`
		Metadata   map[string]any `json:"metadata"`
		ExpiresAt  *time.Time     `json:"expires_at"`
		CreatedAt  time.Time      `json:"created_at"`
	}{
		LicenseKey: generated.LicenseKey,
		Slug:       generated.Slug,
		Status:     generated.Status,
		Metadata:   generated.Metadata,
		ExpiresAt:  generated.ExpiresAt,
		CreatedAt:  generated.CreatedAt,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal generate response: %w", err)
	}

	return body, nil
}

func (s *Store) RevokeLicense(ctx context.Context, licenseKey string) (RevokeResult, error) {
	var revokedAt time.Time
	err := s.db.QueryRow(ctx, `
		UPDATE licenses
		SET status = 'revoked', revoked_at = COALESCE(revoked_at, NOW())
		WHERE key = $1
		RETURNING revoked_at
	`, licenseKey).Scan(&revokedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return RevokeResult{}, ErrNotFound
		}
		return RevokeResult{}, fmt.Errorf("revoke license: %w", err)
	}

	return RevokeResult{
		Valid:      false,
		Status:     string(licensedomain.StatusRevoked),
		LicenseKey: licenseKey,
		RevokedAt:  revokedAt.UTC(),
	}, nil
}

func (s *Store) ActivateLicense(ctx context.Context, licenseKey, fingerprint string, metadata map[string]any) (ActivationResult, error) {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return ActivationResult{}, fmt.Errorf("begin activate tx: %w", err)
	}
	defer tx.Rollback(ctx)

	license, err := loadLicenseByKey(ctx, tx, licenseKey, true)
	if err != nil {
		return ActivationResult{}, err
	}

	aggregate, err := licensedomain.Rehydrate(licensedomain.RehydrateParams{
		Status:         license.Status,
		ExpiresAt:      license.ExpiresAt,
		MaxActivations: license.MaxActivations,
	})
	if err != nil {
		return ActivationResult{}, fmt.Errorf("rehydrate license aggregate: %w", err)
	}

	now := time.Now().UTC()
	activeActivationID, err := findActiveActivationID(ctx, tx, license.ID, fingerprint)
	if err != nil {
		return ActivationResult{}, err
	}

	activeSeats := 0
	if activeActivationID == 0 {
		activeSeats, err = countActiveSeats(ctx, tx, license.ID)
		if err != nil {
			return ActivationResult{}, err
		}
	}

	decision := aggregate.Activate(now, activeSeats, activeActivationID > 0)

	if !decision.Valid {
		return ActivationResult{
			Valid:                       false,
			Status:                      string(decision.Status),
			LicenseID:                   license.ID,
			LicenseKey:                  license.Key,
			Slug:                        license.SlugName,
			Fingerprint:                 fingerprint,
			ExpiresAt:                   license.ExpiresAt,
			OfflineEnabled:              license.OfflineEnabled,
			OfflineTokenLifetimeSeconds: license.OfflineTokenLifetimeSeconds,
			Reason:                      string(decision.Reason),
		}, nil
	}

	if decision.TouchExistingSeat {
		if _, err := tx.Exec(ctx, `
			UPDATE activations
			SET last_validated_at = $1
			WHERE id = $2
		`, now, activeActivationID); err != nil {
			return ActivationResult{}, fmt.Errorf("refresh activation validation timestamp: %w", err)
		}

	}

	if decision.CreateActivation {
		metadataJSON, err := metadataToJSON(metadata)
		if err != nil {
			return ActivationResult{}, err
		}

		if _, err := tx.Exec(ctx, `
			INSERT INTO activations (license_id, fingerprint, metadata, created_at, last_validated_at)
			VALUES ($1, $2, $3, $4, $4)
		`, license.ID, fingerprint, metadataJSON, now); err != nil {
			if !isUniqueViolation(err) {
				return ActivationResult{}, fmt.Errorf("insert activation: %w", err)
			}

			activeActivationID, err = findActiveActivationID(ctx, tx, license.ID, fingerprint)
			if err != nil {
				return ActivationResult{}, err
			}
			if activeActivationID == 0 {
				return ActivationResult{}, fmt.Errorf("activation insert conflict without recoverable activation row")
			}

			if _, err := tx.Exec(ctx, `
				UPDATE activations
				SET last_validated_at = $1
				WHERE id = $2
			`, now, activeActivationID); err != nil {
				return ActivationResult{}, fmt.Errorf("refresh activation validation timestamp after conflict: %w", err)
			}
		}
	}

	if decision.StatusChanged {
		query := `
			UPDATE licenses
			SET status = $1`
		args := []any{string(aggregate.Status()), license.ID}

		if decision.ActivationStateBecame {
			query += `, activated_at = COALESCE(activated_at, $2) WHERE id = $3`
			args = []any{string(aggregate.Status()), now, license.ID}
		} else {
			query += ` WHERE id = $2`
		}

		if _, err := tx.Exec(ctx, query, args...); err != nil {
			return ActivationResult{}, fmt.Errorf("set license active: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return ActivationResult{}, fmt.Errorf("commit activate tx: %w", err)
	}

	return ActivationResult{
		Valid:                       true,
		Status:                      string(aggregate.Status()),
		LicenseID:                   license.ID,
		LicenseKey:                  license.Key,
		Slug:                        license.SlugName,
		Fingerprint:                 fingerprint,
		ExpiresAt:                   license.ExpiresAt,
		OfflineEnabled:              license.OfflineEnabled,
		OfflineTokenLifetimeSeconds: license.OfflineTokenLifetimeSeconds,
	}, nil
}

func (s *Store) ValidateLicense(ctx context.Context, licenseKey, fingerprint string) (ValidationResult, error) {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return ValidationResult{}, fmt.Errorf("begin validate tx: %w", err)
	}
	defer tx.Rollback(ctx)

	license, err := loadLicenseByKey(ctx, tx, licenseKey, true)
	if err != nil {
		return ValidationResult{}, err
	}

	aggregate, err := licensedomain.Rehydrate(licensedomain.RehydrateParams{
		Status:         license.Status,
		ExpiresAt:      license.ExpiresAt,
		MaxActivations: license.MaxActivations,
	})
	if err != nil {
		return ValidationResult{}, fmt.Errorf("rehydrate license aggregate: %w", err)
	}

	now := time.Now().UTC()
	activeActivationID, err := findActiveActivationID(ctx, tx, license.ID, fingerprint)
	if err != nil {
		return ValidationResult{}, err
	}

	decision := aggregate.Validate(now, activeActivationID > 0)
	if !decision.Valid {
		return ValidationResult{
			Valid:                       decision.Valid,
			Status:                      string(decision.Status),
			LicenseID:                   license.ID,
			Slug:                        license.SlugName,
			ExpiresAt:                   license.ExpiresAt,
			OfflineEnabled:              license.OfflineEnabled,
			OfflineTokenLifetimeSeconds: license.OfflineTokenLifetimeSeconds,
			Reason:                      string(decision.Reason),
		}, nil
	}

	if decision.TouchSeatValidation {
		if _, err := tx.Exec(ctx, `
			UPDATE activations
			SET last_validated_at = $1
			WHERE id = $2
		`, now, activeActivationID); err != nil {
			return ValidationResult{}, fmt.Errorf("touch activation validation timestamp: %w", err)
		}
	}

	if decision.StatusChanged {
		query := `
			UPDATE licenses
			SET status = $1`
		args := []any{string(aggregate.Status()), license.ID}

		if decision.ActivationStateBecame {
			query += `, activated_at = COALESCE(activated_at, $2) WHERE id = $3`
			args = []any{string(aggregate.Status()), now, license.ID}
		} else {
			query += ` WHERE id = $2`
		}

		if _, err := tx.Exec(ctx, query, args...); err != nil {
			return ValidationResult{}, fmt.Errorf("set license active during validate: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return ValidationResult{}, fmt.Errorf("commit validate tx: %w", err)
	}

	return ValidationResult{
		Valid:                       decision.Valid,
		Status:                      string(aggregate.Status()),
		LicenseID:                   license.ID,
		Slug:                        license.SlugName,
		ExpiresAt:                   license.ExpiresAt,
		OfflineEnabled:              license.OfflineEnabled,
		OfflineTokenLifetimeSeconds: license.OfflineTokenLifetimeSeconds,
	}, nil
}

func (s *Store) DeactivateLicense(ctx context.Context, licenseKey, fingerprint, reason string) (DeactivationResult, error) {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return DeactivationResult{}, fmt.Errorf("begin deactivate tx: %w", err)
	}
	defer tx.Rollback(ctx)

	license, err := loadLicenseByKey(ctx, tx, licenseKey, true)
	if err != nil {
		return DeactivationResult{}, err
	}

	aggregate, err := licensedomain.Rehydrate(licensedomain.RehydrateParams{
		Status:         license.Status,
		ExpiresAt:      license.ExpiresAt,
		MaxActivations: license.MaxActivations,
	})
	if err != nil {
		return DeactivationResult{}, fmt.Errorf("rehydrate license aggregate: %w", err)
	}

	now := time.Now().UTC()
	activationID, err := findActiveActivationID(ctx, tx, license.ID, fingerprint)
	if err != nil {
		return DeactivationResult{}, err
	}

	released := false
	if activationID > 0 {
		released = true
		if _, err := tx.Exec(ctx, `
			UPDATE activations
			SET deactivated_at = $1,
			    deactivation_reason = NULLIF($2, ''),
			    last_validated_at = COALESCE(last_validated_at, $1)
			WHERE id = $3
		`, now, strings.TrimSpace(reason), activationID); err != nil {
			return DeactivationResult{}, fmt.Errorf("deactivate activation: %w", err)
		}
	}

	activeSeats, err := countActiveSeats(ctx, tx, license.ID)
	if err != nil {
		return DeactivationResult{}, err
	}

	decision := aggregate.Deactivate(activeSeats)
	if decision.StatusChanged {
		if _, err := tx.Exec(ctx, `
			UPDATE licenses
			SET status = $1
			WHERE id = $2
		`, string(aggregate.Status()), license.ID); err != nil {
			return DeactivationResult{}, fmt.Errorf("update license status during deactivate: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return DeactivationResult{}, fmt.Errorf("commit deactivate tx: %w", err)
	}

	return DeactivationResult{
		Valid:          true,
		Released:       released,
		Status:         string(aggregate.Status()),
		ActiveSeats:    activeSeats,
		MaxActivations: license.MaxActivations,
		ExpiresAt:      license.ExpiresAt,
	}, nil
}

func (s *Store) IsAuthorizedServerAPIKey(ctx context.Context, candidate string) (bool, error) {
	trimmed := strings.TrimSpace(candidate)
	if trimmed == "" {
		return false, nil
	}

	var exists int
	err := s.db.QueryRow(ctx, `
		SELECT 1
		FROM api_keys
		WHERE key_type = 'server'
		  AND revoked_at IS NULL
		  AND key_hash = $1
		LIMIT 1
	`, hashAPIKey(trimmed)).Scan(&exists)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("lookup server api key: %w", err)
	}

	return true, nil
}

func (s *Store) CreateAPIKey(ctx context.Context, params CreateAPIKeyParams) (CreatedAPIKey, error) {
	keyType := strings.TrimSpace(params.KeyType)
	if keyType == "" {
		keyType = "server"
	}
	if keyType != "server" {
		return CreatedAPIKey{}, fmt.Errorf("unsupported api key type %q", keyType)
	}

	name := strings.TrimSpace(params.Name)
	if name == "" {
		name = "unnamed"
	}

	for i := 0; i < 8; i++ {
		apiKey, err := generateAPIKeyValue()
		if err != nil {
			return CreatedAPIKey{}, err
		}

		var (
			record    APIKeyRecord
			revokedAt sql.NullTime
		)

		err = s.db.QueryRow(ctx, `
			INSERT INTO api_keys (name, key_type, key_hint, key_hash)
			VALUES ($1, $2, $3, $4)
			RETURNING id, name, key_type, key_hint, created_at, revoked_at
		`, name, keyType, keyHint(apiKey), hashAPIKey(apiKey)).Scan(
			&record.ID,
			&record.Name,
			&record.KeyType,
			&record.KeyHint,
			&record.CreatedAt,
			&revokedAt,
		)
		if err != nil {
			if isUniqueViolation(err) {
				continue
			}
			return CreatedAPIKey{}, fmt.Errorf("insert api key: %w", err)
		}

		if revokedAt.Valid {
			v := revokedAt.Time.UTC()
			record.RevokedAt = &v
		}
		record.CreatedAt = record.CreatedAt.UTC()

		return CreatedAPIKey{APIKey: apiKey, Record: record}, nil
	}

	return CreatedAPIKey{}, fmt.Errorf("failed to create unique api key")
}

func (s *Store) ListAPIKeys(ctx context.Context) ([]APIKeyRecord, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, name, key_type, key_hint, created_at, revoked_at
		FROM api_keys
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("list api keys: %w", err)
	}
	defer rows.Close()

	keys := make([]APIKeyRecord, 0)
	for rows.Next() {
		var (
			record    APIKeyRecord
			revokedAt sql.NullTime
		)

		if err := rows.Scan(
			&record.ID,
			&record.Name,
			&record.KeyType,
			&record.KeyHint,
			&record.CreatedAt,
			&revokedAt,
		); err != nil {
			return nil, fmt.Errorf("scan api key row: %w", err)
		}

		record.CreatedAt = record.CreatedAt.UTC()
		if revokedAt.Valid {
			v := revokedAt.Time.UTC()
			record.RevokedAt = &v
		}

		keys = append(keys, record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate api keys: %w", err)
	}

	return keys, nil
}

func (s *Store) RevokeAPIKey(ctx context.Context, id int64) (APIKeyRecord, error) {
	var (
		record    APIKeyRecord
		revokedAt sql.NullTime
	)

	err := s.db.QueryRow(ctx, `
		UPDATE api_keys
		SET revoked_at = COALESCE(revoked_at, NOW())
		WHERE id = $1
		RETURNING id, name, key_type, key_hint, created_at, revoked_at
	`, id).Scan(
		&record.ID,
		&record.Name,
		&record.KeyType,
		&record.KeyHint,
		&record.CreatedAt,
		&revokedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return APIKeyRecord{}, ErrNotFound
		}
		return APIKeyRecord{}, fmt.Errorf("revoke api key: %w", err)
	}

	record.CreatedAt = record.CreatedAt.UTC()
	if revokedAt.Valid {
		v := revokedAt.Time.UTC()
		record.RevokedAt = &v
	}

	return record, nil
}

func (s *Store) CreateWebhookEndpoint(ctx context.Context, params CreateWebhookEndpointParams) (WebhookEndpoint, error) {
	eventsJSON, err := json.Marshal(params.Events)
	if err != nil {
		return WebhookEndpoint{}, fmt.Errorf("marshal webhook events: %w", err)
	}

	var (
		endpoint  WebhookEndpoint
		eventsRaw []byte
	)

	err = s.db.QueryRow(ctx, `
		INSERT INTO webhook_endpoints (name, url, events, enabled)
		VALUES ($1, $2, $3, $4)
		RETURNING id, name, url, events, enabled, created_at, updated_at
	`, params.Name, params.URL, eventsJSON, params.Enabled).Scan(
		&endpoint.ID,
		&endpoint.Name,
		&endpoint.URL,
		&eventsRaw,
		&endpoint.Enabled,
		&endpoint.CreatedAt,
		&endpoint.UpdatedAt,
	)
	if err != nil {
		return WebhookEndpoint{}, fmt.Errorf("insert webhook endpoint: %w", err)
	}

	if err := json.Unmarshal(eventsRaw, &endpoint.Events); err != nil {
		return WebhookEndpoint{}, fmt.Errorf("decode webhook events: %w", err)
	}

	endpoint.CreatedAt = endpoint.CreatedAt.UTC()
	endpoint.UpdatedAt = endpoint.UpdatedAt.UTC()
	return endpoint, nil
}

func (s *Store) ListWebhookEndpoints(ctx context.Context) ([]WebhookEndpoint, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, name, url, events, enabled, created_at, updated_at
		FROM webhook_endpoints
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("list webhook endpoints: %w", err)
	}
	defer rows.Close()

	endpoints := make([]WebhookEndpoint, 0)
	for rows.Next() {
		var (
			endpoint  WebhookEndpoint
			eventsRaw []byte
		)

		if err := rows.Scan(
			&endpoint.ID,
			&endpoint.Name,
			&endpoint.URL,
			&eventsRaw,
			&endpoint.Enabled,
			&endpoint.CreatedAt,
			&endpoint.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan webhook endpoint row: %w", err)
		}

		if err := json.Unmarshal(eventsRaw, &endpoint.Events); err != nil {
			return nil, fmt.Errorf("decode webhook endpoint events: %w", err)
		}

		endpoint.CreatedAt = endpoint.CreatedAt.UTC()
		endpoint.UpdatedAt = endpoint.UpdatedAt.UTC()
		endpoints = append(endpoints, endpoint)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate webhook endpoints: %w", err)
	}

	return endpoints, nil
}

func (s *Store) UpdateWebhookEndpoint(ctx context.Context, id int64, params UpdateWebhookEndpointParams) (WebhookEndpoint, error) {
	current, err := s.getWebhookEndpointByID(ctx, id)
	if err != nil {
		return WebhookEndpoint{}, err
	}

	if params.Name != nil {
		current.Name = *params.Name
	}
	if params.URL != nil {
		current.URL = *params.URL
	}
	if params.Events != nil {
		current.Events = append([]string(nil), (*params.Events)...)
	}
	if params.Enabled != nil {
		current.Enabled = *params.Enabled
	}

	eventsJSON, err := json.Marshal(current.Events)
	if err != nil {
		return WebhookEndpoint{}, fmt.Errorf("marshal webhook events: %w", err)
	}

	err = s.db.QueryRow(ctx, `
		UPDATE webhook_endpoints
		SET name = $1,
		    url = $2,
		    events = $3,
		    enabled = $4,
		    updated_at = NOW()
		WHERE id = $5
		RETURNING updated_at
	`, current.Name, current.URL, eventsJSON, current.Enabled, id).Scan(&current.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return WebhookEndpoint{}, ErrNotFound
		}
		return WebhookEndpoint{}, fmt.Errorf("update webhook endpoint: %w", err)
	}

	current.UpdatedAt = current.UpdatedAt.UTC()
	return current, nil
}

func (s *Store) DeleteWebhookEndpoint(ctx context.Context, id int64) error {
	result, err := s.db.Exec(ctx, `
		DELETE FROM webhook_endpoints
		WHERE id = $1
	`, id)
	if err != nil {
		return fmt.Errorf("delete webhook endpoint: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

func (s *Store) EnqueueWebhookEvent(ctx context.Context, eventType string, payload map[string]any) error {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal webhook payload: %w", err)
	}

	if _, err := s.db.Exec(ctx, `
		INSERT INTO webhook_deliveries (endpoint_id, event_type, payload, status, next_attempt_at)
		SELECT id, $1, $2, 'pending', NOW()
		FROM webhook_endpoints
		WHERE enabled = TRUE
		  AND events ? $1
	`, strings.TrimSpace(eventType), payloadJSON); err != nil {
		return fmt.Errorf("enqueue webhook deliveries: %w", err)
	}

	return nil
}

func (s *Store) ClaimWebhookDeliveries(ctx context.Context, limit int) ([]WebhookDelivery, error) {
	if limit <= 0 {
		limit = 10
	}

	rows, err := s.db.Query(ctx, `
		WITH due AS (
			SELECT d.id
			FROM webhook_deliveries d
			WHERE d.status = 'pending'
			  AND d.next_attempt_at <= NOW()
			ORDER BY d.next_attempt_at ASC, d.id ASC
			LIMIT $1
			FOR UPDATE SKIP LOCKED
		)
		UPDATE webhook_deliveries d
		SET status = 'sending',
		    attempts = d.attempts + 1,
		    updated_at = NOW()
		FROM due
		WHERE d.id = due.id
		RETURNING d.id,
		          d.event_type,
		          d.payload,
		          d.attempts,
		          d.created_at,
		          (SELECT w.url FROM webhook_endpoints w WHERE w.id = d.endpoint_id)
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("claim webhook deliveries: %w", err)
	}
	defer rows.Close()

	deliveries := make([]WebhookDelivery, 0)
	for rows.Next() {
		var (
			delivery   WebhookDelivery
			payloadRaw []byte
		)

		if err := rows.Scan(
			&delivery.ID,
			&delivery.EventType,
			&payloadRaw,
			&delivery.Attempts,
			&delivery.CreatedAt,
			&delivery.EndpointURL,
		); err != nil {
			return nil, fmt.Errorf("scan claimed webhook delivery: %w", err)
		}

		if err := json.Unmarshal(payloadRaw, &delivery.Payload); err != nil {
			return nil, fmt.Errorf("decode claimed webhook payload: %w", err)
		}

		delivery.CreatedAt = delivery.CreatedAt.UTC()
		deliveries = append(deliveries, delivery)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate claimed webhook deliveries: %w", err)
	}

	return deliveries, nil
}

func (s *Store) MarkWebhookDeliveryDelivered(ctx context.Context, deliveryID int64, statusCode int) error {
	result, err := s.db.Exec(ctx, `
		UPDATE webhook_deliveries
		SET status = 'delivered',
		    delivered_at = NOW(),
		    last_error = NULL,
		    last_response_status = $2,
		    updated_at = NOW()
		WHERE id = $1
	`, deliveryID, statusCode)
	if err != nil {
		return fmt.Errorf("mark webhook delivery delivered: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

func (s *Store) MarkWebhookDeliveryFailed(ctx context.Context, deliveryID int64, nextAttemptAt time.Time, statusCode int, lastError string, permanent bool) error {
	status := "pending"
	if permanent {
		status = "failed"
	}

	result, err := s.db.Exec(ctx, `
		UPDATE webhook_deliveries
		SET status = $2,
		    next_attempt_at = CASE WHEN $2 = 'pending' THEN $3 ELSE next_attempt_at END,
		    last_error = NULLIF($4, ''),
		    last_response_status = NULLIF($5, 0),
		    updated_at = NOW()
		WHERE id = $1
	`, deliveryID, status, nextAttemptAt.UTC(), strings.TrimSpace(lastError), statusCode)
	if err != nil {
		return fmt.Errorf("mark webhook delivery failed: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

func (s *Store) ListWebhookDeliveries(ctx context.Context, limit int) ([]WebhookDeliveryLog, error) {
	if limit <= 0 {
		limit = 25
	}

	rows, err := s.db.Query(ctx, `
		SELECT d.id,
		       d.endpoint_id,
		       w.name,
		       w.url,
		       d.event_type,
		       d.status,
		       d.attempts,
		       d.last_response_status,
		       d.last_error,
		       d.next_attempt_at,
		       d.created_at,
		       d.updated_at,
		       d.delivered_at
		FROM webhook_deliveries d
		JOIN webhook_endpoints w ON w.id = d.endpoint_id
		ORDER BY d.updated_at DESC, d.id DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("list webhook deliveries: %w", err)
	}
	defer rows.Close()

	deliveries := make([]WebhookDeliveryLog, 0)
	for rows.Next() {
		var (
			delivery           WebhookDeliveryLog
			lastResponseStatus sql.NullInt32
			lastError          sql.NullString
			deliveredAt        sql.NullTime
		)

		if err := rows.Scan(
			&delivery.ID,
			&delivery.EndpointID,
			&delivery.EndpointName,
			&delivery.EndpointURL,
			&delivery.EventType,
			&delivery.Status,
			&delivery.Attempts,
			&lastResponseStatus,
			&lastError,
			&delivery.NextAttemptAt,
			&delivery.CreatedAt,
			&delivery.UpdatedAt,
			&deliveredAt,
		); err != nil {
			return nil, fmt.Errorf("scan webhook delivery log: %w", err)
		}

		if lastResponseStatus.Valid {
			v := int(lastResponseStatus.Int32)
			delivery.LastResponseStatus = &v
		}
		if lastError.Valid {
			v := lastError.String
			delivery.LastError = &v
		}
		if deliveredAt.Valid {
			v := deliveredAt.Time.UTC()
			delivery.DeliveredAt = &v
		}

		delivery.NextAttemptAt = delivery.NextAttemptAt.UTC()
		delivery.CreatedAt = delivery.CreatedAt.UTC()
		delivery.UpdatedAt = delivery.UpdatedAt.UTC()
		deliveries = append(deliveries, delivery)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate webhook delivery logs: %w", err)
	}

	return deliveries, nil
}

func (s *Store) getWebhookEndpointByID(ctx context.Context, id int64) (WebhookEndpoint, error) {
	var (
		endpoint  WebhookEndpoint
		eventsRaw []byte
	)

	err := s.db.QueryRow(ctx, `
		SELECT id, name, url, events, enabled, created_at, updated_at
		FROM webhook_endpoints
		WHERE id = $1
	`, id).Scan(
		&endpoint.ID,
		&endpoint.Name,
		&endpoint.URL,
		&eventsRaw,
		&endpoint.Enabled,
		&endpoint.CreatedAt,
		&endpoint.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return WebhookEndpoint{}, ErrNotFound
		}
		return WebhookEndpoint{}, fmt.Errorf("query webhook endpoint: %w", err)
	}

	if err := json.Unmarshal(eventsRaw, &endpoint.Events); err != nil {
		return WebhookEndpoint{}, fmt.Errorf("decode webhook endpoint events: %w", err)
	}

	endpoint.CreatedAt = endpoint.CreatedAt.UTC()
	endpoint.UpdatedAt = endpoint.UpdatedAt.UTC()
	return endpoint, nil
}

func loadLicenseByKey(ctx context.Context, tx pgx.Tx, key string, forUpdate bool) (LicenseRow, error) {
	query := `
		SELECT l.id,
		       l.key,
		       l.status,
		       l.expires_at,
		       l.max_activations,
		       l.created_at,
		       l.activated_at,
		       l.revoked_at,
		       l.metadata,
		       s.name,
		       s.offline_enabled,
		       s.offline_token_lifetime_seconds
		FROM licenses l
		JOIN slugs s ON s.id = l.slug_id
		WHERE l.key = $1
	`
	if forUpdate {
		query += ` FOR UPDATE`
	}

	var (
		metadataBytes []byte
		row           LicenseRow
	)

	err := tx.QueryRow(ctx, query, key).Scan(
		&row.ID,
		&row.Key,
		&row.Status,
		&row.ExpiresAt,
		&row.MaxActivations,
		&row.CreatedAt,
		&row.ActivatedAt,
		&row.RevokedAt,
		&metadataBytes,
		&row.SlugName,
		&row.OfflineEnabled,
		&row.OfflineTokenLifetimeSeconds,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return LicenseRow{}, ErrNotFound
		}
		return LicenseRow{}, fmt.Errorf("query license by key: %w", err)
	}

	if len(metadataBytes) == 0 {
		row.Metadata = map[string]any{}
		return row, nil
	}

	if err := json.Unmarshal(metadataBytes, &row.Metadata); err != nil {
		return LicenseRow{}, fmt.Errorf("decode metadata json: %w", err)
	}

	return row, nil
}

func findActiveActivationID(ctx context.Context, tx pgx.Tx, licenseID, fingerprint string) (int64, error) {
	var activationID int64
	err := tx.QueryRow(ctx, `
		SELECT id
		FROM activations
		WHERE license_id = $1
		  AND fingerprint = $2
		  AND deactivated_at IS NULL
		LIMIT 1
	`, licenseID, fingerprint).Scan(&activationID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, nil
		}
		return 0, fmt.Errorf("query active activation: %w", err)
	}

	return activationID, nil
}

func countActiveSeats(ctx context.Context, tx pgx.Tx, licenseID string) (int, error) {
	var seats int
	err := tx.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM activations
		WHERE license_id = $1
		  AND deactivated_at IS NULL
	`, licenseID).Scan(&seats)
	if err != nil {
		return 0, fmt.Errorf("count active seats: %w", err)
	}

	return seats, nil
}

func resolveExpiration(expirationType string, expirationDays sql.NullInt32, fixedExpiresAt sql.NullTime) (*time.Time, error) {
	now := time.Now().UTC()

	switch expirationType {
	case "forever":
		return nil, nil
	case "duration":
		if !expirationDays.Valid || expirationDays.Int32 <= 0 {
			return nil, fmt.Errorf("slug duration expiration requires positive expiration_days")
		}
		expiresAt := now.Add(time.Duration(expirationDays.Int32) * 24 * time.Hour)
		return &expiresAt, nil
	case "fixed_date":
		if !fixedExpiresAt.Valid {
			return nil, fmt.Errorf("slug fixed_date expiration requires fixed_expires_at")
		}
		expiresAt := fixedExpiresAt.Time.UTC()
		return &expiresAt, nil
	default:
		return nil, fmt.Errorf("unsupported expiration_type %q", expirationType)
	}
}

func metadataToJSON(metadata map[string]any) ([]byte, error) {
	if metadata == nil {
		metadata = map[string]any{}
	}

	b, err := json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("marshal metadata: %w", err)
	}

	return b, nil
}

func copyMetadata(metadata map[string]any) map[string]any {
	if metadata == nil {
		return map[string]any{}
	}

	out := make(map[string]any, len(metadata))
	for k, v := range metadata {
		out[k] = v
	}

	return out
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}

func generateLicenseKey() (string, error) {
	const groupCount = 5
	const charsPerGroup = 6
	const keyBytes = (groupCount * charsPerGroup) / 2

	buf := make([]byte, keyBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("read random bytes: %w", err)
	}

	raw := strings.ToUpper(hex.EncodeToString(buf))
	out := make([]byte, 0, len(raw)+(groupCount-1))
	for i := 0; i < len(raw); i++ {
		if i > 0 && i%charsPerGroup == 0 {
			out = append(out, '-')
		}
		out = append(out, raw[i])
	}

	return string(out), nil
}

func generateAPIKeyValue() (string, error) {
	const randomBytes = 32

	buf := make([]byte, randomBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("read random bytes for api key: %w", err)
	}

	encoded := strings.ToLower(hex.EncodeToString(buf))
	return encoded, nil
}

func hashAPIKey(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func keyHint(raw string) string {
	if len(raw) <= 4 {
		return raw
	}

	return raw[:4]
}
