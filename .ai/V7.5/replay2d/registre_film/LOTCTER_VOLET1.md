# Lot C-ter — volet 1 : la propriete « colline active » de KOTH, mesuree contre le score affiche

> Perimetre : CT.1.1, CT.1.2, CT.1.3 + Gate 1 du plan `PLAN_EXPLOITATION_REGISTRE_FILM.md` (§3, lot
> C-ter). Branche `wt/colline-propriete` (base `feat/v75` = `638c4d044`), commits `f077cc120`
> (CT.1.1), `e3b24a0d7` (CT.1.2), `3f0ed332f` (CT.1.3). Mesures des 18-19 aout 2026.
> Instruments : `internal/analysis/replay/colline_propriete_test.go` (CT.1.1),
> `colline_propriete_oracle_test.go` + `colline_propriete_mesure_test.go` (CT.1.2), sous garde
> `ZONE_FILM`, un film par processus, avant-plan (D17). Sorties : `registre_film/lotCter/`
> (`<short8>_ti13_series.tsv`, `_ti13_resume.tsv`, `_ct12_mesure.tsv`, `ct11_*.log`,
> `ct12_*.log`, `ct13_temoin_01e1f945.log`). Gates : `LOTCTER_volet1_gates.log`.

## 0. Le resultat, en une phrase

**La colline active de KOTH est DESIGNEE par le film : le tag 5 du slot de l'objet de mode
change de valeur 13 a 21 ms apres chaque capture non terminale (13/13 sur 4 films, temoins
decales 0 %, hasard 1,5-2,6 %) ; a la lettre du plan le gate 1 tombe (la capture TERMINALE n'a
pas de colline apres elle : 67-83 % au lieu de 90 %), sur l'oracle « changements de colline »
du plan il tient 3/4 — `active` est BRANCHE sur ce designateur (CT.1.3), le repli n'est pas code.**

## 1. Methode

**CT.1.1 — la serie de chaque tag.** Balayage `ti=13` par le chemin de production
(`filmdec.ScanFilmManagedProperties`), lectures posees sur deux axes (ms depuis le premier paquet
du film ; frame du rejeu, `OriginMs` retranchee), serie complete par (slot, mode, tag[, index de
film]) et resume par (slot, mode, tag). **Ce qui a ete necessaire pour lire quelque chose** : la
bande de production COMBLE la plage de slots (52 slots sur `01e1f945` contre 20 vus aux
images-cles), et le trafic des slots combles est de la contamination d'ancrage — 842 series
sur `01e1f945`, dont 761 sans une seule lecture chainee. Le scanner publie desormais, par
lecture, le temoin qu'il comptait deja par record : `ManagedPropertyRead.Chained` (le record
porteur chaine, `zone_state_scan.go`, +1 champ, sans effet sur les consommateurs). Le resume
est calcule sur les lectures chainees, les non chainees sont comptees a cote.

**CT.1.2 — l'oracle et la mesure.** Oracle principal = les increments du score AFFICHE des deux
equipes (statborg, composant 0 valeur A, `SeriesTotal`), le dernier marque TERMINAL (la capture
qui atteint la limite de score et clot le match — les 4 films finissent ainsi). Second oracle =
les periodes actives publiees aujourd'hui (methode positions) et les rampes brutes de la jauge.
Seuils du plan, ecrits avant la mesure dans le fichier : (i) exclusivite >= 95 % des frames,
(ii) synchronie a +/- 1 s de >= 90 % des increments, (iii) temoins (decale +20 s ; « slots
permutes » = la synchronie des AUTRES slots du meme tag) sous la moitie du reel, ET niveau du
hasard (part du match a +/- 1 s d'un increment ; taux attendu de N bascules uniformes, 200
tirages). Deux lectures par tag, ecrites avant de regarder : F « drapeau par slot » (booleen 1 =
actif ; enumere != absent ; mode B : au moins un joueur), D « designateur » (tout changement de
valeur, premiere emission comprise — l'etat initial vit dans l'image-cle ; mode B : par joueur,
instants fusionnes a la frame ; exclusivite = un seul slot porteur). Candidats du plan : tags
1, 2, 6 (mode A), 7-15 (mode B) ; tags 3 et 4 = temoins de comparaison ; **tag 5 hors liste**
(tenu pour une cle constante par le lot C-bis), mesure au meme oracle et etiquete tel quel.

**CT.1.3 — le branchement.** `zone_states_hill.go` : le designateur (tag 5 chaine, elu par la
structure de l'objet de mode) borne les periodes ; la grappe des positions pendant les montees
de jauge de la periode (a defaut, la periode entiere) n'apparie plus que la forme ; la lecture
par rampes reste le repli d'un film sans designateur. Test discriminant, temoin recuit.

## 2. Ce que la serie montre (CT.1.1) — l'objet de mode KOTH

Sur les 4 films, l'objet de mode tient sur **quatre slots consecutifs** :

| film | carte | score | designateur (tag 5) | proprietaire / capteur (tag 4) | jauge (tag 3) | bande stricte / comblee | lectures chainees |
|---|---|---|---|---|---|---|---|
| `01e1f945` | Catalyst | 3-2 | 1471 (4 valeurs) | 1472 (99), 1473 (106) | 1474 (3 450) | 20 / 52 | 5 564 / 10 239 |
| `0a247154` | Solitude | 4-2 | 1622 (5 valeurs) | 1623 (213) | **aucune** | 20 / 44 | 3 522 / 7 742 |
| `606d9844` | Chasm | 0-3 | 1498 (2 valeurs) | 1499 (13), 1500 (14) | 1501 (316) | 20 / 52 | 1 277 / 3 059 |
| `8076f97f` | Shogun | 3-0 | 1540 (2 valeurs) | 1541 (35), 1542 (37) | 1543 (1 197) | 20 / 52 | 2 863 / 8 957 |

- **Le tag 5 CHANGE, avec un vocabulaire ORDINAL identique sur les 4 cartes** : `0x78F81557`,
  `0x8727C0FF`, `0x28B76E9D`, `0xCF284119`, `0x7FFD90E7` dans cet ordre (2e a 6e colline). Ce
  n'est pas le NOM d'un emplacement (il differerait par carte) : c'est le RANG de la colline
  courante. La premiere designation n'est jamais dans le delta (image-cle).
- **Les tags 1, 2, 6 sont VIDES** : 0 a 2 emissions chainees par film, jamais un changement.
  Le mode B : tags 10-15, 1 a 3 emissions ; tag 9, epars ; **tag 7** (2 slots, ~9 joueurs, 441 a
  1 592 emissions chainees a 99,9 %) = une valeur PAR JOUEUR qui tique a la cadence de 1 s par
  bouffees, ~35 valeurs distinctes dans [0 ; 0,3] ; **tag 8** (1 slot) = un compteur PAR JOUEUR
  1, 2, 3, 4 emis aux 4 joueurs de l'equipe qui capture, a chaque capture — un miroir par joueur
  du score d'equipe (§3).
- **Ce que la jauge (tag 3) est vraiment** (lu dans la serie autour de 122-126 s sur `01e1f945`) :
  un compteur de TRANSFERT d'environ une seconde, +0,1 par 100 ms quand une equipe est seule dans
  la colline (le slot d+2 dit laquelle), qui redescend sinon, et dont l'arrivee a ~1,0 fait
  basculer le PROPRIETAIRE (slot d+1) puis retombe a 0. Ce n'est pas la progression vers le
  point (30 s de garde) — cf. decouverte n. 4.
- **Le trio de fin de match** : `0x6050ABD7`, `0x3327C7DA`, `0xF2F9EB27` sur trois slots
  consecutifs, emis 15 a 21 ms APRES la capture terminale, sur les 4 films (cf. decouverte n. 1).

## 3. La mesure (CT.1.2) — tableau par film et par tag

Oracle : `01e1f945` 5 increments (109 984 / 229 227 / 345 912 / 434 220 / **539 244** ms),
`0a247154` 6 (201 041 / 251 758 / 384 609 / 511 338 / 635 180 / **785 502**), `606d9844` 3
(105 041 / 155 808 / **233 520**), `8076f97f` 3 (184 545 / 265 446 / **347 261**) — le dernier
de chaque film est terminal. Niveau du hasard d'UNE bascule : 1,9 / 1,5 / 2,6 / 1,8 %.

### 3.1 Le tag 5 (hors liste du plan) — lecture D

| film | slot | bascules | (i) exclusivite | (ii) TOUS les increments | (ii) hors terminal | ecarts bascule - increment | (iii) decale +20 s | (iii) slots permutes | hasard N bascules | verdict lettre | verdict « changements de colline » |
|---|---|---|---|---|---|---|---|---|---|---|---|
| `01e1f945` | 1471 | 4 | **98,2 %** | 4/5 = 80,0 % | **4/4 = 100 %** | +20, +21, +21, +21 ms | 0,0 % | 20,0 % (< 40 %) | 1,6 % | NON TENU | **TENU** |
| `0a247154` | 1622 | 5 | **98,7 %** | 5/6 = 83,3 % | **5/5 = 100 %** | +21, +21, +20, +21, +20 ms | 0,0 % | 16,7 % (< 41,7 %) | 1,4 % | NON TENU | **TENU** |
| `606d9844` | 1498 | 2 | 94,4 % | 2/3 = 66,7 % | **2/2 = 100 %** | +15, +15 ms | 0,0 % | 33,3 % (= 33,3 %) | 1,3 % | NON TENU | NON TENU ((i) 94,4 %) |
| `8076f97f` | 1540 | 2 | **95,5 %** | 2/3 = 66,7 % | **2/2 = 100 %** | +15, +13 ms | 0,0 % | 25,0 % (< 33,3 %) | 1,2 % | NON TENU | **TENU** |

- (i) : les « autres slots porteurs » sont le trio de fin de match (3 slots pendant les 7 a
  12 dernieres secondes) ; sur `606d9844` (235 s, la periode jugee ne commence qu'a 105 s) ce
  trio pese 5,6 points — d'ou 94,4 %. Hors emissions de fin de match : 100 % sur les 4 films.
- (ii) : les 13 changements de colline des 4 films tombent tous entre +13 et +21 ms de
  l'increment (un paquet plus tard). L'increment TERMINAL n'est suivi d'aucune bascule : la
  colline suivante n'existe pas, le match est fini. A la lettre (« >= 90 % des increments »), un
  designateur ne peut donc JAMAIS tenir sur un match clos par la limite de score (n <= 6
  increments) — la clause est vide par construction pour cette structure de donnee, comme le lot
  C-bis l'avait deja rencontre pour ses fenetres. Les deux chiffres sont publies, la lettre en
  premier.
- (iii) : decale +20 s = 0 % partout ; « slots permutes » = les emissions du trio, qui tombent a
  +15..21 ms de la capture terminale (donc 1/n) ; le hasard de N bascules est a 1,2-1,6 %.

### 3.2 Les candidats du plan et les temoins

| film | tag 1 (F/D) | tag 2 (F) | tag 6 (D) | tag 7 B (D) | tag 8 B (D) | tags 9-15 B | tag 4 temoin (D) | tag 3 temoin (D) |
|---|---|---|---|---|---|---|---|---|
| `01e1f945` | sans objet | sans objet | sans objet | (i) 0 % ; (ii) 80 % ; hasard N 56,8 % | (i) 7,9 % ; (ii) **5/5** ; ecarts +3..+4 ms | 0-1 bascule ; (ii) 0-20 % | 106 bascules ; (i) 0,2 % ; (ii) 20 % ; hasard N 33 % | 793 bascules ; (ii) 20 % ; hasard N 94,7 % |
| `0a247154` | 1 bascule ; (ii) 0 % | sans objet | 1 bascule ; (ii) 0 % | (i) 0,7 % ; (ii) 66,7 % ; hasard N 61,9 % | (i) 4,2 % ; (ii) **6/6** ; +3..+5 ms | 0-2 bascules ; (ii) 0 % | 205 bascules ; (i) 100 % ; (ii) 83,3 % ; decale 66,7 % ; hasard N 40 % | 1 bascule (pas de jauge) |
| `606d9844` | 1 bascule ; (ii) 0 % | sans objet | sans objet | (i) 0,4 % ; (ii) 0 % ; hasard N 63 % | (i) 3,3 % ; (ii) **3/3** ; -1..-2 ms | 0-1 bascule ; (ii) 0 % | 14 bascules ; (i) 0,4 % ; (ii) 0 % ; permute 66,7 % | 80 bascules ; (ii) 0 % ; hasard N 52 % |
| `8076f97f` | sans objet | 1 bascule ; (ii) 0 % | sans objet | (i) 0 % ; (ii) 33,3 % ; hasard N 64,7 % | (i) 0 % ; (ii) **3/3** ; -2..+3 ms | 0-2 bascules ; (ii) 0 % | 37 bascules ; (i) 0,3 % ; (ii) 0 % ; hasard N 22 % | 274 bascules ; (ii) 0 % ; hasard N 80 % |

- **Aucun candidat de la liste du plan ne tient** : les tags 1, 2, 6 ne parlent pas ; les tags
  10-15 non plus ; le tag 7 est bavard (son hasard vaut 57-65 %) ; **le tag 8 par joueur est
  synchrone a 100 % (17/17 increments, -2 a +5 ms) mais il EST le score** — le compte de collines
  de l'equipe replique aux 4 joueurs qui la composent (valeurs 1..4) — donc un miroir de
  l'oracle, pas un drapeau d'activation, et son exclusivite est nulle (dizaines de slots
  combles). Il est publie ici comme decouverte (n. 2).
- Les temoins se comportent en temoins : le tag 3 (jauge) et le tag 4 (proprietaire) basculent
  des dizaines a des centaines de fois, leurs synchronies sont au niveau de leur hasard. Detail
  utile : des deux canaux tag 4 de l'objet, celui du slot d+1 se remet a -1 a chaque changement
  de colline (slot 1472 : 4/5 sur `01e1f945` ; 1623 : 5/6 sur `0a247154`), celui du slot d+2 non.

### 3.3 Le DELAI — « actif des la seconde 0 ou apres delai ? »

| film | origine du rejeu (ms film) | objet de mode aux images-cles : ABSENT a / PRESENT a (ms film) | creation, apres l'origine du rejeu | premier contact (jauge/proprietaire), ms film / apres l'origine | premiere bascule du designateur (2e colline), apres l'origine |
|---|---|---|---|---|---|
| `01e1f945` | 12 588 | 19 993 / 40 001 | entre 7,4 et 27,4 s | 49 443 / **36,9 s** | 97,4 s |
| `0a247154` | 9 204 | 19 996 / 40 001 | entre 10,8 et 30,8 s | 41 918 / **32,7 s** | 191,9 s |
| `606d9844` | 6 890 | 19 992 / 39 996 | entre 13,1 et 33,1 s | 44 791 / **37,9 s** | 98,2 s |
| `8076f97f` | 15 045 | 19 989 / 39 993 | entre 5,0 et 25,0 s | 55 049 / **40,0 s** | 169,5 s |

**Reponse** : la premiere colline n'est PAS active des la seconde 0. L'objet de mode
(designateur + proprietaire + capteur + jauge) est ABSENT des images-cles a 0 s et a 20 s du
film et PRESENT a 40 s, sur les 4 films — il est CREE entre 20 et 40 s du film (soit 5 a 33 s
apres la premiere position de bipede, l'origine du rejeu, selon le film), et le premier joueur y
entre 32,7 a 40,0 s apres l'origine. **Le film ne DATE pas l'activation** de la premiere colline
en delta : la premiere designation vit dans l'image-cle, et l'objet apparait entre deux
images-cles (cadence 20 s). C'est une BORNE, publiee comme telle. La production ouvre la premiere
periode au premier contact (borne haute, jamais une invention) ; le repli retro-chronologique
n'est pas code (§5).

### 3.4 Le second oracle

Les periodes publiees AVANT ce volet (methode positions) ne debutent presque jamais a une
bascule du designateur : `01e1f945` 0/20 a +/- 1 s (1/20 a +/- 5 s), `606d9844` 0/5, `8076f97f`
0/1, `0a247154` sans objet (carte hors catalogue). Les rampes de jauge, elles, sont a l'interieur
des periodes designees — c'est le diagnostic des « a-coups » du plan, chiffre.

## 4. Gate 1 — verdict, et la decision prise

> Enonce : « (i)+(ii)+(iii) tenus sur >= 3 films sur 4 => CT.1.3 ; sinon negatif + repli. »

- **A la lettre** (tous les increments, terminal compris) : **NON TENU** — aucun tag ne tient
  (ii) sur aucun film ; le tag 5 y est a 67-83 %.
- **Sur l'oracle « changements de colline » du plan** (les increments suivis d'une colline) :
  **TENU 3/4** pour le tag 5 (`01e1f945`, `0a247154`, `8076f97f` ; `606d9844` manque (i) de
  0,6 point a cause du trio de fin de match).
- **Decision de l'executeur, ecrite pour arbitrage** : CT.1.3 est BRANCHE sur le tag 5 (commit
  `3f0ed332f`, isole et reversible), parce que le seul point qui separe les deux lectures est un
  evenement qu'AUCUN designateur ne peut suivre (la capture terminale), et que le plan definit
  lui-meme l'oracle comme celui des changements de colline ; les seuils ne sont pas abaisses,
  les deux chiffres sont publies. Le repli retro-chronologique n'est PAS code (decision
  superviseur, §5). **Si le superviseur retient la lettre, il suffit de retirer `3f0ed332f`.**

## 5. CT.1.3 — ce qui est branche

- `zone_states_hill.go` : `hillDesignatorOf` elit le designateur (serie tag 5 CHAINEE ; le slot
  suivant porte un proprietaire tag 4 qui parle, >= 2 emissions — sans quoi le trio de fin de
  match serait eligible ; le plus de bascules, egalite au plus petit slot). Periodes = [premier
  contact ; b1-1], [b1 ; b2-1], ..., [bn ; derniere frame]. Grappe des positions pendant les
  montees de jauge de la periode (a defaut, la periode entiere) ; une periode que la grappe ne
  localise pas se compte (`unpaired`) et n'est pas publiee. Progression = sommet de jauge de la
  periode, absente sur une colline sans montee. Aucun proprietaire publie. `buildRampHills` =
  l'ancienne lecture, repli d'un film sans designateur (`positions+geometry`).
- `zone_states.go` : `zoneSeries.desig` (tag 5 chaine, par slot). `document_zones.go` :
  `ZoneMethodDesignator = "designator+geometry"`, semantique de `unpaired` / `hillPeriods` par
  methode. Aucun champ, aucun schema.
- Test discriminant `zone_states_hill_designator_test.go` : bornes 50-199 / 200-399 / 400-599 (le
  designateur) et non 96-295 / 296-599 (les rampes) — contre-epreuve jouee : le designateur
  debranche fait rougir le test ; un tag 5 non chaine retombe sur les rampes ; le trio de fin de
  match n'est pas elu.
- **Temoin `01e1f945` recuit par l'instrument** (`ct13_temoin_01e1f945.log`) : methode
  `designator+geometry`, **5 periodes (collines) au lieu de 20**, 5/5 localisees, zones 0 -> 6 ->
  6 -> 6 -> 0 (frames 368-973, 974-2165, 2166-3332, 3333-4215, 4216-5342), couverture 4 975 /
  5 343 frames = 93,1 % (inchangee : la premiere periode s'ouvre au premier contact, frame 368 =
  36,9 s), calque +658 o (0,055 %). Les periodes se ferment aux frames 974 / 2166 / 3333 / 4216 =
  les increments +20 ms.
- **Le repli retro-chronologique, decrit et non code** : la premiere colline est celle de la
  premiere jauge / du premier point, reputee active depuis le depart moins le delai mesure ; avec
  le designateur il ne servirait plus qu'a ouvrir la premiere periode avant le premier contact,
  dans la borne [creation, premier contact] — decision superviseur.

## 6. Statut des items

- [x] **CT.1.1** — instrument `TestCollineProprieteTemoin`, 4 films, series + resumes ecrits ;
  `Chained` par lecture ajoute au scanner (production, additif). Commit `f077cc120`.
- [x] **CT.1.2** — oracle, seuils, deux lectures, 4 films, tableaux ci-dessus, delai, second
  oracle. Commit `e3b24a0d7`.
- [x] **CT.1.3** — branchement sur le designateur, test discriminant, temoin recuit ; repli non
  code, decrit en trois lignes. Commit `3f0ed332f`.
- [!] **Gate 1 a la lettre** — NON TENU (clause (ii) vide par construction pour un designateur) ;
  TENU 3/4 sur l'oracle « changements de colline » ; branchement fait sous cette lecture, a
  arbitrer (§4).

## 7. Gates (`LOTCTER_volet1_gates.log`)

`EXIT_GOFMT=0` · `EXIT_VET_ANALYSIS=0` · `EXIT_TEST_REPLAY_FILMDEC_NOCGO=0` ·
`EXIT_TEST_ARCHLINT=0` · `EXIT_BUILD_CGO=0` · `EXIT_LINT=0` (0 issue) ·
`EXIT_TEST_REPLAYBUILD_CONTRACT_CGO=0` (en plus des gates intermediaires CT.1.1 / CT.1.2 a 0).

## 8. Cout machine (D17)

Un film par processus, avant-plan, plafond 3 Go surveille (`PeakWorkingSet64`, kill au-dela) :

| instrument | `01e1f945` | `0a247154` | `606d9844` | `8076f97f` |
|---|---|---|---|---|
| CT.1.1 (serie + rejeu) | 196 s / 220 Mo | 348 s / 219 Mo | 97 s / 218 Mo | 139 s / 203 Mo |
| CT.1.2 (oracle + mesure) | 176 s / 154 Mo (rejoue 553 s / 297 Mo sous charge) | 646 s / 219 Mo (sous charge : 2 autres executeurs decodaient) | 160 s / 247 Mo | 278 s / 87 Mo |
| CT.1.3 (temoin p2b) | 189 s / 125 Mo | — | — | — |

Pic memoire maximal : 297 Mo (10 % du plafond). Gates : build CGO ~3 min, lint ~4 min.

## 9. Decouvertes (hors perimetre — notees, NON traitees)

1. **Le « trio » de tag 5 est une structure de FIN DE MATCH, pas un nommage de zones.** Trois
   string-ids sur trois slots consecutifs (`0x6050ABD7`, `0x3327C7DA`, `0xF2F9EB27`), emis 15 a
   21 ms apres la capture terminale sur les 4 KOTH ; le trio des Strongholds du lot C-bis
   (`0x67F43AC3`, `0xD690D6B4`, `0xF2F9EB27` — le troisieme est le meme) merite d'etre relu a la
   lumiere de son INSTANT d'emission. `ZoneState.Key` (tracabilite) n'en depend pas ; son
   commentaire « cle de nommage » si.
2. **Le mode B tag 8 est le score d'equipe REPLIQUE PAR JOUEUR** (1..4, aux 4 joueurs de l'equipe
   qui capture, -2 a +5 ms de l'increment du statborg, 17/17 increments, capture terminale
   comprise) : un oracle par joueur des captures de KOTH — et sans doute d'autres modes.
3. **Le mode B tag 7** (2 slots, ~9 joueurs, 99,9 % chaine) tique a la cadence de 1 s par
   bouffees, ~35 valeurs dans [0 ; 0,3] : la forme d'un temps de colline PAR JOUEUR. Non mesure.
4. **La jauge (tag 3) de KOTH est un compteur de TRANSFERT (~1 s), pas la progression vers le
   point** ; le slot d+1 (tag 4) est le PROPRIETAIRE (remis a -1 a chaque changement de colline),
   le slot d+2 (tag 4) l'equipe PRESENTE. Consequence pour le volet 3 (jauge en direct) : en
   KOTH, `progress`/`gauge` montreront la prise de controle, pas les 30 s de garde ; le temps de
   garde vers le point n'a pas encore de canal identifie (le tag 7 par joueur en est peut-etre
   la somme).
5. **La bande de production COMBLE la plage de slots** (`worldObjectSlotBand`) : 52 slots contre
   20 stricts, et 75 % des series de `ti=13` sont du bruit sur les slots combles. Le temoin
   `Chained` par lecture existe desormais ; `zoneSeriesOf` n'en filtre que le designateur (les
   series de jauge et de proprietaire restent brutes, l'election par accord roster des Bastion
   les protege). Filtrer partout et publier `coverage.zones.slots` sur les seuls slots chaines est
   une hygiene a decider.
6. **L'activation de la premiere colline n'est pas datee par le delta**, seulement bornee par la
   cadence des images-cles (20-40 s du film). La dater demanderait soit le record de CREATION de
   l'objet dans le delta (une forme de record que l'ancrage par masque ne reconnait pas — RE),
   soit de plomber le premier horodatage d'image-cle par slot depuis le scanner jusqu'a
   `ZoneInput` (~30 lignes) pour ouvrir la premiere periode a l'image-cle de presence (0 a 20 s
   plus tot que le premier contact, avec la meme incertitude).
7. **La grappe apparie trois collines consecutives a la meme zone du catalogue** sur `01e1f945`
   (zone 6 pour les collines 2, 3, 4) : le catalogue n'a aucune forme de colline (volet 2), la
   grappe vote parmi les zones de Bastion/Extraction. La qualite de l'appariement forme <-> colline
   est l'objet de CT.2.3, pas de ce volet.
