# Plan — Fin du découplage : brique décodeur de film publiée, dépôts dédiés des quatre briques — 2026-11-02

Source : demande de l'utilisateur du 2026-11-02 (« en finir avec le découplage et la brique décodeur, maintenant que
le décodeur est stabilisé ») ; entrées `[archi]` « Décodeur de film : le publier » et « Dépôts dédiés » de
`.ai/BACKLOG.md` ; `.ai/HANDOFF_PUBLICATION_BRIQUES_2026-11-01.md` ; reports D1, D2, D3, Publication et Suites de
`.ai/REGISTRE_REPORTS.md` ; ADR 0034, 0041, 0042, 0043. Exécution sous le contrat du skill `plan-execution` : ordre
strict, un lot à la fois, gate passé avant le lot suivant, items statués `[x]` / `[~]` (référence) / `[!]`
(justification), zéro fix hors périmètre (section « Découvertes »), aucune case vide à la clôture.

**Statut : PROPOSÉ. Rien ne démarre avant le go de l'utilisateur ET les décisions D-1 à D-12 tranchées.**
Les chiffres datent du 2026-11-02 (`release/v8.1.0` à `792d2a0ee9`) : le lot 0 les ROUVRE.

## Objectif et critère de succès

Phase A (lots 1 à 3) : la brique `libs/film` ne nomme plus son hôte et n'expose plus de dette dans son API.
Phase B (lot 4) : `libs/film` devient la quatrième brique publiée (`v0.1.0`), comme `gamefiles`, `himap`, `haloapi`.
Phase C (lots 5 et 6) : les quatre briques vivent dans leur dépôt, LevelUp les consomme par version.

Succès = (1) `go list -deps -test ./...` de chaque brique sans `levelup/go-api` (déjà vrai, tenu) ; (2) aucun
littéral de l'hôte dans la production de `libs/film` (`LEVELUP_*` hors allowlist datée, chemins `apps/go-api`,
`internal/games/halo_infinite/film`, `C:\Users`) : ratchet à plafond 0 ; (3) **aucun changement observable** :
révisions des cinq couches inchangées, empreintes et goldens identiques (sauf régénération à révision constante
décidée en D-6), digests `replay-equiv` identiques, `replay-corpus-gate` 18/19 identiques ; (4) README EN, exemple
compilé, `go get github.com/JGtm/LevelUp/libs/film@v0.1.0` vérifié hors dépôt ; (5) quatre dépôts dédiés, tags
`v0.1.0` à leur nouveau chemin, LevelUp sans `libs/`, CI verte des cinq dépôts ; (6) ADR et docs à jour, registre
des reports et backlog statués.

Effort : phase A ~1,5 jour d'agent, phase B ~0,5 jour, phase C ~2 jours (le changement de chemin de module du
décodeur est le seul point lourd, à cause des empreintes de sources).

## Preuves de départ (2026-11-02, mesures du pilote, commandes rejouables au lot 0)

Décodeur stabilisé :
- Révisions : `source-2026-09-16.2`, `profile-2026-10-16`, `grammar-2026-10-28`, `killsource-2026-10-18`,
  `objectives-2026-10-17` (`grep -rn '^const Rev' libs/film --include=rev.go`). Aucune montée depuis D2.
- `libs/film/decfilm/decfilm.go` inchangé depuis le 2026-10-29 (`git log --since=2026-10-29 -- libs/film/decfilm`).
  Six commits touchent `libs/film` depuis : #116 (références `replay-equiv` re-figées), #127 et #134 (commentaires),
  #130, #131, #137 (desktop : zstd dans `source`, `filmcache` recompression et déplacement, `filmproc` sans fenêtre).

Surface :
- Façade `decfilm` : 179 déclarations exportées = plafond `plafondSurfaceFacade` (179) ; 176 identifiants
  `decfilm.X` distincts cités dans `apps/go-api`.
- Surface compagnon : 316 identifiants `replay.X` distincts hors du décodeur = `plafondSurfaceReplay` (316).
- Paquets publics : 18 (`decfilm`, `replay`, `filmcache`, `filmproc`, `finalise`, `types`, `damagetag`,
  `killicon`, `medalname`, `revision`, `filmshell`, `equipenergy`, `identity`, `playerposition`,
  `objectiveevent`, `highlightevent`, `filmobs`, `digest`) + `research/...` (8 paquets, 40 fichiers, TOUS sous
  `//go:build research`). Fermés : `internal/{source,profile,grammar,facts,constat,atomicfile,filelock,flatjson}`.
- Importeurs de production dans `apps/go-api` : `replay` 112 fichiers, `decfilm` 70, `filmcache` 36, `filmproc` 18,
  `finalise` 12, `types` 7, `objectiveevent` 7, `playerposition` 6, `filmshell` 5, les autres 1 à 4.
- Couches du décodeur : `filmcache` et `filmproc` ne sont importés en production que par `research` ; 88 fichiers
  de test de `grammar`, `replay`, `facts` les utilisent (`filmcache.LoadFilm` ×36, `filmproc.Arm` ×82).

Couplages à l'hôte :
- `go.mod` de `libs/film` : aucun `require` de `libs/gamefiles` (28 fichiers importent `mapvar`) ni de
  `libs/himap` (2 fichiers) ; tiers : `klauspost/compress`, `go-toml/v2`, `x/sys`.
- `filmproc/runner.go` nomme `LEVELUP_REPO_ROOT`, `LEVELUP_RESOURCES_DIR`, `LEVELUP_DATA_DIR`, `LEVELUP_LOGS_DIR`
  (les trois derniers ajoutés par le desktop) ; `revision/porte.go` construit `LEVELUP_UPDATE_<NOM>` (portes de
  régénération des goldens, outil du mainteneur).
- Ancien chemin `internal/games/halo_infinite/film` : 657 lignes dans 559 fichiers (24 de production), dont 33
  goldens dans leur en-tête ; `apps/go-api` cité dans 142 fichiers (8 de production) ; chemin absolu `C:\Users` ou
  `Projects\LevelUp` dans 7 fichiers (6 tests, 1 `research`).
- `filepath.Join(..., "data", ...)` : 44 fichiers (instruments `research` et tests) ; backlog : 9 fichiers visés.
- Registre des replis (`internal/facts/fallback/registre*.go`) : `Site.Fichier` et `Appelants` nomment des fichiers
  de LevelUp (D2-L3-f).

Dette d'API à solder avant un tag (un nom tagué se garde) :
- `filmshell` : 15 variables exportées modifiables (`WeaponIDs`, `WeaponTimingByID`, ...), exclues du ratchet
  `film_exported_vars` par `catalogueArmesHorsRatchet` (D2-L2-e).
- `filmproc` et la copie `internal/atomicfile` : 12 appels `slog` sans contexte, exclus par `horsPerimetreDeJournal`
  (D2-L2-d).
- 21 fichiers `//go:build research && campagne_overlay` qui ne compilent pas (symboles `bis2*` absents, D2-L0-b).
- Six copies de `racineDepot(t)` + un homonyme à `Fatal` (D2-E R2, D2-L2-n).

Données embarquées et fixtures :
- `damagetag/data/jpt_ids.txt` (468 ids lus dans les en-têtes des `.module` du jeu installé), `labels.tsv` (473 L),
  `killicon/data/rules.tsv` (159 L) : même question d'origine que PUB-4-a (ADR 0041 : aucun fichier extrait du jeu).
- `replay/testdata` : 29 Mo (8 minifilms et leurs sorties) ; 202 xuids distincts dans les fixtures. Le dépôt
  `JGtm/LevelUp` est PUBLIC (`gh repo view` : `PUBLIC`) : ces fixtures sont déjà publiées.
- Entrées du décodeur lues par chemin fourni par l'appelant (jamais résolu dans la brique) :
  `map_quant_bounds.json` (sans bornes, pas de coordonnée monde), `map_callouts.json`, `map_objectives.json`,
  `map_weapon_pads.json`, `equipment_energy.json`, fonds `map_backgrounds/` ; tous sous
  `data/titles/halo_infinite/reference/`, produits par des `cmd/*-build` de go-api.

Ratchets :
- D2-L3-c / PUB-3-d : des ratchets d'archlint balaient `apps/go-api/internal` seulement ; leur sujet existe dans
  `libs/*` ; inventaire jamais fait.
- D2-L3-a : l'ordre des imports de 15 sources hachées est dans l'empreinte, sans garde-rail dédié. Préalable du
  lot 6 (le changement de chemin de module réécrit ces imports).

Registre à re-statuer au lot 0 : D2-L0-a (références `replay-equiv` périmées) semble soldé par la PR #116 ;
`f75e7053` absent du parc (D2-L5-b) ; PUB-2.5-e et PUB-4-a ouverts.

### Relevé du lot 0 (2026-11-04, HEAD 421107998f)

HEAD = `origin/release/v8.1.0` (L22b, #143), CI de la base verte (runs `CI` et `Release` du 2026-11-03, `success`).
Les chiffres de la colonne « relevé » font foi pour la suite. Aucune révision de couche, aucune surface de façade ne
bouge depuis le 2026-11-02.

| Mesure | Valeur du 02/11 | Valeur relevée | Écart |
|---|---|---|---|
| Constantes `Rev` (`grep -rn '^const Rev' libs/film --include=rev.go`) | source-2026-09-16.2, profile-2026-10-16, grammar-2026-10-28, killsource-2026-10-18, objectives-2026-10-17 | identiques | 0 |
| `git log --since=2026-10-29 -- libs/film/decfilm` | vide | vide | 0 |
| Commits touchant `libs/film` depuis le 2026-10-29 | six : #116, #127, #134, #130, #131, #137 | les mêmes six (plus #114, D3, daté du 29/10 lui-même) ; aucun commit L20 à L22b sur `libs/film` | 0 |
| Déclarations exportées de la façade `decfilm` (`grep -cE '^(const\|type\|var\|func) [A-Z]'`) | 179 = `plafondSurfaceFacade` | 179 = 179 | 0 |
| Identifiants `decfilm.X` distincts cités dans `apps/go-api` (tests compris) | 176 | 176 (149 en production seule) | 0 |
| Identifiants `replay.X` distincts hors du décodeur | 316 = `plafondSurfaceReplay` | `TestSurfaceDeReplayCiteeHorsDuDecodeur` vert à 316 | 0 |
| Paquets publics de `libs/film` (hors `internal/`, `research/`) | 18 | 18 (mêmes noms) | 0 |
| `research/` | 8 paquets, 40 fichiers, tous sous `//go:build research` | 8 paquets, 40 fichiers hors tests (+ 5 `_test.go`), 45 sous tag, 0 sans tag | 0 (le 40 compte la production) |
| Paquets fermés `internal/` | 8 | 8 : atomicfile, constat, facts, filelock, flatjson, grammar, profile, source | 0 |
| Importeurs de production dans `apps/go-api` (fichiers non test, `internal` + `cmd`) | `replay` 112, `decfilm` 70, `filmcache` 36, `filmproc` 18, `finalise` 12, `types` 7, `objectiveevent` 7, `playerposition` 6, `filmshell` 5, autres 1 à 4 | `replay` 113, `decfilm` 70, `filmcache` 37, `filmproc` 21, `finalise` 13, `types` 7, `objectiveevent` 7, `playerposition` 6, `filmshell` 5 ; `damagetag` 4, `equipenergy` 3, `killicon` 2, `medalname` 2, `digest` 2, `identity` 1, `filmobs` 1, `highlightevent` 1, `revision` 0 | +1 `replay`, +1 `filmcache`, +3 `filmproc`, +1 `finalise` (le plan ne dit pas s'il comptait `cmd/`) |
| `filmcache` / `filmproc` importés en production par autre chose que `research` | non | non : 4 fichiers `research`, 0 hors `research` | 0 |
| Fichiers de test de `grammar`, `replay`, `facts` qui importent `filmcache` ou `filmproc` | 88 (`LoadFilm` ×36, `Arm` ×82) | 96 fichiers ; `LoadFilm` ×40 ; `Arm` ×82 | +8 fichiers, +4 `LoadFilm` |
| `go.mod` de `libs/film` | aucun `require` de `gamefiles` ni `himap` ; tiers `klauspost/compress`, `go-toml/v2`, `x/sys` | identique | 0 |
| Importeurs de `libs/gamefiles/mapvar` | 28 fichiers | 28 ; `libs/gamefiles` en tout : 32 fichiers | 0 |
| Importeurs de `libs/himap` | 2 fichiers | 5 : 2 en production (`replay/map_background.go`, `replay/structure.go`), 3 en test ou `research` | 0 en production (le plan ne comptait que la production) |
| Noms `LEVELUP_*` dans la production de `libs/film` | `filmproc/runner.go` (4 noms) et `revision/porte.go` | `filmproc/runner.go:53,59-61` (`LEVELUP_REPO_ROOT`, `_RESOURCES_DIR`, `_DATA_DIR`, `_LOGS_DIR`) et `revision/porte.go:49` (`LEVELUP_UPDATE_`) ; aucun autre fichier de production | 0 |
| Ancien chemin `internal/games/halo_infinite/film` | 657 lignes, 559 fichiers (24 de production), 33 goldens | 725 lignes, 593 fichiers (24 de production), 21 goldens (53 lignes) sur 34 `*.golden` | +68 lignes, +34 fichiers (cause non établie : la méthode de comptage du plan n'est pas écrite) ; goldens : 21 relevés, les 33 du plan ne se reproduisent pas |
| `apps/go-api` cité dans `libs/film` | 142 fichiers (8 de production), 145 lignes | 149 fichiers (8 de production) | +7 fichiers |
| Chemin absolu `C:\Users` ou `Projects\LevelUp` | 7 fichiers (6 tests, 1 `research`) | 7 fichiers : 6 `_test.go` + `research/cmd_rdata_weapon_scan/main.go` (motif `[A-Za-z]:.{1,3}Users\|Projects.{1,3}LevelUp`) | 0 |
| `filepath.Join(..., "data", ...)` | 44 fichiers ; backlog : 9 fichiers visés | 44 fichiers (0 de production) ; 9 fichiers `Join(depot, "data"` (8 sous `libs/film/replay`, 1 sous `apps/go-api/internal/replaybuild`) | 0 |
| Registre des replis : fichiers de LevelUp nommés | `Site.Fichier` et `Appelants` (qualitatif) | 6 lignes de production sous `internal/facts/fallback/` : `registre_killsource.go:9`, `registre_objectifs.go:21,205,280`, `registre_replay_vehicules.go:22` ; les autres sites (9 de `Fichier:` en `libs/film`) sont relatifs à la racine du dépôt | qualitatif confirmé |
| `filmshell` : variables exportées modifiables | 15 | 15 (`catalogue.go:47` à `:329`) | 0 |
| `slog` sans contexte dans `filmproc` + `internal/atomicfile` | 12 appels | 12 (`memguard.go` 2, `selfpriority_windows.go` 2, `solo.go` 4, `solo_observe.go` 2, `atomicfile.go` 2) | 0 |
| Fichiers `//go:build research && campagne_overlay` | 21 | 21 | 0 |
| Copies de `racineDepot(t)` | 6 + 1 homonyme à `Fatal` | 6 (`equipenergy`, `killsource`, `grammar`, `profile`, `killicon`, `replay`) + `objectives/families_parite_ts_test.go:44` | 0 |
| `damagetag/data/jpt_ids.txt`, `labels.tsv`, `killicon/data/rules.tsv` | 468 ids, 473 L, 159 L | 468 ids (472 L avec en-tête), 473 L, 159 L | 0 |
| `replay/testdata` | 29 Mo, 8 minifilms, 202 xuids distincts | 29,7 Mo (29 669 Ko), 8 dossiers `minifilm_*`, xuids : méthode d'origine non reproductible (290 valeurs de 16 chiffres derrière une clé `xuid`, 288 de motif `25\d{14}`) | xuids : +88 à +90 selon le motif, aucun ne reproduit 202 |
| Dépôt `JGtm/LevelUp` | PUBLIC | PUBLIC (`gh repo view`) | 0 |
| Entrées lues par chemin fourni par l'appelant | 6 noms de fichiers ou dossiers sous `data/titles/halo_infinite/reference/` | les six noms sont cités dans la production de `libs/film` ; produits par `cmd/{film-profiles,mapcallouts,mapfond,mapobj,mapopads,equipenergy}-build` | 0 |
| `go list -deps -test ./...` de chaque brique : `levelup/go-api` | aucun | aucun (film 274 paquets, gamefiles 149, haloapi 233, himap 154) | 0 |
| Registre : `f75e7053` | absent du parc | absent (0 dossier dans `film_chunks`, 0 manifeste) ; encore au manifeste `config/replay_corpus.toml:277` | 0 |
| Tags `libs/*/v0.1.0` (`git ls-remote --tags origin 'libs/*'`) | `gamefiles` et `haloapi` posés, `himap` après #119 | `libs/gamefiles/v0.1.0` (e2c449b3), `libs/haloapi/v0.1.0` (e2c449b3), `libs/himap/v0.1.0` (5c37ca56) tous posés | `himap` posé depuis le 02/11 |

## Décisions à trancher AVANT le go (recommandation du pilote en tête)

| # | Question | Recommandation | Alternative |
|---|---|---|---|
| D-1 | Périmètre public de `v0.1.0` | **Publier tel quel** : les 18 paquets, `research` sous tag, `v0.x` sans promesse ; les ratchets 179 / 316 tiennent la ligne (décision V25 du 2026-09-18 : pas de réduction) | Réduire à `decfilm` + `types` : défait V25, déplace `replay` (112 importeurs), effort L+ |
| D-2 | Le format Theater est le cœur de LevelUp : le rendre consommable par `go get` | **Oui** : le code est déjà public sous AGPL-3.0 ; la publication ajoute un README et une version | Garder la brique non publiée (ADR 0043 D-1 inchangé), ne faire que la phase A |
| D-3 | Noms `LEVELUP_*` dans la brique | **`filmproc` reçoit les noms de variables de l'appelant** (LevelUp passe les siens, même environnement d'enfant) ; **`revision/porte.go` garde `LEVELUP_UPDATE_*`** (commandes du mainteneur documentées), allowlist datée | Tout renommer en `FILM_*` : change les commandes du mainteneur |
| D-4 | Catalogues de référence et tables embarquées | **Catalogues restent dans LevelUp** ; le README décrit leur rôle et dit que l'appelant fournit le chemin. **Tables `damagetag` / `killicon` gardées** (identifiants et libellés, pas de contenu du jeu), origine écrite dans un `NOTICE.md` du module, même traitement que PUB-4-a | Embarquer `map_quant_bounds.json` (28 Ko) pour un exemple autonome |
| D-5 | 21 fichiers `campagne_overlay` | **Supprimer** (règle 7 ; ils ne compilent pas) | Versionner le recouvrement de campagne |
| D-6 | 33 en-têtes de goldens à l'ancien chemin | **Laisser** jusqu'à la prochaine régénération volontaire (les changer = régénération) | Régénérer à révision constante au lot 1 |
| D-7 | Fixtures avec 202 xuids de joueurs réels | **Garder** (déjà publiques dans le dépôt public ; identifiants Xbox publics) | Pseudonymiser : régénère fixtures et goldens, effort M |
| D-8 | Phase C | **Les quatre dépôts ensemble** (« jamais seul » du 2026-11-01 ; le décodeur requiert `gamefiles` et `himap`) ; nom `JGtm/halo-infinite-film-decoder` (« Halo Infinite Film Decoder ») | Trois dépôts d'abord, le décodeur ensuite |
| D-9 | Chemin de module après séparation | **Nouveau chemin** `github.com/JGtm/<dépôt>`, `v0.1.0` dans chaque dépôt ; ancien chemin : tag `libs/<brique>/v0.1.1` dont le `go.mod` porte `// Deprecated: moved to ...`, posé AVANT la suppression dans LevelUp | Garder le chemin avec `replace` : impossible à servir hors du dépôt LevelUp |
| D-10 | Développement croisé après séparation | **LevelUp requiert les tags** ; un fichier `go.work` hors dépôt (variable `GOWORK`) pour travailler sur une brique et LevelUp ensemble ; `go.work` du dépôt supprimé | Garder des copies locales en `replace` commités |
| D-11 | PUB-2.5-e : le jeton Spartan réémis vers `image.halocdn.com` sur 302 | **Le `Doer` de LevelUp retire `Authorization` / en-têtes Spartan sur changement d'hôte** ; T2 amendé de ce seul point | Garder le comportement, l'écrire dans le README de `haloapi` |
| D-12 | Calendrier vis-à-vis du desktop (L21 en cours, il touche `filmproc`, `filmcache`, `source`) | **Lots 0 et 1 maintenant** (worktree dédié, aucun fichier commun avec L21) ; **lots 2 à 4 après la fusion du dernier lot desktop qui touche `libs/film`** ; **phase C après la clôture du desktop ET le passage `release/v8.1.0` → `main`** | Tout après le desktop (décision du 2026-11-01 telle quelle) |

### Décisions TRANCHÉES le 2026-11-04 (message du user, après reformulation en cinq questions) — FONT FOI

- **Réorganisation protocole / exploitation AVANT le tag : VALIDÉE** (remplace D-1 « tel quel » ; la réduction
  de la façade du 18/09 reste NON retenue : le document de rejeu est la sortie, ses types restent publics). Grille
  de l'ADR 0042 appliquée à la brique : elle ne décide ni quand on décode, ni où on stocke, ni avec quelle
  priorité. Détail au lot 2.
- D-2 publier : oui (demande du 2026-11-01). D-3 : réglé par la réorganisation (le lanceur de processus et ses
  variables `LEVELUP_*` partent dans LevelUp ; `revision/porte.go` garde `LEVELUP_UPDATE_*`). D-4 : reco. D-5
  (21 fichiers `campagne_overlay`) : **supprimer**. D-6 : reco (laisser). D-8 : **`JGtm/halo-infinite-film-decoder`**,
  les quatre ensemble. D-9, D-10 : reco. D-11 : **corriger** (lot 5.1). D-12 : **continuer maintenant**, lots 1 à 4
  sans attendre le desktop (au 2026-11-04 : L21, L22, L22b fusionnés, `libs/film` inchangé depuis le 02/11, plan
  desktop réduit à deux gates du user) ; phase C après le desktop et `main`.
- **D-7 : les identifiants Xbox réels des fixtures sont À REMPLACER par des faux** (contre la reco). Les identités
  sont dans les octets des films de test, pas seulement dans les goldens : l'item 1.6 MESURE le coût (positions
  connues de la grammaire, témoins à régénérer) ; le pilote revient vers le user avec le coût avant tout
  remplacement (item 4.0). Si hors de prix : seule autre option propre, d'autres films joués par des comptes du user.

## Pilotage

Le pilote (Opus, session principale) ne code pas : il lance les exécutants, vérifie chaque compte rendu sur pièces
(rouvre fichier, ligne, sortie de gate), statue les items, tient le journal et parle au user en résultats.

| Lot | Exécutant | Pourquoi ce modèle |
|---|---|---|
| 0 relevé, témoins, inventaire des ratchets | Sonnet | Mesures et captures scriptées, gates explicites |
| 1 couplages à l'hôte | Sonnet (script de réécriture déterministe) ; Haiku pour 1.5 | Réécriture mécanique ; la preuve est l'empreinte inchangée |
| 2.1 `filmshell` en accesseurs | **Opus** | Touche des sources hachées de `grammar` : régénération à révision constante, porte à deux verrous (procédure D2 2.12) |
| 2.2 à 2.4 | Sonnet | Changements de signature mécaniques, tests existants |
| 2.5 suppression `campagne_overlay` | Haiku | `git rm` d'une liste fermée + un `go vet` |
| 3 ratchets | Sonnet | Tests neufs avec preuve par mutation |
| 4 README, exemple, `require`, ADR | Sonnet ; Haiku pour 4.8 (propagation docs contrôlée par grep) | Rédaction sur modèle existant |
| 5 préalables aux dépôts | Sonnet ; Haiku pour 5.2 | Petit correctif HTTP avec test ; documentation |
| 6.3 séparation du décodeur (chemin de module + empreintes) | **Opus** | Seul point où une erreur change les révisions de couche |
| 6.3 séparation des trois autres, 6.4 à 6.7 | Sonnet | Recette répétée |
| Revues adversariales (fin de phase B, fin de phase C) | 2 relecteurs Sonnet aveugles, une ronde | Règle du 2026-10-21 |

Coût annoncé : ~10 agents Sonnet, 2 Opus, ~4 Haiku sur l'ensemble, en séquence (pas de workflow). Un exécutant à
la fois, sauf la revue (deux relecteurs en parallèle, lecture seule). Un worktree par exécutant qui écrit.

## Branches

- Phases A et B : `refactor/brique-film-publication` depuis `release/v8.1.0`, worktree `../LevelUp-wt-film`,
  un commit par lot, PR squash vers la release à la fin du lot 4. Jamais `main`.
- Phase C : `refactor/depots-dedies` depuis `release/v8.1.0` (ou la release suivante) une fois le desktop clos et
  la release passée sur `main` par l'utilisateur.
- Caches : `GOCACHE=%LOCALAPPDATA%\go-build-film`, `GOLANGCI_LINT_CACHE=%LOCALAPPDATA%\golangci-film` ; purge à la
  clôture (`scripts/disk-hygiene.ps1 -Lot film -Apply`).
- CGO sous PowerShell : `C:\msys64\ucrt64\bin` en tête du PATH + `CGO_ENABLED=1` pour `apps/go-api`.

## Témoins avant/après (capturés au lot 0, comparés à chaque gate)

| Témoin | Capture | Attendu |
|---|---|---|
| T1 tests | `go test -json ./... -count=1` dans `apps/go-api` et les quatre `libs/*` → (paquet, test, verdict) | mêmes noms et verdicts, 0 fail ; disparitions listées une à une |
| T2 révisions | `go test ./revision/... ./internal/... -run 'Rev|Empreinte|Perimetre'` + copie des goldens de révision et de périmètre | constantes `Rev` inchangées ; goldens identiques (hors régénération actée au lot 2.1) |
| T3 rejeu | `make replay-equiv` : MD5 des 20 fichiers de digests | identiques |
| T4 parc | `make replay-corpus-gate` (`-allow-missing` tant que `f75e7053` manque) | 18/19 identiques, 0 perte, 0 gain |
| T5 autonomie | `go list -deps -test ./...` par brique | aucun `levelup/go-api` |
| T6 surface | `plafondSurfaceFacade`, `plafondSurfaceReplay` | 179 et 316 (sauf décision écrite : la façade passe à 181 le 2026-11-08, décision du pilote, item 4.6b) |
| T7 intégration | `go test -tags=integration -p 1 ./internal/sync/... ./internal/persist/...` | code 0 |


### Capture du lot 0 (2026-11-04, HEAD 421107998f)

Dossier (hors dépôt, 39 Mo) : `C:\Users\GUILLA~1\AppData\Local\Temp\claude\c--Users-Guillaume-Downloads-Scripts-LevelUp\0fb89d41-4333-4829-a27d-6f5539698975\scratchpad\temoins_film\`.
Durée cumulée des captures : ~92 min (T1 1 105 s, T3 1 036 s, T4 1 231 s, T4b 1 374 s, T6 69 s, T7 694 s).

| Témoin | Valeur capturée | Fichiers (dans le dossier ci-dessus) |
|---|---|---|
| T1 tests | **21 556 lignes (paquet, test, verdict)** : 21 176 `pass`, **0 `fail`**, 380 `skip`. `apps/go-api` 17 308 (349 paquets), `libs/film` 3 874 (33 paquets), `libs/gamefiles` 85, `libs/haloapi` 216, `libs/himap` 73. MD5 de la liste triée : `f1f0ed84c7592b8a14f75a6b067a4aed`. Le paquet `internal/archlint` a dépassé `-timeout 10m` dans le `./...` complet de `go-api` (16 paquets en parallèle, 259 tests terminés, panique « test timed out after 10m0s ») : il a été rejoué seul, `go test -json ./internal/archlint -count=1 -timeout 40m`, 487 tests, 487 `pass`, 286 s, et la liste finale prend ces lignes (voir Découvertes) | `T1_final_tous.tsv`, `T1_final_apps_go-api.tsv`, `T1_tests_libs_*.tsv`, `T1_raw_*.json`, `T1_timing.txt` |
| T2 révisions | cinq `Rev` : `source-2026-09-16.2`, `profile-2026-10-16`, `grammar-2026-10-28`, `killsource-2026-10-18`, `objectives-2026-10-17` ; **34 `*.golden`** copiés avec leur SHA-256 ; MD5 de la liste `f8dffb8e4ad6d8c77598cd159a4f061d` | `T2_revisions.txt`, `T2_golden_sha256.txt`, `T2_goldens/` |
| T3 rejeu | `go run ./cmd/replay-equiv -out-dir …` (le `make replay-equiv` sans option ne conserve pas les digests) : **20 identique(s), 0 different(s), 0 ecarte(s), 0 echec(s), 0 illisible(s)**, 1 036 s ; **20 fichiers de digests, MD5 dans `T3_md5.txt`** (MD5 de cette liste `e35052a1baa65ea8519fa373d8e51271`), et les 20 fichiers sont identiques à l'octet aux 20 références `libs/film/replay/testdata/equivalence/*.tsv` (`cmp`) | `T3_replay_equiv.txt`, `T3_digests/`, `T3_md5.txt` |
| T4 parc | `make replay-corpus-gate ARGS="--allow-missing …"` : manifeste de 19 témoins, 18 cuits, `f75e7053` ABSENT (couverture complète : `couverture_incomplete=false`). **Base par défaut `origin/main` (`413971bcc4`, v8.0.7), schéma 104 -> 106 : 5 ok, 8 PERTE, 5 FAUX** (dérive de couverture `carriedEquipment.trackedUnpositioned` / `trackedEffectsUnlinked` / `fallbacks/n` entre v8.0.7 et v8.1.0, code 1, 1 231 s) : ce n'est pas un écart du lot, c'est l'écart de deux versions. **Base = HEAD (`--base 421107998f`, la référence des gates suivants) : 18 identiques, 0 gain, 0 perte, 0 changement, 1 absent, code 0, 1 374 s.** Les gates suivants passent `--base <sha du commit lot 0>` | `T4_gate.txt`, `T4_rapport.json`, `T4b_gate.txt`, `T4b_rapport.json`, `T4_summary/` |
| T5 autonomie | `go list -deps -test ./...` : `levelup/go-api` = **0** dans `libs/film` (274 paquets), `libs/gamefiles` (149), `libs/haloapi` (233), `libs/himap` (154) | `T5_autonomie.txt`, `T5_deps_*.txt` |
| T6 surface | `plafondSurfaceFacade = 179`, `plafondSurfaceReplay = 316` (`film_facade_surface_test.go:155` et `:422`) ; `go test ./internal/archlint -run 'Facade\|Surface' -count=1 -v` : `TestSurfaceDeLaFacadeDuDecodeur` PASS, `TestSurfaceDeReplayCiteeHorsDuDecodeur` PASS (45 s) | `T6_surface.txt` |
| T7 intégration | `go test -tags=integration -p 1 -json ./internal/sync/... ./internal/persist/... -count=1` : **code 0**, 16 paquets `pass`, 2 321 tests : 2 289 `pass`, 0 `fail`, 32 `skip`, 694 s | `T7_raw.json`, `T7_tests.tsv`, `T7_timing.txt` |

Montage du parc dans le worktree (nécessaire : un worktree dédié ne porte ni `db_profiles.json`, ni les chunks, ni la
base partagée ; rien n'a été écrit dans le checkout principal) : jonctions NTFS
`data/cache/film_chunks` et `data/cache/film_manifests` vers celles du checkout principal (lecture seule),
copie de `shared_matches_v2.duckdb` et `metadata.duckdb` sous `data/titles/halo_infinite/warehouse/` (fichiers ignorés
par git), `LEVELUP_REPO_ROOT=<worktree>` pour `replay-equiv`, `--parc-root <worktree> --lock-root <worktree>/data/cache`
pour le gate. Les jonctions sont retirées à la fin du lot (`cmd /c rmdir`, jamais `rm -r`) ; recréation :
`cmd /c mklink /J <worktree>\data\cache\film_chunks <principal>\data\cache\film_chunks` (idem `film_manifests`).

## Lot 0 — Préalable : branche, relevé, témoins

- [x] 0.1 Worktree, branche, caches ; `git log --oneline -5` ; CI de la base verte. — branche `refactor/brique-film-publication` à `421107998f` = `origin/release/v8.1.0` ; CI `CI` et `Release` du 2026-11-03 : success ; caches `go-build-film`, `golangci-film`.
- [x] 0.2 Rejouer chaque mesure de « Preuves de départ » ; écrire l'écart sous la section, la mesure du lot fait foi. — « Relevé du lot 0 » sous « Preuves de départ ».
- [x] 0.3 Capturer T1 à T7 sous `.ai/V8/temoins_film_2026-11/` (hors dépôt si > 1 Mo : scratchpad, chemin noté). — « Capture du lot 0 » sous « Témoins avant/après » (scratchpad, 39 Mo, chemin noté).
- [x] 0.4 Statuer au registre D2-L0-a (PR #116), le 21e `.tsv` de `replay-equiv`, D2-L5-b (`f75e7053`). — D2-L0-a et le 21e `.tsv` (`minifilm.tsv`, référence d un test) soldés, D2-L5-b précisé (toujours absent) au registre.
- [x] 0.5 Inventaire des ratchets (D2-L3-c, PUB-3-d) : pour chaque test de `apps/go-api/internal/archlint` et des
      ratchets hors archlint, une ligne (test, racine balayée, sujet présent dans `libs/*` oui/non). Tableau versé
      dans ce plan. Commande de départ : `grep -lE 'internal/|"apps/go-api' apps/go-api/internal/archlint/*_test.go`.
      Résultat : « Inventaire des ratchets (0.5) » ci-dessous, 3 lignes « oui » à racine go-api seul (périmètre du lot 3.2).
- [x] 0.6 `GOWORK=off go build ./...` dans `libs/film` avec un `go.mod` temporaire qui requiert `gamefiles` et
      `himap` à `v0.1.0` : les symboles utilisés existent-ils dans les versions taguées ? Résultat noté (si non :
      tags `v0.1.1` à demander au user avant le lot 4).
- [x] 0.7 Gate : témoins écrits, inventaire versé, journal. Commit « lot 0 ». — entrée du journal écrite, commit « lot 0 » (le sha est dans le compte rendu du pilote).

### Résultat de l'item 0.6 (2026-11-04)

Copie jetable de `libs/film` sous le scratchpad (`film_copy/film`), `go.mod` complété de
`require github.com/JGtm/LevelUp/libs/gamefiles v0.1.0` et `require github.com/JGtm/LevelUp/libs/himap v0.1.0` ; le
`go.mod` du worktree n'est pas touché (`git diff -- libs/film/go.mod` vide).
Environnement : `GOWORK=off GOFLAGS=-mod=mod GOPROXY=direct GONOSUMDB=github.com/JGtm/LevelUp`.
Tags résolus : `libs/gamefiles/v0.1.0` (`e2c449b3`), `libs/himap/v0.1.0` (`5c37ca56`), tous deux présents sur `origin`
(`git ls-remote --tags origin 'libs/*'`).

| Commande | Résultat |
|---|---|
| `go build ./...` | code 0, aucun symbole manquant |
| `go vet ./...` (tests compilés) | code 0 |
| `go vet -tags research ./...` | code 0 |
| `go list -m all` | `libs/gamefiles v0.1.0`, `libs/himap v0.1.0`, `klauspost/compress v1.19.0`, `go-toml/v2 v2.4.3`, `x/sys v0.49.0` |

Les symboles de `gamefiles` (28 fichiers de `libs/film`) et de `himap` (5 fichiers) existent dans les versions taguées :
aucun tag `v0.1.1` à demander au user avant le lot 4. Sorties : `temoins_film/0.6_build.txt`, `0.6_vet.txt`,
`0.6_vet_research.txt` (scratchpad).

### Inventaire des ratchets (0.5)

Relevé du 2026-11-04 (HEAD 421107998f). Source : les 206 fichiers `apps/go-api/internal/archlint/*_test.go`, plus les tests hors archlint qui balaient un arbre de fichiers (`filepath.Walk`/`WalkDir`) : `grep -rlE 'filepath\.Walk(Dir)?\(|WalkDir\(' --include=*_test.go apps/go-api/internal apps/go-api/cmd` rend 46 fichiers hors archlint : 36 listés au tableau B (33 balaient des sources Go, 3 des sources web ou des fixtures), 10 non listés (répertoires temporaires, e2e, caches d'assets, aide du corpus gamefiles). Colonnes : test ; racine(s) balayée(s) ; sujet présent dans `libs/*` (jugé sur le motif que le test interdit, vérifié par grep ou par comptage dans `libs/*` : « oui » = le motif existe en production dans une brique ; « oui (couvert) » = la racine du test inclut déjà la brique concernée ; « oui (racine partielle) » = couvert pour `libs/film` mais le motif existe aussi dans une autre brique que la racine n'inclut pas ; « sans objet » = pas un balayage de sources Go ; « non »).

Ne sont pas listés : les ~50 tests hors archlint qui lisent les `.go` de leur propre paquet par `os.ReadDir` (un dossier, pas un arbre).

#### Tableau A — `archlint` (206 fichiers)

| Test | Racine(s) balayée(s) | Sujet dans `libs/*` | Note |
|---|---|---|---|
| `admin_handler_sessions_wired_test.go` | go-api seul (internal, cmd) | non | handler HTTP d'administration |
| `aides_uniques_film_test.go` | libs/film (tous les .go) | oui (couvert) | aides de test uniques du décodeur |
| `ambiguous_path_middleware_test.go` | go-api seul (internal/api/middleware) | non | middleware HTTP |
| `briques_garde_test.go` | libs/ (tout dossier portant un .go) | oui (couvert) | table des paquets de briques |
| `briques_imports_test.go` | espace de travail (go-api + 4 libs) | oui (couvert) | imports d'une brique |
| `campaign_exclusion_guard_dispenses_test.go` | aucun arbre Go balayé (table de dispenses) | sans objet | données du garde |
| `campaign_exclusion_guard_registre_test.go` | aucun arbre Go balayé (test unitaire du critère) | sans objet |  |
| `campaign_exclusion_guard_test.go` | go-api seul (internal, racinesExclusionCampagne) | non | lecteurs SQL DuckDB |
| `camps_canonical_single_source_test.go` | go-api seul (internal, cmd) | non | ligne canonique de tactique |
| `checkpoint_single_site_test.go` | go-api seul (cmd, internal, pkg, scripts) | non | CHECKPOINT DuckDB ; aucune brique n'ouvre DuckDB |
| `childproc_single_spawn_test.go` | go-api (internal) + 4 libs | oui (couvert) | enfant du serveur ; libs/* déjà dans les racines |
| `ci_deploy_triggers_test.go` | aucun arbre Go balayé (workflows YAML) | sans objet |  |
| `ci_go_shards_test.go` | aucun arbre Go balayé (go-shards.txt, workflows) | sans objet | nomme libs/film en métadonnée |
| `ci_paths_filter_test.go` | aucun arbre Go balayé (workflows YAML) | sans objet |  |
| `ci_web_reads_go_test.go` | aucun arbre Go balayé (sources web, workflows) | sans objet |  |
| `ci_workflows_security_test.go` | aucun arbre Go balayé (workflows YAML) | sans objet |  |
| `client_ip_test.go` | go-api seul (module) | non | HTTP |
| `commentaires_references_test.go` | espace de travail (balayerGo) | oui (couvert) | citations .ai dans les commentaires |
| `compact_passes_nuit_script_test.go` | aucun arbre Go balayé (script hôte) | sans objet |  |
| `compose_hardening_test.go` | aucun arbre Go balayé (docker-compose.yml) | sans objet |  |
| `copies_identiques_test.go` | libs/film (internal/) contre go-api | oui (couvert) | compare deux copies de fichier |
| `decode_lock_interdit_test.go` | go-api seul (internal, cmd) | oui | verrou de paquet du décodeur (LockProcessDecode) : le lieu du sujet est libs/film/internal/grammar, hors racine |
| `deploy_root_copie_unique_test.go` | espace de travail (balayerGoTexte) | oui (couvert) | chemin du jeu |
| `deployment_single_resolver_test.go` | go-api seul (internal, cmd, pkg, tools, scripts) | non | profil de déploiement |
| `dockerfile_go_version_test.go` | aucun arbre Go balayé (Dockerfile, go.mod, go.work) | sans objet |  |
| `enregistreur_http_unique_test.go` | go-api + libs/haloapi | oui (couvert) | enregistreur de tests HTTP |
| `err_error_response_test.go` | go-api seul (internal/api, cmd/server) | non | texte d'erreur renvoyé au client |
| `fallback_versement_test.go` | go-api + libs/film (fichiers nommés) | oui (couvert) | versement des replis |
| `favorite_weapon_rule_test.go` | go-api seul (platform/duckdb, ReadDir) | non |  |
| `ffmpeg_single_launcher_test.go` | go-api seul (cmd, internal, pkg) | non | aucun ffmpeg dans libs/* |
| `ffprobe_single_launcher_test.go` | go-api seul (cmd, internal, pkg) | non | aucun ffprobe dans libs/* |
| `film_balayage_test.go` | libs/film | oui (couvert) | production du décodeur hors research |
| `film_balayage_unique_test.go` | archlint lui-même (lit ses fichiers) | sans objet |  |
| `film_consultations_comptees_test.go` | go-api (replaybuild) + libs/film/replay | oui (couvert) |  |
| `film_exported_vars_test.go` | go-api (replaybuild, killcollector) + libs/film | oui (racine partielle) | variables exportées modifiables : 9 dans libs/himap (himap/*.go), hors racine |
| `film_facade_surface_test.go` | libs/film (decfilm) + citations dans go-api | oui (couvert) |  |
| `film_faits_sans_octets_test.go` | libs/film (internal/facts) | oui (couvert) |  |
| `film_file_size_test.go` | go-api (replaybuild, killcollector) + libs/film | oui (racine partielle) | seuil 500 L : 3 fichiers de production > 500 L dans libs/himap, hors racine |
| `film_finalise_predicate_test.go` | go-api (cmd, internal) + libs/film | oui (couvert) |  |
| `film_function_length_test.go` | go-api (replaybuild, killcollector) + libs/film | oui (racine partielle) | seuil 80 L : 1 fonction de production > 80 L dans libs/himap, hors racine |
| `film_garde_de_tampon_test.go` | libs/film (un fichier) | oui (couvert) |  |
| `film_handle_lecture_test.go` | libs/film (un fichier) | oui (couvert) |  |
| `film_lane_duckdb_free_test.go` | libs/film (go list -deps) | oui (couvert) |  |
| `film_layers_deps_helpers_test.go` | helper de film_layers_deps | sans objet |  |
| `film_layers_deps_test.go` | libs/film + go-api/internal/analysis | oui (couvert) |  |
| `film_lecture_test.go` | libs/film + go-api | oui (couvert) |  |
| `film_localisateur_unique_test.go` | libs/film | oui (couvert) |  |
| `film_no_slog_in_layers_test.go` | libs/film (grammar, facts) | oui (couvert) |  |
| `film_pont_identite_test.go` | go-api (killcollector) + libs/film/replay | oui (couvert) |  |
| `film_slog_contexte_test.go` | go-api (replaybuild) + libs/film | oui (couvert) |  |
| `film_tri_total_test.go` | libs/film | oui (couvert) |  |
| `film_types_leaf_test.go` | libs/film/types | oui (couvert) |  |
| `film_vue_a_lecteur_unique_test.go` | libs/film | oui (couvert) |  |
| `filmdec_package_vars_test.go` | libs/film/internal/grammar | oui (couvert) |  |
| `filmsource_leaf_test.go` | libs/film/internal/source | oui (couvert) |  |
| `gamefiles_tag_test.go` | espace de travail | oui (couvert) | tag gamefiles |
| `go_source_walk_test.go` | helpers de balayage (pas un ratchet) | sans objet | définit workspaceGoSources, balayerGo, goSourcesUnder |
| `haloapi_doer_garde_test.go` | go-api (internal, cmd) + libs/haloapi | oui (couvert) |  |
| `haloapi_fermeture_domaine_test.go` | go list : domain, analysis, canonical | sans objet | sens des dépendances |
| `haloapi_masque_unique_test.go` | libs/haloapi | oui (couvert) |  |
| `haloapi_parametres_test.go` | libs/haloapi | oui (couvert) |  |
| `haloapi_sans_client_http_test.go` | libs/haloapi | oui (couvert) |  |
| `haloapi_sans_etat_test.go` | libs/haloapi (xblauth) | oui (couvert) |  |
| `i18n_halo_literal_test.go` | go-api seul (platform/duckdb, service) | non |  |
| `i18n_langue_suffixe_test.go` | go-api + libs/film | oui (couvert) | suffixes _fr/_en absents des autres briques |
| `i18n_locale_constant_test.go` | go-api seul (cmd, internal, pkg, tools, scripts) | non | i18n.LocaleFR n'existe pas dans libs/* |
| `i18n_messages_test.go` | go-api seul (cmd, internal, pkg, tools, scripts) | non | catalogue de messages Go |
| `i18n_placement_single_source_test.go` | go-api seul (cmd, internal, pkg, tools, scripts) | non |  |
| `i18n_text_sql_test.go` | go-api seul (module) | non |  |
| `i18n_tier_label_stored_not_served_test.go` | go-api seul (internal/platform) | non |  |
| `invite_code_log_test.go` | go-api seul (module) | non |  |
| `job_error_fixed_message_test.go` | go-api seul (internal/api, service, sync) | non |  |
| `keyframe_walk_proof_test.go` | libs/film (grammar, facts) | oui (couvert) |  |
| `kill_death_tolerance_test.go` | go-api seul (module) | non | ComputeKillerVictimPairs : 1 mention en commentaire dans libs/film |
| `kill_feed_summaries_guard_test.go` | go-api seul (platform/duckdb, api, service, analysis) | non | SQL |
| `killsource_nom_de_remplissage_formes_test.go` | auto-preuve (formes de contournement) | sans objet |  |
| `killsource_nom_de_remplissage_test.go` | libs/film (killsource) | oui (couvert) |  |
| `killsource_recherche_sans_carte_test.go` | go-api (internal, cmd) + libs/film | oui (couvert) |  |
| `known_set_single_source_test.go` | go-api seul (internal, cmd) | non | SQL |
| `liste_liee_ratchet_test.go` | go-api seul (internal) | non | paramètre liste DuckDB |
| `lusr_medal_exploit_option_test.go` | go-api seul (internal, cmd) | non |  |
| `map_label_single_resolution_test.go` | go-api seul (internal, cmd) | non |  |
| `match_id_form_single_source_test.go` | go-api seul (internal, cmd) | non | aucune fonction ni regexp d'identifiant de match dans libs/* |
| `match_registry_ddl_types_test.go` | go-api seul (deux fichiers DDL) | non |  |
| `match_view_repo_erreurs_remontees_test.go` | go-api seul (platform/duckdb/match_view_repo*.go) | non |  |
| `media_binpath_single_resolver_test.go` | go-api seul (internal, cmd) | non |  |
| `media_ext_list_single_source_test.go` | go-api seul (internal, cmd) | non |  |
| `mode_gamevariant_fallback_test.go` | go-api seul (internal) | non |  |
| `nginx_assets_v_test.go` | aucun arbre Go balayé (nginx) | sans objet |  |
| `nginx_conf_parse_test.go` | aucun arbre Go balayé (nginx) | sans objet |  |
| `nginx_conf_test.go` | aucun arbre Go balayé (nginx) | sans objet |  |
| `no_absolute_path_in_versioned_catalog_test.go` | aucun arbre Go balayé (catalogues JSON/TOML du dépôt) | sans objet |  |
| `no_accidental_goos_suffix_test.go` | espace de travail | oui (couvert) |  |
| `no_ad_hoc_source_fingerprint_test.go` | go-api (internal, cmd) + libs/film | oui (couvert) |  |
| `no_analysis_type_in_http_body_test.go` | go-api seul (internal/api) | non |  |
| `no_artefact_reread_in_cmd_test.go` | go-api seul (cmd) | non |  |
| `no_auto_film_recompress_test.go` | go-api seul (cmd, internal) | non | la recompression vit dans libs/film/filmcache ; aucun appelant automatique dans libs/* |
| `no_bare_duckdb_rw_open_test.go` | go-api seul (internal, cmd, pkg, scripts) | non | aucune brique n'ouvre DuckDB |
| `no_bare_instance_lock_read_test.go` | go-api seul (module) | non |  |
| `no_bare_resolve_mode_ui_test.go` | go-api seul (internal) | non |  |
| `no_bare_settings_write_test.go` | go-api seul (internal, paquets nommés) | non | vise les fichiers de réglages ; os.Rename de libs/film/filmcache/deplacement.go n'en est pas un |
| `no_bot_identity_projection_outside_killsource_test.go` | go-api seul (internal, cmd) | non |  |
| `no_collecteur_sans_capture_test.go` | go-api seul (module) | non |  |
| `no_cuisson_depuis_tactique_test.go` | go-api seul (module) | non |  |
| `no_data_path_join_test.go` | go-api seul (internal, cmd/server) | non | le ratchet exclut les _test.go ; libs/film : 0 fichier de production, 44 fichiers de test/instruments |
| `no_decode_in_api_test.go` | go-api seul (internal/api) | non |  |
| `no_demo_forbidden_literal_test.go` | go-api seul (internal, cmd, pkg) | non |  |
| `no_demo_layout_translation_test.go` | go-api seul (internal, cmd) | non |  |
| `no_direct_profile_create_test.go` | go-api seul (module) | non |  |
| `no_discarded_remove_test.go` | go-api seul (internal/api, internal/service) | oui | `_ = os.Remove` en production : libs/film/filmcache/deplacement.go (6), libs/film/internal/atomicfile/atomicfile.go (2) |
| `no_duckdb_import_playerdirectory_test.go` | go-api seul (un paquet) | non |  |
| `no_duplicate_static_ext_list_test.go` | go-api seul (internal, cmd) | non |  |
| `no_error_text_classification_test.go` | go-api seul (replaychild, sync/replayartifacts) | non | aucun strings.Contains(err.Error()) en production dans libs/* |
| `no_film_reread_test.go` | go-api + libs/film | oui (couvert) |  |
| `no_flag_carry_end_outside_close_test.go` | libs/film/replay | oui (couvert) |  |
| `no_french_label_literal_test.go` | go-api (service, analysis, api/handlers, notify, games) + libs/film | oui (couvert) |  |
| `no_global_friend_gamertags_test.go` | go-api seul (module) | non |  |
| `no_group_by_registry_name_test.go` | go-api seul (internal, cmd) | non | SQL |
| `no_halowaypoint_literal_test.go` | go-api (internal, cmd, scripts) + libs/haloapi | oui (couvert) |  |
| `no_hardcoded_base_weapons_test.go` | go-api seul (analysis, sync/replayartifacts, service) + web | non |  |
| `no_hardcoded_film_cache_dirs_test.go` | go-api + libs/film | oui (couvert) |  |
| `no_identity_bridge_outside_registry_test.go` | go-api (killcollector, replaybuild) + libs/film/replay | oui (couvert) |  |
| `no_inline_expected_fda_test.go` | go-api seul (internal) | non |  |
| `no_inline_objective_latest_view_test.go` | go-api seul (internal) | non | DDL |
| `no_keyed_mutex_registry_test.go` | go-api seul (internal, cmd) | non | aucune map de verrous par clé en production dans libs/* |
| `no_life_cause_divergence_test.go` | go-api + libs/film/replay | oui (couvert) |  |
| `no_literal_base_tenue_test.go` | go-api seul (cmd, internal) | non | erreur DuckDB |
| `no_local_longest_run_test.go` | go-api (internal, cmd) + libs/film | oui (couvert) |  |
| `no_local_portee_comparison_test.go` | go-api seul (internal, cmd) | non | coordination.APortee ; aucune comparaison de portée de radar en production dans libs/* |
| `no_local_ptr_helper_test.go` | go-api seul (internal, cmd) | non | motif exact strPtr ; libs/film/replay/document_translocations.go:120 porte coordPtr (autre nom) |
| `no_local_radar_range_lookup_test.go` | go-api seul (internal, cmd) | non |  |
| `no_local_sidecar_freshness_test.go` | go-api seul (module) | non |  |
| `no_local_spawn_depart_test.go` | go-api seul (module) | non |  |
| `no_local_sql_splitter_test.go` | go-api seul (internal, cmd) | non |  |
| `no_local_str_coalesce_test.go` | go-api seul (internal/service) | non |  |
| `no_local_upsert_helper_test.go` | go-api seul (internal, cmd) | non |  |
| `no_local_xuid_pattern_test.go` | go-api seul (internal) | non | libs/haloapi/infinite/stats.go:37 `^xuid\(\d+\)$` n'est pas le motif visé |
| `no_locale_literal_test.go` | go-api seul (cmd, internal, pkg, tools, scripts) | non | 0 comparaison de langue en production dans libs/* (un commentaire dans libs/film/replay/document_labels_json.go) |
| `no_map_variant_suffix_literal_test.go` | go-api (internal, cmd) + libs/film | oui (couvert) |  |
| `no_metadata_readonly_in_sync_test.go` | go-api seul (internal/sync) | non |  |
| `no_mojibake_test.go` | espace de travail (.go) + config/titles (.toml) | oui (couvert) |  |
| `no_objective_submode_list_test.go` | go-api seul (module) | non |  |
| `no_os_is_predicates_test.go` | libs/film + go-api (cmd/replay-*) | oui (couvert) | os.IsNotExist/IsExist/IsPermission : 0 dans gamefiles, himap, haloapi |
| `no_player_index_identity_test.go` | libs/film (grammar, replay) | oui (couvert) |  |
| `no_players_root_join_test.go` | go-api seul (internal, cmd) | non |  |
| `no_pool_hasplayer_gate_test.go` | go-api seul (module) | non |  |
| `no_racine_absente_toleree_test.go` | archlint lui-même (lit ses fichiers) | sans objet |  |
| `no_raw_combat_dr_threshold_test.go` | go-api seul (internal, cmd) | non |  |
| `no_raw_combat_oc_threshold_test.go` | go-api seul (internal, cmd) | non |  |
| `no_raw_film_bytes_extraction_test.go` | go-api + libs/film (fonctions nommées) | oui (couvert) |  |
| `no_raw_film_bytes_outside_source_helpers_test.go` | helper de no_raw_film_bytes_outside_source | sans objet |  |
| `no_raw_film_bytes_outside_source_test.go` | libs/film + go-api (analysis, sync, cmd) | oui (couvert) |  |
| `no_raw_headshot_category_literal_test.go` | go-api (internal, cmd) + libs/film killsource | oui (couvert) |  |
| `no_raw_isbot_literal_test.go` | go-api seul (internal, cmd) | non | SQL ; aucun LIKE 'bid(%' dans libs/* |
| `no_raw_kill_scope_literal_test.go` | go-api (internal, cmd) + libs/film | oui (couvert) |  |
| `no_raw_mmr_known_test.go` | go-api seul (internal, cmd) | non |  |
| `no_raw_outcome_literal_test.go` | go-api seul (internal) | non | 0 comparaison `Outcome == N` dans libs/* |
| `no_raw_real_start_offset_test.go` | go-api seul (internal, cmd) | non | SQL |
| `no_raw_start_time_literal_test.go` | go-api seul (internal, cmd) | non | SQL |
| `no_readall_on_upload_test.go` | go-api seul (internal, cmd) | non | fichier de formulaire multipart |
| `no_recomputed_film_context_test.go` | go-api (replaybuild, killcollector) + libs/film | oui (couvert) |  |
| `no_replay_firefight_literal_test.go` | go-api seul (module) | non |  |
| `no_repo_root_walk_test.go` | go-api (cmd, internal) + libs/film | oui (couvert) |  |
| `no_request_scoped_halo_provider_test.go` | go-api seul (internal/api) | non |  |
| `no_retention_window_inline_test.go` | go-api seul (module) | non |  |
| `no_rewritten_slot_band_test.go` | libs/film/internal/grammar | oui (couvert) |  |
| `no_runtime_versioned_catalog_write_test.go` | go-api (internal, cmd) + libs/film | oui (couvert) |  |
| `no_second_artifact_sink_test.go` | go-api seul (module) | non |  |
| `no_shared_writer_on_request_path_test.go` | go-api seul (internal/api, internal/service) | non |  |
| `no_slug_comparison_test.go` | go-api seul (internal) | non | 0 `slug == "..."` dans libs/* |
| `no_stale_fallback_target_test.go` | go-api + libs/film (registre des replis) | oui (couvert) |  |
| `no_temporal_title_import_test.go` | go-api seul (internal/analysis/temporal) | non |  |
| `no_title_package_in_analysis_test.go` | go-api seul (internal/analysis, internal/games) | non |  |
| `no_unbounded_film_loop_test.go` | go-api (cmd, internal) + libs/film (mentions) | oui (couvert) |  |
| `no_unregistered_fallback_sites_test.go` | go-api + libs/film (registre des replis) | oui (couvert) |  |
| `no_unregistered_fallback_test.go` | go-api + libs/film | oui (couvert) |  |
| `no_users_json_literal_test.go` | go-api seul (module) | non |  |
| `no_weapon_family_label_literal_test.go` | go-api seul (internal/games/weapons) | non |  |
| `no_weapon_kills_sql_test.go` | go-api seul (module) | non |  |
| `pas_de_lecture_sous_ai_test.go` | espace de travail (balayerGoTexte) | oui (couvert) |  |
| `player_read_cache_invalidation_test.go` | go-api seul (internal) | non |  |
| `pool_no_shared_handle_test.go` | go-api seul (internal/platform/duckdb) | non |  |
| `pose_de_carte_unique_test.go` | go-api + libs/film | oui (couvert) |  |
| `record36_grammaire_test.go` | go-api + libs/film (marcherAPIEtBriqueFilm) | oui (couvert) |  |
| `replay_routes_capability_gate_test.go` | go-api seul (module) | non | routes HTTP |
| `repo_root_readers_allowlist_test.go` | table de données de repo_root_readers (pas un test) | sans objet |  |
| `repo_root_readers_test.go` | go-api seul (internal, cmd) | non | champ Config.RepoRoot ; libs/film ne le lit pas |
| `research_tag_test.go` | espace de travail (balayerGo) | oui (couvert) | tag research |
| `response_writer_unwrap_test.go` | go-api seul (internal/api) | non |  |
| `server_stdout_internal_test.go` | go-api seul (internal) | oui | fmt.Printf sur stdout en production : libs/film/filmproc/runner.go:134 (EmitPeak) et :161 (EmitRaison), protocole voulu de l'enfant |
| `session_options_single_source_test.go` | go-api seul (internal, cmd) | non |  |
| `session_player_slug_test.go` | go-api seul (internal, cmd) | non |  |
| `session_role_read_test.go` | go-api seul (internal, cmd) | non |  |
| `sync_root_freeze_test.go` | go-api seul (internal/sync) | non |  |
| `tactical_background_demo_gate_test.go` | go-api seul (module) | non |  |
| `tactical_pure_test.go` | go-api seul (analysis/tactical, coordination) | non |  |
| `tactical_sequence_gate_test.go` | go-api seul (module) | non |  |
| `title_slug_literal_test.go` | go-api seul (internal/analysis, platform/duckdb) | non |  |
| `titleseams_wired_test.go` | go-api seul (cmd) | non |  |
| `todo_expiry_test.go` | espace de travail | oui (couvert) |  |
| `token_cipher_single_decision_test.go` | go-api seul (internal, cmd) | non |  |
| `un_seul_forcage_du_decodage_test.go` | go-api seul (module) | non | réglage défini dans replaybuild |
| `unarmed_dotation_gate_test.go` | libs/film/replay | oui (couvert) |  |
| `unarmed_family_literal_test.go` | go-api + libs/film (marcherAPIEtBriqueFilm) | oui (couvert) |  |
| `v_match_full_ddl_single_source_test.go` | go-api seul (internal, cmd) | non | DDL |
| `worker_unit_hardening_test.go` | aucun arbre Go balayé (unité systemd) | sans objet |  |

#### Tableau B — hors `archlint` (36 fichiers listés)

| Test | Racine(s) balayée(s) | Sujet dans `libs/*` | Note |
|---|---|---|---|
| `internal/analysis/overtime_guard_test.go` | go-api seul (apps/go-api entier) | non | marge d'overtime ; 0 définition ni calcul à la main dans libs/* |
| `internal/analysis/playlist_label_display_test.go` | go-api seul (internal) | non |  |
| `internal/analysis/team_score_label_guard_test.go` | go-api seul (analysis, service) | non |  |
| `internal/analysis/weapon_range_guard_test.go` | go-api seul (analysis et couches voisines) | non |  |
| `internal/api/public_routes_cost_class_test.go` | go-api seul (internal) | non | routes HTTP |
| `internal/config/env_centralization_test.go` | go-api seul (apps/go-api) | non | noms de drapeaux de déploiement précis ; libs/film lit d'autres noms (filmproc/runner.go) |
| `internal/domain/title/film_short_id_test.go` | go-api seul (internal) | non | 0 troncature de match_id dans libs/* ; trou de racine dans go-api même (voir Découvertes) |
| `internal/games/capability_loader_guard_test.go` | go-api seul (module) | non |  |
| `internal/games/halo_infinite/migrations/idx_rec_hist_canonical_test.go` | go-api seul (migrations) | non | DDL |
| `internal/migration/arbitration_clocks_utc_guard_test.go` | go-api seul (internal) | non | SQL |
| `internal/migration/metadata_art_surface_guard_test.go` | go-api seul (internal) | non | DuckDB |
| `internal/notifications/routes_guard_test.go` | go-api seul (internal) | non |  |
| `internal/observability/level_test.go` | go-api seul (internal) | non | 0 prédicat de journal en ligne dans libs/* |
| `internal/platform/auth/sentinel_test.go` | go-api seul (apps/go-api) | non | jetons ADR 0023 ; la chaîne d'échange de libs/haloapi/xblauth est tenue par haloapi_sans_etat |
| `internal/platform/duckdb/annuaire_ratchet_test.go` | go-api seul (internal) | non | DuckDB |
| `internal/platform/duckdb/media_repo_delete_guard_test.go` | go-api seul (internal) | non | DuckDB |
| `internal/platform/duckdb/no_attach_on_social_test.go` | go-api seul (internal) | non | DuckDB |
| `internal/platform/duckdb/no_mode_name_tr_literal_test.go` | go-api seul (couche duckdb) | non |  |
| `internal/platform/duckdb/no_raw_rating_reads_test.go` | go-api seul (internal) | non | DuckDB |
| `internal/platform/duckdb/player_db_recovery_routing_test.go` | go-api seul (paquet duckdb) | non | DuckDB |
| `internal/platform/duckdb/prestige/bare_db_recovery_routing_test.go` | go-api seul (paquet prestige) | non | DuckDB |
| `internal/platform/netguard/netguard_coverage_test.go` | go-api seul (internal) | oui (couvert ailleurs) | requêtes HTTP sortantes : libs/haloapi les émet par un Doer injecté, tenu par haloapi_doer_garde et haloapi_sans_client_http ; la racine exclut libs/haloapi (en-tête du test) |
| `internal/platform/settings/store_update_guard_test.go` | go-api seul (internal, cmd) | non |  |
| `internal/prestige/no_cross_title_aggregation_test.go` | go-api seul (internal/prestige) | non |  |
| `internal/service/compare_weapons_guard_test.go` | go-api seul (internal/service) | non |  |
| `internal/sync/append_only_state_guard_test.go` | go-api seul (internal) | non | DuckDB append-only |
| `internal/sync/combat_write_guard_test.go` | go-api seul (internal/sync) | non | DuckDB |
| `internal/sync/hook_wired_assertion_test.go` | go-api seul (internal) | non | tag integration |
| `internal/sync/mode_category_single_rule_test.go` | go-api seul (internal/sync) | non |  |
| `internal/sync/no_art_killer_victim_pairs_test.go` | go-api seul (internal, cmd) | non | DuckDB |
| `internal/sync/no_art_patterns_test.go` | go-api seul (internal, cmd) | non | DuckDB |
| `internal/sync/no_art_upsert_migrations_test.go` | go-api seul (internal) | non | DuckDB |
| `internal/sync/shared_write_guard_test.go` | go-api seul (internal) | non | DuckDB |
| `internal/domain/title/capabilities_parity_scan_test.go` | aucun arbre Go balayé (sources web) | sans objet |  |
| `internal/games/capabilities_front_parity_test.go` | aucun arbre Go balayé (sources web) | sans objet |  |
| `cmd/anonymise-gamertags/anonymiser_test.go` | aucun arbre Go balayé (chemins fournis, fixtures) | sans objet |  |

#### Décompte

- Tableau A (206) : 106 « non », 71 « oui (couvert) », 3 « oui (racine partielle) », 3 « oui », 23 « sans objet ».
- Tableau B (36) : 32 « non », 1 « oui (couvert ailleurs) », 3 « sans objet ».
- **Lignes « oui » dont la racine est go-api seul : 3** (`decode_lock_interdit`, `no_discarded_remove`, `server_stdout_internal`). C'est le périmètre du lot 3.2 tel que mesuré, avec :
  - 3 lignes « oui (racine partielle) » : `film_exported_vars` (9 variables exportées modifiables dans `libs/himap`), `film_file_size` (3 fichiers de production > 500 L dans `libs/himap`), `film_function_length` (1 fonction de production > 80 L dans `libs/himap`) ; leur racine inclut `libs/film` et pas les trois autres briques ;
  - 1 ligne hors archlint « oui (couvert ailleurs) » : `internal/platform/netguard/netguard_coverage_test.go` (le sujet existe dans `libs/haloapi`, tenu par `haloapi_doer_garde` et `haloapi_sans_client_http`).
- Mesures de contrôle utilisées : `libs/himap` 9 variables exportées hors `Err*` (`CartesForge`, `TypesVolumesDeMort`, `ModulesDesEquipements`, `StylesFond`, `NiveauHautNavmesh`, ...), `gamefiles` 0, `haloapi` 0 ; fichiers de production > 500 L : `himap` 3, `film` 1, `gamefiles` 0, `haloapi` 0 ; fonctions de production > 80 L (AST) : `himap` 1, `film` 10, `gamefiles` 0, `haloapi` 0.

## Lot 1 — Couplages à l'hôte, sans changement de comportement

- [x] 1.1 Commentaires et chaînes de `libs/film` : `internal/games/halo_infinite/film/...` → chemin relatif au
      module ; commandes `go test ./internal/games/halo_infinite/film/...` → `go test ./...` depuis `libs/film` ;
      un fichier de LevelUp cité en commentaire → son rôle (« l'hôte », « le constructeur de rejeu de l'hôte »).
      Script déterministe versé au scratchpad, liste des fichiers touchés dans le compte rendu. Goldens exclus (D-6).
      — FAIT (2026-11-04). Script `reecriture` (Go, deux passes, annexe `scratchpad/lot1_outils/reecriture/main.go`) :
      il ne touche QUE les lignes de commentaire `//` des `.go` de `libs/film`, hors `testdata/` (D-6) et hors
      `revision/` (le chemin historique y est la clé du cadre de hachage : `cadre.go:37`, `couches.go:17`). Règles :
      `./internal/games/halo_infinite/film/` → `./` ; `[apps/go-api/]internal/games/halo_infinite/film/` et
      `games/halo_infinite/film/` → chemin relatif au module ; `games/halo_infinite/filmprofile` → `internal/profile` ;
      `depuis apps/go-api` et `cd apps/go-api` → `depuis libs/film` / `cd libs/film`. Résultat : 561 fichiers, 796
      lignes par le script (liste : `temoins_film/lot1/1.1_fichiers_touches*.tsv`), plus 11 fichiers à la main
      (`temoins_film/lot1/lot1_tous_fichiers_modifies.txt` = les 572 fichiers de `libs/film` modifiés par le lot,
      tous items confondus) : les 5 `rev_test.go` et `types/shapes_test.go` (commande de régénération passée dans un
      message d'échec, jamais écrite dans un golden : vérifié), `filmshell/catalogue_test.go`,
      `highlight_events_jgtm_test.go`, `weaponv3/pi_resolver_test.go`, `replay/layers_test.go`,
      `usage_summary_families_guard_test.go` (un fichier de LevelUp cité → son rôle). Commandes mixtes (un `go run`
      de l'hôte suivi d'un `go test` de la brique) : précisées à la main (`damagetag.go`, `e191_origine_mesure…`,
      `vehicle_end_acceptation_test.go`). Dix fichiers relus à la main (tirés tous les 55) : sens conservé. Contrôle :
      les 53 chemins distincts réécrits existent sous `libs/film` sauf trois références DÉJÀ mortes avant le lot
      (`filmdec`, `internal/facts/killsource/world.go`, `replay/i56_capacity_episodes_research_test.go`, voir D-L1-4).
      Résidu voulu : `revision/` (3 fichiers de production, 4 de test), 8 littéraux de test que des tests ÉCRIVENT dans
      un golden ou un `PROVENANCE.txt` (`golden_render_test.go:28`, `minibobine_test.go:156`,
      `frame_closure_ratchet_test.go:204`, `keyframe_closure_cuisson_ratchet_test.go:81`,
      `keyframe_closure_ratchet_test.go:182`, `marche_fermeture_test.go:36`, `minifilm_builds_test.go:261`,
      `minifilm_test.go:148` : les changer = régénérer, D-6, voir D-L1-3), et les noms de paquets de l'hôte cités sans
      préfixe `apps/go-api` (voir D-L1-1). Preuve : T2, 0 golden changé et cinq `Rev` inchangées.
- [x] 1.2 Les 7 chemins absolus (`C:\Users`, `Projects\LevelUp`) : résolus par `racineDepot(t)` ou retirés.
      — FAIT. `objectives/extract_test.go:18` (prose seule, le chemin mort retiré) ; `grammar/vehicules_v13_deadstate_test.go:565`
      (le repli sur un chemin de poste est retiré : sans `V13_FILM_ROOT` la mesure se saute en le disant ; le fichier
      reste à 580 lignes, plafond du ratchet de taille) ; `vehicules_chassis_inventaire_test.go:32`, `zone_colline_nom_temoin_test.go:21`,
      `zone_etat_initial_temoin_test.go:19`, `zone_proprietaire_nom_temoin_test.go:17` (exemples d'usage en `<depot>/…`) ;
      `research/cmd_rdata_weapon_scan/main.go:44` (constante remplacée par `RDATA_SCAN_CACHE`, arrêt explicite si absente,
      documentée dans l'en-tête). Preuve : `grep -rnE '[A-Za-z]:.{1,3}Users|Projects.{1,3}LevelUp' libs/film --include=*.go` vide.
- [!] 1.3 Registre des replis : `Site.Fichier` relatif au module, `Appelants` nommés par rôle (D2-L3-f). Avant :
      lister les lecteurs du registre (route admin, test) et prouver que la sortie servie ne change pas, ou écrire
      le changement.
      — NON TRAITÉ, justification écrite. Lecteurs (grep `decfilm.Table()` / `decfilm.Lire(` / `.Sites` / `.Fichier` /
      `.Appelants`) : AUCUNE route admin ni document publié (le document de rejeu ne porte que `Name`/`Hits`,
      `fallbacks_publication.go:55-61`) ; mais six GARDES de `apps/go-api` OUVRENT `racine/Site.Fichier` et comparent
      l'ancre : `archlint/no_unregistered_fallback_test.go:190-206` et `:238-243`,
      `no_unregistered_fallback_sites_test.go:106` et `:183-188`, `fallback_versement_test.go:134-140` ;
      `fallback_versement_test.go:211-218` compare `Appelants` à des chemins relatifs au dépôt (`appelantAdmis(rel, …)`) ;
      `service/replay_vehicle_scenery_{aloft,rule}_test.go:143,193` testent `HasSuffix(Sites[0].Fichier,
      "service/replay_vehicle_scenery_rule.go")` ; `cmd/replay-verite/main.go:137` ne lit que `CompteurBranche`.
      `Site.Fichier` est donc un chemin relatif à la racine du dépôt HÔTE (`libs/film/…` pour les 9 sites de la brique,
      déjà dans la forme que les gardes ouvrent) : le passer « relatif au module » ou le nommer « par rôle » casse six
      gardes, et un rôle ne se résout pas en fichier. 32 lignes de production nomment des fichiers de l'hôte
      (`pkgKillcollector`, `pkgReplaybuild`, `siteDecorM7`, `diag_weapons_v3`, `derivations_backlog.go`). La séparation propre
      (les sites de l'hôte déclarés par l'hôte, le registre de la brique ne gardant que les siens) est un changement de
      contrat de la façade : D-L1-2, à décider avec le lot 2. Seule la partie sans effet est faite : les commentaires
      d'en-tête des `registre_*.go` (1.1). `repli.go:174` décrit le contrat réel et reste.
- [x] 1.4 Les 9 instruments `research` du backlog (`filepath.Join(depot, "data", ...)`, 8 sous `libs/film/replay`,
      1 sous `apps/go-api/internal/replaybuild`) : chemin reçu par variable ou drapeau de l'instrument côté brique,
      `PathResolver` côté go-api.
      — FAIT. Les 9 fichiers sont tous sous `//go:build research` (instruments, jamais production). Côté brique :
      `replay/mesure_chemins_research_test.go` (neuf) pose trois variables PROPRES aux mesures, sans préfixe `LEVELUP_` :
      `L1_BORNES` (le fichier `map_quant_bounds.json`), `L1_FAITS` (dossier des `<id>.filmfacts.bin`), `L1_ARTEFACTS`
      (dossier des `<id>.json` de rejeu) ; une mesure qui en a besoin et ne la trouve pas s'arrête en la nommant
      (`cheminMesureRequis`) ; l'artefact facultatif de `score_koth_emissions` garde son caractère facultatif
      (`lireArtefactMesure`). `L1_DEPOT` disparaît des huit en-têtes, qui documentent les nouvelles variables ; les
      tests restent sautés par défaut. Côté go-api : `drapeau_bases_film_parc_research_test.go` passe par
      `title.NewPathResolver(L1_DEPOT).ReplayArtifactPath` et `.FilmFactsPath`. Preuve : `go vet -tags research ./...`
      (libs/film) et `go vet -tags research ./internal/replaybuild/` (go-api) verts ;
      `grep -rnE 'Join\(depot, "data"' libs/film apps/go-api/internal/replaybuild` vide. Restent 36 fichiers de
      `libs/film` avec un autre `filepath.Join(…, "data", …)` : hors des 9 du backlog, D-L1-5.
- [x] 1.5 Haiku : `docs/CONFIGURATION.md:150` (`halo.` → `tokencache.InvalidateCachedPlayerTokens`, PUB-4-c).
      — FAIT. Nom vérifié : `func InvalidateCachedPlayerTokens` à `apps/go-api/internal/platform/auth/tokencache/player_token_cache.go:144`.
      Corrigé en `docs/CONFIGURATION.md:150` et dans la copie FR `docs/FR/CONFIGURATION.md:151`. Les ADR 0023 (`:130`) sont
      de l'historique EN et ne sont pas réécrites.
- [x] 1.6 MESURE (D-7, décision du 2026-11-04 : remplacer les identités réelles), sans rien modifier : pour chaque
      fixture de film de `libs/film` (8 minifilms de `replay/testdata`, `testdata/`, `internal/grammar/testdata`,
      `internal/source/testdata`, `internal/profile/testdata`) : (a) nombre de xuids et de gamertags distincts
      lus par le décodeur (par `decfilm`, pas par grep) ; (b) les champs de la grammaire qui les portent (table
      d'identité, BOT_METADATA, journal des morts, chat ?) et leur forme (entier 64 bits, UTF-16, longueur
      fixe ou variable) : une réécriture à longueur constante est-elle possible sans toucher aucune autre
      structure ? (c) les fichiers de résultat attendu (goldens, `.json`, `.tsv`, `.bin.gz` dérivés) qui portent
      ces identités, par nombre ; (d) les goldens de révision qui changeraient (si les octets des films entrent
      dans une empreinte). Tableau versé au plan sous « Mesure des identités (1.6) » + une estimation en heures
      des deux options (réécriture à longueur constante vs nouveaux films joués par des comptes du user).
      — FAIT, tableau ci-dessous. Aucune fixture modifiée (`git status` : aucun fichier sous `testdata/` touché).
- [x] 1.7 Gate : `go vet ./...` et `go vet -tags research ./...` dans `libs/film` ; T1, T2 (empreintes insensibles
      aux commentaires : 0 golden changé), T5 ;
      `grep -rlE 'internal/games/halo_infinite/film|C:\\\\Users' libs/film --include=*.go | grep -v _test.go` vide.
      Commit « lot 1 ».
      — FAIT. `go vet ./...` et `go vet -tags research ./...` (libs/film) verts, `go build ./...` (apps/go-api) vert,
      `gofmt -l` vide. T1 : 21 556 lignes (paquet, test, verdict), IDENTIQUES au lot 0 (MD5 de la liste triée
      `f1f0ed84c7592b8a14f75a6b067a4aed`, 21 176 pass, 0 fail, 380 skip, 0 disparition, 0 ajout) ; 349 paquets de go-api.
      T2 : cinq `Rev` inchangées, SHA-256 des 34 `*.golden` identiques (`diff` vide), `go test ./revision/…` et
      `./internal/… -run 'Rev|Empreinte|Perimetre'` verts. T5 : `levelup/go-api` = 0 dans les quatre briques (274, 149,
      233, 154 paquets, listes identiques au lot 0). Lint : `golangci-lint run --config ../../apps/go-api/.golangci.yml ./...`
      depuis `libs/film` = 0 issue. Grep final non-test : vide hors `revision/cadre.go`, `couches.go` et `perimetre.go`
      (clé et prose du cadre de hachage historique, exclus par décision de l'item 1.1). Premier passage de T1 : le
      ratchet `TestTailleDesFichiersDuFilmNeCroitPas` a rougi (583 lignes pour un plafond de 580 sur
      `vehicules_v13_deadstate_test.go`), puis `TestPlafondsDeTailleNeSontPasPerimes` (576 < 580) : le fichier a été
      ramené à 580 lignes exactement ; `archlint` rejoué seul (`-timeout 20m`, 487 tests verts, 200 s) et `libs/film`
      rejoué entier (119 s). T3 et T4 ne sont pas rejoués : le lot ne change ni le code de production ni les
      fixtures (comparaison T2 à l'appui).

### Mesure des identités (1.6)

Mesure du 2026-11-04 (HEAD `5d0a239f89` plus le lot 1), aucune fixture modifiée. Outil jetable, NON commité (annexe
`scratchpad/lot1_outils/zz_mesure_identites_jetable_test.go.txt`) : un test de `internal/grammar` ouvre chaque dossier de
film par `source.LoadDir`, lit la table des joueurs (`ScanFilmPlayerTable`) et le fil des morts (`ScanDeaths`), puis
cherche chaque xuid (8 octets, ordre LE et ordre MSB) et chaque gamertag (UTF-16 LE et BE) à CHACUN des huit alignements de
bit, chunk par chunk. Sorties dans `temoins_film/lot1/1.6_*.txt` (mesure brute, faisabilité, xuids par fichier, binaires
dérivés).

**(a) Identités lues par le décodeur** (table des joueurs + fil des morts ; « occ. » = occurrences de xuid trouvées dans
les octets des chunks : LE = paquets d'identité, évènements, clé de jointure ; MSB = xuid de la table du registre).

| Fixture | Chunks | xuids | gamertags | occ. LE | occ. MSB |
|---|---|---|---|---|---|
| `replay/testdata/minifilm_000d5950` (sans registre) | 3 | 8 | 8 | 262 | 0 |
| `minifilm_111fa685` (build HI_1_10_0) | 3 | 25 | 25 | 496 | 24 |
| `minifilm_11de8353` (HI_1_9_0) | 3 | 27 | 27 | 417 | 24 |
| `minifilm_60ae07c4` (HI_1_8_0) | 3 | 8 | 8 | 498 | 8 |
| `minifilm_a521164d` (HI_1_4_1) | 3 | 27 | 27 | 297 | 24 |
| `minifilm_bcb6d393` (HI_1_12_0) | 3 | 11 | 11 | 144 | 8 |
| `minifilm_e5adf7b2` (HI_1_11_0) | 3 | 27 | 27 | 514 | 23 |
| `minifilm_fb1a1a72` (HI_1_13_0) | 3 | 8 | 8 | 396 | 8 |
| `internal/facts/killsource/testdata/minibobine_000d5950` | 7 | 8 | 8 | 334 | 8 |
| `…/minibobine_e5adf7b2` | 3 | 27 | 27 | 652 | 23 |
| `testdata/corpus_objectifs/minibobine_objectifs/{koth_0a247154,strongholds_7344d24f}` | 8 + 8 | 0 | 0 | 0 | 0 |
| `internal/grammar/testdata/v41_chunk_he.bin` (chunk des temps forts, zlib) | 1 | 3 (parmi les 289 connus) | 242 occ. | 113 | 0 |
| `internal/grammar/signaux/testdata/pied_bloc_53ce4390.bin` | 1 | 1 | 1 occ. | 1 | 0 |

Union : **133 xuids pour 133 gamertags** (un gamertag par xuid) dans les octets de film. **130 gamertags sur 133 sont déjà
des pseudonymes** de même longueur (`Joueur000…`, `J000…`, outil `apps/go-api/cmd/anonymise-gamertags`, v8.0.2) ; trois
restent réels, les comptes consentants ou du propriétaire : `JGtm`, `Chocoboflor`, `Madina97294`. Aucune occurrence dans les
24 `entites_`/`equipement_`/`inputs_*.bin.gz` (décompressés), `vue_a_bcb6d393_1_204.bin` et les 15 graines de fuzz.

**(b) Où et sous quelle forme la grammaire porte l'identité**

| Où | Champ | Forme | Réécriture à longueur constante |
|---|---|---|---|
| registre `chunk_00`, table des joueurs (`player_table*.go`) | XUID du siège | entier 64 bits lu MSB d'abord, à un décalage de bit quelconque (85 bits d'en-tête avant) ; le repérage des sièges exige `0x0009000000000000 <= xuid < 0x000A000000000000` (`player_table.go:129-130`, `balayerCandidats`) | oui, À CONDITION que le faux xuid reste dans cette plage (2 533 274 790 395 904 à 2 814 749 767 106 559) ; hors plage la table n'est plus trouvée (« aucune table de 32 slots », constaté sur un premier essai) |
| idem | clé de jointure `sub+0xcb8` | le même xuid, 8 octets dans l'ordre du flux (1 par siège, retrouvé comme motif LE) | oui |
| idem | gamertag | unités UTF-16 de 16 bits MSB d'abord, 16 unités au plus, terminateur nul sauf chaîne pleine ; le nombre d'unités entre dans la longueur prédite de l'enregistrement (`longueurPredite`) ; 2 copies par siège mesurées (une lue par `lireGamertag`, l'autre non attribuée à un champ nommé) | oui à nombre d'unités égal (déjà fait par l'outil de la v8.0.2) |
| chunks de réplication, paquets d'identité (`player_index.go`, `weaponv3.ResolveXuidToPI`) et entrées de roster de type 8 (`roster_type8.go`) | xuid, gamertag | xuid : 8 octets LE à un décalage de bit quelconque, précédés de 5 bits d'index de joueur (type 8 : dans l'ordre d'octets du flux) ; gamertag : UTF-16 | oui |
| chunk des temps forts / pied (`highlight_events.go`) | xuid et gamertag de chaque évènement | xuid : 8 octets LE à décalage quelconque, filtré sur `[2e15, 3e15]` (`minXUID`/`maxXUID`) ; bloc de 60 octets dont gamertag UTF-16LE sur 32 octets FIXES (16 unités, bourrage nul) ; 93 à 491 évènements par fixture | oui, largeur fixe |
| `BOT_METADATA` (type 12) | noms de bots | pas une identité de personne | sans objet |
| chat | aucun lecteur ni motif dans la grammaire | — | — |

Essai en mémoire, rien d'écrit : tous les motifs de xuid et de gamertag remplacés à longueur égale dans les chunks
décompressés de 10 dossiers (xuids factices dans la plage), puis relecture. Résultat : xuids, temps, index de joueur et
nombre de sièges **identiques au mapping près sur 10 dossiers sur 10** ; la table des joueurs et le fil des morts sont
identiques sur 8 sur 10 pour les gamertags, les deux écarts étant des collisions de préfixe entre PSEUDONYMES
(`Joueur000101` remplacé à l'intérieur de `Joueur0001010…`), que la table de correspondance de l'outil existant évite.
La longueur constante est donc possible sans toucher une autre structure ; elle demande de recompresser (zlib au niveau
déclaré par l'en-tête de chaque chunk, déjà géré par l'outil de la v8.0.2).

**(c) Fichiers de résultat attendu qui portent des xuids** (grep `25\d{14}` sur tout `libs/film`, plus les octets)

| Fichier | Nombre | xuids | Remarque |
|---|---|---|---|
| `replay/testdata/equivalence/<id>.facts.json` | 20 | 311 occ. (8 à 29 par fichier) | entrées du harnais `replay-equiv`, extraites du parc réel : 156 des xuids du texte viennent de films SANS fixture binaire |
| `replay/testdata/assembly_<id>.golden` | 8 | 143 occ. | goldens à régénérer (`-update`) |
| `internal/grammar/signaux/testdata/pied_bloc_53ce4390.PROVENANCE.txt` | 1 | 2 | prose |
| `_test.go` de `libs/film` | 32 | 188 valeurs distinctes (343 occ.) | littéraux de test et commentaires |
| `.go` de production (prose de commentaire) | 11 | 14 | `objectives/{helpers,rev,slotidentity_*}.go`, `grammar/rev_chronique.go`, `replay/{document_chronicle,flag_carries_home,flag_objects,identity_registry_pont,vehicle_tracks}.go` |
| `bots_<id>.json` (8), `entites_`/`equipement_`/`inputs_*.bin.gz` (24), `fuzz` (15), goldens killsource (4) | — | 0 | les goldens killsource portent des gamertags (déjà pseudonymes), pas de xuid |
| `equivalence/*.tsv` (21) | 21 | 0 en clair | digests SHA-256 de calques : bougent si le calque cite un xuid, non mesuré |

Total : 420 xuids distincts dans les fichiers de `libs/film` (texte et code), dont 133 dans les octets de film.

**(d) Révisions.** Les octets des films n'entrent dans AUCUNE empreinte : `revision/empreinte.go:127` et
`revision/perimetre.go:195` écartent tout ce qui vit sous `testdata/` et les `_test.go`, et les périmètres hachent des
SOURCES de production (jetons, fichiers embarqués de `damagetag`/`killicon`). Les cinq `Rev` et les 10 goldens de
révision et de périmètre ne bougent donc pas (chiffre confirmé au lot 1 par T2). Ce qui bougerait : les 8
`assembly_*.golden`, `equivalence/minifilm.tsv` et l'empreinte des temps forts v41 (déjà bougée par la v8.0.2), plus les
20 couples `facts.json`/`tsv` du corpus. Point de fond : ces 20 couples servent à `replay-equiv` (T3), qui compare des
films RÉELS du parc à leurs références ; des références à xuids factices ne se comparent plus au parc sans une table
de correspondance inverse dans le harnais.

**Estimation.**

| Option | Contenu | Heures d'agent |
|---|---|---|
| A. Réécriture à longueur constante (étend `cmd/anonymise-gamertags`) | formes xuid binaires LE et MSB aux 8 décalages et décimale à 16 chiffres : 2 h ; table de correspondance hors dépôt et application aux 11 dossiers, `v41`, `pied_bloc`, 20 `facts.json`, 8 `assembly`, `PROVENANCE` : 1,5 h ; preuve par les balayages du décodeur : 2 h ; régénération des goldens et digests dérivés avec chronique : 1,5 h ; littéraux des 32 tests (188 valeurs) et prose des 11 fichiers de production : 2 h ; table inverse dans `replay-equiv` pour que les 20 références restent comparables au parc : 2 à 3 h ; gates T1, T3, T4 : 1 h d'attente | 12 à 14 h (1,5 à 2 jours), aucune partie à jouer, révisions inchangées, aucune recuisson |
| B. Nouveaux films joués par des comptes de l'utilisateur | reconstruire les 10 dossiers, `v41`, `pied_bloc` et les 20 références du corpus ; régénérer tous les goldens qui en dépendent | 25 à 40 h plus 3 à 6 h de parties ; perd la couverture de 7 builds anciens (HI_1_4_1 à HI_1_13_0 : seul le build courant se joue) et des rosters de 8 à 27 identités (un compte remplit un siège) |

Recommandation du lot : A. B ne tient que si la couverture par build est abandonnée.

## Lot 2 — Réorganisation protocole / exploitation et API avant le tag (VALIDÉ le 2026-11-04 ; Opus)

Règle (à écrire dans l'ADR au 2.8) : la brique décode un film et publie ses faits et son document de rejeu ;
elle ne décide ni QUAND on décode, ni OÙ on stocke, ni avec QUELLE PRIORITÉ. Ce qui décide part dans LevelUp.
Aucune des couches hachées n'est touchée : révisions INCHANGÉES, aucune recuisson. Les 96 fichiers de test du
décodeur qui importent `filmcache` / `filmproc` suivent les nouveaux noms (réécriture d'import, pas de logique).

- [x] 2.1 `filmproc` coupé en deux : la sentinelle mémoire (`Arm`, `MeasureLimitGiB`, `Footprint`, `EmitPeak`,
      plafonds) reste dans la brique sous `libs/film/memguard` (nom à confirmer sur pièces : le paquet actuel
      `memguard.go`) ; le lanceur parent/enfant (`NewRunner`, codes de sortie, `Issue*`, `AcquireSolo*`,
      priorité, console cachée, `Roots` et les quatre `Env*` `LEVELUP_*`) part dans `apps/go-api/internal/filmrun`
      (nom libre, un seul paquet). Test : environnement et arguments de l'enfant identiques octet pour octet
      avant/après (témoin capturé AVANT le déplacement). `LEVELUP_*` ne reste dans la brique que dans
      `revision/porte.go`.
      — FAIT (2026-11-04). `libs/film/memguard` : `memguard.go`, `doc.go` (contrat réécrit : la sentinelle ne
      décide ni l'arrêt, ni quand, ni la priorité), `memguard_bytes_test.go`, `memguard_message_test.go` (`git mv`)
      et `memguard_test.go` (les 5 tests de la sentinelle extraits de `filmproc_test.go`). `EmitPeak` est une ligne
      du protocole parent/enfant : il part avec le lanceur. `apps/go-api/internal/filmrun` (`git mv` de 18 fichiers :
      `runner.go`, `solo*.go`, `alive_*`, `priority_*`, `selfpriority_*` et leurs tests, `filmproc_test.go` ->
      `filmrun_test.go` ; `doc.go` neuf). Le lanceur passe par `platform/childproc.Command` : les copies
      `hideconsole_win.go` / `hideconsole_other.go` et leur garde de source `sansfenetre_source_test.go` sont
      supprimées (leur sujet est tenu par `archlint/childproc_single_spawn_test.go`, dont l'allowlist se vide),
      `TestSansFenetreSousWindows` réécrit sur `Runner.command`. Le verrou noyau de `solo.go` vient de
      `platform/filelock`, COPIE JUMELLE neuve de `libs/film/internal/filelock` (tenue par
      `copies_identiques_test.go`). Les tests et instruments de la brique qui prennent le verrou de décodage
      (`replay/filmfacts_feuilles_films_test.go`, quatre `*_research_test.go`) passent par
      `libs/film/internal/decodelock` (même fichier, même description du détenteur, sans journal), parité du
      protocole tenue par `archlint/film_verrou_decodage_parite_test.go` (mutation jouée : `film_decode2.lock` et
      `json:"pid2"` -> rouge, restauré -> vert, `temoins_film/lot2/2.1_mutation_parite_*.txt`). Témoin : crochet
      `Runner.command` + `runner_commande_test.go`, capturé dans `filmproc` AVANT tout `git mv` puis dans `filmrun` :
      341 lignes, IDENTIQUES à deux variables du harnais près (`FILMRUN_TEMOIN_COMMANDE`, le fichier du témoin, et
      `PWD`, posé par `go test` au dossier du paquet) ; normalisées, MD5 `b9e8aae550c19f1a2a2f627024760609` des deux
      côtés (`temoins_film/lot2/2.1_commande_*`). 71 fichiers de `libs/film` et 37 de `apps/go-api` réécrits
      (imports, puis commentaires qui nommaient `filmproc`). Garde-rails suivis : `no_accidental_goos_suffix`,
      `film_layers_deps`, `film_slog_contexte` (exclusion renommée `memguard`, son retrait est l'item 2.5),
      `no_unbounded_film_loop`, `decode_lock_interdit`, `no_cuisson_depuis_tactique`, `no_error_text_classification`,
      `film_tri_total` (racine `internal/filmrun` ajoutée : le lanceur était balayé sous `libs/film`), voie CI Windows
      (`ci.yml`, `docs/COMMANDS.md` EN et FR). `grep -rn LEVELUP_ libs/film --include=*.go | grep -v _test.go` :
      `revision/porte.go` seul.
- [x] 2.2 `filmcache` réduit à la lecture d'un film depuis un dossier (manifeste + chunks, formes brut/zlib/zstd :
      `LoadFilm`, `LoadFilmDir`, `Open`, `Source`, chemins) ; déplacement d'archive, recompression, écriture à la
      réception (`deplacement.go`, `recompress.go`, `write.go` et leurs tests) partent dans
      `apps/go-api/internal/sync/filmfetch` ou un paquet frère (relever les appelants AVANT : `sync/filmfetch`,
      `cmd/levelup film recompress`, `film archive-move`, desktop 14.3/14.5). Garde-rail de disposition
      `filmcache` (desktop 14.4) reste vert.
      — FAIT (2026-11-04). Appelants relevés AVANT (`temoins_film/lot2/2.2_appelants_avant.txt` : symboles par
      fréquence, 26 fichiers appelants de l'écriture, 37 importeurs). Paquet de l'hôte `apps/go-api/internal/filmarchive`
      (un paquet, l'archive côté écriture ; `filmfetch` ne porte que le téléchargement) : `git mv` de `write.go`,
      `deplacement.go`, `recompress.go`, `write_test.go`, `write_finalise_test.go`, `write_robustesse_test.go`,
      `reprise_test.go`, `stockage_test.go` ; le témoin `filmcache/testdata/manifeste_partiel_34.json` reste dans la brique
      (`replay/fil_des_morts_faits_test.go` le lit aussi), l'hôte le relit par `testutil.CheminBriqueFilm` ;
      `Supprimer` sort d'`archive.go` (qui ne garde que `MesurerOccupation`, une lecture) vers `retrait.go` ; `doc.go`
      neuf. L'écriture garde la disposition et le format de la brique : chemins par `filmcache.ManifestPath` /
      `ChunkDir` / `CheminDuChunk` / `ChunksRoot` / `ManifestsRoot`, forme zstd par les fonctions exportées de
      `filmcache/stockage.go` ; la règle « zstd-3 quand c'est plus court » (`octetsAEcrire`) est une décision de
      stockage, elle part avec l'écrivain. Restent en lecture dans la brique : `filmcache.go`, `chunk.go`,
      `stockage.go`, `archive.go`, `filmcache_test.go`, `chunk_test.go` (les deux tests de lecture qui se préparaient
      par `Write` posent désormais leurs fichiers eux-mêmes, `poserFilm` ; `TestChunkZstdTronqueEstDetecte` revient
      de `stockage_test.go` dans `chunk_test.go`, même paquet). 26 fichiers de go-api réécrits (`filmcache.Write` ->
      `filmarchive.Write`, etc.). `no_auto_film_recompress_test.go` suit le paquet (`pkg.Name == "filmarchive"`) ;
      `no_hardcoded_film_cache_dirs_test.go` (disposition) vert sans changement. `atomicfile` : l'écrivain prend
      l'original `platform/atomicfile`.
      — VÉRIFIÉ SUR PIÈCES à la reprise (2026-11-08) : `libs/film/filmcache` ne garde que `archive.go`, `chunk.go`,
      `filmcache.go`, `stockage.go` ; `apps/go-api/internal/filmarchive` porte `write.go`, `deplacement.go`, `recompress.go`,
      `retrait.go` ; `go test ./filmcache/ ./internal/digest/` (libs/film) et `./internal/filmarchive/ ./internal/analysis/digest/
      ./internal/filmrun/` (go-api) verts ; `TestRecompressionEtDeplacementDeLArchiveSontManuels`, `TestNoHardcodedFilmCacheDirs` verts.
- [x] 2.3 `digest` et les helpers de test partagés passent sous `libs/film/internal/` (`internal/digest`,
      `internal/depottest` pour les six copies de `racineDepot(t)` + homonyme d'`objectives` en `Skip`, garde-rail
      règle 6) ; `cmd/replay-equiv` (go-api) importe `digest` : relever et décider sur pièces (copie dans go-api
      avec test d'identité, comme `atomicfile`, ou le harnais entier part dans go-api).
      — FAIT (2026-11-04). `git mv libs/film/digest libs/film/internal/digest`. Importeurs relevés : la brique
      (`replay/equivalence_minifilm_test.go`, qui fige `equivalence/minifilm.tsv`) ET le harnais de l'hôte
      (`cmd/replay-equiv/{child,parent,parent_test}.go`). DÉCIDÉ : copie jumelle dans go-api,
      `apps/go-api/internal/analysis/digest` (son emplacement d'avant D2), tenue identique par
      `copies_identiques_test.go`. Pourquoi pas « le harnais entier dans go-api » : il y est déjà ; c'est la brique
      elle-même qui hache sa mini-bobine avec la même grammaire, et les deux rendus doivent rester les mêmes octets.
      `libs/film/internal/depottest` : `Racine(t)` remplace les six copies de `racineDepot` (quatre `depot_test.go`
      supprimés, deux réduits à leurs chemins) et l'homonyme d'`objectives` (qui cherchait `apps/web` et échouait par
      `t.Fatalf` : il saute désormais hors dépôt hôte, comme les autres) ; 31 fichiers d'appel réécrits. Garde-rail
      règle 6 : `archlint/film_racine_depot_unique_test.go` (aucun `"go.work"` ni `func racineDepot(` hors
      `internal/depottest/depottest.go`, tests et `research` compris ; mutation jouée : une remontée ajoutée dans un
      `_test.go` de `replay` -> rouge, retirée -> vert, `temoins_film/lot2/2.3_mutation_racine_*.txt`).
      `film_layers_deps` : `internal/digest` et `internal/depottest` hors couche.
      — VÉRIFIÉ SUR PIÈCES à la reprise (2026-11-08) : `libs/film/internal/digest`, `libs/film/internal/depottest`,
      `apps/go-api/internal/analysis/digest` existent ; `TestLesCopiesJumellesSontIdentiques`,
      `TestLaBriqueRemonteVersLeDepotParUneSeuleAide` et `TestCouchesDuDecodeur*` verts ; `libs/film/filmproc` et `libs/film/digest` absents.
- [x] 2.3b Registre des replis (D-L1-2, item 1.3 `[!]`) : le registre de la brique ne nomme QUE des sites de la
      brique, `Site.Fichier` relatif au module `libs/film` ; les sites et `Appelants` de LevelUp (32 lignes de
      production au relevé du lot 1) passent dans un registre de l'hôte (`apps/go-api/internal/...`, un paquet) qui
      embarque celui de la brique ; les six gardes de go-api (`no_unregistered_fallback_test.go`,
      `no_unregistered_fallback_sites_test.go`, `fallback_versement_test.go`, `service/replay_vehicle_scenery_*_test.go`)
      ouvrent chaque site avec la racine de SON registre. Aucun nom de repli ni compteur ne change (T4 identique).
      — FAIT (2026-11-04), dans la lettre de l'item : ce sont les SITES et les APPELANTS de l'hôte qui partent, pas
      les entrées. Raison sur pièces : les noms des replis de l'hôte se comptent sur le `Compteur` de la brique (que
      l'hôte crée par `decfilm.NouveauCompteur` et que la brique remplit), lequel signale en `ERROR` tout nom absent
      de SON registre (`compteur.go`, `DiagRepliHorsRegistre`) ; déplacer les entrées changerait ce contrat et la
      façade (12 constantes `decfilm.Nom*`), ce que l'item exclut (« aucun nom ni compteur ne change »). Fait :
      `Site.Fichier` de la brique relatif au module (`replay/...`, `internal/...`) ; 33 sites de l'hôte (18 entrées :
      15 sans site dans la brique, 3 mixtes) et les appelants de l'outil `cmd/diag_weapons_v3` retirés de la brique
      par un outil AST jetable (`scratchpad/lot2_outils/sitesHote`) ; champ neuf `Repli.CompleteeParLHote` (marque
      les 19 entrées que l'hôte complète ; `VerifierRegistre` admet alors une entrée sans site de la brique ou un
      outil sans appelant de la brique). Registre de l'hôte : `apps/go-api/internal/filmreplis` (`sites.go` : la
      table des sites et des appelants, relatifs à `apps/go-api` ; `filmreplis.go` : `Sites`, `Appelants`, qui rendent
      chaque chemin depuis la racine du dépôt en préfixant la racine de SON registre, et `Verifier`). Les six gardes
      lisent `filmreplis.Sites` / `filmreplis.Appelants` ; `TestRegistreDesReplisEstValide` rejoue aussi
      `filmreplis.Verifier`. Témoin AVANT/APRÈS : description complète du registre (125 entrées, tous champs, 292 sites
      triés, appelants) par `decfilm.Table()` avant, par la vue fusionnée après : IDENTIQUES (`diff` vide,
      `temoins_film/lot2/2.3b_registre_{avant,apres}.txt`). Mutation : retrait d'une entrée de `sites.go` -> rouge
      (`filmreplis` et `service`), restauré -> vert. `internal/facts/fallback` n'entre dans aucun périmètre haché
      (grep des cinq `*_perimetre.golden`) : T2 inchangé.
- [x] 2.4 `filmshell` : les 15 variables deviennent non exportées derrière des accesseurs en lecture (copie ou vue
      non modifiable) ; lecteurs migrés (`games/weapons`, `platform/duckdb`, `sync/killcollector`, décodeur) ;
      `catalogueArmesHorsRatchet` retiré dans le même commit. Révisions INCHANGÉES ; si un golden de périmètre
      bouge, régénération UNE fois par la porte à deux verrous, une entrée de chronique par couche.
      — NON TRAITÉ, ARRÊT DU LOT SUR CET ITEM (2026-11-04), décision du pilote attendue. Preuve : `filmshell` est un
      `paquet` du périmètre de `grammar` « haché par ses octets (jetons) » (`internal/grammar/testdata/grammar_perimetre.golden:17`),
      et la grammaire lit six des quinze variables par 19 références (`WeaponIDToName` ×10, `CommonWeaponSuffix` ×3,
      `WeaponBytesMap` ×2, `WeaponFusionMap` ×2, `WeaponIDs`, `AllFormulaASuffixes`). Mesure jetable, annulée aussitôt
      (`git diff -- libs/film/filmshell` vide après) : une fonction non exportée de deux lignes ajoutée à
      `filmshell/catalogue.go` fait rougir `TestGrammarRevSuitLaGrammaire` (« REGENERER le golden ... »,
      `temoins_film/lot2/2.4_mesure_empreinte.txt`). L'item ne peut donc pas se faire sans réécrire la ligne
      d'empreinte de `internal/grammar/testdata/grammar_rev.golden`, un golden de RÉVISION ; la liste des paquets du
      périmètre (`grammar_perimetre.golden`), elle, ne bougerait pas (`filmshell` reste un paquet). L'invariant du lot
      n'admet qu'un golden de PÉRIMÈTRE régénéré au 2.4. À trancher : régénérer l'empreinte de `grammar` à révision
      constante (porte `LEVELUP_UPDATE_GRAMMAR_REV`, entrée de chronique, comme D2 2.12 l'a fait pour les goldens de
      révision ET de périmètre), ou garder les variables exportées.
      — FAIT (2026-11-04) après la décision du pilote du même jour (régénérer l'empreinte de `grammar` à
      révision constante). Témoin des VALEURS capturé AVANT (`temoins_film/lot2/2.4_tables_avant.txt`, 311 lignes
      triées des quinze tables) puis relu APRÈS par les accesseurs : identique (259 lignes ; les 52 autres sont les
      trois tables sans aucun lecteur, retirées : `MeleeMedals`, `GrenadeMedals`, `WeaponBytesToInt`). Les douze
      autres deviennent non exportées ; lecture par `filmshell/accesseurs.go` (consultation ou itérateur `iter.Seq`
      qui parcourt la table sans la livrer) : `NomDArme`, `NombreDArmes`, `Armes`, `EstUneArme`, `EstOctetsDArme`,
      `OctetsDesArmes`, `SuffixeCommun`, `SuffixesFormuleA`, `NomCanonique`, `IDCanonique`, `EstSentinelle`,
      `Sentinelles` (`GetTiming` inchangé). Lecteurs migrés : grammaire (`weaponscan/scanner.go`, `weaponv3/canon.go`,
      3 tests, 4 fichiers `research`), `replay`, go-api (`platform/duckdb/match_view_repo_weapons.go`,
      `sync/killcollector/shots.go` et son test, `persist/weapon_shots_sentinels_test.go`, `cmd/diag_film`).
      `catalogueArmesHorsRatchet` retiré : `TestAucuneVarExporteeModifiableDansLeDecodeur` vert sur `filmshell`.
      Golden régénéré UNE fois par la porte (`-update-grammar-rev` + `LEVELUP_UPDATE_GRAMMAR_REV=1`) :
      `internal/grammar/testdata/grammar_rev.golden`, ligne `grammar-2026-10-28`, empreinte `d47f789c…` ->
      `375f3f47…`, seule ligne changée ; entrée de chronique « COMPLÉMENT DU 2026-11-04 » dans
      `internal/grammar/rev_chronique.go`. `filmshell` n'est que dans `grammar_perimetre.golden` (grep des cinq
      `*_perimetre.golden`) : `killsource` et `objectives` ne bougent pas ; golden de périmètre inchangé. Cinq `Rev`
      inchangées, gates de révision verts.
- [x] 2.5 `ctx` sur les 12 appels `slog` de la sentinelle mémoire restante et d'`internal/atomicfile`
      (`slog.*Context`), signatures mises à jour, appelants migrés ; exclusion `horsPerimetreDeJournal` retirée.
      — FAIT (2026-11-08, reprise après coupure de quota). `memguard.Arm` et `ArmBytes` prennent `ctx` en premier
      paramètre (`slog.InfoContext`) ; `atomicfile.WriteFile(ctx, ...)` dans `libs/film/internal/atomicfile` ET dans sa
      copie jumelle `apps/go-api/internal/platform/atomicfile` (`slog.WarnContext`), `TestLesCopiesJumellesSontIdentiques`
      vert. Appelants migrés : 8 `cmd/*` (le `ctx` de leur `main` ou de leur fonction, `context.Background()` seulement
      là où la signature n'en porte pas), `replaybuild`, `replaychild`, `filmarchive`, `settings`, `dbprofiles`,
      `notify`, `ops`, `sync/replayartifacts`, deux outils `research`. Le décompte « 12 appels » du plan est en réalité
      6 appels de production (2 `memguard`, 2 `atomicfile` de la brique, 2 de la copie jumelle). `horsPerimetreDeJournal`
      retirée de `film_slog_contexte_test.go` (le commentaire périmé de D-L2-4 part avec elle). Mutation : un
      `slog.Info("mutation")` dans `memguard.go` -> ROUGE (`libs/film/memguard/memguard.go:146: slog.Info sans contexte`),
      retiré -> vert (`temoins_film/lot2/2.5_mutation_slog_{rouge,vert}.txt`). Reprise : un remplacement de masse avait
      aussi réécrit `sentinelleTokens` et un message de `no_unbounded_film_loop_test.go` (`memguard.Arm(t.Context(), `) :
      remis à `memguard.Arm(` / `memguard.Arm(ctx, ...)`. Les 11 fichiers `research && campagne_overlay` modifiés par le
      premier passage sont restaurés (ils sont supprimés au 2.6). `go build` et `go vet` des deux modules verts.
- [x] 2.6 `git rm` des 21 fichiers `research && campagne_overlay` (liste fermée par
      `grep -rlE '^//go:build research && campagne_overlay' libs/film`) ; ligne D2-L0-b du registre soldée.
      — FAIT (2026-11-08). `grep -rlE '^//go:build research && campagne_overlay' libs/film` : 21 fichiers (tous sous
      `libs/film/internal/grammar`, liste dans `temoins_film/lot2/2.6_fichiers_supprimes.txt`), `git rm` des 21 ; le grep
      rend 0 ensuite. Ligne D2-L0-b de `.ai/REGISTRE_REPORTS.md` barrée et soldée (« SOLDÉ le 2026-11-08 »). Le commentaire
      de `.github/workflows/ci.yml` qui disait « le jeu de balises `research campagne_overlay` ne compile pas » n'a plus
      d'objet : réduit aux deux faits vrais (instruments `research` compilés, autonomie du module). Aucun autre
      renvoi à `campagne_overlay` hors archives et plans clos.
- [x] 2.7 `research` : reste derrière son tag ; chaque instrument qui importait le lanceur ou le cache déplacé
      prend le nouveau chemin ou son équivalent côté brique ; `go vet -tags research ./...` vert.
      — FAIT (2026-11-08). `go vet -tags research ./...` : code 0 dans `libs/film` ET dans `apps/go-api` (CGO, gcc
      de msys64 ucrt64 en tête du PATH), sorties dans `temoins_film/lot2/2.7_vet_research_{film,api}.txt`. Les instruments
      qui armaient `memguard.Arm` ou écrivaient par `atomicfile` suivent la nouvelle signature (`cmd_fermeture`,
      `cmd_grenadeids`, `emprise_*_research_test.go`, les `*_research_test.go` de `replay`/`killsource`/`objectives`/
      `grammar` hors les 21 supprimés au 2.6) ; ceux qui prenaient le verrou de décodage passent par
      `libs/film/internal/decodelock` (2.1) ; aucun n'importait `filmcache` en écriture.

- [x] 2.8 Garde-rails et ADR : ratchet `film_sans_exploitation` (aucun `os/exec`, aucun `os.Getenv` hors
      `revision/porte.go`, aucune écriture de fichier hors `internal/` et des écritures de goldens sous porte) ;
      ADR 0034 amendée d'un paragraphe « ce que la brique ne décide pas » (règle ci-dessus) ; ADR 0041 : table des
      paquets de la brique mise à jour.
      — FAIT (2026-11-08). Ratchet `apps/go-api/internal/archlint/film_sans_exploitation_test.go`
      (`TestLaBriqueNeFaitPasDExploitation`), balayage AST de la production de `libs/film` (hors `research/`, hors
      `_test.go`, hors fichiers sous tag `research`) : aucun import `os/exec` ; aucun `os.Getenv` / `LookupEnv` /
      `Environ` hors `libs/film/revision/porte.go` ; aucune écriture (`WriteFile`, `Create`, `CreateTemp`, `Rename`,
      `Mkdir*`, `Remove*`, `OpenFile` ouvert en écriture ou à drapeau illisible) hors `libs/film/internal/` et
      `libs/film/revision/`. Allowlist : aucune. Vert d'emblée (relevé du 2026-11-08 : `os.Getenv` nulle part dans la
      brique, écritures seulement sous `internal/` et `revision/`). Mutations jouées dans `libs/film/replay` (fichier
      jetable retiré ensuite) : `os.Getenv("X")` -> ROUGE `zz_mutation.go:5: os.Getenv hors de revision/porte.go` ;
      `os.WriteFile` -> ROUGE ; `os.OpenFile(..., os.O_WRONLY, ...)` -> ROUGE ; `os.OpenFile` à drapeau variable ->
      ROUGE ; import de `os/exec` -> ROUGE ; `os.OpenFile(..., os.O_RDONLY, 0)` -> vert (lecture admise) ; fichier retiré
      -> vert (`temoins_film/lot2/2.8_mutations.txt`). Première version : `os.O_RDONLY` rougissait à tort (le nom du
      sélecteur était lu comme une variable) ; corrigé avant la mutation finale. ADR 0034 : amendement du 2026-11-08
      « what the brick does not decide » (la règle, les trois paquets de l'hôte `filmrun` / `filmarchive` /
      `filmreplis`, le ratchet) et ligne 346 corrigée (`filmproc.AcquireSolo` -> `filmrun.AcquireSolo`, D-L2-5). ADR 0041 :
      la phrase des paquets de la brique `film` mise à jour (`memguard`, `filmcache` en lecture, auxiliaires fermés
      `digest`/`atomicfile`/`filelock`/`decodelock`/`depottest`, copies jumelles, renvoi à l'ADR 0034). Les ADR sont
      EN-only (règle 15) : pas de traduction.

- [x] 2.9 Gate : T1 (disparitions = les 21 fichiers + les tests déplacés, nommés un à un, 0 nom disparu sinon),
      T2 (0 golden changé, ou seuls ceux régénérés en 2.4 avec chronique), T3, T4 (`--base 421107998f`), T5, T7 ;
      lint des cinq modules à 0 issue (`golangci-lint run --config ../../apps/go-api/.golangci.yml ./...`) ;
      `make desktop-test` si `filmproc` était cité par la coquille (vérifier : `grep -r filmproc apps/desktop`).
      Commit « lot 2 ».
      — FAIT (2026-11-08, HEAD `12896aaa9f`, tous témoins rejoués dans cet ordre, un seul gros processus à la fois ;
      sorties sous `temoins_film/lot2/2.9_*`). Tableau « Gate 2.9 » ci-dessous. Lot 2 CLOS : items 2.1 à 2.9 `[x]`.

#### Gate 2.9 (2026-11-08)

| Témoin | Attendu (lot 0) | Obtenu | Identique | Durée |
|---|---|---|---|---|
| build / vet | vert | `go build`, `go vet`, `go vet -tags research` verts dans `apps/go-api` et `libs/film` | oui | 155 s |
| T1 tests | 0 fail ; disparitions justifiées | 21 581 lignes (paquet, test, verdict) : 21 206 `pass`, 0 `fail`, 375 `skip` (lot 0 : 21 556, 21 176, 0, 380) ; MD5 `f91c2acd2bea6ce736f5f54957bb6dbe` ; `go-api` hors archlint 16 914, archlint seul 490 (`-timeout 40m`, comme `outils/run_t1.sh`), `libs/film` 3 803, `gamefiles` 85, `haloapi` 216, `himap` 73 ; six codes de sortie 0 | oui, voir ci-dessous | 536 s |
| T2 révisions | cinq `Rev` inchangées ; goldens identiques hors `grammar_rev.golden` | `source-2026-09-16.2`, `profile-2026-10-16`, `grammar-2026-10-28`, `killsource-2026-10-18`, `objectives-2026-10-17` ; 35 `*.golden` comparés par SHA-256 au commit `421107998f` (`git show`) : 34 identiques, UN changé : `libs/film/internal/grammar/testdata/grammar_rev.golden` (ligne `grammar-2026-10-28`, `d47f789c...` -> `375f3f47...`, seule ligne du diff), régénéré au 2.4 par la porte à deux verrous, entrée de chronique « COMPLÉMENT DU 2026-11-04 » dans `internal/grammar/rev_chronique.go:403` ; `go test ./revision/... -count=1` vert | oui (écart nommé par 2.4) | 4 s |
| T3 rejeu | 20 identiques, MD5 des 20 digests = lot 0 | `replay-equiv` : 20 identiques, 0 différent, 0 écarté, 0 échec, 0 illisible ; MD5 de la liste des 20 `e35052a1baa65ea8519fa373d8e51271` = lot 0 ; les 20 fichiers octet-identiques à `libs/film/replay/testdata/equivalence/*.tsv` (`cmp`) | oui | 1 275 s |
| T4 parc | 18 identiques, 0 gain, 0 perte, 0 changement, 1 absent, code 0 | `replay-corpus-gate --base 421107998f --allow-missing` : 18 `ok` (schéma 106 -> 106), `f75e7053` ABSENT, 0 gain, 0 perte, 0 changement, `couverture_incomplete=false`, code 0 ; jonctions retirées après (`rmdir`, cibles intactes) | oui | 599 s |
| T5 autonomie | `levelup/go-api` = 0 | `libs/film` (278 paquets), `gamefiles` (149), `himap` (154), `haloapi` (233) : 0 | oui | 10 s |
| T6 surface | 179 et 316 | `plafondSurfaceFacade` et `plafondSurfaceReplay` tenus par `TestSurfaceDeLaFacadeDuDecodeur` / `TestSurfaceDeReplayCiteeHorsDuDecodeur`, verts dans le T1 archlint | oui | (dans T1) |
| T7 intégration | code 0, 16 paquets, 2 321 tests (2 289 pass, 32 skip) | code 0, 16 paquets `pass`, 2 321 tests : 2 289 `pass`, 0 `fail`, 32 `skip` | oui | 829 s |
| Lint | 0 issue | `libs/film` (config de go-api) : 0 issue ; `apps/go-api` `--new-from-merge-base=origin/release/v8.1.0` : 0 issue (aucun `//nolint` ajouté) | oui | 86 s + 167 s |
| `filmproc` dans `apps/desktop` | vide ou `make desktop-test` | `grep -r filmproc apps/desktop` : vide, `make desktop-test` sans objet | oui | - |
| Ratchet du lot | vert | `TestLaBriqueNeFaitPasDExploitation`, `TestLeDecodeurJournaliseSousLeContexteDeLAppelant`, `TestLaBriqueRemonteVersLeDepotParUneSeuleAide`, `TestLeVerrouDeDecodageParleLeMemeProtocoleDesDeuxCotes`, `TestLesCopiesJumellesSontIdentiques`, `TestCopiesIdentiques_AutoPreuve` verts. Le motif `SansExploitation\|SlogContexte\|RacineDepotUnique\|VerrouDecodageParite\|CopiesIdentiques` du gate ne retient que `TestCopiesIdentiques_AutoPreuve` (les autres noms de tests ne portent pas ces mots) : rejoué avec les noms réels | oui | 6 s |

T1, détail : 97 lignes du lot 0 absentes, 96 retrouvées NOM POUR NOM, verdict égal, à leur nouvelle adresse (`libs/film/digest` 15 ->
`libs/film/internal/digest` + `apps/go-api/internal/analysis/digest` ; `libs/film/filmcache` 37 -> `internal/filmarchive` ;
`libs/film/filmproc` 35 -> `internal/filmrun` et 9 -> `libs/film/memguard`) ; la 97e, `TestChaqueEnfantDuPaquetPasseParSansFenetre`,
est le seul nom supprimé (la copie `sansFenetre` part avec son sujet ; l'appel par `childproc.Command` est tenu par
`archlint/childproc_single_spawn_test.go`). Les 21 fichiers `research && campagne_overlay` du 2.6 (commit `a415ceb791`) ne
portaient aucun test dans T1 (tag `research` + `campagne_overlay`). 122 lignes neuves, dont 11 sans homologue disparu :
`decodelock` 2, `filmreplis` 2, `platform/filelock` 3, `filmrun` `TestCommandeDeLEnfantDecriteOctetPourOctet`, archlint 3
(`TestLaBriqueNeFaitPasDExploitation`, `TestLaBriqueRemonteVersLeDepotParUneSeuleAide`, `TestLeVerrouDeDecodageParleLeMemeProtocoleDesDeuxCotes`).
0 fail. Cinq verdicts passent de `skip` à `pass` (lot 0 : 380 skip, maintenant 375) : `positions/TestDecodeKeyframePositions_Ref`
et quatre tests de `weaponv3` ; cause : ils sautent quand le film `000d5950` est absent du cache (« cache film 000d5950 absent »,
sortie du lot 0) et les jonctions du parc étaient montées pendant ce T1 (jonctions du 2026-11-08 11:11 vers le checkout principal) ;
amélioration de couverture due au poste, pas à un changement de code. Aucun verdict ne passe à `fail` ou de `pass` à `skip`.

### Arrêt du lot 2 sur l'item 2.4 (2026-11-04)

Items 2.5 à 2.9 NON COMMENCÉS : le lot s'arrête sur 2.4 (`[!]`, décision du pilote) comme le contrat d'exécution
du lot le prescrit pour un invariant qui casse. Les items faits (2.1 à 2.3b) sont vérifiés par un gate partiel,
commité à part (témoins sous `temoins_film/lot2/`) :

| Témoin | Résultat |
|---|---|
| T1 | 21 580 lignes (paquet, test, verdict) : 21 200 `pass`, 0 `fail`, 380 `skip` (380 au lot 0), 0 verdict changé (`go-api` hors archlint, archlint seul `-timeout 40m`, quatre briques ; MD5 `b3d602573eaaacc274b0f817ac4d04e2`). 97 lignes du lot 0 absentes, toutes des trois paquets déplacés et retrouvées NOM POUR NOM à leur nouvelle adresse (`libs/film/digest` 15 -> `libs/film/internal/digest` ; `libs/film/filmcache` 37 -> `apps/go-api/internal/filmarchive` ; `libs/film/filmproc` 45 -> `internal/filmrun` 36 + `libs/film/memguard` 9), sauf UN nom supprimé avec son sujet : `TestChaqueEnfantDuPaquetPasseParSansFenetre` (la copie `sansFenetre` disparaît, l'appel par `childproc.Command` est tenu par `archlint/childproc_single_spawn_test.go`). 121 lignes neuves : les 96 déplacées, `TestCommandeDeLEnfantDecriteOctetPourOctet` (témoin 2.1), `analysis/digest` 15 (copie jumelle), `platform/filelock` 3 (copie jumelle), `internal/decodelock` 2, `filmreplis` 2, archlint 2 (`TestLeVerrouDeDecodageParleLeMemeProtocoleDesDeuxCotes`, `TestLaBriqueRemonteVersLeDepotParUneSeuleAide`). Liste du lot 0 comparée après décodage des échappements JSON (`\\` -> `\` dans 20 noms de sous-tests). Deux corrections pendant le gate : le témoin `filmcache/testdata/manifeste_partiel_34.json` remis dans la brique (lu aussi par `replay/fil_des_morts_faits_test.go`) ; `killicon_test.go` ramené à 610 lignes (plafond figé). `TestGetMatchFilm_ParallelDownloadFasterThanSequential` a rougi une fois sous charge (531 ms > 500 ms, code non touché), vert 3 fois sur 3 seul et rejoué en JSON (D-L2-7) |
| T2 | cinq `Rev` inchangées ; SHA-256 des 34 `*.golden` identiques au lot 0 ; `go test ./revision/... ./internal/... -run 'Rev|Empreinte|Perimetre'` vert |
| T3 | `replay-equiv` : 20 identiques, 0 différent, MD5 des 20 digests identiques au lot 0 et octet-identiques aux références `replay/testdata/equivalence/*.tsv` (732 s) |
| T4 | `replay-corpus-gate --allow-missing --base 421107998f` : 18 `ok`, 1 absent (`f75e7053`), 0 gain, 0 perte, 0 changement, `couverture_incomplete=false`, code 0 (574 s) ; rapport JSON identique à celui du lot 0 (`T4b_rapport.json`) hors `dureeMs` et `baseDuCache` (la base est relue de son cache) |
| T5 | `levelup/go-api` = 0 dans `libs/film` (278 paquets), `gamefiles` (149), `haloapi` (233), `himap` (154) |
| Lint | `libs/film` : 0 issue ; `apps/go-api` `--new-from-merge-base=origin/main` : 3 `goconst` dans `filmreplis/sites.go`, corrigés (constantes de fichier), puis 0 issue sur les cinq paquets neufs ; relancé après les corrections du gate : `apps/go-api` (`--new-from-merge-base=origin/main`) et `libs/film` à 0 issue |

## Lot 3 — Ratchets de la brique

- [x] 3.1 Ordre des imports des sources hachées (D2-L3-a) : test dans `revision` qui compare l'ordre des imports
      de chaque fichier des périmètres à celui figé, message qui nomme la cause. Preuve : un `goimports` simulé
      rougit avec ce message.
      — FAIT (2026-11-08). `libs/film/revision/ordre_imports_test.go` (`TestOrdreDesImportsDesSourcesHachees`).
      MESURE d'abord : les sources hachées sont les 398 fichiers de production de la fermeture des cinq couches
      (arbre de chaque couche + paquets importés). Règle réellement suivie : (a) GOFMT-STABLE, vrai des 398 : chaque
      groupe d'imports (bloc séparé par une ligne vide) est trié ; (b) ORDRE STANDARD (bibliothèque standard triée,
      puis le reste trié) pour 383 d'entre elles ; (c) QUINZE sources, et non quatorze, gardent un ordre historique
      en groupes séparés : les 14 qui portent la ligne « L ordre de ces imports entre dans l empreinte » ET
      `internal/grammar/deaths_source.go`, qui ne la porte pas (D-L3-1). Les quinze n'obéissent pas à une même règle
      de tri (les 14 marquées sont « un bloc unique trié sous l'ancien chemin de module », `deaths_source.go` met `highlightevent`
      avant `finalise`) : le test fige donc leur suite EXACTE dans la table `ordresFiges` (chemins de la brique
      écrits `@/...`, indépendants du chemin du module), sans toucher aux sources. Règle du test : groupes triés
      pour toutes ; les 383 autres = ordre standard ; les quinze = leur suite figée, et une suite devenue standard
      est une erreur (un outil les a regroupées). Le message nomme le fichier, l'import mal placé, sa place, et
      dit que remettre l'ordre en place évite une régénération d'empreinte, sinon que l'empreinte se régénère par
      la porte à deux verrous. Garde contre l'entrée périmée de la table (fichier qui n'est plus haché).
      Preuves : 7 cas synthétiques (échange dans un groupe, source figée regroupée, import changé de groupe, import
      ajouté, fichier neuf à mise en page historique, intactes) et 4 mutations sur fichiers réels (tableau
      « Mutations du lot 3 », M1 à M4).
- [x] 3.2 Pour chaque ligne « oui » de l'inventaire 0.5 : racine étendue à l'espace de travail (règle ADR 0041
      D7), mutation jouée (rouge puis vert) ; une ligne « non » reste telle quelle, statuée.
      — FAIT (2026-11-08), six lignes, vérifiées sur pièces (le lot 2 a déplacé les paquets) :
      * `decode_lock_interdit_test.go` : `[x]`. Sujet vérifié : les trois noms interdits (`LockProcessDecode`,
        `processDecodeMu`, `decode_gate.go`) n'existent plus nulle part (grep : seulement des chroniques en
        commentaire) ; ils reviendraient dans `libs/film/internal/grammar`, hors de l'ancienne racine
        (`internal`, `cmd`). Racine = `workspaceGoSources(t, racinesEspaceDeTravail...)` (aide existante, pas de
        3e copie). Mutations M5, M6.
      * `no_discarded_remove_test.go` : `[x]` borné. Recompte par grep : les 6 sites de `filmcache/deplacement.go`
        sont partis dans `internal/filmarchive` (hors racines), les 2 de `libs/film/internal/atomicfile/atomicfile.go`
        sont restés ; les trois autres briques n'en ont aucun. Le garde-rail est un garde-rail de chemins de
        requête (`internal/api`, `internal/service`) : étendre à tout l'espace de travail ajouterait 49 sites de
        go-api (cmd/, assets, media, ops, persist, platform, transfer...) à une allowlist de ~35 fichiers, ce qui
        change le sens de la règle (D-L3-4). Racines étendues aux quatre briques de `libs/` seulement ; allowlist
        datée 2026-11-08 pour `atomicfile.go` (2 sites : nettoyage après l'erreur d'écriture déjà rendue à
        l'appelant ; jumeau de `platform/atomicfile`), critère de retrait écrit. Mutations M7.
      * `server_stdout_internal_test.go` : `[~]` D-L2-8. Vérifié : la racine `internal` couvre
        `internal/filmrun/runner.go`, qui porte l'entrée datée (`EmitPeak`, `EmitRaison`, défaut `os.Stdout`) ; aucune
        référence à `os.Stdout` ni `fmt.Print*` dans la production des quatre briques hors `libs/film/research/`
        (outils en ligne de commande, jamais liés au serveur). Aucune racine à étendre.
      * `film_exported_vars_test.go` (racine partielle) : `[x]`. Mesure de l'extension à `libs/gamefiles`,
        `libs/haloapi`, `libs/himap` : 9 variables exportées modifiables, toutes dans `libs/himap/himap`
        (`CartesForge`, `TypesVolumesDeMort`, `ModulesDesEquipements`, `StylesFond`, `LumiereRendu`,
        `NiveauHautNavmesh`, `MargeSolVuDuDessousCarte`, `SeuilSubstitution`, `SeuilCouvertureCarte`) ; 0 ailleurs.
        Extension faite avec une baseline nommée par variable (`varsExporteesAdmises`, datée 2026-11-08, fermée,
        critère de retrait écrit) et un test d'entrée périmée ; aucune production touchée. Mutations M8, M9.
      * `film_file_size_test.go` (racine partielle) : `[x]`. Mesure : 6 fichiers au-delà de 500 lignes (himap :
        `cuisson_forge.go` 1 048, `cartes_forge.go` 719, `cuisson.go` 619, `sonde_locs_research_test.go` 818,
        `physique_sonde_gamefiles_test.go` 530 ; gamefiles : `noms_lieux_hunt_research_test.go` 622 ; haloapi : 0).
        Racines étendues, 6 plafonds gelés à la taille du jour (bloc daté 2026-11-08). Garde par racine non vide
        ajouté (les planchers globaux sont trop bas, D-L3-6). Mutations M10, M11.
      * `film_function_length_test.go` (racine partielle) : `[x]`. Mesure : 1 fonction de production au-delà de
        80 lignes (`CuitCarteForge`, 84, himap) ; gamefiles et haloapi : 0. Racines partagées avec la taille
        des fichiers (par référence), 1 plafond gelé. Mutations M12, M13.
- [x] 3.3 Ratchet `film_sans_hote` : aucun littéral `LEVELUP_`, `apps/go-api`, `internal/games/halo_infinite/film`
      ni chemin absolu dans la production de `libs/film` ; allowlist datée = `revision/porte.go` seul (D-3).
      Mutation jouée.
      — FAIT (2026-11-08), avec un ÉCART À L'ALLOWLIST DU PLAN. `apps/go-api/internal/archlint/film_sans_hote_test.go`
      (`TestLaBriqueNeConnaitPasSonHote`), balayage texte, commentaires compris, de la production de `libs/film`
      (719 fichiers, hors `research/`, hors tag `research`, via `balayerLaProductionHorsResearch`). Relevé : `LEVELUP_`
      seulement dans `revision/porte.go` (D-3, comme prévu) ; mais `internal/games/halo_infinite/film` et `apps/go-api`
      figurent aussi dans `revision/cadre.go` (constante `dossierHistoriqueDeLaBrique` et commentaire), `couches.go`
      (constante `racineDesCouches`) et `perimetre.go` (commentaire) : c'est le CADRE HISTORIQUE du hachage (ADR 0041 D8),
      le contrat des empreintes, que le lot 1 ne pouvait pas réécrire. Allowlist : par fichier ET par littéral
      (porte.go : `LEVELUP_` ; cadre.go : les deux chemins ; couches.go et perimetre.go : l'ancienne adresse), datée
      2026-11-08 avec critère de retrait, plus un test d'entrée périmée. Aucun autre site (0 chemin absolu). Message :
      fichier, ligne, littéral, remède. Mutations M14, M15, M16.
- [x] 3.4 Gate : archlint vert, mutations consignées, T1. Commit « lot 3 ».
      — FAIT (2026-11-08) : voir « Gate du lot 3 » ci-dessous.

### Mutations du lot 3

Chaque mutation est jouée sur un fichier réel ou jetable (sauvegarde puis remise en place, `git status` propre
ensuite), le test vu ROUGE avec le message cité, puis VERT après retrait.

| # | Test | Mutation | Message observé (extrait) | Retour au vert |
|---|---|---|---|---|
| M1 | `TestOrdreDesImportsDesSourcesHachees` | `goimports` simulé : `internal/facts/killsource/hybrid.go` regroupé (`cmp`, `slices`, puis le module) | `hybrid.go : figure dans ordresFiges, mais ses imports sont en ordre standard : un outil (goimports...) les a regroupes. [...] remettre l'ordre en place evite une regeneration d'empreinte [...] porte a deux verrous` | oui |
| M2 | idem | permutation `errors` / `fmt` dans `internal/grammar/deaths_source.go` | `dans le groupe d'imports 1, l'import "errors" est place apres "fmt", que gofmt rangerait apres lui. [...]` | oui |
| M3 | idem | `klauspost/compress/zstd` rangé dans le bloc standard de `internal/source/zstd.go` | `l'import "github.com/klauspost/compress/zstd" est a la place 4, goimports le mettrait avant "sync" [...] ordresFiges` | oui |
| M4 | idem | fichier neuf `internal/grammar/zz_mutation_ordre.go` en mise en page historique | `l'import ".../types" est a la place 2, goimports le mettrait avant "slices" [...] l'ajouter a ordresFiges` | oui (fichier retiré) |
| M5 | `TestAucunVerrouDeDecodageDePaquet` | `func LockProcessDecode()` dans `libs/film/internal/grammar/zz_mutation_verrou.go` | `LE VERROU DE DECODAGE DE PAQUET EST DE RETOUR (1) : libs/film/internal/grammar/zz_mutation_verrou.go : declare ou appelle LockProcessDecode` | oui |
| M6 | idem | `var processDecodeMu int` dans `libs/film/internal/decodelock` | idem, `...decodelock/zz_mutation.go : declare ou appelle processDecodeMu` | oui |
| M7 | `TestNoDiscardedRemoveOnRequestPaths` | `_ = os.Remove(p)` dans `libs/film/decfilm` et `defer os.RemoveAll(p)` dans `libs/himap/himap` | `os.Remove / os.RemoveAll dont l'erreur est écartée : tester le résultat et journaliser l'échec : libs/film/decfilm/zz_mutation_remove.go:5, libs/himap/himap/zz_mutation_remove.go:5` | oui |
| M8 | `TestAucuneVarExporteeModifiableDansLeDecodeur` | `var TableMutable` dans `libs/gamefiles/mapvar`, `var ReglageMutable` dans `libs/haloapi` | `libs/gamefiles/mapvar.TableMutable`, `libs/haloapi.ReglageMutable` [...] baseline datee et fermee au 2026-11-08 | oui |
| M9 | `TestVarsExporteesAdmisesNeSontPasPerimees` | entrée fantôme `himap.VarInexistante` | `est dans varsExporteesAdmises mais n est plus une variable exportee modifiable : retirer l entree` | oui |
| M10 | `TestTailleDesFichiersDuFilmNeCroitPas` | `libs/himap/himap/cartes_forge.go` +1 ligne | `720 lignes, plafond fige a 719 [...] UN PLAFOND NE MONTE PAS` | oui |
| M11 | idem | fichier neuf de 521 lignes dans `libs/haloapi` | `libs/haloapi/zz_gros.go : 521 lignes, seuil 500 (CLAUDE.md regle 5)` | oui |
| M12 | `TestLongueurDesFonctionsDuFilmNeCroitPas` | fonction neuve de 89 lignes dans `libs/gamefiles/mapvar` | `ZzLongue : 89 lignes physiques, seuil 80` | oui |
| M13 | idem | une ligne de plus dans `CuitCarteForge` | `CuitCarteForge : 85 lignes physiques, plafond fige a 84` | oui |
| M14 | `TestLaBriqueNeConnaitPasSonHote` | `os.Getenv("LEVELUP_FAUX")`, commentaire `apps/go-api/cmd`, chaîne `/home/guillaume` dans `libs/film/replay/zz_mut_hote.go` | `la production de la brique nomme son hote (3) : zz_mut_hote.go:5 : "LEVELUP_" ; :7 : "apps/go-api" ; :8 : "/home/"` + remède | oui |
| M15 | idem | `// LEVELUP_X` ajouté à `revision/cadre.go` (littéral admis ailleurs, pas ici) | `libs/film/revision/cadre.go:132 : "LEVELUP_"` | oui |
| M16 | `TestHoteAdmisNEstPasPerime` | entrée fantôme `/opt/` pour `revision/couches.go` | `entree(s) de hoteAdmis sans objet, a retirer : libs/film/revision/couches.go : "/opt/"` | oui |

### Gate du lot 3 (2026-11-08)

Commandes jouées dans cet ordre, un seul gros processus à la fois (`GOCACHE=go-build-film`, `CGO_ENABLED=1`) :
`go vet ./...` dans `libs/film` et `apps/go-api` : vert. `go test ./revision/...` (libs/film) : vert. `golangci-lint run ./revision/...`
(libs/film) et `./internal/archlint/...` (apps/go-api) : 0 issue, aucun `//nolint` dans les fichiers neufs. `go test ./internal/archlint/ -timeout 40m -count=1`
(apps/go-api, seul) : `ok` en 299 s, 0 fail. T1 allégé : `go test ./... -timeout 30m -count=1` dans `libs/film` : 28 paquets `ok`, 0 fail ;
`go test ./internal/filmrun/... ./internal/filmarchive/... ./internal/platform/filelock/...` (apps/go-api) : 3 `ok`, 0 fail (archlint compté
ci-dessus). T2 : les cinq `Rev` inchangées (les tests d'équivalence et de golden de `revision` verts), `git status` ne montre aucun `*.golden`
modifié ni aucun fichier de production modifié : le lot ne touche que des tests, ce plan, le registre et le journal. Fichiers du commit : 6 tests
modifiés ou neufs dans `archlint`, `libs/film/revision/ordre_imports_test.go`, plan, registre, journal.

## Lot 4 — Publication de la brique

- [x] 4.0 Identités des fixtures (D-7) — **OPTION A choisie par le user le 2026-11-04** (sur la mesure 1.6) — FAIT le 2026-11-08, voir « Résultat de l'item 4.0 » :
      l'outil `anonymise-gamertags` (go-api, celui de la v8.0.2) est étendu aux xuids : réécriture à longueur
      constante (64 bits LE à décalage de bit quelconque dans les paquets d'identité et les temps forts, MSB dans
      la table du registre), faux xuids tirés dans `[0x0009000000000000, 0x000A000000000000[` (D-L1-8), table de
      correspondance déterministe (même entrée → même sortie) ; les 3 gamertags restants (`JGtm`, `Chocoboflor`,
      `Madina97294`) remplacés aussi ; appliqué aux 10 dossiers de films de test de `libs/film` ; les 20
      `facts.json` et 8 `assembly_*.golden` régénérés par leurs tests, la table inverse de `replay-equiv` mise à
      jour ; les 32 fichiers de test et 11 commentaires de production qui portent des xuids réels réécrits avec la
      même table. Preuve : T1 (mêmes tests, mêmes verdicts), T2 (révisions inchangées, les goldens de révision
      intacts : `testdata/` hors des empreintes), T3 (20 identiques après re-figement des références AVEC chronique
      écrite : les digests changent par construction, les xuids étant dedans), grep des 133 xuids d'origine dans
      `libs/film` = 0. Témoin avant/après : pour chaque film, (index, temps, nombre de joueurs, nombre de morts)
      identiques au mapping près. Exécutant : Sonnet.
- [x] 4.1 `libs/film/go.mod` : `require` de `libs/gamefiles` et `libs/himap` à leur tag (ordre ADR 0043 D-2) ;
      `go.work` résout toujours la copie locale ; `TestBriquesImports` vert.
- [x] 4.2 README EN (modèle ADR 0043 D-3) : ce que c'est (film Theater Halo Infinite → faits → document de rejeu
      2D), ce que ce n'est pas (ni téléchargement, ni stockage d'application, ni rendu), installation, exemple,
      prérequis (Go, pas de CGO, catalogues fournis par l'appelant, D-4 ; instruments `research` = corpus local du
      mainteneur), révisions de couche et build inconnu = erreur typée, licence AGPL-3.0, contribuer,
      non-affiliation à Microsoft, 343 Industries, Halo Studios.
- [x] 4.3 `example_test.go` compilé, sur un minifilm de `replay/testdata` (aucun fichier hors du module).
- [x] 4.4 `NOTICE.md` du module : origine de `jpt_ids.txt`, `labels.tsv`, `rules.tsv` et des fixtures (D-4, D-7).
- [x] 4.5 Vérification `go get` hors dépôt (`GOWORK=off GOPROXY=direct GONOSUMDB=github.com/JGtm/LevelUp`, module
      jetable du scratchpad, sha de la branche poussée) : `go build` de l'exemple. Consignée.
- [x] 4.6 ADR 0043 amendée (quatre briques publiables, D-1 réécrit, licence, particularités des révisions) ; ADR
      0041 et 0034 : renvoi ; `CLAUDE.md` (table, ADR 0043) ; `ARCHITECTURE_V6` EN et FR si la brique y est
      décrite.
- [x] 4.6b (ajouté le 2026-11-08, décision du pilote) FAIT le 2026-11-08, voir « Résultat de l'item 4.6b » : (A) D-L4-12, le
      corpus `testdata/corpus_objectifs/minibobine_objectifs/` passé à l'outil `anonymise-gamertags` : mesure faite, AUCUN
      identifiant de joueur dans ces octets (chunks coupés avant le premier paquet de type 8), rien à réécrire, statut `[x]`
      par la mesure ; (B) D-L4-10, `decfilm.ErrUnknownBuild` et `decfilm.ErrUnknownFormat` exposés par la façade, test depuis
      un paquet externe, ratchet de surface 179 -> 181 (T6 « sauf décision écrite »). Exécutant : Sonnet.
- [x] 4.7 Revue adversariale (skill `adversarial-review`) : deux relecteurs Sonnet aveugles sur le diff des lots 1
      à 4 ; corrections avec test vu rouge ; pas de seconde ronde sauf P0. Ronde 1 corrigée le 2026-11-08 (C1 à C4, voir
      « Résultat de l'item 4.7 ») ; cochée par le pilote le 2026-11-08 après la ronde 2 (aucun constat recevable).
- [x] 4.8 fait le 2026-11-08 (Haiku) : backlog, registre, project_map ; contrôle grep des chemins cités.
- [ ] 4.9 PR vers `release/v8.1.0`, CI verte, squash. Commande de tag préparée pour le user :
      `git tag libs/film/v0.1.0 <sha> && git push origin libs/film/v0.1.0` (geste du user, ADR 0043 D-2).

### Résultat de l'item 4.0 (2026-11-08)

Fait, avec quatre écarts au texte de l'item (D-L4-1 à D-L4-4) : les fixtures encodées par le codec des faits se
réécrivent par décodage et réencodage (le motif binaire ne les voit pas) ; la liste des xuids d'origine reste HORS dépôt
(le registre versionné ne porte que les fictifs) ; les 20 références de `replay-equiv` n'ont PAS bougé ; le registre
versionné est un ratchet.

**Ce qui est en place.** `apps/go-api/internal/identitefictive` : le fictif d'un xuid est `0x0009000000000000 + (8 premiers
octets de SHA-256(graine publique || xuid sur 8 octets) masqués sur 48 bits)`, donc dans la plage de D-L1-8, même entrée
même sortie, injectivité vérifiée par l'outil sur l'ensemble traité. `apps/go-api/cmd/anonymise-gamertags` (étendu) : le
xuid se cherche en 64 bits petit-boutiste et en 64 bits gros-boutiste (MSB) aux huit décalages de bit, puis en texte
(décimal, hexadécimal, hexadécimal à octets inversés) à bornes nettes ; un nom marqué `donnees` dans la table ne se remplace
que sous `testdata` ; mode inventaire par défaut (écriture sous `GAMERTAGS_ECRIRE=1`) avec comptes par forme, par dossier et
par fichier. `libs/film/testdata/IDENTITES.tsv` : registre des 420 fictifs, plus la borne de plage et les sept xuids inventés
des tests (inchangés), avec la règle d'origine en en-tête ; son garde-rail `TestLesXuidsDesFixturesSontFictifs`
(cmd/anonymise-gamertags) refuse tout xuid de la plage hors registre dans un fichier texte de `libs/film`.
`libs/film/replay/golden_inputs_identites_test.go` : garde-rail des 8 `inputs_*.bin.gz` (chaque xuid décodé est au registre)
et porte de réécriture par le codec (`REPLAY_IDENTITES_TABLE`, table `original<TAB>fictif` écrite par l'outil sous
`XUIDS_TABLE_SORTIE`). `cmd/replay-equiv/identites.go` : les faits figés portent des xuids fictifs, le harnais les
RETRADUIT en ceux du parc avant de cuire (`-originaux` / `LEVELUP_IDENTITES_ORIGINALES`) ; sans la liste, des faits fictifs
sont refusés.

**Occurrences remplacées** (outil, `libs/film`, 2 451 fichiers parcourus, 119 touchés au premier passage plus 1 au second) :
64 bits petit-boutiste 4 128, gros-boutiste 150, décimal 822, hexadécimal 2 (`roster_type8.go`, deux formes), noms 715 ;
plus 1 379 xuids réécrits par le codec dans les 8 `inputs_*.bin.gz`. Par film (petit-boutiste / gros-boutiste / noms) :
`minifilm_000d5950` 262 / 0 / 30 ; `111fa685` 496 / 24 / 30 ; `11de8353` 417 / 24 / 8 ; `60ae07c4` 498 / 8 / 62 ;
`a521164d` 301 / 24 / 16 ; `bcb6d393` 144 / 8 / 67 ; `e5adf7b2` 514 / 23 / 25 ; `fb1a1a72` 396 / 8 / 113 ;
`minibobine_000d5950` 334 / 8 / 40 ; `minibobine_e5adf7b2` 652 / 23 / 31 ; `v41_chunk_he.bin` 113 / 0 / 113 ;
`pied_bloc_53ce4390.bin` 1 / 0 / 0. Les petit-boutiste et gros-boutiste sont ceux de la mesure 1.6 colonne pour colonne
(4 124 et 150) sauf `a521164d` (+4) : un xuid présent dans les octets qui n'est ni siège ni victime ; la mesure 1.6 en
annonçait 133 distincts, l'outil en trouve 135 (ce xuid et celui du pied de film).

**Témoins 4.0** (scratchpad `temoins_film/lot4/`) :

| Témoin | Avant | Après |
|---|---|---|
| Identités lues par le décodeur dans les 10 dossiers (table des joueurs, fil des morts : index, temps, xuid, gamertag) | 150 sièges, 1 402 morts, 133 xuids distincts, 10 films | mêmes 150 sièges et 1 402 morts, mêmes index et mêmes temps ligne à ligne ; les 133 xuids sont les 133 fictifs prévus par la dérivation, les gamertags identiques sauf les 155 occurrences des trois noms ; 0 écart (`avant_identites.txt`, `apres_identites.txt`, comparés par `compare_identites.py`), tous au registre |
| T1 `libs/film` (paquet, test, verdict) | 3 813 lignes, 0 fail | 3 823 lignes, 0 fail : les 3 813 anciennes lignes à l'identique, plus 10 neuves (`TestGoldenInputsPorteDesXuidsFictifs` x9, `TestGoldenInputsReecrireIdentites` ignoré) |
| T1 go-api (15 jeux de paquets qui lisent les fixtures de `libs/film`) | 6 102 lignes, 3 fail (`haloadapter.TestPolitiqueParChemin`, avant tout changement) | 6 113 lignes, 0 fail ; seules différences : 11 tests neufs et `TestPolitiqueParChemin` passé au vert (échec avant le lot, sans rapport, non rejoué) |
| T2 | cinq `Rev` : `source-2026-09-16.2`, `profile-2026-10-16`, `grammar-2026-10-28`, `killsource-2026-10-18`, `objectives-2026-10-17` ; 14 tests de `revision` verts ; SHA-256 des 10 `*_rev.golden` et `*_perimetre.golden` | identiques ; 14 tests verts avec les mêmes noms ; les 10 goldens de révision et de périmètre identiques à l'octet (17 sources de production modifiées par une seule chose : un xuid ou un nom dans un commentaire, empreinte insensible aux commentaires vérifiée) |
| T3 `replay-equiv` (20 films du parc) | 20 identiques, MD5 de la liste `e35052a1baa65ea8519fa373d8e51271` | 20 identiques, 0 différent, MD5 de la liste IDENTIQUE, les 20 fichiers octet-identiques aux références ; 1 209 s ; `xuids_retraduits` figure à chaque cuisson |
| Goldens régénérés par leurs tests | 8 `assembly_*.golden`, `minifilm.tsv`, 8 fixtures de contrat web | les 8 `assembly_*.golden` régénérés par `-update` sont identiques à l'octet à ceux que l'outil avait réécrits ; `minifilm.tsv` : `deaths` et `playerIndices` (les seules étapes qui citent un xuid) ; `highlight_event_golden.txt` : la ligne `sha256` ; 8 fixtures web et leur manifeste régénérés par `REPLAY_CONTRACT_UPDATE=1 go test -run ContractFixtures -update` |
| Recherche des originaux | 420 xuids d'origine en texte (dont 1 en hexadécimal seul) ; 3 gamertags | l'outil en inventaire sur tout `libs/film` : 0 occurrence des 420 sous les cinq formes et des 3 noms sous `testdata` ; `git grep -F` : 0 xuid d'origine, 0 `Chocoboflor`, 0 `Madina97294` ; `JGtm` : 0 sous `testdata`, 66 lignes de prose, de noms de tests et de commentaires restent (c'est le nom du mainteneur, aussi dans le chemin de module de chaque fichier) |
| `go test ./internal/archlint/ -timeout 40m -count=1` | vert | `ok`, 340 s |

Aucune `Rev` ni aucun golden de révision n'a bougé : rien à régénérer, aucune chronique de couche à écrire. Les références
de `replay-equiv` n'ont pas été re-figées (D-L4-3) : pas de chronique de re-figement, un paragraphe dans
`replay/testdata/equivalence/CORPUS.txt` dit pourquoi.

Fichiers hors dépôt, à garder : `scratchpad/hors_depot4/xuids_originaux.txt` (liste des 420), `noms_trois.txt` (table des
trois noms), `xuids_synthetiques.txt`. Ils se reconstruisent depuis `git show bb15d7d950` (grep des nombres à seize chiffres de
la plage, plus `2535406167016809` cité en hexadécimal). Sans eux, `replay-equiv` refuse les faits figés et une régénération de
fixture depuis le cache remet des identités réelles (à rejouer : l'outil, puis `REPLAY_IDENTITES_TABLE`).

#### Complément de l'item 4.0 (2026-11-08) : liste locale, tests de la brique, fixtures web

Trois défauts de livraison du commit précédent, corrigés sans autre changement.

1. **La liste des identifiants d'origine vit dans un dossier local stable.** `replay-equiv` lit la liste par `-originaux`,
   sinon `LEVELUP_IDENTITES_ORIGINALES`, sinon `data/film_identites/xuids_originaux.txt` sous la racine du dépôt
   (`originauxEffectifs`, `cmd/replay-equiv/identites.go`, testé). Le parc de films (`data/cache/film_chunks`,
   `film_manifests`) se trouve déjà par la racine du dépôt ; un worktree le monte par des jonctions NTFS vers le dépôt
   principal (gate 2.9) : le dossier `data/film_identites` se monte de la même façon, solution déjà en place, aucune
   nouvelle variable. `.gitignore` : `data/film_identites/` (identifiants réels, jamais versionnés) ; `git ls-files
   data/film_identites` : 0 fichier. Le message de refus des faits fictifs nomme le chemin par défaut ; le commentaire de tête
   d'`identites.go`, l'aide du drapeau, l'en-tête de la cible `make replay-equiv` et le paragraphe de `CORPUS.txt` disent
   l'emplacement local par défaut, sans chemin absolu. Les huit fichiers du dossier (liste des 420, table original/fictif,
   trois noms, etc.) restent à garder : ils remplacent `scratchpad/hors_depot4/` cité plus haut.
2. **Les deux tests de la brique qui rougissaient avec le parc travaillent sur la fixture** (option a).
   `TestResolveXuidToPI_GroundTruth`, `TestResolveXuidToPIStrings_IgnoresBots`, et aussi `TestResolveBest_MergesChunks`
   (qui ne rougissait pas mais devenait vide de sens : xuids fictifs contre un cache réel, rien de résolu) lisent
   `chunk_01.bin` et `chunk_02.bin` de `replay/testdata/minifilm_000d5950` (zlib, fictifs) par `fixtureChunk` ; plus de saut,
   plus de dépendance au parc. `TestUSEstimator_CacheMonotone` (aucun xuid) garde le cache du parc.
3. **Fixtures web vérifiées** : `npm ci` dans le worktree (512 paquets, 13 s), puis `vitest run` complet.

Preuves : `go vet ./...` `libs/film` et `apps/go-api` verts ; `go test ./internal/grammar/weaponv3/... ./replay/...`
(`libs/film`) AVEC le parc monté : `weaponv3` ok 0,4 s, `replay` ok 43 s, `TestUSEstimator_CacheMonotone` joué ; SANS parc : mêmes
paquets `ok`, les quatre tests de résolution passent sans saut, seul `TestUSEstimator_CacheMonotone` se saute ;
`go test ./cmd/replay-equiv/...` ok ; `go test ./internal/archlint/ -timeout 40m -count=1` ok en 244 s ; `golangci-lint` 0
issue (`cmd/replay-equiv`, `weaponv3`) ; `vitest run` sous `apps/web` : 1 062 fichiers passés, 5 sautés, 10 942 tests
passés, 23 sautés, 0 échec (les 8 fixtures régénérées lues par `match-replay` : aucun littéral à corriger, aucun xuid réel attendu).
`go run ./cmd/replay-equiv -out-dir <scratchpad>` depuis le worktree, SANS `-originaux` ni variable (dossier monté par jonction) :
BILAN 20 identiques, 0 différent, 0 écarté, 0 échec, 590 s ; `xuids_retraduits` renseigné à chaque cuisson (la liste est donc trouvée par défaut).

### Résultat des items 4.1 à 4.6 (2026-11-08)

**4.1 `go.mod` aux tags.** `libs/film/go.mod` : `require github.com/JGtm/LevelUp/libs/gamefiles v0.1.0` et
`.../libs/himap v0.1.0` ; `go.sum` complété par `GOWORK=off GOFLAGS=-mod=mod GOPROXY=direct
GONOSUMDB=github.com/JGtm/LevelUp go mod tidy` (4 lignes ajoutées, aucune autre ne bouge). Tags vérifiés sur `origin`
(`git ls-remote --tags origin 'libs/*'`) : `libs/gamefiles/v0.1.0` (`e2c449b3`) et `libs/himap/v0.1.0` (`5c37ca56`).
Preuves : `go build ./...` dans `libs/film` depuis l'espace de travail code 0 et `go list -f '{{.Dir}}'` sur
`libs/gamefiles/mapvar` rend le dossier du worktree (la copie locale est résolue) ; `GOWORK=off ... go build ./...` code 0
(les tags se résolvent) ; `go test ./internal/archlint/ -run Briques -count=1` ok (5 tests). Rien lancé dans `apps/go-api`
qui touche son `go.mod`.

**4.2 README.** `libs/film/README.md` (EN, modèle des trois autres : statut, ce que c'est avec le tableau des cinq couches et
les paquets publics, ce que ce n'est pas, installation, exemple, prérequis, révisions de couche et build inconnu, licence,
contribuer, non-affiliation). Commandes citées, toutes jouées : `go get ...@v0.1.0` (4.5), `CGO_ENABLED=0 go build ./...` et
`CGO_ENABLED=0 go vet ./...` codes 0 (aucune production ne lie du C : `go list -deps ./...` ne contient ni `ooz` ni
`himodule` ; `libs/himap` n'est importé que par des tests), `go vet ./... && go test ./...` (gate), `go test ./revision/`
ok. Pas de fichier `LICENSE` par module : aucune des trois autres briques n'en porte, le README renvoie à `LICENSE` et
`NOTICE` de la racine comme `himap` (D-L4-8). **D-L3-5 reprise** : prose périmée corrigée dans les fichiers ni hachés
ni comparés (vérifié par grep de leurs lecteurs : aucun test ne lit ces textes) : les 8 `PROVENANCE.txt` de
`replay/testdata/minifilm_*` (et les deux générateurs `replay/minifilm_test.go`, `replay/minifilm_builds_test.go` qui
réécrivent ce texte à la régénération : sans eux, la prochaine régénération aurait remis l'ancienne commande), la
`PROVENANCE.txt` de `pied_bloc_53ce4390` (le test est sous `internal/grammar/signaux/`, pas `internal/grammar/`), celle de
`vue_a_bcb6d393_1_204`, `ecs_table.README.md` (le test `G2` est sous `internal/grammar/`), la ligne « Regeneration » de
`bench_baseline.txt` (les lignes `pkg:` et `ok` sont la sortie réelle d'un banc : laissées), et la citation d'un fichier de
`.ai/` dans `testdata/corpus_objectifs/README.md`. Commande de forme : `go test ./replay/ -run ...` « depuis libs/film ».
Laissés : les 33 en-têtes de goldens et les `*_perimetre.golden` (D-6 : leur correction est une régénération), les lignes
193 et 282 de `CORPUS.txt` (la première cite une mesure datée, la seconde est juste : commande depuis `apps/go-api`), les
lignes 78-79 de `corpus_objectifs/README.md` (historique git d'un paquet supprimé, valides). Aucun `*.golden` modifié :
`git status --short | grep golden` vide.

**4.3 Exemple.** `libs/film/decfilm/example_test.go`, `ExampleScanDeaths` : `decfilm.LoadDir` sur
`../replay/testdata/minifilm_000d5950` (chemin relatif au paquet, dans le module), `ScanDeaths`, sortie vérifiée
`93 deaths, 8 distinct victims` (93 = ligne `deaths` de `replay/testdata/equivalence/minifilm.tsv`). Il n'y avait pas de test
de fraîcheur README <-> exemple pour les autres briques (leurs README sont recopiés à la main) : ajout de
`decfilm/readme_test.go` (`TestLeReadmeCiteLExempleCompile`) qui exige que le bloc de code de la section « Example » soit le
corps de l'exemple, à l'octet. Mutation : `94` à la place de `93` dans le README -> test rouge avec le diff ; remis -> vert.
`golangci-lint` 0 issue sur `./decfilm/...` et `./replay/`.

**4.4 NOTICE.** `libs/film/NOTICE.md` (EN, même forme que `libs/gamefiles/ooz/NOTICE.md`) : les trois tables embarquées
avec leurs comptes mesurés (468 identifiants et 468 lignes d'étiquettes, 58 règles `killicon` : le plan et mon premier jet
disaient 158, le fichier compte 58 lignes de règle), leur origine (en-têtes de 126 archives `.module`, identifiants
seulement, tables précalculées par le mainteneur), les films de test (octets de films de matchs du compte du mainteneur,
coupes courtes, PROVENANCE par dossier), la règle des identifiants fictifs (graine publique, 48 bits, plage, registre
`testdata/IDENTITES.tsv`, test de garde, anciens identifiants hors dépôt), les fixtures dérivées, les notes de désassemblage
(`ecs_table.tsv`), les dépendances.

**4.5 `go get` hors dépôt.** Branche poussée sur `origin` (`git push -u origin refactor/brique-film-publication`, seul push,
ni `main` ni tag) : sha `cf133667f72827f8b9d8710eb444c8ce08ec1b12` (`git ls-remote` le confirme). Module jetable
`scratchpad/goget_film/` (`go 1.26.10`), environnement `GOWORK=off GOPROXY=direct GONOSUMDB=github.com/JGtm/LevelUp
GOFLAGS=-mod=mod`. Le dépôt est lisible sans authentification (aucun `GOPRIVATE` ni `url.ssh` nécessaire).

- `go get github.com/JGtm/LevelUp/libs/film@cf133667f72827f8b9d8710eb444c8ce08ec1b12` : `added
  github.com/JGtm/LevelUp/libs/film v0.0.0-20261108145005-cf133667f728`.
- `main.go` : le corps de l'exemple de 4.3, le dossier du minifilm lu en argument (`go list -m -f '{{.Dir}}'` rend le cache de
  module `...\pkg\mod\github.com\!j!gtm\!level!up\libs\film@v0.0.0-20261108145005-cf133667f728`, le film vient de
  `replay/testdata/minifilm_000d5950` DANS ce module). `go mod tidy` puis `go build ./...` : code 0.
- `go run . <cache>/replay/testdata/minifilm_000d5950` : `93 deaths, 8 distinct victims` (identique à la sortie vérifiée de
  l'exemple du module).
- `go list -m all` : `libs/gamefiles v0.1.0` et `libs/himap v0.1.0` (les TAGS, pas les copies locales), `klauspost/compress
  v1.19.0`, `go-toml/v2 v2.4.3`, `x/sys v0.49.0`.
- `go build github.com/JGtm/LevelUp/libs/film/...` : code 0 (tous les paquets du module) ; `CGO_ENABLED=0 go build .` :
  code 0 (le README dit : pas de cgo pour construire et utiliser).

**4.6 ADR et docs.** ADR 0043 : en-tête « Amended », D-1 original marqué réécrit, ligne du vocabulaire, renvoi au bout du
paragraphe « le décodeur n'est pas dans la scission », et une section « Amendment of 2026-11-08 — the film decoder is the
fourth publishable brick » (D-1 réécrit, D-2 ordre des tags et `require`, D-3 exemple et fraîcheur, D-4 AGPL-3.0, particularités :
révisions de couche et cadre historique, build inconnu, identifiants fictifs, catalogues ; D-5 quatre dépôts, ligne
`JGtm/halo-infinite-film-decoder` / « Halo Infinite Film Decoder » et description). ADR 0041 : renvoi d'une ligne en fin de
section ; ADR 0034 : renvoi d'une ligne après l'amendement du 2026-11-08. `CLAUDE.md` : ligne « Briques Go » du tableau
(film publiable depuis 2026-11-08) et résumé de l'ADR 0043 (quatre modules, `film` AGPL-3.0, dépôt du décodeur).
`docs/ARCHITECTURE_V6.md` EN et FR : `grep -in 'non publi\|not publish'` vide, la brique film n'y est pas décrite comme non
publiée : rien à changer. `docs/CHANGELOG.md` et `docs/FR/CHANGELOG.md` (entrée 8.1.0 non publiée) disaient « film n'est pas
publiée » : une phrase corrigée dans chacun (D-L4-9, hors liste de l'item mais la même affirmation).

**Gate (2026-11-08).** `go vet ./...` code 0 et `go test ./... -count=1 -timeout 30m` dans `libs/film` : 29 paquets `ok`, 0 FAIL,
85 s ; aucun `*.golden` modifié (`git status --short | grep golden` vide). `go test ./internal/archlint/ -timeout 40m -count=1`
(`apps/go-api`, seul) : `ok`, 604 s (`commentaires_references`, `briques_imports`, `film_sans_hote`, `film_file_size` compris).
`golangci-lint run` (config d'`apps/go-api`) sur `./decfilm/...` et `./replay/` : 0 issue. `make libs-test` (vet des quatre jeux de
balises, tests, absence de `levelup/go-api` dans les dépendances) sur les quatre modules : code 0, 245 s, 39 paquets `ok`.
Le 4.5 (`go get` hors dépôt) a été joué sur le sha poussé de ce commit (voir ci-dessus).

### Résultat de l'item 4.6b (2026-11-08)

**A. D-L4-12, corpus des objectifs : mesure, aucune réécriture.** Le décodeur lit ces chunks (zlib, 8 + 8), mais ils ne
portent aucun identifiant de joueur. Mesures :

- Structure : chacun des 16 chunks décompressés (506 à 512 Ko) contient exactement deux paquets, un de type 1 puis un de
  type 2 (`grammar.WalkPackets`, 16 sur 16), aucun paquet de type 8 (population de session), aucun `chunk_00` (pas de table
  des joueurs). La recette de fabrication coupait « juste après le premier paquet TYPE_2 complet » : elle coupe donc AVANT le
  paquet de type 8 qui suit le type 2 dans un chunk de réplication.
- Témoin positif sur des chunks entiers : dans la bobine `killsource/minibobine_000d5950` (chunks 01 à 05, 640 à 860 Ko,
  même forme : paquets de type 1 et 2, puis des types 6, 8, 12...), les xuids fictifs des huit sièges (96 occurrences sur les
  cinq chunks, balayage 64 bits petit-boutiste aux huit décalages) et les gamertags (quatre noms essayés, 3 à 6 occurrences
  par chunk, UTF-16 aux huit décalages) n'apparaissent QU'à partir du premier paquet de type 8 (xuids : octet 481 431 du
  chunk 01, 485 786 du 02, 486 322 du 03, 487 911 du 04, 488 557 du 05 ; premier paquet de type 8 du chunk 01 : octet 481 427),
  jamais dans les paquets de type 1 et 2 qui le précèdent.
- Chaînes : aucune chaîne UTF-16 (petit ou gros-boutiste, aux huit décalages de bit) de quatre caractères alphanumériques ou
  plus, petit-boutiste, dans les chunks 00 à 03 de `koth_0a247154` et de `strongholds_7344d24f` (zéro occurrence, contre 16 à
  32 dans les chunks 01 et 03 de la bobine témoin) ; en gros-boutiste, 5 à 8 occurrences de quatre espaces suivis d'un chiffre,
  du bruit ; les chaînes UTF-8 de six caractères ou plus sont du bruit binaire (`UUUUUUUU`, `ZPpZP8`).
- Outil : `GAMERTAGS_CORRESPONDANCE=<noms_trois.txt> XUIDS_ORIGINAUX=<xuids_originaux.txt> GAMERTAGS_CHEMINS=libs/film/testdata/corpus_objectifs
  go test ./cmd/anonymise-gamertags/ -run TestAnonymiserFixtures` (inventaire, rien d'écrit) : 23 fichiers parcourus, 0
  occurrence, 0 xuid sur les 420 de la liste, 0 nom sur les trois. Un balayage 64 bits petit-boutiste aux huit décalages, sur
  la plage `0x0009...`, donne des milliers de candidats par film, dont une cinquantaine communs aux huit chunks : de la
  structure répétée (compteurs, tables, valeurs quantifiées, souvent à bits nuls en queue), les trois ou quatre
  candidats de la fin de chunk (après 85 %, la zone où la bobine témoin porte ses xuids) ne sont présents que dans 2 à 6 chunks
  sur 8 et l'un est `0x9fcffffffffff`, un motif de remplissage, là où les xuids de la bobine témoin sont dans tous les chunks ; l'outil ne peut pas
  chercher un xuid qu'il ne connaît pas, c'est la structure du film (type 8 absent) qui le dit.
- `minibobine_objectifs.golden` et `films_reels.golden` : texte, aucun xuid, aucun gamertag (`grep -E '[0-9]{15,}|0x0009|xuid|gamertag'` vide,
  hors identifiants courts de films `7344d24f` etc., qui sont des préfixes de match, pas de joueur) : rien à réécrire.
- Témoin avant/après : aucun octet du corpus ne bouge (`git status` : aucun fichier du dossier modifié hors `README.md`), donc
  index, temps, nombre de joueurs et de morts identiques par construction ; la liste locale (`xuids_originaux.txt`,
  `table_xuids.tsv`) et le registre `IDENTITES.tsv` n'ont reçu aucune ligne. `git grep -F -f <les 420 originaux> -- libs/film` : 0.
- Docs : `testdata/corpus_objectifs/README.md` dit en une phrase qu'aucun identifiant de joueur ne s'y trouve (renvoi à
  `../IDENTITES.tsv` pour les autres films) ; `NOTICE.md` du module réécrit sur ce point (il parlait de « paquets d'image-clé
  (type 2) » et de « gamertag lisible » : la mesure est plus précise).

Écart au texte de l'item : pas de réécriture ni de ligne ajoutée aux listes, parce qu'il n'y a rien à remplacer. Statut `[x]`
(objectif atteint, par mesure) ; le conserver tel quel ne laisse AUCUN identifiant réel dans la brique pour ce corpus.
Limite de la preuve : une absence ne se prouve pas par balayage ; elle se prouve ici par la structure (les identifiants d'un
chunk de réplication vivent dans le type 8, et ces chunks s'arrêtent avant lui).

**B. D-L4-10, sentinelles typées de la façade.** `libs/film/decfilm/decfilm.go` : `const ErrUnknownBuild = profile.ErrUnknownBuild`
et `const ErrUnknownFormat = profile.ErrUnknownFormat` (constantes plutôt que variables : un consommateur ne peut pas les
réaffecter ; même valeur, `errors.Is` vrai ; aucun équivalent n'existait dans la façade ni dans `types`). `decfilm/erreurs_typees_test.go`
(paquet `decfilm_test`, API publique seule) : un `chunk_00` de la fixture `bcb6d393` dont le build est patché en `HI_9_99_9`
puis dont la version de format est patchée à `0x7f7f`, lus par `replay.CleDeChunk0` : `errors.Is(c.Err, decfilm.ErrUnknownBuild)`
vrai pour le build (et faux pour `ErrUnknownFormat`, l'erreur nomme le build), `errors.Is(c.Err, decfilm.ErrUnknownFormat)` vrai pour le
format (et faux pour `ErrUnknownBuild`), clé connue : aucune erreur. README : deux phrases dans « Layer revisions and unknown builds ».
Ratchet : `plafondSurfaceFacade` 179 -> 181 avec ligne datée (décision du pilote du 2026-11-08), et `plafondsParFamilleFacade["profile"]`
8 -> 10 (la famille qui porte les deux renvois). `decfilm` n'est dans aucune fermeture d'imports hachée : `go test ./revision/...`
vert, cinq `Rev` inchangées, aucun `*.golden` modifié.

**Gate (2026-11-08).** `go vet ./...` et `go test ./... -count=1` dans `libs/film` : 0 échec (29 paquets, `replay` 50 s, `grammar` 112 s) ; `go test ./internal/archlint/ -timeout 40m -count=1`
(`apps/go-api`, seul) : `ok`, 241 s ; `go test ./cmd/anonymise-gamertags/...` : `ok` (`TestLesXuidsDesFixturesSontFictifs` joué, vert) ; `golangci-lint` : 0 issue (`libs/film ./decfilm/...` avec la config d'`apps/go-api`, et `./internal/archlint/...`).

### Résultat de l'item 4.7 (2026-11-08)

Ronde 1 : 2 relecteurs, 4 constats recevables (1 P1, 3 P2), 0 jeté ; 25 conditions vérifiées qui tiennent. Pas de seconde
ronde (aucun P0).

| Constat | Gravité | Où | Correction | Test : rouge, puis vert |
|---|---|---|---|---|
| C1 | P1 | `libs/film/replay/testdata/fuzz/FuzzDecodeFilmFactsFile/seed_01` | La graine « mini-bobine » portait 8 xuids réels (entiers variables du codec, invisibles au balayage de texte). Les quatre graines sont régénérées depuis la fixture fictive par `TestFuzzFilmFactsSeedsRegenerate -update-graines-faits`. Nouveau garde-rail `TestGrainesDeFaitsPortentDesXuidsFictifs` (`replay/graines_faits_identites_test.go`) : chaque graine est décodée par le codec, chaque xuid doit être dans `[0x0009000000000000, 0x000A000000000000[` ET au registre ; une graine illisible doit être le début d'une graine lisible. Côté hôte, `TestLesBinairesDeLaBriqueSontCouverts` (`cmd/anonymise-gamertags/binaires_couverts_test.go`) : tout fichier de `libs/film` ni texte ni listé dans une liste fermée et datée (graines, `inputs_*.bin.gz`, `entites_*`/`equipement_*`, `*.bin`) rougit ; une ligne de la liste qui ne couvre plus rien rougit aussi | Rouge avant : la graine à schéma 5 est illisible (le binaire lit le schéma 18) et le test la nomme, donc les 8 xuids n'étaient pas lisibles par le codec ; leur présence est établie par un balayage binaire hors dépôt des 420 originaux (décodage des graines, LE/BE aux huit décalages, entiers variables) : 8 xuids dans `seed_01`, 0 ailleurs sous `libs`. `TestLeGardeRailDesGrainesVoitUnXuidReel` prouve la détection sur un fichier de faits à xuid réel. Rouge de la liste fermée : un `.dat` et un fichier sans extension déposés sous `testdata` rougissent en étant nommés, puis retirés. Vert après : les 4 graines régénérées, balayage binaire 0, `grep` des 8 décimaux dans `libs/film` 0, `go test -run FuzzDecodeFilmFactsFile -fuzz='^$'` vert sur les graines |
| C2 | P2 | `apps/go-api/cmd/anonymise-gamertags/registre_test.go:37` et `:92` | Le ratchet lisait les seuls nombres à 16 chiffres décimaux. Il lit aussi les suites de 16 chiffres hexadécimaux (avec ou sans `0x`, casse indifférente) et la forme à octets inverses, valeur dans la plage et absente du registre (`xuidsHorsRegistreDuTexte`) ; `.mod` et `.sum` rejoignent les extensions de texte | `TestLeGardeRailVoitUnXuidHorsRegistre` étendu (hexa minuscules avec `0x`, majuscules, sans préfixe, octets inversés, plus les cas qui ne doivent pas lever : fictif au registre dans les deux formes, `deadbeefdeadbeef`, suite de 17 chiffres). Rouge avant : avec les deux appels hexadécimaux neutralisés, `attendu [0009000000000001 0009000000000002 0009000000000003 0400000000000900 2533274790395904], lu [2533274790395904]`. Vert après ; `TestLesXuidsDesFixturesSontFictifs` vert sur tout `libs/film` : aucun faux positif, aucune exclusion nécessaire |
| C3 | P2 | `replay/assaut_armement_gate_test.go:36`, `replay/assaut_bomb_arms_gate_test.go:48`, `replay/bombe_portage_gate_test.go:40`, `internal/grammar/objectif_ti11_bruit_test.go:125` | Les quatre commentaires promettaient le verrou de décodage sans le prendre : chaque test prend maintenant `decodelock.Prendre(t, cache, "<Test>", "parc")` après son saut (`internal/decodelock` est importable par `grammar`, aucun cycle), libération par le test. La phrase est donc vraie, aucune phrase retirée | `go vet ./...` vert. Preuve d'exécution : `TestBombePortageGate` avec `ASSAUT_CACHE` pointé sur un dossier vide : AVANT (ligne retirée) aucun `film_decode.lock` n'apparaît ; APRÈS le fichier `film_decode.lock` est créé sous la racine donnée (le test échoue ensuite, faute de films, ce qui est normal) |
| C4 | P2 | `packaging/systemd/levelup-worker.service:95` | `internal/games/halo_infinite/film/filmproc` remplacé par `apps/go-api/internal/filmrun` (où vit le verrou : `solo.go`, `doc.go`). Autres restes du motif `halo_infinite/film/filmproc` ou `libs/film/filmproc` hors `.ai/` et annexes d'ADR : un seul, `internal/archlint/server_stdout_internal_test.go:18`, une raison datée d'allowlist qui dit d'où vient le lanceur (historique, laissée) ; les mentions nues de `filmproc` (`filmrun/doc.go`, commentaires de tests de `filmrun`, `rev_chronique_archive.go`, CHANGELOG) sont des chroniques ou des noms de drapeaux de test, hors motif, non touchées | `git grep -n "halo_infinite/film/filmproc\|libs/film/filmproc"` : 1 ligne de production avant, 0 après (hors la raison d'allowlist historique) |

Écarts au texte de la consigne : (1) la preuve rouge de C1 ne nomme pas les 8 xuids, la graine périmée (schéma 5 contre 18)
n'étant plus décodable ; elle nomme la graine, et les 8 xuids sont établis par le balayage binaire local. (2) La liste
fermée reçoit aussi les `entites_*` / `equipement_*` (`.bin.gz` du codec, sans champ xuid, mesure du jour : 0 des 420) et
les graines `FuzzFilmRecordReaders` (payloads de records, mesure du jour : 0). Fichiers : voir le commit ; aucun `*.golden`
de révision modifié (`go test ./revision/...` vert). Gates : `go vet ./...` des deux modules ; `go test ./replay/... ./internal/grammar/...` (libs/film) ok ; `go test ./cmd/anonymise-gamertags/...` ok ; `go test ./internal/archlint/ -timeout 40m -count=1` ok en 325 s ; `golangci-lint` (config de go-api) 0 issue sur les fichiers touchés.

Ronde 2 (2026-11-08, un relecteur frais sur le seul commit `7aa5d4a6ec`) : aucun défaut recevable, 14 conditions vérifiées qui tiennent (glob des graines, plage ET registre, trois sites de xuid du codec, clause préfixe sans trou, liste fermée des binaires avec entrée périmée signalée, fuzz vert au schéma courant, 0 faux positif hexadécimal sur le dépôt, verrou pris après le skip et libéré en Cleanup, racine du cache correcte, 0 CRLF, aucun xuid réel ajouté). P0+P1 : 1 → 0, boucle close.

## Lot 5 — Préalables aux dépôts dédiés (après la clôture du desktop)

- [ ] 5.1 D-11 : `netguard.Doer` retire les en-têtes d'authentification quand une redirection change d'hôte ; test
      `httptest` (302 vers un autre hôte : en-tête absent ; même hôte : présent) ; T2 d'URL figées amendé de ce
      point seul.
- [ ] 5.2 Haiku : PUB-4-a, origine des fixtures `libs/gamefiles/{mapvar,hinavmesh}/testdata` écrite dans le
      `NOTICE.md` de `gamefiles` (après instruction par Sonnet si l'origine n'est pas dans l'historique git).
- [ ] 5.3 Gate : `make libs-test`, T1. Commit « lot 5 ».

## Lot 6 — Dépôts dédiés, les quatre ensemble (après la release sur `main`)

- [ ] 6.1 Relevé : tout ce qui nomme `libs/<brique>` dans LevelUp (archlint, CI, `go-shards.txt`, Dockerfile,
      Makefile, `dependabot.yml`, baselines, `go.work`, docs) ; liste fermée versée au plan.
- [ ] 6.2 Commandes `gh repo create` préparées (noms et descriptions d'ADR 0043 D-5 + D-8) ; création = geste du
      user (publication externe).
- [ ] 6.3 Par brique, ordre `gamefiles` → `himap` → `film`, `haloapi` à part : `git subtree split`, push, nouveau
      chemin de module et imports internes réécrits, `LICENSE`, README, CI minimale (vet, test, lint,
      govulncheck). **Film (Opus)** : prouver AVANT le push que le cadre historique de `revision` rend les
      empreintes insensibles au nouveau chemin et que 3.1 tient l'ordre ; révisions et goldens identiques.
- [ ] 6.4 Anciens chemins (D-9) : tags `libs/<brique>/v0.1.1` avec `// Deprecated` préparés ; tags `v0.1.0` des
      nouveaux dépôts préparés ; posés par le user.
- [ ] 6.5 LevelUp : `require` des nouveaux modules, imports de go-api réécrits, `libs/` et `go.work` supprimés
      (D-10), ratchets du relevé 6.1 réécrits pour lire les modules par `go list -deps` (dont
      `TestServeurNeLinkePasDeCodeGPL`), CI et Dockerfile ajustés.
- [ ] 6.6 Gate : T1 (noms identiques modulo chemin), T2, T3, T4, T7, build Docker, lint, CI verte des cinq dépôts.
- [ ] 6.7 Revue adversariale : deux relecteurs Sonnet aveugles, une ronde.
- [ ] 6.8 ADR 0043 D-5 exécutée, ADR 0041 amendée (les modules ne sont plus sous `libs/`), `CLAUDE.md`,
      `ARCHITECTURE_V6` et `COMMANDS` EN et FR ; backlog et registre ; journal ; purge des caches.

## Protocole de reprise de session

1. `git branch --show-current` et `git log --oneline -10` dans le worktree du lot ; `ListAgents`.
2. Premier item sans statut du premier lot non clos = point de reprise.
3. Vérifier sur pièces avant de coder (le desktop a pu toucher `libs/film` entre deux sessions).
4. Journal : `.ai/thought_log.md`, entrées « Fin du découplage — lot N ».

## Découvertes (notées, non traitées)

Lot 0 (2026-11-04), notées, non traitées :

- **D-L0-1** `go test ./...` de `apps/go-api` à parallélisme complet (16 paquets) fait dépasser `-timeout 10m` au paquet
  `internal/archlint` (panique « test timed out after 10m0s », 259 tests sur 487 terminés) ; seul, il passe en 286 s.
  Le T1 des gates suivants doit rejouer `archlint` à part avec `-timeout 40m`, ou le lot de CI doit le porter.
- **D-L0-2** Le ratchet `internal/domain/title/film_short_id_test.go` (`TestUneSeuleTroncatureDeMatchID`) balaie
  `internal/` seulement : une troncature de `match_id` au premier tiret existe hors de sa racine,
  `apps/go-api/cmd/levelup/cmd_backfill_killsource_selection.go:201` (`strings.IndexByte(matchID, '-')`, fonction
  `compterChunks`). Hors `libs/`, dans go-api même.
- **D-L0-3** `make replay-corpus-gate` sans `--base` compare `HEAD` à `origin/main` (v8.0.7 aujourd'hui) : 13 témoins
  sur 18 rendent PERTE ou FAUX (schéma 104 -> 106, couverture `carriedEquipment`) ; le critère « 18/19 identiques » du
  plan n'a de sens qu'avec `--base <sha du lot précédent>`.
- **D-L0-4** Un worktree dédié ne peut lancer ni `replay-equiv` (cherche `db_profiles.json`, chunks sous
  `data/cache`) ni le gate (`levelup replay-facts-export` ouvre `shared_matches_v2.duckdb` du parc) sans montage
  (jonctions + copie de la base) ; `make replay-equiv` ne conserve pas les digests (pas de `ARGS`).
- **D-L0-5** Trois chiffres de « Preuves de départ » ne se reproduisent pas : 33 goldens à l'ancien chemin (21
  relevés), 202 xuids dans `replay/testdata` (méthode inconnue ; 288 à 290 relevés), importeurs de production
  (+1 à +3 sur quatre paquets). Sans effet sur un lot.
- **D-L0-6** `libs/himap` porte 9 variables exportées modifiables, 3 fichiers de production > 500 L et 1 fonction
  > 80 L, hors des racines de `film_exported_vars`, `film_file_size` et `film_function_length`.

Lot 1 (2026-11-04), notées, non traitées :

- **D-L1-1** Les noms de paquets de l'hôte cités SANS préfixe `apps/go-api` dans les commentaires de `libs/film` ne sont pas
  couverts par la règle chiffrée du lot (qui mesurait `apps/go-api` et l'ancien chemin) : `replaybuild` dans 96 fichiers
  (67 de production), `killcollector` 50 (39), `replayartifacts` 8, `cmd/levelup` 5, `filmfetch` 4, `internal/sync/` 6,
  `internal/service/` 7, `cmd/replay-*` 25 (instruments d'usage `go run ./cmd/replay-equiv`, informatifs). Les nommer « par
  rôle » demande un jugement fichier par fichier ; à statuer avant le lot 4 (README) : lesquels expliquent un contrat et se
  gardent en prose, lesquels disparaissent.
- **D-L1-2** Registre des replis (item 1.3, `[!]`) : `Site.Fichier` et `Appelants` sont des chemins relatifs au dépôt hôte
  lus par six gardes de go-api ; 32 lignes nomment des fichiers de l'hôte. La séparation propre (chaque dépôt déclare ses
  sites, la façade `decfilm.Table()` ne rend que ceux de la brique, les gardes de l'hôte ajoutent les leurs) change le
  contrat de la façade (`Site`, `Table`, `Lire` dans `decfilm.go:98-125`) : décision à prendre au lot 2 (qui réorganise
  déjà la façade) ou à reporter, avant le tag.
- **D-L1-3** Huit littéraux de test écrits dans des goldens ou des `PROVENANCE.txt` citent encore
  `./internal/games/halo_infinite/film/…` (`golden_render_test.go:28`, `minibobine_test.go:156`,
  `frame_closure_ratchet_test.go:204`, `keyframe_closure_cuisson_ratchet_test.go:81`, `keyframe_closure_ratchet_test.go:182`,
  `marche_fermeture_test.go:36`, `minifilm_builds_test.go:261`, `minifilm_test.go:148`), plus les en-têtes des goldens (D-6) :
  à changer ensemble à la prochaine régénération volontaire de chacun.
- **D-L1-4** Références périmées DÉJÀ présentes avant le lot, laissées : commandes `go test ./internal/analysis/filmdec/`
  (3 fichiers `mesure_entete_ti9*_test.go`), `./...grammar/`, `./...profile/`, `./...source/` (7 fichiers de test et prose de
  `rev_test.go`, `revision/porte.go:8`, formes tronquées), `go run ./cmd/tmp_tagname` (`damagetag.go`, commande
  absente de l'hôte), fichiers cités mais absents (`internal/facts/killsource/world.go` dans `grammar/registry.go:315`,
  `replay/i56_capacity_episodes_research_test.go`), package nommé `filmdec` (disparu). Aucune ne casse un test.
- **D-L1-5** Hors des 9 instruments du backlog, 36 fichiers de `libs/film` portent encore un `filepath.Join(…, "data", …)` ou
  un `../../../data/titles/…` (par exemple `equipment_changes_film_research_test.go:28`, `duels_sonde_research_test.go:135`,
  `cle_du_film_recensement_research_test.go:53`) : même défaut que l'item 1.4, périmètre différent.
- **D-L1-6** Quatre tests de la brique lisent des fichiers de l'hôte par un chemin éclaté que le grep des chaînes ne voit pas :
  `internal/grammar/highlight_events_jgtm_test.go:213` (`apps/go-api/internal/sync/testdata/jgtm_full_match`),
  `replay/usage_summary_families_guard_test.go:55` (`apps/go-api/internal/games/mappings/loader_replay_labels_equipment.go`,
  `t.Fatalf` si absent), `internal/facts/objectives/families_parite_ts_test.go:52` et `replay/contract_fixtures_test.go:218`
  (`apps/web/…`). Ils se sautent hors du dépôt hôte (`racineDepot`), mais la brique en dépend : à trancher au lot 3 ou 4.
- **D-L1-7** Le ratchet `film_file_size_test.go` fige `vehicules_v13_deadstate_test.go` à 580 lignes ET refuse un plafond périmé
  (`TestPlafondsDeTailleNeSontPasPerimes`) : toute retouche de ce fichier doit se faire à nombre de lignes constant (leçon du
  premier passage de T1 du lot 1).
- **D-L1-8** Un faux xuid doit rester dans `[0x0009000000000000, 0x000A000000000000[` pour que le repérage de la table des
  joueurs le trouve (`player_table.go:129-130`) : contrainte à écrire dans l'outil de pseudonymisation des xuids si l'option A
  de l'item 1.6 est retenue (item 4.0).

Lot 2 (2026-11-04), notées, non traitées :

- **D-L2-1** `apps/go-api/internal/platform/instancelock` est une recopie à la main du verrou noyau de
  `libs/film/internal/filelock` (son en-tête le dit). Depuis l'item 2.1, go-api porte la copie jumelle
  `platform/filelock` (tenue par `copies_identiques_test.go`) : le motif existe en trois exemplaires dans l'espace de
  travail, dont deux dans go-api ; `instancelock` pourrait reposer sur `platform/filelock`.
- **D-L2-2** Les messages d'erreur du verrou de décodage disent `filmrun: ...` depuis 2.1 (ils disaient `filmproc: ...`) ;
  les drapeaux internes des tests d'enfant gardent leur nom (`-filmproc-aide-code`, `-filmproc-aide-dort`,
  `-filmproc-aide-raison`) et le test `TestAideEnfantFilmproc` le sien, pour ne pas faire disparaître de nom de T1.
- **D-L2-3** Les 19 entrées du registre de la brique marquées `CompleteeParLHote` décrivent en prose (`Fait`,
  `Mecanisme`, `CibleRetrait`) des mécanismes de l'hôte (collecteur des morts, constructeur de rejeu, règle du décor),
  et la façade exporte leurs 12 constantes `Nom*`. Une séparation complète (entrées et noms chez l'hôte) demande que le
  `Compteur` de la brique admette les noms d'un registre de l'hôte (aujourd'hui `DiagRepliHorsRegistre` en `ERROR`) et
  fait baisser `plafondSurfaceFacade` : décision à part.
- **D-L2-4** Le commentaire de `horsPerimetreDeJournal` (`archlint/film_slog_contexte_test.go:58-63`) décrit encore
  `internal/filmproc` et les signatures de `AcquireSolo`, `Release`, `LowerOwnPriority` (parties chez l'hôte) : à
  réécrire avec l'item 2.5, qui retire l'exclusion.
- **D-L2-5** `docs/adr/0034-film-decoder-profile-and-layers.md:346` cite `filmproc.AcquireSolo` (état courant) : à
  reprendre avec l'amendement de l'item 2.8. Les ADR 0008 et 0045, `docs/SYNC_GUIDE.md` (EN, FR),
  `docs/RUNBOOK_FILM_PROFILES.md` et `docs/COMMANDS.md` (EN, FR) ont été suivis par 2.1 et 2.2 ; les chroniques
  (`rev_chronique_archive.go:402`, `docs/CHANGELOG.md`, `0034-annex-history.md`) gardent l'ancien nom, c'est leur date.
- **D-L2-6** Les en-têtes des `registre_*.go` de la brique et plusieurs commentaires de `libs/film` nomment encore des
  paquets de l'hôte sans préfixe (`internal/replaybuild/`, `internal/sync/killcollector/`) : même famille que D-L1-1.
- **D-L2-7** `internal/sync/filmfetch/film_parallel_test.go:127` (`TestGetMatchFilm_ParallelDownloadFasterThanSequential`)
  mesure une durée murale (< 500 ms pour 10 × 100 ms) : sous la charge de `go test ./...` il a rendu 531 ms une fois
  (lot 2, premier passage de T1 après correction), vert 3 fois sur 3 seul. Code non touché par le lot.
- **D-L2-8** `archlint/server_stdout_internal_test.go` (ligne « oui » de l'inventaire 0.5, périmètre du lot 3.2) porte
  désormais une entrée datée pour `internal/filmrun/runner.go` : le sujet (le protocole de l'enfant écrit sur sa sortie
  standard, `EmitPeak`, `EmitRaison`, défaut `out = os.Stdout`) est entré dans la racine du garde-rail avec le lanceur.
  Retrait possible le jour où le protocole passe par un descripteur dédié ; le lot 3.2 n'a plus de racine à étendre pour
  cette ligne.

Lot 3 (2026-11-08), notées, non traitées :

- **D-L3-1** `libs/film/internal/grammar/deaths_source.go` est la QUINZIÈME source hachée en ordre historique et la seule
  sans la ligne de commentaire « L ordre de ces imports entre dans l empreinte » que `revision/cadre.go` promet aux sources
  concernées (14 sur 15 la portent) ; sa mise en page ne suit pas non plus la règle des autres (`highlightevent` avant
  `finalise`). Le test de 3.1 la fige dans `ordresFiges`. Source hachée, non touchée.
- **D-L3-2** `libs/himap/himap.CuitCarteForge` pose cinq variables de paquet exportées (`SeuilSubstitution`,
  `SeuilCouvertureCarte`, `NiveauHautNavmesh`, `MargeSolVuDuDessousCarte`) puis les rend par `defer` : deux cuissons
  concurrentes d'un même processus se marcheraient dessus. Sujet de la baseline de `film_exported_vars` ; correctif =
  passer les réglages en paramètre de la cuisson (production de `himap`, hors lot).
- **D-L3-3** Les six fichiers au-delà de 500 lignes de `himap`/`gamefiles` et `CuitCarteForge` (84 lignes) sont gelés à leur
  taille du jour dans les tables datées de `film_file_size` et `film_function_length` ; leur scission est une décision de lot.
- **D-L3-4** `no_discarded_remove` ne garde que `internal/api`, `internal/service` et les briques : 49 sites `_ = os.Remove*`
  restent dans go-api hors de ces racines (`cmd/`, `assets`, `media`, `migration`, `ops`, `persist`, `platform/*`, `transfer`,
  `replaychild`, `mapcatalog`, `filmarchive` (6), `pkg/duckdbbackup`), plus `libs/film/internal/atomicfile` (2, admis) et sa
  copie `platform/atomicfile` (2). Les étendre est une décision sur le sens du garde-rail (chemins de fond), pas sur la brique.
- **D-L3-5** Hors du périmètre `.go` de `film_sans_hote` : les commentaires de prose des `*.golden`, `bench_baseline.txt`,
  `ecs_table.README.md`, `PROVENANCE.txt`, `replay/testdata/equivalence/CORPUS.txt` et `testdata/corpus_objectifs/README.md`
  de `libs/film` citent encore `LEVELUP_UPDATE_*`, `go test ./internal/games/halo_infinite/film/...` et `apps/go-api` (commandes
  de regénération périmées). Les `*_perimetre.golden` portent aussi l'ancien nom dans leurs lignes de données (voulu : cadre
  historique). À reprendre avec le README de la brique (lot 4).
- **D-L3-6** Les planchers de balayage de `film_file_size` (1 200 fichiers) et `film_function_length` (2 300 fonctions) datent du
  2026-09-16 ; les mesures du 2026-11-08 sont 2 677 fichiers et 5 319 fonctions : un balayage à moitié muet passerait. Non relevés
  (hors lot) ; le lot 3.2 a seulement ajouté une garde « racine non vide » par racine.

Lot 4 (2026-11-08), notées, non traitées :

- **D-L4-1** La mesure 1.6 était incomplète sur trois points : son grep `25\d{14}` ignorait les xuids de préfixe 26 à 28 (9 de
  plus dans les textes) ; elle annonçait « 0 occurrence » dans les `inputs_*.bin.gz` alors que ces fixtures portent les
  identités en entiers variables du codec (1 379 xuids, invisibles au motif binaire, et table des index écrite triée par
  xuid : réécriture par décodage et réencodage) ; elle ne voyait pas les xuids qu'un commentaire cite en hexadécimal seul
  (`roster_type8.go`). Corrigés dans l'item.
- **D-L4-2** Sept xuids de tests (`2533274800000000` à `…33`, inventés) et la borne `0x0009000000000000` ne sont l'identifiant
  de personne : les remplacer aurait changé l'ordre dont des tests dépendent, et la borne comme motif binaire de 64 bits (sept
  octets presque nuls) donnait 1 546 faux positifs gros-boutiste sur un seul film. Laissés tels quels, listés au registre.
- **D-L4-3** Écart au texte de l'item : les digests de `replay-equiv` ne changent pas par construction. Le harnais retraduit les
  faits fictifs en xuids du parc avant de cuire, la cuisson voit les mêmes entrées qu'avant, les 20 références restent valables
  (MD5 identique) ; une comparaison à des références FICTIVES aurait demandé de traduire chaque valeur de chaque étape, et
  l'ordre des cuissons (tri par xuid) ne se conserve pas par traduction.
- **D-L4-4** Écart au texte de l'item : la table de correspondance n'est PAS versionnée. Original contre fictif donnerait les
  xuids réels dans le dépôt, ce que l'item vise à retirer (et le grep de contrôle exige 0). Le registre versionné ne porte que
  les fictifs ; la dérivation est publique (graine incluse) : elle empêche de LIRE un xuid d'origine dans le dépôt, pas de
  vérifier un candidat connu. Si la graine doit être secrète, elle passe en paramètre de l'outil et de `replay-equiv` (un
  changement de trois fichiers) ; décision à part.
- **D-L4-5** SOLDÉE le 2026-11-08 (complément de 4.0 : les tests lisent la fixture versionnée). Deux tests de la brique lisaient le cache de films du parc et se SAUTAIENT sans lui :
  `weaponv3.TestResolveXuidToPI_GroundTruth` et `TestResolveXuidToPIStrings_IgnoresBots`. Leurs littéraux sont désormais
  fictifs ; avec le parc monté ils échouent (le cache porte les xuids réels). Même doctrine que `TestGoldenFilms` de killsource
  et `TestGoldenInputsFidelite` depuis la v8.0.2 : ces gates locaux se jouent sur une copie traduite des films.
- **D-L4-6** Hors périmètre, notés : des xuids d'origine restent dans `apps/go-api` (104 fichiers, 57 xuids distincts, dont
  `cmd/replay-corpus-gate/testdata/gcorpus_j11_2026-09-28.json`, `internal/replayverite/testdata/bcb6d393.*`,
  `internal/replaydiff/testdata/paire_bcb6d393_54_60.json`, `internal/api/wire/testdata/film_e2e/c0a82e88/fixture.json`),
  `apps/web` (5 fichiers, 3 xuids : `replayContract.test.ts:603`, `scoreTimeline.test.ts`), `docs` (5), `config/demo/JGtm`
  (2) et `libs/haloapi` (1). Les 8 fixtures web `apps/web/src/features/match-replay/test/fixtures/go/*.json.gz` et leur
  manifeste sont, eux, régénérés par le lot (`TestContractFixturesMatchCommitted`, test de `libs/film`) ; `vitest` a été joué au
  complément du 2026-11-08 : vert (voir « Complément de l'item 4.0 »).
- **D-L4-7** Complétude : un balayage de toutes les valeurs de 64 bits de la plage dans les octets ne discrimine rien (19 864
  valeurs hors registre, jusqu'à 4 151 occurrences : des structures répétitives). Que plus aucun xuid réel ne reste dans les
  octets ne se prouve donc que par ce que le décodeur lit (tout xuid des sièges, des morts et des 8 fixtures d'entrées est
  au registre) et par l'égalité des comptes avec la mesure 1.6, non par une exclusion.
- **D-L4-8** Aucune des quatre briques ne porte un fichier `LICENSE` dans son dossier de module (les README renvoient à
  `LICENSE` et `NOTICE` de la racine). Le proxy de modules ne zippe que le dossier du module : un outil qui détecte la licence
  d'après le module (pkg.go.dev) n'en verra pas. Hors du périmètre de 4.2 (le modèle `himap` n'en a pas non plus) ; à trancher
  avec les dépôts dédiés (lot 6), où chaque module a sa racine.
- **D-L4-9** Corrigé dans le lot parce que c'est la même affirmation que l'item 4.6 : l'entrée 8.1.0 « non publiée » des deux
  `CHANGELOG` disait « `film` n'est pas publiée ». Les entrées de versions déjà sorties ne sont pas touchées.
- **D-L4-10** SOLDÉE le 2026-11-08 (item 4.6b : `decfilm.ErrUnknownBuild` et `decfilm.ErrUnknownFormat`, ratchet 179 -> 181). `profile.ErrUnknownBuild` et `ErrUnknownFormat` vivent sous `internal/` et la façade `decfilm` ne les ré-exporte
  pas (elle expose `UnknownBuildExpvarPairs`, `ErrUnknownMapBounds`, `ErrCarteAbsente`) : un consommateur externe reçoit l'erreur
  typée enveloppée mais ne peut pas écrire `errors.Is(err, ...)`. Le README décrit le comportement sans promettre le test.
  Un ré-export ajoute un symbole à une façade tenue par ratchet (`film_facade_surface_test.go`) : décision à part.
- **D-L4-11** Deux citations de `.ai/` restent dans des chaînes de production de `libs/film` : `internal/profile/profile_table.go:354`
  (champ `Preuve` d'une ligne de la table de profil, source HACHÉE : la changer bouge `profile.Rev`) et
  `internal/facts/fallback/registre_replay_equipement.go:187`. Non touchées (une révision de couche n'est pas un lot de prose).
  La citation du `README` de `testdata/corpus_objectifs` (non hachée) a été retirée par 4.2.
- **D-L4-12** SOLDÉE le 2026-11-08 (item 4.6b : mesure, ces chunks sont coupés avant le premier paquet de type 8 et ne portent aucun identifiant de joueur ; corpus conservé, rien à réécrire). `testdata/corpus_objectifs/minibobine_objectifs/` (16 chunks, environ 8 Mo, deux films du compte du mainteneur) n'est
  lu par aucun test (aucun `.go` ne cite le dossier) et n'a pas été traité par 4.0. Mesures du 2026-11-08 : aucun des 420
  xuids d'origine en 64 bits aux huit décalages, aucune chaîne UTF-16 de six caractères ou plus (donc pas de gamertag lisible).
  Cela ne PROUVE pas l'absence de xuids : les joueurs de ces deux films ne figurent pas dans la liste des 420. `NOTICE.md` le
  dit (octets d'origine, aucun gamertag lisible, aucun test). Choix à faire : les supprimer (règle 7 : corpus sans lecteur), ou les
  faire passer par l'outil après mesure des joueurs de ces deux films.
- **D-L4-13** Des commandes de régénération périmées demeurent dans les en-têtes de goldens et dans les GÉNÉRATEURS de ces
  en-têtes (`killsource/golden_render_test.go`, `minibobine_test.go`, `frame_closure_ratchet_test.go`,
  `keyframe_closure*_ratchet_test.go`, `marche_fermeture_test.go`, `revision/messages_test.go`, `perimetre_test.go`) : corriger
  l'un sans l'autre fait diverger le golden de son générateur ; D-6 (laisser jusqu'à la prochaine régénération) tient.
- **D-L4-14** (2026-11-08, revue 4.7) Les graines de `FuzzDecodeFilmFactsFile` étaient au schéma de faits 5 alors que le
  binaire lit le 18 : depuis la montée de schéma, le fuzz ne rejouait que le refus de version et n'exerçait aucune section.
  Régénérées par C1, et `TestGrainesDeFaitsPortentDesXuidsFictifs` rougit désormais toute graine devenue illisible : une
  montée de `SchemaDesFaits` oblige à régénérer les graines (`-update-graines-faits`). Règle à confirmer par le pilote ;
  non traitée au-delà de C1.
- **D-L4-15** Le motif `\.bin$` de `binairesCouverts` (`apps/go-api/cmd/anonymise-gamertags/binaires_couverts_test.go`) accepte tout `.bin` neuf de `libs/film` sans inscription : un film de test ajouté avec un xuid réel ne rougit que par l'outil d'anonymisation (item 4.0), pas par ce ratchet. Remède possible : liste nominative des dossiers de films. Non traité.
