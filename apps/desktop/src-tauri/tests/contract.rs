//! Le contrat coquille <-> serveur, cote coquille, contre le VRAI faux serveur (ADR 0045 D-21).
//!
//! Chaque test lance `testdata/fake-sidecar` (compile par `go build` dans un dossier temporaire)
//! sous le superviseur reel, avec un hote qui enregistre au lieu d'une fenetre. Aucun test
//! n'ouvre de fenetre : le dev est solo et la CI n'a pas d'ecran.

mod common;

use std::time::{Duration, Instant};

use common::{Call, Reply, fresh_roots, http_get, launch, normalized, port_of, short_timeouts};
use levelup_desktop::lifecycle::Timeouts;
use levelup_desktop::screen::{FailureKind, Screen};
use levelup_desktop::sidecar::SERVER_ARGS;

fn entry_prefix(port: u16, secret: &str) -> String {
    format!("http://127.0.0.1:{port}/desktop/enter?secret={secret}")
}

/// Fermeture complete : demande, attente de la sortie de l'application.
fn close_and_finish(run: &mut common::Run) -> levelup_desktop::supervisor::Outcome {
    run.request_close();
    run.recorder.wait_exit();
    run.finish()
}

#[test]
fn nominal_listening_ready_entree_puis_fermeture_propre() {
    let roots = fresh_roots("nominal");
    let mut run = launch(&roots, &[], short_timeouts());

    let url = run.recorder.wait_url();
    let port = port_of(&url);
    assert_eq!(
        url,
        entry_prefix(port, &run.secret),
        "la premiere page est l'entree du serveur avec le secret"
    );

    // L'entree se joue comme le fera la fenetre : 302 vers /, cookie lie au port.
    let enter: Reply = http_get(port, url.strip_prefix(&format!("http://127.0.0.1:{port}")).unwrap());
    assert_eq!(enter.status, 302, "{}", enter.head);
    assert!(
        enter.head.contains(&format!("levelup_session_{port}=")),
        "{}",
        enter.head
    );
    assert_eq!(http_get(port, "/health").status, 200);

    let started = Instant::now();
    let outcome = close_and_finish(&mut run);
    assert!(
        started.elapsed() < Duration::from_secs(10),
        "fermeture en {:?}",
        started.elapsed()
    );
    assert_eq!(outcome.stopped_clean, Some(true));
    assert!(!outcome.killed);
    assert!(run.stopped_marker(), "le faux serveur a fini son arret avant de sortir");

    let calls = run.recorder.calls();
    assert_eq!(calls.first(), Some(&Call::Screen(Screen::Starting)));
    assert!(calls.contains(&Call::Port(port)));
    let stopping = calls
        .iter()
        .position(|c| *c == Call::Screen(Screen::Stopping))
        .expect("ecran de fermeture");
    assert_eq!(calls.iter().position(|c| *c == Call::Exit), Some(calls.len() - 1));
    assert!(stopping < calls.len() - 1);
    // L'entree n'est chargee qu'une fois par lancement, par la voie de l'entree (jamais la voie des pages
    // d'echec) : son URL porte le secret.
    let entries = calls.iter().filter(|c| matches!(c, Call::LoadEntry(_))).count();
    assert_eq!(entries, 1, "appels : {calls:?}");
    assert!(!calls.iter().any(|c| matches!(c, Call::LoadUrl(_))));
    let log = run.log();
    assert!(log.contains("event=sidecar_exited code=Some(0)"), "{log}");
    assert!(log.contains("event=close_stdin"), "{log}");
}

#[test]
fn le_secret_n_est_ni_un_argument_ni_dans_le_journal() {
    // Le faux serveur sort en 4 si le secret apparait dans ses arguments : le lancement nominal
    // qui aboutit le prouve ; la ligne de commande exacte est aussi verifiee ici.
    assert_eq!(SERVER_ARGS, ["-mode", "desktop", "-arret-sur-stdin"]);
    let roots = fresh_roots("secret");
    let mut run = launch(&roots, &[], short_timeouts());
    let url = run.recorder.wait_url();
    assert!(url.contains(&run.secret), "l'URL d'entree porte le secret");
    let outcome = close_and_finish(&mut run);
    assert_eq!(outcome.stopped_clean, Some(true));
    let log = run.log();
    assert!(!log.contains(&run.secret), "le secret est dans desktop.log");
    assert!(!log.contains("secret="), "une URL d'entree est dans desktop.log");
}

#[test]
fn le_serveur_recoit_les_racines_et_le_repertoire_courant_et_sa_stderr_est_relayee() {
    let roots = fresh_roots("stderr");
    let mut run = launch(&roots, &[], short_timeouts());
    run.recorder.wait_url();
    let boot_log = run.wait_log("[sidecar] fake: boot");
    close_and_finish(&mut run);
    let log = run.log();
    let boot = boot_log
        .lines()
        .find(|l| l.contains("[sidecar] fake: boot"))
        .expect("ligne de boot");
    let field = |name: &str| -> String {
        boot.split_whitespace()
            .find_map(|w| w.strip_prefix(&format!("{name}=")))
            .unwrap_or_default()
            .to_string()
    };
    let same = |reported: String, expected: &std::path::Path| {
        assert_eq!(
            normalized(std::path::Path::new(&reported)),
            normalized(expected),
            "ligne : {boot}"
        );
    };
    same(field("cwd"), &roots.resources);
    same(field("resources"), &roots.resources);
    same(field("data"), &roots.data);
    same(field("logs"), &roots.logs);
    // Les lignes de la coquille et celles du serveur partagent le meme journal.
    assert!(log.contains("[shell] event=sidecar_spawned"));
    assert!(log.contains("[sidecar] fake: stdin ferme, arret demande"));
}

#[test]
fn un_serveur_bavard_de_10_mo_ne_bloque_pas_la_coquille() {
    let roots = fresh_roots("bavard");
    let ten_mib = (10 * 1024 * 1024).to_string();
    let mut run = launch(&roots, &[("FAKE_CHATTY_BYTES", &ten_mib)], short_timeouts());
    // Sans lecture continue de stdout, le serveur resterait bloque et `ready` ne viendrait jamais.
    let url = run.recorder.wait_url();
    assert_eq!(url, entry_prefix(port_of(&url), &run.secret));
    let outcome = close_and_finish(&mut run);
    assert_eq!(outcome.stopped_clean, Some(true));
    let log = run.log();
    let lines = log
        .lines()
        .filter(|l| l.contains("event=stdout_ignored reason=not_json"))
        .count();
    assert!(
        lines > 0 && lines <= 20,
        "{lines} lignes ignorees journalisees, attendu de 1 a 20"
    );
    let total = log
        .lines()
        .find_map(|l| l.split("event=stdout_ignored_total count=").nth(1))
        .and_then(|n| n.trim().parse::<u64>().ok())
        .expect("compte total des lignes ignorees");
    assert!(total >= 10 * 1024 * 1024 / 100 - 1, "total {total}");
}

#[test]
fn jamais_ready_dans_le_delai_montre_le_delai_et_tue_le_serveur() {
    let roots = fresh_roots("delai");
    let timeouts = Timeouts {
        startup: Duration::from_secs(2),
        ..short_timeouts()
    };
    let mut run = launch(&roots, &[("FAKE_VARIANT", "never_ready")], timeouts);
    run.recorder.wait_screen(Screen::Failed(FailureKind::Timeout));
    let log = run.wait_log("event=sidecar_exited");
    assert!(log.contains("event=kill"), "{log}");
    assert!(
        !log.contains("event=sidecar_exited code=Some(0)"),
        "le serveur a ete tue, pas arrete : {log}"
    );
    assert!(
        !run.recorder.calls().iter().any(|c| c.is_navigation()),
        "aucune entree sans ready"
    );
    let outcome = close_and_finish(&mut run);
    assert!(outcome.killed);
    assert!(!run.stopped_marker());
}

/// Le paquet de ressources absent se telecharge AVANT `ready` : tant que le telechargement progresse, la
/// coquille ne tue pas le serveur, meme quand il dure plus que le delai de demarrage.
#[test]
fn un_telechargement_qui_progresse_plus_longtemps_que_le_delai_ne_tue_pas_le_serveur() {
    let roots = fresh_roots("telechargement");
    let timeouts = Timeouts {
        startup: Duration::from_secs(2),
        ..short_timeouts()
    };
    let started = Instant::now();
    let mut run = launch(
        &roots,
        &[("FAKE_PROGRESS_FOR_MS", "4500"), ("FAKE_PROGRESS_EVERY_MS", "300")],
        timeouts,
    );
    run.recorder.wait_url();
    assert!(
        started.elapsed() > Duration::from_millis(4000),
        "l'entree ne vient qu'apres le telechargement : {:?}",
        started.elapsed()
    );
    assert!(
        !run.recorder.has_screen(Screen::Failed(FailureKind::Timeout)),
        "le telechargement progressait : pas de delai"
    );
    let outcome = close_and_finish(&mut run);
    assert!(!outcome.killed);
    assert_eq!(outcome.stopped_clean, Some(true));
    let log = run.log();
    assert!(log.contains("kind=resources phase=downloading"), "{log}");
    assert!(!log.contains("event=kill"), "{log}");
}

/// Un telechargement qui s'arrete (plus aucun avancement) finit quand meme par le delai de demarrage : le
/// serveur est tue et l'ecran du delai s'affiche.
#[test]
fn un_telechargement_bloque_finit_par_le_delai_de_demarrage() {
    let roots = fresh_roots("telechargement_bloque");
    let timeouts = Timeouts {
        startup: Duration::from_secs(2),
        ..short_timeouts()
    };
    let started = Instant::now();
    let mut run = launch(
        &roots,
        &[
            ("FAKE_VARIANT", "never_ready"),
            ("FAKE_PROGRESS_FOR_MS", "1000"),
            ("FAKE_PROGRESS_EVERY_MS", "250"),
        ],
        timeouts,
    );
    run.recorder.wait_screen(Screen::Failed(FailureKind::Timeout));
    assert!(
        started.elapsed() > Duration::from_millis(2500),
        "le delai court depuis le DERNIER avancement, pas depuis le lancement : {:?}",
        started.elapsed()
    );
    let log = run.wait_log("event=sidecar_exited");
    assert!(log.contains("event=kill"), "{log}");
    let outcome = close_and_finish(&mut run);
    assert!(outcome.killed);
}

#[test]
fn un_echec_avec_port_charge_la_page_du_serveur() {
    let roots = fresh_roots("echec");
    let env = [("FAKE_VARIANT", "failed"), ("FAKE_FAILED_STATE", "locked")];
    let mut run = launch(&roots, &env, short_timeouts());
    let url = run.recorder.wait_url();
    let port = port_of(&url);
    assert_eq!(
        url,
        format!("http://127.0.0.1:{port}/"),
        "page de demarrage du serveur, sans secret"
    );
    let page = http_get(port, "/");
    assert_eq!(page.status, 503, "{}", page.head);

    let outcome = close_and_finish(&mut run);
    assert!(!outcome.killed);
    assert!(run.recorder.has_screen(Screen::Stopping));
    assert!(
        !run.stopped_marker(),
        "un serveur qui refuse de demarrer n'annonce pas stopped"
    );
}

#[test]
fn un_echec_sans_port_montre_l_ecran_embarque_de_sa_cause() {
    for (state, kind) in [
        ("port", FailureKind::Port),
        ("locked", FailureKind::Locked),
        ("schema_newer", FailureKind::SchemaNewer),
        ("migration_failed", FailureKind::MigrationFailed),
        ("resources_incompatible", FailureKind::ResourcesIncompatible),
    ] {
        let roots = fresh_roots("echec-sans-port");
        let env = [("FAKE_VARIANT", "failed_no_port"), ("FAKE_FAILED_STATE", state)];
        let mut run = launch(&roots, &env, short_timeouts());
        run.recorder.wait_screen(Screen::Failed(kind));
        assert!(
            !run.recorder.calls().iter().any(|c| c.is_navigation()),
            "{state} : pas de page serveur sans port"
        );
        close_and_finish(&mut run);
    }
}

#[test]
fn un_crash_sans_evenement_montre_l_ecran_arrete() {
    let roots = fresh_roots("crash");
    let mut run = launch(&roots, &[("FAKE_VARIANT", "crash")], short_timeouts());
    run.recorder.wait_screen(Screen::Stopped);
    let log = run.log();
    assert!(log.contains("event=sidecar_exited code=Some(3)"), "{log}");
    // Il n'y a plus de serveur a attendre : la fermeture sort tout de suite.
    let started = Instant::now();
    close_and_finish(&mut run);
    assert!(started.elapsed() < Duration::from_secs(2));
    assert!(!run.recorder.has_screen(Screen::Stopping));
}

#[test]
fn un_crash_apres_ready_montre_l_ecran_arrete_apres_l_entree() {
    let roots = fresh_roots("crash-ready");
    let env = [("FAKE_VARIANT", "crash_after_ready"), ("FAKE_CRASH_AFTER_MS", "200")];
    let mut run = launch(&roots, &env, short_timeouts());
    run.recorder.wait_screen(Screen::Stopped);
    let calls = run.recorder.calls();
    let entry = calls
        .iter()
        .position(|c| c.is_navigation())
        .expect("entree avant le crash");
    assert!(entry < calls.iter().position(|c| *c == Call::Screen(Screen::Stopped)).unwrap());
    close_and_finish(&mut run);
}

#[test]
fn un_arret_trop_tardif_est_tue_apres_le_budget() {
    let roots = fresh_roots("tardif");
    let timeouts = Timeouts {
        close: Duration::from_secs(1),
        grace: Duration::from_secs(1),
        ..short_timeouts()
    };
    let env = [("FAKE_VARIANT", "late_stopped"), ("FAKE_STOP_DELAY_MS", "30000")];
    let mut run = launch(&roots, &env, timeouts);
    run.recorder.wait_url();
    let started = Instant::now();
    let outcome = close_and_finish(&mut run);
    let elapsed = started.elapsed();
    assert!(outcome.killed, "le serveur etait encore vivant apres le budget");
    assert_eq!(outcome.stopped_clean, None, "aucun stopped n'est arrive");
    assert!(
        elapsed >= Duration::from_secs(1) && elapsed < Duration::from_secs(10),
        "sortie apres {elapsed:?}"
    );
    assert!(!run.stopped_marker(), "tue avant la fin de son arret");
    let log = run.log();
    assert!(
        log.contains("event=kill") && log.contains("event=sidecar_exited"),
        "{log}"
    );
}

#[test]
fn fermer_pendant_le_demarrage_arrete_le_serveur_proprement() {
    let roots = fresh_roots("ferme-au-demarrage");
    let mut run = launch(&roots, &[("FAKE_VARIANT", "never_ready")], short_timeouts());
    run.recorder
        .wait_for("le port annonce", |c| c.iter().any(|x| matches!(x, Call::Port(_))));
    let outcome = close_and_finish(&mut run);
    assert_eq!(outcome.stopped_clean, Some(true));
    assert!(!outcome.killed);
    assert!(run.recorder.has_screen(Screen::Stopping));
}

#[test]
fn le_port_est_repris_au_relancement() {
    let roots = fresh_roots("port-repris");
    let mut first = launch(&roots, &[], short_timeouts());
    let port = port_of(&first.recorder.wait_url());
    close_and_finish(&mut first);

    let mut second = launch(&roots, &[], short_timeouts());
    assert_eq!(port_of(&second.recorder.wait_url()), port);
    close_and_finish(&mut second);
}

#[test]
fn un_serveur_absent_est_une_erreur_de_lancement() {
    use levelup_desktop::logfile::{LogSink, MAX_LOG_BYTES};
    use levelup_desktop::secret::Secret;
    use levelup_desktop::sidecar::SidecarSpec;
    use levelup_desktop::supervisor::{self, ShellConfig};

    let roots = fresh_roots("absent");
    let spec = SidecarSpec {
        exe: roots.resources.join("absent.exe"),
        roots: roots.clone(),
        secret: Secret::generate().unwrap(),
        extra_env: Vec::new(),
    };
    let log =
        std::sync::Arc::new(LogSink::open(levelup_desktop::roots::desktop_log_path(&roots), MAX_LOG_BYTES).unwrap());
    let result = supervisor::start(
        ShellConfig {
            spec,
            timeouts: short_timeouts(),
            log,
        },
        common::Recorder::default(),
    );
    assert!(
        result.is_err(),
        "un serveur introuvable doit etre une erreur, pas un ecran qui attend"
    );
}

/// L21.b : la notification que le serveur ecrit sur stdout arrive a l'hote telle quelle (texte compose
/// par le serveur), apres l'entree, et sans effet sur le reste du cycle de vie.
#[test]
fn une_notification_du_serveur_arrive_a_l_hote_apres_l_entree() {
    let roots = fresh_roots("notification");
    let line = r#"{"event":"notification","title":"Sync complete","body":"Your latest matches are up to date.","kind":"sync"}"#;
    let mut run = launch(&roots, &[("FAKE_EVENTS_AFTER_READY", line)], short_timeouts());

    let calls = run.recorder.wait_for("la notification", |c| {
        c.iter().any(|call| matches!(call, Call::Notify(_)))
    });
    let notice = calls
        .iter()
        .find_map(|c| match c {
            Call::Notify(n) => Some(n.clone()),
            _ => None,
        })
        .expect("notification");
    assert_eq!(notice.title, "Sync complete");
    assert_eq!(notice.body, "Your latest matches are up to date.");
    assert_eq!(notice.kind, "sync");
    let entry = calls
        .iter()
        .position(|c| matches!(c, Call::LoadEntry(_)))
        .expect("entree");
    let shown = calls
        .iter()
        .position(|c| matches!(c, Call::Notify(_)))
        .expect("notification");
    assert!(entry < shown, "la notification vient apres l'entree : {calls:?}");

    let outcome = close_and_finish(&mut run);
    assert_eq!(outcome.stopped_clean, Some(true));
    let log = run.log();
    assert!(log.contains("kind=notification toast_kind=sync"), "{log}");
    assert!(
        !log.contains("Your latest matches"),
        "le texte n'est jamais journalise : {log}"
    );
}

/// Une ligne `notification` mal formee (champ manquant) n'est jamais fatale : journalisee, ignoree.
#[test]
fn une_notification_mal_formee_est_ignoree_sans_effet() {
    let roots = fresh_roots("notification-bad");
    let line = r#"{"event":"notification","title":"t","kind":"sync"}"#;
    let mut run = launch(&roots, &[("FAKE_EVENTS_AFTER_READY", line)], short_timeouts());
    run.wait_log("reason=bad_field:body");
    assert!(!run.recorder.calls().iter().any(|c| matches!(c, Call::Notify(_))));
    let outcome = close_and_finish(&mut run);
    assert_eq!(outcome.stopped_clean, Some(true));
}

/// L21.c : les reglages que le serveur annonce arrivent a l'hote, apres l'entree.
#[test]
fn les_reglages_du_serveur_arrivent_a_l_hote_apres_l_entree() {
    let roots = fresh_roots("shell-settings");
    let line = r#"{"event":"shell_settings","native_notifications":false,"close_to_tray":true,"start_with_windows":false,"locale":"en"}"#;
    let mut run = launch(&roots, &[("FAKE_EVENTS_AFTER_READY", line)], short_timeouts());

    let calls = run.recorder.wait_for("les reglages", |c| {
        c.iter().any(|call| matches!(call, Call::Settings(_)))
    });
    let settings = calls
        .iter()
        .find_map(|c| match c {
            Call::Settings(s) => Some(s.clone()),
            _ => None,
        })
        .expect("reglages");
    assert!(settings.close_to_tray && !settings.native_notifications && !settings.start_with_windows);
    assert_eq!(settings.locale, "en");
    let entry = calls
        .iter()
        .position(|c| matches!(c, Call::LoadEntry(_)))
        .expect("entree");
    let applied = calls
        .iter()
        .position(|c| matches!(c, Call::Settings(_)))
        .expect("reglages");
    assert!(entry < applied, "les reglages viennent apres l'entree : {calls:?}");
    close_and_finish(&mut run);
}

/// L21.c : « Synchroniser maintenant » ecrit la commande `sync` sur la stdin du serveur (une ligne), sans
/// l'arreter ; avant `ready`, rien n'est ecrit. Quitter reste la fermeture propre (stdin ferme, `stopped`).
#[test]
fn la_synchronisation_demandee_atteint_le_serveur_par_sa_stdin() {
    let roots = fresh_roots("sync-command");
    let mut run = launch(&roots, &[], short_timeouts());
    run.recorder.wait_url();
    let closer = run.handle.as_ref().expect("poignee").closer();
    closer.request_sync();
    closer.request_sync();

    let commands = roots.data.join("fake-sidecar-commands.txt");
    let deadline = Instant::now() + common::WAIT;
    loop {
        let text = std::fs::read_to_string(&commands).unwrap_or_default();
        if text.lines().filter(|l| *l == "sync").count() == 2 {
            break;
        }
        assert!(Instant::now() < deadline, "la commande sync n'arrive jamais : {text:?}");
        std::thread::sleep(Duration::from_millis(20));
    }
    run.wait_log("event=sync_requested");

    // La commande n'a pas arrete le serveur : la fermeture reste propre.
    let outcome = close_and_finish(&mut run);
    assert_eq!(outcome.stopped_clean, Some(true));
    assert!(!outcome.killed);
    assert!(run.stopped_marker());
}

/// Une demande de synchronisation avant `ready` n'ecrit rien : le serveur n'est pas pret a l'executer.
#[test]
fn la_synchronisation_demandee_avant_ready_n_ecrit_rien() {
    let roots = fresh_roots("sync-before-ready");
    let mut run = launch(&roots, &[("FAKE_READY_DELAY_MS", "700")], short_timeouts());
    run.handle.as_ref().expect("poignee").closer().request_sync();
    run.recorder.wait_url();
    let outcome = close_and_finish(&mut run);
    assert_eq!(outcome.stopped_clean, Some(true));
    let text = std::fs::read_to_string(roots.data.join("fake-sidecar-commands.txt")).unwrap_or_default();
    assert!(text.is_empty(), "aucune commande avant ready : {text:?}");
}
