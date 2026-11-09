# Annexes — enquête sur pièces du 2026-09-23 (retours rejeu 2D, Tactique, tiroir des assets)

Plan qui s'appuie sur ces annexes : `../PLAN_RETOURS_REJEU_2026-09-23.md`.

## Contenu

| Fichier | Point utilisateur |
|---|---|
| `RAPPORT_tirs_vehicules.md` | Ghost sans tir, sans son, sans éclair ; table de vérité de toutes les armes de véhicule |
| `RAPPORT_positions_limbe.md` | Mongoose qui part seul et sort de la carte ; Madina qui « vole » à sa réapparition |
| `RAPPORT_fiche_armes.md` | Fiche de XxDaemonGamerxX sans armes à 2:08 |
| `RAPPORT_equipes_b1ad85eb.md` | Eagle à 3, puis 5 ; les deux équipes à 5 |
| `RAPPORT_ctf_ab526724.md` | Drapeau, score et frise absents ; véhicules de décor sur Starboard |
| `RAPPORT_web_tactique_assets_sprint.md` | Page Tactique qui recharge tout ; doublons « Solution » ; « Sprint » sur les fiches |
| `BRIEF_COMMUN_ENQUETEURS.md` | Règles données aux six enquêteurs (lecture seule, verrou machine, format) |
| `instruments/<lot>/` | Scripts Node de mesure (`*.mjs`) et sondes Go `research` (`*_research_test.go`) |
| `instruments/superviseur/` | Scripts de premier balayage du superviseur |

## Lire les chemins des rapports

Les rapports citent `SP/...` : c'est le scratchpad de la session d'enquête
(`%LOCALAPPDATA%\Temp\claude\...\5e1d6f3f-...\scratchpad`), éphémère. Correspondance :

- `SP/sondes/<lot>/*.mjs|*.go` → `instruments/<lot>/` (scripts copiés ici ; leurs SORTIES `*.txt`,
  `*.tsv`, `*.json` ne sont pas copiées — elles se régénèrent en relançant le script) ;
- `SP/db/*.duckdb` → copies de `data/titles/halo_infinite/warehouse/` du 2026-09-23 14:11 (à
  refaire : copier les bases serveur arrêté, ne jamais ouvrir les bases vivantes) ;
- `SP/diag_q.exe` → `go build ./cmd/diag_q` (lecture seule, sortie TSV) ;
- `SP/voie.sh` → verrou « une commande go à la fois » utilisé pendant l'enquête.

Les sondes Go (`*_research_test.go`) ont tourné dans des worktrees détachés sur `43a01721e`,
retirés depuis ; elles lisent les faits persistés `data/cache/film_facts/halo_infinite/<id8>.filmfacts.bin`
(aucun film). Pour les rejouer : les copier dans
`apps/go-api/internal/games/halo_infinite/film/replay/` d'un worktree de lot, puis
`go test -tags research -count=1 -run <Test> ./internal/games/halo_infinite/film/replay/`.
