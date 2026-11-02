//! Les decisions pures de la mise a jour apres l'arret du serveur : l'installeur ne part que si
//! l'arret a ete propre ([`install_allowed`], [`SkipReason`]), et les messages que la coquille montre
//! dans une boite native ([`Notice`], [`notice_text`], [`Notifier`]).
//!
//! Re-exportes par `updater` : les appelants ecrivent `updater::Notice`, `updater::install_allowed`.

use crate::catalogue;
use crate::supervisor::Outcome;

/// Pourquoi l'installation ne part pas : l'arret du serveur n'a pas ete propre.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum SkipReason {
    /// Le kill a echoue : le serveur peut tourner encore.
    KillFailed,
    /// La coquille a du tuer le serveur (budget d'arret depasse).
    Killed,
    /// Le serveur a annonce `stopped` avec `clean=false`.
    Unclean,
    /// Aucun `stopped` n'est arrive.
    NoStopped,
}

impl SkipReason {
    /// Le nom journalise (`reason=` de l'evenement `update_skipped_unclean_stop`).
    pub fn name(self) -> &'static str {
        match self {
            Self::KillFailed => "kill_failed",
            Self::Killed => "killed",
            Self::Unclean => "unclean",
            Self::NoStopped => "no_stopped",
        }
    }
}

/// La decision d'installer, PURE : l'installeur ne part que si `stopped` est arrive avec `clean=true`
/// ET que le serveur n'a pas ete tue. Un installeur lance sur un serveur qui vit encore (ou dont la
/// base n'a pas ete fermee proprement) remplacerait l'executable sous ses pieds.
pub fn install_allowed(outcome: Outcome) -> Result<(), SkipReason> {
    if outcome.kill_failed {
        Err(SkipReason::KillFailed)
    } else if outcome.killed {
        Err(SkipReason::Killed)
    } else {
        match outcome.stopped_clean {
            Some(true) => Ok(()),
            Some(false) => Err(SkipReason::Unclean),
            None => Err(SkipReason::NoStopped),
        }
    }
}

/// Un message que la coquille montre dans une boite native : apres l'arret du serveur quand la mise a
/// jour ne s'est pas faite, ou en reponse a une verification demandee par l'utilisateur.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Notice {
    /// Arret du serveur non propre : la mise a jour est repoussee au prochain lancement.
    SkippedUnclean,
    /// L'installeur a echoue : l'application se ferme sans installation.
    InstallFailed,
    /// Verification demandee : la version installee est la derniere.
    UpToDate,
    /// Verification demandee : le serveur de mises a jour n'a pas repondu (hors ligne, indisponible).
    CheckFailed,
    /// Verification demandee : la mise a jour n'est pas branchee dans ce build (developpement).
    Unavailable,
    /// Verification demandee pendant qu'une autre est en cours.
    Busy,
}

impl Notice {
    /// Tous les messages, pour les tests de completude.
    pub const ALL: [Notice; 6] = [
        Self::SkippedUnclean,
        Self::InstallFailed,
        Self::UpToDate,
        Self::CheckFailed,
        Self::Unavailable,
        Self::Busy,
    ];

    fn message_key(self) -> &'static str {
        match self {
            Self::SkippedUnclean => "update_skipped_unclean",
            Self::InstallFailed => "update_install_failed",
            Self::UpToDate => "update_none",
            Self::CheckFailed => "update_check_failed",
            Self::Unavailable => "update_unavailable_debug",
            Self::Busy => "update_busy",
        }
    }

    /// Vrai pour une simple information, faux pour un message qui previent d'un probleme.
    pub fn is_informational(self) -> bool {
        matches!(self, Self::UpToDate | Self::Unavailable | Self::Busy)
    }
}

/// Les textes d'une boite d'information, dans la langue de la coquille.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct NoticeText {
    pub title: String,
    pub message: String,
}

/// Version de l'application installee (celle du paquet : `Cargo.toml` fait foi, A9).
pub const CURRENT_VERSION: &str = env!("CARGO_PKG_VERSION");

/// Compose le texte d'un message depuis le catalogue de repli ; `{version}` vaut la version installee.
pub fn notice_text(locale: &str, notice: Notice) -> NoticeText {
    let text = |key: &str| catalogue::text(locale, key).unwrap_or_default().to_string();
    NoticeText {
        title: text("update_title"),
        message: text(notice.message_key()).replace("{version}", CURRENT_VERSION),
    }
}

/// Une boite d'information native (bloquante). Le serveur est deja arrete quand elle s'affiche : le
/// texte vient du catalogue compile dans la coquille.
pub trait Notifier: Send + 'static {
    fn show(&mut self, notice: Notice);
}
