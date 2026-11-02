//! Les evenements du serveur (ADR 0045 D-2) : une ligne JSON chacun sur sa sortie standard.
//!
//! ```text
//! {"event":"listening","port":N}          le port est lie, avant les migrations
//! {"event":"ready","port":N}              le routeur est ouvert
//! {"event":"failed","state":"<etat>"}     le demarrage est refuse
//! {"event":"stopping"}                    la sequence d'arret commence
//! {"event":"stopped","clean":true|false}  la sequence est finie
//! ```
//!
//! Une ligne qui n'est pas un evenement (journal egare, etat futur inconnu) n'est jamais fatale :
//! l'appelant la journalise et l'ignore.

use serde_json::{Map, Value};

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
        other => Err(ParseError::UnknownEvent(other.to_string())),
    }
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
    fn decode_les_cinq_evenements() {
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
    }
}
