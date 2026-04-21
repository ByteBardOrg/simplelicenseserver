package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type stubQueryable struct {
	queryRowFn func(ctx context.Context, query string, args ...any) pgx.Row
}

func (s stubQueryable) QueryRow(ctx context.Context, query string, args ...any) pgx.Row {
	return s.queryRowFn(ctx, query, args...)
}

func (s stubQueryable) Exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	var tag pgconn.CommandTag
	return tag, fmt.Errorf("unexpected Exec call")
}

type stubRow struct {
	scanFn func(dest ...any) error
}

func (r stubRow) Scan(dest ...any) error {
	return r.scanFn(dest...)
}

func TestGenerateLicenseWithQuerierSnapshotsDurationSlugPolicy(t *testing.T) {
	createdAt := time.Date(2026, 4, 21, 12, 0, 0, 0, time.UTC)
	metadata := map[string]any{"email": "user@example.com"}

	callCount := 0
	q := stubQueryable{queryRowFn: func(_ context.Context, _ string, args ...any) pgx.Row {
		callCount++
		switch callCount {
		case 1:
			return stubRow{scanFn: func(dest ...any) error {
				*dest[0].(*int64) = 123
				*dest[1].(*string) = "duration"
				*dest[2].(*sql.NullInt32) = sql.NullInt32{Int32: 30, Valid: true}
				*dest[3].(*sql.NullTime) = sql.NullTime{}
				*dest[4].(*int) = 7
				return nil
			}}
		case 2:
			return stubRow{scanFn: func(dest ...any) error {
				if len(args) != 5 {
					t.Fatalf("expected 5 insert args, got %d", len(args))
				}

				key, ok := args[0].(string)
				if !ok {
					t.Fatalf("expected license key string arg, got %T", args[0])
				}

				pattern := regexp.MustCompile(`^[A-F0-9]{6}(?:-[A-F0-9]{6}){4}$`)
				if !pattern.MatchString(key) {
					t.Fatalf("unexpected license key format %q", key)
				}

				slugID, ok := args[1].(int64)
				if !ok || slugID != 123 {
					t.Fatalf("expected slug id 123, got %v (%T)", args[1], args[1])
				}

				metadataJSON, ok := args[2].([]byte)
				if !ok {
					t.Fatalf("expected metadata []byte, got %T", args[2])
				}

				var metadataDecoded map[string]any
				if err := json.Unmarshal(metadataJSON, &metadataDecoded); err != nil {
					t.Fatalf("unmarshal metadata: %v", err)
				}
				if metadataDecoded["email"] != "user@example.com" {
					t.Fatalf("unexpected metadata %v", metadataDecoded)
				}

				expiresAt, ok := args[3].(*time.Time)
				if !ok || expiresAt == nil {
					t.Fatalf("expected expires_at *time.Time, got %T (%v)", args[3], args[3])
				}

				remaining := time.Until(*expiresAt)
				if remaining < 29*24*time.Hour || remaining > 31*24*time.Hour {
					t.Fatalf("expected duration-based expiration around 30 days, got %s", remaining)
				}

				maxActivations, ok := args[4].(int)
				if !ok || maxActivations != 7 {
					t.Fatalf("expected max_activations 7, got %v (%T)", args[4], args[4])
				}

				*dest[0].(*time.Time) = createdAt
				return nil
			}}
		default:
			t.Fatalf("unexpected QueryRow call %d", callCount)
			return stubRow{scanFn: func(dest ...any) error { return nil }}
		}
	}}

	generated, err := generateLicenseWithQuerier(context.Background(), q, "default", metadata)
	if err != nil {
		t.Fatalf("generate license: %v", err)
	}

	if generated.Slug != "default" {
		t.Fatalf("expected slug default, got %q", generated.Slug)
	}

	if generated.Status != "inactive" {
		t.Fatalf("expected inactive status, got %q", generated.Status)
	}

	if generated.ExpiresAt == nil {
		t.Fatalf("expected generated license expiration")
	}

	if !generated.CreatedAt.Equal(createdAt) {
		t.Fatalf("expected created_at %s, got %s", createdAt, generated.CreatedAt)
	}
}

func TestLoadSlugOptionsRejectsNonPositiveMaxActivations(t *testing.T) {
	q := stubQueryable{queryRowFn: func(_ context.Context, _ string, _ ...any) pgx.Row {
		return stubRow{scanFn: func(dest ...any) error {
			*dest[0].(*int64) = 123
			*dest[1].(*string) = "forever"
			*dest[2].(*sql.NullInt32) = sql.NullInt32{}
			*dest[3].(*sql.NullTime) = sql.NullTime{}
			*dest[4].(*int) = 0
			return nil
		}}
	}}

	_, err := loadSlugOptionsByName(context.Background(), q, "default")
	if err == nil {
		t.Fatalf("expected error for non-positive max_activations")
	}

	if err.Error() != "invalid slug policy in storage: max activations must be greater than 0" {
		t.Fatalf("unexpected error: %v", err)
	}
}
