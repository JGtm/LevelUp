// Package diagnostic produit le rapport de diagnostic de l'application de bureau (lot L22, ADR 0045) : une
// archive zip que l'utilisateur joint à un signalement.
//
// Contenu : les journaux (la fin de chaque fichier `.log`, bornée), `app_info.json` (version, profil, racines,
// capacités, mémoire), `settings.json` (les réglages sans secret), `databases/<clé>.txt` (le diagnostic de schéma
// de chaque base, lu par la porte de lecture du serveur), `scheduled_tasks.json`, `updater.json`.
//
// RIEN DE SECRET : jamais un jeton, un secret ni un code ne sort. Deux barrières, appliquées à TOUT ce qui entre
// dans l'archive : les clés interdites ([ForbiddenKey]) sont retirées des documents structurés, et le texte des
// journaux passe par [Scrub] (formes connues de jetons, paires clé=valeur sensibles, valeurs exactes des secrets
// du processus). Le test sentinelle de ce paquet injecte un faux jeton partout et le cherche dans l'archive.
package diagnostic
