# A — PROTOCOLE du lot Assaut (bombe, donnees de rejeu 2D)

> Ecrit et COMMITE AVANT toute mesure de A1/A3/A4, conformement au plan
> `.ai/V7.5/PLAN_ASSAUT_LOT_A_2026-08-27.md` et au contrat du chantier : protocole
> d'abord, seuils jamais abaisses apres coup, temoins obligatoires, arret propre au seuil
> rate, decouvertes consignees et jamais traitees. Un film = un processus (`filmproc`,
> plafond de mesure 2 Gio). AUCUN oracle API par joueur n'existe pour l'Assaut (verifie le
> 27/08 sur les 9 payloads bruts) : les juges sont INTERNES au film — score de mode,
> manches, kill feed, temoins de selectivite et spatiaux.

## 1. Qualification du corpus (mesuree AVANT ce commit — A0.2, log `A_P0_qualification.log`)

Passe du 2026-08-27, instrument `TestAssautA0Qualification` (gardes `ATT_FILM` +
`ASSAUT_FILM`, un film par processus, pics memoire 0,08-0,48 Gio). Trois criteres par
film : bornes de quantification, sites `assault_bomb` au catalogue d'objectifs, pont
bipede >= 50 % (plancher herite du lot O, `d8PontMinimum`).

| film | variante | carte | bornes | sites `assault_bomb` | pont bipede | verdict |
|---|---|---|---|---|---|---|
| `35b75a31` | Neutral Bomb | Origin | OUI | 0 (carte au catalogue, role absent) | 155/163 = **95,1 %** | **ADMIS** (decodable) |
| `ce083875` | Neutral Bomb | Origin | OUI | 0 (idem) | 19/180 = 10,6 % | **EXCLU** — pont sous le plancher de 50 % (meme classe que `51ebbc0f` au lot O) |
| `69b16f5d` | Neutral Bomb | Origin | OUI | 0 (idem) | 39/48 = **81,2 %** | **ADMIS** |
| `3d58eb37` | Neutral Bomb | Absolution | OUI | 0 (idem) | 61/73 = **83,6 %** | **ADMIS** |
| `34bb3bc8` | Neutral Bomb Squad | Rat's Nest | OUI | carte ABSENTE du catalogue d'objectifs | 215/247 = **87,0 %** | **ADMIS** |
| `df8fcbef` | One Bomb | Curfew | OUI | 0 (idem Origin) | 129/150 = **86,0 %** | **ADMIS** |
| `c75f33b8` | One Bomb | Curfew | OUI | 0 (idem) | 60/80 = **75,0 %** | **ADMIS** |
| `9f57c612` | One Bomb | Curfew | OUI | 0 (idem) | 75/92 = **81,5 %** | **ADMIS** |
| `1c01e34f` | Husky Raid | Urban Raid (Forge) | OUI (bornes presentes — la surprise du recensement) | carte ABSENTE du catalogue d'objectifs | 100/107 = **93,5 %** | **ADMIS** |

**CORPUS ADMIS FIGE (decodable) : 8 films — tous sauf `ce083875`.** Rien ne s'y ajoute ni
ne s'en retire apres ce commit ; un film qui echouerait en cours de mesure sort avec sa
raison et le denominateur le dit.

**LA DECOUVERTE QUI COMMANDE TOUT LE LOT (consignee au §5 du plan, PAS traitee ici) : le
catalogue d'objectifs ne porte AUCUN site `assault_bomb` pour AUCUNE carte du corpus.**
Les 4 seuls sites du catalogue vivent sur Isolation, Snowbound, The Pit et High Ground
(remakes Delta Arena, team_index -1) — aucune n'a de film. Origin, Curfew et Absolution
SONT au catalogue (extraites, 5 roles servis sur Origin) mais sans objet au role
`assault_bomb` ; Rat's Nest et Urban Raid n'y sont pas du tout. Consequence mecanique :

- **le corpus d'ANCRAGE AU SITE de A1 et A3 est VIDE (0 film sur 9)** — l'exclusion vaut
  film par film, la regle du chantier interdit de reparer les catalogues ;
- le gate A1.3 ne peut pas etre TENU (un candidat doit naitre a <= 3 m d'un site) : A1 se
  mesure quand meme pour CHIFFRER le `[!]` (denominateurs du balayage `ti=42`) ;
- le temoin spatial de A3.2 (formes decalees de 12 m) est INMESURABLE sans forme de site :
  A3 ne publiera rien, le diagnostic temporel de A3.1 est mene et consigne ;
- A4 ne depend d'aucun site : il se joue entier sur les 8 films admis.

## 2. Releves A0.3 FIGES — manches et score de mode (schema 12, meme passe, meme log)

Le score de MODE (composant 0, valeur A, slots d'equipe 6 et 8) replique le score API des
9 matchs — corroboration relevee le 27/08 sur `match_registry` (lecture seule) :

| film | manches (brutes) | explosions datees (slot -> t ms, horloge du manifeste) | score film | score API |
|---|---|---|---|---|
| `35b75a31` | 1 | s6: 304013, 541270, 787051 | 3-0 | 3-0 |
| `69b16f5d` | 1 | s6: 154305, 278617, 310215 | 3-0 | 3-0 |
| `3d58eb37` | 1 | s6: 203065, 342196, 386280 | 3-0 (s6 = camp 1) | 0-3 |
| `34bb3bc8` | 1 | s8: 427120 | 0-1 | 1-0 (s8 = camp 0) |
| `df8fcbef` | 4 | s8 m0: 255767 · s6 m1: 309284 · s8 m2: 485860 · s8 m3: 778033 | 1-3 | 1-3 |
| `c75f33b8` | 3 | s6 m0: 109549 · s6 m1: 395724 · s6 m2: 450833 | 0-3 (s6 = camp 1) | 0-3 |
| `9f57c612` | 4 | s8 m0: 83322 · s6 m1: 298489 · s8 m2: 353160 · s6 m3: 469057 | 2-2 | 2-2 |
| `1c01e34f` | 1 | s6: 150546, 273787, 400853 · s8: 335637 | 3-1 | 1-3 (s6 = camp 1) |
| `ce083875` (exclu) | 1 | s8: 512505, 947537 · s6: 686401 (+1 emission PARASITE s8 valeur 127 a 273547 ms, ecartee par la plus longue sous-suite croissante) | 1-2 | 1-2 |

**Trois constats qui LIENT les phases suivantes :**

1. **L'EXPLOSION se date, l'ARMEMENT n'a aucun increment propre** : un point de mode = une
   explosion, rien d'autre ne bouge le score. La « corroboration par armements dates » du
   plan se lit donc : armement ~ explosion - meche, la meche etant une constante moteur
   NON connue d'avance — A3.1 publie les deltas, il ne les suppose pas.
2. **`RealRounds` refuse structurellement les manches de One Bomb** : une manche y porte
   au plus UNE emission de score, sous le critere de suite coherente (valide sur
   Oddball/CTF contre les manches fantomes). Les series FILTREES ne retiennent que la
   manche 0 sur `df8fcbef`/`c75f33b8`/`9f57c612` ; le releve BRUT (log) porte les manches
   1..3 et leurs explosions, et la somme brute = le score API sur les 9 films. Toute
   mesure passant par `SeriesTotal`/`SeriesByRound` (dont le TSV de `statnames-sweep`)
   est donc PARTIELLE sur One Bomb — ses valeurs finales valent « manche 0 seulement ».
   Consigne comme decouverte (§5 du plan), pas traitee.
3. **Debuts de manche** (premier enregistrement d'entite par manche, releves au log) :
   seules les manches brutes en portent sur One Bomb — la classe « debut de manche » de
   A1.2 se calcule sur les manches BRUTES.

Oracle fige avec ce protocole : `A_oracle_participants.tsv` (104 lignes, les 9 matchs :
xuid, camp, frags/morts/assistances — releve `match_participants` du 27/08, lecture
seule). Les bots `bid(...)` n'ont pas de xuid : aucun pont ne peut les nommer.

## 3. A1 — identite de l'objet bombe (gate RECOPIE du plan, ne bouge pas)

- Instrument : recette D4 telle quelle (`attCreationsEcartees`, role `assault_bomb`,
  rayon 3,0 m `attDrapeauRayonM`), garde `ATT_FILM` + `ASSAUT_FILM`, un film par
  processus, sur les 8 films admis.
- Temoin de lisibilite (lecon D4) : un film dont AUCUNE creation ne se resout au
  catalogue d'armes est lu aux mauvaises largeurs — NON EXPLOITABLE, ni pour ni contre.
- Classes de coincidence temporelle de A1.2, ECRITES ICI avant mesure (horloge du
  manifeste, la meme que les releves du §2) :
  - **debut de manche** : creation a <= 5 000 ms APRES le premier enregistrement d'une
    manche BRUTE (§2, constat 3) ;
  - **remise en jeu** : creation dans [0, 15 000] ms APRES une explosion datee (§2) — la
    bombe reapparait apres l'explosion, le delai moteur n'est pas connu d'avance, la
    fenetre est large mais bornee et s'ecrit ici.
- Les DEUX jambes du critere (naissance au site ; coincidence) sont publiees SEPAREMENT
  au log, a titre de denominateur — mais AUCUN candidat ne peut etre elu sans la jambe du
  site (definition du plan, intacte). Avec 0 site sur 9/9 cartes, le resultat force est
  0 candidat au site par film.
- **GATE (recopie du plan A1.3, inchange)** : UN SEUL mot candidat, LE MEME sur >= 2
  films admis, temoin = 0 autre candidat. Si rate : bombe `[!]` avec chiffres, le lot
  continue en A3/A4, A2 tombe.
- Log fige : `A1_identite_bombe.log`.

## 4. A3 — etat des sites d'amorcage (diagnostic ; gate RECOPIE, publication conditionnelle)

- **GATE (recopie du plan A3.2, inchange)** : accord canal <-> armements dates >= 90 %
  des confrontations possibles sur >= 2 films admis ; temoin spatial (formes decalees de
  12 m) <= 20 % ; sinon `[!]` diagnostic consigne, rien ne se publie.
- CONSTAT D'ENTREE : sans forme de site au catalogue (§1), le temoin spatial n'a pas de
  forme a decaler — il est INMESURABLE, donc le gate ne peut pas etre TENU et A3.3 tombe
  d'avance. Le diagnostic A3.1 est mene et consigne pour la reprise.
- Diagnostic A3.1 (instrument neuf sous garde, balayage `ti=13` de la phase 2a reutilise
  tel quel — `p2aScanFilm`, garde de registre `p2aCheckRegistre`, definition de RAMPE
  intacte : >= 3 echantillons croissants, amplitude >= 4 096 quanta) ; par film admis :
  - denominateurs d'ancrage : slots de la bande, chainage scalaire / par joueur / temoin
    decale de 3 bits (la lecon BTB du registre : un chainage effondre = contamination
    d'ancrage, le film sort du diagnostic avec sa raison) ;
  - inventaire des tags par slot (3 = jauge quantifiee, 4 = canal enumerable, 5 =
    string-id) ; rampes du tag 3 avec leurs bornes [t0, t1] ;
  - confrontation temporelle : pour chaque explosion datee du §2, delta a la fin de rampe
    la plus proche AVANT elle (armement ~ explosion - meche) et delta absolu le plus
    proche — histogramme publie. C'est un DIAGNOSTIC : il nomme ce que le canal porte,
    il n'elit rien.
- Log fige : `A3_sites_amorcage.log`.

## 5. A4 — statborg Assaut (CLI `cmd/statnames-sweep`, confrontation SANS API)

- Balayage : `statnames-sweep -films` sur les 8 films admis (parent/enfant filmproc,
  TSV : identite des slots par le pont des instants de mort + valeur finale de chaque
  emplacement composant 0..27 x cotes A/B par slot joueur). Rappel du registre (27/08) :
  le pont STATBORG peut etre etroit (2-3 slots nommes sur des films au pont bipede sain)
  — les `slots_nommes` se publient par film AVANT toute lecture par joueur.
- **MOITIES DISJOINTES, ecrites ICI avant mesure** (ordre alphabetique des ids courts,
  positions impaires = recherche, paires = verification — la regle D10) :
  - **recherche : `1c01e34f`, `35b75a31`, `69b16f5d`, `c75f33b8`** ;
  - **verification : `34bb3bc8`, `3d58eb37`, `9f57c612`, `df8fcbef`**.
- Compteurs derivables SANS API, declares d'avance :
  - **explosions RETENUES par film** (§2, constat 2 — les valeurs finales du TSV passent
    par `SeriesTotal`, donc manches reelles seulement) : recherche `1c01e34f`=4,
    `35b75a31`=3, `69b16f5d`=3, `c75f33b8`=1 ; verification `34bb3bc8`=1, `3d58eb37`=3,
    `9f57c612`=1, `df8fcbef`=1. La note du plan « morts du porteur presume » tombe avec
    A1 (pas d'identite de bombe -> pas de porteur presume) : le derivable restant est
    l'explosion, et cela s'ecrit ici.
  - **morts par joueur** (controle positif de lecture, pas un verdict du lot) : le releve
    participants fige (§2).
- **Confrontation (seuils ecrits d'avance)** :
  - (i) compteur de mode : CANDIDAT tout emplacement (comp, cote) dont la SOMME des
    valeurs finales sur TOUS les slots joueurs vaut exactement les explosions retenues du
    film, sur 4/4 films de recherche (garde anti-zero automatique : attendu >= 1
    partout) ; verdict « REPLIQUE » si 4/4 aussi sur la moitie de verification, sinon
    negatif avec denominateurs.
  - (ii) controle positif : sur les films SANS manche refusee (`35b75a31`, `69b16f5d`,
    `3d58eb37`, `34bb3bc8`, `1c01e34f`), la valeur finale comp 2 B des slots NOMMES doit
    valoir les morts du releve participants pour >= 90 % des paires nommees — controle de
    lecture (le comp 2 B est le pont lui-meme cote instants ; ici on verifie les TOTAUX).
    Sur One Bomb, le controle est le nommage lui-meme (les totaux y sont tronques a la
    manche 0, constat 2 — l'ecart est ATTENDU et ne compte pas contre la lecture).
- Verdict A4.2 nomme (« le statborg replique / ne replique pas X »), log fige
  `A4_statborg_assaut.log`. Diagnostic seulement — aucune publication depuis A4.

## 6. Temoins et garde-fous du lot

- Plafond memoire de mesure 2 Gio par processus partout ; films-bombes connus
  (`51101d1d`, `a349fea8`) hors corpus mais la regle vaut pour tout film.
- Aucun seuil de ce protocole ne se rebaisse ; toute mesure non conforme sort du
  denominateur AVEC sa raison ; l'historique git temoigne que ce protocole precede les
  mesures A1/A3/A4.
- A2 (publication des vies libres de la bombe) NE S'OUVRE QUE si le gate A1.3 tient — le
  §1 etablit qu'il ne peut pas etre tenu sur ce corpus : A2 tombera avec le `[!]` de A1,
  et la re-cuisson des temoins (A2.4) n'aura pas lieu.

## 7. Sorties attendues (logs figes, commites avec leur phase)

- `A_P0_qualification.log` — A0 : qualification 9 films + releves de score (COMMITE avec
  ce protocole).
- `A1_identite_bombe.log` — A1 : denominateurs `ti=42`, jambes du critere, constat du
  gate.
- `A3_sites_amorcage.log` — A3.1 : ancrage, inventaire des tags, rampes, deltas aux
  explosions.
- `A4_statborg_assaut.log` — A4 : balayages, candidats de recherche, verdicts de
  verification, controle positif.
