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

/// L21.d : l'entree de demarrage avec Windows ne s'ecrit QUE par `autostart::reconcile` (qui lit d'abord et
/// ne touche a rien quand le reglage est coupe). Aucun autre fichier ne cree ni ne supprime l'entree : un
/// `.enable()` dans `main.rs` ou ailleurs ferait un demarrage impose, que la decision de l'utilisateur
/// (reglage coupe par defaut) interdit. Le plugin lance l'application avec `--minimized`.
#[test]
fn l_entree_de_demarrage_n_est_ecrite_que_par_le_reglage() {
    for fichier in [
        "src/main.rs",
        "src/shell_tauri.rs",
        "src/tray_tauri.rs",
        "src/toast_tauri.rs",
        "src/update_setup.rs",
        "src/update_tauri.rs",
    ] {
        let source = lire(fichier);
        let code = lignes_de_code(&source).concat();
        assert!(
            !code.contains(".enable()") && !code.contains(".disable()"),
            "{fichier} ne doit pas ecrire l'entree de demarrage : seul `autostart_tauri.rs`, derriere `reconcile`, le fait"
        );
    }
    let main = lire("src/main.rs");
    let init = entre(&main, "tauri_plugin_autostart::Builder::new()", ".build()");
    assert!(
        init.contains("MINIMIZED_ARG"),
        "le plugin d'autostart doit lancer l'application avec l'argument `--minimized` : {init}"
    );
    // Le plugin de memoire de fenetre ne restaure pas la visibilite : il remontrerait au lancement la
    // fenetre qu'un demarrage reduit vient de cacher.
    let shell_source = lire("src/shell_tauri.rs");
    let code = [lignes_de_code(&main).concat(), lignes_de_code(&shell_source).concat()].concat();
    assert!(
        code.contains("!StateFlags::VISIBLE"),
        "le plugin window-state doit exclure StateFlags::VISIBLE"
    );
    // Restaurer l'etat maximise d'une fenetre cachee l'affiche (SW_MAXIMIZE) : en demarrage reduit il est
    // exclu au lancement (decision `launch::restores_maximized_at_start`) puis restaure a la premiere montre.
    assert!(
        code.contains("!StateFlags::MAXIMIZED") && code.contains("launch::restores_maximized_at_start(initial)"),
        "le plugin window-state ne doit pas restaurer MAXIMIZED d'une fenetre cachee au lancement"
    );
    assert!(
        code.contains("StateFlags::MAXIMIZED)")
            && lignes_de_code(&main)
                .concat()
                .contains("restore_maximized_if_pending(app, &window)"),
        "l'etat maximise est restaure a la premiere montre apres un demarrage cache"
    );
    let shell = lire("src/shell_tauri.rs");
    assert!(
        shell.contains("autostart::reconcile(wanted,"),
        "shell_tauri.rs doit appliquer le reglage par `autostart::reconcile`"
    );
}
