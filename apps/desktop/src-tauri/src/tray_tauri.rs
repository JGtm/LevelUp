//! L'icone de la zone de notification (lot L21.c) : sa creation et son menu sur tauri. Les decisions (quand
//! elle existe, ce que fait chaque entree) sont dans `levelup_desktop::shell_settings` ; ici ne reste que ce
//! qui touche tauri.
//!
//! L'icone n'existe que tant que le reglage `close_to_tray` est actif : pas d'icone surprise. Les libelles du
//! menu viennent du catalogue compile (le fichier meme que le serveur sert a `/desktop/strings`), dans la langue
//! que le serveur annonce.

use levelup_desktop::shell_settings::{MENU_OPEN, MENU_QUIT, MENU_SYNC, command_for_menu_id, tray_labels};
use tauri::AppHandle;
use tauri::image::Image;
use tauri::menu::{Menu, MenuItem};
use tauri::tray::{MouseButton, MouseButtonState, TrayIconBuilder, TrayIconEvent};

use crate::focus_main;
use crate::shell_tauri::on_tray_command;

/// Identifiant de l'icone : une seule dans l'application.
const TRAY_ID: &str = "levelup-tray";
/// Le nom du produit, pas un libelle de langue.
const TRAY_TOOLTIP: &str = "LevelUp";
/// L'icone de l'application (le meme fichier que `bundle.icon`).
const TRAY_ICON_PNG: &[u8] = include_bytes!("../icons/32x32.png");

/// Cree l'icone et son menu (Ouvrir, Synchroniser maintenant, Quitter) dans `locale`.
pub fn create(app: &AppHandle, locale: &str) -> Result<(), String> {
    let labels = tray_labels(locale);
    let open = MenuItem::with_id(app, MENU_OPEN, labels.open, true, None::<&str>).map_err(|e| e.to_string())?;
    let sync = MenuItem::with_id(app, MENU_SYNC, labels.sync, true, None::<&str>).map_err(|e| e.to_string())?;
    let quit = MenuItem::with_id(app, MENU_QUIT, labels.quit, true, None::<&str>).map_err(|e| e.to_string())?;
    let menu = Menu::with_items(app, &[&open, &sync, &quit]).map_err(|e| e.to_string())?;
    let icon = Image::from_bytes(TRAY_ICON_PNG).map_err(|e| e.to_string())?;
    TrayIconBuilder::with_id(TRAY_ID)
        .icon(icon)
        .tooltip(TRAY_TOOLTIP)
        .menu(&menu)
        // Le clic gauche rouvre la fenetre ; le menu s'ouvre au clic droit.
        .show_menu_on_left_click(false)
        .on_menu_event(|app, event| {
            if let Some(command) = command_for_menu_id(event.id.as_ref()) {
                on_tray_command(app, command);
            }
        })
        .on_tray_icon_event(|tray, event| {
            if let TrayIconEvent::Click {
                button: MouseButton::Left,
                button_state: MouseButtonState::Up,
                ..
            } = event
            {
                focus_main(tray.app_handle());
            }
        })
        .build(app)
        .map(|_| ())
        .map_err(|e| e.to_string())
}

/// Retire l'icone ; rend vrai si elle existait.
pub fn remove(app: &AppHandle) -> bool {
    app.remove_tray_by_id(TRAY_ID).is_some()
}
