# SONDE P1 — Où le film porte-t-il le tir continu (Ghost) ? (2026-09-23)

Plan : `../PLAN_RETOURS_REJEU_2026-09-23.md` §4.2 P1. Annexe de départ : `RAPPORT_tirs_vehicules.md`.
Worktree `LevelUp-wt-rr-sondes`, branche `feat/rr-sondes`, base `9cb96a6a9`. Lecture seule des films
du checkout principal, sous la voie `film` (un film à la fois, aucune écriture d'artefact, aucune base).
Instruments : `apps/go-api/internal/games/halo_infinite/film/internal/grammar/tcg_tir_continu_*_research_test.go`
(tag `research`, `TestTirContinuGhost`, commandes de rejeu dans l'en-tête du fichier principal).

Légende : **[M]** mesuré · **[D]** déduit · **[H]** hypothèse.

---

## 0. Verdict en six lignes

1. **Le tir continu n'est dans AUCUN canal lisible du film** [M] : aucun record 36
   (`action_weapon_fire`), 35, 37 ou 10 du pilote ou du Ghost, en tête (cadrage certain) comme hors
   tête ; aucun composant `ti=40` / `ti=35` qui bascule avec le tir ; aucun objet arme attaché ;
   aucun type de paquet qui puisse le porter. Les trois candidates du rapport (§2.2) sont réfutées.
2. **Le jeu NUMÉROTE pourtant chaque tir continu** [M sur les sauts, D sur leur sens] : le champ de
   8 bits du record 36 lu par `FUN_141fcf670` est un **numéro de tir par joueur** ; il saute, sans
   aucun record, exactement à travers l'usage d'une arme à tir continu (Ghost +56 mod 256 et +23,
   Banshee +37, canon continu du Wasp +191 intercalé entre les tirs ENREGISTRÉS de son canon à coup,
   porteurs du Rayon de Sentinelle), et reste complet (0 à 5 manques) chez les tireurs à pied.
3. **Ce que le film porte du tir continu : ses TOUCHES** [M] — `damage_aftermath` (type 0) dont
   ref1 est le CORPS du pilote et la source le tag du véhicule (Ghost `F712C64A`, Banshee
   `FA4FAD21`). Sur 81c02726 : 73 touches, **6/6 frags précédés dans les 2 s, 0/12 au témoin
   décalé, 0 hors épisodes** ; cadence en rafale **8 ticks = 133 ms (7,5/s = 225 coups/min x 2)**.
4. **Le gate est donc passé FORMELLEMENT par un signal qui n'est pas le tir** : les tirs manqués ne
   sont nulle part (G MONEY : 73 touches pour 56 ou 312 tirs numérotés ; boredagain7592 sur
   8a485699 : 23 tirs numérotés, 0 touche). Sur 8a485699, le même signal existe pour la Banshee
   (10 touches, 7,2/s) et manque pour le Ghost (aucune touche).
5. **Le candidat de la vue de contrôle (vue C) est réfuté par l'étalonnage** [M] : son bloc d'action
   s'ouvre pour le seul pilote du Ghost et seulement dans ses épisodes sur 81c02726, mais sur
   8a485699 il est FERMÉ à l'instant des 23 tirs de véhicule à coup enregistrés (8 avec une entrée
   dans le même paquet) et des touches continues de la Banshee.
6. **S3 (Ghidra) n'a pas pu être jouée** : Ghidra n'est pas lancé (127.0.0.1:8089 refuse la
   connexion, pont MCP : WinError 10061). **Arrêt et retour à l'utilisateur** (règle du 22/09 :
   deuxième négatif consécutif sur le tir lui-même).

---

## 1. Cadrage (ce qui rend les négatifs opposables)

- **Horloge** : trame du document = (horodatage du paquet − origine) / 100 ms ; origine 451 125 221 µs
  (81c02726) et 2 331 882 869 µs (8a485699), `origineUS` de la sonde des faits de l'enquête. Frags
  81c02726 (kill-feed, copie de base) : t 395, 623, 650, 2159, 2228, 2402.
- **Oracle de la marche de liste**, écrit avant la mesure : une liste est VALIDÉE quand le dernier bit
  du marcheur R7 tombe exactement sur le début de la vue B trouvé par le localisateur de production
  (`marchLocate`, signature du slot 123). 81c02726 : 2 996 listes, **1 339 validées (44,7 %)** ;
  8a485699 : 5 108, **2 308 (45,2 %)**. La TÊTE de liste est certaine dans tous les cas.
- **Lecteur du record 36 champ à champ** (miroir de `r7Charge36`) : fin identique à celle du marcheur
  sur **1 176 / 1 176** et **700 / 700** records (0 désaccord).
- **Témoins de l'instrument** : les 1 063 (81c02726) et 625 (8a485699) têtes 36 que la production lit
  sont retrouvées ; le détecteur d'attache (`i10`) voit les montées à bord des pilotes (545→775,
  512→770, 535→775 sur 8a485699) ; les tirs de véhicule À COUP sont vus avec leur arme
  (`121B4009`, `49E40D17`, `11725DC4`, `0042678E`).

## 2. S1 — la liste complète

| Mesure | 81c02726 | 8a485699 |
|---|---|---|
| records 36 : tête validée / tête non validée / hors tête validée / hors tête non validée | 675 / 388 / 52 / 61 | 386 / 239 / 36 / 39 |
| dont classe VÉHICULE | **0** | 23 (coup : Wraith, Scorpion, Wasp `11725DC4`, Gungoose) |
| records 36 du pilote (index 5 bits ou ref0 = pilote / Ghost) dans ses épisodes | **0** | 0 (Ghost, Banshee, Chopper) |
| têtes 36 de moins de 113 bits (écartées par la production) | **0** | **0** |
| types 35, 37, 10 dans les 2 s avant un frag | 0 | — |
| événements désignant le pilote / le Ghost dans les épisodes | 109 `damage_aftermath`, 7 `projectile_object_impact_effect` (projectiles QUI TOUCHENT le Ghost), 6 `AIDialog`, 2 `unit_exit_vehicle` | 200 / 19 / 17 / 7 (+46 `damage_section_response`, 43 records 36 d'index 4 ou à ref suivie) |

Gate de S1 (81c02726, présence dans [f − 2 s, f] ; témoins −60 s / +60 s ; hors épisodes) :

| catégorie | frags | témoin −60 s | témoin +60 s | hors épisodes |
|---|---|---|---|---|
| 36 du pilote (index 5 bits) | 0/6 | 15 (à pied) | 0 | 129 (à pied) |
| 36 dont une ref désigne pilote / Ghost | 0/6 | 0 | 0 | 47 |
| 37 `weapon_overheat` · 10 `weapon_effect` | 0/6 · 0/6 | 0 · 0 | 0 · 2 | 0 · 0 |
| **0 `damage_aftermath`, ref1 = pilote, source `F712C64A`** | **6/6 [9 5 2 8 8 2]** | **0** | **0** | **0** |

Les candidates du rapport : (1) un ÉTAT de tir sur `ti=40` — réfutée par S2 ; (2) le record 36 hors
tête — réfutée (64 records hors tête dans les épisodes, tous de joueurs à pied) ; (3) tête de moins
de 113 bits — réfutée (0).

## 3. S1 bis — le NUMÉRO DE TIR du record 36

**Le champ.** Dans le record 36, juste après `estCourt` et `estBloc`, `FUN_141fcf670` lit R(7) puis
R(1) : bits 26..33 du payload dans la disposition canonique (préambule 9 bits, ref0 de domaine 1 à
sonde, ref1 et ref2 absentes). **n mod 256 = (v >> 1) | ((v & 1) << 7)** [M : les valeurs brutes
enchaînent … c252 c254 c1 c3 … quand n franchit 128]. Il vaut 0 au premier tir de chaque joueur,
avance de 1 par tir, traverse morts et corps (G MONEY : c204 sur le corps 532, c210 sur le 537).

**Les sauts** (records de cadrage certain seulement, tête ou liste validée) [M] :

| film | tireur | sauts (tirs numérotés sans record) | ce qui les recouvre |
|---|---|---|---|
| 81c02726 | G MONEY (2) | premier record à n = **56** (t 1275) quand les 7 autres commencent à n = 0 ou 1 | épisode Ghost 311-1088 |
| 81c02726 | les 7 autres | 0 à 5 sur tout le match | — |
| 8a485699 | boredagain7592 (4) | **+23** (t 555-1500) | son Ghost 745-1169 (0 touche) |
| 8a485699 | KyleT1848 (7) | **+37** (2194-2661) ; +6 (2692-3424) | sa Banshee 2334-2449 ; son Chopper 3099-3205 |
| 8a485699 | xmatt3434x (0) | **+54, +107, +30** entre ses tirs ENREGISTRÉS `11725DC4` de 4508, 4605, 4655, 4777 | son Wasp 4454-4943 |
| 8a485699 | 0, 1, 2, 4 | +48, +181, +183, +210 | après la prise du Rayon de Sentinelle (1998, 4205, 5612, 5132) |
| 8a485699 | 3, 5 | 0 | — |

**Ce que cela dit** [D] : le jeu compte chaque tir d'arme à tir continu comme une action de tir du
joueur (même compteur que les tirs enregistrés), mais le film n'en reçoit AUCUN record. Le cas du
Wasp le montre dans un même épisode : le mode à coup (`11725DC4`) est enregistré, l'autre mode
(le canon continu, `d3c407ed` au V3F) n'existe que par les trous du compteur — c'est la façon de
distinguer ses deux modes de tir dans le film (réponse de l'utilisateur à Q11).
Réserve : un compteur modulo 256 ne départage pas 56 de 312 (G MONEY).

## 4. S1 ter — les touches (`damage_aftermath`)

- 81c02726, ref1 = pilote (514 puis 541) : source `F712C64A` **[34 | 39 | 0]** (2 s avant frag |
  reste des épisodes | hors épisodes) ; 73 événements sur 73 instants ; victimes 518:11, 516:9,
  545:9, 513:8, 539:8… ; **cadence en rafale : 8 ticks (133 ms) sur 29 des 43 écarts < 400 ms**,
  16 ticks sur 6 — la cadence nominale du Ghost (V3F : 225 coups/min x 2 canons = 7,5/s).
- 8a485699 : Banshee (ref1 = 545, index 7) source `FA4FAD21` **[0 | 10 | 0]**, 9 touches sur la
  même victime (537), cadence 7-8 ticks (7,2/s) ; Chopper (558) : `A6E597D1` x 12 sur l'objet 2142 à
  chaque tick (nature non établie) ; Ghost (516) : **aucune touche**.
- Encodage [M] : refs {1, 1, 7} ; ref0 = BLESSÉ, ref1 = RESPONSABLE, chacune `R(1) garde ; R(1)
  sonde ; R(9) si sonde, sinon R(13) ; R(2) génération`, **slot = 0x200 + index** ; la ref1 d'un
  tir de véhicule est le CORPS du pilote, pas le véhicule ; charge : `R(1)` puis `R(32)` = tag de
  source (grammaire `lot1DecodeDamageAftermath`).

## 5. S2 — les composants

- **Ghost 769 (épisode 1)** [M] : `i0`, `i1`, `i2`, `i3`, `i25` sur 100 % des records DANS et HORS
  des fenêtres de frag ; `i4` / `i7` (dégâts subis) 1,4 % / 1,0 % hors fenêtres ; `i38
  vehicle-weapon-set` une fois, hors épisode. **Aucun composant ne bascule avant un frag.**
- **Ghost 771 (épisode 2)** : aucun record lu par la marche sur tout le film (le document en porte 988
  échantillons, venus du balayage ancré) — limite, notée en découverte.
- **Pilote** : `i21`, `i25`, `i5` seulement (visée, tick de commande, bouclier) ; aucun composant
  d'état d'arme (`i30..i46`).
- **Objet arme** : aucun objet attaché au Ghost par `i10` ; aucune entité de même durée de vie
  qu'une arme (769 : un objet `ti=37`).
- **Types de paquet** : 0 (trame), 10 (32 octets de globales par tick, 1 pour 1 avec les trames),
  1 (table de datums), 2 (image-clé), 6 / 7 / 8 / 12 (un par chunk), 9 (pied de film) : aucun ne
  peut porter un tir.

## 6. S2 bis — la vue de contrôle (vue C) : candidat réfuté

Lue au bit près de la production (`consumeControleVueC` → `consumeEntreeControle` →
`consume1406d025c`) sur les trames dont la vue B a clos sa liste.
- 81c02726 : 31 621 entrées ; index de contrôle 0..7 = les joueurs (contrôlé par l'alignement
  morts / silences sur 8a485699, index 0, 4 et 7) ; **bloc d'action ouvert pour l'index 2 seul
  (636 entrées, toujours `m0=4 r3=1`), dans ses épisodes seulement (0 hors)** ; aucun des 1 176 tirs
  à pied n'a le bloc de son tireur ouvert.
- 8a485699 : bloc ouvert (`m0=4 r3=6`) chez six joueurs, en rafales hors des épisodes connus ; **fermé
  à l'instant des 23 tirs de véhicule à coup enregistrés** (8 fois, 6 instants, avec une entrée du
  même paquet) et **fermé à la touche continue de la Banshee** (t 2422, entrée du même paquet).
- Sens [H] : une action d'unité autre que la gâchette (boost / capacité de véhicule ?). Non utilisable.

## 7. S3 — bloquée

Ghidra n'est pas lancé (connexion refusée sur 127.0.0.1:8089, pont MCP : WinError 10061). S3 devait
lire le chemin d'émission d'`action_weapon_fire` pour un canon à son en boucle contre un canon à
coup — elle dirait POURQUOI les tirs continus, numérotés, ne sont pas écrits dans le film.

## 8. Ce que le lot M4b devra lire (localisation exacte)

- **Pas de canal par tir continu** : M4b.2 ne doit attendre aucun record 36 pour Ghost, Banshee,
  Chopper, LAAG, LMG du Falcon, canon continu du Wasp, Rayon de Sentinelle.
- **Les touches** : `damage_aftermath` (type 0) à TOUTE position de la liste (la production ne lit que
  l'octet de tête `0xC0`, `weapon_hits.go`) : instant du paquet, ref0 = victime, ref1 = corps du
  tireur (→ épisode → véhicule), source = tag de 32 bits de l'arme de véhicule. Un tracé par touche,
  du véhicule vers la victime, à la cadence mesurée (133 ms).
- **Le compte** : le numéro de tir du record 36 (8 bits, bits 26..33 en disposition canonique) ; le
  saut entre deux records d'un joueur dont un épisode d'arme continue couvre l'intervalle donne le
  NOMBRE de tirs non enregistrés (pas leurs instants) — base d'un repli NOMMÉ et COMPTÉ de rafale
  (instants des manqués absents du film).
- **Le record 36 lui-même** : ref0 = unité tireuse (pour une arme de véhicule à coup, la pièce
  enfant : 772 pour le mortier du Wraith sur 8a485699), index 5 bits, 113 records hors tête sur
  81c02726 que la production ignore.
- Montées : faits (`damage_aftermath` à toute position + numéro de tir) → `SchemaDesFaits` et
  `grammar.Rev` montent, re-décodage (vague D).

## 9. Questions pour l'utilisateur

1. Ouvrir Ghidra (projet HI, plugin MCP sur 127.0.0.1:8089) pour jouer S3 (lecture seule) avant de
   figer M4b ?
2. Sinon, M4b rend-il le tir continu par ses TOUCHES (un éclair par touche, à la cadence de l'arme)
   plus, en repli nommé, une rafale du NOMBRE de tirs numérotés — sachant que les tirs manqués
   n'ont pas d'instant dans le film ?

## 10. Découvertes hors périmètre (non traitées)

1. `archlint` `TestAucuneCibleDeRepliNeNommeUnLotClos` est ROUGE sur la base (`fe2106f4b`) : il lit
   `../../.ai/PLAN_DECODEUR_FILM_2026-09-13.md`, déplacé sous `.ai/V7.5/` par « Archivages docs ».
2. La marche ne lit aucun record du Ghost 771 de 81c02726 (0 sur le film) : trou de couverture
   `ti=40` de la marche (le Ghost 769 est lu).
3. Le marcheur R7 ne s'accorde avec le localisateur de la vue B que sur 45 % des listes ; premier
   arrêt opaque : type 109 `PersonalAILifceycleEffect` (208 sur 81c02726).
4. `projectile_object_impact_effect` (type 7) : ref1 de domaine 8 = identifiant SÉQUENTIEL de
   projectile (1081 → 1086 sur 81c02726) — nommerait les projectiles qui touchent un véhicule.
5. Sauts du numéro de tir non expliqués par les épisodes connus (8a485699 : index 1 +159, index 4
   +70 / +170, index 7 +81, index 6 +37) : détecteur possible d'armes continues ou d'épisodes
   non publiés.
6. Bloc d'action de la vue C (`m0=4`, `r3` = 1 ou 6) : sens inconnu, rafales hors véhicule.
