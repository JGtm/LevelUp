//! La verification de mise a jour demandee par l'utilisateur (bouton « Rechercher une mise a jour »
//! de la fenetre « A propos »).
//!
//! Meme sequence que celle du lancement (`updater::run_flow` : proposee, jamais imposee, installee
//! apres l'arret propre du serveur), avec deux differences : l'issue sans mise a jour est annoncee
//! dans une boite, et une seule verification tourne a la fois ([`UpdateBusy`]) : une demande qui
//! arrive pendant qu'une autre est en cours est annoncee, pas empilee.

use std::sync::Arc;
use std::sync::atomic::{AtomicBool, Ordering};

use crate::logfile::{LogSink, Source};
use crate::updater::{Notice, Notifier, Prompt, UpdateContext, UpdateFlow, UpdateSource, run_flow};

/// Verrou de la verification en cours : partage entre celle du lancement et celles de l'utilisateur.
#[derive(Clone, Default)]
pub struct UpdateBusy(Arc<AtomicBool>);

/// Tenue tant qu'une verification tourne ; rend le verrou en tombant.
pub struct BusyGuard(Arc<AtomicBool>);

impl UpdateBusy {
    /// Prend le verrou ; `None` si une verification est deja en cours.
    pub fn try_acquire(&self) -> Option<BusyGuard> {
        self.0
            .compare_exchange(false, true, Ordering::SeqCst, Ordering::SeqCst)
            .ok()
            .map(|_| BusyGuard(self.0.clone()))
    }
}

impl Drop for BusyGuard {
    fn drop(&mut self) {
        self.0.store(false, Ordering::SeqCst);
    }
}

/// Une verification demandee par l'utilisateur. Bloquante (boites natives) : a lancer sur son propre fil.
pub fn run_manual_check(
    source: impl UpdateSource,
    prompt: impl Prompt,
    notifier: &mut dyn Notifier,
    busy: &UpdateBusy,
    context: &UpdateContext<'_>,
) {
    let Some(_guard) = busy.try_acquire() else {
        context.log.write_line(Source::Shell, "event=update_check_busy");
        notifier.show(Notice::Busy);
        return;
    };
    context.log.write_line(Source::Shell, "event=update_check_requested");
    run_flow(UpdateFlow::manual(), source, prompt, Some(notifier), context);
}

/// La verification demandee alors que la mise a jour n'est pas branchee dans ce build (build de
/// developpement, cle publique absente) : la raison est dite a l'utilisateur au lieu d'un clic muet.
pub fn report_unavailable(notifier: &mut dyn Notifier, log: &LogSink) {
    log.write_line(Source::Shell, "event=update_check_unavailable");
    notifier.show(Notice::Unavailable);
}
