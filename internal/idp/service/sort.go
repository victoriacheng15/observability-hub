package service

import (
	"sort"
	"strings"

	corev1 "k8s.io/api/core/v1"
)

func sortSummaries(summaries []Summary) {
	sort.Slice(summaries, func(i, j int) bool {
		if summaries[i].Namespace == summaries[j].Namespace {
			if summaries[i].Kind == summaries[j].Kind {
				return summaries[i].Name < summaries[j].Name
			}
			return summaries[i].Kind < summaries[j].Kind
		}
		return summaries[i].Namespace < summaries[j].Namespace
	})
}

func sortDetails(details []Details) {
	sort.Slice(details, func(i, j int) bool {
		if details[i].Namespace == details[j].Namespace {
			if details[i].Kind == details[j].Kind {
				return details[i].Name < details[j].Name
			}
			return details[i].Kind < details[j].Kind
		}
		return details[i].Namespace < details[j].Namespace
	})
}

func sortHealth(health []Health) {
	sort.Slice(health, func(i, j int) bool {
		if health[i].Namespace == health[j].Namespace {
			if health[i].Kind == health[j].Kind {
				return health[i].Name < health[j].Name
			}
			return health[i].Kind < health[j].Kind
		}
		return health[i].Namespace < health[j].Namespace
	})
}

func sortMetrics(metrics []Metric) {
	sort.Slice(metrics, func(i, j int) bool {
		if metrics[i].Namespace == metrics[j].Namespace {
			if metrics[i].Name == metrics[j].Name {
				return metrics[i].Signal < metrics[j].Signal
			}
			return metrics[i].Name < metrics[j].Name
		}
		return metrics[i].Namespace < metrics[j].Namespace
	})
}

func sortTraces(traces []Trace) {
	sort.Slice(traces, func(i, j int) bool {
		return traces[i].StartTime > traces[j].StartTime
	})
}

func sortOwnership(ownership []Ownership) {
	sort.Slice(ownership, func(i, j int) bool {
		if ownership[i].Namespace == ownership[j].Namespace {
			if ownership[i].Kind == ownership[j].Kind {
				return ownership[i].Name < ownership[j].Name
			}
			return ownership[i].Kind < ownership[j].Kind
		}
		return ownership[i].Namespace < ownership[j].Namespace
	})
}

func SortedMapEntries(values map[string]string) []string {
	entries := make([]string, 0, len(values))
	for key, value := range values {
		entries = append(entries, key+"="+value)
	}
	sort.Strings(entries)
	return entries
}

func labelsSelector(values map[string]string) string {
	parts := make([]string, 0, len(values))
	for key, value := range values {
		parts = append(parts, key+"="+value)
	}
	sort.Strings(parts)
	return strings.Join(parts, ",")
}

func selectorCovers(workloadSelector map[string]string, serviceSelector map[string]string) bool {
	if len(serviceSelector) == 0 {
		return false
	}
	for key, value := range serviceSelector {
		if workloadSelector[key] != value {
			return false
		}
	}
	return true
}

func podReady(pod corev1.Pod) bool {
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodReady {
			return condition.Status == corev1.ConditionTrue
		}
	}
	return false
}

func sortEvents(events []Event) {
	sort.Slice(events, func(i, j int) bool {
		return events[i].Time.After(events[j].Time)
	})
}

func sortPods(pods []corev1.Pod) {
	sort.Slice(pods, func(i, j int) bool {
		return pods[i].Name < pods[j].Name
	})
}
