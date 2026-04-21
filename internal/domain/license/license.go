package license

import (
	"fmt"
	"time"
)

type Status string

const (
	StatusInactive Status = "inactive"
	StatusActive   Status = "active"
	StatusRevoked  Status = "revoked"
)

type Reason string

const (
	ReasonNone                 Reason = ""
	ReasonRevoked              Reason = "revoked"
	ReasonExpired              Reason = "expired"
	ReasonActivationLimit      Reason = "activation_limit_reached"
	ReasonFingerprintNotActive Reason = "fingerprint_not_active"
)

type License struct {
	status         Status
	expiresAt      *time.Time
	maxActivations MaxActivations
}

type MaxActivations struct {
	value int
}

type RehydrateParams struct {
	Status         string
	ExpiresAt      *time.Time
	MaxActivations int
}

type ActivationResult struct {
	Valid                 bool
	Status                Status
	Reason                Reason
	TouchExistingSeat     bool
	CreateActivation      bool
	StatusChanged         bool
	ActivationStateBecame bool
}

type ValidationResult struct {
	Valid                 bool
	Status                Status
	Reason                Reason
	TouchSeatValidation   bool
	StatusChanged         bool
	ActivationStateBecame bool
}

type DeactivationResult struct {
	Status        Status
	StatusChanged bool
}

func Rehydrate(params RehydrateParams) (*License, error) {
	status, err := parseStatus(params.Status)
	if err != nil {
		return nil, err
	}

	maxActivations, err := NewMaxActivations(params.MaxActivations)
	if err != nil {
		return nil, err
	}

	return &License{
		status:         status,
		expiresAt:      cloneTime(params.ExpiresAt),
		maxActivations: maxActivations,
	}, nil
}

func NewMaxActivations(value int) (MaxActivations, error) {
	if value <= 0 {
		return MaxActivations{}, fmt.Errorf("max activations must be greater than 0")
	}

	return MaxActivations{value: value}, nil
}

func (m MaxActivations) Int() int {
	return m.value
}

func (l *License) Status() Status {
	return l.status
}

func (l *License) ExpiresAt() *time.Time {
	return cloneTime(l.expiresAt)
}

func (l *License) MaxActivations() int {
	return l.maxActivations.Int()
}

func (l *License) Activate(now time.Time, activeSeats int, fingerprintAlreadyActive bool) ActivationResult {
	if l.status == StatusRevoked {
		return ActivationResult{Valid: false, Status: l.status, Reason: ReasonRevoked}
	}

	if l.isExpired(now) {
		return ActivationResult{Valid: false, Status: l.status, Reason: ReasonExpired}
	}

	previous := l.status

	if fingerprintAlreadyActive {
		if l.status != StatusActive {
			l.status = StatusActive
		}

		return ActivationResult{
			Valid:                 true,
			Status:                l.status,
			Reason:                ReasonNone,
			TouchExistingSeat:     true,
			StatusChanged:         l.status != previous,
			ActivationStateBecame: l.status == StatusActive,
		}
	}

	if activeSeats >= l.maxActivations.Int() {
		return ActivationResult{Valid: false, Status: l.status, Reason: ReasonActivationLimit}
	}

	if l.status != StatusActive {
		l.status = StatusActive
	}

	return ActivationResult{
		Valid:                 true,
		Status:                l.status,
		Reason:                ReasonNone,
		CreateActivation:      true,
		StatusChanged:         l.status != previous,
		ActivationStateBecame: l.status == StatusActive,
	}
}

func (l *License) Validate(now time.Time, fingerprintActive bool) ValidationResult {
	if l.status == StatusRevoked {
		return ValidationResult{Valid: false, Status: l.status, Reason: ReasonRevoked}
	}

	if l.isExpired(now) {
		return ValidationResult{Valid: false, Status: l.status, Reason: ReasonExpired}
	}

	if !fingerprintActive {
		return ValidationResult{Valid: false, Status: l.status, Reason: ReasonFingerprintNotActive}
	}

	previous := l.status
	if l.status != StatusActive {
		l.status = StatusActive
	}

	return ValidationResult{
		Valid:                 true,
		Status:                l.status,
		Reason:                ReasonNone,
		TouchSeatValidation:   true,
		StatusChanged:         l.status != previous,
		ActivationStateBecame: l.status == StatusActive,
	}
}

func (l *License) Deactivate(activeSeatsAfter int) DeactivationResult {
	previous := l.status

	if l.status != StatusRevoked {
		if activeSeatsAfter <= 0 {
			l.status = StatusInactive
		} else {
			l.status = StatusActive
		}
	}

	return DeactivationResult{
		Status:        l.status,
		StatusChanged: l.status != previous,
	}
}

func (l *License) isExpired(now time.Time) bool {
	if l.expiresAt == nil {
		return false
	}

	return !now.UTC().Before(l.expiresAt.UTC())
}

func parseStatus(status string) (Status, error) {
	s := Status(status)
	switch s {
	case StatusInactive, StatusActive, StatusRevoked:
		return s, nil
	default:
		return "", fmt.Errorf("invalid license status %q", status)
	}
}

func cloneTime(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}

	v := t.UTC()
	return &v
}
