//! La coquille face aux reglages du serveur (lot L21.c) : ce que `levelup_desktop::shell_settings::ShellState`
//! decide (la croix, l'icone de la zone de notification, le menu), execute sur la vraie fenetre. Les decisions
//! sont testees dans la bibliotheque ; ici ne reste que ce qui touche tauri.

use std::sync::Arc;

use levelup_desktop::autostart::{self, AutostartOutcome};
use levelup_desktop::catalogue;
use levelup_desktop::logfile::LogSink;
use levelup_desktop::shell_settings::{CloseAction, ShellAction, ShellSettings, ShellState, TrayChange, TrayCommand};
use tauri::{AppHandle, Manager};

use crate::autostart_tauri::PluginAutostart;
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

/// La langue des libelles de la coquille : celle que le serveur annonce, sinon celle du systeme.
fn tray_locale(announced: &str) -> String {
    if announced.is_empty() {
        catalogue::resolve_locale(&sys_locale::get_locale().unwrap_or_default()).to_string()
    } else {
        announced.to_string()
    }
}

/// Execute sur la vraie fenetre ce que `ShellState` decide des reglages que le serveur annonce : l'icone de
/// la zone de notification et l'entree de demarrage avec Windows.
pub fn apply_shell_settings(app: &AppHandle, settings: &ShellSettings) {
    let Some(shell) = app.try_state::<Shell>() else {
        return;
    };
    let change = shell
        .shell
        .lock()
        .unwrap_or_else(|p| p.into_inner())
        .apply_settings(settings.clone());
    execute_tray_change(app, change, &tray_locale(&settings.locale));
    apply_autostart(app, settings.start_with_windows);
}

/// Ramene l'entree de demarrage avec Windows a l'etat voulu (`autostart::reconcile` : n'ecrit que pour changer).
fn apply_autostart(app: &AppHandle, wanted: bool) {
    let log = shell_log(app);
    let outcome = autostart::reconcile(wanted, &PluginAutostart { app: app.clone() });
    let text = match outcome {
        AutostartOutcome::Enabled => "event=autostart outcome=enabled".to_string(),
        AutostartOutcome::Disabled => "event=autostart outcome=disabled".to_string(),
        // L'etat voulu est deja l'etat reel : rien a dire a chaque annonce des reglages.
        AutostartOutcome::Unchanged => return,
        AutostartOutcome::Failed(reason) => format!("event=autostart outcome=failed err={reason}"),
    };
    report(log.as_deref(), &text);
}

/// Demarrage reduit (`--minimized`) : la fenetre est cachee, l'icone est creee tout de suite.
pub fn start_minimized(app: &AppHandle) {
    let Some(shell) = app.try_state::<Shell>() else {
        return;
    };
    let change = shell.shell.lock().unwrap_or_else(|p| p.into_inner()).start_minimized();
    execute_tray_change(app, change, &tray_locale(""));
}

/// La fenetre vient d'etre montree. Apres un demarrage reduit, l'icone se retire si `close_to_tray` est coupe ;
/// le retrait part sur le fil principal, apres le gestionnaire qui a montre la fenetre (un clic sur l'icone
/// elle-meme ne retire pas l'icone sous son propre gestionnaire).
pub fn window_shown(app: &AppHandle) {
    let Some(shell) = app.try_state::<Shell>() else {
        return;
    };
    let change = shell.shell.lock().unwrap_or_else(|p| p.into_inner()).on_window_shown();
    if change == TrayChange::Keep {
        return;
    }
    let deferred = app.clone();
    let locale = tray_locale(
        &shell
            .shell
            .lock()
            .unwrap_or_else(|p| p.into_inner())
            .settings()
            .locale
            .clone(),
    );
    if let Err(e) = app.run_on_main_thread(move || execute_tray_change(&deferred, change, &locale)) {
        report(shell_log(app).as_deref(), &format!("event=tray_change_failed err={e}"));
    }
}

/// Execute une decision sur l'icone et rend compte a l'etat.
fn execute_tray_change(app: &AppHandle, change: TrayChange, locale: &str) {
    let Some(shell) = app.try_state::<Shell>() else {
        return;
    };
    let log = shell_log(app);
    let state = |f: &dyn Fn(&mut ShellState)| f(&mut shell.shell.lock().unwrap_or_else(|p| p.into_inner()));
    match change {
        TrayChange::Create | TrayChange::Recreate => {
            tray_tauri::remove(app);
            match tray_tauri::create(app, locale) {
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
