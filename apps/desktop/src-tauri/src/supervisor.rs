//! Le superviseur : relie la machine d'etats ([`crate::lifecycle`]) au serveur reel.
//!
//! Quatre fils : un vide la sortie standard du serveur et la decode (A14 : un tube plein bloque
//! le serveur), un relaie sa sortie d'erreur vers `desktop.log`, un surveille sa sortie, un
//! execute la machine (entrees par un canal, minuteries par `recv_timeout`). La fenetre n'est
//! connue que par le trait [`Host`] : les tests branchent un hote qui enregistre, `main.rs` un hote
//! qui pilote la vraie fenetre.

use std::io::{self, BufReader};
use std::process::ExitStatus;
use std::sync::Arc;
use std::sync::mpsc::{self, RecvTimeoutError, Sender};
use std::thread::{self, JoinHandle};
use std::time::{Duration, Instant};

use crate::events::{ParseError, SidecarEvent, parse_line};
use crate::lifecycle::{Action, Input, Lifecycle, Timeouts, Timer};
use crate::linereader::read_line_bounded;
use crate::logfile::{LogSink, Source};
use crate::screen::Screen;
use crate::secret::Secret;
use crate::sidecar::{self, Sidecar, SidecarSpec, Spawned};

/// Longueur maximale d'une ligne lue sur les sorties du serveur.
const MAX_LINE_BYTES: usize = 64 * 1024;
/// Nombre de lignes de stdout non decodables journalisees ; les suivantes sont seulement comptees
/// (un serveur bavard ne doit pas remplir le journal).
const LOGGED_IGNORED_LINES: u64 = 20;
/// Longueur d'une ligne ignoree reproduite dans le journal.
const LOGGED_LINE_CHARS: usize = 200;
/// Periode de surveillance de la sortie du processus.
const WAIT_POLL: Duration = Duration::from_millis(20);
/// Temps laisse a la sortie standard pour se vider apres la sortie du processus (un petit-fils
/// qui garderait le tube ouvert ne retient pas la coquille).
const STDOUT_DRAIN_GRACE: Duration = Duration::from_secs(2);

/// Ce que la machine demande a la fenetre.
pub trait Host: Send + 'static {
    /// Montrer un ecran embarque.
    fn show_screen(&mut self, screen: Screen);
    /// Charger une URL du serveur local (sa page d'echec). Ne porte aucun secret.
    fn load_url(&mut self, url: &str);
    /// Charger l'URL d'entree du serveur, qui PORTE le secret d'entree : ne jamais la journaliser.
    fn load_entry(&mut self, url: &str);
    /// Le port du serveur est connu.
    fn announce_port(&mut self, port: u16);
    /// Terminer l'application.
    fn exit_app(&mut self);
}

pub struct ShellConfig {
    pub spec: SidecarSpec,
    pub timeouts: Timeouts,
    pub log: Arc<LogSink>,
}

/// Ce que le superviseur sait a la fin.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct Outcome {
    /// `clean` du dernier `stopped` recu.
    pub stopped_clean: Option<bool>,
    /// La coquille a du tuer le serveur.
    pub killed: bool,
}

/// Poignee de demande de fermeture, copiable entre fils (l'etat de l'application, les gestionnaires
/// de fenetre).
#[derive(Clone)]
pub struct Closer(Sender<Input>);

impl Closer {
    /// Demande la fermeture : ecran, fermeture de stdin, attente de `stopped`, kill si besoin.
    pub fn request_close(&self) {
        // Un superviseur deja termine n'a plus rien a fermer.
        let _ = self.0.send(Input::CloseRequested);
    }
}

pub struct ShellHandle {
    closer: Closer,
    join: JoinHandle<Outcome>,
}

impl ShellHandle {
    pub fn closer(&self) -> Closer {
        self.closer.clone()
    }

    pub fn request_close(&self) {
        self.closer.request_close();
    }

    /// Attend la fin du superviseur (sortie de l'application ordonnee).
    pub fn wait(self) -> Outcome {
        self.join.join().unwrap_or_default()
    }
}

/// Lance le serveur et le superviseur. Une erreur de lancement (binaire absent, refuse) est rendue
/// a l'appelant, qui montre l'ecran d'arret.
pub fn start(config: ShellConfig, host: impl Host) -> io::Result<ShellHandle> {
    let ShellConfig { spec, timeouts, log } = config;
    let Spawned {
        sidecar,
        stdout,
        stderr,
    } = sidecar::spawn(&spec)?;
    let sidecar = Arc::new(sidecar);
    log_shell(
        &log,
        &format!("event=sidecar_spawned pid={} exe={}", sidecar.pid(), spec.exe.display()),
    );

    let (tx, rx) = mpsc::channel::<Input>();
    let (drained_tx, drained_rx) = mpsc::channel::<()>();
    {
        let (tx, log) = (tx.clone(), log.clone());
        thread::spawn(move || pump_stdout(stdout, &tx, &log, &drained_tx));
    }
    {
        let log = log.clone();
        thread::spawn(move || pump_stderr(stderr, &log));
    }
    {
        let (tx, log, sidecar) = (tx.clone(), log.clone(), sidecar.clone());
        thread::spawn(move || watch_exit(&sidecar, &tx, &drained_rx, &log));
    }

    let (machine, initial) = Lifecycle::start(timeouts);
    let secret = spec.secret.clone();
    let run_log = log.clone();
    let join = thread::spawn(move || run(rx, machine, initial, &sidecar, Box::new(host), &secret, &run_log));
    Ok(ShellHandle {
        closer: Closer(tx),
        join,
    })
}

/// La boucle du superviseur : applique les actions, attend la prochaine entree ou minuterie.
fn run(
    rx: mpsc::Receiver<Input>,
    mut machine: Lifecycle,
    initial: Vec<Action>,
    sidecar: &Sidecar,
    mut host: Box<dyn Host>,
    secret: &Secret,
    log: &LogSink,
) -> Outcome {
    let mut timers: Vec<(Timer, Instant)> = Vec::new();
    let mut actions = initial;
    loop {
        for action in actions.drain(..) {
            apply(action, sidecar, host.as_mut(), secret, log, &mut timers);
        }
        if machine.is_done() {
            break;
        }
        let wait = timers
            .iter()
            .map(|(_, at)| *at)
            .min()
            .map(|at| at.saturating_duration_since(Instant::now()));
        let received = match wait {
            Some(d) => rx.recv_timeout(d),
            None => rx.recv().map_err(|_| RecvTimeoutError::Disconnected),
        };
        match received {
            Ok(input) => {
                if let Input::Event(event) = &input {
                    log_shell(log, &format!("event=server_event {}", describe(event)));
                }
                actions = machine.step(input);
            }
            Err(RecvTimeoutError::Timeout) => {
                let now = Instant::now();
                let (due, pending): (Vec<_>, Vec<_>) = timers.drain(..).partition(|(_, at)| *at <= now);
                timers = pending;
                for (timer, _) in due {
                    log_shell(log, &format!("event=timer_fired timer={timer:?}"));
                    actions.extend(machine.step(Input::TimerFired(timer)));
                }
            }
            Err(RecvTimeoutError::Disconnected) => break,
        }
    }
    Outcome {
        stopped_clean: machine.stopped_clean(),
        killed: machine.was_killed(),
    }
}

fn apply(
    action: Action,
    sidecar: &Sidecar,
    host: &mut dyn Host,
    secret: &Secret,
    log: &LogSink,
    timers: &mut Vec<(Timer, Instant)>,
) {
    match action {
        Action::ShowScreen(screen) => {
            log_shell(log, &format!("event=show_screen screen={}", screen.message_key()));
            host.show_screen(screen);
        }
        Action::LoadEntry { port } => {
            log_shell(log, &format!("event=load_entry port={port}"));
            host.load_entry(&format!(
                "http://127.0.0.1:{port}/desktop/enter?secret={}",
                secret.expose()
            ));
        }
        Action::LoadServerPage { port } => {
            log_shell(log, &format!("event=load_server_page port={port}"));
            host.load_url(&format!("http://127.0.0.1:{port}/"));
        }
        Action::AnnouncePort { port } => host.announce_port(port),
        Action::CloseStdin => {
            log_shell(log, "event=close_stdin");
            sidecar.close_stdin();
        }
        Action::Kill => {
            log_shell(log, "event=kill");
            if let Err(e) = sidecar.kill() {
                log_shell(log, &format!("event=kill_failed err={e}"));
            }
        }
        Action::Arm(timer, after) => {
            timers.retain(|(t, _)| *t != timer);
            timers.push((timer, Instant::now() + after));
        }
        Action::Disarm(timer) => timers.retain(|(t, _)| *t != timer),
        Action::ExitApp => {
            log_shell(log, "event=exit_app");
            host.exit_app();
        }
    }
}

/// Vide stdout : decode chaque ligne en evenement, journalise ce qui n'en est pas un (borne),
/// et continue de lire meme si le superviseur n'ecoute plus (le serveur ne doit jamais bloquer).
fn pump_stdout(stdout: std::process::ChildStdout, tx: &Sender<Input>, log: &LogSink, drained: &Sender<()>) {
    let mut reader = BufReader::new(stdout);
    let mut ignored: u64 = 0;
    loop {
        match read_line_bounded(&mut reader, MAX_LINE_BYTES) {
            Ok(Some(line)) => match parse_line(&line.text) {
                // Un superviseur termine n'ecoute plus : on continue de vider le tube.
                Ok(event) => {
                    let _ = tx.send(Input::Event(event));
                }
                Err(reason) => {
                    ignored += 1;
                    if ignored <= LOGGED_IGNORED_LINES {
                        let text: String = line.text.chars().take(LOGGED_LINE_CHARS).collect();
                        log_shell(
                            log,
                            &format!(
                                "event=stdout_ignored reason={} truncated={} line={text:?}",
                                reason_name(&reason),
                                line.truncated
                            ),
                        );
                    }
                }
            },
            Ok(None) => break,
            Err(e) => {
                log_shell(log, &format!("event=stdout_read_failed err={e}"));
                break;
            }
        }
    }
    if ignored > LOGGED_IGNORED_LINES {
        log_shell(log, &format!("event=stdout_ignored_total count={ignored}"));
    }
    // Le surveillant attend ce signal ; s'il est deja parti, personne ne l'attend plus.
    let _ = drained.send(());
}

/// Relaie la sortie d'erreur du serveur vers `desktop.log`, ligne a ligne.
fn pump_stderr(stderr: std::process::ChildStderr, log: &LogSink) {
    let mut reader = BufReader::new(stderr);
    loop {
        match read_line_bounded(&mut reader, MAX_LINE_BYTES) {
            Ok(Some(line)) => log.write_line(Source::Sidecar, &line.text),
            Ok(None) => break,
            Err(e) => {
                log_shell(log, &format!("event=stderr_read_failed err={e}"));
                break;
            }
        }
    }
}

/// Surveille la sortie du processus, laisse stdout se vider, puis annonce `Exited`.
fn watch_exit(sidecar: &Sidecar, tx: &Sender<Input>, drained: &mpsc::Receiver<()>, log: &LogSink) {
    let status: Option<ExitStatus> = loop {
        match sidecar.try_wait() {
            Ok(Some(status)) => break Some(status),
            Ok(None) => thread::sleep(WAIT_POLL),
            Err(e) => {
                log_shell(log, &format!("event=wait_failed err={e}"));
                break None;
            }
        }
    };
    match status {
        Some(status) => log_shell(log, &format!("event=sidecar_exited code={:?}", status.code())),
        None => log_shell(log, "event=sidecar_exited code=unknown"),
    }
    if drained.recv_timeout(STDOUT_DRAIN_GRACE).is_err() {
        log_shell(log, "event=stdout_drain_timeout");
    }
    // Un superviseur termine n'a plus besoin de cette annonce.
    let _ = tx.send(Input::Exited);
}

fn log_shell(log: &LogSink, text: &str) {
    log.write_line(Source::Shell, text);
}

fn reason_name(reason: &ParseError) -> String {
    match reason {
        ParseError::NotJson => "not_json".into(),
        ParseError::NotAnEvent => "not_an_event".into(),
        ParseError::UnknownEvent(name) => format!("unknown_event:{name}"),
        ParseError::BadField(field) => format!("bad_field:{field}"),
    }
}

fn describe(event: &SidecarEvent) -> String {
    match event {
        SidecarEvent::Listening { port } => format!("kind=listening port={port}"),
        SidecarEvent::Ready { port } => format!("kind=ready port={port}"),
        SidecarEvent::Failed { state } => format!("kind=failed state={state}"),
        SidecarEvent::Stopping => "kind=stopping".into(),
        SidecarEvent::Stopped { clean } => format!("kind=stopped clean={clean}"),
    }
}
