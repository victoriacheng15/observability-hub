package service

import (
	"context"
	"sort"
	"strings"
	"time"

	"observability-hub/internal/idp/kube"
)

func (s *KubernetesService) Ownership(ctx context.Context, opts OwnershipOptions) ([]Ownership, error) {
	details, err := s.Describe(ctx, DescribeOptions{Name: opts.Name, Namespace: opts.Namespace})
	if err != nil {
		return nil, err
	}

	ownership := make([]Ownership, 0, len(details))
	for _, detail := range details {
		ownership = append(ownership, ownershipFor(detail))
	}

	sortOwnership(ownership)
	return ownership, nil
}

func ownershipFor(detail Details) Ownership {
	return Ownership{
		Summary:      detail.Summary,
		Owner:        metadataValue(detail, []string{"platform.observability-hub.io/owner", "app.kubernetes.io/owner", "owner", "team"}),
		Tier:         metadataValue(detail, []string{"platform.observability-hub.io/tier", "tier", "app.kubernetes.io/component", "app.kubernetes.io/part-of"}),
		Source:       metadataValue(detail, []string{"platform.observability-hub.io/source", "platform.observability-hub.io/repo", "app.kubernetes.io/source", "repository", "repo", "source"}),
		Docs:         metadataValues(detail, []string{"platform.observability-hub.io/docs", "platform.observability-hub.io/runbook", "platform.observability-hub.io/dashboard", "docs", "documentation", "runbook", "dashboard"}),
		SourceObject: detail.Kind + " " + detail.Namespace + "/" + detail.Name,
	}
}

func metadataValue(detail Details, keys []string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(detail.Labels[key]); value != "" {
			return value
		}
		if value := strings.TrimSpace(detail.Annotations[key]); value != "" {
			return value
		}
	}
	return "unknown"
}

func metadataValues(detail Details, keys []string) []string {
	values := make([]string, 0)
	for _, key := range keys {
		if value := strings.TrimSpace(detail.Labels[key]); value != "" {
			values = append(values, key+"="+value)
		}
		if value := strings.TrimSpace(detail.Annotations[key]); value != "" {
			values = append(values, key+"="+value)
		}
	}
	if len(values) == 0 {
		return []string{"unknown"}
	}
	sort.Strings(values)
	return values
}

func (s *KubernetesService) age(created time.Time) string {
	return kube.Age(s.now(), created)
}
