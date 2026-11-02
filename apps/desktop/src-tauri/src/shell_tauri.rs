//! La coquille face aux reglages du serveur (lot L21.c) : ce que `levelup_desktop::shell_settings::ShellState`
//! decide (la croix, l'icone de la zone de notification, le menu), execute sur la vraie fenetre. Les decisions
//! sont testees dans la bibliotheque ; ici ne reste que ce qui touche tauri.

use std::sync::Arc;

use levelup_desktop::catalogue;
use levelup_desktop::logfile::LogSink;
use levelup_desktop::shell_settings::{CloseAction, ShellAction, ShellSettings, ShellState, TrayChange, TrayCommand};
use tauri::{AppHandle, Manager};

use crate::{MAIN_WINDOW, Shell, begin_close, focus_main, report, tray_tauri};

/// Ce que fait la croix : cacher dans la zone de notification ou quitter (`ShellState::on_close_requested`).
pub fn close_action(app: &AppHandle) -> CloseAction {
    app.try_state::<Shell>().map_or(CloseAction::Quit, |s| {
        s.shell.lock().unwrap_or_else(|p| p.into_inner()).on_close_requested()
    })
}

/// Cache la fenetre principale (zone de notification) ; le serveur continue.
pub fn hide_main(app: &AppHandle) {
    let log = shell_log(app);
    if let Some(window) = app.get_webview_window(MAIN_WINDOW) {
        match window.hide() {
            Ok(()) => report(log.as_deref(), "event=window_hidden reason=close_to_tray"),
            Err(e) => report(log.as_deref(), &format!("event=window_hide_failed err={e}")),
        }
    }
}

/// Le journal de la coquille, quand il est ouvert.
pub fn shell_log(app: &AppHandle) -> Option<Arc<LogSink>> {
    app.try_state::<Shell>()
        .and_then(|s| s.log.lock().unwrap_or_else(|p| p.into_inner()).clone())
}

/// Une entree du menu de l'icone : ce que decide `ShellState::on_tray_command`, execute ici.
pub fn on_tray_command(app: &AppHandle, command: TrayCommand) {
    let Some(shell) = app.try_state::<Shell>() else {
        return;
    };
    let action = shell
        .shell
        .lock()
        .unwrap_or_else(|p| p.into_inner())
        .on_tray_command(command);
    match action {
        ShellAction::ShowWindow => focus_main(app),
        ShellAction::RequestSync => {
            let closer = shell.closer.lock().unwrap_or_else(|p| p.into_inner()).clone();
            match closer {
                Some(closer) => closer.request_sync(),
                None => report(shell_log(app).as_deref(), "event=sync_skipped reason=no_server"),
            }
        }
        // La fermeture propre (stdin ferme, attente de `stopped`) : la fenetre revient pour montrer
        // « LevelUp se ferme ».
        ShellAction::Quit => {
            focus_main(app);
            begin_close(app);
        }
    }
}

/// Execute sur la vraie fenetre ce que `ShellState` decide des reglages que le serveur annonce.
pub fn apply_shell_settings(app: &AppHandle, settings: &ShellSettings) {
    let Some(shell) = app.try_state::<Shell>() else {
        return;
    };
    let log = shell_log(app);
    let change = shell
        .shell
        .lock()
        .unwrap_or_else(|p| p.into_inner())
        .apply_settings(settings.clone());
    let locale = if settings.locale.is_empty() {
        catalogue::resolve_locale(&sys_locale::get_locale().unwrap_or_default()).to_string()
    } else {
        settings.locale.clone()
    };
    let state = |f: &dyn Fn(&mut ShellState)| f(&mut shell.shell.lock().unwrap_or_else(|p| p.into_inner()));
    match change {
        TrayChange::Create | TrayChange::Recreate => {
            tray_tauri::remove(app);
            match tray_tauri::create(app, &locale) {
                Ok(()) => {
                    state(&ShellState::tray_created);
                    report(log.as_deref(), &format!("event=tray_created locale={locale}"));
                }
                Err(e) => {
                    state(&ShellState::tray_failed);
                    report(log.as_deref(), &format!("event=tray_failed err={e}"));
                }
            }
        }
        TrayChange::Remove => {
            tray_tauri::remove(app);
            state(&ShellState::tray_removed);
            report(log.as_deref(), "event=tray_removed");
            // Sans icone, une fenetre cachee ne se rouvrirait plus : elle revient.
            focus_main(app);
        }
        TrayChange::Keep => {}
    }
}
