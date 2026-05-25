# Backlog — Coach proactif × Prestige (post-V2)

Référence : ADR 0020 — Coach proactif : pont vers Prestige. ADR 0021 — Synthèse
dynamique de Template et Arc ad-hoc.

Ce backlog liste les extensions volontairement reportées **après** la livraison
V2 du pont coach → Prestige. Chaque entrée doit faire l'objet d'une décision
produit séparée avant ouverture d'une nouvelle branche.

## V3 — Squad coach

**Idée** : étendre le pont aux escouades. Quand le coach détecte un pattern
collectif (composition d'équipe orientée combat / objectif / support), il
propose un `SquadChallenge` ou un pool d'arcs calibrés sur la composition.

**Pré-requis** :
- Profil agrégé d'escouade (moyenne LUSR sur les axes par membre).
- Signal coach niveau escouade (à concevoir — `coach.GenerateInput` est
  aujourd'hui per-user).
- Extension de `prestige.RefreshSquadPool` pour accepter un filtre coach.

**Effort estimé** : lourd. Touche `coach`, `coach_advisor`, `prestige`,
front-end squad UI.

## V3 — Coach négatif soft

**Idée** : autoriser le coach à signaler des **tendances dégradées** (LOWESS
négative soutenue, baseline en chute) **sans culpabilisation**. Reformulation
positive : "tu as l'opportunité de stabiliser X" plutôt que "tu régresses sur X".

**Pré-requis** :
- Décision produit explicite (ADR 0014 §6.1 cadre aujourd'hui le coach comme
  strictement positif — il faut un amendement ou une option par joueur).
- Tone guidelines pour i18n (FR + EN) qui maintiennent le cadre positif.
- A/B test ou opt-in séparé pour éviter d'imposer ce mode à tous les joueurs
  qui ont activé le coach proactif standard.

**Effort estimé** : moyen côté code, lourd côté produit/UX.

## V2.1 — Plumbing `Source` → `prestige_telemetry.source`

**Idée** : ajouter une colonne `source` dans la table `prestige_telemetry` et
propager `CreateChallengeRequest.Source` jusqu'à `EmitCreated` (puis aux
EmitTransition pour suivre le devenir des challenges coach).

**Bloqueur actuel** : la table `prestige_telemetry` est écrite mais jamais lue.
Aucun script, aucun handler, aucun dashboard ne l'interroge. Le commentaire
dans `types.go` mentionnant `analyze_prestige_tuning.py` réfère à un script
qui n'existe pas — c'était une intention V1, jamais matérialisée.

**Pré-requis avant d'implémenter** :
- Construire d'abord un consommateur : soit un endpoint `GET /diag/prestige/telemetry`
  qui agrège (taux acceptance par source, complétion par source), soit un
  analyseur CLI Go (`cmd/prestige_tuning_analyze`) lisant directement la DB.
- **PAS DE PYTHON** : analyseur en Go ou DuckDB CLI direct (`duckdb stats.duckdb -c 'SELECT ...'`).
- Définir les métriques cibles : ratio accept/reject par source, taux completion
  coach vs user vs pilot_mode, distribution de strength des proposals acceptées.

**Sans consommateur** : ajouter la colonne maintenant = écriture pour personne,
dette de schéma qui grossit. Reporter jusqu'à ce qu'un besoin analytics concret
émerge.

**Effort estimé** : rapide (ALTER TABLE + 2 lignes Go) — mais l'analyseur côté
consommation est moyen (1 commit endpoint diag ou CLI).

## V2.1 — Job GC `coach_advisor_template_gc`

**Idée** : job nocturne qui supprime les templates `source='coach_synthesized'`
avec `usage_count=0` et `updated_at > 90 j`.

**Pré-requis** :
- Mesure post-livraison V2 du volume de templates synthétisés.
- Si volume reste < 200 templates synthétisés total, ne rien faire.

**Effort estimé** : rapide (1 commit avec scheduler).

## V2.1 — Compteurs expvar

**Idée** : ajouter des compteurs `coach_proposals_generated_total{kind,origin}`,
`coach_proposals_accepted_total{kind,origin}`, `coach_proposals_completed_total`
pour mesurer l'efficacité du coach proactif vs user / pilot_mode.

**Pré-requis** :
- Conformité ADR 0009 (expvar stdlib).
- Décision sur les labels (kind, origin, signal_kind ?) pour ne pas exploser
  la cardinalité.

**Effort estimé** : rapide.

## V3 — Apprentissage automatique de `synthesis_grammar.toml`

**Idée** : analyser `prestige_telemetry` pour ajuster la grammaire de synthèse.
Si les templates synthétisés sur metric=X ont taux de complétion < 30 % sur
50 acceptations, retirer X de la grammaire ou réduire ses windows autorisés.

**Pré-requis** :
- Job analytics qui lit `prestige_telemetry` + `coach_proposal`.
- Mécanisme de PR automatique sur `synthesis_grammar.toml` (ou ajustement runtime
  via override en DB metadata).
- Validation manuelle obligatoire avant application.

**Effort estimé** : lourd. Demande infra analytics.

## V3 — Cross-titre arcs

**Idée** : permettre un arc qui couvre deux titres (ex. progression accuracy
partagée Halo Infinite + futur titre Halo MCC ou cross-game). Aujourd'hui
chaque `Arc` est lié à un `title_slug` unique.

**Pré-requis** :
- Décision produit (les joueurs ont-ils ce besoin ?).
- Refonte mineure `Arc.TitleSlug` → `Arc.TitleSlugs []string` ou table
  `arc_titles` séparée.
- Adapter les répartitions PP par titre (un challenge cross-titre crédite-t-il
  les PP sur chaque titre ?).

**Effort estimé** : moyen côté backend, lourd côté UX.

## V3 — Coach narrative tone

**Idée** : choix par joueur du "ton" du coach (technique / motivant / humour /
neutre) influençant les `labelFR` / `labelEN` synthétisés et les réasons.

**Pré-requis** :
- Banque de templates i18n × 4 tons par signal kind.
- Setting joueur `coach_tone` (extension de `user_preferences`).

**Effort estimé** : moyen (surtout côté contenu i18n).

## V3 — Notifications push externes

**Idée** : émission externe (push mobile, email, Discord) des proposals coach
les plus fortes. Aujourd'hui les notifications restent in-app
(`player_notifications` UI uniquement).

**Pré-requis** :
- Décision produit (sécurité / vie privée).
- Infra push mobile (non disponible aujourd'hui).
- Préférences fines par catégorie de notif.

**Effort estimé** : lourd.

---

**Ordre de priorisation suggéré (V3)** :

1. V2.1 — Compteurs expvar (mesure → décision)
2. V2.1 — Job GC (si volume justifie)
3. V3 — Squad coach (le plus aligné avec les fondations Prestige squad existantes)
4. V3 — Coach négatif soft (demande arbitrage produit explicite)
5. Le reste — décisions au cas par cas
