//! Le choix des notes de mise a jour selon la langue de la coquille.
//!
//! `latest.json` porte `notes` (la section du changelog dans la langue par defaut de
//! `locales.toml`, la seule cle que lit un client sans ce module) et, par langue, `notes_<code>`
//! (aujourd'hui `notes_en`, le code de la langue de repli). Le plugin d'update expose le document
//! entier (`Update::raw_json`) : le choix se fait ici, sans code de langue ecrit en dur.

use serde_json::Value;

use crate::catalogue::{DEFAULT_LOCALE, FALLBACK_LOCALE};

/// Les notes a montrer pour `locale`, ou `None` si le document n'en porte aucune. Pur.
///
/// - langue par defaut : `notes` ;
/// - sinon `notes_<locale>`, puis `notes_<langue de repli>`, puis `notes` (une release sans notes
///   dans la langue de l'utilisateur lui montre celles de la langue par defaut plutot que rien).
///
/// Une cle vide ou faite d'espaces ne compte pas.
pub fn notes_for_locale(locale: &str, document: &Value) -> Option<String> {
    let keys: Vec<String> = if locale == DEFAULT_LOCALE {
        vec!["notes".to_string()]
    } else {
        vec![
            format!("notes_{locale}"),
            format!("notes_{FALLBACK_LOCALE}"),
            "notes".to_string(),
        ]
    };
    keys.iter()
        .filter_map(|key| document.get(key).and_then(Value::as_str))
        .find(|notes| !notes.trim().is_empty())
        .map(str::to_string)
}

#[cfg(test)]
mod tests {
    use super::*;
    use serde_json::json;

    fn doc() -> Value {
        json!({ "version": "8.1.0", "notes": "defaut", "notes_en": "repli" })
    }

    #[test]
    fn la_langue_par_defaut_lit_notes() {
        assert_eq!(notes_for_locale(DEFAULT_LOCALE, &doc()).as_deref(), Some("defaut"));
    }

    #[test]
    fn la_langue_de_repli_lit_sa_cle() {
        if FALLBACK_LOCALE != DEFAULT_LOCALE {
            assert_eq!(notes_for_locale(FALLBACK_LOCALE, &doc()).as_deref(), Some("repli"));
        }
    }

    #[test]
    fn une_langue_sans_cle_propre_prend_la_langue_de_repli() {
        assert_eq!(notes_for_locale("zz", &doc()).as_deref(), Some("repli"));
    }

    #[test]
    fn une_langue_avec_sa_propre_cle_la_prend() {
        let document = json!({ "notes": "defaut", "notes_en": "repli", "notes_zz": "propre" });
        assert_eq!(notes_for_locale("zz", &document).as_deref(), Some("propre"));
    }

    #[test]
    fn sans_cle_de_repli_la_langue_par_defaut_sert() {
        let document = json!({ "notes": "defaut" });
        assert_eq!(notes_for_locale("zz", &document).as_deref(), Some("defaut"));
    }

    #[test]
    fn une_cle_vide_ne_masque_pas_la_suivante() {
        let document = json!({ "notes": "defaut", "notes_en": "  \n" });
        assert_eq!(notes_for_locale("zz", &document).as_deref(), Some("defaut"));
    }

    #[test]
    fn la_langue_par_defaut_ne_lit_jamais_les_autres_cles() {
        let document = json!({ "notes_en": "repli" });
        assert_eq!(notes_for_locale(DEFAULT_LOCALE, &document), None);
    }

    #[test]
    fn un_document_sans_notes_rend_rien() {
        assert_eq!(notes_for_locale("zz", &json!({ "version": "8.1.0" })), None);
    }
}
