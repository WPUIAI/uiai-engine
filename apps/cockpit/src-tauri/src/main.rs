//! UIAI Engine Cockpit — Tauri v2 desktop shell.
//!
//! Bridge commands (§17.3.1, §17.6), Bonjour discovery (§17.1), and
//! browser-profile configuration commands are exposed through one local bridge.

#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

mod bridge;
mod bonjour;
mod browser_profiles;

fn main() {
    tauri::Builder::default()
        .invoke_handler(tauri::generate_handler![
            bridge::focusa_start_bridge_callback,
            bridge::focusa_take_bridge_completion,
            bridge::focusa_clear_bridge,
            bonjour::focusa_discover_via_bonjour,
            browser_profiles::browser_profiles_default,
            browser_profiles::browser_profiles_load,
            browser_profiles::browser_profiles_validate,
            browser_profiles::browser_profiles_save,
        ])
        .setup(|_app| Ok(()))
        .run(tauri::generate_context!())
        .expect("error while running uaiengine-cockpit");
}
