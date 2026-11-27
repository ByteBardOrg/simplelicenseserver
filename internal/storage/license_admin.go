package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	licensedomain "simple-license-server/internal/domain/license"
)

// AdminRemovedReason is stored on activations an operator removed from the management API.
const AdminRemovedReason = "removed_by_admin"

type ActivationRow struct {
	ID              int64
	Fingerprint     string
	Metadata        map[string]any
	CreatedAt       time.Time
	LastValidatedAt *time.Time
}

type LicenseDetail struct {
	License     LicenseRow
	Activations []ActivationRow
}

type UpdateLicenseParams struct {
	MaxActivations *int
	// SetExpiresAt distinguishes "leave expiry alone" from "set it to ExpiresAt" (nil means never expires).
	SetExpiresAt bool
	ExpiresAt    *time.Time
	Revoked      *bool
}

type UpdatedLicense struct {
	Detail LicenseDetail
	// NewlyRevoked is true when this update moved the license into the revoked state.
	NewlyRevoked bool
}

type RemovedActivation struct {
	Fingerprint string
	Detail      LicenseDetail
}

type queryer interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

func (s *Store) GetLicense(ctx context.Context, licenseKey string) (LicenseDetail, error) {
	return getLicenseDetail(ctx, s.db, licenseKey)
}

func (s *Store) UpdateLicense(ctx context.Context, licenseKey string, params UpdateLicenseParams) (UpdatedLicense, error) {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return UpdatedLicense{}, fmt.Errorf("begin update license tx: %w", err)
	}
	defer tx.Rollback(ctx)

	license, err := loadLicenseByKey(ctx, tx, licenseKey, true)
	if err != nil {
		return UpdatedLicense{}, err
	}

	if params.MaxActivations != nil {
		if _, err := licensedomain.NewMaxActivations(*params.MaxActivations); err != nil {
			return UpdatedLicense{}, err
		}
		if _, err := tx.Exec(ctx, `UPDATE licenses SET max_activations = $1 WHERE id = $2`, *params.MaxActivations, license.ID); err != nil {
			return UpdatedLicense{}, fmt.Errorf("update license max activations: %w", err)
		}
	}

	if params.SetExpiresAt {
		if _, err := tx.Exec(ctx, `UPDATE licenses SET expires_at = $1 WHERE id = $2`, params.ExpiresAt, license.ID); err != nil {
			return UpdatedLicense{}, fmt.Errorf("update license expiry: %w", err)
		}
	}

	newlyRevoked := false
	if params.Revoked != nil {
		wasRevoked := license.Status == string(licensedomain.StatusRevoked)
		switch {
		case *params.Revoked && !wasRevoked:
			if _, err := tx.Exec(ctx, `
				UPDATE licenses
				SET status = 'revoked', revoked_at = COALESCE(revoked_at, NOW())
				WHERE id = $1
			`, license.ID); err != nil {
				return UpdatedLicense{}, fmt.Errorf("revoke license: %w", err)
			}
			newlyRevoked = true
		case !*params.Revoked && wasRevoked:
			seats, err := countActiveSeats(ctx, tx, license.ID)
			if err != nil {
				return UpdatedLicense{}, err
			}
			if _, err := tx.Exec(ctx, `
				UPDATE licenses
				SET status = $1, revoked_at = NULL
				WHERE id = $2
			`, string(licensedomain.StatusForSeats(seats)), license.ID); err != nil {
				return UpdatedLicense{}, fmt.Errorf("restore license: %w", err)
			}
		}
	}

	detail, err := getLicenseDetail(ctx, tx, licenseKey)
	if err != nil {
		return UpdatedLicense{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return UpdatedLicense{}, fmt.Errorf("commit update license tx: %w", err)
	}

	return UpdatedLicense{Detail: detail, NewlyRevoked: newlyRevoked}, nil
}

func (s *Store) RemoveActivation(ctx context.Context, licenseKey string, activationID int64) (RemovedActivation, error) {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return RemovedActivation{}, fmt.Errorf("begin remove activation tx: %w", err)
	}
	defer tx.Rollback(ctx)

	license, err := loadLicenseByKey(ctx, tx, licenseKey, true)
	if err != nil {
		return RemovedActivation{}, err
	}

	var fingerprint string
	err = tx.QueryRow(ctx, `
		UPDATE activations
		SET deactivated_at = NOW(), deactivation_reason = $3
		WHERE id = $1
		  AND license_id = $2
		  AND deactivated_at IS NULL
		RETURNING fingerprint
	`, activationID, license.ID, AdminRemovedReason).Scan(&fingerprint)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return RemovedActivation{}, ErrNotFound
		}
		return RemovedActivation{}, fmt.Errorf("remove activation: %w", err)
	}

	if license.Status != string(licensedomain.StatusRevoked) {
		seats, err := countActiveSeats(ctx, tx, license.ID)
		if err != nil {
			return RemovedActivation{}, err
		}
		if _, err := tx.Exec(ctx, `UPDATE licenses SET status = $1 WHERE id = $2`, string(licensedomain.StatusForSeats(seats)), license.ID); err != nil {
			return RemovedActivation{}, fmt.Errorf("update license status after removing activation: %w", err)
		}
	}

	detail, err := getLicenseDetail(ctx, tx, licenseKey)
	if err != nil {
		return RemovedActivation{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return RemovedActivation{}, fmt.Errorf("commit remove activation tx: %w", err)
	}

	return RemovedActivation{Fingerprint: fingerprint, Detail: detail}, nil
}

func getLicenseDetail(ctx context.Context, q queryer, licenseKey string) (LicenseDetail, error) {
	license, err := scanLicenseListRow(q.QueryRow(ctx, licenseRowSelect+` WHERE l.key = $1`, licenseKey))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return LicenseDetail{}, ErrNotFound
		}
		return LicenseDetail{}, err
	}

	rows, err := q.Query(ctx, `
		SELECT id, fingerprint, metadata, created_at, last_validated_at
		FROM activations
		WHERE license_id = $1
		  AND deactivated_at IS NULL
		ORDER BY created_at ASC, id ASC
	`, license.ID)
	if err != nil {
		return LicenseDetail{}, fmt.Errorf("list activations: %w", err)
	}
	defer rows.Close()

	activations := make([]ActivationRow, 0)
	for rows.Next() {
		var (
			activation      ActivationRow
			metadataJSON    []byte
			lastValidatedAt sql.NullTime
		)
		if err := rows.Scan(&activation.ID, &activation.Fingerprint, &metadataJSON, &activation.CreatedAt, &lastValidatedAt); err != nil {
			return LicenseDetail{}, fmt.Errorf("scan activation: %w", err)
		}
		activation.CreatedAt = activation.CreatedAt.UTC()
		if lastValidatedAt.Valid {
			v := lastValidatedAt.Time.UTC()
			activation.LastValidatedAt = &v
		}
		activation.Metadata = map[string]any{}
		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &activation.Metadata); err != nil {
				return LicenseDetail{}, fmt.Errorf("decode activation metadata: %w", err)
			}
		}
		activations = append(activations, activation)
	}
	if err := rows.Err(); err != nil {
		return LicenseDetail{}, fmt.Errorf("iterate activations: %w", err)
	}

	return LicenseDetail{License: license, Activations: activations}, nil
}
