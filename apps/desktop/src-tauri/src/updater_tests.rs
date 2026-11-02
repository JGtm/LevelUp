//! Tests unitaires de `updater` : la machine d'etats, la porte, l'hote et les decisions d'installation.

use super::*;

fn offer(version: &str) -> Offer {
    Offer {
        version: version.into(),
        notes: Some("- un changement\n- un autre".into()),
    }
}

fn drive(flow: &mut UpdateFlow, inputs: Vec<UpdateInput>) -> Vec<Vec<UpdateAction>> {
    inputs.into_iter().map(|input| flow.step(input)).collect()
}

#[test]
fn la_verification_ne_commence_qu_au_serveur_pret_et_une_seule_fois() {
    let mut flow = UpdateFlow::new();
    assert_eq!(flow.state(), &UpdateState::Idle);
    assert_eq!(flow.step(UpdateInput::ServerReady), vec![UpdateAction::Check]);
    assert!(flow.step(UpdateInput::ServerReady).is_empty());
    assert_eq!(flow.state(), &UpdateState::Checking);
}

#[test]
fn rien_ne_se_passe_avant_la_verification() {
    let mut flow = UpdateFlow::new();
    assert!(flow.step(UpdateInput::Decided(Decision::Install)).is_empty());
    assert!(flow.step(UpdateInput::Downloaded(Ok(()))).is_empty());
    assert_eq!(flow.state(), &UpdateState::Idle);
}

#[test]
fn proposee_acceptee_telechargee_puis_porte_armee_et_fermeture() {
    let mut flow = UpdateFlow::new();
    let steps = drive(
        &mut flow,
        vec![
            UpdateInput::ServerReady,
            UpdateInput::Checked(Ok(Some(offer("8.2.0")))),
            UpdateInput::Decided(Decision::Install),
            UpdateInput::Downloaded(Ok(())),
        ],
    );
    assert_eq!(steps[0], vec![UpdateAction::Check]);
    assert_eq!(steps[1], vec![UpdateAction::Ask(offer("8.2.0"))]);
    assert_eq!(steps[2], vec![UpdateAction::Download(offer("8.2.0"))]);
    assert_eq!(steps[3], vec![UpdateAction::ArmAndClose]);
    assert_eq!(flow.state(), &UpdateState::Armed);
}

#[test]
fn plus_tard_ne_telecharge_rien_et_ne_ferme_rien() {
    let mut flow = UpdateFlow::new();
    let steps = drive(
        &mut flow,
        vec![
            UpdateInput::ServerReady,
            UpdateInput::Checked(Ok(Some(offer("8.2.0")))),
            UpdateInput::Decided(Decision::Later),
            // Une reponse tardive ou repetee ne ressuscite pas la proposition.
            UpdateInput::Decided(Decision::Install),
            UpdateInput::Downloaded(Ok(())),
        ],
    );
    assert!(matches!(steps[2].as_slice(), [UpdateAction::Log(l)] if l.contains("update_declined")));
    assert!(steps[3].is_empty() && steps[4].is_empty());
    assert_eq!(flow.state(), &UpdateState::Done);
}

#[test]
fn a_jour_ou_erreur_de_verification_ne_proposent_rien() {
    for checked in [Ok(None), Err("reseau".to_string())] {
        let mut flow = UpdateFlow::new();
        flow.step(UpdateInput::ServerReady);
        let actions = flow.step(UpdateInput::Checked(checked));
        assert!(matches!(actions.as_slice(), [UpdateAction::Log(_)]), "{actions:?}");
        assert_eq!(flow.state(), &UpdateState::Done);
    }
}

#[test]
fn un_telechargement_en_echec_n_arme_pas_la_porte() {
    let mut flow = UpdateFlow::new();
    drive(
        &mut flow,
        vec![
            UpdateInput::ServerReady,
            UpdateInput::Checked(Ok(Some(offer("8.2.0")))),
            UpdateInput::Decided(Decision::Install),
        ],
    );
    let actions = flow.step(UpdateInput::Downloaded(Err("coupe".into())));
    assert!(matches!(actions.as_slice(), [UpdateAction::Log(l)] if l.contains("update_download_failed")));
    assert_eq!(flow.state(), &UpdateState::Done);
}

#[test]
fn un_telechargement_abandonne_se_dit_seulement_a_une_verification_demandee() {
    let steps = || {
        vec![
            UpdateInput::ServerReady,
            UpdateInput::Checked(Ok(Some(offer("8.2.0")))),
            UpdateInput::Decided(Decision::Install),
        ]
    };
    let mut manual = UpdateFlow::manual();
    drive(&mut manual, steps());
    let actions = manual.step(UpdateInput::Downloaded(Err("download_stalled".into())));
    assert!(matches!(
        actions.as_slice(),
        [UpdateAction::Log(l), UpdateAction::Inform(Notice::CheckFailed)] if l.contains("download_stalled")
    ));
    assert_eq!(manual.state(), &UpdateState::Done);
}

#[test]
fn la_boite_de_dialogue_est_complete_dans_chaque_langue() {
    for code in catalogue::LOCALES {
        let text = prompt_text(code, &offer("8.2.0"));
        for (name, value) in [
            ("title", &text.title),
            ("message", &text.message),
            ("install", &text.install),
            ("later", &text.later),
        ] {
            assert!(!value.trim().is_empty(), "{code}/{name} vide");
        }
        assert!(text.message.contains("8.2.0"), "{code} : version absente");
        assert!(!text.message.contains("{version}"), "{code} : gabarit non rempli");
        assert!(text.message.contains("- un changement"), "{code} : notes absentes");
        assert_ne!(text.install, text.later);
    }
}

#[test]
fn sans_notes_le_message_n_a_pas_d_entete_de_notes() {
    let without = Offer {
        version: "8.2.0".into(),
        notes: Some("  \n".into()),
    };
    let locale = catalogue::DEFAULT_LOCALE;
    let message = prompt_text(locale, &without).message;
    assert!(!message.contains(catalogue::text(locale, "update_notes_heading").unwrap()));
    assert!(!message.contains('\n'));
}

#[test]
fn les_notes_trop_longues_sont_coupees_net_avec_une_ellipse() {
    let long = Offer {
        version: "8.2.0".into(),
        notes: Some("Z".repeat(MAX_NOTES_CHARS * 2)),
    };
    let message = prompt_text(catalogue::DEFAULT_LOCALE, &long).message;
    assert!(message.ends_with('\u{2026}'));
    assert_eq!(message.matches('Z').count(), MAX_NOTES_CHARS);
}

#[test]
fn la_porte_s_ouvre_une_seule_fois() {
    let gate = InstallGate::default();
    assert!(!gate.is_armed() && gate.take().is_none());
    gate.arm(Box::new(|| Ok(())));
    assert!(gate.is_armed());
    assert!(gate.take().is_some());
    assert!(gate.take().is_none());
}

#[test]
fn la_poignee_de_fermeture_expire_si_elle_n_est_jamais_posee() {
    assert!(CloserSlot::default().wait(Duration::from_millis(50)).is_none());
}

fn outcome(stopped_clean: Option<bool>, killed: bool, kill_failed: bool) -> Outcome {
    Outcome {
        stopped_clean,
        killed,
        kill_failed,
    }
}

#[test]
fn l_installation_ne_part_que_sur_un_arret_propre_sans_kill() {
    assert_eq!(install_allowed(outcome(Some(true), false, false)), Ok(()));
    assert_eq!(
        install_allowed(outcome(Some(false), false, false)),
        Err(SkipReason::Unclean)
    );
    assert_eq!(install_allowed(outcome(None, false, false)), Err(SkipReason::NoStopped));
    // Un `stopped` propre ne rachete pas un kill : le serveur a ete tue ou a refuse de sortir.
    assert_eq!(
        install_allowed(outcome(Some(true), true, false)),
        Err(SkipReason::Killed)
    );
    assert_eq!(install_allowed(outcome(None, true, false)), Err(SkipReason::Killed));
    // Un kill echoue est la raison la plus precise : le processus vit peut-etre encore.
    assert_eq!(
        install_allowed(outcome(Some(true), true, true)),
        Err(SkipReason::KillFailed)
    );
}

#[test]
fn chaque_raison_a_un_nom_journalise_distinct() {
    let names = [
        SkipReason::KillFailed,
        SkipReason::Killed,
        SkipReason::Unclean,
        SkipReason::NoStopped,
    ]
    .map(SkipReason::name);
    assert_eq!(names, ["kill_failed", "killed", "unclean", "no_stopped"]);
}

#[test]
fn les_messages_de_boite_sont_complets_et_distincts_dans_chaque_langue() {
    for code in catalogue::LOCALES {
        let mut seen = Vec::new();
        for notice in Notice::ALL {
            let text = notice_text(code, notice);
            assert!(!text.title.trim().is_empty(), "{code}/{notice:?} : titre vide");
            assert!(!text.message.trim().is_empty(), "{code}/{notice:?} : message vide");
            assert!(
                !text.message.contains("{version}"),
                "{code}/{notice:?} : version non remplacee"
            );
            assert!(
                !seen.contains(&text.message),
                "{code}/{notice:?} : message deja utilise"
            );
            seen.push(text.message);
        }
    }
}

#[test]
fn le_message_a_jour_nomme_la_version_installee() {
    for code in catalogue::LOCALES {
        assert!(notice_text(code, Notice::UpToDate).message.contains(CURRENT_VERSION));
    }
}

#[test]
fn une_verification_demandee_annonce_a_jour_et_echec_pas_celle_du_lancement() {
    let mut silent = UpdateFlow::new();
    silent.step(UpdateInput::ServerReady);
    let none = silent.step(UpdateInput::Checked(Ok(None)));
    assert_eq!(none, vec![UpdateAction::Log("event=update_none".into())]);

    let mut asked = UpdateFlow::manual();
    asked.step(UpdateInput::ServerReady);
    let none = asked.step(UpdateInput::Checked(Ok(None)));
    assert_eq!(
        none,
        vec![
            UpdateAction::Log("event=update_none".into()),
            UpdateAction::Inform(Notice::UpToDate)
        ]
    );

    let mut asked = UpdateFlow::manual();
    asked.step(UpdateInput::ServerReady);
    let failed = asked.step(UpdateInput::Checked(Err("hors ligne".into())));
    assert_eq!(
        failed,
        vec![
            UpdateAction::Log("event=update_check_failed err=hors ligne".into()),
            UpdateAction::Inform(Notice::CheckFailed)
        ]
    );
}

#[test]
fn une_verification_demandee_propose_la_mise_a_jour_comme_au_lancement() {
    let mut flow = UpdateFlow::manual();
    flow.step(UpdateInput::ServerReady);
    let ask = flow.step(UpdateInput::Checked(Ok(Some(offer("8.2.0")))));
    assert_eq!(ask, vec![UpdateAction::Ask(offer("8.2.0"))]);
}

#[test]
fn la_verification_aupres_du_serveur_a_une_echeance_bornee() {
    assert!(
        CHECK_TIMEOUT >= Duration::from_secs(5),
        "assez pour une connexion lente"
    );
    assert!(
        CHECK_TIMEOUT <= Duration::from_secs(60),
        "assez court pour rendre le verrou dans la minute"
    );
}
