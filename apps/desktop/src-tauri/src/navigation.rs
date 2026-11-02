//! Politique de navigation de la fenetre.
//!
//! La fenetre ne montre que deux choses : ses ecrans embarques et le serveur local de ce
//! lancement (`http://127.0.0.1:<port>`). Tout lien vers le web s'ouvre dans le navigateur par
//! defaut, jamais dans la fenetre ; le reste (fichiers, scripts, schemas du systeme) est refuse.
//! La page du serveur n'a aucun acces aux commandes de la coquille (aucune capacite n'est
//! accordee), ce filtre garde en plus la fenetre sur le serveur.
//!
//! Une demande de NOUVELLE fenetre (`target="_blank"`, `window.open`) suit la meme politique,
//! mais la coquille n'ouvre jamais une seconde fenetre : voir [`decide_new_window`].

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

/// Sort d'une demande de nouvelle fenetre.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum NewWindowDecision {
    /// Le lien part dans le navigateur par defaut ; la fenetre refuse d'en ouvrir une seconde.
    OpenExternally,
    /// Aucune seconde fenetre, aucun lien suivi.
    Deny,
}

/// Decide d'une demande de nouvelle fenetre. Meme politique que [`decide`] : ce qui partirait dans
/// le navigateur y part ; ce que la fenetre suivrait (ecran embarque, serveur local) ou refuserait
/// est refuse, car l'application n'a qu'une fenetre.
pub fn decide_new_window(url: &Url, port: Option<u16>) -> NewWindowDecision {
    match decide(url, port) {
        Navigation::OpenExternally => NewWindowDecision::OpenExternally,
        Navigation::Allow | Navigation::Deny => NewWindowDecision::Deny,
    }
}

/// Origine seule d'une URL (`https://hote[:port]`), la seule part qui se journalise : une URL
/// externe peut porter un code ou un secret (lien de connexion par code, paires `?cle=`). Un schema
/// sans hote (`mailto:`, `javascript:`) ne rend que son schema.
pub fn origin_for_log(url: &Url) -> String {
    match url.host_str() {
        Some(host) => match url.port() {
            Some(port) => format!("{}://{host}:{port}", url.scheme()),
            None => format!("{}://{host}", url.scheme()),
        },
        None => format!("{}:", url.scheme()),
    }
}

/// Variable d'environnement qui remplace l'ouverture du navigateur par le seul journal : les essais
/// automatises de la coquille ne doivent pas ouvrir le navigateur du poste.
pub const OPEN_EXTERNAL_ENV: &str = "LEVELUP_SHELL_OPEN_EXTERNAL";

/// Ce que devient un lien destine au navigateur.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum ExternalOpen {
    /// Navigateur par defaut du systeme.
    System,
    /// Journal seul (`LEVELUP_SHELL_OPEN_EXTERNAL=log`), pour les essais.
    LogOnly,
}

/// Lit la valeur de [`OPEN_EXTERNAL_ENV`] : seule la valeur `log` change le comportement.
pub fn external_open_mode(value: Option<&str>) -> ExternalOpen {
    match value {
        Some("log") => ExternalOpen::LogOnly,
        _ => ExternalOpen::System,
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
    fn le_declencheur_de_mise_a_jour_est_un_chemin_du_protocole_embarque() {
        let port = Some(8123);
        // Sous Windows le protocole embarque s'adresse par `http://levelup-shell.localhost` ; ailleurs par
        // son schema. Dans les deux cas le chemin reste interne a la coquille.
        assert_eq!(
            decide_str("http://levelup-shell.localhost/check-update", port),
            Navigation::Allow
        );
        assert_eq!(
            decide_str("levelup-shell://localhost/check-update", port),
            Navigation::Allow
        );
        // Un autre hote, meme avec ce chemin, n'est pas la coquille : navigateur ou refus, jamais la fenetre.
        assert_eq!(
            decide_str("http://levelup-shell.example.test/check-update", port),
            Navigation::OpenExternally
        );
        assert_eq!(
            decide_str("https://levelup-shell.localhost/check-update", port),
            Navigation::OpenExternally
        );
        assert_eq!(
            decide_str("http://127.0.0.1:9999/check-update", port),
            Navigation::OpenExternally
        );
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

    fn new_window_str(url: &str, port: Option<u16>) -> NewWindowDecision {
        decide_new_window(&Url::parse(url).unwrap(), port)
    }

    #[test]
    fn une_nouvelle_fenetre_externe_part_dans_le_navigateur() {
        let port = Some(8123);
        assert_eq!(
            new_window_str("https://www.microsoft.com/link?otc=ABCD1234", port),
            NewWindowDecision::OpenExternally
        );
        assert_eq!(
            new_window_str("http://127.0.0.1:9999/", port),
            NewWindowDecision::OpenExternally
        );
        assert_eq!(
            new_window_str("mailto:a@b.example", port),
            NewWindowDecision::OpenExternally
        );
    }

    #[test]
    fn une_nouvelle_fenetre_locale_ou_embarquee_est_refusee() {
        let port = Some(8123);
        assert_eq!(new_window_str("http://127.0.0.1:8123/", port), NewWindowDecision::Deny);
        assert_eq!(
            new_window_str("http://levelup-shell.localhost/starting", port),
            NewWindowDecision::Deny
        );
        assert_eq!(
            new_window_str("levelup-shell://localhost/stopping", port),
            NewWindowDecision::Deny
        );
        assert_eq!(new_window_str("about:blank", port), NewWindowDecision::Deny);
        assert_eq!(
            new_window_str("blob:http://127.0.0.1:8123/3f2a", port),
            NewWindowDecision::Deny
        );
    }

    #[test]
    fn une_nouvelle_fenetre_de_script_ou_de_fichier_est_refusee() {
        let port = Some(8123);
        assert_eq!(new_window_str("javascript:alert(1)", port), NewWindowDecision::Deny);
        assert_eq!(new_window_str("data:text/html,<p>x</p>", port), NewWindowDecision::Deny);
        assert_eq!(
            new_window_str("file:///C:/Windows/win.ini", port),
            NewWindowDecision::Deny
        );
        assert_eq!(new_window_str("ms-settings:privacy", port), NewWindowDecision::Deny);
    }

    #[test]
    fn le_journal_ne_garde_que_l_origine() {
        let url = Url::parse("https://www.microsoft.com/link?otc=ABCD1234#frag").unwrap();
        assert_eq!(origin_for_log(&url), "https://www.microsoft.com");
        let url = Url::parse("http://127.0.0.1:9999/chemin?secret=x").unwrap();
        assert_eq!(origin_for_log(&url), "http://127.0.0.1:9999");
        let url = Url::parse("mailto:a@b.example?subject=x").unwrap();
        assert_eq!(origin_for_log(&url), "mailto:");
    }

    #[test]
    fn seule_la_valeur_log_remplace_le_navigateur() {
        assert_eq!(external_open_mode(Some("log")), ExternalOpen::LogOnly);
        assert_eq!(external_open_mode(Some("1")), ExternalOpen::System);
        assert_eq!(external_open_mode(Some("")), ExternalOpen::System);
        assert_eq!(external_open_mode(None), ExternalOpen::System);
    }
}
