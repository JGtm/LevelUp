# PLAN — Résidu de lecture des images-clés : entrées de slot libéré, bit des films 1.8, world-object i0 sous la portée, valeur 2 de l'emplacement désiré (2026-10-14)

> **Statut : EN COURS — GO de l'utilisateur du 2026-10-14 (« ok go »), décisions datées au §3.1.**
> Rédigé le 2026-10-14 par l'exécutant de l'étape dédiée, dans `feat/ri-ecart-108`. Il fait suite au
> plan clos `.ai/PLAN_RI_ETAT_COMPLET_IMAGES_CLES_2026-10-08.md` (LK puis 2.7.d1, fusionné dans
> `feat/v75` le 2026-10-14, `grammar-2026-10-09.3`, schéma 93), dont il reprend les instruments, les
> pièges et la discipline des révisions.
>
> **Contrat** : skill `plan-execution` (ce plan fait foi en cas de divergence) : ordre strict, une
> étape à la fois, aucune étape exécutable différée, chaque item statué `[x]` / `[~]` (référence) /
> `[!]` (justification écrite), vérification sur pièces avant de coder et avant de cocher, zéro fix
> hors périmètre (§7 Découvertes). On ne revient vers l'utilisateur (par le superviseur) que pour : un
> ARRÊT écrit dans ce plan, la fusion dans `feat/v75`, la recuisson du parc, le backfill killsource.
>
> **Pour qui** : l'agent qui exécute R1 à R5, seul, dans son worktree, sans sous-agent. Le premier
> exécutant (2026-10-14) écrit ce plan et exécute R1, puis s'arrête.
>
> **Sources (à lire avant la première étape)** :
> - `.ai/RAPPORT_ECART_108_R_VEH_4_2026-10-09.md` — entrées de slot libéré, règle Rslot, simulation ;
> - `.ai/RAPPORT_I53_I57_ECART_33_2026-10-09.md`, `.ai/RAPPORT_BIT_BUILDS_18_2026-10-09.md` — le bit de
>   moins du bloc malléable d'i53 sur les films de version majeure déclarée ≤ 0x25 ;
> - le plan clos (§2 références, §3 décisions U-0 à U-5 et E-1 à E-10, §5 gates et pièges, §7 D-1 à
>   D-39, journal ; en particulier LK.5.4.1, D-18, D-21, D-30) ;
> - `docs/adr/0037-film-intermediate-representation.md` (IR-4, IR-6, 2.7.d, 2.7.d1) ;
>   `docs/adr/0034-*.md` (D-7, D-10) ; `.ai/V7.5/killweapon/WALK_PORT_NOTES.md` §6 ;
> - `CLAUDE.md` (règles 5, 6, 7, 11, 17).
>
> Les numéros de ligne cités datent du 2026-10-14 sur `199c6984d` ; rouvrir chaque fichier avant
> d'agir (règle 4 du contrat).

---

## 0. Objet, critères de succès, périmètre

**Objet.** Quatre corrections de lecture des images-clés, dans cet ordre, sur une seule branche, puis
une clôture commune :

1. **R1 — entrées de slot libéré (Rslot).** La table d'image-clé compte une entrée par slot (0x1fff
   entrées). Un slot libéré s'écrit en une entrée de 108 bits d'archétype `0xffffffff`, que le port
   prend aujourd'hui pour la sentinelle de fin. Entre deux ancres de slots s et t, la lecture du jeu
   traverse exactement t − s − 1 entrées. La règle entre dans la fermeture d'un record, dans la
   preuve d'image-clé qui arbitre l'élection des ancres, dans `readKeyframeHeader` et les deux marches
   déterministes (`WalkKeyframeRecords`, `ChainKeyframeRecords`), et dans la fin de table.
2. **R2 — le bit des films 1.8.** Le flux des films de version majeure déclarée ≤ 0x25 n'a pas un
   bit du bloc malléable d'i53. On lit d'abord le chemin du jeu pour ces versions (`FUN_142988e98`),
   puis on applique l'option A de l'utilisateur.
3. **R3 — world-object i0 sous la portée** (LK.5.4.1 rejeté le 2026-10-08), rejoué derrière R1 et
   R2, puis rigid-body i18 derrière lui (LK.5.4.9).
4. **R4 — la valeur 2 de l'emplacement désiré** (D-30 du plan clos) : son sens établi sur les films,
   puis l'affichage web corrigé.
5. **R5 — révisions, gates complets, revue adversariale, fusion.**

**Critères de succès (mesurables).**
1. R1 : sur les 28 films, en contexte de cuisson, les records bipèdes d'image-clé fermés montent de la
   référence de R1.0 (5 397 attendus à la tête, 5 401 avant la correction « jeu d'armes refusé ») vers
   la simulation du rapport (8 242 à la base `3aa885e37`) ; aucune fermeture ne baisse sans
   adjudication écrite ; les admissions T1 ∧ T2 ne baissent pas (6 507) ; la carte des trames delta
   est identique ou chaque écart est adjugé ; chaque ancre qui change est adjugée, preuve à l'appui.
2. R2 : la règle du bit est fondée sur une lecture du jeu si `FUN_142988e98` la porte, sinon sur la
   version majeure déclarée par le film (option A, décision de l'utilisateur) ; zéro octet de
   différence sur les films de version majeure ≥ 0x26 ; fermetures de `60ae07c4` en hausse.
3. R3 : la lecture du jeu de world-object i0 sous la portée est retenue si chaque baisse est adjugée,
   sinon ARRÊT et décision de l'utilisateur ; idem pour i18.
4. R4 : le sens de `d = 2` est établi par une mesure écrite ; le web n'affirme plus « rien en main » sur
   une valeur qu'aucun lecteur du jeu n'appuie.
5. R5 : une montée de `grammar.Rev` pour le jalon ; killsource et objectives à révision constante si
   leurs sorties sont identiques ; `SchemaVersion` au-dessus de `feat/v75` si le document change ;
   `SchemaDesFaits` si la forme des faits change ; CI verte ; revue adversariale passée ; une seule
   fusion, sur accord explicite de l'utilisateur demandé par le superviseur.

**Hors périmètre** (explicite ; ce qui s'y rencontre va au §7) :
- les formats 20-21 après i22 (D-5), l'illisibilité de `50247b26` en i22 (D-4), les ~120 records de
  f24-27 encore non fermés hors des cas ci-dessus ;
- les lectures rejetées D-17 (i20 `lireViseeDActeurAncienne`) et D-23 (garde low-frequency de `ti=3`) :
  elles restent rejetées, sauf si R3 en fait une dépendance mesurée (alors : §7 et point à
  l'utilisateur) ;
- la forme en ligne du delta d'i0 (D-1) ; les P2 de la revue D1.4.6 (D-32 à D-38) ; D-39 ;
- toute cuisson du parc, tout backfill, toute écriture dans une base DuckDB ; Halo 5. **Une seule
  exception, nommée** : G-corpus (R5) lance `levelup replay-facts-export`, qui ouvre la base partagée
  EN LECTURE par `OpenReadForQuery`, et seulement après avoir vérifié qu'aucun backfill ni aucune
  recuisson ne tourne (`Get-CimInstance Win32_Process` filtré sur `backfill` et sur les binaires de
  cuisson) ; sinon on attend.

## 1. Base, branche, worktree, coordination

### 1.1 Branche et worktree
- Branche **`feat/ri-ecart-108`**, worktree DÉDIÉ `C:/Users/Guillaume/Downloads/Scripts/LevelUp-wt-ecart108`,
  tête de départ **`199c6984d`** (les trois rapports de recherche et la fusion de `feat/v75`
  `940e27b3f`, qui contient le plan clos, schéma 93, `grammar-2026-10-09.3`). Préfixe `feat/` : la CI
  se déclenche au push. Un seul exécutant.
- Jonctions vers le checkout principal DÉJÀ EN PLACE : `data/cache/film_chunks`,
  `data/cache/film_manifests`, `apps/web/node_modules`. Rien n'est purgé dedans (le checkout principal
  sert à d'autres sessions et fait tourner le serveur :8000). Retrait par `(Get-Item <jonction>).Delete()`
  AVANT tout `git worktree remove` (jamais `--force`).
- `data/cache/film_facts` du worktree : dossier RÉEL (jamais une jonction), renommé avant chaque passe
  de `replay-equiv` (§5, pièges).
- Cache Go : `GOCACHE='C:\Users\Guillaume\AppData\Local\go-build-ri'`,
  `GOLANGCI_LINT_CACHE='C:\Users\Guillaume\AppData\Local\golangci-ri'` ; un seul processus `go` à la
  fois ; jamais `go build ./...` ; tests en avant-plan, découpés sous 10 minutes.
- Push : `feat/ri-ecart-108` seule ; aucune fusion dans `feat/v75` (geste du superviseur, sur accord).

### 1.2 Coordination
- Avant chaque étape : `git -C $WT fetch` ; si `origin/feat/v75` a bougé, `git merge origin/feat/v75`
  (en conflit, `feat/v75` a raison), puis rejouer les références de R1.0 touchées par la fusion. Un
  commit de `feat/v75` qui toucherait `film/` est signalé au superviseur avant la fusion.
- Le checkout principal n'est touché par aucune commande git ; le serveur :8000 n'est pas arrêté.

### 1.3 Variables communes des commandes (bash, depuis `$WT/apps/go-api`)
```bash
export GOCACHE='C:\Users\Guillaume\AppData\Local\go-build-ri' GOLANGCI_LINT_CACHE='C:\Users\Guillaume\AppData\Local\golangci-ri' PATH="/c/msys64/ucrt64/bin:$PATH" CGO_ENABLED=1
WT=/c/Users/Guillaume/Downloads/Scripts/LevelUp-wt-ecart108
R=C:/Users/Guillaume/Downloads/Scripts/LevelUp/data/cache/film_chunks
SP=C:/Users/GUILLA~1/AppData/Local/Temp/claude/c--Users-Guillaume-Downloads-Scripts-LevelUp/2efb0786-9fba-4f95-8fd0-464ee47b91d7/scratchpad/ri
RES=$SP/res                      # sorties de ce plan, un sous-dossier par étape : $RES/r0, $RES/r1.2, ...
REF=$SP/d1                       # références du plan clos (agg.sh, ref_corrigee/, m2/)
O108=$SP/ecart108                # sorties du rapport R-VEH-4 (final/, table_x2.awk, par_film.awk)
G=./internal/games/halo_infinite/film/internal/grammar/
FILMS="000d5950,01e1f945,084a804d,111fa685,11de8353,1c4c63c2,50247b26,51101d1d,53ce4390,60ae07c4,64e8adfa,696a9d7c,7344d24f,9f57c612,a349fea8,a521164d,bcb6d393,d9781168,e5adf7b2,fb1a1a72,0797ce72,396cfc92,4f77afc1,51ebbc0f,bf15f7ab,bfecd02b,c75f33b8,f75e7053"
CARTES="0797ce72=Live Fire;396cfc92=Illusion;4f77afc1=Flood Gulch;51ebbc0f=Banished Narrows;bf15f7ab=Perilous;bfecd02b=Snowbound;c75f33b8=Curfew;f75e7053=Solitude - Ranked"
F20="bcb6d393,fb1a1a72,d9781168,c75f33b8,bf15f7ab,51ebbc0f,084a804d,0797ce72,111fa685,e5adf7b2,60ae07c4,a349fea8,a521164d,11de8353,50247b26,bfecd02b,4f77afc1,396cfc92,f75e7053,1c4c63c2"
F8="000d5950,01e1f945,51101d1d,53ce4390,64e8adfa,696a9d7c,7344d24f,9f57c612"
SKIPREV='TestGrammarRevSuitLaGrammaire|TestChroniqueCouvreLaRevisionCourante|TestKillsourceRevSuitLaSortie|TestChroniqueDeKillsourceCouvreLaRevisionCourante|TestObjectivesRevSuitLaSortie|TestChroniqueDObjectivesCouvreLaRevisionCourante|TestChaqueRevisionEgaleSonGolden'
```
**Instruments nommés** (les mêmes que le plan clos, §1.4, au worktree près ; un dossier de sortie
neuf par passe) :
- **I-d0** : `RI27D1_RECORDS=60ae07c4,50247b26 RI27C_FILMS="$FILMS" RI27C_RACINE=$R RI27C_OUT=$RES/<e> RI27C_CARTES="$CARTES" go test -tags=research -count=1 -run '^TestRI27d0ImagesCles$' -timeout 9m -v $G > $RES/<e>/images_cles.log 2>&1`,
  puis `bash $REF/agg.sh $RES/<e>/images_cles.tsv > $RES/<e>/agg.tsv` ; admissions :
  `awk -F'\t' '$1=="T" && $3 ~ /^canal_/' $RES/<e>/images_cles.tsv | awk -F'\t' '{c[$3]+=$NF} END{for(k in c) print k"\t"c[k]}' | sort`.
- **I-ferm** : `RI27C_FILMS="$FILMS" RI27C_RACINE=$R RI27C_OUT=$RES/<e> RI27C_CARTES="$CARTES" go test -tags=research -count=1 -run '^TestRI27d1FermetureCorpus$' -timeout 9m -v $G > $RES/<e>/fermeture.log 2>&1` ;
  contrôle : `--- PASS` (pas `--- SKIP`) et `fermeture_corpus.tsv` présent.
- **I-ancres**, **I-equipes** : `RI27C_FILMS="$FILMS" RI27C_CARTES="$CARTES" RI27C_RACINE=$R RI27C_OUT=$RES/<e> go test -tags=research -count=1 -run '^TestRI27d1M2Ancres$' -timeout 580s -v $G` (idem `^TestRI27d1M2Equipes$`) ;
  comparaison : colonnes 1-9 des lignes `A`/`E` de `m2_ancres_tete.tsv`, fichier `m2_stats_tete.tsv`,
  `m2_equipes.tsv`.
- **I-e108** (instrument du rapport, `TestEcart108`, simulation Rslot et témoin décalé d'un bit) :
  `RI27C_FILMS="$FILMS" RI27C_RACINE=$R RI27C_OUT=$RES/<e> RI27C_CARTES="$CARTES" go test -tags=research -count=1 -run '^TestEcart108$' -timeout 30m -v $G > $RES/<e>/e108.log 2>&1`,
  puis `awk -F'\t' -f $O108/table_x2.awk $RES/<e>/ecart108.tsv`.
- **I-carte** : `go build -tags=research -o $RES/<e>/cmd_fermeture.exe ./internal/games/halo_infinite/film/research/cmd_fermeture`, puis
  `$RES/<e>/cmd_fermeture.exe -racine $R -films $F20 -sortie $RES/<e>/carte -plafond-gib 4 -top 40 -mode v2 -paquets`,
  idem `-films $F8 -sortie $RES/<e>/carte8`, et `-films $F20,$F8 -mpp-declare -sortie $RES/<e>/carte_mpp`
  (mêmes drapeaux) ; comparaison : `cmp` de chaque TSV, `fermeture_films.tsv` par `cut -f1-16`.
- **G-equiv** : binaire `go build -o $RES/bin/replay-equiv-<e>.exe ./cmd/replay-equiv` ; passe :
  `data/cache/film_facts` du worktree renommé avant, puis `<exe> -repo-root C:/Users/Guillaume/Downloads/Scripts/LevelUp-wt-ecart108 -out-dir $RES/<e>/equiv > $RES/<e>/equiv.log 2>&1`
  (20 films, `depuis_les_faits=false` vérifié dans le journal) ; comparaison base/lot : `diff` des
  TSV d'étapes par film (`$RES/<e>/equiv/<film>.tsv`).
- **killsource** : binaire `go build -o $RES/bin/killsource-<e>.exe ./cmd/killsource` ; passe : `json <film> -carte <carte> -cache C:/Users/Guillaume/Downloads/Scripts/LevelUp-wt-ecart108/data/cache`
  sur les 19 témoins de `config/replay_corpus.toml` (liste et cartes de `$SP/d1y/ks.sh`), sorties
  `$RES/<e>/ks/<film>.json` ; comparaison : `cmp`, puis `$SP/d1y/comparer_ks.sh` adapté au dossier.
- **Gardes de révision passées sous `-skip "$SKIPREV"`** entre deux montées (R1 à R4) ; elles
  rougissent dès qu'une source non-test de `grammar` change : c'est attendu, et c'est pourquoi aucun
  push n'a lieu avant la montée de R5 (E-3).

## 2. Références

### 2.1 Mesures du rapport R-VEH-4 (base `3aa885e37`, sorties `$O108/final/`)

| Réf. | Chiffre | Fichier |
|---|---|---|
| S-1 | Records bipèdes bornés 10 710, fermés 5 401 ; écart de fin : multiple de −108 2 841, autre en deçà 896, dépasse 1 520, arrêt 52 | `agg_bipedes.txt` |
| S-2 | Blocs des écarts −108 : 2 838 records dont les k blocs sont TOUS `id = arch = mot = 0xffffffff, R(4) = 0, R(8) = 0` (263 224 blocs) et k = slot suivant − slot − 1 ; 1 record de `a521164d` (f21) et 2 de `50247b26` (f20) hors de cette forme | `verif_blocs.txt`, `blocs.txt` |
| S-3 | Simulation, ancres inchangées : bipèdes fermés 5 401 → 8 242 (Rslot), 8 239 (R108s) ; admis T1 ∧ T2 6 507 inchangés ; tous archétypes bornés (460 722) 240 470 → 267 811 ; records fermés perdus 44, tous `ti=37` (fermés avec un écart de slots > 0) | `table_x2.txt` |
| S-4 | Témoin décalé d'un bit : bipèdes 223 / 3 hasard → 325 / 4 ; tous 13 440 / 10 042 → 14 210 / 10 494 (dont +182 sur `ti=42`, +128 `ti=38`, +148 `ti=43`) | `table_x2.txt` |
| S-5 | Hausses hors bipèdes (Rslot) : `ti=10` 14 265 → 30 594, `ti=42` 3 125 → 6 810, `ti=4` 383 → 779, `ti=37` 195 → 469, `ti=11` 2 221 → 2 621, `ti=21` 1 315 → 1 739, `ti=29` 713 → 913, `ti=13` 13 625 → 13 984 (100 %) | `table_x2.txt` |
| S-6 | Preuve d'image-clé, indicateur (saut des entrées libérées puis en-tête de slot supérieur, ancres inchangées) : 215 161 → 236 374 records prouvés | `table_x2.txt` (lignes `prouve_*`) |
| S-7 | Queue de table : sur 162 paquets, lire les `0x1ffe − slot` entrées restantes depuis la fin du dernier record ne rencontre que des entrées libérées et finit à moins d'un octet de la fin du payload ; 688 autres paquets : le dernier record est mal lu | `ecart108_residus.tsv` (lignes `H`), `queue_table.txt` |

### 2.2 Lectures Ghidra (HaloInfinite.exe, lecture seule ; relues le 2026-10-14 pour R1.1, sorties `$RES/g/`)

| Fait | Adresse |
|---|---|
| En-tête d'entrée : R(32) → +0, R(32) → +4 (archétype), R(32) → +0xc, R(4) → +8 (`FUN_142e29cf8`), R(8) → +9 = **108 bits** ; corps seulement si `puVar12[1] != 0xffffffff` ; boucle de `*param_2` à `param_2[1]` par pas de 200 octets (`puVar12 + 0x32`), bornée par `DAT_144dbfc90` ; le flux ne porte aucun compte | `FUN_142e2bfd0` |
| Le tableau compte **0x1fff entrées** | `FUN_1408be074` (`FUN_1411b2400(param_1, 0x1fff)`) |
| Slot libéré : `+0 = +4 = +0xc = 0xffffffff`, tampons +0x98/+0xb8 libérés | `FUN_1408f1948` |
| L'écrivain écrit toutes les entrées avec le même en-tête, le corps si l'archétype n'est pas `0xffffffff` | `FUN_142e2d08c` (rapport, §1) |
| Le tableau vivant est indexé par slot, `(eid & 0x3fffffff) * 200 + base` | `FUN_1405d5dbc`, `FUN_1406cb5f0` (rapport, §1) |

## 3. Décisions

### 3.1 Décisions de l'utilisateur (datées, fermes : règle 10 du contrat)
- **U-1 (2026-10-09)** — ouvrir la piste des composants d'image-clé non décodés, en réponse au
  diagnostic : « Oui je veux bien que tu ouvres cette piste, c'est tout de meme important de savoir
  décoder ces composants ».
- **U-2 (2026-10-09)** — bit des films 1.8 : **« Donc l'option A, à mon sens »**. Règle fondée sur la
  version majeure DÉCLARÉE par le film (`chunk_00+0`). Principe : le film est autodescriptif et le jeu
  actuel lit tous les films ; un test de version majeure dans l'exécutable est un AIGUILLAGE vers le
  chemin des anciens formats, pas un refus. Il faut donc LIRE d'abord `FUN_142988e98` (cible de
  `FUN_1428e219c` quand la majeure diffère de 0x29) pour remplacer le seuil mesuré par la lecture du
  jeu, si on la trouve (R2.1).
- **U-3 (2026-10-09)** — valeur 2 de l'emplacement désiré, que le web lit « rien en main » : « les
  joueurs ont sans doute aucune arme en main avant que le jeu leur en attribue, normalement c'est
  quasi impossible d'être mains nues ». Hypothèse À VÉRIFIER (R4.1), pas une règle.
- **U-4 (2026-10-14)** — « ok go » pour cette étape (ce plan).
- **U-5 (consigne de session)** — les sous-agents tournent sur Sonnet par défaut ; ce plan n'en lance
  aucun (revue adversariale de R5 comprise : un relecteur au contexte frais, lancé par le superviseur).
- **Héritées du plan clos, toujours fermes** : U-1 amendée du 2026-10-09 (admission T1 ∧ T2) ; règle
  des largeurs lues (toute largeur affirmée est lue dans le jeu, adresse citée) ; corrections
  générales, jamais par film ; une seule fusion à la fin, sur accord explicite.

- **U-6 (2026-10-15)** — bit des films 1.8, après l'arrêt de R2.3 et les trois options présentées
  (a : garder ≤ 0x25 ; b : la seule valeur déclarée 0x25 ; c : sortir les formats 20-21) : **« Ok avec
  toi, option b »** (relayée par le superviseur). La règle du drapeau final malléable ne vaut que pour
  les films qui DÉCLARENT exactement la version majeure 0x25 ; toutes les autres versions gardent la
  grammaire de l'exécutable, et les formats 20-21 restent intacts. Elle amende U-2 sur le seuil.

- **U-7 (2026-10-15)** — world-object i0 sous la portée (R3.1), après l'arrêt de R3.2 et les trois
  options présentées (a : retenir la lecture du jeu en acceptant les baisses ; b : la laisser rejetée ;
  c : la retenir hors des formats 20-21) : **« Option a »** (relayée par le superviseur). La lecture du
  jeu est retenue ; les cinq baisses mesurées en R3.2 sont acceptées par cette décision (journal du
  2026-10-15).
- **U-8 (2026-10-15)** — surcoût de cuisson mesuré par G-perf (R5.6 : +11,0 % sur `e5adf7b2`, +11,1 % sur
  `11de8353`, au-dessus du seuil de 10 %), après les deux options présentées (a : accepter le coût ;
  b : optimiser d'abord) : **« option a »** (relayée par le superviseur). Le coût est accepté ; une
  piste d'optimisation future est au registre des reports.

### 3.2 Décisions d'exécution (à l'exécutant, justifiées ; une objection de l'utilisateur les rouvre)
- **E-1 — Un seul lecteur d'entrée.** La forme de l'entrée (108 bits, corps si l'archétype n'est pas
  `0xffffffff`) est écrite UNE fois (`keyframe_entrees.go`) ; la fermeture, la preuve d'image-clé,
  les deux marches déterministes et la fin de table la consomment. Le jeu ne teste QUE l'archétype :
  l'entrée libérée se reconnaît à lui seul ; la forme de l'écrivain (`id` et mot +0xc à
  `0xffffffff`, R(4) = R(8) = 0) est documentée, pas exigée. Le slot d'une entrée intercalée n'est pas
  vérifié (le lecteur du jeu ne le vérifie pas) : la preuve est l'atterrissage exact.
- **E-2 — Ordre de R1 par surface d'effet** : la fermeture (R1.2) ne touche pas l'élection des
  ancres ; la preuve d'image-clé (R1.3) la touche ; les marches déterministes et la fin de table
  (R1.4) ne touchent ni l'une ni l'autre hors des témoins de chaînage. Chaque sous-pas est un commit
  mesuré à part, pour que chaque effet s'attribue à sa cause.
- **E-3 — Montée des révisions en R5 seulement** (un seul jalon, comme LK.6 et D1.4 du plan clos) :
  commits intermédiaires sans montée, gardes de révision sous `-skip "$SKIPREV"` en local, AUCUN
  push avant R5. `grammar.Rev` monte une fois (rang au-dessus de celui de `feat/v75` au moment de la
  montée) ; killsource et objectives restent à révision constante, avec un complément daté, si
  `cmd/killsource json` (19 témoins) et l'étape objectives de `replay-equiv` (20 films) sont
  identiques à la base ; `SchemaVersion` monte au-dessus de `feat/v75` si et seulement si le document
  publié change ; `SchemaDesFaits` monte si la forme des faits change.
- **E-4 — Adjudication d'une ancre qui change (R1.3, R3).** Règle écrite AVANT la mesure : une ancre
  qui change entre la base et le sous-pas est ADJUGÉE quand l'instrument I-adj (R1.3.2) montre, dans
  la région changée (de la dernière ancre commune qui la précède à la première qui la suit), que la
  suite NOUVELLE ferme sous Rslot au moins autant de paires d'ancres consécutives que l'ancienne, ET
  qu'aucune ancre nouvelle n'est contredite par un record prouvé. Une région où la suite ancienne
  fermait davantage est une BAISSE ; si une seule baisse ne s'adjuge pas ainsi, ARRÊT (l'exécutant
  rapporte, l'utilisateur décide).
- **E-5 — Adjudication d'une fermeture perdue (R1.2).** Un record fermé à la base qui ne l'est plus
  sous Rslot est adjugé quand sa fermeture de base atterrissait sur l'ancre suivante avec un écart de
  slots strictement positif (impossible dans une table indexée par slot, où chaque entrée fait au
  moins 108 bits) ; toute autre perte est une baisse non adjugée : ARRÊT.
- **E-6 — Pas d'élargissement de la forme des faits** (E-8 du plan clos) : `SchemaDesFaits` ne monte
  que si la survie depuis les faits l'exige (R5).

## 4. Étapes

Une étape commence quand la précédente est close : items statués, gate passé et consigné au journal
(§8), plan et thought_log à jour, commit. Commits préfixés `ri-res(<item>): …`, chemins explicites
(jamais `git add -A` ni `git add .`), pied `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`.

### Étape R1 — Entrées de slot libéré (Rslot)

- [x] R1.0 Base : `git fetch` ; fusion de `origin/feat/v75` si elle a bougé (§1.2) ; références à la
      tête (code de production inchangé) → `$RES/r0` : I-d0 (TOTAL et admissions), I-ferm, I-ancres,
      I-equipes, I-e108, I-carte (20, 8, MPP 28) ; binaires de base `replay-equiv-r0.exe` et
      `killsource-r0.exe` ; G-equiv et killsource de base → `$RES/r0/equiv`, `$RES/r0/ks`.
      Contrôle : I-e108 à la tête reproduit S-3 à la correction « jeu d'armes refusé » près (records
      bipèdes arrêtés sur i42, journal du plan clos, `ti=35` 5 401 → 5 397) ; tout autre écart est
      expliqué avant R1.2.
      *Fait (2026-10-14)* : fusion de `origin/feat/v75` `76c0dce35` sans conflit (`da3da6727`, aucun
      fichier sous `film/`). Références `$RES/r0` : I-d0 TOTAL `10710 10198 5397 9968 332 6954 1180 5323 3
      4320`, admis 6 507 ; I-ferm 837 lignes ; I-ancres 477 636 lignes ; I-e108 : bipèdes 5 397 → 8 237
      sous Rslot (S-3 : 5 401 → 8 242 ; écart = les quatre fermetures et le record perdus par la
      correction « jeu d'armes refusé » de la revue D1.4.6) ; I-carte (20, 8, MPP) ; G-equiv 20 films
      (`depuis_les_faits=false` sur les 20) et killsource 19 témoins avec les binaires `r0`.
- [x] R1.1 Lectures Ghidra relues sur pièces (§2.2) : `FUN_142e2bfd0` (en-tête 108 bits, condition du
      corps, pas de 200 octets, borne), `FUN_142e29cf8` (R(4)), `FUN_1408be074` (0x1fff),
      `FUN_1408f1948` (forme libérée) → `$RES/g/`. Aucune largeur codée sans adresse.
      *Fait* : décompilés (`dec_142e2bfd0.c`, `dec_142e29cf8.c`, `dec_1408be074.c`, `dec_1408f1948.c`) ;
      conformes au §2.2.
- [x] R1.2 **Fermeture sous Rslot.**
  - [x] R1.2.1 `keyframe_entrees.go` : la forme de l'entrée (E-1) — `lireEntreeDImageCle` (libérée /
        record / illisible) et `lireLesEntrees(pay, pos, n, reg, ctx)` (n entrées comme le jeu :
        108 bits pour une libérée, état complet par `WalkKeyframeFullState` sans observation pour un
        record, faux sur une entrée illisible ou un record désynchronisé). Tests unitaires synthétiques
        (entrées libérées seules, record intercalé, archétype hors table, débordement du payload).
  - [x] R1.2.2 `keyframeBorne` porte le slot de l'ancre suivante ; la fermeture d'un record devient
        « la traversée de son état complet, puis la lecture des `slotSuivant − slot − 1` entrées
        intercalées, atterrissent exactement sur l'ancre suivante » ; UNE fonction
        (`fermeSurLAncreSuivante`) pour `preuveDeLEtatComplet` et les compteurs `KeyClosed` des deux
        balayages (`navpoint_radial_scan.go`, `objective_scan.go`) — trois copies de
        `tr.EndBit == b.Want` aujourd'hui, règle 6 : helper ET garde-rail qui interdit l'ancien
        littéral hors du helper.
  - [x] R1.2.3 Mesure → `$RES/r1.2` : I-ferm, I-d0, I-ancres, I-equipes, I-carte, I-e108 ; goldens de
        fermeture (`-run KeyframeClosure`). Pertes adjugées par E-5, liste écrite au journal.
  - [x] R1.2.4 Ratchets régénérés (`keyframe_closure.golden`, `keyframe_closure_cuisson.golden`) avec
        leur ligne d'historique (hausses, pertes adjugées).
  - Gate R1.2 : I-ancres colonnes 1-9 et `m2_stats_tete.tsv` identiques à `$RES/r0` hors la colonne
    `fermes` ; I-equipes identique ; I-carte identique ; I-ferm : aucune ligne (archétype × film) en
    baisse hors pertes adjugées E-5 ; bipèdes fermés à la simulation S-3 près (écart expliqué) ;
    admissions `canal_admis` ≥ r0 ; témoin décalé d'un bit publié par archétype et comparé à S-4 (un
    écart à la simulation est expliqué) ; G-unit (`-skip "$SKIPREV"`) du paquet `grammar`, archlint.
  - *Gate R1.2 PASSÉ (2026-10-14, `$RES/r1.2`)* : I-ancres colonnes 1-9 identiques (`cmp`), stats
    identiques hors `fermes` ; équipes identiques ; carte identique (20, 8, MPP, `cmp`) ; I-ferm
    (`cmp_ferm.txt`) : 283 lignes en hausse, DEUX en baisse, `ti=37` `396cfc92` 16 → 1 et `696a9d7c`
    3 → 2, soit 44 records perdus sur 10 films (`fermes_perdus.txt`), TOUS `ti=37`, TOUS fermés en
    base avec un écart de slots > 0 — ADJUGÉS par E-5 (le témoin décalé d'un bit en fermait 24, dont 18
    par hasard, en base) ; par archétype exactement la simulation S-3/S-5 (bipèdes 5 397 → 8 237, tous
    240 466 → 267 806) ; admissions 6 507 inchangées ; témoin sous Rslot : bipèdes 316 dont 3 par
    hasard (S-4) ; goldens de fermeture régénérés, aucune baisse sur les sept bobines ; G-unit et
    archlint verts.
- [x] R1.3 **Preuve d'image-clé (élection des ancres) sous Rslot.**
  - [x] R1.3.1 `PreuveDImageCle.prouve` : après la traversée, lire les entrées libérées qui suivent
        (k ≥ 0), puis exiger un en-tête de record de slot ÉGAL à `slot + 1 + k` (un écart de slots
        sans entrée intercalée est impossible dans une table indexée par slot). Tests unitaires.
  - [x] R1.3.2 Instrument I-adj (`res_r1_ancres_research_test.go`, tag `research`) : à partir des
        sorties I-ancres de la base et du sous-pas, par paquet, les régions changées, et pour chacune
        le nombre de paires d'ancres consécutives qui ferment sous Rslot dans la suite ancienne et
        dans la nouvelle, et les records prouvés qui contredisent une ancre de chaque suite (E-4).
  - [x] R1.3.3 Mesure → `$RES/r1.3` : I-ancres, I-equipes, I-carte, I-ferm, I-d0, I-e108, I-adj.
        Chaque ancre changée adjugée par E-4 (tableau par film au journal) ; chaque ligne d'I-ferm, de
        la carte ou des équipes en baisse rattachée à une région adjugée, sinon ARRÊT.
  - [x] R1.3.4 Ratchets régénérés si la fermeture bouge par l'élection (ligne d'historique).
  - Gate R1.3 : toutes les ancres changées adjugées (E-4) ; aucune baisse non adjugée (I-ferm,
    carte, équipes, admissions) ; G-unit, archlint.
  - *Gate R1.3 PASSÉ (2026-10-14, `$RES/r1.3`)* : un seul film change d'élection, `64e8adfa`, en DEUX
    régions (`adj.txt`) : morceau 29 paquet 1, l'ancre élue slot 1536 `ti=0` (bit 431 590) cède à 1547,
    1584, 1587 (`ti=10`), fermées avant 0 → après 4 ; morceau 36 paquet 1, la même fausse ancre 1536
    `ti=0` (bit 444 557) cède à sept records (1547 … 1648, `ti=10/12/42`), fermées 0 → 7 ; aucune ancre
    nouvelle contredite par un record prouvé : les deux ADJUGÉES (E-4) — c'est la fausse ancre de slot
    1536 `ti=0` que le lot D-fix décrivait déjà sur `bcb6d393`, désormais réfutée (réfutations 1 → 3,
    élections 1 507 → 1 515, ancres +8). Preuves jouées sur toutes les ancres : en hausse sur les 28
    films (par ex. `084a804d` 13 905 → 16 852). I-ferm : seule la ligne `64e8adfa` `ti=0` (0/2)
    disparaît, `ti=10` 871 → 879/879 et `ti=42` 264 → 267 ; aucune baisse. Équipes, carte (20, 8, MPP),
    I-d0 (admis 6 507) identiques à r1.2 ; goldens de fermeture inchangés (régénérés à l'identique) ;
    G-unit, archlint verts. Test `TestLaPreuveLitLesSlotsLiberesQuiSuivent` (1 133 records prouvés
    par-dessus des slots libérés sur les bobines ; deux mutations rouges). Écart au plan (décision
    d'exécution) : I-adj est un script sur les sorties d'I-ancres (`$RES/adj.sh`), qui portent déjà la
    fermeture et la preuve de chaque ancre, au lieu d'un fichier `research` qui les recalculerait.
- [x] R1.4 **Marches déterministes et fin de table.**
  - [x] R1.4.1 `WalkKeyframeRecords` lit la table entrée par entrée (E-1) : une entrée libérée avance
        de 108 bits et n'est pas rendue comme record ; l'index de l'entrée est son slot (un record
        dont le slot diffère de son index arrête la marche, `KeyframeStopSlot`) ; la fin de table est
        l'entrée 0x1ffe (constante lue, `FUN_1408be074`), puis la fin du payload. Plus d'arrêt « fin de
        table » sur un identifiant `0xffffffff`.
  - [x] R1.4.2 `ChainKeyframeRecords` traverse les entrées libérées sans les compter dans le budget
        de records intercalés, et les compte à part (`SkippedLiberees`).
  - [x] R1.4.3 Témoin de chaînage des deux balayages (`KeyChained`, `Reads[].Chained`) : la position
        d'arrivée porte une entrée lisible (record ou entrée libérée) ; mesuré par G-equiv.
  - [x] R1.4.4 Fin de table de `kfRecherche` (traînée de 2 048 positions de sentinelle) : MESURER sur
        les 28 films combien de recherches s'arrêtent sur elle (instrument temporaire) ; la doc dit
        ce que le jeu fait (fin de table = entrée 0x1ffe ; une suite d'entrées libérées ne forme
        jamais cette traînée, 12 bits nuls par entrée) et ce que la traînée reste (garde de coût du
        balayeur) ; si elle n'est jamais atteinte sur les films, décision écrite au journal (garder
        comme garde de coût couverte par ses tests synthétiques, ou retirer si plus rien ne
        l'exerce — règle 7).
  - [x] R1.4.5 Dernier record d'un paquet : sa fermeture par la queue de table (lire les
        `0x1ffe − slot` entrées qui le suivent ; fermé si la lecture finit dans le dernier octet du
        payload, S-7). Mesuré à part (I-d0, I-ferm inchangé car il ne compte que les bornés,
        admissions, G-equiv) ; retenu seulement sans baisse.
  - [x] R1.4.6 Tests unitaires et mutations (chacune rougit un test) : entrée libérée prise pour une
        fin ; index ≠ slot accepté ; budget de chaînage compté sur les libérées ; queue de table
        acceptée à plus d'un octet de la fin.
  - Gate R1.4 : I-ancres, I-equipes, I-carte identiques à `$RES/r1.3` ; I-ferm identique à
    `$RES/r1.3` ; I-d0 : seules les preuves des derniers records changent (R1.4.5) ; G-unit, archlint.
  - *Gate R1.4 PASSÉ (2026-10-14, `$RES/r1.4a` puis `$RES/r1.4b`)*. R1.4.1 à R1.4.4 (`381539cf7`) :
    I-ancres, statistiques, équipes, I-ferm, I-d0 et carte (20, 8, MPP) IDENTIQUES à `$RES/r1.3`.
    R1.4.4 mesuré (`$RES/r1.4/fin_de_table.tsv`, instrument temporaire non commité) : sur les 28 films,
    aucun paquet d'image-clé n'atteint la traînée de 2 048 positions (plus longue suite de 1 : 1 152
    bits) ; DÉCISION : retrait (règle 7) de la fin de table par sentinelles de `kfRecherche` (`fin`,
    `traine`, `kfFinDeTable`, `sentinelles`), de la tête du motif d'ancre, et, sans autre appelant, de
    `source.Motif64.Tete/AvecTete` et `source.SuiteDeUns` ; deux tests de fin de table retirés, un test
    « les mots 0xffffffff n'arrêtent pas la recherche » ajouté (rouge sur l'ancien code) ; `source.Rev`
    constant (`source-2026-09-16.2` : la lecture des octets ne change pas), empreinte régénérée par sa
    porte. R1.4.3 : le témoin faible de chaînage reconnaît un slot libéré sous la forme EXACTE de
    l'écrivain (trois mots `0xffffffff`), pas sur le seul archétype : une position décalée de −32 à +32
    bits dans un slot libéré porte un archétype `0xffffffff` (mesuré par le test, rouge sous la règle
    du seul archétype) — précision de E-1 pour les témoins ; la lecture entre deux ancres reste celle du
    jeu (archétype seul, l'atterrissage exact fait la preuve). R1.4.2 : le champ existant
    `SkippedSansArchetype` compte les entrées sans archétype (le nom du plan, `SkippedLiberees`, n'est
    pas créé). R1.4.5 (`$RES/r1.4b`) : 164 derniers records de paquet fermés par la queue de table sur
    918 (S-7 : 162 paquets), `ti=10` 113, `ti=42` 21, `ti=43` 13, `ti=38` 7, `ti=37` 4, `ti=11` 4, `ti=47`
    1, `ti=13` 1, aucun bipède ; témoin décalé d'un bit : 9 sur 918 (`queue_temoin.tsv`) ; I-ferm (bornés
    seuls), ancres colonnes 1-9, équipes et I-d0 (admis 6 507) identiques ; aucune perte. Tests :
    `keyframe_entrees_test.go` (marche, chaînage, chaînage faible, queue de table), mutations rouges
    (libérée prise pour une fin, index ≠ slot accepté, budget compté sur les libérées, queue acceptée à
    plus d'un octet, chaînage faible sur le seul en-tête de record) ; G-unit (`grammar`, `source`),
    archlint, vet `research` verts.
- [x] R1.5 **Gate de R1 (cumulé, contre `$RES/r0`)** → `$RES/r1` :
  - I-ferm (`ti=35` et tous archétypes), I-ancres, I-equipes, I-carte, I-d0 (admissions T1 ∧ T2),
    I-e108 : chaque écart avec r0 rattaché à R1.2, R1.3 ou R1.4 et adjugé ;
  - G-equiv, 20 films, faits mis de côté (`depuis_les_faits=false` vérifié), binaire r0 contre
    binaire R1 : étapes divergentes listées par film et adjugées sur pièces (artefacts) ;
  - killsource, 19 témoins, r0 contre R1 : identique, sinon chaque mort qui change adjugée ;
  - `go test ./internal/games/halo_infinite/film/... ./internal/replaybuild/... ./internal/sync/killcollector/... -count=1 -skip "$SKIPREV"` ;
    `go test ./internal/archlint/ -count=1` ; `go vet ./internal/games/halo_infinite/film/...` et
    `go vet -tags=research ./internal/games/halo_infinite/film/...` ; golangci-lint sur les paquets
    touchés (`--new-from-merge-base=origin/feat/v75`) ;
  - plan (cases, journal), entrée `.ai/thought_log.md` (statut En cours), commit. Pas de push (E-3).
  - *Gate R1 PASSÉ (2026-10-14, `$RES/r1`, tête `faf0cd345`)* :
    - I-ferm contre r0 (`cmp_ferm.txt`) : 283 lignes en hausse ; baisses = les 44 records `ti=37` de
      R1.2 (E-5 : `396cfc92` 16 → 1, `696a9d7c` 3 → 2) et la ligne `64e8adfa` `ti=0` (0/2) disparue
      avec la fausse ancre 1536 de R1.3 (E-4) ; rien d'autre. Bipèdes fermés 5 397 → 8 237 sur 10 710 ;
      tous archétypes bornés 240 466 → 267 817 (460 730 bornés, +8 par l'élection de `64e8adfa`) ; preuve
      d'image-clé jouée sur toutes les ancres 215 161 → 235 993.
    - I-ancres : seules les deux régions de `64e8adfa` changent (`adj.sh r0 r1`, ADJUGÉES) ; I-equipes
      et I-carte (20, 8, MPP) identiques à r0 ; I-d0 : admis T1 ∧ T2 6 507 = r0, TOTAL
      `10710 10198 8237 9968 332 6954 1180 8120 3 6533` ; témoin décalé d'un bit, bipèdes 316 dont 3 par
      hasard.
    - killsource, 19 témoins : sorties IDENTIQUES à l'octet à r0 (`cmp`).
    - G-equiv, 20 films, 40 cuissons à `depuis_les_faits=false` (`etapes_divergentes.tsv`) :
      `zoneKeyReads` de `01e1f945` (509 → 530), `696a9d7c` (709 → 710), `7344d24f` (761 → 762) — des
      records `ti=13` désormais fermés, document publié inchangé sur ces trois films ; `64e8adfa` :
      `artifact` (3 243 461 → 3 243 635 octets), `carrierMarks`, `movementStates.stats`, `pads`.
      Adjugé sur pièces (artefacts r0 et r1 de `replay-build --facts`, `$RES/r1/art/`) : UNE arme au sol
      de plus (`groundWeapons`, famille `3e070217`, lâchée par l'entité 516 à 684, vue jusqu'à 694,
      chargeur 13 / réserve 40 — le record `ti=42` du slot 1611 que la fausse ancre effaçait) et les
      compteurs de la marche (élections 1 507 → 1 515, réfutations 1 → 3, records 13 749 → 13 757,
      liaisons oubliées 250 → 248) ; rien d'autre ne bouge. Le document publié change :
      `SchemaVersion` montera en R5 (E-3).
    - Tests : `grammar/...`, `source`, les autres paquets de `film/...`, `replaybuild`, `killcollector`
      (`-skip "$SKIPREV"`) verts ; archlint vert ; vet avec et sans `research` sur `film/...` verts ;
      golangci-lint (`--new-from-merge-base=origin/feat/v75`, `grammar/...` et `source/...`) 0 problème.

### Étape R2 — Le bit des films 1.8 (option A, U-2)
- [x] R2.1 Lire dans Ghidra `FUN_142988e98`, cible de `FUN_1428e219c` (`CMP dword ptr [RDX],0x29` à
      `1428e21a6`) quand la majeure diffère de 0x29, et `FUN_14299bd78` (`14299bd83`) : l'événement
      posé (code 0 sur `+0x1488`) et son traitement. Sortie : LU (le chemin des anciens formats existe ;
      son traitement du bloc malléable d'i53 ou de la version, adresses) ou NON LU (aucun chemin de
      lecture n'en part, adresses de ce qui a été lu). `$RES/g/r2/`.
- [x] R2.2 La version majeure déclarée (`chunk_00+0`) est portée par le contexte du film comme
      `MoteurUnPossible` (`grammaireSousFilm`), jamais par la chaîne de build. Si R2.1 est LU : la règle
      du jeu, adresses citées ; sinon : le seuil mesuré ≤ 0x25 (U-2), documenté comme mesure (rapport
      BIT_BUILDS §2), position du bit = le drapeau final de `FUN_1407efc5c` (lu à `1407efeba`) — la
      variante « R(5) de n sur 4 bits », indiscernable sur les données, est nommée au commentaire.
- [x] R2.3 Mesure → `$RES/r2` : I-ferm, I-d0, I-ancres, I-equipes, I-carte, I-e108 ; zéro différence
      sur les films de majeure ≥ 0x26 ; `60ae07c4`, `a349fea8`, `a521164d`, `50247b26` chiffrés ;
      ancres changées adjugées par E-4.
- [x] R2.4 Tests unitaires (seuil, film sans `chunk_00`, majeure absente) ; mutations ; ratchets.
- Gate R2 : comme R1.5 (instruments, G-equiv, killsource, tests, archlint, vet, lint) contre `$RES/r1`.
- *PREMIÈRE PASSE DE R2 (2026-10-14, règle « ≤ 0x25 ») : ARRÊT AU GATE, DÉCISION DE L'UTILISATEUR DUE ; suite au 2026-10-15 ci-dessous (U-6).* R2.1, R2.2 et R2.4 `[x]`,
  R2.3 `[!]` ; code commité, non poussé ; mesures sous `$RES/r2`, `$RES/r1p`, `$RES/r2p`, `$RES/r2/art`.
  - R2.1 `[x]` — LU, et ce n'est PAS un aiguillage : `FUN_1428e219c` appelle `FUN_142988e98` quand la
    majeure diffère de 0x29 (1428e21a6 `CMP dword ptr [RDX],0x29`, 1428e21b1 `MOV RCX,[RCX+0x130]`, 1428e22b0
    `CALL 0x142988e98`), et aussi quand la voie 0x29 échoue (`FUN_1414e77d0`). `FUN_142988e98` ne lit
    aucun octet du film : il remet à zéro l'état du lecteur de film (`+0x1488`, gabarit code 0 / 0x1e /
    constante `143cd8ff0` copié par `FUN_141cfebac` puis détruit par `FUN_1428df11c`), comme les remises à
    zéro `FUN_142988ce0` et `FUN_142985778`. `FUN_14299bd78` ne fait qu'un test. Aucun chemin des anciens
    formats ne part de là : le seuil est donc la MESURE (U-2, option A). `$RES/g/r2/`.
  - R2.2 `[x]` — `GrammaireBalayage.MalleableSansDrapeauFinal`, dérivé du film (`malleable_du_film.go`,
    majeure déclarée ≤ 0x25 ; majeure illisible : grammaire de l'exécutable) par le contexte de film
    (`poserLaGrammaireDuFilm`, `profilDeLaVueA`) et `GrammaireSousFilm` (killsource) ; le lecteur
    `consumeBipedMalleablePropertyBlock` saute le drapeau final (1407efeba) quand il est posé.
  - R2.4 `[x]` — `malleable_du_film_test.go` (règle par bobine, film sans `chunk_00`, largeur du bloc ;
    mutations « seuil 0x26 » et « champ ignoré » rouges) ; golden de fermeture en cuisson régénéré avec
    sa ligne d'historique (`60ae07c4` `ti=35` 1 → 232/236, `a521164d` 2 → 18/209), golden sans carte
    inchangé ; G-unit (`grammar`, `facts`, `replay`, `replaybuild`, `killcollector`), archlint, vet avec
    et sans `research`, golangci-lint : verts.
  - R2.3 `[!]` — mesures :
    - **Films de majeure ≥ 0x26 : aucune régression.** 24 films du corpus et les 2 témoins HI_1_9_0 du
      parc : fermetures, ancres, équipes, carte delta (8 films à l'identique, et les 16 autres de la
      carte 20) identiques ; artefacts publiés identiques (`replay-equiv`) ; killsource : morts publiées
      identiques. Seuls changent les digests internes `killsource`, `vehicles`, `vehicleDeaths.stats`
      de `replay-equiv`, parce que le profil de balayage porté par les faits gagne un champ (même cas que
      LK.6.4) : `SchemaDesFaits` montera en R5.
    - **Majeure 0x25 (HI_1_8_0) : gain net, aucune perte.** `60ae07c4` : bipèdes fermés 1 → 328/332, carte
      delta +49 paquets fermés (MPP +1 596), artefact : dotations de naissance 5 → 171 lues et publiées,
      postures +10, deux « impulsions de capacité » disparues (lues sur des bits étrangers par l'ancien
      i57 : `otherFamily`, `noIdentity`) ; killsource : mêmes morts, 5 passent du balayage à la marche
      séquentielle. Parc, 12 autres films HI_1_8_0 (carte posée à « Live Fire - Ranked » sans
      vérification, mesure relative) : bipèdes fermés 1 263 → 7 054 sur 7 820 tous films confondus,
      0 perte, ancres identiques, admissions inchangées (5 461).
    - **Majeures plus anciennes (0x1f à 0x22, formats 20, 21, 23) : effets mêlés, NON ADJUGEABLES.**
      Bipèdes : `a349fea8` +68 / −7, `a521164d` +30 / −6, `50247b26` +8 / −3, `58e6f72a` (parc, 0x22)
      15 → 27. Les 13 pertes de `a349fea8` et `a521164d` fermaient en base avec les largeurs modales
      (n d'i53 = 5, i55 = 5, i57 = 2, portes ouvertes) : rien ne dit que leur fermeture de base était un
      hasard (`$RES/r2/fermes_perdus.txt`). Le témoin décalé d'un bit ferme par hasard 5 records sur
      `50247b26` (0 avant), 7 sur `a349fea8` (2 avant), 3 sur `a521164d` (1 avant). Publié : `a349fea8`
      perd 6 dotations de naissance (22 → 16), et les 22 dotations à deux armes (`0x2B1824D5` +
      `0x48C19D2D`) deviennent 16 dotations à une seule arme (`0x48C19D2D`), d'autres joueurs, d'autres
      instants — une régression d'une valeur publiée ; `50247b26` : dotations fermées 3 → 1 ;
      `a521164d` 19 → 20.
  - **Décision demandée à l'utilisateur** (aucune n'est prise par l'exécutant) :
    (a) garder « majeure ≤ 0x25 » telle que décidée le 2026-10-09 et accepter la régression mesurée sur
    les formats 20-21 (films déjà faux après i22, D-4/D-5) ; (b) borner la règle à ce que la mesure
    établit, la majeure 0x25 seule : tout le gain de 1.8, aucun changement sur les formats 20 à 23,
    mais c'est une borne par valeur déclarée et non un seuil ; (c) laisser les formats 20-21 hors de la
    règle et les instruire à part (le bloc malleable n'y est pas établi au-delà de ce bit, découverte
    R-1). Révisions (`grammar`, `killsource` qui change de sortie, `SchemaDesFaits`) : en R5, quelle que
    soit l'option.

- *GATE R2 PASSÉ (2026-10-15, règle « majeure déclarée = 0x25 », U-6 ; `$RES/r2b`, `$RES/r2bp`)* — R2 CLOSE.
  - Code : `malleableSansDrapeauFinal` rend vrai pour la seule majeure déclarée 0x25 ; test par bobine
    (seule `60ae07c4` ; `a521164d` 0x21 et les builds ≥ 0x26 non), mutations `<= 0x25` et `0x26` rouges.
  - Corpus contre `$RES/r1` : UNE ligne bouge, `60ae07c4` `ti=35` 1 → 328/332 (327 records gagnés, 0
    perdu) ; bipèdes 8 237 → 8 564 ; ancres (colonnes 1-9) et équipes identiques ; admissions 6 507 ;
    formats 20-21 IDENTIQUES à r1. Carte delta : seuls 49 paquets de `60ae07c4` passent fermés (MPP
    1 596), tout le reste identique. killsource : morts publiées identiques sur 19 témoins, seul
    `60ae07c4` change de santé et de calibration (5 morts lues par la marche séquentielle au lieu du
    balayage). `replay-equiv` (20 films, `depuis_les_faits=false`) : seul `60ae07c4` change de document,
    à l'identique de la première passe (adjugé : dotations de naissance 5 → 171, postures +10, deux
    impulsions de capacité fausses retirées) ; sur les 19 autres, seuls les digests internes
    `killsource`, `vehicles`, `vehicleDeaths.stats` changent (profil porté par les faits,
    `SchemaDesFaits` en R5), artefacts identiques.
  - Parc contre `$RES/r1p` : les 12 films HI_1_8_0 gagnent tous, bipèdes fermés 1 263 → 7 042 sur 7 820
    tous films confondus, 0 perte ; `58e6f72a` (0x22) et les deux HI_1_9_0 identiques ; ancres
    identiques ; admissions 5 461.
  - Golden de fermeture en cuisson : `60ae07c4` 1 → 232/236 seul (ligne d'historique réécrite) ; golden
    sans carte inchangé. Tests de `film/...`, `replaybuild`, `killcollector` (sous `-skip "$SKIPREV"`),
    archlint, vet avec et sans `research`, golangci-lint : verts.

### Étape R3 — world-object i0 sous la portée, puis rigid-body i18
- [x] R3.1 Réappliquer la lecture du jeu de world-object i0 sous la garde (`git revert 72785f870`,
      qui annule le retrait de `64ef3baa7` ; forme de `FUN_14076e29c`, D-18), sur la tête de R2.
- [x] R3.2 Mesure → `$RES/r3.1` contre `$RES/r2` : I-ferm (baisses `ti=37` et `50247b26` chiffrées
      film par film), goldens, I-ancres (E-4), I-carte, I-equipes, I-d0. Chaque baisse adjugée sur
      pièces ; une baisse non adjugeable : ARRÊT, décision de l'utilisateur (retenir ou non).
- [x] R3.3 Si R3.1 est retenue : réappliquer `consumeGenericRigidBodyTransforms` sous la garde
      (`git revert 421f7c1cb`, D-21), mesurer → `$RES/r3.3` contre `$RES/r3.1`, même règle. Le contrôle
      d'orientation de `FUN_140501798` (D-21) reste hors périmètre sauf s'il explique une baisse.
- Gate R3 : comme R1.5 contre `$RES/r2`.
- *PREMIÈRE PASSE DE R3 (2026-10-15) : ARRÊT EN R3.2 ; décision U-7 « Option a » le même jour, R3.1 rétablie (`106729bfd`), R3.2 close — voir le journal.* R3.1 `[x]`, R3.2 `[!]`, R3.3 `[!]`
  (dépend de R3.1 retenue). Mesures sous `$RES/r3.1` (base : `$RES/r2b`, la tête de R2 close).
  - R3.1 `[x]` — `FUN_14076e29c` relu (`$RES/g/r3/`) : precHigh et R(96) (`FUN_14076e420`), queue
    `FUN_14076e3e4(precHigh)` (`MOV R9B,AL` en 14076e2c5), R(2) si finis (`FUN_140492128`), faux sinon.
    Porté par le geste unique `sousLaGardeSinonException` (le bloc de tête de `64ef3baa7` ne passerait
    plus le garde-rail de LK.5.4.4) ; cas de site sous la portée et test « flottant non fini ».
    Commit mesuré `ebc3a1651`, retiré par `25b7df4f4` (la branche reste à l'état de R2 close).
  - R3.2 `[!]` — mesure contre R2 (28 films, contexte de cuisson) :
    - hausses (82 lignes) : `ti=38` 31 324 → 113 049 / 119 148, `ti=42` 6 813 → 21 449 / 23 445,
      `ti=43` 11 324 → 37 959 / 40 716 ; `ti=37` gagne des records sur 26 films ;
    - INCHANGÉS, à la différence de LK.5.4.1 : élection des ancres (colonnes 1-9 identiques, aucune
      région changée), équipes, carte delta (20, 8 et MPP : aucun paquet ne change de fermeture), I-d0
      (bipèdes, admissions 6 507), goldens de fermeture des sept bobines (aucune baisse) ;
    - CINQ lignes en baisse :
      - `ti=37` `084a804d` 40 → 38 et `1c4c63c2` 65 → 64 : trois records qui fermaient au bit près en base
        (slots 6503 → 6505 et 6687 → 6692 sur `084a804d`, entrées intercalées lues ; 6003 → 6004, slots
        contigus, sur `1c4c63c2`) et que la lecture du jeu allonge de 24, 24 et 1 bits ; le témoin décalé
        d'un bit fermait en base 0 record `ti=37` sur `084a804d` et 3 par hasard sur `1c4c63c2`, sur 3 441 et
        2 720 bornés : rien ne désigne ces trois fermetures comme du hasard —
        NON ADJUGEABLES ;
      - `ti=37` `a349fea8` (format 20) 33 → 15 : 18 records perdus, le témoin ne fermait pas `ti=37` en
        base ; sous la lecture du jeu, le témoin y ferme 10 records par hasard pour 15 fermetures réelles — NON
        ADJUGEABLES par la preuve, film déjà faux après i22 (D-4/D-5 du plan clos) ;
      - `50247b26` (format 20, sans section d'identification) `ti=38` 452 → 159 : en base le témoin
        décalé d'un bit fermait 605 records dont 528 par hasard, davantage que la lecture elle-même —
        ADJUGEABLE comme fermetures de hasard ; `ti=42` 134 → 16 : le témoin n'en fermait que 14 —
        NON ADJUGEABLE par la preuve (arrêts `largeur_handle_moteur_un` et `position_non_finie` d'un film
        dont le type de moteur n'est pas déclaré).
  - **Décision demandée à l'utilisateur** : (a) retenir la lecture du jeu (gain massif sur les objets du
    monde, ancres et carte inchangées) en acceptant ces baisses, dont quatre records `ti=37` sur des films
    récents ; (b) la laisser rejetée ; (c) la retenir hors des formats 20-21 (même logique que U-6) et
    instruire les quatre records `ti=37` récents avant de conclure. R3.3 (i18) attend cette décision :
    D-21 du plan clos le place derrière i0.

### Étape R4 — La valeur 2 de l'emplacement désiré (D-30)
- [x] R4.1 Mesure sur les 28 films (instrument `research`) : pour chaque record bipède admis,
      l'emplacement désiré (param[1] d'i42), l'état de l'emplacement qu'il désigne (vide ou arme),
      l'instant (première image-clé du match, avant ou après la première arme publiée du joueur),
      l'état de vie. Hypothèse U-3 testée : `d = 2` n'apparaît-il que quand l'emplacement 2 est vide,
      avant l'attribution de la dotation ? Sortie chiffrée et conclusion écrite.
- [x] R4.2 Sens retenu, écrit sur pièces : la règle générale (« l'emplacement désiré désigne un
      emplacement vide : rien en main ») si la mesure l'établit ; sinon le sens que la mesure établit.
      Une mesure qui contredit U-3 sans établir de sens : ARRÊT, point à l'utilisateur.
- [x] R4.3 Web : `apps/web/src/features/match-replay/model/equippedLogic.ts` (`holstered: d === 2`)
      et son test « D=2 : RIEN de dégainé » remplacés par la règle de R4.2 ; textes UI éventuels en FR
      ET EN (`i18n.ts`) ; tests vitest ; typecheck (cache purgé) ; lint ; `npx playwright test` (le
      rendu change) ; gate visuel au navigateur sur les témoins que l'utilisateur nomme.
- Gate R4 : vitest, typecheck, lint, Playwright verts ; mesure R4.1 publiée au journal.
- *GATE R4 PASSÉ (2026-10-15, `$RES/r4/emplacement_desire.tsv`)* — R4 CLOSE.
  - R4.1 (instrument `res_r4_emplacement_desire_research_test.go`, 28 films, contexte de cuisson ; une
    ligne par record bipède lu : admis, param[1], param[2], valeur publiée par la fenêtre, contenu des
    quatre emplacements) :
    - records ADMIS (6 507) : param[1] vaut 0 (3 784) ou 1 (2 723), jamais 2 ;
    - la valeur 2 vient des records NON admis (règle T1 : l'emplacement 2 porte une famille hors du
      catalogue des armes), publiée par la fenêtre (705 records à 2 publiés, 553 d'entre eux avec
      param[1] = 2 relu par la grammaire) ;
    - à la PREMIÈRE image-clé à bipèdes de chaque film, 283 records ont param[1] = 2, dont 279 avec
      l'emplacement 2 = `00007CA9`, les MAINS NUES (`hinf_unarmed`), et les emplacements 0 et 1 déjà
      armés : HYPOTHÈSE U-3 CONFIRMÉE — avant le départ, aucune arme n'est en main (mains nues) ;
    - à l'image-clé suivante, param[1] = 2 tombe à 41 records sur 360 (170 à 0, 94 à 1) : la valeur
      disparaît dès que le joueur prend une arme ;
    - ailleurs (455 records), l'emplacement 2 porte surtout un OBJET TENU : bobine à fusion
      (`e9e7ff79`, 76), drapeau (`2a392328` 60, `0017592c` 41), bombe (`3fee4fcf`, 13), autres (37) ;
      mains nues 29 ; vide 98. Mains nues en cours de partie : rare (29 sur 455), comme l'utilisateur
      l'attendait.
  - R4.2 : SENS ÉTABLI — `d` est l'emplacement désiré (index 0..3) ; l'emplacement 2 est celui de
    l'objet tenu hors de la rangée d'armes : les mains nues au départ, un objet d'objectif ensuite.
    D = 2 dit donc « aucune arme de la rangée en main », ce qu'affichait le web ; c'est sa
    DOCUMENTATION qui était fausse (« aucune arme dégainée », sans lecteur). Le web ne montre pas
    « rien en main » à tort avant l'attribution : c'est l'état réel (mains nues), et il disparaît à la
    prise d'arme.
  - R4.3 : `equippedLogic.ts` — en-tête et champ `holstered` réécrits au sens mesuré ; règle
    généralisée à D ≥ 2 (un emplacement hors de la rangée, lu, n'est jamais une lacune : D = 3 ne fait
    plus afficher « dégainée ? ») ; commentaires de `handCellHint.ts`, `ReplayInventoryRow.tsx`,
    `ReplayWeaponsRow.tsx` alignés ; test « D=2 » réécrit, test « D=3 » ajouté. Doc Go de
    `types.KeyframeInventory.DrawnSlot` corrigée (ferme D-30 du plan clos). Aucun texte d'interface
    neuf (i18n FR/EN inchangé), aucune couleur. vitest (match-replay : 3 196 tests), `tsc -b --force`,
    eslint des fichiers touchés : verts. Playwright `[~]` NON JOUÉ : l'affichage du rejeu ne change
    pour aucun artefact — les valeurs publiées sont 0, 1 et 2 (la fenêtre ne publie jamais 3), et le
    rendu de D = 2 est inchangé ; les tests E2E exigent de plus les serveurs de `make dev`, qui sont
    ceux du checkout principal.

### Étape R5 — Révisions, gates complets, revue adversariale, fusion
- [x] R5.1 `git fetch`, fusion de `origin/feat/v75` (§1.2) ; références rejouées si la fusion touche
      `film/`.
  - Trois fusions : `39ead801f` (lot « équipement porté », schéma 95, `75a53aa1e`), puis `46bfa5fad`
    (fin du champ de réparation, schéma 96, `grammar-2026-10-15`, `SchemaDesFaits` 14 : conflits sur
    toutes les révisions, `2c782cf00`), puis `ec7313562` (rattrapage CSR, aucun fichier du film,
    `79e54a67f`). Références rejouées sur la base fusionnée, sources et binaires de `feat/v75`
    `46bfa5fad` (`$RES/base_v75`, sorties `$RES/v75b`, `$RES/v75c`) contre la tête (`$RES/r5g`) :
    - I-d0, I-ferm, I-ancres, I-equipes, I-e108 : sorties de la tête IDENTIQUES à l'octet à celles de la
      première passe de R5 (`$RES/r5`) ; sorties de la base `46bfa5fad` IDENTIQUES à celles de
      `39ead801f` (`$RES/v75`) — le lot de `feat/v75` ne change aucune fermeture.
    - Cumul contre `feat/v75` : bipèdes fermés 5 397 → 8 564 sur 10 710 ; seules baisses `50247b26`
      `ti=38` 451 → 159 et `ti=42` 76 → 16 (acceptées par U-7) et la ligne `64e8adfa` `ti=0` disparue (la
      fausse ancre slot 1536, R1.3) ; ancres : les deux régions de `64e8adfa`, ADJUGÉES (`adj.sh`) ;
      équipes identiques ; admissions 6 507 inchangées.
    - I-carte (20, 8, MPP) : colonnes de fermeture identiques à la première passe des deux côtés ;
      seules la mémoire et la durée varient.
    - killsource, 19 témoins : 18 films identiques à l'octet ; `60ae07c4` : morts, instants, tueurs,
      victimes, sources identiques, santé et calibration changent (5 morts passent du balayage à la
      marche séquentielle) ; sorties de la tête identiques à l'octet à la première passe.
    - `replay-equiv`, 20 films, `depuis_les_faits=false` : mêmes étapes divergentes qu'à la première
      passe (digests internes `killsource`, `vehicles`, `vehicleDeaths.stats` sur les 20 — profil porté
      par les faits ; `zoneKeyReads` de trois films, adjugé en R1.5 ; `60ae07c4` et `64e8adfa`) ; le
      lot de `feat/v75` change `carriedEquipment` sur 19 films à l'identique des deux côtés. Artefact
      `60ae07c4` (`replay-build --facts`, `$RES/r5g/art/`) : dotations 257 → 420, postures 1 945 →
      1 955, équipement porté 27 → 28 (fins de port lues, une faille de plus) et ses objets suivis
      (`trackedObjects` 2 → 3, ceux du nouvel objet porté), tout le reste identique (pistes identiques
      à la première passe). Aucune baisse nouvelle.
    - S8 (`-deux-passes`), 20 films : 20/20 artefacts identiques à l'octet entre décodage et rejeu
      depuis les faits ; le seul écart d'étape est le digest `killsource`, interne à la passe-faits,
      comme à la première passe.
- [x] R5.2 `grammar.Rev` monte (rang au-dessus de `feat/v75`), chronique `rev_chronique.go` (mesures de
      R1 à R3, adjudications) ; porte `LEVELUP_UPDATE_GRAMMAR_REV=1 go test $G -count=1 -run '^TestGrammarRevSuitLaGrammaire$' -update-grammar-rev`.
  - `grammar-2026-10-15.2` (`feat/v75` est à `grammar-2026-10-15`) ; entrée de chronique après celle
    du lot « équipement porté » ; golden par sa porte.
- [x] R5.3 killsource : `cmd/killsource json` 19 témoins, base `origin/feat/v75` contre lot ; identique
      → complément daté dans `facts/killsource/rev_chronique.go` et golden régénéré par sa porte
      (`LEVELUP_UPDATE_KILLSOURCE_REV=1 ... -run '^TestKillsourceRevSuitLaSortie$' -update-killsource-rev`) ;
      sinon montée de `killsource.Rev`.
  - Sortie changée sur `60ae07c4` (santé, calibration) : montée à `killsource-2026-10-15` (`feat/v75`
    est à `killsource-2026-10-14`), entrée datée, golden par sa porte.
- [x] R5.4 objectives : étape `objectives` de `replay-equiv` (20 films) identique → complément daté
      et golden par sa porte (`LEVELUP_UPDATE_OBJECTIVES_REV=1 ... -update-objectives-rev`) ; sinon
      montée d'`objectives.Rev`.
  - Étape `objectives` identique sur les 20 films : `objectives-2026-09-27` constante, complément
    daté après celui du lot « équipement porté », golden par sa porte.
- [x] R5.5 `replay.SchemaVersion` monte au-dessus de `feat/v75` si le document publié change
      (chronique `document_chronicle.go`, plafonds de `archlint/film_file_size_test.go`, goldens
      d'assemblage et de forme, fixtures de contrat du web) ; `SchemaDesFaits` si la forme des faits
      change (survie depuis les faits, mode `-deux-passes` de `replay-equiv`).
  - `SchemaVersion` 97 (`feat/v75` à 96), entrée v97 de la chronique, justification de
    `structure_test.go`, plafonds 3 353 et 1 452 avec leur exception écrite ; `SchemaDesFaits` 15
    (`feat/v75` à 14 : le profil de la section 5 gagne `MalleableSansDrapeauFinal`) ; huit goldens
    d'assemblage, golden de forme, formes des types et huit fixtures de contrat du web
    (`replay_schema_97_*`) régénérés par leurs portes.
- [x] R5.6 Gates complets : G-film sans `-skip` ; archlint ; vet avec et sans `research` ;
      golangci-lint ; `go test -tags=integration -p 1` des paquets touchés ; G-perf (cuisson et sync,
      4 films, paires alternées, seuil +10 %, plan clos §5) ; G-corpus et banc de vérité contre
      `origin/feat/v75` (base partagée EN LECTURE, parc libre vérifié) ; `make gate-push` (étapes) ;
      typecheck et lint web.
  - Verts : `film/...`, `archlint` (avant la fusion de `ec7313562`), `contracttest`, `replayview`,
    `replaydiff`, `replay-corpus-gate` (tests) ; vet de `film/...` avec et sans `research` ;
    golangci-lint du module (`--new-from-merge-base=origin/main`) et de `grammar/...` sous `research` :
    0 problème ; `-tags=integration -p 1` de `film/...`, `replaybuild`, `sync/killcollector` ; web :
    typecheck (trois projets, sans fichier d'état), eslint (0 erreur), vitest `match-replay` et
    `lib/replay` (255 fichiers, 3 624 tests).
  - gate-push : lint Go et web verts ; baseline de tests (suite `-tags=integration -p 1 -json` jouée
    par groupes, JSONL vérifié par `check_test_baseline.sh tests --from-jsonl`) : tous les tests
    présents, UN rouge hérité de `feat/v75` `ec7313562` (lot du rattrapage CSR) :
    `archlint::TestNoNewRawStartTimeLiteral` sur `internal/sync/csr_backfill.go:184`
    (`ORDER BY r.start_time ASC`, règle CLAUDE.md n°8). Corrigé dans un commit séparé (`644e115b4`,
    tri par `analysis.SQLStartTimeCanonical("r")`, le modèle de `sync/events_replay.go` ; le schéma du
    test d'intégration gagne `start_time_utc`, comme la table réelle), sur autorisation du superviseur
    du 2026-10-15 ; levelup-6c porte la même correction vers `feat/v75` : à la refusion, la version de
    `feat/v75` prévaut. CI de `644e115b4` (run 38724066987) : tous les jobs verts, Playwright sauté.
  - G-corpus (après contrôle des processus : la recuisson de levelup-66 finie, aucun `backfill`),
    19 témoins en trois lots, base `origin/feat/v75` `ec7313562` cuite par le gate, racine de parc
    `$RES/parc` (jonctions de lecture) : banc de vérité `ok` sur les 19 ; aucun oracle ne bouge.
    18 témoins : 0 perte, 0 changement. `60ae07c4` (HI_1_8_0) : `[gain] P-1` paquets fermés au bit
    près 38 434 → 40 031 ; les replis comptés baissent (`repli_pistes_du_monde_apres_la_marche`
    29 149 → 24 457, `repli_ancrage_bipede_apres_la_marche` 18 775 → 17 017, …) ; 21 pertes de filet,
    ADJUGÉES sur l'artefact (`$RES/r5g/art/`) :
    - `stances` : durées de sprint raccourcies sur 9 slots, +10 intervalles. Les états de mouvement lus
      passent de 6 025 à 6 050 ; un sprint s'arrête là où le film l'arrête au lieu de courir jusqu'à
      la lecture suivante. Contrôle physique (sprint à vitesse constante, environ 0,28 u/trame) sur les
      portions retirées : `630` 5432-5490 à 0,17, `626` 5437-5447 à 0,14, `659` 7153-7167 à 0,20,
      `531` 1081-1102 à 0,18 (marche, pas sprint) ; `647` 6216-6226 et `641` 5974-5999 sont des sauts
      (z monte puis redescend), où l'état n'est pas le sprint. Adjugé : lecture plus juste.
    - `abilityImpulses` 2 → 0 : les deux impulsions lues sur des bits étrangers (adjugé en R2).
    - `deathsPaths.directScan` 14 → 9 : cinq morts passent du balayage à la marche séquentielle
      (`walk.population` 151 → 156), contenu des morts identique (killsource, R5.3).
    - `birthLoadouts.noLife` 0 → 2 : sur 171 dotations lues au lieu de 5, deux sans vie à rattacher.
    - `carriedEquipment.noIdentity` 118 → 126, `withoutRank` 148 → 159 : objets créés lus 485 → 507.
    - `groundWeapons.accepted` 302 → 299, `powerupAccepted` 351 → 349, `placements.lives` 260 → 257 :
      compteurs intermédiaires ; les armes au sol publiées (188) et les placements publiés (190)
      sont identiques à la base.
    - `stances.dropped` 9 → 11, `refusedNewFalseReads` 15 → 16 : à proportion des lectures en plus.
    Chaque témoin compte une différence de sens « gain » (même compte sur les 19, y compris les films
    dont le document ne change pas : la montée de schéma) ; télémétrie : les deux révisions.
  - G-perf, paires alternées base `46bfa5fad` / lot (`replay-equiv` et `porteurs` construits
    une fois chacun), machine contrôlée avant chaque tour (`$RES/perf/`) :
    - **Sync** (`BenchmarkPorteursAuSync`, 3 paires) : `084a804d` 1,00, `1c4c63c2` 1,00, `51101d1d`
      1,00, `9f57c612` 0,96 (médianes lot/base, ns/op) ; pics dans le bruit de la base. Tenu.
    - **Cuisson** (`replay-equiv`, 3 paires, bruit de la base ≤ 0,6 %) : médianes lot/base
      `084a804d` 1,069 (44,32 → 47,39 s), `e5adf7b2` 1,110 (15,99 → 17,73 s), `60ae07c4` 1,041
      (13,99 → 14,52 s), `11de8353` 1,111 (15,74 → 17,52 s) ; pics inchangés (± 0,03 Gio).
      **SEUIL DÉPASSÉ sur deux films (+11 %)** : ARRÊT du gate selon le §5 du plan clos, mesure
      publiée ; DÉCISION U-8 de l'utilisateur (2026-10-15, « option a ») : coût accepté. Le surcoût suit le volume lu en plus (records fermés,
      tous archétypes, 242 684 → 418 264 sur 28 films ; `ti=38` et `ti=43` sous la portée) ; non
      attribué plus finement (un profil CPU d'une cuisson isolée de `e5adf7b2` par `replay-build`
      est au bruit près des deux côtés et ne le montre pas).
- [x] R5.7 ADR 0037 (IR-4/IR-6 : la table d'entrées, Rslot, la version majeure déclarée) ;
      WALK_PORT_NOTES ; `.ai/REGISTRE_REPORTS.md` ; plan clos (D-18, D-21, D-30 renvoyés ici).
  - ADR 0037 : paragraphe daté sous IR-6 (table d'entrées, fermeture sur l'ancre suivante et sur la fin
    de table, sentinelles retirées, majeure 0x25, i0 et i18 sous la portée, mesures) ; WALK_PORT_NOTES :
    section datée et adresses neuves ; registre : quatre reports ; plan clos : D-18, D-21, D-30 annotés.
- [ ] R5.8 Push de `feat/ri-ecart-108`, CI verte au niveau job (`gh run list --branch feat/ri-ecart-108 --limit 3`,
      `timeout 590 gh run watch <id> --exit-status`).
- [ ] R5.9 Revue adversariale du diff (skill `adversarial-review`, contexte frais, lancée par le
      superviseur) ; constats statués.
- [ ] R5.10 Demande de fusion à l'utilisateur, par le superviseur ; sur accord : fusion, recuisson du
      parc et backfill killsource (gestes du superviseur), pairs prévenus.
- Gate R5 : tout ce qui précède vert ; aucune baisse non adjugée ; CI verte ; revue passée.

## 5. Contrat d'exécution et pièges

Skill `plan-execution` ; jamais `git stash`, `git add -A` ni `git add .` ; jamais Python ; aucune
commande `go` concurrente (contrôle des processus avant une passe longue) ; aucune base DuckDB ouverte
hors G-corpus (§0) ; commentaires = contrat (règle 17) ; fichiers ≤ 500 lignes, fonctions ≤ 80 lignes.

**Pièges d'exploitation** (plan clos §5, toujours valides) : fraîcheur des faits = révisions déclarées
(renommer `data/cache/film_facts`, dossier réel, avant chaque passe de `replay-equiv`) ; durée et pic
varient de 15 % sous charge (binaires alternés) ; un PC en veille gèle une passe sans la tuer ;
`replay-equiv` avec `-repo-root`, jamais `LEVELUP_REPO_ROOT` ; éditer par motif exact ; I-carte écrit
ses lignes dans l'ordre des films (garder l'ordre `$F20,$F8`) ; un dossier de sortie neuf par passe ;
I-ferm sort PASS sans fichier si une variable manque (contrôler `--- PASS` et le fichier) ; un filtre
`FAIL` nu attrape des journaux applicatifs (ancrer `^--- FAIL:` et lire le code de sortie).

## 6. Protocole de reprise

Relire le skill `plan-execution`, puis ce fichier (§2, §3, §8) ; reprendre à la première case non
statuée ; `git -C $WT log --oneline -10` ; vérifier sur `origin/feat/v75` si un lot a été fusionné
depuis la dernière reprise (§1.2 : refusionner, rejouer les références touchées).

## 7. Découvertes (consignées, non traitées)

- R-1 *(R2.3)* Sur les films de majeure 0x1f à 0x22 (formats 20, 21, 23), le bloc malleable d'i53 ne suit
  ni la grammaire de l'exécutable ni celle des films 1.8 pour tous les records : sans drapeau final,
  des records sans porte ouverte ferment (signature +33 ramenée à 0), mais des records à portes ouvertes
  qui fermaient avec les largeurs modales ne ferment plus. Grammaire non établie ; seul un exécutable
  de ces builds la lirait. Non traitée : ces versions gardent la grammaire de l'exécutable (U-6).
- R-2 *(R2.1)* `FUN_142988e98`, que `FUN_1428e219c` appelle pour une version majeure différente de 0x29,
  n'est pas un aiguillage vers un chemin des anciens formats : il remet à zéro l'état du lecteur de
  film (`+0x1488`) sans lire le film. L'exécutable courant ne relit donc aucun film de majeure ≠ 0x29 ;
  la grammaire des anciens formats n'y est lisible nulle part (contraire à l'hypothèse d'aiguillage de
  U-2). Consignée, non traitée.
- R-3 *(R3.2, acceptée par U-7)* Trois records `ti=37` de films récents fermaient au bit près sous l'ancienne
  lecture de world-object i0 et ne ferment plus sous celle du jeu : `084a804d` slot 6503 (suivant 6505)
  et slot 6687 (suivant 6692), allongés de 24 bits chacun ; `1c4c63c2` slot 6003 (suivant 6004, slots
  contigus), allongé d'1 bit. Le témoin décalé d'un bit ne les explique pas. À instruire : un autre
  composant de l'équipement (D-18 du plan clos) ou une forme de i0 non relue pour ces records.

## 8. Journal

- 2026-10-14 : rédaction (exécutant de l'étape dédiée), sur `199c6984d`. Lectures Ghidra de R1.1
  commencées pendant la rédaction (`FUN_142e2bfd0`, `FUN_142e29cf8`, `FUN_1408be074`, `FUN_1408f1948`
  décompilés, `$RES/g/`) : conformes au rapport R-VEH-4. `origin/feat/v75` a avancé à `76c0dce35`
  (lots recos-d et recos-f, documents ; aucun fichier sous `film/`) : fusion à R1.0.
- 2026-10-14 : R1.0, R1.1 et R1.2 CLOSES. Fermeture sous Rslot : `keyframe_entrees.go` (lecteur
  d'entrée unique, `lireLesEntrees`, `fermeSurLAncreSuivante`), `keyframeBorne.SlotSuivant`, la preuve
  d'état complet et les deux `KeyClosed` par le helper, garde-rail `keyframe_fermeture_unique_test.go`
  (mutation : rouge). Effets mesurés : voir le gate R1.2. Trois rouges rencontrés au gate et réglés :
  (1) dérive PRÉEXISTANTE du golden de fermeture en cuisson (bloquant de `a521164d` `ti=35`, arrêt
  `jeu_d_armes_refuse` de la revue D1.4.6) figée dans un commit séparé (`0120bb446`), mesurée sur le
  code de base ; (2) l'invariant « un record fermé finit sur le suivant » de
  `TestLaPhaseDesImagesClesTientSesInvariants` devient « finit sur le suivant, ou avant lui de la place
  des entrées intercalées » ; (3) golden des familles de la mini-bobine : `managedProperties`, records
  `ti=13` d'image-clé fermés 31 → 32 (non prouvés 1 → 0), lectures d'image-clé 31 → 32, les 141 lectures
  delta identiques ligne à ligne (mesuré des deux côtés) — régénéré par sa porte ; le paragraphe
  d'historique n'entre pas dans le fichier (plafond de taille 553, règle 17 : l'histoire va au
  journal). Rouge hérité réparé (bloque archlint) : `ecart108_research_test.go` reconstruisait l'ensemble
  des familles connues, il appelle `weaponv3.FamillesConnues`.
- 2026-10-14 : R1.3, R1.4 et R1.5 CLOSES — ÉTAPE R1 CLOSE (tête `faf0cd345` + ce commit). Gates au
  plan. Décisions d'exécution : I-adj en script sur les sorties d'I-ancres (R1.3) ; témoin faible de
  chaînage sur la forme exacte de l'écrivain (R1.4.3, précision de E-1) ; retrait de la fin de table par
  sentinelles et des options du motif qu'elle seule utilisait, `source.Rev` constant (R1.4.4) ; nom
  `SkippedSansArchetype` gardé (R1.4.2). Aucun push (E-3 : montée des révisions en R5). Prochaine
  étape : R2 (lire `FUN_142988e98` d'abord).
- 2026-10-14 : R2 jouée jusqu'au gate et ARRÊTÉE (R2.3 `[!]`, décision de l'utilisateur due, au plan).
  Lecture du jeu : `FUN_142988e98` remet à zéro l'état du lecteur de film, aucun chemin des anciens
  formats ; règle appliquée sur le seuil mesuré (≤ 0x25, option A). Films ≥ 0x26 sans régression ;
  0x25 : gain net sans perte (corpus et 12 films du parc) ; formats 20-21 : gains et pertes mêlés, dont
  une valeur publiée en régression (`a349fea8`, dotations de naissance). Code et tests commités, rien
  poussé.
- 2026-10-15 : DÉCISION U-6 de l'utilisateur (« Ok avec toi, option b ») : règle bornée à la majeure
  déclarée 0x25. Gate R2 rejoué et PASSÉ : seul `60ae07c4` bouge dans le corpus (gain sans perte), les
  12 films HI_1_8_0 du parc gagnent, tout film ≠ 0x25 est identique ; goldens à jour. R2 CLOSE.
  Découvertes R-1 et R-2. Suite : R3.
- 2026-10-15 : R3.1 appliquée (geste unique de la garde) et mesurée contre R2 close ; R3.2 ARRÊTÉE :
  gains massifs `ti=38/42/43`, ancres, équipes et carte delta inchangées, mais cinq lignes en baisse dont
  quatre records `ti=37` de films récents sans preuve de hasard ; sous-pas retiré (`25b7df4f4`), décision
  de l'utilisateur due (R3.2). R3.3 en attente.
- 2026-10-15 : DÉCISION U-7 de l'utilisateur (« Option a ») : world-object i0 sous la garde retenu,
  retrait annulé (`106729bfd`). BAISSES ACCEPTÉES PAR U-7, ligne par ligne (mesure contre R2 close,
  `$RES/r3.1`, reproduite à l'identique par `$RES/r3a`) :
  1. `ti=37` `084a804d` 40 → 38 / 3 441 (records des slots 6503 et 6687, +24 bits chacun ; R-3) ;
  2. `ti=37` `1c4c63c2` 65 → 64 / 2 720 (record du slot 6003, +1 bit ; R-3) ;
  3. `ti=37` `a349fea8` 33 → 15 / 4 054 (format 20, 18 records ; le témoin y ferme 10 records par
     hasard sous la nouvelle lecture) ;
  4. `ti=38` `50247b26` 452 → 159 / 6 258 (format 20, sans section d'identification ; en base le témoin
     fermait 605 records dont 528 par hasard) ;
  5. `ti=42` `50247b26` 134 → 16 / 2 012 (même film ; arrêts `largeur_handle_moteur_un` et
     `position_non_finie`).
  Gate R3.1/R3.2 rejoué et PASSÉ : I-ferm, ancres, équipes et I-d0 identiques à `$RES/r3.1` ; killsource
  identique à l'octet à R2 (19 témoins) ; `replay-equiv` (20 films, `depuis_les_faits=false`) : digests
  IDENTIQUES à R2 sur les 20 films (aucun document ne change) ; goldens de fermeture régénérés avec leur
  ligne d'historique (`ti=38/42/43` à 100 % sur les sept bobines, aucune baisse) ; tests de `film/...`,
  `replaybuild`, `killcollector`, archlint, vet avec et sans `research`, golangci-lint : verts.
  `origin/feat/v75` a avancé (lot « équipement porté », schéma 95, 78 fichiers sous `film/`, dont
  `grammar.Rev` et `killsource.Rev`) : NON fusionné, signalé au superviseur (§1.2) ; la fusion se fera
  avant R5. R3.1 et R3.2 CLOSES ; suite : R3.3.
- 2026-10-15 : R3.3 FAITE et R3 CLOSE. `FUN_142f036f0` et `FUN_140c1e79c` relus (`$RES/g/r3/`) : masque
  R(8), puis par bit R(1), R(19) si 0, R(8), et `FUN_14076e494(.., 0x10, 0, .., 0)` — R(96) sous la garde.
  Porté par `sousLaGardeSinonException` (cas de site sous la portée ; mutation rouge). Mesure contre
  `$RES/r3a` (`$RES/r3.3`) : I-ferm IDENTIQUE à l'octet (`ti=38` 113 049 / 119 148 inchangé), ancres
  (colonnes 1-9), statistiques d'élection, équipes, I-d0 (admis 6 507), carte delta (20, 8, MPP),
  killsource (19 témoins, à l'octet) et `replay-equiv` (20 films, digests identiques) INCHANGÉS ; goldens
  de fermeture inchangés. Seule la preuve d'image-clé jouée dans le contexte invariant (sans carte) varie,
  sans changer aucune élection : prouvés `11de8353` +31, `e5adf7b2` +5, `1c4c63c2` −54 — ces 54 records
  `ti=38` sont fermés à l'identique dans le contexte de cuisson avant et après ; la différence vient de la
  lecture décalée du contexte sans carte (adjugée : ni élection, ni fermeture, ni document ne change). Le
  contrôle d'orientation `FUN_140501798` (sans bit) reste non porté (D-21 du plan clos). Aucune baisse.
  Tests de `film/...`, `replaybuild`, `killcollector`, archlint, vet avec et sans `research`,
  golangci-lint : verts. Suite : R4.
- 2026-10-15 : R4 CLOSE (gate au plan). La valeur 2 de l'emplacement désiré est l'emplacement de l'objet
  tenu : mains nues à la première image-clé (279/283), objet d'objectif ensuite ; hypothèse U-3
  confirmée ; web : documentation et règle (D ≥ 2) corrigées, rendu inchangé. Suite : R5 (fusion de
  `origin/feat/v75`, qui a avancé du lot « équipement porté », puis révisions et gates complets).
- 2026-10-15 : R5 jouée jusqu'au push. Trois fusions de `origin/feat/v75` (`39ead801f`, `46bfa5fad` au
  schéma 96 et `grammar-2026-10-15`, `ec7313562` sans fichier du film) ; révisions remontées au-dessus
  (`grammar-2026-10-15.2`, `killsource-2026-10-15`, `SchemaDesFaits` 15, `SchemaVersion` 97,
  `objectives` constante) ; références et gates de R1 à R4 rejoués sur la base fusionnée : aucune baisse
  nouvelle, toutes les différences déjà adjugées (R5.1). Gates verts sauf un rouge hérité de `feat/v75`
  (`archlint::TestNoNewRawStartTimeLiteral`, `internal/sync/csr_backfill.go:184`, lot CSR), non
  corrigé. Gate de corpus et G-perf reportés derrière la recuisson du parc (levelup-66). ADR 0037,
  WALK_PORT_NOTES, registre, plan clos à jour. Suite : push, CI, puis gate de corpus, revue
  adversariale et fusion (superviseur).
- 2026-10-15 : R5.6 rejouée après la recuisson de levelup-66. Rouge hérité corrigé à part (`644e115b4`,
  autorisation du superviseur ; levelup-6c porte la même correction vers `feat/v75`, dont la version
  prévaudra à la refusion), CI verte. Gate de corpus, 19 témoins : banc de vérité ok partout, seul
  `60ae07c4` bouge (gain P-1, 21 pertes de filet adjugées sur l'artefact). G-perf : sync tenu ; cuisson
  +11 % sur `e5adf7b2` et `11de8353` (seuil 10 %) : ARRÊT, décision de l'utilisateur due.
- 2026-10-15 : DÉCISION U-8 de l'utilisateur (« option a ») : le surcoût de cuisson est accepté, R5.6
  close ; registre : la ligne devient une piste d'optimisation future. Revue adversariale (deux
  relecteurs, superviseur) : cinq constats P2 corrigés — branche morte `SansArchetype` de
  `walkOneKeyframeRecord` retirée (instruments research gardés par le même filtre), phrase de l'ADR 0037
  sur la fermeture précisée (seule la preuve d'élection se restreint aux slots libérés), deux tests des
  gardes `n < 0` et `b.Want >= 0` (mutations rouges), doc de `Chained` corrigée. Sortie de grammar
  inchangée (I-d0, I-ferm, I-ancres, I-equipes identiques à l'octet) ; empreinte de `grammar-2026-10-15.2`
  recopiée (révision non publiée, aucune lecture changée).
- 2026-10-15 : fusion de `origin/feat/v75` `df51f37c0` (chantier Tactique coéquipiers, étiquettes de barres,
  correction CSR de levelup-6c). `csr_backfill.go` : la version de `feat/v75` (`ORDER BY start_canonique`)
  est gardée et la nôtre (`644e115b4`) retirée, comme convenu ; le schéma du test d'intégration était
  identique des deux côtés. Révisions de `feat/v75` inchangées (`grammar-2026-10-15`,
  `killsource-2026-10-14`, `SchemaDesFaits` 14, schéma 96) : les nôtres restent au-dessus. Tests du film,
  archlint, contrat, CSR, vet, golangci, typecheck et vitest du rejeu verts sur la base fusionnée.
- 2026-10-15 (superviseur) : FUSION dans `feat/v75` (`a9a9ac78a`, accord de l'utilisateur « tu as mon accord »),
  revue adversariale close (aucun défaut fonctionnel, cinq P2 corrigés). Recuisson au schéma 97 après la passe
  coéquipiers de levelup-6c : 238 artefacts, 0 erreur de décodage (30 min) ; projections code 0 ;
  backfill-killsource 2 598 matchs, 278 220 morts, 0 erreur de décodage, 1 erreur d'écriture (D-39, 34db6b33).
  Checkout principal mis à jour avec l'accord de l'utilisateur (« Go pour integration », fusion par-dessus son
  commit local « Archivage », conflit de plan pris côté feat/v75), air et vite relancés. PLAN CLOS.
