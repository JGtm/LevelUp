# Plan : Emprise — la ressource « véhicules » (lot L7 du plan de l'onglet) — 2026-09-28

> Sources, à lire avant tout lot, et qui FONT FOI pour le rendu :
> - maquette de l'onglet `.ai/V7.5/MAQUETTE_ONGLET_TACTIQUE_ESCOUADE_2026-09-26.html` (l. 621
>   « Contrôle des ressources », l. 688 fil de la session, l. 933-934 match par match, l. 946
>   « Frags obtenus avec… », l. 997 « Rendement », l. 1269-1270 pastilles et courbes) ;
> - `.ai/PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26.md` (§0, §1 D2 / D9 / D10, §2 spécification
>   commune, L4 et L5 : une ressource = une entrée de liste, masquée si absente) ;
> - `.ai/V7.5/REFERENCE_CANAUX_EQUIPEMENT_2026-09-09.md` (à lire avant toute affirmation sur
>   l'équipement) ; `.ai/V7.5/HANDOFF_VEHICULES_2026-09-04.md` ; ADR 0034 (décodeur), 0026.
>
> Contrat d'exécution : skill `plan-execution`. Statuts `[x]` / `[~]` réf / `[!]` justifié ; aucune
> case vide à la clôture d'un lot.
>
> Statut : **GO utilisateur le 2026-09-30** (« attaque le plan des véhicules »). Exécution en cours
> sur `wt/emprise` (base `feat/v75`), conditions d'entrée amendées au §1.

## 0. Hors périmètre

- Ligne « pertes » des véhicules (voir D3).
- Sons, sprites, calque du rejeu 2D (chantier véhicules et tourelles, clos côté rejeu).
- Halo 5 (pas de film) : la ressource n'existe pas (règle D10 de l'onglet).
- Rattrapage prod : fait par l'utilisateur après le déploiement.

## 1. Conditions d'entrée (vérifiées par le superviseur avant L7.0 ; une seule manquante = pas de départ)

- **E1 (amendée le 2026-09-30 par le superviseur)** : `feat/suite-audit-decodeur` n'est PAS attendue.
  Elle est en plein travail dans une autre session (six sous-branches actives le 2026-09-30) et la
  fusionner de ce côté lui imposerait un rattrapage en cours de lots. À la place, une règle de
  frontière : L7 NE MODIFIE AUCUN des fichiers qu'elle réécrit (`film/replay/vehicle_rides*.go`,
  `build_vehicles.go`, `grammar/vehicle_occupancy*.go`, `film/replay/document_vehicles.go`) ; il LIT le
  calque véhicules du document de rejeu (`vehicles[].rides[]`, `family`, `part`, `carrier`) et accepte
  tout schéma ≥ 67 (71 sur `feat/v75`, 76 sur l'audit). Quand l'audit fusionnera, la dérivation se
  rejoue par sa commande de rattrapage ; aucun conflit de fichier attendu.
- **E2** Soirées témoins (relevé du superviseur du 2026-09-28 sur une copie de la base locale,
  frags dont la source est de classe véhicule, JGtm avec Madina97294 ou Chocoboflor dans le même
  camp) :
  - **principale : 24/07/2026**, Big Team Battle, 4 matchs (`fccc61cd` Launch Site, `879a4dba`
    Fortitude, `5676a9ba` Insolence, `4f77afc1` Flood Gulch — ce dernier est le film de référence du
    lot 5.10 de l'occupation), 131 frags de classe véhicule ;
  - **secondaire : 01/09/2026**, Quick Play, 4 matchs à véhicules (`f2966f08` et `7b0d89c4`
    Behemoth, `4ecdf3e7` High Ground, `bfecd02b` Snowbound), 30 frags de classe véhicule.
  Films en cache pour les huit ; artefacts au schéma 61 (sans occupation lue). Au total, la
  composition a 59 matchs à frags de véhicule en local (Snowbound, Behemoth, High Ground, Isolation
  en Quick Play surtout). L'utilisateur ne joue pas en général de modes à véhicules en escouade,
  mais d'autres utilisateurs de l'application le peuvent (2026-09-28) : la ressource est gardée.
- **E3 (amendée)** : les mesures de L7.0 construisent le document des huit témoins EN MÉMOIRE (comme
  le lot V0 du plan des vies) ; la recuisson réelle des huit artefacts (serveur arrêté, un film à la
  fois) n'est faite qu'à la clôture L7.5, pour la vérification sur données réelles.

## 2. Décisions tranchées

- **D1 — Lieu du calcul : dérivation de l'artefact** (nouvelle famille de `Deriver`,
  `sync/replayartifacts/derivations.go`), comme les deux autres ressources des mêmes cartes (bonus :
  famille d'usage ; armes spéciales et râteliers : `padtiers`). Une ressource calculée au sync
  pendant que ses voisines le sont à la cuisson donnerait deux populations de matchs à une même
  piste « Contrôle des ressources » et à une même case « sans film ». L'écart à la décision du
  2026-09-07 (« les données d'un match en base sont complètes au sync ») est donc celui de l'onglet
  entier, consigné au plan des vies (§7) et à statuer hors de ce plan. Précédent de forme :
  `match_flag_grabs_net` (famille, table append-only, capability fine, commande de rattrapage, ne
  redécode rien).
- **D2 — Une prise** = une vie de véhicule qui passe à un camp : le premier épisode (`rides[]`) d'un
  joueur de ce camp dans cette vie de véhicule, tout siège confondu ; un retour au même camp après
  un passage adverse est une nouvelle prise. Le joueur crédité est l'occupant de ce premier épisode
  (conducteur d'abord, siège 0, si deux épisodes commencent à la même image). Les épisodes publiés
  par le calque comptent tous (`src = film` et `src = proximity` : le calque a déjà écarté les
  replis contredits) ; la part `proximity` est publiée en couverture.
- **D3 — Pas de ligne « pertes ».** La maquette l. 1270 : « Les armes spéciales et les véhicules n'ont
  pas de perte mesurable par prise : leurs pastilles restent pleines ». La destruction n'est datée
  que sur 16 vies de véhicule sur 329 (relevé du 2026-09-28).
- **D4 — Temps à bord** = somme des épisodes (images × intervalle d'image), par camp et par joueur :
  barre fine « exposition » de « Frags obtenus avec les ressources » ; « Rendement face à
  l'adversaire » = frags par minute à bord.
- **D5 — Frags depuis un véhicule** = frags dont la SOURCE est de classe `vehicle` ou `turret`
  (`match_kill_events_latest`, classe lue par le registre qui sert déjà « Répartition des frags » et
  « Outils de destruction » : une seule définition dans l'app), tout le lobby, camp du tueur par
  `match_participants`. Écrasements (`CollisionDamage`, sans clé de registre) exclus, comme dans la
  Répartition des frags. C'est la lecture annoncée par le plan de l'onglet (L7 : « lecture de
  `match_kill_events_latest` sans filtre de tueur »).
- **D6 — Périmètre des véhicules** = toute vie de véhicule que le calque publie, tourelles fixes
  comprises ; le décor exclu par la MÊME règle que le rejeu (`vehicle_scenery.go`, appelée, jamais
  recopiée) ; une pièce montée (`part = turret`) appartient à son porteur. Grille « match par
  match » par famille (Warthog, Ghost, …) ; famille inconnue → « Véhicule inconnu ».
- **D7 — Couleur** : jeton `resource-vehicle` (déjà dans les quatre palettes depuis L0).
- **D8 — Un artefact sans occupation lue** (schéma < 67) : le match est « véhicules non mesurés »
  (même traitement que « sans film » pour cette ligne seulement), jamais zéro.

## 3. Organisation

Celle du plan des vies (§3) : worktree `wt/emprise`, un exécuteur Opus par lot, lots séquentiels
(un lot ne commence qu'une fois le précédent clos et vérifié), gate commun identique (dont
`lefthook run pre-push` sur les lots web et la clôture), plus `go test -tags=integration -p 1` des
paquets touchés pour L7.2 et L7.3 ; commits `feat(emprise-vehicules/<lot>)`. Reprise de session :
même protocole que le plan des vies (§5).

## 4. Lots

### L7.0 — Témoin et mesures (superviseur puis exécuteur) · rapide

- [x] L7.0.1 Documents des huit témoins construits en mémoire (E3 amendée) ; frontière E1 tenue.
  Les huit documents sortent au schéma 71 (code actuel) ; aucun fichier de la frontière E1 touché
  (`git status` : deux tests de recherche neufs, rien d'autre).
- [x] L7.0.2 Sur les matchs du témoin : prises (D2), temps à bord (D4), part `proximity`, frags de
  classe véhicule par camp (D5). **Seuils écrits avant la mesure : ≥ 90 % des frags de classe
  véhicule d'un joueur de l'escouade tombent pendant un de ses épisodes publiés ; part `proximity`
  ≤ 50 % des épisodes.** Manqué : STOP, rapport à l'utilisateur. **Verdict : les deux seuils sont
  tenus à la lettre** (45,2 % ; 11/11), avec les réserves nommées ci-dessous.

**Instruments** (tests de recherche, sautés sans données, rien écrit dans `data/`) :
`platform/duckdb/emprise_vehicules_l70_frags_research_test.go` (`TestEmpriseL70Frags` : frags D5
par la requête, le classificateur `halo_infinite.NewKillSourceRegistry` et le résolveur de classe
`resolveWeaponKeyDimensions` du lecteur de la « Répartition des frags », sur copies de `shared` et
`metadata`) puis `replaybuild/emprise_vehicules_l70_research_test.go` (`TestEmpriseL70` : document
en mémoire par `v0Document`, horloge du lot V0.1 — film µs = origine + f × pas, match ms = film ms
− `coverage.bridge.deathOffsetMs`). Prises D2 : par vie (pièce montée repliée sur son porteur),
épisodes triés par `t0` puis siège (conducteur d'abord), une prise à chaque changement de camp ;
camp de l'occupant par `roster[].team` du film (0 discordance avec `match_participants.team_id`
sur les huit matchs).

**Prises (D2) et temps à bord (D4), par match** (camp 0 / camp 1 ; temps en s) :

| Match | Vies (sans épisode) | Épisodes (dont `proximity`) | Prises 0 / 1 | Temps à bord 0 / 1 |
|---|---|---|---|---|
| fccc61cd Launch Site | 15 (12) | 3 (3 = 100 %) | 2 / 1 | 15 / 21 |
| 879a4dba Fortitude | 44 (20) | 46 (14 = 30,4 %) | 16 / 12 | 523 / 258 |
| 5676a9ba Insolence | 60 (33) | 63 (41 = 65,1 %) | 16 / 18 | 543 / 944 |
| 4f77afc1 Flood Gulch | 146 (87) | 111 (44 = 39,6 %) | 32 / 29 | 1262 / 1343 |
| f2966f08 Behemoth | 12 (8) | 8 (3 = 37,5 %) | 4 / 2 | 148 / 45 |
| 7b0d89c4 Behemoth | 10 (6) | 5 (3 = 60,0 %) | 3 / 1 | 289 / 38 |
| 4ecdf3e7 High Ground | 9 (7) | 2 (1 = 50,0 %) | 0 / 2 | 0 / 14 |
| bfecd02b Snowbound | 11 (9) | 3 (0 = 0 %) | 1 / 1 | 158 / 119 |
| **Total** | 307 (182) | **241 (109 = 45,2 %)** | 74 / 66 | 2938 / 2782 |

Familles des prises (total des huit, 140) : warthog 42, mongoose 25, ghost 19, wraith 18, falcon 17,
banshee 9, chopper 4, famille inconnue 4 (879a4dba 2, 4f77afc1 2), shade 2. 3 épisodes sans
xuid (4f77afc1), 0 sans camp, 2 pièces montées orphelines (porteur absent du document).

**Frags de classe véhicule ou tourelle (D5), par camp du tueur** : 24/07 = 101 (fccc61cd 2, 879a4dba
13, 5676a9ba 35, 4f77afc1 51 dont 3 tourelle) ; 01/09 = 30 (f2966f08 4, 7b0d89c4 14, 4ecdf3e7 1,
bfecd02b 11). Par match (camp 0 / 1) : 2/0, 2/11, 15/20, 26/25, 4/0, 12/2, 1/0, 5/6.

**Seuil frags (escouade, tueur = JGtm, Madina97294 ou Chocoboflor)** : 11 frags, 11 pendant un de
leurs épisodes publiés, dès la lecture stricte (bornes incluses ; ±500 ms et ±1 s donnent le même
compte) — JGtm 1 (5676a9ba), 5 (4f77afc1), 5 (bfecd02b) ; Madina97294 et Chocoboflor 0.

**Réserves, à lire avant L7.1** :

1. **L'échantillon du seuil est petit : 11 frags**, tous de JGtm ; les 120 autres
   frags d engin des huit matchs sont ceux du reste du lobby.
2. **Tout le lobby, la couverture des épisodes tombe à 62 % (81/131)** : 879a4dba 9/13, 5676a9ba
   27/35, 4f77afc1 20/51 (39 %), f2966f08 1/4, 7b0d89c4 13/14, fccc61cd 0/2, 4ecdf3e7 0/1,
   bfecd02b 11/11. Cause connue : la primitive n'attribue qu'une part des vies de véhicule (limite
   publiée au calque). Conséquence pour D4 : un « rendement par minute à bord » par CAMP a un
   dénominateur (le temps à bord) plus bas que la réalité ; celui d'un joueur de l'escouade, qui
   tient ses frags dans ses épisodes, n'a pas ce biais. À statuer pour le bloc (L7.3), pas ici.
3. **`proximity` : 45,2 % des épisodes au total, mais 65,1 % à Insolence, 60 % à un Behemoth, 100 %
   à Launch Site.** En temps à bord, `proximity` pèse 3023 s sur 5720 s (52,8 %), au-dessus de 50 %.
   Le seuil du plan porte sur le COMPTE d'épisodes ; il est tenu sur l'ensemble des huit.
4. **Écart avec le relevé du 2026-09-28** : D5 ne compte que les sources à clé de registre (classe
   `vehicle` ou `turret` dans `metadata.weapons`) ; 30 frags de plus du 24/07 (13 à 879a4dba, 17 à
   4f77afc1) ont une source de classe « VEHICULE » SANS clé de registre (les écrasements, exclus
   par D5 comme dans la Répartition des frags) — 101 + 30 = 131, le chiffre du relevé.

Journal : [2026-09-30] L7.0 joué en avant-plan (une passe de 1188 s sur les huit films, pic mémoire
sous le plafond de 8 Gio) ; seuils tenus ; L7.1 autorisé.

### L7.1 — Projection pure (Go) · moyen

- [ ] L7.1.1 Fonction pure depuis `ReplayDocument` (vies de véhicule, `rides`, famille, décor) :
  prises par camp et par joueur, temps à bord, par famille ; tests synthétiques (changement de camp,
  sièges simultanés, décor, pièce montée, famille inconnue, artefact sans occupation).

### L7.2 — Écriture (Go, persistance — lot sensible) · lourd

- [ ] L7.2.1 Tables append-only `match_vehicle_takes` (+ vue `_latest`), persister INSERT-only,
  inscriptions aux garde-rails (`no_art_patterns_test.go`, `append_only_state_guard_test.go`,
  `compaction_registry.go` + e2e), ordre des migrations.
- [ ] L7.2.2 Famille dans `Deriver` (une lecture de l'artefact, même marque par match), capability
  fine `film.vehicle_usage` dans `capabilities.toml` de Halo Infinite.
- [ ] L7.2.3 Commande `levelup backfill-vehicle-takes` (modèle `backfill-pad-tiers` : reprenable,
  `--dry-run`, `--match`, serveur arrêté).
- [ ] L7.2.4 Tests d'intégration (`-tags=integration -p 1`).

### L7.3 — Bloc Emprise (Go) · moyen

- [ ] L7.3.1 Ressource `vehicle` dans `analysis/squademprise` (`resourceOrder`, prises, temps,
  frags D5, rendement par minute à bord, grille par famille), lecture bornée (ADR 0036), contrat.

### L7.4 — Cartes (web) · moyen

- [ ] L7.4.1 `RESOURCE_ORDER`, `resourceColors.ts`, textes FR / EN dans un fichier de textes neuf
  (`empriseStrings.ts` à 490 lignes), lignes des sept cartes (maquette l. 621, 688, 933-934, 946,
  997, 1269-1270), pastilles pleines (D3).

### L7.5 — Clôture (superviseur)

- [ ] L7.5.1 Revue adversariale, rattrapage local, vérification sur le témoin, fusion, CI verte,
  gate visuel utilisateur après fusion ; prod par l'utilisateur.

## 5. Découvertes (à consigner ici, pas à traiter)

- Les compteurs de l'API `VehicleDestroys`, `DriverAssists`, `Hijacks` sont déclarés
  (`openspartan/halo_api_payload.go:134-136`) mais jamais persistés.
- `VehicleTransferDamage` (2 256 frags en local) : sens non établi.
