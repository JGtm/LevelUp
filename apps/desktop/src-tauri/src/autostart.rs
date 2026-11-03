//! Le demarrage avec Windows (ADR 0045, lot L21.d) : la decision, sans registre ni tauri.
//!
//! Le reglage `start_with_windows` (defaut non) est annonce par le serveur ; la coquille l'applique par le
//! plugin d'autostart, qui ecrit une entree PAR UTILISATEUR (`HKCU\...\Run`, jamais `HKLM`) lancant
//! l'application avec `--minimized`. [`reconcile`] ramene l'etat du systeme a l'etat voulu, une seule fois par
//! annonce, et ne touche JAMAIS au registre quand l'etat est deja le bon : un reglage coupe n'ecrit ni ne
//! supprime rien tant qu'aucune entree n'existe.
//!
//! Deux faits distincts : l'entree est-elle PRESENTE (valeur `Run`) et est-elle APPROUVEE (valeur de
//! `StartupApproved\Run`, que le gestionnaire des taches de Windows ecrit quand l'utilisateur desactive un
//! element de demarrage) ? Une entree presente mais desactivee par l'utilisateur est son choix : un reglage
//! actif ne la reecrit jamais ([`AutostartOutcome::LeftDisabled`]) ; un reglage coupe la supprime quel que soit
//! son etat d'approbation.
//!
//! Le plugin est derriere [`AutostartControl`] : les tests branchent un faux, aucune entree reelle n'est creee.

/// Le nom de l'entree de demarrage (valeur de la cle `Run` de l'utilisateur courant). Pose explicitement : le
/// plugin prendrait sinon le nom du paquet Cargo, et une entree qu'on veut retrouver (a la main, ou par un
/// futur crochet de desinstallation) doit porter un nom connu.
pub const ENTRY_NAME: &str = "LevelUp";

/// L'etat de l'entree de demarrage de l'utilisateur courant.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum EntryState {
    /// Aucune valeur `Run` : l'application ne demarre pas avec Windows.
    Absent,
    /// Valeur `Run` presente et approuvee : elle demarre avec Windows.
    Approved,
    /// Valeur `Run` presente mais desactivee par l'utilisateur (gestionnaire des taches) : elle ne demarre pas.
    DisabledByUser,
}

/// L'etat de l'entree d'apres la presence de sa valeur `Run` et le contenu brut de sa valeur
/// `StartupApproved\Run` (`None` : pas de valeur, donc approuvee, comme Windows le comprend). Le premier octet
/// porte l'etat : pair (`0x02`, `0x06`) = active, impair (`0x03`, `0x07`) = desactivee, suivi de la date de
/// desactivation.
pub fn entry_state(present: bool, approval: Option<&[u8]>) -> EntryState {
    if !present {
        return EntryState::Absent;
    }
    match approval.and_then(|bytes| bytes.first()) {
        Some(first) if first & 1 == 1 => EntryState::DisabledByUser,
        _ => EntryState::Approved,
    }
}

/// Ce qui lit et ecrit l'entree de demarrage (le plugin d'autostart dans l'application).
pub trait AutostartControl {
    /// L'etat de l'entree de demarrage (presence ET approbation).
    fn entry_state(&self) -> Result<EntryState, String>;
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
    /// Le reglage est actif mais l'utilisateur a desactive l'entree dans le gestionnaire des taches : son choix
    /// est respecte, rien n'a ete ecrit.
    LeftDisabled,
    /// Le systeme a refuse ou n'a pas repondu ; le texte est celui du systeme (pour le journal).
    Failed(String),
}

/// Amene l'entree de demarrage a l'etat `wanted`. Lit d'abord : on n'ecrit que pour changer quelque chose, et
/// jamais pour contredire un choix de l'utilisateur (entree desactivee dans le gestionnaire des taches).
pub fn reconcile(wanted: bool, control: &dyn AutostartControl) -> AutostartOutcome {
    let current = match control.entry_state() {
        Ok(current) => current,
        Err(e) => return AutostartOutcome::Failed(format!("lecture de l'entree : {e}")),
    };
    match (wanted, current) {
        (true, EntryState::Absent) => match control.enable() {
            Ok(()) => AutostartOutcome::Enabled,
            Err(e) => AutostartOutcome::Failed(format!("creation de l'entree : {e}")),
        },
        (true, EntryState::DisabledByUser) => AutostartOutcome::LeftDisabled,
        (false, EntryState::Approved | EntryState::DisabledByUser) => match control.disable() {
            Ok(()) => AutostartOutcome::Disabled,
            Err(e) => AutostartOutcome::Failed(format!("suppression de l'entree : {e}")),
        },
        (true, EntryState::Approved) | (false, EntryState::Absent) => AutostartOutcome::Unchanged,
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::cell::{Cell, RefCell};

    /// Un registre simule : l'etat de l'entree et la liste des ecritures.
    struct FakeControl {
        state: Cell<EntryState>,
        writes: RefCell<Vec<&'static str>>,
        fail_read: bool,
        fail_write: bool,
    }

    impl FakeControl {
        fn new(present: bool) -> Self {
            Self::in_state(if present {
                EntryState::Approved
            } else {
                EntryState::Absent
            })
        }

        fn in_state(state: EntryState) -> Self {
            Self {
                state: Cell::new(state),
                writes: RefCell::new(Vec::new()),
                fail_read: false,
                fail_write: false,
            }
        }
    }

    impl AutostartControl for FakeControl {
        fn entry_state(&self) -> Result<EntryState, String> {
            if self.fail_read {
                return Err("registre illisible".into());
            }
            Ok(self.state.get())
        }
        fn enable(&self) -> Result<(), String> {
            if self.fail_write {
                return Err("acces refuse".into());
            }
            self.writes.borrow_mut().push("enable");
            self.state.set(EntryState::Approved);
            Ok(())
        }
        fn disable(&self) -> Result<(), String> {
            if self.fail_write {
                return Err("acces refuse".into());
            }
            self.writes.borrow_mut().push("disable");
            self.state.set(EntryState::Absent);
            Ok(())
        }
    }

    #[test]
    fn l_etat_se_lit_de_la_presence_et_du_premier_octet_d_approbation() {
        let enabled = [0x02, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0];
        let disabled = [0x03, 0, 0, 0, 0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0x01];
        assert_eq!(entry_state(false, None), EntryState::Absent);
        assert_eq!(
            entry_state(false, Some(&disabled)),
            EntryState::Absent,
            "sans valeur Run, rien ne demarre"
        );
        assert_eq!(
            entry_state(true, None),
            EntryState::Approved,
            "pas de valeur d'approbation : approuvee"
        );
        assert_eq!(entry_state(true, Some(&enabled)), EntryState::Approved);
        assert_eq!(entry_state(true, Some(&[0x06])), EntryState::Approved);
        assert_eq!(entry_state(true, Some(&disabled)), EntryState::DisabledByUser);
        assert_eq!(entry_state(true, Some(&[0x07])), EntryState::DisabledByUser);
        assert_eq!(
            entry_state(true, Some(&[])),
            EntryState::Approved,
            "valeur vide : approuvee"
        );
    }

    #[test]
    fn reglage_actif_et_entree_desactivee_par_l_utilisateur_n_ecrit_rien() {
        let control = FakeControl::in_state(EntryState::DisabledByUser);
        assert_eq!(reconcile(true, &control), AutostartOutcome::LeftDisabled);
        assert_eq!(
            reconcile(true, &control),
            AutostartOutcome::LeftDisabled,
            "a chaque annonce"
        );
        assert!(
            control.writes.borrow().is_empty(),
            "un reglage actif ne reecrit jamais un choix de l'utilisateur"
        );
    }

    #[test]
    fn reglage_coupe_supprime_l_entree_meme_desactivee_par_l_utilisateur() {
        let control = FakeControl::in_state(EntryState::DisabledByUser);
        assert_eq!(reconcile(false, &control), AutostartOutcome::Disabled);
        assert_eq!(*control.writes.borrow(), vec!["disable"]);
        assert_eq!(reconcile(false, &control), AutostartOutcome::Unchanged);
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
