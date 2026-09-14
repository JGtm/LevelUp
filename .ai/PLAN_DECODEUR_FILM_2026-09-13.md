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
| D13 | **La grammaire prime PARTOUT, pas seulement dans `filmdec`.** Toute heuristique de production (fenêtre temporelle, seuil de distance, majorité, inférence statistique) qui décide un FAIT que le film écrit (événement nommé, record de création, composant) est remplacée par la lecture de ce que le film écrit ; l'heuristique ne survit qu'en REPLI COMPTÉ dans la couverture (accord / repli / contradiction), jamais en décision première. Le mur n'est qu'un exemple (événement 103 et panneaux contre fenêtre de 200 ms) ; la règle vaut pour chaque fait. Inventaire au lot 0.E, conversions dans la famille 1.9 | utilisateur, 2026-09-13 |
| D14 | **Un repli est NOMMÉ comme tel, compté, et RETIRÉ quand la lecture est fiable.** (a) Tout repli vit dans un registre unique du code (table `facts` : nom, fait, condition de déclenchement, date de pose, critère de retrait), et le code qui l'exécute le nomme (aucun repli anonyme au milieu d'une fonction) ; ratchet : un repli hors registre = rouge. (b) Un repli ne se déclenche que sur un diagnostic typé « le film est muet ici », JAMAIS sur un désaccord avec la lecture ni sur une lecture disponible : ordre fixe = lire d'abord, repli ensuite ; s'il se déclenche alors que la lecture existait, c'est une contradiction comptée, pas un repli. (c) Chaque fait publie `coverage.<fait>.{grammaire, repli, contradiction}` ; l'artefact dit quelle part de lui vient d'un repli. (d) Critère de retrait obligatoire (règle 11 : date de pose, cible de retrait, critère mesurable) : un repli dont le compte est à 0 sur le corpus gate à la clôture d'un jalon est SUPPRIMÉ au jalon suivant, avec ses tests ; on ne garde pas un repli « au cas où », parce qu'un repli bancal qui se déclenche à tort corrompt un fait que la lecture aurait donné juste | utilisateur, 2026-09-13 |

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

- Le cache de films (`data/cache/film_chunks`, 1 351 films, 7 builds), les manifestes
  (`data/cache/film_manifests`) et le parc (`data/cache/replays`) ne vivent que dans
  `LevelUp-go-migration`. **Le corpus gate n'auto-détecte PAS le parc sur ce poste** (corrigé le
  2026-09-13) : le `.git` commun est `C:/Users/Guillaume/Downloads/Scripts/LevelUp`, un ancêtre
  RENOMMÉ qui ne porte aucune base — la détection tombe à côté en silence. Ses deux racines se
  passent donc explicitement : `--parc-root C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration`
  et `--source-root <le worktree>`. Les instruments corpus lisent `CHUNK00_FILMS` (répertoires
  absolus `C:/...`, séparés par `;`).
- **`replay-equiv` depuis un worktree : `-repo-root <le worktree>` et DEUX jonctions**, jamais
  `LEVELUP_REPO_ROOT` vers le principal (corrigé le 2026-09-13, découverte D3 — l'ancienne
  consigne de cette section était fausse). `dossierEquivalence()`
  (`cmd/replay-equiv/main.go:150`) dérive du SEUL `-repo-root` : pointer la racine sur le
  checkout principal ferait lire ses références et ses `.facts.json`, et `-update` y
  ÉCRIRAIT. `-corpus` ne déplace QUE la liste des films, jamais les références. Poser donc,
  une fois par worktree (gitignorées, lecture seule) :

  ```
  mklink /J data\cache\film_chunks    <principal>\data\cache\film_chunks
  mklink /J data\cache\film_manifests <principal>\data\cache\film_manifests
  ```

  PIÈGE : sans `film_manifests`, `replaybuild` ne voit aucun chunk au manifeste et rend un
  `score` NUL **sans erreur** (`replaybuild/matchfacts.go:95`) — l'écart se lit alors comme
  une régression du décodeur. Les catalogues (bornes, libellés, géométrie) sont versionnés :
  ils viennent du worktree, rien à monter.
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
Tout test RENOMMÉ ou SUPPRIMÉ par un lot met à jour `.ai/baselines/tests_pre_migration.jsonl` dans le
MÊME commit (le job CI « Coverage + Baseline » échoue sur tout nom de baseline absent du run ;
leçon du lot 1.0, 2026-09-14 : un renommage de test oublié = CI rouge sur un lot vert partout ailleurs).

Lots qui touchent le décodeur ou le constructeur (tout M1, M2, M3, 4.1), deux régimes (V2) :

| Régime | Quand | Commandes |
|---|---|---|
| **Court** (à chaque lot) | clôture de tout lot décodeur | `go run ./cmd/replay-equiv -repo-root <worktree> -films <échantillon nommé en tête de CORPUS.txt>` (10 films : un par build, plus `50247b26`, `d9781168`, `fb1a1a72`) |
| **Complet** | clôture de jalon (M0 à M4) ; lots 1.2 et 2.5 ; sur demande du relecteur | `go run ./cmd/replay-equiv -repo-root <worktree>` (corpus entier, 20 films, ~16 min — le découper en sous-ensembles si l outillage borne la durée d une commande) puis `go run ./cmd/replay-corpus-gate --base=<sha de l intégration avant le lot> --parc-root C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration --source-root <worktree>` (les deux racines sont OBLIGATOIRES sur ce poste, cf. §2.2) |

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

- [x] 0.A.1 **Corpus d'équivalence par build.** `replay/testdata/equivalence/CORPUS.txt` gagne une
      colonne `version / build` (découverte 1 du lot H) et un témoin par build présent au cache
      (`HI_1_4_1`, `1_8_0`, `1_9_0`, `1_10_0`, `1_11_0`, `1_12_0`, `1_13_0`), en réutilisant les
      témoins du lot H (`111fa685`, `e5adf7b2`, `60ae07c4`, `a349fea8`) ; cible 20 films au plus.
      Les `.facts.json` neufs sont exportés par le PILOTE (base libre) et remis à l'agent.
      Preuve : `go run ./cmd/replay-equiv` rend zéro différence sur les 13 films existants ; les
      films neufs sont figés par `-films <ids> -update` avec leur `# digest-grammar` ; durée par
      film consignée en §5 (budget de référence). L'ÉCHANTILLON du régime court (V2) est
      nommé en tête de `CORPUS.txt` (`# echantillon-court:`) : un film par build (7) plus
      `60ae07c4` (région sur 2 bits), `d9781168` (pire déroulage sain), `fb1a1a72` (multi-manche).
- [x] 0.A.2 **Mini-films par build et goldens.** Un `testdata/minifilm_<short8>/` par build (V7 :
      `chunk_00` + un chunk portant des images-clés + le pied, au plus 1 Mio, `PROVENANCE.txt`
      avec build, version, carte, commande de fabrication en Go). `TestGoldenAssembly` et
      `TestGoldenInputs*` deviennent des tables sur ces mini-films : un golden d'assemblage et un
      `inputs_<short8>.bin.gz` par build. Preuve : goldens verts en CI ; `TestGoldenInputsVersionGuard`
      refuse toute régénération sans montée explicite.
- [x] 0.A.3 **Inventaire et ratchet de couverture d'image-clé par archétype** (handoff §4 bis,
      étape 1). Fonction de production `filmdec.KeyframeClosure(fc) map[ti]{closed,total,
      blocking string}` fondée sur `WalkKeyframeFullState` (108 bits, mots de taille, état par
      défaut) ; instrument corpus (`CHUNK00_FILMS`) qui imprime, par archétype, les composants
      ordonnés avec leur statut (porté / manquant / BLOQUANT) et le taux de fermeture ; ratchet CI
      sur les mini-films de 0.A.2 : « la couverture par archétype ne descend jamais » (golden
      `testdata/keyframe_closure.golden`). Preuve : golden commis ; une mutation de largeur
      (ti=6) rougit le ratchet.
- [x] 0.A.4 **Empreinte du décodeur étendue à `filmdec`.** Constante `filmdec.GrammarRev`
      (forme `grammar-AAAA-MM-JJ`) ; test miroir de `decoder_rev_fingerprint_test.go` qui hache
      tous les `.go` hors tests de `filmdec/` et de `killsource/` : une source qui change sans
      montée de `GrammarRev` rougit. L'en-tête du test écrit la règle : montée de `GrammarRev`
      obligatoire à tout changement de grammaire ; montée de `KillSourceDecoderRev` si la sortie
      de killsource peut changer (backlog) ; montée de `SchemaVersion` si le contenu cuit change.
      Preuve : golden avec historique ; une ligne changée dans `traverse.go` rougit.
- [x] 0.A.5 **Budget de temps.** `replay-equiv` publie la durée par film si ce n'est pas déjà le
      cas (vérifier sur pièces) ; `BenchmarkBitReaderReadBits`, `BenchmarkTraverseEntity`,
      `BenchmarkKeyframeClosure` (le balayage chaud équivalent — `ScanBipedPositions` ne
      s'exécute PAS sur une mini-bobine, cf. D10) sur la mini-bobine `bcb6d393` ;
      `testdata/bench_baseline.txt` produit par `go test -bench . -run ^$ -count 10` et comparé
      par `benchstat` **sur la MÉDIANE** à chaque clôture de M2. **Le budget de +10 % ne vaut que
      pour les deux bancs serrés** (`BitReaderReadBits`, `TraverseEntity`) ; `KeyframeClosure` est
      INFORMATIF (71 % d'écart intra-passe et +21 % de médiane inter-passes sans changement de
      code). Preuve : baseline commise, commande et dispersion documentées dans
      `docs/COMMANDS.md` et `docs/FR/COMMANDS.md`.

Gate 0.A : gates communs ; `go run ./cmd/replay-equiv` (0 différence sur 13, références neuves
figées) ; `go test ./internal/games/halo_infinite/film/... -run 'Golden|KeyframeClosure|GrammarRev'`.

> **STATUT 0.A au 2026-09-13 — LOT CLOS, 0.A.1 à 0.A.5 tous `[x]`.**
>
> **0.A.1** — Le gate d'entrée était FAUX avant toute modification (13/13 différents) : les
> références dataient du commit `179bd7401` où `replay.SchemaVersion` valait 34, contre 54
> aujourd'hui. Sur décision du pilote, les 20 films ont été re-figés au commit d'intégration
> `cbfdc269d`, APRÈS classification écart par écart (rapport
> `.ai/V7.5/RAPPORT_REFIGEAGE_EQUIVALENCE_2026-09-13.md`) : zéro constat orphelin sur l'oracle
> d'équivalence, puis TROIS constats à instruire trouvés par le croisement au corpus gate (D6, D7,
> D8 en §4, en attente d'arbitrage). Déterminisme prouvé : **20/20 identiques**. `CORPUS.txt` porte
> la colonne `version / build` sur les 20 lignes et l'échantillon court à 10 films.
>
> **0.A.2** — Sept mini-bobines, une par build, toutes avec `chunk_00` (941 808 à 1 046 290 octets,
> sous le plafond V7). `minifilm_000d5950` intacte. Goldens en TABLE : `inputs_<short8>.bin.gz` et
> `assembly_<short8>.golden` par build, `000d5950` entrée de la table.
>
> **0.A.3** — `filmdec.KeyframeClosure` en production, instrument corpus sous `CHUNK00_FILMS`,
> ratchet CI sur les sept bobines (golden 221 lignes). Mutation `R(6)` → `R(7)` sur ti=6 : rouge
> sur six bobines, remise en place, vert.
>
> **0.A.4** — `filmdec.GrammarRev` (const) et son empreinte sur `filmdec` + `killsource`, règle à
> trois étages écrite dans le test et le golden. Mutation dans `traverse.go` : rouge, remise, vert.
>
> **0.A.5** — `replay-equiv` imprimait DÉJÀ la durée par film (vérifié sur pièces). Trois bancs,
> baseline `-count 10` commise, `benchstat` documenté en EN et en FR. Budget +10 % restreint aux
> deux bancs serrés (`BitReaderReadBits`, `TraverseEntity`) ; `KeyframeClosure` INFORMATIF.
>
> Gate de fin : `replay-equiv` **10/10 identiques** sur l'échantillon court après tous les
> changements — aucun comportement modifié, comme le lot l'exigeait.

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

- [x] 0.B.7 (après fusion de 0.A) **Une fixture par build** : `contract_fixtures_test.go` itère
      les mini-films de 0.A.2 ; taille totale bornée (consignée). — S, exécuteur du lot 0.C ou
      relecteur de 0.B.
      **Fait** (2026-09-13) : la table des fixtures est DÉRIVÉE de `goldenBuilds()` (8 entrées,
      7 builds — HI_1_13_0 en a deux, `000d5950` et `fb1a1a72`) et non recopiée : un build
      ajouté au golden d'assemblage publie sa fixture au web sans autre geste. Chaque document
      est cuit depuis son `testdata/inputs_<short8>.bin.gz` figé, par le chemin d'assemblage du
      golden du build — aucun film lu. Fichier `replay_schema_54_<short8>.json.gz` ;
      `manifest.json` à entrées (fichier, film, build, version, taille, `pointsStride`). Côté
      web, `goFixtures.contract.test.ts` (8 × 10 cas) et la matrice de compatibilité (8 × 4)
      itèrent le manifeste sans jamais ouvrir un document pour la seconde ; `testDoc.ts` y prend
      toujours sa version et `testDoc.guard.test.ts` reste vert. La porte de régénération garde
      sa double condition et sort en ÉCHEC en nommant ce qu'elle a réécrit (motif 0.A/R2-C1).
      **TAILLE : 2 108 044 o = 2,01 Mio pour 8 fixtures, sous le plafond de 3 Mio** —
      `000d5950` 401 589 · `a521164d` 191 488 · `60ae07c4` 172 786 · `11de8353` 337 201 ·
      `111fa685` 372 485 · `e5adf7b2` 351 316 · `bcb6d393` 80 469 · `fb1a1a72` 200 710.
      **Coupe A, arbitrée par le pilote** : les documents PLEINS pesaient 6 142 690 o
      (5,86 Mio) — D4 estimait ~2,8 Mio en supposant des mini-bobines, or 0.A.2 décode les
      entrées du FILM ENTIER. `tracks[].points` faisant 85 à 89 % des octets, les 7 fixtures par
      build ne gardent qu'un point sur cinq (premier et dernier toujours gardés : la fenêtre de
      vie et la première position ne bougent pas) ; `000d5950` reste INTACTE, identique octet
      pour octet à la fixture d'avant le lot (`cmp`). Le taux est déclaré par fixture
      (`pointsStride`), jamais déduit : ces fixtures servent le contrat de FORME, pas la
      reproduction du document servi.
      **UN SEUL JEU VIVANT** (même arbitrage) : la régénération écrit la version courante et
      SUPPRIME la précédente (l'historique git garde le reste) — sans quoi chaque montée de
      schéma déposerait 2 Mio de plus dans l'arbre. Tenu hors régénération par
      `TestContractFixturesUnSeulJeuVivant`, éprouvé par mutation (fixture `replay_schema_53_*`
      déposée → rouge, retirée → vert). Vérifié sur pièces : aucune fixture d'une version
      ancienne n'est nécessaire — seul `goFixtures.contract.test.ts` ouvre un document, et la
      matrice comme le badge FORCENT leurs versions en argument
      (`MIN_RENDERABLE_SCHEMA_VERSION ± 1`, version du manifeste).
      Le fichier passant 500 lignes, le plafond et l'inventaire sont sortis dans
      `contract_fixtures_budget_test.go` (433 + 133 lignes).

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

#### Lot 0.D — Synchronisation avec `feat/v75` et instruction de D6 à D9 — M, exécuteur Opus high

Créé le 2026-09-13, APRÈS la fusion de `origin/feat/v75` dans l'intégration du chantier
(commit `215649efd`, 42 commits de finitions). Trois raisons, toutes constatées sur pièces :
F.1 (`c45c411eb`, « l'origine d'une pose d'equipement devient purement temporelle ») change le
CONTENU CUIT sans monter `SchemaVersion`, ce qui périme les goldens d'assemblage, les fixtures
de contrat et les références d'équivalence figées à `cbfdc269d` ; les trois constats du
croisement au corpus gate (D6, D7, D8 de la §4) attendent un verdict ; et D9 dit que le codec
des entrées de golden ne transporte pas tout ce que l'assemblage lit. Sous-lots strictement
séquentiels : un sous-lot = des commits `0.D.<n>`, un gate, une ligne en §5, la main rendue.
ORDRE D'EXÉCUTION (révisé le 2026-09-13 au soir) : 0.D.0, 0.D.1, 0.D.2 sont clos ; viennent
ensuite **0.D.5** (synchronisation de la seconde fusion, avant tout le reste — les oracles
doivent être à jour), puis **0.D.1 bis**, puis 0.D.3 et 0.D.4.

- [x] 0.D.0 **Synchronisation avec `feat/v75` : attribution, régénération, re-figeage.**
      Régime court (10 films) sur `215649efd` : lister, film par film, les étapes qui diffèrent
      des références, et attribuer chaque étape à un commit de `feat/v75`
      (`git log d5f41d2a2..215649efd -- <chemins>`, sha et phrase citées). Attendu : seulement
      des étapes liées à l'équipement (F.1) ; toute autre étape qui bouge est un constat, listé
      à part et NON corrigé. Régénérer les `assembly_*.golden` par leur porte nommée (sans film)
      et les fixtures de contrat par leur double porte ; prouver par `git diff --stat` et un
      extrait que seules les lignes d'origine de pose d'équipement changent. Re-figer les 20
      références d'équivalence à `215649efd` (`-update` par sous-ensembles), puis passe de
      comparaison à 20/20 identiques ; en-tête de `CORPUS.txt` mis à jour. Consigner en §4 que
      F.1 change le contenu cuit SANS montée de `SchemaVersion` (ADR 0034 D-6) et que la
      décision est de NE PAS monter ici : la première montée de M1 (lot 1.6, schéma 55)
      rattrape les artefacts du parc.
      **Fait** (2026-09-13). **UNE SEULE ÉTAPE bouge sur les 20 films : `artifact`.** Les 49
      autres sont identiques partout — aucun balayage n'a changé, et c'est attendu :
      `equipmentOrigin` classe une pose APRÈS le balayage `placements`, qui digère
      `opt.Placements` (la sortie brute de `filmdec`). 11 films sur 20 changent, 9 non.
      **ATTRIBUTION PAR MUTATION, pas par raisonnement** (le raisonnement aurait été faux) :
      revert des seuls fichiers de PRODUCTION d'un commit, re-mesure, comparaison à l'ancienne
      empreinte. Résultat : **F.1 (`c45c411eb`) n'explique que 3 films** — `084a804d`,
      `a521164d`, `d9781168`, −2 octets chacun —, et **D.2 (`3233ec2f8`, « la couverture du
      calque d'objectifs ne compte que les objectifs ») en explique 10**, dont `a521164d` à
      −47 octets. B.3 (`044751026`) n'a AUCUN effet : reverter F.1 et D.2 reproduit l'ancienne
      empreinte **à l'octet sur les 11 films**, ce qui ferme l'attribution. L'attendu du brief
      (« seulement des étapes liées à l'équipement ») était donc incomplet, et D.2 est consigné
      en §4 (D1) avec F.1 : deux commits changent le contenu cuit sans monter `SchemaVersion`.
      Goldens régénérés par la porte nommée (`-update-golden-builds-assembly`, aucun film lu) :
      **un seul golden change, `assembly_a521164d.golden`**, et son diff est exclusivement des
      origines de pose (`16 deployee(s) · 210 lachee(s)` -> `14 · 212` ; deux `grenade_frag`
      passent de `deployed` à `dropped`). Fixtures de contrat régénérées par la double porte :
      **une seule change**, `replay_schema_54_a521164d.json.gz` (191 488 -> 191 487 o, JSON
      1 506 353 -> 1 506 351), plus sa ligne de manifeste ; les 7 autres sont identiques à
      l'octet. Références re-figées à `215649efd` par 6 sous-ensembles, en-tête de `CORPUS.txt`
      réécrit avec le tableau d'attribution. Pas de montée de `SchemaVersion` (décision §4 D1).
      **Découverte D2 (§4)** : trois films ont rendu `ECHEC (code 13)` au plafond mémoire
      pendant que d'autres commandes tournaient, et passent à un vingtième de cette empreinte
      quand rien d'autre ne tourne — un gate de décodage peut donc rougir pour une raison
      étrangère au décodeur.
      Gate : gates communs (§2.3) ; `go test ./internal/games/halo_infinite/film/...
      ./internal/archlint/ ./internal/replaybuild/` vert ; régime court 10/10 identiques ;
      `make go-api-lint`.
- [x] 0.D.1 **D6 — `coverage.score.rounds` 3 -> 1 sur `fb1a1a72` (CTF multi-manche).**
      Instruction bornée à une session : relever `RealRounds` slot par slot sur `fb1a1a72`
      (instrument corpus `CHUNK00_FILMS`), confronter à la feuille (`fb1a1a72.facts.json`, deux
      scores d'équipe) et à `rounds_decide` (`regulation.toml`), sachant que `materialRounds`
      (`statborg.go`) ignore les slots d'ÉQUIPE alors que le registre dit que sur ce film les
      manches viennent de ces slots. Verdict DÉFAUT (le compte publié est faux) ou DIVERGENCE
      VOULUE (entrée citée). Si DÉFAUT : correctif minimal, test par mutation, corpus gate sur
      `fb1a1a72` seul (manifeste réduit, `--base=215649efd`) à zéro perte et gains nommés,
      `SchemaVersion` 55 + entrée de chronique + empreinte de forme + fixtures si le contenu
      cuit change, `GrammarRev` si la grammaire change. Si DIVERGENCE : ligne au registre des
      reports avec condition de reprise, rien de plus.
      **ROUVERT le 2026-09-13 soir sur décision utilisateur (mécanique de jeu) : le verdict
      « manches fantômes » n'est PAS retenu.** Le film ÉCRIT un désignateur de manche 2 sur 148
      enregistrements ; le rejeter par la garde de contiguïté (`contiguousRounds`, « manche 1
      absente donc manche 2 fantôme ») est une heuristique au sens de D13. Faits tranchés par
      l'utilisateur : en Halo Infinite il n'y a PAS de mi-temps (le commentaire de
      `regulation.toml` l. 198 « deux MI-TEMPS » est faux), il y a des MANCHES et des
      PROLONGATIONS ; le score n'est pas un oracle en CTF (0-0 possible pendant 12-13 min, points
      à la fin) ; si le film écrit l'information de manche, on lui fait confiance.
- [!] 0.D.1 bis **Ce que le désignateur 2 ÉCRIT sur `fb1a1a72` veut dire.** Instruction bornée
      (une session) : (a) chez l'ÉCRIVAIN (Ghidra lecture seule) : le champ « manche » des
      enregistrements statborg (joueur ET équipe) : index de manche, phase, prolongation ? quelles
      valeurs le jeu y écrit, et quand ; (b) par mesure : `fb1a1a72` dure 814 s pour 720 s de temps
      réglementaire, la piste PROLONGATION est à tester (les 148 enregistrements en « 2 » sont-ils
      ceux d'une phase, d'une prolongation, d'un type de slot particulier ; leur fenêtre
      [66,7 s ; 814 s] et leur densité 0,20/s sont à EXPLIQUER, pas à écarter) ; comparer à un
      second film avec prolongation avérée et à un film sans ; (c) verdict : ce que le film écrit,
      publié tel quel (manche / prolongation), avec la garde de contiguïté soit supprimée soit
      NOMMÉE comme repli (D14) ; corriger le commentaire de `regulation.toml` (concept de mi-temps
      retiré, remplacé par manches / prolongations, source = décision utilisateur du 2026-09-13) ;
      `SchemaVersion` 55 si le contenu cuit change. Le registre des reports est amendé dans le même
      geste (la ligne 0.D.1 « divergence voulue » devient « non établi, rouvert »).
      **Fait, et STATUÉ `[!]`** (2026-09-14). **Le verdict de 0.D.1 est RETIRÉ ; le verdict de
      remplacement n'est PAS établi.** Ce qui est livré : le retrait motivé sur mesure, la
      correction de `regulation.toml`, l'amendement du registre, et la garde NOMMÉE au sens de
      D14. Ce qui manque : le sens exact du désignateur. Aucun code de production touché, aucune
      montée de schéma.
      **(a) L'ÉCRIVAIN, par la RE déjà au dépôt** (pas de mesure Ghidra neuve : l'existant
      répondait). `FUN_140C18794` est le désérialiseur de l'archétype 6 — décompilé ici : il
      traite une PAIRE de slots de stat (`param_1+8`, `param_1+0xc`), lit pour chacun **un champ
      de 5 bits** qu'il range à `base + 4 + idx*8`, puis une valeur à longueur variable à
      `base + idx*8`, puis deux drapeaux vers le masque `base + 0x1c0`. Le registre ECS du film
      NOMME les 58 composants de cet archétype (`ETAT_DE_L_ART_MODE_SCORE_EVENEMENTS` §17.1) :
      **0-27 `statborg-current-round-value-stat-component`, 28-55
      `statborg-finalized-rounds-values-stat-component`, 56 `statborg-round-outcomes-component`,
      57 `statborg-entry-index-and-type-component`**. Et le getter natif
      `Team_GetCurrentRoundStatValue` @ `0x142C6B118` lit
      `world + statSlot*0x88 + teamIdx*0x1DF0 + 0x38 + round*4` : **la manche EST une dimension
      du moteur**. Ce qui reste NON PROUVÉ : que le champ de 5 bits soit CETTE dimension.
      **(b) La mesure, et elle RÉFUTE une preuve de 0.D.1.** Répartition par tranches de 60 s :
      la « manche 2 » de `fb1a1a72` est **BIMODALE**, pas saupoudrée —
      `[0 50 44 0 0 0 0 0 0 0 0 0 13 41 0]`, soit 94 enregistrements entre 60 et 180 s et 54
      entre **720 et 840 s**. La densité moyenne de 16 % diluait deux amas en un plateau : la
      preuve (3) de 0.D.1 tombe. Forme de référence d'une vraie manche (`d9781168`, 3 manches) :
      blocs CONTIGUS et DISJOINTS (`[248 1173 829 690 1 …]`, `[0 0 0 25 710 831 843 226 …]`,
      `[… 528 732 518 1118 615]`). **Le film ÉCRIT bien de l'information de manche sur ces
      enregistrements** : ils portent **55 lectures des composants 28-55 (manches FINALISÉES)**
      quand la manche 0 du même film en porte ZÉRO. **Contrôles de la MÊME variante sans
      dépassement** : `53ce4390` (CTF:Arena, ~780 s) et `51101d1d` (CTF:Arena Neutral Flag,
      ~270 s) ne déclarent **aucune manche au-delà de 0**, avec 1 et 0 lecture finalisée.
      `fb1a1a72` dure **814 s pour 720 s réglementaires** et son amas tardif tombe exactement
      dans le dépassement : la piste PROLONGATION est SOUTENUE, pas prouvée.
      **PIÈGE À DIRE, trouvé en fin de session : les deux horloges ne coïncident pas.** Le
      `frameCount` du document donne `fb1a1a72` à **757 s** (7 568 frames de 100 ms) quand
      l'horloge DU FILM porte des enregistrements jusqu'à **814 s** — un décalage d'origine que
      je n'ai pas mesuré. Or `analysis.ComputeOvertime` (source unique `OvertimeMarginSeconds`
      = 40 s, `overtime.go`) juge sur la durée du MATCH : à 757 s pour 720 s réglementaires,
      `fb1a1a72` n'est **PAS** en prolongation au sens de la production, alors qu'à 814 s il le
      serait. Et `64e8adfa` (CTF:Arena, **834 s** au document, donc au-delà de 720 + 40) déclare
      des manches 0 et 1 en blocs contigus, pas un « 2 ». **La corrélation prolongation /
      désignateur 2 n'est donc PAS établie** — c'est la première chose à trancher à la reprise,
      et il faut le faire sur une durée de MATCH, jamais sur l'horloge du film.
      **(c) Verdict : NON ÉTABLI.** Ce que le « 2 » désigne (manche jouée, marqueur de manches
      finalisées, index de phase de prolongation) n'est pas tranché, et l'amas de 60-180 s n'est
      pas expliqué. La garde `contiguousRounds` n'est donc NI supprimée NI convertie : elle est
      **NOMMÉE au registre comme heuristique D13 avec son contrat D14** (déclenchement typé,
      compteurs `coverage.score.rounds.{grammaire, repli, contradiction}` à publier, critère de
      retrait « 0 manche refusée par la contiguïté sur le corpus gate », date de pose
      2026-08-18, cible de retrait clôture de M1). `regulation.toml` corrigé (« deux MI-TEMPS »
      -> manches / prolongations, source = décision utilisateur du 2026-09-13) ; la ligne du
      registre passe de « divergence voulue » à « VERDICT RETIRÉ — non établi, rouvert », avec
      ce qui reste vrai de 0.D.1 et ce qui manque. Corpus gate NON joué : sans changement de
      production il ne peut rien mesurer (`git diff` des `.go` hors tests : vide).
- [x] 0.D.2 **D7 — bloc monde/équipement de `60ae07c4` (Live Fire, v37).**
      Séparer l'effet du schéma 53 (porte de région sur 2 bits) de celui de la borne
      `maxUnrollPerStep = 16` : trouver les sha qui encadrent v53 (`document_chronicle.go`),
      jouer le corpus gate sur `60ae07c4` seul avec `--base=<sha juste avant v53>` contre le
      HEAD du commit v53 (worktree temporaire détaché, jonctions `data/cache` comprises, retiré
      après), puis la borne. Attribuer `groundWeapons.spawned 218 -> 35`, `accepted/rejected`
      −95 et `skullCarries.grabs 39 -> 8` à l'un ou à l'autre, chiffres à l'appui. Verdict
      DÉFAUT (dire laquelle des deux entrées affirme « ne bouge pas » à tort, et si la perte est
      réelle) ou DIVERGENCE VOULUE. Correctif seulement si défaut à cause identifiée, même
      protocole qu'en 0.D.1 ; sinon registre.
      **Fait** (2026-09-13). **VERDICT : DIVERGENCE VOULUE sur le bloc monde/équipement — et la
      borne de déroulage n'y est pour RIEN.** Aucun correctif (règle 7). Méthode : quatre points
      de base du corpus gate sur `60ae07c4` seul (manifeste réduit), `--source-root` laissé sur
      ce worktree — la variante « vieux sha en `--source-root` » du brief aurait fait exporter
      les faits par le `cmd/levelup` du 2026-09-11 (`facts.go` lance l'export depuis
      `SourceRoot`), donc **aucun worktree temporaire n'a été créé ni retiré**. Faire varier
      `--base` isole les mêmes révisions et donne QUATRE points au lieu de deux.
      **(1) La borne `maxUnrollPerStep = 16` (`f22474816`) n'explique RIEN** : les rapports aux
      bases `ebd012e3b` (son parent) et `f22474816` sont **identiques octet pour octet** (hors
      durée) — 66 gains, 28 pertes, mêmes valeurs sur tous les axes.
      **(2) La porte de région (`fb71e9b3c`, chronique v53) explique 100 % du bloc** : à la base
      `1a93b34f2` (son parent) le gate rend 17 pertes ; à la base `c5a71dcbd` (le bump v53, juste
      après) il rend **0 perte, 3 gains, exit 0**. Entre les deux il n'y a que ces deux commits.
      **(3) Rien de publié n'est perdu**, lecture des deux artefacts cuits (`--keep-work`) :
      `groundWeapons` publiées **188 -> 188**, `pickups` publiés **297 -> 297**,
      `groundWeapons.kept` **218 -> 218**, `objects` **218 -> 218**. Les « pertes » sont de deux
      natures et d'aucune autre : des compteurs d'ÉCHEC qui baissent (`rejected` 211 -> 116 avec
      `accepted` 429 -> 334, soit −95 des deux côtés et `kept` constant : les 95 partants étaient
      DÉJÀ écartés sur l'identité ; `unknown` 36 -> 5 ; `placements.unknown` 57 -> 4 ;
      `pickups.originUnknown` 56 -> 40 ; `projectiles.truncated` 343 -> 3) et une
      RE-CLASSIFICATION à somme constante (`spawned` 218 -> 35 avec `dropped` **0 -> 183**, somme
      218 des deux côtés : positions fausses, la règle du lâcher ne mordait jamais et tout
      tombait en `spawned` par défaut). Gains massifs en face : `placements` **57 -> 190**,
      `projectiles` publiés **236 -> 417**, `dropperNamed` 0 -> 183, `withOwner` 0 -> 186,
      `dated` 0 -> 31, `cycles` 0 -> 1.
      **(4) L'entrée v53 n'est PAS démentie.** Elle dit « Les armes au sol, les tirs et les
      ramassages ne bougent pas (217, 717, 108 des deux côtés) » — et sur `60ae07c4` les armes au
      sol PUBLIÉES et les ramassages PUBLIÉS ne bougent effectivement pas (188 et 297 des deux
      côtés). Ce qui bouge, ce sont les compteurs de COUVERTURE, dont l'entrée ne parle pas. Le
      constat D7 comparait des compteurs de couverture à une phrase sur des comptes publiés.
      **(5) Aucune part ne vient d'un découpage d'i0 DÉTECTÉ** (question du pilote) : le
      catalogue impose `axisWidths [12 12 11]` et `regionIndexBits 2` pour Live Fire
      (`map_quant_bounds.json`, entrée « live fire », module `sgh_interlock`), ce fichier est
      **identique entre `1a93b34f2` et HEAD** (`git diff` vide), et `film_context.go` — qui
      n'auto-détecte QUE si l'entrée de carte est invalide — est lui aussi **identique aux deux
      révisions**. La détection ([13 12 11]) n'est donc atteinte sur aucun des deux côtés ; le
      seul écart est bien le nombre de bits que la porte consomme. `killcollector/positions.go`
      détecte encore (audit 0.E, A1), mais c'est le chemin des positions de kill, pas celui de la
      cuisson : il ne touche aucun de ces chiffres.
      **(6) TROISIÈME CAUSE, dite et non instruite** (brief : « si une troisième cause apparaît,
      dis-la »). `skullCarries.grabs` 39 -> 8, tout le bloc `equipmentChanges` et les capacités
      ne bougent NI à la porte NI à la borne : ils sont **déjà à leur valeur de HEAD à la base
      `ebd012e3b` (schéma 51)**, donc leur cause est antérieure. Ligne au registre des reports,
      avec ce qui est établi et sa condition de reprise.
- [x] 0.D.3 **D9 — le codec des entrées de golden est incomplet.**
      Compléter le codec `inputs_*.bin.gz` (`golden_inputs_test.go`) pour qu'il transporte les
      rangs de capacité et les origines de pose (tout ce que `decodeFilmInputs` rend et que
      l'assemblage lit), monter la version du fixture (`TestGoldenInputsVersionGuard`),
      régénérer les 7 `inputs_*.bin.gz` (cache de films, un décodage à la fois) puis les
      `assembly_*.golden` et les fixtures de contrat. Preuve : sur les 7 builds, assemblage sur
      entrées FRAÎCHES == assemblage sur entrées RELUES, par un test permanent (plus jamais
      « 36 contre 136 »). Aucun code de production touché.
      **Fait** (2026-09-14). **Les 8 builds passent : assemblage sur entrées FRAÎCHES ==
      assemblage sur entrées RELUES.** Aucun code de production touché (`git diff` des `.go`
      hors tests : vide).
      **L'inventaire a été fait AVANT de coder, par diff d'assemblage sur chaque build** (test
      `TestGoldenInputsFidelite`, neuf, permanent) : **deux familles d'écart, pas une**.
      (i) « N lecture(s) portent le rang SÉLECTIONNÉ » sur **les 8 builds**, relu > frais
      (150/194, 75/379, 100/348, 76/565, 107/570, 109/605, 36/136, 130/346) ;
      (ii) origines de pose sur **`fb1a1a72` seul** (20/319 frais contre 17/322 relu).
      **Trois trous comblés dans le codec**, les deux premiers trouvés par lecture du couple
      struct / encodeur, le troisième par mesure :
      1. **`KeyframeInventory.SelectedGrenadeRank` n'était pas sérialisé.** Le champ vaut **-1**
         quand aucune sélection n'est lue (« une sélection ne se devine pas ») ; relu, il
         revenait à **0**, c'est-à-dire *Fragmentation sélectionnée*. D'où le gonflement de (i)
         sur tous les builds. `GrenadesByPosition` manquait aussi (télémétrie de
         `KeyframeInventoryStats`).
      2. **`InventoryDelta.Ammo` n'était pas sérialisé** du tout : un fixture relu rendait des
         deltas sans munitions. Nouveau couple `encodeDeltaAmmo` / `decodeDeltaAmmo`, les trois
         valeurs derrière leur drapeau de présence (absent et zéro ne sont pas la même chose).
      3. **Les coordonnées étaient ARRONDIES au centimètre**, et l'en-tête du codec affirmait
         que ce n'était pas une perte parce que « toute coordonnée publiée passe par `round2` ».
         **C'est faux pour ce que l'assemblage DÉCIDE** : `equipmentOwner` choisit le poseur
         d'une pose par la plus courte distance, et l'arrondi faisait basculer le poseur de
         trois poses de `fb1a1a72` (poseur 524 -> 514, donc d'autres vies, donc une autre
         origine). Les coordonnées voyagent désormais **à l'identique**, en OU-EXCLUSIF des bits
         du `float32` avec la position précédente du même slot : deux positions voisines
         partagent signe, exposant et haut de mantisse, donc le varint reste court.
      **Version du fixture montée** `REPLAYINPUTS14` -> `REPLAYINPUTS15`,
      `TestGoldenInputsVersionGuard` recalé sur la précédente (le test éprouve toujours un
      refus réel). 8 fixtures régénérées depuis le cache de films (un décodage à la fois), puis
      les goldens d'assemblage par la porte nommée et les fixtures de contrat par la double
      porte.
      **CE QUE LES GOLDENS DISAIENT DE FAUX, et qu'ils ne disent plus** : `bcb6d393` publiait
      **136** lectures à rang sélectionné, la production en produit **36** ; `fb1a1a72` **346**
      contre **130** ; et ses trois poses reprennent `deployed` avec leur vrai poseur. Les 8
      goldens changent, tous dans ce sens.
      **PRIX, ET IL EST NÉGATIF** (arbitrage du pilote, sous-lot 0.D.3 bis) : la première version
      codait les coordonnées en flottants exacts et pesait 17,79 Mio (+72 %) — REFUSÉE, ces
      octets seraient entrés pour toujours dans l histoire partagée. Les positions portent donc
      les QUANTA du film, et les fixtures pèsent **9,86 Mio, soit 511 594 octets de MOINS que
      les 10,35 Mio d origine**. Les fixtures de CONTRAT ne bougent pas (2 108 186 o).
      **Le test permanent SAUTE EN CI** (il exige `REPLAY_FILM_CACHE`, absent de la CI comme
      tous les oracles adossés au cache) : c'est dit dans son en-tête. Ce qui reste gardé en CI :
      le round-trip du codec (`TestGoldenBuildsInputsRoundTrip`) et les goldens d'assemblage.

- [x] 0.D.3 bis **Les entrées de golden portent les QUANTA, pas les flottants dérivés.**
      Arbitrage du pilote après 0.D.3 : la fidélité des coordonnées était acquise, son prix
      (+7,8 Mio de binaires versionnés, pour toujours dans l'histoire partagée, et autant à
      chaque régénération) ne l'était pas.
      **Fait** (2026-09-14). Le codec porte `BipedPosition.Q` — trois entiers, delta-varint par
      slot — et la relecture re-déquantifie par **le même chemin que la production**,
      `filmdec.DequantBipedAxis(q, ax, layout, bornes)`. Le blob ouvre sur le **module de la
      carte** (`MapQuantEntry.Module`) et le décodeur REFUSE une entrée de catalogue qui ne
      correspond pas, par une erreur typée `errGoldenInputsCarte` : sans cette garde la
      confusion serait silencieuse — les bornes d'une autre carte rendent des coordonnées
      FAUSSES, pas approximatives (en-tête de `DequantBipedAxis`). L'entrée arrive en
      PARAMÈTRE (`decodeGoldenInputs(blob, entry)`), elle n'est jamais devinée.
      **UN CHAMP N'A PAS PU ÊTRE DÉDUIT DU CATALOGUE, et il voyage donc dans le blob** : le
      DÉCOUPAGE D'AXE. Le chemin du fixture l'AUTO-DÉTECTAIT (`ScanFilmOptions.Layout` laissé
      nul), et sur **Live Fire la détection rend [13 12 11] quand le catalogue dit [12 12 11]**
      — un bit d'écart sur X double le pas de quantification, donc l'étendue : `60ae07c4`
      rendait `x [-0,39 ; 90,80]` au lieu de `x [-8,56 ; 37,04]`. Déquantifier avec le
      catalogue aurait donc changé le contenu cuit, ce que ce sous-lot s'interdit. Le blob
      porte les trois largeurs employées (3 varints, **coût ~3 octets par fixture**) et
      `decodeFilmInputsForEntry` pose désormais `scan.Layout` explicitement — même valeur, même
      fonction, mais NOMMÉE donc inscriptible.
      **Preuves** : `TestGoldenInputsFidelite` **8/8 vert** (dont les trois poses de `fb1a1a72`
      et leur poseur 524) ; round-trip vert ; **goldens d'assemblage et fixtures de contrat
      IDENTIQUES À L'OCTET** à la version en flottants (`git diff --stat 1b1111379 --` sur les
      deux dossiers : vide) ; taille des 8 `inputs_*.bin.gz` **10 849 119 o -> 10 337 525 o**
      (10,35 Mio -> **9,86 Mio**, −511 594 o), contre 18 656 453 o (17,79 Mio) en flottants.
      Version du fixture `REPLAYINPUTS15` -> `REPLAYINPUTS16`, garde recalée sur la précédente.
- [x] 0.D.4 **D8 — points de piste publiés en baisse (−2 / −9 / −4).**
      Instruction bornée à une session au plus : sur `d9781168`, localiser les points disparus
      (cuisson fraîche à `179bd7401` contre HEAD, diff des pistes), nommer l'étape et le commit
      responsables ; verdict DÉFAUT ou DIVERGENCE ; correctif seulement si défaut à cause
      identifiée (même protocole) ; sinon registre. Si la session ne suffit pas : registre avec
      ce qui a été établi.
      **Fait** (2026-09-14). **VERDICT : DIVERGENCE VOULUE.** Cause nommée, points identifiés à
      l'unité, aucun correctif (règle 7) ; une ligne au registre pour le résidu réel.
      **La cause est `48cf4905d` (schéma 36, « une track = une vie, les bots existent »), et
      elle seule.** Bissection au corpus gate sur `d9781168`, **huit points de base** : la perte
      est PRÉSENTE aux bases 34, 35 et `48cf4905d^`, ABSENTE à `48cf4905d` et à toutes les bases
      suivantes (37, 38, 39, 40, 41, 43, 47). L'hypothèse du plan — « re-segmentation v41, v43,
      v47 » — nommait les bonnes MÉCANIQUES mais les mauvaises versions : c'est **quinze schémas
      plus tôt**.
      **Les deux points perdus, à l'unité** : `slot 558 @ frame 2168` et `slot 573 @ frame 2730`,
      tous deux du même xuid `2535435655459376`. Chacun est le PREMIER point de sa piste et il
      est **ISOLÉ** — le point suivant du même slot arrive **88 frames plus tard (8,8 s)** pour
      l'un, **176 frames (17,6 s)** pour l'autre, très au-delà de `lifeGapUS` (5 s). Rien n'est
      décodé de travers : le nombre de pistes passe de **160 à 174**, les positions sont
      REDISTRIBUÉES, et seuls ces deux échantillons isolés tombent.
      **La chaîne est faite de DEUX règles écrites, qui se composent** : (1) `build.go` ouvre une
      nouvelle vie dès qu'un trou dépasse `lifeGapUS` — même seuil que `buildLifeSpans`, « deux
      découpes divergentes rendraient le nommage par vie inappariable » ; (2)
      `DefaultMinPoints = 2` refuse la vie qui en résulte — « une track d'un seul échantillon
      n'est pas une trajectoire ». Les deux précèdent le constat ; aucune n'est en défaut.
      **CE QUI RESTE, et c'est le vrai résidu** : l'observation est VRAIE (le joueur était là à
      cet instant), elle n'est plus publiée ni comme piste ni comme compteur, et `minPoints`
      refuse **en silence** — aucun compteur de refus dans le paquet, aucune ligne de couverture.
      Un lecteur ne peut pas savoir que le film portait deux positions de plus. C'est la règle
      « jamais d'erreur avalée en silence » appliquée à un refus de publication : au registre,
      avec la question produit attenante (une vie d'UN échantillon doit-elle paraître ?).
      **Second témoin non mesuré, et la raison est dite** : `60ae07c4` ne se cuit PAS au code de
      l'époque du schéma 36 — plafond mémoire dépassé, pic **4,15 Gio**. C'est précisément
      l'ex-bombe que la borne de déroulage `f22474816` a domptée quinze schémas plus tard. Le
      verdict repose donc sur `d9781168`, le témoin que le plan nomme.

- [x] 0.D.7 **Le chemin du fixture impose le découpage d'i0 du catalogue, comme la production.**
      Ferme la découverte D6 du lot 0.D.3 bis : le golden de `60ae07c4` affirmait des
      coordonnées que la production ne produit pas.
      **Fait** (2026-09-14). `decodeFilmInputsForEntry` demande désormais le découpage à
      **`filmdec.NewFilmContextForMap(nil, &entry, nil).ImposedLayout()`** — la fonction de la
      production (`resolveI0Layout`), pas une copie. Appeler `entry.Layout()` aurait été la
      copie que D13 interdit : elle divergerait le jour où la règle change.
      **L'auto-détection ne survit que là où la production l'emploie** — entrée de carte
      invalide (`axisWidths` absent, donc `Valid()` faux) — et elle est alors **NOMMÉE dans le
      blob** (`LayoutDetected`), pour qu'un lecteur sache que ces quanta ne viennent pas du
      catalogue. Sur les 8 builds du corpus, aucune n'y tombe : les 8 fixtures portent
      `LayoutDetected = false`.
      **Contradiction blob / catalogue = erreur typée** `errGoldenInputsDecoupage` : quand le
      fixture dit tenir son découpage du catalogue, il doit être celui que la règle tranche
      aujourd'hui — sinon le catalogue a bougé sous le fixture et les quanta se
      déquantifieraient avec un autre pas, silencieusement.
      **Preuve, et elle est exactement celle qui était demandée** : **UN SEUL golden change,
      `assembly_60ae07c4.golden`** — les 7 autres identiques à l'octet ; **UNE SEULE fixture de
      contrat change**, `replay_schema_54_60ae07c4.json.gz` (172 773 -> 174 897 o) plus sa ligne
      de manifeste. Sur ce film : **`x [-8,56 ; 37,04]` -> `x [-12,90 ; 27,56]`**, pistes
      **174 -> 173**, points de grille **45 137 -> 45 133**, vies publiées **174 -> 173**.
      L'écart n'est pas un simple facteur d'échelle : la détection rendait
      `gate=5 region=0 13/12/11` quand le catalogue rend `gate=6 region=1 12/12/11` — les
      champs d'axe se lisent à d'autres décalages ET la porte de région teste désormais ses DEUX
      bits, ce qui écarte les enregistrements d'une autre AABB.
      **Fidélité 8/8 toujours verte**, version du fixture `REPLAYINPUTS16` -> `REPLAYINPUTS17`,
      garde recalée. Aucun code de production touché (`git diff` des `.go` hors tests : vide).
      **Observation, non instruite** : les points de grille de ce golden passent de 45 137 à
      45 133, soit exactement le **−4** que le constat D8 relevait sur `60ae07c4`. Les deux
      mesures ne portent pas sur le même objet (le golden part d'entrées figées, D8 comparait
      deux cuissons), donc rien n'est conclu — mais la coïncidence mérite d'être dite à qui
      reprendra D8 sur ce témoin.
- [x] 0.D.6 **Les véhicules qui disparaissent au schéma 54 (signalement utilisateur du
      2026-09-13 : « les images des véhicules peuvent disparaître sur le schéma 54 ; les versions
      ont bumpé ces derniers temps sans garder toutes les données »).** Angle mort connu de la
      classification : le calque véhicules naît au schéma 39, donc la comparaison 34 -> 54 ne peut
      PAS voir une dégradation entre 39 et 54 (ses axes sortent « nouveaux »). Instruction bornée
      (une session) : (a) TÉMOIN NOMMÉ PAR L'UTILISATEUR : match
      `bfecd02b-9798-4407-aa55-05244f4c1fa0` (`bfecd02b`, film v41 `HI_1_13_0`, 29 chunks au
      cache, artefact du parc au schéma 54 cuit le 2026-09-13 à 00:43, absent des deux corpus) ;
      l'utilisateur n'a constaté que les véhicules mais soupçonne d'autres dégradations : le gate
      se joue sur TOUS les axes, pas seulement `vehicles.*` ; second témoin `084a804d` (BTB
      véhicules) ; (b) corpus gate sur ces témoins avec `--base` aux bumps successifs 51, 52, 53,
      54 (shas de la chronique) : toutes les pertes (comptes ET durées cumulées par calque),
      chacune nommée avec son bump et attribuée par mutation au commit responsable ; nommer le bump
      où une piste ou une occupation de véhicule raccourcit ou disparaît ; ajouter `bfecd02b` au
      manifeste `config/replay_corpus.toml` (famille `vehicules_v41_utilisateur`, raison écrite) ;
      (c) la règle d'effacement (« le véhicule reste dessiné
      jusqu'à la première preuve mesurée de son absence, fenêtre d'environ 20 s après la dernière
      preuve de présence », texte de `i18n.ts`, code à localiser dans `replay/vehicle_*.go`) est
      une HEURISTIQUE au sens de D13 : confronter à ce que le film ÉCRIT (dead-state ti=40 lisible
      depuis le 2026-09-05, `wt/vehicule-deadstate` ; piège connu : le filtre `DesyncAt == -1`
      jetait des morts de véhicule lues) ; (d) verdict DÉFAUT (données perdues à un bump, ou repli
      qui efface un véhicule que le film montre encore) -> correctif + test par mutation + corpus
      gate zéro perte + `SchemaVersion` 55 si le contenu cuit change, repli NOMMÉ (D14) ; sinon
      registre avec le bump et l'axe.
      **Fait** (2026-09-14). **VERDICT : AUCUN bump n'a perdu de donnée de véhicule, et la
      « disparition d'image » a une cause mesurée qui n'est pas un bump.** Aucun correctif
      (règle 7) ; deux lignes au registre des reports.
      **(a) Le gate, sur TOUS les axes, aux quatre bumps demandés** (`--base` à `b6b198baf^`
      = schéma 51, `b6b198baf` = 52, `c5a71dcbd` = 53, `104b74e15` = 54) sur `bfecd02b` :
      **les quatre rendent exactement les mêmes deux pertes**, et aucune n'est un axe véhicule —
      `coverage.placements.deployed` 4 -> 1 et `byFamilyOrigin.grenade_frag/deployed` 3 -> —,
      c'est-à-dire la règle d'origine des poses (F.1 / H.2), déjà instruite en 0.D.0 et 0.D.5.
      **(b) Élargi jusqu'à la NAISSANCE du calque** (`--base=7c85acf58^`, schéma 39) sur les DEUX
      témoins : 97 pertes au total (15 sur `bfecd02b`, 82 sur `084a804d`), **zéro sur un axe
      `vehicles.*`** — la question « un bump a-t-il raccourci une piste ou une occupation de
      véhicule » se répond NON sur toute la vie du calque. Les 97 se rangent toutes dans des
      familles déjà classées : `geometry.*` (19 axes, chronique v52 props Forge, §7.A A1),
      `objectives.*` + `coverage.objectives` (53 + 2, garde d'effectif `ebd012e3b` et D.2, §7.A
      A3), `projectiles.*` (9, v52 pas impossible, §7.A A6), `placements.*` (règles d'origine),
      `abilities/n` 166 -> 165 (le bloc de la TROISIÈME CAUSE déjà au registre depuis 0.D.2),
      `flagCarries.markerConfirmed/markerObserved` (§7.B B8, déjà « ligne à ACCEPTER »). Une
      seule ligne neuve et minuscule : `coverage.flagCarries.teamBirths` 12 -> 11 sur
      `084a804d` (§4).
      **(c) La cause réelle de la disparition d'image, sur le témoin de l'utilisateur** :
      `bfecd02b` (Snowbound, Team Slayer:Arena) publie **11 vies de véhicule dont 9 de châssis
      `0x038df01a`, absent de la table des familles** — donc sans sprite, dessinées en marqueur
      neutre, comportement ÉCRIT dans `vehicle_families.go` (« VALEUR INCONNUE = FAMILLE VIDE
      [...] le client dessine un marqueur neutre »). Le châssis **n'a jamais figuré au dépôt**
      (`git log -S` sur toute l'histoire : rien) : ce n'est pas une régression. Les neuf sont
      IMMOBILES (aucun échantillon, un `spawn`, fenêtre = le match entier) et `labels.tsv` — la
      seconde source que la table cite elle-même — porte trois entrées `vehi 038df01a` avec la
      banque `sb_003_lvl_moments_ge_shared_autoturret_banished` : **ce sont des tourelles
      automatiques bannies**. Identification neuve, non posée en table (il faudrait décider vers
      quelle famille, et `familleShade` est une tourelle COVENANT — l'y mapper serait l'emprunt
      que l'en-tête interdit).
      **(d) La règle d'effacement confrontée à D13** : elle n'efface JAMAIS avant la dernière
      preuve, elle PROLONGE (`t1max = t1 + 20 s`). Sur le témoin, 10 vies sur 11 ont
      `t1 == t1max` = la dernière frame ; la seule qui s'efface est le `ghost` slot 777, **à la
      frame 2874 (287,4 s), soit 5,3 s APRÈS son dernier échantillon**. **Aucun véhicule que le
      film montre encore n'est effacé.** Le défaut au sens de D13 est ailleurs, et il est réel :
      `VehicleTrack.End` ne prend qu'une valeur (`unknown`), donc la fin de vie est INFÉRÉE d'une
      borne de recensement alors que le film l'ÉCRIT (dead-state `ti=40`, lisible depuis le
      2026-09-05 sur `wt/vehicule-deadstate`, non fusionnée). Registre, avec le repli à NOMMER
      au sens de D14 le jour de la fusion.
      `bfecd02b` est entré au manifeste `config/replay_corpus.toml` (famille
      `vehicules_v41_utilisateur`, raison écrite) : le corpus gate passe de 12 à 13 témoins.

- [x] 0.D.5 **Synchronisation bis : la seconde fusion de `feat/v75` dans l'intégration.**
      Même méthode qu'en 0.D.0, sur la fusion `c28f7da59` (55 commits de finitions et
      d'ajustements : H.1 `flag_neutral`, H.2 « pièce engendrée = déployée » dans
      `equipment_placements.go`, `identity.go`, `usage_summary_families.go`, `flag_carries.go`,
      `replaybuild/flagspawns.go`). Régime court sur le sha de fusion ; lister les étapes qui
      diffèrent, film par film ; **attribution PAR MUTATION** (revert des fichiers de PRODUCTION
      commit par commit jusqu'à reproduire l'ancienne empreinte à l'octet), jamais par lecture du
      code — la leçon de 0.D.0 est que le raisonnement s'y trompe. Régénérer les
      `assembly_*.golden` par `-update-golden-builds-assembly` (sans film) et les fixtures de
      contrat par la double porte ; prouver par `git diff` que seules les lignes attribuées
      bougent. Re-figer les 20 références d'équivalence au sha de fusion (`-update` par
      sous-ensembles), puis passe de comparaison 20/20 identiques ; en-tête de `CORPUS.txt` mis à
      jour. **Tout mouvement orphelin est un constat, listé à part et NON corrigé** (règle 7).
      Gate : gates communs (§2.3) ; `go test ./internal/games/halo_infinite/film/...
      ./internal/archlint/ ./internal/replaybuild/` vert ; régime court 10/10 identiques ;
      `make go-api-lint`.
      **Fait** (2026-09-14). **DEUX étapes bougent sur les 20 films — `flag` et `artifact` — et
      TROIS commits les expliquent entièrement. Zéro mouvement orphelin.** Attribution par
      mutation, jamais par lecture : chaque cause est un revert de fichiers de PRODUCTION suivi
      d'une re-mesure.
      **H.1 (`6e0e5378a`, « la neutralité d'un socle de drapeau est un champ, pas une équipe »)**
      -> étape `flag` sur **17 films sur 20**, compte INCHANGÉ (1), contenu différent. Preuve :
      `flag_carries.go` + `flag_neutral.go` + `replaybuild/flagspawns.go` revertés, `51101d1d` et
      `bcb6d393` rendent l'ANCIENNE empreinte de `flag` à l'octet.
      **H.2 (`267fa1c5a`, « une pièce engendrée est déployée par nature »)** -> `artifact` **+1
      octet** sur `084a804d` et `111fa685`. Preuve : `equipment_placements.go` reverté, le golden
      `111fa685` redevient vert ; le diff du golden est un panneau de mur unique qui passe de
      `dropped` à `deployed` (`wall/deployed` 28 -> 29, `wall/dropped` 26 -> 25, et la ligne
      `wall dropped 0x686b40c9 t=[1482, 1486]` devient `wall deployed`).
      **G.7 (`71aa37fcb`, « le Mutilator entre au registre du rejeu avec son identifiant de
      film »)** -> `artifact` **+316 octets** sur `bcb6d393` et `e5adf7b2`. **Le coupable n'est
      PAS dans `film/replay`** : c'est `internal/games/weapons/{labels.go,registry.go}`, hors du
      périmètre où on l'aurait cherché — trouvé en revertant les six fichiers de production du
      paquet `replay` SANS que l'écart disparaisse, puis en élargissant. Preuve : ces deux
      fichiers revertés, les goldens `bcb6d393` et `e5adf7b2` redeviennent verts ; le diff du
      golden est exactement deux entrées, `0xD7915565` et `0xD791556542C9679F`, toutes deux
      « Mutilator » / « Mutilateur » — l'arme s'affichait en hexadécimal, elle a désormais son
      libellé (c'est un GAIN, et c'est ce que le commit annonce).
      **PREUVE DE COMPLÉTUDE** : reverter les TROIS ensemble reproduit l'ancienne référence à
      l'octet sur les trois films qui portent les trois signatures — `bcb6d393` (flag + G.7),
      `111fa685` (flag + H.2), `51101d1d` (flag seul) : 3 identiques, 0 différent.
      Goldens régénérés par `-update-golden-builds-assembly` (sans film) : **3 sur 7 changent**,
      et leur diff ne porte que les lignes attribuées. Fixtures de contrat par la double porte :
      **3 sur 8 changent** (`111fa685` +2 o, `e5adf7b2` +26 o, `bcb6d393` +23 o compressés) ;
      total 2 108 094 o, sous le plafond de 3 Mio. Références re-figées à `2dad8d6df`, en-tête de
      `CORPUS.txt` réécrit, passe de comparaison **20/20 identiques**.
      **Incident de manipulation, dit parce qu'il a coûté une passe** : un
      `git checkout HEAD -- <paquet replay>` destiné à restaurer la production a aussi restauré
      `testdata/`, effaçant 18 références fraîchement figées et 3 goldens. Le re-figeage a été
      rejoué en entier et rend **exactement le même diff** (17 `flag` + 4 `artifact`, mêmes
      deltas +1/+1/+316/+316) — déterminisme vérifié au passage. Leçon : restaurer par fichiers
      NOMMÉS, jamais par répertoire de paquet quand `testdata/` y vit.

Gate 0.D : gates communs à chaque sous-lot ; régime court 10/10 identiques à chaque sous-lot qui
touche le décodeur ou le constructeur ; corpus gate ciblé (un témoin) sur les sous-lots qui
corrigent ; `make go-api-lint`.

#### Lot 0.E — Inventaire des heuristiques qui décident à la place de la grammaire (D13) — M, audit Opus high, en parallèle de 0.D

Audit (skill `adversarial-audit` : périmètre × axe, registre daté, ne corrige rien). Périmètre :
`film/replay`, `film/killsource`, `analysis/objectiveevents`, `replaybuild` (assemblage des faits),
`sync/killcollector`. Axe : toute décision de production prise par heuristique (fenêtre
temporelle, seuil de distance, majorité, calibration statistique, inférence par le fil des
morts) là où le film ÉCRIT le fait (événement nommé, record de création, composant d'état,
table de `chunk_00`, pied de film).

- [x] 0.E.1 Registre `.ai/V7.5/AUDIT_HEURISTIQUES_DECODEUR_2026-09-13.md` : une ligne par
      heuristique : `fichier:ligne`, fait décidé, heuristique (paramètres), ce que le film écrit
      à la place (canal, événement, record, composant ; PORTÉ aujourd'hui / À PORTER : bloquant
      nommé), preuve ou incertitude (note de RE, rapport, mesure), coût (S / M / L), gain
      attendu (films, poses, kills concernés) ; **et, pour chaque heuristique qui restera un
      repli : sa condition typée de déclenchement (« film muet » : laquelle) et son critère de
      retrait mesurable (D14)** ; noter aussi si le repli actuel est NOMMÉ dans le code ou
      anonyme, et s'il peut se déclencher alors que la lecture existe (risque de déclenchement
      indésirable). Sources à croiser : `REFERENCE_CANAUX_EQUIPEMENT`
      §4 (qui lit quoi), `RAPPORT_F0_DEPLOIEMENT_103`, `RAPPORT_LOT_H_VERSIONS`, notes `film_re/`.
      **Fait** : 133 lignes de registre, 528 sites de décision LUS (fonction entière + appelant),
      201 fichiers de production ouverts, 241 `fichier:ligne` tous vérifiés existants.
      **Amendement D14 (pilote, 2026-09-13) appliqué** : chaque ligne porte le repli associé
      (NOMMÉ / ANONYME), sa condition de déclenchement souhaitable, le risque de déclenchement
      alors que la lecture existe, et son critère de retrait mesurable.
- [x] 0.E.2 Classement en trois tables : (A) le film l'écrit ET le lecteur existe (conversion
      courte, lot 1.9.x) ; (B) le film l'écrit, lecteur À PORTER (lot 3.6, bloquant nommé) ;
      (C) le film ne l'écrit pas (heuristique légitime : reste, avec sa couverture). Chaque ligne
      de (C) cite le négatif MESURÉ qui la fonde (jamais « probablement »).
      **Fait** : (A) 8 · (B) 8, couvrant 40 sites de décision · (C) 38, chacune avec son négatif
      mesuré et sa source · **(D) « non établi » 17** (les lignes sans négatif mesuré y sont
      versées avec la question à instruire) · **(E) replis ANONYMES 62** (table D14, matière du
      lot 1.9.0). **Correction au dimensionnement du plan** : les (B) ne vont PAS toutes au
      lot 3.6 — elles ont déjà leur lot (1.4, 1.5, 1.6, 1.7, 1.8, 3.2, 3.5) ; seuls B7 (composants
      d'objectif ti=11/ti=12) et B8 (respawn timer, état moteur, mapping d'équipe) entrent au 3.6.
- [x] 0.E.3 Ordre proposé des conversions (A), par gain décroissant, recopié en tête de la
      famille 1.9 ; les (B) entrent dans le dimensionnement du lot 3.6.
      **Fait** : items 1.9.2 à 1.9.8 recopiés ci-dessous, plus un item 1.9.0 proposé (registre des
      replis + ratchet), sans lequel le critère de retrait D14 n'est mesurable pour aucune ligne.

**Trois constats du lot 0.E que le pilote doit voir** (détail au registre §10 et §13) :

1. **L'équipe est déjà LUE et JETÉE.** `filmdec/traverse.go:515-517` consomme les 4 bits du
   `managed-player-team-designator-component` et ne publie rien ; neuf décisions de fait s'en
   passent (dont une CALIBRATION sur un oracle de la BASE, `replay/zone_states_owner.go:353`).
   Cinq commentaires du dépôt affirment l'inverse — doc inversée à corriger au lot 1.7. Et, dans le
   même fichier, `replay/document.go:594` affirme « le film ne porte aucun gamertag » quand
   `replay/lives.go:54` documente le contraire.
2. **Onze voies d'inférence décident le même fait** — l'index de joueur d'un slot — que la table des
   joueurs de `chunk_00` écrit à 32 slots sur 1 351 films sur 1 351. Leurs coûts sont chiffrés
   (23 % de désaccord sur l'attribution des tirs ; 51 % d'assistants nommés en BTB ; 6 joueurs
   publiés pour 8 sur `3372e7eb`). Lots 1.5 / 1.6 / 1.8.
3. **62 replis ANONYMES** : une décision de secours sur laquelle aucun compteur ne se pose. Neuf
   portent déjà un défaut mesuré (dont `replay/usage_summary.go:286`, qui crédite au dernier
   occupant du MATCH alors que 32 à 95 % des poses tombent hors de toute fenêtre publiée).

Gate 0.E : chaque `fichier:ligne` existe (grep) ; aucun fichier de production modifié
(`git diff --stat` = le registre et le plan) ; relecture pilote.

**Clôture M0** : fusion dans `feat/v75` sur signal (V3). Les lots 0.A à 0.C ne changent aucun
octet d'artefact. Le lot 0.D, lui, hérite d'un contenu cuit DÉJÀ changé par `feat/v75` (F.1,
`c45c411eb`) : il re-fige les oracles sur ce contenu, et ses sous-lots 0.D.1 / 0.D.2 peuvent
monter `SchemaVersion` s'ils concluent au défaut.

---

### M1 — Les correctifs courts, un par un, sous gate (valeur visible)

Critère d'entrée : M0 fusionné dans l'intégration. Lots strictement séquentiels (même paquet) ;
le second agent est le relecteur du lot précédent. Chaque lot monte `GrammarRev` ; les lots qui
changent le contenu cuit montent `SchemaVersion` (une entrée de chronique chacun) ; UNE
recuisson du parc à la clôture de M1 (D6). Ordre fondé sur le coût et sur les dépendances
(1.4 avant 1.7 ; 1.5 avant 1.6 et 1.8).

#### Lot 1.0 — Le fixture d'entrées rejoue la séquence de balayages de la production — M, high

EN TÊTE DE M1, et avant tout lot qui juge un golden. Né de la découverte D7 (0.D, revue R1) :
`decodeFilmInputsForEntry` est une COPIE de la séquence de `BuildFromFilm`, et les deux ont
déjà divergé.

- [x] 1.0.1 **`BuildFromFilm` expose son étage de balayage** : une fonction, un type d'entrées
      — celui que l'assemblage consomme. Le fixture l'APPELLE au lieu de la recopier. C'est un
      changement de code de PRODUCTION, d'où son placement hors du lot 0.D.
- [x] 1.0.2 **Les cinq canaux absents entrent au codec** : `WeaponChanges`,
      `Pickups`/`PickupStats`, `EquipmentChanges`/`EquipmentChangeStats`, `Vehicles`,
      `BipedCreations`. Ils manquent aujourd'hui au fixture ET au chemin frais, d'où des calques
      VIDES dans les huit goldens — « prises et lachers d arme decodes=0 publies=0 »,
      « vehicules balaye=false » — que le golden ne peut ni garder ni contredire.
- [x] 1.0.3 **Gate** : `TestGoldenInputsFidelite` 8/8 ; équivalence à ZÉRO différence (c'est un
      pas structurel au sens de D4 : le contenu cuit ne doit pas bouger du fait de la
      refactorisation) ; goldens régénérés, et tout écart JUSTIFIÉ ligne à ligne — un calque qui
      cesse d'être vide est un gain à nommer, pas un golden à re-bénir ; `SchemaVersion` monte
      si le contenu cuit change.
- [x] 1.0.4 **Le refus de publication cesse d'être muet** (résidu du lot 0.D.4) : `minPoints`
      refuse une vie d'un seul échantillon sans qu'aucun compteur ne le dise. Publier
      `coverage.tracks` avec le compte de refus, au premier bump de schéma de M1.

Preuve : régime court `replay-equiv` à ZÉRO différence APRÈS 1.0.1-1.0.3 (le pas structurel ne
change pas un octet du document cuit) ; après 1.0.4, les SEULES différences sont le champ neuf
(`coverage.tracks`, 5 axes) et la ligne de schéma — attribution par le diff champ à champ des
deux artefacts du corpus gate. Corpus gate `d9781168` : **0 perte**, 7 gains, schéma 54 -> 55.

#### Lot 1.1 — Pied de film : l'équipe d'un événement est à l'octet 37 — S, high

Sur pièces AVANT le lot : `internal/analysis/objectiveevents/film.go:182,195,251` lisait `b55`
(« NON fiable ») ; le champ `teamRaw` n'avait aucun consommateur. CLOS le 2026-09-14.

- [x] 1.1.1 `decodeTh10Block` lit l'équipe à `ebs+37*8` ; commentaires `:33`, `:182`, `:195`
      corrigés (doc inversée interdite) ; `b38` noté comme doublon observé, non lu.
      FAIT : les offsets du bloc sont nommés (`footerByteTeam = 37`, `footerByteSlot`,
      `footerByteType`, `footerByteTime`, `footerBlockBytes`) au lieu des littéraux `36*8`,
      `47*8`, `55*8` (magic number, anti-patron 6). Le commentaire d'`extract.go:63` (« le champ
      team du film étant non fiable ») est corrigé DANS LE MÊME GESTE : il est la phrase que ce
      lot réfute, et le laisser aurait été la doc inversée que la ligne interdit.
- [x] 1.1.2 Le type exporté des événements d'objectif porte `Team int` (valeur du film, -1 si
      absent) : lecture possible en 1.6 / 1.7, sans consommateur ici.
      FAIT AUTREMENT QUE PRÉVU, et la différence est une DÉCOUVERTE (D3 (1.1) en §4) : ni `NamedEvent`
      ni `IdentifiedEvent` ne peuvent porter ce champ — ils viennent du STATBORG, pas du pied, et
      aucun chemin ne relie `th10Event` à eux. Le seul type exporté que le chemin du pied
      atteigne aujourd'hui est `domain.ObjectiveEvent`, c'est-à-dire une LIGNE DUCKDB : y ajouter
      un champ aurait été un changement de schéma, hors lot. Le porteur est donc
      `objectiveevents.FooterEvent` (ex-`th10Event`, exporté avec `TimeMS`, `Slot`, `Team`,
      `XUID`), lisible par son point d'entrée `FooterEvents(film)` — en mémoire, dans ce paquet,
      et nulle part ailleurs : aucun document cuit, aucune colonne.
      AMENDÉ à la revue R1 (R1-3) : le champ ne porte PLUS de sentinelle « -1 si absent ». Aucun
      producteur ne pouvait la rendre (tous appariés à `ok=false`, jetés), donc elle annonçait un
      contrat — « le film est parfois muet ici » — que la grammaire ne porte pas (D14).
- [x] 1.1.3 Test par mutation sur un bloc de pied du mini-film (`chunk_03` de `000d5950`) :
      remettre `b55` rougit ; corpus : `TestResidusPiedOctetEquipe` reste l'oracle (665/665).
      FAIT SUR UN AUTRE FILM : le pied du mini-film ne porte AUCUN événement th=10 (D2 (1.1) en §4),
      il ne pouvait rien verrouiller. Fixture = 1 930 octets du pied décompressé de `53ce4390`
      (`testdata/pied_bloc_53ce4390.bin` + provenance écrite + porte de régénération
      `PIED_BLOC_UPDATE=1` qui recoupe la tranche sur le film, manifeste à l appui).
      BLOC RE-CHOISI À LA REVUE R1 (R1-1) : le premier portait 1 aux octets 36, 37 ET 38 et ne
      discriminait donc que contre l octet 55 — `footerByteTeam = 36`, le SLOT déclaré la ligne
      au-dessus, le laissait vert. Le bloc actuel (`t=133033`) a slot 2 et équipe 1 ; les tests
      exigent le TRIPLET (slot, équipe, instant) et quatre mutations ont été jouées. Oracle
      corpus rejoué.
- [x] 1.1.4 `GrammarRev` montée ; empreinte verte.
      `grammar-2026-09-13` -> `grammar-2026-09-14`, golden régénéré par sa porte nommée,
      historique du golden complété. L'empreinte NE BOUGE PAS et c'est une découverte (D1 (1.1) en
      §4) : l'ensemble haché est `filmdec/` + `killsource/`, et la grammaire corrigée vit dans
      `analysis/objectiveevents/`. La revision est donc montée À LA MAIN, comme le lot l'exige.
- [x] 1.1.5 (AJOUTÉ PAR LE PILOTE le 2026-09-14) L'ensemble haché de l'empreinte s'étend à
      `internal/analysis/objectiveevents/` : D1 (1.1) se corrige MAINTENANT, pas au lot 2.6.
      `racinesGrammaire` rend trois racines (131 -> 150 fichiers) ; golden régénéré par sa porte
      nommée, `GrammarRev` INCHANGÉE (`grammar-2026-09-14`) puisque aucune grammaire ne bouge
      dans ce commit — seul l'ensemble haché s'élargit, exactement comme la correction inverse
      de la revue R1 du lot 0.A. Mutation jouée : un commentaire ajouté dans
      `objectiveevents/film.go` rend le gate ROUGE, là où le déplacement d'un octet le laissait
      vert ; restauration par fichier NOMMÉ, md5 identique. En-tête du test réécrit (les trois
      paquets, pourquoi le pied est de la grammaire, et le déménagement sous `film/` au pas 5).
- [x] 1.1.6 (AJOUTÉ PAR LE PILOTE le 2026-09-14) Références d'équivalence re-figées à ce HEAD.
      Les 20 `.tsv` dataient d'AVANT le schéma 55 : chaque lot voyait depuis l'écart +104 o à
      l'étape `artifact`, un rouge permanent qui masque le prochain vrai. Re-figées SEULEMENT
      parce que l'attribution par mutation a montré que le lot 1.1 ne change pas un octet du
      document (voir la preuve ci-dessous). AMENDÉ à la revue R1 (R1-5) : la preuve est le GRAPHE
      D APPELS — la chaîne de cuisson n appelle jamais `FooterEvents` ni `Extract` — et la
      mesure par mutation le CONFIRME sans pouvoir le remplacer, puisqu elle ne pouvait pas
      rougir. En-tête de `CORPUS.txt` daté.

Preuve : `replay-equiv` zéro différence (rien de publié ne change) ; corpus gate zéro perte,
zéro gain.
RÉSULTAT (2026-09-14) : le régime court rend `écart à la SEULE étape artifact` sur 10 films —
c'est l'état LAISSÉ PAR LE LOT 1.0.4 (schéma 55 + `coverage.tracks`, références non re-figées),
delta +104 / +103 (`bcb6d393`) / +102 (`51101d1d`), exactement la table de 1.0.4. ATTRIBUTION
FAITE PAR MUTATION, pas par raisonnement : les deux fichiers de production du lot remis à `HEAD`,
`replay-equiv` rend sur `51101d1d` et `bcb6d393` les MÊMES empreintes et les MÊMES comptes
qu'avec le lot (`60d96001…`/628 310 et `fe6f4add…`/1 903 599). **Ce lot ne change pas un octet du
document cuit** — donc corpus gate ciblé NON REQUIS (§2.3 : l'équivalence est à zéro différence
imputable au lot).
ÉTAT FINAL (1.1.6) : les références sont re-figées au commit `a752403da` (schéma 55) et
`replay-equiv` rend **20 identiques sur 20** — le corpus d équivalence est de nouveau un gate qui
peut rougir, au lieu d un rouge permanent que chaque lot devait réexpliquer.

#### Lot 1.2 — Le registre commence à l'octet 8 — M, high (le plus risqué de M1)

Sur pièces : `registry.go:12` suppose `[u32 kind][u32 flags][nom @ +8]` depuis l'octet 0 ; le
jeu lit des entrées de `0x104` octets `[nom @ +0][u32 niveau @ +0x100]` depuis l'octet 8
(`keyframe_fullstate_loop.go:110-120`). Conséquence : `Flags[i]` = niveau du composant `i-1`,
« kind » = queue du nom voisin (0 sur 1 066 slots), et `traverse.go:1301` passe `arch.Level(i)`
à chaque lecteur.

CLOS le 2026-09-14 (branche `feat/decfilm-12`, 6 commits `32f795e2b` -> `845768285` + le commit
de clôture).

- [x] 1.2.1 Mesure AVANT de coder : liste des lecteurs de composants qui consomment le niveau
      (`consumeByNameCapturing` et en dessous) et, sur les mini-films, la distribution des
      niveaux `Level(i)` contre `Level(i+1)` par archétype consommé (ti=9, 11, 12, 35, 40, 42,
      43). Rapport court dans le journal du lot.
      FAIT — instrument permanent `registry_entree_jeu_test.go`
      (`TestRegistreNiveauxVoisinsCensus`), qui lit les OCTETS BRUTS et non l'API : il se relit
      donc des DEUX côtés du correctif sans être réécrit. **Rapport ci-dessous.**
- [x] 1.2.2 `parseRegistry` lit à l'octet 8 avec l'entrée du jeu (nom à +0, niveau à +0x100) ;
      `Flags[i]` = niveau du composant `i` ; le faux « kind » disparaît ; `registryBlockTail` et
      son commentaire (« un cran plus loin ») corrigés ; `FilmMajorVersionFromHeader` inchangé.
      FAIT, avec DEUX écarts au libellé, tous deux dans le sens de la règle : (a) `Flags` est
      RENOMMÉ `Levels` — il n'existe aucun champ « flags » dans une entrée, et garder le nom
      aurait entretenu la lecture qu'on corrige ; (b) `slotName` devient `entryName` et prend
      l'octet de l'ENTRÉE, parce qu'un appelant qui passerait l'ancien offset lirait le nom du
      VOISIN en silence et que le compilateur ne pouvait pas le dire (5 sites de recherche
      revisités un par un). `registryBlockTail` n'exempte plus AUCUN octet : mesuré avant de
      coder sur les 7 bobines par build, même compte de blocs (49 ou 50) et queue nulle 7/7.
- [x] 1.2.3 `shiftArchetypeLevels` et `KeyframeFullStateOpt.LevelShift` supprimés (le décalage
      n'existe plus) ; tests associés adaptés.
      FAIT. `TestKF7ELevelShift` SUPPRIMÉ (il comparait `Level(i)` à `Level(i+1)` : sous le
      cadrage du jeu il ne mesure plus rien) — vérifié absent de
      `.ai/baselines/tests_pre_migration.jsonl`, aucune mise à jour de baseline due.
      `kf7eCases` perd ses 4 lignes « niveaux décalés ». `keyframe_closure.go:55`
      (« `LevelShift` reste FAUX : variable de recherche encore ouverte ») tombe avec l'option.
- [x] 1.2.4 Empreinte du registre (`registry_fingerprint.go`, `testdata/ecs_table.tsv`, table des
      empreintes connues) recalculée sur la nouvelle lecture, provenance datée ; le test corpus
      `lot3_registre_compte_research_test.go` (50 blocs) reste vert.
      FAIT. Le DOMAINE de l'empreinte change avec la lecture : `niveau | nom` au lieu de
      `kind | flags | nom` (deux champs qui n'existent pas) — une empreinte qui hache un décalage
      fige le décalage. `KnownRegistryFingerprint` `0x61e492dd4de7fd4e` ->
      `0x36ca8c3d2a2f9b88`, provenance et domaine écrits au-dessus de la constante.
      `ecs_table.tsv` : 189 lignes de `level` recalculées **depuis le film**, par une porte
      NOMMÉE `-update-ecs-table-level` — voir la découverte D3 (1.2), qui est la raison pour
      laquelle la correction n'a PAS été faite à la main.
- [x] 1.2.5 Test unitaire : sur le `chunk_00` d'un mini-film, le niveau du composant `i` de ti=35
      est celui que `Level(i+1)` rendait avant (preuve de l'équivalence du décalage) et le
      terminateur d'un bloc est entièrement nul.
      FAIT, ÉLARGI aux 7 bobines et à TOUS les archétypes (pas seulement ti=35), plus une
      troisième affirmation : le nom du composant `i` est la chaîne NUL-terminée au PREMIER
      octet de l'entrée `i`. Le test refuse de passer trivialement (il exige qu'au moins un
      composant de ti=35 change de niveau entre les deux cadrages). Deux mutations jouées,
      les deux ROUGES, arbre restauré.
- [x] 1.2.6 `GrammarRev` montée. Si le corpus gate montre une PERTE : c'est un lecteur calibré sur
      le mauvais niveau ; sa correction est DANS le lot (elle bloque le gate), consignée nommément.
      FAIT : `grammar-2026-09-14` -> `grammar-2026-09-14.2`. Le suffixe de rang est NÉ ICI, et
      c'est la découverte D1 (1.2) : la forme datée ne séparait pas deux LOTS du même jour, si
      bien que partager la révision du lot 1.1 aurait voulu dire régénérer le golden sur la
      branche « révision inchangée, empreinte différente » — faire taire le ratchet dans le cas
      précis qui le justifie. AUCUNE PERTE au corpus gate, donc AUCUN lecteur à corriger au
      titre de cet item.

Preuve : `replay-equiv` localise les balayages qui changent (attendu : ceux qui lisent un niveau
qui diffère entre voisins) ; corpus gate zéro perte, gains nommés ; `SchemaVersion` montée si un
octet cuit change.
RÉSULTAT (2026-09-14) : **`replay-equiv` rend 20 IDENTIQUES sur 20**, les 50 étapes de balayage
et l'artefact compris — AUCUN balayage ne change. Ce n'est pas une surprise mais une PRÉDICTION
de la mesure 1.2.1 : les quatre instances dont le niveau change et qu'un déser consomme vivent
en ti=14, 21, 30 et 44, et aucun record de ces archétypes n'est traversé par les balayages du
corpus. Corollaire : **`SchemaVersion` NE MONTE PAS** (55 avant, 55 après) — aucun octet cuit ne
change, donc aucune recuisson du parc n'est due par ce lot. Le corpus gate le confirme.
LE CORRECTIF N'EST DONC PAS INERTE POUR AUTANT : la calibration de `recordStateParam` de
`killsource` — qui, elle, traverse des records bruts — voit son ratio de discrimination bouger
(1,002 -> 1,001 sur la mini-bobine, paramètre retenu et lignes publiées inchangés). C'est la
seule sortie mesurable qui bouge, et elle est consignée comme divergence attendue en §5.

##### Rapport 1.2.1 — les niveaux voisins, mesurés avant de coder

Sur les sept bobines par build (`a521164d` HI_1_4_1, `60ae07c4` HI_1_8_0, `11de8353` HI_1_9_0,
`111fa685` HI_1_10_0, `e5adf7b2` HI_1_11_0, `bcb6d393` HI_1_12_0, `fb1a1a72` HI_1_13_0) :

| Bobine | Composants | Niveaux qui changent | ti=9 | ti=11 | ti=12 | ti=35 | ti=40 | ti=42 | ti=43 |
|---|---|---|---|---|---|---|---|---|---|
| a521164d | 1 033 | 173 (16,7 %) | 1/9 | 3/34 | 6/28 | 19/64 | 18/48 | 10/21 | 10/40 |
| 60ae07c4 | 1 031 | 178 (17,3 %) | 1/9 | 3/34 | 6/28 | 23/64 | 18/48 | 10/21 | 10/41 |
| 11de8353 | 1 031 | 178 (17,3 %) | 1/9 | 3/34 | 6/28 | 23/64 | 18/48 | 10/21 | 10/41 |
| 111fa685 | 1 031 | 186 (18,0 %) | 1/9 | 3/34 | 6/28 | 23/64 | 18/48 | 10/21 | 10/41 |
| e5adf7b2 | 1 031 | 188 (18,2 %) | 1/9 | 3/34 | 6/28 | 23/64 | 18/48 | 10/21 | 10/41 |
| bcb6d393 | 1 067 | 189 (17,7 %) | 1/9 | 3/34 | 6/28 | 23/64 | 18/48 | 10/21 | 10/41 |
| fb1a1a72 | 1 067 | 189 (17,7 %) | 1/9 | 3/34 | 6/28 | 23/64 | 18/48 | 10/21 | 10/41 |

LES LECTEURS QUI CONSOMMENT LE NIVEAU, relevés par `grep -n level traverse.go` — sept étiquettes,
et rien d'autre dans tout le dispatch : `crew-order-component`, `tacmap-poiiconoffset`,
`tacmap-poiicon`, `flock-destination-component`, `player-desired-respawn-location-component`,
`flock-position-component`, `asset-transform-component`. Les 103 autres désérialiseurs ignorent
l'argument `level` : un niveau qui bouge n'y change pas un bit.

CE QUI FONDE LA PORTÉE DU LOT : ces sept étiquettes font **16 instances** dans le registre, et
**4 seulement changent de niveau — identiquement sur les sept builds** :

| Instance | ancien L | L du jeu | Lecteur |
|---|---|---|---|
| ti=14 i0 `crew-order-component` | 0 | 1 | `quantAxisWidth(level)` |
| ti=21 i2 `flock-destination-component` | 1 | 2 | `quantAxisWidth(level)` |
| ti=30 i0 `tacmap-poiicon` | 0 | 1 | `quantAxisWidth(level)` |
| ti=44 i0 `asset-transform-component` | 0 | 1 | `quantAxisWidth(level)` × 5 |

Les douze autres (ti=5 i12, ti=21 i3..i11 et i16, ti=30 i1) gardent le même niveau des deux
côtés. **Aucun composant des archétypes que le lot demandait de mesurer (ti=9, 11, 12, 35, 40,
42, 43) ne consomme le niveau** : leurs 1 à 23 niveaux qui bougent sont donc inertes, et c'est
cette mesure — faite avant d'écrire une ligne — qui prédit le 20/20 identique de l'équivalence.

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
- [ ] 1.6.5 **Refus de publication COMPTÉS** (issu de 0.D.4, 2026-09-14) : `build.go:575`
      refuse en silence toute vie de moins de `DefaultMinPoints = 2` échantillons (deux positions
      réelles de `d9781168` disparaissent sans compteur) : publier `coverage.tracks.refusedMinPoints`
      (vies et points refusés) dans la même montée de schéma ; la question produit « une vie d'un
      seul échantillon paraît-elle ? » est posée à l'utilisateur ; si oui, `DefaultMinPoints = 1`
      dans ce même lot, sinon le compteur seul.

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

#### Famille 1.9 — La grammaire à la place de l'heuristique, un fait par lot (D13) — S à M chacun, high

Ordre fixé par le registre du lot 0.E (table A, gain décroissant). Chaque lot : la lecture de ce
que le film écrit devient la décision ; l'heuristique devient un REPLI NOMMÉ (D14 : entrée au
registre des replis avec sa condition « film muet » et son critère de retrait, déclenché après
la lecture et jamais à sa place), compté (`coverage.<fait>.{grammaire, repli, contradiction}`) ;
test par mutation ; corpus gate zéro perte, gains nommés ; `SchemaVersion` si le contenu cuit
change ; `GrammarRev` si un lecteur change. Le premier lot de la famille pose le registre des
replis et son ratchet (1.9.0). Premier lot de conversion fixé par l'utilisateur :

- [ ] 1.9.0 **Registre des replis** (`facts`, ou `replay` tant que la couche n'existe pas) :
      table nommée (nom, fait, condition typée, date de pose, cible et critère de retrait),
      ratchet `archlint/no_unregistered_fallback_test.go` (un repli hors registre = rouge ;
      convention de nommage détectable), rapport de couverture par fait ; les replis EXISTANTS
      recensés par 0.E y entrent tels quels avec leur critère, sans changer de comportement
      (équivalence zéro différence).

- [ ] 1.9.1 **Origine d'une pose d'équipement.** Sur pièces : `replay/equipment_placements.go`
      (`equipmentOrigin`, fenêtre de 200 ms depuis F.1 `c45c411eb`), `REFERENCE_CANAUX_EQUIPEMENT`
      §1 (le type 103 `EquipmentSpawnedObject` désigne 216 panneaux de mur sur 216 et aucun
      appareil porté). Mur : `deployed` = pose désignée par un 103 (panneaux), sinon `dropped` ;
      **règle déjà en production depuis le lot H.2 des finitions (`267fa1c5a`,
      `equipment_placements.go`, `equipmentIsSpawnedPiece`) : tout objet dont le manifeste
      `replay_labels.toml` dit `kind = "deployed"` reçoit `origin = deployed` AVANT toute mesure
      temporelle ; ce lot y ajoute le garde-rail Go qui apparie les identifiants au manifeste
      (P2 D-H2 de la revue finitions G/H, seul le garde web rougit aujourd'hui) et la lecture du
      103 comme contrôle (216/216 panneaux désignés)** ; la première référence du 103 (`ref0`,
      entité ti=37 de longue durée vue aux images-clés 737/739, jamais créée en delta) est une
      PISTE pour l'équipement SOURCE d'un déploiement, non instruite (table D du registre 0.E) ;
      appareils portés (capteur, traqueur, écran, champ) : AUCUN signal écrit connu (la référence
      de créateur et `ability-enabled-id` du record de création ti=37 ont leur porte FERMÉE sur
      503 records sur 503, `filmdec/equipment_creation.go:29` ; corrigé par l'audit 0.E, B3) ; la
      fenêtre temporelle reste, comme REPLI NOMMÉ et compté (D14), avec la question ouverte de la
      table (D) du registre 0.E comme condition de retrait. Les 22 poses requalifiées par F.1 sont
      rejugées une à une (instruments `f1_origine_*`). Corpus gate sur `0797ce72`, `4f77afc1` et
      l'échantillon court.
- [ ] 1.9.2 **Le découpage d'i0 vient du catalogue de carte, plus de l'auto-détection.**
      `internal/sync/killcollector/positions.go:253` (et `hits.go:157`) construisent
      `DefaultScanFilmOptions()` avec `Layout` nil alors que `entry` est le paramètre de la fonction
      et que `entry.Range()` est lu à la ligne suivante ; `MapQuantEntry.Layout()`
      (`filmdec/map_bounds.go:69`) est déjà imposé sur l'autre chemin (`replay/build_from_film.go:87`).
      Gain : **27 faux enregistrements sur 267 400 éliminés sur Live Fire** (mesure du 2026-09-03 sur
      `60ae07c4`, `filmdec/film_context.go:33-42`) et deux passes de détection supprimées par film. S.
- [ ] 1.9.3 **Le couple (tueur, victime) lu au kill-event 85, plus recollé sur le voisin.**
      `internal/games/halo_infinite/film/killsource/feed.go:162` (`reconstructPairs`, fenêtre de
      2 instants) contre `killsource/eventchain.go:242` (`readKillEvent`, victime ET tueur dans le
      même enregistrement, déjà PORTÉ mais lu pour le seul assistant). Gain : **64 couples sur 372**
      cessent d'être une reconstruction ; supprime la fabrication d'un couple quand la vraie victime
      est un bot (`feed.go:149-151`). M.
- [ ] 1.9.4 **La carte du film vient du nom de match, plus d'une signature de largeurs.**
      `internal/sync/killcollector/hits.go:151` appelle `DetectFilmWorldRange(dir, path, "")` alors
      que le même collecteur résout le nom de carte à `positions.go:210` et que le paramètre
      `mapNameOverride` existe. Gain : **6 cartes jumelles** (3 paires mesurées, F.0 §6 réserve 2)
      récupèrent leurs distances, aujourd'hui désactivées en silence. S.
- [ ] 1.9.5 **Le porteur du crâne lu au canal des armes tenues.**
      `internal/games/halo_infinite/film/replay/skull_carries.go:390` infère le porteur des tics de
      score (trou > 3 s) alors que le crâne voyage dans le canal des armes tenues (famille
      `0x0017592c`, `replay/held_object_carry.go:15`) et que `BuildHeldObjectCarry` est PORTÉ mais
      n'a qu'UN appelant de production, `replay/bomb_carries.go:142`. Les tics deviennent le repli
      compté. Gain non chiffré. S.
- [ ] 1.9.6 **Le drapeau qui rentre pris dans `ev.flag`, déjà nommé en amont.**
      `internal/games/halo_infinite/film/replay/flag_carries_lives.go:267` cherche « le seul drapeau
      au sol » alors que `ev.flag` est posé par `flag_carries_home.go:82-98` et ne sert qu'à un
      court-circuit (`:264`). Gain : les `ambiguousReturns` (compteur publié) que `ev.flag` tranche. S.
- [ ] 1.9.7 **Dead-state et kill-feed appariés par l'identité de paquet, plus par 2,5 s.**
      `internal/games/halo_infinite/film/killsource/options.go:114` (`tolMS = 2500`, justifiée par la
      comparabilité et non par une mesure) employée à `match.go:30`, `:45`, `:103`, `:143` ; les deux
      structures portent `(chunk, pidx)` (`killsource/scan.go:49`, `assist.go:169`) mais `Kill` ne le
      transporte pas (`match.go:186-193`). La fenêtre devient le repli compté. M.
- [ ] 1.9.8 **Le chunk du pied pris au type du manifeste, plus par argmax de kills.**
      `internal/games/halo_infinite/film/killsource/feed.go:82` ; le type est porté par
      `filmsource.Film.Meta()` (`analysis/filmsource/film.go:41`) et déjà lu par ce patron
      (`objectiveevents/extract.go:144`), mais `killsource/chunks.go:78` perd `Meta()`.
      RÉSERVE : le manifeste est un fichier EXTERNE — l'argmax reste en repli COMPTÉ. S.

Arbitrage du pilote (2026-09-13) : l'item 1.9.0 ci-dessus EST le registre des replis proposé par
l'audit ; il entre les **62 replis anonymes** de la table (E) du registre 0.E, et ses 9 replis à
défaut déjà mesuré sont listés dans son journal.

**Hors famille 1.9, routés par le lot 0.E** : `replay/projectiles.go:106` — **6,0 % des trajectoires
du parc** (947 sur 15 735) sont coupées par un garde-fou qui compense une faute de déquantification
de `filmdec` ; c'est une CAUSE à corriger (lot B-bis), pas une conversion.
`replaybuild/zones.go:141` — porter `GameVariantCategory` dans `port.MatchFacts` (la plus petite
correction du périmètre, aucun décodage).

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
| 2026-09-13 | 0.A.1 | **D1 — Le corpus d'équivalence est périmé, pas en régression.** 13/13 films diffèrent à l'étape `score`. Références écrites au commit `179bd7401` (lot 4b) sous `replay.SchemaVersion = 34` ; la constante vaut 54 (`film/replay/document.go:75`). Vingt montées de schéma = vingt changements voulus du contenu cuit depuis le figeage (dont la borne de déroulage `maxUnrollPerStep` 100 000 → 16, commit `f22474816`, `objectiveevents/named_bounds.go:84`, alors que l'en-tête de `CORPUS.txt` disait les quatre bombes figées SOUS 100 000). Aucune référence régénérée. | Décision du pilote : re-figer les 20 films à un commit nommé. Bloque S3 et le gate d'entrée de tout lot décodeur |
| 2026-09-13 | 0.A.1 | **D2 — Ce gate n'a aucun gardien.** `go test ./internal/games/halo_infinite/film/... ./internal/archlint/ ./internal/sync/killcollector/` est VERT (10 paquets, exit 0) alors que l'équivalence est rouge sur 13/13 depuis ~30 commits. La CI ne voit ni `replay-equiv` ni le corpus gate (§2.3) : une dérive de cuisson peut vivre des semaines sans un seul signal. | Renforce le besoin de 0.A.4 (empreinte) et du registre §5 ; candidat à un gate CI au jalon M0 |
| 2026-09-13 | 0.A.1 | **D3 — `-corpus` ne déplace pas les références, contrairement à ce que suppose le mode opératoire des worktrees.** `dossierEquivalence()` (`cmd/replay-equiv/main.go:150`) dérive du SEUL `-repo-root` : `LEVELUP_REPO_ROOT=<principal>` ferait lire les références ET les `.facts.json` du checkout principal, et `-update` y ÉCRIRAIT. Depuis un worktree, la seule voie correcte est `-repo-root <worktree>` + jonctions `data/cache/film_chunks` ET `data/cache/film_manifests` vers le principal (le manifeste manquant rend `score` nul sans erreur : `replaybuild/matchfacts.go:95`). | §2.2 du plan et brief des lots suivants : corriger le mode opératoire |
| 2026-09-13 | 0.A.1 | **D4 — Le régime court ne couvre pas la grammaire la plus ancienne.** L'échantillon défini par V2 (un film par build + 3 nommés) donne 9 films et exclut par construction les deux films SANS section d'identification (`50247b26` v31, `a349fea8` v33), qui ne portent pas de build. Le plafond est de 10 : un dixième slot est libre. | **TRANCHÉ le 2026-09-13 (pilote)** : `50247b26` entre à l échantillon court, qui passe à 10 films — M1 (lots 1.5 à 1.8) exerce précisément le repli des films sans section. Fait dans `CORPUS.txt` et §2.3 |
| 2026-09-13 | 0.A.1c | **D5 — Le re-figeage ne cache aucune régression.** Classification des 110 couples film × étape qui bougent entre le schéma 34 et le schéma 54 (sur 650 comparés) : tous rattachés à une entrée datée, **ZÉRO constat de régression**. Seules baisses de compte : `objectives` → 0 sur 4 films (garde d'effectif `ebd012e3b`, sièges > 8 slots, refus tracé à l'exécution), `projectiles` −11 sur `60ae07c4` (chronique v53, porte d'i0, Live Fire seule carte à index de région sur 2 bits), `artifact` (longueur en octets, grandeur dérivée). Prédiction de la chronique v54 vérifiée : `deaths` et `killRefs` ne bougent que sur les deux films v39 du corpus. Rapport : `.ai/V7.5/RAPPORT_REFIGEAGE_EQUIVALENCE_2026-09-13.md` | **AMENDÉE le même jour** : le croisement par le corpus gate (D6 à D8) voit sur les AXES DU DOCUMENT trois pertes que cet oracle ne pouvait pas voir. D5 reste vraie pour l équivalence, elle ne vaut pas verdict global |
| 2026-09-13 | 0.A.1c bis | **D6 — `coverage.score.rounds` 3 → 1 sur `fb1a1a72`, CTF que le registre tient pour MULTI-MANCHE.** `materialRounds` IGNORE les slots d'équipe (`statborg.go:557`) ; or sur ce film « les enregistrements de slot JOUEUR declarent TOUS la manche 0 [...] les manches viennent des slots d EQUIPE » (`REGISTRE_REPORTS.md` ligne 592, volet statborg **TOUJOURS OUVERT sur ce film exact**). Deux lectures non tranchées : correction de manches fantômes (oracles POUR : feuille `teamScores [0,1]`, `regulation.toml [rounds_decide]` ne liste que les Oddball) contre perte d'information sur un film réellement multi-manche. | À instruire : relever `RealRounds` slot par slot sur une cuisson de `fb1a1a72` (combien de manches déclarent les slots d'ÉQUIPE, `runs` passe-t-il `statMinRoundRun` en manches 1 et 2). Rattacher au volet statborg de la ligne 592 |
| 2026-09-13 | 0.A.1c bis | **D7 — Le bloc monde/équipement s'effondre sur `60ae07c4` et l'entrée la plus proche le CONTREDIT.** `groundWeapons.spawned` 218 → 35, `accepted` 429 → 334 et `rejected` 211 → 116 (exactement −95 chacun : 190 enregistrements quittent la classification), `powerupAccepted` 504 → 399, `skullCarries.grabs` 39 → 8, `placements.lives` 361 → 327, `equipmentChanges.decoded` 38 → 33. La chronique v53 mesure pourtant, sur l'autre film Live Fire : « Les armes au sol, les tirs et les ramassages **ne bougent pas** (217, 717, 108 des deux côtés) ». Les 27 enregistrements hors arène ne couvrent pas 190. Confondant : `60ae07c4` est le seul témoin à la fois Live Fire (v53) et Oddball (borne `f22474816`). | À instruire : cuire `60ae07c4` aux deux révisions encadrant v53 seule et relever `coverage.groundWeapons`. Si v53 explique 190, l'entrée v53 est fausse sur ce point et doit être amendée |
| 2026-09-13 | 0.A.2 | **D9 — Le golden par build décrit l'assemblage sur le sous-ensemble d'entrées que le codec transporte, pas la sortie de production.** *(Corrigée à la revue R1, P2-2 : la première rédaction inversait les chiffres et disait « six builds sur sept ».)* Sur **les sept builds**, l'assemblage bâti sur les entrées fraîchement décodées diffère de celui bâti sur les mêmes entrées RELUES depuis le fixture. Mesuré : `bcb6d393` « lecture(s) portent le rang SELECTIONNE » **36 en frais, 136 en relu** ; `fb1a1a72` origine mesurée des poses **20 déployée(s) / 319 lâchée(s) en frais contre 17 / 322 en relu** — trois poses dont l'**ORIGINE** change, et **479 lignes sur 582 décalées**. Le codec est pourtant un point fixe (`TestGoldenBuildsInputsRoundTrip` vert sur les sept) : il ne perd rien de ce qu'il porte, il ne porte ni les rangs de capacité ni les origines de pose. **Sur `fb1a1a72` le golden publie des origines d'équipement que la production ne produit pas.** | Non corrigé (règle 7). Le golden verrouille la non-régression du CONSTRUCTEUR à entrées constantes, rien de plus. **Condition de reprise** : compléter le codec (`inputs_*.bin.gz`) pour qu'il porte rangs de capacité et origines de pose, puis re-figer les sept goldens. Candidat au **lot 0.D**, décision utilisateur |
| 2026-09-13 | 0.A.2 | **D10 — `ScanBipedPositions` ne s'exécute pas sur une mini-bobine.** Il dérive sa bande de slots bipède des images-clés et refuse le film quand elle est vide (« aucun slot biped (ti=35) dans les keyframes du film », `offline_biped.go`) : les images-clés d'une bobine sont concaténées hors continuité et la bande ne s'y établit pas. Le banc du plan a donc été remplacé par `BenchmarkKeyframeClosure`, le balayage chaud équivalent, que le brief autorisait. | Sans objet si M2 garde des bobines ; à reconsidérer si un banc doit un jour mesurer la dérivation de bande — il faudrait alors un film entier, donc un banc hors CI |
| 2026-09-13 | 0.A.2 | **D11 — `a521164d` (HI_1_4_1) ferme dix fois moins que les autres builds.** Fermeture totale d'image-clé : 1,0 % contre 9,7 à 15,7 % sur les six autres. Le « +200 o de HI_1_4_1 » est déjà au registre des reports (§1.2 du plan, handoff §5) ; cette mesure le CHIFFRE pour la première fois. | Lot 3.6 : traiter HI_1_4_1 en dernier, ou instruire d'abord son décalage d'en-tête — porter un composant n'y rendra rien tant que le cadre est décalé |
| 2026-09-13 | 0.A.1c bis | **D8 — Points de piste publiés en baisse, sans entrée qui les nomme.** `tracks.points/n` −2 (`d9781168`), −9 (`084a804d`), −4 (`60ae07c4`) ; `tracks.points.hp/presents` 463 → 461 ; `abilities/n` 166 → 165. Points disparus en ENTIER (t, x, y, z baissent d'autant). Ce n'est PAS la garde des bornes (`boundsOf` ne mute pas `tracks`, `geometry.go:233-248`). Piste non prouvée : re-segmentation « une track = une vie » (v41, v43, v47) et recalage `bestDeathOffset` (v48) déplaçant les bords de vie. Ampleur 0,005-0,008 %. | À instruire : identifier les 9 points de `084a804d` (piste, instant, bord de vie ou non). Sévérité faible |

| 2026-09-13 | 0.D.0 | **D1 — DEUX commits de `feat/v75` changent le contenu cuit SANS montée de `SchemaVersion`.** F.1 (`c45c411eb`, « l'origine d'une pose d'equipement devient purement temporelle ») et D.2 (`3233ec2f8`, « la couverture du calque d'objectifs ne compte que les objectifs ») modifient tous deux des octets du document publié, et `SchemaVersion` vaut toujours 54 : les artefacts du parc cuits en 54 AVANT eux sont périmés sans être marqués (ADR 0034 D-6, « SchemaVersion rises when the cooked content or the document shape changes »). Les deux commits l'assument par écrit (D.2 : « aucun champ ne bouge, seule la valeur change, et `SchemaVersion` ne monte donc pas » ; « les artefacts deja cuits gardent l'ancien denominateur jusqu'a leur recuisson »). Ampleur MESURÉE par mutation (cf. §5) : D.2 touche 10 des 20 films du corpus, F.1 en touche 3, aucun autre commit du span ne touche l'artefact. **Décision du lot : PAS de montée ici** ; la première montée de M1 (lot 1.6, schéma 55) rattrape ces artefacts, et la recuisson du parc reste un signal séparé (D6). NON TRAITÉ. | lot 1.6 (schéma 55) ; recuisson du parc sur signal de l'utilisateur |
| 2026-09-13 | 0.D.0 | **D2 — le plafond mémoire de `replay-equiv` tombe par CONTENTION CPU, pas par le film.** Trois films du corpus ont rendu `ECHEC (code 13)` à 3,81-3,83 Gio (plafond dur 3,75 Gio) quand une autre commande tournait sur la machine, et les MÊMES films passent à 0,19-1,06 Gio quand rien d'autre ne tourne : `a521164d` 3,81 Gio en lot contre **0,19 Gio seul**, `50247b26` 3,83 contre **1,06**, `111fa685` 3,81 contre **0,30**. Mécanisme : l'enfant tourne en priorité `below_normal` (`filmproc.LowerOwnPriority`) et son plafond souple est un `debug.SetMemoryLimit` — une limite SOUPLE, que le ramasse-miettes tient en tournant plus souvent ; privé de CPU, il prend du retard et l'empreinte dépasse le plafond DUR avant qu'il ne rattrape. Conséquence : un gate de décodage peut rougir pour une raison qui n'a rien à voir avec le décodeur, et le message (« plafond memoire depasse ») envoie chercher une fuite. `1c4c63c2` est le cas limite : 2,70 Gio seul à HEAD, mais 3,79 Gio seul sous mutation — il n'a fini qu'avec `-mem-gib 8` (pic 6,00 Gio, 11 min 58 s). NON TRAITÉ. **Condition de reprise** : un gate de décodage en CI, ou un lot qui fait tomber la mémoire du constructeur. Mesure de contournement, à écrire dans les briefs : ne rien lancer d'autre pendant un décodage. | §2.2 du plan (mode opératoire) ; candidat à un lot de robustesse du harnais |

| 2026-09-13 | 0.D.1 | **D3 — le corpus gate lit TOUTE baisse de `coverage.score.rounds` comme une perte, alors que retirer une manche fantôme est le gain cherché.** `rounds` n'est pas dans la liste fermée des compteurs d'ÉCHEC de `internal/replaydiff/polarite.go`, donc il garde la lecture générique « plus = mieux ». Or l'instruction 0.D.1 prouve que la baisse 3 -> 1 sur `fb1a1a72` est une CORRECTION (manche 1 sans un seul enregistrement, manche 2 à 16 % de la densité de la manche 0 et recouvrant sa fenêtre). Conséquence : tout lot futur qui améliore la détection des manches fantômes sortira en « perte » au gate et devra être ré-instruit à la main, exactement comme celui-ci. Même famille que les 8 lignes de polarité douteuse du §7.B du rapport de re-figeage, et que la ligne « `shotsNoRide` / `ambiguousSlot` » déjà au registre. NON TRAITÉ (règle 7). | Le lot qui inventoriera les compteurs de VOIE et d'ÉCHEC du contrat de couverture (registre des reports, ligne du lot E2-bis) : y faire entrer `score.rounds` avec la bonne polarité, ou l'inscrire en ligne à ACCEPTER au gate |

| 2026-09-13 | 0.D.2 | **D4 — le constat D7 groupait DEUX causes séparées par ~17 montées de schéma.** Les treize axes que D7 citait ensemble sur `60ae07c4` se scindent : le bloc ARMES AU SOL / POSES / PROJECTILES / RAMASSAGES vient à 100 % de la porte de région (`fb71e9b3c`, v53) et n'est pas une perte ; le bloc ÉQUIPEMENT / CRÂNE / CAPACITÉS (`skullCarries.grabs` 39 -> 8, tout `equipmentChanges`, `abilities/n` 29 -> 27, `abilityLabels/n` 4 -> 3) est déjà à sa valeur de HEAD au schéma 51 et vient d'ailleurs. **Leçon de méthode** : un constat de gate qui énumère des axes « qui bougent ensemble » ne prouve pas une cause commune — ici la co-occurrence venait uniquement de la largeur du segment comparé (34 -> 54). Un constat de gate devrait porter le segment le plus étroit où il tient encore. NON TRAITÉ. | Le lot qui révisera le protocole du corpus gate (même famille que la ligne « pas de mécanisme d'acceptation datée d'une perte instruite » au registre) : exiger d'un constat qu'il nomme le segment minimal, pas le segment de la campagne |
| 2026-09-13 | pilote (message inter-sessions des finitions) | **Quatre faits des lots F/H à absorber** (`HANDOFF_DECODEUR_FILM` §3 bis sur `origin/feat/v75` 95e5b2ed6) : (1) pièce engendrée = déployée par nature (règle H.2 en production, garde-rail Go à poser) -> 1.9.1 ; (2) le 103 ne désigne que la pièce engendrée, `ref0` = piste de l'équipement source -> 1.9.1 + table D ; (3) origine des appareils portés purement temporelle -> repli nommé 1.9.1 ; (4) Live Fire, index de région 2 bits imputé à X par `DetectI0Layout` ([13 12 11] au lieu de [12 12 11]) : imposer le catalogue, ne jamais détecter en production -> 1.9.2 (A1 de l'audit). `origin/feat/v75` porte ~55 commits de plus que la fusion 215649efd, dont H.1 (neutralité d'un socle), H.2 (pièce engendrée), H.3 (Aquarius = Live Fire) qui touchent `replay/` : une NOUVELLE fusion + re-figeage attribué sera nécessaire avant la poussée vers `feat/v75` (sous-lot 0.D.5) | plan : 1.9.1, 1.9.2 ; §5 |

| 2026-09-14 | 0.D.6 | **D5 — `coverage.flagCarries.teamBirths` 12 -> 11 sur `084a804d`, sans entrée qui le nomme.** Seule des 97 pertes du segment schéma 39 -> 54 à ne se ranger dans aucune famille déjà classée (§7.A, §7.B, ou les lignes du registre écrites en 0.D.2). `teamBirths` n'est pas un compteur d'échec : c'est un compte de naissances de drapeau d'équipe, donc une baisse est une richesse en moins. Ampleur 1 sur 12. NON TRAITÉ (règle 7). | Le lot drapeau qui reprendra les reliquats de la vague 6, ou le lot qui inventoriera les compteurs de `flagCarries` (même famille que la ligne du lot E2-bis au registre) : nommer le bump qui fait tomber la douzième naissance et dire si elle existait |

| 2026-09-14 | 0.D.3 bis | **D6 (FERMÉE au lot 0.D.7, 2026-09-14) — le chemin du fixture AUTO-DÉTECTAIT le découpage d'axe, la production l'IMPOSE depuis le catalogue, et les deux divergent sur Live Fire.** `decodeFilmInputsForEntry` laissait `ScanFilmOptions.Layout` nul, donc `DetectI0LayoutOf` décidait ; sur `60ae07c4` elle rend **[13 12 11]** quand `map_quant_bounds.json` dit **[12 12 11]**. Un bit d'écart sur X double le pas de quantification : les positions du fixture couvrent `x [-8,56 ; 37,04]` là où le catalogue donnerait `x [-0,39 ; 90,80]`. Le sous-lot a rendu le découpage EXPLICITE (`scan.Layout` posé à la valeur détectée, donc aucun changement de comportement) et l'a inscrit au blob, mais **il n'a PAS tranché laquelle des deux valeurs est juste** — le faire aurait changé le contenu cuit, hors périmètre (règle 7). Note : la mesure du lot 0.D.2 a établi que la CUISSON DE PRODUCTION, elle, impose bien le catalogue (`film_context.go`, entrée valide) ; l'écart est donc entre le fixture et la production, pas dans la production. | Le lot qui tranchera le découpage d'i0 de Live Fire (famille 1.9 ou lot de profil M2) : dire laquelle de la détection et du catalogue a raison sur cette carte, puis aligner le chemin du fixture sur la production. **FERMÉE** : le lot 0.D.7 a aligné le chemin du fixture sur la production — il demande le découpage à `NewFilmContextForMap(...).ImposedLayout()`, la fonction de la production et non une copie ; l auto-détection ne subsiste que là où la production l emploie et elle est alors NOMMÉE dans le blob (`LayoutDetected`), avec une erreur typée sur contradiction blob / catalogue. Seul `60ae07c4` a bougé (`x [-8,56 ; 37,04]` -> `x [-12,90 ; 27,56]`, 45 137 -> 45 133 points), les 7 autres goldens identiques à l octet. Ce qui reste HORS de ce lot et n est PAS tranché : laquelle de la détection et du catalogue dit vrai sur Live Fire — le lot a aligné le fixture sur la PRODUCTION, il n a pas jugé la production |

| 2026-09-14 | 0.D (revue R1) | **D7 (0.D) — le chemin du fixture est une COPIE de la séquence de balayages de `BuildFromFilm`.** `decodeFilmInputsForEntry` rejoue à la main l'ordre des balayages de la production au lieu de l'appeler : les deux séquences ne peuvent que diverger, et elles divergent déjà. **Cinq canaux que `BuildFromFilm` câble n'existent ni au fixture ni au chemin frais** : `WeaponChanges` (`build_from_film.go:190`), `Pickups`/`PickupStats`, `EquipmentChanges`/`EquipmentChangeStats`, `Vehicles` (`:370`), `BipedCreations` (`:156`). Conséquence mesurable dans TOUS les goldens : des calques vides qui ne le sont pas en production — « prises et lachers d arme decodes=0 publies=0 », « vehicules balaye=false ». Le golden ne peut donc rien dire de ces calques, ni en régression ni en progrès. La revue R1 a par ailleurs montré ce que coûte cette copie : deux verdicts de balayage (`InventoryDeltaAmmoRefused`, `FilmMajorVersion`) manquaient au fixture, et « canal munitions refuse » était une CONSTANTE fausse sur 5 des 8 builds. NON TRAITÉ ici : combler les cinq canaux exige de toucher le code de PRODUCTION (exposer l'étage de balayage), donc hors 0.D (règle 7). | **Lot 1.0**, en tête de M1 (ajouté au plan le 2026-09-14) |
| 2026-09-14 | 1.0 | **D1 (1.0) — `VehicleScan.Stats` (`filmdec.EquipmentCreationStats`) n'a AUCUN lecteur dans l'assemblage.** Vérifié sur pièces : `buildVehicleTracks` (`vehicle_tracks.go:106-116`) lit `Scanned`, `Keyframes`, `Creations`, `Positions`, `Events` et `Aims` — jamais `Stats` ; la couverture du calque se calcule sur les vies, les créations et les épisodes. Le champ est donc rempli par `decodeFilmVehicleScan` puis jeté. Le codec du fixture ne le transporte pas, et son en-tête (`encodeVehicleScan`) le dit. NON TRAITÉ : le retirer touche un type de PRODUCTION hors périmètre du lot 1.0. | Audit du calque véhicules, ou lot 2.7 (scission des fichiers), qui rouvre `build_vehicles.go` |
| 2026-09-14 | 1.0.4 | **D2 (1.0) — les vies refusées par `minPoints` portent TOUTES exactement UN point, sur les huit builds.** Mesure du 2026-09-14 (`coverage.tracks`) : `000d5950` 1 vie / 1 point, `111fa685` 2 / 2, `11de8353` 3 / 3, `60ae07c4` 4 / 4, `a521164d` 6 / 6, `bcb6d393` 2 / 2, `e5adf7b2` 2 / 2, `fb1a1a72` 0 / 0 — `refusedPoints == refusedMinPoints` partout. Aucune vie de deux points ou plus n'est refusée : le seuil ne coupe donc pas une trajectoire, il écarte un échantillon isolé (un slot qui s'ouvre et se referme). C'est le CHIFFRE qui manquait pour poser la question. NON TRAITÉ : `DefaultMinPoints` reste à 2, décision utilisateur en attente. | Décision utilisateur ; le compteur est publié, la question peut désormais être posée avec sa mesure |
| 2026-09-14 | 1.0 (revue R1) | **D4 (1.0) — le décodeur du fixture alloue `make([]T, 0, n)` avec `n` LU DU FLUX.** `decodePositionSection`, `decodeCreations`, `decodeGoldenEvenements` et les autres dimensionnent leurs tranches sur un compte que le blob leur donne. Un blob DÉSYNCHRONISÉ À MAGIE ÉGALE (donc que la garde de version laisse passer : une section modifiée sans montée de magie) y lit un entier arbitraire et demande une allocation d'autant — `out of memory` au lieu d'un échec nommé. PRÉ-EXISTANT depuis la v10 au moins, et sans effet tant que la magie monte à chaque changement de suite de sections (ce que le journal des versions et `TestGoldenInputsVersionGuard` tiennent). NON TRAITÉ : borner chaque `n` au reste du flux est un durcissement du codec, hors périmètre du lot. | Lot qui rouvre le codec (0.A ultérieur, ou 2.6 « empreintes par couche et types de contrat ») |
| 2026-09-14 | 1.0 (revue R1) | **D5 (1.0) — `FilmInputs.applyTo` n'a pas de garde par réflexion.** `TestCodecCouvreFilmInputs` exige que tout champ de `FilmInputs` soit transporté par le CODEC ou nommé absent ; rien n'exige qu'il soit POSÉ par `applyTo` sur les `Options`. Un champ ajouté au type et au codec mais oublié dans `applyTo` n'atteindrait l'assemblage par aucun chemin. Aujourd'hui les huit goldens l'attrapent (le calque correspondant resterait vide) — mais ils l'attrapent APRÈS régénération, pas à la compilation. NON TRAITÉ : la garde demanderait de comparer deux structures de formes différentes (`FilmInputs` contre `Options`), champ par champ, avec une table de correspondance à tenir. | Lot 2.6 (types de contrat) ou la revue du prochain canal ajouté |
| 2026-09-14 | 1.0.1 | **D3 (1.0) — `Options.observe` est une méthode SUR VALEUR : chaque étape copie l'intégralité d'`Options`.** La structure porte une soixantaine de champs (une trentaine de tranches, six sous-structures d'entrée) ; l'étage de balayage l'observe 36 fois par film. Aucun effet mesuré — le décodage pèse des dizaines de secondes — et le récepteur par valeur est ce qui rend `observe` sûr : l'observateur ne peut modifier aucune entrée. NON TRAITÉ : ce serait un changement du contrat d'`observe`, hors périmètre. | Lot 2.3 / 2.5 (globales et couches), qui rouvre `Options` |

| 2026-09-14 | 1.1.4 | **D1 (1.1) — FERMÉE AU LOT 1.1.5, le 2026-09-14. L empreinte de `GrammarRev` ne couvrait PAS la grammaire du pied.** L ensemble haché par `grammar_rev_fingerprint_test.go` était `filmdec/` + `killsource/`. Or le pied de film est lu par un TROISIÈME paquet, `internal/analysis/objectiveevents/` (`scanTh10Events`, `decodeTh10Block`) — des offsets d octets dans un bloc de 60, c est-à-dire de la grammaire au sens exact de `grammar_rev.go` (« une largeur, un cadre, un ordre de composants, un lecteur »). VÉRIFIÉ SUR PIÈCES : après le correctif de l octet 37, `TestGrammarRevSuitLaGrammaire` est resté **VERT** sans qu on touche à la revision — exactement le faux négatif que le lot 0.A.4 voulait fermer, et la revision a dû être montée à la main. **CORRECTION (1.1.5)** : `racinesGrammaire` hache désormais les TROIS paquets (131 -> 150 fichiers, +19), golden régénéré (empreinte `3c775720…` -> `474c9faa…`, revision INCHANGÉE : rien de la grammaire n a bougé, seul l ensemble haché s élargit), et la mutation prouve la morsure — un commentaire ajouté dans `objectiveevents/film.go` rend le gate ROUGE, là où le déplacement d un octet le laissait vert. Le renvoi initial au lot 2.6 est ANNULÉ : un garde-rail qui laisse passer le changement qu il existe pour attraper n en est pas un. | Clos. Au pas 5 (V5), `objectiveevents` déménagera sous `film/` : `racinesGrammaire` échouera alors bruyamment sur un dossier vide, ce qui est le comportement voulu |
| 2026-09-14 | 1.1.3 | **D2 (1.1) — le pied de la mini-bobine `000d5950` ne porte AUCUN événement th=10.** Le plan demandait la mutation « sur un bloc de pied du mini-film (`chunk_03`) » : mesuré, `scanTh10Events` rend **0** sur les trois chunks de la bobine (3 987 761 o, 25 140 o, 714 479 o décompressés). C'est cohérent avec ce qu'est ce film — `Slayer:Arena Super Fiesta`, un mode SANS objectif — et avec ce qu'est `chunk_03` d'après sa propre provenance (« le chunk highlight du film, fil des morts »). Une fixture de bloc de pied ne peut donc pas venir de la mini-bobine : il faut un film de mode à objectif. CONSÉQUENCE TRAITÉE DANS LE LOT (fixture prise sur `53ce4390`) ; ce qui reste NON TRAITÉ, c'est le manque d'une mini-bobine de mode à objectif pour les lots suivants. | Lot 1.7 / 3.x s'ils ont besoin d'un pied hors ligne ; V7 (mini-films par build) si le besoin se répète |
| 2026-09-14 | 1.1.2 | **D3 (1.1) — `NamedEvent` et `IdentifiedEvent` ne viennent PAS du pied.** Le plan supposait que l'un des deux porterait l'équipe du film. Sur pièces, les deux naissent du STATBORG (`StatRecords` -> `NamedEventsFrom` -> `IdentifyNamedEvents*`) et leur `Slot` est un slot d'entité statborg (10..24 pairs) ; le pied, lui, donne un `b36` de 0..3 qui n'est pas un `player_index`. AUCUN chemin ne relie `th10Event` à ces types. Le seul type exporté que le chemin du pied atteigne est `domain.ObjectiveEvent`, c'est-à-dire une ligne de `shared.match_objective_events` : y poser un champ aurait été un changement de SCHÉMA DB, interdit dans ce lot. D'où `FooterEvent`, en mémoire dans `objectiveevents`. NON TRAITÉ : le lot 1.7.3 devra brancher `domain.ObjectiveEvent.TeamID` (colonne qui EXISTE déjà, pointeur nullable) sur `FooterEvent.Team` au lieu du roster — pas de champ neuf à créer. | Lot 1.7.3 |
| 2026-09-14 | 1.1.1 | **D4 (1.1) — doc inversée résiduelle dans `internal/domain/objective_events.go:12-13`.** L'en-tête justifie encore la nullabilité de `TeamID` par « team unreliable sur certains matchs » : c'est la phrase que ce lot réfute (l'octet 55 vaut 0 partout, l'octet 37 donne 665/665). NON TRAITÉ ici : `domain/` est hors du périmètre du lot, et la phrase redeviendra juste-ou-fausse selon ce que 1.7.3 fera de la colonne — la corriger avant ce basculement, c'est écrire deux fois. | Lot 1.7.3, dans le commit qui bascule la source de `TeamID` |
| 2026-09-14 | 1.1.3 | **D5 (1.1) — le XUID d'un événement du pied est à 1 866 octets DU BLOC, et cette distance est CONSTANTE.** Mesure du 2026-09-14 sur TOUS les blocs de quatre films et deux builds (`bcb6d393` HI_1_12_0 17 blocs, `53ce4390` 34, `7344d24f` 71, `64e8adfa` 68 — HI_1_13_0) : `ebs - xstart = 14 926 bits` (1 865,75 octets) sur **190 blocs sur 190**, une seule valeur distincte, sans une exception. Le bloc de 60 octets que la production décode ne CONTIENT donc pas le XUID qu'elle lui attribue — elle l'ancre sur un XUID situé loin devant, et la fenêtre de recherche de 20 000 bits (`decodeTh10Block`) est plus large que cette distance de seulement 5 074 bits. Une constante non expliquée dont la marge est de 34 % n'est pas un fait établi : c'est un appariement qui tient par chance jusqu'au build qui l'allongera. NON TRAITÉ. | Lot 1.7.3 (qui rouvre ce bloc pour l'équipe) ou la famille 1.9 (D13 : lire ce que le film écrit plutôt qu'une fenêtre) |
| 2026-09-14 | 1.2.6 | **D1 (1.2) — TRAITÉE DANS LE LOT (elle bloquait le gate). La forme de `GrammarRev` ne séparait pas deux LOTS du même jour.** `grammar_rev.go` disait « deux changements le même jour partagent la même révision — c'est voulu : ce qui compte est qu'un LOT de changements soit séparable du précédent ». Le lot 1.1 (2026-09-14) tenait déjà `grammar-2026-09-14` ; le lot 1.2 tombe le même jour et change VRAIMENT la grammaire. Partager la révision aurait obligé à régénérer le golden sur la branche « révision inchangée, empreinte différente » — c'est-à-dire à faire taire le ratchet dans le cas précis pour lequel il existe. La forme admet désormais un rang `.N` par LOT (`grammar-2026-09-14.2`), et `grammar_rev.go` porte la raison. `GrammarRev` n'a AUCUN consommateur hors de son garde-rail (vérifié par grep), donc le changement de forme ne casse rien. | traitée ici ; à relire au lot 2.6 si la révision acquiert un lecteur |
| 2026-09-14 | 1.2.6 | **D2 (1.2) — la sortie de `killsource` PEUT changer, et `KillSourceDecoderRev` n'a PAS été montée : c'est une décision de production.** Mesure : sur `killsource/testdata/minibobine.golden`, UNE ligne sur 77 bouge, `recordStateParam=3 [croissance x1.002]` -> `x1.001` ; le paramètre retenu, les lignes de kill, la couverture, le contrôle négatif, les voies et la santé sont identiques à l'octet. La calibration RSP traverse des records BRUTS (`calibrate.go:157`), donc elle voit les quatre niveaux corrigés ; les lignes publiées, non — sur ce témoin. Monter `KillSourceDecoderRev` rouvre un backlog de redécodage des lignes en base : c'est un geste de PROD, réservé au pilote sur signal (D6). NON TRAITÉ. Le garde-rail `TestKillSourceDecoderRevSuitLeDecodeur` reste vert de lui-même (il ne hache que `film/killsource/`, que le lot ne touche pas) — ce silence est précisément ce qui rend la décision explicite nécessaire. | décision pilote : bump + backlog killsource, ou constat écrit que les lignes ne peuvent pas changer |
| 2026-09-14 | 1.2.4 | **D3 (1.2) — TRAITÉE DANS LE LOT. Les annotations `niveau_jeu=N` d'`ecs_table.tsv` étaient INCOMPLÈTES : 178 lignes annotées pour 189 réellement décalées.** La table documentait l'écart depuis le lot R7-e ; corriger la colonne `level` À PARTIR DE CES ANNOTATIONS laissait 11 lignes fausses (mesuré : G2 rouge sur 11 clés, dont `35|63|biped-action-component` et `44|0|asset-transform-component`). La colonne est donc régénérée DEPUIS LE FILM par une porte nommée, et le README dit que c'est la seule colonne qui ne se corrige pas à la main. LEÇON GÉNÉRALE, non traitée : toute colonne de cette table qui est une donnée du film (et non un jugement humain) dérive silencieusement tant qu'elle se recopie à la main. | lot 2.6 (grammaire en instruments) : recenser les colonnes dérivées de la table ECS |
| 2026-09-14 | 1.2 (équivalence) | **D4 (1.2) — le corpus d'équivalence ne porte AUCUN témoin des archétypes ti=14, 21, 30 et 44.** Les quatre instances dont le niveau change et qu'un déser consomme vivent dans ces archétypes ; sur les 20 films du corpus, les 50 étapes de balayage et l'artefact sont IDENTIQUES avant et après le correctif. Le corpus prouve donc l'absence de régression, mais il est AVEUGLE au gain : aucun film n'exerce le chemin corrigé. NON TRAITÉ — étendre le corpus exigerait un film portant des `crew`/`flock`/`tacmap`/`asset-transform`, c'est-à-dire probablement du PvE ou de la Forge, pas du matchmaking. | lot 3.6 (ports de composants) ou extension de corpus : nommer un témoin par archétype porté |

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
| 2026-09-13 | 0.A.1a | `cbfdc269d` (arbre inchangé) | `replay-equiv -repo-root <worktree>` (corpus entier, 13 films) | **ROUGE — 0 identique, 13 différents**, tous à l'étape `score`. Durées : `000d5950` 15,3 s · `01e1f945` 19,5 s · `64e8adfa` 40,3 s · `7344d24f` 23,2 s · `696a9d7c` 23,5 s · `084a804d` 2 min 01 s · `1c4c63c2` 2 min 44 s · `53ce4390` 33,2 s · `d9781168` 26,6 s · `9f57c612` 17,1 s · `60ae07c4` 27,7 s · `51101d1d` 5,5 s · `a349fea8` 2 min 10 s. **Total 647 s**, pic max 0,69 Gio (`1c4c63c2`). Ligne de base du budget (§7.8 de l'architecture) |
| 2026-09-13 | 0.A.1b | `<commit du lot>` | lecture d'en-tête `chunk_00` des 20 films (u32 LE offset 0 + chaîne `HI_1_x_y`) | 20/20 résolus, conformes à la table du brief : v31 et v33 `a349fea8` sans section ; `a521164d` 33/HI_1_4_1 ; `60ae07c4` 37/HI_1_8_0 ; `11de8353` 38/HI_1_9_0 ; `084a804d`,`1c4c63c2`,`111fa685` 39/HI_1_10_0 ; `e5adf7b2` 40/HI_1_11_0 ; `bcb6d393` 40/HI_1_12_0 ; le reste 41/HI_1_13_0. `CORPUS.txt` à 20 lignes, 7 champs chacune |
| 2026-09-13 | 0.A (communs) | `<commit du lot>` | `gofmt -l ./internal ./cmd` ; `go vet` ; `go test` (film, archlint, killcollector) | gofmt vide ; vet propre ; **tests VERTS, 10 paquets, exit 0** — cf. découverte D2 : vert malgré l'équivalence rouge |
| 2026-09-13 | 0.A.1c | `061b6f5b7` | `replay-equiv -repo-root <worktree> -films <7 sous-ensembles> -update` | **20/20 figés** au commit `cbfdc269d`, schéma 54, `# digest-grammar: 2`, 50 étapes chacun. Durées : 000d5950 26,2 s · 01e1f945 23,4 s · 64e8adfa 29,3 s · 7344d24f 36,7 s · 696a9d7c 41,5 s · 53ce4390 33,6 s · d9781168 26,1 s · 9f57c612 16,8 s · 60ae07c4 27,3 s · 51101d1d 5,3 s · 084a804d 2 min 46,8 s · 1c4c63c2 2 min 32,0 s · a349fea8 3 min 57,7 s · bcb6d393 23,0 s · a521164d 1 min 15,8 s · 111fa685 42,7 s · 11de8353 42,6 s · e5adf7b2 46,5 s · fb1a1a72 1 min 23,8 s · 50247b26 1 min 19,3 s. **Total ~16 min**, pic max 0,68 Gio (1c4c63c2) |
| 2026-09-13 | 0.A.1c | `<commit du lot>` | classification `.tsv` ancien (`git show cbfdc269d:...`) contre `.tsv` neuf, 13 films × 50 étapes — **aucun décodage** | 650 couples comparés : **540 identiques, 110 différents, 34 étapes sur 50 intactes sur tous les films**. Trois baisses de compte seulement, toutes expliquées (D5). **ZERO constat orphelin sur cet oracle** — amende par le corpus gate, cf. la ligne suivante. Rapport `.ai/V7.5/RAPPORT_REFIGEAGE_EQUIVALENCE_2026-09-13.md` |
| 2026-09-13 | 0.A.1c bis | `6512f14ba` (arbre) | `replay-corpus-gate --base=179bd7401`, manifeste réduit à 5 témoins (pilote) | **PERTE sur 5/5** — gains 133/247/114/445/165, pertes 8/18/22/90/61. Classification bis (aucun décodage de l'exécuteur) : **6 familles expliquées (§7.A), 8 de polarité douteuse (§7.B), 3 constats à instruire (§7.C = D6, D7, D8)**. Le gate compare les axes du DOCUMENT, l'équivalence les sorties de BALAYAGE : les deux oracles ne voient pas la même chose, et c'est le second qui a trouvé les trois constats |
| 2026-09-13 | 0.A.2 (déterminisme) | `0493e3102` | `replay-equiv -repo-root <worktree>` sur les 20 films, 6 sous-ensembles | **20/20 IDENTIQUES** — preuve de déterminisme des références re-figées, premier gate de 0.A.2. Durées : 15,1 / 18,8 / 28,0 / 21,8 / 21,5 / 31,8 / 25,7 / 16,8 / 26,9 / 5,4 s · 084a804d 2 min 06 · 1c4c63c2 2 min 08 · a349fea8 2 min 07 · bcb6d393 12,4 · a521164d 38,9 · 111fa685 44,7 · 11de8353 47,3 · e5adf7b2 42,0 · fb1a1a72 32,8 · 50247b26 1 min 56 |
| 2026-09-13 | 0.A.2 | `1937135cf`, `02047ea06` | `go test -run MiniFilmBuildsRegenerate -update` puis `-run GoldenBuildsRegenerate -update` | 7 bobines écrites, 941 808 à 1 046 290 octets (90 à 100 % du plafond V7) ; 7 fixtures d'entrées (642 Ko à 1,7 Mo compressés, 11 Mio au total) et 7 goldens d'assemblage |
| 2026-09-13 | 0.A.3 | `1937135cf` | `go test -run KeyframeClosureRatchet` ; mutation `consumeDefaultStateTI6` R(6)→R(7) | Golden 221 lignes, ratchet VERT ; sous mutation **6 lignes BAISSE** (ti=6 de 910/911, 1438/1439, 764/766, 670/671, 574/575, 528/528 à ZÉRO), test ROUGE ; largeur remise, vert |
| 2026-09-13 | 0.A.4 | `00340fe16` | `go test -run GrammarRevSuitLaGrammaire` ; ligne ajoutée dans `traverse.go` | Empreinte `bbc80c89…` sur filmdec + killsource ; sous mutation **« LA GRAMMAIRE A CHANGE SANS MONTEE DE REVISION »**, rouge ; ligne retirée, vert, `git diff` vide |
| 2026-09-13 | 0.A.5 | `00340fe16` | `go test -bench . -run '^$' -count 5` | `bench_baseline.txt` commise : BitReaderReadBits ~113 µs/op (≈570 MB/s), TraverseEntity ~367 ms/op, KeyframeClosure ~377-646 ms/op (5 mesures chacun) |
| 2026-09-13 | 0.A (clôture) | `02047ea06` | gates communs + `replay-equiv` échantillon court (10 films) | `gofmt` vide ; `go vet` propre ; `go test` **10 paquets, exit 0** ; `make go-api-lint` **0 issues** ; équivalence **10/10 IDENTIQUES** — aucun comportement modifié par le lot |
| 2026-09-13 | 0.A revue R1 | `<commit R1>` | revue adversariale ronde 1, puis corrections | **1 P1 + 6 P2 recevables, 0 jeté, 7 corrigés dans le lot**, 29 conditions tenues. P1-1 portes de régénération nommées (`-update-grammar-rev`, `-update-keyframe-closure`, `-update-golden-builds-assembly`) + annonce stderr : **preuve — sous mutation ti=6, `go test ./…filmdec/ -update` nu laisse les deux goldens INTACTS et le test ÉCHOUE** (il répondait `ok` et les réécrivait). P2-7 porte d'assemblage séparée : **régénère les 7 goldens SANS `REPLAY_FILM_CACHE`, diff vide**. P2-3 `grammar_rev.go` sorti du hachage (132 → 131) : **la branche « révision changée sans grammaire » est atteignable — montée seule → message dédié, rouge**. P2-4 wording corrigé (217 lignes de données inchangées : la valeur était déjà la plus fréquente). P2-6 baseline `-count 10` re-figée, budget +10 % restreint aux deux bancs serrés. P2-2 D9 réécrite (chiffres inversés, 7 builds, origines de pose). P2-5 ligne dupliquée retirée |
| 2026-09-13 | 0.A revue R1 | `<commit R1>` | gates communs + régime court | `gofmt` vide ; `go vet` propre ; `go test` **10 paquets, exit 0** ; `make go-api-lint` **0 issues** ; `replay-equiv` **10/10 IDENTIQUES**. Aucun fichier de décodage touché (le diff de `keyframe_closure.go` est **100 % commentaires**, vérifié) |
| 2026-09-13 | 0.A revue R2 | `<commit R2>` | revue adversariale ronde 2 (corrections seules), puis retouches | **0 P1 + 5 P2, 0 jeté, 5 corrigés**, 33 conditions tenues — la boucle converge (1 → 0 bloquant). **C1 : une porte de régénération ne rend JAMAIS `ok`.** `go test` jette la sortie d'un paquet qui PASSE : l'annonce stderr posée en R1 était invisible sans `-v` (mesuré : golden réécrit, stdout `ok`, stderr vide). Les trois portes terminent désormais par `t.Fatalf` listant les références réécrites. **Preuve, commande documentée sans `-v`** : `-update-keyframe-closure` → « 1 reference(s) reecrite(s) : testdata/keyframe_closure.golden (9557 octets) », FAIL ; `-update-grammar-rev` → « 1 reference(s) reecrite(s) … », FAIL ; `-update-golden-builds-assembly` → « 7 reference(s) reecrite(s) : … », FAIL ; chacune relancée sans son drapeau → `ok`. C2 : chaque message nomme le drapeau de SA référence (`:238` ne nie plus la coupure P2-7). C3 : fragment de doc inversée retiré. C4 : bloc de commande FR aligné sur l'EN. C5 : item 0.A.5 du plan mis à `-count 10` et budget restreint |

| 2026-09-13 | 0.B (CI) | `a8f8e3c70` | push `feat/recherche-decodeur-film` : workflows CI, Deploy Pre-Check, Secrets | **3 succès sur 3** (CI = Frontend + Go Build/Test + Go Lint) |
| 2026-09-13 | 0.C (CI) | `661936101` | push `feat/recherche-decodeur-film` : workflows CI, Secrets | **2 succès sur 2** |
| 2026-09-13 | 0.B.7 | mesure préalable (documents PLEINS) | `REPLAY_CONTRACT_UPDATE=1 go test ./internal/games/halo_infinite/film/replay/ -run ContractFixturesRegenerate -update` | 8 fixtures écrites depuis les entrées figées, 7,6 s, aucun film lu. JSON 2,1 à 10,4 Mo ; compressé 262 039 à 1 236 186 o ; **TOTAL 6 142 690 o = 5,86 Mio, contre un plafond de 3 Mio**. La porte sort en **ÉCHEC** en nommant les 8 fichiers réécrits (motif 0.A/R2-C1) |
| 2026-09-13 | 0.B.7 | mesure préalable (documents PLEINS) | `cmp replay_schema_54.json.gz replay_schema_54_000d5950.json.gz` (neutralité du changement de chemin) | **IDENTIQUES octet pour octet** — dériver la table de `goldenBuilds()` ne change rien au document du film de référence |
| 2026-09-13 | 0.B.7 | mesure préalable (documents PLEINS) | `gofmt -l ./internal` ; `go vet ./internal/games/halo_infinite/film/replay/` ; `go test ./internal/games/halo_infinite/film/replay/ ./internal/archlint/` | gofmt vide ; vet propre ; archlint **ok** (17,5 s) ; replay **tout vert SAUF `TestContractFixturesTiennentDansLeBudget`** (5,86 Mio > 3 Mio) — `MatchCommitted` (manifeste compris) et `CarryCurrentSchema` verts sur les 8 builds, `Regenerate` sauté sans sa double porte. Paquet 7,8 s → 14,4 s (7 assemblages de plus) |
| 2026-09-13 | 0.B.7 | mesure préalable (documents PLEINS) | `npx tsc -b --force` ; `npx eslint` (3 fichiers web touchés) ; `make go-api-lint` | tsc **exit 0** ; eslint **0 erreur** ; lint Go **0 issues** (baseline non accrue) |
| 2026-09-13 | 0.B.7 | mesure préalable (documents PLEINS) | `npx vitest run src/features/match-replay src/lib/replay` | **202 fichiers passés + 1 sauté, 3 123 tests + 3 sautés** (3 029 avant le lot, soit +94) ; les 8 builds traversent contrat zod strict, normalisation et logiques pures — `HI_1_4_1`, `HI_1_8_0`, `HI_1_9_0`, `HI_1_10_0`, `HI_1_11_0`, `HI_1_12_0` et les deux `HI_1_13_0` — **aucun défaut de frontière par build** |
| 2026-09-13 | 0.B.7 | mesure préalable (documents PLEINS) | simulation des coupes (node, lecture seule des fixtures produites, aucun décodage) | `tracks[].points` = **85 à 89 %** de chaque document. Totaux gzip simulés : 1 point sur 5 **1 842 400 o (1,76 Mio)** · 1 sur 10 1 246 973 (1,19) · 1 sur 25 879 340 (0,84) · 2 points par piste 610 054 (0,58). Sans indentation : −8 % seulement (401 589 → 368 650 sur le film de référence). Full sur 5 builds (les 3 plus lourds retirés) : 2,57 Mio |
| 2026-09-13 | 0.B.7 (coupe A) | ce commit | `REPLAY_CONTRACT_UPDATE=1 go test ./internal/games/halo_infinite/film/replay/ -run ContractFixturesRegenerate -update` | 8 fixtures réécrites à `pointsStride` 5 (1 pour `000d5950`), 3,2 s, aucun film lu. **TOTAL 2 108 044 o = 2,01 Mio, sous le plafond de 3 Mio** — 401 589 · 191 488 · 172 786 · 337 201 · 372 485 · 351 316 · 80 469 · 200 710. La porte sort en **ÉCHEC** en nommant les 8 fichiers ET la fixture périmée supprimée (`replay_schema_54.json.gz`, l'ancien nom) |
| 2026-09-13 | 0.B.7 (coupe A) | ce commit | `git show HEAD:…/replay_schema_54.json.gz \| cmp - …/replay_schema_54_000d5950.json.gz` | **IDENTIQUES octet pour octet** : la référence historique traverse le lot intacte (renommage seul) |
| 2026-09-13 | 0.B.7 (coupe A) | ce commit | ÉPREUVE PAR MUTATION du ratchet « un seul jeu vivant » (copie déposée sous `replay_schema_53_bcb6d393.json.gz`) | `TestContractFixturesUnSeulJeuVivant` **ROUGE**, message nommant l'intruse ; fichier retiré → **vert**. `git status` propre après |
| 2026-09-13 | 0.B.7 (coupe A) | ce commit | `gofmt -l ./internal` ; `go vet` ; `go test ./internal/games/halo_infinite/film/replay/ ./internal/archlint/` | gofmt vide ; vet propre ; **les deux paquets ok** (12,2 s / 12,7 s) — budget **vert à 2,01 Mio**, `MatchCommitted` (manifeste et `pointsStride` compris), `CarryCurrentSchema` et `UnSeulJeuVivant` verts sur les 8 |
| 2026-09-13 | 0.B.7 (coupe A) | ce commit | `npx tsc -b --force` ; `npx eslint` (3 fichiers web) ; `npx vitest run src/features/match-replay src/lib/replay` ; `make go-api-lint` | tsc **exit 0** ; eslint **0** ; vitest **202 fichiers + 1 sauté, 3 124 tests + 3 sautés** ; lint Go **0 issues**. Seuil des 500 lignes tenu par extraction : `contract_fixtures_test.go` 433 L + `contract_fixtures_budget_test.go` 133 L |
| 2026-09-13 | 0.E | ce commit | recensement : `grep -rnE --include='*.go' '(Window\|Fenetre\|Tolerance\|Seuil\|Threshold\|Nearest\|Closest\|Majorit\|Infer\|Guess\|Devin\|Calibr\|Heuristi\|Repli\|Fallback\|MaxDist\|MinDist\|Epsilon\|Eps[A-Z]\|Detect\|Probe\|Auto)' <périmètre + filmdec> \| grep -v '_test\.go:'` puis lecture site par site | **509 hits** (321 hors `filmdec`) ; **528 sites de décision LUS** (fonction entière + appelant), **201 fichiers de production ouverts**, aucun modifié. Registre : **133 lignes** — (A) 8 · (B) 8 couvrant 40 sites · (C) 38 avec négatif mesuré · (D) non établi 17 · (E) replis ANONYMES 62. Vérification adverse par l'auditeur principal : **15 constats re-ouverts, 0 tombé, 2 amendés, 1 trou trouvé en plus** (`replay/identity_registry_section.go:288`), **1 écarté** (entrée périmée du registre des reports) |
| 2026-09-13 | 0.E | ce commit | contrôle « chaque `fichier:ligne` du registre existe » (script en §12 du registre) | **241 références distinctes, 241 résolues, 0 manquante** ; les 53 références de tête vérifiées en plus avec le CONTENU de la ligne. `git diff --stat` = le registre + ce plan, **aucun fichier de production** |


| 2026-09-13 | 0.D.0 | `215649efd` (arbre) | `replay-equiv -repo-root <worktree> -films <6 sous-ensembles> -update` | **20/20 figés** au commit `215649efd`, schéma 54, `# digest-grammar: 2`, 50 étapes chacun. **UNE SEULE étape bouge, `artifact`, sur 11 films** ; les 49 autres sont identiques sur les 20. Durées : 50247b26 2 min 29 · a521164d 1 min 30 · 60ae07c4 1 min 49 · 11de8353 1 min 12 · 111fa685 1 min 06 · e5adf7b2 1 min 18 · bcb6d393 25,6 s · 51101d1d 15,7 s · d9781168 1 min 14 · fb1a1a72 1 min 51 · 000d5950 41,5 s · 01e1f945 22,1 s · 64e8adfa 40,0 s · 7344d24f 26,0 s · 696a9d7c 21,7 s · 53ce4390 33,0 s · 9f57c612 26,2 s · 084a804d 2 min 19 · 1c4c63c2 3 min 09 · a349fea8 3 min 18 |
| 2026-09-13 | 0.D.0 (attribution) | mutation, arbre `215649efd` moins F.1 | `git checkout c45c411eb^ -- equipment_placements.go document_ground_weapon_items.go` puis `replay-equiv` SANS `-update` sur les 11 films qui bougent | **F.1 n'explique que 3 films** : `d9781168` rend l'ANCIENNE empreinte à l'octet (2 641 104 / `3eb6bce0`), donc F.1 y est la cause unique ; `a521164d` rend une TROISIÈME valeur (3 405 277 / `9f0ffac3`, entre l'ancienne 3 405 324 et la neuve 3 405 275) et `084a804d` de même (9 330 764, entre 9 330 766 et 9 330 762) — F.1 y vaut −2 octets. Les 8 autres (`51101d1d`, `bcb6d393`, `696a9d7c`, `7344d24f`, `53ce4390`, `64e8adfa`, `fb1a1a72`, `1c4c63c2`) sont IDENTIQUES aux références neuves : **F.1 n'y change rien** |
| 2026-09-13 | 0.D.0 (attribution) | mutation, arbre moins F.1 ET moins D.2 | `git checkout 3233ec2f8^ -- objectives.go coverage.go matchfacts.go` en plus, puis `replay-equiv` sans `-update` | **L'ANCIENNE empreinte est reproduite À L'OCTET sur les 10 films restants** : 51101d1d `7bc761b4` · bcb6d393 `00e82236` · 696a9d7c 2 223 087 · 7344d24f 2 352 860 · a521164d 3 405 324 `ed42d0f6` · 53ce4390 3 028 577 · 64e8adfa `e6f7ba81` · fb1a1a72 3 015 859 · 084a804d 9 330 766 · 1c4c63c2 6 864 012 `4898ac59`. **Attribution FERMÉE : F.1 + D.2 expliquent 100 % du mouvement, B.3 (`044751026`) n'a aucun effet.** `1c4c63c2` n'a fini qu'avec `-mem-gib 8` (11 min 58, pic 6,00 Gio — cf. §4 D2) |
| 2026-09-13 | 0.D.0 | ce commit | `go test ./internal/games/halo_infinite/film/replay/ -run GoldenBuildsAssemblyRegenerate -update-golden-builds-assembly` puis `git diff` | 7 goldens réécrits, **UN SEUL change** : `assembly_a521164d.golden`, 5 lignes, **exclusivement des origines de pose** (`16 deployee(s) · 210 lachee(s)` -> `14 · 212` ; `grenade_frag/deployed 12 -> 10`, `dropped 110 -> 112` ; deux poses nommées passent de `deployed` à `dropped`). Aucun film lu |
| 2026-09-13 | 0.D.0 | ce commit | `REPLAY_CONTRACT_UPDATE=1 go test ... -run ContractFixturesRegenerate -update` puis `git diff` | 8 fixtures réécrites, **UNE SEULE change** : `replay_schema_54_a521164d.json.gz` (191 488 -> 191 487 o compressés ; JSON 1 506 353 -> 1 506 351) et sa ligne de `manifest.json`. Les 7 autres identiques à l'octet. Total 2 108 043 o, sous le plafond de 3 145 728 ; 0 fixture d'une version périmée |
| 2026-09-13 | 0.D.0 (gates) | ce commit | `gofmt -l ./internal ./cmd` ; `go vet` (film, archlint, replaybuild, killcollector, objectiveevents) | sortie vide ; 0 diagnostic |
| 2026-09-13 | 0.D.0 (gates) | ce commit | `go test ./internal/games/halo_infinite/film/... ./internal/archlint/ ./internal/replaybuild/ ./internal/sync/killcollector/ ./internal/analysis/objectiveevents/ ./internal/domain/replaydoc/` | **12 paquets ok, exit 0** — filmdec 65,3 s · replay 67,0 s · archlint 133,8 s · replaybuild 3,3 s · killcollector 3,6 s · objectiveevents 4,7 s. `TestGoldenBuildsAssembly` et `TestContractFixturesMatchCommitted`, ROUGES à l'entrée du lot, sont verts |
| 2026-09-13 | 0.D.0 (gates) | ce commit | `golangci-lint run --timeout 20m --new-from-merge-base=origin/main` (cache ET dossier temporaire isolés) | **0 issues, exit 0** — baseline non accrue. NOTE : `make go-api-lint` échoue sur ce poste quand un AUTRE agent lint en parallèle (`%TEMP%/golangci-lint.lock` est global, `GOLANGCI_LINT_CACHE` ne l'isole pas), puis sur son `--timeout 5m` quand la machine est chargée |
| 2026-09-13 | 0.D.0 (gates web) | ce commit | `npm ci` (absent du worktree) puis `make check-types` ; `npx vitest run src/features/match-replay src/lib/replay` | tsc **exit 0** ; vitest **202 fichiers + 1 sauté, 3 130 tests + 3 sautés**. 3 tests de `replaySoundAssets.guard.test.ts` sortent en `Test timed out in 5000ms` pendant que `npm`/`golangci-lint` chargent la machine ; rejoués SEULS : **26/26 verts** — flottement de charge, hors diff (sons) |
| 2026-09-13 | 0.D.0 (gate de fin) | ce commit | `replay-equiv -repo-root <worktree>` sur les 20 films, 5 sous-ensembles, machine au repos | **20/20 IDENTIQUES**, exit 0 partout — dont **les 10 films du régime court** (50247b26, a521164d, 60ae07c4, 11de8353, 111fa685, e5adf7b2, bcb6d393, 51101d1d, d9781168, fb1a1a72). Pics 0,08 à 0,74 Gio, tous très en dessous du plafond : `1c4c63c2` à **0,74 Gio** ici contre 3,79 sous charge (§4 D2) |


| 2026-09-13 | 0.D.1 | ce commit | `FILM_CACHE_ROOT=<principal>/data/cache D6_FILMS=fb1a1a72,64e8adfa,d9781168 go test ./internal/analysis/objectiveevents/ -run D6MatiereDesManches -v` | **`fb1a1a72` : `RealRounds = [0]`.** Manches déclarées 0, 2 et 5 — **la manche 1 n'a AUCUN enregistrement**, ni joueur ni équipe. Manche 2 : 148 enregistrements sur une fenêtre [66 671, 814 115] qui recouvre celle de la manche 0 ([3 411, 800 871]) à **124/148**, densité **0,20 enr./s contre 1,26** (16 %). Manche 5 : UN enregistrement. Composant de score de mode : **2 sur 1 153**, donc `runs` = 1 en manche 0 et **0 en manches 1 et 2**. Témoins : `64e8adfa` (CTF, 2 manches) manche 1 = [760 501, 837 364], densité 140 % ; `d9781168` (Oddball, 3 manches) manches 1 et 2 sur segments propres, densité 313 % et 311 % |
| 2026-09-13 | 0.D.1 | ce commit | MUTATION DANS L'INSTRUMENT : `contiguousRounds(runs, material, toutPresent)` — la garde `vue && !present[round]` (`bb06cce5a`) neutralisée sans toucher une ligne de production | `fb1a1a72` rend **`[0 1 2]`, exactement l'ancien 3** : la garde est la cause UNIQUE de la baisse. Les deux témoins multi-manche réels rendent `[0 1]` et `[0 1 2]` **des deux côtés** — la garde ne coûte rien à une vraie manche |
| 2026-09-13 | 0.D.1 | ce commit | Oracles produit : `fb1a1a72.facts.json` et `config/titles/halo_infinite/mappings/regulation.toml` | Feuille : `teamScores [0,1]`, `gameVariantName "CTF:Arena"` — un total de captures. `[rounds_decide]` ne liste que les trois Oddball et son en-tête EXCLUT nommément `CTF:Arena` (« deux MI-TEMPS, pas des manches décisives [...] le compte de manches y vaut 0-1, ce qui serait un contresens ») |
| 2026-09-13 | 0.D.1 (gates) | ce commit | `gofmt -l ./internal ./cmd` ; `go vet` (film, archlint, replaybuild, killcollector, objectiveevents) ; `go test` (12 paquets) ; `golangci-lint --timeout 20m --new-from-merge-base=origin/main` | gofmt vide ; vet 0 diagnostic ; **12 paquets ok** (filmdec 8,9 s, archlint 14,0 s, objectiveevents 0,5 s) ; lint **0 issues, exit 0**. L'instrument SAUTE sans ses deux variables (`--- SKIP`) : il ne pèse pas sur la CI |
| 2026-09-13 | 0.D.1 (gate décodage) | ce commit | `git diff --stat HEAD -- 'apps/go-api/**/*.go' ':!*_test.go'` | **SORTIE VIDE : aucun octet de production ne change dans ce sous-lot** (un instrument `_research_test.go`, deux documents). Le régime court n'est donc PAS rejoué : `replay-equiv` ne dépend que du code de production, et 0.D.0 l'a laissé à 20/20 identiques au commit précédent. Gate consigné comme NON APPLICABLE, pas comme différé — une commande qui ne peut rien mesurer de neuf |


| 2026-09-13 | 0.D.2 | `554cf8339` (arbre) | `replay-corpus-gate --base=ebd012e3b` puis `--base=f22474816`, manifeste réduit à `60ae07c4`, `--source-root` = ce worktree, `--parc-root` = le principal | **Les deux rapports sont IDENTIQUES octet pour octet** (hors `dureeMs`) : schéma de référence 51, **66 gains, 28 pertes**, mêmes valeurs sur tous les axes. **La borne `maxUnrollPerStep = 16` (`f22474816`) n'explique RIEN sur ce témoin.** Durées 52,3 s et 88,0 s |
| 2026-09-13 | 0.D.2 | `554cf8339` (arbre) | `replay-corpus-gate --base=1a93b34f2` (parent de la porte) puis `--base=c5a71dcbd` (bump v53) | base 52 : **17 pertes**, 67 gains, exit 1 — base 53 : **0 perte, 3 gains, exit 0**. Entre les deux bases il n'y a que `fb71e9b3c` (la porte) et `c5a71dcbd` (le bump) : **la porte de région explique 100 % du bloc** |
| 2026-09-13 | 0.D.2 | `554cf8339` (arbre) | `--base=1a93b34f2 --work-root <scratch> --keep-work`, puis comparaison des DEUX artefacts cuits, bloc par bloc | **Rien de publié ne baisse** : `groundWeapons` 188 -> 188, `pickups` 297 -> 297, `kept` 218 -> 218, `objects` 218 -> 218. Pertes = compteurs d'ÉCHEC (`rejected` 211 -> 116 pour `accepted` 429 -> 334, −95 des deux côtés ; `unknown` 36 -> 5 ; `placements.unknown` 57 -> 4 ; `originUnknown` 56 -> 40 ; `truncated` 343 -> 3) + RE-CLASSIFICATION à somme constante (`spawned` 218 -> 35, `dropped` **0 -> 183**, somme 218). Gains : `placements` **57 -> 190**, `projectiles` publiés **236 -> 417**, `dropperNamed` 0 -> 183, `withOwner` 0 -> 186, `dated` 0 -> 31, `cycles` 0 -> 1 |
| 2026-09-13 | 0.D.2 | `554cf8339` (arbre) | `replay-corpus-gate --base=179bd7401` (la base même du constat D7) | **61 pertes**, 165 gains, schéma 34 -> 54. Les 17 de la porte s'y retrouvent, plus `geometry.*` (9 axes, chronique v52 props Forge, §7.A A1), `projectiles.p/n` et frères (4 axes, v52 pas impossible, §7.A A6), `tracks.points` (7 axes, = D8, lot 0.D.4) et **le bloc équipement/crâne/capacités (26 axes) qui n'apparaît PAS à la base 51** — d'où la troisième cause |
| 2026-09-13 | 0.D.2 | `554cf8339` | `git diff 1a93b34f2 HEAD -- data/titles/halo_infinite/reference/map_quant_bounds.json` ; `diff` de `film_context.go` aux deux révisions | **Les deux sont identiques.** Le catalogue impose `axisWidths [12 12 11]` et `regionIndexBits 2` à Live Fire (module `sgh_interlock`), et `FilmContext.I0Layout` n'auto-détecte que si l'entrée de carte est invalide : **aucune part des 190 records ne vient d'un découpage détecté**, sur aucune des deux révisions |
| 2026-09-13 | 0.D.2 (gates) | ce commit | `gofmt -l ./internal ./cmd` ; `go vet` ; `go test` (12 paquets) ; `golangci-lint --timeout 20m --new-from-merge-base=origin/main` | gofmt vide ; vet propre ; 12 paquets ok ; lint **0 issues, exit 0**. Régime court NON APPLICABLE : `git diff HEAD -- 'apps/go-api/**/*.go' ':!*_test.go'` **vide** — ce sous-lot ne change aucun octet de production (deux documents). Aucun worktree temporaire créé (voir le bloc 0.D.2), donc rien à retirer |


| 2026-09-14 | 0.D.5 | `2dad8d6df` (arbre) | `replay-equiv -repo-root <worktree> -films <6 sous-ensembles> -update`, puis `git diff` des 20 `.tsv` | **DEUX étapes bougent, et deux seulement** : `flag` sur **17 films** (compte inchangé à 1, contenu différent) et `artifact` sur **4** (`084a804d` +1 o, `111fa685` +1 o, `bcb6d393` +316 o, `e5adf7b2` +316 o). Les 48 autres étapes sont identiques sur les 20 films |
| 2026-09-14 | 0.D.5 (attribution) | mutation | revert de `flag_carries.go` + `flag_neutral.go` + `replaybuild/flagspawns.go` à `6e0e5378a^`, `replay-equiv` sans `-update` | `51101d1d` et `bcb6d393` rendent l'ANCIENNE empreinte de `flag` à l'octet (`1cd3a724…` et `869a2877…`) : **H.1 est la cause unique de l'étape `flag`** |
| 2026-09-14 | 0.D.5 (attribution) | mutation | revert de `equipment_placements.go` à `267fa1c5a^`, `go test -run TestGoldenBuildsAssembly` | `111fa685` redevient VERT ; les six autres builds inchangés. **H.2 est la cause unique de `111fa685`**. Diff du golden après régénération : `wall/deployed` 28 -> 29, `wall/dropped` 26 -> 25, une ligne `wall dropped 0x686b40c9 t=[1482, 1486]` devient `wall deployed` |
| 2026-09-14 | 0.D.5 (attribution) | mutation | revert des SIX fichiers de production de `film/replay` : l'écart de libellés PERSISTE ; élargissement à `internal/games/weapons/{labels.go,registry.go}` (`71aa37fcb`) | Avec ces deux fichiers revertés, `e5adf7b2` et `bcb6d393` redeviennent VERTS. **Le coupable était hors du paquet `replay`** — le chercher là où on l'attendait aurait conclu à tort. Diff du golden : exactement deux entrées, `0xD7915565` et `0xD791556542C9679F`, « Mutilator » / « Mutilateur » |
| 2026-09-14 | 0.D.5 (complétude) | mutation | revert des TROIS commits ensemble (H.1 + H.2 + G.7), `replay-equiv` sur `bcb6d393,111fa685,51101d1d` contre les anciennes références | **3 identiques, 0 différent** : les trois causes expliquent 100 % du mouvement, **zéro orphelin** |
| 2026-09-14 | 0.D.5 | ce commit | `-update-golden-builds-assembly` (sans film) ; `REPLAY_CONTRACT_UPDATE=1 … -run ContractFixturesRegenerate -update` | **3 goldens sur 7** changent (`111fa685`, `e5adf7b2`, `bcb6d393`), diff limité aux lignes attribuées ; **3 fixtures sur 8** (+2 / +26 / +23 o compressés), total 2 108 094 o sous le plafond de 3 145 728 |
| 2026-09-14 | 0.D.5 (incident) | — | `git checkout HEAD -- <paquet replay>` pour restaurer la production | **A aussi restauré `testdata/`** : 18 références fraîchement figées et 3 goldens effacés. Re-figeage rejoué EN ENTIER, diff **identique** (17 `flag` + 4 `artifact`, mêmes deltas) — déterminisme vérifié. Leçon : restaurer par fichiers NOMMÉS quand `testdata/` vit sous le paquet |
| 2026-09-14 | 0.D.5 (gate de fin) | ce commit | `replay-equiv` sur les 20 films, 5 sous-ensembles | **20/20 IDENTIQUES**, exit 0 partout, dont les 10 films du régime court. Pics 0,08 à 0,73 Gio |
| 2026-09-14 | 0.D.5 (gates) | ce commit | `gofmt` ; `go vet` ; `go test` (12 paquets) ; `golangci-lint --timeout 20m` ; `make check-types` ; `npx vitest run src/features/match-replay src/lib/replay` | gofmt vide ; vet propre ; **12 paquets ok** ; lint **0 issues, exit 0** ; tsc **exit 0** ; vitest **203 fichiers + 1 sauté, 3 126 tests + 3 sautés, 0 échec** |


| 2026-09-14 | 0.D.6 | `fc5db87f7` (arbre) | `replay-corpus-gate --base=b6b198baf^` (51), `b6b198baf` (52), `c5a71dcbd` (53), `104b74e15` (54), manifeste réduit à `bfecd02b` | **Les quatre rendent exactement les MÊMES deux pertes**, aucune sur un axe véhicule : `coverage.placements.deployed` 4 -> 1 et `byFamilyOrigin.grenade_frag/deployed` 3 -> — (règle d'origine des poses, F.1 / H.2, déjà instruite). 5 gains, 2 pertes, ~19 s par run |
| 2026-09-14 | 0.D.6 | `fc5db87f7` (arbre) | `replay-corpus-gate --base=7c85acf58^` (schéma **39**, naissance du calque véhicules) sur `bfecd02b` + `084a804d` | **97 pertes, ZÉRO sur un axe `vehicles.*`** — `bfecd02b` 15 (68 gains, 25,2 s), `084a804d` 82 (227 gains, 2 min 02). Répartition : `objectifs` 53, `carte` 19, `couverture` 11, `grenades` 9, `equipement` 5. Toutes rattachées à une famille déjà classée, sauf `coverage.flagCarries.teamBirths` 12 -> 11 (§4) |
| 2026-09-14 | 0.D.6 | `fc5db87f7` | lecture de l'artefact du parc `bfecd02b` (schéma 54, cuit le 13/09 à 00:43) — **non oracle, lecture seule** | 11 vies publiées, **`familyUnknown 9`, `unknownChassis {038df01a: 9}`** ; les 9 sans famille sont IMMOBILES (aucun `samples`, un `spawn`, `t0=0 t1=t1max=5032` = le match entier) ; les 2 autres sont `ghost` (`5b80c406`) et portent leurs échantillons |
| 2026-09-14 | 0.D.6 | `fc5db87f7` | `git log --oneline --all -S "038df01a"` ; `grep` dans `damagetag/data/labels.tsv` | Le châssis **n'a jamais figuré au dépôt** (aucun commit). `labels.tsv` porte **trois** entrées `vehi 038df01a` (l. 169, 343, 430), toutes `sb_003_lvl_moments_ge_shared_autoturret_banished` : **tourelle automatique bannie** — identification neuve, cohérente avec l'immobilité |
| 2026-09-14 | 0.D.6 | `fc5db87f7` | confrontation D13 de la règle d'effacement (`vehicleCensusTolUS`, `vehicle_tracks.go:33`) au témoin | La règle **prolonge, elle n'efface jamais avant la dernière preuve** : 10 vies sur 11 ont `t1 == t1max` = dernière frame ; la seule effacée est le `ghost` slot 777 à la **frame 2874 (287,4 s), 5,3 s APRÈS son dernier échantillon** (`t1` 2821), son relais 778 reprenant à 3526. **Aucun véhicule que le film montre encore n'est effacé** |
| 2026-09-14 | 0.D.6 | ce commit | `config/replay_corpus.toml` : entrée `bfecd02b` (famille `vehicules_v41_utilisateur`, Snowbound, Team Slayer:Arena) ; `go test ./internal/replaybuild/ ./cmd/replay-corpus-gate/` | corpus témoin de **12 à 13** ; les deux paquets **ok** |
| 2026-09-14 | 0.D.6 (gates) | ce commit | `gofmt` ; `go vet` ; `go test` (12 paquets) ; `golangci-lint --timeout 20m` | gofmt vide ; vet propre ; 12 paquets ok ; lint **0 issues, exit 0**. Régime court NON APPLICABLE : `git diff HEAD -- 'apps/go-api/**/*.go' ':!*_test.go'` **vide** — aucun octet de production ne change (un manifeste de corpus, deux documents) |


| 2026-09-14 | 0.D.1 bis | `7ffdc3f8b` (arbre) | Ghidra lecture seule : `decompile_function 140C18794` ; `get_xrefs_to` | Désérialiseur de l'archétype 6 : traite une PAIRE de slots de stat (`param_1+8`, `param_1+0xc`), lit **un champ de 5 bits par slot** rangé à `base + 4 + idx*8`, puis une valeur à longueur variable à `base + idx*8`, puis deux drapeaux vers le masque `base + 0x1c0`. Référencé **uniquement depuis des DONNÉES** (`145435cd8`, `143c96b38`) : c'est une entrée de vtable, pas un appel direct |
| 2026-09-14 | 0.D.1 bis | `7ffdc3f8b` | RE déjà au dépôt, relue sur pièces (`ETAT_DE_L_ART_MODE_SCORE_EVENEMENTS` §17.1) | Le registre ECS du film NOMME les 58 composants de l'archétype 6 : **0-27 `current-round-value`, 28-55 `finalized-rounds-values`, 56 `round-outcomes`, 57 `entry-index-and-type`**. Getter natif `Team_GetCurrentRoundStatValue` @ `0x142C6B118` : `world + statSlot*0x88 + teamIdx*0x1DF0 + 0x38 + round*4` — **la manche est une dimension du moteur**. NON prouvé : que le champ de 5 bits soit cette dimension |
| 2026-09-14 | 0.D.1 bis | `7ffdc3f8b` | instrument étendu (répartition par tranches de 60 s + recensement des composants), `D6_FILMS=fb1a1a72,d9781168` | **`fb1a1a72` manche 2 est BIMODALE** : `[0 50 44 0 0 0 0 0 0 0 0 0 13 41 0]` — 94 enregistrements à 60-180 s, **54 à 720-840 s**. La preuve (3) de 0.D.1 (« saupoudré, densité 16 % ») est RÉFUTÉE. Forme d'une vraie manche (`d9781168`) : blocs contigus et DISJOINTS. **Composants** : la manche 2 porte **55 lectures de `finalized-rounds-values` (28-55)**, la manche 0 du même film en porte **ZÉRO** |
| 2026-09-14 | 0.D.1 bis | `7ffdc3f8b` | contrôles de la MÊME variante, `D6_FILMS=53ce4390,51101d1d` | `53ce4390` (CTF:Arena, ~780 s) et `51101d1d` (CTF:Arena Neutral Flag, ~270 s) : **`RealRounds = [0]`, aucune manche déclarée au-delà de 0**, 1 et 0 lecture finalisée. `fb1a1a72` (814 s pour 720 s réglementaires) est le seul des trois à écrire un « 2 », et son amas tardif tombe dans le dépassement : **piste PROLONGATION soutenue, pas prouvée** |
| 2026-09-14 | 0.D.1 bis | ce commit | `config/titles/halo_infinite/mappings/regulation.toml` l. 198 ; `.ai/V7.5/REGISTRE_REPORTS.md` | Commentaire corrigé (« deux MI-TEMPS » -> manches / prolongations, source = décision utilisateur du 2026-09-13), motif d'exclusion inchangé. Registre : la ligne 0.D.1 devient « **VERDICT RETIRÉ — non établi, rouvert** » ; ligne neuve qui NOMME `contiguousRounds` comme heuristique D13 avec son contrat D14 (déclenchement typé, compteurs à publier, critère de retrait, date de pose 2026-08-18, cible clôture M1) |
| 2026-09-14 | 0.D.1 bis (gates) | ce commit | `gofmt` ; `go vet` ; `go test` (12 paquets) ; `golangci-lint --timeout 20m` | gofmt vide ; vet propre ; 12 paquets ok ; lint **0 issues, exit 0**. Régime court et corpus gate NON APPLICABLES : `git diff HEAD -- 'apps/go-api/**/*.go' ':!*_test.go'` **vide** — aucun octet de production ne change (un instrument, un commentaire TOML, deux documents) |


| 2026-09-14 | 0.D.3 (inventaire) | `2a44c9031` (arbre) | `TestGoldenInputsFidelite` (neuf) sur les 8 builds, `REPLAY_FILM_CACHE` posé — AVANT tout codage | **DEUX familles d'écart, pas une.** (i) « lecture(s) portent le rang SÉLECTIONNÉ », relu > frais sur **les 8 builds** : 150/194 · 75/379 · 100/348 · 76/565 · 107/570 · 109/605 · 36/136 · 130/346. (ii) origines de pose sur `fb1a1a72` seul : 20/319 frais contre 17/322 relu, 479 lignes sur 584 décalées |
| 2026-09-14 | 0.D.3 | ce commit | lecture struct / encodeur, puis diff `reflect` fresh vs relu restreint à `Placements` / `PlacementStats` | **`Placements` est IDENTIQUE** : le défaut n'est pas dans la liste des poses. Trois trous : `KeyframeInventory.SelectedGrenadeRank` (vaut **-1** = pas de sélection, relu **0** = Fragmentation sélectionnée) et `GrenadesByPosition` non sérialisés ; `InventoryDelta.Ammo` non sérialisé ; coordonnées **arrondies au centimètre** alors qu'`equipmentOwner` choisit le poseur par la plus courte distance (poseur 524 -> 514 sur trois poses de `fb1a1a72`) |
| 2026-09-14 | 0.D.3 | ce commit | codec : `SelectedGrenadeRank` + `GrenadesByPosition` ; `encodeDeltaAmmo` / `decodeDeltaAmmo` ; X/Y/Z en OU-EXCLUSIF des bits `float32` ; magie `REPLAYINPUTS14` -> `15`, garde recalée sur `14` | `gofmt` vide, `go vet` propre. La garde de version refuse toujours un fixture de la version précédente POUR CE QU'IL EST |
| 2026-09-14 | 0.D.3 | ce commit | `TestGoldenInputsRegenerate -update` (000d5950) puis `TestGoldenBuildsRegenerate -update` (7 builds), un décodage à la fois | 8 fixtures réécrites. **Taille : 10,4 Mio -> 17,8 Mio (+7,8 Mio, +72 %)** — coût de la fidélité des coordonnées, seule des trois corrections qui pèse |
| 2026-09-14 | 0.D.3 (preuve) | ce commit | `TestGoldenInputsFidelite` rejoué après régénération | **8/8 PASS, exit 0** (183,6 s). Étape intermédiaire mesurée : après les deux premiers trous comblés, 7 builds sur 8 verts et `fb1a1a72` seul rouge sur les origines — c'est ce reste qui a mené aux coordonnées |
| 2026-09-14 | 0.D.3 | ce commit | `-update-golden-builds-assembly` + `TestGoldenAssembly -update` ; `ContractFixturesRegenerate` (double porte) | **Les 8 goldens changent, tous dans le sens de la production** : `bcb6d393` 136 -> **36** lectures à rang sélectionné, `fb1a1a72` 346 -> **130**, et ses trois poses reprennent `deployed` avec leur vrai poseur (524). Fixtures de contrat : 2 108 186 o, **sous le plafond de 3 Mio** |
| 2026-09-14 | 0.D.3 (gates) | ce commit | `gofmt` ; `go vet` ; `go test` (12 paquets) ; `golangci-lint --timeout 20m` ; `make check-types` ; `npx vitest run src/features/match-replay src/lib/replay` | gofmt vide ; vet propre ; **12 paquets ok** ; lint **0 issues, exit 0** ; tsc **exit 0** ; vitest **203 fichiers + 1 sauté, 3 126 tests + 3 sautés, 0 échec**. `git diff HEAD -- 'apps/go-api/**/*.go' ':!*_test.go'` **vide** |


| 2026-09-14 | 0.D.3 bis | ce commit | codec : positions en QUANTA `Q` (delta-varint par slot), module de carte en tête du blob + erreur typée `errGoldenInputsCarte`, `decodeGoldenInputs(blob, entry)` ; magie `REPLAYINPUTS15` -> `16` | `gofmt` vide, `go vet` propre. L'entrée de catalogue arrive en PARAMÈTRE, jamais devinée ; un fixture cuit pour une autre carte est refusé POUR CE QU'IL EST |
| 2026-09-14 | 0.D.3 bis | ce commit | première tentative : déquantifier avec le découpage du CATALOGUE | **`60ae07c4` ROUGE, et lui seul** : `x [-8,56 ; 37,04]` frais contre `x [-0,39 ; 90,80]` relu — étendue exactement DOUBLE. Cause : le chemin du fixture AUTO-DÉTECTE le découpage (`ScanFilmOptions.Layout` nul) et la détection rend **[13 12 11]** sur Live Fire quand le catalogue dit **[12 12 11]** ; un bit sur X double le pas |
| 2026-09-14 | 0.D.3 bis | ce commit | correctif : le blob porte les 3 largeurs EMPLOYÉES ; `scan.Layout` posé explicitement (même valeur, même fonction) | `TestGoldenInputsFidelite` **8/8 PASS, exit 0** (220 s). Coût du champ : **3 varints par fixture (~3 octets)** |
| 2026-09-14 | 0.D.3 bis (preuve) | ce commit | `git diff --stat 1b1111379 -- testdata/assembly_ fixtures/go/` | **SORTIE VIDE** : goldens d'assemblage et fixtures de contrat **identiques à l'octet** à la version en flottants. Le contenu cuit est le même, seule sa représentation au fixture change |
| 2026-09-14 | 0.D.3 bis (taille) | ce commit | `git cat-file -s` sur les 8 blobs aux trois états | origine `2a44c9031` **10 849 119 o (10,35 Mio)** · flottants `1b1111379` **18 656 453 o (17,79 Mio, +72 %)** · **QUANTA 10 337 525 o (9,86 Mio)** — soit **−511 594 o sous l'origine** et −8,0 Mio sous les flottants |
| 2026-09-14 | 0.D.3 bis (gates) | ce commit | `gofmt` ; `go vet` ; `go test` (12 paquets) ; `golangci-lint --timeout 20m` ; `make check-types` ; `npx vitest run src/features/match-replay src/lib/replay` | gofmt vide ; vet propre ; **12 paquets ok** ; lint **0 issues, exit 0** ; tsc **exit 0** ; vitest **3 126 tests, 0 échec** |


| 2026-09-14 | 0.D.4 | `a35c9e678` (arbre) | `replay-corpus-gate` sur `d9781168` seul (manifeste réduit), **huit points de base** : 179bd7401 (34), 48cf4905d^ (35), 84a53c2cc (36 avant), 48cf4905d (36 après), fa09f4ee5^ (37), e6455cab6^ (38), 7c85acf58^ (39), 79bf2e6d2^ (40), 07236bf88^ (41), b07525b3f^ (43), ceb0f3d2a^ (47) | **Perte PRÉSENTE** aux bases 34, 35 et `48cf4905d^` (`tracks.points/n` 36 581 -> 36 579, 7 axes) ; **ABSENTE** à `48cf4905d` et à toutes les bases suivantes. **La cause est `48cf4905d` (schéma 36, « une track = une vie »), et elle seule.** L'hypothèse du plan (v41/v43/v47) nommait la bonne mécanique, quinze schémas trop tard. ~25 s par run |
| 2026-09-14 | 0.D.4 | `a35c9e678` | `--keep-work` à la base `48cf4905d^`, puis comparaison des DEUX artefacts cuits, point par point sur la clé (slot, t) | **Deux points, identifiés à l'unité** : `slot 558 @ 2168` et `slot 573 @ 2730`, même xuid `2535435655459376`. **0 point neuf**, 0 doublon des deux côtés. Chacun est le PREMIER point de sa piste et il est ISOLÉ : point suivant du même slot à **+88 frames (8,8 s)** et **+176 frames (17,6 s)**, très au-delà de `lifeGapUS` (5 s). Pistes **160 -> 174** : les positions sont redistribuées, pas perdues |
| 2026-09-14 | 0.D.4 | `a35c9e678` | lecture sur pièces des deux règles qui composent | `build.go:526` ouvre une nouvelle vie au-delà de `lifeGapUS` (`lives.go:46`, 5 s) ; `build.go:575` refuse la vie par `DefaultMinPoints = 2` (`build.go:16` : « une track d'un seul échantillon n'est pas une trajectoire »). Les deux précèdent le constat, aucune n'est en défaut. **`grep` du paquet : AUCUN compteur de refus `minPoints`** — la perte est muette |
| 2026-09-14 | 0.D.4 | `a35c9e678` | second témoin `60ae07c4` aux bases `48cf4905d^` et `48cf4905d` | **NON MESURABLE** : `ERREUR : cuisson reference ... exit status 13`, `plafond memoire depasse pic_gio=4,15`. Le code de l'époque du schéma 36 ne cuit pas cet ex-bombe — c'est celui que la borne `f22474816` a dompté quinze schémas plus tard. Dit, pas contourné |
| 2026-09-14 | 0.D.4 (gates) | ce commit | `gofmt` ; `go vet` ; `go test` (12 paquets) ; `golangci-lint --timeout 20m` | gofmt vide ; vet propre ; 12 paquets ok ; lint **0 issues, exit 0**. Régime court et corpus gate de clôture NON APPLICABLES : `git diff HEAD -- 'apps/go-api/**/*.go' ':!*_test.go'` **vide** — aucun octet de production ne change (deux documents) |


| 2026-09-14 | 0.D.7 | ce commit | `decodeFilmInputsForEntry` : le découpage vient de `filmdec.NewFilmContextForMap(nil, &entry, nil).ImposedLayout()` — la fonction de la production, pas `entry.Layout()` | `gofmt` vide, `go vet` propre. L'auto-détection ne subsiste que sur entrée de carte invalide (règle `resolveI0Layout`), et elle est alors NOMMÉE dans le blob (`LayoutDetected`). Sur les 8 builds : `LayoutDetected = false` partout |
| 2026-09-14 | 0.D.7 | ce commit | erreur typée `errGoldenInputsDecoupage` : blob « catalogue » dont les largeurs ne sont plus celles que la règle tranche | Le fixture ne peut plus se déquantifier avec un pas devenu faux sans le dire. Magie `REPLAYINPUTS16` -> `17`, garde recalée sur la précédente |
| 2026-09-14 | 0.D.7 | ce commit | régénération des 8 `inputs_*.bin.gz` (un décodage à la fois), puis goldens (porte nommée) et fixtures de contrat (double porte) | **UN SEUL golden change : `assembly_60ae07c4.golden`** (6 lignes) — les 7 autres identiques à l'octet. **UNE SEULE fixture de contrat change** : `replay_schema_54_60ae07c4.json.gz` (172 773 -> 174 897 o) + sa ligne de manifeste ; total 2 110 310 o, sous le plafond de 3 Mio |
| 2026-09-14 | 0.D.7 (chiffres) | ce commit | diff du golden `60ae07c4` | **`x [-8,56 ; 37,04]` -> `x [-12,90 ; 27,56]`** · pistes **174 -> 173** · points de grille **45 137 -> 45 133** · vies publiées **174 -> 173** · `y [13,35 ; 47,79]` -> `[15,13 ; 47,79]`, `z [-5,30 ; 7,86]` -> `[-5,30 ; 5,05]`. Détection `gate=5 region=0 13/12/11` contre catalogue `gate=6 region=1 12/12/11` : les champs d'axe changent de décalage ET la porte de région teste ses deux bits |
| 2026-09-14 | 0.D.7 (preuve) | ce commit | `TestGoldenInputsFidelite` | **8/8 PASS, exit 0** (179,7 s) : fraîches == relues sur les 8 builds, sous la règle du catalogue |
| 2026-09-14 | 0.D.7 (gates) | ce commit | `gofmt` ; `go vet` ; `go test` (12 paquets) ; `golangci-lint --timeout 20m` ; `make check-types` ; `npx vitest run src/features/match-replay src/lib/replay` | gofmt vide ; vet propre ; **12 paquets ok** ; lint **0 issues, exit 0** ; tsc **exit 0** ; vitest **3 126 tests, 0 échec**. `git diff HEAD -- 'apps/go-api/**/*.go' ':!*_test.go'` **vide** |


| 2026-09-14 | 0.D revue R1 | ce commit | **7 corrections dans le lot** (R1-1, R1-2, R1-4, R1-5, R1-6, R1-7, R1-8) + 1 découverte consignée (D7) et planifiée (lot 1.0) | 1 P1 + 7 P2 recevables, 0 jeté. 21 conditions du relecteur tiennent |
| 2026-09-14 | R1-1 (P1) | ce commit | le blob porte les VERDICTS de balayage : `InventoryDeltaAmmoRefused` et `FilmMajorVersion` ; `options()` les câble ; la fidélité les compare | **LA CONSTANTE ÉTAIT FAUSSE** : après régénération, « canal munitions refuse » passe de `false` à **`true` sur 5 des 8 goldens** (`a521164d`, `11de8353`, `111fa685`, `bcb6d393`, `e5adf7b2`). **Mutation** (`!dStats.AmmoRefused`) : fidélité **ROUGE** sur `bcb6d393`, ligne nommée « canal munitions refuse : false / true » |
| 2026-09-14 | R1-2 (P2) | ce commit | la contradiction blob / catalogue est vérifiée DANS LES DEUX SENS ; « détecté » sur une carte à entrée valide devient `errGoldenInputsDecoupage` | **Mutation** (encodeur forçant `LayoutDetected = true`) : round-trip **ROUGE** sur les 7 builds, message « fixture dit son decoupage [12 12 11] AUTO-DETECTE, or le catalogue en impose un » |
| 2026-09-14 | R1-6 (P2) | ce commit | retrait de `InventoryDelta.Ammo` et `KeyframeInventory.GrenadesByPosition` — portés par le codec, lus par aucun assemblage | Fixtures **10 337 463 -> 10 323 769 o (−13 694 o)**. Fidélité 8/8 toujours verte : le retrait ne change aucun assemblage, ce qui EST la preuve qu'ils n'étaient pas lus |
| 2026-09-14 | R1-8 (P2) | ce commit | `TestGoldenAssembly -update` et `TestGoldenInputsRegenerate` : `t.Logf` -> `t.Fatalf` nommé | Les quatre portes de régénération du paquet se comportent désormais pareil : aucune ne rend `ok` après avoir réécrit |
| 2026-09-14 | R1-4, R1-5 (P2) | ce commit | en-têtes de `golden_builds_test.go` et `contract_fixtures_test.go` réécrits (D9 COMBLÉE, chiffres datés en note historique) ; journal des versions complété v15, v16, v17, v18 | Plus de doc inversée : le fichier ne décrit plus un défaut corrigé comme courant |
| 2026-09-14 | R1-7 (P2) | ce commit | scission par responsabilité et découpage des deux fonctions de ~290 L | `golden_inputs_test.go` **1 560 L -> 493 L** ; 5 fichiers neufs, **tous < 500 L** (codec 311, encode 362, decode 308, film 190, fidélité 118) ; `encodeGoldenInputs` **294 L -> 12 L** + 8 sous-fonctions (38, 53, 36, 36, 53, 45, 27, 23) ; `decodeGoldenInputs` **290 L -> 16 L** + 8 sous-fonctions. **DÉPLACEMENT PUR** : round-trip et fidélité verts après scission |
| 2026-09-14 | 0.D revue R1 (gates) | ce commit | `gofmt` ; `go vet` ; `go test` (12 paquets) ; `golangci-lint --timeout 20m` ; `make check-types` ; `npx vitest run` ; `TestGoldenInputsFidelite` | gofmt vide ; vet propre ; **12 paquets ok** ; lint **0 issues, exit 0** ; tsc **exit 0** ; vitest **203 fichiers, 3 126 tests, 0 échec** ; **fidélité 8/8 PASS**. `git diff HEAD -- 'apps/go-api/**/*.go' ':!*_test.go'` **VIDE** — régime court non applicable, aucun octet de production ne bouge |


| 2026-09-14 | 0.D revue R2 | ce commit | **revue ronde 2 (corrections seules) : 0 P1 + 3 P2, 0 jeté, 3 corrigés.** 14 conditions du relecteur tiennent ; la boucle converge (1 P1 -> 0), pas de ronde 3 | — |
| 2026-09-14 | R2-1 (P2) | ce commit | la moitié `FilmMajorVersion` de R1-1 n'avait AUCUN gate : `renderAssembly` ne l'imprimait pas (0 occurrence dans les 8 goldens) et l'erreur de `filmsource.LoadDir` était jetée en silence | (i) `renderAssembly` publie la ligne « version majeure du film » ; les 8 goldens régénérés par la porte nommée, **diff = cette seule ligne, +1 par golden** — et les valeurs recoupent la table du corpus (33 · 37 · 38 · 39 · 40 ×2 · 41 ×2). (ii) le chemin frais rend une erreur nommée au lieu de se taire. **Mutation** (`_ = v`) : fidélité **ROUGE** sur `bcb6d393`, « NON LUE » contre « 40 » |
| 2026-09-14 | R2-2 (P2) | ce commit | `TestDocumentShapeRegenerate` réécrivait et rendait `ok` ; le commentaire de R1-8 annonçait une parité qui n'existait pas | `t.Fatalf` nommé ; commentaire corrigé (« les CINQ autres portes », comptées). **Preuve par la commande documentée** sans `-v` : réécrit, **sort en échec nommé**, md5 du golden **identique avant/après** (`72d361fa…`), rien à restaurer |
| 2026-09-14 | R2-3 (P2) | ce commit | le journal se contredisait : v15 disait faire entrer `InventoryDelta.Ammo` « parce que l'assemblage le lit », v18 disait l'inverse ; l. 82 l'attribuait à v16 | v15 réécrite au passé avec la mention datée « entré AU MÊME GESTE, mais À TORT […] RETIRÉ en v18 » ; attribution corrigée en « entré en v15 » |
| 2026-09-14 | 0.D revue R2 (gates) | ce commit | `gofmt` ; `go vet` ; `go test ./internal/games/halo_infinite/film/replay/ ./internal/archlint/` ; `TestGoldenInputsFidelite` ; régénération de contrôle des fixtures de contrat | gofmt vide ; vet propre ; **2 paquets ok** (replay 13,1 s, archlint 11,0 s) ; **fidélité 8/8 PASS** ; **fixtures de contrat INCHANGÉES** (la version majeure y était déjà depuis R1 — seul le golden ne l'imprimait pas) |
| 2026-09-14 | 1.0.1 | `33cd1dbcf` | `BuildFromFilm` = `scanFilmInputs` + `BuildFromPositions` ; la séquence des 27 balayages passe dans `film_scan.go` (5 phases ; 4 d'entre elles appellent une sous-phase, 5 sous-phases en tout), le type d'entrées dans `film_inputs.go` | `build_from_film.go` **442 L -> 149 L**, `film_scan.go` 421 L, `film_inputs.go` 176 L — **tous < 500 L** ; fonction la plus longue **66 L** (`balayerPositions`), toutes ≤ 80 L ; 0 `var` de paquet neuve ; verrou et largeurs d'axe INCHANGÉS dans `BuildFromFilm` (`world_object_precision_guard_test.go` non modifié) |
| 2026-09-14 | 1.0.1 | `33cd1dbcf` | `observe_test.go` réécrit : il DESCEND de `scanFilmInputs` dans les phases (`fichiersDuBalayage`), s'arrête aux balayages (feuilles) | **36 étapes dans l'ordre == `BuildFromFilmSteps`, 27 balayages == 27 étapes hors `.stats`** — identiques à avant le découpage. Second cas ajouté : `TestBuildFromFilmNeBalaiePlusLuiMeme` (0 étape, 0 balayage dans `BuildFromFilm`) |
| 2026-09-14 | 1.0.1 (inventaire) | `33cd1dbcf` | inventaire CHAMP PAR CHAMP de `FilmInputs` (35 champs) contre le codec, AVANT de coder | **31 transportés, 4 nommés absents** (`FlagMarks`, `ZoneReads`, `ZoneScanned`, `BombReads` : calques gardés par `Options.Flag`/`.Zone`/`.Bomb`, vides des deux côtés au fixture). SIX canaux manquaient : les cinq du plan + **`ZoomEvents`**, d'où sort `Options.Scoped` (la copie n'avait pas la lunette) |
| 2026-09-14 | 1.0.1 (3e divergence) | `33cd1dbcf` | lecture sur pièces de `installWorldObjectPrecision` contre le geste du fixture | Le fixture posait `SetWorldObjectPrecisionFromLayout(I0Layout{AxisW: entry.AxisWidths})` : **ni `IndexW` ni `Region`**, là où `entry.Layout()` porte les trois. Sur Live Fire (région 1 sur 2 bits) le fixture lisait ses objets du monde **un bit trop tôt** — le défaut corrigé en production le 2026-09-12 (lot B-bis). Le fixture appelle désormais `installWorldObjectPrecision` |
| 2026-09-14 | 1.0.2 | `33cd1dbcf` | six codecs (encodeur collé à son décodeur, `golden_inputs_canaux_test.go`) ; `encodePositionSection` devient le SEUL codec de positions, partagé avec le nuage des véhicules ; magie **`REPLAYINPUTS18` -> `REPLAYINPUTS19`**, garde de version recalée, journal des versions complété | `TestGoldenInputsVersionGuard` PASS ; `TestGoldenInputsRoundTrip` et `TestGoldenBuildsInputsRoundTrip` PASS (8/8) |
| 2026-09-14 | 1.0.2 (garde neuve) | `33cd1dbcf` | `TestCodecCouvreFilmInputs` : réflexion sur `FilmInputs`, aller-retour par champ, chaque champ doit être transporté OU nommé dans `champsNonTransportes` | **35 sous-tests, 0 sauté, PASS.** Il ne lit AUCUN octet de film : il tourne en CI, là où `TestGoldenInputsFidelite` saute |
| 2026-09-14 | 1.0.3 (a) | arbre à `33cd1dbcf` | `go run ./cmd/replay-equiv -repo-root <worktree> -films …` — régime court, DEUX sous-ensembles de 5 | **10 identiques, 0 différent, 0 échec** (`50247b26 a521164d 60ae07c4 11de8353 111fa685` 3 min 30 ; `e5adf7b2 bcb6d393 51101d1d d9781168 fb1a1a72` 1 min 58). LE PAS STRUCTUREL NE CHANGE PAS UN OCTET du document cuit (D4) |
| 2026-09-14 | 1.0.3 (b) | `20a39f481` | `REPLAY_FILM_CACHE=… go test -run TestGoldenInputsFidelite/<build>` — un processus par build (jamais 8 décodages dans un process) | **8/8 PASS** (10,7 / 29,2 / 19,4 / 23,2 / 24,2 / 28,4 / 8,8 / 21,6 s) : assemblage sur entrées FRAÎCHES == assemblage sur entrées RELUES, les six canaux neufs compris |
| 2026-09-14 | 1.0.3 (c) | `20a39f481` | régénération : 8 `inputs_*.bin.gz` (un film par processus), puis `assembly_*.golden` par leurs DEUX portes nommées, puis fixtures de contrat | Fixtures **10 323 769 -> 11 044 407 o (+7,0 %)** ; contrat **2 563 766 o** (plafond 3 145 728). **TOUT ÉCART EST UN GAIN**, deux familles : (1) le lien direct corps -> joueur — traces sans identité `000d5950` 6 -> 0, `a521164d` 46 -> 1, `e5adf7b2` 35 -> 1, `11de8353` 24 -> 0, `111fa685` 20 -> 0 ; tirs rattachés 483 -> 504, 874 -> 1186, 1201 -> 1492, 3037 -> 3275 ; deux verdicts `non publiable : un slot change de porteur` (collisions de slot) DISPARAISSENT ; (2) les largeurs d'axe world-object sur Live Fire — poses 57 -> 190 (dont 186 avec poseur contre 0), projectiles publiés 236 -> 417 et coupures 343 -> 3, socles enfin publiés, deux lectures de capacité fantômes (rangs 2 et 19) retirées. Les ramassages natifs datent les occupations de socle (0 -> 40, 0 -> 15, 0 -> 33, 0 -> 15) |
| 2026-09-14 | 1.0.3 (chiffres) | `20a39f481` | recalibrage des constantes du chantier, avec date et cause | `wantShotsAttached` **483 -> 504** /519 ; `wantClosedByRespawn` **3 -> 0** (le pont vient à 100 % de la LECTURE : 99/99 entrées contre 90/93 — il ne reste rien à déduire) ; `wantGrenades` **69 -> 70** /70. `wantLivesNamed` (90/105), `wantProjectiles` (436), `wantInventory` (184) et `wantIndexReadings` (26) NE BOUGENT PAS |
| 2026-09-14 | 1.0.4 | ce commit | `coverage.tracks` (`published`, `publishedPoints`, `refusedMinPoints`, `refusedPoints`, `minPoints`) ; `SchemaVersion` **54 -> 55** ; chronique + garde de `structure_test.go` + empreinte de forme + jumeau `domain/replaydoc` + projection `replayview` + `openapi.yaml` + `generated.ts` | `DefaultMinPoints` **reste à 2** (décision utilisateur en attente). `TestDocumentShape*` PASS (empreinte `6a8f8714a6f45638`, schéma 55) ; `replayview/parity_test.go` PASS ; `make openapi-gen` +27 lignes (schéma `TrackCoverage`), `generated.ts` +13 lignes |
| 2026-09-14 | 1.0.4 (équivalence) | ce commit | régime court, DEUX sous-ensembles de 5 | **10 films : ÉCART À LA SEULE ÉTAPE `artifact`**, les 49 étapes de balayage identiques. Delta d'octets = exactement la longueur du champ neuf (94 o fixes + les chiffres) : **+104** sur huit films, **+103** sur `bcb6d393` (2 chiffres de `published`), **+102** sur `51101d1d` (2 + 4 chiffres) |
| 2026-09-14 | 1.0.4 (corpus gate) | ce commit | `go run ./cmd/replay-corpus-gate --source-root <worktree> --parc-root …/LevelUp-go-migration --base=50686abda --manifest <toml réduit, hors dépôt>` | `d9781168` oddball : **schéma 54 -> 55, 7 gains, 0 PERTE, statut ok** (24,7 s) |
| 2026-09-14 | 1.0.4 (attribution) | ce commit | `--keep-work`, puis diff CHAMP À CHAMP des deux artefacts cuits (base et HEAD) | **6 différences, toutes nommées** : `coverage.tracks.{published=174, publishedPoints=36579, refusedMinPoints=2, refusedPoints=2, minPoints=2}` absents de la base, et `schemaVersion 54 -> 55`. **AUCUNE autre valeur ne bouge** |
| 2026-09-14 | 1.0 (gates communs) | ce commit | `gofmt -l ./internal ./cmd` ; `go vet` (5 paquets) ; `go test` (14 paquets, dont `replayview` et `api`) ; `golangci-lint run --timeout 20m --new-from-merge-base=origin/main` ; `make check-types` ; `make test-web` | gofmt VIDE ; vet propre ; **14 paquets ok** ; lint **0 issues** (1 min 33) ; `tsc -b` propre ; vitest **711 fichiers, 7 629 tests passés, 1 fichier / 17 tests skippés** (1 min 43) |

| 2026-09-14 | 1.0 revue R1 | ce commit | **revue adversariale ronde 1 : 0 P0, 0 P1, 5 P2 recevables + 1 exigence de plafond, 0 jeté, 6 corrigés** ; 22 conditions tiennent (séquence comparée appel par appel, 49 étapes de balayage identiques à l'octet sur 5 films, codec et schéma 55 prouvés par mutation dans les deux sens) | 2 découvertes de plus consignées en §4 (D4 allocations du décodeur de blob, D5 `applyTo` sans garde) |
| 2026-09-14 | R1-1 (P2) | ce commit | **CINQUIÈME divergence fixture / production** : le fixture passait `RosterXUIDs` NUL (`rosterOf(deaths, nil)` contre `rosterOf(deaths, feuille)`), donc un joueur à ZÉRO MORT manquait à la table d'index — et la fidélité était aveugle (mêmes options des deux côtés). La règle descend dans `replay.RosterXUIDsOf` (fichier `roster_xuids.go`), `replaybuild.rosterXUIDs` n'en est plus que l'adaptateur, le fixture lit le `<short8>.facts.json` du corpus | **4 goldens sur 8 gagnent des joueurs** : `111fa685` 24 → 25, `11de8353` 26 → 27, `a521164d` 26 → 27, `e5adf7b2` 26 → **28**. `a521164d` et `e5adf7b2` passent à **0 trace sans identité** (1 avant) ; `a521164d` : ramasseurs nommés 38 → 40. Les 4 autres films n'ont aucun joueur à 0 mort. Fixtures 11 044 407 → 11 044 440 o. Divergences restantes DÉCLARÉES en tête du fixture (les 3 gardes de mode `Flag`/`Zone`/`Bomb`, seules à commander un balayage) |
| 2026-09-14 | R1-2 (P2) | ce commit | le marcheur descendait dans la PREMIÈRE occurrence d'une phase (`connue && !m.vus[nom]`) ; il descend désormais dans CHAQUE appel, `pile` ne servant plus qu'à refuser un cycle | **Mutation jouée** (`s.balayerCalquesGardes()` appelé deux fois) : `TestObserveEtapesBuildFromFilm` **ROUGE** (« les etapes observees … ne sont pas BuildFromFilmSteps »), VERT après retour. Deux contre-tests neufs sur source fabriquée : `TestMarcheurDescendDansChaqueAppel` (2 étapes, 2 balayages) et `TestMarcheurRefuseUnCycle` |
| 2026-09-14 | R1-3 (P2) | ce commit | le garde n'exigeait que « verrou AVANT installation » ; il exige désormais aussi « installation AVANT `scanFilmInputs` » — l'ORDRE des trois symboles | **Mutation jouée** (installation déplacée d'une ligne après l'étage) : `TestBuildFromFilmWiresWorldObjectPrecision` **ROUGE** (« les largeurs d'axe sont installées APRÈS l'étage de balayage »), VERT après retour |
| 2026-09-14 | R1-4 (P2, règle 5) | ce commit | `decimateTracks`, `TrackCoverage` et `logTrackCoverage` sortent dans `tracks_publication.go` (déplacement pur) ; les notes par version de `document.go` sont SUPPRIMÉES — elles recopiaient `document_chronicle.go`, et v50 y est déjà « v50 AMENDÉ » d'un côté, pas de l'autre | `build.go` **607 → 516** (−91 contre sa taille d'AVANT le lot) · `document.go` **624 → 590** (−34) · `coverage.go` 433 → 437 (le seul champ de `Coverage`, sous 500) · `tracks_publication.go` 177 · `document_chronicle.go` 1 230 → 1 275 avec son **exemption ÉCRITE en tête** (append-only par construction, extracteur unique `testutil.ReplayChronicleVersions`, critère de retrait daté) |
| 2026-09-14 | R1-5 (P2, doc) | ce commit | `observe_test.go` disait « cinq phases dont trois appellent une sous-phase » ; sur pièces QUATRE en appellent et il y a CINQ sous-phases (`balayerPont` n'en appelle aucune). Corrigé au fichier ET à la ligne de gate 1.0.1 de cette §5 | Les deux textes disent la même chose que la source |
| 2026-09-14 | R1-6 (plafond) | ce commit | `golden_inputs_budget_test.go` : plafond **12 Mio** (12 582 912 o) pour les huit `inputs_*.bin.gz`, valeur écrite avec sa date, son historique EN OCTETS (10 849 119 à l'origine → 18 656 453 aux flottants → 10 337 463 en quanta → 10 323 769 à la clôture de 0.D → 11 044 446 aujourd'hui) et la règle « on ne relève pas sans décision écrite » | `TestGoldenInputsTiennentDansLeBudget` PASS : **11 044 446 o = 10,53 Mio** lus sur le disque — le SEUL chiffre, corrigé en R2-2 ; 1 538 466 o libres, 12 % du plafond. Croissance **+7,0 %** contre la clôture de 0.D, **+1,8 %** contre le jeu d'origine. Le jeu de contrat pèse 2 564 084 o (plafond 3 145 728) |
| 2026-09-14 | 1.0 revue R1 (gates) | ce commit | `gofmt -l ./internal ./cmd` ; `go vet` (6 paquets) ; `go test` (14 paquets) ; `golangci-lint --timeout 20m` ; `make check-types` ; `make test-web` ; `TestGoldenInputsFidelite` un processus par build ; régime court 2 × 5 films | gofmt VIDE ; vet propre ; 14 paquets ok, exit 0 ; lint **0 issues** ; `tsc -b` propre ; vitest **711 fichiers / 7 629 tests** ; fidélité **8/8** ; régime court : **écart à la SEULE étape `artifact`**, aux MÊMES empreintes et aux MÊMES comptes qu'avant la ronde (`a521164d` 3 405 379 o des deux côtés) — **la production n'a pas bougé d'un octet, la ronde 1 ne touche que le fixture et les gardes** |
| 2026-09-14 | 1.0 revue R2 | ce commit | **revue adversariale ronde 2 (corrections seules) : 1 P1 + 1 P2, 0 jeté, 2 corrigés.** 22 conditions tiennent (règle du roster identique cas limites compris, mutation « fixture sans roster » rouge sur 574 lignes de 624, marcheur et garde d'ordre rouges sous mutation, déplacement pur à l'octet, budget qui lit le disque, production inchangée). Le P1 était INTRODUIT par R1-4 | Pas de ronde 3 : le relecteur vérifie lui-même |
| 2026-09-14 | R2-1 (P1) | ce commit | **R1-4 avait supprimé la SEULE description de la v51.** En retirant de `document.go` les notes par version — doublon de la chronique pour toutes les autres — il a emporté la v51, qui n'avait JAMAIS eu d'entrée de chronique. Sur pièces : `2fb53db4e` pose `SchemaVersion = 51` le 2026-09-10, `b6b198baf` la remplace par 52 le lendemain — la version a bien été cuite. Trois textes affirmaient pourtant « 32 et 51 ont été SAUTÉS » (`testutil/replay_chronicle.go`, `replaybuild/artifact_schema_history_test.go`, ADR 0034 §« Corrections », point 2) | (i) entrée `// v51 (2026-09-10, lot 4.3 …)` RESTAURÉE à sa place chronologique, dans la forme que l'extracteur reconnaît, avec ses deux changements de contenu cuit (`identity.bipedSlots[].bid`, `abilityLabels[].family` → `UsageSummaryRev` us4→us5) et ses chiffres (`4f77afc1` : 18 vies non résolues, 10 sur des index déclarés par `BOT_METADATA`) ; (ii) `document.go` ne promet plus que ce que le garde tient — `document_shape_test.go` exige une entrée pour la version COURANTE, pas pour les passées ; (iii) les TROIS textes corrigés : 32 renumérotée 33/34 au merge (aucun artefact intégré), 51 cuite, entrée restaurée le 2026-09-14 ; l'ADR porte la leçon (un TROU de chronique se lit exactement comme un numéro sauté — l'entrée s'écrit dans le commit qui monte la version) |
| 2026-09-14 | R2-1 (preuve) | ce commit | l'extracteur `testutil.ReplayChronicleVersions` et les trois tests de `replaybuild` qui en dérivent | **`schema_51` apparaît et PASSE** dans les trois : `TestChaqueSchemaAnterieurSeLitARecuire`, `TestLeDepotRefuseChaqueSchemaAnterieur`, `TestLePointDEcritureRefuseLAppauvrissementAChaqueSchema`. Une version RÉELLEMENT cuite que ces trois tests ne rejouaient pas entre enfin dans la mesure. `TestDocumentShape*` vert, `internal/testutil` vert |
| 2026-09-14 | R2-2 (P2) | ce commit | trois chiffres pour une mesure (11 044 407 = total d'AVANT régénération, 11 044 446 mesuré, 11 044 440 au plan) et un « +12 % » sans base | Un SEUL chiffre partout, celui que le test lit sur le disque : **11 044 446 o**. Les pourcentages portent leur base : +7,0 % contre la clôture de 0.D (10 323 769 o), +1,8 % contre le jeu d'origine (10 849 119 o). L'historique du commentaire passe en OCTETS, plus en Mio arrondis |
| 2026-09-14 | 1.0 revue R2 (gates) | ce commit | `gofmt -l ./internal ./cmd` ; `go vet` ; `go test ./internal/games/halo_infinite/film/replay/ ./internal/replaybuild/ ./internal/archlint/ ./internal/testutil/` ; `git status --short` | gofmt VIDE ; vet propre ; **4 paquets ok, exit 0** ; arbre propre. AUCUN décodage : la ronde 2 ne touche ni le décodeur, ni les fixtures, ni les goldens |

| 2026-09-14 | 1.1 (oracle, AVANT tout changement) | `5a01ad9c2` | `CHUNK00_FILMS="…/53ce4390;…/64e8adfa;…/7344d24f" go test ./internal/games/halo_infinite/film/filmdec/ -run 'TestResidusPiedOctetEquipe\|TestResidusPiedDrapeaux' -v` | **173 événements sur 173** pour l'octet 37 (et pour l'octet 38) ; **84/173** pour l'octet 55, dont la seule valeur observée est `0` (x173) ; **4 lectures en accord parfait sur 180 essayées** — le plancher de bruit mesuré. 3 films retenus, 0 écarté. 2,6 s + 3,6 s |
| 2026-09-14 | 1.1.1 | ce commit | `decodeTh10Block` lit `ebs+footerByteTeam*8` avec `footerByteTeam = 37` ; les quatre littéraux `36*8 / 47*8 / 48*8 / 55*8` deviennent des constantes nommées ; commentaires `film.go:31-35` et `:180-200` réécrits, `extract.go:63` (la phrase « le champ team du film étant non fiable ») réécrite | `gofmt` vide, `go vet` propre. `b38` est écrit comme DOUBLON OBSERVÉ et explicitement NON LU (deux lectures d'un même fait divergent un jour en silence) |
| 2026-09-14 | 1.1.2 | ce commit | `th10Event` -> `FooterEvent` exporté (`TimeMS`, `Slot`, `Team`, `XUID`) + point d'entrée `FooterEvents(film)`, qui remplace les DEUX copies de `footerData` + `scanTh10Events` dans `extractCTF` et `extractFromTh10` | Aucun consommateur de `Team` (1.7.3 le prendra). **Rien ne traverse une frontière sérialisée** : ni `domain.ObjectiveEvent` (ligne DuckDB), ni le document cuit — vérifié par l'équivalence ci-dessous |
| 2026-09-14 | 1.1.3 | ce commit | fixture `testdata/pied_bloc_53ce4390.bin` (1 930 o, tranche [34 977, 36 907) du pied décompressé de `53ce4390`, `chunk_40.bin` type 3) + provenance écrite + porte `-update-pied-bloc` ; `TestPiedEquipeOctet37`, `TestPiedBlocContreLecture55`, `TestPiedBlocProvenance` | Les trois **PASS**. `TestPiedBlocProvenance` recoupe la tranche SUR LE FILM : 1 930 octets identiques. Bloc discriminant : octet 37 = 1, octet 55 = 0, équipe prouvée 1 (`2533274823110022`, t=61 155) |
| 2026-09-14 | 1.1.3 (mutation) | ce commit | `footerByteTeam` remis à 55, tests rejoués, puis remis à 37 | **DEUX tests ROUGES** : `TestPiedEquipeOctet37` (« ÉQUIPE LUE AU MAUVAIS OCTET : rendue 0, prouvée 1 ») et `TestPiedBlocContreLecture55` (« LA FIXTURE NE DISCRIMINE PLUS »). Remis à 37 : **verts**, `git diff` du fichier revenu à l'état du lot (md5 identique) |
| 2026-09-14 | 1.1.4 | ce commit | `GrammarRev` `grammar-2026-09-13` -> `grammar-2026-09-14` ; `-run GrammarRevSuitLaGrammaire -update-grammar-rev` puis relance sans la porte | Avant la montée : le test est **VERT malgré le changement de grammaire** (découverte D1 (1.1)). Après la montée seule : **ROUGE**, « LA REVISION A CHANGE SANS QUE LA GRAMMAIRE BOUGE ». Golden régénéré (empreinte INCHANGÉE `3c775720…`, historique complété), relance **ok** |
| 2026-09-14 | 1.1 (gates communs) | ce commit | `gofmt -l ./internal ./cmd` ; `go vet` (5 racines) ; `go test` (13 paquets : objectiveevents, film/…, archlint, replaybuild, killcollector, replaydoc) ; `make go-api-lint` | gofmt **vide** ; vet **0 diagnostic** ; **13 paquets ok, exit 0** (filmdec 8,9 s ; replay 14,7 s ; archlint 14,3 s) ; lint **0 issues, exit 0** (baseline non accrue) |
| 2026-09-14 | 1.1 (régime court) | ce commit | `go run ./cmd/replay-equiv -repo-root <worktree> -films …` — les 10 films, DEUX sous-ensembles de 5 | **écart à la SEULE étape `artifact`** sur les 10 (les 49 étapes de balayage identiques), deltas **+104** (8 films), **+103** (`bcb6d393`), **+102** (`51101d1d`) : exactement la table du lot 1.0.4, dont les références n'ont pas été re-figées. 3 min 44 + 1 min 58 |
| 2026-09-14 | 1.1 (attribution PAR MUTATION — AMENDÉE à la revue R1, cf. R1-5 : ce contrôle CONFIRME, il ne prouve pas ; la preuve est le graphe d appels) | ce commit | les DEUX fichiers de production du lot remis à `HEAD` (`git checkout HEAD -- film.go extract.go`, test du lot mis de côté), `replay-equiv` sur `51101d1d,bcb6d393`, puis fichiers restaurés (md5 vérifiés) | **MÊMES empreintes, MÊMES comptes qu avec le lot** : `51101d1d` 628 310 / `60d96001…`, `bcb6d393` 1 903 599 / `fe6f4add…`. **Le lot ne change pas un octet du document cuit** — l écart `artifact` est 100 % imputable à 1.0.4. Corpus gate ciblé **NON REQUIS** (§2.3). RÉSERVE DE LA REVUE R1 : cette mutation NE POUVAIT PAS rougir, la chaîne de cuisson n appelant jamais `FooterEvents` ni `Extract` (`replaybuild/matchfacts.go:98` et `:253` sont ses seuls points d entrée dans le paquet) — elle confirme le graphe d appels, elle ne le remplace pas |

| 2026-09-14 | 1.1.5 | `a752403da` | `racinesGrammaire` rend TROIS racines (`filmdec`, `killsource`, `analysis/objectiveevents`), résolues depuis `internal/` par `runtime.Caller` ; en-tête du test réécrit | **131 -> 150 fichiers hachés (+19)**. Empreinte `3c775720…` -> **`474c9faa…`**, `GrammarRev` INCHANGÉE (`grammar-2026-09-14`) : aucune grammaire ne bouge, seul l'ensemble haché s'élargit. Golden régénéré par `-update-grammar-rev`, historique complété |
| 2026-09-14 | 1.1.5 (mutation) | `a752403da` | commentaire ajouté sur `const teamAbsent` dans `objectiveevents/film.go`, gate rejoué, puis `git checkout HEAD -- <le fichier>` | **ROUGE** : « LA GRAMMAIRE A CHANGE SANS MONTEE DE REVISION », empreinte `0feeec29…` sur 150 fichiers. Restauration par fichier NOMMÉ, **md5 identique** (`9f684c1d…`), gate **vert**. C'est exactement le mouvement que le lot 1.1 avait fait passer en silence |
| 2026-09-14 | 1.1.5 (gates) | `a752403da` | `gofmt -l ./internal ./cmd` ; `go test ./internal/games/halo_infinite/film/filmdec/ ./internal/archlint/` | gofmt **vide** ; les deux paquets **ok** (8,1 s / 12,3 s) — le ratchet d'archlint ne voit rien à redire à la racine neuve |
| 2026-09-14 | 1.1.6 | ce commit | `replay-equiv -update` sur les 20 films, CINQ sous-ensembles de 4 (1 min 35 + 3 min 14 + 3 min 01 + 4 min 33 + 2 min 05) | **20 références réécrites, exit 0 partout.** Pics 0,08 à 0,66 Gio. `git diff --stat` : **20 fichiers, 20 insertions, 20 suppressions** — UNE ligne par fichier |
| 2026-09-14 | 1.1.6 (contrôle du diff) | ce commit | `git diff -U0` des 20 `.tsv`, classement des lignes changées par étape et calcul des deltas | **20 lignes `+artifact`, 20 lignes `-artifact`, et RIEN d'autre** : les 49 étapes de balayage sont identiques sur les 20 films. Deltas : **+102 (x1), +103 (x2), +104 (x14), +105 (x2), +106 (x1)** — la longueur du champ `coverage.tracks` (94 o fixes plus les chiffres), donc le lot 1.0.4 et lui seul |
| 2026-09-14 | 1.1.6 (passe de comparaison) | ce commit | `replay-equiv` SANS `-update`, les mêmes cinq sous-ensembles (1 min 27 + 3 min 10 + 2 min 59 + 4 min 31 + 2 min 05) | **20 identiques, 0 différent, 0 écarté, 0 échec, exit 0 partout.** Le corpus d'équivalence est de nouveau un gate qui peut rougir |
| 2026-09-14 | 1.1.6 | ce commit | en-tête de `CORPUS.txt` : bloc « références re-figées au commit `a752403da`, schéma 55 », l'ancien bloc passe en historique | Le bloc porte la raison (schéma 55 du lot 1.0.4), la mesure (une étape, les cinq deltas) et CE QUI AUTORISE le re-figeage : l'attribution par mutation du lot 1.1. Les références de `2dad8d6df` restent lisibles par `git show` |
| 2026-09-14 | 1.1.5-1.1.6 (gates communs) | ce commit | `gofmt -l ./internal ./cmd` ; `go vet` (5 racines) ; `go test` (13 paquets) ; `make go-api-lint` | gofmt **vide** ; vet **0 diagnostic** ; **13 paquets ok, exit 0** (filmdec 8,5 s, replay 14,7 s, archlint 14,7 s) ; lint **0 issues, exit 0** |

| 2026-09-14 | 1.1 revue R1 | ce commit | **revue adversariale ronde 1 : 1 P1 + 5 P2 recevables, 0 jeté, 6 corrigés dans le lot** ; 27 conditions du relecteur tiennent | Le P1 (R1-1) était une FAUSSE PREUVE : la fixture du lot ne discriminait que contre l'octet 55 |
| 2026-09-14 | R1-1 (P1) | ce commit | fixture re-choisie : bloc `t=133033` du même pied (`53ce4390`, tranche [165 000, 166 930)), dont l'**octet 36 vaut 2 et l'octet 37 vaut 1** ; les tests exigent le TRIPLET (slot, équipe, instant) et un contre-test vérifie que la fixture SÉPARE | L'ancienne fixture (`t=61155`) portait **1 aux octets 36, 37 ET 38** : `footerByteTeam = 36` (le SLOT, déclaré la ligne au-dessus) la laissait VERTE. **Quatre mutations jouées** : `Team=36` -> 3 tests ROUGES ; `Team=55` -> 3 ROUGES ; `Slot=37` -> 2 ROUGES ; `Time=47` -> 1 ROUGE. `Team=38` reste VERT, et c'est MESURÉ : `b37 == b38` sur **190 blocs sur 190** (4 films, 2 builds) — aucun bloc réel ne peut les séparer, ce qui est précisément pourquoi l'octet 38 n'est jamais lu |
| 2026-09-14 | R1-2 (P2) | ce commit | en-tête et `rpOctetTeam` de `filmdec/residus_pied_research_test.go` réécrits (« lu à l'octet 37 depuis `25f327d52` ») | L'oracle affirmait encore que la production lit l'octet 55 — doc inversée dans le document même que le lot cite comme preuve. La phrase « le paquet n'exporte pas ces fonctions » est corrigée aussi : `FooterEvent`/`FooterEvents` sont exportés depuis ce lot |
| 2026-09-14 | R1-3 (P2) | ce commit | `teamAbsent = -1` RETIRÉ de `objectiveevents`, contrat « -1 si absent » retiré du champ `Team` | Le sentinel était décoratif : tous ses producteurs sont appariés à `ok=false` et jetés, donc `FooterEvents` ne pouvait jamais le rendre. D14 : pas de repli qui ne peut pas tirer. `Team` est toujours lu quand l'événement existe, et le commentaire le dit maintenant |
| 2026-09-14 | R1-4 (P2) | ce commit | la porte de régénération lit `film_manifests/53ce4390.json`, exige `chunk_type == 3` pour le chunk 40, passe les métadonnées à `filmsource` et sélectionne le pied par [footerData] | Elle prenait `film.Chunk(NumChunks()-1)` : juste sur `53ce4390`, faux sur tout film dont le cache porte des chunks APRÈS son pied. Le `flag.Bool` devient `PIED_BLOC_UPDATE=1` (pas de drapeau global au binaire de test pour un usage annuel) |
| 2026-09-14 | R1-5 (P2) | ce commit | justification du re-figeage réécrite dans `CORPUS.txt` (et la ligne d'attribution de cette §5 amendée) : la preuve est le GRAPHE D'APPELS, la mesure le confirme | Vérifié sur pièces : les points d'entrée de la cuisson dans `objectiveevents` sont `StatRecordsCtx` (`replaybuild/matchfacts.go:98`) et `CaptureBurstTimes` (`:253`) — `CaptureBurstTimes` ne lit que les chunks de type 2 ; `Extract` n'a qu'UN appelant dans le dépôt, `cmd/diag_weapons_v3/process.go:37`. La re-mesure par mutation NE POUVAIT PAS rougir : elle confirme, elle ne prouve pas. La phrase « sans cette preuve, re-figer aurait été bénir une régression » est retirée |
| 2026-09-14 | R1-6 (P2) | ce commit | `TestFooterEventsSurUnFilm` : `filmsource.Film` monté en mémoire depuis la fixture (répertoire temporaire, `chunk_40.bin`, meta de type 3), puis `FooterEvents` et `Extract` ; `TestCaptureScorerPrendLeDernierDuCluster` (fonction pure) | Couverture : `FooterEvents` **0 % -> 75 %**, `extractFromTh10` **0 % -> 87,5 %**, `captureScorer` **0 % -> 100 %**, `finalize` **0 % -> 75 %**, `Extract` **0 % -> 33,3 %**. Paquet à **85,9 %**. `extractCTF` reste à 0 % et c'est DIT dans le fichier : il apparie des bursts de capture lus dans les chunks de type 2, hors de portée d'une fixture de pied |
| 2026-09-14 | 1.1 revue R1 (empreinte) | ce commit | R1-3 touche `objectiveevents/film.go` : `TestGrammarRevSuitLaGrammaire` **ROUGE** (`474c9faa…` -> `7384ed41…`), golden régénéré, `GrammarRev` INCHANGÉE | **Premier effet visible de l'élargissement du lot 1.1.5** : la veille, le même geste n'aurait fait rougir personne. Revision inchangée parce que deux changements du même jour la partagent (règle écrite dans `grammar_rev.go`) et qu'aucun offset ne bouge |
| 2026-09-14 | 1.1 revue R1 (gates) | ce commit | `gofmt -l ./internal ./cmd` ; `go vet` (5 racines) ; `go test` (12 paquets) ; `go test ./...` complet ; `make go-api-lint` ; `replay-equiv` 4 films | gofmt **vide** ; vet propre ; 12 paquets **ok** ; suite complète **exit 0, 325 paquets, zéro `--- FAIL:`** ; lint **0 issues, exit 0** ; équivalence **4 identiques, 0 différent** (1 min 32) — les références re-figées au 1.1.6 tiennent, et le retrait du sentinel ne déplace aucun octet |

| 2026-09-14 | 1.1 revue R2 | ce commit | **revue adversariale ronde 2 (corrections seules) : 0 P0, 0 P1, 4 P2 recevables, 1 jeté, 4 corrigés.** Le jeté : `extract.go:213` `best := FooterEvent{TimeMS: -1}` — sentinelle locale à `captureScorer`, sans conséquence observable (le `ok=false` gouverne, le test le couvre à 100 %) | La boucle converge : 1 P1 (R1) -> 0 (R2). Les quatre constats portent sur des TEXTES écrits par le lot et sur une porte de test, aucun sur le décodeur |
| 2026-09-14 | C1 (P2) | ce commit | motif de la porte par manifeste RÉÉCRIT ; mesure rejouée sur les **1 351 manifestes ET répertoires** de `data/cache/film_chunks` | **0 film sans pied, 0 film avec un chunk APRÈS son pied, 1 351/1 351 dont le dernier `chunk_NN.bin` EST le chunk de pied.** Le motif affirmé au R1-4 (« des films portent des chunks après leur pied ») était FAUX. Le vrai motif est écrit : le type est ce que le manifeste ÉCRIT (grammaire, ADR 0034 D10) ; « le pied est le dernier chunk » serait un REPLI POSITIONNEL — qui donnerait le même résultat partout aujourd'hui, et c'est précisément ce qui le rendait dangereux |
| 2026-09-14 | C2 (P2) | ce commit | table des mutations de la provenance RE-MESURÉE, mutation par mutation, noms des tests rouges relevés | La table annonçait **2 rouges pour `footerByteSlot = 37` (il y en a 3)** et **1 pour `footerByteTime = 47` (il y en a 2)**. Comptes exacts : `Team=36` **3** · `Team=55` **3** · `Slot=37` **3** · `Time=47` **2** (`TestPiedBlocSepareLesChamps` ne lit pas l'instant) · `Team=38` **0, vert**. Les noms des tests sont désormais écrits ligne à ligne |
| 2026-09-14 | C3 (P2) | ce commit | compte des appelants d'`Extract` RE-MESURÉ dans `CORPUS.txt` | `grep -rn "objectiveevents\.Extract(" --include=*.go apps/go-api/` rend **SIX** lignes, pas une : 1 hors test (`cmd/diag_weapons_v3/process.go:37`) et **5 dans des `_test.go` de `film/replay/`** (402, 170, 229, 174, 122). La CONCLUSION tient — aucun binaire de production ne passe par là, et `FooterEvents` n'a que DEUX appelants de production (`extract.go:168` et `:236`), tous deux sous `Extract` — mais elle repose sur six lignes |
| 2026-09-14 | C4 (P2) | ce commit | `dir = filepath.Clean(dir)` UNE FOIS à l'entrée de la porte, avant toute dérivation | Avec un séparateur final (`…/53ce4390/`, ce que produit la complétion d'un shell), `filepath.Base` l'absorbait mais `filepath.Dir(filepath.Dir(dir))` remontait un cran trop bas : la porte échouait sur « `…/film_chunks/film_manifests/53ce4390.json` : chemin introuvable » pour une entrée valide. Rejouée **SANS et AVEC** le séparateur : **les deux PASS**, même log (`1930 octets identiques à la fixture`) |
| 2026-09-14 | 1.1 revue R2 (gates) | ce commit | `gofmt -l ./internal ./cmd` ; `go vet ./internal/analysis/objectiveevents/` ; `go test -count=1` (objectiveevents, filmdec, archlint) | gofmt **vide** ; vet **0 diagnostic** ; **3 paquets ok** (0,42 s / 8,04 s / 12,34 s). **Empreinte de grammaire INCHANGÉE et golden NON régénéré** : la ronde ne touche qu'un `_test.go` et deux fichiers de `testdata/`, tous deux hors de l'ensemble haché — `TestGrammarRevSuitLaGrammaire` est resté vert sans rien faire. Aucun test renommé ni supprimé : baseline JSONL intacte |
| 2026-09-14 | 1.2.1 | `32f795e2b` | `go test ./...filmdec/ -run TestRegistreNiveauxVoisinsCensus -v` (mesure AVANT de coder, 7 bobines) | 1 031 à 1 067 composants par registre, **173 à 189 niveaux changent** (16,7 à 18,2 %) ; **16 instances consomment le niveau, 4 seulement changent**, identiques sur les 7 builds : ti=14 i0 crew-order (L0->L1), ti=21 i2 flock-destination (L1->L2), ti=30 i0 tacmap-poiicon (L0->L1), ti=44 i0 asset-transform (L0->L1). 0,04 s. Rapport complet dans le bloc du lot |
| 2026-09-14 | 1.2.2 (mesure préalable) | `48ad2a4f4` | sonde jetable sur les 7 bobines : compte de blocs et nullité de la queue sous le cadrage à l'octet 8, AVANT de toucher `parseRegistry` | **même compte de blocs 7/7** (49 pour HI_1_4_1 à HI_1_11_0, 50 pour HI_1_12_0 et HI_1_13_0) et **queue entièrement nulle 7/7**. C'est ce qui autorise `registryBlockTail` à supprimer son exemption de 4 octets. Sonde retirée après mesure ; l'assertion permanente est `TestRegistreEntreeDuJeu` (1.2.5) |
| 2026-09-14 | 1.2.4 | `1d824e632` | `ECS_TABLE_FILM=../killsource/testdata/minibobine_000d5950 go test ./...filmdec/ -run 'TestG1\|TestG2\|TestG3\|TestG4' -v` | **4 garde-rails VERTS** ; G2 confronte **1 067 lignes de registre à 1 067 lignes de table (+14 alias)**, 50 blocs, 49 porteurs, 0 écart. 0,15 s |
| 2026-09-14 | 1.2.4 (tentative écartée) | — | correction de la colonne `level` À PARTIR des annotations `niveau_jeu=N` de la table (178 lignes) | **G2 ROUGE sur 11 clés** — les annotations étaient incomplètes (189 lignes réellement décalées). La colonne est donc régénérée DEPUIS LE FILM par la porte nommée `-update-ecs-table-level` : 189 lignes changées, 0 ligne absente du registre. Découverte D3 (1.2) |
| 2026-09-14 | 1.2.4 | `1d824e632` | `LOT3_CORPUS=<cache> go test ./...filmdec/ -run TestLot3CompteRegistre -timeout 40m -v` (corpus ENTIER) | **1 351 chunk_00 lus, exit 0, 14,9 s.** C1 (fin structurelle = bloc d'identification) **TENU, 0 violation sur 1 346 films à section** ; C2' (fantômes au-delà) **TENU, 0** ; C3 (builds de référence : 50 blocs, 1 067 slots) **TENU, 0 sur 1 269 films**. La ré-implémentation indépendante de la règle et `parseRegistry` s'accordent sur les 1 351 films |
| 2026-09-14 | 1.2.4 | `1d824e632` | `go test ./internal/api/wire/ -run TestFixtureFilmRegistreECSLisible -v` (tag cgo) | **PASS** — la nouvelle `KnownRegistryFingerprint` (`0x36ca8c3d2a2f9b88`) est bien celle du fixture de l'ouvrier |
| 2026-09-14 | 1.2.5 | `0e99a81b1` | `go test ./...filmdec/ -run TestRegistreEntreeDuJeu` + DEUX mutations | vert sur les 7 bobines. Mutations, les deux **ROUGES** : `registryEntryLevelOffset` 0x100 -> 0xfc (« l'équivalence du décalage est rompue » dès ti=0 i0) ; `registryEntryBase` 8 -> 0 (« aucun composant de ti=35 ne change de niveau entre les deux cadrages »). Arbre restauré, `git diff` vide sur `registry.go` |
| 2026-09-14 | 1.2.6 | `845768285` | `go test ./...filmdec/ -run GrammarRevSuitLaGrammaire -update-grammar-rev` puis sans le drapeau | porte **sortie en ÉCHEC** comme elle le doit (« 1 référence réécrite »), puis **vert**. `grammar-2026-09-14` -> `grammar-2026-09-14.2`, empreinte `7384ed41…` -> `7a9404b9…` (150 fichiers). Historique du golden complété |
| 2026-09-14 | 1.2.6 | `845768285` | `go test ./...killsource/ -run TestGoldenMiniBobine` puis `-update`, pré-image conservée et `diff` complet | **UNE ligne sur 77** : `recordStateParam=3 [croissance x1.002]` -> `x1.001`. `axisW=14`, `indexW=1`, `recordStateParam=3` INCHANGÉS ; lignes de kill, couverture, contrôle négatif, voies et santé identiques à l'octet. **CLASSÉ DIVERGENCE ATTENDUE** : `RSPRatio` est le quotient best/worst du critère de croissance (`calibrate.go:157`), qui traverse des records bruts de tous les archétypes présents |
| 2026-09-14 | 1.2 (communs) | ce commit | `gofmt -l ./internal ./cmd` ; `go vet` (5 racines) ; `go test` (13 paquets) ; `make go-api-lint` | gofmt **vide** ; vet **0 diagnostic** ; **13 paquets ok, exit 0** (filmdec 7,9 s · killsource 0,5 s · replay 14,8 s · archlint 15,5 s · objectiveevents 0,3 s) ; lint **0 issues** (`--new-from-merge-base=origin/main`), baseline non accrue |
| 2026-09-14 | 1.2 (équivalence, régime COMPLET) | ce commit | `go run ./cmd/replay-equiv -repo-root <worktree>` sur les 20 films, 4 sous-ensembles séquentiels (jamais deux à la fois) | **20 IDENTIQUES sur 20, 0 différent, 0 écarté, 0 échec.** Sous-ensembles : 10 films 5 min 27 s · `084a804d`+`1c4c63c2` 5 min 12 s · `a349fea8`+`a521164d` 5 min 04 s · 6 films 4 min 04 s. **Total 19 min 47 s**, pic max 0,77 Gio (`a349fea8`). AUCUN des 50 balayages ne change, l'artefact compris — donc **aucun octet cuit ne bouge et `SchemaVersion` reste à 55**. Prédit par la mesure 1.2.1 : les 4 instances corrigées vivent en ti=14/21/30/44, qu'aucun balayage du corpus ne traverse (découverte D4 (1.2)) |
| 2026-09-14 | 1.2 (corpus gate, régime COMPLET) | ce commit | `go run ./cmd/replay-corpus-gate --base=191933992 --parc-root <parc> --source-root <worktree> --manifest config/replay_corpus.toml` | **13 témoins, 0 PERTE, 0 GAIN, exit 0**, schéma 55 des deux côtés sur les 13. Durées : `bcb6d393` 15,2 s · `fb1a1a72` 31,5 s · `d9781168` 24,7 s · `c75f33b8` 15,4 s · `bf15f7ab` 14,2 s · `51ebbc0f` 19,2 s · `084a804d` 2 min 21,9 s · `0797ce72` 12,9 s · `111fa685` 34,5 s · `e5adf7b2` 49,6 s · `60ae07c4` 36,7 s · `a349fea8` 3 min 28,3 s · `bfecd02b` 36,8 s. **Total 23 min 10 s** (chaque témoin cuit DEUX fois, base et HEAD). Aucune perte, donc aucun lecteur à corriger au titre de 1.2.6 |
| 2026-09-14 | 1.2 (clôture) | ce commit | doc inversée résiduelle corrigée dans `registry_fingerprint.go` (l'en-tête affirmait « le registre est bit-a-bit IDENTIQUE sur tous les films mesures a ce jour » deux écrans au-dessus du commentaire qui dit l'inverse) ; porte de régénération de la colonne `level` sortie dans `ecs_table_level_gate_test.go` (`ecs_table_guard_test.go` frôlait 500 lignes : 495 -> 442) | Correction de TEXTE seule : aucun octet lu ne change, mais l'empreinte de grammaire hache les octets — `TestGrammarRevSuitLaGrammaire` ROUGE (`7a9404b9…` -> `89d2ecac…`), golden régénéré par sa porte nommée, **révision INCHANGÉE** (`grammar-2026-09-14.2`) : c'est le MÊME lot, et la règle de `grammar_rev.go` est que deux changements d'un même lot la partagent. Puis gates rejoués : gofmt vide, vet 0, **13 paquets ok**, G1/G2/G3/G4 verts avec le film, lint **0 issues** |

## 6. Protocole de reprise de session

1. Relire le skill `plan-execution`, puis la §5 et la première case non statuée de la §3.
2. `git worktree list` ; `git -C LevelUp-wt-recherche-film log --oneline -5` ; vérifier que
   l'intégration porte le dernier lot fusionné (§5).
3. Ne pas re-décider une décision D ou V validée ; une question neuve = ligne en §1.4, posée à
   l'utilisateur, jamais tranchée en silence.
4. Reprendre au lot courant : exécuteur relancé avec le brief du lot (état réel constaté sur
   pièces, pas le plan de mémoire).
