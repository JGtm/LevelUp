# Mesures — maquette « Séries temporelles › Usages » (2026-10-05)

Base : copie `scratchpad/db/shared_matches_v2.duckdb` (copiée le 2026-10-05 18:26), lecture seule,
outil `diag_q.exe`. Vues `_latest` uniquement (sauf `match_participants` / `match_registry`, qui
ne sont pas append-only). Les requêtes passent par `q.sh`, qui préfixe le CTE `solo` ci-dessous.

```sql
WITH solo AS (
 SELECT r.match_id, r.start_time_utc, r.map_name, r.map_name_fr, r.pair_name, ...,
        p.team_id AS my_team, p.outcome AS my_outcome
 FROM match_registry r JOIN match_participants p ON p.match_id=r.match_id AND p.xuid='2533274823110022'
 WHERE COALESCE(r.is_firefight,false)=false AND r.start_time_utc >= TIMESTAMPTZ '2026-07-01 00:00:00+00'
 AND NOT EXISTS (SELECT 1 FROM match_participants f WHERE f.match_id=r.match_id AND f.xuid IN
   ('2533274833178266','2533274858283686','2535405528935279','2535409018618248','2535413181053876',
    '2535430985184703','2535460062932944','2535469190789936','2535472547643888')))
```

Résultat (convention `outcomes.toml` : 2 = victoire, 3 = défaite, 1 = égalité, 4 = abandon).

## 0. Périmètre

```sql
SELECT count(*), min(start_time_utc), max(start_time_utc), count(DISTINCT my_team),
  sum(match_id IN match_usage_films_latest), sum(match_id IN match_pad_pickups_by_tier_latest) FROM solo
```
| matchs | premier | dernier | camps | filmés (résumé d'usage) | niveaux de socle |
|---|---|---|---|---|---|
| 90 | 2026-07-03 15:02 UTC | 2026-09-22 18:19 UTC | 0 et 1 (aucun FFA) | 19 | 19 |

Par mois : juillet 83 (12 filmés), août 1 (1), septembre 6 (6). Résultats : 40 victoires, 48 défaites, 2 abandons.
Par jour : 03/07 9 · 11/07 2 · 14/07 5 · 15/07 4 · 16/07 11 · 17/07 8 · 18/07 5 · 19/07 5 · 20/07 3 · 21/07 3 · 22/07 5 · 23/07 11 · 28/07 7 · 29/07 5 · 27/08 1 · 22/09 6.

**Mode : les 90 matchs sont TOUS « Super Fiesta:Slayer »** (`regexp_replace(pair_name,' on .*','')`), 0 match à
objectif (`match_objective_stats_latest` : 0 ligne sur le périmètre). Les 19 films : `summary_rev` us6,
`artifact_schema` 71, `duration_ms > 0` sur 19/19.

Liste des 19 matchs filmés (heure de Paris, carte, issue, score camp 0–1) : 28/07 11:43 Forest (abandon, 32–50) ·
11:49 Catalyst (D 50–27, camp 1) · 11:58 Bazaar (abandon) · 19:14 Launch Site (V 50–15) · 20:00 Bazaar (V) ·
20:14 Streets (D) · 20:23 Live Fire (V) · 29/07 21:15 Prism (D) · 21:29 Streets (V) · 21:38 Bazaar (D) ·
21:51 Launch Site (D) · 22:05 Recharge (D) · 27/08 21:02 Behemoth (D) · 22/09 19:27 Cliffhanger (V) ·
19:36 Recharge (D) · 19:46 Catalyst (D) · 19:57 Streets (V) · 20:06 Streets (V) · 20:19 Recharge (V).
Liste des 90 matchs (date, heure, carte, issue, score de mon camp, score adverse, filmé) : intégrée telle
quelle dans la maquette (`MATCHES`).

## 1. Contrôle des ressources (bonus, armes spéciales, râteliers, véhicules)

Bonus — clés `powerup_camo` / `powerup_overshield` :
```sql
SELECT k, sum(json_extract(j,'$."'||k||'"')::INT) FROM (SELECT u.taken_json j, unnest(json_keys(u.taken_json)) k
 FROM match_usage_players_latest u JOIN solo USING(match_id)) GROUP BY k
```
| colonne | clés présentes sur le solo |
|---|---|
| taken_json | sensor 61, wall 25 — **aucune clé powerup_*** |
| spent_json | wall 103, sensor 36 |
| kept_json | wall 1, sensor 1 |
| dropped_json | grapple 457, thruster 408, sensor 371, wall 191 — **aucune clé powerup_*** |
| deployed_json | wall 223 |
| pad_pickups_json | vide |
| `match_usage_films_latest.powerup_pickups_json` | `{}` sur les 19 films (0 socle de bonus vidé) |
| `pad_occupancies`, `weapon_pads_json` | 0 et `[]` |

→ **Bonus : 0 prise, 0 socle vidé, dans les deux camps.**

Armes spéciales / râteliers :
```sql
SELECT tier, count(*), sum(pickups), max(pads_total), max(pads_confirmed)
FROM match_pad_pickups_by_tier_latest JOIN solo USING(match_id) GROUP BY 1
```
| tier | lignes | prises | socles vus | socles confirmés |
|---|---|---|---|---|
| aucune_prise | 164 | 0 | 0 | 0 |

→ **0 prise de puissance, 0 prise de terrain : aucun socle d'arme sur les cartes Super Fiesta.**

Véhicules : `SELECT count(*) FROM match_vehicle_takes_latest JOIN solo USING(match_id)` → **0 ligne**
(ni `take`, ni ligne `match` de couverture) : véhicules **non mesurés** sur les 90 matchs.
`match_kill_events_latest.source_category` ne porte pas de classe véhicule (None 6060, Headshot 720,
AttachedDamage 604, HeadshotMultiplier 411, SilentMelee 113, VehicleTransferDamage 31, ChainedProjectile 29,
NULL 20, CollisionDamage 7).

## 2. Au fil des matchs

Aucune prise de ressource sur les 90 matchs → aucune courbe. Résultats par match : liste `MATCHES` (90).
Drapeaux de dominance : calculés par l'application depuis la courbe de score (`analysis/comeback.go`),
non stockés — **non relevés** pour la maquette.

## 3. Carte par carte — frags aux armes spéciales (feuille de match, 90 matchs)

```sql
SELECT s.map_name, count(DISTINCT s.match_id) n, count(DISTINCT CASE WHEN filmé THEN s.match_id END) filmed,
 sum(CASE WHEN p.team_id=s.my_team THEN p.power_weapon_kills ELSE 0 END) us,
 sum(CASE WHEN p.team_id=s.my_team THEN 0 ELSE p.power_weapon_kills END) them, victoires
FROM solo s JOIN match_participants p USING(match_id) GROUP BY 1 ORDER BY 2 DESC, 1
```
| carte | matchs | filmés | mon camp | adversaire | victoires |
|---|---|---|---|---|---|
| Bazaar | 10 | 3 | 202 | 227 | 4 |
| Streets | 10 | 4 | 194 | 196 | 6 |
| Aquarius | 9 | 0 | 176 | 234 | 3 |
| Recharge | 9 | 3 | 163 | 202 | 2 |
| Launch Site | 8 | 2 | 134 | 136 | 4 |
| Catalyst | 6 | 2 | 87 | 133 | 3 |
| Cliffhanger | 6 | 1 | 102 | 127 | 4 |
| Forest | 6 | 1 | 90 | 112 | 4 |
| Live Fire | 6 | 1 | 125 | 104 | 4 |
| Prism | 6 | 1 | 99 | 130 | 1 |
| Chasm | 5 | 0 | 89 | 81 | 2 |
| Behemoth | 3 | 1 | 53 | 46 | 0 |
| Forbidden | 3 | 0 | 66 | 60 | 2 |
| Illusion | 3 | 0 | 45 | 74 | 1 |

`map_name_fr` est NULL sur ces cartes : noms d'origine.

## 4. Mes prises dans mon camp

0 bonus, 0 arme spéciale, 0 râtelier pris ; véhicules non mesurés → aucune barre. « Bonus perdus » sans objet.

## 5-6. Frags obtenus avec les ressources, rendement

```sql
SELECT CASE WHEN u.xuid='2533274823110022' THEN 'moi' WHEN p.team_id=s.my_team THEN 'reste_camp' ELSE 'adversaire' END side,
 count(*), sum(camo_episodes), sum(camo_ms), sum(camo_kills), sum(overshield_episodes), sum(overshield_ms),
 sum(overshield_kills), sum(grapple_pulls), sum(deployed_json.wall), sum(dropped_objects), sum(pad_pickups)
FROM match_usage_players_latest u JOIN solo s USING(match_id) LEFT JOIN match_participants p ON ... GROUP BY 1
```
| côté | lignes | épisodes camo | camo_ms | frags camo | épisodes surbouclier | ms | frags | grappin | murs posés | objets lâchés | socles |
|---|---|---|---|---|---|---|---|---|---|---|---|
| moi | 19 | 98 | 1 447 800 | 56 | 0 | 0 | 0 | 43 | 31 | 132 | 0 |
| reste du camp | 61 | 287 | 1 768 800 | 102 | 0 | 0 | 0 | 224 | 90 | 549 | 0 |
| adversaire | 83 | 347 | 1 993 300 | 107 | 0 | 0 | 0 | 259 | 102 | 746 | 0 |

Aucun participant sans camp (0 ligne). Mon camp = moi + reste : **158 frags pendant l'effet, 3 216 600 ms
(53 min 37)** contre **107 frags, 1 993 300 ms (33 min 13)**. Rendement : 2,95 contre 3,22 frags par
minute d'effet, écart −8,5 %. Les épisodes de camouflage existent SANS aucune prise ni socle vidé :
c'est l'équipement de réapparition (référence équipement §2 : « hors de toute fenêtre de taken …
l'équipement de RÉAPPARITION »).

Frags aux armes spéciales (feuille de match, 90 matchs) :
```sql
SELECT sum(us), sum(them), sum(moi) ... FROM match_participants p JOIN solo s USING(match_id)
```
| mon camp | adversaire | moi | frags totaux mon camp | adversaire | sans camp |
|---|---|---|---|---|---|
| 1 625 | 1 862 | 255 | 4 023 | 4 030 | 0 |

Pas d'exposition pour les armes spéciales (0 prise) : pas de barre fine, pas de rendement.

## 7. Groupés ou isolés

`match_life_placement_latest` : **0 ligne dans toute la base** (table `match_life_placement` vide,
`estimated_size 0`), pas seulement sur le solo. Les vies de JGtm du 22/09 demandées en illustration
n'existent pas non plus dans la copie → carte dessinée sans aucun point.

## 8. Objectif

`match_objective_stats_latest` : 0 ligne sur les 90 matchs (Super Fiesta Assassin, aucun objectif).

## 9. Équipement pris, et ce que j'en ai fait

```sql
... fam AS (SELECT unnest(['wall','sensor','translocator_beacon','shroud_screen','threat_seeker','repair_field','grapple','thruster']) f)
SELECT f, side, sum(taken), sum(spent), sum(deployed), sum(kept), sum(dropped) FROM rows, fam GROUP BY 1,2
```
| famille | côté | pris | consommés | posés | gardés | lâchés |
|---|---|---|---|---|---|---|
| mur | moi | 2 | 10 | 31 | 0 | 15 |
| mur | reste du camp | 9 | 37 | 90 | 1 | 74 |
| mur | adversaire | 14 | 56 | 102 | 0 | 102 |
| capteur | moi | 7 | 6 | 0 | 0 | 38 |
| capteur | reste du camp | 26 | 15 | 0 | 1 | 135 |
| capteur | adversaire | 28 | 15 | 0 | 0 | 198 |
| translocateur, écran, traqueur, champ de réparation | tous | 0 | 0 | 0 | 0 | 0 |
| grappin | moi / reste / adversaire | 0 | 0 | 0 | 0 | 39 / 186 / 232 |
| propulseur | moi / reste / adversaire | 0 | 0 | 0 | 0 | 40 / 154 / 214 |

« Utilisé » = posés pour le mur, consommés sinon (`sessionusage.equipmentUsedOf`). Moi : mur 31 · 0 · 15
(46 objets, dont 2 pris sur la carte) ; capteur 6 · 0 · 38 (44, dont 7 pris). Reste du camp : mur
90 · 1 · 74 ; capteur 15 · 1 · 135. Sur l'ensemble de la base, `taken_json` / `spent_json` /
`kept_json` ne portent JAMAIS `grapple` ni `thruster` (périmètre du bilan, `equipmentOutcomeFamilies`) :
grappin et propulseur n'ont que des lâchers.

## Avant — cartes 4 à 8 (mêmes mesures)

- Carte 4 « Usages d'équipement » (familles touchées par moi, triées par objets) : Camouflage 98 objets
  (pris 0 ; utilisé 98) ; Mur 46 (pris 2 ; 31 · 0 · 15) ; Capteur 44 (pris 7 ; 6 · 0 · 38). Repères
  « reste de mon équipe » / « eux » (taux d'utilisation) : camo 100 % / 100 % ; mur 54,5 % (90/165) /
  50,0 % (102/204) ; capteur 9,9 % (15/151) / 7,0 % (15/213).
- Carte 5 donut (objets d'équipement, utilisé + gardé + lâché, 8 familles du bilan) : moi 188, amis
  suivis 0, reste de mon équipe 603, adversaires 764, lobby 1 555.
- Cartes 6-7 : 0 prise de socle (pad_pickups 0 partout) → état vide « Aucune prise de socle ».
- Carte 8 : 19 matchs projetés, tous `aucune_prise`, 0 socle → état vide, note « Sur 19 matchs, le mode
  n'allume aucun socle » ; « Mesuré sur 19 matchs sur 90 ».

## Avant — cartes 9 à 11 (formes, par match filmé)

Effectifs : présents à la fin (`present_at_completion`), bots inclus.
| match | carte | issue | équipe | lobby | lignes | camo moi/équipe/lobby | murs | grappin | lâchés |
|---|---|---|---|---|---|---|---|---|---|
| 28/07 11:43 | Forest | 4 | 4 | 8 | 9 | 3/13/31 | 0/4/9 | 1/17/29 | 4/37/64 |
| 28/07 11:49 | Catalyst | 3 | 4 | 8 | 10 | 5/10/52 | 2/3/9 | 4/6/26 | 8/47/70 |
| 28/07 11:58 | Bazaar | 4 | 4 | 8 | 13 | 0/33/40 | 1/10/17 | 0/12/18 | 0/25/66 |
| 28/07 19:14 | Launch Site | 2 | 4 | 8 | 9 | 5/11/15 | 2/2/2 | 1/6/8 | 3/14/51 |
| 28/07 20:00 | Bazaar | 2 | 4 | 6 | 13 | 13/38/57 | 2/8/17 | 0/27/51 | 7/32/67 |
| 28/07 20:14 | Streets | 3 | 4 | 8 | 8 | 6/27/38 | 2/7/7 | 1/14/33 | 10/43/86 |
| 28/07 20:23 | Live Fire | 2 | 4 | 8 | 8 | 3/13/23 | 1/4/12 | 2/9/25 | 8/37/84 |
| 29/07 21:15 | Prism | 3 | 4 | 8 | 8 | 8/24/57 | 5/9/19 | 1/16/32 | 6/40/80 |
| 29/07 21:29 | Streets | 2 | 4 | 8 | 10 | 6/19/26 | 2/7/14 | 3/25/36 | 5/19/56 |
| 29/07 21:38 | Bazaar | 3 | 4 | 8 | 10 | 1/17/27 | 1/8/12 | 8/26/49 | 4/41/80 |
| 29/07 21:51 | Launch Site | 3 | 4 | 8 | 8 | 11/43/77 | 1/9/14 | 8/23/48 | 9/37/73 |
| 29/07 22:05 | Recharge | 3 | 4 | 8 | 12 | 5/14/42 | 2/6/11 | 0/4/6 | 5/45/88 |
| 27/08 21:02 | Behemoth | 3 | 4 | 8 | 8 | 6/36/64 | 1/3/7 | 1/10/18 | 11/44/86 |
| 22/09 19:27 | Cliffhanger | 2 | 4 | 8 | 8 | 2/22/41 | 1/3/10 | 0/16/24 | 10/36/81 |
| 22/09 19:36 | Recharge | 3 | 4 | 8 | 8 | 4/7/35 | 1/6/13 | 5/8/19 | 11/42/78 |
| 22/09 19:46 | Catalyst | 3 | 4 | 8 | 10 | 6/19/38 | 1/3/9 | 3/9/38 | 11/39/75 |
| 22/09 19:57 | Streets | 2 | 4 | 7 | 10 | 5/17/33 | 1/4/9 | 1/7/18 | 8/36/82 |
| 22/09 20:06 | Streets | 2 | 4 | 7 | 10 | 6/18/29 | 3/10/14 | 3/20/29 | 7/37/92 |
| 22/09 20:19 | Recharge | 2 | 4 | 8 | 10 | 3/4/7 | 2/15/18 | 1/12/19 | 5/30/68 |

Surbouclier : 0 épisode dans tout le lobby sur les 19 matchs. Lâchés par famille (moi, par match) : grappin,
propulseur, capteur, mur — totaux 39, 40, 38, 15 (= 132 objets lâchés). Les parts, parités et étendues
des cartes 9 et 11 sont calculées dans la page depuis ce tableau, avec les formules de
`formes/model/aggregates.ts` (parité = 100 / effectif moyen) ; leurs valeurs sont reprises au compte rendu.
Cartes 12-14 : état vide « Aucune prise de socle ». Cartes 15-17 : section masquée par l'app (0 match à objectif).

## Portée (contexte, inchangé dans les deux versions)

`kill_positions_latest` couvre les 90 matchs ; distance 3D tueur–victime ; positions NULL écartées.
Morts : jointure `match_kill_events_latest` (même match, même `time_ms`, `feed_killer_xuid = killer_xuid`,
`victim_xuid` = JGtm). Entame : `kill_openings_latest` (même match, tueur, `time_ms`).
| grandeur | valeur |
|---|---|
| portée médiane de mes frags | 5,4 m (861 frags mesurés sur 945 ; p10 1,6 · p90 13,7) |
| portée médiane de mes morts | 4,6 m (841 morts mesurées sur 907 ; p10 0,6 · p90 13,8) |
| distance d'entame médiane | 7,9 m (849 frags mesurés) |
| entame → frag | −2,0 m ; distance fermée dans 78 % des frags |

Rôles de portée : par match, ma médiane de distance de frag et celle de tous les frags du lobby
(90 points, dont 1 sans frag mesuré) — tableau intégré dans la page (`ROLES`). Hauteur d'engagement :
861 frags (ma hauteur − celle de la victime) et 841 morts (ma hauteur − celle du tueur), intégrés (`ELEV`).
Portée PAR ARME : l'arme de chaque frag vient de la base du joueur (`stats.duckdb`, classifieur de
source) — non copiée ; `match_weapon_hit_distance_latest` : 0 ligne sur le solo → **non reproduit**,
la carte montre la seule ligne « toutes armes » mesurée ci-dessus. Ces valeurs sont des relevés de
maquette (SQL écrit pour l'occasion) et peuvent différer d'un dixième des calculs de l'app.

## ILLUSTRATION — tous les matchs de JGtm depuis le 1er juillet (escouade comprise)

Ajouté le 2026-10-05 à la demande du superviseur. Même copie de base, même outil. Périmètre : matchs de JGtm, non Firefight, `start_time_utc >= 2026-07-01`, TOUS contextes. Mon camp = `team_id` de JGtm dans chaque match ; participant sans camp = adversaire.

Extraction : `extract.sh illu` (le CTE du solo sans le filtre « aucun ami »), extraits bruts dans `ex/illu_*.tsv` : matchs, participants, résumés d'usage par joueur, films, niveaux de socle, objectifs, véhicules, positions de frag. Calcul : `compute.js` (règles de `squademprise`, `sessionusage`, `formes/model`) vers `vm.json` et `vm_report.json`. Le même calcul repasse le solo (`extract.sh solo`) et retrouve tous les chiffres des sections précédentes.

### 0. Périmètre

| matchs | filmés | niveaux mesurés sur un match filmé | avec prises puissance/terrain | à objectif | véhicules | avec un ami suivi | V / D / autres |
|---|---|---|---|---|---|---|---|
| 189 | 83 (mois 07 : 44, mois 08 : 8, mois 09 : 31) | 77 | 54 | 29 | 0 ligne | 99 | 90 / 94 / 5 |

Modes : Super Fiesta:Slayer 98 · Community:Team Slayer 29 · Arena:CTF 16 · Arena:Team Slayer 15 · Arena:Slayer 9 · Community:Slayer 9 · Arena:Strongholds 7 · Arena:Neutral Flag CTF 3 · BTB:CTF 2 · BTB:Total Control 1.

Niveaux de socle (toutes lignes) : terrain 857 prises sur 54 matchs, puissance 460 sur 52, base 21 sur 7, non classé 54 sur 6, aucune_prise sur 70 matchs.

### 1. Contrôle des ressources (matchs filmés ; niveaux mesurés pour les armes)

| ressource | mon camp | adversaire | dont moi | perdus mon camp | perdus adversaire |
|---|---|---|---|---|---|
| Bonus | 111 | 87 | 30 | 13 | 12 |
| Armes spéciales (puissance) | 229 | 231 | 58 | — | — |
| Armes de râtelier (terrain) | 350 | 507 | 51 | — | — |

Socles de bonus vidés : camouflage 227, surbouclier 193, pour 198 prises attribuées. Véhicules : 0 ligne.

### 2. Au fil des matchs

189 points, un par match dans l'ordre. Cumul final : bonus 56,1 %, armes spéciales 49,8 %. Les matchs sans film n'ajoutent rien. Série complète : `vm.json`, clé `illu.fil`.

### 3. Carte par carte

48 cartes. Matchs / filmés / niveaux mesurés · frags aux armes spéciales mon camp–adversaire (feuille) · résultats :

- Recharge : 13 / 5 / 5 · 202–243 · 5 V, 8 D
- Aquarius : 11 / 2 / 2 · 188–251 · 4 V, 7 D
- Bazaar : 11 / 3 / 3 · 202–238 · 4 V, 6 D, 1 A
- Streets : 11 / 4 / 4 · 196–204 · 6 V, 5 D
- Catalyst : 9 / 3 / 3 · 129–173 · 3 V, 6 D
- Launch Site : 9 / 3 / 3 · 155–154 · 5 V, 4 D
- Chasm : 8 / 2 / 1 · 130–132 · 2 V, 6 D
- Forest : 8 / 3 / 3 · 106–126 · 6 V, 1 D, 1 A
- Cliffhanger : 7 / 2 / 2 · 112–152 · 4 V, 3 D
- Live Fire : 7 / 2 / 2 · 135–124 · 4 V, 3 D
- Prism : 7 / 2 / 2 · 118–157 · 2 V, 5 D
- Behemoth : 6 / 4 / 3 · 73–67 · 1 V, 5 D
- Illusion : 6 / 3 / 3 · 53–96 · 2 V, 4 D
- Banished Narrows : 4 / 3 / 3 · 27–19 · 3 V, 1 D
- Detachment : 4 / 0 / 0 · 33–54 · 1 V, 3 D
- Forbidden : 4 / 0 / 0 · 74–69 · 2 V, 2 D
- Origin : 4 / 3 / 3 · 30–16 · 2 V, 2 D
- Perilous : 4 / 3 / 2 · 21–24 · 3 V, 1 D
- Shogun : 4 / 3 / 2 · 29–22 · 1 V, 3 D
- Takamanohara : 4 / 2 / 2 · 40–28 · 2 V, 2 D
- Argyle : 3 / 0 / 0 · 18–18 · 1 V, 1 D, 1 A
- Critical Dewpoint : 3 / 2 / 2 · 21–23 · 2 V, 1 D
- Nemesis : 3 / 1 / 1 · 9–5 · 3 V, 0 D
- Shiro : 3 / 0 / 0 · 39–38 · 1 V, 2 D
- Curfew : 2 / 2 / 2 · 22–9 · 2 V, 0 D
- Domicile : 2 / 2 / 2 · 9–29 · 0 V, 2 D
- Empyrean : 2 / 1 / 1 · 8–21 · 1 V, 1 D
- Fortress : 2 / 2 / 2 · 11–8 · 1 V, 1 D
- High Ground : 2 / 2 / 1 · 18–16 · 2 V, 0 D
- Houseki : 2 / 0 / 0 · 5–2 · 1 V, 1 D
- Isolation : 2 / 2 / 2 · 4–18 · 0 V, 2 D
- Kiken'na : 2 / 1 / 1 · 6–11 · 1 V, 0 D, 1 A
- Opulence : 2 / 1 / 1 · 2–3 · 0 V, 2 D
- Snowbound : 2 / 2 / 2 · 0–0 · 1 V, 1 D
- Solution : 2 / 2 / 2 · 5–0 · 2 V, 0 D
- Starboard : 2 / 2 / 2 · 2–8 · 2 V, 0 D
- Absolution : 1 / 1 / 1 · 0–12 · 0 V, 1 D
- Cliffside : 1 / 0 / 0 · 8–6 · 1 V, 0 D
- Dynasty : 1 / 1 / 1 · 9–2 · 1 V, 0 D
- Flood Gulch : 1 / 1 / 1 · 23–38 · 0 V, 0 D, 1 A
- Fortitude : 1 / 1 / 1 · 9–13 · 1 V, 0 D
- Goliath : 1 / 1 / 1 · 8–2 · 1 V, 0 D
- Insolence : 1 / 1 / 1 · 27–31 · 1 V, 0 D
- Kaiketsu : 1 / 0 / 0 · 0–0 · 0 V, 1 D
- Smallhalla : 1 / 0 / 0 · 8–5 · 1 V, 0 D
- Sylvanus : 1 / 1 / 1 · 11–1 · 1 V, 0 D
- The Pit : 1 / 1 / 1 · 15–13 · 1 V, 0 D
- Vagabond : 1 / 1 / 0 · 4–1 · 0 V, 1 D

### 4. Objets (mon camp–adversaire, dont moi)

- Bonus : Camouflage 60–50 (moi 16) · Surbouclier 51–37 (moi 14)
- Armes spéciales : S7 Sniper 107–91 (moi 13) · M41 SPNKr 58–61 (moi 25) · Mutilateur 25–12 (moi 11) · Empaleur 9–16 (moi 0) · SPNKr à combustible 11–12 (moi 4) · Fusil électrique 7–14 (moi 2) · Épée à énergie 5–12 (moi 2) · Crémateur 3–8 (moi 0) · Marteau antigravité 2–2 (moi 1) · CQS48 Bulldog 1–1 (moi 0) · Ravageur 1–1 (moi 0) · Calcineur 0–1 (moi 0)
- Armes de râtelier : BR75 46–78 (moi 9) · Carabine Vestige 41–65 (moi 7) · VK78 Commando 28–56 (moi 5) · Déchiqueteur 41–37 (moi 1) · Disrupteur 26–29 (moi 4) · Bandit EVO 20–31 (moi 4) · Rayon de Sentinelle 23–24 (moi 2) · CQS48 Bulldog 14–30 (moi 5) · Carabine à impulsion 11–30 (moi 0) · Needler 21–19 (moi 4) · Calcineur 4–26 (moi 0) · MA5K Avenger 17–13 (moi 3) · Fusil électrique 11–16 (moi 0) · Hydra 14–12 (moi 3) · Ravageur 8–18 (moi 4) · MK50 Sidekick 13–12 (moi 0) · Pistolet à plasma 8–8 (moi 0) · Fusil traqueur 4–3 (moi 0)

### 5-6. Frags obtenus, rendement

| | mon camp | adversaire |
|---|---|---|
| frags pendant l'effet (bonus) | 315 | 255 |
| temps d'effet (ms) | 7146000 | 5819100 |
| frags aux armes spéciales, 189 matchs (feuille) | 2354 | 2682 |
| frags aux armes spéciales, matchs filmés à niveaux mesurés | 730 | 837 |
| prises d'armes spéciales | 229 | 231 |

Rendements : bonus 2,64 contre 2,63 frags par minute d'effet ; armes spéciales 3,19 contre 3,62 frags par prise.

### 7. Placement des vies

`match_life_placement` : 0 ligne dans toute la base (les deux périmètres).

### 8. Objectif

- Drapeau (21 matchs) : take [Drapeaux capturés 38–43 (moi 9), Aides à la capture 35–41 (moi 6), Drapeaux volés 131–129 (moi 15), Rapatrieurs abattus 20–35 (moi 3)] ; defend [Retours 92–93 (moi 26), Drapeaux sécurisés 191–226 (moi 50), Porteurs abattus 77–94 (moi 16)] ; hold [Temps de portage 2 317–2 029,4 (moi 314,2)]
- Bases (8 matchs) : take [Zones capturées 197–183 (moi 46), Frags offensifs de zone 70–96 (moi 11)] ; defend [Zones sécurisées 37–59 (moi 15), Frags défensifs de zone 50–88 (moi 15)] ; hold [Temps en zone 2 807,8–2 541,7 (moi 665,3)]

Totaux par rôle (Prendre / Défendre / Tenir en secondes) — moi : 90 / 122 / 979,5 ; reste du camp : 401 / 325 / 4 145,3. Rôle dominant : moi defend, reste du camp take. « Bases » réunit les 7 Strongholds et le Total Control (règle Go : zone_scoring_ticks nul, donc Strongholds). Prises nettes de drapeau (film) non intégrées.

### 9. Équipement (servi · gardé · lâché)

- Grappin : non mesuré, 84 lâchés
- Mur de protection : moi 52 · 0 · 32 (pris sur la carte 23), reste du camp 146 · 7 · 151
- Capteur de menaces : moi 6 · 1 · 54 (pris sur la carte 12), reste du camp 16 · 4 · 183
- Translocateur : moi 1 · 2 · 1 (pris sur la carte 4), reste du camp 2 · 7 · 2
- Écran occultant : moi 1 · 1 · 5 (pris sur la carte 5), reste du camp 6 · 6 · 38
- Traqueur de menaces : moi 0 · 0 · 3 (pris sur la carte 3), reste du camp 0 · 2 · 6
- Champ de réparation : moi 0 · 0 · 0 (pris sur la carte 0), reste du camp 1 · 4 · 8
- Propulseur : non mesuré, 65 lâchés

### Avant (cartes 4 à 17)

- Carte 4 : Camouflage pris 16 (utilisé 158 · gardé 0 · lâché 1 ; reste 98,3 % · eux 99,1 %) ; Mur de protection pris 23 (utilisé 52 · gardé 0 · lâché 32 ; reste 48 % · eux 52,5 %) ; Capteur de menaces pris 12 (utilisé 6 · gardé 1 · lâché 54 ; reste 7,9 % · eux 9 %) ; Surbouclier pris 14 (utilisé 13 · gardé 0 · lâché 1 ; reste 93,9 % · eux 86,4 %) ; Écran occultant pris 5 (utilisé 1 · gardé 1 · lâché 5 ; reste 12 % · eux 9,5 %) ; Translocateur pris 4 (utilisé 1 · gardé 2 · lâché 1 ; reste 18,2 % · eux 34,8 %) ; Traqueur de menaces pris 3 (utilisé 0 · gardé 0 · lâché 3 ; reste 0 % · eux 33,3 %)
- Carte 5 (objets) : moi 332, amis suivis 235, reste de mon équipe 865, adversaires 1512.
- Carte 6 : moi 141 prises de socle (résumé d'usage). Carte 7 : moi 141, amis 243, reste 312, adversaires 829.
- Carte 8 (mes prises par niveau) : puissance 58, terrain 51, base 7 ; 6 non classées ; 77 matchs mesurés dont 54 avec socles.
- Cartes 9 et 13 (part équipe / parité · part lobby / parité · au-dessus) : camo 25,8 / 23,3 · 12,3 / 11,7 · 22/51 ; wall 26,3 / 23,3 · 12,6 / 11,7 · 16/44 ; overshield 22 / 23,3 · 13,4 / 11,7 · 10/23 ; grapple 17,3 / 23,3 · 8,1 / 11,7 · 15/50 ; dropped 20,9 / 23,3 · 10,4 / 11,7 · 36/83 ; pads 20,3 / 23,3 · 9,2 / 11,7 · 20/58
- Carte 11 (min, max, moyenne par match, 83 matchs) : camo 0–13 moy 1,9 ; wall 0–5 moy 0,63 ; overshield 0–2 moy 0,16 ; grapple 0–8 moy 0,95 ; dropped 0–11 moy 3,9
- Carte 12 (part du lobby, parité 11,7 %) : toutes 9,2 %, heavy 10,6 % (67/634), precision 8,8 % (28/320), other 8,1 % (46/571)
- Carte 14 : S7 Sniper 3,6 % (16/442, 43 matchs) ; BR75 4,9 % (15/306, 24 matchs) ; M41 SPNKr 11,8 % (27/229, 28 matchs) ; Carabine Vestige 5,4 % (8/148, 26 matchs) ; Déchiqueteur 0,7 % (1/135, 27 matchs) ; VK78 Commando 3,4 % (5/147, 32 matchs) ; Bandit EVO 6 % (5/84, 15 matchs) ; Disrupteur 5,4 % (4/74, 19 matchs) ; Carabine à impulsion 0 % (0/82, 20 matchs) ; Fusil électrique 3 % (2/67, 10 matchs) ; CQS48 Bulldog 6,8 % (5/73, 12 matchs) ; Rayon de Sentinelle 2,9 % (2/68, 19 matchs)
- Carte 15 : take équipe 18,3 % (parité 20,7) · lobby 8,8 % (parité 10,4) ; defend équipe 27,3 % (parité 20,7) · lobby 12,1 % (parité 10,4) ; hold équipe 19,1 % (parité 20,7) · lobby 10,1 % (parité 10,4)
- Carte 16, Drapeau : Drapeaux capturés 23,7 / 11,1 % ; Aides à la capture 17,1 / 7,9 % ; Drapeaux volés 11,5 / 5,8 % ; Rapatrieurs abattus 15 / 5,5 % ; Retours 28,3 / 14,1 % ; Drapeaux sécurisés 26,2 / 12 % ; Porteurs abattus 20,8 / 9,4 % ; Temps de portage 13,6 / 7,2 %
- Carte 16, Bases : Zones capturées 23,4 / 12,1 % ; Frags offensifs de zone 15,7 / 6,6 % ; Zones sécurisées 40,5 / 15,6 % ; Frags défensifs de zone 30 / 10,9 % ; Temps en zone 23,7 / 12,4 %

### Portée (illustration)

Frags 4,9 m médiane (1672 mesurés sur 1881, p10 0,4 · p90 12,1) ; morts 4,9 m (1793 sur 2030) ; entame 7,1 m (1650) ; entame → frag -2 m, distance fermée dans 79 % des frags.
