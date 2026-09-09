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

- `[ ]` A1.1 **TEST ROUGE** `teammates_exact_composition_test.go` : sous
  `FilterExactComposition=true`, la reponse doit porter, pour chaque session, le compte
  AVANT filtre exclusif et la liste des matchs ecartes avec le(s) coequipier(s) connu(s)
  responsables, nommes. Echec attendu : les champs n'existent pas.
- `[ ]` A1.2 `internal/domain/teammates.go` : type `CompositionSessionEntry` (embarque
  `SessionLabelEntry`) + champs `match_count_roster` et `excluded_by_exact_composition` ;
  type `ExcludedMatch` (`match_id`, `start_time`, `map_ui`, `extra_gamertags`).
  `TeammatesPageResponse.CompositionSessions` change de type.
- `[ ]` A1.3 `teammates_service_intersect.go` : `filterExactComposition` rend
  `(kept, excluded []domain.SquadMatchRow)` — un seul balayage, jamais deux regles.
  `matchHasExactComposition` gagne un frere `extraPresentOn(team, extraPool)` qui NOMME les
  xuids fautifs (le predicat booleen reste, il est deja verrouille par ses tests).
- `[ ]` A1.4 `teammates_service.go` : construction de `CompositionSessions` depuis le
  couple (gardes, ecartes) — `match_count` reste le compte POST-filtre (source unique),
  `match_count_roster` est le compte PRE-filtre de la MEME session.
- `[ ]` A1.5 Resolution xuid -> gamertag des fautifs via `topRows` (deja charge) ; un xuid
  non resolu s'ecrit `Joueur <4 derniers>` (meme repli que `Q32b`), jamais vide.
- `[ ]` A1.6 `slog.InfoContext` du denominateur : sessions, gardes, ecartes, fautifs
  distincts. Aucune erreur avalee.
- `[ ]` A1.7 OpenAPI : `api/openapi_manual_fragment.yaml` + `make openapi-gen` ;
  `make generate-types` pour `apps/web/src/lib/api/generated.ts`.

**Gate A1** : le test A1.1 passe · `go test ./internal/...` · `go vet ./...` ·
`make go-api-lint` 0 nouveau · `make generate-types` sans diff residuel.

### Phase A2 — TDD frontend : source unique pour la L2

- `[ ]` A2.1 **TEST ROUGE** `apps/web/src/features/squad/squadSessionCounts.test.ts` : une
  fonction pure `squadSessionCount(label, compositionSessions, fallback)` rend
  `{ shown, total }` depuis `composition_sessions` SEUL des que la reponse teammates est la,
  et ne retombe sur `/filters/resolve` que pendant le chargement initial. Echec attendu : le
  module n'existe pas.
- `[ ]` A2.2 `squadSessionCounts.ts` : le module pur. `mergeSessionCounts` y est absorbe
  (une seule regle de compte cote web) et son ancien site d'appel migre — regle 6 CLAUDE.md
  (factorisation + garde-rail, jamais l'une sans l'autre).
- `[ ]` A2.3 **TEST ROUGE** `PeriodSessionRail.test.tsx` : quand un `sessionCount` est
  fourni, le rail affiche « 4 sur 7 » et NON le compte de son store.
- `[ ]` A2.4 `PeriodSessionRail.tsx` : props optionnelles `sessionCount?: (label) => {shown,
  total} | undefined` et `matchCount` alimente par la composition. Aucune page qui ne passe
  pas la prop ne change de rendu (les autres pages sont hors perimetre).
- `[ ]` A2.5 `SquadLayout.tsx` : le rail recoit le compte composition ; `totalAfter` cesse
  d'alimenter `matchCount`. `getSessionCount` du `SessionMultiSelect` passe par le meme
  module.
- `[ ]` A2.6 **RATCHET** `apps/web/src/features/squad/singleCountSource.guard.test.ts` :
  aucun fichier de `features/squad/` ne lit `total_matches_after_filters` ni
  `session_options` pour en tirer un compte de matchs. Allowlist vide, datee, avec le motif.

**Gate A2** : `make check-types` · `make test-web` (les 3 tests ci-dessus verts) ·
`npx eslint` sur les fichiers touches.

### Phase A3 — lisibilite de l'ecart (D1)

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
- Socles d'EQUIPEMENT non publies : la voie `ti=37` filtre sur le prefixe `powerup_`, donc
  grappin / repulseur / mur / capteur / ecran / propulseur / translocateur poses sur un
  socle de carte ne sont publies nulle part. Le catalogue de cartes connait pourtant 407
  points d'apparition `equipment`. Lot a part entiere.
- Etat actif des deployables (mur 19, capteur 22) : voie `charges-remaining` en reserve
  depuis la decision utilisateur du 2026-08-16.
