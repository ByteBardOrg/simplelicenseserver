package storage

import (
	"context"
	"crypto/rand"
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	licensedomain "simple-license-server/internal/domain/license"
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

type RevokeResult struct {
	Valid      bool
	Status     string
	LicenseKey string
	RevokedAt  time.Time
}

type ActivationResult struct {
	Valid       bool
	Status      string
	LicenseKey  string
	Fingerprint string
	ExpiresAt   *time.Time
	Reason      string
}

type ValidationResult struct {
	Valid     bool
	Status    string
	ExpiresAt *time.Time
	Reason    string
}

type DeactivationResult struct {
	Valid          bool
	Released       bool
	Status         string
	ActiveSeats    int
	MaxActivations int
	ExpiresAt      *time.Time
}

type licenseRow struct {
	ID             string
	Key            string
	Status         string
	ExpiresAt      *time.Time
	CreatedAt      time.Time
	ActivatedAt    *time.Time
	RevokedAt      *time.Time
	Metadata       map[string]any
	SlugName       string
	MaxActivations int
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
	var (
		slugID         int64
		expirationType string
		expirationDays sql.NullInt32
		fixedExpiresAt sql.NullTime
	)

	err := q.QueryRow(ctx, `
		SELECT id, expiration_type, expiration_days, fixed_expires_at
		FROM slugs
		WHERE name = $1
	`, slugName).Scan(&slugID, &expirationType, &expirationDays, &fixedExpiresAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return GeneratedLicense{}, ErrNotFound
		}
		return GeneratedLicense{}, fmt.Errorf("lookup slug: %w", err)
	}

	expiresAt, err := resolveExpiration(expirationType, expirationDays, fixedExpiresAt)
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
			INSERT INTO licenses (key, slug_id, status, metadata, expires_at)
			VALUES ($1, $2, 'inactive', $3, $4)
			RETURNING created_at
		`, licenseKey, slugID, metadataJSON, expiresAt).Scan(&createdAt)
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
			Valid:       false,
			Status:      string(decision.Status),
			LicenseKey:  license.Key,
			Fingerprint: fingerprint,
			ExpiresAt:   license.ExpiresAt,
			Reason:      string(decision.Reason),
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
		Valid:       true,
		Status:      string(aggregate.Status()),
		LicenseKey:  license.Key,
		Fingerprint: fingerprint,
		ExpiresAt:   license.ExpiresAt,
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
			Valid:     decision.Valid,
			Status:    string(decision.Status),
			ExpiresAt: license.ExpiresAt,
			Reason:    string(decision.Reason),
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
		Valid:     decision.Valid,
		Status:    string(aggregate.Status()),
		ExpiresAt: license.ExpiresAt,
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

func loadLicenseByKey(ctx context.Context, tx pgx.Tx, key string, forUpdate bool) (licenseRow, error) {
	query := `
		SELECT l.id,
		       l.key,
		       l.status,
		       l.expires_at,
		       l.created_at,
		       l.activated_at,
		       l.revoked_at,
		       l.metadata,
		       s.name,
		       s.max_activations
		FROM licenses l
		JOIN slugs s ON s.id = l.slug_id
		WHERE l.key = $1
	`
	if forUpdate {
		query += ` FOR UPDATE`
	}

	var (
		metadataBytes []byte
		row           licenseRow
	)

	err := tx.QueryRow(ctx, query, key).Scan(
		&row.ID,
		&row.Key,
		&row.Status,
		&row.ExpiresAt,
		&row.CreatedAt,
		&row.ActivatedAt,
		&row.RevokedAt,
		&metadataBytes,
		&row.SlugName,
		&row.MaxActivations,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return licenseRow{}, ErrNotFound
		}
		return licenseRow{}, fmt.Errorf("query license by key: %w", err)
	}

	if len(metadataBytes) == 0 {
		row.Metadata = map[string]any{}
		return row, nil
	}

	if err := json.Unmarshal(metadataBytes, &row.Metadata); err != nil {
		return licenseRow{}, fmt.Errorf("decode metadata json: %w", err)
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
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	const chars = 16

	buf := make([]byte, chars)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("read random bytes: %w", err)
	}

	out := make([]byte, 0, chars+3)
	for i, b := range buf {
		if i > 0 && i%4 == 0 {
			out = append(out, '-')
		}
		out = append(out, alphabet[int(b)%len(alphabet)])
	}

	return string(out), nil
}
