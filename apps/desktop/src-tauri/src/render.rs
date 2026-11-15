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

/// Chemin que la page du serveur appelle pour demander la verification de mise a jour : la page n'a
/// qu'une commande de la coquille, la boite « Parcourir » (ADR 0045 D-31) ; pour le reste, ce chemin du
/// protocole embarque est son seul canal.
pub const CHECK_UPDATE_PATH: &str = "/check-update";

/// Ce que le protocole embarque fait d'un chemin.
pub enum Route {
    /// Un ecran ou le logo : la reponse est servie telle quelle.
    Page(Embedded),
    /// La demande de verification de mise a jour : la coquille la lance, la reponse est vide (204).
    CheckUpdate,
    /// Tout autre chemin : 404.
    NotFound,
}

/// Aiguille un chemin du protocole embarque. Le chemin est exact : ni suffixe, ni autre casse.
pub fn route(path: &str, locale: &str) -> Route {
    if path == CHECK_UPDATE_PATH {
        return Route::CheckUpdate;
    }
    match respond(path, locale) {
        Some(page) => Route::Page(page),
        None => Route::NotFound,
    }
}

/// La reponse HTTP du protocole embarque.
pub struct Reply {
    pub status: u16,
    pub content_type: &'static str,
    pub body: Vec<u8>,
}

/// Repond a une requete du protocole embarque. `on_check_update` n'est appele que pour la demande de
/// verification de mise a jour, et la reponse (204) ne l'attend pas : c'est a lui de rendre la main tout de suite.
pub fn reply(path: &str, locale: &str, on_check_update: impl FnOnce()) -> Reply {
    const TEXT: &str = "text/plain; charset=utf-8";
    match route(path, locale) {
        Route::Page(page) => Reply {
            status: 200,
            content_type: page.content_type,
            body: page.body,
        },
        Route::CheckUpdate => {
            on_check_update();
            Reply {
                status: 204,
                content_type: TEXT,
                body: Vec::new(),
            }
        }
        Route::NotFound => Reply {
            status: 404,
            content_type: TEXT,
            body: Vec::new(),
        },
    }
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
    fn seul_le_chemin_exact_declenche_la_verification_de_mise_a_jour() {
        let locale = catalogue::DEFAULT_LOCALE;
        assert!(matches!(route("/check-update", locale), Route::CheckUpdate));
        for other in [
            "/check-update/",
            "/check-update/x",
            "/Check-Update",
            "/check_update",
            "/check-updates",
            "check-update",
            "/",
            "",
            "/update",
        ] {
            assert!(
                matches!(route(other, locale), Route::NotFound),
                "{other:?} ne doit rien declencher"
            );
        }
        assert!(matches!(route("/stopping", locale), Route::Page(_)));
        assert!(matches!(route("/logo.png", locale), Route::Page(_)));
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

    #[test]
    fn la_demande_de_verification_est_lancee_et_repond_204_sans_l_attendre() {
        let mut launched = 0;
        let reply = reply("/check-update", catalogue::DEFAULT_LOCALE, || launched += 1);
        assert_eq!(launched, 1);
        assert_eq!(reply.status, 204);
        assert!(reply.body.is_empty());
    }

    #[test]
    fn les_autres_chemins_ne_lancent_aucune_verification() {
        for (path, status) in [
            ("/stopping", 200),
            ("/logo.png", 200),
            ("/inconnu", 404),
            ("/check-update/", 404),
        ] {
            let mut launched = 0;
            let reply = reply(path, catalogue::DEFAULT_LOCALE, || launched += 1);
            assert_eq!(launched, 0, "{path} ne doit rien lancer");
            assert_eq!(reply.status, status, "{path}");
        }
    }
}
