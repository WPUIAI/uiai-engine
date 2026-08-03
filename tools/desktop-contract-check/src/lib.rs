// Lightweight parity harness for the Cockpit Rust contracts. Keeping this
// crate independent of Tauri lets contract fixtures run in ordinary CI loops
// without compiling the complete desktop dependency graph.
#[path = "../../../apps/cockpit/src-tauri/src/desktop_contract.rs"]
pub mod desktop_contract;
