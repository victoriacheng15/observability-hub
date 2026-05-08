package service

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (s *KubernetesService) Events(ctx context.Context, opts EventsOptions) ([]Event, error) {
	details, err := s.Describe(ctx, DescribeOptions{Name: opts.Name, Namespace: opts.Namespace})
	if err != nil {
		return nil, err
	}

	events := make([]Event, 0)
	for _, detail := range details {
		pods, err := s.podsFor(ctx, detail)
		if err != nil {
			return nil, err
		}
		podNames := make(map[string]struct{}, len(pods))
		for _, pod := range pods {
			podNames[pod.Name] = struct{}{}
		}

		kubeEvents, err := s.clientset.CoreV1().Events(detail.Namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, fmt.Errorf("list events: %w", err)
		}
		for _, event := range kubeEvents.Items {
			if !eventMatches(detail, podNames, event) || eventOlderThan(event, opts.Since, s.now()) {
				continue
			}
			events = append(events, Event{
				Name:      detail.Name,
				Namespace: detail.Namespace,
				Kind:      detail.Kind,
				Type:      event.Type,
				Reason:    event.Reason,
				Message:   event.Message,
				Object:    event.InvolvedObject.Kind + "/" + event.InvolvedObject.Name,
				Age:       s.age(eventTime(event)),
				Time:      eventTime(event),
			})
		}
	}

	sortEvents(events)
	if opts.Tail > 0 && len(events) > opts.Tail {
		events = events[:opts.Tail]
	}
	return events, nil
}

func eventMatches(detail Details, podNames map[string]struct{}, event corev1.Event) bool {
	if event.InvolvedObject.Namespace != "" && event.InvolvedObject.Namespace != detail.Namespace {
		return false
	}
	if event.InvolvedObject.Name == detail.Name {
		return true
	}
	_, ok := podNames[event.InvolvedObject.Name]
	return ok
}

func eventOlderThan(event corev1.Event, since time.Duration, now time.Time) bool {
	if since <= 0 {
		return false
	}
	return now.Sub(eventTime(event)) > since
}

func eventTime(event corev1.Event) time.Time {
	if !event.LastTimestamp.IsZero() {
		return event.LastTimestamp.Time
	}
	if !event.EventTime.IsZero() {
		return event.EventTime.Time
	}
	if !event.FirstTimestamp.IsZero() {
		return event.FirstTimestamp.Time
	}
	return event.CreationTimestamp.Time
}
