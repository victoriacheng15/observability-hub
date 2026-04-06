use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::io::{self, Read};

/// LokiResponse represents the raw JSON structure returned by the Loki API.
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
    pub values: Vec<Vec<String>>, // Each entry is [timestamp_ns, message]
}

/// SummaryResult is the optimized output for the AI agent.
#[derive(Debug, Serialize, Deserialize, PartialEq)]
pub struct SummaryResult {
    pub total_raw_lines: usize,
    pub summarized_count: usize,
    pub entries: Vec<String>,
}

/// Core logic to transform raw Loki JSON into summarized entries.
pub fn process_loki_response(response: LokiResponse) -> SummaryResult {
    let mut info_counts: HashMap<String, usize> = HashMap::new();
    let mut warn_counts: HashMap<String, usize> = HashMap::new();
    let mut error_counts: HashMap<String, usize> = HashMap::new();
    let mut total_lines = 0;

    // 1. Process logs
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

    // 2. Construct final summary (Priority: ERROR > WARN > INFO)
    let mut final_entries = Vec::new();
    
    // Process Errors
    let mut err_msgs: Vec<_> = error_counts.into_iter().collect();
    err_msgs.sort_by(|a, b| b.1.cmp(&a.1).then_with(|| a.0.cmp(&b.0)));
    for (msg, count) in err_msgs {
        let suffix = if count > 1 { format!(" (x{})", count) } else { "".to_string() };
        final_entries.push(format!("[ERROR] {}{}", msg, suffix));
    }

    // Process Warnings
    let mut warn_msgs: Vec<_> = warn_counts.into_iter().collect();
    warn_msgs.sort_by(|a, b| b.1.cmp(&a.1).then_with(|| a.0.cmp(&b.0)));
    for (msg, count) in warn_msgs {
        let suffix = if count > 1 { format!(" (x{})", count) } else { "".to_string() };
        final_entries.push(format!("[WARN] {}{}", msg, suffix));
    }

    // Process Info
    let mut info_msgs: Vec<_> = info_counts.into_iter().collect();
    info_msgs.sort_by(|a, b| b.1.cmp(&a.1).then_with(|| a.0.cmp(&b.0)));
    for (msg, count) in info_msgs {
        let suffix = if count > 1 { format!(" (x{})", count) } else { "".to_string() };
        final_entries.push(format!("[INFO] {}{}", msg, suffix));
    }

    SummaryResult {
        total_raw_lines: total_lines,
        summarized_count: final_entries.len(),
        entries: final_entries,
    }
}

fn main() -> io::Result<()> {
    let mut buffer = String::new();
    io::stdin().read_to_string(&mut buffer)?;

    if buffer.trim().is_empty() {
        return Ok(());
    }

    let response: LokiResponse = match serde_json::from_str(&buffer) {
        Ok(res) => res,
        Err(e) => {
            eprintln!("Error parsing Loki JSON: {}", e);
            return Err(io::Error::new(io::ErrorKind::InvalidData, e));
        }
    };

    let result = process_loki_response(response);
    println!("{}", serde_json::to_string(&result).unwrap());

    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    fn create_mock_response(level: &str, messages: Vec<&str>) -> LokiResponse {
        let mut stream = HashMap::new();
        stream.insert("level".to_string(), level.to_string());
        
        let values = messages.into_iter()
            .map(|m| vec!["12345".to_string(), m.to_string()])
            .collect();

        LokiResponse {
            status: "success".to_string(),
            data: LokiData {
                result_type: "streams".to_string(),
                result: vec![LokiStream {
                    stream,
                    values,
                }],
            },
        }
    }

    #[test]
    fn test_deduplication_info() {
        let resp = create_mock_response("info", vec!["pulse", "pulse", "unique"]);
        let result = process_loki_response(resp);
        
        assert_eq!(result.total_raw_lines, 3);
        assert_eq!(result.summarized_count, 2);
        assert!(result.entries.contains(&"[INFO] pulse (x2)".to_string()));
        assert!(result.entries.contains(&"[INFO] unique".to_string()));
    }

    #[test]
    fn test_deduplication_error() {
        let resp = create_mock_response("error", vec!["fail", "fail", "broken"]);
        let result = process_loki_response(resp);
        
        assert_eq!(result.total_raw_lines, 3);
        assert_eq!(result.summarized_count, 2);
        assert!(result.entries.contains(&"[ERROR] fail (x2)".to_string()));
        assert!(result.entries.contains(&"[ERROR] broken".to_string()));
    }

    #[test]
    fn test_level_priority() {
        let mut resp = create_mock_response("info", vec!["i1"]);
        resp.data.result.push(LokiStream {
            stream: {
                let mut h = HashMap::new();
                h.insert("level".to_string(), "error".to_string());
                h
            },
            values: vec![vec!["1".to_string(), "e1".to_string()]],
        });

        let result = process_loki_response(resp);
        
        // Errors should come before Info
        assert_eq!(result.entries[0], "[ERROR] e1");
        assert_eq!(result.entries[1], "[INFO] i1");
    }

    #[test]
    fn test_alternate_level_labels() {
        let mut stream = HashMap::new();
        stream.insert("detected_level".to_string(), "WARN".to_string());
        
        let resp = LokiResponse {
            status: "success".to_string(),
            data: LokiData {
                result_type: "streams".to_string(),
                result: vec![LokiStream {
                    stream,
                    values: vec![vec!["1".to_string(), "careful".to_string()]],
                }],
            },
        };

        let result = process_loki_response(resp);
        assert_eq!(result.entries[0], "[WARN] careful");
    }
}
