# PLAN — Retours utilisateur rejeu 2D + match view (2026-09-02)

> Instruction des 7 retours du 2026-09-02. Quatre enquêtes de code parallèles, constats
> vérifiés sur pièces (fichiers:lignes ci-dessous). Exécution sous le contrat du skill
> `plan-execution`. Branche : `feat/v75` (mode branche unique — lots = commits).
> **Aucun item n'est lancé avant les décisions D1-D4 tranchées par l'utilisateur.**

## Objectif et critère de succès

Chaque retour est soit corrigé (gate vert), soit statué « comportement correct » avec
l'explication, soit cadré en chantier séparé avec sa condition d'ouverture. Aucun retour
sans statut à la clôture.

---

## Constats (analyse)

### P1 + P4 — Joueurs « en attente de respawn » éternels / quit-rejoin-bots — RACINE COMMUNE

Dans l'artefact de rejeu, une vie (`Track`) n'existe pour le front que si elle porte un
`xuid`, et ce xuid ne peut venir QUE d'une mort du fil des morts qui termine la vie à
±150 ms (`analysis/replay/lives.go:41,162-195`). Tout ce qui n'est pas clos par une mort
(partant, bot, survivant de fin de partie, slot recyclé) = vie anonyme = invisible sur la
carte (`rosterLogic.ts:93` `if (!track.xuid) continue` ; `replayMarkers.ts:226-227`),
pendant que la fiche reste au roster et affiche « Éliminé / Réapparition ? » sans fin
(`rosterLogic.ts:409-445` — DEUX états seulement, aucun « hors film » ;
`ReplayVitality.tsx:124-129`).

Mécanismes précis, par plausibilité :
- **H1 (très haute)** : partant = dernière vie jamais nommée + plus aucune vie ensuite →
  `respawnFrame = -1` → « Réapparition ? » permanent. Cas documenté dans
  `document.go:805-808` (15 vies / 105 non nommées sur le film de référence, dont 6
  survivants de fin de partie).
- **H3 (haute — le cas Sylvanus)** : slot de bipède recyclé (partant → arrivant/bot) :
  `decimateTracks` (`build.go:783-856`) produit UNE track par slot sans découpe par trou
  (alors que `buildLifeSpans` découpe à `lifeGapUS = 5 s`), `ownersFromLives`
  (`lives.go:229-232`) donne le slot au PREMIER occupant, `nameTracks` nomme PAR SLOT
  (`identity.go:28-37`). Le second joueur n'a aucune vie ; le premier « vit » à sa place.
  Compteur de preuve : `coverage.bridge.slotCollisions` (`coverage.go:296-298`) — publié
  dans l'artefact, jamais lu par le front.
- **H4 (haute)** : bots sans XUID → jamais au fil des morts → jamais au roster
  (`player_index.go:99-110` `rosterFromDeaths`). Le décodeur BOT_METADATA existe
  (`games/halo_infinite/film/killsource/botmeta.go` — slot, botID, nom, « le roster peut
  se remplir en cours de film ») mais n'est PAS branché sur `internal/analysis/replay`.
- **H5 (moyenne-haute)** : rejoin → index divergent : xuid retiré (`player_index.go:85-93`)
  ou table entière vidée (`injectiveOrEmpty`, `player_index.go:117-133`) → rejeu
  partiellement/totalement anonyme.

Le film porte par ailleurs le compteur de réapparition RÉEL (`player-respawn-timer`,
décodé puis jeté — `filmdec/traverse.go:438-440`), dette déjà notée au
`PLAN_FINALISATION_REJEU_2D.md` §6.3.

### P2 — Translocateur : usage sans effet UI

L'effet de fiche EXISTE et est câblé (`playerCardFx.ts:302-306` translocation flash,
`ReplayTeams.tsx:106-109,246-248,450-455`, CSS `globals.css:457-478`). Il ne s'allume
jamais parce que son seul déclencheur est l'heuristique spatiale `riftTeleports`
(`placementTeleport.ts:200-219`) : rendement mesuré 4 détections / 39 films. Or le
pipeline publie depuis le schéma 26 le signal exact d'usage :
`equipmentChanges[].kind === 'spent'` (`filmdec/equipment_changes.go:185-200`), consommé
par ZÉRO effet (seul `changeRefine.ts` s'en sert pour éteindre la vignette ; décision
« le spent est muet » écrite dans `equipmentChangeSound.ts:11-19` — le translocateur n'y
figure même pas). Second défaut : `RANG_TRANSLOCATEUR = 11` en dur
(`placementTeleport.ts:55`), valable famille A seulement ; le web ne lit jamais
`doc.abilityLabels` (les films famille B, rangs 19-22, sont structurellement muets).

### P3 — Effets de fiche capture de base / prise de drapeau

Aucune branche non fusionnée ne porte ces FX (vérifié : 5 branches en avance sur
`feat/v75`, la seule touchant le rejeu web est `wt/ctf-zone-retour` = zone de retour sur
canvas, pas un effet de fiche). État réel :
- `flag_grabs` publié par le Go (`objectiveevents/named.go:91`) : ZÉRO consommateur dans
  `features/match-replay`.
- `flag_captures` : effet existant mais canvas-only (`flagCaptureFx.ts`, câblé sous la
  double porte de `useReplayFlagCarries.ts:184-187`) — rien sur la fiche.
- Capture de zone : branchée sur la fiche mais comme FILIGRANE STATIQUE
  (`ReplayObjectiveMark.tsx`, aucun keyframe CSS de capture dans `globals.css`).
- `CardFxInput` (`playerCardFx.ts:139-161`) n'a AUCUN canal d'événement d'objectif —
  c'est le chaînon manquant commun.

### P5 — « Duels & confrontations » vide (graphe antagoniste + némésis/souffre-douleur)

Régression datée (bascule `39da43fbf` des lecteurs sur `match_kill_events_latest`,
2026-08-03) : la requête Q20 sert les xuid NULL des bots tels quels
(`queries_match.go:415-427`, doctrine assumée), mais le scan Go les reçoit dans des
`string` nus (`match_view_repo_extras.go:204-213`, `domain.KVPairRaw` sans
`sql.NullString`) → le scan échoue à la PREMIÈRE ligne de bot, `goLoad` avale l'erreur en
WARN best-effort (`match_view_data_loaders.go:257-264`) → `kvPairs == nil` → seule la
section Duels est vide. Le lecteur frère `match_view_repo_assist_pairs.go:157` fait déjà
le `sql.NullString` correct. Mesuré sur base réelle : **245 matchs Infinite cassés par ce
bug (récupérables), 583 sans donnée source (films expirés — vide LÉGITIME)** ; total
42,3 % des matchs Infinite avec section vide. Interdit de normaliser NULL en '' (test
`TestPasDeXuidNormaliseEnChaineVide` — fusionnerait tous les bots).

### P6 — Export vidéo 30 fps

**Déjà le cas.** `EXPORT_FPS = 30` (`replayVideoEncoder.ts:83`), horodatage posé (1 image
= 1/30 s de match, quelle que soit la machine), plan d'échantillonnage à 30
(`replayExportPlan.ts:127`). L'enregistrement temps réel demande aussi 30
(`replayRecording.ts:41`). Aucune action. (Si l'attente était 60 fps : ~×2 taille et
durée d'encodage, faisable, le plomberie `fps` existe déjà de bout en bout.)

### P7 — Frises dominance/score identiques en Assassin

Le masquage EXISTE déjà : `scoreTrack` rend `null` quand `scoreMirrorsFrags`
(`useReplayTimeline.ts:199` → `replayTimelineTracksLogic.ts:348-360`). Il rate parce que
la garde exige l'égalité STRICTE score final == frags comptés pour TOUTES les équipes ;
un seul kill au camp non résolu (`useReplayTimeline.ts:231`), un suicide ou un kill
environnemental suffit à réafficher le doublon. Doctrine du fichier : tri sur la DONNÉE,
jamais sur le libellé de mode (le mode n'est d'ailleurs pas connu du lecteur).

---

## Décisions — TRANCHÉES le 2026-09-02 (user)

- **D1 (P7)** : TRANCHÉ — masquer la frise dominance quand les deux suites de segments
  rendus sont identiques.
- **D2 (P5)** : TRANCHÉ — kills joueur↔joueur récupérés, les bots ne seront JAMAIS
  affichés dans les duels (pas de phase 2 identité bot).
- **D3 (P3)** : RÉSOLU le 02/09 (collègue, vérifié sur pièces) — le design C2 est DÉJÀ
  LIVRÉ sur feat/v75 : commit `ec0c81928` (30/08, « la marque de l'objectif ») +
  extensions `d50b14b38`/`db30ad7c4` (genre bomb, schéma 30). La marque ne s'allume que
  si l'artefact porte flagCarries (≥15) / vipCrown (22) / skullCarries (23) /
  bombCarries (30) / objectives de zone, et seulement sur les modes à objectif. Le
  non-affichage observé = artefacts cuits avant ces schémas, ou mode Slayer. **Lot 4
  reformulé** : rien à coder — vérification visuelle après re-cuisson d'un film témoin
  CTF/Strongholds récent.
- **D4 (P1/P4)** : TRANCHÉ — lot 5 MAINTENANT. Le compteur `player-respawn-timer`
  (§6.3) est inclus dans le même schéma pour ne cuire qu'une fois (item 5.4 ACTIF).
  Re-cuisson : toujours soumise à accord explicite, un film témoin d'abord.
- **D5 (P6)** : TRANCHÉ — on reste à 30 fps, point clos `[x]`.

---

## Lots (ordre = risque/effort croissant)

### Lot 1 — Duels & confrontations : réparer le scan NULL (Go) — RAPIDE

- [ ] 1.1 `domain.KVPairRaw` : `KillerXUID`/`VictimXUID` passent en `*string` (ou
  `sql.NullString` au scan, patron de `match_view_repo_assist_pairs.go:157`). JAMAIS de
  COALESCE en '' (doctrine `kill_events_source.go:35-42`).
- [ ] 1.2 `GetMatchKVPairs` (`match_view_repo_extras.go:183-217`) : scan adapté ;
  l'erreur de requête des lignes 189/197 cesse d'être avalée sans trace — `slog.WarnContext`
  avec l'err avant tout `return nil, nil`.
- [ ] 1.3 Consommateurs (`buildKillerVictimPairs`, `buildNemesisMap`) : les lignes à xuid
  nil sont ignorées explicitement (D2 phase 1) — comportement documenté par commentaire.
- [ ] 1.4 Test : cas de scan avec ligne bot (xuid NULL) + lignes joueurs → les paires
  joueurs survivent.
- [ ] 1.5 UI : différencier le message « aucune donnée de duel (film expiré) » de l'état
  d'erreur — uniquement si trivial (sinon consigner en Découvertes).

Gate : `cd apps/go-api && go test ./internal/platform/duckdb/... ./internal/service/...`
+ vérification manuelle sur un des 245 matchs cassés (ex. `52fc79ef`) : section remplie.

### Lot 2 — Frise dominance masquée quand identique (web) — RAPIDE

- [ ] 2.1 Selon D1 : comparaison des suites de segments dominance vs score dans
  `useReplayTimeline.scoreTrack` (ou tolérance dans `scoreMirrorsFrags`).
- [ ] 2.2 Réviser le test verrou `replayTimelineTracksLogic.test.ts:316-319` + cas
  nouveau (segments identiques → piste masquée ; segments différents → affichée).

Gate : `make check-types && make test-web`.

### Lot 3 — Effet translocateur branché sur l'usage réel (web) — MOYEN

- [ ] 3.1 Résolution du rang translocateur via `doc.abilityLabels` (fin du littéral 11
  seul — couvre les familles A et B).
- [ ] 3.2 Déclencheur : `equipmentChanges[].kind === 'spent'` dont la famille résolue est
  le translocateur → flash translocation existant sur la fiche (daté à la frame du
  spent). L'heuristique `riftTeleports` reste pour le LIEN de téléportation sur carte.
- [ ] 3.3 Mettre à jour la décision écrite `equipmentChangeSound.ts:11-19` (le
  translocateur sort du « muet » pour le VISUEL ; le son reste tel quel — pas de stem
  désigné, même règle que le crâne).
- [ ] 3.4 Tests logic (résolution de rang, sélection des spent translocateur).

Gate : `make check-types && make test-web` + gate visuel utilisateur sur un film où un
translocateur est consommé (témoins à désigner par le user).

### Lot 4 — Effets de fiche : prise de drapeau + captures (web) — MOYEN

- [ ] 4.1 `CardFxInput` gagne un canal d'événement d'objectif daté (grab/capture/zone) —
  UNE entrée générique, pas un champ par mode.
- [ ] 4.2 Sources câblées : `flag_grabs` (premier consommateur), `flag_captures`,
  `zone_captures`/`zone_secures` (le filigrane statique reste ; l'éclat s'y ajoute).
- [ ] 4.3 Rendu selon D3 (patron `replay-flash-*`, tokens sémantiques uniquement).
- [ ] 4.4 i18n FR/EN si libellé d'infobulle ajouté.
- [ ] 4.5 Tests logic + garde du plafond `ReplayCanvas`/fichiers (ratchet taille).

Gate : `make check-types && make test-web` + gate visuel utilisateur (témoins CTF +
zones à désigner).

### Lot 5 — Identité des vies, roster bots, état « hors film » (Go + web) — LOURD, sous D4

Pré-requis : D4 tranché (schéma d'artefact bougera ; visible seulement après re-cuisson
décidée par le user — jamais lancée par un agent).

- [ ] 5.1 Découpe des tracks PAR VIE dans `decimateTracks` (appliquer `lifeGapUS` comme
  `buildLifeSpans`) et nommage PAR VIE dans `nameTracks` (fin du nommage par slot) — corrige H3.
- [ ] 5.2 Roster étendu humains + bots : brancher la lecture BOT_METADATA
  (`killsource/botmeta.go` — RÉUTILISER, ne pas dupliquer le décodeur) dans le build du
  rejeu ; fiches bots nommées, suffixe « [bot] » — corrige H4.
- [ ] 5.3 Diagnostic à l'écran : publier/consommer `coverage.bridge` (livesNamed/Total,
  slotCollisions) ; troisième état front « hors film / non transmis » dans
  `playerStateAt` quand plus aucune vie n'existe pour ce joueur (fin du « Réapparition ? »
  menteur) — corrige H1/H5 côté affichage.
- [ ] 5.4 (selon D4b) `player-respawn-timer` lu et publié → compteur de réapparition réel.
- [ ] 5.5 Golden réassemblé, `EXPECTED_REPLAY_SCHEMA_VERSION` incrémenté, gates films de
  référence (patron des lots précédents : critères écrits AVANT le run).
- [ ] 5.6 Vérification ciblée sur le match Sylvanus cité par le user (slotCollisions
  attendu > 0) — re-cuisson d'UN film uniquement, avec accord explicite préalable.

Gate : `cd apps/go-api && go test ./internal/analysis/... && go vet ./...` + golangci ;
web `make check-types && make test-web` ; gate visuel user sur le film Sylvanus re-cuit.

### P6 — Export vidéo : CLOS, aucune action (déjà 30 fps posés). Statut selon D5.

---

## Protocole

- Contrat : skill `plan-execution`. Ordre strict lot N clos (gate vert + items statués
  `[x]`/`[~]`/`[!]`) avant N+1. Zéro fix hors périmètre — toute trouvaille va en
  **Découvertes** ci-dessous.
- Reprise de session : ce fichier (statuts) + `.ai/thought_log.md` (dernière entrée).
- Tout report → `.ai/V7.5/REGISTRE_REPORTS.md` avec condition de reprise.
- Commits : demander avant chaque commit (règle 16). Pas de re-cuisson sans accord.

## P8 (ajout 02/09) — Graphe « distance des kills par arme » (demande du collègue)

Le graphe n'existe pas, mais 90 % de la chaîne existe. Livré le 30/08 (LOT G.3-POC,
`b4a985163`) : un TABLEAU (`MatchKillDistanceSection.tsx`, onglet Résumé) — libellé,
nb de frags, distance moyenne, plage min–max en suffixe texte. Il rend `null` sans
donnée mesurée — cas de la quasi-totalité des matchs tant que le backfill de masse n'a
pas tourné (40/1948 artefacts = 2,1 % ; DEC-6 : backfill soumis à accord). Côté Go RIEN
à faire : `min/max/avg_distance_m` sont au contrat (`combat_tab.kill_distance_by_weapon`,
jointure `match_kill_events_latest` × `kill_positions_latest`). Le « bâton par arme »
(portée) a été FERMÉ par DEC-8 (plan retours 29/08) — le rouvrir est une décision
produit ; le coût est UN composant web (ECharts custom min→max + marqueur moyenne),
en gardant le dénominateur d'honnêteté (couverture plancher 75,8 %). Ne pas confondre
avec la précision-distance du film (`match_weapon_hit_distance`), remisée le 01/09.

## Relevé des branches non mergées (2026-09-02, demande user pour D3)

En avance sur `feat/v75` (local = distant, fetch fait) :
- `feat/precision-arme` (50) — chantier précision REMIS 01/09, dernier commit retire la
  capability et garde les acquis backend. État final propre → mergeable, mais gros lot :
  à merger avec son gate quand le user le décide (les acquis ne sont PAS dans feat/v75
  tant que ce n'est pas mergé).
- `wt/trame-film` (40) — RE trame (worktree dédié), preuves/négatifs détonation. Reste
  dans son worktree, pas de merge produit.
- `wt/ti11-cadre` (8) — RE ti=11, gates NON tenus (négatifs). Pas de merge.
- `wt/ctf-zone-retour` (4) — LIVRABLE produit (zone de retour drapeau, contestation
  retirée 31/08). Publie « schéma 29 » alors que feat/v75 est à 30 → renuméroter (31) au
  merge + golden. Pertinent de le merger AVANT le lot 5 : une seule re-cuisson couvrira
  zone de retour + identité des vies.
- `wt/assaut-bombe` (3) — RE pied de film (récompenses, gamertags en clair). Merge au
  portage produit du pied, pas avant.
- Distantes anciennes (`feat/filmdec-*` 322/321, `feat/weapon-attribution-v3` 65) :
  lignées mortes/réconciliées (31/07) — ne pas merger.

**Aucune de ces branches ne contient le design C2 des effets de fiche.**

## Découvertes (hors périmètre, à ne pas traiter dans ces lots)

- `objectiveMark.ts:208` : export `objectiveMarkFromPeriods` en attente de source KOTH
  (hill) — déjà connu, source vide par construction.
- 583 matchs Infinite sans données de kill (films expirés 2023/BTB) : vide définitif et
  légitime — éventuel message UI distinct (item 1.5 si trivial).
- `match_view_repo_extras.go:189,197` : patron `return nil, nil` sur erreur avalée —
  d'autres lecteurs du fichier sont peut-être dans le même cas (audit possible).
