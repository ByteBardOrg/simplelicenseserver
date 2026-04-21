package slug

import (
	"testing"
	"time"
)

func TestParseNameValid(t *testing.T) {
	name, err := ParseName("pro-monthly")
	if err != nil {
		t.Fatalf("parse name: %v", err)
	}

	if name.String() != "pro-monthly" {
		t.Fatalf("expected pro-monthly, got %q", name.String())
	}
}

func TestParseNameRejectsInvalid(t *testing.T) {
	if _, err := ParseName("Pro Monthly"); err == nil {
		t.Fatalf("expected invalid slug name error")
	}

	if _, err := ParseName(""); err == nil {
		t.Fatalf("expected missing slug name error")
	}
}

func TestNewPolicyNormalizesAndValidates(t *testing.T) {
	forever, err := NewPolicy(1, "forever", intPtr(30), timePtr(time.Now().UTC().Add(24*time.Hour)))
	if err != nil {
		t.Fatalf("new forever policy: %v", err)
	}

	if forever.ExpirationDays() != nil || forever.FixedExpiresAt() != nil {
		t.Fatalf("expected forever policy to clear optional expiration fields")
	}

	duration, err := NewPolicy(2, "duration", intPtr(30), nil)
	if err != nil {
		t.Fatalf("new duration policy: %v", err)
	}

	if duration.ExpirationDays() == nil || *duration.ExpirationDays() != 30 {
		t.Fatalf("expected duration policy expiration_days=30")
	}

	fixed := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	fixedPolicy, err := NewPolicy(3, "fixed_date", nil, &fixed)
	if err != nil {
		t.Fatalf("new fixed policy: %v", err)
	}

	if fixedPolicy.FixedExpiresAt() == nil || !fixedPolicy.FixedExpiresAt().Equal(fixed) {
		t.Fatalf("expected fixed_expires_at to round-trip")
	}
}

func TestNewPolicyRejectsInvalid(t *testing.T) {
	if _, err := NewPolicy(0, "forever", nil, nil); err == nil {
		t.Fatalf("expected max activations error")
	}

	if _, err := NewPolicy(1, "duration", nil, nil); err == nil {
		t.Fatalf("expected duration validation error")
	}

	if _, err := NewPolicy(1, "fixed_date", nil, nil); err == nil {
		t.Fatalf("expected fixed date validation error")
	}
}

func intPtr(v int) *int {
	return &v
}

func timePtr(v time.Time) *time.Time {
	return &v
}
