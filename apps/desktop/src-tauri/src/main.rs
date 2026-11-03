//! Le binaire de la coquille : relie la bibliotheque (machine d'etats, sidecar) a la fenetre Tauri.
//!
//! Rien ne se decide ici : la fenetre montre ce que la machine d'etats demande (ecran embarque,
//! entree du serveur, page d'echec du serveur), ferme quand elle le dit, et laisse le web
//! s'afficher. Aucune commande n'est exposee a la page du serveur (aucune capacite accordee).

#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

mod autostart_tauri;
mod shell_tauri;
mod toast_tauri;
mod tray_tauri;
mod update_setup;
mod update_tauri;

use std::path::PathBuf;
use std::sync::atomic::{AtomicBool, AtomicU16, Ordering};
use std::sync::{Arc, Mutex};

use levelup_desktop::autostart;
use levelup_desktop::catalogue;
use levelup_desktop::events::Toast;
use levelup_desktop::launch::{self, InitialWindow, MINIMIZED_ARG};
use levelup_desktop::lifecycle::Timeouts;
use levelup_desktop::logfile::{LogSink, MAX_LOG_BYTES, Source};
use levelup_desktop::native_notice;
use levelup_desktop::navigation::{self, ExternalOpen, Navigation, NewWindowDecision, OPEN_EXTERNAL_ENV};
use levelup_desktop::render::{self, SCHEME};
use levelup_desktop::roots::{self, Roots};
use levelup_desktop::screen::Screen;
use levelup_desktop::secret::Secret;
use levelup_desktop::shell_settings::{CloseAction, ShellSettings, ShellState};
use levelup_desktop::sidecar::SidecarSpec;
use levelup_desktop::supervisor::{self, Closer, Host, Outcome, ShellConfig};
use levelup_desktop::updater::UpdateHost;
use tauri::webview::{DownloadEvent, NewWindowResponse};
use tauri::window::Color;
use tauri::{AppHandle, Manager, RunEvent, WebviewUrl, WebviewWindow, WebviewWindowBuilder, WindowEvent};
use tauri_plugin_opener::OpenerExt;
use toast_tauri::{PluginNotifier, window_state};
use update_setup::{UpdateWiring, request_update_check, spawn_update_check};
use update_tauri::DialogNotifier;
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
    /// L'etat maximise de la fenetre n'a pas ete restaure au lancement (demarrage cache) : il l'est a la premiere
    /// montre de la fenetre (`launch::restores_maximized_when_first_shown`).
    restore_maximized_on_show: AtomicBool,
    /// Les reglages annonces par le serveur et l'icone de la zone de notification : ce qui decide ce que
    /// fait la croix (`levelup_desktop::shell_settings`).
    shell: Mutex<ShellState>,
}

fn main() {
    let locale = catalogue::resolve_locale(&sys_locale::get_locale().unwrap_or_default());
    // Demarrage avec Windows (L21.d) : l'entree de demarrage ajoute `--minimized`, la fenetre reste cachee.
    let initial = launch::initial_window(launch::started_minimized(std::env::args()));
    let app = tauri::Builder::default()
        // Premier plugin : une seconde ouverture est detectee avant tout autre travail. Une seconde ouverture
        // reduite (l'entree de demarrage alors que l'application tourne deja) ne montre rien.
        .plugin(tauri_plugin_single_instance::init(|app, args, _cwd| {
            if !launch::started_minimized(&args) {
                focus_main(app);
            }
        }))
        // La visibilite est decidee par la coquille (zone de notification, demarrage reduit) : le plugin
        // ne la memorise ni ne la restaure, sinon il remontrerait une fenetre cachee au lancement.
        .plugin(
            tauri_plugin_window_state::Builder::default()
                .with_state_flags(shell_tauri::window_state_flags(initial))
                .build(),
        )
        // Entree de demarrage avec Windows (L21.d) : par utilisateur, jamais ecrite tant que le reglage
        // `start_with_windows` n'est pas actif (`autostart::reconcile`).
        .plugin(
            tauri_plugin_autostart::Builder::new()
                .app_name(autostart::ENTRY_NAME)
                .arg(MINIMIZED_ARG)
                .build(),
        )
        // Notifications Windows relayees depuis la cloche (L21.b) : montrees depuis Rust seulement, la page
        // du serveur n'a aucune commande a appeler.
        .plugin(tauri_plugin_notification::init())
        // Le script que le plugin injecte par defaut attrape les clics sur `target="_blank"` et appelle la commande
        // `open_url` par le pont IPC, que la page du serveur n'a pas le droit d'appeler (aucune capacite) : le clic
        // etait avale sans rien ouvrir. Sans lui, le clic arrive au moteur, qui demande une nouvelle fenetre :
        // `on_new_window` (create_window) l'envoie au navigateur par `open_external`.
        .plugin(
            tauri_plugin_opener::Builder::new()
                .open_js_links_on_click(false)
                .build(),
        )
        .register_uri_scheme_protocol(SCHEME, move |ctx, request| {
            let reply = render::reply(request.uri().path(), locale, || {
                request_update_check(ctx.app_handle().clone());
            });
            tauri::http::Response::builder()
                .status(reply.status)
                .header("Content-Type", reply.content_type)
                .header("Cache-Control", "no-store")
                .body(reply.body)
                .expect("reponse du protocole embarque")
        })
        .setup(move |app| {
            setup(app.handle(), initial);
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
            match shell_tauri::close_action(app) {
                CloseAction::HideToTray => shell_tauri::hide_main(app),
                CloseAction::Quit => begin_close(app),
            }
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
        shell_tauri::restore_maximized_if_pending(app, &window);
        let _ = window.show();
        let _ = window.set_focus();
        // Apres un demarrage reduit, montrer la fenetre peut retirer l'icone de la zone de notification.
        shell_tauri::window_shown(app);
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

fn setup(app: &AppHandle, initial: InitialWindow) {
    let port = Arc::new(AtomicU16::new(0));
    let exiting_flag = Arc::new(AtomicBool::new(false));
    let prepared = prepare(app);
    let log = prepared.as_ref().ok().map(|p| p.log.clone());
    app.manage(Shell {
        closer: Mutex::new(None),
        exiting: exiting_flag.clone(),
        restore_maximized_on_show: AtomicBool::new(launch::restores_maximized_when_first_shown(initial)),
        log: Mutex::new(log.clone()),
        shell: Mutex::new(ShellState::with_system_locale(
            sys_locale::get_locale().unwrap_or_default(),
        )),
    });

    if let Err(e) = create_window(app, port.clone(), log.clone(), initial) {
        report(log.as_deref(), &format!("event=window_failed err={e}"));
        app.exit(1);
        return;
    }
    // Demarrage reduit : la fenetre est cachee, l'icone de la zone de notification existe des maintenant.
    if initial == InitialWindow::HiddenInTray {
        shell_tauri::start_minimized(app);
    }
    let (update, updater_enabled) = update_setup::install(app, log.clone());

    let started = prepared.and_then(|p| start_server(app, p, port, exiting_flag, update, updater_enabled));
    if let Err(e) = started {
        report(log.as_deref(), &format!("event=start_failed err={e}"));
        // Demarrage reduit : la fenetre est cachee, l'ecran d'arret doit etre vu.
        shell_tauri::reveal_if_hidden(app);
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
    let (gate, closer, locale) = (update.gate.clone(), update.closer.clone(), update.locale);
    let on_ready = if updater_enabled {
        spawn_update_check(app.clone(), update, log.clone())
    } else {
        Box::new(|| {})
    };
    let host = UpdateHost::new(host, gate, log, on_ready, Box::new(DialogNotifier { locale }));
    let handle = supervisor::start(config, host).map_err(|e| format!("lancement du serveur : {e}"))?;
    closer.set(handle.closer());
    let shell = app.state::<Shell>();
    *shell.closer.lock().unwrap_or_else(|p| p.into_inner()) = Some(handle.closer());
    Ok(())
}

fn create_window(
    app: &AppHandle,
    port: Arc<AtomicU16>,
    log: Option<Arc<LogSink>>,
    initial: InitialWindow,
) -> tauri::Result<WebviewWindow> {
    let nav_app = app.clone();
    let nav_log = log.clone();
    let nav_port = port.clone();
    let new_window_app = app.clone();
    let new_window_log = log.clone();
    let download_log = log;
    WebviewWindowBuilder::new(app, MAIN_WINDOW, WebviewUrl::External(embedded_url(Screen::Starting)))
        .title("LevelUp")
        .inner_size(1440.0, 900.0)
        .min_inner_size(1024.0, 640.0)
        .center()
        .background_color(WINDOW_BACKGROUND)
        .visible(initial == InitialWindow::Visible)
        .on_navigation(move |url| match navigation::decide(url, current_port(&nav_port)) {
            Navigation::Allow => true,
            Navigation::OpenExternally => {
                open_external(&nav_app, url, nav_log.as_deref());
                false
            }
            Navigation::Deny => false,
        })
        // `target="_blank"` et `window.open` : l'application n'a qu'une fenetre. Sans ce filtre le
        // moteur en ouvrirait une seconde, hors de la politique de navigation.
        .on_new_window(move |url, _features| {
            let decision = navigation::decide_new_window(&url, current_port(&port));
            let verdict = match decision {
                NewWindowDecision::OpenExternally => "open",
                NewWindowDecision::Deny => "deny",
            };
            let origin = navigation::origin_for_log(&url);
            let log = new_window_log.as_deref();
            report(log, &format!("event=new_window url={origin} decision={verdict}"));
            if decision == NewWindowDecision::OpenExternally {
                open_external(&new_window_app, &url, log);
            }
            NewWindowResponse::Deny
        })
        .on_download(move |_webview, event| on_download(event, download_log.as_deref()))
        .build()
}

/// Envoie un lien dans le navigateur par defaut. Le journal ne porte que l'origine : l'URL entiere
/// peut porter un code (lien de connexion). `LEVELUP_SHELL_OPEN_EXTERNAL=log` remplace l'ouverture
/// par le journal seul, pour les essais automatises qui ne doivent pas ouvrir le navigateur du poste.
fn open_external(app: &AppHandle, url: &Url, log: Option<&LogSink>) {
    let origin = navigation::origin_for_log(url);
    let env = std::env::var(OPEN_EXTERNAL_ENV).ok();
    if navigation::external_open_mode(env.as_deref()) == ExternalOpen::LogOnly {
        report(log, &format!("event=open_external origin={origin} mode=log"));
        return;
    }
    match app.opener().open_url(url.as_str(), None::<&str>) {
        Ok(()) => report(log, &format!("event=open_external origin={origin}")),
        Err(e) => report(log, &format!("event=open_external_failed origin={origin} err={e}")),
    }
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

    fn exit_app(&mut self, _outcome: Outcome) {
        self.exiting.store(true, Ordering::SeqCst);
        self.app.exit(0);
    }

    fn notify(&mut self, toast: &Toast) {
        let window = window_state(&self.app, &self.log);
        let mut notifier = PluginNotifier { app: self.app.clone() };
        native_notice::deliver(window, toast, &mut notifier, &self.log);
    }

    fn apply_shell_settings(&mut self, settings: &ShellSettings) {
        shell_tauri::apply_shell_settings(&self.app, settings);
    }

    fn reveal_window_if_hidden(&mut self) {
        shell_tauri::reveal_if_hidden(&self.app);
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
