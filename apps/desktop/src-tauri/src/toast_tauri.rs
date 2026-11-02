//! Le branchement des notifications Windows sur tauri : le plugin de notification derriere
//! [`NativeNotifier`] et l'etat de la fenetre principale. Les decisions sont dans
//! `levelup_desktop::native_notice` ; ici ne reste que ce qui touche tauri.
//!
//! Le clic sur une notification ramene la fenetre au premier plan par le chemin de la seconde ouverture :
//! une notification Windows activee relance l'executable, le plugin d'instance unique l'intercepte et
//! `focus_main` montre, restaure et met la fenetre au premier plan (le plugin de notification n'offre aucun
//! rappel de clic sur le bureau).

use levelup_desktop::events::Toast;
use levelup_desktop::logfile::LogSink;
use levelup_desktop::native_notice::{NativeNotifier, WindowState};
use tauri::{AppHandle, Manager};
use tauri_plugin_notification::NotificationExt;

use crate::{MAIN_WINDOW, report};

/// Montre une notification par le plugin de notification.
pub struct PluginNotifier {
    pub app: AppHandle,
}

impl NativeNotifier for PluginNotifier {
    fn show(&mut self, toast: &Toast) -> Result<(), String> {
        self.app
            .notification()
            .builder()
            .title(&toast.title)
            .body(&toast.body)
            .show()
            .map_err(|e| e.to_string())
    }
}

/// L'etat de la fenetre principale. Une fenetre introuvable, ou qui ne repond pas, vaut `Hidden` : mieux
/// vaut une notification de trop qu'une notification perdue pour un utilisateur qui ne voit pas l'application.
pub fn window_state(app: &AppHandle, log: &LogSink) -> WindowState {
    let Some(window) = app.get_webview_window(MAIN_WINDOW) else {
        report(Some(log), "event=window_state_unavailable reason=no_window");
        return WindowState::Hidden;
    };
    let flags = (|| Ok::<_, tauri::Error>((window.is_visible()?, window.is_minimized()?, window.is_focused()?)))();
    match flags {
        Ok((visible, minimized, focused)) => WindowState::from_flags(visible, minimized, focused),
        Err(e) => {
            report(Some(log), &format!("event=window_state_unavailable err={e}"));
            WindowState::Hidden
        }
    }
}
