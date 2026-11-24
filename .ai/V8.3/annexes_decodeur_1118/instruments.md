# Instruments de mesure réemployables par les pistes (item 0.6, relevé du 2026-11-18)

Chaque commande a été LANCÉE le 2026-11-18 sur le film `38900a5e` (Team Slayer sur Detachment, build
`HI_1_13_0`, schéma 125, 82 pistes, 9 joueurs ; film présent sous `data/cache/film_chunks/38900a5e` du
dossier principal). Aucune écriture dans le dossier principal ; sorties sous `C:/t/dec/sorties/instr/`.
Mêmes conventions que `marche_de_publication.md` § 0 : `HOTE` = dossier principal (lecture seule),
`LU` = `C:/t/dec-lu`, `DEC` = `C:/t/dec/main`, `GOCACHE="$LOCALAPPDATA/go-build-dec-0"`, une seule commande
lourde à la fois. `GOWORK=<engendré>` = `C:/t/dec-lu/levelup-briques.work` (commandes lancées depuis
`LU/apps/go-api`, elles voient le clone) ; commandes lancées depuis `DEC` : `GOWORK=off`.

## 1. Préalables : faits du match, artefact « après »

Presque tous les instruments de LevelUp lisent des FAITS (`<id>.facts.json`, `<id>.oracle.json`) que seule
la base partagée fournit.

- Faits et oracle (verité de l'API) d'UN film, sans toucher à la base de l'hôte : travailler sur une COPIE
  (`metadata.duckdb` + `.wal` + `shared_matches_v2.duckdb`) rangée dans un parc de travail
  (`C:/t/dec/sorties/parc`, voir la marche § 0) :
  `cd LU/apps/go-api && GOWORK=off CGO_ENABLED=1 LEVELUP_REPO_ROOT=C:/t/dec/sorties/parc go run ./cmd/levelup
  replay-facts-export --out <dossier> --oracle 38900a5e` ; 11 s ; code 0 ; écrit `38900a5e.facts.json` (9
  joueurs, variante, cartes candidates `[Detachment Detachment]`) et `38900a5e.oracle.json` (9 joueurs).
  Lecture seule (`OpenReadForQuery`) ; refuse des faits ou un oracle vides ; échoue si un serveur tient la base
  en écriture. NE PAS le pointer sur l'hôte si on veut zéro lecture de la base vivante.
- Artefact « après » d'un film (rejeu du code courant, donc du clone si `GOWORK` le désigne) :
  `cd LU/apps/go-api && GOWORK=<engendré> CGO_ENABLED=0 LEVELUP_REPO_ROOT=C:/t/dec-lu go run ./cmd/replay-build
  --map Detachment --facts <faits>/38900a5e.facts.json 38900a5e` ; 8 s ; pic 0,39 Gio ; écrit
  `LU/data/cache/replays/halo_infinite/38900a5e.json.gz` (1,08 Mo, 82 pistes). Le film doit se trouver sous
  `LU/data/cache/film_chunks/38900a5e` (copie, pas jonction) ; ou donner son dossier en second argument
  positionnel (`... 38900a5e <dossier du film>`, lecture seule, y compris l'hôte), mais le verrou de décodage
  reste sous `LU/data/cache`. Les options précèdent l'identifiant. Piège : donner l'identifiant COURT écrit
  `matchId` court dans l'artefact (le diff le voit comme un « changement » d'en-tête) ; donner le complet pour
  comparer à l'artefact de l'hôte.
- Artefact « avant » : celui du parc, en lecture :
  `HOTE/data/cache/replays/halo_infinite/38900a5e.json.gz` (schéma 125, build des couches `grammar-2026-11-15.4`).

## 2. Instruments de LevelUp (`apps/go-api`)

| Instrument | Commande minimale (sur UN film) | Vérifié | Ce qui sort |
|---|---|---|---|
| `cmd/replay-verite` (banc de vérité) | `GOWORK=<engendré> go run ./cmd/replay-verite -avant avant.json.gz -apres apres.json.gz -faits 38900a5e.facts.json -oracle 38900a5e.oracle.json -temoin t38900a5e` | 1 s, code 0, verdict `ok` | Par oracle, VP/FP/FN : O-K1 éliminations (74/0/0), O-K2 morts (71/0/0), O-K3 assistances (23/0/0), O-S3 score personnel (VP 8 560, FN 200), O-T1 équipes, O-V1 morts par les vies (FN 7), O-V2 fins de vie ; fermeture au bit des paquets (P-1 30 517) ; violations V-1 à V-8 (V-1 saut : 5, V-2 objet hors emprise : 2, V-7 identité discordante : 2). Sans `-oracle`, O-S3 manque. Code 1 si FAUX ou MANQUE, 2 si usage. |
| `cmd/replay-verite -registre` | `go run ./cmd/replay-verite -registre > registre.json` | 20 s, code 0 | Registre des replis de cette révision (129 entrées, 128 `true`) ; `-registre-avant` le compare. |
| `cmd/replay-diff` | `GOWORK=<engendré> go run ./cmd/replay-diff -ancien avant.json.gz -nouveau apres.json.gz [-json diff.json] [-tout] [-quiet]` | 2 s, code 0 | Mesures nommées par axe (pertes, gains, changements, identiques) : sur ce film `1 ecarts sur 1175 mesures` (le `matchId` court contre complet). Lit les documents en `map[string]any` : voit un champ que le code ne connaît plus. La présence d'écarts n'est pas un échec (code 0). Deux artefacts du même film sont requis : pour une branche, `apres` = `replay-build` de la branche. |
| `cmd/replay-equiv -out-dir` | `GOWORK=<engendré> go run ./cmd/replay-equiv -repo-root C:/t/dec-lu -originaux <HOTE>/data/film_identites/xuids_originaux.txt -films 000d5950 -out-dir D` | 20 films en 343 s (vérifié) ; un film en 9 s | Un TSV de digests par étape (`etape<TAB>compte<TAB>sha`, 82 lignes, `# digest-grammar: 2` en tête) ; un film différent est nommé étape par étape. LIMITE : ne mesure QUE les 20 films de `replay/testdata/equivalence/` (il faut `<id>.facts.json` dans le clone) ; `-films 38900a5e` rend `ECHEC (code 11)` : faits introuvables. Les films du corpus doivent être COPIÉS sous `<repo-root>/data/cache` (pas de jonction). |
| `levelup replay-parc-vies` | `GOWORK=<engendré> CGO_ENABLED=1 LEVELUP_REPO_ROOT=C:/t/dec-lu go run ./cmd/levelup replay-parc-vies --faits <dossier des faits> --cache C:/t/dec-lu/data/cache --sortie D` ; puis `--comparer D1 D2` | 12 s, code 0 ; `--comparer` vérifié sur D contre D | `films.tsv` (une ligne par film : build, issue, variante, 82 pistes, 0 vie sans identité, compteurs du pont) et `identites/<id>.tsv` (slot, bornes, équipe, identité de chaque piste) ; `--comparer` : vies nommées, perdues, noms et équipes changés, pistes redécoupées (ici tout à 0). `--garder-documents` garde aussi les rejeux (relisibles par diff et vérité). `--cache` est explicite : peut désigner l'hôte en lecture, mais il y pose son verrou solo, donc préférer une copie. |
| `levelup replay-facts-export` (préalable) | voir § 1 | 11 s | faits et oracle du match. |
| `cmd/replay-corpus-gate` (si un film du manifeste est visé) | `--temoins <id>` parmi les 19 de `config/replay_corpus.toml` | gate entier : 1 299 s | Voir la marche, geste 9. `38900a5e` n'est pas un témoin du manifeste. |

## 3. Instruments du décodeur (`DEC`, étiquette `research`, `GOWORK=off`)

L'étiquette `research` couvre 458 fichiers de test et 11 dossiers sous `research/` : ce sont les instruments
du mainteneur (README du clone, ligne 134), pas une API. Ils lisent les films EN PLACE (le dossier
`film_chunks` de l'hôte convient : lecture seule) et n'écrivent que dans un dossier hors de `data/`.

| Instrument | Commande minimale | Vérifié | Ce qui sort |
|---|---|---|---|
| `research/cmd_fermeture` (carte de fermeture des trames, modes `fermeture`, `gb1`, `v2`) | `cd DEC && GOWORK=off go run -tags=research ./research/cmd_fermeture -racine "<HOTE>/data/cache/film_chunks" -films 38900a5e -sortie D` | 3 s, code 0 | `38900a5e build=HI_1_13_0 paquets=30461/30518 utiles=230053/230122 pic=73 Mio 1,42 s` ; `fermeture_films.tsv`, `fermeture_archetypes.tsv`, `fermeture_bloquants.tsv` (causes d'arrêt classées), `fermeture_resume.md`. `-mode gb1` = vies sans position, `-mode v2` = carte détaillée, `-paquets` = une ligne par paquet. Ne prend pas le verrou de décodage. Utile à toute piste qui doit dire combien de trames se ferment (lecture arrêtée, piste D et H). |
| `research/duree` (temps de décodage et de publication) | `cd DEC && DUREE_FILMS=f.txt DUREE_RACINE="<HOTE>/data/cache/film_chunks" DUREE_CATALOGUE=C:/t/dec-lu/data/titles/halo_infinite/reference/map_quant_bounds.json DUREE_SORTIE=D GOWORK=off go test -tags=research ./research/duree/ -run TestDureeDesTemoins -v -count=1` ; `f.txt` = une ligne `id<TAB>build<TAB>mode<TAB>module de la carte` (ici `38900a5e<TAB>HI_1_13_0<TAB>Team Slayer:Arena<TAB>fo09_academy`) | 5 s, `PASS` | `duree.tsv` : décodage 2 044 ms, publication 177 ms, pic 329 Mio, rejeu 5,87 Mo, chargement 29 ms, publication depuis les faits 181 ms ; et `<id>.replay.json` (le DOCUMENT de rejeu en clair, options minimales : sans killsource, statborg, objectifs, zones, géométrie ; donc plus court que celui de la cuisson). Instrument du budget de poids et de temps (gate H : poids ajouté < 2 %). |
| `internal/grammar` `TestLot1TirsEtCibles` (événements de tir, type 36) | `cd DEC && LOT1_TRAME_FILM="<HOTE>/data/cache/film_chunks/38900a5e" GOWORK=off go test -tags=research ./internal/grammar/ -run '^TestLot1TirsEtCibles$' -v -count=1` | 17 s (dont compilation), `PASS` | 751 paquets de tir, 0 de type différent, 98,5 % sans cible touchée, 13 armes distinctes (`variant_name`), précision d'arme. Instrument de la piste D (D.2). `TestLot1TypesCorpus` (`LOT1_CORPUS` = toute la racine `film_chunks`, 3 min sur 1 367 films) fait la même mesure sur le parc ENTIER : à ne lancer que sur une racine réduite. |
| `internal/grammar` `TestBancBitProjectile` (positions de projectile, bit par bit) | `cd DEC && BITPROJ_PARC=C:/t/dec-lu BITPROJ_FILMS="38900a5e=Detachment" GOWORK=off go test -tags=research ./internal/grammar/ -run '^TestBancBitProjectile$' -v -count=1` | 3 s, `PASS` | Bornes de la carte, largeurs d'axe, nombre de records retenus (15 393), pas impossibles (> 10 m : 2 sur 14 861, 0,01 %). `BITPROJ_PARC` = une racine portant `data/titles/.../map_quant_bounds.json` ET `data/cache/film_chunks/<id>` (ici `LU` avec la copie du film ; l'hôte convient aussi en lecture). Instrument de la piste D (objets projectile). |
| `research/cmd_grenadeids` | `cd DEC && GOWORK=off go run -tags=research ./research/cmd_grenadeids -racine "<HOTE>/data/cache/film_chunks" -films 38900a5e` | 24 s, code 0 | Où les quatre identifiants de grenade apparaissent ; 132 lancers confirmés ; part des index auteur dans 0..7 par décalage (125/132 au décalage +0). N'écrit aucun fichier. Mesure de grenades de films anciens (lot 3.3) ; lien avec la piste C non établi. |

Non rejoués (entrées qui manquent sur ce poste ou hors du sujet), à ne pas compter comme disponibles :

- `research/divergence` : demande `DIVERGENCE_TEMOIN` = dossier de TSV de la sonde du témoin (libfilmshell),
  que l'agent ne construit pas ; c'est la comparaison paquet par paquet avec le témoin (piste D.2, voix du
  témoin : la sonde est « bâtie hors dépôt »).
- `research/cmd_mouvement`, `research/cmd_reapparition`, `research/cmd_rdata_weapon_scan` : lisent
  `HaloInfinite.exe` (`MOUV_EXE`, `REAP_EXE`) ou `world_dump.txt` ; pistes H (états de mouvement) et F.
- `replay/t0_mouvement_research_test.go` : demande `T0_API_TSV` (premiers instants de l'API) et un corpus de
  rejeux (`REPLAY_CORPUS`) : instrument de la piste A / de 0.4 (coup d'envoi), à alimenter par l'export 0.3.
- `replay/emprise_v0_porteurs_research_test.go` et `..._v1_...` : demandent le scratch du V0
  (`EMPRISE_V0_*`), non versionné (piste E).
- `replay/lettres_corpus_test.go`, `lettres_ordre_research_test.go`, `lettres_stabilite_test.go` : lettres des
  zones (piste G) ; films de Bases requis, `38900a5e` est un Team Slayer.
- `replay/visee_tir_research_test.go` (`tirFilmsDirEnv`, `tirTSVEnv`) : mesure de l'état de lunette sur le
  record de dégât, piste D.

## 4. Ce qu'il reste à écrire (pas d'instrument existant)

Constaté en dressant l'inventaire : aucun outil existant ne sort, pour un lot de films, les compteurs des
pistes (le « harnais de lot » de 0.6). Les briques disponibles : `replay-parc-vies` (lot, boucle un film par
enfant borné, reprise, comparaison avant/après de l'identité), `replay-equiv` (lot de 20 fixe), le gate du
corpus (19 témoins fixes), `replay-verite` et `replay-diff` (un film, deux artefacts). Un harnais de lot
réemploierait `replay-parc-vies --garder-documents` pour cuire N films (documents conservés) puis boucler
`replay-verite` et `replay-diff` sur les paires avant/après ; il n'écrirait que la boucle et l'extraction des
compteurs des pistes (aucun décodage neuf).
