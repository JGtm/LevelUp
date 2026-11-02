//! Coquille LevelUp : tout ce qui se decide sans fenetre (ADR 0045, D-1 a D-6 et D-13 a D-18).
//!
//! La coquille ne calcule rien : elle lance le serveur Go (le sidecar), lit ses evenements, montre
//! le bon ecran et ferme proprement. Cette bibliotheque n'importe pas tauri ; le binaire
//! (`main.rs`) la relie a la fenetre. C'est ce qui permet de tester le contrat contre un faux
//! sidecar, sans ecran.
//!
//! - [`events`]    : les lignes JSON du serveur (D-2).
//! - [`lifecycle`] : la machine d'etats pure, evenements en entree, actions en sortie.
//! - [`supervisor`]: la boucle qui relie la machine au processus, aux minuteries et a l'hote.
//! - [`sidecar`]   : lancement du serveur, environnement, tubes (D-1, A14).
//! - [`screen`], [`catalogue`], [`render`] : les ecrans embarques et leurs libelles (A4).
//! - [`updater`]   : la mise a jour proposee, jamais imposee, installee apres l'arret du serveur (D-16).
//! - [`update_manual`] : la meme verification, demandee par l'utilisateur depuis « A propos ».
//! - `updater_decision` : l'arret propre qui autorise l'installation et les messages d'apres-arret (re-exportes par [`updater`]).
//! - [`roots`], [`secret`], [`logfile`], [`linereader`], [`navigation`] : petits outils.

pub mod catalogue;
pub mod events;
pub mod lifecycle;
pub mod linereader;
pub mod logfile;
pub mod navigation;
pub mod render;
pub mod roots;
pub mod screen;
pub mod secret;
pub mod sidecar;
pub mod supervisor;
pub mod update_manual;
pub mod updater;
mod updater_decision;
