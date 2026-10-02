# PLAN — Représentation intermédiaire du film, étape 1 (2026-10-02)

> **Statut : PRÊT À EXÉCUTER — confié à une autre conversation** par décision de l'utilisateur du
> 2026-10-02 (« si le plan pour la représentation intermédiaire est prêt, je préfère te laisser
> terminer ton chantier et mettre une autre conversation dessus »). Contrat : skill `plan-execution`
> (ce plan fait foi en cas de divergence). Chaque lot démarre quand ses prérequis sont tenus ; aucune
> décision produit n'est laissée ouverte.
>
> **Pour qui** : l'agent qui exécute ce plan dans une autre conversation, et la session de la campagne
> de grammaire, qui travaille en parallèle sur les mêmes paquets.
>
> **Sources (à lire avant le premier lot)** :
> - `.ai/ANALYSE_MISE_EN_OEUVRE_REPRESENTATION_INTERMEDIAIRE_2026-10-01.md` — §2 (huit corrections de la
>   spec, C1 à C8), §3 (forme, API, lots), §6 (ordre) ;
> - `.ai/V7.5/film_re/RAPPORT_IR_CARTOGRAPHIE_GO_2026-10-01.md` — §1 (traversées, avec fichier:ligne),
>   §2 (germes), §3 (types, API, ratchets, migration détaillée) ;
> - `.ai/SPEC_REPRESENTATION_INTERMEDIAIRE_FILM_2026-09-25.md` — cadrage (lire en tenant compte de C1-C8) ;
> - `docs/adr/0034-film-decoder-profile-and-layers.md` et son annexe ;
> - `CLAUDE.md` (règles, dont la règle 17 sur les commentaires et les ratchets de J12).

---

## 0. Objet, critères de succès, périmètre

**Objet.** Faire de la marche de production du décodeur LA lecture unique du film : un marcheur en
deux phases (images-clés, puis trames delta) qui range ce qu'il lit dans une structure typée (paquets,
vues, records, composants, avec leurs étendues en bits et l'état de fermeture), consommée dès cette
étape par ses deux lecteurs actuels (états de mouvement et tir continu ; carte de fermeture), avec les
tests de la spec (T1 fermeture, T3 provenance, T5 robustesse, T6 déterminisme).

**Critères de succès (mesurables).**
1. **Zéro différence de sortie** : `replay-equiv` identique sur tout le corpus, `frame_closure.golden`
   et `keyframe_closure.golden` identiques, sortie `cmd/killsource json` identique à l'octet sur les
   témoins, `go test ./internal/games/halo_infinite/film/...` vert.
2. La recopie du pilotage de la marche dans `frame_closure.go` disparaît (un seul pilotage).
3. Les tests T1, T3, T5, T6 existent et sont verts ; la CI est verte au niveau job.
4. Durée et pic mémoire d'une cuisson : pas de régression de plus de 10 % sur trois témoins et un BTB
   (mesuré avant/après, `replay/observe.go` et la sentinelle `filmproc`).

**Hors périmètre** (explicite) :
- l'étape 2 (migration des canaux vers la structure, couche de récupération mutualisée) et l'étape 3 :
  elles feront l'objet d'un plan suivant, écrit à la fin de cette étape, à partir de l'analyse §3.3 et
  du rapport §3.8 ;
- toute correction de grammaire (lecteur de composant, localisateur, naissances, fermeture) : c'est la
  campagne de grammaire (§1.3) ;
- toute cuisson ou recuisson du parc, tout backfill.

## 1. Base, branche, coordination

### 1.1 Branche et worktree
- Branche **`feat/representation-intermediaire`**, créée depuis `origin/feat/v75` à jour.
- Worktree dédié **à côté du checkout principal** (`LevelUp-wt-ri`) ; jonction `apps/web/node_modules`
  vers le checkout principal pour le hook de push (à retirer par `(Get-Item <jonction>).Delete()`
  AVANT tout `git worktree remove`, jamais `--force` sur un worktree qui porte une jonction).
- Un seul exécutant à la fois dans ce worktree. Films lus EN PLACE depuis le cache du checkout
  principal, en lecture seule.

### 1.2 Fusions
- Au début de chaque lot : `git merge origin/feat/v75` dans la branche (en cas de conflit,
  `feat/v75` a raison).
- La branche se fusionne dans `feat/v75` à la fin de l'étape 1, après accord de l'utilisateur, CI
  verte et `make gate-push`.
- Après chaque fusion d'une vague de la campagne dans `feat/v75`, les références d'équivalence
  changent : la branche refusionne `feat/v75`, re-fige ses références (`go run ./cmd/replay-equiv
  -update` sur le corpus) et rejoue sa preuve « zéro différence » contre elles.

### 1.3 Coordination avec la campagne de grammaire (session parallèle)
La campagne (`feat/campagne-grammaire`, plan `.ai/PLAN_CAMPAGNE_GRAMMAIRE_2026-10-01.md`, sur le PC
de la session « campagne de grammaire ») change des sorties du décodeur par lots. Propriété des
fichiers pendant cette étape :

| Fichiers | Propriétaire | Conséquence pour ce plan |
|---|---|---|
| `grammar/frame_closure*.go`, `frame_vue_controle.go`, `frame_harvest.go`, `debut_de_liste.go`, `tir_continu.go`, `research/cmd_fermeture/` | campagne, lot « définition de la fermeture » (L0, en cours le 2026-10-02) | le lot 1.2 de ce plan NE DÉMARRE PAS avant que L0 soit fusionné dans `feat/v75` |
| lecteurs de composants, `dispatch_*.go`, `components_*.go`, `profile/` | campagne, vague 1 | ce plan n'y touche pas (les lecteurs sont les briques du marcheur, inchangées) |
| `keyframe_world*.go`, `keyframe_closure.go` | campagne (marche d'image-clé de toutes les générations, L9) | le lot 1.3 attend que ce lot de la campagne soit fusionné, ou se coordonne avec elle |
| `object_deaths_march.go`, `facts/killsource/walk.go` | campagne, vague 2 (unification du localisateur, signature du slot 123) | hors de l'étape 1 |
| `grammar/lecture/` (neuf), `grammar/marche_trames*.go` (neufs), `movement_states.go`, `frame_infer.go` | ce plan | la campagne n'y touche pas pendant l'étape 1, sauf le lot L0 déjà en cours |

Chaque session prévient l'autre (Remote Control, message direct) quand elle fusionne dans
`feat/v75`. Une découverte qui touche un fichier de l'autre se signale, ne se corrige pas.

## 2. Décisions (fermes)

**Utilisateur.**
- 2026-10-01 : l'étape 1 et les lots sans différence de la représentation intermédiaire démarrent
  après la fusion de J12 (fait le 2026-10-02, `feat/v75` post-J12).
- 2026-10-02 : la représentation intermédiaire est menée dans une autre conversation, en parallèle de
  la campagne de grammaire.

**Techniques** (issues de l'analyse §2 et §7, retenues pour ce plan ; une objection de l'utilisateur
les rouvre) :
- **DT-1 (C6)** : les types de la structure vivent dans un paquet feuille
  `film/internal/grammar/lecture` (aucune logique, n'importe que `film/types`), pas dans `film/types` :
  sa forme ne fait monter que `grammar.Rev`. Ratchet : ni `replay` ni `decfilm` ne l'importent.
- **DT-2 (C1)** : UNE table d'entités (slot → eid complet, archétype, vue, provenance de la liaison),
  exposée en lecture seule ; le commentaire de `world.go:32` qui parle de trois tables est corrigé
  dans le même lot (règle 17).
- **DT-3 (C2)** : deux phases — images-clés d'abord (`FilmContext.ImagesCles`), puis trames delta
  (`FilmContext.Trames`) — en `iter.Seq2` ; les préliminaires bornés (table anticipée, bande bipède,
  générations vivantes) restent avant la phase delta.
- **DT-4 (C3)** : un paquet porte un état de fermeture : fermé, fermeture refusée (position fautive
  inconnue), ou queue opaque à cause typée et position connue ; jamais confondus.
- **DT-5 (C4)** : la récupération qui vit déjà dans la marche (localisation de la vue B par signature,
  NEW de tête, datums à position libre, liaison par anticipation, élection d'ancre d'image-clé) est
  MARQUÉE comme telle dans la structure (provenance « récupéré »), sans changer son comportement.
- **DT-6 (C8)** : le marcheur EST la marche de production refactorée et consommée dès le lot 1.2 ; il
  n'y a pas de second marcheur parallèle non consommé.
- **DT-7** : l'interprétation reste faite pendant la marche par les crochets de l'`Observation`
  existante (différence nulle) ; aucune persistance de la structure (le cache des faits suffit).
- **DT-8** : noms français, comme le reste du paquet (`ImagesCles`, `Trames`, `Distribuer`, types
  `lecture.Paquet`, `lecture.Record`, `lecture.Composant`, `lecture.Etendue`).

## 3. Lots

Tous les lots : zéro différence de sortie. Révision : la forme du paquet `grammar` change, donc
l'empreinte de `grammar_rev.golden` et `grammar_perimetre.golden` se régénère à RÉVISION CONSTANTE par
la commande du dépôt (`LEVELUP_UPDATE_GRAMMAR_REV=1 go test ./internal/games/halo_infinite/film/internal/grammar/ -run TestGrammarRevSuitLaGrammaire -update-grammar-rev`)
après la preuve de différence nulle ; `grammar.Rev` ne monte pas.

### Lot 0 — ADR (taille S, aucun prérequis)
- [x] 0.1 Écrire `docs/adr/0037-film-intermediate-representation.md` (EN, ADR = anglais seulement,
      règle 15) à partir du projet d'ADR de la spec §12, CORRIGÉ par C1 à C8 et par DT-1 à DT-8 ;
      il amende ADR 0034 D-1, D-2, D-6, D-7, D-10 ; il cite les mesures d'appui (analyse §1) et le
      port Rust comme confirmation indépendante de la direction (5,5 % contre 66,9 % de trames
      fermées : la récupération séparée est nécessaire).
      *Fait* : douze décisions IR-1 à IR-12 (C1 à C8 et DT-1 à DT-8 répartis, plus IR-12 : le seuil
      de 95 % est un indicateur, décision D1 de l'utilisateur du 2026-10-02 consignée au plan de la
      campagne) et une section « Amendments to ADR 0034 » (D-1, D-2, D-6, D-7, D-10).
- [x] 0.2 Ajouter l'ADR à la liste des ADRs de `CLAUDE.md` (une ligne) et au renvoi de l'ADR 0034.
      *Fait* : entrée `0037` en fin de liste de `CLAUDE.md` ; ligne « Amended on 2026-10-02 by
      ADR 0037 » dans le statut de l'ADR 0034.
- Gate : relecture ; `go test ./internal/archlint/` (ratchet des chemins `.ai` cités).
  *Tenu le 2026-10-02* : relecture faite (chemins et chiffres vérifiés sur pièces) ;
  `go test ./internal/archlint/ -count=1` ok (64,8 s). Aucun test Go ne lit `docs/adr/` ni
  `CLAUDE.md` (grep).

### Lot 1.1 — Paquet de types `grammar/lecture` (taille S, aucun prérequis)
- [ ] 1.1.1 Paquet feuille `apps/go-api/internal/games/halo_infinite/film/internal/grammar/lecture`,
      types seuls, d'après le rapport §3.2 : `Etendue`, `Etat` (interprété / délimité /
      infranchissable), `ProvenanceLargeur`, `Composant`, `Record`, `Genre`, `Liaison`, `Preuve`
      (fermé / non prouvé / récupéré), `QueueOpaque` et sa cause, `Fermeture`, `Paquet`, `VueA`,
      `VueC`, `Entites` (interface en lecture seule). Tailles mémoire visées : ~40 o par record,
      ~12 o par composant, arène par paquet (P9).
- [ ] 1.1.2 Ratchet `archlint/film_layers_deps_test.go` : `grammar/lecture` appartient à la couche
      `grammar` ; règle d'interdiction pour `replay` et `decfilm`.
- [ ] 1.1.3 Tests de taille des types (`unsafe.Sizeof`) gelés.
- Gate : G-unit du paquet, G-arch, G-vet ; empreinte régénérée à révision constante.

### Lot 1.2 — Phase delta : la marche de production devient `FilmContext.Trames` (taille M)
**Prérequis** : lot L0 de la campagne (définition de la fermeture) fusionné dans `feat/v75`, puis
`git merge origin/feat/v75` dans la branche et références d'équivalence re-figées.
- [ ] 1.2.1 Le pilotage de `movementStateScanner.marcher` (`movement_states.go`) devient
      `FilmContext.Trames(interets) iter.Seq2[*lecture.Paquet, error]` dans des fichiers neufs
      `grammar/marche_trames*.go`.
- [ ] 1.2.2 `ScanMarcheDesTrames` (états de mouvement, tir continu) et `FrameClosure` en deviennent
      les consommateurs ; la recopie du pilotage dans `frame_closure.go` disparaît.
- [ ] 1.2.3 `HeaderBit` posé dans `decodeInferLoop` (étendue de chaque record) ; étendues des vues ;
      sortie de vue B typée (terminateur / rejet hors datum / rejet de vue, compteurs de
      `observateur.go`) ; état de fermeture du paquet selon DT-4.
- [ ] 1.2.4 Table d'entités exposée en lecture seule (DT-2) ; liaisons marquées par provenance ;
      récupération marquée (DT-5).
- Gate : G-unit, G-arch, G-vet, G-film ; G-equiv : zéro divergence ; `frame_closure.golden`
  identique ; killsource identique à l'octet sur les témoins ; durée et pic mémoire mesurés (critère 4).

### Lot 1.3 — Phase images-clés : `FilmContext.ImagesCles` (taille M)
**Prérequis** : lot 1.2 clos ; lot de la campagne sur la marche d'image-clé (L9) fusionné ou
coordonné (§1.3).
- [ ] 1.3.1 `FilmContext.ImagesCles` sur la mémoire existante de `MarcheDImageCle`
      (`keyframe_world_marche.go`), étendues de l'état complet, mémoire PARTAGÉE entre les contextes
      de la cuisson et de killsource si cela ne change aucune sortie (sinon, découverte consignée).
- [ ] 1.3.2 `KeyframeClosure` en devient le consommateur.
- Gate : `keyframe_closure.golden` identique ; G-equiv zéro divergence ; G-film ; killsource identique.

### Lot 1.4 — Tests de la spec (taille S)
- [ ] 1.4.1 T1 fermeture : pour chaque paquet de la structure, bits consommés et état de fermeture
      cohérents ; ratchet « aucune baisse » sur les bobines du dépôt.
- [ ] 1.4.2 T3 provenance : toute lecture d'état de mouvement et de tir continu cite une étendue qui
      existe dans la structure.
- [ ] 1.4.3 T5 robustesse : le fuzz des lecteurs de records (`FuzzFilmRecordReaders`) étendu au
      marcheur ; aucune panique, allocations bornées par les octets restants.
- [ ] 1.4.4 T6 déterminisme : deux marches du même film donnent la même empreinte ; marches
      parallèles identiques sous `-race` (job CI `film-race`).
- Gate : G-unit, G-arch, `go test -race -run TestDeuxFilmsEnParallele ./internal/games/halo_infinite/film/internal/grammar/`.

### Clôture de l'étape 1
- [ ] Mesure de performance avant/après (critère 4) publiée.
- [ ] G-CI vert au niveau job ; G-push ; accord de l'utilisateur ; fusion dans `feat/v75`.
- [ ] Plan de l'étape 2 écrit (migration des canaux et récupération mutualisée, analyse §3.3 lots 2.1
      à 2.7, 3.1, 3.2), en tenant compte de l'état de la campagne à ce moment.

## 4. Contrat d'exécution

- Skill `plan-execution` : ordre strict, une étape commencée est terminée, aucun report d'une action
  exécutable maintenant, statut de chaque item (`[x]` / `[~]` / `[!]` justifié), clôture d'un lot =
  gate passé + items statués + plan à jour + entrée `.ai/thought_log.md` + point à l'utilisateur.
- Zéro correctif hors périmètre : toute découverte va au §6.
- Commits : au moins un par lot, message préfixé par le lot (`ri(1.2): ...`) ; demander avant tout
  push ; jamais `git stash` ; jamais `git add -A` (chemins explicites).
- Commandes `go` une à la fois sur la machine ; jamais de Python ; aucun emoji ; commentaires = contrat,
  pas d'histoire (règle 17).
- Points à l'utilisateur en LANGAGE CLAIR (ce qui marche maintenant, ce qui reste), sans codes de lots.

**Gates standard** (depuis `apps/go-api`) :

| Nom | Commande |
|---|---|
| G-unit | `go test <paquets du lot> -count=1` |
| G-arch | `go test ./internal/archlint/ -count=1` |
| G-vet | `go vet ./...` (CGO) et `go vet -tags=research ./...` |
| G-film | `go test ./internal/games/halo_infinite/film/... ./internal/replaybuild/... ./internal/sync/killcollector/... -count=1` |
| G-equiv | `go run ./cmd/replay-equiv -repo-root <worktree>` → zéro divergence (références re-figées sur la tête de `feat/v75` fusionnée) |
| killsource | `go run ./cmd/killsource json <film> -carte <carte>` avant/après sur les témoins de `config/replay_corpus.toml` → identique à l'octet |
| G-CI | `gh run list --branch feat/representation-intermediaire --limit 3` → jobs verts |
| G-push | `make gate-push` avant toute fusion dans `feat/v75` |

## 5. Protocole de reprise

Relire le skill `plan-execution`, puis ce fichier (§2 décisions, §7 journal) et le §1.3 ; reprendre à
la première case non statuée ; `git -C <worktree> log --oneline -10` ; vérifier sur `origin/feat/v75`
si la campagne a fusionné un lot depuis la dernière reprise (et refusionner).

## 6. Découvertes (consignées, non traitées)

(vide)

## 7. Journal

- 2026-10-02 : plan écrit par la session de la campagne de grammaire, à la demande de l'utilisateur,
  à partir de l'analyse du 2026-10-01 et des huit corrections de la spec ; confié à une autre
  conversation.
- 2026-10-02 : exécution confiée par l'utilisateur à la session `levelup-57` (« commence par l'ADR et
  le paquet de types ; ne commence pas la phase des trames tant que L0 n'est pas fusionné ; tu peux
  committer, demande avant de pousser ou de fusionner »). Worktree `LevelUp-wt-ri`, branche
  `feat/representation-intermediaire` créée depuis `origin/feat/v75` = `93cea7cdc` (amont désactivé
  pour qu'aucun push n'aille sur `feat/v75`), jonction `apps/web/node_modules` posée. Cache de
  compilation Go DÉDIÉ (`GOCACHE=%LOCALAPPDATA%\go-build-ri`) : la campagne compile en parallèle sur
  la même machine, et deux commandes `go` sur un même cache le corrompent. Campagne prévenue
  (session `levelup-83`) : accusé de réception, L0 ni commité ni fusionné à cette heure.
- 2026-10-02 : lot 0 clos (ADR 0037, gate tenu).
