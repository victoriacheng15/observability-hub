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
    let series_count = response.data.result.len();
    SummaryResult {
        total_raw_lines: series_count,
        summarized_count: series_count,
        entries: vec![format!("Received {} metric series (type: {})", series_count, response.data.result_type)],
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
        assert_eq!(result.entries[0], "Received 2 metric series (type: vector)");
    }

    #[test]
    fn test_process_metrics_matrix() {
        let resp = create_mock_metric_response("matrix", 1);
        let result = process_metrics_response(resp);
        assert_eq!(result.total_raw_lines, 1);
        assert_eq!(result.entries[0], "Received 1 metric series (type: matrix)");
    }
}
