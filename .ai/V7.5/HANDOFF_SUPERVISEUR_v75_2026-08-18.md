# HANDOFF superviseur — pipeline v7.5, rejeu 2D — 2026-08-18 (soir)

> Ecrit par la session de pilotage (Opus 5) a l'approche de la fin de son contexte. Point d'entree
> pour la session qui reprend le PILOTAGE (pas l'execution : chaque lot = un plan ecrit AVANT, un
> agent Opus/Sonnet, verification SUR PIECES du CR avant tout relais a l'utilisateur, fusion des
> worktrees freres par le superviseur avec gates rejoues). Lire aussi : `README.md` de ce
> dossier, `REGISTRE_REPORTS.md` (le registre est la memoire des reports), et
> `.ai/thought_log.md` (entrees en TETE de fichier, les plus recentes d'abord).

## 0. Etat exact au moment du handoff

- Branche unique `feat/v75`, HEAD **`5ab07d67b`**, worktree principal `C:\Users\Guillaume\Projects\LevelUp`.
  **Rien n'est pousse depuis des jours** (mode branche unique : lots = commits, CI verte au niveau
  job a la cloture, UN merge final + tag v7.5.0). La CI n'a donc vu aucun de ces commits.
- Arbre principal : propre HORS trois fichiers non suivis — `.ai/AUDIT_V7.2.0_MAIN_2026-08-06.md`
  et `apps/go-api/internal/himap/sonde_bouillie_gamefiles_test.go` (autres sessions, NE PAS
  committer) et **`.ai/V7.5/replay2d/PLAN_ARMES_AU_SOL_2E_LECTURE.md`** (plan de l'item 6, ecrit
  par cette session, **a committer avec le prochain lot de docs** — il est deja mis a jour avec les
  apports du worktree concurrent, voir §3).
- Aucun agent de fond ne tourne. Aucune fusion en cours.
- Worktrees freres de CETTE session : tous supprimes (effets, drawer, heatmap, sons, retours,
  uiposes, capteur, familles). Les autres worktrees listes par `git worktree list`
  (`wt-habillage`, `wt-fusion-go`, `wt-fusion-finale`, `wt-kf*`, `wt-ti11`, `wt-ti37`,
  `wt-poses-fix`, `wt-review`, `wt-sons-fusion`, `wt-replay2d`, `wt-admin-retours`) appartiennent a
  l'UTILISATEUR ou a ses sessions paralleles : **ne pas y toucher, ne pas les fusionner sans son
  accord** (voir §3 pour `wt/fusion-lots-go`, dont la fusion « revient au superviseur »).

## 1. Ce qui a ete livre sur `feat/v75` depuis le 15/08 (tout verifie sur pieces, tout journalise)

Dans l'ordre, avec le SHA de tete de chaque lot :

| lot | SHA | ce que c'est |
|---|---|---|
| Sons explosions x4 + melee fatale | `f0d712103` | jointure par la VIGNETTE (killicon), pas par weapon_key |
| Mesure equipement `ti=37` (OU oui, QUAND/QUI non) | `d4be4ab95` | + decouverte du defaut de precision |
| **Precision des objets du monde** (largeurs = catalogue `MapQuantEntry`, `Options.MapQuant`) | `4e2084d8e`, `ea9ae50d8` | Bazaar 0,09 % -> 99,4 % ; garde-rails vus rouges |
| Type de grenade par le lancer : NO-GO mesure (0,14 % de cible, temoin ne s'effondre pas) | `69c625e29` | sujet CLOS par l'utilisateur |
| Etat actif : camo (i28 queue[1]) et surbouclier (i5 q>64) SE LISENT ; i54/i57 refutes ; i59 tag==3 = grappin | `26979b2f1`..`5e22f48fa` | |
| Camo + surbouclier publies (schema 7) + effet fiche + 4 sons | `b2c1e6bcb`, `d257ba02f`, `c7096aedb` | |
| **Grappin** : ancre i59 portee et PROUVEE, schema 8, ligne blanche | `020b95eab`, `b52cf07ed`, `040e0d56a` | |
| Fusion effets fiches (verre + or, token `legendary`) + tiroir + filtres sons ; garde-rail frags REPARE (valeur, pas seuil) ; golden minibobine ratifie | `7db99868c`..`1630db8ec` | |
| Alertes : killsource — l'ecart NE MORD PAS (0 ligne changee, sonde d'absurde) ; catalogue de bornes 56 -> 64 | `efec87d71`, `317175a4c` | killsource NON modifie |
| **Bornes des canevas Forge** : critere structurel = REGION 0 du `levl` ; 21 entrees corrigees + 14 ajoutees (78) ; 342 films controles ; ecart vertical containment 1 300 m -> 0,1 m | `64e1a8a0d`, `3f02079cd`, `10c1ff019` | Threshold construite |
| Sons d'armes extraits (branche utilisateur `feat/v75-sons-fusion`, avance-rapide 23:15 le 16/08) | `d7fbf6b93` | |
| **Callouts decoupes** sur le masque publie (alpha du fond) : IoU 0,872, enclaves reprises, tolerance 4 m etalonnee ; 19 decoupe / 3 brut | `755729d00` | |
| Heatmap (presence / eliminations, grille 0,5 m, quantiles) | `e3411d8c7` (fusion) | mode « jusqu'au curseur » non livre (mesure) |
| Sons : duree = propriete de la categorie (`SOUND_CUT_MAX_S` 4 s), Dynamo depuis « Full » a l'attaque mesuree, repulseur = REFUS mesure (aucune source de degat) | `022a7dbbd` (fusion) | |
| Retours de la planche R1 (tiroir overlay, bascules effets tirs ON / morts OFF, explosions 2,4 s, reapparition 1,2 s + texte, grenades en images, glyphe capacite, callouts 9,5 px) | `1eb25d5fd` (fusion) | |
| Icone d'assistance = `killfeed-62` (designee par l'utilisateur) | `d635a96b6` | |
| Habillage (session utilisateur, fusionne par elle) | `18c14740e` | |
| **Identite `ti=37`** = GlobalID `eqip` dans le mot 32 bits du bloc MPP du record de creation ; double chaine mur 90 % / capteur 92 % | `193b3de5a`, `5e7634669` | |
| **Poses publiees (schema 9)** : largeur = DOUBLE champ (9/5 QP, 8/3 BTB), calibration par oracle ; manifeste `[[equipment_objects]]` | `160e4ea7b`..`626c6f90d` | |
| UI poses : mur = ARC concave vers le poseur, capteur, `other` sur bascule, sons pose | `54bc47ca3` (fusion) | |
| **Capteur officiel** (Waypoint saison 4 : 4,25 m, ping 1,8 s, revelation 0,75 s) : onde de ping + marque « revele » | `0ae609b2c` (fusion) | vie mesuree 2,1 s mediane |
| **Nommage structurel** `sofd -> sofa -> {string_id, eqip}` : 20/21 identifiants nommes, 15 familles ; translocateur : balise nommee, retour NEGATIF | `77d77d2cf` | |
| **Origine des poses** : `dropped` (laches a la MORT du porteur, 88,6 %) vs `deployed` ; l'arc du mur va sur les PANNEAUX ; schema 10 | `7a0b672dc`, `1ad453e65` | |
| archlint J4R-3 faux positif leve (jeton « scan » retire du dictionnaire) | `065f11ef0` | |
| Phase W : rendu par famille, filtre `deployed`, panneaux, balise/seeker/champ, son partage le predicat | `6f42332c8` (fusion) | 19 formes au lieu de 47 sur le temoin |
| Journal/registre | `9338786f7`, `5ab07d67b` | |

## 2. Ce qui est EN ATTENTE de l'utilisateur (ne pas relancer sans lui)

- **Gates visuels / d'ecoute** — la planche de validation (artefact
  `https://claude.ai/code/artifact/1b08c056-33bc-4563-96ee-7eeb52946979`, verdicts persistes
  dans la page, bouton « Copier le bilan ») a ete jugee une fois (bilan du 16/08, verbatim en
  annexe de `PLAN_RETOURS_PLANCHE_2026-08-16.md`). Depuis, a re-juger : mur (il a dit « je
  validerai plus tard »), capteur ping/revelation, tiroir overlay, callouts decoupes/police, heatmap,
  sons re-coupes (frag re-coupee a 3,3 s alors qu'il l'avait validee a 1,2 s — a confirmer ou
  annuler), poses (filtre deployed, panneaux, balise/seeker/champ sans temoin visuel), habillage.
  **La planche n'a PAS ete remise a jour depuis la nappe Dynamo** : une mise a jour (rendus,
  nouveaux sons, familles, capteur) est due — sources : `scratchpad/effets.template.html`,
  `app.js`, bundle `fxbundle/replayfx.iife.js` (rolldown des modules canvas), `icons.b64`,
  `sounds.b64` (Opus 48 kb/s), assembleur node — tout dans le scratchpad de la session
  precedente ; a reconstruire si le scratchpad a disparu (la recette est dans le journal du 16/08).
- **Item 6 — armes au sol / socles / power-ups / ramassage** : plan ecrit et precise
  (`PLAN_ARMES_AU_SOL_2E_LECTURE.md`), **A VALIDER PAR L'UTILISATEUR AVANT TOUT LANCEMENT**
  (sa consigne explicite). Deux questions ouvertes pour lui : films a power-ups de socle a
  designer ; source officielle des cycles de reapparition.
- **Fusion de `wt/fusion-lots-go`** (lots R3-R6 de sa session parallele : docs + instruments,
  base `085cda41b`, 20 commits ; conflits limites a `REGISTRE_REPORTS.md` et `thought_log.md`,
  a resoudre par UNION ; `git merge --no-commit` a blanc deja teste, arbre restaure). La synthese
  de cette branche dit « la fusion dans feat/v75 revient au superviseur » — demander son go, puis
  fusionner (agent Sonnet, procedure `FUSION_WT_2026-08-16.md`), gates Go, journal.
- **Fenetre ops** (serveur arrete, ~8 h) : re-build de MASSE des ~949 artefacts (schemas 7->10,
  bornes corrigees — 3 artefacts locaux encore faux : Domicile, Fortitude Heavies `084a804d`,
  Snowbound) + `backfill-killsource`. Prealable (bornes Forge) LEVE. A caler avec lui.
- **Refactor `ReplayCanvas.tsx`** (938 lignes, 5 copies du patron de cuisson de calque, 6 copies du
  cadrage) : au registre, a planifier quand la vague retombe.
- **Objectifs vivants (item 4)** : plan a ecrire. Piste corrigee par R4 : `ti=11` est le
  DESCRIPTEUR d'objectif du HUD, pas l'objet ; le drapeau et le crane sont des ARMES portees — le
  porteur se lit la ou on lit « l'arme en main » (i43-i46 / loadouts d'images-cles) ; le verrou
  commun a tout le reste = l'etat par defaut par archetype, bit-exact ; oracle disponible :
  `.ai/V7.5/dumps/kf_capture_sample.txt` (400 frontieres exactes) + `kf_slot0_live.bin`.

## 3. Regles de pilotage que cette session a fixees (et payees)

1. **Un plan ecrit AVANT chaque lot** (`.ai/V7.5/replay2d/PLAN_*.md`), passe a `plan-review`,
   avec : acquis (ne pas re-mesurer), refutes (ne pas rejouer), decisions tranchees, phases a
   cases, gates d'arret REELS (« si X echoue, le plan s'arrete et le negatif s'ecrit »), seuils
   ecrits AVANT mesure et jamais rebaisses.
2. **Un agent par worktree.** Donnees (films, `data/`, jeu installe) = worktree PRINCIPAL ;
   web pur = worktree FRERE `../LevelUp-wt-<nom>` (branche `wt/<nom>`), cree par
   `git worktree add` (l'isolation du harnais REFUSE sur ce depot). Les agents des worktrees
   freres NE touchent PAS au journal ni au registre : ils fournissent les TEXTES au CR, le
   superviseur consigne a la fusion. Fusion `--no-ff`, gates rejoues sur l'arbre fusionne
   (typecheck apres purge `node_modules/.tmp`, lint, vitest ; Go : build/vet/test/golangci
   `--new-from-merge-base=origin/main`), puis suppression du worktree et de la branche.
3. **JAMAIS `git add -A`** (fichiers d'autres sessions dans l'arbre) ; JAMAIS de pause d'attente
   passive dans un agent (un agent en attente d'une notification externe ne se reveille pas — le
   reveiller par SendMessage avec l'etat constate) ; **verifier l'ID d'agent avant SendMessage**
   (un message envoye a un agent TERMINE le REVEILLE et le fait travailler sur le meme worktree —
   c'est arrive une fois, stoppe par TaskStop sans degat).
4. **Verifier le CR sur pieces avant de le relayer** : SHA, `git status`, fichiers, chiffres
   reproductibles (un `grep`, un `ffprobe`, un `head` du golden). Les agents rendent parfois des
   affirmations fausses par ignorance de l'existant (« aucune icone d'assistance » alors que
   `killfeed-62` existait sans nom dans l'index).
5. **Aucun rendu, son ni nom sans donnee mesuree** ; une valeur d'ecran DECLAREE (rayon du capteur,
   plancher de lisibilite) s'ecrit comme telle avec sa source ; tokens uniquement, FR+EN, un rang
   non resolu s'affiche comme rang, une famille inconnue ne dessine rien.
6. **Les fichiers d'autres sessions** (`PLAN_HABILLAGE_REJEU_2D.md` etc.) : ne pas les toucher, ne
   pas les committer ; l'utilisateur les traite. Coordonner autour de `ReplayCanvas.tsx`, le
   fichier chaud commun.
7. Le flake vitest connu : `PalmaresRelationsPage` (timeout 5 s sous charge, 14/14 isole) et les
   garde-rails a balayage disque (`skillTiers.guard`, `lab-removal.guard`, `generated-types-fresh`,
   `keys.guard`, `xuidMeta.guard`) — verts isoles ; `internal/himap` complet depasse le timeout Go
   local (tests `_gamefiles`, sautent en CI).

## 4. Fichiers a lire dans l'ordre pour reprendre

1. Ce handoff. 2. `.ai/V7.5/REGISTRE_REPORTS.md` (fin de fichier = les lignes du 18/08).
3. `.ai/thought_log.md` (tete). 4. `PLAN_ARMES_AU_SOL_2E_LECTURE.md` (item 6, a valider).
5. `PLAN_ORIGINE_POSES_ET_FAMILLES.md` (dernier lot ferme, W statuee) et
   `PLAN_NOMMAGE_EQIP_TRANSLOCATEUR.md` (nommage, translocateur negatif).
6. `PLAN_RETOURS_PLANCHE_2026-08-16.md` (le bilan utilisateur verbatim et sa conversion en lots).
7. `FUSION_WT_2026-08-16.md` (procedure de fusion d'un worktree frere).
8. Dans `wt/fusion-lots-go` : `.ai/thought_log.md` (tete : synthese superviseur R3-R6) et
   `WALK_PORT_NOTES.md` § IMAGE-CLE (grammaire `ti=42` decompilee, non branchee).

## 5. Ordre de reprise propose

1. Demander a l'utilisateur : (a) go pour la fusion de `wt/fusion-lots-go` ; (b) validation du plan
   item 6 ; (c) date de la fenetre ops.
2. Committer `PLAN_ARMES_AU_SOL_2E_LECTURE.md` avec le prochain lot de docs.
3. Remettre la planche a jour et lui donner la liste des gates en une session.
4. Puis, dans l'ordre : item 6 (sur validation) -> item 4 objectifs vivants (plan d'abord) ->
   refactor `ReplayCanvas.tsx` -> fenetre ops.
