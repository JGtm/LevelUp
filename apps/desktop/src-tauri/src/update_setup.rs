//! Le branchement de la mise a jour dans l'application : enregistrement du plugin, verification du
//! lancement et verification demandee depuis la page. Les decisions (verrou, build sans mise a jour,
//! fil separe) sont dans `levelup_desktop::update_manual` ; ici ne reste que ce qui touche tauri.

use std::sync::Arc;

use levelup_desktop::catalogue;
use levelup_desktop::logfile::LogSink;
use levelup_desktop::update_manual::{
    UpdateBusy, ensure_available, run_launch_check, run_manual_check, spawn_check_thread,
};
use levelup_desktop::updater::{CloserSlot, InstallGate, UpdateContext, run_update};
use tauri::{AppHandle, Manager};

use crate::report;
use crate::update_tauri::{DialogNotifier, DialogPrompt, PluginSource};

/// Ce que la mise a jour partage avec le superviseur : la porte d'installation (ouverte a la sortie de
/// l'application), la poignee de fermeture (posee apres le lancement du superviseur) et le verrou de la
/// verification en cours.
pub struct UpdateWiring {
    pub gate: InstallGate,
    pub closer: CloserSlot,
    pub busy: UpdateBusy,
    pub locale: &'static str,
}

/// Ce dont la verification demandee depuis la page (`/check-update` du protocole embarque) a besoin :
/// les memes pieces que la verification du lancement, plus l'etat du plugin d'update.
struct UpdateCheck {
    /// Le plugin d'update est enregistre (faux en build de developpement ou sans cle publique).
    enabled: bool,
    gate: InstallGate,
    closer: CloserSlot,
    busy: UpdateBusy,
    locale: &'static str,
    log: Option<Arc<LogSink>>,
}

/// Enregistre le plugin et range ce que la verification demandee depuis la page retrouvera. Rend les
/// pieces partagees avec le superviseur et l'etat du plugin.
pub fn install(app: &AppHandle, log: Option<Arc<LogSink>>) -> (UpdateWiring, bool) {
    let enabled = register_updater(app, log.as_deref());
    let update = UpdateWiring {
        gate: InstallGate::default(),
        closer: CloserSlot::default(),
        busy: UpdateBusy::default(),
        locale: catalogue::resolve_locale(&sys_locale::get_locale().unwrap_or_default()),
    };
    app.manage(UpdateCheck {
        enabled,
        gate: update.gate.clone(),
        closer: update.closer.clone(),
        busy: update.busy.clone(),
        locale: update.locale,
        log,
    });
    (update, enabled)
}

/// La verification de mise a jour, lancee une fois par lancement quand le serveur est pret : sur son
/// propre fil, jamais bloquante, chaque issue sans installation est journalisee.
pub fn spawn_update_check(app: AppHandle, update: UpdateWiring, log: Arc<LogSink>) -> Box<dyn FnOnce() + Send> {
    let UpdateWiring {
        gate,
        closer,
        busy,
        locale,
    } = update;
    Box::new(move || {
        let thread_log = log.clone();
        let spawned = spawn_check_thread("update", move || {
            run_launch_check(&busy, || {
                run_update(
                    PluginSource::new(app, locale, closer.clone()),
                    DialogPrompt,
                    locale,
                    &gate,
                    &closer,
                    &thread_log,
                );
            });
        });
        if let Err(e) = spawned {
            report(Some(log.as_ref()), &format!("event=update_thread_failed err={e}"));
        }
    })
}

/// La verification demandee par l'utilisateur depuis la page (`/check-update`) : sur son propre fil, car les
/// boites natives bloquent et la reponse au protocole ne doit pas attendre. Meme sequence que celle du
/// lancement ; en build sans updater, la raison est dite dans une boite.
pub fn request_update_check(app: AppHandle) {
    let Some(check) = app.try_state::<UpdateCheck>() else {
        return;
    };
    let (enabled, gate, closer, busy, locale, log) = (
        check.enabled,
        check.gate.clone(),
        check.closer.clone(),
        check.busy.clone(),
        check.locale,
        check.log.clone(),
    );
    let Some(log) = log else {
        return;
    };
    let thread_log = log.clone();
    let spawned = spawn_check_thread("update-check", move || {
        let mut notifier = DialogNotifier { locale };
        if !ensure_available(enabled, &mut notifier, &thread_log) {
            return;
        }
        let context = UpdateContext {
            locale,
            gate: &gate,
            closer: &closer,
            log: &thread_log,
        };
        run_manual_check(
            PluginSource::new(app, locale, closer.clone()),
            DialogPrompt,
            &mut notifier,
            &busy,
            &context,
        );
    });
    if let Err(e) = spawned {
        report(Some(log.as_ref()), &format!("event=update_check_thread_failed err={e}"));
    }
}

/// L'updater n'est branche que si la cle publique est posee (`build.rs` refuse une release sans elle) et
/// hors des builds de developpement (un build de debug ne se remplace pas par la derniere release).
/// Rend vrai quand le plugin est enregistre.
fn register_updater(app: &AppHandle, log: Option<&LogSink>) -> bool {
    if cfg!(debug_assertions) {
        report(log, "event=updater_disabled reason=debug_build");
        return false;
    }
    let configured = app
        .config()
        .plugins
        .0
        .get("updater")
        .and_then(|conf| conf.get("pubkey"))
        .and_then(|key| key.as_str())
        .is_some_and(|key| !key.trim().is_empty());
    if !configured {
        report(log, "event=updater_disabled reason=no_public_key");
        return false;
    }
    if let Err(e) = app.plugin(tauri_plugin_updater::Builder::new().build()) {
        report(log, &format!("event=updater_failed err={e}"));
        return false;
    }
    true
}
