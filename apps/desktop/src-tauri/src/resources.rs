//! Le paquet de ressources vu par la coquille (ADR 0045, amendement L24).
//!
//! La coquille ne telecharge rien : le serveur le fait (hotes de publication, signature, somme, mode
//! jeu). Elle ne fait que deux choses, ici sans tauri : lire dans `latest.json` (via
//! `Update::raw_json`) le bloc `resources` que la release publie, et attendre que le serveur dise que ce
//! paquet est pret AVANT d'installer le moteur qui l'exige.
//!
//! ```text
//!  mise a jour acceptee --> commande `resources <json>` sur la stdin du serveur
//!                       <-- evenements `resources` (phase, empreinte, code) sur sa stdout
//!      `ready` ou `current` --> l'installeur du moteur se telecharge et s'arme
//!      `failed`, serveur arrete, delai --> rien n'est installe, la raison est journalisee
//! ```

use std::sync::{Arc, Condvar, Mutex};
use std::time::{Duration, Instant};

use serde_json::Value;

/// La commande de la stdin du serveur qui demande de preparer un paquet (une ligne : `resources <json>`).
pub const COMMAND_RESOURCES: &str = "resources";
/// Variable qui porte au serveur la cle publique de mise a jour (le champ `pubkey` de `tauri.conf.json`) : le
/// serveur verifie avec elle la signature du paquet, la meme cle que celle de l'installeur.
pub const ENV_UPDATE_PUBKEY: &str = "LEVELUP_UPDATE_PUBKEY";
/// Delai maximal de l'attente d'un paquet : il couvre une longue partie (le telechargement attend sa fin).
pub const PREPARE_TIMEOUT: Duration = Duration::from_secs(6 * 60 * 60);

/// Le paquet que publie une release : son empreinte, et le bloc JSON complet que le serveur valide.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ResourcePack {
    pub fingerprint: String,
    /// Le bloc `resources` de `latest.json`, en une ligne.
    pub json: String,
}

/// Lit le bloc `resources` du document de mise a jour. `None` quand la release n'en porte pas (le moteur se
/// met alors a jour seul) ou qu'il est incomplet : empreinte, adresse et signature sont obligatoires.
pub fn pack_from_raw_json(raw: &Value) -> Option<ResourcePack> {
    let block = raw.get("resources")?.as_object()?;
    let text = |key: &str| block.get(key).and_then(Value::as_str).filter(|v| !v.trim().is_empty());
    let fingerprint = text("fingerprint")?.to_string();
    text("url")?;
    text("signature")?;
    let json = serde_json::to_string(block).ok()?;
    Some(ResourcePack { fingerprint, json })
}

/// Les variables supplementaires du serveur pour la cle de mise a jour : aucune quand la configuration n'en
/// porte pas (build de developpement), la cle sans espaces de bord sinon.
pub fn pubkey_env(pubkey: Option<&str>) -> Vec<(String, String)> {
    match pubkey.map(str::trim).filter(|key| !key.is_empty()) {
        Some(key) => vec![(ENV_UPDATE_PUBKEY.to_string(), key.to_string())],
        None => Vec::new(),
    }
}

/// La ligne ecrite sur la stdin du serveur.
pub fn command_line(pack: &ResourcePack) -> String {
    format!("{COMMAND_RESOURCES} {}", pack.json)
}

/// L'evenement `resources` du serveur : ou en est la preparation d'un paquet.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ResourcesEvent {
    /// `idle`, `current`, `waiting_game`, `downloading`, `verifying`, `ready` ou `failed`.
    pub phase: String,
    pub fingerprint: String,
    /// Code d'erreur fixe de la phase `failed`.
    pub code: String,
}

impl ResourcesEvent {
    /// Le paquet est utilisable par le moteur qui l'exige : installe (`current`) ou range (`ready`).
    pub fn is_ready(&self) -> bool {
        self.phase == "ready" || self.phase == "current"
    }

    pub fn is_failed(&self) -> bool {
        self.phase == "failed"
    }
}

#[derive(Debug, Default)]
struct BoardState {
    last: Option<ResourcesEvent>,
    failure: Option<String>,
    closed: bool,
}

/// Le tableau ou le superviseur depose les evenements `resources` et ou le fil de mise a jour attend.
#[derive(Clone, Default)]
pub struct ResourcesBoard(Arc<(Mutex<BoardState>, Condvar)>);

impl ResourcesBoard {
    fn state(&self) -> std::sync::MutexGuard<'_, BoardState> {
        self.0.0.lock().unwrap_or_else(|p| p.into_inner())
    }

    /// Oublie ce qui a ete vu : une nouvelle demande ne doit pas lire le verdict d'une precedente.
    pub fn reset(&self) {
        let mut state = self.state();
        state.last = None;
        state.failure = None;
    }

    /// Depose un evenement du serveur.
    pub fn publish(&self, event: ResourcesEvent) {
        self.state().last = Some(event);
        self.0.1.notify_all();
    }

    /// La demande n'a pas pu partir (serveur pas pret, tube ferme) : l'attente se termine tout de suite.
    pub fn fail(&self, reason: &str) {
        self.state().failure = Some(reason.to_string());
        self.0.1.notify_all();
    }

    /// Le serveur est parti : plus aucun evenement ne viendra.
    pub fn close(&self) {
        self.state().closed = true;
        self.0.1.notify_all();
    }

    /// Le dernier evenement vu.
    pub fn last(&self) -> Option<ResourcesEvent> {
        self.state().last.clone()
    }

    /// Attend que le serveur annonce le paquet `fingerprint` pret ; rend la raison en cas d'echec.
    pub fn wait_ready(&self, fingerprint: &str, timeout: Duration) -> Result<(), String> {
        let deadline = Instant::now() + timeout;
        let mut guard = self.state();
        loop {
            if let Some(reason) = guard.failure.clone() {
                return Err(reason);
            }
            if let Some(event) = guard.last.as_ref().filter(|e| e.fingerprint == fingerprint) {
                if event.is_ready() {
                    return Ok(());
                }
                if event.is_failed() {
                    return Err(format!("resources_failed code={}", event.code));
                }
            }
            if guard.closed {
                return Err("server_stopped".to_string());
            }
            let left = deadline.saturating_duration_since(Instant::now());
            if left.is_zero() {
                return Err("timeout".to_string());
            }
            guard = self.0.1.wait_timeout(guard, left).unwrap_or_else(|p| p.into_inner()).0;
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use serde_json::json;
    use std::thread;

    fn event(phase: &str, fingerprint: &str, code: &str) -> ResourcesEvent {
        ResourcesEvent {
            phase: phase.into(),
            fingerprint: fingerprint.into(),
            code: code.into(),
        }
    }

    #[test]
    fn le_bloc_de_la_release_est_lu_en_une_ligne() {
        let raw = json!({
            "version": "8.1.1",
            "resources": {"fingerprint": "ab", "url": "https://github.com/x", "sha256": "cd", "signature": "c2ln"}
        });
        let pack = pack_from_raw_json(&raw).expect("bloc complet");
        assert_eq!(pack.fingerprint, "ab");
        assert!(!pack.json.contains('\n'));
        let line = command_line(&pack);
        assert!(line.starts_with("resources {"));
        let back: Value = serde_json::from_str(line.strip_prefix("resources ").unwrap()).unwrap();
        assert_eq!(back["url"], "https://github.com/x");
        assert_eq!(back["sha256"], "cd");
    }

    #[test]
    fn un_bloc_absent_ou_incomplet_n_est_pas_un_paquet() {
        assert!(pack_from_raw_json(&json!({"version": "8.1.1"})).is_none());
        assert!(pack_from_raw_json(&json!({"resources": "oui"})).is_none());
        for missing in ["fingerprint", "url", "signature"] {
            let mut block = json!({"fingerprint": "ab", "url": "https://x", "signature": "s"});
            block.as_object_mut().unwrap().remove(missing);
            assert!(
                pack_from_raw_json(&json!({"resources": block})).is_none(),
                "sans {missing}"
            );
        }
        let blank = json!({"resources": {"fingerprint": " ", "url": "https://x", "signature": "s"}});
        assert!(pack_from_raw_json(&blank).is_none());
    }

    #[test]
    fn la_cle_de_mise_a_jour_part_au_serveur_seulement_si_elle_existe() {
        assert!(pubkey_env(None).is_empty());
        assert!(pubkey_env(Some("  ")).is_empty());
        assert_eq!(
            pubkey_env(Some(" cle \n")),
            vec![("LEVELUP_UPDATE_PUBKEY".to_string(), "cle".to_string())]
        );
    }

    #[test]
    fn l_attente_se_termine_sur_pret_ou_installe() {
        for phase in ["ready", "current"] {
            let board = ResourcesBoard::default();
            board.publish(event("downloading", "f", ""));
            board.publish(event(phase, "f", ""));
            assert_eq!(board.wait_ready("f", Duration::from_millis(50)), Ok(()));
        }
    }

    #[test]
    fn l_attente_ignore_un_autre_paquet_et_se_reveille_a_l_arrivee_de_l_evenement() {
        let board = ResourcesBoard::default();
        board.publish(event("ready", "autre", ""));
        let waiter = {
            let board = board.clone();
            thread::spawn(move || board.wait_ready("f", Duration::from_secs(5)))
        };
        thread::sleep(Duration::from_millis(50));
        board.publish(event("verifying", "f", ""));
        board.publish(event("ready", "f", ""));
        assert_eq!(waiter.join().unwrap(), Ok(()));
    }

    #[test]
    fn un_echec_un_arret_une_demande_perdue_et_le_delai_sont_des_raisons() {
        let board = ResourcesBoard::default();
        board.publish(event("failed", "f", "resources_signature_invalid"));
        assert_eq!(
            board.wait_ready("f", Duration::from_millis(50)),
            Err("resources_failed code=resources_signature_invalid".into())
        );

        let board = ResourcesBoard::default();
        board.close();
        assert_eq!(
            board.wait_ready("f", Duration::from_millis(50)),
            Err("server_stopped".into())
        );

        let board = ResourcesBoard::default();
        board.fail("stdin_closed");
        assert_eq!(
            board.wait_ready("f", Duration::from_millis(50)),
            Err("stdin_closed".into())
        );

        let board = ResourcesBoard::default();
        assert_eq!(board.wait_ready("f", Duration::from_millis(30)), Err("timeout".into()));
    }

    #[test]
    fn reset_oublie_le_verdict_d_une_demande_precedente() {
        let board = ResourcesBoard::default();
        board.publish(event("failed", "f", "x"));
        board.reset();
        assert!(board.last().is_none());
        assert_eq!(board.wait_ready("f", Duration::from_millis(30)), Err("timeout".into()));
    }
}
