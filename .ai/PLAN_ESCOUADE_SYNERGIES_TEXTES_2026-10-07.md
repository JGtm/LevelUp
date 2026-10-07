# Plan : Escouade › Synergies — retrait de la riposte et de la hauteur, textes sans personne — 2026-10-07

> Source qui FAIT FOI : `.ai/HANDOFF_ESCOUADE_SYNERGIES_TEXTES_2026-10-07.md` (§1 décisions de
> l'utilisateur des 2026-10-05, 06 et 07 ; §3 périmètre A ; §4 périmètre B). Rien n'y est rediscuté.
>
> Contrat d'exécution : skill `plan-execution` (ordre strict, une étape à la fois, gate passé avant
> la suivante, chaque item statué `[x]` fait et vérifié / `[~]` couvert ailleurs / `[!]` non fait
> avec justification, zéro correction hors périmètre — les découvertes vont au §8).
>
> Statut : **pré-approuvé par le superviseur le 2026-10-07** (exécution enchaînée sans attente) ;
> **CLOS côté exécutant le 2026-10-07** : E1 `4cf68289b`, E2 `c23114e18`, E3 `82020eacf`, E4 (commit de
> clôture). Revue adversariale, push et fusion : superviseur.
> Branche `feat/escouade-synergies-textes`, partie de `origin/feat/v75` à `879f31bbf` (Tactique v2
> fusionnée : `git grep mesurerEchange` vide). Worktree `C:\Users\Guillaume\Downloads\Scripts\LevelUp-wt-escouade`.
> Ni push, ni fusion, ni rebase : le superviseur pousse, lance la revue adversariale et fusionne.

## 0. Objectif, critère de succès, hors périmètre

**Objectif.** L'onglet Synergies de l'Escouade ne montre plus la riposte ni la hauteur ; sa section
de coordination devient « Appui et portée » (Appui, puis Rôles de portée seule). Tout ce qui perd
son dernier lecteur part, web, Go et contrat. Les textes de l'Escouade et le message « Match pas
encore synchronisé » de la Vue match sont reformulés sans personne ni « camp », et la garde
`textesSansPersonne.test.ts` couvre désormais tout le jeu de l'Escouade.

**Succès.** (1) Aucun lecteur web de `echange`, de `squad.riposte.*`, de `squad.hauteur.*`, des
champs `elevation_*` ; (2) aucun producteur Go de `SquadEchange`, de `coordination.Echanges` /
`Ripostes`, ni des champs de dénivelé du profil de portée ; contrat régénéré, strictement
soustractif ; (3) la garde vue ROUGE sur l'état avant reformulation, VERTE après ; (4) gates verts.

**Hors périmètre.** `features/tactical/`, `service/tactical_service*.go` ; notes de version (rédigées
en « tu ») ; le bloc `assist_pairs` et `SquadAppuiCard` (gardés) ; le nuage de portée et son Go ;
les onglets Emprise et Objectif ; l'Above / Level / Below de `analysis/weapon_range.go` (Portée par
arme, autre grandeur, lecteur `DeltaZ` conservé).

## 1. Décisions d'exécution (prises sur pièces le 2026-10-07)

- **D1 — `coordination.Mesurer` RESTE.** Le handoff le fait partir « après vérification par grep » :
  le grep montre deux lecteurs de production hors échange, `analysis/coordination/bloc.go:125-126`
  (l'appui) et `isolation.go:62`. Restent aussi `domain.Couverture`, `KillEvent`,
  `EquipesParMatch`, `SeuilEchantillonFaible`. Partent : `trade.go` et `riposte.go` entiers
  (`Echanges`, `Ripostes`, `FenetreEchangeMs`, `PlafondRiposteTardiveMs` et leurs aides non
  exportées, sans autre appelant), `domain.MortSuivie`, `PaireEchange`, `BilanEchanges`.
- **D2 — Le journal des morts reste lu par l'appui** (`service/coordination_block.go:135`) : le
  port `TacticalRepository.KillEvents` et son lecteur DuckDB ne bougent pas (`platform/duckdb`
  non touché). Côté Escouade, `TeammatesService.tacticalRepo`, `caps` et `WithEchange` perdent leur
  seul lecteur et partent, avec l'appel de câblage `registry_pages_home.go`. Précision relevée en E2 :
  l'appui n'appelle `KillEvents` que pour son UNIVERS ; les événements eux-mêmes n'ont plus de
  lecteur — découverte §8.5, laissée au superviseur.
- **D3 — Le dénivelé du profil de portée part de bout en bout** : champs `ElevationMedianM`,
  `ElevationLobbyDeltaM`, `LobbyElevationMedianM`, leur calcul (`matchRangeMesures.dz`,
  `ecartDeDenivele`, `matchRangeLobby.elevationM`) et `match_range_elevation_test.go`.
  `analysis.MeasuredKill.DeltaZ` reste (lecteur `weapon_range.go:230`).
- **D4 — Titre et aide de la section** : « Appui et portée » / « Support and range », dans le jeu
  de l'Appui (`squad/i18n.ts`, bloc `sections`). Aide en deux phrases, la mesure et son
  périmètre, sans personne (texte exact au journal E1).
- **D5 — La garde du vocabulaire de la coordination** (`vocabulaireCoordination.guard.test.ts`)
  recommande « riposte » pour remplacer « vengeance » / « taux d'échange » : la notion étant
  retirée, la garde est retournée — elle bannit des chaînes de `features/squad` et du manifeste
  les formules de la notion retirée (vengeance, venger, taux d'échange, riposte) et garde
  « assistances croisées » → « appui ». C'est la preuve, par garde, du critère (1).
- **D6 — Option `etiquetteBout` du nuage de portée** : seul `grandeur="hauteur"` la passait à vrai ;
  elle part avec sa constante de marge et sa branche `endLabel`. Même sort pour la prop
  `prefixeTest` de `SquadRangeRolesTape` (une seule bande sur la page, toujours `squad-portee`).
- **D7 — Options de graphes partagés devenues sans lecteur** (relevé sur pièces pendant E1) : la
  riposte était le dernier lecteur de `HistogramChart` `binHatched`, `showValues`, `windowMark`,
  `thresholds` (et de la hachure `aria.decal` qui les servait) et de `DonutChart.arcLabelKind`
  (avec `ChartPointDonut.valueLabel`). Elles partent avec leurs tests. `HistogramChart.yAxisLabel`
  et `formatBin`, sans lecteur de production eux non plus, RESTENT : ils font partie de l'API de
  base du wrapper décrite par le catalogue `components/charts/README.md` (au même titre que
  `xAxisLabel`), pas d'un ajout pour la riposte.
- **D8 — Notes de version** : le §4 du handoff les met hors périmètre pour leur registre (« tu »,
  décision antérieure). Leur CONTENU devenu faux (la riposte sur l'Escouade, les rôles de hauteur)
  est corrigé en E4, dans le registre existant, comme l'ont fait les lots Sessions et Vue match.

## 2. Étapes

### E1 — Web : Synergies sans riposte ni hauteur

- [x] E1.1 `SquadSynergiesPage.tsx` : section « Appui et portée », condition `(assistPairs || rangeProfiles)`, deux rangées, « Rôles de portée » seule ; commentaire de section réécrit au présent.
- [x] E1.2 Suppression des fichiers de la riposte : `SquadRiposteCard.tsx` (+ test), `SquadRiposteMatricePanel.tsx` (+ test), `SquadRiposteSessionsChart.tsx`, `charts/squadRiposteSessionsChart.ts` (+ test), `squadRiposte.logic.ts` (+ test), `squadRiposte.fixtures.ts`, `squadRiposte.i18n.test.ts`, `squadRiposteStrings.ts`.
- [x] E1.3 Manifeste `squad.toml` : bloc `squad.riposte.*` retiré ; titre et aide de la section dans `squad/i18n.ts` (D4) ; `generated/squad.ts` régénéré.
- [x] E1.4 Hauteur : prop `grandeur`, `GrandeurProfil`, `mesureDeJoueur` (branche hauteur), paramètre `grandeur` de `seriesPortee`, `PREFIXE_GRANDEUR` / préfixe `squad.hauteur.*` (manifeste), option `etiquetteBout` (D6) ; tests correspondants (`squadRangeRoles.logic.test.ts`, `SquadRangeRolesCard.test.tsx`, `squadRangeRolesStrings.test.ts`) ; `SquadRangeRolesTape.tsx` vérifié.
- [x] E1.5 `lib/api/types.ts` : alias `SquadEchange*`, champ `echange` de `TeammatesPageResponse`, commentaire du dénivelé et intitulé « Riposte » du bloc D22.
- [x] E1.6 Gardes adaptées : `sessionBarsTrendChart.guard.test.ts` (appelant retiré), `vocabulaireCoordination.guard.test.ts` (D5) ; tests de page (`SquadSynergiesPage.test.tsx`) sur la nouvelle section.
- [x] E1.7 Commentaires devenus faux dans les fichiers lecteurs (`SquadAppuiCard.tsx`, `SquadRangeRolesCard.tsx`, `squadRangeRolesStrings.ts`, `SessionBarsTrendCard.tsx`, `sessionBarsTrendChart.ts`, `lowSampleNote.ts`, `objectifStrings.ts`, `timeseriesCoordinationStrings.ts`, `charts/squadRangeRolesChart.ts`) : contrat au présent, sans citer un fichier supprimé.
- [x] E1.8 Grep de vérification : `riposte|Riposte|echange|SquadEchange|squad.hauteur|elevation_|grandeur` dans `apps/web/src` (hors généré) — aucun lecteur ; grep des tests Go qui lisent `apps/web/src` (leçon S5 de Sessions).
- **Gate E1** : manifestes régénérés ; purge `node_modules/.tmp` puis `tsc -b` ; `npm run lint` ; `npm run lint:fields` ; `npm run lint:colors` ; `node tools/lint-cross-feature-imports.mjs` ; vitest `src/features/squad src/components/charts src/lib src/features/timeseries src/features/session-detail` ; knip (indicatif).

### E2 — Go et contrat : le bloc `echange` et le dénivelé

- [x] E2.1 `domain/squad_echange.go` supprimé ; champ `Echange` de `domain.TeammatesPageResponse` retiré.
- [x] E2.2 `service/teammates/teammates_squad_echange.go` et ses tests (`_test`, `_contrat_test`, `_maquette_test`) supprimés ; câblage (`teammates_service_sections.go`, `teammates_service.go` : champ, `WithEchange`, `tacticalRepo`, `caps`) ; part échange de `teammates_service_loads_test.go` ; `api/wire/registry_pages_home.go`.
- [x] E2.3 `analysis/coordination/trade.go`, `riposte.go` et leurs tests supprimés ; `domain/coordination.go` réduit (D1) ; `no_naked_rate_test.go` (liste blanche réduite de `BilanEchanges`, `MortSuivie`, justifications), `doc.go` du paquet au présent ; `bloc_appui_golden_test.go` relu (l'appui reste).
- [x] E2.4 Dénivelé (D3) : `domain/match_range_profile.go`, `analysis/match_range_profile.go`, `analysis/match_range_elevation_test.go`.
- [x] E2.5 Contrat : `make openapi-gen`, `make generate-types`, snapshot `contract-surface.snapshot.json` (`UPDATE_CONTRACT_SURFACE=1`), `tools/lint-contract-ratchet.mjs`, `make openapi-check`.
- [x] E2.6 Baseline : différence AVANT / APRÈS des fonctions `Test*` de l'arbre ; paires retirées de `.ai/baselines/tests_pre_migration.jsonl` + paragraphe daté dans `scripts/check_test_baseline.sh`. Résultat : 38 `Test*` disparus de l'arbre (15 `analysis/coordination`, 2 `analysis`, 21 `service/teammates`), aucun apparu ; AUCUN des 38 n'est dans la baseline (intersection vide, et 0 ligne de la baseline pour ces paquets) : baseline et script inchangés.
- [x] E2.7 Grep : `coordination\.(Echanges|Ripostes)`, `SquadEchange`, `FenetreEchangeMs`, `Elevation` dans `apps/go-api` — aucun reste hors historique.
- **Gate E2** : `gofmt -l` ; `go vet ./internal/...` ; `go build ./internal/... ./cmd/levelup ./cmd/openapi-gen` (jamais `go build ./...`) ; `go test` des paquets touchés (`domain`, `analysis`, `analysis/coordination`, `service/teammates`, `service`, `api/...`, `archlint`) ; golangci-lint des paquets touchés ; contrat ; tsc / vitest du web lié au contrat.

### E3 — Textes sans personne et garde étendue

- [x] E3.1 Garde `textesSansPersonne.test.ts` étendue : `squad/i18n.ts` entier (FR, EN), les `*Strings.ts` de `features/squad/` (`squadFocusStrings`, `squadRangeRolesStrings`, `emprise/vehicleStrings` en plus des trois déjà gardés), manifeste `squad.toml` entier ; listes FR (+ « reviens », « vérifie ») et EN (« you », « your » déjà présents) ; vue ROUGE, nombre de chaînes fautives noté.
- [x] E3.2 Reformulation FR et EN de chaque chaîne fautive (`squad/i18n.ts`, `*Strings.ts`, `squad.toml`) selon la sémantique du 06/10 ; tests qui citent les anciennes chaînes mis à jour.
- [x] E3.3 `match-view/i18n.ts` `notSyncedDescription` FR et EN : phrase factuelle, sans impératif ni personne.
- [x] E3.4 Garde VERTE ; mutations de contrôle (une chaîne fautive réintroduite dans chaque nouvelle source : rouge).
- **Gate E3** : manifestes régénérés ; tsc purgé ; lint ; `lint:fields` ; `lint:colors` ; vitest des features touchées.

### E4 — Clôture

- [x] E4.1 CHANGELOG EN et FR (`docs/CHANGELOG.md`, `docs/FR/CHANGELOG.md`) : lignes de coordination et de portée/hauteur alignées, entrée du lot. Fait : résumé de tête, « Coordination », « Portée face au lobby », entrée « Escouade › Synergies sans riposte ni hauteur », « Page Escouade » ; notes de version EN et FR, deux lignes (D8).
- [x] E4.2 Registre `.ai/REGISTRE_REPORTS.md` : les deux lignes du §2 du handoff closes. Plus une section du lot avec les découvertes 8.5, 8.6, 8.7.
- [x] E4.3 Statut du handoff mis à jour ; statuts du plan ; entrée `.ai/thought_log.md`.
- **Gate E4 (clôture)** : `go test ./internal/...` (au premier plan, par lots si besoin) ; `go vet ./internal/...` ; `make go-api-lint` ; `make openapi-check` ; `tsc -b --force` après purge ; `npm run lint` ; `lint:fields` ; `lint:colors` ; vitest complet ; knip (indicatif) ; `bash scripts/check_test_baseline.sh` en mode présence si exécutable localement.

## 8. Découvertes hors périmètre (consignées, non corrigées)

- **8.1 (E1, incident)** — une commande de E1 a été lancée avec un `python - 2>/dev/null` tapé par
  erreur en tête (stdin vide : aucun code exécuté, sortie jetée). Aucun effet sur l'arbre ; la
  suite de la commande (édition par `node`) a seule agi. Consigné au titre de la règle « pas de
  Python ».
- **8.2 (E1)** — `features/tactical/i18n.ts:5` cite « `squadEchangeStrings` », fichier qui
  n'existe plus. Hors périmètre (`features/tactical/` interdit à ce lot).
- **8.3 (E1)** — `components/charts/sessionBarsTrendChart.test.ts` nomme ses séries d'exemple
  « Je riposte » (données de test d'un composant générique, aucun lecteur produit). Laissé.
- **8.5 (E2) — hypothèse D2 à corriger, à trancher par le superviseur.** Après ce lot,
  `domain.TacticalKillEvents.Events` n'a plus AUCUN lecteur de production : le seul appelant restant
  de `port.TacticalRepository.KillEvents`, le bloc de coordination des pages Sessions et Séries
  temporelles (`service/coordination_block.go:135`, `ajouter` l. 178-198), ne lit que
  `lecture.Univers` (matchs, drapeau de mesure, équipes) — exactement ce que rend déjà
  `port.TacticalRepository.Univers` (`platform/duckdb/tactical_repo_univers.go:312`, même
  `chargerUnivers`). La requête `QTacticalEvents` (`platform/duckdb/tactical_repo.go:303`) lit donc
  le journal des morts pour rien à chaque lecture de coordination, et `domain.KillEvent` n'est plus
  consommé. Correction proposée (lot à part) : faire lire `Univers` au bloc de coordination, puis
  retirer `KillEvents`, `TacticalKillEvents`, `KillEvent`, `QTacticalEvents`, les doubles de test
  (`coordination_block_test.go`, `session_page_coordination_test.go`, `tactical_mock_test.go`) et
  adapter les tests DuckDB qui passent par `KillEvents` (dont le garde-rail I2 de l'ADR 0036,
  `TestTacticalRepo_PerimetreRestreint_FenetresBornees`, et sa référence dans l'ADR). Non fait ici :
  c'est un recâblage de la lecture d'autres pages et d'un garde-rail de l'ADR 0036, pas une
  suppression du périmètre. Les commentaires qui annonçaient un consommateur (port, domaine, paquet
  `coordination`) sont corrigés ; celui de `QTacticalEvents` (« l'echange se mesure… ») est laissé
  au lot qui retirera la requête.
- **8.6 (E3)** — `squad/i18n.ts` `empty.noSelectionDescription` FR « Choisis 1 à 3 coéquipiers pour
  analyser les synergies de l'escouade. » reste à l'impératif de la 2e personne : aucun mot de la liste
  du §4 du handoff ne le porte, la garde ne le voit pas. Non traité (hors des sept chaînes et de la
  liste arrêtées) ; à décider avec l'utilisateur, comme l'extension de la garde aux impératifs.
- **8.7 (E3)** — les clés `squad.header.*` de `squad.toml` (dont `solo_section_title`, reformulée
  parce que la garde lit le manifeste en entier) n'ont aucun lecteur dans `apps/web/src` (grep du
  2026-10-07). Clés mortes antérieures au lot ; non supprimées ici.
- **8.4 (E1)** — `lib/formatters/lowSampleNote.ts` et sa garde racontent la copie historique dans
  `SquadEchangeKpi` (histoire datée, vraie). Laissé.

## 9. Journal

- 2026-10-07 — plan écrit et pré-approuvé ; relevé sur pièces : D1-D6.
- 2026-10-07 — **E1 clos.** Section « Appui et portée » (titre et aide dans `squad/i18n.ts`) ;
  12 fichiers de la riposte supprimés, `squadRangeRolesStrings.test.ts` aussi (il ne gardait que la
  parité entre grandeurs ; les clés sont désormais typées) ; blocs `squad.riposte.*` (42 clés) et
  `squad.hauteur.*` (19 clés) retirés du manifeste ; D6, D7 appliqués ; garde du vocabulaire
  retournée (D5) ; garde de la frise réduite à son appelant restant. Gate : manifestes régénérés
  (23 manifestes, 3 513 clés) ; `tsc -b` après purge 0 ; ESLint 0 erreur (26 avertissements, comme
  la base) ; `lint:fields` 0 ; `lint:colors` 0 ; imports croisés 7 ≤ 7 ; vitest `squad`,
  `components/charts`, `lib`, `timeseries`, `session-detail` : 317 fichiers / 3 021 tests verts
  (premier passage : 5 dépassements du délai de 5 s sur des gardes qui balayent l'arbre, sous
  charge ; verts aux deux passages suivants) ; knip 0 / 0 / 0.
- 2026-10-07 — **E2 clos.** Bloc `echange` retiré de bout en bout (`domain/squad_echange.go`, producteur
  et ses trois fichiers de test, câblage `WithEchange` / `tacticalRepo` / `caps`, champ du contrat) ;
  `trade.go` et `riposte.go` du paquet `coordination` avec leurs tests ; `MortSuivie`,
  `PaireEchange`, `BilanEchanges` ; liste blanche de `no_naked_rate_test.go` réduite des deux types ;
  dénivelé du profil de portée (trois champs, calcul, test). `Mesurer` reste (D1). Contrat :
  `openapi.yaml` −147 lignes, `generated.ts` −56, cinq schémas `SquadEchange*` retirés du snapshot de
  surface par sa procédure (garde vue rouge sur ces cinq noms avant la mise à jour). Gate :
  `gofmt -l` muet ; `go build ./internal/... ./cmd/levelup ./cmd/openapi-gen` 0 ; `go vet ./internal/...`
  0 ; `go test -count=1` `domain`, `analysis`, `analysis/coordination`, `service/teammates`,
  `archlint`, `api/wire`, `service`, `api/...` : tous ok (le golden `TestOpenAPIYAMLIsUpToDate`
  rouge avant régénération, vert après) ; golangci-lint des paquets touchés 0 issue ;
  `lint-contract-ratchet` propre ; `make openapi-check` à jour ; web `tsc -b` purgé 0, vitest
  `lib/api` + `squad` 818 tests verts. `platform/duckdb` et `persist` non touchés : pas de passe
  `-tags=integration`. Découverte 8.5.
- 2026-10-07 — **E3 clos.** Garde `textesSansPersonne.test.ts` étendue au jeu entier de l'Escouade
  (`squad/i18n.ts` FR et EN, libellés de focus, « Rôles de portée » ; véhicules déjà couverts par
  `EMPRISE_TEXT`), au manifeste `squad.toml` entier, et la liste FR complétée de « reviens »,
  « vérifie » (EN : « you », « your » y étaient). Un code de locale (`en-US`) n'est pas un texte :
  exclu par sa forme. **ROUGE avant reformulation : 9 tests sur 34, 34 entrées fautives** —
  19 chaînes sources distinctes (`squad/i18n.ts` 7 FR et 5 EN, `squad.toml` 1 clé FR + EN,
  `session.toml` 3 lignes FR, `timeseries.toml` 1 ligne FR, Vue match 1 FR), vues aussi par les
  jeux qui les relisent (Sessions, Emprise du match), plus le faux positif `en-US`. **VERTE après**
  (34/34). EN reformulés en parité, y compris là où la liste EN ne voyait rien (« Check… »,
  « check back »). Mutations rouges : « Vos » et « Your » dans `squad/i18n.ts`, « Mes rôles de
  portée » et « notre session » dans `squad.toml` (manifeste régénéré), « reviens » dans la Vue
  match, « camp » dans `squad/i18n.ts`, « vérifie » dans `emprise/vehicleStrings.ts` (rouge sur
  les quatre jeux qui le relisent) ; faux positifs contrôlés : « vouloir », « vérifié »,
  « campagne », « mesuré », « youth », « Usage » passent. Gate : manifestes régénérés ; `tsc -b`
  purgé 0 ; ESLint 0 erreur (26 avertissements) ; champs 0 ; couleurs 0 ; vitest `squad`,
  `match-view`, `session-detail`, `timeseries`, `components`, `lib` : 406 fichiers / 3 858 tests
  verts (deux tests mis à jour sur les nouvelles chaînes : `squadCompositionGapHint.test.tsx`,
  `MatchViewPage.test.tsx`). Découvertes 8.6, 8.7.
- 2026-10-07 — **E4 clos, plan clos côté exécutant.** Documents (CHANGELOG EN / FR, notes de version
  EN / FR, registre, handoff). Gate de clôture : `go test -count=1` des 173 paquets de
  `./internal/...` en trois lots au premier plan (156 ok, les autres sans test, code de sortie 0
  partout) ; `go vet ./internal/...` 0 ; `make go-api-lint` 0 issue ; `make openapi-check` à jour ;
  web `tsc -b --force` après purge 0, ESLint 0 erreur (26 avertissements), champs 0, couleurs 0,
  imports croisés 7 ≤ 7, vitest complet 859 fichiers / 9 096 tests verts (5 fichiers ignorés, la
  base ; un premier passage avait un échec isolé de la garde de balayage
  `lib/clipboard/useCopyToClipboard.guard.test.ts`, verte seule et au passage complet suivant),
  knip 0 / 0 / 0. `scripts/check_test_baseline.sh` NON lancé localement : en mode autonome il relance
  la suite du module entier (`./...`, liaison CGO de tous les paquets dont `cmd/`), interdite sur ce
  poste par le superviseur (disque) ; la présence a été vérifiée par différence en E2.6 (intersection
  vide) et la CI Linux joue le script.
