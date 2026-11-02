//! Gardes de TEXTE sur des fichiers que le test ne peut pas exercer : l'installeur NSIS ne tourne pas en
//! test, et le moteur de fenetres de Tauri (WebView2) n'existe pas sans runtime graphique. Chaque garde lit
//! le fichier source et exige la ligne qui porte le comportement ; elle ne prouve pas l'effet, elle tient
//! la regle qu'un autre test ou une verification manuelle a etablie (revue adversariale du lot L19).

use std::fs;
use std::path::Path;

/// Contenu du fichier relatif a `src-tauri`, fins de ligne normalisees.
fn lire(relatif: &str) -> String {
    let chemin = Path::new(env!("CARGO_MANIFEST_DIR")).join(relatif);
    fs::read_to_string(&chemin)
        .unwrap_or_else(|e| panic!("lecture de {}: {e}", chemin.display()))
        .replace("\r\n", "\n")
}

/// Texte compris entre `debut` (exclu) et la premiere occurrence de `fin` qui le suit.
fn entre<'a>(source: &'a str, debut: &str, fin: &str) -> &'a str {
    let apres = source
        .split_once(debut)
        .unwrap_or_else(|| panic!("marqueur absent: {debut}"))
        .1;
    apres
        .split_once(fin)
        .unwrap_or_else(|| panic!("marqueur de fin absent apres {debut}: {fin}"))
        .0
}

/// Lignes de code d'un extrait : sans les commentaires NSIS (`;`) ni Rust (`//`), sans les blancs.
fn lignes_de_code(extrait: &str) -> Vec<&str> {
    extrait
        .lines()
        .map(str::trim)
        .filter(|l| !l.is_empty() && !l.starts_with(';') && !l.starts_with("//"))
        .collect()
}

/// L'installeur interactif passe seul a la page finale quand l'extraction a reussi : sans
/// `SetAutoClose true` dans le crochet de fin d'installation, le modele de Tauri laisse la page
/// d'extraction affichee jusqu'a un clic.
#[test]
fn le_crochet_de_fin_d_installation_ferme_la_page_d_extraction() {
    let hooks = lire("windows/hooks.nsh");
    let corps = entre(&hooks, "!macro NSIS_HOOK_POSTINSTALL", "!macroend");
    assert!(
        lignes_de_code(corps).contains(&"SetAutoClose true"),
        "NSIS_HOOK_POSTINSTALL doit porter `SetAutoClose true` (hors commentaire):\n{corps}"
    );
}

/// Le script d'ouverture de liens du plugin opener appelle une commande IPC refusee a la page du serveur :
/// le clic sur `target="_blank"` etait avale. Il reste coupe.
#[test]
fn le_plugin_opener_n_intercepte_pas_les_clics_sur_les_liens() {
    let main = lire("src/main.rs");
    assert!(
        lignes_de_code(&main)
            .iter()
            .any(|l| l.starts_with(".open_js_links_on_click(false)")),
        "main.rs doit appeler `.open_js_links_on_click(false)` sur le plugin opener"
    );
}

/// L'application n'a qu'une fenetre : la closure `on_new_window` refuse toute nouvelle fenetre, que le lien
/// parte au navigateur ou non (la decision d'ouverture externe est prise avant, par `open_external`).
#[test]
fn la_closure_on_new_window_refuse_toute_nouvelle_fenetre() {
    let main = lire("src/main.rs");
    let closure = entre(&main, ".on_new_window(", ".on_download(");
    let code = lignes_de_code(closure);
    assert_eq!(
        code.last().copied(),
        Some("})"),
        "la closure on_new_window doit se terminer la ou le test l'attend:\n{closure}"
    );
    assert_eq!(
        code[code.len() - 2],
        "NewWindowResponse::Deny",
        "la closure on_new_window doit rendre `NewWindowResponse::Deny` en dernier:\n{closure}"
    );
    for autorise in ["NewWindowResponse::Allow", "NewWindowResponse::Create"] {
        assert!(
            !closure.contains(autorise),
            "la closure on_new_window ne doit pas rendre {autorise}"
        );
    }
}
