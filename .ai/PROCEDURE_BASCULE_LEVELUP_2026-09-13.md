# Procédure de bascule vers le dossier `LevelUp` (tâche Notion 10, item F.4)

> Écrite le 2026-09-13, à exécuter sur le signal de l'utilisateur, après fermeture des autres
> sessions ouvertes dans `LevelUp-go-migration` (17 sessions au 2026-09-13). Durée estimée : 15 à
> 30 minutes, dont l'essentiel en déplacement de fichiers sur le même disque (renommage, pas de
> copie). Retour arrière possible à chaque étape jusqu'à l'étape 7.

## État de départ (vérifié le 2026-09-13)

| Dossier | Rôle git | HEAD | Contenu non suivi |
|---|---|---|---|
| `Scripts/LevelUp` | dépôt principal (`.git` réel) | détaché sur `9b00560e6` (19 juillet) | aucun |
| `Scripts/LevelUp-go-migration` | worktree lié, branche `feat/v75` | à jour, poussé | `data/` (bases DuckDB, caches de films, artefacts de rejeu), `.env.local`, `app_settings.json`, `apps/web/node_modules`, `.claude/settings*.json`, `.claude/hooks/`, binaires `.exe` de diagnostic sous `apps/go-api/` |

Les autres worktrees `LevelUp-wt-*` restants n'ont pas de données (ils lisent celles de
`LevelUp-go-migration` par chemin absolu quand un agent en a besoin).

## Pré-requis, dans l'ordre

1. Toutes les sessions Claude ouvertes dans `LevelUp-go-migration` sont fermées (leurs serveurs
   MCP `python`/`node` aussi : `Get-Process python,node` ne doit plus lister de processus dont la
   ligne de commande cite `LevelUp-go-migration`).
2. Le serveur de dev est arrêté (port 8000 libre) et le serveur Vite aussi (`node vite.js`).
3. `git -C LevelUp-go-migration status --porcelain` ne montre que `data/...tmp/` et des fichiers
   ignorés ; `feat/v75` est poussé (`git rev-list --count origin/feat/v75..feat/v75` = 0).
4. Une sauvegarde récente des bases existe (clé PNY ou `data/backups/`), datée du jour.

## Étapes

1. **Détacher `LevelUp-go-migration` de `feat/v75`** (sinon `LevelUp` ne peut pas l'extraire) :
   `git -C LevelUp-go-migration checkout --detach`.
2. **Extraire `feat/v75` dans `LevelUp`** : `git -C LevelUp checkout feat/v75` puis
   `git -C LevelUp status --porcelain` vide.
3. **Déplacer les données et réglages non suivis** (renommage sur le même disque, instantané) :
   `robocopy LevelUp-go-migration\data LevelUp\data /E /MOVE /R:1 /W:1`, puis `.env.local`,
   `app_settings.json`, `.claude\settings.json`, `.claude\settings.local.json`,
   `.claude\hooks\`, `apps\web\node_modules` (même commande `robocopy /E /MOVE`). Vérifier
   après chaque déplacement que la source est vide et la cible complète (`Get-ChildItem`
   comptés des deux côtés avant/après). Ne PAS déplacer `data\...duckdb.tmp\` ni les `.exe`
   de diagnostic (à supprimer).
4. **Vérifier depuis `LevelUp`** : `go build ./...` dans `apps/go-api`, démarrage du serveur
   (`air` détaché, `/health` 200), page rejeu ouverte sur un match, `npm run typecheck` dans
   `apps/web`. Si un chemin absolu vers `LevelUp-go-migration` subsiste dans un réglage
   (`grep -rn "LevelUp-go-migration" .env.local app_settings.json .claude/`), le corriger.
5. **Repointer les jonctions** des worktrees `LevelUp-wt-*` restants (`apps\web\node_modules`)
   vers `LevelUp\apps\web\node_modules` : `cmd /c rmdir <lien>` puis `cmd /c mklink /J`.
6. **Mémoire et documents** : mettre à jour `.ai/project_map.md` et la mémoire agent
   (dossier principal = `LevelUp`, plus de « worktree principal partagé »).
7. **Supprimer le worktree `LevelUp-go-migration`** une fois tout vérifié :
   `git -C LevelUp worktree remove LevelUp-go-migration` (après `cmd /c rmdir` de sa jonction
   `node_modules` si elle en a une — au 2026-09-13 c'est un vrai dossier, déplacé à l'étape 3),
   puis `git worktree prune`.

## Retour arrière

Jusqu'à l'étape 6 incluse : `robocopy /E /MOVE` dans l'autre sens et `git -C LevelUp checkout
--detach 9b00560e6` puis `git -C LevelUp-go-migration checkout feat/v75`. Après l'étape 7 :
re-créer le worktree (`git worktree add LevelUp-go-migration feat/v75`) et redéplacer les
données ; rien n'est perdu tant que `data/` n'a pas été supprimé.

## Ce que cette procédure ne fait pas

- Elle ne touche ni à `main`, ni au VPS, ni à la copie des bases (tâche Notion suivante).
- Elle ne supprime pas les worktrees `LevelUp-wt-*` non fusionnés (`film-residus`,
  `section3-chunk00`, `ti11-cadre`) ni les 4 branches WIP du lot F.
