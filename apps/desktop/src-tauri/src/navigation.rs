//! Politique de navigation de la fenetre.
//!
//! La fenetre ne montre que deux choses : ses ecrans embarques et le serveur local de ce
//! lancement (`http://127.0.0.1:<port>`). Tout lien vers le web s'ouvre dans le navigateur par
//! defaut, jamais dans la fenetre ; le reste (fichiers, scripts, schemas du systeme) est refuse.
//! La page du serveur n'a aucun acces aux commandes de la coquille (aucune capacite n'est
//! accordee), ce filtre garde en plus la fenetre sur le serveur.

use url::Url;

use crate::render::SCHEME;

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Navigation {
    /// La fenetre suit le lien.
    Allow,
    /// Le lien part dans le navigateur par defaut, la fenetre reste ou elle est.
    OpenExternally,
    /// Le lien est ignore.
    Deny,
}

/// Decide du sort d'une navigation ; `port` est le port du serveur de ce lancement, s'il est connu.
pub fn decide(url: &Url, port: Option<u16>) -> Navigation {
    match url.scheme() {
        // Protocole embarque tel que tauri l'adresse hors Windows.
        s if s == SCHEME => Navigation::Allow,
        "about" => Navigation::Allow,
        "http" | "https" => decide_web(url, port),
        // Un blob est cree par la page : il est permis s'il vient de notre serveur.
        "blob" => match Url::parse(url.path()) {
            Ok(inner) if is_local_server(&inner, port) => Navigation::Allow,
            _ => Navigation::Deny,
        },
        "mailto" => Navigation::OpenExternally,
        _ => Navigation::Deny,
    }
}

fn decide_web(url: &Url, port: Option<u16>) -> Navigation {
    if is_embedded_host(url) || is_local_server(url, port) {
        Navigation::Allow
    } else {
        Navigation::OpenExternally
    }
}

/// Windows adresse un protocole embarque par `http://<nom>.localhost/`.
fn is_embedded_host(url: &Url) -> bool {
    url.scheme() == "http" && url.host_str() == Some(&format!("{SCHEME}.localhost"))
}

fn is_local_server(url: &Url, port: Option<u16>) -> bool {
    url.scheme() == "http" && url.host_str() == Some("127.0.0.1") && port.is_some() && url.port() == port
}

#[cfg(test)]
mod tests {
    use super::*;

    fn decide_str(url: &str, port: Option<u16>) -> Navigation {
        decide(&Url::parse(url).unwrap(), port)
    }

    #[test]
    fn la_fenetre_suit_ses_ecrans_et_son_serveur() {
        let port = Some(8123);
        assert_eq!(
            decide_str("levelup-shell://localhost/stopping", port),
            Navigation::Allow
        );
        assert_eq!(
            decide_str("http://levelup-shell.localhost/starting", None),
            Navigation::Allow
        );
        assert_eq!(decide_str("http://127.0.0.1:8123/", port), Navigation::Allow);
        assert_eq!(
            decide_str("http://127.0.0.1:8123/desktop/enter?secret=x", port),
            Navigation::Allow
        );
        assert_eq!(decide_str("about:blank", None), Navigation::Allow);
        assert_eq!(decide_str("blob:http://127.0.0.1:8123/3f2a", port), Navigation::Allow);
    }

    #[test]
    fn le_web_part_dans_le_navigateur() {
        let port = Some(8123);
        assert_eq!(
            decide_str("https://www.halowaypoint.com/", port),
            Navigation::OpenExternally
        );
        assert_eq!(decide_str("http://127.0.0.1:9999/", port), Navigation::OpenExternally);
        assert_eq!(decide_str("http://localhost:8123/", port), Navigation::OpenExternally);
        assert_eq!(decide_str("mailto:a@b.example", port), Navigation::OpenExternally);
    }

    #[test]
    fn sans_port_connu_aucun_serveur_local_n_est_suivi() {
        assert_eq!(decide_str("http://127.0.0.1:8123/", None), Navigation::OpenExternally);
    }

    #[test]
    fn le_reste_est_refuse() {
        let port = Some(8123);
        assert_eq!(decide_str("file:///C:/Windows/win.ini", port), Navigation::Deny);
        assert_eq!(decide_str("javascript:alert(1)", port), Navigation::Deny);
        assert_eq!(decide_str("data:text/html,<p>x</p>", port), Navigation::Deny);
        assert_eq!(decide_str("ms-settings:privacy", port), Navigation::Deny);
        assert_eq!(decide_str("blob:https://exemple.test/uuid", port), Navigation::Deny);
    }
}
