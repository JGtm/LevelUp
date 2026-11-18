# PLAN — Fond « Relief » et coupe par étage, d'après les principes de Cartographer (2026-11-18)

Statut : **rédigé le 2026-11-18 à la demande du user (« Je suis ok avec ça, tu peux préparer un plan ») ;
décisions D-1 à D-9 proposées, à confirmer au go. Rien n'est lancé avant le go.**
Contrat d'exécution : skill `plan-execution` (ordre strict dans chaque piste, un item clos et vérifié avant
le suivant, aucune case vide à la clôture, zéro fix hors périmètre, découvertes en section 9). Grille relue :
skill `plan-review` (section 11).

Sources : la maquette comparative publiée le 2026-11-18 (https://claude.ai/artifact/CP4aEWyCzLcgFKJAmrV95N, sources
dans le dossier de travail de la session, non versionnées) ; le moteur de dessin de Den, public sous licence MIT
(`OpenSpartan/films-ui` `src/floor.ts`, `OpenSpartan/cartographer` `src/host/web/js/terrain.js`, version 1.0.1) ;
l'ADR 0048 (fonds tenus par le maillage) ; les entrées du 2026-11-18 de `.ai/thought_log.md`.

---

## 0. Objectif et critère de succès

Le user (2026-11-18) : le rendu façon Cartographer lui plaît et peut être **complémentaire** de l'actuel. Le
plan ajoute ce qui manque aujourd'hui, sans retirer ce qui existe :

- un troisième fond, **« Relief »** : le terrain vu de dessus, gris teinté d'une seule couleur, plus clair plus
  haut, ombré du nord-ouest, découpé à la zone jouable, liseré blanc ;
- une couche **« Traits du relief »**, nets à tout zoom : ruptures de hauteur, courbes de niveau sur les pentes,
  liseré ; elle se pose sur n'importe lequel des trois fonds ;
- la **coupe par étage** dans le rejeu : l'étage choisi s'affiche en plein, les étages du dessous pâles, ceux du
  dessus presque effacés, et les sols couverts par un toit réapparaissent.

Le « Plan » reste le fond par défaut (décision du 2026-11-16, inchangée).

| Lot | Ce que le user verra | Critère de succès |
|---|---|---|
| 0 | Une annexe de conception, une page de choix de teinte | Format des fichiers, seuils en unités du jeu, teinte retenue par mesure, inventaire des cartes cuisables ; validés par le user (Gate 0) |
| A | Rien à l'écran | Brique `halo-infinite-map-images` v1.4.0 taguée et publiée : relief, drapeaux et traits calculés, testés |
| B | Rien à l'écran | `mapfond-build --relief-seul` écrit le relief des 4 cartes témoins dans le calage PUBLIÉ, sans toucher aux fonds publiés |
| C | Rien à l'écran | Routes servies et testées ; contrat OpenAPI régénéré |
| D | Fond « Relief », couche « Traits du relief », coupe par étage | Maquette des commandes validée AVANT code ; rendu conforme à la maquette du 2026-11-18 sur les 4 cartes témoins ; marqueurs lisibles sous les 4 visions (garde-rail) |
| E | Le relief sur toutes les cartes qui ont un fond | 114 cartes traitées (ou écart nommé par carte), poids ajouté mesuré, ADR et docs écrits |

Cartes témoins (celles de la maquette) : Recharge (`sgh_blueprint`, intérieur sous plafond), Streets
(`sgh_streets`, étages superposés), Behemoth (`va_behemoth`, terrain naturel), Lattice – Ranked
(`1a6cfc2e-ec86-48e1-9464-1ce1bff6ed48`, Forge).

## 1. Décisions à confirmer au go

Chaque décision porte la proposition du pilote. Le go du user les confirme toutes, ou nomme celles qu'il change.

- **D-1 — place du relief.** Trois fonds au choix dans les trois vues 2D (rejeu, Tactique, Occupation du
  terrain) : Plan (défaut), Rendu, Relief. La couche « Traits du relief » est un interrupteur à part,
  allumé par défaut sur le Relief, éteint sur les deux autres. Même mécanisme de préférence que le fond
  aujourd'hui (`useMapViewPreferences`, ADR 0048 D-4).
- **D-2 — coupe par étage.** Dans le rejeu seulement (les vues Tactique et Occupation agrègent plusieurs matchs).
  Commande : « Tout » (défaut) et une pastille par étage détecté, plus un curseur de hauteur. La coupe
  vaut pour les trois fonds : sur le Plan et le Rendu, elle s'applique par le masque d'opacité du relief.
  Non retenu dans ce plan : une coupe qui suit automatiquement le joueur ouvert (idée notée en section 9,
  à reproposer après livraison).
- **D-3 — teinte du relief.** Une seule teinte de base par palette d'accessibilité, en jeton sémantique
  (`map-relief-base`), sans sélecteur de couleur pour l'utilisateur. Elle est choisie au lot 0 par mesure, contre les
  encres réelles des marqueurs (couleurs d'équipe du rejeu, contour `style.ink`) sous les quatre visions. Seuil
  proposé : écart ΔE ≥ 15 entre le noyau du marqueur et le sol, ou contour du marqueur en contraste ≥ 3:1 avec le
  sol, sous chaque vision. Constat du 2026-11-18 qui motive la mesure : un rouge `#EF4444` sur sol vert tombe à
  ΔE 2 sous protanopie ; un bleu sur sol bleu reste entre 11 et 14 dans toutes les visions. La rampe de clarté
  elle-même tient (paliers voisins à ΔE 6,1 à 7,0 sous les quatre visions, comme la rampe du Plan).
- **D-4 — où se fait le calcul.** Comme chez Den : le lourd se calcule à la cuisson, une fois, en Go, dans la
  brique `halo-infinite-map-images` (hauteurs, drapeaux, traits vectoriels, sols du maillage). Le navigateur
  teinte, coupe et dessine. Motif : trois vues consomment le relief, le site est lu sur téléphone, et la
  maquette a montré environ une seconde de calcul par carte dans le navigateur.
- **D-5 — production.** Un quatrième geste de `mapfond-build` : `--relief-seul`. Il cuit en mémoire, projette le
  relief dans le **calage du fond publié** (lu dans son sidecar), et n'écrit que les fichiers du relief et deux
  champs du sidecar. L'image du fond, l'image plan, le plan des murs et le calage ne changent pas. Constat
  qui l'impose : la recuisson de Recharge du 2026-11-18 sort un cadre de 1605 × 1736 px quand le fond publié
  fait 1605 × 1771 px.
- **D-6 — sol sans géométrie.** Là où le maillage dit « praticable » mais où la cuisson n'a aucune matière
  (terrain naturel comblé à plat dans le fond publié), la hauteur vient du maillage, et un drapeau le dit.
  Mesuré le 2026-11-18 : 32,2 % du sol de Behemoth, 8,6 % de Lattice, 4,3 % de Recharge, 0 % de Streets.
- **D-7 — cartes sans maillage** (Live Fire, Thunderhead et sa variante Heavies). Le relief est servi sans
  découpe ni coupe par étage, avec une dégradation nommée dans ses données. Le web cache alors la commande
  d'étage. Dans le Relief, le liseré suit l'emprise de la matière.
- **D-8 — version.** Le plan part de `release/v8.3.0`, sur la branche `feat/fonds-relief`. Il entre dans la
  version que le user nomme au go. Proposition : la version qui suit la 8.3.0, parce que la 8.3.0 attend
  déjà la validation à l'écran de la refonte du rejeu. S'il entre dans la 8.3.0, le plan s'inscrit au plan
  maître (section 3 : fichiers partagés avec la refonte et la 3D).
- **D-9 — crédit.** Les principes et les seuils de départ viennent de Den. Aucune ligne de son code n'est
  recopiée telle quelle : l'algorithme est réécrit en Go. Le crédit va dans le README de la brique, dans
  l'ADR et dans la note de version (« d'après OpenSpartan Cartographer, Den, licence MIT »).

## 1 bis. Orchestration : pistes parallèles et modèles

Règles (mémoire `feedback_workflows_et_sous_agents_opus`) :
- le pilote n'implémente pas ; un agent par worktree `C:/t/relief-<piste>`, créé SANS jonction vers `data/` ;
- les agents ne font ni commit ni push : le pilote relit le compte rendu sur pièces, puis commite et pousse ;
- au plus 4 agents Sonnet ou Opus d'écriture à la fois ; une seule cuisson de cartes à la fois sur la machine ;
- `model` explicite à chaque lancement ; `generated.ts` est régénéré par le pilote seul ;
- dans le dépôt de la brique, PR, squash, tag et release GitHub sont faits par le pilote (délégation du
  2026-11-13). `main` de LevelUp reste un geste du user.

| Piste | Item | Modèle | Attend | Note |
|---|---|---|---|---|
| P0 Conception | 0.1 conception | **Fable** | go | lecture seule ; rendu `annexes_fonds_relief/conception.md` |
| P0 | 0.2 inventaire des cartes cuisables | Haiku | go | liste à recopier, le pilote contrôle sur pièces |
| P0 | 0.3 mesure de la teinte | Sonnet | 0.1 | page de choix publiée ; le user choisit |
| P0 | Gate 0 | pilote + user | 0.1 à 0.3 | |
| P1 Maquette | D.1 maquette des commandes | Sonnet | go | aucun code ; choix du user |
| P2 Brique | A.1 relief et drapeaux | Sonnet | Gate 0 | dépôt `halo-infinite-map-images` |
| P2 | A.2 traits vectoriels | **Opus** | A.1 | motif : algorithme géométrique (marche des carrés, chaînage, simplification, coupe par étage) |
| P2 | A.3 sols du maillage | Sonnet | A.1 | |
| P2 | A.4 tests, README, exemple | Sonnet | A.2, A.3 | |
| P2 | A.5 PR, tag v1.4.0, release | pilote | A.4 | puis Gate A |
| P3 Cuisson | B.1 geste `--relief-seul` | Sonnet | Gate A | |
| P3 | B.2 WebP du relief | Haiku | B.1 | modèle existant (`mapfond-webp`) |
| P3 | B.3 cuisson des 4 témoins | pilote | B.2 | dossier de sortie temporaire, copie par le pilote ; puis Gate B |
| P4 Service | C.1 à C.3 contrat, service, routes | Sonnet | Gate A (C.1), B.1 (C.2) | puis Gate C |
| P5 Web | D.2 `lib/replay/relief` | Sonnet | Gate 0, C.1 | |
| P5 | D.3 rejeu | Sonnet | D.1 validé, D.2, Gate C | |
| P5 | D.4 Tactique et Occupation | Sonnet | D.3 | |
| P5 | D.5 jetons et garde-rail des couleurs | Sonnet | Gate 0 | puis Gate D |
| P6 Parc | E.1 cuisson des 114 cartes | pilote | Gate D, **accord du user** | une carte à la fois |
| P6 | E.2 ADR 0049, docs, note de version | Sonnet | Gate B | |
| P7 Clôture | R.1 revue finale | **Fable** | Gates A à E | une seule ronde |
| P7 | R.2 corrections retenues | Sonnet ; Opus si l'écart est un algorithme | R.1 | |

Vagues : dès le go, 0.1, 0.2 et D.1 ; après Gate 0, A.1 puis A.2 et A.3 en parallèle, D.5 ; après Gate A, B.1
et C.1 ; après Gate C, D.2 à D.4 ; après Gate D et l'accord du user, E.1 ; à la fin, R.1 et R.2.

## 2. Ce qui existe (relu sur pièces le 2026-11-18, `release/v8.3.0` `728a3965f6`)

- **Brique `halo-infinite-map-images` v1.3.1** (`himap`) : `Rendu` porte un z-buffer et ses normales ;
  `Rendu.Altitude(i, j)` et `Rendu.Eclairement(i, j)` sont exportées. Calage : `CalageFond`, `CalageDuRendu`,
  convention `xMonde = originX + (px + 0,5) × mpp ; yMonde = originY − (py + 0,5) × mpp`. Maillage :
  `ComposantesRetenues` (faces gardées), `PlanDesMurs`, `PlanPNG`, `PlanRogneAuCadre`. Le relief exporté par
  la maquette sortait de `Rendu.Altitude` et des faces de `Retenues.Garde`.
- **Cuisson** : `cmd/mapfond-build` a trois gestes (cuire, `--plan-seul`, rapport TSV), une table des options
  par geste (`options.go`, `TestChaqueOptionSeRattacheAUnGeste`), une écriture du sidecar par
  `internal/mapfondsidecar` (ADR 0048 D-8). `cmd/mapfond-webp` convertit en WebP sans perte (`nativewebp`,
  VP8L) et refuse un aller-retour qui n'est pas identique au bit près.
- **Fonds publiés** : `data/titles/halo_infinite/reference/map_backgrounds/`, 339 fichiers versionnés (45 Mo) :
  114 sidecars, 114 fonds, 111 images plan.
- **Service et routes** : `service/replay_map_background_plan.go` (`MapBackgroundPlanImage`,
  `…ForMap`), `handlers/background_variante.go` (`?variante=plan`), routes
  `/players/{p}/matches/{id}/replay/background.png` et `/players/{p}/tactical/{map}/background.png`. Contrat :
  `domain/replaydoc/map.go` (`MapBackground`, `MapBackgroundPlan`, `PlanImage`), projeté par
  `replayview.MapBackgroundOf`.
- **Web** : `lib/replay/mapBackgroundVariant.ts` (variantes `rendu` | `plan`, défaut `plan`,
  `isWallPlanDrawn`), `lib/replay/wallPlan.ts` et `wallPlanPaint.ts` (un seul trait, 1 px, opacité 0,6),
  `layers/mapBackground.ts`, `mapBackgroundDraw.ts`, `useReplayStaticLayers.ts`, `usePlanClipMask.ts`,
  `settings/useMapViewPreferences.ts` ; vues Tactique (`TacticalPiluleFond.tsx`,
  `useTacticalFondVariante.ts`, `TacticalPlanCard.tsx`, `TacticalReapparitionsPlan.tsx`) et Occupation
  (`match-view/MatchPositionsHeatmap.tsx`).
- **Marqueurs** : `layers/replayMarkers.ts` pose déjà un contour sous le noyau (`style.ink`, `OUTLINE_PAD`,
  `OUTLINE_ALPHA`) et un langage d'étage par anneaux (`floorRings.ts`, `floorOf`). La mesure de D-3 part de
  ces encres réelles, pas des jetons `team-ally` / `team-enemy` de la maquette.
- **Accessibilité** : `lib/accessibility/colorDistance.ts` (`oklab`, `deltaE`, `simulateCvd`), modèle de
  garde-rail `impactRoleTokens.test.ts`.
- **Unités** : les hauteurs du fond et du maillage concordent (écart médian 0 sur les 4 témoins). Un étage
  fait environ 2 unités (Recharge, Streets). Les seuils de Den, en pieds, ne se recopient pas : ceux de la
  maquette (rupture 0,35 ; équidistance 0,5 ; bande d'étage ± 0,7, fondu 0,25 ; toit 1,0 ; marge 0,5 ;
  fermeture 0,5) sont le point de départ du lot 0.

## 3. Lot 0 — conception et mesures

- [ ] 0.1 Conception par un agent **Fable**, en lecture seule. Rendu : `.ai/annexes_fonds_relief/conception.md`,
      qui fixe :
      - le format des fichiers par carte. Proposition à éprouver : `<clé>.relief.webp` (R, G = hauteur du
        dessus sur 16 bits, B = drapeaux, A = 0 ou 255 seulement, parce qu'une alpha intermédiaire se
        prémultiplie dans le canevas) et `<clé>.relief.json` (bornes de hauteur, pas de quantification,
        étages détectés, traits en polylignes delta-encodées au dixième de pixel avec leurs hauteurs basse
        et haute, faces gardées du maillage, dégradations) ; poids mesuré sur Behemoth, la plus lourde ;
      - les drapeaux : sur la carte, sol tiré du maillage (D-6), toit au-dessus d'un sol, et ce qu'il faut de
        plus pour la coupe ;
      - l'algorithme des traits : marche des carrés, chaînage, simplification, tolérance ; séparation entre
        ruptures et courbes ; liseré ;
      - les seuils en unités du jeu, mesurés sur les 4 témoins ;
      - la projection dans le calage publié (D-5), et le cas d'un cadre recuit plus petit ou décalé ;
      - les pièges : prémultiplication, gestion des couleurs à la lecture (`colorSpaceConversion: 'none'`),
        orientation par quarts de tour (ADR 0048 D-4).
- [ ] 0.2 Inventaire, par carte du catalogue, de ce que la cuisson locale peut produire : carte native installée,
      fichiers Forge présents, maillage présent. Rendu : `.ai/annexes_fonds_relief/inventaire.md`. Une carte non
      cuisable est nommée, avec la raison.
- [ ] 0.3 Mesure de la teinte (D-3) : candidates (les 7 préréglages de Den et le gris du thème) contre les
      encres réelles des marqueurs et des objectifs, sous les quatre visions et dans les deux thèmes, avec
      `colorDistance.ts`. Table en annexe ; page de choix publiée (artefact `verifs`) avec les 4 témoins.
- **Gate 0 :** annexes écrites ; format, seuils et teinte validés par le user. Aucun code de brique avant.

## 4. Lot A — brique `halo-infinite-map-images` v1.4.0

Dépôt dédié `github.com/JGtm/halo-infinite-map-images`, développement par `scripts/briques-workspace.sh` et
`GOWORK` (CLAUDE.md, « Briques »). Ajout d'API seulement : version mineure, aucun changement cassant.

- [ ] A.1 `himap` : relief d'un `Rendu` dans un `CalageFond` donné, avec la hauteur du dessus et les drapeaux. Le
      sol sans matière prend la hauteur du maillage (D-6). Aucune constante magique : seuils nommés,
      documentés en unités du jeu.
- [ ] A.2 Traits vectoriels (agent **Opus**) : ruptures, courbes de niveau sur pente, liseré de la zone gardée ;
      polylignes chaînées et simplifiées, chacune avec ses hauteurs basse et haute (la coupe par étage les
      filtre côté web).
- [ ] A.3 Sols du maillage : faces gardées dans le repère du calage ; étages détectés (pics de l'histogramme
      des hauteurs de sol, écart minimal tiré de 0.1).
- [ ] A.4 Tests sur grilles synthétiques : marche, rampe, toit sur un sol, trou enclos, sol sans matière. Un test
      `*_gamefiles_test.go` sous le tag `gamefiles` sur Recharge. README de la brique (section et crédit,
      D-9) et `example_test.go`.
- [ ] A.5 PR dans le dépôt de la brique, gates verts, squash, tag `v1.4.0`, release GitHub
      (`gh release create v1.4.0 -R JGtm/halo-infinite-map-images --title v1.4.0 --generate-notes --verify-tag`).
      Dans LevelUp, `require` monté (`go get …@v1.4.0`, `make tidy`).
- **Gate A :** tests de la brique verts (`go test ./...` du dépôt, et `-tags gamefiles` sur le poste du
  user) ; tag et release posés ; `go build ./...` de LevelUp vert sur le nouveau `require`.

## 5. Lot B — cuisson

- [ ] B.1 `mapfond-build --relief-seul` (D-5) : une ligne dans la table des gestes de `options.go` et ses
      validations ; lecture du calage publié ; écriture du relief et des champs `reliefImage` et `reliefData`
      du sidecar par `internal/mapfondsidecar` seul. Les autres champs restent identiques à l'octet
      près (test). Journal `slog` par carte : poids écrit, part du sol tirée du maillage, dégradations.
- [ ] B.2 `mapfond-webp -convertir` prend aussi l'image du relief (aller-retour identique au bit près,
      règle existante).
- [ ] B.3 Cuisson des 4 témoins par le pilote dans un dossier temporaire, puis copie dans
      `map_backgrounds/` ; diff : seuls les fichiers du relief et deux champs par sidecar changent.
- Docs : section `mapfond-build` de `docs/COMMANDS.md`, en anglais et en français dans le même commit (règle 15).
- **Gate B :** `go test ./cmd/mapfond-build/... ./cmd/mapfond-webp/... ./internal/mapfondsidecar/...` vert ;
  `git diff --stat` de `map_backgrounds/` limité aux 4 témoins ; images de fond et plan inchangées
  (`git diff --quiet -- '*.webp' ':!*.relief.webp'`).

## 6. Lot C — contrat et routes

- [ ] C.1 Contrat : `replaydoc.MapBackground` gagne `ReliefImage` et `ReliefData` (noms de fichiers, vides
      sans relief), et `mapfondsidecar` les lit et les écrit. Règle 18 : chaque champ a son lecteur, le web au lot D.
- [ ] C.2 Service : `MapBackgroundReliefImage`, `MapBackgroundReliefData` et leurs jumelles `…ForMap`,
      sur le modèle de `replay_map_background_plan.go` : même résolution de clé, mêmes gardes sur le nom de
      fichier, `port.ErrMapBackgroundNotAvailable` sans relief. Interfaces dans `port/`.
- [ ] C.3 Handlers sans logique : `?variante=relief` sur les deux routes d'image (le refus de variante
      inconnue reste), et une route sœur pour les données (`…/background.relief.json` sur les deux préfixes), avec
      le même cache HTTP que l'image plan. `openapi.yaml` mis à jour ; `generated.ts` régénéré par le pilote.
- Pas de nouvelle capacité : l'absence de relief se traite comme l'absence d'image plan (404), comme le veut l'ADR 0048.
- **Gate C :** `go test ./internal/service/... ./internal/api/...` vert, avec des tests httptest pour : relief
  présent, absent, nom de fichier hostile, variante inconnue. `go test ./internal/archlint/...` vert.
  `make generate-types` sans diff après la régénération.

## 7. Lot D — web

- [ ] D.1 Maquette des commandes, sans code : trois fonds, interrupteur des traits, commande d'étage. Elle est
      dessinée DANS les cadres validés de la refonte du rejeu (Standard, Cinéma, Plein cadre), plus la pilule
      de fond de la Tactique. Le user la valide. Les fichiers de la maquette sont joints au brief de D.3 et D.4.
- [ ] D.2 `lib/replay/relief/` : lecture de l'image (sans gestion des couleurs ni prémultiplication), teinte par
      table depuis le jeton, coupe par étage (sols empilés tirés des faces, dans un Web Worker), chemins des
      traits filtrés par la coupe, cache par carte. Logique pure testée (vitest) : décodage, coupe, filtre des
      traits, toit sur sol.
- [ ] D.3 Rejeu : variante `relief` dans `mapBackgroundVariant.ts` (le type, la normalisation de la préférence
      et `resolveBackgroundVariant` qui retombe sur le Plan sans relief), dessin dans `mapBackgroundDraw.ts`,
      couche des traits dans `useReplayStaticLayers.ts`, commande d'étage conforme à D.1, état sans maillage
      (D-7). La coupe passe aussi au Plan et au Rendu (D-2).
- [ ] D.4 Tactique et Occupation : la variante Relief et l'interrupteur des traits, sans coupe.
- [ ] D.5 Jetons `map-relief-*` dans chaque palette (`semantic-tokens.ts`, `palettes/`), valeurs de 0.3. Garde-rail
      `lib/accessibility/mapReliefTokens.test.ts` : seuil de D-3 contre les encres des marqueurs, par palette,
      sous les quatre visions, dans les deux thèmes.
- i18n : chaque libellé neuf dans chaque langue de `config/locales.toml` (fr et en), dans les dictionnaires
  de feature. « Relief », « Traits du relief », « Étage », « Tout ».
- **Gate D :** `make check-types` (cache `node_modules/.tmp` purgé avant), `npm run lint`,
  `node tools/lint-cross-feature-imports.mjs`, `node tools/lint-no-hardcoded-colors.mjs`, vitest ciblé par les
  agents. Vitest complet lancé UNE fois par le pilote. Contrôle à l'écran par le user sur les 4 témoins, dans les
  trois vues, contre la maquette du 2026-11-18 et D.1.

## 8. Lot E — parc, documentation

- [ ] E.1 Après l'accord du user : `--relief-seul` sur toutes les cartes de l'inventaire 0.2, une carte à la
      fois, puis `mapfond-webp -convertir`. Poids ajouté mesuré et reporté (base : 45 Mo aujourd'hui). Chaque
      carte non traitée est nommée avec sa raison.
- [ ] E.2 ADR 0049 (en anglais) : fond Relief, format des fichiers, calcul à la cuisson, coupe par étage,
      sol tiré du maillage, cartes sans maillage, crédit. Elle amende 0048 (D-3 : troisième fond ; D-8 : deux
      champs neufs du sidecar). Mise à jour de la liste des ADR dans `CLAUDE.md`, d'une ligne dans la note de
      version et dans le journal des modifications.
- **Gate E :** inventaire soldé (traité ou nommé) ; `git diff --stat` de `map_backgrounds/` limité aux
  fichiers du relief et aux sidecars ; ADR relue par le pilote.

## 8 bis. Clôture

- [ ] R.1 Revue finale adverse (agent **Fable**, skill `adversarial-review`) du diff complet de la branche et de
      celui de la brique. Une seule ronde.
- [ ] R.2 Corrections retenues (P0 et P1). Les P2 vont au backlog.
- [ ] R.3 `make gate-push` vert, avec `LEVELUP_REPO_ROOT` sur le dépôt principal ; CI de la branche lue
      (`gh run list --branch feat/fonds-relief`) après chaque poussée ; entrée au `.ai/thought_log.md` ;
      `.ai/REGISTRE_REPORTS.md` alimenté. Branches d'agents supprimées dès leur intégration.

## 9. Découvertes

Consignées ici, jamais traitées dans ce plan.

- 2026-11-18 : Den a un plan de Live Fire, lu depuis le tag `pfnd` (son `navmesh.rs`). L'ADR 0048 classe Live Fire
  « aucun maillage dans le module `any` ». Il peut y avoir là un maillage que notre lecture manque. À instruire
  à part : sans maillage, Live Fire n'a ni plan ni coupe (D-7).
- 2026-11-18 : Den coupe le plan d'une carte selon le mode joué (volumes mortels et de sécurité, points
  d'apparition : `libfilmshell` `NavMesh::rings_within`). Nous avons un plan par carte.
- 2026-11-18 : le fond publié comble à plat le sol sans géométrie (un tiers de Behemoth). Le relief le
  corrige pour son propre fond, pas pour le Rendu.
- Idée non retenue (D-2) : une coupe qui suit l'étage du joueur ouvert dans le rejeu.
- Lien possible avec le plan 3D : l'image des hauteurs pourrait servir au sol du lecteur 3D. Ce n'est pas un
  item de ce plan.

## 10. Reprise de session

Lire, dans l'ordre : la dernière entrée de `.ai/thought_log.md` qui cite ce plan, les cases de ce fichier,
`.ai/annexes_fonds_relief/`, puis `git log --oneline -10` de `feat/fonds-relief` et du dépôt de la brique.
Reprendre au premier item non coché de chaque piste dont la dépendance est close. Vérifier avec `ListAgents`
qu'aucune session ne tient déjà un worktree `C:/t/relief-*`.

## 11. Grille `plan-review` (relue le 2026-11-18)

- Structure : objectif et critère par lot, lots ordonnés, gates écrits, branche nommée. Effort : moyen à
  lourd (brique et web), environ six agents d'écriture en tout.
- Couches Go : le calcul pur vit dans la brique, l'orchestration de cuisson dans `cmd/`, la lecture dans
  `service/` derrière `port/`, les handlers sans logique ; aucune lecture DuckDB. `PathResolver` :
  `MapBackgroundDir` / `MapBackgroundMetaPath` existants, sans aucun chemin écrit à la main.
- Multi-titre : les fonds sont propres à Halo Infinite ; leur absence se traite comme aujourd'hui, sans
  capacité nouvelle ni comparaison de slug.
- Tests : à chaque couche (brique, cuisson, service, handlers, logique web, garde-rail des couleurs).
- Journalisation : `slog` structuré dans la cuisson et le service ; aucune erreur avalée.
- Web : aucune route neuve, aucune query key inline (clés dans `lib/query/keys.ts`), libellés fr et en,
  couleurs par jetons.
- Exécutabilité : périmètre fermé par item, gates en commandes, statuts `[x]` / `[~]` / `[!]`, ordre par piste,
  découvertes en section 9, reprise en section 10. Les décisions D-1 à D-9 sont à confirmer au go, avant toute
  exécution.
