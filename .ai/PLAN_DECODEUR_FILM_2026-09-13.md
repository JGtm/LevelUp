# PLAN — Fiabiliser le décodeur de film, la cuisson des artefacts et leur lecture

> Écrit le 2026-09-13 à partir de `ARCHITECTURE_CIBLE_DECODEUR_FILM_2026-09-12.md` (sections 1 à
> 13), `HANDOFF_DECODEUR_FILM_2026-09-13.md` (sections 1 à 6), des addendums 1 à 5b du journal
> (entrée `[2026-09-12] Architecture cible du décodeur de film`) et de
> `V7.5/RAPPORT_LOT_H_VERSIONS_2026-09-13.md` (P1 à P5). Vérifié sur pièces le même jour :
> chemins d'après le lot E (`internal/games/halo_infinite/film/{filmdec,replay,killsource}`),
> `SchemaVersion = 54`, `KillSourceDecoderRev = killsource-2026-09-12`, ratchet des variables de
> paquet gelé à **96** (le document d'architecture dit 118 : c'est la mesure du 05/09, le lot E
> a resserré), corpus gate à 12 témoins, corpus d'équivalence à 13 films, 0 `Benchmark`.
>
> **Contrat d'exécution : skill `plan-execution`** (ordre strict, aucune case vide, zéro fix hors
> périmètre, vérification sur pièces avant de coder et avant de cocher). Ce fichier est la source
> de vérité de l'avancement. Revue du plan : skill `plan-review`. Revue de chaque lot : skill
> `adversarial-review` (contexte frais, worktree propre).

## 0. Objectif et critère de succès

**Objectif** : que le décodage d'un film, la cuisson de son artefact et la lecture de cet
artefact par le web soient fiables par construction, c'est-à-dire :

1. un film se lit par un **profil** (build, carte) résolu une fois et immuable, sans globale de
   paquet ni littéral de largeur dans les lecteurs ;
2. les cinq correctifs prouvés par la recherche des 12 et 13 septembre sont en production
   (équipe d'un événement à l'octet 37, cinq états par défaut, cadre d'image-clé d'état complet,
   registre à l'octet 8, table des joueurs et équipe lues dans le film) ;
3. chaque changement de décodeur fait sonner un gate (empreinte, corpus, équivalence, schéma), et
   les preuves traversent la frontière Go / web (fixtures produites par Go, contrat à l'exécution) ;
4. la publication se rejoue depuis des faits persistés sans redécoder, avec une révision par
   calque.

**Critère de succès mesurable à la clôture** (tous vérifiés par commande, consignés en §5) :

| # | Critère | Commande de vérification |
|---|---|---|
| S1 | 0 variable de paquet mutable dans le décodeur, `LockProcessDecode` supprimé, ratchet inversé | `go test ./internal/archlint/ -run 'FilmdecPackageVars|DecodeLock'` |
| S2 | Deux films décodés en parallèle sous `-race` sans course | `go test -race ./internal/games/halo_infinite/film/... -run TestDeuxFilmsEnParallele` |
| S3 | Équivalence octet pour octet à zéro différence sur le corpus figé après chaque pas structurel | `go run ./cmd/replay-equiv` |
| S4 | Corpus gate : zéro perte à chaque lot, gains nommés aux lots de comportement | `go run ./cmd/replay-corpus-gate --base=<sha>` |
| S5 | `Track.Team != -1` et roster complet sur une cuisson **hors ligne** (sans base) d'un témoin | golden par build (`TestGoldenAssembly`) |
| S6 | Fixtures de contrat produites par Go, consommées par vitest, matrice de compatibilité, zod à la frontière, empreinte de forme | `make test-web`, `go test ./internal/games/halo_infinite/film/replay/ -run 'Shape|Contract'` |
| S7 | Build inconnu = erreur typée + compteur expvar + film mis de côté, jamais une lecture au profil précédent | test unitaire nommé au lot 3.1 |
| S8 | Un document rejoué depuis les faits persistés est identique à l'octet au document cuit depuis le film | test du lot 4.1 sur le corpus |
| S9 | Les archétypes utiles (ti=9, 11, 12, 35, 42, 43, véhicules) ferment à 100 % en image-clé | ratchet `KeyframeClosure` (0.A.3) sur les mini-films, oracle `imagecle_fermeture` sur les 6 films de recherche |

## 1. Périmètre fermé et décisions

### 1.1 Ce que ce plan traite (fermé)

Les cinq jalons M0 à M4 de la §3, et rien d'autre. Chaque jalon est fusionnable seul ; l'ordre
est fait pour s'arrêter proprement après n'importe lequel.

### 1.2 Ce que ce plan NE traite PAS (écrit pour ne pas diverger)

| Hors périmètre | Où il vit | Condition de reprise |
|---|---|---|
| La sémantique des valeurs 1..6 de la table par type de `chunk_00` (`vtable+0x30`) | idem | un build dont la grammaire d'un type change sans changer de build en clair |
| Le `+200` o de `HI_1_4_1`, le build des 5 films sans section d'identification, `b36`/`b40`/octets 39-43 du pied, le jeton de 48 bits, ti=13 muet, ti=4 par build | idem (handoff §5) | un témoin au corpus gate qui les rend visibles |
| Le libellé affiché d'un désignateur d'équipe (exige le corpus `gamefiles`) | idem | jamais dans ce chantier : le document publie l'index d'équipe, pas un libellé |
| Toute retouche d'interface (fiches, compact BTB, couleurs) | hors chantier | la densité BTB reste `mode_category === 'BTB'` (décision utilisateur du 2026-09-07), jamais un compte de sièges |
| Le port des chantiers voisins (véhicules, assaut, duels) | leurs plans | sans objet |

Toute découverte pendant l'exécution va en §4 et n'est pas traitée (règle 7), sauf si elle
bloque le gate du lot courant.

### 1.3 Décisions tranchées (fermes, ne se rediscutent pas en cours de route)

| # | Décision | Source |
|---|---|---|
| D1 | `FilmContext` absorbe `Profile` : un seul objet par film, le profil résolu à la construction, les mémos (registre, bande de slots) restent paresseux | architecture §5 |
| D2 | Une inférence n'est pas une valeur de profil : là où le profil sait, l'inférence est un oracle de test ; là où il ne sait pas, erreur typée | architecture §5 |
| D3 | La grammaire vient du jeu ; quand l'écrivain est lisible dans l'exe, elle se prend chez l'écrivain, pas par mesure ; chaque valeur de profil cite sa fonction et sa date dans le code | architecture principes 9 et 14, handoff §5 bis.1 |
| D4 | Un pas structurel (M2) n'est clos qu'à zéro différence d'équivalence ; une différence arrête le pas, elle ne se justifie pas. Un lot de comportement (M1, M3) est jugé par le corpus gate (zéro perte, gains nommés) et localisé par l'équivalence | architecture §7 |
| D5 | Les correctifs courts (M1) passent AVANT la révision (M2) ; la RE en instruments peut courir en parallèle mais aucun port de composant pendant M2 | handoff §5 bis |
| D6 | Une seule recuisson du parc par jalon, sur signal de l'utilisateur, jamais par lot ; le corpus gate et `replay-equiv` cuisent en racine jetable, jamais dans le parc | mémoire (2026-08-31, 2026-09-01) |
| D7 | Les globales retirées au pas 2 le sont famille par famille avec double écriture datée (kill-switch règle 11 : bascule, retrait cible = lot 2.3, critère = 0 globale) | architecture contrôle 10 |
| D8 | Le `cadre` d'image-clé d'état complet devient LA lecture (constantes), l'option `KeyframeFullStateOpt` disparaît avec `shiftArchetypeLevels` : pas de variabilité « au cas où » | architecture §9, handoff §4 |
| D9 | `analysis/` n'importe jamais `games/{slug}` ; l'allowlist (`sessionusage`) se vide au pas 5 | ratchet `no_title_package_in_analysis_test.go` |
| D10 | ADR 0034 porte les invariants (couches, profil, porte unique aux octets, build inconnu, faits / publication) ; écrit à M0, amendé à la clôture de M2 et de M4 ; EN-only | architecture §13, règle 15 |
| D11 | Le web n'est touché qu'à la frontière de normalisation (0.B, 4.2, 4.3) ; aucune retouche de rendu | architecture §11, §12 |
| D12 | Les chemins neufs (faits persistés, profils) passent par `PathResolver` et par un catalogue versionné jamais écrit à l'exécution (ratchet `no_runtime_versioned_catalog_write_test` étendu) | CLAUDE.md, principe 15 |

### 1.4 Décisions validées par l'utilisateur le 2026-09-13 (fermes)

| # | Décision | Verdict utilisateur |
|---|---|---|
| V1 | Les ports de composants manquants (« 62 % ») FONT PARTIE du plan : lot 3.6, dans la structure révisée (M2), archétype par archétype, gate = ratchet de couverture (0.A.3) et oracle de fermeture. Les 103 lecteurs existants gardent leur grammaire (prouvée à l'écrivain, 4 ports sur 5 confirmés largeur pour largeur en R7-d) ; M2 leur donne la FORME cible (fonction pure du couple profil, bits) ; les ports neufs naissent directement dans cette forme | inclus (question utilisateur : « pourquoi hors plan ? ») |
| V2 | Gates de décodage autorisés pour la durée du chantier, mais **par échantillon** : régime court à chaque lot (équivalence sur 10 films au plus, fixés en 0.A.1), régime complet (corpus d'équivalence entier + corpus gate) seulement aux clôtures de jalon et aux lots marqués « risqué » (1.2, 2.5). Racines jetables, un seul décodage à la fois. Recuisson du parc et backlog killsource = signal séparé (D6) | ok sur échantillon représentatif |
| V3 | Fusion dans `feat/v75` à chaque jalon, sur signal, CI verte au niveau job | ok |
| V4 | Équipe (lot 1.7) : **le film est la seule source, sans repli sur la base**. Désignateur > 0 = équipe ; 0 (FFA) = aucune équipe, publiée telle quelle ; film muet = inconnue, comptée. La base ne sert que de CONTRÔLE (compteurs accord / contradiction / silence), jamais de valeur. Vérifié sur pièces : le web colore les joueurs par `team_side` de la feuille de match (`rosterLogic.ts:160`), pas par l'artefact, donc aucun rendu ne change ; les portages de drapeau (`TeamOf`) prennent l'équipe du film | ok : « si le décodeur est fiable, pas besoin du repli » |
| V5 | `analysis/filmsource` passe sous `film/internal/source` au pas 5 | ok |
| V6 | M4 (publication) fait partie du chantier, en dernier ; arrêt possible après M3 sans dette | ok |
| V7 | Mini-films par build : au plus 1 Mio chacun, `chunk_00` obligatoire, un chunk d'image-clé, le pied ; 7 builds | ok |

## 2. Organisation

### 2.1 Branches, worktrees, agents

- **Branche d'intégration** : `feat/recherche-decodeur-film` (worktree
  `C:/Users/Guillaume/Downloads/Scripts/LevelUp-wt-recherche-film`). Un lot = une branche
  `feat/decfilm-<lot>` créée depuis l'intégration, un worktree temporaire
  `LevelUp-wt-decfilm-<lot>` par agent (exécuteur ET relecteur chacun le sien), `GOCACHE` privé
  (`.gocache-<role>`, dans `.git/info/exclude`), `GOLANGCI_LINT_CACHE` isolé. Fusion dans
  l'intégration par le pilote après revue ; push après chaque fusion (préfixe `feat/` = CI).
- **Deux agents au plus** : en régime normal, un exécuteur Opus (effort `high` sur tout lot qui
  touche `filmdec`, `killsource`, `replay` ; `medium` sur docs et web) et un relecteur Opus
  (effort `high`, skill `adversarial-review`, 2 rondes au plus, P0/P1 corrigés par l'exécuteur
  sur la branche du lot avant fusion). Deux muteurs en parallèle seulement quand la §3 le dit
  (0.A avec 0.B) : jamais deux muteurs de `filmdec`.
- **Le pilote garde pour lui** : création des branches et worktrees, fusions, push, surveillance
  CI, exports de faits depuis la base (`levelup replay-facts-export`, base libre), recuissons et
  backlogs (sur signal), registre des reports, journal de clôture, mémoire.

### 2.2 Environnement d'exécution (à recopier dans chaque brief)

- Le cache de films (`data/cache/film_chunks`, 1 351 films, 7 builds) et le parc
  (`data/cache/replays`) ne vivent que dans `LevelUp-go-migration`. Depuis un worktree :
  `LEVELUP_REPO_ROOT=C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration` pour
  `replay-equiv` ; le corpus gate détecte le parc par le `.git` commun ; les instruments corpus
  lisent `CHUNK00_FILMS` (répertoires absolus `C:/...`, séparés par `;`).
- **Un seul décodage à la fois sur la machine** (`filmproc.AcquireSolo`) : jamais deux gates
  de décodage en parallèle, jamais de boucle de shell autour d'un CLI qui décode, jamais
  d'écriture dans le parc. Signal d'étouffement (`fork: Resource temporarily unavailable`) =
  tuer ses processus.
- Ghidra : instance partagée `HaloInfinite.exe` base `0x140000000`, lecture seule.
- Oracle base : `data/backups/pre-chaine-2026-09-09/shared_matches_v2.duckdb` en
  `access_mode=read_only` (`go run ./cmd/diag_q`) ; jamais la shared de production en RW.
- Pas de `-race` dans un paquet qui tire DuckDB ; `-gcflags=all=-d=checkptr=0` si nécessaire.
- Aucun nouveau Python ; les fixtures se génèrent en Go.

### 2.3 Gates communs à tout lot (en plus du gate propre du lot)

```bash
cd apps/go-api
gofmt -l ./internal ./cmd                      # vide
go vet ./internal/games/halo_infinite/film/... ./internal/archlint/ ./internal/replaybuild/ ./internal/sync/killcollector/ ./internal/analysis/objectiveevents/
go test ./internal/games/halo_infinite/film/... ./internal/archlint/ ./internal/replaybuild/ ./internal/sync/killcollector/ ./internal/analysis/objectiveevents/ ./internal/domain/replaydoc/
make -C ../.. go-api-lint                      # baseline non accrue
```

Lots qui touchent le web : `make check-types` puis `make test-web` (vitest hors sandbox).

Lots qui touchent le décodeur ou le constructeur (tout M1, M2, M3, 4.1), deux régimes (V2) :

| Régime | Quand | Commandes |
|---|---|---|
| **Court** (à chaque lot) | clôture de tout lot décodeur | `go run ./cmd/replay-equiv -films <échantillon 0.A.1>` (10 films au plus : un par build, plus `60ae07c4`, `d9781168`, `fb1a1a72`) |
| **Complet** | clôture de jalon (M0 à M4) ; lots 1.2 et 2.5 ; sur demande du relecteur | `go run ./cmd/replay-equiv` (corpus entier) puis `go run ./cmd/replay-corpus-gate --base=<sha de l'intégration avant le lot>` |

Résultat consigné en §5 (commit, commande, sortie, durée). **Un gate non consigné n'a pas eu
lieu.** Un lot dont le régime court montre une différence inattendue passe en régime complet
avant toute analyse.

### 2.4 Clôture d'un lot (dans l'ordre)

1. gates du lot verts, sorties propres ; 2. tous les items statués `[x]` / `[~]` / `[!]` ;
3. ce fichier mis à jour (cases + §4 + §5) dans le dernier commit de code du lot (pas un commit
à part : `paths-ignore` de la CI) ; 4. revue adversariale, constats corrigés ; 5. fusion dans
l'intégration, push, CI verte au niveau job ; 6. entrée `.ai/thought_log.md` ; 7. point d'étape
à l'utilisateur (fait / non fait et pourquoi / découvert).

## 3. Jalons et lots

Légende : S = un lot court (une session), M = une session pleine, L = deux sessions ou plus.
Chaque item est cochable. « Preuve » = ce que le gate doit montrer, en plus des gates communs.

---

### M0 — Fondations : les oracles avant le premier changement

Critère d'entrée : aucun (exécutable aujourd'hui). 0.A et 0.B en parallèle (deux muteurs,
fichiers disjoints) ; 0.C ensuite, pendant les revues.

#### Lot 0.A — Oracles du décodeur (Go) — M, exécuteur Opus high

- [ ] 0.A.1 **Corpus d'équivalence par build.** `replay/testdata/equivalence/CORPUS.txt` gagne une
      colonne `version / build` (découverte 1 du lot H) et un témoin par build présent au cache
      (`HI_1_4_1`, `1_8_0`, `1_9_0`, `1_10_0`, `1_11_0`, `1_12_0`, `1_13_0`), en réutilisant les
      témoins du lot H (`111fa685`, `e5adf7b2`, `60ae07c4`, `a349fea8`) ; cible 20 films au plus.
      Les `.facts.json` neufs sont exportés par le PILOTE (base libre) et remis à l'agent.
      Preuve : `go run ./cmd/replay-equiv` rend zéro différence sur les 13 films existants ; les
      films neufs sont figés par `-films <ids> -update` avec leur `# digest-grammar` ; durée par
      film consignée en §5 (budget de référence). L'ÉCHANTILLON du régime court (V2) est
      nommé en tête de `CORPUS.txt` (`# echantillon-court:`) : un film par build (7) plus
      `60ae07c4` (région sur 2 bits), `d9781168` (pire déroulage sain), `fb1a1a72` (multi-manche).
- [ ] 0.A.2 **Mini-films par build et goldens.** Un `testdata/minifilm_<short8>/` par build (V7 :
      `chunk_00` + un chunk portant des images-clés + le pied, au plus 1 Mio, `PROVENANCE.txt`
      avec build, version, carte, commande de fabrication en Go). `TestGoldenAssembly` et
      `TestGoldenInputs*` deviennent des tables sur ces mini-films : un golden d'assemblage et un
      `inputs_<short8>.bin.gz` par build. Preuve : goldens verts en CI ; `TestGoldenInputsVersionGuard`
      refuse toute régénération sans montée explicite.
- [ ] 0.A.3 **Inventaire et ratchet de couverture d'image-clé par archétype** (handoff §4 bis,
      étape 1). Fonction de production `filmdec.KeyframeClosure(fc) map[ti]{closed,total,
      blocking string}` fondée sur `WalkKeyframeFullState` (108 bits, mots de taille, état par
      défaut) ; instrument corpus (`CHUNK00_FILMS`) qui imprime, par archétype, les composants
      ordonnés avec leur statut (porté / manquant / BLOQUANT) et le taux de fermeture ; ratchet CI
      sur les mini-films de 0.A.2 : « la couverture par archétype ne descend jamais » (golden
      `testdata/keyframe_closure.golden`). Preuve : golden commis ; une mutation de largeur
      (ti=6) rougit le ratchet.
- [ ] 0.A.4 **Empreinte du décodeur étendue à `filmdec`.** Constante `filmdec.GrammarRev`
      (forme `grammar-AAAA-MM-JJ`) ; test miroir de `decoder_rev_fingerprint_test.go` qui hache
      tous les `.go` hors tests de `filmdec/` et de `killsource/` : une source qui change sans
      montée de `GrammarRev` rougit. L'en-tête du test écrit la règle : montée de `GrammarRev`
      obligatoire à tout changement de grammaire ; montée de `KillSourceDecoderRev` si la sortie
      de killsource peut changer (backlog) ; montée de `SchemaVersion` si le contenu cuit change.
      Preuve : golden avec historique ; une ligne changée dans `traverse.go` rougit.
- [ ] 0.A.5 **Budget de temps.** `replay-equiv` publie la durée par film si ce n'est pas déjà le
      cas (vérifier sur pièces) ; `BenchmarkBitReaderReadBits`, `BenchmarkTraverseEntity`,
      `BenchmarkScanBipedPositions` sur le mini-film `000d5950` ; `testdata/bench_baseline.txt`
      produit par `go test -bench . -run ^$ -count 5` et comparé par `benchstat` à chaque clôture
      de M2 (budget : +10 % au plus). Preuve : baseline commise, commande documentée dans
      `docs/COMMANDS.md`.

Gate 0.A : gates communs ; `go run ./cmd/replay-equiv` (0 différence sur 13, références neuves
figées) ; `go test ./internal/games/halo_infinite/film/... -run 'Golden|KeyframeClosure|GrammarRev'`.

#### Lot 0.B — La frontière Go / web (architecture §12) — M, exécuteur Opus high

- [x] 0.B.1 **Fixtures de contrat produites par Go.** Test Go `replay/contract_fixtures_test.go`
      qui cuit `minifilm_000d5950` et écrit
      `apps/web/src/features/match-replay/test/fixtures/go/replay_schema_<N>.json` (mode
      `-update`, sinon compare comme un golden). Vitest fait passer chaque fixture par
      `normalizeReplayDocument` et par les logiques pures des calques (`*_logic.ts`,
      `rosterLogic.ts`, `replayLogic.ts`). `testDoc.ts` se construit depuis la fixture (le
      `schemaVersion: 1` écrit à la main disparaît).
      **Fait** (2026-09-13) : la cuisson réutilise `buildGolden` (mêmes entrées figées que
      `TestGoldenAssembly`, zéro octet de film). La fixture est **compressée**
      (`replay_schema_54.json.gz`, 401 589 o contre 3 318 459 en clair — même régime que
      `testdata/inputs_000d5950.bin.gz`, 1 033 495 o), et le dossier porte un
      `manifest.json` (version + liste) que `testDoc.ts` lit pour sa version de schéma :
      le document complet n'est plus chargé que par le test de contrat. Régénération à
      DEUX conditions (`-update` **et** `REPLAY_CONTRACT_UPDATE=1`) pour qu'une
      régénération du golden d'assemblage ne re-bénisse pas le contrat du web en silence.
      Consommateur : `test/goFixtures.contract.test.ts` (contrat zod, normalisation,
      `replayLogic`, `rosterLogic`, `abilityChargeLogic`, `equipmentUsageLogic`,
      `equippedLogic`, `hillHoldLogic`, `padControlLogic`, `roundsLogic`, `seatLogic`).
- [x] 0.B.2 **Matrice de compatibilité.** `replaySchemaStatusLogic.ts` nomme
      `MIN_RENDERABLE_SCHEMA_VERSION` ; test : toute fixture au-dessus rend, toute fixture en
      dessous donne l'état `stale` du badge, jamais une exception. Chaîne UI neuve éventuelle en
      FR et EN dans `i18n.ts`.
      **Fait** (2026-09-13) : `MIN_RENDERABLE_SCHEMA_VERSION = 27`, valeur **décidée par
      l'exécuteur faute de valeur au plan** et justifiée sur pièces — la chronique ne porte
      que deux montées qui RETIRENT au client un champ promis (v6 `Inventory.a`,
      v27 `weaponChanges[].until`) ; à partir de 27 toute montée est additive ou de contenu.
      **À confirmer par le pilote** (cf. « ce qui a besoin du pilote »). Le test de contrat
      vérifie que toute fixture Go est ≥ ce seuil ; sous le seuil, le badge dit `stale`
      MÊME sans en-tête à comparer (au lieu de `unknown`) et ne nomme aucune cible inventée
      — d'où une chaîne neuve FR + EN (`schemaBadgeStaleNoTargetFmt`), parité par typage.
      **Revue R1, constat C5 (P1) corrigé** : la valeur n'était tenue par AUCUN test (27 -> 50
      laissait 202 fichiers et 3 016 tests verts). Elle est désormais ÉPINGLÉE, avec en
      commentaire les deux retraits qui la fondent et la règle de changement (décision
      produit + entrée de chronique + fixture Go à la nouvelle borne) ; un second test exige
      que la borne soit une version que la chronique déclare vraiment. Mutation 27 -> 50
      rejouée : rouge.
- [x] 0.B.3 **Contrat à l'exécution (zod).** Schéma zod du document de rejeu, assertion de type
      dans les deux sens contre le type généré d'OpenAPI (dérive impossible sans rougir `tsc`),
      actif dans les tests et derrière le badge admin (erreur de validation affichée dans le
      badge, jamais un rendu qui tombe). Aucune dépendance nouvelle.
      **Fait** (2026-09-13) : `lib/replay/replayDocumentSchema.ts` — les 57 clés de la
      RACINE, validées pour leur nature (nombre, chaîne, tableau, objet, table) ; les types
      d'éléments sont portés par `z.custom<T>()` (donc vus de `tsc`) et **non re-validés**,
      choix écrit dans l'en-tête (la forme profonde est gardée côté producteur par 0.B.4).
      Trois assertions de type : égalité stricte des ensembles de clés + assignabilité dans
      les deux sens (`replayDocumentSchema.test.ts`). Branché dans `lib/replay/queries.ts`
      AVANT `normalizeReplayDocument` (après elle, tout tableau est comblé) ; le manquement
      voyage par `ReplayDocumentReady.contractIssue` jusqu'au badge admin, nouvel état
      `invalid` (ton `destructive`), chaîne FR + EN. Aucun rendu ne tombe : le document est
      servi quand même. Effet de bord assumé de la règle n° 6 : le motif `Equals`/`Expect`
      en serait à sa 3e copie -> centralisé dans `lib/types/typeEquality.ts` avec son
      garde-rail `typeEquality.guard.test.ts`, deux copies migrées.
      **Revue R1, constat C1 (P1) corrigé** : le schéma était `z.object`, qui DÉPOUILLE les
      clés inconnues — `shots` renommé `shotz` passait, le badge disait « à jour » et le
      calque se rendait vide. La racine et les bornes sont désormais `z.strictObject` ; le
      manquement nomme la clé (zod la range dans `issue.keys`, pas dans le chemin). Preuve :
      le renommage est rejoué sur le document RÉEL de la fixture Go. L'en-tête du module dit
      maintenant exactement ce qui est couvert et ce qui ne l'est pas (élément d'un calque,
      contenu d'un bloc, clés des tables de libellés).
      **Constat C2 (P1) corrigé** : le garde de `typeEquality` exigeait le littéral `A` comme
      second terme — une copie sous d'autres noms de paramètres passait. Le motif porte sur la
      FORME du conditionnel différé ; contre-test avec trois jeux de noms.
      **Revue R2, constat R2-1 (P1) corrigé** : la généralisation de C2 était à MOITIÉ faite —
      le second terme était libre, mais `<T>` restait littéral, si bien qu'une copie écrite
      `<Q>() => Q extends X ? 1 : 2` traversait encore. Le paramètre générique est désormais
      capturé et lié à sa seconde occurrence par une RÉFÉRENCE ARRIÈRE ; contre-tests avec
      `<Q>` et `<Z>`, plus un négatif (`<Q>() => T extends …`) que la référence arrière rejette.
      **Constat R2-2 (P1) corrigé** : `validateReplayDocument` fabriquait une PHRASE, et cette
      phrase était française — en locale anglaise le badge affichait
      `contract violated (cle(s) inconnue(s) : shotz)`, fragment FR hors `i18n.ts` au milieu
      d'une phrase EN (règle n° 1). Le module rend maintenant une DONNÉE (`ReplayContractIssue` :
      `unknownKeys` / `invalidField` / `malformed`) et c'est le badge qui la met en mots, par
      trois entrées `contract*` FR + EN à parité typée. Le repli `document non conforme au
      contrat` est passé par la même porte dans le même geste. La liste des clés et le texte de
      zod restent bruts : données et diagnostic technique d'administrateur, pas de la langue.
- [x] 0.B.4 **Empreinte de forme côté Go.** `replay/document_shape_test.go` : hachage réfléchi de
      `ReplayDocument` (champs, balises JSON, `omitempty`) figé dans
      `testdata/document_shape.golden` avec la `SchemaVersion` ; rougit si la forme change sans
      montée de `SchemaVersion` ou sans entrée dans `document_chronicle.go` pour la version
      neuve ; même empreinte sur le jumeau `domain/replaydoc` (les deux formes doivent coïncider).
      **Fait** (2026-09-13) : golden de 908 lignes (en-tête + forme EN CLAIR, pas seulement
      le hachage — une empreinte qui bouge ne dit pas ce qui a bougé). Le ratchet est dans la
      **régénération** : elle est REFUSÉE quand l'empreinte change alors que `SchemaVersion`
      n'a pas bougé. Éprouvé par mutation (champ ajouté à `ReplayDocument`) : golden rouge,
      jumeaux rouges, régénération refusée ; mutation annulée. Deux neutralisations
      explicites, sans quoi les jumeaux ne coïncident pas (découvertes D3) : champs triés
      par nom (l'ordre n'est pas le contrat — `replayview/parity_test.go` compare déjà par
      arbres) et types nommés rendus par leur type de BASE. Le test miroir existant
      (`replayview/parity_test.go`) n'est pas refait : il est cité, et cette empreinte
      n'ajoute que l'unicité des deux formes dans le même golden que la version.
      Entrée de chronique exigée pour la version courante via
      `testutil.ReplayChronicleVersions` (helper partagé avec 0.B.5, méta-testé).
- [x] 0.B.5 **Les anciens goldens ne se régénèrent jamais.** Test qui, pour chaque version de la
      chronique inférieure à la courante, vérifie `replaybuild.Digest{SchemaVersion: v}.UpToDate()
      == false`, et que `writeArtifactBytes` refuse une rétrogradation (preuve par version, le
      garde existe).
      **Fait** (2026-09-13) : `replaybuild/artifact_schema_history_test.go`, liste des 51
      versions DÉRIVÉE de `document_chronicle.go` (jamais écrite à la main). Trois preuves
      par version : `Digest.UpToDate()` et `ArtifactUpToDate(path)` faux ; `StoreArtifact`
      refuse le dépôt et n'écrit rien ; `writeArtifactBytes` refuse l'appauvrissement à
      schéma égal. **Découverte D1** : `writeArtifactBytes` ne refuse PAS une rétrogradation
      de VERSION — le refus de version vit en amont dans `validateArtifact`.
- [x] 0.B.6 **Ratchet sur les fixtures.** `testDoc.guard.test.ts` étendu : aucun littéral
      `schemaVersion:` dans les tests web hors `fixtures/go/`.
      **Fait** (2026-09-13) : signature `(?:latestS|s)chemaVersion\s*(?::|=\{)\s*-?\d` — les
      props JSX sont visées comme les littéraux d'objet, et les annotations de type
      (`schemaVersion: number`) comme les valeurs dérivées ne rougissent pas. Contre-test
      d'inertie inclus (trois fautifs, trois légitimes). Les deux tests qui écrivaient un
      numéro (`replaySchemaStatusLogic.test.ts`, `ReplaySchemaBadge.test.tsx`) le dérivent
      désormais du manifeste Go. Les fixtures étant du JSON, elles sont hors du balayage par
      construction.
      **Revue R1, constats C3 et C4 (P1) corrigés** : le motif ne voyait ni l'appel
      POSITIONNEL (`computeReplaySchemaStatus(48, 51)` — la forme même que ce lot venait de
      retirer) ni la CLÉ CITÉE (`"schemaVersion": 3`) ; et le balayage s'arrêtait à la
      feature, laissant deux tests du rejeu écrire un numéro à la main
      (`lib/replay/heatPaint.test.ts`, `routes/.../replay.gate.test.tsx`). Trois signatures
      désormais, chacune avec son contre-test ; balayage étendu à `src/lib/replay/` et
      `src/routes/` (par `featureFiles.fichiersSous`, sans toucher au premier volet du garde,
      qui reste borné à la feature) ; les deux fautifs dérivent du manifeste Go. Les quatre
      mutations rejouées : rouges.

Gate 0.B : gates communs ; `make check-types` ; `make test-web` ; `make openapi-check` ;
`go test ./internal/games/halo_infinite/film/replay/ ./internal/replaybuild/ ./internal/domain/replaydoc/`.

- [ ] 0.B.7 (après fusion de 0.A) **Une fixture par build** : `contract_fixtures_test.go` itère
      les mini-films de 0.A.2 ; taille totale bornée (consignée). — S, exécuteur du lot 0.C ou
      relecteur de 0.B.

#### Lot 0.C — ADR 0034 et registre — S, exécuteur Opus medium

- [x] 0.C.1 `docs/adr/0034-film-decoder-profile-and-layers.md` (EN) : couches cibles et règles de
      dépendance, profil immuable, porte unique aux octets, politique de build inconnu (erreur
      typée, expvar, film mis de côté, entrées « présumées » listées par un test), séparation
      faits / publication, ratchets nommés, kill-switch daté de la double écriture.
      **Fait** (2026-09-13) : neuf décisions D-1 à D-9, statut `Accepted (2026-09-13)` avec
      amendements annoncés aux clôtures de M2 et de M4, gabarit des ADR 0030 / 0033.
      Chaque invariant nomme son garde-rail, et chacun porte son verdict : EXISTANT
      (`archlint/{filmdec_package_vars_test.go,decode_lock_held_test.go,no_film_reread_test.go,`
      `filmsource_leaf_test.go,no_title_package_in_analysis_test.go,`
      `no_runtime_versioned_catalog_write_test.go,no_second_artifact_sink_test.go}`,
      `sync/killcollector/decoder_rev_fingerprint_test.go`,
      `film/replay/{document_shape_test.go,contract_fixtures_test.go}`,
      `replaybuild/artifact_schema_history_test.go`, `service/replayview/parity_test.go`,
      `testDoc.guard.test.ts`, `replayDocumentSchema.ts`) ou « to be added at step N »
      (`GrammarRev` 0.A.4, `KeyframeClosure` 0.A.3, `ErrUnknownBuild` 1.5.2,
      `TestProfilPresumes` 2.1.1, `TestProfilEgaleGlobales` 2.1.3, `TestDeuxFilmsEnParallele`
      2.3.3, `no_raw_film_bytes_outside_source_test.go` 2.4.3, `film_layers_deps_test.go`
      2.5.0, `film_file_size_test.go` 2.7.2, faits persistés 4.1.1, `layers` 4.2.1).
      Kill-switch de la double écriture écrit règle 11 (bascule = date du lot 2.1, retrait
      cible = lot 2.3, critère = 0 variable de paquet mutable). Une section
      « Corrections to statements made elsewhere » consigne D1, D2 et D6 du lot 0.B ; la
      limite de D5 (la borne 27 se vérifie sur de la prose, faute d'inventaire de clés par
      version) est écrite dans D-8.2. Équipe = décision V4, sans repli sur la base.
      Longueur : 274 lignes (218 hors lignes vides) — au-dessus de la cible de 250, assumé :
      les neuf invariants et les trois corrections y tiennent chacun en une sous-section.
- [x] 0.C.2 CLAUDE.md : ligne ADR 0034 dans la liste ; `docs/adr/README` si index.
      **Fait** (2026-09-13) : 3 lignes ajoutées à la suite de `0033`, même style. `docs/adr/`
      n'a PAS de README (`ls docs/adr/ | grep -i readme` vide) : rien d'autre à mettre à jour.
- [x] 0.C.3 `V7.5/REGISTRE_REPORTS.md` : les lignes de la §1.2 (avec condition de reprise).
      **Fait** (2026-09-13) : 5 lignes ajoutées à la fin du tableau principal, origine
      « plan decodeur 2026-09-13 §1.2 » pour les trois items du hors-périmètre écrit
      (sémantique des valeurs 1..6 de la table par type ; les sept résidus de format du
      handoff §5 ; libellé affiché d'un désignateur), plus D7 et D8 du lot 0.B avec leur
      condition de reprise recopiée de la §4. Les ports de composants ne sont PAS inscrits :
      ils sont AU plan (lot 3.6, décision V1), donc pas un report. Le hors-périmètre
      « retouche d'interface » et « port des chantiers voisins » non plus : ils n'ont pas de
      condition de reprise propre (« hors chantier », « sans objet »).

Gate 0.C : `go test ./internal/archlint/ -run Mojibake` ; relecture pilote.

**Clôture M0** : fusion dans `feat/v75` sur signal (V3). Aucun octet d'artefact ne change.

---

### M1 — Les correctifs courts, un par un, sous gate (valeur visible)

Critère d'entrée : M0 fusionné dans l'intégration. Lots strictement séquentiels (même paquet) ;
le second agent est le relecteur du lot précédent. Chaque lot monte `GrammarRev` ; les lots qui
changent le contenu cuit montent `SchemaVersion` (une entrée de chronique chacun) ; UNE
recuisson du parc à la clôture de M1 (D6). Ordre fondé sur le coût et sur les dépendances
(1.4 avant 1.7 ; 1.5 avant 1.6 et 1.8).

#### Lot 1.1 — Pied de film : l'équipe d'un événement est à l'octet 37 — S, high

Sur pièces : `internal/analysis/objectiveevents/film.go:182,195,251` lit `b55` (« NON fiable ») ;
le champ `teamRaw` n'a aujourd'hui aucun consommateur.

- [ ] 1.1.1 `decodeTh10Block` lit l'équipe à `ebs+37*8` ; commentaires `:33`, `:182`, `:195`
      corrigés (doc inversée interdite) ; `b38` noté comme doublon observé, non lu.
- [ ] 1.1.2 Le type exporté des événements d'objectif porte `Team int` (valeur du film, -1 si
      absent) : lecture possible en 1.6 / 1.7, sans consommateur ici.
- [ ] 1.1.3 Test par mutation sur un bloc de pied du mini-film (`chunk_03` de `000d5950`) :
      remettre `b55` rougit ; corpus : `TestResidusPiedOctetEquipe` reste l'oracle (665/665).
- [ ] 1.1.4 `GrammarRev` montée ; empreinte verte.

Preuve : `replay-equiv` zéro différence (rien de publié ne change) ; corpus gate zéro perte,
zéro gain.

#### Lot 1.2 — Le registre commence à l'octet 8 — M, high (le plus risqué de M1)

Sur pièces : `registry.go:12` suppose `[u32 kind][u32 flags][nom @ +8]` depuis l'octet 0 ; le
jeu lit des entrées de `0x104` octets `[nom @ +0][u32 niveau @ +0x100]` depuis l'octet 8
(`keyframe_fullstate_loop.go:110-120`). Conséquence : `Flags[i]` = niveau du composant `i-1`,
« kind » = queue du nom voisin (0 sur 1 066 slots), et `traverse.go:1301` passe `arch.Level(i)`
à chaque lecteur.

- [ ] 1.2.1 Mesure AVANT de coder : liste des lecteurs de composants qui consomment le niveau
      (`consumeByNameCapturing` et en dessous) et, sur les mini-films, la distribution des
      niveaux `Level(i)` contre `Level(i+1)` par archétype consommé (ti=9, 11, 12, 35, 40, 42,
      43). Rapport court dans le journal du lot.
- [ ] 1.2.2 `parseRegistry` lit à l'octet 8 avec l'entrée du jeu (nom à +0, niveau à +0x100) ;
      `Flags[i]` = niveau du composant `i` ; le faux « kind » disparaît ; `registryBlockTail` et
      son commentaire (« un cran plus loin ») corrigés ; `FilmMajorVersionFromHeader` inchangé.
- [ ] 1.2.3 `shiftArchetypeLevels` et `KeyframeFullStateOpt.LevelShift` supprimés (le décalage
      n'existe plus) ; tests associés adaptés.
- [ ] 1.2.4 Empreinte du registre (`registry_fingerprint.go`, `testdata/ecs_table.tsv`, table des
      empreintes connues) recalculée sur la nouvelle lecture, provenance datée ; le test corpus
      `lot3_registre_compte_research_test.go` (50 blocs) reste vert.
- [ ] 1.2.5 Test unitaire : sur le `chunk_00` d'un mini-film, le niveau du composant `i` de ti=35
      est celui que `Level(i+1)` rendait avant (preuve de l'équivalence du décalage) et le
      terminateur d'un bloc est entièrement nul.
- [ ] 1.2.6 `GrammarRev` montée. Si le corpus gate montre une PERTE : c'est un lecteur calibré sur
      le mauvais niveau ; sa correction est DANS le lot (elle bloque le gate), consignée nommément.

Preuve : `replay-equiv` localise les balayages qui changent (attendu : ceux qui lisent un niveau
qui diffère entre voisins) ; corpus gate zéro perte, gains nommés ; `SchemaVersion` montée si un
octet cuit change.

#### Lot 1.3 — Les cinq états par défaut manquants — S, high

Sur pièces : `default_state_arch.go:54-77` ; ti=14 classé stub à tort (en-tête ligne 25).

- [ ] 1.3.1 Entrées ti=14 `V ; R(5)`, ti=17 `V ; R(7)`, ti=21 `R(18)`, ti=29 `V`, ti=47 `V ; R(5)`,
      chacune avec sa fonction Ghidra (relevé 8.5 de `NOTE_IMAGECLE_ETAT_COMPLET_2026-09-13.md`)
      et sa date ; commentaire STUB de ti=14 corrigé.
- [ ] 1.3.2 Test permanent « `n2` constant » sur les mini-films : pour tout archétype à état fixe,
      le second mot de taille est constant ; fausser une largeur (ti=21 → 17) rougit.
- [ ] 1.3.3 Ratchet de couverture (0.A.3) régénéré avec justification : fermeture qui MONTE
      (attendu : + records sur ti=14/17/21/29/47, projection 30,8 % sur les 6 films de recherche).
- [ ] 1.3.4 `GrammarRev` montée.

Preuve : `replay-equiv` (attendu : différences seulement sur les balayages delta des archétypes
14, 17, 21, 29, 47 s'ils passent par `consumeKeyframeDefaultState`) ; corpus gate zéro perte.

#### Lot 1.4 — Le cadre d'image-clé d'état complet en production — M, high

Sur pièces : `navpoint_radial_scan.go:339`, `objective_scan.go:373` (les deux consommateurs),
`keyframe_record_walk.go:181` (`walkOneKeyframeRecord`) lisent
`TraverseEntity(br, reg, 0)` après `SetBitPos(r.Bit + keyframeRecordTIBit)` ; la bonne forme est
`WalkKeyframeFullState` (`keyframe_fullstate_loop.go:69`).

- [ ] 1.4.1 `WalkKeyframeFullState(pay, recBit, reg)` sans option (D8) : en-tête 108, mots de
      taille, état par défaut ; la boucle historique `WalkKeyframeBody` / en-tête 64 + masque
      supprimée si plus aucun appelant de production (inventaire sur pièces ; sinon consignée).
- [ ] 1.4.2 Les deux consommateurs (et `walkOneKeyframeRecord` si en production) branchés ;
      `Mask` d'un record d'image-clé = tous présents.
- [ ] 1.4.3 Compteur expvar `filmdec.keyframe.<ti>.{closed,total}` (ADR 0009) publié par le
      balayage de production ; ratchet 0.A.3 régénéré (attendu : 0 → 14 % sur les 6 films de
      recherche, + 1.3 → 30,8 % ; ti=6/15/18/22 à 100 %).
- [ ] 1.4.4 `GrammarRev` montée.

Preuve : `replay-equiv` (différences attendues : `bombArmings` si un témoin s'engage, sinon
aucune) ; corpus gate zéro perte ; `TestImageCleFermetureParArchetype` rejoué par le pilote sur
les 6 films de recherche = chiffres de la note 5a.

#### Lot 1.5 — L'identité et la table des joueurs lues dans `chunk_00` — M, high

Lecteurs purs, sans consommateur (les consommateurs sont 1.6, 1.7, 1.8). Sources : grammaire des
notes `NOTE_SECTION3_CHUNK00`, `NOTE_SECTION3_SLOTS`, `NOTE_RESIDUS_CHUNK00` et des instruments
`section3_*`, `residus_slots_*`, `profil_roster_*` (`s3sChaine`, `rsChaine`, `rsDelta`).

- [ ] 1.5.1 `filmdec.ReadFilmIdentity(chunk0) (FilmIdentity, error)` : section 2 (table par type,
      version en clair, build, saveur, identifiant de build, changelist) ; `ErrNoFilmIdentity`
      typé pour les 5 films sans section ; horodatage du match (32 bits) porté.
- [ ] 1.5.2 `filmdec.ReadPlayerTable(chunk0, ident) ([]PlayerSlot, PlayerTableReport, error)` :
      32 slots, 16 champs, décalage d'un bit (`0x0CB45C`), largeur du bloc de personnalisation
      PAR BUILD (table `player_table_profile.go` : 1 852 / 1 492 / 1 312 / 2 052 o, un
      commentaire de provenance par ligne), slots vacants écartés ; `ErrUnknownBuild` typé +
      compteur expvar `filmdec.unknown_build.<build>` (principe 10) ; le rapport porte le
      calibrage lu sur le film comme CONTRÔLE (accord / contradiction).
- [ ] 1.5.3 Champs publiés par slot : `FilmIndex` (= rang), `XUID`, `Gamertag`, champs courts.
- [ ] 1.5.4 Tests : unitaires sur les mini-films (7 builds : 32 slots lus, gamertags imprimables,
      rang = index) ; corpus `CHUNK00_FILMS` : 32 slots sur 1 351 films (oracle des instruments) ;
      mutation : fausser une largeur de build rougit.
- [ ] 1.5.5 `GrammarRev` montée.

Preuve : `replay-equiv` zéro différence (aucun consommateur) ; corpus gate zéro différence.

#### Lot 1.6 — Le registre d'identité prend la table du film comme lien direct — M, high

Décision utilisateur du 2026-09-07 : « l'index c'est l'index », table d'identité unique par
film, morts en repli. Sur pièces : registre d'identité du constructeur (`identity_registry*.go`),
`replaybuild/matchfacts.go` (`equipesParXUID`, `PlayerIndexTable`).

- [ ] 1.6.1 Provenance `film_table` dans `identity.players` : le lien `index ↔ xuid ↔ gamertag`
      vient de 1.5 quand la section existe ; la table de la base devient un CONTRÔLE (compteurs
      `accord / contradiction / silence` publiés dans `coverage.identity`) ; l'inférence par le
      fil des morts reste le repli des 5 films sans section (D2).
- [ ] 1.6.2 Roster : gamertag du film quand disponible ; taille réelle de l'escouade publiée
      comme DONNÉE (aucune règle UI ne change, §1.2).
- [ ] 1.6.3 Cuisson hors ligne (sans faits) : le roster est complet (golden par build de 0.A.2
      régénéré avec justification).
- [ ] 1.6.4 `SchemaVersion` 55, chronique, empreinte de forme (0.B.4) régénérée, fixtures de
      contrat régénérées, `openapi.yaml` + `generate-types` si un champ apparaît.

Preuve : corpus gate zéro perte ; gains nommés (`identity.coverage.*.direct`, `unnamedLives` ne
monte nulle part) ; `replay-equiv` différences localisées aux balayages d'identité.

#### Lot 1.7 — L'équipe réelle dans l'artefact, sans base — M, high

Sur pièces : `replay/build.go:580` (`Team: -1`), `document.go:590` (« l'équipe n'est pas dans le
film » : faux), `TeamOf` fourni par la base (`matchfacts.go:254`), `CarrierTeamUnknown`.

- [ ] 1.7.1 `filmdec.ScanPlayerTeams(fc) (map[filmIndex]designator, TeamScanReport)` : records
      d'image-clé ti=9 par la boucle d'état complet (1.4), composant i0 sur 4 bits à
      `108 + 32 + état(ti) + 32` (dérivé, jamais 186 en dur), valeur = désignateur + 1, 0 =
      aucune ; appariement entité ti=9 → joueur selon `NOTE_EQUIPE_FILM_2026-09-12.md` ; stabilité
      par entité sur le film (le rapport compte les divergences).
- [ ] 1.7.2 Règle V4 : le film est la SEULE source. `Track.Team`, `roster[].team` (champ neuf)
      et `TeamOf` des drapeaux viennent du désignateur ; 0 = aucune équipe (FFA), muet =
      inconnue ; aucun repli sur la base. La base n'entre que dans
      `coverage.teams.{film, accord, contradiction, silence}` ; une contradiction ne se corrige
      pas en silence, elle se compte et le relecteur la lit.
- [ ] 1.7.3 Les événements d'objectif (1.1.2) prennent l'équipe de l'octet 37 ; contrôle contre
      l'équipe du porteur (compteur).
- [ ] 1.7.4 Commentaires `document.go:590` et `flag_assign.go:31` corrigés.
- [ ] 1.7.5 `SchemaVersion` 56, chronique, forme, fixtures, goldens par build (attendu :
      `Track.Team != -1` hors ligne, S5).

Preuve : corpus gate zéro perte, gains nommés (`teams.accord` = 160/176 et 24/24 sur les BTB des
notes, `CarrierTeamUnknown` inchangé ou en baisse) ; FFA : `teams.film = 0`, base conservée.

#### Lot 1.8 — Le kill feed prend la table du film — M, high

- [ ] 1.8.1 Sur pièces : la voie d'inférence des index de joueur de killsource / killcollector
      (`resolvePlayerIndices` ou équivalent, `roster.go`, `identities.go`) ; la table de 1.5
      devient la source, l'inférence le repli (5 films).
- [ ] 1.8.2 `KillSourceDecoderRev` montée ; `TestKillSourceDecoderRevSuitLeDecodeur` vert ;
      compteurs de provenance dans les stats de collecte.
- [ ] 1.8.3 Backlog killsource du parc : **signal utilisateur, hors lot** (D6).

Preuve : corpus gate zéro perte sur les axes kills / morts / sources ; `replay-equiv` différences
localisées.

**Clôture M1** : fusion dans `feat/v75` (V3) ; recuisson du parc + backlog killsource sur signal
(tag git du binaire précédent, artefacts précédents conservés jusqu'à validation du corpus gate,
architecture §10) ; corpus d'équivalence re-figé UNE fois (`-update` sur tout le corpus,
consigné) : c'est l'oracle de M2.

---

### M2 — La révision à zéro différence (pas 1 à 7)

Critère d'entrée (mesuré) : M1 fusionné ; `git branch --no-merged feat/v75` filtré par
`git diff --name-only` sur `internal/games/halo_infinite/film/` et `internal/replaybuild/` =
vide ; corpus d'équivalence re-figé. Pendant M2 : aucun autre lot filmdec, aucun port de
composant, un seul muteur. Chaque lot : `replay-equiv` = ZÉRO différence (D4), corpus gate zéro
différence, aucune montée de `SchemaVersion` ; `GrammarRev` montée par lot (la source change).

#### Lot 2.1 (pas 1) — Le profil, résolu une fois, encore recopié — M, high

- [ ] 2.1.1 `filmdec.Profile` immuable : `Identity` (1.5.1), `Map MapQuantEntry`, `Highlight`
      (implantation du gamertag par version : jusqu'à 38, 39-40, dès 41), `Keyframe` (règle
      `172 + état(ti)`, jamais un nombre), `Movement` (quantums, largeur d'axe absolue, drapeaux de
      queue), `Slots` (1.5.2). Table `profile_table.go` : une ligne par build (B.1 de
      `NOTE_PROFIL_PAR_BUILD` + 5b : 7 builds), chaque valeur avec fonction Ghidra ou film témoin
      et date (D3) ; entrées « présumées » listées par `TestProfilPresumes`.
- [ ] 2.1.2 `NewFilmContextForMap` résout le profil à la construction (D1) ; `ErrUnknownBuild`
      remonte à l'appelant (le constructeur le journalise, principe 12).
- [ ] 2.1.3 Double écriture : `installWorldObjectPrecision` lit le profil et écrit encore les
      globales ; kill-switch daté dans le code (bascule = date du lot, retrait cible = lot 2.3,
      critère = 0 globale) ; `TestProfilEgaleGlobales`.
- [ ] 2.1.4 Les trois appels `ParseHighlightEvents(data, version)` lisent `Profile.Highlight`.

Preuve : `replay-equiv` zéro différence ; corpus gate zéro différence.

#### Lot 2.2 (pas 2) — Les lecteurs reçoivent le profil, famille par famille — L, high

Un commit par famille, gate complet à chaque famille, ratchet `filmdecVarsGeles` descendu à
chaque famille (96 → …), mutation obligatoire par valeur migrée (« fausser la valeur rougit un
test nommé »). Le profil se passe par pointeur ou se lit en tête de balayage, jamais dans la
boucle de bits (budget 0.A.5).

- [ ] 2.2.a Positions (`TraversalPrecision`, `PositionFullPrecision`, `PositionDeltaHasHandleTail`,
      `PositionCalibratedSkip`, `absoluteAxisW`).
- [ ] 2.2.b Objets du monde (`WorldObjectPrecision`, `WorldPositionRange`, `DeltaQuantum`,
      `DeltaAxisWidth`).
- [ ] 2.2.c Images-clés (`KeyframeBodyVariants`, largeurs d'en-tête par type → `Profile.Keyframe`).
- [ ] 2.2.d Temps forts et pied (implantation du gamertag, version).
- [ ] 2.2.e Équipement et mobilité (`MobilityActionExtraBits`, tables de grammaire déguisées en
      `var`).
- [ ] 2.2.f Les crochets d'observation (`Set*Hook`, `unitRefHook`, table des largeurs de
      `frame_chain_infer.go`) deviennent un `Observer` passé en paramètre (`nil` en production) ;
      la table sans verrou devient un champ de l'observateur.

Preuve par famille : `replay-equiv` zéro différence ; ratchet descendu ; mutation rouge.

#### Lot 2.3 (pas 3) — Plus de globale, plus de verrou — M, high

- [ ] 2.3.1 Suppression des globales restantes, des `install` / `restore`, de la double écriture
      (kill-switch retiré à sa date cible), de `LockProcessDecode` et de `decode_gate.go`.
- [ ] 2.3.2 Ratchets : `filmdec_package_vars_test.go` gelé à 0 variable mutable (les `const`
      restent) ; `decode_lock_held_test.go` INVERSÉ (toute prise de verrou est interdite) ;
      `no_recomputed_film_context_test.go` inchangé.
- [ ] 2.3.3 `TestDeuxFilmsEnParallele` : deux mini-films de builds différents décodés dans deux
      goroutines, sorties identiques aux décodages séquentiels ; exécuté sous
      `go test -race` (S2).
- [ ] 2.3.4 Les appelants qui prenaient le verrou (`killcollector/positions.go`,
      `replay/build_from_film.go`, `cmd/`) en sont délestés.

Preuve : `replay-equiv` zéro différence ; `-race` propre ; corpus gate zéro différence.

#### Lot 2.4 (pas 4) — Une seule porte aux octets — M, high

- [ ] 2.4.1 `killsource.evReader` absorbé par `filmdec.BitReader` (même consommation de bits,
      test d'équivalence bit à bit sur les chaînes d'événements des mini-films).
- [ ] 2.4.2 Façade unique de lecture brute (`film.Source` : chunks, paquets, lecteur de bits) ;
      `killsource/{chunks,feed,walk,world}.go`, `weaponv3/pi_resolver.go`,
      `filmcache/filmcache.go`, `objectiveevents/film.go` la traversent.
- [ ] 2.4.3 Ratchet `archlint/no_raw_film_bytes_outside_source_test.go`, allowlist VIDE.

Preuve : `replay-equiv` zéro différence ; `TestKillSourceDecoderRevSuitLeDecodeur` (révision
montée, sortie identique) ; corpus gate zéro différence.

#### Lot 2.5 (pas 5) — Les cinq couches, par déplacements purs — L, high

Méthode du lot E : ratchets de dépendance posés AVANT le premier `git mv` ; `git mv` sans ajout
ni suppression de ligne (hors `package` et imports) ; une couche par commit ; gate à chaque
couche. Critère d'entrée re-mesuré (zéro branche filmdec en vol).

- [ ] 2.5.0 Ratchet `archlint/film_layers_deps_test.go` : `source` n'importe rien du décodeur ;
      `profile` importe `source` et le catalogue ; `grammar` importe `source`, `profile` ; `facts`
      importe `grammar` ; `replay` importe `facts` ; jamais l'inverse ; personne hors `source` ne
      lit `chunk[i][j]`.
- [ ] 2.5.a `film/internal/source` ← `analysis/filmsource` (V5) + lecture des quatre sections de
      `chunk_00` (1.5.1) ; `Film` porte `Identity`.
- [ ] 2.5.b `film/internal/profile` ← `Profile`, `profile_table.go`, `MapQuantCatalog`,
      `player_table_profile.go`.
- [ ] 2.5.c `film/internal/grammar` ← `filmdec` (lecteurs, `FilmContext`, inférence de chaînes).
- [ ] 2.5.d `film/internal/facts` ← `killsource`, `analysis/objectiveevents`, registre d'identité,
      équipement, véhicules, projectiles, grenades (inventaire sur pièces, consigné).
- [ ] 2.5.e Façades exportées pour les consommateurs hors `film/` (`sync/killcollector`,
      `replaybuild`, `service`, `ops`, `haloclient`, `ingest`, 7 outils `cmd/`) : compilation seule
      garantit la frontière (principe 11) ; allowlist `no_title_package_in_analysis_test.go`
      vidée (D9) ; `no_film_reread_test`, `decode_lock_held_test` (inversé), `gamefiles_tag_test`
      re-pointés.

Preuve par couche : `go build ./...` ; `replay-equiv` zéro différence ; corpus gate zéro
différence ; `git diff --stat -M` ne montre que des renommages.

#### Lot 2.6 (pas 6) — Empreintes par couche et types de contrat — M, high

- [ ] 2.6.1 `grammar.Rev`, `facts.Rev` (héritière de `KillSourceDecoderRev` pour le backlog) ;
      empreinte par couche ; règle « montée de `facts.Rev` = backlog killsource » écrite dans le
      test et dans `docs/SYNC_GUIDE` (FR + EN).
- [ ] 2.6.2 Paquet de types de sortie sans dépendance (`film/internal/types`, à la manière de
      `games/canonical`) produits par `grammar` et `facts`, consommés par `facts` et `replay` ; un
      test de contrat par type (forme figée).
- [ ] 2.6.3 L'artefact publie `coverage.decoder.{grammarRev, factsRev, build}` (ajout de champ →
      montée de `SchemaVersion` 57 par l'empreinte de forme ; contenu cuit inchangé par ailleurs).

Preuve : une mutation de source rougit l'empreinte de sa couche ; `replay-equiv` différence
limitée au champ `coverage.decoder`.

#### Lot 2.7 (pas 7) — Scission des fichiers de plus de 500 lignes — M, high

- [ ] 2.7.1 `traverse.go` (1 380), `unit_weaponstate.go` (970), `frame_records.go` (793),
      `components_biped_ability.go` (699), `components_movement.go` (554) côté grammaire ;
      `document.go` (624), `build.go` (607), `equipment_placements.go` (594), `lives.go` (565)
      côté publication ; `document_chronicle.go` (1 230) est une chronique : exemption écrite en
      tête, pas de scission.
- [ ] 2.7.2 Ratchet de taille : plafond gelé par fichier, jamais accru
      (`archlint/film_file_size_test.go`).
- [ ] 2.7.3 `benchstat` contre `bench_baseline.txt` : +10 % au plus.

Preuve : `replay-equiv` zéro différence ; baseline lint non accrue.

**Clôture M2** : ADR 0034 amendé (état atteint) ; fusion dans `feat/v75` (V3) ; recuisson sur
signal (seul 2.6.3 change un champ) ; `bench_baseline.txt` re-figé et consigné.

---

### M3 — Exploiter le profil (pas 8) : les divergences par build du lot H

Critère d'entrée : M2 fusionné. Ordre du rapport H (P1, P2, P4, P5 ; P3 est couvert par 1.5 et
1.6), puis les ports de composants (3.6, V1). Chaque lot change le contenu cuit : `SchemaVersion`, corpus gate avec gains attendus et
zéro perte, recuisson à la clôture du jalon sur signal. Retour arrière : les pas 1 à 7 se
défont par `git revert` ; avant la recuisson, tag git du binaire précédent et artefacts conservés.

#### Lot 3.1 — Build inconnu actif, profil comme donnée fabriquée — M, high

- [ ] 3.1.1 Politique active en production : `ErrUnknownBuild` → film mis de côté par
      `killcollector` et `replaybuild` (journal `slog.WarnContext` + expvar par build), jamais un
      décodage au profil précédent (S7) ; test unitaire nommé.
- [ ] 3.1.2 Données de profil dans `data/titles/halo_infinite/reference/film_profiles.json`
      (PathResolver, D12) : ce qui se dérive des fichiers du jeu (bornes de carte, déjà par
      `cmd/mapquant-build`) est produit par l'outil ; ce qui vient de l'exe (largeurs d'état,
      implantations) est une entrée de données avec provenance Ghidra (fonction, date), statut
      `présumé` / `prouvé` ; test `gamefiles` « catalogue commis = catalogue régénéré » pour la
      part fabriquée ; ratchet `no_runtime_versioned_catalog_write_test` étendu.
- [ ] 3.1.3 Procédure d'ajout d'un build (`docs/RUNBOOK`, EN) : outil, témoin au corpus, entrée
      présumée puis prouvée.

#### Lot 3.2 (P1) — Le registre par build — M, high

- [ ] 3.2.1 Table des empreintes de registre connues PAR BUILD (8 empreintes mesurées sur 1 351
      films) dans le profil ; `warnUnknownRegistry` devient une classification typée (connue /
      présumée / inconnue) publiée dans `coverage.decoder`.
- [ ] 3.2.2 Audit des 17 sites qui adressent un composant par index littéral (`indicesOf`,
      `component(i)`) : par NOM, jamais par rang ; ratchet grep.
- [ ] 3.2.3 Témoin par build au corpus gate (déjà fait pour 4 ; compléter à 7).

#### Lot 3.3 (P2) — La liste blanche des grenades par build — M, high

- [ ] 3.3.1 `GrenadeTypeIDsByRank` (`grenade_events.go:95`) devient une entrée de profil par
      build ; identifiants des builds antérieurs à `HI_1_12_0` établis par la méthode d'origine
      (stabilité par rang sur le corpus, marqueur `0x4C0C00`), statut `présumé` jusqu'au témoin.
- [ ] 3.3.2 Gain attendu au corpus gate : lancers > 0 sur `111fa685`, `e5adf7b2`, `60ae07c4`
      (5 000 à 10 000 lancers sur 82 films) ; zéro perte.

#### Lot 3.4 (P4) — La marche des morts calibrée par la carte et le build — M, high

- [ ] 3.4.1 `axisW`, `indexW` lus dans le profil (carte + build) au lieu du « profil plat »
      deviné ; l'inférence reste oracle de test (D2).
- [ ] 3.4.2 `KillSourceDecoderRev` / `facts.Rev` montée ; gain attendu : morts crédibles de
      4-11 % à 28-64 % sur les 7 films v31-37 ; zéro perte ; backlog sur signal.

#### Lot 3.5 (P5) — La bande de slots bipède par build — S (instruction), high

- [ ] 3.5.1 Instruction bornée (une session) : cause de `[512, 7808]` / `[512, 8064]` ; si la
      cause est une donnée de build, entrée de profil et correctif dans le lot ; sinon ligne au
      registre des reports avec condition de reprise.

#### Lot 3.6 — Les composants manquants, archétype par archétype (V1) — L, high, répétable

Méthode du handoff §4 bis, dans la structure révisée : chaque composant porté est une fonction
pure (profil, bits) dans `grammar`, une largeur dépendante d'une config du jeu devient une
entrée de profil, jamais une constante. Un LOT = un archétype (ou le bloc de composants partagés
qui le débloque), une session ; le lot suivant ne s'ouvre qu'à la clôture du précédent.

- [ ] 3.6.0 Dimensionnement sur pièces : l'inventaire de 0.A.3 (composants ordonnés, statut
      porté / manquant / BLOQUANT par archétype) donne le compte exact de composants à porter pour
      les archétypes utiles, consigné ici avant le premier port. Ordre : ti=9 joueur (un
      bloquant, `i4 managed-player-forge-weather-effect-overrides-component`), ti=11 et ti=12
      objectifs, ti=35 bipède (52 composants, majorité portée), ti=42 et ti=43 armes, véhicules.
- [ ] 3.6.a ti=9 ferme à 100 % sur les mini-films et sur les 6 films de recherche.
- [ ] 3.6.b ti=11 et ti=12 ferment à 100 %.
- [ ] 3.6.c ti=35 ferme à 100 %.
- [ ] 3.6.d ti=42 et ti=43 ferment à 100 %.
- [ ] 3.6.e véhicules (ti=40 et voisins) ferment à 100 %.

Par composant bloquant : descripteur trouvé par dump de `.rdata` (`0x143606000..0x144395200`),
écrivain `vtable+0x18` décompilé (Ghidra lecture seule), port, preuve = fermeture de l'archétype
avant / après (oracle `imagecle_fermeture`, mot `n2` pour l'état par défaut, dent de scie en
seconde chaîne pour une largeur constante). Ce qui est interdit : re-mesurer par statistique ce
qu'un écrivain lisible dit en clair. Gate par lot : ratchet 0.A.3 régénéré avec justification
(la couverture monte, jamais ne descend), `grammar.Rev` montée, régime court ; `SchemaVersion`
seulement si un consommateur publie un contenu neuf.

Préparation facultative, sans production : quand le second slot d'agent est libre entre deux
revues (M1, M2), un agent Opus peut décompiler les écrivains des composants bloquants en
INSTRUMENTS et notes (`film_re/`), sans toucher un fichier de production ; 3.6 devient alors un
port de grammaires déjà relevées. Jamais un troisième agent.

**Clôture M3** : fusion (V3) ; recuisson + backlog sur signal ; corpus gate final consigné.

---

### M4 — La publication (architecture §11) : les faits persistés, une révision par calque

Critère d'entrée : 2.5 fusionné (la frontière `facts` existe) ; fenêtre propre ; corpus
d'équivalence propre à ce jalon (oracle = « document rejoué depuis les faits ≡ document cuit »).

#### Lot 4.1 — Les faits persistés par film avec leur révision — L, high

- [ ] 4.1.1 Sérialisation des faits (généralisation de `inputs_*.bin.gz`) dans
      `data/cache/film_facts/{slug}/{short8}.facts.bin` via `PathResolver`, en-tête
      `{grammarRev, factsRev, build, schéma de faits}`.
- [ ] 4.1.2 `replaybuild` rejoue depuis les faits quand ils existent à la révision courante,
      décode sinon ; verrou solo respecté ; aucun second puits d'artefact
      (`no_second_artifact_sink_test`).
- [ ] 4.1.3 Test S8 : sur le corpus, document depuis les faits ≡ document depuis le film, à
      l'octet ; durée consignée (attendu : secondes contre 15 s).

#### Lot 4.2 — La révision par calque portée par le document — M, high

- [ ] 4.2.1 `layers: {nom: révision}` dans le document ; la présence d'un calque se lit dans sa
      révision, jamais dans l'absence d'un champ ; empreinte de forme (0.B.4) et chronique.
- [ ] 4.2.2 `normalizeReplayDocument` lit les révisions de calque (seul point web touché, D11) ;
      fixtures de contrat régénérées ; matrice de compatibilité (0.B.2) étendue aux calques.

#### Lot 4.3 — Un seul type publié — M, high

- [ ] 4.3.1 Le type publié = `domain/replaydoc` enrichi (`schemaVersion`, `layers`) ; `film/replay`
      le produit directement ; la conversion jumelle supprimée (0 code mort) ; l'en-tête
      `X-Replay-Latest-Schema-Version` ne porte que la version du producteur.
- [ ] 4.3.2 `openapi.yaml`, `make generate-types`, `make openapi-check`, contrats 0.B verts.

#### Lot 4.4 — La recuisson sélective par couche — M, high

- [ ] 4.4.1 `replaybuild.Digest` porte les révisions de couche ; « à recuire » se décide par
      couche ; un changement de publication ne redécode pas ; un changement de grammaire ne
      recuit que ce qui en dépend.
- [ ] 4.4.2 Badge admin : état par couche (chaîne FR + EN).

**Clôture M4** : ADR 0034 amendé ; fusion (V3) ; recuisson sur signal ; §5 complet ; critères S1
à S8 re-vérifiés et consignés.

---

## 4. Découvertes (consignées, NON traitées — règle 7)

| Date | Lot | Découverte | Où elle ira |
|---|---|---|---|
| 2026-09-13 | 0.B | **D1 — `writeArtifactBytes` ne refuse PAS une rétrogradation de VERSION.** L'architecture §12 point 5 affirme que « le point d'écriture unique refuse toute rétrogradation » ; sur pièces il ne refuse que l'APPAUVRISSEMENT à schéma ÉGAL (`wouldDowngrade`, `artifact_store.go:88-98`), et à schéma DIFFÉRENT il se tait délibérément (`artifact_store.go:85-87`, verrouillé par `TestWriteArtifact_MonteeDeSchemaToujoursEcrite`). Le refus de version vit en amont, dans `validateArtifact` (`artifact_store.go:112`), sur le seul écrivain qui puisse porter une autre version (`StoreArtifact`, dépôt d'ouvrier) ; les trois autres sérialisent un document du producteur courant. Le résultat est correct, la phrase du document ne l'est pas. NON TRAITÉ. | ADR 0034 (lot 0.C.1) : y écrire où vit chaque refus ; corriger la phrase de l'architecture §12 |
| 2026-09-13 | 0.B | **D2 — la chronique n'a pas une forme d'en-tête mais TROIS**, nées à des mois différents : `// v<N> (` (v2-v20, v40-v50, v52-v54), `// SCHEMA <N> (` (v29), `// CE QUE LA VERSION <N> ` (v21-v39 pour l'essentiel). Et deux numéros n'existent pas : **32 et 51 ont été SAUTÉS** à la renumérotation de deux lots parallèles (écrit dans l'entrée v33) ; la v1 est antérieure à la chronique. Tout garde-rail qui dériverait la liste d'un intervalle `1..N` affirmerait des versions jamais cuites. NON TRAITÉ (l'extracteur `testutil.ReplayChronicleVersions` reconnaît les trois formes et ne comble aucun trou). | ADR 0034 (lot 0.C.1) : forme normative d'une entrée de chronique |
| 2026-09-13 | 0.B | **D3 — deux écarts entre le document STOCKÉ et son jumeau SERVI**, tous deux invisibles sur le fil JSON et donc légitimes, mais qui ont dû être neutralisés explicitement pour que l'empreinte de forme coïncide : (a) `replay.Coverage` et `replaydoc.Coverage` déclarent les mêmes champs dans un ORDRE différent ; (b) `IdentityLink.Method` est de type nommé `LinkMethod` côté stocké, `string` côté servi. NON TRAITÉ. | rien à traiter ; la neutralisation est écrite dans `document_shape_test.go` |
| 2026-09-13 | 0.B (revue R1) | **D5 — la borne `MIN_RENDERABLE_SCHEMA_VERSION` se vérifie sur de la PROSE.** Le relecteur a confirmé sur pièces qu'aucune montée > 27 ne retire ni ne renomme un champ lu par le web, mais cette vérification se fait en relisant les entrées de `document_chronicle.go` une à une : le dépôt n'a **aucun inventaire de champs par version** (rien qui dise « la version N portait ces 54 clés »). Tant qu'il n'en a pas, la borne reste une affirmation relue, jamais calculée — et la relire coûtera une demi-heure à chaque fois qu'on voudra la bouger. NON TRAITÉ. | ADR 0034 (lot 0.C.1) ou lot ultérieur : un golden de clés par version, dérivable de l'empreinte de forme (0.B.4) si elle était figée à chaque montée |
| 2026-09-13 | 0.B (revue R1) | **D6 — `internal/domain/replaydoc` ne porte AUCUN test.** Le paquet du document SERVI est une feuille sans fichier `_test.go` : l'empreinte de sa forme vit dans `games/.../replay` (0.B.4) et sa parité champ par champ dans `service/replayview`. Le jumeau n'est donc gardé que depuis l'extérieur — ce qui suffit tant que ces deux gardes existent, mais laisse le paquet sans filet propre si l'un d'eux déménage. NON TRAITÉ. | ADR 0034 (lot 0.C.1) : nommer où vit la garde de chaque paquet |
| 2026-09-13 | 0.B (revue R2) | **D7 — le ratchet de version ne voit pas la forme INDIRECTE.** `testDoc.guard.test.ts` attrape `schemaVersion: 7` mais pas `const X = 7` suivi de `schemaVersion: X` — forme employée aujourd'hui par `lib/replay/replayDocumentSchema.test.ts` (`VERSION_QUELCONQUE`), et qui n'y figure sous AUCUNE exemption nommée. Ce n'est pas un faux vert dans ce cas précis (la valeur y est délibérément quelconque, le test ne décrit que la nature des champs), mais le garde ne le sait pas : il l'ignore par angle mort, pas par décision. NON TRAITÉ. Condition de reprise : le jour où un second test l'emploie. Alternative : une exemption nommée dans `ALLOWED`, ce qui rendrait l'angle mort explicite sans écrire d'analyseur. | lot ultérieur du chantier, ou 0.C si la liste des reports s'y ouvre |
| 2026-09-13 | 0.B (revue R2) | **D8 — le contrat strict s'arrête à la racine.** `replayDocumentSchema.ts` pose `strictObject` sur la racine et sur `bounds` seulement : une clé inconnue DANS un élément de calque, dans `coverage`, `identity` ou une table de libellés traverse — `validateReplayDocument` rend `null` et le badge dit « à jour ». L'écart est ASSUMÉ et documenté en tête du module (avec renvoi au garde Go de forme, qui lui descend à toute profondeur), parce qu'un contrat strict imbriqué exigerait de modéliser chaque calque en zod à la main — c'est-à-dire une seconde description du document, à tenir en phase avec la première. NON TRAITÉ. Condition de reprise : M4, où le document ne sera plus publié qu'en UN type — le schéma pourra alors être DÉRIVÉ au lieu d'être réécrit. | M4 (publication : faits persistés, une révision par calque) |
| 2026-09-13 | 0.B | **D4 — budget de taille pour 0.B.7.** Le document du film de référence pèse 3 318 459 o indenté, **401 589 o compressé**. À sept mini-films (V7), le jeu de fixtures pèserait ~2,8 Mio compressés si chaque build produit un document de cette taille — à mesurer réellement au lot 0.B.7, les mini-films de 0.A.2 étant plus courts que le film de référence. | lot 0.B.7 (taille totale bornée et consignée) |

## 5. Journal des gates locaux (un gate non consigné n'a pas eu lieu)

| Date | Lot | Commit | Commande | Résultat (compte, empreinte, durée) |
|---|---|---|---|---|
| 2026-09-13 | 0.B | `df2d90c35` | `gofmt -l ./internal ./cmd` | sortie vide |
| 2026-09-13 | 0.B | `df2d90c35` | `go vet ./internal/games/halo_infinite/film/replay/ ./internal/replaybuild/ ./internal/domain/replaydoc/ ./internal/archlint/ ./internal/testutil/` | 0 diagnostic |
| 2026-09-13 | 0.B | `df2d90c35` | `go test ./internal/games/halo_infinite/film/replay/ ./internal/replaybuild/ ./internal/domain/replaydoc/ ./internal/archlint/ ./internal/testutil/` | ok — replay 7,76 s ; replaybuild 0,93 s ; archlint 10,92 s ; testutil 0,17 s ; replaydoc « no test files » |
| 2026-09-13 | 0.B | `df2d90c35` | `go test ./internal/service/replayview/ ./contracttest/` (contrôle ajouté : parité des jumeaux + compte de champs figé) | ok — 0,20 s / 0,36 s |
| 2026-09-13 | 0.B | `df2d90c35` | `go test ./...` (suite complète, `delivery-checklist` §1 ; code de sortie relevé hors tube — piège du pipe) | **code de sortie 0**, 323 paquets, zéro `FAIL`. `-tags=integration` NON joué : le diff ne touche ni `persist/`, ni `sync/`, ni `migration/` |
| 2026-09-13 | 0.B | `df2d90c35` | `make go-api-lint` | 0 issue (2 `govet inline` sur `reflect.Ptr` corrigés en `reflect.Pointer` avant de reverdir ; baseline non accrue) |
| 2026-09-13 | 0.B | `df2d90c35` | `make check-types` puis `cd apps/web && npx tsc -b --force` | 0 erreur dans les deux cas (le `--force` neutralise le cache `.tsbuildinfo`) |
| 2026-09-13 | 0.B | `df2d90c35` | `make test-web` | 700 fichiers passés + 1 sauté ; 7 417 tests passés + 17 sautés ; 98,05 s |
| 2026-09-13 | 0.B | `df2d90c35` | `make openapi-check` | `api/openapi.yaml` à jour (712 826 o) ; `generated.ts` dérive bien — aucun champ neuf au contrat public, ce lot ne touche pas la forme servie |
| 2026-09-13 | 0.B | `df2d90c35` | `npx eslint` (web touché) + `npm run lint:colors` + `npm run lint:fields` | 0 erreur (9 avertissements PRÉEXISTANTS, aucun dans les fichiers du lot) ; couleurs propres ; libellés propres |
| 2026-09-13 | 0.B.1 | `277f9e177` | `REPLAY_CONTRACT_UPDATE=1 go test ./internal/games/halo_infinite/film/replay/ -run ContractFixtures -update` | fixture écrite : 3 318 459 o de JSON, **401 589 o compressés** ; puis comparaison verte sans `-update` |
| 2026-09-13 | 0.B.4 | `765fccd4a` | ÉPREUVE PAR MUTATION (champ `SondeMutation` ajouté à `ReplayDocument`, puis annulé) | golden de forme ROUGE (`6581c32e42b003aa` -> `cffde9839c41a928`), jumeaux ROUGES, **régénération REFUSÉE** (« SchemaVersion est resté à 54 ») ; `git diff` vide après annulation |
| 2026-09-13 | 0.B.5 | `76e15eb9c` | `go test ./internal/replaybuild/ -run 'SchemaAnterieur\|DepotRefuse\|AppauvrissementAChaqueSchema' -v` | 51 sous-tests par version (v2 à v53, moins 32 et 51), 3 familles de preuve, tous verts |
| 2026-09-13 | 0.B.6 | `94f906b71` | `npx vitest run src/features/match-replay/test/testDoc.guard.test.ts` | 5 tests verts, dont le contre-test d'inertie (3 fautifs détectés, 3 formes légitimes ignorées) |
| 2026-09-13 | 0.B revue R1 | `8b9bc339f` | **Revue adversariale ronde 1** | **6 constats recevables, 0 jeté, les 6 corrigés DANS le lot** (ils portent tous sur les garde-rails que le lot pose). 17 conditions vérifiées tiennent. 5 mutations jouées par le relecteur, **3 traversaient** (clé renommée acceptée par `z.object` ; copie `Equals` sous d'autres noms de paramètres ; borne 27 -> 50 sans un test rouge) : les 3 rejouées ROUGES après correction, plus les deux formes d'écriture manquantes du ratchet de version |
| 2026-09-13 | 0.B revue R1 | `8b9bc339f` | Mutations rejouées : `shots` -> `shotz` sur la fixture Go ; copie `Equals<L, R>` dans `bombCountdown.test.ts` ; `computeReplaySchemaStatus(48, 51)` et `"schemaVersion": 3` ; `heatPaint.test.ts` remis à 1 ; `MIN_RENDERABLE_SCHEMA_VERSION` 27 -> 50 | **toutes ROUGES** ; sans le mode strict, le test de renommage rend « expected null not to be null » — arbres restaurés (`git diff` vide) |
| 2026-09-13 | 0.B revue R1 | `8b9bc339f` | `npx tsc -b --force` ; `npx vitest run` (complet) ; `gofmt -l` ; `go test ./internal/games/halo_infinite/film/replay/ ./internal/replaybuild/ ./internal/archlint/ ./internal/testutil/` ; `make go-api-lint` ; `make openapi-check` | tsc **exit 0** ; vitest **exit 0**, 700 fichiers + 1 sauté, **7 425 tests** + 17 sautés (8 de plus qu'avant la revue) ; gofmt vide ; go test **exit 0** ; lint **0 issue** ; openapi à jour |
| 2026-09-13 | 0.B revue R2 | ce commit | **Revue adversariale ronde 2** (sur les corrections de la R1 seulement) | **1 P1 + 3 P2 recevables, 0 jeté ; 2 corrigés dans le lot (R2-1, R2-2), 2 consignés en §4 (D7, D8)**. 18 conditions vérifiées tiennent ; 17 mutations jouées, 15 conformes. Les P0+P1 décroissent : 6 (R1) -> 1 (R2) |
| 2026-09-13 | 0.B revue R2 | ce commit | Mutations rejouées après correction : `type _Same<X, Y> = (<Q>() => Q extends X ? 1 : 2) …` déposée hors `lib/types/` ; fragment FR (`cle(s) inconnue(s)`) remis dans la table EN | **les deux ROUGES** — la première nomme `bombCountdown.test.ts`, la seconde fait échouer le test de locale anglaise (`Unable to find … contract violated (unknown key(s): shotz)`). Arbres restaurés (`git diff` vide sur les deux fichiers) |
| 2026-09-13 | 0.B revue R2 | ce commit | `npx tsc -b --force` ; `npx vitest run src/lib/replay src/lib/types src/features/match-replay` ; `npx eslint` (fichiers touchés) | tsc **exit 0** ; **203 fichiers passés + 1 sauté, 3 029 tests + 3 sautés** ; eslint **0 erreur** |
| 2026-09-13 | 0.C | ce commit | `go test ./internal/archlint/ -run 'Mojibake\|Todo\|TODO' -v` | 6 tests verts (`TestNoMojibakeInGoModule`, `...InTitleConfigTOML`, `TestMojibakeOf_MatchesQ3Corpus`, `TestMojibakeRE_CatchesKnownPatterns`, `..._DetectsAMutation`, `TestNoExpiredTODO`) ; 1,61 s. NOTE : le motif `Todo` du plan ne matche PAS `TestNoExpiredTODO` (casse) — d'où `TODO` ajouté à l'alternance. Lot documentaire : aucun `.go` / `.ts` / `.toml` de production touché, donc pas de gate décodeur (`replay-equiv`, corpus gate) applicable |

## 6. Protocole de reprise de session

1. Relire le skill `plan-execution`, puis la §5 et la première case non statuée de la §3.
2. `git worktree list` ; `git -C LevelUp-wt-recherche-film log --oneline -5` ; vérifier que
   l'intégration porte le dernier lot fusionné (§5).
3. Ne pas re-décider une décision D ou V validée ; une question neuve = ligne en §1.4, posée à
   l'utilisateur, jamais tranchée en silence.
4. Reprendre au lot courant : exécuteur relancé avec le brief du lot (état réel constaté sur
   pièces, pas le plan de mémoire).
