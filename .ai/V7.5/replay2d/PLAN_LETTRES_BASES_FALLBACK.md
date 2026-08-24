# PLAN — Lettres A/B/C des bases (fallback par ordre canonique mesuré)

Date : 2026-08-24. Origine : backlog Notion REPLAY 2D item 2 (libellés A/B/C du HUD),
réfuté dans les données décodées ; arbitrage utilisateur du 24/08 : « en fallback, faute de
mieux, un ordre stable nous suffit à dire A, B ou C » — route « les deux en parallèle »
(ce lot + une RE Ghidra séparée qui cherche la règle vraie ; si elle contredit ce fallback
AVANT fusion, on corrige ici avant l'écran).

Branche : `wt/lettres-bases` (worktree dédié, base feat/v75 `b16ba17e5`). Exécution sous le
contrat du skill `plan-execution` (ordre strict, gates, statuts, zéro fix hors périmètre).
Ce plan est commité PAR le lot (premier commit).

## Objectif et critère de succès

Étiqueter A/B/C les zones des modes à BASES SIMULTANÉES (Strongholds, Total Control) dans
le rejeu 2D, par l'ordre canonique le plus plausible du moteur (les slots ti=13, stables
entre matchs d'une même carte), publié par match, rendu au canvas. Succès = les témoins
re-cuits portent l'ordre, la lettre s'affiche sur les zones (et NULLE PART ailleurs :
jamais KOTH, jamais CTF), les artefacts anciens dégradent sans lettre, gates verts.
Le verdict final (« nos lettres = celles du jeu ? ») appartient au gate Theater de
l'utilisateur — ce lot fournit les témoins et l'item de planche, il ne tranche pas.

## Décisions TRANCHÉES (ne pas rouvrir)

1. Source de l'ordre : l'ordre des SLOTS ti=13 tel que le chantier zones l'a mesuré
   (3 slots / 3 ids identiques sur les 2 matchs Bastion). Vérifier sur pièces d'abord
   (phase 0) : si l'ordre n'est PAS stable inter-match sur une même carte, STOP et CR —
   le fallback tombe, pas de bricolage de remplacement.
2. Publication : champ OPTIONNEL sur les données de zone servies (pas d'incrément de
   `SchemaVersion` — règle du dépôt : un champ optionnel n'incrémente pas, précédent daté
   du 24/08 dans le même dépôt). Nom proposé : `letterRank` (0/1/2), au niveau zone.
3. Rendu : lettre majuscule (A=0, B=1, C=2) à l'ancre de la zone, style des libellés de
   callouts (blanc cerné de noir, insensible au thème), UNIQUEMENT quand le document dit
   des zones simultanées. La règle « aucun texte » des calques zones/objectifs est LEVÉE
   pour ce seul glyphe : le garde-test qui interdit fillText/strokeText est amendé avec
   justification datée (décision utilisateur 2026-08-24, calibrage Theater à suivre) —
   il continue d'interdire tout AUTRE texte.
4. Pas de lettre dans les fiches ni le fil des morts dans ce lot (rendu canvas seulement).
5. Artefact ancien sans champ => aucune lettre, aucun avertissement (dégradation muette).

## Hors périmètre (fermé)

- KOTH (la colline n'a pas de lettre en jeu), CTF, Oddball.
- La preuve que nos lettres = celles du jeu (lot RE Ghidra séparé + gate Theater user).
- Toute re-cuisson de masse (bombe RAM consignée) ; le backfill des artefacts anciens.
- ReplayCanvas.tsx (cliquet 797) : le rendu vit dans le calque zones existant.

## Phase 0 — L'ordre, vérifié sur pièces

- [ ] 0.1 Retrouver dans `internal/analysis/replay/zone_states*.go` (et la construction
      slot->zone du lot C-bis) où vit l'index de slot ti=13 par zone, et vérifier qu'il
      est disponible au moment de la publication.
- [ ] 0.2 Mesurer la stabilité : sur les 2 matchs Bastion du corpus (et tout autre
      Strongholds/TC disponible dans `data/cache/film_chunks` du principal), l'ordre des
      slots donne-t-il la MÊME permutation zone->rang sur une même carte ? Instrument
      jetable gaté par env var si besoin (patron TI47_FILM).
- [ ] 0.3 Verdict de phase : ordre stable => on publie ; instable => STOP + CR.

Gate 0 : permutation identique sur les matchs d'une même carte, chiffres au CR.

## Phase 1 — Publication (champ optionnel, pas de bump)

- [ ] 1.1 `letterRank` optionnel au niveau zone dans la charge servie (là où les zones
      partent déjà au client — suivre l'existant de zoneStates/mapObjectives, pas de
      nouveau canal), omis quand l'ordre n'est pas établi.
- [ ] 1.2 Contrat OpenAPI + `make generate-types` (generated.ts) ; si tableau nullable,
      passer par la liste NULLABLE_ARRAYS existante.
- [ ] 1.3 Re-cuire les témoins nécessaires SEULEMENT (2-3 : les 2 Bastion + 1 Total
      Control s'il y en a un au cache), UN à la fois via `cmd/replay-build --facts`,
      anciens sauvegardés sous le motif `_backup_*` existant. La bombe RAM consignée a
      frappé un comp de mode-score sur un film CTF : si un re-bake dérape en mémoire,
      tuer, consigner, continuer avec les témoins qui passent.
- [ ] 1.4 Tests Go du package touché (table : zone avec rang, sans rang, mode non
      simultané => absent).

Gate 1 : témoin re-cuit porte `letterRank` cohérent avec la mesure 0.2 ; artefact schéma
antérieur servi tel quel => champ absent, aucun crash contrat.

## Phase 2 — Rendu web

- [ ] 2.1 Lettre dessinée à l'ancre de zone dans le calque zones (zoneStatesLayer /
      objectivesLayer selon qui possède l'ancre — suivre l'existant), style libellé
      callout, seulement si `letterRank` présent.
- [ ] 2.2 Amender le garde-test « ni fillText ni strokeText » : il autorise LE glyphe de
      lettre (une seule chaîne d'un caractère, A-C), avec le commentaire daté prescrit en
      décision 3 ; il continue d'échouer sur tout autre texte.
- [ ] 2.3 Tests vitest : lettre présente avec rang, absente sans rang, jamais en KOTH ;
      garde amendé testé dans les deux sens.
- [ ] 2.4 Item de planche pour le calibrage : préparer le texte de l'item (matchs témoins,
      ce que l'utilisateur doit comparer au Theater) et le livrer au CR — la republication
      de la planche appartient au superviseur.

Gate 2 (depuis apps/web du worktree, node_modules/.tmp purgé) : `npx tsc -b` 0 erreur ;
`npx vitest run src/features/match-replay` 0 échec ; `npx eslint src/features/match-replay`
0 erreur nouvelle.

## Garde-rails d'exécution

- Commandes `go` : UNE à la fois, GOCACHE privé (`<worktree>/.gocache`, jamais commité) ;
  `npm ci` dans apps/web du worktree ; vitest hors sandbox si besoin.
- Données du principal (`c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/`) :
  lecture seule, SAUF l'écriture des artefacts témoins re-cuits (JSON du cache replays,
  anciens sauvegardés) — aucune écriture DuckDB nulle part.
- Coordination : une autre session a des modifications non commitées sur `openapi.yaml` /
  `generated.ts` dans le principal — ne pas s'en préoccuper (fusion gérée par le
  superviseur, régénération au besoin) ; ce lot ne touche PAS `SchemaVersion`.
- Ne pas toucher `.ai/thought_log.md` ni `REGISTRE_REPORTS.md` du principal.

## Découvertes

(consigner ici — rien corriger)

## CR attendu

Statut par item, mesures 0.2, sorties des gates, liste des témoins re-cuits (avant/après),
texte de l'item de planche, commits `lettres(pN): ...` (jamais `git add -A`), aucun push.
