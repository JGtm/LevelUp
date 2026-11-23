# PLAN — Quatre entrées du backlog (2026-11-15)

Statut : **en exécution depuis le 2026-11-15** (go du user : « c'est à faire pour la 8.2.0 »). Lots 1, 2,
4 et mesure 3.1 lancés ; 3.2 et suivants attendent la mesure et la décision du user sur la montée du décodeur
de la phase C dans la 8.2.0.
Contrat d'exécution : skill `plan-execution` (ordre strict par lot, aucune case vide à la clôture, zéro fix
hors périmètre, découvertes en section 7). Grille relue : skill `plan-review`.

Les constats ci-dessous ont été relus sur pièces le 2026-11-15 sur la tête de `release/v8.2.0`
(`e114511dcd`, worktree `LevelUp-wt-c820-infobulles`) et sur la tête du dépôt du décodeur
(`558af5f`, worktree `hifd-c2`). L'exécutant les RE-VÉRIFIE avant de coder (le code a pu bouger).

---

## 0. Objectif et critère de succès

| Lot | Ce que le user verra | Critère de succès |
|---|---|---|
| 1 | Rien (valeurs identiques) ; une seule façon de calculer une cadence par minute | Toutes les divisions « par minute » du périmètre D-1 passent par `analysis.PerMinute`, témoins avant/après égaux, garde-rail qui refuse une nouvelle division en ligne |
| 2 | La connexion par le navigateur dit « services Halo injoignables, réessayer plus tard » OU « Halo refuse ce compte » | Les deux chemins navigateur (site hébergé, application de bureau) posent un code distinct pour une panne des services ; le texte d'un refus ne dit plus « réessaie » |
| 3 | Les vraies morts sans cause ont leur croix dans le rejeu ; les fins de manche n'en ont pas | Croix posée seulement quand le film prouve la mort (compte à rebours de réapparition du joueur) ; accord mesuré contre les morts de la feuille de match |
| 4 | « Seul » et « Masquer » s'appliquent aussi aux effets de mort (selon D-4) | Règle D-4 câblée, tests de la règle verts, commentaire de `shownSlotPredicate` aligné |

Base : **`release/v8.2.0`** (user, 2026-11-15 : « c'est à faire pour la 8.2.0 »), tête `e114511dcd`.
Jamais `main`.

Lots 1, 2 et 4 indépendants : un worktree et une sous-branche par agent depuis `origin/release/v8.2.0`
(`feat/bl1115-minute` -> `LevelUp-wt-bl-minute`, `feat/bl1115-auth` -> `LevelUp-wt-bl-auth`,
`feat/bl1115-killfx` -> `LevelUp-wt-bl-killfx`). Règles de la release (pilote levelup-dd, 2026-11-15) :
pas de PR, chaque sujet se pousse directement sur `release/v8.2.0` après ses portes, rebasé juste avant
l'envoi (pre-push d'environ 8 min) ; notes de version et changelog FR/EN dans le même envoi
(`TestReleaseNotesDocs_*` verts) ; une ligne par contrôle à l'écran dans l'artefact des vérifications
(collection `verifs`, format des lignes `c820-*`) ; SHA transmis à levelup-dd à la fin de chaque sujet
(relecture fraîche avant `main`, deux relecteurs pour l'auth). Lot 1 : prévenir levelup-0b (étape 7.5, note
de performance, `analysis/perf`) avant de pousser. Lot 3 : dépend de
la montée du décodeur de la phase C (section 4) ; la livrer en 8.2.0 = monter la phase C et régénérer le parc
une seconde fois, décision du user.

## 1. Décisions à trancher AVANT exécution

- **D-1 (lot 1) — périmètre.** L'entrée du backlog cite six sites ; la relecture en trouve **quatorze** côté Go :
  - `internal/analysis` (8) : `kpi_stats.go:115`, `explorer_target_stats.go:122`,
    `home_canonical_highlights_tiles.go:86`, `squad_breakdown.go:373`, `squad_breakdown_canonical.go:59`
    et `:160` (les six du backlog), plus `squad_profiles.go:52` et `:105` (forme `kills * 60 / secondes`,
    dont l'arrondi diffère au dernier bit, que le commentaire de `PerMinute` cite justement) ;
  - hors `analysis` (6) : `service/stats_service.go:230`, `service/timeseries_service_buckets.go:58`,
    `service/timeseries_service_tabs.go:124`, `service/teammates/teammates_squad_charts_synergy.go:72`,
    `service/squad_service_v2_contributions.go:63` (somme des minutes par match, Σ(s/60) au lieu de Σs/60),
    `sync/performance.go:98` (note de performance).
  - Hors périmètre dans tous les cas : SQL (`platform/duckdb/world_player_stats_repo.go:195`), mise en forme
    de durées (`notify/embeds.go`), débits (`haloadapter/provider.go:57`, `cmd/backfill-csr-history`).
  - **Tranché (user, 2026-11-15, « corriger ») : les 14**, garde-rail sur les packages de `internal/` qui les
    portent (`analysis`, `service`, `sync`).
  - Attention `sync/performance.go` : il nourrit la note de performance ; la forme y est déjà
    `x / (secondes / 60)`, identique à `PerMinute` bit pour bit, donc aucune révision ADR 0047. Si un
    témoin y bouge, arrêter et revenir au user.
- **D-2 (lot 2) — textes.** Reprendre les textes déjà validés du code d'appareil :
  injoignable = « Les services Halo ne répondent pas pour le moment. Réessayez plus tard. » ; refus =
  « Halo a refusé l'accès à ce compte. Vérifiez le compte choisi et son profil Xbox. » (FR et EN existent dans
  `common.toml`). **Tranché (user, 2026-11-15) : oui, mêmes textes.**
- **D-3 (lot 3) — mesure puis régénération.** (a) go pour une mesure en lecture seule, film par film (pas de
  cuisson en lot en mémoire), sur les films à plusieurs manches et un échantillon d'une manche ; (b) la
  régénération du parc se fait UNE fois, groupée avec celle qu'exige la montée du décodeur de la phase C,
  jamais pour ce lot seul. **Tranché (user, 2026-11-15) : oui aux deux, « mais à voir comme Den adresse ce
  sujet, si jamais ça peut aider ».** Lu le jour même dans la copie locale de libfilmshell (section 4).
- **D-4 (lot 4) — la règle.** Le commentaire actuel (`model/playerPanelLogic.ts:111-116`) dit que les effets
  de mort, objectifs, véhicules et objets posés sont « des événements du match » que ni Seul ni Masquer ne
  retirent. **Tranché (user, 2026-11-15)** : « dans le replay on a des calques et des infos qu'on affiche pour
  tout le monde, quand on clique sur "seul" on affiche ceux-ci que pour ce joueur, et "masquer" masque juste ce
  joueur-là. Peu importe quel calque ou info est affichée. » Règle générale : TOUT calque affiché suit Seul et
  Masquer pour ce qui appartient à un joueur (effet de mort où il tue ou meurt, objet qu'il a posé, véhicule
  qu'il occupe, objectif qu'il porte). Ce qui n'appartient à aucun joueur (véhicule vide, drapeau à sa base,
  zone, socle d'arme) reste affiché : c'est le décor du match. Confirmé par le user le 2026-11-15 (« ce qui
  n'appartient à personne on laisse »).

## 2. Lot 1 — cadences par minute par le helper canonique (Sonnet, effort S)

Constat : `analysis.PerMinute(total, seconds)` (`internal/analysis/indicators.go:146`) existe ; 8 utilisations
seulement ; les sites de D-1 refont la division.

- [x] 1.1 Rouvrir chaque site du périmètre D-1, confirmer la forme (`x / (s/60)`, `x * 60 / s`, Σ(s/60)) et
      la garde de durée nulle (`PerMinute` rend 0 ; certains sites laissent un pointeur nil ou sautent :
      conserver le comportement du site, ne pas le remplacer par 0).
- [x] 1.2 Témoins AVANT : pour chaque site, un test de caractérisation sur des entrées fixes (dont une durée
      non multiple de 60 et une durée nulle), écrit et vert sur le code d'avant ; sortie capturée dans
      `.ai/annexes_backlog_1115/temoins_minute_avant.txt` (`go test -run <noms> -v`).
- [x] 1.3 Migration site par site vers `analysis.PerMinute` (un package de service importe `analysis`, déjà le
      cas pour les autres KPI dérivés ; vérifier le lint de couches).
- [x] 1.4 Témoins APRÈS : mêmes tests, sortie capturée (`..._apres.txt`), diff vide. Seuls écarts admis :
      `squad_profiles.go` et `squad_service_v2_contributions.go`, au dernier bit ; ils s'écrivent avec une
      tolérance 1e-12 ET l'arrondi affiché (2 décimales) doit rester identique sur les entrées du test.
      Tout autre écart = arrêt.
- [x] 1.5 Garde-rail `internal/archlint/no_inline_per_minute_test.go` : AST, refuse un `BinaryExpr` `/` ou `*`
      dont un opérande est le littéral `60` / `60.0` dans les packages de D-1, sauf `indicators.go` et une
      allowlist datée (vide au départ). Test rouge prouvé en réintroduisant une division (puis retiré).
- [x] 1.6 Entrée du backlog : déplacée en « Fait » avec les chiffres (Haiku, contrôlé par le pilote).

Porte 1 :
```
cd apps/go-api && go test ./internal/analysis/... ./internal/service/... ./internal/sync/... ./internal/archlint/...
make go-api-lint
```

## 3. Lot 2 — connexion par le navigateur : panne des services distincte d'un refus (Sonnet, effort S)

Constat : le code d'appareil distingue les deux cas depuis la 8.2.0 (`handlers/auth.go:295-298`,
`auth.HaloServicesUnreachable`, code `halo_unreachable`). Les deux chemins navigateur posent un seul code
`halo_exchange_failed` quelle que soit la cause : site hébergé `handlers/auth_xbox_oauth.go:318`
(redirection `/login?oauth_error=`), bureau `handlers/auth_browser_login.go:228` (tentative `Fail` + page de
retour). Côté web, `features/auth/loginError.ts:111` et `:146` envoient ce code sur
`common.auth.oauth_exchange_failed` = « Microsoft ou Xbox n'a pas répondu. Réessaie dans un instant. » : un
compte refusé reçoit le conseil de réessayer.

- [x] 2.1 Helper unique dans `handlers` : `haloExchangeFailureCode(err error, refused string) string` (rend
      `errCodeHaloUnreachable` si `HaloServicesUnreachable(err)`, sinon `refused`). Trois appelants :
      `auth.go` (device flow, `refused` = `halo_exchange_error`), `auth_xbox_oauth.go:318` et
      `auth_browser_login.go:228` (`refused` = `halo_exchange_failed`). Les codes de refus existants ne changent
      pas (contrat déjà servi).
- [x] 2.2 Garde-rail (règle 6, troisième copie) : `HaloServicesUnreachable(` appelé seulement depuis le helper,
      dans `internal/api` (test AST ou grep dans `archlint`).
- [x] 2.3 Journal : le `slog.ErrorContext` existant reste AVANT le choix du code ; ajouter le code choisi en
      attribut (`"error_code", code`).
- [x] 2.4 Web : `OAUTH_ERROR_KEYS` et `DEVICE_FLOW_FAILURE_KEYS` gagnent `halo_unreachable` ;
      `halo_exchange_failed` passe sur le texte de refus (D-2). Clés dans `lib/i18n/manifests/common.toml`, FR
      et EN (toutes les langues de `config/locales.toml`), puis génération des manifests. Aucune chaîne en dur.
- [x] 2.5 Page de retour du bureau (`renderBrowserLoginPage`) : inchangée (texte générique + code) ;
      l'application affiche le message par le sondage de la tentative. Le vérifier par un test, ne pas
      enrichir la page.
- [x] 2.6 Tests Go : `auth_browser_login_test.go` et `auth_xbox_oauth_refusal_test.go`, quatre causes chacun
      (délai dépassé, erreur réseau, 503 -> `halo_unreachable` ; 403 -> code de refus), sur le modèle de
      `TestPollDeviceFlow_ExchangeFailureCode`. Tests web : `loginError.test.ts`, `LoginPage.test.tsx` (FR et EN).
- [~] 2.7 Revue adversariale (lot auth) : UN relecteur Sonnet, une ronde, skill `adversarial-review`. Couvert par la relecture de la session pilote de la release (deux relecteurs pour l'auth, avant `main`, règle du 2026-11-15).

Porte 2 :
```
cd apps/go-api && go test ./internal/api/handlers/ ./internal/platform/auth/ ./internal/archlint/ ./internal/i18n/...
make check-types && cd apps/web && npx vitest run src/features/auth src/features/setup src/lib/i18n
```

## 4. Lot 3 — croix des vraies morts sans cause (Opus pour 3.1-3.2, Sonnet pour 3.3-3.5, effort M)

Constat :
- Le fil fabrique une mort neutre pour chaque vie close qu'aucun frag ne revendique
  (`model/killFeedLogic.ts:308-312`) ; `deathFacts` (`model/livesMarks.ts:225-249`) ne pose une croix que si le
  film en établit le type (`kind`), parce qu'une fin de manche ferme aussi les vies, sans mort.
- Les vraies morts sans type viennent de deux sources (`apps/go-api/internal/replaybuild/replaybuild.go`) :
  refus de l'attribution ligne par ligne (`:436-440`, BTB ou alerte de santé du décodeur : toutes les chutes
  et suicides du match perdent leur type) et source de dégât sans pictogramme (`:449-452`).
- Nouveau depuis l'entrée du backlog : la tête du décodeur publie `respawnCountdowns[]`
  (`hifd-c2/replay/document_horloges.go:51`, schéma 118 et suivants, pas encore dans un tag monté par
  LevelUp) : par joueur du film, les périodes où son compte à rebours de réapparition court, lues dans le
  film. Un compte à rebours suit une mort ; une fin de manche replace les joueurs sans mort. C'est une
  lecture, pas une fenêtre de temps : la piste de l'entrée du backlog (publier « fin de manche » par vie)
  devient probablement inutile. À prouver par 3.1.
- Méthode de Den (libfilmshell, copie locale `<poste>\Downloads\Scripts\libfilmshell`, lue le
  2026-11-15, son code = témoin, tables reprises avec crédit) : `src/motion.rs:135-141` et `src/lives.rs:698-701`.
  Une vie se termine par une mort quand son joueur **lâche la commande du corps** (`unit-control`) ; une vie
  qu'une nouvelle manche termine est **supprimée** sans lâcher de commande ; la fin du match lâche toutes les
  vies après leur dernier échantillon. Contre-épreuve : la mort comptée sur la ligne du joueur au tableau des
  scores, par manche (`src/assemble.rs:361-375`). Nuance qu'il nomme (`src/model/event.rs:248-256`) : les
  morts survenues entre la décision d'une manche et le début de la suivante, et celles que la fin du match
  inflige aux vivants, sont de vraies morts que le tableau ne compte pas. Chez nous, la façade lit déjà
  `unit-control` (`decfilm/presence.go:21`, `Controller` = -1 sans joueur aux commandes), pas encore
  rapproché des fins de vie.
- Deux lectures, donc : (S1) lâcher de commande avant la fin de la vie (Den) ; (S2) compte à rebours de
  réapparition du joueur qui commence à la fermeture. **S1 est la voie visée** (user, 2026-11-15 : « la voie
  de Den est beaucoup plus propre ») : elle lit la fin de la vie elle-même, à l'instant, là où S2 n'en lit
  qu'une conséquence (et la mort de fin de match n'écrit que le début de son compte). S2 sert de contre-épreuve
  dans 3.1. Si la mesure contredit S1, arrêt et retour au user ; aucune fenêtre de temps sur le score.

- [x] 3.1 (Opus) Mesure, lecture seule, après le go D-3(a), film par film avec le harnais de recherche du
      décodeur (jamais un lot en mémoire) sur les documents produits par la tête du décodeur :
      (a) les films à plusieurs manches du parc (51 selon la phase C) ; (b) 30 films d'une manche dont les
      films BTB à attribution refusée. Pour chaque vie close sans frag : (S1) la commande du corps est-elle
      lâchée avant sa fin, ou le corps supprimé encore commandé ? (S2) un compte à rebours du même joueur
      commence-t-il à sa fermeture (écart en frames) ? Classer chaque vie : mort prouvée par S1 et S2, par une
      seule, par aucune. Oracle : morts de la feuille de match par joueur (`match_participants.deaths`, copie
      à froid) = frags subis + morts typées + morts sans type prouvées, en retirant les morts que le tableau
      ne compte pas (nuance de Den ci-dessus) ; contre-épreuve sur libfilmshell pour 10 films (ses morts non
      revendiquées par un frag, `Kill::inferred`). Sortie : `.ai/annexes_backlog_1115/mesure_morts_sans_type.md`
      (comptes par signal, écarts, cas contraires nommés). Critère pour continuer : aucune fin de manche
      classée mort par le signal retenu ; accord avec la feuille au moins égal à celui des morts typées.
      Sinon arrêt et retour au user, aucune heuristique.
- [x] 3.2 (Opus) Règle écrite à partir de 3.1 : signal retenu (S1, S2, ou S1 confirmé par S2), et où il se
      publie. Si S1 l'emporte : il faut une publication par vie dans le décodeur (« fin par lâcher de
      commande »), donc une étape du dépôt du décodeur, dans sa phase C, par sa session, avec crédit à Den
      dans le code et la note de version ; LevelUp ne fait que la lire. Si S2 suffit : rien à ajouter au
      décodeur. Dans les deux cas : tolérance d'appariement tirée de la mesure (constante nommée avec sa
      source), morts de fin de match, joueur sans `filmIndex` (bot ou roster incomplet : pas de croix, compté).
- [x] 3.3 (Sonnet) Dépendance : LevelUp monte un tag du décodeur qui publie le signal retenu en 3.2 (la montée
      de la phase C, faite par sa propre session ; ce lot ne pose ni tag ni `require`). Vérifier auprès des
      sessions paires (pilote, décodeur) avant de commencer.
- [x] 3.4 (Sonnet) Web, fonction pure : `deathFacts` pose la croix d'une mort sans type quand la règle 3.2
      la prouve (passage roster `filmIndex` -> xuid), sinon la compte dans `unplacedDeaths` comme aujourd'hui.
      Document sans `respawnCountdowns` (ancien schéma) : comportement d'aujourd'hui, inchangé. Commentaires
      de `deathFacts` et de `killFeedLogic` réécrits (contrat, pas l'histoire). Tests vitest : fin de manche
      sans compte (pas de croix), mort sans type avec compte (croix), ancien document, bot.
- [x] 3.5 (Sonnet) Notes de version de la release qui porte la montée : la phrase de la 8.2.0 « les morts sans
      type ne sont pas marquées » est remplacée. Vérification à l'écran ajoutée à l'artefact des vérifications
      du user (un BTB à attribution refusée, un Oddball à trois manches), pas dans un message.
- [x] 3.6 Régénération : UNE seule, celle de la montée de la phase C (D-3(b)), par la commande prévue,
      serveur arrêté. Si cette montée est déjà régénérée sans ce lot : demander au user, ne pas relancer seul.

Porte 3 :
```
cd apps/web && npx vitest run src/features/match-replay
make check-types
make replay-equiv     # seuls les champs attendus changent
```

## 5. Lot 4 — effets de mort sous « Seul » et « Masquer » (Sonnet, effort S, après D-4)

Constat : `drawKillFxLayer` (`layers/replayDraw.ts:211`) ne reçoit pas de filtre ; la liaison `morts` de
`ui/ReplayCanvas.tsx:594` ne passe pas `shownAtSlot` ; `KillFxEntry` (`model/killFx.ts:46-71`) porte le slot
du tueur (`slot`) mais ni le slot ni l'image de la vie de la victime.

- [x] 4.0 Inventaire fermé des calques de `layers/replayCompose.ts` : pour chacun, branché ou non sur
      `shownAtSlot`, et ce qui y appartient à un joueur (champ du document qui le prouve : slot, xuid, poseur,
      occupant, porteur). Tableau dans `.ai/annexes_backlog_1115/calques_seul_masquer.md`. Chaque calque qui
      porte un élément attribuable et n'est pas branché devient une case 4.x ci-dessous ; un élément sans
      attribution lisible reste affiché (pas d'attribution devinée), compté dans le tableau.
- [x] 4.1 `buildKillFx` : ajouter `victimSlot` et `victimFrame` (dernière image de la vie de la victime,
      celle que les croix des traces utilisent déjà), `null` si introuvable.
- [x] 4.2 `playerPanelLogic.ts` : fonction pure `killFxShown(focus)` qui rend `null` sans filtre, sinon un
      prédicat sur l'effet. Règle D-4 : Seul = le joueur de la fiche figure dans l'effet (tueur ou victime) ;
      Masquer = il n'y figure pas ; partie inconnue (`null`) : ne prouve rien (Seul : l'autre partie décide ;
      Masquer : l'effet reste si la partie connue n'est pas le joueur masqué).
- [x] 4.3 `drawKillFxLayer` reçoit le prédicat dans son style ; `ReplayCanvas` le passe. Aucune logique dans le
      composant.
- [x] 4.3 bis Les autres calques relevés en 4.0 (objets posés par leur poseur, véhicules par leurs occupants,
      objectifs par leur porteur), une case par calque ajoutée à l'issue de 4.0, même prédicat.
- [x] 4.4 Commentaire de `shownSlotPredicate` (`playerPanelLogic.ts:111-116`) réécrit sur la règle générale.
- [x] 4.5 Tests vitest : prédicat (Seul tueur, Seul victime, Masquer tueur, Masquer victime, parties
      inconnues), couche (un effet filtré n'est pas dessiné).
- [x] 4.6 Vérification à l'écran ajoutée à l'artefact des vérifications du user (en lecture, pas à l'arrêt).

Porte 4 :
```
cd apps/web && npx vitest run src/features/match-replay
make check-types
```

## 6. Clôture

- [ ] 6.1 Les lots 1, 2, 4 réunis sur `feat/backlog-2026-11-15` ; skill `delivery-checklist` ; `make gate-push`
      avant la PR vers la release suivante ; demander avant chaque commit.
- [ ] 6.2 `.ai/thought_log.md` : une entrée par lot clos.
- [ ] 6.3 `.ai/BACKLOG.md` : les quatre entrées en « Fait » ou statuées ; l'entrée « morts sans type » renvoie
      au lot 3 tant que la montée n'est pas faite.
- [ ] 6.4 `.ai/REGISTRE_REPORTS.md` : ce qui reste reporté (lot 3 si la montée tarde), avec condition de reprise.

Statuts : `[x]` fait, `[~]` couvert ailleurs (référence), `[!]` non traité (justification). Aucune case vide à
la clôture. Une étape est close quand sa porte est verte ET ses cases statuées.

Reprise de session : lire ce fichier (cases), puis `git log --oneline -10` sur la branche du lot, puis
`ListAgents` (un autre agent peut tenir le worktree).

## 7. Découvertes (à consigner, ne pas traiter)

- 2026-11-15 (écriture du plan) : `code_exchange_failed` (échange du code Microsoft) a le même défaut que
  `halo_exchange_failed` : un seul texte « n'a pas répondu, réessaie » pour une panne et un refus
  (`invalid_grant`, code réutilisé). Hors périmètre du lot 2.
- 2026-11-15 (écriture du plan) : deux codes différents pour le même refus Halo selon le chemin
  (`halo_exchange_error` au code d'appareil, `halo_exchange_failed` au navigateur). Unifier = changer un
  contrat servi ; hors périmètre.

- 2026-11-16 (3.4) : les huit fixtures de contrat web au schéma 124 (`test/fixtures/go/replay_schema_124_*`) ne
  portent ni `endReason` / `releaseFrame` ni `respawnCountdowns` ni `roundClock` : elles sont cuites sans la marche
  des trames qui lit les commandes d'unité et les suppressions (en production, `film_scan_mouvement` du décodeur
  les remplit). Aucun test de comptage des croix sur un vrai film n'est donc possible dans le dépôt ; un test figé
  à 0 serait vert même sans la règle. Recuire ces fixtures avec la marche des trames sort du lot ; la preuve sur
  film reste la vérification à l'écran après la régénération.

## 8. Répartition des modèles

| Étape | Modèle | Pourquoi |
|---|---|---|
| Lots 1, 2, 4 (code et tests) | Sonnet | Changements bornés, constats déjà localisés |
| 1.6, 6.3, 6.4 (lignes de backlog et registre) | Haiku | Micro-tâches, contrôlées par le pilote |
| 2.7 (revue du lot auth) | Sonnet | Une ronde, un relecteur |
| 3.1, 3.2 (mesure de S1 et S2, règle) | Opus | Lecture du film, deux signaux et oracle croisé ; erreur = croix fausses sur tout le parc |
| Publication de S1 dans le décodeur (si 3.2 la retient) | Sonnet, session du décodeur | Câblage d'une lecture déjà faite par la façade |
| 3.3 à 3.5 | Sonnet | Câblage d'une règle écrite |

## 9. Journal

- 2026-11-15 (lot 1, `feat/bl1115-minute`, agent Sonnet) : 1.1-1.5 `[x]`. Quatorze sites migrés ; `sync/performance.go` portait six divisions par `minutes` (forme déjà `x / (s/60)`), toutes migrées, valeurs identiques. Témoins avant/après identiques au bit (aucun écart, même pour `squad_profiles.go` et `squad_service_v2_contributions.go`), puis tests de caractérisation passés en assertions exactes (rougissent si `PerMinute` est faussé). Garde-rail `archlint/no_inline_per_minute_test.go` (`TestCadenceParMinute*`) restreint à trois formes de cadence (`/ 60`, `x*60 / s`, `(a/b) * 60`) : la règle littérale donnait 9 faux positifs (durées, `60*time.Second`, découpe mm:ss) ; couple `n / 60` + `n % 60` exempté structurellement ; allowlist vide ; rouge prouvé sur l'état d'avant (14 sites listés) et sur une division réintroduite. Seule différence théorique : `timeseries_service_buckets.go` garde `== 0`, une durée négative donnerait 0 (donnée impossible). Changelog FR/EN (« Modifié » de 8.2.0), notes de version non touchées (rien ne change à l'écran). 1.6 : entrée du backlog retirée. Portes : `go test` des packages touchés et `archlint` verts (timeout 40 min, machine chargée), `make go-api-lint` 0 issue.
- Découvertes du lot 1 (non traitées) : `archlint` et `go-api-lint` dépassent leurs délais par défaut quand plusieurs sessions compilent ; directives `//nolint` mal formées signalées par golangci-lint (« unknown linters 2026-09-07), gosec… »).
- 2026-11-15 (lot 2, `feat/bl1115-auth`, agent Sonnet) : 2.1-2.6 `[x]`, 2.7 `[~]`. Helper `haloExchangeFailureCode` (`handlers/auth_halo_exchange_code.go`), trois appelants (code d'appareil, site hébergé, bureau), codes de refus inchangés ; garde-rail AST `archlint/halo_exchange_failure_code_unique_test.go` (rouge prouvé) ; `error_code` au journal des trois chemins ; web : `halo_exchange_failed` sur le texte de refus du code d'appareil, `halo_unreachable` ajouté aux refus du site, clés existantes réutilisées (aucune clé neuve) ; page de retour du bureau inchangée, tenue par un test ; tests Go quatre causes par chemin, point d'injection `WithCodeExchange` (défaut inchangé) ; tests web FR et EN. Notes de version et changelog FR/EN (8.2.0, « Corrigé »). Portes vertes (`archlint` et lint relancés avec délai large, machine chargée).
- Découverte du lot 2 (non traitée) : `TestXboxOAuth_Callback_StateMatches_AttemptsExchange` appelle encore le vrai échange Microsoft avec un code factice ; il pourrait passer par `WithCodeExchange`.
- 2026-11-15 (lot 4, `feat/bl1115-killfx`, agent Sonnet) : 4.0-4.6 `[x]`. Inventaire des 31 calques : 9 suivaient le filtre, 14 portaient un élément attribuable non filtré (morts, chaleur, armes au sol, poses, véhicules, failles, drapeaux, couronne, crâne, bombe portée, déflagration, pulses, croix de mort de la piste, câble du grappin et poussée du répulseur), 8 sans attribution lisible restent affichés. Prédicat unique `ownershipShown` (`playerPanelLogic.ts`), adaptateurs par genre (`model/focusFilter.ts`), garde-rail de câblage de `ReplayCanvas`. Effet de mort : tueur ET victime (`victimSlot`, `victimFrame`, plus `slotFrame`, `killerXuid`, `victimXuid` pour le repli des bots) ; poussée : auteur ET poussé ; Masquer ne retire un véhicule partagé que si le joueur masqué est seul à bord. Test qui figeait « une mort est un événement du match » réécrit (D-4). Notes de version et changelog FR/EN. Porte : 4 388 tests `match-replay` verts, `check-types` vert, lint 0 erreur. Vérification à l'écran : ligne `bl1115-01-seul-masquer` de l'artefact des vérifications.
- Découvertes du lot 4 (non traitées) : un effet de mort sans aucune partie lue reste affiché sous Seul comme sous Masquer (règle « attribution illisible reste ») ; la croix de mort de la piste ne suit que sa victime ; xuid du document comparé à la clé d'un bot porteur d'objectif non vérifié sur un vrai film ; `ReplayCanvas.tsx` à 806 lignes (dette existante).
- 2026-11-15 (3.1, agent Opus, lecture seule) : `[x]`. 99 films décodés un par un (54 à plusieurs manches, 16 d'une manche, 29 BTB dont 5 à attribution refusée), pic mémoire 2,26 Go. S1 (lâcher de commande, tolérance [-2, 6] frames = écart maximal observé sur 14 671 morts revendiquées) : 0 des 653 fins de manche classée mort, 0 des 21 lâchers de fin de match. Accord avec la feuille (morts non comptées retirées) : morts typées 89,1 / 98,5 / 88,1 / 57,0 % (plusieurs manches / une manche / BTB / BTB refusés), S1 99,6 / 100 / 97,9 / 91,6 %, S1 ou S2 99,8 / 100 / 99,3 %. Contre-épreuve libfilmshell sur 68 films. Livrable hors dépôt `C:\t\bl1115-mesure\mesure_morts_sans_type.md` (copie sans xuid à verser en annexe). Décision du user (2026-11-15) : livraison dans la 8.2.0, règle « S1 ou S2 ». Coordination : publication par piste (`endReason`, `releaseFrame`, schéma 123) relue et fusionnée par la session de la phase C du décodeur avant le tag v1.3.9, montée et miroir par la session de la montée, une seule régénération après la montée ; la part web de ce plan (3.4) se pousse après la montée.
- 2026-11-15 (lot 1, ronde 2 après relecture indépendante de la session pilote) : « valeurs servies identiques » ne tenait pas sur trois points, corrigés à témoins rouges avant, verts après : profils d'Escouade (`k * 60 / s` devenu deux arrondis : 0,22 au lieu de 0,23 pour 3 frags en 800 s) -> helper `analysis.PerMinuteOfCount`, 0 écart sur 109 800 cas ; contributions par minute (Σ(s/60) devenu Σs/60, dernier bit) -> `analysis.SecondsToMinutes` ; tranches de score par minute : durée <= 0 exclue explicitement (seule différence de comportement, dite au changelog). Garde-rail étendu aux millisecondes (60 000) : trois sites de plus (`squademprise/production.go`, `temporal/engagement_curve.go`, `temporal/engagement_score.go`) migrés par `analysis.MillisecondsToMinutes`, témoins identiques. Total : 17 sites. Changelog réécrit à la réalité. Portes rejouées après l'arrêt intempestif des `go.exe` de 12 h 40.
- Découverte (relecture du lot 2, non traitée) : le champ `err` des trois journaux d'échec de l'échange Halo (antérieur au lot) contient le corps de la réponse Xbox (`xblHTTPError.Body`, `XErr` / `Message`), sans jeton.
- 2026-11-15 (lot 4, ronde 2 après relecture indépendante de la session pilote, 18 conditions tenues) : garde-rail de câblage lu dans les bornes de chaque appel (16 retraits prouvés rouges) ; test du filtre des armes au sol (rouge si supprimé ou inversé) ; révélation du capteur filtrée avant le dédoublonnage ; contrat de `PlacementScene.shown` réécrit selon le code ; bombe d'un porteur masqué retirée aussi quand elle se dessine libre ; règle des frags (validée par le user via la session pilote : tueur OU victime partout, sauf la croix de mort, victime seule) écrite comme contrat de `ownershipShown` et dans les notes. Constat 6 (propriétaire inconnu traité de trois façons) : en attente de la décision du user. Porte : 4 397 tests `match-replay` verts.
- 2026-11-15 (3.2, agent Opus) : `[x]` règle écrite (`.ai/annexes_backlog_1115/regle_3_2.md`, mesure en `mesure_morts_sans_type.md`, sans xuid) : croix d'une vie close sans frag si lâcher de commande (`endReason == released`, `releaseFrame` dans [-2, +6] frames de la fin de piste) OU début d'un compte à rebours du même joueur dans [-2, +6] frames ; sur l'échantillon, 443 croix nouvelles, 0 fin de manche. Publication décodeur (3.3) en cours (`feat/fin-de-vie-par-commande`, schéma 123), « fenêtre de naissance de 5 s » de la spec remplacée par une borne lue.
- 2026-11-15 (lot 4, ronde 3, constat 6) : décision du user (relayée par la session pilote) « B » : un élément dont le propriétaire est inconnu reste toujours affiché, sous Seul comme sous Masquer, comme le décor. Trois lectures alignées, test rouge avant / vert après chacune : projectiles sans auteur, marques des Vies sans slot, acteur sans slot de la mise en valeur d'une élimination (troisième cas trouvé par l'exécutant). Contrat de `ownershipShown`, notes et changelog FR/EN, inventaire à jour ; en-tête de `layers/authoredShown.test.ts` corrigé. Porte : 4 397 tests `match-replay` verts. Lot 4 clos.
- 2026-11-15 (3.3) : `[x]` schéma 124 publié dans le décodeur v1.3.9 (tag fdd6f24), montée LevelUp par la session de la montée ; ce lot ne pose ni tag ni `require`. Les types web du schéma 124 sont repris provisoirement par deux commits `tmp(bl1115-3.4)` à retirer au rebase sur la montée officielle.
- 2026-11-15 (3.4, `feat/bl1115-croix`, agent Sonnet) : `[x]`. Décision de livraison (coordinateur, 2026-11-15) : la 8.2.0 sort sans le nouveau décodeur, 3.4 et 3.5 partent en 8.2.1. `model/lifeEndProof.ts` (fonction pure `lifeEndProof`, table `countdownStartsByXuid` : `filmIndex` -> roster -> xuid, index porté par plusieurs entrées, bot ou sans xuid = comptes non servis et comptés ; constantes `UNIT_RELEASE_TOL_MS` et `RESPAWN_COUNTDOWN_TOL_MS` = -66 / +200 ms avec leur source) ; `lifeEndsOf` l'appelle, `ReplayDeath.proven` ; `deathFacts` pose la croix si `kind === ''` et `proven`. `startRead` n'est pas filtré (la règle 3.2 lit `from` tel quel). Tests : `lifeEndProof.test.ts` 20, `killFeedLogic.test.ts` +10 (deux chemins d'alignement), `livesMarks.test.ts` +3 ; porte `npx vitest run src/features/match-replay` : 326 fichiers passés, 4487 tests passés (4 fichiers et 7 tests de mesure déjà ignorés hors environnement : `*.mesure.test.ts`, `ReplayTeams.perf`), `npm run typecheck` (`tsc -b`) vert, eslint 0 erreur, knip-ratchet 0/0/0, `lint:colors` propre. Aucune fixture de contrat schéma 124 sous `test/fixtures/go/` (elles sont au schéma 116) : pas de test de comptage sur film réel, à faire à la régénération (3.6).
- 2026-11-15 (3.5) : `[x]` section 8.2.1 ajoutée en tête de `docs/CHANGELOG.md`, `docs/FR/CHANGELOG.md`, `docs/RELEASE_NOTES.md` et `docs/FR/RELEASE_NOTES.md` (la phrase 8.2.0 « une mort sans cause connue n'est pas marquée » reste juste pour la 8.2.0 et n'est pas touchée) ; date de la 8.2.1 laissée vide (inconnue). La vérification à l'écran (un BTB à attribution refusée, un Oddball à trois manches) dans l'artefact `verifs` du user n'est pas faite par cet agent : à ajouter par la session pilote, après la régénération (3.6).
- Découverte du lot 3 (non traitée) : `replayContract.test.ts` (liste exhaustive des tableaux nullables du contrat) ne compile plus avec les types du schéma 124 (quatre tableaux `medals`, `respawnCountdowns`, `roundClock`, `scoreActions` et le chemin `coverage.score.roundsUnplayed`) ; correction minimale dans le commit `tmp` de la branche, à remplacer par celle de la montée officielle.
- 2026-11-16 (3.6, pilote) : `[x]`. Régénération unique faite avec la montée v1.3.9 (schéma 124, 1 455 rejeux, 0 erreur), en prod avec la 8.2.1. Relecture du code (agent Sonnet, contexte frais) : aucun défaut recevable, 15 conditions tiennent. Mesure sur le parc entier (agent Sonnet, lecture seule, vrai code web) : 0 croix sur 1 216 remises en place de fin de manche, 0 sur 20 lâchers de fin de match ; accord avec la feuille, morts non comptées retirées, 99,3 / 100 / 99,5 % (plusieurs manches / une manche / BTB) sur les films à frags complets, contre 99,8 / 100 / 99,3 % sur l'échantillon. Aucun sous-compte ; l'écart restant est un excès de croix sur des morts que la feuille ne compte pas (morts après la décision finale). Rapport hors dépôt `C:\t\bl1115-mesure\mesure_parc_8_2_1.md`. Reste la vérification à l'écran du user (ligne bl1115-03 de l'artefact des vérifications).
