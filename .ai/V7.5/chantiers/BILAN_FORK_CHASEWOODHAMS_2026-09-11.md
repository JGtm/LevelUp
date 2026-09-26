# Bilan du fork ChaseWoodhams/LevelUp — ce qui vaut d'être repris (2026-09-11)

> Document d'inventaire, pas un plan. Chaque point dit : le défaut que le fork a constaté,
> sa mesure, son correctif, et l'état de NOTRE arbre vérifié sur pièces le 2026-09-11
> (branche `feat/citations-artilleur-vehicules`, HEAD `3eaf857f8`). À relire après le
> chantier en cours ; si un point est toujours valide, il devient un plan.

## Contexte

| | |
|---|---|
| Fork | https://github.com/ChaseWoodhams/LevelUp (remote git local : `fork`) |
| Base du fork | `98bd7c143` = tag **7.3.1**, ancêtre de notre `origin/main` |
| Branche utile | `feat/study-path-resolution` (`1b16dc20c`, PR #27, 211 fichiers, +49 576) |
| Divergence | notre HEAD a **2 570 commits** depuis cette base, dont **615** sur `internal/analysis/replay` + `filmdec` |
| Conséquence | **aucun merge ni cherry-pick possible** : ce sont des IDÉES et des MESURES à réimplémenter, pas des commits à prendre |

Ce que le fork construit (hors sujet pour nous) : un « outil d'étude » pour joueur solo —
archiveur de films de pros (`cmd/study-archiver`), serveur dédié (`cmd/study-server`), appli
Vite séparée `apps/study/` qui COPIE nos modules de rejeu, puis heat maps agrégées (issues
#19–#26, toutes ouvertes, rien de livré). Le fork ignore nos règles transverses (title-agnostic
non respecté dans `study_paths.go`, doublon de viewer).

PR #28 « English-only conversion » (877 fichiers) : supprime le français partout. Contraire à
notre règle n°1 ; **rien à reprendre comme conversion**. Elle a en revanche servi de détecteur
de littéraux FR en dur — voir point 6.

Méthode : lecture des diffs `98bd7c143..1b16dc20c` sur les fichiers hors `apps/study`, des
entrées `.ai/thought_log.md` du fork (2026-09-05 et 2026-09-06), puis grep/lecture de nos
fichiers correspondants. Les chiffres cités sont CEUX DU FORK, mesurés sur ses films
(4 à 6 films Streets archivés, 175 484 positions) — pas revérifiés chez nous.

Commits du fork à consulter (`git show <sha>`) :

| SHA | Objet |
|---|---|
| `1a7946bb3` | grenades (auteur juge), projectiles (repli Y), horloge du match, lives (une track = une vie) |
| `37591d6de` | témoin d'arme (`lives_witness.go`) + compteurs de contrôle dans `OwnerReport` |
| `2a8b21a51` | bancs de mesure `filmdec` : en-tête keyframe par type, désignateur d'équipe |
| `621f69040` | props Forge PAR CARTE (`MapGeometryDir` prend un module) |
| `677574859` (PR #28) | traduction des ~70 littéraux FR en dur côté web |

---

## Point 1 — Grenades : le lancer prenait le projectile de l'AUTRE joueur

**Fichier fork** : `apps/go-api/internal/analysis/replay/grenades.go` (`locateThrow`,
`authorBiped`, `birthForThrow`, `birthsInWindow`, `grenadeAuthorRadiusM`).

**Défaut constaté** : `locateThrow` situe un lancer par la naissance de projectile la plus
proche DANS LE TEMPS (fenêtre ±200 ms, `birthNear`), les ex æquo départagés par le tri (donc
par X). Deux joueurs qui lancent dans la même fenêtre — banal — et le lancer reçoit la position
du projectile de l'autre. Second défaut, dans la même branche : le `Grenade` publié par la voie
projectile ne portait **pas de `Slot`** (zéro). Or zéro RESSEMBLE à un slot : côté client,
`grenadeArcs.ts` colore l'arc par le slot du lanceur, et le garde d'ambiguïté « deux candidats
de slots différents s'annulent » ne se déclenchait jamais entre deux zéros.

**Mesure du fork** (film `36e80b83`, avant correctif) : 171 lancers publiés sur 247 sans AUCUN
joueur à moins de 4 m ; distance médiane au joueur le plus proche **7,95 m**, pire cas 24,68 m.
À comparer à la mesure qui fonde la source projectile : 0,77 unité entre une naissance et le
biped de son auteur (6,4 pour un instant permuté). 201/247 lancers sortaient avec `Slot = 0`.
Après correctif : médiane **0,10 m** ; sur le film de référence `000d5950`, 0 lancer à plus de
4 m de son lanceur connu (contre 3), écart max 0,56 m (contre 14,46 m).

**Correctif du fork** :
1. Résoudre l'AUTEUR d'abord (`slotFor` + biped à `TimestampUS`, tolérance `shotPosToleranceUS`).
2. Prendre TOUTES les naissances de la fenêtre (`birthsInWindow`), pas la plus proche.
3. Avec biped connu : garder la naissance la plus proche du biped, refuser si > 4 m
   (`grenadeAuthorRadiusM = 4`, même valeur que `ARC_ORIGIN_RADIUS_M` côté client).
4. Sans biped : une seule candidate = lecture, plusieurs = refus (plus de tirage au sort).
5. Publier `Slot` sur la branche projectile dès que le pont le connaît.
6. Repli biped inchangé quand aucune naissance ne convient.

**Notre arbre (vérifié 2026-09-11)** — `apps/go-api/internal/analysis/replay/grenades.go:144-219` :
`locateThrow` fait encore `birthNear` en premier (temps seul, k = i−1 / i), puis `slotFor` en
repli ; la branche projectile publie `Grenade{X, Y, Src}` **sans `Slot`**. Notre version a
divergé sur un autre axe (elle rend l'index brut de la piste pour `Grenade.Proj`, lien lancer
→ projectile) mais **le défaut d'attribution est identique**. **Présumé toujours valide.**

**Impact chez nous** : tout ce qui attribue un lancer à un joueur — arcs colorés du rejeu,
comptages par joueur (citations, escouade), et le lien `Grenade.Proj` lui-même, qui hérite du
mauvais projectile si la naissance est celle du voisin.

**Comment vérifier avant de planifier** : sur 3 films, pour chaque lancer `Src = projectile`,
distance à `tracks[slotFor(...)].at(t)` ; publier médiane / part > 4 m. Si la médiane est de
l'ordre du mètre ou plus, le point est confirmé.

**Coût estimé** : petit (une fonction, ~80 L, tests unitaires sur fenêtres à 1 et 2 naissances).
Goldens à mettre à jour délibérément (le fork : lancers situés 70 → 67 sur `000d5950`).

---

## Point 2 — Projectiles : repli du quantum Y, un vol traverse toute la carte

**Fichier fork** : `apps/go-api/internal/analysis/replay/projectiles.go`
(`projectileMaxStepM = 10`, `buildProjectiles` rend un compte `truncated`).

**Défaut constaté** (« trouvé en regardant l'écran ») : un pas de 100 ms entre deux points
publiés vaut EXACTEMENT l'étendue Y de la carte (52,88 m pour `sgh_streets`, bornes
Y ∈ [−23,018 ; 29,867]), l'autre axe ne bougeant pas d'un centimètre — ex.
(14,37 ; −23,00) → (14,97 ; 29,84), soit 528 m/s. Jamais sur X, toujours sur Y : le quantum Y
repasse d'un bord à l'autre. **27 à 35 %** des trajectoires de chaque film en portent au moins
un ; le client traçait une droite en travers de la carte.

**Cause** : en amont, dans la déquantification (`filmdec`) — **NON corrigée par le fork**. Le
fork ne corrige que la PUBLICATION : le vol s'arrête au dernier point lisible (pas > 10 m =
coupure, `Rest = false` puisqu'une fin de vol coupée n'est pas certifiée), et compte les
trajectoires tronquées.

**Notre arbre (vérifié 2026-09-11)** — `projectiles.go:48-110` : `buildProjectiles` décime et
trie, **aucun garde-fou de pas**, `Rest` = `AtRest` du dernier point brut. Notre signature a
divergé (rend `map[int]int` brut → publié pour `Grenade.Proj`), mais rien ne borne le pas.
**Présumé toujours valide** — sous réserve que notre déquantification n'ait pas été corrigée
depuis (chantier précision projectiles : `.ai/HANDOFF_PRECISION_PROJECTILES*.md`).

**Ce qui vaut mieux que le pansement** : le fork a caractérisé le défaut assez finement
(saut = étendue Y exacte, X figé) pour qu'on cherche la cause dans `filmdec` : un bit de signe
ou un débordement sur la composante Y de la quantification des projectiles, spécifique à Y
(les bornes Y de Streets sont asymétriques, −23/+29,9). C'est un bug de déquantification à
traquer, PAS seulement une coupure à ajouter.

**Comment vérifier** : sur nos films, compter les pas > 10 m entre points consécutifs de
projectiles et mesurer |Δy| − (ymax − ymin) ; si ≈ 0 sur la majorité, même défaut.

**Coût** : garde-fou = trivial (15 L). Cause racine = investigation `filmdec` (à chiffrer).

---

## Point 3 — En-tête keyframe PAR TYPE d'entité ; l'équipe EST dans le film

**Fichiers fork** : `internal/analysis/filmdec/{prefix_sweep_test, default_state_sweep_test,
default_state_ti0_test, component_probe(.go/_test), archetype_dump_test, respawn_validate_test}`,
`capture.go`, `vitality.go` (`TeamDesignator`, `decodeManagedPlayerTeamDesignator`),
`registry.go` (`compManagedPlayerTeamDesignator`). Commit `2a8b21a51`. **Instruments, pas
fonctionnalités** : tous les bancs sont sautés sans `REPLAY_FILMS`, rien n'est branché en prod.

**Ce que le fork a établi** (Ghidra 12.1.3, binaire importé `-noanalysis` ; second agent
codex/gpt-6-astra qui a exécuté les octets d'origine dans l'émulateur p-code) :

1. **Les grammaires de default-state ti=0/5/6/9 du dépôt sont JUSTES** — vérifié deux fois
   contre le binaire installé (4 680 cas, 0 échec, contrôle ti=35 reproduit). Les 50
   archétypes résolus concordent avec nos ports. L'hypothèse « grammaires fausses » est morte.
2. **L'en-tête d'un record keyframe n'est PAS constant.** Le décodeur suppose 64 bits partout,
   valeur tirée d'un utilitaire écrit pour le BIPED et sans appui dans le binaire. **ti=9
   demande 47 bits** : seul préfixe sur 300 essayés qui rende huit entités joueur d'équipe
   STABLE réparties 4-4 dans les six films ; à 47, le biped passe de 19 736 composants
   décodés à zéro. Donc en-tête PROPRE AU TYPE — et c'est la raison des lectures vides
   (équipe, respawn timers, statborg) : pas une grammaire fausse, un offset faux.
3. **L'équipe est dans le film** (`managed-player-team-designator-component`, ti=9 i0,
   R(4)) : 1 873 lectures sur six films. Manque le lien entité ti=9 → joueur (huit équipes
   sans identité). Ceci contredit notre `document.go` qui affirme « le film ne porte pas
   l'équipe ».
4. **« 0 désync » ne prouve rien** : les premières mesures donnaient 0 désync sur ti=5/6/9/13
   alors qu'ils décodaient ZÉRO composant (masque lu à zéro → sortie immédiate). Les bancs
   du fork publient le compte de composants et refusent un succès sans composant décodé.
   Piège à vérifier dans NOS propres bancs.
5. Confirmés présents : `object-dead-state-component` (1 317 lectures, sur bipeds) et quatre
   signaux de timeline de match. Toujours vides : respawn timers, vies, stats statborg —
   chacun demande sa calibration d'en-tête.

**Notre arbre (vérifié 2026-09-11)** — `filmdec/keyframe_fullstate_loop.go:25-60` : nous
traitons DÉJÀ la largeur d'en-tête comme une **variable à mesurer** (`HeaderBits` : 64
historique vs 108 selon `FUN_142e2bfd0`, `SizeWords`, `DefaultState`, `LevelShift` — plan
R7-e). Le fork apporte une **troisième valeur, 47, spécifique à ti=9**, et surtout la thèse
« largeur PAR TYPE » que notre modèle (une largeur pour tous) ne permet pas d'exprimer.
`managed-player-team-designator-component` est dans notre `traverse.go:508` (consommé R(4),
non capturé). `object-dead-state-component` : `traverse.go:798`, et un gate dédié existe
(`.ai/V7.5/GATE_V13_DEADSTATE_MARCHE_2026-09-05.md`). **Partiellement couvert ; la thèse
« par type » et la valeur 47 sont nouvelles.**

**Comment vérifier** : rejouer notre banc `keyframe_fullstate_loop` avec `HeaderBits = 47`
restreint aux entités ti=9 sur 3 films ; critère du fork = 8 entités, 4-4 stable sur tout le
film. Si ça tient, comparer avec la table des scores (seule vérité connue) — c'est
précisément pourquoi le fork capture le désignateur : calibrer contre une vérité externe.

**Coût** : mesure = une session (banc existant + un paramètre). Exploitation (équipe par
joueur depuis le film) = dépend du lien entité → joueur, non résolu par le fork.

---

## Point 4 — Deux robustesses : verrou fichier Windows EN, film définitivement expiré

### 4a. `IsFileLockError` — libellé Windows anglais

**Fichier fork** : `internal/platform/duckdb/db_recovery.go`. Ajoute
`strings.Contains(strings.ToLower(s), "process cannot access the file because it is being used by another process")`.
Effet chez le fork : un archive DB occupée répond 503 au lieu de 500 sur Windows EN.

**Notre arbre** : `db_recovery.go:66-69` ne connaît que `"File is already open in"` (+ le
commentaire dit que le message FR varie). Sur un poste Windows en locale EN, ou en prod si la
locale système n'est pas FR, le verrou est classé « autre erreur ». **Valide, trivial**
(1 ligne + 1 cas de test dans `is_file_lock_error_test.go`).

### 4b. `IsFilmGoneErr` — manifeste vivant, blobs morts

**Fichier fork** : `internal/sync/haloclient/halo_client_film.go`.

**Constat** : le manifeste et les BLOBS pré-signés expirent sur des calendriers séparés. Un
manifeste qui répond encore alors que ses blobs rendent 404/410 ressort en ERREUR de
`GetFilmChunks`, indiscernable d'un timeout. Un appelant qui retente périodiquement (chez le
fork : passe horaire ; chez nous : cuisson des rejeux, backfill film) retape indéfiniment un
lien mort.

**Correctif** : prédicat exporté `IsFilmGoneErr(err)` = `*HTTPError` 404/410 (manifeste,
statut typé) OU `isNotFoundErr` (blobs, repli textuel — `downloadBlob` formate encore
« downloadBlob HTTP %d » et le typer changerait le pool qui branche sur `*HTTPError`). Une
expiration PARTIELLE compte comme définitive ; si un blob rend 404 et un autre 503,
l'errgroup remonte l'un ou l'autre — un faux « transitoire » se corrige à la passe suivante,
un faux « expiré » serait définitif, donc le biais est du bon côté.

**Notre arbre** : `halo_client_film.go` — `isNotFoundErr` existe (l. 366), les commentaires
disent « (nil, false, nil) si le film est absent » sans distinguer manifeste et blobs ; aucun
prédicat exporté ; grep `IsFilmGone|expired` = 0 hors tests. **Valide si** un de nos chemins
retente des films (à vérifier : `internal/sync` film/cuisson, `cmd/levelup backfill`). Sinon,
c'est de la doc : le commentaire de `GetFilmChunks` mérite au moins la précision
manifeste ≠ blobs.

**Coût** : 20 L + test (`halo_client_isfilmgone_test.go` du fork, 48 L, réutilisable tel quel).

---

## Point 5 (bonus) — Props Forge : UN fichier dessiné sur TOUTES les cartes

**Fichier fork** : `internal/domain/title/registry.go` (`MapGeometryDir(titleSlug, module)`),
`replay/geometry.go` (`LoadGeometry(propsDir, catalogDir)`), commit `621f69040`.

**Constat** : `MapGeometryDir` ne prend que le titre et rend UN répertoire ; le CSV qui s'y
trouve est dessiné sur CHAQUE match quelle que soit la carte. Mesuré sur six films de trois
cartes : 382 props identiques partout, y compris sur des cartes sans fichier de structure. Le
CSV ne porte aucune colonne de carte. Tant que le sol était « en boîtes », ces props étaient les
seuls repères lisibles ; avec un fond correct, un décor d'une AUTRE carte est une donnée fausse
sur un outil où l'on mesure des positions. Le fork met le fichier sous
`map_geometry/UNATTRIBUTED/` + README (la donnée est peut-être juste, c'est l'attribution qui
manque), et sépare props par carte / catalogue de types par titre.

**Notre arbre (vérifié 2026-09-11)** : `registry.go:899` `MapGeometryDir(titleSlug)` → un seul
répertoire ; `data/titles/halo_infinite/reference/map_geometry/` contient `map_objects.csv`
(453 props, sans colonne carte) + `forge_object_types.csv` ; `replaybuild.go:116-123` charge
ce répertoire une fois pour tous les artefacts. **Défaut identique, présumé valide** — sauf si
le client n'affiche plus la couche props (à vérifier côté `apps/web` `match-replay`).

**Coût** : petit (signature + déplacement du CSV + README + 1 test PathResolver). Identifier la
carte du CSV = une session (comparer aux `map_quant_bounds` : 453 props dans quelle emprise ?).

---

## Point 6 (issu de la PR #28) — Littéraux FR en dur hors i18n, côté web

La PR #28 n'apporte rien comme conversion, mais son commit `677574859` a listé des chaînes
visibles écrites EN DUR en français — c'est-à-dire des endroits où la locale n'est PAS
respectée. Vérifié présent chez nous le 2026-09-11 :

| Fichier | Chaîne en dur |
|---|---|
| `apps/web/src/components/shell/ThemeToggle.tsx:45` | « Passer au thème clair / sombre » (aria) |
| `apps/web/src/components/ui/carousel.tsx:90,110` | `aria-label="Précédent"` / `"Suivant"` |
| `apps/web/src/features/palmares/BattlePassRewardCarousel.tsx:58,128-130` | « Voir le détail de … », `'gratuit'`, « Paliers précédents/suivants » |
| `apps/web/src/features/setup/StepPlayer.tsx:75,152,154` | « Erreur lors de la création du profil. », « Création… », « Confirmer et créer mon profil », « Ajouter », placeholder « MonGamertag » |
| `apps/web/src/features/feedback-drawer/FeedbackDrawer.tsx:129` + `buildIssueUrl.ts` | « _(sans titre)_ » + corps d'issue GitHub en FR (~20 lignes accentuées) |
| `apps/web/src/features/lab/ChartsShowcasePage.tsx` | ~41 lignes accentuées (page de labo, faible priorité) |
| `apps/web/src/features/auth/XboxLoginPage.tsx:393` | « Erreur de connexion. » |
| Repli `?? 'FR'` quand `fieldMappings` n'est pas chargé | `SynthesisPage.tsx:385-391` (« Tirs à la tête », « Tirs effectués », « Tirs au but », « Dégâts infligés/reçus »), `TimeseriesPage.distributions.tsx:85-98` (« Score personnel », « Score de performance », « Précision », « MMR équipe/adverse »), `TimeseriesPage.summary.tsx` (« Égalité », « Durée de vie moyenne »), `TimeseriesPage.tsx` (« Égalité », « Abandon »), `TimeseriesPage.progression.tsx:139` |

Les « défauts Go » que la PR cite (`pickAssetNameByPreferredLang` repli alphabétique,
`loadCitationMappingMeta` teste `== "en"`) ne sont **pas** des bugs chez nous :
`PreferredLangsForLocale` cascade déjà fr → en (et inversement), et le middleware
(`middleware/title.go:50`) normalise la locale en `"fr"`/`"en"` avant `ctxkeys`. Ils
n'étaient des défauts que dans un monde sans FR.

**Coût** : mécanique (passage par `Record<Locale, T>` dans les `i18n.ts` de chaque feature,
règle n°1). Les replis `?? 'FR'` sont transitoires (le temps du chargement) mais visibles
sous UI EN ; le bon repli est le libellé i18n de la feature, pas un littéral.

---

## Ce qui n'est PAS à reprendre

- **Témoin d'arme** (`lives_witness.go`, `37591d6de`) : départage des vies ambiguës
  (deux morts au même tick) par accord d'arme entre events de tir (par joueur) et keyframes
  (par slot). Mesure fork : 9 ambiguïtés levées, +8 vies sur six films ; contrôle 593 accords /
  4 contradictions (99,3 %). Chez nous, le pont par morts est passé en **vérification** derrière
  le lien direct corps → joueur du registre d'identité (`b6b0ee4c5`, `2fb53db4e`) : le problème
  résolu par le témoin est probablement caduc. À ne rouvrir que si notre taux de vies anonymes
  (`.ai/AUDIT_LECTEURS_VIES_ANONYMES_2026-09-06.md`) reste significatif après le registre.
  **Idée à garder** : publier des compteurs de CONTRÔLE (accord/contradiction/silence sur les
  cas déjà tranchés) à côté de tout témoin, pour qu'il soit falsifiable dans l'artefact.
- **Horloge du match** (`matchClockZeroMs`, décalage issu de `bestDeathOffset`) : nous
  publions déjà `OriginMs` et le coup d'envoi (`document.go:64-98`, commit `0d5e9ffd5`).
- **Une track = une vie** (`decimateTracks` découpé au `lifeGapUS`) : couvert par `48cf4905d`
  et suivants (schéma 36, 41).
- **Fond de carte Streets par rendu ekur/Blender** (`tools/map-render/`, pipeline Python
  Blender) : hors règle « pas de Python », et notre chantier fonds de carte a sa propre voie
  (`map_backgrounds/`, `map_fond_reglages.json`). La mesure « 175 484 positions, 100 % sur
  géométrie dessinée, zéro calibration » reste un bon critère d'acceptation pour NOTRE fond.
- Tout `apps/study/`, `cmd/study-archiver`, `cmd/study-server`, heat maps (#19–#26).
- PR #28 dans son ensemble.

## Ordre de valeur suggéré (à confirmer après le chantier en cours)

1. Point 1 (grenades) — impact direct sur des stats publiées, correctif court.
2. Point 5 (props par carte) — donnée fausse à l'écran, correctif court.
3. Point 2 (projectiles) — garde-fou immédiat, cause racine à chiffrer.
4. Point 4a/4b — trivial, à glisser dans un lot robustesse.
5. Point 6 — lot i18n mécanique.
6. Point 3 — investigation ; vaut une session de mesure avant tout engagement.
