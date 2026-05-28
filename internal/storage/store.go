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

// ActivationResult includes Metadata from the license
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
	Metadata                    map[string]any // Metadata from the license
}

// ValidationResult includes Metadata from the license
type ValidationResult struct {
	Valid                       bool
	Status                      string
	LicenseID                   string
	Slug                        string
	ExpiresAt                   *time.Time
	OfflineEnabled              bool
	OfflineTokenLifetimeSeconds int
	Reason                      string
	Metadata                    map[string]any // Metadata from the license
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

// ActivateLicense does not accept user-provided metadata; it uses the license's metadata
func (s *Store) ActivateLicense(ctx context.Context, licenseKey, fingerprint string, _ map[string]any) (ActivationResult, error) {
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
		Metadata:       license.Metadata, // Pass license's metadata to Rehydrate
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
			Metadata:                    license.Metadata, // Include license's metadata
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
		// Use the license's metadata for the activation
		metadataJSON, err := metadataToJSON(license.Metadata)
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
		Metadata:                    license.Metadata, // Include license's metadata
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
		Metadata:       license.Metadata, // Pass license's metadata to Rehydrate
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
			Metadata:                    license.Metadata, // Include license's metadata
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
		Metadata:                    license.Metadata, // Include license's metadata
	}, nil
}
