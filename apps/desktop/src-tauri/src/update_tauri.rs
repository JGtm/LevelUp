//! Le branchement de la mise a jour sur Tauri : le plugin d'update derriere [`UpdateSource`], une
//! boite de dialogue native derriere [`Prompt`]. Aucune decision ici (elles sont dans
//! `levelup_desktop::updater`), aucun libelle (ils viennent du catalogue).

use levelup_desktop::updater::{
    Decision, Installer, Notice, Notifier, Offer, Prompt, PromptText, UpdateSource, notice_text,
};
use tauri::AppHandle;
use tauri_plugin_updater::{Update, UpdaterExt};

/// Le plugin d'update de Tauri : interroge `latest.json`, telecharge l'installeur signe (la signature
/// est verifiee par le plugin avec la cle publique de `tauri.conf.json`) et le lance.
pub struct PluginSource {
    app: AppHandle,
    found: Option<Update>,
}

impl PluginSource {
    pub fn new(app: AppHandle) -> Self {
        Self { app, found: None }
    }
}

impl UpdateSource for PluginSource {
    fn check(&mut self) -> Result<Option<Offer>, String> {
        let updater = self.app.updater().map_err(|e| e.to_string())?;
        // Le fil de mise a jour n'est pas un fil asynchrone : il attend le plugin sur le moteur de Tauri.
        let found = tauri::async_runtime::block_on(updater.check()).map_err(|e| e.to_string())?;
        let offer = found.as_ref().map(|update| Offer {
            version: update.version.clone(),
            notes: update.body.clone(),
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
        let bytes = tauri::async_runtime::block_on(update.download(|_, _| {}, || {})).map_err(|e| e.to_string())?;
        // Sous Windows `install` lance l'installeur puis quitte l'application.
        Ok(Box::new(move || update.install(bytes).map_err(|e| e.to_string())))
    }
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

/// La boite d'information native posee apres l'arret du serveur (mise a jour repoussee, installation
/// echouee) : un seul bouton, le texte vient du catalogue.
pub struct DialogNotifier {
    pub locale: &'static str,
}

impl Notifier for DialogNotifier {
    fn show(&mut self, notice: Notice) {
        let text = notice_text(self.locale, notice);
        rfd::MessageDialog::new()
            .set_title(text.title.as_str())
            .set_description(text.message.as_str())
            .set_level(rfd::MessageLevel::Warning)
            .set_buttons(rfd::MessageButtons::Ok)
            .show();
    }
}
