// build.rs : trois taches avant la compilation de la coquille.
//
// 1. Catalogue de repli (ADR 0045 D-13, A4). Les ecrans que la coquille affiche sans l'aide du
//    serveur (demarrage, fermeture, serveur mort, echecs) tirent leurs libelles de
//    `messages/desktop.toml`, le fichier meme que le serveur sert a `GET /api/v1/desktop/strings`,
//    pour TOUTES les langues de `config/locales.toml`. Aucun libelle n'est ecrit dans le code ni
//    dans la page : une langue listee et incomplete refuse la compilation, comme le fait le
//    serveur (une langue est complete ou refusee).
// 2. Garde-fous de release : la cle publique de l'updater et le binaire serveur doivent etre
//    poses. En developpement, un serveur absent est remplace par un fichier vide (le lancement
//    echoue alors proprement, ecran d'arret) pour que `cargo build` et `cargo test` ne dependent
//    pas de la compilation du serveur Go.
// 3. tauri-build (ressources Windows, sidecar, schemas, permissions). La seule commande que la page du
//    serveur appelle, `pick_folder`, est declaree ici : tauri-build en tire la permission `allow-pick-folder`
//    que `capabilities/pick-folder.json` accorde a `http://127.0.0.1:*`.

use std::collections::BTreeMap;
use std::env;
use std::fmt::Write as _;
use std::fs;
use std::path::{Path, PathBuf};

/// Chemin du depot depuis `apps/desktop/src-tauri`.
const REPO_ROOT: &str = "../../..";
const LOCALES_TOML: &str = "config/locales.toml";
const MESSAGES_TOML: &str = "apps/go-api/internal/i18n/messages/desktop.toml";
/// Nom du sidecar declare dans `bundle.externalBin` (sans suffixe de cible).
const SIDECAR_STEM: &str = "binaries/levelup-server";

fn main() {
    let manifest = PathBuf::from(env::var("CARGO_MANIFEST_DIR").expect("CARGO_MANIFEST_DIR"));
    let repo = manifest.join(REPO_ROOT);
    generate_catalogue(&repo);
    release_guards(&manifest);
    let attributes =
        tauri_build::Attributes::new().app_manifest(tauri_build::AppManifest::new().commands(&["pick_folder"]));
    if let Err(error) = tauri_build::try_build(attributes) {
        panic!("tauri-build : {error:#}");
    }
}

/// Lit `locales.toml` et `desktop.toml`, verifie la completude et ecrit le catalogue Rust.
fn generate_catalogue(repo: &Path) {
    let locales_path = repo.join(LOCALES_TOML);
    let messages_path = repo.join(MESSAGES_TOML);
    println!("cargo:rerun-if-changed={}", locales_path.display());
    println!("cargo:rerun-if-changed={}", messages_path.display());

    let locales: toml::Table = read_toml(&locales_path);
    let default = locales
        .get("default")
        .and_then(toml::Value::as_str)
        .unwrap_or_else(|| panic!("{} : cle `default` absente", locales_path.display()))
        .to_string();
    let fallback = locales
        .get("fallback")
        .and_then(toml::Value::as_str)
        .unwrap_or(default.as_str())
        .to_string();
    let codes: Vec<String> = locales
        .get("locale")
        .and_then(toml::Value::as_array)
        .unwrap_or_else(|| panic!("{} : aucune entree [[locale]]", locales_path.display()))
        .iter()
        .map(|entry| {
            entry
                .get("code")
                .and_then(toml::Value::as_str)
                .unwrap_or_else(|| panic!("{} : une entree [[locale]] sans `code`", locales_path.display()))
                .to_string()
        })
        .collect();
    assert!(
        codes.contains(&default),
        "la langue par defaut `{default}` n'est pas une [[locale]]"
    );

    let messages: toml::Table = read_toml(&messages_path);
    let root = messages
        .get("desktop")
        .and_then(toml::Value::as_table)
        .unwrap_or_else(|| panic!("{} : table [desktop] absente", messages_path.display()));
    let mut keys: BTreeMap<String, &toml::Table> = BTreeMap::new();
    collect_messages(root, "", &mut keys);
    assert!(!keys.is_empty(), "{} : aucun message", messages_path.display());

    let mut out = String::from("// @generated par build.rs depuis desktop.toml et locales.toml : ne pas editer.\n\n");
    let _ = writeln!(out, "pub const DEFAULT_LOCALE: &str = {default:?};");
    let _ = writeln!(out, "pub const FALLBACK_LOCALE: &str = {fallback:?};");
    let _ = writeln!(out, "pub const LOCALES: &[&str] = &{codes:?};");
    let key_list: Vec<&String> = keys.keys().collect();
    let _ = writeln!(out, "pub const KEYS: &[&str] = &{key_list:?};\n");
    out.push_str("static ENTRIES: &[(&str, &[(&str, &str)])] = &[\n");
    for code in &codes {
        let _ = writeln!(out, "    ({code:?}, &[");
        for (key, table) in &keys {
            let text = table
                .get(code.as_str())
                .and_then(toml::Value::as_str)
                .map(str::trim)
                .filter(|t| !t.is_empty())
                .unwrap_or_else(|| {
                    panic!(
                        "{} : le message `{key}` n'a pas de texte pour la langue `{code}`",
                        messages_path.display()
                    )
                });
            let _ = writeln!(out, "        ({key:?}, {text:?}),");
        }
        out.push_str("    ]),\n");
    }
    out.push_str("];\n");

    let dest = PathBuf::from(env::var("OUT_DIR").expect("OUT_DIR")).join("catalogue_gen.rs");
    fs::write(&dest, out).unwrap_or_else(|e| panic!("ecriture de {} : {e}", dest.display()));
}

/// Aplatit les tables `[desktop.a.b]` en cles `a_b` (les identifiants que sert le serveur).
fn collect_messages<'a>(table: &'a toml::Table, prefix: &str, keys: &mut BTreeMap<String, &'a toml::Table>) {
    for (name, value) in table {
        let Some(child) = value.as_table() else { continue };
        let key = if prefix.is_empty() {
            name.clone()
        } else {
            format!("{prefix}_{name}")
        };
        // Une table feuille porte des textes (valeurs chaines) ; une table intermediaire des tables.
        if child.values().any(toml::Value::is_str) {
            keys.insert(key, child);
        } else {
            collect_messages(child, &key, keys);
        }
    }
}

fn read_toml(path: &Path) -> toml::Table {
    let raw = fs::read_to_string(path).unwrap_or_else(|e| panic!("lecture de {} : {e}", path.display()));
    raw.parse()
        .unwrap_or_else(|e| panic!("{} : TOML invalide : {e}", path.display()))
}

/// Refuse une release sans cle d'updater ni serveur ; pose un serveur factice hors release.
fn release_guards(manifest: &Path) {
    let conf_path = manifest.join("tauri.conf.json");
    println!("cargo:rerun-if-changed={}", conf_path.display());
    println!("cargo:rerun-if-env-changed=PROFILE");
    let release = env::var("PROFILE").is_ok_and(|p| p == "release");

    let conf: serde_json::Value = serde_json::from_str(
        &fs::read_to_string(&conf_path).unwrap_or_else(|e| panic!("lecture de {} : {e}", conf_path.display())),
    )
    .unwrap_or_else(|e| panic!("{} : JSON invalide : {e}", conf_path.display()));
    let pubkey = conf["plugins"]["updater"]["pubkey"].as_str().unwrap_or("").trim();
    if release && pubkey.is_empty() {
        panic!(
            "plugins.updater.pubkey est vide dans tauri.conf.json : une release signee exige la cle publique \
             de l'updater (ADR 0045 D-16, lot L17). Hors release, la coquille tourne sans updater."
        );
    }

    let target = env::var("TARGET").expect("TARGET");
    let sidecar = manifest.join(format!("{SIDECAR_STEM}-{target}.exe"));
    println!("cargo:rerun-if-changed={}", sidecar.display());
    let present = fs::metadata(&sidecar).map(|m| m.len() > 0).unwrap_or(false);
    if release && !present {
        panic!(
            "le serveur {} est absent ou vide : compiler apps/go-api/cmd/server et le copier a cet endroit \
             (scripts/desktop-dev.ps1 le fait en developpement ; la CI de release le fait au lot L17)",
            sidecar.display()
        );
    }
    if !sidecar.exists() {
        fs::create_dir_all(sidecar.parent().expect("dossier du sidecar")).expect("creation de binaries/");
        fs::write(&sidecar, b"").expect("ecriture du serveur factice");
        println!(
            "cargo:warning=serveur factice (fichier vide) pose a {} : lancer scripts/desktop-dev.ps1 pour le vrai",
            sidecar.display()
        );
    }
}
