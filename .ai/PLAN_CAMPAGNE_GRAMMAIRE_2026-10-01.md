# PLAN — Campagne de recherche sur la grammaire du jeu, phase 1 (2026-10-01)

> **Statut : PHASE 1 CLOSE le 2026-10-02 (tous les items statués) ; PHASE 2 EN ATTENTE des décisions
> du §6.3 et d'un GO daté par lot ; rien ne se fusionne avant J12.** Phase 1 lancée le 2026-10-01
> sur décision de l'utilisateur du même jour
> (questionnaire : « résidu de film dense rouvert dans une campagne bornée qui commence par l'outil
> de mesure » ; puis « tu vas travailler dans un worktree dédié et tu peux te mettre en ultracode »).
> Contrat : skill `plan-execution` ; ce plan fait foi en cas de divergence.
>
> **Pour qui** : le superviseur et les agents de la campagne ; l'utilisateur pour les points d'étape.
>
> **Source** : `.ai/ANALYSE_MISE_EN_OEUVRE_REPRESENTATION_INTERMEDIAIRE_2026-10-01.md` (§5.1, §5.3, §7)
> et ses deux rapports (`.ai/V7.5/film_re/RAPPORT_IR_CARTOGRAPHIE_GO_2026-10-01.md`,
> `RAPPORT_PORT_RUST_2026-10-01.md`).

---

## 0. Objet et périmètre

**Objet de la phase 1** : savoir, sur pièces (Ghidra + mesure), POURQUOI les paquets ne ferment pas,
et écrire le plan des correctifs (phase 2) classés par gain mesuré. La phase 1 ne change AUCUNE sortie
de production : elle ajoute un instrument (code `research` et carte de fermeture), lit le jeu dans
Ghidra, mesure, et écrit.

**Pourquoi maintenant, avant la fusion de J12** : la recherche ne touche presque aucun fichier de J12
(J12 = `go fix`, tris, contexte et `slog`, variables de paquet, tag `research`, docs). Les correctifs
qui CHANGENT une sortie (phase 2) se développent ensuite sur cette branche, mais leur preuve (gate de
corpus, références d'équivalence) et leur fusion se font contre la tête POST-J12.

**Hors périmètre** : tout correctif de grammaire qui change une sortie ; la représentation
intermédiaire (attend J12) ; le retrait des replis nuls (attend la vague J11.4) ; toute cuisson
d'artefact, tout backfill, toute base DuckDB.

**Branche / worktree** : `feat/campagne-grammaire`, worktree
`C:/Users/Guillaume/Downloads/Scripts/LevelUp-wt-campagne-grammaire`, base `feat/v75` = `69564ef7d`.
Pas de jonction : les films se lisent EN PLACE depuis le checkout principal
(`C:/Users/Guillaume/Downloads/Scripts/LevelUp/data/cache/film_chunks`, `film_manifests`), en lecture
seule.

## 1. Règles opérationnelles

- Aucun agent ne commite ; le superviseur commite après accord de l'utilisateur.
- Commandes `go` : UNE à la fois sur la machine ; chaque agent qui compile pose son propre
  `GOCACHE` (répertoire dédié dans le scratchpad) et `GOFLAGS=-count=1` si besoin.
- Films lus un à un, plafond mémoire 4 Gio (`-plafond-gib 4`), sorties HORS de `data/`.
- Ghidra : `HaloInfinite.exe` ouvert, serveur HTTP du plugin sur `http://127.0.0.1:8089` (catalogue
  `GET /mcp/schema`) ; LECTURE SEULE (jamais `rename_*`, `set_*`, `create_*`, `add_*`, `delete_*`,
  `batch_*`, `apply_*`). Image base `0x140000000`. Table composant → désérialiseur STATIQUE : nom ASCII
  → xref DATA → `vtable+0x08` → désérialiseur en `vtable[0x28]` (ou `[0x30]` si c'est le thunk
  `FUN_14076ce9c`).
- Pas de Python ; docs en français ; aucun emoji dans les fichiers versionnés.
- Toute découverte hors périmètre va au §5, non traitée.

## 2. Étapes

Les étapes 1 et 2 sont indépendantes et tournent EN PARALLÈLE (le plan le déclare) ; 3 attend 2 ;
4 attend 1 et 3 ; 5 attend 4.

### Étape 1 — Instrument de fermeture v2 (code `research` + carte, aucune sortie de production)

- [x] 1.1 « Terminateur hors cadre » ventilé par : sortie de la vue B (terminateur / rejet hors datum /
      rejet de vue / autre sortie connue de l'observateur), vue C vide ou nombre d'entrées, reste du
      payload en bits (0-7, 8-63, ≥ 64), dernier composant lu avant la fin (archétype, index).
- [x] 1.2 Pour une sortie par rejet : état du slot rejeté dans le bloc de type 1 du chunk (vivant /
      trace / vide / absent) ; et, quand c'est mesurable, si l'eid rejeté naît après la dernière
      image-clé sans NEW lu (naissance non lue).
- [!] 1.3 Dénominateur des entrées de contrôle utiles (entrées de la vue C qu'un canal de production
      lit), pour que la moitié « entrées » du déclencheur se calcule.
      **Non atteignable, prouvé (2026-10-02, MESURES_BIS_4 §1)** : la présence d'une entrée dans une
      trame dépend de la décision d'envoi par pair (`FUN_1405185b0` : horloge murale, cadence,
      débit) et du budget du pair, états d'exécution du réseau absents du film ; seule la pose du bit
      (joueur avec unité, `FUN_14076ac14`) s'en déduirait. Reste la borne basse (MESURES_BIS_1 §1 :
      HI_1_13_0 ≥ 45,0 %, corpus ≥ 24,2 %). À revoir avec l'utilisateur : la moitié « entrées » du
      déclencheur de la spec n'est pas mesurable exactement.
- [x] 1.4 Colonne `product_use` de `grammar/testdata/ecs_table.tsv` corrigée sur pièces (au moins
      `ti=35 i1, i29, i54, i57, i62`, `ti=9 i0`, `ti=40 i0..i2`, `ti=13 i0..i3`, `ti=42 i0`, et toute
      ligne dont un canal de production publie la valeur), consommateurs et goldens mis à jour.
- [x] 1.5 Compte d'événements DÉCLARÉ du chunk de type 3 (paquet type 9) comparé aux événements
      trouvés (kill-feed, fil des morts), par film.
- [x] 1.6 Mode borné : records et composants dont la lecture dépasse la fin du payload (bits de
      bourrage), et NEW liés sur un corps qui déborde, comptés par film.
- [x] 1.7 Carte v2 jouée sur les 20 films (les 19 témoins de `config/replay_corpus.toml` +
      `1c4c63c2`), rapport `.ai/V7.5/film_re/CARTE_FERMETURE_V2_2026-10-01.md` + TSV.

**Gate 1** : `go vet -tags=research` et tests (`-tags=research`) des paquets touchés ; tests de
fermeture de `grammar` (`-run 'Closure|Fermeture'`) ; si un fichier de production de `grammar`
change, empreinte régénérée à révision CONSTANTE par la commande du dépôt
(`LEVELUP_UPDATE_GRAMMAR_REV=1 … -update-grammar-rev`) et `grammar.Rev` inchangé ; aucune sortie de
production modifiée (aucun fichier lu par une cuisson changé hors `frame_closure*`).

### Étape 2 — Recherche dans le jeu (Ghidra, lecture seule), huit pistes

- [x] T1 Naissances non lues : comment l'écrivain émet la création d'une entité née (et morte) entre
      deux images-clés dans une trame delta ; pourquoi notre marche ne la lit pas (NOTE 5.26).
- [x] T2 Décalage de masque : `FUN_14076cb60` teste `i - decales` là où le Go teste `i` (NOTE 5.16 D1).
- [x] T3 Fin de vue B : ce que fait le lecteur du jeu sur un DELTA dont l'eid n'est pas dans la table
      de datums (codes 2/3), et si une sortie par rejet est jamais légitime dans un flux écrit par le jeu.
- [x] T4 Grammaire de position dépendante du build ou du contenu : les 14 exceptions datées
      (`grammar/lecteur_position_ratchet_test.go`, `exceptionsDuPortage`).
- [x] T5 Vue C : kinds 1 et 2, bloc secondaire `0xbc` (forme courte de `DAT_145121140`).
- [x] T6 Véhicules : composants `ti=40` non portés (`i30-i33`, `i35`, `i36`, `i38-i42`, `i45-i47`) et
      l'écrivain de l'octet `+0x818` (porte de `vehicle-type-physics`).
- [x] T7 Dispositifs et moteur : `ti=43 i19-i23`, `i35`, `i39` ; `ti=2 i15` (`managed-engine-timers`).
- [x] T8 Bourrage : le lecteur de bits du jeu rend-il des zéros au-delà du tampon, ou signale-t-il une
      erreur ?

### Étape 3 — Vérification adverse de chaque constat de l'étape 2

- [x] 3.1 Chaque constat est relu par un contexte frais qui cherche à le RÉFUTER (Ghidra + code) ;
      seuls les constats non réfutés passent à l'étape 4.

### Étape 4 — Mesures ciblées

- [x] 4.1 Chaque constat retenu est confronté à la carte v2 (et, si besoin, à une sonde `research`
      neuve) : combien de paquets « hors cadre » il explique, sur quels builds.

### Étape 5 — Synthèse

- [x] 5.1 Rapport `.ai/V7.5/film_re/RAPPORT_CAMPAGNE_GRAMMAIRE_PHASE1_2026-10-01.md`.
- [x] 5.2 Plan de la phase 2 (correctifs) au §6 de ce fichier : lots classés par gain mesuré, chacun
      avec sa preuve Ghidra, son vecteur de test (T2 de la spec), son effet attendu sur la fermeture,
      et sa dépendance à J12.

## 3. Décisions

- 2026-10-01 (utilisateur) : résidu de film dense rouvert ; campagne bornée commençant par l'outil de
  mesure ; worktree dédié ; ultracode autorisé.
- 2026-10-02 (utilisateur, questionnaire, recommandations retenues) — décisions FERMES :
  - **D-RI** : ordre « composants, naissances, RI » : après la fusion de J12, vague 1 = lots de
    composants + une recuisson ; références re-figées ; vague 2 = LU (unification des localisateurs,
    zéro différence) puis L1 et L7 + une recuisson ; ensuite seulement l'étape 1 de la représentation
    intermédiaire (§6.3, D-RI). Remplace l'ordre « marcheur avant la campagne » de l'analyse.
  - **D-VEH** : oui, tous : `ti=43` en vague 1 ; `ti=40` et l'octet `+0x818` en vague 1 dès que
    leurs prérequis sont levés (table châssis -> type de physique, D5).
  - **D2** : les fermetures factices sont EXCLUES de la mesure : fermé = reste du paquet nul ET aucun
    invariant de l'écrivain violé (lot L0, qui touche l'outil et un fichier lu par la publication).
  - **D1** : le seuil de 95 % devient un INDICATEUR publié à chaque vague (records utiles,
    dénominateur fixe, sans factices), plus un déclencheur ; la RI démarre selon l'ordre D-RI.
  - Restent ouvertes : D3 à D11 (à poser au moment des lots qu'elles concernent) et le GO daté de
    chaque lot ; rien ne se fusionne avant J12.

## 4. Journal

- 2026-10-01 : worktree et branche créés depuis `feat/v75` = `69564ef7d` ; plan écrit ; étapes 1 et 2
  lancées en parallèle (workflow).
- 2026-10-01 (soir) : workflow `wf_8376152d-2f9` terminé (98 agents Opus, 0 erreur). Étape 1 : items
  1.1, 1.2, 1.4 à 1.7 faits ; gate 1 passé (`go vet -tags=research` grammar + cmd_fermeture propre ;
  `go test -tags=research -count=1` grammar 73 s et cmd_fermeture ok ; `-run
  'Closure|Fermeture|GrammarRev'` PASS ; empreinte régénérée à révision constante
  `grammar-2026-09-27.3` ; aucun fichier de production modifié hors `frame_closure_detail*.go` neufs
  et goldens). Item 1.3 NON tenu (estimateur tautologique, critique point 22) : repris en phase 1 bis.
  Étape 2 : huit notes T1-T8. Étape 3 : 43 constats, 41 retenus, 1 contesté (T1-6), 1 réfuté (T6-C5) ;
  verdicts consignés dans `campagne_grammaire_2026-10-01/VERIFICATIONS_ADVERSES.md` (extraits du
  journal du workflow). Étape 4 : mesures faites mais incomplètes (critique de complétude, 39 points,
  `CRITIQUE_COMPLETUDE_1.md`) ; étape 5 : rapport et §6 écrits, à corriger d'après la critique.
  Trois décompilations Ghidra écrites par un agent à la racine du checkout PRINCIPAL (hors worktree)
  déplacées dans `campagne_grammaire_2026-10-01/ghidra_T8/`.
- 2026-10-01 (soir) : phase 1 bis lancée (workflow `wf_a8a23323-1e5`) : mesures manquantes en trois
  agents séquentiels (1.3 et L1 ; fourche 0xbc, positions, T6-C2, témoin 81c02726 ; populations sans
  lot), corrections des documents, critique finale. Les items 1.3, 4.1, 5.1 et 5.2 restent ouverts
  jusqu'à sa fin.
- 2026-10-02 : phase 1 bis terminée (5 agents) : MESURES_BIS_1 (1.3 en borne basse, L1 ventilé,
  invariants des gains, dénominateur fixe), MESURES_BIS_2 (bloc 0xbc réfuté, positions A/B, Live Fire
  établi, T6-C2 mesuré, témoin `81c02726` : +3 580 paquets sous la grammaire T7), MESURES_BIS_3
  (lot `ti=3` +30 618 paquets, image-clé incomplète +1 784, IDLowBits = ceil(log2(entrées du bloc de
  type 1)) = 13 partout) ; critique n° 2 (`CRITIQUE_COMPLETUDE_2.md`, N1 à N21) dont N2 : `archlint`
  ROUGE (deux sondes > 500 lignes), constaté par le superviseur. Clôture (2 agents) : sondes découpées
  (déplacement pur, `archlint` vert) et documents corrigés (ordre en deux vagues, D-RI réécrite, D10,
  D11, gates étendus). Branche J12 (`origin/feat/suite-audit-decodeur-j12` = `bc0e2511a`) récupérée et
  confrontée par diff (`J12_RECOUPEMENTS_2026-10-01.md`). Gates rejoués par le superviseur sur
  l'arbre : `gofmt -l` vide ; `go vet ./internal/games/halo_infinite/film/...` ; `go vet
  -tags=research` (grammar, cmd_fermeture, himodule) ; `go vet -tags=research,campagne_overlay
  -overlay=…` ; `go test ./internal/archlint/` ok ; `go test` grammar complet ok (26 s) ; `-run
  'Closure|Fermeture|GrammarRev'` ok ; `go test -tags=research` cmd_fermeture ok ; `grammar.Rev` =
  `grammar-2026-09-27.3`. Items 4.1, 5.1, 5.2 cochés. Item 1.3 (dénominateur exact, appelant de
  `FUN_14076b0e8`) et reste du point 25 (juge des invariants sur les A/B de position) confiés à un
  dernier agent (MESURES_BIS_4).
- 2026-10-02 : MESURES_BIS_4 rendu. 1.3 statué `[!]` (raison prouvée ci-dessus). Point 25 : net sain
  des treize sites +614 ; `tacmap-displayasset` seul site à gain sans perte saine ; `flock-position`
  +391 mais −6 records utiles sur `60ae07c4`. Gate de l'agent : `gofmt`, `go vet -tags=research`,
  `go vet` avec surcouche, `archlint` verts. **Phase 1 close** : tous les items du §2 statués ;
  commit sur `feat/campagne-grammaire` (sans push) ; décisions du §6.3 soumises à l'utilisateur.
- 2026-10-02 : décisions de l'utilisateur consignées au §3 ; lot LS (D-67) confié à la campagne.
  Recherches préalables R-* lancées (workflow `wf_9088d8bd-e43`, six chantiers en worktrees
  temporaires `LevelUp-wt-cg-*`, détachés sur `fe18bf67c`). J12 fusionné dans `feat/v75`
  (`95b19e635`) par la session de suite d'audit ; **fusion de `feat/v75` dans la campagne faite**
  (`f28a4a816`) : un conflit (`grammar_rev.golden`, régénéré à révision constante), quatre sondes
  mises à l'accesseur de J12.4 ; gates verts sur l'arbre fusionné (`gofmt`, `go vet` film, `go vet
  -tags=research ./...`, surcouche, `archlint` complet, `go test` film 17 paquets, `cmd_fermeture`
  research, révision) ; règle 17 sur le code neuf (`32ba9078d`) ; carte v2 rejouée après J12 :
  identique à la phase 1. Chemins `.ai` cités par le code de la campagne vérifiés à la casse exacte
  contre `git ls-files` (piège CI Linux signalé par l'autre session).

## 5. Découvertes (consignées, non traitées)

Rendues par les agents de la phase 1 (instrument, pistes T1-T8, vérificateurs, mesures). Aucune
n'est traitée ; certaines deviennent un lot au §6 (renvoi entre crochets).

**Mesure et instrument**

- D-1 Fermetures factices : 8 388 paquets « fermés » (2,9 %) contredisent un invariant de l'écrivain
  (surtout un NEW à masque impossible dans un paquet d'un seul record) ; 6 410 sur HI_1_10_0 (22 % de
  ses fermés). `vueCFermee` est nécessaire, pas suffisant. [L0, décision D2]
- D-2 4 598 paquets ferment au bit près après une sortie de vue B par rejet (3 756 sur HI_1_10_0) :
  pied de trame d'une vue vide ou fermeture factice, non tranché. [R-L1]
  **Tranché le 2026-10-02 (R-L1 (b), `R_NAIS.md` §2)** : fermetures FACTICES, y compris les 1 008
  que le juge actuel dit saines. Un début de vue C antérieur ferme le paquet pour 4 402 / 4 598
  (95,7 %) contre 11,0 % au témoin (mesuré) ; « la vue B a débordé son terminateur » est une
  déduction (supposé). [L0, invariant « sortie de vue B par rejet ⇒ non fermé », §6.5]
- D-3 Région (iii') : 1 289 NEW trouvés derrière le point de rejet d'un paquet antérieur (témoin 90),
  contraire à l'ordre NEW-avant-DELTA de l'écrivain. Non tranché. [R-L1]
  **Tranché le 2026-10-02 (R-L1 (a), `R_NAIS.md` §1)** : l'écrivain n'écrit jamais un NEW après un
  DELTA dans un paquet (lu, Ghidra : `FUN_14076b9c8`, `FUN_142f2cc78`, `FUN_142f2cee0`,
  `FUN_14076a148`). Les « NEW » de (iii') ne sont pas des records (X attesté 1 094 / 1 289, soit
  994 / 1 289 sans les 100 « réalloués » qui reposent sur D-43). L1c reste retiré.
- D-4 Naissances ratées = entités nées et mortes dans le chunk : 5 714 des 6 033 eid ont un masque
  vide au bloc suivant ; la table anticipée ne peut pas les rattraper.
- D-5 Image-clé incomplète : 87 eid vivants au bloc du chunk, ni liés au début du chunk ni déclarés
  par l'image-clé, portent 13 644 paquets, dont 13 643 hors cadre (BIS_3 §1.2 ; libellé corrigé le
  2026-10-02) (14 723 rejets « vivant au bloc », 20 % des
  rejets de `1c4c63c2` et `4f77afc1`). Piste de la marche d'image-clé, distincte des naissances.
  **Cause établie le 2026-10-02** (bis 3 §2) : l'en-tête est présent dans l'image-clé pour 87 eid
  sur 87 ; la marche Go rejette la génération 0 (`kfAnchorFromID`) et limite voisin et recalage à la
  génération 1. Oracle : +1 784 / −1. [L9]
- D-6 `0797ce72` : 1 809 de ses 4 544 rejets hors cadre visent un eid qu'aucun bloc de type 1
  n'alloue (position fausse, ou génération 0 née puis morte). **Établi le 2026-10-02** (bis 2 §3.3,
  bis 3 §3.2) : ce sont des véhicules hors de l'arène (index de plage 3) lus aux largeurs de la
  plage 1 ; la lecture par index retire 81 % des rejets « aucune allocation » du film. [L6a]
- D-7 Pont masque -> archétype : sur les 266 naissances où il résout, le `R(6)` trouvé le contredit
  144 fois. Non tranché.
- D-8 Allocateur : la table de pools par plage de slots ne prédit rien avant HI_1_12_0 (≤ 15 % au
  rang 0) : autre table, ou `DAT_144706104` à 0 sur ces films.
  **Seconde branche réfutée le 2026-10-02 (R-L1 (d), `R_NAIS.md` §3.1)** : le bit vaut 1 dans
  629 142 / 629 142 paquets delta ; le modèle à pool unique prédit au plus 2,80 % (HI_1_4_1). La
  cause de la faible prédiction avant HI_1_12_0 reste ouverte (D-81).
- D-9 Chunk des temps forts : le nombre de kills diffère de celui des deaths de 0 à 5 par film ; non
  étudié.
- D-10 T4-C3 : `IndexAbsolus` sous le contexte des instruments (`Region = 0`) contredit en apparence
  la mesure Live Fire de `map_bounds.go` (59 376 / 59 377 i0 d'index 1) ; à remesurer sous le
  contexte de production. **Remesuré le 2026-10-02** (bis 2 §3.3) : sous le contexte de production,
  Live Fire lit l'index de la plage jouée pour 97,0 % (`0797ce72`) et 98,1 % (`60ae07c4`) des
  lectures ; T4-C3 établi sur Live Fire. [L6a]
- D-11 Le dénominateur des entrées de contrôle utiles se confond par construction avec la part des
  paquets fermés ; un dénominateur indépendant exige de lire combien d'entrées l'écrivain produit
  par paquet. **Prouvé tautologique et remplacé le 2026-10-02** (bis 1 §1) par des bornes basses
  mesurées (au plus une entrée par joueur) : HI_1_13_0 ≥ 45,0 %, corpus ≥ 24,2 %. Le dénominateur
  exact attend la lecture de l'appelant de `FUN_14076b0e8`.

**Code et tables (constats, aucune correction faite)**

- D-12 `ScanObjectives` (`grammar/objective_scan.go:179`) et `ScanEquipmentState`
  (`grammar/equipment_state.go:242`) n'ont aucun appelant de production ; la table les disait
  « publie par hook » (corrigée). Statut du code (instrument ou mort) non tranché — règle « 0 code
  mort ».
- D-13 Commentaires faux à annoter : `frame_infer.go:122-125` (« cause nommée : décalage du masque »,
  démenti par T2) ; `frame_infer.go:208-222` et `rev_chronique.go:182-184` (« le jeu ne crée pas sur
  une entrée occupée », démenti par `FUN_1408f18d0`) ; `traverse.go:230` (attribue `FUN_14076cb60`
  à l'écrivain) ; `frame_vue_controle.go:48-50` (kinds 1/2 « dépendants d'un état d'exécution » :
  `FUN_142f2b574` garde aussi le kind 0, le vrai motif est l'absence dans l'écrivain du film) ;
  `component_param4.go` (getter de niveau situé à `+0x10` au lieu de `vtable[0]`) ;
  `World.BindImageCle` (« bits de tête = rang de vue » : c'est la génération de l'eid,
  `FUN_142f30610` -> `FUN_1406d5110`) ; `NOTE_3_6_TI40` et `NOTE_3_6_TI43_*` nomment « écrivain » le
  désérialiseur.
- D-14 `ecs_table.tsv` : `ti=43 i2` statut « partiel » et texte « mode C=1 NON porté » périmés
  (`components_dynprec_orientation.go` le porte) ; `ti=43 i11..i18` colonnes grammar/bits vides ;
  `ti=40 i41/i42` « NE PAS PORTER » ne vaut que pour le delta (lus en image-clé) ;
  `tacmap-poiicon` : désérialiseur = `FUN_142ed4834` (thunk vers `FUN_142ed8418`).
- D-15 Règle des deux copies déjà dépassée : `FUN_140d580d0` (R(n)+R(n)+R(5)) est recopié cinq fois
  en production (`ti=5 i2`, `ti=0 i5`, `i6` `components_game_engine.go:103`, `i7` `:112`, `ti=2 i12`)
  sans helper ni garde-rail. [L3]
- D-16 `decodeInferLoop` ne lit pas la garde `R(1)[+R(8)]` que `FUN_1406cd128` lit pour NEW et DEL
  quand `FUN_14076cea8()` est vrai (`frame_records.go:272,291` la lit). Sans effet tant que
  `HasExtraFields` est faux (20 films sur 20).
- D-17 `marchViews = 8` (`object_deaths_march.go:44`, `killsource.Options.Views`) : l'écrivain prouve
  exactement trois vues.
- D-18 `marchLocateStrict` saute aussi les DELTA de slot < 123 (le premier DELTA est le plus petit
  slot modifié), pas seulement les NEW.
- D-19 `consume14076d528` (`unit_control.go:91`) fige la norme à 10 ; le bloc `0xbc` en exige 8 au
  second site. [ancien L5, sorti le 2026-10-02 : le bloc n'est jamais lu au bon bit sur le corpus ;
  écart de grammaire sans effet mesurable, consigné.]
- D-20 `plafondToursVueC = 64` pourrait descendre à 32 (borne de l'écrivain) ; sans effet de sortie.
- D-21 Le lecteur du jeu plafonne à `0xa00` records sur les trois vues (`FUN_142987460`) ; le Go
  n'applique pas ce plafond.
- D-22 `ecs_widths_guard_test.go` (G4) : comptes gelés à faire bouger avec tout portage ; `i31` sur
  motif `0xFF` (N = 255) = échec du lecteur, catégorie à définir. [L2]
- D-23 `varWidthBits(0/1)` est statique (13 bits) alors que les notes A-D du 17/09 parlent d'une
  largeur « calibrée par film ».
- D-24 La largeur de l'identifiant de record n'est pas constante sur un film (`FUN_1408f1618`,
  `FUN_142f2f0cc` : `DAT_144706100` = slot+1 au-delà du cardinal) alors que le Go fige
  `IDLowBits = 13` (`frame_records.go:144`) ; sans effet sous 8 192 slots. Le bit de configuration
  (`DAT_144706104`) bascule à 0 quand un pool borné est plein. **Réfuté comme cause le 2026-10-02**
  (bis 3 §5) : largeur = `ceil(log2(cardinal))`, cardinal = 8 191 sur chaque bloc des 20 films ;
  toute autre largeur effondre la marche (≤ 30 040 paquets fermés). [L10, garde-fou]
- D-25 Les cinq mots de queue du bloc de type 1 (`type1_datums.go` `Queue`) sont les curseurs de
  l'allocateur (table+0x160), pas des compteurs de magasin (NOTE 5.21 §3.3 à corriger).
- D-26 Registres : `ti=43` en trois familles (HI_1_10_0 à HI_1_13_0 ; HI_1_8_0 = HI_1_9_0 ;
  HI_1_4_1 = v31 = v33) ; `ti=0` à 27 entrées (forge `i18-i26`, non relevés, bloquent `c75f33b8`),
  26 sur version-31 ; registre `ti=40` variable selon le build (contre `CADRAGE_VEHICULES` §1.5).
- D-27 `FUN_14076e29c` : le `R(2)` final n'est lu que si la position est finie ; sur la branche
  R(96), le Go le lit sans condition. `object-position-dynamic-precision` a deux lecteurs (delta
  `FUN_1406cfe44`, état complet `FUN_14076e29c`).
- D-28 Un corps de composant peut être la RECOPIE d'un envoi précédent (`FUN_142e2b368` /
  `FUN_142e2f7f8`, branche cache) ; sans effet sur la largeur.
- D-29 `FUN_142e2ec8c(ti)` exempte certains archétypes de la sentinelle de corruption ; non lue.
- D-30 `FUN_1406caad8` rend 3 quand l'entrée d'historique désignée manque ; le Go lit le corps avec
  une référence nulle (14 records sur `bfecd02b`) : écart de valeur, pas de cadrage.
- D-31 Le lecteur du jeu arrête la vue C (code 3) sur une entrée kind 0 `a = b = 0` que l'écrivain
  peut produire (112 dans des vues C fermées) ; sans effet hors ligne.
- D-32 `DAT_145121140` est un sélecteur de variante de moteur (valeurs 0..3, objet `0x145120f00`),
  pas une constante de format ; il gouverne aussi `FUN_1406cd860`, `FUN_1406d33cc`, `FUN_1407699d0`.
- D-33 Le premier argument de largeur de `FUN_14076d528` varie selon le site (direction
  `[RSP+0x30]`, norme `[RSP+0x28]`).
- D-34 La Kill-cam (`FUN_142f28768`, `KillPlayback.cpp`) construit une seconde vue d'entités de même
  classe et de même index `0x20` que la vue du film.
- D-35 La file de records différés `vue+0x1b320` (NEW lu sans allocation) : aucune écriture trouvée ;
  si elle est active en lecture de film, le jeu lui-même rejetterait.
- D-36 Après un paquet de type 0 en échec, `FUN_1428e27c0` arrête la boucle des blocs ; les records
  du paquet sont déjà appliqués.

**Produit (à soumettre, hors campagne)**

- D-37 `ti=40 i40 vehicle-equipment-turret-parent` porte une référence d'entité : il pourrait LIRE le
  rattachement d'une pièce montée à son porteur (aujourd'hui déduit par voisinage de slot).
- D-38 `ti=40 i35 vehicle-auto-turret-target` porte la cible d'une tourelle automatique.
- D-39 `ti=43 i36 device-dispenser-state` porte une référence vers l'objet distribué (véhicule sorti
  d'un socle).

**Divers**

- D-40 Fichiers non suivis `d_1406cd128.txt`, `d_1406d3140.txt`, `d_141f86704.txt` à la racine du
  checkout principal `LevelUp` : pas d'un agent d'écriture de la campagne (probablement un agent
  Ghidra) ; à faire statuer par l'utilisateur.
- D-41 Tests SKIP préexistants vus au gate (`TestImageCleFermetureParArchetype`,
  `TestKeyframeClosureInventaire`, `TestProfilFermeture186`, `TestProfilFermetureTousArchetypes`,
  `TestResidusSlotFermeture`, `TestG2TableSuitLeRegistreDuFilm`) : non touchés.

**Découvertes de la phase 1 bis (mesures bis 1 à 3 et correction des documents, 2026-10-01/02)**

*Mesure et instrument*

- D-42 **Le dénominateur fixe n'est pas le nombre de records écrits par le jeu ; il monte quand un lot
  lit plus loin.**
  - Sous la grammaire `ti=3`, HI_1_13_0 lit 2 759 700 records utiles, contre 2 504 223 en
    référence ; sous `ti=43`, HI_1_12_0 en lit 139 947 contre 121 628.
  - Tout pourcentage du déclencheur est donc une borne haute tant que la lecture n'est pas complète.
  - (Mesuré : `mb3_ti3.tsv`, `mb2_ti43.tsv`.)
  - **Confirmé le 2026-10-02 par les recherches préalables** : chaque chantier a recalculé son
    propre fixe (HI_1_13_0 : R-COMB 2 880 403, R-LS 2 931 799, R-L1 (d) 2 850 786) ; le maximum
    consolidé des quatre chantiers vaut 2 959 104 sur HI_1_13_0 et 7 176 150 sur le corpus (§6.5.3,
    calcul awk sur leurs TSV). Le fixe14 de bis 1 est périmé (HI_1_12_0 y afficherait 103,7 %).
- D-43 **La carte v2 et les sondes rangent une naissance de génération 0 en « réalloué sous une autre
  génération »** : 12 854 des 12 906 paquets. La cause : `cmAlloueSous` et `v2_datums.go` exigent
  `Gen != 0 || Drapeaux != 0` (bis 3 §3.1). [L0]
- D-44 **La carte v2 range un NEW lu mais désynchronisé en « naissance non lue »** : 3 560 paquets sur
  `81c02726` (bis 2 §7.2). La région (iii) et l'oracle-NEW en dépendent. [L0]
- D-45 **Les liaisons de l'oracle sur les régions (iii') et (iii) sont nuisibles** : −522 et −65
  paquets nets, et 40,5 % et 100 % de leurs gains sont factices (bis 1 §3-4). Elles sortent de la
  borne de L1.
- D-46 **L1a, filtre d'invariants** : il réduit les pertes saines de HI_1_8_0 à HI_1_11_0, sans les
  annuler (utiles sains −1 953, −1 404, −21 094, −1 904). Il faut chercher une condition mesurable
  PAR FILM, par exemple la prédiction de l'allocateur vérifiée sur les NEW lus du chunk. Une branche
  sur le build est interdite. [R-L1 d]
  **Instruit le 2026-10-02 (`R_NAIS.md` §3)** : condition trouvée (allocateur à cinq pools ≥ 50 %
  des NEW propres du film, ≥ 30 NEW) ; 0 film en baisse, +15 070 paquets sains. Le seuil est choisi
  sur le corpus même (aucune validation hors échantillon) et la condition DÉSACTIVE L1a sur
  HI_1_8_0 à HI_1_11_0 (gain nul sur ces builds, pas une correction). [§6.5, D16]
- D-47 **Coquilles des documents** :
  - « 2 836 eid introuvables » doit se lire 2 856 ;
  - « 6 451 » (CARTE §4, compté rejet par rejet) et « 6 455 » (compté par eid) sont tous deux
    justes ;
  - « Moteur `ti=2` (4 331) » contient 123 paquets `ti=0` ;
  - « 12 112 sur un film » désigne `i35` seul ; `ti=43` vaut 12 127 paquets sur HI_1_12_0.
- D-48 **La surcouche de recherche `go test -overlay`** (copies de `capture.go`,
  `lecteur_position.go` et `lecteur_position_exceptions.go` dans `mesures_bis2_overlay/`) est la seule
  voie d'A/B sur le dispatch des composants sans modifier de fichier de production. **Méthode à valider
  par l'utilisateur** (bis 2 §1.2) : les mesures des lots L2, L6 et L8 en dépendent.

*Grammaire et code (constats, aucune correction faite)*

- D-49 **Deux composants s'appellent `high-frequency`, avec deux grammaires** :
  - `ti=4 i0` se lit en `R(8)` (`FUN_14076d034`) ;
  - `ti=3 i1` se lit sur 26 bits (`FUN_142ed4880`).

  Le dispatch route par NOM (`dispatch_item.go:122`). D'autres homonymes peuvent exister : il faut
  recenser, dans le binaire, les tables de composant par nom. [L8]
  **Recensé le 2026-10-02 (R-HOM, `R_COMP.md` §3)** : 326 noms, un seul homonyme de grammaire
  (`high-frequency`) ; la table de `ti=3 i1` est lue dans l'enregistrement de `ti=3`
  (`FUN_140e460fc`, `+0x4754 = 3`) : sa grammaire `R(16)+R(8)+R(2)` est ÉTABLIE.
- D-50 **`ti=3 i0 low-frequency` n'est pas porté** (`FUN_142ed4aec`). Toutes ses briques existent
  déjà dans le Go. [L8]
- D-51 **La génération 0 est une génération valide du jeu, mais le Go la traite comme nulle** :
  dans `kfAnchorFromID` (`keyframe_world.go:72-75`), au voisin et au recalage (génération 1
  seulement, `keyframe_world.go:131-159`), et dans les classes des instruments (D-43). Les autres
  lecteurs d'eid sont à revoir. [L9, L0]
- D-52 **Images-clés `ti=40`** : avec la bonne porte et les 16 composants portés, 98 % des records
  restent non fermés.
  - Une autre largeur de l'archétype est fausse en image-clé ; elle n'est pas identifiée.
  - Les pièces montées dépassent toutes leur frontière : 326 sur 326 sur `4f77afc1`. [R-L4]
  - **Identifié le 2026-10-02 (R-L4 (a), `R_VEH.md` §1)** : ce n'est pas une largeur propre à
    `ti=40` mais trois lectures communes : la portée `DAT_144e61ea0` posée par l'écrivain d'image-clé
    (`FUN_142e2bfd0`, `FUN_142e2c690`) qui fait lire `R(96)` ; le chemin absolu d'`i0` de l'écrivain ;
    le découpage MPP 8/3 sur les formats 24-25 (largeur MESURÉE, aucun exécutable de ces builds).
    Les trois ensemble : 3 058 / 3 058 voisins fermés sur les formats 24 à 27 ; formats 20-21 :
    0 / 2 036 (D-92).
- D-53 **Bloc `0xbc` de la vue C** : jamais lu au bon bit sur le corpus. Ses 1 570 arrêts marquent
  un désalignement, de même nature que les kinds 1/2/3. La fourche `+0x74` reste ouverte : rôle de
  `FUN_1404f293c` en relecture Theater. [L0 : requalification]
- D-54 **Cartes à un seul sbsp** (Illusion, Fragmentation) : 101 et 136 paquets fermés y lisent un
  index de plage 1. Soit la carte déclare une plage de plus, soit ces fermetures sont factices. Par
  ailleurs, l'ordre des plages des cartes à deux sbsp n'est pas lu. [R-L6]
  **Réfuté le 2026-10-02 (R-L6, `R_VEH.md` §5)** : en lecture FINALE des records retenus, 0 paquet
  fermé ne porte d'index impossible ; les 101 et 136 venaient des lectures d'essai de l'inférence,
  que le relevé `bis2NoterIndex` compte aussi (D-96). Ordre des plages ÉTABLI (plage 0 = arène,
  plage 1 = décor lointain, huit modules) ; Illusion et Fragmentation n'ont qu'une plage.
- D-55 **`world-object-i0` lu comme le jeu** : +12 338 records d'image-clé fermés, sauf sur
  version-31 (−364) ; seulement +31 paquets delta nets. [L6b, décision D4]
- D-56 **Coupables non attribués des décalages de curseur** : `ti=20 i1 spawn-filter-weight`
  (694 paquets), `DEL ti=0` suivi d'un en-tête nul (**658**, dont la moitié des « fermés après
  rejet », lien D-2), `ti=14 i1 crew-marked-objects` (241), `ti=41 i2 object-forward-and-up` (161).
  - Corrigé le 2026-10-02 : `mb3_tables.tsv` (marche `reference`, `dernier_composant_x_classe`,
    « aucune allocation ») donne 658 paquets hors cadre pour `DEL ti=0`. Le « 622 » recopiait le compte
    de `ti=21 i16 flock-position`.
  - Porter la branche de niveau de `spawn-filter-type` ne change rien (sortie identique à l'octet).
  - Son cas 3 (`p5 = 1`, `FUN_1407f1ff4`) n'est pas relu. [R-P3]
  - **Instruit le 2026-10-02 (R-P3, `R_COMP.md` §4, statut partiel)** : aucun lecteur n'est en
    cause (Ghidra). G1 (`ti=20 i1`) et G3 (`ti=14 i1`) sont des deltas lus sous une liaison fausse du
    slot vers une unité (bipède d'après les déclarations des slots, l'A/B ne discrimine pas bipède et
    véhicule) ; G2 (`DEL ti=0`) est un en-tête mal lu après un décalage antérieur (635 / 684 eid sans
    allocation) ; G4 (`ti=41 i2`) vient de `consumeObjectPositionMonde`, qui lit les largeurs de la
    plage de la carte au lieu de la ligne de l'index lu. [L6a, LP, L7, §6.5]
- D-57 **`ecs_table.tsv`** : la `deser_addr` de `ti=20 i1` vaut `FUN_14076ce9c`, qui est l'entrée
  commune `+0x20` de toutes les tables. Le vrai lecteur est `FUN_142ed70b8`.
- D-58 **`1c4c63c2` alloue le slot 8 190**, un de moins que la limite qui ferait passer la largeur à
  14 bits : D-24 deviendrait réel à un slot près. [L10]

*Documents et chantiers voisins*

- D-59 **`HANDOFF_RETOURS_REJEU_2026-09-25.md` §3.2 et `PLAN_RETOURS_REJEU_2026-09-23.md` l. 919 sont
  faux** : sur la tête, le NEW `ti=43` de `81c02726` désynchronise sur `i19`, pas sur
  `i20`/`i21`/`i22` (mesuré, bis 2 §5.2). Les 71 NEW `ti=43` portent tous le masque {i19, i21, i22,
  i23, i34, i35}.
- D-60 **La branche J12 est visible localement** : `origin/feat/suite-audit-decodeur-j12` =
  `bc0e2511a`. ANALYSE §8 et la critique (point 1) la disaient invisible. L'inventaire des
  recoupements (§6.0) est fait par `git diff 8b894a677 bc0e2511a`, qui touche 1 453 fichiers du dépôt.
  `J12_RECOUPEMENTS_2026-10-01.md` en compte 1 130 sous `film/`.
- D-61 **Le garde-fou de la recopie** (`TestFrameClosureDetailleeRendLaCarteDeFrameClosure`) couvre la
  branche « paquet à événements non localisé ». Le golden de `ks_000d5950` porte
  `listes_non_localisees=27` et compare la carte entière avec `reflect.DeepEqual`. Aucune pièce ne
  compte, en revanche, les paquets à événements LOCALISÉS de ces deux bobines. [L0, gate de L1]

**Découvertes de la critique de complétude n° 2 (2026-10-02)**

- D-62 **Code de production sans appelant de production (règle « 0 code mort »)** :
  `frame_closure_detail.go` et `frame_closure_detail_records.go` (neufs, phase 1), `FrameClosure` et
  `LireBlocDeDatums` (`type1_datums.go`) n'ont d'appelant que l'instrument `research/cmd_fermeture`
  (et des tests) — vérifié par grep le 2026-10-02. « Comme `FrameClosure` » est un précédent, pas une
  justification. Contraintes : le ratchet `research_tag_test.go` de J12.7 interdit le tag `research`
  sur un fichier non-test hors de `film/research/` ; L1 veut faire passer `LireBlocDeDatums` en
  production. **Question à trancher (D9), non tranchée ici** : déplacer ces fichiers sous
  `film/research/`, ou les garder en production avec une justification écrite (instrument du gate de
  chaque lot, appelant de production attendu avec L1).
- D-63 **La part de l'erreur D-44 sur le corpus n'est pas mesurée.** Le « 3 560 » vient de
  `81c02726`, HORS des 20 films. La part des NEW lus mais désynchronisés dans les 213 033
  « naissances attestées » du corpus est inconnue ; la région (iii) en montre au moins 13 978 paquets
  sur HI_1_13_0. Item ouvert, porté avec L0 (L0.2). [L0]
- D-64 **Sous la grammaire T7, le « hors cadre » de HI_1_12_0 monte de 3 249 à 4 029** alors que ses
  paquets fermés triplent. Mesuré (`mb2_ti43_causes.tsv`, `bcb6d393`, seul film du build) : c'est un
  **déplacement de cause bloquante**. La carte range un paquet sous son PREMIER bloquant. Les 12 127
  paquets bloqués par `ti=43` en référence (dont `i35` 12 112) et 21 listes non localisées ne le sont
  plus : 11 297 ferment, **780 avancent jusqu'à la vue C et y sont arrêtés hors cadre**, 71 butent sur
  `ti=12 i21`/`i22`, `ti=10 i2`, `ti=11 i4`, `ti=12 i16`/`i18`. Aucun paquet n'est perdu (+11 297 / 0).
- D-65 **La colonne « Estimateur CARTE §5 » de BIS_1 §1 n'est pas celle de la CARTE.** BIS_1 applique
  l'estimateur au build (ou au corpus) agrégé, où il redonne exactement la part des paquets fermés ;
  la CARTE l'applique film par film puis somme, ce qui pondère les films (HI_1_10_0 21,9 % contre
  22,2 % ; corpus 43,6 % contre 45,3 %). Les deux sont tautologiques. HI_1_4_1 (aucune entrée utile)
  donne un estimateur indéfini (« - ») pour 6,3 % de paquets fermés. Corrigé dans BIS_1 §1.
- D-66 **Ratchet de taille de `archlint`** : `film_file_size_test.go` balaie tous les `.go` sous
  `film/`, tests compris (500 lignes). La critique n° 2 relevait deux sondes au-delà
  (`campagne_bis1_research_test.go` 616 L, `campagne_bis2_vehicules_research_test.go` 571 L). Elles
  sont découpées le 2026-10-02 à 00 h 27 (nouveaux `campagne_bis1_juge_research_test.go`,
  `campagne_bis2_lecteurs_research_test.go`) ; plus grand fichier de la campagne relevé par `wc -l` :
  480 lignes. `go test ./internal/archlint/` n'est **pas** rejoué dans ce document : item ouvert du
  gate de phase 1.

- D-67 **Signature du localisateur figée sur le slot 123** (2026-10-02, transmis par la session du
  chantier de suite d'audit ; enquête `.ai/V7.5/film_re/ENQUETE_SCAN_SEPTEMBRE_2026-10-02.md` sur
  `origin/feat/suite-audit-decodeur` @ `90014fe79`, non vérifiée ici). Après la vague J11.4, 19,1 %
  des kills de septembre sont servis par le `scan` de killsource, dont 85,3 % dans 19 matchs à
  objectif porté unique (Oddball, One Bomb, One Flag, une variante Squad Battle). Cause racine : le
  localisateur des paquets à événements (`marchLocateStrict` et sa copie dans
  `facts/killsource/walk.go`) cherche un delta « high-frequency » de 35 bits sur le SEUL slot 123,
  alors que les images-clés déclarent d'autres objets de cet archétype (124, 126 à 129) qui portent
  le delta dans ces modes ; le paquet n'est pas localisé et ses morts tombent au repli. Correctif
  proposé par l'enquête : accepter tout slot lié à cet archétype (lu au registre du film), ordre
  « slot 123 strict -> autre slot high-frequency -> repli largeur libre », sur LES DEUX sites ;
  mesuré 362 / 403 kills `scan` rendus à la marche sur trois films. Seconde cause (Banished
  Narrows) : objet transitoire né entre deux images-clés, liaison non gardée = la région (iii) /
  `ti=3` de cette campagne (lot L8). Annexe : morts de bot toujours étiquetées `scan`.
  **Lien avec la campagne** : c'est très probablement une part de la région (ii) « paquets à
  événements non localisés » (lot L1b, R-L1). Le correctif se poserait UNE fois dans le localisateur
  unifié par LU (vague 2), ou juste après LU. **Statut (2026-10-02)** : CONFIÉ À LA CAMPAGNE, lot
  LS (§6.1 et §6.2), par accord entre les deux sessions, l'utilisateur leur ayant laissé la décision.
  **Vérifié et mesuré le 2026-10-02 (R-LS, `R_LOC.md` §2-§3)** : les deux sites comparent bien au
  littéral 123 (code relu à `fe18bf67c`). L'ordre de l'enquête fait baisser `1c4c63c2` sur le site de
  la cuisson ; l'ordre « 123 strict → fermeture par NEW de tête → signature high-frequency » n'en
  fait baisser aucun (+18 119 / −2 paquets, +169 439 records utiles sains, région (ii) de 48 720 à
  26 427). Killsource (28 films) : 1 080 morts passent du `scan` à la marche, tag, statut, crédit et
  origine inchangés ; la voie (`read_path`) est persistée, donc publiée (D-74). [LS, LU, §6.5]

**Découvertes des recherches préalables (2026-10-02)**

Rendues par les six chantiers du workflow `wf_9088d8bd-e43` (notes `R_COMB.md`, `R_LOC.md`,
`R_NAIS.md`, `R_COMP.md`, `R_VEH.md`, `R_FUSION.md` sous `campagne_grammaire_2026-10-01/`) et par
leurs vérificateurs adverses, puis par l'intégration. Aucune n'est traitée. Les codes d'origine des
notes (RC-n, R-LOC-n, N-n, DC-n, R-VEH-n, D-F-n) sont rappelés entre parenthèses.

*Mesure et instrument*

- D-68 **Dénominateur fixe : quatre valeurs, une par chantier** (RC-3). R-COMB, R-LS, R-L1 (d) et
  R-L4 ont chacun pris le maximum de LEURS marches (HI_1_13_0 : 2 880 403, 2 931 799, 2 850 786 ;
  R-L4 ne monte que sur les six films des formats 24-25). Le maximum consolidé par film vaut
  2 959 104 sur HI_1_13_0 et 7 176 150 sur le corpus (§6.5.3). Aucun chiffre de chantier n'est donc
  publié sur le même dénominateur qu'un autre ; la comparaison passe par §6.5.3.
- D-69 **L'oracle L1 dérivé dans un monde où L9 est posé fait perdre 2 381 paquets à `4f77afc1`**
  (RC-1, mesuré) : 18 à 19 liaisons de plus (`ti` 41, 42, 35, 40, 0 ; chunks 27 à 54). Dérivé sans
  L9, il ne perd que 5. L'oracle L1 n'est pas une borne stable en combinaison (réserve D-7).
- D-70 **L1 et L2 se recouvrent** (RC-2). Recouvrement des gains mesuré (sur `81c02726`, marginale
  de L1 = 0 sous L2 contre +2 884 seul ; sur HI_1_13_0, la marginale de L1 remonte à +16 137 sains
  quand L2 est retiré) ; le mécanisme (naissances ratées parce que `ti=43` n'est pas porté) est
  ESTIMÉ, non vérifié record par record.
- D-71 **Trois films HI_1_13_0 dépassent 95 % sous la combinaison** (RC-4) : `f75e7053` 98,3 %,
  `c75f33b8` 97,2 %, `81c02726` (hors corpus) 98,2 %, sur le fixe de R-COMB.
- D-72 **L8 a une marginale négative sur `084a804d`** (−73 sains dans la combinaison) (RC-6). « Une
  seule liaison d'oracle de plus » (410 contre 409) est déduit des comptes ; les oracles ne sont pas
  comparés liaison par liaison.
- D-73 **Les deux sites du localisateur n'ont déjà pas le même ordre** (R-LOC-1) : la cuisson
  (`debutDeLaListe`) n'a pas de repli à largeur libre ; killsource et `marchLocalise` en ont un.
  L'enquête de suite d'audit et D-67 supposaient une seule règle.
- D-74 **La voie de lecture d'une mort est publiée** (vérificateur de R-LS) :
  `internal/sync/killcollector/collector_batch.go:72` écrit `ReadPath` dans `match_kill_events`. Un
  lot qui fait passer une mort du `scan` à la marche change donc une colonne publiée : montée de
  `killsource.Rev` et backfill killsource DUS, pas conditionnels.
- D-75 **Le slot 123 n'est lié à `ti=4` que dans une partie des chunks des films à objectif porté**
  (R-LOC-2 ; `c75f33b8` 5 sur 25, `d9781168` 11 sur 38) ; 392 signatures high-frequency tombent sur
  des slots liés en cours de chunk seulement (R-LOC-3 ; `1c4c63c2` 359). Liste des slots liés
  incomplète dans la note (manquent 3072, 6528 et 4, vérificateur). Justesse non instruite.
- D-76 **`51ebbc0f` : la carte gagne +3 378 paquets sains sous LS, killsource ne change pas**
  (R-LOC-4) ; seule l'égalité des comptes et des voies est vérifiable dans les TSV. Non instruit.
- D-77 **Le repli à largeur libre du slot 123, ajouté à la cuisson, détruit des fermetures**
  (R-LOC-5) : net −14 141 paquets sains (16 673 sains perdus en brut), 12 films en baisse,
  `4f77afc1` de 22 910 à 11 051 fermés. Sa valeur dans killsource n'est pas remesurée.
- D-78 **L'ordre entre « fermeture par NEW de tête » et « signature high-frequency » n'est pas décidé
  par la grammaire** (R-LOC-6) : l'ordre de l'enquête bat l'ordre retenu en records utiles sains
  sur `d9781168` et `c46ef9d1`, mais fait baisser `1c4c63c2` ; l'ordre retenu est le seul mesuré
  sans baisse.
- D-79 **LS gagne plus que la borne de L1b mesurée sur la même région** (arithmétique sur
  `R_LOC.md` §3.1 et BIS_1 §4) : +18 119 paquets sur la région (ii), contre +11 540 nets pour
  l'oracle (ii) de bis 1. La borne de bis 1 n'était donc pas un majorant de tout correctif de la
  région (ii). Cause non instruite (définitions des deux mesures à confronter).
- D-80 **La vue C se resynchronise d'elle-même** (N-2) : 15,5 % des paquets sains fermés après
  terminateur se ferment encore depuis un départ décalé de 1 à 24 bits. « Fermé au bit près » est
  un témoin plus faible que supposé (D2, L0).
- D-81 **Cause de la faible prédiction de l'allocateur avant HI_1_12_0 : ouverte** (N-6, D-8
  réfuté) ; hypothèse non vérifiée : une table de pools propre à ces builds.
- D-82 **`1c4c63c2`** : l'allocateur prédit 65,5 % des NEW propres des paquets sans liste
  d'événements, mais 5,1 % de tous ses NEW propres (N-4) : les NEW lus dans ses paquets à
  événements sont probablement souvent faux. Non étudié.
- D-83 **Invariant candidat pour le juge** (N-5) : le mot de 32 bits des 441 DEL lus juste avant un
  rejet fermé est non nul 441 fois sur 441, alors que l'écrivain l'écrit à 0 hors archétype `0x10`.
  Non mesuré sur les autres DEL. [L0]
- D-84 **Les occurrences de (iii') sont concentrées 16 à 255 bits derrière l'en-tête rejeté** (N-7 ;
  64,5 %, témoin ≥ 256 bits 84,2 %) : hypothèse d'une référence d'entité dans un corps de composant,
  non vérifiée.

*Grammaire et code (constats, aucune correction faite)*

- D-85 **`traverseComponentLoop` ignore en silence les bits de masque au-delà du dernier composant de
  l'archétype** (DC-1) : un masque impossible ne désynchronise pas le record ; 1 017 records
  coupables G1/G3 en portent un. [L0 : invariant violé à la lecture]
- D-86 **`ecs_table.tsv`** (DC-2) : `ti=3 i1 high-frequency` porte `deser_addr = FUN_14076d034`
  (lecteur de `ti=4 i0`) ; le bon lecteur est `FUN_142ed4880`. La remarque « KEYFRAME : 26 bits » de
  `ti=4 i0` est à revoir. [L8]
- D-87 **`equipment-charges-used-component`** (`ti=37 i27`, version-31) est absent de HI_1_13_0 et du
  dispatch Go (renommé `-remaining`) (DC-3).
- D-88 **`spawn-filter-type` cas 2** (DC-4) : le Go lit `readQuantStat(1)`, le jeu lit
  `R(1)[0→R(13)]` puis `R(6)` (`FUN_142b6ee08` / `FUN_142b67f08`). Presque jamais exercé (+1 / −1).
- D-89 **Un garde-fou « dispatch par table » est constructible hors ligne** (DC-5) : les fonctions
  d'enregistrement statiques (`FUN_14064dd28`, `+0x4754 = ti`, 54 fonctions) donnent la table de
  chaque (ti, index) ; index compactés quand un enregistrement conditionnel est inactif. [L8]
- D-90 **Des images-clés du moteur ne ferment pas sous la grammaire complète** (DC-6 ; 15 / 41 sur
  `51ebbc0f`, 1 / 68 sur `1c4c63c2`) et aucun décalage unique ne les ferme.
- D-91 **Un crochet de composant voit aussi les marches d'essai des localisateurs** (DC-7) : les
  comptes par crochet de la marche delta en sont gonflés ; non publiés comme preuve.
- D-92 **Les vieux builds (HI_1_4_1, version-31, version-33) lisent au moins un bit de trop** :
  - `ti=2` (R-L3, mesuré) : un bit de moins ferme 98 / 98 images-clés du moteur ; la position est
    entre le début d'`i4` et l'entrée d'`i10`, NON discriminée entre `i4` et `i9` (vérificateur) ;
  - `ti=40` (R-VEH-3, mesuré) : 0 / 2 036 voisins fermés sous la lecture des formats 24 à 27, état
    par défaut juste, largeur fautive non identifiée ;
  - une cause commune (composant ou champ partagé) est SUPPOSÉE, non instruite. [L3b, L4b]
- D-93 **La portée `DAT_144e61ea0` plus le chemin `i0` de l'écrivain ferment aussi le bipède
  `ti=35` en image-clé** (R-VEH-1) : 142 → 2 006 / 2 008 voisins (format 27) ; aucun archétype du
  format 27 ne baisse. NON CONFIRMÉ comme « critère de bascule de `PorteeBaseline` rempli » : le
  critère écrit (591 records bornés de R7 + non-régression delta pour `GrammaireEcrivainI0`) n'a pas
  été rejoué, et la seule variante delta qui pose les deux perd 9 573 paquets. Sans 8/3, `ti=35`
  baisse sur `111fa685` (15 → 10) et `11de8353` (11 → 6). [LK]
- D-94 **Le découpage MPP 8/3 contredit la case vide de `MPPPourFormat`**
  (`profile/build_profile.go:263`, `:323`) (R-VEH-2) : +80 979 / −2 105 paquets sur les six films
  des formats 24-25. L'explication de la note (« fermeture mesurée alors sans la portée ni `i0` »)
  est démentie par ses propres TSV : 8/3 seul fait déjà monter ces images-clés, et fait BAISSER les
  formats 20-21 qui partagent la case. Cause probable de l'ancienne baisse 246 → 182 : ce
  regroupement 20-21 + 24-25 (supposé). Pertes non jugées. [LM]
- D-95 **Records d'image-clé à voisin non consécutif** : ils s'arrêtent 108·k bits avant la
  frontière, sans en-tête lisible (R-VEH-4) ; hypothèse : des records d'en-tête seul. [L9, D-51]
- D-96 **Le relevé d'index `bis2NoterIndex` compte les lectures d'essai de l'inférence**
  (R-VEH-7) : D-54 et les comptes d'index de BIS_2 §3.3 sont à relire.
- D-97 **Six écrivains posent la portée**, dont `FUN_142e31bf8` omis par la note (R-VEH-5,
  vérificateur) ; les NEW du film ne sont pas lus sous elle (mesuré). La documentation de
  `profil_balayage.go` est incomplète pour `FUN_142e2c690`. La valeur modale `n2` de version-31 est
  `0x890`, non `0x8a0` (vérificateur).
- D-98 **`ScanVehicleCreations` produit des créations à châssis inexistant** (`4118381d`,
  `d0b40d0a`) (R-VEH-6) : faux positifs SUPPOSÉS (vérificateur : `d0b40d0a` se relit à l'identique
  sous 8/3 à une ancre décalée, sur un film des formats 20-21 ; réutilisation de slot non exclue).
- D-99 **Témoins décalés d'un bit peu discriminants sur `396cfc92` et `60ae07c4`** (R-VEH-8) : un
  témoin à décalages de 2 à 8 bits serait plus robuste. [L0]

*Outillage, livrables et règles*

- D-100 **Les surcouches de mesure ne compilent plus ensemble** (intégration du 2026-10-02) :
  - sous le tag commun `campagne_overlay`, toutes les sondes taggées se compilent avec la surcouche
    passée ; `r_veh_delta_research_test.go` exige `rvehPorteeNeuf` et `rvehPorteeEtat`, qui
    n'existent que dans `r_veh_overlay/` : `go vet -tags=research,campagne_overlay` est ROUGE avec
    les surcouches de fusion, R-COMB et R-COMP (vert si cette seule sonde est masquée) ;
  - `r_loc_overlay/` ne remplace pas `capture.go` et s'emploie avec `-tags=research` seul (vert
    ainsi) ;
  - les surcouches de R-COMB, R-COMP, R-LS et R-L4 sont bâties sur `fe18bf67c` et remplacent cinq
    fichiers modifiés depuis par J12 (`default_state_ti40.go` 2 lignes, `lecteur_position.go` 2,
    `lecteur_position_exceptions.go` 8, `object_deaths_march.go` 7, `traverse.go` 6) : elles
    réintroduisent les versions d'avant J12 dans la mesure. Seule `r_fusion_overlay_postj12/` est
    post-J12. [§6.0, D10, D17]
- D-101 **Livrables incomplets ou inexacts** (vérificateurs) :
  - R-L1 (a) : les tables du pont (0 / 33) et du décalage (64,5 % contre 84,2 %) ne sont que dans le
    scratchpad du chantier, pas dans `r_nais_tsv/` ;
  - R-L1 (d) : les lignes « -hors-evenements » de `r_nais_scores.tsv` viennent d'un code retiré
    (non reproductibles) et l'en-tête a 8 colonnes pour 10 ;
  - R-COMB : `rcaDerivation` exclut du maximum les marches de diagnostic (+24 utiles sur
    `4f77afc1`, effet < 0,001 point) ;
  - fusion : la ligne 17 de `r_fusion_commandes.tsv` dit « 6 boucles », la mesure en donne 5.
- D-102 **Écarts aux règles de mission** : deux chantiers ont appelé `python3` en ligne de commande
  (R-COMP : script vide, aucun fichier ; R-L4 : remplacement de texte dans une sonde, aucun fichier
  Python créé). Déclarés par les chantiers eux-mêmes (`R_COMP.md` DC-9, `R_VEH.md` §8).
- D-103 **Chemin `.ai/` cité par une sonde** (D-F2, corrigé par le vérificateur) :
  `campagne_bis2_positions_research_test.go:13` cite `mesures_bis2_overlay/` sous un chemin
  `.ai/` complet, que `TestCheminsAiCitesDansLeCodeExistent` attrape ;
  `campagne_bis3_ti3_research_test.go:9` le cite sans préfixe `.ai/` (non attrapé). Supprimer ou
  renommer ce dossier exige de corriger ce commentaire dans le même commit.

## 6. Phase 2 — lots correctifs (écrits à l'étape 5, révisés le 2026-10-02)

Synthèse : `.ai/V7.5/film_re/RAPPORT_CAMPAGNE_GRAMMAIRE_PHASE1_2026-10-01.md`, révisé d'après
`CRITIQUE_COMPLETUDE_1.md` et les mesures bis 1 à 3, puis d'après `CRITIQUE_COMPLETUDE_2.md`
(points N1 à N21, 2026-10-02). **Rien n'est lancé** : chaque lot attend le GO explicite et daté de
l'utilisateur.

Abréviations :
- BIS_1, BIS_2, BIS_3 = `campagne_grammaire_2026-10-01/MESURES_BIS_1.md`, `MESURES_BIS_2.md` et
  `MESURES_BIS_3_POPULATIONS.md` ;
- ANALYSE = `.ai/ANALYSE_MISE_EN_OEUVRE_REPRESENTATION_INTERMEDIAIRE_2026-10-01.md` ;
- SUITE = `.ai/PLAN_SUITE_AUDIT_DECODEUR_FILM_2026-09-25.md` ;
- « composant » = lot qui change un lecteur appelé par le dispatch commun à toutes les marches ;
- « marche » = lot qui change un marcheur (localisation, liaison, image-clé).

### 6.0 Règles communes

**Gain**
- « Borne » = gain maximal d'UN mécanisme, mesuré par un oracle ou par une copie de recherche
  (surcouche `-overlay`, D-48). Ce n'est ni un gain acquis, ni la borne de tout correctif.
- Le gain réel se mesure par la carte de fermeture v2, rejouée APRÈS le lot.
- Pourcentages (règle complétée le 2026-10-02, N7) : numérateur en paquets sains (BIS_1 §7), et
  DEUX dénominateurs publiés côte à côte :
  - **fixe** = par film, le maximum des records utiles lus sur l'ensemble {marche de référence,
    marches des lots déjà fusionnés, marches des oracles mesurés}. Les A/B en surcouche comptent
    comme oracles mesurés : ils ont lu les films. Ce maximum est **recalculé à chaque vague**, car il
    monte quand un lot lit plus loin (D-42). Le gate d'une vague compare la référence et la vague sur
    le MÊME dénominateur recalculé ;
  - **variable** = records utiles lus par la marche elle-même.
  - L'ensemble actuel est celui des 14 marches de bis 1 (HI_1_13_0 : 2 574 513). Il s'élargit dès
    qu'on y verse les marches mesurées de L8 et de L2 : 2 759 700 sur HI_1_13_0 sous `ti=3`, et la
    référence y tombe à 73,1 % (RAPPORT §3, arithmétique sur les TSV).
  - **Mis à jour le 2026-10-02 (recherches préalables, D-68)** : l'ensemble s'élargit aux marches
    des quatre chantiers qui ont lu les films. Maximum consolidé par film : HI_1_13_0 **2 959 104**,
    corpus **7 176 150** (§6.5.3). Le fixe14 ci-dessus est périmé. Le gate de la vague 1 part de ce
    maximum consolidé, recalculé avec les marches des lots de la vague.

**Gate de tout lot qui change une sortie** (lot marqué « sortie : oui »)

1. **Tests et révision.**
   - Les vecteurs de test du lot, écrits d'après l'écrivain, sont verts.
   - `go vet` et les tests des paquets touchés passent, ainsi que `go test ./internal/archlint/`,
     ratchets de J12 compris (§6.0, inventaire J12).
   - **Révision (règle unique, N6)** : sur la branche `feat/campagne-grammaire`, CHAQUE lot qui
     change une sortie monte `grammar.Rev` avec son entrée de `rev_chronique.go`, régénère
     l'empreinte par la commande du dépôt, et `facts.Rev` suit. Ainsi chaque mesure se rattache à une
     révision exacte. Au parc, rien ne se recuit lot par lot : UNE recuisson par vague fusionnée dans
     `feat/v75`, décidée par l'utilisateur (D7). Le parc ne voit que la révision de tête de la vague.
     Plusieurs montées sur la branche et une seule recuisson au parc ne se contredisent pas.
2. **Carte de fermeture v2, rejouée sur les 20 films.**
   - **Aucune baisse sur AUCUN FILM**, en paquets sains ET en records utiles sains : c'est le niveau
     de J11, qui jugeait par film. Le critère par build seul est retiré, car une baisse sur un film
     peut se compenser dans son build.
   - Seule exception admise : une baisse expliquée par une fermeture factice retirée (décision D2).
     Le juge des invariants de l'écrivain (BIS_1 §0) est joué sur les paquets GAGNÉS comme sur les
     paquets PERDUS, et sa part de gains factices est publiée.
   - Le gate liste en plus, film par film, les témoins nommés du lot.
3. **Killsource, pour TOUT lot qui change une sortie, composants compris** (étendu le 2026-10-02,
   N1).
   - Pourquoi les composants aussi : `facts/killsource/walk.go:69` appelle
     `grammar.DecodeFrameRecords`, donc le dispatch commun à toutes les marches. Un lecteur de
     composant corrigé (L2, L3, L4, L6a, L6b, L8) peut changer la sortie killsource. Les lots de
     marche la changent par `marchLocateStrict` (`object_deaths_march.go`), sa copie
     `facts/killsource/walk.go` et `ScanMarchFacts`.
   - **Équivalence killsource** : `equivalence_lecteur_test.go` et la sortie killsource sur les
     20 films. L'équivalence attendue est nulle, ou bien le **delta killsource** est déclaré ligne à
     ligne, comme le gate de J12 l'exige (SUITE, gate J12).
   - `killsource.Rev` est monté si la sortie change. Le backfill killsource du parc qui en découle
     est écrit dans la séquence de production (geste de l'utilisateur, D7).
   - Pour un lot de marche, D-17 (`marchViews = 8` contre 3 vues prouvées) se tranche dans le même
     lot, ou se déclare inchangé.
4. **Performance et mémoire de cuisson**, pour tout lot qui lit le bloc de type 1 en production (L1,
   L10).
   - Ce bloc pèse 343 019 octets par chunk (en-tête de `type1_datums.go`) ; s'y ajoute le suivi de
     l'allocateur.
   - Mesure : durée par étape (`replay/observe.go`) et pic mémoire (sentinelle `filmproc`), sur trois
     témoins et un BTB, avant et après le lot.
   - Plafond recommandé, soumis à l'utilisateur (D11) : **+10 % de durée de cuisson et +10 % de pic
     mémoire au plus**, sur ces trois témoins et ce BTB.
   - Aucune cuisson en lot.
5. **Garde-fou de la recopie du pilotage**, pour tout lot qui change le pilotage de
   `decodeFrameParRangs` / `debutDeLaListe` (L1b, L1a).
   - `TestFrameClosureDetailleeRendLaCarteDeFrameClosure` doit compter ≥ 1 paquet à événements
     LOCALISÉ et ≥ 1 NON localisé par bobine. Aujourd'hui, seule la branche non localisée est
     prouvée : `listes_non_localisees=27` sur `ks_000d5950` (D-61). Item d'extension : L0.3.
   - La recopie de la sonde `campagne_marche_research_test.go` suit dans le même lot.
6. **Gate de corpus.** `make replay-corpus-gate` sans perte, **banc de vérité**
   (`internal/replayverite`) et références `replay-equiv`, tous joués contre la tête **POST-J12**
   (références régénérées après la fusion de J12). Après chaque vague fusionnée, les références
   `replay-equiv` et les goldens de fermeture (`frame_closure.golden`, `keyframe_closure.golden`)
   sont re-figés AVANT le lot suivant (§6.3, D-RI).
7. **Pas de cuisson en lot.** Aucune cuisson ni recuisson en lot sans accord explicite de
   l'utilisateur.

Revue adversariale en fin de lot pour les lots à risque : L1, L4, L8, L9, LU.

**Où tournent les gates.** Sur CE PC (worktree `LevelUp-wt-campagne-grammaire`). La vague J11.4 tourne
sur l'AUTRE PC de l'utilisateur (le PC principal) : aucune concurrence de RAM ni de CPU. Le jeu de
données des deux PC diffère (`J12_RECOUPEMENTS_2026-10-01.md`) : aucun compte de parc ne se lit ici.

**Inventaire des recoupements avec J12** (fait le 2026-10-02 par
`git diff 8b894a677 bc0e2511a`, lecture seule)
- Branche : `origin/feat/suite-audit-decodeur-j12`, tête `bc0e2511a` (2026-10-01 19:09), 29 commits
  au-dessus de la base commune `8b894a677`. La base de la campagne, `69564ef7d`, contient
  `8b894a677`.
- Elle est **visible localement comme référence distante**, contrairement à ce que disait la version
  précédente (D-60). Le diff est celui de cette référence telle qu'elle a été récupérée ; une tête
  plus récente reste possible.
- J12 touche 1 453 fichiers du dépôt.

Ligne par ligne, fichier de chaque lot → touché par J12 ?

| Lot | Fichier (sous `film/internal/` sauf mention) | J12 | Item et nature |
|---|---|---|---|
| L1 | `grammar/object_deaths_march.go` (`marchLocateStrict`) | **oui**, +3/−4, `ef7f4db1a` | J12.1 : `sort.SliceStable` l. 78 → `slices.SortStableFunc` |
| L1 | `grammar/movement_states.go` | **oui**, +5/−10, `ef7f4db1a` | J12.1 : tri l. 385 et `go fix` |
| L1, L7 | `grammar/frame_infer.go` (`decodeInferLoop`) | **oui**, +6/−6, `95d8d8f91` | J12.5 : doc comment de `inferUnboundArchetype` recollé |
| L1, L10 | `grammar/type1_datums.go` | **oui**, +2/−2, `b830c8934` | J12.1 `go fix` |
| L1 | `grammar/debut_de_liste.go`, `frame_closure.go`, `frame_closure_classement.go`, `keyframe_datums.go` | non | — (les `filmdec:` des messages d'erreur de `frame_closure.go` restent) |
| L1 | `facts/killsource/walk.go` (copie de la marche) | non | mais `facts/killsource/decode.go` +18/−12 (`9e3af163d`, J12.3 D-4), `killsource_rev.golden` et `killsource_perimetre.golden` régénérés |
| tous (gate 3) | `facts/killsource/world.go` ; `facts/killsource/diagnostics.go` ; `facts/killsource/equivalence_lecteur_test.go` | **oui**, +15/−5 (`62bd3e854` J12.1, `9e3af163d` J12.3) ; neuf, +32 (`9e3af163d` J12.3) ; +4/−4 (`b830c8934` J12.1) | ajoutés le 2026-10-02 (reste du point 1 de la critique n° 1) : l'équivalence du gate 3 se rejoue sur la version post-J12 de ces fichiers |
| L1 | `grammar/frame_harvest.go` (`decodeFrameParRangs`) | **oui**, +1/−1 | J12.1 `go fix` (`for range`) |
| L9 | `grammar/keyframe_world.go` | non | — |
| L9 | `grammar/keyframe_closure.go` | **oui**, +4/−2 | J12.1 |
| L2 | nouveau `grammar/components_device_ti43.go`, `capture.go`, `testdata/ecs_table.tsv`, `ecs_widths_guard_test.go` | non | — |
| L2, L8 | `grammar/dispatch_biped.go`, `dispatch_item.go` | **oui**, +1/−1 chacun | J12.1 |
| L8 | `grammar/registry.go` ; `components_probe.go` | **oui**, +11/−16 ; non | J12.4 / J12.1 |
| L3 | `grammar/dispatch_player.go` ; `components_walk_batch9.go` | **oui**, +4/−4 ; +1/−1 | J12.1 |
| L3 | `grammar/components_game_engine.go`, `vitality.go` | non | — |
| L4 | `grammar/composants_vue_b_m4b.go`, `facts/fallback/noms.go`, `registre_filmdec_marche.go` | non | — |
| L6a | `grammar/lecteur_position.go`, `lecteur_position_exceptions.go` | **oui**, +1/−1, +4/−4 | J12.1 |
| L6a | `film/replay/world_object_precision.go`, `film/replay/build_from_film.go` | **oui**, +4/−4, +19/−12 | J12.1, J12.3 (`ctx`) |
| L6a | `grammar/film_context.go` | **oui**, +28/−14 | J12.3 D-4 : plus de `slog` dans `grammar` |
| L0 | `grammar/frame_closure_detail*.go` (neufs) | absents de J12 | J12.7 : production non taguée, appelée par l'instrument seul (statut de `FrameClosure`). Le ratchet `research_tag_test.go` interdit le tag sur un fichier non-test hors `film/research/`. |
| L0 | `film/research/cmd_fermeture/main.go`, `gb1_research_test.go` (modifiés par la phase 1) | **oui**, +2/−2, +3/−1 | J12.1 `strings.SplitSeq` ; J12.2 `errors.Is` |
| L0 | `source/bits.go` (accesseur `Deborde()`) | **oui**, +5/−5 | J12.1 |
| tous | `grammar/testdata/grammar_rev.golden`, `grammar_perimetre.golden` | **oui** | régénérés par les deux branches : à re-régénérer après fusion |
| tous | `grammar/rev.go`, `rev_chronique.go`, `testdata/frame_closure.golden` | non | — |

Conséquences, items J12.1 à J12.7. **État exact (corrigé le 2026-10-02, N9)** : « tous les conflits
sont mécaniques » n'était pas prouvé. Aucun essai de fusion n'a été joué. Deux familles :
- **Mécaniques** (attendu par lecture du diff, non prouvé par un essai) : tris J12.1, `go fix`,
  `errors.Is` / `strings.SplitSeq` J12.2, documentation J12.5.
- **Contraintes de STRUCTURE**, à respecter par les lots qui touchent ces fichiers :
  - J12.3 retire `slog` de `film_context.go` (+28/−14), un fichier de L6a ;
  - J12.4 réécrit `registry.go` (+11/−16), un fichier de L8.

Détail par item :
- **J12.1 (tris, `go fix`)** : conflits textuels attendus mécaniques sur `object_deaths_march.go`,
  `movement_states.go`, `type1_datums.go`, `frame_harvest.go`, les dispatch et les lecteurs de
  position. Après J12, le cliquet `archlint/film_tri_total_test.go` est VIDE : tout
  `sort.Slice*` de production est rouge (les `_test.go` et `film/research/` en sont exclus).
- **J12.2** : ratchet `os.Is*` étendu aux instruments `research` ; `cmd_fermeture` est concerné.
- **J12.3 (contexte, D-4)** : plus aucun `slog` dans `grammar` ni `facts`, et deux ratchets à
  allowlist vide. Le « compteur des liaisons par source » de L1 passe par la couverture ou par un
  diagnostic typé (`film/internal/constat`, paquet créé par J12), jamais par un journal. L6a écrit
  sur la forme post-J12 de `film_context.go` (contexte sans journal).
- **J12.4** : plus de variable de paquet exportée modifiable (ratchet étendu). L8 écrit sur la forme
  post-J12 de `registry.go`.
- **J12.5** : ratchets `TestCheminsAiCitesDansLeCodeExistent` et commandes `film/filmdec/`. Tout
  chemin `.ai/` cité dans un commentaire de lot doit exister.
- **J12.6** : sans recoupement.
- **J12.7** : tous les `*_research_test.go` de la campagne portent déjà le tag (ou
  `research && campagne_overlay`) ; le ratchet est satisfait. `frame_closure_detail*.go` restent non
  tagués.
- **Goldens** : `frame_closure.golden` et `ecs_table.tsv` sont inchangés par J12 ; la carte v2 se
  rejoue donc à l'identique avant et après J12 (attendu, à confirmer).

**Items à la fusion de `feat/v75` post-J12 dans `feat/campagne-grammaire`** (ajoutés le 2026-10-02 ;
fusion FAITE le 2026-10-02, commit `f28a4a816`, `origin/feat/v75` = `95b19e635`) :
- [x] **Essai de fusion AVANT tout développement de phase 2** (N9) — fait : `git merge-tree` contre
  `bc0e2511a` puis contre `95b19e635`, UN seul conflit (`grammar_rev.golden`), le reste fusionné
  seul ; essai complet dans un worktree jetable (`R_FUSION.md`), puis vraie fusion `f28a4a816`.
  **Résultat de l'essai (`R_FUSION.md`, contre `bc0e2511a`, sans commit, vérifié par un contexte
  adverse)** :
  - conflits : UN (`grammar/testdata/grammar_rev.golden`, ligne `.3` régénérée des deux côtés :
    base `4bc8e05a`, campagne `3cbd64ac`, J12 `558380db`), confirmé indépendamment par
    `git merge-tree --write-tree` ; `cmd_fermeture/main.go` et `gb1_research_test.go` fusionnés
    seuls (les `strings.SplitSeq` / `errors.Is` de J12 et le mode v2 coexistent). Résolution : version
    J12 du golden puis porte du dépôt ; empreinte `14b3a79d…` à révision constante (valeur rapportée
    par l'essai, non recalculable sans `go` par le vérificateur ; retrouvée par la vraie fusion,
    journal §4) ;
  - ruptures dues à J12 : quatre sondes (`campagne_bis1_research_test.go:275`,
    `campagne_bis2_chassis_research_test.go:73`, `campagne_bis3_research_test.go:90`,
    `campagne_mesures_research_test.go:88`) prenaient l'adresse de `profile.QuantRangeCEBiped`,
    devenu un accesseur (J12.4). Ce sont des ruptures de COMPILATION sous `-tags=research`, pas des
    violations de ratchet (vérificateur) ; le pas CI de la base (`ci.yml:253`, vet `research` du
    paquet `grammar`) les aurait déjà vues. Patch `r_fusion_patchs/r_fusion_ratchets_j12.patch`
    (une ligne par sonde, valeur identique) ; la vraie fusion a appliqué la même correction.
    Deux sondes neuves des recherches préalables avaient la même ligne
    (`r_veh_chassis_research_test.go:192`, `r_comb_research_test.go:184`), corrigées à
    l'intégration ;
  - ratchets de J12 : 11 / 11 PASS sur l'arbre fusionné (9 créés par J12, 2 resserrés : les deux
    `TestTriTotal*` datent de J10.1) ; lint ratchet 0 issue sur `grammar` et `research` ; aucun autre
    fichier de la campagne en défaut (14 appels `sort.*`, tous dans des `_test.go`, exclus du
    cliquet) ;
  - gates de l'essai : `gofmt` vide ; `go vet` film et `go vet -tags=research ./...` (module) exit 0
    avec le patch ; `go test` film 17 paquets ok ; `-race TestDeuxFilmsEnParallele` ok ; `archlint`
    rouge sur `TestNoExpiredTODO` seul (quatre pas sans journal conservé : `gofmt`, vet sans tag,
    `archlint -skip`, `-race`, rapportés et non contre-vérifiés) ;
  - `TestNoExpiredTODO` : `TODO(expiry:2026-10-01)` échu dans
    `internal/api/handlers/json_huma_coverage_test.go:34`, même blob dans `fe18bf67c`, `bc0e2511a`
    et `origin/feat/v75` : rouge sur ces trois têtes à partir du 2026-10-02 (tous les chantiers l'ont
    vu). SOLDÉ hors campagne par `feat/v75` (`2f1e7b98b`, échéance re-datée au 2027-01-01, décision
    utilisateur du 2026-10-02), arrivé par `da7c2c764` : `archlint` complet vert sur la tête ;
  - découvertes de l'essai : D-F1 (règle 17 sur `frame_closure_detail.go`) SOLDÉE par `32ba9078d` ;
    D-F2 réduite à un commentaire (D-103) ; D-F3 (équivalence des mesures sur films) SOLDÉE par la
    carte v2 rejouée (item ci-dessous) ; D-F4 = `TestNoExpiredTODO` (ci-dessus).
  Énoncé d'origine : : `git merge-tree` de `feat/v75`
  post-J12 contre la branche, lecture seule. Il liste les conflits réels et confirme ou infirme la
  classe « mécanique » ci-dessus. Tant que J12 n'est pas fusionné, un essai contre
  `origin/feat/suite-audit-decodeur-j12` peut servir d'aperçu ; il ne remplace pas l'essai contre
  `feat/v75` post-J12.
- [~] **Re-synchroniser les copies de la surcouche** (N3, D10) — copies post-J12 produites par l'essai
  de fusion (`r_fusion_overlay_postj12/`, fusion à trois voies sans conflit ; seul écart : cinq
  boucles du `go fix` de J12), intégrées avec les recherches préalables. Toute mesure en surcouche de
  la phase 2 part de `r_fusion_overlay_postj12/overlay_campagne.json` ; `mesures_bis2_overlay/` reste
  pour rejouer à l'identique les chiffres de la phase 1. Énoncé d'origine : `capture.go`,
  `lecteur_position.go` et `lecteur_position_exceptions.go` de `mesures_bis2_overlay/` sont des
  fichiers ENTIERS, substitués par chemin absolu. J12 modifie `lecteur_position.go` (+1/−1) et
  `lecteur_position_exceptions.go` (+4/−4). Sans re-synchronisation depuis les fichiers post-J12, une
  mesure en surcouche annulerait ces changements. Reporter les ajouts de recherche sur les fichiers
  post-J12, puis rejouer les contrôles de la surcouche (référence = carte v2).
  **Reste ouvert (intégration du 2026-10-02, D-100)** : les surcouches des chantiers R-COMB, R-COMP,
  R-LS et R-L4 (`r_comb_overlay/`, `r_comp_overlay/`, `r_loc_overlay/`, `r_veh_overlay/`) sont
  bâties sur `fe18bf67c` et réintroduisent dans la mesure les versions d'avant J12 de cinq fichiers ;
  sous le tag commun `campagne_overlay`, `go vet` est rouge avec les surcouches de fusion, R-COMB et
  R-COMP (sonde `r_veh_delta_research_test.go` sans ses crochets) et vert avec celle de R-L4. Quatre
  `overlay_campagne.json` pointant sur ce worktree ont été écrits à l'intégration. À faire avant
  toute mesure en surcouche de la phase 2 : UNE surcouche post-J12 qui réunit les crochets de tous
  les chantiers (base `r_fusion_overlay_postj12/`), ou un tag par chantier (D17).
- [x] `go test ./internal/archlint/` (ratchets de J12 compris, et ratchet de taille D-66), et mise en
  conformité des instruments de la campagne (tris à comparateur total, `errors.Is`,
  `strings.SplitSeq`, aucun `slog` dans `grammar`) — VERT et complet sur l'arbre fusionné ; seule
  non-conformité : quatre sondes suivaient la variable `profile.QuantRangeCEBiped`, devenue un
  accesseur par J12.4, corrigées dans la fusion (les tris des sondes sont dans des `_test.go`, que le
  cliquet de tri exclut). Règle 17 (commentaires) appliquée au code neuf de production et de l'outil
  (`32ba9078d`).
- [x] Re-régénérer `grammar_rev.golden` et `grammar_perimetre.golden` par la commande du dépôt —
  `grammar_rev.golden` régénéré à révision constante `grammar-2026-09-27.3` (empreinte `14b3a79d…`) ;
  `grammar_perimetre.golden` inchangé, tests de révision verts.
- [x] **Équivalence des mesures après J12** (item ouvert de l'essai de fusion) : la carte v2 rejouée sur
  les 20 films avec l'outil construit sur l'arbre fusionné rend des TSV IDENTIQUES à ceux de la
  phase 1 (seuls le pic mémoire et la durée diffèrent) : les chiffres de la phase 1 valent après J12.

**Méthode de la surcouche (`go test -overlay`), règles recommandées (D10)**
- Outillage de MESURE seulement : aucune preuve de gate ne repose sur elle. Les gates se jouent sur
  le code du lot.
- Hors CI : les fichiers taggés `research && campagne_overlay` échappent au
  `go vet -tags=research ./...` de la CI. Personne d'autre ne les compile.
- Les copies sont re-synchronisées à la fusion de J12 (item ci-dessus), et supprimées quand le lot
  correspondant est fusionné (L2, L6a, L6b, L8 ; `capture.go` quand le dernier des quatre l'est).

**Mesures ouvertes avant la phase 2** (recherche, aucun fichier de production)
- [x] **R-COMB** (N10) : mesurer L1 (oracle (i)+(ii)) + L8 + L2 + L9 ENSEMBLE sur HI_1_13_0, en copie
  de recherche (surcouche + oracle de la sonde). Les leviers ne sont mesurés que séparément ; leur
  combinaison n'est ni mesurée ni estimée. La phrase « la phase 2 seule n'atteindra pas le
  déclencheur » reste une estimation tant que R-COMB n'est pas joué.
  — Fait le 2026-10-02 (`R_COMB.md`, 20 films + `81c02726`, avec L6a et L6b en plus) : résultat et
  reformulation de la phrase au §6.5.1 et §6.5.3.
- [ ] **R-COMB-2** (proposée le 2026-10-02) : la même combinaison, plus LS, L3a, L4a, LP et LM
  (§6.5.2), sur la surcouche unique post-J12 (D-100). Sans elle, la phrase « la phase 2 seule
  n'atteindra pas 95 % » n'est plus décidable pour HI_1_13_0 (LS n'était pas dans R-COMB) ni pour
  les formats 24-25 (LM non combiné).
- [ ] **Part de D-44 sur le corpus** (N14, D-63) : avec L0 (L0.2).

**Chaîne qui bloque J12** (sources : SUITE §5 et journal ; `J12_RECOUPEMENTS_2026-10-01.md` pour les
heures transmises, non vérifiées ici)

`J11.4` (vague locale : `backfill-replay` → `usage-summary` → `pad-tiers --force` → `killsource` →
`bomb-stats` → `vehicle-takes` → `healthcheck`, en cours sur l'AUTRE PC depuis le 2026-10-01 17 h 47,
fin estimée vers 5-6 h le 2026-10-02) → `J11.5` (vérification visuelle par l'utilisateur) → fusion de
J12 dans `feat/v75` (preuve « zéro différence » ~1 h, CI ; visée fin de matinée du 2026-10-02) → revue
adversariale finale du chantier, qui peut encore toucher `grammar`.

`J11.6` (fusion J11 → `feat/v75`) est **déjà fait** (`8b894a677`).

Sources non réconciliées (reste du point 3 de la critique n° 1) : ANALYSE §6 dessine la fusion de
J12 AVANT la vague J11.4 ; SUITE dit seulement « la fusion de J12 reste après celle de J11 ». Les
faits transmis (`J12_RECOUPEMENTS_2026-10-01.md`) placent J11.4 en cours AVANT la fusion de J12. La
chaîne ci-dessus suit les faits transmis ; sur ce point, le schéma de l'ANALYSE §6 est périmé.

**Option « prouver maintenant contre les références J11.2, puis reconfirmer après J12 »** (pesée)

- Pour :
  - J12 est prouvé neutre : document publié identique à l'octet sur 20 films. Seuls quatre digests
    d'étape changent (diagnostics killsource, drapeau, crâne ; 2 échantillons au repos sur 86 818,
    départage `compareSample`), et `frame_closure.golden` ne bouge pas.
  - La carte v2, le juge des invariants et le banc rendent donc dès maintenant un verdict qui
    survivra à J12, sur CE PC, sans attendre la fusion.
  - Les références J11.2 (`4c3fd0dbb`) sont ancêtres de la base de la campagne.
- Contre :
  - le gate de corpus et l'équivalence killsource se jouent deux fois (heures de machine) ;
  - les conflits J12.1 se résolvent quand même à la fusion ;
  - la revue finale du chantier peut toucher `grammar` ;
  - une preuve « avant » n'autorise aucune fusion.
- **Recommandation** (précisée le 2026-10-02) :
  - MESURER maintenant, en recherche (carte v2, juge, surcouche, R-COMB, recherches R-*) ;
  - développer les lots de la vague 1 après l'essai de fusion (item ci-dessus) ;
  - jouer les gates 3, 4 et 6 UNE fois par lot, après la fusion de `feat/v75` post-J12 dans
    `feat/campagne-grammaire` (`feat/v75` a raison en cas de conflit) ;
  - ne rien fusionner avant J12.

**Montée de `grammar.Rev` contre la recuisson J11.4**
- Sur la branche, chaque lot « sortie : oui » monte `grammar.Rev` (gate 1), et `killsource.Rev` si
  sa sortie killsource change (gate 3, composants compris).
- Selon l'ADR 0034, la fusion d'une vague dans `feat/v75` rend « à recuire » le parc que J11.4 recuit
  en ce moment : verdict `redecoder` et backlog killsource rouvert.
- Règle (sans contradiction avec le gate 1, N6) : plusieurs montées par vague SUR LA BRANCHE, UNE
  recuisson du parc PAR VAGUE fusionnée, décidée avec l'utilisateur après la fusion de la vague (D7).
  Deux vagues sont prévues (§6.3, D-RI), donc deux recuissons.

### 6.1 Lots, classés par gain mesuré

Mis à jour le 2026-10-02 d'après les recherches préalables (§6.5). « Sains » = paquets fermés sans
invariant contredit (D2). Les lots marqués « proposé » ne sont pas admis : ils attendent la
décision de l'utilisateur (§6.3, D14 et D15).

| Lot | Nature | Titre | Gain mesuré (borne ou A/B) | Sortie | Taille |
|---|---|---|---|---|---|
| **L8** | composant | `ti=3` : `low-frequency` porté, `high-frequency` routé par archétype (BIS_3 §6) | **+30 618 / −10 paquets, +234 454 utiles** (A/B surcouche) ; 34 gains contredits ; 3 films HI_1_13_0. Marginale dans la combinaison R-COMB (HI_1_13_0) : +32 108 sains. Prérequis R-HOM levé : grammaire de `i1` établie | oui | S-M |
| **L1** | marche | Lire les naissances, régions (i) et (ii) | borne oracle (i)+(ii) **+28 168 nets, +360 182 utiles** (L1a seul +16 383 / +196 804 ; L1b seul +11 540 / +162 964) ; dans la combinaison R-COMB : +24 916 (ordre D-RI) ou +26 620 nets (L1 dérivé sans L9). **L1a sous condition par film (R-L1 (d)) : +15 070 paquets sains, +159 864 utiles sains, 0 film en baisse** (seuil choisi sur le corpus). L1b : voir LS et §6.5.2 (proposé : sortir de la vague 2) | oui | L |
| **L2** | composant | Dispositifs `ti=43` (grammaire T7) | **+18 105 / −12 paquets, +143 314 utiles** (A/B surcouche) ; HI_1_13_0 hors cadre −5 872 ; `81c02726` +3 580 ; HI_1_10_0 382/401 gains factices. Recouvre L1 (D-70) : marginale dans la combinaison HI_1_13_0 +2 341 sains (seul : +6 332) | oui | M |
| **LS** | marche | Signature du localisateur : tout slot lié à l'archétype « high-frequency », ordre propre à chaque site (D-67, R-LS) | **carte v2 : +18 119 / −2 paquets, +18 086 sains, +169 439 utiles sains, 0 film en baisse** (ordre « 123 strict → fermeture par NEW de tête → signature high-frequency » sur la cuisson) ; killsource : 1 080 morts du `scan` à la marche sur 28 films (dont 362 / 403 sur les trois films de l'enquête), valeurs inchangées, voie publiée (D-74). Non combiné avec L1 (D-79) | oui | S |
| L6a | composant + donnée de carte | Largeurs par index de plage (T4-C3), Live Fire seulement, plus le site `ti=41 i0` (R-P3) | Live Fire `0797ce72` +3 269 / −3, `60ae07c4` +1 806 / −49 (contexte de production) ; site `ti=41 i0` en plus : `0797ce72` +495 sains, `60ae07c4` −28. **Hors Live Fire : réfuté** (13 films, +15 / −18, −15 sains, R-L6) | oui | M |
| **LM** (proposé) | composant (profil) | Découpage MPP 8/3 sur les formats 24-25 (R-VEH-2, D-94) | **+80 979 / −2 105 paquets** sur 6 films (2 646 gains contredits) ; avec les composants `ti=40` : +83 050 / −2 092. Pertes NON jugées (`1c4c63c2` −1 606). Formats 20-21 en baisse sous 8/3 ; format 27 −118 413 (témoin négatif) | oui | à estimer |
| L3a | composant | Moteur `ti=2` / `ti=0` + helper `FUN_140d580d0`, portage HI_1_13_0 | **A/B mesuré (R-L3) : +3 575 paquets sains, +53 521 utiles sains, 0 sain perdu, 0 film en baisse** (remplace l'estimation ≤ 4 331) | oui | S-M |
| L3b (proposé) | composant | Vieux builds : un bit de trop dans `ti=2` | images-clés du moteur 0/98 → 98/98 avec un bit de moins ; position NON discriminée entre `i4` et `i9` (vérificateur) ; delta +0 / +0 / +4 sains | oui | S |
| L9 | marche (image-clé) | Marche d'image-clé toutes générations (BIS_3 §2) | oracle +1 784 / −1, +45 694 utiles ; 20 gains contredits. Dans la combinaison : marginale −1 112 sains (ordre D-RI, artefact de l'oracle L1, D-69) ou +1 078 (L1 dérivé sans L9) | oui | S-M |
| **LP** (proposé) | marche (image-clé) | Désavouer une déclaration d'image-clé que le bloc de type 1 du même chunk dit non vivante (R-P3) | **+699 / −8 paquets, +694 sains, 0 sain perdu, +7 800 utiles sains** ; G1 694 → 388 ; `bf15f7ab` 93,40 → 95,85 % (son fixe). Exige le bloc de type 1 en production (gate 4) | oui | S |
| L4a | composant | Véhicules `ti=40` en delta : 16 composants, porte posée (loi de l'écrivain) | **A/B mesuré (R-L4) : +1 436 / −0 paquets, 35 contredits (+1 401 sains), +26 843 utiles**, 0 film en baisse (remplace l'estimation ≤ 1 666) | oui | M |
| L4b (proposé) | composant | Véhicules `ti=40` en image-clé (porte lue par châssis) | 14 → 3 058 / 3 058 voisins fermés (formats 24 à 27), à condition de LK et LM ; formats 20-21 : 0 / 2 036 | oui | M |
| LK (proposé) | composant (image-clé) | Lecture d'image-clé sous la portée `DAT_144e61ea0` + chemin `i0` de l'écrivain, tous archétypes (R-VEH-1, D-93) | `ti=40` 3 058 / 3 058 (avec LM sur 24-25) ; `ti=35` (format 27) 142 → 2 006 / 2 008 voisins. Non confirmé comme bascule de `PorteeBaseline` (critère écrit non rejoué) ; image-clé seulement (en delta, la variante qui pose portée ET `i0` perd 9 573) | oui | M |
| L6b | composant | Sites de position au jeu : `flock-position`, `tacmap-displayasset` | +416 / −10 ; +77 / −16 (A/B surcouche, instruments) ; marginale dans la combinaison : +241 sains (HI_1_13_0) | oui | S |
| L7 | marche | NEW sur slot occupé | ≤ 137 paquets (mesuré) ; R-P3 y rattache le NEW refusé après une liaison à masque impossible (`11de8353`) | oui | S |
| LU | marche (structure) | Unifier les localisateurs jumeaux (`marchLocateStrict` et sa copie `facts/killsource/walk.go`), zéro différence, premier lot de la vague 2. Doit porter un paramètre d'ordre (les deux sites n'ont pas le même ordre, D-73) | 0 (sortie identique exigée) | non | S-M |
| L0 | instrument | Invariants de l'écrivain, sortie de vue B, classements corrigés | 0 fermeture ; requalifie 8 388 fermés, 2 832 causes, 12 854 « réalloués », 3 560 « naissances non lues » sur `81c02726`, 1 570 arrêts « bloc 0xbc ». Ajouts proposés le 2026-10-02 : « sortie de vue B par rejet ⇒ non fermé » (4 598 requalifiés, dont 1 008 jugés sains aujourd'hui), « masque au-delà de l'archétype » (D-85), candidat « mot de DEL non nul » (D-83) | carte seulement | M |
| L10 | garde-fou | Cardinal du bloc de type 1 ≠ 8 191 : refusé ou daté, largeur lue sur le cardinal | 0 sur le corpus (mesuré : 8 191 partout) | non sur le corpus | S |
| ~~L1c~~ | — | Naissances lointaines par recherche d'en-tête | **RETIRÉ** (confirmé par R-L1 (a) : les « NEW » de (iii') ne sont pas des records ; R-P6 : aucun filtre d'occurrence général) | — | — |
| ~~L5~~ | — | Bloc `0xbc` de la vue C | **SORTI** : mesuré 6 et 9 paquets sous les deux formes, contre 11 et 10 pour les témoins décalés (BIS_2 §2) ; borne ≤ 953 réfutée | — | — |

Recherches préalables (research, aucun fichier de production) — **toutes jouées le 2026-10-02**
(workflow `wf_9088d8bd-e43`) ; statut détaillé au §6.5.1 :

| Recherche | Objet | Bloque | Statut (2026-10-02) |
|---|---|---|---|
| R-L1 | (a) région (iii') ; (b) les 4 598 paquets fermés après un rejet (D-2, D-56) ; (c) lecture de la vue A des paquets à événements par grammaire ; (d) condition PAR FILM qui sépare les chaînes de tête justes des fausses avant HI_1_12_0 (D-46) | L1b (c), L1a (d) | (a) établi ; (b) établi ; (c) structure établie, gain partiel (estimé) ; (d) établi sur le corpus, seuil choisi sur le corpus (`R_NAIS.md`, `R_LOC.md` §4) |
| R-L3 | bassin `i15` « tout à un » sur HI_1_4_1, v31, v33 (critique, point 30) ; porteur, méthode et gate au §6.2 | gate « aucun film en baisse » de L3 | portage : gate tenu (établi) ; vieux builds : un bit de trop mesuré, localisation à `i4` NON CONFIRMÉE (`R_COMP.md` §2) |
| R-COMB | L1 (oracle (i)+(ii)) + L8 + L2 + L9 ensemble sur HI_1_13_0, en copie de recherche (N10) | la phrase « la phase 2 seule n'atteindra pas le déclencheur » (estimée) | mesuré ; la partie estimée de la phrase n'est pas une borne (vérificateur) ; R-COMB-2 proposée (`R_COMB.md`) |
| R-L4 | largeur `ti=40` fausse en image-clé (D-52) ; châssis inconnus (§6.2 L4) | L4 | (a) établi (MPP 8 bits : mesuré) ; (b) mécanisme mesuré, lecture fausse établie pour `77ef810a`, supposée pour les deux autres (`R_VEH.md` §1-§3) |
| R-L6 | ordre des plages des cartes à deux sbsp ; index 1 sur les cartes à un sbsp (D-54) | L6a hors Live Fire | ordre établi ; D-54 réfuté ; L6a hors Live Fire réfuté (`R_VEH.md` §5) |
| R-P3 | coupables résiduels des décalages : `ti=20 i1`, `DEL ti=0`, `ti=14 i1`, `ti=41 i2` (D-56) ; gate = ces quatre comptes | — | partiel : G2, G3, G4 expliqués, G1 694 → 388 (`R_COMP.md` §4) |
| R-P6 | naissances à plus de 3 paquets (6 724 eid, 61 858 paquets hors cadre) : règle qui localise une naissance lointaine. Bornes mesurées : `propre+alloc+suivant` +3 227 / −578 ; `propre+pont` +295 / −2 (BIS_3 §4) | élargit L1 | partiel : pas de règle générale, pas de lot (`R_NAIS.md` §4) |
| R-HOM | recensement des composants homonymes (nom → tables de composant du binaire, D-49) | L8 (prérequis léger) | établi : un seul homonyme de grammaire (`R_COMP.md` §3) |
| R-LS | vérification et mesure de D-67 (ajoutée au workflow) | LS, LU | établi (`R_LOC.md` §2-§3) |

**Non retenus** (mesurés nuls ou négatifs) :
- T3-C2, « ne pas lire la vue C après un rejet » : 0 paquet gagné, jusqu'à 4 598 perdus. **Réévalué
  le 2026-10-02 (R-L1 (b), N-3)** : ces 4 598 fermetures sont factices ; si l'invariant « sortie de
  vue B par rejet ⇒ non fermé » entre au juge (L0), T3-C2 ne perd plus aucun paquet sain (sous le
  juge actuel, il en perd 1 008).
- Ajoutés le 2026-10-02 : L6a hors Live Fire (−15 sains, 9 films sur 13 en baisse, R-L6) ; le repli
  à largeur libre du slot 123 dans la cuisson (net −14 141 sains, D-77) ; l'ordre de l'enquête
  pour LS sur le site de la cuisson (`1c4c63c2` en baisse) ; le désaveu des NEW à masque impossible
  (G3 → 0 mais −257 sains, R-P3) ; la portée sur les NEW du flux delta (−9 573 sur le format 27,
  avec `i0` écrivain, R-L4) ; la feuille 4 brute dans les NEW (−3 à −7) ; la variante de L1a par
  chunk (`60ae07c4` −59, R-L1 (d)).
- T1-6, garde par eid complet : ≤ 325 paquets.
- T2 et T8-C1/C3 : aucun écart.
- T5-5, kinds 1/2 : jamais écrits dans un film.
- L'oracle d'ordre dur dans `decodeInferLoop`.
- Les liaisons des régions (iii') et (iii) par recherche d'en-tête : −522 et −65 (D-45). L1c ne
  garde donc plus que R-P6.
- Le bloc `0xbc` (ancien L5).
- `IDLowBits` calibré par film : 13 partout.
- Les sites `ti38-i18`, `unit-actor-state`, `tacmap-waypointstate`, `crew-order`,
  `tacmap-cooptetherarea` et `flock-destination`, en perte nette (BIS_2 §3.1).

### 6.2 Détail des lots

**R-L1 — Recherche préalable à L1 (research, Ghidra + sonde)**
- (a) Région (iii') : 1 289 NEW trouvés derrière un rejet antérieur. L'en-tête rejeté était-il un
  DELTA ? À relire : `FUN_14076b9c8`, `FUN_142f2cc78` (budget `0x1000` bits, plafond `0x200`) et le
  rollback `FUN_14076a148` de `FUN_142f2cee0`.
- (b) Les 4 598 paquets fermés après un rejet : pied de trame, ou fermeture factice ? Piste : la
  moitié des « fermés après rejet » de P3 est un `DEL ti=0` suivi d'un en-tête nul (D-56).
- (c) La vue A des paquets à événements se lit-elle jusqu'au bout par grammaire (copie de tampons de
  `FUN_142f2c050` / `FUN_1406d60f4`) ?
- (d) La condition par film qui empêche L1a de perdre sur les vieux builds (D-46).
- Gate : vet et tests `-tags=research` ; aucun fichier de production. Taille S-M. J12 : non.
- **Jouée le 2026-10-02** (`R_NAIS.md` pour (a), (b), (d) ; `R_LOC.md` §4 pour (c)) : (a), (b) et (d)
  établis, (c) partiel ; résultats au §6.5.1.

**R-L3 — Recherche préalable à L3 (research, sonde + Ghidra)** (porteur, méthode et gate ajoutés le
2026-10-02, point 30 de la critique n° 1)
- Objet : sur HI_1_4_1, version-31 et version-33, le bassin `i15` (`managed-engine-timers`) rend des
  masques « tout à un ». Cause non élucidée : masque lu à un mauvais bit, registre `ti=2` différent
  sur ces builds (D-26 : trois familles pour `ti=43`, `ti=0` à 26 entrées sur version-31), ou
  grammaire `i15` différente.
- Porteur : un agent de recherche de la campagne.
- Méthode : (1) relever, par film des trois builds, le registre `ti=2` lu dans le film et le comparer
  à celui de HI_1_13_0 ; (2) A/B en surcouche du lecteur `i15` porté (`FUN_1407ee7b8` +
  `FUN_1407ee87c`) sur les trois builds, avec pour témoin un décalage d'un bit (méthode de BIS_2 §2) ;
  (3) aucune lecture Ghidra des vieux builds n'est possible (un seul exécutable) : la preuve est la
  mesure, sous D6.
- Gate : vet et tests `-tags=research` ; publier, par film des trois builds, les paquets fermés sains
  sous le portage contre la référence. R-L3 rend « aucun film en baisse », ou la liste des films en
  baisse avec leur cause. Aucun fichier de production. Taille S.
- **Jouée le 2026-10-02** (`R_COMP.md` §2) : « aucun film en baisse » pour le portage HI_1_13_0 ;
  registre `ti=2` identique sur les 21 films ; sur les trois vieux builds, un bit de trop entre
  `i4` et `i10` (localisation à `i4` non confirmée, §6.5.1). Proposé : scinder L3 (§6.5.2).

**L1 — Lire les naissances d'entités (marche)**

*Gain mesuré (BIS_1 §4)*
- Borne de l'oracle (i)+(ii) : +29 755 / −1 587 paquets, **net +28 168, +360 182 utiles**,
  0,9 % de gains factices. Sur HI_1_13_0, de 78,4 % à 85,9 % sur le dénominateur fixe maximum
  (sains : de 78,1 % à 85,6 %).
- Borne propre de L1a (région (i)) : +16 383 nets, +196 804 utiles. Le localisateur réel filtré par
  les invariants (`tete-bloc+inv`) fait mieux sur HI_1_13_0 (+153 699 utiles sains). Mais il
  **perd** des utiles sains sur HI_1_8_0 (−1 953), HI_1_9_0 (−1 404), HI_1_10_0 (−21 094) et HI_1_11_0
  (−1 904), et sur les six films de ces builds : il **ne passe pas le gate** sans R-L1 (d).
- Borne de L1b (région (ii)) : +11 540 / +162 964 ; dépend de R-L1 (c).
- L1c sort du lot : (iii') et (iii) sont négatives ; ce qui reste de la recherche d'en-tête relève
  de R-P6, sur décision D3.

*Preuve Ghidra*
- `FUN_142f2e174` : un mot par entité et par paquet ;
- `FUN_142f2f8f0` : état 3 à l'écriture du NEW ;
- `FUN_142f24a78` + `FUN_14076b9c8` : ordre de la vue B ;
- `FUN_142f2c754` : en-tête ;
- `FUN_1406cbaa0` : garde DELTA ;
- `FUN_142987460` : code de vue ignoré ;
- allocateur : `FUN_142f2f634` / `FUN_142f2f0cc` / `FUN_142f2fc08`, table `0x143cefd78`, tête
  `FUN_142f2e598`, drapeau `DAT_144706104`.

*Réserves*
- D-7 : le `R(6)` trouvé contredit le pont dans 144 cas sur 266.
- Le slot prédit par l'allocateur n'est pas établi. La tête est établie sur HI_1_13_0 (97,7 %) ;
  sur HI_1_12_0, ce n'est qu'un indice faible (rang 0 seulement : pool 1 à 13/15, pool 4 à 10/18).
- D-35.

*Sous-lots, dans cet ordre*
- L1a, tête des paquets à événements. Les candidats NEW de tête sont admis par la grammaire (ordre,
  pas de DEL dans la chaîne) ET par l'allocation de l'eid au bloc de type 1, puis filtrés par les
  invariants de masque. S'y ajoute la condition par film issue de R-L1 (d).
- L1b, paquets à événements non localisés : lecture de la vue A selon R-L1 (c).

*Vecteurs* : T1 note §6 V1, V2 ; T3 note §6 ; T8 note §6 V1.

*Fichiers*
- `grammar/debut_de_liste.go` ;
- `grammar/object_deaths_march.go` (`marchLocateStrict`) ET **sa copie
  `facts/killsource/walk.go`** (`locateStrict`, `locateFallback` : la règle des deux copies impose de
  les traiter ensemble ou de centraliser) ;
- `grammar/movement_states.go` ;
- `grammar/frame_infer.go` ;
- `grammar/frame_harvest.go` (`decodeFrameParRangs`) ;
- `grammar/frame_closure.go` / `frame_closure_classement.go` ;
- `grammar/frame_closure_detail.go` (la recopie du pilotage) ;
- `grammar/type1_datums.go` ;
- `grammar/keyframe_datums.go` ;
- la sonde `campagne_marche_research_test.go`.

*Gate* : §6.0, points 1 à 6, dont l'équivalence ou le delta killsource, la performance (lecture du
bloc de type 1 en production) et le garde-fou de la recopie. Le compteur des liaisons par source
passe en diagnostic typé.

*Taille* L. *J12* : conflits J12.1 et J12.5 (inventaire §6.0).

*Place* (recommandation du 2026-10-02, §6.3 D-RI) : vague 2, posé UNE fois dans le localisateur
unifié par le lot LU. Les fichiers `object_deaths_march.go` et `facts/killsource/walk.go` ci-dessus
se réduisent alors au localisateur unifié.

*Mis à jour le 2026-10-02 (recherches préalables, §6.5.2)* :
- L1a : condition par film de R-L1 (d) (allocateur à cinq pools ≥ 50 % des NEW propres du film)
  — 0 film en baisse, +15 070 sains ; deux passes, ou forme causale (−1 689 sains de gain) ; seuil
  choisi sur le corpus (D16).
- L1b : n'est plus un correctif de localisateur. LS localise 45,8 % de la région (ii) ; le reste
  (26 427 paquets) exige les grammaires de charge des messages de la vue A (41 genres de tête).
  Proposé : sortir L1b de la vague 2 (D15).
- Gate : jouer L1 sur une tête qui porte déjà L9, avec `4f77afc1` pour témoin nommé (D-69) ; nommer
  aussi `1c4c63c2` et `084a804d` (l'oracle y perd 1 216 et 214 sains en brut, le critère du gate 2
  restant net par film).

**LU — Unifier les localisateurs jumeaux (zéro différence, vague 2, premier lot)** (ajouté le
2026-10-02)
- Objet : `marchLocateStrict` (`grammar/object_deaths_march.go`) et sa copie
  `facts/killsource/walk.go` (`locateStrict`, `locateFallback`) deviennent UN localisateur, appelé
  par les deux marches. Le commentaire de `object_deaths_march.go` (« UNE SECONDE COPIE DE CETTE
  MARCHE EXISTE ») et la règle des deux copies le demandent.
- Gate : sortie identique. Équivalence killsource NULLE sur les 20 films, `replay-equiv` 0,
  `frame_closure.golden` identique, `grammar.Rev` et `killsource.Rev` inchangés (révision constante),
  plus un garde-rail (test grep) qui interdit une seconde copie (règle 6).
- Taille S-M. Sortie : non.
- *Mis à jour le 2026-10-02 (R-LS, D-73)* : les deux sites n'ont déjà pas le même ordre (la cuisson
  n'a pas de repli à largeur libre) et LS leur en donne deux différents. Le localisateur unifié porte
  donc un paramètre d'ordre (ou deux appelants d'un même cœur) ; « posé une fois » vaut pour le test
  de la signature, pas pour l'ordre.

**LS — Signature du localisateur figée sur le slot 123 (marche, vague 2, juste après LU)** (confié à
la campagne le 2026-10-02 par accord avec la session du chantier de suite d audit, l utilisateur
ayant laissé les deux sessions trancher ; GO daté de l utilisateur requis comme pour tout lot)
- Source : D-67 et l enquête `.ai/V7.5/film_re/ENQUETE_SCAN_SEPTEMBRE_2026-10-02.md`
  (`origin/feat/suite-audit-decodeur` @ `90014fe79`), à relire sur pièces à l entrée du lot.
- Objet : le localisateur unifié (LU) accepte, pour la signature du delta « high-frequency » de 35
  bits, tout slot lié à cet archétype dans le registre et les images-clés du film (124, 126 à 129 dans
  les modes à objectif porté), dans l ordre « slot 123 strict -> autre slot high-frequency -> repli
  largeur libre » ; une seule implémentation pour la marche des morts d objet et killsource.
- Gain attendu : moins de kills servis par le `scan` (19,1 % en septembre, 85,3 % d entre eux dans
  19 matchs à objectif porté) ; part de la région (ii) « paquets à événements non localisés » à
  mesurer par la carte v2 avant et après.
- Gate : §6.0 complet, dont le gate killsource (delta déclaré, montée de `killsource.Rev`, backfill
  killsource déclaré) et le gate par film ; test d après les films de l enquête (trois films cités).
- Taille S. Sortie : oui.
- *Mesuré le 2026-10-02 (R-LS, `R_LOC.md` §3 ; 20 films + 9 de l'enquête)* :
  - l'ordre de l'enquête N'EST PAS retenu pour la cuisson : il fait baisser `1c4c63c2` (13 540 →
    12 691 paquets sains) ;
  - cuisson (`debutDeLaListe`) : « 123 strict → fermeture par NEW de tête → signature
    high-frequency », sans repli à largeur libre : +18 119 / −2 paquets, +18 086 sains, +169 439
    utiles sains, 0 film en baisse sur 30 ; HI_1_13_0 de 68,6 % à 73,8 % sur le fixe de R-LS ;
  - killsource et `marchLocalise` : « 123 strict → signature high-frequency → repli libre du slot
    123 » : 1 080 morts du `scan` à la marche sur 4 366, aucune mort apparue ni disparue, tag,
    statut, crédit et origine inchangés ; les trois films de l'enquête redonnent exactement
    362 / 403 ;
  - la voie (`read_path`) est publiée (D-74) : montée de `killsource.Rev` et backfill killsource DUS ;
  - non mesuré : la marche des morts d'objet (`ScanMarchFacts`) sous LS ; l'ordre retenu n'est
    mesuré que par la sonde (la surcouche de `cmd_fermeture` ne réalise que l'ordre de l'enquête) ;
    `51ebbc0f` (D-76) ;
  - pourcentages « variables » de la note = fermés BRUTS ; en sains, HI_1_13_0 80,3 % → 82,9 %.

**L8 — `ti=3` low-frequency et routage des homonymes (composant)**
- Gain mesuré (BIS_3 §6, surcouche) : +30 618 / −10 paquets, +234 454 utiles, hors cadre de 264 757 à
  235 620.
  - Par film : `fb1a1a72` +19 764, `51ebbc0f` +10 029, `c75f33b8` +817.
  - **Perte par film : `111fa685` −1 (0 gagné, 1 perdu)**. Elle est à juger au gate par film.
  - 34 gains sont contredits par un invariant.
  - Le témoin (lire `ti=4 i0` sur 26 bits) effondre la marche (−256 370) : il prouve que les deux
    tables sont distinctes.
- Preuve Ghidra : `low-frequency` → table `0x143d07b40`, lecteur `FUN_142ed4aec` ; `high-frequency`
  → deux tables, `0x143d06a60` (`FUN_14076d034` = `R(8)`, `ti=4`) et `0x143d07af0` (`FUN_142ed4880` =
  `R(16)+R(8)+R(2)`, `ti=3`). L'appartenance de la seconde à `ti=3` est **déduite** (voisinage,
  champs, mesure), pas lue dans le registre d'archétypes du jeu : à lire avant le portage. La
  grammaire de `i1` est donc **probable** (pas établie) ; celle de `i0` est établie.
- Fichiers : `grammar/dispatch_item.go` (routage par archétype ou par descripteur, l. 122) ;
  `components_probe.go` (`compHighFrequency`) ; un lecteur `ti=3 i0` ; `registry.go` (sur sa forme
  post-J12.4, contrainte de structure) ; `ecs_table.tsv`.
- Prérequis : R-HOM (recensement léger). **Levé le 2026-10-02** (`R_COMP.md` §3) : la table
  `0x143d07af0` est enregistrée par `FUN_140e460fc` avec `+0x4754 = 3` : la grammaire de `i1` est
  ÉTABLIE. Routage par archétype ou par table, jamais par nom ; corriger `ecs_table.tsv` (D-86) ; un
  garde-fou « dispatch par table » est constructible (D-89). Recalcul en paquets sains (BIS_3) :
  +30 596 sains.
- Gate : §6.0, points 1, 2, 3 (killsource, étendu le 2026-10-02 : N1), 6 et 7.
- Taille S-M. Ne dépend d'aucune recherche de marche.

**L2 — Dispositifs `ti=43` (composant), reclassé le 2026-10-02**
- Gain mesuré (BIS_2 §5, surcouche, 21 films) :
  - corpus : +18 105 / −12 paquets, +143 314 utiles ;
  - HI_1_12_0 : de 5 835 à 17 132 paquets fermés (`bcb6d393` +11 297) ;
  - HI_1_13_0 : +6 364 / 0 paquets, hors cadre de 86 921 à 81 049. **Levier de la cause n°1** : T7-5
    et T7-6 sont réfutés.
  - `81c02726` : +3 580 / 0. La fenêtre de la 3e montée passe de 181 à 218 paquets fermés sur 222.
  - Pertes : `1c4c63c2` (HI_1_10_0) +358 / −12, avec 342 gains contredits ; sur HI_1_10_0, le gain
    est factice.
- Ce qui n'est **pas** mesuré : que la 3e montée de G MONEY soit publiée (cela demande une cuisson,
  avec l'accord de l'utilisateur).
- Preuve Ghidra : `i35` `FUN_141076f68` + `FUN_143206f24` (écrivain `FUN_142f0570c` / `FUN_1432070d0`) ;
  `i34` `FUN_140f44104` + `FUN_143206e48` / `FUN_143206d34` ; `i21` `0x1407f0678` ; `i31` `FUN_142f02a48`
  (N ≥ 9 = échec) ; `i37` `FUN_142ba78dc` ; `i19` à `i23` (T7 §2-§5). Liste complète : T7 §2-§5.
- Vecteurs : T7 note §6.1.
- Fichiers : nouveau `grammar/components_device_ti43.go` (≤ 500 L) ; `dispatch_biped.go`,
  `capture.go` ; `ecs_table.tsv` ; garde G4 `ecs_widths_guard_test.go` (D-22).
- Gate : §6.0, avec en plus :
  - le juge des invariants sur `1c4c63c2` et sur HI_1_10_0 (les fermetures factices gagnées n'y
    comptent pas comme gain) ;
  - `81c02726` comme témoin du lot (en plus de `bcb6d393`) ;
  - le registre `ti=43` en trois familles (HI_1_10_0 à HI_1_13_0 ; HI_1_8_0 = HI_1_9_0 ; HI_1_4_1 =
    v31 = v33) : la grammaire lue sur HI_1_13_0 ne vaut que pour la première famille sans autre
    preuve (D6).
- Taille M.

**L6a — Largeurs par index de plage (composant et donnée de carte)**
- Gain mesuré (BIS_2 §3.3, contexte de production) :
  - `0797ce72` : +3 269 / −3 paquets (+17,1 %), +26 664 utiles ;
  - `60ae07c4` : +1 806 / −49, +11 052 utiles ;
  - les paquets qui lisent l'index 3 passent de 3,3 % à 82 % de fermeture ;
  - les rejets « aucune allocation » de ces films baissent de 81 % et 91 % (D-6).
  - Ailleurs : ±0 à ±4 paquets.
- Preuve Ghidra : `FUN_14076e524` lit la ligne de l'index lu.
- Donnée : les bornes de chaque plage déclarée par la carte, lues dans les modules (Live Fire :
  4 plages, `mb2_regions_live_fire.tsv`). Il faut étendre le catalogue `map_quant_bounds.json` et
  `replay.installWorldObjectPrecision`.
- Fichiers : `grammar/lecteur_position.go`, `film/replay/world_object_precision.go`, catalogue de
  cartes ; `grammar/film_context.go` sur sa forme post-J12.3 (plus de `slog`, contrainte de
  structure).
- Gate : §6.0, avec les pertes de `60ae07c4` (49) jugées par le juge des invariants. Hors Live Fire,
  attend R-L6.
- Taille M.
- *Mis à jour le 2026-10-02* :
  - R-L6 (`R_VEH.md` §5) : ordre des plages établi pour toutes les cartes, mais la lecture par
    index ne gagne rien hors Live Fire (13 films : +15 / −18, −15 sains, 9 films en baisse). L6a
    reste LIMITÉ à Live Fire sur le corpus ;
  - R-P3 (`R_COMP.md` §4) : étendre le lot au site `ti=41 i0` (`consumeObjectPositionMonde`, qui lit
    les largeurs de la plage de la carte au lieu de la ligne de l'index lu, `FUN_14076e524`). G4
    (160) ne tombe à 0 qu'avec LES DEUX lectures (position d'objet du monde au jeu ET lecture par
    index) ; `0797ce72` +495 sains de plus, `60ae07c4` −28 (perte déjà vue par D-55, à juger sous
    D4).

**L6b — Sites de position lus comme le jeu (composant)**
- Gain mesuré (BIS_2 §3.1, contexte des instruments) :
  - `flock-position` : +416 / −10 ; les pertes sont sur `60ae07c4` (−8), `11de8353` (−1) et
    `1c4c63c2` (−1) ;
  - `tacmap-displayasset` : +77 / −16, dont −10 sur `51ebbc0f` (déjà connu, R3).
- Les deux sites perdent sur au moins un film. Le juge des invariants n'a pas été joué sur ces A/B :
  il faut le jouer avant de demander D4.
- Fichiers : `lecteur_position_exceptions.go`, `lecteur_position_ratchet_test.go`.
- Gate : §6.0, et D4. Taille S.

**L9 — Marche d'image-clé toutes générations (marche, image-clé)**
- Gain : oracle +1 784 / −1 paquets, +45 694 utiles, 20 gains contredits. Par build : HI_1_13_0
  +1 050, HI_1_10_0 +564, HI_1_8_0 +133. L'oracle lie avec l'archétype du pont ; ce n'est pas un
  lecteur.
- Cause lue dans le code Go et mesurée (BIS_3 §2) :
  - `kfAnchorFromID` rejette la génération 0 ;
  - voisin et recalage n'acceptent que la génération 1 ;
  - l'en-tête est présent dans l'image-clé pour 87 eid sur 87 ;
  - contrôle : la génération 0 est déclarée 0 fois sur 191, la génération 1 373 612 fois sur
    373 612.
- Risque : une ancre de génération 0 a aussi l'aspect d'une zone de données nulle. Le filtre
  `field 26 = 0, ti < 50` reste en place.
- Fichiers : `grammar/keyframe_world.go`, `keyframe_closure.go`.
- Gate : §6.0, avec en plus `keyframe_closure.golden` et la recherche exhaustive d'en-tête comme témoin
  (87 sur 87). Aucune déclaration nouvelle ne doit contredire un bloc.
- Taille S-M.
- *Mis à jour le 2026-10-02* : dans la combinaison R-COMB, la marginale de L9 est négative sur
  HI_1_13_0 (−1 112 sains) quand l'oracle L1 est dérivé après L9, positive (+1 078) sinon : c'est un
  artefact de l'oracle L1 (D-69), pas un défaut de L9. R-P3 propose un lot voisin, LP (désaveu des
  déclarations que le bloc de type 1 dit non vivantes), qui exige la lecture du bloc en production
  (§6.5.2) ; le cas (a) de R-P3 ressemble à une erreur de la marche d'image-clé par voisinage
  (vérificateur, `bf15f7ab` slot 553).

**L3 — Moteur `ti=2` / `ti=0` et helper `FUN_140d580d0` (composant)**
- Gain : **estimé** ≤ 4 331 paquets (`ti=2` 4 208, dont `i15` 3 996 ; `ti=0` 123). Sur HI_1_13_0,
  2 260 paquets `ti=2`, 229 utiles. Le portage n'est pas mesuré : il faut un A/B en surcouche avant le
  GO (même méthode que L2).
- Vieux builds : R-L3 est un prérequis du gate par film.
- Preuve Ghidra : `i15` `FUN_1407ee7b8` + `FUN_1407ee87c`, `FUN_142ba78dc`, `FUN_1424cd048` ->
  `FUN_140d580d0`, écrivain `FUN_142edad74` ; `i16` `FUN_1410d9004` ; `i17` `FUN_141101038`.
- Fichiers : `components_game_engine.go`, `components_walk_batch9.go`, `vitality.go`,
  `dispatch_player.go`. Migration des cinq copies de `FUN_140d580d0` (D-15) : un helper, plus un
  garde-rail grep (règle 6).
- Taille S-M.
- *Mis à jour le 2026-10-02 (R-L3, `R_COMP.md` §2)* : A/B delta du portage en surcouche, 20 films :
  +3 575 paquets sains, +53 521 utiles sains, 3 924 gagnés dont 316 contredits, 0 sain perdu,
  aucun film en baisse (gate par film tenu). Témoins `i15 ±1` : 0 / 715 images-clés fermées. Proposé :
  - L3a = ce portage (vague 1) ;
  - L3b = vieux builds : un bit de moins ferme 98 / 98 images-clés du moteur, mais la position du
    bit n'est pas discriminée entre `i4` et `i9` (vérificateur : un décalage de −1 à l'entrée de
    `i5` à `i10` ferme autant). Une correction posée sur `i4` fermerait les records en rendant une
    valeur fausse si le bit est ailleurs. Prérequis : relire en Ghidra (HI_1_13_0) les lecteurs `i5`
    à `i9`, ou trouver un oracle de VALEUR (numéro de manche, minuteurs). Condition mesurable par
    film (pas de branche sur le build) ; preuve par mesure sous D6. Effet delta mesuré : +0 / +0 /
    +4 sains sur ces trois builds.

**L4 — Véhicules `ti=40` : porte lue et composants (composant)**
- Gain :
  - image-clé, **mesuré** (BIS_2 §4.2) : de 14 à 137 records fermés sur 7 059 avec la porte par
    châssis. Sur les non-VTOL, la porte levée ferme 63 records, la porte posée 1 : la « porte
    supposée posée » serait fausse en image-clé ;
  - delta : **estimé** ≤ 1 666 paquets (causes nommées de la carte v2), portage non mesuré.
- Porte établie : (type de physique == 6), types lus dans les tags `vehi` installés (Falcon et Wasp de
  type 6).
- Prérequis, avec porteur et méthode (critique, point 35) :
  1. **R-L4 (a), largeur `ti=40` fausse en image-clé.**
     - Porteur : un agent de recherche de la campagne.
     - Méthode : A/B par composant sous la surcouche (`i2`, `i43`, état par défaut), avec pour
       témoin les pièces montées de `4f77afc1` (326 qui dépassent leur frontière).
     - Gate : records `ti=40` fermés en image-clé.
  2. **Châssis `77ef810a`, `4118381d`, `d0b40d0a`.** Quatre méthodes sont épuisées (modules
     installés, registre du dépôt, Ghidra, créations des films). Proposition : ne plus les
     identifier. En delta, l'annonce de `i33`/`i34` prouve la porte par la loi de l'écrivain
     (`FUN_142f09c74`), quel que soit le châssis lu. En image-clé, un châssis inconnu prend la porte
     levée (137 contre 88 fermés).
     - Porteur : l'utilisateur, par la décision D5.
  3. **D5** : où vit la table châssis -> type de physique, et levée de « NE PAS PORTER » pour
     `i41`/`i42` en image-clé.
  4. **D-VEH** (ANALYSE §7 (6)) : les blocs véhicules entrent-ils dans la campagne ?
- Fichiers : `grammar/composants_vue_b_m4b.go`, table des types de physique (emplacement selon D5),
  registre des replis (`facts/fallback/noms.go`, `registre_filmdec_marche.go`), `ecs_table.tsv`.
- Gate : §6.0. Porte LUE et composants dans le MÊME lot.
- Taille M-L.
- *Mis à jour le 2026-10-02 (R-L4, `R_VEH.md`)* :
  - prérequis 1 LEVÉ : la « largeur fausse » est en fait trois lectures communes (portée
    `DAT_144e61ea0` en image-clé, chemin `i0` de l'écrivain, MPP 8/3 sur les formats 24-25) ; les
    trois ensemble ferment 3 058 / 3 058 voisins (formats 24 à 27 ; témoins ±1 bit ≤ 7,7 %). Les
    pièces montées portent toutes `bVar14 = 1` : feuille 4 en `R(96)`. La porte par châssis est
    confirmée par la fermeture (120 posée partout, 2 952 levée partout, 3 058 par châssis). Formats
    20-21 : 0 / 2 036 (D-92) ;
  - delta mesuré : 16 composants, porte posée en delta : +1 436 / −0 paquets, +1 401 sains, +26 843
    utiles, aucun film en baisse ;
  - prérequis 2 : les trois châssis sont des LECTURES FAUSSES selon la note (MPP lu à 9 bits au lieu
    de 8 sur les formats ≤ 25 : 0 / 5 417 châssis connus à 9/5 contre 4 526 / 5 417 à 8/3). Établi
    pour `77ef810a` (NEW faux) ; SUPPOSÉ pour `4118381d` et `d0b40d0a` (vérificateur, D-98). La
    proposition « ne plus les identifier » tient, avec cette justification ;
  - « NE PAS PORTER » `i41`/`i42` : démenti en image-clé (D-14) ;
  - proposé : scinder L4 en L4a (delta, prêt sous D5) et L4b (image-clé, après LK et LM) (§6.5.2,
    D15).

**L7 — NEW sur slot occupé (marche)**
- ≤ 137 paquets. Lieu : `grammar/frame_infer.go` (`contreditUneEntiteVivante`).
- Prédiction de l'allocateur : établie pour la tête sur HI_1_13_0 (indice faible sur HI_1_12_0),
  pas pour le slot.
- Place : vague 2, avec L1, dans le localisateur unifié (§6.3 D-RI).
- Priorité basse. Taille S.
- *Mis à jour le 2026-10-02 (R-P3)* : chronique `11de8353` slot 688 : un NEW `ti=14` à masque
  impossible est lié, puis le vrai NEW du bipède est refusé sur ce slot « occupé ». Désavouer les
  NEW à masque impossible ne convient pas (−257 sains).

**L0 — Instrument (carte seulement)**
- L0.1 : invariants de l'écrivain et sortie de vue B dans la carte (`bloquantDuPaquet` : la sortie de
  vue B passe avant toutes les causes de vue C), accesseur `Deborde()` de `source.Bits` (T8-C2).
- L0.2 : classements corrigés, à savoir :
  - naissance de génération 0 (D-43) ;
  - NEW lu-désynchronisé distinct de « naissance non lue » (D-44) ;
  - « bloc `0xbc` » requalifié en désalignement (D-53).
- L0.3 : garde-fou de la recopie étendu (D-61) : compte des paquets à événements localisés et non
  localisés par bobine. Si aucune bobine du dépôt n'en porte de localisé, ajouter une bobine ou un
  paquet synthétique.
- L0.4 : publier les records utiles lus et les deux dénominateurs, fixe (recalculé par vague) et
  variable (D-42, règle « Pourcentages » du §6.0).
- L0.5 (ajouté le 2026-10-02, N14) : mesurer la part des NEW lus mais désynchronisés (D-44) dans
  les 213 033 « naissances attestées » du corpus, une fois la classe L0.2 corrigée (D-63).
- Fichiers : `frame_closure_classement.go`, `frame_closure_detail*.go`, `research/cmd_fermeture/`,
  `frame_closure.golden`, `frame_closure_detail_test.go`, `source/bits.go`. Aucun fichier lu par
  une cuisson, sauf décision D2.
- Gate : Gate 1 du §2, à révision constante.
- Taille M.
- *Ajouts proposés le 2026-10-02* :
  - L0.6 : invariant « sortie de vue B par rejet ⇒ paquet non fermé » (T3-C1 ; R-L1 (b)) : requalifie
    4 598 fermetures, dont 1 008 jugées saines aujourd'hui (0,36 % des sains de référence) ;
  - L0.7 : « masque au-delà du dernier composant de l'archétype » classé violé à la LECTURE (D-85) ;
    le libellé `DEL ti=0` de `dernier_composant_x_classe` désigne un en-tête mal lu (R-P3) ;
  - L0.8 (candidat, à mesurer d'abord) : mot de 32 bits d'un DEL non nul hors archétype `0x10`
    (D-83) ;
  - L0.9 : témoins décalés de 2 à 8 bits en plus de ±1 (D-99), et prise en compte de
    l'auto-synchronisation de la vue C (D-80) dans la lecture des témoins.

**L10 — Garde-fou du cardinal du bloc de type 1**
- Un film dont un bloc de type 1 n'a pas 8 191 entrées est refusé ou daté, et sa largeur est lue sur
  le cardinal (pas par balayage).
- Mesuré : 8 191 partout. `1c4c63c2` alloue le slot 8 190, à un slot de la limite.
- Fichier : `grammar/type1_datums.go` (et `frame_records.go:144`).
- Taille S. Sur décision.

### 6.3 Décisions demandées à l'utilisateur

- **D1 (reformulée ; l'ancienne question est DÉJÀ TRANCHÉE)**. L'ouverture de l'étape 1 de la
  représentation intermédiaire est décidée par l'utilisateur le 2026-10-01 : elle démarre après la
  fusion de J12 (ANALYSE §7 (2)).
  - Question restante : **réviser le seuil du déclencheur ?** Il n'est atteint sur aucun build, même
    sous l'oracle (HI_1_13_0 85,9 % sur dénominateur fixe ; entrées ≥ 45,0 % en borne basse). Son
    dénominateur monte quand on lit plus (D-42).
  - Options :
    - (a) garder 95 %, sur le dénominateur fixe maximum et en paquets sains ;
    - (b) un seuil par build ;
    - (c) remplacer le déclencheur par « plus aucune cause nommée de poids ».
  - Recommandation : (a), en publiant les deux dénominateurs. Le fixe est le maximum des records
    utiles lus sur {référence, lots déjà fusionnés, oracles mesurés}, recalculé à chaque vague
    (§6.0, « Pourcentages », N7).
  - *Tranchée le 2026-10-02 (D1 du §3 : indicateur publié à chaque vague)*. Chiffre à jour (§6.5.3) :
    sous les six leviers de R-COMB, HI_1_13_0 vaut 86,0 à 88,0 % sur le fixe de R-COMB, 83,7 à
    85,7 % sur le fixe consolidé.
- **D-RI (nouvelle) — Où atterrissent les lots, par rapport à la représentation intermédiaire ?**
  - ANALYSE §5.1 et §6 recommandaient « marcheur avant la campagne » : les correctifs de grammaire
    atterrissent une fois, dans un seul marcheur. La version précédente de ce §6 faisait atterrir
    L1 à L7 dans les marcheurs actuels sans le dire.
  - **Les lots de COMPOSANTS** (L2, L3, L4, L6a, L6b, L8) changent des lecteurs appelés par le
    dispatch commun à toutes les marches. Ils atterrissent une seule fois, quel que soit le nombre de
    marcheurs.
  - **Les lots de MARCHE des paquets à événements** (L1, L7) atterriraient aujourd'hui dans TROIS
    marcheurs (L9 touche la marche d'image-clé, `keyframe_world.go`, hors de ces trois) :
    - `debutDeLaListe`, qui sert `movement_states.go`, la carte de fermeture et sa recopie ;
    - `marchLocateStrict` dans `object_deaths_march.go`, pour `ScanMarchFacts` et les morts ;
    - sa copie `facts/killsource/walk.go`.

    Après l'étape 1.2 de la représentation intermédiaire (« la marche de production devient
    `FilmContext.Trames` »), la marche de production n'en fait plus qu'une. Les morts d'objet et
    killsource ne la rejoignent qu'aux lots 2.7 et 3.1 (ANALYSE §3.3).
  - **Recommandation précédente RETIRÉE le 2026-10-02** (critique n° 2, N4 et N5). Elle disait :
    composants fusionnés « avant ou pendant l'étape 1 », et « L1 en entier (avec killsource) avec ou
    après le lot 2.7, comme ANALYSE l'ordonnait ». Deux défauts :
    - **cycle (N4)** : ANALYSE §7 (2), retenue par l'utilisateur, dit « le lot 2.7 attend la
      campagne », et ANALYSE §5.1 dit « avant le lot 2.7 ». Le plus gros levier aurait attendu 2.7,
      qui attend la campagne. L'attribution « comme ANALYSE l'ordonnait » était fausse ;
    - **couplage (N5)** : RI 1.2 et 1.3 ont pour gate `replay-equiv` 0 et `frame_closure.golden` /
      `keyframe_closure.golden` identiques (ANALYSE §3.3). Un lot de composant fusionné pendant
      l'étape 1 oblige à re-figer ces références au milieu de l'étape.
  - **Recommandation (soumise à l'utilisateur, 2026-10-02)** — ordre après la fusion de J12 :
    1. **Vague 1, lots de COMPOSANTS** : L2 (`ti=43`), L8 (`ti=3`), L9 (marche d'image-clé,
       générations 0 et 2), L6a et L6b (positions), L4 si ses prérequis sont levés (R-L4, D5,
       D-VEH), L3 après R-L3. L9 est un lot de marche d'image-clé : il ne touche pas le
       localisateur des paquets à événements, d'où sa place ici.
       - Développés sur la branche, chacun avec ses gates (§6.0) et sa montée de `grammar.Rev`.
       - Fusionnés ENSEMBLE dans `feat/v75`.
       - UNE recuisson du parc, décidée par l'utilisateur (D7).
    2. **Références re-figées** : `replay-equiv` et goldens de fermeture, sur la tête de la vague 1.
    3. **Vague 2, lots de MARCHE** :
       - d'abord LU, un lot à zéro différence qui UNIFIE les localisateurs jumeaux
         (`marchLocateStrict` de `object_deaths_march.go` et sa copie `facts/killsource/walk.go`) ;
         `debutDeLaListe` s'appuie déjà sur `marchLocateStrict` (`debut_de_liste.go:42`) ;
       - puis L1 (L1a, et L1b selon R-L1) et L7, posés UNE fois dans le localisateur unifié. Leur
         effet sur les morts d'objet et sur killsource est un delta déclaré (gate 3) ;
       - deuxième recuisson du parc, décidée par l'utilisateur.
    4. **Ensuite seulement, l'étape 1 de la représentation intermédiaire** (1.1 à 1.4). Elle
       refactore une marche déjà juste, à zéro différence contre les références de la fin de la
       vague 2. Puis 2.1 à 2.6, 2.7, 3.x.
    - R-L1, R-L3, R-L4, R-L6, R-P3, R-P6, R-HOM, R-COMB : sans sortie, possibles tout de suite.
      **Toutes jouées le 2026-10-02** (et R-LS) ; ordre de vague révisé proposé au §6.5.4.
  - **Ce que cet ordre règle** :
    - N4 : L1 n'attend plus le lot 2.7 ; c'est 2.7 qui attend L1. C'est conforme à ANALYSE §7 (2)
      (« le lot 2.7 attend la campagne ») ;
    - N5 : aucun lot de comportement n'est fusionné pendant l'étape 1. Ses gates à zéro différence
      partent des références re-figées à la fin de la vague 2.
  - **C'est un CHANGEMENT par rapport à l'ordre de l'ANALYSE §5.1 et §6** (« marcheur avant la
    campagne » ; lots de recherche « APRÈS l'étape 1 »), que l'utilisateur avait retenu. Pourquoi le
    proposer :
    - le bénéfice visé par l'ANALYSE (« les correctifs atterrissent une fois, dans un seul
      marcheur ») s'obtient sans attendre la représentation intermédiaire : LU réunit le localisateur
      des trois consommateurs, et L1 s'y pose une fois ;
    - les plus gros gains arrivent plus tôt : L8 (+30 618 paquets), L1 (borne +28 168), L2 (+18 105) ;
    - le calendrier de l'étape 1 change : la décision ANALYSE §7 (2) (« l'étape 1 démarre après la
      fusion de J12 ») reste vraie à la lettre, mais l'étape 1 attend les deux vagues. C'est à
      confirmer par l'utilisateur.
  - Coût :
    - deux recuissons du parc au lieu d'une ;
    - un delta killsource déclaré dès la vague 2 (il l'aurait été au lot 2.7) ;
    - l'étape 1.2 migre une marche qui contient déjà L1.
- **D-VEH (reste à prendre, ANALYSE §7 (6))** : les blocs véhicules (`ti=43` de G MONEY, l'octet
  `+0x818`, les composants `ti=40`) entrent-ils dans la campagne ? T6 et T7 sont instruits, et L2 et
  L4 supposaient la décision prise.
- **D2** : la mesure de fermeture doit-elle exclure les fermetures factices ?
  - 8 388 paquets en référence ; HI_1_10_0 perdrait 22 % de ses fermés.
  - Si oui, `vueCFermee` plus les invariants devient la définition, et L0 touche un fichier lu par la
    publication.
  - Le gate du §6.0 suppose déjà que les gains factices ne comptent pas.
- **D3** : accepter une étape de récupération NOMMÉE et COMPTÉE pour les naissances lointaines
  (R-P6, borne +3 227 / −578, filtre non validé au niveau de l'occurrence), ou n'admettre que la
  lecture par grammaire (L1a, L1b) ? Recommandation : grammaire seule tant que R-P6 n'a pas de filtre
  d'occurrence.
  - *Mis à jour le 2026-10-02 (R-P6, `R_NAIS.md` §4)* : des filtres d'occurrence existent mais ne
    couvrent que 40 à 111 eid (`ferme+suivant` +474 / −4, faux positifs estimés ≈ 13 % et
    sous-estimés : le témoin ne compte que les cibles qui en ont un). Le filtre `propre+pont` de
    BIS_3 §4 (68 / 72 contre 6 / 72, +295 / −2), omis par la note, discrimine mieux (vérificateur).
    La borne propre+alloc+suivant (+3 227 / −578) repose sur une propriété de l'eid. La
    recommandation « grammaire seule » est maintenue ; aucun lot P6 n'est proposé.
- **D4** : le critère de retrait des exceptions de position (« monter sans aucune baisse ») est-il
  maintenu ? Aucun site ne le remplit sur tous les builds (BIS_2 §3.4). Avec le juge des invariants,
  les pertes factices peuvent être écartées.
- **D5** : où vit la table châssis -> type de physique (la couche `grammar` ne dépend pas de
  `replay`) ? La mention « NE PAS PORTER » de `ti=40 i41/i42` peut-elle être levée en image-clé ? Et
  l'abandon de l'identification des trois châssis (§6.2 L4) est-il accepté ?
  - *Mis à jour le 2026-10-02 (R-L4)* : `i41`/`i42` se lisent en image-clé (3 058 / 3 058 avec
    LK et LM) ; l'abandon des trois châssis se justifie désormais par des lectures fausses (établi
    pour `77ef810a`, supposé pour les deux autres, D-98).
- **D6** : un seul exécutable (HI_1_13_0) : la preuve d'une grammaire des vieux builds par mesure
  (stabilité, fermeture, juge des invariants) est-elle acceptée ?
  - *Cas concrets apparus le 2026-10-02* : MPP 8 bits des formats 24-25 (LM, mesuré, aucun
    exécutable) ; bit de trop de `ti=2` sur les formats 20-21 (L3b), où la fermeture seule ne
    localise pas le bit (D-92) : la mesure de fermeture ne suffit pas à fixer une valeur.
- **D7** : ordre (D-RI) et GO par lot. Sur la branche, une montée de `grammar.Rev` par lot qui change
  une sortie ; au parc, UNE recuisson par vague fusionnée (deux vagues, donc deux recuissons). La
  première périme le parc recuit par J11.4 : décider quand, et par qui.
- **D8** : intérêt produit de D-37 à D-39.
- **D9** : statut de `ScanObjectives` / `ScanEquipmentState` (D-12), des fichiers non suivis du
  checkout principal (D-40), et de `frame_closure_detail*.go`, `FrameClosure` et `LireBlocDeDatums`
  (D-62 : déplacer sous `film/research/`, ou garder en production avec justification écrite).
- **D10 (complétée le 2026-10-02, N3)** : la méthode de la surcouche `go test -overlay` (D-48) est-elle
  acceptée ?
  - Ce qu'elle est : des copies ENTIÈRES de `capture.go`, `lecteur_position.go` et
    `lecteur_position_exceptions.go`, substituées par chemin absolu à la seule compilation du test.
    Fichiers taggés `research && campagne_overlay`, hors du `go vet -tags=research ./...` de la CI.
  - Les mesures de L2, L6a, L6b et L8 (et la région (iii) de BIS_3) en dépendent.
  - Risque : J12 modifie `lecteur_position.go` (+1/−1) et `lecteur_position_exceptions.go` (+4/−4).
    Après la fusion de J12, des copies non re-synchronisées annuleraient ces changements dans la
    mesure.
  - **Recommandation** :
    - outillage de MESURE seulement, jamais preuve de gate (les gates se jouent sur le code du lot) ;
    - hors CI, et le dire dans chaque document de mesure ;
    - copies RE-SYNCHRONISÉES depuis les fichiers post-J12 à la fusion de J12 (item du §6.0) ;
    - copies supprimées quand le lot correspondant est fusionné.
  - Sinon : refaire les mesures de L2, L6 et L8 autrement (lot écrit sur la branche, puis carte v2).
- **D11 (nouvelle, N8) — plafond de performance et de mémoire du gate 4** (lots qui lisent le bloc de
  type 1 en production : L1, L10).
  - Mesure : durée par étape (`replay/observe.go`) et pic mémoire (sentinelle `filmproc`), avant et
    après le lot.
  - **Recommandation** : **+10 % de durée de cuisson et +10 % de pic mémoire au plus**, mesurés sur
    trois témoins et un BTB. Au-delà, le lot est revu avant tout GO de fusion.
  - *2026-10-02* : LP (proposé) lit aussi le bloc de type 1 en production et tombe sous ce gate.

**Décisions nouvelles, issues des recherches préalables du 2026-10-02 (proposées, NON tranchées)**

- **D12 — Dénominateur fixe consolidé.** Adopter, comme fixe de départ de la vague 1, le maximum par
  film sur les marches de TOUS les chantiers (HI_1_13_0 2 959 104, corpus 7 176 150 ; §6.5.3) ?
  Recommandation : oui ; c'est la règle N7 appliquée à la lettre (les A/B en surcouche comptent comme
  oracles mesurés).
- **D13 — LS : deux ordres, un par site.** Accepter « 123 strict → fermeture par NEW de tête →
  signature high-frequency » pour la cuisson, et « 123 strict → signature high-frequency → repli
  libre » pour killsource et `marchLocalise` (LU porte un paramètre d'ordre) ? Cela s'écarte de la
  règle unique proposée par l'enquête de suite d'audit (D-67, D-73). Conséquence déclarée : montée
  de `killsource.Rev` et backfill killsource (D-74). Recommandation : oui (seul ordre mesuré sans
  baisse).
- **D14 — Lots neufs.** Admettre LM (MPP 8/3, formats 24-25), LK (image-clé sous portée + `i0`
  écrivain) et LP (désaveu des déclarations d'image-clé que le bloc dit non vivantes) ?
  Recommandation : LM et LK en vague 1 (composants), après le jugement des pertes de LM (2 105
  paquets, dont 1 606 sur `1c4c63c2`) et le rejeu du critère écrit de `PorteeBaseline` pour LK
  (D-93) ; LP en vague 2 avec L1 (il lit le bloc de type 1 en production).
- **D15 — Scissions et retraits.** L3 → L3a (portage, vague 1) + L3b (vieux builds, après
  discrimination `i4`..`i9`) ; L4 → L4a (delta, vague 1) + L4b (image-clé, après LK et LM) ; L1b sort
  de la vague 2 et devient un chantier « grammaire des messages de la vue A » (41 genres de tête,
  borne estimée +8 826 sains à 45 % de gains factices près). Recommandation : oui pour les trois.
- **D16 — Condition par film de L1a.** Le seuil (50 %) est choisi sur le corpus même ; aucune
  validation hors échantillon. L'accepter tel quel, ou exiger d'abord une validation sur des films
  hors corpus (les 9 films de l'enquête, `81c02726`) ? Et deux passes (+15 070 sains) ou la forme
  causale à une passe (+13 381) ? Recommandation : validation hors échantillon d'abord (recherche,
  sans sortie), forme causale si le plafond D11 est menacé par la seconde passe.
- **D17 — Surcouche unique post-J12** (complète D10). Les surcouches des chantiers sont mutuellement
  incompatibles sous le tag commun et quatre sur cinq sont d'avant J12 (D-100). Recommandation : UNE
  surcouche post-J12 (base `r_fusion_overlay_postj12/`) réunissant les crochets des cinq chantiers,
  avant R-COMB-2 et toute mesure en surcouche de la vague 1.

### 6.4 Statuts proposés pour le §2 (à reporter par le superviseur ; ce §6 ne modifie pas le §2)

- **1.3** : `[~]` pour la borne basse indépendante (BIS_1 §1 : HI_1_13_0 ≥ 45,0 %, corpus ≥ 24,2 %) ;
  `[!]` pour le dénominateur exact, qui attend la lecture de l'appelant de `FUN_14076b0e8` dans Ghidra.
- **4.1** : `[x]`, avec MESURES_CIBLEES (et sa section de corrections) et BIS_1 à BIS_3.
- **5.1** : `[x]`, rapport révisé le 2026-10-02.
- **5.2** : `[x]`, ce §6 révisé le 2026-10-02.
- **En-tête du plan** : passer à « phase 1 close, phase 2 en attente de décisions ».
- **Gate 1** : trace déjà dans le journal du §4, à compléter par les gates des mesures bis (BIS_1
  §10, BIS_2 §8, BIS_3 §10). Il manque `go test ./internal/archlint/` (critique n° 2, N2 ; D-66) :
  à jouer et consigner avant de clore la phase 1.

### 6.5 Recherches préalables du 2026-10-02 : résultats et effet sur les lots

**Provenance.** Workflow `wf_9088d8bd-e43` : six chantiers en worktrees temporaires détachés sur
`fe18bf67c` (avant J12), chacun suivi d'un vérificateur adverse qui a recalculé les chiffres par awk
sur les TSV et relu Ghidra en lecture seule, sans commande `go`. Notes et pièces intégrées le
2026-10-02 sous `.ai/V7.5/film_re/campagne_grammaire_2026-10-01/` (176 fichiers copiés à l'identique,
aucune collision) : `R_COMB.md`, `R_LOC.md`, `R_NAIS.md`, `R_COMP.md`, `R_VEH.md`, `R_FUSION.md`,
leurs dossiers `r_*_tsv/`, `r_*_ghidra/`, `r_*_overlay/`, et 25 sondes `r_*_research_test.go` (24 sous
`grammar/`, une sous `internal/himap/`). Les chiffres sont mesurés sur `fe18bf67c` ; la carte v2 étant
identique après J12 (§6.0), ils valent pour la tête, sauf ce qui passe par une surcouche d'avant J12
(D-100 ; J12 étant prouvé neutre, l'écart attendu est nul : supposé, non mesuré).

Convention : « mesuré » = compté sur les films ; « établi » = lu (Ghidra ou code) et confirmé par la
mesure ; « estimé » / « supposé » = non mesuré. Un item que le vérificateur ne confirme pas est
marqué **NON CONFIRMÉ** avec sa raison.

#### 6.5.1 Statut de chaque recherche

| Recherche | Statut | Résultat (chiffres) | Verdict adverse | Note |
|---|---|---|---|---|
| R-COMB | mesuré | Six leviers ensemble (L1 oracle, L8, L2, L9, L6a, L6b), 20 films. Indicateur (utiles sains / fixe de R-COMB) : HI_1_13_0 69,8 → **86,0 %** (ordre D-RI) ou **88,0 %** (L1 dérivé sans L9) ; HI_1_12_0 24,1 → 88,8 % ; HI_1_8_0 → 51,3 % ; HI_1_9_0 → 37,0 % ; HI_1_11_0 → 35,4 % ; HI_1_10_0 → 22,2 % ; vieux builds ≤ 0,7 % ; corpus 39,1 → 50,6 / 51,3 % ; `81c02726` 77,1 → 98,2 %. Variable HI_1_13_0 80,3 → 86,9 / 88,3 %. Les leviers ne s'additionnent pas (HI_1_13_0 : somme des seuls +547 609 utiles sains, combinaison +465 529 ou +525 771). Aucun film en baisse nette | tient (chiffres recalculés) ; corrections : « L1 +24 916 dans la combinaison » vaut pour l'ordre D-RI seulement (+26 620 sinon) ; mécanisme L1×L2 estimé ; le gate 2 est NET par film (tenu), les pertes brutes sont une information | `R_COMB.md` |
| R-COMB, phrase « la phase 2 seule n'atteindra pas 95 % » | partiel | Mesuré : aucun build n'atteint 95 % sous les six leviers. HI_1_13_0 : il manque 200 319 utiles sains (6,95 points) | **NON CONFIRMÉ pour la partie estimée** : « au plus ~55 700 records, ~90 % » n'est pas une borne (densité moyenne 5,95 par paquet, alors que les paquets fermés en portent 9,09 : ~85 100 records, ~91 % ; seule borne stricte = 99,5 %) ; L6a hors Live Fire était oublié (mesuré depuis par R-L6 : −15 sains). Reformulation retenue au §6.5.3 | `R_COMB.md` §7 |
| R-L1 (a) | établi | L'écrivain n'écrit jamais un NEW après un DELTA dans un paquet (Ghidra : ordre NEW → DELTA → DEL, budget, retrait du record qui ne tient pas). 1 289 liaisons : X attesté 1 094 (84,9 %) ; paquet relu depuis l'occurrence ferme sain 5 / 2 143 (témoin 2 / 177) ; `R(6)` = pont 0 / 33 | tient ; corrections : sans les 100 « réalloués » (reposent sur D-43), 994 / 1 289 (77,1 %) ; pas de témoin d'attestation fortuite ; tables pont et décalage hors du livrable (D-101) | `R_NAIS.md` §1 |
| R-L1 (b) | établi | Les 4 598 fermés après un rejet sont FACTICES : eid nul 0 / 4 598 ; un début de vue C antérieur ferme 4 402 / 4 598 (95,7 %) contre 11,0 % au témoin ; juge actuel : 3 590 contredits, 1 008 « sains » | tient ; corrections : témoin non symétrique (sous-estime le taux nul sans pouvoir inverser l'écart), échantillon ≈ 1/8 et non 1/16, témoin complet 11,8 % ; « débordement du terminateur » = déduction ; T3-C2 sans perte saine SEULEMENT si l'invariant entre au juge | `R_NAIS.md` §2 |
| R-L1 (c) | structure établie ; gain partiel (estimé) | Vue A = messages déjà sérialisés recopiés sans longueur (`FUN_142f2c050`, `FUN_140bbd474`, lecteur `FUN_14080a9d4`, 123 genres) : se lit jusqu'au bout seulement en portant la charge de chaque genre. Après LS, 26 427 paquets non localisés ; 41 genres de tête, 56,9 % sans grammaire Go. Oracle « départ de vue C » : +8 826 sains / +83 609 utiles sains, dont 45 % de gains factices | tient ; corrections : statistiques et oracle faits sur la région de l'ordre de l'enquête (26 421), pas sur l'ordre retenu ; HI_1_13_0 +1,8 point sur cet ordre-là, non mesuré sur l'ordre retenu ; ce n'est pas un majorant ; « table des gestionnaires construite à l'exécution » = supposé fort | `R_LOC.md` §4 |
| R-L1 (d) | établi sur le corpus | Condition par film : allocateur à cinq pools ≥ 50 % des NEW propres (≥ 30 NEW). +15 070 paquets sains, +159 864 utiles sains, 0 film en baisse (deux passes) ; causal +13 381 / +141 294, 0 film en baisse. HI_1_13_0 70,54 → 75,93 % (fixe de R-L1 (d)). D-8 réfuté (bit à 1 partout) | tient ; corrections : les pourcentages « variables » de la note sont BRUTS (en sains : HI_1_13_0 80,30 → 84,69 %) ; pool unique max 2,80 % et non 2,3 % ; la condition désactive L1a sur HI_1_8_0 à HI_1_11_0 (gain nul) ; seuil choisi sur le corpus, sans validation hors échantillon | `R_NAIS.md` §3 |
| R-LS | établi | Code relu ; registre : un seul archétype `high-frequency` (`ti=4`) sur 30 films. Carte v2 : ordre retenu +18 119 / −2, +18 086 sains, +169 439 utiles sains, 0 film en baisse ; région (ii) 48 720 → 26 427 ; HI_1_13_0 68,6 → 73,8 % (fixe de R-LS). Killsource : 1 080 morts du `scan` à la marche, valeurs inchangées ; 362 / 403 de l'enquête retrouvés | tient ; corrections : voie publiée (D-74), donc `killsource.Rev` et backfill DUS ; 99,4 % (et non 99,5 %) des signatures sur les slots du registre ; repli libre net −14 141 (16 673 = perte brute) ; « variables » = bruts ; l'ordre retenu n'est mesuré que par la sonde | `R_LOC.md` §2-§3 |
| R-P6 | partiel | Pas de règle générale. Filtres d'occurrence : `ferme` 111 liaisons +344 / −26 (témoin 111 contre 47) ; `ferme+suivant` 40 liaisons +474 / −4 (39 contre 5). Pas de lot P6 | tient ; corrections : le filtre `propre+pont` de BIS_3 (68 / 72 contre 6 / 72, +295 / −2) est meilleur et omis ; taux de faux positifs sous-estimés (témoin biaisé) ; extension à (iii') au-delà de 3 paquets extrapolée | `R_NAIS.md` §4 |
| R-L3 | portage : établi ; vieux builds : partiel | Registre `ti=2` identique sur 21 films (empreinte `cb77b82e`) : ni le registre ni `i15` ne sont la cause. A/B delta du portage : +3 575 sains, +53 521 utiles sains, 0 sain perdu, 0 film en baisse. Vieux builds : images-clés moteur 0 / 98 → 98 / 98 avec un bit de moins ; témoins `i15 ±1` 0 / 715 | **NON CONFIRMÉ : la localisation « un bit de trop dans `i4` »**. Le TSV du chantier montre qu'un décalage de −1 à l'entrée de `i5` à `i10` ferme autant : le bit est dans `i4`..`i9`, non discriminé (`i5`..`i9` jamais testés en lecture alternative). Le reste tient (dénominateur variable : baisse sur 5 films et non 4 ; le témoin `i15 −1` gagne +130 sains sur `1c4c63c2`) | `R_COMP.md` §2 |
| R-HOM | établi | 326 noms : 314 à table unique, 8 par tableau de noms, 1 absent de HI_1_13_0, 3 à deux tables dont UN homonyme de grammaire (`high-frequency` : `ti=3 i1` = `R(16)+R(8)+R(2)`, `ti=4 i0` = `R(8)`) ; `simulation-state*` : thunks vers la même fonction. 147 / 147 concordances d'enregistrement | tient (Ghidra relu, thunks décodés à la main) | `R_COMP.md` §3 |
| R-P3 | partiel | Comptes de référence 694 / 658 / 241 / 161 reproduits. Lecteurs justes (Ghidra). G1, G3 : liaison fausse du slot ; G2 : en-tête mal lu (92,8 % sans allocation) ; G4 : `consumeObjectPositionMonde`. Désaveu hors bloc : +694 sains, 0 perdu, +7 800 utiles sains, G1 → 388. Live Fire (position au jeu + index) : G4 160 → 0, `0797ce72` +495, `60ae07c4` −28 | tient ; corrections : l'oracle prouve une liaison fausse vers une UNITÉ, pas un bipède (forcer `ti=40` fait au moins aussi bien) ; G4 → 0 exige la COMBINAISON des deux lectures ; le cas (a) ressemble à une erreur de la marche d'image-clé | `R_COMP.md` §4 |
| R-L4 (a) | établi (MPP 8 bits : mesuré) | Trois lectures communes en image-clé (portée, `i0` écrivain, MPP 8/3) : 3 058 / 3 058 voisins fermés (production 14, BIS_2 137) ; formats 20-21 0 / 2 036. Oracle `n2` (`0x8d8`) 4 166 / 4 189 | tient ; corrections : MPP 8 bits = mesuré, pas établi ; `n2` version-31 = `0x890` ; la classe « non-VTOL » de la note inclut le type 6 (VTOL) ; un 6e écrivain de la portée (`FUN_142e31bf8`) | `R_VEH.md` §1 |
| R-L4 delta | mesuré | 16 composants `ti=40`, porte posée en delta : +1 436 / −0, 35 contredits, +26 843 utiles, 21 films | tient ; correction : la perte de 9 573 de la « portée sur les NEW » est celle de la combinaison portée + `i0`, pas de la portée seule | `R_VEH.md` §3.1 |
| R-L4 (b) | partiel | MPP lu à 9 bits au lieu de 8 sur les formats ≤ 25 : 0 / 5 417 châssis connus à 9/5, 4 526 / 5 417 à 8/3 ; format 27 : 1 457 / 1 642 à 9/5, 0 à 8/3 | **NON CONFIRMÉ : « lecture fausse » pour `4118381d` et `d0b40d0a`** (établi pour `77ef810a` seulement) : `d0b40d0a` se relit à l'identique sous 8/3 à une ancre décalée, sur un film des formats 20-21 ; 878 châssis restent inconnus à 8/3 ; réutilisation de slot non exclue | `R_VEH.md` §2 |
| R-L6 | établi / réfuté | Ordre des plages établi (Ghidra `FUN_140be9a14`, `FUN_140770640` ; modules : plage 0 = arène, 1 = décor lointain ; Illusion et Fragmentation : 1 plage). D-54 réfuté (0 fermé à index impossible en lecture finale). L6a hors Live Fire réfuté : +15 / −18, −15 sains | tient ; corrections : 13 films hors Live Fire et non 12 (9 sur 13 en baisse) ; le relevé type BIS_2 reproduit vaut 102 / 139, pas 101 / 136 | `R_VEH.md` §5 |
| R-VEH-1 (découverte) | partiel | Portée + `i0` écrivain en image-clé ferment `ti=35` (format 27) : 142 → 2 006 / 2 008 | **NON CONFIRMÉ : « critère de bascule de `PorteeBaseline` rempli »** (critère écrit = 591 records bornés de R7 + non-régression delta, non rejoué ; seule variante delta qui pose `i0` : −9 573) | `R_VEH.md` §4 |
| R-VEH-2 (découverte) | mesuré, pertes non jugées | MPP 8/3 sur les formats 24-25 : +80 979 / −2 105 (6 films) | tient pour les chiffres ; **NON CONFIRMÉE : l'explication** de l'ancienne contradiction (8/3 seul fait déjà monter les images-clés ; il fait baisser les formats 20-21 de la même case) | `R_VEH.md` §3.3 |
| Fusion de J12 | établi | Voir §6.0, « Items à la fusion » | « fichiers modifiés : aucun » du rendu est faux à la lettre (le worktree jetable portait la fusion non commitée et le patch des quatre sondes) ; sans effet, worktree jetable | `R_FUSION.md` |

#### 6.5.2 Effet sur les lots (proposé ; ce qui relève de l'utilisateur est en §6.3)

- **Prêts pour la vague 1** (composants ; gain mesuré, gate par film tenu dans la mesure) : L8
  (prérequis levé), L2, L3a, L4a (sous D5), L6a limité à Live Fire et étendu au site `ti=41 i0`, L6b
  (sous D4), L9.
- **À ajouter (D14)** : LM (MPP 8/3, gain massif sur les formats 24-25 mais pertes à juger) ; LK
  (lecture d'image-clé sous portée + `i0`, prérequis de L4b, critère à rejouer) ; LP (désaveu hors
  bloc, +694 sains, 0 perdu, lit le bloc de type 1 : vague 2).
- **À scinder (D15)** : L3 → L3a / L3b ; L4 → L4a / L4b.
- **À retirer ou déplacer (D15)** : L1b sort de la vague 2 (chantier « grammaire des messages de la
  vue A ») ; L1c reste retiré ; pas de lot P6 (D3) ; L6a hors Live Fire non retenu.
- **Gates modifiés** :
  - L1 : joué sur une tête qui porte L9, témoin `4f77afc1` (D-69) ; un L1 réel doit faire mieux
    que son oracle sur `1c4c63c2` et `084a804d` en brut (information ; le critère reste net) ;
  - LS : delta killsource déclaré, `killsource.Rev` et backfill DUS (D-74) ; `ScanMarchFacts` à
    mesurer ; aucun repli libre dans la cuisson ;
  - LU : zéro différence avec un paramètre d'ordre (D-73) ;
  - L2 et L1 : le recouvrement (D-70) se mesure dans R-COMB-2, pas par addition ;
  - L0 : invariants L0.6 à L0.9 ;
  - tout gate en surcouche : surcouche unique post-J12 (D17).
- **Gains à ne plus additionner** : les gains « seuls » du §6.1 se recouvrent (R-COMB) ; seule une
  combinaison mesurée (R-COMB-2) dit ce que donne une vague.

#### 6.5.3 Indicateur (D1) et dénominateur fixe consolidé

Calcul fait le 2026-10-02 pour ce document (awk, aucune commande `go`) : par film, maximum de la
colonne fixe des quatre TSV `r_comb_tsv/r_comb_denominateurs.tsv`,
`r_loc_tsv/rloc_denominateur_fixe_max.tsv`, `r_nais_tsv/r_nais_denominateur_fixe.tsv` et
`r_veh_tsv/r_veh_delta_synthese.tsv` (colonne 3, TSV sans en-tête : lue comme `fixe_max`, valeur
recoupée sur `084a804d` = 731 523, citée par `R_VEH.md` §3.3) ; numérateurs : colonne
`utiles_fermes_sains` de `r_comb_par_build.tsv`.

| Groupe | Fixe R-COMB | Fixe consolidé | Référence | Six leviers, ordre D-RI | Six leviers, L1 dérivé sans L9 |
|---|---|---|---|---|---|
| HI_1_13_0 (10 films) | 2 880 403 | **2 959 104** | 67,9 % | **83,7 %** | **85,7 %** |
| HI_1_12_0 | 146 098 | 146 098 | 24,1 % | 88,8 % | 88,8 % |
| HI_1_8_0 | 290 393 | 316 124 | 26,2 % | 47,1 % | 46,5 % |
| HI_1_9_0 | 255 371 | 316 305 | 20,6 % | 29,8 % | 29,7 % |
| HI_1_11_0 | 305 812 | 359 795 | 22,1 % | 30,1 % | 30,0 % |
| HI_1_10_0 | 1 661 058 | 2 030 145 | 14,6 % | 18,1 % | 17,8 % |
| Corpus (20 films) | 6 585 067 | **7 176 150** | 35,8 % | 46,4 % | 47,1 % |

Le fixe consolidé monte surtout par LM (formats 24-25) et LS (HI_1_13_0, `c75f33b8`, `d9781168`,
`1c4c63c2`), qui lisent plus loin mais ne sont pas dans le numérateur de R-COMB : ces pourcentages
sont donc plus bas que ce que donnerait la combinaison avec LS et LM (non mesurée).

**Phrase « la phase 2 seule n'atteindra pas 95 % », reformulée (vérificateur de R-COMB, complétée)** :
- mesuré : 86,0 à 88,0 % sous les six leviers sur HI_1_13_0 (fixe de R-COMB ; 83,7 à 85,7 % sur le
  fixe consolidé) ; aucun build à 95 % ;
- HI_1_13_0 : environ 90-91 % avec L3, L4, L7 et R-P6 SI leurs paquets ont la densité moyenne
  (hypothèse ; il faudrait 21,4 records par paquet pour 95 %) — mais LS (+5,2 points seul sur
  HI_1_13_0, fixe de R-LS) et LP n'étaient pas dans R-COMB et recouvrent en partie L1 : la phrase
  n'est **plus décidable** pour HI_1_13_0 sans R-COMB-2 ;
- HI_1_12_0 : non décidé ; formats 24-25 : non décidé depuis LM (72 à 84 % par film sous LM et les
  composants `ti=40`, hors leviers de R-COMB) ; formats 20-21 : la phrase tient largement (≤ 0,7 %).

#### 6.5.4 Ordre proposé (révise la recommandation D-RI du §6.3, sans changer les décisions FERMES du §3)

1. Avant la vague 1 (recherche, sans sortie) : surcouche unique post-J12 (D17) ; R-COMB-2 ;
   jugement des pertes de LM ; discrimination `i4`..`i9` pour L3b ; rejeu du critère de
   `PorteeBaseline` pour LK ; validation hors échantillon de la condition de L1a (D16).
2. **Vague 1, composants**, dans l'ordre du gain mesuré : L8 (+30 618), L2 (+18 105), LM (si admis,
   +80 979 sur 6 films, pertes jugées), L3a (+3 575 sains), L6a Live Fire + `ti=41 i0`, L9
   (+1 784), L4a (+1 436), L6b, puis LK et L4b (si admis). Une recuisson (D7).
3. Références re-figées.
4. **Vague 2, marche** : LU (avec paramètre d'ordre) → LS (+18 119) → L1a sous condition par film
   (+15 070 sains) → LP (+694 sains) → L7. Une recuisson.
5. Étape 1 de la représentation intermédiaire (D-RI inchangée).
6. Hors vagues : chantier « grammaire des messages de la vue A » (ex-L1b) ; L3b dès que la position
   du bit est discriminée.

#### 6.5.5 Intégration du 2026-10-02 (rapportée par l'agent d'intégration, non rejouée ici)

- 176 fichiers copiés à l'identique (`cmp`), aucune collision ; aucun fichier suivi modifié ;
  `git status` : fichiers non suivis seulement ; tête `da7c2c764` (postérieure à J12, alors que les
  chantiers sont partis de `fe18bf67c`).
- Deux sondes intégrées corrigées pour l'accesseur de J12.4 (`r_veh_chassis_research_test.go:192`,
  `r_comb_research_test.go:184`) ; quatre `overlay_campagne.json` écrits (R-COMB, R-COMP, R-LS, R-L4),
  32 chemins vérifiés.
- Gates : `gofmt -l` vide sur les 41 `.go` intégrés (tag en ligne 1, plus grand fichier 462 lignes) ;
  `go vet -tags=research ./internal/games/halo_infinite/film/... ./internal/himodule/` rc 0 après
  les deux corrections (rc 1 avant) ; `go vet -tags=research ./internal/himap/` rc 0 ;
  `go vet -tags=research,campagne_overlay` : vert avec `r_veh_overlay`, ROUGE avec les surcouches de
  fusion, R-COMB, R-COMP et R-LS (D-100) ; `go test -count=1 ./internal/archlint/` ok (32 s) ;
  `go test -count=1 -run 'Closure|Fermeture|GrammarRev'` sur `grammar` ok (12 PASS) ; `grammar.Rev`
  = `grammar-2026-09-27.3`.
