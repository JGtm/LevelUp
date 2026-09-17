# PLAN — Arme favorite dans le briefing Explorer (2026-09-17)

> Date : 2026-09-17. Cadrage validé par l'utilisateur en conversation le 2026-09-17
> (empilement sous « Par contexte », 1 ou 2 armes, jamais d'ajout de colonne, coût de
> hauteur anticipé avant le rendu). **Amendé le 2026-09-17 après revue `plan-review`**
> (dix points tranchés avec l'utilisateur : contrat OpenAPI, compilation i18n, gardes de la
> grille, définition du dénominateur, XUID, approximation du helper, piège DP-3, fichier
> propre, outil de mesure, assertion du log) et précision de l'utilisateur sur la
> DÉFINITION de l'arme favorite (D0).
> Branche d'exécution : `wt/arme-favorite-briefing`, worktree dédié
> `../LevelUp-wt-arme-favorite` créé depuis `feat/v75` (le worktree principal est PARTAGÉ
> entre agents, ne jamais y coder).
> Clôture : commits sur `wt/arme-favorite-briefing`, fusion dans `feat/v75` sur signal de
> l'utilisateur.
> Contrat d'exécution : skill `plan-execution` (ordre strict, aucun report, tout item
> statué, zéro fix hors périmètre). Ce fichier est la source de vérité de l'avancement.

## Objectif et critère de succès

Ajouter au bandeau de briefing de l'Explorer (mode Matchs, `include_briefing=true`) un bloc
« Arme favorite » : la ou les deux armes ayant produit **le plus de frags** du joueur sur
le scope filtré, avec le dénominateur de ce qui a été mesuré.

**Critère de succès** : le bloc s'affiche sous « Par contexte » sans changer la hauteur de
la rangée quand une carte de dimension est bien remplie ; il prend la place de « Par
contexte » quand ce bloc est absent ; il ne crée JAMAIS de sixième cellule ; il se réduit à
une ligne compacte (et une seule ligne de rangée en plus) quand toutes les cellules sont
courtes ; il s'affiche sur Halo Infinite (source film, note de couverture visible) comme sur
Halo 5 (source native, note quasi jamais visible) ; il disparaît proprement quand aucun frag
n'est mesuré. Gates Go et web verts, CI de branche verte au niveau job, gate visuel de
l'utilisateur passé.

## Ce qui existe et se réutilise (vérifié sur pièces le 2026-09-17)

| Besoin | Existant | Fichier |
|---|---|---|
| Frags par arme sur un lot de matchs | `port.WeaponKillsRepository.LoadWeaponKillsAggregated` (filtres `MatchIDs` + `XUIDs`) | `internal/port/weapon_kills.go` |
| Choix du lecteur PAR TITRE, sans slug | `ServiceRegistry.weaponKillsRepoFor` (film si `film.kill_source` + classificateur, sinon `weapon_kills` natif) | `internal/api/wire/registry_pages.go:468` |
| Top N armes trié (kills desc, départage libellé) + filtre « libellé résolu, hors sentinelles » | `buildTopWeaponKills` | `internal/service/synthesis_service_builders.go:226` |
| Type de ligne d'arme sérialisé | `domain.SynthesisWeaponKillEntry` (Label, Kills, Class, Role) | `internal/domain/synthesis.go:164` |
| Assemblage du briefing (in-memory sur raw rows) | `buildExplorerBriefing` | `internal/service/match_history_service_briefing.go:60` |
| Injection de dépendance par `WithX` sur ce service | `WithRankedCapable`, `WithPlayerMatchesRepo(repo, slug, gamertag)` | `internal/api/wire/registry_pages_home.go:82` |
| Identité du joueur au câblage | `pdb.XUID` | `internal/api/wire/registry_pages_home.go` (`MatchHistoryCtx`) |
| Gabarit visuel « liste d'armes » (libellé + frags + barre par classe) | `TopArmes` + `fragClassColor` | `features/explorer/ExplorerTargetSampleStats.tsx:150` |
| Précédent « module du briefing dans son fichier » | `ExplorerRankedBlock.tsx` | `features/explorer/` |
| Helpers purs du briefing (+ tests) | `ExplorerBriefing.logic.ts` | `features/explorer/ExplorerBriefing.logic.ts` |
| Grille « Par… », ses gardes de rendu | `ExplorerBriefingModules.tsx:110-121` | `features/explorer/` |
| Garde-rails filtrant sur `/briefing/i` | terminologie + `deltaToken` | `explorerBriefingTerminology.guard.test.ts`, `explorerDeltaToken.guard.test.ts` |
| Contrat OpenAPI GÉNÉRÉ (Huma + fragment manuel) | `make openapi-gen`, `make openapi-check` | `Makefile:115-128` |
| Manifestes i18n compilés | `node apps/web/scripts/build_i18n_manifests.mjs` (aucune cible npm/make) | `apps/web/scripts/` |
| Types front = ré-export du contrat OpenAPI | `components['schemas'][...]` | `lib/api/types.ts:911` |

**Mesures de cadrage (base locale, 2026-09-17)** — couverture de la source d'arme :
Infinite 2026 = 948/949 matchs, 2025 = 412/415, 2024 = 15/54, 2023 = 9/536 ;
Halo 5 = 2 754/3 032 matchs (550 926 lignes natives). Coût d'agrégation mesuré :
0,22 s sur TOUT l'historique du joueur sans filtre de matchs.

## Décisions tranchées AVANT exécution (fermes, ne pas re-décider)

**D0 — Définition.** L'arme favorite est l'arme qui a produit **le plus de frags crédités
au joueur** sur le scope (source de dégât du film sur Infinite, arme native du kill sur
Halo 5). Ce n'est NI l'arme la plus tenue, NI la plus tirée (`film.weapon_shots` est une
autre famille de données, hors sujet). Tri : frags décroissants, départage par libellé
(`buildTopWeaponKills`).

**D1 — Placement.** Le bloc est empilé SOUS « Par contexte », dans la MÊME cellule de la
grille « Par… » (wrapper `space-y-2`). Quand `context_split` est absent, le bloc devient
une cellule propre de cette grille. Dans les deux cas le nombre de cellules reste
identique : **jamais de sixième cellule**, aucun autre bloc déplacé, DP-3 non rouvert.
Les DEUX gardes de rendu de la grille (`dimensions.length > 0 || hasContextOrRanked` et
l'early-return) incluent `weapons != null` : un briefing dont le seul module est l'arme
favorite peint la grille avec cette unique cellule.

**D2 — Contenu.** 1 ou 2 armes, JAMAIS 3. Gabarit = celui de `TopArmes` (libellé tronqué +
frags alignés + barre fine colorée par `fragClassColor`). Pas d'icône d'arme (ni ce registre
visuel ni le payload ne la portent). Une grenade PEUT être l'arme favorite — c'est déjà le
cas dans le top armes de la Synthèse ; on garde la cohérence entre surfaces.

**D3 — La hauteur est décidée AVANT le rendu, jamais mesurée.** Le nombre de lignes de
chaque cellule est une donnée connue du composant : carte de dimension = `entries.length`
(1..6, `selectTopFlop(…, 3)`), « Par contexte » = 2, Classement = `kinds.length` (1..3)
**seulement s'il est réellement affiché** (`showRanked`, qui dépend de la capability
`ranked` que seul le composant connaît). Le helper pur reçoit donc des COMPTES, pas le
briefing :

```
favoriteWeaponSlots({ dimensionLines: number[], rankedLines: number }): 0 | 1 | 2
lignesRangee = max(...dimensionLines, rankedLines, 2)
placeLibre   = lignesRangee - 2
slots        = clamp(placeLibre, 0, 2)
```

- `slots = 2` → deux armes avec barre — la rangée ne bouge pas ;
- `slots = 1` → une arme avec barre — la rangée ne bouge pas ;
- `slots = 0` → **forme compacte** : une seule ligne (libellé + frags, sans barre) — la
  rangée gagne UNE ligne, jamais plus.

Le calcul choisit la FORME, jamais la présence. Il prédit en lignes LOGIQUES ; le gate
visuel valide les hauteurs PHYSIQUES (les lignes du Classement sont `flex-wrap`, un
titre porte une infobulle). Un écart constaté se corrige DANS LA FORMULE (une constante,
par exemple compter une ligne de Classement pour 1,5) — jamais par une mesure du DOM
(`ResizeObserver`, `getBoundingClientRect`…). En cas de doute, la forme compacte est le
repli sûr : elle coûte au plus une ligne.

**D4 — Source.** `port.WeaponKillsRepository` obtenu par `weaponKillsRepoFor(pdb)` — les
deux implémentations (film Infinite / natif Halo 5) derrière le même port, aucune
comparaison de slug. Le service reçoit le repo ET le xuid :
`WithWeaponKillsRepo(repo port.WeaponKillsRepository, xuid string)` (câblage avec
`pdb.XUID`). Filtres : `MatchIDs` = matchs du scope filtré, `XUIDs: []string{xuid}`
(filtre direct sur la colonne xuid des deux lecteurs — jamais `Gamertag`, qui passe par
une jointure `xuid_aliases` rendant zéro ligne en silence si l'alias manque),
`ResolveRoles: true` (nécessaire à `Class` pour la couleur), `IncludeGrenadeMelee: false`
(sinon Halo 5 remonte des lignes sentinelles grenade/mêlée qui ne sont pas des armes),
`MinKills: 0`.

**D5 — La couverture s'exprime en FRAGS, pas en matchs.** Le port rend des lignes agrégées
par arme : il ne dit pas combien de matchs ont contribué, et l'ajout d'un compte de matchs
changerait la signature du port et ses deux implémentations. Le bloc porte :
- `measured_kills` = somme des frags des lignes **retenues par le même filtre que le top**
  (`Label != "" && !IsGrenadeMelee`, celui de `buildTopWeaponKills`), AVANT troncature au
  top 2 — « mesuré » signifie « ce qu'on saurait nommer », le dénominateur reste cohérent
  avec ce qui est affiché ;
- `scope_kills` = somme des `r.Kills` des raw rows du scope (`domain/match_history.go:39`,
  déjà en mémoire, aucune requête).
La note « N frags mesurés sur M » s'affiche si et seulement si `measured < scope`. Si
`measured >= scope` (source film qui crédite autant ou plus que l'API) : pas de note.

**D6 — Un seul critère d'omission : `measured_kills == 0`.** Pas de plancher arbitraire :
un plancher cacherait l'information au lieu de la qualifier, et la note du dénominateur dit
exactement ce qui est mesuré (scope 2023 : « 62 frags mesurés sur 4 210 »). Le bloc hérite
par ailleurs du seuil d'échantillon existant — `buildExplorerBriefing` sort avant les
modules quand `LowSample` est vrai. Aucun nouveau seuil nommé.

**D7 — Libellé d'arme.** Résolution FR-first avec repli EN, telle que le résolveur existant
la fait (`resolveWeaponKeyDimensions` / `weapon_name_labels`). Aucune locale nouvelle à
câbler, aucune décision à prendre.

**D8 — Best-effort strict.** Toute erreur du repo → `slog` puis bloc nil.
`games.ErrCapabilityNotSupported` → `DebugContext` (titre sans source d'arme, cas légitime) ;
toute autre erreur → `WarnContext` (même doctrine que `loadWeaponKillRows`). Le briefing et
la page ne échouent JAMAIS à cause de ce bloc.

**D9 — Hors périmètre, définitivement.** Pas de top 3, pas d'icône, pas de tuile dans le
socle, pas de carte « grenade collée », pas de renommage « Dépositaire » (sujet distinct,
décidé mais non inclus ici), aucun changement de DP-3 ni des blocs existants, aucun script
npm/make nouveau pour la compilation i18n (Découvertes).

**D10 — Fichier propre côté web.** Le bloc vit dans `ExplorerBriefingWeapons.tsx` dès le
premier commit (précédent `ExplorerRankedBlock.tsx`), nommé *Briefing* pour rester sous
les deux garde-rails qui filtrent sur ce motif. Il est une liste `flex flex-col` — JAMAIS
une grille à colonnes nommées (`[grid-template-columns:…]`), car le test DP-3 cible la
DERNIÈRE grille portant cette classe (`ExplorerBriefingStrip.test.tsx:100`) et viserait
la mauvaise.

## Étapes

### Étape 1 — Backend : type, builder, câblage

- [x] `internal/domain/explorer_briefing.go` : type `ExplorerBriefingWeapons`
      (`Entries []SynthesisWeaponKillEntry` (2 max), `MeasuredKills int`, `ScopeKills int`)
      + champ `Weapons *ExplorerBriefingWeapons \`json:"weapons,omitempty"\`` sur
      `ExplorerBriefing`, documenté comme les autres blocs (nil = module non émis).
- [x] Nouveau fichier `internal/service/match_history_service_briefing_weapons.go`
      (le fichier briefing principal fait 490 lignes — ne pas l'alourdir) portant
      `buildBriefingWeapons(ctx, repo, titleSlug, xuid, filtered) *domain.ExplorerBriefingWeapons`.
      Les `MatchIDs` et `scope_kills` se lisent dans les raw rows déjà en mémoire
      (`filtered[].MatchID`, `filtered[].Kills` — `domain/match_history.go:11` et `:39`) :
      aucune requête supplémentaire hors celle du repo d'armes. Filtre des lignes,
      `measured_kills`, tri et troncature au top 2 selon D5 (`buildTopWeaponKills`, même
      package ; le filtre est appliqué UNE fois et partagé entre le dénominateur et le top).
- [x] `buildExplorerBriefing` appelle le builder APRÈS `ContextSplit`, avant `Streaks`.
- [x] `MatchHistoryService` : champs `weaponKillsRepo port.WeaponKillsRepository` +
      `weaponKillsXUID string` ; méthode `WithWeaponKillsRepo(repo, xuid)` sur le modèle de
      `WithPlayerMatchesRepo` (le repo et l'identité voyagent ensemble).
- [x] `registry_pages_home.go` (`MatchHistoryCtx`) :
      `svc = svc.WithWeaponKillsRepo(r.weaponKillsRepoFor(pdb), pdb.XUID)` avec commentaire
      renvoyant à `weaponKillsRepoFor` (même factory que Synthesis, Explorer-cible et
      Sessions — pas de second chemin de lecture).
- [x] Logging conforme D8.

**Gate 1** (depuis `apps/go-api/`) :
```
go build ./... && go vet ./... && go test ./internal/service/... ./internal/domain/...
```

### Étape 2 — Tests Go du builder

- [x] Deux armes remontées, `measured < scope` → bloc avec 2 entrées max, `MeasuredKills`
      = somme des lignes retenues (une ligne à libellé vide dans le jeu de test NE compte
      PAS, D5).
- [x] `measured_kills == 0` (aucune ligne, ou uniquement des lignes à libellé vide) → bloc
      nil (D6).
- [x] Repo nil / `ErrCapabilityNotSupported` → bloc nil, aucune erreur propagée (D8).
- [x] Erreur inattendue du repo → bloc nil (assertion sur le résultat nil UNIQUEMENT ; le
      `WarnContext` se vérifie à la relecture du diff, pas de capteur `slog`).
- [x] Filtres passés au repo factice : `XUIDs == [xuid]`, `Gamertag == ""`,
      `IncludeGrenadeMelee == false`, `ResolveRoles == true` — garde-fou de D4.
- [x] `LowSample` → bloc absent (hérité, vérifié par un test de `buildExplorerBriefing`).

**Gate 2** : `go test ./internal/service/... -run Briefing -v` vert, et
`go test ./...` sans régression.

### Étape 3 — Contrat OpenAPI et types front

- [x] `make openapi-gen` régénère `apps/go-api/api/openapi.yaml` (document Huma + fragment
      manuel ; le fragment n'est PAS touché, le champ se dérive de la struct Go). Ne jamais
      éditer `openapi.yaml` à la main.
- [x] `make generate-types` régénère `apps/web/src/lib/api/generated.ts`.
- [x] `lib/api/types.ts` : ré-export `ExplorerBriefingWeapons` depuis `components['schemas']`
      (jamais de mirror manuel).

**Gate 3** : `make openapi-check` (aucune dérive — aucun job CI ne la vérifie, ce gate
local est le seul filet) PUIS `make check-types` après purge de
`apps/web/node_modules/.tmp` (faux vert incrémental documenté dans `delivery-checklist`).

### Étape 4 — Front : helper pur + son test

- [x] `features/explorer/ExplorerBriefing.logic.ts` :
      `favoriteWeaponSlots({ dimensionLines, rankedLines }): 0 | 1 | 2` exactement selon la
      formule D3, sans aucune dépendance React/DOM ni lecture du briefing brut.
- [x] `ExplorerBriefing.logic.test.ts` : `[6,3,2]`/0 → 2 ; `[3]`/0 → 1 ; `[2,2]`/0 → 0 ;
      `[]`/3 → 1 ; `[]`/0 → 0 ; `[6]`/3 → 2 (le max l'emporte).

**Gate 4** : `npx vitest run src/features/explorer/ExplorerBriefing.logic.test.ts`
(hors sandbox — cf. mémoire `reference_vitest_outside_sandbox`).

### Étape 5 — Front : rendu et i18n

- [ ] Nouveau fichier `features/explorer/ExplorerBriefingWeapons.tsx` (D10) : composant
      `FavoriteWeaponBlock` à trois formes selon `slots` (D3), gabarit `TopArmes` (D2),
      liste `flex flex-col` (jamais de grille à colonnes nommées), note de couverture en
      `text-3xs text-muted-foreground` affichée si et seulement si
      `measured_kills < scope_kills` (D5).
- [ ] `ExplorerBriefingModules.tsx` : calcule `dimensionLines` et `rankedLines`
      (`showRanked ? kinds.length : 0`), appelle le helper, monte le bloc : empilé sous
      `ContextSplitCard` dans la même cellule quand `context_split` existe ; cellule propre
      sinon (D1).
- [ ] `ExplorerBriefingModules.tsx` : les DEUX gardes de rendu incluent `weapons != null`
      (D1).
- [ ] `lib/i18n/manifests/explorer.toml` : titre du bloc + note paramétrée `{n}`/`{m}`,
      FR ET EN, FR sans anglicisme (« Arme favorite » / « Favorite weapon » ;
      « {n} frags mesurés sur {m} » / « {n} of {m} kills measured »).
- [ ] `node apps/web/scripts/build_i18n_manifests.mjs` — régénère `generated/explorer.ts`
      (aucune cible npm/make ne le fait ; sans cet item les clés n'existent pas côté TS).
- [ ] Couleurs : `fragClassColor` uniquement, aucun hex ni classe Tailwind couleur.

**Gate 5** : `make check-types`, `make test-web`, `npm run lint` (depuis `apps/web/`) verts ;
`ExplorerBriefingStrip.test.tsx` — dont le test DP-3, INTOUCHÉ — vert.

### Étape 6 — Tests de rendu

Dans `ExplorerBriefingWeapons.test.tsx` (nouveau) :
- [ ] `slots = 2` → deux lignes avec barre ; `slots = 1` → une ligne avec barre ;
      `slots = 0` → forme compacte sans barre.
- [ ] Note de couverture présente quand `measured < scope`, absente quand
      `measured == scope` (cas Halo 5) ET quand `measured > scope`.

Dans `ExplorerBriefingStrip.test.tsx` (à côté du describe DP-3, qui reste tel quel) :
- [ ] Dimensions pleines + `context_split` + `weapons` → le bloc est DANS la cellule
      « Par contexte » (assertion de parenté, miroir du test DP-3).
- [ ] `weapons` sans `context_split` → le bloc est un enfant DIRECT de la grille.
- [ ] `weapons` SEUL (ni dimension, ni contexte, ni classé) → la grille est rendue avec une
      cellule (gardes D1).
- [ ] `weapons` absent → aucune trace du bloc, grille inchangée.

**Gate 6** : `make test-web` vert.

### Étape 7 — Mesure de coût et gates de livraison

- [ ] Mesurer avec le CLI duckdb en `READ_ONLY` sur la base partagée, forme exacte de la
      requête du lecteur du titre avec `IN (SELECT match_id FROM match_participants WHERE
      xuid = ?)` à la place des paramètres liés (ordre de grandeur, c'est ce qu'on cherche),
      sur le PLUS GRAND scope (tout l'historique) ; consigner le chiffre dans le thought
      log (référence : 0,22 s sans filtre de matchs). Au-delà de ~300 ms : Découvertes +
      remontée à l'utilisateur, AUCUNE optimisation dans ce lot.
- [ ] `go test ./...` puis `go vet ./...` (le diff ne touche ni persist/ ni sync/ ni
      migration/ → tag `integration` non requis ; le noter explicitement à la clôture).
- [ ] `make gate-push`.
- [ ] CI de branche verte AU NIVEAU JOB (`gh run list --branch wt/arme-favorite-briefing`).

### Étape 8 — Gate visuel utilisateur (5 écrans)

L'utilisateur nomme les témoins ; les captures sont faites par lui, jamais par l'agent.
Un écart de hauteur observé se traite par la formule D3, pas par une mesure.

- [ ] Scope complet Infinite : 2 armes, rangée de hauteur inchangée.
- [ ] Scope filtré sur une seule carte : forme compacte, +1 ligne maximum.
- [ ] Scope ancien (2023) : note de couverture explicite et lisible.
- [ ] Scope sans « Par contexte » : bloc en cellule propre, toujours 5 cellules au plus.
- [ ] Halo 5 : bloc nominal, aucune note de couverture, aucune ligne sentinelle
      grenade/mêlée.

### Étape 9 — Clôture

- [ ] Entrée `.ai/thought_log.md` (date, titre, statut, décision technique, résultats
      mesurés dont le chiffre de l'étape 7, prochaine étape).
- [ ] Tout item du plan statué `[x]` / `[~]` / `[!]`.
- [ ] Reports éventuels inscrits dans `.ai/V7.5/REGISTRE_REPORTS.md` avec leur condition
      de reprise.
- [ ] Fusion dans `feat/v75` sur signal explicite de l'utilisateur (mode branche unique).

## Découvertes (consigner, NE PAS traiter)

- 2026-09-17 — Le libellé FR « Dépositaire » de la carte hijacks de la Synthèse est le nom
  de la médaille *Reclaimer* (« s'emparer d'un véhicule ennemi qui vous appartenait »), pas
  celui du détournement générique que compte la carte. Renommage en « Détournements » pour
  tous les titres décidé par l'utilisateur le 2026-09-17, HORS de ce lot.
- 2026-09-17 — Ce bloc introduit la PREMIÈRE requête DuckDB dans un briefing jusqu'ici
  entièrement calculé en mémoire sur les raw rows. À surveiller si d'autres modules suivent.
- 2026-09-17 — La compilation des manifestes i18n (`build_i18n_manifests.mjs`) n'a ni
  cible npm ni cible make : chaque lot qui touche un `.toml` doit y penser. Un script
  `npm run build-i18n` (et son appel dans le gate web) éviterait l'oubli — hors périmètre.
- 2026-09-17 — Aucun job CI ne vérifie la dérive de `openapi.yaml` (`make openapi-check`
  n'est joué qu'en local) ; spectral et les tests YAML ne détectent pas un champ manquant.

## Journal d'exécution

- **Étape 0 (2026-09-17)** — `npm install` dans `apps/web` (508 paquets, sortie 0). Baseline
  AVANT toute modification : `make check-types` sortie 0, `go build ./...` sortie 0,
  `go vet ./...` sortie 0. Baseline verte.
- **Étape 1 (2026-09-17)** — Gate 1 (`go build ./... && go vet ./... && go test
  ./internal/service/... ./internal/domain/...`) sortie 0. Commit `b358fc8c7`.
- **Étape 2 (2026-09-17)** — Gate 2a (`go test ./internal/service/... -run Briefing -v`)
  sortie 0, 6 nouveaux tests verts. Gate 2b (`go test ./...`) : sortie 1 avec UN SEUL échec,
  `TestOpenAPIYAMLIsUpToDate` — le contrat `openapi.yaml` ne porte pas encore le champ
  `weapons` ajouté à l'étape 1 (« relancer `make openapi-gen` », ligne 12831). C'est
  exactement le premier item de l'étape 3 ; le plan ordonne la réparation APRÈS l'étape 2.
  Aucune autre régression : tous les autres paquets `ok`. Re-vérifié vert après l'étape 3
  (voir ci-dessous) et à l'étape 7.
  Incident d'outillage sans effet sur le verdict : un premier `go test ./...` lancé en
  avant-plan a été tué au bout de 10 min (limite d'attente de l'outillage) ; le run rejoué en
  arrière-plan vers un log persistant est allé au bout (ligne `GOTEST_EXIT=1`).

- **Étape 3 (2026-09-17)** — `make openapi-gen` (740 095 octets écrits, +22 lignes :
  `weapons` sur `ExplorerBriefing` + schéma `ExplorerBriefingWeapons`), `make generate-types`
  (+8 lignes dans `generated.ts`), ré-export dans `lib/api/types.ts`. Gate 3 :
  `make openapi-check` sortie 0 (document à jour ET `generated.ts` dérivé),
  `make check-types` sortie 0 après purge de `apps/web/node_modules/.tmp`. L'échec unique du
  gate 2b est levé : `go test ./internal/api/ -run TestOpenAPIYAMLIsUpToDate` sortie 0.

- **Étape 4 (2026-09-17)** — `favoriteWeaponSlots` ajouté à `ExplorerBriefing.logic.ts`
  (plancher de 2 lignes = hauteur de « Par contexte », place libre plafonnée à 2). Gate 4 :
  `npx vitest run src/features/explorer/ExplorerBriefing.logic.test.ts` sortie 0, 4 tests.

## Protocole de reprise de session

1. Lire ce fichier : les cases cochées font foi.
2. `git log --oneline -10` sur `wt/arme-favorite-briefing`.
3. Reprendre à la PREMIÈRE étape dont le gate n'est pas passé — jamais plus loin.
