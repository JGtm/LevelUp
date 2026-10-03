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
- 2026-10-03 : la mémoire de la marche des images-clés n'est PAS partagée entre la cuisson et
  killsource à cette étape (lot 1.3.1) ; elle se reprend à l'étape 2, où killsource devient un
  canal de la même marche (« ok avec toi »).

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
- [x] 1.1.1 Paquet feuille `apps/go-api/internal/games/halo_infinite/film/internal/grammar/lecture`,
      types seuls, d'après le rapport §3.2 : `Etendue`, `Etat` (interprété / délimité /
      infranchissable), `ProvenanceLargeur`, `Composant`, `Record`, `Genre`, `Liaison`, `Preuve`
      (fermé / non prouvé / récupéré), `QueueOpaque` et sa cause, `Fermeture`, `Paquet`, `VueA`,
      `VueC`, `Entites` (interface en lecture seule). Tailles mémoire visées : ~40 o par record,
      ~12 o par composant, arène par paquet (P9).
      *Fait* : cinq fichiers (`doc.go`, `etendue.go`, `record.go`, `paquet.go`, `entites.go`), tous
      les types demandés, plus ce que 1.2.3 et DT-4/DT-5 exigent de la forme : `VueB` et
      `SortieVueB` (sortie de vue B typée), `Verdict` (fermé / refusé / queue opaque, jamais
      confondus) et `CauseDeQueue`, `DebutDeVueB` (début lu en tête ou LOCALISÉ : signature,
      chaîne, fermeture — la récupération dans la marche marquée), `EtatDeVue`, `EntreeVueC`
      (tours de la vue C avec étendue, pour T3), `Entite`. Écarts au rapport, motivés dans les
      commentaires : pas de genre « Fin » (le terminateur est une sortie de vue, pas un record),
      genre `EtatComplet` pour l'image-clé, `Liaison` porte aussi `ImageCleElue` (DT-5), le champ
      `Val` du composant retiré (l'interprétation reste aux crochets, DT-7), la queue opaque est
      une valeur de `Fermeture` et non un pointeur (aucune allocation). Rangs de vue dans la
      numérotation du FILM (vue B = 1). Tailles : `Record` 40 o, `Composant` 12 o, `EntreeVueC`
      12 o, `Etendue` 16 o.
- [x] 1.1.2 Ratchet `archlint/film_layers_deps_test.go` : `grammar/lecture` appartient à la couche
      `grammar` ; règle d'interdiction pour `replay` et `decfilm`.
      *Fait* : la ligne de couche dans `film_layers_deps_test.go` ; la règle dans un fichier voisin,
      `archlint/film_lecture_test.go` (le premier est à 490 lignes, seuil 500), qui tient aussi
      les deux autres propriétés de DT-1 — la feuille (aucun import du dépôt sauf `film/types`) et
      « aucune logique » (ni fonction, ni méthode, ni variable). Mutations jouées, toutes rouges
      puis retirées : import de `source` dans `lecture` (L1), méthode sur `Paquet` (L2), import de
      `lecture` dans `replay/document.go` (L3), ligne de couche retirée
      (`TestCouchesDuDecodeurSontPeupleesEtALeurPlace`).
- [x] 1.1.3 Tests de taille des types (`unsafe.Sizeof`) gelés.
      *Fait* : `lecture/tailles_test.go` — `TestTaillesDesTypesSontGelees` (mutation : un `uint8`
      ajouté à `Record` le fait passer à 48 o, rouge) et `TestLesValeursZeroSontLesSentinelles`.
- Gate : G-unit du paquet, G-arch, G-vet ; empreinte régénérée à révision constante.
  *Tenu le 2026-10-02* : `go test ./internal/games/halo_infinite/film/internal/grammar/lecture/`
  ok ; `go test ./internal/archlint/ -count=1` ok ; `go vet ./...` et `go vet -tags=research ./...`
  sortie 0 ; `grammar_perimetre.golden` (+1 ligne, le paquet `lecture`, aucun autre périmètre ne
  bouge) et `grammar_rev.golden` régénérés à révision CONSTANTE (`grammar-2026-09-27.3`, empreinte
  `558380db…` -> `28cc08a8…` ; aucun consommateur n'importe encore `lecture`, aucune sortie ne
  peut changer) ; `go test ./internal/games/halo_infinite/film/revision/` et les gates de révision
  de `grammar`, `killsource` et `objectives` verts.

### Lot 1.2 — Phase delta : la marche de production devient `FilmContext.Trames` (taille M)
**Prérequis** : lot L0 de la campagne (définition de la fermeture) fusionné dans `feat/v75`, puis
`git merge origin/feat/v75` dans la branche et références d'équivalence re-figées.
- [x] 1.2.1 Le pilotage de `movementStateScanner.marcher` (`movement_states.go`) devient
      `FilmContext.Trames(interets) iter.Seq2[*lecture.Paquet, error]` dans des fichiers neufs
      `grammar/marche_trames*.go`.
      *Fait* : `marche_trames.go` (le marcheur : table anticipée posée une fois, puis chunk par
      chunk la liaison des images-clés au monde, puis paquet par paquet la localisation des listes
      et la marche par rangs ; `FilmContext.Trames(obs)` en est l'itérateur), `marche_trames_rangs.go`
      (la marche d'UNE trame par rangs, `lireTrameParRangs`, seule implantation — `decodeFrameParRangs`
      n'en est plus qu'une porte), `marche_trames_ranger.go` (ce que la marche range dans la
      structure, sans relire un bit), `marche_trames_entites.go` (la table d'entités). Écart au
      libellé : les « intérêts » sont l'`Observation` existante, dont les crochets interprètent
      pendant la marche (DT-7). Le paquet rendu est l'arène de la marche, valide pendant le tour
      qui le rend ; son en-tête est posé AVANT la marche du paquet, parce que les crochets qui
      publient pendant la marche le lisent.
- [x] 1.2.2 `ScanMarcheDesTrames` (états de mouvement, tir continu) et `FrameClosure` en deviennent
      les consommateurs ; la recopie du pilotage dans `frame_closure.go` disparaît.
      *Fait* : les états de mouvement (`movementStateScanner.trame`), le tir continu (le verdict de
      la vue C lu dans la trame rendue), `FrameClosure` et `FrameClosureDetaillee` consomment
      `marcheurDesTrames.parcourir` ; les deux recopies du pilotage (`frame_closure.go`,
      `frame_closure_detail.go`) ont disparu. Garde-rail `marche_trames_unique_test.go` : dans le
      code de production du paquet, `lierLeChunkAuMonde` et `localiserLaListe` ne sont appelés que
      depuis `marche_trames.go` (mutation : un appel ailleurs, rouge). Les formes booléennes
      `debutDeLaListe` / `debutParFermeture`, sans appelant de production, vivent dans le fichier de
      test étiqueté `marche_trames_sondes_research_test.go` pour les sondes de la campagne (avec
      `marcherParRangs` et `largeurTagDeGeneration`) ; `debut_par_fermeture_test.go` teste le rang
      rendu par `debutParFermetureRangee`.
- [x] 1.2.3 `HeaderBit` posé dans `decodeInferLoop` (étendue de chaque record) ; étendues des vues ;
      sortie de vue B typée (terminateur / rejet hors datum / rejet de vue, compteurs de
      `observateur.go`) ; état de fermeture du paquet selon DT-4.
      *Fait* : `HeaderBit` et `FinBit` de chaque record (`finirLeRecord`) ; étendues des vues A, B,
      C et de chaque tour de vue C (`FluxVueC.Tours`) ; chaque sortie de `decodeInferLoop` est
      typée (terminateur, rejet hors datum, rejet de vue — avec l'eid rejeté —, record
      infranchissable, fin de payload, plafond), les compteurs de `observateur.go` inchangés ;
      verdict fermé / refusé / queue opaque à cause typée et position connue (record et composant
      désignés pour un composant non porté), jamais confondus ; preuve de chaque record. Tests :
      `marche_trames_test.go` (paquets synthetiques dont chaque bit est connu ; mutation « fin d'un
      record prise à la fin de son en-tête », rouge) et `marche_trames_bobines_test.go` (sur les
      bobines du dépôt : records et composants contigus et emboîtés, tours de vue C contigus,
      verdict accordé à la vue C et aux preuves ; la marche rend les comptes de la carte de
      fermeture et des états de mouvement).
- [x] 1.2.4 Table d'entités exposée en lecture seule (DT-2) ; liaisons marquées par provenance ;
      récupération marquée (DT-5).
      *Fait* : `lecture.Entites` sur le monde (`entitesDuMonde` : eid, archétype, rang de vue du
      film, provenance, génération connue) ; le commentaire de `world.go` qui parlait de trois
      tables corrigé (une table, la vue en attribut) ; chaque porte de liaison du monde pose sa
      provenance (NEW lu, image-clé, datum, anticipation, inférence, joker) et chaque record dit la
      liaison sous laquelle il a été lu ; récupération marquée dans la marche : le début d'une
      liste d'événements (signature, chaîne, fermeture au premier rang, repli du second rang
      `DebutParFermetureAuBit`), les liaisons de datum / d'anticipation / d'inférence, et la
      provenance des largeurs (écrivain, exception datée, calibrée, bouchon). Restent hors de ce
      lot, par le plan : l'élection d'ancre d'image-clé (`LiaisonImageCleElue`, phase images-clés,
      lot 1.3). Aucun lecteur de production ne porte de largeur présumée (`LargeurPresumee`
      n'a pas de source ; confirmé par la campagne le 2026-10-02 : aucun lot de sa vague 1 n'en
      pose).
- Gate : G-unit, G-arch, G-vet, G-film ; G-equiv : zéro divergence ; `frame_closure.golden`
  identique ; killsource identique à l'octet sur les témoins ; durée et pic mémoire mesurés (critère 4).
  *Tenu le 2026-10-02 (nuit)* : G-unit (`grammar/...`) vert ; G-arch vert ; G-vet sortie 0 avec et
  sans `research` ; G-film vert (23 paquets) après régénération de l'empreinte à révision CONSTANTE
  (`grammar-2026-10-02`, `a38f9ba8…` -> `7f46317a…`, périmètre inchangé) ; `golangci-lint` (ratchet
  du dépôt) 0 problème sur les paquets touchés. G-equiv : `replay-equiv` 20/20 identiques aux
  références re-figées sur la base du lot (`c787c307f`), chaque film DÉCODÉ — les faits de la passe
  de référence mis de côté avant la passe, parce que leur fraîcheur ne regarde que les révisions
  déclarées et qu'une passe à révision constante les aurait relus ; les 20 fichiers de faits
  identiques à l'octet à ceux de la référence ; `frame_closure.golden` inchangé ; killsource
  `json` 19/19 témoins identiques à l'octet, journaux identiques hors horodatage. Critère 4 mesuré
  (binaires alternés film par film, ordre inversé au second tour, quatre films dont le BTB
  `084a804d`) : durée moyenne par film de −7,9 % à +4,7 % (écarts par paire de −16,5 % à
  +10,3 %), pic mémoire moyen par film de −6,7 % à +3,3 % ; mais un même binaire varie jusqu'à 15 %
  d'un tour à l'autre (durée et pic) sous la charge de la campagne (trois agents qui décodent) :
  aucune régression visible, NON CONCLUSIF SOUS CHARGE en deçà de 10 % ; mesure à rejouer machine
  calme (item de clôture de l'étape), quand la campagne aura assemblé sa vague 1 (elle préviendra).

### Lot 1.3 — Phase images-clés : `FilmContext.ImagesCles` (taille M)
**Prérequis** : lot 1.2 clos ; lot de la campagne sur la marche d'image-clé (L9) fusionné ou
coordonné (§1.3).
- [ ] 1.3.1 `FilmContext.ImagesCles` sur la mémoire existante de `MarcheDImageCle`
      (`keyframe_world_marche.go`), étendues de l'état complet, mémoire PARTAGÉE entre les contextes
      de la cuisson et de killsource si cela ne change aucune sortie (sinon, découverte consignée).
      *En cours (2026-10-03)* : `marche_images_cles.go` (fichier neuf, option convenue avec la
      campagne) — `FilmContext.ImagesCles()`, un record d'état complet par ancre de la marche du
      film (mémoire du contexte), traversé par `WalkKeyframeFullState`, ses composants, son
      étendue et sa preuve (fermé quand la traversée finit sur l'ancre suivante) ; découpage MPP du
      format posé pour la durée de l'itération et restauré. Tests (`marche_images_cles_test.go`) :
      sur les sept bobines par build, les preuves de la structure rendent archétype par archétype
      les comptes de `KeyframeClosure` (fermés, bornés, composant bloquant), invariants de la
      structure, restauration du contexte à l'arrêt anticipé (mutation « prouver un record qui
      dépasse sa frontière », rouge). Reste : la marque d'élection par record (après la fusion de
      la vague de la campagne, cf. journal). Mémoire partagée : REPORTÉE À L'ÉTAPE 2 par décision de
      l'utilisateur du 2026-10-03 (§2) — neutre par
      construction (la preuve se joue au profil invariant du film, `keyframe_world_preuve.go`),
      mais les deux contextes naissent dans deux couches qui ne partagent que le `source.Film`
      (`killsource/world.go`, `replay/build_from_film.go`) : la partager demande soit un magasin
      attaché au `source.Film` (responsabilité neuve de la porte aux octets, `source.Rev`), soit de
      la surface de façade (`decfilm`) et une plomberie à travers `killsource`, `replay` et
      `replaybuild` — deux constructions que l'étape 2 rend jetables, puisque killsource y
      deviendra un canal de la MÊME marche (`Distribuer`).
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

1. (2026-10-02, lot 1.1) `apps/go-api/internal/archlint/film_layers_deps_helpers_test.go`, lignes
   159-160 et 167 : trois tirets cadratins corrompus dans des commentaires (octets `C3 A2 C2 80 C2 94`,
   le « — » relu en Latin-1 puis réencodé). `no_mojibake_test.go` ne les voit pas : sa classe de
   second caractère couvre la relecture CP1252 (`€`, `”`…), pas les contrôles C1 `U+0080`-`U+009F`
   d'une relecture Latin-1. Hors périmètre (ni fichier du plan, ni sortie) : à confier à un lot de
   nettoyage, avec l'élargissement du ratchet.
2. (2026-10-02, lot 1.2) `DecodeFrameViews` (`grammar/frame_harvest.go`) n'a plus d'appelant de
   production : la production marche par `FilmContext.Trames`. Elle reste la porte de décodage de
   quatre fichiers de test non étiquetés (`grammar/vehicules_v11_tourelle_test.go`,
   `grammar/vehicules_v5_occupation_test.go`, `replay/attachement_phase0_socle_test.go`,
   `replay/visee_composant_pont_test.go`) et d'une trentaine de sondes de recherche ; son corps est
   la même marche par rangs que la production. À retirer quand ces lecteurs passeront sur la
   structure (étape 2).
3. (2026-10-02, lot 1.2) Les sondes de recherche de la campagne (`campagne_marche_research_test.go`,
   `m4b_*_research_test.go`, `mouvement_*_research_test.go`) recopient encore le pilotage
   (`lierLeChunkAuMonde` puis `DecodeFrameViews`) ; le garde-rail du pilotage unique ne regarde que
   la production. Fichiers de la campagne : signalé, non traité.
4. (2026-10-02, lot 1.2) `SortieVueBAutre` (carte de fermeture détaillée) est inatteignable par
   construction depuis que la boucle de records type sa sortie ; la valeur reste tant que la
   colonne de l'outil de la campagne (`research/cmd_fermeture/`) la porte.
5. (2026-10-02, lot 1.2) `decodeInferLoop` garde deux branches inatteignables en production : la
   réparation de chaîne (`const inferRepair = false`) et la resynchronisation validée
   (`inferResyncTargets` toujours nil depuis le retrait de son réglage, 2026-09-05). Code mort
   antérieur à ce plan ; hors périmètre (aucune sortie ne change à le retirer, mais
   `decodeInferLoop` est sous plafond de longueur et la campagne y travaille).

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
- 2026-10-02 : lot 1.1 clos (paquet `grammar/lecture`, ratchets, tailles gelées, empreinte à
  révision constante ; gate tenu). Le lot 1.2 attend son prérequis : le lot L0 de la campagne
  n'est ni commité ni fusionné (accusé de réception de `levelup-83`) ; les lots 1.3 et 1.4 suivent
  1.2. Report VALIDE (dépendance explicite du plan, consigne de l'utilisateur) ; reprise à la
  fusion de L0 dans `feat/v75` : `git merge origin/feat/v75`, régénération de l'empreinte à la
  révision de `feat/v75`, références d'équivalence re-figées, mesure « avant » (critère 4) sur
  cette base.
- 2026-10-02 (soir) : branche poussée sur accord de l'utilisateur (`c0f29f382`, pre-push vert).
  CI VERTE au niveau job (run `37039754462`, 38 min 42 : build et tests Linux et Windows,
  couverture et baseline, golangci-lint, `film-race`, frontend, OpenAPI ; E2E sauté comme prévu
  hors PR vers `main`), Deploy Pre-Check et gitleaks verts. L'utilisateur autorise la communication directe avec les autres sessions et le mode
  multi-agents si nécessaire (coût annoncé avant). Campagne (`levelup-83`) : L0 est en cours
  d'écriture depuis 18 h 20 (quelques heures jusqu'à sa vérification) ; décision de l'utilisateur
  (~19 h 20) : L0 sera fusionné SEUL dans `feat/v75` dès qu'il sera vérifié, sans recuisson à ce
  moment. Attention pour la reprise : L0 touche aussi `frame_infer.go`, `lecteur.go`,
  `traverse.go`, `frame_records.go` et `rev.go` (exception « sauf L0 » du §1.3), et fait MONTER
  `grammar.Rev` — conflit attendu sur `grammar_rev.golden` et `grammar_perimetre.golden` : prendre
  la version de `feat/v75`, régénérer l'empreinte à SA révision.
- Rappels d'exploitation pour les gates avec décodage du lot 1.2 (mémoire du dépôt) :
  `replay-equiv` se lance avec `-repo-root <worktree>` et des jonctions
  `data/cache/film_chunks` et `data/cache/film_manifests` vers le checkout principal (jamais
  `LEVELUP_REPO_ROOT`, qui ferait écrire les références du principal) ; une passe de plus de
  10 minutes se lance par `Monitor`, pas en arrière-plan Bash (enfants orphelins) ; prévenir la
  campagne avant une passe de décodage (une à la fois sur la machine).
- 2026-10-02 (nuit) : lot 1.2 clos. Prérequis tenu : L0 fusionné dans `feat/v75` (`af6e93e23`),
  refusion dans la branche (`ff86624ba`), références d'équivalence re-figées sur cette base
  (`c787c307f`) et passe de référence (durées, pics, sorties killsource) prise sur la même base.
  La marche des trames est écrite une fois et consommée ; différence nulle prouvée (cf. gate du
  lot). Coordination : la campagne confirme que sa vague 1 ne touche ni `localiserLaListe` ni
  `debutParFermetureRangee` et ne pose aucune largeur présumée ; elle a donné la voie libre pour
  les passes de décodage sans pouvoir réserver de fenêtre (trois agents en parallèle). Pendant le
  lot, `feat/v75` a reçu la fusion falcon-behemoth (schéma 77 ; rejeu, service et web, aucun
  fichier de grammaire) : la preuve est faite sur la base du lot, la refusion et le re-figeage des
  références ouvrent le lot 1.3. Précision d'IR-4 (ADR 0037) et des types de `lecture` : une
  occurrence n'est « interprétée » que si la trace capture sa valeur ; une valeur publiée à un
  crochet de l'observation laisse l'occurrence délimitée jusqu'à ce que son canal lise la structure
  (étape 2) — le lien publication -> étendue des états de mouvement et du tir continu est l'objet
  de T3 (1.4.2).
- Rappel d'exploitation (lot 1.2) : avant toute passe de preuve à révision constante, renommer
  `data/cache/film_facts` du worktree (dossier réel, jamais une jonction) — `replay-equiv` relit
  les faits frais, et leur fraîcheur ne regarde que les révisions déclarées ; vérifier
  `depuis_les_faits=false` dans le journal et comparer les faits neufs à l'octet. Même précaution
  avant chaque cuisson mesurée (critère 4).
- 2026-10-02 (nuit) : ouverture du lot 1.3. Coordination avec la campagne (`levelup-83`, 23 h 20) :
  L9 (marche d'image-clé) est dans sa vague 1 mais pas commencé ; la fusion de la vague dans
  `feat/v75` viendra après assemblage, revue et accord de l'utilisateur, pas avant le 2026-10-03.
  Décidé avec elle (option 2) : 1.3.1 s'écrit dans des fichiers NEUFS (`marche_images_cles*.go`),
  sans toucher `keyframe_world*.go` ni `keyframe_closure.go` ; 1.3.2 (`KeyframeClosure`
  consommateur) se branche après la fusion de la vague. L9 change l'ACCEPTATION des ancres
  (génération 0 acceptée, voisin et recalage au-delà de la génération 1), pas la forme de
  `MarcheDImageCle.Records` ni des records d'image-clé — prévision, la campagne préviendra si
  l'assemblage change ces types. `feat/v75` (falcon-behemoth, schéma 77) refusionné dans la
  branche (`9488f34ff`), passe de référence relancée pour re-figer les références.
  Points techniques relevés pour 1.3.1 (à trancher à l'écriture, avec la campagne pour les deux
  premiers) : (a) la marque d'élection par record (`LiaisonImageCleElue`) — `KeyframeRec` ne dit
  pas comment son ancre a été atteinte (voisin, saut, recalage, élection), seuls les comptes par
  payload le disent (`KeyframeWalkStats`) ; la poser demande un champ écrit sur le chemin
  d'élection de `keyframe_world*.go` (fichiers de la campagne) : après la fusion de la vague ;
  (b) les deux bits de tête d'un identifiant d'image-clé : le monde les lie comme RANG DE VUE
  (`BindImageCle(ns, …)`, `vueDeLEspaceDeNoms`), la marche les nomme `Gen` et L9 parle de
  « génération 0 » — la structure les prend comme rang de vue (`Record.Vue`, décision du lot
  1.1), génération inconnue ; (c) `KeyframeClosure` pose le découpage MPP du format du film sur
  le contexte pour la durée de la mesure (`InstallFilmFormatMPP`) : `ImagesCles` fera de même
  pour la durée de l'itération, restauré à la sortie, arrêt anticipé compris ; (d) la mémoire
  partagée entre les contextes de la cuisson et de killsource touche la construction des
  contextes dans `replay` et `killsource` : à analyser, retenue seulement si aucune sortie ne
  change.
- 2026-10-03 (matin) : branche poussée sur accord de l'utilisateur (`6c0a6541d`). `ImagesCles`
  écrit en fichiers neufs (`018c00188`, local). Décision de l'utilisateur : la mémoire de la marche
  des images-clés n'est pas partagée entre la cuisson et killsource à cette étape (reprise à
  l'étape 2, §2). Coordination avec la campagne : elle préviendra dès que sa vague 1, marche
  d'image-clé comprise, sera fusionnée dans `feat/v75` ; la marque « ancre élue » par record est
  pour ce plan, APRÈS cette fusion (l'agent de son lot a pour consigne de ne pas changer la forme
  publique de `MarcheDImageCle.Records` ni des types de records d'image-clé : le champ ajouté à
  `KeyframeRec` restera à ce plan, sans conflit). Calendrier sans date : son lot de la marche
  d'image-clé, relancé le 2026-10-03 au matin, sera assemblé après ses quatre autres lots puis
  contrôlé, et la fusion attend l'accord de l'utilisateur. Report VALIDE de 1.3.2 et de la marque
  d'élection (dépendance explicite du §1.3).
