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
use levelup_desktop::logfile::{LogSink, MAX_LOG_BYTES};
use levelup_desktop::roots;
use levelup_desktop::screen::Screen;
use levelup_desktop::updater::{
    CloserSlot, Decision, InstallGate, Installer, Offer, Prompt, PromptText, UpdateHost, UpdateSource, run_update,
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
        }
    }
}

struct FakeSource {
    scenario: Scenario,
    events: Events,
    recorder: Recorder,
    stopped_marker: std::path::PathBuf,
}

impl UpdateSource for FakeSource {
    fn check(&mut self) -> Result<Option<Offer>, String> {
        let ready = self.recorder.calls().iter().any(|c| matches!(c, Call::LoadEntry(_)));
        push(&self.events, format!("check ready={ready}"));
        Ok(self.scenario.offer.clone())
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

struct Harness {
    run: Run,
    events: Events,
    gate: InstallGate,
}

impl Harness {
    fn start(name: &str, scenario: Scenario, fake_env: &[(&str, &str)]) -> Self {
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
        let host = UpdateHost::new(recorder.clone(), gate.clone(), log, on_ready);
        let run = launch_with_host(&roots, fake_env, short_timeouts(), recorder, host);
        slot.set(run.handle.as_ref().expect("poignee").closer());
        Self { run, events, gate }
    }

    /// Fermeture demandee par l'utilisateur, puis attente de la sortie de l'application.
    fn close_and_finish(&mut self) -> levelup_desktop::supervisor::Outcome {
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
