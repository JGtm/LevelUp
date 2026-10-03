# HANDOFF — Fetch live cassé (Explorer + connexe : barre XP accueil, défis)

Date : 2026-06-22
Branche : `fix/home-session-cards-nav`
Auteur du handoff : agent "trou banner Explorer" (fix banner livré séparément, sans recouvrement avec ce sujet)

## Symptôme observé (rapporté par l'utilisateur)

Sur la page **Explorer** (recherche d'un joueur), après recherche :
- Le banner Spartan ID + le nom s'affichent, MAIS le bloc Spartan ID devrait être
  **bien plus riche** (rang carrière, emblème, skill peaks CSR/LUSR live) — il est pauvre.
- Les **graphes "Profil de combat" sont vides**.
- Le toggle **"En direct / Local" est bloqué sur "Local"** — impossible de passer "En direct".
- Les blocs "Classements CSR (saison)" et "Matchs par saison" n'affichent que ce qui
  est dérivable des **données locales**.

Connexe (même session utilisateur) : la **barre XP de l'accueil** et les **défis** ne
marchent pas non plus. En revanche le **Battle Pass fonctionne** (en live, ce n'est PAS
du cache — confirmé par l'utilisateur).

## Fait cadrant DÉCISIF (ne pas repartir sur de mauvaises pistes)

- Le **Battle Pass marche en live → les tokens Halo du joueur sont VALIDES**.
  Donc ce **n'est PAS** une absence de tokens / un store vide / un besoin de re-capture
  global. Écarter d'emblée "re-capturer les tokens" comme explication unique.
- Ce sont **certains endpoints / chemins live précis** qui retournent vide, pas tous.

## Indice concret dans les logs

`logs/air_run.log` (et air.err.log) : **403 récurrents** sur l'endpoint appearance :

```
halo_api: GET refusé auth (pas de retry)
  url=https://economy.svc.halowaypoint.com/hi/players/xuid(...)/customization/appearance
  status=403
```

C'est l'endpoint **appearance/customization** (emblème / backdrop / nameplate = identité
Spartan), celui qu'utilise `FetchLiveIdentity`. Pendant ce temps les challenges sont
récupérés ailleurs (`halo_provider: challenges fetched ... total=5`). À recouper en live :
les logs disponibles datent du 19/06 et sont compilés depuis le worktree
`levelup-multititre` — **reproduire frais** avant de conclure.

## Chemin de code (vérifié) — pourquoi le live Explorer est vide

Toutes les sources live de l'encart "Profil joueur cible" sont **hard-gated sur `hasAuth`** :

- `apps/go-api/internal/service/explorer_service.go` :
  - `buildTargetProfile()` calcule `hasAuth := tokens != nil && tokens.SpartanToken != ""`
    (tokens via `ctxkeys.HaloTokens(ctx)`).
- `apps/go-api/internal/service/explorer_service_target.go` :
  - `fetchTargetIdentityRaw` (l.64) : identité live seulement `if hasAuth` → sinon
    identité locale (joueur suivi) ou **banner pool aléatoire** (d'où "banner aléatoire").
  - `fetchTargetServiceRecord` (l.174) : `if !hasAuth → return nil` → pas de carrière / médailles.
  - `fetchTargetCSR` (l.199) : `if !hasAuth → return nil` → CSR saison vide.
  - `computeTargetCombatProfileLive` (l.28) : `if !hasAuth → return nil` → `liveMatches=[]`
    → toggle bloqué sur "Local" (`ExplorerCombatProfile.tsx` l.51 :
    `useState(hasLive ? 'live' : 'local')`).

Le contexte enrichi vient de :
- `apps/go-api/internal/api/registry_pages.go` → `ExplorerCtxWithAuth` (l.252), qui
  appelle `enrichWithHaloTokens(ctx, ownerPdb)`.
- `apps/go-api/internal/api/registry_auth.go` → `enrichWithHaloTokens` (l.103) :
  session HTTP → cache `halo.GetCachedPlayerTokens(xuid)` → `refreshTokensFromDB`
  (MultiUserTokenStore puis legacy). **Keyé sur le xuid du PROPRIÉTAIRE de la page.**

Même `enrichWithHaloTokens(owner xuid)` côté accueil (`HomeCtxWithAuth`) — d'où le lien
avec la barre XP accueil cassée.

## Deux hypothèses à départager (par reproduction live)

1. **H1 — `hasAuth` est FALSE dans les contextes `*CtxWithAuth`** alors que le BP marche.
   → bug d'injection/résolution des tokens dans le chemin HTTP `enrichWithHaloTokens`,
   distinct du chemin par lequel le BP obtient ses tokens. Vérifier : le BP passe-t-il
   par `enrichWithHaloTokens` ou par un autre provider/scheduler qui a les tokens ?
   (`apps/go-api/internal/api/handlers/progression.go`, `registry_career.go`,
   `service/career_live_*.go`, `platform/halo/provider.go`).

2. **H2 — `hasAuth` est TRUE, les fetchs sont tentés mais 403** (cf. appearance ci-dessus).
   → problème de scope / 343-clearance / endpoint, ou privacy de la cible. Dans ce cas
   le hint "Connexion Halo requise" ne s'affiche PAS (gardé sur `!auth_available`),
   d'où l'impression "aucun élément / sans explication".

## Diagnostic recommandé (reproduction)

1. Lancer le serveur depuis CE worktree (`LevelUp-go-migration`), pas un autre, pour des
   logs cohérents.
2. Page Explorer → chercher un joueur → observer la réponse de
   `POST /api/v1/players/{slug}/pages/explorer/player-query` :
   - `target_profile.auth_available` = true ou false ? (→ tranche H1 vs H2)
   - quels champs live sont nuls (`identity` riche, `career_stats`, `season_csrs`,
     `combat_profile`) ?
3. Côté logs (mettre le niveau en DEBUG si besoin) : repérer
   `explorer_common_matches` (porte `auth_available`, `career_served`) et les WARN
   `explorer_target_career_failed` / `explorer_target_identity_live_failed` /
   `explorer_target_csr_failed` / `explorer_target_combat_profile_live_failed` +
   le code HTTP sous-jacent (`halo_api: GET refusé auth ... status=...`).
4. Comparer avec le chemin BP (qui marche) pour isoler la différence d'auth.

## Pistes mémoire pertinentes

- `reference_bp_challenges_staleness_auth403` (mais BP marche ici → divergence à comprendre).
- `reference_local_dev_auth_recovery`, `reference_env_local_secret_public_client_aadsts90023`.
- `feedback_check_api_wrapper_before_blaming_auth` : valider URL/headers vs wrapper Grunt/SPNKr
  AVANT de soupçonner les tokens (l'appearance 403 peut être un host/scope/clearance, pas un token).

## Ce qui est DÉJÀ livré et NE chevauche PAS ce sujet

Fix du "trou à gauche" du banner Explorer : `apps/web/src/features/explorer/ExplorerTargetIdentityBanner.tsx`
(ajout du wrapper interne `relative overflow-hidden` + `lg:min-h-[9rem]`, parité Home).
Purement visuel, aucun lien avec l'auth. Ne pas re-toucher pour ce diagnostic.
