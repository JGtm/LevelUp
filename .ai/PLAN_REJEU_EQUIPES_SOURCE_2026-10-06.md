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

## État : CHECKPOINT (i) atteint à l'étape G2 — retour au superviseur avant toute implémentation

L'équipe des bots sans entité est ÉCRITE dans BOT_METADATA (G2.c), à un champ que le lecteur actuel
(`film/internal/facts/killsource/botmeta.go`) ne lit pas. La lire exige un champ neuf de BOT_METADATA,
donc de toucher `film/internal/facts` (montée de `killsource.Rev`, faits du parc à ré-extraire) : c'est
la condition (i) du brief. Rien n'est implémenté.

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

### G3 — Implémentation dans `film/replay` (si G2 le permet)

- [!] G3.1 à G3.6 : non commencés. Bloqués par le CHECKPOINT (i) : la source se lit dans
      `film/internal/facts/killsource` (champ neuf de BOT_METADATA, montée de `killsource.Rev`, faits du
      parc à ré-extraire, croise la session RI et la campagne de grammaire). Décision au superviseur et à
      l'utilisateur.

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

- [!] G5.1 : dépend de la décision. Si le champ est lu par killsource, ce n'est plus une republication
      depuis les faits mais une ré-extraction (faits périmés par la révision de killsource).

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
