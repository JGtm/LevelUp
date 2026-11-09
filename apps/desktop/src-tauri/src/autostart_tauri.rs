//! Le plugin d'autostart de tauri derriere [`AutostartControl`] (lot L21.d). La decision est dans
//! `levelup_desktop::autostart` ; ici ne reste que le plugin et la lecture du registre. Il ecrit l'entree de
//! l'utilisateur courant (`HKCU\Software\Microsoft\Windows\CurrentVersion\Run`) ; elle lance l'application avec
//! `--minimized`.
//!
//! Le plugin ne sait dire que « presente ET approuvee » : il ne distingue pas une entree absente d'une entree que
//! l'utilisateur a desactivee dans le gestionnaire des taches. Les deux faits se lisent donc dans le registre
//! (`Run` pour la presence, `Explorer\StartupApproved\Run` pour l'approbation) ; l'ecriture reste au plugin.

use levelup_desktop::autostart::{AutostartControl, EntryState};
use tauri::AppHandle;
use tauri_plugin_autostart::ManagerExt;

/// Lit et ecrit l'entree de demarrage par le plugin d'autostart.
pub struct PluginAutostart {
    pub app: AppHandle,
}

impl AutostartControl for PluginAutostart {
    #[cfg(windows)]
    fn entry_state(&self) -> Result<EntryState, String> {
        registry::entry_state()
    }

    /// Hors Windows (compilation seule, la coquille est Windows seul) : le plugin ne voit que « approuvee ».
    #[cfg(not(windows))]
    fn entry_state(&self) -> Result<EntryState, String> {
        let enabled = self.app.autolaunch().is_enabled().map_err(|e| e.to_string())?;
        Ok(if enabled {
            EntryState::Approved
        } else {
            EntryState::Absent
        })
    }

    fn enable(&self) -> Result<(), String> {
        self.app.autolaunch().enable().map_err(|e| e.to_string())
    }

    fn disable(&self) -> Result<(), String> {
        self.app.autolaunch().disable().map_err(|e| e.to_string())
    }
}

#[cfg(windows)]
mod registry {
    use std::io::ErrorKind;

    use levelup_desktop::autostart::{ENTRY_NAME, EntryState, entry_state as decide};
    use winreg::RegKey;
    use winreg::enums::{HKEY_CURRENT_USER, KEY_READ};

    const RUN_KEY: &str = r"Software\Microsoft\Windows\CurrentVersion\Run";
    const APPROVED_KEY: &str = r"Software\Microsoft\Windows\CurrentVersion\Explorer\StartupApproved\Run";

    /// Presence de la valeur `Run` et contenu de sa valeur d'approbation, lus dans `HKCU`.
    pub fn entry_state() -> Result<EntryState, String> {
        let hkcu = RegKey::predef(HKEY_CURRENT_USER);
        let present = match hkcu.open_subkey_with_flags(RUN_KEY, KEY_READ) {
            Ok(run) => match run.get_raw_value(ENTRY_NAME) {
                Ok(_) => true,
                Err(e) if e.kind() == ErrorKind::NotFound => false,
                Err(e) => return Err(e.to_string()),
            },
            Err(e) if e.kind() == ErrorKind::NotFound => false,
            Err(e) => return Err(e.to_string()),
        };
        // Pas de valeur d'approbation (ou cle absente) : Windows la comprend comme approuvee.
        let approval = hkcu
            .open_subkey_with_flags(APPROVED_KEY, KEY_READ)
            .ok()
            .and_then(|key| key.get_raw_value(ENTRY_NAME).ok())
            .map(|value| value.bytes);
        Ok(decide(present, approval.as_deref()))
    }
}
