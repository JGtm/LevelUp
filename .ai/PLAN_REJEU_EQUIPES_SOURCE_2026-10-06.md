# Plan — Rejeu : toute entrée du roster a l'équipe que le film écrit (source Go) — 2026-10-06

Branche `feat/rejeu-equipes-source` (worktree `LevelUp-wt-rejeu-equipes-source`, base `origin/feat/v75`
`1518e6f10`). Superviseur : session levelup-dc. Brief : `BRIEF_REJEU_EQUIPES_SOURCE.md` (scratchpad du
superviseur), il fait foi. Lot web parallèle : `LevelUp-wt-rejeu-equipes-web` (hors périmètre ici).

Contrat d'exécution : skill `plan-execution` (ordre strict, aucun report d'une action faisable, chaque
item statué `[x]` / `[~]` / `[!]`). Périmètre de code : `film/replay` seulement ; tout besoin de toucher
`film/internal/grammar` ou `film/internal/facts` est un CHECKPOINT (retour au superviseur avant code).

Données : parc lu en LECTURE SEULE par chemin absolu dans le worktree principal
(`data/cache/{replays,film_facts,film_chunks}`), aucune jonction, aucune écriture.
Un film à la fois, en processus ; aucune génération d'artefact dans `data/`.

## Décision produit (ferme)

Une entrée du roster a TOUJOURS une équipe ; aucune section « sans équipe », aucun repli sur la feuille
de match. L'équipe se lit dans le film (ADR 0034 D-9, D-10 ; la grammaire prime). Résidu impossible =
compteur + `slog.ErrorContext`, jamais un affichage « inconnu ».

## État : lot fusionné dans `feat/v75` (`b5c9489ef`) ; Phase D (D2, D3, D6) au CHECKPOINT

Le champ d'équipe de BOT_METADATA est établi chez l'écrivain (Ghidra, F.2) et lu dans
`film/internal/facts/killsource` (G3.0) ; la publication du rejeu en fait l'équipe des bots qu'aucune
entité ne porte, et la pose des places assoit le bouche-trou sur la place du partant (G3.1, G3.2).
`killsource.Rev` ne monte pas (aucune ligne de kill ne change) ; `SchemaDesFaits` 5 -> 6 (après la vue A) et
`SchemaVersion` 79 -> 80 : les faits et les artefacts du parc sont à ré-extraire (G5, sur go).

## Étapes

### G1 — Diagnostic sur pièces des 21 entrées `seatSource: index`

- [x] G1.1 Instrument de lecture des faits persistés : `film/replay/rejeu_equipes_research_test.go`
      (tag `research`). Il REJOUE la liaison de production (`lierLesOccupants`) sur les faits et le
      roster de l'artefact, et vérifie d'abord qu'elle rend les équipes publiées (0 écart sur 126 films).
- [x] G1.2 Cause du silence d'équipe, 18/18 bots sans équipe : AUCUNE entité `ti=9` lue, parce
      qu'aucune image-clé porteuse ne tombe pendant leur déclaration BOT_METADATA (17 déclarations
      tiennent strictement entre deux images-clés porteuses consécutives, celle de Ham Sammich commence
      après la dernière ; durées de 1 à 126 frames ; pas des images-clés porteuses : 200 frames), or
      l'équipe ne se lit qu'aux images-clés. Table de contrôle : index absent pour 14 bots (11 sans aucune lecture de leur
      index, 3 sur l'index 24 divergent de `4f77afc1` et `5676a9ba`), présente mais portée par l'entité
      d'un AUTRE bot du même index pour 4 bots (`a6ae19fb`, `c7f94693` x2, `e85d7bad` : refus de
      `indexPorteParUneEntiteLiee`).
- [x] G1.3 Cause de l'absence de place : équipe inconnue -> `chainerLesArrivants` rend `index` (18/18).
      Les 3 entrées AVEC équipe : voir le tableau (présences qui se recouvrent, vies mal nommées).
- [x] G1.4 Chaîne correcte de chaque place : tableau ci-dessous (partant -> bot -> arrivant, équipes
      concordantes des deux côtés sur toutes les chaînes déterminables).

| Match | Entrée (index) | Équipe publiée | Présence | Chaîne de la place (équipe) | Remarque |
|---|---|---|---|---|---|
| 43716616 | 343 Sandwolf (8) | — | 248-281 | 5 (0) : Slowpoke6743 (certain -> 118) -> bot -> KernelPanic10 (318) | KF 118 / 318 |
| 4ecdf3e7 | 343 Cliffton (8) | — | 1236-1246 | 1 (0) : lil conqueror23 (-> 1179) -> bot -> Wrneverchanges (1284) | KF 1084 / 1284 |
| 4f77afc1 | 343 Connmando (24) | — | 3353-3363 | 26 (1) : Narotlcs (-> 3352) -> bot -> entrée sans nom idx 31 (3538) | l'artefact montre la place 1 vide de 934 à 10144, mais le film y lit une entité d'équipe 1 non liée (slot 1836, KF 335..9944) : place 26 seule libre (cf. D8) |
| 4f77afc1 | 343 Darkstar (24) | — | 2219-2220 | 2 (0) : AJM002 (-> 2136) -> bot -> Feelgood Joker (2336) | même remarque : place 2 seule libre selon le film |
| 5676a9ba | 343 Rhinosaurus (24) | — | 3675 | 1 (0) : EddieFingerTits (-> 3590) -> bot -> fur1ousjoe (3759) | KF 3559 / 3759 |
| 572e236b | 343 Bachici (8) | — | 582-594 | 7 (0) : biOly goLab1054 (-> 582) -> bot -> DRghie (695) | dernière vie du partant finie à la frame de la déclaration |
| 72b0a25e | 343 Rhinosaurus (8) | — | 150-162 | 3 (1) : CafeMiroir64382 (-> 80) -> bot -> Patatasxd3526 (214) | KF 14 / 214 |
| 859da825 | 343 Forge Lord (8) | — | 1709-1826 | 2 (0) : opresko (-> 1627) -> bot -> SplinterCell958 (1827) | KF 1627 / 1827 |
| 859da825 | SplinterCell958 (9) | 0 | 1827-4338 | même chaîne | place 2 refusée : vie de 4 frames [3167..3170] (slot 548) nommée opresko par création (index 2) après l'absence prouvée de son entité |
| 879a4dba | 343 Beard (24) | — | 1962-1985 | 19 (1) : jas0n38 (-> 1955) -> bot -> RezzaCapa (2155) | KF 1955 / 2155 |
| 9ffce8ef | 343 SpaceCase (8) | — | 856-857 | 4 (1) : Fxrdzy (-> 739) -> bot -> AreolaAcrobat (939) | KF 739 / 939 |
| a6ae19fb | 343 Hollis (8) | — | 352-431 | 3 (0) : KNEELb3foreZOD (-> 323) -> bot -> entrée sans nom idx 9 (523) | KF 323 / 523 |
| b0fe12b1 | 343 Ben Desk (8) | — | 1697-1707 | 7 (1) : JALiiSCKiiO (-> 1687) -> bot -> MrNormalMrs (1887) | KF 1687 / 1887 |
| c7f94693 | 343 Donos (8) | — (sa vie : 0) | 865-983 | 7 (1) : JGtm (-> 859) -> bot -> Aceshigh949851 (1015) | l'équipe 0 publiée sur sa vie est empruntée à l'index 8 (Byrontron) : FAUSSE |
| c7f94693 | 343 The Thumb (8) | — | 1937 | 6 (0) : SirAvlas (-> 1818) -> bot -> DJTuna77 (2015) | KF 1815 / 2015 |
| d1dfbc02 | 343 Ham Sammich (8) | — | 5947-5986 | indéterminable par les places : un occupant parti après la dernière KF porteuse (5870) -> bot | départ non lu : les 8 occupants sont publiés présents jusqu'à 5986 ; seul le champ BOT_METADATA dit l'équipe (0) |
| e85d7bad | 343 Total Ten (8) | — | 843-854 | 3 (0) : Freeskyfall (-> 788) -> bot -> Stock7981 (965) | KF 765 / 965 |
| f0220a96 | 343 Chilies (8) | — | 1577-1585 | 0 (0) : M0NSTER IlI (-> 1555) -> bot -> Ginger Ninja yt (1679) | KF 1479 / 1679 |
| f2966f08 | 343 Ritzy (8) | — (vies : -1) | 138-263 | 6 (1) : Drowsy Grim (-> 94) -> bot -> SputNick193114 (294) | KF 94 / 294 |
| 43e96765 | 343 PardonMy (8) | 0 (entité KF 1145) | 1136-1344 | 0 (0) : LeodaganQC (-> 1046) -> bot -> Cmillward21 | Cmillward21 lu à la KF 1145, premier corps à 1367 : il attend le retrait du bot (1344) ; place 0 prise par ses tirs dès 1145 -> bot sans place, 209 frames de dépassement |
| bf2a9f05 | AllGodsLove (8) | 0 | 1004-1019, 1824-3056 | 3 (0) : yolojoe13 (-> 1624) -> AllGodsLove (1824) | [1004..1019] = vie (slot 529) nommée par l'index 8 pendant la déclaration du bot Mickey [1008..1023], hors roster ; aucune place libre sur les deux intervalles |

« (-> N) » : fin de la présence CERTAINE publiée du partant. « KF a / b » : images-clés porteuses qui
encadrent la déclaration.

### G2 — Où le film écrit-il l'équipe de ces bots ?

- [x] G2.a Entités `ti=9` : 0/21 aux images-clés (structurel, cf. G1.2). Trames delta, instrument de la
      grammaire (non versionné, rendu au superviseur), MESURÉ SUR 2 FILMS SEULEMENT (`43716616` et
      `a6ae19fb`, tous deux `fo08_wetland`) : avant le coup d'envoi de `43716616`, la marche lit les 8
      créations `ti=9` (slots 1297..1311, index 0..7) avec exactement les désignateurs de la table des
      images-clés (8/8) ; en cours de match, aucune des 8 créations `ti=9` connues par leur entité
      (slot 1894 ; slots 1536, 1808, 1830, 1908, 1916, 1970, 1974) n'est lue, et les NEW `ti=9` que la
      marche y lit portent des slots hors de la séquence d'allocation du film (slot 758 « index 9 » quand
      l'entité de cet index est le slot 1894 ; slot 5764 « index 8 » à la frame 371, dans la déclaration
      de Hollis, sans i0). Sur ces 2 films (2 bots sans entité, 3 bots de contrôle), la création des bots
      n'est pas lisible aujourd'hui ; non établi pour les 16 autres films (cartes et largeurs d'axe
      différentes). G2.d n'en dépend pas.
- [x] G2.b Bipède : l'équipe n'y est pas décodée (corps du bloc MPP `i9` sauté, `ecs_table.tsv`) ; 15/18
      bots n'ont aucune vie ; les équipes publiées sur les 3 vies de bot viennent du pont slot -> index
      (Donos 0, emprunté ; Forge Lord et Ritzy -1).
- [x] G2.c BOT_METADATA : TROUVÉ. Oracle : l'équipe des 56 bots du parc (28 + 28, 36 films) que leur
      entité `ti=9` donne, liée par leurs déclarations ; 55 de ces bots sont sur HI_1_13_0, un seul sur
      HI_1_12_0 (`343 KaleDucky`, équipe 0) : le champ est établi sur HI_1_13_0 seulement, et les 21 bots
      sans entité sont tous HI_1_13_0. Positions relatives au
      DÉBUT de l'entrée : aucun champ de 1 à 8 bits ne vaut l'équipe (meilleur accord 36/56). Positions
      relatives à la FIN DU NOM (terminateur exclu) — l'entrée est de longueur variable, paquets à un bot
      de 2070 à 2086 octets, soit une queue fixe de 1930 octets après le nom : les octets 0x783 et 0x784
      valent 0x00 en équipe 0 et 0x04 en équipe 1, sur 56/56 bots ; aucun bot ne voit ces octets changer
      d'un paquet à l'autre (jusqu'à 29 paquets) ; 0x783 = 0x784 sur les 77 bots ; un même bot change de
      valeur d'un match à l'autre avec son équipe (`343 PardonMy` : 1, 0, 0, 1). Contrôle indépendant
      par la règle des places sur les 21 bots sans entité : sur les 20 dont une place libre détermine
      l'équipe (16 à place unique, dont Connmando et Darkstar une fois la place 1 de `4f77afc1` rendue à
      l'entité d'équipe 1 que le film y lit ; Bachici à une frame près ; Oscar, BF Scrub et Mickey hors
      roster), 20/20 accords — 19/19 si l'on s'en tient aux places publiées par l'artefact (Connmando y a
      deux places libres d'équipes différentes, Darkstar deux places d'équipe 0). Le champ vaut
      0 pour 13 bots et 1 pour 8 : l'accord ne vient pas d'une valeur constante. Ham Sammich :
      indéterminable par les places, le champ dit 0. L'écrivain du paquet n'a pas été lu dans Ghidra.
- [x] G2.d Verdict : la source existe dans le film (BOT_METADATA) et se lit dans la couche des faits, pas
      dans `film/replay` -> CHECKPOINT (i). Retour au superviseur.

### Levée du CHECKPOINT (i) par le superviseur (2026-10-06)

Décision : lire le champ à la source, grammaire d'abord, aucune déduction par les places ; pas de
republication. Ordre imposé : fusion de `feat/v75`, Ghidra (lecture seule), champ dans killsource,
G3, mesure de la durée de ré-extraction sur UN film.

- [x] F.1 Fusion de `origin/feat/v75` (`f8a14b3b9`, lot LR de la campagne et lot web des équipes) :
      `e0cd93662`, automatique, sans conflit.
- [x] F.2 Ghidra (HTTP 127.0.0.1:8089, `HaloInfinite.exe`, lecture seule, analyse au repos) :
      - ÉCRIVAIN du paquet : `FUN_14299bda0` pose le type `0xc` et la taille `(bits + 7) / 8` ; il
        écrit `W(32)` le nombre d'entrées (pas de 0x1440 octets), puis par bot trois `W(32)` (index
        absolu du bot, slot, identifiant `bid`) et le corps par `FUN_1407edea8(entree + 0x10)` — le
        corps commun aux fiches de joueur (table de `chunk_00`, paquet de type 8 ; note 5.17).
      - LECTEUR miroir : `FUN_1429875e4` (branche `0xc` du répartiteur `FUN_1428e22c0`), corps par
        `FUN_1407eeba4` ; le premier mot est l'`absoluteBotIndex` (assertion de `FUN_142c26748`).
      - CORPS, après le nom (`<= 16 x R(16)`, arrêt après l'unité nulle) : `R(0x80)`, `R(32)`
        `desired-representation`, `R(64)`, six champs courts (10, 14, 6, 8, 7, 1), le bloc de
        personnalisation (`R(0x39e0)` sur cet exécutable, `profile.PersonnalisationOctets(build)`),
        puis `R(0x160)` : 44 octets bruts en `enregistrement + 0x1400`.
      - `FUN_1424d512c` recopie ces 44 octets en `configuration + 0xCC0` (structure de 0xCF0 octets) ;
        `FUN_140a20620` initialise `+0xCE4`, `+0xCE5`, `+0xCE6` à `0xFF` (-1, aucun) et `+0xCE8` à -1.
      - `+0xCE5` (octet 0x25 du bloc) EST L'ÉQUIPE : `FUN_140ad37f8` -> `FUN_140ad389c` la pose sur le
        joueur (`joueur + 0x285`, masquée par `XOR 0x9E`) ; un changement notifie le moteur (méthode
        `+0xD0`, `FUN_140adedd8`, qui en fait un masque `1 << équipe`) et la télémétrie
        (`FUN_14113dc2c`) ; `FUN_142b70528` compare `+0x285` entre joueurs (coéquipiers).
      - `+0xCE6` (octet 0x26), le jumeau : -1 par défaut, lu seulement par les charges d'événement
        `FUN_1430e17bc` / `FUN_1430e1de0` (avec `+0xCE0`, `+0xCE5`, `+0xCE8`) ; `+0xCE4` est tiré au
        hasard (0..7) à la création du bot (`FUN_142c2e510`), ce qui explique l'octet variable mesuré.
      - POSITION : 270 + perso + 296 bits après l'unité nulle du nom ; sur HI_1_12_0 et HI_1_13_0
        (perso 14 816 bits) : bit 15 382, largeur 8, signé. Le flux étant décalé de 6 bits, l'octet
        0x783 vaut `(équipe & 0x3F) << 2 | jumeau >> 6` : 0x00 / 0x04 pour 0 / 1, 0x784 de même pour le
        jumeau, et 0x785 = `(+0xCE7 & 0x3F) << 2` = 0x04 sur les 77 bots. La mesure (champs de 8 bits
        aux bits 15 382 et 15 390, 56/56) est exactement cette grammaire : AUCUNE contradiction.
- [x] F.3 Fusions de `feat/v75` avant la livraison (consignes du superviseur, feat/v75 a raison) :
      `f8293e52d` (2707fdb31 : vue A V2/V3 de la campagne, ts-usages), `976109dc7` (9321554d6 : lot
      lint du décodeur), `43a1fc4ab` (e4dbc147e : clôture documentaire de la campagne). Conflits de
      la première : chroniques killsource (version de feat/v75, complément de la vue A puis celui du
      lot ; rang `killsource-2026-09-20` archivé pour tenir les 500 lignes), golden d'empreinte
      (révision de feat/v75, empreinte régénérée à révision constante), `SchemaDesFaits` porté à 6
      (le 5 est celui de la vue A, dont les faits ne portent pas l'équipe des bots ; deux entrées
      de chronique), fixtures Go du web régénérées au schéma 80 (seule la version diffère du jeu 79
      de feat/v75). `decode.go` et `sieges_places.go` fusionnés sans conflit (`siegeJamaisTenu` du
      lot lint autour de la priorité de la place du bot). Goldens killsource régénérés : seules la
      ligne des bots et celle du bilan diffèrent de feat/v75. Témoins en processus avec le binaire
      de tête : 0 / 3 / 3 / 60, roster, places et équipes identiques au binaire d'avant les fusions
      sur les 19 films.


### G3 — Implémentation

- [x] G3.0 Champ dans `facts/killsource` (`botmeta_equipe.go`, nouveau ; `botmeta.go`, `roster.go`,
      `decode.go`, `diagnostics.go`) : le paquet de type 12 se lit EN ENTIER par la grammaire de
      l'écrivain (`FUN_14299bda0`, corps `FUN_1407edea8`), perso par build
      (`profile.PersonnalisationOctets`) ; une équipe n'est retenue que d'un paquet dont la marche
      FERME (reste < 8 bits) et d'une entrée au même slot, `bid` et nom que le lecteur historique.
      `BotEntry.Team` (`*int`, nil = non lue), bilan `Roster.BotEquipes` (lues, illisibles,
      contradictoires, hors grammaire, entrées hors balayage, jumeaux discordants, hors domaine,
      paquets non fermés, perso inconnue) ; diagnostics `killsource.equipes_de_bots` (ERREUR : bot
      sans équipe) et `killsource.entrees_de_bots` (AVERTISSEMENT). Aucun repli au registre.
      Parc (instrument `botmeta_equipe_research_test.go`, 48 films, 1 683 paquets) : 79 bots du
      lecteur historique, 78 équipes lues, oracle par entité 56/56 (55 HI_1_13_0 + 1 HI_1_12_0),
      0 paquet non fermé, 0 jumeau discordant, 0 hors domaine, 1 « hors grammaire » (`8076f97f`,
      `43 KaleDucky` slot 0 `bid` 0 : fragment de la copie petit-boutiste du nom que le balayage
      historique lit, cf. D9). `killsource.Rev` NE MONTE PAS (aucune ligne de kill ne change :
      goldens des 4 films de référence et de la mini-bobine, seules la ligne des bots et la ligne du
      bilan changent, cumul inchangé ; 19 témoins : clés `kills`/`killRefs` des artefacts
      identiques) — précédent du lot M2.1 ; complément du 2026-10-06 dans `rev_chronique.go`
      (rang `killsource-2026-09-18` archivé pour tenir les 500 lignes), golden d'empreinte
      régénéré à révision constante. C'est `replay.SchemaDesFaits` (5 -> 6 après la vue A) qui périme les faits.
      Projection `replayidentity.BotIdentities` -> `replay.BotIdentity.Team`.
- [x] G3.1 Équipe d'une entrée (`occupants.go`, `equipeDe`) : entités `ti=9` à l'unanimité, sinon
      `BotIdentity.Team`, sinon (humain, ou film non balayé) la table de contrôle ; un bot d'un film
      balayé sans entité ni déclaration lue reste sans équipe (plus d'emprunt à l'index). Contradiction
      entité / déclaration : l'entité est publiée, ERREUR + compteur expvar
      `rejeu_bots_equipe_contre_declaration` (0 sur les 19 témoins) ; entrée présente sans équipe :
      ERREUR (`journaliserLesPlaces`). D1 corrigé : la vie [947..981] de `343 Donos` passe de 0 à 1.
- [x] G3.2 Places (`sieges_places.go`) : relais à la frame admis quand un des deux est un bot daté
      par BOT_METADATA (D5 corrigé : `572e236b`, Bachici sur la place 7) ; l'humain ARRIVÉ pendant la
      déclaration du bot, sans vie avant son retrait, lui succède — place du bot en priorité du
      chaînage, présence ouverte au lendemain du retrait (D4 corrigé : `43e96765`, place 0 =
      LeodaganQC -> PardonMy -> Cmillward21 dès 1345, 0 dépassement). D2, D3, D6 : consignés
      (ne découlent pas de l'équipe, cf. G3.5). D7, D8 : consignés.
- [x] G3.3 `SchemaVersion` 79 -> 80 (contenu, AUCUN champ neuf : les trois compteurs envisagés en
      couverture auraient changé le contrat servi, donc `generated.ts` du web — retirés, journal et
      expvar à la place) ; chronique v80 ; `structure_test.go` ; plafonds `film_file_size_test.go`
      (+30, +4) ; empreinte de forme (schéma seul) ; 8 goldens d'assemblage (ligne de schéma
      seule) ; fixtures Go du web régénérées (`replay_schema_80_*`, contenu identique hors version).
- [x] G3.4 Tests : killsource (10 unitaires sur paquets fabriqués par la grammaire de l'écrivain,
      goldens réels), `replayidentity` (projection), `replay` (5 tests d'équipe déclarée, 5 tests de
      successions, transport de l'équipe par les faits), suites complètes de `games/halo_infinite/...`,
      `replaybuild`, `service/replayview`, `sync/killcollector`, `archlint` : vertes ; `go vet` avec
      et sans `research` ; golangci-lint (paquets touchés, depuis `e0cd93662`) : 0 constat.
- [x] G3.5 Compteurs sur les 19 témoins, en processus (racine de scratch, copie de la base) :
      | | sansEquipe | sansPlace | placesEnTrop | depassements |
      |---|---|---|---|---|
      | AVANT (`e0cd93662`) | 18 | 21 | 3 | 229 |
      | APRÈS | 0 | 3 | 3 | 60 |
      Restes, un par film, aucun ne vient de l'équipe : `859da825` (SplinterCell958 sans place,
      1 place en trop, 4 dépassements : D2), `bf2a9f05` (AllGodsLove, 1, 16 : D3), `d1dfbc02`
      (`343 Ham Sammich`, équipe 0 lue, aucune place libre, 1, 40 : D6). Témoin clé `43716616` :
      place 5 = Slowpoke6743 (0-247) -> `343 Sandwolf` (248-281, `apparie`) -> KernelPanic10
      (318-) ; plus de place 8. Équipes de vies changées sur les 19 témoins : 4 vies de bots
      seulement (Forge Lord -1 -> 0, Donos 0 -> 1, Ritzy -1 -> 1 deux fois). Chaînes de place des
      18 bots conformes au tableau G1, sauf `4f77afc1` Darkstar (place 1 au lieu de 2 : la place 1
      paraît libre faute de liaison de ses entités d'équipe 1, D8).
- [x] G3.6 `adversarial-review`, `delivery-checklist`, commits, push, CI verte.
      - Commits `d7e329695` (G3.0 à G3.5), `16e992c63` (ronde 1, relecteur A), puis celui des
        corrections du relecteur B ; push et CI suivis au premier plan (`gh run watch`), cf. le CR.
      - `delivery-checklist` : paquets touchés et voisins (30 paquets, `games/halo_infinite/...`,
        `replaybuild`, `service/replayview`, `sync/killcollector`), `archlint`, golangci-lint (0
        constat depuis `e0cd93662`), `go vet` avec et sans `research` ; aucun `persist`, `sync` ni
        `migration` touché (pas de passe `integration`) ; la suite complète `go test ./...` est
        celle de la CI.
      - Ronde 1, relecteur A (lecture BOT_METADATA, `killsource` et projection) : 0 P0, 0 P1,
        16 conditions vérifiées qui tiennent (ordre des bits, largeurs champ à champ, bit 15 382,
        fermeture, appariement slot + `bid` + nom, nil jamais 0, transport, `SchemaDesFaits` 5) ;
        4 P2, corrigés dans le lot : (1) un test passe par `Decode` en CI (mini-bobine + un paquet
        BOT_METADATA fabriqué, `botmeta_equipe_decode_test.go`) ; (2) une équipe -1 se publie -1
        (extension de signe et borne basse tenues par un test) ; (3) la marche passe par le
        curseur méfiant du paquet (`curseurEv`, eventchain.go) : plus de troisième copie du lecteur
        borné ; (4) un paquet non fermé se dit en AVERTISSEMENT même quand chaque bot tient son
        équipe d'un autre paquet. En-tête : 78 bots lus (79 du lecteur historique). Les quatre
        mutations du relecteur rougissent (la borne basse : à la compilation puis au test).
      - Relecteur B (places et rejeu, `film/replay`) : interrompu une première fois par la limite de
        session, relancé au premier plan. 0 P0, 0 P1, 17 conditions vérifiées qui tiennent ; 6 P2,
        traités dans le lot :
        (1) deux humains « successeurs » d'un même bot pouvaient s'asseoir sur sa place et laisser le
        second en recouvrement avec le bot ; (2) un humain parti avant le retrait du bot passait pour
        son successeur, assis `apparie` sur une présence ensuite vidée — corrigés ensemble : le
        successeur doit être ENCORE LÀ après le retrait (`succedeAuBot`), et l'ouverture de sa
        présence se pose AVANT le tri, pour tous les successeurs d'une place (`ouvrirApresLeBot`) ;
        (3) l'affectation de `declaree` n'était tenue par aucun test : test de bout en bout par la
        liaison (S-LIAISON) ; (4) la chronique v80 et `structure_test.go` annonçaient le verdict
        `redecoder` : le digest rend `republier` (aucune révision de couche ne monte) et ce sont les
        faits au schéma 4, refusés sur leur en-tête, qui font redécoder — texte corrigé, cf. D10 ;
        (5) la lecture `tirs` admet désormais la place d'un bot daté à l'humain qui lui succède :
        comportement GARDÉ (Q23), écrit dans `sieges.go` et testé (S-TIRS) ; (6) mutations non
        tenues : tests S-PRIORITE, S-RELAIS-DATE, S-COTOIE-DEBUT, S-PARTI-AVANT, E-DOUBLE. Sept
        mutations rejouées (priorité retirée, `a < t`, clause de début, désaccord de déclarations,
        `declaree` jamais posée, condition « encore là », ouverture retirée) : toutes rougissent.
      - [!] Ronde 2 non jouée : aucun P0 ni P1 en ronde 1 (le skill n'arme la ronde 2 que pour relire
        des corrections de P0/P1) ; les corrections sont tenues par les mutations ci-dessus.
      - Vérification finale en processus (binaire du lot, faits des 19 témoins au schéma 5) :
        compteurs inchangés (0 / 3 / 3 / 60), roster, places, équipes, vies et kills identiques à
        la passe précédente hors les trois compteurs retirés de la couverture.

### G4 — Mesure de l'hypothèse « bouche-trou » (lecture seule)

Reprise demandée par le superviseur après le CHECKPOINT (G4.1 et G4.3 seulement, lecture seule).
Instrument `film/replay/rejeu_bouche_trou_research_test.go` (tag `research`) : 126 films, faits et
artefacts ; l'équipe lue par le champ vient de la sortie de l'instrument de la couche des faits
(`g2c_botmeta.txt`), sans nouvelle lecture des chunks. Méthode : les places sont les sièges publiés des
entrées HUMAINES (2 humains sans place exclus) ; un intervalle vide va de la fin CERTAINE d'un occupant à
la veille du début du suivant sur la même place (`relais`), ou de la frame 0 à la veille du premier
occupant arrivé après elle (`debut`) ; il est couvert par l'union des déclarations BOT_METADATA (frames
exactes) de n'importe quel bot. « Compatible avec une place jamais vide » : couverture d'un seul tenant
dont le bot arrive au plus tard le lendemain de la fin AFFICHÉE du partant et reste déclaré jusqu'à la
dernière image-clé porteuse où le suivant n'est pas lu (les bornes humaines ne sont connues qu'à
l'image-clé près, 20 s).

- [x] G4.1 Intervalles demandés (`relais` + `debut`) : 50.
      - Films où au moins un bot est déclaré (48 films sur 126) : 44 intervalles. `relais` 37 (durée
        médiane 199 f, max 1594 f) : entièrement couverts 2, en partie 34 dont 31 compatibles avec une
        place jamais vide, pas du tout 1 (143 f : `0d265ab0`, place 1, NotThtGuyPal -> SpiffyDart86537).
        `debut` 7 : entièrement 2, en partie 5 dont 1 compatible, pas du tout 0.
      - Films sans aucun bot déclaré (78 films) : 6 intervalles, aucun couvert (`relais` 4 : 103, 199,
        202 et 356 f ; `debut` 2 : 315 et 354 f ; films `0301037e`, `a464e20b`, `c259789d`).
      - Les 7 intervalles non couverts, tous films confondus : médiane 202 f, le plus long 356 f
        (`0301037e`, place 3, Kai Cyr -> BroseJose7).
      - Hors demande, compté à part : 59 places dont le dernier occupant part avant la fin sans
        successeur (`fin`) ; films avec bots 39 (entièrement 9, en partie 26 dont 18 compatibles, pas du
        tout 4), films sans bot 20 (aucun couvert). Durées médianes : 1 413 f dans les films avec bots
        (départs sans successeur humain, que des bots couvrent), 298 f dans les films sans bot (départs
        de fin de match) ; plus quelques artefacts de placement (successeur resté sans place :
        `859da825`, `bf2a9f05`).
      - Troisième contrôle du champ : 76 bots recoupent au moins un intervalle. Un seul intervalle :
        52 accords, 0 désaccord entre l'équipe lue (octet 0x783) et celle de la place. Plusieurs
        intervalles : 13 bots dont tous les intervalles sont de l'équipe lue, 11 dont l'une des
        équipes est l'équipe lue, 0 sans elle.
- [!] G4.2 « Après G3, combien sont tenus par le bot à l'écran » : dépend de G3, bloqué par le
      CHECKPOINT (i).
- [x] G4.3 « Pas encore apparu » (présence publiée sans corps à son début ; frame 0 = premier
      échantillon de position du film) :
      - humains présents dès la frame 0 : 1 072 présences, 461 sans corps à la frame 0 (médiane 37 f,
        max 1 303 f, 228 de 5 s ou plus), 4 sans aucun corps ;
      - humains arrivés en cours de match : 53 présences, 51 sans corps au début (médiane 276 f, max
        1 951 f, 50 de 5 s ou plus), 0 sans aucun corps. Exemples : KernelPanic10 lu à 318, premier
        corps à 831 ; Wrneverchanges 1284 -> 1688 ; Aceshigh949851 1015 -> 1284 ; Cmillward21 1145 -> 1367 ;
      - bots présents dès la frame 0 : 5 présences, 1 sans corps (1 f) ;
      - bots arrivés en cours de match : 77 présences, 34 sans corps au début (médiane 82 f, max 211 f),
        25 sans aucun corps de toute leur déclaration.

### G5 — Republication (préparée, NON exécutée sans go explicite)

- [!] G5.1 Mesure de la durée sur UN film : ABANDONNÉE sur instruction du superviseur (l'utilisateur veut
      être consulté après la fusion et connaître la durée RÉELLE à la fin). Repère, non une mesure de
      la passe : les 19 témoins décodés en entier, en processus, à `d7e329695` : 296 s pour 530 Mo de
      chunks (6,1 s à 72,0 s par film ; pic mémoire 0,99 Gio sur `4f77afc1`) ; le parc local compte
      126 films, 3 236 Mo, 3 704 chunks.
- [x] G5.2 Séquence de ré-extraction du parc local, à jouer APRÈS la fusion dans `feat/v75`, serveur
      ARRÊTÉ, depuis `apps/go-api` du worktree principal :
      1. `go run ./cmd/levelup backfill-replay --only-existing --dry-run` (le plan, rien n'est écrit) ;
      2. `go run ./cmd/levelup backfill-replay --only-existing` : un processus enfant par film, en
         série, verrou solo (`filmproc.AcquireSolo`) ; les faits au schéma des faits 5 au plus sont refusés
         sur leur en-tête, chaque film se REDÉCODE (le récapitulatif peut le compter « republié »,
         D10) ; écrit `data/cache/replays/halo_infinite/<id>.json` (schéma 80) et
         `data/cache/film_facts/halo_infinite/<id>.filmfacts.bin` (schéma des faits 6) ;
      3. passes aval qui relisent les artefacts (ordre de `docs/COMMANDS.md`, base partagée en
         écriture) : `backfill-usage-summary`, `backfill-pad-tiers --force`,
         `backfill-vehicle-takes --force`, puis `tactical-rasters --backfill` (fichiers annexes,
         aucune base) ; pas de `backfill-killsource` (aucune ligne de kill ne change,
         `killsource.Rev` constante) ;
      4. vérification : `coverage.seats` des 19 témoins (script `verifier_temoins.sh` du
         scratchpad, en lecture) — attendu schéma 80, `sansEquipe` 0 partout, `sansPlace` 3,
         `placesEnTrop` 3, `depassements` 60 (restes `859da825`, `bf2a9f05`, `d1dfbc02`), place 5 de
         `43716616` = Slowpoke6743 -> `343 Sandwolf` -> KernelPanic10, aucune place 8. AVANT (publié,
         schéma 78) : 18 / 21 / 3 / 229.

### Clôture

- [x] C.1 `adversarial-review` : ronde 1 sur les conclusions de mesure, un relecteur frais (le diff ne
      porte aucun algorithme de production : l'algorithme des places n'est pas écrit) ; constats corrigés
      (cf. Découvertes, dernier point) ; pas de ronde 2 (corrections purement rédactionnelles qui
      restreignent des affirmations). `delivery-checklist` : `gofmt`, `go vet` (avec et sans `research`)
      sur `film/replay`, `go test ./internal/archlint/` ; aucun code de production touché, donc ni
      `go test ./...` ni tag `integration` locaux : la CI de la branche fait foi.
- [x] C.2 Commit `b0d1db8e4` (G1, G2), poussé ; CI `37470511028` verte (tous les jobs, E2E sauté hors
      PR vers `main`). Commit de G4 : CI suivie après le push.
- [x] C.3 Entrée `.ai/thought_log.md` (2026-10-06, statut En cours, arrêt au CHECKPOINT) ; CR au
      superviseur.

### Phase D — aucune fiche en trop : D2, D3, D6 (consigne de l'utilisateur du 2026-10-06, « Y a pas de places en trop »)

Relance du superviseur après la fusion du lot dans `feat/v75` (`b5c9489ef`) : même worktree, même branche.
Cible 0 / 0 / 0 / 0 (`sansEquipe`, `sansPlace`, `placesEnTrop`, `depassements`) sur les 19 témoins, et au
parc si c'est mesurable en processus. CHECKPOINT avant toute déduction si le film ne dit pas qui part.

**État : CHECKPOINT** — prémisse de D2 contredite par le film, départ de D6 non écrit. Aucune ligne de
production écrite.

- [x] D.0 Départ : `git pull --ff-only` (avance rapide sur `b5c9489ef`). Compteurs de départ, 19 témoins
      en processus (faits au schéma des faits 6) : 0 / 3 / 3 / 60 ; `859da825` 0/1/1/4,
      `bf2a9f05` 0/1/1/16, `d1dfbc02` 0/1/1/40.
- [x] D.1 Instruments. Dans le dépôt : `film/replay/rejeu_fiches_en_trop_research_test.go` (faits et
      artefacts seuls) ; `rjeBots` de `rejeu_equipes_research_test.go` copie désormais l'équipe (sinon
      la liaison rejouée diffère de la publiée). Hors dépôt (scratchpad du superviseur,
      `instruments_rejeu_equipes/rejeu_departs_research_test.go`) : la marche des trames rejouée sous
      le contexte de la cuisson (profil calibré des faits, carte), qui importe `grammar/lecture`
      (ADR 0037 IR-9 : la publication ne voit pas la structure de lecture). Sorties : `departs2.txt`,
      `departs3.txt`, `morts_corps.txt`, `vies_apres_depart.txt`, `fiches_en_trop.txt`.
- [x] D.2 Diagnostic D2 (`859da825`). La « vie » [3167..3170] du slot 548 n'est la vie de PERSONNE :
      - le corps 548 est créé à 1519 avec l'index 2 (opresko, présent) ; la marche des trames lit ses
        records de 1519 à 1534, puis plus rien jusqu'à la fin ; ni suppression, ni dead-state ;
      - l'entité d'opresko (slot 1329) est lue jusqu'à l'image-clé 9 (1627), son absence est prouvée à
        l'image-clé 10 (1827) ;
      - les 5 échantillons [3167..3170] (quantum X constant 528) viennent du balayage de positions par
        ancrage (`grammar.ScanBipedPositions`) ; la marche lit TOUS les paquets des frames 3164 à 3172
        jusqu'à leur terminateur, verdict de fermeture « fermé », et aucun ne porte de record du slot
        548. C'est une lecture fausse du balayage par ancrage : il n'y a pas d'autre joueur à nommer.
      - Empreinte sur les 19 témoins : 1 vie commence après le départ prouvé de son entrée (celle-ci),
        corps créé avant ce départ.
      - Propositions : P2 (recommandée, publication) : une vie qui commence après le départ prouvé du
        joueur que la création de son corps nomme, sur un corps créé avant ce départ, n'est pas
        publiée (comptée et journalisée ; « la grammaire prime » : la preuve d'absence aux images-clés
        contre des échantillons d'ancrage). P1 : la vie reste publiée mais n'ouvre aucune présence
        (pion fantôme de 4 frames). Source : faire recouper l'ancrage par la marche dans `grammar`
        (faits, re-cuisson complète, hors périmètre). Attendu avec P2 : 0/0/0/0, SplinterCell958 sur
        la place 2 après `343 Forge Lord`.
- [x] D.3 Diagnostic D3 (`bf2a9f05`), confirmé par la marche :
      - le corps 529 est créé à 1004 avec l'index 8, lu jusqu'à 1019, supprimé à 1021 ;
      - BOT_METADATA : `343 Mickey` [1008..1023] et `343 BF Scrub` [772..783] sur l'index 8 ;
      - l'entité d'AllGodsLove (slot 1758, index 8) est créée à 1640 (marche), lue de l'image-clé 11
        (1824), absence prouvée à l'image-clé 10 (1624) ;
      - cause : le registre retombe sur l'index -> xuid (AllGodsLove) parce que la lecture par
        déclaration (`identity_registry_declarations.go`) se limite aux index sans humain ET exige que
        la déclaration couvre la création (1004 < 1008 : le paquet BOT_METADATA suit la création,
        cas déjà décrit en tête du fichier) ; Mickey, sans entité, n'a pas d'entrée de roster.
      - Correction (publication) : lecture par déclaration étendue à l'index d'un humain quand ses
        entités prouvent son absence sur toute la vie, et à une création qu'aucune déclaration ne
        couvre et qui précède la seule déclaration croisant la vie ; le bot ainsi nommé entre au roster
        (présence = sa déclaration, équipe lue 0). Attendu : 0/0/0/0, Mickey sur la place 0
        (Aeroflame -> Mickey -> Luigi107763), AllGodsLove sur la place 3 (après yolojoe13, parti à
        1631). BF Scrub (aucune vie) reste hors roster.
- [x] D.4 Diagnostic D6 (`d1dfbc02`). Le film n'écrit pas qui part :
      - dernière image-clé porteuse 30 (5870), les 8 humains y sont lus ; `343 Ham Sammich` déclaré de
        5947 à la fin (index 8, équipe 0), son entité de participant créée à 5947 (marche, slot 2530) ;
      - aucune suppression d'entité de participant des 8 humains avant la fin du film (la marche en lit
        ailleurs : `bf2a9f05` Aeroflame 846, yolojoe13 1631, AllGodsLove 3177, Shiva1663 3592 ;
        `859da825` Witty Hole 4248, après la dernière image-clé 4227 ; elle en manque : opresko,
        HiEmilio9212) ;
      - dead-states de bipède (marche ; les faits ne gardent que ceux des véhicules) : le corps de
        NerdGaiden est tué à 5932 par l'index 3 (Da5BearJud3n), lui-même tué à 5928 par l'index 5 —
        deux morts absentes du fil des morts (dernière : 5910). NerdGaiden joue donc jusqu'à 5932 ;
      - délais de réapparition du film (faits) : 97 écarts entre deux vies d'un joueur, tous de 80 à
        90 frames ;
      - équipe 0 à 5947 : JGtm en vie (records jusqu'à 5988) ; NerdGaiden mort à 5932 (réapparition au
        plus tôt 6012, après la fin 5986) ; BlU3KN1GHT5479 mort à 5910 (au plus tôt 5990) ;
        `stitch vs all` mort à 5845, attendu entre 5925 et 5935, non réapparu jusqu'à la fin (141
        frames). Seule lecture qui désigne quelqu'un : le délai de réapparition — une DÉDUCTION.
      - Options : (a) laisser D6 compté (0/1/1/40) ; (b) déduction par le délai de réapparition
        (publication) : un membre de l'équipe du bot arrivant, mort et non réapparu au-delà du plus long
        écart mesuré dans le film, est réputé parti à son réapparition manquée ; le bot lui succède
        (attendu 0/0/0/0) ; (c) source : départs à la frame près par les suppressions d'entités de
        participant (faits, re-cuisson) — ne règle pas `d1dfbc02`.
- [x] D.5 `SchemaDesFaits` : aucune des corrections proposées (D2 P1/P2, D3, D6 b) ne touche les faits :
      republication depuis les faits au schéma des faits 6 (`SchemaVersion` 80 -> 81). Seules les voies
      « source » (D2 dans `grammar`, D6 c) montent `SchemaDesFaits` (re-cuisson complète). Mesure au
      parc : les faits du parc local sont au schéma des faits 5, refusés au schéma 6 ; une mesure en
      processus redécode les 126 films un par un (environ 30 min, pic proche de 1 Gio par film) : pas
      faite sans go.
- [ ] D.6 Implémentation selon la décision du superviseur (D2, D3, D6), tests sur chaque cas, compteurs
      sur les 19 témoins.
- [ ] D.7 Relecture adversariale au premier plan, push, CI au premier plan, CR.

## Découvertes (notées, non traitées)

- D1 (`c7f94693`) : la vie [947..981] de `343 Donos` est publiée d'équipe 0 par le pont slot -> index
  (désignateur de l'index 8, lu sur l'entité d'un AUTRE bot, Byrontron, en fin de match). Son équipe est
  1 (BOT_METADATA et place). Le pont par index prête une équipe que l'occupant n'a pas.
- D2 (`859da825`) : la vie [3167..3170] (slot 548) est nommée opresko par lien direct de création
  (index 2) alors que l'absence de son entité est prouvée depuis l'image-clé 1827 ; elle rouvre une
  présence et refuse la place 2 à SplinterCell958.
- D3 (`bf2a9f05`) : la vie [1004..1019] (slot 529) est nommée AllGodsLove par l'index 8 pendant la
  déclaration du bot `343 Mickey` [1008..1023] ; Mickey et `343 BF Scrub` (index 8 tenu par un humain,
  sans entité) n'ont pas d'entrée de roster ; idem `343 Oscar` sur `0d265ab0`.
- D4 (`43e96765`) : l'humain qui succède au bot est lu (entité) avant le retrait du bot et n'apparaît
  qu'après lui (KF 1145, retrait 1344, premier corps 1367) : la règle « l'humain succède au bot » demande
  de borner son affichage par le retrait du bot.
- D5 (`572e236b`) : relais dans la même frame (dernière vie du partant finie à la frame de la
  déclaration du bot) : `libre` et `derniereFinAvant` refusent la place à une frame près.
- D6 (`d1dfbc02`) : un départ après la dernière image-clé porteuse n'est pas lu ; l'occupant reste
  « présent jusqu'à la fin ».
- D7 : deux entrées du roster sans nom (`4f77afc1` idx 31, `a6ae19fb` idx 9) occupent des places.
- D8 (`4f77afc1`) : l'index 1 porte deux entités d'équipe 1 que la liaison ne lie à personne
  (`entitesNonLiees = 2` ; slot 1299 aux KF -866..134, slot 1836 aux KF 335..9944), puis l'entité
  d'équipe 0 de Truly Elusive (slot 6600, dès 10144), seul occupant que la table de `chunk_00` nomme à
  ce siège. L'artefact assoit sur la place 1 les bots The Thumb et Razzle (équipe 0, `apparie`) et la
  montre vide de 934 à 10144, pendant que le film y lit un occupant d'équipe 1.
- D9 (`8076f97f`, HI_1_12_0) : le lecteur historique de BOT_METADATA (`scanBotEntries`) lit un bot
  FANTÔME `43 KaleDucky` slot 0 `bid` 0 — un fragment de la copie petit-boutiste du nom (bloc de
  44 octets) qu'il lit à un octet près quand l'octet qui la précède n'est pas nul ; ses offsets
  négatifs tombent dans le bloc de personnalisation (zéros). Il n'est pas épinglé (l'index 0 est un
  humain) et `replayidentity` l'écarte ; ses 4 paquets se comptent « incomplets » (2 entrées pour
  `nbBots = 1`). La marche de l'écrivain ne le lit pas : il se compte « hors grammaire » (G3.0).
  Correctif hors lot (le lecteur historique épingle le roster du kill-feed).
- D10 : après la montée au schéma 80, un artefact 79 dont les faits sont sur disque se lit « décodage
  intact » (aucune révision de couche ne monte : `SchemaDesFaits` n'est pas une famille de `layers`,
  `replaybuild/artifact_digest.go`) ; le verdict est `republier`, mais ses faits au schéma 4 sont
  refusés sur leur en-tête et la cuisson redécode. Le récapitulatif de `backfill-replay` range donc ces
  films parmi les « republiés » alors que chacun redécode : la durée d'une telle passe est celle d'un
  décodage. Non traité (le comportement est juste, seul le compte trompe).
- D11 (`d1dfbc02`) : le fil des morts n'a pas les deux dernières morts que les dead-states de bipède lisent
  (5928 : Da5BearJud3n tué par l'index 5 ; 5932 : NerdGaiden tué par l'index 3) ; sa dernière mort est à
  5910. La vie de NerdGaiden finit donc à 5931 sans mort. Non traité.
- D12 : la marche des trames lit à la frame près la création et la suppression des entités de
  participant (`ti=9`), que les faits ne portent pas (images-clés seulement). `859da825` : Witty Hole
  supprimé à 4248, après la dernière image-clé (4227), publié présent jusqu'à la fin (4338). La marche
  manque des suppressions (opresko, HiEmilio9212). Non traité (faits, re-cuisson).
- Revue adversariale (ronde 1, relecteur frais) : 0 P0 ; 1 P1 (G2.a généralisé au-delà des 2 films
  mesurés : corrigé en restreignant l'affirmation) ; 5 P2 d'imprécision (formulation de G1.2 et durée
  maximale, place 1 de `4f77afc1`, décompte des films et build HI_1_12_0 à N = 1, équipe de Ham Sammich
  donnée sans mesure, preuve de G2.a par le compte de trames fermées) : tous corrigés dans ce plan. A3
  (le champ) reproduit à l'identique par le relecteur.

## Journal

- 2026-10-06 : plan écrit ; lecture du brief, de CLAUDE.md, de `.ai/V7.5/README.md`, des ADR 0034 et
  0037, des mémoires sur la règle des places et les vies de bots, du code cité par le brief.
- 2026-10-06 : G1 clos (instrument `film/replay`, liaison rejouée = publiée sur 126 films). G2 clos :
  champ d'équipe trouvé dans BOT_METADATA (56/56, contrôle par les places 20/20, 19/19 sur les places publiées) ; CHECKPOINT (i).
  Instruments de `killsource` et de `grammar` rangés hors du dépôt (scratchpad du superviseur,
  `instruments_rejeu_equipes/`), sorties dans le même scratchpad.
- 2026-10-06 : reprise du superviseur, G4.1 et G4.3 seulement (lecture seule, ni G3, ni Ghidra, ni champ
  implémenté). Instrument `film/replay/rejeu_bouche_trou_research_test.go` ; sorties
  `g4_bouche_trou.txt` et `g4_intervalles.tsv` (scratchpad du superviseur). Dans les films où des bots
  existent, 43 des 44 intervalles demandés sont recoupés par un bot ; troisième contrôle du champ :
  0 désaccord sur 76 bots.
- 2026-10-06 : Phase D (D2, D3, D6) après la fusion du lot (`b5c9489ef`). Instruments sur les faits et la
  marche des trames ; D2 : la vie fautive est une lecture fausse du balayage par ancrage (la marche
  ferme tous les paquets des frames 3164 à 3172 sans record du slot 548) ; D3 : cause confirmée,
  correction de publication ; D6 : le film n'écrit pas le départ, seule une déduction par le délai
  de réapparition désigne `stitch vs all`. CHECKPOINT, retour au superviseur ; aucune ligne de
  production écrite.
