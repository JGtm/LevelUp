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
  il entérinait la duplication comme « deux formes légitimes » au lieu de l'attraper ; un
  `debug.SetMemoryLimit(` brut hors `internal/filmproc` est désormais une violation du
  ratchet. Le message d'erreur de `TestPointsDEntreeDeDecodageArmentUneSentinelle` mis à jour
  en cohérence.
- Commentaires stale corrigés (référençaient les fichiers supprimés) :
  `internal/domain/build_queue.go` (doc de `BuildJobErrorCodeMemoryExceeded`), six mentions
  « cf. memlimit.go » dans `cmd/replay-worker/job.go`/`main.go`.
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

`lefthook` `docs-fr-sync` : satisfait à chacun des quatre commits du lot (aucun avertissement
de désynchronisation EN/FR — `docs/COMMANDS.md` et `docs/FR/COMMANDS.md` toujours stagés
ensemble dans le commit G.4, seul commit touchant un guide bilingue majeur).

Tous les quatre items du lot sont statués `[x]`, gate passé sans allowlist nouvelle, sans
test désactivé.

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

## Questions ouvertes

- Aucune question bloquante identifiée. Point d'attention pour un lecteur futur de
  `docs/COMMANDS.md` : la section G.4 documente l'état constaté le 2026-09-06 ; certaines
  chaînes (`mapstruct-build` en particulier, cf. le retrait différé du champ `structure`) sont
  susceptibles de changer de statut lors d'un futur lot — la doc le signale mais ne peut pas
  se mettre à jour toute seule.
