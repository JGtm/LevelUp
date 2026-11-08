//! Les evenements du serveur (ADR 0045 D-2) : une ligne JSON chacun sur sa sortie standard.
//!
//! ```text
//! {"event":"listening","port":N}          le port est lie, avant les migrations
//! {"event":"ready","port":N}              le routeur est ouvert
//! {"event":"failed","state":"<etat>"}     le demarrage est refuse
//! {"event":"stopping"}                    la sequence d'arret commence
//! {"event":"stopped","clean":true|false}  la sequence est finie
//! {"event":"notification","title":"..","body":"..","kind":".."}
//!                                         une notification de la cloche a montrer comme notification
//!                                         Windows : texte deja compose dans la langue du joueur
//! {"event":"shell_settings","native_notifications":b,"close_to_tray":b,"start_with_windows":b,"locale":".."}
//!                                         les reglages que la coquille applique (zone de notification,
//!                                         demarrage avec Windows) et la langue de ses libelles ; ecrit une
//!                                         fois le serveur pret, puis a chaque changement
//! {"event":"resources","phase":"..","fingerprint":"..","code":"..",...}
//!                                         ou en est la preparation du paquet de ressources demandee par la
//!                                         commande `resources` (L24) ; la coquille attend `ready` ou `failed`
//! ```
//!
//! Une ligne qui n'est pas un evenement (journal egare, etat futur inconnu) n'est jamais fatale :
//! l'appelant la journalise et l'ignore.

use serde_json::{Map, Value};

use crate::resources::ResourcesEvent;
use crate::shell_settings::ShellSettings;

/// Un evenement du contrat.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum SidecarEvent {
    Listening {
        port: u16,
    },
    Ready {
        port: u16,
    },
    /// L'etat tel que le serveur l'ecrit ; [`crate::screen::FailureKind::from_wire`] le range.
    Failed {
        state: String,
    },
    Stopping,
    Stopped {
        clean: bool,
    },
    /// Une notification de la cloche, a montrer comme notification Windows (L21.b).
    Notification(Toast),
    /// Les reglages que la coquille applique (L21.c).
    ShellSettings(ShellSettings),
    /// L'avancement de la preparation d'un paquet de ressources (L24).
    Resources(ResourcesEvent),
}

/// Une notification a montrer : titre et corps sont ceux que le serveur a composes dans la langue du
/// joueur (la coquille n'ecrit aucun libelle), `kind` ne sert qu'au journal.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Toast {
    pub title: String,
    pub body: String,
    pub kind: String,
}

/// Pourquoi une ligne n'est pas un evenement.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum ParseError {
    /// Pas du JSON.
    NotJson,
    /// Du JSON, mais pas un objet portant un champ `event` texte.
    NotAnEvent,
    /// Un evenement que cette version de la coquille ne connait pas.
    UnknownEvent(String),
    /// Un champ obligatoire manque ou a le mauvais type.
    BadField(&'static str),
}

/// Decode une ligne de la sortie standard du serveur.
pub fn parse_line(line: &str) -> Result<SidecarEvent, ParseError> {
    let value: Value = serde_json::from_str(line.trim()).map_err(|_| ParseError::NotJson)?;
    let object = value.as_object().ok_or(ParseError::NotAnEvent)?;
    let name = object
        .get("event")
        .and_then(Value::as_str)
        .ok_or(ParseError::NotAnEvent)?;
    match name {
        "listening" => Ok(SidecarEvent::Listening { port: port_of(object)? }),
        "ready" => Ok(SidecarEvent::Ready { port: port_of(object)? }),
        "failed" => {
            let state = object
                .get("state")
                .and_then(Value::as_str)
                .ok_or(ParseError::BadField("state"))?;
            Ok(SidecarEvent::Failed {
                state: state.to_string(),
            })
        }
        "stopping" => Ok(SidecarEvent::Stopping),
        "stopped" => {
            let clean = object
                .get("clean")
                .and_then(Value::as_bool)
                .ok_or(ParseError::BadField("clean"))?;
            Ok(SidecarEvent::Stopped { clean })
        }
        "notification" => Ok(SidecarEvent::Notification(Toast {
            title: text_of(object, "title")?,
            body: text_of(object, "body")?,
            kind: text_of(object, "kind")?,
        })),
        "shell_settings" => Ok(SidecarEvent::ShellSettings(ShellSettings {
            native_notifications: flag_of(object, "native_notifications")?,
            close_to_tray: flag_of(object, "close_to_tray")?,
            start_with_windows: flag_of(object, "start_with_windows")?,
            locale: object
                .get("locale")
                .and_then(Value::as_str)
                .unwrap_or_default()
                .trim()
                .to_string(),
        })),
        "resources" => Ok(SidecarEvent::Resources(ResourcesEvent {
            phase: text_of(object, "phase")?,
            fingerprint: optional_text(object, "fingerprint"),
            code: optional_text(object, "code"),
        })),
        other => Err(ParseError::UnknownEvent(other.to_string())),
    }
}

/// Un champ texte facultatif : absent ou d'un autre type, il vaut le texte vide.
fn optional_text(object: &Map<String, Value>, field: &str) -> String {
    object
        .get(field)
        .and_then(Value::as_str)
        .unwrap_or_default()
        .to_string()
}

/// Un champ booleen obligatoire.
fn flag_of(object: &Map<String, Value>, field: &'static str) -> Result<bool, ParseError> {
    object
        .get(field)
        .and_then(Value::as_bool)
        .ok_or(ParseError::BadField(field))
}

/// Un champ texte obligatoire et non vide.
fn text_of(object: &Map<String, Value>, field: &'static str) -> Result<String, ParseError> {
    object
        .get(field)
        .and_then(Value::as_str)
        .map(str::trim)
        .filter(|t| !t.is_empty())
        .map(str::to_string)
        .ok_or(ParseError::BadField(field))
}

fn port_of(object: &Map<String, Value>) -> Result<u16, ParseError> {
    object
        .get("port")
        .and_then(Value::as_u64)
        .and_then(|p| u16::try_from(p).ok())
        .filter(|p| *p != 0)
        .ok_or(ParseError::BadField("port"))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn decode_les_sept_evenements() {
        assert_eq!(
            parse_line(r#"{"event":"listening","port":8123}"#),
            Ok(SidecarEvent::Listening { port: 8123 })
        );
        assert_eq!(
            parse_line(r#"{"event":"ready","port":8123}"#),
            Ok(SidecarEvent::Ready { port: 8123 })
        );
        assert_eq!(
            parse_line(r#"{"event":"failed","state":"locked"}"#),
            Ok(SidecarEvent::Failed { state: "locked".into() })
        );
        assert_eq!(parse_line(r#"{"event":"stopping"}"#), Ok(SidecarEvent::Stopping));
        assert_eq!(
            parse_line("{\"event\":\"stopped\",\"clean\":false}\r\n"),
            Ok(SidecarEvent::Stopped { clean: false })
        );
        assert_eq!(
            parse_line(r#"{"event":"notification","title":"Synchronisation terminee","body":"A jour.","kind":"sync"}"#),
            Ok(SidecarEvent::Notification(Toast {
                title: "Synchronisation terminee".into(),
                body: "A jour.".into(),
                kind: "sync".into(),
            }))
        );
    }

    #[test]
    fn decode_l_avancement_du_paquet_de_ressources() {
        assert_eq!(
            parse_line(
                r#"{"event":"resources","phase":"failed","fingerprint":"ab","expected":"cd","percent":12,"code":"resources_checksum_mismatch","restart_needed":false}"#
            ),
            Ok(SidecarEvent::Resources(ResourcesEvent {
                phase: "failed".into(),
                fingerprint: "ab".into(),
                code: "resources_checksum_mismatch".into(),
            }))
        );
        // Empreinte et code sont facultatifs ; la phase ne l'est pas.
        assert_eq!(
            parse_line(r#"{"event":"resources","phase":"idle"}"#),
            Ok(SidecarEvent::Resources(ResourcesEvent {
                phase: "idle".into(),
                fingerprint: String::new(),
                code: String::new(),
            }))
        );
        assert_eq!(
            parse_line(r#"{"event":"resources","fingerprint":"ab"}"#),
            Err(ParseError::BadField("phase"))
        );
    }

    #[test]
    fn decode_les_reglages_de_la_coquille() {
        assert_eq!(
            parse_line(
                r#"{"event":"shell_settings","native_notifications":false,"close_to_tray":true,"start_with_windows":false,"locale":"fr"}"#
            ),
            Ok(SidecarEvent::ShellSettings(ShellSettings {
                native_notifications: false,
                close_to_tray: true,
                start_with_windows: false,
                locale: "fr".into(),
            }))
        );
        // Une langue absente vaut vide (le menu retombe sur celle du systeme) ; un drapeau absent ou d'un
        // autre type refuse la ligne.
        assert_eq!(
            parse_line(
                r#"{"event":"shell_settings","native_notifications":true,"close_to_tray":true,"start_with_windows":true}"#
            ),
            Ok(SidecarEvent::ShellSettings(ShellSettings {
                native_notifications: true,
                close_to_tray: true,
                start_with_windows: true,
                locale: String::new(),
            }))
        );
        assert_eq!(
            parse_line(
                r#"{"event":"shell_settings","native_notifications":true,"close_to_tray":"oui","start_with_windows":true}"#
            ),
            Err(ParseError::BadField("close_to_tray"))
        );
        assert_eq!(
            parse_line(r#"{"event":"shell_settings","close_to_tray":true}"#),
            Err(ParseError::BadField("native_notifications"))
        );
    }

    #[test]
    fn une_ligne_qui_nest_pas_un_evenement_est_distinguee() {
        assert_eq!(parse_line("2026-11-01 boot: migrations"), Err(ParseError::NotJson));
        assert_eq!(parse_line("[1,2]"), Err(ParseError::NotAnEvent));
        assert_eq!(parse_line(r#"{"msg":"x"}"#), Err(ParseError::NotAnEvent));
        assert_eq!(parse_line(r#"{"event":7}"#), Err(ParseError::NotAnEvent));
        assert_eq!(
            parse_line(r#"{"event":"rebooted"}"#),
            Err(ParseError::UnknownEvent("rebooted".into()))
        );
    }

    #[test]
    fn les_champs_obligatoires_sont_verifies() {
        assert_eq!(
            parse_line(r#"{"event":"listening"}"#),
            Err(ParseError::BadField("port"))
        );
        assert_eq!(
            parse_line(r#"{"event":"ready","port":0}"#),
            Err(ParseError::BadField("port"))
        );
        assert_eq!(
            parse_line(r#"{"event":"ready","port":70000}"#),
            Err(ParseError::BadField("port"))
        );
        assert_eq!(parse_line(r#"{"event":"failed"}"#), Err(ParseError::BadField("state")));
        assert_eq!(parse_line(r#"{"event":"stopped"}"#), Err(ParseError::BadField("clean")));
        // Une notification sans titre, ou avec un champ vide ou d'un autre type, n'est pas montree.
        assert_eq!(
            parse_line(r#"{"event":"notification","body":"b","kind":"k"}"#),
            Err(ParseError::BadField("title"))
        );
        assert_eq!(
            parse_line(r#"{"event":"notification","title":"t","body":"  ","kind":"k"}"#),
            Err(ParseError::BadField("body"))
        );
        assert_eq!(
            parse_line(r#"{"event":"notification","title":"t","body":"b","kind":7}"#),
            Err(ParseError::BadField("kind"))
        );
    }
}
