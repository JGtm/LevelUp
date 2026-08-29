# PLAN — Afficher les MANCHES gagnees/perdues sur les modes a manches

> Date : 2026-08-29 · Branche de travail : a creer depuis `feat/v75` · Statut : AUDIT FAIT,
> EN COURS (E0 clos le 2026-08-29 ; arbitrages §6 tous tranches).
>
> Demande utilisateur : « Lorsqu'un mode compte les manches remportees pour determiner la
> victoire ou la defaite, il faut mieux afficher les manches gagnees/perdues plutot que les
> points retournes par l'API » — au rejeu, ET partout ou un score s'affiche (vue match,
> tableaux de resultats).

---

## 1. VERDICT EN UNE PAGE

1. **La donnee n'existe pas chez nous.** `CoreStats.RoundsWon / RoundsLost / RoundsTied` est
   DECLARE dans la structure de payload (`internal/openspartan/halo_api_payload.go:114-116`)
   et **lu nulle part** : aucun mapper, aucune colonne `match_registry` ou
   `match_participants`, aucun champ canonical. Rien a afficher aujourd'hui sans une
   migration + une re-lecture des payloads.
2. **Le score affiche vient d'une source unique cote sync** (`ExtractTeamScoresByID`,
   `internal/sync/transforms_helpers.go:178`) mais **le LIBELLE est reconstruit en 5
   endroits** (§3.2) — violation de la regle 6 du CLAUDE.md (<= 2 copies). Toute regle
   « manches plutot que points » posee sans centraliser d'abord divergera en 5 exemplaires.
3. **Le rejeu sait deja ventiler les manches** (film, `roundsLogic.ts` + pastilles du
   bandeau) mais **n'ecrit jamais le compte de manches** ; pire, l'ecran de victoire et le
   panneau d'export video lisent `readScoreBanner` a la borne de fin, donc affichent en
   « score final » **les points de la DERNIERE MANCHE** (ex. Oddball 100-43) au lieu du
   resultat du match (2-1). C'est le defaut le plus net du lot.
4. **La detection « mode a manches » ne peut PAS etre une liste de modes.** Mesure du
   2026-08-24 (`.ai/V7.5/replay2d/RAPPORT_QUALITE_SCORE_EQUIPE.md` §1.1) : deux matchs KOTH
   du corpus portent `RoundsWon` 1/0 et 0/1 — ils se jouent en UNE manche, et leur `Score`
   3/0 est le score de colline affiche. Le meme nom de mode peut donc etre a manches ou non
   selon la variante/playlist. **La detection doit etre par match, sur la donnee.**
5. `regulation.toml [score_target]` **exclut deja explicitement** Oddball et KOTH « modes a
   MANCHES — le total du match deborde le plateau d'une manche » : le probleme est connu et
   documente cote rejeu, il n'a jamais ete traite cote produit.

---

## 2. PERIMETRE AUDITE

| Axe | Fichiers ouverts |
|---|---|
| Payload API | `internal/openspartan/halo_api_payload.go` |
| Sync / extraction | `internal/sync/transforms_helpers.go`, `transforms.go` |
| Schema | `internal/games/halo_infinite/migrations/steps_shared_core.go` |
| Backfill existant | `cmd/backfill-team-scores/main.go` |
| Config variante | `config/titles/halo_infinite/mappings/regulation.toml`, `internal/games/mappings/loader_regulation.go` |
| Producteurs de libelle | `service/match_view_builders_header.go`, `service/match_history_service_enrich.go`, `service/teammates/teammates_service_assets.go`, `analysis/home_locale.go`, `analysis/home_canonical_recent.go` |
| Domaine / contrat | `domain/{match_view,match_history,explorer,home,teammates,match_facts}.go`, `api/handlers/projections.go` |
| Rejeu | `analysis/replay/score_timeline.go`, `web/src/features/match-replay/{roundsLogic,scoreBannerLogic,ReplayScoreBanner,ReplayVictoryOverlay,exportOverlayPanels,ReplayRoundBreakOverlay}.*` |
| Front score | `web/src/features/match-view/MatchHeader.card.tsx`, `MatchScoreCurveChart.tsx`, `features/explorer/ExplorerMatchesTable.tsx`, `components/ui/match-card.tsx`, `features/session-detail/SessionMatchesTable.tsx` |
| Multi-titre | `internal/games/halo_5/capture.go`, `platform/duckdb/halo5/halo5_match_history_source.go`, `config/titles/*/mappings/capabilities.toml` |

Non audite (hors demande) : agregats de saison, LUSR/CSR, medailles.

---

## 3. CONSTATS DETAILLES

### 3.1 Couche donnee — CONSTAT A1 : la verite est jetee a la sync

- `StatsBundle.CoreStats` type bien `RoundsWon/Lost/Tied` (int), au niveau **equipe**
  (`Teams[].Stats.CoreStats`) **et** joueur. Aucun appelant : grep sur tout le depot ne
  ramene que la declaration.
- `match_registry` porte `team_0_score / team_1_score` (+ `team_0_ps_score / team_1_ps_score`)
  et **rien sur les manches**. `match_participants` non plus.
- `persistMatchRegistry` est un **INSERT NU** (pas d'`ON CONFLICT`, doctrine anti-ART) : un
  re-sync ne reecrit JAMAIS une ligne existante. Toute colonne nouvelle exige donc un
  **backfill dedie** — c'est exactement la lecon de `cmd/backfill-team-scores`.

### 3.2 CONSTAT A2 : 5 copies du meme libelle « X - Y »

| # | Producteur | Consommateurs |
|---|---|---|
| 1 | `service/match_view_builders_header.go:284` `buildScoreLabelFromMeta` | Vue match (en-tete) |
| 2 | `service/match_history_service_enrich.go:194-196` | Historique + **Explorateur** + Carriere (via `api/handlers/projections.go`) |
| 3 | `service/teammates/teammates_service_assets.go:265-267` | Page coequipiers |
| 4 | `analysis/home_locale.go:138` `buildHomeScoreLabel` | Accueil (chemin legacy) |
| 5 | `analysis/home_canonical_recent.go:222` `buildScoreLabelCanonical` | Accueil (chemin canonical) |

Trois variantes de format cohabitent deja (`"%d-%d"` vs `"%d - %d"`), et deux sources de
donnee (registre vs `canonical.Summary.Teams`). Front : `score_label` est une **chaine
opaque** — le client ne peut ni la localiser, ni savoir si elle parle de points ou de
manches.

### 3.3 CONSTAT A3 : le rejeu contredit le match sur sa surface la plus visible

- Le film ventille les manches (`objectiveevents.SeriesByRound`) ; `roundsLogic.ts` en tire
  le nombre de manches jouees, le vainqueur de chacune, les bascules — **deja teste**.
- `readScoreBanner` affiche, sur un mode a manches, les points de la **manche courante**
  (choix assume, bon pour la lecture en direct) + une rangee de pastilles.
- **Defaut** : `ReplayVictoryOverlay` et `exportOverlayPanels` appellent `readScoreBanner`
  **a `playWindow.endFrame`** pour composer le « score final ». Sur un mode a manches, cela
  rend les points de la derniere manche (100-43), presente comme le score du match. Le
  compte de manches n'est ecrit nulle part en clair.
- `MatchScoreCurveChart` trace le **cumul** du match (Oddball 200-121) sans separation de
  manches : la courbe ne dit pas le resultat non plus.
- Couverture : le rejeu depend d'un artefact de film (rare). La voie API couvre **100 %** des
  matchs, y compris sans film.

### 3.4 CONSTAT A4 : detection par mode = faux positifs garantis — AMENDE PAR E0

Le nom du mode ne suffit pas (KOTH une manche dans le corpus, cf. §1.4). La regle candidate
etait `rounds_total >= 2`. **E0 l'a REFUTEE** : le CTF d'arene se joue en deux MI-TEMPS
(`rounds_total = 2`) alors que son score est le total de captures — la regle y aurait affiche
« 0 - 1 » a la place de « 2 - 3 ».

Regle retenue apres mesure (rapport §5), trois conditions cumulatives :

```
afficher les manches  <=>  game_variant_name declaree dans regulation.toml [rounds_decide]
                           ET rounds_total (MAX des deux camps) >= 2
                           ET rounds_won(A) != rounds_won(B)
```

Table initiale MESUREE : `Arena:Oddball`, `Ranked:Oddball`, `Oddball:Arena` — les seules
variantes ou l'affichage actuel MENT (4 matchs, vainqueur avec moins de points). Tout le reste
garde les points, sans regression.

### 3.5 CONSTAT A5 : multi-titre

Halo 5 alimente `team_0_score/team_1_score` par `carnageTeamScores`
(`internal/games/halo_5/capture.go:354`, `TeamStats[].Score`) et a lui aussi des modes a
manches (Breakout). La regle doit passer par une **capability** (ex. `match.rounds`) declaree
dans `capabilities.toml` + un champ canonical, **jamais** par `slug == "halo_infinite"`
(ratchet `no_slug_comparison_test.go`). Halo 5 peut rester `not_exposed` au depart :
degradation = comportement actuel (points).

---

## 4. CE QU'IL FAUT CONSTRUIRE — ARCHITECTURE CIBLE

```
API payload (Teams[].CoreStats.RoundsWon/Lost/Tied)
  -> sync.ExtractTeamRoundsByID          (voisin de ExtractTeamScoresByID, MEME fichier)
  -> match_registry.team_{0,1}_rounds_won / rounds_total   (colonnes additives)
  -> canonical.TeamSnapshot.RoundsWon (+ MatchSummary.RoundsTotal)
  -> analysis.TeamScoreDisplay(...)      SOURCE UNIQUE de la regle points|manches
  -> analysis.BuildTeamScoreLabel(...)   SOURCE UNIQUE du libelle (les 5 copies migrent)
  -> contrat API : score_label (compat) + score_kind ("points"|"rounds") + valeurs numeriques
  -> front : i18n FR/EN (« manches » / « rounds »), aucune regle cote client
  -> rejeu : le compte de manches vient du REGISTRE (API), le film reste le repli
```

Points non negociables :
- Ecritures : INSERT-only sur `match_registry`, aucune UPSERT (ADR 0019/0026/0030).
- Colonnes additives + `AddColumnIfMissing` (pas de `CREATE TABLE IF NOT EXISTS` retouche).
- Aucun libelle FR/EN en dur cote Go : le mot « manches » vit dans l'i18n web (et/ou
  `mappings/` si un libelle serveur s'avere necessaire).

---

## 5. PLAN D'EXECUTION (etapes ordonnees, gate a chaque fin)

> Contrat `plan-execution` : ordre strict, une etape a la fois, aucun report d'une etape
> executable, statut `[x]` / `[~]` / `[!]` sur chaque item a la cloture.

### E0 — MESURE (aucun code de prod) — BLOQUANT — **CLOS le 2026-08-29**

Livrable : `.ai/V7.5/RAPPORT_MANCHES_2026-08-29.md`.

- [x] Lister les matchs a score : 1 942 lignes (`diag_q`, registre a 1 948).
- [x] Re-fetch des payloads (`diag_matchstats_dump --rps 4`, 17 lots) : **1 942 / 1 942, 0 erreur**, ~10 min.
- [x] Extraire par match : variante, categorie, `Outcome`, `Score`, `RoundsWon/Lost/Tied` des deux camps.
- [x] Question (a) : 57 matchs multi-manches (2,9 %), 9 variantes, tableau au rapport §2.1.
- [x] Question (b) : invariant vrai sur **56/57** ; l'exception est une manche NULLE (§4.2).
- [x] Question (c) : **4 matchs Oddball** ou le VAINQUEUR affiche MOINS de points (§3).
- [x] Question (d) : `RoundsTied > 0` sur **18 matchs** — la manche nulle existe.
- [x] Livrable ecrit et date.
- **Gate PASSE** : la regle de §3.4 est **REFUTEE** et remplacee (cf. §3.4 amende + rapport §5).

**Ce que la mesure a change dans le plan** : detection declarative mesuree (et non deduite),
`rounds_total` pris comme `max` des deux camps, condition supplementaire « manches non a
egalite », perimetre initial reduit aux 3 variantes Oddball.

### E1 — Persistance (schema + sync) — **CLOS le 2026-08-29**

- [x] Migration additive `add_team_rounds_to_match_registry` (fichier dedie
      `steps_shared_team_rounds.go` — le god-file `steps_shared_core.go` est deja a 632 L,
      regle 5 : ne pas accroitre la dette) : `team_0_rounds_won`, `team_1_rounds_won`,
      `rounds_total` en SMALLINT via `AddColumnIfMissing`, + entree dans
      `migration.canonicalOrder`, + `steps.go`. Aucun backfill SQL : la valeur n'existe que
      dans le payload API (d'ou E2).
- [x] `sync.ExtractTeamRoundsByID` dans `transforms_helpers.go`, juste sous
      `ExtractTeamScoresByID` : indexe par `TeamId`, nil par camp absent, `rounds_total` =
      MAX des deux camps. **7 tests** dont 4 temoins du corpus E0 (Oddball 3 manches, Slayer
      a une manche, manche nulle, abandon 1 vs 0).
- [x] Cablage `ExtractRegistry` (`transforms.go`) + INSERT `persistMatchRegistry` + champs
      `domain.MatchRegistryRow`.
- [x] Trois garde-rails de schema mis a jour **parce qu'ils ont casse** (et c'est leur role) :
      `persist.MatchRegistryColumns` (auto-parite de l'INSERT), l'allowlist `persistOnly` du
      seeder demo (justification ecrite : le corpus demo n'a aucun mode a manches), et
      `sync.sharedSchemaSQL` (bootstrap des DB fraiches — 3 E2E echouaient a l'INSERT).
- [!] Halo 5 : **non cable, faute de donnee**. `H5CarnageTeam` (`games/halo_5/dto_carnage.go`)
      ne porte que `TeamId / Score / Rank` — l'API carnage ne publie AUCUN compteur de
      manches. Les colonnes restent NULL pour ce titre, degradation = points (le comportement
      actuel). A revoir si 343 expose la donnee ailleurs.
- **Gate PASSE** : `go test ./...` (143 paquets ok), `go test -tags=integration ./internal/persist ./internal/sync` **ok**.
      Deux echecs PRE-EXISTANTS, sans rapport avec ce lot, notes en §9 (non traites, regle 7).

### E2 — Backfill historique — **CLOS le 2026-08-29**

- [x] `cmd/backfill-team-rounds` calque sur `cmd/backfill-team-scores` : phase A sans droit
      d'ecriture (fetch + decisions), phase B courte sous `--apply`, re-lecture avant
      ecriture, jamais de NULL/negatif/hors-bornes, jamais un total inferieur aux manches
      gagnees. 4 fichiers, 20 tests (decision, jonction SQL, garde-rail ART local).
- [x] **Difference assumee avec son modele** : pas de fichier de liste. La population se
      definit exactement par `rounds_total IS NULL` — la base sait le dire. L'outil est donc
      reprenable apres interruption, et un second passage ne re-telecharge que le manquant.
      (Effet de bord heureux : aucune copie de `ids.go`, la regle des 2 copies tient.)
- [x] Repetition a blanc sur 20 matchs : `lus=20 planifiees=20 skippes=0 echecs=0`.
- [x] **Filtre par variante declaree, ajoute apres question de l'utilisateur** (« pourquoi
      1 942 matchs si seul Oddball est impacte ? » — question juste). La liste de travail est
      restreinte par defaut aux variantes de `[rounds_decide]`, lues dans le MEME
      `regulation.toml` que celui qui commande l'affichage : impossible de rattraper une
      population differente de celle qui en a besoin. **26 matchs au lieu de 1 942, 7 s au
      lieu de 10 min.** Les matchs FUTURS sont renseignes a la sync pour TOUTES les variantes ;
      le jour ou une variante est ajoutee, un second passage rattrape son historique.
      `--all` reste disponible. Variantes liees en arguments, jamais interpolees.
- [x] **`--apply` EXECUTE** (serveur arrete par l'utilisateur) : `lus=26 planifiees=26
      ecrits=26 skippes=0 echecs=0`.
- **Gate PASSE** : couverture verifiee en base — 26/26 lignes renseignees
      (`Arena:Oddball` 13, `Ranked:Oddball` 9, `Oddball:Arena` 4), dont **25 basculent en
      affichage manches** ; la 26e est le temoin `adb93fb7` (manches a egalite) qui reste en
      points, exactement ce que la regle prescrit.

### E3 — La regle, en UN seul endroit — **CLOS le 2026-08-29**

- [x] Section `[rounds_decide]` dans `config/titles/halo_infinite/mappings/regulation.toml`
      (`schema_version` 3) : les 3 variantes Oddball mesurees, avec au-dessus la liste
      EXPLICITE de ce qui est volontairement absent et pourquoi.
- [x] Loader : `RegulationSet.roundsDecide` + accesseur `RoundsDecide(variante) bool`,
      nil-safe, cle trimee. **Une entree `false` est REFUSEE** — l'absence de cle est deja
      le « non », deux facons de dire non finiraient par se contredire.
- [x] `internal/analysis/team_score_display.go` — `ReadTeamScore(TeamScoreInput)
      (TeamScoreDisplay, bool)`, PURE, title-agnostic, sans I/O. `Points` reste renseigne
      meme en lecture manches (le secondaire grise de la vue match, arbitrage 6.2).
- [x] Regle E0 aux trois conditions cumulatives ; tout echec revient aux points.
- [x] 11 cas table-driven, dont 4 temoins du corpus (Oddball 181-186 qui ment, CTF d'arene
      non declare, manches a egalite, colonnes NULL) + 3 tests loader dont un qui EPINGLE le
      contenu livre (Oddball declare, CTF d'arene non).

### E4 — Centralisation du libelle (dette regle 6) — **CLOS le 2026-08-29**

- [x] `analysis.TeamScoreLabel(TeamScoreInput) string` + `FormatTeamScoreLabel(display)` —
      les **5** producteurs de §3.2 delegent, plus aucun `fmt.Sprintf` de score ailleurs.
      Refactor PUR : les manches ne sont pas encore cablees jusqu'a ces appelants (elles
      arrivent en E5), donc `nil` partout et lecture en points, a l'identique.
- [x] Effet de bord : trois `import "fmt"` devenus inutiles retires (analysis/home_locale,
      service/teammates).
- [x] **Garde-rail** `team_score_label_guard_test.go` : scan de `internal/analysis` +
      `internal/service` (hors tests et hors fichier canonique) interdisant `%d-%d` et
      `%d - %d`, commentaires retires avant scan, + un test qui prouve que le garde-rail
      MORD. Perimetre volontairement limite a ces deux couches : un scan global mordrait sur
      les `cmd/diag_*` sans rien proteger de plus.
- [x] **CHANGEMENT VISIBLE ASSUME** : le format est unifie sur « X - Y » (espaces), celui
      que 3 des 5 appelants utilisaient deja. La vue match et l'accueil passent donc de
      « 50-30 » a « 50 - 30 ». 9 tests existants migres (pas supprimes) vers le nouveau
      format.
- **Gate PASSE** : `go test ./internal/analysis ./internal/service/... ./internal/api/...` ok,
  garde-rail vert.

### E5 — Contrat API — **CLOS le 2026-08-29**

- [x] `score_kind` (`"points"|"rounds"`) sur `MatchViewHeader`, `MatchHistoryRow` et
      `ExplorerMatchesRow`. `score_label` conserve, inchange pour les modes en points.
- [~] `my_score`/`enemy_score` numeriques : **remplaces par `score_points_label`** sur la
      seule surface qui en a besoin (la vue match, arbitrage 6.2 — score API en petit et
      grise a cote). Un libellé pret a afficher au lieu de quatre nombres que le client
      devrait reformater ; le format reste produit a UN endroit.
- [x] Cablage complet de la donnee jusqu'au contrat :
      - vue match : 3 colonnes de plus a `Q13MatchMeta`, `MatchMetaRaw`, et un
        `applyMatchHeaderScore` pose APRES le builder (comme le flag « Prolongation » —
        la table `[rounds_decide]` est portee par le service, pas par le builder) ;
      - historique/Explorateur/carriere : 3 colonnes de plus a `Q5SharedHistory`,
        `teamScorePair` porte desormais points ET manches (elles se permutent ENSEMBLE
        selon `team_id` — les dissocier afficherait les manches d'un camp a cote des
        points de l'autre) ;
      - wiring : `WithRoundsDecide` sur les deux services, table PAR TITRE
        (`roundsDecideFor(pdb)`), lookup par cle de map, jamais de comparaison de slug.
- [x] **PIEGE RENCONTRE, ET C'EST UN VRAI** : `v_match_full` est un `SELECT mr.*` — une vue
      DuckDB FIGE son etoile a la creation. Les colonnes ajoutees en E1 n'y apparaissaient
      donc pas, et l'historique serait tombe en « Binder Error » en prod. D'ou un SECOND
      step de migration (`refresh_views_after_team_rounds`) qui recree la vue. Il est separe
      du premier a dessein : le step d'E1 est deja applique sur les bases de dev, or une
      migration enregistree ne rejoue jamais.
- [x] `openapi-gen` + `generate-types` : +8 lignes au contrat, +4 au `generated.ts`.
- **Gate** : `go test` vert sur `platform/duckdb`, `api/...`, `analysis`, `domain/...`,
  `games/...`. `internal/service` **NON GATE** : une autre session refactore
  `buildViewerFragDistribution` dans ce meme worktree et son fichier de test ne compile pas
  encore. Mon code du paquet compile (`go build ./internal/service/` vert) ; a re-gater des
  que leur refactor est pose.

### E6 — Front (surfaces produit) — **CLOS le 2026-08-29**

- [x] i18n FR/EN : `scoreRoundsHint` + `scorePointsAside` (match-view), et l'infobulle
      d'en-tete de colonne de l'Explorateur enrichie dans le manifeste TOML (regeneration
      des 21 manifestes, 2 997 cles).
- [x] Vue match : `ScoreRoundsAside` — (i) infobulle + « 181 - 186 points » en petit et
      grise, **uniquement** quand `score_kind === 'rounds'`. Sur un mode en points le
      libelle principal EST deja le score de l'API : l'ecrire deux fois n'apprendrait rien.
      Aucune couleur en dur (`text-muted-foreground`).
- [x] Explorateur/Historique/Carriere : la colonne « Score » portait DEJA une infobulle
      d'en-tete (`col_score_tooltip`) — elle a ete etendue plutot que doublee. La valeur
      reste nue (« 2 - 1 »), conformement a l'arbitrage 6.2.
- [x] Le client ne recalcule RIEN : il lit `score_kind`. La regle vit cote Go.
- [!] Accueil (`match-card`) et coequipiers : **non cables**. Ces deux surfaces lisent des
      lignes (`legacymatch.HomeMatchRow` canonique via `canonical.TeamSnapshot`, et
      `domain.SquadMatchRow`) qui ne portent pas les manches ; les brancher demande 3
      chemins SQL + le champ canonical + un parametre de config au constructeur pur de
      l'accueil. Elles affichent donc encore les points sur un Oddball. Ecart assume et
      SIGNALE : l'utilisateur a nomme le rejeu, la vue match et les tableaux de resultats.
      A trancher en fin de chantier.
- [x] Tests vitest : 5 cas sur la vue match (manches, points, champ absent, EN, absence
      d'aide en points). Suites match-view + explorer : **53 fichiers, 469 tests, 0 echec**.
- **Gate PASSE** : `npm run typecheck` (tsc -b) vert, ESLint vert sur les fichiers touches.

### E7 — Rejeu et export video

- [ ] Passer le compte de manches du registre a l'artefact (`replaybuild/matchfacts.go` ->
      `ScoreInput`), le film restant le repli quand le registre est muet.
- [ ] `ReplayVictoryOverlay` + `exportOverlayPanels` : sur un mode a manches, le « score
      final » devient le compte de manches (2-1), pas les points de la derniere manche.
- [ ] Bandeau (arbitrage §6.1) : points de manche + pastilles CONSERVES, et le compte de
      manches ecrit en clair a cote de l'indicateur de manche.
- [ ] Tests : le temoin Oddball du corpus doit rendre 2-1 en fin de match.

### E8 — Cloture

- [ ] ADR `docs/adr/0032-score-affiche-par-manches.md` (EN-only) : la regle, sa source, sa
      degradation.
- [ ] Skill `db-schema` : nouvelles colonnes. `docs/COMMANDS.md` : le backfill.
- [ ] Entree `.ai/thought_log.md` (obligatoire) + `delivery-checklist` + `make gate-push`.
- [ ] `adversarial-review` du diff (lot persist/sync a risque).

---

## 6. ARBITRAGES — TRANCHES PAR L'UTILISATEUR LE 2026-08-29

### 6.1 Bandeau du rejeu — TRANCHE : points de manche + compte de manches
Le bandeau garde les points de la MANCHE COURANTE (verite du direct) et ses pastilles ; on
ajoute le compte de manches en clair. **Seuls** l'ecran de victoire et le panneau d'export
video basculent sur le compte de manches comme score du match.

### 6.2 Format du libelle — TRANCHE : « 2 - 1 » nu + une aide explicite
- **Tableaux** (Explorateur / historique / carriere / accueil / coequipiers) : la valeur reste
  « 2 - 1 », et un **(i) avec infobulle dans l'EN-TETE de colonne** explique que la colonne
  affiche les manches gagnees / perdues quand le mode se joue en manches.
- **Vue match** : meme (i) + infobulle a cote du score, ET le **score API affiche en petit,
  grise, a cote** du compte de manches.
- Consequence de contrat : le front a besoin, en plus de `score_kind`, des **deux** valeurs —
  manches ET points — pour la vue match (cf. E5, a amender : `points_label` ou
  `my_points`/`enemy_points`).

### 6.3 `MatchScoreCurveChart` — TRANCHE : ON N'Y TOUCHE PAS
Decision utilisateur du 2026-08-29, apres reformulation : la carte reste en l'etat, hors
perimetre de ce chantier. Rappel de ce dont il s'agit :
De quoi il s'agit : la carte **« Score dans le temps »** de la vue match
(`web/src/features/match-view/MatchScoreCurveChart.tsx`), qui ne s'affiche **que si le match
a un artefact de rejeu**. Elle trace le score du mode image par image, decode du film. Sur un
mode a manches elle trace le **cumul du match** (Oddball : la courbe monte jusqu'a 200-121)
sans montrer ou une manche finit et ou la suivante commence — donc elle ne dit pas non plus
« 2-1 ». **Aucune modification** de cette carte dans ce chantier ; aucun item de backlog
n'est ouvert pour elle (decision explicite, pas un oubli).

---

## 7. RISQUES

| Risque | Parade |
|---|---|
| E0 refute la regle `rounds_total >= 2` | Le gate E0 est bloquant : rien n'est code avant. |
| Backfill incomplet (matchs supprimes cote API) | Degradation `KindPoints` : jamais de trou d'affichage. |
| Divergence des 5 libelles pendant la migration | E4 avant E5/E6, garde-rail grep pose dans le meme commit. |
| Corruption ART sur `match_registry` | Colonnes additives + INSERT-only + `go test -tags=integration`. |
| Regression Halo 5 | Capability `not_exposed` par defaut -> comportement actuel intact. |

---

## 8. CE QUI N'EST PAS FAIT DANS CE CHANTIER

- Les agregats de saison / winrate : ils lisent `outcome`, pas le score — non concernes.
- Le score PERSONNEL (`personal_score`) : distinct, non touche.
- La detection de prolongation (`regulation_seconds`) : voisine mais independante.

---

## 9. DECOUVERTES (hors perimetre — notees, NON traitees)

- **2026-08-29 (E1)** — `go test ./...` sort DEUX echecs sans aucun rapport avec ce lot, deja
  presents avant : `internal/archlint TestNoLocalLongestRun` (balayage « plus longue serie »
  local dans `cmd/oddball-terrain/confront.go`, fichier non touche ici) et
  `internal/himap` qui depasse le timeout de 600 s. Notes, NON traites (regle 7).
- **2026-08-29 (E1)** — une AUTRE session travaille dans le meme worktree (chantier « kills
  hors arme a feu ») : `.ai/thought_log.md`, `apps/web/src/features/match-replay/*` et
  `apps/web/src/lib/fda.ts` portent ses modifications non commitees. Consequence pour ce
  chantier : ne stager QUE ses propres fichiers, et grouper l'entree thought_log a la fin
  pour ne pas emporter l'entree en cours de l'autre session.

---

## 10. JOURNAL D'EXECUTION

| Date | Etape | Statut | Note |
|---|---|---|---|
| 2026-08-29 | Plan | Ecrit | Audit fait, 3 arbitrages tranches (§6). |
| 2026-08-29 | E2b | **CLOS** | Filtre par variante declaree (question utilisateur) : 26 matchs au lieu de 1 942. `--apply` passe, 26/26 ecrits, 25 basculent en manches. |
| 2026-08-29 | E3 | **CLOS** | Table `[rounds_decide]` mesuree (3 variantes Oddball) + `analysis.ReadTeamScore`, regle unique aux 3 conditions cumulatives. Une entree TOML a `false` est refusee : l absence vaut deja non. |
| 2026-08-29 | E2 | **PARTIEL** | Outil `cmd/backfill-team-rounds` livre et repete a blanc (20/20). `--apply` en attente : le serveur de dev tient la base, son arret est une decision utilisateur. La suite du plan n est pas bloquee. |
| 2026-08-29 | E1 | **CLOS** | Colonnes + extraction + persist. 3 garde-rails de schema ont casse (auto-parite INSERT, seeder demo, bootstrap `sharedSchemaSQL`) et ont ete mis a jour. Halo 5 `[!]` : l'API carnage ne publie pas les manches. Entree thought_log groupee en fin de chantier (worktree partage avec une autre session). |
| 2026-08-29 | E0 | **CLOS** | 1 942/1 942 payloads, 0 erreur. Regle `rounds_total >= 2` REFUTEE ; detection declarative mesuree adoptee. 4 matchs Oddball prouvent le mensonge. Rapport `.ai/V7.5/RAPPORT_MANCHES_2026-08-29.md`. |
