package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type SigningKeyRecord struct {
	ID                  int64
	Name                string
	Kid                 string
	Algorithm           string
	Status              string
	PrivateKeyEncrypted string
	PublicKeyPEM        string
	CreatedAt           time.Time
	ActivatedAt         *time.Time
	RetiredAt           *time.Time
}

type CreateSigningKeyParams struct {
	Name                string
	Kid                 string
	Algorithm           string
	PrivateKeyEncrypted string
	PublicKeyPEM        string
}

func (s *Store) ListSigningKeys(ctx context.Context) ([]SigningKeyRecord, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, name, kid, algorithm, status, private_key_encrypted, public_key_pem, created_at, activated_at, retired_at
		FROM signing_keys
		ORDER BY created_at DESC, id DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("list signing keys: %w", err)
	}
	defer rows.Close()

	keys := make([]SigningKeyRecord, 0)
	for rows.Next() {
		key, err := scanSigningKey(rows)
		if err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate signing keys: %w", err)
	}

	return keys, nil
}

func (s *Store) ListPublicSigningKeys(ctx context.Context) ([]SigningKeyRecord, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, name, kid, algorithm, status, private_key_encrypted, public_key_pem, created_at, activated_at, retired_at
		FROM signing_keys
		WHERE status IN ('active', 'verify_only')
		ORDER BY status ASC, activated_at DESC NULLS LAST, created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("list public signing keys: %w", err)
	}
	defer rows.Close()

	keys := make([]SigningKeyRecord, 0)
	for rows.Next() {
		key, err := scanSigningKey(rows)
		if err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate public signing keys: %w", err)
	}

	return keys, nil
}

func (s *Store) GetActiveSigningKey(ctx context.Context) (SigningKeyRecord, error) {
	row := s.db.QueryRow(ctx, `
		SELECT id, name, kid, algorithm, status, private_key_encrypted, public_key_pem, created_at, activated_at, retired_at
		FROM signing_keys
		WHERE status = 'active'
		LIMIT 1
	`)

	key, err := scanSigningKey(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return SigningKeyRecord{}, ErrNotFound
		}
		return SigningKeyRecord{}, err
	}

	return key, nil
}

func (s *Store) CreateSigningKey(ctx context.Context, params CreateSigningKeyParams) (SigningKeyRecord, error) {
	name := strings.TrimSpace(params.Name)
	if name == "" {
		name = "unnamed"
	}

	algorithm := strings.TrimSpace(params.Algorithm)
	if algorithm == "" {
		algorithm = "Ed25519"
	}
	if algorithm != "Ed25519" {
		return SigningKeyRecord{}, fmt.Errorf("unsupported signing key algorithm %q", algorithm)
	}

	row := s.db.QueryRow(ctx, `
		INSERT INTO signing_keys (name, kid, algorithm, status, private_key_encrypted, public_key_pem)
		VALUES ($1, $2, $3, 'verify_only', $4, $5)
		RETURNING id, name, kid, algorithm, status, private_key_encrypted, public_key_pem, created_at, activated_at, retired_at
	`, name, strings.TrimSpace(params.Kid), algorithm, params.PrivateKeyEncrypted, params.PublicKeyPEM)

	key, err := scanSigningKey(row)
	if err != nil {
		if isUniqueViolation(err) {
			return SigningKeyRecord{}, ErrConflict
		}
		return SigningKeyRecord{}, fmt.Errorf("create signing key: %w", err)
	}

	return key, nil
}

func (s *Store) ActivateSigningKey(ctx context.Context, id int64) (SigningKeyRecord, error) {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return SigningKeyRecord{}, fmt.Errorf("begin activate signing key tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var status string
	if err := tx.QueryRow(ctx, `SELECT status FROM signing_keys WHERE id = $1 FOR UPDATE`, id).Scan(&status); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return SigningKeyRecord{}, ErrNotFound
		}
		return SigningKeyRecord{}, fmt.Errorf("query signing key before activate: %w", err)
	}
	if status == "retired" {
		return SigningKeyRecord{}, ErrConflict
	}

	if _, err := tx.Exec(ctx, `
		UPDATE signing_keys
		SET status = 'verify_only'
		WHERE status = 'active'
	`); err != nil {
		return SigningKeyRecord{}, fmt.Errorf("demote current active signing key: %w", err)
	}

	row := tx.QueryRow(ctx, `
		UPDATE signing_keys
		SET status = 'active', activated_at = COALESCE(activated_at, NOW())
		WHERE id = $1
		RETURNING id, name, kid, algorithm, status, private_key_encrypted, public_key_pem, created_at, activated_at, retired_at
	`, id)

	key, err := scanSigningKey(row)
	if err != nil {
		return SigningKeyRecord{}, fmt.Errorf("activate signing key: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return SigningKeyRecord{}, fmt.Errorf("commit activate signing key tx: %w", err)
	}

	return key, nil
}

func (s *Store) RetireSigningKey(ctx context.Context, id int64) (SigningKeyRecord, error) {
	row := s.db.QueryRow(ctx, `
		UPDATE signing_keys
		SET status = 'retired', retired_at = COALESCE(retired_at, NOW())
		WHERE id = $1
		RETURNING id, name, kid, algorithm, status, private_key_encrypted, public_key_pem, created_at, activated_at, retired_at
	`, id)

	key, err := scanSigningKey(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return SigningKeyRecord{}, ErrNotFound
		}
		return SigningKeyRecord{}, fmt.Errorf("retire signing key: %w", err)
	}

	return key, nil
}

type signingKeyScanner interface {
	Scan(dest ...any) error
}

func scanSigningKey(row signingKeyScanner) (SigningKeyRecord, error) {
	var (
		key         SigningKeyRecord
		activatedAt sql.NullTime
		retiredAt   sql.NullTime
	)

	if err := row.Scan(
		&key.ID,
		&key.Name,
		&key.Kid,
		&key.Algorithm,
		&key.Status,
		&key.PrivateKeyEncrypted,
		&key.PublicKeyPEM,
		&key.CreatedAt,
		&activatedAt,
		&retiredAt,
	); err != nil {
		return SigningKeyRecord{}, err
	}

	key.CreatedAt = key.CreatedAt.UTC()
	if activatedAt.Valid {
		v := activatedAt.Time.UTC()
		key.ActivatedAt = &v
	}
	if retiredAt.Valid {
		v := retiredAt.Time.UTC()
		key.RetiredAt = &v
	}

	return key, nil
}
