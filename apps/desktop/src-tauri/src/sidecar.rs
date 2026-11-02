//! Lancement du serveur (le sidecar) par `std::process::Command` (ADR 0045 D-1, D-14).
//!
//! Pas le plugin shell de tauri : la coquille tient le tube de stdin (sa fermeture est l'ordre
//! d'arret, y compris quand la coquille meurt : l'OS ferme le tube), lit stdout ligne a ligne et
//! garde le controle du kill. Il n'y a pas de Job object (A2) : il tuerait le serveur a la mort de
//! la coquille, exactement quand l'EOF de stdin doit declencher l'arret propre.

use std::ffi::{OsStr, OsString};
use std::io;
use std::path::PathBuf;
use std::process::{Child, ChildStderr, ChildStdin, ChildStdout, Command, ExitStatus, Stdio};
use std::sync::Mutex;

use crate::roots::{ENV_DATA_DIR, ENV_LOGS_DIR, ENV_RESOURCES_DIR, ENV_WEB_DIST, Roots};
use crate::secret::Secret;

/// Variable qui porte le secret d'entree.
pub const ENV_DESKTOP_SECRET: &str = "LEVELUP_DESKTOP_SECRET";
/// Les arguments du serveur : le mode, et l'arret a la fermeture de stdin. Jamais le secret.
pub const SERVER_ARGS: [&str; 3] = ["-mode", "desktop", "-arret-sur-stdin"];
/// Prefixes d'environnement que le serveur ne doit pas heriter de la session de l'utilisateur :
/// un reglage de developpement oublie dans le terminal ne change pas le comportement de
/// l'application installee. Les racines et le secret sont poses explicitement.
const STRIPPED_PREFIXES: [&str; 2] = ["LEVELUP_", "SPNKR_"];
/// `CREATE_NO_WINDOW` : pas de fenetre de console pour un serveur lance par une application graphique.
#[cfg(windows)]
const CREATE_NO_WINDOW: u32 = 0x0800_0000;

/// Ce qu'il faut pour lancer le serveur.
pub struct SidecarSpec {
    pub exe: PathBuf,
    pub roots: Roots,
    pub secret: Secret,
    /// Variables supplementaires posees telles quelles apres celles du contrat. La coquille livree
    /// n'en pose aucune ; les tests s'en servent pour regler le faux serveur sans toucher a
    /// l'environnement du processus de test (partage entre ses fils).
    pub extra_env: Vec<(String, String)>,
}

/// Construit la commande. `inherited` est l'environnement de la coquille (injecte pour les tests) :
/// les variables `LEVELUP_*` et `SPNKR_*` qu'il porte sont retirees, puis les quatre variables du
/// contrat sont posees.
pub fn build_command(spec: &SidecarSpec, inherited: impl IntoIterator<Item = (OsString, OsString)>) -> Command {
    let mut cmd = Command::new(&spec.exe);
    cmd.args(SERVER_ARGS).current_dir(&spec.roots.resources);
    for (name, _) in inherited {
        if is_stripped(&name) {
            cmd.env_remove(&name);
        }
    }
    cmd.env(ENV_RESOURCES_DIR, &spec.roots.resources)
        .env(ENV_DATA_DIR, &spec.roots.data)
        .env(ENV_LOGS_DIR, &spec.roots.logs)
        .env(ENV_DESKTOP_SECRET, spec.secret.expose());
    if let Some(web_dist) = &spec.roots.web_dist {
        cmd.env(ENV_WEB_DIST, web_dist);
    }
    for (name, value) in &spec.extra_env {
        cmd.env(name, value);
    }
    cmd.stdin(Stdio::piped()).stdout(Stdio::piped()).stderr(Stdio::piped());
    #[cfg(windows)]
    {
        use std::os::windows::process::CommandExt;
        cmd.creation_flags(CREATE_NO_WINDOW);
    }
    cmd
}

/// Vrai pour une variable d'environnement a ne pas transmettre (insensible a la casse : les noms
/// d'environnement de Windows le sont).
fn is_stripped(name: &OsStr) -> bool {
    let upper = name.to_string_lossy().to_ascii_uppercase();
    STRIPPED_PREFIXES.iter().any(|prefix| upper.starts_with(prefix))
}

/// Le serveur lance : son processus et le bout ecrivain de son stdin.
pub struct Sidecar {
    child: Mutex<Child>,
    stdin: Mutex<Option<ChildStdin>>,
    pid: u32,
}

/// Un serveur lance et ses trois flux a vider.
pub struct Spawned {
    pub sidecar: Sidecar,
    pub stdout: ChildStdout,
    pub stderr: ChildStderr,
}

/// Lance le serveur avec l'environnement de la coquille.
pub fn spawn(spec: &SidecarSpec) -> io::Result<Spawned> {
    let mut child = build_command(spec, std::env::vars_os()).spawn()?;
    let missing = |name: &str| io::Error::other(format!("flux {name} du serveur indisponible"));
    let stdin = child.stdin.take().ok_or_else(|| missing("stdin"))?;
    let stdout = child.stdout.take().ok_or_else(|| missing("stdout"))?;
    let stderr = child.stderr.take().ok_or_else(|| missing("stderr"))?;
    let pid = child.id();
    Ok(Spawned {
        sidecar: Sidecar {
            child: Mutex::new(child),
            stdin: Mutex::new(Some(stdin)),
            pid,
        },
        stdout,
        stderr,
    })
}

impl Sidecar {
    pub fn pid(&self) -> u32 {
        self.pid
    }

    /// Ferme le stdin du serveur : l'ordre d'arret propre. Sans effet la seconde fois.
    pub fn close_stdin(&self) {
        let mut stdin = self.stdin.lock().unwrap_or_else(|p| p.into_inner());
        stdin.take();
    }

    /// Tue le serveur (TerminateProcess). Une erreur n'est pas fatale : un processus deja sorti
    /// n'est plus a tuer ; l'appelant la journalise.
    pub fn kill(&self) -> io::Result<()> {
        let mut child = self.child.lock().unwrap_or_else(|p| p.into_inner());
        match child.try_wait()? {
            Some(_) => Ok(()),
            None => child.kill(),
        }
    }

    /// `Some` quand le processus est sorti.
    pub fn try_wait(&self) -> io::Result<Option<ExitStatus>> {
        self.child.lock().unwrap_or_else(|p| p.into_inner()).try_wait()
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::collections::HashMap;

    fn spec() -> SidecarSpec {
        let base = std::env::temp_dir();
        SidecarSpec {
            exe: base.join("levelup-server.exe"),
            roots: Roots {
                resources: base.join("res"),
                data: base.join("data"),
                logs: base.join("data").join("logs"),
                web_dist: None,
            },
            secret: Secret::generate().unwrap(),
            extra_env: Vec::new(),
        }
    }

    fn os(s: &str) -> OsString {
        OsString::from(s)
    }

    #[test]
    fn les_arguments_sont_le_mode_et_l_arret_sur_stdin_sans_le_secret() {
        let spec = spec();
        let cmd = build_command(&spec, Vec::new());
        let args: Vec<&OsStr> = cmd.get_args().collect();
        assert_eq!(
            args,
            vec![
                OsStr::new("-mode"),
                OsStr::new("desktop"),
                OsStr::new("-arret-sur-stdin")
            ]
        );
        for arg in args {
            assert!(!arg.to_string_lossy().contains(spec.secret.expose()));
        }
    }

    #[test]
    fn le_repertoire_courant_est_la_racine_des_ressources() {
        let spec = spec();
        assert_eq!(
            build_command(&spec, Vec::new()).get_current_dir(),
            Some(spec.roots.resources.as_path())
        );
    }

    #[test]
    fn l_environnement_pose_les_trois_racines_et_le_secret() {
        let spec = spec();
        let cmd = build_command(&spec, Vec::new());
        let envs: HashMap<OsString, Option<OsString>> = cmd
            .get_envs()
            .map(|(k, v)| (k.to_os_string(), v.map(OsStr::to_os_string)))
            .collect();
        assert_eq!(
            envs[&os(ENV_RESOURCES_DIR)],
            Some(spec.roots.resources.clone().into_os_string())
        );
        assert_eq!(envs[&os(ENV_DATA_DIR)], Some(spec.roots.data.clone().into_os_string()));
        assert_eq!(envs[&os(ENV_LOGS_DIR)], Some(spec.roots.logs.clone().into_os_string()));
        assert_eq!(envs[&os(ENV_DESKTOP_SECRET)], Some(os(spec.secret.expose())));
    }

    #[test]
    fn le_front_est_pose_seulement_s_il_est_connu_et_jamais_herite() {
        let mut with_web = spec();
        let dist = std::env::temp_dir().join("front");
        with_web.roots.web_dist = Some(dist.clone());
        let envs = |spec: &SidecarSpec| -> HashMap<OsString, Option<OsString>> {
            let inherited = vec![(os(ENV_WEB_DIST), os("ailleurs"))];
            build_command(spec, inherited)
                .get_envs()
                .map(|(k, v)| (k.to_os_string(), v.map(OsStr::to_os_string)))
                .collect()
        };
        assert_eq!(envs(&with_web)[&os(ENV_WEB_DIST)], Some(dist.into_os_string()));
        // Sans designation de la coquille, la valeur heritee est retiree, jamais transmise.
        assert_eq!(envs(&spec())[&os(ENV_WEB_DIST)], None);
    }

    #[test]
    fn les_reglages_de_la_session_ne_sont_pas_transmis_mais_le_systeme_oui() {
        let inherited = vec![
            (os("LEVELUP_API_PORT"), os("1")),
            (os("levelup_auth_mode"), os("none")),
            (os("SPNKR_OAUTH_REFRESH_TOKEN_X"), os("t")),
            (os("SystemRoot"), os("C:\\Windows")),
            (os("PATH"), os("C:\\x")),
        ];
        let cmd = build_command(&spec(), inherited);
        let envs: HashMap<OsString, Option<OsString>> = cmd
            .get_envs()
            .map(|(k, v)| (k.to_os_string(), v.map(OsStr::to_os_string)))
            .collect();
        for stripped in ["LEVELUP_API_PORT", "levelup_auth_mode", "SPNKR_OAUTH_REFRESH_TOKEN_X"] {
            assert_eq!(envs.get(&os(stripped)), Some(&None), "{stripped} doit etre retiree");
        }
        assert!(!envs.contains_key(&os("SystemRoot")) && !envs.contains_key(&os("PATH")));
    }
}
