//! Les notifications Windows relayees depuis la cloche (ADR 0045, lot L21.b) : les decisions, sans fenetre.
//!
//! Le serveur decide s'il faut relayer (reglage `native_notifications`, categories retenues) et compose
//! le texte dans la langue du joueur ; la coquille decide seulement si l'utilisateur la voit deja. Une
//! notification systeme qui doublerait la cloche d'une fenetre ouverte au premier plan serait du bruit :
//! elle n'est montree que fenetre masquee (zone de notification), reduite ou sans le focus.
//!
//! Le plugin de notification est derriere [`NativeNotifier`] : les tests branchent un faux qui enregistre.

use crate::events::Toast;
use crate::logfile::{LogSink, Source};

/// Ou en est la fenetre principale du point de vue de l'utilisateur.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum WindowState {
    /// Visible, non reduite et avec le focus : la cloche est sous ses yeux.
    Foreground,
    /// Visible mais sans le focus (une autre application est devant).
    Background,
    /// Masquee (zone de notification) ou reduite dans la barre des taches.
    Hidden,
}

impl WindowState {
    /// Range les trois questions que la fenetre sait repondre : visible, reduite, avec le focus.
    pub fn from_flags(visible: bool, minimized: bool, focused: bool) -> Self {
        if !visible || minimized {
            Self::Hidden
        } else if focused {
            Self::Foreground
        } else {
            Self::Background
        }
    }
}

/// Une notification n'est montree que si la fenetre n'est pas au premier plan.
pub fn should_show(window: WindowState) -> bool {
    window != WindowState::Foreground
}

/// Ce qui montre une notification systeme (le plugin de notification dans l'application).
pub trait NativeNotifier {
    /// Montre la notification ; une erreur est rendue en texte pour le journal.
    fn show(&mut self, notice: &Toast) -> Result<(), String>;
}

/// Ce qu'est devenue une notification.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Delivery {
    Shown,
    /// Retenue : la fenetre est au premier plan.
    Suppressed,
    Failed,
}

/// Applique la decision et journalise l'issue (le type de la notification, jamais son texte).
pub fn deliver(window: WindowState, notice: &Toast, notifier: &mut dyn NativeNotifier, log: &LogSink) -> Delivery {
    if !should_show(window) {
        log.write_line(
            Source::Shell,
            &format!(
                "event=native_notification kind={} outcome=suppressed reason=foreground",
                notice.kind
            ),
        );
        return Delivery::Suppressed;
    }
    match notifier.show(notice) {
        Ok(()) => {
            log.write_line(
                Source::Shell,
                &format!(
                    "event=native_notification kind={} outcome=shown window={window:?}",
                    notice.kind
                ),
            );
            Delivery::Shown
        }
        Err(e) => {
            log.write_line(
                Source::Shell,
                &format!("event=native_notification kind={} outcome=failed err={e}", notice.kind),
            );
            Delivery::Failed
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::logfile::MAX_LOG_BYTES;

    #[derive(Default)]
    struct Recorder {
        shown: Vec<Toast>,
        fail: bool,
    }

    impl NativeNotifier for Recorder {
        fn show(&mut self, notice: &Toast) -> Result<(), String> {
            if self.fail {
                return Err("toast refuse".into());
            }
            self.shown.push(notice.clone());
            Ok(())
        }
    }

    fn notice() -> Toast {
        Toast {
            title: "Synchronisation terminee".into(),
            body: "Tes dernieres parties sont a jour.".into(),
            kind: "sync".into(),
        }
    }

    fn sink(name: &str) -> (LogSink, std::path::PathBuf) {
        let path = std::env::temp_dir().join(format!("native-notice-{name}-{}.log", std::process::id()));
        let _ = std::fs::remove_file(&path);
        (LogSink::open(path.clone(), MAX_LOG_BYTES).expect("journal"), path)
    }

    #[test]
    fn l_etat_de_la_fenetre_se_deduit_de_ses_trois_drapeaux() {
        assert_eq!(WindowState::from_flags(true, false, true), WindowState::Foreground);
        assert_eq!(WindowState::from_flags(true, false, false), WindowState::Background);
        assert_eq!(WindowState::from_flags(true, true, true), WindowState::Hidden);
        assert_eq!(WindowState::from_flags(false, false, false), WindowState::Hidden);
        assert_eq!(WindowState::from_flags(false, false, true), WindowState::Hidden);
    }

    #[test]
    fn montree_seulement_si_la_fenetre_n_est_pas_au_premier_plan() {
        assert!(!should_show(WindowState::Foreground));
        assert!(should_show(WindowState::Background));
        assert!(should_show(WindowState::Hidden));
    }

    #[test]
    fn fenetre_masquee_ou_sans_focus_la_notification_est_montree_telle_quelle() {
        for window in [WindowState::Hidden, WindowState::Background] {
            let (log, path) = sink("shown");
            let mut notifier = Recorder::default();
            assert_eq!(deliver(window, &notice(), &mut notifier, &log), Delivery::Shown);
            assert_eq!(notifier.shown, vec![notice()], "texte du serveur, sans retouche");
            let journal = std::fs::read_to_string(&path).unwrap();
            assert!(
                journal.contains("event=native_notification kind=sync outcome=shown"),
                "{journal}"
            );
            assert!(
                !journal.contains("Synchronisation terminee"),
                "le texte n'est jamais journalise"
            );
        }
    }

    #[test]
    fn fenetre_au_premier_plan_la_notification_est_retenue() {
        let (log, path) = sink("suppressed");
        let mut notifier = Recorder::default();
        assert_eq!(
            deliver(WindowState::Foreground, &notice(), &mut notifier, &log),
            Delivery::Suppressed
        );
        assert!(notifier.shown.is_empty());
        let journal = std::fs::read_to_string(&path).unwrap();
        assert!(journal.contains("outcome=suppressed reason=foreground"), "{journal}");
    }

    #[test]
    fn un_echec_du_plugin_est_journalise_et_ne_panique_pas() {
        let (log, path) = sink("failed");
        let mut notifier = Recorder {
            fail: true,
            ..Recorder::default()
        };
        assert_eq!(
            deliver(WindowState::Hidden, &notice(), &mut notifier, &log),
            Delivery::Failed
        );
        let journal = std::fs::read_to_string(&path).unwrap();
        assert!(journal.contains("outcome=failed err=toast refuse"), "{journal}");
    }
}
