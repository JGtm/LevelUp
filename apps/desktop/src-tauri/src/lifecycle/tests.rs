//! Tests de la machine d'etats : evenements en entree, actions en sortie, rien n'est lance.

use super::*;

const T: Timeouts = Timeouts {
    startup: Duration::from_secs(120),
    close: Duration::from_secs(45),
    grace: Duration::from_secs(5),
};

fn machine() -> Lifecycle {
    Lifecycle::start(T).0
}

fn listening(port: u16) -> Input {
    Input::Event(SidecarEvent::Listening { port })
}
fn ready(port: u16) -> Input {
    Input::Event(SidecarEvent::Ready { port })
}
fn failed(state: &str) -> Input {
    Input::Event(SidecarEvent::Failed { state: state.into() })
}
fn stopped(clean: bool) -> Input {
    Input::Event(SidecarEvent::Stopped { clean })
}

#[test]
fn le_debut_montre_l_attente_et_arme_le_delai_de_demarrage() {
    let (_, actions) = Lifecycle::start(T);
    assert_eq!(
        actions,
        vec![
            Action::ShowScreen(Screen::Starting),
            Action::Arm(Timer::Startup, T.startup)
        ]
    );
}

#[test]
fn listening_puis_ready_chargent_l_entree_du_serveur() {
    let mut m = machine();
    assert_eq!(m.step(listening(8123)), vec![Action::AnnouncePort { port: 8123 }]);
    assert_eq!(
        m.step(ready(8123)),
        vec![
            Action::Disarm(Timer::Startup),
            Action::AnnouncePort { port: 8123 },
            Action::LoadEntry { port: 8123 }
        ]
    );
}

#[test]
fn ready_sans_listening_est_accepte() {
    let mut m = machine();
    assert!(m.step(ready(9000)).contains(&Action::LoadEntry { port: 9000 }));
}

#[test]
fn un_echec_avec_port_charge_la_page_du_serveur() {
    let mut m = machine();
    m.step(listening(8123));
    assert_eq!(
        m.step(failed("migration_failed")),
        vec![Action::Disarm(Timer::Startup), Action::LoadServerPage { port: 8123 }]
    );
}

#[test]
fn un_echec_sans_port_montre_l_ecran_embarque_de_sa_cause() {
    for (wire, kind) in [
        ("locked", FailureKind::Locked),
        ("migration_failed", FailureKind::MigrationFailed),
        ("schema_newer", FailureKind::SchemaNewer),
        ("port", FailureKind::Port),
    ] {
        let mut m = machine();
        assert_eq!(
            m.step(failed(wire)),
            vec![Action::Disarm(Timer::Startup), Action::ShowScreen(Screen::Failed(kind))]
        );
    }
}

#[test]
fn un_etat_inconnu_sans_port_montre_l_ecran_arrete() {
    let mut m = machine();
    assert_eq!(
        m.step(failed("etat_futur")),
        vec![Action::Disarm(Timer::Startup), Action::ShowScreen(Screen::Stopped)]
    );
}

#[test]
fn la_sortie_du_serveur_apres_un_echec_avec_port_montre_l_ecran_de_sa_cause() {
    let mut m = machine();
    m.step(listening(8123));
    m.step(failed("locked"));
    let actions = m.step(Input::Exited);
    assert!(actions.contains(&Action::ShowScreen(Screen::Failed(FailureKind::Locked))));
    assert!(!m.is_done());
}

#[test]
fn pas_de_ready_dans_le_delai_tue_le_serveur_et_montre_le_delai() {
    let mut m = machine();
    m.step(listening(8123));
    let screen = Screen::Failed(FailureKind::Timeout);
    assert_eq!(
        m.step(Input::TimerFired(Timer::Startup)),
        vec![Action::Kill, Action::ShowScreen(screen)]
    );
    assert!(m.was_killed());
    // La sortie qui suit le kill ne remplace pas l'ecran du delai.
    assert!(!m.step(Input::Exited).iter().any(|a| matches!(a, Action::ShowScreen(_))));
}

#[test]
fn le_delai_de_demarrage_tardif_apres_ready_ne_fait_rien() {
    let mut m = machine();
    m.step(ready(8123));
    assert_eq!(m.step(Input::TimerFired(Timer::Startup)), vec![]);
    assert!(!m.was_killed());
}

#[test]
fn la_sortie_du_processus_sans_stopped_montre_l_ecran_arrete() {
    let mut m = machine();
    m.step(ready(8123));
    let actions = m.step(Input::Exited);
    assert!(actions.contains(&Action::ShowScreen(Screen::Stopped)));
    assert!(!m.is_done());
    // La fermeture qui suit sort directement, il n'y a plus de serveur a attendre.
    assert_eq!(m.step(Input::CloseRequested), vec![Action::ExitApp]);
    assert!(m.is_done());
}

#[test]
fn la_fermeture_ferme_stdin_et_attend_la_sortie_du_processus() {
    let mut m = machine();
    m.step(ready(8123));
    assert_eq!(
        m.step(Input::CloseRequested),
        vec![
            Action::ShowScreen(Screen::Stopping),
            Action::CloseStdin,
            Action::Disarm(Timer::Startup),
            Action::Arm(Timer::Close, T.close),
        ]
    );
    assert_eq!(m.step(Input::Event(SidecarEvent::Stopping)), vec![]);
    assert_eq!(m.step(stopped(true)), vec![Action::Arm(Timer::Grace, T.grace)]);
    assert_eq!(m.stopped_clean(), Some(true));
    let exit = m.step(Input::Exited);
    assert_eq!(exit.last(), Some(&Action::ExitApp));
    assert!(m.is_done() && !m.was_killed());
}

#[test]
fn une_seconde_demande_de_fermeture_est_ignoree() {
    let mut m = machine();
    m.step(ready(8123));
    m.step(Input::CloseRequested);
    assert_eq!(m.step(Input::CloseRequested), vec![]);
}

#[test]
fn le_budget_depasse_tue_le_serveur_puis_sort_apres_la_grace() {
    let mut m = machine();
    m.step(ready(8123));
    m.step(Input::CloseRequested);
    assert_eq!(
        m.step(Input::TimerFired(Timer::Close)),
        vec![Action::Kill, Action::Arm(Timer::Grace, T.grace)]
    );
    assert!(m.was_killed());
    // Le processus sort avant la grace : sortie immediate.
    assert_eq!(m.step(Input::Exited).last(), Some(&Action::ExitApp));
}

#[test]
fn si_le_kill_ne_suffit_pas_la_grace_ordonne_quand_meme_la_sortie() {
    let mut m = machine();
    m.step(ready(8123));
    m.step(Input::CloseRequested);
    m.step(Input::TimerFired(Timer::Close));
    assert_eq!(m.step(Input::TimerFired(Timer::Grace)), vec![Action::ExitApp]);
    assert!(m.is_done());
}

#[test]
fn stopped_sans_sortie_du_processus_finit_par_un_kill_puis_la_sortie() {
    let mut m = machine();
    m.step(ready(8123));
    m.step(Input::CloseRequested);
    m.step(stopped(true));
    assert_eq!(
        m.step(Input::TimerFired(Timer::Grace)),
        vec![Action::Kill, Action::ExitApp]
    );
}

#[test]
fn fermer_pendant_le_demarrage_arrete_le_serveur_sans_attendre_ready() {
    let mut m = machine();
    m.step(listening(8123));
    let actions = m.step(Input::CloseRequested);
    assert!(actions.contains(&Action::CloseStdin) && actions.contains(&Action::Disarm(Timer::Startup)));
    // L'echec annonce pendant la fermeture ne change rien : le processus sort, l'application aussi.
    assert_eq!(m.step(failed("locked")), vec![]);
    assert_eq!(m.step(Input::Exited).last(), Some(&Action::ExitApp));
}

#[test]
fn fermer_apres_un_echec_dont_le_processus_vit_encore_ferme_stdin() {
    let mut m = machine();
    m.step(listening(8123));
    m.step(failed("schema_newer"));
    assert!(m.step(Input::CloseRequested).contains(&Action::CloseStdin));
    assert_eq!(m.step(Input::Exited).last(), Some(&Action::ExitApp));
}

#[test]
fn un_arret_decide_par_le_serveur_montre_l_ecran_de_fermeture_puis_l_arret() {
    let mut m = machine();
    m.step(ready(8123));
    assert_eq!(
        m.step(Input::Event(SidecarEvent::Stopping)),
        vec![Action::ShowScreen(Screen::Stopping)]
    );
    assert_eq!(m.step(stopped(true)), vec![]);
    assert!(m.step(Input::Exited).contains(&Action::ShowScreen(Screen::Stopped)));
}

#[test]
fn apres_la_sortie_ordonnee_plus_rien_ne_repond() {
    let mut m = machine();
    m.step(Input::Exited);
    m.step(Input::CloseRequested);
    assert!(m.is_done());
    assert_eq!(m.step(ready(8123)), vec![]);
    assert_eq!(m.step(Input::CloseRequested), vec![]);
}

#[test]
fn l_entree_n_est_chargee_qu_une_fois_meme_si_ready_est_repete() {
    // Le secret d'entree est a usage unique cote serveur : un second chargement donnerait un 403.
    let mut m = machine();
    let first = m.step(ready(8123));
    assert!(first.contains(&Action::LoadEntry { port: 8123 }));
    assert_eq!(m.step(ready(8123)), vec![]);
    assert_eq!(m.step(ready(9000)), vec![]);
    assert_eq!(m.step(listening(8123)), vec![Action::AnnouncePort { port: 8123 }]);
}

fn notification() -> Input {
    Input::Event(SidecarEvent::Notification(Toast {
        title: "Synchronisation terminee".into(),
        body: "Tes dernieres parties sont a jour.".into(),
        kind: "sync".into(),
    }))
}

#[test]
fn une_notification_du_serveur_pret_est_montree_telle_quelle() {
    let mut m = machine();
    m.step(ready(8123));
    let expected = Toast {
        title: "Synchronisation terminee".into(),
        body: "Tes dernieres parties sont a jour.".into(),
        kind: "sync".into(),
    };
    assert_eq!(m.step(notification()), vec![Action::Notify(expected)]);
}

#[test]
fn une_notification_hors_application_ouverte_est_ignoree() {
    // Avant `ready` : rien a montrer.
    let mut m = machine();
    assert!(m.step(notification()).is_empty());
    m.step(listening(8123));
    assert!(m.step(notification()).is_empty());
    // Apres un echec, et pendant la fermeture.
    let mut failed_machine = machine();
    failed_machine.step(failed("locked"));
    assert!(failed_machine.step(notification()).is_empty());
    let mut closing = machine();
    closing.step(ready(8123));
    closing.step(Input::CloseRequested);
    assert!(closing.step(notification()).is_empty());
}

fn shell_settings(close_to_tray: bool) -> ShellSettings {
    ShellSettings {
        close_to_tray,
        locale: "fr".into(),
        ..ShellSettings::default()
    }
}

#[test]
fn les_reglages_du_serveur_pret_sont_appliques_tels_quels() {
    let mut m = machine();
    m.step(ready(8123));
    let wanted = shell_settings(true);
    assert_eq!(
        m.step(Input::Event(SidecarEvent::ShellSettings(wanted.clone()))),
        vec![Action::ApplyShellSettings(wanted)]
    );
}

#[test]
fn les_reglages_hors_application_ouverte_sont_ignores() {
    let event = || Input::Event(SidecarEvent::ShellSettings(shell_settings(true)));
    let mut m = machine();
    assert!(m.step(event()).is_empty(), "avant ready");
    let mut failed_machine = machine();
    failed_machine.step(failed("locked"));
    assert!(failed_machine.step(event()).is_empty(), "apres un echec");
    let mut closing = machine();
    closing.step(ready(8123));
    closing.step(Input::CloseRequested);
    assert!(closing.step(event()).is_empty(), "pendant la fermeture");
}

#[test]
fn la_synchronisation_demandee_part_vers_le_serveur_pret_seulement() {
    let mut m = machine();
    assert_eq!(
        m.step(Input::SyncRequested),
        vec![Action::SyncSkipped { phase: "starting" }],
        "avant ready : abandonnee, et journalisee"
    );
    m.step(ready(8123));
    assert_eq!(m.step(Input::SyncRequested), vec![Action::SendSync]);
    assert_eq!(
        m.step(Input::SyncRequested),
        vec![Action::SendSync],
        "autant de fois qu'on la demande"
    );

    let mut closing = machine();
    closing.step(ready(8123));
    closing.step(Input::CloseRequested);
    assert_eq!(
        closing.step(Input::SyncRequested),
        vec![Action::SyncSkipped { phase: "closing" }],
        "pendant la fermeture"
    );

    let mut dead = machine();
    dead.step(ready(8123));
    dead.step(Input::Exited);
    assert_eq!(
        dead.step(Input::SyncRequested),
        vec![Action::SyncSkipped { phase: "dead" }],
        "serveur sorti"
    );
}

#[test]
fn un_serveur_mort_montre_la_fenetre_avant_l_ecran_arrete() {
    // Fenetre cachee dans la zone de notification : l'ecran terminal ne s'affiche jamais sur une fenetre invisible.
    for ready_first in [false, true] {
        let mut m = machine();
        if ready_first {
            m.step(ready(8123));
        }
        let actions = m.step(Input::Exited);
        let reveal = actions
            .iter()
            .position(|a| *a == Action::RevealWindow)
            .expect("fenetre montree");
        let screen = actions
            .iter()
            .position(|a| *a == Action::ShowScreen(Screen::Stopped))
            .expect("ecran arrete");
        assert!(reveal < screen, "la fenetre est montree AVANT l'ecran : {actions:?}");
    }
}

#[test]
fn un_serveur_mort_apres_un_echec_montre_aussi_la_fenetre() {
    let mut m = machine();
    m.step(listening(8123));
    m.step(failed("locked"));
    assert!(m.step(Input::Exited).contains(&Action::RevealWindow));
}

#[test]
fn la_fermeture_ordonnee_ne_montre_pas_la_fenetre() {
    let mut m = machine();
    m.step(ready(8123));
    m.step(Input::CloseRequested);
    assert!(!m.step(Input::Exited).contains(&Action::RevealWindow));
}
