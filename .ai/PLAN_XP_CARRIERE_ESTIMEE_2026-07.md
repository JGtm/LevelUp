# PLAN — XP de carrière estimée par match (Timeseries) — 2026-07

> Statut : PRÊT, non exécuté. Exécution sous contrat du skill `plan-execution`.
> Branche cible : `feat/xp-carriere-estimee`. Indépendant de
> `PLAN_EXPLORER_LIVE_REPAIR_2026-07.md` (aucune dépendance croisée).
> Base factuelle : thought_log 2026-07-22 (validation empirique) + mémoire
> `project_explorer_live_target_diag`.

## Faits établis (ne pas re-vérifier, sauf B0.2/B0.3)

1. **Formule validée sur nos données** : XP de carrière gagnée par match =
   `multiplicateur(éra) × personal_score`. 4/4 paires propres (snapshots encadrant un
   seul match connu, gardes 6 min) à ratio **2,00 exact**, défaite incluse → pas de
   modificateur victoire/défaite.
2. **Éras officielles** : la progression Career Rank = Personal/Applied Score du match
   (343). Depuis **Operation: Infinite, 18 novembre 2025**, « the Applied Score
   multipliers for all matchmaking playlists will be doubled » — doublement confirmé
   permanent. Sources : annonce officielle
   https://www.halowaypoint.com/fr/news/operation-infinite-preview (fournie par
   l'utilisateur, citation exacte ci-dessus), patch notes Operation Infinite
   (support.halowaypoint.com), halopedia.org Rank_(Halo_Infinite), @HaloSupport
   18/11/2025 (permanence). Donc : ×1 avant le 2025-11-18, ×2 depuis. Le pluriel
   « multipliers » confirme des multiplicateurs PAR playlist (tous doublés) — cohérent
   avec la décision n°1 (uniforme en v1, raffinement playlist en backlog).
   (SpartanRecord n'applique pas ce ×2 sur son graphe → il affiche la moitié de l'XP
   réelle actuelle.)
3. **`xp_total` des snapshots est fidèle à l'API** — le saut de +173 230 XP du
   25/05 (JGtm) correspond à un passage rang 179 → 184 rendu par l'API elle-même
   (re-crédit côté 343 ; catalogue local inchangé, aucun déploiement local ce jour-là).
   Conséquence : `xp_total` sert d'**oracle de validation** (deltas), PAS de série
   d'affichage long terme (des re-crédits serveur peuvent créer des sauts).
4. Le score personnel est en base pour tout l'historique (`match_participants.personal_score`),
   matchs matchmade PvP uniquement (Firefight quasi absent de la base, customs absents).

## Objectif et critère de succès

Un graphe « XP de carrière (estimée) » par match sur la page Timeseries (profil suivi,
historique complet — mieux que les 25 matchs de SpartanRecord), étiqueté honnêtement,
Infinite uniquement (capability), validé contre les deltas `xp_total` (±5 % sur fenêtres
propres). Aucun flag OFF : la feature part active.

## Décisions tranchées

1. **v1 = multiplicateur uniforme par éra** : ×1 avant 2025-11-18, ×2 après (borne à
   minuit UTC ; l'imprécision d'heure de déploiement est négligeable à l'échelle d'un
   graphe). Pas de multiplicateurs par playlist en v1 : BTB (SpartanRecord ×1,8) et
   bots ne sont PAS calibrables sur nos données (échantillon vide) — raffinement en
   backlog, PAS dans ce plan.
2. Étiquette : « XP de carrière (estimée) » + tooltip méthodologique (formule + éras +
   limites : matchs connus uniquement, playlists à multiplicateur spécial approximées).
3. Les éras vivent en TOML versionné (modèle `damage_model` + constants.toml), pas en
   dur dans le Go.
4. Matchs invisibles (Firefight non synchronisé) : la feature affiche l'XP des matchs
   CONNUS uniquement — c'est l'objet du graphe (per-match). B0.3 quantifie le manque ;
   seuil de décision : si la part d'XP « invisible » dépasse 10 % chez un joueur suivi,
   proposer un chantier séparé d'audit du sync PvE (PAS exécuté dans ce plan).
5. `xp_total` : AUCUNE adaptation (fait n°3). Interdiction documentée d'en tracer la
   série brute sans garde anti-discontinuité.

## Lot B0 — Calibration, mesures préalables (lecture seule)

- [ ] B0.1 Créer la table d'éras :
      `config/titles/halo_infinite/mappings/constants.toml` (ou fichier existant du
      même rôle — vérifier l'emplacement exact du precedent damage_model avant d'écrire)
      → section `career_xp_eras` : `[{from = "", to = "2025-11-18", multiplier = 1.0},
      {from = "2025-11-18", to = "", multiplier = 2.0}]` + commentaire source (patch
      notes Operation: Infinite).
- [ ] B0.2 Vérifier sur 3 fenêtres propres pré-18/11/2025 que ×1 tient : introuvable en
      local (snapshots depuis 2026-02 seulement) → statut attendu `[!]` avec
      justification « invérifiable localement, source officielle seule » (c'est
      l'issue acceptée ; ne pas bloquer).
- [ ] B0.3 Quantifier les matchs invisibles : requête diag_q (parquets staging
      career_progression × copie shared) — part d'XP des fenêtres à 0 match connu, par
      joueur suivi (JGtm, Madina, Chocoboflor). Consigner les % dans « Découvertes ».
      Si > 10 % : rédiger (sans exécuter) une proposition de chantier « audit sync
      PvE/Firefight » en un paragraphe, référencée au backlog.
- [ ] B0.4 Documenter dans le plan-même la borne d'éra retenue et le résultat B0.3
      (mise à jour de ce fichier).

Gate B0 : items statués, % consignés, TOML relu (`go test ./internal/games/...` si un
loader est touché, sinon lecture seule).

## Lot B1 — Backend

- [ ] B1.1 `internal/analysis/xp_estimate.go` : fonction pure
      `EstimateCareerXP(personalScore int, endedAt time.Time, eras []CareerXPEra) int`
      + type `CareerXPEra` dans `internal/domain` (ou réutiliser un type mappings
      existant si le loader TOML impose son type — vérifier avant de créer).
      Tests unitaires purs : bornes d'éra (veille/jour J/lendemain du 18/11/2025),
      score 0, éras vides (→ 0, jamais de panic).
- [ ] B1.2 Loader TOML : brancher `career_xp_eras` dans le chargement des mappings du
      titre (`internal/games/mappings`), exposé via le semantic/data adapter selon le
      pattern du damage_model existant (copier le precedent, pas de nouveau pattern).
- [ ] B1.3 Capability produit : clé fine `analytics.career_xp_estimate` dans
      `config/titles/halo_infinite/mappings/capabilities.toml` (halo_infinite
      uniquement). Le service Timeseries la teste via `CapabilityMap`/`HasCapability`
      — jamais `slug ==` (ratchet `no_slug_comparison_test.go`). Titre sans capability
      → série absente de la réponse, dégradation silencieuse propre (pas d'erreur).
- [ ] B1.4 Service Timeseries : nouvelle série « xp_estimee » (per-match + binning
      serveur existant, ADR 0010) — matchs matchmade du joueur, exclusion
      `is_firefight`, score NULL → point exclu (compté dans un champ `excluded_count`).
      Timezone : fragment canonique `COALESCE(end_time_utc, end_time AT TIME ZONE 'UTC')`.
      Aucune requête SQL dans le service : passer par le repo/adapter existant de la
      page Timeseries (étendre la projection).
- [ ] B1.5 `openapi.yaml` + `make generate-types` + types manuels si interface
      hand-written (leçon D-P2-1).
- [ ] B1.6 Tests : service avec mock repo (éras appliquées, capability absente →
      pas de série) ; contracttest si la page Timeseries en a un (vérifier l'existant
      avant d'écrire).

Gate B1 : `cd apps/go-api && go test ./internal/analysis/... ./internal/service/...
./internal/games/...` puis suite complète `go test ./...` exit 0 ; `go vet ./...`.

## Lot B2 — Front (page Timeseries)

- [ ] B2.1 Nouveau graphe selon le blueprint `.ai/SPEC_ECHARTS_TIMESERIES.md`
      (buildOption, wrappers `components/charts`, skill foundations-usage) — type ligne
      par match avec binning existant de la page.
- [ ] B2.2 i18n FR **et** EN : titre « XP de carrière (estimée) » / « Career XP
      (estimated) », tooltip méthodo (formule, éras, « matchs connus uniquement »).
      Pas d'anglicisme côté FR. Aucune couleur hex (tokens sémantiques).
- [ ] B2.3 Query keys : réutiliser la query Timeseries existante (pas de nouvelle clé
      si la série arrive dans le payload existant — décision par lecture de l'existant,
      sinon clé dans `lib/query/keys.ts`).
- [ ] B2.4 `make check-types` (purge `node_modules\.tmp`) + `make test-web` ; test du
      helper de construction de série si logique non triviale (fichier `*_logic.ts`).

Gate B2 : `make generate-types && make check-types && make test-web` exit 0 ;
aucun hex dans `features/` ; vérification visuelle du graphe (JGtm).

## Lot B3 — Validation croisée (gate final)

- [ ] B3.1 Script de recoupement (diag_q, documenté dans le plan) : par joueur suivi,
      Σ XP estimée vs Δ`xp_total` sur les fenêtres SANS match invisible et SANS
      re-crédit serveur → écart attendu ≤ 5 %. Consigner le tableau dans thought_log.
- [ ] B3.2 Paires propres (méthode gardes 6 min) rejouées avec la formule finale :
      ≥ 90 % des paires à ±1 %.
- [ ] B3.3 Revue visuelle utilisateur (gate humain) + entrée thought_log de clôture.

## Gate final (delivery-checklist)

- [ ] Suite Go complète + vet + lint ; pas de diff persist/sync/migration attendu
      (sinon `-tags=integration -p 1` obligatoire)
- [ ] `make check-types` (cache purgé) + `make test-web`
- [ ] i18n parité FR/EN par typage ; zéro couleur hex ; zéro `fmt.Println`
- [ ] Thought_log par lot + clôture ; aucun item sans statut

## Hors périmètre (consigner, ne pas traiter)

- Multiplicateurs par playlist (BTB ×1,8 / bots / Firefight) — backlog, revalider
  empiriquement quand des échantillons existeront (fenêtres BTB encadrées).
- Audit sync PvE/Firefight (déclenché seulement si B0.3 > 10 %, chantier séparé).
- XP estimée sur l'Explorer target (matchs live) et les Sessions — extensions backlog.
- Halo 5 : système d'XP distinct (Spartan Rank) — jamais couvert par cette capability.

## Découvertes en cours d'exécution

(vide — B0.3 y consignera les % de matchs invisibles)

## Protocole de reprise de session

Identique au plan Explorer : thought_log + `git log --oneline -10` + premier item non
`[x]` du premier lot non clos. Lot clos = items statués + gate passé (exit code vérifié).

## Effort estimé

B0 : rapide (mesures déjà outillées). B1 : moyen. B2 : petit. B3 : rapide.
Total : ~1 session de travail.
