//! Les reglages que le serveur annonce a la coquille (evenement `shell_settings`, ADR 0045 D-2) et ce que la
//! coquille en decide, sans fenetre ni tauri : la machine d'etats de la zone de notification (lot L21.c).
//!
//! Le serveur reste la source des reglages (`close_to_tray`, `native_notifications`, `start_with_windows`) ;
//! la coquille ne les lit pas dans un fichier, elle les recoit : une fois le serveur pret, puis a chaque
//! changement. [`ShellState`] decide : l'icone existe-t-elle, la croix cache-t-elle ou quitte-t-elle, que fait
//! chaque entree du menu. Le binaire (`main.rs`) execute ces decisions sur la vraie fenetre.
//!
//! Invariant : la fenetre n'est jamais cachee sans que l'icone, seul moyen de la rouvrir, existe. Si la creation
//! de l'icone echoue, la croix quitte.

use crate::catalogue;

/// Identifiants des entrees du menu de l'icone.
pub const MENU_OPEN: &str = "open";
pub const MENU_SYNC: &str = "sync";
pub const MENU_QUIT: &str = "quit";

/// Les reglages que la coquille applique, tels que le serveur les annonce.
#[derive(Debug, Clone, PartialEq, Eq, Default)]
pub struct ShellSettings {
    /// Relayer les notifications de la cloche en notifications Windows (decision du serveur ; la coquille
    /// n'en lit que l'annonce).
    pub native_notifications: bool,
    /// La croix reduit l'application dans la zone de notification au lieu de la quitter.
    pub close_to_tray: bool,
    /// Lancer l'application reduite a l'ouverture de la session Windows.
    pub start_with_windows: bool,
    /// Langue des libelles de la coquille (menu de l'icone) : celle du joueur ; vide = celle du systeme.
    pub locale: String,
}

/// Ce que fait la croix de la fenetre.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum CloseAction {
    /// Cacher la fenetre ; le serveur continue.
    HideToTray,
    /// Quitter : la sequence d'arret propre (fermeture de stdin, attente de `stopped`).
    Quit,
}

/// Ce que l'application doit faire de l'icone apres un changement de reglages.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum TrayChange {
    /// L'icone doit exister et n'existe pas.
    Create,
    /// L'icone existe mais ses libelles ne sont plus dans la langue du joueur : la refaire.
    Recreate,
    /// L'icone ne doit plus exister.
    Remove,
    /// Rien a faire.
    Keep,
}

/// Une entree du menu de l'icone.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum TrayCommand {
    Open,
    SyncNow,
    Quit,
}

/// Ce que l'application execute pour une commande du menu.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum ShellAction {
    ShowWindow,
    RequestSync,
    /// Meme chemin que la croix sans icone : la sequence d'arret propre.
    Quit,
}

/// L'entree de menu de cet identifiant.
pub fn command_for_menu_id(id: &str) -> Option<TrayCommand> {
    match id {
        MENU_OPEN => Some(TrayCommand::Open),
        MENU_SYNC => Some(TrayCommand::SyncNow),
        MENU_QUIT => Some(TrayCommand::Quit),
        _ => None,
    }
}

/// Les libelles du menu, dans la langue donnee (repli : la langue par defaut du catalogue compile).
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct TrayLabels {
    pub open: &'static str,
    pub sync: &'static str,
    pub quit: &'static str,
}

/// Les libelles du menu dans `locale`. Le catalogue est compile depuis `desktop.toml` (le fichier meme que le
/// serveur sert a `/desktop/strings`) : aucune langue n'est ecrite ici.
pub fn tray_labels(locale: &str) -> TrayLabels {
    let label = |key: &str| catalogue::text(locale, key).unwrap_or_default();
    TrayLabels {
        open: label("tray_open"),
        sync: label("tray_sync"),
        quit: label("tray_quit"),
    }
}

/// L'etat de la coquille face aux reglages : decide, n'execute rien.
#[derive(Debug, Clone, Default)]
pub struct ShellState {
    settings: ShellSettings,
    /// L'icone existe vraiment (la creation a reussi).
    tray_present: bool,
    /// Langue des libelles de l'icone existante.
    tray_locale: String,
    /// Des reglages ont ete recus au moins une fois.
    configured: bool,
}

impl ShellState {
    pub fn new() -> Self {
        Self::default()
    }

    /// Les reglages courants (ceux du dernier `shell_settings`).
    pub fn settings(&self) -> &ShellSettings {
        &self.settings
    }

    pub fn tray_present(&self) -> bool {
        self.tray_present
    }

    /// L'icone est voulue : tant que le reglage `close_to_tray` est actif.
    fn tray_wanted(&self) -> bool {
        self.settings.close_to_tray
    }

    /// Prend les reglages annonces et dit quoi faire de l'icone. L'appelant execute puis rend compte par
    /// [`ShellState::tray_created`], [`ShellState::tray_removed`] ou [`ShellState::tray_failed`].
    pub fn apply_settings(&mut self, settings: ShellSettings) -> TrayChange {
        self.settings = settings;
        self.configured = true;
        match (self.tray_wanted(), self.tray_present) {
            (true, false) => TrayChange::Create,
            (true, true) if self.tray_locale != self.settings.locale => TrayChange::Recreate,
            (false, true) => TrayChange::Remove,
            _ => TrayChange::Keep,
        }
    }

    /// L'icone a ete creee avec les libelles de la langue courante.
    pub fn tray_created(&mut self) {
        self.tray_present = true;
        self.tray_locale = self.settings.locale.clone();
    }

    /// L'icone a ete retiree.
    pub fn tray_removed(&mut self) {
        self.tray_present = false;
    }

    /// La creation de l'icone a echoue : elle n'existe pas, la croix quittera (jamais une fenetre cachee
    /// sans icone pour la rouvrir).
    pub fn tray_failed(&mut self) {
        self.tray_present = false;
    }

    /// La croix, Alt+F4 ou la barre des taches : cacher dans la zone de notification seulement si le reglage
    /// est actif ET que l'icone existe.
    pub fn on_close_requested(&self) -> CloseAction {
        if self.settings.close_to_tray && self.tray_present {
            CloseAction::HideToTray
        } else {
            CloseAction::Quit
        }
    }

    /// Ce que fait une entree du menu de l'icone.
    pub fn on_tray_command(&self, command: TrayCommand) -> ShellAction {
        match command {
            TrayCommand::Open => ShellAction::ShowWindow,
            TrayCommand::SyncNow => ShellAction::RequestSync,
            TrayCommand::Quit => ShellAction::Quit,
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn settings(close_to_tray: bool, locale: &str) -> ShellSettings {
        ShellSettings {
            close_to_tray,
            locale: locale.into(),
            ..ShellSettings::default()
        }
    }

    #[test]
    fn par_defaut_la_croix_quitte_et_aucune_icone_n_existe() {
        let state = ShellState::new();
        assert!(!state.tray_present());
        assert_eq!(state.on_close_requested(), CloseAction::Quit);
    }

    #[test]
    fn reglage_coupe_la_croix_quitte_et_aucune_icone_n_est_creee() {
        let mut state = ShellState::new();
        assert_eq!(state.apply_settings(settings(false, "fr")), TrayChange::Keep);
        assert_eq!(state.on_close_requested(), CloseAction::Quit);
    }

    #[test]
    fn reglage_actif_l_icone_est_creee_puis_la_croix_cache_sans_arreter() {
        let mut state = ShellState::new();
        assert_eq!(state.apply_settings(settings(true, "fr")), TrayChange::Create);
        // Avant que l'icone existe vraiment, la croix quitte : jamais de fenetre cachee sans icone.
        assert_eq!(state.on_close_requested(), CloseAction::Quit);
        state.tray_created();
        assert_eq!(state.on_close_requested(), CloseAction::HideToTray);
        // La croix cache : l'etat ne change pas, le serveur continue (aucune action d'arret).
        assert_eq!(state.on_close_requested(), CloseAction::HideToTray);
        assert!(state.tray_present());
    }

    #[test]
    fn une_icone_qui_n_a_pas_pu_etre_creee_laisse_la_croix_quitter() {
        let mut state = ShellState::new();
        assert_eq!(state.apply_settings(settings(true, "fr")), TrayChange::Create);
        state.tray_failed();
        assert_eq!(state.on_close_requested(), CloseAction::Quit);
        // Un nouvel envoi des reglages retente la creation.
        assert_eq!(state.apply_settings(settings(true, "fr")), TrayChange::Create);
    }

    #[test]
    fn couper_le_reglage_retire_l_icone_et_la_croix_quitte_de_nouveau() {
        let mut state = ShellState::new();
        state.apply_settings(settings(true, "fr"));
        state.tray_created();
        assert_eq!(state.apply_settings(settings(false, "fr")), TrayChange::Remove);
        // Tant que l'icone n'est pas retiree, le reglage coupe decide deja : la croix quitte.
        assert_eq!(state.on_close_requested(), CloseAction::Quit);
        state.tray_removed();
        assert!(!state.tray_present());
        assert_eq!(state.apply_settings(settings(false, "fr")), TrayChange::Keep);
    }

    #[test]
    fn un_envoi_identique_ne_fait_rien_et_un_changement_de_langue_refait_l_icone() {
        let mut state = ShellState::new();
        state.apply_settings(settings(true, "fr"));
        state.tray_created();
        assert_eq!(state.apply_settings(settings(true, "fr")), TrayChange::Keep);
        assert_eq!(state.apply_settings(settings(true, "en")), TrayChange::Recreate);
        state.tray_created();
        assert_eq!(state.apply_settings(settings(true, "en")), TrayChange::Keep);
    }

    #[test]
    fn le_menu_ouvre_synchronise_et_quitte_par_la_sequence_d_arret() {
        let state = ShellState::new();
        assert_eq!(state.on_tray_command(TrayCommand::Open), ShellAction::ShowWindow);
        assert_eq!(state.on_tray_command(TrayCommand::SyncNow), ShellAction::RequestSync);
        // « Quitter » est l'action Quit : l'application l'execute par le meme chemin que la croix sans icone
        // (fermeture de stdin, attente de `stopped`).
        assert_eq!(state.on_tray_command(TrayCommand::Quit), ShellAction::Quit);
    }

    #[test]
    fn les_identifiants_du_menu_se_rangent_en_commandes() {
        assert_eq!(command_for_menu_id(MENU_OPEN), Some(TrayCommand::Open));
        assert_eq!(command_for_menu_id(MENU_SYNC), Some(TrayCommand::SyncNow));
        assert_eq!(command_for_menu_id(MENU_QUIT), Some(TrayCommand::Quit));
        assert_eq!(command_for_menu_id("autre"), None);
    }

    #[test]
    fn les_libelles_du_menu_existent_dans_chaque_langue_et_le_repli_est_la_langue_par_defaut() {
        for code in catalogue::LOCALES {
            let labels = tray_labels(code);
            assert!(
                !labels.open.is_empty() && !labels.sync.is_empty() && !labels.quit.is_empty(),
                "{code}"
            );
        }
        assert_eq!(tray_labels("zz"), tray_labels(catalogue::DEFAULT_LOCALE));
        assert_eq!(tray_labels(""), tray_labels(catalogue::DEFAULT_LOCALE));
        // Deux langues ne partagent pas leurs libelles (un catalogue copie-colle serait refuse).
        let codes: Vec<&str> = catalogue::LOCALES.to_vec();
        if codes.len() > 1 {
            assert_ne!(tray_labels(codes[0]).open, tray_labels(codes[1]).open);
        }
    }
}
