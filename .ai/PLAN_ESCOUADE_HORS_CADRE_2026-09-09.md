# PLAN — Compte unique de la population escouade + bornage des pions hors cadre

> Cree le 2026-09-09. Branche : `wt/escouade-hors-cadre`. Worktree :
> `C:/Users/Guillaume/Downloads/Scripts/LevelUp-wt-escouade-hors-cadre` (dedie, cf. regle
> memoire « worktree dedie obligatoire »). Base : `feat/v75` @ `19194fd31`.
>
> **Contrat d'execution : skill `plan-execution` fait foi.** Ordre strict, une etape a la
> fois, aucun report d'etape executable, chaque item statue `[x]` / `[~]` (couvert ailleurs,
> avec reference) / `[!]` (non traite, justification ecrite), zero fix hors perimetre.
>
> **Mode TDD impose par l'utilisateur.** Chaque item de code s'ouvre par un test qui ECHOUE
> pour la bonne raison, et le journal de phase note l'echec observe AVANT l'implementation.
> Un test qui passe du premier coup n'est pas un TDD : il est declare VERROU (ratchet) et
> nomme comme tel.

---

## 1. Objectif et criteres de succes

### Chantier A — le compte de la population escouade

**Defaut mesure (2026-09-09, session du 27 aout).** Trois nombres differents pour une meme
session a l'ecran : la pastille du rail L2 et son « · N matchs » viennent de
`/filters/resolve` (population du JOUEUR PRINCIPAL, aveugle a la composition selectionnee ET
a `filter_exact_composition`), le `SessionMultiSelect` vient de `composition_sessions`
(population reelle de la page). Sur le 27/08 : 7 cote rail, 4 cote page. `3862ff083` avait
unifie le multi-select ; le rail n'avait pas ete touche. **C'est la deuxieme occurrence du
meme defaut** — d'ou le ratchet exige ci-dessous.

**Cadrage produit de l'utilisateur (2026-09-09) :** « un joueur peut avoir une raison de
quitter un match d'une session sans pour autant quitter la session. Le jeu peut crasher, le
PC aussi. » L'appartenance d'un match a la session d'une composition ne doit donc JAMAIS
dependre de la presence du coequipier a la FIN du match.

**Criteres de succes.**

1. Un seul nombre par session a l'ecran, quelle que soit la surface.
2. Ce nombre est la population reelle des tableaux et graphes de la page.
3. Sous « composition exacte », l'ecart est LISIBLE (« 4 sur 7 ») et EXPLIQUE (info-bulle
   nommant, match par match, le coequipier connu qui a ecarte le match).
4. Un ratchet interdit a toute surface escouade de re-compter depuis `/filters/resolve`.
5. Un ratchet interdit d'introduire un filtre de PRESENCE (`present_at_completion`,
   `left_in_progress`, `last_leave_time`) dans les requetes de population escouade.
6. La regle est ecrite une fois pour toutes dans un ADR (EN-only, regle 15 CLAUDE.md).

### Chantier B — le pion hors cadre

**Defaut.** `worldToCanvas` est une projection affine sans bornage ; hors toile, le marqueur
est ecrete par le bitmap du canvas et disparait. A zoom > 1, on perd donc la position des
joueurs hors fenetre — sans aucun repere.

**Critere de succes.** Tout ce qui est borne (cf. perimetre tranche §2) reste visible : une
FLECHE plaquee a la marge interieure, orientee vers la position reelle, dans la couleur
d'equipe, portant le nom du joueur et sa distance hors champ.

---

## 2. Decisions produit — TRANCHEES avant execution (2026-09-09)

| # | Decision | Valeur retenue |
|---|---|---|
| D1 | Ce qu'affiche la L2 sous composition exacte | **« 4 sur 7 » + info-bulle du detail** (matchs ecartes, avec le coequipier responsable nomme) |
| D2 | Forme du repere hors cadre | **Fleche pleine a base concave** (gabarit fourni par l'utilisateur : `Pictures/Screenpresso/2026-09-09_14h26_04.png`), orientee vers la position reelle, couleur d'equipe, **avec le nom ET la distance** |
| D3 | Perimetre du bornage | **Joueurs vivants + porteurs d'objectif (drapeau, crane, bombe) + croix de mort.** Vehicules occupes : HORS perimetre (consigne en Decouvertes) |
| D4 | Ou travailler | **Worktree dedie** `LevelUp-wt-escouade-hors-cadre`, branche `wt/escouade-hors-cadre` |

Deux points laisses a l'implementation, tranches ici pour qu'aucune etape ne s'arrete :

| # | Point | Tranche |
|---|---|---|
| D5 | Identite (losange ami / disque cercle joueur de la page) sur la fleche | **Non.** Le NOM est ecrit a cote : il identifie mieux qu'une forme, et cumuler forme d'identite + fleche rendrait la silhouette illisible a 12 px |
| D6 | Nom et distance sur une croix de mort bornee | **Non.** La croix dure 2,5 s et s'efface ; elle est bornee (D3) mais reste muette — sa position au bord dit deja la direction |

---

## 3. CHANTIER A — compte unique de la population escouade

### Phase A0 — ecrire la regle (ADR + verrou semantique)

- `[x]` A0.1 ADR `docs/adr/0033-squad-session-population-single-count.md` (**EN-only**,
  regle 15). Il ecrit : (a) la definition d'appartenance d'un match a la session d'une
  composition — le main l'a joue ET chaque coequipier selectionne figure sur son equipe
  alliee, **quelle que soit sa presence a la fin** ; (b) sous l'option exclusive, aucun autre
  coequipier connu sur l'equipe ; (c) `composition_sessions[].match_count` est la SEULE
  source d'un compte de session en contexte escouade ; (d) `/filters/resolve` ne sert qu'au
  repli tant que la reponse teammates n'est pas arrivee.
- `[x]` A0.2 Index ADR mis a jour dans `CLAUDE.md` (§ Decisions architecturales).
- `[x]` A0.3 **VERROU (pas TDD)** `internal/service/teammates/composition_presence_test.go` :
  un coequipier selectionne avec `present_at_completion=false` / `left_in_progress=true`
  reste dans `allSquadRows`, dans `composition_sessions` et dans `match_history`. Le test
  passe des l'ecriture — c'est un ratchet semantique, il est nomme comme tel dans son
  en-tete.
- `[x]` A0.4 **VERROU** `internal/service/teammates/no_presence_filter_test.go` : grep sur
  `internal/platform/duckdb/queries_squad.go` + `squad_repo*.go` interdisant
  `present_at_completion`, `present_at_beginning`, `left_in_progress`, `last_leave_time`
  dans les requetes de population escouade (allowlist vide et datee).

**Gate A0** : `cd apps/go-api && go test ./internal/service/teammates/... ./internal/platform/duckdb/...`
+ `go vet ./...`. Aucun item non statue.

**Journal A0 — CLOSE le 2026-09-09.** ADR 0033 ecrit (EN-only, regle 15) ; index CLAUDE.md a
jour. Les deux verrous sont des RATCHETS, pas des TDD, et ils sont declares comme tels dans
leur en-tete : le moteur respectait deja la regle, elle n'etait ecrite nulle part.

**Preuve de mordant (mutation testee, puis revertee).** Ajout temporaire de
`AND p2.present_at_completion` dans `Q30SquadMatchesSharedQuery` ET d'un champ
`PresentAtCompletion` a `domain.AllyParticipant` : les deux tests ECHOUENT avec le bon
message (`no_presence_filter_test.go:69`, `composition_presence_test.go:130`). Revert
verifie a `git status`. Un ratchet qui ne peut pas echouer ne verrouille rien.

**Gate A0 passe** : `go vet ./...` 0 · `go test ./internal/service/teammates/...
./internal/platform/duckdb/...` ok (46,5 s sur duckdb). Aucun item non statue.


### Phase A1 — TDD backend : publier l'ecart

- `[x]` A1.1 **TEST ROUGE** `teammates_exact_composition_test.go` : sous
  `FilterExactComposition=true`, la reponse doit porter, pour chaque session, le compte
  AVANT filtre exclusif et la liste des matchs ecartes avec le(s) coequipier(s) connu(s)
  responsables, nommes. Echec attendu : les champs n'existent pas. **Echec observe** (avant
  tout code) : erreurs de COMPILATION —
  `s1.MatchCountRoster undefined (type domain.SessionLabelEntry has no field or method
  MatchCountRoster)`, idem `ExcludedByExactComposition`, et `undefined:
  domain.CompositionExcludedMatch` (cf. A1.2, renommage). Test ajoute :
  `TestGetPage_ExactComposition_PublishesRosterCountAndExcludedMatches` + fixture
  `newExactCompositionGapRepo` (session S1 : 1 match garde, 4 ecartes — Nilton410 seul,
  passivemarquise seul, xuid sans gamertag resolu, Nilton410+passivemarquise ensemble).
- `[x]` A1.2 `internal/domain/teammates.go` : type `CompositionSessionEntry` (embarque
  `SessionLabelEntry`) + champs `match_count_roster` et `excluded_by_exact_composition` ;
  type `CompositionExcludedMatch` (`match_id`, `start_time`, `map_ui`, `extra_gamertags`).
  `TeammatesPageResponse.CompositionSessions` change de type.
  **DECOUVERTE traitee dans le perimetre** (pas hors-perimetre : c'est le choix du nom de
  CET item) : `domain.ExcludedMatch` existe deja (`match_exclusion.go`, exclusion MANUELLE
  d'un match par l'utilisateur — concept sans rapport). Renomme en
  `CompositionExcludedMatch` pour eviter la collision ; justifie en commentaire GoDoc sur le
  type.
- `[x]` A1.3 `teammates_service_intersect.go` : `filterExactComposition` rend
  `(kept, excluded []domain.SquadMatchRow)` — un seul balayage, jamais deux regles.
  `matchHasExactComposition` gagne un frere `extraPresentOn(team, extraPool)` qui NOMME les
  xuids fautifs (le predicat booleen reste, il est deja verrouille par ses tests). Site
  d'appel `no_raw_squad_intersection_test.go` (ratchet cablage) mis a jour sur les nouveaux
  litteraux (`allSquadRows, _ = ...` / `allSquadRowsForTimeline, excludedForTimeline = ...`).
- `[x]` A1.4 `teammates_service.go` : construction de `CompositionSessions` depuis le
  couple (gardes, ecartes) — `match_count` reste le compte POST-filtre (source unique),
  `match_count_roster` est le compte PRE-filtre de la MEME session. Logique extraite dans un
  nouveau fichier `teammates_service_composition_sessions.go` (limite 500 L, CLAUDE.md regle
  5 — `teammates_service.go` etait deja a 508 L avant ce lot, dette gelee non accrue).
- `[x]` A1.5 Resolution xuid -> gamertag des fautifs via `topRows` (deja charge) ; un xuid
  non resolu s'ecrit `Joueur <4 derniers>` (meme repli que `Q32b`), jamais vide. Verifie par
  le sous-cas "xuid0009" (Gamertag vide dans topRows) du test A1.1.
- `[x]` A1.6 `slog.InfoContext` du denominateur : sessions, gardes, ecartes, fautifs
  distincts. Aucune erreur avalee. Log `teammates.exact_composition_gap` (seulement si
  excluded non vide) ; observe en sortie de test :
  `sessions=1 kept_matches=1 excluded_matches=4 distinct_culprits=3`.
- `[x]` A1.7 OpenAPI : `make openapi-gen` (reflexion Huma sur les nouveaux types domain,
  aucune entree manuelle necessaire dans `openapi_manual_fragment.yaml` — la reponse
  Teammates n'y est pas documentee a la main) ; `make generate-types` pour
  `apps/web/src/lib/api/generated.ts`. **Blocage leve** : `node_modules/` absent du worktree
  dedie (jamais installe) faisait echouer `openapi-typescript` — `make install-web` execute
  (prerequis d'outillage, pas un fix hors perimetre) avant de regenerer.

**Gate A1 passe** : test A1.1 vert · `go test ./internal/...` 100% pass (repo complet,
inclut la suite `internal/api` qui verifie `openapi.yaml` a jour) · `go vet ./...` exit 0 ·
`make go-api-lint` 0 issue · `make generate-types` verifie IDEMPOTENT (2e run : diff
identique, aucun residu).

### Phase A2 — TDD frontend : source unique pour la L2

- `[x]` A2.1 **TEST ROUGE** `apps/web/src/features/squad/squadSessionCounts.test.ts` : une
  fonction pure `squadSessionCount(label, compositionSessions, fallback)` rend
  `{ shown, total }` depuis `composition_sessions` SEUL des que la reponse teammates est la,
  et ne retombe sur `/filters/resolve` que pendant le chargement initial. Echec attendu : le
  module n'existe pas. **Echec observe** (avant tout code) : `Failed to resolve import
  "./squadSessionCounts"` (Vite). 9 cas ecrits, dont "SOURCE UNIQUE : la composition prime
  meme si son nombre est plus BAS que le repli" (scenario 7 vs 4 explicite de l'ADR 0033).
- `[x]` A2.2 `squadSessionCounts.ts` : le module pur (`squadSessionCount` +
  `squadSessionShownCount`, variante 1-nombre pour SessionMultiSelect). `mergeSessionCounts`
  ABSORBE (retire de `squadPending.ts`, tests migres vers `squadSessionCounts.test.ts`, ancien
  bloc de test remplace par un commentaire de renvoi) et son ancien site d'appel
  (`SquadLayout.tsx`) migre — regle 6 CLAUDE.md.
- `[x]` A2.3 **TEST ROUGE** `PeriodSessionRail.test.tsx` : quand un `sessionCount` est
  fourni, le rail affiche « 4 sur 7 » et NON le compte de son store. **Echec observe** (avant
  code) : 2/10 tests rouges (le texte "4 sur 7"/"3 match" attendu absent, `sessionCount`
  silencieusement ignore par le composant) ; le 3e cas ("sans sessionCount, rendu inchange")
  passait deja au vert — verifie AVANT le code que c'est bien parce que le comportement par
  defaut est deja correct (pages hors perimetre), pas un faux negatif du test.
- `[x]` A2.4 `PeriodSessionRail.tsx` : prop optionnelle `sessionCount?: (label) =>
  {shown, total} | undefined`, cablee UNIQUEMENT sur `SessionRail` (le seul mode qui lisait le
  compte brut `session.match_count` sans passer par la prop `matchCount` existante — c'est
  exactement le point du defaut mesure 7 vs 4). Nouveau texte `matchCountOfRosterSuffix`
  ("4 sur 7 matchs" / "4 of 7 matches") a cote de `matchCountSuffix` existant ; reutilise ce
  dernier quand `shown === total` (pas de "X sur X" redondant). Aucune page qui ne passe pas
  la prop ne change de rendu (verifie par le 3e cas A2.3 + tests existants tous verts).
- `[x]` A2.5 `SquadLayout.tsx` : le rail recoit `sessionCount={getSessionCount}` (compo
  composition, source unique). `getSessionCount`/`getSessionShownCount` passent tous deux par
  `squadSessionCounts.ts` (`SessionMultiSelect.getMatchCount` <- `squadSessionShownCount`).
  **DECOUVERTE traitee dans le perimetre de l'item** (necessaire au ratchet A2.6, allowlist
  vide) : `totalAfter` (lecture directe de `total_matches_after_filters`, population du
  JOUEUR PRINCIPAL) alimentait `matchCount` du rail (tous modes hors session unique) ET la
  visibilite du bouton "Voir les matchs". Retire des deux : `matchCount` n'est plus passe
  (aucun total composition fiable pour period/multi-session/all-time dans ce lot — afficher
  rien plutot qu'un nombre faux) ; le bouton se fie desormais a `squadEntryMatchId` seul
  (1er match de `match_history` = population escouade, deja suffisant : la condition
  `totalAfter > 0` etait redondante). L'extraction du repli `/filters/resolve`
  (`session_options.all_sessions`) est centralisee dans `resolveSquadSessionFallback`
  (squadSessionCounts.ts) — SquadLayout.tsx ne lit plus `session_options` directement.
- `[x]` A2.6 **RATCHET** `apps/web/src/features/squad/singleCountSource.guard.test.ts` :
  aucun fichier de `features/squad/` (recursif — `charts/`, `components/`, `v2/` inclus) ne
  lit `total_matches_after_filters` ni `session_options` pour en tirer un compte de matchs.
  Allowlist vide et datee (2026-09-09) : SEUL `squadSessionCounts.ts` (module canonique) et
  le garde-rail lui-meme (il NOMME les litteraux en prose) sont exclus du scan, par
  CONCEPTION (comparaison de nom de fichier dans le walker), pas par exception nominative.
  **Preuve de mordant** : le premier jet du garde-rail (avant le refactor
  `resolveSquadSessionFallback`) a bien ECHOUE sur `SquadLayout.tsx` (lecture directe de
  `session_options`) — la correction est venue APRES cette detection, pas avant.

**Gate A2 passe** : `make check-types` 0 erreur (2 iterations : un premier `null` non
autorise dans `ResolvedWithSessionOptions.all_sessions`, corrige) · `make test-web` 7032
passed / 17 skipped / 0 failed (suite complete, aucune regression) · `npx eslint` sur les 8
fichiers touches : 0 issue.

### Phase A3 — lisibilite de l'ecart (D1)

**EXECUTEE le 2026-09-09 (lot 1.5, worktree d'integration `LevelUp-wt-vague1`,
branche `feat/vague1-integration`).** Le conflit de coordination note ci-dessus (i18n.ts
partage) est leve : ce lot travaille dans le worktree d'integration dedie, sur la
branche d'integration de la vague 1 (deja fusionnee), donc plus de risque de fusion
concurrente sur `features/squad/i18n.ts`.

- `[x]` A3.1 **TEST ROUGE** rendu : sous composition exacte avec des ecartes, la L2 porte
  « 4 sur 7 » et une info-bulle listant chaque match ecarte (date, carte, coequipier
  responsable). Fait : `PeriodSessionRail.test.tsx` (2 cas : hint present -> tooltip
  revele au survol/focus ; hint absent -> aucune info-bulle) + `squadCompositionGapHint.test.tsx`
  (6 cas : contenu pur de l'info-bulle, accord singulier/pluriel FR/EN, repli quand le
  coequipier n'est pas resolu). **Echec observe** (avant tout code) : `squadCompositionGapHint.test.tsx`
  echoue par resolution de module (`Failed to resolve import "./squadCompositionGapHint"`,
  Vite) ; `PeriodSessionRail.test.tsx` echoue sur l'assertion `getByRole('tooltip')` (aucune
  info-bulle rendue, le prop `hint` n'existait pas encore).
- `[x]` A3.2 Composant d'info-bulle : reutilise le patron d'aide d'en-tete existant
  (`InfoTooltip`/`HeaderLabelTooltip`, convention V73-L2 2.4c) — pas de nouveau primitif.
  `squadCompositionGapHint.tsx` ne fait que BATIR le `ReactNode` du contenu ; le rendu
  passe par `InfoTooltip` (deja utilise par `cardTitleAdornment`), cable dans
  `PeriodSessionRail.tsx` (`SessionCountLabel`, nouveau) via un prop `hint?: ReactNode`
  generique — le composant shell reste agnostique du domaine escouade.
- `[x]` A3.3 i18n FR **et** EN (`features/squad/i18n.ts`, parite par typage
  `Record<Locale, T>`). Nouveau groupe `compositionGap` (heading, excludedLine, culpritUnknown)
  dans l'interface `SquadText` + `FR_TEXT` + `EN_TEXT`. FR sans anglicisme : « 4 sur 7
  matchs » (deja livre en A2), « ecarte : X etait dans ton equipe » (accord pluriel
  « etaient » quand plusieurs coequipiers responsables).
- `[x]` A3.4 Couleurs : tokens semantiques uniquement (skill `color-tokens`), zero hex,
  zero classe Tailwind de couleur — verifie par grep (`grep -rn "#[0-9a-fA-F]\{6\}"` et
  classes `text-{red,green,blue,yellow,amber,rose}-*` : 0 occurrence dans les fichiers
  touches). Les classes utilisees (`text-xs`, `text-muted-foreground`, `font-medium`,
  `space-y-*`) sont deja les tokens/utilitaires neutres du depot, aucune couleur ajoutee.
- `[~]` A3.5 Revue navigateur sur la session du 27/08 : **couvert par le superviseur, revue
  de vague** (consigne d'execution du lot 1.5 : « ne lance pas de navigateur »). La logique
  est verrouillee par les tests A3.1 (contenu exact « Nilton410 était dans ton équipe » /
  « Nilton410 et passivemarquise étaient dans ton équipe » couvert par
  `squadCompositionGapHint.test.tsx`), la revue visuelle reste a faire par le superviseur.

**Gate A3 passe** (2026-09-09) : `make check-types` (tsc -b) 0 erreur ·
`cd apps/web && npx vitest run src/features/squad src/components/shell` → 73 fichiers,
648 tests verts · `npx eslint` sur les 8 fichiers touches (`squadSessionCounts.ts`,
`squadSessionCounts.test.ts`, `squadCompositionGapHint.tsx`, `squadCompositionGapHint.test.tsx`,
`i18n.ts`, `SquadLayout.tsx`, `PeriodSessionRail.tsx`, `PeriodSessionRail.test.tsx`) : 0 issue.

---

## 4. CHANTIER B — bornage des pions hors cadre

### Phase B0 — TDD : la geometrie, pure

- `[x]` B0.1 **TEST ROUGE** `edgeClamp.test.ts` : `edgeMarkFor(c, view, margeEcran, echelle)`
  rend `null` dedans ; au bord droit, `x = w - marge` et l'angle pointe a droite (0 rad,
  verifie aussi a gauche, ± PI) ; dans un coin, les deux axes sont bornes et l'angle vise la
  diagonale (-PI/4 sur un cas a 45°) ; la distance rendue est en METRES (pixels /
  `scaleOf(view)`, verifie avec deux cadrages d'echelle differente projetant le MEME point
  canvas) ; l'inversion de Y est respectee (un point au sud du monde sort par le bas du
  cadre) ; `echelle` grandit la marge (`marge = margeEcran * echelle`). **Echec observe**
  (avant tout code) : `Failed to resolve import "./edgeClamp" from
  ".../edgeClamp.test.ts". Does the file exist?` (Vite). Un premier jet du sous-cas `echelle`
  s'est aussi trompe (attente sur le bord BAS alors que l'inversion Y place (50, 85) pres du
  HAUT du canevas) — corrige AVANT de regarder `edgeClamp.ts`, en recalculant la projection a
  la main plutot qu'en devinant.
  **ECART DE CHEMIN AU PLAN, CORRIGE SUR PIECE** : le plan situait le module dans
  `apps/web/src/lib/replay/edgeClamp.ts`. Verifie sur pieces (regle 4) : `CanvasView` /
  `projectTo` / `scaleOf` ne vivent PLUS dans `lib/replay/` depuis le lot K3 (2026-09-05) — ils
  vivent dans `apps/web/src/features/match-replay/model/replayView.ts`, dont l'en-tete est
  justement la citation du plan (« aucune seconde regle de projection »). Aucun fichier de
  production de `lib/replay/` n'importe quoi que ce soit de `features/` (verifie par grep : les
  3 seules occurrences sont des fixtures de *test*) — l'inverse casserait le sens de
  dependance bas/haut du dépôt. Le module et son test vivent donc dans
  `apps/web/src/features/match-replay/model/` (`edgeClamp.ts` / `edgeClamp.test.ts`), a cote de
  `replayView.ts` qu'ils consomment, et sont couverts par le garde-rail
  `replayView.guard.test.ts` (« un seul cadrage, une seule projection ») comme tout le reste de
  la feature. Consigne aussi en Decouvertes (§8).
- `[x]` B0.2 `edgeClamp.ts` : le module pur (voir ecart de chemin ci-dessus). `EdgeMark { at,
  angle, distanceM }` ; `edgeMarkFor` projette via `projectTo(view, c)`, borne aux deux axes a
  `margeEcran * echelle`, et ne recalcule jamais l'inversion Y (deja faite par `projectTo`).

**Gate B0 passe** (2026-09-10) : `npx vitest run src/features/match-replay/model/edgeClamp.test.ts`
7/7 verts · `npx vitest run src/features/match-replay/model` 64 fichiers / 931 tests verts
(aucune regression sur les gardes, dont `replayView.guard.test.ts`) · `npx tsc -b --force`
(purge `node_modules/.tmp` prealable) exit 0.

### Phase B1 — TDD : le gabarit de la fleche

- `[x]` B1.1 **TEST ROUGE** `offscreenChevron.test.ts` : le trace ferme 4 sommets (pointe,
  arriere-gauche, encoche, arriere-droite), il est a l'echelle de l'ECRAN (`k`, jamais du
  canevas — verifie via `recordingContext`, l'appel `scale` porte `CHEVRON_SIZE_PX * k`), il
  est pivote de `angle`, et il est rempli de la couleur passee. **Echec observe** (avant tout
  code) : `Failed to resolve import "./offscreenChevron" from ".../offscreenChevron.test.ts".
  Does the file exist?` (Vite). Couvre aussi B1.3 dans le meme fichier (memes item de code,
  meme TDD) : `offscreenLabelAnchor` (le cote INTERIEUR, oppose a `angle`) et
  `drawOffscreenLabel` (contour puis remplissage, aucun `strokeText` quand `labelStroke` est
  vide).
- `[x]` B1.2 `layers/offscreenChevron.ts` : le dessin. Gabarit normalise pointant +X :
  pointe `(1, 0)`, arriere-gauche `(-0,75, -0,7)`, encoche `(-0,35, 0)`, arriere-droite
  `(-0,75, 0,7)` — la base concave du gabarit fourni par l'utilisateur. `save`/`restore`
  encadrent le geste (meme regle que `drawRotatedSprite`).
- `[x]` B1.3 L'etiquette « nom · distance » : posee du COTE INTERIEUR de la fleche
  (`offscreenLabelAnchor`, a l'oppose de `angle` par `LABEL_GAP_PX * k`), encre de lisibilite
  passee par l'appelant (meme convention que `replayLabels.ts` — jamais `readInk` appele
  d'ici, jamais un litteral). Le texte lui-meme (« nom · distance ») est compose par
  l'APPELANT (B2) : ce module ne recoit qu'une chaine deja faite — l'unite de distance (B4.3)
  n'a donc pas a etre tranchee ici.

**Gate B1 passe** (2026-09-10) : `npx vitest run src/features/match-replay/layers/offscreenChevron.test.ts`
9/9 verts · `npx vitest run src/features/match-replay/layers` 50 fichiers / 652 tests verts ·
`npx tsc -b --force` (purge `node_modules/.tmp` prealable) exit 0.

### Phase B2 — cablage des marqueurs joueurs

- `[x]` B2.1 **TEST ROUGE** `replayMarkers.test.ts` : un joueur vivant hors fenetre dessine
  la fleche a la marge, et NE dessine ni trainee, ni cone de visee, ni anneau d'apparition,
  ni marqueur d'etage. **Echec observe** (avant tout code) : les 3 nouveaux cas echouent sur
  `count(ops, 'rotate')` attendu a 1, obtenu 0 — le calque ne bornait encore rien. La distance
  attendue est calculee via `edgeMarkFor` (meme fonction que `edgeClamp.test.ts`), jamais
  recopiee a la main, pour ne tester que le CABLAGE. `OFFSCREEN_MARGIN_PX` (16 px ecran de
  reference) devient la marge CANONIQUE, exportee depuis `edgeClamp.ts` pour que B3 la
  reutilise sans copie.
- `[x]` B2.2 `drawLivingTrack` : branche hors cadre — `edgeMarkFor` d'abord ; si hors fenetre,
  `drawOffscreenChevron` + `drawOffscreenLabel` (si un nom est resolu) puis `return` immediat
  (meme structure que la branche pion embarque juste au-dessus) : rien d'autre n'est atteint.
- `[x]` B2.3 **TEST ROUGE** croix de mort hors fenetre : la croix est plaquee a la marge,
  meme fondu, SANS nom ni distance (D6). **Echec observe** (avant tout code) : meme signature
  (`rotate` attendu 1, obtenu 0).
- `[x]` B2.4 `drawDeathMark` : branche hors cadre — `ctx.globalAlpha = DEATH_ALPHA * fade`
  POSE AVANT `drawOffscreenChevron` (le chevron ne fixe pas l'alpha lui-meme, `save`/`restore`
  la traverse) : meme calcul de fondu que la croix en X, aucune etiquette.
  **DECOUVERTE traitee dans le perimetre** : `MarkerStyle` gagne un champ obligatoire
  `offscreenLabelOf: (name, meters) => string` (texte deja compose, resolu par l'appelant —
  meme convention que `ink`/`labelStroke`). Deux fichiers de test construisaient un
  `MarkerStyle` complet (`replayMarkers.test.ts`, `replayAimCone.test.ts` — extrait de l'un a
  l'autre le 2026-09-06) : les deux mis a jour. L'i18n de l'unite (B4.3) est resolue ICI, pas
  reportee : `offscreenMarkerFmt: (name, meters) => string` ajoute a `i18nContract.ts` +
  `i18n.ts` (FR/EN, meme valeur « m » — symbole international, pas un anglicisme, mais la
  parite de typage CLAUDE.md n°1 est tenue) ; cable dans `ReplayCanvas.tsx`
  (`offscreenLabelOf` resout `REPLAY_TEXT[locale].offscreenMarkerFmt`, `locale` ajoute aux
  dependances du `useCallback`).

**Gate B2 passe** (2026-09-10) : `npx vitest run src/features/match-replay` 181 fichiers /
2617 tests verts, 1 skipped (aucune regression) · `npx tsc -b --force` (purge prealable)
exit 0 · `ReplayTeams.perf.test.tsx` : 3 skipped — gate `REPLAY_PERF=1` + temoin
`data/cache/replays/...` absent de ce worktree dedie (comportement inchange, aucun module
qu'il importe n'est touche par ce lot — verifie par lecture de ses imports) ·
`npx eslint --max-warnings=0` sur les 10 fichiers touches (B0-B2) : 1 avertissement
PRE-EXISTANT et INCHANGE, `ReplayCanvas.tsx:532` (`zoneInk.outline` manquant aux
dependances d'un AUTRE `useCallback`, sans rapport avec ce lot — confirme par `git diff`,
la ligne n'est pas dans le diff de ce lot).

### Phase B3 — porteurs d'objectif (D3)

- `[ ]` B3.1 **TEST ROUGE** par calque : `flagCarriesLayer`, `bombCarrierLayer`,
  `skullCarrierLayer` — le glyphe du porteur hors fenetre est plaque a la marge.
- `[ ]` B3.2 Cablage des trois calques sur `edgeMarkFor` (jamais une copie de la regle).
- `[ ]` B3.3 `vipCrownLayer` : statuer `[x]` (meme traitement) ou `[!]` (justifie) — la
  couronne suit le marqueur du joueur, verifier sur piece avant de trancher.

**Gate B3** : `make test-web` · `make check-types`.

### Phase B4 — parite export et finition

- `[ ]` B4.1 Verifier sur piece que l'export video passe par les MEMES peintres
  (`bindPainters` / `composeScene`) : la parite doit etre gratuite, pas supposee.
- `[ ]` B4.2 Revue navigateur : zoom 2x et 3x sur un match temoin, plusieurs joueurs hors
  cadre du meme cote, lisibilite des etiquettes.
- `[ ]` B4.3 i18n de l'unite de distance (FR + EN) si un libelle est necessaire.

**Gate B4** : `make check-types` · `make test-web` · revue navigateur consignee.

---

## 5. Revues (exigence utilisateur : « de multiples revues »)

- `[ ]` R1 Skill `adversarial-review` sur le diff du chantier A, contexte frais, AVANT le
  chantier B.
- `[ ]` R2 Skill `adversarial-review` sur le diff du chantier B.
- `[ ]` R3 Skill `delivery-checklist` sur l'ensemble, avant toute demande de commit.
- `[ ]` R4 Entree `.ai/thought_log.md` (regle obligatoire CLAUDE.md) avant de rendre la
  main.

Sobriete des quotas (regle memoire) : les revues R1/R2 se font **une fois par chantier**,
jamais par lot.

---

## 6. Gates transverses avant « c'est livre »

```bash
cd apps/go-api && go build ./... && go vet ./... && go test ./...
make go-api-lint
make check-types
make test-web
make generate-types
```

`go test -tags=integration ./...` : **`[~]` non requis** — aucun chemin persist/sync n'est
touche par ce chantier (lectures escouade + calques canvas uniquement). A re-statuer si le
perimetre bouge.

---

## 7. Reprise de session

Avancement = les cases de ce fichier. Reprendre a la premiere case non cochee de la premiere
phase non close. `git log --oneline -10` sur `wt/escouade-hors-cadre` pour l'etat reel.

## 8. Decouvertes (a consigner, PAS a traiter)

- Vehicules occupes hors cadre : un pion embarque n'est pas dessine (`vehiclesLayer` porte
  l'info) ; sans bornage du vehicule, il disparait sans repere. Hors perimetre (D3).
- Collision de nom `domain.ExcludedMatch` (A1.2) : traitee DANS le perimetre de l'item
  (renommage en `CompositionExcludedMatch`), consignee ici pour memoire seulement — aucune
  action restante.
- Worktree dedie sans `node_modules/` (A1.7) : un worktree fraichement cree n'a jamais
  `npm install` — `make generate-types`/`make check-types`/`make test-web` echouent tant que
  `make install-web` n'a pas tourne une fois. Observation generale pour les prochains
  chantiers en worktree dedie, pas une action a mener ici (deja fait pour ce worktree).
- Socles d'EQUIPEMENT non publies : la voie `ti=37` filtre sur le prefixe `powerup_`, donc
  grappin / repulseur / mur / capteur / ecran / propulseur / translocateur poses sur un
  socle de carte ne sont publies nulle part. Le catalogue de cartes connait pourtant 407
  points d'apparition `equipment`. Lot a part entiere.
- Etat actif des deployables (mur 19, capteur 22) : voie `charges-remaining` en reserve
  depuis la decision utilisateur du 2026-08-16.
- B0.1/B0.2 : le plan situait `edgeClamp.ts` dans `apps/web/src/lib/replay/` ; le module vit
  en realite dans `apps/web/src/features/match-replay/model/` (a cote de `replayView.ts`, qui a
  deplace `CanvasView`/`projectTo`/`scaleOf` hors de `lib/replay/` le 2026-09-05, lot K3).
  Traite DANS le perimetre de B0.1 (ecart de chemin d'un document qui rote plus vite qu'il
  n'est maintenu, pas un changement de perimetre) — consigne ici pour memoire seulement,
  aucune action restante. A repercuter si un futur plan cite a nouveau `lib/replay/` pour le
  cadrage.
