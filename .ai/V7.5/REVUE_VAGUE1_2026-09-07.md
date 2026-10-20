# Revue adversariale — Vague 1 orchestration (666b02d17 -> 744c5ce37)

Relecteur : contexte frais, lecture seule, worktree `LevelUp-wt-orchestration`, branche
`wt/orchestration-0907`. Diff relu : `git diff 666b02d17 744c5ce37 -- apps` (163 fichiers,
~4600 insertions). Méthode : grille `adversarial-review`, un sous-agent lecture-seule par
lot (Q3, Q5, Q6, Q7, Q8) + vérification directe du superviseur pour Q4 (priorité contractuelle)
et pour les règles transverses (ART, multi-titre, couleurs, query keys, thought_log).

Aucun test de suite complète relancé (gates en cours en parallèle sur ce worktree) : tests
ciblés uniquement, par paquet/fichier.

---

## Constats

### P1-1 — Duplication du motif « réserve échantillon faible » introduite par ce lot (Q8)

- **Fichiers** :
  - `apps/web/src/features/squad/SquadEchangeKpi.tsx:53` (pré-existant, base 666b02d17) :
    `` `${t.kpiSecondary(echange.couverture.brut, echange.couverture.n)} — ${t.lowSample}` ``
  - `apps/web/src/features/tactical/TacticalAnalysisView.tsx:215-222` (nouveau, ce lot) :
    fonction `kpiSecondaryWithReserve`, ligne 222 :
    `` return echantillonFaible ? `${base} — ${t.lowSample}` : base ``
  - `apps/web/src/features/squad/SquadIsolementNuageCard.tsx:224-228` (nouveau, ce lot) :
    `` return pointAttenue(p) ? `${base}<br/>${t.lowSample}` : base ``
- **Déclenchement** : le commentaire ajouté au-dessus de `kpiSecondaryWithReserve`
  (`TacticalAnalysisView.tsx:211-214`) dit explicitement « MÊME FORME que
  `SquadEchangeKpi.tsx` » — la duplication est CONNUE au moment de l'écriture, pas une
  redécouverte accidentelle. Le lot ajoute ainsi une 2e et une 3e implémentation ad hoc du
  même concept (« accoler le libellé de réserve au texte secondaire quand le drapeau est
  vrai ») sans extraire de helper partagé, alors qu'une implémentation existait déjà.
- **Conséquence observable** : violation directe de la règle 6 de CLAUDE.md (« ≤ 2 copies
  d'un même pattern : à la 3e, centraliser dans un helper ET ajouter un garde-rail ») et de
  l'anti-pattern #8 (« factorisation abandonnée »). Effet concret déjà visible : les deux
  nouveaux sites divergent sur le séparateur (` — ` côté Tactique vs `<br/>` côté nuage
  Escouade) pour signaler EXACTEMENT le même concept — un futur changement de forme du
  libellé de réserve (icône, tooltip, wording) demande une modification manuelle
  synchronisée sur 3 fichiers, sans garde-rail qui le rappelle.
- **Correction minimale** : extraire un helper partagé (ex.
  `withLowSampleReserve(base: string, flag: boolean, label: string, sep = ' — ')`) dans un
  module commun (`features/_shared/` ou `lib/`), migrer les 3 sites, poser un test grep
  interdisant un nouveau littéral `${...} — ${t.lowSample}` / `` `${...}<br/>${t.lowSample}` ``
  hors de ce helper.
- **Preuve** : lecture directe des 3 fichiers au HEAD `744c5ce37` ; `SquadEchangeKpi.tsx`
  confirmé inchangé depuis la base (`git show 666b02d17:...` identique).
- **Gravité : P1** (règle écrite du dépôt, pas une préférence de style).

---

## Constats P2 (dette réelle, hors périmètre de ce lot, à consigner — ne pas corriger ici)

- **P2-1** `apps/go-api/internal/archlint/no_mojibake_test.go:98-121` (Q3) — le garde-rail ne
  scanne que `cmd/` et `internal/` sous `apps/go-api/`, jamais `pkg/`, `scripts/`, `tests/`,
  `contracttest/` (23 fichiers `.go` au total, vérifiés indemnes de mojibake aujourd'hui par
  l'agent de revue via un scan indépendant). Le commentaire d'en-tête du test promet une
  couverture totale du module — trou de couverture, pas de défaut latent actuel.
- **P2-2** `apps/go-api/internal/analysis/stats_canonical.go:46` (Q3) — double espace
  résiduel dans un commentaire réencodé (« à  tort »), cosmétique, aucun effet sur la
  compilation ou l'exécution.
- **P2-3** (déjà consigné par l'auteur dans `.ai/thought_log.md`, Q4) —
  `domain.MatchPersonalResult.Outcome`/`OutcomeColor` (`match_view.go:216-283`) : aucun
  lecteur trouvé dans `apps/web/src/features` au-delà des types générés — candidat code
  mort plus large que le périmètre Q4, correctement journalisé comme découverte, non
  traité ici (conforme à la règle « zéro fix opportuniste »).
- **P2-4** (déjà consigné par l'auteur, Q4) — `match_history_explorer_options.go`
  (`computeAvailableOutcomes`) sert un champ `LabelValue.Label` que le web
  (`ExplorerPage.filterOptions.ts`, fonction `withCounts`) ignore totalement (seul `.value`
  et `.count` sont lus) — champ mort côté consommation, sans impact visuel puisque déjà
  inutilisé avant ce lot ; vérifié indépendamment par le relecteur, conforme à la
  découverte déjà journalisée.
- **P2-5** (Q7, agent) — `apps/web/src/lib/replay/heatPaint.guard.test.ts:30-46` : le
  garde-rail anti-duplication fonctionne par recherche de nom de fonction — contournable
  par un renommage trivial d'une future implémentation dupliquée. Limite documentée,
  inhérente à ce type de garde-rail, pas un défaut de ce lot.
- **P2-6** (Q7, agent) — absence d'assertion de position pixel pour prouver le flip Y côté
  rejeu dans `heatPaint.test.ts` (le tactique a cette assertion explicite, le rejeu non) —
  dette préexistante (même absence dans l'ancien `heatmapLayer.test.ts`), non introduite
  par ce lot.
- **P2-7** (Q8, agent) — `apps/go-api/internal/api/server_apiv1.go:319` : `slog.Warn` sans
  variante `Context` alors qu'un contexte est disponible plus haut dans l'arbre d'appel —
  style, pas une violation de la règle anti-silence (l'erreur est bien loguée).

---

## Vérifications qui tiennent (aucun défaut recevable)

- **Q4 (clé canonique d'issue, PRIORITÉ L4)** : table code Halo -> clé canonique exacte et
  vérifiée sur les 3 `outcomes.toml` (`config/titles/{halo_infinite,halo_5,synthetic_title_b}
  /mappings/outcomes.toml`, `raw_code` win=2/loss=3/tie=1/dnf=4, identique à
  `domain.OutcomeWin/Loss/Draw/DNF` et à `outcomeKeyFromHaloCode`,
  `internal/service/outcome_label.go:79-95`). Les 8 champs `outcome` de `api/openapi.yaml`
  portent tous l'enum `win|loss|tie|dnf` (aucun read hors sync avec `generated.ts`/`types.ts`,
  diff cohérent des deux côtés). Dégradation propre confirmée : `OutcomeMappingSet.Canonical/
  Get` sont nil-safe (`internal/games/mappings/outcomes.go:65-124`), `outcomeKey`/`outcomeText`
  renvoient `""` sans adapter câblé, le web (`useOutcomeLabel`, `fieldMappings.ts:203-206`)
  affiche alors une chaîne vide plutôt qu'un mot fabriqué. CSV export
  (`handlers/match_history.go:159-160`) passe bien par `MatchHistoryService.OutcomeText` (texte
  serveur, seule exception documentée). Aucune comparaison de slug introduite par ce lot
  (`no_slug_comparison_test.go` : diff vide). Le filtre « Résultat » de la vue Match
  (`analysis/match_filter.go` + `match-view/i18n.ts:950-957`, valeurs `win|loss|draw|dnf`) est
  un sous-système SÉPARÉ et auto-cohérent des deux côtés (web et Go s'accordent sur `draw`, pas
  `tie`), non touché par ce diff — dette pré-existante hors périmètre, pas une régression. Le
  filtre Explorer (`ExplorerPage.filterOptions.ts:71-78`) utilise ses propres codes numériques
  et clés i18n locales, ignore le champ `Label` de l'API (cf. P2-4) — 0 impact visuel du
  changement de contenu de ce champ.
- **Q5 (flakes CI)** : `pollJobUntilDone` attend `job.IsTerminal()` (succès/échec/annulé/
  interrompu), pas un état intermédiaire ; `OnPersistOK` (`internal/persist/worker.go:186-188`)
  se déclenche strictement après `ACK` et après `Persist`, garanti par l'ordre séquentiel
  intra-goroutine + synchronisation par channel — une régression simulée (Persist sauté,
  suppression WAL non effective) serait détectée par les assertions existantes, aucune
  assertion retirée. `TestWorker_Run_PersistFailure_NoACK` garde son `time.Sleep(200ms)`,
  dette assumée hors périmètre, conforme à l'annonce. 5/5 runs verts sur les deux tests ciblés.
- **Q6 (dette mécanique du rejeu)** : déplacement mécanique pur vérifié fichier par fichier
  (concaténation triée avant/après identique, hors doublon de `package replay`) pour
  `document.go`/`document_chronicle.go`, `usage_summary.go`/`usage_summary_chronicle.go`,
  `replaybuild.go`/`options.go`, `flag_carries_*_test.go`. `SchemaVersion = 48` inchangé
  (`analysis/replay/document.go:27`). 0 fichier sous `testdata/` touché. Garde-rail
  `film_stats_cables_guard_test.go` repointé (regex d'ancrage adaptée au nouveau fichier) sans
  affaiblissement de la boucle d'assertion. `document_chronicle.go` (959 L) vérifié à 956/959
  lignes de commentaire — la justification « chronique = commentaires » est factuellement
  exacte, pas un contournement du seuil de 500 L déguisé.
- **Q7 (peintre de chaleur unique)** : noyau unique `apps/web/src/lib/replay/heatPaint.ts`
  (498 L), flip Y et facteur `k` passés explicitement par l'appelant (pas de détection
  implicite dans le noyau) — `drawHeatmapLayer` (rejeu, flip actif) appelé avec `k: dpr`
  depuis `useReplayStaticLayers.ts:144`, `drawTacticalHeatmap` (tactique, pas de flip) appelé
  avec `k: 1` depuis `TacticalPlanCard.tsx:90`. Comparaison formule-à-formule avec les deux
  anciens fichiers supprimés (`git show 666b02d17:...`) : géométrie identique caractère pour
  caractère des deux côtés. `lib/replay/heatPaint.ts` n'importe rien de `features/`. 33/33
  tests ciblés verts.
- **Q8 (clôture Tactique)** : réserve `echantillon_faible` réellement RENDUE à l'écran (texte
  visible, pas juste une opacité) sur les tuiles KPI Tactique
  (`TacticalAnalysisView.tsx:186-223`, fonction `kpiSecondaryWithReserve` — cf. P1-1 pour la
  duplication) et sur le tooltip du nuage Escouade (`SquadIsolementNuageCard.tsx:210-228`),
  les deux avec un test qui interroge le DOM/l'option ECharts réellement rendue (pas un mock
  qui contourne le composant). Note `matchs_sans_rayon` affichée conditionnellement
  (`TacticalAnalysisView.tsx:196-201`), FR/EN présents dans `tactical.toml` (clé
  `tactical.kpi.no_radius_note`). 4 clés i18n mortes retirées de `tactical.toml` (parité
  FR/EN du retrait vérifiée, pas de résidu asymétrique). `replay.EtatParti`/`Partis` : plus
  aucune occurrence en code exécutable (seulement 2 mentions en commentaire d'historique,
  `death_context.go:74-75` et `death_context_test.go:16`). `TeammatesLeft: 0`
  (`sync/killcollector/isolation_facts.go:211`) : commenté et daté ; l'unique lecteur avec
  logique (`lives_persister.go:229-234`, invariant `Visible+Waiting+OutOfSight+Left==Total`)
  reste vrai car `Total` et le switch à 3 cas de `death_context.go` garantissent la somme sans
  jamais compter sur `Left` — ce n'est pas une donnée fausse servie, c'est une colonne gelée
  documentée des deux côtés. `no_raw_rating_reads_test.go` : passage de 6 à 9 tables
  couvertes, strict sur-ensemble (élargissement, pas d'affaiblissement), commentaire daté.
  `lives_export_test.go` : le test tautologique (comparaison à lui-même) a été remplacé par un
  appel au vrai `ResolveSlotXUID`, devient capable d'échouer sur régression réelle.
  `isolation_facts_integration_test.go` : roster dérivé de `replay.ScanDeaths(film)`, plus un
  mock vide qui aurait fait sauter la vérification. WARN structuré (`slog`, `"err", err`) posé
  aux deux appelants de `settingsStore.Load()` avant dégradation.
- **Règles transverses** : aucun `git diff` sur `internal/archlint/no_slug_comparison_test.go`,
  `internal/sync/no_art_patterns_test.go` (aucune tentative d'affaiblir un garde-rail ART
  existant). Aucun hex ni classe Tailwind couleur ajoutés dans `features/`/`components/`
  (grep ciblé sur les lignes ajoutées du diff web). Aucune `queryKey: [...]` inline ajoutée.
  Aucun `time.Sleep(` ajouté nulle part dans le diff Go. `openapi.yaml`/`generated.ts`/
  `types.ts` cohérents entre eux. Entrée `.ai/thought_log.md` du lot Q4 présente, détaillée,
  documente elle-même 4 découvertes hors périmètre (2 recoupées indépendamment par cette
  revue, cf. P2-3/P2-4).

---

## Décompte

- **P0 : 0**
- **P1 : 1** (duplication du motif « réserve échantillon faible », Q8 — règle 6 CLAUDE.md)
- **P2 : 7** (dette réelle, hors périmètre de ce lot, à consigner sans corriger ici)

## Verdict

La vague 1 est fusionnable dans `feat/v75` sous réserve du P1 : aucune corruption de
données, aucune régression ART, aucune donnée fausse servie à l'écran n'a été trouvée sur
les six lots (Q3, Q4, Q5, Q6, Q7, Q8), et les points sensibles prioritaires (clé canonique
d'issue, `TeammatesLeft: 0`, identité de rendu du peintre de chaleur) tiennent tous à la
vérification sur pièces. Le seul point recevable qui mérite une correction avant merge est
la duplication du motif de réserve d'échantillon faible (P1-1) — un extract-helper de
quelques lignes, pas une réouverture de lot. Les 7 P2 sont à verser aux registres existants
(`.ai/thought_log.md`, `.ai/DECOUVERTES_TACTIQUE_2026-09-07.md` selon le lot) sans bloquer
ce merge.
