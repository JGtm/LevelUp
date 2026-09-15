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
| V8 (2026-09-14) | Revue adversariale EN FIN DE JALON seulement, plus par lot (décision utilisateur après les lots 1.0-1.2 : trois lots, six rondes, 2 h à 3 h par lot). Par lot : vérification sur pièces par le pilote (grep, gates rejoués, mutations de l'exécuteur) ; en fin de jalon : fan-out de relecteurs aveugles, un par lentille, sur le diff des lots NON relus (M1 : `783ae680d..<HEAD M1>`), deux rondes au plus, puis fusion dans `feat/v75`. Coût accepté : un défaut introduit par un lot n'est vu qu'à la fin du jalon, après que d'autres lots ont bâti dessus | ok |
| V9 (2026-09-14) | GO utilisateur pour la clôture de M1 : fusion dans `feat/v75`, recuisson du parc, backlog killsource — sans nouvelle demande, dès que le dernier lot de M1 est fusionné et la revue de jalon close | ok |
| V10 (2026-09-16) | **Famille 1.9 RESSERRÉE** (question utilisateur : « j'ai l'impression qu'on part loin »). Constat du pilote : 1.9.1 bis / ter sont devenus de la rétro-ingénierie longue (état par défaut des films anciens : 12 commits, 25 découvertes, clé bornée mais non trouvée au 1.9.1 bis) et 1.9.4 a rendu un gain de production nul (passe des touches éteinte). Mesure : les builds anciens (HI_1_4_1 à HI_1_11_0, plus 5 films sans section) pèsent **82 films sur 1 351** au cache (6 %) ; HI_1_13_0 en pèse 1 123. Décision : (a) 1.9.1 ter BORNÉ à une seule passe — chargeur, condition, mesure ; ce qui ne se ferme pas chez l'écrivain passe à M3 (divergences par build) ; (b) M1 garde de 1.9.5 à 1.9.14 les items à effet visible dans le rejeu, 1.9.7 et 1.9.9 à 1.9.14 ; (c) 1.9.5, 1.9.6, 1.9.8 REPORTÉS à M2 (`[!]`, replis non câblés donc fréquence non mesurée) ; (d) puis revue de jalon et clôture M1 (V9) | « vas y continue » |

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
Lots qui touchent le contrat publié (`replaydoc`, `replayview`, `openapi`) : `apps/go-api/api/openapi.yaml`
ne s'édite JAMAIS à la main, il se RÉGÉNÈRE en dernier par `make openapi-gen` puis
`make generate-types` ; gate `go test ./internal/api/ -run TestOpenAPIYAMLIsUpToDate -count=1`
(CGO) avant le dernier commit (leçon du lot 1.9.1, 2026-09-15 : `byFamily` / `byCause` écrits à la
main dans un ordre que le générateur ne produit pas, CI « Coverage + Baseline » rouge sur ce seul test).
Tout test RENOMMÉ ou SUPPRIMÉ par un lot met à jour `.ai/baselines/tests_pre_migration.jsonl` dans le
MÊME commit (le job CI « Coverage + Baseline » échoue sur tout nom de baseline absent du run ;
leçon du lot 1.0, 2026-09-14 : un renommage de test oublié = CI rouge sur un lot vert partout ailleurs).
**Instruments lourds = tag `research` (règle posée le 2026-09-16, D1 (CI) en §4).** Un instrument qui
REBALAYE les 7 bobines (ou lit des films entiers du cache) porte `//go:build research` en première
ligne et se joue à la demande : `go test -tags research ./internal/games/halo_infinite/film/filmdec/
-run TestE191` ; la CI le COMPILE (`go vet -tags research`, job unitaire Linux) sans l'exécuter. Un
helper qu'un test du build par défaut emploie aussi vit dans un fichier NON tagué (modèle :
`filmdec/e191_helpers_test.go`). Budget écrit : le paquet `filmdec` sans tag tient sous **30 s** en
local (CGO_ENABLED=0, `-count=1`) ; chaque compte rendu de lot qui touche `filmdec` cite cette durée,
et un lot qui la dépasse tague ses instruments avant de clore. Leçon : après le lot 1.9.1 bis le
paquet valait 89,9 s (72 s d'instruments E191b/E191c), le job unitaire Windows a rougi à 60 s puis le
job de couverture Linux à 600 s (run 34978931976) — les budgets ne sont pas la défense, le tag l'est.

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
à part : `paths-ignore` de la CI) ; 4. vérification sur pièces par le pilote (V8 : la revue
adversariale est en fin de jalon depuis le 2026-09-14 ; les lots 0.A à 1.2 ont eu la leur) ; 5. fusion dans
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

Colonnes par archétype = « niveaux qui changent / composants ». Tableau REÉCRIT CELLULE PAR
CELLULE depuis la sortie de l'instrument le 2026-09-14 (revue R1, constat C3 : treize cellules
avaient été recopiées d'une ligne à l'autre) ; la sortie brute est collée en §5, ligne
« 1.2 revue R1 (C3) », pour être rejouable.

| Bobine | Composants | Niveaux qui changent | ti=9 | ti=11 | ti=12 | ti=35 | ti=40 | ti=42 | ti=43 |
|---|---|---|---|---|---|---|---|---|---|
| a521164d | 1 033 | 173 (16,7 %) | 1/9 | 3/34 | 6/28 | 19/64 | 18/48 | 10/21 | 10/40 |
| 60ae07c4 | 1 031 | 178 (17,3 %) | 1/9 | 3/34 | 6/28 | 23/64 | 18/48 | 10/21 | 10/41 |
| 11de8353 | 1 031 | 178 (17,3 %) | 1/9 | 3/34 | 6/28 | 23/64 | 18/48 | 10/21 | 10/41 |
| 111fa685 | 1 031 | 186 (18,0 %) | 1/9 | 3/34 | 6/28 | 23/64 | 20/48 | 10/21 | 12/41 |
| e5adf7b2 | 1 031 | 188 (18,2 %) | 1/9 | 3/34 | 6/28 | 25/64 | 20/48 | 10/21 | 12/41 |
| bcb6d393 | 1 067 | 189 (17,7 %) | 1/10 | 3/34 | 6/28 | 25/64 | 20/48 | 10/21 | 12/41 |
| fb1a1a72 | 1 067 | 189 (17,7 %) | 1/10 | 3/34 | 6/28 | 25/64 | 20/48 | 10/21 | 12/41 |

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

Sur pièces AVANT le lot, et trois citations du plan à corriger : la table `defaultStateDeserByTI`
est aux lignes **44-66** (pas `:54-77`), le commentaire STUB de ti=14 à la ligne **24** (pas 25),
et le relevé qui donne les cinq grammaires est la section **B.2** de
`NOTE_IMAGECLE_ETAT_COMPLET_2026-09-13.md` — il n'y a pas de « 8.5 » dans cette note.
CLOS le 2026-09-14 (branche `feat/decfilm-13`, 5 commits `ca9b8116c` -> le commit de clôture).

- [x] 1.3.1 Entrées ti=14 `V ; R(5)`, ti=17 `V ; R(7)`, ti=21 `R(18)`, ti=29 `V`, ti=47 `V ; R(5)`,
      chacune avec sa fonction Ghidra (relevé B.2 de `NOTE_IMAGECLE_ETAT_COMPLET_2026-09-13.md`)
      et sa date ; commentaire STUB de ti=14 corrigé.
      FAIT, et les cinq fonctions ont été RELUES CHEZ L'ÉCRIVAIN le 2026-09-14 (Ghidra, base
      `0x140000000`, lecture seule) au lieu d'être recopiées de la note : `FUN_140FED6F4` (ti=14)
      et `FUN_1410F44F8` (ti=47) font `FUN_1406cf008` puis `+= 5` ; `FUN_14101A0A4` (ti=17)
      `+= 7` ; `FUN_141133C24` (ti=21) un unique `+= 0x12` SANS appel de préfixe ; `FUN_14116F514`
      (ti=29) le seul préfixe. `FUN_1406cf008` a lui aussi été relu : R(1) sec.
      ti=47 partage le PORTEUR de ti=14 (même forme au bit près), comme ti=43 partage celui de
      ti=36 — le commentaire de la ligne cite son adresse propre.
      Le commentaire STUB est corrigé avec sa CAUSE : `FUN_140467a20` n'est pas un déserialiseur
      mais un `return;` partagé par treize symboles exportés sans rapport
      (`AK::MemoryMgr::GetCategoryStats`, `ManagedDebug_LogError`, ...) — une résolution qui
      atterrit là ne trouve pas un stub, elle se trompe de table.
      ÉCRIT EN PLUS, parce que D14 l'exige : le repli « absent = 0 bit » est NOMMÉ au-dessus de la
      table, avec les quatre archétypes qui en vivent encore (ti=23, 40, 41, 44 — les seuls REAL
      de la table de descripteurs sans grammaire relue, hors le bipède), sa date de pose, son
      critère de retrait et l'endroit où il se compte (le golden de fermeture).
- [x] 1.3.2 Test permanent « `n2` constant » sur les mini-films : pour tout archétype à état fixe,
      le second mot de taille est constant ; fausser une largeur (ti=21 → 17) rougit.
      FAIT (`default_state_n2_constant_test.go`, sans garde d'environnement, sur les sept bobines
      par build). **77 groupes jugés** (bobine × archétype à largeur FIXE), 34 écartés pour largeur
      VARIABLE, 6 faute de records ; plancher de non-trivialité à 60. La MUTATION demandée est
      jouée : `R(18)` → `R(17)` rougit sur les CINQ bobines qui portent des records ti=21, `n2`
      passant de `244` constant à `122 / 2147483770` ; remise à 18, diff contre `HEAD` vide.
      Le test écrit aussi ce qu'il N'affirme PAS : `n2` constant ne prouve pas qu'une largeur est
      juste (ti=29 était déjà constant à 0 bit) — c'est un détecteur, la fermeture est l'autre
      chaîne.
- [x] 1.3.3 Ratchet de couverture (0.A.3) régénéré avec justification : fermeture qui MONTE
      (attendu : + records sur ti=14/17/21/29/47, projection 30,8 % sur les 6 films de recherche).
      FAIT, et la justification est DANS le golden (l'en-tête que le générateur écrit, donc elle
      survit à la prochaine régénération) : **21 lignes montent, 0 descend, aucune ne disparaît,
      aucun total ne bouge** — ti=14 `0/3520 -> 3520/3520`, ti=17 `0/3729 -> 3729/3729`, ti=29
      `0/110 -> 102/110`. Les archétypes touchés par le diff sont EXACTEMENT ti=14, 17 et 29
      (`git diff -U0` sur le golden). **ti=21 (0/357) et ti=47 (0/1716) ne montent pas**, et la
      note le prédisait : leur largeur est prouvée par deux chaînes, un composant reste faux
      (ti=47 bute sur `i2 personal-ai-data-component`) — lot 3.6.
      La projection est VÉRIFIÉE, pas recopiée : sur les 6 films de recherche,
      **8 796/62 686 (14,0 %) -> 19 337/62 686 (30,8 %)**, delta = 5 024 + 5 379 + 138, et le
      plancher de hasard (même lecture, en-tête décalé d'un bit) DESCEND de 529 à 391.
- [x] 1.3.4 `GrammarRev` montée.
      `grammar-2026-09-14.2` -> `grammar-2026-09-14.3` (troisième lot du même jour, forme de rang
      née au lot 1.2). L'empreinte avait ROUGI D'ELLE-MÊME dès le commit de 1.3.1 (« LA GRAMMAIRE
      A CHANGE SANS MONTEE DE REVISION ») : c'est la preuve par mutation naturelle, aucune
      mutation artificielle n'était nécessaire. Golden régénéré par sa porte nommée (qui réécrit
      PUIS échoue), historique complété.
      `KillSourceDecoderRev` N'EST PAS montée : c'est un geste de PROD (backlog de redécodage),
      réservé au pilote sur signal — même arbitrage qu'au lot 1.2, découverte D2 (1.2) toujours
      ouverte. La mesure de ce lot la précise : sur `50247b26`, la sortie de `killsource` change
      d'UN SEUL caractère, dans une chaîne de DIAGNOSTIC (`calibration`, « médiane 77 » -> « 76 »)
      — les lignes de kill, le catalogue, la couverture et la santé sont identiques à l'octet, et
      le golden de `killsource` sur la mini-bobine ne bouge pas.

Preuve : `replay-equiv` (attendu : différences seulement sur les balayages delta des archétypes
14, 17, 21, 29, 47 s'ils passent par `consumeKeyframeDefaultState`) ; corpus gate zéro perte.
RÉSULTAT (2026-09-14) : régime court **9 identiques / 1 différent**, et l'unique différence est à
la SEULE étape `killsource` sur `50247b26` — l'étape `artifact` est IDENTIQUE sur les dix films,
donc **`SchemaVersion` NE MONTE PAS** (55 avant, 55 après) et aucune recuisson du parc n'est due.
L'attribution n'est pas un raisonnement : les deux sorties `killsource json` de `50247b26` ont été
produites de part et d'autre du correctif (fichier restauré par NOM au commit `783ae680d`) et
diffèrent d'UNE valeur sur 228 800 octets.

##### Rapport 1.3.0 — la population touchée, mesurée avant de coder

`defaultStateDeserByTI` a **un seul lecteur de production**, `TraverseEntity`
(`traverse.go:1107`, `useArchDefaultStateDeser` à `true`), et il ne le consulte que sur un record
NEW ; `WalkKeyframeFullState` / `KeyframeClosure` n'ont AUCUN appelant hors de `filmdec`
(`grep -rn "WalkKeyframeFullState(\|KeyframeClosure(" --include=*.go internal/ cmd/ | grep -v
_test.go` rend 3 lignes, toutes dans `filmdec`). La population qu'une entrée neuve touche est donc
celle des records NEW de la trame — et c'est elle qu'il fallait mesurer avant d'écrire.

Histogramme publié par `TestDeltaWalkWitness` (12 premiers chunks de réplication, sortie brute
collée en §5), records NEW par archétype AVANT le lot :

| Film | ti=14 | ti=17 | ti=21 | ti=29 | ti=47 | total NEW |
|---|---|---|---|---|---|---|
| 000d5950 | 13 | 43 | 29 | 25 | 17 | 3 322 |
| 06dfe6d9 | 6 | 8 | 5 | — | 2 | 811 |
| 64e8adfa | 13 | 395 | 71 | 7 | 9 | 2 518 |

PRÉDICTION TIRÉE DE CETTE MESURE, écrite avant le gate : la trame traverse bien des records des
cinq archétypes, donc l'équivalence NE POUVAIT PAS rendre 20/20 comme au lot 1.2 ; elle devait
montrer des écarts sur les balayages qui consomment la marche delta. Résultat : un seul en rend un
(`killsource`), et d'un seul caractère. LIMITE HONNÊTE DE L'HISTOGRAMME : il compte aussi des
`TypeIndex` de 50 à 63, qui n'existent pas (l'exe en déclare 50) — ce sont des records lus après
une désynchronisation, donc les comptes ci-dessus sont un MAJORANT.

Côté image-clé, la fermeture avant le lot est celle du golden 0.A.3 : ti=14 `0/3520`, ti=17
`0/3729`, ti=21 `0/357`, ti=29 `0/110`, ti=47 `0/1716` sur les sept bobines ; `0/5024`, `0/5379`,
`0/373`, `0/157`, `0/1679` sur les six films de recherche.

#### Lot 1.4 — Le cadre d'image-clé d'état complet en production — M, high

Sur pièces, RE-VÉRIFIÉ au commit de base `15309e89e` le 2026-09-14 : **les trois citations du plan
étaient EXACTES** — `navpoint_radial_scan.go:339` (dans `scanKeyframe`, ouvert l. 325),
`objective_scan.go:373` (dans `scanKeyframe`, ouvert l. 360) et `keyframe_record_walk.go:181`
(dans `walkOneKeyframeRecord`, ouvert l. 176) appelaient bien `TraverseEntity(br, reg, 0)` après
`SetBitPos(r.Bit + keyframeRecordTIBit)`. La bonne forme est `WalkKeyframeFullState`
(`keyframe_fullstate_loop.go`).
CLOS le 2026-09-14 (branche `feat/decfilm-14`, 5 commits, `44112034c` → le commit de clôture).

- [x] 1.4.0 (item ajouté par le pilote) **Attribuer la dérive du témoin figé de la marche delta
      AVANT de coder** (découverte D1 (1.3)). FAIT par bisection sur la chaîne premier-parent
      `4ad72a4a1..8f35efb72` (690 points, worktrees détachés jetables, `DELTA_WITNESS_FILM` en
      chemin absolu vers le cache du principal, un seul décodage à la fois).
      **LES SIX POINTS D'INTÉGRATION NOMMÉS PAR LE PILOTE PORTENT TOUS LA DÉRIVE DÈS LE PREMIER**
      (tableau collé en §5) : `8f35efb72`, `fc5db87f7`, `9e6a9e08d` (1.0) et `191933992` (1.1)
      rendent EXACTEMENT les mêmes triplets — M0, 0.D, 1.0 et 1.1 ne bougent RIEN ; `783ae680d`
      (1.2) et `15309e89e` (1.3) bougent de leur propre effet, déjà chiffré dans leur journal.
      La dérive est donc ANTÉRIEURE au chantier, et l'instruction a été étendue à l'amont.
      **QUATRE MARCHES, TOUTES NOMMÉES, TOUTES ANTÉRIEURES À M0** : `62ba098b8` (2026-09-01, merge
      `wt/bombe-visuel` : désérialiseurs d'objectif/bombe portés, +2/+2) · `8f309ce86` (2026-09-02,
      merge `feat/precision-arme` : le registre borné à sa fin structurelle, −5/−7) · `736ccf3c3`
      (2026-09-05, merge cuisson-perf + véhicules, 0/−8) · `ffb27238c` (2026-09-11, merge
      `wt/munitions-objet`, grammaire d'i9 relue au désassemblage, +17/+10).
      **LE « SENS QUE LE CONTRAT REFUSE » EST UN ARTEFACT D'AGRÉGATION.** D1 (1.3) relevait que
      sur `06dfe6d9` les records MONTAIENT (+16) pendant que les traversées abouties DESCENDAIENT
      (−3). Aucune marche ne fait cela : pris un à un, les six mouvements sont cohérents
      (+2/+2, −5/−7, 0/−8, +17/+10, +2/0, +7/+6). Ce n'est pas un lot qui a produit un sens
      impossible, c'est un témoin figé laissé en place pendant quatre lots — et il est sous garde
      d'environnement, donc la CI ne l'a jamais joué.
      **VERDICT : DIVERGENCE sur les quatre, aucune RÉGRESSION**, et le point le plus suspect le
      prouve. Sur `8f309ce86` (le seul qui perd des traversées sans en gagner), la sonde par
      archétype ne fait changer de verdict QUE `ti=49` : `0/7` non portés → `7/7`. Cause MESURÉE,
      pas déduite : avant ce merge, `parseRegistry` découpait le `chunk_00` par « taille du
      fichier / taille d'un bloc » et résolvait **64 archétypes** sur `06dfe6d9`, dont ti=49 à 63
      **à ZÉRO composant** — du bourrage. Une traversée sur un archétype vide se termine sans rien
      lire, donc elle ABOUTISSAIT. Depuis, le registre s'arrête à sa fin structurelle : **49
      archétypes, ti=49 ABSENT**. Les 7 traversées perdues ne lisaient rien.
      Les deux autres pertes sont localisées : `736ccf3c3` perd 6 sur ti=0 (le puits de
      désynchronisation), 1 sur ti=33 et 1 sur ti=38 — aucune ligne ne bascule en bloc.
      **AUCUN DES QUATRE POINTS N'EST DANS LE PÉRIMÈTRE DU CADRE D'IMAGE-CLÉ** : ils vivent tous
      dans la marche DELTA et dans le registre. Donc aucun correctif dans ce lot — report §4 et
      ligne au bloc « Clôture M1 ».
      TÉMOIN RE-FIGÉ, avec l'attribution ÉCRITE DANS LE FICHIER (édition datée : ce témoin n'a pas
      de porte nommée, et le contrat du fichier exige la cause avant le chiffre) :
      `000d5950` {14 350, **38 945**, **30 118**} · `06dfe6d9` {6 606, **10 636**, **8 505**} ·
      `64e8adfa` {14 357, **39 936**, **31 933**}. Les trois rendent CONFORME après re-figeage, et
      **ils sont restés conformes à la fin du lot** — preuve de plus que le cadre d'image-clé ne
      touche pas la marche delta.
- [x] 1.4.1 `WalkKeyframeFullState(pay, recBit, reg)` sans option (D8) : en-tête 108, mots de
      taille, état par défaut ; la boucle historique `WalkKeyframeBody` / en-tête 64 + masque
      supprimée si plus aucun appelant de production (inventaire sur pièces ; sinon consignée).
      FAIT. `KeyframeFullStateOpt` disparaît, comme `shiftArchetypeLevels` au lot 1.2 ; les deux
      mots de taille deviennent inconditionnels.
      CE QUI SUBSISTE, ET IL EST NOMMÉ : `keyframeFullStateTemoin`, **non exporté**, est le bouton
      des DEUX témoins négatifs que la méthode EXIGE (règle 4 de `METHODE_RETRO_INGENIERIE_FILM` :
      un plancher de faux positifs se MESURE) — le témoin de hasard (+1 bit : 391/62 686 contre
      19 337/62 686) et l'oracle `n2` (état par défaut remplacé par un décalage, la chaîne qui a
      donné les cinq largeurs du lot 1.3). GARDE-RAIL NEUF `keyframe_fullstate_guard_test.go` :
      un fichier NON-test de `filmdec` qui cite `keyframeFullStateTemoin` ou
      `walkKeyframeFullState(` hors de son fichier déclarant rougit.
      LA BOUCLE HISTORIQUE EST **CONSIGNÉE, PAS SUPPRIMÉE**, et l'inventaire dit pourquoi. Grep
      collé en §5 : ZÉRO appelant de production, mais SIX fichiers la citent — cinq instruments
      `_test.go` du paquet (bipède bit-exact, véhicules v5b, grammaire d'écrivain, matrice de
      variantes) et sa déclaration. Supprimer, c'est perdre cinq comparateurs A/B d'autres
      chantiers. Elle est donc **UNEXPORTÉE** (`walkKeyframeBody`, `keyframeBodyVariant(s)`) — le
      compilateur garantit désormais qu'aucune production hors paquet ne peut l'atteindre — avec
      date de pose (2026-09-14), cible de retrait (lot 3.6) et critère mesurable (zéro fichier la
      citant hors de sa déclaration).
      RÈGLE 6 appliquée : l'appariement « record i, frontière i+1 » était écrit deux fois et il en
      fallait deux de plus → centralisé dans `keyframeBornesToutes` / `keyframeBornes`.
- [x] 1.4.2 Les deux consommateurs (et `walkOneKeyframeRecord` si en production) branchés ;
      `Mask` d'un record d'image-clé = tous présents.
      FAIT pour les deux consommateurs (`Mask = ^0` posé par `WalkKeyframeFullState`).
      `walkOneKeyframeRecord` : **[~] NON branché, et c'est motivé** — il n'est PAS en production
      (grep §5 : ses seuls appelants sont `WalkKeyframeRecords` / `ChainKeyframeRecords`, qui
      n'ont eux-mêmes aucun appelant de production, et deux instruments de recherche). Il porte la
      **colonne « production » du tableau archétype × modèle** : la brancher effacerait le
      comparateur qui mesure ce que ce lot gagne. Son en-tête le dit désormais.
      CE QUE LE CHANGEMENT COÛTE, MESURÉ AVANT ET APRÈS (tableaux collés en §5) :
      **ti=11 (instrument)** passe de « 27 marches abouties / 0 chaînée / **0 FERMÉE** » à
      « 27 cassées » sur les 6 films de recherche, et de « 326 / 47 / **0 FERMÉE** » à
      « 326 cassées » sur les 7 bobines. La marche désynchronise à
      `i4 managed-objective-interaction-filter-component`, non porté (lot 3.6). **Ce qui disparaît
      n'a jamais fermé un seul record** : c'était du bruit qui ressemblait à une donnée.
      **ti=12, LE SEUL CHEMIN DE PRODUCTION, SE MESURE AILLEURS — ET LA NOTE 5a SE TROMPE SUR CE
      POINT** (découverte D5 (1.4)). `TestImageCleProductionBalayagesReels` appelle
      `ScanFilmNavpointRadial(dir, map[int]int{})`, c'est-à-dire SANS horloge de manifeste ; or
      `scanChunk` (`navpoint_radial_scan.go`) compte `PacketsNoClock++` et **saute TOUS les
      paquets** quand le chunk n'a pas de `start_ms`. Cet instrument rend donc `KeyRecords 0`
      pour ti=12 sur N'IMPORTE QUEL film, Assaut compris — le « 0 » de la section C.2 de la note
      n'est pas « aucun de ces films n'est un Assaut », c'est « l'instrument ne branche pas
      l'horloge ». Vérifié sur pièces sur le témoin d'Assaut du corpus gate (`c75f33b8`) : même
      instrument, `KeyRecords 0`, alors que le film porte **569 records ti=12 en image-clé**.
      LA VRAIE MESURE DE ti=12, sur ce témoin (`TestImageCleProductionCompteurs`, qui lit les
      payloads sans passer par l'horloge) : ancien cadre **242 marches / 7 chaînées (1,2 %) /
      0 FERMÉE sur 569** ; cadre d'état complet **0 marche / 569 cassées / 0 fermée** (butée
      `i1 managed-navpoint-flags-component`, lot 3.6). Et le corpus gate le chiffre EN
      PRODUCTION, horloge branchée : `coverage.bombArmings.reads` **1 169 → 1 148 (−21)** et
      `.rises` **94 → 73 (−21)**, tandis que **le calque `bombArmings` publié ne bouge PAS**.
      Vingt-et-une lectures, vingt-et-une montées : une montée chacune, c'est-à-dire des points
      ISOLÉS — la signature du bruit, pas d'une jauge qui se remplit. Aucune n'a produit
      d'armement.
- [x] 1.4.3 Compteur expvar `filmdec.keyframe.<ti>.{closed,total}` (ADR 0009) publié par le
      balayage de production ; ratchet 0.A.3 régénéré (attendu : 0 → 14 % sur les 6 films de
      recherche, + 1.3 → 30,8 % ; ti=6/15/18/22 à 100 %).
      COMPTEUR FAIT : `filmdec_keyframe_ti12_{closed,total}` (forme physique snake_case d'ADR
      0009, comme `killsource_*` et `replay_artifact_*`), publié par `replay.decodeFilmBombReads`
      — SEUL appelant de production de `ScanNavpointRadial`. `filmdec` NOMME ses compteurs et ne
      dépend PAS d'`observability` : même patron que `KillSourceHealth.ExpvarPairs`, câblé par
      `killcollector`. **Définition FORTE** (`closed` = atterrissage exact sur la frontière),
      jamais `KeyChained` (définition faible, où la production plafonnait à 1,9 % en fermant 0 sur
      390). Aucun ratio publié. **Aucune variable de paquet ajoutée** : `filmdecVarsGeles` reste à
      96, vérifié.
      **LE RATCHET 0.A.3 N'EST PAS RÉGÉNÉRÉ, ET C'EST UNE CITATION DU PLAN À CORRIGER.** Vérifié
      sur pièces : `keyframe_closure.go` mesure le cadre d'ÉTAT COMPLET **depuis le lot 0.A.3** —
      il n'a jamais mesuré la production. Le « 0 → 14 % » attendu ici était le compteur de
      PRODUCTION, pas ce golden, qui vaut 30,8 % depuis le lot 1.3. Ce lot fait rejoindre la
      production à la mesure, donc le golden ne bouge pas : `TestKeyframeClosureRatchet` reste
      VERT sans régénération, et l'en-tête du fichier porte désormais cette correction.
      Un seul golden a dû être régénéré, et il est nommé : `golden_minibobine_familles.tsv`, UNE
      ligne (`navpointRadial`), dont la population reste VIDE (0 lecture avant, 0 après) —
      l'empreinte hache la structure du balayage, qui gagne deux compteurs à zéro.
- [x] 1.4.4 `GrammarRev` montée.
      `grammar-2026-09-14.3` → `grammar-2026-09-14.4` (quatrième lot du même jour). L'empreinte
      avait ROUGI D'ELLE-MÊME dès le commit de 1.4.1 (« LA GRAMMAIRE A CHANGE SANS MONTEE DE
      REVISION ») : preuve par mutation naturelle, aucune mutation artificielle nécessaire. Golden
      régénéré par sa porte nommée (`-update-grammar-rev`, qui réécrit PUIS échoue), historique
      complété dans le golden.
      `SchemaVersion` NE MONTE PAS (55 → 55) et `KillSourceDecoderRev` non plus : aucun octet cuit
      ne change (équivalence : l'étape `artifact` est identique sur les 10 films) et `killsource`
      ne lit pas ce cadre.

Preuve : `replay-equiv` (différences attendues : `bombArmings` si un témoin s'engage, sinon
aucune) ; corpus gate zéro perte ; `TestImageCleFermetureParArchetype` rejoué par le pilote sur
les 6 films de recherche = chiffres de la note 5a.
RÉSULTAT (2026-09-14), ÉQUIVALENCE : régime court **9 identiques / 1 différent**, et la
différence de `50247b26` (étape `killsource`) est **HÉRITÉE DU LOT 1.3, PAS PRODUITE PAR
CELUI-CI** — contrôle décisif : la même commande jouée dans un worktree détaché au commit de base
`15309e89e` rend la MÊME empreinte obtenue (`2c4ebf2071ca…`). Ce lot produit donc **ZÉRO
différence d'équivalence**, et aucun témoin d'Assaut ne figure dans l'échantillon court (d'où
l'absence de `bombArmings` — cf. le corpus gate, qui en porte un).

RÉSULTAT (2026-09-14), CORPUS GATE : **12 témoins sur 13 à zéro gain / zéro perte ; UN témoin en
PERTE**, et c'est celui que la ligne « Preuve » ci-dessus annonçait — `c75f33b8`, famille
`assaut_bombe`, le seul du corpus où le chemin de production s'engage. **Les deux pertes sont sur
l'axe COUVERTURE, aucune sur le calque publié** : `coverage.bombArmings.reads` 1 169 → 1 148 et
`coverage.bombArmings.rises` 94 → 73 (−21 chacune) ; `bombArmings` est un axe mesuré du gate
(`replaydiff/empreinte_axes.go:65`, famille `assaut`) et **il ne bouge pas** : aucun armement
publié ne change. Les 21 lectures perdues sont celles de la voie image-clé, où l'ancien cadre
marchait 242 records sur 569 en n'en fermant **AUCUN** (7 chaînés, 1,2 %) ; elles produisaient
21 montées, soit UNE montée chacune — des points isolés, la signature du bruit, jamais une jauge
qui se remplit. Schéma inchangé des deux côtés (55 → 55) sur les 13 témoins.
**Le gate n'est donc PAS « zéro perte » au sens littéral, et il ne faut pas l'écrire ainsi** : il
est « zéro perte sur 12 témoins, et sur le treizième une perte de BRUIT mesurée, nommée, sans
effet sur la publication ». C'est le point que le pilote doit vérifier sur pièces avant la fusion.

COMMANDE POUR LE PILOTE (rejeu de la fermeture par archétype, depuis `apps/go-api`) :

```bash
C="C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks"
F="000d5950 00162144 00502e52 0014603f 02784ce1 00ba2e1c"
L=""; for f in $F; do L="$L;$C/$f"; done; L="${L#;}"
CGO_ENABLED=0 CHUNK00_FILMS="$L" go test ./internal/games/halo_infinite/film/filmdec/ \
  -run 'TestImageCleFermetureParArchetype' -v -count=1 -timeout 30m
```

Dernière ligne attendue (mesurée le 2026-09-14, identique avant et après le lot — ce golden
mesure la MESURE, que le lot ne change pas) :
`TOTAL | 19337/62686  30.8% d15134 s7885 u20330 |   391/62686   0.6% … |     0/62686   0.0% …`
— état complet 30,8 %, témoin de hasard 0,6 %, colonne « production » (= l'ANCIEN cadre delta,
rejoué par `walkOneKeyframeRecord`) 0,0 %.

#### Lot 1.5 — L'identité et la table des joueurs lues dans `chunk_00` — M, high

Lecteurs purs, sans consommateur (les consommateurs sont 1.6, 1.7, 1.8). Sources : grammaire des
notes `NOTE_SECTION3_CHUNK00`, `NOTE_SECTION3_SLOTS`, `NOTE_RESIDUS_CHUNK00` et des instruments
`section3_*`, `residus_slots_*`, `profil_roster_*` (`s3sChaine`, `rsChaine`, `rsDelta`).
CLOS le 2026-09-14 (branche `feat/decfilm-15`, 4 commits, `0475daa8b` → le commit de clôture).

##### Rapport 1.5.0 — l'oracle des instruments, mesuré AVANT d'écrire une ligne

Deux passes, collées depuis la sortie brute (§5).

`TestResidusSlotChaineCorpus` sur les **1 351 films du cache** (152 s), une ligne par build :

| build | films | delta(s) mesuré(s) | `s3sChaine` | `rsChaine` | balayage corrigé | films au compte | `s3sChaine` MORT | vacants | **lus + vacants == 32** |
|---|---|---|---|---|---|---|---|---|---|
| `HI_1_13_0` | 1 123 | `0` ×1 123 | 8 596 | 9 003 | 9 003 | 1 123/1 123 | 19 | 26 933 | **1 123/1 123** |
| `HI_1_12_0` | 146 | `0` ×146 | 1 312 | 1 347 | 1 347 | 146/146 | 0 | 3 325 | **146/146** |
| `HI_1_11_0` | 39 | `-2 880` ×39 | 39 | 913 | 913 | 39/39 | 39 | 335 | **39/39** |
| `HI_1_10_0` | 26 | `-2 880` ×26 | 26 | 576 | 576 | 26/26 | 26 | 256 | **26/26** |
| `HI_1_8_0` | 10 | `-4 320` ×10 | 10 | 160 | 160 | 10/10 | 10 | 160 | **10/10** |
| (sans identification) | 5 | `+1 600` ×5 | 5 | 120 | 120 | 5/5 | 5 | 40 | **5/5** |
| `HI_1_9_0` | 1 | `-4 320` ×1 | 1 | 24 | 24 | 1/1 | 1 | 8 | **1/1** |
| `HI_1_4_1` | 1 | `+1 600` ×1 | 1 | 24 | 24 | 1/1 | 1 | 8 | **1/1** |

Sonde jetable sur les mêmes 1 351 films (3,7 s), qui ajoute ce que l'oracle ne dit pas et dont la
section 2 a besoin — le nombre de blocs de registre et la FERMETURE du cardinal de la table par
type, dérivé par `(offset de la chaîne de build − 0x20) − fin du registre` :

| build | films | blocs de registre | offset de la chaîne de build | octets de table par type | = entrées |
|---|---|---|---|---|---|
| `HI_1_13_0` | 1 123 | 50 | `0x0CB414` | 492 | **123** |
| `HI_1_12_0` | 146 | 50 | `0x0CB414` | 492 | **123** |
| `HI_1_11_0` | 39 | 49 | `0x0C7310` | 488 | **122** |
| `HI_1_10_0` | 26 | 49 | `0x0C730C` | 484 | **121** |
| `HI_1_8_0` | 10 | 49 | `0x0C730C` | 484 | **121** |
| `HI_1_9_0` | 1 | 49 | `0x0C730C` | 484 | **121** |
| `HI_1_4_1` | 1 | 49 | `0x0C72F8` | 464 | **116** |
| (sans identification) | 5 | 49 | absente | — | — |

**LA FERMETURE TOMBE SANS UN OCTET DE RESTE, et elle corrige un résidu de la recherche.** Le 123
du build courant est la valeur que l'ÉCRIVAIN écrit (`MOV R9D,0xf60` = 3 936 bits = 492 octets) ;
l'heuristique `lireEntete` des instruments — remonter tant que la valeur tient sur 16 bits — en
comptait **124** (résidu G.2 de `NOTE_SECTION3_SLOTS`). La dérivation structurelle rend le compte
de l'écrivain. Les trois écarts d'en-tête de la note ferment aussi : `16 640 + 4×1` (123−122),
`16 640 + 4×2` (123−121), `16 640 + 4×7` (123−116).

**LES 5 FILMS SANS SECTION D'IDENTIFICATION, NOMMÉS** : `03af54c3`, `13b00e35`, `47d20b5d`,
`50247b26`, `a349fea8`. `50247b26` est dans l'échantillon court de `replay-equiv`.

- [x] 1.5.1 `filmdec.ReadFilmIdentity(chunk0) (FilmIdentity, error)` : section 2 (table par type,
      version en clair, build, saveur, identifiant de build, changelist) ; `ErrNoFilmIdentity`
      typé pour les 5 films sans section ; horodatage du match (32 bits) porté.
      FAIT, et **sans aucun offset absolu** : la carte de la note ne vaut que pour
      `HI_1_12_0`/`HI_1_13_0` (rapport 1.5.0). Le lecteur dérive la fin du registre du parse
      (cadrage du jeu, lot 1.2), ancre la section sur la chaîne de build — l'ancre que la
      recherche a suivie — et déduit tout le reste relativement à elle. Trois erreurs typées, et
      la mesure a corrigé une attente : couper dans la section 2 rend `ErrChunk00Truncated`, PAS
      `ErrNoFilmIdentity`, parce que le registre s'arrête à sa fin STRUCTURELLE et qu'un tampon
      coupé avant cette fin épuise la boucle de blocs. La cause première est la troncature.
      CONTRÔLE GRATUIT PASSÉ : les sept bobines rendent des horodatages ORDONNÉS par build
      (2023-09-02 → 2026-07-23). Une lecture fausse ne produirait pas sept dates plausibles NI
      leur ordre correct par rapport à un champ indépendant.
- [x] 1.5.2 `filmdec.ReadPlayerTable(chunk0, ident) ([]PlayerSlot, PlayerTableReport, error)` :
      32 slots, 16 champs, décalage d'un bit (`0x0CB45C`), largeur du bloc de personnalisation
      PAR BUILD (table `player_table_profile.go` : 1 852 / 1 492 / 1 312 / 2 052 o, un
      commentaire de provenance par ligne), slots vacants écartés ; `ErrUnknownBuild` typé +
      compteur expvar `filmdec.unknown_build.<build>` (principe 10) ; le rapport porte le
      calibrage lu sur le film comme CONTRÔLE (accord / contradiction).
      FAIT. Compteur sous sa forme PHYSIQUE `filmdec_unknown_build_<build>` (snake_case d'ADR
      0009, comme `killsource_*` et `filmdec_keyframe_ti12_*`) ; il est NOMMÉ, pas câblé — le lot
      1.5 ne livre aucun consommateur, et son premier appelant de production est le registre
      d'identité du lot 1.6, ce que l'en-tête de `UnknownBuildExpvarPairs` écrit avec sa date.
      LE CONTRÔLE A CINQ FAMILLES, PAS DEUX, ET CHACUNE A ÉTÉ IMPOSÉE PAR UNE MESURE : accord /
      vacant / **invisible** / **parasite** / contradiction. « Invisible » = l'intervalle porte un
      enregistrement que la MARCHE a lu et que le balayage ne pouvait pas voir ; « parasite » = un
      bout du couple n'a pas été retenu. Sans ces deux-là, un enregistrement bien réel se lisait
      comme une contradiction de grammaire.
      DEUX DÉFAUTS TROUVÉS PAR LA MESURE ET CORRIGÉS DANS LE LOT, tous deux écrits dans le code :
      (a) les LECTURES vont jusqu'au bout du TAMPON, pas jusqu'au dernier octet écrit — un
      enregistrement vacant est écrit entièrement à zéro, donc les 24 vacants de queue d'une
      partie d'arène tombent APRÈS le dernier octet non nul, et borner là faisait échouer 24 des
      32 slots ; (b) « la marche ferme à 32 slots » NE SUFFIT PAS comme critère d'acceptation,
      parce que le bourrage de queue satisfait le prédicat de vacance indéfiniment : « 1 occupé +
      31 vacants » ferme avec n'importe quelle largeur (mesuré : quatre largeurs fausses sur cinq
      bobines). Le critère retenu est double — la marche ferme à 32 ET elle VISITE TOUS les
      enregistrements du balayage ; entre deux lectures recevables, la plus complète gagne.
      **AUCUN SEUIL N'EST PORTÉ EN PRODUCTION** : le `40 000` bits du regroupement terminal des
      instruments n'existe pas dans ce lecteur, et c'est ce qui lui fait lire juste neuf films que
      l'instrument lit faux (cf. 1.5.4).
- [x] 1.5.3 Champs publiés par slot : `FilmIndex` (= rang), `XUID`, `Gamertag`, champs courts.
      FAIT, plus le jeton de session (48 bits), la position et la longueur en bits. Les neuf
      champs courts sont publiés dans `PlayerSlotShorts` — ils sont mesurés CONSTANTS sur le
      corpus, et c'est justement pour cela qu'ils se publient : un champ constant qui se met à
      varier est le premier signe qu'une grammaire a bougé.
      **`FilmIndex` EST LE RANG ABSOLU DANS LA TABLE DE 32, VACANTS COMPRIS** — l'index du tableau
      que l'écrivain parcourt (`enregistrement += 0x1450`). CE QUI EST MESURÉ, ET CE QUI NE L'EST
      PAS : l'ordre des enregistrements EST le `player_index` de production (`filmIndex − rang`
      constant sur 76/76 films, phase 2), mais aucun de ces 76 films ne porte de vacant INTERCALÉ,
      donc l'oracle ne sépare pas « rang absolu » de « index parmi les occupés ». Les 13 films du
      cache à vacant intercalé n'ont AUCUN document de rejeu (mesuré le 2026-09-14) : la question
      n'est pas tranchable sur ce corpus. Le rapport porte donc `InterleavedVacant`, pour que le
      consommateur du lot 1.6 sache quand les deux lectures divergent. Découverte D3 (1.5).
- [x] 1.5.4 Tests : unitaires sur les mini-films (7 builds : 32 slots lus, gamertags imprimables,
      rang = index) ; corpus `CHUNK00_FILMS` : 32 slots sur 1 351 films (oracle des instruments) ;
      mutation : fausser une largeur de build rougit.
      FAIT. Unitaires SANS garde d'environnement (donc joués en CI) sur les sept bobines : 32
      slots, gamertags imprimables, XUID de la plage Xbox, rang = index, calibrage du film égal à
      celui du profil, **0 contradiction sur 7/7**. Les comptes attendus sont ceux que `rsChaine`
      rendait AVANT que ce lecteur existe.
      **LA GARDE DU CORPUS EST `CHUNK00_CORPUS` (UNE RACINE), PAS `CHUNK00_FILMS` (UNE LISTE), et
      ce n'est pas une commodité** : 1 351 chemins absolus séparés par `;` pèsent une centaine de
      kilo-octets, au-delà de la borne d'une variable d'environnement Windows (32 767 caractères).
      La forme en liste reste acceptée pour un petit corpus. Écart au libellé du plan, assumé.
      CORPUS (82 s) : **1 346 films lus à 32 slots** (12 080 occupés + 30 992 vacants), 5 mis de
      côté sans section, **0 build inconnu, 0 contradiction de grammaire, 0 calibrage en
      désaccord**.
      **NEUF FILMS OÙ LA PRODUCTION ET L'INSTRUMENT DIVERGENT, ET C'EST L'INSTRUMENT QUI SE
      TROMPE** — tableau collé en §5. Sur `19ef6b04`, `23ffd885`, `3104391d`, `3b1cfde3`,
      `59b8abb9`, `652907bb`, `92f7c713`, `a92bab93`, `d4ddf054`, la production lit 7 ou 8
      enregistrements là où `rsChaine` en lit 1 à 6. Chacun porte DEUX écarts de balayage au-delà
      de 40 000 bits : un slot vacant intercalé ajoute 16 499 bits à l'écart entre deux
      enregistrements et pousse le regroupement terminal de l'instrument à perdre la TÊTE de la
      table — le défaut que la phase 2 nommait déjà (« le lecteur perd la tête sur 4 films sur
      76 »). Le test EXIGE l'explication : une divergence sans coupure de grappe le fait échouer,
      et la production ne doit JAMAIS lire moins que l'instrument.
      **CE QUE LA MESURE DU CORPUS APPREND SUR L'ORACLE LUI-MÊME** : son critère
      « lus + vacants == 32 sur 1 351/1 351 » est satisfait par le BOURRAGE de queue, donc il est
      vrai ET il ne prouve pas la tête de la table. Découverte D1 (1.5).
      MUTATION JOUÉE : `HI_1_13_0`/`HI_1_12_0` passés de 1 852 à **1 848** octets (quatre octets),
      `TestReadPlayerTableSurLesBobines` **ROUGE** sur les deux bobines concernées (« aucune table
      de 32 slots dans le corps de chunk_00 »), plus `TestPersonnalisationOctetsProfil` et
      `TestReadPlayerTableTronquee`. Fichier restauré PAR NOM, md5 identique
      (`1d58470a8b22f70d33a3a1f5e190ea9a`). Le test automatique de mutation dit ce qu'il n'affirme
      PAS : une largeur trop GRANDE fait toujours fermer « 1 occupé + 31 vacants » dans le
      bourrage — c'est pour cela que le critère retient la lecture la plus COMPLÈTE, et que le
      test vérifie la DÉGRADATION plutôt que l'échec.
      ENTRÉE TRONQUÉE (obligatoire, leçon du lot 1.2) : huit coupes d'identité — tampon vide,
      en-tête seul, frontière de bloc, dernier bloc entier, avant / dans / après la chaîne de
      build, juste avant le corps — et deux coupes de table. Aucune ne panique, toutes rendent une
      erreur typée, aucune ne rend de lecture partielle.
- [x] 1.5.5 `GrammarRev` montée.
      `grammar-2026-09-14.4` → `grammar-2026-09-14.5` (cinquième lot du même jour). L'empreinte
      avait ROUGI D'ELLE-MÊME dès le commit de 1.5.1 (« LA GRAMMAIRE A CHANGE SANS MONTEE DE
      REVISION », 151 → 154 fichiers) : preuve par mutation naturelle, aucune mutation
      artificielle nécessaire. Golden régénéré par sa porte nommée (`-update-grammar-rev`, qui
      réécrit PUIS échoue), historique complété dans le golden.
      `SchemaVersion` NE MONTE PAS (55 → 55) et `KillSourceDecoderRev` non plus : ces lecteurs
      n'ont AUCUN consommateur, donc aucun octet cuit ne change — et l'équivalence le confirme
      (10/10 identiques, les 50 étapes de balayage comprises).

Preuve : `replay-equiv` zéro différence (aucun consommateur) ; corpus gate zéro différence.
RÉSULTAT (2026-09-14) : **LES DEUX PREUVES SONT À ZÉRO DIFFÉRENCE, AU SENS LITTÉRAL.**
`replay-equiv` rend **10 IDENTIQUES sur 10** au régime court (deux sous-ensembles séquentiels de
5, 3 min 58 + 2 min 10), les 50 étapes de balayage et l'artefact compris ; le corpus gate rend
**13 témoins sur 13 à 0 gain / 0 perte, schéma 55 → 55 partout** (9 min). Ce n'est pas une
surprise, c'est ce que « lecteurs purs, sans consommateur » veut dire, et la mesure le dit sans
détour : `grep -rn "ReadFilmIdentity(\|ReadPlayerTable(" --include=*.go internal/ cmd/ | grep -v
_test.go` ne rend que les DEUX lignes de déclaration. `SchemaVersion` reste à 55,
`KillSourceDecoderRev` à `killsource-2026-09-12` : aucune recuisson ni backlog dus par ce lot.

#### Lot 1.6 — Le registre d'identité prend la table du film comme lien direct — M, high

Décision utilisateur du 2026-09-07 : « l'index c'est l'index », table d'identité unique par
film, morts en repli. Sur pièces : registre d'identité du constructeur (`identity_registry*.go`),
`replaybuild/matchfacts.go` (`equipesParXUID`, `PlayerIndexTable`).
CLOS le 2026-09-14 (branche `feat/decfilm-16`, 5 commits, `2f32f576e` → le commit de clôture).

##### Rapport 1.6.0 — la mesure AVANT de coder, qui a changé la conception du lot

Sonde jetable sur les 8 builds du golden (supprimée après la mesure), table du film confrontée à
la table d'index du fixture — collée depuis la sortie brute (§5) :

| film | build | sièges au film | table d'index | accord | contradiction | le film SEUL | le contrôle SEUL |
|---|---|---|---|---|---|---|---|
| `000d5950` | `HI_1_13_0` | 8 | 8 | 8 | 0 | 0 | 0 |
| `a521164d` | `HI_1_4_1` | 24 | 27 | 23 | 0 | **1** | **4** |
| `60ae07c4` | `HI_1_8_0` | 8 | 8 | 8 | 0 | 0 | 0 |
| `11de8353` | `HI_1_9_0` | 24 | 27 | 23 | 0 | **1** | **4** |
| `111fa685` | `HI_1_10_0` | 24 | 25 | 24 | 0 | 0 | **1** |
| `e5adf7b2` | `HI_1_11_0` | 23 | 28 | 23 | 0 | 0 | **5** |
| `bcb6d393` | `HI_1_12_0` | 8 | 11 | 8 | 0 | 0 | **3** |
| `fb1a1a72` | `HI_1_13_0` | 8 | 8 | 8 | 0 | 0 | 0 |

**DEUX FAITS, ET LE SECOND A CHANGÉ LA CONCEPTION DU LOT.** (1) Là où les deux lectures parlent
du même joueur, elles DISENT LA MÊME CHOSE : **125 accords, 0 contradiction**, sur huit builds et
six générations de jeu. (2) **La table du film est celle du DÉBUT du film** : un joueur qui
rejoint en cours de partie n'y a pas de siège (0 à 5 par film, **13 au total**), et inversement
elle assoit un joueur que le balayage des chunks ne trouve pas (2 films sur 8). Le brief disait
« la table du film devient le lien, l'inférence par le fil des morts reste le repli des 5 films
sans section » : **REMPLACER l'une par l'autre aurait PERDU 13 joueurs sur 8 films** — une perte
au corpus gate. La table du film PRÉCÈDE donc la lecture des chunks au lieu de la remplacer, et
le repli est nommé et compté sur son diagnostic propre (« ce xuid n'a pas de siège »), D14 (b).
0 vacant INTERCALÉ sur les 8 builds : la question D3 (1.5) ne se pose sur aucun d'eux.

- [x] 1.6.0 (ajouté par le pilote, hérité du lot 1.5) Le compteur expvar
      `filmdec_unknown_build_<build>` est CÂBLÉ chez le consommateur de production ; mesure à 0
      sur les témoins ; un film au build inconnu est mis de côté avec son erreur typée (D-4).
      FAIT. Le consommateur est `replay.ScanFilmPlayerTable` (`film_player_table.go`), appelé par
      l'étage de balayage à l'étape `filmTable`, et il publie
      `filmdec.UnknownBuildExpvarPairs` — le patron de `publierFermetureImageCle`. **LE CÂBLAGE
      EST PROUVÉ PAR MUTATION, ET IL FALLAIT** : aucun des 1 351 films du cache n'a de build hors
      profil (mesure 1.5.4), donc aucun corpus ne peut le déclencher. Le test patche la chaîne de
      build d'une bobine saine en `HI_9_99_0` à l'offset que `ReadFilmIdentity` DÉSIGNE, et exige
      `filmdec_unknown_build_hi_9_99_0` à +1. Mesuré à **0 sur les 13 témoins du corpus gate et
      les 10 films d'équivalence** (aucune ligne de refus au journal). Cinq causes de refus
      NOMMÉES, dont trois **mesurées et non supposées** (`tampon vide` → `tronque`, `corps amputé
      de moitié` → `table_introuvable`, `coupe au début du corps` → `tronque`) ; couper la QUEUE
      d'un tampon sain ne refuse RIEN — les slots vacants y sont des zéros.
- [x] 1.6.1 Provenance `film_table` dans `identity.players` : le lien `index ↔ xuid ↔ gamertag`
      vient de 1.5 quand la section existe ; la table de la base devient un CONTRÔLE (compteurs
      `accord / contradiction / silence` publiés dans `coverage.identity`) ; l'inférence par le
      fil des morts reste le repli des 5 films sans section (D2).
      FAIT, avec **UN ÉCART AU LIBELLÉ, imposé par le rapport 1.6.0** : le repli ne sert pas
      seulement les 5 films sans section, il sert aussi les **13 joueurs** que la table n'assoit
      pas sur 5 des 8 builds. Il reste un repli au sens de D14 — déclenché sur le diagnostic
      « ce xuid n'a pas de siège », jamais sur un désaccord — et il est compté
      (`coverage.identity.filmTable.repli`). Le CONTRÔLE est la lecture des chunks de
      réplication, et c'est bien « la table de la base » : son roster d'entrée vient de la feuille
      de match (`rosterOf(deaths, opt.RosterXUIDs)`), sans laquelle elle ne voit que les joueurs
      qui meurent. Un vacant INTERCALÉ est une ABSTENTION nommée (`vacant_intercale`) et non une
      lecture : D3 (1.5) reste ouverte, et le compteur suffit — aucun des 8 builds ni des
      13 témoins n'en porte.
- [x] 1.6.2 Roster : gamertag du film quand disponible ; taille réelle de l'escouade publiée
      comme DONNÉE (aucune règle UI ne change, §1.2).
      FAIT. `buildRoster` prend la table EFFECTIVE du registre et les gamertags du film. Gains
      mesurés : `a521164d` et `11de8353` passent de **27 à 28 joueurs** ; `111fa685` idx=10
      (« FlukiestGolf ») et `e5adf7b2` idx=13 (« MarshallG6443 ») étaient publiés **SANS NOM** et
      prennent le leur — le fil des morts ne nomme que les joueurs qui MEURENT, et ceux-là n'ont
      aucune mort. Taille d'escouade : `coverage.identity.filmTable.sieges` (8/24/8/24/24/23/8/8),
      et ce n'est PAS le compte du roster, qui porte en plus les arrivants.
- [x] 1.6.3 Cuisson hors ligne (sans faits) : le roster est complet (golden par build de 0.A.2
      régénéré avec justification).
      FAIT, et c'est le gain le plus fort du lot. `roster_hors_ligne_test.go` cuit les 8 builds
      SANS feuille, sans tableau, sans bots, la table d'index restreinte aux xuids du fil des
      morts (modèle EXACT de la lecture hors ligne : `ScanPlayerIndices` ne cherche que ceux-là,
      et le corpus donne 0 désaccord d'index). Sans / avec la table du film :
      8→8 · **26→27** · 8→8 · **26→27** · **24→25** · **26→27** · 11→11 · 8→8, et **0 siège de la
      table absent du roster hors ligne sur 8/8**. Le trou structurel « un joueur qui ne meurt
      jamais est invisible » (`3372e7eb`, 6 publiés pour 8) se comble SANS base.
- [~] 1.6.4 `SchemaVersion` 55 → 56, chronique, empreinte de forme (0.B.4) régénérée, fixtures de
      contrat régénérées, `openapi.yaml` + `generate-types` si un champ apparaît.
      COUVERT PAR 1.6.1 ET 1.6.5, et ce n'est pas un choix : l'empreinte de forme REFUSE de se
      refiger quand la forme change sans montée de version, donc la montée tombe dans le PREMIER
      commit qui change la forme (1.6.1), et la chronique avec elle (garde-rail
      `document_shape_test.go`). Chaîne complète vérifiée à la clôture : `SchemaVersion = 56`,
      entrée de chronique v56 (identité + seuil), raison écrite dans `structure_test.go`, empreinte
      de forme `2c1ea5c7b555c95f`, jumeau `replaydoc.FilmTableCounts`, `replayview.toFilmTableCounts`
      (parité verte), `openapi.yaml` + `generated.ts` régénérés (schéma `FilmTableCounts`), 8
      fixtures de contrat `replay_schema_56_<short8>.json.gz` (**2 565 193 o**, plafond 3 145 728,
      un seul jeu vivant — les 8 fixtures 55 supprimées), `MIN_RENDERABLE_SCHEMA_VERSION = 27`
      **INCHANGÉ**, `make check-types` et `make test-web` verts.
- [x] 1.6.5 **Refus de publication COMPTÉS** : `[~]` pour le compteur (fait au lot 1.0.4) ;
      `DefaultMinPoints = 2` → **1** dans ce lot (décision utilisateur du 2026-09-14, item 1.9.12
      : « si le film le dit, on publie »), le compteur reste et **tombe à 0 sur les 8 builds**.
      FAIT. **20 vies entrent** (1+6+4+3+2+2+2+0), traces 104→105, 243→245, 243→246, 173→177,
      179→185, 56→58, 254→256, 147→147. ORACLE MESURÉ, tableau collé en §5 : **1 mort ÉCRITE**
      (`e5adf7b2` slot 689, écart 72 ms), **5 fins de film** (la réplication du slot ne reprend
      jamais), **14 ORPHELINES** — toutes fermées sur un TROU DE RÉPLICATION (`cause = cut`), la
      mort la plus proche du même joueur à 0,95 s à 300 s. Consigné en §4 comme DÉFAUT DE LECTURE
      (D1 (1.6)), jamais filtré. Instrument permanent : `vies_un_echantillon_test.go`, qui gèle la
      table par build. La source de mort est le fil des morts du film ; le dead-state de bipède
      n'est pas lisible sur cette branche (`ti=40` vit sur `wt/vehicule-deadstate`, non fusionnée),
      et c'est dit dans l'en-tête de l'instrument plutôt que tu.

Preuve : corpus gate zéro perte ; gains nommés (`identity.coverage.*.direct`, `unnamedLives` ne
monte nulle part) ; `replay-equiv` différences localisées aux balayages d'identité.

RÉSULTAT (2026-09-14), ET IL NE TIENT PAS LA LIGNE « PREUVE » À LA LETTRE — c'est écrit ici plutôt
que tu :

1. **`replay-equiv` régime court : 10/10 films, DEUX lignes changent et deux seulement** —
   `filmTable` (l'étape neuve du lot 1.6.0) et `artifact` (+49 à +987 o). **Les 49 autres
   balayages sont IDENTIQUES sur les dix films.** Classification écrite AVANT acceptation du
   re-figeage (§5). Aucune régression de décodage.
2. **Corpus gate `--base=06530e63d` : 267 gains, 39 « pertes » sur 13/13 témoins, exit 1 — et les
   39 tiennent en QUATRE classes, toutes imputables au seul `DefaultMinPoints = 2 → 1`
   (décision utilisateur, item 1.9.12), AUCUNE à la table du film** :
   (a) `coverage.tracks.minPoints 2 → 1` (13/13) — c'est le SEUIL lui-même, publié ; le gate lit
   une baisse de nombre là où il y a un élargissement de publication ;
   (b) `coverage.tracks.refusedMinPoints` et `refusedPoints` → 0 (11/13) — ce sont les REFUS qui
   tombent, c'est-à-dire exactement le gain ;
   (c) `bounds.min*` sur `084a804d` et `e5adf7b2` — les bornes s'ÉLARGISSENT (un minimum baisse) ;
   conséquence de cadrage consignée D5 (1.6) ;
   (d) `coverage.bridge.unnamedLives` 0 → 1 sur `084a804d` (avec `unnamedLivesContested` et
   `flagCarries.ambiguousSlot`) et 334 → 337 sur `a349fea8` — **la ligne « `unnamedLives` ne
   monte nulle part » n'est PAS tenue** : le seuil à 2 masquait quatre vies sans nom, les publier
   les rend visibles. Défaut de nommage PRÉEXISTANT, consigné D4 (1.6), à arbitrer avec
   l'utilisateur avant la recuisson du parc.
3. **Contrôle décisif `--base=444d0b7c6`** (le commit de 1.6.3, soit la table du film SANS le
   changement de seuil), joué pour SÉPARER les deux causes : **le même ensemble de pertes,
   métrique par métrique et valeur par valeur** (39, schéma 56 → 56), et 250 gains contre 267.
   Donc **la table du film coûte ZÉRO perte sur 13/13 témoins**, et ce qu'elle apporte est
   l'écart des gains : +1 sur neuf témoins, **+3 sur `111fa685` et `e5adf7b2`**, +1 sur
   `a349fea8` et `60ae07c4`. Les trois témoins dont le gain tombe à 0 (`fb1a1a72`, `bf15f7ab`,
   `bfecd02b`) n'avaient de gain que par le seuil.

#### Lot 1.7 — L'équipe réelle dans l'artefact, sans base — M, high

Sur pièces, RE-VÉRIFIÉ au commit de base `943d8cf4b` : les trois citations du plan avaient BOUGÉ,
et c'est la règle 4 qui l'a rattrapé — `Team: -1` vit dans `replay/tracks_publication.go:143`
(sorti de `build.go` au lot 1.0), le commentaire « l'ÉQUIPE N'EST PAS DANS LE FILM » à
`document.go:563` (et un second, sur la zone de retour de drapeau, à `:351`), `TeamOf` fourni par
la base à `matchfacts.go:254` (exact), `CarrierTeamUnknown` à `flag_carries.go:256` et
`document_objectives_live.go:318`.
CLOS le 2026-09-14 (branche `feat/decfilm-17`, 4 commits, `3382acb88` → le commit de clôture).

##### Rapport 1.7.0 — la mesure AVANT de coder, et ce qu'elle a changé au lot

**PREMIÈRE PASSE — LA NOTE SE REJOUE À L'IDENTIQUE.** `TestEquipeFilmOracleXuid` sur les 22 films
de `NOTE_EQUIPE_FILM_2026-09-12.md`, oracle reconstruit depuis la sauvegarde
`pre-chaine-2026-09-09` : **16 films sur 18 en accord TOTAL, 160 slots sur 176**, dont
`03af54c3` et `213a87dc` à **24/24**, plancher de bruit **0 touche sur 576** décalages voisins,
et les deux FFA (`1950c59b`, `610363ee`) à `[0 0 0 0 0 0 0 0]` — le film y dit « aucune équipe »
et c'est la BASE qui fabrique un camp par joueur. 20,7 s. Tableau collé en §5.

**SECONDE PASSE — ET ELLE A CHANGÉ L'APPARIEMENT DU LOT.** Sonde jetable (supprimée après la
mesure) sur les 18 films de l'union « 8 builds ∪ 13 témoins ∪ échantillon court », confrontant la
table des joueurs de `chunk_00` (lot 1.5), la table d'index des chunks de réplication, la feuille
de match et les entités ti=9. Deux faits, le second décisif :

1. **L'APPARIEMENT ORDINAL DE LA NOTE TIENT — et il n'est plus nécessaire.** Sur le premier
   paquet d'image-clé, le k-ième record ti=9 porte bien le camp du k-ième siège : **8/8 sur dix
   films, 24/24 sur `084a804d` et `111fa685`, 23/23 sur `e5adf7b2`**. Les deux « 23/24 »
   (`a521164d`, `11de8353`) ne sont PAS des désaccords : le rang en écart est un siège dont le
   xuid est ABSENT de la feuille de match (`manistoff` idx=18, `Iskra 20252993` idx=23), donc
   l'oracle prédisait un camp qu'il n'avait pas.
2. **LE FILM ÉCRIT L'INDEX DE JOUEUR DANS LE RECORD ti=9.** Le premier `R(6)` de son état par
   défaut (`consumeDefaultStateTI9`, `FUN_1410d7540`) vaut exactement le rang du siège, entité
   par entité, **CONSTANT sur toute la vie de l'entité**, sur les 18 films et les 7 builds — y
   compris sur les deux films SANS section d'identification, où la table de `chunk_00` est
   refusée et où il continue de rendre `0..23` / `0..24`. Le contrôle qui interdit d'y lire un
   simple ordinal : sur `50247b26` la suite lue est **trouée** (`0 1 3 4 … 22 24`). Un ordinal
   est contigu par construction ; celui-ci ne l'est pas.

CONSÉQUENCE SUR LE LOT : 1.7.1 ne fait AUCUN appariement. Il lit l'index que le film écrit, et
l'ordinal de la note devient le CONTRÔLE (test `TestScanPlayerTeamsIndexEstLeSiege`). C'est la
doctrine D13 appliquée à la lettre — la grammaire prime sur l'inférence — et c'est ce qui rend
les REMPLAÇANTS lisibles : ils n'ont pas de siège, mais ils ont un index et une équipe.

- [x] 1.7.1 `filmdec.ScanPlayerTeams(fc) (map[filmIndex]designator, TeamScanReport)` : records
      d'image-clé ti=9 par la boucle d'état complet (1.4), composant i0 sur 4 bits à
      `108 + 32 + état(ti) + 32` (dérivé, jamais 186 en dur), valeur = désignateur + 1, 0 =
      aucune ; appariement entité ti=9 → joueur selon `NOTE_EQUIPE_FILM_2026-09-12.md` ; stabilité
      par entité sur le film (le rapport compte les divergences).
      FAIT, et la position est DOUBLEMENT dérivée : l'index sort de l'état par défaut rejoué à
      `en-tête + n1`, le désignateur sort de la boucle de composants de PRODUCTION (qui nomme i0
      depuis le registre DU FILM), et **la concordance des deux est VÉRIFIÉE à chaque record** —
      si le composant ne tombe pas là où la grammaire le place, la lecture est REFUSÉE. `186`
      n'apparaît nulle part dans le code du lot.
      **L'APPARIEMENT N'EST PLUS ORDINAL** (rapport 1.7.0, fait 2) : `readManagedPlayerDefaultState`
      rend le premier `R(6)` de ti=9, qui EST l'index de joueur. Écart au libellé, assumé et
      mesuré ; l'ordinal de la note reste le contrôle, joué sur les sept bobines.
      DOMAINES ET REFUS, tous comptés : index hors de la table de 32 (**UN sur le corpus** —
      `111fa685`, index 59, un unique paquet), valeur brute hors de `0..9`, marche qui n'atteint
      pas i0, i0 qui n'est pas le désignateur, ti=9 absent du registre. Un index dont deux
      lectures ne s'accordent pas n'est PAS publié.
      Tests sans garde d'environnement (donc en CI) sur les sept bobines par build : **0 record
      inatteint, 0 divergence d'entité, 0 divergence d'index** ; témoin négatif mesuré — le même
      champ relu à UN bit diffère sur **1 616 lectures sur 1 617** ; dix troncatures, aucune
      panique, aucune lecture partielle.
- [x] 1.7.2 Règle V4 : le film est la SEULE source. `Track.Team`, `roster[].team` (champ neuf)
      et `TeamOf` des drapeaux viennent du désignateur ; 0 = aucune équipe (FFA), muet =
      inconnue ; aucun repli sur la base. La base n'entre que dans
      `coverage.teams.{film, accord, contradiction, silence}` ; une contradiction ne se corrige
      pas en silence, elle se compte et le relecteur la lit.
      FAIT. `FlagInput.TeamOf` **DISPARAÎT** (et `replaybuild.equipesParXUID` avec lui, qui
      faisait doublon avec `teamByXUID`) ; la feuille de match arrive par `Options.ScoreboardTeams`
      et n'alimente QUE les trois compteurs de contrôle. L'invariant « jamais son propre drapeau »
      tient désormais sur une cuisson HORS LIGNE, où il se taisait faute de lignes de match.
      **`roster[].team` EST UN POINTEUR, ET C'EST LA SEULE FORME JUSTE** : trois états existent,
      pas deux — ABSENT (le film n'a pas nommé ce joueur, ou l'artefact précède le schéma 57),
      `-1` (aucune équipe), `0..8` (le camp). Un entier nu ferait dire « camp 0 » à tout artefact
      ancien, et `MIN_RENDERABLE_SCHEMA_VERSION` vaut 27. `Track.Team` reste un entier : il
      existe depuis la v2 et vaut `-1` sur tous les artefacts antérieurs, donc il ne ment pas ;
      `coverage.teams.noTeam` et `.unread` distinguent ce que le champ ne distingue pas.
- [x] 1.7.3 Les événements d'objectif (1.1.2) prennent l'équipe de l'octet 37 ; contrôle contre
      l'équipe du porteur (compteur).
      FAIT. `objectiveevents.Extract` rend `([]domain.ObjectiveEvent, TeamControl)` et pose
      `team_id` depuis `FooterEvent.Team` ; le roster n'en est plus que le contrôle
      (`Film / Accord / Contradiction / Silence`). **LA VÉRITÉ TERRAIN TIENT APRÈS LE
      BASCULEMENT** : `TestExtractCTFCaptureCount` rejoue `0f9550e5` (5-0) et `53ce4390` (1-2)
      sur le cache, le partage par équipe reste EXACT, et le test exige désormais
      `contradiction == 0`. Deux tests neufs sans film ni base tiennent les deux moitiés que le
      corpus ne tient pas : roster VIDE → l'équipe du pied est publiée quand même ; roster qui
      CONTREDIT → la valeur publiée ne bouge pas, le compteur monte.
- [x] 1.7.4 Commentaires `document.go:590` et `flag_assign.go:31` corrigés.
      FAIT, et le grep en a trouvé **huit autres** : `document.go` (`Track.Team` ET la zone de
      retour de drapeau), `flag_assign.go`, `flag_carries.go`, `flag_assign_test.go`,
      `document_vehicles.go`, `replaybuild/flagspawns.go`, `objectiveevents/extract.go` (doc de
      `Roster`), `domain/objective_events.go` (**découverte D4 (1.1) FERMÉE** : « team unreliable
      sur certains matchs » décrivait l'octet 55), `sync/replayartifacts/positions.go` et
      `persist/player_positions_persister.go`. Sur ces deux derniers la RÈGLE ne change pas —
      elle disait déjà « l'artefact prime quand il porte l'équipe » — mais sa JUSTIFICATION si :
      ce n'est plus « le film ne la porte pas », c'est « un artefact antérieur au schéma 57 ne la
      porte pas ».
- [~] 1.7.5 `SchemaVersion` 56, chronique, forme, fixtures, goldens par build (attendu :
      `Track.Team != -1` hors ligne, S5).
      COUVERT PAR 1.7.2, et pour la même raison qu'au lot 1.6.4 : l'empreinte de forme REFUSE de
      se refiger quand la forme change sans montée de version, donc la montée tombe dans le
      PREMIER commit qui change la forme. Chaîne vérifiée à la clôture : `SchemaVersion = 57`,
      entrée de chronique v57, raison écrite dans `structure_test.go`, empreinte de forme
      `1344869006f05f16`, jumeau `replaydoc` (`RosterEntry.Team`, `TeamCoverage`),
      `replayview.toTeamCoverage` (parité verte), `openapi.yaml` + `generated.ts`, **8 fixtures
      de contrat `replay_schema_57_<short8>.json.gz` (2 566 758 o, plafond 3 145 728 ; les 8
      fixtures 56 supprimées)**, codec du fixture d'entrées v20 → **v21**,
      `MIN_RENDERABLE_SCHEMA_VERSION = 27` **INCHANGÉ**, `make check-types` et `make test-web`
      verts.
      **S5 TENU SUR LES HUIT BUILDS, CUISSON HORS LIGNE** (aucune base ouverte) : 105/105,
      177/177, 185/185, 246/246, 245/245, 256/256, 58/58 et 147/147 vies portent une équipe du
      film ; **0 joueur non lu, 0 divergence** ; le contrôle est à `silence` partout, ce qui EST
      la définition d'une cuisson sans feuille de match. Le golden d'assemblage porte désormais
      une section `### EQUIPES` : sans elle, S5 n'aurait été vérifiable que hors du dépôt.

Preuve : corpus gate zéro perte, gains nommés (`teams.accord` = 160/176 et 24/24 sur les BTB des
notes, `CarrierTeamUnknown` inchangé ou en baisse) ; FFA : `teams.film = 0`, base conservée.

RÉSULTAT (2026-09-14) : **LA LIGNE « PREUVE » EST TENUE, À UNE FORMULATION PRÈS QUI EST ÉCRITE
ICI PLUTÔT QUE TUE.**

1. **CORPUS GATE : ZÉRO PERTE SUR 13 TÉMOINS SUR 13, EXIT 0.** C'est le premier lot de M1 dont le
   gate sort vert au sens littéral — 9 gains sur onze témoins, 10 sur `111fa685`, 5 sur
   `a349fea8`, schéma 56 → 57 partout. **`CarrierTeamUnknown` : aucune perte sur l'axe `ports`
   sur 13/13**, donc inchangé ou en baisse.
2. **`teams.accord` NE SE MESURE PAS EN 160/176 : CE CHIFFRE EST CELUI DE LA NOTE, PAS DU
   DOCUMENT.** Les 160/176 et les 24/24 se rejouent à l'identique (§5, passe 1 du rapport 1.7.0),
   et c'est bien la preuve demandée — mais l'artefact, lui, compte PAR JOUEUR DU ROSTER et non
   par slot d'un corpus de recherche. Sa mesure est : `accord` = `film` sur huit des dix films
   d'équivalence, `28/27` sur `a521164d` et `11de8353` (un siège que la FEUILLE ne porte pas :
   un silence, pas une contradiction), et **0 contradiction sur 36 cuissons** — preuve par
   l'absence de l'avertissement que `logTeamCoverage` émet dès qu'il y en a une.
3. **« FFA : `teams.film = 0` » DÉCRIT UN COMPTEUR QUE CE LOT A DÉFINI AUTREMENT, ET MIEUX.**
   `film` compte les joueurs dont le film DONNE l'équipe, « aucune équipe » comprise — parce que
   `-1` est une LECTURE, pas un silence. Sur un film FFA, `film` vaut donc le nombre de joueurs
   et `noTeam` la même valeur ; l'attendu du plan (`film = 0`) aurait confondu « le film dit
   qu'il n'y a pas de camps » avec « le film n'a rien dit ». Le corpus gate ne porte aucun témoin
   FFA ; la mesure vient de l'instrument, sur les deux FFA du cache : `[0 0 0 0 0 0 0 0]` sur les
   huit entités, sur les deux films.
4. **UN FILM PEUT ÊTRE LU SANS QUE PERSONNE REÇOIVE L'ÉQUIPE.** `50247b26` : 680 records ti=9
   lus, `film = 0` — sans section d'identification, la table d'index des chunks n'est pas
   injective et se fait écarter, donc le roster est vide. Les deux nombres côte à côte le disent
   exactement, et c'est ce que la couverture existe pour dire.

#### Lot 1.8 — Le kill feed prend la table du film — M, high

Sur pièces, RE-VÉRIFIÉ au commit de base `c6a3b751c` : les citations du brief avaient toutes bougé
d'un cran ou changé de sens, et c'est la règle 4 qui l'a rattrapé. `resolvePlayerIndices`
(`shots.go:122`) N'EST PAS la voie d'inférence des morts : c'est celle des **tirs** et des
**touches** (`match_weapon_shots`, `match_weapon_accuracy`), sous deux révisions distinctes de
celle des morts. La voie d'inférence que ce lot devait traiter est ailleurs et elle est plus
lourde : `killsource/bijection.go` — une matrice de votes du kill-feed résolue par l'algorithme
**hongrois** puis une montée locale par transpositions, ajustée à graine fixe.
CLOS le 2026-09-14 (branche `feat/decfilm-18`, 5 commits, `9fa12968d` → le commit de clôture).

##### Rapport 1.8.0 — la mesure AVANT de coder, sur 30 films et 8 builds

Sonde jetable (supprimée après la mesure), `Result.Roster.IndexToName` de la bijection inférée
confronté à la table de `chunk_00` (lot 1.5), index par index. Tableaux collés depuis la sortie
brute en §5. **314 accords sur 322 sièges lus** ; les HUIT écarts tiennent en deux familles, et
AUCUNE n'est une contradiction entre deux lectures fiables :

| famille | films | ce que c'est |
|---|---|---|
| le gamertag du siège est ABSENT du kill-feed | **6** — `FlukiestGolf` (`111fa685` i10), `MarshallG6443` (`e5adf7b2` i13), `manistoff` (`a521164d` i18), `Iskra 20252993` (`11de8353` i23), `probablybxllets` (`1c5c10cc` i22), `Alpha122092` (`23ffd885` i4) | le feed ne nomme que les joueurs qui TUENT ou qui MEURENT ; l'inférence n'avait aucun moyen de placer ces six-là et a mis un AUTRE joueur sur leur indice. Ce sont les MÊMES joueurs que le lot 1.6.2 avait vus publiés sans nom |
| le gamertag EST au feed | **2** — `CR951802` (`23ffd885` i2), `SerdarTsn` (`b1bcbe24` i13) | les deux tombent sur un film à **marge de bijection NULLE**, c'est-à-dire où l'inférence DIT elle-même que deux joueurs sont interchangeables et où les lignes ne sont pas publiables. **Zéro écart sur un film où l'inférence se déclare fiable.** |

**LA SECONDE MESURE FERME À MOITIÉ UNE QUESTION LAISSÉE OUVERTE AU LOT 1.5.** Les treize films du
cache à slot vacant INTERCALÉ — la seule population où « rang absolu » et « index parmi les
occupés » divergent — rendent **119 accords sur 123**, et les quatre écarts appartiennent aux deux
familles ci-dessus. C'est donc le **rang ABSOLU** que le dead-state emploie. Découverte D1 (1.8).

- [x] 1.8.1 Sur pièces : la voie d'inférence des index de joueur de killsource / killcollector
      (`resolvePlayerIndices` ou équivalent, `roster.go`, `identities.go`) ; la table de 1.5
      devient la source, l'inférence le repli (5 films).
      FAIT. `killsource/film_table.go` (neuf) lit la table par `filmdec.ReadFilmIdentity` /
      `ReadPlayerTable` ; `buildRoster` ÉPINGLE les indices qu'elle nomme, dans un ordre qui est
      le résultat : BOT_METADATA d'abord (lecture la plus ancienne et la plus éprouvée, et un bot
      n'est jamais au kill-feed), la table du film ensuite, l'inférence en dernier sur ce qui
      reste. Cinq causes de refus NOMMÉES, et le décodeur retombe alors sur l'inférence ENTIÈRE,
      comptée. **ÉCART AU LIBELLÉ, ASSUMÉ ET MESURÉ** : le repli ne sert pas que les 5 films sans
      section — il sert aussi les remplaçants, que la table du film (roster du DÉBUT, D2 (1.6))
      n'assoit pas.
      **UN SIÈGE QUE LE KILL-FEED IGNORE ENTRE AU ROSTER**, exactement comme un bot : sans cela,
      les six joueurs de la première famille du rapport 1.8.0 seraient restés inépinglables.
      DEUX EFFETS DE BORD, ET LE PREMIER A ÉTÉ TROUVÉ PAR LA MESURE, PAS PAR LA RELECTURE :
      (a) `isBotIndex` lisait « présent dans `pin` », et la table remplit désormais `pin` aussi —
      les huit joueurs d'un film entièrement lu passaient pour des BOTS et la mini-bobine tombait
      de dix lignes publiées à **DEUX**, permutation IDENTIQUE. Un test qui n'aurait regardé que
      la bijection n'aurait rien vu ; garde-rail posé (`TestUnSiegeDuFilmNEstPasUnBot`).
      (b) le problème d'affectation n'est plus CARRÉ dès que la table ajoute un nom (`111fa685` :
      25 joueurs pour 24 indices, un remplaçant partage l'indice d'un partant) ; le hongrois est
      paddé à un carré, colonnes fictives au coût pire que tout coût réel — donc résultat
      identique à celui d'avant le lot quand les deux listes ont la même longueur.
      `LineByLinePublishable` distingue une marge NULLE d'une marge SANS OBJET : un film
      entièrement lu n'a plus rien d'interchangeable. Le drapeau est POSÉ par le décodeur
      (`BijectionDetermined`) et non dérivé du roster — un `Result` à zéro doit valoir le
      comportement d'avant le lot, jamais le plus permissif (même leçon que `roster[].team` en
      pointeur au lot 1.7).
      QUATRE COMMENTAIRES FAUX CORRIGÉS dans le même geste qu'au lot 1.7.4 : « le film ne porte
      AUCUN xuid côté réplication » (`collector.go`, `roster.go`, `hits.go`) est démenti par le
      lot 1.5, « le film ne porte AUCUN camp (`Track.Team` vaut -1 partout) » (`identities.go`)
      par le lot 1.7. Ce qui reste vrai — les CHUNKS DE RÉPLICATION ne portent pas de xuid, et la
      table est celle du début du film — est écrit à la place.
- [x] 1.8.2 `KillSourceDecoderRev` montée ; `TestKillSourceDecoderRevSuitLeDecodeur` vert ;
      compteurs de provenance dans les stats de collecte.
      FAIT. `killsource-2026-09-12` → **`killsource-2026-09-14`**. Le garde-rail avait ROUGI DE
      LUI-MÊME au commit de 1.8.1 (« LE DECODEUR A CHANGE ») : preuve par mutation naturelle,
      aucune mutation artificielle nécessaire. Golden re-figé avec son entrée d'historique datée,
      qui nomme les DEUX canaux par lesquels les lignes bougent (l'identité des indices, et la
      porte de publication ligne par ligne).
      **CINQ COMPTEURS, PAS QUATRE** : les quatre du brief (`killsource_bijection_table_film` /
      `_inference` / `_silence` / `_contradiction`) plus
      `killsource_bijection_table_refusee_<cause>` — « la table a été refusée » sans dire pourquoi
      n'oriente aucun diagnostic, et la cause entre donc dans le NOM, comme
      `filmdec_unknown_build_<build>`. Ce dernier trouve ici son SECOND câbleur de production
      (D-4) : aucun corpus ne peut déclencher ce refus (0 build hors profil sur 1 351 films), donc
      seul un test peut prouver que le câblage existe, et il le fait.
      `killsource_bijection_inference` est le compteur qui informe : tant qu'il monte, des indices
      sont encore DEVINÉS, et c'est lui qui dira quand le repli pourra être retiré (D14 d).
- [!] 1.8.3 Backlog killsource du parc : **signal utilisateur, hors lot** (D6).
      NON TRAITÉ, et c'est la décision D6 du plan : une seule recuisson du parc par jalon, sur
      signal de l'utilisateur, jamais par lot. Le lot BUMPE la révision, il ne rejoue rien.
      CHIFFRE MESURÉ POUR LE PILOTE (oracle `data/backups/pre-chaine-2026-09-09/`, lecture seule,
      §5) : **1 384 matchs** portent des lignes de kill, tous sous une révision ANTÉRIEURE à
      `killsource-2026-09-14` — 792 en `killsource-2026-09-05`, 589 en `killsource-2026-07-31`,
      3 en `highlight-credit-2026-08-01`. La shared du parc n'a PAS été lue : un `server.exe` la
      tient (§2.2 — jamais de `read_only` forcé sur une DB tenue RW), et l'oracle suffit à donner
      l'ordre de grandeur. Renvoi au bloc « Clôture M1 » : le pilote joue le backlog à la clôture
      du jalon, go V9.

Preuve : corpus gate zéro perte sur les axes kills / morts / sources ; `replay-equiv` différences
localisées.

RÉSULTAT (2026-09-14) : **LA LIGNE « PREUVE » EST TENUE SUR SES DEUX MOITIÉS, ET LA PREMIÈRE L'EST
D'UNE FAÇON QU'IL FAUT DIRE PLUTÔT QUE TAIRE.**

1. **CORPUS GATE : ZÉRO PERTE SUR 13 TÉMOINS SUR 13, EXIT 0 — ET ZÉRO GAIN.** Schéma 57 → 57
   partout, ~17 min. Le lot change la SOURCE du lien `indice -> joueur` sans déplacer une seule
   valeur publiée sur ce corpus, et la raison est exactement celle que le rapport 1.8.0 a mesurée :
   là où l'inférence se trompait, les joueurs concernés n'ont AUCUNE ligne au kill-feed — ils n'ont
   ni tué ni sont morts — donc aucune ligne de kill ne les porte. Le gain du lot est structurel
   (une lecture remplace une inférence, et elle est comptée), pas métrique sur ces treize témoins.
   Un lot qui n'aurait annoncé que « 0 perte » aurait laissé croire à un gain ; il n'y en a pas ici.
2. **`replay-equiv` : LES DIFFÉRENCES SONT LOCALISÉES, ET À UNE SEULE ÉTAPE.** 10 films sur 10
   changent, et le `git diff` des références après re-figeage rend `10 +killsource` / `10
   -killsource` **et rien d'autre** : sur 520 lignes d'étapes, 510 sont identiques à l'octet,
   `artifact`, `killRefs` et `neutralDeaths` comprises. Ces deux dernières sont des PROJECTIONS du
   même `Result` : si elles ne bougent pas, aucune ligne de kill ne bouge. Ce qui fait bouger
   `killsource` est la FORME de l'objet observé — trois champs neufs — et non son contenu, ce que
   `50247b26` prouve à lui seul (table REFUSÉE, bijection identique au bit près, digest changé).
   Découverte D4 (1.8). Passe de comparaison après re-figeage : **10 identiques sur 10**.
3. **CE QUE LE LOT NE PROUVE PAS.** Le corpus gate ne porte aucun témoin où la bijection était
   AMBIGUË et devient déterminée : sur les 13 témoins, aucun film ne change de statut de
   publication ligne par ligne. Les deux films du cache qui l'auraient montré (`23ffd885`,
   `b1bcbe24`, marge nulle, écart sur un nom présent au feed) ne sont ni au corpus gate ni au
   corpus d'équivalence. Le bénéfice de `BijectionDetermined` est donc DÉMONTRÉ PAR CONSTRUCTION et
   par test unitaire, pas par le corpus.

#### Famille 1.9 — La grammaire à la place de l'heuristique, un fait par lot (D13) — S à M chacun, high

Ordre fixé par le registre du lot 0.E (table A, gain décroissant). Chaque lot : la lecture de ce
que le film écrit devient la décision ; l'heuristique devient un REPLI NOMMÉ (D14 : entrée au
registre des replis avec sa condition « film muet » et son critère de retrait, déclenché après
la lecture et jamais à sa place), compté (`coverage.<fait>.{grammaire, repli, contradiction}`) ;
test par mutation ; corpus gate zéro perte, gains nommés ; `SchemaVersion` si le contenu cuit
change ; `GrammarRev` si un lecteur change. Le premier lot de la famille pose le registre des
replis et son ratchet (1.9.0). Premier lot de conversion fixé par l'utilisateur :

- [x] 1.9.0 **Registre des replis** (`facts`, ou `replay` tant que la couche n'existe pas) :
      table nommée (nom, fait, condition typée, date de pose, cible et critère de retrait),
      ratchet `archlint/no_unregistered_fallback_test.go` (un repli hors registre = rouge ;
      convention de nommage détectable), rapport de couverture par fait ; les replis EXISTANTS
      recensés par 0.E y entrent tels quels avec leur critère, sans changer de comportement
      (équivalence zéro différence).
      **FAIT le 2026-09-14.** Paquet `film/replay/fallback` (FEUILLE : aucun import du dépôt,
      donc son déplacement vers `film/facts/fallback` au pas 5 de M2 est un déplacement pur) —
      `Repli{Nom, Fait, Mecanisme, Condition, Ordre, Sites[]{Fichier, Ancre}, DatePose,
      CibleRetrait, CritereRetrait, CompteurBranche, CibleComptage}`, **95 entrées** dont
      **59 des 60 lignes de la table (E)** de l'audit 0.E (la soixantième, `build.go:580`
      `Team: -1`, est CONVERTIE par le lot 1.7 — recensement collé en §5) plus les replis nommés
      par 0.D à 1.8 et ceux que la famille 1.9 désigne. `Compteur` PAR CUISSON (nil-safe, jamais
      de variable de paquet : critères S1 et S2), publié en `coverage.fallbacks[]` — **schéma
      57 -> 58**, chaîne complète. Ratchet à DEUX directions (code -> registre par convention de
      nommage à frontière camelCase ; registre -> code par relecture de l'ancre, ce qui rend
      D14 (d) mécanique), **mutation `repliBidon` jouée et restaurée**. Sortie humaine :
      `go test …/fallback/ -run RapportDuRegistre -v`, et les **huit goldens d'assemblage**
      portent désormais un bloc « REPLIS DECLENCHES » — la table par build, versionnée.
      **10 compteurs câblés sur 95** : la limite de 5 paramètres du dépôt borne le câblage, et
      les 85 autres le seront par leur lot de conversion (`CibleComptage` le dit entrée par
      entrée) — D3 (1.9.0) en §4.

- [x] 1.9.1 **Origine d'une pose d'équipement.**
      **FAIT le 2026-09-15.** L'origine d'une pose (`deployed` / `dropped` / `unknown`) se
      décidait par deux règles de SECOURS — le manifeste (`kind = "deployed"` -> `deployed` sans
      mesure, item H.2 des finitions) puis une FENÊTRE TEMPORELLE de 200 ms entre la création de
      l'objet et la fin de la vie de son poseur (`equipmentOrigin`, depuis F.1 `c45c411eb`).
      **Elle se LIT désormais** (`replay/equipment_origin.go`, `origineDeLaPose`), par trois
      signaux que le film écrit :
      (1) l'événement de liste type 103 `EquipmentSpawnedObject` — « une PIÈCE a été engendrée » —
      dont la deuxième référence DÉSIGNE la vie de l'objet créé (lecteur de production NEUF,
      `filmdec/equipment_spawn_events.go`, lecture de TÊTE de liste : 927 des 931 occurrences du
      parc y sont, rapport F.0 §1.1) ;
      (2) la MORT ÉCRITE du poseur — une vie du siège que le fil des morts ferme, le registre
      d'identité des lots 1.6 / 1.8 étant le seul producteur du lien ;
      (3) sa PRISE ÉCRITE (`equipmentChanges.taken`) — le porteur ramasse autre chose, donc il
      lâche ce qu'il tenait (lâcher volontaire à mi-vie, rapport E0 question 5).

      **DÉCISION UTILISATEUR DU 2026-09-15 — « LÂCHÉ », ET LE MOT DU JEU.** Question de
      l'utilisateur : « comment c'est appelé dans le code du jeu ? ». Réponse sur pièces
      (`filmdec/testdata/ecs_table.tsv`, archétype 37) : **le jeu n'écrit AUCUN événement
      « equipment drop »** — il n'en existe pas (seul `weapon_drop`, type 46, existe, pour les
      armes). Ce qu'il écrit sur un objet d'équipement, ce sont des **COMPOSANTS D'ÉTAT** :
      `i20 equipment-deployed`, `i21 equipment-activated`, `i18 item-at-rest`,
      `i10 object-parent-state` (le porteur), `i23 equipment-creator` ; plus UN événement
      d'apparition, `EquipmentSpawnedObject` (103). Les ramassages sont `biped_pickup` (9) et
      `biped_pickup_item_request` (57), les activations `biped_equipment_activation` (30) et
      `activate_spartan_ability` (93).
      **NOTRE VOCABULAIRE SE CALQUE SUR CELUI-LÀ** : « déployé » est le mot du jeu (`deployed`) ;
      **« lâché » = l'objet quitte le porteur SANS être déployé** — à la mort ou à mi-vie par
      échange, les deux sont « lâché », et la cause va dans `byCause`, jamais dans l'étiquette.
      (L'utilisateur a retiré la notion de « volontaire » : un échange l'est aussi.)
      Un appareil porté n'a aujourd'hui **aucun déploiement écrit** — le 103 en désigne 0 sur 91
      et 0 sur 4 853 (rapport F.0) — il sort donc toujours `dropped`. Le vocabulaire est tranché :
      `deployed` est **réservé à une pose DÉSIGNÉE par un record 103** (les panneaux de mur, et
      eux seuls) ; un appareil PORTÉ qui tombe sort **`dropped`**, que la cause soit la mort
      écrite de son porteur ou son `taken` écrit — l'étiquette ne distingue pas les deux lâchers,
      **la CAUSE se publie dans `coverage.placements.byCause`** ; une pose dont le film ne dit
      RIEN sort **`unknown`**. Conséquence directe : **la fenêtre de 200 ms n'a plus AUCUNE
      décision à prendre et elle est RETIRÉE du décodeur** — elle ne survit que comme TÉMOIN de
      mesure dans un fichier de recherche (`f1OrigineParFenetre`). Ses DEUX entrées sortent du
      registre des replis le même jour (D14 d) : `repli_origine_pose_fenetre_temporelle` et
      `repli_origine_pose_vie_la_plus_proche`. **Le ratchet des sept `devant_la_lecture` reste à
      7, et ce n'est pas un oubli : vérifié sur pièces, aucune des deux n'était de cet ordre**
      (`non_resolu / apres_lecture` et `film_muet / sans_lecture`). Registre : 95 -> 94 entrées,
      10 compteurs câblés.

      **LES TOLÉRANCES SONT MESURÉES, PAS CHOISIES** (13 films — 8 builds, échantillon court,
      `0797ce72`, `4f77afc1` —, 4 583 poses ; instrument versionné
      `e191_origine_{mesure,rapport}_research_test.go` ; tableaux collés en §5) : mort **200 ms**
      (max 171,7 ms d'un côté, min 205,3 ms de l'autre — un intervalle VIDE de 33,6 ms) ; prise
      **50 ms** (103 des 108 prises retenues sont à MOINS D'UNE ms, contre 1 pose `dropped`) ;
      désignation 103 dans **[0, +200] ms** après la création (les 115 poses de panneau désignées
      le sont entre +32,2 et +70,2 ms, toutes POSITIVES ; aucun autre objet désigné, 0 sur 4 459).
      La clé (slot, génération) NE SUFFIT PAS — la génération fait 2 bits et reboucle : par clé
      seule, 3 événements « désignaient » 83 poses de `d9781168`.

      **CE QUE LE LOT CHANGE, MESURÉ POSE PAR POSE — 757 sur 4 583 (16,5 %)** :
      `deployed -> dropped` par un `taken` écrit **108** (le lâcher à mi-vie cessait de dessiner
      un geste qui n'a pas eu lieu) ; `deployed -> unknown` **188** et `dropped -> unknown` **461**
      (les 649 poses que la fenêtre étiquetait par corrélation). **3 826 poses ne bougent pas** :
      3 255 `death_written`, 115 `spawn_event`, 9 `manifest_piece`, 1 `both`, 446 `no_owner`.
      Totaux publiés : `deployed` **420 -> 124**, `dropped` **3 717 -> 3 364**, `unknown`
      **446 -> 1 095**. `byCause` somme exactement à 4 583.

      **`unknown` NE SUBSISTE QUE SUR DEUX SILENCES, comptés et nommés** : `none` **649** (un
      poseur EST mesuré, le film ne dit rien de cette pose) et `no_owner` **446** (aucun bipède
      contemporain à moins de 3 m). Les confondre coûterait le diagnostic ; ils sont publiés
      séparément.

      **LE SEUL REPLI QUI RESTE** : `repli_piece_engendree_sans_evenement` (NEUF) — une pièce
      engendrée au manifeste qu'aucun 103 ne désigne sort `deployed`, parce que le manifeste du
      titre est une donnée ÉCRITE et qu'un panneau n'existe que déployé. **9 poses sur 124,
      toutes sur les deux films de build les plus anciens** (`a521164d` HI_1_4_1 lit 0 événement
      sur 4 956 listes, `50247b26` v31 en lit 2) : une limite de BUILD, pas une incertitude.
      C'est le SEUL endroit où `deployed` se publie sans un 103 — dit ici pour qu'il puisse être
      retiré en une ligne si l'utilisateur le veut.

      **GARDE-RAIL P2 D-H2 POSÉ** : `TestPanneauxDuMurMatchManifest` recolle `usageWallPanelIDs`
      aux identifiants `kind = "deployed"` du manifeste, dans les deux sens et sur la famille —
      seul le garde WEB le faisait. Lecteur de manifeste écrit UNE fois
      (`manifesteObjetsEquipement`), partagé avec le garde des familles.

      **MUTATIONS JOUÉES** (`equipment_origin_lecture_test.go`, chaque lecture deux fois) :
      103 retiré -> la pose passe au repli du manifeste, compté ; mort retirée -> `unknown` par
      `none` ; prise retirée -> `unknown` par `none` ; fenêtre faussée (pose à mi-vie PUIS à la
      fin) -> RIEN ne change pour un panneau désigné ; contradiction -> comptée sous `both` et
      aucun repli déclenché ; la clé seule ne désigne pas (±1 image, ±10 s, génération voisine).

      **TESTS RETIRÉS AVEC LA RÈGLE QU'ILS VERROUILLAIENT** :
      `TestEquipmentOriginSepareLacherDuDeploiement`, `TestEquipmentOriginSansVieEstInconnue`,
      `TestEquipmentOriginChoisitLaVieQuiContientLInstant` —
      `.ai/baselines/tests_pre_migration.jsonl` mis à jour dans le MÊME commit (3 lignes).

      **SCHÉMA 58 -> 59** (chaîne complète, §5) : `coverage.placements.byCause` et
      `coverage.placements.{spawnEvents, spawnLists}`. **`GrammarRev` grammar-2026-09-14.6 ->
      grammar-2026-09-15.1** (un lecteur d'octets neuf). `equipment_placements.go` (652 L) SCINDÉ
      en trois. `4f77afc1` ajouté au manifeste du corpus gate, famille
      `equipement_origine_utilisateur`. `REFERENCE_CANAUX_EQUIPEMENT` corrigée dans le même
      commit (§1 deux lignes, canal des événements, §6 deux entrées).

      **LES POSES REQUALIFIÉES PAR F.1, REJUGÉES UNE À UNE** : sur les 13 films de ce corpus,
      **18** poses portent un verdict différent entre la règle d'AVANT F.1 (fenêtre + distance)
      et ce qui était publié — **18 accords, 0 désaccord** avec la grammaire. 17 sont tranchées
      par une MORT ÉCRITE (F.1 confirmée pose par pose), 1 par un ÉVÉNEMENT 103 (`111fa685`,
      panneau `0x686b40c9` : H.2 confirmée par le film). Le brief en annonçait 22 : c'est le
      compte de F.1 sur SON corpus (21 films, 5 363 poses), pas sur celui-ci — tableau collé en §5.

      **CE QUI N'EST PAS FAIT, ET POURQUOI** : la piste `ref0` (table D du registre 0.E) reste
      NON INSTRUITE — hors lot ; le lecteur la publie brute sans l'interpréter.
- [ ] 1.9.1 bis **La grammaire de l'ÉQUIPEMENT, entière : l'archétype 37 fermé à 100 %, famille
      par famille.** Directive utilisateur du 2026-09-15 : « on ne se limite pas au mur ; on a
      tout plein d'équipements dont il faut gérer la grammaire parfaitement ; c'est le point de
      départ pour tout le décodeur, si on ne la connaît pas bien rien n'est fiable ensuite ».
      Sur pièces le 15/09 : l'archétype 37 (« equipment / item, objet du monde ») compte 31
      composants (`testdata/ecs_table.tsv`) ; le décodeur n'en lit que 7 par nom (`traverse.go` :
      i18 `item-at-rest`, i19 `item-ignore-player`, i22 `equipment-control-signal`, i25
      `equipment-being-hacked`, i28 `equipment-tracked-object-handles-stack`, i29
      `equipment-command-tick`, i30 `equipment-has-infinite-uses`) et sa FERMETURE d'image-clé est
      de 0 ou 1 record sur 220 à 762 par bobine (golden 1.4, `ti=37`) ; les composants qui portent
      l'ÉTAT écrit par le jeu ne sont pas lus : i10 `object-parent-state` (le porteur), i20
      `equipment-deployed`, i21 `equipment-activated`, i23 `equipment-creator`, i24
      `equipment-energy`, i26 `equipment-energy-delay-ticks-left`, i27
      `equipment-charges-remaining`. (a) Chaque composant de ti=37 relu chez l'écrivain (Ghidra,
      fonction par composant, comme 1.3 pour les états par défaut), lecteur porté, fermeture
      ti=37 100 % sur les 7 bobines (ratchet 0.A.3 qui MONTE, test « n2 constant »), `GrammarRev`
      montée ; (b) le vocabulaire du jeu devient le nôtre : par objet d'équipement, créateur,
      porteur, déployé, activé, au repos, énergie, charges restantes, usages infinis — publiés
      comme FAITS par famille (mur, capteur, traqueur, écran, champ, balise, camouflage,
      surbouclier, grappin, translocateur, propulseur, répulseur, bonus), avec un tableau
      « famille × ce que le jeu écrit » mesuré sur les 8 builds et les 13 témoins, et la
      référence `REFERENCE_CANAUX_EQUIPEMENT` mise à jour depuis cette mesure (ses négatifs
      mesurés se re-jugent contre les composants, pas contre les seuls événements) ; (c) les
      canaux publiés dérivés de l'équipement (`equipmentPlacements`, `equipmentChanges`,
      `equipmentEpisodes`, `abilityCharges`, `grappleLines`, `translocations`,
      `abilityImpulses`, porteurs de crâne et de balle) se re-dérivent de ces états, les anciennes
      lectures devenant des CONTRÔLES comptés puis des replis au registre (1.9.0) ; corpus gate
      zéro perte, gains nommés ; `SchemaVersion` si le contenu cuit change. Ordre : juste après
      1.9.1, AVANT 1.9.2 ; le même traitement pour les autres archétypes qui alimentent une
      publication (ti=9 joueur, ti=35 bipède, ti=40 véhicule, ti=42/43) est le lot 3.6, à
      remonter dans M1 si l'utilisateur le demande. L.
      **RE-CADRÉ PAR LA MESURE DU PAS 1 (2026-09-15, `ec74685ed`)** : les 31 composants de ti=37
      sont TOUS dispatchés (le « 7 lus » était un artefact de grep ; 0 désynchronisation sur
      3 331 records) ; ce qui empêche la fermeture est le PRÉFIXE OBJET partagé par les cinq
      archétypes du monde (ti=37/38/41/42/43, fermés à 0,85 %) : i15 `object-low-frequency`, i6
      `object-region-state`, i14 `object-dissolver`, i9 `object-multiplayer-properties`, i17
      `object-frame-configuration`, i7 `object-damage-sections` — et 12 composants de la table ECS
      sans adresse d'écrivain. Le périmètre devient : **le préfixe objet relu chez l'écrivain, la
      fermeture mesurée sur les CINQ archétypes ensemble, puis les états d'équipement publiés**.
      BLOQUÉ tant que l'instance Ghidra n'est pas ouverte (D13 : pas de grammaire posée ailleurs
      que chez l'écrivain) ; les bobines ne portent aucun paquet delta, la mesure des états se fait
      sur films entiers une fois la marche fermée. Les lots de conversion sans Ghidra (1.9.2,
      1.9.3, 1.9.4, 1.9.6, 1.9.7, 1.9.8, 1.9.10, 1.9.11, 1.9.13, 1.9.14) avancent en attendant.

      **PAS 1 FAIT le 2026-09-15 (branche `feat/decfilm-191b`) — ET IL RETOURNE LE LOT.**
      La règle du chantier (mesurer avant de coder) a rendu un verdict que le brief n'anticipait
      pas : **le défaut qui empêche l'archétype 37 de fermer n'est PAS dans l'équipement.**
      Trois instruments versionnés, lecture seule, sans garde d'environnement pour deux d'entre
      eux (les 7 bobines par build sont versionnées) : `filmdec/e191b_carte_ti37_research_test.go`
      (`TestE191bCarteTI37`), `filmdec/e191b_carte_ti37_carte_test.go`
      (`TestE191bFermetureAvecCarte`), `filmdec/e191b_carte_ti37_masque_test.go`
      (`TestE191bMasqueTI37`, sous `CHUNK00_FILMS`). Tableaux collés en §5.

      **(1) LES 31 COMPOSANTS SONT TOUS CONSOMMÉS, ET LA CITATION DU BRIEF EST À CORRIGER.**
      Vérifié sur pièces par grep : les 31 composants de ti=37 ont TOUS un `case` dans le
      dispatch de `traverse.go`. Huit d'entre eux y entrent par une CONSTANTE et non par le
      littéral — `compObjectBodyVitality` (i4, `registry.go:58`),
      `compObjectMultiplayerProperties` (i9, `ground_weapon_ammo.go:82`), `compEquipmentDeployed`
      (i20), `compEquipmentActivated` (i21), `compEquipmentCreator` (i23), `compEquipmentEnergy`
      (i24), `compEquipmentEnergyDelay` (i26), `compEquipmentCharges` (i27)
      (`equipment_state.go:35-40`) — et c'est **exactement pourquoi un grep du nom de composant
      dans `traverse.go` n'en trouvait que 7** : il cherchait une chaîne que ces huit `case`
      n'écrivent pas. La mesure le confirme indépendamment : **0 désynchronisation sur 3 331
      records bornés** — la colonne « bloquant » du golden 0.A.3 est vide pour ti=37 parce
      qu'aucun composant n'est sans lecteur. Le « 7 lus par nom » du brief est donc un artefact
      de grep, pas un état du décodeur.

      **(2) LA FERMETURE ÉCHOUE SUR TOUTE LA FAMILLE « OBJET DU MONDE », PAS SUR L'ÉQUIPEMENT.**
      Contrôle [3], 7 bobines, tous archétypes : les archétypes qui portent
      `object-position-component` ferment **184 / 21 698 (0,85 %)** ; ceux qui ne le portent pas
      **12 654 / 28 573 (44,29 %)** — 52 fois plus. Détail de la première population :
      ti=37 **3/3 331**, ti=38 **180/12 064**, ti=41 **0/110**, ti=42 **1/2 087**, ti=43 **0/4 106**.
      ti=42 (`ground-weapon`) est l'archétype dont le dépôt dit la grammaire « réputée complète » :
      il ferme 1 record sur 2 087. **Fermer ti=37 à 100 % sans toucher au préfixe objet est donc
      impossible**, et c'est pourquoi l'item (a) ne peut pas être exécuté tel qu'il est écrit.

      **(3) LES LARGEURS D'AXE DE LA CARTE NE SONT PAS LA CAUSE — MESURÉ, PAS SUPPOSÉ.**
      Premier suspect nommé par le code : `object-position-component` lit ses trois axes aux
      largeurs de la CARTE (`WorldObjectPrecision`, `traverse.go:154`), que `replay.BuildFromFilm`
      installe et que `KeyframeClosure` n'installe PAS. Contrôle joué (`TestE191bFermetureAvecCarte`,
      catalogue versionné `map_quant_bounds.json`, la même entrée que la production) :
      **3/3 331 au défaut Cliffhanger, 3/3 331 aux largeurs de la carte jouée.** Deux bobines
      échangent leur record fermé (`11de8353` 1 -> 0, `e5adf7b2` 0 -> 1), le total ne bouge pas.
      i0 est HORS DE CAUSE pour la fermeture (il reste une dette de mesure : cf. D2 (1.9.1 bis)).

      **(4) LE COMPOSANT QUI FAIT FRANCHIR LA FRONTIÈRE EST NOMMÉ, ET IL EST STABLE SUR LES
      7 BOBINES.** Mesure : pour chaque record, le composant qui précède immédiatement le premier
      composant dont le bit de départ dépasse déjà la frontière du record suivant. C'est une
      PREUVE bornante (un composant qui commence après la fin du record n'appartient pas à ce
      record), pas une corrélation. Cumul des 3 331 records, largeur moyenne consommée entre
      parenthèses : **i15 `object-low-frequency` 655 (w~570)** · **i6 `object-region-state` 341
      (w~664)** · **i14 `object-dissolver` 305 (w~113)** · **i9 `object-multiplayer-properties`
      296 (w~1 508 300)** · **i17 `object-frame-configuration` 266 (w~87)** · **i7
      `object-damage-sections` 188 (w~317)** · i10 74 · i28 71. **Les six premiers sont TOUS du
      préfixe objet ; aucun composant d'équipement (i18-i30) n'est au-dessus de 71.** i9 consomme
      en moyenne 1,5 MILLION de bits : un TLV dont la longueur se lit dans le flux, sans borne.
      i14 consomme 113 bits dans **3 306 records sur 3 331** alors que la table ECS donne 4 bits
      au cas nominal (`v == 13`) — sa garde est lue à l'envers, ou sa grammaire est fausse.

      **(5) LA DISTRIBUTION DES RÉSIDUS INTERDIT L'EXPLICATION SIMPLE.** `EndBit - Want` prend
      **1 109 valeurs distinctes sur 3 331 records** (la plus fréquente, `+50`, 37 fois) : ce
      n'est pas une largeur fixe fausse de N bits, c'est une marche qui dérive tôt et lit ensuite
      des longueurs variables sur des bits de bruit.

      **(6) LES 31 COMPOSANTS SONT LES MÊMES SUR LES 8 BUILDS, À UN PRÈS.** Registre relu bobine
      par bobine : 31 composants dans le même ordre partout, SAUF `a521164d` (HI_1_4_1) qui en
      porte **30** — `i30 equipment-has-infinite-uses` n'existe pas sur ce build. C'est un fait de
      profil par build (M3), pas une variabilité à absorber par une heuristique.

      **(7) LA PRÉSENCE AU MASQUE NE SE MESURE PAS ENCORE.** Les bobines par build ne portent
      **AUCUN paquet delta** (mesuré : 0 sur les 7) — elles sont `chunk_00` + un chunk d'image-clé
      + le pied (V7). Sur 7 films ENTIERS du cache (12 chunks chacun), la marche générique rend
      92 records NEW et 70 records DELTA de ti=37, et la présence au masque est **plate entre
      10,9 % et 34,3 % sur les 31 index** — la signature de bits aléatoires, pas d'un masque. La
      question « où et quand le jeu écrit i10/i11/i18/i20/i21/i23 » NE PEUT PAS être répondue tant
      que la marche ti=37 ne ferme pas : toute réponse tirée de ces comptes serait du bruit.

      **STATUT DES ITEMS DU BLOC.**
      (a) `[!]` **NON TRAITÉ, et le blocage est double** : (i) l'outil Ghidra n'est pas disponible
      dans la session (`list_instances` : aucune instance ; `connect_instance` : connexion refusée
      sur 127.0.0.1:8089) — or D3 et D13 interdisent de poser une grammaire autrement que chez
      l'écrivain, et le brief le dit explicitement ; (ii) la mesure (2) montre que la cible
      « fermeture ti=37 à 100 % » passe par le préfixe objet partagé (ti=38/41/42/43), c'est-à-dire
      par un périmètre que cet item n'énonce pas. Les deux se lèvent ensemble au prochain passage.
      (b) `[!]` NON TRAITÉ — publier `créateur / porteur / déployé / activé / au repos / énergie /
      charges` comme FAITS exige que la marche ti=37 ferme ; mesure (7) : elle ne ferme pas, et
      les valeurs lues aujourd'hui sont du bruit. Publier maintenant serait exactement le
      « ça passe » que D13 interdit.
      (c) `[!]` NON TRAITÉ — dépend de (b).
      (d) `[~]` COUVERT PAR LA NATURE DU PAS : aucun octet cuit ne change (les trois fichiers sont
      des `_test.go`), donc ni `SchemaVersion` (59 -> 59) ni `GrammarRev`
      (`grammar-2026-09-15.1`, inchangée) ne montent, et rien ne change à l'affichage web.
      Gates joués et consignés en §5.

      **DÉCOUPAGE : pas 1 `[x]` · pas 2 à 5 `[!]` (non entamés, arrêt propre en fin de pas 1,
      conformément au brief).** Ce que le prochain passage doit faire, dans cet ordre, est
      désormais MESURÉ et non plus supposé : relire chez l'écrivain **i15, i6, i14, i9, i17, i7**
      — les six composants du préfixe « objet du monde » qui font franchir la frontière — puis
      re-mesurer la fermeture des CINQ archétypes objet ensemble (ti=37, 38, 41, 42, 43), et
      seulement ensuite publier les états d'équipement. Les quatre premiers n'ont d'ailleurs
      aucune adresse d'écrivain dans `testdata/ecs_table.tsv` (colonne `deser_addr` vide pour i6,
      i7, i8, i12, i13, i16, i17 ; `grammar` = « boucle de regions », « boucle de sections »,
      « inconnue ») : c'est là que la relecture manque, et la mesure vient de le chiffrer.

      **PAS 2 FAIT le 2026-09-15 (branche `feat/decfilm-191c`, commits `991a11ad0` et suivants) —
      LES SIX SUSPECTS SONT BLANCHIS, ET LE CADRAGE DU PAS 1 EST À CORRIGER.**
      Ghidra ouvert en HTTP direct (`HaloInfinite.exe`, base `140000000`, 311 103 fonctions,
      LECTURE SEULE : aucun point d'entrée d'écriture appelé). La chaîne de résolution est la
      même qu'au lot 1.3 et elle est désormais MÉCANIQUE :
      `chaîne du nom -> xref (thunk de nom) -> xref (slot de vtable +0x08) -> vtable_base+0x30`.

      **(1) LES QUINZE COMPOSANTS DU PRÉFIXE OBJET SONT RELUS, ET AUCUNE LARGEUR NE CHANGE.**
      Les six que le pas 1 nommait, PLUS les douze sans adresse d'écrivain (recouvrement de
      trois) — fonction citée, grammaire relue sur le désassemblage ou la décompilation :

      | i | composant | fonction (vtable) | verdict |
      |---|---|---|---|
      | i2 | `object-forward-and-up` | FUN_14076e278 -> FUN_140c5f938 -> FUN_140c5fa84 (143d06358) | R(1) ; si 0 : R(19) ; puis R(8) **inconditionnel** — conforme |
      | i4 | `object-body-vitality` | FUN_140fb8978 (143d0b810) | R(8) déquant (largeur littérale) + 3 x R(1) — conforme |
      | i5 | `object-shield-vitality` | FUN_140d50cbc (143d0b7c0) | R(8) + R(1)[si 1 : 2 x b1ac0] + R(16) + 4 x R(1) — conforme |
      | i6 | `object-region-state` | FUN_140e1bfa0 (143d0b8b0) | R(1) + R(6) n + n x R(3) [si présent : n x R(10)] — conforme ; max théorique **826** = le max observé |
      | i7 | `object-damage-sections` | FUN_142f03c80 (143d0b860) | R(6) n + n x (R(1)[si 1 : R(7)+R(16)]) — conforme |
      | i8 | `object-constraint` | FUN_142f039cc (143d0b9f0) | R(5) n [si n != 0 : 2 x R(n)] — conforme |
      | i9 | `object-multiplayer-properties` | FUN_140f53308 -> FUN_1407d4c94 (143d0b9a0) | l'enveloppeur est identifié ; corps conforme |
      | i10 | `object-parent-state` | FUN_140c1e4d0 (143d0ba90) | conforme ; **param_3 = 1** sur la branche attachée (sonde lue), 0 sur l'autre |
      | i12 | `object-scale` | FUN_1407dc6e4 (143d0b950) | conforme |
      | i13 | `object-maximum-vitalities` | FUN_1407ee054 -> FUN_1407eef08 (143d0b900) | enveloppeur confirmé |
      | i14 | `object-dissolver` | FUN_140dd9f9c (143d0bf78) | conforme ; **la table portait l'adresse d'i15**, corrigée |
      | i15 | `object-low-frequency` | FUN_1407ef088 (143d0bf28) | toutes largeurs relues instruction par instruction — conforme |
      | i16 | `object-physics-flags` | FUN_1407ee070 (143d0bfc8) | 5 x R(1) — conforme |
      | i17 | `object-frame-configuration` | FUN_1407f0534 -> FUN_1407f0550 (143d0be80) | conforme ; le compte est `R(6) + 1` et les itérations sont **3**, bornées par le tableau |
      | i21 | `equipment-activated` | FUN_140c1dc80 (143d05ef0) | R(1) inversée ; si 0 : R(3) ; si 1 : `FUN_1408f0ac4(param_3 = 4)` — conforme |

      **FERMETURE DES CINQ ARCHÉTYPES OBJET, AVANT ET APRÈS : IDENTIQUE — 184 / 21 698 (0,85 %).**
      C'est le résultat ATTENDU d'une relecture qui ne corrige rien, et ce n'est pas un
      non-événement : il ÉLIMINE les six suspects que le pas 1 avait nommés.

      **(2) LA DICHOTOMIE DU PAS 1 EST UN CONFONDANT — mesuré, pas supposé.** Les 44,29 %
      « sans `object-position-component` » viennent de HUIT archétypes seulement (ti=6, 14, 17,
      18, 22, 29, 4, 15) ; tous les autres ferment ZÉRO sans porter le préfixe objet : ti=5
      **0/3 616**, ti=10 **0/2 261**, ti=9 **0/1 717**, ti=47 **0/1 716**, ti=13 8/1 491, ti=35
      **0/1 364**, ti=12 0/879, ti=40 0/777. La complexité n'explique pas davantage : ti=6 porte
      58 composants et ferme 90,1 %, ti=47 en porte 3 et ferme 0. **AUCUN des 30 composants de
      ti=37 n'apparaît dans un archétype qui ferme** : la famille « objet » n'a jamais été
      prouvée par une fermeture, nulle part.

      **(3) LES PETITS ARCHÉTYPES QUI NE FERMENT PAS NE FERMENT PAS PAR LARGEUR FAUSSE, MAIS PAR
      DÉSYNCHRONISATION.** ti=25 (1 composant), ti=45 (2), ti=3 (2), ti=20 (3), ti=47 (3) :
      **100 % de leurs records** butent sur un composant non porté. ti=43 est dans ce cas aussi —
      `i19 device-position-animation-name-component`, dont la grammaire est RELUE ici
      (`FUN_1410156e4` : R(32) identifiant + R(10) déquantifié dans [0, 10], 42 bits) mais **NON
      PORTÉE** : les 22 composants `device-*` sont le lot 3.6, et porter i19 seul ne fermerait
      aucun record.

      **(4) DEUX EXPLICATIONS CONCURRENTES SONT ÉLIMINÉES, MESURÉES.** Les ANCRES FORTUITES : le
      filtre `n1` modal (la taille de tampon que `FUN_142e2bfd0` lit à position fixe) en retire
      **138 sur 50 384 (0,27 %)** et la fermeture des cinq ne bouge pas (184/21 594). La LARGEUR
      D'ÉTAT PAR DÉFAUT de ti=37 : le balayage 0..512 bits ne trouve **aucune** largeur
      substituée qui ferme plus de 32 records sur 3 331 (la largeur portée en ferme 3).

      **(5) CE QUE LA RELECTURE A TROUVÉ ET QUI RESTE OUVERT — la seule piste neuve.**
      `FUN_1406d3140` (l'entier à largeur variable, lu par i10, i21, i26 et les boucles de slots)
      ne lit PAS toujours 0x1FFF : sa plage vient d'une TABLE INDEXÉE PAR `param_3`
      (`DAT_1451f98d0`) dès que la garde `DAT_144706104` est levée — et cette garde **vaut 1 dans
      l'image statique** (lu le 2026-09-15). La table y est NULLE : elle est remplie AU
      CHARGEMENT DE LA CARTE, exactement comme les largeurs d'axe d'i0. Les appelants passent des
      `param_3` DIFFÉRENTS — 0, 1 (i10 attaché), 4 (i21) — là où le décodeur lit UNE seule
      largeur (13 + 2 bits). Non corrigé : la table est vide dans le binaire, D13 interdit de
      deviner. Consigné en D3 (1.9.1 bis, pas 2).

      **STATUT DES SOUS-POINTS DU PAS 2.** grammaire des six `[x]` · les douze sans adresse
      `[x]` (adresses posées, quinze grammaires relues) · colonnes `deser_addr` / `grammar` /
      `bits_typ` de `ecs_table.tsv` `[x]` (treize adresses neuves + la correction d'i14) ·
      tests de largeur `[x]` (`components_prefixe_objet_largeurs_test.go`, mutation vérifiée) ·
      fermeture des cinq mesurée après `[x]` (inchangée, tableau ci-dessus) · ratchet 0.A.3
      `[~]` (il ne peut pas MONTER : aucune largeur ne change ; il n'a pas bougé) · `GrammarRev`
      `[x]` (`grammar-2026-09-15.1` -> `.2`) · D2 (1.9.1 bis), l'installation du découpage du
      catalogue dans `KeyframeClosure`, `[!]` — le plan le conditionne à « la montée de fermeture
      qu'il rend visible », et il n'y a pas de montée.

      **DÉCOUPAGE : pas 1 `[x]` · pas 2 `[x]` · pas 3 à 5 `[!]` — NON ENTAMÉS, ET LE BLOCAGE EST
      NOMMÉ.** Le pas 3 (les états d'équipement publiés comme FAITS) exige que la marche ti=37
      ferme : elle ferme 3 records sur 3 331, et les quinze composants du préfixe viennent
      d'être blanchis chez l'écrivain. Publier maintenant serait le « ça passe » que D13
      interdit. Les pas 4 et 5 dépendent du pas 3. Les GATES DE DÉCODAGE (`replay-equiv`,
      `replay-corpus-gate`) ne sont PAS joués : la machine est partagée avec le lot 1.9.2 et le
      pilote n'a pas donné « voie libre » — le pas ne touche de toute façon aucun octet cuit
      (`SchemaVersion` 59 -> 59, seuls des commentaires, des `_test.go` et une table `testdata`
      changent côté production).

      **PAS 2 BIS FAIT le 2026-09-15 — LA PISTE D3 EST RESOLUE CHEZ L'ECRIVAIN, ET ELLE N'EST
      PAS UNE DONNEE DE PROFIL : C'EST UNE TABLE DE CONSTANTES DU BINAIRE.**
      D3 (1.9.1 bis, pas 2) supposait que la table de plages de `FUN_1406d3140` etait remplie au
      chargement de la carte, comme les largeurs d'axe d'i0. **C'EST REFUTE PAR L'ECRIVAIN** :
      `FUN_140d10bb0` (appelee par `FUN_140d10a78`) la remplit avec des LITTERAUX et pose
      `DAT_144706104 = 1`, et les deux autres ecrivains (`FUN_1408f1618` @1423503d3,
      `FUN_142f2f0cc` @142f2f2b1) ne touchent que les entrees 0, 1, 7, 8 et `DAT_144706100`.

      **(1) LA TABLE, RELUE LIGNE A LIGNE** — plage, puis `W = FUN_1406d310c(plage)`
      (rang du bit de poids fort, plus un si des bits plus bas sont mis) :

      | `param_3` | base | plage | W | le depot lisait |
      |---|---|---|---|---|
      | 0, 1 | 0x200 | `DAT_144706100` − 0x200 = 0x1DFF | **13** | 13 (juste) |
      | 2 | 0x200 | 0x100 | **8** | 13 |
      | 3 | 0x300 | 0x100 | **8** | 13 |
      | 4 | 0x200 | 0x200 | **9** | 13 |
      | 5 | 0x400 | 0x100 | **8** | 13 |
      | 6 | 0 | 0x200 | **9** | 13 |
      | 7, 8 | 0 | `DAT_144706100` = 0x1FFF | **13** | 13 (juste) |

      **ET LA CATEGORIE 1 BASCULE** : `si (param_3 == 1 && R(1) != 0)` le jeu prend
      `DAT_1451f98f0` / `DAT_1451f98f4`, c'est-a-dire **l'entree 4** (verifie :
      1451f98d0 + 4×8 = 1451f98f0) — 9 bits, pas 13. Le bit de sonde n'est lu QUE pour
      `param_3 == 1` (la condition le teste en premier). La base (0x200 / 0x300 / 0x400) est un
      DECALAGE ajoute a la valeur : elle ne change aucun bit lu.

      **(2) LES APPELANTS, `param_3` RELEVE SUR LE DESASSEMBLAGE** (`MOV R8D, imm` avant le
      `CALL` ; `FUN_1408f0ac4` transmet son `param_3` tel quel — `MOV R14D,R8D` puis
      `MOV R8D,R14D` avant `CALL 0x1406d3140`). Les sites qui changent quelque chose :

      | site | fonction du jeu (adresse du CALL) | `param_3` | lecture juste | le depot lisait | ecart |
      |---|---|---|---|---|---|
      | i10 `object-parent-state`, branche attachee | `FUN_140c1e4d0` @140c1e51d | **1** | sonde ; si 1 → 9+2, si 0 → 13+2 | sonde + 13+2 | **−4 quand la sonde rend 1** |
      | i21 `equipment-activated` | `FUN_140c1dc80` @140c1dcbb | **4** | porte ; si 1 → 9+2 | porte + 13+2 | **−4** |
      | i22 `equipment-control-signal` | `FUN_14101cd94` @14101cdc3 | **4** | R(4) + porte ; si 1 → 9+2 | R(4) + porte + sonde + 13+2 | **−5** |
      | i28 `equipment-tracked-object-handles-stack` | `FUN_140f72dec` @140f72e41 | **0** | porte ; si 1 → 13+2 | porte + sonde + 13+2 | **−1** |

      Sites RELUS et INCHANGES (`param_3 = 0` ou `1`, donc W = 13 comme avant) : i10 branche
      libre @1423ce7d8, i0 queue de handle `FUN_14076e3e4` @14230d055, etats par defaut ti=3
      @142eea359 et ti=38 @1408f0bae, les deux slots d'`unit-actor-control` `FUN_1408f0778`
      @1408f0948 / @1408f0962 (`param_3 = 1`). Sites NON RELUS, laisses a `param_3 = 0` (donc au
      comportement d'avant) parce que leur `R8D` est VARIABLE au site d'appel :
      `FUN_140c9e990` @140c9e9cd (`MOV R8D,EBP`), `FUN_1406cd128` @1422f4546 (`[RCX+0x7]`),
      `FUN_142ef45e8` @142ef461b, `FUN_142ef8138`/`8334` (`[RBP-0x3a]`), `FUN_1408eff64`
      @14234fe20 (`[RSI-0x3f]`). Consigne : aucun n'est sur le chemin de ti=37.

      **(3) LE PREFIXE OBJET EST DESORMAIS RELU EN ENTIER.** i1 `object-translational-velocity`
      (`FUN_14076e228` → `FUN_14076d528(..., 10, 0x13)`) et i3 `object-angular-velocity`
      (`FUN_140d70998` → `FUN_14076d528(..., 8, 0x13)`) sont les deux derniers composants
      variables du prefixe : leurs largeurs (19 de direction + 10, resp. 8, de magnitude) sont
      **conformes** au portage. Avec le pas 2, i0 a i17 sont tous relus ou valides.

      **(4) L'EFFET EST REEL, MESURE, ET PETIT.** Fermeture des cinq archetypes objet
      **184 → 188 (0,85 % → 0,87 %)** : ti=37 3 → 4, ti=38 180 → 183, ti=41/42/43 inchanges.
      Attribution site par site (chaque correction neutralisee a tour de role) : **i10 porte tout
      le gain de ti=38** (173 → 183 quand elle seule est retiree), et le record supplementaire de
      ti=37 **n'existe que si i21, i22 ET i28 sont corriges ensemble** — retirer n'importe
      laquelle des trois le fait disparaitre. C'est la signature d'un record qui ferme parce que
      TOUS ses composants sont justes, et c'est la meilleure preuve disponible que les quatre
      lectures sont bonnes. **LA FONDATION N'EST PAS LA POUR AUTANT** : 0,87 % reste 0,87 %.

      **(5) LA CARTE DU DEFAUT SE PRECISE, ET ELLE RETIRE LA DERNIERE AMBIGUITE.** La colonne
      « bloquant » du golden 0.A.3 separe les archetypes qui ne ferment pas en DEUX classes :
      **(A) DESYNCHRONISATION**, un composant sans lecteur est nomme — ti=0, 2, 3, 5, 9, 10, 11,
      12, 19, 25, 26, 27, 34, 35, 43, 45, 47 : leur correction est un PORTAGE (lot 3.6), pas une
      largeur ; **(B) AUCUNE DESYNCHRONISATION et pourtant aucune fermeture — ti=13, 21, 37, 38,
      41, 42**, SIX archetypes, dont DEUX (13 et 21) ne portent meme pas `object-position-component`.
      C'est la population a expliquer, et elle est bien plus petite que « la famille objet ».
      **AUCUN composant n'est commun aux six** (`TestE191cListeCourte` : 51 composants distincts,
      **0 blanchi** par un archetype qui ferme ; ti=13 porte des `managed-object-*`, ti=21 des
      `flock-*`, les quatre autres le prefixe objet). Il n'y a donc PAS de cause unique partagee :
      la suite se traite archetype par archetype. Les couples de meme taille ne discriminent pas
      non plus (`TestE191cPairesDiscriminantes` : ti=19 est 32 fois `sound-placement-state-data`,
      ti=25 un seul `powerframe-player-selection-data` — des classes (A), pas des largeurs).

      **CE QUI EST EN ATTENTE D'ARBITRAGE DU PILOTE, ET C'EST LE SEUL POINT ROUGE.** Le golden
      0.A.3 n'est PAS regenere : la correction fait MONTER quatre lignes (`11de8353` ti=38
      32 → 48, `a521164d` ti=37 1 → 2, `60ae07c4` ti=37 0 → 1, `fb1a1a72` ti=37 0 → 1) et
      DESCENDRE trois (`60ae07c4` ti=38 43 → 30, `11de8353` ti=37 1 → 0, `111fa685` ti=37 1 → 0),
      pour un net de +4. D14 dit que le ratchet des sept ne descend que, et la regle 9 du contrat
      d'execution dit qu'un blocage qui appelle une decision se STATUE et se remonte, il ne se
      contourne pas. L'analyse est ecrite : un record dont le compte de bits ne change pas ne peut
      pas basculer, donc chaque record qui cesse de fermer fermait sur un total que l'ecrivain
      contredit — c'est une DIVERGENCE de bord sur une marche a 0,1 %, pas une REGRESSION de
      grammaire. Le geste, si le pilote le ratifie, tient en une commande :
      `go test ./internal/games/halo_infinite/film/filmdec/ -run KeyframeClosureRatchet -update-keyframe-closure`.
      **`SchemaVersion` n'est PAS montee non plus** : des bits changent, donc le contenu cuit PEUT
      changer, et seul `replay-corpus-gate` le dit — il exige « voie libre ».

      **PAS 2 TER ET 2 QUATER FAITS le 2026-09-15 — LE DEFAUT EST LOCALISE : IL EST DANS
      L'ETAT PAR DEFAUT, ET IL VAUT TROIS BITS DANS `FUN_14080cfe8`.**

      **(1) LES ETATS PAR DEFAUT SONT TOUS CONFORMES (pas 2 ter).** ti=38 `FUN_1408f0b48`,
      ti=36 `FUN_1407f2224`, ti=43 `FUN_140fe7630`, ti=37 `FUN_1407f105c`, ti=42 `FUN_1407f0c68`,
      ti=13 `FUN_140ce55e8`, ti=21 `FUN_141133c24`, ti=9 `FUN_1410d7540`, ti=10 `FUN_141020244` —
      forme relue, conforme au portage. ti=41 n'est resoluble par aucune lecture statique
      (descripteur construit au runtime). Une seule divergence, sur le chemin CUIT et non sur la
      fermeture : les deux largeurs du bloc MPP sont des LITTERAUX chez l'ecrivain (R(9) en
      `141fd72de`, R(5) inline) et deux globales CALIBREES en production. Conversion demandee,
      **en attente de voie libre** (elle exige un gate).

      **(2) LES DEUX MOTS DE TAILLE SONT DES GARDES, ET CELA VAUT +68 RECORDS (pas 2 quater).**
      `FUN_142e2bfd0` porte deux fois `if (0 < (int)uVar7)` : `n1 == 0` -> aucun etat par defaut,
      `n2 == 0` -> aucun composant, comparaison SIGNEE. Porte. Fermeture des cinq archetypes
      objet **184 -> 246 / 21 698 (0,85 % -> 1,13 %)**, ti=41 **0 -> 34/110**, et des archetypes
      hors prefixe objet montent aussi (ti=35 bipede 0 -> 6). 26 lignes du golden montent,
      2 descendent, net **+68**.

      **(3) `n2` EST UN ORACLE GRATUIT, ET IL DESIGNE L'ETAT PAR DEFAUT.** `n1` est CONSTANT sur
      TOUS les archetypes mesures (4, 12, 1, 136, 100, 164) — l'en-tete de 108 bits est donc
      exact partout, y compris la ou rien ne ferme. `n2` est constant chez ceux qui ferment
      (28, 432, 12, 256, 7 896) et du BRUIT chez ceux qui echouent. Comme `n2` se lit JUSTE APRES
      l'etat par defaut, **le premier bit faux est dans l'etat par defaut, avant le premier
      composant** : relire quinze composants ne pouvait rien donner. Le garde-rail qui porte deja
      cet oracle etait desarme precisement sur ces archetypes-la.

      **(4) LE DEFAUT EST BORNE A TROIS BITS DANS UNE FONCTION NOMMEE.** Balayage 16 × 16 des
      deux largeurs du bloc MPP contre `n2` : la MEME paire (`lead=8 index=3`) fait passer la
      part modale de 0,204 a 0,639 (ti=37), de 0,053 a 0,617 (ti=38) et de 0,059 a 0,732
      (ti=42), avec des `n2` modaux plausibles (1 396 / 1 764 / 1 300). NON PORTE : 8 et 3
      contredisent les litteraux de l'ecrivain, et accorder un decodeur a une mesure est ce que
      D13 interdit. La lecture juste : **`FUN_14080cfe8` consomme trois bits de trop**. La part
      modale plafonne a 0,62-0,73 : le bloc porte encore un element variable au-dela.

      **CE QUE LA REPRISE DOIT FAIRE, DANS CET ORDRE** : relire `FUN_14080cfe8` champ par champ
      pour trouver OU sont les trois bits ; faire juger ti=36..39, 42, 43 par
      `default_state_n2_constant_test.go` ; puis la conversion MPP (D9) et les gates.

      **PAS 2 QUINQUIES FAIT le 2026-09-15 — LA GRAMMAIRE EST VERSIONNEE PAR BUILD, ET LE
      PORTAGE ETAIT JUSTE POUR LES BUILDS RECENTS.**
      Le balayage du pas 2 quater, rejoue BOBINE PAR BOBINE, coupe net a `HI_1_12_0` : les cinq
      builds anciens gagnent a `lead=8 index=3` (0,988 a 0,996) et les deux recents a
      `lead=9 index=5` — **les largeurs de l'ecrivain**, qui y rendent 0,949 et **1,000**. La
      version MAJEURE du film ne discrimine pas (`e5adf7b2` et `bcb6d393` sont tous deux `v=40`
      et tombent de part et d'autre) ; le CARDINAL DE LA TABLE PAR TYPE de `chunk_00`, si :
      **116 / 121 / 121 / 121 / 122 d'un cote, 123 / 123 de l'autre**, et il est lisible hors
      ligne (`FilmIdentity.TypeVersions`). Les « trois bits de trop » de D12 ne sont donc PAS une
      largeur fausse : c'est un PROFIL MANQUANT pour les builds <= HI_1_11_0.
      `FUN_14080cfe8` a ete relu en entier : toutes ses largeurs sont des litteraux et son seul
      `if` runtime ne consomme aucun bit — **il n'y a aucune branche de version dans le bloc**,
      et la grammaire des builds anciens n'est pas relisible dans cet executable-la. C'est la
      LIMITE de la regle D13, et elle se nomme : pour une grammaire versionnee, « relire chez
      l'ecrivain » exige l'executable DU BUILD. La version sur laquelle le jeu branche ailleurs
      vient d'une structure runtime attachee au film charge — donc DU FILM, que `chunk_00` porte.
      **Consequence pour la conversion MPP** : la retirer sans profil casserait les cinq builds
      anciens. `CalibrateMPPWidths` EST le profil manquant, ecrit en heuristique. La conversion
      juste est « le PROFIL PAR BUILD remplace la calibration, la calibration devient le
      controle », avec build inconnu = erreur typee (ADR 0034). **Rien n'est porte : le pas est
      une mesure et deux instruments.**

      **PAS 3 FAIT le 2026-09-15 — LE PROFIL PAR BUILD EXISTE, ET LA CASE MPP DES CINQ BUILDS
      ANCIENS RESTE VIDE PARCE QUE LES DEUX ORACLES SE CONTREDISENT.**

      **(1) UN SEUL LIEU.** `player_table_profile.go` devient **`build_profile.go`** et porte les
      deux donnees par build — largeur du bloc de personnalisation (lot 1.5.2) et largeurs du
      bloc MPP — avec une PROVENANCE par ligne, dans les deux formes que la regle du pilote
      distingue : **RELU** (`HI_1_12_0`, `HI_1_13_0`, fonction citee) et **MESURE** (l'oracle et
      son score). `BuildProfileFor` rend le profil ou `ErrUnknownBuild` (D-4 : jamais le profil
      du build le plus proche) ; `BuildProfileFromFilm` le resout depuis la section 2 ;
      `InstallBuildProfileMPP` l'installe pour la duree d'une mesure, et **n'installe rien quand
      la largeur est indeterminee**. `KeyframeClosure` l'appelle.

      **(2) LA MESURE QUI A CHANGE LA CONCLUSION.** L'oracle `n2` designait `8/3` sur les cinq
      builds `<= HI_1_11_0` (0,988 a 0,996 contre 0,304 a 0,522). Le POSER fait DESCENDRE la
      fermeture de **246 a 182**. Le balayage de la FERMETURE, bobine par bobine, ne designe
      aucun decoupage coherent — et **il se disqualifie lui-meme** : sur `fb1a1a72`, le build de
      l'executable, ou l'ecrivain ecrit `R(9)` par un litteral, il prefere `10/5` (140 records) a
      `9/5` (11). A moins de 5 % de fermeture, maximiser un compte de fermetures revient a
      chercher des coincidences. `n2`, lui, est VALIDE la ou l'ecrivain est connu (1,000 a `9/5`
      sur ce meme film). **Poser `8/3` aurait fait descendre le ratchet de 64 records (D14
      l'interdit) ; poser `9/5` partout aurait affirme une largeur que `n2` refute.** La case
      reste vide, ce qui est l'etat reel de la connaissance, et l'arbitrage remonte.

      **(3) GARDE-RAILS.** `build_profile_test.go` : table figee EN CLAIR, refus d'un build
      inconnu, et **la mutation qui rougit** — fausser le decoupage d'UN bit sur `HI_1_13_0`,
      dans les quatre sens, disperse `n2`. Fermeture **inchangee a 246/21 698**. `GrammarRev`
      `.4` -> `.5` : aucun bit lu ne change, mais le CADRE si — une grammaire peut desormais
      dependre du build.

      **CE QUI N'EST PAS FAIT, ET POURQUOI.** (a) Les cinq films SANS section 2 (`03af54c3`,
      `13b00e35`, `47d20b5d`, `50247b26`, `a349fea8`) : leur repli au registre devait leur donner
      un PROFIL — or le profil MPP des builds anciens est indetermine, donc il n'y a rien a leur
      donner. `[!]` en attente de l'arbitrage. (b) `CalibrateMPPWidths` en CONTROLE et hors du
      chemin cuit : la retirer laisserait les cinq builds anciens SANS largeur du tout, puisque
      leur case est vide. `[!]` meme dependance. (c) Le corpus gate et l'equivalence : **en
      attente de voie libre**.

      **CLÔTURE DU LOT 1.9.1 bis le 2026-09-16 — CE QUI EST ACQUIS, CE QUI RESTE OUVERT.**

      **ACQUIS, ET TENU PAR DES GATES.**
      1. **Les deux mots de taille sont des GARDES** (`FUN_142e2bfd0`, deux `if (0 < (int)uVar7)`,
         comparaison SIGNÉE) : `n1 == 0` -> aucun état par défaut, `n2 == 0` -> aucun composant.
         **+68 records fermés** sur les sept bobines ; les cinq archétypes objet passent de
         184/21 698 à **246/21 698**, ti=41 de 0/110 à **34/110**, et des archétypes hors préfixe
         objet montent aussi (ti=35 bipède 0 -> 6, ti=10 0 -> 3, ti=12 0 -> 1).
      2. **Le profil par build, un seul lieu** : `build_profile.go` porte les deux données par
         build (personnalisation, largeurs MPP) avec **une provenance par ligne** — RELU
         (`HI_1_12_0`, `HI_1_13_0` : 9/5, `FUN_141fd72c0` littéral `141fd72de`) ou MESURE.
         `BuildProfileFor` rend `ErrUnknownBuild` (D-4), et le profil passe DEVANT la calibration
         sur le chemin cuit pour les builds relus.
      3. **Le repli est NOMMÉ au registre** : `repli_largeurs_mpp_calibrees_sur_le_film`,
         condition neuve `build_sans_profil_relu`, ordre `apres_lecture`, cible et critère de
         retrait écrits.
      4. **Le préfixe objet est relu en entier** : les quinze composants i0..i17 de ti=37, chacun
         avec sa fonction citée, tous bit-exacts — aucune largeur n'a changé.
      5. **L'oracle du corpus gate est réparé** : il JETAIT `BilanAxe.Changements` (D25).
      6. **Gates joués** : équivalence 10/10 après re-figeage classé, corpus gate schéma 59 -> 59,
         0 changement, 31 gains, 3 pertes toutes classées divergence. `SchemaVersion` reste **59**.
      7. **24 découvertes consignées** (§4, D1 à D25) et huit instruments versionnés, rejouables
         sans garde d'environnement sur les sept bobines.

      **OUVERT, ET BORNÉ.** La **CONDITION versionnée de l'état par défaut des films anciens**.
      Ce qui est établi : le film est autoportant (la boucle de composants lit le registre DU
      FILM, `param_3 * 0x4100 + 8 + lVar4`) ; le bloc MPP est INVARIANT (aucune branche de
      version, `FUN_141fd72c0` n'a qu'un appelant) ; le préfixe `V` ne discrimine pas (identique
      7/7) ; aucun seuil de version MAJEURE ne le peut non plus (`e5adf7b2` et `bcb6d393` sont
      tous deux `v=40` de part et d'autre) ; la différence est STRUCTURELLE et non un offset
      (aucun décalage de -6 à +2 ne rend `n2` constant) ; la clé est la table PAR TYPE, alignée
      PAR LA FIN, bornée à **douze positions** ; et elle n'est pas rangée derrière le registre
      (`0x145008` : zéro instruction sur 13,6 M).
      **POINT DE REPRISE (lot 1.9.1 ter, exécuteur frais)** : remonter le CHARGEUR de la
      section 2 par ses ÉCRITURES sur la structure du film, et y trouver la condition. Le plan
      seul suffit — §4 D1 à D25 et §5 portent tout.
      **CADUC LE 2026-09-15 (lot 1.9.1 ter)** : la piste « douze positions alignées PAR LA
      FIN » est réfutée par la mesure (D4, 1.9.1 ter) — la table est alignée PAR LE DÉBUT. Le
      chargeur a bien été remonté, et la clé est `chunk_00+4` : voir l'item 1.9.1 ter ci-dessous.
- [x] 1.9.1 ter **LA CONDITION VERSIONNÉE : LE CHARGEUR EST TROUVÉ, LA CLÉ EST `chunk_00+4`, ET
      LES TROIS BITS NE SONT PAS UNE BRANCHE DE VERSION.** Fait le 2026-09-15 (branche
      `feat/decfilm-191t`, Ghidra HTTP direct, LECTURE SEULE).

      **(1) LE CHARGEUR DE LA SECTION 2, ET CE QU'IL EN FAIT.** Le sérialiseur de `chunk_00` a
      deux faces, et elles ne sont PAS symétriques — c'est tout le sujet :

      | face | fonction | largeur du registre | largeur de la table par type |
      |---|---|---|---|
      | écriture | `FUN_14299b198(film, écrivain)` | `W(0x659000)` **littéral** | `W(0xF60)` **littéral** (`MOV R9D,0xf60` @14299b1e4) |
      | lecture | `FUN_14299ab50(film, lecteur)` | `R(FUN_141cfff30(film+4))` | `R(FUN_141cffe20(film+4))` |

      **Le second u32 de `chunk_00` est LA VERSION DE FORMAT**, et c'est le seul paramètre dont
      la lecture de l'en-tête dépende. Les deux fonctions de taille sont des recherches dans
      deux `std::map<uint,int>` construites au démarrage, littéraux relus sur le désassemblage
      de `FUN_140268ec0` et `FUN_140268f40` : blocs de registre `{13:47, 17:48, 18:49, 25:25}`
      défaut 50 ; entrées de table `{13:110, 16:113, 17:114, 21:117, 22:118, 24:121, 25:122}`
      défaut 123. **C'est la forme mécanique du fait utilisateur du 2026-09-16** : la grammaire
      des formats anciens EST dans l'exécutable courant, indexée par ce u32 — aucun exécutable
      ancien n'est nécessaire. En queue, `FUN_14299bcb0(film, film+4)` REMPLIT les entrées de
      table au-delà du cardinal du film depuis les descripteurs construits au runtime
      (`DAT_144e61d88 + 0x210 + i*8`, appel virtuel `vtable+0x30`) : la piste « descripteur
      runtime » du brief est donc VRAIE, mais elle ne sert qu'au COMBLEMENT, pas à brancher.

      **CE QUE LA TABLE PAR TYPE DEVIENT, ET QUI LA CONSULTE.** Elle vit à `film+0xCB208` et n'a
      qu'**UN** lecteur indexé — `FUN_1428e1c64` (quatre sites d'instruction `0xcb208` sur
      13,6 M balayées), sous verrou, qui choisit entre le tampon d'enregistrement
      (`*(film+0x108)`) et celui de lecture (`*(film+0x120)+0x130`). Son **unique** appelant est
      `FUN_141102ed0(i)` : en mode Theater il rend la version du type `i` DU FILM, hors film
      celle de la table native `DAT_14474cd90` (123 entrées `{version, taille}` dans l'image).
      **Quinze fonctions** s'en servent pour brancher, sur neuf index littéraux — 0x23, 0x24,
      0x30, 0x59, 0x5a, 0x5b, 0x5d, 0x61, 0x72.

      **(2) LA CONDITION — ET ELLE N'EST PAS OÙ LE BRIEF LA CHERCHAIT.** La version de format
      SÉPARE exactement les deux groupes mesurés : `21 / 24 / 24 / 24 / 25` pour les cinq
      bobines `8/3`, `27 / 27` pour les deux bobines `9/5` — seuil dans **]25, 27]**
      (`TestE191tVersionDeFormat`). Mais **aucune branche de l'exécutable n'y répond** :

      - la version de format n'a que **SIX** sites d'appel (`FUN_1428e1c0c`), de seuils **4, 7,
        12, 13/14, 16** — aucun entre 25 et 27 ;
      - la table par type ne discrimine PAS : **24 index varient** entre bobines, **0** sépare
        les deux groupes, et aucun des neuf index interrogés n'en fait partie
        (`TestE191tIndexQuiVarient`) ;
      - la chaîne `FUN_1407f105c` (ti=37) → `FUN_1407f2224` (ti=36) → `FUN_14080cfe8` (MPP) →
        sous-lecteurs ne contient **ni** `FUN_1428e1c0c` **ni** `FUN_141102ed0`, et toutes ses
        largeurs sont des littéraux (`FUN_141fd72c0` : `ADD dword ptr [RCX+0x2c],0x9`
        @141fd72de, un seul appelant).

      **L'exécutable lit donc l'état par défaut de ti=37 à l'identique pour un film de format 21
      et un film de format 27.** Les trois bits ne sont pas une branche de version : D3 (1.9.1
      ter). **La seule condition de version sur le chemin du record est ailleurs, et elle est
      nommée** : `FUN_142e2bfd0` @142e2c020 fait `si (7 < version_de_format)
      DAT_144706104 = R(1)` — un bit DU FILM en tête du paquet d'image-clé, qui commande la
      table de plages de `FUN_1406d3140` (i10, i21, i22, i28, états par défaut de ti=3 et
      ti=38). Le lot 1.9.1 bis avait pris cette garde pour une constante statique ; elle ne
      l'est pas, et la version dont elle dépend est désormais LISIBLE HORS LIGNE. D2 (1.9.1
      ter).

      **(3) CE QUI EST PORTÉ.** (a) `film_format_version.go` : `FilmFormatVersionFromHeader` /
      `FilmFormatVersion`, avec la carte du chargeur et les deux tables de l'écrivain écrites.
      (b) `FilmIdentity.FormatVersion`. (c) `build_profile.go` : le découpage MPP est keyé par
      la **VERSION DE FORMAT** (`mppWidthsPourFormat`), plus par le nom de build — ce que la
      directive utilisateur demande. **La largeur de personnalisation reste keyée par le BUILD,
      et la mesure l'IMPOSE** : le format 24 porte `HI_1_10_0` (1 492 octets) ET `HI_1_9_0` /
      `HI_1_8_0` (1 312) — une version de format, deux largeurs. Deux clés donc, et elles ne
      disent pas la même chose : le FORMAT dit la grammaire des bits, le BUILD la taille des
      structures de contenu. (d) `InstallBuildProfileMPP` devient `InstallFilmFormatMPP` et ne
      lit plus la section d'identification. (e) `varwidth.go` : le commentaire qui affirmait que
      la garde « ne se lit pas offline, la version venant d'un objet de configuration du jeu et
      non du film » est CORRIGÉ. (f) `ErrUnknownFormat` + son test de refus (formats 0, 19, 22,
      23, 26, 28, 40 — dont le **26**, entre deux formats connus, là où l'interpolation serait
      tentante). **AUCUN BIT LU NE CHANGE** : sur les 1 351 films du cache les deux clés rendent
      le même découpage. `GrammarRev` `.7` -> `.8` (le CADRE change), `SchemaVersion` reste
      **59**.

      **(4) LES CINQ FILMS SANS SECTION 2 — RÉSOLU, MESURÉ.** `03af54c3`, `13b00e35`,
      `47d20b5d`, `50247b26`, `a349fea8` portent tous **format 20**, lisible à `chunk_00+4` sans
      aucune chaîne de build, et ce sont les **cinq seuls** du cache à le porter. La section
      d'identification apparaît donc au **format 21** : leur absence de chaîne de build n'est
      pas une anomalie de fichier, c'est un format antérieur. Ils entrent désormais dans
      `InstallFilmFormatMPP` par la porte principale ; leur largeur MPP reste indéterminée comme
      les quatre autres formats anciens, donc rien n'est installé et aucun bit ne change.
      Distribution complète sur les 1 351 `chunk_00`, couple (version majeure, version de
      format) : `(41,27)x1123 · (40,27)x146 · (40,25)x39 · (39,24)x26 · (37,24)x10 · (31,20)x3 ·
      (33,20)x2 · (38,24)x1 · (33,21)x1` — **elle reproduit à l'unité la partition par build que
      `film_identity.go` dérivait par arithmétique** (1 269 / 39 / 37 / 1) et la couvre là où la
      dérivation s'arrêtait.

      **(5) CE QUI RESTE OUVERT, ET C'EST BORNÉ.** La case MPP des formats 20/21/24/25 reste
      VIDE : l'oracle `n2` mesure bien trois bits de moins sur ces films, mais leur ENDROIT
      n'est pas établi et l'exécutable vient d'exclure l'état par défaut. La piste suivante
      n'est plus « quelle version branche » — la question est close — mais « quel bit du paquet
      n'est pas consommé », et D2 (1.9.1 ter) la nomme : mesurer la valeur de `DAT_144706104`
      film par film. **STATUT DES ITEMS DU BRIEF** : 1 (le chargeur) `[x]` · 2 (la condition)
      `[x]` pour la recherche et la mesure, `[!]` pour la fermeture n2/archétypes — il n'y a pas
      de branche à porter, donc pas de montée de fermeture à mesurer · 3 (portage) `[x]` pour la
      grammaire versionnée lue depuis le film et `GrammarRev`, `[!]` pour la sortie de
      `CalibrateMPPWidths` du chemin cuit — elle reste, nommée, tant que la case des formats
      anciens est vide (même dépendance qu'au 1.9.1 bis) · 4 (les cinq films) `[x]` · 5 (gates)
      `[x]` — régime court joué, et **équivalence jouée le 2026-09-15 sur « voie libre » du
      pilote** : classification AVANT tout `-update` (10 films, **une seule étape,
      `placements.stats`**), cause établie par LECTURE DU CODE et non par inférence sur les sha,
      re-figeage sur feu vert (`git diff` = 10 fichiers, 10 lignes `placements.stats` et rien
      d'autre), passe de comparaison **10/10**. Bloc de re-figeage dans `CORPUS.txt`, quatre
      entrées au §5. **Le corpus gate n'est pas requis** : aucun octet cuit ne change — l'étape
      `artifact` est identique à sa référence sur les dix films, avant comme après le re-figeage,
      et les deux champs ajoutés n'atteignent pas le document (preuve structurelle, §5).
      `SchemaVersion` reste **59**.


      **(6) LES CONSÉQUENCES DE LA CLÉ FORMAT, TRAITÉES (arbitrage du pilote, 2026-09-15).**
      Les trois constats remontés à la clôture du pas précédent entraient dans ce lot : la clé
      format est de moi, ses conséquences aussi.

      **(a) LA CONDITION DU REPLI EST RENOMMÉE.** `build_sans_profil_relu` ->
      **`format_sans_profil_relu`** (`fallback/repli.go` : constante `CondBuildSansProfilRelu` ->
      `CondFormatSansProfilRelu`, valeur, domaine fermé `conditionsConnues`, et son commentaire ;
      `fallback/registre_filmdec.go` : l'usage et toute la prose de l'entrée, qui disait
      « VERSIONNEE PAR BUILD » ; les deux commentaires de site, `filmdec/equipment_placements.go`
      et `replay/build_ground_weapons.go`). L'ancien nom décrivait une clé qui n'existe plus.
      **Les traces DATÉES gardent l'ancien nom** — l'entrée `.6` du journal de
      `grammar_rev.golden`, le bloc 1.9.1 bis de ce plan, le thought_log : ce sont des relevés du
      jour où la condition a été posée, pas la doc courante, et les réécrire effacerait
      l'historique au lieu de le corriger. Le journal `.9` du golden porte le renommage, et
      l'en-tête de `e191c_n2_oracle_research_test.go` porte désormais la correction de son
      interprétation (sa MESURE tient, sa clé était fausse).

      **(b) UN COMPTEUR CÂBLÉ, SUR LE MODÈLE EXACT DE `UnknownBuildExpvarPairs`.**
      `filmdec.UnknownFormatExpvarPairs(format)` -> **`filmdec_unknown_format_<n>`**
      (`film_format_version.go`), avec `MPPWidthsForFormat` qui expose enfin la frontière entre
      les **deux « pas de profil »** : format CONNU sans largeur relue (20, 21, 24, 25) = état
      normal du parc ancien, **rien à compter** ; format INCONNU = l'événement « patch du jeu ».
      Câblage dans `replay/mpp_format_inconnu.go`, même patron que `publierBuildInconnu` —
      `filmdec` NOMME, `replay` CÂBLE. **Le compteur tombe aux DEUX sites du registre** :
      `replay/equipment_placements.go` (par le drapeau neuf `EquipmentPlacementStats.
      FormatSansProfil`, parce que la décision est prise dans `filmdec`, qui ne publie pas) et
      `gwWidthsForFilm` (qui sert les armes au sol ET les véhicules). **UN SEUL avertissement par
      film**, émis par `BuildFromFilm` avant tout balayage, avec `match_id`, `format`, `build` —
      trois lignes identiques par cuisson noieraient le signal. Le comptage des replis par
      `FilmContext` **reste différé au pas 2 de M2** comme le registre l'écrit
      (`CompteurBranche: false`) : ce compteur-ci ne compte pas un repli ordinaire.
      **La DÉCISION n'est pas touchée** : `gwWidthsForFilm` continue de se décider sur le profil
      COMPLET (`BuildProfileFromFilm`, qui refuse aussi sur build inconnu) ; la re-poser sur le
      seul format ferait basculer un film au build inconnu mais au format connu vers le profil,
      donc changerait des octets cuits.

      **(c) LA BRANCHE DE PRODUCTION EST GARDÉE, ET LA MUTATION EST JOUÉE.**
      `replay/mpp_format_inconnu_test.go` : une bobine **présentée sous le format 28** — les
      quatre octets que la grammaire désigne (`chunk_00+4`) réécrits, jamais une position devinée
      — doit (a) garder les largeurs **CALIBRÉES** et (b) incrémenter
      `filmdec_unknown_format_28`. Second test, l'autre moitié du contrat : une bobine INTACTE au
      format 21 (parc ancien) ne compte **rien**, sans quoi le compteur ne signalerait plus aucun
      patch. **Mutations vérifiées dans les deux sens** : refus sec (`return MPPWidths{}`) ->
      `largeurs 0/0, 7/4 attendues` ; compteur retiré -> `0 -> 0, attendu +1` ; **restauration par
      NOM** -> vert. Aucun des 1 351 films du cache n'a de format hors table (20/21/24/25/27), la
      réécriture est la seule façon de jouer cette branche.

      **(d) `keyframe_closure.go:125` NON TOUCHÉ**, sur instruction : consigné en D5 (1.9.1 ter).
      L'écart `slog.Warn` / `slog.WarnContext` est consigné en D6, avec sa raison mesurée : il
      n'existe aucun `context.Context` sur ce chemin de cuisson.

      **BUDGET DE TEST (règle du 2026-09-15 : instrument neuf = tag `research`).**
      `e191t_version_format_research_test.go` porte `//go:build research` ; les quatre
      instruments y vivent (0,12 s avec `-tags research`, le balayage du cache resté derrière
      `CHUNK00_FILMS`). Le BUILD PAR DÉFAUT ne reçoit que `film_format_version_test.go`, trois
      tests à **0,05 s** (plancher du paquet 0,052 s, avec eux 0,106 s, `CGO_ENABLED=0`) : ils
      lisent les sept `chunk_00` et rien d'autre, et ils ne dépendent d'AUCUN helper tagué — la
      table des largeurs MPP mesurées y est recopiée, avec sa provenance. `go vet` passe SANS
      tag et AVEC `-tags=research`. Mesure du paquet complet sur cette branche : **92,4 s**,
      dont **77,3 s** pour les `TestE191` de 1.9.1 bis, qui ne portent pas encore le tag ici
      (le commit de l'intégration est postérieur à la base de la branche, résolu à la fusion) —
      soit **~15 s** une fois le tag appliqué, budget 30 s tenu.

- [x] 1.9.2 **Le découpage d'i0 vient du catalogue de carte, plus de l'auto-détection.**
      `internal/sync/killcollector/positions.go:253` (et `hits.go:157`) construisent
      `DefaultScanFilmOptions()` avec `Layout` nil alors que `entry` est le paramètre de la fonction
      et que `entry.Range()` est lu à la ligne suivante ; `MapQuantEntry.Layout()`
      (`filmdec/map_bounds.go:69`) est déjà imposé sur l'autre chemin (`replay/build_from_film.go:87`).
      Gain : **27 faux enregistrements sur 267 400 éliminés sur Live Fire** (mesure du 2026-09-03 sur
      `60ae07c4`, `filmdec/film_context.go:33-42`) et deux passes de détection supprimées par film. S.

      **FAIT le 2026-09-15.**

      **LES CITATIONS DU BLOC ONT DÉRIVÉ, ET LA CARTE EXACTE A ÉTÉ REFAITE PAR GREP.** `hits.go:157`
      ne construit plus rien : le code a déménagé dans `filmdec/weapon_hit_distance_resolver.go`
      (`BuildBipedTracks`). Les sites de PRODUCTION de `DefaultScanFilmOptions()` sont QUATRE, pas
      deux — `weapon_hit_distance_resolver.go:55`, `replay/build_from_film.go:110`,
      `replay/build_vehicles.go:173`, `killcollector/positions.go:253` — et les deux du milieu
      posaient déjà `ImposedLayout()`. Ce lot convertit les DEUX autres. Les TIRS
      (`ScanFilmWeaponShots`) et les DÉGÂTS (`ScanFilmWeaponDamages`) ne lisent PAS i0 : ni
      `I0Layout` ni `DetectI0` n'apparaissent dans `weapon_hits.go` (grep collé en §5), donc rien à
      y convertir et aucun enregistrement de tir ne change.

      **LA MESURE AVANT DE CODER — 17 FILMS (les 14 témoins du corpus gate et les 8 builds),
      TROIS JEUX DE RÉGLAGES.** Instrument versionné `filmdec/e192_i0_catalogue_mesure_research_test.go`
      (garde `CHUNK00_FILMS`, table des cartes recopiée de `config/replay_corpus.toml` et de
      `CORPUS.txt`) ; tableau intégral collé en §5. **Catalogue et auto-détection donnent le MÊME
      découpage sur QUINZE films sur dix-sept**, et les positions y sont identiques au record près.
      **DEUX films divergent, les deux Live Fire** — seule carte du catalogue dont la région jouée
      n'est pas la première du bloc structure-BSP (`gate=6 region=1 12/12/11` contre l'auto-détecté
      `gate=5 region=0 13/12/11`, MÊME longueur totale d'i0 : 41 bits) :

      | film | build | carte | positions (réglages de production) | pistes de touche | enregistrements bruts |
      |---|---|---|---|---|---|
      | `60ae07c4` | HI_1_8_0 | Live Fire - Ranked | 267 368 -> 267 365 (**−3**) | 267 390 -> 267 374 (**−16**) | **267 400 -> 267 374 (−26)** |
      | `0797ce72` | HI_1_13_0 | Live Fire | 146 811 -> 146 807 (**−4**) | 146 853 -> 146 849 (**−4**) | 146 860 -> 146 849 (**−11**) |
      | les 15 autres | 8 builds | 15 cartes | **+0** | **+0** | **+0** |

      **LA MESURE DU 2026-09-03 SE REJOUE, ET SON DÉNOMINATEUR EST RETROUVÉ À L'UNITÉ** : sur
      `60ae07c4`, la population d'enregistrements bruts (tag exigé, aucun filtre) vaut **267 400**
      sous l'auto-détection et **267 374** sous le catalogue — **26**, quand le plan en cite 27.
      L'écart d'un enregistrement n'est pas expliqué (le jeu de réglages exact du 2026-09-03 n'a pas
      été consigné) et ne change aucune décision : D4 (1.9.2) en §4.

      **CE QUE LE LOT CHANGE.** (a) `killcollector/positions.go` ouvre un `FilmContext` SOUS LA
      RÈGLE DU CATALOGUE (`NewFilmContextForMap(film, &entry, nil)`) et en tire les réglages par
      `optionsDeBalayageDesPositions` — un seul découpage pour les positions ET pour les créations
      de bipède, qui partagent désormais ce contexte au lieu d'en ouvrir deux. (b)
      `filmdec/weapon_hit_distance_resolver.go` prend l'ENTRÉE DE CATALOGUE (`MapQuantEntry`) là où
      il ne prenait que ses bornes : `BuildBipedTracks` et `FilmWeaponHitDistance` imposent
      `entry.Layout()`, et `DetectFilmWorldRange` devient `DetectFilmMapEntry` (elle rend l'entrée
      entière). Une entrée sans découpage exploitable rend une **erreur typée** (`ErrUnknownMapBounds`)
      et la carte hors catalogue est **comptée** (`killsource_hits_cartes_hors_catalogue`, D-4) — le
      chemin des positions avait déjà son compteur (`killsource_positions_sans_carte`).

      **CE QUI DÉCIDE DÉSORMAIS, VÉRIFIÉ PAR GREP (collé en §5).** Les QUATRE sites de production
      posent `Layout` depuis le catalogue ; `NewFilmContext` (le constructeur SANS catalogue) n'a
      plus AUCUN appelant de production hors des enveloppes D2 de `filmdec` lui-même ;
      `DetectI0Layout` garde UN appelant de production, `DetectFilmMapEntry`, où ce n'est plus le
      DÉCOUPAGE qu'elle décide mais l'IDENTITÉ de la carte par signature de largeurs — l'objet du
      lot 1.9.4.

      **LE REGISTRE DES REPLIS : RÉTROGRADATION, ET LE RATCHET DES SEPT DESCEND À SIX.**
      `repli_i0_porte_et_region_par_defaut` ne SORT pas (son ancre vit toujours, et l'identité de
      carte l'emprunte encore) : il passe de `inconditionnel / devant_la_lecture` à
      **`carte_absente_du_catalogue / apres_lecture`**, cible de retrait `lot 1.9.4`. La condition
      `CondCarteAbsenteDuCatalogue` est NEUVE au domaine fermé — `film_muet` aurait menti : le film
      n'est pas muet, c'est le référentiel qui manque, et le geste qui retire ce repli est d'ajouter
      la carte au catalogue, jamais de mieux lire le film. Registre : **94 entrées** (inchangé),
      `NbDevantLaLecture` **7 -> 6**, ratchet baissé dans le même commit.

      **LES MUTATIONS, JOUÉES ET RESTAURÉES PAR NOM.** (A) catalogue versionné faussé — `live fire`
      `axisWidths` X **12 -> 13** dans `map_quant_bounds.json` : `filmdec` ROUGE
      (`TestI0CatalogueEstLaSourceDuDecoupage` + `TestI0CatalogueEcarteLesAutresRegions`) et
      `killcollector` ROUGE (`TestMutationDuCatalogueChangeLeDecoupageDesPositions`) ; restauré par
      `git checkout -- <le fichier>`. (B) câblage retiré — `opt.Layout = fc.ImposedLayout()`
      supprimé de `positions.go` : `TestPositionsImposentLeDecoupageDuCatalogue` ROUGE sur les
      **79 cartes** du catalogue (« l'auto-détection déciderait »), et
      `TestMutationDuCatalogueChangeLeDecoupageDesPositions` rouge aussi ; restauré par nom depuis
      une copie hors dépôt. **La première rédaction du témoin `filmdec` était FAUSSE et la mutation
      (A) l'a prouvé** : elle écrivait les bits SOUS le découpage du catalogue puis les relisait
      sous le même — écriture et lecture bougeaient ensemble, et fausser le catalogue restait vert.
      Les bits du témoin sont désormais écrits sous un découpage FIGÉ
      (`decoupageDeReferenceLiveFire`), comme un film le ferait.

      **RÉVISIONS.** `GrammarRev` **grammar-2026-09-15.1 -> grammar-2026-09-15.2** (golden régénéré,
      historique écrit). **`KillSourceDecoderRev` NE MONTE PAS, et c'est délibéré** : le journal des
      morts ne change pas d'un octet, `film/killsource/` n'a pas bougé, et
      `decoder_rev_fingerprint_test.go` rougirait à juste titre (« la révision a changé sans le
      décodeur ») ; la faire monter rouvrirait un backlog complet pour rien. C'est
      **`IsolationDecoderRev`** qui porte le redécodage — `isolement-2026-09-10-pont-a-l-instant ->
      isolement-2026-09-15-decoupage-du-catalogue` — exactement comme au lot 6.1 (2026-09-10), qui
      l'avait bumpée pour un changement de `kill_positions` / `kill_openings` : ces deux tables ne
      portent PAS de `decoder_rev`, et cette révision-ci est la seule que `matchsAJour`
      (`cmd_backfill_killsource_selection.go`) consulte pour elles. **POPULATION CONCERNÉE : 70
      matchs Live Fire au registre du parc sur 1 967, dont 52 ont un film en cache** — les seuls dont
      les lignes changent ; les 1 299 autres films réécriraient une passe au contenu identique. Le
      backlog reste HORS LOT (bloc « Clôture M1 »). **`SchemaVersion` : 59, INCHANGÉE** — le chemin
      de cuisson imposait déjà le catalogue, et l'équivalence 10/10 identique le vérifie. Aucun
      contrat publié (`replaydoc`, `replayview`, `openapi`) ne change.

      **STATUT DES ITEMS DU BRIEF.** (1) `[x]` mesure avant de coder, tableau collé. (2) `[x]` pour
      le catalogue sur tous les chemins, la rétrogradation au registre et la baisse du ratchet ;
      **`[~]` pour le « contrôle compté dans les stats de collecte »** — couvert par l'ORACLE DE
      TEST, ce que D2 du plan prescrit (« là où le profil sait, l'inférence est un oracle de
      test ») : compter l'accord en production coûterait une SECONDE passe de détection par film,
      c'est-à-dire précisément le gain que ce lot revendique. Justification en D3 (1.9.2). (3) `[x]`
      mutation par catalogue faussé, jouée des deux côtés. (4) `[x]` révisions, population chiffrée.
      (5) `[~]` sans objet : aucun contrat publié ne change, `openapi.yaml` n'est pas touché (`git
      status` collé en §5).
- [x] 1.9.3 **Le couple (tueur, victime) lu au kill-event 85, plus recollé sur le voisin.**
      `internal/games/halo_infinite/film/killsource/feed.go:162` (`reconstructPairs`, fenêtre de
      2 instants) contre `killsource/eventchain.go:242` (`readKillEvent`, victime ET tueur dans le
      même enregistrement, déjà PORTÉ mais lu pour le seul assistant). Gain : **64 couples sur 372**
      cessent d'être une reconstruction ; supprime la fabrication d'un couple quand la vraie victime
      est un bot (`feed.go:149-151`). M.

      **FAIT le 2026-09-15 (branche `feat/decfilm-193`), GATES DE DÉCODAGE COMPRIS.**

      **LA MESURE AVANT DE CODER — 21 FILMS ENTIERS PLUS LA BOBINE VERSIONNÉE.** Instrument
      versionné `killsource/e193_couple_evenement_mesure_research_test.go` (garde `CHUNK00_FILMS`
      pour les films entiers ; la bobine `testdata/minibobine_000d5950` est mesurée SANS garde, en
      CI). Tableaux intégraux collés en §5. **LES 64 SUR 372 SE REJOUENT À L'UNITÉ** sur la série de
      référence du chantier : couples recollés `000d5950` 17, `9b191a7f` 11, `fccc61cd` 23,
      `78919882` 13 = **64** ; couples reconstruits 93 + 85 + 95 + 99 = **372**.

      **CE QUE LE FILM ÉCRIT, MESURÉ CONTRE CE QUE LA RECONSTRUCTION REND.** Le kill-event 85 porte
      des INDICES, pas des noms ; les traduire par la bijection serait CIRCULAIRE (elle s'ajuste sur
      `killFeed.pairs`, donc sur le recollage à juger). La mesure — et la conversion — n'emploient
      donc que les indices ÉPINGLÉS : la table des joueurs de `chunk_00` (lots 1.5 / 1.8) et
      BOT_METADATA, posés AVANT toute inférence. Sur les **21 films entiers** (8 builds, les
      14 témoins du corpus gate, les 4 films de la série de référence) :

      | population | total | accord | désaccord | victime BOT | ambigu | muet |
      |---|---|---|---|---|---|---|
      | kills sans mort en face, régime NAÏF (tueur seul) | 271 | 186 | 3 | 1 | 13 | 68 |
      | kills sans mort en face, régime **d'ASSIGNATION** (celui du lot) | **281** | **198** | **0** | **1** | **1** | **81** |

      **LE PREMIER TEMPS EST LE RÉSULTAT, ET IL EST MESURÉ.** La fenêtre d'appariement vaut 2,5 s de
      part et d'autre : un joueur qui tue deux fois en 2,5 s produit DEUX kill-events au même nom de
      tueur. On donne donc d'abord à chaque couple que le FEED écrit au même instant le kill-event
      dont LE COUPLE ENTIER correspond — une CONSOMMATION, pas une décision —, puis on lit les kills
      orphelins dans ce qui reste. Sans ce temps : 13 ambigus et **3 désaccords apparents** ; avec
      lui : **1 ambigu et ZÉRO désaccord**. Les 3 « désaccords » étaient des enregistrements qui
      appartenaient à une autre mort.

      **CE QUE LE LOT CHANGE, MESURÉ LIGNE PAR LIGNE — UNE SEULE SUR 21 FILMS.** Là où la lecture se
      prononce, elle écrit **exactement** le couple que le recollage rendait : **198 accords sur
      198**. Le gain n'est donc PAS métrique, il est de NATURE — 198 couples cessent d'être une
      reconstruction et deviennent une lecture, comptée. La seule ligne qui bouge est celle qui
      compte : sur `4f77afc1`, le recollage FABRIQUAIT un couple dont le film dit que la victime est
      un **BOT** ; la victime est désormais nommée à la source (« les vies anonymes n'existent pas »,
      décision utilisateur du 2026-09-06), l'instant part vers la population des morts de bot et la
      mort du voisin retourne aux morts que personne ne revendique. Effet de bord mesuré sur la
      mini-bobine : les morts de BOT **proposées** passent de 17 à 16 (un couple que la lecture
      décide n'en est plus un candidat).

      **CE QUI DÉCIDE DÉSORMAIS, ET OÙ.** `killFeed.resoudreCouples` (`killsource/feed_couples.go`,
      NEUF) remplace `reconstructPairs` + `killFeed.split`, tous deux SUPPRIMÉS (grep collé en §5 :
      zéro site de production). `decodeCtx.prepare` localise les kill-events UNE fois
      (`c.killEvents = scanKillEvents(c.film)`) et les partage entre le couple et l'assistant — la
      passe n'est donc pas payée deux fois. `roster.nomEpingle` (NEUF) expose la part LUE du lien
      `indice -> joueur`, celle qui ne dépend d'aucune bijection ; `nameOf`, qui lit `perm`, reste
      réservé à l'aval. `resolveBotDeaths` accepte une TROISIÈME population, celle des victimes de
      bot NOMMÉES, et contraint alors l'appariement à CET indice au lieu de « un bot quelconque ».

      **LE REGISTRE DES REPLIS : UNE ENTRÉE NEUVE, ET LE RATCHET DES SIX NE BOUGE PAS.**
      `repli_couple_recolle_sur_le_voisin` (`section_absente` / **`apres_lecture`**), date de pose
      2026-09-15, cible de retrait « clôture de M1 puis lot 3.6 », critère « `CoupleStats.Recolles`
      à 0 sur les 8 builds et sur le corpus gate ». Registre **94 -> 95 entrées**, 10 compteurs
      câblés. **LE RATCHET DES `devant_la_lecture` RESTE À 6, ET CE N'EST PAS UN OUBLI** : vérifié
      sur pièces, le recollage des couples n'avait **aucune entrée au registre** — il vient de la
      table (A) de l'audit 0.E (ligne A3), et le lot 1.9.0 n'y a entré que les 62 replis ANONYMES de
      la table (E) plus ceux que les lots 0.D à 1.8 avaient nommés. L'entrée est donc CRÉÉE, et elle
      naît `apres_lecture` : le repli ne se déclenche que sur un silence (`Muet`) ou une indécision
      (`Ambigu`) de la lecture, jamais sur un désaccord avec elle (D14 b).

      **LES COMPTEURS, DANS LES STATS DE COLLECTE.** `killsource.CoupleStats` (`MemeInstant`, `Lus`,
      `Recolles`, `Perdus`, `VictimesBotLues`, `Muet`, `Ambigu`, `Accord`, `Contradiction`) porté par
      `Result.Stats.Couples`, publié en expvar par `killcollector.publishCoupleProvenance` sous
      `killsource_couple_{meme_instant,lu,recolle,muet,ambigu,contradiction}` et
      `killsource_victime_bot_lue`. Les trois compteurs de diagnostic disent POURQUOI il a fallu se
      replier — sans eux, un compte de replis ne désigne aucune correction.

      **LES MUTATIONS, JOUÉES ET RESTAURÉES PAR NOM.** (A) **tueur et victime échangés dans la
      fixture** (`coupleRec(2000, 1, 3)` -> `coupleRec(2000, 3, 1)`) :
      `TestLeCoupleVientDuKillEvent85` ROUGE — « couples = [{2000 C D}], attendu le seul (C, E) ».
      C'est la preuve demandée, et elle n'est possible que parce que le témoin est bâti pour que
      lecture et repli DIVERGENT : sur 198 des 281 kills du corpus ils s'accordent, donc un témoin
      ordinaire resterait vert. (B) **la lecture débranchée en production** (`lireVictime` remplacée
      par `-1, -1, false` dans `resoudreUnKillSansMort`) : QUATRE tests ROUGES, dont le golden
      `TestGoldenMiniBobine`. Les deux arbres restaurés depuis une copie hors dépôt ; suite verte
      après restauration.

      **RÉVISIONS.** **`KillSourceDecoderRev` killsource-2026-09-14 -> `killsource-2026-09-15`**
      *(le brief proposait `killsource-2026-09-16` ; la convention du golden est la DATE DU
      MOUVEMENT, et il a lieu le 15)*. Le ratchet d'empreinte a **rougi de lui-même** au commit du
      décodeur (« LE DECODEUR A CHANGE », `729624c1…` -> `ed7e45ca…`) : preuve par mutation
      naturelle, aucune mutation artificielle nécessaire. **`GrammarRev` grammar-2026-09-15.2 ->
      `grammar-2026-09-15.3`** (l'ensemble haché porte `killsource/`). **`SchemaVersion` : **59, INCHANGÉE**, et
      c'est MESURÉ et non raisonné — voir le bloc « les deux gates de décodage » ci-dessous.

      **LES DEUX GATES DE DÉCODAGE, JOUÉS LE 2026-09-15 SUR SIGNAL DU PILOTE (voie libre).**

      **ÉQUIVALENCE, régime court (10 films).** La classification a été faite AVANT tout `-update`,
      en mode ENFANT (un film par processus, `-child -film <short8> -out <fichier>`), en comparant
      chaque `.tsv` produit à sa référence : **10 films sur 10, UNE SEULE étape bouge, `killsource`**
      — les 52 autres sont identiques à l'octet, **`artifact` compris (longueur ET sha)**, ainsi que
      `killRefs`, `neutralDeaths` et `deaths`. Puis `-update` en DEUX sous-ensembles séquentiels :
      `git diff` des références rend **10 `+killsource` / 10 `-killsource` et rien d'autre**, une
      ligne par fichier. Passe de comparaison : **10/10 identiques**. **CE QUI BOUGE EST LA FORME DE
      L'OBJET OBSERVÉ, PAS SON CONTENU** — `replaybuild.observe("killsource", ksRes)` hache le
      `Result` ENTIER et `Stats.Couples` est un champ neuf ; `killRefs`, `neutralDeaths` et `deaths`
      sont des PROJECTIONS du même `Result`, et elles ne bougent pas d'un bit. C'est mot pour mot le
      motif du lot 1.8 (découverte D4 (1.8)).

      **CORPUS GATE, `--base=9848b7387` sans `--manifest`, 28 cuissons, 14 témoins : SORTIE 0, ZÉRO
      PERTE, ZÉRO GAIN, schéma 59 des deux côtés.** `bcb6d393` 16,2 s · `fb1a1a72` 33,0 s ·
      `d9781168` 24,6 s · `c75f33b8` 14,6 s · `bf15f7ab` 14,0 s · `51ebbc0f` 18,2 s · `084a804d`
      1 min 51 · `0797ce72` 13,1 s · `111fa685` 47,3 s · `e5adf7b2` 39,6 s · `60ae07c4` 25,3 s ·
      `a349fea8` 2 min 36 · `bfecd02b` 19,5 s · `4f77afc1` 1 min 39.

      **ET ZÉRO PERTE NE SUFFISAIT PAS À STATUER `SchemaVersion` : LE BILAN IMPRIMÉ NE PORTE QUE
      `gains` ET `pertes`, JAMAIS `changements`.** `replaydiff.BilanAxe` compte quatre catégories et
      le tableau n'en montre que deux ; `pertesDetail` du rapport JSON ne retient que PERTE et
      DISPARU (D5 (1.9.3) en §4). Sur le seul film dont un couple change — `4f77afc1` — un
      `changement` serait donc resté invisible. **TRANCHÉ PAR COMPARAISON OCTET POUR OCTET DES DEUX
      ARTEFACTS CUITS** (gate rejoué sur manifeste réduit avec `--work-root` et `--keep-work`, puis
      `cmp`) : **10 526 185 octets de chaque côté, sha256
      `62979da9924702d3a826c585ea7bf6a38752ad089164978ff142b7f79660a476` IDENTIQUE, `cmp` muet.**
      `SchemaVersion` reste donc à **59**, aucune chaîne complète, aucun `openapi.yaml` à régénérer
      (`git status` le confirme : ni `api/openapi.yaml` ni `apps/web/` ne sont touchés).

      **POURQUOI LA LIGNE NE BOUGE PAS ALORS QUE LE COUPLE CHANGE, ET C'EST VÉRIFIÉ PAR GREP.**
      L'instant que le recollage fabriquait sortait DÉJÀ en [`OriginBot`] avec le nom du bot : le
      couple fabriqué entrait dans `pairs`, puis `resolveBotDeaths` le reclassait en mort de bot et
      `ghostPairs` le RETIRAIT du dénominateur. Ce que le lot change est donc la CLASSIFICATION
      interne et les dénominateurs `Coverage.ReconstructedPairs` / `GhostPairs` — et ceux-là ne sont
      consommés QUE par la CLI de diagnostic `cmd/killsource` (grep : aucun autre appelant hors du
      paquet), jamais par le document ni par la base. Le gain du lot reste ce qu'il est : la victime
      est désormais NOMMÉE PAR LE FILM au lieu d'être devinée, et la mort du voisin cesse d'être
      consommée à tort.

      **POPULATION DU PARC.** Oracle `data/backups/pre-chaine-2026-09-09/shared_matches_v2.duckdb`
      (lecture seule, `cmd/diag_q`) : **1 384 matchs** portent des lignes de kill, **tous** sous une
      révision antérieure — 751 `killsource-2026-09-05`, 589 `killsource-2026-07-31`, 41
      `sync-kill-feed-2026-08-02`, 3 `highlight-credit-2026-08-01`. **Le backlog n'est donc PAS
      élargi par ce lot** : c'est la MÊME population que celle que le lot 1.8 a inscrite et que la
      clôture M1 doit rejouer (D6 : aucune recuisson depuis). Hors lot.

      **STATUT DES ITEMS DU BRIEF.** (1) `[x]` mesure avant de coder, tableaux collés, 64/372
      rejoués. (2) `[x]` le kill-event décide, le recollage devient un repli nommé et compté, la
      fabrication sur victime bot est supprimée. (3) `[x]` deux mutations jouées et restaurées.
      (4) `[x]` révision montée, ratchet rouge de lui-même, golden `minibobine.golden` régénéré par
      sa porte nommée — **UNE seule ligne change**, attribuée. (5) `[x]` **`SchemaVersion` STATUÉE À 59** par
      les deux gates de décodage joués le même jour, et par la comparaison octet pour octet des
      deux artefacts de `4f77afc1` — le seul film dont un couple change. Aucune chaîne complète,
      aucun `openapi.yaml` régénéré : rien du contrat publié ne bouge.
- [x] 1.9.4 **La carte du film vient du nom de match, plus d'une signature de largeurs.**
      `internal/sync/killcollector/hits.go:151` appelle `DetectFilmWorldRange(dir, path, "")` alors
      que le même collecteur résout le nom de carte à `positions.go:210` et que le paramètre
      `mapNameOverride` existe. Gain : **6 cartes jumelles** (3 paires mesurées, F.0 §6 réserve 2)
      récupèrent leurs distances, aujourd'hui désactivées en silence. S.

      **FAIT le 2026-09-15.**

      **LES CITATIONS ONT DÉRIVÉ, ET LA MESURE A RETOURNÉ LE CHIFFRAGE DU LOT.** `DetectFilmWorldRange`
      n'existe plus (renommée `DetectFilmMapEntry` au lot 1.9.2) et le site est `hits.go:170`. Surtout,
      les « 6 cartes jumelles » sont **68 sur 79** : instrument versionné
      `filmdec/e194_carte_par_nom_mesure_research_test.go`, qui groupe le catalogue par la clé EXACTE
      que la signature comparait (`e.AxisWidths == lay.AxisW`) et rend **cinq classes ambiguës** —
      `15/15/17` **59 cartes**, `17/17/15` 3 (`behemoth`, `fragmentation`, `launch site`), `14/13/15` 2
      (`prism`, `scarr`), `15/15/15` 2 (`catalyst`, `deadlock`), `18/18/17` 2 (`chasm`, `illusion`).
      **Seules 11 cartes sur 79 avaient une signature unique.** La cause est structurelle et ferme le
      sujet pour tout le chantier : `W = min(26, ceilLog2(ceil(60*étendue)))` quantifie l'étendue au
      facteur 2, et les cartes d'Infinite tombent dans la même case. D2 (1.9.4) en §4.

      **ET LA SIGNATURE NE SE TAISAIT PAS SUR LIVE FIRE : ELLE MENTAIT.** Mesure (b), 17 films (les
      14 témoins du corpus gate et les 8 builds), tableau collé en §5 : **2 accords** (`bcb6d393`
      Cliffhanger, `50247b26` Oasis — les deux seules cartes du corpus à signature unique),
      **13 ambiguës**, **2 DÉSACCORDS**. Les deux désaccords sont les films Live Fire : l'auto-détection
      impute le bit d'index de région à l'axe X et rend `13/12/11`, qui est **exactement** la signature
      d'`aquarius` — `len(hits)` valait **1**, pas 0. `DetectFilmMapEntry` rendait donc l'AABB d'une
      AUTRE carte (Aquarius 77,8 x 46,2 x 18,1 m contre Live Fire 63,2 x 63,8 x 22,9 m, origines
      différentes) et les distances y étaient **fausses**, pas absentes. **D2 (1.9.2) est corrigée** —
      elle concluait « aucune entrée trouvée, distances déjà désactivées ». D1 (1.9.4) en §4.

      **CE QUE LE LOT CHANGE.** (a) La carte du match se résout en **UN SEUL endroit pour les deux
      passes** du collecteur : `killcollector/map_identity.go` (fichier neuf, 100 L), qui sépare enfin
      les deux échecs que `resolveMapBounds` confondait — `ErrSansNomDeCarte` (la base ne nomme aucune
      carte) et `filmdec.ErrUnknownMapBounds` (le nom est connu, la carte n'est pas au catalogue, D-4).
      `positions.go` passe de **508 à 494 lignes** au passage : la dette baisse. (b)
      `hits.resolveHitDistanceFunc` lit la carte au NOM (`entreeDeCarteDesTouches`) avec **trois sorties
      et trois compteurs** — `killsource_hits_carte_non_cablee`, `killsource_hits_matchs_sans_nom_de_carte`,
      `killsource_hits_cartes_hors_catalogue` —, jamais une carte « au plus proche ». (c)
      `filmdec.DetectFilmMapEntry` est **SUPPRIMÉE** ; `DetectI0Layout` n'a plus **AUCUN appelant de
      production** (elle garde ~45 appelants de test : c'est l'enveloppe D2 des instruments de
      recherche, et elle reste sous deux gardes — l'allowlist de
      `archlint/no_recomputed_film_context_test.go`, dont l'entrée est retirée dans ce commit, et la
      liste des enveloppes interdites de `archlint/no_film_reread_test.go`, où elle figure déjà).
      (d) `ConfigureFilmAccuracy` **perd son second paramètre** : le collecteur n'a plus qu'UN catalogue
      de bornes, celui de `WithPositionCapture`, là où il en portait deux configurations.

      **LA SIGNATURE N'EST PAS DEVENUE UN REPLI, ET C'EST UN ÉCART ASSUMÉ AU BRIEF (D3 (1.9.4) en §4).**
      Le brief demandait de la rétrograder en repli nommé (D14) pour le cas où le nom manque. L'inscrire
      au registre aurait voulu dire garder une mécanique mesurée **juste 2 fois sur 17**, et **fausse**
      sur Live Fire : exactement le « repli bancal qui se déclenche à tort et corrompt un fait que la
      lecture aurait donné juste » que D14 (d) nomme. Le nom manquant est donc une **erreur typée
      comptée**, et la dégradation qui suit est celle qui existait déjà au registre
      (`repli_distances_de_touche_desactivees`). **ACCEPTÉ PAR LE PILOTE le 2026-09-15** — « une mécanique juste 2 fois sur 17 et fausse sur Live Fire est le repli bancal que D14 (d) interdit, ta mesure tranche » ; le `[!]` est CONSERVÉ avec sa justification et porté à l'utilisateur.

      **LE REGISTRE DES REPLIS : 96 ENTRÉES, INCHANGÉ, ET LE RATCHET DES SIX NE DESCEND PAS.** Aucune
      entrée n'entre (la signature est supprimée, pas rétrogradée), aucune ne sort. **`NbDevantLaLecture`
      reste à 6, et ce n'est pas un oubli** : relevé sur pièces, les six sont `repli_largeur_absolue_uniforme`,
      `repli_chunk_du_pied_par_argmax`, `repli_type_de_chunk_perdu_du_manifeste`,
      `repli_manches_contigues_decretees`, `repli_vie_coupee_au_trou_de_replication` et
      `repli_fin_de_vie_vehicule_par_recensement` — cibles des lots 1.9.8, 1.9.10, 1.9.11 et 1.9.13,
      aucune de celui-ci. TROIS entrées sont mises à jour, et la constante partagée `lot194` est
      **supprimée** parce que les quatre cibles qui la citaient ont divergé :
      `repli_i0_porte_et_region_par_defaut` (moitié faite — `DetectI0Layout` est hors production,
      restent les deux appelants de `DetectI0LayoutOf` -> lot 3.x),
      `repli_distances_de_touche_desactivees` (critère « cartes jumelles » TENU ; la dégradation reste,
      nouvelle cible = le lot qui rallume la précision par arme) et `repli_carte_premier_nom_resolu`
      (site unique désormais ; NON retiré — il porte l'arbitrage entre noms candidats, une question du
      REGISTRE DES MATCHS, hors famille 1.9).

      **LES MUTATIONS, JOUÉES ET RESTAURÉES PAR NOM** (`killcollector/hits_carte_par_nom_test.go`) :
      (A) la carte témoin remplacée par sa jumelle des DEUX côtés -> deux tests rouges, dont le garde du
      présupposé en premier ; (B) **LA FIXTURE seule faussée** — le collecteur reçoit `Behemoth` quand
      l'attente reste `Fragmentation` -> rouge sur les bornes (`[-621,99 510,37]` rendu contre
      `[-1136,27 543,44]` attendu). **Avant ce lot, la mutation (B) n'avait aucune prise** : la passe des
      touches ne recevait AUCUN nom de carte. Un test garde le présupposé lui-même
      (`TestJumellesDuCatalogueSontIndistinguablesParSignature`) pour qu'un catalogue qui changerait ne
      laisse pas la mutation prouver le vide.

      **RÉVISIONS.** `GrammarRev` **grammar-2026-09-15.7 -> .8** (golden régénéré, ligne d'historique
      écrite) : aucun bit lu ne change, l'empreinte hache des octets de source — ce qui change est QUELLE
      carte, donc quelles bornes et quel découpage, s'applique à un film. **`KillSourceDecoderRev` NE
      MONTE PAS** (`film/killsource/` n'a pas bougé, son ratchet d'empreinte fait foi). **`IsolationDecoderRev`
      NE MONTE PAS** : aucune sortie PERSISTÉE ne change — `kill_positions` / `kill_openings` viennent de
      la passe des POSITIONS, que ce lot ne touche pas ; seule celle des TOUCHES change, et elle est
      éteinte (**0 match** concerné). **`SchemaVersion` : 59, inchangée** (le chemin de cuisson n'appelait
      pas cette fonction) — **ÉTABLI, PAS SUPPOSÉ** : équivalence **10/10 identiques** et corpus gate **14 témoins à 0 gain / 0 perte / 0 CHANGEMENT**, schéma 59 des deux côtés, la colonne des changements lue au rapport `--json` (D5 (1.9.3)). Aucun contrat publié ne change (`git status`
      sur `api/openapi.yaml` et `apps/web/` : vide).

      **LE GAIN EFFECTIF EN PRODUCTION, DIT TEL QUEL (D5 (1.9.4) en §4).** D1 (1.9.2) est re-vérifiée :
      `"match.weapon.accuracy" = "not_exposed"` et `ConfigureFilmAccuracy` n'a aucun appelant de
      production — **les distances de touche ne sont calculées pour aucun match**. Ce que le lot
      apporte aujourd'hui : une auto-détection de moins dans le décodeur, un site de résolution de carte
      au lieu de deux, une erreur typée comptée sur carte inconnue, `DetectI0Layout` hors production sous
      ratchet, et 14 lignes de dette en moins. Ce qu'il apporterait si la passe était rallumée, mesuré :
      15 des 17 films témoins passent de « pas de distance » ou « distance fausse » à « distance juste »,
      et 68 cartes sur 79 sortent de l'ambiguïté.

      **LES DEUX GATES DE DÉCODAGE, JOUÉS SUR VOIE LIBRE DU PILOTE (2026-09-15).** Équivalence,
      régime court : **10 identiques, 0 différent, 0 écarté, 0 échec**, en deux moitiés
      séquentielles de 5 films (3 min 33 + 2 min 11), **aucun `-update`** — et aucun n'était
      nécessaire, ce qui est l'attendu : le chemin de CUISSON imposait déjà le catalogue depuis le
      2026-09-03 et n'appelait pas `DetectFilmMapEntry`, donc retirer l'identification par signature
      ne pouvait pas déplacer un octet cuit. Corpus gate `--base=d71282e09` (sans `--manifest`,
      28 cuissons, ~21 min) : **sortie 0, ZÉRO PERTE, ZÉRO GAIN, ZÉRO CHANGEMENT** sur les
      14 témoins, schéma 59 des deux côtés. La colonne des CHANGEMENTS a été lue au rapport `--json`
      et non au seul tableau (consigne du pilote, née de D5 (1.9.3)) : **14 lignes à
      `"changements": 0`**. Relu sur pièces au passage, `ligneRapport.aUnePerte()`
      (`cmd/replay-corpus-gate/report.go:55`) compte déjà `Changements > 0` comme un refus — le gate
      BLOQUAIT donc dessus ; le `--json` le chiffre au lieu de seulement le détecter. **Un seul
      décodage à la fois** du début à la fin (verrou `filmproc.AcquireSolo` visible au journal,
      priorité CPU `below_normal`), aucune écriture dans le parc, racine de travail jetable
      supprimée, 0 processus résiduel.

      **RÈGLE NEUVE DU CHANTIER, APPLIQUÉE AVANT CLÔTURE (pilote, 2026-09-15).** Tout
      `*_research_test.go` porte désormais `//go:build research` — le job de couverture CI a dépassé
      ses 600 s sur `filmdec` à cause des instruments, qui tournent sous `go test -tags research`.
      Le tag est posé sur `e194_carte_par_nom_mesure_research_test.go` ; `go vet` rend **0 diagnostic
      dans les DEUX builds** (`./internal/sync/killcollector/` et `./internal/games/halo_infinite/film/...`,
      sans tag puis avec), `archlint` est vert des deux côtés, et l'effet est prouvé dans les deux
      sens (« no tests to run » sans tag, `ok` avec). **CE QUE LE TAG COÛTE EST DIT** : la mesure (c)
      ne demandait aucun film et coûtait 0,06 s, elle quitte quand même le run par défaut — un tag
      posé « sauf exceptions » ne se tient pas. **CE QUI LA REMPLACE EN CI** :
      `TestJumellesDuCatalogueSontIndistinguablesParSignature` dans `sync/killcollector/`, qui garde
      sur le catalogue versionné que les deux cartes du témoin de mutation partagent bien leur
      signature et non leurs bornes. Sans lui, le tag aurait fait disparaître du run par défaut la
      seule vérification que la mutation du lot a un sens.

      **STATUT DES ITEMS DU BRIEF.** (1) `[x]` mesure avant de coder — (a) grep des appelants, (b) 17
      films, (c) classes du catalogue, les trois collées en §5. (2) `[x]` pour « le nom décide », l'erreur
      typée sur carte inconnue et le sort de `DetectI0Layout` ; **`[!]` pour « la signature devient un
      repli déclaré »** — non traité DÉLIBÉRÉMENT, justification mesurée en D3 (1.9.4), **ACCEPTÉE par le
      pilote le 2026-09-15** — le `[!]` est CONSERVÉ avec sa justification et porté à l'utilisateur ; **`[~]` pour « contrôle accord / contradiction compté »** — oracle de test, même motif que
      D3 (1.9.2), cf. D4 (1.9.4). (3) `[x]` les distances de touche non rallumées, ce que le lot
      changerait mesuré et consigné en §4. (4) `[x]` mutation par fixture faussée, jouée des deux côtés.
      (5) `[x]` révisions statuées une par une, population chiffrée (0 match), et **établies par les deux gates** :
      équivalence 10/10 identiques, corpus gate 14 témoins à 0 gain / 0 perte / 0 changement.

- [!] 1.9.5 **Le porteur du crâne lu au canal des armes tenues.** REPORTÉ À M2 (décision
      utilisateur du 2026-09-16, resserrement de la famille 1.9 : M1 garde les items à effet visible
      dans le rejeu, 1.9.7 et 1.9.9 à 1.9.14). Ses deux replis (`repli_crane_porteur_sans_vie_nommee`,
      `repli_porteur_anonyme_sans_fin_par_mort`) ne sont PAS câblés au registre : leur fréquence sur le
      corpus n'est pas mesurée, elle le sera au câblage (M2) et décidera du rang du lot.
      `internal/games/halo_infinite/film/replay/skull_carries.go:390` infère le porteur des tics de
      score (trou > 3 s) alors que le crâne voyage dans le canal des armes tenues (famille
      `0x0017592c`, `replay/held_object_carry.go:15`) et que `BuildHeldObjectCarry` est PORTÉ mais
      n'a qu'UN appelant de production, `replay/bomb_carries.go:142`. Les tics deviennent le repli
      compté. Gain non chiffré. S.
- [!] 1.9.6 **Le drapeau qui rentre pris dans `ev.flag`, déjà nommé en amont.** REPORTÉ À M2 (décision
      utilisateur du 2026-09-16, même resserrement que 1.9.5). Repli `repli_drapeau_seul_en_jeu` non
      câblé : fréquence non mesurée ; `ambiguousReturns` est publié et vaut oracle le jour venu.
      `internal/games/halo_infinite/film/replay/flag_carries_lives.go:267` cherche « le seul drapeau
      au sol » alors que `ev.flag` est posé par `flag_carries_home.go:82-98` et ne sert qu'à un
      court-circuit (`:264`). Gain : les `ambiguousReturns` (compteur publié) que `ev.flag` tranche. S.
- [ ] 1.9.7 **Dead-state et kill-feed appariés par l'identité de paquet, plus par 2,5 s.**
      `internal/games/halo_infinite/film/killsource/options.go:114` (`tolMS = 2500`, justifiée par la
      comparabilité et non par une mesure) employée à `match.go:30`, `:45`, `:103`, `:143` ; les deux
      structures portent `(chunk, pidx)` (`killsource/scan.go:49`, `assist.go:169`) mais `Kill` ne le
      transporte pas (`match.go:186-193`). La fenêtre devient le repli compté. M.
- [!] 1.9.8 **Le chunk du pied pris au type du manifeste, plus par argmax de kills.** REPORTÉ À M2
      (décision utilisateur du 2026-09-16, même resserrement que 1.9.5). Ses deux replis
      (`repli_chunk_du_pied_par_argmax`, `repli_type_de_chunk_perdu_du_manifeste`) sont
      `devant_la_lecture` au ratchet des six : le ratchet reste à 6 tant que ce lot n'est pas joué,
      et la clôture M1 le consigne tel quel.
      `internal/games/halo_infinite/film/killsource/feed.go:82` ; le type est porté par
      `filmsource.Film.Meta()` (`analysis/filmsource/film.go:41`) et déjà lu par ce patron
      (`objectiveevents/extract.go:144`), mais `killsource/chunks.go:78` perd `Meta()`.
      RÉSERVE : le manifeste est un fichier EXTERNE — l'argmax reste en repli COMPTÉ. S.

- [ ] 1.9.9 **Les tourelles automatiques bannies nommées, et dessinées comme éléments de carte.**
      Décision utilisateur du 2026-09-14 (amendée le même jour) : le châssis `0x038df01a` (banque
      `sb_003_lvl_moments_ge_shared_autoturret_banished`, 9 vies immobiles sur `bfecd02b`, un
      spawn, fenêtre = le match entier) est un objet de carte qui interdit la sortie de la zone de
      jeu, PAS un véhicule jouable ; le parc d'assets véhicules est COMPLET, un châssis absent de la
      table est un mismatch à nommer, jamais un véhicule manquant. L'utilisateur VEUT les voir
      dessinées (« ce sont des éléments de la map ») mais ne possède aucun asset pour elles.
      L'entrée « valeur inconnue = famille vide, marqueur neutre » de `vehicle_families.go` est un
      repli anonyme (D14) : le châssis entre en table sous une famille NOMMÉE `tourelle_auto_bannie`
      (élément de carte, non jouable, immobile), publiée avec son libellé FR/EN et dessinée par un
      pictogramme de tourelle DÉDIÉ (pas le marqueur neutre) tant qu'aucun asset n'est fourni ;
      le jour où l'utilisateur fournit l'asset, seule la table d'assets change. Le refus du
      marqueur neutre reste compté pour tout AUTRE châssis inconnu. Témoin : `bfecd02b`, 9 tourelles
      nommées visibles à leur position. S.
- [x] 1.9.10 **La fin de vie d'un véhicule lue au dead-state écrit, plus inférée.** Décision
      utilisateur du 2026-09-14 : un véhicule est vivant ou détruit et le film l'écrit (`ti=40`,
      lisible depuis le 2026-09-05 sur `wt/vehicule-deadstate`, non fusionnée ; piège : le filtre
      `DesyncAt == -1` jetait des morts lues) ; un respawn (sur socle ou aux coordonnées monde) est
      une nouvelle vie. `VehicleTrack.End` ne prend qu'une valeur (`unknown`) et la fin est inférée
      d'une borne de recensement (« 5 s après le dernier échantillon ») : c'est le repli qui a
      effacé le `ghost` slot 777 de `bfecd02b` à 287,4 s alors que l'utilisateur le PILOTE pendant
      de longues minutes. Conversion : `End` porte l'instant du dead-state ; l'inférence devient le
      repli compté, puis retiré. TEST D'ACCEPTATION sur `bfecd02b` : le ghost reste dessiné tant que
      le film ne l'écrit pas détruit ; si ses positions cessent à 282 s SANS dead-state, le défaut
      est la LECTURE des échantillons d'un véhicule occupé, à instruire dans le même lot (sur
      pièces : `replay/vehicle_*.go`). ORACLES donnés par l'utilisateur (2026-09-14) pour
      vérifier les vies de véhicule lues : (i) les kills faits DEPUIS un véhicule (kill feed, arme
      = le véhicule) bornent la vie par en bas : le véhicule est vivant et occupé à ces instants ;
      (ii) la médaille « véhicule détruit » et ses variantes qui NOMMENT le véhicule, et les
      personal score awards, datent une destruction ; (iii) inconsistances connues, à ne pas
      lire comme des défauts : un véhicule peut vivre sans faire de kill, un joueur peut quitter
      le véhicule avant qu'il explose. Méthode : choisir au corpus les matchs où ces trois sources
      sont assez denses, et confronter chaque vie lue (début, fin par dead-state) aux instants
      qu'elles donnent ; tout kill depuis un véhicule APRÈS sa fin lue, ou toute destruction
      datée SANS dead-state lu, est un constat nommé. M.

      **FAIT le 2026-09-16 (branche `feat/decfilm-1910`), GATES DE DÉCODAGE COMPRIS.**

      **LA MESURE AVANT DE CODER, SANS DÉCODER.** 295 vies de véhicule sur 20 artefacts du
      parc : le repli `repli_fin_de_vie_vehicule_par_recensement` se déclenchait sur **88**
      d'entre elles, et `goneByUS == 0` veut dire EXACTEMENT « la dernière image-clé recense
      encore cette vie » — le repli ne frappait donc QUE des véhicules qui finissent avec le
      film. Critère de retrait tenu par construction, pas par échantillon.
      **Oracle (i)** (kills depuis un véhicule, `source_tag` ∈ les 91 tags de classe `VEHICULE`
      de `damagetag`, le tag NOMMANT le châssis) : 177 kills croisés aux vies publiées — 98 dans
      une vie de la bonne famille, 0 dans un trou, **7 APRÈS la dernière fin lue** (D2), 72 de
      familles jamais publiées (D3). **Oracle (ii)** : le dépôt ne date AUCUNE destruction hors
      du film (D4) — `medals_earned` n'est pas horodatée, `VehicleDestroys` n'est persisté nulle
      part ; seules `Ballista` / `Dogfight` / `Boom Block` en impliquent une, 18 sur 7 films.

      **LA CONVERSION.** `filmdec.ScanObjectDeaths` (`object_deaths*.go`) porte LA MARCHE en
      production — timeline chronologique des images-clés, localisateur de liste d'événements,
      huit vues, snapshot/restore — et lit `object-dead-state` là où aucun balayage ancré ne
      l'atteint (mesure V10 : l'ancre accepte 153 535 à 240 115 records `ti=35` et en rend UN
      portant `i11`, contre 47 à 66 pour la marche). **Le verrou levé est le filtre
      `DesyncAt == -1`** : sur `ti=40`, 65 des 69 records qui DÉCLARENT le dead-state rompent à
      `i30`..`i36`, tous APRÈS `i11` — la tête était lue, seule la queue était inconnue. Règle
      posée : accepter si `DesyncAt == -1` OU `DesyncAt > index(dead-state)`, les deux qualités
      comptées à part. **D8 le prouve** : sur `4f77afc1` les 14 morts lues sont TOUTES à queue
      désynchronisée — sans la règle, ce film rendrait zéro.

      **CE QUE LE DOCUMENT PUBLIE.** `VehicleTrack.End` prend trois valeurs, toutes des faits :
      `destroyed` (daté par `TEnd`), `film_end`, `unknown` ; huit compteurs ventilent la lecture,
      ce qu'elle décide et ce qu'elle contredit. **Le repli est SUPPRIMÉ, pas rétrogradé** — la
      fenêtre d'une vie que le recensement ne ferme jamais n'a plus de borne haute, l'entrée sort
      du registre avec son ancre (D14 d), `ratchetDevantLaLecture` **6 -> 5**.

      **LA CALIBRATION DU CADRE, REPRISE APRÈS VÉRIFICATION DU PILOTE.** Le premier jet balayait
      DEUX dimensions, dont `PacketPreambleBits` — une propriété du FORMAT prouvée par trois
      témoins concordants (`frame_records.go:58-97`). Corrigé : l'amorce ne se balaye plus
      (18 cadres candidats -> 6), le critère est la LOCALISATION seule (le rendement en records
      propres est réfuté par V13), et un **garde-fou de domination** repris de
      `killsource/calibrate.go` conserve le cadre par défaut quand le profil est plat — cas
      NOMMÉ, COMPTÉ et daté au registre (`repli_cadre_de_marche_par_defaut_conserve`,
      `non_resolu` / `apres_lecture`, compteur câblé, cible M3).

      **TEST D'ACCEPTATION SUR `bfecd02b` — LE FILM ÉCRIT LA DESTRUCTION DU GHOST 777.**
      `end = destroyed`, `tEnd = 2740` (**274,0 s**) ; dernier épisode d'occupation jusqu'à
      270,1 s, positions répliquées jusqu'à 282,1 s (8,1 s d'épave, D7), `t1max` resserré de
      2874 à 2821. Sur ce témoin **`unknown` tombe à ZÉRO** (1 `destroyed`, 10 `film_end`). Le
      test porte sur l'artefact CUIT et **discrimine** : rouge sur l'artefact de la base, vert
      sur celui du HEAD. La prémisse de cet item était inexacte et D1 la corrige : cette fin
      n'était pas le repli mais une preuve d'absence écrite.

      **CE QUE LE LOT NE FERME PAS, ET LE COMPTE LE DIT** : 92 vies sur 97 restent `unknown` sur
      `4f77afc1`, les kills tardifs n'y sont pas couverts (D9), et 20 morts restent orphelines
      (D6). M.
- [ ] 1.9.11 **Le désignateur de manche lu tel que le film l'écrit, la garde `contiguousRounds`
      retirée.** Décision utilisateur du 2026-09-14 (« le film porte le compteur de manche ; oui,
      tu peux le faire »). Sur `fb1a1a72` (CTF:Arena, 814 s > 720 s de temps réglementaire), 148
      records statborg portent le désignateur `2` (0.D.1 bis) ; la garde `contiguousRounds` publie
      1 manche parce que la manche 1 est absente : c'est le repli nommé qui jette ce que le film
      écrit. Session bornée : lister au corpus les matchs dont la durée dépasse le temps
      réglementaire de leur mode, lire leur désignateur record par record, établir ce que vaut `2`
      (manche, prolongation) par mesure ou chez l'écrivain ; puis publier le désignateur lu et
      retirer la garde (`coverage.score.rounds` repasse à la valeur écrite). Hypothèse de
      l'utilisateur (2026-09-14) : « ça ressemble à une prolongation ; si le temps réglementaire se
      finit sur une égalité ça peut arriver, mais je ne sais pas si c'est le seul critère ». À
      confronter aux deux sens : tout match à désignateur `2` a-t-il un score à égalité à la fin
      du temps réglementaire (piste de score du film, pas la feuille) ? tout match à égalité à cet
      instant porte-t-il un désignateur `2` ? Les contre-exemples, s'il y en a, nomment l'autre
      critère. M.
- [~] 1.9.12 **La vie d'un seul échantillon publiée.** TRAITÉE AU LOT 1.6.5 (2026-09-14), dans la
      même montée de schéma que la table du film : `DefaultMinPoints = 1`, compteur à 0 sur les
      8 builds, 20 vies publiées, oracle mesuré (1 mort écrite, 5 fins de film, 14 orphelines
      consignées en §4 comme défaut de lecture). Rien ne reste de cet item. Texte d'origine :
      Résidu de 0.D.4 et compteur de 1.0.4 :
      `DefaultMinPoints = 2` refuse toute vie d'un seul échantillon (une position écrite par le
      film pour un joueur à un instant) ; toutes les vies refusées en portent exactement un, 0 à 6
      par film sur les 8 builds. DÉCISION utilisateur du 2026-09-14 : « si le film le dit, on
      publie » -> `DefaultMinPoints = 1`, le compteur de refus reste (il doit tomber à 0), schéma
      monté avec le lot qui change le contenu cuit. ORACLE donné par l'utilisateur : l'heuristique
      ignorait les morts à l'apparition (spawn kill), or toutes les morts sont enregistrées ; pour
      chaque vie d'un échantillon publiée, une mort écrite (kill feed / dead-state) à cet instant
      pour ce joueur, OU la dernière image avant une fin de manche, doit exister — mesurer les deux
      cas sur les 20 vies des 8 builds et consigner tout orphelin (une vie d'un échantillon sans
      mort ni fin de manche est un défaut de lecture, pas un cas à filtrer). S.

- [x] 1.9.13 **Une vie finit à une mort ÉCRITE, plus à un trou de réplication.**
      **FAIT le 2026-09-15.** La découpe des vies se LIT désormais (`replay/lives_decoupe.go`,
      `decouperAuxFaitsEcrits`) : `buildLifeSpans` n'est plus qu'un ÉCHAFAUDAGE de séjours de
      réplication — la grille sur laquelle `bestDeathOffset` mesure le calage, rien de plus — et
      toute coupure qu'aucun fait du film ne justifie est REFERMÉE, le trou devenant une LACUNE
      de la même vie. Trois lectures ferment une vie, dans cet ordre : **la mort ÉCRITE du joueur
      de cette vie** dans la fenêtre du trou (le lien vient du registre d'identité 1.6 — et
      l'identité fait partie du critère : l'appariement 1:1 du pont est glouton sur le TEMPS
      SEUL, il peut poser sur une fin de séjour la mort d'un AUTRE joueur, ce qui laissait
      5 coupures sans mort du joueur concerné) ; **l'apparition ÉCRITE d'un corps** (un record de
      création de bipède dans le trou — c'est le signal du slot RECYCLÉ, sans lequel deux
      occupants fusionneraient) ; **la fin de manche**
      (`objectiveevents.RoundBounds.Starts`, accesseur neuf sur des bornes déjà mesurées).
      **MESURE AVANT / APRÈS, les 8 builds** (`replay/decoupe_des_vies_mesure_test.go`, neuf et
      permanent) : **212 coupures → 4**, dont mort écrite 4 → 4, apparition 0 → 0, fin de manche
      0 → 0, **RIEN 208 → 0**. Les 208 sont devenues des lacunes (6 + 33 + 64 + 8 + 55 + 1 + 38
      + 3 = 208, table par build dans les goldens d'assemblage). L'oracle des vies d'un seul
      échantillon suit : **20 vies → 6**, 1 mort → 2, 5 fins de film → 4, **14 ORPHELINES → 0**.
      **Champs publiés (optionnels) :** `coverage.tracks.gaps` / `.gapMs` et `Point.G` — la DURÉE
      en ms de la lacune qui précède le point, portée par celui qui ROUVRE la piste, pour que le
      client n'interpole pas au travers. **`SchemaVersion` NON touché** (règle de la vague) ;
      `GrammarRev` `.7` → `.8` (faux positif d'empreinte : l'accesseur ajouté à
      `objectiveevents`, aucun octet lu autrement) ; `KillSourceDecoderRev` inchangé.
      **Le repli qui reste :** `repli_vie_coupee_au_trou_de_replication`, passé
      `devant_la_lecture` → **`apres_lecture`** (condition `film_muet` : le film n'écrit AUCUNE
      mort de ce joueur, ou le registre ne rend aucune vie), compteur CÂBLÉ, **0 déclenchement sur
      les 8 builds** — **LE RATCHET DES SIX DESCEND À 5**.
      **Gain collatéral mesuré** (goldens) : les bords élargis de `repli_rang_capacite_vie_elargie`
      disparaissent avec les coupures — impulsions de capacité `sans identite` 4 → 3 sur
      `000d5950`, 2 → 0 sur `11de8353`, charges 20 → 13 et 8 → 1 ; `vies nommées` garde son
      NUMÉRATEUR (90, 187, 150, 161) et perd son dénominateur (105 → 99, 245 → 212, 246 → 182,
      177 → 169) : la conversion n'a retiré aucune vie nommée, elle a recollé des morceaux.
      **GATES DE DÉCODAGE (voie libre du pilote, 2026-09-15).** Équivalence, régime court : 10/10 différents AVANT `-update`, et **une seule étape des 53 — `artifact`** ; les 52 étapes de décodage sont identiques à l'octet sur les dix films. Le delta d'octets s'explique film par film par le nombre de lacunes (~195 o par vie qui ne se coupe plus) ; le seul film qui MONTE est `50247b26` (+79 o), dont le registre ne rend AUCUNE vie — le repli y reproduit l'ancienne découpe, et les +79 o sont `gaps`, `gapMs` et l'entrée neuve de `coverage.fallbacks[]`. Re-figeage puis passe : **10/10 identiques**, `git diff` des références = 1 insertion + 1 suppression par fichier, toutes `artifact`. Corpus gate, 14 témoins, schéma 59 des deux côtés : **104 gains, 491 « pertes », 2 changements** — les 491 CLASSÉES sans exception (451 « moins de vies », 38 compteurs de DÉFAUT qui tombent, 2 `livesNamed`), **aucune métrique de sens `disparu`**, et **AUCUNE perte sur `publishedPoints`, `bounds`, `shots`, `grenades`, `kills`, `objectives`, `flagCarries`, `scoreTimeline`, `projectiles`, `roster`** : la conversion recolle, elle ne retire rien. **335 vies fusionnées sur les 14 témoins**, `unnamedLives` en baisse sur les 13 qui bougent (`084a804d` 154 -> 66). Le 14e, `a349fea8`, sort **0 perte / 0 changement / 1 gain** : c'est le film sans section d'identification, le repli y garde le seuil — la signature attendue.
      Texte d'origine : Né de l'oracle
      utilisateur du 1.6.5 (D1 (1.6)) : sur les 20 vies d'un seul échantillon des 8 builds,
      14 sont ORPHELINES — ni mort écrite, ni fin de manche, ni fin de film — et toutes sont
      fermées par la règle de `build.go` qui ouvre une nouvelle vie dès qu'un trou de positions
      dépasse `lifeGapUS` (5 s), le même seuil que `buildLifeSpans`. C'est une heuristique au
      sens de D13 : la grammaire dit qu'une vie commence à une apparition et finit à une mort
      écrite (kill feed / dead-state), à une fin de manche ou à la fin du film ; un trou de
      réplication n'est pas une mort. Conversion : les tracks se découpent aux morts écrites du
      joueur (registre d'identité 1.6 pour le lien), le trou devient une LACUNE de la même vie
      (comptée : `coverage.tracks.gaps`), `lifeGapUS` reste le repli NOMMÉ et compté des seuls
      joueurs sans mort écrite ; l'instrument `vies_un_echantillon_test.go` doit rendre 0 orpheline
      sur les 8 builds. Tranche au passage D4 (1.6) (les 4 vies sans nom que le seuil cachait : des
      fragments de vies nommées) et D5 (1.6) (les bornes de scène élargies par un point isolé, à
      re-mesurer après conversion : si le point à −216 m de `084a804d` subsiste, c'est un fait du
      film à instruire, pas à filtrer). M.

- [ ] 1.9.14 **Le roster à l'instant T, c'est les occupants ; le remplaçant prend le siège du
      partant.** Constat utilisateur du 2026-09-15 sur le schéma 54 : le rejeu affiche tout le
      roster du match tout le temps, les joueurs arrivés en cours de partie paraissent « sans
      équipe » et ne remplacent pas réellement les partants dans leur siège ; « on n'a pas de
      raison d'afficher les joueurs qui ne jouent pas à l'instant T ». Deux causes distinctes :
      (a) l'équipe des remplaçants — réglée par le lot 1.7 (désignateur du film pour TOUTE entité
      ti=9, remplaçants compris, V4) ; (b) le siège — la mesure ajoutée au 1.7 (« D-remplaçants
      (1.7) ») dit si le film DONNE le siège directement (le remplaçant reprend l'index du
      partant : « l'index c'est l'index », remplacement direct) ou non (alors l'appariement
      ordinal par équipe du modèle des sièges du 02/09 reste le repli NOMMÉ et compté). Conversion :
      le document publie, par joueur, ses intervalles de présence (déjà portés par les vies) et
      son siège (index de film) ; la RÈGLE DE LECTURE (web, match-replay) affiche à l'instant T
      les seuls occupants présents, un siège = une fiche, le remplaçant dans la fiche du partant ;
      aucune fiche pour un joueur absent à T. Témoins : `e5adf7b2` (5 remplaçants), `bcb6d393`,
      `a521164d`, `11de8353`. M (Go S + web S).

Arbitrage du pilote (2026-09-13) : l'item 1.9.0 ci-dessus EST le registre des replis proposé par
l'audit ; il entre les **62 replis anonymes** de la table (E) du registre 0.E, et ses 9 replis à
défaut déjà mesuré sont listés dans son journal.

**Hors famille 1.9, routés par le lot 0.E** : `replay/projectiles.go:106` — **6,0 % des trajectoires
du parc** (947 sur 15 735) sont coupées par un garde-fou qui compense une faute de déquantification
de `filmdec` ; c'est une CAUSE à corriger (lot B-bis), pas une conversion.
`replaybuild/zones.go:141` — porter `GameVariantCategory` dans `port.MatchFacts` (la plus petite
correction du périmètre, aucun décodage).

**Clôture M1** (GO utilisateur donné le 2026-09-14, V9) : 1. revue adversariale de jalon (V8)
sur `783ae680d..<HEAD M1>` — fan-out de relecteurs aveugles, un par lentille : grammaire et
replis nommés (D10/D13/D14), entrées tronquées et paniques, textes et chiffres rejoués, ce que les
tests ne couvrent pas (L6) ; deux rondes au plus, P0/P1 corrigés, P2 consignés ; 2. régime
complet (équivalence 20 films + corpus gate 13 témoins à `--base 783ae680d`) — PERTES ATTENDUES
DÉJÀ CLASSÉES, à retrouver telles quelles et pas d'autres : `c75f33b8` `coverage.bombArmings.reads`
1 169 -> 1 148 et `.rises` 94 -> 73 (lot 1.4, cadre d'état complet : l'ancien cadre produisait
des lectures isolées sur des records mal cadrés ; calque publié `bombArmings` inchangé ; rejoué
par le pilote sur manifeste réduit le 2026-09-14) ; référence d'équivalence `50247b26` re-figée
au lot 1.4 (étape `killsource`, chaîne de diagnostic du lot 1.3, D2 (1.3) / D4 (1.4)) ; le gate de
jalon se joue avec `--json <fichier>` et la catégorie `changements` de `replaydiff.BilanAxe` est
LUE et classée témoin par témoin comme les pertes (D5 (1.9.3), 2026-09-16 : le bilan imprimé ne
montre que gains / pertes, une valeur publiée qui bouge sans gain ni perte sortirait « 0 / 0 ») ;
durcissement du gate (changements dans le bilan et dans le code de sortie) au jalon M2 ; 3. fusion dans
`feat/v75` (V3 : fetch + merge + gates locaux, fenêtre de 5 min demandée aux sessions du
checkout principal, push, signal de fin) ; 4. recuisson du parc + backlog killsource (tag git du
binaire précédent, artefacts précédents conservés jusqu'à validation du corpus gate, architecture
§10) ; 5. corpus d'équivalence re-figé UNE fois (`-update` sur tout le corpus, consigné) : c'est
l'oracle de M2.
**LA LECTURE DES `changements` FAIT PARTIE DU RÉGIME COMPLET** (ajoutée le 2026-09-16, lot
1.9.1 bis) : elle passe désormais par le STATUT du corpus gate — un témoin qui porte un
changement sort `PERTE` — et non plus par une inspection du seul `--json`. Avant ce lot, la
catégorie était jetée par `bilanDepuisRapport` et aucun des deux chemins ne la montrait.

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

- [~] 2.7.1 `traverse.go` (1 380), `unit_weaponstate.go` (970), `frame_records.go` (793),
      `components_biped_ability.go` (699), `components_movement.go` (554) côté grammaire ;
      `document.go` (624), `build.go` (607), `equipment_placements.go` (594), `lives.go` (565)
      côté publication ; `document_chronicle.go` (1 230) est une chronique : exemption écrite en
      tête, pas de scission.
      **CIBLES AJOUTÉES PAR LA REVUE DE JALON M1 (constats C1 / C2, 2026-09-16)** — mesure
      `783ae680d` → `34fa53da5`, les tailles ci-dessus datent de l'écriture du plan et ont
      toutes MONTÉ : `document_chronicle.go` **1 476** (+171, exemption de chronique à
      re-justifier à ce volume), `sync/killcollector/collector.go` **773** (+138, ABSENT de la
      liste ci-dessus), `document.go` **627** (+30), `build.go` **532** (+16),
      `replay/zone_states_hill.go` **526** (+13, ABSENT de la liste ci-dessus),
      `traverse.go` **1 388** (+3). **Et TROIS fonctions au-delà de 80 L qui ont grossi dans le
      jalon**, à traiter au même lot : `BuildFromPositions` 449 → **469** (`replay/build.go`),
      `decimateTracks` 95 → **107** (`replay/tracks_publication.go`), `consumeByName`
      812 → **815** (`filmdec/traverse.go`).
      **ÉTAT AU 2026-09-16 (lot 2.7 volet GRAMMAIRE, `9547d7a4e` + `be0f51504`)** : le volet
      grammaire est FAIT — `traverse.go` 1 388 → **390** (le `switch` de 815 lignes de
      `consumeByName` devient une chaîne de sept maillons, plus `component_param4.go` et
      `traverse_precision.go`), `unit_weaponstate.go` 956 → **486**, `frame_records.go`
      793 → **278**, `components_biped_ability.go` 699 → **343**, `components_movement.go`
      554 → **196**, `sync/killcollector/collector.go` 718 → **211**. `consumeByName` 815 →
      **118**, `consumeObjectPositionDynamicPrecisionD` 111 → **72**, `collect` 91 → **53** ;
      `decodeInferLoop` reste à 114 avec une exemption écrite (boucle à sorties multiples).
      Le volet PUBLICATION (`document.go`, `build.go`, `lives.go`, `zone_states_hill.go`,
      `equipment_placements.go`, `tracks_publication.go`, `document_chronicle.go`, et les deux
      fonctions `BuildFromPositions` / `decimateTracks`) est `[!]` **non traité ici** : les lots
      1.9.9 / 1.9.11 / 1.9.14 tenaient ces fichiers en gates au moment du lot. Il part au lot
      **2.7p**, après leur fusion ; d'ici là le ratchet 2.7.2 les gèle à leur taille du jour.
- [x] 2.7.2 Ratchet de taille : plafond gelé par fichier, jamais accru
      (`archlint/film_file_size_test.go`). FAIT (ce commit) : seuil 500 par défaut sur les
      quatre racines (`film/`, `replaybuild/`, `killcollector/`, `objectiveevents/`), table
      datée de 33 fichiers déjà au-delà (leur taille du 2026-09-16, qui ne peut que DESCENDRE),
      second test contre une entrée périmée, plancher de 1 200 fichiers balayés contre un
      balayage muet. Deux mutations jouées puis restaurées par nom (§5).
- [x] 2.7.3 `benchstat` contre `bench_baseline.txt` : +10 % au plus. FAIT : `TraverseEntity`
      **+3,36 %**, `KeyframeClosure` +5,54 % (informatif, cf. l'en-tête de la ligne de base),
      `BitReaderReadBits` **~** — budget tenu. Lecture honnête : la paire AVANT / APRÈS mesurée
      dans la MÊME session, sur la même machine chargée, ne montre AUCUNE différence
      significative sur les trois bancs (p = 0,393 / 0,143 / 0,436, géomoyenne −1,42 %) : l'écart
      contre la ligne de base figée est la charge machine (six exécuteurs en parallèle), pas le
      déplacement. `bench_baseline.txt` n'est PAS re-figé ici — la clôture de M2 le prescrit.

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
| 2026-09-14 | 1.2.6 | **D2 (1.2) — la sortie de `killsource` PEUT changer, et `KillSourceDecoderRev` n'a PAS été montée : c'est une décision de production.** Mesure : sur `killsource/testdata/minibobine.golden`, UNE ligne sur 76 bouge, `recordStateParam=3 [croissance x1.002]` -> `x1.001` ; le paramètre retenu, les lignes de kill, la couverture, le contrôle négatif, les voies et la santé sont identiques à l'octet. La calibration RSP traverse des records BRUTS (`calibrate.go:157`), donc elle voit les quatre niveaux corrigés ; les lignes publiées, non — sur ce témoin. Monter `KillSourceDecoderRev` rouvre un backlog de redécodage des lignes en base : c'est un geste de PROD, réservé au pilote sur signal (D6). NON TRAITÉ. Le garde-rail `TestKillSourceDecoderRevSuitLeDecodeur` reste vert de lui-même (il ne hache que `film/killsource/`, que le lot ne touche pas) — ce silence est précisément ce qui rend la décision explicite nécessaire. | décision pilote : bump + backlog killsource, ou constat écrit que les lignes ne peuvent pas changer |
| 2026-09-14 | 1.2.4 | **D3 (1.2) — TRAITÉE DANS LE LOT. Les annotations `niveau_jeu=N` d'`ecs_table.tsv` étaient INCOMPLÈTES : 178 lignes annotées pour 189 réellement décalées.** La table documentait l'écart depuis le lot R7-e ; corriger la colonne `level` À PARTIR DE CES ANNOTATIONS laissait 11 lignes fausses (mesuré : G2 rouge sur 11 clés, dont `35|63|biped-action-component` et `44|0|asset-transform-component`). La colonne est donc régénérée DEPUIS LE FILM par une porte nommée, et le README dit que c'est la seule colonne qui ne se corrige pas à la main. LEÇON GÉNÉRALE, non traitée : toute colonne de cette table qui est une donnée du film (et non un jugement humain) dérive silencieusement tant qu'elle se recopie à la main. | lot 2.6 (grammaire en instruments) : recenser les colonnes dérivées de la table ECS |
| 2026-09-14 | 1.2 (équivalence) | **D4 (1.2) — le corpus d'équivalence ne porte AUCUN témoin des archétypes ti=14, 21, 30 et 44.** Les quatre instances dont le niveau change et qu'un déser consomme vivent dans ces archétypes ; sur les 20 films du corpus, les 50 étapes de balayage et l'artefact sont IDENTIQUES avant et après le correctif. Le corpus prouve donc l'absence de régression, mais il est AVEUGLE au gain : aucun film n'exerce le chemin corrigé. NON TRAITÉ — étendre le corpus exigerait un film portant des `crew`/`flock`/`tacmap`/`asset-transform`, c'est-à-dire probablement du PvE ou de la Forge, pas du matchmaking. | lot 3.6 (ports de composants) ou extension de corpus : nommer un témoin par archétype porté |
| 2026-09-14 | 1.2 revue R1 (C2) | **D5 (1.2) — le contrôle G4 ne regarde que les `bits_typ` ENTIERS ; les approximatifs (`~100`, `~115`, `11-30`, « variable ») n'ont AUCUN garde-rail.** `ecs_table_guard_test.go` ne confronte au code que les cellules qui parsent en entier (`ecsRow.BitsTyp = -1` sinon) : la cellule `~100` d'`asset-transform-component` est restée fausse après le lot alors même que le lot changeait le niveau de ce composant, et c'est la revue qui l'a vue, pas le gate. NON TRAITÉ — contrôler un budget approximatif demande de distinguer largeur FIXE et largeur NOMINALE, ce que la table ne fait pas (même obstacle que la note de la ligne 935). | lot 2.6 (grammaire en instruments) : décider si `bits_typ` doit porter une forme contrôlable |
| 2026-09-14 | 1.2 revue R2 (non retenu) | **D7 (1.2) — la cellule `code_source` de la ligne 976 d'`ecs_table.tsv` cite `traverse.go:921` alors que le `case` d'`asset-transform-component` est à `traverse.go:996`.** Décalage PRÉEXISTANT au lot 1.2 (la ligne n'a pas bougé de fichier, c'est le fichier qui a grandi au-dessus d'elle) ; le contrôle G1 confronte la table au code par le NOM du `case`, pas par le numéro de ligne, donc il reste vert. Remarque NON RETENUE par la ronde 2 comme hors périmètre du lot, et NON CORRIGÉE : une correction isolée de cette cellule laisserait les autres dériver de la même façon. | lot 2.6 (grammaire en instruments) : soit `code_source` cite un symbole plutôt qu'une ligne, soit un garde-rail vérifie le numéro |
| 2026-09-14 | 1.2 revue R1 (C1), **complétée à la R2** | **D6 (1.2) — la troncature d'un `chunk_00` est NOMMÉE dans `Registry` mais n'a encore AUCUN LECTEUR, et le seul signal qui sort en exploitation attribue la mauvaise cause.** `Registry.Truncated` dit que le parse a épuisé le tampon sans rencontrer la fin structurelle, `Registry.TruncatedBytes` mesure la queue non couverte (elle peut valoir 0 sur une coupe alignée). Les TROIS appelants de production — relevé du 2026-09-14, `grep -rn "ParseRegistryChunk(" --include=*.go internal/ cmd/ | grep -v _test.go` : `filmdec/film_context.go:254`, `killsource/world.go:58` (via `killsource/decode.go:123`, paquet importé par `killcollector` ET par `replaybuild`), `killcollector/hits.go:113` — n'en journalisent aucun. Ce qui sort alors d'un `chunk_00` tronqué, c'est le WARN de `warnUnknownRegistry` (« grammaire des composants suspecte (mise a jour du jeu ?) ») : le bon signal, la mauvaise cause. NON TRAITÉ dans ce lot — brancher un journal touche trois appelants hors périmètre, et amender le message du WARN touche `registry_fingerprint.go` hors du constat. | **à compter au registre des replis du lot 1.9.0** (nom, fait, condition de déclenchement, date de pose, critère de retrait), ET y porter que `warnUnknownRegistry` doit dire « registre tronqué » quand `Truncated` est vrai |

| 2026-09-14 | 1.3 (mesure avant de coder) | **D1 (1.3) — le témoin figé de la marche delta (`delta_walk_witness_test.go`) est PÉRIMÉ, et il l'était AVANT ce lot.** Mesuré au commit d'intégration `783ae680d`, arbre PROPRE (fichier restauré par nom, mêmes chiffres des deux côtés) : `000d5950` figé {14 350 paquets, 38 878 records, 30 080 aboutis} contre mesuré {14 350, **38 897**, **30 101**} ; `06dfe6d9` figé {6 606, 10 613, 8 502} contre {6 606, **10 629**, **8 499**} ; `64e8adfa` figé {14 357, 39 806, 31 973} contre {14 357, **39 820**, **31 988**}. Le contrat écrit dans le fichier (« si l'une bouge, c'est la GRAMMAIRE qui a bougé, et c'est ce qu'il faut expliquer avant de réécrire le chiffre ») n'a donc pas été tenu par au moins un lot depuis le 2026-08-18. Le test est sous garde `DELTA_WITNESS_FILM` : la CI ne le voit jamais, et c'est pour cela que la dérive a pu traverser 0.A à 1.2 sans être consignée. **Le cas de `06dfe6d9` est le plus parlant : les records MONTENT (+16) mais les traversées abouties DESCENDENT (-3)** — un sens que le contrat du fichier n'accepte pas. NON TRAITÉ : re-figer ici absorberait en silence la dérive d'un autre lot. | **TRAITÉE au lot 1.4.0 (2026-09-14)** : dérive attribuée par bisection (quatre marches, toutes ANTÉRIEURES au chantier), verdict DIVERGENCE, témoin re-figé avec sa cause écrite dans le fichier. Cf. D1 (1.4) pour ce qui reste ouvert (le témoin vit hors CI) |
| 2026-09-14 | 1.3 (équivalence) | **D2 (1.3) — une chaîne de DIAGNOSTIC entre dans l'empreinte d'équivalence de l'étape `killsource`.** Sur `50247b26`, le seul écart des 10 films du régime court est le champ `Result.Calibration`, une phrase lisible : `axisW=14 indexW=1 [PROFIL PLAT (score 88, mediane 77) : valeurs par defaut conservees] | recordStateParam=2 [croissance x1.003]` -> la même avec `mediane 76`. **Un caractère sur 228 800 octets de sortie JSON** ; les paramètres RETENUS (`axisW`, `indexW`, `recordStateParam`, le facteur de croissance), les lignes de kill, le catalogue, la couverture et la santé sont identiques à l'octet. Conséquence : un gate d'équivalence peut rougir pour une phrase de journal, et un lot doit alors prouver que rien de publié ne bouge — ce que la comparaison champ à champ a fait ici, mais au prix d'une passe supplémentaire. NON TRAITÉ. | lot 2.6 (empreintes par couche) : sortir les chaînes de diagnostic de l'empreinte, ou les isoler dans une sous-empreinte nommée |
| 2026-09-14 | 1.3 (équivalence) | **D3 (1.3) — la découverte D4 (1.2) est formulée trop largement, et la mesure la réfute sur ce point.** D4 (1.2) écrit « le corpus d'équivalence ne porte AUCUN témoin des archétypes ti=14, 21, 30 et 44 ». Or `000d5950` et `64e8adfa` SONT au corpus (`CORPUS.txt` l. 141 et 143) et leur marche delta traverse des records NEW de ces archétypes (13 et 13 pour ti=14, 29 et 71 pour ti=21 — histogramme du témoin, §5) ; surtout, le lot 1.3 ne change QUE les états par défaut de ti=14/17/21/29/47 et il fait bouger `50247b26` au balayage `killsource` : le corpus n'est donc pas aveugle à ces archétypes. Ce que D4 voulait dire — aucun film n'exerce les COMPOSANTS qui consomment le niveau (`crew-order`, `flock-destination`, `tacmap-poiicon`, `asset-transform`) — reste plausible et n'est PAS mesuré par ce lot. NON TRAITÉ (hors périmètre 1.3). | reformuler D4 (1.2) au lot 3.6, où le témoin par archétype se nomme |
| 2026-09-14 | 1.4.0 | **D1 (1.4) — TRAITÉE DANS LE LOT : la dérive du témoin de marche delta est attribuée, et elle est ANTÉRIEURE au chantier.** La découverte D1 (1.3) refusait de re-figer sans cause ; la bisection (690 points, §5) nomme QUATRE marches, toutes dans `feat/v75` avant M0 : `62ba098b8` (composants d'objectif/bombe portés), `8f309ce86` (le registre borné à sa fin structurelle), `736ccf3c3` (cuisson-perf + véhicules), `ffb27238c` (grammaire d'i9 relue). Les quatre sont des DIVERGENCES — le point le plus suspect (`8f309ce86`, −7 traversées) ne fait changer de verdict que `ti=49`, qui n'existe PAS dans le registre de `06dfe6d9` (49 blocs) : avant, le parseur le fabriquait à partir du bourrage, avec ZÉRO composant, et une traversée sur un archétype vide « aboutissait » sans rien lire. **CE QUI RESTE OUVERT, ET N'EST PAS TRAITÉ ICI** : le témoin est sous garde `DELTA_WITNESS_FILM`, donc la CI ne le joue jamais et rien n'oblige un lot à le jouer. La leçon du lot 1.3 (le test « `n2` constant » posé SANS garde d'environnement, sur les bobines versionnées) s'applique mot pour mot : tant que ce témoin vit hors CI, il re-dérivera. | clôture M1 ou lot 2.6 : porter le témoin sur les sept bobines versionnées, sans garde d'environnement, comme `default_state_n2_constant_test.go` |
| 2026-09-14 | 1.4.3 | **D2 (1.4) — TRAITÉE DANS LE LOT : le plan attendait « ratchet 0.A.3 régénéré, 0 → 14 % », et c'était une erreur de citation.** Vérifié sur pièces : `keyframe_closure.go` mesure le cadre d'ÉTAT COMPLET depuis sa création (lot 0.A.3) ; il n'a jamais mesuré la production. Le « 0 » attendu était celui des compteurs de PRODUCTION (`KeyRecords`/`KeyWalked` des deux balayages), pas celui de ce golden, qui vaut 30,8 % depuis le lot 1.3. Le lot 1.4 fait rejoindre la production à la mesure : le golden ne bouge pas, et `TestKeyframeClosureRatchet` reste vert SANS régénération. L'en-tête du fichier porte désormais la correction, pour qu'un lecteur futur ne re-déduise pas l'erreur. | corrigée ici ; à relire au lot 3.6, où le ratchet doit enfin MONTER |
| 2026-09-14 | 1.4.2 | **D3 (1.4) — `WalkKeyframeRecords` et `ChainKeyframeRecords` lisent toujours le cadre DELTA sur une table d'image-clé, et ils sont EXPORTÉS.** Grep collé en §5 : zéro appelant de production, quatre appelants `_test.go` dont un HORS du paquet (`replay/visee_etiquettes_keyframe_test.go`), ce qui interdit de les unexporter comme `walkKeyframeBody`. Deux instruments mesurent donc encore des records d'image-clé sous un cadre que ce lot vient de prouver faux pour cette table (`zone_census_report_test.go:254`, `visee_etiquettes_keyframe_test.go:94`) : leurs chiffres ne sont pas des mesures de la grammaire, ce sont des mesures de l'ancien cadre. NON TRAITÉ — hors périmètre 1.4, qui nomme les deux consommateurs et `walkOneKeyframeRecord`. | lot 2.7 (scission / surface exportée) ou 3.6 : rebaser ces deux instruments sur `WalkKeyframeFullState`, puis unexporter |
| 2026-09-14 | 1.4 (équivalence) | **D4 (1.4) — la référence d'équivalence de `50247b26` est PÉRIMÉE depuis le lot 1.3, et chaque lot suivant la paiera.** Le régime court rend 9/10 sur ce lot, et l'unique écart (étape `killsource`, chaîne de diagnostic `calibration`) est le MÊME que celui du lot 1.3 : contrôle décisif, la même commande jouée au commit de base `15309e89e` dans un worktree détaché rend la MÊME empreinte obtenue (`2c4ebf2071ca…`). Le lot 1.4 produit donc zéro différence. Mais tant que la décision pilote de D2 (1.3) n'est pas prise (bump `KillSourceDecoderRev` + backlog, OU re-figeage de la seule référence de `50247b26`), **tout lot de M1 lira un rouge permanent qui masque le prochain vrai** — exactement la situation que le lot 1.1.6 a dû défaire au schéma 55. NON TRAITÉ (geste de PROD, réservé au pilote, D6). | décision pilote, avant le prochain lot décodeur : re-figer `50247b26` ou trancher le backlog killsource |
| 2026-09-14 | 1.4 (mesure) | **D5 (1.4) — la section C.2 de `NOTE_IMAGECLE_ETAT_COMPLET_2026-09-13.md` attribue à tort un « 0 » qui vient de l'INSTRUMENT, pas du corpus.** Elle écrit « le chemin de production ne s'engage pas sur ce corpus : `ScanNavpointRadial` sort avant la boucle quand la bande de slots observés est vide, aucun des six films n'est un Assaut, donc 0 record d'image-clé n'est parsé ». Sur pièces : `TestImageCleProductionBalayagesReels` appelle `ScanFilmNavpointRadial(dir, map[int]int{})` — SANS horloge — et `scanChunk` (`navpoint_radial_scan.go`) fait `PacketsNoClock++` puis `continue` pour TOUT paquet dont le chunk n'a pas de `start_ms`. Cet instrument rend donc `KeyRecords 0` pour ti=12 sur n'importe quel film. **Contre-exemple mesuré** : `c75f33b8` (témoin `assaut_bombe` du corpus gate, où le chemin s'engage réellement — 1 169 lectures d'anneau) rend lui aussi `KeyRecords 0` sous cet instrument, alors qu'il porte **569 records ti=12 en image-clé**. Le « 0 » de la note n'est pas faux comme chiffre, il est faux comme CAUSE — et toute conclusion tirée de cet instrument sur la population ti=12 est à reprendre. NON TRAITÉ (hors périmètre 1.4 : corriger l'instrument, c'est le rebaser sur un manifeste). | lot 2.6 (empreintes / instruments) ou la prochaine reprise de la note 5a : passer une horloge réelle à l'instrument, ou publier `PacketsNoClock` à côté de `KeyRecords` pour qu'un 0 muet devienne un 0 expliqué |

| 2026-09-14 | 1.5.4 | **D1 (1.5) — le critère de fermeture de l'oracle (« lus + vacants == 32 ») est satisfait par le BOURRAGE de queue, donc il ne prouve pas la tête de la table.** La note `NOTE_RESIDUS_CHUNK00` publie « le lecteur lit exactement 32 enregistrements sur 1 351 films sur 1 351 » comme « la fermeture la plus forte du lot ». Elle est vraie, et elle prouve moins qu'elle n'en a l'air : la queue du tampon de `chunk_00` est faite de zéros, donc le prédicat de vacance y passe INDÉFINIMENT, et une marche qui démarre au DERNIER enregistrement rend « 1 occupé + 31 vacants = 32 » — avec n'importe quelle largeur de bloc de personnalisation. Mesuré à l'écriture de ce lot : quatre largeurs fausses sur cinq bobines ferment ainsi. C'est pourquoi le lecteur de production ajoute un second volet (visiter TOUS les enregistrements du balayage) et retient la lecture la plus complète. La note n'est PAS corrigée par ce lot (règle 7) : sa mesure reste juste, c'est sa portée qui est à préciser. | `.ai/V7.5/film_re/NOTE_RESIDUS_CHUNK00_2026-09-13.md` section 2.3, à amender par qui la reprendra |
| 2026-09-14 | 1.5.4 | **D2 (1.5) — `rsChaine` perd la tête de la table sur NEUF films du cache, et son seuil de regroupement en est la cause.** `19ef6b04`, `23ffd885`, `3104391d`, `3b1cfde3`, `59b8abb9`, `652907bb`, `92f7c713`, `a92bab93`, `d4ddf054` : l'instrument lit 1 à 6 enregistrements là où la grammaire en lit 7 ou 8. Chacun porte deux écarts de balayage au-delà de `s3rEcartMax = 40 000` bits, parce qu'un slot vacant intercalé ajoute 16 499 bits (build courant) à l'écart entre deux enregistrements — exactement le défaut que la phase 2 nommait sur 4 films sur 76 (« troncature de tête »), jamais chiffré à l'échelle du cache. Le lecteur de production n'a pas de seuil et ne perd rien ; l'instrument, lui, reste tel quel — c'est un oracle de recherche, et le corriger changerait les chiffres publiés par trois notes. NON TRAITÉ. | l'instrument garde son seuil ; la divergence est GARDÉE par `player_table_corpus_test.go`, qui exige l'explication film par film |
| 2026-09-14 | 1.5.3 | **D3 (1.5) — « rang absolu » et « index parmi les occupés » ne sont pas départageables sur ce corpus, et la population qui les sépare n'a aucun oracle.** L'ordre des enregistrements EST le `player_index` de production (`filmIndex − rang` constant sur 76/76 films, phase 2), mais aucun de ces 76 films ne porte de slot vacant INTERCALÉ : les deux définitions y coïncident. Les 13 films du cache à vacant intercalé (`07f6af1b`, `0d1dddfb`, `19ef6b04`, `1c5c10cc`, `23ffd885`, `3104391d`, `3b1cfde3`, `59b8abb9`, `652907bb`, `92f7c713`, `a92bab93`, `b1bcbe24`, `c744aa29`) n'ont AUCUN document de rejeu — vérifié le 2026-09-14 sur le parc et la sauvegarde. Le lot publie le rang ABSOLU (l'index du tableau que l'écrivain parcourt) et porte `InterleavedVacant` dans le rapport pour que la divergence soit lisible. NON TRANCHÉ. | lot 1.6 : cuire un artefact pour l'un de ces 13 films, ou confronter à `match_participants` — la décision appartient au consommateur |
| 2026-09-14 | 1.5.4 | **D4 (1.5) — le balayage de la table est AVEUGLE à certains enregistrements bien réels, et la grammaire les lit.** Sur `d4ddf054` (HI_1_13_0), le balayage rend 7 enregistrements et la marche en lit 8 ; la trame du même film porte 8 entités `ti=9`, donc 8 joueurs. L'enregistrement manquant échoue l'un des critères d'en-tête (jeton de 48 bits nul, ou XUID hors de la plage Xbox `[0x0009…, 0x000A…)`) — lequel n'est pas établi, et le savoir dirait s'il existe des comptes joueur hors de cette plage. Le lecteur compte le cas (`GapsHidden`), il ne le diagnostique pas. NON TRAITÉ. | lot 1.6 ou 1.8 : si un compte hors plage existe, la borne de XUID de tous les balayages du dépôt est à revoir |
| 2026-09-14 | 1.5.0 | **D5 (1.5) — `lireEntete` compte UNE entrée de trop dans la table par type, et la dérivation structurelle le prouve.** Le résidu G.2 de `NOTE_SECTION3_SLOTS` relevait « 124 entrées sur `HI_1_13_0` là où l'écrivain en écrit 123 » sans trancher. La mesure du 2026-09-14 tranche : `(offset de la chaîne de build − 0x20) − fin du registre` vaut 492 octets sur les 1 269 films des deux builds courants, soit exactement 123 u32, la valeur de l'écrivain (`MOV R9D,0xf60`) ; l'écart vient de l'heuristique de l'instrument (remonter tant que la valeur tient sur 16 bits), qui avale un u32 de plus. `ReadFilmIdentity` n'emploie pas cette heuristique. L'instrument n'est PAS corrigé (règle 7). | `registre_events_research_test.go` : `lireEntete` garde son heuristique ; le résidu G.2 de la note est FERMÉ par cette mesure |

| 2026-09-14 | 1.6.5 | **D1 (1.6) — QUATORZE des vingt vies d'un seul échantillon désormais publiées sont ORPHELINES, et c'est un DÉFAUT DE LECTURE.** L'oracle imposé par l'utilisateur (« une mort écrite à cet instant, OU la dernière image avant une fin de manche ») est mesuré sur les 20 vies des 8 builds (`vies_un_echantillon_test.go`, tableau collé en §5) : **1 mort écrite** (`e5adf7b2` slot 689, écart 72 ms), **5 fins de film** (`cause = film_end` : la réplication du slot ne reprend jamais), **14 ORPHELINES**. Les quatorze se ferment TOUTES sur un trou de réplication (`cause = cut`), et la mort la plus proche du même joueur est à **0,95 s à 300 s** — il n'y a donc pas de mort à cet instant. Lecture la plus économique : un slot réplique une seule image puis se tait plus de `lifeGapUS` (5 s), ce qui DÉCOUPE en deux une vie que le film écrit d'un seul tenant — le défaut n'est pas dans le seuil de publication mais dans la découpe des vies. Sept de ces quatorze portent une mort à ~10,1-10,25 s, une régularité que ce lot ne sait pas expliquer. NON TRAITÉ (hors périmètre 1.6, et l'utilisateur a tranché qu'on publie). | famille 1.9 (conversion d'une heuristique en lecture) ou un lot dédié « découpe des vies » : la borne `lifeGapUS` est exactement le genre de seuil que D13 vise |
| 2026-09-14 | 1.6.0 (mesure avant de coder) | **D2 (1.6) — la table des joueurs de `chunk_00` est le roster du DÉBUT du film, pas celui du match.** Mesuré sur les 8 builds : elle assoit 8 à 24 joueurs, et la lecture des chunks de réplication en connaît **jusqu'à 5 de plus** (`e5adf7b2` 23 contre 28, `a521164d`/`11de8353` 24 contre 27, `bcb6d393` 8 contre 11) — des joueurs arrivés en cours de partie, qui prennent des index AU-DELÀ du dernier siège (8..11 sur `bcb6d393`). La conséquence a été TRAITÉE dans le lot (la table précède la lecture des chunks au lieu de la remplacer, sans quoi 13 joueurs disparaissaient), mais la QUESTION reste ouverte : le film écrit-il ailleurs l'entrée en cours de partie ? Si oui, `link.method` de ces 13 liens cesserait d'être un repli. NON INSTRUIT. | recherche `.ai/V7.5/` : y a-t-il un enregistrement de JOIN dans le flux delta ? Condition de reprise : un témoin au corpus gate dont le repli coûte un fait |
| 2026-09-14 | 1.6.1 | **D3 (1.6) — le comparateur de `replay-equiv` est POSITIONNEL, donc une étape de balayage AJOUTÉE rend toutes les suivantes « différentes ».** `comparer` (`cmd/replay-equiv/parent.go:250`) confronte les lignes TSV par INDICE et s'arrête à la première qui diffère : l'ajout de l'étape `filmTable` a fait rendre « ECART à l'étape "filmTable" : attendu sha=<celui de playerIndices> » sur les 10 films, un message qui NOMME la mauvaise cause. Contourné en classant sur le `git diff` des références après `-update` (10 fichiers, 2 lignes chacun : `+filmTable` et `artifact` modifié — les 49 autres identiques), ce qui donne la classification COMPLÈTE là où le harnais n'en donne qu'une ligne. NON TRAITÉ : le harnais reste juste (il refuse), il est seulement peu diagnostique. | M2 (lot 2.6, empreintes par couche) ou un correctif court : comparer par NOM d'étape et rendre TOUTES les différences, pas la première |
| 2026-09-14 | 1.6.5 (corpus gate) | **D4 (1.6) — le seuil à 2 CACHAIT quatre vies SANS NOM, et les publier les rend visibles : `unnamedLives` monte sur DEUX témoins.** `084a804d` 0 → 1 (`unnamedLivesContested` 0 → 1, `flagCarries.ambiguousSlot` 0 → 1) et `a349fea8` 334 → 337. La cause est MÉCANIQUE et mesurée : ces deux films portaient 9 et 3 vies d'un seul échantillon refusées, et 1 + 3 d'entre elles ne sont nommées par aucune voie. Ce n'est PAS une régression du nommage — le registre ne nomme rien de moins qu'avant, et sur les mêmes vies il nomme mieux (lot 1.6.1/1.6.2) — c'est un défaut de nommage PRÉEXISTANT que le seuil masquait. Il heurte de front la décision utilisateur du 2026-09-06 (« les vies anonymes n'existent pas ; une vie est un humain ou un bot »). La ligne « Preuve » du lot 1.6 (« `unnamedLives` ne monte nulle part ») n'est donc PAS tenue à la lettre, et c'est écrit ici plutôt que tu. NON TRAITÉ : nommer ces quatre vies demande d'instruire la découpe des vies (D1 (1.6)) ou le pont sur un slot qui ne réplique qu'une image. | famille 1.9 / lot « découpe des vies » ; arbitrage utilisateur si les quatre vies doivent être nommées avant la recuisson du parc |
| 2026-09-14 | 1.6.5 (corpus gate) | **D5 (1.6) — une vie d'un seul échantillon élargit les BORNES de la scène, et sur `084a804d` elle les élargit de 184 mètres.** `bounds.minX` −32,21 → −216,30 et `bounds.minY` −65,51 → −89,83 sur `084a804d` ; `bounds.minX` −2,30 → −5,95 et `bounds.minZ` −1,95 → −16,26 sur `e5adf7b2`. Les bornes suivent les traces PUBLIÉES, et le client s'en sert pour cadrer la scène : une vie d'un point à −216 m dézoome le rejeu de ce match. L'échantillon a passé le filtre d'aberration (`boundsRejectSpreads`), donc il est « plausible » au sens de ce filtre — mais un point isolé à 184 m du reste du nuage sur un film de véhicules mérite d'être instruit avant la recuisson du parc. NON TRAITÉ (hors périmètre 1.6 : le seuil est une décision utilisateur, et le cadrage est un sujet de rendu). | lot de rendu / cadrage, ou le lot « découpe des vies » : si la vie d'un point est un artefact de découpe (D1 (1.6)), la borne disparaît avec elle |

| 2026-09-14 | 1.7.0 (mesure avant de coder) | **D1 (1.7) — TRAITÉE DANS LE LOT : le film ÉCRIT l'index de joueur dans le record ti=9, et l'appariement ordinal de la note n'est plus nécessaire.** Le premier `R(6)` de l'état par défaut de ti=9 (`consumeDefaultStateTI9`, `FUN_1410d7540`) vaut exactement le rang du siège de `chunk_00`, entité par entité, CONSTANT sur toute la vie de l'entité, sur 18 films et 7 builds — y compris sur les deux films SANS section d'identification (`a349fea8`, `50247b26`), où la table de `chunk_00` est refusée et où il continue de rendre `0..23` / `0..24`. LE CONTRÔLE QUI INTERDIT D'Y LIRE UN ORDINAL : sur `50247b26` la suite lue est TROUÉE (`0 1 3 4 … 22 24`) ; un rang de parcours est contigu par construction. Même forme que le `R(6)` de ti=5 (`player-waypoint`), que l'exécutable borne à `< 0x20`. Conséquence : `ScanPlayerTeams` n'apparie rien, il LIT ; l'ordinal de la note devient le contrôle (`TestScanPlayerTeamsIndexEstLeSiege`). | fermée ici ; la note `NOTE_EQUIPE_FILM_2026-09-12` garde sa question ouverte n°1 (« expliquer 186 ») — ce lot la ferme aussi, par la dérivation `108 + 32 + 14 + 32` |
| 2026-09-14 | 1.7.1 | **D2 (1.7) — UN record ti=9 du corpus annonce un index HORS de la table de 32.** `111fa685`, entité de slot 5, un unique paquet d'image-clé (t = 6 830 484 ms), `R(6)` = **59**, désignateur brut 0. C'est le SEUL des 3 449 records ti=9 des sept bobines et des 18 films mesurés. Il est REFUSÉ et COMPTÉ (`TeamScanReport.OutOfDomainIndex`), jamais lu : un index de 6 bits accepte 0..63, la table du film n'en porte que 32. NON TRAITÉ : sa cause (record de bourrage, entité d'un autre type mal ancrée, ou slot réel hors table) n'est pas établie, et un seul cas ne suffit pas à la trancher. | lot 3.6 (ports de composants) ou le premier film qui en porte plusieurs |
| 2026-09-14 | 1.7.0 (demande utilisateur du 2026-09-15) | **D-remplacants (1.7) — LE FILM DONNE L'INDEX D'UN REMPLAÇANT, ET IL RÉUTILISE PARFOIS CELUI D'UN PARTANT (2 fois sur 35).** Mesure sur les 18 films (tableau complet en §5). **(1) Index** : chaque arrivant en cours de partie porte son index de joueur dans son record ti=9, comme les autres — il n'y a rien à deviner. **(2) Réutilisation** : sur 35 arrivées, **33 prennent un index NEUF** au-delà du dernier siège, et **DEUX reprennent l'index d'un partant** — `11de8353` index 23 (l'entité de slot 1343 s'arrête au paquet 7, t = 2 328 475 ms ; celle de slot 1789 démarre au paquet 8, t = 2 348 503 ms, écart 20 s) et `51101d1d` index 6 (slot 1309 s'arrête au paquet 2, t = 2 684 409 ; slot 1603 démarre au paquet 5, t = 2 744 424, écart 60 s). **L'ENTITÉ, ELLE, N'EST JAMAIS RÉUTILISÉE** : le slot de réplication d'un arrivant est toujours neuf. **(3) Désignateur** : STABLE sur les 35, sans exception ; sur les deux index réutilisés le partant et l'arrivant portent le MÊME camp, donc aucune divergence d'index n'est levée sur ce corpus. **(4) Instants** : premier et dernier paquet d'image-clé de chaque entité, colonnes du tableau. **VERDICT POUR LE PRODUIT** : le film donne le SIÈGE directement (l'index), donc un remplacement peut être rendu comme tel au lieu d'un appariement ordinal ; ce que le film ne donne PAS ici, c'est le lien `index → xuid` d'un arrivant — la table de `chunk_00` est celle du début (D2 (1.6)) et la lecture des chunks de réplication ne résout pas toujours les arrivants (`11de8353` : 5 entités tardives pour 4 xuids résolus). NON TRAITÉ au-delà de l'équipe : l'affichage « le remplaçant prend la place du partant » est un lot de produit. | lot produit du rejeu (le web affiche tout le roster tout le temps) ; le décodeur, lui, a fini sa part |
| 2026-09-14 | 1.7.2 | **D3 (1.7) — `ZoneInput.TeamByXUID` prend TOUJOURS l'équipe de la base, et ce lot ne l'a pas touché.** `replaybuild/options.go` passe `teamByXUID(facts)` au calque des zones, qui s'en sert pour attribuer une prise de zone à un camp. La règle V4 nomme `Track.Team`, `roster[].team` et `TeamOf` des drapeaux — pas les zones. NON TRAITÉ (règle 7) : le basculer demanderait de vérifier ce que `zone_states` publie et de rejouer ses témoins, ce qui est un lot à soi. | famille 1.9 (la grammaire à la place de l'heuristique) ou un lot de zones |
| 2026-09-14 | 1.7.3 | **D4 (1.7) — `objectiveevents.Extract` n'a AUCUN appelant de production : `match_objective_events.team_id` n'est écrit par PERSONNE par cette voie.** Grep collé en §5 : ses deux seuls appelants hors tests sont `cmd/diag_weapons_v3` (un diagnostic) — le chemin de sync écrit les lignes d'objectif par `persist/bomb_stats_persister.go`, pas par là. Le basculement de source du lot 1.7.3 est donc JUSTE et SANS EFFET sur la base tant que ce point d'entrée n'a pas de producteur. Ce n'est pas une raison de ne pas le faire (il serait faux le jour où il en aura un), c'en est une de ne pas attendre un gain mesurable en base. NON TRAITÉ : rebrancher ou supprimer ce point d'entrée est un arbitrage produit. | lot 1.8 (kill feed) ou un lot de sync |
| 2026-09-14 | 1.7.2 | **D5 (1.7) — CE QUE LE WEB DEVRA FAIRE, et il ne le fait pas encore.** L'artefact publie désormais `roster[].team` et `tracks[].team`, et le web continue de colorer par `team_side` de la feuille de match (`features/match-view/rosterLogic.ts`) : c'est CONFORME au §1.2 de ce plan (aucune règle d'affichage ne change dans ce lot), et c'est aussi ce qui laisse un rejeu SANS feuille de match sans camps à l'écran. Ce qu'il faudra : lire `roster[].team` quand il est PRÉSENT (absent = artefact < 57 ou joueur non nommé par le film), retomber sur `team_side` sinon, et ne JAMAIS confondre `-1` (aucune équipe, mode FFA) avec une absence. Le contrat le permet déjà — le champ est optionnel et `coverage.teams` dit quelle part de l'artefact vient du film. NON TRAITÉ (§1.2). | lot de produit web, hors de ce chantier |

| 2026-09-14 | 1.8.0 (mesure avant de coder) | **D1 (1.8) — TRAITÉE DANS LE LOT, et elle ferme à moitié D3 (1.5) : c'est le RANG ABSOLU que le dead-state emploie, vacants compris.** Les treize films du cache à slot vacant INTERCALÉ — la seule population où « rang absolu » et « index parmi les occupés » divergent, et que le lot 1.5 ne pouvait pas départager faute de document de rejeu — rendent **119 accords sur 123** entre la table de `chunk_00` et la bijection inférée par le kill-feed : `07f6af1b` 7/7, `0d1dddfb` 7/7, `19ef6b04` 7/7, `3104391d` 7/7, `3b1cfde3` 7/7, `59b8abb9` 7/7, `652907bb` 7/7, `92f7c713` 7/7, `a92bab93` 7/7, `c744aa29` 7/7, `1c5c10cc` 22/23, `b1bcbe24` 22/23, `23ffd885` 5/7. Les quatre écarts sont ceux des deux familles du rapport 1.8.0 (gamertag absent du feed ; film à marge nulle), pas un décalage d'index. **CE QUE LA MESURE NE DIT PAS** : elle ne prouve le rang absolu que sur la population que le kill-feed nomme — un siège vacant intercalé SUIVI d'un joueur qui ne tue ni ne meurt reste hors de portée de cet oracle. | fermée pour le décodeur des morts ; D3 (1.5) reste ouverte pour le `player_index` de l'artefact, où l'oracle manque toujours |
| 2026-09-14 | 1.8.1 | **D2 (1.8) — la traduction « erreur typée de `filmdec` → cause NOMMÉE » existe maintenant en DEUX exemplaires, et c'est la dernière copie tolérable.** `replay/film_player_table.go` (lot 1.6.0) et `killsource/film_table.go` (ce lot) portent les mêmes six causes (`sans_registre`, `sans_section`, `build_inconnu`, `tronque`, `table_introuvable`, plus la valeur nominale) et les deux mêmes fonctions de traduction. Les deux paquets sont volontairement disjoints (`killsource` n'importe pas `replay`, et ne doit pas), donc la centralisation n'est pas un simple déplacement : elle irait chez `filmdec`, propriétaire des erreurs. NON TRAITÉ (CLAUDE.md règle 6 : ≤ 2 copies ; à la troisième, centraliser ET poser le garde-rail). | lot 2.7 (scission des fichiers) ou le premier lot qui aurait besoin d'un TROISIÈME lecteur de la table |
| 2026-09-14 | 1.8.1 | **D3 (1.8) — `resolvePlayerIndices` reste une INFÉRENCE, et elle sert deux tables que ce lot ne touche pas.** La voie nommée par le brief (`killcollector/shots.go:122`) ne résout pas les morts : elle résout les **tirs** (`match_weapon_shots`) et les **touches** (`match_weapon_accuracy`, `match_weapon_hit_distance`), en CHERCHANT le motif du xuid dans le flux de réplication et en lisant les 5 bits qui le précèdent — 77,0 % d'accord contre l'oracle killsource (239 films, 16 411 kills). La table de `chunk_00` donne le même lien par LECTURE. **ET LA MESURE DE CE LOT DÉSAMORCE L'URGENCE** : confrontée directement à la table du film sur les 12 témoins lisibles, cette recherche rend **143 accords sur 143 sièges, 0 contradiction, 0 non résolu** (sonde jetable, §5). Le « 77 % » ne mesurait donc PAS sa lecture : il mesurait son accord avec l'oracle killsource, c'est-à-dire avec la bijection hongroise dont ce lot montre justement qu'elle se trompe sur les joueurs absents du kill-feed. Ce que le basculement apporterait n'est pas la justesse de l'indice mais la fiabilité du ROSTER D'ENTRÉE (les xuids du film au lieu de ceux de la base) et la disparition des non-résolus. Il monte `WeaponShotsDecoderRev` ET `migration.WeaponHitDistanceDecoderRev` — deux backlogs de redécodage de plus, et deux axes que la ligne « Preuve » de ce lot (kills / morts / sources) ne couvre pas. NON TRAITÉ ; le renvoi est écrit dans l'en-tête de `hits.go`. | un lot de la famille 1.9 (D13 : la grammaire à la place de l'heuristique, un fait par lot), après décision pilote sur les deux backlogs |
| 2026-09-14 | 1.8 (équivalence) | **D4 (1.8) — l'observateur d'équivalence hache le `Result` ENTIER de `killsource`, donc un champ AJOUTÉ fait rougir les dix films sans qu'un seul octet publié ne bouge.** `replaybuild.go:357` fait `b.observe("killsource", ksRes)` : les trois champs neufs du lot (`Roster.IndexSource`, `Roster.FilmTable`, `BijectionDetermined`) changent le digest de l'étape sur **10 films sur 10**, y compris `50247b26` où la table est REFUSÉE et où la bijection est identique au bit près. La preuve que rien de publié ne bouge est indirecte mais nette : `neutralDeaths`, `killRefs` et `artifact` — trois PROJECTIONS du même `Result` — sont identiques à l'octet sur les dix films. Conséquence pratique : tout lot qui enrichit un type observé paie un re-figeage complet des références, et doit prouver par les étapes VOISINES ce que l'étape elle-même ne peut plus dire. Même famille que D2 (1.3) (une chaîne de diagnostic dans l'empreinte). NON TRAITÉ. | lot 2.6 (empreintes par couche) : observer la PROJECTION consommée plutôt que l'objet entier, ou séparer une sous-empreinte « diagnostic » d'une sous-empreinte « publié » |

| 2026-09-14 | 1.9.0 (recensement) | **D1 (1.9.0) — la table (E) de l'audit 0.E compte 60 LIGNES pour 98 SITES, et son en-tête annonce « 62 replis anonymes ».** L'écart n'est pas une erreur : l'audit REGROUPE par FAIT décidé (« sept sites qui décident le même fait sont UNE ligne à sept sites, parce qu'ils se convertissent ensemble », §3 de l'audit) et son compte de clôture est intermédiaire entre les deux. Le registre suit le regroupement par fait, quitte à scinder une ligne en deux entrées quand les deux moitiés n'ont NI la même condition NI la même cible de retrait (`usage_summary.go` : « dernier occupant du match » et « première vie du slot » se retirent au même lot mais ne mesurent pas la même chose). 95 entrées au total. | sans objet — écrit ici pour qu'un futur lot ne cherche pas 62 entrées |
| 2026-09-14 | 1.9.0 (recensement) | **D2 (1.9.0) — UNE ligne de la table (E) est CONVERTIE et UN de ses sites a DISPARU depuis le 2026-09-13.** (a) `replay/build.go:580`, « `Team: -1` sur TOUTE piste, inconditionnel » : le lot 1.7 en a fait une INITIALISATION, l'équipe est posée juste après depuis le désignateur du film (`build.go:130`, `equipes.poserSurLesTraces`), et ce qui reste à -1 est compté par `coverage.teams.{noTeam, unread}`. Elle n'entre donc PAS au registre. (b) `replaybuild/matchfacts.go`, site « `ParseUint` échoue (bot) » : `grep -c ParseUint internal/replaybuild/matchfacts.go` rend **0** — la règle a déménagé dans `replay.RosterXUIDsOf` au lot 1.0 (revue R1, constat R1-1). Les 58 autres lignes existent toutes, ancre vérifiée une par une par le garde-rail. | traité dans le lot |
| 2026-09-14 | 1.9.0 | **D3 (1.9.0) — CE QUI BORNE LE CÂBLAGE DES COMPTEURS N'EST PAS LE TEMPS, C'EST LA LIMITE DE CINQ PARAMÈTRES.** Un compteur par cuisson doit traverser la chaîne d'appel jusqu'au site ; or les fonctions pures du décodeur sont DÉJÀ à cinq paramètres sur les chemins les plus intéressants (`buildEquipmentPlacements`, `attachVehicles`, `buildDesignatedHills`, `grappleLine`). Dix sites ont été câblés en passant par des porteurs qui existaient déjà — `replayClock` (qui portait déjà `families`, donc n'est plus une horloge depuis longtemps), `flagCarryCtx`, `zoneCtx`, `Options` — et rien d'autre n'a été élargi. Les 85 autres entrées portent `CibleComptage` : leur lot de conversion ouvre déjà le fichier, le compteur y coûte une ligne ; et le pas 2 de M2 (« les lecteurs reçoivent le profil, famille par famille ») fournira le porteur générique. Câbler maintenant puis re-câbler au pas 2 serait payer deux fois. | famille 1.9 (entrée par entrée) + pas 2 de M2 |
| 2026-09-14 | 1.9.0 | **D4 (1.9.0) — SEPT replis décident DEVANT une lecture disponible, et c'est un ratchet neuf.** `fallback.OrdreDevantLaLecture` nomme la VIOLATION de D14 (b) : `repli_vie_coupee_au_trou_de_replication` (le film écrit les morts), `repli_fin_de_vie_vehicule_par_recensement` (`ti=40` est lisible), `repli_chunk_du_pied_par_argmax` et `repli_type_de_chunk_perdu_du_manifeste` (le manifeste donne le type), `repli_manches_contigues_decretees` (148 records portent le désignateur `2`), `repli_i0_porte_et_region_par_defaut` (`I0LayoutReport.IndexBitOnes` MESURE le cas et le rapport est jeté), `repli_largeur_absolue_uniforme`. Le compte est gelé à 7 par `TestReplisDevantLaLectureNeMontentPas` ; chaque conversion 1.9.x le fait descendre, et le baisser EST le geste qui clôt la conversion. | famille 1.9 |
| 2026-09-14 | 1.9.0 | **D5 (1.9.0) — les 38 lignes NOMMÉES de la table (C) de l'audit ne sont PAS au registre, et c'est un choix.** Ce sont des replis LÉGITIMES (le film n'écrit pas le fait, chacune cite son négatif mesuré) déjà nommés et comptés par leur propre couverture ; les recopier n'ajouterait aucun contrôle et diluerait le seul chiffre qui pilote — le nombre de replis À RETIRER. N'y entrent que celles qu'un item 1.9.x vise nommément (fenêtre temporelle des poses, bornes de vie de véhicule, cap de véhicule). Conséquence assumée : le garde-rail ne les voit pas, puisqu'elles ne suivent pas la convention de nommage (elles se nomment par leur compteur, `OriginUnknown`, `Owner: -1`, `VehicleRideSrcGap`). | à trancher à la clôture de M1 : les entrer en bloc, ou laisser la table (C) faire foi |
| 2026-09-14 | 1.9.0 | **D6 (1.9.0) — TROIS garde-rails existants ont mordu sur le REGISTRE, parce qu'un registre documentaire est du code comme un autre.** (1) `no_french_label_literal_test.go` compte les littéraux ACCENTUÉS de `internal/games/` : les 339 phrases du registre en portaient. Résolution : les CHAÎNES du paquet s'écrivent sans accent (les commentaires les gardent), convention déjà appliquée par `killsource/` — l'allowlist, dont l'objectif est de se vider, n'a pas grandi d'un fichier. (2) `no_identity_bridge_outside_registry_test.go` interdit `.SlotXUID` hors du registre d'identité, commentaires exclus mais CHAÎNES comprises : une ancre qui citait la ligne de code l'a fait rougir, l'ancre cite désormais la garde voisine. (3) `filmdec/world_object_precision_guard_test.go` exige que tout fichier mentionnant le global de précision dise d'où il tient ses largeurs : un commentaire du registre le nommait. Leçon générale : **une ancre doit être choisie contre les garde-rails du dépôt autant que contre la dérive du code.** | sans objet (traité) |
| 2026-09-14 | 1.9.0 | **D7 (1.9.0) — le chemin du FIXTURE ne déclenche que 2 des 10 compteurs câblés, et cela confirme D7 (0.D) plutôt que de l'infirmer.** Les huit goldens d'assemblage portent `repli_origine_pose_vie_la_plus_proche` (92 à 506 par film) et `repli_fin_de_vie_vehicule_par_recensement` (0 à 38) ; les huit autres restent à zéro parce que leurs canaux ne sont pas au fixture (inventaire, largeurs d'axe : étage de BALAYAGE) ou parce que leur mode n'est pas au banc (drapeau, colline, bombe). Le régime court d'équivalence, lui, cuit la production entière : c'est là que leur compte se lit. | sans objet — écrit pour qu'un zéro de golden ne se lise pas comme un repli mort |

| 2026-09-15 | 1.9.1 | **D1 (1.9.1) — LE VOCABULAIRE A ETE TRANCHE PAR L'UTILISATEUR LE 2026-09-15, ET LE LOT L'APPLIQUE.** **LA DEFINITION, ET LE MOT DU JEU** : le jeu n'ecrit AUCUN evenement « equipment drop » (seul `weapon_drop`, type 46, existe, pour les armes) ; il ecrit des COMPOSANTS D'ETAT sur l'objet — `i20 equipment-deployed`, `i21 equipment-activated`, `i18 item-at-rest`, `i10 object-parent-state`, `i23 equipment-creator` — plus l'evenement `EquipmentSpawnedObject` (103). Notre vocabulaire s'y calque : « deploye » est le mot du jeu ; « lache » = l'objet quitte le porteur SANS etre deploye, mort ou echange confondus, la cause allant dans `byCause`. Un appareil porte n'a AUCUN deploiement ecrit (0/91, 0/4 853, F.0), donc toujours `dropped`. La question etait : une pose d'appareil PORTE que le film explique par un `taken` du poseur a moins d'une milliseconde n'est pas un deploiement, c'est un lacher VOLONTAIRE a mi-vie (rapport E0, question 5) — fallait-il une quatrieme origine ? **REPONSE : non — « lache ».** `deployed` est reserve a une pose DESIGNEE par un record 103 ; un appareil porte qui tombe est `dropped`, mort ou echange ; la CAUSE va dans `coverage.placements.byCause`, pas dans l'etiquette. **Ce que cela a change, mesure : 108 poses `deployed -> dropped`** (le lacher a mi-vie), **plus 649 poses `-> unknown`** (la fenetre de 200 ms ne les etiquette plus). **CE QUI RESTE OUVERT, ET C'EST LE SEUL POINT** : `repli_piece_engendree_sans_evenement` publie `deployed` sur **9 poses de panneau sur 124** SANS qu'un 103 les designe, au nom du manifeste du titre (`kind = "deployed"`, donnee ECRITE, decision H.2 du 2026-09-13). C'est la seule entorse a la regle « `deployed` = designe par un 103 », elle est nommee, comptee et datee, et elle se retire en une ligne. | a signaler a l'utilisateur : garder le repli du manifeste (9 poses, les 2 builds les plus anciens) ou faire sortir ces poses en `unknown` ? |
| 2026-09-15 | 1.9.1 | **D1 bis (1.9.1) — LE VOCABULAIRE DU JEU EST UN ETAT ECRIT SUR L'OBJET, ET AUCUN DE SES COMPOSANTS N'EST AU RECORD DE CREATION : 0 SUR 4 583.** Question de l'utilisateur (« comment c'est appele dans le code du jeu ? ») : le jeu n'ecrit AUCUN evenement « equipment drop » — il n'en existe pas, seul `weapon_drop` (type 46) existe et il est pour les armes. Ce qu'il ecrit sur un objet d'equipement, ce sont des COMPOSANTS D'ETAT de l'archetype 37 (`filmdec/testdata/ecs_table.tsv`) : `i10 object-parent-state` (le porteur), `i11 object-dead-state`, `i18 item-at-rest`, `i20 equipment-deployed`, `i21 equipment-activated`, `i23 equipment-creator`. **MESURE DU 2026-09-15** (`e191_composants_research_test.go`, 13 films, 334,5 s, PRESENCE AU MASQUE seulement — aucune grammaire de composant portee) : les poses publiees s'apparient a leur record de creation **4 583 sur 4 583, 0 orpheline**, et **AUCUN des six composants n'y figure — zero, sur toutes les familles et toutes les origines**. `i20` n'est donc PAS discriminant a l'instant de la pose : il est ABSENT partout. **CONSEQUENCE, CONFORME AU CADRAGE DU PILOTE** : la regle du lot reste 103 / mort ecrite / `taken` du porteur, et la grammaire complete de `ti=37` (31 composants, dont 7 seulement sont lus par nom aujourd'hui : i18, i19, i22, i25, i28, i29, i30) est le lot 1.9.1 bis. **CE QUE LA MESURE NE DIT PAS, ET IL FAUT L'ENTENDRE** : elle porte sur le record de CREATION. Un composant ECRIT PLUS TARD dans la vie de l'objet — un capteur qu'on deploie une seconde apres l'avoir lache — n'y figure pas par construction. Les records DELTA de la meme vie n'ont PAS ete mesures ici (hors perimetre). | lot 1.9.1 bis : relire les 31 composants chez l'ecrivain, fermer l'image-cle a 100 %, mesurer i20/i21/i18/i10 sur les records DELTA de la vie de l'objet — c'est la qu'ils se trouvent, s'ils s'y trouvent |
| 2026-09-15 | 1.9.1 | **D1 ter (1.9.1) — LE CALQUE DES RAMASSAGES BOUGE AVEC LES POSES, ET CE N'ETAIT PAS PREVU AU BRIEF.** `pickup_origin.go` classe un ramassage `ground` quand il tombe a moins d'un metre d'une pose dont l'origine MESUREE est `dropped` — il REUTILISE la mesure des poses, il ne la refait pas (c'est ecrit dans son en-tete). En changeant l'ensemble des poses `dropped` (3 717 -> 3 364 : +108 lachers a mi-vie, -461 poses muettes), le lot change donc la classification des ramassages. Le corpus gate le montre sur 9 temoins : `coverage.pickups.originUnknown` **BAISSE** partout ou il bouge (236->222, 150->138, 69->59, 57->41, 55->49, 49->45, 43->40, 39->38, 32->28) — c'est-a-dire que MOINS de ramassages restent non classes —, et `originGround` baisse sur un seul temoin (21->18, `pickups.origin/presents` 75->72). **CE N'EST PAS UNE REGRESSION, ET LA DIRECTION EST FAVORABLE** : les 108 poses qui entrent dans `dropped` sont des lachers A MI-VIE, donc des objets reellement au sol pendant qu'un joueur vit et peut les ramasser ; celles qui en sortent sont les poses dont le film ne dit rien. `originSpawner` ne peut pas avoir bouge : il ne depend que du catalogue de points de la carte et des positions, que ce lot ne touche pas. | sans objet — ecrit pour qu'un futur lot ne cherche pas une cause dans le calque des ramassages |
| 2026-09-15 | 1.9.1 | **D2 (1.9.1) — LE TYPE 103 NE SE LIT PAS SUR LES BUILDS LES PLUS ANCIENS, ET LE COMPTE LE DIT.** Événements 103 lus par film (lecture de tête, mesure du 2026-09-15) : `a521164d` HI_1_4_1 **0 sur 4 956 listes non vides**, `50247b26` v31 sans section **2 sur 7 779**, `60ae07c4` HI_1_8_0 **0 sur 6 056**, `51101d1d` HI_1_13_0 **0 sur 1 599**, `fb1a1a72` HI_1_13_0 **0 sur 7 850**, `d9781168` **3**, `bcb6d393` HI_1_12_0 **1** — contre 30 à 86 sur `000d5950`, `11de8353`, `111fa685`, `e5adf7b2`, `0797ce72`, `4f77afc1`. **DEUX CAUSES NE SONT PAS DÉPARTAGÉES** et il faut les distinguer avant de conclure : (a) le film ne porte pas l'événement parce qu'aucune pièce n'a été engendrée — c'est le cas de `51101d1d` et `fb1a1a72`, qui ne publient AUCUN panneau ; (b) la grammaire de la liste ou les largeurs de référence diffèrent par build — seul candidat restant : `a521164d` (2 panneaux publiés, 0 événement) et `50247b26` (7 panneaux, 2 événements dont aucun ne les désigne). Ces 9 poses sont exactement le compte de `repli_piece_engendree_sans_evenement`. **NON TRAITÉ** : instruire (b) demande le profil par build (M3, lot 3.x) ; le repli les couvre, nommé, compté et daté. | lot 3.x (profil par build) ; critère de retrait du repli : 0 déclenchement sur les 8 builds |
| 2026-09-15 | 1.9.1 | **D3 (1.9.1) — 649 POSES A POSEUR MESURE SORTENT `unknown` PARCE QUE LE FILM NE DIT RIEN D'ELLES, ET 461 D'ENTRE ELLES ONT UN SIEGE SANS AUCUNE MORT ECRITE DU MATCH.** Mesure [3] du 2026-09-15 : sur les 3 717 poses qu'on publiait `dropped`, 3 256 ont une mort ecrite (toutes a 171,7 ms au plus) et **461 n'ont aucune mort ecrite sur ce siege, a aucun instant du film** — ce n'est donc pas une question de fenetre, c'est le PONT siege -> mort qui ne couvre pas ces sieges (bots, sieges non nommes, vies fermees autrement). Avec la decision du 2026-09-15 elles ne recoivent plus d'etiquette : `byCause.none` en compte **649** au total (461 ex-`dropped` + 188 ex-`deployed`). **NON TRAITE** : c'est la matiere du lot 1.9.13 (« une vie finit a une mort ECRITE »), qui borne les vies sur les morts du film ; `byCause.none` est exactement le compteur qui doit tomber. | lot 1.9.13 ; critere mesurable deja publie dans l'artefact |
| 2026-09-15 | 1.9.1 | **D4 (1.9.1) — `ref0` du type 103 est LUE ET PUBLIÉE BRUTE, jamais interprétée.** Le lecteur neuf rend `EquipmentSpawnEvent.Source` (index + base 512, génération) parce que la jeter aurait obligé un futur lot à rouvrir la porte aux octets. Ce que le rapport F.0 en dit tient : elle désigne un `ti=37` que les images-clés voient (737 sur 739) et qu'AUCUNE création delta ne porte (2,8 %) — une entité de longue durée, PISTE pour l'équipement SOURCE d'un déploiement. **NON INSTRUITE** (hors lot, table D du registre 0.E) : aucune décision ne repose dessus, et `Ref2Present` est compté sans être lu pour la même raison. | table (D) du registre 0.E ; condition de reprise du repli `repli_origine_pose_fenetre_temporelle` |
| 2026-09-15 | 1.9.1 | **D5 (1.9.1) — LE 103 DÉSIGNE MAJORITAIREMENT DES PROJECTILES, ET C'EST POURQUOI SON COMPTE NE SE LIT PAS COMME UN COMPTE DE MURS.** Sur les 13 films mesurés, 341 événements 103 sont lus et 115 seulement désignent une pose publiée (toutes des panneaux de mur). Les autres désignent des vies `ti=41` — le rapport F.0 §1.2 le mesurait déjà (68,6 % de projectiles) et la mesure de ce lot le confirme sans l'instrumenter : `d9781168` lit 3 événements et ne publie AUCUN panneau, `bcb6d393` en lit 1 pour 0 panneau. Le type dit « un OBJET a été engendré », pas « un équipement a été déployé » — `coverage.placements.spawnEvents` est donc un dénominateur de LECTURE, jamais un compte de murs. | sans objet — écrit pour qu'un futur lot ne lise pas `spawnEvents` comme un compte de déploiements |
| 2026-09-15 | 1.9.1 bis | **D1 (1.9.1 bis) — LA FERMETURE D'IMAGE-CLÉ ÉCHOUE SUR TOUTE LA FAMILLE « OBJET DU MONDE », ET ti=37 N'EN EST QU'UNE VICTIME.** Mesure du 2026-09-15 (`TestE191bCarteTI37`, 7 bobines par build, tous archétypes) : les archétypes qui portent `object-position-component` ferment **184 / 21 698 records bornés (0,85 %)**, ceux qui ne le portent pas **12 654 / 28 573 (44,29 %)**. Détail : ti=37 **3/3 331**, ti=38 **180/12 064**, ti=41 **0/110**, ti=42 **1/2 087** (l'archétype `ground-weapon`, celui dont le dépôt dit la grammaire « réputée complète »), ti=43 **0/4 106**. Le composant qui FAIT franchir la frontière (preuve bornante : le composant qui précède le premier dont le bit de départ dépasse déjà la frontière) est, sur les 3 331 records et de façon stable sur les 7 bobines : **i15 `object-low-frequency` 655 (largeur moyenne 570 bits)**, **i6 `object-region-state` 341 (664)**, **i14 `object-dissolver` 305 (113)**, **i9 `object-multiplayer-properties` 296 (1 508 300 !)**, **i17 `object-frame-configuration` 266 (87)**, **i7 `object-damage-sections` 188 (317)** — six composants du PRÉFIXE OBJET ; aucun composant d'équipement (i18-i30) ne dépasse 71. Quatre d'entre eux (i6, i7, i17 et le voisin i8) n'ont AUCUNE adresse d'écrivain dans `testdata/ecs_table.tsv` et leur colonne `grammar` dit « boucle de regions » / « boucle de sections » / « inconnue ». **NON TRAITÉ** (D3 : une grammaire se relit chez l'écrivain, et l'outil n'était pas disponible — D5 ci-dessous). | lot 1.9.1 bis pas 2, périmètre ÉLARGI au préfixe objet (ou lot 3.6 si le pilote préfère l'y router) : relire i15, i6, i14, i9, i17, i7 chez l'écrivain, re-mesurer la fermeture des cinq archétypes objet ensemble |
| 2026-09-15 | 1.9.1 bis | **D2 (1.9.1 bis) — `KeyframeClosure` MESURE LES ARCHÉTYPES OBJET AUX LARGEURS D'AXE D'UNE AUTRE CARTE, ET PERSONNE NE LE DISAIT.** `object-position-component` lit ses trois axes dans `WorldObjectPrecision` (`traverse.go:154`), un descripteur de paquet dont le défaut est celui de Cliffhanger (13/13/14, index de région 1 bit) et que seul `replay.BuildFromFilm` installe depuis le catalogue pour la durée d'une cuisson. Le ratchet 0.A.3 et son golden ne l'installent pas : les lignes ti=37/38/41/42/43 du golden sont donc mesurées, pour six bobines sur sept, aux largeurs d'une carte qui n'est pas la leur. **CE N'EST PAS LA CAUSE DE L'ÉCHEC DE FERMETURE, ET C'EST MESURÉ** (`TestE191bFermetureAvecCarte`, catalogue versionné) : **3/3 331 au défaut, 3/3 331 aux largeurs de la carte jouée** ; deux bobines échangent leur record fermé (`11de8353` 1 -> 0, `e5adf7b2` 0 -> 1), le total est identique. **NON TRAITÉ** : corriger l'instrument ferait DESCENDRE une ligne du golden (`11de8353` ti=37 1 -> 0) et rougir le ratchet pour un gain net nul ; le geste n'a de sens qu'une fois le préfixe objet relu, quand la fermeture aura une vraie valeur à défendre. | lot 1.9.1 bis pas 2 (ou 3.6) : installer le découpage du catalogue dans `KeyframeClosure` DANS le même commit que la montée de fermeture qu'il rend visible |
| 2026-09-15 | 1.9.1 bis | **D3 (1.9.1 bis) — LES 31 COMPOSANTS DE ti=37 SONT TOUS DISPATCHÉS ; LE « 7 LUS PAR NOM » DU BRIEF EST UN ARTEFACT DE GREP.** Huit composants entrent dans `consumeByName` par une CONSTANTE et non par le littéral : `compObjectBodyVitality` (i4, `registry.go:58`), `compObjectMultiplayerProperties` (i9, `ground_weapon_ammo.go:82`), `compEquipmentDeployed` / `Activated` / `Creator` / `Energy` / `EnergyDelay` / `Charges` (i20, i21, i23, i24, i26, i27, `equipment_state.go:35-40`). Un grep du nom de composant dans `traverse.go` en manque donc huit — et le décodeur, lui, les consomme : **0 désynchronisation sur 3 331 records bornés de ti=37**, colonne « bloquant » vide dans le golden 0.A.3. La vraie question n'est pas « le composant est-il lu ? » mais « sa largeur est-elle juste ? », et seule la fermeture y répond. | sans objet — écrit pour qu'un futur brief ne reparte pas de ce compte ; la règle de mesure est : compter les `case`, pas les littéraux |
| 2026-09-15 | 1.9.1 bis | **D4 (1.9.1 bis) — LES BOBINES PAR BUILD NE PORTENT AUCUN PAQUET DELTA, ET LES MASQUES ti=37 LUS SUR FILMS ENTIERS SONT DU BRUIT.** Mesuré : **0 paquet delta sur les 7 bobines** (elles sont `chunk_00` + un chunk d'image-clé + le pied, V7) — toute mesure de masque y rendrait 0 par construction du fixture, pas par un fait du jeu. Sur 7 films ENTIERS du cache (12 chunks chacun, `TestE191bMasqueTI37`) : 7 142 à 14 358 paquets delta, 10 718 à 40 672 records tous archétypes, mais seulement **92 records NEW et 70 records DELTA de ti=37**, et la présence au masque est **PLATE entre 10,9 % et 34,3 % sur les 31 index** — la signature de bits aléatoires. Conséquence directe pour D1 bis (1.9.1) : la question « où et quand le jeu écrit i10/i11/i18/i20/i21/i23 » NE PEUT PAS être répondue par cette voie tant que la marche ti=37 ne ferme pas. Le chemin de production spécialisé (`ScanFilmEquipmentCreations`) reste, lui, la seule lecture dont on ait mesuré l'appariement (4 583/4 583 au lot 1.9.1). | lot 1.9.1 bis pas 3, APRÈS la fermeture : la mesure de masque se rejoue alors, et elle vaut |
| 2026-09-15 | 1.9.1 bis | **D5 (1.9.1 bis) — GHIDRA N'ÉTAIT PAS DISPONIBLE DANS LA SESSION, ET AUCUNE GRAMMAIRE N'A DONC ÉTÉ POSÉE.** `list_instances` rend « No running Ghidra instances found » ; `connect_instance` rend « connexion refusée » sur `127.0.0.1:8089` (UDS : 0 trouvé). D3 et D13 sont sans ambiguïté : une grammaire se relit chez l'écrivain, jamais sur un motif de bits, et un composant non résolu reste non résolu et compte. Le pas 2 du lot (porter la grammaire des composants) est donc **NON ENTAMÉ**, et l'arrêt est propre en fin de pas 1 (mesure), comme le brief l'autorise. Ce qui MANQUE est chiffré, pas vague : six composants nommés, 2 051 des 3 331 records de ti=37 expliqués par eux. | reprise du lot : ouvrir l'instance `HaloInfinite.exe` (base `0x140000000`, lecture seule) AVANT de relancer l'exécuteur |
| 2026-09-15 | 1.9.1 bis (pas 2) | **D1 (1.9.1 bis, pas 2) — LES QUINZE COMPOSANTS DU PRÉFIXE OBJET SONT RELUS CHEZ L'ÉCRIVAIN, ET AUCUNE LARGEUR NE CHANGE. D1 (1.9.1 bis) EST DONC RÉFUTÉE : LE PRÉFIXE OBJET N'EST PAS LA CAUSE.** Ghidra ouvert en HTTP direct, lecture seule. La résolution est mécanique et reproductible : `chaîne du nom -> xref (thunk de nom) -> xref (slot de vtable +0x08) -> vtable_base+0x30`. Relus : i2 `FUN_14076e278 -> FUN_140c5f938 -> FUN_140c5fa84` (R(1) ; si 0 : R(19) ; puis R(8) INCONDITIONNEL) · i4 `FUN_140fb8978` · i5 `FUN_140d50cbc` · i6 `FUN_140e1bfa0` (R(1) + R(6) n + n×R(3) [si présent : n×R(10)] — max théorique **826**, exactement le max observé au pas 1) · i7 `FUN_142f03c80` (R(6) n + n×(R(1)[si 1 : R(7)+R(16)])) · i8 `FUN_142f039cc` · i9 `FUN_140f53308 -> FUN_1407d4c94` (l'enveloppeur de vtable est identifié) · i10 `FUN_140c1e4d0` · i12 `FUN_1407dc6e4` · i13 `FUN_1407ee054 -> FUN_1407eef08` · i14 `FUN_140dd9f9c` · i15 `FUN_1407ef088` (toutes largeurs relues instruction par instruction : 2/7/8/4/6/6, boucle 12+12, 7×1, 4, 3/2/5/8×3, 3/32/32/14) · i16 `FUN_1407ee070` · i17 `FUN_1407f0534 -> FUN_1407f0550` (compte = `R(6)+1`, **trois** itérations bornées par le tableau) · i21 `FUN_140c1dc80`. **FERMETURE DES CINQ ARCHÉTYPES OBJET, AVANT = APRÈS : 184 / 21 698 (0,85 %).** Résultat ATTENDU d'une relecture qui ne corrige rien — et il ÉLIMINE les six suspects du pas 1. | pas 3 BLOQUÉ : la cause de non-fermeture de ti=37 n'est dans aucun des quinze composants du préfixe, ni dans l'état par défaut, ni dans les ancres |
| 2026-09-15 | 1.9.1 bis (pas 2) | **D2 (1.9.1 bis, pas 2) — LA DICHOTOMIE DU PAS 1 EST UN CONFONDANT, ET LA VRAIE POPULATION QUI ÉCHOUE EST « TOUT CE QUI N'EST PAS UN PETIT ARCHÉTYPE DE MOTEUR ».** Mesure du 2026-09-15 (`TestE191cPrefixeObjet`, 7 bobines versionnées, 2,7 s) : les 44,29 % « sans `object-position-component` » du pas 1 viennent de **huit archétypes seulement** (ti=6 4 884/5 418, ti=17 3 729/3 729, ti=14 3 520/3 520, ti=18 113/113, ti=22 113/113, ti=15 113/113, ti=29 102/110, ti=4 72/105). Tous les autres ferment ZÉRO **sans porter le préfixe objet** : ti=5 **0/3 616**, ti=10 **0/2 261**, ti=9 **0/1 717**, ti=47 **0/1 716**, ti=13 8/1 491, ti=35 **0/1 364**, ti=12 0/879, ti=40 0/777, ti=21 0/357, ti=11 0/325. La COMPLEXITÉ n'est pas non plus la variable : ti=6 porte 58 composants et ferme 90,1 %, ti=47 en porte 3 et ferme 0 (dichotomie « moins de 12 composants » : 66,83 % contre 13,57 %, très en deçà de 0,85 / 44,29). **ET AUCUN des 30 composants de ti=37 n'apparaît dans un archétype qui ferme** : la famille « objet » n'a jamais été prouvée par une fermeture, nulle part. | le cadrage « le défaut est dans le préfixe objet » est CLOS ; le lot 3.6 (ti=9, 35, 40, 42, 43) et ce lot partagent la même cause inconnue |
| 2026-09-15 | 1.9.1 bis (pas 2) | **D3 (1.9.1 bis, pas 2) — LA LARGEUR DE `FUN_1406d3140` VIENT D'UNE TABLE CHARGÉE AVEC LA CARTE, INDEXÉE PAR `param_3`, ET LE DÉCODEUR EN LIT UNE SEULE. C'EST LA SEULE PISTE NEUVE DU PAS.** Relu le 2026-09-15 : le prologue de `FUN_1406d3140` fait `uVar7 = DAT_144706100` (= `0x1FFF` dans l'image) **puis**, `si (DAT_144706104 != 0)`, `uVar7 = (&DAT_1451f98d4)[param_3 * 2]` — et la garde `DAT_144706104` **vaut 0x01 dans l'image statique** (lue à l'octet). La table `DAT_1451f98d0` y est NULLE : elle est remplie AU CHARGEMENT DE LA CARTE, exactement comme les largeurs d'axe d'i0 (`WorldObjectPrecision`). Or les appelants passent des `param_3` DIFFÉRENTS : **0** (les boucles de slots, via `FUN_1408f0ac4`), **1** (la branche attachée d'i10 `object-parent-state`, où la sonde R(1) est lue — `param_3 == 1` est justement sa condition), **4** (i21 `equipment-activated`). Le décodeur lit `bitLen(0x1FFF) = 13` + 2 bits de queue pour TOUS. **NON CORRIGÉ, ET C'EST DÉLIBÉRÉ** : la table est vide dans le binaire, aucune lecture statique ne donne les trois largeurs, et D13 interdit de les deviner. Consigné à son point d'usage (`filmdec/unit_weaponstate.go`, en-tête de `defaultReplRange`). | reprise : soit une capture runtime de `DAT_1451f98d0` (Cheat Engine, comme le calibrage d'i0), soit un balayage de largeur par `param_3` jugé par la FERMETURE — c'est la première hypothèse depuis le pas 1 qui n'ait pas été réfutée |
| 2026-09-15 | 1.9.1 bis (pas 2) | **D4 (1.9.1 bis, pas 2) — DEUX EXPLICATIONS CONCURRENTES SONT ÉLIMINÉES, MESURÉES, ET LEURS INSTRUMENTS SONT VERSIONNÉS.** (a) LES ANCRES FORTUITES NE SONT PAS LA CAUSE : `TestE191cAncresFortuites` retire les records dont le `n1` (la taille de tampon que `FUN_142e2bfd0` lit à position FIXE, donc constante par archétype et par build) s'écarte du modal de leur archétype, PUIS ré-apparie les survivants entre eux — **138 retirées sur 50 384 (0,27 %)**, fermeture des cinq **184/21 594**, inchangée. (b) LA LARGEUR D'ÉTAT PAR DÉFAUT DE ti=37 N'EST PAS FAUSSE : `TestE191cEtatParDefaut` substitue un décalage de `w` bits au désérialiseur d'état par défaut pour `w` de 0 à 512 (le bouton `keyframeFullStateTemoin.SansEtatParDefaut`, qui existe pour cela) — **aucun `w` ne ferme plus de 32 records sur 3 331** (la largeur portée en ferme 3, et consomme 115 bits dans 952 records, 92 dans 348, 60 dans 319). Sur ti=38 en revanche un `w` fixe de **173** ferme 901/12 064 contre 180 pour le désérialiseur porté : à consigner, hors périmètre de ce lot. | le lot suivant sur ti=38 a une piste chiffrée ; pour ti=37 les deux hypothèses sont closes |
| 2026-09-15 | 1.9.1 bis (pas 2) | **D5 (1.9.1 bis, pas 2) — LES PETITS ARCHÉTYPES QUI NE FERMENT PAS BUTENT SUR UN COMPOSANT NON PORTÉ, PAS SUR UNE LARGEUR FAUSSE — ET ti=43 EST DANS CE CAS.** Sonde fine (`TestE191cPrefixeObjet`, tableau [D], archétypes de moins de cinq composants) : ti=25 (1 composant, `powerframe-player-selection-data`), ti=45 (2), ti=3 (2), ti=20 (3), ti=47 (3) **désynchronisent sur 100 % de leurs records** ; ti=22 (1 composant, `physics-state`) ferme 113/113 et ti=14 (3) 3 520/3 520. Les résidus énormes de ti=4 (33 records sur 105 à -42 552, -69 552 bits) et de ti=8 (0 composant, 2 records à -3 727) sont des ANCRES FORTUITES, et c'est ce qui a motivé le contrôle (a) de D4. **ti=43 est bloqué à 100 % par `i19 device-position-animation-name-component`**, dont la grammaire est relue ici — `FUN_1410156e4` (vtable `143d0cea8`) : `FUN_141015740` = R(32) identifiant puis `FUN_1406d84b4` largeur `0xa` déquantifié dans [0, 10] (`14101571c`), **42 bits inconditionnels** — mais NON PORTÉE : les 22 composants `device-*` (i19 à i40) sont le lot 3.6, et porter i19 seul ne fermerait aucun record. L'adresse et la grammaire sont posées dans `ecs_table.tsv` pour ce lot-là. | lot 3.6 : la grammaire d'i19 de ti=43 est écrite, il reste i20 à i40 |
| 2026-09-15 | 1.9.1 bis (pas 2 bis) | **D6 (1.9.1 bis, pas 2 bis) — D3 EST RESOLUE, ET SA PREMISSE ETAIT FAUSSE : LA TABLE DE PLAGES EST UNE CONSTANTE DU BINAIRE, PAS UNE DONNEE DE PROFIL PAR CARTE.** D3 supposait `DAT_1451f98d0` remplie au chargement de la carte (classe D-3, comme les largeurs d'axe d'i0). L'ECRIVAIN DIT LE CONTRAIRE : `FUN_140d10bb0` (appelee par `FUN_140d10a78`) pose `DAT_144706104 = 1` et remplit les NEUF entrees avec des LITTERAUX — base/plage `0x200`/`iVar1−0x200` pour 0 et 1, `0x200`/`0x100` pour 2, `0x300`/`0x100` pour 3, `0x200`/`0x200` pour 4, `0x400`/`0x100` pour 5, `0`/`0x200` pour 6, `0`/`iVar1` pour 7 et 8, ou `iVar1 = DAT_144706100 = 0x1FFF`. Les deux autres ecrivains (`FUN_1408f1618` @1423503d3, `FUN_142f2f0cc` @142f2f2b1) ne touchent QUE les entrees 0, 1, 7, 8 et `DAT_144706100`, si bien que les entrees 2 a 6 sont INVARIANTES. `W = FUN_1406d310c(plage)` donne donc **13** pour 0/1/7/8, **8** pour 2/3/5, **9** pour 4/6 — et la categorie 1 BASCULE sur l'entree 4 (9 bits) quand son bit de sonde vaut 1 (`DAT_1451f98f0`/`f4` = 1451f98d0 + 4×8, verifie). **CONSEQUENCE POUR LE POINT 2 DU BRIEF** : il n'y a RIEN a deriver du film ni des fichiers du jeu, donc ni section de `chunk_00`, ni `himap`, ni catalogue par carte facon `map_quant_bounds.json`. Porte dans `filmdec/varwidth.go`. | ferme D3 ; la question « derivation par carte » est SANS OBJET pour ce champ |
| 2026-09-15 | 1.9.1 bis (pas 2 bis) | **D7 (1.9.1 bis, pas 2 bis) — QUATRE SITES DE ti=37 LISAIENT TROP DE BITS, ET LE GAIN EST REEL MAIS PETIT.** `param_3` releve sur le desassemblage a chaque site (`MOV R8D, imm` avant le `CALL` ; `FUN_1408f0ac4` le transmet tel quel, `MOV R14D,R8D` -> `MOV R8D,R14D`) : **i10 `object-parent-state` branche attachee** (`FUN_140c1e4d0` @140c1e51d, categorie **1**) lisait 13 bits meme quand la sonde bascule sur l'entree 4 — **−4** ; **i21 `equipment-activated`** (`FUN_140c1dc80` @140c1dcbb, categorie **4**) — **−4** ; **i22 `equipment-control-signal`** (`FUN_14101cd94` @14101cdc3, categorie **4**) lisait en plus une sonde que le jeu ne lit pas — **−5** ; **i28 `equipment-tracked-object-handles-stack`** (`FUN_140f72dec` @140f72e41, categorie **0**) — **−1**, la sonde de trop. MESURE sur les sept bobines : fermeture des cinq archetypes objet **184 -> 188 (0,85 % -> 0,87 %)**, ti=38 180 -> 183, ti=37 3 -> 4. ATTRIBUTION par neutralisation site par site : **i10 porte tout le gain de ti=38** (173 sans elle, 183 avec), et le record supplementaire de ti=37 **n'existe que si i21, i22 ET i28 sont corriges ensemble** — la signature d'un record qui ferme parce que TOUS ses composants sont justes. **LA FONDATION N'EST PAS LA** : 0,87 % reste 0,87 %. | la correction est acquise ; la cause de la non-fermeture est ailleurs |
| 2026-09-15 | 1.9.1 bis (pas 2 bis) | **D8 (1.9.1 bis, pas 2 bis) — IL N'Y A PAS DE CAUSE UNIQUE PARTAGEE : LA POPULATION A EXPLIQUER EST DE SIX ARCHETYPES, ET AUCUN COMPOSANT NE LEUR EST COMMUN.** La colonne « bloquant » du golden 0.A.3 separe proprement : **(A)** un composant SANS LECTEUR est nomme — ti=0 et 2 (`game-engine-soft-ceilings`), 3 (`low-frequency`), 5 (`player-aim-assist`), 9 (`managed-player-forge-weather-effect-overrides`), 10 (`managed-object-navpoint`), 11 (`managed-objective-interaction-filter`), 12 (`managed-navpoint-flags`), 19 (`sound-placement-state-data`), 25 (`powerframe-player-selection-data`), 26/27 (`supply-lines-*`), 34 (`tacmap-mapdismissallock`), 35 (`simulation-state`), 43 (`device-position-animation-name`), 45 (`matchflow-sequence-data`), 47 (`personal-ai-data`) : dix-sept archetypes dont la correction est un PORTAGE (lot 3.6), pas une largeur ; **(B)** aucune desynchronisation et aucune fermeture : **ti=13, 21, 37, 38, 41, 42** — SIX, dont ti=13 (`managed-object-property-*`) et ti=21 (`flock-*`) qui ne portent PAS `object-position-component`. `TestE191cListeCourte` : ces six portent **51 composants distincts, 0 blanchi** par un archetype qui ferme, et **aucun composant n'est commun aux six**. Les couples de meme taille ne discriminent pas non plus (`TestE191cPairesDiscriminantes` : ti=19 = 32 x `sound-placement-state-data`, ti=25 = un seul composant — classe (A)). **La suite se traite archetype par archetype, pas par une cause commune.** | piste suivante : ti=42 (`ground-weapon`, 21 composants, etat par defaut VALIDE PAR ORACLE, 2 087 records) est le plus petit de la classe (B) dont tout le cadre est deja prouve — c'est la ou un residu se lit le plus proprement |
| 2026-09-15 | 1.9.1 bis (pas 2 ter) | **D9 (1.9.1 bis, pas 2 ter) — LES ETATS PAR DEFAUT DES ARCHETYPES QUI ECHOUENT SONT TOUS CONFORMES CHEZ L'ECRIVAIN, SAUF UNE HEURISTIQUE SUR LE CHEMIN CUIT.** Relus un par un : ti=38 `FUN_1408f0b48` (V + MPP `FUN_14080cfe8` + `FUN_1408f0ac4(…,0)`) · ti=36 `FUN_1407f2224` (V + MPP) · ti=43 `FUN_140fe7630` (idem) · ti=37 `FUN_1407f105c` (V + ti=36 + `FUN_1407f2058` + R(1)[si 1 : R(32)]) · ti=42 `FUN_1407f0c68` (V + ti=36 + R(12) + R(7) + liste chargeurs + ref5) · ti=13 `FUN_140ce55e8` (V + R(32) + R(1) ; si 0 -> 1×R(4), sinon 32×R(4) — et le « R8D » du desassemblage EST la constante 4, `140ce59bc : if (iVar7 < 4)`) · ti=21 `FUN_141133c24` (R(18) sec — les trois `CALL` vus en desassemblage sont du bruit de frontiere Ghidra, 0 site de lecture) · ti=9 `FUN_1410d7540` (V + R(6) + R(6) + R(1)) · ti=10 `FUN_141020244` (V + `FUN_1408f0ac4(…,0)`). **ti=41 n'est resoluble par AUCUNE lecture statique** : le descripteur d'archetype se construit au runtime (`DAT_144e61d88` nulle dans l'image) — consigne, non devine. **LA SEULE DIVERGENCE** : le bloc MPP lit `R(9)` (litteral `141fd72de : ADD [RCX+0x2c],0x9`) et `R(5)` (inline) chez l'ecrivain, alors que le depot en fait DEUX GLOBALES que la production CALIBRE et INSTALLE (`filmdec/equipment_placements.go:168`, `replay/build_ground_weapons.go:97`, `replay/build_vehicles.go:86`). `KeyframeClosure` utilise deja les defauts 9/5, donc **cela n'explique pas la non-fermeture** — mais c'est une heuristique qui contredit l'ecrivain sur le chemin CUIT. | conversion nommee demandee par le pilote : la grammaire remplace la calibration, les globales deviennent un controle puis sortent — **exige un gate de decodage, EN ATTENTE DE VOIE LIBRE** |
| 2026-09-15 | 1.9.1 bis (pas 2 quater) | **D10 (1.9.1 bis, pas 2 quater) — LES DEUX MOTS DE TAILLE SONT DES GARDES, ET LES PORTER VAUT +68 RECORDS.** `FUN_142e2bfd0` porte DEUX fois `if (0 < (int)uVar7)` : le premier garde `vtable[0x60]` (l'etat par defaut), le second garde `vtable[0x88]` PUIS `FUN_1428e2b68`, qui mene a la boucle de composants `FUN_142e2c690`. Donc **`n1 == 0` -> aucun etat par defaut n'est ecrit, `n2 == 0` -> aucun composant**, et la comparaison est SIGNEE. Le depot lisait les deux inconditionnellement. **MESURE, sept bobines : 26 lignes du golden 0.A.3 montent (+89 records), 2 descendent (-21), NET +68.** Les cinq archetypes objet passent de 184/21 698 (0,85 %) a **246/21 698 (1,13 %)** ; ti=41 de 0/110 a **34/110 (30,9 %)**, ti=42 de 1 a 10, ti=37 de 3 a 15 ; et des archetypes HORS prefixe objet montent aussi — ti=35 (bipede) 0 -> 6, ti=10 0 -> 3, ti=12 0 -> 1. Attribution : `n1` seule ne change rien sur les cinq (188), `n2` seule vaut 220, les deux ensemble 246. **Deux effets de bord, tous deux DES AMELIORATIONS** : les deux descentes (`a521164d` ti=13 8 -> 0, `60ae07c4` ti=38 43 -> 30) sont des fermetures obtenues APRES desalignement (preuve ci-dessous, D11) ; et `TestScanPlayerTeamsSurLesBobines` figeait « 1 index hors table » sur `111fa685` — ce record a `n2 == 0`, il n'a PAS de composants, l'index qu'on y lisait etait du bruit. | acquis ; `GrammarRev` .3 -> .4, `SchemaVersion` reste 59 (corpus gate du jour : 59 -> 59 sur 14 temoins, 26 gains, 1 « perte » qui est une amelioration) |
| 2026-09-15 | 1.9.1 bis (pas 2 quater) | **D11 (1.9.1 bis, pas 2 quater) — `n2` EST UN ORACLE GRATUIT, ET IL DIT QUE LE PREMIER BIT FAUX EST DANS L'ETAT PAR DEFAUT, PAS DANS UN COMPOSANT.** `n1` se lit a une position FIXE (108 bits) et `n2` APRES l'etat par defaut ; les deux sont des tailles de tampon, donc constantes par archetype et par build. Mesure sur deux bobines : **les archetypes qui FERMENT ont `n1` ET `n2` constants** — ti=14 4/28, ti=17 4/432, ti=22 12/12, ti=29 1/256, ti=6 4/7896 — **ceux qui echouent ont `n1` constant et `n2` du BRUIT** : ti=13 `n1=136` sur 454 records et `n2` ∈ {-1 ×48, 2147483392 ×24, 32768, 98304, 0, 68, 422710486, 2073479726, 1239369857, 4, 3, 2…} ; ti=37 `n1=100` sur 420 et `n2` ∈ {-1073741824 ×222, -1744002798, 1308139706…} ; ti=38 `n1=100` sur 2 098 et `n2` ∈ {0, 110, 14112, 14113, 14114, 3528, 3612672, 451709…}. **`n1` constant partout PROUVE que l'en-tete de 108 bits est exact**, y compris la ou rien ne ferme ; `n2` separe les deux populations sans exception. Relire quinze composants ne pouvait donc rien donner : le defaut est EN AMONT d'eux. **Et le garde-rail qui porte deja cet oracle (`default_state_n2_constant_test.go`) ECARTE de son jugement « les archetypes a etat VARIABLE (ti=3, 8, 10, 13, 24, 36..39, 48, le bipede) » : il etait desarme PRECISEMENT sur ceux qui echouent.** | la suite du chantier est la : faire juger ces archetypes-la par `n2` |
| 2026-09-15 | 1.9.1 bis (pas 2 quater) | **D12 (1.9.1 bis, pas 2 quater) — LE DEFAUT EST BORNE A TROIS BITS DANS UNE FONCTION NOMMEE : LE BLOC MPP `FUN_14080cfe8`.** Balayage 16 × 16 des deux largeurs du bloc contre l'oracle `n2`, deux bobines, part des records dont `n2` prend la valeur modale : ti=37 (1 187 records) **0,204 -> 0,639**, ti=38 (3 402) **0,053 -> 0,617**, ti=42 (951) **0,059 -> 0,732** — et **la MEME paire gagne sur les trois**, `lead = 8, index = 3`, avec des `n2` modaux qui deviennent des tailles de tampon plausibles (1 396, 1 764, 1 300) au lieu de 0. Trois populations independantes ne designent pas la meme paire par hasard. **CE N'EST PAS UNE AUTORISATION DE POSER 8 ET 3** : `lead = 8` contredit le litteral `R(9)` de l'ecrivain et `index = 3` le `R(5)` inline ; accorder un decodeur a une mesure est exactement ce que D13 interdit. La lecture juste est : **le bloc MPP consomme TROIS BITS DE TROP, et ces deux largeurs sont les seules molettes disponibles pour les absorber**. La part modale plafonne a 0,62-0,73 et non a 1 : le bloc porte encore un element variable au-dela de ces trois bits. | prochaine relecture chez l'ecrivain : `FUN_14080cfe8` champ par champ, en cherchant OU sont les trois bits — le defaut passe de « quelque part dans 31 composants » a « trois bits dans une fonction nommee » |
| 2026-09-15 | 1.9.1 bis (pas 2 quinquies) | **D13 (1.9.1 bis, pas 2 quinquies) — LA GRAMMAIRE DU BLOC MPP EST VERSIONNEE PAR BUILD, ET LE PORTAGE EST JUSTE POUR LES BUILDS RECENTS. LES « TROIS BITS DE TROP » DE D12 SONT UN PROFIL MANQUANT, PAS UNE LARGEUR FAUSSE.** Balayage 16 × 16 rejoue BOBINE PAR BOBINE (`TestE191cOracleN2ParBuild`), part des records dont `n2` prend la valeur modale, ti=37 (ti=38 et ti=42 donnent la MEME coupure) : `a521164d` HI_1_4_1 **8/3 → 0,995** (9/5 : 0,304) · `60ae07c4` HI_1_8_0 **8/3 → 0,988** (0,522) · `11de8353` HI_1_9_0 **8/3 → 0,990** (0,492) · `111fa685` HI_1_10_0 **8/3 → 0,993** (0,432) · `e5adf7b2` HI_1_11_0 **8/3 → 0,996** (0,472) · `bcb6d393` HI_1_12_0 **9/5 → 0,949** · `fb1a1a72` HI_1_13_0 **9/5 → 1,000**. **LA BASCULE EST A `HI_1_12_0`, ET LA VERSION MAJEURE DU FILM NE LA DONNE PAS** : `e5adf7b2` et `bcb6d393` portent tous deux `v=40` et tombent de part et d'autre. Ce qui les separe est le BUILD, et dans `chunk_00` le CARDINAL DE LA TABLE PAR TYPE — **123 d'un cote, 116 a 122 de l'autre** — qui est LISIBLE HORS LIGNE (`FilmIdentity.TypeVersions`). **Le portage 9/5 est donc EXACT sur les builds >= HI_1_12_0** (0,95 a 1,000, le meilleur score du balayage entier) ; ce qui manque est un PROFIL pour les builds <= HI_1_11_0. | la conversion MPP demandee doit devenir « le PROFIL PAR BUILD remplace la calibration, la calibration devient le controle », avec build inconnu = erreur typee (ADR 0034) |
| 2026-09-15 | 1.9.1 bis (pas 2 quinquies) | **D14 (1.9.1 bis, pas 2 quinquies) — D13 (LA REGLE) RENCONTRE SA LIMITE, ET IL FAUT LA NOMMER : L'EXECUTABLE OUVERT EST UN SEUL BUILD.** `FUN_14080cfe8` a ete relu en entier : ses largeurs sont des LITTERAUX (`141fd72de : ADD [RCX+0x2c],0x9` pour le champ de tete, `R(5)` inline pour l'index, `R(32)`, `R(1)[si 0 : R(32)]`, `R(1)[si 1 : R(18)]`, `FUN_14080d524`, `R(2)`, `R(3)` compte, boucle, queue `R(1)[si 1 : R(32) + opt32 + R(14)]`) et **le seul `if` runtime du bloc (`DAT_145121140 == 1`) ne consomme AUCUN bit** — c'est une resolution d'objet (`FUN_14074d064` / `FUN_1404785a0` / `FUN_142ad4f20`), verifiee ligne a ligne. **Il n'y a donc AUCUNE branche de version dans le bloc, et la grammaire des builds anciens n'est pas relisible chez cet ecrivain-la.** La version sur laquelle le jeu branche ailleurs — `FUN_1428e1c0c(&DAT_144c23178)`, qui gouverne le bit `DAT_144706104` (D6) — vient d'une structure RUNTIME attachee au film charge (`*(param_1 + 0x108)`, ou `*(param_1 + 0x120) + 0x130` selon `FUN_1428e1e94`) : elle vient donc DU FILM, et le film la porte dans `chunk_00`. **Lire le profil dans le film n'est pas deviner — c'est lire le film.** | pour une grammaire versionnee, « relire chez l'ecrivain » exige l'executable DU BUILD ; a defaut, le profil se lit dans le film et se MESURE par l'oracle `n2`, qui est interne au film |
| 2026-09-15 | 1.9.1 bis (pas 3) | **D15 (1.9.1 bis, pas 3) — LES DEUX ORACLES INTERNES SE CONTREDISENT SUR LES LARGEURS MPP, ET LA FERMETURE EST CELLE QUI A TORT.** La regle du pilote pour la limite de D13 est : « quand l'executable du build n'est pas disponible, un profil par build se MESURE sur le film par un oracle interne (`n2` constant, fermeture) ». **Ici les deux divergent, et il fallait le mesurer avant de choisir.** `n2` designe `8/3` sur les cinq builds <= HI_1_11_0 (part modale 0,988 a 0,996 contre 0,304 a 0,522 pour `9/5`). Poser `8/3` fait DESCENDRE la fermeture de **246 a 182** records sur les cinq archetypes objet (`ti=38` perd 62). Balayage de la FERMETURE elle-meme sur un voisinage `lead` ∈ {7,8,9,10} × `index` ∈ {3,4,5}, bobine par bobine (`TestE191cFermetureMPP`) : **elle ne designe aucun decoupage coherent** — `a521164d` prefere 9/5 (39), `60ae07c4` 7/4 (95), `111fa685` 10/4 (58), `e5adf7b2` 7/4 (28), `bcb6d393` 8/3 (69). **ET ELLE SE DISQUALIFIE ELLE-MEME** : sur `fb1a1a72`, le build de l'executable desassemble, ou `FUN_141fd72c0` ecrit `R(9)` par un litteral, la fermeture prefere `10/5` (140 records) a `9/5` (11) — elle contredit l'ecrivain la ou l'ecrivain est certain. A moins de 5 % de fermeture, maximiser un compte de fermetures revient a chercher des coincidences. **`n2`, lui, est VALIDE la ou l'ecrivain est connu** : 1,000 a `9/5` sur `fb1a1a72`. | la fermeture ne redeviendra un oracle de largeur que pres de 100 % ; d'ici la, `n2` est le seul utilisable, et il ne suffit pas a faire descendre un ratchet |
| 2026-09-15 | 1.9.1 bis (pas 3) | **D16 (1.9.1 bis, pas 3) — LE PROFIL PAR BUILD EXISTE, UN SEUL LIEU, ET LA CASE MPP DES CINQ BUILDS ANCIENS RESTE VIDE — C'EST L'ETAT REEL DE LA CONNAISSANCE.** `player_table_profile.go` devient **`build_profile.go`** et porte les DEUX donnees par build (largeur du bloc de personnalisation, largeurs MPP) avec **une provenance par ligne** dans les deux formes que la regle du pilote distingue : **RELU** (`HI_1_12_0`, `HI_1_13_0` — `FUN_141fd72c0` litteral `141fd72de`, aucune branche de version dans `FUN_14080cfe8`) et **MESURE** (l'oracle et son score). `BuildProfileFor` rend le profil ou `ErrUnknownBuild` (D-4, jamais le profil du build le plus proche) ; `BuildProfileFromFilm` le resout depuis la section 2 ; `InstallBuildProfileMPP` l'installe pour la duree d'une mesure et **n'installe RIEN quand la largeur est indeterminee** — un decoupage nul ne lirait aucune identite. `KeyframeClosure` l'appelle : fermeture **inchangee a 246/21 698**, puisque les deux builds dont la largeur est posee la portent a `9/5`, deja le defaut de paquet. **POSER `8/3` AURAIT FAIT DESCENDRE LE RATCHET DE 64 RECORDS** (D14 l'interdit) et poser `9/5` partout aurait affirme une largeur que `n2` refute : les deux gestes auraient ete une decision deguisee en mesure. Garde-rails : `build_profile_test.go` (table figee en clair, refus d'un build inconnu, et **la mutation qui rougit** — fausser le decoupage d'UN bit sur `HI_1_13_0` disperse `n2`, verifie dans les quatre sens). | ce qui leverait l'indetermination, par ordre de force : l'executable d'un build <= HI_1_11_0 (la ligne devient RELU), ou une fermeture qui vaille quelque chose sur ces archetypes |
| 2026-09-16 | 1.9.1 bis (pas 3 bis) | **D17 (1.9.1 bis, pas 3 bis) — LE FAIT TRANCHE PAR L'UTILISATEUR EST VERIFIE CHEZ L'ECRIVAIN : LA BOUCLE DE COMPOSANTS EST PILOTEE PAR LE REGISTRE DU FILM.** L'utilisateur tranche le 2026-09-16 : « les films sont independants des builds ; ils sont enregistres a l'instant T et jamais touches ensuite ; le film ne depend que de lui-meme pour expliquer au mode Theater comment le lire ». **Verifie** : `FUN_1428e2b68` — le pont entre l'en-tete d'etat complet et la boucle de composants — appelle `FUN_142e2c690(plVar1 + 1, param_2, &local_58, param_3 * 0x4100 + 8 + lVar4)`, et `0x4100 = 64 × 0x104` est EXACTEMENT 64 entrees de registre au cadrage du lot 1.2. `lVar4` vient de `*(param_1 + 0x108)` ou `*(param_1 + 0x120) + 0x130` selon `FUN_1428e1e94(&DAT_144c23178)`, c'est-a-dire selon qu'un film est charge. **La boucle lit donc le registre DU FILM, pas celui du jeu** — et l'executable ouvert lit bien les films anciens. Aucun executable ancien n'est necessaire. | le profil n'est pas « par build » : ce qui varie est ECRIT DANS LE FILM |
| 2026-09-16 | 1.9.1 bis (pas 3 bis) | **D18 (1.9.1 bis, pas 3 bis) — LES TROIS BITS NE SONT PAS DANS LES LARGEURS MPP, ET C'EST PROUVE CHEZ L'ECRIVAIN. LE PAS 3 S'EN TROUVE CORRIGE.** `FUN_14080cfe8` relu ligne a ligne : **aucune branche de version**. Toutes ses largeurs sont des LITTERAUX (9, 32, 1[+32], 1[+18], 2, 5, 3, la boucle, la queue) et son seul `if` runtime (`DAT_145121140 == 1`) ne consomme AUCUN bit — c'est une resolution d'objet. `FUN_141fd72c0` (le champ de tete, `R(9)`) n'a qu'**UN SEUL appelant**, ce bloc, donc pas de fonction soeur a `R(8)`. Et `FUN_1428e1c0c`, l'accesseur de version du film, n'a que **six sites d'appel** (`FUN_142923bdc`, `FUN_142e2bfd0`, `FUN_140ce620c`, `FUN_140ee5b04` ×2, `FUN_142982868`) — **aucun dans la chaine des etats par defaut**. **Le bloc MPP lit donc les MEMES bits pour tous les films.** Les trois bits que l'oracle `n2` mesure en moins sur les films anciens sont AILLEURS dans l'etat par defaut : le balayage les avait attribues a `lead`/`index` parce que c'etaient les deux seules molettes dont il disposait. **Ce qui reste vrai** : `n2` mesure bien un etat par defaut plus court de trois bits, la case des builds anciens reste VIDE — non parce que la largeur MPP serait inconnue (elle ne l'est pas : 9/5, RELUE) mais parce que l'ENDROIT des trois bits ne l'est pas. | la cle est dans le film et la mesure la designe (D19) ; la fonction qui la consulte est dans l'executable ouvert et reste a trouver |
| 2026-09-16 | 1.9.1 bis (pas 3 bis) | **D19 (1.9.1 bis, pas 3 bis) — LA TABLE PAR TYPE DISCRIMINE, MAIS ALIGNEE PAR LA FIN : DOUZE POSITIONS CANDIDATES.** `FilmIdentity.TypeVersions` (lot 1.5) alignee PAR LE DEBUT ne rend **aucune** position dont la version separe les deux groupes de films ; alignee PAR LA FIN elle en rend **DOUZE** (`fin-16`, `fin-26`, `fin-34`, `fin-42`, `fin-54`, `fin-63`, `fin-67`, `fin-77`, `fin-83`, `fin-88`, `fin-93`, `fin-105`), toutes a la version **1** sur les cinq films courts et a **2, 3, 4 ou 5** sur les deux autres. Les types s'ajoutent donc EN TETE de la table, et le cardinal (116 / 121 / 121 / 121 / 122 / 123 / 123) suit. Le discriminant est l'une de ces douze, et il est LU DANS LE FILM — aucune table externe a maintenir, aucun build inconnu possible. Instrument : `e191c_typeversions_research_test.go`. | prochaine session : trouver, dans l'executable ouvert, la fonction qui consulte l'une de ces douze versions et choisit une grammaire |
| 2026-09-16 | 1.9.1 bis (pas 3 ter) | **D20 (1.9.1 bis, pas 3 ter) — LE PREFIXE `V` N'EST PAS LA CLE, ET C'EST MESURE SUR LES SEPT BOBINES.** Piste (3) du pas 3 ter, fermee. `V` = `R(1)` [si 1 : `R(8)`] est une version lue DANS LE RECORD ; sa valeur est **identique partout** : `ti=37` V1 = absent (porte a 0) sur 100 % des records et V2 = absent de meme ; `ti=38` V1 = **3** sur les 1 303 / 2 098 / 2 262 / 1 365 / 1 044 / 1 837 / 2 147 records ; `ti=42` V1 = **2** sur les 696 / 250 / 151 / 141 / 186 / 420 / 216. Le prefixe ne separe RIEN. **Fait utile au passage** : le `V` de `ti=37` coute UN bit et non neuf (sa porte est a zero partout), et celui de `ti=36` qu'il enchaine aussi — l'etat par defaut de `ti=37` commence donc par DEUX bits de prefixe. Instrument : `e191c_prefixe_v_research_test.go`. | la cle est ailleurs que dans le record |
| 2026-09-16 | 1.9.1 bis (pas 3 ter) | **D21 (1.9.1 bis, pas 3 ter) — LE MOTIF « GRAMMAIRE GARDEE PAR LA VERSION DU FILM » EXISTE ET EST CARACTERISE, MAIS AUCUN SEUIL DE VERSION MAJEURE NE PEUT ETRE NOTRE DISCRIMINANT.** `FUN_1428e1c0c(&DAT_144c23178)` rend la VERSION MAJEURE du film : elle est stockee en `lVar3 + 4`, et le REGISTRE DU FILM suit immediatement en `lVar3 + 8` (50 archetypes × `0x4100` = 50 × 64 × `0x104`), ou `lVar3` vaut `*(param_1 + 0x108)` ou `*(param_1 + 0x120) + 0x130` selon `FUN_1428e1e94` — ce qui reconfirme D17. **LE MOTIF RECHERCHE EXISTE** : `FUN_140ee5b04` est un deserialiseur qui garde ses champs sur cette version comparee a des litteraux (`3 < uVar8` en `142442cb8`, `0xf < uVar8` en `142442cdf`). Et `FUN_140ce620c` est un resolveur version -> generation dont la table ne porte que **DEUX seuils, 13 et 14** (`DAT_143b86f68` = 0x0d, `DAT_143b86f6c` = 0x0e). **MAIS AUCUN SEUIL NE PEUT SEPARER NOS FILMS** : ils sont en v33 a v41, et surtout `e5adf7b2` (HI_1_11_0) et `bcb6d393` (HI_1_12_0) portent TOUS DEUX `v=40` en tombant de part et d'autre de la frontiere mesuree. Les six sites de lecture de la version majeure sont donc TOUS ecartes. | la cle est necessairement la table PAR TYPE de la section 2 ; son chargeur reste a trouver, et il ne passe PAS par `FUN_1428e1c0c` |
| 2026-09-16 | 1.9.1 bis (pas 3 ter) | **D22 (1.9.1 bis, pas 3 ter) — OU CHERCHER LE CHARGEUR DE LA TABLE PAR TYPE, ET POURQUOI LA TRACE STATIQUE S'ARRETE LA.** Les deux consommateurs du resolveur (`FUN_1406cf548`, `FUN_142f28e94`) ne sont references QUE depuis des DONNEES, a deux adresses ADJACENTES (`1436a87b0`, `1436a87d0`) : ce sont deux slots d'une meme vtable, appeles virtuellement — la trace statique s'arrete. **CE QUE LA MESURE DONNE QUAND MEME COMME CIBLE** : dans `chunk_00`, la table par type SUIT le registre (c'est `RegistryBlocks` qui l'ancre, lot 1.5) ; dans la structure chargee, le registre occupe `lVar3 + 8` sur `50 × 0x4100 = 0x145000` octets, donc la table par type, si elle y est recopiee, commence en **`lVar3 + 0x145008`**. C'est l'adresse a sonder (`search_instructions` sur ce deplacement, ou xrefs en ecriture sur la structure du film). | reprise : sonder `lVar3 + 0x145008` et, a defaut, remonter le chargeur de la section 2 par les ecritures sur la structure du film |
| 2026-09-16 | 1.9.1 bis (pas 3 quater) | **D23 (1.9.1 bis, pas 3 quater, voie B) — LA DIFFERENCE DES FILMS ANCIENS EST STRUCTURELLE, PAS UN OFFSET : AUCUN DECALAGE PUR NE MARCHE, ET LE `8/3` N'ETAIT PAS UN DECALAGE.** `n2` relu a la fin PORTEE de l'etat par defaut PLUS un decalage de -6 a +2 (`TestE191cDeficit`), part modale sur `ti=37` : `fb1a1a72` **+0 -> 1,000** (les autres decalages 0,58 a 0,72) · `bcb6d393` **+0 -> 0,949** · `a521164d` **PLAT de 0,27 a 0,32** · `60ae07c4` **PLAT a 0,52** · `11de8353` 0,49 partout (+2 -> 0,578) · `111fa685` 0,43 partout (+2 -> 0,545) · `e5adf7b2` 0,47 partout (+2 -> 0,581). **DEUX CONSEQUENCES.** (1) **Le portage est EXACT sur les films recents** : `+0` rend 1,000 sur `ti=37` ET `ti=42` de `fb1a1a72` — la lecture n'a pas a etre touchee la. (2) **Le decoupage `8/3` du pas 3, qui atteignait 0,988 a 0,996, N'EST PAS UN DECALAGE** : il deplace la fin de l'etat de exactement trois bits, or `-3` ne rend ici que 0,27 a 0,49. C'est donc une autre ANALYSE — le bloc MPP porte des champs a longueur dependante des DONNEES (`R(3)` de compte puis boucle, portes sur valeurs lues), si bien que changer une largeur de tete change les valeurs lues ensuite, donc les branches prises, donc la longueur totale. **Chercher « les trois bits » comme une constante est une impasse : c'est la CONDITION qu'il faut trouver.** PIEGE A NE PAS REFAIRE : `ti=38` rend 1,000 a TOUS les decalages sur les films recents — son `n2` y est constant quoi qu'on lise, il ne discrimine rien. | la suite ne peut pas etre un balayage : il faut la branche, chez l'ecrivain |
| 2026-09-16 | 1.9.1 bis (pas 3 quater) | **D24 (1.9.1 bis, pas 3 quater, voie A) — LA TABLE PAR TYPE N'EST PAS RECOPIEE DERRIERE LE REGISTRE : LA CIBLE `lVar3 + 0x145008` EST REFUTEE.** `search_instructions` sur le motif d'operande, **13 607 556 instructions balayees** : `0x145008` **0 occurrence**, `0x14500c` **0 occurrence**, `0x145000` deux occurrences seulement et ce sont des immediats 64 bits sans rapport (`MOV R8, 0x145000000000000` dans `FUN_141d9ef70`). Aucune instruction n'adresse ce deplacement : la table par type de la section 2 n'est donc PAS rangee juste apres le registre du film dans la structure chargee. Elle vit ailleurs, ou n'est pas chargee du tout. | la piste « offset calcule » est fermee ; reprendre par le CHARGEUR de la section 2 (ecritures sur la structure du film) plutot que par une adresse devinee |
| 2026-09-15 | 1.9.1 ter | **D1 (1.9.1 ter) — LA TABLE DE L'ÉCRIVAIN CONTREDIT LA MESURE SUR DEUX LIGNES, ET C'EST LA MESURE QUI EST PORTÉE.** L'exécutable dérive la largeur du registre et celle de la table par type de la version de format (`FUN_141cfff30` / `FUN_141cffe20`, deux `std::map` construites par `FUN_140268ec0` / `FUN_140268f40`, littéraux relus). Deux de leurs lignes sont fausses au regard des octets : **format 21** — la table annonce 117 entrées, `a521164d` en porte 116 (chaîne de build à 815 864, fin de registre à 815 368) ; **format 25** — la table annonce 25 blocs, les 39 films mesurés en portent 49 (`e5adf7b2` : chaîne de build à 815 888 = 49 blocs + 122 entrées, exactement). La dérivation structurelle de `ReadFilmIdentity` (ancrage sur la chaîne de build) ferme à l'unité sur les 1 351 `chunk_00` du cache : c'est elle qui reste le lecteur, et les deux tables ne sont PAS portées. NON TRAITÉ — instrument `TestE191tContradictionsDeLEcrivain`. | soit la règle de recherche dans la map est lue de travers (le désassemblage de `141cffe20` donne « plus grande clé <= B, et si B dépasse la dernière clé, le défaut »), soit le jeu porte un défaut latent sur ces deux formats ; départageable par un film de format 13 à 18 |
| 2026-09-15 | 1.9.1 ter | **D2 (1.9.1 ter) — LA GARDE `DAT_144706104` EST UN BIT DU FILM, ET SA VALEUR N'EST TOUJOURS PAS MESURÉE.** `FUN_142e2bfd0` @142e2c020 fait `si (7 < version_de_format) DAT_144706104 = R(1)` en tête du paquet d'image-clé, et `FUN_142987460` (chemin delta) le fait SANS condition. Les 1 351 films du cache portent un format de 20 à 27, donc **tous > 7** : la branche « garde statique à 1 » est morte pour tout film réel, et ce qui reste ouvert n'est plus « peut-on savoir si le bit est lu » (oui, toujours — `varwidth.go` corrigé dans ce lot) mais « que VAUT-il, film par film ». Cette garde commande la table de plages de `FUN_1406d3140` (i10, i21, i22, i28, les états par défaut de ti=3 et ti=38) : garde à 0, toutes les catégories lisent 13 bits ; garde à 1, elles lisent 13/8/8/9/8/9/13/13. Le portage du lot 1.9.1 bis suppose 1 PARTOUT. NON TRAITÉ — le bit se lit au premier bit du paquet, là où `frame_records.go` porte déjà `PacketPreambleBits`, et il vaut 1 sur 100,00 % des 30 418 payloads mesurés de `000d5950`. | mesurer le bit par film et le rendre à `varWidthRange` ; à traiter AVANT tout re-figeage du golden de fermeture, car il peut déplacer i10/i21/i22/i28 |
| 2026-09-15 | 1.9.1 ter | **D3 (1.9.1 ter) — LES TROIS BITS DE L'ÉTAT PAR DÉFAUT DE ti=37 NE SONT PAS UNE BRANCHE DE VERSION, ET L'EXÉCUTABLE LE PROUVE.** Relevé exhaustif : la version de format n'a que **six sites d'appel** dans tout l'exécutable (`FUN_1428e1c0c` : `FUN_142e2bfd0`, `FUN_140ce620c`, `FUN_140ee5b04` ×2, `FUN_142923bdc`, `FUN_142982868`), et leurs seuils sont **4, 7, 12, 13/14, 16** — aucun entre 25 et 27, la frontière mesurée. La table par type n'a qu'**un** lecteur indexé (`FUN_1428e1c64`, quatre sites d'instruction sur 13,6 M), un seul appelant (`FUN_141102ed0`) et **quinze** consommateurs, dont les neuf index littéraux (0x23, 0x24, 0x30, 0x59, 0x5a, 0x5b, 0x5d, 0x61, 0x72) ne séparent PAS les deux groupes (mesure : 24 index varient entre bobines, 0 discrimine). Et la chaîne `FUN_1407f105c` → `FUN_1407f2224` → `FUN_14080cfe8` → sous-lecteurs ne contient aucune de ces deux lectures. **L'exécutable lit donc l'état par défaut de ti=37 à l'identique pour un film de format 21 et un film de format 27.** Les trois bits sont ailleurs — le plus probablement en amont du record (D2), pas dans l'état par défaut. NON TRAITÉ. | la piste suivante n'est plus « quelle version branche » mais « quel bit du paquet n'est pas consommé » : D2 d'abord |
| 2026-09-15 | 1.9.1 ter | **D5 (1.9.1 ter) — `keyframe_closure.go:125` AVALE L'ERREUR DE `InstallFilmFormatMPP`.** `if restore, err := InstallFilmFormatMPP(fc.Film()); err == nil { defer restore() }` : un format inconnu y est écarté sans log ni compteur, et la mesure de fermeture continue au défaut de paquet (9/5). C'est **délibéré et documenté** (l.119-124 : « un build inconnu ne change rien et n'est pas une erreur ICI ; c'est la PRODUCTION qui doit mettre le film de côté ») et c'est **pré-existant au lot 1.9.1 ter** — `KeyframeClosure` est un instrument de mesure, pas le chemin cuit. Reste que la règle 10 du dépôt (« swallowed error ») s'y applique à la lettre. NON TRAITÉ, sur instruction du pilote. | le jour où `KeyframeClosure` publie un rapport : y porter la cause, comme `EquipmentPlacementStats.FormatSansProfil` le fait pour les poses |
| 2026-09-15 | 1.9.1 ter | **D6 (1.9.1 ter) — L'AVERTISSEMENT « FORMAT INCONNU » EST UN `slog.Warn`, PAS UN `slog.WarnContext`.** Le pilote a demandé `WarnContext` ; il n'y a **aucun `context.Context` sur ce chemin** — ni `BuildFromFilm(matchID, titleSlug, film, opt)`, ni `Options`, ni son appelant `replaybuild.BuildBytes(matchID, mapNames, filmDir, facts)` n'en portent un, et les quarante et quelques appels `slog` de `replay` sont tous des `slog.Warn`/`slog.Info` pour cette raison. Passer `context.Background()` n'ajouterait aucune clé et romprait l'uniformité du paquet ; plomber un ctx sur trois couches dépasse le périmètre. Les trois clés demandées (`match_id`, `format`, `build`) sont bien là. NON TRAITÉ. | le jour où la cuisson reçoit un ctx (M4, publication) : convertir les appels `slog` de `replay` ensemble, pas un seul |
| 2026-09-15 | 1.9.1 ter | **D4 (1.9.1 ter) — LA TABLE PAR TYPE EST ALIGNÉE PAR LE DÉBUT, ET LA CLÔTURE DU LOT 1.9.1 bis DIT L'INVERSE.** La clôture désignait « douze positions alignées PAR LA FIN » comme point de reprise, en affirmant que « par le début elle n'en porte aucune : les types s'ajoutent en tête ». **Réfuté par la mesure** : les trente premières valeurs sont IDENTIQUES sur les sept bobines et l'index 18 y vaut 2 partout, exactement comme la table NATIVE de l'exécutable (`DAT_14474cd90`, 123 entrées `{version, taille}`, index 18 = version 2). Les types s'ajoutent donc EN QUEUE, et les « douze positions » n'étaient qu'un artefact de l'alignement par la fin. `FUN_14299bcb0` le confirme mécaniquement : au chargement, les entrées au-delà du cardinal du film sont REMPLIES depuis les descripteurs runtime (`DAT_144e61d88 + 0x210 + i*8`, `vtable+0x30`). CORRIGÉ dans la doc de `film_identity.go` ; le point de reprise du 1.9.1 bis est caduc. | rien à traiter ; la correction est portée |
| 2026-09-16 | 1.9.1 bis (clôture) | **D25 (1.9.1 bis, clôture) — LE CORPUS GATE JETAIT UNE CATEGORIE ENTIERE : `BilanAxe.Changements`. CORRIGE, ET C'EST LA D5 DU LOT 1.9.3 QUI SE FERME ICI.** `replaydiff.BilanAxe` compte TROIS categories — gains, pertes, et **changements** (une valeur publiee qui BOUGE sans etre ni l'un ni l'autre). `cmd/replay-corpus-gate/report.go:52-54` (`bilanDepuisRapport`) sommait `b.Gains` et `b.Pertes` et s'arretait la : la troisieme n'etait pas seulement ABSENTE DE L'AFFICHAGE, elle etait **JETEE** — elle n'atteignait ni le tableau, ni le JSON, ni le statut. **Un temoin dont une valeur publiee bougeait sortait `ok`**, et le lot suivant heritait d'un changement que personne n'avait classe. CORRIGE dans ce lot au titre de la regle 7 (un fix hors perimetre est admis quand il bloque le gate de l'etape courante — ici il rendait le verdict inverifiable) : `Changements` traverse desormais `ligneRapport` -> colonne `chang.` du tableau -> champ `"changements"` du JSON -> `aUnePerte()`, donc **un changement classe le temoin en `PERTE`** et force sa classification. Garde-rail : `cmd/replay-corpus-gate/changements_guard_test.go`, trois tests qui tiennent les trois maillons ; **mutation verifiee** — retirer `changements += b.Changements` fait rougir `TestBilanPorteLesChangements`, restaurer par nom le rend vert. **VERDICT DU GATE UNE FOIS L'ORACLE REPARE : `chang.` = 0 sur les 14 temoins.** | l'oracle est desormais complet ; tout lot ulterieur voit les trois categories |
| 2026-09-15 | 1.9.2 | **D1 (1.9.2) — LE CHEMIN DES TOUCHES EST ÉTEINT EN PRODUCTION POUR INFINITE, ET CE LOT LE CONVERTIT QUAND MÊME.** Sur pièces : `games.CapWeaponAccuracy` vaut `CapNotExposed` (`games/halo_infinite/adapter_data.go:200`) et `config/titles/halo_infinite/mappings/capabilities.toml:80` dit `"match.weapon.accuracy" = "not_exposed"`, donc `collectHits` sort à sa première ligne ; `ConfigureFilmAccuracy` n'a AUCUN appelant de production (grep : un seul site, `hits_capability_test.go:56`). La conversion de `BuildBipedTracks` / `FilmWeaponHitDistance` / `DetectFilmMapEntry` n'a donc **aucun effet mesurable en production aujourd'hui** — elle est faite parce que le code existe, qu'il décidait un fait à la place du catalogue, et que D13 ne connaît pas d'exception « chemin éteint ». **NON TRAITÉ** : rallumer la passe est hors lot. | le lot qui rallume la précision par arme (déjà nommé comme cible de retrait dans deux allowlists d'`archlint`) |
| 2026-09-15 | 1.9.2 | **D2 (1.9.2) — SUR LIVE FIRE, LA SIGNATURE DE LARGEURS NE RETROUVE AUCUNE ENTRÉE DU CATALOGUE : LES DISTANCES DE TOUCHE Y SONT DÉJÀ DÉSACTIVÉES.** `DetectFilmMapEntry` identifie la carte en comparant `e.AxisWidths == lay.AxisW` ; la mesure de ce lot établit que l'auto-détection rend `[13 12 11]` sur les deux films Live Fire quand le catalogue dit `[12 12 11]` — donc `0 carte` trouvée, `ErrUnknownMapBounds`, distance nulle. Ce n'est pas une régression de ce lot (le comportement est antérieur) et il ne l'aggrave pas ; c'est un CAS D'ESPÈCE de ce que le lot 1.9.4 ferme, puisque le collecteur résout déjà le nom de carte du match par la base (`resolveMapBounds`). **NON TRAITÉ.** | lot 1.9.4 (la carte du film vient du nom de match) |
| 2026-09-15 | 1.9.2 | **D3 (1.9.2) — LE CONTRÔLE « ACCORD / CONTRADICTION » DE L'AUTO-DÉTECTION EST UN ORACLE DE TEST, PAS UN COMPTEUR DE PRODUCTION, ET C'EST D2 DU PLAN QUI LE DIT.** Le brief demandait de compter l'accord et la contradiction « dans les stats de collecte ». Les poser exigerait de FAIRE TOURNER `DetectI0LayoutOf` à chaque cuisson en plus du catalogue — six chunks marchés bit à bit, plus sa propre bande de slots — c'est-à-dire précisément les « deux passes de détection supprimées par film » que ce lot compte comme gain, et une dette posée à l'endroit où le chantier en retire (S1 : zéro variable de paquet dans le décodeur, donc un porteur de compteur à inventer). D2 (§1.3) tranche sans ambiguïté : « là où le profil sait, l'inférence est un ORACLE DE TEST ; là où il ne sait pas, erreur typée » — et `map_bounds.go` écrivait déjà cette doctrine (« le découpage lu dans le film reste le CONTRÔLE, jamais l'entrée »). Le contrôle vit donc dans `filmdec/e192_i0_catalogue_mesure_research_test.go` (17 films, garde `CHUNK00_FILMS`) et dans les deux tests permanents de mutation. Item 2 du brief statué `[~]`. **NON TRAITÉ** en tant que compteur. | sans objet, sauf si un lot de profil (M3) fait passer la détection en contrôle GRATUIT — elle le devient si un jour le profil du build la porte déjà |
| 2026-09-15 | 1.9.2 | **D4 (1.9.2) — LA MESURE DU 2026-09-03 SE REJOUE À 26, PAS 27, SUR LE MÊME FILM ET LE MÊME DÉNOMINATEUR.** `60ae07c4` : la population d'enregistrements bruts (tag exigé, aucun filtre de saturation, de vitesse ni d'isolement) vaut **267 400** sous l'auto-détection et **267 374** sous le catalogue. Le dénominateur est retrouvé À L'UNITÉ, ce qui identifie le jeu de réglages du 2026-09-03 ; l'écart d'UN enregistrement n'est pas expliqué — la mesure d'origine (`film_context.go:33-42`) ne consigne pas ses options. Sous les réglages de PRODUCTION des positions l'écart vaut 3, sous ceux des pistes de touche 16 : les trois chiffres mesurent trois populations, et les confondre est le piège que cette entrée existe pour nommer. Aucune décision ne repose sur l'unité manquante. **NON TRAITÉ.** | sans objet — écrit pour qu'un futur lot ne cherche pas à retrouver « 27 » |
| 2026-09-15 | 1.9.3 | **D1 (1.9.3) — LA LECTURE SE TAIT SUR 81 DES 281 KILLS SANS MORT EN FACE, ET LA MOITIÉ DE CE SILENCE A UNE CAUSE NOMMÉE.** 46 des 81 muets viennent de trois films dont la table des joueurs n'épingle rien ou presque : `a349fea8` (31, v33 sans section), `a521164d` (12, HI_1_4_1) et `50247b26` (3, v31 sans section). Sur `a521164d` les 24 indices SONT épinglés et le film ne rend pourtant que **30 kill-events pour 101 kills** : ce n'est pas l'identité qui manque, c'est la CHAÎNE D'ÉVÉNEMENTS qui s'arrête tôt sur ce build (déjà nommé D11 (0.A.2) : HI_1_4_1 ferme dix fois moins que les autres). Les 35 muets restants sont répartis sur les films récents et suivent le même mécanisme : `evChainLen` n'atteint pas le kill-event. **NON TRAITÉ.** | lot 3.1 (build inconnu / profil) et 3.6 (composants manquants) ; le compteur `killsource_couple_muet` est la mesure de progrès |
| 2026-09-15 | 1.9.3 | **D2 (1.9.3) — LA FENÊTRE DE 2,5 s SURVIT, ET ELLE EST MAINTENANT DU CÔTÉ DE LA LECTURE.** `tolMS` ne décide plus le COUPLE, mais il reste le seul moyen de rapprocher un kill-event d'un instant du kill-feed : la lecture le franchit pour trouver son enregistrement. Le lot le rend inoffensif en consommant d'abord les couples écrits au même instant (mesure : 13 ambigus -> 1), il ne le supprime pas. C'est exactement l'objet du lot 1.9.7, qui remplacera la fenêtre par l'identité de paquet `(chunk, pidx)` — que `killEventRec` porte DÉJÀ (`assist.go`) et que `Kill` ne transporte pas. **NON TRAITÉ.** | lot 1.9.7 (dead-state et kill-feed appariés par l'identité de paquet) |
| 2026-09-15 | 1.9.3 | **D3 (1.9.3) — `CoupleStats.Contradiction` VAUT ZÉRO SUR LES 21 FILMS, ET SA BRANCHE N'EST EXERCÉE QUE PAR UN TEST.** Le cas « le film nomme une victime HUMAINE dont aucun instant voisin ne porte la mort » est prévu (la lecture prime, le xuid vient de la table du feed, l'écart se compte) mais ne se produit sur aucun film du corpus — cohérent avec « le kill-feed est humain-seul et porte TOUTES les morts humaines ». Le compteur existe pour que ce zéro soit MESURÉ et non construit. **NON TRAITÉ** (rien à corriger ; écrit pour qu'un futur lot ne prenne pas ce zéro pour une absence de code). | sans objet ; le compteur alerte si un build le fait bouger |
| 2026-09-15 | 1.9.3 | **D4 (1.9.3) — LE BACKLOG DE REDÉCODAGE N'EST PAS ÉLARGI, IL EST LE MÊME QU'AU LOT 1.8.** L'oracle (`pre-chaine-2026-09-09`, lecture seule) montre 1 384 matchs porteurs de lignes de kill, **aucun** sous `killsource-2026-09-14` : la recuisson que 1.8 a inscrite n'a pas été jouée (D6). La montée `-> killsource-2026-09-15` ne recrute donc aucun match supplémentaire. **NON TRAITÉ** — geste de production, réservé au pilote. | bloc « Clôture M1 », point 4 |
| 2026-09-15 | 1.9.3 | **D5 (1.9.3) — LE BILAN DU CORPUS GATE N'IMPRIME QUE `gains` ET `pertes` ; LA CATÉGORIE `changements` N'APPARAÎT NULLE PART.** `replaydiff.BilanAxe` compte QUATRE catégories (`Pertes`, `Gains`, `Changements`, `Identiques`) et `bilanDepuisRapport` (`report.go:51`) n'en remonte que deux ; `PertesDetail` (`report.go:41`) ne retient que les sens PERTE et DISPARU. Conséquence : une valeur publiée qui CHANGE sans perdre ni gagner — exactement ce qu'un lot de conversion produit — sort « 0 / 0 » et se lit comme « rien n'a bougé ». Le lot l'a contourné par une comparaison octet pour octet des deux artefacts cuits (`--work-root` + `--keep-work` puis `cmp`), mais le gate lui-même ne le dit pas. **NON TRAITÉ** (règle 7). | lot de durcissement des gates (candidat au jalon M2) : imprimer `changements` dans le tableau, ou faire porter au rapport JSON le détail des trois sens |
| 2026-09-16 | pilote | **D1 (CI, après 1.9.3) — LE PAQUET `filmdec` DÉPASSE LE BUDGET `-timeout 60s` DU JOB UNITAIRE SUR LE RUNNER WINDOWS.** Run 34968331540 sur 4abf6f466 : `Go Build + Test (windows-latest)` rouge, `panic: test timed out after 1m0s` dans `kfScanNext` (`keyframe_world.go:170`) ; ubuntu vert, les sept autres jobs verts. Mesure locale (CGO_ENABLED=0, `-count=1`) : le paquet vaut **26,1 s**, dont sept tests entre 2,5 et 5,3 s qui REBALAYENT chacun les images-clé des 7 bobines (`TestE191bCarteTI37` 5,25 s, `TestE191bFermetureAvecCarte` 5,17 s, `TestEtatParDefautN2Constant` 2,65 s, `TestKeyframeClosureRatchet` 2,63 s, `TestScanPlayerTeamsSurLesBobines` 2,55 s, `TestScanPlayerTeamsTemoinDUnBit` 2,54 s, `TestGoldenMiniBobineFamilles` 2,52 s) ; le runner Windows est deux à trois fois plus lent que le poste. Le paquet compte 137 fichiers `*_research_test.go` et grossira à chaque lot de la famille 1.9. Réparation immédiate (pilote, sur l'intégration) : budget du pas `go test (unit, no CGo DB)` porté de 60 s à 300 s, le budget que tenait l'ancien job baseline, commentaire daté dans le workflow. **À la fusion du lot 1.9.1 bis (2026-09-16) : 89,9 s en local**, dont dix instruments `TestE191c*` pour 62 s (`TestE191cEtatParDefaut` seul 24,5 s, `TestE191cPrefixeObjet` 7,5 s, `TestE191cPrefixeV` 7,2 s) ; budget porté à 600 s, celui du job de couverture. **RÉSOLU le 2026-09-16 (pilote)** : le job de couverture Linux a rougi à son tour (run 34978931976, `FAIL filmdec 600.236s`, 33 min de job sur 45) ; les 14 fichiers E191b / E191c (17 tests, 72 s) sont passés sous `//go:build research`, compilés en CI par `go vet -tags=research`, quatre helpers partagés déplacés dans `e191_helpers_test.go` ; `filmdec` = **15,0 s**. Taguer TOUS les `*_research_test.go` (223) a été essayé et ÉCARTÉ : 97 satellites cascadent dont de vrais gates — le démêlage des helpers est un lot de M2. | M2 : démêler les helpers des 223 instruments pour poser le ratchet par nom (modèle `gamefiles_tag_test.go`) ; règle §2.3 en vigueur d ici là |
| 2026-09-16 | pilote | **D2 (pilote, vague 2) — LE HARNAIS `replay-equiv` N'AFFICHE QUE LA PREMIÈRE ÉTAPE DIFFÉRENTE PAR FILM, ET SUPPRIME LES TSV PRODUITS EN SORTIE.** `cmd/replay-equiv/parent.go` (`comparer`, `return` à la première ligne divergente ; `os.MkdirTemp` + `defer os.RemoveAll`) : la sortie du lot 1.9.10 annonçait « une seule étape » alors que deux bougeaient (`vehicles` et `artifact`) ; tout lot qui classe depuis la seule sortie du harnais se trompe. Parade des lots 1.9.10 et 1.9.9 : rejouer le rôle `-child -film <f> -out <tsv>` et joindre les 53 étapes. | M2 (durcissement des oracles, avec D5 (1.9.3)) : le harnais rend TOUTES les étapes divergentes par film et garde les TSV produits sous une racine nommée |
| 2026-09-15 | 1.9.4 | **D1 (1.9.4) — SUR LIVE FIRE, LA SIGNATURE NE « NE TROUVAIT RIEN » : ELLE TROUVAIT `AQUARIUS`. D2 (1.9.2) EST CORRIGEE.** D2 (1.9.2) affirmait que « sur Live Fire la signature de largeurs ne retrouve AUCUNE entree du catalogue : les distances de touche y sont DEJA desactivees ». Mesure du 2026-09-15 (`TestE194SignatureContreNomDeMatch`, 17 films) : l'auto-detection rend `13/12/11` sur les deux films Live Fire, et `aquarius` porte EXACTEMENT ces largeurs au catalogue (`ctf_aquarius`, verifie a la ligne). `len(hits)` valait donc **1**, pas 0 : `DetectFilmMapEntry` rendait l'entree d'aquarius, et les distances de Live Fire etaient calculees dans une AABB etrangere — Aquarius fait 77,8 x 46,2 x 18,1 m, Live Fire 63,2 x 63,8 x 22,9 m, et leurs origines different — avec en plus un decoupage d'i0 decale d'un bit sur l'axe X. Ce n'etait pas une degradation silencieuse, c'etait une VALEUR FAUSSE silencieuse, ce qui est la categorie que ce chantier traite en premier. **CORRIGE PAR CE LOT** (la carte vient du nom de match) ; l'entree existe pour que D2 (1.9.2) ne soit pas relue comme vraie. | sans objet — correction d'une entree anterieure, faite dans ce lot |
| 2026-09-15 | 1.9.4 | **D2 (1.9.4) — LES « 6 CARTES JUMELLES » DU PLAN SONT 68 SUR 79, ET LA SIGNATURE N'IDENTIFIE QUE 11 CARTES.** Le plan (item 1.9.4) et l'audit 0.E (A2) annoncaient « 3 paires mesurees, F.0 §6 reserve 2 » — six cartes. Mesure du catalogue versionne (`TestE194CartesJumellesDuCatalogue`, sans film, en CI) : les 79 entrees se repartissent en classes de meme `axisWidths`, et **cinq classes en contiennent plusieurs** : `15/15/17` **59 cartes**, `17/17/15` 3, `14/13/15` 2, `15/15/15` 2, `18/18/17` 2 — **68 cartes indistinguables, 11 identifiables**. Le rapport F.0 n'avait pas tort, il mesurait autre chose : il cherchait un TEMOIN d'invariance d'echelle par film et s'arretait au premier trouve. La cause est structurelle : `W = min(26, ceilLog2(ceil(60*etendue)))` quantifie l'etendue au facteur 2 — toutes les cartes d'arene d'Infinite tombent dans la meme case. **CONSEQUENCE POUR LE CHANTIER** : aucune identite de carte ne peut jamais se deriver des largeurs d'axe, sur aucun chemin. Si un lot de profil (M3) veut identifier la carte SANS la base, il lui faudra un autre signal du film. | lot 3.x (profil par carte) : ne pas repartir de la signature de largeurs ; consigne ici pour qu'un futur brief ne la reprenne pas |
| 2026-09-15 | 1.9.4 | **D3 (1.9.4) — LA SIGNATURE N'A PAS ETE RETROGRADEE EN REPLI, ET C'EST LA MESURE QUI L'A DECIDE CONTRE LE BRIEF.** Le brief du lot demandait d'en faire un repli nomme (D14) pour le seul cas ou le nom de carte manque. Applique tel quel, cela aurait inscrit au registre une mecanique mesuree **juste 2 fois sur 17** (2 accords, 13 ambiguites, 2 desaccords) et **fausse sur Live Fire**. D14 (d) le dit en toutes lettres : « un repli bancal qui se declenche a tort corrompt un fait que la lecture aurait donne juste » — et D14 (b) exige qu'un repli s'ouvre sur un diagnostic typé, pas sur une devinette. La signature est donc **SUPPRIMEE** (`DetectFilmMapEntry` retiree, `DetectI0Layout` sans appelant de production) et l'absence de nom est une ERREUR TYPEE COMPTEE (`killsource_hits_matchs_sans_nom_de_carte`), au meme titre que la carte hors catalogue (`killsource_hits_cartes_hors_catalogue`, D-4) et que le cablage absent (`killsource_hits_carte_non_cablee`). Les distances sont alors desactivees, les touches restent comptees — la degradation existante, deja nommee au registre. **A VALIDER PAR LE PILOTE** : c'est un ecart assume au brief, pris sur la mesure. | decision du pilote a confirmer ; si le repli etait exige malgre tout, il faudrait d'abord expliquer ce qu'un mecanisme juste 2 fois sur 17 protege |
| 2026-09-15 | 1.9.4 | **D4 (1.9.4) — LE CONTROLE « ACCORD / CONTRADICTION » RESTE UN ORACLE DE TEST, POUR LA RAISON EXACTE DE D3 (1.9.2).** Le brief demandait de compter l'accord et la contradiction entre signature et nom. Les poser en production exigerait de faire tourner `DetectI0Layout` a chaque passe — six chunks marches bit a bit, plus sa propre bande de slots — c'est-a-dire de garder le cout de la detection qu'on vient de supprimer, pour comparer une lecture fiable a une mecanique mesuree fausse. D2 du plan (§1.3) tranche : « la ou le profil sait, l'inference est un ORACLE DE TEST ». Le controle vit donc dans `filmdec/e194_carte_par_nom_mesure_research_test.go` (17 films, garde `CHUNK00_FILMS`, sous `-tags research` depuis la regle du 2026-09-15) et dans les deux mutations permanentes de `killcollector/hits_carte_par_nom_test.go`. **NON TRAITE** en tant que compteur de production. | sans objet — meme motif que D3 (1.9.2) |
| 2026-09-15 | 1.9.4 | **D5 (1.9.4) — LE GAIN DE CE LOT EST NUL EN PRODUCTION AUJOURD'HUI, ET IL FAUT LE DIRE TEL QUEL.** D1 (1.9.2) reste vraie et re-verifiee sur pieces : `"match.weapon.accuracy" = "not_exposed"` (`config/titles/halo_infinite/mappings/capabilities.toml:80`, `games/halo_infinite/adapter_data.go`), donc `collectHits` sort a sa premiere ligne ; et `ConfigureFilmAccuracy` n'a **aucun appelant de production** (grep : deux sites, tous deux dans `hits_capability_test.go`). Les distances de touche ne sont donc calculees pour **aucun match**. **CE QUE LE LOT CHANGERAIT SI LA PASSE ETAIT RALLUMEE, MESURE** : sur les 17 films temoins, 15 passent de « pas de distance » (13 ambiguites) ou « distance FAUSSE » (2 Live Fire) a « distance juste » ; a l'echelle du catalogue, 68 cartes sur 79 sortent de l'ambiguite. **CE QUE LE LOT CHANGE REELLEMENT AUJOURD'HUI** : une auto-detection de moins dans le decodeur, un site de resolution de carte au lieu de deux, une erreur typee comptee sur carte inconnue, et `DetectI0Layout` hors production sous ratchet. | le lot qui rallume la precision par arme (deja nomme comme cible de retrait dans deux allowlists d'`archlint` et au registre des replis) |
| 2026-09-15 | 1.9.13 | **D1 (1.9.13) — AUCUN DES 8 BUILDS NE PORTE DE FRONTIÈRE DE MANCHE POSABLE, SI BIEN QUE LA LECTURE « FIN DE MANCHE » EST CÂBLÉE MAIS NON MESURÉE.** `objectiveevents.ResolveRoundBounds(...).Starts()` rend **0 borne sur les huit builds**, `fb1a1a72` (CTF, 3 manches) compris — et c'est cohérent, pas surprenant : `round_bounds.go` nomme `fb1a1a72` parmi les TROIS films du parc dont l'étiquetage de manche est faux, qu'il laisse volontairement INTACTS. Conséquence à lire honnêtement : la colonne « fin de manche » du tableau des coupures vaut 0 partout, et ce zéro dit « aucune borne n'était posable », jamais « aucune coupure ne tombe sur une manche » (l'instrument publie `bornesDeManchePosees` à côté, pour que les deux ne se confondent pas). La branche est couverte par un test unitaire (`TestUneFinDeMancheCoupeLaVie`, mutation jouée), pas par une mesure de film. **NON TRAITÉ.** | le corpus gate (12 témoins, dont des multi-manches à étiquetage sain) est le premier endroit où ce chemin peut être MESURÉ ; sinon un lot M2 sur l'étiquetage de manche |
| 2026-09-15 | 1.9.13 | **D2 (1.9.13) — DEUX CONSOMMATEURS GARDENT L'ANCIENNE DÉCOUPE, ET CE N'EST PLUS LA MÊME QUE LA DÉCOUPE PUBLIÉE.** `replay/killpos_opening.go` (`livesBySlot`, le « même vie » qui valide l'ouverture d'un kill) et `replay/film_inputs.go:184` (`buildScopedLookup`, la tenue de lunette) appellent encore `buildLifeSpans` directement, c'est-à-dire l'ÉCHAFAUDAGE. Depuis ce lot, leurs vies sont donc plus courtes que celles du registre : sur `000d5950`, 105 séjours contre 99 vies. L'effet attendu est conservateur (`sameLife` refuse plus souvent, `OpeningOutOfLife` monte plutôt qu'il ne baisse), mais deux définitions de « une vie » coexistent dans le même paquet — exactement le genre de divergence que ce chantier ferme ailleurs. **NON TRAITÉ** (hors périmètre : l'item porte sur les vies PUBLIÉES). | lot M2 (pas 5, réorganisation de `film/`) ou un lot court dédié : faire passer les deux par `IdentityRegistry.Vies()` |
| 2026-09-15 | 1.9.13 | **D3 (1.9.13) — VERDICT SUR D5 (1.6) : LES BORNES DE SCÈNE NE BOUGENT PAS, ET LE POINT À −216 m DE `084a804d` SUBSISTE PAR CONSTRUCTION.** La conversion ne retire ni n'ajoute AUCUN point : elle recolle des vies. Contrôle sur pièces : le `git diff` des huit goldens d'assemblage ne touche aucune ligne de bornes, et `29221 point(s) de grille` est identique avant / après sur `000d5950`. Le point isolé qui élargissait `bounds.minX` de −32,21 à −216,30 n'était donc PAS un artefact de découpe : il est le premier point d'une vie que le film écrit, et l'échantillon d'un seul point qui le portait est devenu le premier point d'une vie plus longue. **CONFIRMÉ AU CORPUS GATE (14 témoins, 2026-09-15) : AUCUNE perte sur `bounds`, sur AUCUN témoin — `084a804d` compris**, celui-là même qui porte le point à −216 m. C'est un FAIT DU FILM à instruire (cadrage), jamais à filtrer. **NON TRAITÉ.** | lot de rendu / cadrage (la question devient « comment cadrer une scène dont un point légitime est à 184 m du nuage », pas « d'où vient ce point ») |
| 2026-09-15 | 1.9.13 | **D4 (1.9.13) — LA RÈGLE DE LA VAGUE (SCHÉMA FIGÉ) LAISSE DEUX GOLDENS ROUGES, ET C'EST MÉCANIQUE.** Le lot change la FORME du document (`coverage.tracks.gaps` / `.gapMs`, `Point.G`) et son CONTENU (moins de vies, plus longues), mais le brief commun de la vague 2 interdit de monter `SchemaVersion` et de régénérer la fixture web — le pilote monte le schéma UNE fois à la fusion. Or `TestDocumentShapeRegenerate` REFUSE de régénérer tant que la version n'a pas monté (c'est tout son ratchet), et `TestContractFixturesMatchCommitted` compare des fixtures que la règle interdit de réécrire. **`TestDocumentShapeMatchesGolden` et `TestContractFixturesMatchCommitted` sont donc ROUGES à la clôture de ce lot**, et ils le seront pour tout lot de la vague qui touche le document. Les huit goldens d'assemblage, eux, ont leur propre porte sans garde de schéma : ils SONT régénérés. **NON TRAITÉ — geste du pilote.** | à la fusion de la vague : monter `SchemaVersion` (59 → 60) + entrée de chronique listant les champs de tous les lots, puis `REPLAY_CONTRACT_UPDATE=1 go test …/replay/ -run 'DocumentShape\|ContractFixtures' -update`, `make openapi-gen`, `make generate-types` |
| 2026-09-15 | 1.9.13 | **D5 (1.9.13) — VERDICT SUR D4 (1.6) : LES VIES SANS NOM ÉTAIENT BIEN DES FRAGMENTS, ET ELLES SE RECOLLENT — SAUF SUR UN FILM SANS SECTION D'IDENTIFICATION.** Mesure au corpus gate (14 témoins) : sur `084a804d`, le témoin même de D4 (1.6), les vies sans nom passent de **154 à 66** (`livesTotal` 353 → 263, `livesNamed` 199 → 197) ; sur `4f77afc1` de **183 à 72** ; sur les onze autres témoins à fusions, `livesNamed` est INCHANGÉ alors que `livesTotal` baisse, donc `unnamedLives` baisse partout. La réponse à la question de D4 (1.6) est donc OUI : une vie sans nom que le seuil de publication révélait est, dans l'écrasante majorité des cas, un FRAGMENT de vie nommée, et elle disparaît quand la vie cesse d'être coupée. **MAIS `a349fea8`, le SECOND témoin de D4 (1.6), ne bouge pas d'un octet de découpe** (0 perte, 0 changement) : ce film ne porte AUCUNE section d'identification, le registre n'y rend aucune vie, le repli garde donc le seuil — ses 337 vies sans nom restent. C'est une limite de BUILD, pas de la conversion, et elle est nommée ici plutôt que tue. **NON TRAITÉ** (la décision produit du 2026-09-06, « les vies anonymes n'existent pas », reste ouverte pour ce film). | M3 (divergences par build) ou le lot qui donnera une table de joueurs aux films sans section d'identification ; arbitrage utilisateur si `a349fea8` doit être nommé avant la recuisson du parc |
| 2026-09-15 | Revue M1 — L6 | **L6-1 — LA MOITIE VICTIME DU COUPLE ECRIT N ETAIT TESTEE PAR RIEN.** `killsource/feed_couples.go:146` (`chercherCoupleEcrit`) : mutation `v == victime` -> `v != "" && victime != ""` jouee, paquet `ok`. Cause : `feed_couples_test.go` ne passe jamais plus d UN `killEventRec`, si bien que comparer la victime ou non rend le meme enregistrement. TRAITE dans ce lot (tests seuls, aucun code de production touche). | FERME par `killsource/feed_couples_victime_test.go` (deux kills du meme tueur dans la fenetre de 2,5 s, l orphelin ne recoit plus la victime de l autre mort) |
| 2026-09-15 | Revue M1 — L6 | **L6-2 — LA GARDE DE DIVERGENCE D INDEX N ETAIT TESTEE PAR RIEN.** `filmdec/player_teams.go:241` (`publierEquipes`) : `continue` retire apres `rep.IndexDivergences++`, et le SEUL rouge du paquet etait `TestGrammarRevSuitLaGrammaire`, c est-a-dire l empreinte des sources. Cause : les sept bobines versionnees rendent 0 divergence — un invariant qu aucune donnee n exerce se tient par une fixture. TRAITE dans ce lot. | FERME par `filmdec/player_teams_divergence_test.go` (assertions sur l ABSENCE de l index, jamais sur sa valeur : sans la garde, `out[idx]` depend de l ordre d iteration d une map) |
| 2026-09-15 | Revue M1 — L6 | **L6-3 — L ORDRE DES CANDIDATS DE CARTE (mecanisme du repli `repli_carte_premier_nom_resolu`) N ETAIT TESTE PAR RIEN, ET LE CRITERE DE RETRAIT EST DESORMAIS MESURE.** `sync/killcollector/map_identity.go:94` : boucle parcourue a l envers, tout vert (l empreinte `KillSourceDecoderRev` ne hache pas `killcollector`). MESURE sur le parc oracle (1 967 matchs) : **7 matchs a deux candidats distincts, `0` ou les deux resolvent des entrees DIFFERENTES** — les seconds candidats sont des UUID bruts. Le critere « 0 match du parc ou deux noms candidats resolvent DES ENTREES DIFFERENTES — mesure a faire » du registre est donc TENU. Le registre des replis n est PAS modifie ici (un autre lot le tient) : NON TRAITE. | le lot qui tient `replay/fallback/registre_killsource_carte.go` : y reporter la mesure, et l arbitrage `asset_translations` contre `match_registry.map_name` reste hors perimetre de la famille 1.9 |
| 2026-09-15 | Revue M1 — L6 | **L6-5 — L ORDRE « XUID D ABORD, PONT SLOT ENSUITE » N ETAIT TESTE PAR RIEN.** `replay/player_teams.go:160` (`poserSurLesTraces`) : voies permutees, et seuls les deux goldens deja rouges par construction rougissaient. Cause : sur les bobines versionnees les deux voies s accordent partout ; un ordre ne se teste que la ou les sources divergent (siege recycle). TRAITE dans ce lot. | FERME par `replay/player_teams_ordre_test.go` (trois temoins : le lien direct prime, le pont reste la seule voie d un bot, le silence garde `-1`) |
| 2026-09-15 | Revue M1 — L6 | **L6-6 — LA PORTE `-update` DU GATE D EMPREINTE RENDAIT `ok`.** `sync/killcollector/decoder_rev_fingerprint_test.go:205-208` ecrivait le golden puis `return` : `go test` jette la sortie d un paquet qui passe, donc la reecriture du couple (revision, empreinte) etait INVISIBLE sans `-v` — le defaut exact que les deux autres portes du chantier (`-update-keyframe-closure`, `-update-grammar-rev`) refusent depuis la revue R2 (C1). TRAITE dans ce lot (edition minimale d un `_test.go` existant). | FERME : `t.Fatalf` apres reecriture, message d usage aligne sur les deux autres portes |
| 2026-09-16 | Revue M1 — L4 | **CONSTAT 1 (P0) TENU — la porte de publication ligne par ligne ne comptait QU'UN COTE du probleme d'affectation.** `Result.BijectionDetermined` valait `roster.table.Inferred <= 1` (`decode.go:187`), sur la premisse « un seul indice libre, donc une seule affectation possible ». Verifie sur pieces : la premisse est fausse DEPUIS LE LOT 1.8 qui l'a posee — `pinUnSiege` (`roster.go:207-234`) ajoute au roster un nom que le kill-feed ignore sans que `nPlay` bouge quand l'indice est deja dans l'espace, donc `len(freeNames) - len(free) = len(names) - nPlay > 0` ; `hungarianStart` (`bijection.go:160-176`) PADDE d'ailleurs la matrice pour cette raison, en citant `111fa685` (25 joueurs, 24 indices). Un indice libre pour deux noms libres se tranche alors par les votes, tous nuls pour un joueur sans kill ni mort ; `refine` n'a pas deux indices a echanger et `bijectionMargin` (`bijection.go:265-284`) rend zero par construction. **CORRIGE** : `FilmTablePinning.FreeNames` + `FilmTablePinning.AffectationUnique()`, compteur d'exploitation `killsource_bijection_noms_libres_en_trop`, godoc de `freeSlots` (qui affirmait l'egalite des deux listes « par construction ») corrige. Tests `TestUnIndiceLibrePourDeuxNomsLibresNEstPasDETERMINE` + `TestAffectationUniqueNeRegardePasQueLesIndices` ; mutation `case 1: return true` rouge puis restauree. `KillSourceDecoderRev` `killsource-2026-09-15` -> `killsource-2026-09-16` (ratchet regenere), `GrammarRev` `.11` -> `.12`. sha `7e08b416d` | **POPULATION NON MESURABLE A L'ORACLE** : `data/backups/pre-chaine-2026-09-09/shared_matches_v2.duckdb` n'a AUCUNE colonne portant `Inferred`, `AddedNames`, `BijectionMargin` ni `BijectionDetermined` (`match_kill_events` s'arrete a `publishable` ; requete `information_schema.columns` avec les quatre motifs : 0 ligne), et sa revision la plus recente est `killsource-2026-09-05` (548 films publiables + 337 a `killsource-2026-07-31`), donc ANTERIEURE a l'apparition de `BijectionDetermined` (2026-09-14) : aucun de ses films n'a ete decode sous cette porte. Le compte se mesurera a la recuisson, par le compteur neuf. Sens du changement CONNU sans mesure : la porte ne fait que se FERMER — aucun film ne gagne la publication ligne par ligne |
| 2026-09-16 | Revue M1 — L4 | **CONSTAT 2 (P1) TENU — le pont slot -> index publiait l'equipe du PREMIER occupant sur la vie du SECOND.** `equipeDuSlot` (`player_teams.go:141-150`) lisait `reg.IndexParSlot()` (`identity_registry.go:221`) sans consulter `SlotsAmbigus()` ni `PontEpure()`. Verifie sur pieces : `ownersFromLives` (`lives.go:492-514`) garde le premier occupant nomme dans `Owner` et se contente de marquer `ambigus[l.slot]` — `PontDeSlot` (`identity_registry.go:235-248`) s'abstient pour cette raison exacte, et `PontParSlot` a ete RETIRE au lot 6.1 pour ce meme defaut. **CORRIGE** : `equipeDuSlot` rend `(equipe, lue, ambigu)` et se tait sur un slot ambigu ; `poserSurLesTraces` compte les abstentions ; champ publie neuf et OPTIONNEL `coverage.teams.tracksSlotAmbiguous` ; WARN par film. Tests dans `player_teams_slot_ambigu_test.go` (3 tests, dont la primaute du xuid et le temoin a occupant unique) ; mutation « retrait de la garde » rouge puis restauree. sha `cad29309c` | — |
| 2026-09-16 | Revue M1 — L4 | **CONSTAT 3 (P1) TENU — la table d'index COMPOSEE n'etait pas injective par index, la lecture des chunks seule l'etait.** `completerParLesChunks` (`identity_registry_film_table.go:117-126`) ne testait que la duplication de XUID, et `BuildIdentityRegistry` (`identity_registry.go:190`) REMPLACE par sa sortie la table gardee par `injectiveOrEmpty` (`player_index.go:144-165`). Verifie sur pieces : `indexToXUIDOf` (`identity_registry_pont.go:164-170`) ecrase sans garde et sert trois lecteurs (`identity_registry_pont.go:121` et `:146`, `identity_registry_scoreboard.go:102`, `identity_registry_creation.go:141`) — l'identite d'une vie se decidait donc a l'ORDRE D'ITERATION D'UNE MAP, non deterministe d'une cuisson a l'autre. **CORRIGE** : `retirerLesIndexEnCollision` retire l'index POUR LES DEUX xuids ; `Direct` / `Fallback` ne comptent que la table publiee ; champ neuf et OPTIONNEL `coverage.identity.filmTable.collisionsIndex` ; WARN. Tests `TestCompositionRetireUnIndexQueDeuxXUIDSeDisputent` (avec sous-test « determinisme », 50 compositions) et `TestCompositionSansCollisionNeRetireRien` ; mutation « retrait de l'appel » -> 5 assertions rouges. sha `644a524d4` | — |
| 2026-09-16 | Revue M1 — L4 | **CONSTATS 4 ET 5 (P2) TENUS — deux docs a leur mauvaise place.** (4) `domain/replaydoc/coverage.go` : le godoc de `TrackCoverage` etait colle a `TeamCoverage` et `type TrackCoverage struct` n'avait plus de doc. (5) `filmdec/build_profile.go` : le bloc de `mppWidthsPourFormat` finissait par « la cle est le BUILD » alors que la fonction commute sur le FORMAT depuis 1.9.1 ter, et que son propre premier paragraphe l'explique (anti-pattern n 9 du depot). **CORRIGES** tous deux ; `GrammarRev` reste a `.12` (meme lot, meme revision — la forme `.N` le prevoit), empreinte regeneree (elle hache les commentaires : faux positif assume du garde-rail). sha `(ce commit)` | — |
| 2026-09-15 | revue M1 (D13) | **Revue M1 — D13 c1 : `repli_vie_coupee_au_trou_de_replication` avait un SECOND SITE non inscrit, à une AUTRE condition, et son compte sous-décrivait le fait d'un facteur 212.** `replay/tracks_publication.go` (`bornesDesVies`, lot 1.9.13) déclenchait le repli quand le registre d'identité ne rend AUCUNE vie ; l'entrée ne citait que `lives_decoupe.go`. Vérifié sur pièces : `buildOwnersFromTracks` sort avant toute découpe dès que `IdentityInput.PlayerIndices.ByXUID` est vide, donc la condition de ce site est « la table d'index des joueurs manque » (`section_absente`) et non « le film est muet sur les morts » (`film_muet`) — deux gestes de retrait différents sous un seul nom. Et `Declenche` comptait **1 par cuisson** là où le seuil coupait jusqu'à 212 vies : un compte qui ne suit pas la population qu'il décrit rend D14 (d) inopérant. **CORRIGÉ PAR CE LOT** : second site inscrit avec sa propre `Condition` ([fallback.Site] porte désormais un diagnostic par site), `DeclencheN(coupuresDuSeuil(vies))`, et direction (C) du garde-rail `archlint` — tout `Declenche`/`DeclencheN` du dépôt vit dans un fichier que les `Sites` de son entrée citent. | sans objet — corrigé dans ce lot |
| 2026-09-15 | revue M1 (D13) | **Revue M1 — D13 c2 : les DEUX sites du découpage MPP résolvaient le profil par le BUILD, pas par la VERSION DE FORMAT que le registre déclare.** `replay/gwWidthsForFilm` ET `filmdec.ScanEquipmentPlacements` appelaient `BuildProfileFromFilm` (table de sept builds en dur) alors que l'entrée `repli_largeurs_mpp_calibrees_sur_le_film` déclare `format_sans_profil_relu` / `apres_lecture` (lot 1.9.1 ter). Le constat du relecteur disait « site 2 seulement » ; sur pièces les DEUX étaient touchés — `st.FormatSansProfil` ne servait qu'au compteur, la décision venait du build des deux côtés. Un film au format 27 (largeurs relues 9/5) dont le build est hors table tombait sur les largeurs CALIBRÉES, `formatSansProfil()` rendait faux, donc ni compteur ni avertissement : un repli DEVANT une lecture disponible, hors du ratchet des `devant_la_lecture`. **CORRIGÉ PAR CE LOT** : porte unique `filmdec.MPPWidthsForFilm` (clé = version de format), employée par les deux sites et par `formatSansProfil` ; ratchet `TestAucuneResolutionMPPParLeProfilComplet` (appels nus ET qualifiés). **MESURE (cache, 657 films au 2026-09-15, `TestMPPResolutionCorpus`) : 6 films au build hors table — 5 sans section d'identification (format 20), 1 `HI_1_5_1` (format 23) — et AUCUN à un format dont la largeur est relue, donc ZÉRO octet cuit ne change sur ce cache.** `GrammarRev` `.11` -> `.12`. | sans objet — corrigé dans ce lot |
| 2026-09-15 | revue M1 (D13) | **Revue M1 — D13 c3 : `TestRegistrePorteToutesSesFamilles` énumérait CINQ tranches quand `concat` en assemblait SIX.** `registreKillsourceCarte` (lot 1.9.4, 2 entrées) n'était vérifiée par rien. **CORRIGÉ PAR CE LOT** : `fallback.Tranches()` est la source unique dont `registre` découle et que le test parcourt, plus un chaînon « la somme des tranches est le registre » qui rougit si `concat` et `Tranches()` divergent. Mutation jouée (tranche retirée du `concat`). | sans objet — corrigé dans ce lot |
| 2026-09-15 | revue M1 (arch) | **Revue M1 — arch C2 : l'allowlist `replisDeLEcrivainDuJeu` n'était lue qu'en consultation, et trois de ses sept lignes mentaient.** `logFallbacks` n'existe nulle part dans le dépôt (le journaliseur réel est `attachFallbackCoverage`) ; `FallbackHit` et `fallbackHitsOf` étaient annotés `replay/coverage.go` alors qu'ils vivent dans `replay/fallbacks_publication.go`. Une exemption sans site ne rougissait pas, contrairement au ratchet frère `no_title_package_in_analysis_test.go`. **CORRIGÉ PAR CE LOT** : l'exemption porte son fichier dans un champ typé, l'entrée morte est retirée, les deux références sont corrigées, et `TestChaqueExemptionDeLEcrivainATouJoursSonSite` relit chaque fichier cité par l'AST. Mutation jouée (`logFallbacks` remis -> rouge). | sans objet — corrigé dans ce lot |
| 2026-09-15 | revue M1 (L6) | **Revue M1 — L6 c4 : la couverture du garde-rail était un ensemble GLOBAL d'identifiants, testé sans regarder le fichier.** Témoin du relecteur rejoué à l'identique : `func locateFallback() int { return 0 }` ajouté dans `filmdec/varwidth.go` passait VERT, parce que `locateFallback` est cité par l'ancre d'une entrée qui pointe `killsource/`. Un repli neuf reprenant un nom déjà employé ailleurs entrait en production sans entrée. **CORRIGÉ PAR CE LOT** : `couvertureParFichier()` indexe par (fichier, identifiant), et l'exemption ne vaut que dans le fichier qu'elle cite. Mutation jouée : le témoin exact rougit. | sans objet — corrigé dans ce lot |
| 2026-09-15 | revue M1 (D13) | **D1 (revue D13) — `TestChaqueNomConstantEstAuRegistre` était ANNONCÉ par l'en-tête de `fallback/noms.go` depuis le lot 1.9.0 et N'EXISTAIT PAS.** Le fichier affirme « parse CE fichier et exige que chaque constante déclarée ici corresponde à une entrée du registre » ; grep du dépôt : une seule occurrence du nom, celle du commentaire. Les constantes de `noms.go` échappaient à la direction (A) (le répertoire `fallback` est `SkipDir` du parcours). Une documentation qui décrit un garde-rail absent fait croire la direction tenue. **TRAITÉ DANS CE LOT** parce que le mécanisme — parser `noms.go` pour bâtir `identifiant -> valeur` — était déjà écrit pour la direction (C) : trois lignes de plus, pas un périmètre de plus. Mutation jouée (constante orpheline -> rouge). | sans objet — corrigé dans ce lot |
| 2026-09-15 | revue M1 (D13) | **D2 (revue D13) — `BuildProfileFromFilm` n'a PLUS AUCUN APPELANT DE PRODUCTION, et `BuildProfile.MPP` non plus.** Après la bascule du constat 2, ses seuls appels vivent dans deux instruments `research` et dans le ratchet qui l'interdit. `BuildProfileFor` ne sert plus qu'à `BuildProfileFromFilm` et à un instrument. Le champ `MPP` du profil est mort : la clé du bloc MPP est le format, pas le build (lot 1.9.1 ter) ; ce qui reste vivant du profil est `PersoBytes`, qui EST keyé par le build (le format 24 couvre deux largeurs). **NON TRAITÉ** (règle 7, et `filmdec/build_profile.go` est dans le périmètre d'un lot parallèle de la même vague). | M2 (pas 1 ou pas 4, le profil résolu une fois) : retirer `BuildProfile.MPP` et statuer sur `BuildProfileFromFilm` |
| 2026-09-15 | revue M1 (D13) | **D3 (revue D13) — UN FILM DU CACHE PORTE LE FORMAT 23, ABSENT DE `mppWidthsPourFormat`, ET COMPTE DONC COMME « PATCH DU JEU ».** Mesure du 2026-09-15 : 1 film `HI_1_5_1` au format 23 sur 657 ; `MPPWidthsForFormat(23)` rend `ErrUnknownFormat`, donc `filmdec_unknown_format_23` monte alors qu'il s'agit d'un film ANCIEN, pas d'un patch. Le compteur reste juste au sens strict (« le dépôt ne connaît pas ce format »), mais un exploitant qui le voit conclura à une mise à jour du jeu. Le geste serait d'ajouter 23 à la ligne des formats anciens à largeur indéterminée — comme le lot 1.9.1 ter l'a fait pour le format 20 —, ce qui ne change aucun bit lu. **NON TRAITÉ** (hors périmètre : la table vit dans `filmdec/build_profile.go`, fichier d'un lot parallèle). | lot M3 (divergences par build) ou un lot court : ranger le format 23 avec 20/21/24/25 |
| 2026-09-16 | revue M1 — L3 | **C3 (L3) — TENU, CORRIGÉ (`804b643f1`). Huit entrées du registre des replis nommaient en cible un lot FUSIONNÉ (1.9.13), trois d'entre elles décidant un fait publié sans compteur.** (a) Les trois compteurs sont câblés : `BuildUsageSummary` crée son propre `fallback.Compteur` (elle lit un document DÉJÀ CUIT, ses replis sont hors couverture de cuisson) et publie son rapport dans `UsageMatchSummary.Fallbacks` — témoin d'outillage NON persisté, même statut qu'`EquipmentChanges`, aucun champ du document publié ne bouge. **Mesure sur les 8 builds** (`usage_summary_replis_test.go`, fixtures d'assemblage) : `repli_geste_dernier_occupant_du_match` **0**, `repli_geste_premiere_vie_du_slot` **0** (les deux critères TENUS — le recollage des vies du lot 1.9.13 a fermé le défaut mesuré 32-95 % de l'audit 0.E), `repli_garde_equipement_negatif_a_zero` **30 sur 7 des 8 builds** (premier chiffre de cette contradiction entre trois canaux). (b) Les cinq autres cibles mesurées puis réécrites — dont deux constats : `repli_identite_piste_meilleur_recouvrement` est à 0/8 mais son entrée NE SORT PAS (son ancre est la sélection par recouvrement maximal, qui nomme chaque piste : la retirer laisserait l'arbitrage anonyme, D14 a, en supprimant le seul instrument qui prouve le zéro) ; `repli_position_lacher_prend_la_prise` est à 0/8 **NON EXERCÉ** (cf. D2 (L3)). (c) Ratchet neuf `archlint/no_stale_fallback_target_test.go`, mutation jouée. | FAIT |
| 2026-09-16 | revue M1 — L3 | **D1 (L3) — LA MESURE DES TROIS COMPTEURS CÂBLÉS NE PEUT PAS ATTEINDRE `coverage.fallbacks[]`, ET C'EST STRUCTUREL.** `BuildUsageSummary` est une fonction PURE d'un artefact DÉJÀ CUIT (c'est ce qui permet au backfill de re-résumer sans re-décoder) : ses replis se déclenchent après la cuisson, donc hors du compteur qui alimente la couverture du document. Le canal ouvert ici (`UsageMatchSummary.Fallbacks`) est lisible par un test et par les deux producteurs de passes, mais le **corpus gate ne le verra jamais** — ses chiffres viennent de l'artefact. Le seul corpus qui mesure ces trois replis est donc l'échantillon des 8 fixtures d'assemblage, tant que personne ne journalise le champ côté `sync/replayartifacts` et `cmd/levelup` (hors périmètre de ce lot : consigne du pilote de ne rien toucher d'autre dans `replayartifacts/`). **NON TRAITÉ.** | M2 : journaliser `Fallbacks` chez les deux producteurs de passes, comme `EquipmentChanges` |
| 2026-09-16 | revue M1 — L3 | **D2 (L3) — LES 8 GOLDENS D'ASSEMBLAGE N'EXERCENT PAS LE CALQUE DRAPEAU, SI BIEN QU'UN COMPTEUR DE REPLI Y AFFICHE 0 SANS AVOIR ÉTÉ ATTEINT.** `FilmInputs.applyTo` (`replay/film_inputs.go`) pose `opt.Flag.Marks` et JAMAIS `opt.Flag.Scanned` : `attachFlagCarries` prend donc sa branche vide sur les huit builds, `buildFlagCarries` ne reçoit aucun portage, et `attachFlagCarryPositions` — le site de `repli_position_lacher_prend_la_prise` — n'est jamais atteint. Son 0 au bloc « REPLIS DECLENCHES » dit « NON EXERCÉ », pas « non déclenché ». C'est le piège exact que D14 (d) transforme en défaut : un retrait sec sur cette base supprimerait un repli actif. La limite est de la même famille que celle écrite en tête de `golden_builds_test.go` (« cinq canaux manquent des deux côtés »), mais elle n'y était pas nommée pour le drapeau. **NON TRAITÉ.** | M2 : un fixture d'entrées qui porte le canal drapeau (ou la mesure au corpus gate, qui cuit des films de CTF) ; puis verdict de retrait sur `repli_position_lacher_prend_la_prise` |
| 2026-09-16 | revue M1 — L3 | **C4 (L3) — TENU, CORRIGÉ (`1f2e8e9f9`). Trois API de production sans aucun appelant de production.** Greps du 2026-09-16 : `DetectI0Layout(` hors `DetectI0LayoutOf(` = **48 appels, tous dans des `_test.go`** (40 `filmdec`, 8 `replay`), 0 en production depuis le lot 1.9.4 ; `walkKeyframeBody` / `keyframeBodyVariants` = **5 instruments `_test.go` et sa propre déclaration**, 0 production depuis le lot 1.4 ; `ScanFilmEquipmentSpawnEvents` = **1 appel** (`replay/e191_origine_mesure_research_test.go`). Les deux premières passent dans un `_test.go` du même paquet (`*_instrument_helpers_test.go`), la troisième est SUPPRIMÉE au profit de sa forme film chez son unique appelant. Les huit appelants `replay` de `DetectI0Layout` ne pouvant pas importer un symbole de test, `replay` porte les deux mêmes lignes dans son propre helper — deux copies, sous le plafond de la règle 6. `GrammarRev` `.11` → `.12` (faux positif d'empreinte : les trois formes de production sont inchangées à l'octet). | FAIT |
| 2026-09-16 | revue M1 — L3 | **C5 (L3) — TENU, CORRIGÉ (`2d73528d0`). Doc inversée sur la version de format.** `film_format_version.go` renvoyait à l'imparfait (« c'est ainsi que `film_major_version.go` le décrivait ») une correction que `film_major_version.go` n'avait jamais reçue : son en-tête disait seulement que le second u32 « suit (25 / 24 / 27) ». Il le NOMME désormais — version de format de `chunk_00` — et renvoie à `FilmFormatVersionFromHeader` ; l'imparfait est remplacé par le renvoi. Même commit : **arch-C4**, `projeterPositions` (`sync/replayartifacts/positions.go`) affirmait « L'EQUIPE N'EST PAS RESOLUE ICI : le document ne la porte pas » alors que `positionsDeLaTrajectoire` écrit `Team: t.Team` depuis le schéma 57 et que `poserEquipes` ne comble plus que les `-1` — seul site qui décide l'origine du camp persisté dans `player_positions.team`. | FAIT |
| 2026-09-16 | revue M1 — L3 | **C1 / C2 (L3) — TENUS, NON TRAITÉS (règle 7 : M2, lot 2.7).** Mesure sur pièces du 2026-09-16, `783ae680d` → `34fa53da5` : **six fichiers déjà au-delà de 500 L ont GROSSI** — `replay/document_chronicle.go` 1 305 → **1 476** (+171), `sync/killcollector/collector.go` 635 → **773** (+138), `replay/document.go` 597 → **627** (+30), `replay/build.go` 516 → **532** (+16), `replay/zone_states_hill.go` 513 → **526** (+13), `filmdec/traverse.go` 1 385 → **1 388** (+3) ; et **trois fonctions déjà au-delà de 80 L** — `BuildFromPositions` 449 → **469**, `decimateTracks` 95 → **107** (`tracks_publication.go`), `consumeByName` 812 → **815**. Deux d'entre eux ne sont PAS dans l'item 2.7.1 (`collector.go`, `zone_states_hill.go`) : ils y sont ajoutés, ainsi que les trois fonctions. Le lot de revue lui-même n'a rien accru : `usage_summary.go` serait passé de 538 à 592 L, son résolveur de slot est sorti dans `usage_summary_owners.go` par déplacement pur (377 + 231). | M2, lot 2.7 (cibles ajoutées à l'item 2.7.1) |
| 2026-09-16 | revue M1 — L3 | **D3 (L3) — LE RECENSEMENT DU REGISTRE COLLÉ EN §5 (lot 1.9.0) PORTE LES MÊMES CIBLES PÉRIMÉES, ET AUCUN RATCHET NE LE LIT.** Le ratchet neuf relit les champs du CODE (`fallback.Table()`) ; le tableau de 95 lignes collé au journal §5 est un instantané daté du 2026-09-14 qui cite encore « lot 1.9.13 », « n/i », etc. Il n'est pas faux (il dit ce qu'il disait ce jour-là) mais il se relit comme un état. Deuxième limite écrite du ratchet : une FAMILLE de lots (`lot 1.6`, `lot 0.E`) n'est pas un item coché du plan, donc une cible qui nomme une famille entièrement close passe — deux entrées sont dans ce cas (`repli_nom_piste_par_le_pont`, `repli_piste_drapeau_sans_pont_ecartee`, toutes deux « lot 1.6 »). **NON TRAITÉ.** | M2 : régénérer le recensement §5 depuis `RapportDuRegistre` à la clôture de M2, et décider si le ratchet doit couvrir les familles |
| 2026-09-16 | 1.9.10 | **D1 (1.9.10) — LA PRÉMISSE DU LOT SUR LE `ghost` 777 DE `bfecd02b` EST INEXACTE, ET LA CORRIGER CHANGE LE DIAGNOSTIC.** Le plan et le brief disent « c'est le repli qui a effacé le `ghost` slot 777 à 287,4 s ». Sur pièces (artefact du parc, schéma 54) : sa vie porte `t1max = 2874` avec `t1max < frameCount-1`, donc `goneByUS > 0` — c'est une **preuve d'absence ÉCRITE** (la première image-clé qui ne le recense plus), pas l'inférence à 20 s. Le repli ne s'est PAS déclenché sur cette vie : les 10 « fins inférées » de ce film sont les **9 tourelles** `038df01a` et le **ghost 778**, né à 352,6 s. Le 777 est piloté 31,8→36,0 s puis **136,7→270,1 s**, ses positions cessent à 282,1 s (12 s après la sortie de l'occupant — exactement la mise au repos de 13 à 36 s mesurée au lot V3), et il disparaît du recensement à 287,4 s. La question que le lot doit trancher n'est donc PAS « le repli coupe-t-il une vie » mais « le film écrit-il la destruction de cette vie » — d'où le test d'acceptation à disjonction (`vehicle_end_acceptation_test.go`). **NON TRAITÉ au-delà de la correction de la phrase** : le verdict tombe au gate de décodage. | ce §4 ; la phrase de l'item 1.9.10 reste telle quelle, corrigée ici |
| 2026-09-16 | 1.9.10 | **D2 (1.9.10) — 7 KILLS DEPUIS UN VÉHICULE APRÈS LA FIN PUBLIÉE DE TOUTES LES VIES DE SA FAMILLE, SUR `4f77afc1`.** Oracle (i) de l'utilisateur, joué sans décodage : `match_kill_events_latest.source_tag` ∈ les 91 tags de classe `VEHICULE` de `damagetag/data/labels.tsv` (le tag NOMME le châssis : `f712c64a` → `sb_010_veh_cv_ghost`), frame = `(time_ms − originMs)/100`, croisé aux 295 vies publiées de 20 artefacts du parc. **177 kills, 98 dans une vie de la bonne famille, 0 dans un trou, 0 avant la première, 72 de familles jamais publiées, et 7 APRÈS la dernière fin lue** : ghost aux frames 8785, 9007, 9069, 9247, 9295, 9393 contre une dernière fin à 8743 (+65,0 s) ; falcon à 10825 contre 10144 (+68,1 s). Le véhicule est vivant ET occupé à ces instants. **NON TRAITÉ** : à confronter aux dead-states lus, au gate de décodage. | clôture du lot 1.9.10 (confrontation) ; si l'écart persiste, lot de M3 sur la lecture des vies de véhicule |
| 2026-09-16 | 1.9.10 | **D3 (1.9.10) — 72 KILLS-VÉHICULE DE FAMILLES QUE LE DOCUMENT NE PUBLIE JAMAIS** (wraith, tourelles `sb_010_tur_*`, pelican, tags à porteurs multiples), sur les mêmes 20 artefacts. `5676a9ba` en porte 20 sur 33, `4f77afc1` 30 sur 55, `0a44c6cc` 12 sur 13. Le film atteste donc des châssis que le calque ne recense pas — soit absents de la table des familles, soit absents du recensement `ti=40`. **NON TRAITÉ** (recoupe le lot 1.9.9, hors périmètre). | lot 1.9.9 (tourelles nommées) et un lot de M3 sur la table des châssis |
| 2026-09-16 | 1.9.10 | **D4 (1.9.10) — AUCUNE DESTRUCTION DE VÉHICULE N'EST DATÉE HORS DU FILM DANS LE DÉPÔT.** Oracle (ii) de l'utilisateur, instruit sur pièces : `medals_earned` n'est **pas horodatée** (`match_id, xuid, medal_name_id, count, created_at`) ; `highlight_events(event_type='medal')` l'est, mais **aucune médaille ne NOMME le véhicule détruit** — les trois qui en impliquent une sont `Ballista`, `Dogfight`, `Boom Block` ; `VehicleDestroys` de l'API est **parsé et persisté nulle part** (`openspartan/halo_api_payload.go:134`, référence unique, inchangé depuis la note V13 du 2026-09-05). Les personal score awards ne sont pas persistés. **Le dead-state est donc la SEULE source datante**, ce que le plan supposait et qui est maintenant établi. Matière disponible pour le contrôle : **18 destructions datées sur 7 films du corpus**, dont `084a804d` (5 Ballista + 3 Dogfight), `111fa685` (4 Boom Block), `11de8353`, `50247b26`, `a349fea8`, `e5adf7b2`. **NON TRAITÉ**. | clôture du lot 1.9.10 (contrôle) ; la persistance de `VehicleDestroys` reste hors chantier |
| 2026-09-16 | 1.9.10 | **D5 (1.9.10) — DEUX COPIES DE PRODUCTION DE LA MARCHE COEXISTENT.** `killsource/walk.go` (avec son filtre de crédibilité de roster, ses goldens et ses ancres Theater) et `filmdec/object_deaths_march.go` (neuf). Elles partagent le localisateur — inscrit comme UN repli à DEUX sites (`repli_localisation_largeur_libre`) — mais pas leur critère d'acceptation : `killsource` garde `DesyncAt == -1` (configuration mesurée et gelée), `filmdec` accepte la queue désynchronisée. Les fusionner exigerait de rejouer les goldens de `killsource`, ce qui n'est pas le périmètre de ce lot. Une troisième copie, de RECHERCHE et gardée par variable d'environnement, vit dans `filmdec/vehicules_v13_marche_helpers_test.go`. **NON TRAITÉ** (règle 7). | pas 4 de M2 (« une seule porte aux octets ») |
| 2026-09-16 | 1.9.10 | **D6 (1.9.10) — 20 MORTS `ti=40` QU'AUCUNE VIE RECENSÉE NE REPREND.** Sur les 14 témoins du gate : `60ae07c4` **7 orphelines sur 8** (carte d'aréne, 4 vies recensées et 0 publiée — toutes sans position), `a349fea8` **12 sur 15** (265 vies recensées, 109 publiées), `e5adf7b2` **1 sur 4**. Ce sont soit des vies absentes du recensement, soit des lectures fausses — la classe « hors bande » de la note V13 § 5.3 (slots 116, 1449, 5507). **Elles ne changent AUCUNE valeur publiée** : sans vie porteuse, elles ne datent aucune fin ; `deathsUnmatched` existe exactement pour qu'elles se voient. **NON TRAITÉ**. | M3 (lecture des vies de véhicule) ; le compteur alerte si un build le fait monter |
| 2026-09-16 | 1.9.10 | **D7 (1.9.10) — 683 ÉCHANTILLONS PUBLIÉS APRÈS UNE FIN ÉCRITE, ET CE N'EST PAS UN DÉFAUT.** `084a804d` 407, `4f77afc1` 115, `e5adf7b2` 93, `bfecd02b` 40, `111fa685` 28. Sur les 9 vies détruites de `084a804d`, `t1` dépasse `tEnd` de ~9 s en moyenne ; sur le ghost 777 de `bfecd02b`, de **8,1 s**. C'est l'ÉPAVE : le film continue de répliquer la position de l'objet après avoir écrit sa mort — même profil que la mise au repos de 13 à 36 s mesurée au lot V3 après l'abandon. Le dépôt PUBLIE la contradiction (`coverage.vehicles.samplesAfterEnd`) et ne coupe pas la trajectoire : couper affirmerait une absence que les octets du film contredisent. **NON TRAITÉ** (écrit pour qu'un futur lot ne lise pas ce compteur comme une régression). | sans objet ; le compteur est le contrat |
| 2026-09-16 | 1.9.10 | **D8 (1.9.10) — SUR `4f77afc1`, LES 14 MORTS LUES SONT **TOUTES** À QUEUE DÉSYNCHRONISÉE.** `deathsRead = 14`, `deathsTailDesync = 14`. Sans la règle du lot — accepter un dead-state quand `DesyncAt > index(dead-state)` — ce film rendrait **ZÉRO** mort de véhicule. C'est la démonstration directe, sur un témoin du gate, que le verrou levé (`DesyncAt == -1`) était bien celui qui cachait le fait. Profil confirmé ailleurs : `084a804d` 6 sur 9, `a349fea8` 3 sur 15, `bfecd02b` 1 sur 1. **NON TRAITÉ** (c'est un résultat, pas une dette). | sans objet |
| 2026-09-16 | 1.9.10 | **D9 (1.9.10) — LES KILLS DEPUIS UN VÉHICULE APRÈS LA FIN PUBLIÉE NE SONT PAS COUVERTS PAR LE DEAD-STATE.** Confrontation sur l'artefact FRAIS de `4f77afc1` (`originMs = 86593`) : les 6 kills ghost des frames 8785, 9007, 9069, 9247, 9295, 9393 et le kill falcon de la frame 10825 ne sont couverts par **aucune vie de leur famille** — les deux dernières vies ghost (slots 952, 1019) s'arrêtent à `t1max = 8743` en `end = unknown`, la vie falcon 974 à 10144 en `unknown`. Le **slot 986** (`t0=7128 t1=8787 t1max=8943`, châssis non résolu), auquel le lot 1.9.9 attribue trois kills à +6,5 s et +22,8 s, est lui aussi `end = unknown` : **le film n'écrit pas sa mort**. Sur ce témoin **92 vies sur 97 restent `unknown`**. Le lot DATE ce que le film écrit ; il ne comble pas ce qu'il n'écrit pas, et le compte le dit. **NON TRAITÉ**. | M3 (lecture des vies de véhicule) ; recoupe D2 (1.9.10) |
| 2026-09-16 | 1.9.10 | **D10 (1.9.10) — UNE ASSERTION DU TEST D'ACCEPTATION A ÉTÉ ÉCRITE AVANT LA MESURE ET RÉFUTÉE PAR ELLE.** Elle exigeait `t1max <= tEnd` (« la borne d'affichage ne dépasse pas la fin écrite »). Sur `bfecd02b`, le ghost 777 est écrit détruit à 274,0 s et le film réplique encore sa position jusqu'à 282,1 s : l'assertion demandait d'affirmer une absence que les octets contredisent. Remplacée par l'invariant qui attrape le vrai défaut — quand `t1max` dépasse `tEnd`, il doit s'arrêter EXACTEMENT sur `t1`, jamais au-delà, sinon la borne est inférée. Le changement est consigné ici parce qu'il a été fait APRÈS avoir vu le résultat. **Contrôle de discrimination joué** : le même test est ROUGE sur l'artefact de la base (`end="unknown"`) et VERT sur celui du HEAD. | sans objet ; l'assertion corrigée est au test |
| 2026-09-16 | revue M1 — ronde 2 | **F1 — TENU, CORRIGÉ. Le « chaînon » qui fermait `TestRegistrePorteToutesSesFamilles` était une TAUTOLOGIE, et les deux planchers ne mordaient sur rien.** `registre.go:62` pose `var registre = concat(Tranches())` : `somme(tranches) == len(Table())` est donc vrai par construction et aucune mutation ne peut le casser ; `len(tranches) < 2` laissait passer 6 -> 5 et `plancherEntrees = 60` laissait passer 96 -> 94. Mutation du relecteur rejouée à l'identique (tranche `{"killsource/carte", registreKillsourceCarte}` retirée de `Tranches()`) : **`fallback` ET `archlint` VERTS** (`ok fallback 0.488s` · `ok archlint 58.539s`) pendant que deux entrées quittaient le registre. **CORRIGÉ** : liste NOMMÉE `famillesAttendues` (les six noms, dans l'ordre), présence + ordre + non-vacuité vérifiés nom par nom, et `plancherTranches = 6` daté (mesure du 2026-09-16 : 14/22/26/2/21/11 = 96 entrées). Aucune assertion sur le CONTENU d'une tranche — une entrée de plus ou de moins n'y rougit pas (contre-épreuve : entrée renommée -> `ok`). Mutations : tranche retirée -> **ROUGE** (`Tranches() ne rend que 5 familles (plancher 6)`) ; tranche retirée **ET** plancher baissé à 5 -> **ROUGE quand même** (`la famille "killsource/carte" a QUITTE Tranches()` + deux erreurs d'ordre). Restaurées par nom. | sans objet — corrigé dans ce lot |
| 2026-09-16 | revue M1 — ronde 2 | **F2 — TENU, CORRIGÉ. `UsageMatchSummary.Fallbacks` était écrit et LU PAR AUCUN CODE DE PRODUCTION, et trois cibles de retrait nommaient un instrument incapable de les mesurer.** Grep du 2026-09-16 sur `Fallbacks` hors `_test.go` : aucun lecteur de `Match.Fallbacks` — `sync/replayartifacts/usage.go:161` journalise `EquipmentChanges`, jamais les replis ; `cmd/levelup/cmd_backfill_usage_summary.go`, `cmd/replay-corpus-gate`, `cmd/replay-equiv` : rien. Le critère « le corpus gate de cloture de M1 le confirme » est donc **structurellement inatteignable** — le gate lit les ARTEFACTS et ces replis se déclenchent APRÈS la cuisson (c'est la découverte D1 (L3), désormais fermée). **CORRIGÉ** : les DEUX producteurs journalisent (`journaliserReplisUsage`, `journaliserReplisUsageCorpus`), une ligne `slog.Info` par match déclenchant (retrouver le film) **et une par passe, TOUJOURS, « aucun » compris** — c'est elle qui distingue « jamais déclenché » de « jamais instrumenté » (D14 d) ; helpers partagés `fallback.Compteur.Cumuler` + `fallback.Texte` (une seule boucle d'addition pour deux producteurs) ; les trois critères nomment les DEUX instruments réels (les 8 builds de `replay/usage_summary_replis_test.go`, la ligne `replis de la passe` sur le parc) ; la doc du champ dit qui le lit. Trois tests (`usage_replis_journal_test.go`), dont un qui part du POINT D'ENTRÉE `persisterResumesUsage`. Mutation (ligne de câblage retirée) -> **ROUGE**, restaurée par nom. | sans objet — corrigé dans ce lot |
| 2026-09-16 | revue M1 — ronde 2 | **F3 — TENU, CORRIGÉ. `killsource_bijection_noms_libres_en_trop` SURCOMPTAIT la population qu'il prétend mesurer.** `collector.go:788` posait `t.Inferred > 0 && t.FreeNames > t.Inferred`, donc il s'incrémentait aussi sur `Inferred >= 2` — un régime où l'ancienne porte (`Inferred <= 1`) refusait DÉJÀ, où rien ne bascule et où aucune publication n'est perdue. **CORRIGÉ** : `t.Inferred == 1 && t.FreeNames >= 2`, la seule population que `AffectationUnique` fait passer de vrai à faux ; doc du compteur alignée (`Inferred >= 2` relève de `killsource_bijection_inference`). Test des TROIS régimes avec contrôle croisé sur les deux portes (`TestAmbiguiteNeCompteQueLesFilmsQuiBascule`). Mutation (ancienne condition remise) -> **ROUGE sur le seul régime concerné** (`2 indices libres / 3 noms libres : delta 1, attendu 0`), restaurée par nom. | sans objet — corrigé dans ce lot |
| 2026-09-16 | revue M1 — ronde 2 | **F4 — TENU, CORRIGÉ. `collector.go` portait 97 lignes d'historique d'une constante qui ne le concerne pas.** Le fichier déclare en en-tête « enchaîner les trois — CE FICHIER » et pesait 816 lignes (804 avant la correction F3). **CORRIGÉ par DÉPLACEMENT PUR** : `KillSourceDecoderRev` et son historique sortent dans `killcollector/killsource_decoder_rev.go` (121 L) ; `wc -l collector.go` **816 -> 718**. Contrôle ligne à ligne : les 98 lignes retirées de `collector.go` (hors les 7 lignes F3) sont EXACTEMENT les 98 lignes ajoutées au fichier neuf (`diff` des deux extraits : vide). `metricBijAmbigue` RESTE dans `collector.go` — ce n'est pas un bloc autonome mais une ligne du groupe `const` des compteurs de santé ; l'en sortir aurait scindé un groupe et ne serait plus un déplacement pur. **Le ratchet d'empreinte reste VERT SANS régénération** (`TestKillSourceDecoderRevSuitLeDecodeur` `ok 0.01s`, `testdata/` non modifié) : il hache `internal/games/halo_infinite/film/killsource/`, jamais `killcollector/`. | sans objet — corrigé dans ce lot |
| 2026-09-16 | revue M1 — ronde 2 | **F5 — TENU, CORRIGÉ, ET LA CHRONOLOGIE DU BRIEF ÉTAIT FAUSSE SUR PIÈCES.** La chronique de `grammar_rev.go` empilait TROIS blocs annonçant chacun « `.11` -> `.12` » suivis de deux lignes « FUSION … au rang suivant » : elle s'arrêtait à `.12` quand la constante valait `.14`, et `.13`/`.14` n'avaient aucune entrée — ni dans le godoc, ni dans le golden (dont la dernière entrée était de surcroît ÉTIQUETÉE `.12` pour le lot L3). Relevé commit par commit (`git log --first-parent` + `git show <sha>:…/grammar_rev.go`), l'intégration a fait **`.11` (944e7c691, L6) -> `.12` (99644996e, L4 : porte killsource `AffectationUnique`) -> `.13` (1f478d5c3, D13 : porte unique `MPPWidthsForFilm`) -> `.14` (29c5d6c85, L3 : code musée)** : le brief attribuait `.13` à une « fusion L4 + D13 » qui n'a jamais eu lieu. **CORRIGÉ** : chronique LINÉAIRE, une entrée par rang, forme uniforme « ENTREE + révision entre accents graves » dans le godoc et ligne d'HISTORIQUE dans le golden ; l'entrée `.13` dit explicitement que la porte killsource relève de `KillSourceDecoderRev` et a été traitée au rang `.12`. Ratchet neuf `TestChroniqueCouvreLaRevisionCourante` (modèle `TestDocumentShapeSchemaHasChronicleEntry`), qui exige l'entrée des DEUX côtés. **`GrammarRev` NE BOUGE PAS** : `grammar_rev.go` est hors du hachage (`fichierHorsGrammaire`) et le golden ne fige que sa ligne de données — porte `-update-grammar-rev` rejouée, golden **byte-identique** (md5 `aa1c189a…` avant et après) et refus après réécriture. | sans objet — corrigé dans ce lot |
| 2026-09-16 | 2.7 (grammaire) | **D1 (2.7g) — la colonne `code_source` de `ecs_table.tsv` etait DEJA fausse aux trois quarts avant le lot.** Mesure sur pieces, collee au compte rendu : sur les 176 pointeurs `traverse.go:<ligne>` distincts (498 lignes de table), **36 seulement tombaient sur une ligne `case`** du `traverse.go` d'avant le lot ; les 140 autres avaient derive au fil des lots sans que rien ne le voie — `checkCodeSource` ne verifie que « le fichier existe et la ligne est dans le fichier ». La scission a repointe les 498 pointeurs par DECALAGE (l'image exacte du deplacement), plus 7 pointeurs dont l'image serait tombee dans un fichier qui ne porte aucun `case` (`traverse_precision.go`, l'en-tete de fonction) et qui ont ete remis sur leur vrai `case`. NON TRAITE : realigner les 140 autres sort du perimetre d'un lot de deplacement, et durcir `checkCodeSource` (exiger un `case` du composant a la ligne citee) est un changement de garde-rail, pas un deplacement. | le lot qui rouvre la table ECS (3.6, ports de composants), ou un lot de garde-rails |
| 2026-09-16 | 2.7 (grammaire) | **D2 (2.7g) — `(e chunk00Error) Error()` fait 101 lignes** (`filmdec/film_identity.go`), au-dela du seuil de 80. Releve par le meme instrument que les autres fonctions du lot. NON TRAITE : `film_identity.go` est l'un des cinq fichiers que le lot 2.1 (profil) tenait EN MEME TEMPS, le partage des fichiers de la vague l'interdisait. | le lot 2.1, ou le premier lot qui rouvre `film_identity.go` |
| 2026-09-16 | 2.7 (grammaire) | **D3 (2.7g) — deux constats `unparam` latents du paquet, reveles par le ratchet de lint sans appartenir au lot** : `consume140c1e9d4 - w always receives 12` (`components_biped_ability.go`) et `consumeDynPrecVec3 - mag always receives 19` (`components_movement.go`). Ils n'apparaissent PAS contre la base du lot (`--new-from-rev=f950b7179` : 0 issue) mais apparaissent contre `origin/main`, parce que ces deux fichiers sont touches par la vague. NON TRAITE : retirer un parametre est un changement de signature, pas un deplacement. | le lot qui rouvre ces deux deserialiseurs (3.6) |
| 2026-09-16 | 2.7 (grammaire) | **D4 (2.7g) — un lot de DEPLACEMENT PUR paie la dette `goconst` latente du paquet, et c'est structurel.** Le gate de lint est un ratchet `--new-from-merge-base=origin/main` : deplacer une ligne la rend NEUVE, donc un constat que le ratchet cachait depuis des mois se met a rougir sans qu'une seule ligne ait change de sens. Mesure : 10 constats `goconst` sur les seules etiquettes de composant deplacees. Traites DANS le lot (six etiquettes nommees dans le bloc de `registry.go`, copies du paquet migrees — regle 6), mais le cout est a connaitre avant le volet 2.7p, qui deplacera autant de lignes cote publication. | methode : a citer dans le brief du lot 2.7p |
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
| 2026-09-14 | 1.2.6 | `845768285` | `go test ./...killsource/ -run TestGoldenMiniBobine` puis `-update`, pré-image conservée et `diff` complet | **UNE ligne sur 76** : `recordStateParam=3 [croissance x1.002]` -> `x1.001`. `axisW=14`, `indexW=1`, `recordStateParam=3` INCHANGÉS ; lignes de kill, couverture, contrôle négatif, voies et santé identiques à l'octet. **CLASSÉ DIVERGENCE ATTENDUE** : `RSPRatio` est le quotient best/worst du critère de croissance (`calibrate.go:157`), qui traverse des records bruts de tous les archétypes présents |
| 2026-09-14 | 1.2 (communs) | ce commit | `gofmt -l ./internal ./cmd` ; `go vet` (5 racines) ; `go test` (13 paquets) ; `make go-api-lint` | gofmt **vide** ; vet **0 diagnostic** ; **13 paquets ok, exit 0** (filmdec 7,9 s · killsource 0,5 s · replay 14,8 s · archlint 15,5 s · objectiveevents 0,3 s) ; lint **0 issues** (`--new-from-merge-base=origin/main`), baseline non accrue |
| 2026-09-14 | 1.2 (équivalence, régime COMPLET) | ce commit | `go run ./cmd/replay-equiv -repo-root <worktree>` sur les 20 films, 4 sous-ensembles séquentiels (jamais deux à la fois) | **20 IDENTIQUES sur 20, 0 différent, 0 écarté, 0 échec.** Sous-ensembles : 10 films 5 min 27 s · `084a804d`+`1c4c63c2` 5 min 12 s · `a349fea8`+`a521164d` 5 min 04 s · 6 films 4 min 04 s. **Total 19 min 47 s**, pic max 0,77 Gio (`a349fea8`). AUCUN des 50 balayages ne change, l'artefact compris — donc **aucun octet cuit ne bouge et `SchemaVersion` reste à 55**. Prédit par la mesure 1.2.1 : les 4 instances corrigées vivent en ti=14/21/30/44, qu'aucun balayage du corpus ne traverse (découverte D4 (1.2)) |
| 2026-09-14 | 1.2 (corpus gate, régime COMPLET) | ce commit | `go run ./cmd/replay-corpus-gate --base=191933992 --parc-root <parc> --source-root <worktree> --manifest config/replay_corpus.toml` | **13 témoins, 0 PERTE, 0 GAIN, exit 0**, schéma 55 des deux côtés sur les 13. Durées : `bcb6d393` 15,2 s · `fb1a1a72` 31,5 s · `d9781168` 24,7 s · `c75f33b8` 15,4 s · `bf15f7ab` 14,2 s · `51ebbc0f` 19,2 s · `084a804d` 2 min 21,9 s · `0797ce72` 12,9 s · `111fa685` 34,5 s · `e5adf7b2` 49,6 s · `60ae07c4` 36,7 s · `a349fea8` 3 min 28,3 s · `bfecd02b` 36,8 s. **Total 23 min 10 s** (chaque témoin cuit DEUX fois, base et HEAD). Aucune perte, donc aucun lecteur à corriger au titre de 1.2.6 |
| 2026-09-14 | 1.2 (clôture) | ce commit | doc inversée résiduelle corrigée dans `registry_fingerprint.go` (l'en-tête affirmait « le registre est bit-a-bit IDENTIQUE sur tous les films mesures a ce jour » deux écrans au-dessus du commentaire qui dit l'inverse) ; porte de régénération de la colonne `level` sortie dans `ecs_table_level_gate_test.go` (`ecs_table_guard_test.go` frôlait 500 lignes : 495 -> 442) | Correction de TEXTE seule : aucun octet lu ne change, mais l'empreinte de grammaire hache les octets — `TestGrammarRevSuitLaGrammaire` ROUGE (`7a9404b9…` -> `89d2ecac…`), golden régénéré par sa porte nommée, **révision INCHANGÉE** (`grammar-2026-09-14.2`) : c'est le MÊME lot, et la règle de `grammar_rev.go` est que deux changements d'un même lot la partagent. Puis gates rejoués : gofmt vide, vet 0, **13 paquets ok**, G1/G2/G3/G4 verts avec le film, lint **0 issues** |
| 2026-09-14 | 1.2 revue R1 | ce commit | **revue adversariale ronde 1 : 2 P1 + 2 P2 recevables, 4 remarques non retenues, 29 conditions du relecteur tiennent** ; les 4 corrigées dans le lot | Le premier P1 (C1) est une PANIQUE de production introduite par le lot : borne de boucle relâchée -> `zeroTail` avec `from > to`. Le second (C2) est une doc inversée sur l'une des quatre instances que le lot désigne lui-même |
| 2026-09-14 | 1.2 revue R1 (C1) | ce commit | REPRODUCTION des deux cas du relecteur, borne fautive REMISE : `go test ./...filmdec/ -run TestParseRegistreTronque` | **les deux PANIQUENT, aux offsets exacts du rapport** : `(A)` synthétique de 13 octets -> `slice bounds out of range [268:13]` ; `(B)` registre réel coupé à `registryEntryBase + 6*archetypeBlockSize + 41*registrySlotSize + 2` = 110 510 -> `[110768:110510]`. Pile : `zeroTail` <- `registryBlockTail` <- `parseRegistry`. Borne remise à des blocs ENTIERS : **les deux PASSENT** (0 et 6 blocs rendus, `TruncatedBytes` 5 et 10 662), et `TestParseRegistreCompletNEstPasTronque` tient le cas nominal (`TruncatedBytes = 0`, 50 archétypes) |
| 2026-09-14 | 1.2 revue R1 (C1) | ce commit | fixtures synthétiques recadrées : `buildBlock` -> `buildRegistry` (en-tête + N blocs entiers) et `TestParseRegistryChunkAccepteUnRegistreInflate` | Elles construisaient `n*archetypeBlockSize` octets depuis l'octet 0 : sous une borne à blocs ENTIERS le dernier bloc est incomplet de huit octets et disparaît. `TestParseRegistrySynthetic` rend de nouveau **2 archétypes** et le test d'inflate **1 composant**. `KnownRegistryFingerprint` **INCHANGÉE** (`TestRegistreReelDeLaMiniBobine` vert) : la lecture nominale ne bouge pas |
| 2026-09-14 | 1.2 revue R1 (C2) | ce commit | `traverse.go` (commentaire d'`asset-transform-component`) et `testdata/ecs_table.tsv` ligne 976 (`bits_typ`) | Le commentaire disait « ti44 i0 = L0 -> 6 bits/axe » alors que le lot met ce composant à **L1** et que `quantAxisWidth(1) = 7` : budget écrit 5 x (1+1+3x6) = 100, lu 5 x (1+1+3x7) = **115**. Les deux corrigés (`~100` -> `~115`). G4 ne contrôle que les `bits_typ` ENTIERS — consigné en découverte D5 (1.2) |
| 2026-09-14 | 1.2 revue R1 (C3) | ce commit | `go test -count=1 ./...filmdec/ -run TestRegistreNiveauxVoisinsCensus -v` REJOUÉ, tableau du rapport 1.2.1 réécrit cellule par cellule. **Sortie brute, rejouable** : `a521164d` 1033 · 173 (16.7 %) / ti=9 1/9 · ti=11 3/34 · ti=12 6/28 · ti=35 19/64 · ti=40 18/48 · ti=42 10/21 · ti=43 10/40 — `60ae07c4` 1031 · 178 (17.3 %) / 1/9 · 3/34 · 6/28 · 23/64 · 18/48 · 10/21 · 10/41 — `11de8353` 1031 · 178 (17.3 %) / 1/9 · 3/34 · 6/28 · 23/64 · 18/48 · 10/21 · 10/41 — `111fa685` 1031 · 186 (18.0 %) / 1/9 · 3/34 · 6/28 · 23/64 · 20/48 · 10/21 · 12/41 — `e5adf7b2` 1031 · 188 (18.2 %) / 1/9 · 3/34 · 6/28 · 25/64 · 20/48 · 10/21 · 12/41 — `bcb6d393` 1067 · 189 (17.7 %) / 1/10 · 3/34 · 6/28 · 25/64 · 20/48 · 10/21 · 12/41 — `fb1a1a72` 1067 · 189 (17.7 %) / 1/10 · 3/34 · 6/28 · 25/64 · 20/48 · 10/21 · 12/41 | **13 cellules étaient fausses** (colonnes ti=35 / ti=40 / ti=43 des quatre dernières bobines recopiées des lignes 2-3 ; ti=9 de `bcb6d393` et `fb1a1a72` = 1/**10**). Les lignes GLOBAL et la table des quatre instances touchées étaient, elles, justes — c'est le tableau par archétype qui avait dérivé à la recopie |
| 2026-09-14 | 1.2 revue R1 (C4) | ce commit | `awk 'END{print NR}' killsource/testdata/minibobine.golden` | **76 lignes** (5 592 octets), avant comme après. Les deux occurrences de « UNE ligne sur 77 » (§4 D2 et §5) corrigées en « UNE ligne sur 76 » |
| 2026-09-14 | 1.2 revue R1 (vocabulaire) | ce commit | `grep -n 'flags(registre)\|flags du registre' *.go` | 4 occurrences (`components_movement.go:509,527`, `components_world.go:12`, `traverse.go:548`) : **0 après correction**. Le champ s'appelle `Archetype.Levels` depuis 1.2.2, il n'existe aucun « flags » dans une entrée de registre |
| 2026-09-14 | 1.2 revue R1 (gates) | ce commit | `gofmt -l ./internal ./cmd` ; `go vet` + `go test -count=1` sur `./internal/games/halo_infinite/film/... ./internal/archlint/ ./internal/sync/killcollector/` ; `make go-api-lint` | gofmt **vide** ; vet **0 diagnostic** ; **10 paquets ok, exit 0** ; lint **0 issues**. `TestGrammarRevSuitLaGrammaire` a rougi (`registry.go`, `traverse.go`, `components_*.go` changent) : golden régénéré par `-update-grammar-rev`, **`GrammarRev` INCHANGÉE** (`grammar-2026-09-14.2`) — c'est le MÊME lot, et la règle de `grammar_rev.go` est que deux changements d'un même lot la partagent. Aucun test renommé ni supprimé : baseline JSONL intacte (le test de C1 porte un nom NEUF) |
| 2026-09-14 | 1.2 revue R2 | ce commit | **revue adversariale ronde 2 (corrections de la R1 seulement) : 0 P0, 0 P1, 2 P2 recevables, 1 non retenu, 23 conditions tiennent** ; les 2 corrigés, les 4 corrections de la R1 font ce qu'elles prétendent | La boucle converge : 2 P1 (R1) -> **0** (R2). Le non retenu est consigné en découverte D7 (1.2) — cellule `code_source` décalée, préexistante, G1 la voit par le nom du `case` |
| 2026-09-14 | 1.2 revue R2 (P2-1) | ce commit | `Registry.Truncated bool` ajouté (= le parse a ÉPUISÉ le tampon), `TruncatedBytes` gardé comme MESURE (peut valoir 0) ; sous-test `(C)` : coupe alignée à `registryEntryBase + 6*archetypeBlockSize` sur la bobine versionnée | Une coupe SUR une frontière de bloc ne laisse aucun octet de queue : `TruncatedBytes = 0` et la troncature redevenait MUETTE, dans le mode de panne même que le champ devait fermer. **MUTATION JOUÉE** : `reg.Truncated = epuise && queue > 0` (exactement le défaut trouvé) -> seul le sous-test `(C)` ROUGIT (« Truncated = false sur un tampon coupé »), `(A)` et `(B)` restent verts ; arbre restauré. `TestParseRegistreCompletNEstPasTronque` assert désormais aussi `Truncated == false` |
| 2026-09-14 | 1.2 revue R2 (P2-2) | ce commit | `grep -rn "ParseRegistryChunk(" --include=*.go internal/ cmd/ \| grep -v _test.go`, sortie collée à côté du compte dans les trois textes | **TROIS appelants de production, pas deux** : `filmdec/film_context.go:254`, `killsource/world.go:58` (via `killsource/decode.go:123`, paquet importé par `killcollector` ET par `replaybuild`), `killcollector/hits.go:113` ; plus `cmd/rdata_weapon_scan/main.go` (756, 765, 798), outil de recherche hors production. Corrigé dans `registry.go`, `registry_tronque_test.go` et la découverte D6. **Deuxième dénombrement d'appelants faux du chantier : désormais tout compte d'appelants écrit dans un texte est produit par un grep collé à côté du compte** |
| 2026-09-14 | 1.2 revue R2 (gates) | ce commit | `gofmt -l ./internal ./cmd` ; `go vet` + `go test -count=1` sur `./internal/games/halo_infinite/film/filmdec/ ./internal/archlint/ ./internal/sync/killcollector/ ./internal/games/halo_infinite/film/killsource/` | gofmt **vide** ; vet **0 diagnostic** ; **4 paquets ok, exit 0**. `TestGrammarRevSuitLaGrammaire` a rougi (`registry.go`) : golden régénéré par `-update-grammar-rev`, **`GrammarRev` INCHANGÉE** (`grammar-2026-09-14.2`, même lot). `KnownRegistryFingerprint` **inchangée** (`TestRegistreReelDeLaMiniBobine` vert). Aucun test renommé : baseline JSONL intacte |


| 2026-09-14 | 1.3 (Ghidra, AVANT de coder) | `0e830ab33` | `decompile_function` sur `0x140FED6F4`, `0x14101A0A4`, `0x141133C24`, `0x14116F514`, `0x1410F44F8` et `0x1406cf008` (instance partagée `HaloInfinite.exe`, base `0x140000000`, LECTURE SEULE) | Les cinq largeurs viennent du décompilé, pas de la note : ti=14 `FUN_1406cf008` puis `*(param_4+0x2c) += 5` · ti=17 `+= 7` · ti=21 un unique `+= 0x12`, AUCUN appel avant · ti=29 le préfixe SEUL, `return 1` · ti=47 `FUN_1406cf008` puis `+= 5`. `FUN_1406cf008` relu aussi : `+= 1` (R(1) sec). **`FUN_140467a20`, que le commentaire donnait pour le déserialiseur de ti=14, est un `return;` partagé par TREIZE symboles exportés** (`AK::MemoryMgr::GetCategoryStats`, `?GetDefaultSettings@StreamMgr@AK@@`, `ManagedDebug_LogError`, `Variant_InitializeStaticScriptComponents`, ...) |
| 2026-09-14 | 1.3 (mesure avant de coder) | `ca9b8116c` | `grep -rn "WalkKeyframeFullState(\|KeyframeClosure(" --include=*.go internal/ cmd/ \| grep -v _test.go` | **TROIS lignes, toutes dans `filmdec`** (`keyframe_closure.go:76`, `keyframe_closure.go:120`, `keyframe_fullstate_loop.go:71`) : le cadre d'état complet n'a aucun appelant de production. Le seul consommateur de production de la table est `TraverseEntity` (`traverse.go:1107`), sur les records NEW — c'est LUI qui fixe la population touchée |
| 2026-09-14 | 1.3 (mesure avant de coder) | `ca9b8116c` | `CGO_ENABLED=0 DELTA_WITNESS_FILM=<cache>/<film> go test ./...filmdec/ -run TestDeltaWalkWitness -v -count=1`, un film par process | **SORTIE BRUTE, records NEW par archétype** — `000d5950` : `ti=0:945 … ti=14:13 … ti=17:43 … ti=21:29 … ti=29:25 … ti=47:17 \| total NEW 3322` · `06dfe6d9` : `ti=14:6 ti=17:8 ti=21:5 ti=47:2 \| total NEW 811` (aucun ti=29) · `64e8adfa` : `ti=14:13 ti=17:395 ti=21:71 ti=29:7 ti=47:9 \| total NEW 2518`. MAJORANT : l'histogramme compte aussi des `TypeIndex` 50 à 63, qui n'existent pas (records lus après désync) |
| 2026-09-14 | 1.3 (état du témoin AVANT le lot) | `783ae680d` (arbre restauré par nom) | même commande, les trois films | **LE TÉMOIN FIGÉ EST DÉJÀ ROUGE AU COMMIT D'INTÉGRATION** : `000d5950` {14 350, 38 897, 30 101} contre figé {14 350, 38 878, 30 080} · `06dfe6d9` {6 606, 10 629, 8 499} contre {6 606, 10 613, 8 502} · `64e8adfa` {14 357, 39 820, 31 988} contre {14 357, 39 806, 31 973}. Vérifié des DEUX côtés de mon ajout (mêmes chiffres) : l'histogramme n'y est pour rien. Découverte D1 (1.3), NON TRAITÉE |
| 2026-09-14 | 1.3 (fermeture AVANT) | `783ae680d` | `CHUNK00_FILMS=<6 films de recherche> go test ./...filmdec/ -run TestImageCleFermetureParArchetype -v -timeout 30m` | `TOTAL \| 8796/62686 14.0% d15134 s8023 u30733 \| 529/62686 0.8% \| 0/62686 0.0%` ; `BILAN par archetype : etat complet GAGNE sur 8, PERD sur 0, EGALITE sur 25`. ti=14 `0/5024` (u5024), ti=17 `0/5379`, ti=21 `0/373`, ti=29 `0/157`, ti=47 `0/1679` |
| 2026-09-14 | 1.3.1 | `0e830ab33` | cinq entrées ajoutées à `defaultStateDeserByTI` ; `consumeDefaultStateTI14/17/21` créés, ti=47 partage le porteur de ti=14 ; commentaire STUB corrigé ; repli « absent = 0 bit » NOMMÉ (D14) | `grep` de contrôle collé : REAL de la table de descripteurs = `5 6 8 9 10 11 12 13 14 17 20 21 23 24 28 29 35 36 37 38 39 40 41 42 43 44 47 48 49` ; clés de la map après le lot = `3 5 6 8 9 10 11 12 13 14 17 20 21 24 28 29 36 37 38 39 42 43 47 48 49`. Différence = **ti=23, 40, 41, 44** (plus ti=35, le bipède, traité à part) : ce sont EXACTEMENT les quatre archétypes que le repli nommé déclare |
| 2026-09-14 | 1.3.1 (empreinte) | `0e830ab33` | `go test ./...filmdec/ -count=1` juste après l'ajout | `TestGrammarRevSuitLaGrammaire` **ROUGE de lui-même** : « LA GRAMMAIRE A CHANGE SANS MONTEE DE REVISION », `8e118038…` -> `29a60ed4…` (150 fichiers). C'est la preuve par mutation NATURELLE du gate — aucune mutation artificielle n'a été nécessaire. Les 40+ autres tests du paquet restent verts |
| 2026-09-14 | 1.3.2 | `5f66051d0` | `go test ./...filmdec/ -run TestEtatParDefautN2Constant -v -count=1` | **77 groupes jugés (largeur FIXE), 34 écartés (largeur VARIABLE), 6 écartés (moins de 8 records retenus)**, PASS en 2,58 s. Valeurs de `n2` par archétype, constantes sur les sept bobines : ti=14 `28`, ti=17 `432`, ti=21 `244`, ti=29 `256`, ti=47 `252` — les mêmes que le relevé B.2 de la note (qui donne 28, 432, 244, 252 ; `256` est le `128` de la note décalé d'un bit, cf. B.4) |
| 2026-09-14 | 1.3.2 (mutation) | `5f66051d0` | `consumeDefaultStateTI21` mis à `br.ReadBits(17)`, test rejoué, puis remis à 18 | **ROUGE sur les CINQ bobines qui portent des records ti=21** : `a521164d` (50 records), `60ae07c4` (87), `11de8353` (26), `e5adf7b2` (50), `bcb6d393` (144) — `n2` passe de `244` constant à `122:x30 2147483770:x20`. Remise à 18 : `git diff` du fichier VIDE, test vert |
| 2026-09-14 | 1.3.3 | `58733810e` | `go test ./...filmdec/ -run KeyframeClosureRatchet -update-keyframe-closure` puis la même commande SANS la porte | La porte **réécrit PUIS échoue** (« 1 reference(s) reecrite(s) … 10943 octets »), la passe de vérification est **ok**. `git diff -U0` du golden : **21 lignes `+`, 21 lignes `-`, et RIEN d'autre** ; les archétypes touchés sont exactement `ti=14 ti=17 ti=29`. Totaux recalculés depuis le golden par `awk` : ti=14 `3520/3520`, ti=17 `3729/3729`, ti=29 `102/110`, ti=21 `0/357`, ti=47 `0/1716` |
| 2026-09-14 | 1.3.3 (fermeture APRÈS) | `58733810e` | `CHUNK00_FILMS=<6 films de recherche> go test ./...filmdec/ -run TestImageCleFermetureParArchetype -v -timeout 30m` | `TOTAL \| 19337/62686 30.8% d15134 s7885 u20330 \| 391/62686 0.6% \| 0/62686 0.0%` ; `BILAN par archetype : etat complet GAGNE sur 11, PERD sur 0, EGALITE sur 22`. **30,8 % est la projection exacte du plan, MESURÉE.** Par archétype : ti=14 `5024/5024 100.0%`, ti=17 `5379/5379 100.0%`, ti=29 `138/157 87.9%`, ti=21 `0/373` (s373), ti=47 `0/1679` (d1679). Le plancher de hasard DESCEND (529 -> 391) : le gain n'est pas du bruit |
| 2026-09-14 | 1.3.4 | ce commit | `GrammarRev` `grammar-2026-09-14.2` -> `.3` ; entrée d'historique écrite dans le golden ; `-run GrammarRevSuitLaGrammaire -update-grammar-rev` puis sans la porte | Porte sortie en ÉCHEC comme elle le doit ; passe de vérification **ok**. `go test ./...filmdec/ -count=1` : **ok, zéro échec**. Empreinte FINALE du lot : `89fd5e547ac49293cca945ce94266c27cfd4d8dfe9d2c846dcf40d57d0d68b55` — elle a été régénérée DEUX fois dans le lot, la seconde après une relecture de mon propre diff qui a corrigé deux commentaires de `default_state_arch.go` (une date de pose affirmée sans preuve, remplacée par le commit qui l'établit `3f0ec70b3` ; un renvoi « l. 45-47 » remplacé par une citation de texte). **Révision INCHANGÉE** : même lot, et aucune largeur ne bouge |
| 2026-09-14 | 1.3.4 (killsource) | ce commit | `go test ./internal/games/halo_infinite/film/killsource/ -count=1` | **ok, 0,56 s** — le golden de la mini-bobine ne bouge pas. `KillSourceDecoderRev` NON montée : geste de prod réservé au pilote (D2 (1.2), toujours ouverte) |
| 2026-09-14 | 1.3 (gates communs) | ce commit | `gofmt -l ./internal ./cmd` ; `go vet` (6 racines) ; `go test -count=1` (6 racines) ; `golangci-lint run --timeout 20m --new-from-merge-base=origin/main` | gofmt **vide** ; vet **0 diagnostic** (9,5 s) ; **12 paquets ok, exit 0** en 18,7 s (filmdec 12,4 s · replay 14,4 s · archlint 14,1 s · replaybuild 0,99 s · killcollector 0,15 s · objectiveevents 0,53 s) ; lint **`0 issues.`**, exit 0, 1 min 41 (baseline non accrue) |
| 2026-09-14 | 1.3 (équivalence, régime COURT) | ce commit | `go run ./cmd/replay-equiv -repo-root C:/Users/Guillaume/Downloads/Scripts/LevelUp-wt-decfilm-13 -films …`, DEUX sous-ensembles de 5, séquentiels | Sous-ensemble 1 : `BILAN : 4 identique(s), 1 different(s), 0 ecarte(s), 0 echec(s), 0 illisible(s)` en 5 min 18 · sous-ensemble 2 : `BILAN : 5 identique(s), 0 different(s)…` en 2 min 41. Durées par film : `50247b26` 1 m 33,8 · `a521164d` 1 m 02,3 · `60ae07c4` 55,5 · `11de8353` 40,6 · `111fa685` 1 m 04,7 · `e5adf7b2` 42,1 · `bcb6d393` 13,2 · `51101d1d` 5,5 · `d9781168` 54,3 · `fb1a1a72` 40,8 s. Pics 0,08 à 0,37 Gio |
| 2026-09-14 | 1.3 (le SEUL écart, classé) | ce commit | `50247b26 : ECART a l'etape "killsource" : attendu compte=1 sha=3cade5d0…, obtenu compte=1 sha=2c4ebf20…` | **DIVERGENCE ATTENDUE**, famille « balayage delta d'un des cinq archétypes » : `killsource` traverse des records BRUTS, donc il voit les cinq états par défaut. Les 49 autres étapes, **`artifact` compris**, sont identiques sur les DIX films — donc aucun octet cuit ne change |
| 2026-09-14 | 1.3 (attribution de l'écart, PAR MESURE) | ce commit | `go run ./cmd/killsource json 50247b26` des deux côtés du correctif (fichier `default_state_arch.go` restauré par NOM à `783ae680d`, puis remis ; md5 identique après remise) | **228 800 octets des deux côtés, UNE seule valeur change** : `calibration` = `axisW=14 indexW=1 [PROFIL PLAT (score 88, mediane 77) : valeurs par defaut conservees] \| recordStateParam=2 [croissance x1.003]` -> la même avec `mediane 76`. Paramètres retenus, lignes de kill, catalogue, couverture, santé : IDENTIQUES. Découverte D2 (1.3) |
| 2026-09-14 | 1.3 (corpus gate) | ce commit | `go run ./cmd/replay-corpus-gate --base=783ae680d --parc-root …/LevelUp-go-migration --source-root …/LevelUp-wt-decfilm-13 --manifest …/config/replay_corpus.toml` | **13 témoins, 0 PERTE, 0 GAIN, exit 0**, schéma `55 -> 55` sur les treize, 19 min 33 s (26 cuissons, un seul décodage à la fois). Table collée depuis la sortie : `bcb6d393` ctf_mono_manche 11,59 s · `fb1a1a72` ctf_multi_manche 30,38 s · `d9781168` oddball 24,91 s · `c75f33b8` assaut_bombe 14,3 s · `bf15f7ab` slayer 14,02 s · `51ebbc0f` deux_manches 20,26 s · `084a804d` vehicules 1 m 54,05 s · `0797ce72` region_index_2_bits 15,59 s · `111fa685` version_39 49,28 s · `e5adf7b2` version_40_build_1_11 1 m 10,81 s · `60ae07c4` version_37 27,61 s · `a349fea8` version_33_sans_identification 2 m 11,62 s · `bfecd02b` vehicules_v41_utilisateur 22,11 s — tous `ok`. **ZÉRO GAIN est le résultat attendu** : le correctif ouvre la lecture d'image-clé, que rien en production ne consomme encore (le branchement est le lot 1.4) |
| 2026-09-14 | 1.3 (portée du lot, dite sans détour) | ce commit | lecture croisée des trois gates | Ce lot ne change AUCUN octet publié : 0 gain au corpus gate, `artifact` identique aux dix films de l'équivalence, `SchemaVersion` inchangée. Ce qu'il change est la GRAMMAIRE et sa mesure : cinq états par défaut relus chez l'écrivain, la fermeture d'image-clé de 14,0 % à 30,8 %, et un oracle permanent de plus. La valeur produit arrive au lot 1.4, qui branche le cadre d'état complet sur `navpoint_radial_scan` et `objective_scan` |
| 2026-09-14 | 1.4.0 | `44112034c` | BISECTION du témoin de marche delta, chaîne premier-parent `4ad72a4a1..8f35efb72` (690 points), worktrees détachés jetables, `CGO_ENABLED=0 DELTA_WITNESS_FILM=C:/…/LevelUp-go-migration/data/cache/film_chunks/<film> go test ./internal/games/halo_infinite/film/filmdec/ -run TestDeltaWalkWitness -v` | **LES SIX POINTS NOMMÉS PAR LE PILOTE** (paquets / records / aboutis, sortie brute) :<br>`figé 2026-08-18` — 000d5950 {14350, 38878, 30080} · 06dfe6d9 {6606, 10613, 8502} · 64e8adfa {14357, 39806, 31973}<br>`8f35efb72` (M0→v75) — {14350, **38892**, **30095**} · {6606, **10627**, **8499**} · {14357, **39826**, **31993**}<br>`fc5db87f7` — IDENTIQUE à `8f35efb72` sur les trois films<br>`9e6a9e08d` (1.0) — IDENTIQUE<br>`191933992` (1.1) — IDENTIQUE<br>`783ae680d` (1.2) — {14350, 38897, 30101} · {6606, 10629, 8499} · {14357, 39820, 31988}<br>`15309e89e` (1.3) — {14350, 38945, 30118} · {6606, 10636, 8505} · {14357, 39936, 31933}<br>**PREMIER POINT QUI BOUGE : le premier de la liste.** M0, 0.D, 1.0 et 1.1 ne bougent RIEN ; la dérive est antérieure au chantier |
| 2026-09-14 | 1.4.0 | `44112034c` | EXTENSION de la bisection en amont (témoin `06dfe6d9`, un seul worktree détaché réutilisé, 15 points) | `4ad72a4a1` 2026-08-18 **10 613 / 8 502 CONFORME** → `62ba098b8` 2026-09-01 **10 615 / 8 504** (merge `wt/bombe-visuel`) → `8f309ce86` 2026-09-02 **10 610 / 8 497** (merge `feat/precision-arme`) → `736ccf3c3` 2026-09-05 **10 610 / 8 489** (merge cuisson-perf + véhicules) → `ffb27238c` 2026-09-11 **10 627 / 8 499** (merge `wt/munitions-objet`, grammaire d'i9) → `8f35efb72` inchangé. Points intermédiaires vérifiés : 172, 258, 301, 323, 325 (conformes) ; 328, 334, 337, 338, 339 (après 62ba098b8) ; 383, 394, 399, 401 (après 8f309ce86) ; 405, 427, 515, 600, 606 (après 736ccf3c3) ; 607, 608, 609, 611, 622, 633, 645 (après ffb27238c) |
| 2026-09-14 | 1.4.0 | `44112034c` | SONDE JETABLE par archétype (ko/total des traversées, 12 premiers chunks, `06dfe6d9`), jouée de part et d'autre des trois points qui PERDENT des traversées — fichier de sonde supprimé après mesure, jamais commité | `8f309ce86` : **UNE SEULE ligne change de verdict, `ti=49` : 0/7 → 7/7** (−7 aboutis, exactement le delta global) ; ti=0 et ti=10 perdent chacun 1 record au total sans perdre d'abouti. `736ccf3c3` : ti=0 −6, ti=33 −1, ti=38 −1 (= −8). `ffb27238c` : gains répartis (ti=1, 6, 8, 11, 22, 36, 40, 41, 44), +10 |
| 2026-09-14 | 1.4.0 | `44112034c` | SONDE JETABLE du registre (`Archetype(i)` pour i de 0 à 63 sur `06dfe6d9`), de part et d'autre de `8f309ce86` | **AVANT : 64 archétypes résolus, ti=49 à 63 PRÉSENTS avec ZÉRO composant** (du bourrage — une traversée s'y termine sans rien lire, donc « aboutit »). **APRÈS : 49 archétypes, ti=49 ABSENT.** C'est la preuve que les 7 traversées perdues ne lisaient rien : DIVERGENCE, pas régression |
| 2026-09-14 | 1.4.0 | `44112034c` | `go test … -run TestDeltaWalkWitness` sur les trois films après re-figeage | **3/3 CONFORME au compte figé** (0,19 s / 0,41 s / 0,21 s). Re-joués à la clôture du lot : **toujours 3/3 CONFORME** — le cadre d'image-clé ne touche pas la marche delta |
| 2026-09-14 | 1.4 (mesure avant de coder) | `15309e89e` (base) | `CGO_ENABLED=0 CHUNK00_FILMS="<6 films de recherche>" go test … -run TestImageCleProductionBalayagesReels -v` | **AVANT** — ti=12 (production) : `KeyRecords 0` sur les SIX films — et ce 0 NE DIT RIEN du corpus : l instrument passe une horloge VIDE, et `scanChunk` saute tous les paquets sans `start_ms` (découverte D5 (1.4)). ti=11 (instrument) : `00162144` 1/1/0/0 · `00502e52` 1/1/0/0 · `00ba2e1c` 25/25/0/0 ; trois films sans slot ti=11. **CUMUL ti=11 : records 27 · marches 27 · CASSÉES 0 · chaînées 0** — et 0 fermée, cf. la ligne suivante. 9,57 s |
| 2026-09-14 | 1.4 (mesure avant de coder) | `15309e89e` (base) | même commande, `CHUNK00_FILMS="<les 7 bobines>"` | **AVANT** — ti=12 : `KeyRecords 0` sur les SEPT bobines (même artefact d horloge vide, cf. D5 (1.4)). ti=11 : a521164d 18/18/0/**0** · 60ae07c4 135/135/0/**20** · 11de8353 14/14/0/0 · 111fa685 13/13/0/0 · e5adf7b2 11/11/0/0 · bcb6d393 85/85/0/**17** · fb1a1a72 50/50/0/**10**. **CUMUL : records 326 · marches 326 · CASSÉES 0 · chaînées 47** (14,4 % sur la définition FAIBLE ; 0 sur la forte). 8,00 s |
| 2026-09-14 | 1.4 (mesure avant de coder) | `15309e89e` (base) | `CHUNK00_FILMS="<6 films>" go test … -run TestImageCleFermetureParArchetype -v` | **AVANT** — dernière ligne : `TOTAL | 19337/62686  30.8% d15134 s7885 u20330 | 391/62686 0.6% d15134 s13048 u34113 | 0/62686 0.0% d3589 s57737 u1360`. État complet 30,8 %, témoin de hasard 0,6 %, ancien cadre (colonne « production ») **0,0 %**. 3,53 s |
| 2026-09-14 | 1.4.1 | `2316eeee3` | `grep -rn "walkKeyframeBody\|keyframeBodyVariant" --include=*.go internal/ cmd/ \| cut -d: -f1 \| sort \| uniq -c` | **ZÉRO appelant de production.** 6 fichiers : `keyframe_record_walk.go` (10, sa déclaration) et CINQ instruments `_test.go` du paquet — `keyframe_biped_bitexact_test.go` (5), `keyframe_biped_fullstate_test.go` (10), `keyframe_record_walk_test.go` (3), `keyframe_writer_grammar_test.go` (1), `vehicules_v5b_controle_test.go` (4). D'où : UNEXPORTÉE, pas supprimée (cf. item 1.4.1) |
| 2026-09-14 | 1.4.2 | `e5b0d600f` | `grep -rn "walkOneKeyframeRecord(\|WalkKeyframeRecords(\|ChainKeyframeRecords(" --include=*.go internal/ cmd/` | 11 lignes, **aucune en production** : 4 dans `keyframe_record_walk.go` (déclarations + chaînage interne), 2 dans les instruments d'image-clé (`imagecle_fermeture_research_test.go:189`, `imagecle_production_research_test.go:124` — la colonne « ancien cadre »), 3 dans `keyframe_record_walk_test.go` / `zone_census_report_test.go`, 1 HORS paquet (`replay/visee_etiquettes_keyframe_test.go:94`). Justifie le `[~]` de `walkOneKeyframeRecord` et la découverte D3 (1.4) |
| 2026-09-14 | 1.4.1 | `2316eeee3` | `go test … -run TestCadreImageCleSansOptionEnProduction` (garde-rail neuf) | vert ; il lit les 113 fichiers non-test de `filmdec` et échoue si l'un d'eux cite `keyframeFullStateTemoin` ou `walkKeyframeFullState(` hors de `keyframe_fullstate_loop.go` |
| 2026-09-14 | 1.4.2 | `e5b0d600f` | `CHUNK00_FILMS="<6 films>"` puis `"<7 bobines>"`, `-run TestImageCleProductionBalayagesReels -v` | **APRÈS** — ti=12 : `KeyRecords 0` partout, avant comme après — artefact d horloge vide, la vraie mesure de ti=12 est celle du corpus gate et de `TestImageCleProductionCompteurs` (deux lignes plus bas). ti=11 : **6 films CUMUL records 27 · marches 0 · CASSÉES 27 (100,0 %) · chaînées 0** (10,71 s) ; **7 bobines CUMUL records 326 · marches 0 · CASSÉES 326 (100,0 %) · chaînées 0** (8,0 s). Désynchronisation à `i4 managed-objective-interaction-filter-component`, non porté — lot 3.6. Les 47 « chaînées » perdues ne fermaient AUCUN record |
| 2026-09-14 | 1.4 (ti=12, la VRAIE mesure) | `3edc44b75` | `CHUNK00_FILMS="<…/c75f33b8>" go test … -run TestImageCleProduction -v` (le témoin d'Assaut du corpus gate, lu SANS passer par l'horloge) | **569 records ti=12 en image-clé.** ANCIEN CADRE : 242 marches · 327 cassées (57,5 %) · 7 chaînées (1,2 %) · **0 FERMÉE / 569**. CADRE D'ÉTAT COMPLET : 0 marche · 569 cassées (100 %) · 0 chaînée · 0 fermée (butée `i1 managed-navpoint-flags-component`, lot 3.6). Le même film sous `TestImageCleProductionBalayagesReels` rend `KeyRecords 0` : **preuve de l'artefact d'horloge vide** (D5 (1.4)) |
| 2026-09-14 | 1.4 (corpus gate) | ce commit | `go run ./cmd/replay-corpus-gate --base=15309e89e --parc-root C:/…/LevelUp-go-migration --source-root C:/…/LevelUp-wt-decfilm-14` | **17 min 16 s, 13 témoins. 12 à ZÉRO gain / ZÉRO perte** (`bcb6d393`, `fb1a1a72`, `d9781168`, `bf15f7ab`, `51ebbc0f`, `084a804d`, `0797ce72`, `111fa685`, `e5adf7b2`, `60ae07c4`, `a349fea8`, `bfecd02b`), schéma 55 → 55 partout. **UN témoin en PERTE : `c75f33b8` (famille `assaut_bombe`), 0 gain / 2 pertes, toutes deux sur l'axe COUVERTURE** : `coverage.bombArmings.reads` **1169 → 1148**, `coverage.bombArmings.rises` **94 → 73**. **Le calque publié `bombArmings` est un axe mesuré du gate (`replaydiff/empreinte_axes.go:65`) et il NE BOUGE PAS** : aucun armement publié ne change. C'est la différence que la ligne « Preuve » du lot annonçait (« `bombArmings` si un témoin s'engage ») ; −21 lectures pour −21 montées = UNE montée par lecture, donc des points isolés, la signature du bruit |
| 2026-09-14 | 1.4.2 | `e5b0d600f` | `go test … -run GoldenMiniBobineFamilles -update-golden-familles` puis sans le drapeau | **UNE ligne régénérée**, `navpointRadial` : `f4eaa343…` → `ebac4160…`, **population VIDE avant ET après** (0 lecture). La bobine `minifilm_000d5950` n'a pas de `chunk_00` : le balayage sort sur bande vide, donc les deux compteurs neufs valent 0 et l'empreinte ne bouge que parce qu'elle hache la structure. Vert après régénération |
| 2026-09-14 | 1.4.3 | `3edc44b75` | `go test ./internal/archlint/ -run FilmdecPackageVars` | vert — `filmdecVarsGeles` reste à **96** : le compteur n'ajoute aucune variable de paquet (un type + une méthode + une fonction) |
| 2026-09-14 | 1.4.3 | `3edc44b75` | `go test … -run TestKeyframeClosureRatchet` | **VERT SANS RÉGÉNÉRATION** — et c'est le résultat attendu : ce golden mesure le cadre d'état complet depuis le lot 0.A.3, ce lot fait rejoindre la PRODUCTION à la mesure. Le « 0 → 14 % » du plan était une erreur de citation (découverte D2 (1.4)) |
| 2026-09-14 | 1.4.4 | ce commit | `go test … -run GrammarRevSuitLaGrammaire` AVANT la montée | **ROUGE de lui-même dès 1.4.1** : « LA GRAMMAIRE A CHANGE SANS MONTEE DE REVISION » — preuve par mutation naturelle, aucune mutation artificielle nécessaire |
| 2026-09-14 | 1.4.4 | ce commit | `go test … -run GrammarRevSuitLaGrammaire -update-grammar-rev` puis sans le drapeau | porte nommée : « 1 reference(s) reecrite(s) … (revision grammar-2026-09-14.4, empreinte e5e4291f0261…) », **FAIL** comme prévu ; relancé sans le drapeau : **ok**, 0,06 s. Historique complété DANS le golden |
| 2026-09-14 | 1.4 (communs) | ce commit | `gofmt -l ./internal ./cmd` | sortie **vide** |
| 2026-09-14 | 1.4 (communs) | ce commit | `go vet` + `go test` sur film/…, archlint, replaybuild, killcollector, objectiveevents, replaydoc | vet propre ; **tous verts** — filmdec 11,98 s · replay 13,84 s · archlint 13,78 s · replaybuild 0,88 s · objectiveevents 0,51 s · killsource 0,58 s · killcollector 0,11 s (celui-ci joué avec `CGO_ENABLED=1` et `msys64/ucrt64/bin` en tête du PATH : il tire DuckDB, et sans CGO il sort en `[setup failed]`) ; `replaydoc` sans fichier de test |
| 2026-09-14 | 1.4 (communs) | ce commit | `make go-api-lint` (`GOLANGCI_LINT_CACHE` isolé) | **0 issues**, baseline non accrue. Une seule remontée en cours de route, corrigée : `ST1016` (nom de receveur `sc` contre `s` déjà utilisé sur `NavpointRadialScan`) |
| 2026-09-14 | 1.4 (équivalence, régime court) | ce commit | `go run ./cmd/replay-equiv -repo-root C:/…/LevelUp-wt-decfilm-14 -films 50247b26,a521164d,60ae07c4,11de8353,111fa685` puis `-films e5adf7b2,bcb6d393,51101d1d,d9781168,fb1a1a72` | **9 identiques / 1 différent** en 3 min 48 s + 2 min 20 s. L'unique écart : `50247b26`, étape `killsource`, `3cade5d0…` attendu contre `2c4ebf20…` obtenu |
| 2026-09-14 | 1.4 (équivalence, contrôle) | `15309e89e` (worktree détaché jetable, deux jonctions posées puis DÉLIÉES par PowerShell avant `worktree remove`) | `go run ./cmd/replay-equiv -repo-root C:/…/LevelUp-wt-bisect-base14 -films 50247b26` | **MÊME écart, MÊME empreinte obtenue `2c4ebf2071ca…`** au commit de BASE du lot. L'écart est donc **hérité du lot 1.3** (découverte D2 (1.3), jamais re-figée), pas produit par 1.4. **Le lot 1.4 produit ZÉRO différence d'équivalence.** Cache du principal vérifié intact après retrait (1 351 films) |

| 2026-09-14 | 1.5.0 (oracle, AVANT tout changement) | `15bc6c82f` (base) | `CGO_ENABLED=0 CHUNK00_FILMS=<les 7 bobines> go test ./...filmdec/ -run TestResidusSlotChaineParBuild -v -count=1` | **7/7 films où le lecteur par grammaire calibré rend exactement le compte du balayage corrigé** (0,73 s). `a521164d` +1600 24+8 · `60ae07c4` -4320 8+24 · `11de8353` -4320 24+8 · `111fa685` -2880 24+8 · `e5adf7b2` -2880 23+9 · `bcb6d393` +0 8+24 · `fb1a1a72` +0 8+24. `s3sChaine` est MORT (1 enregistrement) sur les cinq builds anciens. |
| 2026-09-14 | 1.5.0 (oracle, AVANT tout changement) | `15bc6c82f` (base) | `CGO_ENABLED=0 CHUNK00_CORPUS=<cache> go test ./...filmdec/ -run TestResidusSlotChaineCorpus -v -count=1 -timeout 120m` | **152,0 s ; 1 351 chunk_00 lus** ; delta UNIQUE par build (`0` ×1 269, `-2 880` ×65, `-4 320` ×11, `+1 600` ×6) ; `rsChaine` = balayage corrigé sur **1 351/1 351** ; `s3sChaine` MORT sur **101** films ; **lus + vacants == 32 sur 1 351/1 351**. Tableau complet collé au rapport 1.5.0. |
| 2026-09-14 | 1.5.0 (sonde JETABLE, supprimée après mesure) | `15bc6c82f` (base) | balayage des 1 351 `chunk_00` : blocs de registre par `parseRegistry`, offset de `HI_` par `s3bBuild`, dérivation `(offset − 0x20) − fin du registre` | **3,7 s.** 50 blocs / 492 o / **123 entrées** sur les 1 269 films `HI_1_12_0`+`HI_1_13_0` ; 49 / 488 / **122** sur `HI_1_11_0` ; 49 / 484 / **121** sur `HI_1_10_0`, `HI_1_9_0`, `HI_1_8_0` ; 49 / 464 / **116** sur `HI_1_4_1`. FERMETURE SANS RESTE, et le 123 est la valeur de l'ÉCRIVAIN (`MOV R9D,0xf60`) là où `lireEntete` en comptait 124 — découverte D5 (1.5). **Les 5 films sans section NOMMÉS** : `03af54c3 13b00e35 47d20b5d 50247b26 a349fea8`. |
| 2026-09-14 | 1.5.0 (sonde JETABLE, supprimée après mesure) | `15bc6c82f` (base) | rang ABSOLU contre index parmi les OCCUPÉS, sur les films à vacant intercalé, oracle = `roster[].filmIndex` du parc de rejeu et de la sauvegarde du 2026-08-20 | **72,6 s ; 5 films à vacant intercalé trouvés par cette passe (`07f6af1b 0d1dddfb 1c5c10cc b1bcbe24 c744aa29`), ZÉRO avec document de rejeu** — la question n'est pas tranchable sur ce corpus (découverte D3 (1.5)). La passe du lecteur de production en trouvera 13 : elle part d'un balayage sans regroupement. |
| 2026-09-14 | 1.5.1 | `0475daa8b` | `go test ./...filmdec/ -run TestReadFilmIdentit -v -count=1` | **VERT**, 0,10 s. Les sept bobines rendent build, version, saveur, identifiant de build, changelist, blocs et cardinal de table conformes ; horodatages **ORDONNÉS par build**, 2023-09-02T11:14:38Z → 2026-07-23T20:48:06Z. |
| 2026-09-14 | 1.5.1 (entrée tronquée) | `0475daa8b` | `TestReadFilmIdentiteTronquee` : huit coupes, dont une sur une frontière de bloc | **VERT.** Deux attentes CORRIGÉES PAR LA MESURE : couper juste après le registre ou dans la chaîne de build rend `ErrChunk00Truncated`, pas `ErrNoFilmIdentity` — le registre s'arrête à sa fin STRUCTURELLE, donc un tampon coupé avant cette fin épuise la boucle de blocs, et la cause première est la troncature. Un film SANS section est donc testé autrement : `TestReadFilmIdentiteSansSection` efface les trois champs de chaîne d'un film sain. |
| 2026-09-14 | 1.5.2 (mesure qui a changé le code) | `f43bf03de` | première version bornée sur le dernier octet ÉCRIT : `go test … -run TestReadPlayerTable` | **ROUGE sur 7/7 bobines** (« aucune table de 32 slots »). CAUSE : un enregistrement VACANT est écrit entièrement à zéro, donc les 24 vacants de queue d'une partie d'arène tombent APRÈS le dernier octet non nul. Les LECTURES vont désormais jusqu'au bout du TAMPON ; le BALAYAGE, lui, garde la borne du dernier octet écrit. |
| 2026-09-14 | 1.5.2 (mesure qui a changé le code) | `f43bf03de` | critère « la marche ferme à 32 slots » seul, confronté à six largeurs fausses par bobine | **LE CRITÈRE NE MORD PAS** : `2052` ferme sur `60ae07c4` (profil 1312), `1852` sur `a521164d` (2052), `1492` et `2052` sur `11de8353`… La queue du tampon étant faite de zéros, le prédicat de vacance y passe indéfiniment et « 1 occupé + 31 vacants » ferme avec n'importe quelle largeur. Critère complété : la marche doit VISITER TOUS les enregistrements du balayage, et la lecture la plus complète gagne. Découverte D1 (1.5). |
| 2026-09-14 | 1.5.2 (mesure qui a changé le code) | `f43bf03de` | sonde JETABLE sur `d4ddf054`, refusé par la première forme du second volet | **9 candidats, 7 réels ; la marche ancrée au premier en lit 8 et les visite tous.** L'écart `10 170` bits entre deux candidats est plus court que le plus court enregistrement mesuré (16 611) : le balayage rate un enregistrement bien réel entre deux autres. Exiger l'ÉGALITÉ des deux suites jetait le film ; l'INCLUSION (la marche visite tous les candidats, elle peut en lire plus) le lit juste — et la trame confirme, 8 entités `ti=9`. Découverte D4 (1.5). |
| 2026-09-14 | 1.5.2 / 1.5.3 | `f43bf03de` | `go test ./...filmdec/ -run 'TestReadPlayerTable\|TestPersonnalisationOctetsProfil\|TestUnknownBuildCompteur' -v -count=1` | **VERT**, 0,39 s. Sept bobines : 32 slots (24+8, 8+24, 24+8, 24+8, 23+9, 8+24, 8+24), calibrage du film = celui du profil sur 7/7, **écarts 23/0/0/0/0 … 7/0/0/0/0 (accord / vacant / invisible / parasite / contradiction)**, 0 vacant de tête, 0 intercalé. Parasites du balayage : 1 sur cinq bobines, 0 sur deux. |
| 2026-09-14 | 1.5.4 (corpus) | `a2fbc3f01` | `CGO_ENABLED=0 CHUNK00_CORPUS=<cache> go test ./...filmdec/ -run TestTableJoueursCorpus -v -count=1 -timeout 120m` | **VERT, 81,8 s.** `1 346 film(s) lus a 32 slots (12080 occupes + 30992 vacants) ; 1337/1346 en accord avec l'ORACLE des instruments ; 0 contradiction(s) de grammaire ; 0 calibrage(s) en desaccord avec le profil ; 1 enregistrement(s) invisible(s) au balayage sur 1 film(s)`. `5 film(s) sans section d'identification (03af54c3 13b00e35 47d20b5d 50247b26 a349fea8) ; 0 film(s) a build inconnu`. `13 film(s)` à vacant intercalé. |
| 2026-09-14 | 1.5.4 (les NEUF divergences, classées) | `a2fbc3f01` | sortie du même test, colonne par colonne | `19ef6b04` 7+25 contre 4+28 · `23ffd885` 7+25 contre 1+31 · `3104391d` 7+25 contre 1+31 · `3b1cfde3` 7+25 contre 6+26 · `59b8abb9` 7+25 contre 4+28 · `652907bb` 7+25 contre 1+31 · `92f7c713` 7+25 contre 5+27 · `a92bab93` 7+25 contre 4+28 · `d4ddf054` 8+24 contre 5+27. **CHACUN porte 2 écarts de balayage au-delà de 40 000 bits**, et le test ÉCHOUE si une divergence n'a pas cette explication. Contrôle indépendant sur les mêmes films : `TestProfilRosterEcarts` rend « attendu (entités ti=9) 8 » sur 9/9, et « 2 écart(s) au-delà de 40000 ». |
| 2026-09-14 | 1.5.4 (mutation MANUELLE) | `a2fbc3f01` | `HI_1_13_0`/`HI_1_12_0` mis à **1 848** octets au lieu de 1 852, tests rejoués, fichier restauré PAR NOM | **ROUGE** : `bcb6d393` et `fb1a1a72` rendent « aucune table de 32 slots dans le corps de chunk_00 » ; `TestPersonnalisationOctetsProfil` et `TestReadPlayerTableTronquee` rougissent aussi. Restauration vérifiée : md5 `1d58470a8b22f70d33a3a1f5e190ea9a` avant et après, `git diff --stat` VIDE, tests verts. |
| 2026-09-14 | 1.5.5 | ce commit | `go test ./...filmdec/ -count=1` AVANT la montée | **ROUGE DE LUI-MÊME dès le commit de 1.5.1** : « LA GRAMMAIRE A CHANGE SANS MONTEE DE REVISION », empreinte figée `e5e4291f…` contre obtenue `f383983c…` (154 fichiers, +3). Preuve par mutation naturelle, aucune mutation artificielle nécessaire. |
| 2026-09-14 | 1.5.5 | ce commit | `go test ./...filmdec/ -run GrammarRevSuitLaGrammaire -update-grammar-rev` puis sans le drapeau | porte nommée : « 1 reference(s) reecrite(s) … relancer sans -update-grammar-rev pour verifier » (sortie en ÉCHEC, comme elle doit), puis **ok**. `grammar-2026-09-14.4` → `grammar-2026-09-14.5`, empreinte `4c85322c54e53e5834ea38a34a1093cb44a051a6dd882eb488523e57f60a6b14` (156 fichiers). Entrée d'historique écrite dans le golden. |
| 2026-09-14 | 1.5 (communs) | ce commit | `gofmt -l ./internal ./cmd` | sortie **vide** |
| 2026-09-14 | 1.5 (communs) | ce commit | `CGO_ENABLED=1 go vet` puis `go test -count=1` sur `film/…`, `archlint`, `replaybuild`, `killcollector`, `objectiveevents`, `replaydoc` (msys64/ucrt64 en tête du PATH) | vet **propre** ; **13 paquets ok en 18,9 s**, dont `filmdec` 13,15 s et `archlint` 14,31 s (`TestFilmdecPackageVarsNeCroitPas` VERT : le ratchet reste à **96**, les trois fichiers neufs n'ajoutent aucune variable de paquet — sentinelles en `const`, table de profil en `switch`). |
| 2026-09-14 | 1.5 (communs) | ce commit | `make go-api-lint` (`GOLANGCI_LINT_CACHE` isolé) | **0 issues**, 1 min 40 — baseline non accrue. |
| 2026-09-14 | 1.5 (équivalence, régime COURT) | ce commit | `go run ./cmd/replay-equiv -repo-root C:/…/LevelUp-wt-decfilm-15 -films …` — les 10 films, DEUX sous-ensembles séquentiels de 5 | **`BILAN : 5 identique(s), 0 different(s), 0 ecarte(s), 0 echec(s), 0 illisible(s)`** puis le même, soit **10/10 IDENTIQUES** (3 min 58 + 2 min 10). Les 50 étapes de balayage et l'artefact compris : aucun octet cuit ne change. |
| 2026-09-14 | 1.5 (portée, dite sans détour) | ce commit | `grep -rn "ReadFilmIdentity(\|ReadPlayerTable(" --include=*.go internal/ cmd/ \| grep -v _test.go` | **DEUX lignes, et ce sont les deux déclarations** (`film_identity.go:170`, `player_table.go:245`). Aucun appelant de production : le 10/10 de l'équivalence n'est pas une surprise, c'est ce que « lecteurs purs, sans consommateur » veut dire. |
| 2026-09-14 | 1.5 (corpus gate) | ce commit | `go run ./cmd/replay-corpus-gate --base=15bc6c82f --parc-root C:/…/LevelUp-go-migration --source-root C:/…/LevelUp-wt-decfilm-15` (les deux racines sont OBLIGATOIRES sur ce poste, §2.2 ; `--manifest` PREND UN CHEMIN, ce n'est pas un drapeau booleen) | **13 temoins sur 13 : 0 gain, 0 perte, schema 55 → 55 partout**, `EXIT=0`, ~9 min. `bcb6d393` 11,96 s · `fb1a1a72` 30,39 s · `d9781168` 24,56 s · `c75f33b8` 14,47 s · `bf15f7ab` 13,65 s · `51ebbc0f` 17,82 s · `084a804d` 1 min 53 · `0797ce72` 15,55 s · `111fa685` 34,79 s · `e5adf7b2` 39,58 s · `60ae07c4` 25,45 s · `a349fea8` 2 min 07 · `bfecd02b` 19,3 s. **Zéro différence, au sens littéral** — c'est ce que « lecteurs sans consommateur » doit donner. |
| 2026-09-14 | 1.5 (clôture) | ce commit | retrait d'une branche INATTEIGNABLE trouvée à la relecture : la famille « parasite » du contrôle des écarts | Le critère d'acceptation exige que la marche VISITE tous les enregistrements du balayage, donc les deux bouts de chaque couple du contrôle sont toujours retenus : `GapsParasite` ne pouvait plus incrémenter (mesure sur les 1 351 films : **0**). Retirée du rapport, du `switch` et des tests — règle 7, pas de branche morte. `GrammarRev` INCHANGÉE (même lot), empreinte reprise, corpus rejoué **VERT** (77,0 s, chiffres identiques). |

| 2026-09-14 | 1.6.0 (oracle, AVANT tout changement) | `06530e63d` (base) | sonde JETABLE (supprimée après mesure) : `filmdec.ReadFilmIdentity` + `ReadPlayerTable` sur les 8 films du golden, confrontés à `PlayerIndices` du fixture figé | **8/8 lus, 0 build inconnu, 0 vacant intercalé.** Sièges 8 · 24 · 8 · 24 · 24 · 23 · 8 · 8 ; table d'index 8 · 27 · 8 · 27 · 25 · 28 · 11 · 8 ; **125 accords, 0 contradiction** ; le film SEUL 1 sur `a521164d` et `11de8353` ; le contrôle SEUL 4 · 4 · 1 · 5 · 3 (**13 au total**). C'est cette dernière colonne qui a changé la conception du lot : remplacer la lecture des chunks aurait PERDU 13 joueurs (rapport 1.6.0). |
| 2026-09-14 | 1.6.0 | `2f32f576e` | `go test …/replay/ -run 'TestScanFilmPlayerTable\|TestCompteurAbsentVautZero' -count=1` | **VERT**, 0,29 s. Sept bobines à 32 slots avec gamertags imprimables et XUID non nul ; bobine historique sans `chunk_00` refusée par `sans_registre` ; trois coupes aux causes **MESURÉES** (`tampon vide` → `tronque`, `corps amputé de moitié` → `table_introuvable`, `coupe au début du corps` → `tronque` — les deux dernières avaient été écrites à l'envers avant la mesure) ; mutation `HI_9_99_0` : `filmdec_unknown_build_hi_9_99_0` passe de 0 à 1. |
| 2026-09-14 | 1.6.0 (fixtures d'entrées) | `2f32f576e` | magie v19 → v20, régénération des 8 fixtures depuis les films (`REPLAY_FILM_DIR` puis `REPLAY_FILM_CACHE`) | **8 réécrites**, 13,8 s + 2 min 38. Poids **11 045 116 o** pour un plafond de 12 582 912 (87,8 %). Les huit portent désormais `FilmInputs.FilmTable` ; aucun assemblage ne la lit encore, donc aucun golden ne bouge à ce commit. |
| 2026-09-14 | 1.6.1 | `c7b43802d` | `go test …/replay/ -run 'TestComposition' -count=1` | **VERT.** Six tests : le film prime sur le contrôle (index 3 gardé contre 9), le repli prend la voie `PlayerIndexTable`, accord/contradiction/silence comptés, la composition n'est JAMAIS plus pauvre que le contrôle (4 cas : table partielle, refusée, vide, vacant intercalé), le vacant intercalé s'abstient, les cinq causes de refus traversent jusqu'à la couverture. |
| 2026-09-14 | 1.6.1 (empreinte de forme) | `c7b43802d` | `REPLAY_CONTRACT_UPDATE=1 go test … -run DocumentShape -update` puis sans les drapeaux | La porte a REFUSÉ de refiger tant que `SchemaVersion` valait 55 — c'est le garde-rail qui impose la montée au PREMIER commit qui change la forme. Après montée : golden réécrit, **schéma 56, empreinte `2c1ea5c7b555c95f`**, puis vert. |
| 2026-09-14 | 1.6.1 (fixtures de contrat) | `c7b43802d` | `REPLAY_CONTRACT_UPDATE=1 go test … -run ContractFixtures -update` | **8 réécrites**, `replay_schema_55_*` supprimées (un seul jeu vivant). Poids final au lot : **2 565 193 o** pour un plafond de 3 145 728 (81,5 %). |
| 2026-09-14 | 1.6.1 (contrat web) | `c7b43802d` | `go run ./cmd/openapi-gen` (CGO, msys64/ucrt64 en tête du PATH) puis `npm run generate-types` | `openapi.yaml` **+37 lignes** (schéma `FilmTableCounts` et son `$ref` dans `IdentityCoverage`), `generated.ts` **+17 lignes**. 15,3 s + 0,3 s. |
| 2026-09-14 | 1.6.2 | `8346b6b0c` | `go test …/replay/ -run 'TestRosterPrendLesSieges' -count=1` puis les huit goldens | **VERT.** Gains lus dans le `git diff` des goldens : `a521164d` et `11de8353` « 27 joueur(s) » → « 28 joueur(s) » ; `111fa685` `idx=10 ""` → `idx=10 "FlukiestGolf"` ; `e5adf7b2` `idx=13 ""` → `idx=13 "MarshallG6443"`. Les quatre autres builds : identiques à l'octet. |
| 2026-09-14 | 1.6.3 | `444d0b7c6` | `go test …/replay/ -run 'TestRosterHorsLigne' -count=1 -v` | **VERT, 1,8 s.** Cuisson hors ligne, sans / avec la table du film : `000d5950` 8→8 · `a521164d` **26→27** · `60ae07c4` 8→8 · `11de8353` **26→27** · `111fa685` **24→25** · `e5adf7b2` **26→27** · `bcb6d393` 11→11 · `fb1a1a72` 8→8. **0 siège de la table du film absent du roster hors ligne sur 8/8.** |
| 2026-09-14 | 1.6.5 (oracle des 20 vies, IMPOSÉ par l'utilisateur) | ce commit | `go test …/replay/ -run 'TestViesDUnEchantillon' -count=1 -v` — tableau collé depuis la sortie brute | **20 vies · 1 mort écrite · 5 fins de film · 14 ORPHELINES.** `000d5950` slot 542 f1614 cut 8 125 ms ORPH · `a521164d` slot 517 f110 cut 41 226 ORPH, slot 525 f113 cut 88 556 ORPH, slot 524 f2447 **film_end** 8 506 FIN, slot 580 f1507 cut **954** ORPH, slot 579 f1600 cut 10 183 ORPH, slot 639 f3373 **film_end** 30 931 FIN · `60ae07c4` slots 512 f0 / 514 f0 / 519 f1 / 550 f1869, tous cut (30 184 / 35 071 / 56 928 / 10 105) ORPH · `11de8353` slot 525 f2 cut 126 473 ORPH, slot 529 f3 cut 299 737 ORPH, slot 563 f1329 **film_end** 10 233 FIN · `111fa685` slots 672 f3985 / 695 f4461 cut (10 153 / 10 213) ORPH · `e5adf7b2` slot 689 f4480 **death 72 ms MORT ÉCRITE**, slot 726 f5150 cut 10 211 ORPH · `bcb6d393` slots 539 f1568 / 544 f1762 **film_end** (10 118 / 10 134) FIN · `fb1a1a72` **aucune**. Découverte D1 (1.6). |
| 2026-09-14 | 1.6.5 (seuil) | ce commit | `git diff` des huit goldens après `DefaultMinPoints = 2 → 1` | `refusedMinPoints` **0 sur 8/8** ; traces 104→105 · 243→245 · 243→246 · 173→177 · 179→185 · 56→58 · 254→256 · 147→147, soit **+20**, exactement les vies refusées mesurées au lot 1.0.4. `TestBuildFromPositions_Decimation` réécrit : il affirmait « 1 seul point -> exclu », et l'exclusion est désormais testée par le seuil que l'APPELANT règle (`Options.MinPoints = 2`), seule voie qui la déclenche. |
| 2026-09-14 | 1.6 (communs) | ce commit | `gofmt -l ./internal ./cmd` | sortie **vide** |
| 2026-09-14 | 1.6 (équivalence, régime COURT) | ce commit | `go run ./cmd/replay-equiv -repo-root C:/…/LevelUp-wt-decfilm-16 -films …` — les 10 films de l'échantillon court, DEUX sous-ensembles séquentiels de 5 | **Première passe : 0/5 identiques**, et le message NOMMAIT la mauvaise cause (« ECART à l'étape "filmTable" : attendu sha=<celui de playerIndices> ») — le comparateur est POSITIONNEL, découverte D3 (1.6). Classification faite sur le `git diff` des références après `-update` (2 min 53 + 2 min 45), **et elle est complète** : les 10 fichiers portent EXACTEMENT deux lignes changées, `+filmTable` (l'étape neuve) et `artifact` modifié. **Les 49 autres balayages sont identiques sur les 10 films.** Deltas d'`artifact` : `fb1a1a72` +49 · `51101d1d` +148 · `111fa685` +274 · `e5adf7b2` +305 · `bcb6d393` +316 · `d9781168` +319 · `60ae07c4` +495 · `50247b26` +542 · `11de8353` +841 · `a521164d` +987. DIVERGENCES, aucune RÉGRESSION : re-figeage accepté après cette classification. Les 10 autres films du corpus seront re-figés à la clôture de M1. |
| 2026-09-14 | 1.6 (corpus gate) | ce commit | `go run ./cmd/replay-corpus-gate --base=06530e63d --parc-root C:/…/LevelUp-go-migration --source-root C:/…/LevelUp-wt-decfilm-16 --json …` | **16 min 33, 13 témoins, schéma 55 → 56 partout, 267 gains, 39 « pertes », exit 1.** Gains par témoin : `c75f33b8` 52 · `084a804d` 35 · `e5adf7b2` 27 · `60ae07c4` 26 · `111fa685` 24 · `bcb6d393` / `0797ce72` 22 · `d9781168` / `51ebbc0f` 20 · `a349fea8` 16 · `fb1a1a72` / `bf15f7ab` / `bfecd02b` 1. Le rapport N'ÉNUMÈRE PAS les gains (il ne détaille que les pertes) : leurs familles sont nommées par le `git diff` des goldens — `identity.coverage.filmTable.*` (bloc neuf), `identity.coverage.filmIndex.direct` (+1 sur deux builds), `coverage.tracks.published` / `publishedPoints`. Les 39 pertes en QUATRE classes : le seuil publié (`minPoints` 2 → 1, 13/13), les refus qui tombent (`refusedMinPoints` / `refusedPoints` → 0, 11/13), les bornes qui S'ÉLARGISSENT (`084a804d`, `e5adf7b2` — D5 (1.6)), et `unnamedLives` qui MONTE (`084a804d` 0 → 1, `a349fea8` 334 → 337 — D4 (1.6)). |
| 2026-09-14 | 1.6 (corpus gate, CONTRÔLE DÉCISIF) | ce commit | le même, `--base=444d0b7c6` (le commit de 1.6.3 : la table du film SANS le changement de seuil) | **16 min 30, 13 témoins, schéma 56 → 56, 250 gains, 39 pertes — LE MÊME ENSEMBLE DE PERTES, métrique par métrique et valeur par valeur.** Conclusion : la table du film (1.6.0 à 1.6.3) coûte **ZÉRO perte sur 13/13 témoins**, et les 39 pertes viennent TOUTES de `DefaultMinPoints = 2 → 1`. Ce qu'elle apporte se lit dans l'écart des gains entre les deux passes : **+1 sur neuf témoins, +3 sur `111fa685` et `e5adf7b2`, +1 sur `a349fea8` et `60ae07c4`** — et `fb1a1a72`, `bf15f7ab`, `bfecd02b` passent de 1 à 0 gain (leur unique gain venait du seuil). |
| 2026-09-14 | 1.6 (communs) | ce commit | `CGO_ENABLED=1 go vet` puis `go test -count=1` sur `film/…`, `archlint`, `replaybuild`, `killcollector`, `objectiveevents`, `replaydoc`, `replayview` (msys64/ucrt64 en tête du PATH) | vet **propre** (1,4 s) ; **13 paquets ok en 17,9 s**, dont `replay` 16,46 s et `archlint` 13,83 s. `TestFilmdecPackageVarsNeCroitPas` VERT : le ratchet reste à **96** — les deux fichiers neufs de `replay` n'ajoutent aucune variable de paquet (sentinelles en `const`, causes de refus en `const`). |
| 2026-09-14 | 1.6 (communs) | ce commit | `make go-api-lint` (`GOLANGCI_LINT_CACHE` isolé) | **0 issues**, 1 min 26 — baseline non accrue. |
| 2026-09-14 | 1.6 (communs, web) | ce commit | `make check-types` puis `make test-web` (après purge de `node_modules/.tmp`) | `tsc -b` **propre** ; vitest **711 fichiers, 7 629 tests verts**, 1 ignoré / 17 ignorés, 94 s. Rejoué APRÈS chaque régénération de fixtures (1.6.1, 1.6.2, 1.6.5). |
| 2026-09-14 | 1.6 (clôture) | ce commit | `MIN_RENDERABLE_SCHEMA_VERSION` | **27, INCHANGÉ** (`features/match-replay/model/replaySchemaStatusLogic.ts:43`) : un artefact 56 se rend comme un 55, et la matrice de compatibilité ne bouge pas. |
| 2026-09-14 | 1.6 (clôture, seuil de fichier) | ce commit | `wc -l build.go` avant / après le déplacement de `DefaultMinPoints` | `build.go` pesait **516 lignes au commit de base** — déjà au-delà des 500 du dépôt — et le lot l'avait porté à **534** : dette ACCRUE, ce que la règle 5 interdit. La constante et sa doctrine sont descendues dans `tracks_publication.go` (le fichier qui DÉCIDE quelles vies sont publiées, 181 → 202 lignes), déplacement pur ; `build.go` revient à **516**, exactement son poids de départ. `golangci-lint` rejoué : **0 issues**. |
| 2026-09-14 | 1.7.0 (mesure AVANT de coder, passe 1) | `943d8cf4b` (arbre propre) | `CHUNK00_FILMS=<22 films> CHUNK00_XUID_EQUIPES=<oracle> go test …/filmdec/ -run TestEquipeFilmOracleXuid -v -count=1` — oracle reconstruit par `cmd/diag_q` sur `data/backups/pre-chaine-2026-09-09/shared_matches_v2.duckdb` | **LA NOTE SE REJOUE À L'IDENTIQUE, 20,7 s.** Dernière ligne : `ORACLE EXTERNE : 16/18 films en accord TOTAL, 160/176 slots ; CONTROLE NEGATIF : 0 touches sur 576 decalages voisins`. Les 14 films d'arène à `accord 8/8`, `03af54c3` et `213a87dc` à **`accord 24/24`**, les deux FFA (`1950c59b`, `610363ee`) à `predit [1 2 3 4 5 6 7 8] ; lu [0 0 0 0 0 0 0 0] ; accord 0/8` — le film dit « aucune équipe », la base fabrique un camp par joueur. Quatre BTB comptés à part (cardinaux différents, défaut connu du lecteur de `chunk_00` sur les gros rosters). |
| 2026-09-14 | 1.7.0 (mesure AVANT de coder, passe 2) | `943d8cf4b` (arbre propre) | sonde jetable (supprimée après la mesure) sur les **18 films** de « 8 builds ∪ 13 témoins ∪ échantillon court », oracle = la même sauvegarde | **TABLEAU COLLÉ DEPUIS LA SORTIE BRUTE.** Colonnes : sièges de `chunk_00` · roster de la base · table des chunks de réplication · remplaçants · accord de l'appariement ORDINAL (1er vecteur ti=9 contre les sièges). `000d5950` 8 · 8 · 8 · 0 · **8/8** — `a521164d` 24 · 27 · 27 · 4 · 23/24 — `60ae07c4` 8 · 8 · 8 · 0 · **8/8** — `11de8353` 24 · 27 · 27 · 4 · 23/24 — `111fa685` 24 · 25 · 25 · 1 · **24/24** — `e5adf7b2` 23 · 28 · 28 · 5 · **23/23** — `bcb6d393` 8 · 11 · 11 · 3 · **8/8** — `fb1a1a72` 8 · 8 · 8 · 0 · **8/8** — `d9781168` 8 · 8 · 8 · 0 · **8/8** — `c75f33b8` 8 · 10 · 10 · 2 · **8/8** — `bf15f7ab` 8 · 8 · 8 · 0 · **8/8** — `51ebbc0f` 8 · 8 · 8 · 0 · **8/8** — `084a804d` 24 · 26 · 26 · 2 · **24/24** — `0797ce72` 8 · 8 · 8 · 0 · **8/8** — `a349fea8` **sans_section** · 25 · 25 · 25 · appariement impossible — `bfecd02b` 8 · 8 · 8 · 0 · **8/8** — `50247b26` **sans_section** · 30 · 29 · 29 · appariement impossible — `51101d1d` 8 · 10 · 10 · 2 · **8/8**. **LES DEUX « 23/24 » NE SONT PAS DES DÉSACCORDS** : le rang en écart est un siège dont le xuid est ABSENT de la feuille de match (`a521164d` idx=18 `manistoff`, `11de8353` idx=23 `Iskra 20252993`) — l'oracle prédisait un camp qu'il n'avait pas. Sur tout siège que la base connaît : **accord 100 %**. |
| 2026-09-14 | 1.7.0 (mesure AVANT de coder, passe 2) | idem | la même sonde, colonne `champA` (premier `R(6)` de l'état par défaut de ti=9) | **L'INDEX DE JOUEUR EST ÉCRIT DANS LE RECORD.** `champA` du premier paquet, dans l'ordre des slots, vaut la suite des `FilmIndex` des sièges, terme à terme, sur **18 films et 7 builds** : `[0 1 … 7]` sur les films d'arène, `[0 1 … 23]` sur les BTB, `[0 1 … 22]` sur `e5adf7b2`. Il est CONSTANT sur toute la vie de l'entité (histogramme à une seule valeur, 3 449 records). **CONTRÔLE QUI INTERDIT D'Y LIRE UN ORDINAL** : sur `50247b26` la suite lue est `[0 1 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20 21 22 24]` — TROUÉE. Sur les deux films sans section d'identification, où la table de `chunk_00` est refusée, `champA` rend quand même `0..23` / `0..24`. Découverte D1 (1.7). |
| 2026-09-14 | 1.7.0 (demande utilisateur du 2026-09-15) | idem | la même sonde, rapport `REMPL-FILM` — une ligne par arrivée | **36 LIGNES, 35 ARRIVÉES RÉELLES + 1 RECORD HORS DOMAINE.** `film | index | entité (slot) | arrivée (paquet, t ms) | fin (paquet, t ms) | désignateur | réutilisation`. `a521164d` : 24/2029/pk4 t1 079 747→pk15 t1 299 944/des 2 ×12 · 25/2581/pk9 t1 179 835→pk19 t1 380 082/des 1 ×11 · 26/2951/pk13 t1 259 936→pk19/des 2 ×7 · 27/2981/pk13→pk19/des 2 ×7 — tous index NEUFS. `11de8353` : **23/1789/pk8 t2 348 503→pk31 t2 808 976/des 2 ×24/INDEX RÉUTILISÉ — partant slot 1343, dernier paquet 7, t2 328 475** · 24/1790/pk8→pk31/des 2 ×24 · 25/2386/pk14 t2 468 628→pk31/des 1 ×17 · 26/2719/pk18 t2 548 713→pk31/des 1 ×14 · 27/2851/pk20 t2 588 759→pk31/des 1 ×12. `111fa685` : 24/2529/pk14 t6 910 561→pk29 t7 210 865/des 1 ×16 ; **+ le record HORS DOMAINE index 59, slot 5, un seul paquet (D2 (1.7))**. `e5adf7b2` : 23/1787/pk4 t9 297 518→pk29 t9 798 013/des 1 ×26 · 24/1799/pk4→pk29/des 2 ×26 · 25/2897/pk14 t9 497 712→pk29/des 2 ×16 · 26/2935/pk14→pk15 t9 517 733/des 1 ×2 · 27/3155/pk16 t9 537 768→pk29/des 1 ×14. `bcb6d393` : 8/2175/pk15 t8 902 188→pk18 t8 962 199/des 2 ×4 · 9/1888/pk9 t8 782 165→pk11 t8 822 173/des 2 ×3 · 10/1972/pk10→pk18/des 2 ×9 · 11/2065/pk12 t8 842 179→pk18/des 2 ×7. `c75f33b8` : 8/2693/pk20 t10 164 808→pk21 t10 184 809/des 1 ×2 · 9/1873/pk5 t9 864 751→pk20/des 1 ×16 · 10/2417/pk18 t10 124 799→pk21/des 1 ×4 · 11/2726/pk21→pk21/des 1 ×1. `084a804d` : 24/2222/pk7 t12 391 831→pk55 t13 352 813/des 1 ×49 · 25/2982/pk14 t12 531 974→pk55/des 1 ×42. `a349fea8` : 24/4374/pk26 t9 475 366→pk49 t9 935 734/des 2 ×24. `50247b26` : 25/1997/pk3 t6 465 943→pk13 t6 666 160/des 2 ×11 · 26/2015/pk4→pk15 t6 706 210/des 2 ×12 · 27/2226/pk7 t6 546 048→pk16 t6 726 223/des 2 ×10 · 28/2971/pk15→pk29 t6 986 486/des 2 ×15 · 29/3186/pk16→pk24 t6 886 377/des 2 ×9 · 30/3329/pk18 t6 766 249→pk28 t6 966 484/des 2 ×11. `51101d1d` : **6/1603/pk5 t2 744 424→pk6 t2 764 427/des 1 ×2/INDEX RÉUTILISÉ — partant slot 1309, dernier paquet 2, t2 684 409** · 9/1533/pk3 t2 704 414→pk10 t2 844 440/des 1 ×8 · 10/1752/pk7 t2 784 429→pk10/des 1 ×4. **VERDICT : 33 index NEUFS, 2 RÉUTILISÉS, 35 désignateurs STABLES sur 35, 0 entité réutilisée.** Découverte D-remplacants (1.7). |
| 2026-09-14 | 1.7.1 | `3382acb88` | `CGO_ENABLED=0 go test …/filmdec/ -run TestScanPlayerTeams -v -count=1` | **5 tests VERTS, 5,5 s.** E-LUE : les sept bobines rendent `0 inatteint, 0 divergence d'entité, 0 divergence d'index` — `a521164d` 262 records/26 index · `60ae07c4` 240/8 · `11de8353` 393/26 (27 entités : l'index 23 est servi par DEUX entités, et les deux disent le MÊME camp) · `111fa685` 337 records dont **1 hors domaine**/25 · `e5adf7b2` 261/25 · `bcb6d393` 144/12 · `fb1a1a72` 80/8. E-INDEX : l'index du film = le rang du siège, terme à terme, `24/24 · 8/8 · 24/24 · 24/24 · 23/23 · 8/8 · 8/8`. E-TEMOIN : le même champ relu à UN bit diffère sur **1 616 lectures sur 1 617** (le seul cas où il coïncide est sur `111fa685`). E-COUPE : dix troncatures (0, 1, 2, 8, 64, 512, 4 096, ¼, ½, n−1 octets), aucune panique, aucune lecture hors domaine. E-REFUS : film nil ET bobine sans `chunk_00` rendent `ArchetypeAbsent`. |
| 2026-09-14 | 1.7.1 (grammaire) | `3382acb88` | `grep -n "186" …/filmdec/player_teams.go` | **3 occurrences, TOUTES dans l'en-tête de doctrine** (lignes 24, 29, 30) : « ce nombre N'APPARAIT NULLE PART ICI », la dérivation `108 + 32 + 14 + 32`, et pourquoi un `186` en dur ne suivrait pas un changement de build. **Zéro dans le code.** |
| 2026-09-14 | 1.7.1 (grammaire) | `3382acb88` | `go test …/filmdec/ -run GrammarRevSuitLaGrammaire -update-grammar-rev` puis sans | `grammar-2026-09-14.5` → **`.6`**, empreinte `fdb77e17a94db7ea…`, historique complété dans le golden. Rejouée au lot 1.7.3 : **révision INCHANGÉE** (même lot, règle du suffixe), empreinte reprise une fois → `e5a873d8c52c1d5c…`. |
| 2026-09-14 | 1.7.2 (S5, hors ligne) | `7bfb0c6e5` | `go test …/replay/ -run 'TestGoldenAssembly$|TestGoldenBuildsAssembly' -count=1` après régénération, puis `git diff` des huit goldens | **S5 TENU SUR 8/8, SANS AUCUNE BASE.** Section `### EQUIPES` neuve dans chaque golden : `000d5950` 200 records ti=9 / **105 vies sur 105** portent une équipe / répartition `0 ×55 · 1 ×50` — `a521164d` 449 / **185/185** / `0 ×90 · 1 ×95` — `60ae07c4` 336 / **177/177** / `0 ×87 · 1 ×90` — `11de8353` 753 / **246/246** / `0 ×149 · 1 ×97` — `111fa685` 697 (1 rejeté) / **245/245** / `0 ×128 · 1 ×117` — `e5adf7b2` 687 / **256/256** / `0 ×131 · 1 ×125` — `bcb6d393` 144 / **58/58** / `0 ×25 · 1 ×33` — `fb1a1a72` 312 / **147/147** / `0 ×72 · 1 ×75`. **0 joueur non lu, 0 divergence sur 8/8** ; le contrôle est à `silence` partout (8, 28, 8, 28, 25, 28, 11, 8), ce qui EST la définition d'une cuisson sans feuille de match. Le roster porte `equipe=N` par ligne. |
| 2026-09-14 | 1.7.2 (chaîne du schéma) | `7bfb0c6e5` | `go test …/replay/ -count=1` (tout le paquet), puis `openapi-gen` + `generate-types` | `SchemaVersion` **56 → 57** ; chronique v57 ; empreinte de forme **`1344869006f05f16`** (re-figée UNE fois, après restauration PAR NOM du golden depuis git — la porte a d'abord REFUSÉ un second re-figeage au même schéma, et elle avait raison) ; `openapi.yaml` **+60 lignes** puis ajusté au pointeur (schéma `TeamCoverage`, `$ref` dans `Coverage`, champ `team` dans `RosterEntry`) ; `generated.ts` régénéré ; **8 fixtures de contrat `replay_schema_57_*` (2 566 758 o pour un plafond de 3 145 728, 81,6 %), les 8 fixtures 56 SUPPRIMÉES** ; codec du fixture d'entrées **v20 → v21** (`PlayerTeams` + `TeamScan`), 8 fixtures d'entrées régénérées (2 min 54) ; `MIN_RENDERABLE_SCHEMA_VERSION` **27, INCHANGÉ**. |
| 2026-09-14 | 1.7.3 | `aa1dc5ac4` | `FILM_CACHE_ROOT=C:/…/LevelUp-go-migration/data/cache go test …/objectiveevents/ -run TestExtractCTFCaptureCount -count=1` | **VERT, 0,3 s — LA VÉRITÉ TERRAIN TIENT APRÈS LE BASCULEMENT DE SOURCE.** `0f9550e5` (5 captures, split 5-0) et `53ce4390` (3 captures, split 1-2) rendent le MÊME partage par équipe une fois `team_id` pris à l'octet 37 au lieu de `match_participants`, et le test exige désormais `contradiction == 0`. |
| 2026-09-14 | 1.7.3 | `aa1dc5ac4` | `grep -rn "objectiveevents.Extract(" --include=*.go internal/ cmd/ \| grep -v _test.go` | **UNE seule ligne** : `cmd/diag_weapons_v3/process.go:37`. Aucun appelant de production — découverte D4 (1.7). |
| 2026-09-14 | 1.7.4 | `aa1dc5ac4` | `grep -rniE "equipe[^.]{0,40}(n.est pas\|pas) dans le film\|film ne porte (ni\|pas).{0,20}(camp\|equipe)" --include=*.go internal/ cmd/` | **0 occurrence** hors de la chronique (qui CITE la phrase pour dire qu'elle était fausse) et hors d'un log de recherche sans rapport (`r11_charges_research_test.go`, « ce film ne porte pas cet equipement »). Dix fichiers corrigés. |
| 2026-09-14 | 1.7 (communs) | `aa1dc5ac4` | `gofmt -l ./internal ./cmd` | sortie **vide** |
| 2026-09-14 | 1.7 (communs) | `aa1dc5ac4` | `CGO_ENABLED=1 go vet` puis `go test -count=1` sur `film/…`, `archlint`, `replaybuild`, `killcollector`, `objectiveevents`, `replaydoc`, `replayview` (msys64/ucrt64 en tête du PATH) | vet **propre** ; **13 paquets ok**, dont `replay` 18,0 s, `filmdec` 19,8 s, `archlint` 14,2 s. `TestFilmdecPackageVarsNeCroitPas` VERT : le ratchet reste à **96** — `player_teams.go` n'ajoute aucune variable de paquet (tout est `const`). |
| 2026-09-14 | 1.7 (communs) | `aa1dc5ac4` | `make go-api-lint` (`GOLANGCI_LINT_CACHE` isolé) | **0 issues** — baseline non accrue. |
| 2026-09-14 | 1.7 (communs, web) | `7bfb0c6e5` | `make check-types` puis `make test-web` (après purge de `node_modules/.tmp`) | `tsc -b` **propre** ; vitest **711 fichiers, 7 629 tests verts**, 1 ignoré / 17 ignorés, 108 s. **LE PREMIER `make check-types` A ROUGI SUR 19 LIGNES**, et c'est ce qui a imposé le POINTEUR : `roster[].team` en entier NU devenait REQUIS au contrat, donc toute fixture web construisant une entrée de roster cassait — et, plus grave, un artefact antérieur au schéma 57 se serait servi avec `team: 0`, c'est-à-dire « tout le monde dans le camp 0 ». |
| 2026-09-14 | 1.7 (équivalence, régime COURT) | `aa1dc5ac4` | `go run ./cmd/replay-equiv -repo-root C:/…/LevelUp-wt-decfilm-17 -films …` — les 10 films de l'échantillon court, DEUX sous-ensembles séquentiels de 5 | **CLASSIFICATION AVANT TOUT RE-FIGEAGE, faite sur les digests des dix enfants rejoués à part et comparés ligne à ligne aux références.** Résultat IDENTIQUE sur les 10 films : **exactement TROIS lignes changent sur 52, et les 49 autres balayages sont identiques**. (a) `flag` — l'entrée `FlagInput` perd son champ `TeamOf` : DIVERGENCE DE FORME DE L'ENTRÉE, aucun octet décodé ; (b) `playerTeams` — l'étape NEUVE du lot ; (c) `artifact` — le document cuit, `+63` (`60ae07c4`) à `+237` (`a521164d`) octets : `fb1a1a72` +93 · `d9781168` +64 · `111fa685` +150 · `50247b26` +166 · `e5adf7b2` +166 · `11de8353` +176 · `bcb6d393` +209 · `51101d1d` +217. **DIVERGENCES, aucune RÉGRESSION.** Re-figeage accepté APRÈS cette classification (3 min 45 + 2 min 02), puis passe de comparaison : **10 IDENTIQUES sur 10** (3 min 36 + 2 min 04). `git diff --stat` des références : **10 fichiers, 30 insertions, 20 suppressions** — trois lignes par fichier, pas une de plus. |
| 2026-09-14 | 1.7 (couverture des équipes, AVEC feuille de match) | `aa1dc5ac4` | lignes `rejeu : equipes lues dans le film` des dix cuissons de `replay-equiv` (qui cuit AVEC les faits, contrairement aux goldens) | **`film` / `accord` par film** : `60ae07c4` 8/8 · `fb1a1a72` 8/8 · `d9781168` 8/8 · `51101d1d` 10/10 · `bcb6d393` 11/11 · `111fa685` 25/25 · `e5adf7b2` 28/28 · **`a521164d` 28/27** · **`11de8353` 28/27** · `50247b26` **0** (film lu — 680 records — mais AUCUN roster : sans section d'identification la table d'index des chunks n'est pas injective et se fait écarter, donc il n'y a personne à qui donner l'équipe ; les deux nombres côte à côte disent exactement cela). Les deux « 28/27 » sont les sièges que la FEUILLE ne porte pas (`manistoff`, `Iskra 20252993`, nommés en 1.7.0) : un SILENCE du contrôle, pas une contradiction. `rejetes` 0 partout sauf `111fa685` (1, le record d'index 59) ; `divergences` **0 sur 10/10** ; `nonLus` **0 sur 10/10**. |
| 2026-09-14 | 1.7 (contrôle, preuve par l'absence) | `aa1dc5ac4` | `grep -c "la base CONTREDIT le film"` sur les journaux des 10 cuissons d'équivalence ET des 26 cuissons du corpus gate | **0.** L'avertissement est émis dès que `contradiction > 0` (cf. `logTeamCoverage`) : son absence sur 36 cuissons est la preuve que **la feuille de match ne contredit le film sur AUCUN joueur**. Même compte pour `equipes NON LUES` : **0** — aucun refus de balayage sur le corpus. |
| 2026-09-14 | 1.7 (corpus gate) | `aa1dc5ac4` | `go run ./cmd/replay-corpus-gate --base=943d8cf4b --parc-root C:/…/LevelUp-go-migration --source-root C:/…/LevelUp-wt-decfilm-17 --json …` | **ZÉRO PERTE SUR 13 TÉMOINS SUR 13, EXIT 0 — le premier lot de M1 dont le gate sort vert au sens littéral.** ~18 min, schéma **56 → 57 partout**. Gains : **9** sur onze témoins (`bcb6d393`, `fb1a1a72`, `d9781168`, `c75f33b8`, `bf15f7ab`, `51ebbc0f`, `084a804d`, `0797ce72`, `e5adf7b2`, `60ae07c4`, `bfecd02b`), **10** sur `111fa685`, **5** sur `a349fea8` (le film sans section d'identification : la moitié de ses compteurs d'équipe est à zéro, faute de roster). Durées : de 11,97 s (`bcb6d393`) à 2 min 10 (`a349fea8`), `084a804d` 1 min 51. **LE RAPPORT N'ÉNUMÈRE PAS LES GAINS** (il ne détaille que les pertes), et l'empreinte dit d'où ils ne peuvent PAS venir : `aplatir` ne pose de feuille que sous `coverage` et `bombStats` (`replaydiff/empreinte.go:193`), `mesurerTableau` ne mesure que les champs des éléments de `roster` et `tracks` (`empreinte_axes.go:80-85`). Les seuls champs neufs ou changés du document étant `coverage.teams.*`, `roster[].team` et `tracks[].team`, **les gains sont ceux-là et rien d'autre**. `CarrierTeamUnknown` : **aucune perte sur l'axe `ports` sur 13/13**, donc inchangé ou en baisse — la condition de la ligne « Preuve » est tenue. |
| 2026-09-14 | 1.7 (clôture) | ce commit | `logTeamCoverage` sur un chemin sans balayage | Une TROISIÈME cause de refus est nommée APRÈS la mesure : `non_balaye`. Un appelant qui assemble depuis des positions déjà décodées (`BuildFromPositions`, le chemin du collecteur de kills) ne balaie aucun film, et sa couverture d'équipes se serait lue comme « un balayage qui n'a rien trouvé ». D14 : un refus se NOMME. Aucun golden ni aucune fixture ne bouge (tous passent par `BuildFromFilm`, où `Component` est toujours posé) — 13 paquets Go rejoués verts après le changement. |

| 2026-09-14 | 1.8.0 (mesure AVANT de coder, passe 1) | `c6a3b751c` (arbre propre) | sonde JETABLE (supprimée après la mesure) : `killsource.Decode` sur les **17 films** de « 13 témoins ∪ échantillon court », `Result.Roster.IndexToName` confronté à `filmdec.ReadPlayerTable` index par index | **132,7 s. TABLEAU COLLÉ DEPUIS LA SORTIE BRUTE.** Colonnes : `film · build · refus · npl · hum · bot · sièges · vacInt · marge · accord · contradiction · film seul · inférence seule`. `bcb6d393` HI_1_12_0 11/11/5/8/false/1/**8**/0/0/3 · `fb1a1a72` HI_1_13_0 8/8/0/8/false/49/**8**/0/0/0 · `d9781168` 8/8/0/8/false/47/**8**/0/0/0 · `c75f33b8` 12/10/4/8/false/1/**8**/0/0/3 · `bf15f7ab` 8/8/0/8/false/24/**8**/0/0/0 · `51ebbc0f` 8/8/0/8/false/23/**8**/0/0/0 · `084a804d` HI_1_10_0 26/26/0/24/false/5/**24**/0/0/2 · `0797ce72` 8/8/0/8/false/31/**8**/0/0/0 · `111fa685` 24/24/0/24/false/8/**23**/**1**/0/0 · `e5adf7b2` HI_1_11_0 26/26/0/23/false/5/**22**/**1**/0/3 · `60ae07c4` HI_1_8_0 8/8/0/8/false/53/**8**/0/0/0 · `a349fea8` **sans section** 25/25/0/0/false/15/0/0/0/25 · `bfecd02b` 8/8/0/8/false/29/**8**/0/0/0 · `50247b26` **sans section** 27/27/0/0/false/0/0/0/0/27 · `a521164d` HI_1_4_1 27/27/0/24/false/0/**23**/**1**/0/3 · `11de8353` HI_1_9_0 26/26/0/24/false/4/**23**/**1**/0/2 · `51101d1d` 9/9/3/8/false/1/**8**/0/0/1. **TOTAL : 195 accords, 4 écarts, 199 sièges lus, 2 films au repli complet.** |
| 2026-09-14 | 1.8.0 (mesure AVANT de coder, passe 2) | `c6a3b751c` (arbre propre) | la même sonde, colonne `auFeed` (le gamertag du siège est-il au kill-feed ?), sur les 4 écarts de la passe 1 et sur les **13 films du cache à slot vacant INTERCALÉ** | **96,7 s. LES QUATRE ÉCARTS SONT DES JOUEURS QUE LE KILL-FEED NE NOMME PAS** : `111fa685` i10 `FlukiestGolf`(auFeed=**false**) contre `themaninblack42` · `e5adf7b2` i13 `MarshallG6443`(false) contre `Sergio98666` · `a521164d` i18 `manistoff`(false) contre `Brauhausmann` · `11de8353` i23 `Iskra 20252993`(false) contre `GenesisA1011`. Ce sont les MÊMES joueurs que le lot 1.6.2 avait vus publiés sans nom. **VACANTS INTERCALÉS, 119 accords sur 123** : `07f6af1b` 7/7 (marge 7) · `0d1dddfb` 7/7 (0) · `19ef6b04` 7/7 (23) · `3104391d` 7/7 (14) · `3b1cfde3` 7/7 (0) · `59b8abb9` 7/7 (0) · `652907bb` 7/7 (13) · `92f7c713` 7/7 (13) · `a92bab93` 7/7 (0) · `c744aa29` 7/7 (12) · `1c5c10cc` 22/23 (`probablybxllets` auFeed=false) · `b1bcbe24` 22/23 (`SerdarTsn` auFeed=**true**, marge **0**) · `23ffd885` 5/7 (`CR951802` auFeed=**true** et `Alpha122092` false, marge **0**). **LES DEUX SEULS ÉCARTS SUR UN NOM PRÉSENT AU FEED TOMBENT SUR UN FILM À MARGE NULLE**, c'est-à-dire où l'inférence se déclare elle-même ambiguë. **TOTAL DES DEUX PASSES : 314 accords sur 322 sièges, 30 films distincts, 8 builds.** |
| 2026-09-14 | 1.8.1 (défaut trouvé par la mesure) | avant `9fa12968d` | `TestGoldenMiniBobine` après le premier épinglage, puis diagnostic à deux passes (`buildRoster` avec et sans table, sur la même bobine) | **DIX LIGNES PUBLIÉES TOMBÉES À DEUX, PERMUTATION IDENTIQUE.** Les deux passes rendent `perm=[7 2 3 4 1 0 6 5]` et le même `IndexToName`, mais `kills` vaut 10 sans la table et **2** avec : `isBotIndex` lisait « présent dans `pin` », que la table remplit désormais aussi, donc les huit joueurs passaient pour des bots et leurs morts tombaient dans la population « mort de bot », jamais publiée. Corrigé (`seatPin` exclu du prédicat), re-mesuré **10 et 10**. Un test qui n'aurait regardé que la bijection n'aurait rien vu — d'où `TestUnSiegeDuFilmNEstPasUnBot`. |
| 2026-09-14 | 1.8.1 (golden mini-bobine) | `9fa12968d` | PRÉ-IMAGE relevée AVANT la porte : md5 `50329245245946faa0bd78bb4921d00d`, 77 lignes ; premier écart **ligne 75** ; puis `go test …/killsource/ -run TestGoldenMiniBobine -update` | **UNE ligne modifiée sur 77, six ajoutées.** `publication ligne par ligne : AUTORISEE (marge de bijection 3)` → `… (marge de bijection 0, bijection DETERMINEE (rien a inferer))`, plus la section `## PROVENANCE DES INDICES` (8 sièges, 8 indices LUS, 0 ajout, 0 repli, accord 8 / contradiction 0 / silence 0). **Les dix lignes publiées, la couverture, le contrôle négatif, les voies et la santé sont IDENTIQUES À L'OCTET.** |
| 2026-09-14 | 1.8.1 (mutation) | `9fa12968d` | `readFilmTable` neutralisée (retour `FilmTableNoSection` inconditionnel), suite du paquet rejouée, puis restauration PAR NOM | **QUATRE tests + le golden ROUGES** (`TestTableDuFilmLueSurLesDeuxBobines` sur les deux bobines, `TestTableDuFilmEpingleTousLesIndicesDeLaBobine`, `TestRefusDeTableNommeEtRepliComplet` sur trois coupes, `TestGoldenMiniBobine`). Fichier restauré par nom, **md5 identique `3067bc89601f951697c65a48cd562b30`**, paquet vert. |
| 2026-09-14 | 1.8.1 (entrée tronquée, obligatoire) | `9fa12968d` | `TestRefusDeTableNommeEtRepliComplet` : tampon vide, un octet, en-tête seul, registre amputé de moitié | **Aucune panique, aucune lecture partielle, quatre causes NOMMÉES.** Et la mesure a corrigé une attente écrite avant elle : couper à MOITIÉ rend `table_introuvable`, pas `tronque` — la section d'identification est encore lisible, c'est la TABLE qui ne ferme plus à 32 slots. Même observation qu'au lot 1.6.0 sur le chemin du rejeu. |
| 2026-09-14 | 1.8.2 (mutation naturelle) | `a50c4008a` | `go test ./internal/sync/killcollector/ -run TestKillSourceDecoderRevSuitLeDecodeur` au commit de 1.8.1, sans rien toucher au gate | **ROUGE DE LUI-MÊME : « LE DECODEUR A CHANGE »** — empreinte `4e7cf973…` → `729624c1…`. Aucune mutation artificielle n'a été nécessaire : le garde-rail a mordu sur le changement qu'il existe pour attraper. Puis `-update` : `killsource-2026-09-12` → **`killsource-2026-09-14`**, entrée d'historique datée ajoutée au golden. |
| 2026-09-14 | 1.8.2 (empreinte de grammaire) | `a50c4008a` | `go test …/filmdec/ -run GrammarRevSuitLaGrammaire` puis `-update-grammar-rev` | **ROUGE** (« LA GRAMMAIRE A CHANGE SANS MONTEE DE REVISION », 150 → **158** fichiers), puis **REVISION INCHANGÉE `grammar-2026-09-14.6`**, empreinte `e5a873d8…` → `8fcf0804…`. Décision explicite écrite dans l'historique du golden : aucune largeur, aucun cadre, aucun ordre de composants, aucun lecteur d'octets ne bouge — c'est le CONSOMMATEUR qui change, et `KillSourceDecoderRev` porte ce changement-là. Même nature que l'entrée `botSuffix -> BotSuffix`. |
| 2026-09-14 | 1.8.3 (oracle base, LECTURE SEULE) | ce commit | `go run ./cmd/diag_q "…/data/backups/pre-chaine-2026-09-09/shared_matches_v2.duckdb" "SELECT decoder_rev, COUNT(DISTINCT match_id), COUNT(*) FROM match_kill_events_latest GROUP BY 1 ORDER BY 2 DESC"` | **1 384 matchs, 138 807 lignes, TOUS sous une révision antérieure** : `killsource-2026-09-05` 792 matchs / 71 288 lignes · `killsource-2026-07-31` 589 / 67 085 · `highlight-credit-2026-08-01` 3 / 434. **LA SHARED DU PARC N'A PAS ÉTÉ LUE** : `Get-Process` montre un `server.exe` actif depuis `LevelUp-go-migration` (§2.2 — jamais de `read_only` forcé sur une DB tenue RW), et l'oracle suffit à l'ordre de grandeur du backlog. |
| 2026-09-14 | 1.8 (communs) | ce commit | `gofmt -l ./internal ./cmd` | sortie **vide** |
| 2026-09-14 | 1.8 (communs) | ce commit | `CGO_ENABLED=1 go vet` puis `go test -count=1` sur `film/…`, `archlint`, `replaybuild`, `killcollector`, `objectiveevents`, `replaydoc` (msys64/ucrt64 en tête du PATH) | vet **propre** ; **13 paquets ok**, dont `filmdec` 19,8 s, `replay` 18,4 s, `archlint` 14,2 s, `killsource` 1,03 s. `TestFilmdecPackageVarsNeCroitPas` VERT : le ratchet reste à **96**. |
| 2026-09-14 | 1.8 (communs) | ce commit | `make go-api-lint` (`GOLANGCI_LINT_CACHE` isolé) | **0 issues** — baseline non accrue. |
| 2026-09-14 | 1.8 (communs, intégration) | ce commit | `CGO_ENABLED=1 go test -tags=integration -p 1 ./internal/sync/killcollector/ -count=1` (le diff touche `internal/sync/`, §2.3 ; code de sortie relevé hors tube) | **ok, 12,97 s, EXIT=0.** |
| 2026-09-14 | 1.8 (communs, web) | ce commit | — | **AUCUN OCTET CUIT NE CHANGE** : `SchemaVersion` reste à **57**, aucun champ de `ReplayDocument` ne bouge, aucune fixture de contrat ni `openapi.yaml` ne sont touchés. `make check-types` et `make test-web` NON JOUÉS, et c'est dit plutôt que tu : le diff ne contient pas une ligne de `apps/web/`. |
| 2026-09-14 | 1.8 (baseline des tests) | ce commit | `git diff c6a3b751c..HEAD -- '*_test.go' \| grep -E "^-func Test"` | **0 ligne** : aucun test renommé ni supprimé, donc `.ai/baselines/tests_pre_migration.jsonl` ne bouge pas (leçon du lot 1.0). Quatre fichiers de test touchés, **+384 lignes**, dont deux fichiers neufs. |
| 2026-09-14 | 1.8 (équivalence, régime COURT — passe 1 : mesure) | `a50c4008a` | `go run ./cmd/replay-equiv -repo-root C:/…/LevelUp-wt-decfilm-18 -films …` — les 10 films de l'échantillon court, DEUX sous-ensembles séquentiels de 5 | **0 identique, 10 différents**, l'écart nommé étant à l'étape `killsource` sur **10 films sur 10**. Durées : `50247b26` 1 min 13 · `a521164d` 40,6 s · `60ae07c4` 28,6 s · `11de8353` 34,5 s · `111fa685` 37,4 s · `e5adf7b2` 48,4 s · `bcb6d393` 13,3 s · `51101d1d` 5,5 s · `d9781168` 27,1 s · `fb1a1a72` 33,6 s. Pics de 0,08 à 0,33 Gio. |
| 2026-09-14 | 1.8 (équivalence — CLASSIFICATION AVANT LE RE-FIGEAGE) | `a50c4008a` | `-update` sur les mêmes 10 films, puis `git diff` des `.tsv` : `git diff … \| grep -E "^[+-][a-zA-Z]" \| cut -f1 \| sort \| uniq -c` | **`10 +killsource` / `10 -killsource`, ET RIEN D'AUTRE.** 10 fichiers, 10 insertions, 10 suppressions, sur **520 lignes d'étapes** (52 par film) : les **510 autres — `artifact`, `killRefs`, `neutralDeaths`, `filmTable`, `playerTeams` comprises — sont IDENTIQUES À L'OCTET.** C'est la classification, et elle est décisive : `killRefs` et `neutralDeaths` sont des PROJECTIONS du même `Result`, donc si elles ne bougent pas, aucune ligne de kill ne bouge. La cause de l'écart de `killsource` est la FORME de l'objet observé — trois champs neufs — et non son contenu : `50247b26`, dont la table est REFUSÉE et dont la bijection est identique au bit près, change lui aussi. Découverte D4 (1.8). |
| 2026-09-14 | 1.8 (équivalence — passe de comparaison après re-figeage) | `a50c4008a` | la même commande SANS `-update`, deux sous-ensembles | **10 IDENTIQUES sur 10, exit 0 des deux côtés.** `BILAN : 5 identique(s), 0 different(s), 0 ecarte(s), 0 echec(s), 0 illisible(s)` ×2. Le re-figeage est déterministe et vérifié. |
| 2026-09-14 | 1.8 (corpus gate) | `6dda937c1` | `go run ./cmd/replay-corpus-gate --base=c6a3b751c --parc-root C:/…/LevelUp-go-migration --source-root C:/…/LevelUp-wt-decfilm-18` | **ZÉRO PERTE SUR 13 TÉMOINS SUR 13, EXIT 0** — et **ZÉRO GAIN**, ce qui se dit plutôt que se tait. ~17 min, schéma **57 → 57 partout**. Tableau collé : `bcb6d393` 12,27 s · `fb1a1a72` 31,13 s · `d9781168` 25,19 s · `c75f33b8` 14,71 s · `bf15f7ab` 14,1 s · `51ebbc0f` 18,43 s · `084a804d` **1 min 51** · `0797ce72` 13,06 s · `111fa685` 34,75 s · `e5adf7b2` 39,48 s · `60ae07c4` 26,09 s · `a349fea8` **2 min 11** · `bfecd02b` 19,95 s, tous `ok` à 0/0. **CE QUE CE ZÉRO-ZÉRO VEUT DIRE, ET IL FAUT L'ÉCRIRE** : le lot change la SOURCE du lien `indice -> joueur` sans changer une seule valeur publiée sur ce corpus, parce que là où l'inférence se trompait, les joueurs concernés n'ont AUCUNE ligne au kill-feed (ils n'ont ni tué ni sont morts) — donc aucune ligne de kill ne les porte. Le gain du lot est d'ordre structurel (une lecture remplace une inférence, et elle est comptée), pas métrique sur ces treize témoins. La ligne « Preuve » du bloc est tenue sur les trois axes qu'elle nomme : kills, morts, sources — 0 perte. |
| 2026-09-14 | 1.8 (D3, mesure qui désamorce le report) | ce commit | sonde JETABLE (supprimée après la mesure) : `resolvePlayerIndices` — la voie d'inférence des TIRS et des TOUCHES nommée par le brief — confrontée à `filmdec.ReadPlayerTable` sur les 12 témoins lisibles, roster d'entrée = les xuids de la table | **143 sièges, 143 ACCORDS, 0 contradiction, 0 non résolu, 0,33 s.** `bcb6d393` 8/8 · `fb1a1a72` 8/8 · `d9781168` 8/8 · `c75f33b8` 8/8 · `bf15f7ab` 8/8 · `51ebbc0f` 8/8 · `084a804d` **24/24** · `0797ce72` 8/8 · `111fa685` **24/24** · `e5adf7b2` **23/23** · `60ae07c4` 8/8 · `bfecd02b` 8/8 (`a349fea8`, sans section, écarté). Le « 77 % d'accord » de l'en-tête de `hits.go` ne mesurait donc pas sa LECTURE mais son accord avec l'oracle killsource — c'est-à-dire avec la bijection que ce lot vient de remplacer. Découverte D3 (1.8) amendée en conséquence. |

| 2026-09-14 | 1.9.0 (recensement, AVANT de coder) | `d473cbd79` (arbre propre) | relecture sur pieces des **98 sites** cites par la table (E) de l'audit 0.E, un par un (`sed -n "<ligne>p"` sur les 60 fichiers, puis `awk` de la fonction englobante) | **59 des 60 lignes EXISTENT ENCORE**, ancre relue. **UNE convertie** : `replay/build.go:580` (`Team: -1` inconditionnel) — le lot 1.7 en a fait une INITIALISATION (`tracks_publication.go`, `Team: -1`, puis `build.go:130 equipes.poserSurLesTraces` pose le designateur du film ; ce qui reste est compte par `coverage.teams.{noTeam, unread}`). **UN site disparu** dans une ligne qui subsiste : `grep -c ParseUint internal/replaybuild/matchfacts.go` rend **0** (la regle a demenage dans `replay.RosterXUIDsOf` au lot 1.0, `roster_xuids.go:33`). D1 et D2 (1.9.0) en §4. |
| 2026-09-14 | 1.9.0 (registre) | ce commit | `go test …/replay/fallback/ -run 'RegistreEstStructurellement\|RegistrePorteToutes\|CompteurBranche' -v` | **VERT.** « compteurs branches : 10 sur 95 entrees ». Repartition mesuree — **paquet** : `film/replay` 37, `film/killsource` 18, `analysis/objectiveevents` 12, `film/filmdec` 10, `replaybuild` 9, `sync/killcollector` 9. **condition** : `non_resolu` 53, `section_absente` 17, `film_muet` 11, `inconditionnel` 7, `contradiction` 4, `lecture_non_portee` 3. **ordre** : `apres_lecture` 77, `sans_lecture` 11, **`devant_la_lecture` 7** (le ratchet neuf, D4 (1.9.0)). |
| 2026-09-14 | 1.9.0 (ratchet, direction A — mutation) | ce commit | `func repliBidon() int { return 0 }` ajoute a `replay/identity.go`, `go test ./internal/archlint/ -run ReplinNomme`, puis restauration PAR NOM | **ROUGE A LA MUTATION** : « REPLI HORS REGISTRE (1) : repliBidon (internal/games/halo_infinite/film/replay/identity.go) ». **VERT apres restauration** (`git status --porcelain` sur le fichier : vide). |
| 2026-09-14 | 1.9.0 (ratchet, direction B — mutation) | ce commit | ancre de `repli_largeurs_mpp_par_defaut` changee en `"var mppLeadBits = 9 // CONVERTI"`, `go test ./internal/archlint/ -run SiteDuRegistre`, puis restauration | **ROUGE** : « l'ancre est absente de …/filmdec/default_state.go … Si le repli a ete CONVERTI, retirer son entree du registre (D14 d) ». **VERT apres restauration.** C'est D14 (d) rendu mecanique. |
| 2026-09-14 | 1.9.0 (ratchet, couverture du parcours) | ce commit | `find internal/games/halo_infinite/film internal/replaybuild internal/sync/killcollector internal/analysis/objectiveevents -name '*.go' ! -name '*_test.go' -not -path '*/testdata/*' -not -path '*/fallback/*' \| wc -l` | **351** fichiers de production scannes ; plancher du ratchet pose a **250** (un ratchet qui ne scanne rien passe en silence). |
| 2026-09-14 | 1.9.0 (sortie humaine) | ce commit | `go test …/replay/fallback/ -run RapportDuRegistre -v` | « REGISTRE DES REPLIS — 95 entrees, 6 paquets », puis une fiche par repli (fait, cible et critere de retrait, sites). |
| 2026-09-14 | 1.9.0 (table par build, VERSIONNEE) | ce commit | bloc « REPLIS DECLENCHES » ajoute au rendu des goldens d'assemblage, puis `-update-golden-builds-assembly` et `-update` ; `git diff` des huit goldens | **CHAQUE GOLDEN PORTE DESORMAIS SA TABLE.** `000d5950` origine_pose 250 · `111fa685` fin_vehicule 9 + origine_pose 493 · `11de8353` 11 + 379 · `60ae07c4` 2 + 184 · `a521164d` 38 + 212 · `bcb6d393` origine_pose 92 · `e5adf7b2` 10 + 506 · `fb1a1a72` origine_pose 319. Les huit autres compteurs cables restent a 0 sur ce banc : leurs canaux ne sont pas au fixture (D7 (1.9.0)). |
| 2026-09-14 | 1.9.0 (chaine du schema) | ce commit | `SchemaVersion` 57 -> 58 ; chronique v58 ; raison ecrite dans `structure_test.go` ; `REPLAY_CONTRACT_UPDATE=1 … -run DocumentShape -update` ; `… -run ContractFixtures -update` ; `go run ./cmd/openapi-gen` ; `npm run generate-types` | empreinte de forme **cd9d54d2ef218027** (schema 58) ; **8 fixtures 58** ecrites, **2 567 380 o** au total (plafond 3 145 728), **8 fixtures 57 supprimees** ; `openapi.yaml` +18 lignes (`FallbackHit` + `fallbacks`), `generated.ts` +6. |
| 2026-09-14 | 1.9.0 (frontiere web, D11) | ce commit | `make check-types` puis `make test-web` | `tsc -b` **propre** apres avoir comble `coverage.fallbacks` a la frontiere (`replayNormalize.ts`, `ReplayCoverageReady`, chemin ajoute a `NULLABLE_ARRAY_PATHS`) — les deux assertions de type `_CarteExhaustive` / `_FrontiereProfonde` ont attrape le tableau nullable neuf, ce pour quoi elles existent. vitest **711 fichiers, 7 629 tests verts**, 1 ignore, 17 sautes, 103 s. **Aucune retouche de rendu.** |
| 2026-09-14 | 1.9.0 (communs) | ce commit | `gofmt -l ./internal ./cmd` | sortie **vide** |
| 2026-09-14 | 1.9.0 (communs) | ce commit | `CGO_ENABLED=1 go vet` puis `go test -count=1` sur `film/…`, `archlint`, `replaybuild`, `killcollector`, `objectiveevents`, `replaydoc`, `replayview` (msys64/ucrt64 en tete du PATH) | **0 diagnostic** au vet ; **tout ok** : filmdec 19,2 s · replay 17,9 s · `replay/fallback` 0,17 s · archlint 13,6 s · replaybuild 0,8 s · killcollector 0,12 s · objectiveevents 0,49 s · replayview 0,29 s · replaydoc « no test files ». |
| 2026-09-14 | 1.9.0 (communs) | ce commit | `make go-api-lint` (`GOLANGCI_LINT_CACHE` isole) | **0 issues** — baseline non accrue. Trois constats corriges avant de reverdir : 2 `goconst` (cibles de comptage repetees -> constantes `comptageStatborg` / `comptageLot1911`) et 1 `prealloc`. |
| 2026-09-14 | 1.9.0 (garde-rails du depot) | ce commit | trois rouges du REGISTRE lui-meme, tous fermes | `no_french_label_literal_test.go` (339 litteraux accentues -> les CHAINES du paquet s'ecrivent sans accent, allowlist NON agrandie) ; `no_identity_bridge_outside_registry_test.go` (une ancre citait `.SlotXUID` — ancre deplacee sur la garde voisine) ; `filmdec/world_object_precision_guard_test.go` (un commentaire nommait le global). D6 (1.9.0) en §4. |
| 2026-09-14 | 1.9.0 (baseline des tests) | ce commit | `git diff d473cbd79..HEAD -- '*_test.go' \| grep -E "^-func Test"` | **0 ligne** : aucun test renomme ni supprime (quatre ajoutes), donc `.ai/baselines/tests_pre_migration.jsonl` reste inchange. |
| 2026-09-14 | 1.9.0 (equivalence, regime COURT — passe 1 : mesure) | ce commit | `go run ./cmd/replay-equiv -repo-root C:/…/LevelUp-wt-decfilm-190 -films …` — les 10 films de l'echantillon court, DEUX sous-ensembles de 5 | **10/10 : une seule etape bouge, `artifact`.** Les **51 autres etapes sont IDENTIQUES sur les 10 films** — aucun balayage ne change, ce qui est la definition du lot (declarer et compter, ne rien decider). |
| 2026-09-14 | 1.9.0 (equivalence — CLASSIFICATION AVANT LE RE-FIGEAGE, a l'octet) | ce commit | pour chaque film : longueur attendue du bloc `,"fallbacks":[…]` recalculee depuis les lignes `rejeu : repli declenche` du journal de cuisson, confrontee au delta d'octets de l'etape `artifact` | **LES DIX DELTAS SONT EXACTEMENT LE BLOC NEUF, A L'OCTET PRES** : `d9781168` 127=127 · `60ae07c4` 189=189 · `51101d1d` 126=126 · `50247b26` 190=190 · `a521164d` 190=190 · `11de8353` 190=190 · `111fa685` 189=189 · `e5adf7b2` 190=190 · `bcb6d393` 184=184 · `fb1a1a72` 184=184. Aucun autre octet du document ne bouge. |
| 2026-09-14 | 1.9.0 (equivalence — comptes de PRODUCTION, 10 films) | ce commit | les memes journaux, une ligne par repli declenche | `origine_pose_vie_la_plus_proche` **41 a 506** par film (10/10) ; `plafond_grenade_par_defaut` **1 sur 10/10** — la cuisson appelle TOUJOURS `ScanKeyframeInventory` sans plafond, donc TOUJOURS le defaut de 2 grenades ; `fin_de_vie_vehicule_par_recensement` 2 a 38 sur 6 films ; `piste_drapeau_sans_pont_ecartee` 4 (`bcb6d393`) ; `position_lacher_prend_la_prise` 2 (`fb1a1a72`). Les cinq autres compteurs cables : **0 sur les 10** (colline, bombe, largeurs d'axe, identite par recouvrement) — un zero LISIBLE, puisque leur compteur EST cable. |
| 2026-09-14 | 1.9.0 (equivalence — re-figeage puis passe de comparaison) | ce commit | `-update` sur les 10 films, `git diff` des `.tsv`, puis la meme commande SANS `-update` | `git diff --stat` : **10 fichiers, 10 insertions, 10 suppressions** — une seule ligne par film, `artifact`. Passe de verification : **10/10 identiques, 0 ECART, exit 0 des deux sous-ensembles.** |
| 2026-09-15 | 1.9.0 (corpus gate) | ce commit | `go run ./cmd/replay-corpus-gate --base=d473cbd79 --parc-root C:/…/LevelUp-go-migration --source-root C:/…/LevelUp-wt-decfilm-190 --json …` | **ZERO PERTE SUR 13 TEMOINS SUR 13, exit 0**, 16 min 30. Schema 57 -> 58 partout. **TROIS gains par temoin, les memes sur les treize, tous nommes** (verifies en rejouant `replaydiff.Comparer` sur la paire de fixtures de contrat 57/58 de `bcb6d393`, outil jetable supprime apres la mesure) : `entete / schemaVersion` 57 -> 58 · `couverture / coverage/n` **28 -> 29** (la couverture porte une cle de plus) · `couverture / coverage.fallbacks/n` **apparu** (le nombre de replis declenches par la cuisson). Rien d'autre ne bouge. |
| 2026-09-15 | 1.9.0 (corpus gate — pertes attendues) | ce commit | comparaison avec le bloc « Cloture M1 » du plan | Les deux pertes deja classees (`c75f33b8` `coverage.bombArmings.reads` 1 169 -> 1 148 et `.rises` 94 -> 73) **n'apparaissent PAS ici, et c'est correct** : elles sont relatives a `783ae680d` (avant le lot 1.4), alors que la base de ce lot est `d473cbd79` (apres) — elles sont donc DEJA dans la base. Le regime complet de la cloture de M1 les retrouvera. |
| 2026-09-15 | 1.9.0 (seuils de fichier) | ce commit | `wc -l` sur les fichiers touches, confronte a leur taille au commit de base | `build.go` **530 -> 532** et `coverage.go` **445 -> 460** : les deux plafonds du depot etaient DEJA depasses avant ce lot (530 et 445 pour un seuil de 500 / 500), donc la publication a ete extraite dans un fichier neuf, `fallbacks_publication.go` (**63 lignes**), et l'initialisation du compteur dans un accesseur d'`Options` — l'assemblage n'appelle plus que deux lignes. Le registre lui-meme tient en six fichiers, **461 lignes au plus** (`registre_killsource.go`). |
| 2026-09-15 | 1.9.1 bis (pas 1, commun) | ce commit | `gofmt -l ./internal ./cmd` | sortie vide |
| 2026-09-15 | 1.9.1 bis (pas 1, commun) | ce commit | `CGO_ENABLED=0 go vet ./internal/games/halo_infinite/film/... ./internal/archlint/ ./internal/replaybuild/ ./internal/analysis/objectiveevents/ ./internal/domain/replaydoc/ ./internal/service/replayview/` puis `CGO_ENABLED=1 go vet ./internal/sync/killcollector/` (msys64/ucrt64 en tête du PATH) | 0 diagnostic dans les deux cas |
| 2026-09-15 | 1.9.1 bis (pas 1, commun) | ce commit | `go test` sur les mêmes paquets + `./internal/sync/killcollector/` (CGO) | ok — filmdec 33,3 s · replay 18,6 s · archlint 35,1 s · replaybuild 1,0 s · objectiveevents 0,5 s · replayview 0,3 s · killcollector 0,1 s ; `replaydoc` « no test files ». **Aucun test renommé ni supprimé** : `.ai/baselines/tests_pre_migration.jsonl` intouché, et c'est correct (trois tests NEUFS seulement) |
| 2026-09-15 | 1.9.1 bis (pas 1, commun) | ce commit | `make go-api-lint` (`GOLANGCI_LINT_CACHE` isolé) | **0 issue**, baseline non accrue. Les trois fichiers neufs font 469 / 191 / 136 lignes — sous le seuil de 500 |
| 2026-09-15 | 1.9.1 bis (pas 1, gate ajouté par le pilote) | ce commit | `CGO_ENABLED=1 go test ./internal/api/ -run TestOpenAPIYAMLIsUpToDate -count=1` | **ROUGE À LA PRISE EN MAIN, ET LE ROUGE EST HÉRITÉ** : `byFamily` / `byCause` écrits à la main au lot 1.9.1 dans un ordre que le générateur ne produit pas (`deba3261f`). Réparé par la porte prévue — `make openapi-gen` (725 148 octets réécrits, 2 lignes) puis `make generate-types` (`generated.ts`, 2 lignes) —, JAMAIS à la main. Test **VERT** après. `make check-types` vert ; `npx vitest run src/lib/api/generated-types-fresh.guard.test.ts` **1 passed** |
| 2026-09-15 | 1.9.1 bis (pas 1, régime COURT) | ce commit | `go run ./cmd/replay-equiv -repo-root C:/…/LevelUp-wt-decfilm-191b -films …` en DEUX sous-ensembles (échantillon court de `CORPUS.txt:128`) | `50247b26,a521164d,60ae07c4,11de8353,111fa685` : **BILAN 5 identique(s), 0 différent(s)** (4 min 10 s) ; `e5adf7b2,bcb6d393,51101d1d,d9781168,fb1a1a72` : **BILAN 5 identique(s), 0 différent(s)** (1 min 59 s). **10/10 identiques** — attendu : le pas ne touche que des `_test.go` |
| 2026-09-15 | 1.9.1 bis (pas 1, MESURE [1]+[2] — la carte) | ce commit | `CGO_ENABLED=0 go test ./internal/games/halo_infinite/film/filmdec/ -run '^TestE191bCarteTI37$' -v -count=1` (13,1 s, 7 bobines versionnées, aucune garde d'environnement) | **ti=37 : 3 fermés / 3 331 bornés, 0 désynchronisation.** Par bobine : `a521164d` 1/762 · `60ae07c4` 0/425 · `11de8353` 1/514 · `111fa685` 1/569 · `e5adf7b2` 0/489 · `bcb6d393` 0/352 · `fb1a1a72` 0/220. Résidu `EndBit-Want` : **1 109 valeurs distinctes** (la plus fréquente `+50`, 37 fois). Composant qui FAIT franchir la frontière, cumul : **i15 655 (w~570) · i6 341 (w~664) · i14 305 (w~113) · i9 296 (w~1 508 300) · i17 266 (w~87) · i7 188 (w~317) · i10 74 · i28 71** (+14 autres). Largeurs consommées : i0 {45:2380, 60:481, 44:470} · i4 {11:3331} · i11 {1:3331} · i14 **{113:3306, 4:25}** · i16 {5:3331} · i18 {1:3331} · i20 {1:3331} · i24 {14:3331} · i25 {8:3331} · i26 {10:3331} · i27 {8:3331} · i30 {1:2569} — contre i15 **531 largeurs distinctes (max 1 347)**, i7 169 (max 713), i9 128 (**max 16 777 294**), i6 96 (max 826), i17 96 (max 190), i28 101 (max 276) |
| 2026-09-15 | 1.9.1 bis (pas 1, CONTRÔLE [3] — la population) | ce commit | même commande, table [3] | **AVEC `object-position-component` : 184 fermés / 21 698 (0,85 %)** — ti=37 3/3 331, ti=38 180/12 064, ti=41 0/110, ti=42 1/2 087, ti=43 0/4 106. **SANS : 12 654 / 28 573 (44,29 %).** Le défaut n'est pas propre à l'équipement : il est dans le préfixe « objet du monde » |
| 2026-09-15 | 1.9.1 bis (pas 1, CONTRÔLE [4] — le registre) | ce commit | même commande, table [4] | Les 31 composants, dans le MÊME ordre, sur les 7 bobines — **sauf `a521164d` (HI_1_4_1) qui en porte 30** : `i30 equipment-has-infinite-uses` n'existe pas sur ce build (et la table [2] le confirme : i30 vu 2 569 fois, soit 3 331 − 762) |
| 2026-09-15 | 1.9.1 bis (pas 1, CONTRÔLE — les largeurs de la carte) | ce commit | `CGO_ENABLED=0 go test …/filmdec/ -run '^TestE191bFermetureAvecCarte$' -v -count=1` (12,8 s, catalogue versionné `map_quant_bounds.json`) | **3/3 331 au défaut Cliffhanger, 3/3 331 aux largeurs de la carte jouée** : `a521164d` Fragmentation Heavies 17/17/15 r1@0 1/762 -> 1/762 · `60ae07c4` Live Fire 12/12/11 **r2@1** 0/425 -> 1/425 · `11de8353` Thunderhead 15/15/17 1/514 -> **0/514** · `111fa685` Command 15/15/17 1/569 -> **0/569** · `e5adf7b2` Fragmentation 17/17/15 0/489 -> 1/489 · `bcb6d393` Cliffhanger 13/13/14 0/352 -> 0/352 · `fb1a1a72` Banished Narrows 15/15/17 0/220 -> 0/220. **i0 est hors de cause pour la fermeture** ; la dette de mesure reste (D2 (1.9.1 bis)) |
| 2026-09-15 | 1.9.1 bis (pas 1, MESURE — les masques) | ce commit | `CGO_ENABLED=0 CHUNK00_FILMS='…/a521164d;…/60ae07c4;…/11de8353;…/111fa685;…/e5adf7b2;…/bcb6d393;…/fb1a1a72' go test …/filmdec/ -run '^TestE191bMasqueTI37$' -v -count=1` (7,6 s, films ENTIERS du cache par les jonctions du worktree) | **Les 7 bobines versionnées rendent 0 paquet delta** (mesuré avant, même instrument) : aucune mesure de masque n'y est possible. Sur les films entiers (12 chunks) : 7 142 à 14 358 paquets delta, 10 718 à 40 672 records tous archétypes, **92 records NEW et 70 records DELTA de ti=37**, présence au masque **plate de 10,9 % à 34,3 % sur les 31 index** (NEW min i11/i13/i20/i28 10,9 %, max i26 27,2 % ; DELTA min i11 8,6 %, max i22 34,3 %) — signature de bits aléatoires, pas d'un masque. La question « où le jeu écrit i10/i11/i18/i20/i21/i23 » reste SANS RÉPONSE tant que ti=37 ne ferme pas (D4 (1.9.1 bis)) |
| 2026-09-15 | 1.9.1 bis (pas 1, corpus gate) | ce commit | `go run ./cmd/replay-corpus-gate --base=deba3261f --parc-root C:/…/LevelUp-go-migration --source-root C:/…/LevelUp-wt-decfilm-191b` — **le `--manifest` du brief est à corriger : le drapeau EXIGE une valeur** (`flag needs an argument: -manifest`, sortie 2) et son défaut `<source-root>/config/replay_corpus.toml` est exactement ce qu'on veut ; première tentative perdue là-dessus, rejouée en entier | **14 témoins, 0 gain, 0 perte, schéma 59 -> 59 partout, statut `ok` sur les 14** (18 min 46 s) : `bcb6d393` 11,5 s · `fb1a1a72` 30,0 s · `d9781168` 23,8 s · `c75f33b8` 14,2 s · `bf15f7ab` 13,5 s · `51ebbc0f` 17,7 s · `084a804d` 1 min 45,8 s · `0797ce72` 12,4 s · `111fa685` 33,1 s · `e5adf7b2` 37,6 s · `60ae07c4` 24,6 s · `a349fea8` 2 min 4,7 s · `bfecd02b` 18,9 s · `4f77afc1` 1 min 32,9 s. Résultat ATTENDU et non un non-événement : il PROUVE que le pas n'a touché aucun octet cuit, donc que la mesure est bien une mesure |
| 2026-09-15 | 1.9.1 bis (pas 2) | ce commit | `gofmt -l ./internal ./cmd` | sortie vide |
| 2026-09-15 | 1.9.1 bis (pas 2) | ce commit | `CGO_ENABLED=0 go vet` sur `./internal/games/halo_infinite/film/filmdec/ ./internal/archlint/ ./internal/replaybuild/ ./internal/analysis/objectiveevents/ ./internal/domain/replaydoc/ ./internal/games/halo_infinite/film/replay/` | 0 diagnostic |
| 2026-09-15 | 1.9.1 bis (pas 2) | ce commit | `CGO_ENABLED=0 go test ./internal/games/halo_infinite/film/... ./internal/archlint/ ./internal/replaybuild/ ./internal/analysis/objectiveevents/ ./internal/domain/replaydoc/ -count=1` | **tout vert** — filmdec 84,0 s · replay 14,8 s · archlint 21,4 s · replaybuild 1,1 s · objectiveevents 0,6 s · killsource 1,1 s ; `replaydoc` « no test files ». **Aucun test renommé ni supprimé** : `.ai/baselines/tests_pre_migration.jsonl` intouché, et c'est correct (six tests NEUFS seulement) |
| 2026-09-15 | 1.9.1 bis (pas 2) | ce commit | `make -C . go-api-lint` (`GOLANGCI_LINT_CACHE` isolé) | **0 issue**, baseline non accrue. Fichiers neufs : 406 / 220 / 186 / 165 lignes — sous le seuil de 500 |
| 2026-09-15 | 1.9.1 bis (pas 2) | ce commit | `go test ./…/filmdec/ -run 'TestG4\|TestG5\|ECSTable'` après chaque écriture de `ecs_table.tsv` | vert. **PIÈGE RENCONTRÉ ET CONSIGNÉ** : `bits_typ` n'est lu comme largeur VÉRIFIÉE que s'il porte un ENTIER NU ; y écrire une plage (« 7 a 826 ») sort la ligne du contrôle G4, et remplir la colonne sur des lignes jusque-là VIDES y fait ENTRER des dizaines de lignes d'un coup (114/65 -> 122/80 au premier essai, puis 113/62). La règle de la table : `bits_typ` = le cas nominal à largeur fixe, la plage va dans `grammar`. Le remplissage final ne touche `bits_typ` que sur des lignes `inconnu` de ti=37 |
| 2026-09-15 | 1.9.1 bis (pas 2, mutation) | ce commit | `consumeObjectRegionState` : `br.ReadBits(6)` -> `br.ReadBits(5)`, puis `go test -run TestConsumeObjectRegionStateLargeurs` | **ROUGE sur les 5 lignes** (6 bits au lieu de 7, 409 au lieu de 826) ; remis à 6 : vert. Le garde-rail mord |
| 2026-09-15 | 1.9.1 bis (pas 2, MESURE — la fermeture des cinq) | ce commit | `CGO_ENABLED=0 go test ./…/filmdec/ -run '^TestE191cPrefixeObjet$' -v -count=1` (2,7 s puis 7,6 s avec les tableaux [C] et [D], 7 bobines versionnées, aucune garde d'environnement) | **AVANT = APRÈS, 184 fermés / 21 698 bornés (0,85 %)** : ti=37 3/3 331 · ti=38 180/12 064 · ti=41 0/110 · ti=42 1/2 087 · ti=43 0/4 106. Dichotomie 1 (préfixe objet) AVEC 184/21 698 (0,85 %) / SANS 12 654/28 573 (44,29 %) ; dichotomie 2 (moins de 12 composants) SIMPLE 7 547/11 293 (66,83 %) / COMPLEXE 5 291/38 978 (13,57 %) — **aucune des deux n'est la variable explicative**. Tableau [C] : **0 composant de ti=37 blanchi sur 30** |
| 2026-09-15 | 1.9.1 bis (pas 2, CONTRÔLE — les ancres fortuites) | ce commit | `CGO_ENABLED=0 go test ./…/filmdec/ -run '^TestE191cAncresFortuites$' -v -count=1` (5,0 s) | **138 ancres retirées sur 50 384 records (0,27 %)** ; fermeture des cinq **184/21 594**, soit 0,85 % — INCHANGÉE. ti=37 3/3 293 · ti=38 180/12 056 · ti=41 0/79 · ti=42 1/2 060 · ti=43 0/4 106 |
| 2026-09-15 | 1.9.1 bis (pas 2, CONTRÔLE — l'état par défaut) | ce commit | `CGO_ENABLED=0 go test ./…/filmdec/ -run '^TestE191cEtatParDefaut$' -v -count=1` (21,7 s, balayage 0..512 bits) | **ti=37 : le désérialiseur porté ferme 3/3 331 et consomme 115 bits dans 952 records, 92 dans 348, 60 dans 319, 110 dans 270, 147 dans 205 ; la MEILLEURE largeur substituée ferme 32 records (w=1), puis 31 (w=0), 23 (w=27)** — aucune largeur d'état par défaut ne ferme ti=37. ti=38 : porté 180, substitué **w=173 -> 901**, w=157 -> 565, w=144 -> 564 (hors périmètre, consigné D4). ti=41 : porté 0/110, aucun état par défaut dans la table. ti=43 : **« (aucune) »** — les 4 106 records désynchronisent tous (D5) |
| 2026-09-15 | 1.9.1 bis (pas 2, GrammarRev) | ce commit | `go test ./…/filmdec/ -run GrammarRevSuitLaGrammaire -update-grammar-rev` puis sans le drapeau | `grammar-2026-09-15.1` -> **`grammar-2026-09-15.2`**, empreinte `c05826e4…` ; vert après. Historique du golden complété (le pourquoi : commentaires et garde-rails, **aucun bit lu ne change**) |
| 2026-09-15 | 1.9.1 bis (pas 2, GATES DE DÉCODAGE) | — | `replay-equiv` et `replay-corpus-gate` | **NON JOUÉS — machine partagée avec le lot 1.9.2, « voie libre » non donnée par le pilote.** Le pas ne touche AUCUN octet cuit : `SchemaVersion` 59 -> 59, et côté production seuls des commentaires changent (`components_object_state.go`, `components_batch7.go`, `unit_weaponstate.go`) plus `grammar_rev.go` et deux fichiers de `testdata/`. Le reste est en `_test.go` |
| 2026-09-15 | 1.9.1 bis (pas 2 bis) | ce commit | `gofmt -l ./internal ./cmd` | sortie vide |
| 2026-09-15 | 1.9.1 bis (pas 2 bis) | ce commit | `CGO_ENABLED=0 go vet ./internal/games/halo_infinite/film/... ./internal/archlint/ ./internal/replaybuild/ ./internal/analysis/objectiveevents/` | 0 diagnostic |
| 2026-09-15 | 1.9.1 bis (pas 2 bis) | ce commit | `make -C . go-api-lint` (`GOLANGCI_LINT_CACHE` isolé) | **0 issue, sortie 0**, baseline non accrue. `varwidth.go` 92 lignes, `e191c_suspects_research_test.go` 126 — sous le seuil. **PIÈGE RENCONTRÉ** : `make … \| tail -3` affichait « 0 issues. » PUIS « Erreur 7 » — c'est le masquage d'exit par le tube (mémoire du dépôt) ; relancé sans tube, `exit=0` |
| 2026-09-15 | 1.9.1 bis (pas 2 bis) | ce commit | `CGO_ENABLED=0 go test ./internal/games/halo_infinite/film/... ./internal/archlint/ ./internal/replaybuild/ ./internal/analysis/objectiveevents/ -count=1` | **vert PARTOUT SAUF `TestKeyframeClosureRatchet`**, qui est le point d'arbitrage décrit ci-dessous. Aucun test renommé ni supprimé : `.ai/baselines/tests_pre_migration.jsonl` intouché (deux tests NEUFS) |
| 2026-09-15 | 1.9.1 bis (pas 2 bis, MESURE — la fermeture) | ce commit | `CGO_ENABLED=0 go test …/filmdec/ -run '^TestE191cPrefixeObjet$' -v -count=1` | **184 -> 188 fermés / 21 698 bornés (0,85 % -> 0,87 %)** : ti=37 **3 -> 4**, ti=38 **180 -> 183**, ti=41 0/110, ti=42 1/2 087, ti=43 0/4 106 inchangés. Par bobine, ti=37 : `a521164d` 1 -> **2** · `60ae07c4` 0 -> **1** · `11de8353` 1 -> **0** · `111fa685` 1 -> **0** · `fb1a1a72` 0 -> **1** ; ti=38 : `60ae07c4` 43 -> **30** · `11de8353` 32 -> **48** |
| 2026-09-15 | 1.9.1 bis (pas 2 bis, ATTRIBUTION) | ce commit | chaque correction NEUTRALISÉE à tour de rôle, même instrument | **i10 porte tout le gain de ti=38** : 173 sans elle, 183 avec. Le record supplémentaire de ti=37 **disparaît dès qu'on retire i21, OU i22, OU i28** (3/3 331 dans les trois cas, 4/3 331 avec les trois) — un record ne ferme que si TOUS ses composants sont justes, et c'est la meilleure preuve disponible que les quatre lectures sont bonnes |
| 2026-09-15 | 1.9.1 bis (pas 2 bis, la carte du défaut) | ce commit | colonne « bloquant » du golden 0.A.3 relue, + `TestE191cListeCourte` et `TestE191cPairesDiscriminantes` | **17 archétypes de classe (A)** (un composant sans lecteur, nommé) contre **6 de classe (B)** (aucune désynchronisation, aucune fermeture) : ti=13, 21, 37, 38, 41, 42. Les six portent **51 composants distincts, 0 blanchi**, et **aucun composant commun** |
| 2026-09-15 | 1.9.1 bis (pas 2 bis, GrammarRev) | ce commit | `go test …/filmdec/ -run GrammarRevSuitLaGrammaire -update-grammar-rev` puis sans le drapeau | `grammar-2026-09-15.2` -> **`.3`**, empreinte `2621e7f7…`. **LE RANG MONTE MALGRÉ LE MÊME LOT**, et c'est écrit dans l'historique du golden : `.2` a été publiée comme « aucun bit lu ne change », lui accrocher une correction de largeur en ferait une doc inversée |
| 2026-09-15 | 1.9.1 bis (pas 2 bis, POINT D'ARBITRAGE) | — | `TestKeyframeClosureRatchet` | **ROUGE, ET DÉLIBÉRÉMENT LAISSÉ ROUGE.** 4 lignes MONTENT, 3 DESCENDENT, net +4. D14 dit que le ratchet des sept ne descend que ; la règle 9 du contrat d'exécution dit qu'un blocage qui appelle une décision se STATUE et se remonte, il ne se contourne pas. Analyse écrite : un record dont le compte de bits ne change pas ne peut pas basculer, donc chaque record qui cesse de fermer fermait sur un total que l'écrivain contredit — **DIVERGENCE de bord sur une marche à 0,1 %, pas RÉGRESSION de grammaire**. Le geste, si le pilote ratifie : `go test ./internal/games/halo_infinite/film/filmdec/ -run KeyframeClosureRatchet -update-keyframe-closure` |
| 2026-09-15 | 1.9.1 bis (pas 2 bis, GATES DE DÉCODAGE) | — | `replay-equiv`, `replay-corpus-gate` | **NON JOUÉS — « voie libre » non donnée (1.9.2 tourne).** Cette fois ce n'est PAS neutre : des BITS changent, donc le contenu cuit PEUT changer et **`SchemaVersion` reste à 59 en attendant que le corpus gate le dise**. C'est la première chose à jouer à la reprise |
| 2026-09-15 | 1.9.1 bis (pas 2 ter + quater) | ce commit | `gofmt -l ./internal ./cmd` | sortie vide |
| 2026-09-15 | 1.9.1 bis (pas 2 ter + quater) | ce commit | `CGO_ENABLED=0 go test ./internal/games/halo_infinite/film/... ./internal/archlint/ ./internal/replaybuild/ ./internal/analysis/objectiveevents/ -count=1` | **tout vert** (filmdec 83-87 s). Une seule fixture mise a jour : `TestScanPlayerTeamsSurLesBobines`, dont le `horsIdx: 1` de `111fa685` passe a 0 et gagne un champ `nonAtteint: 1` — le record en question a `n2 == 0`, donc AUCUN composant, et l'index hors table qu'on y lisait etait du bruit. Aucun test renomme ni supprime : `.ai/baselines/tests_pre_migration.jsonl` intouche |
| 2026-09-15 | 1.9.1 bis (pas 2 ter + quater) | ce commit | `make -C . go-api-lint` (`GOLANGCI_LINT_CACHE` isole) | **0 issue, sortie 0** ; baseline non accrue |
| 2026-09-15 | 1.9.1 bis (pas 2 quater, MESURE — les gardes) | ce commit | `go test …/filmdec/ -run '^TestE191cPrefixeObjet$' -v -count=1`, puis chaque garde NEUTRALISEE a tour de role | **246 fermes / 21 698 (1,13 %) contre 188 avant.** ti=37 4 -> **15** · ti=38 183 -> **187** · ti=41 0 -> **34/110 (30,9 %)** · ti=42 1 -> **10** · ti=43 0. Attribution : garde `n1` SEULE 188 (aucun effet sur les cinq), garde `n2` SEULE 220, **les deux 246** — elles interagissent |
| 2026-09-15 | 1.9.1 bis (pas 2 quater, ratchet) | ce commit | `go test …/filmdec/ -run '^TestKeyframeClosureRatchet$'`, deltas totalises | **26 hausses (+89 records), 2 baisses (-21), NET +68.** Golden 0.A.3 REGENERE, avec l'historique qui porte les deux descentes record par record |
| 2026-09-15 | 1.9.1 bis (pas 2 quater, L'ORACLE) | ce commit | `go test …/filmdec/ -run '^TestE191cPremierBitFaux$' -v -count=1` | **`n1` CONSTANT sur tous les archetypes mesures** (ti=14 4 · ti=17 4 · ti=22 12 · ti=29 1 · ti=6 4 · ti=13 136 · ti=37 100 · ti=38 100 · ti=42 164) — l'en-tete de 108 bits est donc exact PARTOUT. **`n2` constant chez ceux qui ferment** (28, 432, 12, 256, 7896) et **du BRUIT chez ceux qui echouent** (ti=13 : -1 ×48, 2147483392 ×24, 32768, 98304, 0, 68… ; ti=37 : -1073741824 ×222… ; ti=38 : 0, 110, 14112, 3528, 3612672…). Le premier bit faux est DANS L'ETAT PAR DEFAUT |
| 2026-09-15 | 1.9.1 bis (pas 2 quater, LE BALAYAGE) | ce commit | `go test …/filmdec/ -run '^TestE191cOracleN2$' -v -count=1` (16 × 16, ancres precalculees, 1,0 s) | **La meme paire gagne sur les trois archetypes.** ti=37 0,204 -> **0,639** · ti=38 0,053 -> **0,617** · ti=42 0,059 -> **0,732**, toutes a `lead=8 index=3`, `n2` modal 1 396 / 1 764 / 1 300 au lieu de 0. **Non porte** : 8 contredit le litteral `R(9)` (`141fd72de`) et 3 le `R(5)` inline — la lecture juste est « le bloc MPP consomme TROIS BITS DE TROP », pas « poser 8 et 3 » |
| 2026-09-15 | 1.9.1 bis (pas 2 ter + quater, GATES DE DECODAGE) | — | `replay-equiv`, `replay-corpus-gate` | **NON JOUES : la voie de decodage a ete TRANSFEREE au lot 1.9.3 par le pilote.** Le `replay-corpus-gate` du pas 2 bis, joue AVANT le transfert, reste valide pour l'etat d'alors ; les gardes `n1`/`n2` de ce commit n'ont PAS ete passees au corpus gate. **EN ATTENTE DE VOIE LIBRE**, et c'est la premiere chose a jouer a la reprise, avec la conversion MPP (D9) |
| 2026-09-15 | 1.9.1 ter (équivalence, régime court — CLASSIFICATION AVANT TOUT `-update`) | ce commit | `go run ./cmd/replay-equiv -repo-root C:/…/LevelUp-wt-decfilm-191t -films 50247b26,a521164d,60ae07c4,11de8353,111fa685,e5adf7b2,bcb6d393,51101d1d,d9781168,fb1a1a72`, sortie COMPLÈTE capturée (la première passe était tronquée par un `tail`, elle ne montrait qu'un film — rejouée entière plutôt que d'extrapoler) | **BILAN : 0 identique(s), 10 different(s), 0 ecarte(s), 0 echec(s)** — et **UNE SEULE étape sur les 53, `placements.stats`, sur les DIX films** (`50247b26`, `a521164d`, `60ae07c4`, `11de8353`, `111fa685`, `e5adf7b2`, `bcb6d393`, `51101d1d`, `d9781168`, `fb1a1a72` — un écart chacun, jamais deux). **Ni `placements`, ni `artifact`, ni `killsource`, ni aucune étape d'image-clé ne diverge.** Durées : 1m51 · 41 s · 35 s · 1m05 · 51 s · 46 s · 17 s · 6,6 s · 29 s · 34 s ; pic 0,09 à 0,34 Gio |
| 2026-09-15 | 1.9.1 ter (équivalence, CAUSE établie avant le geste) | ce commit | lecture du code, pas une inférence sur les sha : `replay/film_scan.go:329` + `replay/equipment_placements.go` (`buildEquipmentPlacements`) | `observe("placements.stats", s.in.PlacementStats)` hache **la struct entière** ; le lot y a ajouté `FormatVersion` et `FormatSansProfil`. **Les deux champs n'atteignent PAS le document cuit** : `EquipmentPlacementCoverage` est construit à partir de champs NOMMÉS (`Scanned`, `Calibration`, `Lives`, `Anchors`, `Confirmed`, `ByFamily`…), jamais par recopie. Contrôle sur `fb1a1a72` : `poses=341`, `bytes=3016354` au journal du run — identiques à la référence (`placements 341`, `artifact 3016354`). **DIVERGENCE DE FORME, pas régression** — motif D4 (1.8), rejoué au lot 1.9.3 sur `killsource` |
| 2026-09-15 | 1.9.1 ter (équivalence, re-figeage sur feu vert du pilote) | ce commit | même commande avec `-update` sur les DIX, puis `git diff` des références | `-update` : **BILAN : 10 identique(s), 0 different(s), 0 ecarte(s), 0 echec(s)**. `git diff --stat` : **10 fichiers, 10 insertions, 10 suppressions**. `git diff -U0` hors en-tête : **10 `+placements.stats` et 10 `-placements.stats`, ET RIEN D'AUTRE** — une seule ligne par fichier, exactement celle que la classification annonçait. C'est la preuve mécanique de la classification, pas sa paraphrase |
| 2026-09-15 | 1.9.1 ter (équivalence, passe de comparaison) | ce commit | même commande SANS `-update`, après le re-figeage | **BILAN : 10 identique(s), 0 different(s), 0 ecarte(s), 0 echec(s), 0 illisible(s)** — **10/10**, et **zéro ligne `ECART` dans toute la sortie**. Durées : 1m45 · 46 s · 29 s · 1m34 · 43 s · 1m16 · 22 s · 6,2 s · 29 s · 36 s ; pic 0,08 à 0,34 Gio. `SchemaVersion` reste **59** : le corpus gate n'est pas requis ici, aucun octet cuit ne changeant — l'étape `artifact` est identique à sa référence sur les dix films AVANT comme APRÈS le re-figeage (seule `placements.stats` a bougé) |
| 2026-09-16 | 1.9.1 bis (clôture, GATE 1/2) | ce commit | `go run ./cmd/replay-equiv -films <régime court>` en DEUX sous-ensembles, **passe de classification AVANT tout `-update`** | **7 différents sur 10, et UNE SEULE étape sur les 53 : `killsource`.** Différents : `60ae07c4`, `111fa685`, `e5adf7b2`, `bcb6d393`, `51101d1d`, `d9781168`, `fb1a1a72`. **Identiques : `50247b26`, `a521164d`, `11de8353`** — ils ne portent aucun record à `n2 == 0` sur le chemin de `killsource`, ce qui borne la correction à ce qu'elle doit toucher. Aucune étape d'image-clé, de positions ni d'`artifact` ne diverge |
| 2026-09-16 | 1.9.1 bis (clôture, GATE 1/2 — re-figeage) | ce commit | `-update` sur les deux sous-ensembles, puis passe de comparaison | **10/10 identiques.** Sept `.tsv` modifiés, soit exactement les sept films classés. Bloc de re-figeage ajouté à `CORPUS.txt` (commit `76d36b911`, schéma 59, les deux corrections relues chez l'écrivain, la mesure) |
| 2026-09-16 | 1.9.1 bis (clôture, GATE 2/2) | ce commit | `go run ./cmd/replay-corpus-gate --base=462480085 --json …` (14 témoins, ~13 min) | **schéma 59 -> 59 sur les 14**, **`chang.` = 0 sur les 14**, **31 gains** (`c75f33b8` +7, `a349fea8` +12, `4f77afc1` +12), **3 « pertes » qui sont TOUTES des compteurs de défauts en baisse** : `coverage.equipmentChanges.missedEstimate` 17 -> 16 (`a349fea8`), `coverage.teams.divergences` 4 -> 2 et `coverage.teams.unread` 4 -> 3 (`4f77afc1`). **Classées DIVERGENCE** : les gardes `n1`/`n2` suppriment des lectures de bruit (`n2 == 0` -> le jeu n'écrit AUCUN composant), donc moins de manques estimés, moins de divergences d'équipe, moins de non-lus — la même chose que `TestScanPlayerTeamsSurLesBobines` perdant son « 1 index hors table ». Aucune régression |
| 2026-09-16 | 1.9.1 bis (clôture, l'oracle réparé) | ce commit | `go test ./cmd/replay-corpus-gate/` + mutation | **Le gate JETAIT `BilanAxe.Changements`** (D25) — corrigé, et `changements_guard_test.go` le tient : mutation `changements += b.Changements` retirée -> `--- FAIL: TestBilanPorteLesChangements`, restaurée par nom -> vert. Le premier run du gate (avant correctif) et le second (après) rendent le MÊME verdict, ce qui vaut contrôle |
| 2026-09-16 | 1.9.1 bis (clôture, SchemaVersion) | ce commit | verdict des deux gates | **`SchemaVersion` RESTE 59.** Schéma inchangé sur les 14 témoins ET zéro changement de valeur publiée : aucun octet cuit ne change de forme, donc **pas de chaîne `openapi-gen` / `generate-types`** |
| 2026-09-16 | 1.9.1 bis (clôture, budget CI) | ce commit | `CGO_ENABLED=0 go test ./internal/games/halo_infinite/film/filmdec/ -count=1` | **87,6 s.** Le budget porté à 300 s par le pilote (`a02caf36e`) tient, mais la marge n'est plus que de **3,4×** : ce lot ajoute sept instruments de recherche qui rebalayent chacun les sept bobines. Un lot de plus au même rythme la ramènerait sous 2× |
| 2026-09-15 | 1.9.1 bis (pas 2 quinquies) | ce commit | `go test …/filmdec/ -run '^TestE191cOracleN2ParBuild$' -v -count=1` (2,9 s, sept bobines, ancres precalculees) | **La coupure est a `HI_1_12_0`, et elle est nette sur les trois archetypes.** ti=37 : 8/3 rend 0,995 · 0,988 · 0,990 · 0,993 · 0,996 sur HI_1_4_1 a HI_1_11_0 (9/5 y rend 0,304 a 0,522), et 9/5 rend 0,949 · **1,000** sur HI_1_12_0 et HI_1_13_0. La version MAJEURE ne discrimine pas (`e5adf7b2` et `bcb6d393` sont tous deux `v=40`) ; le cardinal de la table par type, si : **116/121/121/121/122 contre 123/123** |
| 2026-09-15 | 1.9.1 bis (pas 2 quinquies) | ce commit | `gofmt -l ./internal` ; suite complete `go test` filmdec | sortie vide ; **vert** (105 s). Aucun code de production touche : le pas est une MESURE et deux instruments |
| 2026-09-15 | 1.9.1 bis (pas 3) | ce commit | `go test …/filmdec/ -run '^TestE191cFermetureMPP$' -v -count=1` (3,0 s, 7 bobines × 12 decoupages) | **La fermeture ne designe aucun decoupage coherent, et elle se disqualifie.** `a521164d` 9/5=39 · `60ae07c4` 7/4=95 (9/5=35) · `11de8353` 9/5=50 · `111fa685` 10/4=58 (9/5=31) · `e5adf7b2` 7/4=28 (9/5=13) · `bcb6d393` 8/3=69 (9/5=33) · **`fb1a1a72` 10/5=140 contre 9/5=11** — sur le build de l'executable, ou l'ecrivain ecrit `R(9)` |
| 2026-09-15 | 1.9.1 bis (pas 3) | ce commit | `go test …/filmdec/ -run '^TestE191cPrefixeObjet$'` avec le profil installe | **246 / 21 698 (1,13 %) — INCHANGE.** Les deux builds dont la largeur MPP est posee la portent a `9/5`, deja le defaut ; les cinq autres n'installent rien. Un premier essai qui posait `8/3` rendait **182** (-64) : ecarte, D14 |
| 2026-09-15 | 1.9.1 bis (pas 3) | ce commit | `go test …/filmdec/ -run 'BuildProfile' -v -count=1` | **3 tests verts** : table figee en clair, `ErrUnknownBuild` sur quatre builds absents (dont la chaine vide), et **la mutation qui rougit** — a `±1` sur `lead` comme sur `index`, `n2` cesse d'etre constant sur `HI_1_13_0` |
| 2026-09-15 | 1.9.1 bis (pas 3) | ce commit | `make -C . go-api-lint` | **0 issue.** `goconst` a mordu sur les sept noms de build (5 occurrences chacun) : sept constantes nommees, cote PRODUCTION seulement — les tests gardent leurs litteraux, un test qui relit la constante qu'il verifie ne verifie rien. Un garde-rail a bouge : `TestWorldObjectPrecisionReadersAreAllowlisted` — `build_profile.go` CITE `replay.installWorldObjectPrecision` en commentaire (le contrat du verrou) sans lire le global ; entree d'allowlist avec sa raison |
| 2026-09-15 | 1.9.1 bis (pas 3) | ce commit | suite complete `go test` film + archlint + replaybuild + objectiveevents | **tout vert** (filmdec 114 s). Aucun test renomme ni supprime : `.ai/baselines/tests_pre_migration.jsonl` intouche (trois tests NEUFS) |
| 2026-09-15 | 1.9.1 bis (pas 3, GrammarRev) | ce commit | `-update-grammar-rev` puis sans | `.4` -> **`.5`**. **Aucun bit lu ne change sur les sept bobines** ; la revision monte parce que le CADRE change — une grammaire peut desormais dependre du build, et un consommateur qui decide de redecoder doit le voir |
| 2026-09-15 | Revue M1 — L6 (constat 1, mutation AVANT) | ce commit | `killsource/feed_couples.go:146`, `v == victime` remplace par `v != "" && victime != ""` (moitie VICTIME du predicat supprimee), puis `go test ./internal/games/halo_infinite/film/killsource/ -count=1` | **`ok levelup/go-api/internal/games/halo_infinite/film/killsource 1.570s`** — le trou est confirme : aucun temoin ne passe deux `killEventRec` a la fois |
| 2026-09-15 | Revue M1 — L6 (constat 1, mutation APRES) | ce commit | meme mutation, apres l ajout de `feed_couples_victime_test.go` | **ROUGE** : `couple lu = {timeMS:2000 killer:C victim:E victimXUID:55}, attendu (C, D) a t=2000` ; `le couple ecrit a consomme rec[0] (C -> D), qui est celui du kill ORPHELIN` ; `le couple ecrit (C, E) n a PAS consomme rec[1]`. Restauree par nom, paquet `ok` |
| 2026-09-15 | Revue M1 — L6 (constat 2, mutation AVANT) | ce commit | `filmdec/player_teams.go:241`, `continue` retire apres `rep.IndexDivergences++`, puis `go test …/filmdec/ -count=1` | **Un seul rouge : `TestGrammarRevSuitLaGrammaire`** — c est l EMPREINTE DES SOURCES, pas un test de comportement. Les sept bobines rendent 0 divergence, la branche n y est jamais prise |
| 2026-09-15 | Revue M1 — L6 (constat 2, mutation APRES) | ce commit | meme mutation, apres `player_teams_divergence_test.go`, `-count=5` (garde anti-intermittence : `out[idx]` depend de l ordre d iteration d une map) | **ROUGE aux 5 runs, a l identique** : `l index 3 est publie (equipe 1)…` ; `Indices = 2, attendu 1` ; `table = map[3:1], attendue nil`. Les assertions portent l ABSENCE de l index, jamais sa valeur. Restauree par nom |
| 2026-09-15 | Revue M1 — L6 (constat 3, mutation AVANT) | ce commit | `sync/killcollector/map_identity.go:94`, boucle de `entreeDeCatalogueParNom` parcourue A L ENVERS, puis `go test ./internal/sync/killcollector/ ./internal/games/halo_infinite/film/... ./internal/archlint/ -count=1` | **TOUT VERT sauf les deux goldens deja rouges par construction** (`TestDocumentShapeMatchesGolden`, `TestContractFixturesMatchCommitted`). L empreinte `KillSourceDecoderRev` ne hache que `film/killsource`, jamais `killcollector` |
| 2026-09-15 | Revue M1 — L6 (constat 3, mutation APRES) | ce commit | meme mutation, apres `carte_ordre_des_candidats_test.go` | **ROUGE sur les DEUX sens** : `TestLePremierNomQuiResoutGagne/temoin_d'abord` et `/jumelle_d'abord`, plus `positions […] et touches […], attendues toutes deux celles de "Behemoth"`. Restauree par nom |
| 2026-09-15 | Revue M1 — L6 (constat 3, MESURE du critere de retrait) | ce commit | `go run ./cmd/diag_q` sur une COPIE de `metadata.duckdb` (scratchpad, aucune ouverture de la base tenue) avec `ATTACH data/backups/pre-chaine-2026-09-09/shared_matches_v2.duckdb AS parc (READ_ONLY)` ; cascade de langues de `PreferredLangsForLocale("en")` rejouee en SQL | **1 967 matchs, 1 933 a DEUX candidats, 7 a deux candidats DISTINCTS, et `0` ou les deux resolvent des entrees DIFFERENTES du catalogue.** Les 7 seconds candidats sont des UUID bruts de `match_registry.map_name` ; un seul des deux resout a chaque fois. Cas remarquable : sur `944396dd-…` (Narrows) c est le SECOND qui resout — le catalogue versionne ne porte pas la cle `narrows`, mais bien l UUID. Le critere de retrait du registre est donc TENU sur le parc oracle ; le registre n est PAS modifie (un autre lot le tient) |
| 2026-09-15 | Revue M1 — L6 (constat 5, mutation AVANT) | ce commit | `replay/player_teams.go:160`, les deux voies de `poserSurLesTraces` PERMUTEES (pont du slot d abord), puis `go test ./internal/games/halo_infinite/film/replay/... ./internal/replaybuild/ ./internal/service/replayview/ -count=1` | **Seuls les deux goldens deja rouges par construction.** Sur les bobines versionnees les deux voies s accordent partout : un ordre ne se teste que la ou les sources DIVERGENT |
| 2026-09-15 | Revue M1 — L6 (constat 5, mutation APRES) | ce commit | meme mutation, apres `player_teams_ordre_test.go` | **ROUGE** : `equipe posee = 1, attendue 0 (celle du XUID 111)`. Les deux autres temoins (vie sans xuid nommee par le pont, vie que personne ne nomme) restent verts — l ordre est ce qui bascule, pas le pont. Restauree par nom |
| 2026-09-15 | Revue M1 — L6 (constat 6, porte `-update`) | ce commit | `go test ./internal/sync/killcollector/ -run '^TestKillSourceDecoderRevSuitLeDecodeur$' -count=1` puis la meme avec `-update` | **run normal `ok 0.135s` ; porte `-update` ROUGE** (`1 reference(s) reecrite(s) : testdata/killsource_decoder_rev.golden (revision killsource-2026-09-15, empreinte ed7e45ca…)`), et `git status` du `testdata/` **VIDE** : la reecriture est a l octet pres. La porte s aligne sur `-update-keyframe-closure` et `-update-grammar-rev` (revue R2, C1) |
| 2026-09-15 | Revue M1 — L6 (gate) | ce commit | `gofmt -l ./internal ./cmd` | sortie vide |
| 2026-09-15 | Revue M1 — L6 (gate) | ce commit | `go vet ./internal/games/halo_infinite/film/... ./internal/archlint/ ./internal/replaybuild/ ./internal/sync/killcollector/ ./internal/analysis/objectiveevents/ ./internal/domain/replaydoc/ ./internal/service/replayview/` | 0 diagnostic |
| 2026-09-15 | Revue M1 — L6 (gate) | ce commit | `go test` sur les memes paquets, `-count=1` | **verts, aux DEUX goldens rouges par construction pres** : filmdec 24,77 s · killsource 2,31 s · replay 25,51 s (`TestDocumentShapeMatchesGolden`, `TestContractFixturesMatchCommitted`) · archlint 20,90 s · replaybuild 1,92 s · killcollector 0,28 s · objectiveevents 0,94 s · replayview 0,45 s |
| 2026-09-15 | Revue M1 — L6 (gate) | ce commit | `go test -tags=integration -p 1 ./internal/sync/killcollector/ -count=1` (`internal/sync/` touche) | **`ok 26.423s`** |
| 2026-09-15 | Revue M1 — L6 (gate) | ce commit | `CGO_ENABLED=0 go test ./internal/games/halo_infinite/film/filmdec/ -count=1` (budget du paquet sans tag) | **`ok 22.659s`** — tres en dessous des 300 s |

LA TABLE DU REGISTRE DES REPLIS (lot 1.9.0, 95 entrees, collee depuis le registre : un
`go run` jetable sur `fallback.Table()`, supprime apres la mesure). Les declenchements sont
ceux du REGIME COURT (cuisson de production sur les 10 films de l'echantillon) ; « — (non
cable) » dit que le compteur n'existe pas encore, ce qui n'est PAS un zero (cf.
`Repli.CompteurBranche`).

| repli | site(s) | condition / ordre | compteur | déclenchements (régime court, 10 films) | cible de retrait |
|---|---|---|---|---|---|
| 2026-09-15 | 1.9.1 (mesure AVANT de coder) | `3718228c9` + le lecteur 103 seul (aucune décision changée) | `CGO_ENABLED=0 E191_ROOT=<parc>/data/cache/film_chunks go test …/replay/ -run '^TestE191OrigineMesure$' -v` — instrument VERSIONNÉ `e191_origine_{mesure,rapport}_research_test.go`, 13 films (8 builds + échantillon court + `0797ce72` + `4f77afc1`), lecture seule | **345,9 s. 4 583 poses.** Balayage 103 par film, COLLÉ : `000d5950` 46 év. / 3 508 listes · `a521164d` **0 / 4 956** · `60ae07c4` **0 / 6 056** · `11de8353` 62 / 7 325 · `111fa685` 86 / 7 951 · `e5adf7b2` 60 / 7 751 · `bcb6d393` 1 / 2 730 · `fb1a1a72` **0 / 7 850** · `50247b26` **2 / 7 779** · `51101d1d` **0 / 1 599** · `d9781168` 3 / 8 737 · `0797ce72` 51 / 3 949 · `4f77afc1` 30 / 17 441. **`ref2` posée 0 fois sur 341 événements.** DÉSIGNATION par objet : `wall/0x528fce46` **47 poses, 47 désignées**, écart +32,2 à +70,2 ms, **TOUS POSITIFS** ; `wall/0x686b40c9` **77 poses, 68 désignées** (+32,6 à +69,6 ms), **9 sans aucun événement** ; **tous les autres objets : 0 désigné sur 4 459** (grappin 242, grenades 3 290, répulseur 253, capteur 128, écran 110, propulseur 183, champ 78, balise 3, bonus 8, appareils de mur 164) |
| 2026-09-15 | 1.9.1 (mesure, distribution des écarts — c'est elle qui FIXE les tolérances) | idem | table [3] de l'instrument, collée depuis la sortie brute | `origine deployed : 296 poses` — `mort ECRITE du poseur n=296 sans signal=63 min=205.3 p50=19084.7 p90=85392.2 max=249834.3 \| cumul : <=1ms:0 <=10ms:0 <=50ms:0 <=100ms:0 <=200ms:0 <=500ms:9 …` · `prise ECRITE du poseur n=296 sans signal=141 min=0.0 p50=0.0 … \| cumul : <=1ms:103 <=10ms:103 <=50ms:108 <=100ms:111 …`. `origine dropped : 3717 poses` — `mort ECRITE du poseur n=3717 sans signal=461 min=5.6 p50=37.1 p90=40.1 max=171.7 \| cumul : <=1ms:0 <=10ms:7 <=50ms:3161 <=100ms:3238 <=200ms:3256 <=500ms:3256 … <=60000ms:3256` · `prise ECRITE du poseur n=3717 sans signal=2775 min=0.0 … \| cumul : <=1ms:1 <=10ms:1 <=50ms:1 <=100ms:8 …`. **LECTURE** : la mort SÉPARE sans recouvrement (max 171,7 d'un côté, min 205,3 de l'autre — 33,6 ms de vide) ; la prise sépare à 50 ms (108 contre 1), au-delà de 100 ms elle mord les lâchers (8). D'où **mort 200 ms, prise 50 ms, désignation [0, +200] ms** |
| 2026-09-15 | 1.9.1 (verdict, APRÈS la décision utilisateur « lâché » du 2026-09-15) | ce commit | table [5] de l'instrument, 320,5 s, 13 films | **757 poses sur 4 583 BASCULENT (16,5 %)**, et c'est le but : `deployed -> dropped` par un `taken` écrit **108** ; `deployed -> unknown` **188** ; `dropped -> unknown` **461**. **3 826 ne bougent pas** : `death_written` 3 255, `spawn_event` 115, `manifest_piece` 9, `both` 1, `no_owner` 446. Totaux publiés : `deployed` **420 -> 124**, `dropped` **3 717 -> 3 364**, `unknown` **446 -> 1 095**. `byCause` somme à 4 583 exactement — invariant testé (`TestByCauseSommeAuxPlacements`) |
| 2026-09-15 | 1.9.1 (les poses requalifiées par F.1, rejugées une à une) | ce commit | table [6] de l'instrument : les poses dont la règle d'AVANT F.1 (fenêtre + distance) diffère de ce qui était publié | **18 poses, 18 ACCORDS, 0 désaccord.** 17 tranchées par une MORT ÉCRITE (F.1 confirmée pose par pose : `a521164d` ×2 à 98,4 ms / 1,76-1,77 m, `d9781168` ×2 à 62,7 ms, `0797ce72` ×1 à 20,8 ms / 2,69 m, `4f77afc1` ×12 de 44,0 à 171,7 ms / 1,51-2,65 m) ; **1 tranchée par un ÉVÉNEMENT 103** — `111fa685`, panneau `0x686b40c9`, écart 164,3 ms, F.1-avant `dropped` -> publié `deployed` -> **grammaire `deployed` par `spawn_event`** : H.2 confirmée par le film. Le brief en annonçait 22 : c'est le compte de F.1 sur SON corpus (21 films, 5 363 poses), pas sur celui-ci |
| 2026-09-15 | 1.9.1 (communs §2.3) | ce commit | `gofmt -l ./internal ./cmd` | sortie vide |
| 2026-09-15 | 1.9.1 (communs §2.3) | ce commit | `go vet` sur `film/... archlint/ replaybuild/ analysis/objectiveevents/ domain/replaydoc/ service/replayview/` | 0 diagnostic |
| 2026-09-15 | 1.9.1 (communs §2.3) | ce commit | `go test` sur les mêmes paquets | **ok** — filmdec 18,5 s · replay 17,3 s · fallback 0,21 s · archlint 13,7 s · replaybuild 1,10 s · objectiveevents 0,55 s · replayview 0,34 s |
| 2026-09-15 | 1.9.1 (communs §2.3) | ce commit | `CGO_ENABLED=1` + msys64/ucrt64 en tête du PATH : `go vet` puis `go test ./internal/sync/killcollector/` | **ok** 0,11 s |
| 2026-09-15 | 1.9.1 (communs §2.3) | ce commit | `make -C ../.. go-api-lint` (`GOLANGCI_LINT_CACHE` isolé) | **0 issue**, baseline non accrue |
| 2026-09-15 | 1.9.1 (web, le schéma monte) | ce commit | `make generate-types` puis `make check-types` | `openapi.yaml -> generated.ts` (3 champs neufs) ; `tsc -b` **0 erreur** |
| 2026-09-15 | 1.9.1 (web) | ce commit | `make test-web` | **711 fichiers passés + 1 sauté ; 7 629 tests + 17 sautés ; 103,0 s** — aucune retouche de rendu (D11) : le web lit déjà `deployed` / `dropped` / `unknown` |
| 2026-09-15 | 1.9.1 (mutations) | ce commit | `go test …/replay/ -run 'TestOriginePose\|TestFenetreFaussee\|TestByCause\|TestDesignationExige\|TestPieceEngendree'` — `equipment_origin_lecture_test.go`, chaque lecture jouée DEUX FOIS | **VERT.** (a) 103 retiré -> la pose passe de `byCause.spawn_event` à `manifest_piece`, repli compté 1 ; (b) mort retirée -> `death_written` -> **`unknown` par `none`** ; (c) prise retirée -> `taken_written` -> **`unknown` par `none`** ; (d) **fenêtre faussée (pose à mi-vie PUIS à la fin de vie) : RIEN ne change pour un panneau désigné** ; (e) contradiction -> comptée sous `both`, AUCUN repli déclenché (D14 b) ; (f) la clé (slot, gen) seule ne désigne pas : +200 ms passe, +300 ms non, l'événement qui PRÉCÈDE non, une génération voisine non |
| 2026-09-15 | 1.9.1 (registre des replis) | ce commit | recensement du paquet `fallback` après retrait | **95 -> 94 entrées** (deux sorties, une neuve), **10 compteurs câblés**, ordres `apres_lecture` 77 / `sans_lecture` 10 / **`devant_la_lecture` 7 — INCHANGÉ**. Vérifié sur pièces : ni `repli_origine_pose_fenetre_temporelle` (`film_muet / sans_lecture`) ni `repli_origine_pose_vie_la_plus_proche` (`non_resolu / apres_lecture`) n'était de cet ordre, donc le ratchet des sept ne pouvait pas baisser ici |
| 2026-09-15 | 1.9.1 (tests retirés) | ce commit | `TestEquipmentOriginSepareLacherDuDeploiement`, `TestEquipmentOriginSansVieEstInconnue`, `TestEquipmentOriginChoisitLaVieQuiContientLInstant` supprimés avec la règle qu'ils verrouillaient | `.ai/baselines/tests_pre_migration.jsonl` : **3 lignes retirées dans le MÊME commit** (leçon du lot 1.0 : un test de baseline absent du run = CI rouge) |
| 2026-09-15 | 1.9.1 (équivalence, régime court — CLASSIFICATION AVANT tout `-update`) | ce commit | les 10 enfants produits à part (`replay-equiv -child -out <scratch>`), puis comparaison ÉTAPE PAR ÉTAPE **contre les références de la base `3718228c9`** (le comparateur du parent est POSITIONNEL : une étape neuve décale tout, il ne sait pas classer) | **1 étape NEUVE (`spawnEvents`), 0 étape PERDUE, et seul `artifact` change — sur 10 films sur 10.** Les 52 autres étapes sont IDENTIQUES partout : aucun balayage n'a changé. Delta d'octets d'`artifact` : 50247b26 5 601 211 -> 5 600 811 · a521164d 3 406 793 -> 3 406 910 · 60ae07c4 3 068 240 -> 3 068 264 · 11de8353 5 347 717 -> 5 347 685 · 111fa685 5 935 635 -> 5 935 757 · e5adf7b2 5 836 274 -> 5 836 251 · bcb6d393 1 904 308 -> 1 904 306 · 51101d1d 628 801 -> 628 798 · d9781168 2 641 716 -> 2 641 712 · fb1a1a72 3 016 287 -> 3 016 354. **DIVERGENCE VOULUE, pas régression** : deux champs de couverture apparaissent et les origines changent comme la décision du 2026-09-15 l'exige |
| 2026-09-15 | 1.9.1 (équivalence, re-figeage puis passe de comparaison) | ce commit | références re-figées depuis les 10 sorties classées, puis `go run ./cmd/replay-equiv -repo-root <worktree> -films <échantillon court>` en DEUX sous-ensembles | **BILAN : 5 identique(s), 0 différent(s) … puis 5 identique(s), 0 différent(s)** — 10/10 |
| 2026-09-15 | 1.9.1 (mesure des COMPOSANTS ECS, demandée par l'utilisateur) | ce commit | `CGO_ENABLED=0 E191_ROOT=… go test …/replay/ -run '^TestE191Composants$' -v` — `e191_composants_research_test.go`, PRÉSENCE AU MASQUE du record de création seulement, aucune grammaire de composant portée | **334,5 s, 13 films, 4 583 poses appariées à leur record de création, 0 orpheline.** Table [7] collée : sur **toutes** les familles et **toutes** les origines, la signature est `i10- i11- i18- i20- i21- i23-`. **AUCUN des six composants d'état n'est présent au record de création — 0 sur 4 583.** `i20 equipment-deployed` n'est donc PAS discriminant à l'instant de la pose : il est ABSENT partout. La règle du lot (103 / mort écrite / `taken`) tient ; D1 bis (1.9.1) en §4 |
| 2026-09-15 | 1.9.1 (corpus gate, régime complet) | ce commit | `go run ./cmd/replay-corpus-gate --base=3718228c9 --parc-root C:/…/LevelUp-go-migration --source-root C:/…/LevelUp-wt-decfilm-191` — 14 témoins (dont `4f77afc1`, ajouté par ce lot), 28 cuissons | **Sortie 1, et c'est ATTENDU : les 14 témoins sont en « PERTE », et toutes les pertes sont le DÉPLACEMENT VOULU par la décision utilisateur du 2026-09-15.** `bcb6d393` 15 gains / 8 pertes · `fb1a1a72` 16/8 · `d9781168` 12/13 · `c75f33b8` 12/9 · `bf15f7ab` 7/4 · `51ebbc0f` 14/5 · `084a804d` 26/18 · `0797ce72` 18/7 · `111fa685` 29/13 · `e5adf7b2` 28/20 · `60ae07c4` 8/6 · `a349fea8` 21/20 · `bfecd02b` 11/5 · `4f77afc1` 21/13. Schéma 58 -> 59 partout |
| 2026-09-15 | 1.9.1 (corpus gate — CLASSIFICATION DES PERTES, clé par clé) | ce commit | dépouillement exhaustif des lignes `perte` / `disparu` des 14 témoins | **TROIS familles de clés, et rien d'autre.** (1) `coverage.placements.deployed` / `.dropped` / `.byFamilyOrigin.<famille>/<origine>` — **le déplacement lui-même** : `deployed` s'effondre partout (81->…, 78->…, 70->…, 52->…, 33->…, 31->13 sur `4f77afc1`…) parce que le mot est désormais réservé aux pièces engendrées, et `dropped` baisse (504->…, 483->…, 458->…, 437->427…) parce que les poses muettes sortent en `unknown`. Les `byFamilyOrigin.<famille>/deployed` de toutes les familles PORTÉES DISPARAISSENT — c'est l'énoncé même de la décision. (2) `coverage.fallbacks/n` **3 -> 2** : le registre perd deux entrées (`repli_origine_pose_fenetre_temporelle`, `repli_origine_pose_vie_la_plus_proche`) et en gagne une. (3) `coverage.pickups.originUnknown` / `.originGround` / `pickups.origin/presents` — **effet DOWNSTREAM, nommé en D1 ter (1.9.1)** : `pickup_origin.go` réutilise `Origin == dropped` pour classer un ramassage `ground`. `originUnknown` **BAISSE** sur les 9 témoins où il bouge (moins de ramassages non classés) ; `originGround` ne baisse que sur un témoin (21->18). **ZÉRO perte hors ces trois familles** : identité, calques, faits, score, véhicules, drapeau — rien ne bouge |
| 2026-09-15 | 1.9.1 (gate rejoué après interruption) | ce commit | le corpus gate lancé avant l'interruption de session avait été TUÉ au démarrage (exit 4, `verification des modifications locales` interrompue) — un gate non consigné n'a pas eu lieu | **rejoué en entier**, résultat ci-dessus. Une première passe informative (code antérieur à `spawnLists`) avait rendu 0 perte / 3 à 9 gains sur 14 témoins : elle ne vaut PAS pour le code livré et n'est citée que pour mémoire |
| 2026-09-15 | 1.9.2 (MESURE AVANT DE CODER — catalogue contre auto-détection) | ce commit | `CHUNK00_FILMS='<les 17 répertoires>' CGO_ENABLED=0 go test ./internal/games/halo_infinite/film/filmdec/ -run '^TestE192CatalogueContreDetection$' -v -count=1 -timeout 3600s` (214,1 s ; 17 films = les 14 témoins du corpus gate + `50247b26`, `a521164d`, `11de8353` pour compléter les 8 builds) | **Tableau collé de l'instrument, colonnes : positions sous les réglages de PRODUCTION (`DefaultScanFilmOptions` + bornes), pistes de touche (`BuildBipedTracks` : filtres désarmés), enregistrements BRUTS (tag exigé, aucun filtre).** `0797ce72` HI_1_13_0 Live Fire `gate=6 region=1 12/12/11` contre auto `gate=5 13/12/11` **DIFF** 146 811 -> 146 807 (−4) · pistes −4 · brut 146 860 -> 146 849 (−11) ; `084a804d` HI_1_10_0 Fortitude Heavies `gate=5 15/15/17` = 330 769 (+0) ; `111fa685` HI_1_10_0 Command `15/15/17` = 267 391 (+0) ; `11de8353` HI_1_9_0 Thunderhead `15/15/17` = 240 309 (+0) ; `4f77afc1` HI_1_13_0 Flood Gulch `15/15/17` = 378 171 (+0) ; `50247b26` sans section Oasis `15/15/14` = 188 201 (+0) ; `51ebbc0f` HI_1_13_0 Banished Narrows `15/15/17` = 162 225 (+0) ; **`60ae07c4` HI_1_8_0 Live Fire - Ranked `gate=6 region=1 12/12/11` contre auto `gate=5 13/12/11` DIFF 267 368 -> 267 365 (−3) · pistes 267 390 -> 267 374 (−16) · brut 267 400 -> 267 374 (−26)** ; `a349fea8` sans section Fragmentation Heavies `17/17/15` = 332 367 (+0) ; `a521164d` HI_1_4_1 Fragmentation Heavies `17/17/15` = 135 972 (+0) ; `bcb6d393` HI_1_12_0 Cliffhanger `13/13/14` = 110 004 (+0) ; `bf15f7ab` HI_1_13_0 Perilous `15/15/17` = 169 423 (+0) ; `bfecd02b` HI_1_13_0 Snowbound `15/15/17` = 162 426 (+0) ; `c75f33b8` HI_1_13_0 Curfew `15/15/17` = 126 143 (+0) ; `d9781168` HI_1_13_0 Dredge `15/15/17` = 213 539 (+0) ; `e5adf7b2` HI_1_11_0 Fragmentation `17/17/15` = 251 164 (+0) ; `fb1a1a72` HI_1_13_0 Banished Narrows `15/15/17` = 264 219 (+0). **TOTAL : Δpositions −7, Δpistes −20, Δbruts −37. CARTES OÙ CATALOGUE ET DÉTECTION DIFFÈRENT : 2 — Live Fire (`0797ce72`) et Live Fire - Ranked (`60ae07c4`).** Tag de handle LIBRE sur les deux films divergents : `0797ce72` 146 893 -> 146 856 (−37), `60ae07c4` 267 468 -> 267 390 (−78) |
| 2026-09-15 | 1.9.2 (carte des sites, par grep — les citations du plan avaient dérivé) | ce commit | `grep -rn "DefaultScanFilmOptions()" --include=*.go internal/ cmd/ \| grep -v _test.go` puis `grep -A 12` sur chaque site | **QUATRE sites de production, pas deux** : `filmdec/weapon_hit_distance_resolver.go:70` -> `opt.Layout = &lay` (NEUF) · `replay/build_from_film.go:110` -> `s.scan.Layout = s.fc.ImposedLayout()` (déjà en place) · `replay/build_vehicles.go:173` -> `opt.Layout = fc.ImposedLayout()` (déjà en place) · `sync/killcollector/positions.go:251` -> `opt.Layout = fc.ImposedLayout()` (NEUF). `hits.go:157` du plan ne construit plus rien : le code avait déménagé |
| 2026-09-15 | 1.9.2 (les tirs et les dégâts ne lisent pas i0) | ce commit | `grep -c "I0Layout\|DetectI0" internal/games/halo_infinite/film/filmdec/weapon_hits.go` | **0** — `ScanFilmWeaponShots` et `ScanFilmWeaponDamages` n'ont aucune dépendance au découpage d'i0 : aucun enregistrement de tir ni de dégât ne change, et il n'y a rien à y convertir |
| 2026-09-15 | 1.9.2 (ce qui décide APRÈS le lot) | ce commit | `grep -rn "NewFilmContext(" --include=*.go internal/ cmd/ \| grep -v _test.go \| grep -v film/filmdec/` puis `grep -rn "DetectI0Layout" … \| grep -v _test.go` | **`NewFilmContext` (le contexte SANS catalogue) : ZÉRO appelant de production hors des enveloppes D2 de `filmdec`** (`killcollector/positions.go` était le dernier). **`DetectI0Layout` : UN appelant de production, `weapon_hit_distance_resolver.go:181`**, où elle décide l'IDENTITÉ de la carte et non son découpage — cible du lot 1.9.4. `DetectI0LayoutOf` n'est plus atteint que par `film_context.go:227` et `offline_biped_band.go:124`, tous deux conditionnés à « rien ne s'impose » |
| 2026-09-15 | 1.9.2 (la passe de touches est éteinte pour Infinite) | ce commit | `grep -rn "ConfigureFilmAccuracy(" --include=*.go internal/ cmd/` + `adapter_data.go:200` + `capabilities.toml:80` | **Aucun appelant de production** (deux sites, tous deux dans `hits_capability_test.go`) ; `games.CapWeaponAccuracy = CapNotExposed` et `"match.weapon.accuracy" = "not_exposed"`. La conversion du chemin des touches est donc sans effet en production — D1 (1.9.2) en §4 |
| 2026-09-15 | 1.9.2 (MUTATION A — le catalogue versionné faussé) | ce commit | `live fire` / `axisWidths[0]` **12 -> 13** dans `data/titles/halo_infinite/reference/map_quant_bounds.json`, puis les deux paquets | **ROUGE des deux côtés.** `filmdec` : `TestI0CatalogueEstLaSourceDuDecoupage` (« le catalogue donne `gate=6 region=1 13/12/11` à Live Fire, le témoin de bits est écrit sous `gate=6 region=1 12/12/11` ») et `TestI0CatalogueEcarteLesAutresRegions`. `killcollector` : `TestMutationDuCatalogueChangeLeDecoupageDesPositions` (même message). **Fichier restauré par `git checkout -- <le fichier>` ; `git status` vide sur `data/` après restauration, et les deux paquets reverts au VERT.** Cette mutation a d'abord été jouée contre une PREMIÈRE rédaction du témoin `filmdec` qui écrivait les bits sous le catalogue : **elle est restée VERTE**, ce qui a fait réécrire le témoin sur un découpage figé |
| 2026-09-15 | 1.9.2 (MUTATION B — le câblage retiré) | ce commit | `opt.Layout = fc.ImposedLayout()` supprimé de `killcollector/positions.go` | **ROUGE** : `TestPositionsImposentLeDecoupageDuCatalogue` échoue sur les **79 cartes** du catalogue (« `ScanFilmOptions.Layout` est nil — l'auto-détection déciderait du découpage d'i0 alors que le catalogue le porte (D-3 d'ADR 0034) ») et `TestMutationDuCatalogueChangeLeDecoupageDesPositions` sur « découpage de référence nil ». **Fichier restauré par NOM depuis une copie hors dépôt ; `grep -c "MUTATION B"` = 0, `opt.Layout = fc.ImposedLayout()` retrouvé à la ligne 254** |
| 2026-09-15 | 1.9.2 (registre des replis) | ce commit | `CGO_ENABLED=0 go test ./…/replay/fallback/ -run RapportDuRegistre -v -count=1` | **94 entrées, 6 paquets** (inchangé : la rétrogradation ne retire ni n'ajoute d'entrée) ; **6 replis `devant_la_lecture`** — `repli_manches_contigues_decretees`, `repli_largeur_absolue_uniforme`, `repli_chunk_du_pied_par_argmax`, `repli_type_de_chunk_perdu_du_manifeste`, `repli_fin_de_vie_vehicule_par_recensement`, `repli_vie_coupee_au_trou_de_replication`. `ratchetDevantLaLecture` **7 -> 6** dans le même commit |
| 2026-09-15 | 1.9.2 (gates communs §2.3) | ce commit | `gofmt -l ./internal ./cmd` ; `go vet` et `go test -count=1` sur `./internal/games/halo_infinite/film/... ./internal/archlint/ ./internal/replaybuild/ ./internal/sync/killcollector/ ./internal/analysis/objectiveevents/ ./internal/domain/replaydoc/` (CGO_ENABLED=1, msys64/ucrt64 en tête du PATH) | **gofmt vide ; vet 0 diagnostic ; 14 paquets `ok`** — `filmdec` 25,7 s, `replay` 14,7 s, `archlint` 11,7 s, `killsource` 1,1 s, `replaybuild` 1,1 s, `killcollector` 0,2 s, `objectiveevents` 0,5 s, `fallback` 0,2 s ; `domain/replaydoc` sans test |
| 2026-09-15 | 1.9.2 (gate intégration, le diff touche `internal/sync/`) | ce commit | `CGO_ENABLED=1 go test -tags=integration -p 1 ./internal/sync/killcollector/ -count=1` | **`ok  levelup/go-api/internal/sync/killcollector  14.799s`, code de sortie 0** |
| 2026-09-15 | 1.9.2 (lint baseline) | ce commit | `make -C ../.. go-api-lint` (`GOLANGCI_LINT_CACHE` isolé) | **`0 issues.`** — une première passe avait rendu **2 `goconst`** sur le littéral `"lot 1.9.4"` (4 occurrences après la rétrogradation) : centralisé en `fallback.lot194` dans le même commit, pas d'allowlist |
| 2026-09-15 | 1.9.2 (GrammarRev) | ce commit | `go test ./…/filmdec/ -run GrammarRevSuitLaGrammaire -update-grammar-rev` puis sans le drapeau | `grammar-2026-09-15.1 -> grammar-2026-09-15.2`, empreinte `02a22f5e8e5a550ec6e8f7aa50878341b3a00c8bdd8072f84e6f391297eeb2b6`, entrée d'historique écrite dans le golden. **`KillSourceDecoderRev` NON touchée** (le golden `killsource_decoder_rev.golden` reste vert : `killsource/` n'a pas bougé) |
| 2026-09-15 | 1.9.2 (population du parc concernée) | ce commit | `go run ./cmd/diag_q <backup pre-chaine-2026-09-09>/shared_matches_v2.duckdb "SELECT COUNT(*) total_matchs, COUNT(*) FILTER (WHERE lower(map_name) LIKE 'live fire%') live_fire FROM match_registry"` (read_only), croisé au cache de films | **1 967 matchs au registre, dont 70 Live Fire** (`live fire` 68 + `live fire - ranked` 2) ; **52 d'entre eux ont un film en cache** sur les 1 351 du cache. Ce sont les SEULS matchs dont les lignes changent ; `IsolationDecoderRev` rend éligible au redécodage tout match à positions, comme au lot 6.1 |
| 2026-09-15 | 1.9.2 (équivalence, régime court) | ce commit | `go run ./cmd/replay-equiv -repo-root C:/…/LevelUp-wt-decfilm-192 -films <échantillon court>` en DEUX sous-ensembles (jamais `LEVELUP_REPO_ROOT`) | `50247b26,a521164d,60ae07c4,11de8353,111fa685` : **BILAN : 5 identique(s), 0 different(s), 0 ecarte(s), 0 echec(s)** ; `e5adf7b2,bcb6d393,fb1a1a72,51101d1d,d9781168` : **BILAN : 5 identique(s), 0 different(s), 0 ecarte(s), 0 echec(s)**. **10/10 identiques, y compris `60ae07c4`** — et c'était l'attendu : le chemin de CUISSON imposait déjà le catalogue, ce lot ne touche que `killcollector` et le résolveur de distance. C'est aussi ce qui établit que `SchemaVersion` reste à 59 |
| 2026-09-15 | 1.9.2 (corpus gate, 14 témoins) | ce commit | `go run ./cmd/replay-corpus-gate --base=ec74685ed --parc-root C:/…/LevelUp-go-migration --source-root C:/…/LevelUp-wt-decfilm-192` (SANS `--manifest` : son défaut est le bon), 28 cuissons, 8 min 18 s | **SORTIE 0 — ZÉRO PERTE, ZÉRO GAIN sur les 14 témoins, schéma 59 des deux côtés.** `bcb6d393` ctf_mono_manche 59/59 0/0 · `fb1a1a72` ctf_multi_manche 0/0 · `d9781168` oddball 0/0 · `c75f33b8` assaut_bombe 0/0 · `bf15f7ab` slayer 0/0 · `51ebbc0f` deux_manches 0/0 · `084a804d` vehicules 0/0 (1 min 46) · **`0797ce72` region_index_2_bits 0/0** · `111fa685` version_39 0/0 · `e5adf7b2` version_40_build_1_11 0/0 · **`60ae07c4` version_37 0/0** · `a349fea8` version_33_sans_identification 0/0 (2 min 07) · `bfecd02b` vehicules_v41_utilisateur 0/0 · `4f77afc1` equipement_origine_utilisateur 0/0 (1 min 55). **ET C EST L ATTENDU, PAS UNE MESURE VIDE** : le corpus gate juge l ARTEFACT DE REJEU, que `replay.BuildFromFilm` cuit — or ce chemin imposait déjà le catalogue depuis le 2026-09-03. Ce lot convertit `sync/killcollector` (tables `kill_positions` / `kill_openings` / `match_lives`, hors artefact) et le résolveur de distance de touche (passe éteinte) : aucun octet du document ne pouvait bouger, et les deux témoins Live Fire — dont un que le manifeste nomme littéralement `region_index_2_bits` — le confirment |
| 2026-09-15 | 1.9.3 (MESURE AVANT DE CODER — le couple écrit contre le couple recollé) | ce commit | `CHUNK00_FILMS='<21 répertoires>' CGO_ENABLED=0 go test ./internal/games/halo_infinite/film/killsource/ -run TestE193CoupleEcritContreRecollage -v -count=1 -timeout 3600s` (21,3 s ; 21 films entiers = les 14 témoins du corpus gate + `50247b26`, `a521164d`, `11de8353` pour compléter les 8 builds + les 4 films de la série de référence `9b191a7f`, `000d5950`, `fccc61cd`, `78919882` ; plus la bobine versionnée, mesurée sans garde) | **[1] LE KILL-FEED, film par film — `film · build · épinglés · bots · kills · morts · couples · même-instant · recollés · orphK · orphD · kill-events 85` :** `000d5950` HI_1_13_0 8/0 93 93 93 76 **17** 0 0 100 · `000d5950(bob)` 8/0 93 93 93 76 **17** 0 0 11 · `0797ce72` HI_1_13_0 8/0 92 93 92 81 **11** 0 1 101 · `084a804d` HI_1_10_0 24/0 354 356 354 330 **24** 0 2 444 · `111fa685` HI_1_10_0 24/0 191 193 191 177 **14** 0 2 226 · `11de8353` HI_1_9_0 24/0 165 166 165 150 **15** 0 1 198 · `4f77afc1` HI_1_13_0 26/2 298 300 294 276 **18** 4 6 411 · `50247b26` sans section 0/0 150 154 150 147 **3** 0 4 48 · `51ebbc0f` HI_1_13_0 8/0 70 71 70 61 **9** 0 1 82 · `60ae07c4` HI_1_8_0 8/0 162 163 162 155 **7** 0 1 170 · `78919882` HI_1_13_0 8/0 99 99 99 86 **13** 0 0 106 · `9b191a7f` HI_1_13_0 9/1 87 87 85 74 **11** 2 3 101 · `a349fea8` sans section 0/0 290 293 290 259 **31** 0 3 69 · `a521164d` HI_1_4_1 24/0 101 106 101 89 **12** 0 5 30 · `bcb6d393` HI_1_12_0 8/0 49 49 48 47 **1** 1 1 56 · `bf15f7ab` HI_1_13_0 8/0 75 76 75 70 **5** 0 1 84 · `bfecd02b` HI_1_13_0 8/0 84 84 84 74 **10** 0 0 96 · `c75f33b8` HI_1_13_0 9/1 62 61 61 49 **12** 1 0 70 · `d9781168` HI_1_13_0 8/0 142 144 142 132 **10** 0 2 150 · `e5adf7b2` HI_1_11_0 23/0 197 199 197 186 **11** 0 2 236 · `fb1a1a72` HI_1_13_0 8/0 140 141 140 126 **14** 0 1 159 · `fccc61cd` HI_1_13_0 9/1 97 96 95 72 **23** 2 1 105. **TOTAL bobine comprise : 3 081 couples reconstruits, 2 793 au même instant, 288 recollés, 10 kills orphelins, 37 morts orphelines.** **LES 64 SUR 372 DE L'AUDIT SE REJOUENT À L'UNITÉ** sur la série de référence : 17 + 11 + 23 + 13 = **64** recollés pour 93 + 85 + 95 + 99 = **372** couples |
| 2026-09-15 | 1.9.3 (MESURE — ce que le kill-event 85 écrit en face) | ce commit | même instrument, tableaux [2] à [5] | **[2] RÉGIME NAÏF (le tueur seul, 21 films entiers) : 271 kills sans mort en face — accord 186 · désaccord 3 · victime BOT 1 · ambigu 13 · muet 68.** **[5] RÉGIME D'ASSIGNATION (celui du lot : les couples du même instant consomment d'abord leur kill-event) : 281 — accord 198 · désaccord **0** · victime BOT **1** (`4f77afc1`) · sans-mort-au-feed 0 · ambigu **1** (`bf15f7ab`) · muet 81.** Détail des muets : `a349fea8` 31, `a521164d` 12, `4f77afc1` 10, `084a804d` 6, `c75f33b8` 5, `50247b26` 3, `11de8353` 3, `9b191a7f` 3, `e5adf7b2` 3, `111fa685` 2, `fccc61cd` 2, `bcb6d393` 1 (D1 (1.9.3) en §4). **[3] CONTRÔLE sur les 2 793 couples du MÊME INSTANT, régime naïf : 1 898 accords, 10 désaccords, 3 victimes bot, 120 ambigus, 762 muets** — ces désaccords et ces ambigus sont des enregistrements qui appartiennent à une AUTRE mort du même joueur dans la fenêtre de 2,5 s, et c'est précisément ce que le premier temps d'assignation élimine. **[4] les 10 kills orphelins sont TOUS muets** : aucune victime humaine n'est ajoutée par la lecture, donc aucun couple neuf n'apparaît |
| 2026-09-15 | 1.9.3 (carte des sites, par grep) | ce commit | `grep -rn "reconstructPairs\|kf.split()" --include=*.go . \| grep -v _test.go` puis `grep -rn "resoudreCouples" --include=*.go .` | **`reconstructPairs` et `killFeed.split` : ZÉRO site de production** (une seule occurrence restante, dans un commentaire d'historique de `feed_couples.go`). **`resoudreCouples` : UN appelant de production, `killsource/decode.go:142`**, dans `prepare`, après `buildRoster` et avant `solveBijection` — l'ordre est forcé et il est le résultat |
| 2026-09-15 | 1.9.3 (MUTATION A — tueur et victime échangés dans la fixture) | ce commit | `coupleRec(2000, 1, 3)` -> `coupleRec(2000, 3, 1)` dans `feed_couples_test.go`, puis le paquet | **ROUGE** : `TestLeCoupleVientDuKillEvent85` — « couples = [{timeMS:2000 killer:C victim:D victimXUID:44}], attendu le seul (C, E) — le film ECRIT E, le voisin porte D ». Le témoin est bâti pour que lecture et repli DIVERGENT, sans quoi la mutation resterait verte (198 accords sur 281 dans la vraie vie). **Fichier restauré depuis une copie hors dépôt ; paquet revert au VERT** |
| 2026-09-15 | 1.9.3 (MUTATION B — la lecture débranchée en production) | ce commit | `lireVictime(...)` remplacée par `-1, -1, false` dans `resoudreUnKillSansMort` | **ROUGE sur QUATRE tests** : `TestLeCoupleVientDuKillEvent85`, `TestUneVictimeBotNeFabriquePlusDeCouple`, `TestLeRepliNeSertPasUneLectureAMBIGUE` et le golden `TestGoldenMiniBobine`. **Fichier restauré depuis une copie hors dépôt ; `go test` du paquet VERT après restauration** |
| 2026-09-15 | 1.9.3 (golden `minibobine.golden`) | ce commit | `CGO_ENABLED=0 go test ./…/killsource/ -run TestGoldenMiniBobine -update -count=1` puis sans le drapeau | **UNE SEULE LIGNE CHANGE sur les 83 figées, et elle est attribuée** : `morts de BOT : 17 propose(s)` -> `16 propose(s)`. Cause : le seul couple de la bobine que le kill-event 85 décide cesse d'être un couple RECOLLÉ, donc un candidat à la mort de bot. **Les 10 lignes publiées, les trois dénominateurs de couverture (93/93/93), le roster, la provenance des indices et le contrôle négatif sont IDENTIQUES.** `git diff --stat` : 1 insertion, 1 suppression |
| 2026-09-15 | 1.9.3 (registre des replis) | ce commit | `CGO_ENABLED=0 go test ./…/replay/fallback/ -run 'RegistrePorteToutesSesFamilles|ReplisDevantLaLectureNeMontentPas|CompteurBranche' -v -count=1` | **95 entrées** (94 -> 95 : `repli_couple_recolle_sur_le_voisin` CRÉÉE), **10 compteurs câblés**, **`NbDevantLaLecture` = 6, ratchet INCHANGÉ** — vérifié sur pièces : le recollage des couples venait de la table (A) de l'audit 0.E (ligne A3), il n'avait aucune entrée à rétrograder |
| 2026-09-15 | 1.9.3 (révisions) | ce commit | `go test ./…/filmdec/ -run GrammarRevSuitLaGrammaire -update-grammar-rev` ; `CGO_ENABLED=1 go test ./internal/sync/killcollector/ -run TestKillSourceDecoderRevSuitLeDecodeur -update` | **`GrammarRev` grammar-2026-09-15.2 -> grammar-2026-09-15.3**, empreinte `be7d58fcb7abde479867dabbf6d2e6914b1cc8549a42974fcca2c86d2e6f2618`. **`KillSourceDecoderRev` killsource-2026-09-14 -> killsource-2026-09-15** — le ratchet d'empreinte a **ROUGI DE LUI-MÊME** au commit du décodeur (« LE DECODEUR A CHANGE », `729624c1…` -> `ed7e45ca…`, 22 fichiers non-test) : preuve par mutation naturelle. Entrées d'historique écrites dans les DEUX goldens |
| 2026-09-15 | 1.9.3 (population du parc, oracle en lecture seule) | ce commit | `go run ./cmd/diag_q <backup pre-chaine-2026-09-09>/shared_matches_v2.duckdb "SELECT rev, COUNT(*) FROM (SELECT match_id, MAX(decoder_rev) rev FROM match_kill_events GROUP BY 1) GROUP BY 1"` | **1 384 matchs portent des lignes de kill** (751 `killsource-2026-09-05`, 589 `killsource-2026-07-31`, 41 `sync-kill-feed-2026-08-02`, 3 `highlight-credit-2026-08-01`) — **aucun sous `killsource-2026-09-14`** : le backlog inscrit par le lot 1.8 n'a pas été joué, donc cette montée ne recrute AUCUN match supplémentaire (D4 (1.9.3) en §4). La shared de production n'a PAS été lue |
| 2026-09-15 | 1.9.3 (gates communs §2.3) | ce commit | `gofmt -l ./internal ./cmd` ; `go vet` et `go test -count=1` sur `./internal/games/halo_infinite/film/... ./internal/archlint/ ./internal/replaybuild/ ./internal/sync/killcollector/ ./internal/analysis/objectiveevents/ ./internal/domain/replaydoc/` (CGO_ENABLED=1, msys64/ucrt64 en tête du PATH) | **gofmt vide ; vet 0 diagnostic ; 13 paquets `ok`** — `filmdec` 31,9 s, `replay` 17,8 s, `archlint` 14,4 s, `killsource` 1,1 s, `replaybuild` 0,9 s, `killcollector` 0,1 s, `objectiveevents` 0,5 s, `fallback` 0,2 s ; `domain/replaydoc` sans test |
| 2026-09-15 | 1.9.3 (gate intégration, le diff touche `internal/sync/`) | ce commit | `CGO_ENABLED=1 go test -tags=integration -p 1 ./internal/sync/killcollector/ -count=1` | **`ok  levelup/go-api/internal/sync/killcollector  14.255s`, code de sortie 0** |
| 2026-09-15 | 1.9.3 (lint baseline) | ce commit | `make -C ../.. go-api-lint` (`GOLANGCI_LINT_CACHE` isolé) | **`0 issues.`** — aucune dette ajoutée, aucune allowlist élargie |
| 2026-09-15 | 1.9.3 (équivalence — CLASSIFICATION AVANT TOUT `-update`) | ce commit | `replay-equiv -child -film <short8> -out <fichier> -repo-root C:/…/LevelUp-wt-decfilm-193` pour chacun des 10 films de l'échantillon court, UN PAR PROCESSUS, puis `diff` de chaque `.tsv` contre sa référence | **10 films sur 10 : UNE SEULE étape différente, `killsource`** (`50247b26`, `a521164d`, `60ae07c4`, `11de8353`, `111fa685`, `e5adf7b2`, `bcb6d393`, `fb1a1a72`, `51101d1d`, `d9781168` — 1 étape sur 53 chacun). **Les 52 autres sont identiques à l'octet**, vérifié nommément sur `artifact` (`111fa685` 5 935 757 o sha `8f123555…` des deux côtés ; `d9781168` 2 641 712 o sha `1ca74549…`), `killRefs`, `neutralDeaths` et `deaths`. Cause : la FORME de l'objet observé change (`observe("killsource", ksRes)` hache le `Result` entier, `Stats.Couples` est neuf), pas son contenu — les trois projections du même `Result` ne bougent pas d'un bit. Motif identique au lot 1.8 (D4 (1.8)) |
| 2026-09-15 | 1.9.3 (équivalence — re-figeage puis comparaison) | ce commit | `go run ./cmd/replay-equiv -repo-root C:/…/LevelUp-wt-decfilm-193 -films <5 films> -update` en DEUX sous-ensembles séquentiels, puis les deux mêmes SANS `-update` | `-update` : **5 + 5 identiques, 0 différent, 0 écarté, 0 échec**. `git diff` des références : **10 fichiers, 10 `+killsource` / 10 `-killsource` ET RIEN D'AUTRE** (1 insertion, 1 suppression par fichier). Passe de comparaison : **5 + 5 identiques, 0 différent** |
| 2026-09-15 | 1.9.3 (corpus gate, 14 témoins) | ce commit | `replay-corpus-gate --base=9848b7387 --parc-root C:/…/LevelUp-go-migration --source-root C:/…/LevelUp-wt-decfilm-193` (SANS `--manifest`), 28 cuissons | **SORTIE 0 — ZÉRO PERTE, ZÉRO GAIN sur les 14 témoins, schéma 59 des deux côtés.** `bcb6d393` ctf_mono_manche 0/0 16,2 s · `fb1a1a72` ctf_multi_manche 0/0 33,0 s · `d9781168` oddball 0/0 24,6 s · `c75f33b8` assaut_bombe 0/0 14,6 s · `bf15f7ab` slayer 0/0 14,0 s · `51ebbc0f` deux_manches 0/0 18,2 s · `084a804d` vehicules 0/0 1 min 51 · `0797ce72` region_index_2_bits 0/0 13,1 s · `111fa685` version_39 0/0 47,3 s · `e5adf7b2` version_40_build_1_11 0/0 39,6 s · `60ae07c4` version_37 0/0 25,3 s · `a349fea8` version_33_sans_identification 0/0 2 min 36 · `bfecd02b` vehicules_v41_utilisateur 0/0 19,5 s · **`4f77afc1` equipement_origine_utilisateur 0/0 1 min 39** — le seul film dont un couple change |
| 2026-09-15 | 1.9.3 (SchemaVersion — ZÉRO PERTE NE SUFFISAIT PAS) | ce commit | gate rejoué sur manifeste RÉDUIT (`4f77afc1` seul, hors dépôt) avec `--work-root` et `--keep-work`, puis `cmp` et `sha256sum` des DEUX artefacts cuits | **LE BILAN IMPRIMÉ NE PORTE QUE `gains` ET `pertes`** : `replaydiff.BilanAxe` compte aussi `Changements`, que ni le tableau ni `pertesDetail` ne remontent (D5 (1.9.3) en §4) — sur le seul film dont un couple change, un `changement` serait resté invisible. **VERDICT PAR LES OCTETS : base et HEAD rendent 10 526 185 octets, sha256 `62979da9924702d3a826c585ea7bf6a38752ad089164978ff142b7f79660a476` des DEUX côtés, `cmp` muet — IDENTIQUES À L'OCTET.** Donc **`SchemaVersion` = 59, INCHANGÉE** ; aucune chaîne complète, aucun `openapi.yaml` régénéré (`git status` : ni `api/openapi.yaml` ni `apps/web/` touchés). Racine de travail supprimée après mesure (529 Mio), verrou de décodage rendu, 0 processus de décodage restant |
| 2026-09-15 | 1.9.4 (MESURE (a) — les appelants, par grep collé) | ce commit | `grep -rn "DetectFilmMapEntry(\|DetectFilmWorldRange(\|DetectI0Layout(\|DetectI0LayoutOf(" --include=*.go . \| grep -v _test.go` | **`DetectFilmWorldRange` n'existe plus** (renommée au lot 1.9.2). `DetectFilmMapEntry` : 1 définition + **1 appelant de production**, `sync/killcollector/hits.go:170`, avec `mapNameOverride` VIDE — et la carte y EST connue par le nom (`positions.go`, `resolveMapBounds`, même collecteur). `DetectI0Layout` : **1 appelant de production**, à l'intérieur de `DetectFilmMapEntry` ; ~45 appelants de TEST. `DetectI0LayoutOf` : **2 appelants de production** (`film_context.go:227`, `offline_biped_band.go:124`), tous deux le repli « aucune entrée de catalogue imposée » — la carte n'y est PAS connue, ils ne relèvent pas de ce lot |
| 2026-09-15 | 1.9.4 (MESURE (c) — cartes jumelles du catalogue) | ce commit | `CGO_ENABLED=0 go test -tags research …/filmdec/ -run '^TestE194CartesJumellesDuCatalogue$' -v -count=1` (instrument versionné `e194_carte_par_nom_mesure_research_test.go`) | **LE PLAN EN ANNONÇAIT 6, IL Y EN A 68 SUR 79.** 5 classes d'équivalence de signature : **`15/15/17` 59 cartes**, `17/17/15` 3 (`behemoth`, `fragmentation`, `launch site`), `14/13/15` 2 (`prism`, `scarr`), `15/15/15` 2 (`catalyst`, `deadlock`), `18/18/17` 2 (`chasm`, `illusion`). **Seules 11 cartes sur 79 ont une signature unique.** Cause : `W = min(26, ceilLog2(ceil(60*étendue)))` par axe est une grandeur trop grossière pour distinguer des cartes de taille voisine |
| 2026-09-15 | 1.9.4 (MESURE (b) — signature contre nom de match, 17 films) | ce commit | `CHUNK00_FILMS='<les 17 répertoires>' CGO_ENABLED=0 go test -tags research …/filmdec/ -run '^TestE194SignatureContreNomDeMatch$' -v -count=1` — 3,15 s, lecture seule, au plus 6 chunks par film (`detectMaxChunks`), aucune cuisson | **2 ACCORDS, 13 AMBIGUËS, 2 DÉSACCORDS.** Accords : `bcb6d393` Cliffhanger (`13/13/14`, 1 candidat), `50247b26` Oasis (`15/15/14`, 1 candidat). Ambiguës : 10 films en classe `15/15/17` (59 candidats) et 3 films Fragmentation/Heavies en `17/17/15` (3 candidats) — `len(hits) != 1`, distances désactivées. **Désaccords : `0797ce72` et `60ae07c4` (Live Fire)** — le catalogue dit `gate=6 region=1 12/12/11`, la signature détectée est `13/12/11` et elle retrouve **EXACTEMENT UNE entrée : `aquarius`**. Tableau intégral dans la sortie de l'instrument |
| 2026-09-15 | 1.9.4 (contrôle du présupposé de mutation) | ce commit | `go test ./internal/sync/killcollector/ -run TestJumellesDuCatalogueSontIndistinguablesParSignature -count=1` | vert — `fragmentation` et `behemoth` partagent `17/17/15` et ont des AABB différentes (étendue X 1 679,7 m contre 1 132,4 m). Le test garde le témoin AVANT la mutation : si le catalogue les rendait un jour distinguables, il le dirait au lieu de laisser la mutation prouver le vide |
| 2026-09-15 | 1.9.4 (MUTATION A — la carte témoin remplacée par sa jumelle des DEUX côtés) | ce commit | `carteTemoinDesJumelles` `"Fragmentation"` -> `"Behemoth"`, puis `go test …/killcollector/ -run 'TestCarteDesTouchesVientDuNomDeMatch\|TestJumellesDuCatalogueSontIndistinguablesParSignature'` | **DEUX tests ROUGES**, dont le garde du présupposé en premier (« Behemoth et Behemoth ont desormais les MEMES bornes : fausser le nom serait sans effet, donc intestable »). Restauré par copie hors dépôt |
| 2026-09-15 | 1.9.4 (MUTATION B — LA FIXTURE seule faussée, celle que le brief demande) | ce commit | le collecteur reçoit `jumelleDeLaCarteTemoin` quand l'attente reste la carte témoin, puis `go test …/killcollector/ -run TestCarteDesTouchesVientDuNomDeMatch` | **ROUGE, deux assertions** : « bornes `[{-621.99 510.37} {-664.67 566.11} {-123.75 376.25}]`, attendues celles de Fragmentation `[{-1136.27 543.44} {-702.30 784.08} {-250 250}]` » puis « les bornes rendues sont celles de Behemoth, la JUMELLE d'échelle de Fragmentation ». **AVANT le lot cette mutation n'avait aucune prise** : la passe des touches ne recevait AUCUN nom de carte. Restauré par copie hors dépôt, constante revérifiée à `"Fragmentation"` |
| 2026-09-15 | 1.9.4 (gofmt) | ce commit | `gofmt -l ./internal ./cmd` | sortie vide |
| 2026-09-15 | 1.9.4 (vet) | ce commit | `CGO_ENABLED=1 go vet ./internal/games/halo_infinite/film/... ./internal/archlint/ ./internal/replaybuild/ ./internal/sync/killcollector/ ./internal/analysis/objectiveevents/ ./internal/domain/replaydoc/` (msys64/ucrt64 en tête du PATH) | 0 diagnostic |
| 2026-09-15 | 1.9.4 (tests) | ce commit | `CGO_ENABLED=1 go test` sur les mêmes paquets, `-count=1` | **tous `ok`** — filmdec 88,8 s · replay 14,2 s · archlint 11,6 s · killsource 1,17 s · replaybuild 0,89 s · killcollector 0,12 s · objectiveevents 0,39 s · fallback 0,11 s · replaydoc « no test files » |
| 2026-09-15 | 1.9.4 (intégration — le diff touche `internal/sync/`) | ce commit | `CGO_ENABLED=1 go test -tags=integration -p 1 ./internal/sync/killcollector/ -count=1` | **ok, 20,4 s** |
| 2026-09-15 | 1.9.4 (lint baseline) | ce commit | `make -C ../.. go-api-lint` (`GOLANGCI_LINT_CACHE` isolé) | **`0 issues.`** — aucune dette ajoutée, aucune allowlist élargie |
| 2026-09-15 | 1.9.4 (ratchet archlint — l'allowlist rougit dans les DEUX sens) | ce commit | retrait de l'entrée `weapon_hit_distance_resolver.go/DetectFilmMapEntry -> DetectI0Layout` de `appelsAutorisesDuContexte`, puis `go test ./internal/archlint/` | vert. C'EST LE RATCHET : l'allowlist refuse aussi bien un appel en trop qu'une entrée MORTE, donc toute réapparition d'un appel de `DetectI0Layout` dans `filmdec` rougira désormais |
| 2026-09-15 | 1.9.4 (garde-rail des replis, direction registre -> code) | ce commit | `go test ./internal/archlint/ -run TestToutSiteDuRegistreExiste` AVANT mise à jour du registre | **ROUGE comme prévu** : « `repli_carte_premier_nom_resolu` : l'ancre est absente de `sync/killcollector/positions.go` » — la résolution a déménagé dans `map_identity.go`. Entrée corrigée dans le même commit, puis vert. D14 (d) rendu mécanique, vérifié en situation |
| 2026-09-15 | 1.9.4 (registre des replis) | ce commit | `go test …/replay/fallback/ -run 'TestReplisDevantLaLectureNeMontentPas\|TestCompteurBranche' -v` | **96 entrées, inchangé** (aucune n'entre, aucune ne sort : la signature n'a PAS été rétrogradée en repli, elle est supprimée) ; **`NbDevantLaLecture` = 6, LE RATCHET NE DESCEND PAS, et ce n'est pas un oubli** — relevé sur pièces, les six sont `repli_largeur_absolue_uniforme`, `repli_chunk_du_pied_par_argmax`, `repli_type_de_chunk_perdu_du_manifeste`, `repli_manches_contigues_decretees`, `repli_vie_coupee_au_trou_de_replication`, `repli_fin_de_vie_vehicule_par_recensement` (cibles 1.9.8, 1.9.10, 1.9.11, 1.9.13), aucune de ce lot. TROIS entrées mises à jour (cibles et sites), la constante partagée `lot194` SUPPRIMÉE — les quatre cibles ont divergé |
| 2026-09-15 | 1.9.4 (révisions) | ce commit | `go test …/filmdec/ -run GrammarRevSuitLaGrammaire` puis `-update-grammar-rev` puis re-vérification | **`GrammarRev` `grammar-2026-09-15.7` -> `.8`**, empreinte `8463b183…` -> `84442e2e…`, golden régénéré AVEC sa ligne d'historique. **`KillSourceDecoderRev` NE MONTE PAS** (`film/killsource/` n'a pas bougé d'un octet ; `decoder_rev_fingerprint_test.go` vert). **`IsolationDecoderRev` NE MONTE PAS** : aucune sortie PERSISTÉE ne change — la passe qui écrit `kill_positions` / `kill_openings` n'est pas touchée, seule celle des TOUCHES l'est, et elle est ÉTEINTE en production (0 match). **`SchemaVersion` : 59, inchangée**, à confirmer par l'équivalence |
| 2026-09-15 | 1.9.4 (contrat publié) | ce commit | `git status --short apps/go-api/api/openapi.yaml apps/web/` | **sortie vide** — aucun contrat publié ne change, `openapi.yaml` n'est ni édité ni régénéré, `make test-web` et `check-types` sans objet |
| 2026-09-15 | 1.9.4 (seuils du dépôt — RELEVÉ INCOMPLET, corrigé plus bas) | ce commit | `wc -l` des fichiers touchés | `positions.go` **508 -> 494** (la résolution de carte en sort), `hits.go` 246 -> 296, `map_identity.go` 100, `weapon_hit_distance_resolver.go` 234 -> 209. **CE RELEVÉ NE COUVRAIT QUE LE CHEMIN DE COLLECTE** et concluait « aucun fichier au-dessus de 500 » à tort : il manquait `registre_killsource.go` (492 -> 523) et `collector.go` (773 -> 780), tous deux trouvés par la vérification sur pièces du pilote et fermés au commit de scission (ligne « seuil des 500 lignes » ci-dessous). La leçon est écrite pour le lot suivant : la liste des fichiers d'un gate de seuil se prend à `git diff --name-only`, jamais à ce qu'on a en tête |
| 2026-09-15 | 1.9.4 (baseline de tests) | ce commit | aucun test renommé ni supprimé — 8 tests AJOUTÉS (6 dans `killcollector`, 2 dans `filmdec`) | `.ai/baselines/tests_pre_migration.jsonl` **non modifiée**, et c'est correct : la baseline échoue sur un nom ABSENT du run, jamais sur un nom neuf |
| 2026-09-15 | 1.9.4 (règle neuve — tag `research` sur l'instrument) | ce commit | `//go:build research` posé sur `e194_carte_par_nom_mesure_research_test.go`, puis `go vet` SANS tag et AVEC `-tags research` sur `./internal/sync/killcollector/ ./internal/games/halo_infinite/film/...` | **0 diagnostic dans les DEUX builds**, `gofmt -l` vide, `go test ./internal/archlint/` vert dans les deux builds aussi. Effet du tag prouvé dans les deux sens : `go test …/filmdec/ -run '^TestE194'` rend « no tests to run », `go test -tags research …/filmdec/ -run '^TestE194CartesJumellesDuCatalogue$'` rend `ok` 0,06 s. **CE QUE LE TAG COÛTE, ET CE QUI LE COUVRE** : la mesure (c) ne coûtait que 0,06 s et tournait au run par défaut ; la règle s'applique quand même (un tag posé « sauf exceptions » ne se tient pas). Le présupposé de la mutation reste gardé SANS tag par `sync/killcollector/hits_carte_par_nom_test.go`, `TestJumellesDuCatalogueSontIndistinguablesParSignature` — sans lui, le tag aurait fait disparaître du run par défaut la seule vérification que la mutation du lot a un sens |
| 2026-09-15 | 1.9.4 (équivalence, régime court — SUR VOIE LIBRE DU PILOTE) | ce commit | `go run ./cmd/replay-equiv -repo-root C:/…/LevelUp-wt-decfilm-194 -films <les 10 de l'échantillon court>`, joué en DEUX moitiés séquentielles de 5 films (plafond de durée d'une commande), un seul décodage à la fois | **10 IDENTIQUES, 0 DIFFÉRENT, 0 ÉCARTÉ, 0 ÉCHEC, 0 ILLISIBLE.** Moitié 1 (`50247b26`, `a521164d`, `60ae07c4`, `11de8353`, `111fa685`) 3 min 33 ; moitié 2 (`e5adf7b2`, `bcb6d393`, `51101d1d`, `d9781168`, `fb1a1a72`) 2 min 11. **AUCUN `-update` joué, et aucun n'était nécessaire** : c'est exactement l'attendu du brief — le chemin de CUISSON imposait déjà le catalogue depuis le 2026-09-03 et n'appelait pas `DetectFilmMapEntry`, donc supprimer l'identification par signature ne pouvait pas déplacer un octet cuit. Les 53 étapes de chaque film sont identiques |
| 2026-09-15 | 1.9.4 (corpus gate, 14 témoins — SUR VOIE LIBRE DU PILOTE) | ce commit | `go run ./cmd/replay-corpus-gate --base=d71282e09 --parc-root C:/…/LevelUp-go-migration --source-root C:/…/LevelUp-wt-decfilm-194 --json <fichier>` (SANS `--manifest`), 28 cuissons, ~21 min | **SORTIE 0 — ZÉRO PERTE, ZÉRO GAIN, ZÉRO CHANGEMENT sur les 14 témoins, schéma 59 des deux côtés.** `bcb6d393` 11,8 s · `fb1a1a72` 30,8 s · `d9781168` 24,7 s · `c75f33b8` 14,8 s · `bf15f7ab` 13,7 s · `51ebbc0f` 18,1 s · `084a804d` 1 min 52 · `0797ce72` 12,9 s · `111fa685` 33,9 s · `e5adf7b2` 37,7 s · `60ae07c4` 24,8 s · `a349fea8` 2 min 07 · `bfecd02b` 19,3 s · `4f77afc1` 1 min 57 |
| 2026-09-15 | 1.9.4 (la colonne des CHANGEMENTS, lue au JSON — D5 (1.9.3) appliquée) | ce commit | dépouillement du rapport `--json` : `changements` / `gains` / `pertes` / `schemaReference` / `schemaHead` sur les 14 lignes | **14 lignes à `"changements": 0`**, 14 à `"gains": 0`, 14 à `"pertes": 0`, 14 à `"schemaReference": 59` et `"schemaHead": 59`. D5 (1.9.3) disait que le TABLEAU IMPRIMÉ n'affichait que gains et pertes ; relu sur pièces, le tableau porte désormais une colonne `chang.` ET `ligneRapport.aUnePerte()` (`cmd/replay-corpus-gate/report.go:55`) compte déjà `Changements > 0` comme un refus — le gate BLOQUAIT donc déjà dessus. Le `--json` le CHIFFRE au lieu de seulement le détecter. **VERDICT : `SchemaVersion` reste 59**, et il est établi par les trois colonnes, pas par la seule absence de perte |
| 2026-09-15 | 1.9.4 (deux gates de décodage, discipline machine) | ce commit | verrou `filmproc.AcquireSolo` observé dans les journaux (`verrou de decodage pris`, priorité CPU `below_normal`), racine de travail jetable supprimée à la fin | **un seul décodage à la fois** du début à la fin des deux gates ; aucune écriture dans le parc, aucune base ouverte en RW, 0 erreur et 0 `FATAL` au journal ; aucun processus de décodage résiduel |
| 2026-09-15 | 1.9.4 (seuil des 500 lignes — vérification sur pièces du pilote AVANT fusion) | ce commit | `wc -l` de CHAQUE fichier du lot, base `d71282e09` contre HEAD | **DEUX DÉPASSEMENTS TROUVÉS PAR LE PILOTE, LES DEUX FERMÉS.** (1) `registre_killsource.go` **492 -> 523** : SCINDÉ en `registre_killsource_carte.go` (les DEUX entrées que ce lot a réécrites, `repli_distances_de_touche_desactivees` et `repli_carte_premier_nom_resolu` — 67 lignes déplacées TELLES QUELLES, aucun champ, aucune valeur, aucun ordre modifié ; la tranche est branchée dans `concat` de `registre.go`). Résultat : **456 + 92**, les deux sous 500. Le registre était DÉJÀ découpé pour cette raison (« le découpage ne suit que la limite de 500 lignes du dépôt »), le commentaire passe de cinq à SIX fichiers. Contrôle d'intégrité : `go test …/fallback/` rend **96 entrées, 6 paquets, ratchet `NbDevantLaLecture` = 6, 10 compteurs branchés** — identique à avant la scission, donc déplacement PUR. (2) `collector.go` **773 -> 780 (+7)** : la croissance était de la PROSE que ce lot avait ajoutée aux commentaires de `mapNames`/`filmDir` et de `ConfigureFilmAccuracy`, alors que la substance vit déjà dans `map_identity.go`. Les trois blocs sont RESSERRÉS (renvoi au lieu de redite, rien de perdu) : **773 -> 773, croissance nette NULLE** sur un fichier de dette gelée. **Tous les autres fichiers du lot sont sous 500** : `hits.go` 296, `positions.go` 494 (-14), `map_identity.go` 100, `weapon_hit_distance_resolver.go` 209 (-25), `registre_filmdec.go` 254, `hits_carte_par_nom_test.go` 170, `e194…_research_test.go` 247. Gates rejoués après la scission : `gofmt -l` vide, `go vet` 0 diagnostic SANS tag ET avec `-tags research`, tests verts sur les 7 paquets (filmdec 96,1 s · archlint 12,9 s · killcollector 0,14 s), `make go-api-lint` **0 issues** |
| 2026-09-15 | 1.9.13 (mesure AVANT, l'instrument des orphelines tel quel) | base `6db15a9bc` | `CGO_ENABLED=0 go test ./…/replay/ -run TestViesDUnEchantillonOntLeurOracle -count=1 -v` | **20 vies d'un échantillon · 1 mort écrite · 5 fins de film · 14 ORPHELINES**, 1,20 s — la mesure du 2026-09-14 (D1 (1.6)) reproduite à l'identique |
| 2026-09-15 | 1.9.13 (mesure AVANT, la population que le lot convertit) | base `6db15a9bc` | `CGO_ENABLED=0 go test ./…/replay/ -run TestCoupuresDeVieOntLeurCause -count=1 -v` (instrument NEUF) | **212 coupures décidées par `lifeGapUS` sur les 8 builds · 4 mort écrite · 0 apparition écrite · 0 fin de manche · 0 fin de film · 208 RIEN (98,1 %)**. Par build : `000d5950` 6/0/6 · `a521164d` 57/2/55 · `60ae07c4` 8/0/8 · `11de8353` 64/0/64 · `111fa685` 33/0/33 · `e5adf7b2` 40/2/38 · `bcb6d393` 1/0/1 · `fb1a1a72` 3/0/3 (total/mort/RIEN). `bornesDeManchePosees=0` PARTOUT (D1 (1.9.13)) et `RIENdontJoueurSansAucuneMort=0` partout — la population du repli est VIDE sur cet échantillon |
| 2026-09-15 | 1.9.13 (mesure APRÈS) | ce commit | les deux mêmes instruments | coupures **212 → 4**, toutes `mort ecrite`, **RIEN 208 → 0** ; vies d'un échantillon **20 → 6** (2 morts écrites, 4 fins de film), **ORPHELINES 14 → 0**. Les deux instruments portent désormais un `t.Errorf` sur le total (`RIEN != 0`, `ORPHELINES != 0`) : le zéro est VERROUILLÉ, pas seulement constaté |
| 2026-09-15 | 1.9.13 (lacunes) | ce commit | `grep -h "lacune(s) de replication" …/testdata/assembly_*.golden` | **6 + 33 + 64 + 8 + 55 + 1 + 38 + 3 = 208 lacunes**, exactement les 208 coupures sans cause — l'identité des deux comptes est le contrôle de la conversion. Durées : 96 664 · 673 378 · 1 096 451 · 147 865 · 756 013 · 5 922 · 720 810 · 39 861 ms |
| 2026-09-15 | 1.9.13 (mutations, 6 jouées et restaurées PAR NOM) | ce commit | `go test ./…/replay/ -run <le test> -count=1` après chaque mutation | **(a) fixture, la mort écrite du trou retirée** → « 1 vie(s), attendu 2 » ROUGE · **(b) fixture, `lifeGapUS` remis en décideur (`vies: deuxSejoursDUnCorps()`)** → « 2 piste(s), attendu 1 » ROUGE · **(c) production, `fb.Declenche` retiré** → « repli declenche 0 fois, attendu 1 » ROUGE · **(d) production, la fin de manche cesse de fermer** → « 1 vie(s), attendu 2 » ROUGE · **(e) production, l'apparition écrite cesse d'ouvrir** → « 1 vie(s), attendu 2 » ROUGE · **(f) production, la lacune cesse de se publier** → « gaps=0 gapMs=0 » + « 0 point(s) portent une lacune » ROUGE. Toutes restaurées (`gofmt -l` vide, les 8 tests verts) |
| 2026-09-15 | 1.9.13 (registre des replis) | ce commit | `go test ./…/replay/fallback/ ./internal/archlint/ -count=1` | **verts**. `repli_vie_coupee_au_trou_de_replication` : `devant_la_lecture` → `apres_lecture`, site `tracks_publication.go` → `lives_decoupe.go`, `CompteurBranche` false → **true**. **`ratchetDevantLaLecture` 6 → 5** dans ce commit, avec sa justification datée |
| 2026-09-15 | 1.9.13 (goldens d'assemblage, porte nommée) | ce commit | `go test ./…/replay/ -run GoldenBuildsAssemblyRegenerate -update-golden-builds-assembly` puis `-run '^TestGoldenAssembly$' -update`, relancés sans drapeau | **8 références réécrites**, puis vertes. Diff : `vies nommées` garde son NUMÉRATEUR partout (90, 187, 150, 161) et perd son dénominateur (105 → 99, 245 → 212, 246 → 182, 177 → 169) ; impulsions de capacité `sans identite` 4 → 3 et 2 → 0, charges 20 → 13 et 8 → 1 ; **aucune ligne de BORNES ne bouge** (D3 (1.9.13)) |
| 2026-09-15 | 1.9.13 (`GrammarRev`) | ce commit | `go test ./…/filmdec/ -run GrammarRevSuitLaGrammaire` puis `-update-grammar-rev`, relancé sans drapeau | ROUGE de lui-même (« LA GRAMMAIRE A CHANGE SANS MONTEE DE REVISION ») : l'accesseur ajouté à `objectiveevents` entre dans l'empreinte. **`.7` → `.8`**, empreinte `06d7247408468122…`, historique complété DANS le golden. **`KillSourceDecoderRev` NON touché** (`killsource/` intact), **`SchemaVersion` NON touché** (règle de la vague) |
| 2026-09-15 | 1.9.13 (communs, sans décodage) | ce commit | `gofmt -l ./internal ./cmd` ; `go vet` (sans tag ET `-tags research`) ; `go test` sur film/…, archlint, replaybuild, objectiveevents, replaydoc, replayview ; `go test` + `-tags=integration -p 1` sur killcollector (CGO) | `gofmt` **vide** ; vet propre des deux côtés ; **tous verts SAUF les deux goldens que la règle de la vague fige** (D4 (1.9.13)) : `TestDocumentShapeMatchesGolden` et `TestContractFixturesMatchCommitted`. Durées : filmdec **16,276 s** (budget 30 s tenu) · replay 19,6 s · archlint 14,7 s · replaybuild 1,2 s · objectiveevents 0,6 s · killcollector 0,34 s puis 14,5 s en intégration |
| 2026-09-15 | 1.9.13 (lint) | ce commit | `make go-api-lint` (`GOLANGCI_LINT_CACHE` isolé) | **0 issues**, baseline non accrue |
| 2026-09-15 | 1.9.13 (équivalence, régime court — CLASSIFICATION avant tout `-update`) | ce commit | `go run ./cmd/replay-equiv -repo-root C:/…/LevelUp-wt-decfilm-1913 -films <5>` puis `<5>` | **0 identique, 10 différents, 0 écarté, 0 échec** (4 min 09 + 2 min 02). **UNE SEULE ÉTAPE DES 53 BOUGE : `artifact`.** Les **52 étapes de DÉCODAGE sont identiques à l'octet sur les dix films** — le lot ne lit aucun octet autrement, il change l'ASSEMBLAGE. C'est la signature attendue, et elle est exacte |
| 2026-09-15 | 1.9.13 (équivalence — le delta d'octets expliqué film par film) | ce commit | comparaison des comptes `artifact` + journaux de cuisson du même run | **une lacune = une vie qui ne se coupe plus = ~195 o d'en-tête de piste en moins.** `bcb6d393` 1 lacune −186 o · `fb1a1a72` 3 −600 · `51101d1d` 5 −970 · `60ae07c4` 8 −1 557 · `d9781168` 16 −3 251 · `111fa685` 33 −6 174 · `e5adf7b2` 38 −7 353 · `a521164d` 55 −11 227 · `11de8353` 64 −12 769. **`50247b26` MONTE de +79 o, et c'est le SEUL** : son registre ne rend AUCUNE vie (`slots=0 viesTotal=0`, film sans section d'identification), donc le **repli** reproduit l'ancienne découpe à l'identique et les +79 o sont les deux champs neufs (`gaps`, `gapMs`) plus l'entrée neuve de `coverage.fallbacks[]`. Aucune hausse d'un compteur de défaut nulle part |
| 2026-09-15 | 1.9.13 (équivalence — re-figeage puis comparaison) | ce commit | les deux mêmes sous-ensembles avec `-update`, puis les deux mêmes SANS | `-update` : **5 + 5 identiques, 0 différent, 0 écarté, 0 échec** (3 min 35 + 1 min 59). `git diff` des références : **10 fichiers, 1 insertion + 1 suppression chacun, TOUTES `artifact`** — aucune autre ligne. Passe de comparaison : **10/10 identiques**. Bloc de classification ajouté en fin de `CORPUS.txt` |
| 2026-09-15 | 1.9.13 (repli, sur les 10 films du régime court) | ce commit | `grep "repli=repli_vie_coupee_au_trou_de_replication"` des journaux de cuisson | **1 déclenchement, sur `50247b26` SEUL** (le film sans table de joueurs) ; **0 sur les neuf autres**. Le repli est donc exactement ce que D14 exige : il ne s'ouvre que sur un silence MESURÉ du film, jamais à la place d'une lecture |
| 2026-09-15 | 1.9.13 (corpus gate, 14 témoins) | ce commit | `replay-corpus-gate --base=6db15a9bc --parc-root C:/…/LevelUp-go-migration --source-root C:/…/LevelUp-wt-decfilm-1913 --json …` (SANS `--manifest`), 28 cuissons, ~9 min | **schéma 59 des DEUX côtés partout · 104 gains · 491 « pertes » · 2 changements · sortie 1.** `bcb6d393` 4/19/0 · `fb1a1a72` 3/21/0 · `d9781168` 3/29/**1** · `c75f33b8` 3/24/0 · `bf15f7ab` 3/25/0 · `51ebbc0f` 3/19/0 · `084a804d` 19/78/0 · `0797ce72` 16/30/0 · `111fa685` 17/65/0 · `e5adf7b2` 17/60/0 · `60ae07c4` 3/28/**1** · **`a349fea8` 1/0/0 (ok)** · `bfecd02b` 3/27/0 · `4f77afc1` 9/66/0 |
| 2026-09-15 | 1.9.13 (corpus gate — LES 491 « PERTES » CLASSÉES, AUCUNE NON CLASSÉE) | ce commit | `awk` sur `pertesDetail` du JSON | **Aucune métrique n'a le sens `disparu` : les 491 sont des compteurs qui BAISSENT, jamais une grandeur qui s'évanouit.** **(A) moins de vies — l'effet même du lot : 179** comptes de vies et de pistes (`tracks/n`, `tracks/vies-nommees`, `tracks.{slot,team,xuid,startFrame,endFrame}/presents`, `coverage.tracks.published`, `bridge.livesTotal`, `teams.tracks(Named)`, `t0Film.tracks(moving)`, `equipment.tracksTotal`) **+ 272** `tracks/(vies-)par-xuid/<joueur>`. **(B) compteurs de DÉFAUT qui tombent — 38** : `bridge.directByCreationPropagated` **13** (le record de création ouvre désormais LA vie du corps, plus le premier de ses séjours : il tombe à 0 ou 2 partout), `placements.unknown` / `byCause.none` / `byFamilyOrigin.*/unknown` **11**, `abilityImpulses/Charges.noIdentity` **8**, `grapple.pullLives` **4** (deux tractions du même joueur sont une seule vie), `bridge.discordant` **2** (27 → 22 et 21 → 17). **(C) `bridge.livesNamed` — 2 seulement**, −2 sur `084a804d` et −2 sur `4f77afc1`, quand leur `livesTotal` perd 90 et 113 |
| 2026-09-15 | 1.9.13 (corpus gate — LE CONTRÔLE QUI COMPTE) | ce commit | recherche des métriques ABSENTES des pertes | **`coverage.tracks.publishedPoints` : AUCUNE perte. `bounds` : AUCUNE perte. `shots`, `grenades`, `kills`, `objectives`, `flagCarries`, `scoreTimeline`, `projectiles`, `roster` : AUCUNE perte.** Aucun point de trajectoire, aucune borne de scène, aucun calque ne bouge : la conversion RECOLLE des vies, elle ne retire rien. C'est ce qui sépare une amélioration d'une régression, et c'est mesuré sur les 14 témoins |
| 2026-09-15 | 1.9.13 (corpus gate — `unnamedLives`, verdict D4 (1.6)) | ce commit | `livesTotal − livesNamed` par témoin | **`084a804d` : 154 → 66 vies sans nom (−88)** pour 90 fusions — les vies anonymes que le seuil révélait étaient bien des FRAGMENTS de vies nommées, elles se recollent. **`4f77afc1` : 183 → 72 (−111)** pour 113 fusions. Sur les 11 autres témoins à fusions, `livesNamed` est INCHANGÉ alors que `livesTotal` baisse : **`unnamedLives` baisse sur les 13**. **MAIS `a349fea8` — le second témoin de D4 (1.6) — ne bouge PAS** : 0 perte, 0 changement, découpe strictement identique, parce qu'il ne porte AUCUNE section d'identification (le registre ne rend aucune vie, le repli garde le seuil). D4 (1.6) est donc résolu sur `084a804d` et NON sur `a349fea8`, pour une raison de BUILD, pas de découpe |
| 2026-09-15 | 1.9.13 (corpus gate — lacunes déduites, concordance avec l'équivalence) | ce commit | delta de `coverage.bridge.livesTotal` par témoin | **335 vies fusionnées sur les 14 témoins** : `bcb6d393` 1 · `51ebbc0f` 3 · `fb1a1a72` 3 · `c75f33b8` 7 · `0797ce72` 7 · `bf15f7ab` 8 · `60ae07c4` 8 · `bfecd02b` 8 · `d9781168` 16 · `111fa685` 33 · `e5adf7b2` 38 · `084a804d` 90 · `4f77afc1` 113 · **`a349fea8` 0**. Les six témoins communs au régime court donnent le MÊME nombre que les journaux de cuisson de `replay-equiv` (1, 3, 8, 16, 33, 38) : les deux gates mesurent la même grandeur |
| 2026-09-15 | 1.9.13 (corpus gate — repli et colonne `changements`) | ce commit | lecture du rapport | **Le repli suit sa règle** : `a349fea8`, seul témoin SANS section d'identification, sort **0 perte / 0 changement / 1 gain** — la signature exacte d'une découpe inchangée plus l'entrée neuve de `coverage.fallbacks[]` ; les 13 autres ont une découpe qui CHANGE, donc le repli de `bornesDesVies` ne s'y est pas déclenché. Mesure directe du compte sur 7 des 14 témoins (journaux de `replay-equiv`) : **0 déclenchement**. **LIMITE ÉCRITE** : le rapport du gate ne détaille NI les gains NI les `changements` (D5 (1.9.3), déjà consignée), donc les 2 changements — 1 sur `d9781168`, 1 sur `60ae07c4`, 0 sur les douze autres — ne sont pas nommés par lui, et le contenu de `coverage.fallbacks[]` des 14 artefacts n'en sort pas |
| 2026-09-15 | 1.9.13 (dette gelée — vérification sur pièces du pilote) | ce commit | `wc -l` des fichiers touchés, base `6db15a9bc` contre HEAD | **DEUX FICHIERS DÉJÀ AU-DELÀ DU SEUIL DE 500 L AVAIENT GROSSI** : `build.go` 540 → 545 et `lives.go` 565 → 578. Corrigé par DÉPLACEMENT PUR : `buildLifeSpans` (l'échafaudage — sa place est désormais auprès de la découpe) passe de `lives.go` à `lives_decoupe.go`, et la pose des traces et des bornes de `build.go` à `tracks_publication.go` (`poserLesTraces`). **Résultat : `build.go` 540 → 532 (−8 sous la base), `lives.go` 565 → 536 (−29)** ; `lives_decoupe.go` 204 → 246, `tracks_publication.go` 202 → 326, tous deux sous le seuil. **Preuve du déplacement pur** : le `git diff` de `lives.go` (−42) et celui de `lives_decoupe.go` (+42) sont IDENTIQUES À L'OCTET, ligne pour ligne ; et les **huit goldens d'assemblage restent verts SANS régénération**, ce qu'un changement de logique aurait fait rougir |
| 2026-09-15 | 1.9.13 (gates rejoués après l'extraction) | ce commit | `gofmt` ; `go vet` sans tag ET `-tags research` ; `go test` film/…, archlint, replaybuild, objectiveevents, replayview ; killcollector (CGO) ; `make go-api-lint` | `gofmt` **vide** · vet propre des deux côtés · **tous verts** sauf les deux goldens que la règle de la vague fige (D4 (1.9.13)) · killcollector **ok** · lint **0 issues**. Aucune référence d'équivalence touchée : un déplacement de code dans `replay/` ne change ni un octet cuit ni l'empreinte `GrammarRev`, qui ne hache que `filmdec`, `killsource` et `objectiveevents` |
| 2026-09-15 | revue M1 — D13 (mesure AVANT de coder) | ce commit | `CHUNK00_CORPUS=<jonction> go test -tags research ./internal/games/halo_infinite/film/filmdec/ -run TestMPPResolutionCorpus -v` | **657 films lus** (cache en cours de restauration ; le plan en cite 1 351 au complet). Table (build × format) : `(sans section)`/20 ×5 · `HI_1_10_0`/24 ×30 · `HI_1_11_0`/25 ×48 · `HI_1_12_0`/27 ×131 **RELUE** · `HI_1_13_0`/27 ×430 **RELUE** · `HI_1_4_1`/21 ×1 · `HI_1_5_1`/23 ×1 **FORMAT INCONNU** · `HI_1_8_0`/24 ×10 · `HI_1_9_0`/24 ×1. **Builds hors table : 6 sur 657** (2 distincts). **OCTETS CUITS QUI CHANGERAIENT (format à largeur relue + profil complet refusé) : 0.** 4,20 s |
| 2026-09-15 | revue M1 — D13 (constat 1, mutation) | ce commit | `Declenche` déplacé sur `NomArmementBombeDebutAZero` dans `tracks_publication.go` | **ROUGE** : « le repli "repli_armement_bombe_debut_a_zero" y est déclenché, mais aucun [fallback.Site] de son entrée ne cite ce fichier (sites : [.../bomb_armings.go]) ». Restauré par nom |
| 2026-09-15 | revue M1 — L6 c4 (mutation, témoin du relecteur rejoué à l'identique) | ce commit | `func locateFallback() int { return 0 }` ajouté dans `filmdec/varwidth.go` | **ROUGE** : « REPLI HORS REGISTRE (1) : locateFallback (internal/games/halo_infinite/film/filmdec/varwidth.go) ». Vert avant l'indexation par fichier. Restauré par nom |
| 2026-09-15 | revue M1 — arch C2 (mutation) | ce commit | entrée morte `logFallbacks` remise dans `replisDeLEcrivainDuJeu` | **ROUGE** : « exemption logFallbacks SANS SITE : .../replay/build.go ne declare plus cet identifiant ». Restauré par nom |
| 2026-09-15 | revue M1 — D13 (mutation, constante orpheline) | ce commit | `NomOrphelinBidon Nom = "repli_orphelin_bidon"` ajoutée à `fallback/noms.go` | **ROUGE** : « la constante NomOrphelinBidon vaut "repli_orphelin_bidon", qui n'est au registre d'AUCUNE entree ». Restauré par nom |
| 2026-09-15 | revue M1 — D13 (constat 3, mutation) | ce commit | `concat(Tranches()[:5])` — une tranche retirée de l'assemblage | **ROUGE** : « les tranches portent 96 entrees, le registre en rend 85 — `concat` et `Tranches()` ont diverge ». Restauré par nom |
| 2026-09-15 | revue M1 — D13 (constat 2, mutation du défaut d'origine) | ce commit | `gwWidthsForFilm` remis à `BuildProfileFromFilm` puis `return calibrees` | **ROUGE** : « largeurs installees = 8/3, 9/5 attendues (les relues du format 27) — le site consulte-t-il encore le nom de build ? ». Restauré par nom |
| 2026-09-15 | revue M1 — D13 (constat 2, mutation du ratchet, appel NU) | ce commit | `BuildProfileFromFilm(fc.Film())` rétabli dans `filmdec/equipment_placements.go` | **ROUGE** : « ../filmdec/equipment_placements.go appelle `BuildProfileFromFilm` ». Les deux formes d'appel (nue et qualifiée) mordent. Restauré par nom |
| 2026-09-15 | revue M1 — D13 | ce commit | `gofmt -l ./internal ./cmd` | sortie vide |
| 2026-09-15 | revue M1 — D13 | ce commit | `go vet` (sans tag) sur `film/...`, `archlint`, `objectiveevents`, `replaydoc`, `replayview` ; puis `go vet -tags research ./internal/games/halo_infinite/film/...` | 0 diagnostic dans les deux passes |
| 2026-09-15 | revue M1 — D13 | ce commit | `go test ./internal/games/halo_infinite/film/... ./internal/archlint/ ./internal/replaybuild/ ./internal/sync/killcollector/ ./internal/analysis/objectiveevents/ ./internal/domain/replaydoc/ ./internal/service/replayview/ -count=1` (CGO=1) | **Verts partout sauf les DEUX rouges attendus par construction** (`TestDocumentShapeMatchesGolden`, `TestContractFixturesMatchCommitted` — D4 (1.9.13), le schéma monte UNE fois à la fusion de la vague). Le diff ne touche aucun champ publié : ces deux rouges préexistent à ce lot |
| 2026-09-15 | revue M1 — D13 | ce commit | `go test ./internal/games/halo_infinite/film/filmdec/ -count=1` (CGO=0, budget §2.3) | ok **23,1 s** — sous le budget de 30 s |
| 2026-09-15 | revue M1 — D13 | ce commit | `go test .../filmdec/ -run GrammarRevSuitLaGrammaire -update-grammar-rev` puis sans le drapeau | `grammar-2026-09-15.12` / empreinte `83d8367625eb850de9642cdf2b2fdd86abd5178bc29a66be1593cc3cef33984c` ; vérification verte |
| 2026-09-15 | revue M1 — D13 | ce commit | `golangci-lint run --timeout 5m --new-from-merge-base=origin/main` (cache isolé) | **0 issue** sur l'état final, après la scission du fichier `archlint` (555 L → 341 + 236). Cinq tentatives ont buté sur `Error: parallel golangci-lint is running` — le verrou de `golangci-lint` est global au poste et un agent voisin le tenait ; la sixième est passée |
| 2026-09-16 | revue M1 — L3 (mesure des trois compteurs câblés) | `804b643f1` | `go test ./…/replay/ -run TestReplisDuResumeDUsageSurLesHuitBuilds -v` | **8/8 builds, aucun octet de film lu** (fixtures d'assemblage) : `repli_geste_dernier_occupant_du_match` **0**, `repli_geste_premiere_vie_du_slot` **0**, `repli_garde_equipement_negatif_a_zero` **30** (présent sur 7 des 8 builds ; seul `bcb6d393` à 0). C'est la mesure que les cibles de retrait réclamaient depuis le 2026-09-13 et qu'aucun compteur ne pouvait rendre. |
| 2026-09-16 | revue M1 — L3 (mutations des trois compteurs, jouées et restaurées PAR NOM) | `804b643f1` | retrait d'un `Declenche` à la fois, puis `go test ./…/replay/ -run TestRepli…` | **3/3 rouges au bon endroit** : « repli_geste_dernier_occupant_du_match = 0, attendu 1 — le site de `usageOwners.at` ne compte pas » ; idem `atOrJustBefore` ; « le clamp de `deriveUsageKept` ne compte pas ». Restauration vérifiée par `diff` avec la copie de référence : **fichier identique**. |
| 2026-09-16 | revue M1 — L3 (ratchet neuf des cibles + sa mutation) | `804b643f1` | `go test ./internal/archlint/ -run TestAucuneCibleDeRepliNeNommeUnLotClos` ; mutation `CibleRetrait: "lot 1.9.13"` | **Vert après correction de 9 entrées** (les 8 du constat C3 plus `repli_distances_de_touche_desactivees`, que le ratchet a trouvée). Mutation : **rouge**, « CIBLE PERIMEE (1) … repli_geste_premiere_vie_du_slot .CibleRetrait : lot(s) coche(s) au plan : 1.9.13 ». Restaurée, vert. |
| 2026-09-16 | revue M1 — L3 (inventaire C4, greps collés) | `1f2e8e9f9` | `grep -rn "DetectI0Layout(" --include=*.go . \| grep -v "DetectI0LayoutOf("` ; idem `walkKeyframeBody\|keyframeBodyVariants` et `ScanFilmEquipmentSpawnEvents` | **48 appels de `DetectI0Layout`, tous dans des `_test.go`** (40 `filmdec` + 8 `replay`), 0 production ; **`walkKeyframeBody` : 5 instruments `_test.go` + sa déclaration**, 0 production ; **`ScanFilmEquipmentSpawnEvents` : 1 appel**, `replay/e191_origine_mesure_research_test.go:228`. |
| 2026-09-16 | revue M1 — L3 (`GrammarRev`) | `1f2e8e9f9` | `go test ./…/filmdec/ -run GrammarRevSuitLaGrammaire` puis `-update-grammar-rev` puis re-vérification | `.11` → **`.12`**, empreinte `e7bbe970…e97d1` (était `f856b325…a61c5`). **Faux positif d'empreinte assumé** : les trois formes de production (`DetectI0LayoutOf`, `ScanEquipmentSpawnEvents`, `WalkKeyframeFullState`) sont inchangées à l'octet. `KillSourceDecoderRev` et `SchemaVersion` inchangés. |
| 2026-09-16 | revue M1 — L3 (communs, sans décodage) | `1f2e8e9f9` | `gofmt -l ./internal ./cmd` ; `go vet` SANS tag et `-tags research` sur `film/…` ; `go test ./…/film/... ./internal/archlint/ ./internal/replaybuild/ ./internal/analysis/objectiveevents/ ./internal/service/replayview/` ; `go test ./internal/sync/killcollector/ ./internal/domain/replaydoc/` (CGO) | `gofmt` **vide** ; `go vet` **0 diagnostic** dans les deux régimes ; tous les paquets **ok** SAUF les deux rouges attendus par construction de la vague (`TestDocumentShapeMatchesGolden`, `TestContractFixturesMatchCommitted` — schéma figé). `killcollector` **ok 0,15 s**, `replaydoc` sans fichier de test. |
| 2026-09-16 | revue M1 — L3 (budget `filmdec` sans tag) | `1f2e8e9f9` | `CGO_ENABLED=0 go test ./…/filmdec/ -count=1` | **17,5 s** puis **21,8 s** en suite complète — sous le budget écrit de 30 s (§2.3). |
| 2026-09-16 | revue M1 — L3 (lint) | `1f2e8e9f9` | `make go-api-lint` (`GOLANGCI_LINT_CACHE` isolé) | **0 issues**, baseline non accrue. |
| 2026-09-16 | revue M1 — L3 (dette gelée : aucun fichier au-delà de 500 L n'est accru par ce lot) | `804b643f1` | `wc -l` avant / après sur `usage_summary.go` | `usage_summary.go` serait passé de **538 à 592 L** par le câblage ; le résolveur de slot sort dans `usage_summary_owners.go` par déplacement pur : **377 + 231**. Aucun autre fichier du lot ne franchit 500 L. |
| 2026-09-16 | revue M1 — L3 (ratchet des variables de paquet) | ce commit | `go test ./internal/archlint/ -run TestFilmdecPackageVarsNeCroitPas -v` | Le déplacement de `keyframeBodyVariants` hors production fait **descendre** l'état global de `filmdec` **96 → 94** ; le ratchet ne fait pas échouer une baisse, il l'ANNONCE — `filmdecVarsGeles` est resserré à 94, daté et justifié dans son en-tête. |
| 2026-09-16 | 1.9.10 (gates sans décodage) | `686b19aec` | `gofmt -l ./internal ./cmd` · `go vet` + `go vet -tags research` sur les 7 paquets du brief · `go test` sur `film/... archlint/ replaybuild/ objectiveevents/ replaydoc/ replayview/` · `make -C ../.. go-api-lint` | gofmt VIDE · vet VIDE (avec et sans tag) · **lint `0 issues.`** · `filmdec` **16,5 s** seul (base sans les tests du lot : 15,4 à 16,7 s ; budget 30 s tenu) · tous les paquets verts SAUF les deux que la règle de la vague rend inévitables : `TestDocumentShapeMatchesGolden` et `TestContractFixturesMatchCommitted` (8 builds) — la porte de régénération REFUSE explicitement sans montée de `SchemaVersion`, que le lot n'a pas le droit de faire. Mutation jouée et restaurée : l'appel `assignVehicleDeaths` retiré de `vehicleLives` rend `TestVehicleLivesPoseLaMortEcrite` rouge |
| 2026-09-16 | 1.9.10 (corpus gate, 14 témoins) | ce commit | `replay-corpus-gate --base=6db15a9bc --parc-root C:/…/LevelUp-go-migration --source-root C:/…/LevelUp-wt-decfilm-1910 --json <f> --work-root <f> --keep-work` (SANS `--manifest`), 28 cuissons | **14/14 témoins comparés, schéma 59 -> 59 partout, colonne `changements` À ZÉRO sur les quatorze.** 7 témoins `ok` 0/0/0 ; 7 `PERTE` dont AUCUNE n'est une régression : `084a804d` 10/5, `4f77afc1` 10/5, `e5adf7b2` 11/2, `111fa685` 10/2, `a349fea8` 10/2, `bfecd02b` 9/2, `60ae07c4` 4/1. Les « pertes » sont des compteurs qui BAISSENT — `coverage.fallbacks/n` (le repli retiré) et `vehicles/par-end/unknown` (97->78, 15->7, 53->46, 109->103, et **11->ZÉRO** sur `bfecd02b`) — plus DEUX épisodes d'occupation qui cessaient de survivre à leur véhicule (`084a804d` slot 793 : -184 frames, publié jusqu'à 22,5 s APRÈS la mort écrite ; `4f77afc1` slot 776 : -157 frames, 69,7 s après). Un premier passage (18:44-18:53) avait été coupé par une purge du cache à 7 témoins sur 14 ; son JSON, conservé, sert de contrôle de stabilité : les 7 témoins communs rendent des comptes IDENTIQUES |
| 2026-09-16 | 1.9.10 (acceptation `bfecd02b`) | ce commit | `LOT1910_ARTEFACT=<work>/…/bfecd02b.json go test …/replay/ -run Acceptation1910 -v -count=1`, sur l'artefact CUIT par le gate | **VERT. Le film ÉCRIT la destruction du ghost 777** : `end=destroyed`, `tEnd=2740` (274,0 s), `t1max` resserré de 2874 à 2821, 1 855 échantillons conservés. `unknown` tombe à **ZÉRO** sur ce témoin (1 `destroyed`, 10 `film_end`). **Contrôle de discrimination** : le même test est ROUGE sur l'artefact de la BASE (`end="unknown" — le film n ECRIT PAS la fin de cette vie`) |
| 2026-09-16 | 1.9.10 (équivalence — CLASSIFICATION AVANT TOUT `-update`) | ce commit | `replay-equiv -child -film <short8> -out <fichier> -repo-root C:/…/LevelUp-wt-decfilm-1910` pour chacun des 10 films, UN PAR PROCESSUS (binaire compilé une fois), puis `diff` de chaque `.tsv` contre sa référence | **10 films sur 10 : EXACTEMENT DEUX étapes différentes, `vehicles` et `artifact`** ; les **51 autres sont identiques à l'octet** sur les dix. Le harnais parent ne nomme que la PREMIÈRE ligne différente (`comparer`, `parent.go:250`) : sa sortie annonçait « une seule étape » et c'était faux — la classification a exigé les TSV complets. Cause : (1) `VehicleScan` observé gagne `Deaths` et `DeathStats` (la FORME de l'objet observé change, motif du lot 1.9.3 avec `Stats.Couples`) ; (2) le document publie `End` lu et huit compteurs. Delta `artifact` : **+141 EXACTEMENT** sur les 4 films sans entrée de repli, +79 à +113 sur les 6 autres où les octets de l'entrée retirée et des épisodes raccourcis se soustraient |
| 2026-09-16 | 1.9.10 (équivalence — re-figeage puis passe de comparaison) | ce commit | les 10 `.tsv` d'enfants recopiés sur leurs références (`-update` écrit EXACTEMENT `GrammarLine + les lignes de l'enfant`, `parent.go:173-180` — ligne de grammaire, nombre de lignes et fin de fichier vérifiés identiques avant copie), puis `replay-equiv -repo-root C:/…/LevelUp-wt-decfilm-1910 -films <les 10>` SANS `-update` | `git diff` des références : **10 fichiers, 2 lignes chacun, TOUTES `vehicles` ou `artifact` — 10 `+artifact` / 10 `+vehicles` / 10 `-artifact` / 10 `-vehicles` et RIEN d'autre**. Passe de comparaison : **BILAN 10 identique(s), 0 different(s), 0 ecarte(s), 0 echec(s), 0 illisible(s)** (durées 5,8 s à 2 min 50, pic 0,08 à 0,34 Gio) |

| `repli_ancre_sans_vie_delta_ecartee` | `games/halo_infinite/film/filmdec/equipment_creation_width.go` | non_resolu / apres_lecture | n/i | — (non câblé) | lot 3.x (largeurs de creation par build, la calibration disparait) |
| `repli_armement_bombe_debut_a_zero` | `games/halo_infinite/film/replay/bomb_armings.go` | non_resolu / apres_lecture | **câblé** | **0 sur 10** | lot de conversion de l'origine du rejeu (coverage.originResolved) |
| `repli_assistant_non_resolu_abandonne` | `replaybuild/kills.go` | non_resolu / apres_lecture | n/i | — (non câblé) | lot 1.8 porte (la table du film nomme les indices) |
| `repli_bande_bipede_comblee` | `games/halo_infinite/film/filmdec/offline_biped_band.go` | inconditionnel / apres_lecture | n/i | — (non câblé) | lot 3.5 (la bande de slots bipede par build) |
| `repli_bijection_hongroise_du_feed` | `games/halo_infinite/film/killsource/bijection.go` | section_absente / apres_lecture | n/i | — (non câblé) | cloture de M1 puis recuisson : la table du film (lot 1.8) doit couvrir 100 % des indices |
| `repli_calibration_paquet_exclu` | `games/halo_infinite/film/killsource/calibrate.go` | non_resolu / sans_lecture | n/i | — (non câblé) | lot 3.4 (les largeurs deviennent une donnee de profil, la calibration disparait) |
| `repli_calibration_paquet_non_localise` | `games/halo_infinite/film/killsource/calibrate.go` | non_resolu / sans_lecture | n/i | — (non câblé) | lot 3.4 |
| `repli_camp_inconnu_retire_de_la_table` | `replaybuild/matchfacts.go` | section_absente / apres_lecture | n/i | — (non câblé) | lot 1.7 (V4) porte aux zones — cf. D3 (1.7) : ZoneInput.TeamByXUID prend TOUJOURS l'equipe de la base |
| `repli_cap_vehicule_vitesse_insuffisante` | `games/halo_infinite/film/replay/vehicle_tracks.go` | film_muet / sans_lecture | n/i | — (non câblé) | aucune tant que le negatif tient (i2 REFUTE, i21 ABSENT de ti=40) ; le COMPTE des reports est ce qui manque |
| `repli_carte_premier_nom_resolu` | `sync/killcollector/map_identity.go` (SITE UNIQUE depuis le lot 1.9.4 : les positions ET les touches partagent la resolution) | non_resolu / apres_lecture | n/i | — (non câblé) | **NON RETIRE par 1.9.4** : le lot a bien fait passer le nom devant la signature, mais ce repli-ci porte une autre question — lequel des noms candidats décide quand la base en rend plusieurs. L'ordre est documenté « du plus fiable au moins fiable » (`replay_map_repo.go`) ; arbitrer demande d'instruire le REGISTRE DES MATCHS, hors famille 1.9 |
| `repli_catalogue_de_zones_absent` | `replaybuild/zones.go` | section_absente / apres_lecture | n/i | — (non câblé) | aucune (degradation gracieuse multi-titre, `ErrCapabilityNotSupported`) ; le COMPTE est ce qui manque |
| `repli_chaine_evenement_code_non_modelise` | `games/halo_infinite/film/killsource/eventbody.go<br>games/halo_infinite/film/killsource/eventchain.go` | lecture_non_portee / apres_lecture | n/i | — (non câblé) | lot 3.6 (les composants manquants, archetype par archetype) |
| `repli_chassis_vehicule_marqueur_neutre` | `games/halo_infinite/film/replay/vehicle_families.go` | film_muet / apres_lecture | n/i | — (non câblé) | lot 1.9.9 (les tourelles automatiques bannies nommees, et dessinees comme elements de carte) |
| `repli_chunk_de_replication_saute` | `games/halo_infinite/film/replay/player_index.go` | section_absente / apres_lecture | n/i | — (non câblé) | lot 1.6 (la table du film remplace cette voie) : le repli tombe quand la table de chunk_00 est le lien direct partout |
| `repli_chunk_du_pied_par_argmax` | `games/halo_infinite/film/killsource/feed.go` | non_resolu / devant_la_lecture | n/i | — (non câblé) | lot 1.9.8 (le chunk du pied pris au type du manifeste) |
| `repli_chunks_apres_trou_abandonnes` | `games/halo_infinite/film/filmdec/film_chunks.go` | non_resolu / apres_lecture | n/i | — (non câblé) | lot de conversion du manifeste (le manifeste dit quels chunks existent ; un trou est une donnee, pas une borne) |
| `repli_coequipier_hors_de_vue_par_defaut` | `games/halo_infinite/film/replay/death_context.go` | non_resolu / apres_lecture | n/i | — (non câblé) | lot de conversion du contexte d'isolement (hors famille 1.9 a ce jour) |
| `repli_coequipiers_partis_constante_nulle` | `sync/killcollector/isolation_facts.go` | inconditionnel / sans_lecture | n/i | — (non câblé) | retrait de la colonne (append-only, ADR 0026 : la colonne reste, c'est son ECRITURE qui doit devenir nulle explicite) |
| `repli_colline_dernier_intervalle_ouvert` | `games/halo_infinite/film/replay/zone_states_hill.go` | film_muet / sans_lecture | **câblé** | **0 sur 10** | aucune tant que le canal reste un ETAT sans emission de fin ; le COMPTE est ce qui manque |
| `repli_colline_votes_periode_entiere` | `games/halo_infinite/film/replay/zone_states_hill.go` | non_resolu / apres_lecture | **câblé** | **0 sur 10** | lot de conversion du calque des collines (hors famille 1.9 a ce jour) |
| `repli_composant_hors_table_parametre_un` | `games/halo_infinite/film/filmdec/traverse.go` | lecture_non_portee / apres_lecture | n/i | — (non câblé) | lot 3.6 (les composants manquants, archetype par archetype) |
| `repli_composants_statborg_arretes` | `analysis/objectiveevents/statborg.go` | lecture_non_portee / apres_lecture | n/i | — (non câblé) | lot 3.6 (les composants manquants, archetype par archetype) |
| `repli_crane_porteur_sans_vie_nommee` | `games/halo_infinite/film/replay/skull_carries.go` | non_resolu / apres_lecture | n/i | — (non câblé) | lot 1.9.5 (le porteur du crane lu au canal des armes tenues) |
| `repli_couple_recolle_sur_le_voisin` | `games/halo_infinite/film/killsource/feed_couples.go` | section_absente / apres_lecture (POSÉ au lot 1.9.3 le 2026-09-15) | n/i — le compte EXISTE sous `CoupleStats.Recolles` et sort en `killsource_couple_recolle` | **81 sur 281 kills sans mort en face, 21 films entiers** : `a349fea8` 31, `a521164d` 12, `4f77afc1` 10, `084a804d` 6, `c75f33b8` 5, `50247b26` 3, `11de8353` 3, `9b191a7f` 3, `e5adf7b2` 3, `111fa685` 2, `fccc61cd` 2, `bcb6d393` 1 | clôture M1 puis lot 3.6 ; critère : `CoupleStats.Recolles` à 0 sur les 8 builds et sur le corpus gate |
| `repli_deadstate_categorie_hors_enum` | `games/halo_infinite/film/killsource/walk.go` | non_resolu / apres_lecture | n/i | — (non câblé) | lot 3.x (enumeration par build) |
| `repli_deadstate_hors_bande_bipede` | `games/halo_infinite/film/killsource/walk.go` | non_resolu / apres_lecture | n/i | — (non câblé) | lot 3.5 (la bande de slots bipede par build) |
| `repli_deadstate_indice_hors_roster` | `games/halo_infinite/film/killsource/walk.go` | non_resolu / apres_lecture | n/i | — (non câblé) | lot 1.8 (le kill feed prend la table du film) porte jusqu'au rejet |
| `repli_debut_de_manche_au_minimum` | `analysis/objectiveevents/slotidentity_rounds.go` | non_resolu / apres_lecture | n/i | — (non câblé) | lot 1.9.11 (le designateur de manche lu tel que le film l'ecrit) |
| `repli_distances_de_touche_desactivees` | `sync/killcollector/hits.go` | section_absente / apres_lecture | n/i (TROIS compteurs expvar câblés par 1.9.4, un par cause) | — (non câblé au `fallback.Compteur` : la passe ne porte aucune cuisson) | **NON RETIRE par 1.9.4, et son critère « cartes jumelles » est TENU** : la carte vient du nom, plus d'une signature. Ce qui reste est la dégradation elle-même (carte hors catalogue, positions illisibles), légitime. Nouvelle cible : le lot qui rallume la précision par arme — tant que la passe est éteinte (`match.weapon.accuracy` = `not_exposed`), ses compteurs sont à zéro par construction et ne prouvent rien |
| `repli_drapeau_seul_en_jeu` | `games/halo_infinite/film/replay/flag_assign.go` | non_resolu / apres_lecture | n/i | — (non câblé) | lot 1.9.6 (le drapeau qui rentre pris dans ev.flag, deja nomme en amont) |
| `repli_emission_du_compteur_de_morts_jetee` | `analysis/objectiveevents/slotidentity_deaths.go` | non_resolu / apres_lecture | n/i | — (non câblé) | lot de conversion statborg |
| `repli_emission_hors_domaine_jetee` | `analysis/objectiveevents/named_series.go` | non_resolu / apres_lecture | n/i | — (non câblé) | lot de conversion des series nommees (hors famille 1.9 a ce jour) |
| `repli_enregistrement_statborg_abandonne` | `analysis/objectiveevents/statborg.go` | non_resolu / apres_lecture | n/i | — (non câblé) | volet statborg du registre des reports (toujours ouvert) ; hors famille 1.9 a ce jour |
| `repli_episode_occupation_par_trou_de_position` | `games/halo_infinite/film/replay/vehicle_rides.go` | film_muet / apres_lecture | n/i | — (non câblé) | lot 1.9.10 (la fin de vie d'un vehicule lue au dead-state ecrit) et le chantier vehicules |
| `repli_famille_arme_identifiant_brut` | `games/halo_infinite/film/replay/ground_weapon_rules.go` | film_muet / apres_lecture | n/i | — (non câblé) | aucune tant que le catalogue d'armes est incomplet ; retrait sec des que le compte est nul sur le parc |
| `repli_famille_objectif_vide` | `analysis/objectiveevents/extract.go` | non_resolu / apres_lecture | n/i | — (non câblé) | question NE13 de la table (D) : le classement par `strings.Contains` sur le nom de variante est hors doctrine multi-titre (skill `halo-modes`) |
| `repli_fin_de_vie_vehicule_par_recensement` | `games/halo_infinite/film/replay/vehicle_tracks.go` | film_muet / devant_la_lecture | **câblé** | 111fa685 9, 11de8353 11, 50247b26 13, 60ae07c4 2, a521164d 38, e5adf7b2 10 | lot 1.9.10 (la fin de vie d'un vehicule lue au dead-state ecrit) |
| `repli_fraicheur_des_derivations_par_taille` | `replaybuild/derivations_index.go` | film_muet / sans_lecture | n/i | — (non câblé) | lot 4.4 (la recuisson selective par couche) : une revision par calque remplace la taille |
| `repli_gamertag_par_xuid_brut` | `games/halo_infinite/film/killsource/feed.go` | section_absente / apres_lecture | n/i | — (non câblé) | lot 1.8 porte : la table du film nomme les joueurs a zero mort |
| `repli_gamertag_premier_xuid_gagne` | `replaybuild/kills.go` | contradiction / apres_lecture | n/i | — (non câblé) | lot 1.8 porte : un gamertag qui porte deux xuids est une contradiction a compter, pas a trancher |
| `repli_garde_equipement_negatif_a_zero` | `games/halo_infinite/film/replay/usage_summary_outcomes.go` | contradiction / apres_lecture | n/i | — (non câblé) | lot 1.9.13, puis retrait sec si le compte reste nul |
| `repli_geste_dernier_occupant_du_match` | `games/halo_infinite/film/replay/usage_summary.go` | non_resolu / apres_lecture | n/i | — (non câblé) | lot 1.9.13 (une vie finit a une mort ECRITE), qui borne les vies sur les morts du film |
| `repli_geste_premiere_vie_du_slot` | `games/halo_infinite/film/replay/usage_summary.go` | non_resolu / apres_lecture | n/i | — (non câblé) | lot 1.9.13 |
| `repli_homonymes_sans_xuid` | `sync/killcollector/roster.go` | contradiction / apres_lecture | n/i | — (non câblé) | lot 1.8 porte : la table du film donne l'index, pas le nom, donc l'homonymie cesse d'etre un obstacle |
| `repli_i0_porte_et_region_par_defaut` | `games/halo_infinite/film/filmdec/i0_layout.go` | **carte_absente_du_catalogue / apres_lecture** (rétrogradé au lot 1.9.2 le 2026-09-15 : les deux chemins de `killcollector` imposent le catalogue, la lecture du référentiel précède partout) | n/i | — (non câblé) | **MOITIÉ FAITE AU LOT 1.9.4** : `DetectI0Layout` n'a plus AUCUN appelant de production (l'identité de carte par signature est supprimée, ratchet = retrait de l'entrée d'allowlist `archlint/no_recomputed_film_context`). Restent les DEUX appelants de `DetectI0LayoutOf` (`FilmContext.I0Layout`, `offline_biped_band.bipedI0Layout`), atteints quand aucune entrée de catalogue n'est imposée -> **lot 3.x (profil par carte)** |
| `repli_identite_piste_meilleur_recouvrement` | `games/halo_infinite/film/replay/identity.go` | non_resolu / apres_lecture | **câblé** | **0 sur 10** | lot 1.9.13 (les vies se decoupent aux morts ecrites, donc ne se recouvrent plus) |
| `repli_identite_pont_par_morts` | `sync/killcollector/positions.go` | section_absente / apres_lecture | n/i | — (non câblé) | lot 1.6 (le registre d'identite prend la table du film comme lien direct) |
| `repli_identite_premier_occupant_du_siege` | `games/halo_infinite/film/replay/identity_registry_pont.go` | non_resolu / apres_lecture | n/i | — (non câblé) | lot 1.9.14 (le roster a l'instant T, c'est les occupants) et 1.9.13 |
| `repli_impulsion_fusionnee_dans_le_geste` | `games/halo_infinite/film/replay/document_ability_impulses.go` | film_muet / sans_lecture | n/i | — (non câblé) | aucune tant que le film ne borne pas un geste ; le COMPTE des fusions est ce qui manque |
| `repli_index_de_region_largeur_un` | `games/halo_infinite/film/filmdec/traverse.go` | non_resolu / apres_lecture | n/i | — (non câblé) | lot 3.4 |
| `repli_index_drapeau_zero_pour_tous` | `games/halo_infinite/film/replay/flag_assign.go` | section_absente / sans_lecture | n/i | — (non câblé) | lot de completion du catalogue de socles (hors famille 1.9) |
| `repli_indice_en_collision_jete` | `sync/killcollector/shots.go` | contradiction / apres_lecture | n/i | — (non câblé) | lot 1.8 porte aux tirs et aux touches (D3 (1.8) : cette voie sert `match_weapon_shots` et `match_weapon_accuracy`) |
| `repli_instant_sur_la_premiere_manche` | `analysis/objectiveevents/slotidentity_rounds.go` | non_resolu / apres_lecture | n/i | — (non câblé) | lot 1.9.11 |
| `repli_invariant_propre_drapeau_muet` | `games/halo_infinite/film/replay/flag_assign.go` | non_resolu / apres_lecture | n/i | — (non câblé) | lot 1.7 (l'equipe vient du film) : l'equipe du porteur est desormais lue, donc le silence doit disparaitre |
| `repli_largeur_absolue_uniforme` | `games/halo_infinite/film/filmdec/position_capture.go` | inconditionnel / devant_la_lecture | n/i | — (non câblé) | lot 3.4 (largeurs par carte et par build) |
| `repli_largeurs_axe_par_defaut_conservees` | `games/halo_infinite/film/replay/world_object_precision.go` | section_absente / apres_lecture | **câblé** | **0 sur 10** | lot 3.4 (largeurs par carte et par build, donnees de profil) |
| `repli_largeurs_monde_par_defaut_conservees` | `games/halo_infinite/film/filmdec/traverse.go` | non_resolu / apres_lecture | n/i | — (non câblé) | lot 3.4 (les largeurs sont une donnee de la carte et du build) |
| `repli_largeurs_mpp_par_defaut` | `games/halo_infinite/film/filmdec/default_state.go` | inconditionnel / apres_lecture | n/i | — (non câblé) | lot 3.x (profil par build : les largeurs MPP sont une donnee du build) |
| `repli_libelle_de_source_autres` | `games/halo_infinite/film/killsource/label.go` | section_absente / apres_lecture | n/i | — (non câblé) | completion du catalogue de sources (206 tags sur 468 concernes, audit 0.E) |
| `repli_lien_prise_arme_abandonne` | `games/halo_infinite/film/replay/document_ground_weapon_items.go` | non_resolu / apres_lecture | n/i | — (non câblé) | lot de conversion du lien de prise (table C13 : trois hypotheses de lien natif REFUTEES — le repli restera, son COMPTE PAR CAUSE est ce qui manque) |
| `repli_localisation_largeur_libre` | `games/halo_infinite/film/killsource/walk.go` | non_resolu / apres_lecture | n/i | — (non câblé) | lot 3.4 (largeurs calibrees par la carte et le build) |
| `repli_manche_du_slot_sautee` | `analysis/objectiveevents/named_series.go` | non_resolu / apres_lecture | n/i | — (non câblé) | question NE17 de la table (D) de l'audit instruite (`longestRun` ecarte-t-il des points reels ?) |
| `repli_manches_contigues_decretees` | `analysis/objectiveevents/statborg.go` | non_resolu / devant_la_lecture | n/i | — (non câblé) | lot 1.9.11 (le designateur de manche lu tel que le film l'ecrit) |
| `repli_mort_de_bot_premier_candidat` | `games/halo_infinite/film/killsource/match.go` | non_resolu / apres_lecture | n/i | — (non câblé) | lot 1.9.7 |
| `repli_mort_ecartee_hors_equipe_de_base` | `games/halo_infinite/film/replay/death_context.go` | section_absente / apres_lecture | n/i | — (non câblé) | lot 1.7 (l'equipe vient du film, V4) porte jusqu'a ce calque |
| `repli_mort_neutre_sans_xuid_abandonnee` | `replaybuild/replaybuild.go` | non_resolu / apres_lecture | n/i | — (non câblé) | lot 1.8 porte jusqu'au constructeur |
| `repli_mort_non_revendiquee_la_plus_proche` | `games/halo_infinite/film/killsource/hybrid.go` | non_resolu / apres_lecture | n/i | — (non câblé) | lot 1.9.7 (dead-state et kill-feed apparies par l'identite de paquet) |
| `repli_mort_sans_xuid_ignoree` | `analysis/objectiveevents/slotidentity_deaths.go` | non_resolu / apres_lecture | n/i | — (non câblé) | lot 1.8 porte jusqu'a ce calque |
| `repli_nom_piste_par_le_pont` | `games/halo_infinite/film/replay/published_tracks.go` | non_resolu / apres_lecture | n/i | — (non câblé) | lot 1.6 (le registre d'identite prend la table du film comme lien direct) : le compte doit tomber avec la couverture du lien direct |
| ~~`repli_origine_pose_fenetre_temporelle`~~ | **SORTIE DU REGISTRE le 2026-09-15 (lot 1.9.1)** : la fenetre de 200 ms ne decide plus aucune origine de pose (decision utilisateur « lache »). Son code a disparu, son entree avec lui (D14 d). Elle survit comme TEMOIN de mesure (`f1OrigineParFenetre`, fichier de recherche). | — | — | — | sans objet |
| ~~`repli_origine_pose_vie_la_plus_proche`~~ | **SORTIE DU REGISTRE le 2026-09-15 (lot 1.9.1)** : elle n'existait que pour choisir la vie que la fenetre confrontait. Avant le lot : 111fa685 493, 11de8353 379, 50247b26 307, 51101d1d 41, 60ae07c4 184, a521164d 212, bcb6d393 92, d9781168 275, e5adf7b2 506, fb1a1a72 319. | — | — | — | sans objet |
| `repli_piece_engendree_sans_evenement` | `games/halo_infinite/film/replay/equipment_origin.go` | film_muet / apres_lecture | **cable** | **a521164d 2, 50247b26 7** — et ZERO partout ailleurs sur les 13 films mesures (pose au lot 1.9.1, 2026-09-15) | lot 3.x (profil par build) : la liste d'evenements des builds anciens ; critere : 0 declenchement sur les 8 builds |
| `repli_participant_sans_xuid_retire` | `replaybuild/matchfacts.go` | section_absente / apres_lecture | n/i | — (non câblé) | lot 1.6.3 (la cuisson HORS LIGNE publie un roster complet, sans base) porte au tableau |
| `repli_piste_drapeau_sans_pont_ecartee` | `games/halo_infinite/film/replay/flag_carrier_tracks.go` | non_resolu / apres_lecture | **câblé** | bcb6d393 4 | lot 1.6 (lien direct par la table du film) |
| `repli_plafond_grenade_par_defaut` | `games/halo_infinite/film/replay/inventory_decode.go` | inconditionnel / sans_lecture | **câblé** | 111fa685 1, 11de8353 1, 50247b26 1, 51101d1d 1, 60ae07c4 1, a521164d 1, bcb6d393 1, d9781168 1, e5adf7b2 1, fb1a1a72 1 | lot 3.x (profil par build et par carte) : un plafond est une donnee de mode, pas une constante |
| `repli_portage_ferme_a_la_prise_suivante` | `games/halo_infinite/film/replay/held_object_carry.go` | film_muet / apres_lecture | n/i | — (non câblé) | lot 1.9.5 |
| `repli_porteur_anonyme_sans_fin_par_mort` | `games/halo_infinite/film/replay/held_object_carry.go` | non_resolu / apres_lecture | n/i | — (non câblé) | lot 1.9.5 (le porteur du crane lu au canal des armes tenues) et le registre d'identite 1.6 |
| `repli_position_lacher_prend_la_prise` | `games/halo_infinite/film/replay/flag_carries.go` | non_resolu / apres_lecture | **câblé** | fb1a1a72 2 | lot 1.9.13 (les vies couvrent alors la fin du portage) |
| `repli_precision_par_arme_passe_sautee` | `sync/killcollector/hits.go` | section_absente / sans_lecture | n/i | — (non câblé) | aucune (configuration d'exploitation) ; le COMPTE est ce qui manque |
| `repli_premiere_occurrence_sans_concordance` | `sync/killcollector/shots.go` | non_resolu / apres_lecture | n/i | — (non câblé) | lot 1.8 porte aux tirs et aux touches |
| `repli_rang_capacite_vie_elargie` | `games/halo_infinite/film/replay/document_ability_impulses.go` | non_resolu / apres_lecture | n/i | — (non câblé) | lot 1.9.13 (les vies cessent d'etre coupees par un trou de replication, donc les bords disparaissent) |
| `repli_record_desynchronise_jete` | `games/halo_infinite/film/killsource/walk.go` | non_resolu / apres_lecture | n/i | — (non câblé) | lot 3.x (registre ECS par build) : une desynchronisation est une grammaire fausse, pas une donnee |
| `repli_registre_inconnu_sans_lecteur_de_troncature` | `games/halo_infinite/film/filmdec/registry_fingerprint.go` | non_resolu / apres_lecture | n/i | — (non câblé) | lot 3.1 (build inconnu actif, profil comme donnee fabriquee) : un build inconnu devient une erreur typee, une troncature en est une autre |
| `repli_relais_de_bot_abandonne` | `replaybuild/replaybuild.go` | non_resolu / apres_lecture | n/i | — (non câblé) | lot 1.9.14 (le remplacant prend le siege du partant) |
| `repli_repere_neutre_generique_conserve` | `replaybuild/replaybuild.go` | section_absente / apres_lecture | n/i | — (non câblé) | completion de la table d'assets de morts neutres |
| `repli_roster_indice_hors_bijection` | `games/halo_infinite/film/killsource/roster.go` | non_resolu / apres_lecture | n/i | — (non câblé) | lot 1.8 porte jusqu'a l'ecriture en base |
| `repli_roster_nom_invente` | `games/halo_infinite/film/killsource/roster.go` | non_resolu / apres_lecture | n/i | — (non câblé) | lot 1.8 (la table du film donne le lien indice -> joueur) : un nom fabrique ne doit plus servir |
| `repli_slot_abandonne_au_premier_arrive` | `analysis/objectiveevents/slotidentity_rounds.go` | non_resolu / apres_lecture | n/i | — (non câblé) | lot 1.6 porte a ce calque |
| `repli_sonde_non_lancee_porte_relachee` | `games/halo_infinite/film/killsource/decode.go` | non_resolu / apres_lecture | n/i | — (non câblé) | lot de cloture du diagnostic killsource (hors famille 1.9) |
| `repli_table_identite_vide` | `analysis/objectiveevents/slotidentity_deaths.go` | section_absente / apres_lecture | n/i | — (non câblé) | lot 1.6 porte a ce calque (la table du film donne le lien direct) |
| `repli_traction_vie_du_tir` | `games/halo_infinite/film/replay/grapple_lines.go` | non_resolu / apres_lecture | n/i | — (non câblé) | lot 1.9.13 (les trous de replication cessent de couper les vies) |
| `repli_traction_vie_la_plus_proche` | `games/halo_infinite/film/replay/grapple_lines.go` | non_resolu / apres_lecture | n/i | — (non câblé) | lot 1.9.13 |
| `repli_type_de_chunk_perdu_du_manifeste` | `games/halo_infinite/film/killsource/chunks.go` | inconditionnel / devant_la_lecture | n/i | — (non câblé) | lot 1.9.8 (cause racine de l'argmax du pied) |
| `repli_vie_coupee_au_trou_de_replication` | **DEUX sites** : `…/replay/lives_decoupe.go` et `…/replay/tracks_publication.go` (inscrit à la revue M1) | site 1 film_muet / site 2 **section_absente** — **apres_lecture** (converti au lot 1.9.13) | **câblé** (site 2 : `DeclencheN`, une unité par COUPURE) | **0 sur 8 builds** | lot M2 : retrait sec si le compte reste nul au corpus gate (D14 d) |
| `repli_xuid_vide_pour_nom_inconnu` | `sync/killcollector/identities.go` | non_resolu / apres_lecture | n/i | — (non câblé) | lot 1.8 porte jusqu'a l'ecriture (la table du film nomme les joueurs a zero mort) |
| `repli_zone_camp_sans_roster` | `games/halo_infinite/film/replay/zone_states_owner.go` | section_absente / apres_lecture | n/i | — (non câblé) | lot 1.7 (l'equipe vient du film) porte au calque des zones — cf. D3 (1.7), ZoneInput.TeamByXUID prend TOUJOURS l'equipe de la base |
| `repli_zone_proprietaire_sans_roster` | `games/halo_infinite/film/replay/zone_states_owner.go` | section_absente / apres_lecture | n/i | — (non câblé) | meme cible que repli_zone_camp_sans_roster |


### Revue de jalon M1 — lentille L4 (correction des donnees), 2026-09-16

Branche `feat/decfilm-rev1-l4`, base `34fa53da5`. Les DEUX seuls rouges sont ceux que la regle de
la vague rend mecaniques (D4 (1.9.13) en §4) : `TestDocumentShapeMatchesGolden` et
`TestContractFixturesMatchCommitted`, deja rouges sur la base par construction.

| Date | Lot | Commit | Commande | Résultat (compte, empreinte, durée) |
|---|---|---|---|---|
| 2026-09-16 | Revue M1 — L4 | `(ce commit)` | `gofmt -l ./internal ./cmd` | sortie vide |
| 2026-09-16 | Revue M1 — L4 | `(ce commit)` | `go vet` (film/… + archlint + replaybuild + killcollector + objectiveevents + replaydoc + replayview + games/canonical + cmd/killsource), CGO_ENABLED=1 | 0 diagnostic |
| 2026-09-16 | Revue M1 — L4 | `(ce commit)` | `go vet -tags research` (mêmes paquets) | 0 diagnostic |
| 2026-09-16 | Revue M1 — L4 | `(ce commit)` | `go test` (mêmes paquets, `-count=1`) | ok partout SAUF les deux rouges de la vague. damagetag 0,71 s · filmcache 0,65 s · filmdec 31,58 s · killicon 0,74 s · killsource 3,01 s · medalname 0,45 s · replay **FAIL** (`TestContractFixturesMatchCommitted`, `TestDocumentShapeMatchesGolden` — et eux seuls) · fallback 0,69 s · mapvar 1,32 s · archlint 63,18 s · replaybuild 7,88 s · killcollector 1,64 s · objectiveevents 12,57 s · replaydoc « no test files » · replayview 0,59 s · canonical 0,34 s |
| 2026-09-16 | Revue M1 — L4 | `(ce commit)` | `go test -tags=integration -p 1 ./internal/sync/killcollector/ -count=1` (obligatoire : `internal/sync/` bouge) | ok — 27,99 s |
| 2026-09-16 | Revue M1 — L4 | `(ce commit)` | `make go-api-lint` | **0 issues** (baseline non accrue) |
| 2026-09-16 | Revue M1 — L4 | `(ce commit)` | `go test ./internal/games/halo_infinite/film/filmdec/ -count=1` (CGO_ENABLED=0, budget 30 s) | **20,76 s / 21,43 s / 23,89 s** sur trois passes consécutives — budget tenu. Deux passes antérieures à 31,58 s et 35,91 s ont été mesurées machine chargée (six exécuteurs en vol), elles sont citées pour ne rien taire |
| 2026-09-16 | Revue M1 — L4 | `7e08b416d` | `go test ./internal/sync/killcollector/ -run TestKillSourceDecoderRev -count=1` (après `-update`) | ok — golden `killsource-2026-09-16` / `1701d5fa95820c73626ef5c754451cf3a48e115b9c7143d889ddfd69e9f893ad` |
| 2026-09-16 | Revue M1 — L4 | `(ce commit)` | `go test …/filmdec/ -run TestGrammarRevSuitLaGrammaire -count=1` (après `-update-grammar-rev`) | ok — golden `grammar-2026-09-15.12` / `4a41ab873b187ac3083156b6bbc28ce7e63f94af36c8970741a8577aeccca0d0` (162 fichiers) |
| 2026-09-16 | Revue M1 — L4 | `7e08b416d` | mutation constat 1 — `AffectationUnique` : `case 1: return true` | **ROUGE** : `TestUnIndiceLibrePourDeuxNomsLibresNEstPasDETERMINE/deux_noms_libres_pour_un_indice_libre` (affectation unique = true ; publication ligne par ligne = true) + `TestAffectationUniqueNeRegardePasQueLesIndices/1_indices_2_noms` et `/1_indices_9_noms`. Restaurée par nom, vert |
| 2026-09-16 | Revue M1 — L4 | `cad29309c` | mutation constat 2 — retrait de `if p.slotsAmbigus[slot]` dans `equipeDuSlot` | **ROUGE** : `TestEquipeDuSlotSAbstientSurUnSlotAmbigu` (equipe=0 lue=true) + `TestVieDuSecondOccupantNeRecoitPasLEquipeDuPremier` (comptes 2/2/0 au lieu de 2/1/1). Restaurée par nom, vert |
| 2026-09-16 | Revue M1 — L4 | `644a524d4` | mutation constat 3 — retrait de l'appel à `retirerLesIndexEnCollision` | **ROUGE** : `TestCompositionRetireUnIndexQueDeuxXUIDSeDisputent`, 5 assertions dont le sous-test `determinisme` (`map[…11:7 …22:7 …33:2]`). Restaurée par nom, vert |
| 2026-09-16 | Revue M1 — L4 | oracle (lecture seule) | `go run ./cmd/diag_q data/backups/pre-chaine-2026-09-09/shared_matches_v2.duckdb "SELECT table_name, column_name FROM information_schema.columns WHERE lower(column_name) LIKE '%bijection%' OR '%inferred%' OR '%roster%' OR '%determin%' OR '%margin%' OR '%health%'"` | **(0 rows)** — aucune colonne ne porte la mesure du constat 1 |
| 2026-09-16 | Revue M1 — L4 | oracle (lecture seule) | `… "SELECT decoder_rev, publishable, count(DISTINCT match_id), count(*) FROM match_kill_events_latest GROUP BY 1,2"` | `highlight-credit-2026-08-01/true` 3 films · `killsource-2026-07-31` 252 false / 337 true · `killsource-2026-09-05` 244 false / 548 true. **La révision la plus récente de l'oracle précède l'apparition de `BijectionDetermined` (2026-09-14)** : la population du constat 1 s'y mesurera à la recuisson, pas ici |

## 6. Protocole de reprise de session

1. Relire le skill `plan-execution`, puis la §5 et la première case non statuée de la §3.
2. `git worktree list` ; `git -C LevelUp-wt-recherche-film log --oneline -5` ; vérifier que
   l'intégration porte le dernier lot fusionné (§5).
3. Ne pas re-décider une décision D ou V validée ; une question neuve = ligne en §1.4, posée à
   l'utilisateur, jamais tranchée en silence.
4. Reprendre au lot courant : exécuteur relancé avec le brief du lot (état réel constaté sur
   pièces, pas le plan de mémoire).
| 2026-09-16 | Revue M1 — ronde 2 (F1, mutation AVANT) | ce commit | tranche `{"killsource/carte", registreKillsourceCarte}` retiree de `Tranches()`, puis `go test ./internal/games/halo_infinite/film/replay/fallback/ ./internal/archlint/ -count=1` | **TOUT VERT** : `ok fallback 0.488s` · `ok archlint 58.539s`. Le chainon `total != len(Table())` est une tautologie (`registre = concat(Tranches())`), `len(tranches) < 2` ne mord pas sur 6 -> 5, `plancherEntrees = 60` ne mord pas sur 96 -> 94 |
| 2026-09-16 | Revue M1 — ronde 2 (F1, mesure) | ce commit | `Tranches()` imprime famille par famille (test jetable, retire) | `replay/equipement 14` · `replay/identites 22` · `killsource 26` · `killsource/carte 2` · `objectifs et construction 21` · `filmdec 11` — **TOTAL 6 tranches, 96 entrees (Table=96)**. `plancherTranches` fixe a 6 |
| 2026-09-16 | Revue M1 — ronde 2 (F1, mutations APRES) | ce commit | (a) tranche retiree ; (b) tranche retiree **et** `plancherTranches` baisse a 5 | (a) **ROUGE** : `Tranches() ne rend que 5 familles (plancher 6) — un fichier de tranche a quitte l assemblage` ; (b) **ROUGE quand meme** : `famille de rang 3 : "objectifs et construction", attendue "killsource/carte"`, `famille de rang 4 : "filmdec", attendue "objectifs et construction"`, `la famille "killsource/carte" a QUITTE Tranches()`. Restaurees par nom. CONTRE-EPREUVE (entree renommee dans une tranche) : `ok 0.415s` — le test ne compte aucune entree |
| 2026-09-16 | Revue M1 — ronde 2 (F2, mesure AVANT) | ce commit | `grep -rn "Fallbacks" --include=*.go internal/ cmd/` filtre des `_test.go` | **AUCUN lecteur de `Match.Fallbacks` en production.** `replayartifacts/usage.go:161` lit `Match.EquipmentChanges`, jamais les replis ; `cmd_backfill_usage_summary.go`, `cmd/replay-corpus-gate`, `cmd/replay-equiv` : rien. Le champ etait ecrit (`usage_summary.go:230`) et jete |
| 2026-09-16 | Revue M1 — ronde 2 (F2, mutation) | ce commit | ligne `journaliserReplisUsage(ctx, d, prets)` retiree de `persisterResumesUsage`, puis `go test ./internal/sync/replayartifacts/ -run Replis -count=1` | **ROUGE** : `le producteur post-sync n'ecrit AUCUNE ligne de passe : le canal Fallbacks est de nouveau sans lecteur (constat F2)`. Restauree par nom, `ok 0.096s` |
| 2026-09-16 | Revue M1 — ronde 2 (F3, mutation) | ce commit | `t.Inferred == 1 && t.FreeNames >= 2` remis a `t.Inferred > 0 && t.FreeNames > t.Inferred`, puis `go test ./internal/sync/killcollector/ -run Ambiguite -count=1` | **ROUGE sur le SEUL regime concerne** : `2_indices_libres_/_3_noms_libres` -> `killsource_bijection_noms_libres_en_trop : delta 1, attendu 0`. Les deux autres regimes (0 nom libre en trop, 1 indice / 2 noms) restent verts. Restauree par nom |
| 2026-09-16 | Revue M1 — ronde 2 (F4, deplacement pur) | ce commit | `wc -l` avant / apres, `diff` ligne a ligne des extraits, `go test ./internal/sync/killcollector/ -run '^TestKillSourceDecoderRevSuitLeDecodeur$' -count=1` | `collector.go` **816 -> 718** ; `killsource_decoder_rev.go` **121**. Les 98 lignes retirees (hors 7 lignes F3) sont EXACTEMENT les 98 ajoutees (`diff` vide hors la ligne blanche de separation). Ratchet d empreinte **`ok 0.01s` SANS regeneration**, `testdata/` non modifie |
| 2026-09-16 | Revue M1 — ronde 2 (F5, porte de regeneration) | ce commit | `go test ./…/filmdec/ -run '^TestGrammarRevSuitLaGrammaire$' -update-grammar-rev -count=1`, md5 du golden avant / apres | **La porte REFUSE apres reecriture** (`1 reference(s) reecrite(s) … relancer sans -update-grammar-rev`) et le golden est **byte-identique** : md5 `aa1c189acef9636a1c9cb7b678f3d802` des deux cotes, ligne de donnees `grammar-2026-09-15.14  a9623c7e4a05bb180c3f6ef3c33a6e4ea6932695a0826f8884d979394b3affee` inchangee. **`GrammarRev` ne bouge pas** |
| 2026-09-16 | Revue M1 — ronde 2 (F5, mutation) | ce commit | (a) `GrammarRev` montee a `.15` sans entree ; (b) la meme, **golden regenere** par `-update-grammar-rev` (ce qui tait le ratchet d empreinte) | (a) **DEUX rouges** : `LA REVISION A CHANGE SANS QUE LA GRAMMAIRE BOUGE` et `TestChroniqueCouvreLaRevisionCourante`. (b) **le ratchet d empreinte se TAIT, la chronique reste ROUGE des DEUX cotes** : `GrammarRev = grammar-2026-09-15.15 n'a AUCUNE entree dans le godoc de grammar_rev.go (entrees declarees : [.12 .13 .14])` et `… dans l'HISTORIQUE du golden`. C est le trou exact du constat F5. Restauree par nom (md5 du golden identique) |
| 2026-09-16 | Revue M1 — ronde 2 (gate) | ce commit | `gofmt -l ./internal ./cmd` | sortie vide |
| 2026-09-16 | Revue M1 — ronde 2 (gate) | ce commit | `go vet` sans tag (9 paquets : `film/...`, `archlint`, `replaybuild`, `killcollector`, `replayartifacts`, `objectiveevents`, `replaydoc`, `replayview`, `cmd/levelup`) puis `go vet -tags research ./internal/games/halo_infinite/film/...` | **0 diagnostic** dans les deux cas (exit 0) |
| 2026-09-16 | Revue M1 — ronde 2 (gate) | ce commit | `go test` sur les memes paquets, `-count=1` | **verts, aux DEUX goldens rouges par construction pres** : filmdec 21,78 s · killsource 2,12 s · replay 21,40 s (`TestDocumentShapeMatchesGolden`, `TestContractFixturesMatchCommitted` — le lot ne touche AUCUN champ publie) · fallback 0,24 s · archlint 15,26 s · replaybuild 1,25 s · killcollector 0,16 s · replayartifacts 0,69 s · objectiveevents 0,70 s · replayview 0,40 s · cmd/levelup 0,86 s |
| 2026-09-16 | Revue M1 — ronde 2 (gate) | ce commit | `go test -tags=integration -p 1 ./internal/sync/killcollector/ ./internal/sync/replayartifacts/ -count=1` (`internal/sync/` touche) | **`ok killcollector 42.990s` · `ok replayartifacts 113.293s`** |
| 2026-09-16 | Revue M1 — ronde 2 (gate) | ce commit | `CGO_ENABLED=0 go test ./internal/games/halo_infinite/film/filmdec/ -count=1` (budget 30 s), quatre executions | **21,1 s / 22,1 s** en regime etabli — budget TENU. Les deux premieres (41,3 s et 36,5 s) sont de la contention : cinq autres executeurs travaillaient sur la machine |
| 2026-09-16 | Revue M1 — ronde 2 (gate) | ce commit | `make go-api-lint` (`golangci-lint run --timeout 5m --new-from-merge-base=origin/main`, cache isole) | **0 issues** |
| 2026-09-16 | 2.7 (grammaire) | (mesure AVANT) | `wc -l` des six cibles a la base `f950b7179` | `traverse.go` **1 388** · `unit_weaponstate.go` **956** · `frame_records.go` **793** · `components_biped_ability.go` **699** · `components_movement.go` **554** · `killcollector/collector.go` **718** |
| 2026-09-16 | 2.7 (grammaire) | (mesure AVANT) | releve des fonctions > 80 L des six cibles (parcours `func` vers `}`, colle) | `consumeByName` **815** · `decodeInferLoop` **114** · `consumeObjectPositionDynamicPrecisionD` **111** · `(c *KillSourceCollector) collect` **91** |
| 2026-09-16 | 2.7 (grammaire) | `9547d7a4e` | CONTROLE DE DEPLACEMENT PUR : `comm` sur le multi-ensemble TRIE des lignes de chaque fichier d'origine contre celui de ses fichiers d'arrivee | **ZERO ligne perdue** sur les cinq fichiers de `filmdec`. Les seules lignes ajoutees sont les en-tetes de fichier, les six signatures de maillon, `variant = noVariant` / `switch name {` / `default:` / `}` et les six `return` de chainage. Controle rejoue a l'identique sur `collector.go` (`be0f51504`) : zero ligne perdue, seules lignes neuves les quatre en-tetes et les blocs d'import redistribues |
| 2026-09-16 | 2.7 (grammaire) | `9547d7a4e` | `wc -l` APRES (filmdec) | `traverse.go` **390** · `dispatch_object.go` 163 · `dispatch_item.go` 149 · `dispatch_player.go` 362 · `dispatch_biped.go` 275 · `component_param4.go` 95 · `traverse_precision.go` 100 · `unit_weaponstate.go` **486** · `unit_control.go` 378 · `bit_leaf_readers.go` 118 · `frame_records.go` **278** · `frame_harvest.go` 272 · `frame_infer.go` 261 · `components_biped_ability.go` **343** · `components_biped_spartan.go` 366 · `components_movement.go` **196** · `components_position_i0.go` 393 |
| 2026-09-16 | 2.7 (grammaire) | `be0f51504` | `wc -l` APRES (killcollector) | `collector.go` **211** · `collector_run.go` 288 · `collector_batch.go` 116 · `collector_metrics.go` 175 |
| 2026-09-16 | 2.7 (grammaire) | ce commit | fonctions > 80 L apres le lot, dans les fichiers touches | `consumeCrewFlockAndMusicComponent` 146 · `consumeManagedAndObjectiveComponent` 143 · `consumeItemAndTacmapComponent` 139 · `consumePlayerTailAndGameEngineComponent` 122 · `consumeByName` 118 · `consumeCaptureAndBipedComponent` 112 — **les six maillons de la chaine de dispatch, exemption ecrite en tete de `dispatch_object.go`** (une table, un arm par composant ECS) ; `decodeInferLoop` 114, exemption ecrite en tete (boucle a sorties multiples). Aucune autre |
| 2026-09-16 | 2.7 (grammaire) | `9547d7a4e` | `go test ./…/filmdec/ -run TestG1TableSuitLeCode` apres extension du scanner a la chaine de `default` | **vert** — le garde-rail suit desormais la chaine au lieu de ne lire que `consumeByName`, et refuse un composant traite deux fois (un maillon ajoute est vu par construction) |
| 2026-09-16 | 2.7 (grammaire) | ce commit | MUTATION 1 (ratchet de taille) : `+1` ligne dans `internal/replaybuild/replaybuild.go` (577 vers 579) | **ROUGE** : « internal/replaybuild/replaybuild.go : 579 lignes, plafond fige a 577 (2026-09-16, lot 2.7) ». Fichier **restaure**, test **vert** |
| 2026-09-16 | 2.7 (grammaire) | ce commit | MUTATION 2 (ratchet de taille) : entree `internal/replaybuild/fichier_disparu.go` ajoutee a la table | **ROUGE** : « est au plafond mais n existe plus (renomme, deplace ou supprime) ». Entree **retiree**, test **vert** |
| 2026-09-16 | 2.7 (grammaire) | ce commit | `gofmt -l ./internal ./cmd` | sortie vide |
| 2026-09-16 | 2.7 (grammaire) | ce commit | `go vet ./internal/games/halo_infinite/film/... ./internal/archlint/ ./internal/replaybuild/ ./internal/sync/killcollector/ ./internal/analysis/objectiveevents/ ./internal/domain/replaydoc/ ./internal/service/replayview/` (CGO) | 0 diagnostic |
| 2026-09-16 | 2.7 (grammaire) | ce commit | `go test` sur les memes paquets, `-count=1` | **verts, aux DEUX goldens rouges par construction pres** (`TestDocumentShapeMatchesGolden`, `TestContractFixturesMatchCommitted` — message identique a celui releve sur la base AVANT tout changement) : filmdec 20,71 s · killsource 1,67 s · replay 20,00 s · archlint 15,71 s · replaybuild 1,40 s · killcollector 0,16 s · objectiveevents 0,65 s · replayview 0,39 s |
| 2026-09-16 | 2.7 (grammaire) | `be0f51504` | `go test -tags=integration -p 1 ./internal/sync/killcollector/ -count=1` (`internal/sync/` touche) | **ok, 37,298 s** |
| 2026-09-16 | 2.7 (grammaire) | ce commit | `CGO_ENABLED=0 go test ./…/filmdec/ -count=1` (budget 30 s) | **20,7 s** — budget tenu (26,2 s sur la base AVANT le lot, meme machine) |
| 2026-09-16 | 2.7 (grammaire) | `be0f51504` | `golangci-lint run --new-from-rev=f950b7179` (ce que le LOT introduit) | **0 issues**. Contre `origin/main` (`make go-api-lint`) il reste **2 constats `unparam` LATENTS** du paquet, absents contre la base du lot : §4, D3 |
| 2026-09-16 | 2.7 (grammaire) | ce commit | `benchstat testdata/bench_baseline.txt apres.txt` (`-count 10`) | `BitReaderReadBits` **~** (p=0,123) · `TraverseEntity` **+3,36 %** (p=0,005) · `KeyframeClosure` +5,54 % (informatif). **Budget de +10 % tenu.** Paire AVANT / APRES de la meme session : **aucune difference significative** sur les trois (p = 0,393 / 0,143 / 0,436), geomoyenne -1,42 % — l'ecart contre la ligne de base figee est la charge machine |
