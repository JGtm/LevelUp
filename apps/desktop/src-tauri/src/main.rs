//! Le binaire de la coquille : relie la bibliotheque (machine d'etats, sidecar) a la fenetre Tauri.
//!
//! Rien ne se decide ici : la fenetre montre ce que la machine d'etats demande (ecran embarque,
//! entree du serveur, page d'echec du serveur), ferme quand elle le dit, et laisse le web
//! s'afficher. Aucune commande n'est exposee a la page du serveur (aucune capacite accordee).

#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

mod update_tauri;

use std::path::PathBuf;
use std::sync::atomic::{AtomicBool, AtomicU16, Ordering};
use std::sync::{Arc, Mutex};

use levelup_desktop::catalogue;
use levelup_desktop::lifecycle::Timeouts;
use levelup_desktop::logfile::{LogSink, MAX_LOG_BYTES, Source};
use levelup_desktop::navigation::{self, Navigation};
use levelup_desktop::render::{self, SCHEME};
use levelup_desktop::roots::{self, Roots};
use levelup_desktop::screen::Screen;
use levelup_desktop::secret::Secret;
use levelup_desktop::sidecar::SidecarSpec;
use levelup_desktop::supervisor::{self, Closer, Host, ShellConfig};
use levelup_desktop::updater::{CloserSlot, InstallGate, UpdateHost, run_update};
use tauri::webview::DownloadEvent;
use tauri::window::Color;
use tauri::{AppHandle, Manager, RunEvent, WebviewUrl, WebviewWindow, WebviewWindowBuilder, WindowEvent};
use tauri_plugin_opener::OpenerExt;
use update_tauri::{DialogPrompt, PluginSource};
use url::Url;

/// Etiquette de la fenetre principale.
const MAIN_WINDOW: &str = "main";
/// Nom du serveur a cote de l'executable (le bundler retire le suffixe de cible).
const SERVER_EXE: &str = "levelup-server.exe";
/// Fond de la fenetre avant son premier rendu : evite un eclair blanc.
const WINDOW_BACKGROUND: Color = Color(11, 18, 32, 255);

/// Etat partage de l'application.
struct Shell {
    /// Demande de fermeture du superviseur ; absent si le serveur n'a pas pu etre lance.
    closer: Mutex<Option<Closer>>,
    /// La sortie de l'application est ordonnee : plus rien ne la retient.
    exiting: Arc<AtomicBool>,
    /// Journal, pour les ecrans montres avant que le superviseur existe.
    log: Mutex<Option<Arc<LogSink>>>,
}

/// Ce que la mise a jour partage avec le superviseur : la porte d'installation (ouverte a la sortie de
/// l'application) et la poignee de fermeture (posee apres le lancement du superviseur).
struct UpdateWiring {
    gate: InstallGate,
    closer: CloserSlot,
    locale: &'static str,
}

fn main() {
    let locale = catalogue::resolve_locale(&sys_locale::get_locale().unwrap_or_default());
    let app = tauri::Builder::default()
        // Premier plugin : une seconde ouverture est detectee avant tout autre travail.
        .plugin(tauri_plugin_single_instance::init(|app, _args, _cwd| focus_main(app)))
        .plugin(tauri_plugin_window_state::Builder::default().build())
        .plugin(tauri_plugin_opener::init())
        .register_uri_scheme_protocol(SCHEME, move |_ctx, request| {
            let reply = render::respond(request.uri().path(), locale);
            let (status, content_type, body) = match reply {
                Some(page) => (200, page.content_type, page.body),
                None => (404, "text/plain; charset=utf-8", Vec::new()),
            };
            tauri::http::Response::builder()
                .status(status)
                .header("Content-Type", content_type)
                .header("Cache-Control", "no-store")
                .body(body)
                .expect("reponse du protocole embarque")
        })
        .setup(move |app| {
            setup(app.handle());
            Ok(())
        })
        .build(tauri::generate_context!())
        .expect("construction de l'application");

    app.run(|app, event| match event {
        // La croix, Alt+F4 et la barre des taches : la fenetre reste, la fermeture suit son cours.
        RunEvent::WindowEvent {
            label,
            event: WindowEvent::CloseRequested { api, .. },
            ..
        } if label == MAIN_WINDOW && !exiting(app) => {
            api.prevent_close();
            begin_close(app);
        }
        // Une sortie demandee sans fenetre (fin de session) : meme chemin ; `exit(0)` du superviseur passe.
        RunEvent::ExitRequested { api, code, .. } if code.is_none() && !exiting(app) => {
            api.prevent_exit();
            begin_close(app);
        }
        _ => {}
    });
}

fn exiting(app: &AppHandle) -> bool {
    app.try_state::<Shell>()
        .is_none_or(|s| s.exiting.load(Ordering::SeqCst))
}

/// Lance la fermeture propre, ou sort tout de suite s'il n'y a pas de serveur a attendre.
fn begin_close(app: &AppHandle) {
    let Some(shell) = app.try_state::<Shell>() else {
        app.exit(0);
        return;
    };
    let closer = shell.closer.lock().unwrap_or_else(|p| p.into_inner()).clone();
    match closer {
        Some(closer) => closer.request_close(),
        None => {
            shell.exiting.store(true, Ordering::SeqCst);
            app.exit(0);
        }
    }
}

fn focus_main(app: &AppHandle) {
    if let Some(window) = app.get_webview_window(MAIN_WINDOW) {
        // Mettre au premier plan est un effort : une fenetre deja la ne signale rien.
        let _ = window.unminimize();
        let _ = window.show();
        let _ = window.set_focus();
    }
}

/// Ce qui est pret avant de creer la fenetre.
struct Prepared {
    roots: Roots,
    log: Arc<LogSink>,
    secret: Secret,
    exe: PathBuf,
}

fn prepare(app: &AppHandle) -> Result<Prepared, String> {
    let bundled = app
        .path()
        .resource_dir()
        .map_err(|e| format!("dossier des ressources : {e}"))?;
    let local = std::env::var_os("LOCALAPPDATA").map(PathBuf::from);
    let roots = roots::resolve(|name| std::env::var_os(name), local, bundled).map_err(|e| e.to_string())?;
    roots::ensure_dirs(&roots).map_err(|e| format!("creation des dossiers de {} : {e}", roots.data.display()))?;
    let log = LogSink::open(roots::desktop_log_path(&roots), MAX_LOG_BYTES)
        .map(Arc::new)
        .map_err(|e| format!("ouverture du journal : {e}"))?;
    let secret = Secret::generate().map_err(|e| e.to_string())?;
    let exe = std::env::current_exe()
        .map_err(|e| format!("chemin de l'executable : {e}"))?
        .with_file_name(SERVER_EXE);
    Ok(Prepared {
        roots,
        log,
        secret,
        exe,
    })
}

fn setup(app: &AppHandle) {
    let port = Arc::new(AtomicU16::new(0));
    let exiting_flag = Arc::new(AtomicBool::new(false));
    let prepared = prepare(app);
    let log = prepared.as_ref().ok().map(|p| p.log.clone());
    app.manage(Shell {
        closer: Mutex::new(None),
        exiting: exiting_flag.clone(),
        log: Mutex::new(log.clone()),
    });

    if let Err(e) = create_window(app, port.clone(), log.clone()) {
        report(log.as_deref(), &format!("event=window_failed err={e}"));
        app.exit(1);
        return;
    }
    let updater_enabled = register_updater(app, log.as_deref());
    let update = UpdateWiring {
        gate: InstallGate::default(),
        closer: CloserSlot::default(),
        locale: catalogue::resolve_locale(&sys_locale::get_locale().unwrap_or_default()),
    };

    let started = prepared.and_then(|p| start_server(app, p, port, exiting_flag, update, updater_enabled));
    if let Err(e) = started {
        report(log.as_deref(), &format!("event=start_failed err={e}"));
        show(app, Screen::Stopped);
    }
}

fn start_server(
    app: &AppHandle,
    prepared: Prepared,
    port: Arc<AtomicU16>,
    exiting_flag: Arc<AtomicBool>,
    update: UpdateWiring,
    updater_enabled: bool,
) -> Result<(), String> {
    let Prepared {
        roots,
        log,
        secret,
        exe,
    } = prepared;
    let config = ShellConfig {
        spec: SidecarSpec {
            exe,
            roots,
            secret,
            extra_env: Vec::new(),
        },
        timeouts: Timeouts::default(),
        log: log.clone(),
    };
    let host = TauriHost {
        app: app.clone(),
        port,
        exiting: exiting_flag,
        log: log.clone(),
    };
    let UpdateWiring { gate, closer, locale } = update;
    let on_ready = if updater_enabled {
        spawn_update_check(app.clone(), locale, gate.clone(), closer.clone(), log.clone())
    } else {
        Box::new(|| {})
    };
    let host = UpdateHost::new(host, gate, log, on_ready);
    let handle = supervisor::start(config, host).map_err(|e| format!("lancement du serveur : {e}"))?;
    closer.set(handle.closer());
    let shell = app.state::<Shell>();
    *shell.closer.lock().unwrap_or_else(|p| p.into_inner()) = Some(handle.closer());
    Ok(())
}

/// La verification de mise a jour, lancee une fois par lancement quand le serveur est pret : sur son
/// propre fil, jamais bloquante, chaque issue sans installation est journalisee.
fn spawn_update_check(
    app: AppHandle,
    locale: &'static str,
    gate: InstallGate,
    closer: CloserSlot,
    log: Arc<LogSink>,
) -> Box<dyn FnOnce() + Send> {
    Box::new(move || {
        let thread_log = log.clone();
        let spawned = std::thread::Builder::new().name("update".into()).spawn(move || {
            run_update(
                PluginSource::new(app),
                DialogPrompt,
                locale,
                &gate,
                &closer,
                &thread_log,
            );
        });
        if let Err(e) = spawned {
            report(Some(log.as_ref()), &format!("event=update_thread_failed err={e}"));
        }
    })
}

/// L'updater n'est branche que si la cle publique est posee (`build.rs` refuse une release sans elle) et
/// hors des builds de developpement (un build de debug ne se remplace pas par la derniere release).
/// Rend vrai quand le plugin est enregistre.
fn register_updater(app: &AppHandle, log: Option<&LogSink>) -> bool {
    if cfg!(debug_assertions) {
        report(log, "event=updater_disabled reason=debug_build");
        return false;
    }
    let configured = app
        .config()
        .plugins
        .0
        .get("updater")
        .and_then(|conf| conf.get("pubkey"))
        .and_then(|key| key.as_str())
        .is_some_and(|key| !key.trim().is_empty());
    if !configured {
        report(log, "event=updater_disabled reason=no_public_key");
        return false;
    }
    if let Err(e) = app.plugin(tauri_plugin_updater::Builder::new().build()) {
        report(log, &format!("event=updater_failed err={e}"));
        return false;
    }
    true
}

fn create_window(app: &AppHandle, port: Arc<AtomicU16>, log: Option<Arc<LogSink>>) -> tauri::Result<WebviewWindow> {
    let nav_app = app.clone();
    let nav_log = log.clone();
    let download_log = log;
    WebviewWindowBuilder::new(app, MAIN_WINDOW, WebviewUrl::External(embedded_url(Screen::Starting)))
        .title("LevelUp")
        .inner_size(1440.0, 900.0)
        .min_inner_size(1024.0, 640.0)
        .center()
        .background_color(WINDOW_BACKGROUND)
        .on_navigation(move |url| match navigation::decide(url, current_port(&port)) {
            Navigation::Allow => true,
            Navigation::OpenExternally => {
                if let Err(e) = nav_app.opener().open_url(url.as_str(), None::<&str>) {
                    report(nav_log.as_deref(), &format!("event=open_external_failed err={e}"));
                }
                false
            }
            Navigation::Deny => false,
        })
        .on_download(move |_webview, event| on_download(event, download_log.as_deref()))
        .build()
}

fn current_port(port: &AtomicU16) -> Option<u16> {
    match port.load(Ordering::SeqCst) {
        0 => None,
        p => Some(p),
    }
}

/// Telechargements (ADR 0045 D-6) : boite « Enregistrer sous » native, aucun changement cote web.
fn on_download(event: DownloadEvent<'_>, log: Option<&LogSink>) -> bool {
    match event {
        DownloadEvent::Requested { destination, .. } => {
            let mut dialog = rfd::FileDialog::new();
            if let Some(name) = destination.file_name() {
                dialog = dialog.set_file_name(name.to_string_lossy());
            }
            if let Some(dir) = destination.parent() {
                dialog = dialog.set_directory(dir);
            }
            match dialog.save_file() {
                Some(chosen) => {
                    *destination = chosen;
                    true
                }
                None => {
                    report(log, "event=download_cancelled");
                    false
                }
            }
        }
        DownloadEvent::Finished { success, .. } => {
            report(log, &format!("event=download_finished success={success}"));
            true
        }
        _ => true,
    }
}

/// L'hote que pilote le superviseur : la fenetre principale.
struct TauriHost {
    app: AppHandle,
    port: Arc<AtomicU16>,
    exiting: Arc<AtomicBool>,
    log: Arc<LogSink>,
}

impl Host for TauriHost {
    fn show_screen(&mut self, screen: Screen) {
        self.navigate(embedded_url(screen));
    }

    fn load_url(&mut self, url: &str) {
        match Url::parse(url) {
            Ok(url) => self.navigate(url),
            // Jamais l'URL dans le journal : elle peut porter le secret d'entree.
            Err(e) => report(Some(self.log.as_ref()), &format!("event=bad_url err={e}")),
        }
    }

    fn load_entry(&mut self, url: &str) {
        // Navigation de la barre d'adresse, PAS `location.replace` depuis la page embarquee : mesure faite,
        // une navigation lancee par un script de l'origine embarquee est inter-sites pour le serveur, et le
        // cookie SameSite=Strict qu'il pose n'est pas renvoye a la redirection vers `/` (401). L'historique
        // de WebView2 inscrit l'URL dans les deux cas ; le secret etant a usage unique cote serveur, cette
        // trace est morte (ADR 0045 D-3). Jamais l'URL dans le journal.
        match Url::parse(url) {
            Ok(url) => self.navigate(url),
            Err(e) => report(Some(self.log.as_ref()), &format!("event=bad_entry_url err={e}")),
        }
    }

    fn announce_port(&mut self, port: u16) {
        self.port.store(port, Ordering::SeqCst);
    }

    fn exit_app(&mut self) {
        self.exiting.store(true, Ordering::SeqCst);
        self.app.exit(0);
    }
}

impl TauriHost {
    fn navigate(&self, url: Url) {
        match self.app.get_webview_window(MAIN_WINDOW) {
            Some(window) => {
                if let Err(e) = window.navigate(url) {
                    report(Some(self.log.as_ref()), &format!("event=navigate_failed err={e}"));
                }
            }
            None => report(Some(self.log.as_ref()), "event=navigate_skipped reason=no_window"),
        }
    }
}

/// Montre un ecran embarque depuis `setup`, avant que le superviseur existe.
fn show(app: &AppHandle, screen: Screen) {
    let log = app
        .state::<Shell>()
        .log
        .lock()
        .unwrap_or_else(|p| p.into_inner())
        .clone();
    if let Some(window) = app.get_webview_window(MAIN_WINDOW)
        && let Err(e) = window.navigate(embedded_url(screen))
    {
        report(log.as_deref(), &format!("event=navigate_failed err={e}"));
    }
}

/// URL d'un ecran embarque. Windows adresse un protocole personnalise par `http://<nom>.localhost`.
fn embedded_url(screen: Screen) -> Url {
    let base = if cfg!(windows) {
        format!("http://{SCHEME}.localhost")
    } else {
        format!("{SCHEME}://localhost")
    };
    Url::parse(&format!("{base}{}", screen.path())).expect("URL d'ecran embarque valide")
}

/// Journalise dans `desktop.log` quand il est ouvert, sinon sur la sortie d'erreur.
fn report(log: Option<&LogSink>, text: &str) {
    match log {
        Some(log) => log.write_line(Source::Shell, text),
        None => eprintln!("{text}"),
    }
}
