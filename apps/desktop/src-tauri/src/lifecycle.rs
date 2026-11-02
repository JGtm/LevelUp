//! La machine d'etats de la coquille (ADR 0045 D-2 a D-5), PURE : elle recoit des entrees
//! (evenements du serveur, sortie du processus, demande de fermeture, minuteries) et rend des
//! actions. Elle ne touche ni processus, ni fenetre, ni horloge : `supervisor` les branche, les
//! tests la jouent sans rien lancer.
//!
//! ```text
//!  Starting --listening--> Listening --ready--> Ready          (ready : charger /desktop/enter)
//!     |  \________failed________/ \                 |
//!     |             |              \                |
//!     |             v               \               |
//!     |          Failed  (port connu : page du serveur ; sinon ecran embarque)
//!     |             |                               |
//!     +--120 s------+--> Failed(delai) + kill        |
//!                                                    |
//!  fermeture demandee (toute phase, processus vivant) :
//!     ecran "fermeture", stdin ferme, minuterie 40 s + 5 s
//!     -> processus sorti : sortie de l'application
//!     -> minuterie : kill, puis sortie apres un delai de grace
//!  processus sorti hors fermeture : ecran "arrete" (ou l'echec annonce)
//! ```

use std::time::Duration;

use crate::events::{SidecarEvent, Toast};
use crate::screen::{FailureKind, Screen};
use crate::shell_settings::ShellSettings;

/// Delai accorde au serveur pour annoncer `ready` (plan, section 6).
pub const STARTUP_TIMEOUT: Duration = Duration::from_secs(120);
/// Budget d'arret du serveur (`shutdownBudget`, 40 s) plus 5 s de marge (ADR 0045 D-4).
pub const CLOSE_BUDGET: Duration = Duration::from_secs(40 + 5);
/// Delai laisse au processus pour sortir apres `stopped` ou apres un kill.
pub const EXIT_GRACE: Duration = Duration::from_secs(5);

/// Les trois delais de la machine (reglables pour les tests).
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct Timeouts {
    pub startup: Duration,
    pub close: Duration,
    pub grace: Duration,
}

impl Default for Timeouts {
    fn default() -> Self {
        Self {
            startup: STARTUP_TIMEOUT,
            close: CLOSE_BUDGET,
            grace: EXIT_GRACE,
        }
    }
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub enum Timer {
    Startup,
    Close,
    Grace,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub enum Input {
    /// Une ligne du serveur decodee.
    Event(SidecarEvent),
    /// Le processus est sorti et sa sortie standard est entierement lue.
    Exited,
    /// L'utilisateur (ou le systeme) demande la fermeture.
    CloseRequested,
    /// « Synchroniser maintenant » du menu de la zone de notification.
    SyncRequested,
    TimerFired(Timer),
}

/// Ce que la machine demande au superviseur. Aucune action ne porte le secret : l'URL d'entree
/// est construite par celui qui le detient.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum Action {
    ShowScreen(Screen),
    /// Charger `/desktop/enter?secret=...` du serveur de ce port.
    LoadEntry {
        port: u16,
    },
    /// Charger la page de demarrage du serveur (`/`), qui sert l'etat d'echec.
    LoadServerPage {
        port: u16,
    },
    /// Le port du serveur est connu (politique de navigation).
    AnnouncePort {
        port: u16,
    },
    CloseStdin,
    Kill,
    Arm(Timer, Duration),
    Disarm(Timer),
    ExitApp,
    /// Montrer une notification systeme relayee depuis la cloche (L21.b).
    Notify(Toast),
    /// Appliquer les reglages annonces par le serveur : icone de la zone de notification, croix (L21.c).
    ApplyShellSettings(ShellSettings),
    /// Ecrire la commande `sync` sur la stdin du serveur : un cycle de synchronisation immediat (L21.c).
    SendSync,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
enum Phase {
    Starting,
    Listening,
    Ready,
    /// Echec affiche ; le processus peut etre encore vivant (il attend l'ordre d'arret).
    Failed,
    /// Fermeture en cours : stdin ferme, on attend la sortie du processus.
    Closing,
    /// Le processus est sorti sans que la coquille l'ait demande : un ecran terminal est affiche.
    Dead,
    /// La sortie de l'application a ete ordonnee.
    Done,
}

pub struct Lifecycle {
    timeouts: Timeouts,
    phase: Phase,
    port: Option<u16>,
    process_alive: bool,
    /// L'ecran terminal deja affiche par la coquille pour l'echec courant (le delai).
    failure_shown: Option<Screen>,
    /// Cause annoncee par le serveur ; sert a l'ecran quand sa page ne repond plus.
    failure: Option<FailureKind>,
    stopped_clean: Option<bool>,
    killed: bool,
}

impl Lifecycle {
    /// Debut de vie : le processus vient d'etre lance. Rend l'ecran d'attente et la minuterie de demarrage.
    pub fn start(timeouts: Timeouts) -> (Self, Vec<Action>) {
        let machine = Self {
            timeouts,
            phase: Phase::Starting,
            port: None,
            process_alive: true,
            failure_shown: None,
            failure: None,
            stopped_clean: None,
            killed: false,
        };
        (
            machine,
            vec![
                Action::ShowScreen(Screen::Starting),
                Action::Arm(Timer::Startup, timeouts.startup),
            ],
        )
    }

    /// Vrai quand la sortie de l'application a ete ordonnee.
    pub fn is_done(&self) -> bool {
        self.phase == Phase::Done
    }

    /// `clean` du dernier `stopped` recu, s'il y en a eu un.
    pub fn stopped_clean(&self) -> Option<bool> {
        self.stopped_clean
    }

    /// Vrai si la coquille a du tuer le serveur.
    pub fn was_killed(&self) -> bool {
        self.killed
    }

    pub fn step(&mut self, input: Input) -> Vec<Action> {
        if self.phase == Phase::Done {
            return Vec::new();
        }
        match input {
            Input::Event(event) => self.on_event(event),
            Input::Exited => self.on_exited(),
            Input::CloseRequested => self.on_close_requested(),
            Input::SyncRequested => self.on_sync_requested(),
            Input::TimerFired(timer) => self.on_timer(timer),
        }
    }

    fn on_event(&mut self, event: SidecarEvent) -> Vec<Action> {
        let booting = matches!(self.phase, Phase::Starting | Phase::Listening);
        match event {
            SidecarEvent::Listening { port } => {
                self.port = Some(port);
                if self.phase == Phase::Starting {
                    self.phase = Phase::Listening;
                }
                vec![Action::AnnouncePort { port }]
            }
            SidecarEvent::Ready { port } if booting => {
                self.port = Some(port);
                self.phase = Phase::Ready;
                vec![
                    Action::Disarm(Timer::Startup),
                    Action::AnnouncePort { port },
                    Action::LoadEntry { port },
                ]
            }
            SidecarEvent::Failed { state } if booting => self.on_failed(&state),
            SidecarEvent::Stopping if matches!(self.phase, Phase::Ready | Phase::Listening | Phase::Starting) => {
                // Arret decide par le serveur lui-meme (signal, route d'arret) : meme ecran que la fermeture.
                vec![Action::ShowScreen(Screen::Stopping)]
            }
            // Une notification n'a de sens que pour une application ouverte : avant `ready` il n'y a rien a
            // montrer, pendant la fermeture ou apres un echec non plus.
            SidecarEvent::Notification(toast) if self.phase == Phase::Ready => vec![Action::Notify(toast)],
            // Les reglages n'ont de sens que pour une application ouverte, et ne retiennent rien a la fermeture.
            SidecarEvent::ShellSettings(settings) if self.phase == Phase::Ready => {
                vec![Action::ApplyShellSettings(settings)]
            }
            SidecarEvent::Stopped { clean } => {
                self.stopped_clean = Some(clean);
                if self.phase == Phase::Closing {
                    vec![Action::Arm(Timer::Grace, self.timeouts.grace)]
                } else {
                    Vec::new()
                }
            }
            // `ready` ou `failed` hors du demarrage, `stopping` pendant une fermeture ou un echec : rien a faire.
            _ => Vec::new(),
        }
    }

    fn on_failed(&mut self, state: &str) -> Vec<Action> {
        self.phase = Phase::Failed;
        self.failure = FailureKind::from_wire(state);
        let mut actions = vec![Action::Disarm(Timer::Startup)];
        match self.port {
            // La porte du serveur repond avant son routeur : sa page affiche l'etat.
            Some(port) => actions.push(Action::LoadServerPage { port }),
            None => actions.push(Action::ShowScreen(self.failure_screen())),
        }
        actions
    }

    /// L'ecran embarque d'un echec : sa cause si le serveur l'a nommee, sinon l'arret.
    fn failure_screen(&self) -> Screen {
        self.failure.map_or(Screen::Stopped, Screen::Failed)
    }

    fn on_exited(&mut self) -> Vec<Action> {
        self.process_alive = false;
        let mut actions = vec![
            Action::Disarm(Timer::Startup),
            Action::Disarm(Timer::Close),
            Action::Disarm(Timer::Grace),
        ];
        match self.phase {
            Phase::Closing => {
                self.phase = Phase::Done;
                actions.push(Action::ExitApp);
            }
            Phase::Failed => {
                self.phase = Phase::Dead;
                // La page du serveur n'existe plus : l'ecran embarque prend le relais, sauf s'il est deja la.
                let screen = self.failure_shown.unwrap_or_else(|| self.failure_screen());
                if self.failure_shown != Some(screen) {
                    actions.push(Action::ShowScreen(screen));
                }
            }
            Phase::Starting | Phase::Listening | Phase::Ready => {
                self.phase = Phase::Dead;
                actions.push(Action::ShowScreen(Screen::Stopped));
            }
            Phase::Dead | Phase::Done => {}
        }
        actions
    }

    /// La commande de synchronisation n'est ecrite que pour un serveur pret : avant `ready`, pendant la
    /// fermeture ou apres un echec, personne ne l'executerait.
    fn on_sync_requested(&mut self) -> Vec<Action> {
        if self.phase == Phase::Ready && self.process_alive {
            vec![Action::SendSync]
        } else {
            Vec::new()
        }
    }

    fn on_close_requested(&mut self) -> Vec<Action> {
        match self.phase {
            Phase::Closing | Phase::Done => Vec::new(),
            _ if !self.process_alive => {
                self.phase = Phase::Done;
                vec![Action::ExitApp]
            }
            _ => {
                self.phase = Phase::Closing;
                vec![
                    Action::ShowScreen(Screen::Stopping),
                    Action::CloseStdin,
                    Action::Disarm(Timer::Startup),
                    Action::Arm(Timer::Close, self.timeouts.close),
                ]
            }
        }
    }

    fn on_timer(&mut self, timer: Timer) -> Vec<Action> {
        match (timer, self.phase) {
            // Pas de `ready` dans le delai : echec "delai", le serveur est tue.
            (Timer::Startup, Phase::Starting | Phase::Listening) => {
                self.phase = Phase::Failed;
                self.killed = true;
                let screen = Screen::Failed(FailureKind::Timeout);
                self.failure_shown = Some(screen);
                vec![Action::Kill, Action::ShowScreen(screen)]
            }
            // Le serveur n'a pas fini son arret dans le budget : kill, puis sortie apres un delai de grace.
            (Timer::Close, Phase::Closing) => {
                self.killed = true;
                vec![Action::Kill, Action::Arm(Timer::Grace, self.timeouts.grace)]
            }
            // `stopped` recu (ou kill fait) et le processus tarde a sortir : on ne retient plus la fermeture.
            (Timer::Grace, Phase::Closing) => {
                self.phase = Phase::Done;
                let mut actions = Vec::new();
                if self.process_alive && !self.killed {
                    self.killed = true;
                    actions.push(Action::Kill);
                }
                actions.push(Action::ExitApp);
                actions
            }
            _ => Vec::new(),
        }
    }
}

#[cfg(test)]
mod tests;
