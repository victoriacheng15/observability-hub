use clap::{Parser, ValueEnum};
use serde::{Deserialize, Serialize};
use std::collections::{HashMap, hash_map::Entry};
use std::io::{self, Read};

#[cfg(test)]
mod tests;

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
    #[serde(skip_serializing_if = "Option::is_none")]
    pub context: Option<HashMap<String, String>>,
}

#[derive(Debug)]
struct LogAggregate {
    count: usize,
    first_timestamp_ns: String,
    last_timestamp_ns: String,
    context: Option<HashMap<String, String>>,
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
                    let context = extract_log_context(&stream.stream, ContextScope::Error);
                    record_log_entry(&mut error_entries, message, timestamp_ns, context);
                }
                "warn" => {
                    let context = extract_log_context(&stream.stream, ContextScope::Warn);
                    record_log_entry(&mut warn_entries, message, timestamp_ns, context);
                }
                _ => {
                    record_log_entry(&mut info_entries, message, timestamp_ns, None);
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
    context: Option<HashMap<String, String>>,
) {
    match entries.entry(message.to_string()) {
        Entry::Occupied(mut occupied) => {
            let entry = occupied.get_mut();
            entry.count += 1;
            if timestamp_before(timestamp_ns, &entry.first_timestamp_ns) {
                entry.first_timestamp_ns = timestamp_ns.to_string();
            }
            if timestamp_after(timestamp_ns, &entry.last_timestamp_ns) {
                entry.last_timestamp_ns = timestamp_ns.to_string();
            }
            merge_log_context(&mut entry.context, context);
        }
        Entry::Vacant(vacant) => {
            vacant.insert(LogAggregate {
                count: 1,
                first_timestamp_ns: timestamp_ns.to_string(),
                last_timestamp_ns: timestamp_ns.to_string(),
                context,
            });
        }
    }
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
            context: entry.context,
        });
    }
}

#[derive(Copy, Clone)]
enum ContextScope {
    Error,
    Warn,
}

fn extract_log_context(
    stream: &HashMap<String, String>,
    scope: ContextScope,
) -> Option<HashMap<String, String>> {
    let allowed_keys = match scope {
        ContextScope::Error => [
            "service_name",
            "repo",
            "error",
            "status",
            "path",
            "ref",
            "action",
        ]
        .as_slice(),
        ContextScope::Warn => ["service_name", "status", "path", "action", "error"].as_slice(),
    };

    let mut context = HashMap::new();
    for key in allowed_keys {
        if let Some(value) = stream.get(*key).filter(|value| !value.trim().is_empty()) {
            context.insert((*key).to_string(), trim_context_value(value));
        }
    }

    if matches!(scope, ContextScope::Error) {
        if let Some(output) = stream.get("output").and_then(|value| preview_output(value)) {
            context.insert("output_preview".to_string(), output);
        }
    }

    if context.is_empty() {
        None
    } else {
        Some(context)
    }
}

fn trim_context_value(value: &str) -> String {
    const MAX_CONTEXT_VALUE_LEN: usize = 160;
    let trimmed = value.trim();
    if trimmed.chars().count() <= MAX_CONTEXT_VALUE_LEN {
        return trimmed.to_string();
    }

    let mut truncated: String = trimmed.chars().take(MAX_CONTEXT_VALUE_LEN).collect();
    truncated.push_str("...");
    truncated
}

fn preview_output(output: &str) -> Option<String> {
    let trimmed = output.trim();
    if trimmed.is_empty() {
        return None;
    }

    if let Ok(value) = serde_json::from_str::<serde_json::Value>(trimmed) {
        if let Some(message) = value.get("msg").and_then(|msg| msg.as_str()) {
            return Some(trim_context_value(message));
        }
    }

    trimmed.lines().next().map(trim_context_value)
}

fn merge_log_context(
    existing_context: &mut Option<HashMap<String, String>>,
    new_context: Option<HashMap<String, String>>,
) {
    let Some(new_context) = new_context else {
        return;
    };

    let existing_context = existing_context.get_or_insert_with(HashMap::new);
    for (key, value) in new_context {
        existing_context
            .entry(key)
            .and_modify(|existing_value| {
                if existing_value != &value {
                    *existing_value = "<multiple>".to_string();
                }
            })
            .or_insert(value);
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
