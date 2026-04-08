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
    pub value: Option<(f64, String)>,       // Instant Vector: [timestamp, "value"]
    pub values: Option<Vec<(f64, String)>>, // Range Vector: [[timestamp, "value"], ...]
}

// --- Unified Output ---

#[derive(Debug, Serialize, Deserialize, PartialEq)]
pub struct SummaryResult {
    pub total_raw_lines: usize,
    pub summarized_count: usize,
    pub entries: Vec<String>,
}

// --- Implementation Logic ---

pub fn process_loki_response(response: LokiResponse) -> SummaryResult {
    let mut info_counts: HashMap<String, usize> = HashMap::new();
    let mut warn_counts: HashMap<String, usize> = HashMap::new();
    let mut error_counts: HashMap<String, usize> = HashMap::new();
    let mut total_lines = 0;

    for stream in response.data.result {
        let level = stream
            .stream
            .get("level")
            .or_else(|| stream.stream.get("detected_level"))
            .or_else(|| stream.stream.get("severity_text"))
            .map(|l| l.to_lowercase())
            .unwrap_or_else(|| "info".to_string());

        for entry in stream.values {
            total_lines += 1;
            if entry.len() < 2 {
                continue;
            }
            let msg = &entry[1];

            match level.as_str() {
                "error" | "err" | "fatal" | "panic" => {
                    *error_counts.entry(msg.clone()).or_insert(0) += 1;
                }
                "warn" | "warning" => {
                    *warn_counts.entry(msg.clone()).or_insert(0) += 1;
                }
                _ => {
                    *info_counts.entry(msg.clone()).or_insert(0) += 1;
                }
            }
        }
    }

    let mut final_entries = Vec::new();
    let mut err_msgs: Vec<_> = error_counts.into_iter().collect();
    err_msgs.sort_by(|a, b| b.1.cmp(&a.1).then_with(|| a.0.cmp(&b.0)));
    for (msg, count) in err_msgs {
        final_entries.push(format!("[ERROR] {}{}", msg, if count > 1 { format!(" (x{})", count) } else { "".to_string() }));
    }

    let mut warn_msgs: Vec<_> = warn_counts.into_iter().collect();
    warn_msgs.sort_by(|a, b| b.1.cmp(&a.1).then_with(|| a.0.cmp(&b.0)));
    for (msg, count) in warn_msgs {
        final_entries.push(format!("[WARN] {}{}", msg, if count > 1 { format!(" (x{})", count) } else { "".to_string() }));
    }

    let mut info_msgs: Vec<_> = info_counts.into_iter().collect();
    info_msgs.sort_by(|a, b| b.1.cmp(&a.1).then_with(|| a.0.cmp(&b.0)));
    for (msg, count) in info_msgs {
        final_entries.push(format!("[INFO] {}{}", msg, if count > 1 { format!(" (x{})", count) } else { "".to_string() }));
    }

    SummaryResult {
        total_raw_lines: total_lines,
        summarized_count: final_entries.len(),
        entries: final_entries,
    }
}

pub fn process_metrics_response(response: MetricResponse) -> SummaryResult {
    let mut final_entries = Vec::new();
    let series_count = response.data.result.len();

    for series in response.data.result {
        let name = series.metric.get("__name__").cloned().unwrap_or_else(|| "unknown".to_string());
        
        // Format labels for context: {job="proxy", instance="..."}
        let labels: Vec<String> = series.metric.iter()
            .filter(|(k, _)| k.as_str() != "__name__")
            .map(|(k, v)| format!("{}=\"{}\"", k, v))
            .collect();
        let label_str = if labels.is_empty() { "".to_string() } else { format!("{{{}}}", labels.join(", ")) };

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
                    let mut floats: Vec<f64> = values.iter()
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

                        let trend_symbol = if trend > 0.001 { "↗" } else if trend < -0.001 { "↘" } else { "→" };

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
        let values = messages.into_iter().map(|m| vec!["123".to_string(), m.to_string()]).collect();
        LokiResponse {
            status: "success".to_string(),
            data: LokiData { result_type: "streams".to_string(), result: vec![LokiStream { stream, values }] },
        }
    }

    fn create_mock_metric_response(result_type: &str, series_count: usize) -> MetricResponse {
        let mut result = Vec::new();
        for i in 0..series_count {
            result.push(MetricResult {
                metric: HashMap::from([("__name__".to_string(), format!("metric_{}", i))]),
                value: if result_type == "vector" { Some((1.0, "1".to_string())) } else { None },
                values: if result_type == "matrix" { Some(vec![(1.0, "1".to_string())]) } else { None },
            });
        }
        MetricResponse {
            status: "success".to_string(),
            data: MetricData { result_type: result_type.to_string(), result },
        }
    }

    #[test]
    fn test_deduplication_info() {
        let resp = create_mock_loki_response("info", vec!["pulse", "pulse", "unique"]);
        let result = process_loki_response(resp);
        assert_eq!(result.total_raw_lines, 3);
        assert_eq!(result.summarized_count, 2);
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
        let values: Vec<(f64, String)> = (1..=10)
            .map(|i| (i as f64, (i * 10).to_string()))
            .collect();

        let resp = MetricResponse {
            status: "success".to_string(),
            data: MetricData {
                result_type: "matrix".to_string(),
                result: vec![MetricResult { metric, value: None, values: Some(values) }],
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
                result: vec![MetricResult { metric, value: None, values: Some(values) }],
            },
        };

        let result = process_metrics_response(resp);
        assert!(result.entries[0].contains("↘ (-20.00)"));
    }
}
