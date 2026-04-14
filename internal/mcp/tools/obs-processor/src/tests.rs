use super::*;

fn create_mock_loki_response(level: &str, messages: Vec<&str>) -> LokiResponse {
    let mut stream = HashMap::new();
    stream.insert("level".to_string(), level.to_string());
    let values = messages
        .into_iter()
        .enumerate()
        .map(|(i, m)| vec![(100 + i).to_string(), m.to_string()])
        .collect();
    LokiResponse {
        status: "success".to_string(),
        data: LokiData {
            result_type: "streams".to_string(),
            result: vec![LokiStream { stream, values }],
        },
    }
}

fn create_mock_metric_response(result_type: &str, series_count: usize) -> MetricResponse {
    let mut result = Vec::new();
    for i in 0..series_count {
        result.push(MetricResult {
            metric: HashMap::from([("__name__".to_string(), format!("metric_{}", i))]),
            value: if result_type == "vector" {
                Some((1.0, "1".to_string()))
            } else {
                None
            },
            values: if result_type == "matrix" {
                Some(vec![(1.0, "1".to_string())])
            } else {
                None
            },
        });
    }
    MetricResponse {
        status: "success".to_string(),
        data: MetricData {
            result_type: result_type.to_string(),
            result,
        },
    }
}

#[test]
fn test_deduplication_info() {
    let resp = create_mock_loki_response("info", vec!["pulse", "pulse", "unique"]);
    let result = process_loki_response(resp);
    assert_eq!(result.total_raw_lines, 3);
    assert_eq!(result.summarized_count, 2);
    assert_eq!(
        result.entries[0],
        LogSummaryEntry {
            level: "info".to_string(),
            message: "pulse".to_string(),
            count: 2,
            first_timestamp_ns: "100".to_string(),
            last_timestamp_ns: "101".to_string(),
            context: None,
        }
    );
    assert_eq!(
        result.entries[1],
        LogSummaryEntry {
            level: "info".to_string(),
            message: "unique".to_string(),
            count: 1,
            first_timestamp_ns: "102".to_string(),
            last_timestamp_ns: "102".to_string(),
            context: None,
        }
    );
}

#[test]
fn test_process_loki_response_orders_errors_warnings_info() {
    let error_resp = create_mock_loki_response("fatal", vec!["boom", "boom"]);
    let warn_resp = create_mock_loki_response("warning", vec!["slow"]);
    let info_resp = create_mock_loki_response("info", vec!["ok"]);

    let mut streams = Vec::new();
    streams.extend(error_resp.data.result);
    streams.extend(warn_resp.data.result);
    streams.extend(info_resp.data.result);

    let result = process_loki_response(LokiResponse {
        status: "success".to_string(),
        data: LokiData {
            result_type: "streams".to_string(),
            result: streams,
        },
    });

    assert_eq!(result.total_raw_lines, 4);
    assert_eq!(result.summarized_count, 3);
    assert_eq!(result.entries[0].level, "error");
    assert_eq!(result.entries[0].message, "boom");
    assert_eq!(result.entries[0].count, 2);
    assert_eq!(result.entries[1].level, "warn");
    assert_eq!(result.entries[1].message, "slow");
    assert_eq!(result.entries[2].level, "info");
    assert_eq!(result.entries[2].message, "ok");
}

#[test]
fn test_process_loki_response_adds_error_context() {
    let mut stream = HashMap::new();
    stream.insert("level".to_string(), "error".to_string());
    stream.insert("service_name".to_string(), "proxy".to_string());
    stream.insert("repo".to_string(), "bioHub".to_string());
    stream.insert("error".to_string(), "exit status 1".to_string());
    stream.insert("status".to_string(), "500".to_string());
    stream.insert("path".to_string(), "/webhook".to_string());
    stream.insert("ref".to_string(), "refs/heads/main".to_string());
    stream.insert("action".to_string(), "sync".to_string());
    stream.insert(
        "output".to_string(),
        r#"{"msg":"Repository is not a valid git repository.","repo":"bioHub"}"#.to_string(),
    );

    let resp = LokiResponse {
        status: "success".to_string(),
        data: LokiData {
            result_type: "streams".to_string(),
            result: vec![LokiStream {
                stream,
                values: vec![
                    vec!["120".to_string(), "webhook_sync_failed".to_string()],
                    vec!["100".to_string(), "webhook_sync_failed".to_string()],
                ],
            }],
        },
    };

    let result = process_loki_response(resp);
    let entry = &result.entries[0];
    let context = entry.context.as_ref().expect("expected error context");

    assert_eq!(entry.level, "error");
    assert_eq!(entry.count, 2);
    assert_eq!(entry.first_timestamp_ns, "100");
    assert_eq!(entry.last_timestamp_ns, "120");
    assert_eq!(context.get("service_name").unwrap(), "proxy");
    assert_eq!(context.get("repo").unwrap(), "bioHub");
    assert_eq!(context.get("error").unwrap(), "exit status 1");
    assert_eq!(
        context.get("output_preview").unwrap(),
        "Repository is not a valid git repository."
    );
}

#[test]
fn test_process_loki_response_keeps_warning_context_small() {
    let mut stream = HashMap::new();
    stream.insert("level".to_string(), "warn".to_string());
    stream.insert("service_name".to_string(), "proxy".to_string());
    stream.insert("repo".to_string(), "hidden-for-warn".to_string());
    stream.insert("status".to_string(), "429".to_string());
    stream.insert("path".to_string(), "/api".to_string());
    stream.insert("output".to_string(), "verbose warning output".to_string());

    let resp = LokiResponse {
        status: "success".to_string(),
        data: LokiData {
            result_type: "streams".to_string(),
            result: vec![LokiStream {
                stream,
                values: vec![vec!["100".to_string(), "rate_limited".to_string()]],
            }],
        },
    };

    let result = process_loki_response(resp);
    let context = result.entries[0]
        .context
        .as_ref()
        .expect("expected warn context");

    assert_eq!(result.entries[0].level, "warn");
    assert_eq!(context.get("service_name").unwrap(), "proxy");
    assert_eq!(context.get("status").unwrap(), "429");
    assert_eq!(context.get("path").unwrap(), "/api");
    assert!(!context.contains_key("repo"));
    assert!(!context.contains_key("output_preview"));
}

#[test]
fn test_process_metrics_vector() {
    let resp = create_mock_metric_response("vector", 2);
    let result = process_metrics_response(resp);
    let entry = &result.entries[0];

    assert_eq!(result.result_type, "vector");
    assert_eq!(result.total_raw_lines, 2);
    assert_eq!(result.summarized_count, 2);
    assert_eq!(entry.metric, "metric_0");
    assert_eq!(entry.kind, "gauge");
    assert_eq!(entry.status, "normal");
    assert_eq!(entry.sample_count, 1);
    assert_eq!(entry.timestamp, Some(1.0));
    assert_eq!(entry.current, Some(1.0));
    assert!(entry.labels.is_empty());
}

#[test]
fn test_process_metrics_matrix_stats() {
    let mut metric = HashMap::new();
    metric.insert("__name__".to_string(), "test_latency".to_string());
    metric.insert("service".to_string(), "proxy".to_string());

    // Test values: 10, 20, 30, 40, 50, 60, 70, 80, 90, 100
    let values: Vec<(f64, String)> = (1..=10).map(|i| (i as f64, (i * 10).to_string())).collect();

    let resp = MetricResponse {
        status: "success".to_string(),
        data: MetricData {
            result_type: "matrix".to_string(),
            result: vec![MetricResult {
                metric,
                value: None,
                values: Some(values),
            }],
        },
    };

    let result = process_metrics_response(resp);
    let entry = &result.entries[0];

    assert_eq!(result.result_type, "matrix");
    assert_eq!(result.total_raw_lines, 1);
    assert_eq!(result.summarized_count, 1);
    assert_eq!(entry.metric, "test_latency");
    assert_eq!(entry.kind, "gauge");
    assert_eq!(entry.status, "normal");
    assert_eq!(entry.labels.get("service").unwrap(), "proxy");
    assert_eq!(entry.sample_count, 10);
    assert_eq!(entry.min, Some(10.0));
    assert_eq!(entry.max, Some(100.0));
    assert_eq!(entry.avg, Some(55.0));
    assert_eq!(entry.p95, Some(100.0));
    assert_eq!(entry.p99, Some(100.0));
    assert_eq!(entry.first, Some(10.0));
    assert_eq!(entry.last, Some(100.0));
    assert_eq!(entry.trend_delta, Some(90.0));
    assert_eq!(entry.first_timestamp, Some(1.0));
    assert_eq!(entry.last_timestamp, Some(10.0));
}

#[test]
fn test_process_metrics_trend_down() {
    let mut metric = HashMap::new();
    metric.insert("__name__".to_string(), "cpu_usage".to_string());

    // Test values: 100, 90, 80
    let values = vec![
        (1.0, "100".to_string()),
        (2.0, "90".to_string()),
        (3.0, "80".to_string()),
    ];

    let resp = MetricResponse {
        status: "success".to_string(),
        data: MetricData {
            result_type: "matrix".to_string(),
            result: vec![MetricResult {
                metric,
                value: None,
                values: Some(values),
            }],
        },
    };

    let result = process_metrics_response(resp);
    assert_eq!(result.entries[0].trend_delta, Some(-20.0));
    assert_eq!(result.entries[0].first_timestamp, Some(1.0));
    assert_eq!(result.entries[0].last_timestamp, Some(3.0));
}
