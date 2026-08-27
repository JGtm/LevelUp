# PROTOCOLE PRE-ENREGISTRE — Re-mesure Oddball statborg (methode A4) + VIP statborg natif

> A ECRIRE ET COMMITTER AVANT TOUTE MESURE, comme les protocoles D et A du chantier
> obj-etat / modes-porteurs. Prochain lot Go, execution SERIALISEE (un seul build Go a la
> fois — pas de builds concurrents sur ce poste, `reference_no_concurrent_go_builds_cache_corruption`).
> Contrat : protocole d'abord, seuils JAMAIS abaisses apres coup, temoins obligatoires,
> arret propre au seuil rate, decouvertes consignees jamais traitees. Tout decodage de film
> passe par l'executeur borne `internal/filmproc` (un film = un processus, plafond MESURE
> 2 Gio, priorite basse — garde-rail `no_unbounded_film_loop_test`). AUCUNE base ouverte en
> ecriture ; lectures DuckDB par `OpenReadForQuery` uniquement.
>
> Ce protocole rouvre DEUX chantiers negatifs/non-mesures et un seul :
> - **LOT O-A4** : re-mesure du statborg Oddball. O4 a rendu « NE REPLIQUE PAS » (0 candidat
>   sur 56 emplacements, `D10_P4_statborg.log:1850-1856`). La re-mesure applique la methode
>   qui a REUSSI en Assaut (A4, `A4_statborg_assaut.log:2-4`) : confrontation par SOMME-FILM
>   immune au pont, PLUS une confrontation par-joueur sur pont ELARGI. Le negatif O4 n'est
>   PAS annule par ce protocole ; il est RE-INTERROGE par une methode dont O4 ne disposait
>   pas, avec ses propres seuils et son propre critere d'arret.
> - **LOT V** : le statborg VIP n'a jamais ete balaye (`named.go:85-107` : 0 table VIP ;
>   `NamedEventsFrom` rend nil pour un mode sans table, `named.go:174-178`). Voie neuve.

---

## 0. CORPUS ADMIS PAR MODE — FIGE (rien ne s'ajoute ni ne se retire apres ce commit)

### 0.1 Oddball — corpus deja qualifie (D10_P0), REPRIS TEL QUEL

Qualification figee au commit D10 (`D10_PROTOCOLE.md:27-37`, log `D10_P0_pont.log`,
precondition de pont >= 50 % de slots de bipede nommes, handoff acquis n°6) :

| film (id court) | carte | pont bipede | statut |
|---|---|---|---|
| `24dbb67d` | Recharge - Ranked | 87/97 = 89,7 % | ADMIS |
| `43716616` | Smallhalla | 62/72 = 86,1 % | ADMIS |
| `92f18088` | Lattice - Ranked | 106/117 = 90,6 % | ADMIS |
| `d9781168` | Dredge | 140/160 = 87,5 % | ADMIS |
| `60ae07c4`, `c88ec007` | Live Fire | bornes de quantification absentes | EXCLUS |
| `51ebbc0f` | Banished Narrows | 9/84 = 10,7 % | EXCLU D'OFFICE |

**CORPUS ODDBALL = 4 films : `24dbb67d`, `43716616`, `92f18088`, `d9781168`.** Le corpus
NE PEUT PAS s'agrandir : les 19 vieux matchs (2021-2023) n'ont aucun manifeste et l'API
rend 404/410 (`PLAN_MODES_PORTEURS_2026-08-27.md:34-37`, correction de fait n°1). Le seul
levier futur est que l'utilisateur REJOUE (le sync capture les films recents), ou que
`map_quant_bounds.json` / `map_objectives.json` soient completes pour admettre les films
Live Fire / Lattice (decouverte `PLAN_MODES_PORTEURS §4:239-244`, NON traitee ici).

### 0.2 VIP — corpus a QUALIFIER AVANT confrontation (phase V0, commitee separement)

Trois films VIP en cache, decodables a priori (manifeste + `film_chunks` presents :
`data/cache/film_chunks/{00761d27,9903b1c5,99553e4a}`), mode confirme `GameVariantCategory=23`,
8 joueurs (`PlayerType=1`) et 2 equipes par film, une seule playlist
`AssetId=1b1691dc-d8b9-4b1f-825d-cb1c065184c1` (dissection VIP, jq sur les 3 dumps).

**Phase V0 — qualification, MESUREE ET COMMITEE AVANT toute confrontation** (meme discipline
que A0.2 `A_PROTOCOLE.md:11-32` et O0.2) : par film, (a) bornes de quantification presentes,
(b) pont bipede >= 50 % (`d8PontMinimum`, plancher herite du lot O). Un film qui echoue sort
AVEC sa raison et le denominateur le dit. Le corpus VIP admis est FIGE a la sortie de V0 —
au plus 3 films `00761d27`, `9903b1c5`, `99553e4a` (ids courts, ordre alphabetique).

**Reserve de corpus ecrite d'avance** : 3 films est un corpus MINCE. Aucun split
train/test disjoint franc n'est possible (meme limite que KOTH/Oddball, `named.go:162`). Le
gate VIP (§3.5) est donc « >= 90 % sur >= 2/3 films » avec controle de STABILITE (meme comp
meilleur sur 3/3) ET test somme-film immune au pont (§3.6) — trois garde-fous a la place du
split disjoint. Tout verdict positif se publie avec cette reserve inscrite au log.

---

## 1. ACQUIS QUI NE SE RE-PROUVENT PAS · REFUTE QUI NE SE REJOUE PAS

- **Oracle Oddball officiel** (`match_objective_stats_latest`) : par joueur
  `time_as_skull_carrier_seconds`, `skull_grabs`, `skull_scoring_ticks`,
  `skull_carriers_killed`, `longest_time_as_skull_carrier_seconds`. **Un tic de score de
  crane = une seconde de portage (3-4 %)** — donc `skull_scoring_ticks` est un ACCUMULATEUR
  DE TEMPS deguise en entier (`HANDOFF_ODDBALL_PORTAGE_2026-08-27.md:26-30`). C'est le fait
  qui commande la prediction du §2.7.
- **O4 est un negatif fortement pourvu sur les colonnes de temps/tics** (14 paires non
  nulles / 15, best <= 27 %) et FAIBLEMENT pourvu sur `skull_grabs` seul (4 paires non
  nulles / 15, 80 % domine par 11 zeros) : `D10_P4_statborg.log:1835-1856`. La re-mesure ne
  rejoue pas ce que O4 a solidement etabli ; elle attaque la ou O4 etait faible (grabs
  sous-pourvu, pont de verification effondre) et ce que O4 n'a JAMAIS teste (la somme-film).
- **Delta A4/O4 — le fait central** : A4 (Assaut) confronte la SOMME des valeurs finales
  d'un comp sur TOUS les slots joueurs a une quantite film-interne (les explosions), immune
  au nombre de slots nommes ; verdict REPLIQUE, comp 0 A, 4/4 recherche + 4/4 verification
  (`A4_statborg_assaut.log:2-3`). O4 confronte PAR JOUEUR contre l'API, donc dependant du
  pont (`confront.go:153-174`, rejet des slots non nommes `confront.go:259-261`). La
  SOMME-FILM n'a JAMAIS ete appliquee a Oddball — c'est le gap nomme par la dissection
  assaut-a4.
- **Pont statborg Oddball effondre en verification** : `SlotIdentityByDeaths`
  (`sweep.go:103`) ne nomme que **2 slots sur `43716616`** et **3 sur `d9781168`** alors que
  le pont bipede y est sain (86,1 % / 87,5 %) — `D10_P4_statborg.log:1857-1862`, cause NON
  instruite (`PLAN_MODES_PORTEURS §4:245-249`). C'est ce que l'elargissement du §2.2 vise.
- **REFUTE, ne se rejoue pas** : fenetre de queue 22,6 s (bruit, 80,7 % au seul premier
  film) ; oracle du score PERSONNEL comme gate ; sommeil de replication ; VIP par le FILM
  (le film ne porte pas le bit VIP — `PLAN §2.3`, `§LOT V:212`). VIP se juge par CONTRAINTES
  API + kill feed, jamais par re-mesure du bit dans le film.

---

## 2. LOT O-A4 — RE-MESURE ODDBALL STATBORG PAR LA METHODE A4

### 2.1 Ce que la re-mesure change vs O4 (le delta, arme par arme)

O4 avait UN mode de confrontation (par-joueur contre l'API) et UN pont (deaths seuls). La
re-mesure ajoute :
1. **Test A — SOMME-FILM, immune au pont** (la methode A4) : pour chaque emplacement
   (comp, cote), somme de la valeur finale sur TOUS les slots JOUEURS (10..24,
   `IsTeamSlot` `statborg.go:149`), y compris les slots non nommes (xuid « - » que le
   balayage imprime deja, `sweep.go:128-132`) ; confrontee au TOTAL-FILM de la colonne
   oracle (somme des valeurs par joueur du film). Ne depend d'AUCUN slot nomme.
2. **Test B — PAR-JOUEUR, sur pont ELARGI** : la confrontation O4 (`confront.go`) rejouee,
   mais le pont passe de `SlotIdentityByDeaths` (deaths seuls) a `SlotIdentityResolved`
   (§2.2), ce qui remonte le denominateur de la moitie de verification.

### 2.2 Elargissement du pont — COMMENT, d'apres le delta A4/O4

Deux leviers, tous deux PRE-ENREGISTRES, aucun n'abaisse un seuil d'identite :

- **W1 — pont RESOLU au lieu de deaths-seuls.** Remplacer, dans l'enfant du balayage,
  `SlotIdentityByDeaths(recs, deaths)` (`sweep.go:103`) par `SlotIdentityResolved`
  (`slotidentity_deaths.go:95-118`), qui prend le pont par TOTAUX (frags/morts/assistances,
  `SlotIdentityFrom`) et ne se replie sur les instants de mort QUE s'ils nomment STRICTEMENT
  plus de slots (`slotidentity_deaths.go:105`). Sa doc garantit qu'il « ne peut pas degrader
  l'existant » : sur un film complet il rend au moins ce que le pont par totaux rend, soit
  jusqu'a 8/8. Les desaccords entre les deux ponts sont ECARTES, pas arbitres
  (`slotidentity_deaths.go:110-116`). Le pont par totaux exige le triplet final du joueur —
  fourni par un oracle participants FIGE (W1bis), jamais par une ouverture de base dans
  l'enfant.
- **W1bis — oracle participants Oddball fige.** Relever `match_participants` des 4 films
  admis (xuid, frags, morts, assistances), en committer un TSV
  `D10bis_oracle_participants.tsv` AVEC ce protocole (le patron exact de
  `A_oracle_participants.tsv`, `A_PROTOCOLE.md:84-86`). Les bots `bid(...)` sans xuid en sont
  exclus, aucun pont ne les nomme.
- **AUCUN relachement des constantes de prudence du pont** : `deathInstantMin = 3`,
  `deathInstantMargin = 2`, `deathInstantToleranceMS = 150`, `maxDeathsPerSlot = 1000`
  (`slotidentity_deaths.go:40-47,182`) restent FIGEES. Les toucher pour gagner des slots
  serait un reglage d'instrument apres coup — INTERDIT. L'elargissement vient de la VOIE
  (totaux ∪ deaths), pas d'un seuil abaisse.

### 2.3 Instruments et oracles figes

- Balayage : `cmd/statnames-sweep -films 24dbb67d,92f18088,43716616,d9781168`
  (parent/enfant `filmproc`, plafond 2 Gio — `sweep.go:29-86`). Le TSV publie l'IDENTITE
  (pont resolu W1) et la valeur finale de chaque emplacement comp 0..27 (`sweepMaxComp=27`,
  `sweep.go:26`) x cotes A/B par slot, slots non nommes COMPRIS (xuid « - »,
  `sweep.go:128-132`). Serie nettoyee et cumulee sur les manches (`SeriesTotal`,
  `sweep.go:121` ; cumul `named.go:225-227`).
- Oracle par-joueur : `D10_oracle_objective_stats.json` FIGE (5 colonnes de crane par xuid,
  toutes lignes y compris nulles — `D10_PROTOCOLE.md:49-53`). REPRIS TEL QUEL, ne se
  re-releve pas.
- Oracle total-film (Test A) : DERIVE du meme JSON par somme par colonne sur les joueurs du
  film — committe fige `D10bis_oracle_totaux.json` (film -> colonne -> total). Aucune mesure
  neuve : c'est une somme de l'oracle deja fige.

### 2.4 Moities disjointes — CELLES DE D10, INCHANGEES

`D10_PROTOCOLE.md:152-155` (ordre alphabetique des ids courts, positions impaires =
recherche, paires = verification) :
- **RECHERCHE : `24dbb67d`, `92f18088`** (pont large : 8 et 7 slots nommes en deaths-seuls,
  potentiellement 8/8 en resolu) ;
- **VERIFICATION : `43716616`, `d9781168`** (les deux films au pont effondre — c'est EUX que
  W1 doit remonter).
Le candidat est ELU sur la recherche et JUGE sur la verification, jamais sur la moitie qui
l'a elu. Cette repartition NE BOUGE PAS.

### 2.5 Test A — SOMME-FILM (bridge-immune) — colonnes, methode, seuils, temoin

- **Colonnes confrontees** (entiers additifs seulement — un ACCUMULATEUR se somme comme un
  compteur au niveau film) : `skull_grabs`, `skull_scoring_ticks`, `skull_carriers_killed`.
  Les durees (`time_as_skull_carrier_seconds`, `longest_...`) sont EXCLUES du Test A : leur
  total-film n'est pas un entier de comptage.
- **Methode** : pour (comp, cote), `S(film) = somme sur les slots JOUEURS de la valeur
  finale` ; `O(film) = total-film oracle de la colonne`. Egalite EXACTE `S(film) == O(film)`.
- **SEUILS, ECRITS D'AVANCE (non negociables)** :
  - CANDIDAT si `S == O` exactement sur **2/2 films de RECHERCHE** ;
  - garde anti-zero : `O(film) >= 1` requis sur les deux films de recherche (un total nul
    matcherait tout comp muet) ;
  - VERDICT « le statborg REPLIQUE le total-film de X » si `S == O` sur **2/2 films de
    VERIFICATION** ; sinon negatif avec denominateurs.
- **TEMOIN DECALE (obligatoire)** : re-apparier chaque film a l'oracle-total de L'AUTRE film
  de sa moitie (permutation cyclique du couple film -> total). Un comp est un FAUX candidat
  s'il satisfait encore `S == O_decale` sur 2/2. **Exigence : 0 comp faux candidat** sous le
  decalage (le total-film d'un autre film est un nombre etranger — un match residuel
  signalerait un compteur trivial). Publie au log.

### 2.6 Test B — PAR-JOUEUR (pont elargi) — colonnes, encodages, seuils, temoin

- **Colonnes et encodages : ceux de O4, INCHANGES** (`confront.go:33-42`, recopie du
  protocole D10 §7) : `skull_grabs` [n], `skull_scoring_ticks` [n], `skull_carriers_killed`
  [n] ; `time_as_skull_carrier_seconds` et `longest_...` en [s]=round(v) ET [ds]=round(10v),
  chacun separement. L'accent de la re-mesure est sur les ENTIERS (mission), les durees sont
  gardees pour la continuite du denominateur.
- **SEUILS, ECRITS D'AVANCE — CEUX DE D10, NON NEGOCIABLES** (`confront.go:26-28`) :
  CANDIDAT si accord >= **90 %** des paires (joueur, film) sur la RECHERCHE avec >= **6
  paires** dont >= **3 non nulles** (garde anti-zero). VERDICT REPLIQUE si accord >= **90 %**
  sur la VERIFICATION. La seule chose qui change vs O4 : le pont resolu (§2.2) qui remonte le
  denominateur de verification.
- **TEMOIN DECALE (obligatoire, absent d'O4 — ajoute ici)** : dans chaque film, permutation
  cyclique de l'affectation xuid -> oracle (chaque joueur recoit la colonne d'un AUTRE joueur
  du meme film). **Exigence : accord <= 20 %** sur la moitie de recherche. Un accord > 20 %
  sous permutation invaliderait le test (le compteur matcherait n'importe qui). Publie au log.

### 2.7 Predictions falsifiables (ecrites AVANT mesure)

D'apres la matrice statborg x mode (dissection matrice) et le fait « 1 tic ≈ 1 s » :
- `skull_grabs` (evenement DISCRET, ramassage atomique) est la MEILLEURE cible du Test B ;
  son near-miss O4 (80 %, comp 4 B, 4 non nulles) doit MONTER si le pont de verification se
  remplit. Prediction : Test B franchit 90 % pour `skull_grabs` si le pont resolu donne >= 6
  paires de verification non nulles.
- `skull_scoring_ticks` (ACCUMULATEUR, se comporte comme une duree par joueur) : prediction
  Test B NEGATIF par-joueur (confirme les 20 % d'O4, `D10_P4:1837`), mais Test A POSITIF au
  niveau film — le total-film de tics = le score de mode d'Oddball (plafond 200,
  `statMaxModeScore=250` `statborg.go:126`), vraisemblablement porte par le comp de score de
  mode (comp 0 A, `modeScoreComp=0` `score.go:27`). Les deux tests sont donc COMPLEMENTAIRES
  et leurs verdicts opposes seraient une CONFIRMATION, pas une contradiction.
- Consequence a nommer d'avance : meme un Test A positif sur les tics ne ressuscite PAS une
  attribution de portage PAR JOUEUR — il etablit seulement que le statborg porte le total de
  mode. La condition de reprise D4 (« un compteur ball-control per-joueur au statborg »)
  reste NEGATIVE au sens per-joueur tant que le Test B echoue (dissection assaut-a4, caveat).

### 2.8 CRITERE D'ARRET si le pont reste etroit malgre l'elargissement

**Test B est declare SOUS-POURVU et clos `[!]` — sans abaisser aucun seuil — si, apres W1,
la moitie de VERIFICATION (`43716616` + `d9781168`) n'atteint pas >= 6 paires (joueur, film)
confrontables dont >= 3 a valeur oracle NON NULLE, pour la colonne consideree.** Dans ce
cas :
- le verdict par-joueur n'est PAS rendu (le denominateur ne le porte pas), il est publie
  « NON MESURABLE sur ce corpus » avec le compte exact de paires atteintes ;
- SEUL le verdict du Test A (immune au pont) est publie ;
- **condition de reprise inscrite au registre** : de nouveaux films Oddball au pont statborg
  sain cote verification (l'utilisateur rejoue), OU l'elucidation — non instruite a ce jour
  (`PLAN §4:245-249`) — de l'effondrement du pont sur `43716616`/`d9781168`. Le corpus ne
  peut pas croitre autrement (§0.1).
Le test n'est NI rejoue sur une autre repartition, NI son seuil abaisse : l'arret est propre.

### 2.9 Verdicts nommes et log fige

Log `D10bis_statborg_A4.log` : par film, `slots_nommes` (pont resolu) AVANT toute lecture ;
Test A (S/O par film, candidats, verdict verification, temoin decale) ; Test B (meilleur
accord recherche par colonne, candidats, verdict verification, temoin permute) ; verdict
nomme final par colonne (« le statborg replique / ne replique pas X, au niveau
film / par-joueur »). Diagnostic seulement — aucune publication produit depuis ce lot.

---

## 3. LOT V — VIP STATBORG NATIF

### 3.1 Etat d'entree (verifie sur pieces)

- `named.go` ne declare AUCUNE table VIP (`named.go:85-107` : seulement `ObjectiveTypeFlag`
  et `ObjectiveTypeZone`) ; `NamedEventsFrom` rend nil pour un mode sans table
  (`named.go:174-178`). L'INDEX du comp qui porte `TimesSelectedAsVip` est donc INCONNU — le
  balayage le DECOUVRE, il ne le suppose pas.
- `VipStats` = 7 colonnes (extraction `objective_stats.go:141-150`, persistees
  `rows.go:315-322`) : 5 entieres (`KillsAsVip`, `VipKills`, `VipAssists`,
  `TimesSelectedAsVip`, `MaxKillingSpreeAsVip` via `intPtrFrom`) + 2 durees float64 secondes
  (`TimeAsVipSeconds`, `LongestTimeAsVipSeconds` via `objectiveDurationSeconds`
  `objective_stats.go:157-179`).
- Additivite MESUREE (dissection VIP) : `TimesSelectedAsVip`, `KillsAsVip`, `VipKills`,
  `VipAssists` — somme des 4 joueurs d'une equipe == agregat `Teams[].VipStats`, 24/24
  controles, 0 ecart. `MaxKillingSpreeAsVip` s'agrege comme un MAX (6/6), PAS un cumul —
  inadapte au modele increment-par-evenement, ECARTE des cibles de comptage.

### 3.2 Oracle VIP fige (phase V0, committee avec le protocole)

Relever par joueur (xuid) les 4 colonnes entieres additives + les 2 durees, pour les films
admis, et committer `V_oracle_vipstats.json` (film -> xuid -> colonne -> valeur), au patron
de `D10_oracle_objective_stats.json`. Source : `match_objective_stats_latest` si les 3
matchs sont ingeres ; sinon les payloads bruts `GetMatchStats` deja verifies (dissection
VIP), le releve etant fige et committe. Toutes les lignes, y compris nulles.
`TimesSelectedAsVip` varie de 0 a 7 par joueur (dissection : films `[2,2,1,2,2,5,2,0]`,
`[7,1,1,1,4,1,0,2]`, `[4,2,2,1,1,5,0,0]`) — la premisse « 2 partout » est FAUSSE, et c'est
tant mieux : la variation par-joueur, decouplee de la duree, fait la force de l'empreinte.

### 3.3 Cibles ordonnees (ecrites d'avance)

1. **`TimesSelectedAsVip`** — MEILLEURE cible : entier, additif, signature 0..7 par joueur,
   UNIQUE au VIP donc ZERO aliasing avec les comps generiques 2 A (kills) / 3 A (assists).
   C'est la cible du gate §3.5.
2. **`KillsAsVip`** — additif, plage 0..6 ; mais SOUS-COMPTE des frags -> risque d'aliasing
   avec comp 2 A. Cible SECONDAIRE sous contrainte de sous-compte (§3.5).
3. **`VipKills`** — additif, petite magnitude (max 2 sur 2 films) -> empreinte faible. Cible
   secondaire.
`VipAssists` (sous-compte de 3 A), `MaxKillingSpreeAsVip` (MAX non cumulatif),
`TimeAsVipSeconds`/`LongestTimeAsVipSeconds` (durees float, cibles directes de l'objection
duree) : HORS gate. Les durees peuvent etre confrontees en diagnostic (encodages [s]/[ds])
mais NE portent aucun verdict VIP.

### 3.4 Balayage et pont VIP

`cmd/statnames-sweep -films 00761d27,9903b1c5,99553e4a` (parent/enfant filmproc, meme
regime que §2.3). Pont d'identite RESOLU (W1, §2.2) — le VIP peut aussi avoir un pont
statborg etroit, l'immunite du test somme-film (§3.6) le couvre. Le TSV publie
`slots_nommes` par film AVANT toute lecture par joueur.

### 3.5 Test par-joueur VIP — seuils ECRITS D'AVANCE

- CANDIDAT : l'emplacement (comp, cote) dont l'accord par-joueur avec `TimesSelectedAsVip`
  (egalite entiere, encodage [n]) est le MEILLEUR en moyenne sur les 3 films.
- **GATE VERDICT (non negociable)** : « le statborg replique `TimesSelectedAsVip` » si le
  MEME comp atteint un accord par-joueur >= **90 %** sur >= **2 des 3 films**, avec >= 3
  paires non nulles par film compte (garde anti-zero).
- **STABILITE (garde-fou du corpus mince)** : le comp retenu doit etre le MEILLEUR sur 3/3
  films (pas seulement 2/3). Un comp meilleur sur 2 films mais supplante sur le 3e est
  REJETE — coincidence probable.
- **TEMOIN (obligatoire)** : attribution ALEATOIRE des comps = permutation cyclique de
  l'affectation xuid -> valeur oracle dans chaque film (« attribution aleatoire des comps »
  de la mission). **Exigence : accord <= 20 %** sur CHACUN des 3 films.
- **Cibles 2/3** (`KillsAsVip`, `VipKills`) : memes seuils, PLUS contrainte anti-aliasing —
  le comp candidat doit etre DISTINCT de comp 2 A / 3 A ET sa valeur par slot doit etre
  <= la valeur du comp generique correspondant (kills pour `KillsAsVip`) sur le meme slot
  (un sous-compte ne depasse jamais son total). Si un candidat viole le sous-compte, il est
  ECARTE.

### 3.6 Test somme-film VIP (bridge-immune) — confirmation secondaire

Comme §2.5 : `S(film) = somme sur slots JOUEURS de la valeur finale du comp` ;
`O(film) = total-film de TimesSelectedAsVip` (somme des per-joueur). CANDIDAT si `S == O`
sur >= 2/3 films (`O >= 1` requis) ; temoin decale = re-appariement cyclique film -> total,
**0 faux candidat** exige. Ce test CONFIRME §3.5 sans dependre du pont ; un desaccord entre
§3.5 (positif) et §3.6 (negatif) signalerait un pont VIP trop etroit et se dirait au log.

### 3.7 Corroboration temporelle par le kill feed (diagnostic, NON gating)

« Le kill feed date les periodes » : le fil des morts est deja decode
(`replay.ScanFilmDeaths`, exploite par `sweep.go:99`). Le comp de `TimesSelectedAsVip` doit
INCREMENTER (une unite = une selection, `incrementTimes` `named.go:312-321`) a des instants
qui suivent la mort du VIP precedent (rotation par mort) OU un pas de minuterie.
- Diagnostic : histogramme des ecarts entre chaque increment du comp candidat et
  l'evenement de mort le plus proche du kill feed. Accord = part des increments a <= **1000
  ms** (`d4EcartEvenementMS`, tolerance DEJA COMMITEE en D4 — PAS un reglage neuf) d'une
  mort.
- **Temoin PORTE DANS la metrique** (patron O3.2, `D10_PROTOCOLE.md:175-177`) : l'accord se
  mesure contre DEUX classes concurrentes — morts du VIP vs instants ALEATOIRES du match. Si
  les deux rendent le meme accord, la datation est « NON ETABLIE ». Verdict « les selections
  suivent les morts » seulement si accord morts >= **80 %** ET nettement > la classe
  aleatoire.
- Ce diagnostic n'ELIT rien et ne fait pas le gate §3.5 ; il etablit, si positif, la REGLE
  DE ROTATION (mort du VIP) que le lot V publiera pour dater les periodes de couronne. S'il
  ne se nomme pas, la regle de rotation reste `[!]` avec condition (`PLAN §LOT V:218`), le
  gate §3.5 restant valable pour l'attribution du comp.

### 3.8 Verdicts nommes et log fige

Log `V_statborg_vip.log` : qualification V0 (bornes, pont) ; `slots_nommes` par film ; Test
par-joueur (accord par comp sur les 3 films, candidat, gate 2/3, stabilite 3/3, temoin
permute) pour `TimesSelectedAsVip` puis `KillsAsVip`/`VipKills` ; Test somme-film ;
corroboration kill feed. Verdict nomme (« le statborg replique / ne replique pas
`TimesSelectedAsVip` », comp et accord chiffres). Diagnostic — la couronne VIP et
l'attribution de periode se cablent au lot R sur la DONNEE livree, pas depuis ce lot.

---

## 4. ORDRE D'EXECUTION ET CORPUS ADMIS PAR MODE

**Execution SERIALISEE — un seul build Go a la fois** (cache Go partage, builds concurrents =
corruption). L'ordre du plan maitre est respecte : O avant V (`PLAN §1.1:46-48`).

| # | Lot | Corpus admis (ids courts) | Denominateur | Instrument | Sorties |
|---|---|---|---|---|---|
| 1 | **V0** — geler oracles | Oddball 4 + VIP 3 | releve API fige | lecture DuckDB `OpenReadForQuery` / payloads bruts | `D10bis_oracle_participants.tsv`, `D10bis_oracle_totaux.json`, `V_oracle_vipstats.json`, qualification VIP commitee |
| 2 | **O-A4** — re-mesure Oddball | `24dbb67d`, `92f18088` (recherche) · `43716616`, `d9781168` (verification) | 15 paires recherche / verif elargie par W1 ; Test A immune | `cmd/statnames-sweep` (pont resolu W1 + mode somme-film W2) | `D10bis_statborg_A4.log` |
| 3 | **V** — VIP statborg | `00761d27`, `9903b1c5`, `99553e4a` (admis a V0) | 3 films, 8 joueurs/film = 24 lignes ; gate 2/3 | `cmd/statnames-sweep` + confront VIP + kill feed | `V_statborg_vip.log` |

Regles de corpus, ecrites d'avance :
- **Oddball** : corpus FIGE a 4 films (§0.1), moities disjointes FIGEES (§2.4), ne bougent
  pas. Un film qui echoue en cours de mesure (plafond, chunk illisible) sort AVEC sa raison
  et le denominateur le dit.
- **VIP** : corpus FIGE a la sortie de V0 (au plus 3 films). Pas de split disjoint franc ;
  gate >= 2/3 + stabilite 3/3 + somme-film (§0.2 reserve).
- Precondition de pont >= 50 % obligatoire dans les deux modes (handoff acquis n°6).
- Films-bombes connus hors corpus (`51101d1d`, `a349fea8`), mais le plafond 2 Gio vaut pour
  TOUT film (`D10_PROTOCOLE.md:179-180`).

Chaque lot est un commit prefixe (`oddball-a4(...)`, `vip-statborg(...)`), jamais
`git add -A`, jamais de push, sur une branche dediee depuis `feat/v75`. `.ai/thought_log.md`
et `REGISTRE_REPORTS.md` : textes fournis au CR, le superviseur consigne.

---

## 5. TEMOINS ET GARDE-FOUS COMMUNS (rappel — non negociables)

1. **Tout decodage de film sous `internal/filmproc`** : un film = un processus enfant,
   plafond MESURE 2 Gio, priorite basse, codes de sortie du protocole ; aucune base ouverte
   dans l'enfant (`sweep.go:29-86`). Le garde-rail `no_unbounded_film_loop_test` le force.
2. **Aucun seuil de ce protocole ne se rebaisse.** Test A : egalite exacte 2/2 + 2/2. Test B
   Oddball : 90 % / 6 paires / 3 non nulles (`confront.go:26-28`), temoin permute <= 20 %.
   VIP : 90 % sur 2/3, stabilite 3/3, temoin <= 20 %. Corroboration temporelle : 80 % contre
   deux classes, tolerance 1000 ms (deja commitee). Toute mesure non conforme sort du
   denominateur AVEC sa raison.
3. **Constantes d'identite FIGEES** (`slotidentity_deaths.go:40-47,182`) : ni min, ni marge,
   ni tolerance, ni plafond ne se touchent pour gagner des slots. L'elargissement vient de la
   VOIE (pont resolu), pas d'un seuil.
4. **Temoin decale/permute obligatoire dans CHAQUE test** — c'est l'equivalent statborg du
   temoin spatial 12 m du chantier Oddball (`HANDOFF §1 acquis n°5`) : un test dont le temoin
   ne s'effondre pas n'a rien mesure.
5. **Critere d'arret ecrit (§2.8)** : un pont qui reste etroit ferme le Test B par-joueur en
   `[!]` propre avec condition de reprise, sans rejouer ni abaisser. Le Test A (immune)
   garde toujours sa puissance.
6. **L'historique git temoigne que ce protocole precede les mesures** : commit du protocole +
   oracles figes AVANT tout `-films`.

---

## 6. SORTIES ATTENDUES (logs figes, committes avec leur phase)

- **V0** (committe avec ce protocole) : `D10bis_oracle_participants.tsv`,
  `D10bis_oracle_totaux.json`, `V_oracle_vipstats.json`, tableau de qualification VIP.
- **O-A4** : `D10bis_statborg_A4.log` — balayage (pont resolu, slots nommes par film), Test A
  somme-film (candidats recherche, verdicts verification, temoin decale), Test B par-joueur
  (candidats, verdicts, temoin permute), verdicts nommes par colonne, application du critere
  d'arret le cas echeant.
- **V** : `V_statborg_vip.log` — qualification, balayage, Test par-joueur
  (`TimesSelectedAsVip` puis `KillsAsVip`/`VipKills`), Test somme-film, corroboration kill
  feed a deux classes, verdicts nommes.

Gate du lot (O-A4 + V) : les logs figes existent, chaque test a son verdict ecrit, le
protocole n'a pas bouge apres coup (historique git), `go vet ./...` et la compilation des
packages touches passent, CR au superviseur avec textes journal/registre PRETS.

---

### Note de perimetre (lecture seule, cette session)

Ce protocole est un LIVRABLE TEXTE. Il n'a ete produit qu'a partir d'artefacts DEJA committes
(sources Go citees file:line, logs figes `D10_P4`/`A4`, protocoles `D10`/`A`, plan
`PLAN_MODES_PORTEURS`, handoff Oddball, dumps VIP en cache). Aucune mesure neuve, aucun
build/test/run Go. Points NON etablis par les artefacts, laisses ouverts et a ne pas
inventer : (a) la CAUSE de l'effondrement du pont statborg sur `43716616`/`d9781168` (non
instruite) ; (b) que `SlotIdentityResolved` remontera EFFECTIVEMENT le pont de verification
Oddball (attendu par construction — « ne peut pas degrader » — mais NON mesure) ; (c) que la
somme-film d'un comp Oddball vaut le total de tics/grabs (c'est precisement l'objet du Test
A) ; (d) la decodabilite reelle des 3 films VIP sous filmproc et leur pont bipede (objet de
la phase V0). La « methode A4 » de la mission est ici formalisee comme la confrontation par
somme-film immune au pont, telle que `assaut_a4_confront_test.go` la realise et que
`A4_statborg_assaut.log:2-3` la publie.