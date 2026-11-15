# Rapport d'analyse 8.1.2 (kpi_lusr_i18n_p10_p13), 2026-11-09

Rapport brut d'un agent en lecture seule, annexe de `.ai/PLAN_V8_1_2_2026-11-09.md`. Le plan fait foi en cas d'écart.

Analyse terminée : le dépôt n'est pas modifié (même `git status` qu'au départ) et aucune base du dépôt n'a été ouverte en place. Les mesures ont été faites sur des copies à froid et des tests hors dépôt, sous `C:\Users\GUILLA~1\AppData\Local\Temp\claude\17871fa88c29004d\21dc6e8a-f21e-4f7f-80b4-739a03f09e2d\scratchpad\analyse-kpi\` (`data\`, `epx\`, `p11\`, `overlay\`). Le serveur actif est l'appli installée (données sous `%LOCALAPPDATA%\LevelUp`) ; `data/levelup.lock` du dépôt n'est pas tenu.

## Ordre recommandé

P10 est indépendant. P13 avant P11, car les deux touchent `behavioral.go` et le message du tilt change de nature avec P11 (seuil absolu, chute en points). P12 est indépendant de P13 mais suit P11 pour le texte.

## P10 : matchs dont le calcul ne converge pas

**(a) État constaté**
- `ep/match2team.go:184-187` : `runner.Run()` s'arrête à `MaxIters=500` (`:116`), tolérance 1e-4 sur `max(|Δπ|, √|Δτ|)` (`ep/gaussian.go`). Il rend alors `ErrNotConverged`.
- `skill_v2_shadow.go:285-292` : `noteComputeFailure` compte `skippedNotConverged` et sort, sans tenir le groupe. Les matchs suivants avancent le filigrane par-dessus.
- Backlog périmé : la classification est en `lusr_gap_scan.go:109-116`, pas `:116-120`. Le panneau (`admin_lusr_gaps.go`), l'auto-heal (`scheduler/data_health_lusr.go`, OFF par défaut) et le rejeu retombent tous sur la même non-convergence.
- **Reproduit sur données réelles** (rejeu en mémoire de la chaîne LUSR, outcome, rosters, poids de temps joué, quit, comptes) :

| Joueur | Matchs éligibles | Non convergents |
|---|---|---|
| Madina97294 | 1198 | 16 |
| JGtm | 1160 | 12 |
| Chocoboflor | 587 | 7 |
| XxDaemonGamerxX | 38 | 0 |

  - Rosters de 4v5 à 16v16, soit environ 1 à 1,4 %, ce qui colle aux 18 de la prod.
  - Hors périmètre de cette mesure : décalage d'escouade (non activé dans ma simulation), mise à jour croisée des états entre propriétaires (chaque joueur est rejoué isolément) et recalibrage des hyperparamètres par groupe (ces derniers sont lus en base : seules les bornes de paliers y figurent).
- **Cause trouvée** : ce sont les observations de comptes (frags/morts : SumFactor + PriorFactor par joueur contre la moyenne adverse) qui divergent.
  - Sans elles, **35 non-convergences sur 35 convergent** (poids de temps joué et pénalités d'abandon conservés).
  - **L'amortissement ne règle pas le problème.** Un amortissement fixe rattrape 3 à 6 de ces matchs sur 12 à 16 selon le joueur (0,3 avec 3000 itérations). Aucun réglage essayé (0,1 à 0,5, 500 à 20000 itérations) n'en rattrape plus de la moitié. Le backlog et l'ADR 0024 §139/178 (« Phase 3e damping ») sont donc à corriger.
- **Un correctif ne change pas que les matchs non convergents** (preuve en rejeu, états propagés) :
  - Avec le repli sans comptes, 844 des 1182 notes déjà posées de Madina changent (922/1148 pour JGtm, 434/580 pour Chocoboflor).
  - Sur Madina, 186 notes bougent d'au moins 5 points de rating et 85 d'au moins 20 ; le maximum est de 130 points.
  - Même un seul match rattrapé change 170 notes ultérieures de Madina.
  - **La révision n'est donc pas un choix** : sans elle, un rejeu (bouton admin, auto-heal) produirait un historique différent de celui stocké.

**(b) Correctif recommandé : une seule issue.** Repli sans observations de comptes quand le solveur ne converge pas, puis montée de `LUSRRev`. Le repli est côté appelant (qui a `ctx` pour journaliser) : sur `errors.Is(err, skillv2.ErrNotConverged)`, journaliser un `slog.WarnContext`, incrémenter un expvar, puis rappeler `UpdateTwoTeamWithCountsEP` avec une copie de `counts` où `Kills` et `Deaths` valent nil pour tous. Les `Weight` et les `Quit` sont conservés et `flattenCountInputs` n'émet plus d'observation.

Distinguer les non-convergents dans le rapport n'est pas retenu : il faudrait résoudre en lecture seule ou persister un marqueur, et le trou persiste. Une fois le repli posé, ces matchs reçoivent une note et le trou disparaît.

Fichiers fermés à modifier ou créer :
1. Créer `apps/go-api/internal/sync/skill/skill_v2_solver_fallback.go` : helper `withoutCountObservations`, appel de repli, expvar `levelup.lusr_v2.solver_count_fallback_total`.
2. Modifier `apps/go-api/internal/sync/skill/skill_v2_shadow.go` (`:657-664`, quelques lignes seulement : le fichier fait 765 lignes, dette gelée).
3. Modifier `apps/go-api/internal/sync/skill/skill_v2_options.go` (commentaires de `WithMatchFailureCounter` / `noteComputeFailure`, qui disent « un nouvel essai échouerait pareil »).
4. Modifier `apps/go-api/internal/sync/skill/scoring_revision.go` : `LUSRRev` passe à une nouvelle valeur datée.
5. Modifier `apps/go-api/internal/sync/scoring_revision_test.go` : ajouter l'entrée de la nouvelle révision avec le **même** empreinte `ef84e6e1…3443a`, l'ancienne entrée restant en place (rien de hashé ne change, car le repli est de la logique).
6. Modifier `apps/go-api/internal/analysis/skill_v2/ep/match2team.go` (commentaire `:105-114` : amortissement mesuré insuffisant, cause = comptes).
7. Créer ou étendre `apps/go-api/internal/analysis/skill_v2/ep/match2team_convergence_test.go` (fixture ci-dessous).
8. Créer `apps/go-api/internal/sync/skill/skill_v2_solver_fallback_test.go`.
9. Corriger le commentaire de `apps/go-api/internal/sync/skill/skill_v2_compute_failure_test.go` (« aucune donnée de test ne fait diverger le solveur » est devenu faux).
10. Docs : `docs/adr/0024-*.md` (amendement : taux mesuré, amortissement non retenu, cause, repli), `docs/adr/0047-*.md` (« Partial failure »).
11. `.ai/BACKLOG.md` (retirer l'entrée) et `.ai/thought_log.md` (entrée obligatoire).
12. Au moment de la release 8.1.2 : `docs/CHANGELOG.md`, `docs/FR/CHANGELOG.md`, `docs/RELEASE_NOTES.md` et `docs/FR/RELEASE_NOTES.md`, une ligne pour dire que l'historique est recalculé au premier démarrage et que les notes de ces joueurs bougent.

**(c) Tests et commandes de gate**
- Fixture déterministe trouvée, à prendre dans le test `ep` : 4v6, tous à `N(25, 25/3)`, poids de l'équipe B `[1 1 1 1 .5 .5]`, A gagne, β=25/6, τ=25/300, `DrawMargin = Φ⁻¹(0,55)·√10·β`.
  - Kills A `[17 17 15 1]`, morts A `[9 13 10 13]`, kills B `[18 7 20 3 0 12]`, morts B `[9 3 11 20 7 18]`.
  - Attendu : `ErrNotConverged` avec comptes (500 itérations), convergence sans comptes.
- Test `sync/skill` : le repli rend une note et incrémente le compteur ; un échec sans comptes reste classé `ErrNotConverged` ; `Quit` et `Weight` sont conservés.
- Commandes :
  - `cd apps/go-api && go test ./internal/analysis/skill_v2/... ./internal/sync/skill/ ./internal/sync/ -run 'Skill|ScoringParametersMatchRevision|ScoringRevisionComposition|NoteComputeFailure'`
  - `go test -tags=integration ./internal/sync/...` (obligatoire, voie sync/persist)
  - `make go-api-lint`, puis `make gate-push`.

**(d) Décision restante** : monter `LUSRRev` ou non. Je recommande de la monter (cohérence rejeu/historique, ADR 0047 D14), avec une ligne dans les notes de version. Sans la montée, les 18 matchs prod restent des trous jusqu'à un rejeu manuel, et ce rejeu changerait tout l'historique des joueurs concernés.

**(e) Effort et modèle** : M, Sonnet. Algo sensible mais petit ; le contrôle est par test.

**(f) Dépendances** : aucune avec P11, P12 ou P13. Le rejeu du premier démarrage est celui de l'ADR 0047 (quelques secondes). Penser à l'annoncer, comme pour une recuisson.

## P11 : FDA en quotient sur Solo / Escouade

**(a) État constaté**
- `platform/duckdb/patterns_repo.go:450-455` : `row.KDA = (kills + assists/2) / max(1, deaths)`. C'est le quotient avec un poids de 1/2, ni le net ni le tiers.
- Chaîne complète : `loadShared` (`:153-215`) → `mergePatternRows` → `analysis/patterns/context.go:80` et `:102` (`AvgKDA`, champ JSON `avg_kda`) → `SquadVsSoloCard.tsx:38,50`.
- `behavioral.go` : `meanKDA` (`:112-122`), `detectTilt` (`:56-62`, `:65`), `detectSessionFatigue` (`:137-139`). Les seuils sont `TiltKDADropPct=0,25` (`types.go:46,70`) et la sévérité haute à 0,35 en dur (`:65`).
- La colonne native `match_participants.kda` existe, sans NULL (28 993 lignes).
  - Elle vaut `k + a/3 − d` sur 28 840 lignes. Les 153 restantes ne l'égalent pas, ce qui reste à examiner : c'est probablement un compteur NULL, pas un écart de formule.
  - La requête de `loadShared` ne la lit pas.
- Fonction canonique : `analysis.AggregateKDA(k,a,d,matches)` (`indicators.go:38`) ; SQL moyen `AVG(mp.kda)` (`compare_repo.go:94`). `CombatEfficiency` est le quotient interne du score de performance, à ne pas utiliser. Gabarit de repli : `effectiveKDA` (`service/session_compare_stat_helpers.go:66`).
- ADR 0006 : son texte écrit encore `KDA = (K+A)/max(1,D)` et n'a aucun amendement. CLAUDE.md règle 9 et `lib/fda.ts` font foi pour le net.

**(b) Correctif**
- Lire `p.kda` dans la requête de `loadShared`, avec repli sur `analysis.AggregateKDA(k,a,d,1)` si NULL (même valeur que la valeur API native).
- Seuil d'alerte : la chute relative `(hors − pendant)/hors` n'a plus de sens avec un net, pouvant être négatif ou proche de zéro.
  - Sur les copies locales, 3 joueurs : moyenne de FDA par match = +4,98 (Madina), −0,39 (JGtm), +0,19 (Choco) ; écart-type 4,4 à 9,1.
  - Avec le même seuil relatif de 0,25, la moyenne hors-série de JGtm est ≤ 0 dans 522 des 1163 fenêtres, et 174 sur 567 chez Choco.
- Proposé : un seuil absolu en points de FDA, `TiltFDADropPoints`. Calibré sur 2 959 fenêtres glissantes de 50 matchs avec série de 3 défaites ou plus, à taux de déclenchement égal à l'ancien (55 %) : **2,0 points** pour le tilt, **4,0 points** pour la sévérité haute (ancien 35 %, soit 31 % des fenêtres).
  - Le taux de 55 % est lui-même élevé. Une valeur plus stricte (2,5 / 5) est à trancher.
  - Pour les chutes de FDA en 50 matchs, la médiane est de 3,27 points chez Madina contre 1,6 chez Choco : une normalisation par écart-type est la piste rigoureuse, au prix de la lisibilité.
- Fatigue : seuil « pente < 0 », invariant d'échelle ; `FatigueSessionCovPct=0,6` est conservé. Sur les fenêtres de 50 matchs, le déclenchement est de 1283 (ancien) contre 1239 (nouveau), soit −3,4 %.
- Effet visible : la carte Solo/Escouade de JGtm sur 50 matchs passe de 1,11 / 1,04 à −0,69 / −0,29, et la formule d'affichage est alors enfin celle de l'ADR.

Fichiers fermés :
1. `apps/go-api/internal/platform/duckdb/patterns_repo.go` (SQL + `patternSharedRow.KDA` + `mergePatternRows`).
2. `apps/go-api/internal/analysis/patterns/behavioral.go` (tilt en points absolus ; `meanKDA`).
3. `apps/go-api/internal/analysis/patterns/types.go` (config + commentaire `KDA`).
4. `apps/go-api/internal/analysis/patterns/behavioral_test.go`.
5. `apps/go-api/internal/platform/duckdb/patterns_repo_test.go` et `patterns_repo_db_test.go` (colonne `kda` dans le schéma de fixture, attendus 2.4 → net : m1 = `10+4/3−5 = 6,33`, m2 = `0`).
6. `docs/adr/0006-canonical-indicators-and-units.md` (amendement EN : le texte écrit encore le quotient).
7. `.ai/BACKLOG.md`, `.ai/thought_log.md`.

`SquadVsSoloCard.tsx` ne change pas : il affiche `avg_kda` tel quel et le libellé vient de `useMetricLabel('kda')` (FDA en fr).

**(c) Tests**
- Mettre à jour `TestDetectTilt_FiveConsecutiveLosses` : les fixtures `makeRowWithKDA(2, 2.0)` / `(3, 1.0)` supposaient une chute relative de 50 % ; il faut un net équivalent, par exemple +5 hors série contre +1 pendant, soit 4 points ≥ 4,0 → sévérité haute.
- Ajouter : un cas moyenne hors-série ≤ 0 (la détection se fait quand même sur la chute en points) et un cas sous seuil (`TestDetectTilt_NotEnoughLosses` conservé).
- `go test ./internal/analysis/patterns/ ./internal/platform/duckdb/ -run 'Pattern|Tilt|Fatigue'`.
- `make go-api-lint` ; pas de régénération OpenAPI (champ inchangé).

**(d) Décision restante** : le seuil (2,0 / 4,0 à taux égal, ou plus strict). Je recommande 2,5 / 4,5 (plus strict que le taux de déclenchement actuel de 55 %) ; c'est le seuil du produit.

**(e) Effort et modèle** : M, Sonnet.

**(f) Dépendances** : P13 d'abord. Le message « chute de FDA » de P13 doit déjà porter l'écart en points. P12 reprend la même formule.

## P12 : formule de la FDA fausse dans l'Aide

**(a) État constaté**
- `lib/fda.ts` fait foi : `(frags + assistances/3 − morts) / nb_matchs`, 3 paliers (négatif, de 0 à 1, au-delà de 1).
- `features/help/i18n/fr.ts:77-82` : « Ratio FDA », formule en quotient `/ max(1, morts)`, 15/4/6 → 2,72, et la phrase « retourné directement par l'API » est fausse. `en.ts:77-82` : « KDA », `(k + a) / max(1, d)`, assistances « comptent pleinement », 15/4/6 → 3,17. Le backlog tient donc (lignes 80 et 82).
- Découvertes voisines, à signaler seulement :
  - `config/titles/halo_infinite/mappings/fields.toml:132` et `config/titles/halo_5/mappings/fields.toml:134` : description « Éliminations + Assistances/2 − Morts » (faux, c'est /3).
  - Le tableau du score de performance (`fr.ts:45,47` ; `en.ts:44`) appelle « Ratio FDA » / « KDA » une mesure interne qui est en réalité `CombatEfficiency = (k+a)/max(1,d)` (`sync/performance.go:109`).
  - `features/ascension/i18n/fr.ts:94` a « KDA hebdomadaire » (le FDA est toujours « FDA » en fr).

**(b) Texte exact** (entrée `'FDA'` en fr, `'KDA'` conservé en en : le libellé UI anglais est KDA, fr FDA)

*fr* (`term: 'FDA'`) :
- `definition` : « Bilan net d'un match : les frags, plus un tiers des assistances, moins les morts. C'est la valeur que l'API Halo renvoie directement pour chaque match. Elle peut être négative : un joueur qui meurt plus qu'il ne produit de frags effectifs passe sous zéro. Le poids d'un tiers reflète le fait qu'aider à abattre un adversaire vaut moins que le faire soi-même : trois assistances valent un frag.\n\nSur plusieurs matchs, le FDA est la moyenne des FDA de chaque match : le net cumulé (frags + assistances / 3 − morts) divisé par le nombre de matchs. Ce n'est jamais un quotient par les morts, et il ne faut pas le confondre avec le K/D (frags / morts), qui est une autre mesure.\n\nLa lettre D vient de « Décès », mais Halo affiche le mot « Morts » dans son interface : les deux désignent la même chose. Lecture : négatif = déficitaire, de 0 à 1 = à l'équilibre, au-delà de 1 = bénéficiaire. Mesure de référence mais sensible au volume : un joueur très actif peut afficher le même FDA qu'un joueur passif. Le score de performance complète cette lecture en intégrant 12 autres mesures. »
- `formula` : `FDA d'un match = frags + assistances / 3 − morts\nFDA sur plusieurs matchs = (frags + assistances / 3 − morts) / nombre de matchs`
- `example` : `15 frags, 4 assistances, 6 morts → FDA = 15 + 4/3 − 6 ≈ 10,33\n4 frags, 3 assistances, 9 morts → FDA = 4 + 3/3 − 9 = −4,00 (sous zéro : plus de morts que de frags effectifs)\n0 frag, 0 assistance, 0 mort → FDA = 0,00\nSur les deux premiers matchs : FDA = (10,33 − 4,00) / 2 ≈ 3,17`

*en* (`term: 'KDA'`) :
- `definition` : 'Net result of a match: kills, plus one third of assists, minus deaths. This is the value the Halo API returns directly for each match. It can go negative: a player who dies more than their effective kills goes below zero. One third reflects that helping to down an opponent is worth less than doing it yourself: three assists are worth one kill.\n\nOver several matches, KDA is the average of the per-match values: the net total (kills + assists / 3 − deaths) divided by the number of matches. It is never a quotient by deaths, and it is not K/D (kills / deaths), which is a separate measure.\n\nReading: negative = deficit, 0 to 1 = even, above 1 = surplus. A reference measure but volume-sensitive: a very active player can show the same KDA as a passive one. The performance score complements this reading with 12 other measures.'
- `formula` : 'KDA of a match = kills + assists / 3 − deaths\nKDA over several matches = (kills + assists / 3 − deaths) / number of matches'
- `example` : '15 kills, 4 assists, 6 deaths → KDA = 15 + 4/3 − 6 ≈ 10.33\n4 kills, 3 assists, 9 deaths → KDA = 4 + 3/3 − 9 = −4.00 (below zero: more deaths than effective kills)\n0 kills, 0 assists, 0 deaths → KDA = 0.00\nOver the first two matches: KDA = (10.33 − 4.00) / 2 ≈ 3.17'

Vérifié à la main : 15+4/3−6 = 10,333 ; (10,333−4)/2 = 3,167 ; agrégat : frags 19, assistances 7, morts 15 → 6,333/2 = 3,167.

Fichiers fermés : `apps/web/src/features/help/i18n/fr.ts` et `en.ts`. `contract.ts` est inchangé (seules les valeurs des champs). `i18n.test.ts` n'a rien à changer, mais ajouter une assertion `expect(flatten('fr')).not.toContain('max(1, morts)')` (et l'équivalent en).

**(c) Tests** : `cd apps/web && npx vitest run src/features/help` ; `make check-types`.

**(d) Décision restante** : trancher ce qui est hors périmètre, à ajouter seulement avec l'accord de l'utilisateur. Je recommande de traiter ensemble :
- les deux `fields.toml` (`Assistances/2` → `/3`) ;
- la ligne « Ratio FDA » du tableau du score (renommer en « Efficacité de combat » / « Combat efficiency », formule `(frags + assistances) / max(1, morts)`) ;
- le fr « KDA hebdomadaire ».

**(e) Effort et modèle** : S, Haiku (mécanique et contrôlable par test et grep).

**(f) Dépendances** : après P11 (même formule). Aucune autre.

## P13 : alertes de comportement composées en français côté Go

**(a) État constaté.** Liste exhaustive de 11 chaînes (le backlog en donne 10 sur ces lignes ; c'est la concaténation `engagement −%.0f%%` qui compte à part) :
- `behavioral.go:69` `"KDA chute de %.2f → %.2f"` ; `:71` `" ; engagement −%.0f%%"` ; `:76` `"%d+ défaites consécutives"` ; `:158` `"Sessions de %d+ matchs"` ; `:159` `"KDA diminue sur %.0f%% des sessions analysées"`.
- `behavioral_engagement.go:70` `"%d matchs récents sous les seuils habituels"` ; `:71` `"Engagement perso : %.0f%% vs habitude ; %.0f%% vs le lobby"` ; `:104` `"30 matchs sans progression"` ; `:105` `"Précision stable à %.1f%% (σ=%.3f)"` ; `:163` `"Progression LOWESS plate sur 30+ matchs"` ; `:164` `"Plafond de performance à %.0f pts (top %d moyen : %.0f)"`.
- Exception à retirer : `archlint/no_french_label_literal_test.go` (entrée `behavioral.go` = 2 et `behavioral_engagement.go` = 3, vers `:92-93`) ; mettre aussi à jour le compte total du commentaire de tête.
- Consommateurs : handler `api/handlers/patterns.go:99-128` → `patterns.Analyze` → `BehaviorAlertList.tsx:45`, qui affiche `trigger` et `evidence` tels quels. Clé de cache web déjà localisée (`lib/query/keys.ts:450`, `['progression', slug, title, 'patterns', locale, n]`). Pas de cache serveur sur cette route.
- La locale arrive par `i18n.FromContext(ctx)` (le handler utilise aujourd'hui `ctxkeys.Locale(ctx)` en chaîne). `AnalyzeInput` n'a aucun champ de locale, et `internal/i18n` n'importe aucun paquet LevelUp, donc `patterns` peut l'importer sans cycle.
- Piège : `progression/coach/generator.go:389-402` recopie `Trigger` et `Evidence` dans les `Params` d'une alerte persistée. Aucun site ne renseigne `PatternReport` en production (champ jamais alimenté), mais si on le câblait, la phrase serait figée dans une seule langue. À noter dans le commentaire, sans traiter.

**(b) Correctif**
- Ajouter à `AnalyzeInput` un champ `Locale i18n.Locale`.
- `analyzeBehavior(rows, cfg, loc)` et chaque `detect*(rows, cfg, loc)` composent via `i18n.Msg(loc, clé, i18n.Params{…})`.
- Le handler passe `Locale: i18n.FromContext(ctx)`.
- Clés dans `internal/i18n/messages/patterns.toml` (11, une par chaîne) :
  - `patterns.behavior.tilt.trigger` : « {n}+ défaites consécutives » / « {n}+ consecutive losses » ;
  - `patterns.behavior.tilt.evidence` : « FDA moyen {before} → {during} » / « Average KDA {before} → {during} », et `tilt.evidence_engagement` avec `; engagement −{pct}%` pour éviter la concaténation de fragments ;
  - `fatigue.trigger` : « Sessions de {n}+ matchs » / « Sessions of {n}+ matches » ;
  - `fatigue.evidence` : « FDA en baisse sur {pct}% des sessions analysées » / « KDA dropping in {pct}% of analysed sessions » ;
  - `engagement_drop.trigger` / `.evidence`, `accuracy_plateau.trigger` / `.evidence`, `perf_ceiling.trigger` / `.evidence` (avec `{n}`, `{max}`, `{topn}`, `{mean_top}`).
- Paramètres numériques pré-formatés avec `fmt.Sprintf` aux mêmes précisions qu'aujourd'hui (`%.2f`, `%.0f`, `%.1f`, `%.3f`). Il n'existe aucun helper de décimale localisée : le séparateur reste le point dans les deux langues, comme l'actuel.
- Dans les messages de tilt et de fatigue, écrire « FDA » en fr et « KDA » en en (libellé UI).
- Les constantes `30` en dur (`plateauWindow`, `PerfCeilingWindow`) passent en paramètre `{n}`, sans changer le texte rendu par défaut.

Fichiers fermés :
1. `apps/go-api/internal/analysis/patterns/behavioral.go`
2. `apps/go-api/internal/analysis/patterns/behavioral_engagement.go`
3. `apps/go-api/internal/analysis/patterns/engine.go`
4. `apps/go-api/internal/analysis/patterns/types.go` (champ `Locale`)
5. `apps/go-api/internal/api/handlers/patterns.go`
6. `apps/go-api/internal/i18n/messages/patterns.toml`
7. `apps/go-api/internal/analysis/patterns/behavioral_test.go` (les 10 appels `detect*` prennent `i18n.Default`)
8. `apps/go-api/internal/archlint/no_french_label_literal_test.go` (retirer les 2 entrées)
9. `.ai/BACKLOG.md`, `.ai/thought_log.md`

La réponse d'API est inchangée (mêmes champs, chaînes résolues) : pas de régénération des types.

**(c) Tests**
- Ajouter `TestAnalyze_LocalisationDesAlertes` : même jeu de lignes en `fr` et `en` ; `en` ne contient aucun caractère accentué ni mot français ; `fr` contient les phrases attendues ; une locale inconnue (par exemple `i18n.Locale("zz")`) retombe sur le français.
- `cd apps/go-api && go test ./internal/analysis/patterns/ ./internal/i18n/ ./internal/archlint/ ./internal/api/handlers/ -run 'Pattern|Behavio|Message|Catalogue|FrenchLabel|I18n'`
- Les gardes `i18n_messages_test.go` (clés littérales présentes, clé appelée au moins une fois) et la complétude du catalogue le vérifient.

**(d) Décision restante** : aucune bloquante.

**(e) Effort et modèle** : S, Haiku (mécanique, contrôlable par `archlint` et par test).

**(f) Dépendances** : à faire avant P11 (même fichier ; le tilt devient un seuil absolu en points, et son message doit citer cet écart).

## Effort global

| Point | Effort | Modèle |
|---|---|---|
| P10 | M | Sonnet |
| P11 | M | Sonnet |
| P12 | S | Haiku |
| P13 | S | Haiku |

Gate final : `make gate-push` (~25 min) et `go test -tags=integration ./...` pour P10.
