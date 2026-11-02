//! Le branchement de la mise a jour sur Tauri : le plugin d'update derriere [`UpdateSource`], une
//! boite de dialogue native derriere [`Prompt`]. Aucune decision ici (elles sont dans
//! `levelup_desktop::updater`), aucun libelle (ils viennent du catalogue).

use levelup_desktop::download_watch::{POLL_INTERVAL, Progress, Slot, watch_download};
use levelup_desktop::update_notes::notes_for_locale;
use levelup_desktop::updater::{
    CHECK_TIMEOUT, Decision, Installer, Notice, Notifier, Offer, Prompt, PromptText, UpdateSource, notice_text,
};
use tauri::AppHandle;
use tauri_plugin_updater::{Update, UpdaterExt};

/// Le plugin d'update de Tauri : interroge `latest.json`, telecharge l'installeur signe (la signature
/// est verifiee par le plugin avec la cle publique de `tauri.conf.json`) et le lance.
pub struct PluginSource {
    app: AppHandle,
    locale: &'static str,
    found: Option<Update>,
}

impl PluginSource {
    /// `locale` : la langue de la coquille, qui choisit les notes montrees (`notes_for_locale`).
    pub fn new(app: AppHandle, locale: &'static str) -> Self {
        Self {
            app,
            locale,
            found: None,
        }
    }
}

impl UpdateSource for PluginSource {
    fn check(&mut self) -> Result<Option<Offer>, String> {
        let updater = self
            .app
            .updater_builder()
            .timeout(CHECK_TIMEOUT)
            .build()
            .map_err(|e| e.to_string())?;
        // Le fil de mise a jour n'est pas un fil asynchrone : il attend le plugin sur le moteur de Tauri.
        let found = tauri::async_runtime::block_on(updater.check()).map_err(|e| e.to_string())?;
        let offer = found.as_ref().map(|update| Offer {
            version: update.version.clone(),
            // `body` est la cle `notes` : le repli quand le document n'en porte aucune pour la langue.
            notes: notes_for_locale(self.locale, &update.raw_json).or_else(|| update.body.clone()),
        });
        self.found = found;
        Ok(offer)
    }

    fn download(&mut self, offer: &Offer) -> Result<Installer, String> {
        let update = self.found.take().ok_or("aucune mise a jour verifiee")?;
        if update.version != offer.version {
            return Err(format!(
                "la mise a jour verifiee ({}) n'est pas celle proposee ({})",
                update.version, offer.version
            ));
        }
        let bytes = download_watched(&update)?;
        // Sous Windows `install` lance l'installeur puis quitte l'application.
        Ok(Box::new(move || update.install(bytes).map_err(|e| e.to_string())))
    }
}

/// Telecharge l'installeur sous surveillance : le telechargement tourne sur le moteur de Tauri, ce fil
/// lit le compteur de progression et abandonne (`abort`) sur le verdict de `download_watch` (aucun octet
/// depuis la limite d'inactivite, ou duree maximale). L'abandon rend une erreur : le verrou de
/// verification est rendu par l'appelant, qui journalise.
fn download_watched(update: &Update) -> Result<Vec<u8>, String> {
    let progress = Progress::new();
    let slot: Slot<Vec<u8>> = Slot::default();
    let (seen, done, target) = (progress.clone(), slot.clone(), update.clone());
    let task = tauri::async_runtime::spawn(async move {
        let result = target
            .download(move |_, _| seen.record(), || {})
            .await
            .map_err(|e| e.to_string());
        *done.lock().unwrap_or_else(|p| p.into_inner()) = Some(result);
    });
    // La boucle (resultat lu avant le verdict, abandon une fois) vit dans `download_watch`, testee.
    watch_download(
        &slot,
        || progress.check(),
        || task.abort(),
        || std::thread::sleep(POLL_INTERVAL),
    )
}

/// La boite de dialogue native : deux boutons nommes par le catalogue (« Installer et redemarrer »,
/// « Plus tard »). Fermer la boite (croix, Echap) vaut « Plus tard ».
pub struct DialogPrompt;

impl Prompt for DialogPrompt {
    fn ask(&mut self, text: &PromptText) -> Decision {
        let answer = rfd::MessageDialog::new()
            .set_title(text.title.as_str())
            .set_description(text.message.as_str())
            .set_level(rfd::MessageLevel::Info)
            .set_buttons(rfd::MessageButtons::OkCancelCustom(
                text.install.clone(),
                text.later.clone(),
            ))
            .show();
        match answer {
            // Sans les controles v6 la boite retombe sur OK / Annuler : OK est « installer ».
            rfd::MessageDialogResult::Ok => Decision::Install,
            rfd::MessageDialogResult::Custom(label) if label == text.install => Decision::Install,
            _ => Decision::Later,
        }
    }
}

/// La boite d'information native (mise a jour repoussee, installation echouee, resultat d'une verification
/// demandee) : un seul bouton, le texte vient du catalogue.
pub struct DialogNotifier {
    pub locale: &'static str,
}

impl Notifier for DialogNotifier {
    fn show(&mut self, notice: Notice) {
        let text = notice_text(self.locale, notice);
        rfd::MessageDialog::new()
            .set_title(text.title.as_str())
            .set_description(text.message.as_str())
            .set_level(if notice.is_informational() {
                rfd::MessageLevel::Info
            } else {
                rfd::MessageLevel::Warning
            })
            .set_buttons(rfd::MessageButtons::Ok)
            .show();
    }
}
