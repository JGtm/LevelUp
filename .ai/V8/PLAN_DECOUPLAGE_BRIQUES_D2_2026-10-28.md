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
- [ ] 2.2 `film/filmobs` + puits (D2-D) ; les 14 sites et 4 tests ; câblage du puits dans
  `internal/api/wire` et les binaires relevés ; tolérance `observability` = 0.
- [ ] 2.3 `domain/playerposition` → `film/playerposition`, `domain/objectiveevent` → `film/objectiveevent`
  (`git mv`, lecteurs LevelUp réécrits, paquets `domain/*` supprimés) ; tolérances = 0.
- [ ] 2.4 `canonical/film_identity.go` (+ test) → `film/identity` ; `replayview/convert_identity.go`
  réécrit ; ratchets `film_pont_identite_test`, `no_identity_bridge_outside_registry_test` ajustés ;
  texte d'amendement de l'ADR 0034 écrit ; tolérance `canonical` = 0.
- [ ] 2.5 `filmshell` → `film/filmshell`, `equipenergy` → `film/equipenergy` (`git mv`, 7 paquets
  réécrits) ; `paquetsInterTitres` de `no_title_package_in_analysis` relu (`weapons` importe
  désormais le film : fuite transitive inscrite datée au test de fermeture du 1.4) ; tolérances = 0.
- [ ] 2.6 `filmproc` → `film/filmproc` ; `atomicfile`, `filelock` déplacés ou copiés selon D2-D (test
  d'identité des copies) ; 10 paquets LevelUp réécrits ; tolérances `filmproc`, `atomicfile` = 0.
- [ ] 2.7 `types.HighlightEvent` + constantes ; `grammar`, `killsource`, `decfilm` basculent ;
  adaptateur dans `internal/games/halo_infinite/` ; quatre appelants de `sync` + `decfilm.Kill` /
  `killcollector` / `cmd/killsource` vérifiés sur pièces ; 15 tests du film basculent ; tolérance
  `highlightevent` = 0. Témoin : `go test ./internal/sync/... ./internal/games/halo_infinite/...`
  identique à T1.
- [ ] 2.8 Familles d'équipement locales dans `replay` + test LevelUp d'égalité des deux listes et
  des deux prédicats ; tolérance `equipmentusage` = 0 (2 tests compris).
- [ ] 2.9 `replay.Text` + `film/internal/flatjson` ; 8 champs et 2 catalogues (`callouts_catalog`,
  `document_labels*`, `document_objective_objects`) ; conversions dans `replaylabels` ; `replayview`
  réécrit ; test d'égalité des octets des deux codecs (deux sens, les cinq dispositions `FlatLayout`
  du fichier) ; les documents de `replay/testdata` relus à l'identique ; tolérance `i18n` = 0 (2 tests
  compris). T3 vert.
- [ ] 2.10 Reliquats de production : `title.FilmShortMatchID`, `mappings.ObjectiveModeEntry`,
  `testutil.ReplayChronicleVersions` (règle D2-D dernier point) ; tolérances = 0.
- [ ] 2.11 Tests couplés restants (R1 → `internal/filmcorpus/`, R2 → helpers locaux ; 19 chemins
  `Projects\LevelUp` retirés) : compte R0 / R1 / R2 / devenus instruments au journal, nom par nom
  pour R1 et « instrument » ; toutes les tolérances à zéro, allowlist retirée du test.
- [ ] 2.12 Gate : `grep -rhoE --include='*.go' '"levelup/go-api/internal/[^"]+"' internal/games/halo_infinite/film | grep -v halo_infinite/film | sort -u`
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

- [ ] 3.1 `git mv apps/go-api/internal/games/halo_infinite/film libs/film` ; `libs/film/go.mod`
  (`module github.com/JGtm/LevelUp/libs/film`, `go 1.26.10`, `require` tiers à la main aux versions
  d'`apps/go-api`, aucun `require` de brique) ; `go.work` (`use ./libs/film`) ; `go.work.sum`.
- [ ] 3.2 Réécriture des imports dans LevelUp (198 + 188 fichiers) : préfixe
  `levelup/go-api/internal/games/halo_infinite/film/` → `github.com/JGtm/LevelUp/libs/film/` (motif
  ancré : `filmprofile` ne bouge PAS) ; dans la brique : imports internes au même préfixe.
- [ ] 3.3 Cadre historique (2.1) : la constante du module courant devient `github.com/JGtm/LevelUp/libs/film` ;
  T2 identique (goldens non réécrits, `git diff` vide).
- [ ] 3.4 `archlint` : les 52 fichiers rerootés (`libs/film/...` dans les 20 constantes et les
  littéraux, parcours par `workspaceRoot` / `workspaceGoSources`), `racinesEspaceDeTravail` +=
  `libs/film`, `research_tag_test` (`dossiersInstruments`), `pas_de_lecture_sous_ai_test` (table),
  `no_repo_root_walk_test` (allowlist des helpers R2), `briques` (paquets = `libs/film`), `dockerfile_go_version_test`
  (quatrième `go.mod`), licence GPL (`cmd/server` ne lie pas `ooz` : la chaîne `film` → `himap` →
  `himodule` → `ooz` EXISTE maintenant pour tout paquet qui importe `replay/map_background` : le
  ratchet doit le voir et `cmd/server` doit rester propre — vérifier sur pièces ce que `cmd/server`
  tire).
- [ ] 3.5 CI (`ci.yml`) : pas vet/test/lint/govulncheck de `libs/film` (vet aussi `-tags research`
  et `-tags 'research campagne_overlay'` : compilation des instruments), `go list -deps -test`
  garde, lots de couverture du module (D2-I ; `go-shards.txt` + `ci_go_shard.sh`), job « Film race »
  dans `libs/film`, filtre `croisee`, `cache-dependency-path` ×10, baseline JSONL de la brique +
  `check_test_baseline.sh` à deux modules (RETRAIT daté des 1 052 entrées du module go-api, qui
  renaissent dans la baseline de la brique).
- [ ] 3.6 `Dockerfile` (COPY `libs/film/go.mod` + `go.sum`), `.github/dependabot.yml` (`/libs/film`),
  `Makefile` (cibles 201, 235, 308, 362 ; `libs-test` : délai mesuré pour `libs/film`),
  `apps/go-api/Makefile` (`tidy` saute `film`), `.golangci.yml` règle 220 (module), hooks `lefthook`
  vérifiés (globs `libs/**` déjà là).
- [ ] 3.7 Docs (D2-J) : ADR 0041, ADR 0034, CLAUDE.md, `ARCHITECTURE_V6` FR/EN et `COMMANDS.md` si
  concernés ; commentaires de la brique qui citent un chemin de l'hôte : relevés, pas réécrits en
  masse (découverte à consigner, comme la 29 de D1).
- [ ] 3.8 Gate : `go build ./...` dans `apps/go-api` ET `libs/film` ; `go vet` trois tags dans
  `libs/film` ; `go list -deps -test ./...` dans `libs/film` sans `levelup/go-api` ; T1 (union
  go-api + brique = mêmes noms modulo préfixe, mêmes verdicts) ; T2 identique ; T3 vert ; archlint
  vert ; lint des trois modules vert ; `make libs-test` vert ; `docker build` local du stage Go si
  Docker est disponible, sinon le job CI « Docker Build » fait foi. Commit « D2 lot 3 », CI verte.

**Clos quand** : 3.8 passé, CI verte, thought_log, point d'étape.

## Lot 4 — Revue adversariale (une ronde) et corrections

- [ ] 4.1 Contrat de lot écrit (6 lignes du skill `adversarial-review`) ; trois relecteurs Sonnet en
  parallèle, aveugles : L3 (anti-patterns) + « frontière de module » (imports, copies, ratchets
  rerootés, lint relâché), L6 (ce que les tests ne couvrent pas : tests déménagés/réécrits, tolérances,
  baseline), et une lentille « fil JSON et empreintes » (codec à plat, cadre historique, goldens).
- [ ] 4.2 Triage par le pilote (P0–P3) ; P0/P1 corrigés avec test de non-régression ; P2 consignés en
  Découvertes ; aucune seconde ronde sauf P0/P1 corrigés (relecture des seules corrections).
- [ ] 4.3 Gate : lot 3 rejoué (3.8) après corrections ; commit « D2 lot 4 » ; CI verte.

## Lot 5 — Preuve de non-régression et clôture

- [ ] 5.1 T4 : `go run ./cmd/replay-corpus-gate --reference=base --base=<sha du lot 0>` (plafond
  mémoire par défaut, racine de travail jetable, 19 témoins) : code 0, rapport JSON conservé en témoin.
- [ ] 5.2 T1, T2, T3 recapturés dans `temoins_d2_lot5_apres/` et comparés (diff vide modulo préfixe).
- [ ] 5.3 `go test -tags=integration -p 1 ./internal/sync/... ./internal/persist/...` (imports
  réécrits dans `persist` et `sync`) : code de sortie 0.
- [ ] 5.4 `gate-push` commande par commande (le `$(MAKE)` GnuWin32 du poste échoue : découverte 28/33
  de D1), web non touché (aucun fichier `apps/web` modifié : typecheck/lint web non requis, à
  confirmer par `git diff --stat`).
- [ ] 5.5 Docs et registres : `.ai/REGISTRE_REPORTS.md` (une entrée par `[!]` et par découverte
  reportée), addendum « D2 exécuté » dans l'audit, mémoire du pilote, thought_log « lot 5 ».
- [ ] 5.6 Caches : `scripts/disk-hygiene.ps1 -Lot d2 -Apply` après la CI.
- [ ] 5.7 PR vers `release/v8.1.0`, CI verte, fusion squash (titre Conventional Commits), branche et
  worktree supprimés, dépôt principal remis au niveau.

**Clos quand** : 5.7 fait. D3 s'ouvre ensuite (plan distinct, à écrire depuis l'audit et les
découvertes de D1 et D2).

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
