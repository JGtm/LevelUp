# Plan — retours du 2026-10-17 sur Ascension > Tactique

Contrat d'exécution : skill `plan-execution` (fait foi). Branche `feat/retours-tactique`,
worktree `../LevelUp-wt-retours-tactique`, base `ef74ce9ac` (origin/feat/v75). Lot jumeau en
parallèle : `feat/retours-escouade`, aucun fichier commun attendu hors `.ai/thought_log.md`.

## 0. Décisions de l'utilisateur (messages du 2026-10-17)

- « Il faudrait une vue de type cockpit de la même hauteur que la vue du mode rejeu, et on
  devait avoir un genre de mini rejeu de 3 secondes quand on clique sur une tuile de la colonne
  de droite ; je n'ai pas vu ça. »
- Puis : « Peu importe ce que le plan prévoyait, je n'ai rien, ni aucun placeholder. C'est ça le
  problème : on ne peut pas distinguer un manque de fonction d'un manque de données quand on est
  utilisateur. » « Respecte bien la convention de l'app pour ce type de placeholder. »
- Accord (« oui ») sur : la route de séquence renvoie la cause ; la page affiche toujours un état
  (séquence, ou état vide qui dit pourquoi) ; au clic sur une mort, la séquence se JOUE seule de
  −4 s à +3 s, bouton pour la relancer ; plan à la même règle de hauteur que le rejeu (plafond,
  légende et barre de lecture comptées).

## 1. Constats sur pièces (2026-10-17)

- Vue D existante : barre de 8 étapes statiques (`features/tactical/TacticalPlanCoequipiers.tsx:150`
  `BarreDEtapes`, rendue par `PiedDuPlan` l.123/141 seulement si `vue && seq`), fiche d'étape
  `TacticalMortDetail.tsx:109`. Aucune animation.
- Requête : `features/tactical/queries.ts:395-408` (`retry: false`, 404 silencieux) ;
  `vueDeMort.ts:50-55` `sequenceDemandee` = null sauf mort + `replay_available` + xuid.
- Route Go : `apps/go-api/internal/api/handlers/tactical_sequence.go`, montée dans le groupe du
  rejeu (`server_apiv1.go` ~818, `RequireCapability(CapReplay)` + `ReplayGate`). Service
  `internal/service/tactical_service_sequence.go` `Sequence` : 404 indiscernable pour capability
  `film.replay_artifact` absente, match non ouvrable, carte sans univers/rayon, artefact absent,
  horloge du film absente (`OriginMs == nil` / `FrameIntervalMS <= 0`), victime sans position
  (`ErrSequenceSansMort`). Pistes ré-échantillonnées 200 ms sur [−8 s, +3 s].
- Garde : `handlers/replay_local_gate.go` (`LocalOnlyReplay`), ratchets
  `internal/archlint/tactical_sequence_gate_test.go`, `tactical_background_local_gate_test.go`.
- Signal « analyse en cours » existant : `analysis_pending` / `analysis_pending_parts`
  (commits `047b97f4c`, `597d7adfc`, `ad2e1dec2`), `components/ui/analysis-pending-slot.tsx`.
- Convention d'état vide : bloc parent + `EmptyStateNotice` (`components/ui/empty-state.tsx`)
  dedans ; `AnalysisPendingSlot` pour une analyse en cours.
- Hauteur : `features/tactical/useHauteurDuPlan.ts` + `cockpit.logic.ts:30-32` (min 360, marge
  40, aucun plafond, pied du plan non compté) contre le rejeu
  `features/match-replay/hooks/useReplayViewport.ts` (marge 16, chrome déduit, pas 8 px) et
  `useReplayView.ts:57-58` (`CANVAS_HEIGHT_MIN = 360`, `CANVAS_HEIGHT_CEILING = 720`).

## 2. Étapes

Statuts : `[x]` fait · `[~]` couvert ailleurs · `[!]` non traité (justification).

### T1 — La route de séquence dit pourquoi (Go)
- [x] T1.1 Erreurs typées du service pour chaque cause « données » : analyse du film en cours
      (même règle que `analysis_pending_parts`, pas de seconde règle), film non analysé
      (artefact absent), film à réanalyser (horloge absente), carte non mesurée (univers ou
      rayon absent), position de la mort absente. Match non ouvrable et capability absente
      restent des refus (pas de fuite d'information).
- [x] T1.2 Le handler les sert avec un code machine stable dans le corps (Huma), statut cohérent ;
      contrat openapi + `make generate-types`. Le garde du rejeu inchangé (ratchets verts) : si le
      rejeu est fermé, la vue de séquence est absente comme le lien de rejeu (fonction absente).
- [x] T1.3 Tests Go : une cause = un test.

### T2 — La page affiche toujours un état (web)
- [x] T2.1 Contribution sélectionnée dans la colonne droite : mort avec séquence servie →
      lecteur (T3) ; mort sans séquence → état vide conforme (bloc parent + `EmptyStateNotice`,
      ou `AnalysisPendingSlot` si analyse en cours) avec la cause en texte factuel FR/EN ;
      `replay_available` faux côté liste → même état « Film non analysé » sans requête ;
      tuile de frag (ou autre face) → état « La séquence se lit sur les morts » (FR/EN).
- [x] T2.2 La requête ne se tait plus : cause lue dans le corps de l'erreur ; erreur réseau ou
      inconnue → état vide générique, jamais rien.
- Textes : factuels, neutres, sans personne ni imperatif (voir mémoire Tactique v2).

### T3 — Mini-rejeu animé
- [x] T3.1 Au clic sur une mort dont la séquence est servie, lecture automatique de −4 s à
      +3 s sur le plan principal, à partir des pistes 200 ms déjà servies (interpolation
      entre échantillons), traînes comme aujourd'hui ; arrêt sur +3 s.
- [x] T3.2 Bouton « Rejouer » ; la barre d'étapes reste utilisable (clic = pause + saut à
      l'étape, comme un curseur) ; `prefers-reduced-motion` : pas de lecture automatique,
      l'étape de la mort affichée.
- [x] T3.3 Instants écrits à l'écran : horloge du rejeu, conversion partagée existante
      (`matchClock`), jamais une seconde conversion. Logique pure (horloge de lecture,
      interpolation) dans un `*.logic.ts` testé ; composant sans logique métier.

### T4 — Hauteur du cockpit = règle du rejeu
- [x] T4.1 Le plan suit la même règle que la toile du rejeu : même plafond (720), même plancher,
      même marge basse, même pas ; la légende et la barre de lecture (pied du plan) sont
      déduites. Réutiliser le helper du rejeu (pas de copie, règle CLAUDE.md n°6) en déclarant
      l'import croisé dans `lint-cross-feature-imports` (précédent : commit `44349fcd0`), ou le
      remonter dans un module partagé.
- [~] T4.2 Colonnes latérales : toujours à la hauteur du plan. (couvert : TacticalPage.test.tsx l.251-256 tient `self-stretch` + `contain: size` sur les deux colonnes ; la carte du plan, enveloppée d'un div qui devient l'élément de grille, reste ce qui fixe la hauteur de la rangée)

### T5 — Clôture
- [x] T5.1 `make check-types`, `make test-web` (vitest hors sandbox), lint web,
      `go test` des paquets touchés + `internal/archlint`, `make go-api-lint`.
- [x] T5.2 Entrée `.ai/thought_log.md` ; découvertes hors périmètre listées au rapport.
- [x] T5.3 Commits sur `feat/retours-tactique` uniquement, un par étape ; pas de push.

### T6 — Libellés d'état sans action ni jargon (retour du 2026-10-17)
- [x] T6.1 « Film à réanalyser » retiré (l'utilisateur n'a aucun contrôle dessus) : constat factuel sur la donnée,
      « Séquence absente de l'analyse du film ». Code machine renommé `sequence_film_clock_missing` (erreur Go
      `ErrSequenceFilmSansHorloge`), cause web `film_sans_horloge`, clés `tactical.mort.seq_missing_clock_*`.
- [x] T6.2 Relecture des autres états (FR et EN, Go et web) : « Film non analysé », « Film en cours d'analyse », « Rayon du
      radar inconnu », « Position de la mort absente », « Séquence indisponible », « La séquence se lit sur les morts » sont
      des constats, inchangés ; messages d'erreur Go de `port.ErrSequence*` neutralisés (plus d'« artefact »). Garde-rail :
      test de chaînes sur `tactical.mort.seq_missing_*`, `seq_only_deaths_*`, `seq_pending`, `seq_loading` (aucun mot de
      réanalyse, cuisson, artefact, reconstruction, relance).

## 3. Hors périmètre

Activation du rejeu en prod (release) ; bots dans les positions ; autres pages.

## 4. Journal d'exécution

### T1 (2026-10-17)
- Causes servies (404, code dans le corps ; `handlers/tactical_sequence.go` `sequenceCauses`) :
  `sequence_analysis_pending`, `sequence_film_not_analysed`, `sequence_film_clock_missing`,
  `sequence_map_unmeasured`, `sequence_no_death_position`. Le refus (`replay_not_available`) garde
  match non ouvrable, carte étrangère, forme invalide et garde du rejeu.
- « Analyse en cours » = cause `replay` du résolveur partagé (`AnalysisPendingResolver`), jamais une
  seconde règle ; câblé par `WithAnalysisPending` (wire `Tactical`).
- Contrat openapi : inchangé (le code vit dans `ApiError.code`, `default` documenté) ; `TestOpenAPIYAMLIsUpToDate`
  vert, `make generate-types` sans objet.
- Écart au plan (T2.1) à porter en T2 : la requête de séquence part aussi quand `replay_available` est
  faux, car seul le serveur sait distinguer « analyse en cours » (état dédié) de « film non analysé », et
  seul le 404 `replay_not_available` du garde distingue « fonction absente » de « donnée absente ».

### T2 + T3 (2026-10-17)
- Un seul commit : la zone « séquence » du pied du plan héberge à la fois les états (T2) et le lecteur (T3)
  (`TacticalSequenceSlot.tsx`), les deux partagent `VueDeMort` / `useVueDeMort`.
- T2.1, écart au plan : la requête de séquence part aussi quand `replay_available` est faux (voir T1) ; le texte
  « Film non analysé » vient donc du serveur, et « Film en cours d'analyse » (`AnalysisPendingSlot`) n'est possible
  que par lui. Un 404 `replay_not_available` (garde du rejeu, match non joué) ou un titre sans capability `replay`
  = aucune zone, comme le lien de rejeu. Frag, entrée, réapparition : « La séquence se lit sur les morts ».
- T2.2 : `causeDeLErreur` lit le code de l'`ApiError` ; erreur réseau ou code inconnu = « Séquence indisponible ».
  Tant que la cause est « analyse en cours », la requête se refait toutes les 60 s (`ANALYSIS_PENDING_REFETCH_MS`).
- T3.1 : `useLectureDeSequence` (rAF, horloge à abonnés lue par `useSyncExternalStore` : la tête de lecture n'est
  pas un état React), bornes = première et dernière étape servies, pas borné à 100 ms, arrêt sur +3 s ; la scène est
  `sceneALInstant` (interpolation linéaire entre échantillons consécutifs, jamais à travers une coupure).
- T3.2 : bouton Pause / Reprendre / Rejouer, barre d'étapes (clic = pause + saut), « Instantané » ; mouvement réduit
  = pas de lecture automatique, étape de la mort affichée.
- T3.3 : instants écrits = `instant_affiche_ms` de l'étape de la mort (déjà converti par le serveur sur l'horloge du
  coup d'envoi) + décalage, plancher 0 ; `formatClock` partagé. Aucune conversion d'axe refaite côté web. (`matchClock`
  est l'outil du document de rejeu complet, absent ici : la séquence ne charge que ses pistes.)

### T4 (2026-10-17)
- `useHauteurDuPlan(carte, cadre)` = `useReplayViewport(carte, cadre).freeHeight` : le helper du rejeu, pas une copie
  (paire `tactical=>match-replay/hooks/useReplayViewport` déclarée dans `lint-cross-feature-imports`). Règle avant :
  hauteur = fenêtre − haut du cadre − 40 px, plancher 360, aucun plafond, pas de pas, légende et barre non comptées.
  Règle après (celle du rejeu) : plancher 360, plafond 720, marge basse 16, pas de 8 px, mesure refaite 120 ms après le
  dernier redimensionnement, chrome déduit = hauteur de la carte entière (titre, états, légende, zone séquence) moins
  celle du cadre.
- `useReplayViewport` : le second paramètre accepte tout élément (le cadre du plan) et l'observateur de taille est
  optionnel (jsdom), comme le patron du dépôt ; comportement du rejeu inchangé (son test passe).
- `PLAN_HAUTEUR_MIN_PX` et `PLAN_MARGE_BAS_PX` supprimés (code mort) ; `hauteurDisponible` supprimée.

### T5 (2026-10-17)
Gates rejoués sur l'arbre final : `go test` (service, port, api, api/handlers, api/wire, analysis/tactical, archlint,
contracttest) ; `go vet` des paquets touchés ; `golangci-lint run --new-from-merge-base=origin/main ./...` (0 issue) ;
`npx tsc -b --force` ; `npm run lint` (0 erreur) ; `npm run lint:colors` / `lint:fields` ; `node tools/lint-cross-feature-imports.mjs`
(pas d'échec, 7 occurrences préexistantes sous plafond) ; `node tools/knip-ratchet.mjs` (0/0/0) ; `npx vitest run` complet.
Ordre des commits : T1, T2+T3 (indissociables : la zone séquence héberge les deux), T4, T5.

## 5. Découvertes (non traitées, hors périmètre)

- En production, aucun artefact de rejeu n'est cuit et le garde local ferme la route : le 404 `replay_not_available`
  fait taire la zone. Le jour où le garde s'ouvre, tout match sans artefact (donc la quasi-totalité) affichera « Film non
  analysé » : c'est exact, mais la fenêtre de 24 h de « en cours d'analyse » ne couvre que les matchs récents.
- `go-vet` du pre-commit tourne sans l'environnement CGO : il liste des dizaines de paquets « build constraints exclude
  all Go files » et réussit quand même ; le bruit masque une vraie erreur éventuelle.
- Les tests de chaînes de la feature (`mort.logic.test.ts`) interdisent « non mesuré », « heure » et les possessifs dans
  tout le préfixe `tactical.mort.*` : à retenir pour toute nouvelle clé de ce préfixe.
- L'instant écrit pendant la lecture suppose que l'étape de la mort (`offset_ms == 0`) porte `instant_affiche_ms` ; sans
  coup d'envoi connu, l'horloge écrit le décalage (« −1,5 s ») : couvert par un test, jamais vu sur le parc.
