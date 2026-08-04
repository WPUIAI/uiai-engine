//! UIAI Engine Cockpit — Tauri v2 desktop shell.
//!
//! Bridge commands (§17.3.1, §17.6) and Bonjour discovery (§17.1)
//! match the menubar FirstRunWizard implementation one-to-one.

#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

mod bonjour;
mod bridge;
// Contract types are bound by T004-04; keep the strict release gate green while the scaffold is intentionally ahead of its presenter.
mod deep_link;
#[allow(dead_code)]
mod desktop_contract;
mod desktop_handoff;
#[allow(dead_code)]
mod focusa_credentials;
mod focusa_manifest_client;
mod focusa_manifest_server;
#[allow(dead_code)]
mod focusa_pairing_contract;

fn main() {
    let focusa_manifest_endpoint = focusa_manifest_server::start()
        .expect("Focusa compatibility manifest server must bind loopback");
    tauri::Builder::default()
        .manage(focusa_manifest_endpoint)
        // Must be registered first so secondary activations are forwarded into the running app.
        .plugin(
            tauri_plugin_single_instance::Builder::new()
                .callback(|app, _args, _cwd| deep_link::focus_main_window(app))
                .build(),
        )
        .plugin(tauri_plugin_deep_link::init())
        .plugin(tauri_plugin_process::init())
        .plugin(tauri_plugin_updater::Builder::new().build())
        .invoke_handler(tauri::generate_handler![
            bridge::focusa_start_bridge_callback,
            bridge::focusa_take_bridge_completion,
            bridge::focusa_clear_bridge,
            bonjour::focusa_discover_via_bonjour,
            deep_link::cockpit_take_deep_link,
            desktop_handoff::cockpit_open_focusa_handoff,
            focusa_manifest_server::cockpit_focusa_manifest_endpoint,
            focusa_manifest_client::cockpit_fetch_focusa_manifest,
        ])
        .manage(deep_link::PendingDeepLink::default())
        .setup(|app| {
            deep_link::install(app);
            Ok(())
        })
        .run(tauri::generate_context!())
        .expect("error while running uaiengine-cockpit");
}
