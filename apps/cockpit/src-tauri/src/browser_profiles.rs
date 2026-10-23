use serde_json::{json, Value as JsonValue};
use serde_yaml::{Mapping, Value as YamlValue};
use std::fs;
use std::path::{Path, PathBuf};

fn browser_key() -> YamlValue {
    YamlValue::String("browser".to_string())
}

fn read_yaml_root(path: &Path) -> Result<YamlValue, String> {
    let raw = fs::read_to_string(path)
        .map_err(|error| format!("read {}: {error}", path.display()))?;
    serde_yaml::from_str(&raw).map_err(|error| format!("parse {}: {error}", path.display()))
}

fn browser_json_from_root(root: &YamlValue) -> Result<JsonValue, String> {
    let mapping = root
        .as_mapping()
        .ok_or_else(|| "engine config root must be a YAML mapping".to_string())?;
    let browser = mapping
        .get(&browser_key())
        .cloned()
        .unwrap_or_else(|| serde_yaml::to_value(default_browser_profiles()).unwrap_or(YamlValue::Null));
    serde_json::to_value(browser).map_err(|error| format!("convert browser config to JSON: {error}"))
}

fn validate_browser_json(browser: &JsonValue) -> Result<(), String> {
    let object = browser
        .as_object()
        .ok_or_else(|| "browser settings must be an object".to_string())?;
    let default_profile = object
        .get("default_profile")
        .and_then(JsonValue::as_str)
        .ok_or_else(|| "browser.default_profile is required".to_string())?;
    let profiles = object
        .get("profiles")
        .and_then(JsonValue::as_object)
        .ok_or_else(|| "browser.profiles must be an object".to_string())?;
    if !profiles.contains_key(default_profile) {
        return Err(format!("default profile {default_profile:?} does not exist"));
    }
    for (name, profile) in profiles {
        let profile = profile
            .as_object()
            .ok_or_else(|| format!("profile {name:?} must be an object"))?;
        let mode = profile
            .get("mode")
            .and_then(JsonValue::as_str)
            .or_else(|| {
                if profile.get("extends").is_some() {
                    None
                } else {
                    Some("detect")
                }
            });
        if let Some(mode) = mode {
            if !matches!(mode, "detect" | "no_detect" | "operator" | "research" | "auto") {
                return Err(format!("profile {name:?} has invalid mode {mode:?}"));
            }
        }
        if let Some(engine) = profile.get("engine").and_then(JsonValue::as_str) {
            if !matches!(engine, "chromium" | "system_chromium" | "camoufox") {
                return Err(format!("profile {name:?} has invalid engine {engine:?}"));
            }
        }
    }
    Ok(())
}

fn atomic_write(path: &Path, contents: &str) -> Result<(), String> {
    let parent = path.parent().unwrap_or_else(|| Path::new("."));
    let filename = path
        .file_name()
        .and_then(|name| name.to_str())
        .ok_or_else(|| "invalid config filename".to_string())?;
    let temporary = parent.join(format!(".{filename}.uiai.tmp"));
    fs::write(&temporary, contents)
        .map_err(|error| format!("write {}: {error}", temporary.display()))?;
    fs::rename(&temporary, path)
        .map_err(|error| format!("replace {}: {error}", path.display()))
}

fn backup_path(path: &Path) -> PathBuf {
    let mut backup = path.as_os_str().to_os_string();
    backup.push(".bak");
    PathBuf::from(backup)
}

#[tauri::command]
pub fn browser_profiles_default() -> JsonValue {
    default_browser_profiles()
}

#[tauri::command]
pub fn browser_profiles_load(path: String) -> Result<JsonValue, String> {
    let root = read_yaml_root(Path::new(&path))?;
    browser_json_from_root(&root)
}

#[tauri::command]
pub fn browser_profiles_validate(browser: JsonValue) -> Result<JsonValue, String> {
    validate_browser_json(&browser)?;
    Ok(json!({ "valid": true, "browser": browser }))
}

#[tauri::command]
pub fn browser_profiles_save(path: String, browser: JsonValue) -> Result<JsonValue, String> {
    validate_browser_json(&browser)?;
    let path = PathBuf::from(path);
    let mut root = if path.exists() {
        read_yaml_root(&path)?
    } else {
        YamlValue::Mapping(Mapping::new())
    };
    let mapping = root
        .as_mapping_mut()
        .ok_or_else(|| "engine config root must be a YAML mapping".to_string())?;
    let browser_yaml = serde_yaml::to_value(&browser)
        .map_err(|error| format!("convert browser settings to YAML: {error}"))?;
    mapping.insert(browser_key(), browser_yaml);

    if path.exists() {
        fs::copy(&path, backup_path(&path))
            .map_err(|error| format!("backup {}: {error}", path.display()))?;
    }
    let encoded = serde_yaml::to_string(&root)
        .map_err(|error| format!("encode {}: {error}", path.display()))?;
    atomic_write(&path, &encoded)?;
    Ok(json!({
        "saved": true,
        "path": path,
        "backup": backup_path(&path),
        "browser": browser
    }))
}

fn default_browser_profiles() -> JsonValue {
    json!({
        "default_profile": "detect",
        "profiles": {
            "detect": {
                "label": "Detect / Diagnostic",
                "mode": "detect",
                "engine": "system_chromium",
                "headless": true,
                "network": { "route": "direct" },
                "challenge": { "policy": "detect", "max_attempts": 1 },
                "observability": {
                    "diagnostics_level": "developer",
                    "fingerprint_capture": true,
                    "challenge_capture": true,
                    "network_capture": true
                }
            },
            "no_detect": {
                "extends": "detect",
                "label": "No Detect",
                "mode": "no_detect",
                "identity": {
                    "patch_webdriver": true,
                    "patch_chrome_object": true,
                    "patch_plugins": true,
                    "patch_languages": true,
                    "disable_automation_controlled": true
                },
                "network": {
                    "route": "local_ip_pool",
                    "route_ref": "captcha-default",
                    "dns_mode": "proxy",
                    "webrtc_mode": "proxy_only",
                    "geo_consistency": true,
                    "sticky": true
                },
                "challenge": {
                    "policy": "solve_and_retry",
                    "max_attempts": 3,
                    "route_rotation": true,
                    "operator_escalation": true
                }
            },
            "operator": {
                "extends": "detect",
                "label": "Operator",
                "mode": "operator",
                "headless": false,
                "launch": {
                    "persistent_context": true,
                    "user_data_dir": "${UIAI_OPERATOR_PROFILE_DIR}"
                },
                "network": { "route": "operator_route", "sticky": true },
                "storage": {
                    "cookie_mode": "persistent",
                    "cache_mode": "persistent",
                    "local_storage_mode": "persistent",
                    "exclusive_lock": true
                },
                "challenge": { "policy": "assist", "operator_escalation": true }
            },
            "research": {
                "extends": "no_detect",
                "label": "Research / Eval",
                "mode": "research",
                "observability": {
                    "diagnostics_level": "full",
                    "fingerprint_capture": true,
                    "challenge_capture": true,
                    "network_capture": true,
                    "evidence_grade": true
                }
            }
        },
        "domain_rules": []
    })
}
