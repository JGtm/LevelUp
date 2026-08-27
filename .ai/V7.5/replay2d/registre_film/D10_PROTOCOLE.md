# D10 — PROTOCOLE de la campagne diagnostique « fragmentation des longs portages » (Oddball)

> Ecrit et COMMITE AVANT toute mesure de O1/O3/O4, conformement au plan
> `.ai/V7.5/PLAN_MODES_PORTEURS_2026-08-27.md` (lot O, phases O0-O4) et au contrat du
> chantier obj-etat : protocole d'abord, seuils jamais abaisses apres coup, temoins
> obligatoires, arret propre. D10 est un DIAGNOSTIC : elle ne re-mesure pas le gate
> historique (ce serait O2, un second lot) — elle NOMME par les chiffres pourquoi un long
> portage se fragmente, elucide les `th=10`, et inventorie le statborg Oddball.
>
> CE QUI EST ACQUIS ET NE SE RE-PROUVE PAS (handoff §1) : oracle API officiel
> (`match_objective_stats_latest`, un tic de crane = une seconde de portage a 3-4 % pres) ;
> primitive de proximite bimodale (seuil constate ~1,5 m = `originDropMaxDist`) ;
> « mourir = lacher » 91,7 % ; sommeil refute ; signal SPATIAL (temoin 12 m : 0-3,3 %) ;
> precondition de pont >= 50 % de slots nommes.
> CE QUI EST REFUTE ET NE SE REJOUE PAS (handoff §2) — en particulier :
> **INTERDIT DE LA FENETRE DE QUEUE** : la fenetre a 22,6 s ferait franchir 80,7 % au seul
> premier film — bruit demontre au CR D9, choisi apres coup. Elle ne se recalcule pas, ne
> se cite pas comme signal, et aucun chiffre de D10 ne s'interprete a travers elle.
> La fenetre de la chaine reste `d9FenetreMS = 8000 ms`, FIGEE.

## 1. Corpus — qualification du pont (mesuree AVANT ce commit, instrument D8 tel quel)

Passe du 2026-08-27, un film par processus (`TestOddballSommeilD8`, gardes
`ATT_FILM` + `ODDBALL_FILM`, plafond mesure 2 Gio), log fige `D10_P0_pont.log`.
Le taux est celui que publie `d8Charge` : slots de bipede nommes par le pont / slots.

| film | carte | pont (slots nommes) | verdict |
|---|---|---|---|
| `60ae07c4` (2024-10) | Live Fire - Ranked | indecodable : bornes de quantification de la carte ABSENTES du catalogue (`map_quant_bounds.json`) — les positions ne sortent qu'en quanta, aucune distance n'a de sens | **EXCLU** (raison nommee, pas repare) |
| `92f18088` | Lattice - Ranked | 106/117 = **90,6 %** | **ADMIS** — reserve : 0 socle `oddball_spawn` au catalogue d'objectifs (carte absente de `map_objectives.json`) → la classe « retour » est INDISPONIBLE sur ce film, un trou ferme au socle y sortira « sans traversee » |
| `24dbb67d` | Recharge - Ranked | 87/97 = **89,7 %** | **ADMIS** |
| `43716616` | Smallhalla | 62/72 = **86,1 %** | **ADMIS** |
| `51ebbc0f` | Banished Narrows | 9/84 = 10,7 % (mesure D9, lecon commitee) | **EXCLU D'OFFICE** |
| `c88ec007` | Live Fire | indecodable : memes bornes absentes (Live Fire hors catalogue de quantification) | **EXCLU** |
| `d9781168` | Dredge | 140/160 = **87,5 %** | **ADMIS** |

**CORPUS ADMIS FIGE : 4 films — `24dbb67d`, `43716616`, `92f18088`, `d9781168`.**
Rien ne s'y ajoute ni ne s'en retire apres ce commit. Un film qui echouerait en cours de
mesure (plafond memoire, chunk illisible) sort avec sa raison et le denominateur le dit.

## 2. Oracles FIGES (releves du 2026-08-27, vue `match_objective_stats_latest` UNIQUEMENT)

58 lignes relevees sur les 7 matchs (le compte attendu). Fichiers figes, commites avec ce
protocole :

- `D10_oracle_api_portage.json` — xuid -> `time_as_skull_carrier_seconds` (joueurs a temps
  non nul ; meme contrat que l'oracle D6, dont les valeurs communes sont IDENTIQUES).
- `D10_oracle_api_tics.json` — xuid -> `skull_scoring_ticks` (joueurs a tics non nuls).
- `D10_oracle_objective_stats.json` — xuid -> les 5 colonnes de crane (`time_as_skull_
  carrier_seconds`, `skull_grabs`, `skull_scoring_ticks`, `skull_carriers_killed`,
  `longest_time_as_skull_carrier_seconds`), TOUTES les lignes y compris nulles ; la 58e
  ligne (bot `bid(44.0` de `43716616`) est ECARTEE — un bot n'a pas de xuid, aucun pont ne
  peut le nommer.

**Plus gros porteur API par film admis** (le denominateur de O1.2) :
`24dbb67d` -> `2533274822068549` (79,5 s) ; `43716616` -> `2535462017472349` (94,1 s) ;
`92f18088` -> `2533274858283686` (148,3 s) ; `d9781168` -> `2533274974091007` (116,1 s).

## 3. La CHAINE est celle de D9, FIGEE — rien ne se regle

O1 rejoue la reconstruction premiere-traversee de D9 a parametres IDENTIQUES :
`d9FenetreMS = 8000`, `d6RayonRamassageM = 1,5 m` (= `originDropMaxDist`),
`d6AmbiguiteM = 1,0 m`, `d6EcartMaxMS = 250`, `d6SocleM = 3,0 m`, pas d'image
`d7ImageUS = 100 ms`, cloture par la mort du porteur (`d9FinPortage`). L'instrument D10
REPRODUIT la boucle de `d9Reconstruit` en publiant le detail par trou/vie ; il VERIFIE a
chaque film que ses totaux par joueur coincident avec la sortie de `d9Reconstruit`
inchangee (auto-controle publie dans le log — un ecart invaliderait la mesure du film).

## 4. DEFINITIONS (O1)

Sequence du film : vie libre 0, trou 0, vie libre 1, trou 1, ... Le trou i separe la vie i
de la vie i+1. Attribution du trou i = celle de la chaine D9 (premier traversant du lieu de
repos dans la fenetre, ambiguite -> non attribue, cloture a la mort).

- **Porteur precedent de la vie i** : l'attribution du trou i-1 (zero si non attribue ou si
  la vie est la premiere du film).
- **Ramasseur de la vie i** : l'attribution du trou i (celui qui commence quand la vie se
  tait).
- **Distance porteur-precedent -> re-prise** : distance du porteur precedent (sa piste,
  tolerance d'echantillon `d6EcartMaxMS`) a la DERNIERE position de la vie i, a l'instant
  de la traversee retenue par la chaine pour le trou i.
- **Meme-joueur** : porteur precedent == ramasseur (tous deux non nuls).
- **Vie libre INTERIEURE** : une vie libre i telle que (1) le trou i-1 ET le trou i
  existent et sont tous deux ATTRIBUES par la chaine, et (2) la vie ne nait PAS au socle
  (`d6NaitAuSocle` faux ; sur `92f18088`, sans socle au catalogue, la condition (2) est
  vide et cela se dit dans le log). C'est l'interruption d'un portage reconstruit — le
  **micro-lacher** est son cas court : bousculade, escalade, saut, l'objet touche le sol
  et re-emet quelques images.

### Les 4 causes de la ventilation (O1.2) — nommees d'avance

Secondes manquantes du plus gros porteur API P : `M(P) = max(0, API(P) - reconstruit(P))`.
Chaque trou i dont le PORTEUR PRECEDENT est P (l'attribution du trou i-1 = P) verse dans
UNE cause :

- **(a) vol d'attribution par un TIERS** : trou i attribue a T != P alors que P est a
  <= 1,5 m (`d6RayonRamassageM`) de la re-prise a l'instant de la traversee. Secondes
  comptees : la duree ATTRIBUEE du trou i (celles que T recoit a la place de P).
- **(b) MEME joueur re-compte (fragmentation sans vol)** : trou i attribue a P lui-meme.
  Secondes comptees : la duree de la vie libre i + la tete perdue du trou i (instant de
  traversee - debut du trou) — les secondes qu'aucune attribution ne recoit alors que
  l'API les credite vraisemblablement a P d'un seul tenant.
- **(c) trou sans traversee** : trou i NON attribue (ni retour) ; secondes comptees : la
  duree totale du trou i.
- **(d) autre / hors intervalle** : `M(P) - (a) - (b) - (c)`, plancher zero. Si
  (a)+(b)+(c) DEPASSE M(P), le depassement est PUBLIE tel quel (les causes se mesurent sur
  le film, M sur l'API — un excedent se dit, il ne se lisse pas).

### Distribution des durees (O1.3)

Durees des vies libres INTERIEURES (par film ET agregees sur le corpus admis), quantiles
q50/q75/q90/max. **REGLE DU PLAN, RECOPIEE SANS MODIFICATION : le N de chainage d'une
eventuelle O2 = q90 arrondi a la seconde superieure — sur la distribution AGREGEE du
corpus admis.** D10 le publie ; elle ne s'en sert pas.

## 5. SEUIL D'OUVERTURE DE O2 — recopie du plan §O1, sans modification

> Causes (a)+(b) couvrent >= 50 % des secondes manquantes du plus gros porteur sur >= la
> moitie des films admis. Sinon : P1 INFIRMEE, le lot s'arrete en O3/O4 et le CR le dit.

Corpus admis = 4 films : « la moitie » = **2 films sur 4**. D10 CONSTATE ce seuil, elle ne
decide rien — l'ouverture de O2 est un arbitrage superviseur.

## 6. O3 — ce que datent les `th=10` (diagnostic borne)

- Evenements : `objectiveevents.Extract` (variante `Oddball:Arena`), type `skull_carry`,
  chacun date (horloge du MATCH) et NOMME (xuid de l'acteur).
- Transitions de la chaine : NAISSANCES de vies libres (`T0US` — l'objet reapparait : un
  LACHER ou une re-creation) et SILENCES (`T1US` — l'objet se tait : un RAMASSAGE). Les
  debuts/fins de trous sont LES MEMES instants, par construction.
- Conversion d'horloge : `matchMS = (US - ScanFilmClockOrigin) / 1000` — l'expression de
  D4, celle qui a etabli la coincidence a 3-6 ms entre creations et `th=10`.
- Publication : histogramme des ecarts (au plus proche des deux classes), accord par
  classe = part des `th=10` a <= 1000 ms (`d4EcartEvenementMS`, tolerance DEJA COMMITEE en
  D4 — pas un reglage neuf) d'une transition de la classe ; et rapport compte-`th=10` /
  tics API par joueur (oracle tics fige).
- **Verdict (seuil du plan O3.2, recopie)** : « les `th=10` datent X » si accord >= 80 %
  pour la classe X (naissances, silences, ou l'union) ; sinon « NON ETABLI ». S'ils datent
  ramassages et/ou lachers a >= 80 %, ils deviennent l'ANCRAGE CANDIDAT de O2 (une seule
  remesure, protocole amende et commite avant — hors de ce lot).

## 7. O4 — inventaire du statborg (CLI durable `cmd/statnames-sweep`)

- Lecture de film sous `internal/filmproc` OBLIGATOIRE : un film = un processus enfant,
  plafond MESURE 2 Gio, priorite basse, codes de sortie du protocole. AUCUNE base ouverte.
- Par film admis, l'enfant publie : la valeur FINALE (serie nettoyee et cumulee sur les
  manches, `objectiveevents.SeriesTotal`) de CHAQUE emplacement (composants 0..27, cotes
  A et B) pour chaque slot de joueur, et le pont d'identite `SlotIdentityByDeaths`
  (slot statborg -> xuid par les instants de mort — le pont dont la garde de mode du
  drapeau se protege ; ici il tourne SOUS le plafond `maxDeathsPerSlot` ET sous filmproc,
  exactement le regime des instruments D4-D9 qui l'appellent deja sur Oddball).
- **MOITIES DISJOINTES, ecrites ICI avant mesure** (ordre alphabetique des ids courts,
  positions impaires = recherche, paires = verification) :
  - **recherche : `24dbb67d`, `92f18088`** ;
  - **verification : `43716616`, `d9781168`**.
- Confrontation aux colonnes de `D10_oracle_objective_stats.json`, par (composant, cote) x
  (colonne, encodage). Encodages testes, declares d'avance : compteurs entiers
  (`skull_grabs`, `skull_scoring_ticks`, `skull_carriers_killed`) -> egalite avec la
  valeur ; colonnes de temps (`time_as_skull_carrier_seconds`,
  `longest_time_as_skull_carrier_seconds`) -> egalite avec `round(v)` (secondes) OU
  `round(10*v)` (dixiemes), chaque encodage confronte SEPAREMENT.
- **Seuils, ecrits d'avance** : candidat si accord >= 90 % des paires (joueur, film)
  confrontables sur la moitie de RECHERCHE avec >= 6 paires, dont >= 3 paires a valeur
  oracle NON NULLE (garde anti-zero : une colonne nulle partout « matche » tout compteur
  muet). Verdict « le statborg REPLIQUE ce compteur » si l'accord du candidat est >= 90 %
  sur la moitie de VERIFICATION (denominateurs publies). Sinon : negatif, avec son
  denominateur.

## 8. Temoins et garde-fous de D10

- O1 est un DIAGNOSTIC de la chaine D9 deja jugee contre ses deux temoins (spatial 12 m a
  0,0-3,3 %, joueur-aleatoire) — D10 ne rejoue pas ces temoins, elle decompose un ecart
  deja etabli. Son garde-fou propre est l'AUTO-CONTROLE (§3) : totaux D10 == totaux
  `d9Reconstruit`, publie par film.
- O3 porte son temoin DANS sa metrique : l'accord se mesure contre DEUX classes de
  transitions concurrentes (naissances vs silences) — si les deux classes rendent le meme
  accord, rien n'est date et le verdict est « NON ETABLI ».
- O4 porte le sien par construction : moities disjointes + garde anti-zero.
- Plafond memoire 2 Gio par processus partout ; films-bombes connus (`51101d1d`,
  `a349fea8`) hors corpus mais la regle vaut pour tout film.
- Aucun seuil de ce protocole ne se rebaisse ; toute mesure non conforme sort du
  denominateur AVEC sa raison.

## 9. Sorties attendues (logs figes, commites avec leur phase)

- `D10_P1_ventilation.log` — O1 : detail par vie libre, ventilation (a)/(b)/(c)/(d) par
  film, distributions q50/q75/q90/max, auto-controle, constat du seuil d'ouverture.
- `D10_P3_th10.log` — O3 : histogrammes, accords par classe, rapports aux tics, verdict.
- `D10_P4_statborg.log` — O4 : balayage par film, candidats de la moitie de recherche,
  verdict de la moitie de verification.
