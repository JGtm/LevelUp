//! La mise a jour contre le VRAI faux serveur (ADR 0045 D-16, plan A3) : proposee seulement une fois le
//! serveur pret, jamais imposee, installee uniquement apres la sortie du serveur. La source de mise a
//! jour et la boite de dialogue sont des doubles ; le superviseur, l'hote qui installe et le serveur
//! sont les vrais.

mod common;

use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::{Arc, Mutex};
use std::time::{Duration, Instant};

use common::{Call, Recorder, Run, fresh_roots, launch_with_host, short_timeouts};
use levelup_desktop::catalogue;
use levelup_desktop::lifecycle::Timeouts;
use levelup_desktop::logfile::{LogSink, MAX_LOG_BYTES};
use levelup_desktop::resources::ResourcePack;
use levelup_desktop::roots;
use levelup_desktop::screen::Screen;
use levelup_desktop::supervisor::Outcome;
use levelup_desktop::updater::{
    CloserSlot, Decision, InstallGate, Installer, Notice, Notifier, Offer, Prompt, PromptText, UpdateHost,
    UpdateSource, run_update,
};

type Events = Arc<Mutex<Vec<String>>>;

fn push(events: &Events, what: impl Into<String>) {
    events.lock().unwrap().push(what.into());
}

fn snapshot(events: &Events) -> Vec<String> {
    events.lock().unwrap().clone()
}

fn wait_event(events: &Events, prefix: &str) -> Vec<String> {
    let deadline = Instant::now() + common::WAIT;
    loop {
        let seen = snapshot(events);
        if seen.iter().any(|e| e.starts_with(prefix)) {
            return seen;
        }
        assert!(Instant::now() < deadline, "jamais vu {prefix:?} : {seen:?}");
        std::thread::sleep(Duration::from_millis(20));
    }
}

/// Ce que le test regle : la mise a jour offerte, la reponse a la boite, l'issue du telechargement et de l'installation.
#[derive(Clone)]
struct Scenario {
    offer: Option<Offer>,
    decision: Decision,
    download_ok: bool,
    install_ok: bool,
    /// Le paquet de ressources que la release publie (bloc `resources` de `latest.json`).
    pack: Option<ResourcePack>,
}

impl Scenario {
    fn accepted() -> Self {
        Self {
            offer: Some(Offer {
                version: "9.9.9".into(),
                notes: Some("- une nouveaute".into()),
            }),
            decision: Decision::Install,
            download_ok: true,
            install_ok: true,
            pack: None,
        }
    }

    fn with_pack(fingerprint: &str) -> Self {
        Self {
            pack: Some(ResourcePack {
                fingerprint: fingerprint.into(),
                json: format!(r#"{{"fingerprint":"{fingerprint}","url":"https://github.com/x","signature":"c2ln"}}"#),
            }),
            ..Self::accepted()
        }
    }
}

struct FakeSource {
    scenario: Scenario,
    events: Events,
    recorder: Recorder,
    stopped_marker: std::path::PathBuf,
    closer: CloserSlot,
}

impl UpdateSource for FakeSource {
    fn check(&mut self) -> Result<Option<Offer>, String> {
        let ready = self.recorder.calls().iter().any(|c| matches!(c, Call::LoadEntry(_)));
        push(&self.events, format!("check ready={ready}"));
        Ok(self.scenario.offer.clone())
    }

    fn prepare_resources(&mut self, _offer: &Offer) -> Result<(), String> {
        let Some(pack) = self.scenario.pack.clone() else {
            return Ok(());
        };
        push(&self.events, format!("prepare {}", pack.fingerprint));
        let closer = self.closer.wait(Duration::from_secs(5)).ok_or("serveur indisponible")?;
        closer.prepare_resources(&pack)
    }

    fn download(&mut self, offer: &Offer) -> Result<Installer, String> {
        push(&self.events, format!("download {}", offer.version));
        if !self.scenario.download_ok {
            return Err("reseau coupe".into());
        }
        let (events, recorder, marker, install_ok) = (
            self.events.clone(),
            self.recorder.clone(),
            self.stopped_marker.clone(),
            self.scenario.install_ok,
        );
        Ok(Box::new(move || {
            // A3 : l'installeur ne se lance qu'apres l'arret du serveur (`stopped` recu, processus sorti) et
            // avant la sortie de l'application.
            let stopping = recorder.calls().contains(&Call::Screen(Screen::Stopping));
            let exited = recorder.calls().contains(&Call::Exit);
            push(
                &events,
                format!(
                    "install stopped={} stopping_screen={stopping} app_already_exited={exited}",
                    marker.exists()
                ),
            );
            if install_ok {
                Ok(())
            } else {
                Err("installeur refuse".into())
            }
        }))
    }
}

struct FakePrompt {
    decision: Decision,
    events: Events,
}

impl Prompt for FakePrompt {
    fn ask(&mut self, text: &PromptText) -> Decision {
        assert!(!text.title.is_empty() && text.message.contains("9.9.9"), "{text:?}");
        push(&self.events, "prompt");
        self.decision
    }
}

struct RecordingNotifier(Arc<Mutex<Vec<Notice>>>);

impl Notifier for RecordingNotifier {
    fn show(&mut self, notice: Notice) {
        self.0.lock().unwrap().push(notice);
    }
}

struct Harness {
    run: Run,
    events: Events,
    gate: InstallGate,
    notices: Arc<Mutex<Vec<Notice>>>,
}

impl Harness {
    fn start(name: &str, scenario: Scenario, fake_env: &[(&str, &str)]) -> Self {
        Self::start_with(name, scenario, fake_env, short_timeouts())
    }

    fn start_with(name: &str, scenario: Scenario, fake_env: &[(&str, &str)], timeouts: Timeouts) -> Self {
        let roots = fresh_roots(name);
        let recorder = Recorder::default();
        let events: Events = Arc::default();
        let gate = InstallGate::default();
        let slot = CloserSlot::default();
        let log = Arc::new(LogSink::open(roots::desktop_log_path(&roots), MAX_LOG_BYTES).expect("journal"));

        let source = FakeSource {
            scenario: scenario.clone(),
            events: events.clone(),
            recorder: recorder.clone(),
            stopped_marker: roots.data.join("fake-sidecar-stopped.txt"),
            closer: slot.clone(),
        };
        let prompt = FakePrompt {
            decision: scenario.decision,
            events: events.clone(),
        };
        let on_ready = {
            let (gate, slot, log) = (gate.clone(), slot.clone(), log.clone());
            Box::new(move || {
                std::thread::spawn(move || {
                    run_update(source, prompt, catalogue::DEFAULT_LOCALE, &gate, &slot, &log);
                });
            })
        };
        let notices: Arc<Mutex<Vec<Notice>>> = Arc::default();
        let host = UpdateHost::new(
            recorder.clone(),
            gate.clone(),
            log,
            on_ready,
            Box::new(RecordingNotifier(notices.clone())),
        );
        let run = launch_with_host(&roots, fake_env, timeouts, recorder, host);
        slot.set(run.handle.as_ref().expect("poignee").closer());
        Self {
            run,
            events,
            gate,
            notices,
        }
    }

    /// Fermeture demandee par l'utilisateur, puis attente de la sortie de l'application.
    fn close_and_finish(&mut self) -> Outcome {
        self.run.request_close();
        self.run.recorder.wait_exit();
        self.run.finish()
    }
}

#[test]
fn acceptee_la_mise_a_jour_s_installe_apres_l_arret_du_serveur_et_avant_la_sortie() {
    let mut h = Harness::start("maj-acceptee", Scenario::accepted(), &[("FAKE_READY_DELAY_MS", "600")]);
    let events = wait_event(&h.events, "install");
    h.run.recorder.wait_exit();
    let outcome = h.run.finish();

    assert_eq!(
        events[..3],
        [
            "check ready=true".to_string(),
            "prompt".to_string(),
            "download 9.9.9".to_string()
        ],
        "la verification attend `ready` ; l'ordre est verification, boite, telechargement : {events:?}"
    );
    assert_eq!(
        events[3], "install stopped=true stopping_screen=true app_already_exited=false",
        "installation apres `stopped`, apres l'ecran de fermeture, avant la sortie : {events:?}"
    );
    assert_eq!(outcome.stopped_clean, Some(true));
    assert!(!outcome.killed);
    assert!(h.run.stopped_marker());
    assert!(!h.gate.is_armed(), "la porte s'ouvre une seule fois");
    let log = h.run.log();
    assert!(log.contains("event=update_offered version=9.9.9"), "{log}");
    assert!(log.contains("event=update_close_requested"), "{log}");
    assert!(log.contains("event=update_install"), "{log}");
    assert!(!log.contains("event=update_install_failed"), "{log}");
    assert!(!log.contains("event=update_skipped_unclean_stop"), "{log}");
    assert!(
        h.notices.lock().unwrap().is_empty(),
        "aucun message sur un arret propre"
    );
}

#[test]
fn refusee_rien_ne_se_telecharge_ni_ne_se_ferme_et_la_fermeture_n_installe_rien() {
    let scenario = Scenario {
        decision: Decision::Later,
        ..Scenario::accepted()
    };
    let mut h = Harness::start("maj-refusee", scenario, &[]);
    wait_event(&h.events, "prompt");
    h.run.wait_log("event=update_declined version=9.9.9");
    assert!(
        !h.run.recorder.has_screen(Screen::Stopping),
        "l'application reste ouverte"
    );
    assert!(!h.gate.is_armed());

    let outcome = h.close_and_finish();
    assert_eq!(outcome.stopped_clean, Some(true));
    let events = snapshot(&h.events);
    assert!(
        !events
            .iter()
            .any(|e| e.starts_with("download") || e.starts_with("install")),
        "{events:?}"
    );
    assert!(!h.run.log().contains("event=update_install"));
}

#[test]
fn un_telechargement_en_echec_n_interrompt_pas_l_application() {
    let scenario = Scenario {
        download_ok: false,
        ..Scenario::accepted()
    };
    let mut h = Harness::start("maj-echec", scenario, &[]);
    h.run.wait_log("event=update_download_failed err=reseau coupe");
    assert!(!h.run.recorder.has_screen(Screen::Stopping));
    assert!(!h.gate.is_armed());
    let outcome = h.close_and_finish();
    assert_eq!(outcome.stopped_clean, Some(true));
    assert!(!snapshot(&h.events).iter().any(|e| e.starts_with("install")));
}

#[test]
fn a_jour_rien_n_est_propose() {
    let scenario = Scenario {
        offer: None,
        ..Scenario::accepted()
    };
    let mut h = Harness::start("maj-a-jour", scenario, &[]);
    h.run.wait_log("event=update_none");
    assert_eq!(snapshot(&h.events), vec!["check ready=true".to_string()]);
    assert!(!h.run.recorder.has_screen(Screen::Stopping));
    h.close_and_finish();
}

#[test]
fn un_serveur_qui_echoue_au_demarrage_ne_declenche_aucune_verification() {
    let mut h = Harness::start(
        "maj-serveur-en-echec",
        Scenario::accepted(),
        &[("FAKE_VARIANT", "failed"), ("FAKE_FAILED_STATE", "locked")],
    );
    h.run.recorder.wait_url();
    // Laisse a un fil de verification, s'il existait, le temps de se manifester.
    std::thread::sleep(Duration::from_millis(500));
    assert!(snapshot(&h.events).is_empty(), "{:?}", snapshot(&h.events));
    h.close_and_finish();
}

#[test]
fn une_installation_refusee_est_journalisee_et_l_application_sort_quand_meme() {
    let scenario = Scenario {
        install_ok: false,
        ..Scenario::accepted()
    };
    let mut h = Harness::start("maj-install-refusee", scenario, &[]);
    wait_event(&h.events, "install");
    h.run.recorder.wait_exit();
    let outcome = h.run.finish();
    assert_eq!(outcome.stopped_clean, Some(true));
    assert!(
        h.run
            .log()
            .contains("event=update_install_failed err=installeur refuse")
    );
    assert_eq!(
        *h.notices.lock().unwrap(),
        vec![Notice::InstallFailed],
        "l'utilisateur est prevenu que l'installation a echoue"
    );
}

#[test]
fn l_hote_n_installe_rien_sans_mise_a_jour_armee() {
    // Fermeture ordinaire : l'hote qui enveloppe ne change rien au contrat de sortie.
    let ran = Arc::new(AtomicBool::new(false));
    let roots = fresh_roots("maj-sans-porte");
    let recorder = Recorder::default();
    let log = Arc::new(LogSink::open(roots::desktop_log_path(&roots), MAX_LOG_BYTES).expect("journal"));
    let flag = ran.clone();
    let host = UpdateHost::new(
        recorder.clone(),
        InstallGate::default(),
        log,
        Box::new(move || flag.store(true, Ordering::SeqCst)),
        Box::new(RecordingNotifier(Arc::default())),
    );
    let mut run = launch_with_host(&roots, &[], short_timeouts(), recorder, host);
    run.recorder.wait_url();
    run.request_close();
    run.recorder.wait_exit();
    let outcome = run.finish();
    assert_eq!(outcome.stopped_clean, Some(true));
    assert!(
        ran.load(Ordering::SeqCst),
        "le signal de pret est donne une fois l'entree chargee"
    );
    assert!(!run.log().contains("event=update_install"));
}

#[test]
fn un_arret_non_propre_n_installe_rien_previent_et_sort_quand_meme() {
    let mut h = Harness::start(
        "maj-arret-non-propre",
        Scenario::accepted(),
        &[("FAKE_STOP_CLEAN", "false")],
    );
    h.run.wait_log("event=update_close_requested");
    h.run.recorder.wait_exit();
    let outcome = h.run.finish();

    assert_eq!(outcome.stopped_clean, Some(false));
    assert!(
        !outcome.killed,
        "le serveur est sorti seul : c'est `clean=false` qui interdit l'installation"
    );
    let events = snapshot(&h.events);
    assert!(
        !events.iter().any(|e| e.starts_with("install")),
        "l'installeur ne doit pas partir : {events:?}"
    );
    let log = h.run.log();
    assert!(
        log.contains("event=update_skipped_unclean_stop reason=unclean"),
        "{log}"
    );
    assert!(!log.contains("event=update_install"), "{log}");
    assert_eq!(*h.notices.lock().unwrap(), vec![Notice::SkippedUnclean]);
    assert!(
        !h.gate.is_armed(),
        "la porte est ouverte une seule fois, meme sans installer"
    );
}

#[test]
fn un_serveur_qui_ne_sort_pas_dans_le_budget_est_tue_et_rien_ne_s_installe() {
    let timeouts = Timeouts {
        close: Duration::from_secs(1),
        grace: Duration::from_secs(1),
        ..short_timeouts()
    };
    let env = [("FAKE_VARIANT", "late_stopped"), ("FAKE_STOP_DELAY_MS", "30000")];
    let mut h = Harness::start_with("maj-serveur-tue", Scenario::accepted(), &env, timeouts);
    h.run.wait_log("event=update_close_requested");
    h.run.recorder.wait_exit();
    let outcome = h.run.finish();

    assert!(outcome.killed, "le serveur etait encore vivant apres le budget");
    assert_eq!(outcome.stopped_clean, None);
    let events = snapshot(&h.events);
    assert!(
        !events.iter().any(|e| e.starts_with("install")),
        "l'installeur ne doit pas partir sur un serveur tue : {events:?}"
    );
    let log = h.run.log();
    assert!(log.contains("event=update_skipped_unclean_stop reason=killed"), "{log}");
    assert!(!log.contains("event=update_install"), "{log}");
    assert_eq!(*h.notices.lock().unwrap(), vec![Notice::SkippedUnclean]);
}

const PACK_READY: &str = r#"{"event":"resources","phase":"downloading","fingerprint":"F1","percent":40}|{"event":"resources","phase":"ready","fingerprint":"F1","restart_needed":true}"#;
const PACK_FAILED: &str = r#"{"event":"resources","phase":"downloading","fingerprint":"F1"}|{"event":"resources","phase":"failed","fingerprint":"F1","code":"resources_signature_invalid"}"#;

/// L24 : le paquet de ressources est pret AVANT le telechargement de l'installeur du moteur qui l'exige, puis le
/// moteur s'installe comme d'habitude. La commande `resources <json>` est arrivee sur la stdin du serveur.
#[test]
fn le_paquet_de_ressources_est_pret_avant_le_telechargement_de_l_installeur() {
    let mut h = Harness::start(
        "maj-paquet-pret",
        Scenario::with_pack("F1"),
        &[("FAKE_RESOURCES_REPLY", PACK_READY)],
    );
    let events = wait_event(&h.events, "install");
    h.run.recorder.wait_exit();
    let outcome = h.run.finish();
    assert_eq!(
        events[..4],
        [
            "check ready=true".to_string(),
            "prompt".to_string(),
            "prepare F1".to_string(),
            "download 9.9.9".to_string()
        ],
        "le paquet precede l'installeur : {events:?}"
    );
    assert_eq!(outcome.stopped_clean, Some(true));
    let commands = std::fs::read_to_string(h.run.roots.data.join("fake-sidecar-commands.txt")).unwrap_or_default();
    assert!(
        commands
            .lines()
            .any(|l| l.starts_with("resources {") && l.contains("\"fingerprint\":\"F1\"")),
        "la commande resources est arrivee : {commands:?}"
    );
    let log = h.run.log();
    assert!(log.contains("event=resources_requested"), "{log}");
    assert!(log.contains("kind=resources phase=ready fingerprint=F1"), "{log}");
    assert!(log.contains("event=update_install"), "{log}");
}

/// Un paquet qui n'a pas pu etre prepare (signature refusee) : l'installeur n'est jamais telecharge et le moteur
/// qui l'exige n'est pas installe ; l'application reste ouverte et la raison est journalisee.
#[test]
fn un_paquet_en_echec_empeche_l_installation_du_moteur() {
    let mut h = Harness::start(
        "maj-paquet-echec",
        Scenario::with_pack("F1"),
        &[("FAKE_RESOURCES_REPLY", PACK_FAILED)],
    );
    h.run.wait_log("event=update_download_failed");
    let events = snapshot(&h.events);
    assert!(
        !events
            .iter()
            .any(|e| e.starts_with("download") || e.starts_with("install")),
        "{events:?}"
    );
    assert!(!h.gate.is_armed());
    assert!(
        !h.run.recorder.has_screen(Screen::Stopping),
        "l'application reste ouverte"
    );
    let log = h.run.log();
    assert!(
        log.contains("resources_failed code=resources_signature_invalid"),
        "{log}"
    );
    let outcome = h.close_and_finish();
    assert_eq!(outcome.stopped_clean, Some(true));
    assert!(!h.run.log().contains("event=update_install"));
}

/// Un serveur qui s'arrete pendant la preparation reveille l'attente avec la raison, sans attendre le delai.
#[test]
fn un_serveur_arrete_pendant_la_preparation_n_installe_rien() {
    let mut h = Harness::start("maj-paquet-serveur-mort", Scenario::with_pack("F1"), &[]);
    h.run.wait_log("event=resources_requested");
    h.run.request_close();
    h.run.recorder.wait_exit();
    h.run.finish();
    let events = snapshot(&h.events);
    assert!(!events.iter().any(|e| e.starts_with("download")), "{events:?}");
    assert!(!h.run.log().contains("event=update_install"));
}
