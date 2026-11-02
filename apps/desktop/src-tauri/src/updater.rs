//! La mise a jour (ADR 0045 D-16, plan A3) : proposee, jamais imposee, installee seulement apres un
//! arret propre du serveur.
//!
//! Tout ce qui se decide sans fenetre vit ici, sans tauri : la machine d'etats ([`UpdateFlow`]), la
//! porte d'installation ([`InstallGate`]), l'hote qui installe a la sortie ([`UpdateHost`]) et le
//! texte de la boite de dialogue ([`prompt_text`], tire du catalogue : aucun libelle de langue ici).
//! Le binaire branche le plugin d'update de Tauri derriere [`UpdateSource`] et une boite de dialogue
//! native derriere [`Prompt`] ; les tests branchent des doubles et le faux serveur.
//!
//! ```text
//!  serveur `ready` --> verification (fil a part, jamais bloquante)
//!     |-- pas de mise a jour, ou erreur (journalisee)      --> rien
//!     `-- mise a jour proposee --> boite de dialogue
//!            |-- « Plus tard »   --> rien (une seule proposition par lancement)
//!            `-- « Installer »   --> telechargement (le serveur tourne toujours)
//!                  |-- echec     --> rien (journalise)
//!                  `-- termine   --> porte armee, demande de fermeture propre
//!                        ... fermeture : stdin ferme, `stopped`, sortie du serveur ...
//!                        --> `exit_app` : l'installeur se lance si l'arret a ete propre (`stopped`
//!                            avec `clean=true`, serveur non tue), sinon rien n'est installe, la raison
//!                            est journalisee et une boite previent ; l'installeur quitte l'application
//! ```

use std::collections::VecDeque;
use std::sync::{Arc, Condvar, Mutex};
use std::time::{Duration, Instant};

use crate::catalogue;
use crate::logfile::{LogSink, Source};
use crate::screen::Screen;
use crate::supervisor::{Closer, Host, Outcome};

/// Longueur maximale des notes montrees dans la boite de dialogue (en caracteres).
pub const MAX_NOTES_CHARS: usize = 1200;
/// Echeance de l'interrogation du serveur de mises a jour : sans elle, un serveur qui accepte la connexion
/// puis ne repond jamais garderait le verrou de verification toute la session.
pub const CHECK_TIMEOUT: Duration = Duration::from_secs(30);
/// Delai d'attente de la poignee de fermeture, posee juste apres le lancement du superviseur.
pub const CLOSER_WAIT: Duration = Duration::from_secs(30);

/// Une mise a jour proposee : le numero de version et les notes publiees avec elle.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Offer {
    pub version: String,
    pub notes: Option<String>,
}

/// Les textes de la boite de dialogue, dans la langue de la coquille.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct PromptText {
    pub title: String,
    pub message: String,
    pub install: String,
    pub later: String,
}

/// Compose les textes de la boite de dialogue depuis le catalogue de repli. Les notes sont tronquees
/// a [`MAX_NOTES_CHARS`] caracteres (une coupure nette, signalee par une ellipse).
pub fn prompt_text(locale: &str, offer: &Offer) -> PromptText {
    let text = |key: &str| catalogue::text(locale, key).unwrap_or_default().to_string();
    let mut message = text("update_available").replace("{version}", &offer.version);
    if let Some(notes) = offer.notes.as_deref().map(str::trim).filter(|n| !n.is_empty()) {
        message.push_str("\n\n");
        message.push_str(&text("update_notes_heading"));
        message.push('\n');
        message.push_str(&truncate_chars(notes, MAX_NOTES_CHARS));
    }
    PromptText {
        title: text("update_title"),
        message,
        install: text("update_install"),
        later: text("update_later"),
    }
}

fn truncate_chars(text: &str, max: usize) -> String {
    if text.chars().count() <= max {
        return text.to_string();
    }
    let mut cut: String = text.chars().take(max).collect();
    cut.push('\u{2026}');
    cut
}

/// La reponse de l'utilisateur.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Decision {
    Install,
    Later,
}

/// Une boite de dialogue qui propose la mise a jour (bloquante : elle tourne sur son propre fil).
pub trait Prompt: Send + 'static {
    fn ask(&mut self, text: &PromptText) -> Decision;
}

/// Ce que l'installation fait une fois le serveur arrete. Sur Windows, l'installeur lance quitte
/// l'application : un retour `Ok` n'arrive que sur une plateforme ou il ne la quitte pas.
pub type Installer = Box<dyn FnOnce() -> Result<(), String> + Send>;

/// Le service de mise a jour (le plugin de Tauri en production, un double dans les tests).
pub trait UpdateSource: Send + 'static {
    /// Interroge le serveur de mises a jour : `None` si la version installee est la derniere.
    fn check(&mut self) -> Result<Option<Offer>, String>;
    /// Telecharge la mise a jour proposee ; rend l'installation a lancer plus tard.
    fn download(&mut self, offer: &Offer) -> Result<Installer, String>;
}

/// Etat de la proposition.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum UpdateState {
    Idle,
    Checking,
    Offered(Offer),
    Downloading,
    /// Telechargee, installation en attente de l'arret du serveur.
    Armed,
    /// Rien d'autre a faire pendant ce lancement (a jour, refusee, en echec).
    Done,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub enum UpdateInput {
    /// Le serveur est pret : la verification peut commencer.
    ServerReady,
    Checked(Result<Option<Offer>, String>),
    Decided(Decision),
    Downloaded(Result<(), String>),
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub enum UpdateAction {
    Check,
    Ask(Offer),
    Download(Offer),
    /// Armer la porte d'installation, puis demander la fermeture propre.
    ArmAndClose,
    /// Journaliser une raison de ne rien faire.
    Log(String),
    /// Dire a l'utilisateur ce qu'a donne une verification qu'il a demandee (jamais emis pour la
    /// verification du lancement, qui reste silencieuse sans mise a jour).
    Inform(Notice),
}

/// La machine d'etats de la proposition, PURE.
#[derive(Debug)]
pub struct UpdateFlow {
    state: UpdateState,
    /// Verification demandee par l'utilisateur : « a jour » et « echec » se disent dans une boite.
    manual: bool,
}

impl Default for UpdateFlow {
    fn default() -> Self {
        Self::new()
    }
}

impl UpdateFlow {
    /// La verification du lancement : silencieuse tant qu'il n'y a rien a proposer.
    pub fn new() -> Self {
        Self {
            state: UpdateState::Idle,
            manual: false,
        }
    }

    /// Une verification demandee par l'utilisateur : meme sequence, mais chaque issue sans mise a
    /// jour est annoncee.
    pub fn manual() -> Self {
        Self {
            state: UpdateState::Idle,
            manual: true,
        }
    }

    pub fn state(&self) -> &UpdateState {
        &self.state
    }

    pub fn step(&mut self, input: UpdateInput) -> Vec<UpdateAction> {
        let state = std::mem::replace(&mut self.state, UpdateState::Done);
        match (state, input) {
            (UpdateState::Idle, UpdateInput::ServerReady) => {
                self.state = UpdateState::Checking;
                vec![UpdateAction::Check]
            }
            (UpdateState::Checking, UpdateInput::Checked(Ok(Some(offer)))) => {
                self.state = UpdateState::Offered(offer.clone());
                vec![UpdateAction::Ask(offer)]
            }
            (UpdateState::Checking, UpdateInput::Checked(Ok(None))) => {
                self.with_notice("event=update_none".into(), Notice::UpToDate)
            }
            (UpdateState::Checking, UpdateInput::Checked(Err(e))) => {
                self.with_notice(format!("event=update_check_failed err={e}"), Notice::CheckFailed)
            }
            (UpdateState::Offered(offer), UpdateInput::Decided(Decision::Install)) => {
                self.state = UpdateState::Downloading;
                vec![UpdateAction::Download(offer)]
            }
            (UpdateState::Offered(offer), UpdateInput::Decided(Decision::Later)) => {
                vec![UpdateAction::Log(format!(
                    "event=update_declined version={}",
                    offer.version
                ))]
            }
            (UpdateState::Downloading, UpdateInput::Downloaded(Ok(()))) => {
                self.state = UpdateState::Armed;
                vec![UpdateAction::ArmAndClose]
            }
            // Un telechargement abandonne (coupe, inactivite, duree maximale) se dit comme un echec de
            // verification : la boite ne s'ouvre que pour une verification demandee par l'utilisateur.
            (UpdateState::Downloading, UpdateInput::Downloaded(Err(e))) => {
                self.with_notice(format!("event=update_download_failed err={e}"), Notice::CheckFailed)
            }
            // Une entree hors sequence (deuxieme `ServerReady`, reponse apres coup) ne change rien.
            (state, _) => {
                self.state = state;
                Vec::new()
            }
        }
    }

    /// Le journal, puis la boite si l'utilisateur a demande cette verification.
    fn with_notice(&self, log: String, notice: Notice) -> Vec<UpdateAction> {
        let mut actions = vec![UpdateAction::Log(log)];
        if self.manual {
            actions.push(UpdateAction::Inform(notice));
        }
        actions
    }
}

/// La porte d'installation : la mise a jour telechargee attend ici l'arret du serveur. Partagee entre
/// le fil de mise a jour (qui l'arme) et l'hote (qui l'ouvre a la sortie de l'application).
#[derive(Clone, Default)]
pub struct InstallGate(Arc<Mutex<Option<Installer>>>);

impl InstallGate {
    pub fn arm(&self, installer: Installer) {
        *self.0.lock().unwrap_or_else(|p| p.into_inner()) = Some(installer);
    }

    /// Reprend l'installation armee (une seule fois).
    pub fn take(&self) -> Option<Installer> {
        self.0.lock().unwrap_or_else(|p| p.into_inner()).take()
    }

    pub fn is_armed(&self) -> bool {
        self.0.lock().unwrap_or_else(|p| p.into_inner()).is_some()
    }
}

/// La poignee de fermeture du superviseur, connue seulement apres son lancement : le fil de mise a
/// jour l'attend ici.
#[derive(Clone, Default)]
pub struct CloserSlot(Arc<(Mutex<Option<Closer>>, Condvar)>);

impl CloserSlot {
    pub fn set(&self, closer: Closer) {
        let (slot, changed) = &*self.0;
        *slot.lock().unwrap_or_else(|p| p.into_inner()) = Some(closer);
        changed.notify_all();
    }

    pub fn wait(&self, timeout: Duration) -> Option<Closer> {
        let (slot, changed) = &*self.0;
        let deadline = Instant::now() + timeout;
        let mut guard = slot.lock().unwrap_or_else(|p| p.into_inner());
        while guard.is_none() {
            let left = deadline.saturating_duration_since(Instant::now());
            if left.is_zero() {
                return None;
            }
            guard = changed.wait_timeout(guard, left).unwrap_or_else(|p| p.into_inner()).0;
        }
        guard.clone()
    }
}

/// Ce que partagent toutes les verifications : la langue des boites, la porte d'installation, la
/// poignee de fermeture du superviseur et le journal.
pub struct UpdateContext<'a> {
    pub locale: &'a str,
    pub gate: &'a InstallGate,
    pub closer: &'a CloserSlot,
    pub log: &'a LogSink,
}

/// Une fois le serveur pret : verifie, propose, telecharge, arme la porte et demande la fermeture.
/// Tourne sur son propre fil et ne bloque jamais l'interface ; chaque issue sans installation est
/// journalisee.
pub fn run_update(
    source: impl UpdateSource,
    prompt: impl Prompt,
    locale: &str,
    gate: &InstallGate,
    closer: &CloserSlot,
    log: &LogSink,
) {
    let context = UpdateContext {
        locale,
        gate,
        closer,
        log,
    };
    run_flow(UpdateFlow::new(), source, prompt, None, &context);
}

/// Deroule une verification de bout en bout. `notifier` n'est donne que pour une verification
/// demandee par l'utilisateur : c'est par lui que « a jour » et « echec » se disent.
pub(crate) fn run_flow(
    mut flow: UpdateFlow,
    mut source: impl UpdateSource,
    mut prompt: impl Prompt,
    mut notifier: Option<&mut dyn Notifier>,
    context: &UpdateContext<'_>,
) {
    let UpdateContext {
        locale,
        gate,
        closer,
        log,
    } = *context;
    let mut installer: Option<Installer> = None;
    let mut pending: VecDeque<UpdateAction> = flow.step(UpdateInput::ServerReady).into();
    while let Some(action) = pending.pop_front() {
        let next = match action {
            UpdateAction::Check => flow.step(UpdateInput::Checked(source.check())),
            UpdateAction::Ask(offer) => {
                log_shell(log, &format!("event=update_offered version={}", offer.version));
                flow.step(UpdateInput::Decided(prompt.ask(&prompt_text(locale, &offer))))
            }
            UpdateAction::Download(offer) => match source.download(&offer) {
                Ok(ready) => {
                    installer = Some(ready);
                    flow.step(UpdateInput::Downloaded(Ok(())))
                }
                Err(e) => flow.step(UpdateInput::Downloaded(Err(e))),
            },
            UpdateAction::ArmAndClose => {
                if let Some(ready) = installer.take() {
                    gate.arm(ready);
                }
                match closer.wait(CLOSER_WAIT) {
                    Some(closer) => {
                        log_shell(log, "event=update_close_requested");
                        closer.request_close();
                    }
                    None => log_shell(log, "event=update_close_unavailable"),
                }
                Vec::new()
            }
            UpdateAction::Log(line) => {
                log_shell(log, &line);
                Vec::new()
            }
            UpdateAction::Inform(notice) => {
                if let Some(notifier) = notifier.as_mut() {
                    notifier.show(notice);
                }
                Vec::new()
            }
        };
        pending.extend(next);
    }
}

fn log_shell(log: &LogSink, text: &str) {
    log.write_line(Source::Shell, text);
}
/// L'hote qui enveloppe celui de la fenetre : lance la verification quand le serveur est pret et
/// installe la mise a jour armee, juste avant la sortie de l'application et seulement apres un arret
/// propre (voir [`install_allowed`]) : sinon il journalise la raison, previent l'utilisateur et sort
/// sans installer.
pub struct UpdateHost<H: Host> {
    inner: H,
    gate: InstallGate,
    on_ready: Option<Box<dyn FnOnce() + Send>>,
    log: Arc<LogSink>,
    notifier: Box<dyn Notifier>,
}

impl<H: Host> UpdateHost<H> {
    pub fn new(
        inner: H,
        gate: InstallGate,
        log: Arc<LogSink>,
        on_ready: Box<dyn FnOnce() + Send>,
        notifier: Box<dyn Notifier>,
    ) -> Self {
        Self {
            inner,
            gate,
            on_ready: Some(on_ready),
            log,
            notifier,
        }
    }

    /// Lance l'installeur armee quand l'arret le permet ; sinon journalise et previent.
    fn install_if_allowed(&mut self, install: Installer, outcome: Outcome) {
        match install_allowed(outcome) {
            Ok(()) => {
                log_shell(&self.log, "event=update_install");
                if let Err(e) = install() {
                    log_shell(&self.log, &format!("event=update_install_failed err={e}"));
                    self.notifier.show(Notice::InstallFailed);
                }
            }
            Err(reason) => {
                log_shell(
                    &self.log,
                    &format!("event=update_skipped_unclean_stop reason={}", reason.name()),
                );
                self.notifier.show(Notice::SkippedUnclean);
            }
        }
    }
}

impl<H: Host> Host for UpdateHost<H> {
    fn show_screen(&mut self, screen: Screen) {
        self.inner.show_screen(screen);
    }

    fn load_url(&mut self, url: &str) {
        self.inner.load_url(url);
    }

    fn load_entry(&mut self, url: &str) {
        self.inner.load_entry(url);
        // L'entree n'est chargee que sur `ready` : c'est le signal de la verification (une seule fois).
        if let Some(start) = self.on_ready.take() {
            start();
        }
    }

    fn announce_port(&mut self, port: u16) {
        self.inner.announce_port(port);
    }

    fn exit_app(&mut self, outcome: Outcome) {
        if let Some(install) = self.gate.take() {
            self.install_if_allowed(install, outcome);
        }
        self.inner.exit_app(outcome);
    }
}

pub use crate::updater_decision::{
    CURRENT_VERSION, Notice, NoticeText, Notifier, SkipReason, install_allowed, notice_text,
};

#[cfg(test)]
#[path = "updater_tests.rs"]
mod updater_tests;
