# Engagement — Validation E2E manuelle

> Référence : `.ai/PLAN_ENGAGEMENT_IMPLEMENTATION.md` §11 + doc réflexion §11
> Phase 8 du plan engagement.

Ce document liste les **7 cas de validation E2E** à exécuter sur l'application
en mode dev avec un joueur réel **avant tout déploiement production**.

---

## Pré-requis

1. App dev tournante (`npm run dev` côté web + `go run ./cmd/levelup` côté api)
2. Joueur sync avec **≥ 30 matchs PvP** sur la catégorie cible
3. Migration Phase 2 appliquée automatiquement au premier démarrage API
4. Backfill engagement scores exécuté :
   - via UI Settings → tab Sync → cocher "Score d'engagement" → "Lancer le recalcul rétroactif"
   - OU via CLI : `go run ./cmd/levelup-cli backfill --player MonGT --engagement-scores`

## Outil de pré-validation rapide (Phase 0)

Avant de faire les 7 cas E2E, lancer la validation programmatique des
hypothèses critiques :

```bash
cd apps/go-api
go run ./cmd/engagement-validate --player MonGT
go run ./cmd/engagement-validate --player MonGT --mode-category PvP_unranked
```

Sortie attendue :
- **H1 PASS** si `corr(engagement, performance) < 0.5` (axes décorrélés)
- **H3 PASS** si ratio coef récent/ancien < 1.3 (style stable)
- VERDICT global "OK pour deploiement" si H1 et H3 passent

Si H1 FAIL : engagement et performance sont redondants — revoir le modèle
attendu (cf §13 plan, baselines conditionnelles).

---

## Cas de validation E2E

### Cas 1 — Joueur en bonne forme connue

**Setup** : sélectionner 5 matchs récents où le joueur "savait que ça tournait"

**Procédure** :
1. Ouvrir Match View pour chaque match
2. Onglet "Équipe" → vérifier la section Engagement en tête
3. Lire la `subtitle` (narratif basé sur percentile)

**Critère d'acceptation** : 4/5 matchs affichent `Au-dessus de votre habitude (P>60)`

---

### Cas 2 — Joueur en creux

**Setup** : sélectionner 5 matchs récents où le joueur "n'y arrivait pas"

**Procédure** : idem cas 1

**Critère d'acceptation** : 4/5 matchs affichent `Sous votre habitude (P<40)`

---

### Cas 3 — Session warmup → burnout

**Setup** : identifier une session de 6+ matchs consécutifs avec pattern
montée puis descente (le joueur sait laquelle après une longue série)

**Procédure** :
1. Ouvrir individuellement chaque match
2. Noter le score d'engagement de chaque match dans l'ordre

**Critère d'acceptation** : la séquence des scores trace un pattern montée
puis descente reconnaissable (pas tous identiques).

---

### Cas 4 — Joueur skill mais atone (⚠ critique séparation skill/forme)

**Setup** : trouver un match avec **PerfScore > 70 ET KD favorable** mais où
le joueur a peu engagé (passif, défensif). Hypothèse : 15-20% des matchs
PerfScore élevé.

**Procédure** :
1. Ouvrir Match View → onglet "Résumé" pour confirmer PerfScore > 70
2. Onglet "Équipe" → lire EngagementScore

**Critère d'acceptation** : EngagementScore < 50 sur ce match.
**Si fail** : H1 a probablement échoué — engagement et perf sont corrélés.

---

### Cas 5 — Joueur moyen mais investi

**Setup** : match avec **PerfScore ~ 50** et beaucoup d'engagements
(multiplie les actions sans toujours réussir, bcp de morts mais bcp de
kills/assists/médailles).

**Critère d'acceptation** : EngagementScore > 60.

---

### Cas 6 — Joueur sur équipe steamrollée (⚠ critique normalisation team)

**Setup** : match perdu lourdement (équipe ennemie ~2x events de l'alliée).

**Procédure** :
1. Ouvrir Match View → onglet "Équipe"
2. Vérifier que le joueur n'est **PAS** systématiquement marqué "sous-forme"
   alors qu'il a fait sa part dans son équipe affaiblie

**Critère d'acceptation** : si le joueur a contribué proportionnellement à
son équipe, EngagementScore ~ 50 (normal pour son contexte) — **pas < 30**.

---

### Cas 7 — Joueur passager sur équipe dominante

**Setup** : match gagné lourdement (équipe alliée écrase) où le joueur
particulier a peu participé (KD modeste, peu d'events).

**Critère d'acceptation** : EngagementScore < 50 (sous-engagé par rapport
à son attendu élevé dans une équipe en feu) — **pas > 70**.

---

## Cas supplémentaires (modes asymétriques)

### Cas 8 — CTF avec porteur de drapeau

**Setup** : match CTF où le joueur a passé bcp de temps porteur (kills bas
mais personal_score élevé via captures).

**Critère d'acceptation** : EngagementScore raisonnable (>= 40), pas
artificiellement bas malgré le faible nombre de kills. Validation que le
proxy `events_objectif_estimes` fonctionne.

---

### Cas 9 — Match court (< 3 min)

**Setup** : match interrompu rapidement (DC ou forfait).

**Critère d'acceptation** : pas de section Engagement affichée (HTTP 200
avec EngagementCurve null OU 422 / section masquée silencieusement).

---

### Cas 10 — Mode FFA / 1v1

**Setup** : Slayer FFA (NTeam = 1).

**Critère d'acceptation** : Engagement calculé via fallback lobby_share —
le score est cohérent (pas null par erreur).

---

## Cas backfill / Settings

### Cas 11 — Force backfill via UI Settings

**Procédure** :
1. Ouvrir Settings → tab Synchronisation → BackfillCard
2. Vérifier que "Score d'engagement" apparaît dans la grille des options
3. Cocher uniquement cette option, sélectionner un joueur
4. Cocher "Forcer le recalcul" (bouton de confirmation s'affiche)
5. Confirmer → job lancé
6. Vérifier le toast "Démarré" puis "Terminé" quelques secondes plus tard

**Critère d'acceptation** : le job termine en succès, le `done` message
indique `engagement: N` matchs calculés.

---

### Cas 12 — Glossaire applicatif

**Procédure** :
1. Ouvrir page Help → onglet Glossaire
2. Section "Métriques de performance"
3. Vérifier les 3 nouvelles entrées :
   - "Score d'engagement"
   - "Coefficient d'engagement (team_share, lobby_share)"
   - "Intensité du match"

**Critère d'acceptation** : les 3 entrées sont visibles, avec definition,
formule et exemple correctement formattés.

---

## Tableau de bord validation

| Cas | Statut | Commentaire |
|---|---|---|
| 1. Joueur bonne forme | ☐ | |
| 2. Joueur creux | ☐ | |
| 3. Session warmup/burnout | ☐ | |
| 4. Skill mais atone (séparation) | ☐ | |
| 5. Moyen mais investi | ☐ | |
| 6. Équipe steamrollée (norm. team) | ☐ | |
| 7. Passager équipe dominante | ☐ | |
| 8. CTF porteur (modes asymétriques) | ☐ | |
| 9. Match court | ☐ | |
| 10. FFA / 1v1 | ☐ | |
| 11. Force backfill UI | ☐ | |
| 12. Glossaire | ☐ | |

---

## Critères go/no-go production

- **GO production** : 10+/12 cas valides, H1+H3 PASS via `engagement-validate`
- **À ajuster** : 6-9/12 valides, ou H1 FAIL (revoir modèle attendu §13 plan)
- **Bloquant** : <6/12, ou H1 FAIL ET H3 FAIL ensemble

---

## Wirings encore TBD (Phase 4.b future)

- **Squad Page Mock 15 v2** : nécessite endpoint dédié
  `/players/{slug}/pages/squad/v2/engagement` retournant `SquadEngagementSession`
  (lobby + team_expected + per-player observed). Composant
  `SquadEngagementView.tsx` prêt à consommer.

- **Timeseries intensity tab** : nécessite endpoint dédié pour aggregat
  multi-matchs (les engagement_score sont par-match dans `player_match_enrichment`,
  un endpoint dédié devrait les exposer en série temporelle).

- **Carte synthèse Match View / Timeseries** (Mock 4) : pastille FormScore
  + sparkline pour intégration dans match list — composant non encore créé.

Tant que ces 3 endpoints n'existent pas, les composants frontend correspondants
restent inutilisables en production (ils sont prêts à être wirés mais sans
data backend).

---

## Phase recompute coefs long-term — extension du protocole de validation

> Ajout 2026-05-05. Le bug visible "courbes Attendu et Équipe superposées"
> était causé par `coef_team_share = 1.0` cold-start jamais recomputé. Le fix
> a 12 phases (cf. plan associé). La validation se fait via 4 cas
> additionnels.

### Cas A — Coef ≠ 1.0 après backfill

**Setup** :
1. DuckDB CLI : `SELECT * FROM engagement_coefficients WHERE xuid = ?`
   → table vide (cold-start) ou pas encore recompute.
2. Lancer le backfill : `POST /players/MonGT/backfill/start` body `{"engagement_scores":true}`.

**Critère** :
- Après backfill, `engagement_coefficients` contient 1 row PvP_ranked
  (et 1 PvP_unranked si historique suffisant) avec `coef_team_share` calculé
  (typiquement 0.7-1.5, jamais exactement 1.0 sauf hasard statistique extrême).
- `n_matches >= 10` (sinon cold-start préservé, c'est OK).

### Cas B — Courbes visuellement distinctes

**Setup** : ouvrir Match View onglet Combat sur un match récent du joueur
ayant désormais un coef ≠ 1.0.

**Critère** :
- Sur le chart `<EngagementMatchSection granularity="intra">`, les 3 courbes
  (Équipe alliée, Attendu, Joueur) sont **visuellement distinctes**.
- En particulier, "Attendu" (gris pointillé) ne se superpose plus à
  "Équipe alliée" (gris fonce continu).

### Cas C — Endpoint admin recompute force

**Setup** : modifier directement `engagement_coefficients` via DuckDB CLI
pour mettre `coef_team_share = 1.0` (simulation cold-start). Puis :
```
curl -X POST http://localhost:8080/api/v1/players/MonGT/engagement/recompute_coefficients
```

**Critère** :
- Réponse 200 avec `{"modes_updated": ["PvP_ranked", ...], "n_coefs_persisted": ≥1}`.
- DB : `coef_team_share` est de nouveau != 1.0 (recompute a écrasé la valeur).

### Cas D — Métriques expvar exposées

**Setup** : après un sync ou backfill, ouvrir `http://localhost:8080/debug/vars`.

**Critère** : sous la clé `levelup`, on observe les compteurs :
- `engagement_score_computed_total >= 1`
- `engagement_coef_recomputed_total >= 1`
- `engagement_coef_team_bucket_*` (au moins un bucket non nul)
- `engagement_unavailable_skips_total = 0` (si la migration est à jour)

---

## Hypothèses H1-H5 (Phase 0 du plan original)

> Le plan v7 demandait un rapport `engagement_validation_*.md` avant
> implémentation. Ce rapport n'a pas été produit avant le pivot long-term.
> Il sera produit a posteriori sur les données de production une fois la
> migration `add_engagement_pace_columns_*` rolled out, via :

```bash
cd apps/go-api
go run ./cmd/engagement-validate --player MonGT --report-output \
  ../../.ai/V7/engagement_validation_$(date +%Y-%m-%d).md
```

Si H1 ou H2 fail, c'est un signal fort que le modèle "attendu = coef × team"
est trop simple et qu'il faut passer aux baselines conditionnelles
(cf doc réflexion §13). Le code des coefs reste correct pour autant — c'est
la **modélisation produit** qui devra évoluer.
