//! Le relais de la tache planifiee (ADR 0045, L21.e) : `LevelUp.exe --refresh-only --data-dir <p>
//! --resources-dir <p> --logs-dir <p>`.
//!
//! La tache de Windows lance la coquille, pas le serveur : la coquille est un executable graphique
//! (aucune console), alors qu'un serveur lance par le planificateur ouvrirait une fenetre de console
//! visible au premier plan. Elle lance le serveur voisin en `-mode refresh-only`, sans fenetre
//! (`CREATE_NO_WINDOW`) et sans flux, attend sa fin et sort avec son code. Rien d'autre : ni instance
//! unique, ni fenetre, ni plugin, ni decision (le verrou d'instance du serveur dit si l'application
//! tourne deja).

use std::ffi::OsString;
use std::fmt;
use std::path::{Path, PathBuf};
use std::process::{Command, Stdio};

use crate::logfile::{LogSink, MAX_LOG_BYTES, Source};
use crate::roots::{Roots, desktop_log_path};
use crate::sidecar::{apply_roots_env, hide_window};

/// Le drapeau qui demande le relais.
pub const FLAG: &str = "--refresh-only";
const FLAG_DATA_DIR: &str = "--data-dir";
const FLAG_RESOURCES_DIR: &str = "--resources-dir";
const FLAG_LOGS_DIR: &str = "--logs-dir";
/// Les arguments du serveur : le mode qui fait tourner les jetons et sort.
pub const SERVER_ARGS: [&str; 2] = ["-mode", "refresh-only"];
/// Code de sortie quand le serveur n'a pas pu etre lance ou attendu.
pub const FAILURE_CODE: i32 = 1;

/// Pourquoi les arguments sont refuses.
#[derive(Debug, PartialEq, Eq)]
pub enum ParseError {
    /// Un drapeau n'est suivi d'aucune valeur.
    MissingValue(&'static str),
    /// Une racine est absente.
    MissingRoot(&'static str),
    /// Une racine n'est pas un chemin absolu : le serveur change de repertoire courant.
    NotAbsolute { flag: &'static str, value: PathBuf },
    /// Un argument que le relais ne connait pas.
    Unknown(OsString),
}

impl fmt::Display for ParseError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::MissingValue(flag) => write!(f, "{flag} attend une valeur"),
            Self::MissingRoot(flag) => write!(f, "{flag} est obligatoire"),
            Self::NotAbsolute { flag, value } => {
                write!(f, "{flag} doit etre un chemin absolu, recu {}", value.display())
            }
            Self::Unknown(arg) => write!(f, "argument inconnu : {}", arg.to_string_lossy()),
        }
    }
}

impl std::error::Error for ParseError {}

/// Vrai quand `args` (sans le nom du programme) demandent le relais.
pub fn requested(args: impl IntoIterator<Item = OsString>) -> bool {
    args.into_iter().any(|arg| arg == FLAG)
}

/// Lit `args` (sans le nom du programme) : le drapeau `--refresh-only` et les trois racines, absolues.
pub fn parse(args: impl IntoIterator<Item = OsString>) -> Result<Roots, ParseError> {
    let (mut data, mut resources, mut logs) = (None, None, None);
    let mut args = args.into_iter();
    while let Some(arg) = args.next() {
        let slot = match arg.to_str() {
            Some(FLAG) => continue,
            Some(FLAG_DATA_DIR) => (&mut data, FLAG_DATA_DIR),
            Some(FLAG_RESOURCES_DIR) => (&mut resources, FLAG_RESOURCES_DIR),
            Some(FLAG_LOGS_DIR) => (&mut logs, FLAG_LOGS_DIR),
            _ => return Err(ParseError::Unknown(arg)),
        };
        *slot.0 = Some(args.next().ok_or(ParseError::MissingValue(slot.1))?);
    }
    Ok(Roots {
        data: root(data, FLAG_DATA_DIR)?,
        resources: root(resources, FLAG_RESOURCES_DIR)?,
        logs: root(logs, FLAG_LOGS_DIR)?,
        web_dist: None,
    })
}

fn root(value: Option<OsString>, flag: &'static str) -> Result<PathBuf, ParseError> {
    let value = value.ok_or(ParseError::MissingRoot(flag))?;
    let path = PathBuf::from(value);
    if path.is_absolute() {
        Ok(path)
    } else {
        Err(ParseError::NotAbsolute { flag, value: path })
    }
}

/// La commande du serveur : `-mode refresh-only`, l'environnement de la coquille sans ses `LEVELUP_*` et
/// `SPNKR_*`, les trois racines posees, la racine des ressources comme repertoire courant, aucun flux, aucune
/// fenetre. `inherited` est l'environnement de la coquille (injecte pour les tests).
pub fn command(server_exe: &Path, roots: &Roots, inherited: impl IntoIterator<Item = (OsString, OsString)>) -> Command {
    let mut cmd = Command::new(server_exe);
    cmd.args(SERVER_ARGS).current_dir(&roots.resources);
    apply_roots_env(&mut cmd, inherited, roots);
    cmd.stdin(Stdio::null()).stdout(Stdio::null()).stderr(Stdio::null());
    hide_window(&mut cmd);
    cmd
}

/// Ecrit `text` dans le journal de la coquille ; sans journal, sur la sortie d'erreur (invisible en release).
fn note(log: Option<&LogSink>, text: &str) {
    match log {
        Some(log) => log.write_line(Source::Shell, text),
        None => eprintln!("{text}"),
    }
}

/// Lance le serveur, l'attend et rend son code de sortie ; un lancement impossible rend [`FAILURE_CODE`].
/// Le debut, l'echec eventuel et le code de sortie vont au journal de la coquille (`log`) : en release la
/// coquille n'a pas de console, le journal est la seule trace d'une tache planifiee qui echoue.
pub fn relay(server_exe: &Path, roots: &Roots, log: Option<&LogSink>) -> i32 {
    note(log, &format!("refresh-only : debut, serveur {}", server_exe.display()));
    let code = match command(server_exe, roots, std::env::vars_os()).status() {
        Ok(status) => status.code().unwrap_or(FAILURE_CODE),
        Err(e) => {
            note(
                log,
                &format!("refresh-only : lancement de {} impossible : {e}", server_exe.display()),
            );
            FAILURE_CODE
        }
    };
    note(log, &format!("refresh-only : fin, code de sortie {code}"));
    code
}

/// Le relais complet : lit `args` (sans le nom du programme), cherche le serveur voisin de l'executable `server_name`
/// et rend le code de sortie du processus (a passer a `std::process::exit`).
///
/// Des arguments illisibles n'ont pas de racines, donc pas de journal ou ecrire : seul cet echec va sur la sortie
/// d'erreur (invisible en release, visible depuis une console). Une fois les racines lues, tout va au journal.
pub fn run(args: impl IntoIterator<Item = OsString>, server_name: &str) -> i32 {
    let roots = match parse(args) {
        Ok(roots) => roots,
        Err(e) => {
            eprintln!("refresh-only : {e}");
            return FAILURE_CODE;
        }
    };
    let log = match LogSink::open(desktop_log_path(&roots), MAX_LOG_BYTES) {
        Ok(log) => Some(log),
        Err(e) => {
            eprintln!("refresh-only : journal de la coquille illisible : {e}");
            None
        }
    };
    match std::env::current_exe() {
        Ok(exe) => relay(&exe.with_file_name(server_name), &roots, log.as_ref()),
        Err(e) => {
            note(log.as_ref(), &format!("refresh-only : chemin de l'executable : {e}"));
            FAILURE_CODE
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::roots::{ENV_DATA_DIR, ENV_LOGS_DIR, ENV_RESOURCES_DIR};
    use std::collections::HashMap;
    use std::ffi::OsStr;

    fn os(s: impl Into<OsString>) -> OsString {
        s.into()
    }

    fn base() -> PathBuf {
        std::env::temp_dir()
    }

    fn valid_args() -> Vec<OsString> {
        let b = base();
        vec![
            os(FLAG),
            os(FLAG_DATA_DIR),
            b.join("data").into_os_string(),
            os(FLAG_RESOURCES_DIR),
            b.join("res").into_os_string(),
            os(FLAG_LOGS_DIR),
            b.join("data").join("logs").into_os_string(),
        ]
    }

    fn roots() -> Roots {
        let b = base();
        Roots {
            resources: b.join("res"),
            data: b.join("data"),
            logs: b.join("data").join("logs"),
            web_dist: None,
        }
    }

    #[test]
    fn requested_reconnait_le_drapeau_seulement() {
        assert!(requested(valid_args()));
        assert!(requested([os("--minimized"), os(FLAG)]));
        assert!(!requested([os("--minimized")]));
        assert!(!requested(Vec::<OsString>::new()));
    }

    #[test]
    fn parse_lit_les_trois_racines_absolues_dans_n_importe_quel_ordre() {
        assert_eq!(parse(valid_args()), Ok(roots()));
        let b = base();
        let shuffled = vec![
            os(FLAG_LOGS_DIR),
            b.join("data").join("logs").into_os_string(),
            os(FLAG_DATA_DIR),
            b.join("data").into_os_string(),
            os(FLAG),
            os(FLAG_RESOURCES_DIR),
            b.join("res").into_os_string(),
        ];
        assert_eq!(parse(shuffled), Ok(roots()));
    }

    #[test]
    fn parse_garde_les_chemins_avec_des_espaces_et_des_caracteres_de_cmd() {
        let weird = base().join("Test User").join("a&b%c^d!e");
        let mut args = valid_args();
        args[2] = weird.clone().into_os_string();
        assert_eq!(parse(args).unwrap().data, weird);
    }

    #[test]
    fn parse_refuse_une_racine_absente() {
        let mut args = valid_args();
        args.truncate(5); // plus de --logs-dir
        assert_eq!(parse(args), Err(ParseError::MissingRoot(FLAG_LOGS_DIR)));
        assert_eq!(parse([os(FLAG)]), Err(ParseError::MissingRoot(FLAG_DATA_DIR)));
    }

    #[test]
    fn parse_refuse_un_drapeau_sans_valeur() {
        let mut args = valid_args();
        args.truncate(6); // --logs-dir en dernier, sans valeur
        assert_eq!(parse(args), Err(ParseError::MissingValue(FLAG_LOGS_DIR)));
    }

    #[test]
    fn parse_refuse_un_chemin_relatif() {
        let mut args = valid_args();
        args[4] = os("relatif");
        assert_eq!(
            parse(args),
            Err(ParseError::NotAbsolute {
                flag: FLAG_RESOURCES_DIR,
                value: PathBuf::from("relatif"),
            })
        );
    }

    #[test]
    fn parse_refuse_un_argument_inconnu() {
        let mut args = valid_args();
        args.push(os("--autre"));
        assert_eq!(parse(args), Err(ParseError::Unknown(os("--autre"))));
    }

    fn envs(cmd: &Command) -> HashMap<OsString, Option<OsString>> {
        cmd.get_envs()
            .map(|(k, v)| (k.to_os_string(), v.map(OsStr::to_os_string)))
            .collect()
    }

    #[test]
    fn la_commande_lance_le_mode_refresh_only_dans_la_racine_des_ressources() {
        let roots = roots();
        let server = base().join("levelup-server.exe");
        let cmd = command(&server, &roots, Vec::new());
        assert_eq!(cmd.get_program(), server.as_os_str());
        let args: Vec<&OsStr> = cmd.get_args().collect();
        assert_eq!(args, vec![OsStr::new("-mode"), OsStr::new("refresh-only")]);
        assert_eq!(cmd.get_current_dir(), Some(roots.resources.as_path()));
    }

    #[test]
    fn la_commande_pose_les_trois_racines_et_aucun_secret() {
        let roots = roots();
        let env = envs(&command(&base().join("s.exe"), &roots, Vec::new()));
        assert_eq!(
            env[&os(ENV_RESOURCES_DIR)],
            Some(roots.resources.clone().into_os_string())
        );
        assert_eq!(env[&os(ENV_DATA_DIR)], Some(roots.data.clone().into_os_string()));
        assert_eq!(env[&os(ENV_LOGS_DIR)], Some(roots.logs.clone().into_os_string()));
        for secret in ["LEVELUP_DESKTOP_SECRET", "LEVELUP_SHELL_EXE", "LEVELUP_WEB_DIST"] {
            assert!(!env.contains_key(&os(secret)), "{secret} n'est pas pose au relais");
        }
    }

    #[test]
    fn la_commande_retire_les_variables_heritees_mais_garde_le_systeme() {
        let inherited = vec![
            (os("LEVELUP_API_PORT"), os("1")),
            (os("levelup_auth_mode"), os("none")),
            (os("SPNKR_OAUTH_REFRESH_TOKEN_X"), os("t")),
            (os(ENV_DATA_DIR), os("ailleurs")),
            (os("SystemRoot"), os("C:\\Windows")),
            (os("PATH"), os("C:\\x")),
        ];
        let roots = roots();
        let env = envs(&command(&base().join("s.exe"), &roots, inherited));
        for stripped in ["LEVELUP_API_PORT", "levelup_auth_mode", "SPNKR_OAUTH_REFRESH_TOKEN_X"] {
            assert_eq!(env.get(&os(stripped)), Some(&None), "{stripped} doit etre retiree");
        }
        // La racine heritee est ecrasee par celle des arguments, jamais transmise telle quelle.
        assert_eq!(env[&os(ENV_DATA_DIR)], Some(roots.data.into_os_string()));
        assert!(!env.contains_key(&os("SystemRoot")) && !env.contains_key(&os("PATH")));
    }

    #[test]
    fn un_serveur_absent_rend_un_code_non_nul() {
        let absent = base().join("levelup-test-absent").join("levelup-server.exe");
        assert_eq!(relay(&absent, &roots(), None), FAILURE_CODE);
    }

    #[test]
    fn des_arguments_invalides_rendent_un_code_non_nul_sans_rien_lancer() {
        assert_eq!(run([os(FLAG)], "levelup-server.exe"), FAILURE_CODE);
    }
}
