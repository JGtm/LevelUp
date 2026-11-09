//! Le secret d'entree (ADR 0045 D-1, D-3) : 32 octets aleatoires tires a chaque lancement.
//!
//! Il voyage par l'environnement du serveur (`LEVELUP_DESKTOP_SECRET`) et par l'URL de la premiere
//! page, jamais par un argument de ligne de commande, jamais dans un journal : le type n'implemente
//! ni `Display` ni un `Debug` qui le montre.

use std::fmt;
use std::io;

/// Nombre d'octets aleatoires du secret (64 caracteres hexadecimaux).
pub const SECRET_BYTES: usize = 32;

#[derive(Clone, PartialEq, Eq)]
pub struct Secret(String);

impl Secret {
    /// Tire un secret du generateur du systeme.
    pub fn generate() -> io::Result<Self> {
        let mut bytes = [0u8; SECRET_BYTES];
        getrandom::fill(&mut bytes)
            .map_err(|e| io::Error::other(format!("aleatoire du systeme indisponible : {e}")))?;
        Ok(Self::from_bytes(&bytes))
    }

    fn from_bytes(bytes: &[u8]) -> Self {
        let mut hex = String::with_capacity(bytes.len() * 2);
        for b in bytes {
            hex.push(char::from_digit(u32::from(b >> 4), 16).expect("quartet"));
            hex.push(char::from_digit(u32::from(b & 0x0f), 16).expect("quartet"));
        }
        Secret(hex)
    }

    /// La valeur, pour l'environnement du serveur et l'URL d'entree. Ne jamais la journaliser.
    pub fn expose(&self) -> &str {
        &self.0
    }
}

impl fmt::Debug for Secret {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.write_str("Secret(<masque>)")
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn le_secret_fait_64_caracteres_hexadecimaux_et_change_a_chaque_tirage() {
        let a = Secret::generate().unwrap();
        let b = Secret::generate().unwrap();
        assert_eq!(a.expose().len(), SECRET_BYTES * 2);
        assert!(
            a.expose()
                .chars()
                .all(|c| c.is_ascii_hexdigit() && !c.is_ascii_uppercase())
        );
        assert_ne!(a, b);
    }

    #[test]
    fn l_encodage_hexadecimal_est_exact() {
        assert_eq!(Secret::from_bytes(&[0x00, 0x0f, 0xa5, 0xff]).expose(), "000fa5ff");
    }

    #[test]
    fn le_debug_ne_montre_pas_le_secret() {
        let s = Secret::generate().unwrap();
        assert!(!format!("{s:?}").contains(s.expose()));
        assert!(!format!("{:#?}", Some(&s)).contains(s.expose()));
    }
}
