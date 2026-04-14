use clap::{Parser, ValueEnum};
use serde::{Deserialize, Serialize};
use std::collections::{BTreeMap, HashMap, hash_map::Entry};
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

// --- Output Structs ---

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

#[derive(Debug, Serialize, Deserialize, PartialEq)]
pub struct MetricSummaryResult {
    pub result_type: String,
    pub total_raw_lines: usize,
    pub summarized_count: usize,
    pub entries: Vec<MetricSummaryEntry>,
}

#[derive(Debug, Serialize, Deserialize, PartialEq)]
pub struct MetricSummaryEntry {
    pub metric: String,
    pub kind: String,
    pub status: String,
    pub labels: BTreeMap<String, String>,
    pub sample_count: usize,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub timestamp: Option<f64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub current: Option<f64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub min: Option<f64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub max: Option<f64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub avg: Option<f64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub p95: Option<f64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub p99: Option<f64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub first: Option<f64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub last: Option<f64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub trend_delta: Option<f64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub delta: Option<f64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub average_rate_per_second: Option<f64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub resets_detected: Option<usize>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub first_timestamp: Option<f64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub last_timestamp: Option<f64>,
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

pub fn process_metrics_response(response: MetricResponse) -> MetricSummaryResult {
    let mut final_entries = Vec::new();
    let series_count = response.data.result.len();
    let result_type = response.data.result_type.clone();

    for series in response.data.result {
        let name = series
            .metric
            .get("__name__")
            .cloned()
            .unwrap_or_else(|| "unknown".to_string());
        let labels = metric_labels(&series.metric);

        match result_type.as_str() {
            "vector" => {
                if let Some((timestamp, val_str)) = series.value {
                    if let Ok(current) = val_str.parse::<f64>() {
                        let kind = metric_kind(&name);
                        final_entries.push(MetricSummaryEntry {
                            metric: name,
                            kind: kind.to_string(),
                            status: "normal".to_string(),
                            labels,
                            sample_count: 1,
                            timestamp: Some(timestamp),
                            current: Some(current),
                            min: None,
                            max: None,
                            avg: None,
                            p95: None,
                            p99: None,
                            first: None,
                            last: None,
                            trend_delta: None,
                            delta: None,
                            average_rate_per_second: None,
                            resets_detected: None,
                            first_timestamp: None,
                            last_timestamp: None,
                        });
                    }
                }
            }
            "matrix" => {
                if let Some(values) = series.values {
                    let samples: Vec<(f64, f64)> = values
                        .iter()
                        .filter_map(|(timestamp, value)| {
                            value.parse::<f64>().ok().map(|value| (*timestamp, value))
                        })
                        .collect();

                    if !samples.is_empty() {
                        let first_timestamp = samples[0].0;
                        let first = samples[0].1;
                        let last_timestamp = samples[samples.len() - 1].0;
                        let last = samples[samples.len() - 1].1;
                        let kind = metric_kind(&name);

                        if kind == "counter" {
                            let (delta, resets_detected) = counter_delta_and_resets(&samples);
                            let elapsed_seconds = last_timestamp - first_timestamp;
                            let average_rate_per_second = if elapsed_seconds > 0.0 {
                                Some(delta / elapsed_seconds)
                            } else {
                                None
                            };

                            final_entries.push(MetricSummaryEntry {
                                metric: name,
                                kind: kind.to_string(),
                                status: "normal".to_string(),
                                labels,
                                sample_count: samples.len(),
                                timestamp: None,
                                current: None,
                                min: None,
                                max: None,
                                avg: None,
                                p95: None,
                                p99: None,
                                first: Some(first),
                                last: Some(last),
                                trend_delta: None,
                                delta: Some(delta),
                                average_rate_per_second,
                                resets_detected: Some(resets_detected),
                                first_timestamp: Some(first_timestamp),
                                last_timestamp: Some(last_timestamp),
                            });
                            continue;
                        }

                        let trend_delta = last - first;

                        let mut floats: Vec<f64> =
                            samples.iter().map(|(_, value)| *value).collect();
                        floats.sort_by(|a, b| a.partial_cmp(b).unwrap());
                        let min = floats[0];
                        let max = floats[floats.len() - 1];
                        let sum: f64 = floats.iter().sum();
                        let avg = sum / floats.len() as f64;

                        // P95 calculation
                        let p95_idx = (floats.len() as f64 * 0.95).floor() as usize;
                        let p95 = floats[p95_idx.min(floats.len() - 1)];
                        let p99_idx = (floats.len() as f64 * 0.99).floor() as usize;
                        let p99 = floats[p99_idx.min(floats.len() - 1)];

                        final_entries.push(MetricSummaryEntry {
                            metric: name,
                            kind: "gauge".to_string(),
                            status: "normal".to_string(),
                            labels,
                            sample_count: samples.len(),
                            timestamp: None,
                            current: None,
                            min: Some(min),
                            max: Some(max),
                            avg: Some(avg),
                            p95: Some(p95),
                            p99: Some(p99),
                            first: Some(first),
                            last: Some(last),
                            trend_delta: Some(trend_delta),
                            delta: None,
                            average_rate_per_second: None,
                            resets_detected: None,
                            first_timestamp: Some(first_timestamp),
                            last_timestamp: Some(last_timestamp),
                        });
                    }
                }
            }
            _ => {}
        }
    }

    MetricSummaryResult {
        result_type,
        total_raw_lines: series_count,
        summarized_count: final_entries.len(),
        entries: final_entries,
    }
}

fn metric_labels(metric: &HashMap<String, String>) -> BTreeMap<String, String> {
    metric
        .iter()
        .filter(|(key, _)| key.as_str() != "__name__")
        .map(|(key, value)| (key.clone(), value.clone()))
        .collect()
}

fn metric_kind(name: &str) -> &'static str {
    if name.ends_with("_total") || name.ends_with("_count") || name.ends_with("_sum") {
        "counter"
    } else {
        "gauge"
    }
}

fn counter_delta_and_resets(samples: &[(f64, f64)]) -> (f64, usize) {
    if samples.len() < 2 {
        return (0.0, 0);
    }

    let mut delta = 0.0;
    let mut resets = 0;

    for window in samples.windows(2) {
        let previous = window[0].1;
        let current = window[1].1;
        if current >= previous {
            delta += current - previous;
        } else {
            resets += 1;
            delta += current;
        }
    }

    (delta, resets)
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
