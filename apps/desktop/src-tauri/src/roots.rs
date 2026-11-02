//! Les trois racines posees au serveur (ADR 0045 D-1, D-14 du plan).
//!
//! - ressources : le dossier de l'application installee (config/, static/, front compile) ;
//! - donnees    : `%LOCALAPPDATA%\LevelUp`, jamais supprimee par le desinstalleur ;
//! - journaux   : `<donnees>\logs`.
//!
//! `LEVELUP_RESOURCES_DIR`, `LEVELUP_DATA_DIR` et `LEVELUP_WEB_DIST`, s'ils sont poses dans
//! l'environnement de la coquille, l'emportent : c'est le point d'entree du developpement (`desktop-dev.ps1`) et des
//! tests, qui lancent la coquille sur la racine du depot et sur un dossier de donnees jetable.

use std::ffi::OsString;
use std::fmt;
use std::fs;
use std::io;
use std::path::PathBuf;

pub const ENV_RESOURCES_DIR: &str = "LEVELUP_RESOURCES_DIR";
pub const ENV_DATA_DIR: &str = "LEVELUP_DATA_DIR";
pub const ENV_LOGS_DIR: &str = "LEVELUP_LOGS_DIR";
/// Dossier du front compile que le serveur sert (sa variable propre). Hors du contrat D-1, qui ne
/// nomme que les trois racines : la coquille la transmet seulement si SA propre environnement la
/// porte (developpement, tests) ; l'emplacement du front installe est une decision du lot L17.
pub const ENV_WEB_DIST: &str = "LEVELUP_WEB_DIST";
/// Dossier de donnees sous `%LOCALAPPDATA%`.
const DATA_DIR_NAME: &str = "LevelUp";
/// Dossier des journaux sous la racine de donnees.
const LOGS_DIR_NAME: &str = "logs";

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Roots {
    pub resources: PathBuf,
    pub data: PathBuf,
    pub logs: PathBuf,
    /// Front compile a servir, quand l'environnement de la coquille le designe.
    pub web_dist: Option<PathBuf>,
}

#[derive(Debug, PartialEq, Eq)]
pub enum RootsError {
    /// Ni `LEVELUP_DATA_DIR` ni `%LOCALAPPDATA%` : aucun endroit ou ecrire.
    NoDataRoot,
    /// Un dossier pose par l'environnement n'est pas un chemin absolu : le serveur change de
    /// repertoire courant, un chemin relatif designerait autre chose.
    NotAbsolute { var: &'static str, value: PathBuf },
}

impl fmt::Display for RootsError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::NoDataRoot => {
                write!(
                    f,
                    "{ENV_DATA_DIR} et LOCALAPPDATA sont absents : pas de dossier de donnees"
                )
            }
            Self::NotAbsolute { var, value } => {
                write!(f, "{var} doit etre un chemin absolu, recu {}", value.display())
            }
        }
    }
}

impl std::error::Error for RootsError {}

/// Calcule les racines. `env` lit une variable ; `local_app_data` est `%LOCALAPPDATA%` ;
/// `bundled_resources` est le dossier des ressources de l'application installee.
pub fn resolve(
    env: impl Fn(&str) -> Option<OsString>,
    local_app_data: Option<PathBuf>,
    bundled_resources: PathBuf,
) -> Result<Roots, RootsError> {
    let from_env = |var: &'static str| -> Result<Option<PathBuf>, RootsError> {
        match env(var).filter(|v| !v.is_empty()) {
            None => Ok(None),
            Some(v) => {
                let path = PathBuf::from(v);
                if path.is_absolute() {
                    Ok(Some(path))
                } else {
                    Err(RootsError::NotAbsolute { var, value: path })
                }
            }
        }
    };
    let resources = from_env(ENV_RESOURCES_DIR)?.unwrap_or(bundled_resources);
    let data = match from_env(ENV_DATA_DIR)? {
        Some(path) => path,
        None => local_app_data.ok_or(RootsError::NoDataRoot)?.join(DATA_DIR_NAME),
    };
    let logs = data.join(LOGS_DIR_NAME);
    let web_dist = from_env(ENV_WEB_DIST)?;
    Ok(Roots {
        resources,
        data,
        logs,
        web_dist,
    })
}

/// Cree les dossiers de donnees et de journaux s'ils manquent.
pub fn ensure_dirs(roots: &Roots) -> io::Result<()> {
    fs::create_dir_all(&roots.data)?;
    fs::create_dir_all(&roots.logs)
}

/// Le chemin du journal de la coquille.
pub fn desktop_log_path(roots: &Roots) -> PathBuf {
    roots.logs.join("desktop.log")
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::collections::HashMap;

    fn env_of(pairs: &[(&str, &str)]) -> impl Fn(&str) -> Option<OsString> + use<> {
        let map: HashMap<String, String> = pairs.iter().map(|(k, v)| (k.to_string(), v.to_string())).collect();
        move |k| map.get(k).map(OsString::from)
    }

    fn abs(name: &str) -> PathBuf {
        std::env::temp_dir().join(name)
    }

    #[test]
    fn par_defaut_les_donnees_vivent_sous_localappdata() {
        let local = abs("local");
        let roots = resolve(env_of(&[]), Some(local.clone()), abs("res")).unwrap();
        assert_eq!(roots.data, local.join("LevelUp"));
        assert_eq!(roots.logs, local.join("LevelUp").join("logs"));
        assert_eq!(roots.resources, abs("res"));
    }

    #[test]
    fn l_environnement_de_la_coquille_l_emporte() {
        let (res, data) = (abs("depot"), abs("jetable"));
        let env = env_of(&[
            (ENV_RESOURCES_DIR, res.to_str().unwrap()),
            (ENV_DATA_DIR, data.to_str().unwrap()),
        ]);
        let roots = resolve(env, Some(abs("local")), abs("bundled")).unwrap();
        assert_eq!(
            roots,
            Roots {
                resources: res,
                data: data.clone(),
                logs: data.join("logs"),
                web_dist: None
            }
        );
    }

    #[test]
    fn une_variable_vide_est_ignoree_une_relative_est_refusee() {
        let roots = resolve(env_of(&[(ENV_DATA_DIR, "")]), Some(abs("local")), abs("res")).unwrap();
        assert_eq!(roots.data, abs("local").join("LevelUp"));
        let err = resolve(env_of(&[(ENV_DATA_DIR, "donnees")]), Some(abs("local")), abs("res")).unwrap_err();
        assert!(matches!(err, RootsError::NotAbsolute { var, .. } if var == ENV_DATA_DIR));
    }

    #[test]
    fn le_front_n_est_transmis_que_si_l_environnement_de_la_coquille_le_designe() {
        let dist = abs("front");
        let env = env_of(&[(ENV_WEB_DIST, dist.to_str().unwrap())]);
        assert_eq!(
            resolve(env, Some(abs("local")), abs("res")).unwrap().web_dist,
            Some(dist)
        );
        assert_eq!(
            resolve(env_of(&[]), Some(abs("local")), abs("res")).unwrap().web_dist,
            None
        );
        let err = resolve(env_of(&[(ENV_WEB_DIST, "dist")]), Some(abs("local")), abs("res")).unwrap_err();
        assert!(matches!(err, RootsError::NotAbsolute { var, .. } if var == ENV_WEB_DIST));
    }

    #[test]
    fn sans_aucun_dossier_de_donnees_c_est_une_erreur() {
        assert_eq!(
            resolve(env_of(&[]), None, abs("res")).unwrap_err(),
            RootsError::NoDataRoot
        );
    }

    #[test]
    fn les_dossiers_manquants_sont_crees() {
        let base = abs(&format!("levelup-roots-{}", std::process::id()));
        let data = base.join("d");
        let roots = resolve(env_of(&[(ENV_DATA_DIR, data.to_str().unwrap())]), None, abs("res")).unwrap();
        ensure_dirs(&roots).unwrap();
        assert!(roots.logs.is_dir());
        assert_eq!(desktop_log_path(&roots), roots.logs.join("desktop.log"));
        fs::remove_dir_all(&base).unwrap();
    }
}
