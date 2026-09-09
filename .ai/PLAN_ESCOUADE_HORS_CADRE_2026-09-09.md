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

**EN ATTENTE (2026-09-09) — pas `[!]`, coordination inter-sessions.** A3.3 touche
`apps/web/src/features/squad/i18n.ts`, fichier que le worktree PARTAGE
(`LevelUp-go-migration`) modifie actuellement sans commit (cf. regle memoire « worktree
dedie obligatoire »). Toucher ce fichier depuis ce worktree dedie risquerait un conflit de
fusion sur du travail en vol ailleurs. Le lot s'arrete donc proprement apres le gate A2 ; le
superviseur ordonnera l'execution de A3 (et du chantier B) une fois ce conflit potentiel
leve. Aucun item A3 n'est traite ni juge hors-perimetre : la case reste a cocher au prochain
lot.

- `[ ]` A3.1 **TEST ROUGE** rendu : sous composition exacte avec des ecartes, la L2 porte
  « 4 sur 7 » et une info-bulle listant chaque match ecarte (date, carte, coequipier
  responsable).
- `[ ]` A3.2 Composant d'info-bulle : reutilise le patron d'aide d'en-tete existant
  (`cardHint*`, convention V73-L2 2.4c) — pas de nouveau primitif.
- `[ ]` A3.3 i18n FR **et** EN (`features/squad/i18n.ts`, parite par typage
  `Record<Locale, T>`). FR sans anglicisme : « 4 sur 7 matchs », « ecarte : X etait dans
  ton equipe ».
- `[ ]` A3.4 Couleurs : tokens semantiques uniquement (skill `color-tokens`), zero hex,
  zero classe Tailwind de couleur.
- `[ ]` A3.5 Revue navigateur sur la session du 27/08 : la L2 doit lire « 4 sur 7 » et
  l'info-bulle nommer Nilton410 (1 match) et passivemarquise (2 matchs).

**Gate A3** : `make check-types` · `make test-web` · revue navigateur consignee.

---

## 4. CHANTIER B — bornage des pions hors cadre

### Phase B0 — TDD : la geometrie, pure

- `[ ]` B0.1 **TEST ROUGE** `apps/web/src/lib/replay/edgeClamp.test.ts` : `edgeMarkFor(c,
  view, margeEcran, echelle)` rend `null` dedans ; au bord droit, `x = w - marge` et l'angle
  pointe a droite ; dans un coin, les deux axes sont bornes et l'angle vise la diagonale ;
  la distance rendue est en METRES (pixels / `scaleOf(view)`) ; l'inversion de Y est
  respectee (monde +Y haut, toile +Y bas).
- `[ ]` B0.2 `edgeClamp.ts` : le module pur. Il vit dans `lib/replay/` avec la projection —
  aucune seconde regle de projection (cf. en-tete de `replayView.ts`).

**Gate B0** : `make test-web` (edgeClamp vert) · `make check-types`.

### Phase B1 — TDD : le gabarit de la fleche

- `[ ]` B1.1 **TEST ROUGE** `offscreenChevron.test.ts` : le trace ferme 4 sommets (pointe,
  arriere-gauche, encoche, arriere-droite), il est a l'echelle de l'ECRAN (`style.k`, jamais
  du canevas), il est pivote de `angle`, et il est rempli de la couleur passee.
- `[ ]` B1.2 `layers/offscreenChevron.ts` : le dessin. Gabarit normalise pointant +X :
  pointe `(1, 0)`, arriere-gauche `(-0,75, -0,7)`, encoche `(-0,35, 0)`, arriere-droite
  `(-0,75, 0,7)` — la base concave du gabarit fourni par l'utilisateur.
- `[ ]` B1.3 L'etiquette « nom · distance » : posee du COTE INTERIEUR de la fleche (sinon
  elle sort de la toile), encre de lisibilite depuis `canvasInk`, jamais un litteral.

**Gate B1** : `make test-web` · `make check-types`.

### Phase B2 — cablage des marqueurs joueurs

- `[ ]` B2.1 **TEST ROUGE** `replayMarkers.test.ts` : un joueur vivant hors fenetre dessine
  la fleche a la marge, et NE dessine ni trainee, ni cone de visee, ni anneau d'apparition,
  ni marqueur d'etage.
- `[ ]` B2.2 `drawLivingTrack` : branche hors cadre.
- `[ ]` B2.3 **TEST ROUGE** croix de mort hors fenetre : la croix est plaquee a la marge,
  meme fondu, SANS nom ni distance (D6).
- `[ ]` B2.4 `drawDeathMark` : branche hors cadre.

**Gate B2** : `make test-web` · `ReplayTeams.perf.test.tsx` sans regression.

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
