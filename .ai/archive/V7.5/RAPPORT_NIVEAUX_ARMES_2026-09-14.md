# RAPPORT — Niveaux d'armes, étape 0 : couverture de la jointure emplacements × socles (2026-09-14)

> Étape 0 du plan `.ai/PLAN_NIVEAUX_ARMES_2026-09-13.md`. Diagnostic seul, aucun code de
> production. Branche `feat/niveaux-armes`, worktree `LevelUp-wt-niveaux-armes`.
>
> **VERDICT : COUVERTURE SUFFISANTE.** « Non classé » pèse **2,95 % des prises de socle**
> (85 sur 2 881), sous le seuil de 5 % fixé par le plan. **Aucune carte du parc local n'est
> absente de la référence** (76 cartes jouées, 76 au catalogue). Les étapes 1 à 4 peuvent
> s'ouvrir sans compléter la référence au préalable.

## 1. Méthode

Test de recherche jetable `internal/research/nivarmes/niveaux_armes_research_test.go`
(supprimé après la mesure — sortie brute intégrale en annexe §6).

Chaîne mesurée, la même que celle du service en production
(`internal/service/replay_map_weapon_pads.go`) :

1. Corpus d'artefacts : `data/cache/replays/halo_infinite/*.json` du dépôt principal,
   `*.derived.json` exclus. **76 artefacts**, schéma 54.
2. `map_id` du match : lecture **seule** de `match_registry` (`map_name`, `map_id`,
   `pair_name`) dans `shared_matches_v2.duckdb` via `duckdb.OpenReadForQuery` — jamais
   d'ouverture RW, le serveur local tenant la base. **76 / 76 matchs résolus.**
3. Référence des emplacements : `replay.LoadMapWeaponPadsMerged` sur
   `data/titles/halo_infinite/reference/map_weapon_pads.json` (+ overlay `generated/`,
   absent ici). Schéma 1, **76 cartes, 1 549 emplacements** (1 217 `rack`, 230 `power`,
   102 `powerup`).
4. Jointure emplacement × socle : **copie verbatim** de `replay.confirmePar` (glouton, le
   plus proche l'emporte et est ensuite pris, rayon `MapWeaponPadMatchM` = 1 m) — donc
   exactement le résultat que `BuildMapWeaponPads` produit aujourd'hui, la famille
   `MapWeaponPadSpot.Family` conservée en plus.
5. Pondération par les **prises** : `padPickups[].pad` compté par socle. C'est le
   dénominateur du critère de succès du plan (« Non classé ≤ 5 % des prises »).
6. Rôle de l'arme : `weapons.FilmshellWeaponKeysByFamily()` sur le high-32 hexadécimal de
   `WeaponPad.Weapon`, puis `weapons.RolesByKey()`. Aucune lecture de `metadata.duckdb`.
7. Armes de départ : canal `loadouts` (`Loadout.W`), **première émission de chaque `slot`**
   du match, ventilées par `halo.InferModeCategoryFromPairName(pair_name)`.

## 2. Item 0.1 — Couverture de la jointure

### Totaux

| Grandeur | Total | rack | power | powerup | non confirmé |
|---|---|---|---|---|---|
| **Socles du film** | 669 | 466 (69,66 %) | 144 (21,52 %) | 48 (7,17 %) | **11 (1,64 %)** |
| **Prises de socle** | 2 881 | 1 480 (51,37 %) | 935 (32,45 %) | 381 (13,22 %) | **85 (2,95 %)** |

- **0 match sans `map_id`.**
- **17 matchs sur 76 ne publient AUCUN socle** (`weaponPads` absent). Neuf d'entre eux sont
  sur quatre cartes qui n'en publient jamais dans le parc — Bazaar (×3), Launch Site (×3),
  Prism (×2), Cliffhanger (×1) ; les huit autres sont sur des cartes qui en publient dans
  d'autres matchs. C'est le cas documenté en tête de `map_weapon_pads.go` — *le fichier de
  carte pose, le mode allume*. Sur ces matchs il n'y a ni terrain ni puissance à afficher :
  ce n'est **pas** du « Non classé », c'est une absence de mesure, et le bloc doit le dire
  comme tel.
- Les 11 socles non confirmés se concentrent sur **6 cartes** : Flood Gulch (3), Domicile
  (2), Perilous (2), Curfew (2), Solution (1), Dynasty (1).
- **Flood Gulch porte à elle seule 52 des 85 prises non classées (61 %)** sur un unique
  match à 297 prises. Hors Flood Gulch, « Non classé » tombe à **33 prises sur 2 584 =
  1,28 %**. C'est le seul point qui mériterait une vérification de la référence, et il
  n'est pas bloquant.

Le détail carte par carte (41 cartes) est en annexe §6, section `== 0.1 PAR CARTE ==`.

## 3. Item 0.2 — Contrôle croisé rôle × famille d'emplacement

Tableau socle par socle (rôle du registre canonique × famille de l'emplacement confirmé) :

| rôle (registre) | rack | power | powerup | non confirmé |
|---|---|---|---|---|
| (bonus) | 1 | 0 | 48 | 0 |
| automatic | 122 | 2 | 0 | 2 |
| power | 17 | 54 | 0 | 7 |
| precision | 143 | 1 | 0 | 1 |
| shotgun | 34 | 13 | 0 | 0 |
| sidearm | 96 | 0 | 0 | 0 |
| sniper | 9 | 74 | 0 | 1 |
| special | 44 | 0 | 0 | 0 |

**Lecture, et elle confirme la décision D1 du plan.** Le rôle du registre n'est pas un
proxy de la nature de l'emplacement, et ne doit jamais le devenir :

- Les 70 cas « rôle lourd sur râtelier » sont presque tous des armes que le jeu POSE bel et
  bien sur râtelier : **Hydra** (17 cas, rôle `power`), **Needler** et **Sentinel Beam**
  (44 cas, rôle `special`), **Shock Rifle** (9 cas, rôle `sniper`). Ce n'est pas une erreur
  de jointure, c'est la carte qui décide — exactement ce que D1 énonce.
- Le sens inverse, bien plus rare, est le seul signal à surveiller : **3 socles** portent
  une arme de rôle léger sur un socle de puissance — Ravager (`automatic`) sur Fortitude
  (2 socles, même match `879a4dba`) et Stalker Rifle (`precision`) sur Smallhalla
  (1 socle). Soit **0,45 % des 669 socles**.
- Un seul socle de bonus tombe sur un emplacement `rack` (1 sur 49) : la séparation
  arme / bonus est nette.

**Recommandation pour l'étape 1** : garder le contrôle croisé comme garde-rail journalisé
(`slog.WarnContext`) au seul sens « rôle léger sur socle de puissance », et **ne pas**
alerter sur « rôle lourd sur râtelier » — il est nominal à 10 % des socles et noierait le
signal. Seuil proposé, mesuré ici : au-delà de **2 % des socles d'un match** en « rôle
léger sur puissance », la jointure est suspecte.

Les 73 incohérences sont nommées en annexe §6 (69 listées, le relevé plafonnant à 40 cas par
catégorie).

## 4. Item 0.3 — Cartes absentes de la référence

**Aucune.** Les 76 cartes du parc local sont au catalogue versionné (76 entrées, overlay
`reference/generated/map_weapon_pads.json` absent, donc jamais sollicité).

Rien à décider côté utilisateur : la référence n'a pas à être complétée avant de livrer.

Réserve documentaire, sans effet sur le verdict : sur plusieurs cartes le film voit **moins**
de socles que le fichier n'en pose (High Ground 15 vus / 33 au fichier, The Pit 20 / 31,
Smallhalla 14 / 27, Insolence 40 / 45). C'est le comportement attendu et déjà documenté —
le mode n'allume qu'une partie des emplacements.

## 5. Item 0.4 (bonus) — Les armes de DÉPART se lisent bien du film

Première émission `loadouts` de chaque `slot` (un `slot` = une vie : 100 slots distincts
pour 113 pistes sur le match témoin `01e1f945`), par catégorie de mode.

| Catégorie | Vies mesurées | Concentration des 3 premières clés |
|---|---|---|
| Assassin (classé / partie rapide) | 4 406 | **94,4 %** — MA40 AR 45,5 %, Sidekick 29,5 %, BR75 19,4 % |
| BTB | 612 | **92,6 %** — Bandit 47,6 %, MA40 AR 45,1 %, BR75 1,9 % |
| Super Fiesta | 1 309 | **19,7 %** — 22 clés entre 3,2 % et 7,3 % |

**Conclusion : D2 tient.** « Base » se mesure par match depuis le film, sans liste figée.
La signature d'un mode à départs aléatoires est franche et lisible **sans regarder le nom
du mode** : distribution quasi plate sur 22 clés contre trois clés à 94 %. Fiesta et Husky
Raid ne sont pas représentés dans le parc local (seules Assassin, BTB et Super Fiesta
apparaissent) — le comportement y est présumé identique à Super Fiesta, non mesuré ici.

**Piège pour l'étape 1, mesuré.** Prendre **toutes** les émissions `loadouts` au lieu de la
première par `slot` dilue le signal : en Assassin, les trois clés de base passent de 94,4 %
à 85,6 %, et le S7 Sniper monte de 0,73 % à 3,15 %. Le canal ré-émet en cours de vie après
un changement d'arme. `WeaponTierOf` doit donc lire **la première émission de chaque slot**,
jamais l'ensemble du canal.

Réserve : même sur la première émission, une queue de 5,6 % (Assassin) porte des armes qui
ne sont pas des armes de départ — des vies dont la première émission publiée arrive déjà
après un ramassage. Un seuil de publication (par exemple : ne retenir comme « base » qu'une
clé présente sur au moins 5 % des vies du match) évite de promouvoir un Skewer au rang
d'arme de base. À trancher à l'étape 1.2.

## 6. Annexe — Sortie brute intégrale

```
ARTEFACTS: 76
CATALOGUE: 76 cartes
META RESOLUE: 76 / 76 matchs
MATCHS SANS META: 0 ; MATCHS SANS weaponPads: 17
== 0.1 PAR CARTE ==
carte | map_id | matchs | socles_film | catalogue | rack | power | powerup | non_confirme | prises_tot | pr_rack | pr_power | pr_powerup | pr_non_classe
Banished Narrows | 9ad226d8-8947-4c5b-95bc-d220187698c1 | 4 | 46 | AU CATALOGUE(18) | 27 | 11 | 8 | 0 | 193 | 73 | 62 | 58 | 0
Live Fire | 6c01f693-c968-4a71-b157-efc35ffcf71f | 4 | 16 | AU CATALOGUE(10) | 13 | 2 | 1 | 0 | 73 | 47 | 13 | 13 | 0
Streets | 9c7b0b0f-e933-4c2d-9d4a-3e4500d0de99 | 3 | 6 | AU CATALOGUE(9) | 5 | 1 | 0 | 0 | 18 | 13 | 5 | 0 | 0
Isolation | 01af558d-53ab-4f05-ba68-92d805fc6260 | 3 | 49 | AU CATALOGUE(29) | 40 | 9 | 0 | 0 | 157 | 105 | 52 | 0 | 0
Launch Site | 5646ce03-a86e-40d7-9cac-08d442d32606 | 3 | 0 | AU CATALOGUE(19) | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0
Illusion | 9e821f5e-042f-407c-97f3-de165b1cdb26 | 3 | 23 | AU CATALOGUE(17) | 15 | 8 | 0 | 0 | 65 | 25 | 40 | 0 | 0
Forest | e8d56863-9ad4-4efe-9059-81270884589c | 3 | 30 | AU CATALOGUE(19) | 25 | 4 | 1 | 0 | 129 | 84 | 34 | 11 | 0
Recharge | 2b6d2baf-7645-4e16-8a80-c7006f595812 | 3 | 14 | AU CATALOGUE(9) | 10 | 2 | 2 | 0 | 76 | 46 | 10 | 20 | 0
Behemoth | e9a5a982-6c4e-4db6-9383-7b03671460eb | 3 | 30 | AU CATALOGUE(21) | 25 | 4 | 1 | 0 | 112 | 75 | 27 | 10 | 0
Catalyst | f7e8cde9-0c0a-487c-94a3-61bfa0f20465 | 3 | 21 | AU CATALOGUE(11) | 12 | 8 | 1 | 0 | 99 | 29 | 61 | 9 | 0
Bazaar | 3e1e4cec-4f2c-44c6-b8d2-96b85c66c702 | 3 | 0 | AU CATALOGUE(13) | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0
Critical Dewpoint | bae4df14-4f4a-424c-aac1-2f795c807146 | 2 | 18 | AU CATALOGUE(13) | 12 | 6 | 0 | 0 | 79 | 37 | 42 | 0 | 0
Takamanohara | edcd4467-6846-455f-ac44-f1034476f774 | 2 | 30 | AU CATALOGUE(17) | 23 | 6 | 1 | 0 | 146 | 91 | 48 | 7 | 0
Prism | 2fdb8370-e5ac-4a1a-bdce-a08bc738b9ad | 2 | 0 | AU CATALOGUE(13) | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0
Shogun | 33075df7-01c8-40e1-8b3e-1baee0054c76 | 2 | 15 | AU CATALOGUE(13) | 10 | 4 | 1 | 0 | 60 | 32 | 23 | 5 | 0
Aquarius | c395f3ac-4614-45f9-a83a-56f69e8ae962 | 2 | 12 | AU CATALOGUE(9) | 9 | 2 | 1 | 0 | 60 | 37 | 16 | 7 | 0
Fortress | 0d1c9255-d912-416c-befc-5f3e5e176df2 | 2 | 14 | AU CATALOGUE(9) | 8 | 4 | 2 | 0 | 92 | 51 | 23 | 18 | 0
Chasm | a455572d-3141-48bc-ac55-dac78d9b52c9 | 2 | 19 | AU CATALOGUE(14) | 11 | 6 | 2 | 0 | 91 | 31 | 45 | 15 | 0
Origin | b302eb62-da9a-480b-a409-3c89df8c1a04 | 2 | 13 | AU CATALOGUE(8) | 9 | 4 | 0 | 0 | 49 | 28 | 21 | 0 | 0
Snowbound | 410f1c01-aca6-4567-9df5-9b16bd550cb2 | 2 | 31 | AU CATALOGUE(19) | 21 | 6 | 4 | 0 | 115 | 51 | 38 | 26 | 0
Perilous | c5ac9f12-660e-4f1a-83e7-2e7536bbcb04 | 2 | 21 | AU CATALOGUE(12) | 13 | 4 | 2 | 2 | 93 | 40 | 31 | 16 | 6
Domicile | 921aebb1-783d-45e4-bacd-7ad869fa8dae | 2 | 22 | AU CATALOGUE(15) | 12 | 6 | 2 | 2 | 68 | 22 | 30 | 10 | 6
The Pit | 648ae7aa-c5d0-4f80-861a-79eb30440fcb | 1 | 20 | AU CATALOGUE(31) | 14 | 4 | 2 | 0 | 76 | 31 | 27 | 18 | 0
Flood Gulch | 7097bc4f-efcf-4c5a-a96e-4ddb03e84d2a | 1 | 23 | AU CATALOGUE(26) | 14 | 4 | 2 | 3 | 297 | 147 | 65 | 33 | 52
Goliath | 504ebf22-12b6-46c3-a9c1-ea20ca5bf03c | 1 | 7 | AU CATALOGUE(12) | 4 | 2 | 1 | 0 | 36 | 11 | 14 | 11 | 0
Dynasty | cfd90b63-62fd-441a-8015-8d7804b9c3c3 | 1 | 11 | AU CATALOGUE(12) | 7 | 3 | 0 | 1 | 50 | 18 | 21 | 0 | 11
Solution | ee43d273-8677-45c2-a8cd-aedd2c463dc9 | 1 | 5 | AU CATALOGUE(10) | 3 | 1 | 0 | 1 | 18 | 11 | 5 | 0 | 2
Kiken'na | df7dbf08-b8de-4ade-9d7f-1947128c9ae4 | 1 | 10 | AU CATALOGUE(11) | 7 | 2 | 1 | 0 | 32 | 12 | 12 | 8 | 0
Dredge | e4bb06db-065f-4902-b93b-d8dac315eac4 | 1 | 8 | AU CATALOGUE(10) | 6 | 2 | 0 | 0 | 59 | 39 | 20 | 0 | 0
Starboard | 7a9265af-a880-487b-8829-68d88fcfb145 | 1 | 7 | AU CATALOGUE(8) | 5 | 1 | 1 | 0 | 28 | 14 | 5 | 9 | 0
Empyrean | d035fc3e-f298-4c14-9487-465be2e1dc1f | 1 | 10 | AU CATALOGUE(17) | 6 | 3 | 1 | 0 | 41 | 13 | 21 | 7 | 0
Fortitude | 1ede38fa-4d30-4dfa-a8b7-5d08bf4e46e3 | 1 | 29 | AU CATALOGUE(38) | 20 | 6 | 3 | 0 | 111 | 62 | 32 | 17 | 0
Absolution | 78da545f-a168-4a5e-9c8d-dd379067c352 | 1 | 8 | AU CATALOGUE(14) | 6 | 1 | 1 | 0 | 33 | 19 | 7 | 7 | 0
Sylvanus | 95b69e4b-485f-4c6c-9b00-4bd68c94c1e9 | 1 | 9 | AU CATALOGUE(10) | 5 | 3 | 1 | 0 | 30 | 9 | 12 | 9 | 0
Smallhalla | 98783453-ce40-4020-9e87-62099a290b62 | 1 | 14 | AU CATALOGUE(27) | 10 | 4 | 0 | 0 | 32 | 20 | 12 | 0 | 0
Curfew | 63d634be-0319-489d-8c21-9c4e012f664f | 1 | 9 | AU CATALOGUE(13) | 7 | 0 | 0 | 2 | 28 | 20 | 0 | 0 | 8
Nemesis | 2be34415-bc96-4d02-875c-c4f2aa135f89 | 1 | 6 | AU CATALOGUE(8) | 4 | 1 | 1 | 0 | 31 | 13 | 9 | 9 | 0
Cliffhanger | 5324364b-39a8-4f93-96a6-b80a1f18ce8a | 1 | 0 | AU CATALOGUE(18) | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0
Opulence | 255bbe78-b191-476e-b0ae-0763c3bc2f44 | 1 | 8 | AU CATALOGUE(12) | 6 | 1 | 1 | 0 | 24 | 10 | 7 | 7 | 0
Insolence | d5c5eb4f-0dcb-4677-a866-eae0dcbfde9b | 1 | 40 | AU CATALOGUE(45) | 33 | 5 | 2 | 0 | 149 | 101 | 33 | 15 | 0
High Ground | bb7b78ae-3468-46ce-b5ba-cca61c3a338a | 1 | 15 | AU CATALOGUE(33) | 9 | 4 | 2 | 0 | 31 | 13 | 12 | 6 | 0
TOTAL SOCLES: 669 (rack 466 = 69.66%, power 144 = 21.52%, powerup 48 = 7.17%, non confirme 11 = 1.64%)
TOTAL PRISES: 2881 (rack 1480 = 51.37%, power 935 = 32.45%, powerup 381 = 13.22%, NON CLASSE 85 = 2.95%)
== 0.2 CROISEMENT role x famille d'emplacement (socles) ==
role | rack | power | powerup | non_confirme
(bonus) | 1 | 0 | 48 | 0
automatic | 122 | 2 | 0 | 2
power | 17 | 54 | 0 | 7
precision | 143 | 1 | 0 | 1
shotgun | 34 | 13 | 0 | 0
sidearm | 96 | 0 | 0 | 0
sniper | 9 | 74 | 0 | 1
special | 44 | 0 | 0 | 0
-- incoherences nommees --
[automatic sur power] 2 cas listes
    879a4dba [Fortitude] hinf_ravager (automatic)
    879a4dba [Fortitude] hinf_ravager (automatic)
[power sur rack] 17 cas listes
    2cf24f30 [Snowbound] hinf_hydra (power)
    4ecdf3e7 [High Ground] hinf_hydra (power)
    4ecdf3e7 [High Ground] hinf_hydra (power)
    5676a9ba [Insolence] hinf_hydra (power)
    5676a9ba [Insolence] hinf_hydra (power)
    5676a9ba [Insolence] hinf_hydra (power)
    5676a9ba [Insolence] hinf_hydra (power)
    8076f97f [Shogun] hinf_hydra (power)
    81c02726 [Isolation] hinf_hydra (power)
    879a4dba [Fortitude] hinf_hydra (power)
    879a4dba [Fortitude] hinf_hydra (power)
    a36c8bed [Isolation] hinf_hydra (power)
    a36c8bed [Isolation] hinf_hydra (power)
    b1ad85eb [Domicile] hinf_hydra (power)
    bfecd02b [Snowbound] hinf_hydra (power)
    daaa17d6 [Isolation] hinf_hydra (power)
    daaa17d6 [Isolation] hinf_hydra (power)
[precision sur power] 1 cas listes
    43716616 [Smallhalla] hinf_stalker_rifle (precision)
[sniper sur rack] 9 cas listes
    5676a9ba [Insolence] hinf_shock_rifle (sniper)
    5676a9ba [Insolence] hinf_shock_rifle (sniper)
    7f1bbf06 [Streets] hinf_shock_rifle (sniper)
    879a4dba [Fortitude] hinf_shock_rifle (sniper)
    879a4dba [Fortitude] hinf_shock_rifle (sniper)
    94a28b8b [Aquarius] hinf_shock_rifle (sniper)
    a0c36016 [Forest] hinf_shock_rifle (sniper)
    f0220a96 [Starboard] hinf_shock_rifle (sniper)
    faff9935 [Shogun] hinf_shock_rifle (sniper)
[special sur rack] 40 cas listes
    01e1f945 [Catalyst] hinf_sentinel_beam (special)
    0891225f [Perilous] hinf_needler (special)
    0891225f [Perilous] hinf_needler (special)
    0d265ab0 [Sylvanus] hinf_sentinel_beam (special)
    1b2d9e08 [Dynasty] hinf_sentinel_beam (special)
    21ece4d8 [Live Fire] hinf_sentinel_beam (special)
    28c9b538 [Chasm] hinf_sentinel_beam (special)
    2cf24f30 [Snowbound] hinf_needler (special)
    2cf24f30 [Snowbound] hinf_sentinel_beam (special)
    30a23d15 [Critical Dewpoint] hinf_sentinel_beam (special)
    32d9a94f [Perilous] hinf_needler (special)
    3923bede [Recharge] hinf_sentinel_beam (special)
    396cfc92 [Illusion] hinf_sentinel_beam (special)
    58864b3c [Domicile] hinf_needler (special)
    64e8adfa [Catalyst] hinf_sentinel_beam (special)
    7b0d89c4 [Behemoth] hinf_sentinel_beam (special)
    7b0d89c4 [Behemoth] hinf_sentinel_beam (special)
    8076f97f [Shogun] hinf_sentinel_beam (special)
    81c02726 [Isolation] hinf_needler (special)
    879a4dba [Fortitude] hinf_needler (special)
    879a4dba [Fortitude] hinf_sentinel_beam (special)
    a0c36016 [Forest] hinf_needler (special)
    a4083bd2 [The Pit] hinf_needler (special)
    a4083bd2 [The Pit] hinf_needler (special)
    a6ae19fb [Kiken'na] hinf_needler (special)
    a6ae19fb [Kiken'na] hinf_needler (special)
    ac03413d [Empyrean] hinf_sentinel_beam (special)
    b1ad85eb [Domicile] hinf_needler (special)
    b1ad85eb [Domicile] hinf_sentinel_beam (special)
    b8a44fe8 [Forest] hinf_sentinel_beam (special)
    bc60b4d9 [Illusion] hinf_needler (special)
    bfecd02b [Snowbound] hinf_needler (special)
    bfecd02b [Snowbound] hinf_sentinel_beam (special)
    c88ec007 [Live Fire] hinf_needler (special)
    d9781168 [Dredge] hinf_needler (special)
    d9781168 [Dredge] hinf_needler (special)
    e1259a69 [Takamanohara] hinf_sentinel_beam (special)
    e1259a69 [Takamanohara] hinf_sentinel_beam (special)
    e85d7bad [Recharge] hinf_needler (special)
    f0220a96 [Starboard] hinf_sentinel_beam (special)
== 0.3 CARTES ABSENTES DE LA REFERENCE ==
== 0.4 ARMES DE DEPART par categorie de mode (premiere vie de chaque slot) ==
-- Assassin (8819 armes de depart, 28 cles distinctes ; 8923 emissions loadout, 4406 slots)
    premiere-vie | hinf_ma40_ar | 4014 | 45.52%
    premiere-vie | hinf_sidekick | 2605 | 29.54%
    premiere-vie | hinf_br75 | 1709 | 19.38%
    premiere-vie | hinf_vestige_carbine | 69 | 0.78%
    premiere-vie | hinf_s7_sniper | 64 | 0.73%
    premiere-vie | hinf_bandit | 63 | 0.71%
    premiere-vie | hinf_vk78_commando | 41 | 0.46%
    premiere-vie | hinf_mangler | 30 | 0.34%
    premiere-vie | hinf_sentinel_beam | 28 | 0.32%
    premiere-vie | hinf_needler | 19 | 0.22%
    premiere-vie | hinf_disruptor | 17 | 0.19%
    premiere-vie | hinf_cqs48_bulldog | 16 | 0.18%
    premiere-vie | hinf_m41_spnkr | 15 | 0.17%
    premiere-vie | hinf_hydra | 14 | 0.16%
    premiere-vie | hinf_shock_rifle | 12 | 0.14%
    premiere-vie | hinf_skewer | 11 | 0.12%
    premiere-vie | hinf_mutilator | 11 | 0.12%
    premiere-vie | hinf_heatwave | 10 | 0.11%
    premiere-vie | hinf_cindershot | 9 | 0.10%
    premiere-vie | hinf_ravager | 9 | 0.10%
    premiere-vie | hinf_gravity_hammer | 9 | 0.10%
    premiere-vie | hinf_ma5k_avenger | 8 | 0.09%
    premiere-vie | hinf_dynamo_grenade | 7 | 0.08%
    premiere-vie | hinf_fuel_rod_spnkr | 7 | 0.08%
    premiere-vie | hinf_pulse_carbine | 7 | 0.08%
    premiere-vie | hinf_plasma_pistol | 7 | 0.08%
    premiere-vie | hinf_energy_sword | 6 | 0.07%
    premiere-vie | hinf_stalker_rifle | 2 | 0.02%
    toutes-vies  | hinf_ma40_ar | 6979 | 39.00%
    toutes-vies  | hinf_sidekick | 4684 | 26.17%
    toutes-vies  | hinf_br75 | 3651 | 20.40%
    toutes-vies  | hinf_s7_sniper | 563 | 3.15%
    toutes-vies  | hinf_bandit | 236 | 1.32%
    toutes-vies  | hinf_vestige_carbine | 207 | 1.16%
    toutes-vies  | hinf_m41_spnkr | 199 | 1.11%
    toutes-vies  | hinf_mangler | 142 | 0.79%
    toutes-vies  | hinf_vk78_commando | 141 | 0.79%
    toutes-vies  | hinf_mutilator | 139 | 0.78%
    toutes-vies  | hinf_skewer | 89 | 0.50%
    toutes-vies  | hinf_sentinel_beam | 88 | 0.49%
    toutes-vies  | hinf_needler | 86 | 0.48%
    toutes-vies  | hinf_cqs48_bulldog | 69 | 0.39%
    toutes-vies  | hinf_fuel_rod_spnkr | 68 | 0.38%
    toutes-vies  | hinf_dynamo_grenade | 68 | 0.38%
    toutes-vies  | hinf_shock_rifle | 66 | 0.37%
    toutes-vies  | hinf_disruptor | 60 | 0.34%
    toutes-vies  | hinf_gravity_hammer | 50 | 0.28%
    toutes-vies  | hinf_energy_sword | 49 | 0.27%
    toutes-vies  | hinf_hydra | 46 | 0.26%
    toutes-vies  | hinf_ravager | 44 | 0.25%
    toutes-vies  | hinf_ma5k_avenger | 37 | 0.21%
    toutes-vies  | hinf_heatwave | 35 | 0.20%
    toutes-vies  | hinf_cindershot | 33 | 0.18%
    toutes-vies  | hinf_pulse_carbine | 30 | 0.17%
    toutes-vies  | hinf_plasma_pistol | 28 | 0.16%
    toutes-vies  | hinf_stalker_rifle | 9 | 0.05%
-- BTB (1223 armes de depart, 20 cles distinctes ; 1735 emissions loadout, 612 slots)
    premiere-vie | hinf_bandit | 582 | 47.59%
    premiere-vie | hinf_ma40_ar | 552 | 45.13%
    premiere-vie | hinf_br75 | 23 | 1.88%
    premiere-vie | hinf_hydra | 11 | 0.90%
    premiere-vie | hinf_vestige_carbine | 7 | 0.57%
    premiere-vie | hinf_ravager | 7 | 0.57%
    premiere-vie | hinf_cqs48_bulldog | 6 | 0.49%
    premiere-vie | hinf_s7_sniper | 6 | 0.49%
    premiere-vie | hinf_vk78_commando | 5 | 0.41%
    premiere-vie | hinf_ma5k_avenger | 5 | 0.41%
    premiere-vie | hinf_pulse_carbine | 4 | 0.33%
    premiere-vie | hinf_disruptor | 3 | 0.25%
    premiere-vie | hinf_sidekick | 2 | 0.16%
    premiere-vie | hinf_shock_rifle | 2 | 0.16%
    premiere-vie | hinf_mutilator | 2 | 0.16%
    premiere-vie | hinf_mangler | 2 | 0.16%
    premiere-vie | hinf_needler | 1 | 0.08%
    premiere-vie | hinf_m41_spnkr | 1 | 0.08%
    premiere-vie | hinf_gravity_hammer | 1 | 0.08%
    premiere-vie | hinf_stalker_rifle | 1 | 0.08%
    toutes-vies  | hinf_bandit | 1391 | 39.70%
    toutes-vies  | hinf_ma40_ar | 1235 | 35.25%
    toutes-vies  | hinf_br75 | 211 | 6.02%
    toutes-vies  | hinf_s7_sniper | 96 | 2.74%
    toutes-vies  | hinf_hydra | 90 | 2.57%
    toutes-vies  | hinf_cqs48_bulldog | 71 | 2.03%
    toutes-vies  | hinf_ravager | 49 | 1.40%
    toutes-vies  | hinf_dynamo_grenade | 46 | 1.31%
    toutes-vies  | hinf_fuel_rod_spnkr | 46 | 1.31%
    toutes-vies  | hinf_m41_spnkr | 45 | 1.28%
    toutes-vies  | hinf_disruptor | 39 | 1.11%
    toutes-vies  | hinf_ma5k_avenger | 34 | 0.97%
    toutes-vies  | hinf_pulse_carbine | 33 | 0.94%
    toutes-vies  | hinf_shock_rifle | 20 | 0.57%
    toutes-vies  | hinf_mutilator | 19 | 0.54%
    toutes-vies  | hinf_vk78_commando | 18 | 0.51%
    toutes-vies  | hinf_vestige_carbine | 14 | 0.40%
    toutes-vies  | hinf_gravity_hammer | 14 | 0.40%
    toutes-vies  | hinf_skewer | 7 | 0.20%
    toutes-vies  | hinf_mangler | 6 | 0.17%
    toutes-vies  | hinf_energy_sword | 5 | 0.14%
    toutes-vies  | hinf_cindershot | 5 | 0.14%
    toutes-vies  | hinf_sidekick | 3 | 0.09%
    toutes-vies  | hinf_stalker_rifle | 2 | 0.06%
    toutes-vies  | hinf_needler | 2 | 0.06%
    toutes-vies  | hinf_sentinel_beam | 1 | 0.03%
    toutes-vies  | hinf_heatwave | 1 | 0.03%
    toutes-vies  | hinf_plasma_pistol | 1 | 0.03%
-- Super Fiesta (2622 armes de depart, 22 cles distinctes ; 2631 emissions loadout, 1309 slots)
    premiere-vie | hinf_gravity_hammer | 191 | 7.28%
    premiere-vie | hinf_energy_sword | 185 | 7.06%
    premiere-vie | hinf_plasma_pistol | 141 | 5.38%
    premiere-vie | hinf_disruptor | 130 | 4.96%
    premiere-vie | hinf_shock_rifle | 127 | 4.84%
    premiere-vie | hinf_mangler | 126 | 4.81%
    premiere-vie | hinf_ravager | 123 | 4.69%
    premiere-vie | hinf_pulse_carbine | 122 | 4.65%
    premiere-vie | hinf_stalker_rifle | 120 | 4.58%
    premiere-vie | hinf_needler | 119 | 4.54%
    premiere-vie | hinf_sentinel_beam | 116 | 4.42%
    premiere-vie | hinf_cqs48_bulldog | 114 | 4.35%
    premiere-vie | hinf_s7_sniper | 112 | 4.27%
    premiere-vie | hinf_sidekick | 111 | 4.23%
    premiere-vie | hinf_ma40_ar | 108 | 4.12%
    premiere-vie | hinf_heatwave | 107 | 4.08%
    premiere-vie | hinf_cindershot | 106 | 4.04%
    premiere-vie | hinf_m41_spnkr | 103 | 3.93%
    premiere-vie | hinf_vk78_commando | 98 | 3.74%
    premiere-vie | hinf_skewer | 92 | 3.51%
    premiere-vie | hinf_hydra | 88 | 3.36%
    premiere-vie | hinf_br75 | 83 | 3.17%
    toutes-vies  | hinf_gravity_hammer | 511 | 9.63%
    toutes-vies  | hinf_energy_sword | 374 | 7.05%
    toutes-vies  | hinf_sidekick | 263 | 4.96%
    toutes-vies  | hinf_shock_rifle | 257 | 4.84%
    toutes-vies  | hinf_s7_sniper | 253 | 4.77%
    toutes-vies  | hinf_disruptor | 253 | 4.77%
    toutes-vies  | hinf_m41_spnkr | 250 | 4.71%
    toutes-vies  | hinf_stalker_rifle | 248 | 4.67%
    toutes-vies  | hinf_needler | 242 | 4.56%
    toutes-vies  | hinf_mangler | 241 | 4.54%
    toutes-vies  | hinf_sentinel_beam | 241 | 4.54%
    toutes-vies  | hinf_plasma_pistol | 238 | 4.49%
    toutes-vies  | hinf_cqs48_bulldog | 234 | 4.41%
    toutes-vies  | hinf_ravager | 222 | 4.18%
    toutes-vies  | hinf_heatwave | 208 | 3.92%
    toutes-vies  | hinf_cindershot | 204 | 3.85%
    toutes-vies  | hinf_pulse_carbine | 201 | 3.79%
    toutes-vies  | hinf_ma40_ar | 191 | 3.60%
    toutes-vies  | hinf_vk78_commando | 190 | 3.58%
    toutes-vies  | hinf_skewer | 170 | 3.20%
    toutes-vies  | hinf_br75 | 162 | 3.05%
    toutes-vies  | hinf_hydra | 152 | 2.87%
```
