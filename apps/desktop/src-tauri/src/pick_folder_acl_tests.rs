//! Les permissions de la commande `pick_folder` (capabilities/pick-folder.json), contre le runtime factice de tauri.
//!
//! La page du serveur est une origine DISTANTE (`http://127.0.0.1:<port>`) : tauri refuse tout appel venu d'une
//! telle origine sans capacite explicite. Ces tests lisent les permissions que `build.rs` et la capacite
//! embarquent dans le binaire (`generate_context!`) et verifient la forme exacte de ce qui est accorde : cette
//! commande, la fenetre `main`, la boucle locale seulement, ni http ailleurs ni les ecrans embarques.
//!
//! La commande enregistree ici n'est qu'une doublure du MEME NOM : la vraie ouvre une boite modale. Seul le nom
//! compte pour les permissions.
//!
//! Ce module vit dans le binaire et non sous `tests/` : tauri-build n'embarque le manifeste Windows
//! (Common-Controls v6, requis par la fenetre) que dans les binaires, un executable de test d'integration qui
//! lie tauri ne demarre pas (STATUS_ENTRYPOINT_NOT_FOUND).

use tauri::ipc::{CallbackFn, InvokeBody};
use tauri::test::{INVOKE_KEY, get_ipc_response, mock_builder};
use tauri::webview::InvokeRequest;
use tauri::{App, WebviewUrl, WebviewWindow, WebviewWindowBuilder};

#[tauri::command]
fn pick_folder(initial: Option<String>) -> Option<String> {
    initial
}

fn app_with_windows() -> (
    App<tauri::test::MockRuntime>,
    WebviewWindow<tauri::test::MockRuntime>,
    WebviewWindow<tauri::test::MockRuntime>,
) {
    let app = mock_builder()
        .invoke_handler(tauri::generate_handler![pick_folder])
        .build(tauri::generate_context!())
        .expect("construction de l'application factice");
    let page = WebviewUrl::External("http://127.0.0.1:8123/".parse().unwrap());
    let main = WebviewWindowBuilder::new(&app, "main", page.clone())
        .build()
        .expect("fenetre main");
    let other = WebviewWindowBuilder::new(&app, "autre", page)
        .build()
        .expect("fenetre autre");
    (app, main, other)
}

fn call(
    window: &WebviewWindow<tauri::test::MockRuntime>,
    cmd: &str,
    origin: &str,
) -> Result<serde_json::Value, serde_json::Value> {
    let request = InvokeRequest {
        cmd: cmd.into(),
        callback: CallbackFn(0),
        error: CallbackFn(1),
        url: origin.parse().expect("origine valide"),
        body: InvokeBody::Json(serde_json::json!({ "initial": "choisi" })),
        headers: Default::default(),
        invoke_key: INVOKE_KEY.to_string(),
    };
    get_ipc_response(window, request).map(|body| body.deserialize::<serde_json::Value>().unwrap())
}

#[test]
fn la_page_du_serveur_appelle_pick_folder_depuis_la_boucle_locale_sur_la_fenetre_main() {
    let (_app, main, _other) = app_with_windows();
    for origin in [
        "http://127.0.0.1:8123",
        "http://127.0.0.1:8123/",
        "http://127.0.0.1:1",
        "http://127.0.0.1:65535",
    ] {
        assert_eq!(
            call(&main, "pick_folder", origin),
            Ok(serde_json::json!("choisi")),
            "origine {origin}"
        );
    }
}

#[test]
fn une_autre_origine_est_refusee() {
    let (_app, main, _other) = app_with_windows();
    for origin in [
        "http://localhost:8123",
        "https://127.0.0.1:8123",
        "http://127.0.0.2:8123",
        "http://127.0.0.1.exemple.test:8123",
        "http://exemple.test",
        "https://exemple.test:8123",
    ] {
        assert!(call(&main, "pick_folder", origin).is_err(), "origine {origin} acceptee");
    }
}

#[test]
fn les_ecrans_embarques_de_la_coquille_n_ont_aucune_commande() {
    let (_app, main, _other) = app_with_windows();
    let embedded = if cfg!(windows) {
        "http://levelup-shell.localhost/stopped"
    } else {
        "levelup-shell://localhost/stopped"
    };
    let tauri_local = if cfg!(any(windows, target_os = "android")) {
        "http://tauri.localhost"
    } else {
        "tauri://localhost"
    };
    for origin in [embedded, tauri_local] {
        assert!(call(&main, "pick_folder", origin).is_err(), "origine {origin} acceptee");
    }
}

#[test]
fn une_autre_fenetre_est_refusee() {
    let (_app, _main, other) = app_with_windows();
    assert!(call(&other, "pick_folder", "http://127.0.0.1:8123").is_err());
}

#[test]
fn aucune_autre_commande_n_est_accordee_a_la_page() {
    let (_app, main, _other) = app_with_windows();
    for cmd in [
        "open_url",
        "plugin:opener|open_url",
        "plugin:notification|notify",
        "plugin:updater|check",
    ] {
        assert!(
            call(&main, cmd, "http://127.0.0.1:8123").is_err(),
            "commande {cmd} acceptee"
        );
    }
}
