# Plan — Callouts decoupes sur le decor reel, et carte de chaleur du rejeu

> Ecrit le 2026-08-16. Priorite utilisateur : « pour les callouts et la heatmap c'est la
> priorite apres les equipements ». Deux items du cahier des charges Notion (REPLAY 2D) :
> **9** « decouper les zones de callout sur le decor reel » (+ 9.2 « ajuster la zone des
> callouts sur les bordures exterieures ») et **13** « proposer un mode heatmap pour voir les
> lieux les plus chauds [...] et les plus froids aussi ». Branche `feat/v75`, worktree
> principal (le decoupage exige les fichiers du jeu ; la heatmap est du web pur — voir la
> section « parallelisme »). Contrat `plan-execution`.

## Ce qui est acquis — ne pas re-chercher

| acquis | ou | consequence |
|---|---|---|
| Catalogue de callouts : 816 zones / 22 cartes natives, prismes (polygone + top/bottom), FR/EN officiels, `provenance` = `brut` sauf Ridgeline = `decoupe` (parts/holes) | `replay/callouts_catalog.go`, `data/.../reference/map_callouts.json`, `cmd/mapcallouts-build` | la mecanique de rendu accepte DEJA parts + holes (pair-impair) ; seule la PRODUCTION des contours decoupes manque |
| Le decoupage de Ridgeline a ete fait UNE fois a partir d'un dump `.ai/V7.5/dumps/callout_zones_ridgeline_clipped.json` (chaine du POC) — pas par la chaine cartes | `mapcallouts-build -decoupe` | c'est un point de comparaison (oracle faible) pour le decoupage universel |
| Le sol praticable existe PAR CARTE : le raster de cuisson `himap.Rendu` (cellules a `z > -inf`, `Cell` = 0,092 m/px, coquille `sddt`, eau) et sa projection publiee : `map_backgrounds/{cle}.png` — **alpha 0 = pas de matiere** — + sidecar de calage (`metersPerPixel`, origine, Y descendant) | `himap/rendu.go`, `himap/fond_png.go`, `replay/map_background.go` | l'union des emprises praticables est un MASQUE RASTER deja publie et versionne : le decoupage peut se faire sur le PNG (offline-pur, sans le jeu) |
| Zones fines = etages imbriques ; le rendu ne les remplit pas ; le libelle s'ecrit une fois sur la zone la plus haute ; affectation d'un joueur en 3D | `web/calloutsLayer.ts` (spec POC) | le decoupage doit se faire PAR ETAGE (top/bottom du prisme) : une zone haute ne se rogne pas par le sol du dessous |
| Le user a signale que les zones « depassent sur le grand fond transparent, zone inaccessible sans mourir » (9.2) | Notion | c'est exactement le masque alpha du fond |
| Composants graphiques existants : `Heatmap2DChart` (ECharts, `components/charts`), `heatmapColors.ts` + garde-rail | web | ne pas reinventer une echelle de couleur ; le rejeu, lui, dessine au canvas — le calque heatmap sera un CALQUE CANVAS qui reutilise `heatmapColors` (tokens), pas un ECharts pose sur la carte |
| Donnees du document : `tracks` (positions par vie, 100 ms), `kills` (fil des morts, tueur + victime + positions relues), `shots`, `grenades` | `replayNormalize.ts` | tout ce qu'il faut pour « chaud/froid » est DEJA cote client |

## Decisions tranchees avant execution

1. **Le decoupage se fait sur le MASQUE PUBLIE (PNG alpha + sidecar), pas en relancant la
   cuisson** : offline-pur, reproductible en CI, et il suit automatiquement les cartes deja
   publiees (21 natives + Forge). Le raster `himap.Rendu` reste la source de verite de la
   cuisson ; le PNG est sa projection versionnee. Si une carte n'a pas de fond publie, sa
   provenance reste `brut` — jamais un decoupage devine.
2. **Le brut est CONSERVE a cote du decoupe** (regle Notion 9) : `provenance` dit lequel est
   servi ; le polygone designer reste dans le catalogue.
3. **Par etage** : le masque a une seule altitude par cellule (z-buffer du dessus) ; on
   decoupe une zone avec le masque seulement si l'altitude praticable de la cellule tombe
   dans `[bottom, top]` du prisme (tolerance ecrite, mesuree sur Ridgeline). Une zone dont
   l'etage est SOUS un toit (masque = le toit) n'est PAS rognee par lui — elle garde son
   brut la ou le masque ne la voit pas.
4. **Oracle faible du decoupage** : Ridgeline decoupee par le POC — la nouvelle chaine doit
   rendre un accord (IoU par zone) eleve avec elle. Le seuil est fixe AVANT la mesure : IoU
   median >= 0,85 sur les 11 grandes zones. Et l'oracle des positions jouees : la part des
   positions de joueur qui tombent DANS une zone decoupee ne doit pas baisser de plus de
   2 points par rapport au brut (un decoupage trop agressif rognerait des zones accessibles —
   Notion 9 le dit).
5. **Heatmap = un calque du canvas, deux modes** : DENSITE DE PRESENCE (positions des tracks,
   ponderees par le temps) et DENSITE D'ELIMINATIONS (positions des morts, victime ; option
   tueur). Le « froid » est ce que la palette montre en bas d'echelle — pas un troisieme
   calcul. Rendu par CELLULES du raster de fond (meme grille, meme calage), noyau gaussien
   discret, echelle par quantiles (pas par max, qu'un seul point ecraserait). Bascule dans le
   TIROIR de reglages (livre le 16/08), avec un choix presence / eliminations, et EXCLUSIF
   avec le calque zones ? NON — cumulable, la heatmap se pose SOUS les zones et les
   trajectoires, opacite bornee.
6. **Palette** : `heatmapColors.ts` (tokens sequentiels, garde-rail existant), rendue en canvas
   par lecture des tokens resolus (patron `readInk` / `fxInk`). Zero hex dans `features/`.
7. **La heatmap est calculee UNE fois par document** (patron `buildShotFx`), pas par frame ;
   elle est STATIQUE sur tout le match par defaut (« ou ca chauffe dans ce match »), avec
   une option « jusqu'au curseur » si le cout le permet — a mesurer, pas a promettre.

## Phases

### Phase A — DECOUPER les callouts (Go, exige `data/` pour l'oracle ; le decoupage lui-meme
### ne lit que des fichiers versionnes)

- [x] A.1 Lecteur du masque praticable depuis `map_backgrounds/{cle}.png` + sidecar
      (`replay/map_background.go` a deja `MondeVersPixel`) : `PraticableAt(x, y) bool` +
      altitude si le PNG la porte (sinon : le sidecar ou le raster ; a verifier sur pieces —
      si l'altitude n'est pas publiee, la decision n°3 se degrade en « pas de test d'etage »
      et cela s'ECRIT, avec la mesure de son cout sur Ridgeline).
      → `internal/mapdecoupe/masque.go`. **L'ALTITUDE N'EST PAS PUBLIEE** (verifie sur
      pieces) : le PNG est un RGBA d'habillage — au style de production `jeu` la couleur est
      `TeinteNiveauDeJeu(dz, eclairement)`, deux inconnues melees sur 8 bits, que l'arete
      (division par 3) et l'eau ecrasent ; le sidecar ne publie qu'UNE altitude pour toute la
      carte (`stats.playLevelZ`). Le test d'etage de la decision n°3 est donc IMPOSSIBLE.
      Degradation ECRITE en tete du paquet, et elle va dans le bon sens : le masque dit
      « cette colonne porte de la matiere » sans dire a quelle hauteur, donc le rognage ne
      retire que les cellules vides a TOUTE altitude — une zone haute au-dessus de l'arene
      comme une zone basse sous un toit gardent leur emprise. Cout mesure en A.4.
- [x] A.2 Decoupage polygone x masque : rasteriser le prisme sur la grille du fond,
      intersecter avec le masque (etage compris), re-vectoriser (contours + trous, parts
      multiples) — sortie au format DEJA accepte (`Parts`/`Holes`). Simplification de
      contour tolerante (< 1 cellule).
      → `decoupe.go` (balayage par lignes sur la grille EXACTE du fond, aucun
      re-echantillonnage), `contours.go` (chainage des aretes interieur-a-droite, signe de
      l'aire pour separer contours et trous, regle de virage pour les pincements
      diagonaux), `simplifie.go` (RDP a 0,08 m = 0,87 cellule). Sortie arrondie au
      centimetre — un neuvieme de cellule (catalogue 2,78 Mo -> 2,06 Mo).
      **AJOUT non prevu par le plan, et c'est le coeur de l'item 9.2** : `reprendLesEnclaves`
      ne retire que le vide qui COMMUNIQUE avec le dehors de la zone. Un vide entoure de
      decor est un trou de reconstruction, pas un debordement. Sans regle ni reglage, et
      mesure : +10 points de positions gardees sur btb_highpower.
- [x] A.3 `cmd/mapcallouts-build` produit `decoupe` pour TOUTE carte qui a un fond publie,
      `brut` sinon ; le brut reste stocke. Bordures exterieures (9.2) : le masque les coupe
      par construction.
      → `cmd/mapcallouts-build/decoupe_masque.go`. **19 cartes `decoupe`** (18 par la chaine
      universelle + Ridgeline, qui GARDE le dump du POC — decoupage par etage, strictement
      meilleur, et c'est ce qui en fait un oracle et pas un miroir), **3 `brut`** faute de
      fond publie : `academy_tutorial`, `pve_house`, `sgh_interlock`.
      Le brut est conserve au niveau CATALOGUE (`MapCalloutsCatalog.Brut`, 625 zones sur
      18 cartes) et NON dans la zone servie : `MapCalloutsEntry` est la charge utile OpenAPI,
      y mettre le brut l'enverrait sur le reseau a chaque match. **Contrat inchange**,
      `make openapi-check` vert, aucune regeneration web.
- [x] A.4 MESURES (instrument versionne, garde d'environnement) : IoU par zone contre le
      decoupe POC de Ridgeline ; part des positions jouees dans une zone (brut vs decoupe)
      sur >= 5 films de >= 3 cartes ; nombre de zones qui DISPARAISSENT ou tombent sous 1 m2
      (a lister — chacune est un cas a regarder, pas a ecraser).
      → `internal/mapdecoupe/oracle*_test.go`, hors ligne pur, `t.Skip` si le catalogue, les
      fonds, le dump ou les documents de rejeu manquent. Chiffres au journal ci-dessous.
- [x] A.5 Golden du catalogue mis a jour, `callouts_catalog` tests, contrat inchange (le
      format existait).
      → `TestCatalogueCalloutsLivreEstExploitable` fige desormais la REGLE et non une liste :
      une carte est `decoupe` SI ET SEULEMENT SI son fond est publie (verifie sur l'arbre).
      `verifieBrutConserve` interdit qu'un decoupage perde son pave d'origine.

**Gate A** : IoU median >= 0,85 sur Ridgeline, perte de positions <= 2 points, liste des zones
degeneres publiee, tests verts. Sinon : ajuster la TOLERANCE d'etage ou la simplification —
jamais le seuil — ou statuer `[!]` avec la mesure.

**GATE A PASSE le 2026-08-16**, seuils inchanges : IoU median **0,872** (>= 0,85) ; pire perte
de positions **-1,10 point** sur 7 films de 7 cartes (<= 2) ; 3 zones degenerees listees.

## Journal de la phase A (2026-08-16)

**Le premier verdict etait un ECHEC, et il a designe la vraie cause.** A tolerance nulle :
IoU median 0,809 et jusqu'a **-40,47 points** de positions jouees sur `btb_highpower`. La
colonne de diagnostic ajoutee a l'instrument (« part des positions posees sur du decor
publie ») a tranche : elle valait 60,11 % quand le decoupe en gardait 59,55 %. Le decoupage
ne rognait pas trop — **le fond publie n'a pas de matiere sous 40 % des positions jouees**.
Ce n'est pas un defaut de cette chaine : la cuisson ecarte les instances au maillage grossier
(`himap.AireMaxTriangleJouable`), mesure a 10,8 % de la carte validee manquants sur
Cliffhanger, et ce qui manque manque par INSTANCE ENTIERE — une rampe, une dalle, un segment
de passerelle.

**La tolerance a ete etalonnee, pas choisie** (`TestOracleTolerance`, 7 films / 7 cartes) :

| rayon | IoU median | pire perte de positions | aire retiree (btb_highpower / Ridgeline) |
|---|---|---|---|
| 0,00 | 0,809 | -40,47 pt (btb_highpower) | 62,9 % / 30,6 % |
| 1,00 | 0,816 | -17,53 pt (btb_highpower) | 54,0 % / 28,0 % |
| 2,00 | 0,819 | -7,39 pt (ctf_breaker) | 49,4 % / 26,5 % |
| 3,00 | 0,865 | -2,58 pt (ctf_breaker) | 48,1 % / 24,5 % |
| **4,00** | **0,872** | **-1,09 pt (ctf_breaker)** | **47,7 % / 22,9 %** |
| 5,00 | — | -0,16 pt | chasm s'effondre a 7,3 % de rognage |

4,00 m est le PREMIER rayon qui passe les deux oracles sur tout l'echantillon. Il n'ajoute que
3,4 points de matiere au cadre de Ridgeline (18,6 % -> 22,0 %), et le rognage de Ridgeline
tombe a 22,9 % — l'ordre de grandeur exact du decoupage par etage du POC (21 %).

**A.4 — IoU par zone (Ridgeline, 11 grandes zones, tolerance 4,00 m)**

| zone | vol | POC | IoU | aire brute | gardee | gardee POC |
|---|---|---|---|---|---|---|
| Antenna | 11 | decoupe | 0,781 | 700,6 m2 | 54,5 % | 68,9 % |
| Platform | 12 | decoupe | 0,868 | 862,9 m2 | 80,3 % | 90,7 % |
| Sanctuary | 13 | decoupe | 0,872 | 903,1 m2 | 84,2 % | 89,3 % |
| Pipes | 15 | brut | 0,996 | 137,6 m2 | 100,0 % | 100,0 % |
| Creek | 16 | decoupe | 0,992 | 207,1 m2 | 100,0 % | 100,1 % |
| Deck | 17 | decoupe | 0,991 | 179,0 m2 | 100,0 % | 99,7 % |
| Radiator | 18 | decoupe | 0,661 | 434,5 m2 | 56,7 % | 61,1 % |
| Tanks | 19 | decoupe | 0,968 | 191,6 m2 | 100,0 % | 97,4 % |
| Enclave | 20 | decoupe | 0,501 | 504,8 m2 | 97,6 % | 49,1 % |
| Icicles | 21 | decoupe | 0,994 | 144,5 m2 | 100,0 % | 100,1 % |
| Cliffside Trail | 22 | decoupe | 0,747 | 760,1 m2 | 51,5 % | 63,6 % |

Mediane **0,872**. Les deux desaccords qui restent ont un SENS lisible : `Enclave` (0,501) est
la seule zone ou la chaine universelle garde beaucoup plus que le POC (97,6 % contre 49,1 %) —
sans altitude, elle ne sait pas que l'etage de la zone s'arrete ; `Radiator` et
`Cliffside Trail` sont l'inverse, elle rogne un peu plus que les emprises AABB de
`map_structure`, qui sont des boites englobantes genereuses.

**A.4 — Positions jouees, brut contre decoupe (7 films, 7 cartes)**

| film | carte | points | brut | decoupe | ecart | sur matiere |
|---|---|---|---|---|---|---|
| 04023f8a | btb_highpower | 23 285 | 99,29 % | 99,29 % | +0,00 pt | 99,51 % |
| 01e1f945 | catalyst | 29 553 | 95,90 % | 95,91 % | +0,00 pt | 100,00 % |
| 606d9844 | chasm | 10 915 | 98,16 % | 98,16 % | +0,00 pt | 100,00 % |
| e94163af | ctf_bazaar | 10 457 | 94,93 % | 94,93 % | +0,00 pt | 100,00 % |
| 145908d1 | ctf_breaker | 30 224 | 100,00 % | 98,90 % | -1,10 pt | 98,93 % |
| 000d5950 | ridgeline | 29 221 | 100,00 % | 99,96 % | -0,04 pt | 100,00 % |
| b8d1fe0c | sgh_blueprint | 10 196 | 99,97 % | 99,97 % | +0,00 pt | 100,00 % |

La reconnaissance de carte est GEOMETRIQUE et hors ligne (le document de rejeu ne nomme pas sa
carte, et un instrument n'ouvre pas de base) : sol joue publie par le fond, tranche verticale
des prismes, puis **couverture mutuelle** — la carte contient-elle le match ET le match a-t-il
visite ses grandes zones. La couverture mutuelle est ce qui a debloque l'echantillon (3 films
-> 7) : sans elle, une grande carte contient une petite arene tout entiere et marque 100 %.
Une erreur d'attribution ne peut faire ECHOUER le gate, jamais le faire passer.

**A.4 — Zones degenerees : 3 sur 628** (elles gardent leur pave brut, aucune n'est ecrasee)

| carte | zone | vol | aire brute -> decoupee |
|---|---|---|---|
| btb_engine | Triple Sandwich jr | 26 | 13,7 m2 -> 0,0 m2 |
| btb_fragmentation | Cliff's Thinking Spot | 95 | 0,5 m2 -> 0,1 m2 |
| sgh_streets | Oscar's House | 37 | 0,5 m2 -> 0,5 m2 |

Une seule DISPARAIT vraiment (`Triple Sandwich jr`, entierement au-dessus du vide) ; les deux
autres sont des paves deja plus petits que le plancher de 1 m2 — pas des echecs de masque.

**Cout de production** : 625 zones decoupees, 29 055 sommets publies (46 par zone), catalogue
750 Ko -> 2,06 Mo. Charge utile par match : la plus grosse carte (`btb_fragmentation`,
5 726 sommets) reste tres en dessous du document de rejeu lui-meme (1 a 6 Mo).

## Decouvertes (NON traitees — regle 7 du contrat d'execution)

1. **`MapBackgroundCalibration.MondeVersPixel` tronque au lieu d'arrondir vers le bas**
   (`int()` sur une valeur negative tronque vers zero) : un point situe jusqu'a une cellule a
   GAUCHE ou AU-DESSUS du cadre est rendu comme le pixel 0 au lieu d'etre declare hors cadre.
   Effet negligeable ici (le cadre est le voisinage des ancres, les positions jouees sont loin
   du bord), mais c'est une erreur de conversion dans le SEUL depositaire de la formule.
2. **Quatre fonds publies perdent des ancres** — `btb_highpower` 38/51, `btb_exiled` 30/35,
   `btb_fragmentation` 36/46, `forest` 14/18. C'est l'oracle faible du chantier cartes, et
   c'est exactement ce qui a impose une tolerance de 4 m : la qualite du decoupage est bornee
   par la completude du fond, carte par carte.
3. **`btb_drydock` est la seule carte dont la part gardee mediane s'ecarte de 100 %** (75,3 %) :
   ses paves debordent beaucoup, ou son fond est incomplet. A regarder au gate visuel.
4. **`Trous` est desormais quasi inutilise par la chaine universelle** : avec la reprise des
   enclaves, un pave simple ne peut plus produire d'evidement. Les 15 trous du catalogue
   viennent tous du dump du POC (Ridgeline). Le champ reste donc VIVANT, mais si Ridgeline
   passait un jour a la chaine universelle, `Holes` deviendrait du code mort cote web.

### Phase B — HEATMAP (web pur, parallelisable sur un worktree frere)

- [ ] B.1 `heatmapLayer.ts` : logique pure testee — accumulation par cellule (presence
      ponderee par la duree entre deux points ; eliminations aux positions des victimes),
      lissage gaussien discret (rayon en metres, pas en pixels), normalisation par quantiles
      (p50/p95), sortie = grille + echelle.
- [ ] B.2 Rendu canvas : SOUS le fond ? NON — sur le fond, sous les zones et les traces,
      opacite max 0,55, cellules aux dimensions du calage du fond ; palette
      `heatmapColors` lue par tokens resolus ; legende discrete (froid -> chaud) dans le coin
      de la carte, FR/EN.
- [ ] B.3 Reglages dans le TIROIR : bascule « Carte de chaleur », choix « presence /
      eliminations », persistes par `replayPreferences` (le helper existe). Libelles FR sans
      anglicismes (« carte de chaleur », pas « heatmap », a l'ecran).
- [ ] B.4 `prefers-reduced-motion` : la heatmap est statique, rien a faire ; mais le mode
      « jusqu'au curseur », s'il existe, ne doit pas repeindre a chaque frame plus vite que
      la lecture — mesurer le cout (rAF budget) avant de l'activer.
- [ ] B.5 Tests : accumulation (un point isole -> une bosse ; deux points -> deux bosses ;
      quantiles insensibles a un point extreme), rendu (garde-rail zero hex), tiroir.

**Gate B** : gates web complets (purge `.tmp`, typecheck, lint, vitest exit 0), zero hex ;
gate VISUEL utilisateur sur `000d5950` (arene dense) et `64e8adfa` (Catalyst CTF).

## Parallelisme

- Phase A sur le worktree PRINCIPAL (Go ; l'oracle des positions lit `data/`).
- Phase B sur un worktree FRERE (`LevelUp-wt-heatmap`, branche `wt/heatmap`) : web pur, les
  fixtures versionnees suffisent (`apps/web/src/features/match-replay/__fixtures__` ou le
  golden inputs). L'agent B ne modifie NI le journal NI le registre (textes fournis au CR),
  et rapporte branche + SHA ; fusion par l'orchestrateur, procedure du 16/08 (FUSION_WT).
- Recouvrement previsible : `ReplaySettingsDrawer.tsx`, `i18n.ts`, `ReplayCanvas.tsx`
  (calque). A signaler dans le CR pour la fusion.

## Regles dures

Aucun decoupage devine (pas de fond = brut) ; le brut reste stocke ; tokens uniquement ; FR et
EN ; JAMAIS `git add -A` (fichiers d'une autre session dans l'arbre) ; jamais de pause
d'attente passive ; AUCUNE base DuckDB en ecriture ; zero fix hors perimetre — decouvertes au
registre.

## Statuts et cloture

`[x]` / `[~]` / `[!]` avec justification ; aucune case vide ; journal date ; registre (la ligne
« Decoupage universel des callouts sur sol praticable » sort si le gate A passe) ; commits sur
`feat/v75` (A) et `wt/heatmap` (B), pas de push.
