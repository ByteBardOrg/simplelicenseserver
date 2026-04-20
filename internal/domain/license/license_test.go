package license

import (
	"testing"
	"time"
)

func TestActivateRevokedDenied(t *testing.T) {
	lic := mustRehydrate(t, RehydrateParams{Status: string(StatusRevoked), MaxActivations: 3})

	result := lic.Activate(time.Now().UTC(), 0, false)
	if result.Valid {
		t.Fatalf("expected revoked activation to be denied")
	}

	if result.Reason != ReasonRevoked {
		t.Fatalf("expected reason %q, got %q", ReasonRevoked, result.Reason)
	}
}

func TestActivateExpiredDenied(t *testing.T) {
	expiresAt := time.Now().UTC().Add(-time.Minute)
	lic := mustRehydrate(t, RehydrateParams{Status: string(StatusInactive), MaxActivations: 3, ExpiresAt: &expiresAt})

	result := lic.Activate(time.Now().UTC(), 0, false)
	if result.Valid {
		t.Fatalf("expected expired activation to be denied")
	}

	if result.Reason != ReasonExpired {
		t.Fatalf("expected reason %q, got %q", ReasonExpired, result.Reason)
	}
}

func TestActivateExistingFingerprintAtLimitAllowed(t *testing.T) {
	lic := mustRehydrate(t, RehydrateParams{Status: string(StatusActive), MaxActivations: 1})

	result := lic.Activate(time.Now().UTC(), 1, true)
	if !result.Valid {
		t.Fatalf("expected existing active fingerprint to remain valid")
	}

	if !result.TouchExistingSeat {
		t.Fatalf("expected existing seat touch")
	}

	if result.CreateActivation {
		t.Fatalf("did not expect new activation creation")
	}
}

func TestActivateNewFingerprintAtLimitDenied(t *testing.T) {
	lic := mustRehydrate(t, RehydrateParams{Status: string(StatusActive), MaxActivations: 1})

	result := lic.Activate(time.Now().UTC(), 1, false)
	if result.Valid {
		t.Fatalf("expected new fingerprint activation to be denied")
	}

	if result.Reason != ReasonActivationLimit {
		t.Fatalf("expected reason %q, got %q", ReasonActivationLimit, result.Reason)
	}
}

func TestActivateAfterSeatReleasedAllowed(t *testing.T) {
	lic := mustRehydrate(t, RehydrateParams{Status: string(StatusInactive), MaxActivations: 1})

	result := lic.Activate(time.Now().UTC(), 0, false)
	if !result.Valid {
		t.Fatalf("expected activation to succeed after seat release")
	}

	if !result.CreateActivation {
		t.Fatalf("expected activation row creation")
	}

	if lic.Status() != StatusActive {
		t.Fatalf("expected aggregate status active, got %q", lic.Status())
	}
}

func TestValidateFingerprintMissingDenied(t *testing.T) {
	lic := mustRehydrate(t, RehydrateParams{Status: string(StatusActive), MaxActivations: 2})

	result := lic.Validate(time.Now().UTC(), false)
	if result.Valid {
		t.Fatalf("expected validation failure when fingerprint inactive")
	}

	if result.Reason != ReasonFingerprintNotActive {
		t.Fatalf("expected reason %q, got %q", ReasonFingerprintNotActive, result.Reason)
	}
}

func TestDeactivateStatusTransitions(t *testing.T) {
	revoked := mustRehydrate(t, RehydrateParams{Status: string(StatusRevoked), MaxActivations: 2})
	if result := revoked.Deactivate(0); result.Status != StatusRevoked {
		t.Fatalf("expected revoked status to stay revoked, got %q", result.Status)
	}

	active := mustRehydrate(t, RehydrateParams{Status: string(StatusActive), MaxActivations: 2})
	if result := active.Deactivate(0); result.Status != StatusInactive {
		t.Fatalf("expected inactive after releasing last seat, got %q", result.Status)
	}

	inactive := mustRehydrate(t, RehydrateParams{Status: string(StatusInactive), MaxActivations: 2})
	if result := inactive.Deactivate(1); result.Status != StatusActive {
		t.Fatalf("expected active when seats remain, got %q", result.Status)
	}
}

func TestRehydrateValidatesInput(t *testing.T) {
	if _, err := Rehydrate(RehydrateParams{Status: "bad", MaxActivations: 1}); err == nil {
		t.Fatalf("expected invalid status error")
	}

	if _, err := Rehydrate(RehydrateParams{Status: string(StatusActive), MaxActivations: 0}); err == nil {
		t.Fatalf("expected invalid max activations error")
	}
}

func mustRehydrate(t *testing.T, params RehydrateParams) *License {
	t.Helper()
	lic, err := Rehydrate(params)
	if err != nil {
		t.Fatalf("rehydrate license: %v", err)
	}
	return lic
}
