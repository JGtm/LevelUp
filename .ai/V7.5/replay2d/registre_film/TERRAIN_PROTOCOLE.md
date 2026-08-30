# TERRAIN_PROTOCOLE — confrontation du canal de score a la VERITE TERRAIN Oddball

> Ecrit et COMMITE AVANT toute mesure (regle du chantier). Aucune valeur ci-dessous n'est
> ajustee apres coup. Film calibrant : d9781168 (Dredge, 2 manches). Corpus de generalisation :
> les films Oddball du cache (oracle fige `D10_oracle_objective_stats.json`).

## 0. Question tranchee par ce lot

Le canal de SCORE decode du film (score personnel a la ms, et emplacements du statborg)
reproduit-il QUI PORTE le crane, confronte a une verite terrain observee image par image ?
Si oui, on publie le porteur. Sinon, negatif propre + ou il echoue.

## 1. Verite terrain FIGEE (manche 1 de d9781168)

Source : `.ai/V7.5/replay2d/ODDBALL_VERITE_TERRAIN_d9781168.md` (JGtm, Theater, 2026-08-28).
Sequence des PRISES de la manche 1 (temps de LECTURE mm:ss -> secondes) et porteur d'intervalle :

| # | prise (mm:ss / s) | joueur qui prend | porte jusqu'a | mecanique de fin |
|---|---|---|---|---|
| 1 | 0:48 / 48  | SHROOM GOD3261 | ~1:01 | lache |
| 2 | 1:05 / 65  | scuderiasven   | 2:01  | MORT DANS LE VIDE -> respawn socle 2:04 |
| 3 | 2:04 / 124 | LadyJezz       | 2:07  | mort NORMALE -> crane au sol |
| 4 | 2:10 / 130 | L0UDEN13       | 2:32  | kill au crane a 2:28 ; mort -> lache |
| 5 | 2:35 / 155 | scuderiasven   | 2:37  | mort -> crane au sol |
| 6 | 2:40 / 160 | LadyJezz       | 2:48  | MORT DANS LE VIDE -> respawn socle 2:53 |
| 7 | 3:00 / 180 | L0UDEN13       | 3:02  | lance puis meurt -> crane au sol |
| 8 | 3:09 / 189 | DinoR00        | ~3:35 | lache |
| 9 | 3:39 / 219 | SHROOM GOD3261 | fin M1| — |

9 intervalles de porteur (le gate parle de « >= 8/10 » ; on statue sur ces 9 transitions
documentees et on reporte N/9, seuil = 8/9 pour tenir l'esprit du 80 %). Roster :
Eq0 = LadyJezz, SHROOM GOD3261, scuderiasven, OFB4203689 ; Eq1 = DinoR00, EvilestSmile946,
JGtm, L0UDEN13. Le mm:ss est un temps de LECTURE : l'offset vers l'horloge du film est
INCONNU et se MESURE (section 4), il ne se suppose pas.

## 2. Ce qu'on decode (etabli, non re-etabli ici)

- Score PERSONNEL a la ms : `objectiveevents.PersonalScoreComponent` (comp 1, valeur B),
  mesure 374/381 contre capture Cheat Engine. C'est un TOTAL (frags 100, assists 50,
  +10 prise, +50 controle, tics de score).
- Emplacements du statborg par (comp 0..27, cote A/B), par slot d'entite, PAR MANCHE :
  `objectiveevents.SeriesByRound(recs, StatComponent{Comp,SideB}, false)` — series non
  cumulees par manche, plus longue sous-suite non decroissante.
- Pont slot -> xuid SANS base : `SlotIdentityByDeaths` (fil des morts du film,
  `replay.ScanFilmDeaths`). xuid -> gamertag : `match_participants` en lecture
  `duckdb.OpenReadForQuery` (le serveur tient la base RW ; read_only echouerait).
- Oracle par xuid (fige, sans base) : `D10_oracle_objective_stats.json` —
  `time_as_skull_carrier_seconds`, `skull_scoring_ticks`, `skull_grabs`,
  `skull_carriers_killed`. Observation calibrante : `skull_scoring_ticks` ~ 1 tic/seconde
  de portage (113 tics <-> 116,1 s ; 102 <-> 105 ; 50 <-> 51,1 ...).

## 3. MULTI-MANCHE (piege prioritaire, d9781168 = premier mode a 2 manches en replay)

- Le crane se REINITIALISE entre les manches (respawn socle ; porteur M1 sans rapport avec M2).
- Le score personnel du statborg est indexe PAR MANCHE dans le getter natif
  (`world + slot*0x88 + equipe*0x1DF0 + 0x38 + manche*4`) : `StatRecord.Round` porte la manche
  et `SeriesByRound` rend des series QUI REPARTENT DE ZERO par manche. A VERIFIER
  empiriquement (dump : round 0 et round 1 doivent tous deux partir pres de 0).
- La borne M1/M2 se lit par la manche des enregistrements (`RealRounds`) — pas de nombre de
  manches cable. Une prise ne se FERME jamais par une prise fantome au passage de manche : on
  borne toutes les fenetres de portage PAR round.
- Si l'infra de rejeu (document, `objectiveObjects`, `zone_states`, calque) suppose une seule
  manche quelque part, c'est une DECOUVERTE a signaler au CR (pas un correctif en dur ici).

## 4. Alignement d'horloge (offset UNIQUE, mesure)

On aligne la manche 1 seule. Le canal de score donne, par joueur, des instants de PRISE
(emplacement skull_grabs — voir section 5). Pour un offset candidat o (balaye au pas de
0,5 s sur une plage large, ex. -60..+60 s), on compte les prises terrain (section 1) dont
une prise decodee du MEME joueur tombe a |t_decode - (t_terrain + o)| <= tolerance (5 s).
o* = l'offset qui MAXIMISE ce compte. Un seul o* pour toute la manche. On publie o* et son
compte. Aucun offset par intervalle.

## 5. Identification des emplacements Oddball (general, par l'oracle — pas par d9781168)

Aucun emplacement Oddball n'est encore nomme (`namedStatSlots` n'a pas de table `skull`).
On les identifie par CONFRONTATION A L'ORACLE, films confondus, jamais par « ca colle sur
d9781168 » :
- `skull_grabs` = l'emplacement (comp,cote) dont le total par joueur (somme des manches)
  egale `skull_grabs` de l'oracle sur le PLUS de (film,joueur).
- `skull_scoring_ticks` = idem contre `skull_scoring_ticks`.
Un emplacement n'est retenu que s'il est le meilleur candidat sur >= 2 films et sans
concurrent egal. On publie les taux d'accord par emplacement.

## 6. Reconstruction du porteur (regle UNIVERSELLE, calibree par le terrain)

- PORTEUR a l'instant t = le joueur dont l'emplacement `skull_scoring_ticks` s'incremente
  autour de t (tics ~1/s pendant le portage). Un intervalle de portage = un train de tics
  consecutifs du meme joueur (trou <= tolerance de tic, ex. 3 s).
- PRISE = increment de `skull_grabs`. FERMETURE = mort du porteur (fil des morts /
  `skull_carriers_killed`) OU nouvelle prise d'un autre joueur OU fin de manche.
- Mecanique VIDE : une mort dans le vide renvoie le crane au socle (pas de lacher local) ;
  cote score, cela n'ouvre pas de porteur — la reprise se fait au socle par une nouvelle
  prise. On ne modelise donc PAS de position ici : le train de tics porte le porteur, la
  mecanique vide n'affecte que la POSITION du crane (couche separee), pas l'attribution.
- Tout est borne PAR MANCHE (section 3).

## 7. GATES (ecrits avant mesure)

- GATE 3 (central, manche 1 de d9781168) : apres offset unique o*,
  (a) les prises decodees reproduisent >= 8/9 des prises terrain (bon joueur, |dt| <= 5 s) ;
  (b) le porteur reconstruit (train de tics) egale le porteur observe sur >= 8/9 intervalles.
  Verdict global = min(a,b) >= 8/9. Sinon : consigner OU ca echoue (intervalle, cause),
  verdict [!], STOP (negatif valide qui oriente vers le canal des awards itemises).
- GATE 4 (oracle, generalisation) : reconstruction du portage par film -> par joueur ;
  recouvrement `sum(intervalles de portage) / time_as_skull_carrier_seconds` >= 80 % par
  joueur ET porteur PRINCIPAL correct sur >= 3/5 films du corpus en cache.
- GATE 5 (publication, si 4 tient) : porteur publie dans `objectiveObjects` au patron
  `vip_crown` / `flag_carries` ; triplet schema Go/contrat/web + i18n FR+EN + calque + garde
  de mode Oddball ; re-cuisson temoins avec verification de contenu ; gates go test +
  contracttest + tsc -b + vitest match-replay + lint web + parite schema.

## 8. Outillage

`cmd/oddball-terrain` (un seul binaire, compile une fois) : PARENT (lecture DB
OpenReadForQuery pour xuid->gamertag ; oracle fige ; orchestre les enfants filmproc ;
alignement + confrontation + gates ; ecrit les logs) ; ENFANT (`-child -match <id>` :
sentinelle memoire filmproc 2 Gio armee, decode StatRecords + fil des morts, slot->xuid,
emet le dump tague par slot/round/emplacement ; AUCUNE base). Logs :
`TERRAIN_scores.log`, `TERRAIN_confrontation.log`, `TERRAIN_gate_oracle.log`.
