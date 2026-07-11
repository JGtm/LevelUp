# PLAN — Refonte engagement : ancre lobby + attendu conditionné par l'intensité

> Rédigé le 2026-07-07. Origine : diagnostic du match `bc918a5a-ed48-4ba6-8c0c-a0a117cd9461`
> (courbes « Équipe réelle » / « Joueur attendu » visuellement confondues pour JGtm,
> coef_team_share = 1.005) + décisions produit prises en session avec l'utilisateur.
> Exécution : agent (Opus) sous contrat du skill `plan-execution` (ordre strict, une
> étape à la fois, gate par étape, statuts `[x]`/`[~]`/`[!]`, zéro fix hors périmètre).
> Reprise de session : lire ce fichier + la section AVANCEMENT en bas, reprendre à la
> première case non statuée de la phase la plus basse non close.

## 1. Contexte et diagnostic

Le modèle actuel (PLAN_ENGAGEMENT_IMPLEMENTATION, réflexion §6) :

- `pace_attendu(t) = coef_team_share × pace_team(t)` en mode équipe (lobby seulement en
  fallback FFA/1v1). `coef_team_share` = médiane glissante de `pace_joueur/pace_team`
  sur 200 matchs (`temporal.ComputeEngagementCoefficient`).
- `pace_team` inclut le joueur (1/N de la courbe) — choix de cohérence num/dénom
  (thought_log 2026-06-18).
- Poids des events (`engagement_weights.go`, validés 2026-06-26) : mode 1.5, assist 0.5,
  death 0.4, défaut 1.0.

Problèmes constatés / décisions produit (session 2026-07-07) :

1. **Sémantique voulue** : l'engagement mesure la **réponse** du joueur à l'activité
   totale du match (tout le lobby), pas sa part relative à sa seule équipe. Avec la
   référence équipe, le cas « l'adversaire nous met la misère et toute l'équipe (moi
   inclus) répond mal » est invisible : l'attendu s'effondre avec l'équipe.
2. **Attendu conditionné par l'intensité** : l'attendu n'est PAS proportionnel
   (`coef global × intensité`). C'est « ce que ce joueur fait D'HABITUDE dans un match
   d'intensité similaire ». Un joueur qui réagit mal aux matchs intenses doit avoir un
   attendu BAS dans un match intense — le trait structurel se lit sur la courbe attendu
   (basse vs équipe réelle), la forme du jour se lit sur l'écart réel−attendu.
3. **Double comptage kill/mort** : un frag d'un côté = une mort de l'autre. Le compter
   deux fois (1.0 + 0.4) n'apporte rien, et au niveau individuel le poids mort fait
   « répondre » un joueur qui se fait farmer. Poids mort → 0. Chaque affrontement
   compte une fois, côté acteur.
4. **3 courbes maximum** sur le graphe match : Équipe réelle (contexte), Joueur attendu,
   Joueur réel. PAS de courbe « Équipe attendue » (écartée : 4 courbes = trop, et elle
   exigerait une « habitude d'équipe » mal définie avec des coéquipiers inconnus).
   Le pace lobby (l'ancre) est exposé dans le TOOLTIP, pas dessiné.

Vérifié sur pièces (2026-07-07) :

- Enrichment persisté (`persist/rows.go` l.36-43) : `engagement_score`,
  `engagement_score_brut`, `engagement_score_confidence`, `engagement_pace_player`,
  `engagement_pace_team`, `engagement_pace_lobby`, `engagement_player_activity` —
  lus via `player_match_enrichment_latest` (append-only ADR 0026).
- `match_intensity` n'est PAS persisté ; la clé d'intensité disponible par match est
  `engagement_pace_lobby` (moyenne pondérée de la courbe, per-player). C'est elle qu'on
  utilise (cohérente avec l'univers de pondération, invariante à la taille du lobby).
- Coefficients : table `engagement_coefficients` PK (xuid, mode_category), écriture
  ART-safe SELECT-then-UPDATE-or-INSERT (`sync/engagement_recompute.go` l.205-233).
- Chemin de recompute force existant : `batchComputeEngagementScores(ctx, playerDB,
  sharedDB, xuid, force)` appelé avec force par `engine_backfills.go` l.93 et
  `enrichment_backfill.go` l.122 → l'outillage de re-backfill existe.
- Serving : `GetMatchEngagement` et `GetTimeseries`/`computeMatchSummary` recomputent la
  courbe LIVE depuis les events ; les valeurs persistées ne servent qu'aux échantillons
  de coefficients (`loadRatioSamples`) et à l'historique percentile (`LoadPlayerHistory`
  sur `engagement_score_brut`).
- Squad : `TeamExpected` = moyenne du `PaceAttendu` du joueur principal
  (`engagement_squad_service.go` l.99) → hérite automatiquement du nouveau modèle,
  aucun changement structurel côté squad.
- Équipe d'inconnus : les events de TOUS les participants humains sont ingérés dans
  shared (`highlight_events` — vérifié sur le match témoin : 8/8 humains ont 23-84
  events chacun). « Équipe réelle » ne dépend d'aucun historique des coéquipiers.
  Bots exclus partout (`SQLIsNotBotCol`) et n'émettent pas d'events. Seul bruit connu :
  un quitter reste compté au dénominateur NTeam après son départ (accepté, inchangé).

## 2. Objectif et critères de succès

**Objectif** : l'attendu du joueur devient « sa réponse habituelle à un match d'intensité
similaire », ancré sur le lobby ; chaque affrontement compte une fois ; le graphe match
reste à 3 courbes lisibles.

**Critères de succès** :

- Sur le match témoin `bc918a5a-…` vu JGtm : « Joueur attendu » n'est plus confondu avec
  « Équipe réelle » (les deux courbes ont des définitions indépendantes).
- Un match intense pour un joueur qui répond habituellement mal aux matchs intenses
  produit un attendu bas (bin chaotique → coef du bin), pas un attendu gonflé.
- `pace_joueur` d'un joueur farmé (0 kill, N morts) tend vers 0 (les morts ne génèrent
  plus de pace).
- Percentile/score : redéfini proprement APRÈS re-backfill (les résidus historiques sont
  recalculés dans le même univers — pas de mélange ancien/nouveau modèle).
- Gates verts : `go test ./...`, `go test -tags=integration -p 1 ./...` (sync/persist
  touchés — OBLIGATOIRE), `go vet`, typecheck front (cache purgé), vitest, lint i18n.

**Branche cible** : `feat/engagement-lobby-response` (nouvelle, depuis `main` une fois la
campagne audits mergée — sinon depuis `main` courant ; 1 tâche = 1 branche, N commits).

**Effort estimé** : lourd (modèle + migration + re-backfill 2 titres + front). ~6 phases.

## 3. Modèle cible (formalisé)

Pour un match du joueur X, mode_category m (PvP_ranked / PvP_unranked) :

```
pace_joueur(t)  = Σ poids(events X)  dans fenêtre 90s / 1.5 min          (inchangé sauf poids)
pace_team(t)    = Σ poids(events équipe incl. X) / NTeam / 1.5 min       (inchangé sauf poids)
pace_lobby(t)   = Σ poids(events lobby humains) / NHumansLobby / 1.5 min (inchangé sauf poids)

I               = mean_t pace_lobby(t)                 # intensité du match
b               = bin(I)  ∈ {calme, standard, chaotique}   # bornes du joueur (terciles)
pace_attendu(t) = coef[m][b] × pace_lobby(t)
résidu brut     = mean_t (pace_joueur(t) − pace_attendu(t))
score           = percentile(résidu, historique 200 matchs même mode)     (inchangé)
```

Coefficients par bin (nouveau, remplace l'usage de coef_team_share dans l'attendu) :

```
Pour chaque (xuid, m) sur la fenêtre 200 matchs (player_match_enrichment_latest) :
  - clé d'intensité par match : engagement_pace_lobby (recalculé avec les nouveaux poids)
  - bornes de bins : terciles de la distribution des intensités du joueur (adaptatif
    par joueur et par mode) — persistées avec les coefs
  - coef[m][b] = médiane(engagement_pace_player / engagement_pace_lobby) des matchs du
    bin b, filtres inchangés (activity ≥ 3, pace_lobby ≥ seuil), clamp [0.1, 5.0]
```

Chaîne de fallback de l'attendu (persistée dans la réponse API — champ `expected_basis`) :

1. `bin` : ≥ 10 échantillons valides dans le bin du match → coef du bin.
2. `global` : sinon, coef lobby global (l'actuel `coef_lobby_share`, ≥ 10 échantillons).
3. `cold_start` : sinon coef = 1.0 ET la série « Joueur attendu » est MASQUÉE côté front
   (le masquage est désormais indexé sur `expected_basis`, plus sur `confidence` — cela
   corrige au passage le défaut relevé le 2026-07-07 : masquage indexé sur le mauvais
   signal, cf. thought_log).

Poids des events (`engagement_weights.go`) :

| type | avant | après |
|---|---|---|
| mode (objectif) | 1.5 | 1.5 |
| assist | 0.5 | 0.5 (action menée, pas un double comptage du frag) |
| death | 0.4 | **0.0** |
| défaut (kill, medal, …) | 1.0 | 1.0 |

Seuils de filtre des échantillons (`engagement_coefficients.go`) : la suppression du
poids mort baisse mécaniquement tous les paces (~25 % sur un mix kills≈morts).
`PaceTeamMinThreshold` et `PaceLobbyMinThreshold` passent de 1.0 → **0.75**. Gate
empirique en Phase 4 : sur les 4 joueurs Infinite, le taux de rejets d'échantillons
(hors AFK activity<3) doit rester < 5 % — sinon ajuster à 0.6 et documenter.

Décisions produit TRANCHÉES (ne pas rouvrir en cours d'exécution) :

- D1 : ancre = lobby pour l'attendu, partout (unifie équipe/FFA — la branche
  `selectExpectedReference` disparaît, `IsTeamMode` ne sert plus qu'à l'affichage de la
  courbe équipe).
- D2 : attendu conditionné par bins d'intensité (terciles adaptatifs par joueur+mode),
  libellés `calme` / `standard` / `chaotique` (vocabulaire existant de
  `computeMatchIntensity`).
- D3 : poids mort = 0. Marqueurs morts passives, bandes post-mort, filtre AFK
  (K+A+D) : INCHANGÉS (indépendants des poids).
- D4 : graphe match = 3 courbes (Équipe réelle fine, Joueur attendu pointillé, Joueur
  réel épais). Lobby dans le tooltip uniquement. Pas de courbe « Équipe attendue ».
- D5 : `coef_team_share` n'alimente plus rien → retiré du payload `engagement_profile`,
  du recompute et des textes d'aide. La COLONNE DuckDB reste en place (inerte,
  commentée) — pas de DROP COLUMN sur les player DBs. La table `engagement_coefficients`
  continue de porter le coef lobby global (fallback `global`).
- D6 : nouvelle table player DB `engagement_response_bins` (voir Phase 2) — table neuve
  AVEC PK à la création (évite le piège CREATE TABLE IF NOT EXISTS + PK).
- D7 : re-backfill des DEUX titres (le compute est title-agnostic, H5 events synthétisés
  via killer_victim_pairs — inchangé).
- D8 : sémantique du champ `confidence` (full/partial/insufficient_history) inchangée —
  il qualifie l'historique du PERCENTILE. `expected_basis` qualifie l'ATTENDU. Deux
  signaux distincts dans le payload.

## 4. Phases

### Phase 1 — Poids des events (petite, isolée, livrable seule)

Périmètre fermé :

- [x] `engagement_weights.go` : death 0.4 → 0.0 ; mettre à jour le commentaire de tête
      (recensement, date, décision user 2026-07-07) — pas de doc inversée.
- [x] `engagement_weights_test.go` : adapter les attentes (somme fenêtre 0-1000 = 3.0).
- [x] `engagement_coefficients.go` : seuils 1.0 → 0.75 (constantes commentées avec la
      justification ~-25 % de pace). Commentaires `=1.0` du test coefficients corrigés en
      `=0.75` (valeurs 0.4/0.5 restent sous seuil → comportement inchangé).
- [x] Vérifier que `PassiveDeathThresholdMS` / `annotateDeaths` ne dépendent PAS du poids
      (lecture seule des event types) — confirmé : `annotateDeaths`/`isPassiveDeath` ne
      lisent que `e.EventType` ; tests `PassiveDeathDetected` / `ActiveDeathNotMarkedPassive`
      verts.
- [x] Tests unitaires `temporal` verts.

Gate : `cd apps/go-api && go test ./internal/analysis/temporal/... && go vet ./...` = 0.
→ PASSÉ (2026-07-11) : temporal ok ; go vet exit 0.

### Phase 2 — Bins de réponse + attendu lobby (cœur du modèle)

Périmètre fermé — couche analysis (pur, zéro DB) :

- [ ] `temporal/engagement_response_bins.go` (nouveau) :
      `ComputeEngagementResponseBins(samples []RatioSample) (*ResponseBinsResult, error)`
      — terciles sur PaceLobby, médiane par bin, filtres, clamp, min 10/bin,
      `ErrInsufficientBinHistory` par bin (le résultat porte les bins calculables).
- [ ] `temporal/engagement_score.go` : `EngagementScoreInput` gagne
      `ResponseBins *domain.EngagementResponseBins` ; suppression de
      `selectExpectedReference` (D1) ; l'attendu = `resolveExpectedCoef(meanLobby)` ×
      pace_lobby(t). ATTENTION ordre de calcul : la courbe des paces se construit
      d'abord, l'intensité moyenne du match ensuite, l'attendu en 2e passe sur la courbe.
- [ ] Le résultat (`domain.EngagementScoreResult`) gagne `ExpectedBasis string`
      (`bin`/`global`/`cold_start`) + `IntensityBin string` (vide si non-bin).
- [ ] Tests unitaires : joueur « répond mal aux matchs intenses » (coef bin chaotique <
      coef bin calme) → attendu bas dans un match chaud ; fallback global ; cold_start ;
      FFA (NTeam=1) identique au mode équipe côté attendu.

Couche domain/port/platform :

- [ ] `domain/engagement_score.go` : types `EngagementResponseBins` (bornes + coef + n
      par bin) et champs résultat ci-dessus.
- [ ] `port/engagement_score.go` : `LoadResponseBins(ctx, xuid, modeCategory)` ;
      `SaveResponseBins` côté sync.
- [ ] Migration player DB (2 titres — enregistrer dans l'ordre des deux registres de
      migrations) : `CREATE TABLE engagement_response_bins (xuid VARCHAR, mode_category
      VARCHAR, intensity_bin VARCHAR, lower_bound DOUBLE, upper_bound DOUBLE,
      coef_lobby DOUBLE, n_matches INTEGER, last_updated TIMESTAMP,
      PRIMARY KEY (xuid, mode_category, intensity_bin))` — table neuve avec PK (D6).
- [ ] `platform/duckdb/engagement_score_repo*.go` : loader + intégration au
      `LoadEngagementCoefficient` existant (un seul round-trip si possible).
- [ ] Écriture sync : `engagement_recompute.go` — après le recompute des coefs globaux,
      recompute des bins (mêmes samples, ART-safe SELECT-then-UPDATE-or-INSERT,
      sérialisé sous lease KindPlayer comme `saveCoefficient`).
- [ ] `service/engagement_player_service.go` : `buildInputForMatch` charge les bins
      (best-effort, dégradation cold_start si table absente) ; `loadCoefsSafe` devient
      `loadExpectedInputs`.
- [ ] Tests : repo (DuckDB :memory:), service (mock port), recompute (fixture existante
      `engagement_recompute_test.go` étendue).

Gate : `go test ./internal/... ` vert + `go vet` 0. AUCUN test d'intégration encore requis
(pas de write path partagé touché) mais lancer `-tags=integration -p 1` sur
`./internal/sync/...` par sécurité (recompute touché).

### Phase 3 — Contrat API

Périmètre fermé :

- [ ] `EngagementScoreResult` JSON : + `expected_basis`, + `intensity_bin`. Handler
      inchangé (aucune logique métier dedans).
- [ ] `GET /engagement_profile` : payload évolue — retire `coef_team_share` (D5),
      ajoute les bins (bornes + coef + n par bin et par mode). Adapter
      `domain.EngagementCoefficient` ou introduire `EngagementProfile` dédié (préférer
      le type dédié, l'existant est aussi utilisé comme porteur xuid/gamertag côté
      squad — NE PAS le casser, cf. `engagement_squad_service.go`).
- [ ] openapi : régénérer (`make generate-types`) ; garde-fou
      `TestNoJSONRouteBypassesHuma` vert.
- [ ] Tests handlers (`engagement_test.go`) adaptés.

Gate : `go test ./internal/api/... ./internal/service/...` + `make generate-types` sans
diff inattendu hors fichiers générés.

### Phase 4 — Re-backfill 2 titres (séquence en DEUX passes — ordre critique)

Le résidu persisté dépend des coefs, les coefs dépendent des paces persistées. Séquence :

- [ ] Passe A : recompute force des enrichments engagement (nouveaux poids → nouvelles
      paces ; résidus provisoirement calculés avec les coefs périmés) — chemin force
      existant (`enrichment_backfill.go` l.122 / `engine_backfills.go` l.93) via la CLI
      `levelup` (identifier la commande exacte : `go run ./apps/go-api/cmd/levelup --help`,
      c'est un backfill enrichment par joueur ; la documenter ici à l'exécution).
- [ ] Recompute coefficients + bins pour tous les joueurs des 2 titres
      (`POST /engagement/recompute_coefficients` par joueur, ou l'équivalent CLI).
- [ ] Passe B : recompute force à nouveau (paces identiques, résidus/scores recalculés
      avec les coefs/bins définitifs).
- [ ] Vérification chiffrée (gate) : sur Madina/JGtm/Chocoboflor Infinite —
      (a) `engagement_response_bins` porte 3 bins × modes éligibles avec n ≥ 10 ;
      (b) taux de rejets d'échantillons hors-AFK < 5 % (sinon seuil 0.6, documenter) ;
      (c) sur le match témoin `bc918a5a-…` vu JGtm : `pace_attendu ≠ pace_team`
      (définitions désormais indépendantes) et `expected_basis` ∈ {bin, global} ;
      (d) aucun coef hors [0.1, 5.0].
- [ ] Écritures via le chemin existant (BatchBuilder/Persister append-only) — aucun
      UPSERT nouveau sur table partagée. Les tests anti-ART restent verts.

Gate : `go test -tags=integration -p 1 ./...` (OBLIGATOIRE ici — sync/persist touchés),
code de sortie vérifié, filtre `^--- FAIL:` ancré.

Note prod : le re-backfill prod se rejoue APRÈS merge (push main = deploy auto — prévenir
l'utilisateur avant). Étape listée mais exécution différée au runbook de merge.

### Phase 5 — Front

Périmètre fermé :

- [ ] `EngagementCurve.tsx` : la série « Joueur attendu » se masque sur
      `expected_basis === 'cold_start'` (le prop `hideAttendu` est rekeyé côté
      `EngagementMatchSection` — le commentaire actuel décrivant confidence devient
      FAUX, le mettre à jour) ; tooltip : ajouter la ligne « Lobby » (paceLobby déjà
      dans les points).
- [ ] Sous-titre du graphe : afficher la base de l'attendu — ex. FR « attendu : ta
      réponse habituelle aux matchs {calmes|standards|chaotiques} », EN équivalent.
      Clés dans `manifests/engagement.toml` (FR + EN, parité typée).
- [ ] `types.ts` / `generated.ts` : régénérés (`make generate-types`).
- [ ] Textes d'aide (`features/help/i18n.ts` l.140-155) : réécrire les 3 blocs qui
      décrivent le modèle team-share (formule, exemple chiffré, coefficients) pour le
      modèle lobby+bins — FR ET EN.
- [ ] Page profil engagement (consommateur de `engagement_profile`) : adapter à
      `EngagementProfile` (retrait coef_team_share, affichage des bins).
- [ ] `EngagementCurve.test.tsx` + tests sections adaptés.
- [ ] Aucun hex/classe couleur ; tokens uniquement ; aucune string UI hors manifests.

Gate : purge `node_modules\.tmp` puis `npm run typecheck && npm run lint && npm run test`
= 0 (vitest hors sandbox, cf. reference_vitest_outside_sandbox).

### Phase 6 — Nettoyage, doc, clôture

Périmètre fermé :

- [ ] Retirer `coef_team_share` du recompute (`ComputeEngagementCoefficient` ne calcule
      plus que le ratio lobby ; renommer si besoin) + du payload — la colonne DB reste,
      commentée dans la migration d'origine (D5). Supprimer le code mort qui en découle
      (règle 0 code mort — y compris tests devenus sans objet).
- [ ] `selectExpectedReference` : supprimé avec ses tests (D1).
- [ ] `.ai/REFLEXION_ENGAGEMENT_SCORE_INTRA_MATCH.md` : addendum daté « modèle v2
      lobby-anchored » (ne pas réécrire l'historique, ajouter une section).
- [ ] `docs/` : si le guide FOUNDATIONS/features mentionne le modèle team-share, MAJ
      FR+EN dans le même commit.
- [ ] Vérifier qu'aucun garde-rail n'a été affaibli (allowlists no_art_patterns,
      baselines lint) sans justification datée.
- [ ] Entrée thought_log de clôture (décision, chiffres des gates, prochaine étape =
      re-backfill prod post-merge).
- [ ] Gate global final : suite complète Go + intégration `-p 1` + front + lint.

## 5. Hors périmètre (NE PAS TRAITER — consigner en Découvertes)

- Activation engagement H5 côté UI (chantier calibration séparé, cf. mémoire
  project_h5_engagement_canonicalization_chantier).
- Toute retouche du binning serveur Timeseries, du coach, ou de LUSR.
- Le bruit « quitter compté au dénominateur NTeam » (accepté en l'état).
- Pondération temps-joué LUSR, weapon_kills, etc.

## 6. Découvertes (à remplir en cours d'exécution)

- (vide)

## 7. AVANCEMENT (à tenir à jour par l'exécutant)

- Phase 1 : COMPLÉTÉE (2026-07-11) — death→0.0, seuils→0.75 ; temporal ok, vet 0.
- Phase 2 : non commencée
- Phase 3 : non commencée
- Phase 4 : non commencée
- Phase 5 : non commencée
- Phase 6 : non commencée
