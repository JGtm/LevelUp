# PLAN — Retours utilisateur du 2026-08-29 (8 points) : analyse, plan, pilotage

> Écrit le 2026-08-29 par la session pilote (Fable 5), sur la base de 6 rapports d'agents
> d'analyse (2 Opus, 3 Sonnet, 1 Haiku) — tous en lecture seule, constats vérifiés sur
> pièces avec file:ligne. Contrat d'exécution : skill `plan-execution` (le présent plan
> précise deux aménagements au contrat, cf. §6 Gates et §5 Commits).
> Branche : ~~`feat/v75` (worktree partagé `LevelUp-go-migration`)~~ — **CORRIGÉ le
> 2026-08-29 ~16h45 sur retour utilisateur** : le chantier vit désormais dans le worktree
> DÉDIÉ `LevelUp-wt-retours-0829`, branche `wt/retours-0829` (base `7befe6192`, feat/v75).
> Il porte : le socle kills-hors-arme + les lots A/B/C/D + ce plan. Les lots étrangers
> (rejeu/sons, assistances, score-manches) sont restés dans l'arbre partagé ; mes fichiers
> EXCLUSIFS en ont été retirés. Restent dans l'arbre partagé des hunks résiduels de mes
> lots dans les fichiers CHEVAUCHÉS (fragClass.ts+guard, rules.tsv, frag_distribution.go,
> weaponRoleInsight.ts+test, session_page_service.go, timeseries_service.go, thought_log,
> PLAN_KILLS_HORS_ARME.md, REGISTRE_REPORTS.md, frags.toml, weapon_names.toml et les
> fichiers Go du socle kills) — identiques au contenu porté ici, à purger là-bas lors de
> l'arbitrage des commits (cf. §7).

## 0. Correspondance points utilisateur → verdicts → lots

| Pt | Sujet | Verdict d'analyse (résumé) | Lot |
|---|---|---|---|
| 1 | Backfill armes de kills après le nouveau parser | Mesuré : `weapon_kills` 1 361/1 948 matchs (69,9 %) ; kill events film 1 365 ; artefacts rejeu 40/1 948 (2,1 %). ~581 matchs (30 %) DÉFINITIVEMENT irrécupérables (film Theater expiré, fenêtre ~30 j). Il reste : 2 films en cache non décodés + 911 films sans artefact rejeu (~8 h de cuisson). | E (rapport + commandes, décision utilisateur) |
| 2 | Donuts répartition frags 2 niveaux + page Escouade | Lot `kills-hors-arme` COMPLET côté code, blocage tiers LEVÉ (tous gates Go verts aujourd'hui, intégration comprise). 12 défauts résiduels D1-D12 relevés, dont 2 majeurs : rôles de niveau 2 peints en BLANC PUR (invisibles) et détail Escouade non plafonné (~11 lignes noient les 8 armes). | 0 + A |
| 3 | Premier frag / première mort flou + tooltip uuid | Chart `FirstBloodLanes` (3 surfaces : session, timeseries, escouade). « Flou » = points 6 px à 40 % d'opacité qui se superposent ; tooltip = uuid car le DTO `FirstBloodMatchPoint` ne porte QUE `match_id`. Métadonnées déjà en scope aux 3 points de construction Go. | C |
| 4 | Échelles rendement/résistance (escouade, dernière session) | Axe Y VOLONTAIREMENT figé 50-200 % (`ONE_LIFE_RATE_BOUNDS`), formules Go sans clamp → écrêtage. 2 surfaces (escouade + timeseries solo adapté). | B |
| 5 | Échelle du graphe avec bonus | 2 occurrences : butterfly Frags/Morts escouade + `TimeseriesKdaTrend`. Axe auto recalculé sur les séries VISIBLES → activer « Bonus » rescale tout. | B |
| 6 | Breadcrumb page rejeu | `MatchBreadcrumb` (match-view) réutilisable tel quel : retour historique + repli explorer. ~15 lignes. | D |
| 7 | Équipement : ramassage vs usage | Le film ne porte AUCUN événement de ramassage (négatif mesuré). Déployé vs lâché : déjà livré et affiché. Camo/surbouclier : la distinction n'existe pas dans la donnée (l'écran le dit déjà). Reste 1 sonde de recherche jamais faite (`unit-equipment-component`). | F (plan) |
| 8a | Frags sous camo/surbouclier/translocateur actifs | FAISABLE pour les frags à coût quasi nul (fenêtres `EquipmentEpisode` déjà publiées, killsource déjà décodé au même endroit, même horloge). Dégâts chiffrés : RÉFUTÉ (aucun montant HP dans le film ; interdiction écrite `RE_LOG_KILLWEAPON.md:10927`). Translocateur : BLOQUÉ (2 poses / 11 films). | F (plan) |
| 8b | Affichage (pages ? forme ?) | Recommandation : données brutes (2 colonnes tableau équipement existant) + « temps forts » (badge narratif), VUE MATCH UNIQUEMENT. Rien en agrégé : 2,3 % de couverture d'artefacts (précédent GATE 0 kills-hors-arme, seuil 40 %, échoué à 29,4 %). | F (décision utilisateur) |

## 1. Décisions tranchées avant exécution (pilote, réversibles, consignées)

- **DEC-1 (D4/R3)** : les classes non-combat (`equipment`, `environmental`) sont EXCLUES
  du breakdown « par arme » côté web (`fragDetailBreakdown.ts`) — alignement sur
  l'intention Go du lot (`nonCombatFragClasses`, test
  `TestKillSource_ClassesHorsBreakdownParArme`) et sur le commentaire
  `squadFragTools.ts:8` qui redevient vrai. Elles restent visibles au SUNBURST (niveau 1+2).
- **DEC-2 (D6/R9)** : PAS de niveau 2 sur les barres empilées Escouade dans ce lot
  (l'utilisateur a dit « à voir ») — consigné en Découvertes, option chiffrée M.
- **DEC-3 (8b)** : la recommandation A+C (vue match uniquement) est SOUMISE à
  l'utilisateur avant toute exécution du lot F. Aucune ligne de code équipement dans
  cette passe. **VALIDÉE par l'utilisateur le 2026-08-29 soir (« ok pour 8a et 8b ») —
  F.0 (mesure d'entrée) lancé dans la foulée ; F.1-F.3 conditionnés à son verdict
  (seuil 30 % écrit d'avance).** Précision utilisateur sur le point 7 : l'inventaire du
  rejeu SE MET À JOUR sans événement de ramassage — exact, et c'est la clé : le film
  publie l'ÉTAT porté (continu), pas l'ÉVÉNEMENT de ramassage (négatif mesuré). Le
  « ramassage » est donc DÉRIVABLE des transitions d'état — F.4 requalifié : sonde
  `ti=35 i26 unit-equipment-component` + dérivation des transitions d'inventaire
  (lot de RE complémentaire, comme pressenti par l'utilisateur).
  **MISE AU CONDITIONNEL (29/08 ~21h30, RE session équipement)** : la réfutation
  « aucun événement de ramassage » (0/149 fenêtres, phase 0.2) est désormais
  SUSPECTE — la passe Ghidra sur `FUN_140c1e4d0` (déser i10 object-parent-state)
  désigne le handle du parent à +0x274, champ que la mesure 0/149 n'a PAS testé, et
  un défaut `param_4` (i10 absent de paramByComponent → défaut 1) peut désynchroniser
  la lecture des objets DÉTACHÉS sur le chemin même de la mesure. Statut : ni prouvé
  ni réfuté — verdict attendu de la session équipement (balayage SetRecordStateParam
  puis remesure oracle CTF sur +0x274 seul). Le chemin PAR DÉFAUT du lot F reste la
  dérivation i42-delta (100 % atteignable, mesure du 24/08) ; il serait REMPLACÉ par
  l'événement daté si la RE le prouve. Répartition actée : RE/film = session
  équipement (Ghidra) ; 8a/8b vue match + F.0 + ce plan = session pilote.
  Visée/ADS (hors périmètre, pour mémoire) : pas de composant ECS de zoom ; piste
  vivante = drapeaux flag0/flag1/flag2 d'i21 (second vecteur 23 bits jeté par le
  port Go) ; le descope part en télémétrie Xbox.
- **DEC-4 (P3)** : le tooltip premier frag/mort affiche carte · mode · date (plus jamais
  l'uuid) ; dégradation propre si un libellé manque (jamais de clé brute ni d'uuid).
- **DEC-5 (P4/P5)** : échelle A (rendement/résistance) = élargissement SANS rétrécissement
  (la fenêtre 50-200 % reste le plancher de comparabilité) ; échelle B (bonus) = extent
  figé calculé sur le JEU COMPLET (bonus + joueurs masqués inclus), indépendant des toggles.
- **DEC-6 (P1)** : AUCUN backfill lancé sans accord explicite (écritures prod, arrêt
  serveur requis pour killsource, passe rejeu ~8 h).

## 2. Périmètre FERMÉ par lot

### LOT 0 — Clôture kills-hors-arme (statut : gates déverrouillés, reste la paperasse)
- [x] 0.1 Mettre à jour `.ai/V7.5/PLAN_KILLS_HORS_ARME.md` : GATE 5 / GATE 6 rejouables →
      constats du jour (build+vet+tests service/api/contracttest/platform verts,
      intégration duckdb verte, 4 échecs `team_0_rounds_won` disparus).
- [x] 0.2 Registre `.ai/V7.5/REGISTRE_REPORTS.md` : ligne datée ajoutée (blocage tiers
      levé + lot déplacé en worktree dédié + littéral `'marche'` du test d'intégration
      corrigé vers `killscope.ReadPathFilmWalk` — le garde archlint J4R-3 le refusait,
      manqué par les gates par-paquet du lot).
- Gate : couvert par le gate final VF (§6) — aménagement assumé du contrat (une seule
  passe de tests globale couvre lots 0+A+B+C+D, l'arbre étant partagé).

### LOT A — Donuts 2 niveaux + Escouade (12 défauts → 5 items) — agent Opus
- [x] A.1 (D1/R2) `fragClass.ts` : rampe de teinte du niveau 2 bornée (normalisée sur le
      nombre de rôles, plafond ~0,7 — plus jamais de blanc pur), et `FragSunburst.tsx:262` :
      le texte de la ligne de rappel cesse d'être peint avec la couleur d'arc (token texte
      standard). Test : cas ≥ 5 rôles dans `FragSunburst.test.tsx`.
- [x] A.2 (D4/DEC-1) `fragDetailBreakdown.ts:42` : filtrer les classes non-combat du
      breakdown par-arme + test web dédié (le premier sur ce cas).
- [x] A.3 (D5/R4) `squadFragTools.ts:119-122` : plafonner aussi les lignes de détail
      (regroupement « Environnement » accepté) + cas de test.
- [x] A.4 (D7+D8+D9/R5) : restaurer le commentaire du Mutilator (`rules.tsv:81`, depuis
      `git show HEAD:` — SANS toucher aux `weapon_key` posées par le lot) ; corriger la doc
      de `FragRoleEntry.Label` et `perWeaponFragClasses` (commentaires seulement, zéro
      comportement) ; corriger la prétention de miroir Go↔web (documenter la divergence
      réelle des deux ensembles).
- [x] A.5 (D10/R6) : token de couleur hors famille bleue pour `equipment` ; inclure
      `unattributed` dans le contrôle ΔE du guard — la collision PRÉ-EXISTANTE
      `spartan_ability`/`unattributed` (6,89) se traite par exception datée ciblée, jamais
      par baisse du seuil global.
- Gate A : `npx vitest run src/lib/accessibility src/features/squad src/components/charts`
  (ciblé) + `npm run typecheck` + ESLint fichiers touchés, 0 erreur.

### LOT B — Échelles (points 4+5) — agent Sonnet
- [x] B.1 `oneLifeWindow.ts` : `oneLifeWindowBoundsForData(values, base)` (élargit, ne
      rétrécit jamais) ; appliquer à `squadEfficiencyChart.ts:269-279` et
      `TimeseriesSquadAdapted.tsx:438-443` ; adapter les assertions pinnées
      (`squadEfficiencyChart.test.ts:150-254`, `oneLifeWindow.test.ts`,
      `TimeseriesEfficiency.test.tsx`) + nouveau cas « dépasse la fenêtre ».
- [x] B.2 Helper `stackedAxisExtent(...)` dans `components/charts/_utils.ts` ; appliquer à
      `squadPerformanceLineCharts.ts` (butterfly : min négatif morts, max kills+bonus,
      indépendant de `hiddenTypes`/joueurs masqués) et `TimeseriesKdaTrend.tsx:93-99`
      (min 0, max sur jeu complet) ; tests neufs : « toggler Bonus ne change pas
      yAxis.min/max » (B1) + premier test de `TimeseriesKdaTrend` (B2).
- Gate B : `npx vitest run src/features/squad src/features/timeseries src/lib/charts` +
  `npm run typecheck`, 0 erreur.

### LOT C — Premier frag / première mort (point 3) — agent Sonnet
- [x] C.1 Lisibilité `FirstBloodLanes` : nuage supprimé quand la lane porte ≤ 2 matchs ;
      points/opacité relevés modérément ; médianes inchangées ; test modèle.
- [x] C.2 DTO : `FirstBloodMatchPoint` gagne carte/mode/date (champs choisis d'après ce que
      portent DÉJÀ les lignes en scope aux 3 builders — `StatsMatchRow`, `SquadMatchRow.MapUI/
      PairNameFR/StartTime`) ; peupler dans les builders (`buildSoloFirstBlood`,
      `buildSquadFirstBlood`), en touchant les call-sites uniquement si une signature l'exige.
- [x] C.3 Régénération contrat : openapi + `make generate-types` (une seule fois, ce lot est
      le SEUL de la vague à toucher au contrat).
- [x] C.4 Web : `_shared/firstBlood.ts`, tooltip `FirstBloodLanes.tsx:386-390`, manifest
      `first_blood.toml` FR+EN (placeholders `{map}`/`{mode}`/`{date}` — plus de `{match}`),
      régénération i18n ; tests Go (3 services) + `FirstBloodLanes.test.tsx` adaptés.
- Gate C : `cd apps/go-api && go test ./internal/service/... ./internal/domain/...
  ./contracttest/...` + `npx vitest run src/components/charts src/features/_shared` +
  `npm run typecheck`, 0 erreur.

### LOT D — Breadcrumb rejeu (point 6) — agent Haiku
- [x] D.1 Réutiliser `MatchBreadcrumb` (export depuis `MatchHeader.tsx` si nécessaire) dans
      `routes/.../replay.tsx` ; label via `buildMatchHeadingStr` sur `matchView.header` ;
      NE PAS toucher `features/match-replay/i18n.ts` (fichier d'un autre lot) — réutiliser
      les strings existantes.
- Gate D : `npm run typecheck` + ESLint fichiers touchés, 0 erreur.

### LOT E — Backfill (point 1) — RAPPORT, exécution différée [!] (DEC-6, décision utilisateur)
**CORRECTION UTILISATEUR (2026-08-29 soir) — la doctrine « fenêtre ~30 j » est au moins
partiellement FAUSSE** : vérifié la veille (par l'agent « réparation des assistances »),
les films des matchs 2025 ET 2026 sont TOUS disponibles côté serveur, et l'utilisateur les
a aussi en local. Conséquences actées : (a) le bit22 `MBitWeaponKillsNoFilm` est un
« échec constaté un jour », PAS une preuve d'expiration — le moteur de convergence le
traite en terminal et ne re-tente jamais : c'est LUI le trou ; (b) le « ~30 % irrécupérable »
est SUSPENDU en attendant la ventilation par année des 581 matchs bit22 (agent G, point 5) ;
(c) toute passe de backfill doit RE-TENTER les bit22 (`--force-no-film`) au lieu de les
tenir pour perdus ; (d) coordonner avec le chantier tiers EN VOL
`cmd_backfill_killsource_online.go` / `killcollector/remote_films.go` (récupération
distante des films — ne pas dupliquer). ORDRE VOULU PAR L'UTILISATEUR : d'abord la
conception headshot/distance (LOT G) pour que LE backfill capture tout en UNE passe, et
paramétrer le SYNC pour porter ces données au fil de l'eau ; la réflexion narrative vient
après, données en poche.
Verdict initial (à relire avec la correction ci-dessus) : la couverture n'est PAS complète.
Actions proposées, dans l'ordre, quand l'utilisateur valide :
1. `go run ./apps/go-api/cmd/levelup backfill-replay --repair-impoverished` (2 artefacts,
   secondes, dry-run déjà validé le 2026-08-25, serveur actif OK).
2. `go run ./apps/go-api/cmd/levelup backfill-killsource --dry-run` puis sans flag
   (~2 films restants, minutes) — **SERVEUR ARRÊTÉ obligatoire** (OpenReadWrite direct, ADR 0013).
3. Passe de masse artefacts rejeu : `backfill-replay` par tranches `--limit N` (total ~8 h,
   ~2 Go, serveur actif OK — RO côté DB) — c'est ELLE qui donne le kill feed du rejeu partout.
4. Optionnel, gain quasi nul : `backfill --weapons --force-no-film` (581 no-film re-tentés).

### LOT F — Équipement (points 7/8) — F.0-F.3 RENDUS (30/08) ; F.4-F.6 non planifiés
Découpage validé par l'analyse (7 sous-lots, ordre strict, le 0 peut tuer les suivants) :
F.0 mesurer (S) — [x] RENDU le 29/08 soir (35 artefacts traités, 149 épisodes, 26 matchs
    à épisode). RÉSULTAT : **30,2 % des épisodes portent ≥ 1 frag en lecture large
    (GATE 30 % : PASSÉ au seuil près) ; 39,3 % en lecture stricte** (matchs
    `LineByLinePublishable`, la porte que le rejeu applique réellement). Détail : camo
    26,2 % (SOUS le seuil seul), surbouclier 40,5 % ; 60 frags + 8 assistances sous
    effet ; par match : min 0, médiane 1,5, max 10. Incertitude ±7-8 pts (n=149).
    DEUX ACQUIS D'IMPLÉMENTATION (vérifiés, à réutiliser tels quels en F.1) :
    (a) horloge : `replayMs = Kill.TimeMS − OriginMs` directement (validé 99,9 % des
    1 822 kills dans [0, FrameCount)) — PAS la formule event_time_ms+t0_ms ;
    (b) `Feed.Killer` porte un GAMERTAG (pas un xuid) — résolution via
    `killcollector.MatchIdentities.Resoudre` (code de prod, 99,0 % résolus), et le
    pont slot→xuid est DÉJÀ dans l'artefact (`Track.XUID`).
    **DEC-7 (pilote, RÉVISÉE après contre-lecture de la session équipement)** : GO
    F.1-F.3, mais le verdict est reformulé honnêtement.
    (a) La contre-lecture est exacte sur l'agrégat en lecture LARGE : 30,2 % = passé à
    UN épisode près (45 requis 44,7 ; 44 = 29,5 %) — marge 0,3 épisode, et le camo y
    est SOUS le seuil (26,2 %) porté par le surbouclier. Sur ces chiffres-là, gate
    INDÉCIS, pas franchi.
    (b) Ce qui tranche : la population qui AFFICHE des chiffres est, par construction
    de F.1/F.2, la population STRICTE (`LineByLinePublishable` ; porte fermée →
    `KillsRead=false` → « — », jamais un zéro). Le but écrit du gate (« pas un champ
    de zéros à l'écran ») s'évalue donc sur ELLE, DÉNOMINATEURS EN FACE (règle de
    traçabilité appliquée à soi-même, 2e passe de la même contre-lecture) :
    **camo 35,2 % = 25/71 épisodes (seuil 30 % = 22 → marge 3 ÉPISODES) ; surbouclier
    55,6 % = 10/18 (seuil 6 → marge 4) ; global 39,3 % = 35/89 (seuil 27 → marge 8)**.
    Les DEUX familles passent, jugées séparément — mais les marges se comptent en
    UNITÉS d'épisodes : c'est un GO à petite population, pas un GO statistiquement
    confortable, et c'est précisément ce que la re-mesure post-cuisson (n ~ ×10) doit
    consolider ou infirmer.
    (c) Concessions actées : la colonne camo est livrée AVEC sa mesure écrite en face
    (infobulle : 26,2 % en lecture large — sous le seuil) ; re-mesure OBLIGATOIRE des
    deux familles après la cuisson de masse (n=149 → ~10×) ; si l'utilisateur préfère
    surbouclier seul, retirer la colonne camo est un diff de quelques lignes.
    (d) LEÇON (Découvertes) : un gate chiffré doit pré-écrire sa POPULATION et sa
    règle de MARGE — celui-ci ne fixait ni l'une ni l'autre, d'où deux lectures
    possibles après coup. Crédit : l'objection vient de la session équipement.
F.1 Go (M) — [x] RENDU le 30/08 (agent Sonnet, worktree dédié) : `Options.Kills`
    (`KillsInput{Read,Kills}`), `attachEpisodeKills`/`attachAllEquipmentKills` PURS
    (`internal/analysis/replay/equipment_episode_kills.go`, 15 tests synthétiques — fenêtre,
    hors-fenêtre, bornes T0/T1, porteur≠tueur, assist connu/inconnu, épisode fermé par mort),
    `EquipmentEpisode.K/A` (omitempty), `Coverage.Equipment.KillsRead`. Câblage
    `internal/replaybuild/kills.go` (nouveau) : décodage killsource PARTAGÉ avec
    `neutralDeaths` (une seule passe, `decodeKillSource`), résolution gamertag→xuid HORS LIGNE
    via `replay.ScanFilmDeaths` (le fil des morts du film porte xuid+gamertag dans le même
    enregistrement — aucune base, contrat `replaybuild` intact ; `killcollector.MatchIdentities
    .Resoudre` est DB-backed et n'a donc pas été importé — même RÈGLE à deux cas
    (`xuid:` préfixe / table gamertag), copie déclarée et justifiée en commentaire, 2ᵉ copie
    sur 2 tolérées). `KillRef` renommé `EquipmentKillRef` (collision avec le type existant de
    `killpos.go`, chantier arme-par-kill, non liée). SchemaVersion 23→24 (document.go +
    structure_test.go + golden fixture + openapi.yaml régénéré + generated.ts régénéré + copie
    web `replaySchemaLogic.ts`). Gate : `go build ./... && go vet ... && go test ...`
    418 sous-tests verts, 0 échec. Re-cuisson des 44 artefacts NON FAITE (pas de `data/` dans
    ce worktree) — commande notée au rapport, à jouer post-merge.
F.2 Web (S) — [x] RENDU le 30/08 : 2 colonnes « Frags sous camo/surbouclier » dans le groupe
    États actifs (`equipmentUsageLogic.ts` : `EquipmentUsageEpisode.kills`,
    `EquipmentUsageCoverage.killsRead` ; `equipmentUsageColumns.ts` : `killsCell` avec repli
    « — » quand `killsRead` est faux). i18n FR/EN (`activeKillsFamily`, `i18nContract.ts`+
    `i18n.ts`) ; réserve mesurée AJOUTÉE au `groupActiveHint` existant (même groupe, même
    infobulle — pas un second texte) plutôt qu'un nouveau mécanisme de tooltip par colonne.
    Aucune couleur hex. Tests existants corrigés (kills:0 dans les fixtures, décalage
    d'indices de cellules dans MatchEquipmentUsageSection.test.tsx) + tests neufs. Gate :
    vitest match-replay+match-view 2067 tests verts, typecheck propre.
F.3 Temps forts (S/M) — [x] RENDU le 30/08 : `features/match-view/equipmentKillBadges.ts`
    (nouveau, pur) — meilleur épisode par famille (max k), seuil 3 ÉCRIT EN DUR
    (`EQUIPMENT_KILL_BADGE_THRESHOLD`, non ajusté), un badge par famille max, aucun badge sans
    propriétaire de slot mesuré. Câblé dans `MatchImpactBadgesBar.tsx` (auto-alimentation
    `useMatchReplay`, même clé de cache que `MatchEquipmentUsageSection`, aucun appel réseau de
    plus) : rendu `NarrativeBadge` (pilule `tokenVar('success')`, délibérément distinct de la
    carte des badges serveur — deux sources différentes) + `Tooltip` portant la réserve
    (renvoie à celle de F.2). i18n `killBadgeFmt`/`killBadgeHint` posés dans le dictionnaire du
    rejeu (`match-replay/i18n.ts`), importés par `match-view` (même sens que
    `MatchEquipmentUsageSection`, déjà réutilisé tel quel par ce fichier). Gate : vitest
    match-view+components/feedback 265 tests verts (9 dédiés au seuil/meilleur-épisode/
    unicité-par-famille/sans-propriétaire), typecheck propre.
F.4 Sonde recherche ramassage (M) : `ti=35 i26 unit-equipment-component` puis `ti=37 i19`.
F.5 Sonde fin de pose (M) : `ti=37 i24 equipment-energy` R(14).
F.6 Agrégats (L) : CONDITIONNÉ (couverture ≥ 40 % après cuisson de masse), pas planifié.

## 3. Vague 2 (après le gate C — même arbre, régénération de contrat séquencée)
- [x] V2.1 (D2+D3/R7) : libellés de niveau 2 honorant la locale — `FragRoleEntry.LabelEN`
      (json `label_en`) posé sur les DEUX chemins (registre via `weapon_kills` ET
      killsource via `port.KillSourceClassRow.LabelEN`), résolveur à double COALESCE
      FR-first/EN-first (`weapon_resolver.go`), web `fragRoleLabel.ts` choisit selon la
      locale avec repli croisé puis clé générique `frags.role.generic_object` (FR
      « Objet » / EN « Object ») — plus AUCUN chemin n'affiche de clé brute ; contrat
      régénéré (openapi + generated.ts + manifests). Exécution : agent Sonnet interrompu
      à ~95 % par une limite de quota (reset 20h20), repris et complété par le pilote ;
      gates rejoués verts.
- [!] V2.2 (D11/R10) : mesure de latence `LoadKillSourceClassesAggregated` sur Synthèse
      « tout l'historique » — NON MESURABLE dans le worktree dédié : `data/` (bases prod)
      n'y existe pas, et les autres arbres sont tenus par des sessions concurrentes.
      Non bloquant (D11 = « à mesurer, pas une régression »). À jouer sur un arbre portant
      `data/` (ou après merge) : instrumentation temporaire + chiffre au journal.

## 3bis. LOT G — Tir à la tête + distance par kill/assistance + portée par arme (demande du 2026-08-29 soir)
Demande : pour chaque kill et assistance, headshot oui/non + distance tueur-victime ;
en déduire une portée par arme (corpus + guide Reddit weapon variants) ; AU MINIMUM :
concevoir la capture au sync + le paramétrage du backfill pour que les données existent,
la réflexion narrative venant après.
- [x] G.0 Faisabilité + conception — RENDUE le 29/08 soir (agent Opus, lecture seule).
      VERDICTS (sources file:ligne au rapport) :
      (a) HEADSHOT PAR KILL : DÉJÀ EN BASE — `match_kill_events.source_category`
      (modificateur du rapport de dégât, 4 bits du dead-state victime), 74 569 lignes
      renseignées, ORACLE contre `match_participants.headshot_kills` (API officielle) :
      **99,3 % d'accord exact** avec le filtre STRICT `= 'Headshot'` (ajouter
      `HeadshotMultiplier` fait chuter à 84,4 % : INTERDIT, garde-rail à poser).
      Colonne ORPHELINE : décodée, écrite, relue par PERSONNE. Le pictogramme
      killfeed-64 « headshot » existe dans l'atlas, jamais référencé par killicon.
      Contrat canonique déjà prêt (`canonical.MatchEvent.Headshot`, rempli par H5
      seul, capability `match.killfeed.per_kill = degraded` pour HINF).
      Assistances : « headshot de l'assistant » N'EXISTE PAS par construction (UN
      modificateur PAR MORT) — RÉFUTÉ ; la DISTANCE de l'assistant est mesurable
      (22 708 lignes avec assist_xuid).
      (b) DISTANCE : code ÉCRIT, TESTÉ, JAMAIS BRANCHÉ (`replay.BuildKillPositions`,
      60 Hz, tolérance 120 ms, Z inclus, mètres monde via AABB BSP — 79 cartes au
      catalogue) ; table `kill_positions` migrée, 0 ligne pour Infinite. Mesure de
      validation (36 artefacts, 1 994 couples) : mêlée p50 **0,39 m**, assassinat
      0,43, marteau 1,84, AR 4,57, Sidekick 5,38, Bandit 7,37, **Sniper 16,19 m** —
      l'ordre du design du jeu reproduit sans le connaître. Horloge confirmée (95 %
      des kills à ≤ 1 frame de la fin de vie victime). Couverture plancher 75,8 %.
      (c) PORTÉE : guide Reddit INACCESSIBLE (403, dit sans détour) — notion de
      référence = RRR (Halopedia) ; méthode : par `source_tag` (les VARIANTES sont
      déjà distinguées), n ≥ 100, p50/p90 jamais le max, normalisation PAR CARTE
      obligatoire (bornes 78 m → 1 130 m — sinon on mesure la carte, pas l'arme).
      NOTRE corpus mesure l'USAGE, pas le RRR — confronter, jamais confondre.
      (d) CONCEPTION : headshot = AUCUN schéma (chantier de LECTURE) ; distance =
      remplir `kill_positions` (JAMAIS de colonne dans match_kill_events — doctrine
      « on stocke une mesure, pas une résolution améliorable ») ; greffe dans
      `replaybuild` (a déjà les deux moitiés) ; DETTE BLOQUANTE avant remplissage :
      `kill_positions` n'est PAS append-only (ni id/written_at/vue _latest → doublons
      silencieux au re-décodage) ; backfill = bump de `KillSourceDecoderRev` + la
      branche `--online` du chantier tiers ; CONTRAINTE d'ordonnancement : un
      producteur crédit-seul repassant APRÈS le film effacerait la source de la vue
      `_latest`.
- [x] G.1 (S) — LECTURE headshot — RENDU le 30/08 (agent Sonnet, worktree dédié).
      Filtre `= 'Headshot'` STRICT : constante+prédicat canoniques
      `killscope.CategoryHeadshot`/`IsHeadshotCategory` (`internal/domain/killscope/
      killscope.go`) — MÊME paquet que les voies de lecture (précédent J4R-3), verrou
      d'égalité avec le décodeur dans `sync/killcollector/category_headshot_test.go`
      (le seul paquet qui importe `killscope` ET `killsource`). Garde-rail
      `internal/archlint/no_raw_headshot_category_literal_test.go` : bannit le
      littéral `"HeadshotMultiplier"` hors de ses 2 propriétaires — SCOPÉ à ce seul
      littéral (pas `"Headshot"` bare, qui est AUSSI le nom d'une médaille du jeu :
      un ratchet plus large aurait banni des fixtures légitimes sans rapport, trouvé
      en le rodant contre le dépôt réel). Chaîne de lecture : `Q21bKillSources`
      (`platform/duckdb/queries_match.go`) lit désormais `source_category` en plus de
      `source_tag`, avec la MÊME garde d'unanimité appliquée INDÉPENDAMMENT aux deux
      colonnes (`HAVING count(DISTINCT source_tag)=1 AND count(DISTINCT
      source_category)=1` — un double kill peut porter la même arme et des catégories
      différentes, cas qui n'existait pas avant ce lot) ; `domain.KillSourceRaw.
      Headshot bool` (présence de ligne = connu, jamais un pointeur — même garde que
      `SourceTag`) ; `domain.MatchHighlightEvent.Headshot *bool` (nil = non
      mesurable, jamais `false` par défaut — doctrine `KillerDamagePct`) ; posé par
      `decorateKillFeed` (`service/match_view_killfeed_weapon.go`)
      INDÉPENDAMMENT de la résolution d'icône (le headshot ne dépend d'aucune table
      de noms, contrairement à l'icône d'arme). Contrat régénéré (`make openapi-gen`
      + `make generate-types` — seul lot à toucher au contrat dans cette passe).
      Icône : TRANCHÉ pour le FRONT, PAS `killicon`/`rules.tsv` — la vignette
      killfeed-64 (`nom_jeu: "headshot"`, `static/weapons-assets/halo_infinite/jeu/
      index.json:3091`) est un badge de MODIFICATEUR indépendant de l'arme
      (`source_category`, pas `source_tag`), alors que TOUT le système `killicon`
      résout par `source_tag` → aucune règle n'a de sens à y écrire ; l'URL fixe se
      compose côté front via `staticAssetURL` (`apps/web/src/lib/staticAssets.ts`,
      précédent CSR/rangs déjà établi) — pas de nouvelle méthode d'adapter Go.
      Capability `match.killfeed.per_kill` : **RESTE `degraded`, décision justifiée**
      — cette clé gate l'arme-par-kill (coverage ~33,7 % du feed, tag "Clé FINE, ne
      pas élargir" déjà posé sur les clés soeurs `film.kill_source`/
      `film.weapon_shots`) ET le contrat canonique `canonical.MatchEvent`
      (Kind/Weapon/positions, tous encore degraded/absents) ; le headshot devenu
      lisible est un fait PLUS ÉTROIT qui ne change ni l'un ni l'autre — la
      capability décrirait un plafond faux si elle bascule sur ce seul gain.
      `canonical.MatchEvent.Headshot` (events.go:131, pipeline SÉPARÉ de la
      Chronologie/timeline, `games/halo_infinite/events.go: mapInfiniteEvents`) :
      **PAS rempli dans cette passe** — décision délibérée, pas un oubli : cette
      timeline reconstruit ses kills par un appariement DIFFÉRENT
      (`analysis.ComputeKillerVictimPairs`, pas Q21b) et son propre commentaire
      documente déjà `Headshot` en dégradation groupée avec `Kind`/`Weapon`/`Loc`
      pour une raison qui reste vraie (RE film non câblé pour ces trois-là) ; le
      brancher aurait exigé une DEUXIÈME voie de plomberie source→événement, hors
      du périmètre « kill feed » que ce volet cible. UI : `MatchTugOfWarChart.tsx`
      (carte Dominance, PAS `ReplayKillFeed.tsx` — verrouillé Lot F) affiche
      désormais un décompte headshot dans le tooltip de vague
      (`combatHeadshotCountFmt`, FR/EN typé `MatchViewText`) ; le détail PAR KILL
      reste dans le rejeu 2D (fichier verrouillé) — la donnée est plombée jusqu'à
      `_momentum.ts` (`KillEvent.headshot?: boolean`, champ OPTIONNEL exprès : des
      fixtures de test du rejeu construisent des `KillEvent` complets sans lui,
      les rendre obligatoires aurait cassé des fichiers verrouillés) et prête pour
      un futur lot qui l'affichera par kill. Gate G.1 : VERT — `go build` + tests
      ciblés service/duckdb/api/contracttest/archlint 0 échec (hors 1 échec
      PRÉ-EXISTANT documenté `TestNoLocalLongestRun`) ; `-tags=integration
      ./internal/platform/duckdb/...` : 19 échecs `team_0_rounds_won` DÉCOUVERTS,
      vérifiés PRÉ-EXISTANTS (fichiers pristine, `git status` vide, sans rapport
      avec ce lot — tâche déportée, cf. Découvertes) ; web `vitest match-view` 257
      verts + `typecheck` propre.
- [~] G.2 (M) — POSITIONS : DETTE BLOQUANTE fermée, CAPTURE LIVE non câblée —
      RENDU PARTIEL le 30/08 (agent Sonnet, worktree dédié), justifié §9 rapport
      session. FAIT : `kill_positions` est désormais append-only (id PK + written_at
      + vue `kill_positions_latest`, recette ADR 0026 — `migration.
      ApplyAppendOnlyRebuild`, PAS le mécanisme `decode_pass` de
      `match_kill_events` : la clé fonctionnelle (match_id, killer_xuid, time_ms)
      n'a pas besoin de plus, la table n'a jamais eu de colonne `victim_xuid` pour
      affiner — découverte notée). Step `shared_append_only_kill_positions_v1`
      dans `games/halo_infinite/migrations/steps_appendonly_misc.go` (même famille
      que `match_csrs`/`pve_match_stats`), positionné dans `canonicalOrder`
      (`internal/migration/order.go`) juste après `shared_create_kill_positions` —
      testé de bout en bout via la VRAIE chaîne de migration (`RunForDB`), pas
      seulement le helper générique : 3 tests neufs dont un qui préserve des lignes
      H5 legacy à travers le swap et un qui verrouille le dédoublonnage
      `_latest` sur un re-décodage simulé (`games/halo_infinite/migrations/
      shared_kill_positions_appendonly_test.go`). L'UNIQUE lecteur brut existant
      (`h5KillFeedQuery`, `platform/duckdb/halo5/halo5_match_events_source.go`)
      bascule sur `kill_positions_latest`. Vérifié : `SetTitleStepsProvider(
      halomigrations.StepsFor)` est le provider UNIQUE utilisé aussi par
      `cmd/h5-sync`/`cmd/h5-backfill` — la conversion s'applique bien à la vraie DB
      partagée Halo 5, pas seulement à Infinite (aucun `StepsFor` séparé sous
      `games/halo_5/migrations`). NON FAIT (déféré, justifié) : câbler la capture
      Infinite dans `killcollector`. Investigation menée jusqu'au bout (pas un
      abandon sur inconnu) : `replay.BuildKillPositions` est PUR mais ses DEUX
      fournisseurs de données exportés sont DISQUE (`filmdec.
      ScanFilmBipedPositions(dir, opt)`, `replay.ScanFilmClockOrigin(dir)`,
      `replay.ScanFilmPlayerIndices(dir, roster)`) alors que `killcollector`
      décode aujourd'hui les chunks EN MÉMOIRE (`killsource.Decode(...,
      ChunkSourceOf(chunks), nil)`), sans jamais les écrire sur disque — la
      pipeline replay-artifact qui les utilise déjà (`internal/sync/
      replayartifacts`) le fait via un ENFANT DE PROCESS à plafond mémoire dur
      (`internal/replaychild`), disproportionné pour ce seul besoin et de toute
      façon hors périmètre (`internal/replaybuild` verrouillé). Solution
      identifiée et NON risquée (pont "écrire les chunks déjà en mémoire dans un
      répertoire temporaire, appeler les 3 fonctions exportées telles quelles, PAS
      de nouvelle logique de décodage) mais le tour complet — pont disque + nom de
      carte à faire transiter jusqu'à `MatchIdentities` + bornes monde via
      `filmdec.LoadMapQuantCatalog(paths.MapQuantBoundsPath(slug))` + inversion du
      pont slot→xuid de `ScanFilmPlayerIndices` + nouveau persister
      `KillPositionInsert` + nouvelle capability dédiée (doctrine « clé fine » des
      2 clés film soeurs) + tests d'intégration bout en bout — est un lot à part
      entière, pas un branchement d'une session. Formule VÉRIFIÉE sur le test
      `TestKillPositionsAppliqueLeDecalageDHorloge` (`internal/analysis/replay/
      killpos_test.go:88`) : `offsetUS = replay.ScanFilmClockOrigin(filmDir)`
      DIRECTEMENT (pas la machinerie `bestDeathOffset`/témoin, réservée au rejeu
      2D) — ce point n'était PAS évident avant lecture du test et vaut d'être
      consigné pour la reprise. **Déclencheur de backfill DÉCIDÉ** (S3) : bump de
      `killcollector.KillSourceDecoderRev` — PAS de nouveau flag `--positions`.
      Vérifié sur pièces (`cmd/levelup/cmd_backfill_killsource.go:357`,
      `matchsAJour`) : la commande EXISTANTE ne considère « à jour » que les
      matchs dont la passe `_latest` porte déjà `decoder_rev = KillSourceDecoderRev`
      — bumper la constante remet TOUS les matchs déjà décodés candidats à un
      nouveau passage, sans code neuf. Contrainte d'ordonnancement (crédit-seul
      après film efface la source) : PAS spécifique aux positions — propriété déjà
      documentée de `match_kill_events`, déjà couverte par la précondition
      existante « serveur arrêté » du backfill killsource (ADR 0013) ; le risque ne
      se matérialise que si cette précondition est enfreinte. Point de jonction
      avec le chantier tiers `backfill-killsource --online`/`remote_films` (arbre
      partagé, PAS ici) : complémentaire, pas concurrent — ce chantier élargit QUELS
      films sont disponibles, le bump de rev élargit CE QU'ON EN EXTRAIT ; aucun
      code dupliqué, se composent naturellement (plus de films dispo × decoder_rev
      bumpé = plus de matchs avec positions). Gate G.2 : `go build` + `go vet` +
      tests ciblés sync/migration/persist 0 échec ; `-tags=integration -p 1
      ./internal/migration/... ./internal/persist/... ./internal/platform/duckdb/...
      ./internal/sync/...` : SEULS les 19 échecs `team_0_rounds_won` déjà repérés en
      G.1 (mêmes fichiers pristine) — 0 échec imputable à ce lot.
- [x] G.2bis (M) — CAPTURE LIVE des positions câblée dans `killcollector`, RENDU le
      30/08 (agent Sonnet, worktree dédié). Pont retenu, conforme à l'option (a)
      identifiée par G.2 : `writeChunksToTempDir` recopie les chunks DÉJÀ EN MÉMOIRE
      vers `chunk_NN.bin` (même format que le cache film hérité,
      `haloclient.LocalFilmCache` — octets zlib ou clairs, `ReadFilmChunk` décompresse
      à la lecture, opération idempotente) puis appelle les scanners disque EXPORTÉS
      tels quels ; option (b) (variantes mémoire) écartée — un second décodeur des
      mêmes octets, exactement ce que la doctrine du paquet interdit. REFUS SUR TROU
      DE SÉQUENCE ajouté (non anticipé par G.2) : `filmdec.CountFilmChunks` s'arrête
      au premier index manquant, donc un film troué ferait lire les quatre scanners
      sur un PRÉFIXE silencieux — le pont compare l'index max déclaré au compte réel
      après écriture et refuse plutôt que de laisser passer une lecture partielle.
      DÉCOUVERTE NON ANTICIPÉE PAR G.2 (le point qui a le plus fait déraper l'estimé) :
      le pont slot→xuid qu'exige `BuildKillPositions` n'est PAS un sous-produit de
      `MatchIdentities` (résolution nom→xuid, gamertag↔xuid) comme le plan le
      supposait — c'est le pont DISTINCT `analysis/replay/owners.go` (fil des morts
      DU REJEU + index de joueur du film, LU et pas voté, mesuré 90/105 vies nommées
      0 collision sur le film témoin), jusque-là UNEXPORTÉ et interne à
      `BuildFromPositions`. Réimplémenter `buildLifeSpans`/`bestDeathOffset`/
      `nameLivesByDeaths`/`ownersFromLives` dans `killcollector` aurait été un second
      décodeur du même fil des morts — la phrase « deux décodeurs du même fait
      divergeraient » revient 3 fois dans le code de ce paquet. Solution : exporter
      `replay.ResolveSlotXUID` (nouveau fichier `killpos_bridge.go`, ~15 lignes,
      compose `indexBySlot`+`buildOwners` EXISTANTS, zéro ligne de logique touchée,
      `fire=nil` — pas de fermeture par tir, version la PLUS CONSERVATRICE) — SEUL
      fichier neuf dans `internal/analysis/replay/**`, famille tenue verrouillée par
      la session G précédente pour ne pas percuter le lot F CONCURRENT ; jugée
      touchable ici parce que (i) le lot F est livré, plus concurrent, (ii) l'ajout
      est additif pur et testé isolément (3 tests, `killpos_bridge_test.go`),
      (iii) l'alternative (réimplémenter) était le risque de corruption silencieuse
      que ce chantier existe pour éviter. 4e lecture disque nécessaire, non comptée
      par l'estimé G.2 : `replay.ScanFilmDeaths` (le fil des morts du REJEU — même
      horloge que le kill-feed de `killsource`, mais un lecteur SÉPARÉ, déjà
      précédenté par le lot F équipement pour la même raison : `replaybuild` n'ouvre
      aucune base).
      Fichiers : `internal/sync/killcollector/positions.go` (nouveau, ~300 L —
      `collectPositions`/`resolveMapBounds`/`buildPositionRows`/`writeChunksToTempDir`
      /`killRefsFromDeaths`/`toKillPositionRows`) ; `collector.go` (+~25 L — champs
      `mapNames port.ReplayMapNameRepo`/`mapBounds *filmdec.MapQuantCatalog`, wither
      `WithPositionCapture` gardant `NewKillSourceCollector` à 5 paramètres, appel
      `collectPositions` après `collectShots` dans `collect()`, même best-effort :
      un échec de positions n'invalide jamais la passe de morts déjà écrite) ;
      `internal/analysis/replay/killpos_bridge.go` (nouveau, `ResolveSlotXUID`) ;
      `internal/persist/kill_position_persister.go` (nouveau — réutilise l'INSERT
      déjà existant de `shared_persister.go`, refuse `killer_xuid` vide et
      `match_id` incohérent, AUCUNE entrée nouvelle dans l'allowlist
      `no_art_patterns_test.go` : INSERT pur, rien à y déclarer) ; capability
      `film.kill_positions` (`games/adapter.go` + `capabilities.go` +
      `capabilities.toml` halo_infinite SEUL — H5 absente, positions déjà natives
      via `match.events.spatial` = supported) ; `cmd_backfill_killsource.go`
      (+~45 L — `positionCaptureDeps`/`staticSharedReader`, câblée PAR DÉFAUT dans
      `passeDesFilms`, best-effort si catalogue/metadata indisponibles — CLAUDE.md
      règle 11 : pas de feature prête livrée OFF).
      Décision `KillSourceDecoderRev` : PAS bumpée. `kill_positions` n'a AUCUNE
      colonne de révision (clé fonctionnelle `match_id, killer_xuid, time_ms` +
      `written_at` seul, cf. migration G.2) — le rev suit le décodage des MORTS,
      bumper aurait forcé un re-décodage COMPLET (morts + tirs + positions, jusqu'à
      1h15 sur le corpus mesuré) pour un gain qui ne concerne QUE les positions.
      Les nouveaux matchs capturent les positions SANS bump (le code est actif dès
      ce commit) ; le rattrapage des matchs déjà décodés passe par la commande
      EXISTANTE `levelup backfill-killsource --force` (redécode tout, y compris les
      positions désormais câblées) — décision opérateur explicite plutôt qu'un bump
      qui aurait aussi fait re-décoder 8 anciens matchs/cycle dans un futur hook
      post-sync. Verrou d'ordonnancement (crédit-seul après film efface la source,
      signalé par G.2) : VÉRIFIÉ NON APPLICABLE à `kill_positions` — cette table n'a
      qu'UN SEUL producteur candidat par titre (Infinite : le film SEUL, aucun
      chemin crédit-seul n'a de position à offrir ; H5 : natif seul), contrairement
      à `match_kill_events` qui a un second producteur crédit-seul avec lequel la
      passe film doit composer. Aucun verrou nouveau nécessaire ; la précondition
      « serveur arrêté » déjà en vigueur pour `backfill-killsource` (ADR 0013)
      couvre le seul risque réel (deux writers concurrents sur `shared`).
      Capability LECTURE `match.events.spatial` : NON touchée pour Infinite (reste
      `not_exposed`) — elle gouverne la lecture canonique, `film.kill_positions`
      gouverne la CAPTURE ; aucun consommateur ne lit `kill_positions` pour Infinite
      dans cette passe (G.3, hors périmètre G.2bis).
      Tests : pont chunks synthétiques → répertoire (round-trip verbatim, round-trip
      zlib, refus de trou de séquence, chunk déclaré-mais-vide traité comme un trou,
      cleanup) ; composition pure (filtrage des identités à deux xuids résolus,
      traduction en lignes avec positions partielles) ; CINQ refus propres qui NE
      TENTENT AUCUNE ÉCRITURE (capability absente, collecteur non câblé, aucune
      identité résolue, carte hors catalogue, film illisible) — preuve par un
      `acquireShared` qui PANIQUE s'il est appelé (panic non rattrapée = test en
      échec) ; persister `:memory:` (5 cas dont l'idempotence append-only : 2 passes
      → `kill_positions_latest` ne montre que la ligne la plus RÉCENTE par clé, la
      TABLE garde les deux — et, découverte notée, le dédoublonnage est PAR LIGNE
      et pas par passe entière comme `match_kill_events` : une 2e passe qui retrouve
      MOINS de kills n'efface pas les positions des kills qu'elle n'a pas
      retrouvés) ; 1 test d'intégration sur film RÉEL
      (`positions_integration_test.go`, gate `KILLSOURCE_FIXTURES` identique au
      test historique de G.1/G.2, catalogue de bornes RÉEL chargé via PathResolver
      — DONNÉE DE RÉFÉRENCE VERSIONNÉE donc disponible même sans `data/` de
      travail) — NON REJOUÉ dans ce worktree (contrainte de session, aucun
      `KILLSOURCE_FIXTURES`) : setup vérifié sur pièces (catalogue réel chargé,
      79 cartes, schemaVersion=1 conforme), mais le CHEMIN POSITIF (chunks réels →
      positions non vides) n'a jamais tourné dans cette session — à rejouer avant
      confiance totale sur la géométrie produite. Le pont et la composition, eux,
      sont vérifiés par les tests synthétiques ET par le fait que chaque brique
      réutilisée (ScanFilmBipedPositions, ScanFilmClockOrigin, ScanFilmPlayerIndices,
      ScanFilmDeaths, buildOwners, BuildKillPositions) porte déjà sa PROPRE
      couverture éprouvée dans son paquet d'origine.
      Gate G.2bis : `go build ./...` + `go vet ./internal/sync/...
      ./internal/analysis/filmdec/... ./internal/persist/...` propres ; `go test
      ./internal/sync/... ./internal/analysis/... ./internal/persist/...
      ./internal/migration/... -count=1` 0 échec (killcollector, analysis/replay,
      persist, games — tous verts) ; `-tags=integration -p 1 ./internal/migration/...
      ./internal/persist/... ./internal/platform/duckdb/... ./internal/sync/...` :
      REJOUÉ en entier — SEUL `internal/platform/duckdb` échoue, avec 24 échecs
      `team_0_rounds_won` (pas 19 : le compte du 30/08 matin était incomplet — 5
      échecs supplémentaires dans `match_repos_test.go`, MÊME cause exacte, MÊME
      fichier pristine, détail au thought_log) — 0 échec imputable à ce lot,
      `internal/sync` (dont `killcollector`) et TOUS les autres paquets ciblés 100 %
      verts, `killcollector` inclus (3,9 s, le test réel se SKIPPE proprement sans
      `KILLSOURCE_FIXTURES` dans ce worktree).
- [x] G.3-préparation (S) — guide Reddit rendu lisible, RENDU le 30/08. Extraction du
      corps du post (`t3_xfcz4n-post-rtjson-content` — piège noté : un premier
      conteneur `-post-rtjson-content` SANS préfixe existe aussi mais sert le flair
      auteur, pas le post) via script Node jetable en scratchpad, jamais commité.
      Document produit : `.ai/V7.5/killweapon/REFERENCE_VARIANTES_ARMES_REDDIT.md`
      — 22 variantes sourcées (auteur, date, URL), rapprochées de notre registre par
      nom (`games/weapons/registry.go` + `killicon/data/rules.tsv`, vérifié sur
      pièces : 22/22 déductibles, 0 UNKNOWN). Réserve méthodologique consignée
      (section dédiée du document) : nos `source_tag` identifient l'ARME, pas le
      TUNING de mode Fiesta/Yappening que documente ce guide — un kill à une
      variante et un kill à l'arme vanilla partagent le même `weapon_key`,
      illisible après coup dans `match_kill_events`. Zéro fichier de code touché
      (vérifié : `git status` ne montre que le `.md` neuf).
- [x] **DEC-8 (utilisateur, 2026-08-30 soir)** : G.3 RÉDUIT à un POC vue match,
      cadré MOT POUR MOT par l'utilisateur : « sur la page match view on peut,
      comme si c'était un POC, mettre le nombre de kills par armes sur la
      distance et indiquer la distance moyenne pour chaque arme, tout ça pour
      chaque joueur. Pour le moment c'est tout ce qu'on va faire au niveau de la
      lecture de la distance. » Périmètre FERMÉ par cette phrase : vue match
      UNIQUEMENT, par joueur, kills par arme × distance + distance moyenne par
      arme — RIEN d'autre. Explicitement FERMÉS : agrégat multi-matchs, portée
      par arme (confrontation RRR — le reste de G.3 « plein » ci-dessous), et
      **arme ET distance de l'ASSISTANT** (G.0 avait signalé la distance de
      l'assistant comme mesurable via `assist_xuid`, 22 708 lignes — DEC-8 la
      ferme explicitement, comme l'arme de l'assistant : ce POC ne lit QUE le
      TUEUR).
      CORRECTION portée par la même décision, sur une ligne distincte du sujet
      distance : les PARTS DE DÉGÂTS tueur/assistant (`killer_damage_pct`/
      `assist_damage_pct`) sont DÉJÀ affichées PAR KILL dans le kill feed du
      rejeu depuis le 2026-08-24 (`ReplayKillFeed.tsx:489`, i18n
      `killFeedKillerShare`/`killFeedAssistShare`,
      `apps/web/src/features/match-replay/i18n.ts:39-41` — sur une rangée
      assistée, la part de l'assistant s'affiche seule, celle du tueur sort de
      la rangée, cf. `.ai/thought_log.md:3949`). La réserve écrite dans
      `.ai/V7.5/killweapon/GUIDE_KILLSOURCE.md:670-674` (chemin kill-event film
      → `KillerPercentageDamageDone`/`AssistantPercentageDamageDone` NON
      démontré ; valeurs NON bornées à 100, 1,7 % vont jusqu'à 228 — verrouillé
      aussi côté schéma, `internal/migration/steps_shared_kill_events.go:322-324`)
      NE VAUT QUE pour (a) présenter ces nombres comme des DÉGÂTS EXACTS en HP,
      ou (b) les AGRÉGER (moyenne/somme sur plusieurs kills) — jamais pour la
      part BRUTE par kill déjà en prod, qui est un simple affichage d'un champ
      mesuré, sans interprétation ajoutée. Aucune ligne du plan n'affirmait le
      contraire noir sur blanc ; cette entrée sert de repère écrit pour ne pas
      le laisser se former plus tard par confusion avec la réserve distance
      (G.0, valeurs non plafonnées elle aussi mais sur un TERRAIN différent).
- [x] G.3-POC (S) — RENDU le 30/08 (agent Sonnet, worktree dédié), périmètre
      DEC-8 ci-dessus. Chaîne complète, gate par gate :
      **Go lecture** : `port.KillDistanceRepository.LoadMatch(ctx, matchID)`
      (`internal/port/repository_data.go`, à côté de `PlayerPositionsRepository`,
      même forme mono-match) → `duckdb.KillDistanceRepo`
      (`internal/platform/duckdb/kill_distance_repo.go`, nouveau) : jointure
      `match_kill_events_latest` × `kill_positions_latest` sur
      `(match_id, killer_xuid = feed_killer_xuid, time_ms)`, `GROUP BY
      (feed_killer_xuid, time_ms) HAVING count(DISTINCT source_tag) = 1` (même
      garde d'unanimité double-kill que Q21b), `publishable` requis (lecture PAR
      KILL, pas un agrégat qui tolère l'individuel faux — contrairement à
      `KillSourceClassRepo`), positions NULL des deux côtés exigées non-NULL
      (une position partielle — un seul côté connu, cf.
      `internal/analysis/replay/killpos.go:58-59` `Killer`/`Victim *Vec3` — est
      exclue, jamais approchée). Distance = `hypot3D` en Go (jamais en SQL, même
      politique que `KillSourceClassRepo` : le SQL ne connaît que des nombres).
      Résolution weapon_key : RÉUTILISE le classificateur existant
      (`port.KillSourceClassifier`/`killicon.Lookup`, même injection que
      `KillSourceClassRepo`) — mais SANS le filtre anti-double-comptage
      « hors arsenal seul » : nouvelle fonction
      `resolveWeaponKeyLabelsAny` (`weapon_resolver.go`, à côté de
      `resolveOffArsenalKeys`) qui résout TOUTE arme (à feu du registre COMME
      hors arsenal) par `weapon_key` — sans jointure `weapon_ids` (qui aurait
      fan-out sur les armes à variantes multiples), donc AUCUNE nouvelle table
      de résolution, juste une requête de plus sur `weapons`/`weapon_name_labels`
      déjà existantes. 9 tests `:memory:` + vraies migrations
      (`kill_distance_repo_test.go`, tag `integration`, harnais réutilisé de
      `killsource_class_repo_test.go` même package) : moyennes exactes (2
      joueurs, 2 armes, distances connues), clé D'ARSENAL qui remonte
      (`hinf_br75`, LA différence de comportement avec `KillSourceClassRepo`),
      kill sans position exclu, position partielle exclue, double kill à armes
      divergentes exclu en entier, passe non publiable exclue, table vide →
      zéro ligne zéro erreur, classificateur nil → rien, matchID vide → refus
      (jamais de scan complet).
      **Go service** : `domain.MatchKillDistancePlayer`/`MatchKillDistanceWeapon`
      (nouveau fichier `domain/match_view_kill_distance.go` — PAS dans
      `match_view.go`, déjà 805 L) ; `MatchCombatTab.KillDistanceByWeapon`
      (`omitempty`) ; `MatchViewService.WithKillDistanceRepo` (nil-safe, même
      doctrine que `WithKillSourceRepo`) ; chargement dans l'errgroup
      (`match_view_data_loaders.go`, `goLoad` — WARN automatique sur échec,
      c'est la brique qui couvre « slog sur échec de lecture ») ; assemblage
      direct `combat.KillDistanceByWeapon = d.killDistances` (pas de
      transformation supplémentaire, contrairement à `FragDistribution`).
      **Gate de capability — ÉCART ASSUMÉ sur la consigne initiale** : câblé sur
      `games.CapFilmKillSource` (`killDistanceRepoFor`,
      `internal/api/wire/registry_pages.go`, miroir exact de
      `killSourceClassRepoFor`), **PAS** `film.kill_positions` comme la mission
      le nommait. Vérifié sur pièces AVANT de coder (règle 4) :
      `games/adapter.go:160-166` et
      `config/titles/halo_infinite/mappings/capabilities.toml:112-114` disent
      NOIR SUR BLANC que `film.kill_positions` « GOUVERNE LA CAPTURE (l'écriture),
      PAS LA LECTURE ». L'utiliser pour gater CETTE lecture aurait en plus
      cassé Halo 5 EN PRATIQUE : `kill_positions` y est natif
      (`match.events.spatial = supported`) mais H5 ne pose JAMAIS
      `film.kill_positions` dans sa capability map (c'est une clé Infinite-only,
      le film est son SEUL besoin de capture) — gater dessus aurait masqué une
      donnée déjà disponible pour H5. `CapFilmKillSource` est la capability dont
      CE lecteur dépend RÉELLEMENT (même table `match_kill_events_latest`, même
      classificateur que `KillSourceClassRepo`) ; `match.events.spatial` a
      aussi été écarté (gouverne le pipeline `canonical.MatchEvent` CROSS-TITRE
      de la timeline, `games/halo_infinite/events.go:130-151`, un axe distinct
      — le promouvoir aurait fait mentir le message statique de
      `infiniteEventLimitations()`, qui dit encore à raison que la timeline
      CANONIQUE ne branche pas les positions). Décision documentée dans le code
      (commentaire `killDistanceRepoFor`) et ici — micro-décision tranchée par
      les conventions du dépôt (skill `plan-execution` §3), pas déléguée.
      **Web** : `MatchKillDistanceSection.tsx` (nouveau,
      `features/match-view/`, gabarit structurel `MatchObjectivesSection.tsx`)
      — table groupée par joueur (en-tête `{gamertag} — {mesurés}/{total} kills
      mesurés`, gamertag/total résolus depuis le scoreboard déjà chargé,
      JAMAIS dupliqués côté backend), une ligne par arme (libellé FR/EN via la
      locale + repli `weapon_key`, pastille de comptage `bg-primary/10`,
      distance moyenne locale-aware + plage min–max entre parenthèses si
      `measured_kills > 1`), badge « POC » (`bg-warning/10`), footnote réserve
      de couverture. Rendu `null` si aucune donnée (pas de cadre vide — cas de
      la quasi-totalité des matchs tant que le backfill de masse n'a pas
      tourné). Placé en `summary` tab (il n'existe PAS de tab « Combat »
      littéral — vérifié sur pièces, seuls `summary`/`chronology`/`players`
      existent, `MatchViewPage.tsx:75-78` — la vue backend `combat_tab`
      alimente le tab `summary` ; `MatchFragCard`, « stats d'armes » la plus
      proche, y vit aussi) juste après `MatchFragCard`. Tokens sémantiques
      SEULEMENT (`primary`/`warning`/`info`/`muted-foreground`, 0 hex). i18n FR
      + EN typées (`Record<Locale,T>`, `features/match-view/i18n.ts`,
      formatteurs `Intl.NumberFormat('fr-FR'|'en-US', {minimumFractionDigits:1,
      maximumFractionDigits:1})` — virgule FR / point EN vérifiés par test).
      Contrat régénéré (`make openapi-gen` + `make generate-types`) + alias
      manuels `apps/web/src/lib/api/types.ts` (pattern existant, pas de
      réécriture des champs).
      **Tests** : 9 Go intégration (ci-dessus, tous verts) + 8 web
      (`MatchKillDistanceSection.test.tsx` : rendu nominal FR, repli
      `weapon_key`, pas de plage sur 1 seul kill, locale EN complète, repli
      xuid hors scoreboard, 3 états vides null/undefined/[]) — 265/265
      `vitest match-view` (257 pré-existants + 8 neufs, 0 régression).
      **Gate G.3-POC** : `go build ./... ` + `go vet ./internal/...` propres ;
      `go test ./internal/platform/duckdb/... ./internal/service/...
      ./internal/domain/... ./contracttest/... -count=1` 100 % vert ;
      `go test -tags=integration -p 1 ./internal/platform/duckdb/... -count=1` :
      SEULS les mêmes 24 échecs `team_0_rounds_won` PRÉ-EXISTANTS documentés en
      G.2bis (fichiers jamais touchés par ce POC : `match_repos_test.go`,
      `player_matches_repo_test.go`, `pool_migration_test.go`) — 0 échec
      imputable ; `npx vitest run src/features/match-view` 265/265 ; `npm run
      typecheck` propre.
- [ ] G.3 (L, plein) — PORTÉE + narratif : agrégat par tag × carte, confrontation
      RRR, moteur narratif. NON ENGAGÉ, désormais fermé par DEC-8 (portée par
      arme = hors scope de ce cadrage, pas seulement conditionné à la donnée) —
      resterait à rouvrir par une décision utilisateur ultérieure, pas une
      simple question de cuisson de masse. La capture live reste câblée
      (G.2bis) si ce chantier rouvre un jour.
- [x] **G.4 — Assistances empilées (vue match) : DÉJÀ LIVRÉ, ZÉRO CODE NEUF**
      (demande du 2026-08-30 soir, agent Sonnet, worktree dédié). Cadrage
      utilisateur MOT POUR MOT : « comme le graphe de barres empilées de la page
      match view : on compte les assistances totales et on met des barres
      empilées pour dire qui on a assisté et combien de fois. » L'élaboration du
      pilote au moment de la mission (carte dédiée, dénominateur d'honnêteté PAR
      JOUEUR contre le scoreboard, infobulle « part moyenne », nouvelle chaîne
      Go+web calquée sur `kill_distance_repo.go`) N'EST PAS dans les mots de
      l'utilisateur — à distinguer du besoin vérifié ci-dessous, cf. Découvertes.
      VÉRIFIÉ SUR PIÈCES (règle 4) AVANT toute ligne de code : le graphe demandé
      EXISTE déjà, livré et branché depuis le 2026-08-25 par un chantier
      antérieur (« notion5(C) », commits `5bf2e6128` match + `48ca9b065`
      escouade/Synergies + `97ae1cefc` correctifs web — TOUS ancêtres de HEAD
      sur cette branche, `git merge-base --is-ancestor` vérifié), donc PAS par ce
      plan ni cette session. Correspondance point par point avec le cadrage :
      « chaque joueur a sa barre = total d'assistances MESURÉES » = 1 barre par
      ASSISTANT, longueur = somme des segments = `assist_count` sous
      `publishable AND assist_known` (Q21d) ; « empilée par BÉNÉFICIAIRE (le
      coéquipier assisté) » = segments empilés par `feed_killer_xuid` (le tueur
      crédité — nécessairement un coéquipier) ; « nombre d'assistances envers ce
      coéquipier » = `assist_count` par paire (assistant, tueur), `GROUP BY
      assist_xuid, assist_gamertag, feed_killer_xuid`.
      CHAÎNE DÉJÀ EN PROD sur cette branche — Go :
      `platform/duckdb/match_view_repo_assist_pairs.go` (Q21d, sur
      `KillEventsCanonicalTable` = `match_kill_events_latest`, MÊME table que
      tout le LOT G) → `domain/match_assist_pairs.go`
      (`MatchAssistPairRaw/Pair/ScopeRaw/Pairs`) → `domain/match_view.go:456-460`
      (`MatchCombatTab.AssistPairs`) → `port/repository.go:257-262`
      (`GetMatchAssistPairs`) → `service/match_view_builders_assists.go`
      (`buildAssistPairs`, gamertag du tueur résolu depuis le scoreboard) →
      `service/match_view_data_loaders.go:174-176,428` (chargement errgroup
      inconditionnel, best-effort — même patron que `d.killDistances`) → contrat
      `api/openapi.yaml:14945,15071-15072` (+ pendant escouade
      `SquadAssistPairs:22207,24176-24177`) → `lib/api/types.ts:1770,1802`. Web :
      `features/match-view/MatchAssistChart.tsx` (barres empilées horizontales,
      `BarStackedChart`, MIROIR du graphe des antagonistes, note « dont N
      volées » quand `assist_damage_pct > killer_damage_pct`) + `_chartSeries.ts`
      (`assistStackedSeries`/`assistStolenLookup`) + `i18n.ts:437-440,731-734`
      (FR/EN typées) — monté dans `MatchViewTabPlayers.tsx:23-24,77-90` (onglet
      Joueurs, section Duels & confrontations, juste sous les Antagonistes) via
      `MatchViewPage.tsx:222,423`.
      LA QUESTION `publishable` DE LA MISSION EST DÉJÀ TRANCHÉE PAR CE
      PRÉCÉDENT, PAS PAR fragdist/`KillSourceClassRepo` : Q21d EXIGE
      `publishable = TRUE` (`match_view_repo_assist_pairs.go:82`), aligné sur
      `KillDistanceRepo` (G.3-POC) et PAS sur `KillSourceClassRepo` — une PAIRE
      nomme DEUX joueurs (lecture ligne à ligne), contrairement au tally
      mono-joueur de `KillSourceClassRepo` qui tolère l'agrégat non publiable.
      Même raisonnement tranché deux fois indépendamment (G.3-POC et ce
      « notion5(C) » antérieur) : cohérent.
      VÉRIFICATION REJOUÉE dans ce worktree, aujourd'hui (pas seulement lue à
      l'historique git) : `go build ./...` + `go vet ./internal/...` propres ;
      les 7 `TestQ21dAssistPairs_*` (`platform/duckdb`), les
      `TestBuildAssistPairs_*` (`service`) et `TestBuildSquadAssistPairs_*`
      (`service/teammates`) verts ; `go test ./internal/platform/duckdb/...
      ./internal/service/... ./internal/domain/... ./contracttest/...
      -count=1` 100 % vert ; web `npx vitest run src/features/match-view`
      265/265 (dont les 23 tests `MatchAssistChart.test.tsx` +
      `_chartSeries.test.ts` ciblés) ; `npm run typecheck` propre ; `gofmt -l
      apps/go-api/internal/` vide.
      DÉCISION : ZÉRO code neuf. Construire une deuxième chaîne Go+web pour
      recalculer le MÊME agrégat depuis la MÊME table avec le MÊME filtre
      aurait violé CLAUDE.md règle 6 (≤ 2 copies d'un même pattern — ici une
      copie littérale du composant lui-même), règle 7 (0 code mort — l'une des
      deux chaînes ne serait jamais appelée par l'écran) et l'interdit explicite
      de la mission elle-même (« toucher aux lots livrés hors besoin direct ») :
      le besoin direct est déjà satisfait par le lot livré. Deux idées du
      cadrage du pilote restent NON posées, faute de besoin utilisateur vérifié
      pour les justifier (consignées en Découvertes §8, pas traitées) : moyenne
      de `assist_damage_pct` en infobulle (« part moyenne ») et un dénominateur
      d'honnêteté PAR JOUEUR contre le total scoreboard.
      LIMITE HONNÊTE : pas de vérification visuelle navigateur (pas de `data/`
      dans ce worktree, aucun match réel à afficher) — la preuve tient sur les
      tests d'intégration REJOUÉS contre le VRAI schéma
      (`migration.EnsureMatchKillEvents`, même harnais que le commit
      `5bf2e6128`) et sur le branchement de bout en bout vérifié par lecture de
      code (aucun maillon manquant entre la table et l'écran).
- Gate G.0 : PASSÉ. Gate G.1 : PASSÉ (détail ci-dessus). Gate G.2 : PASSÉ pour le
      périmètre RENDU (append-only). Gate G.2bis : PASSÉ (détail ci-dessus) — la
      capture live n'est plus `[!]`. Gate G.3-préparation : PASSÉ. Gate G.3-POC :
      PASSÉ (détail ci-dessus). G.3 (plein) : non engagé, fermé par DEC-8. Gate
      G.4 : PASSÉ — vérification seule (détail ci-dessus), aucun code de
      production modifié.

## 4. Hors périmètre (fermé)
- R9/D6 (niveau 2 barres Escouade) — Découvertes.
- Toute écriture DB / tout backfill (lot E : décision utilisateur).
- Tout code équipement (lot F : validation 8b attendue).
- Véhicules/tourelles Halo Infinite au sunburst (déjà consigné au registre kills-hors-arme).

## 5. Commits — aménagement assumé
Règle 8 du contrat (1 commit/étape) SUSPENDUE : le worktree porte 3 lots non committés et
`68e44770b` a déjà scindé le lot kills-hors-arme (réserve au registre : « décision
utilisateur avant tout commit »). AUCUN commit dans cette passe. Le §7 donne la carte
fichier→lot pour l'arbitrage. Après arbitrage : stage par CHEMINS NOMMÉS, `git add -p`
obligatoire sur les fichiers à 2 lots.

## 6. Gates
- Par lot : commandes ci-dessus, jouées par l'agent du lot, sorties dans son rapport.
- **Gate final VF (pilote)** : purge `node_modules\.tmp` → `npm run typecheck` →
  `npm run lint` → `make test-web` → `cd apps/go-api && go test ./... && go vet ./...` →
  greps délivrance (fmt.Println, hex sous features/, TODO) → parité i18n.
  `-tags=integration` : non requis (aucun fichier persist/sync/migration touché par la
  vague ; l'intégration duckdb a été rejouée verte AUJOURD'HUI pour le lot 0).
  `make gate-push` : différé au pré-merge (la CI de branche, autorité, est de toute façon
  ROUGE depuis le 2026-08-28 16:05, AVANT ces lots — découverte à traiter séparément).

## 7. Carte des lots non committés du worktree (pour l'arbitrage des commits)
- **kills-hors-arme** (Go) : `domain/frag_distribution.go`, `killicon/data/rules.tsv`,
  `games/weapons/registry.go(+test)`, `platform/duckdb/weapon_resolver.go`,
  `service/{explorer_service,explorer_target_frag_distribution,fragdist/*,
  match_view_builders_combat,match_view_frag_distribution_test,session_page_frag_distribution,
  session_page_service,synthesis_service,timeseries_service}.go`,
  `service/teammates/{squad_frag_distribution_test,teammates_squad_charts_weapons_perf}.go`,
  `api/wire/registry_pages_explorer.go` + nouveaux `killsource_registry*.go`,
  `killsource_class_repo*.go`, `port/kill_source_class.go`, `service/killsourceload/`,
  `fragdist_killsource_test.go`, `off_arsenal_guard_test.go` ;
  config `weapon_names.toml` ; web `fragClass.ts(+guard)`, `frags.toml(+généré)`,
  `weaponRoleInsight.ts(+test)`.
- **score-manches (suite E8)** : `domain/{squad,teammates}.go`,
  `platform/duckdb/{queries_squad,squad_repo}.go`,
  `service/teammates/{teammates_service_assets,teammates_extra_test,teammates_squad_mode_test}.go`,
  `api/wire/registry_pages_home.go`.
- **rejeu/sons (autre session)** : `features/match-replay/*` (10 modifiés +
  `ReplayObjectiveMark.tsx`, `objectiveMark.ts(+test)`, `skullSound.ts(+test)`),
  `lib/fda.ts(+test)`, `static/sounds/halo_infinite/objective_skull_*.wav` (10).
- **Fichiers à DEUX lots (git add -p)** : `service/teammates/teammates_service.go`
  (kills + manches), `.ai/thought_log.md` (rejeu + kills).
- **Ce plan (retours-0829)** ajoute : les fichiers des lots A/B/C/D ci-dessus + ce fichier
  + entrées journal/registre. Chevauchement avec kills-hors-arme : `fragClass.ts(+guard)`
  (A.1/A.5 par-dessus le WIP) — même famille de sujet, arbitrage simple.

## 8. Découvertes (à consigner, PAS à traiter ici)
- CI `feat/v75` ROUGE depuis le push du 2026-08-28 16:05 (AVANT tous ces lots) — à
  diagnostiquer avant tout push.
- LOT G.3-POC (30/08) — `make go-api-lint` (ratchet `--new-from-merge-base=origin/main`)
  remonte 2 issues, AUCUNE dans les fichiers de ce POC : (a) `funlen` sur
  `service/teammates/teammates_service_assets.go:238` `buildSquadMatchHistory` (82 > 80 L)
  — fichier du lot score-manches, jamais touché ici ; (b) `unparam` sur
  `service/match_view_data_loaders.go:706` `loadMatchKillSourceClasses` (le retour error
  est toujours nil) — fonction PRÉ-EXISTANTE du lot kills-hors-arme/V2.1, byte-identique
  avant/après ce POC (vérifié). Les deux remontent seulement parce que le ratchet compare
  au dernier merge-base `origin/main`, et ce worktree porte plusieurs lots non mergés — pas
  une régression de ce POC.
- `spartan_ability`/`unattributed` ΔE 6,89 (pré-existant, sous seuil 8) — traité en
  exception datée par A.5, à re-trancher si refonte palette.
- `ecs_table.tsv:800` : doc inversée (`i27 charges-remaining` présentée « en réserve »
  alors que RÉFUTÉE le 2026-08-15) — 1 ligne à corriger dans un lot doc.
- `buildHomeScoreLabel` sans appelant de prod (code mort testé) — déjà noté au plan
  score-manches, lot dédié.
- `weapon_kills_v3` : table shadow jamais peuplée ici (0 ligne) — statuer son avenir.
- D6/R9 : niveau 2 sur les barres Escouade (`squadFragBreakdownChart.ts`) — option M si
  l'utilisateur confirme le besoin.
- Accueil (`match-card`) et page coéquipiers toujours en points (réserve `[!]` du plan
  score-manches, décision utilisateur attendue) — rappelée ici pour visibilité.
- LEÇON DE MÉTHODE (gate F.0, 29/08 soir) : tout gate chiffré doit pré-écrire sa
  POPULATION (large vs stricte/affichée) et sa règle de MARGE (que faire d'un passage
  à une unité de comptage près) — sinon deux lectures légitimes coexistent après
  mesure. Constaté sur F.0 (30,2 % à 0,3 épisode près en large ; 39,3 % en stricte),
  objection levée par la session équipement, tranché sur la population d'affichage.
- BUG PRÉ-EXISTANT DÉCOUVERT (LOT G, 30/08) : 19 tests `-tags=integration` échouent en
  l'état ACTUEL DU DÉPÔT (fichiers pristine, `git status` vide — pas un artefact
  d'arbre partagé) — `internal/platform/duckdb/player_matches_repo_test.go` (18 cas) +
  `pool_migration_test.go` (1 cas) construisent `match_registry` via une VALUES-list
  SQL brute aliasée `r(...)` qui n'a jamais reçu `team_0_rounds_won`/
  `team_1_rounds_won` quand ADR 0032 (score-manches) les a ajoutées en prod — les
  requêtes réelles qui les sélectionnent échouent au bind contre la fixture. Tâche
  spawnée (`task_fb60be2a`) plutôt que corrigée ici (hors périmètre G, fichiers
  jamais touchés par ce lot).
- `internal/games/halo_infinite/migrations/steps.go` dépasse largement le seuil de
  500 lignes (1 483 aujourd'hui) — accumulateur historique de migrations DDL
  individuellement petites, dette gelée déjà connue de ce paquet (voir aussi
  `steps_appendonly_misc.go`, qui reste sous le seuil). Non traité : ajouter une
  migration de plus n'aggrave pas qualitativement une dette déjà documentée par la
  structure même du fichier (chaque step est nommé, daté, isolé).

## 8bis. Journal complémentaire (nuit du 29 au 30/08)
- Lots F.1-F.3 et G livrés par reprises Sonnet séquentielles (les deux lancements Opus
  sont morts au démarrage sur quota, reset 1h30) — rapports détaillés au §LOT F / §3bis,
  entrées thought_log posées par les agents.
- VF de clôture (30/08 midi) : web typecheck (cache purgé) OK + vitest COMPLET
  **5 474 tests / 0 échec** ; Go `build -p 4` OK + `vet ./...` OK + `go test ./...`
  vert HORS les 2 pré-existants documentés (archlint TestNoLocalLongestRun, himap
  timeout). Deux flakes d'environnement rencontrés et écartés sur preuve :
  épuisement de ressources du linker CGO quand build complet + vitest complet tournent
  ensemble (relance bridée `-p 4` : OK), et `ops/migrate [build failed]` transitoire
  (vert isolé, 2 fois).
- G.2bis (câblage capture positions dans killcollector) RENDU le 30/08 (agent Sonnet,
  worktree dédié) — détail complet au §3bis. RESTE OUVERT : les 19 tests
  d'intégration `team_0_rounds_won` PRÉ-EXISTANTS (fixture du lot score-manches, tâche
  spawnée par l'agent G) ; re-cuisson des artefacts post-merge (commande au §LOT F) ;
  arbitrage des commits (§5/§7) ; vérifications visuelles utilisateur.

## 9. Reprise de session
Avancement = les cases de CE fichier + le journal §10. Reprendre à la première case non
statuée du premier lot non clos. Les rapports d'agents détaillés ne sont PAS dans le repo :
les file:ligne essentiels sont dans les items ci-dessus ; re-vérifier sur pièces avant de
coder (règle 4).

## 10. Journal
- 2026-08-29 : plan écrit ; lot 0 gates déverrouillés constatés (build+vet+service+api+
  contracttest+platform+intégration duckdb verts) ; vague 1 lancée (A Opus, B Sonnet,
  C Sonnet, D Haiku).
- 2026-08-29 ~16h : vague 1 LIVRÉE, gates par lot verts (A : 1 215 tests ciblés ; B : 759 ;
  C : 801 web + Go service/domain/contracttest, contrat régénéré ; D : typecheck+eslint).
  Incidents d'arbre partagé pendant la vague : E9/E10 committés par la session
  score-manches (embarquant au passage des fichiers du lot kills, encore), `openapi.yaml`
  remis à HEAD par un tiers entre deux régénérations du lot C, `squad/i18n.ts` stagé par
  un tiers, 4e lot « assistances » apparu, 5e chantier « backfill killsource online /
  remote films » apparu.
- 2026-08-29 ~16h45 : retour utilisateur — le chantier devait vivre dans un worktree DÉDIÉ.
  Correction : création `LevelUp-wt-retours-0829` (branche `wt/retours-0829`, base
  `7befe6192`), transfert patch+untracked du périmètre (kills + lots A-D + docs), retrait
  des lots étrangers du worktree dédié, retrait de mes fichiers EXCLUSIFS de l'arbre
  partagé. Sanity dans le dédié : `go build ./...` + `npm run typecheck` verts.
  `generated/frags.ts` régénéré (2 clés kills perdues par un revert croisé, réparées).
  V2.1 (labels locale) lancée ici ; V2.2 statuée `[!]` (pas de `data/` dans ce worktree).
- 2026-08-29 ~20h50 : V2.1 complétée (agent Sonnet coupé par quota à ~95 %, repris par le
  pilote — tout était posé, gates rejoués). **GATE FINAL VF VERT** dans le worktree dédié :
  Go `go build` + `go vet ./...` OK, `go test ./...` vert HORS les 2 échecs PRÉ-EXISTANTS
  documentés d'avant-chantier (archlint `TestNoLocalLongestRun` sur cmd/oddball-terrain,
  `internal/himap` en timeout) ; web `npm run typecheck` (cache purgé) OK, `npm run lint`
  0 erreur / 24 avertissements pré-existants, vitest COMPLET 530 fichiers / 5 457 tests /
  0 échec ; greps livraison propres (0 fmt.Println, 0 TODO, 0 hex de style, 0
  filepath.Join data). Correctif bloquant au passage : `killsource_class_repo_test.go`
  portait le littéral brut `'marche'` (garde archlint J4R-3, manqué par les gates
  par-paquet du lot kills) → constante `killscope.ReadPathFilmWalk`, archlint + tests
  d'intégration KillSourceClass rejoués verts.
- 2026-08-29 ~21h : INCIDENT DE MA FAUTE, réparé — mon nettoyage de l'arbre partagé avait
  reverti `timeseries_service_events.go`/`first_blood.go` (exclusifs C) en LAISSANT les
  3 hunks d'appel à 3 arguments dans les fichiers chevauchés `session_page_service.go` /
  `timeseries_service.go` → compilation cassée pour les autres sessions. Les 3 appels ont
  été remis à 2 arguments dans l'arbre partagé, `go build internal/service(+teammates)`
  vert. (La version 3 arguments vit intacte dans CE worktree.) Corrections utilisateur du
  soir intégrées : films 2025-2026 disponibles (cf. LOT E), 8a/8b validés (F.0 lancé),
  LOT G créé (headshot/distance/portée), agents G (Opus) + F.0 (Sonnet) en cours dans
  l'arbre partagé (data/), en lecture seule.
- 2026-08-30 : LOT F.1-F.3 exécutés dans ce worktree dédié (agent Sonnet), dans l'ordre, gate
  par gate. F.1 (Go) : jointure épisodes×kills pure et testée (15 tests synthétiques),
  câblage `replaybuild` réutilisant le décodage killsource déjà fait pour les morts neutres
  (une seule passe au lieu de deux), résolution d'identité hors ligne via le fil des morts du
  film (le pont slot→xuid publié dans l'artefact suffit — pas d'ouverture de base, contrat
  `replaybuild` intact). SchemaVersion 24, tous les points de pin retrouvés et mis à jour
  (const Go, garde de parité web, golden fixture, contrat OpenAPI, generated.ts) —
  `structure_test.go` et `replaySchemaLogic.ts` n'étaient PAS dans la liste donnée par le
  plan et ont été trouvés par grep, comme demandé. 418 sous-tests Go verts. F.2 (Web) : 2
  colonnes dans le tableau équipement existant, distinction 0/non-mesuré par
  `Coverage.KillsRead`, réserve mesurée FUSIONNÉE dans l'infobulle de groupe existante plutôt
  que dupliquée. 2067 tests web verts (match-replay+match-view). F.3 (badge) : le badge
  regarde le MEILLEUR ÉPISODE (pas la somme du joueur) — distinction vérifiée par test dédié
  (deux épisodes de 2 frags n'ouvrent pas de badge à 4). Auto-alimentation de
  `MatchImpactBadgesBar` sur le même artefact que `MatchEquipmentUsageSection`, rendu
  `NarrativeBadge` volontairement distinct des cartes de badges serveur (deux sources, pas une
  liste unique). 265 tests verts (match-view+components/feedback). Aucune re-cuisson
  (worktree sans `data/`) — commande post-merge notée au rapport de session. Découverte
  notée, non traitée : `MatchImpactBadgesBar` n'avait aucun test dédié avant ce lot (mocké en
  bloc par `MatchViewTabs.test.tsx`) — la couverture de sa logique de tri/valence reste à
  faire, hors périmètre F.1-F.3.
- 2026-08-30 : LOT G.1/G.2/G.3-préparation exécutés dans ce worktree dédié (agent Sonnet),
  dans l'ordre, gate par gate — fichiers Lot F (`internal/analysis/replay/**` sauf import,
  `internal/replaybuild/**`, `features/match-replay/**`, `MatchViewTabChronology.tsx`,
  `equipmentKillBadges*`, `MatchImpactBadgesBar.tsx`) intégralement respectés (0 fichier
  touché). G.1 (lecture headshot) RENDU complet, gates verts. G.2 (positions) RENDU
  PARTIEL — la dette bloquante `kill_positions` non-append-only est FERMÉE (migration +
  3 tests neufs, H5 prod préservé, testé via la vraie chaîne `RunForDB`) mais la capture
  live Infinite dans `killcollector` est DÉFÉRÉE `[!]`, justifiée sur pièces (détail
  §3bis) : les fournisseurs exportés de `replay.BuildKillPositions` sont tous disque-only
  et `killcollector` décode aujourd'hui en mémoire pure ; le pont identifié (écrire les
  chunks déjà en mémoire dans un répertoire temporaire, appeler les fonctions exportées
  telles quelles) est SÛR mais le tour complet (pont disque + nom de carte jusqu'à
  `MatchIdentities` + bornes monde + inversion slot→xuid + nouveau persister + nouvelle
  capability + tests d'intégration) est un lot à part entière — pas rushé pour ne pas
  risquer une corruption SILENCIEUSE de positions (aucun symptôme visible, exactement la
  classe de bug que ce chantier ADR 0026 existe pour éradiquer). Déclencheur de backfill
  DÉCIDÉ malgré tout (bump `KillSourceDecoderRev`, réutilise `backfill-killsource`
  existant sans nouveau flag) : décision indépendante de l'implémentation, vérifiée sur
  le code réel de `matchsAJour`. G.3-préparation RENDU : guide Reddit extrait (script
  Node jetable, jamais commité) et transformé en référence sourcée
  `.ai/V7.5/killweapon/REFERENCE_VARIANTES_ARMES_REDDIT.md`, 22/22 variantes rapprochées
  du registre par nom, réserve méthodologique majeure consignée (nos `source_tag`
  n'identifient pas le tuning Fiesta que documente le guide). G.3 (plein, portée+narratif)
  non engagé — conditionné à la capture live de G.2, pas seulement à l'append-only.
  Découverte : 19 tests `-tags=integration` pré-existants cassés (`team_0_rounds_won`
  absent d'une fixture VALUES-list, fichiers pristine, sans rapport avec ce lot) — tâche
  spawnée plutôt que corrigée ici.
- 2026-08-30 : LOT G.2bis exécuté dans ce worktree dédié (agent Sonnet) — câblage de la
  CAPTURE live des positions dans `killcollector`, dernière pièce déférée par G.2. Pont
  disque retenu (option a du plan) : `writeChunksToTempDir`, avec un refus sur trou de
  séquence non anticipé par l'estimé initial. Découverte majeure en cours de route : le
  pont slot→xuid n'était PAS un sous-produit de `MatchIdentities` comme supposé — c'est
  le pont interne (non exporté) d'`analysis/replay/owners.go`, qui a exigé d'exporter
  `replay.ResolveSlotXUID` (fichier neuf, additif pur, testé) plutôt que de le
  réimplémenter (risque de second décodeur du même fil des morts). Détail complet,
  fichiers touchés, décisions (KillSourceDecoderRev non bumpée, verrou d'ordonnancement
  vérifié non applicable à cette table) et gates : §3bis, item G.2bis, et
  `.ai/thought_log.md`. G.3 (portée + narratif) reste non engagé, désormais conditionné
  à une cuisson de masse plutôt qu'à la capture live (câblée).
- 2026-08-30 : LOT G.3-POC exécuté dans ce worktree dédié (agent Sonnet) — DEC-8
  (utilisateur) réduit G.3 à un POC vue match (kills par arme × distance + distance
  moyenne, par joueur, TUEUR seul), ferme le reste de G.3 (portée RRR, narratif,
  arme/distance assistant), et corrige la lecture de la réserve damage-pct (parts
  tueur/assistant déjà en prod par kill dans le kill feed du rejeu, réserve non
  applicable à l'affichage brut). Chaîne complète RENDUE : repo Go
  (`KillDistanceRepo`, jointure `kill_positions_latest`×`match_kill_events_latest`,
  9 tests intégration) → service (wiring, capability `film.kill_source` — ÉCART
  ASSUMÉ vs la consigne `film.kill_positions`, justifié sur pièces §3bis) → contrat
  régénéré → carte web (`MatchKillDistanceSection`, tab `summary` à côté de
  `MatchFragCard`, 8 tests). Gates : build+vet propres, suite ciblée 100 % verte,
  `-tags=integration` platform/duckdb SEULS les 24 échecs `team_0_rounds_won`
  pré-existants (G.2bis), `go test ./...` MODULE ENTIER seuls les 2 échecs
  pré-existants déjà documentés au VF du 30/08 (archlint, himap timeout) — 0
  régression. Web vitest match-view 265/265, typecheck propre. Détail complet :
  §3bis item G.3-POC, `.ai/thought_log.md`.
