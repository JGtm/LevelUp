//! Rendu des ecrans embarques : un gabarit HTML (`ui/screen.html`), des libelles du catalogue.
//!
//! La page est servie par le binaire lui-meme (protocole embarque), jamais par le reseau : aucune
//! requete, aucun script, aucune dependance. Ni le gabarit ni ce module ne portent de libelle.

use crate::catalogue;
use crate::screen::Screen;

const TEMPLATE: &str = include_str!("../ui/screen.html");
const LOGO: &[u8] = include_bytes!("../ui/logo.png");

/// Nom du protocole embarque (`<nom>://localhost/...`, `http://<nom>.localhost/...` sous Windows).
pub const SCHEME: &str = "levelup-shell";

/// Ce que le protocole embarque repond a un chemin.
pub struct Embedded {
    pub content_type: &'static str,
    pub body: Vec<u8>,
}

/// Repond au chemin demande : un ecran, le logo, ou `None` (404).
pub fn respond(path: &str, locale: &str) -> Option<Embedded> {
    if path == "/logo.png" {
        return Some(Embedded {
            content_type: "image/png",
            body: LOGO.to_vec(),
        });
    }
    let screen = Screen::from_path(path)?;
    Some(Embedded {
        content_type: "text/html; charset=utf-8",
        body: render(screen, locale).into_bytes(),
    })
}

/// La page HTML de `screen` dans la langue `locale`.
pub fn render(screen: Screen, locale: &str) -> String {
    let message = catalogue::text(locale, screen.message_key()).unwrap_or_default();
    let kind = match screen {
        Screen::Starting | Screen::Stopping => "waiting",
        Screen::Stopped => "stopped",
        Screen::Failed(_) => "failed",
    };
    TEMPLATE
        .replace("{{lang}}", &escape(locale))
        .replace("{{kind}}", kind)
        .replace("{{message}}", &escape(message))
}

/// Echappe le texte pour un noeud HTML ou un attribut entre guillemets.
fn escape(text: &str) -> String {
    let mut out = String::with_capacity(text.len());
    for c in text.chars() {
        match c {
            '&' => out.push_str("&amp;"),
            '<' => out.push_str("&lt;"),
            '>' => out.push_str("&gt;"),
            '"' => out.push_str("&quot;"),
            '\'' => out.push_str("&#39;"),
            other => out.push(other),
        }
    }
    out
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::screen::FailureKind;

    #[test]
    fn chaque_ecran_se_rend_dans_chaque_langue_avec_son_libelle() {
        for code in catalogue::LOCALES {
            for screen in Screen::all() {
                let html = render(screen, code);
                let expected = escape(catalogue::text(code, screen.message_key()).unwrap());
                assert!(html.contains(&expected), "{screen:?}/{code} : libelle absent");
                assert!(html.contains(&format!("lang=\"{code}\"")));
                assert!(!html.contains("{{"), "{screen:?}/{code} : emplacement non rempli");
            }
        }
    }

    #[test]
    fn la_page_ne_charge_rien_hors_du_protocole_embarque() {
        let html = render(Screen::Starting, catalogue::DEFAULT_LOCALE);
        assert!(!html.contains("<script"));
        assert!(!html.contains("http://") && !html.contains("https://"));
    }

    #[test]
    fn les_ecrans_terminaux_ne_portent_pas_l_attente() {
        let locale = catalogue::DEFAULT_LOCALE;
        assert!(render(Screen::Starting, locale).contains("class=\"waiting\""));
        assert!(render(Screen::Stopping, locale).contains("class=\"waiting\""));
        assert!(render(Screen::Stopped, locale).contains("class=\"stopped\""));
        assert!(render(Screen::Failed(FailureKind::Timeout), locale).contains("class=\"failed\""));
    }

    #[test]
    fn le_texte_est_echappe() {
        assert_eq!(escape("a<b>&\"'"), "a&lt;b&gt;&amp;&quot;&#39;");
    }

    #[test]
    fn le_protocole_sert_les_ecrans_et_le_logo_et_refuse_le_reste() {
        let locale = catalogue::DEFAULT_LOCALE;
        let page = respond(&Screen::Stopping.path(), locale).expect("ecran");
        assert!(page.content_type.starts_with("text/html"));
        let logo = respond("/logo.png", locale).expect("logo");
        assert_eq!(logo.content_type, "image/png");
        assert!(logo.body.starts_with(&[0x89, b'P', b'N', b'G']));
        assert!(respond("/inconnu", locale).is_none());
        assert!(respond("/../Cargo.toml", locale).is_none());
    }
}
