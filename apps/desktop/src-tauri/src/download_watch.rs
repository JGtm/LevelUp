//! L'echeance du telechargement de l'installeur : sans elle un flux bloque garderait le verrou de
//! verification (`UpdateBusy`) jusqu'a la fermeture de l'application, et plus aucune verification ne
//! partirait.
//!
//! Une echeance totale courte couperait un telechargement lent mais vivant (l'installeur pese
//! environ 200 Mo). La regle est donc une echeance d'INACTIVITE ([`STALL_LIMIT`] : aucun octet recu),
//! plus un plafond de duree tres large ([`MAX_DURATION`]) contre un flux qui goutte sans jamais
//! s'arreter. La decision ([`verdict`]) est pure ; [`Progress`] n'est que le compteur que le
//! rappel de progression du plugin alimente.

use std::sync::atomic::{AtomicU64, Ordering};
use std::sync::{Arc, Mutex};
use std::time::{Duration, Instant};

/// Duree sans aucun octet recu apres laquelle le telechargement est abandonne.
pub const STALL_LIMIT: Duration = Duration::from_secs(60);
/// Duree totale maximale d'un telechargement, quelle que soit sa progression (200 Mo a 0,5 Mo/s).
pub const MAX_DURATION: Duration = Duration::from_secs(60 * 60);
/// Intervalle de surveillance.
pub const POLL_INTERVAL: Duration = Duration::from_millis(250);

/// Ce que la surveillance decide a un instant donne.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Verdict {
    /// Le telechargement progresse (ou vient de commencer) : attendre.
    Wait,
    /// Aucun octet depuis [`STALL_LIMIT`] : abandon.
    Stalled,
    /// [`MAX_DURATION`] depassee : abandon.
    TooLong,
}

impl Verdict {
    /// La raison journalisee (`err=` de `update_download_failed`) ; `None` pour `Wait`.
    pub fn reason(self) -> Option<&'static str> {
        match self {
            Self::Wait => None,
            Self::Stalled => Some("download_stalled"),
            Self::TooLong => Some("download_too_long"),
        }
    }
}

/// La decision, PURE : `elapsed` depuis le debut du telechargement, `idle` depuis le dernier octet
/// recu (ou depuis le debut s'il n'en est venu aucun). Le plafond de duree passe avant l'inactivite.
pub fn verdict(elapsed: Duration, idle: Duration) -> Verdict {
    if elapsed >= MAX_DURATION {
        Verdict::TooLong
    } else if idle >= STALL_LIMIT {
        Verdict::Stalled
    } else {
        Verdict::Wait
    }
}

/// Le compteur de progression, partage entre le rappel du plugin (qui y ecrit) et la surveillance
/// (qui le lit).
#[derive(Clone)]
pub struct Progress {
    start: Instant,
    /// Millisecondes depuis `start` au dernier octet recu.
    last_ms: Arc<AtomicU64>,
}

impl Default for Progress {
    fn default() -> Self {
        Self::new()
    }
}

impl Progress {
    pub fn new() -> Self {
        Self {
            start: Instant::now(),
            last_ms: Arc::new(AtomicU64::new(0)),
        }
    }

    /// Note qu'un bloc d'octets vient d'arriver.
    pub fn record(&self) {
        let ms = u64::try_from(self.start.elapsed().as_millis()).unwrap_or(u64::MAX);
        self.last_ms.store(ms, Ordering::Relaxed);
    }

    /// Le verdict a l'instant present.
    pub fn check(&self) -> Verdict {
        let elapsed = self.start.elapsed();
        let last = Duration::from_millis(self.last_ms.load(Ordering::Relaxed));
        verdict(elapsed, elapsed.saturating_sub(last))
    }
}

/// Ou la tache de telechargement depose son resultat pour la surveillance.
pub type Slot<T> = Arc<Mutex<Option<Result<T, String>>>>;

/// La boucle de surveillance : a chaque tour, lit D'ABORD le resultat de la tache (un telechargement
/// fini a l'instant ou le verdict tombe est rendu, pas abandonne), puis demande le verdict ; sur un
/// abandon appelle `abort` UNE fois et rend la raison en erreur. `pause` attend entre deux tours
/// (le fil dort `POLL_INTERVAL`). Les entrees sont des fermetures pour que la boucle se teste sans
/// horloge ni reseau.
pub fn watch_download<T>(
    slot: &Mutex<Option<Result<T, String>>>,
    mut verdict_now: impl FnMut() -> Verdict,
    abort: impl FnOnce(),
    mut pause: impl FnMut(),
) -> Result<T, String> {
    loop {
        if let Some(result) = slot.lock().unwrap_or_else(|p| p.into_inner()).take() {
            return result;
        }
        if let Some(reason) = verdict_now().reason() {
            abort();
            return Err(reason.to_string());
        }
        pause();
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::cell::Cell;

    const SECOND: Duration = Duration::from_secs(1);

    #[test]
    fn un_telechargement_qui_progresse_attend() {
        assert_eq!(verdict(10 * SECOND, SECOND), Verdict::Wait);
        assert_eq!(verdict(Duration::ZERO, Duration::ZERO), Verdict::Wait);
    }

    #[test]
    fn un_telechargement_lent_mais_vivant_n_est_pas_coupe() {
        // Cinquante minutes de transfert, un octet il y a quelques secondes.
        assert_eq!(verdict(50 * 60 * SECOND, 5 * SECOND), Verdict::Wait);
    }

    #[test]
    fn soixante_secondes_sans_octet_abandonnent() {
        assert_eq!(
            verdict(2 * STALL_LIMIT, STALL_LIMIT - Duration::from_millis(1)),
            Verdict::Wait
        );
        assert_eq!(verdict(2 * STALL_LIMIT, STALL_LIMIT), Verdict::Stalled);
    }

    #[test]
    fn un_flux_qui_ne_donne_jamais_rien_est_abandonne_des_le_depart() {
        assert_eq!(verdict(STALL_LIMIT, STALL_LIMIT), Verdict::Stalled);
    }

    #[test]
    fn le_plafond_de_duree_passe_avant_l_inactivite() {
        assert_eq!(verdict(MAX_DURATION, Duration::ZERO), Verdict::TooLong);
        assert_eq!(verdict(MAX_DURATION, STALL_LIMIT), Verdict::TooLong);
    }

    #[test]
    fn chaque_abandon_a_une_raison_nommee() {
        assert_eq!(Verdict::Wait.reason(), None);
        assert_eq!(Verdict::Stalled.reason(), Some("download_stalled"));
        assert_eq!(Verdict::TooLong.reason(), Some("download_too_long"));
    }

    #[test]
    fn le_compteur_neuf_attend() {
        let progress = Progress::new();
        progress.record();
        assert_eq!(progress.check(), Verdict::Wait);
    }

    fn empty_slot() -> Slot<u32> {
        Arc::default()
    }

    fn deposit(slot: &Slot<u32>, result: Result<u32, String>) {
        *slot.lock().unwrap() = Some(result);
    }

    #[test]
    fn la_boucle_rend_le_resultat_arrive_avant_le_verdict() {
        // Le verdict dirait « abandon » : le resultat deja la passe d'abord, aucun abandon.
        let slot = empty_slot();
        deposit(&slot, Ok(7));
        let aborted = Cell::new(0);
        let out = watch_download(&slot, || Verdict::Stalled, || aborted.set(aborted.get() + 1), || {});
        assert_eq!(out, Ok(7));
        assert_eq!(aborted.get(), 0, "un telechargement fini ne doit pas etre abandonne");
    }

    #[test]
    fn la_boucle_attend_tant_que_le_verdict_attend_puis_rend_le_resultat() {
        let slot = empty_slot();
        let pauses = Cell::new(0);
        let out = watch_download(
            &slot,
            || Verdict::Wait,
            || panic!("aucun abandon attendu"),
            || {
                pauses.set(pauses.get() + 1);
                if pauses.get() == 3 {
                    deposit(&slot, Ok(42));
                }
            },
        );
        assert_eq!(out, Ok(42));
        assert_eq!(pauses.get(), 3);
    }

    #[test]
    fn un_flux_inactif_est_abandonne_une_fois_avec_sa_raison() {
        let slot = empty_slot();
        let aborted = Cell::new(0);
        let out = watch_download(&slot, || Verdict::Stalled, || aborted.set(aborted.get() + 1), || {});
        assert_eq!(out, Err("download_stalled".to_string()));
        assert_eq!(aborted.get(), 1);
    }

    #[test]
    fn le_plafond_de_duree_abandonne_apres_quelques_tours() {
        let slot = empty_slot();
        let ticks = Cell::new(0);
        let aborted = Cell::new(false);
        let out = watch_download(
            &slot,
            || {
                ticks.set(ticks.get() + 1);
                if ticks.get() > 2 {
                    Verdict::TooLong
                } else {
                    Verdict::Wait
                }
            },
            || aborted.set(true),
            || {},
        );
        assert_eq!(out, Err("download_too_long".to_string()));
        assert!(aborted.get());
    }

    #[test]
    fn l_erreur_de_la_tache_est_rendue_telle_quelle() {
        let slot = empty_slot();
        deposit(&slot, Err("signature invalide".to_string()));
        let out = watch_download(&slot, || Verdict::Wait, || panic!("aucun abandon"), || {});
        assert_eq!(out, Err("signature invalide".to_string()));
    }
}
