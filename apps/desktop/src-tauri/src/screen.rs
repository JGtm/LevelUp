//! Les ecrans que la coquille affiche elle-meme (A4) : ceux des etats ou le serveur ne repond pas
//! (demarrage, fermeture, serveur mort, echecs). Le reste est la page web du serveur.

/// Cause d'un echec de demarrage : les etats du contrat (D-2) plus le delai propre a la coquille.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum FailureKind {
    Locked,
    MigrationFailed,
    SchemaNewer,
    Port,
    /// Le paquet de ressources manque ou n'est pas celui du moteur, et son telechargement a echoue (L24).
    ResourcesIncompatible,
    /// Pas de `ready` dans le delai : decide par la coquille, jamais annonce par le serveur.
    Timeout,
}

impl FailureKind {
    /// Tous les cas, dans l'ordre du catalogue.
    pub const ALL: [FailureKind; 6] = [
        Self::Locked,
        Self::MigrationFailed,
        Self::SchemaNewer,
        Self::Port,
        Self::ResourcesIncompatible,
        Self::Timeout,
    ];

    /// Range l'etat ecrit par le serveur ; `None` pour un etat que cette version ne connait pas.
    /// Le delai n'est pas un etat du serveur : il n'a pas de forme sur le fil.
    pub fn from_wire(state: &str) -> Option<Self> {
        match state {
            "locked" => Some(Self::Locked),
            "migration_failed" => Some(Self::MigrationFailed),
            "schema_newer" => Some(Self::SchemaNewer),
            "port" => Some(Self::Port),
            "resources_incompatible" => Some(Self::ResourcesIncompatible),
            _ => None,
        }
    }

    /// Identifiant dans le catalogue (le meme que celui de `GET /api/v1/desktop/strings`).
    pub fn message_key(self) -> &'static str {
        match self {
            Self::Locked => "failed_locked",
            Self::MigrationFailed => "failed_migration_failed",
            Self::SchemaNewer => "failed_schema_newer",
            Self::Port => "failed_port",
            Self::ResourcesIncompatible => "failed_resources_incompatible",
            Self::Timeout => "failed_timeout",
        }
    }
}

/// Un ecran embarque.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Screen {
    /// Logo et animation d'attente, jusqu'a `ready`.
    Starting,
    /// Fermeture en cours : le serveur termine ses ecritures.
    Stopping,
    /// Le serveur est arrete (arret termine ou mort inattendue).
    Stopped,
    Failed(FailureKind),
}

impl Screen {
    /// Tous les ecrans, pour les tests de completude.
    pub fn all() -> Vec<Screen> {
        let mut all = vec![Self::Starting, Self::Stopping, Self::Stopped];
        all.extend(FailureKind::ALL.map(Self::Failed));
        all
    }

    pub fn message_key(self) -> &'static str {
        match self {
            Self::Starting => "starting",
            Self::Stopping => "stopping",
            Self::Stopped => "stopped",
            Self::Failed(kind) => kind.message_key(),
        }
    }

    /// Vrai pour les ecrans d'attente (animation), faux pour les ecrans terminaux.
    pub fn is_waiting(self) -> bool {
        matches!(self, Self::Starting | Self::Stopping)
    }

    /// Chemin de l'ecran sous le protocole embarque (`/<cle du catalogue>`).
    pub fn path(self) -> String {
        format!("/{}", self.message_key())
    }

    /// Operation inverse de [`Screen::path`] ; `None` pour tout autre chemin.
    pub fn from_path(path: &str) -> Option<Screen> {
        let key = path.strip_prefix('/')?;
        Self::all().into_iter().find(|screen| screen.message_key() == key)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::catalogue;

    #[test]
    fn chaque_ecran_a_un_libelle_dans_chaque_langue() {
        for screen in Screen::all() {
            for code in catalogue::LOCALES {
                assert!(
                    catalogue::text(code, screen.message_key()).is_some(),
                    "{screen:?} sans libelle en {code}"
                );
            }
        }
    }

    #[test]
    fn le_chemin_fait_l_aller_retour() {
        for screen in Screen::all() {
            assert_eq!(Screen::from_path(&screen.path()), Some(screen));
        }
        assert_eq!(Screen::from_path("/logo.png"), None);
        assert_eq!(Screen::from_path("starting"), None);
        assert_eq!(Screen::from_path("/"), None);
    }

    #[test]
    fn les_etats_du_serveur_se_rangent_et_le_delai_n_a_pas_de_forme_sur_le_fil() {
        assert_eq!(FailureKind::from_wire("locked"), Some(FailureKind::Locked));
        assert_eq!(
            FailureKind::from_wire("migration_failed"),
            Some(FailureKind::MigrationFailed)
        );
        assert_eq!(FailureKind::from_wire("schema_newer"), Some(FailureKind::SchemaNewer));
        assert_eq!(FailureKind::from_wire("port"), Some(FailureKind::Port));
        assert_eq!(
            FailureKind::from_wire("resources_incompatible"),
            Some(FailureKind::ResourcesIncompatible)
        );
        assert_eq!(FailureKind::from_wire("timeout"), None);
        assert_eq!(FailureKind::from_wire("etat_futur"), None);
    }

    #[test]
    fn seuls_le_demarrage_et_la_fermeture_attendent() {
        let waiting: Vec<_> = Screen::all().into_iter().filter(|s| s.is_waiting()).collect();
        assert_eq!(waiting, vec![Screen::Starting, Screen::Stopping]);
    }
}
