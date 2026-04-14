use clap::{Parser, ValueEnum};
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::io::{self, Read};

#[derive(Parser)]
#[command(author, version, about, long_about = None)]
struct Args {
    /// Type of telemetry to process
    #[arg(short, long, value_enum, default_value_t = TelemetryType::Logs)]
    r#type: TelemetryType,
}

#[derive(Copy, Clone, PartialEq, Eq, PartialOrd, Ord, ValueEnum)]
enum TelemetryType {
    /// Loki logs (json format)
    Logs,
    /// Prometheus metrics (json format)
    Metrics,
}

// --- Logs (Loki) Structs ---

#[derive(Debug, Deserialize, Serialize)]
pub struct LokiResponse {
    pub status: String,
    pub data: LokiData,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct LokiData {
    #[serde(rename = "resultType")]
    pub result_type: String,
    pub result: Vec<LokiStream>,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct LokiStream {
    pub stream: HashMap<String, String>,
    pub values: Vec<Vec<String>>,
}

// --- Metrics (Prometheus) Structs ---

#[derive(Debug, Deserialize, Serialize)]
pub struct MetricResponse {
    pub status: String,
    pub data: MetricData,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct MetricData {
    #[serde(rename = "resultType")]
    pub result_type: String, // "matrix" or "vector"
    pub result: Vec<MetricResult>,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct MetricResult {
    pub metric: HashMap<String, String>,
    pub value: Option<(f64, String)>, // Instant Vector: [timestamp, "value"]
    pub values: Option<Vec<(f64, String)>>, // Range Vector: [[timestamp, "value"], ...]
}

// --- Unified Output ---

#[derive(Debug, Serialize, Deserialize, PartialEq)]
pub struct SummaryResult {
    pub total_raw_lines: usize,
    pub summarized_count: usize,
    pub entries: Vec<String>,
}

#[derive(Debug, Serialize, Deserialize, PartialEq)]
pub struct LogSummaryResult {
    pub total_raw_lines: usize,
    pub summarized_count: usize,
    pub entries: Vec<LogSummaryEntry>,
}

#[derive(Debug, Serialize, Deserialize, PartialEq)]
pub struct LogSummaryEntry {
    pub level: String,
    pub message: String,
    pub count: usize,
    pub first_timestamp_ns: String,
    pub last_timestamp_ns: String,
}

#[derive(Debug)]
struct LogAggregate {
    count: usize,
    first_timestamp_ns: String,
    last_timestamp_ns: String,
}

// --- Implementation Logic ---

pub fn process_loki_response(response: LokiResponse) -> LogSummaryResult {
    let mut info_entries: HashMap<String, LogAggregate> = HashMap::new();
    let mut warn_entries: HashMap<String, LogAggregate> = HashMap::new();
    let mut error_entries: HashMap<String, LogAggregate> = HashMap::new();
    let mut total_lines = 0;

    for stream in response.data.result {
        let level = stream
            .stream
            .get("level")
            .or_else(|| stream.stream.get("detected_level"))
            .or_else(|| stream.stream.get("severity_text"))
            .map(|l| l.to_lowercase())
            .unwrap_or_else(|| "info".to_string());
        let normalized_level = normalize_log_level(&level);

        for entry in stream.values {
            total_lines += 1;
            if entry.len() < 2 {
                continue;
            }
            let timestamp_ns = &entry[0];
            let message = &entry[1];

            match normalized_level {
                "error" => {
                    record_log_entry(&mut error_entries, message, timestamp_ns);
                }
                "warn" => {
                    record_log_entry(&mut warn_entries, message, timestamp_ns);
                }
                _ => {
                    record_log_entry(&mut info_entries, message, timestamp_ns);
                }
            }
        }
    }

    let mut final_entries = Vec::new();
    append_log_entries(&mut final_entries, "error", error_entries);
    append_log_entries(&mut final_entries, "warn", warn_entries);
    append_log_entries(&mut final_entries, "info", info_entries);

    LogSummaryResult {
        total_raw_lines: total_lines,
        summarized_count: final_entries.len(),
        entries: final_entries,
    }
}

fn normalize_log_level(level: &str) -> &'static str {
    match level {
        "error" | "err" | "fatal" | "panic" => "error",
        "warn" | "warning" => "warn",
        _ => "info",
    }
}

fn record_log_entry(
    entries: &mut HashMap<String, LogAggregate>,
    message: &str,
    timestamp_ns: &str,
) {
    entries
        .entry(message.to_string())
        .and_modify(|entry| {
            entry.count += 1;
            if timestamp_before(timestamp_ns, &entry.first_timestamp_ns) {
                entry.first_timestamp_ns = timestamp_ns.to_string();
            }
            if timestamp_after(timestamp_ns, &entry.last_timestamp_ns) {
                entry.last_timestamp_ns = timestamp_ns.to_string();
            }
        })
        .or_insert_with(|| LogAggregate {
            count: 1,
            first_timestamp_ns: timestamp_ns.to_string(),
            last_timestamp_ns: timestamp_ns.to_string(),
        });
}

fn timestamp_before(left: &str, right: &str) -> bool {
    match (left.parse::<u128>(), right.parse::<u128>()) {
        (Ok(left), Ok(right)) => left < right,
        _ => left < right,
    }
}

fn timestamp_after(left: &str, right: &str) -> bool {
    match (left.parse::<u128>(), right.parse::<u128>()) {
        (Ok(left), Ok(right)) => left > right,
        _ => left > right,
    }
}

fn append_log_entries(
    final_entries: &mut Vec<LogSummaryEntry>,
    level: &str,
    entries: HashMap<String, LogAggregate>,
) {
    let mut sorted_entries: Vec<_> = entries.into_iter().collect();
    sorted_entries.sort_by(|a, b| b.1.count.cmp(&a.1.count).then_with(|| a.0.cmp(&b.0)));

    for (message, entry) in sorted_entries {
        final_entries.push(LogSummaryEntry {
            level: level.to_string(),
            message,
            count: entry.count,
            first_timestamp_ns: entry.first_timestamp_ns,
            last_timestamp_ns: entry.last_timestamp_ns,
        });
    }
}

pub fn process_metrics_response(response: MetricResponse) -> SummaryResult {
    let mut final_entries = Vec::new();
    let series_count = response.data.result.len();

    for series in response.data.result {
        let name = series
            .metric
            .get("__name__")
            .cloned()
            .unwrap_or_else(|| "unknown".to_string());

        // Format labels for context: {job="proxy", instance="..."}
        let labels: Vec<String> = series
            .metric
            .iter()
            .filter(|(k, _)| k.as_str() != "__name__")
            .map(|(k, v)| format!("{}=\"{}\"", k, v))
            .collect();
        let label_str = if labels.is_empty() {
            "".to_string()
        } else {
            format!("{{{}}}", labels.join(", "))
        };

        match response.data.result_type.as_str() {
            "vector" => {
                if let Some((_, val_str)) = series.value {
                    if let Ok(val) = val_str.parse::<f64>() {
                        final_entries.push(format!("{}{} = {:.4}", name, label_str, val));
                    }
                }
            }
            "matrix" => {
                if let Some(values) = series.values {
                    let mut floats: Vec<f64> = values
                        .iter()
                        .filter_map(|(_, v)| v.parse::<f64>().ok())
                        .collect();

                    if !floats.is_empty() {
                        let trend = if floats.len() >= 2 {
                            let first = floats[0];
                            let last = floats[floats.len() - 1];
                            last - first
                        } else {
                            0.0
                        };

                        floats.sort_by(|a, b| a.partial_cmp(b).unwrap());
                        let min = floats[0];
                        let max = floats[floats.len() - 1];
                        let sum: f64 = floats.iter().sum();
                        let avg = sum / floats.len() as f64;

                        // P95 calculation
                        let p95_idx = (floats.len() as f64 * 0.95).floor() as usize;
                        let p95 = floats[p95_idx.min(floats.len() - 1)];

                        let trend_symbol = if trend > 0.001 {
                            "↗"
                        } else if trend < -0.001 {
                            "↘"
                        } else {
                            "→"
                        };

                        final_entries.push(format!(
                            "{}{} | stats: [min:{:.2}, max:{:.2}, avg:{:.2}, p95:{:.2}] trend: {} ({:+.2})",
                            name, label_str, min, max, avg, p95, trend_symbol, trend
                        ));
                    }
                }
            }
            _ => {}
        }
    }

    SummaryResult {
        total_raw_lines: series_count,
        summarized_count: final_entries.len(),
        entries: final_entries,
    }
}

fn main() -> io::Result<()> {
    let args = Args::parse();

    let mut buffer = String::new();
    io::stdin().read_to_string(&mut buffer)?;

    if buffer.trim().is_empty() {
        return Ok(());
    }

    match args.r#type {
        TelemetryType::Logs => {
            let response: LokiResponse = match serde_json::from_str(&buffer) {
                Ok(res) => res,
                Err(e) => {
                    eprintln!("Error parsing Loki JSON: {}", e);
                    return Err(io::Error::new(io::ErrorKind::InvalidData, e));
                }
            };
            let result = process_loki_response(response);
            println!("{}", serde_json::to_string(&result).unwrap());
        }
        TelemetryType::Metrics => {
            let response: MetricResponse = match serde_json::from_str(&buffer) {
                Ok(res) => res,
                Err(e) => {
                    eprintln!("Error parsing Prometheus JSON: {}", e);
                    return Err(io::Error::new(io::ErrorKind::InvalidData, e));
                }
            };
            let result = process_metrics_response(response);
            println!("{}", serde_json::to_string(&result).unwrap());
        }
    }

    Ok(())
}

#[cfg(test)]
mod tests {
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
    fn test_process_metrics_vector() {
        let resp = create_mock_metric_response("vector", 2);
        let result = process_metrics_response(resp);
        assert_eq!(result.total_raw_lines, 2);
        assert!(result.entries[0].contains("metric_0 = 1.0000"));
    }

    #[test]
    fn test_process_metrics_matrix_stats() {
        let mut metric = HashMap::new();
        metric.insert("__name__".to_string(), "test_latency".to_string());
        metric.insert("service".to_string(), "proxy".to_string());

        // Test values: 10, 20, 30, 40, 50, 60, 70, 80, 90, 100
        let values: Vec<(f64, String)> =
            (1..=10).map(|i| (i as f64, (i * 10).to_string())).collect();

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
        assert_eq!(result.summarized_count, 1);
        let entry = &result.entries[0];

        assert!(entry.contains("test_latency"));
        assert!(entry.contains("service=\"proxy\""));
        assert!(entry.contains("min:10.00"));
        assert!(entry.contains("max:100.00"));
        assert!(entry.contains("avg:55.00"));
        assert!(entry.contains("p95:100.00"));
        assert!(entry.contains("↗ (+90.00)"));
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
        assert!(result.entries[0].contains("↘ (-20.00)"));
    }
}
