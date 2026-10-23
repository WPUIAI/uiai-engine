//! UIAI Engine Cockpit — Tauri v2 desktop shell.
//!
//! Bridge commands (§17.3.1, §17.6), Bonjour discovery (§17.1), and
//! browser-profile configuration commands are exposed through one local bridge.

#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

mod bonjour;
mod bridge;
mod browser_profiles;
// Contract types are bound by T004-04; keep the strict release gate green while the scaffold is intentionally ahead of its presenter.
mod deep_link;
#[allow(dead_code)]
mod desktop_contract;
mod desktop_handoff;
#[allow(dead_code)]
mod focusa_credentials;
#[allow(dead_code)]
mod focusa_credentials_linux;
#[cfg(target_os = "macos")]
#[allow(dead_code)]
mod focusa_credentials_macos;
#[allow(dead_code)]
mod focusa_credentials_windows;
mod focusa_manifest_client;
mod focusa_manifest_server;
#[allow(dead_code)]
mod focusa_menubar_fallback;
#[allow(dead_code)]
mod focusa_menubar_mint;
#[allow(dead_code)]
mod focusa_pairing_client;
#[allow(dead_code)]
mod focusa_pairing_completion;
#[allow(dead_code)]
mod focusa_pairing_contract;
#[allow(dead_code)]
mod focusa_pairing_verify;

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
            bridge::focusa_start_pairing_bridge,
            bridge::focusa_clear_bridge,
            bonjour::focusa_discover_via_bonjour,
            deep_link::cockpit_take_deep_link,
            desktop_handoff::cockpit_open_focusa_handoff,
            focusa_manifest_server::cockpit_focusa_manifest_endpoint,
            focusa_manifest_client::cockpit_fetch_focusa_manifest,
            browser_profiles::browser_profiles_default,
            browser_profiles::browser_profiles_load,
            browser_profiles::browser_profiles_validate,
            browser_profiles::browser_profiles_save,
        ])
        .manage(deep_link::PendingDeepLink::default())
        .setup(|app| {
            deep_link::install(app);
            Ok(())
        })
        .run(tauri::generate_context!())
        .expect("error while running uaiengine-cockpit");
}
