package slug

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

const MaxNameLength = 128

var slugNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

type Name string

type ExpirationType string

const (
	ExpirationTypeForever   ExpirationType = "forever"
	ExpirationTypeDuration  ExpirationType = "duration"
	ExpirationTypeFixedDate ExpirationType = "fixed_date"
)

type Policy struct {
	maxActivations int
	expirationType ExpirationType
	expirationDays *int
	fixedExpiresAt *time.Time
}

func ParseName(raw string) (Name, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("slug name is required")
	}

	if len(trimmed) > MaxNameLength {
		return "", fmt.Errorf("slug name exceeds max length of %d", MaxNameLength)
	}

	if !slugNamePattern.MatchString(trimmed) {
		return "", fmt.Errorf("slug must be lowercase URL-safe (a-z, 0-9, hyphen)")
	}

	return Name(trimmed), nil
}

func (n Name) String() string {
	return string(n)
}

func NewPolicy(maxActivations int, expirationType string, expirationDays *int, fixedExpiresAt *time.Time) (Policy, error) {
	if maxActivations <= 0 {
		return Policy{}, fmt.Errorf("max activations must be greater than 0")
	}

	typeValue, err := parseExpirationType(expirationType)
	if err != nil {
		return Policy{}, err
	}

	policy := Policy{
		maxActivations: maxActivations,
		expirationType: typeValue,
	}

	switch typeValue {
	case ExpirationTypeForever:
		return policy, nil
	case ExpirationTypeDuration:
		if expirationDays == nil || *expirationDays <= 0 {
			return Policy{}, fmt.Errorf("expiration_days must be provided and greater than 0 for duration expiration")
		}
		v := *expirationDays
		policy.expirationDays = &v
		return policy, nil
	case ExpirationTypeFixedDate:
		if fixedExpiresAt == nil {
			return Policy{}, fmt.Errorf("fixed_expires_at must be provided for fixed_date expiration")
		}
		v := fixedExpiresAt.UTC()
		policy.fixedExpiresAt = &v
		return policy, nil
	default:
		return Policy{}, fmt.Errorf("expiration_type must be one of forever, duration, fixed_date")
	}
}

func (p Policy) MaxActivations() int {
	return p.maxActivations
}

func (p Policy) ExpirationType() string {
	return string(p.expirationType)
}

func (p Policy) ExpirationDays() *int {
	if p.expirationDays == nil {
		return nil
	}
	v := *p.expirationDays
	return &v
}

func (p Policy) FixedExpiresAt() *time.Time {
	if p.fixedExpiresAt == nil {
		return nil
	}
	v := p.fixedExpiresAt.UTC()
	return &v
}

func parseExpirationType(raw string) (ExpirationType, error) {
	v := ExpirationType(strings.TrimSpace(raw))
	switch v {
	case ExpirationTypeForever, ExpirationTypeDuration, ExpirationTypeFixedDate:
		return v, nil
	default:
		return "", fmt.Errorf("expiration_type must be one of forever, duration, fixed_date")
	}
}
