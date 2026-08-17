# V7.5 — recherche et rétro-ingénierie (rejeu 2D, événements de frags/morts)

Ce dossier rassemble les documents de **recherche et de rétro-ingénierie** produits dans les
deux worktrees `feat/filmdec-*` (décodage du film Theater, arme/source de dégât par kill) et
`feat/replay2d-*` (rejeu 2D), réunis sur `feat/replay2d-prod` le 2026-07-31.

Ce sont des **archives de chantier** : elles font foi sur ce qui a été prouvé, mesuré ou
réfuté. Ce qui reste à faire ou à terminer n'est pas ici mais à la racine de `.ai/`
(voir « Ce qui est resté à la racine » plus bas).

## Organisation

| Dossier | Contenu | Fichiers |
|---|---|---|
| `film_re/` | Format du film Theater : grammaire, chunks, décodeur ECS, keyframes, RE Ghidra, handoffs externes | 25 |
| `killweapon/` | Arme / source de dégât par kill : kill feed, dead-state, same-clock, walk biped, journal RE | 22 |
| `replay2d/` | POC du rejeu 2D puis ses chantiers : trajectoires, inventaire/loadout, vérité terrain, plans de lots | 42 |
| `cartes/` | Géométrie 2D des maps depuis les `.module`, triangles, noms de zones | 3 |
| `icones/` | Icônes d'armes et du **kill feed** extraites des `.module` : chaîne, tables de correspondance, page de nommage, planches-contact | 5 |
| `dumps/` | Captures binaires, CSV, PNG (ex-`.ai/re_dump/`) — 69 Mo, lus par du code | 40 entrées |

Plan actif d'habillage du rejeu 2D (marqueurs, noms, amis, logo, rangee `fil | carte | fiches`) :
`replay2d/PLAN_HABILLAGE_REJEU_2D.md` (ecrit le 2026-08-16, decisions D1-D8 a valider par le user).

Quatre lots de RECHERCHE du 2026-08-17 (branches `wt/ti37-identite`, `wt/ti11-objectifs`,
`wt/kf-grammaire`, `wt/kf-file-entite`, fusion triee sur `wt/fusion-lots-go`) — leurs plans
portent le detail mesure, et trois d'entre eux sont des NEGATIFS qui ferment des voies :

- `replay2d/PLAN_R3_IDENTITE_TI37.md` — l'identite de l'objet `ti=37` est le GlobalID d'un tag
  `eqip` (428 occurrences sur 428, zero ailleurs). Confirmation INDEPENDANTE de la meme
  decouverte que le lot de production `PLAN_IDENTITE_TI37.md` ; c'est ce dernier qui a livre
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
`../PLAN_DETTE_AVANT_MERGE.md`.

### Points d'entrée par sujet

- **Arme par kill** : `../README_KILLWEAPON_INDEX.md` (index maître, à greper en premier),
  puis `killweapon/RE_LOG_KILLWEAPON.md` (journal, ne jamais le lire par le haut).
- **Grammaire ECS (archétype × composant)** : la table de référence est
  `apps/go-api/internal/analysis/filmdec/testdata/ecs_table.tsv` (1 067 couples du registre du
  film + 14 alias ; statut de portage, niveau, source `fichier:ligne`, deser, champ du document),
  tenue par les garde-rails G1-G3 de `filmdec/ecs_table_guard_test.go`. Le plan qui l’a produite
  et la vérification de l’inventaire qui l’a précédée : `film_re/PLAN_TABLE_ECS.md`.
- **Format du film** : `film_re/GRAMMAIRE_RECORD_FILM.md` puis
  `film_re/RECETTE_DECODAGE_FILM_CHUNKS.md`.
- **Reverse externe / handoff** : `film_re/HANDOFF_FILM_EXTRACTION_EXTERNAL_DEV.md`,
  `film_re/GITHUB_RE_FINDINGS_EN.md` (EN).
- **Cartes** : `cartes/HANDOFF_MAP_GEOMETRY_FROM_MODULES.md`.
- **Icônes (armes, véhicules, kill feed)** : `icones/ETAT_DE_L_ART_ICONES.md` — chaîne complète,
  tables index → arme/nom, pistes réfutées. Le nommage restant se fait dans
  `icones/NOMMAGE_ICONES.html` (page locale, hors app).

## Ce qui est resté à la racine de `.ai/`

Les documents **encore vivants** : les états de l'art (référence courante), l'index maître
arme-par-kill, et les plans/handoffs qui restent à traiter ou à terminer.

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

État de l'art et index : `ETAT_DE_L_ART_KILLWEAPON.md`,
`ADDENDUM_ETAT_DE_L_ART_2026-07-26.md`, `ETAT_DE_L_ART_CHANTIER_VOISIN.md`,
`README_KILLWEAPON_INDEX.md`.

Dette et merge : `PLAN_DETTE_AVANT_MERGE.md`, `PLAN_MASTER_FILM_KILLFEED_REJEU.md`
(document d'autorité), `HANDOFF_SUPERVISEUR_2026-08-03.md` (point d'entrée du rôle superviseur).

Rejeu 2D : `PLAN_FINALISATION_REJEU_2D.md`, `SUIVI_REPLAY_2D.md`, `ETAT_DU_POC.md`,
`CAHIER_DES_CHARGES_POC.md`, `HANDOFF_REPLAY_2D_2026-07-29.md`,
`PLAN_OBJECTIFS_TEMPS_REEL.md`, `PLAN_CAPACITES_ACTIVES.md`,
`PLAN_RECHERCHE_ASSETS_ICONES.md`, `PLAN_BELLE_CARTE_TRIANGLES.md`, `CLE_USB_REJEU_2D.md`.

Killsource / armes : `PLAN_BRANCHEMENT_KILLSOURCE.md`, `CONCEPTION_INVERSION_PRESEANCE.md`,
`GUIDE_WEAPON_SHOTS.md`, `HANDOFF_PRECISION_PROJECTILES.md`, `PLAN_VARIABLES_JETEES.md`.

Captures et dumps : `HANDOFF_DUMPS_2026-07-31.md`, `SESSION_CAPTURE_AVANT_PC.md`.

Journaux : `thought_log.md` (principal) et `thought_log_replay.md` (chantier rejeu 2D).

## Pièges

- **Les journaux n'ont pas été réécrits.** `thought_log.md` et `thought_log_replay.md`
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
execution) : `replay2d/PLAN_ARMES_AU_SOL_2E_LECTURE.md`. Plan de l'item 4 (objectifs vivants,
deuxieme lecture : porteur avant objet, colline par periode ; en attente de l'item 6 et des
fusions utilisateur) : `replay2d/PLAN_OBJECTIFS_VIVANTS_2E_LECTURE.md`.
