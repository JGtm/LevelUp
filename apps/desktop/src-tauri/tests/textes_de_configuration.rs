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

/// Position de la premiere ligne de `code` qui commence par `debut`.
fn position(code: &[&str], debut: &str) -> Option<usize> {
    code.iter().position(|l| l.starts_with(debut))
}

/// L22c.1 : la desinstallation retire l'entree de demarrage et la tache planifiee par le serveur livre. Le
/// crochet NSIS doit appeler le mode `-mode uninstall-cleanup` du serveur installe, hors flux de mise a jour
/// (l'utilisateur qui met a jour garde son entree), sans jamais supprimer de dossier de donnees. Revue V7 :
/// l'appel est A L'INTERIEUR du bloc `${If} $UpdateMode <> 1` (ordre des lignes), pas seulement dans le crochet.
#[test]
fn le_crochet_de_desinstallation_appelle_le_nettoyage_du_serveur() {
    let hooks = lire("windows/hooks.nsh");
    let corps = entre(&hooks, "!macro NSIS_HOOK_PREUNINSTALL", "!macroend");
    let code = lignes_de_code(corps);
    let appel = position(&code, "nsExec::ExecToLog").unwrap_or_else(|| panic!("aucun nsExec::ExecToLog:\n{corps}"));
    assert!(
        code[appel].contains(r#""$INSTDIR\levelup-server.exe" -mode uninstall-cleanup"#),
        "NSIS_HOOK_PREUNINSTALL doit lancer `$INSTDIR/levelup-server.exe -mode uninstall-cleanup` par nsExec::ExecToLog:\n{corps}"
    );
    let garde = code
        .iter()
        .position(|l| *l == "${If} $UpdateMode <> 1")
        .unwrap_or_else(|| panic!("le nettoyage doit etre garde par `${{If}} $UpdateMode <> 1`:\n{corps}"));
    let fin_garde = code
        .iter()
        .rposition(|l| *l == "${EndIf}")
        .expect("le bloc garde par $UpdateMode doit se refermer");
    assert!(
        garde < appel && appel < fin_garde,
        "nsExec doit etre DANS le bloc `${{If}} $UpdateMode <> 1` (lignes {garde} < {appel} < {fin_garde}):\n{corps}"
    );
    assert!(
        code[appel].contains("/TIMEOUT="),
        "l'appel doit etre borne par /TIMEOUT : un nettoyage pendu bloquerait la desinstallation:\n{corps}"
    );
    for ligne in &code {
        assert!(
            !ligne.starts_with("RMDir") && !ligne.starts_with("Delete"),
            "le crochet de desinstallation ne supprime rien lui-meme (jamais la racine de donnees): {ligne}"
        );
    }
    assert!(
        corps.contains(r"$LOCALAPPDATA\${PRODUCTNAME}"),
        "a defaut de LEVELUP_DATA_DIR, la racine de donnees est %LOCALAPPDATA%/LevelUp, comme celle de la coquille"
    );
}

/// Revue V5 : la branche « LEVELUP_DATA_DIR absente » du desinstalleur n'est jouee par aucun essai (l'essai de bout
/// en bout pose la variable, et la jouer viserait la vraie racine de donnees du poste). Le texte tient sa forme : le
/// crochet pose la variable par `SetEnvironmentVariable` sur la racine par defaut, et ce SEULEMENT quand elle est vide.
#[test]
fn le_crochet_pose_la_racine_par_defaut_quand_la_variable_est_absente() {
    let hooks = lire("windows/hooks.nsh");
    let code = lignes_de_code(entre(&hooks, "!macro NSIS_HOOK_PREUNINSTALL", "!macroend"));
    let lecture = position(&code, "ReadEnvStr $0 LEVELUP_DATA_DIR").expect("lecture de LEVELUP_DATA_DIR");
    let test = code
        .iter()
        .position(|l| *l == r#"${If} $0 == """#)
        .expect("condition `variable vide`");
    let pose = position(&code, "System::Call 'Kernel32::SetEnvironmentVariable(t, t)i(")
        .expect("pose de la variable par SetEnvironmentVariable");
    assert!(
        code[pose].contains(r#"("LEVELUP_DATA_DIR", "$LOCALAPPDATA\${PRODUCTNAME}").r0'"#),
        "la racine par defaut posee est %LOCALAPPDATA%\\LevelUp (celle de la coquille) : {}",
        code[pose]
    );
    let appel = position(&code, "nsExec::ExecToLog").unwrap();
    assert!(
        lecture < test && test < pose && pose < appel && code[pose + 1] == "${EndIf}",
        "ordre attendu : lecture, condition `vide`, pose, fin de condition, puis l'appel du mode"
    );
}

/// Revue V2 : le crochet tourne avant la verification de l'application en cours du modele de Tauri ; « Annuler » a
/// « LevelUp est en cours d'execution » ne doit rien retirer. Le crochet fait donc la verification EN TETE, avant le
/// nettoyage du serveur livre.
#[test]
fn le_crochet_verifie_que_l_application_ne_tourne_pas_avant_de_nettoyer() {
    let hooks = lire("windows/hooks.nsh");
    let code = lignes_de_code(entre(&hooks, "!macro NSIS_HOOK_PREUNINSTALL", "!macroend"));
    assert_eq!(
        code.first().copied(),
        Some(r#"!insertmacro CheckIfAppIsRunning "$INSTDIR\${MAINBINARYNAME}.exe" "${PRODUCTNAME}""#),
        "le crochet commence par la verification de l'application en cours"
    );
    let verif = position(&code, "!insertmacro CheckIfAppIsRunning").unwrap();
    let appel = position(&code, "nsExec::ExecToLog").unwrap();
    assert!(verif < appel, "CheckIfAppIsRunning doit preceder nsExec");
}

/// Revue V1 : apres le crochet, le modele de Tauri supprime la valeur Run du produit sans regarder ce qu'elle lance.
/// Le crochet de fin la REECRIT quand elle ne visait pas le dossier desinstalle : la valeur est lue AVANT le nettoyage
/// (le serveur retire la sienne), gardee dans une variable, comparee au debut de `$INSTDIR\` apres retrait du
/// guillemet initial, et reposee sinon.
#[test]
fn le_crochet_rend_l_entree_de_demarrage_d_une_autre_copie() {
    let hooks = lire("windows/hooks.nsh");
    let avant = lignes_de_code(entre(&hooks, "!macro NSIS_HOOK_PREUNINSTALL", "!macroend"));
    let apres_texte = entre(&hooks, "!macro NSIS_HOOK_POSTUNINSTALL", "!macroend");
    let apres = lignes_de_code(apres_texte);
    let cle = r#"HKCU "Software\Microsoft\Windows\CurrentVersion\Run" "${PRODUCTNAME}""#;
    assert!(
        hooks.contains("Var LevelUpRunValue"),
        "la valeur lue est gardee dans une variable globale"
    );
    let lecture = position(&avant, "ReadRegStr $LevelUpRunValue").expect("lecture de la valeur Run avant le nettoyage");
    assert!(
        avant[lecture].ends_with(cle),
        "la valeur lue est Run\\${{PRODUCTNAME}} : {}",
        avant[lecture]
    );
    assert!(
        lecture < position(&avant, "nsExec::ExecToLog").unwrap(),
        "la valeur Run est lue AVANT que le serveur ne retire la sienne"
    );
    assert!(
        apres.contains(&"${If} $UpdateMode <> 1"),
        "la reecriture suit le modele : hors mise a jour seulement:\n{apres_texte}"
    );
    let retrait = apres
        .iter()
        .position(|l| l.starts_with(r#"StrCpy $0 $LevelUpRunValue "" 1"#))
        .expect("retrait du guillemet initial");
    assert!(
        apres.contains(&r#"StrLen $1 "$INSTDIR\""#),
        "longueur de `$INSTDIR` plus l'antislash:\n{apres_texte}"
    );
    let prefixe = position(&apres, "StrCpy $2 $0 $1").expect("debut de la commande, a la longueur du dossier");
    let comparaison = apres
        .iter()
        .position(|l| *l == r#"${If} $2 != "$INSTDIR\""#)
        .expect("comparaison au dossier desinstalle");
    let ecriture = position(&apres, "WriteRegStr").expect("reecriture de la valeur");
    assert!(
        apres[ecriture].ends_with(&format!("{cle} $LevelUpRunValue")),
        "la valeur reecrite est celle qui a ete lue : {}",
        apres[ecriture]
    );
    assert!(
        retrait < prefixe && prefixe < comparaison && comparaison < ecriture && apres[ecriture + 1] == "${EndIf}",
        "ordre attendu : retrait du guillemet, debut de chaine, comparaison, ecriture DANS la condition:\n{apres_texte}"
    );
}

/// L22c.1 : le nom de l'entree de demarrage que la coquille pose (`autostart::ENTRY_NAME`) et celui que le
/// serveur retire (`startupentry.EntryName`) sont la meme chaine : deux copies, un seul nom.
#[test]
fn le_serveur_retire_l_entree_de_demarrage_sous_le_nom_que_la_coquille_pose() {
    let go = lire("../../go-api/internal/platform/startupentry/startupentry.go");
    let nom = levelup_desktop::autostart::ENTRY_NAME;
    assert!(
        go.contains(&format!("const EntryName = \"{nom}\"")),
        "startupentry.EntryName doit valoir \"{nom}\", le nom pose par autostart::ENTRY_NAME"
    );
}
