# Revue adversariale — vague 2 (wt/orchestration-0907, HEAD 62c77b56d vs feat/v75 6a1496e30)

Relecteur : contexte frais, lecture seule, sans sous-agent. Skill `adversarial-review` appliqué.
Diff relu : `git diff 6a1496e30 62c77b56d -- apps` (92 fichiers). Aucune suite complète relancée
(gates déjà en cours dans le worktree) ; lecture sur pièces uniquement, tous les fichiers cités
ci-dessous ont été rouverts au HEAD `62c77b56d`.

## Méthode

Pour chaque lot (M1/M1b, M2, M3, M5, M6), j'ai énoncé les conditions qui devraient tenir pour
que le code soit correct, puis je suis allé les vérifier dans le code réel (appelants, schéma,
tests), pas seulement dans le diff. Périmètre : uniquement ce qui figure dans le diff — dette
déjà connue (`lives.go`, `document.go` > seuils) non re-signalée, conformément au brief.

## Décompte

- P0 : 0
- P1 : 0
- P2 : 1 (réserve documentée, pas un défaut prouvé)
- P3 (jetés / non recevables) : 1, listé pour traçabilité

---

## P2

**P2-1 — `apps/go-api/internal/analysis/replay/lives.go:273-360` (`voteDeathOffsets` /
`paniersParPivot`) — le dédoublonnage `min(morts distinctes, fins distinctes)` n'est pas
prouvé au-delà des 7 témoins du parc.**

Fait : la voix d'un panier de calage vaut désormais `min(n, parFin[g][b])` (borne de
Hall/König de l'appariement 1:1). C'est un majorant EXACT de ce que `refineDeathOffset` mesure
réellement, donc il ne peut pas faire GAGNER un calage faux par inflation de voix (le risque
fermé par le lot R7 est bien fermé — `TestUnAmasPlusGrosQueLeVraiCalageNEmportePasLeBudget`
le prouve par mutation, `pont_marge_test.go:196-249`).

Ce qui reste sans preuve écrite : la direction opposée — un vrai calage qui perdrait la
localisation parce que ses `fins de vie` (life-ends) en fenêtre sont, pour une raison de
qualité de données, moins nombreuses que ses morts appariables, pendant qu'un panier
concurrent atteint un `min()` plus élevé par coïncidence. `lifeEndsMS` (lives.go:474-481)
inclut TOUTES les fins de vie (mort, fin de film, coupure > 5 s) et pas seulement celles
causées par une mort, donc en pratique le compte de fins dépasse largement le compte de
morts sur un match sain — mais aucun test ni aucune mesure du parc ne borne formellement ce
cas dégénéré (« fins de vie manquantes ») que le brief demandait de vérifier. Le gate corpus
du lot M3 (superviseur, 2026-09-08, 7 témoins, 0 perte/1 gain, 48/48) est une preuve empirique
forte mais pas une preuve du cas dégénéré lui-même — aucun des 7 témoins ne semble avoir été
choisi pour le stresser spécifiquement.

Pourquoi P2 et pas plus : je n'ai pas de `fichier:ligne` en échec ni de scénario reproductible
concret — la charge de la preuve n'est pas remplie côté défaut (le filtre §4 du skill l'aurait
rejeté comme non recevable si je l'avais présenté comme un P0/P1). Je le consigne parce que
c'est exactement la question posée par le contrat (item b) et que la garantie écrite dans
`lives.go` ne couvre que la moitié du problème (l'inflation, pas la privation).

Correction minimale suggérée (si le sujet est repris) : mesurer sur le parc complet la
distribution `#fins en fenêtre / #morts appariables` pour le vrai calage de chaque témoin, et
ajouter un test qui construit délibérément un match où les fins de vie du vrai calage sont
rares (beaucoup de coupures/survivants côté adversaire, peu côté panier correct).

---

## P3 (non recevable, pour traçabilité — pas comptée dans le verdict)

**Commentaire potentiellement optimiste sur `domain/compare.go:50-51`** (`HighestCSRLabel` —
« Vide si non classé ») : ce fichier n'est PAS dans le diff de la vague 2 (dette
pré-existante, hors périmètre — règle 6 du contrat). Vérifié que la chaîne de valeur réelle
(`compare_service.go` → `ComparePage.tsx:44-46` → `localizeTierLabel`) traite bien `"unranked"`
correctement malgré ce commentaire ; aucune régression introduite par ce lot. Signalé mais
non corrigé, hors scope.

---

## Ce qui a été vérifié et TIENT (contre-preuve du contrat)

**1(a) Lien cellule — clock/offset/conversion.**
- `domain/tactical_cellule.go:1-102` : `TacticalContribution.Clock` (`"match"`/`"film"`)
  documenté et testé par question — `tactical_service_cellule_test.go` vérifie la valeur de
  `Clock` pour CHAQUE question (morts/kills/gagne/isole → `match` ; temps/routes → `film`),
  mutation possible en inversant les deux constantes.
- `coverage.go:353-366` (`deathOffsetMsIfKnown`) : témoin de connaissance =
  `DeathOffsetMatches > 0`, jamais `DeathOffsetMS != 0` — un calage mesuré à zéro exact reste
  publié comme une valeur connue, pas confondu avec une absence. Pointeur `*int64,omitempty`
  cohérent de bout en bout : `coverage_bridge.go:73-89` → `domain/replaydoc/coverage.go:75-84`
  → `service/replayview/convert_coverage.go:73` → `apps/web/src/lib/api/generated.ts:33`
  (`deathOffsetMs?: number`) — vérifié sur pièces à chaque étape, aucune perte de type ni de
  sémantique nil/0 en chemin.
- `apps/web/src/lib/replay/replayLogic.ts:343-390` (`resolveTacticalReplayInstant` /
  `resolveTacticalOpenAtFrame`) : `clock: 'match'` avec offset `null`/`undefined` rend
  `unknown-offset`, JAMAIS un repli sur 0 — testé (`tacticalReplayInstant.test.ts`). La
  conversion ms→frame (`Math.round(msToFrames(...))`) est l'inverse mathématique de
  `frameToMs` (même `frameIntervalMs`, même formule) ; `seekTo`
  (`useReplayPlayback.ts:381-387`) clampe ensuite entre `leadInFrame` et `endFrame`, donc un
  offset négatif ou un instant hors bornes ne produit ni frame négative ni crash — testé
  (`useReplayPlayback.seek.test.tsx`, cas « gagne sur le cadrage même si la fenêtre arrive
  après »).
- Avis FR/EN quand l'offset est inconnu : `i18nContract.ts`/`i18n.ts` portent
  `openAtUncalibratedTitle`/`openAtUncalibratedDescription` dans les DEUX locales (parité
  typée `Record<ReplayLocale, ReplayText>`), affiché par la route
  (`replay.tsx:257-267`) uniquement quand `showUncalibratedNotice` est vrai — jamais un saut
  approximatif présenté comme exact.
- `t` reste une CHAÎNE dans `validateSearch` (`z.object({ t: z.string().optional()... })`,
  `replay.tsx`), `clock` un `z.enum(['match','film'])` — conforme au point 4 du contrat.

**1(b) Calage `min(morts, fins)`.** Voir P2-1 ci-dessus : le sens « inflation » est fermé et
prouvé (mutation + corpus), le sens « privation » reste un point ouvert non prouvé (ni en
défaut, ni en garantie).

**1(c) `buildSlotOwnership`/`rosterEntryKey` sur un slot NON recyclé.**
`rosterLogic.ts:237-272` (`buildSlotOwnership`/`ownerAtFrame`) : pour un slot occupé par une
seule vie couvrant tout le match, `ownerAtFrame` renvoie ce joueur pour toute frame dans la
fenêtre — identique au comportement de l'ancien `indexBySlot`. Le changement ne modifie
l'attribution QUE sur un slot recyclé (plusieurs vies), confirmé par le test de mutation
`equipmentUsageLogic.test.ts` (« attribue un geste à la vie qui occupe le slot À L'INSTANT du
geste, pas au dernier occupant (P2-4) ») et par le test `rosterEntryKey` (bot join, P2-5).
`rosterEntryKey` est bien centralisé (≤ 2 copies respecté, `seatLogic.filmIndexByIdentity`
explicitement noté hors périmètre par un commentaire daté).

**1(d) Libellés — aucun écran ne perd de texte.**
- Accueil : `Title`/`OutcomeText` supprimés (`home.go`, `home_canonical_recent.go`,
  `home_service.go`) ; seul lecteur web (`MatchCard.buildMatchHeading`) recompose le repli
  via la clé `common.match_card.map_unknown` (déjà présente en FR/EN dans
  `common.toml`/`generated/common.ts`, non touchée par ce lot) — testé
  (`match-card.test.tsx`, `getAllByText('Map inconnue')`).
- Comparaison : `csrUnrankedLabel` = clé `"unranked"` (plus de littéral FR) ; chaîne de
  valeur vérifiée bout en bout jusqu'à `ComparePage.tsx:44-46` (`localizeTierLabel`) et
  `skillTiers.ts` (`TIER_NAME_BY_KEY['unranked']` FR/EN) — testé côté Go
  (`compare_service_test.go`) et web (`skillTiers.test.ts`).
- Playlists classées : `ranked_playlists_labels.toml` couvre les 16 `asset_id` de
  `rankedplaylists.go` (compté sur pièces, correspondance 1:1) ; `NameEN()`/`NameFR()`
  deviennent des méthodes lisant ce TOML — tous les appelants (`career.go`,
  `csr_history_backfill.go`, `leaderboard_world_repo.go`, `probe-world-stats/main.go`)
  migrés en cohérence (recherché `pl.NameEN`/`pl.NameFR` sans parenthèses : aucun résidu).
- Armes : `weapon_families.name_en/name_fr` vérifié SANS LECTEUR restant, Go et web (grep
  croisé `weapon_families` × `name_en|name_fr` : seuls les fichiers de la migration/du
  registre/des tests y touchent). `weapon_name_labels`/`weapon_names.toml` reste la source
  du nom par arme, non affectée.

**2. ART / DB.**
- `tactical_repo_ownership.go` (nouveau) : lecture via `SharedReadDB()` (même pattern que le
  reste du fichier `tactical_repo.go`, pas de nouvelle primitive) ; utilise
  `StartTimeCanonicalSQL` (garde-rail horodatage) et `campaignExclusionToken` avec le même
  usage que `queries_career.go` (pas un nouveau pattern SQL).
- Migration `purge_weapon_families_labels_columns` : rebuild CTAS-swap transactionnel, calqué
  sur le précédent `purge_weapons_name_fr_column` ; idempotente (`columnExists` en garde),
  garde anti-perte (`rebuilt != before` → erreur avant tout DROP), rollback complet sur
  erreur. Ordonnancement : `order.go` la place juste après `add_weapon_registry` (créatrice
  de la table), dépendance déclarée dans `order_dependency_test.go`. Tests dédiés
  (`steps_metadata_purge_weapon_families_labels_test.go`, tag `integration`) couvrent
  legacy→purgé, déjà-purgé (no-op), conservation des rows + PK.
- Aucun élargissement d'allowlist ART (`no_art_patterns_test.go` absent du diff) ; aucune
  lecture brute d'une vue `_latest` contournée.

**3. Multi-titre.**
- Aucune comparaison `slug == "..."` introduite (grep sur tout le diff : zéro résultat).
- `rankedplaylists` reste dans `internal/games/halo_infinite/` (title-scoped par
  construction), le TOML embarqué documente lui-même pourquoi il n'est pas encore exposé au
  mécanisme `assets.toml` générique (limite du loader pour des appelants purs
  migration/sync — condition de reprise écrite dans le TOML).
- `handleGetCellule` dégrade proprement sur `ErrCapabilityNotSupported` → 503, testé au
  niveau service (`TestCellule_CapabiliteAbsente`) ET au niveau handler
  (`TestTacticalHandler_CelluleCapabilityNotSupported503`).

**4. Web — hygiène.**
- Aucune couleur hex ni classe Tailwind couleur ajoutée dans `features/`/`components/`
  (grep du diff, zéro résultat).
- Query key `tacticalCellule` déclarée dans `lib/query/keys.ts`, guard title-scope mis à jour
  (`keys.title-slug.guard.test.ts`).
- `routeTree.gen.ts` non touché par le diff.
- `instantToFrame`/`TACTICAL_REPLAY_FRAME_INTERVAL_MS` retirés avec leurs tests
  (`tacticalView.logic.ts`/`.test.ts`) — code mort éliminé, pas laissé « au cas où ».

**5. Hygiène générale.** `slog.*Context` avec `"err", err` partout dans le nouveau code
observé ; aucune erreur avalée repérée (`MatchsOuvrables`, `Cellule`, migration) ; pas de
nouveau fichier > 500 L ni fonction > 80 L parmi les fichiers neufs revus ; le ratchet
`no_french_label_literal_test.go` ne fait que BAISSER (538 → 500, aucune entrée remontée) ;
nouveaux garde-rails (`no_weapon_family_label_literal_test.go`,
`TestNoNewModePlaylistLabelLiteral`) volontairement étroits mais cohérents avec le style
existant du dossier `archlint`.

**M6.** `projeterRastersTactiques` réutilise bien `r.doc` (le document déjà lu par
`lireArtefacts`) au lieu de rouvrir le fichier ; seul `ProjeterRasterTactique` (rattrapage
CLI, pas de document en main) relit encore via `ouvrirArtefact` — le seam de test
(`raster_lecture_unique_test.go`) prouve `lectures == 0` pour le chemin du cycle.

---

## Verdict

La vague 2 est fusionnable dans `feat/v75` telle quelle : aucun constat P0 ni P1 recevable
après vérification sur pièces de chaque surface à risque du contrat (lien cellule/horloge,
budget de calage, attribution de slot, libellés, migration DB, multi-titre, hygiène web).
Le seul point ouvert (P2-1, le sens « privation » du dédoublonnage `min()` dans le calage)
est une question non tranchée plutôt qu'un défaut prouvé : à consigner dans
`.ai/thought_log.md` / registre des reports comme réserve pour un futur lot décodeur, pas un
bloquant. Recommandation : merger, puis ouvrir une entrée de réserve citant
`lives.go:273-360` pour qu'un futur travail sur le décodeur mesure explicitement le ratio
fins/morts du vrai calage sur l'ensemble du parc.
