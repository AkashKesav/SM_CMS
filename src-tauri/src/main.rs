// Prevents additional console window on Windows in release
#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

use tauri::{
    menu::{Menu, MenuItem},
    tray::{MouseButton, MouseButtonState, TrayIconBuilder, TrayIconEvent},
    Emitter, Manager,
};

fn main() {
    tauri::Builder::default()
        .plugin(tauri_plugin_shell::init())
        .plugin(tauri_plugin_store::Builder::default().build())
        .plugin(tauri_plugin_fs::init())
        .plugin(tauri_plugin_dialog::init())
        .plugin(tauri_plugin_notification::init())
        .setup(|app| {
            // Create tray menu
            let toggle_menu_item =
                MenuItem::with_id(app, "toggle", "Show/Hide", true, None::<&str>)?;
            let sync_schema_item =
                MenuItem::with_id(app, "sync_schema", "Sync Schema", true, None::<&str>)?;
            let clear_cache_item =
                MenuItem::with_id(app, "clear_cache", "Clear Cache", true, None::<&str>)?;
            let quit_item = MenuItem::with_id(app, "quit", "Quit", true, None::<&str>)?;

            let tray_menu = Menu::with_items(
                app,
                &[
                    &toggle_menu_item,
                    &sync_schema_item,
                    &clear_cache_item,
                    &quit_item,
                ],
            )?;

            // Build tray icon
            let _tray = TrayIconBuilder::new()
                .icon(app.default_window_icon().unwrap().clone())
                .menu(&tray_menu)
                .on_menu_event(|app, event| match event.id.as_ref() {
                    "toggle" => {
                        if let Some(window) = app.get_webview_window("main") {
                            if window.is_visible().unwrap_or(true) {
                                let _ = window.hide();
                            } else {
                                let _ = window.show();
                                let _ = window.set_focus();
                            }
                        }
                    }
                    "sync_schema" => {
                        // Emit event to frontend to trigger schema refresh
                        if let Some(window) = app.get_webview_window("main") {
                            let _ = window.emit("sync-schema-request", ());
                        }
                    }
                    "clear_cache" => {
                        // Clear application cache
                        if let Some(window) = app.get_webview_window("main") {
                            let _ = window.emit("clear-cache-request", ());
                        }
                    }
                    "quit" => {
                        app.exit(0);
                    }
                    _ => {}
                })
                .on_tray_icon_event(|tray, event| {
                    if let TrayIconEvent::Click {
                        button: MouseButton::Left,
                        button_state: MouseButtonState::Up,
                        ..
                    } = event
                    {
                        let app = tray.app_handle();
                        if let Some(window) = app.get_webview_window("main") {
                            let _ = window.show();
                            let _ = window.set_focus();
                        }
                    }
                })
                .build(app)?;

            Ok(())
        })
        .invoke_handler(tauri::generate_handler![
            commands::get_app_config,
            commands::update_app_config,
            commands::check_backend_health,
        ])
        .run(tauri::generate_context!())
        .expect("error while running CMS Desktop");
}

mod commands {
    use serde::{Deserialize, Serialize};
    use std::collections::HashMap;

    #[derive(Serialize, Deserialize, Clone)]
    pub struct AppConfig {
        pub backend_url: String,
        pub supabase_url: String,
        pub supabase_anon_key: String,
        pub theme: String,
        pub language: String,
    }

    #[tauri::command]
    pub fn get_app_config(_app_handle: tauri::AppHandle) -> Result<AppConfig, String> {
        // Load config from store or return defaults
        let config = AppConfig {
            backend_url: std::env::var("BACKEND_URL")
                .unwrap_or_else(|_| "http://localhost:8080".to_string()),
            supabase_url: std::env::var("VITE_SUPABASE_URL").unwrap_or_default(),
            supabase_anon_key: std::env::var("VITE_SUPABASE_ANON_KEY").unwrap_or_default(),
            theme: "system".to_string(),
            language: "en".to_string(),
        };
        Ok(config)
    }

    #[tauri::command]
    pub fn update_app_config(
        _app_handle: tauri::AppHandle,
        _config: HashMap<String, String>,
    ) -> Result<(), String> {
        // Save config to store
        Ok(())
    }

    #[tauri::command]
    pub async fn check_backend_health(url: String) -> Result<bool, String> {
        // Check if backend is responding
        let health_url = format!("{}/health", url.trim_end_matches('/'));

        // Simple health check - in production use proper HTTP client
        match reqwest::get(&health_url).await {
            Ok(response) => Ok(response.status().is_success()),
            Err(_) => Ok(false),
        }
    }
}
