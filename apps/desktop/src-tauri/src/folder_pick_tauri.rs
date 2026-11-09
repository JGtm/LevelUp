//! La commande `pick_folder` : la boite « Parcourir » du systeme, ouverte a la demande de la page du serveur.
//!
//! C'est la SEULE commande que la page peut appeler (ADR 0045). Elle n'est autorisee que par la capacite
//! `capabilities/pick-folder.json` : la fenetre `main`, origine distante `http://127.0.0.1:*`, permission
//! `allow-pick-folder` (generee par `build.rs` a partir de la liste des commandes de l'application). Sans cette
//! capacite, tauri refuse tout appel venu d'une origine distante.
//!
//! Aucune decision ici : la page envoie la valeur du champ, la coquille ouvre la boite et rend le chemin choisi
//! tel quel (chaine opaque), ou rien si l'utilisateur annule. Le serveur valide le dossier a l'enregistrement.

use levelup_desktop::folder_pick::initial_dir;

/// Ouvre la boite de choix de dossier, modale a la fenetre appelante. `initial` : la valeur courante du champ,
/// dont le plus proche dossier existant sert de point de depart. Rend `None` quand l'utilisateur annule.
///
/// `async` et `spawn_blocking` : la boite de `rfd` bloque son thread jusqu'au choix, et lire le handle de la
/// fenetre passe par le fil principal. Une commande synchrone s'executerait sur le fil principal et le figerait
/// (la fenetre modale n'aurait plus de proprietaire qui reponde).
#[tauri::command]
pub async fn pick_folder(window: tauri::Window, initial: Option<String>) -> Result<Option<String>, String> {
    let chosen = tauri::async_runtime::spawn_blocking(move || {
        let mut dialog = rfd::FileDialog::new().set_parent(&window);
        if let Some(dir) = initial
            .as_deref()
            .and_then(|raw| initial_dir(raw, |path| path.is_dir()))
        {
            dialog = dialog.set_directory(dir);
        }
        dialog.pick_folder()
    })
    .await
    .map_err(|e| format!("boite de choix de dossier : {e}"))?;
    chosen
        .map(|path| {
            path.into_os_string()
                .into_string()
                .map_err(|path| format!("chemin non Unicode : {path:?}"))
        })
        .transpose()
}
