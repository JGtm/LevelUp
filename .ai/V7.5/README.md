# V7.5 — recherche et rétro-ingénierie (rejeu 2D, événements de frags/morts)

Ce dossier rassemble les documents de **recherche et de rétro-ingénierie** produits dans les
deux worktrees `feat/filmdec-*` (décodage du film Theater, arme/source de dégât par kill) et
`feat/replay2d-*` (rejeu 2D), réunis sur `feat/replay2d-prod` le 2026-07-31.

Ce sont des **archives de chantier** : elles font foi sur ce qui a été prouvé, mesuré ou
réfuté. Ce qui reste à faire ou à terminer n'est pas ici mais à la racine de `.ai/`
(voir « Ce qui est resté à la racine » plus bas).

## Organisation

Comptes relevés le 2026-09-23 (lot d'archivage de la racine) : fichiers posés directement dans
le dossier / fichiers suivis par git, sous-dossiers compris (un seul nombre quand ils sont égaux).

| Dossier | Contenu | Fichiers |
|---|---|---|
| racine de `V7.5/` | Plans, audits, rapports et handoffs de la campagne v7.5 archivés avant la création des sous-dossiers thématiques (passes H6 et 2026-09-13) | 152 |
| `film_re/` | Format du film Theater : grammaire, chunks, décodeur ECS, keyframes, RE Ghidra, handoffs externes, préparations des jalons du décodeur | 158 / 203 |
| `killweapon/` | Arme / source de dégât par kill : kill feed, dead-state, same-clock, walk biped, journal RE, précision des projectiles | 26 |
| `replay2d/` | POC du rejeu 2D puis ses chantiers : trajectoires, inventaire/loadout, vérité terrain, plans de lots, export vidéo, retours du rejeu | 103 / 735 |
| `chantiers/` | **Créé le 2026-09-23.** Chantiers v7.5 hors film : UI (Explorer, couleurs), escouade et annuaire, sync, fork, classement mondial, pilotage, arbitrages, revues | 14 |
| `cartes/` | Géométrie 2D des maps depuis les `.module`, triangles, noms de zones | 27 / 28 |
| `icones/` | Icônes d'armes et du **kill feed** extraites des `.module` : chaîne, tables de correspondance, page de nommage, planches-contact | 8 |
| `v2/` | Chantier v2 rejeu/film (gelé le 2026-09-07 ; suite `../PLAN_V2_RESTES_2026-09-07.md`) | 31 |
| `briefs_ajustements_2026-09-13/` | Briefs d'exécutant des lots de `../PLAN_AJUSTEMENTS_PRE_V75_2026-09-13.md` | 9 |
| `outillage/` | Sorties d'outillage (couverture des objectifs, palette Forge, précision des projectiles) | 0 / 41 |
| `reference/` | Référence de mécanique de jeu (disparition des armes) | 1 |
| `dumps/` | Captures binaires, CSV, PNG (ex-`.ai/re_dump/`) — 76 Mo, lus par du code | 39 / 78 |

Plan actif d'habillage du rejeu 2D (marqueurs, noms, amis, logo, rangee `fil | carte | fiches`) :
`replay2d/PLAN_HABILLAGE_REJEU_2D.md` (ecrit le 2026-08-16, decisions D1-D8 a valider par le user).

Quatre lots de RECHERCHE du 2026-08-17 (branches `wt/ti37-identite`, `wt/ti11-objectifs`,
`wt/kf-grammaire`, `wt/kf-file-entite`, fusion triee sur `wt/fusion-lots-go`) — leurs plans
portent le detail mesure, et trois d'entre eux sont des NEGATIFS qui ferment des voies :

- `replay2d/PLAN_R3_IDENTITE_TI37.md` — l'identite de l'objet `ti=37` est le GlobalID d'un tag
  `eqip` (428 occurrences sur 428, zero ailleurs). Confirmation INDEPENDANTE de la meme
  decouverte que le lot de production `replay2d/PLAN_IDENTITE_TI37.md` ; c'est ce dernier qui a livre
  les poses et le nommage. Le code d'instrumentation de R3 n'a PAS ete fusionne (il aurait
  double le lecteur de `equipment_creation.go`).
- `replay2d/PLAN_R4_OBJECTIFS_VIVANTS_TI11.md` — `ti=11` est le DESCRIPTEUR d'objectif du HUD,
  pas l'objet : aucun de ses 34 composants ne porte de position. Voies delta et image-cle
  refutees, chacune par son temoin.
- `replay2d/PLAN_R5_GRAMMAIRE_IMAGE_CLE.md` — le corps d'un record d'image-cle n'est PAS un
  record NEW (128 decalages x 16 lectures x 3 films, jamais plus de 1,8 %). Acquis positif :
  la grammaire de l'etat par defaut de `ti=42` est decompilee bit-exact (`FUN_1407f0c68`), mais
  NON BRANCHEE dans le decodeur — aucun oracle ne la valide (decision du 17/08). Elle vit dans
  le plan et dans `killweapon/WALK_PORT_NOTES.md` § IMAGE-CLE §4.
- `replay2d/PLAN_R6_FILE_PAR_ENTITE.md` — le lecteur de film du jeu SAUTE le payload type-2 :
  il n'y a aucun consommateur a decompiler. La file par entite n'est pas une transformation,
  et la capture live de juillet portait sur le premier paquet DELTA, pas sur une image-cle.

Cinq lots de RECHERCHE du 2026-08-17 sur l'IMAGE-CLE du bipede (branches `wt/kf-biped-etat-complet`,
`wt/kf-biped-bit-exact`, `wt/kf-encodage-drapeau`, `wt/kf-ecrivain-vtable`,
`wt/kf-boucle-etat-complet`), lus dans cet ordre : chacun corrige une conclusion du precedent, et
le dernier pose la BORNE D'ARRET. **La RE de l'image-cle est ARRETEE (decision utilisateur apres
R7-e)** — ce qui suit est ce qui reste acquis, pas un chantier ouvert.

- `replay2d/PLAN_R7A_IMAGE_CLE_BIPEDE_ETAT_COMPLET.md` — le corps d'un record d'image-cle a la
  TAILLE d'un etat complet (102-104 % de la longueur reelle) mais pas les BITS (0,51 % d'exactitude
  sur 591 records). La FORME est tranchee ; le verrou qu'il nomme (i57/i59/i60) sera refute par
  R7-b.
- `replay2d/PLAN_R7B_BIPEDE_IMAGE_CLE_BIT_EXACT.md` — **le seul lot de la serie a corriger la
  PRODUCTION** : la porte du composant i9 `object-multiplayer-properties` etait INVERSEE dans le
  port Go (le bloc TLV se lit quand le bit vaut ZERO). Chemin delta, donc tous les films. Ecart
  median -45 a -55 % ; l'exactitude, elle, ne bouge pas.
- `replay2d/PLAN_R7C_ENCODAGE_DRAPEAU_IMAGE_CLE.md` — NEGATIF net : les deux drapeaux d'encodage
  existent (`DAT_144e61ea0` portee, `DAT_145121140` reglage de process), mais le payload d'image-cle
  est ecrit HORS de la portee — sa position est QUANTIFIEE aux largeurs de la carte. Acquis :
  le lecteur d'etat complet du jeu est NOMME.
- `replay2d/PLAN_R7D_ECRIVAIN_VTABLE.md` — l'ECRIVAIN est la case `+0x18` de la vtable d'un
  descripteur de composant, retrouvee sans xref par dump de `.rdata`. Quatre ports Go sur cinq
  confirmes largeur pour largeur, **dont la polarite d'i9 de R7-b, verifiee independamment**.
- `replay2d/PLAN_R7E_BOUCLE_ETAT_COMPLET.md` — la boucle d'etat complet portee telle quelle, ses
  cinq variables mesurees une a la fois : aucune n'ecrase la dispersion. **Le CADRE du record n'est
  pas la cause ; la derive est DANS les deserialiseurs.** Borne d'arret atteinte et respectee.

- `replay2d/PLAN_CORRECTIF_REVUE_POSES.md` — le correctif de la revue adversariale du lot des poses
  d'equipement (2026-08-17). Son acquis central est un NEGATIF MESURE qui change l'affichage : `t1`
  n'est PAS la disparition d'un objet pose, c'est sa MISE AU REPOS — une borne INFERIEURE. Le film
  ne date la disparition d'aucun equipement (record DEL noye dans 78 090 / 158 098 candidats). Le
  calque cesse donc d'effacer a `t1` : le capteur se tient a ses 15 s officielles, les autres poses
  vont jusqu'a la fin du rejeu.

À la racine de `V7.5/` : `RECHERCHE_CTF_TIRS_PERDUS.md` — le verdict de la **décision #2** du
master plan (pourquoi le rejeu perd des tirs, et si le rejeu public est livrable). Ses sorties
brutes sont sous `replay2d/mesures_ctf_2026-08-08/`.

À la racine de `V7.5/` également : `PLAN_RECONCILIATION_BRANCHES.md` — la réconciliation des deux
lignées (killweapon + rejeu 2D) sur `feat/replay2d-prod`, close le 2026-07-31. Son §5 porte les
sept grandeurs de non-régression des trois films, encore citées comme gate par
`PLAN_DETTE_AVANT_MERGE.md`.

### Points d'entrée par sujet

- **Arme par kill** : `../README_KILLWEAPON_INDEX.md` (index maître, à greper en premier),
  puis `killweapon/RE_LOG_KILLWEAPON.md` (journal, ne jamais le lire par le haut).
- **Grammaire ECS (archétype × composant)** : la table de référence est
  `apps/go-api/internal/games/halo_infinite/film/internal/grammar/testdata/ecs_table.tsv` (1 067 couples du registre du
  film + 14 alias ; statut de portage, niveau, source `fichier:ligne`, deser, champ du document),
  tenue par les garde-rails G1-G3 de `film/internal/grammar/ecs_table_guard_test.go`. Le plan qui l’a produite
  et la vérification de l’inventaire qui l’a précédée : `film_re/PLAN_TABLE_ECS.md`.
- **Format du film** : `film_re/GRAMMAIRE_RECORD_FILM.md` puis
  `film_re/RECETTE_DECODAGE_FILM_CHUNKS.md`.
- **Proprietaire d'un corps (qui occupe ce slot de bipede)** :
  `film_re/SONDAGE_E2_BIPEDE_INDEX_2026-09-08.md` — POSITIF. Le record de CREATION d'une entite
  `ti=35` porte l'index de participant sur 5 bits (`ECS_ReadEntityRefIndex5`, offset +67 de
  l'en-tete) ; 90,3 / 86,8 / 95,0 % des vies nommees directement sur trois films, temoin fantome
  a zero. Il ferme le point aveugle E2 de `v2/RESTES_P1_INVENTAIRE_2026-09-07.md` et montre que
  le pont par morts ECHANGE les noms de deux vies qui finissent a la meme image (7 paires).
  Mesures brutes : `film_re/mesures_e2_2026-09-08/`.
- **Reverse externe / handoff** : `film_re/HANDOFF_FILM_EXTRACTION_EXTERNAL_DEV.md`,
  `film_re/GITHUB_RE_FINDINGS_EN.md` (EN).
- **Cartes** : `cartes/HANDOFF_MAP_GEOMETRY_FROM_MODULES.md`.
- **Icônes (armes, véhicules, kill feed)** : `icones/ETAT_DE_L_ART_ICONES.md` — chaîne complète,
  tables index → arme/nom, pistes réfutées. Le nommage restant se fait dans
  `icones/NOMMAGE_ICONES.html` (page locale, hors app).

## Ce qui est resté à la racine de `.ai/` (état final au 2026-09-23)

La racine porte **52 fichiers suivis** (plus `mock_effets_fiches_zones.html`, ignoré par
`.gitignore`, présent dans le seul checkout de l'utilisateur). Chaque document y est pour une
raison écrite ci-dessous ; le verdict a été rendu sur preuves le 2026-09-23 (en-tête du
document, `thought_log.md`, `git log`, références entrantes). Les chemins sont relatifs à `.ai/`.

**Journal, carte, backlog** : `thought_log.md`, `project_map.md`, `BACKLOG.md`.

**Chantier vivant du décodeur de film** : `PLAN_DECODEUR_FILM_2026-09-13.md` (le lot 5.26 s'y
écrit) et `HANDOFF_DECODEUR_FILM_SERIE5_2026-09-22.md` (vivant tant que 5.26 n'est pas statué).

**Plans et handoffs ouverts (VIVANT)** :

| Document | Pourquoi il reste |
|---|---|
| `PLAN_FINITIONS_2026-09-13.md` | arbitrages utilisateur en attente |
| `PLAN_NIVEAUX_ARMES_2026-09-13.md` | planifié, non démarré |
| `PLAN_PRISES_NETTES_DRAPEAU_2026-09-13.md` | décision utilisateur du 13/09, non livré |
| `PLAN_LIBELLES_EN_DUR_GO_2026-09-07.md` | ouvert, six décisions en attente |
| `PLAN_TACTIQUE_SUITE_2026-09-07.md`, `DECOUVERTES_TACTIQUE_2026-09-07.md` | suite du chantier Tactique et son registre de découvertes |
| `PLAN_V2_RESTES_2026-09-07.md` | restes du chantier v2 (lot P) |
| `PLAN_DUELS_PORTEE_2026-09-06.md` | livré sur `feat/duels`, non fusionné (décisions utilisateur) |
| `PLAN_FRISE_POINT_DE_VUE_2026-09-06.md` | livré, non poussé |
| `PLAN_FICHES_COMPACTES_BTB_2026-09-06.md` | livré, fusion au signal |
| `PLAN_AJUSTEMENTS_PRE_V75_2026-09-13.md` | livré, découvertes §4 en attente d'arbitrage |
| `PLAN_FORK_ET_RELEASE_2026-09-11.md`, `PROCEDURE_BASCULE_LEVELUP_2026-09-13.md` | F.4 (bascule vers le dossier `LevelUp`) non faite ; la procédure attend le signal de l'utilisateur |
| `PLAN_EQUIPEMENT_GACHIS_2026-09-09.md` | en-tête : « reste ouvert ici : les découvertes §6 et P5 » |
| `PLAN_ORCHESTRATION_2026-09-07.md` | table de suivi ouverte sur P3-P5, L6-L8 et R9 (release) |
| `PLAN_QUANTUM_PROJECTILES_ET_BORNES_FORGE_2026-09-14.md` | plan écrit le 14/09, 30 cases vides, aucune exécution au journal |
| `PLAN_REPLI_GAME_CHANGERS_2026-09-05.md` | lot I (graphes partagés) non fait, « les graphes attendent la clé » (journal du 05/09) |
| `PLAN_RETOURS_REJEU_MATCHVIEW_2026-09-02.md` | lots 1-3 et 5 faits, lot 4 (effets de fiche) jamais exécuté |
| `HANDOFF_ASSAUT_DESAMORCAGE_2026-09-04.md` | désamorçage hors lot, condition de reprise écrite |
| `HANDOFF_VEHICULES_2026-09-04.md` | journal du 04/09 : « 3 validations utilisateur en attente » |

**Références (citées par `CLAUDE.md`, du code, un ADR ou un plan vivant)** :
`REFERENCE_CANAUX_EQUIPEMENT_2026-09-09.md` (CLAUDE.md), `ETAT_DE_L_ART_KILLWEAPON.md` et son
`ADDENDUM_ETAT_DE_L_ART_2026-07-26.md`, `README_KILLWEAPON_INDEX.md`, `GUIDE_WEAPON_SHOTS.md`,
`ETAT_DE_L_ART_FORGE_PALETTE_ZONES.md` (chantier Forge post-release),
`ARCHITECTURE_CIBLE_DECODEUR_FILM_2026-09-12.md` (ADR 0034, plan du décodeur),
`AUDIT_LECTEURS_VIES_ANONYMES_2026-09-06.md` (code web, restes v2), `REFERENCE_WEAPON_IDS.md`
(`docs/WEAPONS.md`), `I18N_REFERENCE.md`, `ENRICHMENTS_CATALOG.md` (`persist/doc.go`, ADR 0019),
`CHARTS_AND_TABLES.md` (`charts_specs/README.md`), `MCC_UNOFFICIAL_API_REFERENCE.md`
(`cmd/probe-mcc`), `STEAKTACULAR.md` (code du comeback), `duckdb_7659_upstream_report.md`
(ADR 0021, réponse amont attendue).

**À ARBITRER (restés en place, la question est posée à l'utilisateur)** :

| Document | Question |
|---|---|
| `HANDOFF_DECODEUR_FILM_2026-09-13.md`, `PREPARATION_M2_PAS_4_A_6_2026-09-17.md`, `PREPARATION_M4_ORDRE_ET_FRONTIERES_2026-09-17.md` | Recherche close et jalons M2/M4 clos, mais `PLAN_DECODEUR_FILM_2026-09-13.md` les cite et ne se touche pas pendant 5.26 : les déplacer vers `film_re/` à la clôture de 5.26, avec la citation réécrite ? |
| `PLAN_ARME_FAVORITE_BRIEFING_EXPLORER_2026-09-17.md` | Journal « gate visuel en attente », puis trois fusions dont deux « retour du gate visuel » (`e80841a6f`, `a21640979`, `248f49750`) ; les cinq scénarios de l'étape 7 ne sont pas cochés. Gate visuel tenu ? |
| `PLAN_COMPARE_PROFIL_ARMES_2026-09-17.md` | Fusionné (`5b2298f06`) ; 5.2, 5.4 (gate visuel) et 5.5 non cochés, journal « gate visuel et fusion en attente ». Gate visuel tenu ? |
| `PLAN_EXPLORER_PORTEE_FRAGS_2026-09-17.md` | Fusionné (`753005b1a`) ; journal : gate visuel encore à faire, 4.2/4.4/4.5 non cochés. Gate visuel tenu ? |
| `PLAN_ESCOUADE_HORS_CADRE_2026-09-09.md` | Phases fusionnées (vagues 1, 3, 4) ; R1-R4 (revues, checklist, journal de clôture) jamais statués. Couverts par les revues de vague du plan maître ? |
| `PLAN_FINALISATION_REJEU_2D.md` | Plan « actif » au 2026-08-05, intact depuis le 08/08, 36 cases vides : encore l'autorité du rejeu 2D, ou dépassé par les chantiers v2 et décodeur ? |
| `PLAN_MASTER_FILM_KILLFEED_REJEU.md` | J5.1-J5.4 (merge, backfill prod, archivage des branches de recherche) non cochés ; cité par `migration/build_queue_schema.go`. Clos par la release v7.5 à venir ? |
| `PLAN_OBJECTIFS_TEMPS_REEL.md` | 16 cases vides depuis le 31/07 ; cité avec son chemin par `apps/go-api/internal/games/halo_infinite/film/internal/grammar/sonde_ti11_objectifs_test.go`, source hachée de la révision de grammaire : le déplacer exige soit une référence cassée, soit une montée de `grammar.Rev`. Le garder ici ? |
| `PLAN_DEPS_ECHARTS_TS7_2026-07-27.md` | Lots A (deps Go : `httprate` 0.16.0) et B (`echarts` ^6.1.0) faits hors plan ; lot C (TypeScript 7) non fait (`typescript` ~6.0.2). Garder pour TS7 ou clore ? |
| `PLAN_REVUE_ANALYTIQUE_TIMESERIES_SQUAD_2026-07.md` | Seul Z1 (tournée visuelle avec l'utilisateur) reste vide depuis le 26/07 (v7.2.1) : faite, ou à clore vers `archive/` ? |
| `mock_effets_fiches_zones.html` (hors git) | Maquette locale ignorée par `.gitignore` : liée au lot 4 de `PLAN_RETOURS_REJEU_MATCHVIEW_2026-09-02.md` ? La garder, la ranger sous `mocks/`, ou la supprimer ? |

Documents autrefois listés dans cette section et archivés depuis (chemins relatifs à `V7.5/`) :
`killweapon/ETAT_DE_L_ART_CHANTIER_VOISIN.md`, `killweapon/HANDOFF_PRECISION_PROJECTILES.md`,
`killweapon/PLAN_BRANCHEMENT_KILLSOURCE.md`, `replay2d/CONCEPTION_INVERSION_PRESEANCE.md`,
`PLAN_DETTE_AVANT_MERGE.md`, `HANDOFF_SUPERVISEUR_2026-08-03.md`, `PLAN_CAPTURE_EXPORT_REJEU.md`,
`PLAN_EXPORT_VIDEO_HORS_TEMPS_REEL.md`, `replay2d/SUIVI_REPLAY_2D.md`, `replay2d/ETAT_DU_POC.md`,
`replay2d/CAHIER_DES_CHARGES_POC.md`, `replay2d/HANDOFF_REPLAY_2D_2026-07-29.md`,
`replay2d/PLAN_CAPACITES_ACTIVES.md`, `icones/PLAN_RECHERCHE_ASSETS_ICONES.md`,
`cartes/PLAN_BELLE_CARTE_TRIANGLES.md`, `replay2d/CLE_USB_REJEU_2D.md`,
`film_re/PLAN_VARIABLES_JETEES.md`, `film_re/HANDOFF_DUMPS_2026-07-31.md`,
`film_re/SESSION_CAPTURE_AVANT_PC.md`, `replay2d/thought_log_replay.md`.

## Passes d'hygiène de la racine (historique)

> **Passe d'archivage du 2026-09-23 (lot `feat/ai-archivage-v75`).** 21 documents clos ont quitté
> la racine par `git mv`, contenu intact ; les références de chemin ont été réécrites dans les
> documents restés vivants, dans cinq commentaires de code et, pour l'ADR 0035, par une ligne
> « moved to » à côté de la citation. Une ligne par document, rangée par dossier :
>
> `chantiers/` (nouveau) :
>
> - `chantiers/ARBITRAGE_DETTE_2026-09-11.md` (11/09) — la dette consignée de la vague 6 pesée en gain × effort ; tranchée par l'utilisateur le 13/09, décisions reprises dans `../PLAN_FINITIONS_2026-09-13.md`.
> - `chantiers/BILAN_FORK_CHASEWOODHAMS_2026-09-11.md` (11/09) — inventaire du fork, point par point contre notre arbre ; devenu `../PLAN_FORK_ET_RELEASE_2026-09-11.md`, dont les lots A à C sont fusionnés (journal du 12/09).
> - `chantiers/PLAN_AMIS_PAR_JOUEUR_ET_INVITATIONS_2026-09-15.md` (15/09) — amis par profil et invitations sans groupe ; étapes 0 à 7 closes, fusionné (`13c4b6c61`).
> - `chantiers/PLAN_ANNUAIRE_JOUEURS_2026-09-15.md` (15/09) — annuaire des joueurs, une clé d'identité (ADR 0035) ; clôturé au journal du 16/09, fusionné (`9ee1835c8`).
> - `chantiers/REVUE_ANNUAIRE_JOUEURS_2026-09-15.md` (15/09) — registre des revues du pilote et des deux rondes adversariales de l'annuaire (R1-R6, A1-A8, B1-B5), clos avec le plan.
> - `chantiers/PLAN_CORRECTIFS_REVUE_LOTS_V75_2026-09-17.md` (17/09) — les correctifs de la revue des huit lots ; fusionné (`ed04d6c0f`), CI verte.
> - `chantiers/PLAN_COULEURS_STATS_COMBAT_2026-09-17.md` (17/09) — jetons de couleur dédiés aux stats de combat, garde-rail anti-emprunt ; fusionné dans `feat/v75` (journal du 17/09).
> - `chantiers/PLAN_EXPLORER_RANGEE3_2026-09-17.md` (17/09) — troisième rangée de l'encart cible de l'Explorer ; cinq étapes closes, fusionné (`c8ee9b188`) ; suite dans `../PLAN_EXPLORER_PORTEE_FRAGS_2026-09-17.md`.
> - `chantiers/PLAN_LEADERBOARD_MONDE_REPRISE_2026-09-03.md` (03/09) — reprise du scrape du classement mondial ; lots 1 à 4 et revue clos, livré en hotfix 7.3.1 (`2751a484f`).
> - `chantiers/PLAN_MASTER_2026-09-09.md` (09/09) — pilotage des vagues 0 à 6 ; vague 6 close le 11/09 (CI 3/3) ; le reste consigné devient `chantiers/ARBITRAGE_DETTE_2026-09-11.md`.
> - `chantiers/PLAN_REPRISE_FORK_2026-09-05.md` (05/09) — reprise du fork en deux volets ; volet A fusionné sur `main` (`cf333a388`), volet B fermé sans code ; le soak restant est au `REGISTRE_REPORTS.md`.
> - `chantiers/PLAN_RETOURS_UTILISATEUR_2026-08-29.md` (29/08) — huit retours utilisateur, lots A à G ; G.3 plein fermé par DEC-8 ; la portée par arme a repris dans `../PLAN_DUELS_PORTEE_2026-09-06.md`.
> - `chantiers/PLAN_ROBUSTESSE_SYNC_2026-09-16.md` (16/09) — verdict fidèle du sync, rejeu du 429, plafond de slots ; fusionné (`3858eae59`).
> - `chantiers/PLAN_SYNC_POOL_SEAMS_SCHEMA_2026-09-16.md` (16/09) — sync par le pool pour tout profil suivi, dérive de schéma `match_registry` ; fusionné (`b4de1fc16`).
>
> `film_re/` :
>
> - `film_re/MESURE_ENTETE_TI9_47BITS_2026-09-11.md` (11/09) — l'en-tête d'un record d'image-clé est propre au type d'entité (`ti=9` : 47 bits) ; branche de mesure fusionnée (`f899967db`).
> - `film_re/PREPARATION_M3_3_2_A_3_4_2026-09-17.md` (17/09) — préparation des lots 3.2 à 3.4 du décodeur ; jalon M3 clos (`f9c2a5c74`).
> - `film_re/PREPARATION_M4_2026-09-17.md` (17/09) — préparation des lots 4.1 à 4.4 ; jalon M4 clos (`7df45f4f4`) ; son complément `../PREPARATION_M4_ORDRE_ET_FRONTIERES_2026-09-17.md` reste à la racine (cité par le plan du décodeur).
>
> `killweapon/` :
>
> - `killweapon/ETAT_DE_L_ART_CHANTIER_VOISIN.md` (27/07) — index de renvoi vers le worktree `filmdec-killweapon` ; sans objet depuis la réconciliation des deux lignées (`PLAN_RECONCILIATION_BRANCHES.md`, close le 31/07).
> - `killweapon/HANDOFF_PRECISION_PROJECTILES.md` (08/08) — déclaré « CONSOMMÉ » par son propre en-tête ; l'état fait foi dans `HANDOFF_PRECISION_PROJECTILES_2026-08-08.md` et `VERDICT_PRECISION_PROJECTILES.md`.
>
> `replay2d/` :
>
> - `replay2d/PLAN_EXPORT_FORMATS_VIDEO_2026-09-16.md` (16/09) — formats 1080p / 720p de l'export du rejeu ; gate visuel validé le 17/09, plan clos (`720c62a33`).
> - `replay2d/PLAN_RETOURS_VAGUE_A_LOT_COURT_2026-09-08.md` (08/09) — les sept correctifs courts des retours du 08/09 ; fusionnés (`e7926ded3`), item 0.2 du plan maître statué « reste = 0 ».
>
> Restés à la racine faute de preuve nette : la table « À ARBITRER » ci-dessus.

> **Passe d'hygiène du 2026-08-05 (lot H6).** Trois documents 100 % clos ont rejoint `V7.5/`
> et leurs liens croisés ont été mis à jour : `PLAN_RECONCILIATION_BRANCHES.md` (racine
> `V7.5/`), `replay2d/PLAN_REJEU_2D_FIABILISATION.md`, `killweapon/HANDOFF_KILLSOURCE_REPRISE.md`.
> **Trois candidats ont été REFUSÉS après vérification, et le refus est le résultat** :
> `HANDOFF_REPLAY_2D_2026-07-29.md` est la porte d'entrée déclarée du plan ACTIF
> `PLAN_FINALISATION_REJEU_2D.md` ; `HANDOFF_DUMPS_2026-07-31.md` et
> `SESSION_CAPTURE_AVANT_PC.md` portent 19 cases jamais statuées, donc rien ne dit qu'ils sont
> clos. `CONCEPTION_INVERSION_PRESEANCE.md` et `PLAN_BRANCHEMENT_KILLSOURCE.md` restent aussi à
> la racine : six fichiers Go citent leur CHEMIN en commentaire — ce sont des références de code
> vivant, pas des archives.

> **Passe d'hygiene du 2026-09-13 (item F.3 du `PLAN_FORK_ET_RELEASE_2026-09-11.md`, tache
> Notion 10).** 28 documents 100% clos (toutes cases `[x]`/`[~]`/`[!]`, ou cloture declaree
> par leur propre journal / le `thought_log`) ont rejoint `V7.5/` -- racine sauf mention :
>
> - `AUDIT_CUISSON_REPLAY_PERF_2026-09-02.md` -- audit source de `PLAN_CUISSON_PERF.md`, deja ici.
> - `AUDIT_V7.2.0_MAIN_2026-08-06.md` -- registre d'audit du diff v7.2.0->main, aucun item repris ailleurs.
> - `v2/AUDIT_V75_DEPUIS_V7.3.0_2026-09-05.md` -- audit source du chantier v2 rejeu/film, fusionne.
> - `ETAT_DE_L_ART_MODE_SCORE_EVENEMENTS.md` -- recherche mode->score, integree le 2026-08-05 (`feat/re-mode-score`).
> - `HANDOFF_CONTAINMENT_ZONES_2026-08-08.md` -- lot 4 v7.5, negatifs etablis (lettre de zone hors portee).
> - `HANDOFF_LECTURE_EQUIPEMENT_2026-09-04.md` -- chantier equipement du 03/09, "tout est merge" (dixit le handoff).
> - `HANDOFF_PRECISION_PROJECTILES_2026-08-08.md` -- piste close le jour meme (doublon date du `HANDOFF_PRECISION_PROJECTILES.md` reste a la racine ; deplace vers `killweapon/HANDOFF_PRECISION_PROJECTILES.md` le 2026-09-23).
> - `HANDOFF_SESSION_USAGE_BDD_2026-09-04.md` -- a nourri `PLAN_SESSION_USAGE_BDD_EXECUTION.md`, clos avec lui.
> - `HANDOFF_TACTIQUE_2026-09-07.md` -- chantier Tactique fusionne dans `feat/v75` le 07/09.
> - `v2/HANDOFF_V2_REJEU_FILM_2026-09-06.md` et `v2/HANDOFF_V2_REJEU_FILM_2026-09-07.md` -- etat du chantier v2, fusionne (schema 48).
> - `PLAN_FONDS_CARTE_WEBP_ETAG_2026-09-09.md` -- tous items `[~]` "fusionne le 09-09".
> - `PLAN_LECTURE_FIABLE_EQUIPEMENT_2026-09-03.md` -- clos, source du handoff equipement ci-dessus.
> - `PLAN_LEGENDES_COULEURS_RETOURS_2026-09-09.md` -- statut CLOS (E1->E7) declare en tete du fichier.
> - `PLAN_PERF_NOTE_OBJECTIFS.md` et `RAPPORT_SIM_PERF_NOTE_2026-08.md` -- note de perf, close le 28/08.
> - `PLAN_RETOURS_VAGUE_B_IDENTITES_2026-09-08.md` -- abandonne le 09/09, objectif atteint par une autre voie (sondage E2).
> - `PLAN_RETOURS_VAGUE_C_FORMES_2026-09-08.md` -- execute via `PLAN_MASTER_2026-09-09.md` (deplace vers `chantiers/PLAN_MASTER_2026-09-09.md` le 2026-09-23), "reste ouvert ici : rien".
> - `PLAN_SCORE_PAR_MANCHES.md` -- E0-E7 clos (ADR 0032), seul E2 partiel non bloquant.
> - `PLAN_SESSION_USAGE_BDD_EXECUTION.md` et `PLAN_SESSION_USAGE_LISIBILITE_2026-09-09.md` -- toutes etapes `[x]`, commits identifies.
> - `PLAN_TACTIQUE_2026-09-06.md` -- fusionne dans `feat/v75` le 07/09 (suite : `PLAN_TACTIQUE_SUITE_2026-09-07.md`, reste a la racine).
> - `v2/PLAN_V2_REJEU_FILM_2026-09-05.md` -- chantier v2 gele le 07/09 (suite : `PLAN_V2_RESTES_2026-09-07.md`, reste a la racine).
> - `RAPPORT_BTB_2025_ABSTENTION_2026-09-12.md`, `RAPPORT_LOT_B_DECODEUR_FORK_2026-09-11.md`,
>   `RAPPORT_LOT_BBIS_BIT_PROJECTILE_2026-09-12.md`, `RAPPORT_LOT_E_DECODEUR_SOUS_TITRE_2026-09-12.md`,
>   `RAPPORT_LOT_H_VERSIONS_2026-09-13.md` -- rapports des lots B/B-bis/E/G/H du `PLAN_FORK_ET_RELEASE_2026-09-11.md`,
>   tous fusionnes dans `feat/v75`.
>
> References croisees mises a jour dans les documents restes vivants (`REFERENCE_CANAUX_EQUIPEMENT_2026-09-09.md`,
> `HANDOFF_PRECISION_PROJECTILES.md` (deplace vers `killweapon/HANDOFF_PRECISION_PROJECTILES.md` le 2026-09-23), `PLAN_EQUIPEMENT_GACHIS_2026-09-09.md`, `DECOUVERTES_TACTIQUE_2026-09-07.md`,
> `PLAN_ORCHESTRATION_2026-09-07.md`, `PLAN_V2_RESTES_2026-09-07.md`, `docs/COMMANDS.md`, `docs/FR/COMMANDS.md`) ;
> les references internes aux documents deja archives (V7.5 ou eux-memes deplaces dans cette meme passe)
> ne sont pas reecrites (convention posee par ce README au H6 ci-dessus).
> **Decouverte non traitee** : `.ai/AUDIT_V75_DEPUIS_V7.3.0_2026-09-05_annexes/` (28 fichiers,
> verifications par worker) est un sous-dossier a la racine de `.ai/`, hors du perimetre "fichiers
> `.ai/*.md`" de cette passe -- non deplace ; ses trois references au chemin racine de l'audit
> (`G10.md`, `V-GO-B1.md`, `V-GO-B2.md`) et le renvoi de `V-WEB-3a.md` vers le handoff usage BDD
> pointent donc desormais vers un fichier qui vit sous `V7.5/`.
> **Gardes a la racine malgre le doute** (justification ecrite au lieu d'un deplacement) :
> `AUDIT_LECTEURS_VIES_ANONYMES_2026-09-06.md` (registre encore cite par les restes v2 ouverts),
> `ETAT_DE_L_ART_FORGE_PALETTE_ZONES.md` (chantier Forge pas encore ouvert, post-release),
> `HANDOFF_ASSAUT_DESAMORCAGE_2026-09-04.md` (piste ouverte non bloquante, decision utilisateur
> en attente), `HANDOFF_VEHICULES_2026-09-04.md` (branche jamais poussee, CI jamais executee),
> `PLAN_ESCOUADE_HORS_CADRE_2026-09-09.md` (items R1-R4 de cloture de vague non statues),
> `PLAN_ORCHESTRATION_2026-09-07.md` (table de suivi encore ouverte sur P3-P5/L6-L8/R9),
> `STEAKTACULAR.md` et `duckdb_7659_upstream_report.md` (hors du perimetre thematique de `V7.5`,
> qui ne couvre que la recherche rejeu/film ; le second attend une reponse amont externe).

## Pièges

- **Les journaux n'ont pas été réécrits.** `thought_log.md` et `replay2d/thought_log_replay.md`
  citent les anciens chemins plats (de la forme `.ai/<NOM>.md`, `.ai/re_dump/...`) pour les
  entrées antérieures au 2026-07-31 : ce sont des archives, on ne réécrit pas l'histoire.
  Pour retrouver un document cité dans une vieille entrée, chercher son nom, pas son chemin.
- **`dumps/` est lu par du code**, pas seulement cité : `cmd/replay-build` (défaut de
  `-geometry`), plusieurs `cmd/tmp_*`, et `internal/analysis/replay/mapvar` (test). Déplacer
  ce dossier oblige à corriger ces chemins.
- Les `cmd/tmp_*` qui pointent en absolu vers `.claude/worktrees/filmdec-continuation/.ai/re_dump/`
  n'ont pas été touchés : ce worktree conserve son ancienne arborescence.

Handoff superviseur du 2026-08-18 (soir) — etat exact, livre depuis le 15/08, en attente
utilisateur, regles de pilotage, ordre de reprise : `HANDOFF_SUPERVISEUR_v75_2026-08-18.md`.
Plan de l'item 6 (armes au sol / socles / power-ups / ramassage, VALIDE le 2026-08-17, en
execution) : `replay2d/PLAN_ARMES_AU_SOL_2E_LECTURE.md`. **Handoff superviseur du 2026-08-20 (le plus recent) : `HANDOFF_SUPERVISEUR_v75_2026-08-20.md`.** Plan de l'item 4 (objectifs vivants,
deuxieme lecture : porteur avant objet, colline par periode ; en attente de l'item 6 et des
fusions utilisateur) : `replay2d/PLAN_OBJECTIFS_VIVANTS_2E_LECTURE.md`.
- `HANDOFF_SUPERVISEUR_REGISTRE_FILM_2026-08-18.md` — handoff du pilotage « exploitation du Registre du film Theater » (score dans le temps, elevation, ti=13 etat des zones : lot C-bis phase 2b en revue, schema 16).
- `PLAN_REMEDIATION_CACHE.md` (2026-08-25, branche `wt/remediation-cache`) — **remediation du cache
  d'artefacts de rejeu APPAUVRIS** : le mode `--repair-impoverished` de `levelup backfill-replay`.
  Ferme la dette « le cache DEJA empoisonne » de `PLAN_OUVRIER_DISTANT.md` §5ter, qui devait etre
  ouverte AVANT la premiere activation prod de l'ouvrier. Planification pure du parent (aucun second
  chemin de cuisson) : re-cuit l'artefact au schema courant, sans compteurs de joueur, dont la base a
  des lignes — meme predicat que `replayartifacts.etatArtefact`. Temoin dry-run du 2026-08-25 : 2
  reparables sur 951 films du cache. La commande exacte de remediation est au §5ter du plan ouvrier.
- `replay2d/TI11_GRAMMAIRE_34_CHAMPS.md` (2026-09-13) — grammaire de l'entité ti=11 (descripteur d'objectif du HUD) conservée comme asset de recherche après suppression de la branche ; code figé sous le tag `archive/ti11-cadre`.
