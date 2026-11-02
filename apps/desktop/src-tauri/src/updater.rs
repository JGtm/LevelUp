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
            (UpdateState::Downloading, UpdateInput::Downloaded(Err(e))) => {
                vec![UpdateAction::Log(format!("event=update_download_failed err={e}"))]
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

/// Pourquoi l'installation ne part pas : l'arret du serveur n'a pas ete propre.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum SkipReason {
    /// Le kill a echoue : le serveur peut tourner encore.
    KillFailed,
    /// La coquille a du tuer le serveur (budget d'arret depasse).
    Killed,
    /// Le serveur a annonce `stopped` avec `clean=false`.
    Unclean,
    /// Aucun `stopped` n'est arrive.
    NoStopped,
}

impl SkipReason {
    /// Le nom journalise (`reason=` de l'evenement `update_skipped_unclean_stop`).
    pub fn name(self) -> &'static str {
        match self {
            Self::KillFailed => "kill_failed",
            Self::Killed => "killed",
            Self::Unclean => "unclean",
            Self::NoStopped => "no_stopped",
        }
    }
}

/// La decision d'installer, PURE : l'installeur ne part que si `stopped` est arrive avec `clean=true`
/// ET que le serveur n'a pas ete tue. Un installeur lance sur un serveur qui vit encore (ou dont la
/// base n'a pas ete fermee proprement) remplacerait l'executable sous ses pieds.
pub fn install_allowed(outcome: Outcome) -> Result<(), SkipReason> {
    if outcome.kill_failed {
        Err(SkipReason::KillFailed)
    } else if outcome.killed {
        Err(SkipReason::Killed)
    } else {
        match outcome.stopped_clean {
            Some(true) => Ok(()),
            Some(false) => Err(SkipReason::Unclean),
            None => Err(SkipReason::NoStopped),
        }
    }
}

/// Un message que la coquille montre dans une boite native : apres l'arret du serveur quand la mise a
/// jour ne s'est pas faite, ou en reponse a une verification demandee par l'utilisateur.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Notice {
    /// Arret du serveur non propre : la mise a jour est repoussee au prochain lancement.
    SkippedUnclean,
    /// L'installeur a echoue : l'application se ferme sans installation.
    InstallFailed,
    /// Verification demandee : la version installee est la derniere.
    UpToDate,
    /// Verification demandee : le serveur de mises a jour n'a pas repondu (hors ligne, indisponible).
    CheckFailed,
    /// Verification demandee : la mise a jour n'est pas branchee dans ce build (developpement).
    Unavailable,
    /// Verification demandee pendant qu'une autre est en cours.
    Busy,
}

impl Notice {
    /// Tous les messages, pour les tests de completude.
    pub const ALL: [Notice; 6] = [
        Self::SkippedUnclean,
        Self::InstallFailed,
        Self::UpToDate,
        Self::CheckFailed,
        Self::Unavailable,
        Self::Busy,
    ];

    fn message_key(self) -> &'static str {
        match self {
            Self::SkippedUnclean => "update_skipped_unclean",
            Self::InstallFailed => "update_install_failed",
            Self::UpToDate => "update_none",
            Self::CheckFailed => "update_check_failed",
            Self::Unavailable => "update_unavailable_debug",
            Self::Busy => "update_busy",
        }
    }

    /// Vrai pour une simple information, faux pour un message qui previent d'un probleme.
    pub fn is_informational(self) -> bool {
        matches!(self, Self::UpToDate | Self::Unavailable | Self::Busy)
    }
}

/// Les textes d'une boite d'information, dans la langue de la coquille.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct NoticeText {
    pub title: String,
    pub message: String,
}

/// Version de l'application installee (celle du paquet : `Cargo.toml` fait foi, A9).
pub const CURRENT_VERSION: &str = env!("CARGO_PKG_VERSION");

/// Compose le texte d'un message depuis le catalogue de repli ; `{version}` vaut la version installee.
pub fn notice_text(locale: &str, notice: Notice) -> NoticeText {
    let text = |key: &str| catalogue::text(locale, key).unwrap_or_default().to_string();
    NoticeText {
        title: text("update_title"),
        message: text(notice.message_key()).replace("{version}", CURRENT_VERSION),
    }
}

/// Une boite d'information native (bloquante). Le serveur est deja arrete quand elle s'affiche : le
/// texte vient du catalogue compile dans la coquille.
pub trait Notifier: Send + 'static {
    fn show(&mut self, notice: Notice);
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

#[cfg(test)]
mod tests {
    use super::*;

    fn offer(version: &str) -> Offer {
        Offer {
            version: version.into(),
            notes: Some("- un changement\n- un autre".into()),
        }
    }

    fn drive(flow: &mut UpdateFlow, inputs: Vec<UpdateInput>) -> Vec<Vec<UpdateAction>> {
        inputs.into_iter().map(|input| flow.step(input)).collect()
    }

    #[test]
    fn la_verification_ne_commence_qu_au_serveur_pret_et_une_seule_fois() {
        let mut flow = UpdateFlow::new();
        assert_eq!(flow.state(), &UpdateState::Idle);
        assert_eq!(flow.step(UpdateInput::ServerReady), vec![UpdateAction::Check]);
        assert!(flow.step(UpdateInput::ServerReady).is_empty());
        assert_eq!(flow.state(), &UpdateState::Checking);
    }

    #[test]
    fn rien_ne_se_passe_avant_la_verification() {
        let mut flow = UpdateFlow::new();
        assert!(flow.step(UpdateInput::Decided(Decision::Install)).is_empty());
        assert!(flow.step(UpdateInput::Downloaded(Ok(()))).is_empty());
        assert_eq!(flow.state(), &UpdateState::Idle);
    }

    #[test]
    fn proposee_acceptee_telechargee_puis_porte_armee_et_fermeture() {
        let mut flow = UpdateFlow::new();
        let steps = drive(
            &mut flow,
            vec![
                UpdateInput::ServerReady,
                UpdateInput::Checked(Ok(Some(offer("8.2.0")))),
                UpdateInput::Decided(Decision::Install),
                UpdateInput::Downloaded(Ok(())),
            ],
        );
        assert_eq!(steps[0], vec![UpdateAction::Check]);
        assert_eq!(steps[1], vec![UpdateAction::Ask(offer("8.2.0"))]);
        assert_eq!(steps[2], vec![UpdateAction::Download(offer("8.2.0"))]);
        assert_eq!(steps[3], vec![UpdateAction::ArmAndClose]);
        assert_eq!(flow.state(), &UpdateState::Armed);
    }

    #[test]
    fn plus_tard_ne_telecharge_rien_et_ne_ferme_rien() {
        let mut flow = UpdateFlow::new();
        let steps = drive(
            &mut flow,
            vec![
                UpdateInput::ServerReady,
                UpdateInput::Checked(Ok(Some(offer("8.2.0")))),
                UpdateInput::Decided(Decision::Later),
                // Une reponse tardive ou repetee ne ressuscite pas la proposition.
                UpdateInput::Decided(Decision::Install),
                UpdateInput::Downloaded(Ok(())),
            ],
        );
        assert!(matches!(steps[2].as_slice(), [UpdateAction::Log(l)] if l.contains("update_declined")));
        assert!(steps[3].is_empty() && steps[4].is_empty());
        assert_eq!(flow.state(), &UpdateState::Done);
    }

    #[test]
    fn a_jour_ou_erreur_de_verification_ne_proposent_rien() {
        for checked in [Ok(None), Err("reseau".to_string())] {
            let mut flow = UpdateFlow::new();
            flow.step(UpdateInput::ServerReady);
            let actions = flow.step(UpdateInput::Checked(checked));
            assert!(matches!(actions.as_slice(), [UpdateAction::Log(_)]), "{actions:?}");
            assert_eq!(flow.state(), &UpdateState::Done);
        }
    }

    #[test]
    fn un_telechargement_en_echec_n_arme_pas_la_porte() {
        let mut flow = UpdateFlow::new();
        drive(
            &mut flow,
            vec![
                UpdateInput::ServerReady,
                UpdateInput::Checked(Ok(Some(offer("8.2.0")))),
                UpdateInput::Decided(Decision::Install),
            ],
        );
        let actions = flow.step(UpdateInput::Downloaded(Err("coupe".into())));
        assert!(matches!(actions.as_slice(), [UpdateAction::Log(l)] if l.contains("update_download_failed")));
        assert_eq!(flow.state(), &UpdateState::Done);
    }

    #[test]
    fn la_boite_de_dialogue_est_complete_dans_chaque_langue() {
        for code in catalogue::LOCALES {
            let text = prompt_text(code, &offer("8.2.0"));
            for (name, value) in [
                ("title", &text.title),
                ("message", &text.message),
                ("install", &text.install),
                ("later", &text.later),
            ] {
                assert!(!value.trim().is_empty(), "{code}/{name} vide");
            }
            assert!(text.message.contains("8.2.0"), "{code} : version absente");
            assert!(!text.message.contains("{version}"), "{code} : gabarit non rempli");
            assert!(text.message.contains("- un changement"), "{code} : notes absentes");
            assert_ne!(text.install, text.later);
        }
    }

    #[test]
    fn sans_notes_le_message_n_a_pas_d_entete_de_notes() {
        let without = Offer {
            version: "8.2.0".into(),
            notes: Some("  \n".into()),
        };
        let locale = catalogue::DEFAULT_LOCALE;
        let message = prompt_text(locale, &without).message;
        assert!(!message.contains(catalogue::text(locale, "update_notes_heading").unwrap()));
        assert!(!message.contains('\n'));
    }

    #[test]
    fn les_notes_trop_longues_sont_coupees_net_avec_une_ellipse() {
        let long = Offer {
            version: "8.2.0".into(),
            notes: Some("Z".repeat(MAX_NOTES_CHARS * 2)),
        };
        let message = prompt_text(catalogue::DEFAULT_LOCALE, &long).message;
        assert!(message.ends_with('\u{2026}'));
        assert_eq!(message.matches('Z').count(), MAX_NOTES_CHARS);
    }

    #[test]
    fn la_porte_s_ouvre_une_seule_fois() {
        let gate = InstallGate::default();
        assert!(!gate.is_armed() && gate.take().is_none());
        gate.arm(Box::new(|| Ok(())));
        assert!(gate.is_armed());
        assert!(gate.take().is_some());
        assert!(gate.take().is_none());
    }

    #[test]
    fn la_poignee_de_fermeture_expire_si_elle_n_est_jamais_posee() {
        assert!(CloserSlot::default().wait(Duration::from_millis(50)).is_none());
    }

    fn outcome(stopped_clean: Option<bool>, killed: bool, kill_failed: bool) -> Outcome {
        Outcome {
            stopped_clean,
            killed,
            kill_failed,
        }
    }

    #[test]
    fn l_installation_ne_part_que_sur_un_arret_propre_sans_kill() {
        assert_eq!(install_allowed(outcome(Some(true), false, false)), Ok(()));
        assert_eq!(
            install_allowed(outcome(Some(false), false, false)),
            Err(SkipReason::Unclean)
        );
        assert_eq!(install_allowed(outcome(None, false, false)), Err(SkipReason::NoStopped));
        // Un `stopped` propre ne rachete pas un kill : le serveur a ete tue ou a refuse de sortir.
        assert_eq!(
            install_allowed(outcome(Some(true), true, false)),
            Err(SkipReason::Killed)
        );
        assert_eq!(install_allowed(outcome(None, true, false)), Err(SkipReason::Killed));
        // Un kill echoue est la raison la plus precise : le processus vit peut-etre encore.
        assert_eq!(
            install_allowed(outcome(Some(true), true, true)),
            Err(SkipReason::KillFailed)
        );
    }

    #[test]
    fn chaque_raison_a_un_nom_journalise_distinct() {
        let names = [
            SkipReason::KillFailed,
            SkipReason::Killed,
            SkipReason::Unclean,
            SkipReason::NoStopped,
        ]
        .map(SkipReason::name);
        assert_eq!(names, ["kill_failed", "killed", "unclean", "no_stopped"]);
    }

    #[test]
    fn les_messages_de_boite_sont_complets_et_distincts_dans_chaque_langue() {
        for code in catalogue::LOCALES {
            let mut seen = Vec::new();
            for notice in Notice::ALL {
                let text = notice_text(code, notice);
                assert!(!text.title.trim().is_empty(), "{code}/{notice:?} : titre vide");
                assert!(!text.message.trim().is_empty(), "{code}/{notice:?} : message vide");
                assert!(
                    !text.message.contains("{version}"),
                    "{code}/{notice:?} : version non remplacee"
                );
                assert!(
                    !seen.contains(&text.message),
                    "{code}/{notice:?} : message deja utilise"
                );
                seen.push(text.message);
            }
        }
    }

    #[test]
    fn le_message_a_jour_nomme_la_version_installee() {
        for code in catalogue::LOCALES {
            assert!(notice_text(code, Notice::UpToDate).message.contains(CURRENT_VERSION));
        }
    }

    #[test]
    fn une_verification_demandee_annonce_a_jour_et_echec_pas_celle_du_lancement() {
        let mut silent = UpdateFlow::new();
        silent.step(UpdateInput::ServerReady);
        let none = silent.step(UpdateInput::Checked(Ok(None)));
        assert_eq!(none, vec![UpdateAction::Log("event=update_none".into())]);

        let mut asked = UpdateFlow::manual();
        asked.step(UpdateInput::ServerReady);
        let none = asked.step(UpdateInput::Checked(Ok(None)));
        assert_eq!(
            none,
            vec![
                UpdateAction::Log("event=update_none".into()),
                UpdateAction::Inform(Notice::UpToDate)
            ]
        );

        let mut asked = UpdateFlow::manual();
        asked.step(UpdateInput::ServerReady);
        let failed = asked.step(UpdateInput::Checked(Err("hors ligne".into())));
        assert_eq!(
            failed,
            vec![
                UpdateAction::Log("event=update_check_failed err=hors ligne".into()),
                UpdateAction::Inform(Notice::CheckFailed)
            ]
        );
    }

    #[test]
    fn une_verification_demandee_propose_la_mise_a_jour_comme_au_lancement() {
        let mut flow = UpdateFlow::manual();
        flow.step(UpdateInput::ServerReady);
        let ask = flow.step(UpdateInput::Checked(Ok(Some(offer("8.2.0")))));
        assert_eq!(ask, vec![UpdateAction::Ask(offer("8.2.0"))]);
    }
}
