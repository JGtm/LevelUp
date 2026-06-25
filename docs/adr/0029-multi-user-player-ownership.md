# ADR 0029 — Contrôle d'accès multi-utilisateur : propriété des joueurs + participation aux ressources

> **Note de renumérotation** : ce document a porté le numéro **0024** à sa création.
> Renuméroté **0029** pour lever la collision avec `0024-lusr-v2-trueskill2-with-counts`. Les références
> code « ADR 0024 » relatives à l'ownership / `RequirePlayerOwnership` / « Couche A/B » désignent
> ce document (0029) ; celles relatives à LUSR/skill restent l'ADR 0024.

> Statut : Accepté (2026-06-01)
> Branche d'implémentation : `fix/sync-combat-completion-persist`
> Remplace le comportement implicite « tout joueur configuré est accessible à quiconque est connecté »

## Contexte

L'app est destinée au multi-utilisateur (cf. ADR 0009). Or, jusqu'ici, **aucun lien
n'existait entre l'utilisateur authentifié et le joueur dont il consulte les données** :

1. `/bootstrap` renvoyait **tous** les profils de `db_profiles.json` dans `available_players`,
   sans filtrage par propriétaire.
2. La closure `ServiceRegistry.resolve` ouvrait la base de **n'importe quel** slug présent
   dans `db_profiles.json`, sans vérifier la propriété.
3. `RequireAuth` ne vérifie que « est connecté », jamais « a le droit d'accéder à CE joueur ».

Conséquence : changer le slug dans l'URL (`/players/{slug}/...`) bascule vers les données d'un
autre joueur — acceptable en mono-utilisateur, **fuite de données** en multi-utilisateur strict.

Second problème, distinct : à l'intérieur de son propre slug, ouvrir un match auquel on n'a pas
participé affichait une page « mal renseignée » (stats personnelles vides, scoreboard d'autrui),
et une session inexistante renvoyait une page vide trompeuse au lieu d'une erreur.

## Décision

### Mécanisme de propriété = correspondance XUID

Le pont existe déjà : `userstore` (`data/auth/users.json`) associe à chaque utilisateur un
`XUID` (via `LinkIdentity` / `CreateFromXbox`), et chaque profil de `db_profiles.json` a un `xuid`.

**Un utilisateur possède le profil joueur dont le `xuid` == son xuid lié.** Pas de nouveau champ
`owner` dans `db_profiles.json` (évite une double source de vérité à synchroniser).

Règle d'autorisation (package pur `internal/authz`) :

| Cas | Décision |
|---|---|
| Mode `demo` ou `auth_mode` ∉ {password, xbox} | Accès ouvert (non concerné) |
| `role = admin` | Accès à tous les profils |
| `role = user`, `user.xuid == profile.xuid` | Accès au profil |
| `role = user`, `user.xuid != profile.xuid` | **403 `player_forbidden`** |
| Utilisateur non authentifié / non lié (xuid vide) | Ne possède rien → refus |

L'utilisateur courant est résolu depuis la session : `sess.Username` → `userstore.Get`
(mode password), ou `sess.LinkedHaloIdentity.XUID` → `userstore.GetByXUID` (mode xbox).

### Deux couches d'enforcement

- **Couche A — Propriété du joueur.** Un seul chokepoint : le middleware
  `RequirePlayerOwnership` monté sur le groupe de routes `/players/{player_slug}`. Il mappe
  slug → xuid via `cfg.LoadPlayers` (sans ouvrir de DB), vérifie la propriété, et renvoie
  **403 `player_forbidden`** avant d'atteindre le handler. Toute future route player-scoped
  DOIT être montée sous ce groupe.
  En complément UX, `bootstrap_service.Build` filtre `available_players` aux profils possédés
  (le menu L1 ne liste que les joueurs de l'utilisateur).

- **Couche B — Participation à la ressource.** Dans son propre slug :
  - Match non-participé → `MatchViewService.GetMatchView` renvoie `match_not_participant`
    (404) après vérification `IsParticipant` (EXISTS sur `match_participants`), **avant** les
    ~20 chargements parallèles (fail-fast + perf).
  - Session demandée explicitement mais introuvable → `SessionPageService.GetPage` renvoie
    `session_not_found` (404) au lieu d'une page vide 200.

Le frontend mappe ces codes (`ApiError.code`) vers une page « Indisponible » avec boutons
(Accueil / Mes matchs), sans redirection automatique.

### Choix de codes HTTP

- **403** (et non 404) pour un slug étranger : choix explicite. Accepte la fuite mineure
  « ce joueur existe mais ne t'appartient pas » au profit d'un message clair. Un slug
  *inconnu* reste 404 `player_not_found` (le middleware laisse passer, le handler répond).
- **404** pour match non-participé / session introuvable : dans l'espace d'URL scopé joueur,
  la ressource « n'existe pas pour toi ».

## Conséquences

- Sécurité : la fuite cross-utilisateur par édition d'URL est fermée par un point unique testable.
- Un utilisateur password sans xuid lié ne possède aucun joueur → invité à lier son compte Halo
  (UX frontend). Les admins (dont le premier compte auto-admin) voient tout.
- Contextes background (post-sync, backfill) : la closure `resolve` reste non gardée ; ces flux
  n'ont pas de session HTTP et ne transitent pas par le middleware — comportement inchangé.
- Squad V2 (`resolveByGT`, coéquipiers par gamertag) n'est pas gardé : c'est de la donnée de
  match partagée consultée depuis la page du joueur requérant (déjà autorisé). Hors scope.

## Extensions différées

- **Alts multiples par utilisateur** : `User.XUID` est singulier (1 user ↔ 1 xuid). Le support
  de plusieurs gamertags par humain passera par `User.XUIDs []string` + adaptation de
  `authz.CanAccessPlayer`. Non nécessaire au scope initial.
