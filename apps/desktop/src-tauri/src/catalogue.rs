//! Catalogue de repli des ecrans embarques (ADR 0045 D-13, A4).
//!
//! Le fichier genere par `build.rs` (`catalogue_gen.rs`) porte les libelles de
//! `messages/desktop.toml` dans CHAQUE langue de `config/locales.toml` ; ce module n'ecrit aucun
//! libelle ni aucun code de langue. La langue courante est celle du systeme, le repli est la
//! langue par defaut de `locales.toml`.

include!(concat!(env!("OUT_DIR"), "/catalogue_gen.rs"));

/// Range une etiquette de langue du systeme (BCP-47 ou POSIX : `xx-YY`, `xx_YY`, `xx`) dans une
/// langue du catalogue : le sous-etiquette primaire s'il existe, sinon [`DEFAULT_LOCALE`].
pub fn resolve_locale(tag: &str) -> &'static str {
    let primary = tag.split(['-', '_', '.']).next().unwrap_or("").to_ascii_lowercase();
    LOCALES
        .iter()
        .copied()
        .find(|code| *code == primary)
        .unwrap_or(DEFAULT_LOCALE)
}

/// Le libelle `key` dans la langue `locale` ; la langue par defaut si `locale` est inconnue.
/// `None` seulement pour une cle absente du catalogue (un test l'interdit pour tout ecran).
pub fn text(locale: &str, key: &str) -> Option<&'static str> {
    let table = |code: &str| ENTRIES.iter().find(|(c, _)| *c == code).map(|(_, entries)| *entries);
    let entries = table(locale).or_else(|| table(DEFAULT_LOCALE))?;
    entries.iter().find(|(k, _)| *k == key).map(|(_, v)| *v)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn chaque_langue_est_complete() {
        assert!(!LOCALES.is_empty() && LOCALES.contains(&DEFAULT_LOCALE));
        for code in LOCALES {
            for key in KEYS {
                let t = text(code, key).unwrap_or_else(|| panic!("{code}/{key} absent"));
                assert!(!t.trim().is_empty(), "{code}/{key} vide");
            }
        }
    }

    #[test]
    fn la_langue_du_systeme_se_range_par_son_code_primaire() {
        for code in LOCALES {
            assert_eq!(resolve_locale(code), *code);
            assert_eq!(resolve_locale(&format!("{code}-XX")), *code);
            assert_eq!(resolve_locale(&format!("{}_XX.UTF-8", code.to_uppercase())), *code);
        }
    }

    #[test]
    fn une_langue_inconnue_ou_vide_retombe_sur_le_defaut() {
        assert_eq!(resolve_locale("zz-ZZ"), DEFAULT_LOCALE);
        assert_eq!(resolve_locale(""), DEFAULT_LOCALE);
        assert_eq!(text("zz", KEYS[0]), text(DEFAULT_LOCALE, KEYS[0]));
        assert_eq!(text(DEFAULT_LOCALE, "cle_inexistante"), None);
    }
}
