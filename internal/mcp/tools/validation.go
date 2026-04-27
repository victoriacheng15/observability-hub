package tools

import (
	"fmt"
	"strings"
	"unicode"

	k8svalidation "k8s.io/apimachinery/pkg/util/validation"
)

// OptionalDNS1123Label validates optional Kubernetes label-style identifiers
// such as namespaces and container names.
func OptionalDNS1123Label(field, value string) error {
	if value == "" {
		return nil
	}
	return DNS1123Label(field, value)
}

// DNS1123Label validates required Kubernetes label-style identifiers.
func DNS1123Label(field, value string) error {
	if value == "" {
		return fmt.Errorf("%s is required", field)
	}
	if errs := k8svalidation.IsDNS1123Label(value); len(errs) > 0 {
		return fmt.Errorf("%s is invalid: %s", field, strings.Join(errs, "; "))
	}
	return nil
}

// DNS1123Subdomain validates required Kubernetes subdomain-style identifiers
// such as pod names.
func DNS1123Subdomain(field, value string) error {
	if value == "" {
		return fmt.Errorf("%s is required", field)
	}
	if errs := k8svalidation.IsDNS1123Subdomain(value); len(errs) > 0 {
		return fmt.Errorf("%s is invalid: %s", field, strings.Join(errs, "; "))
	}
	return nil
}

// OptionalPodRef validates optional pod references accepted by Hubble:
// either pod-name or namespace/pod-name.
func OptionalPodRef(field, value string) error {
	if value == "" {
		return nil
	}
	parts := strings.Split(value, "/")
	switch len(parts) {
	case 1:
		return DNS1123Subdomain(field, parts[0])
	case 2:
		if err := DNS1123Label(field+" namespace", parts[0]); err != nil {
			return err
		}
		return DNS1123Subdomain(field+" pod", parts[1])
	default:
		return fmt.Errorf("%s must be pod-name or namespace/pod-name", field)
	}
}

// OptionalSafeToken validates a short CLI filter token. It intentionally allows
// only characters commonly used in protocol, verdict, method, and entity names.
func OptionalSafeToken(field, value string, maxLen int) error {
	if value == "" {
		return nil
	}
	if len(value) > maxLen {
		return fmt.Errorf("%s must be %d characters or fewer", field, maxLen)
	}
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' || r == '+' {
			continue
		}
		return fmt.Errorf("%s contains unsupported character %q", field, r)
	}
	return nil
}

// OptionalTextFilter rejects control characters and caps unstructured filter
// strings that are passed as a single process argument.
func OptionalTextFilter(field, value string, maxLen int) error {
	if value == "" {
		return nil
	}
	if len(value) > maxLen {
		return fmt.Errorf("%s must be %d characters or fewer", field, maxLen)
	}
	for _, r := range value {
		if unicode.IsControl(r) {
			return fmt.Errorf("%s contains control character %q", field, r)
		}
	}
	return nil
}

// OptionalPort validates optional TCP/UDP port filters.
func OptionalPort(field string, value int) error {
	if value == 0 {
		return nil
	}
	if value < 1 || value > 65535 {
		return fmt.Errorf("%s must be between 1 and 65535", field)
	}
	return nil
}
