# PLAN — finaliser le rejeu 2D dans l'app

> Écrit le 2026-07-31. Branche `feat/filmdec-continuation`.
> Contrat d'exécution : skill `plan-execution`. Ordre strict, une étape close avant la suivante,
> aucun report d'une action faisable, chaque item statué à la clôture.
>
> Ce plan REMPLACE `V7.5/replay2d/PLAN_POC_DANS_L_APP.md` (dont les étapes 1 à 7 sont closes) pour la suite.
> Il ne répète pas `HANDOFF_REPLAY_2D_2026-07-29.md` : celui-ci reste la porte d'entrée.

---

## CE QUE CE PLAN VISE

Le rejeu 2D fonctionne et n'est atteignable par personne. Le finaliser veut dire quatre choses,
dans cet ordre de dépendance :

1. **Le rendre atteignable** — aujourd'hui la page est orpheline : aucun lien, nulle part.
2. **Le rendre défendable** — aucun des chiffres mesurés du chantier n'est verrouillé par un
   test. Ils vivent dans des Markdown et dans un artefact généré, pas dans le dépôt.
3. **Le rendre propre** — trois couches à moitié séparées, cinq catalogues Halo en dur, deux
   tables de grenades qui se contredisent, un contrat d'API qui décrit 6 champs sur 22.
4. **Le rendre général** — 2 cartes sur 14 ont un fond de carte. La recette existe et tourne.

---

## L'ÉTAT DES LIEUX, MESURÉ

Relevé le 2026-07-31 par six audits parallèles. Chaque ligne est vérifiée sur pièce.

| domaine | constat |
|---|---|
| **Atteignabilité** | `REJEU_2D_ENABLED` (feature-flags.ts:14) : **zéro import depuis sa création** le 2026-04-21. La page `/replay` fonctionne mais **aucun lien ne mène à elle**, ni dans match-view, ni dans la navigation |
| **Garde** | `replay_local_gate.go` est le seul garde vivant. Son critère de retrait est **déjà satisfait sur un film** (475/519 = 91,5 %, bridge nominal, 0 collision) ; il en manque un second |
| **Couches (Go)** | `internal/analysis/replay` = 3 075 lignes : **883 de décodage** (I/O + bits) et **2 192 d'assemblage**, soudées par `BuildFromFilm` (build.go:93, 83 lignes, 8 appels de décodage) |
| **Couches (web)** | La séparation logique/rendu est **réellement tenue** (replayLogic, rosterLogic, coverageLogic, mapFloor, shotEffects sont purs et testés). Une exception : `ReplayCanvas.tsx`, fonction exportée de **363 lignes** |
| **Title-agnostic** | **Aucune comparaison de slug** — mais 5 catalogues Halo en dur, et `doc.titleSlug` **n'est jamais lu**. Aucune capability ne garde la feature. `config/titles/halo_infinite/mappings/weapon_names.toml` **existe déjà, bilingue** |
| **Contradiction** | Le rang 2 de grenade s'appelle **Dynamo** dans `inventory.go:29` et **Shock** dans `filmdec/grenade_events.go:71` — les deux peuvent apparaître sur la même fiche |
| **Contrat** | L'artefact publie **22 champs**, l'OpenAPI en décrit **6**, les types TS sont écrits à la main hors du fichier généré. **12 champs publiés ne sont lus par personne** |
| **Tests** | 42 tests Go + 102 web, **tous sur données synthétiques**. Couverture réelle : **58,5 %** (replay), **18,2 %** (filmdec), **30,7 %** (web). **Aucun chiffre mesuré n'est verrouillé** |
| **Golden** | Le dépôt en fait **déjà** à quatre endroits, dont une **fixture binaire de film de 196 Ko versionnée avec son outil de régénération** |
| **Duplication** | `keep*OfPublishedTracks` est recopié **4 fois** (la règle est 2) |
| **Cartes** | **14** cartes ont des bornes, **2** ont un fond de carte, 34 variantes ont des objectifs — que **personne ne lit à l'exécution** |
| **Cartes (débloquant)** | `mapstruct-build` a produit **les 14 cartes en ~2 minutes, sans modification de code** |
| **Cartes (mesure neuve)** | Le lien nom → module est **établi pour 21 niveaux** : le `level_id` du `.mvar` apparaît dans **exactement un** `.module`, 21 fois sur 21, avec 14/14 d'accord sur l'existant et 0 sur 2 témoins |
| **Poly** | `Surface.Poly` existe depuis le 2026-07-26 et **aucun code ne le remplit** : 0 sur 194 724 emprises |

---

## LA QUESTION DES MUNITIONS À ÉNERGIE — TRANCHÉE

**Votre intuition était juste sur le fond, et la mienne était fausse sur un point.**

### Ce que le décodeur lit

Une union à deux branches : soit un **chargeur** entier `R(8)` + réserve `R(11)`, soit une
**jauge** `R(12) / 4095` — donc une fraction, par construction du code.

### Ce que la mesure dit

**La grandeur sous-jacente EST discrète.** Par arme, les valeurs 12 bits forment un **treillis
arithmétique exact** :

| arme | valeurs brutes | structure |
|---|---|---|
| Gravity Hammer | 410, 820, 1230, 2050 | **410 × {1, 2, 3, 5}** |
| Energy Sword | 594, 1188 | **594 × {1, 2}** |
| Stalker Rifle | 127, 316, 442, 568 | tous **congrus à 1 modulo 63** (et 63 × 65 = 4095 pile) |

Ce n'est donc **pas un pourcentage** : c'est un **compteur de charges normalisé**. Mais son
dénominateur — environ 10 pour le marteau, 7 pour l'épée, 65 pour le Stalker — est **propre à
l'arme et n'est pas porté par le film**. On ne peut donc pas afficher « 3 / 10 » sans une table
qu'on n'a pas.

Contrôle : `gauge × 100` ne tombe **jamais** sur un entier (0 sur 17 valeurs distinctes), et le
plus petit dénominateur commun cherché de 2 à 4200 est **exactement 4095**.

### CALIBRER LE PLEIN SUR UN SPAWN : l'idée est juste, mais le film l'interdit

L'utilisateur propose de lire la jauge **au spawn** — en Fiesta un joueur apparaît avec le plein
— pour en déduire le dénominateur. Mesuré sur les **deux** films :

| film | arme à jauge | lecture la plus proche d'une apparition |
|---|---|---|
| `000d5950` | Gravity Hammer | **15,7 s** après le spawn |
| `000d5950` | Energy Sword | 18,3 s |
| `01e1f945` | Energy Sword | 25,0 s |

**Aucune lecture ne tombe près d'un spawn, et ce n'est pas un hasard** : l'inventaire n'est lu
qu'aux **images-clés**, à cadence fixe d'environ 20 s, indépendante des apparitions. Une
image-clé ne coïncide donc presque jamais avec un spawn — c'est structurel, pas une malchance de
ce film.

**Conséquence** : la calibration par le spawn exige une source **continue**, pas les images-clés.
Soit une capture mémoire (qui échantillonne en continu), soit un autre champ du film.

Ce que les deux films disent tout de même : le quantum de l'épée est **594** sur les deux
(valeurs 594, 1188, 1782 = 594 × {1, 2, 3}), ce qui confirme le treillis sur une seconde carte
et un second match.

### SUJET CLOS — ce qu'on affiche, et ce qu'on n'affichera pas

**Oui, on peut afficher un pourcentage, et c'est fait.** La jauge compte la consommation, le
plein est l'absence de champ : la part restante vaut donc `100 × (1 − valeur/4095)`. La barre
affiche ce complément depuis le 2026-07-31 — elle était inversée avant.

**Le dénominateur se dérive, sans capture** : c'est le pas du treillis.

| arme | quantum | charges = 4095 / quantum |
|---|---|---|
| Gravity Hammer | 410 | **10** |
| Energy Sword | 594 | **7** |
| Stalker Rifle | 63 | **65** |

- [ ] **Ce qui reste à faire, et c'est petit** : confirmer le quantum sur un **second film**
      (déjà vrai pour l'épée : 594 sur les deux) avant de publier le compte de charges.
- [ ] **Ce qu'on n'affichera pas tant que ce n'est pas confirmé** : « 3 sur 10 ». Le compte
      dérive d'une régularité observée sur un match, pas d'une lecture. La barre, elle, ne
      suppose rien.

### La prémisse « armes à énergie » est en partie fausse

Le Disruptor (7/14), le Shock Rifle (15/30), le Heatwave (8/8) et le Sentinel Beam (80/80)
sortent des **chargeurs entiers** conformes à la table de référence. La jauge concerne le
**marteau, l'épée, le Stalker Rifle et le Ravager** — c'est-à-dire les armes à charge, pas les
armes à plasma.

### ET J'AVAIS PUBLIÉ UNE RÉFUTATION FAUSSE

J'avais conclu le 2026-07-29 que « la cellule de munitions *k* n'est pas l'arme *k* », sur une
mesure faite sur **tous** les appariements. Refaite en ne gardant que les lectures **uniques**
(`cand === 1`) :

| | tous (300) | lectures uniques (198) |
|---|---|---|
| armes à 100 % chargeur | 15 sur 22 | **16 sur 21** |
| Gravity Hammer | 13 chargeur / 17 jauge / 8 aucune | **1 / 13 / 5** |
| Energy Sword | 3 / 5 / 3 | **0 / 3 / 3** |
| Stalker Rifle | 5 / 9 / 1 | **0 / 5 / 1** |

**197 appariements sur 198 concordent.** Ce que j'avais pris pour un échec de rattachement était
le bruit des parses à plusieurs candidats (51 records sur 150). La correspondance tient ; c'est
la **confiance dans la lecture** qui doit être filtrée, pas le rattachement qui est cassé.

**Corrigé le 2026-07-31** dans l'infobulle FR/EN, le type TS et le commentaire du composant.

---

## LOT 1 — RENDRE LE REJEU ATTEIGNABLE  *(le plus court chemin vers une valeur visible)*

### 1.1 Supprimer le drapeau mort
- [ ] Effacer `apps/web/src/lib/feature-flags.ts` (14 lignes, un export, zéro import depuis le
      2026-04-21). Abaisser le plafond `knip` en conséquence — **abaisser, pas relever**.

### 1.2 Publier la disponibilité par match
- [ ] Ajouter `ReplayAvailable bool \`json:"replay_available"\`` à `MatchViewHeader`
      (`internal/domain/match_view.go`), à côté de `WaypointURL` — c'est le patron du dépôt pour
      une disponibilité par-match.
- [ ] Le remplir depuis la présence de l'artefact via `PathResolver.ReplayArtifactPath`.
      **Un `os.Stat`, pas une lecture** : le handler ne doit pas charger 2 Mo pour dire oui.

### 1.3 Poser le lien
- [ ] Dans `MatchHeader.tsx` / `MatchNavigationBar`, un lien conditionné à `replay_available`.
      i18n FR+EN. **Pas de lien vers une page vide** : c'est toute la raison de 1.2.

### 1.4 Le garde local — SON CRITÈRE EST ATTEINT (mesuré le 2026-07-31)

Le garde exigeait « couverture des tirs > 85 % et `verdict.bridge` nominal sur au moins **deux
films de cartes différentes**, sans collision de trace ». Le second artefact a été construit :

| film | carte | tirs | verdict pont | collisions | désaccords d'index |
|---|---|---|---|---|---|
| `000d5950` | Cliffhanger | 475/519 = **91,5 %** | nominal | 0 | 0 |
| `01e1f945` | Catalyst | 1862/2154 = **86,4 %** | nominal | 0 | 0 |

### UN TROISIÈME FILM TEMPÈRE CETTE CONCLUSION — mesuré le 2026-07-31

| film | carte / mode | tirs rattachés | |
|---|---|---|---|
| `000d5950` | Cliffhanger, Slayer | 475/519 = **91,5 %** | au-dessus |
| `01e1f945` | Catalyst, Slayer | 1862/2154 = **86,4 %** | au-dessus |
| `64e8adfa` | Catalyst, **CTF** | 2312/2879 = **80,3 %** | **SOUS LE SEUIL** |

Le critère est **littéralement** satisfait — deux films, deux cartes, tous deux au-dessus de
85 %. Mais un troisième tombe à 80,3 %, et il était déjà en cache : il suffisait de le
construire.

**Ce que cela apprend : le critère était trop faible tel qu'il était écrit.** « Au moins deux
films » n'est pas « tous ceux qu'on mesure », et un critère qu'on peut satisfaire en choisissant
ses films ne protège de rien. Les trois ponts restent pourtant **nominaux, sans aucune collision
de trace** — ce n'est donc pas le rattachement qui faiblit, c'est la part d'événements dont le
tireur n'a pas de trace publiée.

- [ ] Avant de retirer le garde : comprendre pourquoi le CTF perd 564 tirs pour « slot
      introuvable » là où le Slayer en perd 44. Un mode où l'on meurt davantage produit plus de
      vies courtes, donc plus de traces non publiées — hypothèse à mesurer, pas à supposer.
- [ ] Puis **réécrire le critère** sur ce qu'on veut vraiment : un plancher sur **tous** les
      films mesurés, et une date.

- [ ] Retirer `replay_local_gate.go` **d'un bloc**, avec son test et sa variable
      `LEVELUP_REPLAY_PUBLIC` — un garde dont le critère est atteint et qu'on garde « au cas où »
      est exactement le « compatibility guard forever » que le CLAUDE.md interdit.
- [ ] Si l'utilisateur préfère le conserver malgré tout, alors son critère était mal écrit :
      le réécrire avec ce qui manque VRAIMENT (une revue visuelle ? un troisième film ?) et une
      **date calendaire**, comme l'exige la règle 11.

**Note sur le second film** : Catalyst n'a **pas** de fond de carte figé (`structure=0`) — il a
été volontairement écarté, sa couverture mesurée n'étant que de 40-49 %. Le rejeu y fonctionne
donc sans sol reconstruit. C'est un cas de dégradation utile à regarder à l'écran.

**GATE 1** : depuis la page d'un match qui a un artefact, un lien mène au rejeu ; sur un match
sans artefact, aucun lien. Revue visuelle utilisateur.

---

## LOT 2 — VERROUILLER CE QUI EST ÉTABLI  *(sans quoi tout le reste est réversible)*

Le dépôt a déjà le patron : une fixture binaire versionnée avec son outil de régénération. Le
paquet voisin `killsource` le pousse plus loin — **sorties versionnées, fixtures non versionnées,
et un second test qui tourne SANS fixture** pour interdire que les golden dégénèrent en nombres
nus. C'est ce modèle qu'on reprend.

> **LOT 2 CLOS le 2026-07-31 (jalon J2).** Les six items sont statués ci-dessous. Gate 2
> atteint : `go test ./internal/analysis/replay/... ./internal/analysis/filmdec/...` vert, et
> tout chiffre du chantier qui bouge fait tomber un test NOMMÉ. Commits : `5e37f4c79` (2.1+2.2),
> `d1870b3a7` (2.3), `a05d7d448` (2.4), `96bc56175` (2.5), `eb0f12d23` (2.6).
> Couverture Go du paquet `replay` : **58,5 % → 79,2 %**.

### 2.1 Le golden le plus rentable : les entrées décodées
- [x] `BuildFromPositions` est **déjà pur** — il ne lui manque que ses entrées. Sérialiser
      depuis le film de référence les positions, tirs, loadouts, grenades, projectiles, morts et
      index de joueur, puis rejouer l'assemblage.
      *FAIT.* `testdata/inputs_000d5950.bin.gz` (633 Ko) porte les 171 826 positions, 519 tirs,
      150 loadouts, 70 lancers, 580 trajectoires, 184 inventaires, 93 morts et les 8 index.
      Format delta-codé **au centimètre** — la précision exacte que `round2` publie, sans
      exception, donc pas une décimale perdue. La régénération est la seule porte d'écriture et
      exige `-update` **et** `REPLAY_FILM_DIR`.
- [x] **Verrouille sans un octet de film** : 475/519 tirs, 90/105 vies nommées (99 traces
      publiées dont 90 nommées), 70 lancers, 439 projectiles, 184 états d'inventaire, la
      couverture et ses verdicts.
      *FAIT.* Un golden lisible en diff (`assembly_000d5950.golden`) + six tests NOMMÉS qui
      portent chacun un chiffre, + un garde-fou qui vérifie que le golden n'a pas dégénéré en
      nombres nus (patron killsource) et qu'aucun pourcentage n'y figure sans sa fraction.

### 2.2 La mini-bobine : verrouiller le DÉCODAGE
- [x] ~560 Ko de **paquets réels concaténés** — contre 20,2 Mo pour le film et 23 Go pour le
      cache.
      *FAIT, 698 Ko* (chunk_01 527 Ko + chunk_02 7 Ko + chunk_03 171 Ko). L'écart avec
      l'estimation vient de deux **mesures faites en la construisant**, toutes deux nécessaires :
      (a) la PREMIÈRE image-clé du film ne rend NI loadout NI inventaire — c'est une image
      d'amorce — et la bande de slots d'objets se lit sur l'ENSEMBLE des images-clés ; en garder
      une seule aurait verrouillé deux décodeurs sur un résultat vide, ce qui est pire que ne
      rien verrouiller. (b) le second maillon du pont exige DEUX chunks de réplication, sans quoi
      l'exigence de concordance n'est jamais exercée.
- [x] Versionner sous `internal/analysis/replay/testdata/minifilm_000d5950/`, **avec son outil
      de régénération** (patron du dépôt).
      *FAIT*, plus un `PROVENANCE.txt` versionné qui dit d'où vient chaque octet — une fixture
      binaire sans provenance écrite est un fait sans source.
- [x] **Ajout J2** : cible de fuzz Go native sur les lecteurs de records + corpus de graines
      régénérable depuis la mini-bobine (§6.7 du master plan). Elle a trouvé **deux paniques dès
      sa première exécution** (consignées en Découvertes, non corrigées). Campagne de contrôle :
      334 625 exécutions, aucun autre plantage.

### 2.3 La grammaire d'inventaire — sept fonctions à 0 %
- [x] Les tests actuels ne couvrent que le sous-bloc de munitions sur un flux fabriqué. Les
      **quatre règles d'ancrage** (R1 capacité, R2 grenades, R3 armes, R4 munitions) ne sont pas
      testées. C'est le plus gros trou du paquet.
      *FAIT.* Chaque règle est testée par ce qu'elle REFUSE autant que par ce qu'elle trouve —
      notamment R2, dont le point de départ est la preuve : un motif de grenade identique placé
      AVANT l'ancre de capacité doit rester invisible, sans quoi la règle se validerait
      elle-même. Plus un test des quatre règles ENSEMBLE sur le binaire réel : 184 états, 132
      capacités, 120 compteurs, 150 blocs de munitions, 51 lectures à plusieurs candidats — les
      NON-lectures sont verrouillées comme les lectures.

### 2.4 Le rendu canvas, sans navigateur
- [x] Un **contexte enregistreur** : un objet qui empile `{op, args}`. Une seule lecture à
      traiter (`createRadialGradient`), aucune dépendance nouvelle.
      *FAIT* (`canvasRecording.test.ts`, 21 tests).
- [x] Verrouille les huit formes d'effet de tir et la trame du sol.
      *FAIT*, et le correctif J1-a (champ `w`) rend enfin ce test significatif : avant lui, les
      huit familles étaient inatteignables. **Découverte** : il n'y a que SEPT géométries pour
      huit familles (cf. Découvertes).

### 2.5 Les trois décodeurs d'événements de `filmdec`, tous à 0 %
- [x] Les 439 projectiles, les 70 lancers et les tirs sortent de fonctions **jamais testées**.
      *FAIT.* Un test par offset mesuré — l'index de lanceur à +103 bits et non +102, les deux
      moitiés 32 bits de l'arme, la visée lue SEULEMENT sur le chemin où elle est localisable —
      plus les portes de rejet du balayage de projectile (slot hors bande, i0 absent, composants
      non strictement croissants, quantum saturé).
- [x] Tester aussi le second maillon du pont : `ScanFilmPlayerIndices`, `rosterFromDeaths`,
      `injectiveOrEmpty` — ce sont eux qui décident si un tir est publié.
      *FAIT*, sur binaire réel (2 chunks concordants, 8 identités, table bijective) et sur les
      deux refus qui comptent : une table non injective est ÉCARTÉE EN ENTIER, jamais nettoyée.

### 2.6 Aligner les trois contrats
- [x] Go publie 22 champs, l'OpenAPI en décrit 6, le TS est écrit à la main. Compléter le
      schéma, et poser un test qui **fait tomber la divergence** au lieu de la laisser passer.
      *PRÉMISSE PÉRIMÉE, VÉRIFIÉE SUR PIÈCES : l'alignement était déjà fait par J1.2.* Le
      contrat décrit bien les 22 champs et porte déjà l'arité des tuples (`minItems` =
      `maxItems` sur `Surface.poly` et `Projectile.p`) ; `openapi-gen -check` dit « à jour » et
      `generate-types` ne produit aucune dérive. **Ce qui manquait était le test**, et il est
      posé : côté Go (`contracttest`, sans CGO) chaque type publié est confronté à son schéma
      DANS LES DEUX SENS, plus l'arité et la cohérence `omitempty` ↔ `required` ; côté web, la
      complétude de la frontière `replayNormalize` est prouvée **par le compilateur** contre les
      types générés (falsifié : en simulant un champ qui lui échappe, `tsc -b` refuse de
      compiler).
- [~] Les 12 champs que personne ne lit : **consignés seulement**, inventaire champ par champ en
      tête de `contracttest/replay_contract_test.go`. Leur sort est le lot **3.6**.

**GATE 2** : `go test ./internal/analysis/replay/...` verrouille les chiffres du chantier ; un
changement de décodeur qui les déplace fait tomber un test nommé. **ATTEINT le 2026-07-31.**

---

## LOT 3 — LES COUCHES ET LE TITLE-AGNOSTIC

### 3.1 Les corrections immédiates (aucune raison d'attendre)
- [ ] **Unifier les deux tables de grenade** — « Dynamo » contre « Shock » pour le même rang.
      Une seule table, un garde-rail qui interdit la seconde.
- [ ] **Factoriser les 4 copies** de `keep*OfPublishedTracks` en un helper, avec garde-rail.
- [ ] Sortir `mapvar` de `internal/analysis/replay` : 673 lignes qu'aucun fichier du rejeu ne
      consomme.
- [ ] Faire passer la géométrie par `PathResolver` : `replay-build` lit aujourd'hui `.ai/V7.5/dumps`
      **par défaut**, un répertoire de rétro-ingénierie hors de `data/`.

### 3.2 Sortir les catalogues Halo vers les mappings de titre
- [ ] Armes : brancher `WeaponLabels` sur `config/titles/halo_infinite/mappings/weapon_names.toml`
      — **il existe déjà et il est bilingue**.
- [ ] Grenades, capacités : même traitement. Aujourd'hui les noms de capacité sont **en français
      dans du Go**, ce qui interdit l'anglais autant que l'ajout d'un titre.
- [ ] Web : les 22 noms d'armes en dur de `shotEffects.ts` doivent venir du document, pas du code.

### 3.3 Découper le paquet Go
- [ ] Un paquet de **décodage** (883 lignes : `inventory_decode`, `deaths_source`, `player_index`,
      `geometry`, `structure`) et un paquet d'**assemblage** (le reste, pur et testable sans I/O).
- [ ] **À faire APRÈS le lot 2** : découper sans filet, c'est déplacer du code non testé.

### 3.4 Découper `ReplayCanvas.tsx`
- [ ] 363 lignes qui portent l'état, la résolution des couleurs, la trame du sol, la boucle
      d'animation et l'ordre des calques. Extraire au minimum la boucle et la composition.

### 3.5 Poser la capability
- [ ] Déclarer `film.replay2d` et y brancher la route et le lien. Aujourd'hui la seule porte est
      un 404 sur un fichier absent : un titre qui ne sait pas produire de rejeu n'a aucun moyen
      de le dire.

### 3.6 Statuer les 12 champs que personne ne lit
- [ ] `Track.Name` n'est **jamais écrit** → supprimer. Les autres : soit un consommateur, soit la
      suppression. Un champ publié que personne ne lit est du code mort qui coûte de la bande
      passante.

**GATE 3** : `go test ./...` + `make check-types` + lint ; le ratchet de code mort baisse au lieu
de monter ; aucun catalogue Halo ne subsiste dans le code du rejeu.

---

## LOT 4 — TOUTES LES CARTES

L'audit a montré que **ce lot est bien plus proche qu'on ne croyait** : la production tourne déjà
sur les 14 cartes du catalogue, en deux minutes, sans modification de code.

### 4.1 Réparer la mesure avant de trier avec elle
- [ ] `coverage()` renvoie **0 en silence** au-delà de 40 M de cellules (`extract.go:144`) :
      `btb_highpower`, `chasm` et `illusion` sont déclarées à « 0,0 % » alors qu'elles ne sont pas
      mesurées. Une mesure impossible doit se dire, pas se taire.
- [ ] La couverture est calculée sur les bornes du **BSP d'enceinte** (jusqu'à 4041 × 5508 m)
      alors que le client rasterise sur l'étendue des **joueurs**. Le chiffre publié ne décrit pas
      ce qui sera affiché.

### 4.2 Le lien nom → module, canonique et gardé
- [ ] Le `level_id` du `.mvar` apparaît dans **exactement un** `.module` : 21/21, 14/14 d'accord
      avec l'existant, 0 sur 2 témoins. **C'est l'action qui débloque tout le reste.**
- [ ] Étendre `map_quant_bounds.json` aux cinq cartes désormais résolues et immédiatement
      lisibles : Deadlock → `btb_drydock`, Oasis → `btb_exiled`, Scarr → `btb_engine`,
      Prism → `sgh_crystalcaves`, Recharge → `sgh_blueprint`.
- [ ] Normaliser les suffixes de mode dans `NormalizeMapName` : « Fragmentation Heavies »
      (22 matchs) et « Oasis Heavies » (16) ne s'apparient à rien aujourd'hui.

### 4.3 Figer les fonds de carte
- [ ] Une fois 4.1 corrigé, produire et versionner les fichiers de structure des cartes
      exploitables. La commande tourne déjà.
- [ ] **Lever le doute sur `chasm` et `ctf_illusion`**, qui publient des bornes **identiques** :
      deux cartes ne peuvent pas avoir la même enveloppe monde.
- [ ] `Live Fire` est résolu vers `sgh_interlock` mais le module local **ne contient aucun tag
      sbsp** : il n'est pas téléchargé. Statuer.

### 4.4 Le champ `Poly` : le remplir ou le retirer
- [ ] Déclaré, lu par le client, testé, et **vide partout** (0 sur 194 724). Soit `mapstruct-build`
      le produit, soit on retire le champ et le code qui l'attend. Un champ toujours vide qui a
      l'air d'une fonctionnalité, c'est la définition du code mort.

### 4.5 Écrire la recette dans `docs/`
- [ ] Elle n'existe que dans des en-têtes de fichier et des notes `.ai/`. Pré-requis (jeu
      installé, arborescence), commandes, sorties attendues.

### 4.6 Statuer `map_objectives.json`
- [ ] 385 Ko versionnés, 34 variantes, **aucun lecteur à l'exécution**. Le brancher ou acter la
      dette avec une date.

**GATE 4** : toutes les cartes jouées ont des bornes ; celles dont la structure est exploitable
ont un fond figé ; la recette est dans `docs/` et un tiers peut la suivre.

---

## LOT 5 — LE FIL DES ÉLIMINATIONS  *(bloqué, et il faut le dire)*

Rien de ce lot ne peut avancer avant **votre décision** sur le rapprochement des branches
(cf. `HANDOFF_REPLAY_2D_2026-07-29.md`). Rappel du fait décisif : `feat/filmdec-killweapon`
**n'a aucun ancêtre commun avec `main`**, et les deux branches ont fait diverger `filmdec` dans
les deux sens.

Quand la décision sera prise :
- [ ] 5.1 Le fil entre par une **entrée de données** (`Options.Kills`), comme `Deaths` ou
      `Loadouts`. Un adaptateur écrit à part connaît les deux formes. **Aucun type `Kill` n'est
      déclaré avant qu'un producteur ne l'alimente.**
- [ ] 5.2 La colonne du fil : arme du dégât fatal, horodatage, liseré d'équipe, assistance et
      ses deux parts de dégâts.
- [ ] 5.3 Les médailles en images, qui en dépendent.
- [ ] 5.4 Les effets de tir s'appliquent alors aussi aux **morts**, pas seulement aux tirs.

---

## LOT 6 — LES VARIABLES DÉCODÉES PUIS JETÉES

Le décodeur est un **sauteur de bits par construction** : environ 200 grammaires de composants
sont portées, et **quatre seulement rendent une valeur**. Une trentaine de grandeurs sont donc
lues bit-exact puis abandonnées. Toutes ne se valent pas — voici l'arbitrage.

### 6.1 À BRANCHER — l'état actif des capacités (`i57`)

C'est la seule qui serve une fonctionnalité **déjà dessinée** : le POC prévoit le surbouclier en
encadré doré, le camouflage en effet de verre, le translocateur en bordure animée (Notion 21.1,
rang 3 du SUIVI).

**Où on en est, exactement :**
- le composant `i57` est **absent du switch** du décodeur — il n'est pas lu du tout ;
- l'ADDENDUM du 2026-07-26 le mesure à **2 bits** sur 990 lectures ;
- le POC avait tenté un `i57 bit 0` qui valait **1 sur 386 occurrences sur 386** — un
  interrupteur qui ne bascule jamais. C'est pour cela que le badge d'état a été retiré.

**Ce qui manque avant de coder** : savoir ce que portent réellement ces 2 bits. La capture d'un
match avec surbouclier et camouflage (cf. `SESSION_CAPTURE_AVANT_PC.md`) donne le relevé terrain
qui permettra de le dire — et de le falsifier.

- [ ] 6.1a Brancher `i57` dans le switch et publier ses 2 bits **sans les interpréter**.
- [ ] 6.1b Confronter au relevé terrain de la capture. Un bit constant se retire, il ne se
      publie pas.

### 6.2 À BRANCHER — la table des capacités, lue sur 6 bits et non sur 3

**C'est la mesure déclarée prioritaire depuis le 2026-07-27 et jamais faite**, et elle explique
sans doute pourquoi notre table est partielle :

| | |
|---|---|
| ce qu'on lit | l'index de capacité sur **3 bits** → 8 valeurs possibles |
| ce que le binaire décrit | **6 bits précédés d'une porte** → 64 valeurs |
| ce que le jeu contient | **11 capacités** |
| ce que notre table connaît | **4** (mur portatif, grappin, propulseur, capteur de menace) |

Trois bits ne suffisent pas à coder 11 capacités. Le contrôle croisé le disait déjà : la palette
`sofd` donne mur = rang 2 et répulseur = rang 6, notre table dit mur = 3 et capteur = 6 —
**2 confirmés sur 4**, et les deux confirmés sont adossés à un triplet de joueurs quand les deux
contredits reposent sur une observation unique.

- [ ] 6.2 Relire les **mêmes records** sur 6 bits et vérifier si le mur ressort au rang 2 et le
      capteur au rang 1. **Aucune capture n'est nécessaire** — le document le dit explicitement,
      « le problème des deux films est contourné sans nouvelle capture ».

### 6.3 À BRANCHER — le compteur de réapparition et l'horloge de manche

Ils font partie des **quatre composants qui rendent déjà une valeur**, et ils ne sortent
pourtant jamais de `filmdec` : leurs seuls appelants sont une sonde jetable.

Or le rejeu affiche aujourd'hui un compteur de réapparition **déduit** de l'image de départ de
la vie suivante. Le film porte le compteur **réel**. Remplacer une déduction par une lecture est
exactement la règle de ce chantier.

- [ ] 6.3 Publier `player-respawn-timer` et `game-engine-round-timer` dans l'artefact ; faire du
      compteur lu la source, et de la déduction un repli explicite et compté.

### 6.4 À REGARDER — le pitch de visée, l'orientation du corps, la vélocité

Décodés dans le **même record** que la position, jamais publiés : seul le lacet de visée sort,
sous `Point.H`. Le pitch dirait si un joueur vise le sol ou une passerelle ; l'orientation du
corps, distincte du regard, dirait s'il recule en tirant.

- [ ] 6.4 Mesurer d'abord leur **couverture** (le lacet n'est présent que sur ~52 % des points).
      Publier ensuite, seulement si la couverture le justifie.

### 6.5 À NE PAS FAIRE — et il faut le dire aussi

| variable | pourquoi on n'y touche pas |
|---|---|
| **le code de médaille** du chunk highlight | **L'API le lit très bien**, et les visuels sont un mapping à reprendre de nos pages existantes. Le film le porte et le perd à l'écriture : c'est une redondance, pas un manque. *Mon audit avait surestimé ce point.* |
| **le dead-state `i11`** (victime, tueur, catégorie, source du dégât) | **C'est exactement ce que le chantier voisin décode** — sa « source du dégât » est lue dans l'état de mort de la victime. Deux décodeurs du même fait divergeraient. Il reste chez lui. |
| parties du corps touchées, camouflage, accroupissement, vies restantes, dernier traître | aucune n'a de destination dans le produit. On les laisse sautées ; les rouvrir demanderait une raison. |

---

## LOT 7 — LES OBJECTIFS

Le dépôt en a **déjà beaucoup plus qu'il n'en montre** : un endpoint câblé de bout en bout, deux
tables peuplées de **8 140 événements sur 237 matchs**, un décodeur de film complet — mais le
front n'en consomme que **577, soit 7 %**, et le seul producteur est un CLI de diagnostic
qu'aucune synchronisation ne déclenche.

### 7.1 Le témoin, mesuré

| film | mode | événements d'objectif |
|---|---|---|
| `64e8adfa` | Catalyst, CTF | **68** |
| `000d5950` | Cliffhanger, Slayer | **0** |

L'archétype des objectifs est bien présent : **5 entités par image-clé** sur le film CTF, aucune
sur le Slayer, et 34 composants dont `progress`, `required-progress`, `state`,
`object-reference`.

### 7.2 Ce qui bloque vraiment — et ce n'était pas ce qu'on croyait

Le SUIVI disait que le blocage était `interaction-filter` en `i4`, polymorphe à 6 sous-types.
**C'est faux** : aucun des 162 désérialiseurs portés ne sait lire **un seul** composant
`managed-objective`. La traversée bute **dès `i0`**, pas à `i4`.

- [ ] 7.2a **Capturer la table des désérialiseurs** — c'est le préalable absolu, et il exige
      Cheat Engine (cf. `SESSION_CAPTURE_AVANT_PC.md`, étape 1).
- [ ] 7.2b Décompiler les 34 désérialiseurs de l'archétype d'objectif dans Ghidra, **hors
      ligne**, avec le projet désormais sur la clé.
- [ ] 7.2c Les porter en Go, un par un, en commençant par `state` et `progress`.

### 7.3 Le chemin court, qui ne demande AUCUNE capture

Il existe, et il vaut d'être pris en premier :

- [ ] 7.3a **Brancher le producteur** : les 8 140 événements viennent d'un CLI que rien ne
      déclenche. Les faire produire par la synchronisation.
- [ ] 7.3b **Consommer les 93 % restants** côté front.
- [ ] 7.3c **Lire `map_objectives.json`** : 385 Ko versionnés, 34 variantes, et il contient déjà
      les 4 socles de livraison de Catalyst. **Personne ne le lit à l'exécution.**
- [ ] 7.3d La sémantique du type d'événement en mode à zones **est établie** : le nombre
      d'événements d'un joueur égale `zone_captures + zone_secures`, exact sur **418 paires sur
      503**. En revanche les libellés de capture de colline et de port du crâne **ne sont validés
      par rien**, et le décodeur **sur-compte le crâne d'un facteur 20**. À corriger avant de
      montrer quoi que ce soit.

### 7.4 Les scores de mode à objectif — la logique ne tient pas

Mesuré sur les données réelles, et c'est le constat le plus sérieux de tout cet audit :

| constat | mesure |
|---|---|
| la famille `zones_strongholds` **mélange deux modes** | **61,6 % de Total Control**, 36,8 % de Strongholds |
| sa calibration est la **moyenne des deux** | un joueur au P80 réel de Strongholds sort **saturé à 100/100** ; au P80 réel de Total Control, **56,8/100**. Le design annonce 80 pour les deux |
| le split KOTH / Strongholds se fait **par ligne** | **24 des 47 matchs KOTH (51 %)** sont scindés entre deux familles aux calibrations différentes |
| les scores bruts ne sont **jamais interprétés par mode** | Oddball contredit le résultat **2 fois sur 26** · Total Control affiche 0-0 avec un vainqueur **16 fois sur 124** · KOTH est bimodal sous le **même** nom de variante |
| il existe **six classifications de mode parallèles** | alors que `canonical.ModeCanonical` existe et n'est pas utilisé par le calcul |

- [ ] 7.4a **Séparer Strongholds de Total Control** et recalibrer chacun sur son propre P80.
- [ ] 7.4b Faire le split **par match**, pas par ligne.
- [ ] 7.4c Interpréter le score final **par mode**, ou ne pas l'afficher.
- [ ] 7.4d Ramener les six classifications à `canonical.ModeCanonical`.

**ATTENTION — ce lot ne vit pas sur cette branche.** La logique de score qui tourne réellement
est sur `main`, et ce worktree en est **1 857 commits derrière**. Le lot 7.4 se fait sur `main`
ou sur une branche qui en descend, pas ici.

---

## DÉCOUVERTES — consignées, NON traitées

> Règle 7 du contrat `plan-execution` : une découverte se note, elle ne se traite pas. Aucune
> des cinq n'a été corrigée — toutes touchent la LOGIQUE du décodeur ou du rendu, hors périmètre
> de J2 (verrouiller précède modifier).

### D1 — `decodeFireEvent` panique sur un payload court *(la plus sérieuse)*

Trouvée par la cible de fuzz, à sa première exécution. Le décodeur lit sans garde jusqu'au
bit 112 (les drapeaux) : **tout payload de moins de 14 octets le fait sortir des bornes**, parce
que `readBitsAt` indexe le tableau sans le borner — contrairement à `PeekBits`.

**Le chemin est ATTEIGNABLE EN PRODUCTION** : `ScanFilmFireEvents` n'exige que `p.Size >= 1`
avant d'appeler ce décodeur. Un paquet delta tronqué dont le premier octet vaut `0xD2` suffit à
faire tomber tout le décodage d'un film. Aucun film du cache ne le produit aujourd'hui ; un
téléchargement partiel, si.

### D2 — la tolérance de `PeekBits` est à sens unique

Sa documentation annonce qu'elle « ne panique jamais sur un payload tronqué ». C'est vrai
au-delà de la fin du buffer (les bits y valent 0), **faux avant le début** : une position
négative panique (`index out of range [-1]`). Aucun appelant ne le fait aujourd'hui — tous
bornent leur balayage — donc la portée est celle d'une documentation qui promet plus que le code.

### D3 — l'artefact de rejeu n'est PAS reproductible à l'octet

Mesuré au gate de clôture, en reconstruisant deux fois de suite le MÊME film avec le MÊME
binaire :

| couche | deux constructions successives |
|---|---|
| traces, tirs, inventaire, couverture | **identiques** |
| projectiles | **différents** (201 des deux côtés, contenu permuté) |
| lancers de grenade | **1 position sur 130 diffère** (film `01e1f945`) |

Cause : `ScanFilmWorldObjects` regroupe les vies dans une **map** puis les trie par instant de
premier point avec un `sort.Slice` **instable** — l'ordre des ex æquo dépend donc de l'ordre
d'itération de la map, qui est aléatoire en Go. Cet ordre se propage à `projectileBirths`, donc
au choix de la naissance la plus proche d'un lancer, donc à la POSITION publiée de ce lancer.

**Ce que cela n'invalide pas** : tous les chiffres du §5 de la réconciliation sont stables
(475/519, 90/105, 70/70, 439, 184, 99/29 221) — le gate porte sur eux, et il est passé. Les
goldens de J2 ne flottent pas non plus : l'étage 1 rejoue des entrées FIGÉES, donc un ordre figé.

**Conséquence mesurée sur la régénération, et vérifiée** : le fixture d'entrées **n'est pas
reproductible à l'octet** non plus (deux régénérations successives depuis le même film donnent
deux empreintes différentes) — c'est la même cause. En revanche le golden d'assemblage **survit**
à une régénération : il a été rejoué contre un fixture fraîchement régénéré, et il passe. La
raison est structurelle, et c'est ce qui rend le résultat solide plutôt que chanceux : le golden
rend des COMPTES et des verdicts, pas l'ordre des trajectoires.

**Ce que cela invalide** : l'idée qu'on pourrait comparer deux artefacts par leur empreinte.
À traiter avec le lot 3 (un tri total sur `(slot, gen, instant)` suffirait).

### D4 — huit familles d'effet de tir, mais SEPT géométries

`plain` (arme hors catalogue) et `ballistic` émettent exactement les mêmes primitives, dans le
même ordre ; seuls le poids du trait (1 px contre 1,6) et l'opacité (0,7 contre 0,9) les
séparent. Le rendu « neutre » est donc un balistique aminci, pas une forme qui n'affirme rien —
alors que l'en-tête de `shotEffects.ts` promet qu'une arme inconnue « ne tombe jamais sur un
rendu approchant ». Un test fige l'état actuel pour qu'il cesse d'être invisible.

### D5 — `js-yaml` est une dépendance fantôme d'`apps/web`

Elle est déclarée dans `overrides` (donc épinglée comme dépendance transitive d'`openapi-typescript`)
et **aucun fichier du dépôt ne l'importe**. La première version du test de contrat web s'appuyait
dessus ; elle a été réécrite pour ne dépendre que des types générés. À statuer avec le lot 3.

---

## CE QUI RESTE OUVERT, ET QUI N'EST PAS DANS CE PLAN

- **Les zones nommées et les objectifs vivants** (rangs 3 et 4 du `SUIVI_REPLAY_2D.md`). La
  contrainte que vous aviez posée tient : une règle valable pour les 30 cartes, pas pour
  Cliffhanger seule.
- **Le compteur d'utilisations de capacité** : 36 006 positions testées, aucune ne reproduit le
  relevé. Reste « ? » à l'écran.
- **Le dénominateur de la jauge de charge** : propre à l'arme, absent du film. Il faudrait une
  table par arme — donc une source hors film.
- **`ownersFromLives` tranche** sur collision de slot (le premier gagne). 0 collision sur le film
  de référence ; à reposer sur le second.

---

## PROTOCOLE DE REPRISE

1. Relire le contrat `plan-execution`.
2. Lire `HANDOFF_REPLAY_2D_2026-07-29.md`, puis ce fichier.
3. Reprendre à la **première case non statuée**, en respectant l'ordre des lots : le lot 2
   protège les lots 3 et 4, et le lot 1 est ce qui rend le travail visible.

**Statuts autorisés** : `[x]` fait et vérifié · `[~]` couvert ailleurs, avec la référence ·
`[!]` non traité, avec justification écrite.
