# Lot G — Outils et catalogues (Go, docs) — journal d'exécution

> Exécuteur : agent Sonnet, worktree `C:\Users\Guillaume\Downloads\Scripts\LevelUp-wt-v2-outils`,
> branche `feat/v2-outils`. Plan source : `.ai/PLAN_V2_REJEU_FILM_2026-09-05.md` (section Lot G).
> Contrat : skill `plan-execution`.

## Items

### G.1 (H5) — Sentinelle mémoire unifiée sur `internal/filmproc.Arm` — [x]

`cmd/levelup/backfill_memlimit.go` et `cmd/replay-worker/memlimit.go` portaient chacun leur
propre copie de la sentinelle mémoire (double plafond souple/dur, échantillonnage 250 ms)
alors que `internal/filmproc.Arm` est canonique et déjà importé par les deux fichiers pour
d'autres besoins (`AcquireSolo`, `EmitPeak`, `LowerOwnPriority`). Vérifié sur pièces
(V-GO-D1 constat 6) : `internal/filmproc` n'a aucun import projet non-stdlib, donc aucun coût
de dépendance — la justification écrite des deux copies (« factoriser exigerait d'ouvrir un
paquet interne partagé ») était fausse au moment même où elle était écrite.

Fait :
- `cmd/levelup/backfill_memlimit.go` et `cmd/replay-worker/memlimit.go` supprimés entièrement.
- `cmd_backfill_replay_child.go` (enfant one-shot) et `cmd/replay-worker/job.go` (ouvrier
  long-vivant) arment désormais `filmproc.Arm(nomOutil, giB, onExceeded)` avec leur callback
  `onExceeded` propre, préservant les deux doctrines d'arrêt distinctes (protocole stdout +
  `os.Exit(filmproc.CodeMemory)` côté enfant one-shot ; rapport HTTP au serveur puis
  `os.Exit(3)` côté ouvrier). Mêmes limites numériques (3 GiB souple, +25 % dur,
  échantillonnage 250 ms) : le calcul vit désormais dans le seul `internal/filmproc/memguard.go`.
- Les deux constantes de défaut (`plafondMemoireDefautGiB`, `memGuardDefaultGiB`, toutes deux
  `= 3`) sont éliminées au profit de `filmproc.DefaultLimitGiB` aux deux sites d'appel du flag
  `-mem-limit-gib`.
- `internal/archlint/no_unbounded_film_loop_test.go` : `sentinelleTokens` réduit à
  `{"filmproc.Arm("}` (retrait de `"debug.SetMemoryLimit("}`), avec le commentaire réécrit —
  il entérinait la duplication comme « deux formes légitimes » au lieu de l'attraper. Le
  message d'erreur de `TestPointsDEntreeDeDecodageArmentUneSentinelle` mis à jour en
  cohérence. **Rectification (revue R1, constat C6)** : ce retrait NE FAISAIT PAS d'un
  `debug.SetMemoryLimit(` brut hors `internal/filmproc` une violation du ratchet, contrairement
  à ce que cette entrée affirmait — il l'empêchait seulement de compter comme preuve de
  sentinelle. Le ratchet qui tient la promesse est arrivé au correctif C6 ci-dessous
  (`TestPasDeSentinelleBruteHorsFilmproc`).
- Commentaires stale corrigés (référençaient les fichiers supprimés) : six mentions
  « cf. memlimit.go » dans `cmd/replay-worker/job.go`/`main.go`. Le commentaire de
  `internal/domain/build_queue.go` (doc de `BuildJobErrorCodeMemoryExceeded`), qui citait
  aussi les deux fichiers supprimés, avait été corrigé au même titre puis REVERTÉ : ce paquet
  est hors périmètre du lot G (réservé aux lots B/C) — cf. Découvertes.
- Deux petits helpers de mise en forme humaine (`libelleOctets` dans `cmd/levelup`,
  `formatMemGuardBytes` dans `cmd/replay-worker`) restent dupliqués entre les deux `main` —
  2 copies, sous le seuil de centralisation de la règle 6 ; leur constante `octetsParGiB` /
  `memGuardOctetsParGiB` relocalisée dans le fichier qui la consomme réellement, nom et valeur
  inchangés (aucune référence de test à mettre à jour).
- Tests des deux sentinelles copiées supprimés avec elles ; les tests de la logique de
  compte-rendu (motif mémoire explicite, message, requête HTTP) et les deux tests déplacés
  (`TestLibellePlafond`, `TestLibelleOctets`, dans `cmd_backfill_replay_passe_test.go`)
  survivent inchangés.

Gates : `go build ./cmd/levelup/... ./cmd/replay-worker/...` propre ;
`go test ./internal/archlint/... ./internal/filmproc/... ./cmd/levelup/... ./cmd/replay-worker/...`
→ `ok` sur les quatre paquets ; `golangci-lint run --new-from-merge-base=origin/main` sur ces
mêmes paquets + `internal/domain/...` → `0 issues.`

Commit : `14dd46fd3` — `v2(G.1): sentinelle memoire unifiee sur internal/filmproc.Arm`.

### G.2 (I2) — `internal/himap/heightfield.go` : code mort retiré — [x]

Vérifié sur pièces (grep sur tout `apps/go-api`) : `HeightField` (type, `NewHeightField`,
`AddMesh`, `faceMarchable`, `rasteriseTriangle`, `At`, `Cellule`, `Couverture`,
`MinNormalZWalkable`) n'avait plus aucun appelant hors de son propre test — approche de champ
d'altitude pour la seule surface marchable, abandonnée (mesure du 2026-08-08 sur Cliffhanger,
`HANDOFF_PORT_TRIANGLES_2026-08-08.md` §3), le problème qu'elle visait ayant été résolu
autrement par l'écrêtage des toits sur le rendu existant (`ecretage_toits.go`).

**Incident de méthode, corrigé en cours de route** : un premier essai de suppression totale
du fichier a cassé la compilation — `borne()` et `altitudeAuPoint()`, définies dans le même
fichier, sont en réalité des primitives de rasterisation PARTAGÉES, appelées par
`Rendu.triangleBorne` (`rendu.go`), `Rendu.poseReferenceTriangle` (`reference_navmesh.go`),
et par `sddt.go`/`volume.go`. Restauré depuis git, puis retraité en ne retirant QUE la partie
morte : `heightfield.go` renommé `projection_triangle.go` (`git mv`, historique préservé),
réduit de 175 à 47 lignes, avec un en-tête qui documente la mesure et ce qui a été retiré.
Même mésaventure sur `heightfield_test.go` (97 L) : son helper `instanceIdentite()` est
utilisé par cinq autres fichiers de test sans rapport (`rendu_test.go`,
`rendu_couleur_test.go`, `rendu_reference_test.go`, `sddt_test.go`, `volume_test.go`) —
déplacé vers `instance_identite_test.go`, même patron que `chemin_depot_test.go` déjà établi
dans ce paquet pour ce type d'extraction.

**Découverte (staging git, hors périmètre)** : le commit `b8528da7c` n'a d'abord enregistré
qu'un `git mv` PUR (100 % de similarité) — au moment du `git mv`, le fichier portait encore
son contenu d'origine ; la réécriture est arrivée après, et un `git add` multi-chemins a
échoué EN BLOC sur un pathspec déjà supprimé (`heightfield_test.go`), sans rien stager du
reste. Corrigé par un second commit (`b46b481b9`) qui apporte la réduction réellement voulue.
Leçon retenue pour la suite du lot : `git add` un chemin à la fois quand un `git mv`/`git rm`
a eu lieu juste avant, et revérifier `git diff HEAD` après chaque commit.

Gates : `go build ./internal/himap/...` propre (le premier essai avait échoué avec
`undefined: borne` / `undefined: altitudeAuPoint` — corrigé avant de committer) ;
`go vet ./internal/himap/...` propre ; `go test ./internal/himap/...` → `ok` (2,8 s, sans le
tag `gamefiles`) ; `golangci-lint run --new-from-merge-base=origin/main ./internal/himap/...`
→ `0 issues.` ; `git diff HEAD -- apps/go-api/internal/himap/` vide après le second commit
(HEAD = disque).

Commits : `b8528da7c` puis `b46b481b9` (correction) —
`v2(G.2): himap/heightfield.go, code mort retire (partie vivante deplacee)`.

### G.3 (H2) — `weapon-sounds -mode livrer` : portage Go de `_outils/livraison.py` — [x]

Dernier maillon Python hors dépôt de la chaîne des sons d'armes. Portage FIDÈLE (pas une
réécriture) de `_outils/livraison.py` (archive Desktop, hors dépôt, lu mais jamais modifié) :
mêmes structures JSON (`lot1.json`, `lot2.json`, `manifeste.json`, `coups.json`,
`votes-final.json`), même algorithme de choix de source (rôle confirmé, puis vote de coup
trié 1p avant 3p, puis vote d'événement en repli), même troncature passthrough (format
d'origine préservé, aucun échantillon décodé pour la copie/troncature), et un port BIT À BIT
du générateur pseudo-aléatoire de CPython (Mersenne Twister MT19937, graine `20260816`) pour
l'unique arme rendue par événement plutôt que copiée depuis un fichier voté
(`Covenant_provoker` → `hinf_ravager`).

Fichiers créés sous `apps/go-api/cmd/weapon-sounds/` : `livraison.go` (modèle JSON, `joli`,
votes, `choixDossier`), `livraison_variation.go` (filtrage de fourchette RANGED, tri des
dossiers candidats), `livraison_orchestrate.go` (chargement, boucle principale, génération
TS), `livraison_audio.go` (I/O WAV : index, troncature, lecture 16 bits, mixage par couches),
`livraison_mt19937.go` (le générateur), `livraison_test.go` (11 tests). Mode `livrer` câblé
dans `main.go` (flags `-donnees`/`-sons`/`-depot`, bullet dans l'en-tête de package).

**Preuve du générateur pseudo-aléatoire** : avant d'écrire le port, la séquence de référence
de CPython a été obtenue une fois (`python3 -c "import random; r=random.Random(20260816);
..."`) pour `getrandbits(32)` brut ET pour `choice()` sur plusieurs longueurs consécutives ;
le port Go reproduit cette séquence EXACTEMENT (vérifié dans un scratchpad isolé avant
intégration, puis figé en test de régression permanent : `TestMT19937_SequenceIdentiqueACPython`,
`TestMT19937_ChoiceIdentiqueACPython`).

**Preuve octet à octet, avec la limite honnête de ce poste** : les dossiers d'armes avec
leurs `.wav` sources réels ont disparu de la machine depuis la livraison du 2026-08-16 (seuls
`_donnees/*.json` et les scripts survivent — vérifié : `ls` à la racine du chantier ne montre
que `_donnees/`, `_outils/`, `_fin_partie/` (chantier annonceur sans rapport), `LISEZ-MOI.md`,
`TRIER.html`). La comparaison directe contre l'artefact de PRODUCTION versionné
(`static/sounds/halo_infinite/hinf_*.wav` actuels) n'a donc pas pu être rejouée depuis les
mêmes sources. À la place, conformément à la clause de repli du plan :
1. Un jeu d'entrées SYNTHÉTIQUE minimal a été construit dans le scratchpad de la session
   (hors dépôt) — cinq dossiers d'armes exerçant les quatre chemins de décision (rôle à
   rendre, rôle à copier, vote de coup, vote d'événement en repli) et le dédoublonnage
   variante/base, avec de vrais fichiers `.wav` PCM 16 bits/48 kHz.
2. Les DEUX scripts Python d'origine (copiés tels quels depuis l'archive Desktop, jamais
   modifiés — vérifié `cmp` avant et après) ont été exécutés sur ce jeu synthétique.
3. Le mode Go `livrer` a été exécuté sur le MÊME jeu synthétique.
4. Comparaison : les 4 fichiers `.wav` produits sont IDENTIQUES octet à octet (`cmp` +
   `md5sum`) entre les deux implémentations, y compris `hinf_ravager.wav` (le seul chemin qui
   dépend du générateur pseudo-aléatoire). `weaponSoundVariations.ts` ne diffère QUE sur les
   trois lignes d'en-tête « GENERE PAR » (mises à jour vers le mode Go, comme demandé) — le
   reste (données, import, export) est identique caractère pour caractère.
5. Exécuté aussi contre les VRAIES données `_donnees` de production (disponibles), sans les
   `.wav` sources manquants : le port Go et le script Python échouent avec le MÊME message
   (`rendu impossible Covenant_provoker bb31841b`), au même dossier (le premier traité par
   l'ordre de tri, qui place les rôles confirmés en tête) — preuve que le JSON de production
   réel (592 Ko de `lot1.json`, 817 Ko de `manifeste.json`, etc.) est lu et interprété de
   façon identique jusqu'au point exact où l'entrée manquante arrête les deux.

`[!]` **Item non couvert, justifié** : la comparaison octet à octet contre l'artefact de
production ACTUELLEMENT VERSIONNÉ (`git diff --exit-code` sur `static/sounds/halo_infinite/`
après une exécution réelle) n'a pas pu être produite — ressource externe (les fichiers `.wav`
sources du chantier) indisponible sur ce poste, condition explicitement anticipée par le
plan. La fidélité algorithmique est prouvée par ailleurs (ci-dessus) avec un niveau de preuve
qui dépasse ce qu'un simple examen de code aurait donné.

`.ai/V7.5/RECETTE_SONS_ARMES.md` mis à jour : tableau d'état des six scripts `_outils/`
(`akpk_unpack.py` et `livraison.py` portés en Go ; `conv_lot.py`, `coups_lot.py`,
`manifeste2.py`, `reinjecte.py` restent hors dépôt — ils alimentent le triage humain, pas les
assets versionnés finaux), section 1 et section 7 réécrites sur l'invocation Go, ancienne
invocation Python gardée pour mémoire (« NE PLUS L'UTILISER »).

Gates : `go build ./cmd/weapon-sounds/...` propre ; `go vet ./cmd/weapon-sounds/...` propre ;
`go test ./cmd/weapon-sounds/...` → `ok` (0,131 s, 11 nouveaux tests + suite existante) ;
`gofmt -l` sur les six fichiers → vide ; `golangci-lint run --new-from-merge-base=origin/main
./cmd/weapon-sounds/...` → `0 issues.`

Commit : `c82a4be94` —
`v2(G.3): weapon-sounds -mode livrer, portage Go de _outils/livraison.py`.

### G.4 (H3) — Section « Chaînes de fabrication des assets versionnés » — [x]

Onze chaînes de fabrication d'assets versionnés ne figuraient dans aucune documentation.
Chaque commande, sortie, prérequis et déclencheur de rejeu a été vérifié SUR PIÈCES avant
d'être écrit (lecture directe des `main.go` et fichiers voisins, comptage des fichiers
réellement commités via `git ls-files`) — pas de recopie du tableau de l'annexe G9 du
registre, qui s'est d'ailleurs révélée imprécise sur deux points (cf. Découvertes).

Neuf des onze chaînes ont été vérifiées par un agent de recherche dédié (lecture seule,
aucune modification), avec pour consigne explicite de citer les lignes de code exactes ; les
deux autres (`weapon-icons` build+table, `weapon-sounds`) étaient déjà connues de l'exécuteur
(section préexistante / travail du lot G.3) et `mappos-build`/`mapnav-fetch`/`vehicle-sprite`
ont été relues directement par l'exécuteur en parallèle pour ne pas rester inactif pendant la
recherche.

Remplace la section « Game-asset extraction » (EN) / « Extraction d'assets du jeu » (FR), qui
ne couvrait que `weapon-icons-build`, par une section élargie « Asset production chains
(versioned outputs) » / « Chaînes de fabrication des assets versionnés » qui l'inclut et
ajoute les dix autres : `weapon-icons-table`, `mapquant-build`, `mapcallouts-build`,
`mapfond-build`, `mapobj-build`, `mapopads-build`, `mapstruct-build`, `mappos-build`,
`mapnav-fetch`, `vehicle-sprite`, `weapon-sounds` (mode `livrer`).

Deux corrections apportées à l'annexe G9 en cours de vérification (cf. Découvertes) :
`map_objects.csv`/`forge_object_types.csv` n'ont aucun producteur automatisé (import manuel,
pas une chaîne) ; la sortie de `mapnav-fetch` (`.ai/re_dump/navmesh/`) est explicitement
ignorée par git — documentée comme cache de travail local, pas comme un douzième asset
versionné.

Parité EN/FR vérifiée : les 11 noms d'outils cités le même nombre de fois des deux côtés,
nombre de fences de code identique (70/70), aucun lien mort vers l'ancien titre de section
dans tout le dépôt.

Gates : les deux fichiers relisent proprement en Markdown (fences équilibrées) ; le hook
`docs-fr-sync` n'a émis aucun avertissement au commit (les deux fichiers stagés ensemble).

Commit : `d412c96a9` —
`v2(G.4): section Chaines de fabrication des assets versionnes (COMMANDS.md EN+FR)`.

## Gate G (clôture du lot)

Commandes exactes, jouées en avant-plan depuis `apps/go-api` avec
`GOCACHE=/c/Users/Guillaume/AppData/Local/go-build-v2-outils CGO_ENABLED=1` :

```
go build ./...
```
→ sortie vide (succès).

```
go test ./internal/himap/... ./internal/archlint/... ./internal/filmproc/... \
  ./cmd/levelup/... ./cmd/replay-worker/... ./cmd/weapon-sounds/...
```
→
```
ok  	levelup/go-api/internal/himap	15.956s
ok  	levelup/go-api/internal/archlint	18.180s
ok  	levelup/go-api/internal/filmproc	(cached)
ok  	levelup/go-api/cmd/levelup	(cached)
ok  	levelup/go-api/cmd/replay-worker	(cached)
ok  	levelup/go-api/cmd/weapon-sounds	(cached)
```

```
golangci-lint run --timeout 5m --new-from-merge-base=origin/main \
  ./internal/himap/... ./internal/archlint/... ./internal/filmproc/... \
  ./cmd/levelup/... ./cmd/replay-worker/... ./cmd/weapon-sounds/...
```
→ `0 issues.`

`lefthook` `docs-fr-sync` : satisfait à chacun des commits du lot qui touchent
`docs/COMMANDS.md` (aucun avertissement de désynchronisation EN/FR — les deux fichiers
toujours stagés ensemble).

**Correction post-gate** : en relisant le diff complet du lot (`git diff --stat
a21fd77f4..feat/v2-outils`) après la clôture ci-dessus, l'exécuteur a constaté que le commit
G.1 touchait un fichier hors périmètre (`internal/domain/build_queue.go`, réservé aux lots
B/C — cf. Découvertes). Reverté dans un commit dédié (`132967520`), pur retrait de commentaire
sans effet sur le code compilé — `go build ./...` et le hook `go-vet` du commit (module
entier) restent propres après ce commit. Le Gate G ci-dessus n'a pas eu besoin d'être rejoué :
aucun des six paquets qu'il couvre n'est affecté par ce fichier.

Tous les quatre items du lot sont statués `[x]`, gate passé sans allowlist nouvelle, sans
test désactivé.

**Surveillance CI (partielle, arrêtée sur consigne)** : après le push final
(`2b9afd7af`), le run `https://github.com/JGtm/LevelUp/actions/runs/33997437532` a été
observé jusqu'à interruption explicite de la consigne (contrainte de quota, l'intégrateur
vérifiera la CI à l'integration). État constaté avant l'arrêt, tous verts :
`Go Lint (golangci-lint)`, `OpenAPI Lint`, `Go Build + Test (windows-latest)`,
`Go Build + Test (ubuntu-latest)`, `Frontend (TypeScript + Vite build)`,
`Go Lease Enforcement (ADR 0013)`, `Go Contract Test (OpenAPI YAML)`, `Secrets (gitleaks)`,
`Deploy Pre-Check`. Encore en cours au moment de l'arrêt, verdict NON CONNU :
`Go Coverage + Baseline non-régression` (le job long — tests CGO + intégration sur tout le
module) et `E2E React (Playwright)` (pas encore démarré). Risque évalué avant l'arrêt pour le
job de couverture : les 14 lignes de `scripts/coverage_baseline.txt` (ratchet par fonction) et
le seuil global `apps/go-api/coverage_baseline.txt` (69.0 %) ne référencent aucun fichier du
lot — aucun test supprimé par G.1/G.2 n'apparaît dans `.ai/baselines/tests_pre_migration.jsonl`
(vérifié par grep, zéro occurrence sur 14 noms de test). Risque jugé faible mais NON CONFIRMÉ.

## Découvertes (hors périmètre, consignées, non traitées)

- **Mojibake Bash heredoc → `git commit -m`** : un seul caractère accentué français
  (« entérinait ») tapé dans un message de commit passé par heredoc Bash (`git commit -m
  "$(cat <<'EOF' ... EOF)"`) est ressorti corrompu (`entรฉrinait`) dans le message de commit
  final (`14dd46fd3`), alors que la même méthode pour des dizaines d'autres mots
  non-accentués n'a rien corrompu. Non corrigé (commit déjà créé, coût de l'amend jugé
  supérieur au bénéfice pour un seul mot de prose de message de commit — pas de code
  affecté). Leçon appliquée pour les commits suivants du lot : éviter les caractères
  accentués dans les messages de commit générés par heredoc, écrire dans le style
  sans-accents déjà dominant dans les commentaires Go du dépôt.
- **`git add` multi-chemins avorte EN BLOC sur un pathspec invalide** : si un des chemins
  passés à un seul appel `git add a b c d` ne correspond à rien (fichier déjà supprimé par un
  `git rm` antérieur, par exemple), la commande entière échoue SANS rien stager des autres
  chemins valides — y compris ceux qui avaient une modification de contenu en attente après
  un `git mv`. À l'origine de l'incident du G.2 (voir ci-dessus). Leçon : après tout
  `git mv`/`git rm` suivi d'une réécriture de contenu, stager le chemin réécrit SEUL, et
  vérifier `git status --short` avant chaque commit.
- **`internal/analysis/replay/geometry.go` / `map_geometry/*.csv`** : vérifié en marge de
  G.4 (pour ne pas documenter à tort `mapobj-build` comme producteur) — ces deux CSV
  (`map_objects.csv`, `forge_object_types.csv`) n'ont AUCUN producteur automatisé dans
  `cmd/` (grep : zéro référence dans `cmd/mapobj-build/*.go` ou ailleurs hors lecteurs). Le
  registre G10 le confirmait déjà (« import manuel ») mais l'annexe G9 du registre les
  attribuait à tort à la chaîne `mapobj-build` dans son tableau des commandes. Documenté avec
  précision dans G.4 (pas de commande de régénération pour ces deux fichiers), non retraité
  ailleurs (hors périmètre fermé du lot).
- **`cmd/vehicle-sprite` : pas une chaîne à commande unique** — contrairement aux dix autres
  chaînes de fabrication (une commande, une sortie), `vehicle-sprite` est un outil à
  sous-commandes (`inventaire`/`render`/`variantes`/`diag`/`assemble`/`compose2d`) dont la
  recette exacte pour reproduire les 18 PNG actuellement versionnés a évolué par vagues
  successives documentées dans plusieurs notes `.ai/V7.5/film_re/*.md` (la plus complète
  trouvée, `V4_RAPPORT_SPRITES_2026-08-31.md` §9, ne couvre que 13 des 18 véhicules actuels).
  Documenté honnêtement dans G.4 avec le motif vérifié et un pointeur vers ces notes, sans
  prétendre à une commande unique reproduisant l'état actuel complet — hors périmètre de
  reconstituer cette recette complète dans ce lot.
- **`.ai/re_dump/navmesh/` (sortie de `mapnav-fetch`) est GITIGNORÉE, pas versionnée** —
  vérifié `.gitignore:254` + `git ls-files` (0 fichier) alors que l'annexe G9 range
  `mapnav-fetch` sans réserve parmi les producteurs d'assets « versionnés ». Corrigé dans la
  documentation G.4 (présenté comme cache de travail local qui alimente `mapfond-build`, pas
  comme un asset commité) ; non retraité au-delà (hors périmètre : ce n'est pas un défaut de
  code, juste une imprécision de classement de l'audit).
- **`mapopads-build` n'est pas le seul écrivain de `map_weapon_pads.json`** — la même sortie
  est aussi touchée par le rattrapage Forge du runtime de synchro
  (`internal/sync/replayartifacts/mvar_rattrapage.go`), sujet de l'item A.3 du plan (lot A,
  hors périmètre de ce lot). Mentionné en une ligne dans l'entrée de doc G.4 pour ne pas
  laisser croire que cet outil est l'unique écrivain, sans reprendre le fond du sujet (couches
  ART, overlay non versionné) qui appartient au lot A.
- **`internal/domain/build_queue.go` : commentaire stale non corrigé (périmètre)** — la doc de
  `BuildJobErrorCodeMemoryExceeded` cite encore `cmd/replay-worker/memlimit.go` et
  `cmd/levelup/backfill_memlimit.go`, tous deux supprimés par G.1. `internal/domain/` est
  explicitement réservé aux lots B/C par le plan : la correction faite par erreur dans le
  commit G.1 a été revertée (`132967520`) dès que l'exécuteur l'a repérée en relisant le diff
  complet du lot. Le lot qui touchera ce fichier peut mettre à jour ce commentaire au passage
  (même correction que celle déjà faite dans `cmd/replay-worker/job.go`).

## Réparation CI (2026-09-06) — `TestJoliDossier` rouge sous ubuntu-latest

**Signalement du coordinateur** : job `Go Coverage + Baseline` du run `33997437532` rouge —
`TestJoliDossier` échoue sous Linux avec `joliDossier("C:\Steam\SFX\sb_010_wea_un_
assaultrifle.pck") = "C:\Steam\SFX\wea_un_assaultrifle"` au lieu de `"UNSC_assaultrifle"`
(et 3 cas semblables — les 4 entrées de la table de test).

**Cause, vérifiée sur pièces** : `joliDossier` construisait le nom de fichier nu via
`strings.TrimSuffix(filepath.Base(pck), filepath.Ext(pck))`. `path/filepath` de la stdlib Go
est dépendant de l'OS DE COMPILATION : sous Linux, `filepath.Base` ne coupe que sur `/`, pas
sur `\`. Or les chemins de `lot1.json`/`lot2.json` sont TOUJOURS des chemins Windows (machine
d'extraction du chantier, confirmé par un `jq` sur les fixtures réelles :
`"C:\Program Files (x86)\Steam\...\sb_010_tur_bt_gatlingmortar.pck"`), quelle que soit la
plateforme qui exécute le binaire. Sous Linux, `filepath.Base` rendait donc le chemin ENTIER
non coupé, la regex `sb_010_(wea|tur|whizby)_...` ne matchait jamais, et le repli
`strings.ReplaceAll(base, "sb_010_", "")` laissait le préfixe de répertoire Windows dans le
résultat — bug latent RÉEL (pas seulement un artefact de test) : le mode `livrer` aurait
produit un catalogue faux sur toute machine Linux traitant les vraies données du chantier.

**Correctif** : nouvelle fonction `joliBaseSansExt` qui coupe sur le DERNIER `/` OU `\`
(`strings.LastIndexAny`) puis sur le dernier `.`, INDÉPENDAMMENT de l'OS de compilation — port
fidèle de `os.path.basename`/`splitext` tels qu'ils se comportent sous Windows (module
`ntpath` : les deux séparateurs sont valides), le seul OS sur lequel `_outils/livraison.py` a
jamais tourné. `joliDossier` délègue à cette fonction ; `path/filepath` reste importé dans
`livraison.go` (toujours utilisé par `livraisonChoixParRole` pour `filepath.Rel`/`ToSlash` sur
de VRAIS chemins du système de fichiers local, un cas différent qui n'a pas ce défaut).
Nouveau test permanent `TestJoliDossier_IndependantDuSeparateur` (séparateurs mixtes, nom nu
sans répertoire, forme `/` et forme `\` côte à côte) — le test original (entrées `\` telles
quelles) n'a pas eu besoin d'être modifié, seule la fonction sous-jacente était en cause.

**Vérifications** : `GOOS=linux go vet ./cmd/weapon-sounds/...` n'a pas pu être joué
jusqu'au bout sur ce poste — `cmd/weapon-sounds` importe transitivement `internal/ooz`
(décompression Kraken, CGO), et ce Windows ne dispose d'aucune chaîne de cross-compilation
vers Linux (`CGO_ENABLED=1 GOOS=linux` échoue sur des en-têtes POSIX manquantes dans le gcc
MinGW local — limite d'environnement, pas un défaut de code). `go vet` n'aurait de toute
façon pas pu détecter cette classe de bug (différence de COMPORTEMENT runtime d'un paquet
stdlib selon l'OS, pas une construction suspecte). Preuve retenue à la place : test unitaire
explicite avec des chemins `/` ET `\` mélangés, qui passe sur CE poste (Windows) exactement
comme il passerait sur Linux — la fonction ne dépend plus de `path/filepath` du tout pour cette
opération, donc plus d'aucun comportement spécifique à l'OS de compilation.
`grep -n '\\' cmd/weapon-sounds/*_test.go cmd/levelup/*_test.go internal/himap/*_test.go` :
les seules occurrences pertinentes sont les deux tests `TestJoliDossier*` (désormais corrects
sur les deux séparateurs) ; le reste est soit un faux positif (`\n`, `\s`, `\"` dans des
chaînes/regex sans rapport), soit un fichier `*_gamefiles_test.go` PRÉEXISTANT et hors
périmètre du lot (`internal/himap/sonde_locs_gamefiles_test.go`, un chemin Windows codé en
dur pour une machine de développement spécifique, derrière le tag `gamefiles` donc jamais
exécuté en CI).

Gates rejoués (avant-plan, `GOCACHE`/`GOLANGCI_LINT_CACHE` dédiés au lot) :
`go build ./cmd/weapon-sounds/...` propre ; `go test ./cmd/weapon-sounds/... -run
"TestJoliDossier|TestLivraison|TestMT19937"` → 12 tests PASS ; `go test -count=1
./internal/himap/... ./internal/archlint/... ./internal/filmproc/... ./cmd/levelup/...
./cmd/replay-worker/... ./cmd/weapon-sounds/...` → `ok` sur les 6 paquets ;
`golangci-lint run --new-from-merge-base=origin/main` sur ces 6 paquets (avec
`GOLANGCI_LINT_CACHE=/c/Users/Guillaume/AppData/Local/golangci-v2-outils`) → `0 issues.` ;
`go build ./...` a échoué une fois sur une erreur de linker sans rapport avec ce lot
(`cmd/rebuild_shared_social`, `cmd/rebuild_pme_art`, `cmd/refresh-metadata`... — collision de
répertoire de travail du linker, probablement des builds Go concurrents d'autres agents du
même poste), propre au second essai.

Commit : `df896e5b1` — `v2(G.3-fix): joliDossier portable, independant de l OS de compilation`.

## Corrections après revue (revue adversariale R1, 2026-09-06)

> Verdict source : ronde 1, dix constats recevables (aucun P0), trois P1 et sept P2. Périmètre
> FERMÉ à ces dix points ; un commit par point, préfixe `v2(G.fix-n)`, chacun prouvé rouge puis
> vert avant d'être coché. Le script Python d'origine (`_outils/livraison.py` et
> `_outils/coups_lot.py`, archive Desktop hors dépôt) a été EXÉCUTÉ pour produire les sorties de
> référence, jamais modifié — copies vérifiées au `cmp` avant et après chaque exécution.

### C1 (P1) — le mode `livrer` exigeait le jeu installé — [x]

`main.go` résolvait la racine `deploy` ET construisait l'index large des `.pck` AVANT le
`switch *mode`, donc pour tous les modes. `livrer`, qui n'ouvre aucun module du jeu, mourait
`racine deploy introuvable` sur un poste sans Halo et payait en plus une indexation de 15 798
identifiants `.wem` inutiles. Résolution paresseuse via la table `modesSansJeu` (une seule
entrée, `livrer` : c'est le seul mode que le constat couvre).

Le second prérequis nié par la doc, cgo, est en revanche RÉEL et n'a pas été retiré :
`cmd/weapon-sounds` importe `internal/himap` → `internal/himodule` → `internal/ooz`
(décompression Kraken) pour ses autres modes, et `CGO_ENABLED=0 go build ./cmd/weapon-sounds`
échoue sur `build constraints exclude all Go files`. L'isoler demanderait de scinder ce `main` —
hors périmètre : `docs/COMMANDS.md:317` et `docs/FR/COMMANDS.md:331` le DISENT désormais au lieu
de le nier (option explicitement offerte par la consigne, « l'un ou l'autre, pas une doc fausse »).

Preuve : `LEVELUP_HALO_DEPLOY="C:\pas\de\jeu\ici" go run ./cmd/weapon-sounds -mode livrer
-donnees <jeu synthétique>` → avant : `racine deploy introuvable`, exit 1, rien d'écrit ; après :
le mode s'exécute de bout en bout (exit 0, six armes livrées). Commit `39d91d0d4`.

### C2 (P1) — l'en-tête « GENERE PAR » du fichier versionné — [x]

`apps/web/src/features/match-replay/weaponSoundVariations.ts` annonçait encore
`_outils/livraison.py`, l'outil que la recette déclare dans le même lot « NE PLUS L'UTILISER » :
doc inversée (CLAUDE.md, diagnostic n°9). Trois lignes recopiées du gabarit Go
(`livraisonTSTemplate`), rien d'autre touché — `git diff` : 3 lignes.

Le fichier ne peut pas être régénéré sur ce poste ; `TestEnTeteTSVersionneeSuitLeGabarit` tient
la promesse à sa place et compare, octet pour octet, l'en-tête versionné à celui que `livrer`
produirait. Preuve : test rouge sur l'en-tête d'avant (les deux en-têtes imprimés côte à côte),
vert après. Commit `d37d49c97`.

### C3 (P1) — aucun test ne gardait la fidélité octet à octet — [x]

`TestLivrerOctetPourOctet` (`livraison_golden_test.go`), avec deux moitiés :

- le JEU D'ENTRÉES est GÉNÉRÉ par le test (`livraison_synth_test.go`, 222 L) : les `.wav` du
  chantier n'existent plus sur aucun poste et pesaient des centaines de Mo. Il exerce les deux
  rôles confirmés, le vote de coup trié (1p avant 3p), le vote d'événement en repli, le
  dédoublonnage variante/base, `INTROUVABLE`, `SANS FICHIER`, mono et stéréo, 44,1 kHz et
  48 kHz, `_EMBARQUES`, la couche sautée sans consommer le générateur et la couche rejetée
  après consommation, l'exemple de vote vide (C7), la fourchette RANGED dégénérée, le chemin
  relatif au lecteur (C8) et tous les formats de nombre (C9) ;
- les SORTIES ATTENDUES ne sont pas générées : `testdata/livraison/goldens/` (301 Ko) a été
  produit UNE FOIS par le script Python d'origine sur cette même arborescence — six `.wav`, le
  `weaponSoundVariations.ts` et la sortie console. La recette pour les refaire, et
  l'interdiction de les régénérer avec le code Go (« le test ne dirait plus que Go est d'accord
  avec Go »), sont écrites en tête du test. `.gitattributes` force `*.wav binary` : une seule
  substitution CRLF→LF dans un flux PCM ferait rougir le test sur un checkout Windows.

La console est comparée elle aussi — elle attrape ce que les octets ne disent pas : ordre de
livraison, source retenue par arme, lignes `INTROUVABLE`/`SANS FICHIER`/« variante servie par la
base », colonnage. La seule divergence VOULUE (les trois lignes d'en-tête du `.ts`) est comparée
au gabarit Go plutôt qu'effacée.

Preuve : mutation M3 du verdict rejouée (`int16(v*att)` → `int16(math.Round(v*att))` ET
`livraisonDureeLivreeS` 1.2 → 1.25) → ROUGE sur `hinf_ma40_ar.wav` (110 294 octets au lieu de
105 884) et `hinf_ravager.wav` (même taille, octets différents) — exactement les deux fichiers
annoncés par le relecteur ; verte après retrait. Commit `4253a8d3a`.

### C4 (P2) — le pic mémoire n'était plus ré-échantillonné à la sortie — [x]

`filmproc.EmitPeak(g.Peak())` émettait ZÉRO quand l'enfant meurt avant le premier tick (250 ms) :
verrou de décodage refusé, constructeur indisponible, toute sortie très précoce. Le récap de la
passe imprimait « (pic inconnu) ». `Peak()` note désormais l'empreinte courante avant de rendre
le maximum, exactement comme l'ancien `picObserve` supprimé par G.1 ; les huit sites qui
appellent `Peak()` dans un différé en bénéficient sans changer une ligne.

Preuve : `TestPeakReechantillonneALAppel` (sentinelle à période d'une heure, donc aucun tick) —
rouge avant (« pic rendu avant le premier echantillonnage = 0, attendu 4242 »), vert après.
Commit `09eab4b47`.

### C5 (P2) — les messages de journal de la sentinelle — [x]

`filmproc.WithArmMessage(...)`, option variadique : le texte de la ligne d'armement redevient le
choix de l'appelant. Les deux textes d'avant sont restaurés MOT POUR MOT — « plafond memoire
arme » (`cmd_backfill_replay_child.go`) et « replay-worker: plafond memoire arme pour ce job »
(`replay-worker/job.go`) — et le défaut (« plafond memoire arme pour ce decodage ») sert les six
autres binaires, qui n'en avaient jamais eu d'autre. Le jeton `filmproc.Arm(` du ratchet archlint
est inchangé (l'option est variadique, pas un nouveau point d'entrée).

`filmproc.HardLimitFor(giB)` rend le plafond dur sans recopier la marge de 25 % : c'est ce qui
remet `plafond_dur_octets` sur la ligne fatale de l'enfant sans réintroduire la duplication que
G.1 avait supprimée (un accès via `Guard` depuis le callback aurait été une course sur la
variable que `Arm` n'a pas encore rendue).

Lecture retenue de la consigne : « restaure les textes exacts d'avant » porte sur les deux
messages d'ARMEMENT qu'elle nomme ; sur la ligne FATALE, la consigne ne demande que le champ
`plafond_dur_octets`. Le préfixe « backfill-replay (enfant): » du message fatal est donc conservé
(il est cohérent avec toutes les autres lignes de ce fichier, et `PLAFOND MEMOIRE DEPASSE` reste
grepable), de même que `match_id`, qui s'ajoute au champ restauré au lieu de le remplacer.

Preuve : comparaison ligne à ligne avec `git show a21fd77f4:apps/go-api/cmd/levelup/backfill_memlimit.go`
(`slog.Info("plafond memoire arme", "souple_gib", …, "dur_octets", …)` ; fatale :
`"empreinte_octets", v, "plafond_dur_octets", s.plafondDur`) et
`a21fd77f4:apps/go-api/cmd/replay-worker/memlimit.go`. `TestArmJournaliseLeMessageDeLAppelant`
capture le journal (handler `slog` de test, plafond souple du processus restauré) et vérifie
message ET champs ; il ne compilait pas avant l'option. Commit `60e7c4aff`.

### C6 (P2) — le ratchet promis contre `debug.SetMemoryLimit(` brut — [x]

Le journal de G.1 sur-promettait : retirer ce jeton de `sentinelleTokens` l'empêchait seulement
de COMPTER comme preuve de sentinelle, aucune règle ne le rejetait (mutation M2 verte). Choix
retenu : le ratchet, pas la correction du journal.

`TestPasDeSentinelleBruteHorsFilmproc` balaie `apps/go-api` (fichiers de test compris), rejette
le jeton hors de `internal/filmproc/`, avec une allowlist datée et VIDE. Deux garde-fous du
ratchet lui-même : il s'exclut explicitement (il porte le jeton en clair dans sa constante et
dans son message — l'épeler par morceaux pour y échapper cacherait ce qu'un lecteur doit pouvoir
grep), et il rougit aussi si le paquet canonique cesse de poser un plafond souple, pour ne pas
garder une porte qui n'existe plus. Le commentaire de `sentinelleTokens` pointe désormais le
test qui tient la promesse.

Preuve : mutation M2 rejouée (`debug.SetMemoryLimit(3 << 30)` ajouté dans `cmd/replay-worker/`,
`filmproc.Arm` conservé) → ROUGE, violation nommée avec fichier et ligne ; verte après retrait.
Commit `0fb83a65d`.

### C7 (P2) — exemple de vote vide, et publication tout ou rien — [x]

Deux défauts indépendants réunis par un même scénario.

1. `livraisonFichierDuVote` rendait `("", true)` quand `exemples_retenus[0]` vaut la chaîne
   vide. Python rend cette valeur FAUSSE et ses appelants sautent le vote pour passer au
   SUIVANT — jamais un repli sur `exemples_proposes` du même vote. Go construisait la source
   `"<dossier>/"`, que `os.Stat` accepte (c'est un répertoire) et que la troncature refuse.
2. La cible était vidée de ses `hinf_*.wav` AVANT toute production, comme dans le script Python.
   Le lot est désormais produit dans un répertoire d'attente VOISIN de la cible (même volume,
   publication par renommage), et l'ordre est inversé à la publication : déplacer d'abord,
   retirer ensuite les armes que le nouveau lot ne remplace pas — la cible passe de l'ancien lot
   à « ancien plus nouveau » puis au nouveau, sans jamais être vide. État final identique, miroir
   toujours strictement limité au préfixe `hinf_`.

Preuves : `TestLivraisonChoixDossier_ExempleVideSauteLeVote` et `…DEvenement` rouges sur
l'ancienne fonction (« Source = "Arme/" »), verts après ;
`TestLivrerNEffaceRienQuandLaProductionEchoue` (source corrompue au milieu du lot) rouge dès
qu'on remet le vidage avant la production (« hinf_perime.wav a disparu »), vert après ; bout en
bout, le mode passait de `echec: read …\Covenant_needler` avec cible à moitié vidée, à exit 0 et
six armes livrées. Commit `52cc3c97f`.

### C8 (P2) — `joliBaseSansExt` n'était pas le port de `ntpath` promis — [x]

Le commentaire promettait `os.path.basename`/`splitext` « tels qu'ils se comportent sous
Windows » ; la fonction coupait sur le dernier séparateur et le dernier point, sans le préfixe de
lecteur ni la règle du point de tête. Option retenue : implémenter (pas retirer la promesse).
`livraisonNTSplitDrive` (lettre de lecteur, UNC, périphérique `\\.\`, préfixe étendu
`\\?\UNC\`), `livraisonNTBasename`, `livraisonNTSansExtension`.

Preuve : `TestJoliBaseSansExt_FideleANtpath`, table de 30 entrées entièrement relevée sur la
sortie RÉELLE de `ntpath.splitext(ntpath.basename(p))[0]` sous CPython 3.12 (sonde hors dépôt) —
10 rouges sur l'ancienne implémentation, dont les deux formes du constat
(`C:sb_010_wea_un_relatifdrive.pck` → `C:wea_un_relatifdrive` au lieu de `UNSC_relatifdrive` ;
`.pck` → `""` au lieu de `.pck`) ; vert après. `TestJoliDossier_FormesLimites` montre la
conséquence produit. Le jeu synthétique du golden porte un `pck` relatif au lecteur : la
régression se verrait aussi de bout en bout. Commit `21e5dac3d`.

### C9 (P2) — le formatage des nombres du `.ts` — [x]

`strconv.FormatFloat(v, 'g', -1, 64)` n'est ni `str(int)` ni `repr(float)` de Python.
`livraisonReprFloatPy` reproduit `repr` : plus courte écriture qui redonne la valeur, toujours
pointée, exposant seulement hors de [1e-4, 1e16) (règle `decpt <= -4 || decpt > 16` de
`format_float_short`), signe et deux chiffres au moins à l'exposant. La branche entière normalise
le zéro négatif (`-0` → `0`) et laisse intacts les entiers hors `int64`.

Preuve : sonde de 480 littéraux (60 choisis + 420 tirés au sort) confrontée à
`"%s" % json.loads(litteral)` sous CPython 3.12 — 144 divergences avant, 0 après. Les 56
littéraux qui portent une règle sont figés dans `TestLivraisonFormatNombrePy`, qui ne probait que
`-48`, `800`, `0` et `-3.5` — les quatre cas qui coïncidaient. Commit `8b5ba31f4`.

### C10 (P2) — l'erreur de `DirEntry.Info()` avalée — [x]

Option retenue : REMONTER l'erreur, ce qui est aussi le port fidèle (`sum(os.path.getsize(...))`
lève en Python). La mesure passe par `os.Stat` et se fait APRÈS l'impression de la ligne de
compte — l'ordre des deux `print()` de `livraison.py`, pour que le chemin d'erreur ait la même
sortie que lui. Preuve : la sortie console du mode sur le jeu synthétique reste identique octet
pour octet à celle du script Python (comparée par `TestLivrerOctetPourOctet`). Commit `f2ba8c4a1`.

### Gate des corrections

Commandes exactes, en avant-plan, depuis `apps/go-api`, avec
`GOCACHE=/c/Users/Guillaume/AppData/Local/go-build-v2-outils`,
`GOLANGCI_LINT_CACHE=/c/Users/Guillaume/AppData/Local/golangci-v2-outils`, `CGO_ENABLED=1` :

```
go build ./...
```
→ sortie vide (succès).

```
go test -count=1 ./cmd/weapon-sounds/... ./cmd/levelup/... ./cmd/replay-worker/... \
  ./internal/filmproc/... ./internal/archlint/...
```
→
```
ok  	levelup/go-api/cmd/weapon-sounds	0.242s
ok  	levelup/go-api/cmd/levelup	0.709s
ok  	levelup/go-api/cmd/replay-worker	0.165s
ok  	levelup/go-api/internal/filmproc	2.334s
ok  	levelup/go-api/internal/archlint	28.076s
```

```
golangci-lint run --timeout 15m --new-from-merge-base=origin/main ./...
```
→ `0 issues.` (précédé du warning préexistant du dépôt sur les directives `//nolint:gosec`,
cf. Découvertes).

`lefthook` (gofmt, check-merge-conflict, docs-fr-sync, gitleaks, go-vet) : vert à chacun des dix
commits ; `docs-fr-sync` satisfait au commit qui touche `docs/COMMANDS.md` (EN et FR stagés
ensemble).

### Découvertes de la passe de corrections (hors périmètre, non traitées)

- **`//nolint:gosec` désigne un linter NON ACTIVÉ.** `apps/go-api/.golangci.yml` n'active pas
  `gosec` (ni dans `default: standard`, ni dans `linters.enable`) ; les cinq et quelques
  directives `//nolint:gosec` du dépôt sont donc inertes et font émettre à chaque exécution
  `Found unknown linters in //nolint directives: gosec …`. La directive que cette passe avait
  ajoutée par mimétisme a été retirée ; les autres (dont
  `internal/archlint/no_unbounded_film_loop_test.go:241`) restent — hors périmètre.
- **`golangci-lint` prend un verrou GLOBAL** (« parallel golangci-lint is running ») malgré des
  `GOLANGCI_LINT_CACHE` distincts par lot : deux worktrees qui lintent en même temps se
  bloquent. Contournement utilisé : réessayer jusqu'à obtention du verrou (deux essais ici).
  À savoir pour tout exécuteur de lot parallèle.
- **Deux autres modes de `weapon-sounds` n'ouvrent aucun module du jeu** (`pck-dump`,
  `mesurer-wav`) et paient encore la résolution de la racine `deploy` et l'index large. Le
  constat C1 ne porte que sur `livrer` : `modesSansJeu` n'a donc qu'une entrée, et l'extension
  est laissée au lot qui touchera ces modes.
- **`livraisonRapportFinal` compte le préfixe `hinf_` sans exiger `.wav`**, là où le miroir
  exige les deux : c'est fidèle à `livraison.py` (`f.startswith("hinf_")` pour le bilan,
  `startswith and endswith` pour la suppression). Deux prédicats voisins mais distincts,
  volontairement non fusionnés.

## Questions ouvertes

- Aucune question bloquante identifiée. Point d'attention pour un lecteur futur de
  `docs/COMMANDS.md` : la section G.4 documente l'état constaté le 2026-09-06 ; certaines
  chaînes (`mapstruct-build` en particulier, cf. le retrait différé du champ `structure`) sont
  susceptibles de changer de statut lors d'un futur lot — la doc le signale mais ne peut pas
  se mettre à jour toute seule.
