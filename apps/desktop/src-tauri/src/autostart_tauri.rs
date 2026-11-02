//! Le plugin d'autostart de tauri derriere [`AutostartControl`] (lot L21.d). La decision est dans
//! `levelup_desktop::autostart` ; ici ne reste que le plugin. Il ecrit l'entree de l'utilisateur courant
//! (`HKCU\Software\Microsoft\Windows\CurrentVersion\Run`) ; elle lance l'application avec `--minimized`.

use levelup_desktop::autostart::AutostartControl;
use tauri::AppHandle;
use tauri_plugin_autostart::ManagerExt;

/// Lit et ecrit l'entree de demarrage par le plugin d'autostart.
pub struct PluginAutostart {
    pub app: AppHandle,
}

impl AutostartControl for PluginAutostart {
    fn is_enabled(&self) -> Result<bool, String> {
        self.app.autolaunch().is_enabled().map_err(|e| e.to_string())
    }

    fn enable(&self) -> Result<(), String> {
        self.app.autolaunch().enable().map_err(|e| e.to_string())
    }

    fn disable(&self) -> Result<(), String> {
        self.app.autolaunch().disable().map_err(|e| e.to_string())
    }
}
