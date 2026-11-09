# Lot `equipes_b1ad85eb` — 3 puis 5 joueurs par équipe dans le rejeu de `b1ad85eb`

Enquête en lecture seule, 2026-09-23, commit de référence `43a01721e` (`feat/v75`), documents au
schéma 68. Instruments : `SP/sondes/equipes_b1ad85eb/` (Node) et une sonde Go `research` sur les faits
persistés de trois films (worktree `LevelUp-wt-inv-equipes_b1ad85eb`, voir §7). Légende :
**MESURÉ** (sortie d'instrument, fichier:ligne, requête), **DÉDUIT** (raisonnement sur pièces),
**HYPOTHÈSE** (non tranché, avec la sonde qui tranche).

---

## 1. Constat reproduit sur pièces

### 1.1 La composition des panneaux, rejouée à l'identique

`panneaux_lib.mjs` recopie la chaîne du web, fonction par fonction, sans règle ajoutée :
`buildPlayers` (`lib/replay/rosterLogic.ts:107-149`), `buildSeats` / `campsParCote` / `cleDeCamp` /
`groupSeatsByTeam` / `seatOccupantAt` (`features/match-replay/model/seatLogic.ts:123-311`), le rendu
d'une colonne (`ui/ReplayTeams.tsx:197-238` : `absent` → rien, `parti` → tuile « A quitté »,
`present` → fiche), le nom écrit par la fiche (`model/playerCardReadings.ts:99-100`). Le tableau de
score est celui que sert l'API : la requête Q12 (`platform/duckdb/queries_match.go:47-112`), filtre
« tout à zéro » compris, et `team_side = t{team_id}` (`service/match_view_builders_team.go:116-119`).

**MESURÉ** (`repro_b1ad85eb.txt`, `frise_b1ad85eb.txt`), horloge `t = affiché_s × 10 + 227` :

| Instant affiché (frame) | Eagle (f0) | Cobra (f1) |
|---|---|---|
| 0:00 (227), préambule compris (217) | **3** : MONEY x BUTTER, Namikidori, DRghie | 4 : FairyNectar5788, Madina97294, Chocoboflor, JGtm |
| 2:14 (1567) | **5** : MONEY x BUTTER, Namikidori, DRghie, **« Joueur inconnu »** (fantôme), Claudors | 4 : idem |
| 6:24 (4067) | **5** : idem 2:14 | **5** : **FairyNectar5788** (fantôme), Madina97294, Chocoboflor, JGtm, 343 Brew Dog |

Frise des bascules (même instrument) : Eagle 3 → 4 à 0:44.0 (f661), 4 → 5 à 1:36.0 (f1185) ;
Cobra 4 → 5 à 5:27.0 (f3497). Les trois constats de l'utilisateur sont exacts à la tuile près.

**Qui est qui** :
- **Le 4e d'Eagle manquant au départ** : le bot `343 Hundy` (bid 16), présent au coup d'envoi. Son
  corps existe dans le film (slot 512, créé à f0, positions jusqu'à f270) mais il est ANONYME : aucune
  vie nommée, donc aucun siège, donc aucune fiche. Sa ligne de tableau de score est de surcroît
  filtrée par Q12 (0/0/0/0).
- **Le 5e d'Eagle à 2:14** : « Joueur inconnu » = `2535449383340628` (Hanover Cat, gamertag connu de
  `v_gamertag_lookup`, mais sa ligne 0/0/0/0 est filtrée par Q12 et le film ne lui écrit aucun nom).
  Sa seule vie publiée dure 0,2 s (slot 523, f661–662, 5 positions) ; ensuite la règle
  `seatOccupantAt` le garde **`present`** jusqu'à la fin parce qu'aucun successeur n'occupe SON siège
  (siège 9) — Claudors est assis au siège 10.
- **Le 5e de Cobra à 6:24** : FairyNectar5788, dernière vie finie à 4:49.0 (f3117, tuée par MONEY),
  gardée `present` pour la même raison (siège 1 sans successeur) ; 343 Brew Dog est au siège 8.

### 1.2 Ce que disent l'API et le film (pièces brutes)

API (`SP/db/shared_matches_v2.duckdb`, `match_participants`), toujours 4 contre 4 : Eagle = MONEY,
Namikidori, DRghie + la 4e place tenue successivement par Hundy (−0:01.7 → 0:34.0), Hanover Cat
(0:33.9 → 1:16.1), PardonMy (1:16.1 → 1:29.8), Claudors (1:29.7 → fin) ; Cobra = Madina, Chocoboflor,
JGtm + FairyNectar (→ 5:22.3) puis Brew Dog (5:22.4 → fin).

Faits persistés du film (sonde Go, `sonde_faits.txt`) — **MESURÉ** :
- table de `chunk_00` : 8 sièges, dont **WNBA Fan A5 au siège 5** ; il n'a **aucun** corps (aucun
  record de création d'index 5), aucun tir, et il est absent de l'API : un joueur du salon parti avant
  le coup d'envoi, dont la place a été prise par Hundy.
- BOT_METADATA (roster killsource) : `{Slot:8 BotID:16 343 Hundy}`, `{Slot:8 BotID:7 343 PardonMy}`,
  `{Slot:8 BotID:19 343 Brew Dog}`, `botsSuccedes=2` : **les trois bots déclarent l'index 8**.
- records de création de bipède : les **neuf** vies `index_hors_table` (slots 512 ×2, 526, 564, 569,
  573, 577, 585, 594) ont **toutes** l'index de participant **8** — ce sont neuf corps de bots.
- `ScanPlayerTeams` : 26 paquets, 207 records, **12 entités ti=9**, `divEntite=0`, **`divIndex=1`**,
  table publiée `0:0 1:1 2:1 3:0 4:1 6:0 7:1 9:0 10:0` (ni 5, ni 8).

### 1.3 Mesure sur le parc (111 documents, `parc.mjs`, une image par seconde de jeu)

Capacité = taille d'équipe du mode, MESURÉE sur les 111 matchs (`present_at_beginning` /
`present_at_completion` par équipe) : BTB 12, Squad Battle 8, autres 4.

| Indicateur (règle d'aujourd'hui) | Documents | Volume |
|---|---|---|
| Équipe affichant **plus** de tuiles que la capacité | **14 / 111** | 5 630 s-équipe (b1ad85eb : 539) |
| Équipe affichant **moins** de présents que l'API | **76 / 111** (33 au-delà de 30 s continues) | 3 485 s |
| Tuile « fantôme » d'un joueur que l'API dit PARTI | **22 / 111** | 8 376 tuile-s |
| Tuile « fantôme » d'un joueur resté (mort sans réapparaître avant la fin — légitime) | 109 | 8 815 tuile-s |
| Troisième colonne « sans équipe » | 3 (`a6ae19fb`, `f2966f08`, `db1b00b3`) | — |
| Bots au roster / sans équipe | 41 / **7** (5 docs) | — |
| Entrées de roster sans équipe | 17 | — |
| Vies `index_hors_table` | **58** (16 docs) | — |
| Pistes anonymes (ni xuid ni bot) | 36 (12 docs) | — |
| `coverage.seats.sansPresence` | 19 (13 docs) | — |
| Index partagé par plusieurs entrées | 2 docs : `b1ad85eb` (index 8, 3 bots, **deux équipes**) ; `94a28b8b` (index 8, 2 bots, **même équipe** → équipe publiée) | — |
| Participants API exclus du tableau de score (Q12, « tout à zéro ») | 64 (31 docs) | — |

Le « moins que l'API » a deux familles : les joueurs présents dont la première vie n'est pas encore
dans le film (ex. `3ba5a548` : Cobra affiche **0** tuile à 0:00, première vie d'ooSilentDeathoo à
1:00) et les bots sans vie nommée (b1ad85eb : Eagle 3/4 de 0:00 à 0:43.9).

---

## 2. Cause racine — prouvée

Quatre défauts se composent ; les deux premiers suffisent à produire 3 / 5 / 5+5.

### C1 (web) — Un occupant sorti SANS successeur SUR SON SIÈGE reste `present` pour toujours

`seatLogic.ts:123-134` : une présence finie rend `present` tant qu'aucun occupant suivant n'est
assis sur le **même** `seat` (« mourir n'est pas partir », D2 du lot 1.9.14). La règle était voulue
pour les morts de fin de partie ; elle devient un fantôme dès que le remplaçant est assis sur un AUTRE
siège — ce qui est le cas général (C2). **MESURÉ** : Hanover Cat (siège 9, présence 661–662) et
FairyNectar (siège 1, présence 0–3117) restent affichés jusqu'à la fin ; 8 376 tuile-s de ce type sur
22 documents.

### C2 (Go) — Le « siège » publié est l'INDEX DE PARTICIPANT, et un remplaçant n'en hérite presque jamais

`sieges.go:131-162` pose `seat = filmIndex`, puis le repli `apparierLesArrivants` (`sieges.go:263-333`)
ne chaîne que (a) des partants ORIGINES de la table de départ, (b) dont l'équipe est LUE, (c) qui ont
des vies, et (d) dont la présence (enveloppe des vies) finit AVANT l'arrivée. Sur b1ad85eb
(**DÉDUIT** ligne à ligne, `coverage.seats.apparies = 0` **MESURÉ**) :
- la place réellement libérée (siège 5, WNBA Fan A5) n'a ni vie ni équipe → jamais partante ;
- Hundy / PardonMy (bots, index 8) n'ont pas d'équipe → exclus ;
- Hanover Cat (index 9) est un ARRIVANT → ne peut pas être partant pour Claudors (index 10) ;
- les seuls partants d'Eagle sont MONEY (5042) et DRghie (5050), « partis » parce que morts avant
  la dernière frame : ils ne sont libres qu'après l'arrivée de 9 et 10.
Résultat : Eagle a 5 sièges (0, 3, 6, 9, 10), Cobra 5 (1, 2, 4, 7, 8). Le film réutilise l'index d'un
partant 2 fois sur 35 arrivées (lot 1.7) ; la règle « siège = index » ne peut donc pas porter
« remplaçant = même place ».

### C3 (Go) — L'équipe est agrégée PAR INDEX, et l'index 8 est tenu par des bots de DEUX équipes

`grammar/player_teams.go:128-166, 240-262` : les lectures du désignateur (composant i0 de ti=9) sont
accumulées par **index de joueur** ; un index qui porte deux désignateurs est « divergent » et n'est
pas publié. **Preuve par élimination (MESURÉ + DÉDUIT)** : 12 entités, 9 index publiés
(0,1,2,3,4,6,7,9,10), `divEntite=0`, `divIndex=1`. Si WNBA Fan A5 (index 5) avait une entité, elle
aurait un désignateur unique et l'index 5 serait publié : il ne l'est pas, donc WNBA n'a aucune entité
et les **3 entités restantes portent l'index 8**, avec au moins deux désignateurs (sinon 8 serait
publié). Les trois bots déclarent l'index 8 et l'API les met en t0, t0, t1. Conséquence : les trois
entrées de bot n'ont pas d'équipe (`coverage.teams.unread = 4` = WNBA + 3 bots) et leurs pistes
portent `team: -1`. Le témoin contraire existe : sur `94a28b8b`, deux bots de MÊME équipe se relaient
sur l'index 8 → pas de divergence → équipe publiée. Le film porte donc l'équipe de chaque bot (par
ENTITÉ, désignateur stable : 88/88 entités au lot 1.9.14) ; c'est l'agrégation par index qui la perd.

### C4 (Go) — Les corps de bots sur un index partagé ne sont jamais nommés ; les relais sont datés par une API en retard

- `identity_registry_creation.go:214-230` : un corps dont l'index est celui d'un bot déclaré est
  refusé avec la cause `index_hors_table` (libellé trompeur : l'index EST dans la table des bots).
- `identity_registry_scoreboard.go:209-226` (`bidsParIndex`) et `identity.go:328-345`
  (`botNamesBySeat`) **s'abstiennent** sur un index que plusieurs bots déclarent → aucun nom.
- Seuls les relais (`successions.go:61-87`) nomment un bot, et seulement un bot `joined_in_progress`
  (Hundy ne l'est pas), dans la fenêtre [heure API − 2 s, heure API + 20 s]. **MESURÉ** : le corps
  564 (index 8, f3236–3416) est Brew Dog — le fil des morts le tue à 360 411 ms = f3417 — mais il naît
  **21,4 s AVANT** l'heure API du relais (f3450,5) : hors fenêtre, il reste anonyme ; les vies 569…594
  sont nommées. Même écart **exact** sur l'autre relais : le corps 526 (index 8) naît à f774 =
  heure API du relais Hanover→PardonMy (f988,3) − 21,4 s. Sur le parc (`decalage_relais.txt`) : sur
  63 relais de bot datés par l'API, 45 ont une vie de bot qui naît AVANT l'heure API, médiane
  **−22,3 s**, 18 entre −15 et −31 s. **DÉDUIT** : l'heure de participation API d'un relais est en
  retard d'environ 20 s sur le film.
- `sansPresence 3` = WNBA Fan A5 (jamais joué), Hundy (corps 512 anonyme), PardonMy (corps 526
  anonyme). **La présence n'est publiée nulle part** : le web la reconstruit comme l'enveloppe des
  vies NOMMÉES, et `coverage.seats.presencesCloses` (9) compte indistinctement départs et morts de fin
  de partie (le code le dit, `sieges.go:95-101`).

### Le fait qui ouvre la solution — le film écrit la PLACE (MESURÉ, 3 remplacements sur 3)

L'index de tireur des événements de tir (record type 105, `grammar/fire_events.go:35-40`) n'est pas
l'index de participant : c'est la **place**, et le remplaçant hérite de celle du partant
(`tirs_vs_vies.txt`, jointure des tirs persistés avec les vies publiées, tolérance 2 frames) :

| Film | Index de tireur | Tirs | Tombés dans les vies de… | Index de participant du tireur |
|---|---|---|---|---|
| b1ad85eb | 5 (siège de WNBA Fan A5) | 84 | **Claudors 100 %** | 10 |
| 43716616 | 5 (siège de Slowpoke6743, parti) | 65 | **KernelPanic10 100 %** | 9 |
| 43e96765 | 0 (siège de LeodaganQC, parti) | 38 | Cmillward21 89 % (le reste = LeodaganQC avant son départ) | 9 |

Chaque autre index de tireur tombe à 89–100 % dans les vies du joueur de la table au même numéro.
Sur b1ad85eb, Claudors a donc **0 tir publié** : ses 84 tirs sont rangés `noSlot` (144 au total),
parce que `buildShots` (`shots.go:75-117`) joint l'index de tireur à l'index de participant.

---

## 3. Historique — ce qui a été fait, et pourquoi ça n'a pas suffi

- **2026-09-02** (retour utilisateur) : « la fiche est un siège » ; appariement ordinal côté web sur la
  participation API.
- **Lot 1.7 (2026-09-14)** : équipe lue dans le film (ti=9 i0), agrégée par index. La mesure
  « D-remplaçants » relevait pourtant déjà PAR ENTITÉ l'index, le désignateur et la première / dernière
  image-clé (18 films, 35 arrivées, « l'entité n'est jamais réutilisée », « désignateur stable ») —
  et notait « aucune divergence d'index sur ce corpus » : aucun index repris par des bots de deux
  équipes. b1ad85eb est ce cas.
- **Lot 1.9.14 (2026-09-15)** : siège = index publié, repli ordinal côté Go, règle web
  `present/parti/absent`. D2 (1.9.14) : « le film n'écrit nulle part "ce joueur a quitté" » → la fiche
  d'un occupant sans successeur reste. Or l'instrument du même lot (`grammar/sieges_remplacants_research_test.go`)
  lisait le départ : « partants idx 1 (pk 9), 2 (pk 8), 6 (pk 14) » sur `bcb6d393` — la disparition de
  l'entité ti=9 EST le départ, au pas des images-clés (≈ 20 s). Ce fait n'a jamais été publié.
- **E2 / P1-8 (2026-09-06/08)** : abstention voulue sur les sièges de bot partagés (`index_hors_table`,
  `bidsParIndex`, `botNamesBySeat`) « le relais saura les départager » — le relais fenêtré sur l'heure
  API ne le peut pas (C4).
- **Lot 5.2a.4 (2026-09-20)**, constat « trois équipes, dont deux Cobra » sur ce même match : corrige
  le LIBELLÉ (`campsParCote`), pas le NOMBRE. Son témoin fige le défaut : `seatLogic.test.ts:125-150`
  attend `[3, 5]` sièges — 5 dans Cobra. Aucun test n'a jamais porté l'invariant « ≤ taille d'équipe ».

---

## 4. Solution proposée

La règle d'affichage demandée — **une place par siège occupé à l'instant T, départ = sortie du
panneau, remplaçant = même place** — n'exige côté web qu'une lecture : « la place P montre l'occupant
dont la présence couvre T, sinon rien ». Tout le travail est dans ce que le document publie : pour
CHAQUE occupant (bots compris, un par bot même sur un index partagé), son **équipe**, sa **présence**
et sa **place**.

### Option A — recommandée : la place, la présence et l'équipe LUES dans le film (taille L)

Go :
1. `film/internal/grammar/player_teams.go` (+ un fichier neuf pour rester sous 500 L) : le balayage
   ti=9 rend les ENTITÉS (`slot, index, désignateur, première/dernière image-clé`) — le prototype existe
   (`balayerEntitesTi9`, `grammar/sieges_remplacants_research_test.go:94-146`) ; la table par index
   devient un CONTRÔLE dérivé (divergences comptées). BOT_METADATA garde l'instant de ses paquets
   (`facts/killsource/botmeta.go:56-83`) pour lier chaque bot déclaré à SON entité par le temps (deux
   lectures du film, aucune fenêtre).
2. Faits : `FilmInputs` + codec (`film_inputs.go`, `filmfacts_encode.go`, `filmfacts_decode.go`) —
   `SchemaDesFaits` monte, verdict « à recuire : redécoder ».
3. Publication (`replay/`) : `roster_entry.go` gagne `presence: [{from, to, toMax?}]` (entité ti=9
   affinée par les vies : `from` = min(première image-clé vue, première vie) ; `to` = fin du film si
   l'entité est à la dernière image-clé, sinon max(dernière vie, dernière image-clé vue), `toMax` =
   image-clé suivante) ; `player_teams.go` : équipe PAR ENTRÉE via son entité ; `sieges.go` : `seat`
   = PLACE — siège de la table pour les occupants du départ, **place lue dans les tirs** pour un
   remplaçant qui tire (lecture), sinon chaînage par équipe sur les présences publiées (repli nommé,
   compté, généralisé à tout partant, bots et arrivants compris) ; `identity_registry_creation.go` /
   `identity_registry_scoreboard.go` : un corps d'index de bot partagé prend le `bid` du bot dont
   l'entité vit à sa création (les 9 vies `index_hors_table` de b1ad85eb deviennent `direct`) ;
   `successions.go` reste un repli compté.
4. Web : `seatLogic.ts` — `seatOccupantAt` lit `presence` ; la règle « `present` sans successeur » est
   SUPPRIMÉE (la présence du film tranche D2 (1.9.14)) ; regroupement par `roster[].team` seul
   (`campsParCote` ne reste qu'en repli compté pour les artefacts < 69, retrait quand 0 artefact < 69
   en prod) ; `ReplayTeams.tsx` ; fiche « présent sans corps » (état « pas encore apparu », chaînes FR
   + EN dans `i18n.ts` / `i18nContract.ts`) ; `replayNormalize.ts` comble `presence`.
5. **Montée de schéma 68 → 69 : OUI. Re-décodage du parc (faits) puis republication : OUI — décision
   utilisateur**, jamais lancée par ce lot.

Tests et garde-rails :
- Go, ROUGE aujourd'hui : un témoin au gabarit de b1ad85eb (8 sièges dont un jamais joué, 3 bots
  index 8 de désignateurs 0/0/1, arrivants 9 et 10) — places Eagle 5 : Hundy → Hanover Cat → PardonMy
  → Claudors, Cobra 1 : FairyNectar → Brew Dog ; équipe de chaque bot ; au plus 4 occupants par équipe
  à chaque frame. Mutation : réagréger par index → rouge.
- Web, ROUGE aujourd'hui : réécrire le témoin `seatLogic.test.ts:125` (qui fige `[3, 5]`) avec les
  tuiles attendues à f227 / f1567 / f4067 :
  Eagle {MONEY x BUTTER, Namikidori, DRghie, 343 Hundy} → {…, Claudors} ;
  Cobra {FairyNectar5788, Madina97294, Chocoboflor, JGtm} → {343 Brew Dog, Madina97294, Chocoboflor, JGtm}.
- Garde-rail : test de propriété sur toutes les fixtures de contrat et goldens d'assemblage —
  « à aucune frame une équipe n'a plus d'occupants que de places » ; test que `seatOccupantAt` rend
  `absent` après la présence d'un occupant sans successeur.
- **Gate sur documents réels** : l'instrument du parc (à porter en test `research` Go ou vitest sur
  `data/cache/replays`) sur les 111 documents republiés : 0 équipe au-delà de sa capacité, 0 tuile d'un
  joueur sorti, « moins que l'API » limité aux fenêtres du retard API mesuré (≤ 30 s autour d'un
  relais) ; b1ad85eb exactement 4 + 4 aux trois instants, noms ci-dessus.

Risques : coût du re-décodage ; départ pendant la mort daté à l'image-clé près (≤ 20 s) ; bot présent
moins de 20 s entre deux images-clés (aucune entité vue : présence par ses corps seuls, sinon comptée
`sansPresence`) ; les tirs de bots ne sont pas lus aujourd'hui (aucun tir d'index ≥ 8 sur les trois
films) : leur place passe par le chaînage.

Sonde à jouer AVANT de coder (décodage interdit dans cette enquête — elle demande un GO) :
`TestSiegesDesRemplacants` pointé sur `data/cache/film_chunks/b1ad85eb` (répertoire par variable
d'environnement, ~10 lignes) + horodatage des paquets BOT_METADATA. Attendu (HYPOTHÈSE, tirée du
retard API constant de 21,4 s de C4) : trois entités d'index 8 (désignateurs 0, 0, 1 ; fenêtres
≈ [0, 353], [774, 911], [3236, fin]), entité d'index 1 absente dès l'image-clé qui suit f3117, aucune
entité d'index 5 ; 207 records sur 26 paquets = 25 × 8 + 7 (DÉDUIT : huit occupants à toutes les
images-clés sauf une — le film lui-même dit 4 contre 4). Coût : un film 4v4 (25 Mo compressés, 26
images-clés porteuses), une passe d'images-clés, ≈ 1 min, quelques centaines de Mo.

### Option B — relais : présence et équipe de la participation API, en repli nommé (taille M)

Sans re-décodage : publication depuis les faits persistés + la participation que `replaybuild`
fournit déjà (`Participants`, en la passant TOUTE, lignes « tout à zéro » comprises). Présence =
vies ∪ fenêtre API, bornée par l'arrivée du successeur sur la même place ; équipe API pour une entrée
que le film tait (repli nommé, compté) ; place lue dans les tirs sinon chaînage. Même règle web,
schéma 69, republication DEPUIS LES FAITS (rapide, aucun film).

**Simulée sur les 111 documents** (`regle_apres.mjs`, `parc_apres_agg.json`) : équipes au-delà de la
capacité 14 → **10** documents (5 630 → **374** s) ; « moins que l'API » 76 → **8** documents
(3 485 → **198** s) ; tuiles de joueurs sortis 8 376 → **0** ; b1ad85eb exactement 4 + 4 aux trois
instants avec les noms attendus (`frise_apres_b1ad85eb.txt`). Le résidu est l'imprécision de l'API :
bots dont l'heure API précède la fin des vies du partant (`4ecdf3e7`, `859da825`, `bf2a9f05`…) et
micro-vies (≤ 0,5 s) d'un partant après son départ (`859da825`, opresko, f3167–3170).

Risques : contraire à ADR 0034 D-9 (« la base est un compteur de contrôle, jamais un repli ») pour
l'équipe des bots ; retard API ≈ 20 s (C4) qui décale arrivées et départs ; critère de retrait
obligatoire (= option A livrée).

### Recommandation

**Option A.** Le film porte les trois faits nécessaires — l'entité par occupant (présence et équipe),
la place dans les tirs — et deux lots ont déjà su les lire en recherche. L'option B n'a de sens que si
le re-décodage du parc est refusé pour l'instant, avec ses replis nommés, comptés et datés.

---

## 5. Questions pour l'utilisateur

1. Pendant un relais (le partant est sorti, le remplaçant pas encore arrivé) : la place reste-t-elle
   visible **vide** (tuile neutre sans nom, la grille ne bouge pas) ou **disparaît**-elle (l'équipe
   montre 3 tuiles le temps du relais) ?
2. Un joueur présent mais sans corps à l'instant (pas encore apparu, ou en chargement après avoir
   rejoint) : on affiche sa tuile avec l'état « pas encore apparu » — d'accord ?
3. Départ pendant qu'il est mort (le film ne le date qu'à l'image-clé suivante, ≤ 20 s) : la tuile sort
   à la **fin de sa dernière vie** ou à la **première image-clé où il n'est plus là** ?
4. Mécanique : un bot remplace immédiatement le partant et prend SA place, puis un humain qui rejoint
   prend la place du bot (le bot sort) — c'est ce que les tirs lisent (index de tireur = place). Exact ?
5. Option A = re-décodage du parc (faits de film) puis republication au schéma 69 : GO ? Sinon,
   l'option B en attendant (équipe et présence de l'API en repli nommé, contraire à D-9) est-elle
   acceptable ?

---

## 6. Hors périmètre découvert (noté, non traité)

1. **Tirs des remplaçants perdus** : l'index de tireur (type 105) est la place, pas l'index de
   participant — `shots.go:75-117` les range `noSlot` (b1ad85eb : Claudors 84 tirs, 0 publié ;
   `noSlot` 144). À vérifier : `shared.match_weapon_shots` / `resolvePlayerIndices` (précision par
   arme, clés sur l'index de tireur 5 bits) imputeraient ces tirs au joueur qui a QUITTÉ la place.
2. **Heure API des relais en retard d'environ 20 s** sur le film (médiane −22,3 s sur 45 relais ;
   b1ad85eb −21,4 s deux fois) : `successions.go` (fenêtre −2 s / +20 s) manque la première vie du bot
   remplaçant ; `presenceFeed.ts` date les lignes « a rejoint / a quitté » avec le même retard.
3. **Q12 exclut les participants « tout à zéro »** (64 sur 31 documents) : le rejeu perd leur nom
   (Hanover Cat → « Joueur inconnu ») et leurs lignes de présence.
4. `43e96765` : index 0 (LeodaganQC) divergent (`divIndex=1`) — une seconde entité d'index 0 d'un
   autre désignateur, à instruire.
5. Bots présents moins de 20 s : aucune équipe lue (`43716616`, `72b0a25e`, `d1dfbc02`, `f2966f08`,
   `divergences 0`) — `ScanPlayerTeams` ne lit que les images-clés.
6. Micro-vies (≤ 0,5 s) : b1ad85eb slot 523 (Hanover Cat, 5 positions), `859da825` slot 548 (opresko,
   après son départ API).
7. Trois documents affichent une 3e colonne « sans équipe » (`a6ae19fb` bot hors roster, `f2966f08`
   bot fantôme toute la partie, `db1b00b3` film tronqué de 38 s).
8. `rosterLogic.groupByTeam` / `sideResolver` lisent toujours la feuille (D4 1.9.14, non traité).

---

## 7. Instruments et worktree

- Node (`SP/sondes/equipes_b1ad85eb/`) : `panneaux_lib.mjs` (règle web rejouée), `repro_b1ad85eb.mjs`,
  `frise.mjs`, `parc.mjs [web|apres]`, `regle_apres.mjs`, `frise_apres.mjs`, `tirs_vs_vies.mjs`,
  `decalage_relais.mjs` ; exports DB `participants.tsv`, `registry.tsv` ; sorties `*.txt`, `parc_*.tsv`,
  `parc_*_agg.json`.
- Go : `apps/go-api/internal/games/halo_infinite/film/replay/equipes_b1ad85eb_research_test.go`
  (`//go:build research`, lit les faits de `b1ad85eb`, `43e96765`, `43716616`, aucun film), joué deux
  fois sous verrou `voie.sh`, `GOCACHE` privé, `CGO_ENABLED=0`. Sortie `sonde_faits.txt`, `tirs_*.tsv`.
- **Worktree à retirer** : `C:/Users/Guillaume/Downloads/Scripts/LevelUp-wt-inv-equipes_b1ad85eb`
  (détaché sur `43a01721e`, un seul fichier ajouté, non commité, `.gocache` privé).
