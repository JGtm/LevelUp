# PLAN — Chantier véhicules et tourelles (rejeu 2D)

> Ouvert le 2026-08-31 sur demande utilisateur. Worktree dédié
> `C:\Users\Guillaume\Projects\LevelUp-wt-vehicules`, branche `wt/vehicules-tourelles`
> (base `feat/v75` = `14a115bb1`). Mode : superviseur-orchestrateur + agents exécuteurs
> (Opus/Sonnet/Haiku), un lot = un commit sur cette branche, fusion `feat/v75` en clôture.

## Objectif (demande du 2026-08-31)

1. **Décoder les films** pour trouver les véhicules. Porteurs sûrs : Behemoth et
   Launch Site en Super Fiesta 4v4 (probable en Slayer classique — moins de véhicules,
   non aléatoires, à confirmer).
2. **Emplacements, spawns, cooldowns, état** (détruit ou non).
3. **Sons** : véhicules en mouvement, tirs de véhicules, tirs de tourelles.
4. **Sprites vue de dessus** par véhicule, style « jeu d'icônes un peu détaillé »,
   teintables à la couleur de l'équipe du conducteur (travaux cartes/icônes + Reclaimer
   en appui).

## État de l'art au 2026-08-31

- `film_re/VEHICULES_ARCHETYPE_40.md` (2026-07-27) : l'archétype « véhicule » est identifié
  (~48 composants). La moitié est COMMUNE au biped — i0 position, i1 vitesse, i2 orientation,
  i3 vitesse angulaire, i4/i5 vitalité corps/bouclier, i11 dead-state, i18 unit-control… —
  donc décodable avec les recettes existantes. Propres : i30-i37 (auto-turret ×3, type-state,
  type-physics, transformed/open, sentry-state, emp-timer). ATTENTION : doc écrit quand le
  compte du registre était faux — les INDEX sont à revérifier sur la table actuelle.
- **Percée trame 30-31/08** (Notion « Percer la trame du film » ; chantier mené HORS de ce
  dépôt — commits `fa9a2f30d`/`9c6f684e3` ABSENTS d'ici, notes NOTE_*_2026-08-30/31 absentes
  de `feat/v75`) : modèle de paquet `[1 bit config][liste événements : (1 [R(7) type]
  [3 réfs gardées][charge])* 0][trame de records ECS]` ; registre `chunk_00` = **50 blocs /
  49 porteurs / 1 067 couples** (les comptes « 118 » et « 174 » sont tous deux faux) ;
  événements véhicules MESURÉS sur 1 367 films : `biped_board_vehicle` 374 (154 films),
  `unit_exit_vehicle` 5 600 (279 films), `unit_enter_vehicle` (type 53) = 0 en arène ;
  occupant + siège lisibles. **Piège documenté** : tout NOM/NUMÉRO de type d'événement
  antérieur au 30/08 est sans valeur (« 125 embarquements sur carte sans véhicule »).
- **R-VÉHICULE** (killweapon, résolu) : 62 `weap` d'armement véhicule sur 194 (46 par `vehi`
  direct, 16 par la chaîne `vcdd → sofd → sofa → uwfa → weap`) ; distinction tourelle/fixe
  par classe ASCII + banque `_tur_`/`_veh_` du châssis. Ancre nommée : Gungoose.
- **Sons** : chaîne `.wem` prouvée deux fois (sons de fin de partie extraits du jeu Steam
  local ; lot equip-sons : chaîne `eqip → effe → snd! → sbnk → wem` ouverte).
- **Icônes** : 168 PNG d'armes extraits des `.module` (atlas `contour`), chaîne documentée
  dans `icones/ETAT_DE_L_ART_ICONES.md` (racine V7.5).
- **Tirs de TOURELLES : déjà trouvés et écoutés** (précision utilisateur du 31/08) — l'artefact
  « Tri des sons d'armes — Halo Infinite » (16/08,
  https://claude.ai/code/artifact/caed7613-83b7-40b1-867f-b35815e5f7b0) porte les sélections du
  chantier armes, tourelles détachables incluses. V3 REPART de ces sélections et n'extrait que
  ce qui manque (boucles moteur, tirs de véhicules non couverts).
- **Validation des sons = écoute utilisateur, toujours** : tout candidat sonore passe par une
  page d'écoute (artefact « Le Garage »,
  https://claude.ai/code/artifact/5d4451f5-abf6-4c32-b371-2129c7f3ef70) avant toute intégration.
- `internal/analysis/replay/attachement_phase0_vehicules_test.go` : sonde existante —
  l'attachement i10 ne dit ni porteur ni véhicule (gate 0 négatif, lot du 18/08).

## Contrainte de coordination

Le chantier trame/visée est ACTIF dans un autre espace de travail (Notion à jour du 31/08
17h44). **Interdit de réimplémenter son walker d'événements sans décision explicite** : le
lot V1 s'appuie d'abord sur la TRAME ECS (présente dans ce dépôt). L'« occupation par
événements » (board/exit) attend l'atterrissage de ce code ou une décision utilisateur.

## Lots

| Lot | Contenu | Gate |
|---|---|---|
| V0 | Cadrage (Opus) : archétype véhicule sur pièces, corpus Behemoth/Launch Site × cache films, voies d'identité du châssis, plan V1/V2 détaillé | Doc de cadrage vérifié sur pièces par le superviseur ; corpus exploitable chiffré OU alerte corpus |
| S1 | Scout sons (Sonnet) : outillage existant, banques `_veh_`/`_tur_`, boucles moteur | Rapport options + reco, preuve minimale |
| S2 | Scout sprites (Sonnet) : icônes véhicules dans les `.module` ?, voie modèle 3D → rendu orthographique, voie communautaire | Rapport options + reco |
| V1 | Détection + identité + position + état des véhicules | Sur ≥ 1 film Behemoth Super Fiesta : liste des véhicules + trajectoires plausibles + état détruit ; témoin négatif (carte sans véhicule → 0) ; seuils écrits AVANT mesure |
| V2 | Emplacements, spawns, cooldowns | Croisement créations/dead-state × `.mvar` (précédent : socles au centimètre ; piège : carte Forge = canevas + rack) |
| V3 | Sons : boucles moteur + tirs de véhicules (les tirs de tourelles repartent des sélections déjà écoutées de l'artefact du 16/08) | Écoute utilisateur sur page dédiée — les votes priment sur tout critère |
| V4 | Sprites top-down teintables | Gate visuel utilisateur |
| V5 | Intégration rejeu (schéma artefact +1, calque véhicules) | Hors périmètre immédiat — après V1/V2 |

## Règles d'exécution

- Agents : JAMAIS de commit, pas d'écriture `.ai/` hors livrables désignés, découvertes
  dans le rapport ; DuckDB en LECTURE SEULE (`ATTACH ... (READ_ONLY)`) — le serveur dev
  peut tenir les DB ; gates en avant-plan uniquement (jamais de run d'arrière-plan).
- Données réelles = `C:\Users\Guillaume\Projects\LevelUp\data` (le worktree n'a pas de
  data utile) : binaires PathResolver lancés avec `LEVELUP_REPO_ROOT` pointé sur le
  checkout principal. Cache films : `data/cache/film_chunks` + `film_manifests` (~1 367
  films), artefacts cuits : `data/cache/replays`.
- Un seul constructeur Go à la fois (cache de build partagé) : pendant V0, seul l'agent
  cadrage compile.
- Offline-pur et universel : aucune dépendance réseau au rejeu.
- Avant tout commit : skill `delivery-checklist` + entrée `thought_log.md` (copie du
  worktree, fusionnée avec le lot).
