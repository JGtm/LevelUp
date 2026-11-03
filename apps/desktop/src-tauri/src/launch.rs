//! Les arguments de lancement de la coquille (lot L21.d) : le seul qui compte est `--minimized`, que pose
//! l'entree de demarrage avec Windows. Decision pure : l'argument donne l'etat initial de la fenetre.

/// L'argument que l'entree de demarrage avec Windows ajoute : demarrer dans la zone de notification.
pub const MINIMIZED_ARG: &str = "--minimized";

/// Ou commence la fenetre principale.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum InitialWindow {
    /// Montree : lancement ordinaire.
    Visible,
    /// Cachee dans la zone de notification : demarrage avec Windows (implique l'icone).
    HiddenInTray,
}

/// Vrai si la ligne de commande demande un demarrage reduit. `args` est la ligne complete (le premier
/// element est le chemin de l'executable, jamais interprete).
pub fn started_minimized<I, S>(args: I) -> bool
where
    I: IntoIterator<Item = S>,
    S: AsRef<str>,
{
    args.into_iter().skip(1).any(|arg| arg.as_ref() == MINIMIZED_ARG)
}

/// L'etat initial de la fenetre pour un lancement.
pub fn initial_window(minimized: bool) -> InitialWindow {
    if minimized {
        InitialWindow::HiddenInTray
    } else {
        InitialWindow::Visible
    }
}

/// La memoire de fenetre restaure-t-elle l'etat MAXIMISE au lancement ? Non quand la fenetre demarre cachee :
/// maximiser une fenetre cachee l'affiche (`SW_MAXIMIZE`), ce qui annule le demarrage reduit. L'etat maximise
/// est alors restaure au moment ou la fenetre est montree pour la premiere fois
/// ([`restores_maximized_when_first_shown`]).
pub fn restores_maximized_at_start(initial: InitialWindow) -> bool {
    initial == InitialWindow::Visible
}

/// L'etat maximise reste-t-il a restaurer a la premiere fois que la fenetre est montree ? Oui apres un demarrage
/// cache : la memoire de fenetre ne l'a pas restaure au lancement.
pub fn restores_maximized_when_first_shown(initial: InitialWindow) -> bool {
    !restores_maximized_at_start(initial)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn l_argument_minimized_demarre_dans_la_zone_de_notification() {
        assert!(started_minimized(["levelup.exe", "--minimized"]));
        assert_eq!(initial_window(true), InitialWindow::HiddenInTray);
    }

    #[test]
    fn sans_argument_le_lancement_est_ordinaire() {
        assert!(!started_minimized(["levelup.exe"]));
        assert!(!started_minimized(Vec::<String>::new()));
        assert_eq!(initial_window(false), InitialWindow::Visible);
    }

    #[test]
    fn un_demarrage_cache_ne_restaure_pas_l_etat_maximise_au_lancement() {
        // Sinon `maximize()` montre la fenetre cachee : le demarrage reduit ne l'est plus.
        assert!(!restores_maximized_at_start(initial_window(true)));
        assert!(restores_maximized_at_start(initial_window(false)));
    }

    #[test]
    fn un_demarrage_cache_restaure_l_etat_maximise_a_la_premiere_montre() {
        assert!(restores_maximized_when_first_shown(initial_window(true)));
        assert!(!restores_maximized_when_first_shown(initial_window(false)));
    }

    #[test]
    fn le_chemin_de_l_executable_n_est_jamais_un_argument() {
        // Un executable qui s'appellerait `--minimized` ne change rien : seul argv[1..] compte.
        assert!(!started_minimized(["--minimized"]));
    }

    #[test]
    fn un_autre_argument_ou_une_forme_proche_ne_suffit_pas() {
        assert!(!started_minimized(["levelup.exe", "--minimised"]));
        assert!(!started_minimized(["levelup.exe", "minimized"]));
        assert!(!started_minimized(["levelup.exe", "--minimized=false"]));
        assert!(started_minimized(["levelup.exe", "--autre", "--minimized"]));
    }
}
