# Revue de code — fenêtre 2026-07-07 → 2026-07-17 (24fc02f2f..HEAD)

> Revue adversariale multi-agents du travail des 10 derniers jours : 897 fichiers,
> +86 950 / −47 219 lignes. Méthode : 9 chercheurs (3 angles correctness, 3 angles
> qualité, altitude, conventions CLAUDE.md) → 45 candidats → déduplication →
> 9 vérificateurs adversariaux (verdict par candidat, réfutation uniquement sur
> preuve constructible). Bilan : 31 confirmés/plausibles, 8 réfutés. Aucun fix
> appliqué (revue seule).

## Top 10 (rapportés via findings, les plus sévères d'abord)

1. **Pollution player-DB via season pass** — `internal/api/wire/registry_auth.go:105`.
   `SeasonPassCtxWithAuth` n'applique pas `forcePageIdentityXUID` : BP/défis du
   COMPTE CONNECTÉ fetchés puis persistés dans la DB du JOUEUR DE LA PAGE
   (estampillés xuid=X, servis en fallback 24 h). 4e occurrence du bug PR #63 ;
   2 constructeurs forcés sur 5, aucun garde-rail ratchet. CONFIRMED.
2. **Login → onboarding forcé (cache bootstrap anonyme)** —
   `apps/web/src/features/auth/XboxLoginPage.tsx:70`. `fetchQuery` sans
   staleTime:0 sert le bootstrap anonyme (défaut global 5 min) après le
   device-flow → joueur établi coincé sur l'onboarding (garde qui relit le même
   store périmé). Régression du fix e4f3da775. CONFIRMED.
3. **Squash M3-M5 : DB mi-bloc irréparable** — `internal/migration/registry.go:258`.
   `supersededBaselineSatisfied` ne teste que la sentinelle ; backup pré-05/2026
   restauré → bind de `player_csr_snapshots_latest` échoue sur l'ancien schéma →
   `OpenPlayerDB` en échec permanent + INSERT challenges cassé (colonnes render).
   Aucun step restant pour réparer. CONFIRMED.
4. **Squash : sentinelle antérieure à des steps ordonnés avant elle** —
   `internal/games/halo_infinite/migrations/steps_player_baseline.go:72`.
   Sentinelle 05-24 < `player_add_expected_win_prob` 05-28 ; le chemin DM-5
   n'exécute jamais l'ensure additif → persist LUSR définitivement cassé pour les
   DBs de la fenêtre. Faille structurelle (idem render_columns 06-22,
   response_bins 07-11). CONFIRMED.
5. **Augment CSR limité à 1 playlist sur ~7** — `internal/sync/engine_postsync_csr.go:38`.
   `MAX(fetched_at)` global au titre vs estampillage PAR playlist du scraper ;
   résultat non vide → fallback jamais déclenché. La vue `_latest_by_batch`
   partitionne, mais cette requête lit la table brute. CONFIRMED.
6. **Trous réels du sweep recovery player-DB** —
   `internal/platform/duckdb/milestones_earned_repo.go:69` : `r.db.QueryRow` PLAT
   sur un handle qui est `pdb.Player` (invisible au garde-rail grep, champ nu) ;
   + 4 `.Player.Exec(` plats (career_progression_partial.go:86,
   career_live_repo.go:213, engagement_score_repo.go:202, fanout_repo.go:93)
   alors qu'`ExecRecovered` existe. La race « database is closed » reste
   atteignable. CONFIRMED.
7. **DELETE avec LIKE non échappé (cleanup média)** — `cmd/cleanup_media_index/main.go:126`.
   Les `_` de « Halo_5_Guardians- » matchent tout caractère → prédicat plus large
   que l'indexeur (HasPrefix strict) → purge --foreign-only peut supprimer likes
   et associations non reconstructibles. Amortisseurs : --dry-run, outil one-shot.
   CONFIRMED.
8. **Rotation RT non persistée avant le XSTS** — `internal/platform/auth/refresh_user_xsts.go:47`.
   RT rotaté écrit seulement au Upsert final ; échec transitoire XSTS → store
   garde le RT consommé → invalid_grant. LATENT : aucun appelant prod (prévu
   câblage OnAuthExpired) — à corriger AVANT câblage (modèle : persistRotatedRT).
   CONFIRMED.
9. **N+1 courbe d'engagement** — `internal/service/engagement_player_service.go:340`.
   `LoadResponseBins` par match (2 requêtes/appel dont scan information_schema)
   pour un résultat fonction de (xuid, mode_category) → ~400 requêtes/affichage
   au lieu de ≤4. Mémoïser par mode_category + cacher le check d'existence.
   CONFIRMED.
10. **Budget API débité sur le mauvais bucket** — `internal/service/career_live_fetcher.go:158`.
    Limiteur clé sur HaloXUID (= xuid de la PAGE post-forçage) alors que les
    tokens sont ceux du compte connecté → quota réel consommé hors comptage,
    bucket de la page faussement throttlé (retour possible des 429 que Sujet 2
    devait éliminer). CONFIRMED.

## Confirmés sous le cap (non rapportés, à traiter en lots qualité)

### Correctness secondaires
- Tooltips per-kill/vagues morts sur la carte Dominance — passage du tooltip
  global à `trigger:'axis'` alors que les `series.tooltip {trigger:'item'}` ne
  sont honorés qu'en trigger global item (`MatchTugOfWarChart.tsx:311` ; la
  partie « mauvais index » du candidat est réfutée).
- Annuler du prompt admin n'annule pas — `window.prompt(...) ?? undefined` puis
  mutate inconditionnel (`DetectionsPanel.tsx:49`).
- Notifications relues après F5 dropdown ouvert — flush uniquement à la
  transition open→false d'un composant monté (`NotificationsBell.tsx:62`).
  Portée limitée (nav SPA OK).
- Libellés placement FR en dur (« Non classé », « En placement (x/y) ») visibles
  en UI EN — `HomeRecentPlaylistsCard.tsx:127,159-163` (préexistant mais lignes
  retouchées par le diff sans localisation).
- `store.Load` avalé silencieusement sur le chemin store-first —
  `access_token_store_first.go:58` (bascule legacy sans log, télémétrie
  legacy_source_used trompeuse ; gate D2 faussable).
- Cron leaderboard rapporté toujours vert — `world_leaderboard_cron.go:187`
  (`ReportCronRun(..., nil, ...)` inconditionnel ; choix « liveness » commenté et
  partagé par 4 autres crons — à trancher comme politique, pas au cas par cas).
- Graine de saison figée `csrseason13-2` — `leaderboard_scraper.go:166`, unique
  point d'entrée de la découverte ; graine morte = leaderboard gelé (déclencheur
  hypothétique, fixture à l'appui du contraire).

### Efficacité (VPS 2 vCPU / 2 Go)
- Panneau admin fraîcheur : résolution des player-DBs jetées + 13 scans
  MAX+JOIN par ouverture (`registry_monitoring_freshness.go:126`) ; 1 requête
  groupée par titre suffit ; ouvre même des DBs pour les profils auth_only.
- Tables monitoring append-only SANS rétention ni sweep, vues ROW_NUMBER sur
  table entière polées 30 s (`monitoring_schema.go:91`) — croissance non bornée ;
  pattern CapAndSweep des notifications à répliquer.
- Briefing serveur calculé même en mode player (query non gatée) + input
  match-ID non débouncé → 1 POST par frappe avec recompute complet
  (`ExplorerPage.tsx:205-228`) ; surcoût CPU pur (pas d'I/O suppl.).

### Duplications (règle n°6)
- Mapping outcome int→win/loss/tie/dnf : 4 copies front à DÉFAUTS DIVERGENTS
  ('tie' vs null vs 'dnf') — `ExplorerBriefing.logic.ts:56`, `_shared.ts:62`,
  `TimeseriesPage.summary.tsx:30`, `SquadSynergiesPage.tsx:23`. À centraliser
  avec garde-rail (contact ADR 0006).
- « Plus longue série » : 4e copie Go (`briefing_streaks.go:44` vs
  highlights/synthesis/detectTilt).
- Delta signé ±0 : 3 copies strictes + 1 approchée, glyphes '−'/'-' divergents
  (`ExplorerBriefing.logic.ts:14`).
- `coalesceStr` duplique `coalesce` du MÊME package (`briefing.go:443` vs
  `enrich.go:332`).
- Seeder synthétique : listes de colonnes des tables critiques recopiées de
  persist sans abstraction (`seed_demo_synthetic_player.go:263`,
  `seed_demo_synthetic_shared.go:109/172/218`) — la recette ADR 0026 ne touche
  que persist → corpus démo rotera silencieusement.
- `deltaToken` copié dans 2 fichiers briefing (sous seuil mais hôte naturel
  disponible : `ExplorerBriefing.logic.ts`).
- `campaign_exclusion.go` : triplication interne dès la création (quoting GUID
  ×3 + 2 fonctions identiques à la source près).

### Altitude / architecture
- Identité de page : le sujet devrait être un paramètre explicite
  (GetSpartanIdentityFor existe) ou l'invariant imposé par un garde-rail
  ratchet — 3 patchs point-par-point dans la fenêtre, la 4e occurrence (season
  pass) est le finding n°1.
- Recovery player-DB par construction (type n'exposant que les variantes
  *Recovered) fermerait les 3 classes de trous du garde-rail grep.
- Préfixe RpsTicket d=/t= : provenance du token connue à l'acquisition mais
  jamais stockée → retry aveugle qui masque les 401 d'autres causes (coût borné,
  assumé en commentaire — dette consciente).
- Lightbox : 2 écrivains de video.muted coordonnés par attribut DOM
  data-audio-off + hypothèse d'ordre des effets (couvert par un test de
  régression — PLAUSIBLE, à surveiller).
- Fallback noms média : mini-framework à closures ~90 L à usage unique
  (`media_repo_registry.go:205`) — forme directe équivalente.
- `opts ...PostSyncDeltaOptions` : seul opts[0] lu, 2e option silencieusement
  ignorée (`post_sync_deltas.go:280`).

### Conventions
- `cmd/engagement-calibrate/main.go` (fichier neuf) : chemin data/ à la main
  (l.101, PathResolver.PlayersRootDir existe), log.Printf (l.58/114/176), champ
  `Bins` jamais utilisé + `_ = ref` contredisant la godoc. NB : le ratchet
  no_data_path_join ne couvre pas cmd/ (~20 précédents).
- Emojis ajoutés : `cmd/levelup/cmd_data.go:233` (sortie CLI) ;
  `internal/notify/discord.go:297-299` (nuance : l'emoji est le contenu du
  message Discord, convention locale préexistante du fichier).

## Réfutés notables (huit, avec preuve)

- « engagement_response_bins jamais créée en prod » : le step discret A ÉTÉ
  déployé pendant la fenêtre 07-12 11:25 → 07-13 12:26 (reflog) avant le squash ;
  le cas résiduel (backup pré-07-12) est couvert par les findings squash.
- « MSAL-cache-only invisible » : `resolver.go:197-203` traite « token vide sans
  erreur » → signalReauth → bannière + panneau santé dès le premier cycle.
- « Titre absent des query keys » : `switchTitle` fait `queryClient.clear()`
  global (appShellStore.ts:198-199, antérieur à la base du diff) — le bug allégué
  ne peut pas se produire ; le fix home ec36e7b40 était ceinture-bretelles.
- « Explorer sert l'identité du compte connecté » : chemin cible dédié
  `FetchLiveIdentity(ctx, targetXUID)` avec xuid explicite (le commentaire périmé
  de career_live_service.go:171-173 induisait en erreur).
- Fonction 188 L du squash : exemption lint justifiée et datée
  (`.golangci.yml:224-232`, K3f — un seul statement DDL).
- `filepath.Join` de cleanup_media_index : préexistant, le diff n'a que
  paramétré le slug.
- `adminRelativeTime` : préexistant à la base (dette ancienne, pas une copie du
  diff).
- Mauvais index des tooltips de bins (moitié du candidat Dominance) : le
  formatter ancre sur la série bar / lane ennemie dont dataIndex = bin.

## Zones vérifiées propres

Anti-ART (aucun nouvel UPSERT sur table critique, lectures `_latest` respectées
hors cmd/ documentés), timezone canonique, KDA/KDR (ADR 0006), aucune
comparaison de slug, couleurs (tokens sémantiques), query keys inline, routes
file-based, les ~40 conversions *Recovered du sweep sont conformes (hors trous
listés), i18n globalement (hors HomeRecentPlaylistsCard), contrat briefing
domain/openapi/generated/front aligné.

## Localisation des findings vs déploiement

L'essentiel des findings lourds (n°1-6, 8-10) vit dans du code DÉJÀ déployé
(trains 07-10 → 07-16). Le train local non poussé (briefing V2 + durcissement
média + correctifs CI) ne porte que des findings mineurs (duplications logic.ts,
lightbox PLAUSIBLE, framework fallback noms). Rien de bloquant pour le push en
attente ; les correctifs des findings majeurs sont à planifier comme chantier
séparé.
