# Sonde P3 — les armes de naissance (2026-09-23)

Plan : `.ai/V7.5/PLAN_RETOURS_REJEU_2026-09-23.md` §4.3 (P3), prépare M3.2. Annexe de départ :
`RAPPORT_fiche_armes.md` §2.3, §4 (S2).
Légende : **MESURÉ** (sortie chiffrée de l'instrument), **DÉDUIT**, **HYPOTHÈSE**.

## Cadre

- Films lus un à un, sous la voie film, lecture seule, aucun artefact : 81c02726 (Arena, Isolation),
  b1f01a33 (Super Fiesta, Prism — départs aléatoires, le test décisif), a0c36016 (CTF, Forest).
  Oracles : le document publié du même match (`data/cache/replays/halo_infinite/<id>.json`).
- Instruments (paquet `grammar`, `//go:build research`) :
  - `p3_armes_naissance_research_test.go` — marche de la vue B (`decodeInferLoop` via `t519Marcher`,
    monde lié comme la sonde P1) avec un `HeldWeaponHook` posé ; lecture du document et des oracles ;
  - `p3_armes_naissance_recherche_research_test.go` — recherche d'en-tête EXACT du record NEW ti=35 de
    chaque vie, témoin de la recherche, position du record dans le paquet, localisation des familles
    des oracles, chaînage grammatical ;
  - `p3_armes_naissance_catalogue_research_test.go` — lecture par catalogue (sans oracle de vie) ;
  - `p3_armes_naissance_rapport_research_test.go` — `TestP3ArmesNaissance`, couvertures et accords.
- Horloge : origine = premier paquet du chunk 1 + `originMs` du document (formule de
  `resolveOriginMs`).
- Oracles : **O1** première entrée `loadouts` de la vie sans prise intermédiaire (aucun
  `weaponChanges` du slot entre la naissance et l'image-clé, aucun `pickups` d'arme après naissance
  + 1 frame) ; **O2** premier tir de la vie sans prise intermédiaire ; **TÉMOIN** la lecture d'une vie
  contre l'oracle de la vie SUIVANTE (autre slot) ; **HASARD** l'accord moyen sur toutes les paires.

Commande (depuis `apps/go-api`, sous la voie film) :
`MOUV511_FILM=<data>/cache/film_chunks/<id> MOUV511_BORNES=<data>/titles/halo_infinite/reference/map_quant_bounds.json MOUV511_CARTE=<isolation|prism|forest> P3_DOC=<data>/cache/replays/halo_infinite/<id>.json go test -tags=research -count=1 -v -run '^TestP3ArmesNaissance$' ./internal/games/halo_infinite/film/internal/grammar/`
(quelques secondes par film).

## Verdict : le record NEW du bipède PORTE les armes de naissance — aucun chemin de production ne les lit juste

### 1. La marche de production ne voit pas les naissances (MESURÉ)

| film | vies | naissances rendues par la marche de la vue B, rattachées à une vie |
|---|---:|---:|
| 81c02726 | 45 | 3 (1 avec une arme présente, fausse) |
| b1f01a33 | 107 | 6 |
| a0c36016 | 142 | 3 |

Cause (MESURÉ, `p3SituerDansMarche`) : le record NEW de naissance est dans un paquet À LISTE
d'événements, AVANT le début de vue B que `marchLocate` choisit — 41/45, 99/107, 125/142 records.
La liste d'événements de ces paquets, marchée jusqu'à son terminateur par le marcheur R7 de la sonde
P1, **finit exactement sur l'en-tête du NEW** dans 40/41, 91/99, 123/125 cas (événement commun :
type 82 `PlayerGameEventSmall`, présent dans chacun). Le NEW est donc le PREMIER record de la vue B ;
le localisateur par signature (slot 123) démarre plus loin et le saute.

### 2. Le record NEW existe à la frame de naissance (MESURÉ)

Recherche d'en-tête exact (`[0][01][idLow][tag][R(6)=35]`, slot de la vie) dans les paquets delta de
[début − 1 s, début + 5 s], puis lecture :

| film | vies avec un record NEW lu | en-têtes au témoin (slot cherché à l'instant d'une autre vie) | dont lus par catalogue |
|---|---:|---:|---:|
| 81c02726 | 45/45 | 4 | 0 |
| b1f01a33 | 106/107 | 12 | 0 |
| a0c36016 | 142/142 | 7 | 0 |

La vie manquante de b1f01a33 (slot 571, 3178..3178) est une vie d'UN point (classe M1/Q15). Les
en-têtes fortuits du témoin (22 bits fixes) ne donnent jamais de lecture d'arme.

### 3. La traversée de production arrive désalignée sur i43 (MESURÉ)

`TraverseEntity` sur ces records : « propre » dans 35/46, 88/115, 107/150 cas, mais les familles lues
en i43..i46 sont fausses — accord avec O1 **0/17, 0/55, 0/62**, avec O2 0/18, 0/40, 1/84. Les
familles des oracles sont pourtant dans le record (localisation d'un mot de 32 bits, P3.6) :
69/69, 168/173, 166/169 occurrences, à une position quasi fixe par film (porte du 1er emplacement à
l'en-tête +1327/+1336 sur 81c02726, +1328/+1360 sur a0c36016, +1311..+1388 sur b1f01a33), alors que
la production place i43 entre −599 et +731 bits de là (une occurrence à +2128 sur a0c36016). Le
désalignement naît dans les composants i1..i42 du record NEW (exemple 81c02726 slot 516 : i15
`object-low-frequency` y est lu sur 552 bits, contre 50 à 81 bits chez les slots 513 à 515). Le « VALIDÉ BIT-EXACT » de `traverse.go` ne couvre que l'état par défaut, pas ce corps.

### 4. Depuis la bonne position, le désérialiseur de production enchaîne les emplacements (MESURÉ)

En partant de la porte qui précède la première famille localisée, `consumeWeaponStateTypeInfoVariant`
lit la famille puis s'arrête EXACTEMENT sur la porte de l'emplacement suivant : 31/31, 79/79, 60/61
records (l'échec d'a0c36016 est un oracle défectueux, §6). Forme mesurée : trois emplacements
annoncés au masque (i43, i44, i46) ; à la réapparition [arme 1 (170 bits), arme 2 (170 bits), vide
(17 bits)] ; au coup d'envoi [arme 1 (162), arme 2 (162), `00007CA9` (107)] — `00007CA9` est aussi le
`w` des `pickups` d'arme à t=0 de b1f01a33 (HYPOTHÈSE : objet de départ non-arme, poings ou
équivalent ; absent du catalogue `weaponLabels`, 8 records par film).

### 5. Taux et accords — lecture par catalogue, sans oracle de vie (MESURÉ)

Lecture : dans le record trouvé, le bit de [+1000, +1700) où l'enchaînement des emplacements annoncés
donne le plus de familles du CATALOGUE du film (`weaponLabels`, toutes armes du match), le premier
emplacement étant présent et au catalogue. C'est un instrument de mesure, PAS une lecture à produire
(§ impact).

| film | naissances lues | O1 ensemble | O1 ordre des emplacements | O2 premier tir | témoin O1 / O2 | hasard O1 / O2 |
|---|---:|---:|---:|---:|---:|---:|
| 81c02726 (Arena) | 45/45 (100 %) | 31/31 (100 %) | 31/31 | 32/32 (100 %) | 100 % / 100 % | 100 % / 100 % |
| **b1f01a33 (Super Fiesta)** | **106/107 (99,1 %)** | **80/81 (98,8 %)** | 80/81 | 61/63 (96,8 %) | **0 % / 9,5 %** | **0,6 % / 9,1 %** |
| a0c36016 (CTF) | 142/142 (100 %) | 61/63 (96,8 %) | 61/63 | 88/89 (98,9 %) | 96,8 % / 98,9 % | 96,8 % / 98,9 % |

- Super Fiesta est le seul film discriminant (dotation tirée au sort) : accord 98,8 % contre 0,6 %
  au hasard et 0 % au témoin. Sur Arena et CTF, dotation fixe : le hasard vaut l'accord, ces films
  ne mesurent que la couverture et la cohérence.
- Ordre : l'accord « ordre des emplacements » égale l'accord « ensemble » partout — i43 = première
  arme de `loadouts[].w`, i44 = la seconde (MESURÉ).

### 6. Les désaccords (MESURÉ sur le document ; cause DÉDUITE ou HYPOTHÈSE)

- b1f01a33 slot 528 (O2) : les « tirs » à t 889-891 portent `1833A5A8592CF3E9` (moitié basse
  différente de `42C9679F`, sans libellé) ; le premier tir d'arme personnelle (t 908) est le Crémateur
  lu à la naissance. DÉDUIT : oracle faux (tir d'un objet non-arme).
- b1f01a33 slot 576 (O1 et O2) : lu [Fusil traqueur, Crémateur] à t 3167 ; image-clé à t 3351 (18 s
  plus tard) [Sidekick, Crémateur], tirs au Sidekick dès 3369, aucune prise publiée. HYPOTHÈSE : prise
  non publiée (le repli `spawnSetFrom` classe `restated` et retire de vraies prises, §1.3 du plan).
- a0c36016 slots 602 et 613 (O1) : lu [AR, Sidekick] ; image-clé 24 s et 11 s plus tard avec un Sniper
  / un BR à la place, tir au BR avant l'image-clé, aucune prise publiée. Même HYPOTHÈSE.

Seuils de production du plan : ≥ 95 % des naissances lues — **tenu** (100 / 99,1 / 100 %) ; accord
≥ 98 % — **tenu sur l'oracle discriminant** (O1 Fiesta 98,8 %), O2 Fiesta 96,8 % et O1 CTF 96,8 %
en dessous, chaque écart étant un défaut d'oracle identifié (§6) ; témoin au niveau du hasard —
**tenu** (Fiesta 0 % / 0,6 %, 9,5 % / 9,1 %).

## Impact sur le plan (M3.2)

1. **Le canal existe, P3 confirme M3.2** : les armes de naissance sont dans le record NEW ti=35 de la
   naissance, emplacements `weapon-state-type-info` i43 / i44 / i46, famille = moitié HAUTE (celle des
   `loadouts`), i43 = arme 1, i44 = arme 2 (`weaponChanges[].k` 0/1 = index d'emplacement).
2. **Deux réparations de grammaire sont PRÉALABLES à tout branchement** — brancher
   `HeldWeaponHook` sur la marche actuelle ne lirait que du bruit (0 % d'accord) et presque aucune
   naissance :
   - (a) **atteindre le record** : dans un paquet à liste, la vue B commence à la fin de la liste
     d'événements, et le premier record y est le NEW de naissance (123/125, 91/99, 40/41) ;
     `marchLocate` (signature du slot 123) le saute. Il faut démarrer la vue B à la fin de liste
     (marcheur de liste en production), ou accepter ce record comme ancre ;
   - (b) **traverser le corps** : les composants i1..i42 d'un record NEW de bipède ne sont pas lus au
     bit près (désalignement de −599 à +731 bits sur i43) ; leur grammaire est à porter jusqu'à i43.
     Oracle de gate disponible : la position de la famille (porte du 1er emplacement à +1311..+1388),
     et depuis elle le désérialiseur de production enchaîne déjà i43 → i44 → i46 au bit près.
3. **Interdit comme lecture de production** : la lecture par catalogue de cet instrument (fenêtre de
   bits + score par catalogue) est une heuristique de mesure ; elle ne doit pas être portée — ni
   comme repli, sauf décision explicite (repli nommé, compté, daté).
4. **`00007CA9`** (troisième emplacement au coup d'envoi) : à exclure des armes affichées ou à
   nommer — à trancher dans M3.2 (non vu dans le catalogue des armes).
5. **Oracles de gate M3** : la dotation lue à la naissance contredit l'image-clé suivante dans 3 vies
   sur 175 dotées de l'oracle O1 (b1f01a33 576, a0c36016 602 et 613), toujours avec une prise non publiée
   entre les deux : cohérent avec le défaut `spawnSetFrom` déjà planifié dans M3.2 (reclasser les
   premières émissions contre la dotation de naissance). Test utile : ces trois vies doivent
   publier une prise après M3.2.
6. **Rien ne change dans la vague C** (republication depuis les faits) : le canal demande un
   redécodage (vague D), comme prévu.

## Découvertes hors périmètre (non traitées)

- Le record NEW de naissance de la vie de 81c02726 slot 523 (t 645 au document) est à t 691 (+4,6 s) :
  confirme le constat M1 (vie ouverte avant la création du corps).
- b1f01a33 : des « tirs » portent une arme `1833A5A8592CF3E9` sans libellé, à la naissance du tireur
  (slot 528, t 889-891) — à rapprocher des tirs publiés par erreur.
- Les en-têtes NEW ti=35 fortuits sont fréquents (4, 12, 7 dans les fenêtres témoins) : toute
  recherche d'en-tête en production demande une confirmation au-delà de l'en-tête.
