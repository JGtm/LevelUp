# Handoff superviseur — rejeu 2D v7.5 : habillage, revue des poses, RE image-clé (R3 → R7-e)

> Ecrit le 2026-08-18 par la session superviseur (pilotage d'executeurs Opus en worktrees).
> Objet : reprendre exactement ou cette session s'arrete. Lire EN PREMIER, puis
> `REGISTRE_REPORTS.md`, puis les plans cites. Rien de ce fichier n'est une decision nouvelle :
> tout renvoie a une piece.

## 0. Ou en est le depot au moment du handoff

- Branche d'integration : `feat/v75` (mode branche unique, CI verte AU NIVEAU JOB = cloture, un seul
  merge final vers `main` + tag v7.5.0 — hors de ce handoff).
- **Fusion finale en cours** dans le worktree `LevelUp-wt-fusion-finale` (branche `wt/fusion-finale`,
  base `feat/v75` = `5ab07d67b`) : un executeur fusionne dans l'ordre `wt/fusion-lots-go` →
  `wt/kf-boucle-etat-complet` → `wt/kf-encodage-drapeau` → `wt/poses-revue-fix`, avec A/B gradient
  sur la correction i9, docs remises en etat final, gates Go + web. **Le superviseur a le GO user
  pour fusionner `wt/fusion-finale` dans `feat/v75` et pousser** (message user du 2026-08-18 :
  « Tu peux fusionner du coup »). Si cette session s'arrete avant : la reprise fait la revue de
  `wt/fusion-finale` (gates rejoues : `tsc -b --force`, `vitest src/features/match-replay`,
  `go test filmdec+replay`), puis `git merge --no-ff wt/fusion-finale` dans le principal sur
  `feat/v75` (verifier `git status` propre et `feat/v75` immobile depuis `5ab07d67b`, sinon
  realigner d'abord), `git push origin feat/v75`, puis CI verte au niveau job.
- **Deja fusionne dans feat/v75 par cette session** : l'habillage du rejeu 2D (`8f33ffe19` + statut
  `18c14740e`) — noms sous les points, style de la planche, amis/moi, logo, rangee
  `fil | carte | fiches`. Plan : `replay2d/PLAN_HABILLAGE_REJEU_2D.md` (§7 = journal superviseur).
- **Gate visuel USER OUVERT** (decision user : valider apres fusion) sur le vite du principal
  `http://localhost:5173` : habillage (temoins listes dans PLAN_HABILLAGE §7 et thought_log 17/08)
  + poses (apres fusion : fenetre d'affichage — capteur 15 s officiel, autres poses jusqu'a la fin
  du rejeu, F11 les deux murs a 0,13 m sur `000d5950` ~3:55, ligne d'assistance a DEUX
  pictogrammes vignette + glyphe ami/moi).

## 1. Les branches de cette session (toutes POUSSEES sur origin)

| branche | contenu | plan | verdict |
|---|---|---|---|
| `wt/habillage-rejeu` | habillage (phases 0-6), 2 realignements | `replay2d/PLAN_HABILLAGE_REJEU_2D.md` | FUSIONNE feat/v75 |
| `wt/ti37-identite` (R3) | identite ti=37 = GlobalID tag `eqip` (428/428) ; mur/capteur non nommes | `replay2d/PLAN_R3_IDENTITE_TI37.md` | clos ; CODE ECARTE a la fusion (doublon du R3 bis de l'autre session), docs gardees |
| `wt/ti11-objectifs` (R4) | ti=11 = descripteur HUD sans position (negatif mesure) | `replay2d/PLAN_R4_OBJECTIFS_VIVANTS_TI11.md` | clos ; code + docs fusionnes |
| `wt/kf-grammaire` (R5) | corps d'image-cle ≠ record NEW ; ti=42 default-state porte (DEBRANCHE a la fusion : sans oracle) | `replay2d/PLAN_R5_GRAMMAIRE_IMAGE_CLE.md` | clos negatif ; instruments + docs |
| `wt/kf-file-entite` (R6) | la file par entite n'est pas une transformation ; le jeu ne relit JAMAIS le type 2 ; buffers live de juillet = premiers paquets delta | `replay2d/PLAN_R6_FILE_PAR_ENTITE.md` | clos negatif ; instruments + docs |
| `wt/fusion-lots-go` | R3-R6 TRIES (docs de tout, code R4 + instruments R5/R6, pas R3, ti=42 debranche), constante `keyframeRecordTIBit` centralisee | — | pret, base 085cda41b |
| `wt/kf-biped-etat-complet` (R7-a) | image-cle = ETAT COMPLET (102-104 % de la longueur), 0,51 % bit-exact | `replay2d/PLAN_R7A_IMAGE_CLE_BIPEDE_ETAT_COMPLET.md` | clos |
| `wt/kf-biped-bit-exact` (R7-b) | **faute de PROD corrigee : polarite i9 `consumeObjectMultiplayerProperties`** (bloc TLV present quand bit==0) ; i60 grammaire, i57 tag3 ; 0,54 % ; residu = DERIVE sur 64 composants | `replay2d/PLAN_R7B_BIPEDE_IMAGE_CLE_BIT_EXACT.md` | clos |
| `wt/kf-encodage-drapeau` (R7-c) | drapeau d'encodage : 2 drapeaux reels (`DAT_144e61ea0` portee, `DAT_145121140`), porte OFF, degrade la lecture ; position payload QUANTIFIEE aux largeurs de carte ; **lecteur d'etat complet du jeu NOMME** `FUN_1428e2a04→FUN_142e2bfd0` ; `FUN_1411b259c`=R(96) | `replay2d/PLAN_R7C_ENCODAGE_DRAPEAU_IMAGE_CLE.md` | clos negatif |
| `wt/kf-ecrivain-vtable` (R7-d) | ecrivain = `vtable+0x18` (52/52, methode dump `.rdata` + pointeurs) ; boucle d'etat complet `FUN_142e2c690` SANS masque, table fixe par archetype ; i0 port faux (2 bits inconditionnels en dernier) ; 0,85 % | `replay2d/PLAN_R7D_ECRIVAIN_VTABLE.md` | clos |
| `wt/kf-boucle-etat-complet` (R7-e) | boucle portee telle quelle : en-tete d'entite 108 bits, ordre = index de table, niveaux du registre decales d'un cran (25/64), controle R(32) inconditionnel, portee tranchee ; **0,51 %, borne d'arret atteinte** | `replay2d/PLAN_R7E_BOUCLE_ETAT_COMPLET.md` | clos negatif — **ARRET DE LA RE IMAGE-CLE (decision user)** |
| `wt/poses-revue-fix` | correctif de la revue adversariale du lot poses de l'autre session : t1 = borne inferieure (fin non datable — negatif mesure), calque n'efface plus a t1, garde-rail 4 enumerations de familles, MPPLeadBits retire, ReplayCanvas 952→860 L + cliquet, `coverage.placements.scanned`, 4 mineurs au registre | `replay2d/PLAN_CORRECTIF_REVUE_POSES.md` | pret, base 0ae609b2c |

Le worktree `LevelUp-wt-review` (detache sur `085cda41b`) a servi a la revue adversariale du lot
poses (rapport dans le transcript de la session ; findings F1-F11 recopies dans le plan correctif
et le registre). A supprimer (`git worktree remove`) une fois la fusion faite, comme tous les
`LevelUp-wt-kf*`, `LevelUp-wt-ti*`, `LevelUp-wt-poses-fix`, `LevelUp-wt-fusion-go`,
`LevelUp-wt-habillage` (branches conservees sur origin).

## 2. Ce que la serie R3 → R7-e a etabli (a ne PAS re-mesurer)

1. Identite ti=37 = tag `eqip` (deux chaines independantes : R3 428/428, R3 bis 21/21 sonde jeu).
2. ti=11 = descripteur HUD, aucun composant ne porte de position ; l'objet physique est designe par i3.
3. Le lecteur de film du jeu **ne relit jamais** le payload type-2 (aiguillage `FUN_1428e22c0` sans
   handler, `FUN_142989418` le saute) ; les buffers live de juillet (`keyframe_buffer_live.bin`,
   `kf_slot0_live.bin`) sont des PREMIERS PAQUETS DELTA (949/951 films en prefixe), pas l'image-cle.
4. Le corps d'un record d'image-cle = ETAT COMPLET, composant par composant, sans masque (boucle
   `FUN_142e2c690`, ordre = index de table, en-tete d'entite 108 bits) ; la position y est
   QUANTIFIEE aux largeurs de carte (payload ecrit HORS de la portee `DAT_144e61ea0`).
5. Bit-exact NON atteint (0,51-0,85 % sur 591 records ti=35, 3 films) : le residu est une DERIVE
   dispersee sur les 64 deser (dispersion sans palier), pas un bloc manquant ni un deser casse ; le
   cadre (en-tete, ordre, niveaux, controle, portee) est innocente. Voies fermees : largeurs de
   carte, corruption-check (artefact de la sur-lecture d'i9), drapeau baseline, bloc manquant.
6. Acquis durables : ecrivain par vtable+0x18 (methode dump `.rdata`), lecteur d'etat complet
   nomme, i9 corrige en production, i60 grammaire etablie (`simStateComplete` reporte : queue aux
   largeurs d'axe reelles), niveaux du registre decales d'un cran (`registry.go` sert `Flags[k]`,
   le jeu lit `Flags[k+1]` — inerte sur le bipede, a verifier sur d'autres archetypes).
7. Le levier jamais consomme : `kf_capture_sample.txt` (400 frontieres exactes, 266 NEW + 134
   DELTA, sur un paquet DELTA) = oracle de largeur d'etat par defaut par archetype.

## 3. Reports et decisions user (le registre fait foi)

- Armes de pouvoir / power-ups : emplacements de spawn + minuteries (statique, racks NON extraits)
  et ramassage (dynamique : evenements de cycle de vie non decodes ; ti=42 bloque) — **A FAIRE,
  user : « pas maintenant, on reste sur ton perimetre »**.
- RE image-cle : ARRETEE apres R7-e (borne d'arret). Conditions de reprise au registre (lignes R7-*).
- Re-cuisson de masse du schema 9 : 3 temoins re-cuits seulement — decision ops (fenetre), lot de
  l'autre session.
- Deux sessions ont travaille en parallele sur ti=37 (decision user « deux points de vue ») : le R3
  bis de l'autre session est la PRODUCTION (poses + nommage + UI + capteur officiel + familles) ; ce
  handoff n'y touche pas au-dela du correctif de revue.

## 4. Methode et pieges (a relire avant de piloter)

- Le worktree PRINCIPAL est PARTAGE entre agents : tout lot = worktree dedie `wt/<lot>` depuis
  `feat/v75`, fusion par le superviseur ; « ok pour le plan » ≠ go d'execution ; le superviseur
  PILOTE (executeurs Opus), ne code pas (memoire `feedback_worktree_dedie_et_ok_plan_pas_go`).
- Chaque executeur Go : `GOCACHE` propre au worktree, UNE commande `go` a la fois, `CGO_ENABLED=0`
  pour filmdec/replay ; `go build ./...` exige CGO (gcc winlibs).
- Ghidra : instance user (PID 10104) ; le pont MCP `mcp__ghidra__*` refuse la connexion (UDS
  « unknown ») — contournement = API HTTP du plugin `http://127.0.0.1:8089`, seul
  `GET /decompile_function?address=0x...` est fiable ; `/disassemble_function` sur une adresse sans
  fonction rend 200 Mo ; le `HaloInfinite.exe` du disque est un STUB (le vrai `.text` est dans
  l'instance). Aucun rename/script/analyse (programme du user).
- Coupures API (529, reponse interrompue) : relancer l'agent par SendMessage sur son id (contexte
  conserve), lui faire commiter ses acquis d'abord.
- vitest : suite complete non deterministe sous charge (timeouts 5 s isoles : `langSegmentInheritance`,
  `PalmaresRelationsPage`, `calendar.guard`) — verts relances seuls ; la CI Linux reste l'autorite.
- Toujours `--no-verify` INTERDIT (deux executeurs l'ont fait une fois, corrige) ; hooks pre-push
  (govulncheck ~2 min) → timeout long.

## 5. Artefact pour le user

`Registre du film Theater` — https://claude.ai/code/artifact/95a814f5-6957-4b9d-8a55-87a2e5547409
(inventaire 33 archetypes / 216 composants / 16 flux, top des exploitations non faites : courbe de
score statborg, vitesse par image, chrono officiel, etat de mouvement, second axe de visee,
canaux de l'equipement, zones de mode, marqueurs positionnes, objectifs vivants, decor mobile).
Donnees source : scratchpad de la session (`ecs_inventaire.json`), compilees en lecture seule.

## 6. Reprise en 5 lignes

1. `git -C C:\Users\Guillaume\Projects\LevelUp-wt-fusion-finale log --oneline -3` : la fusion
   finale est-elle poussee (`origin/wt/fusion-finale`) ? Sinon relancer/terminer l'executeur.
2. Revue sur pieces (gates rejoues) → `merge --no-ff` dans `feat/v75` → push → CI verte au niveau job.
3. Gate visuel USER (habillage + poses) sur :5173 ; corrections eventuelles en `wt/*`.
4. Nettoyage des worktrees ; memoire `project_v75_etat_courant` a jour.
5. Puis cloture v7.5 globale (autre session / user) : tag v7.5.0, merge unique vers `main`.
