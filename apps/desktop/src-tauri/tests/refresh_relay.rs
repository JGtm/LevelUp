//! Le relais de la tache planifiee (`LevelUp.exe --refresh-only`) contre le faux serveur : le code de sortie du
//! serveur est celui du relais, et un echec de lancement laisse sa trace dans le journal de la coquille.

mod common;

use std::fs;

use levelup_desktop::logfile::{LogSink, MAX_LOG_BYTES};
use levelup_desktop::refresh_only::{FAILURE_CODE, relay};
use levelup_desktop::roots::{self, Roots};

fn fresh_roots(name: &str) -> Roots {
    let base = std::env::temp_dir().join(format!("levelup-refresh-relay-{name}-{}", std::process::id()));
    let _ = fs::remove_dir_all(&base);
    let roots = Roots {
        resources: base.join("res"),
        data: base.join("data"),
        logs: base.join("data").join("logs"),
        web_dist: None,
    };
    fs::create_dir_all(&roots.resources).unwrap();
    roots::ensure_dirs(&roots).unwrap();
    roots
}

fn open_log(roots: &Roots) -> LogSink {
    LogSink::open(roots::desktop_log_path(roots), MAX_LOG_BYTES).expect("journal")
}

fn journal(roots: &Roots) -> String {
    fs::read_to_string(roots::desktop_log_path(roots)).expect("lecture du journal")
}

#[test]
fn le_code_de_sortie_du_serveur_est_celui_du_relais() {
    for expected in [0, 7] {
        let roots = fresh_roots(&format!("code{expected}"));
        fs::write(roots.data.join("fake-refresh-exit.txt"), expected.to_string()).unwrap();
        let log = open_log(&roots);

        assert_eq!(relay(common::fake_sidecar(), &roots, Some(&log)), expected);

        // Le serveur a bien tourne : dans la racine des ressources, avec les racines, sans secret.
        let ran = fs::read_to_string(roots.data.join("fake-refresh-ran.txt")).expect("temoin du faux serveur");
        assert!(
            ran.contains(&format!("resources={}", roots.resources.display())),
            "{ran}"
        );
        assert!(ran.contains(&format!("data={}", roots.data.display())), "{ran}");
        assert!(ran.contains("secret_set=false"), "{ran}");
        assert!(
            journal(&roots).contains(&format!("fin, code de sortie {expected}")),
            "{}",
            journal(&roots)
        );
    }
}

#[test]
fn un_echec_de_lancement_ecrit_sa_ligne_dans_le_journal_de_la_coquille() {
    let roots = fresh_roots("absent");
    let log = open_log(&roots);
    let absent = roots.resources.join("absent.exe");

    assert_eq!(relay(&absent, &roots, Some(&log)), FAILURE_CODE);

    let text = journal(&roots);
    assert!(text.contains("refresh-only : debut"), "{text}");
    assert!(text.contains("lancement de") && text.contains("impossible"), "{text}");
    assert!(text.contains(&format!("fin, code de sortie {FAILURE_CODE}")), "{text}");
}
