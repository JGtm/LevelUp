# HANDOFF superviseur — exploitation du « Registre du film Theater » — 2026-08-18 (soir)

> Ecrit par la session de pilotage du plan `replay2d/PLAN_EXPLOITATION_REGISTRE_FILM.md` a
> l'approche de la fin de son contexte (demande utilisateur : « fais-toi un handoff pour survivre
> a l'auto-compact »). Point d'entree pour la session qui reprend le PILOTAGE de ce chantier :
> un executeur Opus par worktree, verification SUR PIECES de chaque CR avant relais, fusion par
> le superviseur avec gates rejoues, jamais de code ecrit par le superviseur hors docs.
> Lire aussi : le plan (§7 journal = source de verite, §8 orchestration), les journaux
> `replay2d/registre_film/LOT*.md`, `REGISTRE_REPORTS.md`, `.ai/thought_log.md` (entrees en tete).

## 0. Etat exact au moment du handoff

- Branche unique `feat/v75`, HEAD **`f41fd362c`** = `origin/feat/v75` (poussee). Le principal
  `C:\Users\Guillaume\Projects\LevelUp` est PARTAGE avec les sessions de l'utilisateur (item 4
  « objectifs vivants » : `wt/drapeau`, `wt/attache`, ...) : n'y faire QUE des fusions, quand
  `git status` est propre (2 fichiers non suivis d'autres sessions a laisser :
  `.ai/AUDIT_V7.2.0_MAIN_2026-08-06.md`, `apps/go-api/internal/himap/sonde_bouillie_gamefiles_test.go`).
- **CI de `feat/v75` : ROUGE sur un job** (« Go Coverage + Baseline ») : `TestReplayFacts*`
  (`internal/platform/duckdb`, tag `integration`) — la requete de production lit `map_id`
  (ajoute par l'item 4 en `b0fb3e10f`) et le DDL du test ne le porte pas. Correctif = R1-9 du
  lot C-bis phase 2b (ci-dessous), livre a la prochaine fusion. Tous les autres jobs sont verts.
- Un executeur de fond travaille dans `C:\Users\Guillaume\Projects\LevelUp-wt-zones-ti13-p2b`
  (branche `wt/zones-ti13-p2b`) — voir §2. Les autres worktrees de ce chantier (`wt-registre-film`,
  `wt-score-*`, `wt-visee-*`, `wt-zones-film`, `wt-zones-ti13`, `wt-zones-ti13-p2`,
  `wt-joueur-moteur`, `wt-usages-film`) sont FUSIONNES : a supprimer a la cloture (`git worktree
  remove`, jamais avant que la CI de la fusion soit verte).

## 1. Ce qui est LIVRE sur `feat/v75` (tout verifie sur pieces, tout journalise)

| Fusion | Contenu | Journal |
|---|---|---|
| `104f468c6` (18/08 ~11h10) | tout le plan hors C-bis : lot 0 (hooks nommes, empreinte du registre, i26/i27), lot A phases 1+2 (**score dans le temps tous modes** : schema 12 `scoreTimeline` + `coverage.score/originResolved`, en-tete/fiches vivants, courbe Dominance), lot E phases 1+2 (**elevation de visee** `Point.p` schema 13, cone x max(0,35, cos p) + tick), lot F (sondes), lots C/D/B/P clos `[!]` (negatifs ecrits) ; correctif ratchet cross-feature (`lib/replay/scoreTimeline.ts`, `bb6eb7694`) | `registre_film/LOT0.md`, `LOTA_PHASE0/1/2.md`, `LOTE_PHASE12.md`, `LOTC_*.md`, `LOTD_PHASE0.md`, `LOTBP_PHASE0.md`, `LOTEF_PHASE0.md` |
| `9b959de28` | **lot C-bis phases 0+1** : ti=13 (`managed-object-property` i1, `player-masked-property` i2..i33) RESOLU et PORTE (`components_managed_property.go`, 33 lignes de table ECS, DesyncAt +8..+128 sur les 12 films) ; tag 3 = jauge de capture (97,2 %/94,8 % vs hasard 56,7-60,8 %), tag 4 = premier canal enumerable, tag 5 = cle de nommage des zones (3 slots/3 identifiants identiques sur les 2 Strongholds), KOTH une seule zone active ; gate 1 NON atteint a la lettre -> arbitrage | `LOTCBIS_PHASE0.md`, `LOTCBIS_PHASE1.md`, `LOTCBIS_ARBITRAGE_PHASE1.md` |
| `2753e8ee2` | **lot C-bis phase 2a** (mesure dans `replay`, instrument sous garde) — GATE 2a TENU : slot ti=13 -> zone du catalogue coherent 93,1 %/98,4 % (temoins 41-57 %), table stable entre les 2 Strongholds ; **tag 4 = PROPRIETAIRE** (0xFFFFFFFF personne / 0 / 1 ; 100 %/91 % vs equipe du capteur ; un slot proprietaire par zone) ; KOTH periodes 91/84/82 % ; un slot ti=13 = une propriete reseau NOMMEE (i0 lu) | `LOTCBIS_PHASE2A.md`, arbitrage §Phase 2a |
| (item 4, autre session) `75794c10a` schema 14 `flagCarries`, `1ee29328d`/`f41fd362c` schema **15** (corrections flagCarries + fix `attachFlagCarries` 19-22 Go) | a connaitre : c'est pourquoi NOTRE `zoneStates` prend le **schema 16** | `PLAN_DRAPEAU_OBJET.md` |

Hors depot : planche de validation utilisateur (artefact `cb1f0981-0a80-4611-9190-2c633a542535`,
14 points R1-R6/V1-V4/C1-C3/D7 ; les 4 temoins sont poses dans `data/cache/replays/halo_infinite/`
du principal, ancien `000d5950.json` sauvegarde sous `_backup_gate_registre_film/`) ; Notion
« Backlog LevelUp » : section « Rejeu 2D — exploitation du film Theater » (livre / pret a
exploiter / ferme / a savoir).

## 2. EN COURS — lot C-bis phase 2b : `zoneStates` (publication + rendu)

Worktree `../LevelUp-wt-zones-ti13-p2b`, branche `wt/zones-ti13-p2b` (base `32ee72107`).

- **Livraison** (`480bafef7` assemblage `replay/zone_states.go` + `document_zones.go` +
  `build_zones.go` + `replaybuild/zones.go` ; `4dcbbebb9` contrat 35 -> 36 + `generated.ts` ;
  `038a78436` web `zoneStatesLayer.ts` + `useZoneStates.ts` + `ReplayCanvas.tsx` — zones teintees
  par proprietaire, colline active, arc de progression ; `dd5c5364e` mesure) : **captures ->
  intervalle du capteur 57/59 = 96,6 % et 64/66 = 97,0 %** (Bastion `7344d24f`, `696a9d7c`, seuil
  90 %), **KOTH `01e1f945` 4 975/5 343 frames = 93,1 %** (seuil 80 %), +0,22 % de l'artefact ;
  le controle se lit SUR LA FORME PUBLIEE. Journal : `LOTCBIS_PHASE2B.md`.
- **Revue adversariale (1 relecteur, contrat + L3/L5/L6)** : 2 P1 — repli colline branche sur
  `len(pairs)==0` au lieu du mode (CTF publierait des collines) ; `objectivesLayer.ts` retranchait
  `originMs` une 2e fois depuis le lot A (pulses decales) ; 5 P2 — election du proprietaire sans
  seuil ni unicite, rampes non localisees non comptees, periodes actives recouvrantes possibles,
  hook non memoise, garde catalogue cote client absente ; L3 — code mort dans `zone_state_scan.go`
  (`Names`, `dominantName`, 2e `SetProbeHook`), instruments 2a redondants.
- **Ronde 1 de corrections** : R1-1..R1-6 COMMITES (`5165d8b00`, `83fca8fe4`, `9c9a530c3`,
  `8e1b5b903`, `b0a296ec6`, `7843433d0` = HEAD) ; R1-7 (garde client) EN COURS non commite (4
  fichiers web modifies) ; RESTENT R1-8 (code mort scanner), R1-9 (DDL `map_id` du test des
  faits -> repare la CI), la FUSION de `origin/feat/v75` (`f41fd362c`) dans la branche avec
  **re-numerotation : `SchemaVersion` 15 -> 16** (chronique du 15 = celle de l'item 4 ;
  `wantReplayDocumentFields` = leur compte + 1 ; `generated.ts`, `openapi.yaml`, goldens
  regeneres), les gates (Go build/vet/tests analysis no-CGO/replaybuild+contracttest CGO/
  integration TestReplayFacts/lint ; web tsc --force, vitest match-replay + lib, lint 19 =
  baseline, cross-feature <= 7), la re-cuisson des 3 temoins par `cmd/replay-build --facts`
  (censee etre DEBLOQUEE par le fix de l'item 4 ; sinon l'instrument du journal, en le disant),
  et le CR. Executeur en cours (relance apres deux coupures API 529 le 18/08 soir) ; s'il est
  mort : `git -C ../LevelUp-wt-zones-ti13-p2b log --oneline -12` + `git status`, puis relancer un
  agent frais avec ce brief en 7 points (un commit par point, aucun push).
- **Ensuite (superviseur)** : ronde 2 = relecteur FRAIS sur les seules corrections R1-1..R1-9
  (P0+P1 doit decroitre strictement, sinon escalade) ; fusion `--no-ff` de `wt/zones-ti13-p2b`
  dans `feat/v75` depuis le principal propre ; `git push origin feat/v75` ; `gh run list --branch
  feat/v75` VERTE AU NIVEAU JOB ; SHA a l'utilisateur ; section « zones » ajoutee a la planche
  (temoins Strongholds `7344d24f` + KOTH `01e1f945` a poser dans le cache du principal) ; docs
  (journal du plan §7, registre, thought_log, `LOTCBIS_PHASE2B.md` §revue).

## 3. Apres la phase 2b

- **Gate visuel utilisateur** (planche `cb1f0981`) sur son `make dev` : a ses verdicts, retouches
  en worktree `wt/<retouche>` depuis `feat/v75`, puis fusion. Decisions ouvertes dans la planche :
  D7 (arrondi de `p` : 0,1 deg / 0,5 / entier), jauge de capture (`radial-progress` largement
  couverte par `zoneStates.progress`), temps mort par trajectoires (mini-lot), ouvrier sans faits.
- **Hygiene de cloture** (P2 des revues, worktree depuis `feat/v75`, avant le tag v7.5.0) :
  `RealRounds` (`objectiveevents/statborg.go` ~450-471 : manche du trou + adjacence + test) ;
  `Coverage.OriginResolved` -> `*bool` omitempty (contrat regenere) ; `components_hooks_test.go`
  600 L a scinder ; lignes 177/182/201 du registre mal formees ; 4 leviers
  `absPerIndexAxisW`/`SetAbsPerIndexAxisW`/`SetAbsDequantMode`/`AbsIndexHistogram` + item 0.2 ;
  instruments de la phase 2a redondants avec le scanner de production ; catalogue de zones :
  `instance_id` = 0 partout et aucun role « colline » en KOTH (plan item 4 phase 2) ; ouvrier
  distant sans faits AVANT toute activation.
- **Coordination avec l'item 4** (session de l'utilisateur) : il ne touche ni `objectives.go`
  ni `objectivesLayer.ts` ; ses champs = `flagCarries` (`document_objectives_live.go`), i10
  `object-parent-state` sur `wt/attache` ; les numeros de schema se prennent DANS L'ORDRE DES
  FUSIONS sur `feat/v75` (leur 15 est passe avant notre 16 : toujours verifier `SchemaVersion`
  sur `origin/feat/v75` avant de fusionner).

## 4. Regles de pilotage (rappel, toutes apprises sur pieces)

- Un executeur par worktree ; jamais le principal ; jamais `git add -A` ; jamais `--no-verify` ;
  jamais `main` (push = deploy) ; `feat/v75` = CI seulement, verte au niveau JOB avant de dire
  « clos ». Conflits de docs (thought_log, registre) : garder les deux cotes, en octets bruts (Bash).
- Machine (D17) : un film par processus `go test`, avant-plan, plafond 3 Go surveille, jamais de
  boucle sur le corpus (film `1b1e380f` = bombe RAM), une commande `go` a la fois par worktree
  avec son `GOCACHE`, `CGO_ENABLED=0` pour filmdec/replay/objectiveevents, CGO pour le lint et
  les paquets DuckDB, « parallel golangci-lint is running » = reessayer a 60 s, jamais de
  `go test` coupe par un pipe tronque (test.exe orphelin).
- Revue adversariale : contrat de 6 lignes, relecteur frais qui ne sait pas qui a ecrit,
  recevabilite fichier:ligne + declencheur + consequence, 2 rondes max, temoins contre le NIVEAU
  DU HASARD (un temoin sous « la moitie du reel » est inatteignable sur un canal bavard).
- Ne JAMAIS relayer un CR sans l'avoir verifie sur pieces (rouvrir le fichier, rejouer un gate).
