//! Outils des tests de contrat : le faux serveur compile une fois, un hote qui enregistre, un
//! client HTTP minimal. Aucune fenetre, aucun tauri.

#![allow(dead_code)] // chaque fichier de test n'utilise qu'une partie de ces outils

use std::fs;
use std::io::{Read, Write};
use std::net::TcpStream;
use std::path::{Path, PathBuf};
use std::sync::atomic::{AtomicU32, Ordering};
use std::sync::{Arc, Condvar, Mutex, OnceLock};
use std::time::{Duration, Instant};

use levelup_desktop::lifecycle::Timeouts;
use levelup_desktop::logfile::{LogSink, MAX_LOG_BYTES};
use levelup_desktop::roots::{self, Roots};
use levelup_desktop::screen::Screen;
use levelup_desktop::secret::Secret;
use levelup_desktop::sidecar::SidecarSpec;
use levelup_desktop::supervisor::{self, Host, Outcome, ShellConfig, ShellHandle};

/// Delai maximal d'attente d'un evenement dans les tests (jamais atteint si tout va bien).
pub const WAIT: Duration = Duration::from_secs(60);

/// Compile le faux serveur (une fois par processus de test) et rend le chemin de l'executable.
/// `go build` dans un dossier temporaire de la cible : le depot ne recoit aucun binaire.
pub fn fake_sidecar() -> &'static Path {
    static BUILT: OnceLock<PathBuf> = OnceLock::new();
    BUILT.get_or_init(|| {
        let source = Path::new(env!("CARGO_MANIFEST_DIR")).join("../testdata/fake-sidecar");
        let out_dir = Path::new(env!("CARGO_TARGET_TMPDIR")).join("fake-sidecar");
        fs::create_dir_all(&out_dir).expect("dossier du faux serveur");
        let exe = out_dir.join("fake-sidecar.exe");
        let output = std::process::Command::new("go")
            .args(["build", "-o"])
            .arg(&exe)
            .arg(".")
            .current_dir(&source)
            .env("GOWORK", "off")
            .env("CGO_ENABLED", "0")
            .output()
            .expect("`go` introuvable : les tests de contrat compilent le faux serveur");
        assert!(
            output.status.success(),
            "go build du faux serveur : {}{}",
            String::from_utf8_lossy(&output.stdout),
            String::from_utf8_lossy(&output.stderr)
        );
        exe
    })
}

/// Ce que l'hote a recu.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum Call {
    Screen(Screen),
    LoadUrl(String),
    /// L'URL d'entree (porte le secret).
    LoadEntry(String),
    Port(u16),
    Exit,
}

impl Call {
    /// Vrai pour tout chargement d'une URL du serveur, entree ou page d'echec.
    pub fn is_navigation(&self) -> bool {
        matches!(self, Call::LoadUrl(_) | Call::LoadEntry(_))
    }
}

type Shared = Arc<(Mutex<Vec<Call>>, Condvar)>;

/// Un hote qui enregistre tout ce que la machine lui demande.
#[derive(Clone, Default)]
pub struct Recorder(Shared);

impl Recorder {
    fn push(&self, call: Call) {
        let (calls, changed) = &*self.0;
        calls.lock().unwrap().push(call);
        changed.notify_all();
    }

    pub fn calls(&self) -> Vec<Call> {
        self.0.0.lock().unwrap().clone()
    }

    /// Attend qu'`accept` soit vrai des appels recus ; panique avec le detail apres `WAIT`.
    pub fn wait_for(&self, what: &str, accept: impl Fn(&[Call]) -> bool) -> Vec<Call> {
        let deadline = Instant::now() + WAIT;
        let (calls, changed) = &*self.0;
        let mut guard = calls.lock().unwrap();
        while !accept(&guard) {
            let left = deadline.saturating_duration_since(Instant::now());
            assert!(
                !left.is_zero(),
                "delai depasse en attendant : {what}\nappels : {:#?}",
                *guard
            );
            guard = changed.wait_timeout(guard, left).unwrap().0;
        }
        guard.clone()
    }

    /// L'URL de la premiere navigation vers le serveur.
    pub fn wait_url(&self) -> String {
        let calls = self.wait_for("une URL du serveur", |c| c.iter().any(Call::is_navigation));
        calls
            .into_iter()
            .find_map(|c| match c {
                Call::LoadUrl(url) | Call::LoadEntry(url) => Some(url),
                _ => None,
            })
            .expect("URL")
    }

    pub fn wait_screen(&self, screen: Screen) {
        self.wait_for(&format!("l'ecran {screen:?}"), |c| c.contains(&Call::Screen(screen)));
    }

    pub fn wait_exit(&self) {
        self.wait_for("la sortie de l'application", |c| c.contains(&Call::Exit));
    }

    pub fn has_screen(&self, screen: Screen) -> bool {
        self.calls().contains(&Call::Screen(screen))
    }
}

impl Host for Recorder {
    fn show_screen(&mut self, screen: Screen) {
        self.push(Call::Screen(screen));
    }
    fn load_url(&mut self, url: &str) {
        self.push(Call::LoadUrl(url.to_string()));
    }
    fn load_entry(&mut self, url: &str) {
        self.push(Call::LoadEntry(url.to_string()));
    }
    fn announce_port(&mut self, port: u16) {
        self.push(Call::Port(port));
    }
    fn exit_app(&mut self) {
        self.push(Call::Exit);
    }
}

/// Un lancement : racines jetables, journal, secret, hote enregistreur.
pub struct Run {
    pub roots: Roots,
    pub secret: String,
    pub recorder: Recorder,
    pub handle: Option<ShellHandle>,
}

static RUN_COUNTER: AtomicU32 = AtomicU32::new(0);

/// Racines jetables et neuves pour un test.
pub fn fresh_roots(name: &str) -> Roots {
    let id = RUN_COUNTER.fetch_add(1, Ordering::SeqCst);
    let base = Path::new(env!("CARGO_TARGET_TMPDIR"))
        .join("runs")
        .join(format!("{name}-{}-{id}", std::process::id()));
    let _ = fs::remove_dir_all(&base);
    let roots = Roots {
        resources: base.join("resources"),
        data: base.join("data"),
        logs: base.join("data").join("logs"),
        web_dist: None,
    };
    fs::create_dir_all(&roots.resources).expect("ressources");
    roots::ensure_dirs(&roots).expect("donnees");
    roots
}

/// Delais courts par defaut : aucun test ne depend des 120 s ni des 45 s du contrat.
pub fn short_timeouts() -> Timeouts {
    Timeouts {
        startup: Duration::from_secs(30),
        close: Duration::from_secs(20),
        grace: Duration::from_secs(5),
    }
}

/// Lance le faux serveur sur `roots` ; `fake_env` regle ses variantes (FAKE_*).
pub fn launch(roots: &Roots, fake_env: &[(&str, &str)], timeouts: Timeouts) -> Run {
    let recorder = Recorder::default();
    let secret = Secret::generate().expect("secret");
    let exposed = secret.expose().to_string();
    let log = Arc::new(LogSink::open(roots::desktop_log_path(roots), MAX_LOG_BYTES).expect("journal"));
    let spec = SidecarSpec {
        exe: fake_sidecar().to_path_buf(),
        roots: roots.clone(),
        secret,
        extra_env: fake_env.iter().map(|(k, v)| (k.to_string(), v.to_string())).collect(),
    };
    let handle =
        supervisor::start(ShellConfig { spec, timeouts, log }, recorder.clone()).expect("lancement du faux serveur");
    Run {
        roots: roots.clone(),
        secret: exposed,
        recorder,
        handle: Some(handle),
    }
}

impl Run {
    pub fn request_close(&self) {
        self.handle.as_ref().expect("poignee").request_close();
    }

    /// Attend la fin du superviseur.
    pub fn finish(&mut self) -> Outcome {
        self.handle.take().expect("poignee").wait()
    }

    pub fn log(&self) -> String {
        fs::read_to_string(roots::desktop_log_path(&self.roots)).unwrap_or_default()
    }

    /// Attend qu'une ligne du journal contienne `needle` ; rend le journal.
    pub fn wait_log(&self, needle: &str) -> String {
        let deadline = Instant::now() + WAIT;
        loop {
            let log = self.log();
            if log.contains(needle) {
                return log;
            }
            assert!(
                Instant::now() < deadline,
                "delai depasse : le journal ne dit jamais {needle:?}\n{log}"
            );
            std::thread::sleep(Duration::from_millis(20));
        }
    }

    pub fn stopped_marker(&self) -> bool {
        self.roots.data.join("fake-sidecar-stopped.txt").exists()
    }
}

/// Le port d'une URL `http://127.0.0.1:<port>/...`.
pub fn port_of(url: &str) -> u16 {
    url.strip_prefix("http://127.0.0.1:")
        .and_then(|rest| rest.split('/').next())
        .and_then(|p| p.parse().ok())
        .unwrap_or_else(|| panic!("URL du serveur inattendue : {url}"))
}

/// Reponse HTTP minimale.
pub struct Reply {
    pub status: u16,
    pub head: String,
}

/// `GET path` sur le port local, sans suivre les redirections (Host = l'autorite exacte).
pub fn http_get(port: u16, path: &str) -> Reply {
    let mut stream = TcpStream::connect(("127.0.0.1", port)).expect("connexion au faux serveur");
    stream.set_read_timeout(Some(Duration::from_secs(10))).unwrap();
    write!(
        stream,
        "GET {path} HTTP/1.1\r\nHost: 127.0.0.1:{port}\r\nConnection: close\r\n\r\n"
    )
    .unwrap();
    let mut raw = String::new();
    stream.read_to_string(&mut raw).expect("reponse");
    let head = raw.split("\r\n\r\n").next().unwrap_or_default().to_string();
    let status = head
        .split_whitespace()
        .nth(1)
        .and_then(|s| s.parse().ok())
        .expect("code HTTP");
    Reply { status, head }
}

/// Chemin normalise pour comparer ce que dit le faux serveur et ce que la coquille a pose.
pub fn normalized(path: &Path) -> String {
    let canonical = fs::canonicalize(path).unwrap_or_else(|_| path.to_path_buf());
    canonical
        .to_string_lossy()
        .trim_start_matches(r"\\?\")
        .replace('\\', "/")
        .to_lowercase()
}
