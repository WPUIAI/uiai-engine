//! Focusa Cockpit — Tauri v2 desktop shell.
//!
//! Bridge commands (§17.3.1, §17.6) and Bonjour discovery (§17.1)
//! match the menubar FirstRunWizard implementation one-to-one.

#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

mod bridge;
mod bonjour;

fn main() {
    tauri::Builder::default()
        .invoke_handler(tauri::generate_handler![
            bridge::focusa_start_bridge_callback,
            bridge::focusa_take_bridge_completion,
            bridge::focusa_clear_bridge,
            bonjour::focusa_discover_via_bonjour,
        ])
        .setup(|_app| Ok(()))
        .run(tauri::generate_context!())
        .expect("error while running focusa-cockpit");
}