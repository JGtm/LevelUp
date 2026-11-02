//! La verification de mise a jour demandee par l'utilisateur (bouton « Rechercher une mise a jour » de la
//! fenetre « A propos ») : meme sequence que celle du lancement, issues sans mise a jour annoncees dans
//! une boite, une seule verification a la fois. La source, la boite de proposition et la boite
//! d'information sont des doubles ; la sequence, le verrou et le journal sont les vrais.

use std::fs;
use std::path::PathBuf;
use std::sync::atomic::{AtomicU32, Ordering};
use std::sync::{Arc, Mutex};

use levelup_desktop::catalogue;
use levelup_desktop::logfile::{LogSink, MAX_LOG_BYTES};
use levelup_desktop::update_manual::{UpdateBusy, report_unavailable, run_manual_check};
use levelup_desktop::updater::{
    CloserSlot, Decision, InstallGate, Installer, Notice, Notifier, Offer, Prompt, PromptText, UpdateContext,
    UpdateSource,
};

#[derive(Clone)]
enum Answer {
    UpToDate,
    Offer(&'static str),
    Failure(&'static str),
}

struct FakeSource {
    answer: Answer,
    calls: Arc<AtomicU32>,
    downloads: Arc<AtomicU32>,
}

impl UpdateSource for FakeSource {
    fn check(&mut self) -> Result<Option<Offer>, String> {
        self.calls.fetch_add(1, Ordering::SeqCst);
        match &self.answer {
            Answer::UpToDate => Ok(None),
            Answer::Offer(version) => Ok(Some(Offer {
                version: (*version).into(),
                notes: None,
            })),
            Answer::Failure(why) => Err((*why).into()),
        }
    }

    fn download(&mut self, _offer: &Offer) -> Result<Installer, String> {
        self.downloads.fetch_add(1, Ordering::SeqCst);
        Err("pas de telechargement dans ce test".into())
    }
}

struct FakePrompt {
    decision: Decision,
    asked: Arc<AtomicU32>,
}

impl Prompt for FakePrompt {
    fn ask(&mut self, _text: &PromptText) -> Decision {
        self.asked.fetch_add(1, Ordering::SeqCst);
        self.decision
    }
}

#[derive(Default)]
struct RecordingNotifier(Arc<Mutex<Vec<Notice>>>);

impl Notifier for RecordingNotifier {
    fn show(&mut self, notice: Notice) {
        self.0.lock().unwrap().push(notice);
    }
}

struct Rig {
    log: Arc<LogSink>,
    log_path: PathBuf,
    gate: InstallGate,
    closer: CloserSlot,
    busy: UpdateBusy,
    notices: Arc<Mutex<Vec<Notice>>>,
    calls: Arc<AtomicU32>,
    downloads: Arc<AtomicU32>,
    asked: Arc<AtomicU32>,
}

impl Rig {
    fn new(name: &str) -> Self {
        let dir = std::env::temp_dir().join(format!("levelup-desktop-test-{name}-{}", std::process::id()));
        fs::create_dir_all(&dir).expect("dossier de test");
        let log_path = dir.join("desktop.log");
        Self {
            log: Arc::new(LogSink::open(log_path.clone(), MAX_LOG_BYTES).expect("journal")),
            log_path,
            gate: InstallGate::default(),
            closer: CloserSlot::default(),
            busy: UpdateBusy::default(),
            notices: Arc::default(),
            calls: Arc::default(),
            downloads: Arc::default(),
            asked: Arc::default(),
        }
    }

    fn run(&self, answer: Answer, decision: Decision) {
        let context = UpdateContext {
            locale: catalogue::DEFAULT_LOCALE,
            gate: &self.gate,
            closer: &self.closer,
            log: &self.log,
        };
        let source = FakeSource {
            answer,
            calls: self.calls.clone(),
            downloads: self.downloads.clone(),
        };
        let prompt = FakePrompt {
            decision,
            asked: self.asked.clone(),
        };
        let mut notifier = RecordingNotifier(self.notices.clone());
        run_manual_check(source, prompt, &mut notifier, &self.busy, &context);
    }

    fn notices(&self) -> Vec<Notice> {
        self.notices.lock().unwrap().clone()
    }

    fn journal(&self) -> String {
        fs::read_to_string(&self.log_path).unwrap_or_default()
    }
}

#[test]
fn a_jour_une_boite_le_dit_et_rien_ne_se_propose() {
    let rig = Rig::new("a-jour");
    rig.run(Answer::UpToDate, Decision::Install);
    assert_eq!(rig.notices(), vec![Notice::UpToDate]);
    assert_eq!(rig.asked.load(Ordering::SeqCst), 0);
    assert_eq!(rig.downloads.load(Ordering::SeqCst), 0);
    assert!(!rig.gate.is_armed());
    let journal = rig.journal();
    assert!(journal.contains("event=update_check_requested"), "{journal}");
    assert!(journal.contains("event=update_none"), "{journal}");
}

#[test]
fn un_echec_de_verification_est_journalise_et_annonce() {
    let rig = Rig::new("echec");
    rig.run(Answer::Failure("hors ligne"), Decision::Install);
    assert_eq!(rig.notices(), vec![Notice::CheckFailed]);
    assert_eq!(rig.asked.load(Ordering::SeqCst), 0);
    assert!(rig.journal().contains("event=update_check_failed err=hors ligne"));
}

#[test]
fn une_mise_a_jour_trouvee_est_proposee_et_plus_tard_ne_telecharge_rien() {
    let rig = Rig::new("plus-tard");
    rig.run(Answer::Offer("9.9.9"), Decision::Later);
    assert_eq!(rig.asked.load(Ordering::SeqCst), 1);
    assert_eq!(rig.downloads.load(Ordering::SeqCst), 0);
    assert!(
        rig.notices().is_empty(),
        "une mise a jour proposee ne s'accompagne d'aucune autre boite"
    );
    assert!(!rig.gate.is_armed());
    assert!(rig.journal().contains("event=update_declined version=9.9.9"));
}

#[test]
fn accepter_telecharge_et_un_telechargement_en_echec_n_arme_rien() {
    let rig = Rig::new("telechargement");
    rig.run(Answer::Offer("9.9.9"), Decision::Install);
    assert_eq!(rig.downloads.load(Ordering::SeqCst), 1);
    assert!(!rig.gate.is_armed());
    assert!(rig.journal().contains("event=update_download_failed"));
}

#[test]
fn une_demande_pendant_une_verification_est_annoncee_pas_empilee() {
    let rig = Rig::new("occupe");
    let held = rig.busy.try_acquire().expect("verrou libre");
    rig.run(Answer::UpToDate, Decision::Install);
    assert_eq!(rig.notices(), vec![Notice::Busy]);
    assert_eq!(
        rig.calls.load(Ordering::SeqCst),
        0,
        "la source ne doit pas etre interrogee"
    );
    assert!(rig.journal().contains("event=update_check_busy"));

    // Le verrou rendu, la demande suivante aboutit.
    drop(held);
    rig.run(Answer::UpToDate, Decision::Install);
    assert_eq!(rig.notices(), vec![Notice::Busy, Notice::UpToDate]);
    assert_eq!(rig.calls.load(Ordering::SeqCst), 1);
}

#[test]
fn le_verrou_se_rend_apres_chaque_verification() {
    let rig = Rig::new("verrou");
    rig.run(Answer::Failure("x"), Decision::Later);
    assert!(
        rig.busy.try_acquire().is_some(),
        "le verrou doit etre libre apres la verification"
    );
}

#[test]
fn sans_updater_la_raison_est_dite_dans_une_boite() {
    let rig = Rig::new("indisponible");
    let mut notifier = RecordingNotifier(rig.notices.clone());
    report_unavailable(&mut notifier, &rig.log);
    assert_eq!(rig.notices(), vec![Notice::Unavailable]);
    assert!(rig.journal().contains("event=update_check_unavailable"));
}
