# Plan D2 — Découplage de la brique « décodeur de film » — 2026-10-28

Source : `.ai/AUDIT_DECOUPLAGE_BRIQUES_2026-10-27.md` (axe D2, garde-fou G, décisions 1 à 3 de
l'utilisateur, addendum du 2026-10-28, section « D1 exécuté ») et les découvertes 9, 18 à 38 du
plan D1 (`.ai/V8/PLAN_DECOUPLAGE_BRIQUES_D1_2026-10-28.md`). Exécution sous le contrat du skill
`plan-execution` : ordre strict, un lot à la fois, gate passé avant le lot suivant, chaque item
statué `[x]` / `[~]` (référence) / `[!]` (justification écrite), zéro fix hors périmètre (section
« Découvertes » en fin de fichier). Le modèle d'infrastructure multi-module est celui du plan D1
(ADR 0041) : il se réutilise tel quel, il ne se rediscute pas.

## Objectif et critère de succès

Un troisième module Go sous `libs/` :

| Module | Paquets au premier niveau | Fermés (`internal/`) | Importe |
|---|---|---|---|
| `libs/film` (`github.com/JGtm/LevelUp/libs/film`) | `decfilm`, `replay`, `filmcache`, `finalise`, `types`, `damagetag`, `killicon`, `medalname`, `revision`, `filmshell`, `equipenergy`, `filmproc`, `identity`, `playerposition`, `objectiveevent`, `filmobs`, `research/...` | `source`, `profile`, `grammar/...`, `facts/...`, `constat`, `atomicfile`, `filelock`, `flatjson` | `libs/gamefiles` (`mapvar`), `libs/himap` (`himap`) — rien du dépôt |

Succès = (1) `go list -deps ./...` dans `libs/film` (tests compris, `-test`) ne montre aucun paquet
`levelup/go-api/...` ; (2) la table `archlint/briques_imports_test.go` porte la ligne `film` et son
allowlist est VIDE à la clôture ; (3) **aucune des cinq RÉVISIONS de couche ne monte** (les faits persistés sont
clés par les constantes `Rev`, pas par les empreintes de sources) : aucun fait persisté ne devient
périmé, aucune recuisson en production ; les empreintes et goldens de périmètre sont régénérés UNE
SEULE FOIS, à révision constante, par la porte à deux verrous, à la fin du lot 2 (décision D2-F amendée
le 2026-10-29, découverte D2-L2-a), avec la preuve d'équivalence des sorties (T3) ; le lot 3 ne
régénère rien ; (4) `replay-corpus-gate --reference=base` (base =
commit du lot 0) : les 19 témoins du corpus (`config/replay_corpus.toml`) rendent des artefacts
identiques ; `replay-equiv` vert ; (5) même liste de tests, mêmes verdicts, avant et après (modulo le
préfixe de paquet) ; (6) CI verte (36 contrôles), image Docker construite, lint des trois modules
vert.

Effort estimé : 5 à 6 jours d'agent (l'audit disait 2 à 4 : il ne comptait ni les 148 fichiers de
test couplés hors du film, ni les 52 ratchets `archlint`, ni les 386 fichiers de LevelUp à
réécrire). Aucune réécriture d'algorithme : déplacements, copies bornées, conversions aux
frontières.

## Branches

- Lots 0 à 5 : `refactor/decouplage-d2-decodeur-film` depuis `release/v8.1.0` (`a5bae50b4`),
  worktree dédié `../LevelUp-wt-decouplage-d2`, un commit par lot (préfixe « D2 lot N »),
  fusion squash par PR dans `release/v8.1.0`. Jamais sur `main` (push `main` = déploiement prod).
- Caches de compilation : `GOCACHE=%LOCALAPPDATA%\go-build-d2`, `GOLANGCI_LINT_CACHE=%LOCALAPPDATA%\golangci-d2`
  (clôture : `scripts/disk-hygiene.ps1 -Lot d2 -Apply`).
- Autorisation : message de l'utilisateur du 2026-10-28 (« Tu t'arrêtes jamais, tu enchaînes quand
  t'as fini D2 sur D3 ») = feu vert pour commits, PR et fusion squash dans la release ; `main` n'est
  pas touché.

## Chiffres de départ (relevé du 2026-10-28 sur `a5bae50b4`, à rouvrir à l'exécution)

- `film/**` : 153 749 lignes de production, 344 695 de tests ; 1 486 fichiers de test dont 579 sous
  `//go:build research` (21 aussi `campagne_overlay`), 1 sous `cgo`.
- Imports de production sortant du film (paquets `levelup/go-api/internal/...`, nombre de fichiers) :
  `games/weapons/filmshell` 11, `observability` 9, `games/canonical` 7 (uniquement les symboles de
  `film_identity.go`), `domain/highlightevent` 6, `i18n` 5 (`replay` : `Text`, `MarshalFlat`,
  `UnmarshalFlat`, `FlatLayout`, `FlatEnd`), `domain/playerposition` 3, `games/halo_infinite/equipenergy` 2,
  `filmproc` 2 (`research/cmd_*`), `domain/objectiveevent` 2, `domain/equipmentusage` 2,
  `platform/atomicfile` 1 ; plus `testutil.ReplayChronicleVersions` (2 fichiers de production),
  `title.FilmShortMatchID` (1), `mappings.ObjectiveModeEntry` (1). Briques déjà importées :
  `libs/gamefiles/mapvar` 27, `libs/himap/himap` 2.
- Tests du film couplés hors du film : 87 fichiers non `research` (`filmproc` 33, `testutil` 14,
  `highlightevent` 9, `canonical` 8, `domain/title` 5, `observability` 4, `mappings` 3, `testfixtures` 2,
  `i18n` 2, `filmshell` 2, `equipenergy` 2, `replaydoc` 2, `equipmentusage` 2, `weapons` 1,
  `objectiveevent` 1, `analysis/digest` 1) et 61 fichiers `research` (`filmproc` 36, `domain/title` 9,
  `testutil` 7, `highlightevent` 6, `mappings` 4, `filmshell` 2, `weapons` 2, `domain` 1). 19 fichiers
  de test citent encore `C:\Users\Guillaume\Projects\LevelUp` (découverte 9 de D1).
- LevelUp importe le film depuis 45 paquets (198 fichiers de production, 188 de test) : `replay` 111,
  `decfilm` 72, `filmcache` 34, `finalise` 10, `types` 7, `damagetag` 4, `medalname` 2, `killicon` 2 ;
  `filmproc` depuis 10 paquets, `filmshell` 5, `equipenergy` 2, `playerposition` 5, `objectiveevent` 5,
  `canonical.Link*` 1 (`service/replayview/convert_identity.go`).
- `archlint` : 52 fichiers citent `internal/games/halo_infinite/film` (20 constantes nommées), tous
  racinés sur le module `apps/go-api`.
- Empreinte des couches (`film/revision/empreinte.go`) : le chemin haché est RELATIF À LA RACINE DE
  LA COUCHE (un `git mv` du dossier ne coûte rien), MAIS (a) le flux de jetons garde les littéraux,
  donc les chaînes d'import `"levelup/go-api/internal/games/halo_infinite/film/..."` entrent dans le
  hachage, (b) `couches.go` nomme les racines relatives au module (`internal/games/halo_infinite/film/internal/...`),
  (c) `empreinteur.paquet` préfixe chaque fichier d'un paquet importé par son dossier relatif au
  module, et les goldens de périmètre écrivent ce dossier. Un déplacement nu changerait les cinq
  empreintes. D'où la décision D2-F.
- Sans lien avec le code : `filmprofile` (`games/halo_infinite/filmprofile`) est « HORS du décodeur »
  par contrat (lit `data/titles/{slug}/reference/film_profiles.json` par `PathResolver`, n'importe
  rien du film) : il reste dans LevelUp.

## Décisions tranchées avant exécution

Prises par le pilote sur pièces le 2026-10-28 ; l'utilisateur a délégué les décisions techniques au
pilote (2026-10-28, D1) et demandé d'enchaîner sans s'arrêter. Elles sont FERMES.

- **D2-A — Un module `libs/film`**, chemin `github.com/JGtm/LevelUp/libs/film`, ligne `go 1.26.10`,
  paquets au premier niveau de `libs/film/` (le dossier `film/` d'aujourd'hui DEVIENT la racine du
  module : `libs/film/decfilm`, `libs/film/replay`, `libs/film/internal/grammar`, ...). `internal/`
  reste fermé au compilateur pour LevelUp, ouvert aux paquets de la brique : c'est l'ADR 0034 D-1
  inchangé. Rang : `film` importe `gamefiles` et `himap` ; rien n'importe `film` dans `libs/`.
  Pas de `require` des briques voisines ni de `replace` (ADR 0041 D2) ; les dépendances tierces du
  module sont écrites à la main à la version d'`apps/go-api` ; `make tidy` saute le module.
- **D2-B — Types de sortie : la brique possède ce qu'elle seule produit ; LevelUp garde ce que
  plusieurs sources produisent ou que `internal/analysis` lit.**
  - `domain/playerposition` et `domain/objectiveevent` (33 et 63 lignes, feuilles, produits par le
    décodeur, lus par `handlers`, `platform/duckdb`, `port`, `service`, `cmd/diag_weapons_v3` : aucun
    lecteur dans `analysis`) : DÉPLACÉS dans la brique (`libs/film/playerposition`,
    `libs/film/objectiveevent`), paquets `domain/*` supprimés, lecteurs réécrits.
  - `domain/highlightevent` : RESTE dans LevelUp (produit aussi par l'import OpenSpartan et le
    rattrapage des médailles ; lu par `analysis`). La brique définit SON type
    `types.HighlightEvent` (+ constantes `EventType*`, mêmes champs, même JSON) ; la conversion vers
    `highlightevent.HighlightEvent` se fait dans l'adaptateur de titre
    `internal/games/halo_infinite/` (ADR 0012, même règle que la décision 4 de l'audit pour D3) ;
    les quatre appelants de `decfilm.ParseHighlightEvents` dans `internal/sync` passent par lui.
    `decfilm.Kill` et le flux de `killcollector` : vérifier sur pièces s'ils portent le type
    `highlightevent` ; si oui, même conversion au même endroit.
  - `domain/equipmentusage` : RESTE dans LevelUp (lu par `analysis/{sessionusage,squademprise,trends}`).
    La brique porte ses propres constantes de famille et les deux fonctions (`EquipmentOutcomeFamilies`,
    `UsageFamilySpawnsPiece`) dans `replay` (copie bornée, ~60 lignes). Garde-rail : un test de
    LevelUp compare les deux listes de familles et les deux prédicats (règle 6 : deux copies, un test
    qui refuse la divergence). Dette consignée : `analysis` lit de la connaissance d'équipement
    Halo Infinite sous un nom `domain/` (préexistant, hors D2).
  - `domain/replaydoc` : RESTE dans LevelUp (document d'API produit par `service/replayview`). Les
    deux tests du film qui l'importent suivent la règle D2-E.
  - `games/canonical/film_identity.go` (364 L, aucun import) : DÉPLACÉ dans `libs/film/identity`
    (décision 3 de l'audit) ; `service/replayview/convert_identity.go` l'importe de là ; `canonical`
    perd le fichier et son test ; ADR 0034 amendé (« le contrat d'identité des entités est celui du
    décodeur ; un second titre convertirait côté LevelUp »).
- **D2-C — Catalogues** : `games/weapons/filmshell` → `libs/film/filmshell`,
  `games/halo_infinite/equipenergy` → `libs/film/equipenergy` (décision 2 de l'audit). Lecteurs
  LevelUp réécrits (`games/weapons`, `platform/duckdb`, `sync/killcollector`, `games/halo_infinite`,
  `cmd/diag_film`, `replaybuild`, `cmd/equipenergy-build`).
- **D2-D — Infrastructure de la brique** :
  - `observability` (14 sites `AddInt` / `IncCounter` dans `replay` et `filmcache`, 4 tests `LoadCounter`) :
    paquet `libs/film/filmobs` = compteurs tenus par la brique (`AddInt`, `IncCounter`, `Load`,
    `Snapshot`) + un puits optionnel `SetSink(func(name string, delta int64))` (variable NON exportée,
    posée une fois au démarrage : `internal/api/wire` et chaque binaire qui lit ces compteurs via
    `observability.LoadCounter`, à relever par grep). Les NOMS de compteurs ne changent pas :
    `/debug/vars` est identique. Les tests de la brique lisent `filmobs.Load`.
  - `platform/atomicfile` (161 L) et `platform/filelock` (289 L), tirés par `filmproc` et `filmcache` :
    s'ils n'ont pas d'autre importeur que ce qui part dans la brique → DÉPLACÉS sous
    `libs/film/internal/` ; sinon COPIÉS sous `libs/film/internal/` (deux copies, règle 6) avec un
    test LevelUp qui exige les deux sources identiques à l'octet (hors ligne `package`).
  - `filmproc` (1 865 L, importé par 10 paquets de LevelUp, 69 tests du film) → `libs/film/filmproc`
    (garde-fou mémoire du décodage : c'est au décodeur). LevelUp l'importe de là.
  - `i18n` dans `replay` : la brique définit `replay.Text` (`map[string]string`, code de langue →
    libellé ; AUCUNE liste de langues dans la brique) et porte `libs/film/internal/flatjson`, copie
    du codec « à plat » (`i18n/flat.go`) travaillant sur ce type. LE FIL JSON NE CHANGE PAS D'UN
    OCTET (artefacts cuits relus sans recuisson, ADR 0040 D-9). Conversions `i18n.Text` <-> `replay.Text`
    dans `internal/games/halo_infinite/replaylabels` (producteur des libellés) et utilisées par
    `service/replayview` (consommateur). Garde-rails : (a) test LevelUp qui encode les mêmes
    libellés par `i18n.MarshalFlat` et par le codec de la brique et exige les mêmes octets, dans les
    deux sens ; (b) `replay-equiv` et `replay-corpus-gate` (documents identiques).
  - `title.FilmShortMatchID` (1 site), `mappings.ObjectiveModeEntry` (1 site),
    `testutil.ReplayChronicleVersions` (2 fichiers de PRODUCTION) : à rouvrir ; règle : si le
    seul usage est le film, le helper DÉMÉNAGE dans la brique et LevelUp l'importe de là ; sinon la
    brique porte sa définition locale (type ou fonction de 10 lignes) et LevelUp convertit.
- **D2-E — Tests du film couplés à LevelUp** (148 fichiers), trois règles, appliquées fichier par
  fichier et comptées au journal :
  - R0 : l'import visé DÉMÉNAGE dans la brique (D2-B, C, D : `filmproc`, `filmshell`, `equipenergy`,
    `playerposition`, `objectiveevent`, `identity`, `filmobs`, `types.HighlightEvent`) → réécriture
    d'import seule.
  - R1 : le test n'exerce que la SURFACE EXPORTÉE de la brique et a besoin de LevelUp (données du
    parc par `testutil.RepoRoot` / `title.NewPathResolver`, `mappings.Load*`, `replaydoc`,
    `equipmentusage`, `weapons`, `analysis/digest`, `testfixtures`) → il DÉMÉNAGE dans
    `apps/go-api/internal/filmcorpus/` (nouveau paquet de tests seuls, mêmes tags de build, mêmes
    noms de test, suffixe `_research_test.go` conservé).
  - R2 : le test exerce des internes NON exportés de la brique → il RESTE et se réécrit sur des
    helpers locaux à la brique : remontée à la racine du dépôt par un helper de test par paquet
    (marqueur `go.work`, patron `mapvar/depot_research_test.go` du lot 2c de D1, inscrit dans les
    allowlists de `no_repo_root_walk_test` et `pas_de_lecture_sous_ai_test`), `t.Skip` si le parc
    est absent ; libellés par une fixture minimale construite à la main (jamais un chargeur TOML
    copié). Un test R2 qui ne peut pas se réécrire sans copier un chargeur de LevelUp est signalé
    au journal avec son nom et traité comme R1 après EXPORT du seul symbole nécessaire si c'est un
    vrai contrat, sinon il devient un instrument `research` (décision « option 2 » de l'utilisateur,
    D1 lot 2c).
  - Les 19 fichiers qui citent `Projects\LevelUp` perdent ce chemin dans le même geste (helper R2).
  - `film/research/cmd_*` (5 `main`, importent `grammar`, `source`, `filmshell`, `filmproc`,
    `highlightevent`) restent dans la brique sous `libs/film/research/cmd_*` : R0 les couvre.
- **D2-F — Révisions INCHANGÉES ; cadre de hachage HISTORIQUE ; une régénération à révision constante.**
  AMENDÉE le 2026-10-29 (découverte D2-L2-a du lot 2.1) : le cadre historique rend le hachage insensible
  au chemin du MODULE de la brique (lot 3 sans effet), mais les paquets qui changent de dossier au lot 2
  (`domain/playerposition`, `domain/objectiveevent`, `games/weapons/filmshell`) et le passage de
  `grammar`/`killsource` au type `types.HighlightEvent` (2.7) changent légitimement les jetons et les
  lignes `paquet` des goldens de `grammar`, `killsource`, `objectives`. Décision du pilote : PAS de table
  de renommage par paquet ; les goldens de révision ET de périmètre sont régénérés UNE FOIS à révision
  constante (porte à deux verrous : drapeau `-update-<nom>` + variable `LEVELUP_UPDATE_<NOM>`, item 2.12),
  avec dans la chronique de chaque couche touchée une entrée « empreinte régénérée, révision inchangée :
  découplage D2, sortie identique prouvée par les digests `replay-equiv` (T3) et T1 ». Précédent : la
  chronique de `grammar` porte déjà une régénération à révision constante (`rev_chronique.go`, « amendé au
  lieu de monter à .4, empreinte régénérée »). Les révisions (`source-2026-09-16.2`, `profile-2026-10-16`,
  `grammar-2026-10-28`, `killsource-2026-10-18`, `objectives-2026-10-17`) ne bougent pas. Texte d'origine :
  `film/revision` définit le cadre `cadreHistorique = "levelup/go-api/internal/games/halo_infinite/film"` :
  (a) les jetons d'une chaîne d'import qui commence par le chemin du module courant
  (`github.com/JGtm/LevelUp/libs/film`) sont hachés avec le préfixe historique à la place ; (b) les
  racines de couche sont relatives à la racine du MODULE de la brique (`internal/source`, ...) mais
  nommées, dans le hachage et dans les goldens de périmètre, par leur dossier historique
  (`internal/games/halo_infinite/film/internal/source`, ...). Le cadre se pose AU LOT 2 (dans le
  module actuel, où il est l'identité : préfixe historique = préfixe courant), avec un test qui
  prouve qu'une source écrite avec le préfixe du futur module hache comme avec l'ancien ; au lot 3
  le déplacement ne peut alors rien changer. Gate à chaque lot : les goldens de `revision`
  (`porte` à deux verrous, `perimetre_golden`) passent SANS réécriture et `git diff` des goldens est
  vide. Raison : une révision qui monte sans qu'un octet de grammaire ait changé périmerait TOUS les
  faits persistés (ADR 0034 : la publication rejoue depuis les faits quand ils sont frais) et
  déclencherait une recuisson générale en production (mémoire : jamais de cuisson en lot). La doc
  d'`empreinte.go` dit déjà que le cadre est un contrat choisi pour survivre à un `git mv` : ce lot
  l'étend au chemin de module, et le dit dans l'ADR 0034.
- **D2-G — Garde-fou G avant tout déplacement (lot 1)** : ligne `film` dans la table
  `archlint/briques_imports_test.go` (paquets = `apps/go-api/internal/games/halo_infinite/film`,
  `.../games/weapons/filmshell`, `.../games/halo_infinite/equipenergy` au lot 1, `libs/film` à partir
  du lot 3 ; permis : `gamefiles`, `himap`), avec une tolérance DATÉE par import visé et par compte
  de fichiers (production + tests : les chiffres de départ ci-dessus), qui ne peut que descendre et
  vaut zéro à la clôture du lot 2 (l'allowlist est alors retirée). Plus : (a) un test qui exige que
  CHAQUE dossier Go sous `libs/**` figure dans la table (découverte 38 de D1) ; (b) le pas CI
  `go list -deps` des briques passe aussi `-test` ; (c) `no_title_package_in_analysis_test` interdit
  aussi l'import DIRECT de `github.com/JGtm/LevelUp/libs/film/...` depuis `internal/analysis`, et un
  second test lit la FERMETURE (`go list -deps ./internal/analysis/...`) avec une allowlist datée
  des fuites transitives constatées (`games/weapons` → `filmshell` au moins ; `domain/equipmentusage`
  n'en est pas une puisqu'il reste dans LevelUp) qui ne peut que descendre.
- **D2-H — Ce qui ne bouge pas** : `replaybuild`, `replay-worker`, `replaychild`, `sync/replayartifacts`,
  `sync/killcollector`, `service/replayview`, `mapcatalog`, `filmprofile`, `cmd/replay-*`,
  `cmd/killsource`, `cmd/diag_*`, `tools/film_re` : LevelUp (audit, traitement 6). Réécriture
  d'imports seulement.
- **D2-I — CI et outillage, mêmes formes que D1** : pas CI vet + test + `go list -deps` (-test) +
  lint + govulncheck pour `libs/film` ; les trois lots de couverture du film (`film-grammar-lourd`,
  `film-grammar-reste`, `film` de `.github/ci/go-shards.txt`) deviennent les lots du MODULE
  `libs/film` (le script `ci_go_shard.sh` apprend le module, ou un second fichier de lots ; aucun lot
  au-dessus de 600 s, mêmes filtres `run:` / `skip:` des tests lourds) ; le job « Film race » tourne
  dans `libs/film` ; la baseline JSONL de non-régression des tests GAGNE un fichier pour la brique,
  produit en réécrivant le préfixe de paquet des 1 052 entrées existantes (aucun test ne sort du
  contrôle de présence), et `check_test_baseline.sh` juge les deux modules ; filtre `croisee` de
  `ci.yml`, `cache-dependency-path` (10 occurrences), `Dockerfile` (COPY des `go.mod`/`go.sum`),
  `go.work`, `.github/dependabot.yml`, `Makefile` (cibles qui énumèrent des paquets du film,
  `libs-test` : délai par module mesuré au lot 3), `.golangci.yml` (règle 220 sur les chemins du
  film : réécrite pour le module, et vérifiée ACTIVE après le déplacement — un `path:` qui ne
  matche plus relâche le lint en silence).
- **D2-J — Docs dans le même lot** : ADR 0041 (troisième module, rang), ADR 0034 (identité au
  décodeur ; cadre historique de l'empreinte ; chemins de la façade et des ratchets), CLAUDE.md
  (tableau des briques et chemins `libs/film`), `docs/ARCHITECTURE_V6.md` + `docs/FR/` si le chemin
  du décodeur y est cité, `docs/COMMANDS.md` si une commande cite le chemin. Pas de nouvelle doc.

## Lot 0 — Préalable : branche, témoins, état de la base

- [x] 0.1 Worktree `../LevelUp-wt-decouplage-d2`, branche `refactor/decouplage-d2-decodeur-film`
  depuis `release/v8.1.0` ; CI de la base verte constatée (`gh run list --branch release/v8.1.0 --limit 1`).
- [x] 0.2 Témoin T1 : `go test -json ./... -count=1` (CGO, `-p 1` non requis : pas d'intégration)
  dans `apps/go-api`, réduit à la liste `(paquet, test, verdict)` triée → `temoins_d2_lot0_avant/tests_go_api.tsv`
  + liste des paquets `ok`. Même capture avec `-tags research` en `go vet` seulement (compilation
  des 579 instruments : `go vet -tags research ./internal/games/halo_infinite/film/...` et
  `-tags 'research campagne_overlay'`) → sortie vide attendue.
- [x] 0.3 Témoin T2 : les cinq révisions et empreintes de couche + les goldens de périmètre :
  `go test ./internal/games/halo_infinite/film/revision/ -run . -v -count=1` (sortie capturée) et
  `md5sum` de tous les fichiers golden de `revision` et de `film/**/testdata/equivalence/*.tsv`
  → `temoins_d2_lot0_avant/empreintes.txt`.
- [x] 0.4 Témoin T3 : `go run ./cmd/replay-equiv` (mode par défaut, références versionnées) → code
  de sortie et journal capturés. Si un film de référence manque dans le parc local, c'est l'état de
  base : il manquera aussi après, et le même verdict est attendu.
- [x] 0.5 Les chiffres de départ ci-dessus rouverts (mêmes commandes que le relevé) ; écart consigné
  au journal s'il y en a.
- [x] 0.6 Gate : témoins écrits, MD5 au journal, `.ai/thought_log.md` « Découplage D2 — lot 0 ».

**Clos quand** : 0.6 passé. Aucun commit de code (le plan est commité au lot 1 avec le garde-fou).

**Exécution du lot 0 (2026-10-28/29, poste local, worktree `LevelUp-wt-decouplage-d2`, base `a5bae50b4`)** — témoins sous
`%TEMP%\claude\c--Users-Guillaume-Downloads-Scripts-LevelUp\77a8b90a-21a6-40a0-ac64-eb152f616862\scratchpad\temoins_d2_lot0_avant\`
(`md5s.txt` = MD5 de chaque témoin). Aucun fichier du dépôt touché hors ce plan et le journal.
- 0.1 : arbre propre (seul le plan est non suivi), HEAD `a5bae50b4`, branche `refactor/decouplage-d2-decodeur-film`.
  CI de `release/v8.1.0` sur `a5bae50b4` : success (run 40769279871, 13 min 39 s).
- 0.2 T1 : `go test -json ./... -count=1 -timeout 1200s` (CGO) : code 0, 351 s. 358 paquets : 212 `ok`, 146 sans test
  (skip), 0 rouge. Événements de test : 19 748 pass, 366 skip, 0 fail (sous-tests compris ; 20 114 lignes
  `(paquet, test, verdict)` dans `tests_go_api.tsv`). `go vet -tags research ./…/film/...` : code 0, sortie vide.
  `go vet -tags 'research campagne_overlay' ./…/film/...` : **code 1, état de base** (les 21 fichiers
  `research && campagne_overlay` de `film/internal/grammar` n'ont pas leurs symboles `bis2*` : ils se compilent contre un
  recouvrement de campagne qui n'est pas versionné ; sortie dans `vet_research_overlay.txt`). La voie `research` seule
  est le témoin de compilation exploitable ; l'overlay ne peut pas servir de témoin avant/après (même erreur attendue après).
- 0.3 T2 : `go test ./…/film/revision/ -run . -v -count=1` : code 0, 40 tests PASS (`revision_v.txt` porte les empreintes
  calculées et les nombres de fichiers de chaque couche). 202 fichiers MD5-és sous tous les `testdata/` de `film/**`
  (`empreintes.txt` : goldens de révision et de périmètre, 21 `equivalence/*.tsv` (pour 20 films au corpus), 20 `*.facts.json`, autres goldens).
  Révisions : source-2026-09-16.2, profile-2026-10-16, grammar-2026-10-28, killsource-2026-10-18, objectives-2026-10-17 ;
  `SchemaVersion` 106, `SchemaDesFaits` 18, `UsageSummaryRev` us7 (`revisions.txt`).
- 0.4 T3 : `go run ./cmd/replay-equiv -repo-root <dépôt principal>` (le worktree n'a pas `data/cache/film_chunks` ; le dépôt
  principal est sur le même commit, sans modification Go ; les 20 films du corpus sont présents au parc). **Code 1,
  « BILAN : 0 identique, 20 different, 0 ecarte, 0 echec, 0 illisible »** : les références versionnées de
  `replay/testdata/equivalence/*.tsv` sont PÉRIMÉES (64 étapes figées contre 67 attendues ; `knockbacks` insérée, digests
  suivants décalés ; `score` lui-même diffère ; fichiers figés au commit de la v8.0.0). État de base : le mode par défaut ne
  peut pas rendre « 0 écart ». Témoin retenu à la place : les digests OBTENUS, conservés par `-out-dir`
  (`replay_equiv_obtenus/*.tsv`, MD5 dans `md5s.txt`), identiques entre deux exécutions successives de la base (1 047 lignes
  obtenues comparées, 0 écart) : après le découplage, mêmes digests obtenus, mêmes MD5. Durées : 409 s et 715 s.
- 0.5 : chiffres de départ rouverts (`chiffres.txt`) : identiques sauf (a) `testutil.ReplayChronicleVersions`,
  `title.FilmShortMatchID`, `mappings.ObjectiveModeEntry` ne sont pas des imports de production mais des mentions en commentaire ;
  (b) l'agrégat « 45 paquets / 198 fichiers de production / 188 de test » importeurs du film se relit 44 répertoires en production,
  47 en tout / 179 fichiers de production / 178 de test (les comptes par sous-paquet sont, eux, identiques) ;
  (c) archlint : 51 fichiers au lieu de 52.
- 0.6 : témoins écrits, MD5 dans `md5s.txt`, entrée « Découplage D2 — lot 0 » au journal.

## Lot 1 — Garde-fou G posé AVANT tout déplacement

- [x] 1.1 Ligne `film` dans `briques` (paquets du lot 1 : `apps/go-api/internal/games/halo_infinite/film`,
  `apps/go-api/internal/games/weapons/filmshell`, `apps/go-api/internal/games/halo_infinite/equipenergy`
  — chemins RÉCURSIFS ; permis : `gamefiles`, `himap`).
- [x] 1.2 Tolérances datées par import visé (compte de fichiers, production + tests), égales au relevé
  du lot 0 ; le test rougit si un compte MONTE et exige qu'une tolérance à zéro soit retirée.
- [x] 1.3 Test d'exhaustivité : chaque dossier qui porte un `.go` sous `libs/**` figure dans la table
  (découverte 38 de D1) — vert aujourd'hui avec `gamefiles` et `himap`.
- [x] 1.4 `no_title_package_in_analysis_test` : interdit direct de `github.com/JGtm/LevelUp/libs/film/...`
  dans `internal/analysis` ; test de fermeture `go list -deps` avec allowlist datée des fuites
  transitives (mesurées au lot 1, dont `games/weapons` → `filmshell` par anticipation du lot 2 : à
  inscrire quand elle apparaît, pas avant) qui ne peut que descendre.
- [x] 1.5 CI : `-test` ajouté aux trois pas `go list -deps` des briques (`ci.yml`) et à `make libs-test`.
- [x] 1.6 Gate : `go test ./internal/archlint/ -run 'TestBriques|TestNoTitlePackageInAnalysis|TestFermeture' -count=1`
  vert ; mutation : une tolérance baissée de 1 → rouge avec fichier et import, remise → vert.
  Commit « test(archlint): garde-fou d'autonomie du décodeur de film (D2 lot 1) » avec ce plan.

**Clos quand** : 1.6 passé, CI de la branche verte.

**Exécution du lot 1 (2026-10-29, worktree `LevelUp-wt-decouplage-d2`, base `a5bae50b4`)** — fichiers : `archlint/briques_imports_test.go`
(champs `recursif` et `tolerances` de `brique`, ligne `film`, `tolerancesFilm`, parcours récursif), `archlint/briques_garde_test.go`
(neuf : auto-preuves récursif / tolérances, exhaustivité de `libs/`), `archlint/no_title_package_in_analysis_test.go` (interdit direct +
fermeture), `.github/workflows/ci.yml` (2 lignes), `Makefile` (1 ligne + son commentaire).
- 1.1 : ligne `film` = trois arbres récursifs (`film`, `games/weapons/filmshell`, `games/halo_infinite/equipenergy`), permis = les
  paquets de `gamefiles` et de `himap`. L'appartenance se juge à la frontière de chemin (`film` ne couvre pas `filmprofile` ; auto-preuve).
  `filmshell` et `equipenergy` n'importent RIEN du dépôt (aucun franchissement mesuré en leur nom).
- 1.2 : tolérances recomptées par le parcours du test (un fichier compte une fois par import visé, tests `research` compris, `testdata` et
  dossiers `.x`/`_x` sautés) ; 189 franchissements, égaux ligne à ligne au relevé du lot 0 (`chiffres.txt`) :

  | Import visé (`levelup/go-api/internal/...`) | Fichiers | Lot qui le vide |
  |---|---|---|
  | `filmproc` | 71 (2 prod, 33 tests, 36 research) | 2.6 |
  | `testutil` | 21 (tests seuls) | 2.11 |
  | `domain/highlightevent` | 21 (6 prod, 15 tests) | 2.7 |
  | `games/canonical` | 15 (7 prod, 8 tests) | 2.4 |
  | `domain/title` | 14 (tests seuls) | 2.10 / 2.11 |
  | `observability` | 13 (9 prod, 4 tests) | 2.2 |
  | `i18n` | 7 (5 prod, 2 tests) | 2.9 |
  | `games/mappings` | 7 (tests seuls) | 2.10 / 2.11 |
  | `domain/equipmentusage` | 4 (2 prod, 2 tests) | 2.8 |
  | `games/weapons` | 3 | 2.11 |
  | `domain/playerposition` | 3 (prod 3) | 2.3 |
  | `domain/objectiveevent` | 3 (2 prod, 1 test) | 2.3 |
  | `testfixtures`, `domain/replaydoc` | 2 + 2 | 2.11 |
  | `platform/atomicfile`, `domain`, `analysis/digest` | 1 + 1 + 1 | 2.6 ; 2.11 ; 2.11 |

  Règles du test, chacune prouvée par une auto-preuve sur la fonction pure (`TestBriquesImports_TolerancesAutoPreuve`) : compte supérieur
  (liste fichier -> import), inférieur (« baisse la tolérance à N »), entrée à zéro (« retirer l'entrée »), entrée sans fichier, import
  absent de la table (rouge immédiat). Tests de la brique contrôlés comme la production (règle D1-D).
- 1.3 : `TestChaqueDossierGoDeLibsEstDansLaTable` (5 dossiers Go aujourd'hui : `gamefiles` x4, `himap` x1) ; vert ; mutation réelle :
  `libs/himap/nouveau/x.go` -> rouge (`libs/himap/nouveau`), fichier retiré ensuite ; auto-preuve sur la fonction pure `dossiersHorsTable`.
- 1.4 : (a) l'import direct de la brique (les trois arbres du lot 1 ET `github.com/JGtm/LevelUp/libs/film/...`) depuis `internal/analysis/**`
  est une violation sans allowlist ; (b) `TestFermetureDeAnalysisSansDecodeurDeFilm` (`go list -deps -test -f {{.ImportPath}}
  ./internal/analysis/...`, 0,3 s). Fermeture mesurée : 312 lignes (paquets + variantes de test), AUCUN paquet du décodeur, ni
  `filmshell` ni `equipenergy` ; `games/canonical` et `games/mappings` y sont, `canonical` n'est pas un paquet du film.
  `fuitesTransitivesAnalysis` posée VIDE (entrée morte = rouge). Mutation réelle : `import _ ".../games/weapons/filmshell"` dans
  `games/mappings/assets.go` -> rouge avec `levelup/go-api/internal/games/weapons/filmshell` ; fichier restauré.
- 1.5 : `go list -deps -test ./... | grep -E 'levelup/go-api'` dans les deux pas de `ci.yml` (le plan disait trois : il n'y en a que deux,
  `libs/himap` et `libs/gamefiles`) et dans `libs-test` ; vérifié à la main dans les deux modules (aucun paquet `levelup/go-api`, 145 et 152 lignes).
- 1.6 : `go test ./internal/archlint/ -run 'TestBriques|TestChaqueDossier|TestNoTitlePackage|TestFermeture' -count=1 -v` vert (le test de
  l'interdit direct s'appelle `TestAnalysisImporteAucunPaquetDeTitre`, `TestNoTitlePackage` ne correspond à rien : lancé aussi) ;
  `go test ./internal/archlint/ -count=1` complet vert (100 s) ; `go vet`, `gofmt -l` vides ; `golangci-lint run ./internal/archlint/...` 0 issue.
  Mutation de tolérance : `domain/equipmentusage` 4 -> 3, sortie rouge :
  `[film] apps/go-api/internal/games/halo_infinite/film/replay/usage_summary_families.go  -> levelup/go-api/internal/domain/equipmentusage  (compte 4 > tolérance 3)`
  (idem pour `usage_summary_families_guard_test.go`, `usage_summary_outcomes.go`, `usage_summary_outcomes_test.go`) ; remise à 4 -> vert ;
  4 -> 5 : `[film] levelup/go-api/internal/domain/equipmentusage : compte mesuré 4 < tolérance 5 — baisse la tolérance à 4`.

## Lot 2 — Coupes dans le module actuel (sans déplacement du dossier `film/`)

Chaque item fait descendre une tolérance du lot 1 ; T1 à T3 relancés en fin de lot.

- [x] 2.1 Cadre historique de l'empreinte (D2-F) dans `film/revision` : constante, normalisation des
  chaînes d'import dans `jetons.go`, racines de couche nommées par le dossier historique, test
  `TestCadreHistoriqueDesImports` (source avec le préfixe `github.com/JGtm/LevelUp/libs/film` =
  même empreinte qu'avec l'ancien) ; goldens inchangés (`git diff` vide sur `revision/testdata`
  et les goldens de périmètre) ; T2 identique.
- [x] 2.2 `film/filmobs` + puits (D2-D) ; les 14 sites et 4 tests ; câblage du puits dans
  `internal/api/wire` et les binaires relevés ; tolérance `observability` = 0.
- [x] 2.3 `domain/playerposition` → `film/playerposition`, `domain/objectiveevent` → `film/objectiveevent`
  (`git mv`, lecteurs LevelUp réécrits, paquets `domain/*` supprimés) ; tolérances = 0.
- [x] 2.4 `canonical/film_identity.go` (+ test) → `film/identity` ; `replayview/convert_identity.go`
  réécrit ; ratchets `film_pont_identite_test`, `no_identity_bridge_outside_registry_test` ajustés ;
  texte d'amendement de l'ADR 0034 écrit ; tolérance `canonical` = 0.
- [x] 2.5 `filmshell` → `film/filmshell`, `equipenergy` → `film/equipenergy` (`git mv`, 7 paquets
  réécrits) ; `paquetsInterTitres` de `no_title_package_in_analysis` relu (`weapons` importe
  désormais le film : fuite transitive inscrite datée au test de fermeture du 1.4) ; tolérances = 0.
- [x] 2.6 `filmproc` → `film/filmproc` ; `atomicfile`, `filelock` déplacés ou copiés selon D2-D (test
  d'identité des copies) ; 10 paquets LevelUp réécrits ; tolérances `filmproc`, `atomicfile` = 0.
- [x] 2.7 `types.HighlightEvent` (posé en `film/highlightevent`, cf. D2-L2-i) + constantes ; `grammar`, `killsource`, `decfilm` basculent ;
  adaptateur dans `internal/games/halo_infinite/` ; quatre appelants de `sync` + `decfilm.Kill` /
  `killcollector` / `cmd/killsource` vérifiés sur pièces ; 15 tests du film basculent ; tolérance
  `highlightevent` = 0. Témoin : `go test ./internal/sync/... ./internal/games/halo_infinite/...`
  identique à T1.
- [x] 2.8 Familles d'équipement locales dans `replay` + test LevelUp d'égalité des deux listes et
  des deux prédicats ; tolérance `equipmentusage` = 0 (2 tests compris).
- [x] 2.9 `replay.Text` + `film/internal/flatjson` ; 8 champs et 2 catalogues (`callouts_catalog`,
  `document_labels*`, `document_objective_objects`) ; conversions dans `replaylabels` ; `replayview`
  réécrit ; test d'égalité des octets des deux codecs (deux sens, les cinq dispositions `FlatLayout`
  du fichier) ; les documents de `replay/testdata` relus à l'identique ; tolérance `i18n` = 0 (2 tests
  compris). T3 vert.
- [x] 2.10 Reliquats de production : `title.FilmShortMatchID`, `mappings.ObjectiveModeEntry`,
  `testutil.ReplayChronicleVersions` (règle D2-D dernier point) ; tolérances = 0. **Statué `[x]` :
  les trois étaient des MENTIONS EN COMMENTAIRE (lot 0, 0.5), aucun import de production ; les quatre
  commentaires qui nommaient un symbole de LevelUp (ces trois-là et `title.PathResolver`) disent la notion.**
- [x] 2.11 Tests couplés restants (R1 → `internal/filmcorpus/`, R2 → helpers locaux ; 19 chemins
  `Projects\LevelUp` retirés) : compte R0 / R1 / R2 / devenus instruments au journal, nom par nom
  pour R1 et « instrument » ; toutes les tolérances à zéro, allowlist retirée du test.
- [x] 2.12 Gate : `grep -rhoE --include='*.go' '"levelup/go-api/internal/[^"]+"' internal/games/halo_infinite/film | grep -v halo_infinite/film | sort -u`
  vide (production ET tests) ; `go vet ./...`, `go vet -tags research ./internal/games/halo_infinite/film/...`,
  `-tags 'research campagne_overlay'` propres ; `go test ./... -count=1` = T1 (mêmes noms, mêmes
  verdicts ; les tests déménagés en `filmcorpus` comptés comme déplacés, nommés) ; T3 : digests obtenus
  identiques au témoin du lot 0 (`replay_equiv_obtenus/*.tsv`) ; PUIS régénération à révision constante
  des goldens des couches dont le gate rougit (porte à deux verrous, entrée de chronique par couche,
  révisions inchangées : diff des goldens = ligne `revision` réécrite seulement) ; T2 : mêmes révisions,
  mêmes goldens pour les couches non touchées (`source`, `profile`), `testdata/equivalence` intact ;
  archlint vert (`go test ./internal/archlint/`) ; lint go-api vert. Commit « D2 lot 2 ».

**Clos quand** : 2.12 passé, CI de la branche verte, entrée thought_log, point d'étape à l'utilisateur.

**Exécution du lot 2 (2026-10-29, worktree `LevelUp-wt-decouplage-d2`)** — témoins sous
`%TEMP%\claude\c--Users-Guillaume-Downloads-Scripts-LevelUp\77a8b90a-21a6-40a0-ac64-eb152f616862\scratchpad\temoins_d2_lot2_1\`.
- 2.1 (cadre historique de l'empreinte, D2-F). **Sur pièces** : les trois dépendances au chemin du module annoncées dans
  « Chiffres de départ » sont confirmées — (a) `jetons.go` garde le littéral de chaque jeton `STRING`, donc les chaînes d'import ;
  (b) `couches.go` nomme les racines `internal/games/halo_infinite/film/internal/...` relativement au module ; (c)
  `empreinteur.paquet` préfixe les fichiers d'un paquet importé par son dossier relatif au module, qui est aussi la ligne
  `paquet` des goldens de périmètre. Une quatrième entrée en découle sans être distincte : le nom haché d'un fichier EMBARQUÉ
  (`embed:<dossier du paquet>/data/...`, `damagetag`) reprend le préfixe de (c). Aucune autre : les 783 lignes de production du
  film qui citent `levelup/go-api` sont des lignes d'import (aucun littéral hors import, aucune directive `//go:` ne cite le
  module, les `//go:embed` sont relatifs) ; `chronique.go`, `porte.go`, `messages.go` ne hachent rien. `racineModule` vient de
  `ModuleDe(dossier de la couche)` (remontée au premier `go.mod`), appelé par `EmpreinteDeCouche` depuis les cinq gates
  `film/internal/*/rev_test.go` : mécanisme gardé tel quel. **Mécanisme retenu** (`film/revision/cadre.go`, neuf) :
  `cadreHistorique = "levelup/go-api/internal/games/halo_infinite/film"` ; `cheminModuleCourant` LU et non écrit
  (`path.Dir(reflect.TypeFor[Resultat]().PkgPath())` : le chemin d'import du paquet `revision` moins `/revision`), de sorte que
  le lot 3 (3.3) n'a AUCUNE constante à changer — une constante oubliée rendrait des empreintes fausses, le chemin lu suit le
  paquet ; le contrat « `revision` vit directement sous la racine `film` de la brique » est tenu par
  `TestCheminModuleCourantEstLaRacineDeLaBrique`. Jetons : un littéral chaîne (interprété ou brut) qui commence par le chemin
  courant de la brique, à la frontière d'un chemin (`/` ou fin), est haché sous `cadreHistorique` ; rien d'autre ne change
  dans le flux. Dossiers : les déclarations (racines de `CouchesRevisees`, oracle d'`equivalence_test.go`, goldens) restent
  en noms HISTORIQUES ; `Module.SurDisque(nom)` les résout dans le module courant, `Module.historique(rel)` ramène au nom
  historique ce que le disque et les imports donnent (identité sous `apps/go-api`, préfixe `internal/games/halo_infinite/film/`
  ajouté quand le module EST la brique, identité pour un module qui ne porte pas la brique). Écart assumé au libellé de D2-F
  (b) : les racines ne deviennent pas `internal/source` dans `couches.go` ; elles gardent leur nom historique, si bien que ni
  `couches.go` ni l'oracle ne changent au lot 3. Le test de périmètre résout son golden par `m.SurDisque(c.Racine)`.
  Tests neufs (`cadre_test.go`, internes au paquet) : `TestCadreHistoriqueDesImports` (3 sous-cas : source écrite sous
  `github.com/JGtm/LevelUp/libs/film` = même flux de jetons que sous l'ancien préfixe ; littéraux hors brique — `libs/gamefiles`,
  `libs/filmx`, `libs/film_ancien/x`, `internal/domain/...` — hachés tels quels ; traduction de dossier dans les deux sens sous
  les trois sortes de module), `TestCadreHistorique_LaBriqueDeplaceeRendLaMemeEmpreinte` (la même brique de fixture posée dans
  un module `levelup/go-api` et comme module `libs/film` : même `Resultat`, même `Perimetre`, amont, paquet importé hors couche
  et fichier embarqué compris ; une ligne de la table embarquée changée fait bouger l'empreinte), `TestCheminModuleCourantEstLaRacineDeLaBrique`.
  Mutations vues rouges puis remises : littéral non traduit (3 tests rouges), dossier non traduit (2), frontière de chemin
  retirée (1). **Gates** : `go test ./…/film/revision/ -count=1 -v` code 0, les 40 verdicts du témoin identiques + 6 neufs
  (3 tests, 3 sous-tests), les cinq empreintes calculées identiques au témoin (`source` 92ce10ae…, `profile` a520bb98…,
  `grammar` 8353c693…, `killsource` 3a3992d4…, `objectives` 2b45f688…) ; `go test ./internal/games/halo_infinite/film/... -count=1`
  code 0 (18 paquets `ok`, les mêmes qu'à T1, 73 s ; les cinq gates de couche verts sans réécriture) ; aucun fichier sous
  `testdata/` dans `git status` ; MD5 des 202 fichiers du témoin `empreintes.txt` identiques ; `go vet`, `gofmt -l` vides,
  `golangci-lint run ./…/film/revision/...` 0 issue ; `go test ./internal/archlint/ -count=1` vert (99 s).

## Lot 3 — Module `libs/film` + espace de travail + infrastructure

- [x] 3.1 `git mv apps/go-api/internal/games/halo_infinite/film libs/film` ; `libs/film/go.mod`
  (`module github.com/JGtm/LevelUp/libs/film`, `go 1.26.10`, `require` tiers à la main aux versions
  d'`apps/go-api`, aucun `require` de brique) ; `go.work` (`use ./libs/film`) ; `go.work.sum`.
- [x] 3.2 Réécriture des imports dans LevelUp (198 + 188 fichiers) : préfixe
  `levelup/go-api/internal/games/halo_infinite/film/` → `github.com/JGtm/LevelUp/libs/film/` (motif
  ancré : `filmprofile` ne bouge PAS) ; dans la brique : imports internes au même préfixe.
- [x] 3.3 Cadre historique (2.1) : la constante du module courant devient `github.com/JGtm/LevelUp/libs/film` ;
  T2 identique (goldens non réécrits, `git diff` vide).
- [x] 3.4 `archlint` : les 52 fichiers rerootés (`libs/film/...` dans les 20 constantes et les
  littéraux, parcours par `workspaceRoot` / `workspaceGoSources`), `racinesEspaceDeTravail` +=
  `libs/film`, `research_tag_test` (`dossiersInstruments`), `pas_de_lecture_sous_ai_test` (table),
  `no_repo_root_walk_test` (allowlist des helpers R2), `briques` (paquets = `libs/film`), `dockerfile_go_version_test`
  (quatrième `go.mod`), licence GPL (`cmd/server` ne lie pas `ooz` : la chaîne `film` → `himap` →
  `himodule` → `ooz` EXISTE maintenant pour tout paquet qui importe `replay/map_background` : le
  ratchet doit le voir et `cmd/server` doit rester propre — vérifier sur pièces ce que `cmd/server`
  tire).
- [x] 3.5 CI (`ci.yml`) : pas vet/test/lint/govulncheck de `libs/film` (vet aussi `-tags research`
  et `-tags 'research campagne_overlay'` : compilation des instruments), `go list -deps -test`
  garde, lots de couverture du module (D2-I ; `go-shards.txt` + `ci_go_shard.sh`), job « Film race »
  dans `libs/film`, filtre `croisee`, `cache-dependency-path` ×10, baseline JSONL de la brique +
  `check_test_baseline.sh` à deux modules (RETRAIT daté des 1 052 entrées du module go-api, qui
  renaissent dans la baseline de la brique).
- [x] 3.6 `Dockerfile` (COPY `libs/film/go.mod` + `go.sum`), `.github/dependabot.yml` (`/libs/film`),
  `Makefile` (cibles 201, 235, 308, 362 ; `libs-test` : délai mesuré pour `libs/film`),
  `apps/go-api/Makefile` (`tidy` saute `film`), `.golangci.yml` règle 220 (module), hooks `lefthook`
  vérifiés (globs `libs/**` déjà là).
- [x] 3.7 Docs (D2-J) : ADR 0041, ADR 0034, CLAUDE.md, `ARCHITECTURE_V6` FR/EN et `COMMANDS.md` si
  concernés ; commentaires de la brique qui citent un chemin de l'hôte : relevés, pas réécrits en
  masse (découverte à consigner, comme la 29 de D1).
- [x] 3.8 Gate : `go build ./...` dans `apps/go-api` ET `libs/film` ; `go vet` trois tags dans
  `libs/film` ; `go list -deps -test ./...` dans `libs/film` sans `levelup/go-api` ; T1 (union
  go-api + brique = mêmes noms modulo préfixe, mêmes verdicts) ; T2 identique ; T3 vert ; archlint
  vert ; lint des trois modules vert ; `make libs-test` vert ; `docker build` local du stage Go si
  Docker est disponible, sinon le job CI « Docker Build » fait foi. Commit « D2 lot 3 », CI verte.

**Clos quand** : 3.8 passé, CI verte, thought_log, point d'étape.

**Exécution du lot 3 (2026-10-29, worktree `LevelUp-wt-decouplage-d2`, base `915203a32`)** — témoins sous
`%TEMP%\claude\c--Users-Guillaume-Downloads-Scripts-LevelUp\77a8b90a-21a6-40a0-ac64-eb152f616862\scratchpad\temoins_d2_lot3\`.
- **Ouverture.** Le lot s'ouvre avec la CI du lot 2 (run 40808984883 sur `915203a32`) rouge sur le SEUL job `npm audit` (avis GHSA
  seroval publié le 2026-10-29, sans lien avec le code de la branche ; Deploy Pre-Check et gitleaks verts) : règle 9 de plan-execution,
  arrêt propre documenté, correctif en PR séparée (#112, `fix/npm-audit-seroval`, `apps/web/package-lock.json` seul). Au préalable
  (`git fetch origin` + `git merge origin/release/v8.1.0`) la release n'avait rien de neuf et la PR n'était pas fusionnée ; elle l'a été
  pendant le lot (`4c5597a13` sur `origin/release/v8.1.0`) et est embarquée par un `git merge origin/release/v8.1.0` juste avant le push
  de clôture.
- **Commits.** 3.1+3.2+3.3 `dab11f773` ; 3.4 `4d8cb2728` ; 3.5 `443bf1f8d` ; 3.6 `e36f1483b` ; 3.7 `d46cf951a` ; clôture (3.8, plan,
  journal, exemption de lint ci-dessous).
- 3.1 : `git mv apps/go-api/internal/games/halo_infinite/film libs/film` (aucun fichier non suivi laissé) ; `libs/film/go.mod`
  (`github.com/JGtm/LevelUp/libs/film`, `go 1.26.10`, `require` À LA MAIN des seules dépendances tierces importées par le module :
  `github.com/pelletier/go-toml/v2 v2.4.3` (un test de `killicon`) et `golang.org/x/sys v0.49.0` (`filmproc`), versions d'`apps/go-api`) ;
  `go.sum` = leurs quatre lignes copiées d'`apps/go-api/go.sum` ; `go.work` : `use ./libs/film` ; `go.work.sum` inchangé.
- 3.2 : imports réécrits PAR ANALYSE DES `ImportSpec` (outil `go/parser` jetable, pas de sed sur le texte) : 1 767 fichiers ; vérifié :
  les lignes ôtées, préfixe traduit, égalent les lignes ajoutées (multiset, lignes vides exclues) ; `filmprofile` intact ; puis `gofmt` et
  `goimports` (463 fichiers regroupés : un chemin `github.com/...` se sépare du standard). Les chaînes non-import qui citaient l'ancien
  chemin sont traitées en 3.4. Chemins de fichiers du film dans go-api (tests et outils) : `filmcorpus` x2 et
  `testutil.ReplayChroniclePath` (D2-L2-p, traités), `filmprofile` x2, `replaybuild` x9, `replaydiff`, `replayverite`, `sync` x4,
  `killcollector` x5, `cmd/replay-equiv`, défauts de `cmd/refresh_golden_fixture` et de `research/cmd_fermeture` (et son usage), quatre
  messages. La garde « le banc n'importe rien du décodeur » de `replayverite/dependances_test.go` testait la sous-chaîne
  `/games/halo_infinite/film` : elle teste le préfixe du module (sinon muette). Dans la brique, 17 échelles relatives de « .. » vers le
  dépôt raccourcies de quatre niveaux par outil (`Join` et littéraux), trois à la main (`named_derivation_unique` : racine du module ;
  `map_bounds` ; `world_object_precision_guard` : la garde balaie désormais la brique, clés relatives au module), et
  `usage_summary_families_guard_test.go` lit le valideur de LevelUp par `racineDepot`.
- 3.3 : **défaut du cadre trouvé et corrigé sans régénérer.** Après 3.2, `grammar`, `killsource` et `objectives` rougissaient
  (`TestChaqueRevisionEgaleSonGolden` et les trois gates de couche ; périmètres verts). Cause : le flux de jetons garde l'ORDRE des
  imports, et `gofmt` / `goimports` trient un bloc selon le chemin COURANT : `"levelup/..."` se rangeait entre `"cmp"` et `"slices"`,
  `"github.com/..."` se range avant `"hash"` et goimports le sépare du standard. Sur pièces : 25 sources de production réordonnées, dont
  15 dans des périmètres hachés. Correctif dans les sources, pas dans les goldens : ces 15 fichiers gardent l'ORDRE HISTORIQUE en groupes
  séparés par une ligne vide (chacun trié, `gofmt` et `goimports` propres) et portent une ligne de commentaire au-dessus de leur
  `import` ; `revision/cadre.go` documente la limite (« L'ORDRE DES IMPORTS N'EST PAS DANS LE CADRE »). Les trois tests de D2-L2-b
  reracinés : `racineAPI` devient `moduleDeLaBrique` (remontée au `go.mod` du paquet), goldens et dossier de `grammar` résolus par
  `Module.SurDisque` depuis leur nom historique. Gate : `go test ./revision/ -v -count=1` : 46 PASS, 0 FAIL, 0 SKIP, sans porte ;
  `go test ./... -count=1` du module vert ; `git status` : aucun fichier sous `testdata/` modifié (seuls les renommages).
- 3.4 (archlint, 64 fichiers touchés) : `racineBriqueFilm = "libs/film"` et `importBriqueFilm` (dans `film_layers_deps_helpers_test.go`),
  dont dérivent toutes les constantes nommées ; `racinesEspaceDeTravail` += `libs/film` ; `repertoireDuDepot` (inverse de
  `cheminImport`) ; pour les tables qui mêlent go-api et la brique, un DOUBLE REPÈRE sans recouvrement (`surDisque` /
  `cheminDeGardeRail` : sous `libs/` relatif à la racine du dépôt, sinon relatif à `apps/go-api`) ; `marcherAPIEtBriqueFilm` pour les
  balayages de tout go-api qui couvraient le décodeur ; `goSourcesUnder`, `balayerLaProduction`, `parcourirSourcesProduction` acceptent la
  brique. Ligne `film` de la table des briques = `libs/film` (récursif) ; `racinesDeLaBriqueFilm` lit la table (préfixe anticipé du lot 1
  retiré). **Ratchets que le déplacement rendait MUETS sans rougir, rebranchés** : feuilles `source` et `types` (« import du dépôt » =
  préfixe `levelup/go-api/`, qu'aucun import de la brique ne porte plus), `no_identity_bridge_outside_registry` et `no_player_index_identity`
  (`os.Stat` + `continue` sur le dossier disparu), `no_rewritten_slot_band` (`t.Skipf`), `no_french_label_literal` (une baisse n'est que
  journalisée), `no_raw_kill_scope_literal`, `no_raw_headshot_category_literal`, `no_hardcoded_film_cache_dirs`, `no_local_longest_run`,
  `killsource_recherche_sans_carte`, `no_ad_hoc_source_fingerprint`, `unarmed_family_literal`, `record36_grammaire`,
  `no_unbounded_film_loop` (deux tests), zlib, `no_repo_root_walk` (trois gardes : la brique est balayée). Registre des replis
  (production de la brique, `facts/fallback`, hors périmètres hachés) : `Site.Fichier` et `Appelants` relatifs à la RACINE DU DÉPÔT
  (`libs/film/...`, `apps/go-api/...`), contrat écrit dans `repli.go`. Tests go-api qui lisaient des fichiers du film par une échelle de
  « .. » (vus par `TestNoAdHocRepoRootLadderInTests`) : helper `testutil.CheminBriqueFilm` (+ son test), 12 sites. Trois fichiers à leur
  plafond de taille (`imagecle_oracle_n2_research_test.go` 500, `vehicules_v2_items_test.go` 533, `killcollector/positions_test.go` 557)
  rendent la ligne que goimports a ajoutée en retirant une ligne vide. **Licence GPL** : la chaîne `film/replay -> himap -> himodule -> ooz`
  N'EXISTE PAS en production (`go list -deps ./...` de `libs/film` : ni `himap` ni `ooz` ; seuls des tests tirent `himap`) et
  `go list -deps ./cmd/server | grep -E 'ooz|himap'` est VIDE (`licence_gpl.txt`) : pas de P0 ; `TestServeurNeLinkePasDeCodeGPL` et
  `TestLaChaineDeHimapAtteintOoz` verts (le parcours suit tout `libs/`). `no_title_package_in_analysis` : l'interdit direct et la fermeture
  voient `github.com/JGtm/LevelUp/libs/film/...` (auto-preuve mise à jour). `dockerfile_go_version` vert avec quatre `go.mod`.
  **Mutations jouées, rouge puis vert après retrait** (`mutations_archlint.txt`) : `TestBriquesImports` (import
  `levelup/go-api/internal/analysis` dans `libs/film/replay`), `TestFilmsourceEstUneFeuille` (import de `filmshell` dans
  `internal/source`), `TestCouchesDuDecodeurRespectentLeSensEtLeLieu` (R1, `profile -> grammar` par un fichier sous balise inerte ; sans
  balise le compilateur refuse le cycle), `TestNoPlayerIndexInFilmScope`, `TestAucunPredicatOSIsDansLePerimetreDuFilm`,
  `TestGrammarEtFactsNeJournalisentPas`, `TestPasDeSentinelleBruteHorsFilmproc` : sept ratchets. Gate : `go test ./internal/archlint/
  -count=1` vert (93 s) après 3.6 (les quatre ratchets qui lisent `ci.yml`, `go-shards.txt`, `check_test_baseline.sh` et
  `dependabot.yml` attendaient 3.5 et 3.6).
- 3.5 (CI). **Choix** : une directive `module <dossier>` dans `.github/ci/go-shards.txt` (source unique, pas de second fichier) ;
  `ci_go_shard.sh` prouve la partition module par module (`--verifier` : 365 paquets, 2 modules, 7 lots) et rend `--module <lot>`,
  `--modules`, `--lots-du-module <m>` ; un lot se joue depuis son module, ses pièces s'écrivent dans `apps/go-api` pour l'upload. Lots :
  go-api `persist-ops` (`./internal/persist`, `./internal/ops` : 6 + 57 s hors couverture au relevé du lot 0, gardés en lot dédié plutôt
  que d'alourdir `reste`), `platform`, `sync`, `reste` ; libs/film `film-grammar-lourd`, `film-grammar-reste` (mêmes `run:` / `skip:`),
  `film` (complément du module). Fusion : contrôle d'ensemble et baseline par module ; la couverture des lots `libs/film` n'entre pas dans
  le ratchet de go-api (affichée à part) : le total de go-api change mécaniquement (les paquets du décodeur sortent du dénominateur),
  comme à la découverte 26 de D1. Baseline : les 1 050 lignes (877 paires) des paquets du décodeur quittent `tests_pre_migration.jsonl`
  et forment `tests_libs_film.jsonl` à l'identique, préfixe réécrit (preuve : `diff` des deux transformations vide) ; bloc « RETRAIT DU
  2026-10-29 » dans `check_test_baseline.sh` (mécanisme existant : lignes retirées du fichier, datées dans l'en-tête), option
  `--baseline`, mode autonome à deux voies jugées chacune contre SA baseline. Vérifié : `libs/film` frais contre sa baseline vert
  (877 présents, 0 échec) ; go-api sur le JSONL du 2.12 : liste d'absences identique avec l'ancienne et la nouvelle baseline hormis la
  ligne du paquet du film (absences dues aux seuls tests `integration`, le JSONL du 2.12 étant sans balise). go-build : vet + tests des
  couches du décodeur sans CGo dans `libs/film` (deux OS, même sous-ensemble que l'ancien pas « unit, no CGo DB ») ; vet `research` +
  autonomie `go list -deps -test` (Linux) ; `campagne_overlay` NON joué (D2-L0-b, écrit dans le pas). Film race dans `libs/film`. Filtre
  `croisee` : `libs/film/{replay,internal/grammar,killicon/data}/**`. `cache-dependency-path` : `libs/film/go.sum` aux 10 occurrences.
  Lint : quatrième passage `libs/film` ; règle `path:` de `.golangci.yml` réécrite en `libs/film/(replay|internal/(facts|source|grammar|profile))/`
  et PROUVÉE ACTIVE (ligne de 243 caractères : exemptée sous `replay/`, signalée sous `filmcache/`, signalée sous `replay/` quand la règle
  est neutralisée ; `golangci_regle220_mutation.txt`). govulncheck `libs/film` ; dependabot `/libs/film`. Tests Vitest qui lisent des
  sources du décodeur (4 fichiers, 5 chemins) reracinés et verts (101 tests) ; la garde `ci_web_reads_go_test` voit aussi les chemins
  `libs/` (auto-preuve étendue).
- 3.6 : Dockerfile (COPY du `go.mod` / `go.sum` de `libs/film` ; Docker absent du poste : simulation `GOPROXY=off go mod download` verte,
  rouge sans le `go.mod` de film ; le job CI « Docker Build » fait foi) ; Makefile (`go-api-test`, repli vet de `go-api-lint`,
  commentaires) ; `libs-test` : boucle inchangée, le `go test ./...` de `libs/film` tient sous le `-timeout 600s` (90 s pour les trois
  briques) ; `scripts/gate-push.ps1` joue aussi `libs/film` contre sa baseline ; `apps/go-api/Makefile tidy` saute film (il importe
  `libs/gamefiles/mapvar`) ; lefthook et `go-vet-cgo.sh` bouclent déjà sur `libs/**`.
- 3.7 : ADR 0041 (troisième module, rang, D7 réécrit, D8 empreintes), ADR 0034 (amendement 2026-10-29 bis), CLAUDE.md, ARCHITECTURE_V6,
  COMMANDS, SYNC_GUIDE, WEAPONS (EN + FR), runbook des profils. Relevé des chaînes de la brique qui citent un chemin de l'hôte : D2-L3-h.
- 3.8 (gate final, sorties sous `temoins_d2_lot3\`) : `go build ./...` et `go vet ./...` verts dans `apps/go-api` et `libs/film` ;
  `go vet -tags research ./...` vert dans `libs/film` ; `research campagne_overlay` rouge comme à la base (mêmes 10 erreurs `bis2*`,
  `vet_research_overlay.txt`) ; `go list -deps -test ./...` de `libs/film` sans `levelup/go-api` : VIDE ; `go list -deps -test
  ./internal/analysis/...` sans `libs/film` : VIDE (`gate_build_vet.txt`). **T1** : `go test -json ./... -count=1` dans `apps/go-api` puis
  `libs/film` (codes 0 et 0) réduits et unis : 20 263 verdicts (19 897 pass, 366 skip, 0 fail), 218 paquets `ok` et 146 sans test ;
  contre le 2.12 traduit (`levelup/go-api/internal/games/halo_infinite/film` -> `github.com/JGtm/LevelUp/libs/film`, à la frontière de
  chemin : `filmprofile` ne se traduit pas) : 0 couple disparu, 1 apparu (`testutil.TestCheminBriqueFilmDesigneLeModuleDuDecodeur`,
  neuf), 0 changement de verdict, liste des paquets et verdicts de paquet identiques. **T2** : les 202 fichiers du témoin `empreintes.txt`
  (goldens de révision et de périmètre, 43 fichiers `equivalence`, faits, autres goldens) ont, sous `libs/film`, le MD5 de leur version au
  commit du 2.12 (`915203a32`) : 202/202 ; 6 diffèrent du lot 0, exactement les 6 régénérés au 2.12 (`t2_md5.txt`). **T3** :
  `replay-equiv -repo-root <racine de jonctions> -out-dir` : code 1 et « BILAN : 0 identique, 20 different » comme au lot 0 (références
  périmées, D2-L0-a), 437 s ; les 20 fichiers de digests obtenus ont des MD5 IDENTIQUES à `temoins_d2_lot0_avant\replay_equiv_obtenus\`
  (1 360 lignes ; `md5_obtenus_lot0.txt` = `md5_obtenus_lot3.txt`). archlint vert ; `golangci-lint run ./...` à
  `--new-from-merge-base=origin/release/v8.1.0` (et `origin/main`, comme la CI) : 0 issue dans `apps/go-api`, `libs/film`,
  `libs/gamefiles`, `libs/himap`, après exemption sur sa ligne `func` de la seule issue de dette déplacée (D2-L3-e) ; `make libs-test` vert
  (88 s) ; `bash scripts/ci_go_shard.sh --verifier` vert ; `bash -n` des deux scripts ; YAML des workflows, de dependabot et de lefthook
  relus par un analyseur (`yaml` de `apps/web`, actionlint et yamllint absents du poste). CI : voir la fin du journal de clôture.

## Lot 4 — Revue adversariale (une ronde) et corrections

- [x] 4.1 Contrat de lot écrit (6 lignes du skill `adversarial-review`) ; trois relecteurs Sonnet en
  parallèle, aveugles : L3 (anti-patterns) + « frontière de module » (imports, copies, ratchets
  rerootés, lint relâché), L6 (ce que les tests ne couvrent pas : tests déménagés/réécrits, tolérances,
  baseline), et une lentille « fil JSON et empreintes » (codec à plat, cadre historique, goldens).
- [x] 4.2 Triage par le pilote (P0–P3) ; P0/P1 corrigés avec test de non-régression ; P2 consignés en
  Découvertes ; aucune seconde ronde sauf P0/P1 corrigés (relecture des seules corrections).
- [x] 4.3 Gate : lot 3 rejoué (3.8) après corrections ; commit « D2 lot 4 » ; CI verte.

**Exécution du lot 4 (2026-10-29, worktree `LevelUp-wt-decouplage-d2`, base `6542417d9`)** — une ronde, trois relecteurs Sonnet aveugles
en parallèle, trois lentilles : L3 anti-patterns + frontière de module (imports, copies, ratchets rerootés, lint relâché), L6 ce que les
tests ne couvrent pas (tests déménagés ou réécrits, tolérances, baseline), fil JSON et empreintes (codec à plat, cadre historique, goldens).
Résultat : 0 P0, 0 P1, 3 constats P2 recevables et 1 remarque de documentation ; le reste = conditions qui tiennent (24 + 11 + 14 = 49).
Triage par le pilote, sur pièces :
- P2-1 CORRIGÉ : `brancherCompteursDuFilm()` appelé dans `NewServiceRegistry` sans test qui prouve le branchement (le relecteur l'a remplacé par
  `_ = 0` : `wire` et `archlint` restaient verts). Nouveau test `TestNewServiceRegistryBrancheLePuitsDesCompteursDuFilm`
  (`internal/api/wire/film_compteurs_test.go`) : puits remis à nil, construction par `NewServiceRegistry(&config.AppConfig{}, nil)` (chemin réel,
  sans DuckDB), puis `filmobs.AddInt` de 3 lu par `observability.LoadCounter`. **Mutation** : `registry.go:285` remplacé par `_ = 0` ->
  `--- FAIL: TestNewServiceRegistryBrancheLePuitsDesCompteursDuFilm ... observability a reçu 0, attendu 3 : NewServiceRegistry ne branche pas
  le puits` ; appel remis (`git diff registry.go` vide) -> `PASS`.
- P2-2 CORRIGÉ : le commentaire « L ordre de ces imports entre dans l'empreinte de la couche (revision/cadre.go) : ne pas regrouper » de
  `libs/film/internal/facts/fallback/compteur.go` était faux pour ce fichier (`facts/fallback` n'est dans aucun golden de périmètre ; le relecteur
  a regroupé les imports, `revision` est resté vert). Retiré de ce seul fichier ; les 14 autres sources hachées portent la contrainte et ne sont
  pas touchées. `libs/film : go test ./revision/` vert.
- P2-3 CORRIGÉ (doc, règle 17) : `textLabel` (`replaylabels/catalog.go`) disait « un membre par langue renseignée » ; depuis `TexteVersFilm` c'est
  chaque langue de la liste, vides comprises.
- Consignés sans correction : D2-L4-a à D2-L4-d (Découvertes).
Gate : `go build ./...`, `go vet ./...` ; `go test ./internal/api/wire/ ./internal/games/halo_infinite/replaylabels/ -count=1` ; `libs/film :
go test ./revision/ ./internal/facts/... -count=1` ; `go test ./internal/archlint/ -count=1` (112 s) ; `gofmt -l` vide ; `golangci-lint run
--new-from-merge-base=origin/release/v8.1.0 ./...` : 0 issue dans `apps/go-api` et dans `libs/film` (avec `--config ../../apps/go-api/.golangci.yml`,
comme la CI ; sans cette config, 30 issues de dette gelée, sans rapport avec le lot). Rejeu complet du lot 3.8 (T1 à T3) non refait : aucun
code de production n'a changé, seuls un test et deux commentaires.

## Lot 5 — Preuve de non-régression et clôture

- [x] 5.1 T4 : `go run ./cmd/replay-corpus-gate --reference=base --base=<sha du lot 0>` (plafond
  mémoire par défaut, racine de travail jetable, 19 témoins) : code 0, rapport JSON conservé en témoin.
- [x] 5.2 T1, T2, T3 recapturés dans `temoins_d2_lot5_apres/` et comparés (diff vide modulo préfixe).
- [x] 5.3 `go test -tags=integration -p 1 ./internal/sync/... ./internal/persist/...` (imports
  réécrits dans `persist` et `sync`) : code de sortie 0.
- [x] 5.4 `gate-push` commande par commande (le `$(MAKE)` GnuWin32 du poste échoue : découverte 28/33
  de D1), web : l'hypothèse « non touché » est tombée (D2-L3-d), typecheck, lint et Vitest web rejoués (cf. Exécution du lot 5).
- [x] 5.5 Docs et registres : `.ai/REGISTRE_REPORTS.md` (une entrée par `[!]` et par découverte
  reportée), addendum « D2 exécuté » dans l'audit, mémoire du pilote, thought_log « lot 5 ».
- [x] 5.6 (fait par le pilote après la fusion, 2026-10-29 : `disk-hygiene.ps1 -Lot d2 -Apply`) Caches : `scripts/disk-hygiene.ps1 -Lot d2 -Apply` après la CI.
- [x] 5.7 (PR #113, 36 contrôles verts, fusion squash par le pilote le 2026-10-29 ; branche et worktree supprimés ensuite) PR vers `release/v8.1.0`, CI verte, fusion squash (titre Conventional Commits), branche et
  worktree supprimés, dépôt principal remis au niveau.

**Clos quand** : 5.7 fait. D3 s'ouvre ensuite (plan distinct, à écrire depuis l'audit et les
découvertes de D1 et D2).

**Exécution du lot 5 (2026-10-29, worktree `LevelUp-wt-decouplage-d2`, HEAD de départ `32f118cd9`)** — témoins sous
`%TEMP%\claude\c--Users-Guillaume-Downloads-Scripts-LevelUp\77a8b90a-21a6-40a0-ac64-eb152f616862\scratchpad\temoins_d2_lot5_apres\`.
- 5.1 T4 : `go run ./cmd/replay-corpus-gate -reference=base -base=a5bae50b4 -parc-root <dépôt principal> -source-root <worktree> -work-root <scratchpad>\gate_work_d2
  -json corpus_gate.json`, plafond mémoire par défaut. **Première passe** (sans `-allow-missing`, base cuite à froid, 11 min 31 s) : 18 témoins `ok` (0 gain, 0 perte,
  0 changement, banc de vérité `ok` sur les 18) et UN témoin **ABSENT** : `f75e7053` (famille `koth_collines`), « manifeste du film f75e7053 absent du parc
  (`data/cache/film_manifests/f75e7053.json`) » et « match absent du registre » à l'export des faits ; couverture incomplète = code 4 (`go run` le rend 1 ; la
  ligne finale du journal dit `exit status 4`). Rapport : `corpus_gate_run1_sans_allow_missing.json`. **Seconde passe** avec `-allow-missing` (base relue du cache,
  5 min 44 s) : **code 0**, `couverture_incomplete=false`, mêmes 18 témoins identiques, 1 ABSENT nommé (`f75e7053`) ; rapport `corpus_gate.json`. Le témoin absent est une
  propriété du parc local (film jamais présent), pas du diff : la base ne peut pas non plus le cuire. Réserve : le serveur de développement du dépôt principal
  (`apps/go-api/tmp/server.exe`) tenait `metadata.duckdb` en écriture pendant le gate ; l'export des faits a prévenu « résolution des noms EN dégradée (map_name brut
  seul) » ; les faits sont exportés une seule fois et servent aux deux côtés (HEAD et base), donc la dégradation est la même des deux côtés. Racine de travail supprimée
  après usage ; le cache de la base (`data/cache/replay_corpus_gate_base/a5bae50b4…`) reste dans le parc, ignoré par git.
- 5.2 T1/T2/T3.
  - **T1** : `go test -json ./... -count=1 -timeout 1200s` dans `apps/go-api` (code 0) puis `libs/film` (code 0), redirection binaire par `cmd` (jq rend des fins de
    ligne CRLF sous Windows : retirées avant tri). Union : 20 264 verdicts (19 898 pass, 366 skip, 0 fail), 218 paquets `ok`, 146 sans test. Contre le lot 3 : **un seul
    couple de plus**, `levelup/go-api/internal/api/wire TestNewServiceRegistryBrancheLePuitsDesCompteursDuFilm` (le test du lot 4) ; paquets et verdicts de paquet
    identiques. Contre le lot 0 (traduit : `levelup/go-api/internal/games/halo_infinite/film` -> `github.com/JGtm/LevelUp/libs/film`) : 70 couples disparus (EXACTEMENT la
    liste du 2.12 : 15 `analysis/digest`, 39 `filmproc`, 3 `canonical`, 3 `equipenergy`, 2 `filmshell`, 3 `filelock`, 5 tests de `replay` partis en `filmcorpus`), **0 nom de
    test disparu** (chaque nom disparu d'un paquet existe dans un autre), 220 apparus (les 218 du 2.12 + le test du lot 3 + celui du lot 4), 0 changement de verdict, 0 fail,
    aucun pass -> skip.
  - **T2** : MD5 des 202 fichiers du témoin `empreintes.txt` (goldens de révision et de périmètre, `equivalence`, faits, autres goldens) sous `libs/film` : **202/202 identiques
    au lot 3** ; 6 diffèrent du lot 0, exactement les 6 régénérés au 2.12 (`*_rev.golden` et `*_perimetre.golden` de `grammar`, `killsource`, `objectives`). `go test ./revision/ -v
    -count=1` : 46 PASS, 0 FAIL, 0 SKIP, lignes (empreintes comprises) identiques à `temoins_d2_lot3\revision_v.txt`.
  - **T3** : `replay-equiv` avec `-repo-root` = racine de jonctions (`data`, `config` du dépôt principal ; `libs` du worktree : le dépôt principal est encore à l'ancienne
    arborescence et l'outil lit les références sous `<repo-root>/libs/film`, cf. D2-L3-i) et `-out-dir` : code 1, « BILAN : 0 identique(s), 20 different(s), 0 ecarte(s), 0 echec(s),
    0 illisible(s) » comme au lot 0 (références périmées, D2-L0-a), 6 min 37 s ; **les 20 fichiers de digests obtenus ont des MD5 identiques** à `temoins_d2_lot0_avant\replay_equiv_obtenus\`
    (1 360 lignes ; `md5_obtenus_lot5.txt`). Racine de jonctions retirée par `rmdir` (cibles intactes).
- 5.3 `go test -tags=integration -p 1 ./internal/sync/... ./internal/persist/... -count=1 -timeout 1800s` : **code 0** (`$LASTEXITCODE`), 16 paquets `ok`, 3 sans test, aucune ligne
  `--- FAIL`, 6 min 25 s (`integration.log`).
- 5.4 Gates. **L'hypothèse « web non touché » tombe** (D2-L3-d : 4 tests Vitest reracinés). `node_modules` de `apps/web` = jonction vers celui du dépôt principal (vérifiée) ;
  `node_modules\.tmp` supprimé ; `npm run typecheck` code 0 (16 s) ; `npm run lint` code 0, 0 erreur, 27 avertissements (les mêmes que D1, découverte 35) ; `npm run test -- --run`
  (Vitest) code 0 : 1 026 fichiers passés, 5 sautés, 10 570 tests passés, 23 sautés (390 s). `gate-push` commande par commande (`make gate-push` n'est pas lancé tel quel : D1, découvertes 28 et 33) :
  (1) `golangci-lint run --timeout 5m --new-from-merge-base=origin/main` dans `apps/go-api` : code 0, « 0 issues » ; (2) le même ratchet dans `libs/film`, `libs/gamefiles`, `libs/himap` (`--config ../../apps/go-api/.golangci.yml`) : codes 0, 0 issue chacun ; (3) `make libs-test` (depuis git-bash, `make` GnuWin32 ; sous `cmd` il est introuvable, code 9009) : code 0 ; (4) `npm run typecheck`, `npm run lint` : codes 0 (voir plus haut) ; (5) la baseline de tests, produite comme `scripts/gate-push.ps1` (`go test -tags=integration -count=1 -timeout=600s -p 1 -json ./...` dans `apps/go-api` puis `libs/film`, codes 0 et 0, aucun `fail`) et jugée par `scripts/check_test_baseline.sh tests --from-jsonl` : `apps/go-api` contre `tests_pre_migration.jsonl` (8 359 tests de baseline, 18 345 au run) code 0 « Tous les tests baseline présents », `libs/film` contre `tests_libs_film.jsonl` (877 de baseline, 3 842 au run) code 0 ; 26 min environ de calcul pour les deux modules. Écart de méthode, consigné : la première capture du JSONL par PowerShell 5.1 (`$out = & go test ...`, console cachée) a ABÎMÉ les sous-tests dont le nom porte des accents ou des flèches (« 58 absents » dans `service`, `analysis`, `sync` : faux négatif, tous les noms sont dans le run binaire) ; le JSONL jugé est celui écrit par redirection binaire de `cmd` (comme pour T1). Le script `gate-push.ps1` capture comme la première méthode : le même faux négatif y est possible selon la page de code de la console (D2-L5-a).
- 5.5 Docs et registres : (a) `.ai/REGISTRE_REPORTS.md` : section « découplage D2 » (17 lignes, dont D2-L5-a/b/c, une par découverte reportée qui appelle une action ; aucun `[!]` ni `[~]` dans le plan ; D2-L1-b/c,
  D2-L2-a/b/c/f/g/h/i/k/l/m/o/p/q/r, D2-L3-d/i/j sont des constats clos ou traités, non reportés) ; (b) addendum « D2 exécuté le 2026-10-29 » dans `.ai/AUDIT_DECOUPLAGE_BRIQUES_2026-10-27.md` ;
  (c) grep de `internal/games/halo_infinite/film` : `CLAUDE.md`, ADR 0041 (ligne du cadre historique, volontaire), `ARCHITECTURE_V6` EN/FR et `COMMANDS` EN/FR ne le citent plus comme emplacement
  actuel ; ADR 0034 : la phrase de D-1 qui donnait l'emplacement est corrigée, restent les amendements du 2026-10-29 (qui décrivent le déplacement) et l'historique de la section D-10 bis et de
  `0034-annex-history.md` (chemins d'époque, volontaires) ; `docs/CHANGELOG.md` et sa jumelle FR gardent la ligne de l'historique de la v7 ; les ADR sont EN seul (aucune jumelle à écrire),
  `ARCHITECTURE_V6` et `COMMANDS` ont leurs deux versions à jour ; (d) plan relu : aucune case `[ ]` hors 5.6 et 5.7, aucun `[!]` ni `[~]` ; (e) entrée `.ai/thought_log.md`.
- 5.6 laissé au pilote (`disk-hygiene` après la CI) ; 5.7 (PR, fusion) au pilote.

- 2.2 (`filmobs`). **Sur pièces** : 11 appels `AddInt` / `IncCounter` dans 9 fichiers de production (`replay` x8, `filmcache/write.go`) — le relevé de
  départ en annonçait 14 — et 4 fichiers de test qui lisaient `observability.LoadCounter` ; deux tests de `replay`
  (`occupants_equipe_arbitree_test`, `positions_porte_depart_test`) lisaient la carte `expvar` « levelup » par leur helper `compteurExpvar`
  (carte créée par `observability`, absente de la brique) : helper supprimé, ils lisent `filmobs.Load`. **Fait** : `film/filmobs`
  (`filmobs.go`, `filmobs_test.go`) : `AddInt`, `IncCounter`, `Load`, `Snapshot`, `SetSink` ; compteurs en `sync.Map` d'`atomic.Int64`, puits en
  `atomic.Pointer` (variable non exportée), même sémantique qu'`observability.AddInt` (création au premier usage, delta négatif admis) ; 13 fichiers du film
  réécrits (9 prod, 4 tests) + 2 tests `replay` ; les noms de compteurs ne changent pas. **Puits câblé** : `wire.NewServiceRegistry` appelle
  `brancherCompteursDuFilm()` (`internal/api/wire/film_compteurs.go`, `filmobs.SetSink(observability.AddInt)`) — seul point de démarrage du serveur
  (un seul appelant, `api/server_apiv1.go`), test `film_compteurs_test.go`. **Aucun binaire à câbler** : aucun lecteur de compteur du film hors tests
  (`cmd/levelup` ne lit que `killsource_*` et `highlight_events_parse_anomaly_total`, publiés côté LevelUp) ; les deux tests LevelUp qui lisaient un compteur
  publié par `replay.PublierCleInconnue` (`replaybuild/cle_inconnue_test.go`, `sync/killcollector/cle_inconnue_test.go`) lisent `filmobs.Load` pour ces
  noms-là. Ratchets : `film_layers_deps_test` classe `filmobs` hors couche ; `film_exported_vars`, `filmdec_package_vars`, `film_no_slog_in_layers` verts.
  Tolérance `observability` retirée (13 fichiers -> 0). Commit `934616aa4`. Gates : `go build`, `go vet`, `go vet -tags research` propres ; `go test ./... -count=1`
  = 211 paquets `ok` + `filmobs` (2 rouges de départ corrigés : classement de `filmobs`, helper `compteurExpvar`) ; `golangci-lint --new-from-rev` 0 issue.
- 2.3 (`playerposition`, `objectiveevent`). `git mv internal/domain/{playerposition,objectiveevent}` -> `film/` ; 19 fichiers réécrits (5 dans le décodeur :
  `decfilm`, `facts/objectives` x2, `grammar/positions` x2 ; 14 côté LevelUp : `cmd/diag_weapons_v3` x3, `api/handlers` x3, `api/wire` x1 test, `platform/duckdb` x4,
  `port` x2, `service` x1) ; aucun import depuis `internal/analysis` (fermeture verte). Ratchet : `film_layers_deps_test` classe les deux paquets hors couche. ADR 0034
  (lignes ~393-397) et `SYNC_GUIDE` (EN, FR, ligne 153) mis aux nouveaux chemins ; commentaires de chronique et lignes de goldens laissés (historique ; goldens à
  régénérer au 2.12). Tolérances `domain/playerposition`, `domain/objectiveevent` retirées. Commit `6b373cc88`.
  **TESTS ROUGES ATTENDUS (D2-L2-a, régénération au 2.12)** : `grammar` : `TestGrammarRevSuitLaGrammaire` ; `facts/objectives` : `TestObjectivesRevSuitLaSortie` ;
  `revision` : `TestChaqueRevisionEgaleSonGolden/{grammar,objectives}` et `TestPerimetreDeChaqueCoucheEgaleSonGolden/{grammar,objectives}`. `source`, `profile`
  et `killsource` restent verts. Les mêmes rouges restent après 2.4, 2.5 et 2.6 (aucun autre ne s'ajoute).
- 2.4 (`identity`). `git mv games/canonical/film_identity.go` -> `film/identity/identity.go` et `film_identity_test.go` -> `identity/identity_test.go` ; clause `package identity`,
  en-tête de paquet réécrit (la propriété du contrat est au décodeur) ; noms exportés inchangés. `canonical` ne perd que ces symboles (aucun autre fichier de
  `canonical` ne s'en servait : `go build ./...` vert). 16 fichiers réécrits (`canonical.X` -> `identity.X`) : 15 dans `film/replay` (7 prod, 8 tests) +
  `service/replayview/convert_identity.go`. Ratchets : `film_layers_deps_test` classe `identity` hors couche ; `no_unregistered_fallback_test` (le dossier entre dans
  son périmètre) : `FilmTableCounts.Fallback` est un COMPTE publié, exemption datée 2026-10-29 dans `replisDeLEcrivainDuJeu` ; `film_pont_identite_test` et
  `no_identity_bridge_outside_registry_test` ne citent pas le chemin : inchangés. ADR 0034 amendé (section « Amendment of 2026-10-29 », texte demandé). Tolérance
  `games/canonical` retirée (15 -> 0). Commit `f01c07f4d`.
- 2.5 (`filmshell`, `equipenergy`). `git mv` des deux dossiers sous `film/` ; 29 fichiers réécrits (décodeur : `grammar` x7 dont `weaponscan` et `weaponv3`, `replay` x8,
  `research/cmd_rdata_weapon_scan` x4 ; LevelUp (10) : `games/weapons`, `games/halo_infinite` (adapter d'URL d'assets), `platform/duckdb`, `replaybuild` x2, `sync/killcollector` x2,
  `persist` (test), `cmd/diag_film`, `cmd/equipenergy-build`). Table `briques` : la ligne `film` n'a plus qu'une racine. Ratchets adaptés sans les affaiblir :
  `film_layers_deps_test` (deux paquets hors couche), `unarmed_family_literal_test` (chemin), auto-preuve du test de fermeture d'`analysis` ; deux EXCLUSIONS
  datées et nommées pour `filmshell` (il était hors périmètre sous `games/weapons`) : `film_exported_vars_test` (`catalogueArmesHorsRatchet`, 15 variables exportées
  modifiables) et `no_raw_film_bytes_outside_source_test` (`binary.BigEndian` de `hexToUint64`). **Fermeture d'`analysis`** : re-mesurée, elle ne contient NI `games/weapons`
  NI `filmshell` (`go list -deps -test ./internal/analysis/...`) : `fuitesTransitivesAnalysis` reste VIDE (D2-L1-c confirmée) ; `paquetsInterTitres` ne cite pas `filmshell`.
  `equipenergy_test` lisait le parc par un chemin relatif à `..` x6 : un `..` de plus (à remplacer par le helper R2 du 2.11). Docs `WEAPONS` (EN, FR), `SYNC_GUIDE` (EN, FR)
  et ADR 0034 mis au nouveau chemin. Tolérances : aucune entrée n'existait pour ces deux paquets (ils sont DANS la brique). Commit `f69110026`.
- 2.6 (`filmproc`, `filelock`, `atomicfile`). `git mv internal/filmproc` -> `film/filmproc` (20 fichiers). **Décision D2-D** : `platform/filelock` n'a QUE `filmproc` pour
  importeur -> DÉPLACÉ (`git mv` -> `film/internal/filelock`, 4 fichiers) ; `platform/atomicfile` a 11 autres fichiers importeurs de production (`notify`, `ops`, `dbprofiles`, `settings`,
  `replaybuild` x2, `scheduler`, `replayartifacts` x2, `cmd/replay-corpus-gate`, `pkg/duckdbbackup`) -> COPIÉ sous `film/internal/atomicfile` (3 fichiers dont 2 tests, même nom
  de paquet, en-tête qui nomme la copie jumelle et la règle 6) + `archlint/copies_identiques_test.go` (neuf : chaque `.go` de l'original existe dans la copie et porte le
  même contenu octet pour octet à partir de la ligne qui suit la clause `package`, aucun fichier de plus ; auto-preuve ; mutation d'une ligne de code vue rouge puis remise).
  `filmcache/write.go` et `filmproc/solo.go` basculent sur les paquets internes. 98 fichiers importaient `filmproc` (10 paquets LevelUp de production, le film et ses tests,
  `research` compris) : réécrits ; 23 fichiers dont les commentaires citaient `internal/filmproc` mis au nouveau chemin. Ratchets : `film_layers_deps_test` (3 paquets
  hors couche), `film_tri_total_test` (racine `filmproc` redondante retirée), `no_accidental_goos_suffix_test` (quatre chemins), `no_unbounded_film_loop_test` (chaînes de chemin),
  `film_slog_contexte_test` (exclusion datée `horsPerimetreDeJournal` pour `filmproc` et `atomicfile`, cf. D2-L2-d). Ni `.golangci.yml`, ni `Makefile`, ni `.github` ne citaient
  `filmproc`. Tolérances `filmproc` (71 -> 0) et `platform/atomicfile` (1 -> 0) retirées. Vérifié aussi `GOOS=linux go vet` des paquets déplacés (propre).
- Bilan 2.2 à 2.6 : tolérances restantes dans `tolerancesFilm` (11 imports, 83 fichiers) : `domain/highlightevent` 21 (2.7), `domain/equipmentusage` 4 (2.8), `i18n` 7 (2.9),
  `testutil` 21, `domain/title` 14, `games/mappings` 7, `games/weapons` 3, `testfixtures` 2, `domain/replaydoc` 2, `domain` 1, `analysis/digest` 1 (2.10 / 2.11). Dernier `go test ./... -count=1` :
  212 paquets `ok`, soit les 212 du témoin T1 moins `grammar`, `facts/objectives` et `revision` (rouges attendus) plus `filmobs`, `identity` et `internal/atomicfile` neufs ; paquets déplacés
  comptés comme déplacés : `filmshell`, `equipenergy`, `filmproc`, `internal/filelock`.
- 2.7 (`HighlightEvent`). **Écart au libellé, décidé sur pièces (D2-L2-i)** : le type n'est PAS posé dans `film/types` mais dans une feuille
  neuve `film/highlightevent` (mêmes champs, mêmes `EventType*`, aucun import), classée hors couche dans `film_layers_deps_test`. Raison : `film/types`
  est haché par ses octets dans l'empreinte de `source` (golden de périmètre `source` : `paquet internal/games/halo_infinite/film/types`) ; y ajouter un fichier
  aurait fait rougir `source`, qui doit rester vert, et aurait exigé une régénération du golden des formes `types/testdata/shapes.golden`
  (`TestTousLesTypesDuPaquetSontFiges`). Le paquet porte le même nom que celui de LevelUp : les 21 fichiers du film (6 prod dont `research/cmd_fermeture/v2_chunk3.go`,
  15 tests) ne changent que leur ligne d'import ; `killsource/rev.go` (commentaire de périmètre) mis au nouveau chemin. Adaptateur de titre
  `internal/games/halo_infinite/film_highlight_events.go` : `ParseHighlightEvents` (appelle `decfilm.ParseHighlightEvents`, `nil` reste `nil`, erreur
  transmise), `HighlightEventDepuisFilm` (pure) ; tests `film_highlight_events_test.go` (mêmes champs, types et tags par réflexion ; aller-retour champ par
  champ ; `EventType*` égaux ; nil et ordre). Appelants basculés : `sync/collect.go`, `sync/convergence_backfill_events.go`, `sync/engine_highlight_events.go`,
  `ops/medal_feed_backfill.go` (+ un message de test). **Sur pièces** : `decfilm.Kill` (= `killsource.Kill`) ne porte pas `highlightevent` (le fil est
  interne à `buildFeed`) ; `sync/killcollector` ne lit que les constantes `EventType*` pour du SQL (côté LevelUp) ; `cmd/killsource` n'importe pas le paquet :
  aucune autre conversion. `internal/games/halo_infinite` importait déjà `internal/analysis`, sans cycle (`go build ./...` vert). Tolérance
  `domain/highlightevent` retirée (21 -> 0). Rouges : ceux attendus, plus `killsource` (`TestKillsourceRevSuitLaSortie`, `revision/.../killsource` x2) comme
  annoncé. Commit `86b7924a1`.
- 2.8 (familles d'équipement). `replay/usage_summary_bilan.go` (neuf) : liste ordonnée du bilan, table des familles à pièce engendrée,
  `EquipmentOutcomeFamilies()` et `UsageFamilySpawnsPiece()` exportées (le garde-rail de LevelUp les compare), en-tête qui nomme la copie jumelle et la
  règle 6 ; les six clés de `usage_summary_families.go` redeviennent des littéraux (+ `usageFamilyRepairField`, exigé par goconst), `usage_summary_outcomes.go`
  lit la copie locale ; 2 tests réécrits (le recollement au manifeste parcourt la table locale). Garde-rail `domain/equipmentusage/jumeau_film_test.go` :
  listes égales (ordre compris), prédicat égal sur l'union des familles des deux côtés + clés hors bilan, clé du mur ; mutation (famille renommée, famille
  ajoutée à la table) vue rouge puis remise. **Code mort retiré (règle 7)** : `EquipmentFamilyGrapple`, `EquipmentFamilyThruster`, `EquipmentFamilySensor`,
  `UsageFamiliesSpawningPiece` de `domain/equipmentusage`, dont le décodeur était le seul lecteur ; `EquipmentFamilyWall` gardé (un test de `sessionusage`
  le lit). Plafond `plafondSurfaceReplay` 312 -> 314 (justification datée). Tolérance `domain/equipmentusage` retirée (4 -> 0). Commit `464f7feb8`.
- 2.9 (`replay.Text`, `flatjson`). `replay/text.go` (neuf) : `type Text map[string]string`, `Clone`, `Empty`, `Compare` (union des codes triés : même ordre
  que `i18n.Text.Compare` sur les codes de la liste) — aucune liste de langues, aucun `Pick`. `film/internal/flatjson` (neuf, `flat.go` + `flat_test.go`) :
  copie de `i18n/flat.go`, fonctions génériques `MarshalFlat[T ~map[string]string]` / `UnmarshalFlat[T]` (le champ texte est trouvé par le type `T`,
  `textType` de l'original) ; **deux différences assumées, écrites dans l'en-tête** : l'écriture émet les CLÉS du texte (triées), la lecture reconnaît une
  langue par sa FORME (nom hors champs du parent, deux ou trois minuscules ASCII, valeur chaîne). 5 fichiers de production de `replay` réécrits (8 champs :
  `Label`, `WeaponLabel`, `VehicleLabel`, `VehicleFamilyInfo.Label`, `CalloutZone`, `ObjectiveObjectLife`, `VehicleWeapon`), 2 tests (fixture à deux codes
  écrite à la main ; tests neufs du contrat : la brique écrit les langues du texte, relecture par la forme). Conversions dans
  `replaylabels/texte.go` : `TexteVersFilm` (chaque langue de la liste, vide comprise, aucune autre : le fil d'`i18n.MarshalFlat`) et `TexteDepuisFilm`
  (langues de la liste seulement : la lecture d'`i18n.UnmarshalFlat`). Sites LevelUp convertis : producteurs `replaylabels/catalog.go`,
  `mapcatalog/callouts_entree.go`, `service/replay_vehicle_weapons.go`, `cmd/mapcallouts-build/main.go` ; lecteurs `replayview/convert_{document,map,objectives,vehicles}.go`,
  `service/tactical_callouts.go`, `service/squadagg/{squad_formes,vehicle_labels}.go`, `cmd/mapcallouts-build/decoupe{,_masque}.go`, `cmd/mapplafond-mesure/catalogue.go`
  (13 fichiers de production) ; 15 tests LevelUp (dont `replayview/parity_test.go`, cf. D2-L2-j). `callouts_catalog.go` ne lit pas le CSV (c'est
  `cmd/mapcallouts-build`, côté LevelUp) : il relit le catalogue JSON, aucune langue en dur. En-tête d'`i18n/flat.go` : la copie jumelle y est nommée.
  Garde-rails : (a) `replaylabels/fil_jumeau_test.go` (100 sous-cas : 6 types, 10 dispositions dont les cinq de `document_labels_json.go` réécrites en
  `i18n.FlatLayout`, 10 textes — nil, vide, complet, langues vides, partiels, hors liste, échappements) : mêmes octets, relecture croisée dans les deux
  sens, membre du contrat `i18n.FlatMember` ; mutations (disposition changée, `OmitEmpty` ignoré) vues rouges puis remises ; (b) `replay` vert sans
  régénération (`document_shape`, goldens d'assemblage, artefact réel `labels_flat_artefact_1d2a26d8.json` relu et réécrit à l'identique) ; (c) T3 :
  `replay-equiv -repo-root <dépôt principal> -out-dir` (437 s, code 1 et « 20 différents » comme au lot 0, références périmées D2-L0-a) : les 20
  `replay_equiv_obtenus/*.tsv` IDENTIQUES au témoin du lot 0 (MD5), étape `artifact` comprise. Ratchets i18n (`no_locale_literal`, `i18n_locale_constant`,
  `i18n_text_sql`, `i18n_langue_suffixe`) verts. Plafond `plafondSurfaceReplay` 314 -> 315 (`replay.Text`, justification datée). `flatjson` classé hors
  couche. Tolérance `i18n` retirée (7 -> 0). Commit de clôture du présent bloc.
- Gates 2.7 à 2.9, à chaque item : `go build ./...`, `go vet ./...`, `go vet -tags research ./…/film/...` propres (au 2.9 aussi `-tags research`,
  `-tags integration`, `-tags gamefiles` sur tout `./...`) ; `go test ./... -count=1` : 211, 212 puis 213 paquets `ok` (au 2.9 : les 212 du témoin T1 moins `grammar`,
  `facts/objectives`, `facts/killsource`, `revision` rouges attendus, plus les paquets neufs `filmobs`, `identity`, `internal/atomicfile`, `domain/equipmentusage`
  (premier test, 2.8), `internal/flatjson` (2.9) ; paquets déplacés comptés comme déplacés) ; `gofmt -l` vide ; `golangci-lint --new-from-rev=HEAD~1` 0 issue.
- Bilan 2.7 à 2.9 : tolérances restantes dans `tolerancesFilm` (8 imports, 51 fichiers) : `testutil` 21, `domain/title` 14, `games/mappings` 7,
  `games/weapons` 3, `testfixtures` 2, `domain/replaydoc` 2, `domain` 1, `analysis/digest` 1 (2.10 / 2.11). Rouges attendus (D2-L2-a, régénération unique au
  2.12) : `grammar` `TestGrammarRevSuitLaGrammaire`, `facts/killsource` `TestKillsourceRevSuitLaSortie`, `facts/objectives` `TestObjectivesRevSuitLaSortie`,
  `revision` `TestChaqueRevisionEgaleSonGolden/{grammar,killsource,objectives}` et `TestPerimetreDeChaqueCoucheEgaleSonGolden/{grammar,killsource,objectives}` ;
  `source`, `profile`, `types` (golden des formes) verts.

- 2.10 (reliquats de production). **Sur pièces** (`grep -rn 'testutil\.\|title\.\|mappings\.' film --include='*.go' --exclude='*_test.go'`) : 5 occurrences, toutes en
  COMMENTAIRE (aucun import de production ne subsiste : `grep` des imports `levelup/go-api/internal/...` hors du film en production = vide) :
  `equipenergy/equipenergy.go` (`title.PathResolver.EquipmentEnergyPath`, relevée ici : le lot 0 n'en comptait que trois), `filmcache/filmcache.go`
  (`title.FilmShortMatchID`), `replay/document_chronicle.go` (`testutil.ReplayChronicleVersions`, 2 mentions), `replay/map_objectives.go`
  (`mappings.ObjectiveModeEntry.PointsOnly`). Quatre fichiers réécrits pour dire la notion (règle 17) : « chemin que l'hôte résout pour le titre »,
  « les 8 caractères hexadécimaux qui précèdent le premier tiret », « l'extracteur des versions déclarées (helper de test côté hôte) », « le champ
  `points_only` de la table des modes d'objectif du titre (objective_roles.toml) ». Aucune règle D2-D à appliquer (pas d'import). Commit `05bbbf8cf`.
- 2.11 (tests couplés restants). **Le « 51 fichiers » de `tolerancesFilm` compte des COUPLES (fichier, import)** : 43 fichiers distincts (8 imports :
  `testutil` 21, `domain/title` 14, `games/mappings` 7, `games/weapons` 3, `testfixtures` 2, `domain/replaydoc` 2, `domain` 1, `analysis/digest` 1) ; 11 hors de
  `replay` (grammar 8, killsource 1, profile 1, killicon 1) et 32 dans `replay`. Les 19 fichiers qui citent `Projects\LevelUp` : UN seul est dans les 43
  (`zone_state_p2b_temoin_test.go`) ; les 18 autres n'importent rien de LevelUp (13 en commentaire de commande, 5 avec un chemin par défaut dans le code).
  **Lecture des règles** (décision du pilote, appliquée) : R1 = le test a besoin de CODE de LevelUp (chargeurs `mappings`, document servi `replaydoc`, extracteur
  de chronique de `testutil`, registre d'armes) ET n'exerce que la surface exportée de la brique ; R2 = il a besoin d'un paquet interne de la brique, ou seulement
  de DONNÉES du dépôt (catalogues versionnés, parc) : il reste, derrière `racineDepot`. Aucun des 43 fichiers n'est un R1 ENTIER (tous lisent des helpers non
  exportés de leur paquet ou `film/internal/*`) : **43 fichiers R2, 5 tests R1 extraits de 4 fichiers, 1 instrument créé**.

  *Mécanismes posés.*
  - `racineDepot(t)` : un fichier `depot_test.go` par paquet (`replay`, `internal/grammar`, `internal/facts/killsource`, `internal/profile` [`profile_test`],
    `killicon`, `equipenergy`), même nom, même corps : remonte depuis l'emplacement du fichier source (`runtime.Caller`) jusqu'au premier dossier qui porte
    `go.work`, `t.Skip` s'il n'y en a pas (la brique copiée seule). `replay` y ajoute `titreDeReference` (= `"halo_infinite"`, littéral de test),
    `bornesDeCartes(t)`, `cacheDuDepot(t)` et `idCourtDeMatch` (copie de 6 lignes de `title.FilmShortMatchID`, ids bien formés seulement ; dans `depot_research_test.go`, tag `research` : seuls les instruments s'en servent et `unused` refuse un symbole mort du build par défaut) ; `grammar` :
    `referenceDuDepot`, `bornesDeCartes`, `cacheDuDepot`. Les chemins du parc se construisent depuis cette racine avec les MÊMES segments que `PathResolver`
    (`data/titles/halo_infinite/reference/<fichier>` pour `MapQuantBoundsPath`, `EquipmentEnergyPath`, `MapBackgroundDir` ; `data/cache`).
    `repoRootForTest` (replay, remontait au dossier `config/titles`, 13 fichiers) est SUPPRIMÉ au profit de `racineDepot`. **Allowlists : aucune entrée à ajouter**
    (le helper ne porte ni échelle `../../..`, ni `title.FindRepoRoot`, ni littéral `.ai`) ; l'entrée `ctf_research_test.go` de `repoRootInTestsAllowlist`
    est RETIRÉE (le fichier n'appelle plus le helper de production). Le helper homonyme de `facts/objectives/families_parite_ts_test.go` (marqueur
    `apps/web/package.json`, `Fatal`) n'importe rien de LevelUp : laissé tel quel (D2-L2-n).
  - **Instantané du catalogue de libellés du titre** (la pièce centrale). `golden_catalog_test.go` bâtissait le catalogue avec les chargeurs de LevelUp
    (`mappings.Load*`, `weapons.FilmshellWeaponKeysByFamily`) pour une trentaine de tests de `replay` (goldens d'assemblage, de builds, de socles). Une fixture
    inventée aurait changé les goldens (aucun golden n'est régénéré ici). Le catalogue est donc un INSTANTANÉ versionné,
    `replay/testdata/golden_catalogue_titre.json` (`LabelCatalog` sérialisé, 9 727 octets), produit par la jointure de la production ; `goldenCatalog` le désérialise.
    La garde de fraîcheur est côté hôte : `filmcorpus/catalogue_titre_test.go` (`TestInstantaneDuCatalogueDuTitreEstFrais` : octets égaux ET relecture
    `reflect.DeepEqual` au catalogue construit ; porte de régénération à deux verrous `-update-catalogue-titre` + `LEVELUP_UPDATE_CATALOGUE_TITRE=1`).
    Un libellé du titre qui change rougit d'abord cette garde ; la chaîne vers les goldens est celle d'avant avec un maillon visible de plus.
    `goldenReplayLabels` (rendait un `*mappings.ReplayLabelSet`) est supprimé : ses 8 appelants lisent `goldenCatalog(t).EquipmentFamilies`, `.Abilities`,
    `.Keys`. Les instruments `f0_103_*` (paquet `grammar`, qui ne peut pas importer `replay`) lisent le même JSON par une structure locale de deux champs :
    la variable `F0_LABELS` désigne désormais l'instantané et non `replay_labels.toml` (D2-L2-o).
  - **Digest déménagé** : `internal/analysis/digest` (feuille de la bibliothèque standard, 443 lignes + tests) → `film/digest` (`git mv`) ; ses seuls
    utilisateurs sont le test de la mini-bobine du film et `cmd/replay-equiv` (outil d'équivalence du décodeur), réécrits. Règle D2-D (le film est le seul
    usage : le helper déménage). Ses trois `sort.Slice` passent à `slices.SortFunc` / `SortStableFunc` (le ratchet `film_tri_total_test` couvre désormais
    le dossier) sans changer un octet de sortie (tests de `digest` et `TestEquivalenceMiniFilm` verts) ; classé hors couche dans `film_layers_deps_test`.
  - **Export** : `replay.ContinuousWeaponKeys()` (cles d'arme à tir continu que la table des chassis déclare ; vrai contrat avec le registre du titre) ;
    plafond de surface de `replay` 315 → 316, justification datée.
  - Fixtures locales : `highlight_events_jgtm_test.go` lit le fixture JGtm de l'hôte (gitignoré, 6 Mo) par un chargeur de 86 lignes (manifeste et chunks seuls,
    `jgtmDisponible` / `chargerJGtm`) ; `vehicle_end_acceptation_test.go` lit l'artefact cuit dans le type de la brique (`ReplayDocument`) et non dans le jumeau
    servi ; `emprise_v0_porteurs_research_test.go` remplace `domain.MatchFacts` par une structure locale à un champ lu (les joueurs et leur triplet) ;
    `document_shape_test.go` fige la forme servie = la forme stockée (la garde est côté hôte).

  *R1 — liste nominative (5 tests, 4 fichiers sources, nom CONSERVÉ ; destination `apps/go-api/internal/filmcorpus/`).*

  | Test | Fichier d'origine (reste en R2 pour le reste) | Fichier destination |
  |---|---|---|
  | `TestCatalogueDuTitreNommeLesArmesDuFilmDeReference` | `replay/golden_catalog_test.go` | `catalogue_titre_test.go` |
  | `TestLeManifesteNommeSesObjetsDObjectifDansLesDeuxLangues` | `replay/ground_weapon_flag_exclusion_test.go` | `objets_objectif_titre_test.go` |
  | `TestArmesDesChassisAuRegistre` (le test de la brique qui reste s'appelle `TestArmesDesChassisOntUneCadence`, nom neuf) | `replay/tir_continu_armes_test.go` | `armes_continues_registre_test.go` |
  | `TestDocumentShapeTwinsAgree` | `replay/document_shape_test.go` | `document_shape_jumeaux_test.go` |
  | `TestDocumentShapeSchemaHasChronicleEntry` | `replay/document_shape_test.go` | `document_shape_jumeaux_test.go` |

  *Garde-rails et instruments créés dans `filmcorpus`* : `TestInstantaneDuCatalogueDuTitreEstFrais`, `TestRenduDesFormesJumeauxSuitLeGolden` (la copie du rendu de forme
  de `filmcorpus` rend la forme en clair et l'empreinte du golden de la brique : deux copies, règle 6), `TestRRM4AGatesG4` (instrument `research` : la gate G4
  du lot M4a, qui lisait le registre des armes de véhicule, sortie de `TestRRM4AGates` ; les gates G3 restent dans la brique sous le même nom ; même variable
  `RR_M4A_DOCS`). `filmcorpus/doc.go` porte le contrat du paquet.

  *Table des 43 fichiers.* Imports visés → règle → mécanisme. Sauf mention, « R2 / `racineDepot` » = le test reste, `testutil.RepoRoot()` remplacé par `racineDepot(t)`.

  | Fichier (sous `film/`) | Imports visés | Règle | Destination / mécanisme |
  |---|---|---|---|
  | `internal/facts/killsource/carte_du_match_test.go` | `testutil` | R2 | `racineDepot` |
  | `internal/grammar/f0_103_artefact_research_test.go` | `mappings` | R2 | instantané du catalogue (`F0_LABELS`), structure locale `f0InstantaneCatalogue` |
  | `internal/grammar/generations_vivantes_datees_canaux_films_test.go` | `testutil` | R2 | `racineDepot` |
  | `internal/grammar/generations_vivantes_datees_films_test.go` | `testutil` | R2 | `racineDepot` |
  | `internal/grammar/highlight_events_jgtm_test.go` | `testfixtures` | R2 | chargeur local du fixture JGtm |
  | `internal/grammar/livefire_positions_test.go` | `testutil` | R2 | `racineDepot` |
  | `internal/grammar/poussees_films_test.go` | `testutil` | R2 | `racineDepot` |
  | `internal/grammar/r2_positions_aberrantes_research_test.go` | `testutil` | R2 | `racineDepot` (2 sites) |
  | `internal/grammar/r2bis_canaux_dates_research_test.go` | `testutil` | R2 | `racineDepot` |
  | `internal/profile/loi_largeurs_test.go` | `testutil` | R2 | `racineDepot` (`profile_test`) |
  | `killicon/killicon_test.go` | `testfixtures` | R2 | `racineDepot` ; `spritePath(t, stem)` (+ `neutral_test.go`, 3 appels) |
  | `replay/callouts_catalog_test.go` | `testutil` | R2 | `racineDepot` (2 sites) |
  | `replay/contract_fixtures_test.go` | `testutil` | R2 | `racineDepot` |
  | `replay/ctf_research_test.go` | `domain/title` | R2 | `bornesDeCartes` ; entrée d'allowlist retirée |
  | `replay/document_shape_test.go` | `replaydoc`, `testutil` | R2 + 2 tests R1 | forme servie = forme stockée ; 2 tests → `filmcorpus` |
  | `replay/emprise_v0_porteurs_research_test.go` | `domain`, `domain/title`, `testutil` | R2 | structure locale `Faits`, `bornesDeCartes`, `idCourtDeMatch` |
  | `replay/emprise_v0_rapport_research_test.go` | `domain/title` | R2 | `titreDeReference` |
  | `replay/emprise_v0_synthese_research_test.go` | `domain/title` | R2 | `idCourtDeMatch` |
  | `replay/emprise_v1_porteurs_research_test.go` | `domain/title`, `testutil` | R2 | `bornesDeCartes`, `idCourtDeMatch` |
  | `replay/equipment_pickup_manifest_research_test.go` | `games/weapons` | R2 | `goldenCatalog(t).Keys`, `.EquipmentFamilies` |
  | `replay/equipment_uses_research_test.go` | `mappings` | R2 | `goldenCatalog(t).EquipmentFamilies` |
  | `replay/equivalence_minifilm_test.go` | `analysis/digest` | R2 | `digest` déménagé sous `film/digest` |
  | `replay/flag_spawn_neutral_label_test.go` | `testutil` | R2 | `racineDepot` |
  | `replay/golden_catalog_test.go` | `mappings`, `games/weapons` | R2 + 1 test R1 | `goldenCatalog` = instantané ; test → `filmcorpus` |
  | `replay/golden_equipement_test.go` | `domain/title` | R2 | chemin `reference/equipment_energy.json` depuis `racineDepot` |
  | `replay/ground_weapon_flag_exclusion_test.go` | `mappings` | R2 + 1 test R1 | test du manifeste → `filmcorpus` |
  | `replay/ground_weapon_pads_research_test.go` | `domain/title` | R2 | `bornesDeCartes` |
  | `replay/lettres_ordre_research_test.go` | `domain/title` | R2 | `titreDeReference` |
  | `replay/map_background_index_catalogue_test.go` | `domain/title`, `testutil` | R2 | `racineDepot` + `reference/map_backgrounds` |
  | `replay/map_weapon_pads_catalog_test.go` | `testutil` | R2 | `racineDepot` |
  | `replay/objectives_catalog_test.go` | `testutil` | R2 | `racineDepot` |
  | `replay/origine_poses_research_test.go` | `mappings` | R2 | `goldenCatalog(t).EquipmentFamilies` |
  | `replay/positions_porte_temoins_research_test.go` | `testutil` | R2 | `racineDepot` |
  | `replay/reapparition_37_collation_research_test.go` | `domain/title`, `testutil` | R2 | `bornesDeCartes` |
  | `replay/rr_m4a_gates_research_test.go` | `mappings` | R2 + instrument | G3 reste ; G4 → `TestRRM4AGatesG4` (`filmcorpus`) |
  | `replay/tir_continu_armes_test.go` | `mappings` | R2 + 1 test R1 | `replay.ContinuousWeaponKeys` ; test du registre → `filmcorpus` |
  | `replay/vehicle_end_acceptation_test.go` | `replaydoc` | R2 | types de la brique (`ReplayDocument`, `VehicleTrack`) |
  | `replay/vehicle_families_test.go` | `testutil` | R2 | `racineDepot` |
  | `replay/weapon_burst_research_test.go` | `games/weapons` | R2 | `goldenCatalog(t).Keys` |
  | `replay/zone_56_camp_research_test.go` | `domain/title`, `testutil` | R2 | `bornesDeCartes` |
  | `replay/zone_etat_initial_temoin_test.go` | `domain/title` | R2 | `titreDeReference` |
  | `replay/zone_state_p2a_corpus_test.go` | `domain/title` | R2 | `titreDeReference` |
  | `replay/zone_state_p2b_temoin_test.go` | `domain/title` | R2 | `titreDeReference` ; chemin `Projects\LevelUp` retiré |

  *Autres fichiers touchés (hors des 43, conséquences directes).* 18 fichiers `Projects\LevelUp` : 13 commentaires de commande → `<depot>/data/...`
  (`delta_walk_witness`, `ti13_*` x3, `zone_census*` x2, `zone_state_measure`, `zone_vectors`, `colline_propriete*` x2, `hill_shapes_measure`,
  `zone_state_p2a`, `zone_gauge_temoin`) ; 5 défauts de code (`vehicules_v10_deadstate`, `vehicules_v2_items`, `vehicules_v2b_vitalite` côté `grammar` ;
  `vehicules_v2_deaths`, `vehicules_v2b_cooldown` côté `replay`) → `cacheDuDepot(t)` / `bornesDeCartes(t)` / `referenceDuDepot(t, ...)` (les fonctions `v10Root`,
  `v2Root`, `v2bRoot`, `v2cRoot`, `v2dDir` reçoivent `t`) ; `equipenergy_test.go` (D2-L2-f : l'échelle de `..` x7 → `racineDepot`) ;
  8 appelants de `goldenReplayLabels` / de la remontée `repoRootForTest` ; `filmfacts_fichier.go` (commentaire). `archlint` : `briques_imports_test.go`
  (table `tolerancesFilm`, champ `tolerances`, `violationsDeTolerances` RETIRÉS ; la ligne `film` se juge comme `gamefiles` et `himap`), `briques_garde_test.go`
  (auto-preuve des tolérances retirée), `film_layers_deps_test.go` (`digest` hors couche), `film_file_size_test.go` (entrée de `document_shape_test.go` retirée,
  il passe sous 500 lignes), `film_facade_surface_test.go` (316), `no_repo_root_walk_test.go` (entrée de `ctf_research_test.go`).
- Bilan 2.10 et 2.11 : `tolerancesFilm` VIDE puis retirée ; `TestBriquesImports` vert pour `film` (le décodeur n'importe plus rien du dépôt, tests compris :
  `grep -rhoE --include='*.go' '"levelup/go-api/internal/[^"]+"' film | grep -v halo_infinite/film` ne rend plus que le littéral de chaîne d'un test du cadre
  historique). Rouges attendus, inchangés : `grammar/TestGrammarRevSuitLaGrammaire`, `killsource/TestKillsourceRevSuitLaSortie`, `objectives/TestObjectivesRevSuitLaSortie`,
  `revision/TestChaqueRevisionEgaleSonGolden` et `TestPerimetreDeChaqueCoucheEgaleSonGolden` (sous-tests grammar, killsource, objectives). Aucun golden régénéré.
  Gates : `go build ./...`, `go vet ./...`, `go vet -tags research` (film, filmcorpus) et `go test -tags research -run XXX` propres ; `gofmt` vide ; `golangci-lint --new-from-rev=0bfe57feb`
  0 issue ; `go test ./internal/archlint/` vert ; `go test ./... -count=1` (193 s) : 214 paquets `ok`, seuls rouges les attendus. **Comparaison au témoin T1 du lot 0**
  (`(paquet, test, verdict)` réduit par `reduire.sh`) : 70 couples (paquet, test) ont quitté leur paquet depuis le lot 0 (filmproc 39, digest 15, replay 5 -> `filmcorpus`,
  filelock 3, equipenergy 3, canonical 3, filmshell 2) ; leurs 70 NOMS existent tous dans le run courant (aucun nom ne disparaît) ; sur les couples communs, aucun
  passage `pass` -> `skip`, seuls changements de verdict les 11 attendus ; le parc `data/cache` est absent du worktree comme au lot 0 (mêmes tests sautés). Baseline de présence
  (`.ai/baselines/tests_pre_migration.jsonl`) : 2 lignes relocalisées `film/replay` -> `filmcorpus`, entrée datée dans `scripts/check_test_baseline.sh`. Commits : 2.10 `05bbbf8cf` ;
  2.11 `2ea08ac1b` (R2 hors replay), `07f8f0802` (replay, filmcorpus, digest, archlint) et le commit de clôture (baseline, plan, journal).
- 2.12 (gate du lot 2, 2026-10-29). Témoins sous `scratchpad\temoins_d2_lot2_12\`. **A. Autonomie.** `grep` des imports `levelup/go-api/internal/...` hors
  `halo_infinite/film` sous `film/` : une seule occurrence, le littéral de chaîne `"levelup/go-api/internal/domain/highlightevent"` du cas négatif de
  `revision/cadre_test.go` (un littéral, pas un import : `go list -deps` ne le voit pas) ; `film/research` : 0. `go build ./...`, `go vet ./...`,
  `go vet -tags research ./…/film/... ./internal/filmcorpus/...` : code 0 (CGO : gcc `C:\msys64\ucrt64\bin` dans le PATH). `go vet -tags 'research campagne_overlay'` :
  rouge comme à la base (D2-L0-b), mêmes 10 erreurs `bis2*` indéfinies dans `grammar/campagne_bis2_positions_research_test.go` (lignes 46, 54, 55, 57, 61, 72, 73, 74, 87, 125)
  et « too many errors », rien d'autre (`vet_research_overlay.txt`). `go test ./internal/archlint/` vert (avant et après la régénération).
  **B. Régénération unique à révision constante** (`grammar`, `killsource`, `objectives` ; `source` et `profile` non touchés). Avant : exactement
  `TestGrammarRevSuitLaGrammaire`, `TestKillsourceRevSuitLaSortie`, `TestObjectivesRevSuitLaSortie`, `TestChaqueRevisionEgaleSonGolden` et
  `TestPerimetreDeChaqueCoucheEgaleSonGolden` (sous-tests des trois couches) rouges. Entrée de chronique « COMPLEMENT DU 2026-10-29 ... REVISION CONSTANTE » écrite d'abord dans
  `grammar/rev_chronique.go`, `facts/killsource/rev_chronique.go`, `facts/objectives/rev.go` (forme des compléments existants ; `ENTREE` est réservé aux montées). Porte à deux verrous
  (`-update-grammar-rev` + `LEVELUP_UPDATE_GRAMMAR_REV`, idem `-update-killsource-rev` / `LEVELUP_UPDATE_KILLSOURCE_REV`, `-update-objectives-rev` / `LEVELUP_UPDATE_OBJECTIVES_REV`,
  `-update-perimetres` / `LEVELUP_UPDATE_PERIMETRES`), chaque variable retirée après usage. **Diff exact (8 lignes changées dans 6 goldens)** : révisions : `grammar-2026-10-28`
  `8353c693...27121` -> `d47f789c...bd064` ; `killsource-2026-10-18` `3a3992d4...ebf9d` -> `2b07bd0d...f45` ; `objectives-2026-10-17` `2b45f688...62b67` -> `bbab436e...e38d` (une ligne chacun, MÊME révision) ;
  périmètres : `killsource` `internal/domain/highlightevent` -> `internal/games/halo_infinite/film/highlightevent` ; `objectives` `internal/domain/objectiveevent` ->
  `internal/games/halo_infinite/film/objectiveevent` ; `grammar` `internal/domain/highlightevent`, `internal/domain/playerposition`, `internal/games/weapons/filmshell` ->
  `film/filmshell`, `film/highlightevent`, `film/playerposition` (3 lignes ; l'ordre est celui du tri). Rien d'autre (aucune ligne `amont`, aucun golden de `source` ni de `profile`).
  Gates sans porte : `go test ./internal/games/halo_infinite/film/... -count=1` vert (27 paquets, `grammar` 71 s) ; `revision -v` : 46 verdicts PASS, 0 FAIL, 0 SKIP : les 40 du témoin y sont tous, plus 6 du lot 2.1 (`TestCadreHistoriqueDesImports` et ses 3 sous-tests, `TestCadreHistorique_LaBriqueDeplaceeRendLaMemeEmpreinte`, `TestCheminModuleCourantEstLaRacineDeLaBrique`).
  **C. Gate complet.** T1 `go test -json ./... -count=1 -timeout 1200s` : code 0 ; 20 262 verdicts (20 114 au témoin : +148), 19 896 pass / 366 skip / 0 fail (19 748 / 366 / 0 au témoin) ; 218 paquets `ok`
  (212), 0 rouge, 146 sans test (146). Aucun NOM de test ne disparaît ; 70 couples (paquet, test) ont changé de paquet, leurs 70 noms existent dans le run (analysis/digest 15 -> film/digest ; filmproc 39 -> film/filmproc ;
  games/canonical 3 -> film/identity ; games/halo_infinite/equipenergy 3 -> film/equipenergy ; film/replay 5 -> filmcorpus ; games/weapons/filmshell 2 -> film/filmshell ; platform/filelock 3 -> film/internal/filelock) ;
  0 changement de verdict sur les couples communs, 0 pass -> skip. Paquets `ok` : 5 sortent par déplacement (analysis/digest, filmproc, halo_infinite/equipenergy, weapons/filmshell, platform/filelock), 11 entrent
  (film/digest, film/equipenergy, film/filmobs, film/filmproc, film/filmshell, film/identity, film/internal/atomicfile, film/internal/filelock, film/internal/flatjson, filmcorpus, domain/equipmentusage) : 212 - 5 + 11 = 218.
  T3 `replay-equiv -repo-root <dépôt principal> -out-dir` : code 1 et « BILAN : 0 identique, 20 different, 0 ecarte, 0 echec, 0 illisible » comme au lot 0 (références périmées D2-L0-a, 566 s) ;
  les 20 fichiers de digests obtenus ont des MD5 IDENTIQUES à `temoins_d2_lot0_avant\replay_equiv_obtenus\` (`md5_avant.txt` = `md5_apres.txt`, 1 360 lignes). T2 : 202 fichiers de `empreintes.txt` recalculés, 196 identiques ;
  les 6 qui diffèrent sont exactement les 6 goldens régénérés ci-dessus ; `equivalence/*.tsv` (21), `*.facts.json` (20), goldens de `source` et de `profile` identiques. `gofmt -l` vide.
  `golangci-lint run ./...` complet (cache dédié) : 396 issues, dette gelée (Makefile : ~479) ; ratchet CI `--new-from-merge-base=origin/main` : **0 issue** (apps/go-api, `libs/gamefiles`, `libs/himap`), donc zéro sur
  tout fichier neuf. Un comptage de contrôle sur une copie propre de `a5bae50b4` (267) n'est pas comparable : la copie ne charge pas les fichiers non suivis du poste, des paquets n'y sont pas analysés.
  Un premier T1 a été perdu sur ses libellés (redirection PowerShell : UTF-16 et accents illisibles) et refait avec une redirection binaire par `cmd`.

## Témoins avant/après (règle utilisateur du 2026-10-28)

| Témoin | Capture | Comparaison |
|---|---|---|
| T1 tests Go | `go test -json ./... -count=1` → `(paquet, test, verdict)` trié ; lot 0 (go-api) ; lot 3/5 (go-api + `libs/film`) | mêmes noms et verdicts modulo le préfixe de paquet ; les tests déménagés en `filmcorpus` listés nommément |
| T2 empreintes | sortie `-v` des tests de `film/revision` + MD5 des goldens de révision et de périmètre et de `testdata/equivalence/*.tsv` | identiques à l'octet, aucun golden réécrit |
| T3 `replay-equiv` | code de sortie + journal | même verdict |
| T4 `replay-corpus-gate --reference=base` | lot 5 seulement (base = sha du lot 0) | code 0, 19 témoins identiques |
| compteurs | noms des constantes de compteur du film (grep) | inchangés |
| fil JSON des libellés | test d'égalité des deux codecs + documents de `replay/testdata` relus | mêmes octets |

## Protocole de reprise de session

1. `git branch --show-current` ; `git log --oneline -10` dans `../LevelUp-wt-decouplage-d2`.
2. Relire ce plan : premier item sans statut du premier lot non clos = point de reprise.
3. Vérifier sur pièces avant de coder (le code a pu bouger).
4. Journal : `.ai/thought_log.md`, entrées « Découplage D2 — lot N ».

## Découvertes (notées, non traitées)


- D2-L0-a (lot 0, 2026-10-29) : les références de `replay-equiv` (`replay/testdata/equivalence/*.tsv`) sont périmées depuis la
  v8.0.0 (64 étapes figées, 67 attendues) : le mode par défaut rend « 20 différents » sur la base. Hors périmètre D2 (re-figer
  les références = décision du pilote, `-update`). Le témoin T3 de D2 devient « digests obtenus identiques avant/après ».
- D2-L0-b (lot 0) : le jeu de balises `research campagne_overlay` ne compile pas (symboles `bis2*` absents de
  `film/internal/grammar` : recouvrement de campagne non versionné). 21 fichiers de test inatteignables ; hors périmètre.
- D2-L0-c (lot 0) : un 21e `equivalence/*.tsv` existe sans film correspondant au corpus de 20 films (`CORPUS.txt`) ; non investigué.
- D2-L1-a (lot 1, 2026-10-29) : le plan (1.5, D2-G b) parle de trois pas CI `go list -deps` ; `ci.yml` n'en porte que deux (`libs/himap`,
  `libs/gamefiles`), plus la cible `libs-test`. Un module de plus (`libs/film`) n'ajoute pas de pas : la boucle de `libs-test` suffit, le
  pas CI est à ajouter au lot 3 s'il n'y a pas de boucle côté CI.
- D2-L1-b (lot 1) : le ratchet `TestSurfaceDeReplayCiteeHorsDuDecodeur` compte tout identifiant `replay.X` cité dans un `.go` hors du film,
  commentaires compris : un commentaire de garde-rail qui nomme un symbole de `replay` en fait monter la surface. Écrire ces commentaires
  sans la forme `replay.Symbole` (fait ici pour `Text`). Non traité.
- D2-L1-c (lot 1) : la fermeture `go list -deps -test` d'`internal/analysis/...` ne contient aujourd'hui aucun paquet du décodeur ni
  `filmshell` : l'allowlist `fuitesTransitivesAnalysis` naît vide ; la fuite `games/weapons` -> `filmshell` n'apparaîtra qu'au lot 2.5.
- D2-L2-a (lot 2.1, 2026-10-29 ; TRANCHÉE le 2026-10-29 par le pilote : décision D2-F amendée, régénération unique à révision constante à l'item 2.12) — **conflit entre le critère de succès (3) et les items 2.3, 2.5, 2.7 tels qu'écrits.** Les
  goldens de périmètre portent des paquets HORS du dossier `film` qui décident la sortie des couches :
  `internal/domain/playerposition` et `internal/games/weapons/filmshell` (grammar), `internal/domain/highlightevent` (grammar,
  killsource), `internal/domain/objectiveevent` (objectives). Les déplacer dans la brique (2.3, 2.5) change (i) les chaînes
  d'import des sources de ces couches (`levelup/go-api/internal/domain/playerposition` -> `.../film/playerposition`), que le
  cadre historique ne couvre pas (elles ne commencent pas par le chemin de la brique), et (ii) la ligne `paquet` du golden de
  périmètre et le préfixe haché des fichiers du paquet ; 2.7 (`types.HighlightEvent`) change en plus des jetons de `grammar` et
  `killsource`. Les empreintes de `grammar`, `killsource` et `objectives` bougeront donc au lot 2, et leurs goldens (révision
  ET périmètre) devront être régénérés — ce que le gate D2-F interdit. Pistes pour le pilote (non tranchées) : étendre le cadre
  par une table « nom historique par paquet déplacé » (couvre les déplacements nus 2.3 et 2.5 : chaînes d'import et dossiers),
  et pour 2.7 accepter une régénération À RÉVISION CONSTANTE (sortie inchangée ; les faits persistés portent les révisions,
  constantes `Rev` lues par exemple dans `replay/coverage_decoder.go`, pas les empreintes de sources : à confirmer), ce qui ne
  déclenche aucune recuisson mais rouvre une écriture de golden.
- D2-L2-b (lot 2.1) : côté tests, trois lectures de chemins de `film/revision` resteront à reraciner au lot 3 (aucune ne touche
  un golden) : `arborescence_test.go` (`racineAPI` remonte de 5 dossiers vers `apps/go-api`), les chemins des goldens de
  révision dans `equivalence_test.go` et le dossier de `grammar` dans `chronique_test.go` (tous deux construits depuis
  `racineAPI` + `internal/games/halo_infinite/film/internal`). Le test des goldens de périmètre passe déjà par `Module.SurDisque`.
- D2-L2-c (lot 2.1) : le gate `go test ./internal/games/halo_infinite/film/...` dure 73 s sur ce poste (le brief l'estimait à
  ~10 min) ; mêmes 18 paquets `ok` qu'au témoin T1, donc pas un trou de couverture, mais l'estimation est à corriger.
- D2-L2-d (lots 2.5 et 2.6, 2026-10-29) : `filmproc` (`memguard.go`, `solo.go`, `solo_observe.go`, `selfpriority_windows.go`, 10 sites) et la copie `film/internal/atomicfile`
  (2 sites) journalisent par `slog.Info/Warn` SANS contexte ; ils n'étaient pas dans le périmètre du ratchet J12.3 (`film_slog_contexte_test`) tant qu'ils vivaient hors de `film/`.
  Exclusion datée `horsPerimetreDeJournal` posée. Donner un `ctx` change l'API publique de `Arm`, `ArmBytes`, `AcquireSolo`, `Release`, `LowerOwnPriority` (98 fichiers appelants) et
  la signature d'`atomicfile.WriteFile` (onze autres fichiers importeurs, copie jumelle). Non traité.
- D2-L2-e (lot 2.5) : `film/filmshell` porte 15 variables exportées modifiables (`WeaponIDs`, `WeaponTimingByID`, `SentinelIDs`, ...) lues par `games/weapons`, `platform/duckdb`,
  `sync/killcollector` et le décodeur ; exclu des ratchets `film_exported_vars` (`catalogueArmesHorsRatchet`) et `no_raw_film_bytes_outside_source`. Les rendre accesseurs est un lot à part (dette
  ADR 0012 préexistante). Non traité.
- D2-L2-f (lot 2.5) : `equipenergy_test.go` remonte au parc par un chemin relatif en dur (`..` x7 maintenant) ; à remplacer par le helper de remontée par `go.work` au 2.11 (règle R2).
- D2-L2-g (lot 2.2) : le relevé de départ annonçait 14 sites `observability` de production ; il y a 11 appels dans 9 fichiers (les 14 comptaient sans doute des mentions en commentaire).
  Sans conséquence : la tolérance comptait 13 fichiers (9 prod, 4 tests) et est tombée à 0.
- D2-L2-h (lot 2.6) : `GOOS=linux go build ./...` échoue depuis ce poste Windows (cgo : `duckdb-go-bindings/lib/linux-amd64`, `libs/gamefiles/ooz`) : limite de la compilation croisée, sans lien avec D2 ;
  les paquets déplacés (`filmproc`, `internal/filelock`, `filmcache`, `internal/...`) passent `GOOS=linux go vet`.
- D2-L2-i (lot 2.7, 2026-10-29 ; TRANCHÉE le 2026-10-29 par le pilote : le type d'événement RESTE dans la feuille `film/highlightevent`, pas dans `film/types`, qui entre dans l'empreinte de `source` ; rien à faire dans le code) : **le libellé « `types.HighlightEvent` » du plan (D2-B, 2.7, D2-E R0) est en conflit avec l'exigence « `source` reste vert »** :
  `film/types` est haché par ses octets dans l'empreinte de `source` (son golden de périmètre liste `film/types`) et chaque type de `film/types` doit être figé
  dans `types/testdata/shapes.golden`. Exécuté : feuille `film/highlightevent` (cf. « Exécution du lot 2 », 2.7). À confirmer par le pilote ; si `film/types` est
  voulu, déplacer le fichier coûte une régénération des goldens de `source` (révision constante) et du golden des formes, à joindre au 2.12.
- D2-L2-j (lot 2.9) : **le contrat d'écriture des libellés a changé de porteur.** `i18n.MarshalFlat` complétait toute disposition sans `OmitEmpty` avec chaque
  langue de la liste (vide comprise), même pour un texte nil ; `flatjson` écrit les clés du texte. Le fil est identique tant que chaque texte qui entre dans
  la brique passe par `replaylabels.TexteVersFilm` (tous les producteurs de production le font, et le décodeur ne crée de libellé qu'en clonant ceux du
  catalogue) ; un type du rejeu construit avec un texte nil hors de ce chemin écrirait moins de membres qu'avant. Garde-fous : `fil_jumeau_test` (producteur
  + codec), `replay-equiv` (étape `artifact`, 20 films identiques). Conséquence assumée : `replayview/parity_test.go` remplit désormais le texte du document
  stocké comme le producteur (`TexteVersFilm`), y compris aux bords « nul » et « vide ». Relecture : un membre de langue de la liste à valeur non chaîne
  faisait échouer `i18n.UnmarshalFlat` ; `flatjson` l'écarte (il n'est pas une langue). Non traité au-delà.
- D2-L2-k (lots 2.8 et 2.9) : le ratchet `TestSurfaceDeReplayCiteeHorsDuDecodeur` monte de 312 à 315 (`EquipmentOutcomeFamilies`, `UsageFamilySpawnsPiece`,
  `Text`) : effet mécanique de toute copie jumelle dont LevelUp garde le garde-rail (le test nomme les symboles de la brique). Justifications datées
  écrites ; à prévoir pour les copies du 2.10.
- D2-L2-l (lot 2.9) : trois tests du film lisent des libellés de `games/mappings` (type `i18n.Text` de LevelUp) par `Pick("en")` / `Pick("fr")`
  (`golden_catalog_test.go`, `ground_weapon_flag_exclusion_test.go`, `equipment_pickup_naming_research_test.go`) : ils relèvent de la tolérance `games/mappings`
  (2.11) et devront, au déménagement R1 vers `filmcorpus`, passer par `TexteVersFilm` plutôt que reconstruire un texte à deux codes.
- D2-L2-m (lot 2.11, 2026-10-29) : le « 51 fichiers » de `tolerancesFilm` est un compte de COUPLES (fichier, import) : 43 fichiers distincts. Sur les 19 fichiers qui citent
  `Projects\LevelUp`, un seul est dans les 43 ; les 18 autres n'ont aucun import de LevelUp (leur chemin est en commentaire de commande ou en défaut de code) et ont été
  traités dans le même geste. Non traité au-delà.
- D2-L2-n (lot 2.11) : `facts/objectives/families_parite_ts_test.go` porte un `racineDepot` homonyme dont le marqueur est `apps/web/package.json` et qui échoue (`Fatal`) au lieu de
  sauter ; il n'importe rien de LevelUp. Deux corps pour un nom : à unifier au lot 3, quand la remontée par `go.work` devient le marqueur de la brique. Non traité.
- D2-L2-o (lot 2.11) : la variable `F0_LABELS` des instruments `f0_103_*` (`grammar`) désigne désormais l'instantané du catalogue (`replay/testdata/golden_catalogue_titre.json`) et non
  `replay_labels.toml` ; les instruments `eqUses*` / `origine*` de `replay` lisent l'instantané au lieu du manifeste TOML. Les trois fichiers de commande en tête de ces instruments sont
  mis à jour ; les valeurs sont celles du manifeste au jour de la régénération de l'instantané, que la garde de `filmcorpus` tient fraîche.
- D2-L2-p (lot 2.11) : au lot 3, trois chemins en dur de `filmcorpus` / `testutil` visent le dossier du film et sont à reraciner sous `libs/film` : `cheminInstantaneCatalogue`
  (`catalogue_titre_test.go`), `cheminGoldenDeForme` (`document_shape_jumeaux_test.go`) et `testutil.ReplayChroniclePath` (`replay/document_chronicle.go`, lu par
  `TestDocumentShapeSchemaHasChronicleEntry` et par `replaybuild/artifact_schema_history_test.go`).
- D2-L2-q (lot 2.11) : le ratchet `film_tri_total_test` (DT-9) couvre `film/**` ; le paquet `digest`, déménagé, y est tombé avec trois `sort.Slice`, convertis (comparateurs totaux ou tri
  stable sur l'ordre du rendu, sortie inchangée). Un déménagement vers `film/` hérite des ratchets du dossier (taille, tri, couches, `slog`) : à prévoir pour le lot 3.
- D2-L2-r (lot 2.11) : `replay/vehicules_v2_deaths_test.go` construisait le dossier d'un film par `root + "\film_chunks\" + short8` (séparateur Windows en dur) ; `filepath.Join` le remplace.
  Sans effet sur Windows, correct ailleurs.
- D2-L3-a (lot 3.3, 2026-10-29) : **l'ordre des imports est dans le flux haché, pas dans le cadre historique.** Renommer le module fait
  retrier les blocs par `gofmt` / `goimports` ; 15 sources hachées gardent leur ordre historique en groupes (commentaire au-dessus de
  `import`, limite écrite dans `revision/cadre.go`, ADR 0034 et 0041 D8). Un outil qui regroupe ces blocs fera rougir les gates de
  couche : la parade est l'ordre historique, jamais une régénération sans montée décidée. Le test du lot 2.1
  (`TestCadreHistorique_LaBriqueDeplaceeRendLaMemeEmpreinte`) n'avait pas de source où un import de la brique côtoie le standard :
  à enrichir si le cadre est retouché. Non traité au-delà.
- D2-L3-b (lot 3.4) : **ratchets muets par construction** : plusieurs gardes sautent ou ignorent un dossier introuvable
  (`no_identity_bridge_outside_registry`, `no_player_index_identity` : `os.Stat` + `continue` ; `no_rewritten_slot_band` : `t.Skipf`)
  ou ne rougissent pas sur une baisse (`no_french_label_literal`) ; d'autres n'avaient pas de contrôle d'entrée morte. Rebranchés sur la
  brique, mais le défaut de forme (un dossier déplacé rend la garde muette) reste : il faudrait `t.Fatal` au lieu de `continue` / `Skip`.
  Non traité (hors périmètre).
- D2-L3-c (lot 3.4) : les ratchets PROPRES à go-api qui balaient `internal/` sans citer le film (ex. littéraux de langue, `Remove`
  ignoré, lectures par périmètre) ne couvrent plus le décodeur, comme ceux qui couvraient `himap` avant D1 ; seuls ceux qui le citaient
  ont été rebranchés (D3/ADR 0041 D7 : règle générale = espace de travail). Inventaire non fait. Non traité.
- D2-L3-d (lot 3.5) : **des tests Vitest lisent des sources du décodeur** (4 fichiers, 5 chemins) : le lot 3 touche donc `apps/web`
  (contrairement à l'hypothèse de 5.4 « web non touché ») ; typecheck, lint et Vitest web sont à rejouer au lot 5 (les 4 fichiers : 101
  tests verts au lot 3).
- D2-L3-e (lot 3.8) : le ratchet de lint `--new-from-merge-base` ne voit pas un renommage qui franchit la racine d'un module (golangci
  tourne dans `libs/film`, l'ancien chemin est hors de sa vue) : la dette gelée des fichiers déplacés réapparaît comme neuve. Une issue
  (`replay/document_equipement_porte.go`, `projeter`, gocyclo 16) : exemptée sur sa ligne `func` avec sa raison (règle du 2026-10-06
  écrite dans `.golangci.yml`). Lint complet de `libs/film` : cette seule issue.
- D2-L3-f (lot 3.4) : contrat du registre des replis changé : `Site.Fichier` et `OutilHorsProduction.Appelants` sont relatifs à la
  racine du dépôt hôte (ils l'étaient à `apps/go-api`) ; le registre de la brique nomme encore des fichiers de LevelUp
  (`apps/go-api/internal/sync/...`, `replaybuild`, `service`, `cmd/diag_weapons_v3`) : couplage de donnée préexistant, à revoir si la
  brique sort du dépôt. Non traité.
- D2-L3-g (lot 3.2) : la cible `go-api-test` du Makefile jouait (et joue, depuis `libs/film`) les couches du décodeur avec
  `-timeout 60s` alors que `grammar` dure 71 à 125 s sur ce poste : probablement rouge depuis longtemps, gardée à l'identique. Non vérifié,
  non traité.
- D2-L3-h (lot 3.7) : chaînes et commentaires de `libs/film` qui citent un chemin de l'hôte, NON réécrits : ancien chemin
  `internal/games/halo_infinite/film` 657 lignes dans 559 fichiers, dont 298 commandes de régénération `go test ./internal/games/halo_infinite/film/...`
  (messages d'erreur et EN-TÊTES DE GOLDENS : ces dernières ne changeront qu'à une régénération) ; `apps/go-api` 145 lignes / 142 fichiers ;
  chemins `internal/{sync,replaybuild,service,api,platform,persist,analysis,domain,games/...}` 71 lignes / 37 fichiers ; `cmd/` 129 lignes.
- D2-L3-i (lot 3.8) : `replay-equiv` lit les références sous `<repo-root>/libs/film/...` ET le parc sous `<repo-root>/data/` ; le dépôt
  principal (qui porte le parc) est encore à l'ancienne arborescence : T3 a tourné sur une racine de jonctions (`data` et `config` du dépôt
  principal, `libs` du worktree). Après la fusion, plus besoin.
- D2-L3-j (lot 3.8) : le JSONL du 2.12 converti « u8 » porte des noms de sous-tests abîmés par l'encodage (7 sous-tests de `replay`, 7 de
  `games/halo_infinite`) ; les comparaisons T1 utilisent les JSONL écrits par redirection binaire (`cmd`).
- D2-L4-a (lot 4, revue) : la conversion de libellé neutralisée n'a aucun test qui rougisse à `service/squadagg/squad_formes.go:208` et
  `cmd/mapcallouts-build/main.go:452` (`WeaponCatalog` sans test ; catalogue généré sans assertion de libellé). Dette de couverture
  préexistante (`WeaponCatalog`), site de conversion neuf. Non traité.
- D2-L4-b (lot 4, revue) : l'ordre des imports des 15 sources hachées n'a aucun garde-rail dédié ; un `goimports -w` futur n'est attrapé que par
  le gate de couche, avec un message qui ne nomme pas la cause. Piste : un test de `revision` qui vérifie l'ordre des imports des fichiers des
  périmètres, ou un message de gate qui le suggère (voir D2-L3-a). Non traité.
- D2-L4-c (lot 4, revue) : `flatjson.UnmarshalFlat` s'écarte de `i18n.UnmarshalFlat` sur trois points, tous sur des fils qu'aucun producteur
  n'écrit : casse des codes de langue, membre de langue non chaîne, langue hors liste conservée par la brique puis filtrée par
  `TexteDepuisFilm`. Non traité.
- D2-L4-d (lot 4, revue) : `libs/film/digest/digest.go` : les `sort.Slice` remplacés par des tris à comparateur total donnent la même sortie pour
  les structs de 12 champs ou moins ; non prouvé au-delà. Usage : harnais d'équivalence seulement. Non traité.
- D2-L5-a (lot 5.4, 2026-10-29) : **la capture du JSONL de la baseline par PowerShell 5.1 abîme les noms de sous-tests non ASCII** (`$out = & go test -json ...`, console cachée) : `check_test_baseline.sh`
  rend alors « 58 absents » dans `service`, 38 dans `analysis`, 36 dans `sync` (flèches, accents) alors que tous les tests existent. `scripts/gate-push.ps1` capture de cette façon (même cause que D2-L3-j). Le JSONL
  écrit par redirection binaire de `cmd` est exact. Non traité (hors périmètre) ; piste : écrire le JSONL par `cmd /c` ou fixer `[Console]::OutputEncoding` à UTF-8 dans le script.
- D2-L5-b (lot 5.1) : `f75e7053` (famille `koth_collines`) du manifeste `config/replay_corpus.toml` est ABSENT du parc local (ni manifeste de film, ni match au registre) : le gate rend 18 témoins sur 19
  (code 4 sans `-allow-missing`). Propriété du poste, indépendante du diff. Non traité.
- D2-L5-c (lot 5.1) : `go run ./cmd/replay-corpus-gate` ne propage pas le code 4 (couverture incomplète) : `go run` le rend 1 et ne laisse que la ligne `exit status 4` du journal ; un appelant qui lit
  `$LASTEXITCODE` après `go run` confond « perte » (1) et « couverture incomplète » (4). Non traité ; piste : lire la ligne finale ou compiler le binaire.
