# Sonde P4 — entités d'équipe, bots et places sur b1ad85eb (2026-09-23)

Plan : `.ai/V7.5/PLAN_RETOURS_REJEU_2026-09-23.md` §4.3 (P4), prépare M2. Annexe de départ :
`RAPPORT_equipes_b1ad85eb.md` §1.2, §2, §4 (sonde à jouer avant de coder).
Légende : **MESURÉ** (sortie chiffrée de l'instrument), **DÉDUIT**, **HYPOTHÈSE**.

## Cadre

- Un seul film (b1ad85eb, arène 4v4, 29 chunks), sous la voie film, lecture seule, aucun artefact
  écrit. Trois passes courtes (faits : 0,02 s ; film : 2,5 s ; BOT_METADATA : 0,01 s).
- Horloge commune : frame = (ts − origine) / 100 ms, origine = premier horodatage de position des
  faits = **1 576 674 705 µs** (même horloge que les documents publiés : corps 564 à f3236 comme
  l'annexe).
- Instruments (`//go:build research`) :
  - `replay/p4_equipes_faits_research_test.go` (`TestP4EquipesFaits`, `P4_FAITS`, `P4_CATALOGUE`) :
    origine, table de chunk_00, vies (création → index de participant lu, fenêtre de positions),
    jointure tirs × vies par index de participant (tolérance 2 frames) ;
  - `grammar/p4_entites_ti9_research_test.go` (`TestP4EntitesTi9`, `P4_FILM`, `P4_ORIGIN_US`) :
    même lecteur que `sieges_remplacants_research_test.go` (`noterEntitesDuPaquet`, marche
    d'image-clé de production), par image-clé et par entité, trous compris ; tirs × fenêtres
    d'entités ;
  - `grammar/sieges_remplacants_research_test.go` : `TestSiegesDesRemplacants` accepte
    `SIEGES_FILM=<film complet>` (rejoué tel quel sur b1ad85eb) ;
  - `facts/killsource/p4_botmeta_horodatage_research_test.go` (`TestP4BotMetaHorodatage`) :
    chaque paquet BOT_METADATA (type 12) avec son instant, lu par `scanBotEntries`.
- Étalonnage de la jointure tirs × entités : les sept index de tireur tenus par un occupant de
  bout en bout (0, 1, 2, 3, 4, 6, 7) sont couverts à **100 %** par l'entité de MÊME index
  (131/131, 96/96, 224/224, 195/195, 126/126, 184/184, 96/96). Jointure tirs × vies : l'index
  propre est le meilleur candidat pour chacun (0 hors vie pour 0, 4, 7 ; 1 → {0, 1} ; 3 : 194/1 ;
  2 : 215/9 ex æquo avec 6 ; 6 : 164/20). Le critère « 0 tir hors vie » est donc trop strict en
  général (tirs hors positions de bipède, véhicules probablement), le critère « couvert par une
  entité » ne l'est pas.

Commandes (depuis `apps/go-api`, sous la voie film, `GOCACHE` privé, `CGO_ENABLED=0`) :
`P4_FAITS=<data>/cache/film_facts/halo_infinite/b1ad85eb.filmfacts.bin P4_CATALOGUE=<data>/titles/halo_infinite/reference/map_quant_bounds.json go test -tags research -count=1 -run '^TestP4EquipesFaits$' -v ./internal/games/halo_infinite/film/replay/`
`P4_FILM=<data>/cache/film_chunks/b1ad85eb SIEGES_FILM=<idem> P4_ORIGIN_US=1576674705 go test -tags research -count=1 -run '^(TestP4EntitesTi9|TestSiegesDesRemplacants)$' -v ./internal/games/halo_infinite/film/internal/grammar/`
`P4_FILM=<idem> P4_ORIGIN_US=1576674705 go test -tags research -count=1 -run '^TestP4BotMetaHorodatage$' -v ./internal/games/halo_infinite/film/internal/facts/killsource/`

## Verdict : l'hypothèse de l'enquête est CONFIRMÉE sur les trois points (MESURÉ)

| Attendu (annexe §4) | Mesuré |
|---|---|
| trois entités d'index 8, désignateurs 0, 0, 1 | **oui** : slots 1530 (des 0), 1687 (des 0), 2145 (des 1) |
| fenêtres ≈ [0, 353], [774, 911], [3236, fin] | [f12..f212], [f813..f813], [f3213..fin] (strictes, pas de 20 s ; voir table) |
| entité d'index 1 absente dès l'image-clé qui suit f3117 | **oui** : dernière image-clé f3013, absente à f3213 |
| aucune entité d'index 5 | **oui** : 0 entité, 0 corps, 0 création d'index 5 |
| 207 records sur 26 paquets = 25 × 8 + 7 | **oui** : 27 images-clés dont la première (f−186, préambule) vide ; 207 vus = 207 lus ; l'image-clé à 7 est f613 |

### Table entité → (index, désignateur, images-clés, occupant)

Images-clés porteuses : f12, f212, f412, f613, f813, f1013 … f5014 (pas de 200 frames). « Large » =
entre les deux images-clés voisines, exclues.

| Entité (slot ti=9) | Index | Dés. | Première / dernière image-clé | Fenêtre large | Trous | Occupant (lien) | Corps liés (création) |
|---|---|---|---|---|---|---|---|
| 1297 | 0 | 0 | f12 / f5014 | ]début..fin[ | **rang 4 (f613)** | MONEY x BUTTER (table 0) | 513, 521, 530, … |
| 1299 | 1 | 1 | f12 / **f3013** | ]..f3213[ | — | FairyNectar5788 (table 1) | 514 … 548 (dernière vie → f3117) |
| 1301 | 2 | 1 | f12 / f5014 | fin | — | Madina97294 (table 2) | |
| 1303 | 3 | 0 | f12 / f5014 | fin | — | Namikidori (table 3) | |
| 1305 | 4 | 1 | f12 / f5014 | fin | — | Chocoboflor (table 4) | |
| 1309 | 6 | 0 | f12 / f5014 | fin | — | DRghie (table 6) | |
| 1311 | 7 | 1 | f12 / f5014 | fin | — | JGtm (table 7) | |
| 1530 | 8 | 0 | f12 / f212 | ]f−186..f412[ | — | **343 Hundy** (bid 16) — BOT_METADATA f12..f271, `nbBots=0` à **f273** | 512 (f0, positions → f270) |
| 1610 | 9 | 0 | f412 / f613 | ]f212..f813[ | — | Hanover Cat (arrivant, pas dans la table) | 523 (f661, 0,2 s) |
| 1687 | 8 | 0 | f813 / f813 | ]f613..f1013[ | — | **343 PardonMy** (bid 7) — BOT_METADATA f813..f829, `nbBots=0` à **f831** | 526 (f774, positions → f829) |
| 1713 | 10 | 0 | f1013 / f5014 | ]f813..fin[ | — | Claudors (arrivant) | 532 (f1184), 534, 572, 576, 582 |
| 2145 | 8 | 1 | f3213 / f5014 | ]f3013..fin[ | — | **343 Brew Dog** (bid 19) — BOT_METADATA dès **f3155** (paquet de changement en milieu de chunk 17) jusqu'à la fin | 564 (f3236), 569, 573, 577, 585, 594 |

Occupants par image-clé (MESURÉ) : 4 + 4 à chacune des 25 images-clés pleines, **3 + 4 à f613**
(MONEY manquant). Par désignateur, jamais plus de 4. Eagle (des 0) : 0, 3, 6 + la 4e place tenue
successivement par 1530 (Hundy) → 1610 (Hanover Cat) → 1687 (PardonMy) → 1713 (Claudors), fenêtres
disjointes. Cobra (des 1) : 2, 4, 7 + la place de 1299 (FairyNectar, → f3013) → 2145 (Brew Dog, f3155).

Le trou de MONEY à f613 n'est PAS un départ (DÉDUIT) : son corps 521 a 3 071 positions continues sur
[f425..f958], l'entité 1297 réapparaît à f813 avec le même slot, et la marche n'a vu que 7 records
ti=9 dans cette image-clé (le plus bas slot ti=9, 1297, est celui qui manque : cohérent avec la perte
de préfixe de P2 — HYPOTHÈSE sur le sous-mécanisme exact, non tracée ici).

### BOT_METADATA : un état, réécrit à chaque tête de chunk et à chaque changement (MESURÉ)

41 paquets type 12 : un en tête de CHAQUE chunk (paquet 4, juste après l'image-clé du paquet 1) plus
des paquets de changement en milieu de chunk (f270-273, f693, f829-833, f3155). Un paquet de 4 octets
(`nbBots=0`) dit « plus aucun bot ». Conséquences :
- **départ d'un bot = instant exact** du premier paquet qui ne le déclare plus (Hundy f273, PardonMy
  f831) — à la frame près, là où l'image-clé ne date qu'à 20 s près ;
- **arrivée d'un bot** : exacte quand un paquet de changement l'annonce (Brew Dog f3155, 58 frames
  AVANT son image-clé f3213 et 81 avant son corps f3236) ; sinon bornée par la tête de chunk
  suivante (PardonMy : premier paquet à f813 alors que son corps 526 naît à **f774** — le corps
  précède la déclaration, MESURÉ). Un corps ne se lie donc pas à un bot par « BOT_METADATA vivant
  à sa création » : il se lie à l'ENTITÉ (même index, création dans la fenêtre large), et l'entité
  au bot (intervalle BOT_METADATA ∩ fenêtre stricte). Sur b1ad85eb les trois fenêtres larges
  d'index 8 sont disjointes : chaque lien est unique (512 → 1530 → Hundy, 526 → 1687 → PardonMy,
  564…594 → 2145 → Brew Dog).
- le paquet de changement à f693 (`nbBots=0`, aucun bot déclaré avant lui depuis f273) n'a pas de
  sens établi — HYPOTHÈSE : réécriture sur un changement de joueur humain (départ de Hanover Cat,
  dont le corps s'arrête à f662 et l'entité à f613/f813). Non tranché, non nécessaire à M2.

### Règle des places sur les tirs (MESURÉ)

- Index de tireur 5 : **84 tirs [f1284..f4694]**, 0 couvert par une entité d'index 5 (il n'y en a
  pas) ; parmi les entités ARRIVANTES, **une seule couvre les 84** : 1713 (index 10, des 0 =
  Claudors). Jointure avec les vies : seul l'index de participant 10 a 0 tir hors vie (84/0) ; le
  suivant, 0, en a 3 hors vie. La place 5 (siège de WNBA Fan A5, jamais joué) est donc tenue par
  Claudors, et le film l'écrit dans les tirs.
- **Les bots n'écrivent AUCUN tir long (record 105) sous leur place ni sous leur index** (MESURÉ) :
  0 tir d'index ≥ 8 ; 0 tir d'index 5 pendant [f12..f831] (Hundy, PardonMy) ; 0 tir d'index 1 après
  f3108 alors que Brew Dog vit 170 s sur la place 1 (6 vies, f3236..f4935). HYPOTHÈSE (non
  vérifiée) : leurs tirs passent par la variante courte du 105 ou un autre canal. La place d'un bot
  ne se lit donc PAS dans les tirs.
- Hanover Cat (0,2 s de vie) n'a aucun tir : sa place n'est pas lue non plus.

Ce qui est MESURÉ pour la règle de l'utilisateur (« un partant libère sa place pour son remplaçant,
un humain remplace le bot ») : Claudors, humain, hérite de la place 5 (tirs) ; les quatre occupants
successifs de la 4e place d'Eagle ont des fenêtres disjointes et le même désignateur ; Brew Dog
entre (f3155) entre la dernière vie de FairyNectar (f3117) et la première image-clé sans elle
(f3213). Que Hundy, Hanover Cat et PardonMy aient tenu la place **5** (et non une autre) est DÉDUIT
(chaînage : seule place libre d'Eagle), pas lu.

### Retards de l'API sur le film (MESURÉ côté film, horloge annexe `f = s × 10 + 227`)

| Relais | API | Film |
|---|---|---|
| Hundy sort | 0:34.0 (f567) | f273 (BOT_METADATA), corps → f270 |
| Hanover Cat entre / sort | f566 / f988 | entité ]f212..f412], corps f661-662, sortie ]f662..f813[ |
| PardonMy entre / sort | f988 / f1125 | corps f774, sortie **f831** (BOT_METADATA) |
| Claudors entre | 1:29.7 (f1124) | entité ]f813..f1013], corps f1184 |
| FairyNectar sort / Brew Dog entre | 5:22.3-5:22.4 (f3450) | FairyNectar ≤ f3155 ; Brew Dog **f3155** |

## `TestSiegesDesRemplacants` rejoué tel quel (MESURÉ)

12 entités, 4 départs avant le dernier paquet, « 1 index tenu par plusieurs entités », verdict
« siège écrit directement = true », occupants par paquet `0 8 8 … 8`, « fiches en trop si on affiche
tout le roster = 4 ». **Piège pour M2** : le « siège repris » qu'il rapporte est l'index 8 PARTAGÉ
DES BOTS (1530 → 1687 → 2145, dont deux équipes), pas une place héritée ; sur ce film aucun humain ne
reprend l'index d'un partant. Le verdict ne doit pas se lire « le film écrit la place dans ti=9 ».

## Ce que M2 doit lire

1. **M2.1 entités** : le balayage ti=9 PAR ENTITÉ tient (12 entités, désignateur constant, index
   constant, 0 instable). La première image-clé du film peut être VIDE (préambule, f−186) : « présent
   au départ » = présent à la première image-clé PORTEUSE, pas au rang 0. Un TROU au milieu de la
   fenêtre d'une entité n'est pas un départ (MONEY, f613) : une entité n'est jamais réutilisée, sa
   présence est [première, dernière] image-clé, trous comblés et comptés.
2. **Lien bot → entité → corps** : BOT_METADATA doit garder l'instant de chaque paquet (aujourd'hui
   `loadBotMeta` déduplique et perd tout instant) et rendre, par bot, l'intervalle [premier paquet
   qui le déclare, premier paquet suivant qui ne le déclare plus). Bot → entité par intersection avec
   la fenêtre STRICTE de l'entité de même index ; corps → entité par index + création dans la
   fenêtre LARGE (pas par l'intervalle BOT_METADATA : le corps de PardonMy le précède de 39 frames).
   Unicité à vérifier et compter (repli nommé si deux entités candidates).
3. **Présence** : `from` / `to` des entités à 20 s près, AFFINÉS pour les bots par BOT_METADATA
   (départ exact : f273, f831 ; arrivée exacte quand un paquet de changement existe : f3155) ; le
   départ d'un humain remplacé par un bot est borné par l'arrivée du bot (FairyNectar ≤ f3155).
4. **Équipe** : par entité (désignateur), jamais par index ; les trois bots d'index 8 → Eagle,
   Eagle, Cobra = l'API (t0, t0, t1). Le contrôle par index (`divIndex=1` sur l'index 8) reste un
   compteur.
5. **Place** : table de chunk_00 pour les occupants du départ ; **tirs** pour un remplaçant humain
   qui tire (Claudors → 5) ; **chaînage par équipe** pour les bots (ils n'écrivent aucun tir long)
   et pour un arrivant sans tir (Hanover Cat) — repli nommé, compté. Un siège de la table SANS
   entité (WNBA Fan A5, place 5) est une place libre dès la première image-clé porteuse, occupée par
   le premier arrivant de même équipe (Hundy) ; l'équipe de cette place ne se lit que par ses
   occupants (WNBA n'a ni entité ni désignateur).
6. **Gate du témoin** : à chaque image-clé porteuse, occupants par désignateur ≤ 4 (mesuré : 4 + 4
   partout sauf 3 + 4 à f613, trou de marche) ; place 5 d'Eagle : Hundy → Hanover Cat → PardonMy →
   Claudors ; place 1 de Cobra : FairyNectar → Brew Dog.

## Hors périmètre (noté, non traité)

- Les tirs des bots sont absents des records 105 longs (0 sur ce film) : à instruire hors M2 si la
  précision ou le rejeu des bots doivent en dépendre.
- Le trou d'entité à f613 (MONEY) est un effet probable de la perte de préfixe de la marche d'image-clé
  (P2, M3) : M3 devrait le faire disparaître ; M2 ne doit pas en dépendre.
- Paquet BOT_METADATA de changement à f693 sans bot : sens non établi.

Aucun arrêt au titre de la règle du 22/09 : l'hypothèse est confirmée, aucun négatif.
