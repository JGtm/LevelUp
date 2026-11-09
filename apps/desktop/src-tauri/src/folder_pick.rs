//! Le dossier ou la boite « Parcourir » s'ouvre (commande `pick_folder`, ADR 0045).
//!
//! La page envoie la valeur courante du champ : une saisie libre, qui peut etre vide, relative, perimee ou
//! nommer un dossier qui n'existe plus. Decision pure, sans fenetre : [`initial_dir`] rend le dossier le plus
//! proche qui existe, ou rien (la boite s'ouvre alors ou le systeme la met). La page ne decide ni du chemin
//! rendu ni de sa validite : le serveur valide le dossier a l'enregistrement, comme pour une saisie.

use std::path::{Component, Path, PathBuf, Prefix};

/// Le dossier de depart de la boite pour la saisie `raw` : la saisie elle-meme si c'est un dossier, sinon son
/// plus proche parent existant. `is_dir` dit si un chemin est un dossier (`Path::is_dir` en vrai).
///
/// Rien n'est rendu pour une saisie vide, relative (elle n'a pas de sens hors du dossier du serveur) ou
/// reseau : tester l'existence d'un `\\hote\partage` ouvre une connexion vers cet hote, ce qu'une valeur
/// venue de la page ne doit pas pouvoir provoquer.
pub fn initial_dir(raw: &str, is_dir: impl Fn(&Path) -> bool) -> Option<PathBuf> {
    let path = Path::new(raw.trim());
    if raw.trim().is_empty() || !path.is_absolute() || is_network_path(path) {
        return None;
    }
    path.ancestors()
        .find(|candidate| is_dir(candidate))
        .map(Path::to_path_buf)
}

/// Le chemin designe-t-il une ressource reseau (partage UNC, chemin de peripherique) ? Toujours faux hors Windows,
/// ou un chemin n'a pas de prefixe.
fn is_network_path(path: &Path) -> bool {
    matches!(
        path.components().next(),
        Some(Component::Prefix(prefix))
            if matches!(prefix.kind(), Prefix::UNC(..) | Prefix::VerbatimUNC(..) | Prefix::DeviceNS(..))
    )
}

#[cfg(test)]
mod tests {
    use super::*;

    /// Un chemin absolu valable sur toute plateforme.
    fn abs(parts: &[&str]) -> PathBuf {
        parts.iter().fold(std::env::temp_dir(), |acc, part| acc.join(part))
    }

    #[test]
    fn une_saisie_vide_ou_faite_d_espaces_ne_donne_rien() {
        assert_eq!(initial_dir("", |_| true), None);
        assert_eq!(initial_dir("   ", |_| true), None);
    }

    #[test]
    fn un_chemin_relatif_ne_donne_rien() {
        assert_eq!(initial_dir("sauvegardes", |_| true), None);
        assert_eq!(initial_dir("..", |_| true), None);
    }

    #[test]
    fn un_dossier_existant_se_rend_tel_quel() {
        let dir = abs(&["a", "b"]);
        let wanted = dir.clone();
        assert_eq!(initial_dir(&dir.to_string_lossy(), move |p| p == wanted), Some(dir));
    }

    #[test]
    fn les_espaces_autour_de_la_saisie_sont_ignores() {
        let dir = abs(&["a"]);
        let wanted = dir.clone();
        let raw = format!("  {}  ", dir.display());
        assert_eq!(initial_dir(&raw, move |p| p == wanted), Some(dir));
    }

    #[test]
    fn un_dossier_disparu_se_replie_sur_son_plus_proche_parent_existant() {
        let parent = abs(&["a"]);
        let missing = abs(&["a", "b", "c"]);
        let wanted = parent.clone();
        assert_eq!(
            initial_dir(&missing.to_string_lossy(), move |p| p == wanted),
            Some(parent)
        );
    }

    #[test]
    fn rien_n_existe_rien_n_est_rendu() {
        assert_eq!(initial_dir(&abs(&["a", "b"]).to_string_lossy(), |_| false), None);
    }

    #[cfg(windows)]
    #[test]
    fn un_chemin_reseau_n_est_jamais_teste() {
        for raw in [r"\\hote\partage\dossier", r"\\?\UNC\hote\partage", r"\\.\pipe\nom"] {
            let probed = std::cell::Cell::new(false);
            let found = initial_dir(raw, |_| {
                probed.set(true);
                true
            });
            assert_eq!(found, None, "{raw}");
            assert!(!probed.get(), "{raw} : l'existence a ete testee");
        }
    }

    #[cfg(windows)]
    #[test]
    fn un_lecteur_se_rend_quand_seul_le_lecteur_existe() {
        assert_eq!(
            initial_dir(r"C:\disparu\encore", |p| p == Path::new(r"C:\")),
            Some(PathBuf::from(r"C:\"))
        );
    }
}
