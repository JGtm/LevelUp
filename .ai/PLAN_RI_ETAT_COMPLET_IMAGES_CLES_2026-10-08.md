# PLAN — Lecture de l'état complet du bipède aux images-clés (LK), puis 2.7.d1 (2026-10-08)

> **Statut : EN COURS — GO de l'utilisateur du 2026-10-08 (« Je suis ok »), décisions datées au
> §3.1.** Rédigé le 2026-10-08 par la seconde passe du workflow 2.7.d1 ; il remplace le plan de la
> première passe (`scratchpad/ri/d1/wf1_plan.json`) et intègre les corrections de ses deux
> contre-vérifications (`wf1_verdicts.json`), les relectures Ghidra de la seconde passe et les deux
> mesures (mesure 1 : `ref_corrigee/`, mesure 2 : `m2/`). Installé le 2026-10-08 dans
> `feat/ri-lk-images-cles` avec les treize corrections de sa propre contre-vérification
> (`REF/wf2_verdict.json`, champ `corrections` ; détail au journal, §8). Item parent : 2.7.d1 du plan
> `.ai/V7.5/PLAN_REPRESENTATION_INTERMEDIAIRE_ETAPE2_2026-10-03.md` (statué `[!]` jusqu'à la clôture de
> ce plan), découverte 36 de ce plan.
>
> **Contrat** : skill `plan-execution` (ce plan fait foi en cas de divergence) : ordre strict, une
> étape à la fois, aucune étape exécutable différée, chaque item statué `[x]` / `[~]` (référence) /
> `[!]` (justification écrite), vérification sur pièces avant de coder et avant de cocher, zéro fix
> hors périmètre (§7 Découvertes). On ne revient vers l'utilisateur que pour : un ARRÊT écrit dans
> ce plan, une fusion dans `feat/v75`, la recuisson du parc, le backfill killsource.
>
> **Pour qui** : l'agent Opus qui exécute LK puis 2.7.d1, seul, dans son worktree. Le lot A
> (exécutant du 2026-10-08) couvre l'étape 0 et LK.1 à LK.4, et s'arrête à la clôture de LK.4.
>
> **Sources (à lire avant la première étape)** :
> - `docs/adr/0037-film-intermediate-representation.md` — IR-6 (couche de récupération, règle de
>   2.7.b, paragraphe 2.7.d, dernier alinéa : « The keyframe bit windows ... still decide alone ») ;
> - `docs/adr/0034-*.md` — D-7 (faits persistés, publication rejouée depuis eux), D-10 (replis) ;
> - `.ai/V7.5/PLAN_REPRESENTATION_INTERMEDIAIRE_ETAPE2_2026-10-03.md` — item 2.7.d (2.7.d0, d1), §4
>   (gates, pièges d'exploitation), §6 découvertes 36 et 45 ;
> - `.ai/V7.5/killweapon/WALK_PORT_NOTES.md` §6 (portée, R7) ;
> - sorties de la passe de mesure, dossier `REF` (§1.4) : `README.txt` de `ref_corrigee/` et de `m2/`,
>   `wf1_verdicts.json`, et le relevé Ghidra de la seconde passe (`g2/`) ;
> - `CLAUDE.md` (règles 5, 6, 7, 11, 17).
>
> Les numéros de ligne cités datent du 2026-10-08 sur `83dc72eab` ; rouvrir chaque fichier avant
> d'agir (règle 4 du contrat).

---

## 0. Objet, critères de succès, périmètre

**Objet.** Deux jalons, dans cet ordre, sur une seule branche :

1. **LK** — la grammaire lit les records d'image-clé SOUS LA PORTÉE `DAT_144e61ea0`, là où le jeu la
   pose : autour de l'état par défaut et du mot de contrôle (`FUN_142e2bfd0`, 142e2c46f..142e2c530)
   et sur toute la boucle de l'état complet (`FUN_142e2c690`, 142e2c6b8..142e2c76a), jamais en delta,
   jamais en traversée, jamais en vue A. Sous la portée, chaque lecteur de position lit la forme du
   jeu (R(96) brut, puis, pour l'i0 absolu, la queue de `FUN_14076e3e4` et le R(2) final). Les deux
   bascules de profil `PorteeBaseline` et `GrammaireEcrivainI0` (kill-switches, échéance écrite
   2026-10-31, `profil_balayage.go:237-249`) sont retirées à la bascule (règle 11).
2. **2.7.d1** — les armes portées, l'inventaire et la marque de portage des images-clés sont lus
   par la grammaire (un canal de la phase des images-clés) ; les fenêtres de bits passent derrière,
   en replis nommés, ordonnés « après la lecture », comptés, leurs records marqués
   `PreuveRecupere` (IR-6, option A de l'utilisateur du 2026-10-04).

**Critères de succès (mesurables).**
1. LK : sur les 28 films, en contexte de cuisson, les records bipèdes d'image-clé fermés passent de
   410 à au moins 5 399 sur 10 710 (référence R-2) ; aucun autre archétype ne perd une fermeture ;
   l'élection des ancres, les preuves d'image-clé et les équipes sont identiques à la base ; la carte
   des trames delta est identique à l'octet.
2. LK : un ratchet COMMITÉ, joué en CI, garde la fermeture `ti=35` en contexte de cuisson sur les 7
   bobines du golden (53/1 368 en base, 666/1 368 sous LK, référence R-7).
3. 2.7.d1 : chaque valeur d'image-clé publiée vient de la grammaire sur les records admis (règle
   d'admission U-1) et de la fenêtre seulement sur les autres, comptés au record près ; aucune fiche
   ne perd de données (records avec armes, grammaire + repli, au moins égaux à la fenêtre seule,
   film par film).
4. Coût : pas plus de 10 % de durée ni de pic mémoire, cuisson ET lecture des porteurs au sync, sur
   4 films (critère 4 du plan de l'étape 2), mesuré en D1.1 et en D1.4.
5. Faits : la marque « récupéré » et les comptes des trois nouveaux replis sont identiques entre une
   cuisson fraîche et une publication rejouée depuis les faits (ADR 0034) ; sinon `SchemaDesFaits`
   monte (D1.3).

**Hors périmètre** (explicite ; ce qui s'y rencontre va au §7) :
- la forme en ligne du delta d'i0 (`consumePositionHandleTail` sous `bUsePred`/keep-baseline,
  `bHandle = 1`) : même largeur de handle fausse (`pd.IndexW` au lieu de 13), relevée par Ghidra —
  découverte D-1, non traitée ;
- `GrammaireEcrivainI0` en delta : mesuré marginal (+2 trames saines au mieux, 0 perdue, R-9) ; non
  étendu (décision E-3) ;
- `consumeAbsoluteWithGate` sous `fullPrecision()` seul (`DAT_145121140`, hors portée) ;
- les gardes `etatComplet` de `ti=40` (lot L4b de la campagne) ;
- le résidu non fermé sous LK (formats 20 et 21, `60ae07c4` +33 bits, écarts multiples de −108,
  arrêts sur i59/i58) : instruit et nommé, pas corrigé ;
- toute cuisson du parc, tout backfill, toute ouverture d'une base DuckDB ; Halo 5. **Une seule
  exception, nommée** : G-corpus (LK.6.5, D1.4.4) fait lancer `levelup replay-facts-export`, qui ouvre
  la base partagée EN LECTURE par `OpenReadForQuery` (`cmd/replay-corpus-gate/facts.go`, sous
  `parcRoot/data/titles/{slug}/warehouse`), jamais en écriture. Admis parce qu'aucun backfill ne tourne
  (le backfill killsource du 2026-10-08 est fini, journal du plan de l'étape 2) ; vérifier avant chaque
  passe de G-corpus qu'aucun backfill ni aucune cuisson du parc n'est en cours, sinon attendre.

## 1. Base, branche, worktree, coordination

### 1.1 Branche et worktree
- Branche **`feat/ri-lk-images-cles`**, créée par le superviseur le 2026-10-08 depuis
  `feat/ri-etape2` **`81d4831d6`** (= `feat/v75` `a4515e66c` + le commit de la seconde passe de mesure,
  documents seulement ; le code est celui de `83dc72eab`, sur lequel les références §2 ont été
  mesurées). Partir de là règle le conflit documentaire que `83dc72eab` aurait garanti à la première
  fusion (correction 12 de la contre-vérification : `a4515e66c` modifie le plan de l'étape 2 et le
  thought_log). Préfixe `feat/` : la CI se déclenche. Worktree DÉDIÉ, RÉUTILISÉ :
  `C:/Users/Guillaume/Downloads/Scripts/LevelUp-wt-ri` (il n'y a plus de `LevelUp-wt-lk`), un seul
  exécutant.
- Jonctions vers le checkout principal DÉJÀ EN PLACE dans ce worktree (vérifiées à l'étape 0, créées
  par `cmd //c mklink /J`, jamais `ln -s`, qui copie) : `data/cache/film_chunks`,
  `data/cache/film_manifests`, `apps/web/node_modules`. Retirées par `(Get-Item <jonction>).Delete()`
  AVANT tout `git worktree remove` (jamais `--force`).
- `feat/v75` a avancé depuis (`21a680108`, fusion des lots sync « connu = au registre » et noms du
  registre : aucun fichier sous `film/`). Elle n'est PAS fusionnée pendant le lot A (consigne du
  superviseur) ; un commit de `feat/v75` qui toucherait les fichiers de ce plan est signalé, jamais
  fusionné sans accord. La fusion du §1.3 se fait à l'ouverture du jalon suivant.
- `data/cache/film_facts` du worktree : dossier RÉEL (jamais une jonction), renommé avant chaque
  passe de preuve (§5, pièges).
- Cache Go : `GOCACHE='C:\Users\Guillaume\AppData\Local\go-build-ri'` et
  `GOLANGCI_LINT_CACHE='C:\Users\Guillaume\AppData\Local\golangci-ri'`, un seul processus `go` à la
  fois ; jamais `go build ./...` (disque).

### 1.2 Reprise des instruments de recherche (patch relu, jamais le crochet)
Le worktree de mesure `LevelUp-wt-imagecle` (`feat/imagecle-etat-complet` = `83dc72eab`) porte des
modifications NON COMMITÉES qui y restent. Sont repris, par un patch relu fichier par fichier :
- `grammar/ri27d0_images_cles_research_test.go` (instrument 2.7.d0 corrigé : classes a/b/c) ;
- `grammar/ri27d1_instrument_research_test.go` (admission A/B/C, témoin 109, formats, M/MX/MB, Y, R) ;
- `grammar/ri27d1_fermeture_portee_research_test.go` (`TestRI27d1FermetureCorpus`, `TestRI27d1Formats`) ;
- `grammar/ri27d1_m2_critere_research_test.go`, `grammar/ri27d1_m2_ancres_research_test.go`.

Ne sont PAS repris : le crochet `var rechercheLK bool` et son bloc dans
`grammar/keyframe_fullstate_loop.go` ; `grammar/ri27d1_crochet_env_research.go` ;
`research/cmd_fermeture/ri27d1_variante.go` et la ligne ajoutée à `research/cmd_fermeture/main.go`.
Le patch retire des fichiers repris toute référence à `rechercheLK` et à `RI27D1_PORTEE` /
`RI27D1_LK_ENV` / `RI27D1_I0E_ENV` : dans le worktree LK, l'instrument n'a qu'UN mode, celui du code
de production à la tête (base avant LK.3, portée après). Sont aussi laissés de côté, parce qu'ils ne
mesuraient que l'écart entre la tête et le crochet : `TestRI27d1FermeturePortee` (les bobines du
golden en contexte golden, que `TestKeyframeClosureRatchet` tient déjà) et `TestRI27d1M2Bobines` avec
son contexte `ri27d1M2ContexteCarte` (la mesure R-7 devient le ratchet commité de LK.2.4 ; garder
l'instrument ferait une copie de plus du contexte de cuisson, règle 6, et un instrument redondant,
règle 7). Noms des sorties, fixés par le patch (une étiquette de mode unique, `tete`) :

| Instrument | Sortie (dans `$S/<e>`) | Comparée à `REF` (avant LK.3 / après LK.3) |
|---|---|---|
| I-d0 | `images_cles.tsv` | `ref_corrigee/images_cles_base.tsv` / `ref_corrigee/images_cles_portee.tsv` |
| I-ferm | `fermeture_corpus.tsv` (colonne 1 = `tete`) | lignes `base` / `portee` de `fermeture_corpus.tsv`, colonne 1 retirée des deux côtés |
| I-ancres | `m2_ancres_tete.tsv`, `m2_stats_tete.tsv` | `m2/ancres/m2_{ancres,stats}_base.tsv` / `m2_{ancres,stats}_lk.tsv` |
| I-equipes | `m2_equipes.tsv` (colonne 1 = `tete`) | lignes `base` / `lk` de `m2/ancres/m2_equipes.tsv`, colonne 1 retirée |

### 1.3 Coordination
- Le plan de l'étape 2 (2.7.d1 `[!]`) renvoie à ce plan ; il est mis à jour à chaque clôture de
  jalon (LK.6, D1.4).
- Avant chaque jalon : `git -C <WT> fetch` et, si `origin/feat/v75` a bougé, `git merge
  origin/feat/v75` (en conflit, `feat/v75` a raison), puis rejouer les références de l'étape 0 sur la
  tête fusionnée. Une session de la campagne qui touche `grammar/keyframe_*`, `components_position_i0.go`
  ou `lecteur_position*.go` est prévenue avant la première modification de ces fichiers.

### 1.4 Variables communes des commandes (bash, depuis `$WT/apps/go-api`)
```bash
export GOCACHE='C:\Users\Guillaume\AppData\Local\go-build-ri' GOLANGCI_LINT_CACHE='C:\Users\Guillaume\AppData\Local\golangci-ri' PATH="/c/msys64/ucrt64/bin:$PATH" CGO_ENABLED=1
WT=/c/Users/Guillaume/Downloads/Scripts/LevelUp-wt-ri
R=C:/Users/Guillaume/Downloads/Scripts/LevelUp/data/cache/film_chunks
REF=C:/Users/GUILLA~1/AppData/Local/Temp/claude/c--Users-Guillaume-Downloads-Scripts-LevelUp/2efb0786-9fba-4f95-8fd0-464ee47b91d7/scratchpad/ri/d1
S=$REF/../lk                       # scratchpad de la session (le même que REF) ; un sous-dossier par étape : $S/e0, $S/lk3, ...
G=./internal/games/halo_infinite/film/internal/grammar/
FILMS="000d5950,01e1f945,084a804d,111fa685,11de8353,1c4c63c2,50247b26,51101d1d,53ce4390,60ae07c4,64e8adfa,696a9d7c,7344d24f,9f57c612,a349fea8,a521164d,bcb6d393,d9781168,e5adf7b2,fb1a1a72,0797ce72,396cfc92,4f77afc1,51ebbc0f,bf15f7ab,bfecd02b,c75f33b8,f75e7053"
CARTES="0797ce72=Live Fire;396cfc92=Illusion;4f77afc1=Flood Gulch;51ebbc0f=Banished Narrows;bf15f7ab=Perilous;bfecd02b=Snowbound;c75f33b8=Curfew;f75e7053=Solitude - Ranked"
F20="bcb6d393,fb1a1a72,d9781168,c75f33b8,bf15f7ab,51ebbc0f,084a804d,0797ce72,111fa685,e5adf7b2,60ae07c4,a349fea8,a521164d,11de8353,50247b26,bfecd02b,4f77afc1,396cfc92,f75e7053,1c4c63c2"
F8="000d5950,01e1f945,51101d1d,53ce4390,64e8adfa,696a9d7c,7344d24f,9f57c612"
```
Commandes nommées, réutilisées par les gates (§5) :
- **I-d0** (instrument des images-clés, 28 films, ~20 s) :
  `RI27D1_RECORDS=60ae07c4,50247b26 RI27C_FILMS="$FILMS" RI27C_RACINE=$R RI27C_OUT=$S/<e> RI27C_CARTES="$CARTES" go test -tags=research -count=1 -run '^TestRI27d0ImagesCles$' -timeout 9m -v $G > $S/<e>/images_cles.log 2>&1`
  puis `bash $REF/agg.sh $S/<e>/images_cles.tsv > $S/<e>/agg.tsv`.
- **I-ferm** (KeyframeClosure, 28 films, contexte de cuisson) — CORRIGÉE (correction 1 de la
  contre-vérification : avec `RI27D1_OUT` seul, `ri27cEnv` saute le test, la commande sort PASS sans
  écrire de fichier et le gate passerait à vide) :
  `RI27C_FILMS="$FILMS" RI27C_RACINE=$R RI27C_OUT=$S/<e> RI27C_CARTES="$CARTES" go test -tags=research -count=1 -run '^TestRI27d1FermetureCorpus$' -timeout 9m -v $G > $S/<e>/fermeture.log 2>&1`.
  Contrôle obligatoire avant toute comparaison : le journal porte `--- PASS: TestRI27d1FermetureCorpus`
  et non `--- SKIP`, et `$S/<e>/fermeture_corpus.tsv` existe (839 lignes). Comparaison :
  `diff <(awk -F'\t' '$1=="base"' $REF/fermeture_corpus.tsv | cut -f2-) <(cut -f2- $S/<e>/fermeture_corpus.tsv)`
  (`portee` à la place de `base` après LK.3).
- **I-ancres** / **I-equipes** : `RI27C_FILMS="$FILMS" RI27C_CARTES="$CARTES" RI27C_RACINE=$R RI27C_OUT=$S/<e> go test -tags=research -count=1 -run '^TestRI27d1M2Ancres$' -timeout 580s -v $G` (idem `^TestRI27d1M2Equipes$`).
  Même contrôle `--- PASS` contre `--- SKIP`.
- **I-critere** : `KF35_ROOT=$R go test -count=1 -run '^TestKF7EFullStateLoop$' -timeout 580s -v $G` et `KF35_ROOT=$R go test -tags=research -count=1 -run '^TestRI27d1M2Critere$' -timeout 580s -v $G` ;
  à l'étape 0 seulement (bascules encore en place), aussi l'instrument que la doc nomme :
  `KF35_ROOT=$R go test -count=1 -run '^TestKF35CBaselineScope$' -timeout 580s -v $G` (correction 7).
- **I-carte** (carte v2 delta) : `go build -tags=research -o $S/<e>/cmd_fermeture.exe ./internal/games/halo_infinite/film/research/cmd_fermeture` puis
  `$S/<e>/cmd_fermeture.exe -racine $R -films $F20 -sortie $S/<e>/carte -plafond-gib 4 -top 40 -mode v2 -paquets`,
  idem `-films $F8 -sortie $S/<e>/carte8`, et `-films $F20,$F8 -mpp-declare -sortie $S/<e>/carte_mpp` (cet ORDRE est celui de la référence `m2/carte_v2_mpp_base` : la carte écrit ses lignes dans l'ordre des films, et `-films $FILMS` rend les mêmes lignes dans un autre ordre, constaté le 2026-10-08).
  « idem » = les MÊMES drapeaux (`-racine $R -plafond-gib 4 -top 40 -mode v2 -paquets`) : sans eux, la
  carte MPP n'écrit que cinq fichiers (constaté en LK.3, 2026-10-08) ; un dossier de sortie neuf par
  passe, sans quoi les fichiers d'une passe précédente restent et faussent le `cmp`.
  Comparaison : `cmp` de chaque TSV contre la référence, `fermeture_films.tsv` par `cut -f1-16`
  (`fermeture_resume.md` porte le pic mémoire, il diffère d'une passe à l'autre).
- **Gardes de révision passées sous `-skip`** (gates intermédiaires, entre deux montées ; correction 4) :
  `SKIPREV='TestGrammarRevSuitLaGrammaire|TestChroniqueCouvreLaRevisionCourante|TestKillsourceRevSuitLaSortie|TestChroniqueDeKillsourceCouvreLaRevisionCourante|TestObjectivesRevSuitLaSortie|TestChroniqueDObjectivesCouvreLaRevisionCourante|TestChaqueRevisionEgaleSonGolden'` (le dernier, `film/revision`, rejoue l'empreinte de chaque couche contre son golden : ajouté le 2026-10-08 en LK.2, rouge comme le gate de la couche),
  puis `go test <paquets> -count=1 -skip "$SKIPREV"`. Toute source non-test de `grammar` touchée avant la
  montée du jalon (LK.2.3 compris) fait rougir `TestGrammarRevSuitLaGrammaire` : c'est attendu, et
  c'est pourquoi aucun push n'a lieu avant la montée (E-7).
- **Portes de régénération des révisions** (LK.6, D1.4 ; correction 2 : chaque porte exige le drapeau
  ET sa variable, sinon elle saute ; elle finit toujours en échec, c'est voulu, la réécriture
  s'annonce par `t.Fatal`) :
  - `grammar.Rev` : `LEVELUP_UPDATE_GRAMMAR_REV=1 go test ./internal/games/halo_infinite/film/internal/grammar/ -count=1 -run '^TestGrammarRevSuitLaGrammaire$' -update-grammar-rev` → `grammar/testdata/grammar_rev.golden` ;
  - `killsource.Rev` : `LEVELUP_UPDATE_KILLSOURCE_REV=1 go test ./internal/games/halo_infinite/film/internal/facts/killsource/ -count=1 -run '^TestKillsourceRevSuitLaSortie$' -update-killsource-rev` → `killsource/testdata/killsource_rev.golden` ;
  - `objectives.Rev` : `LEVELUP_UPDATE_OBJECTIVES_REV=1 go test ./internal/games/halo_infinite/film/internal/facts/objectives/ -count=1 -run '^TestObjectivesRevSuitLaSortie$' -update-objectives-rev` → `objectives/testdata/objectives_rev.golden` (révision ET empreinte ; `objectives_perimetre.golden` fige la fermeture des imports et ne bouge que par `LEVELUP_UPDATE_PERIMETRES=1 ... -run TestPerimetreDeChaqueCoucheEgaleSonGolden -update-perimetres`, si un import entre ou sort).

## 2. Références mesurées (chaque chiffre : commande et fichier)

Toutes rejouées ou relues dans la session de rédaction (2026-10-08), sorties sous `REF/plan_verif/`
sauf mention. Rien n'a été commité ; aucune base ouverte.

| Réf. | Chiffre | Commande | Fichier |
|---|---|---|---|
| R-1 | Base (crochet éteint), 28 films : records 10 710, traversés 9 849, fermés 410, grenades ≠ 4 9 204, armes égales 386 ; desync_59 829, desync_58 32 | `bash agg.sh ref_corrigee/images_cles_base.tsv` ; `awk '$1=="S" && ($3=="desync_59"||$3=="desync_58")'` | `plan_verif/agg_base.tsv`, `plan_verif/desync_5958.txt` |
| R-2 | Portée (LK), 28 films : records 10 710, traversés 10 622, fermés 5 399, grenades à 4 9 972, ≠ 4 380, armes égales 6 955, fenêtre en plus 1 612 ; parmi les fermés : grenades à 4 5 324, ≠ 4 4, armes égales 4 319 ; desync_59 55, desync_58 33 (88 arrêts) | idem sur `images_cles_portee.tsv` | `plan_verif/agg_portee.tsv`, `plan_verif/desync_5958.txt` |
| R-3 | Par format, LK (records / fermés / ≠ 4 / armes égales / A / B / C) : f20 1 652/15/380/41/15/1 014/623 ; f21 403/9/0/18/9/390/4 ; f24 3 097/1 852/0/2 561/1 852/1 231/14 ; f25 653/445/0/496/445/204/4 ; f27 4 905/3 078/0/3 839/3 078/1 809/18 ; témoin +1 bit : 227 fermés, dont 223 aussi fermés en vrai | relu | `ref_corrigee/par_format_portee.tsv` |
| R-4 | Accord grammaire/fenêtre par classe (LK). f24-27 A 5 375 records : armes 4 319/4 399, grenades 4 367/4 370, chargeur0 3 764/3 780, réserve0 4 147/4 367. f24-27 B 3 244 : armes 2 577/2 636, grenades 2 606/2 612, chargeur0 2 241/2 251, réserve0 2 479/2 608. f20-21 A 24 : armes 0/16, grenades 11/11, munitions 0. f20-21 B 1 404 : armes 55/1 187, grenades 1 139/1 158 | `bash ref_corrigee/taux.sh images_cles_{base,portee}.tsv` (sortie IDENTIQUE à `ref_corrigee/taux_accord_adm.txt`) | `plan_verif/taux_accord_adm.txt` |
| R-5 | Écart de fin sous LK : 0 pour 5 328, multiple de −108 pour 2 800, autre pour 2 224 dont 2 100 sur 4 films (`a349fea8` 967, `50247b26` 415, `a521164d` 387, `60ae07c4` 331) | `awk '$1=="F" && $3=="ecart_fin"'` sur `ref_corrigee/images_cles_portee.tsv` | `plan_verif/ecart_fin_portee.txt` |
| R-6 | KeyframeClosure 28 films : seul `ti=35` change (27 lignes) ; 410 → 5 399 ; une seule baisse : `60ae07c4` 2 → 1 ; bloquant `ti=35` vide sur 21 films, arrêt sur 7 (i59 : `0797ce72`, `084a804d`, `111fa685`, `60ae07c4`, `a349fea8`, `a521164d` ; i58 : `50247b26`) | `awk` sur `fermeture_corpus.tsv` (jointure base/portée) | `plan_verif/bloquant_ti35_portee.txt` |
| R-7 | Bobines du golden (7) : contexte golden (sans carte) base = LK, LK invisible ; contexte carte = contexte cuisson (215 lignes identiques) ; `ti=35` 53/1 368 → 666/1 368 (`a521164d` 1→1, `60ae07c4` 2→1, `11de8353` 12→180, `111fa685` 13→163, `e5adf7b2` 16→186, `bcb6d393` 6→78, `fb1a1a72` 3→57) ; aucune autre ligne ne bouge | `RI27D1_OUT=plan_verif/ratchet go test -tags=research -run '^TestRI27d1M2Bobines$'` (rejoué : `m2_bobines.tsv` identique à l'octet à celui de la mesure 2) | `plan_verif/ratchet/m2_bobines.tsv`, `ti35.txt` |
| R-8 | Critère écrit (591 records bornés annoncés, 599 mesurés) : (d+e) = 125/192 + 124/209 + 120/198 = **369/599 = 61,6 %**, AVEC et SANS bouchons, bascules du profil = crochet ; REF 25/599 = 4,17 % ; **(d) portée seule = 1/192 + 5/209 + 2/198 = 8/599 = 1,3 %** (le critère tel qu'écrit, portée seule, n'est PAS tenu : sa tenue passe par l'amendement U-0) ; (e) seul 25/599 (= REF) ; `TestKF35CBaselineScope` : 0/599 sur toutes ses variantes (instrument aveugle : `walkKeyframeBody` ne pose pas le contexte) | `KF35_ROOT=$R go test -tags=research -run '^TestRI27d1M2Critere$'` (rejoué, 3,5 s) ; logs de la mesure 2 relus | `plan_verif/critere_rejoue.log` ; `m2/c1_kf7e_avec_bouchons.log`, `m2/c1_kf35c_baseline_scope.log` |
| R-9 | Carte delta : base = LK à l'octet (13/14 TSV sur 20 films, 13/14 sur 8, 14/15 sous MPP ; le TSV restant identique colonnes 1-16) ; sains 387 241 (20 films) et 749 981 (28 films, MPP) dans les deux modes. Variante i0 de l'écrivain en delta : sains 387 243 (+2), 749 981 (=) | `cmp` des TSV de `m2/carte_v2_*` ; `awk` colonnes 6-7 de `fermeture_paquets.tsv` | `plan_verif/carte_cmp.txt`, `plan_verif/carte_sains.txt` |
| R-10 | Élection : colonnes 1-9 des 477 665 lignes (461 562 ancres, 16 103 écartés) identiques ; seules diffèrent les colonnes de fermeture du record ; totaux identiques (élections 59 294, réfutations 174, recalages 3 720, glissements 190, prouvés 180 012, écartés prouvés 3) sauf `fermes` 182 439 → 187 428 (+4 989 = +4 999 − 10) ; équipes `ti=9` identiques sur 28 films | `cmp <(cut -f1-9 ...)` ; `paste` des stats ; `diff` des équipes | `plan_verif/ancres_stats_diff.txt`, `ancres_tot_{base,lk}.txt`, `equipes_diff.txt` (vide) |
| R-11 | Marque de portage sous LK : 92 records fermés (f24 7, f27 85), toujours i11 aux décalages 13..16 ; **149** records non fermés (MX : 596 marques / 4 vues = 148 records en i11 + 1 record de `60ae07c4` « après la traversée » ; correction 10) ; « Dynamo Grenade » en plus dans la fenêtre : 80 records A, 57 B (mot lu un bit plus tôt = M41 SPNKr dans 100 % des cas, mesure 1) | `awk '$1=="MF"'`, `'$1=="MX"'`, `'$1=="AF" && $3=="tous" && $5~/enplus_record/'` | `plan_verif/marque_mf.txt` |
| R-12 | Dérive du golden `keyframe_closure` à la tête (contexte golden, base) contre le golden commis : `ti=43` fermés 0 → 34 (`111fa685`), 9 (`11de8353`), 52 (`60ae07c4`), 31 (`a521164d`), 57 (`bcb6d393`), 15 (`e5adf7b2`), 6 (`fb1a1a72`) ; bloquant le plus fréquent changé pour `ti=10`, `ti=43` et `ti=45` sur les 7 bobines ; 21 lignes | `join` de `testdata/keyframe_closure.golden` (commis) et des lignes `golden base` de R-7 | `plan_verif/ratchet/derive_golden.txt` |

**Relectures Ghidra acquises (seconde passe, lecture seule, sorties `REF/g2/`).**
- G-1 Queue d'i0 quand h = 1 (`FUN_14076e3e4`, 14230d04c) : `FUN_1408f0ac4(cat 0)` = R(1) [R(W0) R(2)],
  puis R(1) [R(11)] ; W0 = `ceil(log2 N0)` avec N0 = 0x1DFF ou 0x1FFF (`DAT_144706100` = 0x1FFF,
  image statique ; « ne croît que si `m_gameEngineType` = 1 » était NON VÉRIFIÉ à la rédaction — LU le
  2026-10-08 en LK.1.10 : la croissance exige les drapeaux +0x138/+0x158 de la table d'objets, posés à
  `(DAT_145121140 == 1)`, c'est-à-dire `m_gameEngineType` = 1, WALK_PORT_NOTES §6.5 D), donc **W0 = 13, LU**, déjà porté par
  `varWidthBits(0)` (`varwidth.go:86`) et `consume1408f0ac4(br, 0)` (`bit_leaf_readers.go:92`).
  `consumePositionHandleTail` (`components_position_i0.go:266`) lit le handle sur `pd.IndexW` = 1 et
  ajoute deux R(1) (sélection, extension) : écart de 12 bits de handle et 2 bits structurels quand le
  handle est présent.
- G-2 Branche absolue d'i0 sous portée (`FUN_1406cfe44`) : R(1) h, R(96) (`FUN_1411b259c`), queue
  `FUN_14076e3e4(h)`, R(2) seulement si les trois flottants sont finis (`FUN_140492128`) ; un flottant
  non fini fait échouer le lecteur, ce qui coupe la boucle `FUN_142e2c690` : c'est un ARRÊT de la
  marche, pas un compteur.
- G-3 Treize exceptions datées de position passent par la portée chez le jeu (tableau complet dans
  le relevé Ghidra de la seconde passe) ; une seule est conforme en Go (`consumeFlockPosition`).
  Largeurs sous portée LUES pour toutes ; `FUN_140c1e79c` (`ti=38` i18 : R(1) [R(19)] R(8)) et `FUN_1424e268c`
  (`ti=21` i2..i11 : R(2)), reprises de commentaires Go à la rédaction, sont LUES le 2026-10-08 (LK.1.6).
- G-4 L'état par défaut de `ti=9` (`FUN_1410d7540`) et son désignateur (`FUN_140f581e8` →
  `FUN_1407ef804`, R(4)) n'ont aucun lecteur de position : LK ne change pas les équipes (confirmé
  par R-10).
- G-5 i42 (`FUN_1406d01fc`) : param[0] R(3) = identifiant de demande de jeu d'armes (modulo 7),
  param[1] = emplacement DÉSIRÉ en main principale (unité+0x389), param[2] = emplacement désiré en
  seconde main (unité+0x38a, −1 en temps normal). L'emplacement DÉGAINÉ (unité+0x38c) n'est PAS dans
  i42. Largeurs et correspondance des champs LUES ; le nom « désiré » est DÉDUIT des consommateurs
  (`FUN_1409725b8`, `FUN_142c83228`, `FUN_1407f2dc0`). La fenêtre publie le DERNIER R(2) présent
  (param[2] en ambidextrie) comme `DrawnSlot`.
- G-6 La marque de portage (motif 0x00010005) n'est pas un champ : c'est une configuration des
  petits champs de fin d'i11 (sous sa forme par défaut de 42 bits), du bit d'i12, et de la tête du
  R(5) de drapeaux d'i13 `object-maximum-vitalities` (`FUN_1407ee054` → `FUN_1407eef08`), qui vaut
  01111 (drapeaux 0x4 et 0x8) dans les 92 records marqués et 00011 dans 4 303 records fermés par
  défaut ; 0 record i11 par défaut + i13 01111 sans la marque ; 43 records i13 01111 avec i11 non par
  défaut, que la fenêtre ne voit pas (mesure 1). Le SENS des drapeaux 0x4/0x8 n'est PAS relu dans
  Ghidra.

**Adjudication des baisses de LK (mesures 1 et 2).** Les deux baisses relevées par la
contre-vérification et les 10 fermetures `ti=35` perdues (R-10 : −10) sont des **fermetures de
hasard de la base** : `60ae07c4` slot 539 (n(i22) = 0, aucune famille lue en base, largeurs i0 43 /
i15 427 / i20 208 aberrantes ; sous LK n(i22) = 4 et Bandit Evo égal à la fenêtre, puis +33 bits) ;
`50247b26` (illisible dès i22 dans les deux modes, le 4 de la base tombait au taux du hasard, ~1/8 ;
au témoin 23/190) ; les 10 records perdus (`m2/perdus.txt`) : 9 sur 10 ont n(i22) ≠ 4 en base ; **8
d'entre eux passent à 4 sous LK, le 9e (`50247b26` slot 524, bit 192 288) reste à n(i22) = 0 dans les
deux modes** (film illisible dès i22, D-4 ; classe A → C) ; le 10e (`a349fea8` slot 723, format 20) a
n(i22) = 4 dans les deux modes et aucune famille lue (correction 6, relu sur les lignes R de
`m2/dump_{base,portee}` le 2026-10-08 : 50247b26 0→0, 60ae07c4 0→4, a349fea8 541 2→4, 624 5→4, 666
5→4, 711 3→4, 737 0→4, a521164d 0→4, fb1a1a72 6→4, a349fea8 723 4→4).

**Baisse des traversés de `50247b26` (636 → 635, R-1/R-2), ADJUGÉE (correction 6).** C'est le solde
de deux flux, relevés record par record (clé trame:slot:bit des lignes R de `m2/dump_{base,portee}`,
`join` puis `awk` sur la désynchronisation) : 19 records qui traversaient en base s'arrêtent sur
`i58` sous LK, et 18 records arrêtés sur `i58` en base traversent sous LK. Les 37 sont NON FERMÉS
dans les deux modes (preuve 2), 30 d'entre eux en classe C dans les deux modes (n(i22) ≠ 4), dans
un film illisible dès i22 dans les deux modes (D-4). Une traversée non fermée n'y prouve aucune
lecture : le solde −1 est du bruit de lecture, pas une perte. Les fermetures du film montent
(4 → 6). Statut : ADJUGÉES (mesure 1 `baisses`, mesure 2 `carte_trames`, dumps
`m2/dump_{base,portee}`). Elles entrent dans l'historique du ratchet de cuisson (LK.3.8), nommées ;
le golden sans carte `keyframe_closure` ne voit pas LK (R-7) et n'en porte aucune.

## 3. Décisions

### 3.1 Décisions de l'utilisateur (2026-10-08)

Réponse de l'utilisateur, le 2026-10-08, à la liste des décisions recommandées et au coût annoncé
(relayée par le superviseur) : **« Je suis ok »**. Les décisions ci-dessous sont donc DATÉES et
fermes (règle 10 du contrat : on ne les re-décide pas). Les options et les chiffres qui les
fondaient restent écrits sous chacune, pour l'exécution de D1.0.

- **U-0 — Critère écrit des bascules : amendement ACCEPTÉ** (2026-10-08, « Je suis ok » ; correction 7
  de la contre-vérification : ce n'était pas une décision du rédacteur). La portée et la forme d'`i0`
  du jeu se jugent ENSEMBLE, dans le cadre de production (`TestKF7EFullStateLoop`, variante (d+e)) :
  **369/599 = 61,6 %** avec et sans bouchons ; la portée seule rend **8/599** (1,3 %), `i0` seul
  **25/599** (= la référence). Le critère tel qu'écrit (portée seule, mesurée par
  `TestKF35CBaselineScope`) n'est donc PAS tenu : c'est l'amendement qui permet la bascule. Rejoué
  bascules EN PLACE à l'étape 0 (0.5), avant tout retrait. `TestKF35CBaselineScope`, aveugle depuis le
  lot 2.3 (`walkKeyframeBody` sans `PoserContexte` : 0/599 partout), est retiré avec son cadre s'il
  n'a plus d'appelant (règle 7, LK.3.5). La doc des bascules et leur critère
  (`components_movement.go` `keyframeBaselineScope`, `profil_balayage.go:237-249`) passent dans la
  chronique de `grammar.Rev` avec la mesure (LK.3.4, LK.6.1). Le dénominateur 599 (et non 591) est
  un écart nommé (§7 D-3).
- **U-5 — Fusions : (a), UNE seule fusion dans `feat/v75`, à la fin de D1.4** (une recuisson, un
  backfill killsource ; 2026-10-08, « Je suis ok »). **Accord DONNÉ d'avance pour ces trois gestes**
  (message de l'utilisateur du 2026-10-08, « Pour le 3 tu as déjà mon accord », en réponse au
  point « mise en commun, recalcul des rejeux et du fil des morts » de la liste du reste à faire) :
  ils se font à la fin de D1.4 sans redemander, sous les conditions du gate D1.4 (gates verts, CI
  verte, revue adversariale passée), en prévenant les sessions pairs avant la coupure du serveur
  local et à la fusion, et avec un compte rendu à l'utilisateur après. Les montées de révision
  restent une par jalon (LK.6, D1.4).
  *Fondement* : LK et 2.7.d1 font chacun monter `grammar.Rev`, donc `killsource.Rev` change de valeur
  hachée ; l'option (b) (fusion de LK seul à LK.6) coûtait deux recuissons et deux backfills ; LK ne
  change ni les ancres ni les équipes (R-10).
- **U-1 — Règle d'admission d'une valeur d'image-clé : option (b)** (2026-10-08, « Je suis ok ») :
  valeur admise si le record est fermé OU si n(i22) = 4, ET le témoin T après i43 passe, appliqué aux
  deux classes (A et B). **Si T échoue au gate de D1.0 : ARRÊT, l'utilisateur choisit entre (a) et
  (c).** C'est un amendement de la règle de 2.7.b (ADR 0037, D1.4.3).
  *Fondement* : la règle de 2.7.b (trame non prouvée : la marche passe devant l'ancrage) ne se
  transpose pas : n(i22) = 4 ne valide le curseur que jusqu'à i22, et dans les formats 20-21 même un
  record fermé rend des valeurs fausses (R-4 : A f20-21 armes 0/16). Options présentées : (a) record
  fermé seul ET témoin T après i43 : couverture ≤ 5 399 / 10 710 (50,4 %) ; (b) (record fermé OU
  n(i22) = 4) ET témoin T, à A comme à B : couverture ≤ 10 047 / 10 710 (A + B, 93,8 %), au taux exact
  que dira la mesure de T (D1.0) ; (c) (record fermé OU n(i22) = 4), borné aux formats 24, 25 et 27,
  sans témoin : 8 619 records admissibles (A 5 375 + B 3 244), formats 20-21 tout entiers à la
  fenêtre — (c) branche sur une version de format, pas sur une lecture du jeu. Chiffres : f24-27 B
  aussi juste que A champ par champ (armes 97,8 % contre 98,2 %, 100 % contre 100 % à l'emplacement
  hors faux positif Dynamo ; grenades 99,8 / 99,9 ; chargeurs 99,6 / 99,6 ; réserves ~95-96 % des
  deux côtés) ; f20-21 faux après i22 en A et en B ; n(i22) = 4 au hasard ≈ 1/8 (témoin : 629/6 600
  base, 737/6 600 LK).
- **U-2 — Marque de portage : option (a) si la relecture Ghidra de D1.0.4 établit le sens des
  drapeaux 0x4/0x8 du R(5) d'i13, sinon (b)** (2026-10-08, « Je suis ok ») ; les 43 porteurs
  supplémentaires sont listés à l'utilisateur pour vérification AVANT publication.
  *Fondement* : la fenêtre voit 92 records fermés porteurs ; la grammaire peut lire la même
  configuration (i11 par défaut + i12 + tête d'i13 = 01111), ou le seul drapeau d'i13 (0x4 et 0x8),
  qui ajoute 43 records que la fenêtre ne voit pas (G-6). (a) prédicat = drapeaux 0x4 et 0x8 d'i13
  (135 records fermés marqués, la couche des drapeaux peut changer) ; (b) prédicat = la configuration
  exacte de la fenêtre, lue par la grammaire (92 records), sens non affirmé ; (c) (non retenue) la
  fenêtre seule, « devant la lecture », contraire à l'option A du 2026-10-04.
- **U-3 — Les quatre changements de valeur publiée sont ACCEPTÉS** (2026-10-08, « Je suis ok » ;
  validés sur pièces dans `replay-equiv` en D1.4) : « Dynamo Grenade » retirée des dotations
  d'image-clé lues par la grammaire (faux positif de la fenêtre : 80 records A, 57 B, R-11) ; grenade
  sélectionnée lue en base 0 (sous LK : 393 égales, 27 différentes, 3 420 sélections nulles mises à
  part) ; `DrawnSlot` = emplacement DÉSIRÉ en main principale = param[1] d'i42, égal au dégainé hors
  changement en cours (G-5), au lieu du dernier R(2) présent ; rangs de capacité lus hors 16..23
  (3 593 records sous LK) comptés et non publiés au-delà du domaine.
- **U-4 — En découle : param[1] = −1 est publié comme une absence**, et la doc de
  `types.KeyframeInventory.DrawnSlot` (« 2 = aucune arme dégainée », qu'aucun lecteur relu
  n'appuie) est corrigée, sans inventer de valeur (2026-10-08, « Je suis ok »).
- **Acceptées dans le même message (proposées par le superviseur ; corrections 8 et 9 de la
  contre-vérification : elles n'étaient appuyées par aucun message daté)** :
  - **E-2 — LK sur tous les formats**, sans branche de format (le jeu actuel lit tous les films) :
    gains en f20 (13 → 15 fermés, ≠ 4 1 226 → 380) et f21 (4 → 9, ≠ 4 336 → 0) ; aucune baisse non
    adjugée.
  - **E-6 — Dérive préexistante du golden `keyframe_closure` (R-12) figée dans un commit SÉPARÉ**
    (LK.2.2), avant LK, avec sa ligne d'historique qui l'attribue à son origine (LK.2.1).
  - **D1.2.3 — Date et critère de retrait des nouveaux replis** : `CibleRetrait` = 2026-12-31 ; critère :
    part des records bipèdes d'image-clé non admis ≤ 5 % sur les 28 films ET sur le parc à la
    recuisson.

### 3.2 Décisions d'exécution (à l'exécutant, justifiées par écrit) — une objection de l'utilisateur les rouvre
E-1 est devenue la décision U-0 de l'utilisateur ; E-2 et E-6 ont été acceptées par lui (§3.1).
Restent à l'exécutant : E-3, E-4 (après la relecture Ghidra de LK.1.10), E-5 (recompté), E-7, E-8,
E-9.

- **E-3 — `GrammaireEcrivainI0` borné à la portée.** En delta : +2 trames saines au mieux, 0 perdue
  (R-9) ; l'étendre changerait la grammaire delta pour un gain marginal (hors périmètre, D-2).
- **E-4 — Aucune largeur utilisée non lue n'est codée.** W0 = 13 est LU (G-1) pour l'image statique
  et réutilisé depuis `varWidthBits(0)`. Le cas où la table d'objets grandit (W0 ≠ 13) n'est pas
  présumé : l'arrêt nommé `largeur_handle_moteur_un` (la marche s'arrête au lieu de présumer 13) n'est
  codé en LK.3.7 qu'APRÈS la relecture Ghidra de LK.1.10 (correction 9) — le lien entre la croissance
  de `DAT_144706100` et `m_gameEngineType` = 1 n'est appuyé par aucun fichier de `g2/`. Si la relecture
  l'établit, la garde porte sur ce prédicat (lu dans le film) ; si elle établit un autre prédicat
  observable, la garde porte sur celui-là ; si elle ne permet d'en écrire aucun, LK.3.7 est statué `[!]`
  et l'exécution s'ARRÊTE (décision de doctrine : présumer 13 ou ne pas lire). Les deux largeurs NON
  LUES de G-3 (`FUN_140c1e79c`, `FUN_1424e268c`) ne sont codées qu'après relecture (LK.1.6, LK.5).
  *Relu le 2026-10-08 (LK.1.10, WALK_PORT_NOTES §6.5 D)* : la table ne grandit, donc W0 ne diffère de
  13, que si `DAT_145121140 == 1`, c'est-à-dire si `m_gameEngineType` vaut 1 ; une variante absente
  du film (`Presente` faux) vaut `FUN_14051a4b8(type) == 0`, donc pas de croissance. La garde de LK.3.7
  porte donc sur ce que le film déclare : type de moteur 1, ou type NON ÉTABLI (identité ou variante
  non lue), arrêt nommé ; les deux largeurs de G-3 sont LUES (LK.1.6).
- **E-5 — Gate de non-régression de LK = ratchet en contexte de cuisson** (R-7), pas le golden des
  bobines sans carte (où LK est invisible). RECOMPTÉ (correction 3). Le contexte de carte d'une
  cuisson — les largeurs de la carte posées sur le profil, puis le découpage MPP que la grammaire
  résout — a déjà DEUX copies de production : la cuisson et le sync (`replay.poserProfilPuisCarte` :
  `installWorldObjectPrecision`, `world_object_precision.go:55`, puis `poserLeDecoupageMPPDuFilm`,
  `build_from_film.go:180`) et killsource (`ProfilDeDepartPourCarte`, `decode.go:161`, puis
  `poserLeProfil`, `decode.go:378-382`) ; une vingtaine de tests et d'instruments de `grammar` (et deux
  de `replay`) appellent en outre `PoserLargeursObjetDuMondeDepuisDecoupage`. Le ratchet serait la 3e
  copie : la règle 6 impose le helper.
  **Choix : un helper unique de la grammaire pour le contexte de carte de la cuisson, que `replay`
  (cuisson et sync) et le ratchet appellent, avec un garde-rail RESTREINT AU CODE DE PRODUCTION
  (fichiers non `_test.go` du module) et une allowlist datée** : la définition de la pose
  (`profil_balayage.go`), la méthode du contexte que le helper appelle et l'enveloppe D2
  `contexteDeBobine` (`film_context.go` : découpage DÉTECTÉ dans le film, jamais le catalogue), et
  `facts/killsource/decode.go` (exception datée du 2026-10-08).
  **Pourquoi killsource n'entre pas dans le helper** : killsource ne pose pas la carte sur un CONTEXTE
  mais sur son PROFIL DE DÉPART, avant la calibration (`ProfilDeDepartPourCarte`), puis pose le profil
  calibré et le découpage (`poserLeProfil`) ; le faire passer par le helper changerait l'ordre de ses
  gestes et son périmètre haché (`killsource.Rev`) dans un lot dont killsource n'est pas l'objet, et
  son gate (`cmd/killsource json` sur les 19 témoins) jugerait alors deux changements à la fois.
  Les tests et instruments restent hors du garde-rail : la plupart posent un découpage détecté pour
  une mesure datée, et aucun n'est un chemin de production. Les ancres littérales du registre des
  replis (`registre_filmdec.go:158-161`, gardées par `TestToutSiteDuRegistreExiste`) suivent le helper.
- **E-7 — Montées de révision, une par jalon** (LK.6, D1.4), selon la pratique de 2.7.d (commits
  intermédiaires sans montée, gardes de révision passées sous `-skip` en local — commande `SKIPREV`
  du §1.4 — et jamais de push avant la montée). Chaque montée de `grammar.Rev` porte : sa chronique
  avec les mesures ; la décision du gate objectives (porte `LEVELUP_UPDATE_OBJECTIVES_REV=1` du §1.4,
  qui régénère `testdata/objectives_rev.golden` — révision ET empreinte — à révision constante, avec
  un COMPLÉMENT daté dans `facts/objectives/rev.go`, si l'étape objectives de `replay-equiv` est
  identique ; sinon montée d'`objectives.Rev` ; correction 2 : ce n'est pas `objectives_perimetre`,
  qui fige la fermeture des imports) ; `killsource_rev.golden` régénéré par sa porte
  (`LEVELUP_UPDATE_KILLSOURCE_REV=1`, son empreinte hache la VALEUR de `grammar.Rev`), avec un
  complément à révision constante si `cmd/killsource json` est identique sur les 19 témoins, sinon
  montée de `killsource.Rev`.
- **E-8 — Pas d'élargissement de la forme des faits** (emplacement, variante, méthode de
  récupération) : `SchemaDesFaits` ne monte que si le test de survie de D1.3 l'exige.
- **E-9 — Instruction du résidu sans correction** : le +33 bits de `60ae07c4`, l'illisibilité de
  `50247b26` en i22, les formats 20-21 après i22, les traversées `ti=40` (6 183), `ti=21` (1 750,
  +76/+77 bits) et `ti=44` (5) modifiées sans fermeture, l'écart de réserve ~5 %, le témoin +1 bit
  absorbé sur 223 records : nommés au §7, instruits seulement là où une étape l'exige (réserves en
  D1.0).

## 4. Étapes

Une étape commence quand la précédente est close : items statués, gate passé et consigné au journal
(§8). « Clos » = ces trois conditions. Commits préfixés `ri-lk(<étape>): …` (`ri-lk(0)`,
`ri-lk(LK.3)`, …) puis `ri-d1(<étape>): …`, chemins explicites (jamais `git add -A`), pied
`Co-Authored-By`. Le GO de l'utilisateur (2026-10-08) vaut accord de commit et de push de la branche,
pas de fusion ; AUCUN push avant la montée de révision du jalon (E-7 : les gardes de révision
rougissent entre deux montées).

### Étape 0 — Worktree, instruments, références rejouées à la tête (aucun code de production)
- [x] 0.1 Branche `feat/ri-lk-images-cles` (créée par le superviseur depuis `feat/ri-etape2`
      `81d4831d6`) dans le worktree réutilisé `LevelUp-wt-ri` (§1.1) : branche, propreté de l'arbre
      et trois jonctions vérifiées ; `data/cache/film_facts` absent ou dossier réel.
- [x] 0.2 Copier ce plan dans `.ai/PLAN_RI_ETAT_COMPLET_IMAGES_CLES_2026-10-08.md` du worktree, avec
      les corrections de la contre-vérification et les décisions de l'utilisateur ; dans le plan de
      l'étape 2, l'item 2.7.d1 renvoie à ce fichier.
- [x] 0.3 Patch relu des instruments (§1.2) : `git -C LevelUp-wt-imagecle diff` et copie des
      fichiers non suivis, crochet et variables d'environnement retirés, mode unique et noms de
      sortie du §1.2 ; relecture ligne à ligne consignée au journal ; `go vet -tags=research $G` vert.
- [x] 0.4 Références à la tête, dans le worktree LK (code de production inchangé) :
      I-d0 → `$S/e0` ; I-ferm → `$S/e0` ; I-ancres, I-equipes → `$S/e0` ; I-carte → `$S/e0`.
- [x] 0.5 Critère écrit rejoué dans le worktree LK, bascules EN PLACE (I-critere, AVEC
      `TestKF35CBaselineScope`, l'instrument que la doc nomme ; correction 7) → `$S/e0/critere*.log`.
- [!] 0.6 Coût, mesure de la base (correction 13). Cuisson : 4 films, `084a804d`, `e5adf7b2`,
      `60ae07c4`, `11de8353` (ceux de la mesure M.1 de l'étape 2). Lecture des porteurs au sync :
      les trois premiers films à drapeau et le premier film à bombe de `$FILMS` (ordre de la liste)
      qui ont des faits d'équivalence versionnés (`replay/testdata/equivalence/<id>.facts.json`) —
      la garde de mode réelle est `replay.GardesDeLaVariante(gameVariantName)` (`porteurs_entrees.go` ;
      « ModeAPorteur » n'existe pas dans le code), et ses deux familles qui lisent des images-clés
      sont `Drapeau` (équipes, objets du monde, marques) et `Bombe` (canal des armes tenues, que D1.3
      branche sur la lecture) : `084a804d` (BTB Heavies:CTF), `1c4c63c2` (BTB:One Flag CTF),
      `51101d1d` (CTF:Arena Neutral Flag), `9f57c612` (Assault:One Bomb). Banc
      `BenchmarkPorteursAuSync` dans le paquet `internal/sync/killcollector` (fichier
      `porteurs_au_sync_bench_research_test.go`, tag `research` : il lit le cache de films local,
      absent de la CI). Paquet du collecteur et non `replay` (décision d'exécution, 2026-10-08) :
      l'entrée de `PortagesAuSync` s'y rebâtit par les COUTURES de production du collecteur
      (`decoderLeFilm`, `lireLePontDuCollecteur`, `entreeDuRegistre`, `entreeDesPorteurs`,
      `cataloguesDuPlacement`) au lieu de les recopier (règle 6), et `replay` ne peut pas importer
      `replayidentity` (cycle). Variables : `PORTEURS_BENCH_FILMS` (ids courts, virgules) et
      `PORTEURS_BENCH_CACHE` (le `data/cache` qui contient `film_chunks/`), saut sans elles ; le
      roster, la feuille, les équipes, la variante et la carte viennent des faits d'équivalence (le
      banc n'ouvre aucune base). La boucle `b.Loop` ne chronomètre que `PortagesAuSync`, sur un
      contexte neuf à chaque tour (étage du pont rejoué chronomètre arrêté) ; métriques `ns/op`,
      `pic-Mio` (empreinte `filmproc.Footprint` échantillonnée à 10 ms), `portages`, `B/op`.
      Mesurer la base : G-perf (§5) → `$S/e0/perf_base.txt`.
      *Statut `[!]` (2026-10-08, ressource externe indisponible)* : banc écrit, vet `research` vert,
      joué sur les quatre films (il saute sans variables) ; binaires de base gardés
      (`$S/e0/bench_base.test`, `$S/e0/replay-equiv_base.exe`) ; cuisson de base jouée une fois
      (4/4 identiques aux références d'équivalence). Mais AUCUNE mesure n'a pu être prise sur machine
      calme : de 19:02 à 19:34, d'autres sessions ont fait tourner leurs tests Go en continu (2 à 13
      processus `go`/`*.test` étrangers : tests `-tags=integration` de sync, duckdb, ops, archlint,
      sonde `i59` de la grammaire) ; le script qui n'accepte qu'une mesure sans processus étranger
      avant ET après n'en a obtenu aucune en 20 minutes. Les mesures prises sont publiées, marquées
      perturbées (`perf_base.txt`). Report valide : la machine calme est une ressource externe, et
      LK.1 à LK.6 n'en dépendent pas ; la mesure de base se refait EN PAIRES avec le lot au gate de
      coût de D1.1 (D1.1.7), qui en avait besoin de toute façon.
- Gate 0 :
  - `diff <(sort $REF/ref_corrigee/images_cles_base.tsv) <(sort $S/e0/images_cles.tsv)` vide
    (instrument repris = instrument mesuré) ; `agg.tsv` TOTAL = R-1 ;
  - I-ferm : `--- PASS` (pas `--- SKIP`) et lignes `base` de `$REF/fermeture_corpus.tsv` = sortie I-ferm
    (colonne d'étiquette retirée, §1.4) ;
  - `m2_ancres_tete.tsv`, `m2_stats_tete.tsv` identiques à `$REF/m2/ancres/m2_ancres_base.tsv`,
    `m2_stats_base.tsv` ; lignes `tete` de `m2_equipes.tsv` = lignes `base` de `$REF/m2/ancres/m2_equipes.tsv`
    (colonne 1 retirée) ;
  - cartes : `cmp` contre `$REF/m2/carte_v2_base`, `carte_v2_base8`, `carte_v2_mpp_base` (films :
    colonnes 1-16) ;
  - critère : (d+e) = 369/599 avec ET sans bouchons, (d) seule 8/599, REF 25/599,
    `TestKF35CBaselineScope` 0/599. **Si (d+e) < 50 % : ARRÊT, décision de l'utilisateur** (ne rien
    retirer) ;
  - toute différence avec `REF` est expliquée (fusion de `feat/v75` entre-temps) avant LK.

### Étape LK.1 — Préalables lus, consignés avant tout code
- [x] LK.1.1 Critère écrit rejoué bascules en place : R-8 (et 0.5 à la tête).
- [x] LK.1.2 Élection des ancres, preuves, équipes mesurées sous LK : R-10 ; carte delta : R-9.
- [x] LK.1.3 Queue d'i0 h = 1, W0 = 13 : G-1 ; forme absolue sous portée : G-2.
- [x] LK.1.4 Équipes : G-4.
- [x] LK.1.5 Baisses adjugées (§2, « Adjudication »).
- [x] LK.1.6 Relire dans Ghidra les deux largeurs NON LUES de G-3 : `FUN_140c1e79c` (`ti=38` i18, par
      bit du masque de `FUN_142f036f0`) et `FUN_1424e268c` (`ti=21`). Sortie : adresses, largeurs,
      statut LU / NON LU.
- [x] LK.1.7 Faire correspondre chaque appelant lecteur de `FUN_14076f91c` (`FUN_1406cfe44`,
      `FUN_14076e494`, `FUN_14076e4ec`, `FUN_1408f02c8`, `FUN_140ee7270`,
      `FUN_140f04d88/f18/f68/fb8`, `FUN_140fb8af0`, `FUN_14076f75c`, `FUN_1408096f8`,
      `FUN_1410f03b4`, `FUN_14112134c`) et les 14 lecteurs en ligne `141dc8600..141ddb460` à leur
      lecteur Go, avec : consulte la garde (`fullPrecisionGate` / `lireE494*` / `lireE420`) oui ou non.
      Recontrôler les CALL des sites non exceptés (navpoint, spawn-filter, selectable-zone,
      asset-transform, i0 prédit).
- [x] LK.1.8 Relire le lien entre la garde low-frequency de `ti=3` (`components_frequences.go:62-66`,
      `ecs_table.tsv:76` « partiel ») et `FUN_14076f91c`.
- [x] LK.1.9 Consigner le tout dans `.ai/V7.5/killweapon/WALK_PORT_NOTES.md` §6 (sous-section LK) :
      un tableau composant/exception, adresse, lecture sous portée, largeur, statut.
- [x] LK.1.10 Relire dans Ghidra, AVANT de coder l'arrêt de E-4 (LK.3.7 ; correction 9), le lien
      entre la croissance de `DAT_144706100` (écrivains `FUN_142f2f0cc` @142f2f2b1 et `FUN_1408f1618`
      @1423503d3, octets +0x138/+0x158 de la table d'objets) et le type de moteur : qui pose ces
      octets (`FUN_140d10a78`, sous quelle condition), et ce qui pose `DAT_145121140`
      (`FUN_14051a4b8`, lien à `m_gameEngineType`). Sortie : prédicat établi (LU) ou non (NON LU),
      adresses ; la forme de la garde de LK.3.7 en découle (E-4).
- Commandes Ghidra : `curl -s http://127.0.0.1:8089/<point>` avec `decompile_function`,
  `disassemble_function`, `get_xrefs_to`, `get_xrefs_from`, `read_memory` uniquement (jamais un point
  qui écrit). Sorties sous `$S/lk1/`.
- Gate LK.1 : tableau complet, aucune case vide ; toute largeur restée NON LUE est marquée « non
  codée » et son site reste hors portée (sa lecture actuelle), nommé au §7 ; LK.1.10 statué.

### Étape LK.2 — Dérive du golden et ratchet de cuisson, à la base (deux commits séparés)
- [x] LK.2.1 Attribuer la dérive R-12 au lot qui l'a produite : `git log --oneline -- $G/testdata/keyframe_closure.golden`
      (dernier `-update`), puis `git log` des lecteurs `ti=43` depuis ce commit ; bisection sur les 7
      bobines si besoin (`go test -run KeyframeClosureRatchet $G`, ~2 s par passe).
- [x] LK.2.2 Commit 1 `ri-lk(LK.2): golden keyframe_closure, derive de la tete figee` :
      `go test $G -run KeyframeClosureRatchet -update-keyframe-closure`, ligne d'historique datée qui
      nomme les 7 hausses `ti=43` (0 → 34, 9, 52, 31, 57, 15, 6), les bloquants `ti=10/43/45` changés
      et le lot d'origine (LK.2.1) ; aucune autre ligne ne doit changer.
- [x] LK.2.3 Helper unique de contexte de carte (E-5, recompté) : une fonction exportée de la
      grammaire qui pose les largeurs de la carte du profil du film sur le profil de balayage
      (`PoserLargeursObjetDuMondeDepuisDecoupage`) et le découpage MPP (`ResolutionMPP` +
      `PoserMPP`), et rend ce qu'elle a fait pour que l'appelant journalise et compte comme avant ;
      `replay.poserProfilPuisCarte` (cuisson et sync) l'appelle (comportement inchangé, journaux et
      comptes de replis compris) ; ancres du registre des replis (`registre_filmdec.go:158-161`)
      mises à jour ; garde-rail (test grep dans `archlint`) RESTREINT AU CODE DE PRODUCTION : hors des
      fichiers de l'allowlist datée du 2026-10-08 (définition `profil_balayage.go`, méthode du
      contexte et enveloppe D2 `film_context.go`, le helper, `facts/killsource/decode.go`, chacun avec
      sa raison), aucun fichier non `_test.go` du module n'appelle
      `PoserLargeursObjetDuMondeDepuisDecoupage` ; mutation : un appel ajouté dans un fichier de
      `replay` le fait rougir.
- [x] LK.2.4 Commit 2 `ri-lk(LK.2): ratchet de fermeture d image-cle en contexte de cuisson` :
      `grammar/keyframe_closure_cuisson_ratchet_test.go` + `testdata/keyframe_closure_cuisson.golden`.
      Contexte : `NewFilmContextForMap(film, &entree, nil)` puis le helper LK.2.3 ; carte tirée du
      catalogue commis `data/titles/halo_infinite/reference/map_quant_bounds.json` (déjà lu en CI par
      `i0_catalogue_mutation_test.go`) ; nom de carte tiré des faits d'équivalence versionnés
      (`FragmentationHeavies`, `LiveFire-Ranked`, `Thunderhead`, `Command`, `Fragmentation`,
      `Cliffhanger`, `BanishedNarrows`) ; comparateur `comparerFermeture` (rouge sur baisse ou ligne
      disparue) ; drapeau `-update-keyframe-closure-cuisson` ; décodage mémoïsé comme `fermetureMemo` ;
      historique de création. Le helper LK.2.3 entre dans le même commit que son garde-rail.
- Tests (gardes de révision sous `-skip`, correction 4 : le helper est une source non-test de
  `grammar`, donc `TestGrammarRevSuitLaGrammaire` rougit jusqu'à LK.6) :
  `go test $G -count=1 -run 'KeyframeClosure'` ; `go test ./internal/archlint/ -count=1` ;
  `go test ./internal/games/halo_infinite/film/replay/ -count=1` (helper : aucun golden ne bouge) ;
  `go test ./internal/games/halo_infinite/film/internal/facts/... -count=1 -skip "$SKIPREV"` (ancres du
  registre).
- Gate LK.2 :
  - le golden de cuisson à la base = lignes `cuisson base` de `$REF/plan_verif/ratchet/m2_bobines.tsv`
    (215 lignes, `ti=35` 53/1 368) ;
  - mutation jouée : poser la portée sur toute la boucle par un crochet local de test → le ratchet
    signale la hausse (53 → 666) ; retirer la pose des largeurs de carte → `ti=37/38/42/43` changent
    (rouge) ;
  - `replay-equiv` inutile ici si `go test ./internal/games/halo_infinite/film/... -count=1 -skip "$SKIPREV"`
    est vert (le helper ne change aucune valeur) ; sinon, instruire avant de continuer.

### Étape LK.3 — La portée en production, i0 absolu sous portée, retrait des bascules
- [x] LK.3.1 Champ `Lecteur.portee` (miroir de `DAT_144e61ea0`, doc = contrat) ;
      `fullPrecisionGate` = `br.portee || br.fullPrecision()`.
      *Fait* : `lecteur.go` (champ et contrat : qui la pose, où, ce qu'elle change, qui ne la pose
      jamais) ; `components_movement.go` (garde et doc réécrites).
- [x] LK.3.2 `consumeFullStateDefaultBlock` pose la portée avant l'état par défaut et la retire après
      le mot de contrôle (n2 hors portée) ; `walkKeyframeFullState` la pose sur toute la boucle et la
      remet à faux sur chaque sortie (y compris n2 = 0 et échec de lecteur) ; elle n'est posée que si
      n1 > 0 pour l'état par défaut et n2 > 0 pour la boucle, comme le jeu.
      *Fait* : branche `n1 > 0` de `consumeFullStateDefaultBlock` (écritures 142e2c46f / 142e2c530,
      relues dans Ghidra) ; `traverserSousLaPortee` (142e2c6b8 / 142e2c76a) autour de
      `traverseComponentLoop`, remise à faux au retour, arrêt de lecteur compris ; `n2 = 0` ne lance
      pas la boucle, donc ne pose rien.
- [x] LK.3.3 Branche absolue d'i0 sous portée = forme du jeu (G-2) : R(1) h, R(96), queue, R(2)
      conditionné à la finitude ; un flottant non fini ARRÊTE la marche (cause nommée
      `position_non_finie`). À ce pas la queue reste `consumePositionHandleTail` (reproduire
      exactement le crochet mesuré) ; son port fidèle est LK.5.2.
      *Fait* : `consumeAbsoluSousLaPortee` (`components_position_i0.go`) ; finitude = `FUN_140492128`
      sur l'image mémoire (`bits.ReverseBytes32` du mot lu : `FUN_1406d676c` range les octets dans
      l'ordre du flux), exposant `0x7f800000` ; échec = `grammar.ArretDuLecteur`
      (`ArretPositionNonFinie`), porté par `EntityTrace.Arret` ; la boucle de composants s'arrête au
      début du composant (`DesyncAt`, curseur remis au début). Reproduction du crochet PROUVÉE : les
      deux arrêts neutralisés le temps d'une mesure, I-d0 et I-ancres rendent la référence du crochet
      à l'octet (journal LK.3).
- [x] LK.3.4 Retrait de `PorteeBaseline` et `GrammaireEcrivainI0` du profil (kill-switches retirés à
      la bascule) et de leur doc ; doc du critère et sa mesure (R-8) déplacées dans la chronique de
      `grammar.Rev` (LK.6) et dans WALK_PORT_NOTES ; doc de `components_movement.go`
      (`keyframeBaselineScope`, D-REV-1) réécrite (règle 17).
      *Fait* : champs et doc retirés de `profil_balayage.go` (plus aucun lecteur dans le module ; seules
      les graines de fuzz de `replay` les nomment encore, et restent valides : la charge des morts
      d'objet est du JSON relu sans exiger ses clés — `TestCodecCouvreFilmInputs` et
      `FuzzDecodeFilmFactsFile` verts) ; critère et mesure : WALK_PORT_NOTES §6.5 E (la chronique de
      `grammar.Rev` suit à LK.6.1) ; `components_movement.go` : blocs `keyframeBaselineScope` et
      `keyframeWriterI0Grammar` retirés, `fullPrecision` / `fullPrecisionGate` réécrits ;
      `components_frequences.go` et le commentaire de `consumeAbsoluteWithGate`, qui citaient les
      bascules, corrigés.
- [x] LK.3.5 Tests qui posaient les bascules : `components_arrets_vue_b_test.go`,
      `lecteur_position_sites_test.go`, `keyframe_fullstate_loop_test.go` (`kf7eCases` : (d+e) devient
      REF), `r_veh_ti40_research_test.go`, `r_veh_imagecle_research_test.go` → posent `br.portee` par
      le seul assistant de test, ou sont retirés quand leur instrument est périmé ;
      `keyframe_baseline_scope_test.go` et `walkKeyframeBody` retirés s'ils n'ont plus d'appelant
      (règle 7) ; instruments repris (§1.2) relus pour qu'aucun ne pose plus une bascule disparue.
      En particulier, la relecture à l'étendue de l'instrument d0 (`ri27d0`, fonction `lire`) posait
      les bascules sur sa copie du profil en mode portée : elle pose désormais `br.portee` par
      l'assistant de test, sans quoi elle relirait quantifié et le gate 3 ci-dessous ne pourrait pas
      reproduire `images_cles_portee.tsv` (découverte 36).
      *Fait* : assistant unique `sousLaPortee` (`harnais_portee_test.go`) ; marqueur ti=12 et
      translocateur par lui ; `TestLaGrammaireDeLEcrivainI0LitLaTableAPrecHaut` RETIRÉ (la grammaire
      d'écrivain hors portée n'existe plus, E-3) et remplacé par les tests de
      `portee_etat_complet_test.go` ; `kf7eCases` = REF (l'ancienne (d+e)) et (c) ; `TestKF7EProfileI0`
      retiré (A/B d'une bascule disparue) ; `keyframe_baseline_scope_test.go` SUPPRIMÉ (instrument
      aveugle, U-0) ; `walkKeyframeBody` CONSERVÉ (quatre instruments l'appellent encore) ; `ri27d0`
      `lire` par `sousLaPortee` (fichier toujours à 500 lignes) ; `ri27d1_m2_critere` réduit à REF ;
      table de `lecteur_position_ratchet_test.go` : le site de la bascule (`lireE420` depuis
      `consumeObjectPositionDynamicPrecisionD`) retiré, le jeu n'y appelle pas `FUN_14076e420` ;
      instruments `campagne_overlay` `r_veh_ti40`, `r_veh_ti40_variantes`, `r_veh_imagecle` : variantes
      de portée et d'i0 retirées (elles sont la production), relectures d'état complet sous
      `sousLaPortee` — non compilables sans leur surcouche, absente du dépôt (§7, D-15).
- [x] LK.3.6 Ratchet d'écriture (archlint ou grammar) : `portee` n'est écrit que dans
      `keyframe_fullstate_loop.go` (et, en D1.1, dans l'assistant de relecture à l'étendue) ;
      `TraverseEntity`, `decodeDelta`, la vue A (`vue_a_charges*.go`) ne la posent jamais.
      *Fait* : `grammar/portee_ecriture_guard_test.go` (AST de tous les fichiers du paquet, tests et
      instruments compris ; compte exact : `keyframe_fullstate_loop.go` 4, `harnais_portee_test.go`
      1). Mutation « portée posée dans `TraverseEntity` » : ROUGE (ce ratchet et
      `TestLeRecordNeufEtLeDeltaNePosentPasLaPortee`), retirée.
- [x] LK.3.7 Garde E-4 : sous la portée, h = 1 et le prédicat de croissance de la table d'objets
      établi en LK.1.10 (attendu : `TypeDeMoteur` = 1) → arrêt nommé `largeur_handle_moteur_un` ; test
      unitaire. Si LK.1.10 n'établit aucun prédicat lisible dans le film : `[!]` et ARRÊT (E-4).
      *Fait* : `GrammaireBalayage.MoteurUnPossible`, dérivé du film comme le contrôle de corruption
      (`grammaireSousFilm` : identité non lue, variante non lue, ou type 1 déclaré → vrai ; variante
      absente → type 0 → faux ; défaut de structure faux = valeur du jeu sans variante), posé à chaque
      rendu par `FilmContext.poserLaGrammaireDuFilm` (profil du contexte et preuve d'image-clé) ;
      arrêt `ArretLargeurHandleMoteurUn` avant la queue. Tests : `TestLaGardeDuHandleArreteSousLeMoteurUn`,
      `TestLeTypeDeMoteurVientDuFilm`, cas « moteur 1 possible » de `TestLaMarcheDEtatCompletPoseLaPortee`.
- [x] LK.3.8 Régénérer `keyframe_closure_cuisson.golden` (`-update-keyframe-closure-cuisson`) avec
      sa ligne d'historique : `ti=35` 53 → 666 / 1 368 ; baisse `60ae07c4` 2 → 1 = slot 539, bit
      200 424, fermeture de hasard de la base (adjugée, §2) ; le golden `keyframe_closure` (sans carte)
      ne doit PAS bouger.
      *Fait* : 666/1 368 exactement, seules les sept lignes `ti=35` changent (fermés, et bloquant
      vidé sur quatre bobines) ; historique dans le générateur ; golden sans carte inchangé
      (`-run KeyframeClosure` vert sans régénération).
- Tests unitaires : i0 sous portée h = 0 (1 + 1 + 1 + 96 + 2 = 101 bits) et h = 1 ; flottant non fini
  → arrêt ; trame média de l'état par défaut bipède sous portée = 96 bits ; la vue A et un paquet
  delta lisent à l'identique (portée jamais posée). Mutations (chacune doit rougir un test) : poser
  la portée dans `TraverseEntity` ; oublier la remise à faux ; lire R(2) avant la queue ; couvrir n2
  sans couvrir l'état par défaut.
- Commandes : `go test $G -count=1 -skip "$SKIPREV"` (gardes de révision au jalon, E-7 ; §1.4) ;
  `go test ./internal/archlint/ -count=1` ; `go vet -tags=research $G`.
- Gate LK.3 (dans cet ordre : élection AVANT carte) :
  1. I-ancres / I-equipes → `$S/lk3` : colonnes 1-9 de `m2_ancres_tete.tsv` et les totaux autres que
     `fermes` identiques à `$S/e0` ; fichiers entiers identiques à `$REF/m2/ancres/m2_ancres_lk.tsv`,
     `m2_stats_lk.tsv` ; équipes identiques (lignes `lk` de la référence, colonne 1 retirée).
  2. I-carte → `$S/lk3` : identique à `$S/e0` (colonnes 1-16 pour `fermeture_films.tsv`) sur 20, 8 et
     28 films (MPP). Toute différence est adjugée par les ancres (étape 1 de ce gate) avant de
     continuer.
  3. I-d0 → `$S/lk3` : `diff <(sort $REF/ref_corrigee/images_cles_portee.tsv) <(sort $S/lk3/images_cles.tsv)`
     vide ; `agg.tsv` TOTAL = R-2.
  4. I-ferm → `$S/lk3` (`--- PASS`, pas `--- SKIP`) : lignes `portee` de `$REF/fermeture_corpus.tsv`
     reproduites (colonne 1 retirée) ; `ti=35` 5 399 ; aucun autre archétype ne change.
  5. I-critere : REF = 369/599 avec et sans bouchons (`TestKF7EFullStateLoop`, `TestRI27d1M2Critere`).
  6. `go test $G -run 'KeyframeClosure' -count=1` vert (golden sans carte inchangé, cuisson à 666).
  Tout écart avec la mesure du crochet est expliqué avant LK.4.

### Étape LK.4 — Instruction du résidu sous LK (sans correction)
- [x] LK.4.1 Publier, depuis I-d0 de `$S/lk3`, le résidu chiffré : 88 arrêts (55 i59, 33 i58) ;
      bloquant vide sur 21 films, arrêt sur 7 (R-6) ; écarts de fin R-5.
      *Fait (sous la production de LK.3, sorties `$S/lk4/`)* : **135 arrêts** sur 10 710 records
      bipèdes — 55 sur i59 (`0797ce72` 1, `084a804d` 1, `111fa685` 2, `60ae07c4` 2, `a349fea8` 34,
      `a521164d` 15 ; inchangé), 28 sur i58 et 52 au début d'i0, tous sur `50247b26` (51
      `largeur_handle_moteur_un`, 1 `position_non_finie`) ; les 88 du crochet deviennent 135 par la
      seule garde E-4 et l'arrêt de finitude sur ce film sans section d'identification (5 de ses 33
      arrêts i58 et 47 de ses traversées s'arrêtent désormais sur i0 ; journal LK.3). Bloquant `ti=35`
      (I-ferm) vide sur 21 films, arrêt sur 7 : i59 sur `0797ce72`, `084a804d`, `111fa685`,
      `60ae07c4`, `a349fea8`, `a521164d` ; i0 sur `50247b26` (i58 sous le crochet, R-6). Écarts de
      fin : 0 pour 5 328, multiple de −108 pour 2 801, autre pour 2 223 dont 2 099 sur 4 films
      (`a349fea8` 967, `50247b26` 414, `a521164d` 387, `60ae07c4` 331) — R-5 à un record près, passé
      de « autre » à « −108 » sur `50247b26`. Commandes : `awk` des lignes `S` `desync_*` de
      `$S/lk3/images_cles.tsv` (`$S/lk4/desync_lk3.txt`) ; colonne 6 des lignes `ti=35` de
      `$S/lk3/fermeture_corpus.tsv` (`$S/lk4/bloquant_ti35_lk3.txt`) ; `awk -f $S/lk4/ecart_fin.awk`
      (règle vérifiée : rend R-5 à l'identique sur `images_cles_portee.tsv`) → `$S/lk4/ecart_fin_lk3.txt`.
- [x] LK.4.2 `60ae07c4` +33 bits : premier composant de largeur divergente (dump R, largeurs modales
      comparées à `084a804d` et `111fa685` : `sonde_f24/modal.txt`) ; candidats nommés i57 (28 bits
      dans 250/331 records contre 2) et i53 (36 contre 30) ; conclusion écrite (adjugé ou nommé).
      *Fait — NOMMÉ, non adjugé.* Sonde rejouée sous LK.3 (`RI27D1_RECORDS=60ae07c4,084a804d,111fa685
      RI27C_FILMS=` les mêmes, I-d0 → `$S/lk4/sonde_f24/`) ; largeurs modales par `$S/lk4/modal.awk`
      (règle vérifiée : rend `sonde_f24/modal.txt` à l'identique sur l'ancien `r.tsv`) : identiques à
      la sonde du crochet. Registre `ti=35` relu sur les trois films (instrument temporaire, non
      commité) : i43..i59 de mêmes noms et niveaux (`60ae07c4` = HI_1_8_0, les deux autres
      HI_1_10_0). Largeurs de `60ae07c4` jamais vues dans les records fermés des deux références :
      i43 (201) et i44 — les armes, de largeur propre à l'arme : Bandit Evo, lue ÉGALE à la fenêtre
      dans 254 records sur 332, l'alignement tient donc au moins jusqu'à l'identité de l'arme d'i43 —,
      puis **i53** `biped-malleable-property-component`
      (`FUN_140ff6764`) 36 bits dans 308/332 records contre 30 dans 953/956 records fermés des
      références, et i57 `biped-spartan-ability-component` 28 contre 2. Écart de fin PAR RECORD
      (frontière = bit du record du slot suivant de la même trame, 223 records appariés) : les 163
      records à +33 ont TOUS i53 = 36 et i57 = 28 (+6 + 26 = +32) ; 46 records à +18 ont i53 = 36,
      i55 = 4, i57 = 2, i58 = 19 (+6 − 1 + 12 = +17). Les deux populations partagent i53 = 36 et un bit
      de plus que la somme des écarts de largeur. Conclusion : le premier composant de largeur
      divergente est **i53** ; i57, i55 et i58 divergent EN AVAL, selon la population ; le bit
      constant restant n'est pas localisé par la comparaison modale (il peut précéder i53, dans un
      composant de largeur dépendante des données). Non adjugé faute d'oracle : la grammaire de
      `FUN_140ff6764` (dont le champ de largeur R(5) puis R(n)) est à relire dans le jeu sur ce build
      avant toute correction (§7, D-16).
- [x] LK.4.3 Formats 20-21 après i22, `50247b26` avant i22, témoin +1 bit absorbé (223/227) : une
      ligne d'état par point au §7 (D-4 à D-6), sans correction.
      *Fait* : une ligne « État sous LK.3 » à D-4, D-5 et D-6, chacune avec ses chiffres et sa
      commande (`taux.sh`, `par_format.sh`, lignes `S` de l'instrument d0).
- Gate LK.4 : le §7 porte un chiffre et une commande pour chaque point ; aucun code.

### Étape LK.5 — Fidélité sous la portée, une correction à la fois, retenue seulement sans baisse
Règle (découvertes retenues une à une) : chaque sous-pas est un commit ; il est rejeté (revert du
commit, noté au §7) si une ligne (archétype × film) baisse dans I-ferm, si `ti=35` baisse dans le
ratchet de cuisson, ou si la carte delta change ; jamais retenu en partie. Hausses chiffrées par
archétype, ratchet régénéré avec une ligne d'historique par sous-pas retenu.
- [ ] LK.5.1 i20 `lireViseeDActeurAncienne` (emplacements a = 0 : `FUN_14058c058`, CALL 1422cddc1 /
      1422cde0e) : sous la portée, R(96) par emplacement au lieu de 16 bits plats ; hors portée,
      l'exception reste.
- [ ] LK.5.2 Port fidèle de `FUN_14076e3e4` pour la seule branche absolue sous portée :
      `consume1408f0ac4(br, 0)`, R(1), [R(11)] (G-1) ; `consumePositionHandleTail` inchangé pour la
      forme en ligne.
- [ ] LK.5.3 `consumePrecHautDuBipede` : vérifier qu'il n'est plus atteint sous la portée (G-3) ;
      test de non-atteinte.
- [ ] LK.5.4 Exceptions d'objets sous la portée, une par sous-pas, dans cet ordre, chacune sur sa
      lecture LUE (G-3) : `consumeObjectPositionMonde` (`ti=36..43` i0) ;
      `consumePlayerDesiredRespawnLocation` (`ti=5` i12) ; `consumeCrewOrder` (`ti=14` i0) ;
      `consumeTacmapPoiIcon` (`ti=30` i0) ; `consumeTacmapAreaOfInterest` (`ti=32` i0) ;
      `consumeTacmapDisplayAsset` (`ti=33` i0) ; `consumeTacmapCoopTetherArea` (`ti=34` i11) ;
      `consumeTacmapWaypointState` (`ti=34` i7) ; `consumeGenericRigidBodyTransforms` (`ti=38` i18) et
      `consumeFlockDestination` (`ti=21` i2..i11) SEULEMENT si LK.1.6 a lu leurs largeurs, sinon `[!]`.
- [ ] LK.5.5 Garde low-frequency de `ti=3` levée sous la portée SEULEMENT si LK.1.8 a établi le lien
      (sinon `[!]`, D-7) ; `ecs_table.tsv:76` « partiel » → « porte » avec elle.
- Tests par sous-pas : largeurs sous portée et hors portée du site (`lecteur_position_sites_test.go`) ;
  mutation : retirer la condition `br.portee` → la carte delta (I-carte) ou un test de site rougit.
- Gate par sous-pas : I-ferm (aucune baisse), ratchet de cuisson (aucune baisse), I-carte identique
  à `$S/e0`, I-d0 (fermés ≥ valeur précédente, grenades ≠ 4 ≤ valeur précédente) ; sorties
  `$S/lk5.<n>`.

### Étape LK.6 — Révisions, gates de cuisson, clôture du jalon LK
- [ ] LK.6.1 `grammar.Rev` (aujourd'hui `grammar-2026-10-08`) monte au rang suivant ;
      `rev_chronique.go` : les mesures (R-2, R-7, R-8, LK.5), le critère amendé et son instrument
      (U-0), l'adjudication des baisses ; golden par la porte `LEVELUP_UPDATE_GRAMMAR_REV=1` (§1.4).
- [ ] LK.6.2 Gate objectives (E-7) : étape objectives de `replay-equiv` identique → porte
      `LEVELUP_UPDATE_OBJECTIVES_REV=1 go test ./internal/games/halo_infinite/film/internal/facts/objectives/ -count=1 -run '^TestObjectivesRevSuitLaSortie$' -update-objectives-rev`,
      qui régénère `testdata/objectives_rev.golden` (révision ET empreinte) à révision constante, +
      COMPLÉMENT daté dans `facts/objectives/rev.go` ; sinon montée d'`objectives.Rev` (correction 2).
- [ ] LK.6.3 killsource (E-7) : `killsource_rev.golden` régénéré par
      `LEVELUP_UPDATE_KILLSOURCE_REV=1 go test ./internal/games/halo_infinite/film/internal/facts/killsource/ -count=1 -run '^TestKillsourceRevSuitLaSortie$' -update-killsource-rev` ;
      `cmd/killsource json` sur les 19 témoins avant/après → identique : complément à révision
      constante dans `facts/killsource/rev_chronique.go` ; sinon montée de `killsource.Rev`.
- [ ] LK.6.4 `replay-equiv` sur les 20 films, faits mis de côté (`data/cache/film_facts` renommé,
      `depuis_les_faits=false` vérifié) ; `replay.SchemaVersion` (89) monte si et seulement si le
      document publié change (chronique `document_chronicle.go`, plafonds, goldens d'assemblage et de
      forme, fixtures de contrat) ; `SchemaDesFaits` (10) inchangé (forme des faits intacte).
- [ ] LK.6.5 `replay-corpus-gate` (banc de vérité compris) contre `83dc72eab` ; chaque divergence
      adjugée par un record d'image-clé qui ferme désormais, ou nommée. Ouvre la base partagée EN
      LECTURE (`OpenReadForQuery` via `replay-facts-export`, correction 5) : vérifier avant qu'aucun
      backfill ni cuisson du parc ne tourne (§0).
- [ ] LK.6.6 ADR 0037 IR-6 : la portée est une propriété de la marche d'état complet (paragraphe
      2.7.d, dernier alinéa réécrit : la lecture existe désormais, 2.7.d1 la branche) ; WALK_PORT_NOTES ;
      plan de l'étape 2 (journal) ; `.ai/REGISTRE_REPORTS.md` (lignes LK) ; `.ai/thought_log.md`.
- [ ] LK.6.7 `make gate-push`, push de la branche, CI (`gh run list --branch feat/ri-lk-images-cles --limit 3`).
- [ ] LK.6.8 Revue adversariale du diff du jalon (skill `adversarial-review`, contexte frais) ;
      chaque constat statué.
- [ ] LK.6.9 Point à l'utilisateur en langage clair ; U-5 (a) : pas de demande de fusion à ce jalon.
- Gate LK.6 : tests du film, `archlint`, `replaybuild`, vet (avec `research`), golangci-lint verts ;
  `go test -tags=integration` des paquets touchés ; killsource : aucune mort publiée perdue sur les
  19 témoins ; banc de vérité vert ; CI verte au niveau job.

### Étape D1.0 — Préalables de 2.7.d1 (Ghidra, mesures, aucun code de production)
- [x] D1.0.1 Sens et largeurs des trois champs d'i42 : G-5.
- [x] D1.0.2 Composant de la marque de portage localisé (i11 + i12 + tête d'i13) : G-6, R-11.
- [x] D1.0.3 « Dynamo Grenade » = faux positif de la fenêtre (identifiant du SPNKr relu un bit plus
      tôt : `0x9d6aaed2 << 1` + bit de tête de variante = `0x3ad55da4`) : mesure 1.
- [ ] D1.0.4 Relire dans Ghidra le consommateur des drapeaux 0x4 et 0x8 d'i13
      (`FUN_1407ee054` → `FUN_1407eef08`, écrivain et lecteurs de l'unité) : sont-ils posés par
      l'état de porteur ? Sortie LU / NON LU ; les 43 records en plus listés (film, slot, horodatage)
      pour U-2.
- [ ] D1.0.5 Crochet i42 de l'instrument : publier les deux `FUN_1406d00ec` ; comparer param[1] au
      `DrawnSlot` de la fenêtre (classes : égal, ambidextrie, changement en cours, −1).
- [ ] D1.0.6 Témoin T après i43 (U-1) : mesurer dans l'instrument, par classe A/B et par format, deux
      candidats : T1 = au moins un emplacement non vide ET chaque famille d'emplacement non vide
      connue au registre ; T2 = masque i47 égal à la bitmap i22 ; et leur taux sur le témoin +1 bit.
- [ ] D1.0.7 Réserves : écart ~5 % (réserve0 220/4 367 en A, 129/2 608 en B) : 30 records tirés,
      largeurs du composant relues (Ghidra) et comparées ; conclusion : la grammaire lit comme le jeu
      (l'écart vient de la fenêtre) ou écart nommé (la réserve reste à la fenêtre, repli).
- Gate D1.0 :
  - T retenu si, sous LK : en f24-27, B∧T aussi juste que A∧T champ par champ (armes, grenades,
    chargeurs, réserves, à 0,5 point près) ; en f20-21, A∧T et B∧T n'admettent aucun record à armes
    fausses au-delà du taux du témoin +1 bit ; sur le témoin +1 bit, T admet au plus 0,1 % des
    records. **Si aucun candidat ne passe : ARRÊT, l'utilisateur choisit entre U-1 (a) et (c).**
  - U-2 et U-3 : présentés à l'utilisateur avec les sorties de D1.0.4 et D1.0.5 s'ils changent les
    hypothèses de §3.1 ; sinon les réponses données avant l'exécution valent.

### Étape D1.1 — Un seul canal de la phase des images-clés pour l'état complet du bipède
- [ ] D1.1.1 `grammar/relecture_a_l_etendue.go` : assistant unique de relecture (Lecteur sur le
      payload, `PoserContexte`, `etatComplet` + `portee`, observation locale, `SetBitPos(co.Debut)`,
      `consumeByNameCapturing`, contrôle de débordement compté).
- [ ] D1.1.2 `grammar/keyframe_etat_complet_bipede.go` : canal `Interet{PhaseImagesCles, TI 35, nom}`
      résolu PAR NOM pour i11, i12, i13, i22, `weapon-state-ammo`, `rounds-inventory`, `overheated`
      (rangs 0..3), i42, `weapon-state-type-info` (emplacements 0..3), i47 et son Alt, i48 et son Alt.
      Sorties en `types.*` : familles par emplacement (sentinelle d'emplacement vide), `Grenades[4]`
      (n ≠ 4 → refusé et compté), munitions, `DrawnSlot` = param[1] (U-3, U-4),
      `SelectedGrenadeRank` = sel − 1 (0 = aucune), rang de capacité (hors domaine compté), marque de
      portage selon U-2.
- [ ] D1.1.3 Règle d'admission PAR RECORD selon U-1, écrite dans le code (doc = contrat) ; un record
      non admis ne rend rien (la fenêtre le prendra en D1.2).
- [ ] D1.1.4 `lireJeuDArmes` (`unit_weaponstate.go`) : crochet publiant param[1] et param[2], seule
      copie sous `lecteur_jeu_darmes_guard_test.go`.
- [ ] D1.1.5 `player_teams.go` (`lireEquipeA`) et l'instrument d0 passent par l'assistant ; garde-rail
      (règle 6, 3e copie du motif) : aucune écriture de `etatComplet` ni de `portee` hors de
      `keyframe_fullstate_loop.go` et de l'assistant.
- [ ] D1.1.6 Une seule distribution remplace les trois balayages actuels des fenêtres pour les
      records admis ; `KeyframeWalkCoverage` conservé ; aucun import de lecture côté `replay`.
- [ ] D1.1.7 Coût : G-perf cuisson (4 films) et `BenchmarkPorteursAuSync` (4 films), en PAIRES
      alternées des binaires de base de l'étape 0 (`$S/e0/bench_base.test`,
      `$S/e0/replay-equiv_base.exe`) et de ceux du lot, sur machine calme → `$S/d11/perf.txt`
      (la mesure de base seule de 0.6 est non concluante, machine chargée).
- Tests : unitaires sur records fermés tirés de fixtures (i22, i42, i43..i46, i47 base 0, capacité hors
  domaine, débordement compté) ; mutations : retirer `portee` dans l'assistant (la grammaire relit
  quantifié, rouge) ; inverser la base d'i47 ; admettre un record sans T ; garde-rails
  `lecteur_jeu_darmes_guard_test`, `film_lecture_test`, `film_file_size_test` (500 lignes),
  `no_raw_film_bytes_extraction_test`, `keyframe_fullstate_guard_test`.
- Gate D1.1 : instrument d0 rebranché sur le canal : valeurs égales à l'instrument sur 100 % des
  records admis ; couverture publiée (admis / 10 710, par format et par film) ; aucun débordement ;
  **coût ≤ +10 % de durée et de pic, cuisson et sync, sur chacun des 4 films (sinon ARRÊT : mesure
  publiée, l'utilisateur décide)**.

### Étape D1.2 — Les fenêtres derrière la lecture : replis nommés, ordonnés, comptés
- [ ] D1.2.1 `keyframe_loadout.go`, `keyframe_carrier_mark.go`, `inventory_decode.go`,
      `inventory_ammo_rules.go`, `inventory_grenades_rules.go`, `inventory_grenade_selection.go` :
      appelées seulement sur les records non admis.
- [ ] D1.2.2 Trois entrées au registre (`facts/fallback`, nouveau `registre_filmdec_images_cles.go`,
      constantes dans `noms.go`) : `repli_fenetre_armes_image_cle`,
      `repli_fenetre_inventaire_image_cle`, `repli_fenetre_marque_de_portage` (la dernière seulement
      si U-2 (a) ou (b)). Chacune : `OrdreApresLecture`, `CondLectureNonPortee`, sites et ancres
      littérales, `DatePose` (date du commit), `CibleRetrait` = 2026-12-31, `CritereRetrait` (D1.2.3),
      `CompteurBranche`.
- [ ] D1.2.3 `CritereRetrait` (corrigé sur la mesure, plus « 861 arrêts ») : part des records
      bipèdes d'image-clé non admis ≤ 5 % sur les 28 films ET sur le parc à la recuisson. Population
      à la pose, nommée : formats 20-21 (2 055 records, valeurs fausses après i22) ; `60ae07c4`
      (+33 bits, 331 records) ; écarts de fin multiples de −108 (2 800, en-têtes non vus, R-VEH-4) ;
      autres écarts non nuls (2 224, dont 2 100 sur 4 films) ; 88 arrêts sur i59/i58 (7 films) ;
      dernier record de paquet sans frontière.
- [ ] D1.2.4 Records rendus par la fenêtre marqués `PreuveRecupere` avec leur méthode (DT2-4) ;
      `ComptesDesReplis` ; cliquet `NbDevantLaLecture` inchangé ; `repli_plafond_grenade_par_defaut`
      resserré à la fenêtre, doc corrigée ; `replay/versement_des_replis.go`,
      `archlint/keyframe_walk_proof_test.go` (allowlist, dans les deux sens),
      `archlint/fallback_versement_test.go`.
- Tests : `TestToutSiteDuRegistreExiste`, `TestChaqueNomConstantEstAuRegistre`,
  `fallback_versement_test` (directions C et E), `keyframe_walk_proof_test` ; unitaire : un record
  admis ne passe jamais par la fenêtre, un non admis y passe et il est compté ; mutation : fenêtre
  appelée avant la lecture → le test d'ordre rougit.
- Gate D1.2 : compte des replis = records non admis, au record près, sur les 28 films ; archlint vert.

### Étape D1.3 — Consommateurs du rejeu et survie depuis les faits
- [ ] D1.3.1 `replay/film_scan.go` (`balayerInventaire`), `porteurs_au_sync.go`
      (`lireLesArmesTenues`, l.199), `build_objectives_live.go` (marques), `loadouts.go`,
      `inventory.go`, `grenade_reads.go`, `abilities.go`, `document_weapon_changes.go` lisent le canal,
      puis la fenêtre sur les seuls records récupérés. L'ordre de `Loadout.W` ne bouge pas
      (`inventory[].am`) ; aucun ajout à `decfilm` (`film_facade_surface_test`).
- [ ] D1.3.2 Survie depuis les faits (ADR 0034) : test du paquet `replay` qui cuit une bobine
      fraîche puis publie depuis ses faits, et compare les comptes des trois replis, les marques
      `PreuveRecupere` et les valeurs d'inventaire publiées. Identiques → `SchemaDesFaits` 10 reste ;
      sinon les faits portent ce qui manque (section `rapport de replis`, `filmfacts_fichier.go:206`)
      et `SchemaDesFaits` 10 → 11, avec sa chronique.
- Tests : goldens d'assemblage et de forme ; unitaires `buildInventory`, `buildGrenadeReads`,
  `buildAbilityReads` avec Src « kf » ; `markInventoryDeadReadings` conservé (règles 6 et 8 de 2.7.b).
- Gate D1.3 : tests du paquet `replay` verts ; diffs de goldens tous expliqués par une valeur de la
  grammaire (U-3) ; test de survie vert.

### Étape D1.4 — Gates, révisions, clôture de 2.7.d1
- [ ] D1.4.1 `grammar.Rev` monte (chronique : couverture, admission, replis ; porte
      `LEVELUP_UPDATE_GRAMMAR_REV=1`, §1.4) ; gate objectives et killsource comme LK.6.2 et LK.6.3, par
      leurs portes (E-7, §1.4) ; `objectives.Rev` monte si le calque des drapeaux change (U-2).
- [ ] D1.4.2 `replay.SchemaVersion` monte (valeurs d'inventaire, dotations, capacités) avec chronique,
      plafonds, goldens, fixtures ; `SchemaDesFaits` selon D1.3.2.
- [ ] D1.4.3 ADR 0037 : AMENDEMENT de la règle de 2.7.b pour les images-clés (texte de U-1 : n(i22) = 4
      ne valide le curseur que jusqu'à i22 ; le témoin T après i43 ; la fermeture seule ne garantit
      pas les valeurs dans les formats 20-21) ; IR-6 (les fenêtres derrière la lecture) ; dernier
      alinéa de 2.7.d réécrit.
- [ ] D1.4.4 Mesure finale : instrument d0 (grammaire puis fenêtre derrière) ; `replay-equiv` 20
      films, faits frais ; `cmd/killsource json` 19 témoins ; `replay-corpus-gate` et banc de vérité
      (lecture de la base partagée par `OpenReadForQuery`, §0 : aucun backfill ni cuisson du parc en
      cours) ; Playwright (`npx playwright test`) si l'affichage des fiches change ; G-perf cuisson et
      sync sur les 4 films.
- [ ] D1.4.5 Plan de l'étape 2 : 2.7.d1 `[x]`, 2.7.d clos ; REGISTRE_REPORTS ; thought_log ;
      `make gate-push` ; push ; CI.
- [ ] D1.4.6 Revue adversariale du diff de 2.7.d1 (contexte frais) ; constats statués.
- [ ] D1.4.7 Fusion dans `feat/v75` (U-5, accord donné d'avance le 2026-10-08), puis recuisson du
      parc et backfill killsource, après la fin de tout backfill en cours ; pairs prévenus avant la
      coupure du serveur local et à la fusion ; compte rendu à l'utilisateur.
- Gate D1.4 : fiches au moins aussi pleines qu'avant (records avec armes = grammaire + repli ≥
  fenêtre seule, par film) ; aucune perte non adjugée dans `replay-equiv` ; aucune mort perdue
  (killsource) ; banc de vérité vert ; **coût ≤ +10 % (cuisson et sync, 4 films)** ; CI verte.

## 5. Contrat d'exécution et gates

Skill `plan-execution` (ordre strict, une étape commencée est terminée, aucun report d'une action
exécutable maintenant, statut de chaque item, clôture d'étape = gate + items statués + plan à jour +
entrée `.ai/thought_log.md` au jalon) ; jamais `git stash` ; jamais Python ; aucune commande `go`
concurrente ; aucune base DuckDB ouverte, hors la lecture `OpenReadForQuery` de G-corpus (§0) ;
commentaires = contrat (règle 17).

| Nom | Commande (depuis `$WT/apps/go-api`) |
|---|---|
| G-unit | `go test <paquets du lot> -count=1` (entre deux jalons : `-skip "$SKIPREV"`, la liste des sept gardes de révision, de chronique et d'équivalence du §1.4) |
| G-arch | `go test ./internal/archlint/ -count=1` |
| G-vet | `go vet ./internal/games/halo_infinite/film/...` (CGO) et `go vet -tags=research ./internal/games/halo_infinite/film/...` |
| G-film | `go test ./internal/games/halo_infinite/film/... ./internal/replaybuild/... ./internal/sync/killcollector/... -count=1` |
| G-equiv | `go run ./cmd/replay-equiv -repo-root $WT` (faits mis de côté, `depuis_les_faits=false`) |
| killsource | `go run ./cmd/killsource json <film> -carte <carte> -cache $WT/data/cache` sur les 19 témoins de `config/replay_corpus.toml`, avant/après, `cmp` |
| G-corpus | `go run ./cmd/replay-corpus-gate` (mode `base` contre `83dc72eab`, banc `cmd/replay-verite` compris). OUVRE LA BASE PARTAGÉE EN LECTURE : `levelup replay-facts-export` par `OpenReadForQuery` (`cmd/replay-corpus-gate/facts.go`) ; seulement quand aucun backfill ni cuisson du parc ne tourne (§0, correction 5) |
| G-perf | (correction 13) Binaires de base (la tête de l'étape 0, code de production de `83dc72eab`) et du lot, construits une fois chacun et ALTERNÉS (base, lot, base, lot…), au moins TROIS paires, machine calme (aucune autre commande `go`, aucun `*.test.exe` d'une autre session : `tasklist` contrôlé avant et après chaque tour), faits jamais relus. **Cuisson** : `replay-equiv` (binaire `go build -o <bin> ./cmd/replay-equiv`) `-repo-root $WT -films 084a804d,e5adf7b2,60ae07c4,11de8353` ; il force le décodage (`SansFaitsPersistes`) et imprime durée et pic (`filmproc.Footprint`, sentinelle) par film. **Sync** : banc `BenchmarkPorteursAuSync` du paquet `internal/sync/killcollector` (binaire `go test -c -tags=research -o <bin> ./internal/sync/killcollector/`, lancé depuis le dossier du paquet), un film par lancement : `PORTEURS_BENCH_FILMS=<film> PORTEURS_BENCH_CACHE=$WT/data/cache <bin> -test.run '^$' -test.bench '^BenchmarkPorteursAuSync$' -test.benchtime=<N>x -test.count=1 -test.benchmem`, N = 1 pour `084a804d` et `1c4c63c2` (10 à 30 s par appel), N = 5 pour `51101d1d` et `9f57c612` (moins d'une seconde par appel : la moyenne de cinq tours amortit le bruit) ; métriques `ns/op` et `pic-Mio`. **Jugement du seuil de 10 % sous un bruit annoncé de 15 %** : on compare des PAIRES (base et lot mesurés l'un après l'autre, dans le même état de machine), jamais deux sessions ; le verdict porte sur la MÉDIANE des rapports lot/base par film ; le bruit du moment se mesure par l'écart entre les mesures de base de la même session (max/min − 1). Si ce bruit dépasse 5 %, la mesure n'est PAS concluante : on la refait sur machine calme, on ne conclut rien. Bruit ≤ 5 % et médiane ≤ 1,10 sur chaque film : seuil tenu ; médiane > 1,10 sur un film : ARRÊT du gate (mesure publiée, l'utilisateur décide). Le pic de la cuisson se juge en plus sur le tas vivant par phase (trace du ramasse-miettes) quand il bouge, découverte 9 du plan de l'étape 2. Écart par paire, médiane et bruit publiés |
| G-CI | `gh run list --branch feat/ri-lk-images-cles --limit 3` → jobs verts |
| G-push | `make gate-push` avant toute demande de fusion |

**Pièges d'exploitation** (plan de l'étape 2, §4, toujours valides) : fraîcheur des faits = révisions
déclarées (renommer `data/cache/film_facts`, dossier réel) ; durée et pic varient de 15 % sous charge
(binaires alternés) ; un PC en veille gèle une passe sans la tuer ; `replay-equiv` avec `-repo-root`,
jamais `LEVELUP_REPO_ROOT` ; éditer par motif exact, jamais par numéro de ligne après un premier ajout ;
le nom d'un fichier peut porter un retour chariot (constaté sur `wf1_plan.json`) : passer par un motif.

## 6. Protocole de reprise

Relire le skill `plan-execution`, puis ce fichier (§2, §3, §8) ; reprendre à la première case non
statuée ; `git -C $WT log --oneline -10` ; `ListAgents` si le worktree semble repris par une autre
session ; vérifier sur `origin/feat/v75` si un lot a été fusionné depuis la dernière reprise (§1.3 :
refusionner, rejouer l'étape 0).

## 7. Découvertes (consignées, non traitées)

- D-1 *(Ghidra, seconde passe)* La forme en ligne du delta d'i0 (`consumePositionHandleTail`,
  `bHandle = 1`) lit le handle sur `pd.IndexW` (= 1, largeur d'index de région `DAT_144632be0`) au lieu
  de `varWidthBits(0)` = 13. Hors périmètre (delta).
- D-2 *(mesure 2)* `GrammaireEcrivainI0` en delta : +2 trames saines (`11de8353` 27:1060, `1c4c63c2`
  47:1956), 0 perdue, 15 paquets ne changent que de cause. Source (correction 11) : la carte v2 sur
  20 films, `m2/carte_v2_base/` contre `m2/carte_v2_i0e/` (`paste` des deux `fermeture_paquets.tsv` :
  20 paquets changés, dont 15 sans changement de `ferme_au_bit` ni de `ferme`) ;
  `m2/i0e_mpp_paquets_changes.txt` n'a que les 4 lignes de la carte MPP sur 28 films.
- D-3 *(mesure 2)* Le critère écrit annonce 591 records bornés ; la marche actuelle en borne 599. Non
  instruit (la marche `WalkKeyframeWorld` a probablement changé depuis R7).
- D-4 *(mesure 1)* `50247b26` (format 20) illisible en i22 dans les deux modes, alors que `a349fea8`
  (même format) lit n(i22) = 4 dans 979 records sous LK.
  *État sous LK.3 (2026-10-08, LK.4.3, sans correction)* : inchangé. `50247b26` : n(i22) = 4 dans 37
  records sur 668 (11 égales, 14 différentes, 12 grammaire seule), i22 lu à un autre compte dans 332,
  non atteint dans 299 (247 sous le crochet) — dont les 52 records qui s'arrêtent désormais au début
  d'i0 (51 `largeur_handle_moteur_un`, 1 `position_non_finie` ; le film n'a pas de section
  d'identification) ;
  `a349fea8` : 979 sur 984. Commande : I-d0 (§1.4) → `$S/lk3/images_cles.tsv`, puis
  `awk -F'\t' '$1=="S" && ($2=="50247b26"||$2=="a349fea8") && $3 ~ /^(grenades|desync_)/ && $3 !~ /\|/'`.
- D-5 *(mesure 1)* Formats 20-21 : records fermés (24) et à n(i22) = 4 (1 404) à armes et munitions
  fausses après i22.
  *État sous LK.3 (LK.4.3, sans correction)* : inchangé pour les fermés — classe A 24 records, armes
  0/16, grenades 11/11, chargeurs 0/12 et 0/10, réserves 0/15 et 0/15 ; classe B 1 400 records (1 404
  sous le crochet : les quatre records de `50247b26` arrêtés au début d'i0 passent en C), armes
  55/1 184, grenades 1 138/1 155, chargeur0 45/791, réserve0 49/1 155. Commande :
  `bash $REF/ref_corrigee/taux.sh $S/lk3/images_cles.tsv` → `$S/lk4/taux_lk3.txt`.
- D-6 *(mesure 1)* Sous LK, le témoin +1 bit ferme 227 records dont 223 aussi fermés en vrai : le
  décalage d'un bit est absorbé, le témoin n'y est plus indépendant ; taux de hasard propre 4/5 311.
  *État sous LK.3 (LK.4.3, sans correction)* : 226 fermés au témoin, dont 222 aussi fermés en vrai
  (f24 63/62, f25 12/12, f27 149/148, f20 1/0, f21 1/0) ; le record de `f25` perdu par le témoin s'y
  arrête sur `position_non_finie` (lecture décalée ; attribué par la mesure sans les arrêts,
  `$S/lk3/sans_arrets/`) ; taux de hasard propre inchangé, 4/5 311 (226 − 222 fermetures du seul
  témoin sur 10 710 − 5 399 records non fermés). Commande :
  `bash $REF/ref_corrigee/par_format.sh $S/lk3/images_cles.tsv` → `$S/lk4/par_format_lk3.tsv`.
- D-7 *(plan)* La garde low-frequency de `ti=3` repose sur une sonde (D-REV-2 : `fb1a1a72` 0/26 →
  19/26), pas sur une lecture : à relire (LK.1.8).
- D-8 *(mesure 2)* Sous LK, traversées changées sans changement de fermeture : `ti=40` 6 183 (désync
  i30 → i30 pour 6 047), `ti=21` 1 750 (+76/+77 bits), `ti=44` 5 (`m2/non_bipedes_lk.txt`).
- D-9 *(Ghidra)* Le témoin « precHigh = 1 → R(59) » de world-object i0 hors portée
  (`lecteur_position_exceptions.go:38`) n'est atteignable qu'avec un handle de 11 bits (axes
  14/14/14) ou de 13 (13/13/14) : tension non tranchée (relire `FUN_140be9b88`).
- D-10 *(Ghidra)* Huit fonctions posent `DAT_144e61ea0`, pas six ; quatre travaillent sur un tampon
  local de comparaison (`FUN_142e309b4` exécute la boucle DELTA `FUN_14076cb60` sous la portée) : la
  delta n'est hors portée que dans son contexte d'appel par le film — d'où une portée portée par le
  Lecteur et posée par la seule marche d'état complet.
- D-11 *(mesure 1)* Écart de réserve ~5 % identique en A et en B (D1.0.7 l'instruit).
- D-12 *(LK.3)* La structure de lecture ne porte pas la cause d'un arrêt de lecteur : `EntityTrace.Arret`
  la nomme, mais `lecture.Record` ne garde que l'index (`Desync`), et `KeyframeClosure` comme I-ferm
  nomment « bloquant » le composant arrêté (`i0 object-position-dynamic-precision-component` sur
  `50247b26` `ti=35`, à la place de `i58`) : un arrêt de lecteur s'y lit comme un composant non porté.
- D-13 *(LK.3)* `Mouvement.FullPrecision`, miroir de `DAT_145121140 == 1`, est présumé faux pour tous
  les films (`profile_table.go`, « Movement.FullPrecision » = false), alors que le film déclare son type
  de moteur (`VarianteDePartie.TypeDeMoteur`, déjà lu par la vue A) ; un film de type 1 lirait toutes
  ses positions en brut. Sans effet mesuré : type 2 sur les 26 films à identité du corpus
  (`$S/lk3/variantes_28.tsv`) et sur les 1 656 du cache (LOT VA).
- D-14 *(LK.3)* Sous la portée, les 96 bits de la branche absolue d'i0 sont la position EXACTE du
  bipède (trois `float32`) ; `semerPositionAbsolue` ne sème pas une forme brute, la graine
  d'accumulation ne la lit donc pas (consommateurs du rejeu : D1.3).
- D-15 *(LK.3)* Les instruments `campagne_overlay` (`r_veh_ti40`, `r_veh_ti40_variantes`,
  `r_veh_imagecle`) ont été adaptés sans pouvoir être compilés : leur surcouche (`r_veh_overlay/`, les
  `overlay_campagne.json`) n'est pas dans le dépôt et remplace des fichiers entiers d'une tête ancienne.
- D-16 *(LK.4.2)* `60ae07c4` (HI_1_8_0) : le premier composant de largeur divergente du +33 est i53
  `biped-malleable-property-component` (`FUN_140ff6764` : bloc `FUN_1407efc5c`, puis R(5) = n et
  R(n)) — 36 bits dans 308/332 records contre 30 dans 953/956 records fermés de `084a804d` et
  `111fa685` (HI_1_10_0, même registre) ; i57, i55, i58 divergent en aval selon la population ; un
  bit constant reste non localisé. Commandes et sorties : LK.4.2 (`$S/lk4/sonde_f24/`). À relire dans
  le jeu avant toute correction (règle des largeurs lues).
- D-17 *(LK.5.1, REJETÉ)* i20 `lireViseeDActeurAncienne` sous la garde de pleine précision : R(96) par
  emplacement a = 0, comme le jeu (`FUN_14058c058`, les deux appels `FUN_14076e494(.., 0x10, 0,
  param_3, 0)` relus le 2026-10-08, `$S/lk5/dec_14058c058.txt`). Mesure contre la tête (`$S/lk5.0`,
  identique à `$S/lk3` à l'octet) → `$S/lk5.1` : I-ferm, UNE ligne change, `50247b26` `ti=35` 6 → 5,
  aucune hausse sur les 28 films ; I-d0 : seul `50247b26` bouge (traversés 588 → 590, fermés 6 → 5,
  n(i22) ≠ 4 332 → 328) ; goldens de fermeture (sans carte, cuisson) inchangés ; carte identique à
  `e0` (14/14, 14/14, 15/15). Le record perdu (TS 6586064022, slot 552, bit 198 373, classe A,
  n(i22) = 0) fermait sans lire aucune famille ; sous la garde il lit les armes de la fenêtre
  (Bandit Evo + MA40 AR) et finit 33 bits plus loin (2 663 contre 2 630). Commit mesuré `f4f8f893b`,
  retiré par le commit suivant. La lecture du jeu reste à porter : la retenir demande d'adjuger cette
  perte (fermeture sans lecture d'un film illisible dès i22, D-4), ce que la règle de LK.5 ne prévoit
  pas.
- D-18 *(LK.5.4.1, REJETÉ — le levier le plus fort de LK.5)* world-object i0
  (`consumeObjectPositionMonde`) sous la garde : la forme de `FUN_14076e29c` (precHigh R(1), R(96) par
  `FUN_14076e420` → `FUN_14076e494`, queue `FUN_14076e3e4(precHigh)` — `MOV R9B,AL` en 14076e2c5 —,
  R(2) si les trois flottants sont finis, faux sinon en 14076e2fe ; `REF/g2/dis_14076e29c.txt`), la
  même que la branche absolue d'i0 (`consumeAbsoluSousLaPortee`). Mesure contre `$S/lk5.2` →
  `$S/lk5.4.1` : I-ferm, 82 hausses — `ti=38` 6 565 → 110 221 / 119 142, `ti=42` 1 107 → 10 162 /
  23 402, `ti=43` 2 560 → 36 550 / 40 712 sur 28 films — et 15 BAISSES : `ti=37` (équipement, qui porte
  le même i0) 196 → 143 sur 11 films (`396cfc92` 16 → 0 à records constants, `084a804d` 20 → 14,
  `a349fea8` 17 → 4, `1c4c63c2` 21 → 14, …) ; `50247b26` `ti=38` 488 → 158 et `ti=42` 76 → 11 (arrêt
  `position_non_finie` sur i0 : les 96 bits n'y sont pas trois flottants finis ; sans la garde E-4,
  mesure temporaire `$S/lk5.4.1/sans_garde_e4`, 158 et 23) ; `1c4c63c2` `ti=12` 5 → 4 ; une ligne
  disparue (`696a9d7c` `ti=0`, 0/1). Goldens de fermeture (7 bobines) : hausses `ti=38/42/43`, BAISSE
  `ti=37` (cuisson `11de8353` 4 → 1 ; sans carte `11de8353` et `111fa685` 2 → 1). I-d0 (bipèdes)
  inchangé. ÉLECTION DES ANCRES CHANGÉE sur 5 films (I-ancres `$S/lk5.4.1/m2_*` contre `$S/lk3`) :
  élections `1c4c63c2` 4 926 → 4 898, `a349fea8` 9 044 → 9 008, `696a9d7c` 1 047 → 1 065, `0797ce72`
  390 → 401, `53ce4390` 1 807 → 1 806 (des ancres `ti=37`/`ti=42` passent d'élues à écartées et
  inversement), et par elle la CARTE DELTA change : `1c4c63c2` 44 424 → 44 374 paquets fermés (−50),
  `696a9d7c` +1, listes non localisées de ±1 à ±9 sur 4 films. Commit mesuré `64ef3baa7`, retiré par le
  commit suivant. Le retenir demande une décision : adjuger les baisses `ti=37` (fermetures rares,
  0,66 % → 0,48 %, d'un archétype dont un autre composant est probablement mal porté), celles de
  `50247b26` (film illisible dès i22, D-4), et accepter le changement d'élection et de carte, contraire
  au critère 1 de LK (ancres identiques à la base).
- D-19 *(LK.5.4.1)* Avec la queue fidèle de `FUN_14076e3e4` (LK.5.2), la largeur W0 du handle n'est
  utilisée que quand la porte de `FUN_1408f0ac4` vaut 1 ; la garde E-4 (LK.3.7) arrête dès h (ou
  precHigh) = 1. La placer sur cette porte arrêterait moins de records des films sans section
  d'identification sans présumer aucune largeur. Non mesuré sur les bipèdes ; non traité (décision de
  LK.3.7).

## 8. Journal

- 2026-10-08 : rédaction (seconde passe du workflow 2.7.d1, aucune ligne de production modifiée,
  rien de commité). Références rejouées dans la session : agrégats R-1 à R-6 et R-11 sur
  `ref_corrigee/`, critère R-8 (`TestRI27d1M2Critere`, 369/599 = 61,6 %), bobines R-7
  (`TestRI27d1M2Bobines`, identique à l'octet à la mesure 2), cartes R-9 et ancres R-10 (`cmp`),
  dérive R-12 ; sorties `REF/plan_verif/`. Verdict : GO pour LK sous réserve des réponses U-1 à U-5.
- 2026-10-08 (soir) : INSTALLATION du plan dans `feat/ri-lk-images-cles` (worktree `LevelUp-wt-ri`,
  base `feat/ri-etape2` `81d4831d6`, item 0.2). Décisions de l'utilisateur du 2026-10-08 (« Je suis
  ok », réponse à la liste des décisions recommandées et au coût annoncé) écrites au §3.1 comme
  décisions datées : U-0 (amendement du critère écrit), U-5 (a), U-1 (b), U-2 (a) sinon (b), U-3,
  U-4, et E-2, E-6, D1.2.3 acceptées ; restent à l'exécutant E-3, E-4, E-5, E-7, E-8, E-9 (§3.2).
  Les treize corrections de la contre-vérification (`REF/wf2_verdict.json`, verdict « partiel »)
  sont intégrées, chacune relue sur pièces : (1) I-ferm porte `RI27C_FILMS/RACINE/OUT/CARTES` et un
  contrôle `--- PASS` contre `--- SKIP` (§1.4) ; (2) portes de révision avec leurs variables
  `LEVELUP_UPDATE_*_REV=1`, golden objectives = `testdata/objectives_rev.golden` (§1.4, E-7, LK.6.1 à
  LK.6.3, D1.4.1) ; (3) helper de contexte de carte recompté — deux copies de production déjà
  (cuisson/sync et killsource), choix d'un garde-rail restreint au code de production avec allowlist
  datée, killsource hors du helper, raison écrite (E-5, LK.2.3) ; (4) `-skip` des gardes de révision
  dans les gates intermédiaires (`SKIPREV`, §1.4, LK.2, LK.3) ; (5) G-corpus signalé comme ouvrant la
  base partagée en lecture (`OpenReadForQuery`), admis puisqu'aucun backfill ne tourne (§0, §5,
  LK.6.5, D1.4.4) ; (6) adjudication « 8 sur 9 » (`50247b26` slot 524 reste à n(i22) = 0) et baisse
  des traversés 636 → 635 de `50247b26` adjugée — solde de 19 records qui s'arrêtent sur `i58` sous
  LK et de 18 qui cessent de s'y arrêter, tous non fermés dans les deux modes (lignes R des dumps
  `m2/dump_{base,portee}`, rejouées le 2026-10-08) (§2) ; (7) chiffre de la portée seule publié,
  8/599, et le critère rendu à l'utilisateur (U-0), avec `TestKF35CBaselineScope` rejoué en 0.5 ;
  (8) E-6 et (9) E-2, D1.2.3 : acceptées par l'utilisateur ; E-4 conditionnée à une relecture Ghidra
  (nouvel item LK.1.10) ; (10) 149 records MX (et non 148) ; (11) source des 15 paquets de D-2
  (`m2/carte_v2_base` contre `m2/carte_v2_i0e`) ; (12) base de la branche (§1.1, 0.1) ; (13) G-perf :
  paquet `replay`, variables `PORTEURS_BENCH_FILMS` / `PORTEURS_BENCH_CACHE`, prédicat réel
  `GardesDeLaVariante(...).Drapeau || .Bombe`, jugement par paires alternées, médiane et bruit (§5,
  0.6). Instruments : noms de sortie en mode unique fixés au §1.2 ; `TestRI27d1FermeturePortee` et
  `TestRI27d1M2Bobines` laissés de côté (mesures du crochet seulement). Plan de l'étape 2 : 2.7.d1
  renvoie à ce plan (reste `[!]`). Item 0.2 `[x]` au commit `ri-lk(0)`.
- 2026-10-08 (soir) : ÉTAPE 0 CLOSE (0.1 à 0.5 `[x]`, 0.6 `[!]`). Sorties sous `$S/e0/` (`S` =
  `scratchpad/ri/lk`). **0.1** : branche `feat/ri-lk-images-cles`, arbre propre, jonctions
  `film_chunks`, `film_manifests`, `node_modules` en place, `data/cache/film_facts` absent.
  **0.3** : patch relu fichier par fichier contre `LevelUp-wt-imagecle` (`diff` consigné à la
  session) : `ri27d0_images_cles_research_test.go` = version du worktree de mesure moins le bloc
  `rechercheLK` de `ImageCle`, moins le choix de sortie par `RI27D1_PORTEE`/`RI27D1_NOM` (sortie
  unique `images_cles.tsv`), moins le champ `marques` devenu mort, en-tête réécrit ;
  `ri27d1_instrument_research_test.go` = identique, en-tête réécrit ;
  `ri27d1_fermeture_portee_research_test.go` = `TestRI27d1FermetureCorpus` en mode unique (étiquette
  `tete`) et `TestRI27d1Formats`, `TestRI27d1FermeturePortee` laissé de côté ;
  `ri27d1_m2_critere_research_test.go` = cas REF et « (d+e) bascules du profil », cas du crochet
  retiré ; `ri27d1_m2_ancres_research_test.go` = `TestRI27d1M2Ancres` (sorties `m2_*_tete.tsv`) et
  `TestRI27d1M2Equipes` en mode unique, `TestRI27d1M2Bobines` et `ri27d1M2ContexteCarte` laissés de
  côté (§1.2). Non repris : le crochet, `ri27d1_crochet_env_research.go`, la variante de
  `cmd_fermeture`. `gofmt` propre ; `go vet -tags=research $G` code 0. **0.4** : I-d0 (21,5 s) :
  71 496 lignes, `diff` des lignes triées contre `ref_corrigee/images_cles_base.tsv` VIDE, agrégat
  TOTAL = R-1 à l'octet (`cmp` contre `plan_verif/agg_base.tsv`) ; I-ferm : `--- PASS` (12,8 s), 839
  lignes, identiques aux lignes `base` de la référence ; I-ancres : `m2_ancres_tete.tsv` (477 665
  lignes) et `m2_stats_tete.tsv` identiques à l'octet à `m2_*_base.tsv` ; I-equipes : 28 lignes
  identiques aux lignes `base` ; I-carte : 20 films 14/14 TSV identiques, 8 films 14/14, 28 films
  sous MPP déclaré 15/15 — après correction de la commande : la référence MPP a été mesurée dans
  l'ordre `$F20,$F8`, et `-films $FILMS` rend les mêmes lignes dans un autre ordre (contenu trié
  identique) ; la commande I-carte du §1.4 nomme désormais l'ordre. **0.5** : bascules en place,
  `TestKF35CBaselineScope` 0/599 sur ses 24 variantes (lignes identiques à la mesure 2) ;
  `TestKF7EFullStateLoop` : REF 9 + 9 + 7 = 25/599, (d) 1 + 5 + 2 = 8/599, (e) 25/599, (d+e)
  125 + 124 + 120 = 369/599 ; `TestRI27d1M2Critere` : REF 25/599, (d+e) 369/599 = 61,60 % avec ET
  sans bouchons. **0.6** : `[!]`, détail à l'item (banc `BenchmarkPorteursAuSync` dans
  `internal/sync/killcollector`, paquet du collecteur pour en réutiliser les coutures ; films du
  sync : trois à drapeau et un à bombe ; aucune mesure prise sur machine calme, d'autres sessions
  testant en continu ; cuisson de base 4/4 identique aux références). **Gate 0** : toutes les
  lignes vertes ; aucune différence avec `REF` hors l'ordre des films de la carte MPP, expliqué.
  `feat/v75` a avancé de deux commits DOCUMENTAIRES pendant l'étape (`fe6308b71` : journal ; `e3384ec3b`
  « Archivage » : le plan de l'étape 2 passe de `.ai/` à `.ai/V7.5/`, avec trente-quatre autres
  documents), aucun fichier `film/` ; non fusionnés (consigne du superviseur). À la fusion du §1.3,
  le renvoi de 2.7.d1 (commit `ri-lk(0)`) suit le fichier déplacé et les chemins de ce plan vers le
  plan de l'étape 2 sont à mettre à jour.
- 2026-10-08 (soir) : ÉTAPE LK.1 CLOSE (LK.1.1 à LK.1.10 `[x]`, aucune largeur restée non lue).
  Relectures Ghidra en lecture seule (sorties `$S/lk1/`), consignées en WALK_PORT_NOTES §6.5.
  **LK.1.6** : `FUN_140c1e79c` = R(1) (140c1e7d9) ; si 0 : R(19) (140c1e84e, `FUN_1406d8288(.., 0x13)`
  140c1e875), sinon vecteur constant ; puis `FUN_1406d84b4` avec la largeur 8 posée en `[RSP+0x20]`
  (140c1e80f) = R(8) — égal au Go `consumeCompressedDir140c1e79c` ; `FUN_1424e268c` = R(2). Les deux
  LUES. **LK.1.7** : `get_xrefs_to(14076f91c)` = 18 appels dans 17 fonctions ; deux absentes de la
  liste du plan sont des ÉCRIVAINS (`FUN_1407eb61c`, `FUN_142e2c9bc`, ce dernier écrit la branche
  absolue d'i0), `FUN_14076f75c` quantifie un vec3 sans lecteur de bits ; les lecteurs consultent la
  garde en Go sauf deux exceptions datées (waypoint-state, flock-destination) ; quatre sont des
  charges de vue A (jamais sous la portée), `ObjectCollisionDamage` n'est pas porté. Les 14 lectures
  en ligne de `DAT_144e61ea0` : 7 lecteurs de la famille 0x1E (non portés) et 7 écrivains. CALL
  recontrôlés : navpoint (garde en ligne 140f04f72 puis 140f04f8b -> `FUN_14076e524`), spawn-filter
  142b6ef31, selectable-zone 14145437e, asset-transform 142ed9556 (-> `FUN_14076e494`), i0 prédit
  140f7ea5c (-> `FUN_14076e4ec`) : tous gardés, comme leurs lecteurs Go. **LK.1.8** : lien LU — les
  positions de `low-frequency` passent par `FUN_1424e0e38` -> `FUN_14076e494` -> la garde ; son
  avant/haut (`FUN_140c5f938`) ne lit que `DAT_145121140` : la garde `etatComplet` de
  `consumeLowFrequency` est levable sous la portée (LK.5.5). **LK.1.10** : LU — W0 ne diffère de 13
  que si `m_gameEngineType` = 1 (drapeaux de croissance +0x138/+0x158 posés par `FUN_140d10a78` à
  `DAT_145121140 == 1` ; `FUN_140a938b4` -> `FUN_14051a4b8` -> `FUN_140a93ec8` ; recherche
  d'instructions rejouée sur tout le programme, résidu nommé : une copie de structure) ; E-4 et
  LK.3.7 mis à jour. **Gate LK.1** : tableau complet, aucune case vide, aucune largeur non lue.
- 2026-10-08 (soir) : ÉTAPE LK.2 CLOSE (LK.2.1 à LK.2.4 `[x]`). Sorties sous `$S/lk2/`.
  **LK.2.1** : dernier `-update` du golden sans carte = `e9a64d87b` (2.7.c3) ; bissection par
  `git archive` d'`apps/go-api` à chaque commit (lecture seule du dépôt) et régénération du golden
  hors du worktree : `e9a64d87b` régénéré = golden commis ; `1685ee2bf` (juste avant la fusion de la
  vue B) = golden commis ; `167bd211a` (fusion de feat/v75 `acfe4851a`, arrêts de la vue B) = la
  tête, 21 lignes de dérive, rien ne bouge après. Dans le lot de la vue B : `ec9897101` (C1, ti=43
  device-*) change les 7 lignes ti=43, `0962d0970` (C3) les 7 bloquants ti=45, `e480f6dbb` (C4) les 7
  bloquants ti=10 ; C2 et C5 rien. **LK.2.2** : commit `383faf5a2`, golden régénéré = sortie de
  `167bd211a` à l'octet, 21 lignes (7 × ti=10, ti=43, ti=45), 0 baisse, historique dans le générateur
  (`keyframe_closure_ratchet_test.go` 410 → 419 lignes). **LK.2.3** : geste unique
  `grammar.FilmContext.PoserLaCarteEtLeDecoupage` (`grammar/contexte_de_carte.go`, nouveau, hors de
  `film_context.go` qui est à 500 lignes) : la séquence de `replay.installWorldObjectPrecision`
  déplacée telle quelle (profil rendu, pose, profil reposé) puis la résolution MPP ;
  `installWorldObjectPrecision` l'appelle et garde son compte et son journal d'une entrée sans
  largeurs ; `poserLeDecoupageMPPDuFilm` devient `signalerLeDecoupageMPP` (journal seul) ;
  `poserProfilPuisCarte` inchangé dans son ordre ; ancres du registre déplacées
  (`repli_largeurs_monde_par_defaut_conservees` -> `contexte_de_carte.go` ;
  `repli_largeurs_axe_par_defaut_conservees` : décision dans `contexte_de_carte.go`, compte dans
  `world_object_precision.go`) ; renvois de commentaires corrigés (`film_context.go`, `mpp_declare.go`
  de la grammaire et de `cmd_fermeture`, `killsource/decode.go` — commentaires seuls, hors empreinte) ;
  `TestLaCuissonPoseLeDecoupageDeclareParLeFilm` joue désormais `poserProfilPuisCarte`. Garde-rail
  `archlint/pose_de_carte_unique_test.go` (AST, appels de `PoserLargeursObjetDuMondeDepuisDecoupage`
  en production, liste fermée datée : le geste, `film_context.go`, `killsource/decode.go` en
  exception) ; mutation jouée (appel ajouté dans `replay/world_object_precision.go`) : ROUGE, retirée.
  **LK.2.4** : `keyframe_closure_cuisson_ratchet_test.go` + `testdata/keyframe_closure_cuisson.golden`
  (215 lignes, `ti=35` 53/1 368) = lignes « cuisson base » de la référence R-7 à l'octet. **Gate
  LK.2** : mutation « portée sur toute la marche » : HAUSSE de ti=35 sur 5 bobines (6 → 78, 13 → 163,
  3 → 57, 16 → 186, 12 → 180) et BAISSE 60ae07c4 2 → 1, ROUGE (total 666) ; mutation « sans les
  largeurs de la carte » : 15 lignes en BAISSE (ti=35 ×2, 37, 38 ×2, 42 ×6, 43 ×4), ROUGE ; les deux
  restaurées. Tests : grammaire 56 s, reste de `film/...` 34 s, `replaybuild`, `killcollector`,
  `archlint` verts sous `-skip "$SKIPREV"` ; vet avec et sans `research` vert. Correction du plan
  sur pièces : `TestChaqueRevisionEgaleSonGolden` (`film/revision`) rougit lui aussi entre deux
  montées (il rejoue l'empreinte de chaque couche) : ajouté à `SKIPREV` (§1.4).
- 2026-10-08 (nuit) : ÉTAPE LK.3 CLOSE (LK.3.1 à LK.3.8 `[x]`). Sorties sous `$S/lk3/`. Production :
  `Lecteur.portee` posée par la seule marche d'état complet (`consumeFullStateDefaultBlock`,
  `traverserSousLaPortee`), `fullPrecisionGate` = portée ou réglage, branche absolue d'i0 sous la
  portée (`consumeAbsoluSousLaPortee`), arrêts nommés `grammar.ArretDuLecteur` (`position_non_finie`,
  `largeur_handle_moteur_un`) portés par `EntityTrace.Arret`, `GrammaireBalayage.MoteurUnPossible`
  dérivé du film (`grammaireSousFilm`, `FilmContext.poserLaGrammaireDuFilm`), `PorteeBaseline` et
  `GrammaireEcrivainI0` retirés. **Tests unitaires** (`portee_etat_complet_test.go`) : i0 sous portée
  h = 0 = 101 bits, h = 1 = 115 bits (queue entre les 96 bits et le R(2)) ; flottant infini → arrêt
  après 99 bits, sans R(2) ; garde E-4 ; marche d'état complet (fermeture, deux arrêts au début d'i0) ;
  état par défaut `ti=40` sous la portée (feuille quaternion R(96)) ; trame média du bipède sous la
  portée = 96 + 1 bits ; remise à faux des deux portées (arrêt de lecteur compris) ; record NEW et
  DELTA hors portée (i0 = 47 bits) ; type de moteur dérivé du film. **Mutations**, chacune ROUGE puis
  retirée (`$S/lk3/mut/m*.log`) : portée posée dans `TraverseEntity` ; remise à faux oubliée après
  l'état par défaut, puis après la boucle ; R(2) lu avant la queue (108 bits au lieu de 115) ; portée
  sur la boucle sans l'état par défaut (fin 355 au lieu de 388) ; et en plus : finitude lue sans
  inverser les octets, garde E-4 désarmée, dérivation du type de moteur retirée du contexte.
  **Commandes** : `go test $G -count=1 -skip "$SKIPREV"` vert (64,7 s) ; `go test ./internal/archlint/
  -count=1` vert ; `go vet ./internal/games/halo_infinite/film/...` et `go vet -tags=research
  ./internal/games/halo_infinite/film/... ./internal/sync/killcollector/` code 0 ; `golangci-lint run
  --new-from-merge-base=origin/main` sur `grammar` et `archlint` : 0 problème ; registre des replis et
  codec des faits (`TestCodecCouvreFilmInputs`, `FuzzDecodeFilmFactsFile`) verts. **Gate LK.3**, dans
  l'ordre : (1) I-ancres : colonnes 1-9 de `m2_ancres_tete.tsv` identiques à `$S/e0` (0 ligne),
  totaux de `m2_stats_tete.tsv` hors `fermes` identiques à `$S/e0`, `m2_stats_tete.tsv` identique à
  `m2_stats_lk.tsv` à l'octet ; `m2_ancres_tete.tsv` diffère de `m2_ancres_lk.tsv` sur 75 lignes, toutes
  de `50247b26` (52 `ti=35`, 23 `ti=40`) qui s'arrêtent désormais au début d'i0 ; I-equipes : 28
  lignes identiques aux lignes `lk` de la référence et à `$S/e0`. (2) I-carte : 20 films, 8 films et
  28 films sous MPP déclaré (`-plafond-gib 4 -top 40 -mode v2 -paquets -mpp-declare`, ordre
  `$F20,$F8`) : tous les TSV identiques à `$S/e0`, `fermeture_films.tsv` colonnes 1-16 identiques,
  `fermeture_resume.md` ne diffère que par la colonne du pic mémoire. (3) I-d0 : `agg.tsv` TOTAL
  10710 10575 5399 9968 332 6955 1566 5324 4 4319 contre R-2 10710 10622 5399 9972 380 6955 1612 5324 4
  4319 : seule la ligne `50247b26` change (traversés 635 → 588, grenades lues 41 → 37, compte ≠ 4
  380 → 332, fenêtre en plus 358 → 312 ; fermés 6 inchangés) ; `diff` des lignes triées contre
  `images_cles_portee.tsv` : 3 862 lignes, réparties entre `50247b26` (ses lignes et les agrégats de
  format 20 et « tous ») et les lignes `T`/`TF` des témoins négatifs (en-tête décalé d'un bit) de
  tous les films. (4) I-ferm : `--- PASS`, 839 lignes, colonnes film/archétype/fermés/total
  identiques aux lignes `portee` de la référence ; `ti=35` 5 399 / 10 710 (base 410) ; une seule ligne
  change, le bloquant le plus fréquent de `50247b26` `ti=35` (`i58` → `i0`). (5) I-critere : REF
  125 + 124 + 120 = 369/599 = 61,60 % dans `TestKF7EFullStateLoop` et `TestRI27d1M2Critere`, avec et
  sans bouchons ; la ligne (c) vaut l'ancienne (c+d+e), 3/599. (6) `-run KeyframeClosure` vert :
  golden sans carte inchangé, cuisson régénérée à 666/1 368. **Écart avec le crochet, EXPLIQUÉ** :
  rejoué sur `$S/lk3/sans_arrets/` avec les deux arrêts neutralisés le temps de la mesure (code
  restauré, `MESURE TEMPORAIRE` absent de l'arbre), I-d0 rend `images_cles_portee.tsv` à l'octet (0
  ligne de `diff`) et I-ancres `m2_ancres_lk.tsv` à l'octet ; avec la garde E-4 seule neutralisée
  (`$S/lk3/sans_garde_e4/`), il reste une ligne. Donc : 74 records de `50247b26` s'arrêtent sur
  `largeur_handle_moteur_un` — `50247b26` et `a349fea8` n'ont pas de section d'identification
  (instrument V3, `$S/lk3/variantes_28.tsv` : 26 films à identité, tous `TypeDeMoteur` 2), le type de
  moteur n'y est pas établi, et la garde E-4 s'arme comme le plan l'écrit ; 1 record de `50247b26`
  (tranche 30, slot 672, bit 199 752) et des témoins négatifs s'arrêtent sur `position_non_finie`.
  Aucun de ces records ne fermait sous le crochet (fermés inchangés partout). Découvertes D-12 à
  D-15 (§7).
- 2026-10-08 (nuit) : ÉTAPE LK.4 CLOSE (LK.4.1 à LK.4.3 `[x]`), aucune ligne de code (un instrument
  temporaire de relevé du registre, joué puis supprimé, jamais commité). Sorties sous `$S/lk4/`.
  **LK.4.1** : résidu sous la production de LK.3 — 135 arrêts (55 i59 inchangés ; 28 i58 et 52 au
  début d'i0, tous sur `50247b26`, film sans section d'identification : 51 `largeur_handle_moteur_un`,
  1 `position_non_finie`) ; bloquant `ti=35` vide sur 21 films, arrêt sur 7 (i59 sur six, i0 sur
  `50247b26`) ; écarts de fin 5 328 / 2 801 / 2 223 (R-5 à un record près). **LK.4.2** : NOMMÉ, non
  adjugé — premier composant de largeur divergente de `60ae07c4` : i53
  `biped-malleable-property-component` (36 contre 30) ; les 163 records à +33 ont tous i53 = 36 et
  i57 = 28, les 46 à +18 ont i53 = 36 et i55, i58 divergents ; un bit constant non localisé (D-16).
  **LK.4.3** : lignes d'état à D-4 (`50247b26` toujours illisible en i22 : 37 records à n = 4 sur 668 ;
  `a349fea8` 979/984), D-5 (formats 20-21 : classe A 24 records, armes 0/16, munitions 0 ; classe B
  1 400 records, armes 55/1 184), D-6 (témoin : 226 fermés dont 222 en vrai, hasard propre 4/5 311).
  **Gate LK.4** : chaque point du §7 porte un chiffre et une commande ; aucun code de production ni de
  test modifié. Prochaine étape : LK.5 (non commencée, consigne du superviseur).
- 2026-10-08 (soir, superviseur) : lot A relu sur pièces. Gates rejoués par le superviseur sur la
  tête du lot (`31e150144`) : `go vet` du film, d'archlint et de killcollector, vet `research` : verts ;
  `go test` de grammar (-skip des gardes de révision), des paquets facts, replay, types et decfilm,
  et d'archlint : verts ; gardes de révision : seul `TestGrammarRevSuitLaGrammaire` rouge, attendu
  jusqu'à la montée de LK.6 (killsource et objectives verts). Refusion de `origin/feat/v75`
  (`bc3f28e89` : archivage du plan de l'étape 2 sous `.ai/V7.5/` et repointage des références,
  disk-hygiene, lots aj-apparence et aj-demarrage) sans conflit ; les deux renvois de ce plan vers
  le plan de l'étape 2 repointés. Accord de l'utilisateur donné d'avance pour la fusion finale, la
  recuisson et le backfill (U-5, D1.4.7). Garde E-4 sur les films sans section d'identification :
  maintenue (aucune largeur présumée ; 74 records arrêtés sur `50247b26`, format où les valeurs après
  `i22` sont fausses de toute façon). Coordination : levelup-5c corrige la lecture HORS portée de
  `consumeObjectPositionMonde` (`39510278d`, branche `feat/grammaire-arrets-vue-b-2`, non fusionnée) ;
  LK.5 pose la branche SOUS portée de la même exception en tête de fonction, pour un recollage
  trivial. Suite : lot B (LK.5 puis LK.6).
