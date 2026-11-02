//! Le demarrage avec Windows (ADR 0045, lot L21.d) : la decision, sans registre ni tauri.
//!
//! Le reglage `start_with_windows` (defaut non) est annonce par le serveur ; la coquille l'applique par le
//! plugin d'autostart, qui ecrit une entree PAR UTILISATEUR (`HKCU\...\Run`, jamais `HKLM`) lancant
//! l'application avec `--minimized`. [`reconcile`] ramene l'etat du systeme a l'etat voulu, une seule fois par
//! annonce, et ne touche JAMAIS au registre quand l'etat est deja le bon : un reglage coupe n'ecrit ni ne
//! supprime rien tant qu'aucune entree n'existe.
//!
//! Le plugin est derriere [`AutostartControl`] : les tests branchent un faux, aucune entree reelle n'est creee.

/// Le nom de l'entree de demarrage (valeur de la cle `Run` de l'utilisateur courant). Pose explicitement : le
/// plugin prendrait sinon le nom du paquet Cargo, et une entree qu'on veut retrouver (a la main, ou par un
/// futur crochet de desinstallation) doit porter un nom connu.
pub const ENTRY_NAME: &str = "LevelUp";

/// Ce qui lit et ecrit l'entree de demarrage (le plugin d'autostart dans l'application).
pub trait AutostartControl {
    /// L'entree de demarrage existe-t-elle ?
    fn is_enabled(&self) -> Result<bool, String>;
    /// Cree l'entree de demarrage de l'utilisateur courant.
    fn enable(&self) -> Result<(), String>;
    /// Supprime l'entree de demarrage de l'utilisateur courant.
    fn disable(&self) -> Result<(), String>;
}

/// Ce que `reconcile` a fait.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum AutostartOutcome {
    /// L'entree a ete creee.
    Enabled,
    /// L'entree a ete supprimee.
    Disabled,
    /// L'etat etait deja le bon : rien n'a ete ecrit.
    Unchanged,
    /// Le systeme a refuse ou n'a pas repondu ; le texte est celui du systeme (pour le journal).
    Failed(String),
}

/// Amene l'entree de demarrage a l'etat `wanted`. Lit d'abord : on n'ecrit que pour changer quelque chose.
pub fn reconcile(wanted: bool, control: &dyn AutostartControl) -> AutostartOutcome {
    let current = match control.is_enabled() {
        Ok(current) => current,
        Err(e) => return AutostartOutcome::Failed(format!("lecture de l'entree : {e}")),
    };
    match (wanted, current) {
        (true, false) => match control.enable() {
            Ok(()) => AutostartOutcome::Enabled,
            Err(e) => AutostartOutcome::Failed(format!("creation de l'entree : {e}")),
        },
        (false, true) => match control.disable() {
            Ok(()) => AutostartOutcome::Disabled,
            Err(e) => AutostartOutcome::Failed(format!("suppression de l'entree : {e}")),
        },
        _ => AutostartOutcome::Unchanged,
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::cell::{Cell, RefCell};

    /// Un registre simule : l'etat de l'entree et la liste des ecritures.
    struct FakeControl {
        present: Cell<bool>,
        writes: RefCell<Vec<&'static str>>,
        fail_read: bool,
        fail_write: bool,
    }

    impl FakeControl {
        fn new(present: bool) -> Self {
            Self {
                present: Cell::new(present),
                writes: RefCell::new(Vec::new()),
                fail_read: false,
                fail_write: false,
            }
        }
    }

    impl AutostartControl for FakeControl {
        fn is_enabled(&self) -> Result<bool, String> {
            if self.fail_read {
                return Err("registre illisible".into());
            }
            Ok(self.present.get())
        }
        fn enable(&self) -> Result<(), String> {
            if self.fail_write {
                return Err("acces refuse".into());
            }
            self.writes.borrow_mut().push("enable");
            self.present.set(true);
            Ok(())
        }
        fn disable(&self) -> Result<(), String> {
            if self.fail_write {
                return Err("acces refuse".into());
            }
            self.writes.borrow_mut().push("disable");
            self.present.set(false);
            Ok(())
        }
    }

    #[test]
    fn reglage_coupe_sans_entree_rien_n_est_ecrit() {
        let control = FakeControl::new(false);
        assert_eq!(reconcile(false, &control), AutostartOutcome::Unchanged);
        assert!(
            control.writes.borrow().is_empty(),
            "le defaut (coupe) ne touche jamais au registre"
        );
    }

    #[test]
    fn activer_cree_l_entree_une_seule_fois() {
        let control = FakeControl::new(false);
        assert_eq!(reconcile(true, &control), AutostartOutcome::Enabled);
        assert_eq!(reconcile(true, &control), AutostartOutcome::Unchanged);
        assert_eq!(*control.writes.borrow(), vec!["enable"]);
    }

    #[test]
    fn couper_supprime_l_entree_une_seule_fois() {
        let control = FakeControl::new(true);
        assert_eq!(reconcile(false, &control), AutostartOutcome::Disabled);
        assert_eq!(reconcile(false, &control), AutostartOutcome::Unchanged);
        assert_eq!(*control.writes.borrow(), vec!["disable"]);
    }

    #[test]
    fn une_entree_deja_la_et_voulue_n_est_pas_reecrite() {
        let control = FakeControl::new(true);
        assert_eq!(reconcile(true, &control), AutostartOutcome::Unchanged);
        assert!(control.writes.borrow().is_empty());
    }

    #[test]
    fn un_refus_du_systeme_est_rendu_sans_paniquer() {
        let mut control = FakeControl::new(false);
        control.fail_write = true;
        assert_eq!(
            reconcile(true, &control),
            AutostartOutcome::Failed("creation de l'entree : acces refuse".into())
        );
        let mut present = FakeControl::new(true);
        present.fail_write = true;
        assert_eq!(
            reconcile(false, &present),
            AutostartOutcome::Failed("suppression de l'entree : acces refuse".into())
        );
        let mut unreadable = FakeControl::new(false);
        unreadable.fail_read = true;
        assert_eq!(
            reconcile(true, &unreadable),
            AutostartOutcome::Failed("lecture de l'entree : registre illisible".into())
        );
        assert!(
            unreadable.writes.borrow().is_empty(),
            "illisible : on n'ecrit pas a l'aveugle"
        );
    }
}
