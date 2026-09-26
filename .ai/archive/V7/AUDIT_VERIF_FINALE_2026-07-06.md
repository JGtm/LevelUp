# AUDIT COMPLÉMENTAIRE — Vérification finale du traitement des audits 2026-07

> Date : 2026-07-06 (soir). Vérificateur : Fable (session dédiée), à la demande de Guillaume.
> Objet : contrôle qualité/robustesse/complétude du travail d'Opus sur
> [PLAN_TRAITEMENT_AUDITS_2026-07.md](PLAN_TRAITEMENT_AUDITS_2026-07.md) (lots S → N, J en cours).
> Méthode : gates mécaniques complets (build/vet/tests unitaires Go, suite intégration `-p 1`,
> typecheck/lint/vitest front) + 6 passes de vérification sur pièces en parallèle
> (tracker/journal, garde-rails, greps de gates, structure K, sécurité S, lots H/I/L/M/N)
> + revue manuelle des commits J2/J3/J7/J9 et du câblage Prestige.
> ATTENTION : la session Opus travaillait EN PARALLÈLE de cette vérification (J7, J9 puis
> la clôture J `18f3c7ee7` commités pendant l'audit). État vérifié = `8d385cc70` + arbre ;
> les findings résorbés en direct par `18f3c7ee7` sont annotés [RÉSORBÉ EN DIRECT].
>
> Plan de reprise associé : [PLAN_CLOTURE_AUDITS_2026-07.md](PLAN_CLOTURE_AUDITS_2026-07.md)
> (les findings ci-dessous y sont mappés item par item — AUCUNE correction appliquée ici).

---

## 1. Verdict global

**Le travail est massivement livré et de bonne facture** : sur ~120 items, chaque `[x]`
sondé a son commit, les grands invariants (anti-ART, _latest, auth, PathResolver, dédups H)
tiennent sur pièces, et les garde-rails livrés mordent réellement. Les reports `[!]`/`[~]`
sondés sont réels et honnêtement justifiés.

**MAIS la campagne n'est PAS terminable en l'état** : 1 gate transversal est cassé
(typecheck front, 13 erreurs), 1 bug fonctionnel majeur dort avec tests verts (hook
Prestige post-sync jamais exécuté), 1 endpoint révélateur d'identité reste sans garde,
le tracker/journal a décroché de la réalité sur le lot J et la fin du lot K, plusieurs
découvertes §7 marquées « à traiter » n'ont jamais été reprises, et la « vérification
finale » prévue par le plan (relecture des 4 audits) n'a jamais été exécutée.

| Lot | Verdict | Notes |
|---|---|---|
| S | ✅ conforme, 1 trou | `GET /jobs/{job_id}` non gardé (VF-3) ; table à rafraîchir |
| A | ✅ tient | greps A1-A5 verts sur l'arbre actuel |
| B | ✅ tient | garde-rails B8/B15 OK, allowlists vivantes |
| C | ✅ tient | docs d'orientation vraies |
| D1 | ✅ tient | flags supprimés, télémétrie D1a en place ; date prod D1a à noter au merge (arme D2) |
| E | ✅ tient | tripwires étendus OK ; 1 entrée morte allowlistRawDelete (VF-6) |
| G | ⚠️ tient, résidus | dead code transitif §7 jamais purgé (VF-5) ; coverage.html versionné (VF-9) |
| F | ⚠️ tient, promesse manquante | garde-rail `halowaypoint` promis au gate F jamais créé (VF-7) |
| H | ✅ conforme | 8/8 vérifiés sur pièces, garde-rails mordants |
| I | ✅ conforme, 2 accrocs | 2 littéraux FR restants XboxLoginPage ; typecheck cassé par I2/I4 (VF-2) |
| J | ✅ clos en direct (`18f3c7ee7`) | J3/J7 corrects sur pièces (J3 réplique une dette start_time préexistante, VF-13) ; J4/J6 différés measure-first ; DETTE_ASSUMEE §4 + entrée §6 restent dus |
| K | ✅ largement conforme | mesures vérifiées (NewRouter 89 L, api 5 fichiers, freeze 80) ; débris doc (VF-10) ; server_apiv1.go = god-file neuf 1286 L |
| L | ✅ conforme, doc inversée | doc CONTRACT_VALIDATE non purgée après L4 (VF-8) |
| M | ✅ conforme | CI -p 1 réel et bloquant ; M1/M5 différés OK |
| N | ✅ conforme (partiel assumé) | N4/N5 livrés ; N1/N2/N3 différés documentés |
| D2 | ⏸ différé légitime | pré-conditions non réunies (branche pas en prod) |

## 2. Gates transversaux (re-exécutés ce soir)

| Gate | Résultat |
|---|---|
| `go build ./...` (CGO msys64) | ✅ exit 0 |
| `go vet ./...` | ✅ exit 0 |
| `go test ./...` (unitaires) | ✅ exit 0, 0 FAIL |
| `go test -tags=integration -p 1 -timeout 900s ./...` | ✅ exit 0, 112 packages ok, 0 FAIL (filtre ancré `^--- FAIL:`) |
| `npm run lint` (apps/web) | ✅ 0 erreur (68 warnings baseline) |
| `npm run test` (vitest, hors sandbox) | ✅ 2071 passed / 14 skipped |
| `npm run typecheck` (tsc -b) | ❌ **13 erreurs** (VF-2) |

## 3. Findings (VF-x), par sévérité décroissante

### MAJEUR

**VF-1 — Le hook Prestige post-sync n'est exécuté sur AUCUN chemin de sync (feature
silencieusement morte, tests verts).** Vérifié de première main :
`handlers/sync_handler.go:226` `WithPrestigeHook(_ PrestigeHook)` est un stub qui JETTE
son paramètre (`return h`) alors que `server_apiv1.go:489` lui passe
`prestigeBundle.RunPostSync` ; et `SyncEngine.WithPrestigeHook` (`sync/engine_options.go:78`)
n'a AUCUN caller prod (grep : définition + stub seulement) → `e.prestigeHook` reste nil,
`engine.go:713-714` ne tire jamais. Conséquence : `prestige.RunPostSyncHook`
(ré-évaluation des défis actifs après sync) ne tourne jamais — ni sync HTTP, ni
auto-sync scheduler, ni V2. Aggravant : C7 a ACTÉ Prestige défaut ON (ADR 0005) ; la
découverte était consignée en §7 (« LOT D1/D1f — BUG latent ») avec la mention « à
VÉRIFIER puis câbler », jamais reprise. `prestige/sync_hook_test.go` teste le hook en
isolation → vert trompeur (anti-pattern « dead code museum »).

**VF-2 — Typecheck front CASSÉ à HEAD (13 erreurs) alors que les gates I2/I4/L5
déclaraient « typecheck 0 ».** Trois sources :
- 6 × TS2345 `string` → `ManifestLocale` introduites par I4 `ce9cdf08e` (dédup
  intlLocale) : `SessionMultiSelect.tsx:76`, `HomeCitationsNearCompletion.tsx:159`,
  `LeaderboardBlock.tsx:419/502/544`, `MediaPage.tsx:71` — props `locale` typées
  `string` passées à `intlLocale(ManifestLocale)`/`getTexts`.
- 1 × TS2345 `string | null` : `media/queries.ts:345` — `queryKeys.mediaMatchCandidates`
  (centralisée par L5 `91492e360`) exige `string`, le hook passe `filePath: string | null`.
- 6 × TS2591 (`node:fs`/`node:path`/`process` introuvables) dans les 2 garde-rails
  `lib/formatters/calendar.guard.test.ts` (I2) et `lib/query/keys.guard.test.ts` (L5) :
  `tsconfig.app.json` a `types: ["vite/client", "vitest/globals"]` sans `node` — ces
  fichiers n'ont JAMAIS pu typechecker.
Cause process probable : `npm run typecheck` = `tsc -b` (incrémental,
`.tsbuildinfo` dans node_modules/.tmp) → un cache chaud peut rendre un faux vert ; ou
le gate n'a pas été relancé après coup. Même famille de leçon que le `-p 1` (faux vert
d'outillage). Vitest, lui, passe (esbuild ne typecheck pas).

**VF-3 — `GET /jobs/{job_id}` accessible anonymement** (monté sur `r` nu via
`registerJobsHuma`, `server_apiv1.go:485` + `huma_routes.go:57`) : répond le
`AsyncJobStatus` complet (PlayerSlug, type de job, warnings, messages d'erreur) =
révélateur d'identité, hors table S3. Aggravant : `newJobID()` =
`job_<UnixNano>` (`platform/jobs/store.go:253-255`), horodaté donc énumérable en
théorie. Tous les jobs sont créés depuis des flux déjà authentifiés → `RequireAuth`
serait no-op single-user/démo, cohérent lot S. Gap de méthode associé : la revue S3
était un grep MANUEL, aucun ratchet « routes nues » automatisé (un chi.Walk vs
allowlist aurait attrapé /jobs).

### MAJEUR (ajout post-audit, 2026-07-06 soir)

**VF-16 — La CI de branche est ROUGE et personne ne la regardait.** `gh run list
--branch refactor/audits-2026-07` : job « CI » en `failure` au moins depuis le push K2a
(runs 28810616253, 28818636316...). Deux jobs échouent : (1) « Frontend (TypeScript +
Vite build) » étape Type-check = VF-2 ; (2) **« Go Baseline Tests (non-régression) »**
étape « Vérifier suite baseline de tests pré-migration » — très probablement les
renommages/déplacements de tests du lot K (K3a-e ont déplacé ~40 fichiers de test de
package) : la baseline référence des noms/paquets disparus. Règle mémoire applicable :
« vérifier les renommages avant rebaseline » (jamais rebaser pour masquer une vraie
disparition de test). Leçon process : les gates étaient re-joués LOCALEMENT mais jamais
croisés avec les runs CI réels de la branche — deux signaux rouges publics ignorés
pendant toute la fin de campagne.

### MOYEN

**VF-4 — Tracker/journal décrochés de la réalité (contrat §0 du plan violé sur la fin
de campagne).** Détail vérifié :
- J2 (`df5832d60`+`305b6b959`), J3 (`dfeb199f3`), J7 (`f6b8cce4a`), J9 (`8d385cc70`,
  doc) LIVRÉS mais non cochés au plan ; blocs « J2/J3/J7... DIFFÉRÉS measure-first » et
  « Gate J (PARTIEL) » désormais faux. **[RÉSORBÉ EN DIRECT — `18f3c7ee7` a statué
  J1-J9 + thought_log pendant cet audit ; DETTE_ASSUMEE §4 et l'entrée §6 J restent dus.]**
- I2 : I2b COMPLET (commits `974ac5a33`…, DETTE_ASSUMEE à jour) mais le plan garde
  « RESTE ~24 sites [!] ». I4 : marqueur `[!] non exécuté ce tour` contredit par deux
  « ✅ LIVRÉ » dans le corps ; la sous-puce (ii) EncounterSplitBars est livrée
  (`7f70297b8`) mais non purgée.
- §6 Journal : entrées de clôture ABSENTES pour H, I, J, K, L, M, N (le Gate H exige
  « comptes au Journal » — ils ne sont que dans thought_log).
- thought_log : rien pour J3/J7/J2(2)/J9 (règle CLAUDE.md « entrée avant tout commit »).
- K3f : « TOUS traités ✅ » suivi d'un paragraphe « RESTE (7) » listant... les 7 traités
  (périmé non purgé). Bloc « BILAN SESSION /goal » (l.1072-1096) en retard sur ses
  propres items. P1/P2 jamais statués.
- DETTE_ASSUMEE_2026-Q3.md : entrée « E7 » dont le CONTENU décrit D1a/D2 (le vrai E7 —
  DDL bootstrap sync/schema.go — n'est nulle part) ; résidus K post-2026-07-06 absents
  (K1b-legacy, K1d-reste, K1h, K1j, K1k, K1l-reste, K1n[!], K2b-drain, K3b-ratchet,
  K3f décisions packages) ; §4 J périmé ; N3(b/c/e) absents ; footer (« les livrés
  sortent ») contredit par les entrées ✅ conservées.
- **La vérification finale prévue (§5 : relire les 4 audits, confirmer chaque finding
  statué, bilan utilisateur) n'a jamais été exécutée ni planifiée.**

**VF-5 — Découvertes §7 « à traiter » jamais reprises → dead code museum réel** :
- `insertHighlightEventsFromData` (`sync/engine_highlight_events.go:163`) : 0 caller
  prod (son caller `insertFetchedMatch` est supprimé depuis D1b), 2 callers test ;
  docstring cite le caller mort.
- Trio `sync/writes.go` `InsertRegistryIfNotExists:34` / `InsertParticipants:99` /
  `InsertMedals:188` : 0 caller prod, tests only ; la justification de
  `shared_write_guard_test.go:54,64` (« import OpenSpartan ») est PÉRIMÉE depuis E1 —
  2 entrées d'allowlist ON CONFLICT protègent du code mort.

**VF-6 — Allowlists de garde-rails avec entrées mortes (trous latents)** :
- `platform/auth/sentinel_test.go:50` et `:155` : `internal/api/registry.go` (fichier
  supprimé au lot K) toujours allowlisté dans `allowedEnvReaders` ET
  `allowedDuckDBWriters` → un fichier recréé à ce chemin lirait les env vars auth sans
  déclencher le sentinel. (Signalé au journal S, jamais purgé.)
- `sync/no_art_patterns_test.go:146` : `allowlistRawDelete` pointe
  `skill_rating_postsync_persist.go` (n'existe plus) ET ce map n'est PAS couvert par
  `TestAllowlistJustifiesEverything` (qui ne vérifie que `allowlistArtPatterns`).
- `platform/duckdb/no_attach_on_social_test.go:312` : entrée spéculative morte
  `social_persister_combined.go`.

**VF-7 — Garde-rail `halowaypoint` promis au gate F jamais créé.** Le gate F (plan
l.780-781) : « grep halowaypoint hors games/halo_infinite + platform/halo → 0 (allowlist
temporaire datée si besoin) ». Réalité : K3e a déplacé le client dans
`internal/sync/haloclient/` (5 fichiers prod hors zones), plus `platform/auth/halo_exchange.go`,
`domain/title/auth_descriptor.go` (defaults), `internal/assets/fetcher_gamecms.go`,
`halotest/fake_server.go`, ~10 cmd/. AUCUN ratchet n'existe → l'invariant re-divergera
(règle 6 : factorisation sans garde-rail).

**VF-8 — Doc inversée sur flag supprimé** : L4 a supprimé `LEVELUP_CONTRACT_VALIDATE`
mais `docs/CONFIGURATION.md:224`, `docs/FR/CONFIGURATION.md:225` et
`.env.local.example:203` le documentent encore comme actif (anti-pattern n°9, celui-là
même que D1d venait de soigner).

**VF-9 — `apps/go-api/coverage.html` (+ `coverage_baseline.txt`) versionnés** : l'HTML
contient le code source COMPLET de fonctions supprimées (session_compare handler, Q4
morts...) — artefact de build à dé-tracker, entretient l'illusion de code vivant.

### MINEUR (réels, bornés)

**VF-10 — Débris de doc/refactor K** : `server.go:490` `//nolint:gocyclo // Routeur
central : mount de ~80 endpoints...` collé sur `startSessionPurgeLoop` (~20 L, trivial) —
justification mensongère héritée ; fragment de doc orphelin `server.go:126-127` ;
« 4 fichiers racine api » réel = 5 (server_apiv1.go oublié du bilan) ;
`server_apiv1.go` = god-file NEUF de 1286 L sans exemption fichier (nolint fonctionnels
seulement) ; `sync_root_freeze_test.go:21` historique arrêté à 106 (const = 80) ;
`player_repos_test.go:112` nolint:funlen nu ; `db.go` à 499 L (1 ligne du seuil).

**VF-11 — I1 incomplet sur un bloc** : `features/auth/XboxLoginPage.tsx:390` (`Identifiant`)
et `:422` (`'Connexion…' : 'Se connecter'`) FR en dur alors que les clés
`common.auth.username_label/login_action/login_pending` existent et que les champs
voisins utilisent `t()`.

**VF-12 — Commentaires stale post-suppression** (doc inversée dispersée) :
`sync/engine_fetch.go:29-32` (décrit « les deux chemins » dont insertFetchedMatch mort) ;
`engine_highlight_events.go:162` ; mentions `processMatch` vivantes dans
`backfill_personal_scores.go:7`, `engine.go:100`, `csr_shared_backfill.go:5`,
`csr_writes.go:5` ; `service/session_compare_service.go:1,22` (header décrit le service
supprimé) ; commentaire orphelin `ReassociateMedia` en FIN de
`media_service_upload.go:189-190` ; `ops/healthcheck.go:8` doc RECOMMANDE `fmt.Println`
(contraire règle 3) ; `eslint.config.js:30-31` (« warn Phase 0 » périmé) ;
`.golangci.yml:8,15` header (« 5 args », « 60 statements ») périmé ;
`engine.go:175` cite RunBackfillLUSR comme existant ; `notify/discord_extra_test.go:51`
fixture écrit la clé morte `discord_notify_new_media`.

**VF-13 — J3 réplique une dette préexistante** : `Q29HistoryForAvg` (queries_match.go:244)
ET sa variante bulk (`:289`) ordonnent par `r.start_time` BRUT (règle CLAUDE.md n°8 —
fragment canonique obligatoire). Préexistant à J3 (« sémantique identique » respectée),
mais le garde-rail H1 ne couvre pas ce cas (il n'interdit que le littéral COALESCE
recopié, pas un `ORDER BY start_time` nu) → la fenêtre « 50 derniers matchs » peut
mal trier les matchs à `start_time` NULL/TZ-décalés (imports OpenSpartan).

**VF-14 — Divers front** : 4 copies locales de `perfTierToken`
(`SessionSummaryCard.tsx:35`, `mapPerfVsHistoryChart.ts:37`,
`squadSessionTimelineChart.ts:33`, `PlayerDetailPanel.tsx:288`) = règle 6 (≥3 copies)
non couverte par le ratchet perfScale ; `lib/formatters/date.ts:52` `'fr-FR'` en dur
dans un formatter partagé (dette préexistante) ; garde-rail experienceCascade absent
(2 copies — tolérable, sous le seuil).

**VF-15 — Lot S cosmétique** : table S3 — « GET /session (302) » ≠ réel
`POST /session/context` ; « /gamertags?q= » ≠ réel `/directory/gamertags/search` ;
lignes décalées post-J2. `groups.go` dans l'allowlist Huma sans date cible.

## 4. Leçons process (à capitaliser)

1. **`tsc -b` incrémental peut produire un faux vert** (VF-2) — même famille que le
   faux vert `-p 1` du 2026-07-03. Gate fiable = purge du `.tsbuildinfo` ou `tsc -b --force`.
2. **Un garde-rail sans self-check d'allowlist pourrit** (VF-6) : le tripwire ART a un
   self-check pour UN map sur deux ; sentinel/attach n'en ont pas.
3. **Les découvertes §7 « à traiter » sans propriétaire ne sont jamais reprises**
   (VF-1, VF-5) : le plan de clôture doit les convertir en items cochables.
4. **Le tracker doit être mis à jour DANS le commit qui livre** (VF-4) : les 4 derniers
   commits J n'ont touché ni plan, ni thought_log, ni DETTE_ASSUMEE.
