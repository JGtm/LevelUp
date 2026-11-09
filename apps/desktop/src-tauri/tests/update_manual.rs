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
use levelup_desktop::update_manual::{
    UpdateBusy, ensure_available, report_unavailable, run_launch_check, run_manual_check, spawn_check_thread,
};
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

#[test]
fn la_verification_du_lancement_est_sautee_quand_le_verrou_est_pris() {
    let busy = UpdateBusy::default();
    let held = busy.try_acquire().expect("verrou libre");
    let mut ran = false;
    assert!(!run_launch_check(&busy, || ran = true));
    assert!(
        !ran,
        "la verification ne doit pas partir pendant qu'une autre tient le verrou"
    );
    drop(held);
    assert!(run_launch_check(&busy, || ran = true));
    assert!(ran);
}

#[test]
fn la_verification_du_lancement_tient_le_verrou_puis_le_rend_meme_sur_panique() {
    let busy = UpdateBusy::default();
    let probe = busy.clone();
    assert!(run_launch_check(&busy, || {
        assert!(
            probe.try_acquire().is_none(),
            "le verrou est tenu pendant la verification"
        );
    }));
    assert!(
        busy.try_acquire().is_some(),
        "verrou rendu apres une verification normale"
    );

    let panicked = std::panic::catch_unwind(std::panic::AssertUnwindSafe(|| {
        run_launch_check(&busy, || panic!("verification en panne"));
    }));
    assert!(panicked.is_err());
    assert!(
        busy.try_acquire().is_some(),
        "verrou rendu meme quand la verification panique"
    );
}

#[test]
fn sans_updater_la_verification_demandee_ne_part_pas_et_dit_pourquoi() {
    let rig = Rig::new("non-branche");
    let mut notifier = RecordingNotifier(rig.notices.clone());
    assert!(!ensure_available(false, &mut notifier, &rig.log));
    assert_eq!(rig.notices(), vec![Notice::Unavailable]);

    let mut quiet = RecordingNotifier(Arc::default());
    let seen = quiet.0.clone();
    assert!(ensure_available(true, &mut quiet, &rig.log));
    assert!(seen.lock().unwrap().is_empty(), "un build branche ne dit rien");
}

#[test]
fn la_verification_part_sur_un_fil_separe_sans_faire_attendre_l_appelant() {
    let (release, wait) = std::sync::mpsc::channel::<()>();
    let (report, seen) = std::sync::mpsc::channel::<(std::thread::ThreadId, Option<String>)>();
    let caller = std::thread::current().id();
    let handle = spawn_check_thread("update-check", move || {
        wait.recv().expect("liberation");
        let current = std::thread::current();
        report
            .send((current.id(), current.name().map(String::from)))
            .expect("rapport");
    })
    .expect("fil");
    // L'appelant est deja revenu alors que le travail attend encore : il n'est pas execute en ligne.
    assert!(seen.try_recv().is_err());
    release.send(()).expect("liberation");
    let (thread, name) = seen.recv().expect("rapport");
    handle.join().expect("fin du fil");
    assert_ne!(thread, caller);
    assert_eq!(name.as_deref(), Some("update-check"));
}
