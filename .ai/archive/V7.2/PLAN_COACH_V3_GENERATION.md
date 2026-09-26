# Plan — Coach V3 : extensions de génération

> **Créé le** : 2026-06-09
> **Statut** : **LIVRÉ à ~95 %** le 2026-06-09 (commits `7cefd911a`, `b0dd5256a`, `b916e1191`,
> `c00b7850f`) — voir « État d'exécution » ci-dessous pour le détail par phase. Ce fichier
> n'avait jamais été mis à jour après livraison (statut affiché « Proposé » à tort — corrigé
> le 2026-07-25, chantier v7.2.1 item V721-07).
> **Priorité** : 🟡 produit (chaque phase = arbitrage produit propre avant branche)
> **Origine backlog** : section `Coach proactif × Prestige` — items Squad coach / Coach négatif soft / Coach tone

## État d'exécution (2026-07-25 — vérifié sur pièces, chantier v7.2.1 item V721-07)

| Phase | Statut | Preuve |
|---|---|---|
| A — Coach négatif soft | `[x]` fait | Détection : `internal/progression/coach/generator.go:326-354` (`LOWESSSoftNegativeThreshold = -0.10`, `buildLOWESSSoftNegativeAlerts`, gate `LOWESSObservationWindow = 14j` défini `coach/types.go:91-93`). Type d'alerte `coach/types.go:40-43` (`AlertTypeLOWESSSoftNegative`). Catégorie neutre `coach/emitter.go:42-45` (`CategoryTrendConsolidate`). Câblé dans le générateur `coach/generator.go:105`. Pont vers signal `coach_advisor/types.go:37-40` (`SignalLOWESSSoftNegative`), traduction `coach_advisor/signals.go:45-46,82-97`. UI `apps/web/src/features/ascension/CoachFocusCard.tsx` (accents `success`/`info`, `FOCUS_THRESHOLD = 0.02`), montée en 1re position dans `AscensionCoachingTab.tsx:66`. `TrendBadge` de `profile/PerformanceSection.tsx` neutralisé (`:138-164`, token `info` au lieu de `outcome-loss`). Tests `coach/generator_test.go:370-391,580-586`, `coach_advisor/signals_test.go:70-71`. ADR 0014 amendée le 2026-07-25 (§ dédiée en tête du fichier) — l'ADR affichait encore « positif uniquement », corrigé (item 07.2 du chantier v7.2.1). |
| B — Coach tone | `[~]` couvert par A | Comme prévu par ce plan (§ Phase B « fusion de fait avec Phase A ») : aucun setting `coach_tone`, aucune matrice de tons — la banque i18n universelle IS la Phase A (`profile.coach.*` + `profile.lusr.*` dans `apps/web/src/lib/i18n/manifests/profile.toml`, catégories notif `internal/notifications/`). Rien à livrer séparément. |
| C — Squad coach | `[x]` fait | Entité `Squad`/`SquadMember` re-keyée xuid : `internal/prestige/types.go:235` (`Squad`), `:253` (`SquadMember`), `:268` (`SquadChallenge`) — *référence corrigée : le plan disait `:246-259`, périmée*. Résolution d'accès membre-user : `internal/prestige/service_squads.go:212-226` (`assertMemberUser`). Règle no-overlap : `internal/prestige/squad_progress.go:38` (`MatchCountsForSquad`). Profil agrégé d'escouade + axe focal : `internal/prestige/squad_coach.go:16-33` (`AggregateSquadAxes`), `:35-52` (`SquadFocusAxis`), `:78-96` (`BiasTemplatesByAxis`). Provider de profil (perf 8 composantes → 6 axes) : `internal/api/wire/prestige_squad_profile.go:45-48` (`SquadAxes`), câblé `internal/api/server_apiv1.go:1063`. Filtre coach sur le pool : `internal/prestige/service_pilot_pool.go:181-239` (`RefreshSquadPool`) — *référence corrigée : le plan disait `:145-195`, périmée*. UI : `apps/web/src/features/squad/SquadFocusStrip.tsx` + `SquadObjectivesPanel.tsx`. **L'affirmation « il n'existe aucun signal coach niveau escouade » (ligne originale de ce plan, § Phase C) est invalidée — retirée ci-dessous.** |
| Cooldown anti-répétition du cap soft-négatif (mentionné § Phase A, densité/fréquence) | `[!]` non traité — décision assumée | Jamais implémenté et **ne le sera pas** : la carte `CoachFocusCard` est déjà bornée à 1 seul axe focal affiché (`focal` = la composante qui bouge le plus, cf. boucle `Math.abs(c.trend) > Math.abs(focal.trend)`) et à un seuil d'amplitude `FOCUS_THRESHOLD = 0.02` — un cooldown ajouterait un état persistant (dernier axe montré + horodatage, par joueur) pour un bénéfice non mesuré (aucune plainte produit sur la répétition observée). Revisiter seulement si un signal utilisateur concret l'exige. |

## Cadre commun

Tout passe par `internal/progression/coach_advisor/` (génération de signaux/proposals) et son pont
vers `internal/prestige/` (challenges/arcs). État actuel vérifié :
- `coach.GenerateInput` est **strictement per-user** (`coach_advisor/service.go:67-78`).
- LOWESS calcule une pente (`profile/lowess.go`) mais n'émet **que** des signaux positifs.
- Les labels/réasons coach sont **génériques** (pas paramétrés par ton).
- Notifications coach **in-app uniquement** (`player_notifications`).

ADR de référence : 0014 (couches progression), 0020 (pont coach→Prestige), 0021 (synthèse dynamique).

---

## Phase A — Coach négatif soft (neutre / soft-négatif, pas pessimiste) — `[x]` LIVRÉ

> Note produit : il s'agit de **neutre à soft-négatif**, jamais culpabilisant. ADR 0014 §6.1 cadrait
> le coach comme strictement positif → cette phase **amende l'ADR** avec garde-fous — amendement
> effectivement rédigé le 2026-07-25 dans `docs/adr/0014-progression-tracking-v2-ascension.md`
> (§ dédiée en tête du fichier).

### Détection (backend)
1. Étendre `profile/lowess.go` : exposer la **pente négative soutenue** (déjà calculable —
   `Slope = last - first`), avec seuils de significativité (durée mini, amplitude mini) pour éviter le
   bruit match-à-match.
2. Nouveau `SignalKind` (ex. `LOWESS_SOFT_NEGATIVE`) dans `coach_advisor/signals.go`, **distinct** du
   positif, avec garde-fous : n'émettre que si tendance soutenue ET au-dessus d'un plancher de
   confiance.
3. Mapping vers une **opportunité** (pas un reproche) : signal → proposal de stabilisation
   (`prestige` arc/challenge orienté « consolider X »).

### Narration UI/UX — mini-spec validée (2026-06-09, option **B « Cap du moment »**)

Maquette retenue après arbitrage : **un seul headline adaptatif** en tête de l'onglet Entraînement
(`AscensionCoachingTab`), qui **absorbe** l'actuel `TrendBadge` de `PerformanceSection` (retiré là-bas).
Pas de nouvelle section « à surveiller » : on consolide au lieu d'ajouter (Ascension déjà dense).

- **Composant** : nouvelle carte `CoachFocusCard` (réutilise `KpiCard` avec `accent` dynamique),
  montée **en première position** de `AscensionCoachingTab`, au-dessus de `CoachProposalsCard`.
- **Contenu** : 1 axe focal (libellé) + état + 1 phrase + CTA `[Voir le défi →]` qui scrolle/ouvre la
  proposition correspondante dans `CoachProposalsCard`. Le coach ne ré-affiche **ni stats ni streaks**
  (déjà dans Profil V3 / Réalisations) — il donne **interprétation + action**.
- **Bascule adaptative** (accent = token sémantique, jamais de hex) :
  - progression → `accent="success"`, registre « tu consolides X, continue ».
  - soft-négatif → `accent="info"` (**neutre, jamais `outcome-loss`/rouge**), registre « X mérite ton
    attention » / « opportunité de stabiliser X » — **jamais** « tu régresses ».
  - état neutre / pas de tendance soutenue → carte **non rendue** (pas de bruit).
- **Recadrage `PerformanceSection`** : le `TrendBadge` « Downtrend » en `outcome-loss` (rouge) est
  **supprimé** ou neutralisé (le focus passe par `CoachFocusCard`). Les `TrendArrow` inline par
  composant peuvent rester (granularité fine), mais la **headline** négative n'est plus rouge.
- **Densité / fréquence** : **1 seul** axe focal affiché à la fois (le plus significatif). Si plusieurs
  signaux, priorité au positif en cas d'égalité de force ; cooldown pour ne pas re-pousser le même axe
  soft-négatif à chaque session.
- **Ton** : universel (cf. Phase B) — banque i18n FR + EN soft / non-culpabilisante.

**Découpe de livraison** : le backend (détection + `SignalKind` soft-négatif) se livre et se teste
indépendamment ; le signal n'est **émis vers l'UI** qu'une fois `CoachFocusCard` + i18n en place.

### Tests
- Détection : séries synthétiques (tendance négative soutenue → signal ; bruit → pas de signal).
- Garde-fou : un signal positif et un soft-négatif ne se contredisent pas sur la même métrique.

---

## Phase B — Coach tone (ton narratif **universel**) — `[~]` couvert par la Phase A

> **Décision produit (2026-06-09, validée user)** : le ton n'est **PAS** un setting joueur. Un seul
> ton **par défaut et universel** : soft, non-culpabilisant, factuel-encourageant. Pas de
> `coach_tone` dans `user_preferences`, pas de sélecteur UI, pas de matrice `× tons`. L'ancienne
> proposition (setting `neutral|technical|motivating|playful`) est **abandonnée**.

### Backend
- **Aucun** champ setting. Le ton vit entièrement dans la banque i18n (contenu), pas dans le code.

### Contenu i18n
- Banque de templates i18n **× `SignalKind`** (FR + EN), rédigée selon une **tone guideline** unique
  et documentée (registre soft, jamais « tu régresses »). C'est le gros de l'effort (contenu).
- Cette banque est **la même** que celle de Phase A : Phase B se réduit à la rédaction de la guideline
  + des templates universels. → **Fusion de fait avec Phase A** ; ne reste plus une phase séparée à
  arbitrer.

### Tests
- Résolution template : (kind, lang) → bon template (plus de dimension `tone`).

---

## Phase C — Squad coach (signal niveau escouade) — `[x]` LIVRÉ (cf. tableau État d'exécution)

> La plus structurelle. Au moment de la rédaction de ce plan (2026-06-09), `SquadChallenge` et
> `RefreshSquadPool` **existaient déjà** (`prestige/types.go:268`, `service_pilot_pool.go:181-239`
> — références corrigées le 2026-07-25, les lignes d'origine avaient bougé) mais **sans filtre
> coach** (sélection aléatoire), et il n'existait **aucun** signal coach niveau escouade.
> **Ces deux lacunes sont comblées depuis le 2026-06-09** : `squad_coach.go` fournit le signal
> (profil agrégé + axe focal) et `RefreshSquadPool` l'utilise pour biaiser le pool
> (`BiasTemplatesByAxis`) au lieu d'un shuffle pur. Section conservée ci-dessous pour l'historique
> de conception ; le detail livré est dans le tableau « État d'exécution » en tête de fichier.

### Backend
1. **Profil agrégé d'escouade** : moyenne LUSR par axe sur les membres (réutiliser le profil per-user
   existant, agréger).
2. **Signal coach escouade** : variante de `coach.GenerateInput` acceptant un contexte squad (ajouter
   `SquadID`/membres au lieu d'un seul `UserID`). Détecter un pattern collectif (orientation
   combat/objectif/support).
3. **Filtre coach sur le pool** : étendre `RefreshSquadPool` pour accepter un filtre dérivé du signal
   escouade (au lieu du shuffle pur) → proposer un `SquadChallenge` calibré sur la composition.

### Tests
- Profil agrégé déterministe sur fixtures multi-membres.
- `RefreshSquadPool` avec filtre coach ≠ shuffle aléatoire (sélection orientée).

### UX — mini-spec validée (2026-06-09, option **1 « Cap d'escouade »** + drawer)

Maquette retenue : un **strip compact `SquadFocusStrip`** en tête de l'onglet **Synergies**
(`SquadSynergiesPage`), **sous** le `SessionBriefing` et **avant** les charts — élément *prospectif*
nettement isolé du contenu *rétrospectif* (charts inchangés, simplement poussés plus bas).

- **Strip (consultation rapide)** :
  - 1 ligne : orientation détectée de l'escouade (combat / objectif / support — dérivée du **profil
    agrégé**, point 1 backend ; pas de ré-affichage du radar) + l'axe en retrait.
  - Défi d'escouade actif OU suggéré : libellé + barre de progression (`▓▓▓▓░░ 3/5`).
  - CTA `[Rejoindre →]` (si suggéré) / `[Voir →]` (si actif) → **ouvre le drawer** ci-dessous.
  - Accent token sémantique (orientation = `info`/neutre ; pas d'alarme).
- **Drawer « Objectifs d'escouade » (définir + consulter)** — destination demandée par le user :
  - **Consulter** : liste des défis actifs via `listSquadChallenges(squadId)` + progression par membre.
  - **Rejoindre** : `joinSquadChallenge(id, { chosen_tier })` (hook `useJoinSquadChallenge` existe déjà).
  - **Définir** : créer un défi via `createSquadChallenge` (le pool filtré-coach de `RefreshSquadPool`
    alimente les suggestions ; le user peut aussi composer manuellement).
  - Toute la plomberie API/hook existe déjà (`apps/web/src/lib/prestige.ts`) — **seule l'UI manque**.
- **Arcs d'escouade** : **hors V1** (V1 = défis uniquement ; on valide l'adoption avant d'investir). Non
  modélisés backend aujourd'hui (seuls les arcs *perso* existent ; `service_arcs_squads.go` n'expose que
  `*SquadChallenge`). **Anticipation à coût quasi nul** (sans coder les arcs) :
  - l'entité `Squad` / `SquadMember` **existe déjà** (cf. ci-dessous) et est **générique** — pas liée aux
    défis → réutilisable telle quelle pour des arcs d'escouade ;
  - la **résolution d'accès** (`xuid`→user, droits member-user) = **helper partagé**, pas enfoui dans le
    code défis ;
  - le drawer prévoit structurellement un **2ᵉ onglet « Arcs »** (vide en V1, aucune restructuration le
    jour venu).
- **Escalade éventuelle** : si le drawer devient trop riche (historique défis + récompenses + classement
  d'escouade), basculer alors vers l'option 2 (3ᵉ onglet « Objectifs ») — **pas avant**.

### Identité d'escouade — modèle validé (2026-06-09)

Décision : **escouade = entité partagée, explicite, multiple, accès auto pour les membres-users, sans
consentement.** (Pas de `squadID` déterministe-hash ; pas de roster privé au owner.)

- **Entité déjà existante (corrigé 2026-06-09)** : `Squad {ID, Name, CreatedBy, CreatedAt}` +
  `SquadMember {SquadID, UserID, JoinedAt}` + `SquadRepo` (`Create / Get / AddMember / RemoveMember /
  ListMembers / ListSquadsForUser`) **existent déjà** dans `shared_social.duckdb` (`prestige/types.go:229`,
  `repository.go:149`). → **PAS de table à créer.** `ListSquadsForUser(userID)` assure déjà la convergence
  des co-membres. (Correction : l'affirmation antérieure « squad_id = string libre sans roster » était
  fausse.)
- **Vrai delta backend = re-keyer `SquadMember` en `xuid`** : aujourd'hui membre = `UserID` only →
  (a) impossible d'inclure un ami **pas sur l'app**, (b) incompatible avec la mesure sur
  `shared.match_participants` (clé `xuid`). Extension : **`xuid` obligatoire** (tout membre, app ou non) +
  **`userID` optionnel** (présent ⇒ accès lecture/écriture auto = règle « membre-user »). C'est le vrai —
  et petit — chantier d'identité, à faire **en amont** du front, en réutilisant `SquadRepo` (cf. anti-ART).
- **Explicite & multiple** : le user crée/nomme ses escouades (plusieurs autorisées : trio ranked,
  4-stack BTB, duo chill…). **Bootstrap anti-friction** : bouton « Enregistrer cette compo comme
  escouade » depuis la sélection de coéquipiers / une `composition_session` détectée.
- **Accès partagé sans consentement** : un membre dont le `xuid` mappe à un user LevelUp (via
  `db_profiles.json` ⇒ `userID` renseigné) a accès **auto en lecture *et* écriture** aux objectifs (et
  futurs arcs) — il peut définir/éditer. Pas d'accept/refuse (les membres s'accordent en jeu). Les membres
  `xuid` **non-users** (`userID` vide) sont juste comptés pour la présence/mesure, aucun accès.
- **Convergence (anti-doublon)** : une escouade s'identifie par son **roster exact** (ensemble de `xuid`).
  Un co-membre user retrouve **la même** entité via `ListSquadsForUser` — pas de doublon. `{A,B}` et
  `{A,B,C}` sont **deux escouades distinctes** (pas de matching « superset » flou — abandonné).
- **Règle de comptage de progression — no-overlap (corrigé 2026-06-09, raisonnement « session »)** : un
  match compte pour l'escouade `S` **ssi** :
  1. `roster(S) ⊆ participants(match)` (tout le roster présent), **et**
  2. **aucun autre coéquipier connu** présent : aucun `xuid` de `(⋃ rosters des escouades du user) \
     roster(S)` n'est dans `participants(match)`, **et**
  3. les **randoms** (xuid dans aucun roster — comblent les trous 4v4 / 16v16) sont **ignorés**.
  - Effet : un match `{A,B,C}` ne crédite **que** le trio (C présent disqualifie le duo `{A,B}`). Le duo
    n'avance que sur de vrais matchs sans C. → un membre qui revient en session ne voit jamais l'objectif
    « avoir bougé sans lui ». **Overlap supprimé** (l'hypothèse « overlap assumé » est abandonnée).

**Anti-ART** : réutiliser `SquadRepo` / `SquadChallengeRepo` existants. `shared_social` n'est **pas** sur
le hot-path `BatchBuilder` (per-match `shared_matches_v2`) — écritures escouade = HTTP basse fréquence.
Discipline : suivre la convention d'écriture des repos prestige, **pas de nouveau `ON CONFLICT … DO
UPDATE` concurrent**, vérifier l'allowlist `internal/sync/no_art_patterns_test.go` avant l'extension `xuid`.

**Reste à faire en amont du front** : re-key `SquadMember`→`xuid` (via `SquadRepo`) + helper de résolution
d'accès `xuid`→`userID` + l'évaluation de la règle no-overlap. **Pas de nouvelle table.**

---

## Ordre suggéré & estimation (révisé 2026-06-09)

**Priorité produit user : Phase C (squad) d'abord.** Maquettes A & C validées (cf. mini-specs).

1. **Phase C** (squad) — **prioritaire**. La plus structurelle (touche `coach_advisor`, `prestige`,
   front squad). Identité d'escouade **tranchée** (cf. § Identité d'escouade) ; **pré-requis amont** (pas
   de table à créer — `Squad`/`SquadRepo` existent) : re-key `SquadMember`→`xuid` + helper résolution
   d'accès + évaluation no-overlap, avant le front. Backend (profil agrégé + signal squad + filtre pool)
   testable indépendamment.
2. **Phase A** (négatif soft) — backend testable vite ; front = `CoachFocusCard` (option B validée).
3. **Phase B** (tone) — **plus une phase autonome** : réduite à la guideline + banque i18n universelle,
   mutualisée avec A.

Chaque phase = **branche dédiée**. Ne pas tout ouvrir d'un coup.

## Références
- `internal/progression/coach_advisor/{service.go, signals.go, synthesis_grammar.go}`
- `internal/progression/profile/lowess.go`
- `internal/prestige/{types.go, service_pilot_pool.go, squad_coach.go, service_squads.go}`
- `internal/domain/settings.go` (settings joueur)
- ADR 0014 §6.1 (cadre positif — **amendé le 2026-07-25**, cf. § dédiée en tête de l'ADR),
  ADR 0020, ADR 0021

## Journal d'exécution

- **2026-06-09** : Phases A et C livrées (commits `7cefd911a`, `b0dd5256a`, `b916e1191`,
  `c00b7850f`). Phase B fusionnée de fait dans A comme anticipé par ce plan. Ce fichier plan
  n'a pas été mis à jour à ce moment (statut resté « Proposé ») — c'est la source de l'écart
  documentaire corrigé ci-dessous.
- **2026-07-25** (chantier v7.2.1, item V721-07 — cf. `.ai/V7.2.1/PLAN_V721_NOTION_BATCH.md`) :
  - 07.1 — ce plan corrigé sur pièces : statut réel par phase (tableau « État d'exécution »),
    2 références de lignes périmées corrigées (`types.go:246-259`→`:268`,
    `service_pilot_pool.go:145-195`→`:181-239`), affirmation « aucun signal coach niveau
    escouade » retirée/invalidée (§ Phase C), cooldown soft-négatif statué `[!]` (décision
    assumée : pas de bénéfice mesuré face au coût d'un état persistant supplémentaire).
  - 07.2 — ADR 0014 amendée (`docs/adr/0014-progression-tracking-v2-ascension.md`) : nouvelle
    section datée documentant le signal soft-négatif et ses garde-fous exacts (seuil -0.10,
    fenêtre 14j, catégorie neutre `CategoryTrendConsolidate`, jamais de rendu "défaite"), plus
    correction ponctuelle des deux passages qui affirmaient encore "positif uniquement".
  - 07.3 — `CoachFocusCard.tsx` migré de l'objet `STR` local vers le manifest i18n
    `profile.coach.*` (`apps/web/src/lib/i18n/manifests/profile.toml` +
    `apps/web/src/lib/i18n/generated/profile.ts` mis à jour à la main, cohérent avec le format
    produit par `build_i18n_manifests.mjs` — à régénérer/vérifier au gate `make generate-types`
    ou équivalent i18n).
  - 07.4 — ce plan archivé vers `.ai/archive/` et retiré de `.ai/BACKLOG.md` (chantiers suivis
    dans des plans dédiés).
