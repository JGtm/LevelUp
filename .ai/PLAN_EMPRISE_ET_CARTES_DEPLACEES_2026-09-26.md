# Plan : onglet Usages → « Emprise » et cartes déplacées (Escouade) — 2026-09-26

> Sources, à lire avant tout lot, et qui FONT FOI pour le rendu :
> - maquette de l'onglet : https://claude.ai/artifact/6ZKxrug6cLzkxpYs2VWYaP (v21), copie
>   `.ai/V7.5/MAQUETTE_ONGLET_TACTIQUE_ESCOUADE_2026-09-26.html` ;
> - maquette des cartes déplacées : https://claude.ai/artifact/C3EWYk6cNEhNzhqnWqPo45 (v9), copie
>   `.ai/V7.5/MAQUETTE_TRI_CARTES_DEPLACEES_2026-09-26.html` ;
> - `.ai/HANDOFF_ONGLET_TACTIQUE_ESCOUADE_2026-09-26.md` (§3 goûts de l'utilisateur, §7 décisions) ;
> - `.ai/thought_log.md`, entrée « [2026-09-26] Escouade, onglet Usages… » et compléments.
>
> Contrat d'exécution : skill `plan-execution` (ordre strict, aucun item sans statut, zéro fix
> hors périmètre, découvertes consignées ici). Statuts : `[x]` fait, `[~]` couvert ailleurs (réf),
> `[!]` non fait (justification écrite). Aucune case vide à la clôture d'un lot.
>
> Statut du plan : **GO utilisateur le 2026-09-26** (« Ok pour "Emprise" … Go »). Exécution en cours.

## 0. Hors périmètre (décidé par l'utilisateur ou faute de données)

- **Bloc « Groupés ou isolés » et carte « Isolement, soirée après soirée »** : mis de côté le
  2026-09-26 (« on met de côté l'isolement et la riposte »). Rien n'est rendu à leur place ; la
  maquette v21 garde leur état pour plus tard. La riposte reste sur Synergies, inchangée.
- **Véhicules** (prises, pertes, temps à bord, frags depuis un véhicule côté adversaire) : aucune
  donnée en base (seulement dans l'artefact de rejeu). Demandés par l'utilisateur, ils font
  l'objet du lot **L7**, APRÈS L6. D'ici là, les lignes « Véhicules » ne sont PAS rendues (pas de
  ligne « à brancher » en production) ; L4/L5 construisent les cartes de sorte qu'une ressource
  « véhicules » s'y ajoute sans refonte (ressource = entrée d'une liste, masquée si absente).
- **Retrait de la route `/pages/squad/v2`** (hors `/engagement`, vivante) : morte côté web, mais
  hors de ce chantier → section Découvertes.
- Pages solo (Synthèse, Séries temporelles, Sessions) : seul changement = couleur des bonus (L0).

## 1. Décisions tranchées

### 1.1 Confirmée par l'utilisateur le 2026-09-26

- **D1 — Nom de l'onglet : « Emprise »** (EN « Map control »). « Tactique » (page
  `features/tactical`) et « Contrôle » (onglet de la vue match) sont déjà pris. Route :
  `squad/emprise`, et `squad/usages` redirige vers elle (liens existants préservés).

### 1.2 Tranchées (ne pas rediscuter pendant l'exécution)

- **D2 — Périmètre** : toutes les cartes de l'onglet et les cartes d'objectif de Contributions
  lisent les matchs de la **composition exacte** intersectés avec les matchs filtrés
  (`allSquadRows` ∩ `filteredMatches`) ; repli sur `filteredMatches` sans coéquipier sélectionné.
  Respecter `no_raw_squad_intersection_test.go` (mention `filterExactComposition`).
- **D3 — Armes spéciales** = prises sur les socles de niveau `puissance`
  (`match_pad_pickups_by_tier_latest`) ; armes de râtelier = niveau `terrain`. Jamais la classe
  `heavy/precision/other`.
- **D4 — Bonus** = camouflage + surbouclier ; la part porte sur les prises ATTRIBUÉES à un joueur
  (les socles vidés sans ramasseur ne comptent dans aucun camp, l'infobulle le dit).
- **D5 — « D'habitude » (ressources)** = les 10 soirées précédentes de la composition, sur les
  familles de mode jouées ce soir-là, chaque soirée comptée à part. Famille de mode = niveau 1 de
  la normalisation des modes (skill `halo-modes`), résolue par le même code que
  `SquadMatchHistoryRow.ModeUI` ; aucune nouvelle liste. Médiane tracée si ≥ 3 soirées précédentes.
- **D6 — « D'habitude » (objectif)** = les 10 soirées précédentes de la composition ayant **au
  moins 3 matchs à objectif, tous modes confondus** ; drapeau neutre exclu via
  `skillchain/objective_family.go:75` (ratchet `no_objective_submode_list_test.go`). Soirée sous
  3 matchs à objectif : pas de point, carte masquée.
- **D7 — Parts de l'objectif** = notre camp / lobby par match ; une soirée = moyenne des parts de
  ses matchs, **chaque match pèse pareil**. Rôles = partition `narrative/objective_roles.go`,
  colonnes facultatives (Prises nettes) exclues des rôles.
- **D8 — Outils de destruction : chaque frag nommé** — variante PROPRE à l'Escouade (ne pas
  toucher `fragdist.Build`, partagé par 6 appelants) : une ligne par clé d'arme du film ; les
  grenades détaillées par type depuis le film, SANS le repli « Grenade » de `fragdist.go:366` ;
  la mêlée depuis la feuille de match (le film n'a pas de clé de mêlée) ; « Objet explosif
  (bidon) » et « Chute, environnement » depuis la catégorie de source du film ; le reliquat
  feuille − film, s'il est > 0, en ligne « Non attribué ». Plus de plafond « 8 armes + Autres ».
- **D9 — Couleurs** (valeur unique par jeton, le système n'a pas de variante de thème ; valeurs
  passées au validateur `dataviz/scripts/validate_palette.js` en clair ET en sombre) :
  - famille ressource : `resource-powerup` #0D9488 (sarcelle), `resource-power-weapon` #7C3AED
    (violet), `resource-vehicle` #D55E00 (orange), `resource-rack` #0072B2 (bleu). Paire
    bonus ↔ râtelier sous 15 de ΔE en « toutes paires » : acceptée, les deux ne sont jamais
    voisines et toujours nommées ;
  - rôles d'objectif : `objective-role-take` #6366F1, `objective-role-defend` #0891B2,
    `objective-role-hold` #A924BD (toutes vérifications passées en clair et en sombre) ;
  - échanger les VALEURS de `frag-vehicle` et `frag-turret` (véhicule = orange des véhicules) ;
    ne pas toucher le mapping `fragClass.ts:76-77` ;
  - palettes cividis / okabe-ito / tol-bright : l'exécuteur choisit chaque nouvelle valeur PARMI
    les teintes déjà présentes dans cette palette, et la fait passer au validateur (gate).
- **D10 — Halo 5** (pas de film) : l'onglet ne garde que ce que la feuille de match sait (frags
  aux armes spéciales, barre épaisse seule) ; sans aucune donnée, l'onglet se masque. Branchement
  par capability (`film.usage_summary`), jamais par slug.
- **D11 — Données non transmises à publier** (pas de nouvelle mesure du film) : `TakenByFamily`
  bonus, comptes pris/utilisé/gardé/lâché par camp (aujourd'hui seuls des taux sortent de
  `computeOutcomes`), `camo_kills`, `overshield_kills`, `camo_ms`, `overshield_ms`
  (`match_usage_players_latest`, jamais lus), socles de bonus vidés (`powerup_pickups_json`),
  niveau de socle par prise et par match.
- **D12 — Code mort** : les cartes squad de `formes/` (liste §L5), la branche
  `contexte === 'squad'`, le mode `'squad'` d'`EquipmentUsageSection`, `SquadObjectiveStatsPanel`
  et le champ `objective_stats_by_xuid` sont SUPPRIMÉS avec leurs tests et leurs chaînes i18n.
  `formes_retenues` et `equipment_usage` restent vivants pour le solo (Séries temporelles,
  Synthèse) ; sur `/pages/teammates`, `equipment_usage` est retiré s'il n'a plus de lecteur.

## 2. Spécification de rendu (commune, NON NÉGOCIABLE)

Chaque carte suit la maquette à la lettre ; en cas de doute, la maquette fait foi, puis ce §2.

- **S1 Titres factuels** = nom de la mesure (tableau §3), jamais une tournure conversationnelle.
- **S2 Légendes** en bas du bloc et centrées ; le graphe centré verticalement dans sa carte ;
  deux cartes voisines ont la même hauteur.
- **S3 Valeurs dans les barres** si elles tiennent avec 6 px de marge de chaque côté ; sinon :
  barre camp contre camp → ligne de repli au-dessus (pastille d'équipe + « compte · part ») ;
  barre empilée → ligne au-dessus alignée sur le premier segment masqué (pastille + compte).
  Jamais tronquée, jamais seulement en infobulle.
- **S4 Couleurs d'équipe** `team-ally` / `team-enemy` à la place des mots « Nous » / « Eux »
  (y compris la ligne « Bonus perdus ») ; la hachure est réservée à « sans film ».
- **S5 Trait à 50 %** pointillé `warning`, libellé « 50 % : autant que l'adversaire » ; le mot
  « parité » disparaît.
- **S6 Courbes** : cumul au fil de la soirée ; valeur courte au bout de la courbe seulement ;
  aucun texte de synthèse ni statistique en bout de ligne. Un seul graphe quand les séries
  partagent mesure et échelle.
- **S7 Fiches** : coquille des fiches de médailles (`MedalDigest.tsx` : liseré gauche 4 px à la
  couleur du joueur, emblème 32 px, nom, pastille « … dominante » en haut à droite, étiquettes
  de section en capitales 9 px, pied de fiche) ; mêmes lignes dans le même ordre sur toutes les
  fiches, un zéro reste une ligne atténuée. Coquille EXTRAITE en composant partagé (utilisé par
  les médailles, les prises et l'objectif) — pas de copie.
- **S8 Code des ressources** : pastille de couleur devant chaque nom de ressource.
- **S9 Résultat toujours visible** (bande `OutcomeSequenceTape` ou pastille « Victoire 3–0 »),
  badge / encoche de dominance en plus quand le drapeau existe.
- **S10 Rôles sans jugement** : ni classement ni vert/rouge sur la répartition interne.
- **S11 Infobulles** : 3 phrases au plus, textes des maquettes.
- **S12 i18n** : toute chaîne FR et EN (`Record<Locale, T>`), aucun libellé FR en dur côté Go.
- **S13 Tokens** : aucune couleur hex ni classe Tailwind couleur dans `features/` ou `components/`.

## 3. Cartes cibles (ordre à l'écran)

| Page | Bloc | Titre FR | Titre EN | Forme (maquette) |
|---|---|---|---|---|
| Emprise | Bilan de la soirée | Contrôle des ressources | Resource control | Une piste par ressource, notre camp / adversaire, « compte · part » dans chaque segment, trait 50 % |
| Emprise | Bilan de la soirée | Contrôle des ressources au fil de la session | Resource control over the session | Un graphe, une courbe cumulée par ressource, points par match (taille = volume), bande de résultats + encoche de dominance |
| Emprise | Rôles dans l'escouade | Répartition des prises dans l'escouade | Pickups within the squad | Fiches S7 (JGtm, coéquipiers, reste du camp), « Ressource dominante », une pastille par prise, pastille vide = bonus perdu, ligne « Bonus perdus » en couleurs d'équipe |
| Emprise | Carte par carte | Contrôle des ressources, match par match | Resource control, match by match | Colonnes = matchs (heure, carte, mode, « Victoire 3–0 », badge de dominance) ; lignes = ressource puis chaque objet ; case « 5–2 » colorée plus / moins ; armes de râtelier repliées ; « sans film » hachuré |
| Emprise | Prendre, et s'en servir | Frags obtenus avec les ressources | Kills with resources | Barre épaisse = part des frags (valeurs dedans), barre fine = part de l'exposition (temps d'effet / prises), ligne d'exposition dessous |
| Emprise | Prendre, et s'en servir | Rendement face à l'adversaire | Efficiency against the opponent | Écart relatif, axe −50/+50 %, valeur au bout, rendements bruts de l'autre côté du zéro |
| Emprise | Par rapport à d'habitude | Contrôle des ressources, soirée après soirée | Resource control, session by session | Courbes bonus / armes spéciales, ce soir à droite, médiane pointillée par couleur, trait 50 % |
| Contributions | — | Répartition des frags | (existant) | Barres empilées par classe, comptes dans les segments + repli S3, total au bout |
| Contributions | — | Outils de destruction | (existant) | Légende des joueurs, compte au bout de chaque barre, pastille de classe devant l'arme, % en infobulle, lignes D8 |
| Contributions | Objectif | Rapport de force par famille de mode | (existant) | Un cadre par famille (bandeau « Bases · 4 matchs »), actions rangées Prendre / Défendre / Tenir, barre camp contre camp, durées en min/s |
| Contributions | Objectif | Rapport de force au fil de la session | Balance of power over the session | 3 courbes de rôle cumulées (D7), points par match, bande de résultats, mode sous chaque match ; masquée sous 3 matchs à objectif |
| Contributions | Objectif | Répartition de l'objectif dans l'escouade | Objective share within the squad | Fiches S7, « Rôle dominant », une ligne par action et par famille, barre à l'échelle de la ligne, pied Prendre / Défendre / Tenir |
| Contributions | Objectif | Rapport de force, soirée après soirée | Balance of power, session by session | D6 + D7 : 3 courbes, ce soir à droite, médiane, sous chaque soirée barre victoires / défaites + « x sur y » + modes |
| Dynamique | — | Écart cumulé au FDA attendu | (existant) | Même abscisse que la balance des dégâts, valeur de fin au bout des courbes, pastilles « écart moyen » retirées |

Cartes RETIRÉES (code mort, D12) : « Ce que mon camp prend de l'objectif » (remplacée), toutes les
cartes squad de l'ancien onglet Usages (liste §L5), les 9 tuiles d'objectifs de l'en-tête.

## 4. Organisation

- Worktree dédié : `git worktree add ../LevelUp-wt-emprise -b wt/emprise feat/v75` (le principal
  est partagé). Un exécuteur Opus par lot, lots SÉQUENTIELS dans ce worktree (fichiers communs :
  `teammates_service*`, `openapi.yaml`, `generated.ts`, `features/squad/i18n.ts`).
- Superviseur : vérifie chaque clôture sur pièces (gates rejoués, diff relu), fusionne `wt/emprise`
  dans `feat/v75` en fin de chantier (les branches `wt/**` n'ont pas de CI : verdict = run de
  `feat/v75`). Pas de push par les agents ; commits préfixés `feat(emprise/<lot>)`.
- Règles d'environnement (tous lots) : une commande `go` à la fois ; CGO avec le gcc winlibs
  (voir mémoire) ; `GOCACHE` isolé au worktree ; aucun serveur, navigateur, backfill ni ouverture
  RW des bases réelles par un agent ; vitest hors sandbox.
- Gate commun de clôture de lot (en plus du gate propre) :
  `cd apps/go-api && go test ./...` ; `make go-api-lint` ; `make check-types` ; `make test-web` ;
  `cd apps/web && npm run lint` ; `node tools/knip-ratchet.mjs` ;
  `node tools/lint-cross-feature-imports.mjs` ; `node tools/lint-no-hardcoded-colors.mjs` ;
  si le contrat change : `make openapi-gen && make generate-types && make openapi-check`.
- Clôture de lot = gate vert + items statués + section du lot mise à jour ici (dans le worktree,
  inclus dans le commit du lot) + entrée `.ai/thought_log.md` du worktree (commitée) + rapport
  (fait / non fait / découvertes). Le plan de référence est la copie du WORKTREE.

## 5. Lots

### L0 — Jetons de couleur (web) · rapide

Périmètre : `lib/accessibility/semantic-tokens.ts`, `lib/accessibility/palettes/{default,cividis,okabe-ito,tol-bright}.ts`,
`styles/globals.css`, `lib/accessibility/__tests__/__snapshots__/coverage.test.ts.snap`,
`features/_shared/usage/usageMetricKinds.ts` (+ test), `features/session-detail/SessionUsageSection.tsx`.

- [x] L0.1 Ajouter les 7 jetons D9 (type, `ALL_TOKENS`, 4 palettes, repli CSS, snapshot).
- [x] L0.2 Échanger les valeurs `frag-vehicle` / `frag-turret` dans les 4 palettes et `globals.css` (`globals.css` n'a aucun repli `--ac-frag-*` : rien à y échanger, vérifié par grep ; commentaire périmé de `tol-bright.ts:61` corrigé).
- [x] L0.3 `usageMetricKinds.ts:120-121` : camouflage et surbouclier → `resource-powerup`.
- [x] L0.4 `ROLE_TOKENS` / `roleToken` (`usageMetricKinds.ts:138-147`) → `objective-role-*`.
- [x] L0.5 Valeurs cividis / okabe-ito / tol-bright passées au validateur (clair et sombre, toutes paires) ; sorties collées dans le rapport.
- Gate : `coverage.test.ts`, `fragClass.guard.test.ts`, `colorDistance.guard.test.ts`, `usageMetricKinds.test.ts` + gate commun.

Journal L0 (2026-09-26, exécuteur Opus, `wt/emprise`) : 7 jetons ajoutés, véhicule / tourelle échangés dans les 4 palettes, bonus et rôles branchés sur leurs familles (`SessionUsageSection.tsx` lit `roleToken` : aucun changement requis, il suit). Valeurs retenues (teintes déjà présentes dans chaque palette, validateur dataviz toutes paires, clair #fdfdfe et sombre #171717) — ressources powerup / armes spéciales / véhicule / râtelier : Okabe-Ito #009E73 / #CC79A7 / #A04700 / #0072B2, Cividis #009E73 / #AA4499 / #A04700 / #0072B2, Tol Bright #117733 / #AA4499 / #CC3311 / #0077BB ; rôles prendre / défendre / tenir : Okabe-Ito #0072B2 / #009E73 / #B77E00, Cividis #0072B2 / #009E73 / #686B00, Tol Bright #0077BB / #117733 / #AA3377. Tout passe sauf Okabe-Ito ressources en sombre : Reddish Purple hors bande de clarté de 0,009 (L 0,679 > 0,67) — AUCUN jeu de 4 teintes Okabe-Ito ne passe tout (recherche exhaustive), l'alternative sans pourpre (ΔE 14,5 entre armes spéciales et véhicules, voisines) est pire ; écart documenté dans `okabe-ito.ts`. Tol Bright : l'orange des véhicules (`TOL_VIBRANT_ORANGE`, désormais `frag-vehicle`) échoue le validateur, `resource-vehicle` prend le Vibrant Red. Gate : typecheck, lint (0 erreur), vitest complet, knip-ratchet, lint-no-hardcoded-colors, lint-cross-feature-imports verts.

### L1 — Dynamique : écart cumulé au FDA attendu · rapide

Périmètre : `SquadFdaGapCumulativeCard.tsx` (+ test), `charts/squadFdaGapChart.ts` (+ test),
`SquadDynamiquePage.tsx` (+ test), `SquadUsagesPage.tsx` (retrait du montage), `SquadFragSection.tsx`
(slot `leftOfBreakdown` supprimé), `features/squad/i18n.ts` (`fdaGap.averageCaption` supprimée).

- [x] L1.1 Monter la carte sur Dynamique, à côté de « Balance des dégâts cumulée » ; `playerOrder` filtré sur `performanceSeries` (déjà le cas, vérifié : `SquadDynamiquePage.tsx`, `roster.filter((p) => performanceSeries?.[p])`).
- [x] L1.2 Retirer les pastilles « écart moyen par match » et leur clé i18n (`fdaGap.averageCaption` FR/EN + type ; `meanFdaGapPerMatch` devenu mort supprimé avec ses tests ; `meanFdaGap` reste vivant, lu par `TimeseriesFdaGapTrend`).
- [x] L1.3 Valeur de fin au bout de chaque courbe (précédent `squadRangeRolesChart.ts:199`). Retouche superviseur : dernier point non nul grossi comme la maquette (diamètre 9 px, liseré 2 px couleur de carte), autres points à 4 px.
- [x] L1.4 Légende en bas centrée, graphe centré (S2).
- [x] L1.5 Retirer le montage de l'onglet Usages et le slot devenu mort ; corriger le commentaire périmé (`Synergies`).
- [x] L1.6 Test du chemin sans `expected_stats` (carte absente) si inexistant (existait au niveau carte ; ajouté au niveau page Dynamique).
- Gate : tests cités + gate commun.

Journal L1 (2026-09-26, exécuteur Opus, `wt/emprise`) : « Écart cumulé au FDA attendu » quitte Usages (slot `leftOfBreakdown` de `SquadFragSection` supprimé avec sa branche de rangée ; `useCapability` et `perfSeriesByPlayer` retirés d'`SquadUsagesPage`) et rejoint Dynamique sur la rangée de « Balance des dégâts cumulée » : grille `md:grid-cols-2`, balance à gauche, écart à droite, mêmes `perfSeriesByPlayer` / `playerOrder` / couleurs ; la grille étire les deux cartes à la même hauteur, la carte d'écart est `fluid` (le graphe remplit la carte). Si une seule des deux capabilities existe (`damage_taken` / `expected_stats`), la survivante prend la rangée (`md:[&>*:only-child]:col-span-2`), et la rangée vide se retire (`empty:hidden`). Graphe : même abscisse `xAxisLabels(n)` que la balance ; `endLabel` par série (valeur signée, une décimale, locale de l'interface « +3,0 » / « +3.0 », couleur du joueur, zéro arrondi sans signe) posé par ECharts sur le dernier point non nul ; `labelLayout.moveOverlap: 'shiftY'` écarte les étiquettes qui se chevauchent (vérifié par un rendu SVG hors DOM d'ECharts 6.1 : trois fins à 3,0 / 3,1 / 2,9 passent de 80-88 px à 80 / 92 / 104 px) ; marge droite de grille 48 px pour ces valeurs ; légende ECharts en bas (`getLegendBase`) et `left: 'center'` explicite ; infobulle d'axe et graduations suivent le même format. Pastilles et leur texte retirés : plus rien sous le graphe. Infobulle : texte partagé `common.charts.fda_gap_tooltip` conservé (2 phrases, S11 tenu) — voir Découvertes. Gate : typecheck OK, lint 0 erreur (26 avertissements préexistants, aucun sur les fichiers touchés), vitest complet 807 fichiers / 8 663 tests verts (5 fichiers / 23 tests ignorés préexistants), knip-ratchet 0/0/0, lint-no-hardcoded-colors 0 violation, lint-cross-feature-imports 7 ≤ plafond 7. Volet Go du gate commun (`go test`, `go-api-lint`) non joué : aucun fichier Go touché, gate du lot fixé sans Go par le superviseur.

Retouche L1 (2026-09-26, demande superviseur, maquette à la lettre) : `withEndPoint` dans `squadFdaGapChart.ts` grossit le dernier point non nul de chaque courbe (`symbolSize` 9, `borderColor` = couleur de carte, `borderWidth` 2), les autres points gardent 4 px ; test dédié (dont trou d'intersection en fin de soirée). Gate : typecheck OK, lint 0 erreur, suite vitest squad 74 fichiers / 709 tests verts.

### L2 — Contributions : frags · moyen

Périmètre web : `SquadFragSection.tsx`, `charts/squadFragBreakdownChart.ts`, `charts/squadFragTools.ts`,
`lib/accessibility/scales/fragClass.ts` (COMMENTAIRES des lignes 76-77 seulement, périmés depuis L0),
`charts/squadWeaponKillsChart.ts` (+ tests), `features/squad/i18n.ts`. Périmètre Go : nouveau
builder Escouade à côté de `buildSquadWeaponKills` (`teammates_squad_charts_weapons_perf.go`),
`domain/teammates.go` (champ), tests associés, contrat.

- [x] L2.1 Répartition des frags : comptes dans les segments avec repli S3, total au bout, légende en bas centrée (rendu DOM `SquadFragBreakdownCard`, mesure au pixel `components/charts/segmentLabelFit.ts`).
- [x] L2.2 Outils de destruction, Go : lignes D8 (film, grenades par type, mêlée feuille, bidon, chute, reliquat), sans toucher `fragdist.Build` ; `fragdist_halo5_golden_test.go` inchangé (builder `teammates_squad_weapon_tools.go`, champ `weapon_tools` ; lecture par catégorie de source : voir journal, périmètre étendu).
- [x] L2.3 Outils de destruction, web : plus de plafond `SQUAD_TOOLS_TOP_GUNS/DETAILS` ni de libellé « Autres armes » ; légende des joueurs ; compte au bout ; pastille de classe ; % en infobulle.
- [x] L2.5 Corriger les commentaires couleur de `fragClass.ts:76-77` (découverte L0 ; mapping inchangé).
- [x] L2.4 Chiffres témoins (22/09, maquette C3EW) retrouvés par test : BR75 22/22/35, Mutilateur 1 et VK78 Commando 1 (JGtm) nommés, grenade à fragmentation 2/1/4 (`TestSquadWeaponTools_Soiree2209`, via `buildSquadWeaponKills`).
- [x] L2.6 (ajouté par le superviseur le 2026-09-27) Monter « Répartition des frags » puis « Outils de destruction » sur Contributions, entre la rangée « Stats par minute » / « Radar synergie » et la section « Performance » ; retirer le montage d'Usages et ce qui y devient mort ; tests des deux pages. Complément du même jour : sans film, les grenades forment une ligne « Grenade » (total de la feuille) au lieu de tomber en « Non attribué ».
- Gate : `squadFragBreakdownChart.test.ts`, `squadFragTools.test.ts`, `fragdist_test.go`, tests du nouveau builder + gate commun + contrat.

Journal L2 (2026-09-26/27, exécuteur Opus, `wt/emprise`) :
- **Répartition des frags** : l'option ECharts `buildFragBreakdownOption` laisse la place à un rendu DOM (`SquadFragBreakdownCard.tsx`, modèle pur `buildFragBreakdownRows` / `repliOffsetPct` / `segmentTextTone` dans `charts/squadFragBreakdownChart.ts`). Une barre par joueur sur une échelle commune (le plus gros total = 100 %), segments dans l'ordre canonique des classes. Le compte s'écrit dans son segment s'il tient avec 6 px de marge de chaque côté : mesure au pixel par `components/charts/segmentLabelFit.ts` (`labelFitsInSegment`, `useSegmentLabelFit` avec ResizeObserver ; placé sous `components/charts/` pour que L5.3 le réutilise). Sinon, ligne de repli au-dessus de la barre, décalée à la position du premier segment masqué, pastille de classe + compte. L'étiquette reste dans le DOM en `visibility: hidden`, ce qui permet de la remesurer. Une largeur inconnue (0) envoie la valeur au repli : jamais perdue. Écriture sombre ou claire selon la luminance de la classe (seuil WCAG). Total au bout (colonne 2,5 rem), légende des classes en pied de carte centrée, graphe `my-auto` dans une carte `h-full`. Aide ⓘ et infobulles de segment : textes de la maquette.
- **Outils de destruction, Go (D8)** : `teammates_squad_weapon_tools.go`, builder pur `buildSquadWeaponTools`, appelé par `buildSquadWeaponKills` sur les MÊMES lignes que la Répartition. `fragdist.go` n'est pas touché ; `squadFragClassesByPlayer` passe seulement par le nouvel helper `playerFragCounts` (même calcul). Lignes produites :
  - une ligne par clé d'arme (film sur Infinite, table native sur Halo 5), `Kills − MechanicKills`, grenades par type ;
  - la mêlée tirée de la feuille de match (série de performance) ;
  - assassinat, coup au sol et charge d'épaule, seulement si `CapNativeKillMechanics` ;
  - « objet explosif » et « chute, environnement » d'après la catégorie de source du film. Les lignes de bobine et d'environnement recouvertes perdent ces frags, par (joueur, clé) ;
  - le reliquat feuille − lignes s'il est > 0, en « Non attribué », toujours en dernier.

  Une arme sans nom ne donne pas de ligne : ses frags vont au reliquat. Tri par total décroissant. Contrat : `weapon_kills` (`SquadWeaponKills`/`SquadWeaponBar`) est REMPLACÉ par `weapon_tools` (`SquadWeaponTools{players, lines[]}`, ligne = `kind`, `weapon_key?`, `label?`, `label_en?`, `class`, `kills_by_player`, `total_squad`), car l'ancien champ n'a plus de lecteur. `aggregateSquadWeaponBars` est supprimé. Aucun libellé côté Go : des natures (`domain.SquadToolKind*`) et le nom du registre.
- **Périmètre Go étendu (décision de l'exécuteur, nécessaire à D8)** : la plupart des objets explosifs du décor n'ont PAS de clé de registre (15 des 19 tags `OBJET_EXPLOSIF`). Avec les seules lignes par clé, leurs frags retombaient en « Non attribué » et les bobines apparaissaient une à une. La « catégorie de source du film » demandait donc une lecture de plus, ajoutée sans rien changer aux lecteurs existants :
  - interface optionnelle `port.KillSourceCategorizer` (domain `KillSourceCategory*`), implémentée par Halo Infinite (`killsource_registry.go` : `OBJET_EXPLOSIF` → objet explosif, `DEGAT_GLOBAL` → environnement) ;
  - `KillSourceWeaponKillsRepo.LoadKillSourceCategoryKills` (nouveau fichier `killsource_category_kills_repo.go`, même requête sur `match_kill_events_latest`, parcours factorisé `forEachSourceTally`) ;
  - `SquadV2LoaderAdapter.LoadKillSourceCategories` et interface optionnelle `squadagg.SquadKillSourceCategoryLoader`, découverte par assertion.

  Sans film (Halo 5), la lecture rend `games.ErrCapabilityNotSupported` et le builder se passe des deux lignes. Hors liste du lot, touchés aussi : `SquadUsagesPage.tsx` (nom du champ), `SquadKillMechanicsChart.tsx` (nouvelle forme de données, rendu INCHANGÉ en mode `share`), `lib/api/types.ts`. Trace : `teammates_weapon_tools_built` (lignes, joueurs dont les lignes dépassent la feuille).
- **Outils de destruction, web** : `charts/squadFragTools.ts` ne fait plus que nommer (`buildSquadToolRows`, `toolLineLabel`). Une arme prend son nom de registre dans la locale ; les autres natures sont nommées par le manifeste `frags` (mêlée, mécaniques, non attribué) ou par l'i18n Escouade (`weaponKills.explosiveObject` / `environment`, FR et EN). Plafonds, « Autres armes » et « Autres frags » sont supprimés, clés i18n comprises. `charts/squadWeaponKillsChart.ts` prend en entrée des lignes génériques (`SquadBarRows`) :
  - compte au bout de chaque barre non nulle ;
  - pastille de classe en texte riche devant le nom (`fragClassColor`) ;
  - infobulle « 22 frags (34 % des siens) » (`weaponKills.killsShare`) ;
  - hauteur sans plafond ;
  - mode `share` conservé pour les Mécaniques de frag.

  Légende des joueurs en pied de carte (`ChartLegend`), aide ⓘ de la maquette.
- **L2.4** : `TestSquadWeaponTools_Soiree2209` passe par `buildSquadWeaponKills` avec un chargeur qui rend la soirée du 22/09 telle que le lecteur du film la rend, catégories comprises. Il retrouve : BR75 22 / 22 / 35 ; Mutilateur 1 et VK78 Commando 1 (JGtm), nommés ; « Grenade frag » (clé `hinf_frag_grenade`) 2 / 1 / 4 ; mêlée 6 / 13 / 16 (feuille) ; objet explosif 2 / 1 / 2 ; chute 0 / 1 / 1 ; aucune ligne de bobine ; 16 lignes ; BR75 en tête.
- Gate :
  - Go : `go test ./...` vert sauf `internal/config` (`TestLoadPlayers_RacyWindowSnapshotNeverStored`, fenêtre de 1 s, échec sous charge pendant que vitest tournait en parallèle), vert rejoué isolé ; `go test -tags=integration` des lecteurs de source vert ; `make go-api-lint` 0 issue ;
  - contrat : `make openapi-gen && make generate-types && make openapi-check` OK ;
  - web : typecheck OK ; lint 0 erreur (26 avertissements préexistants, aucun sur les fichiers touchés) ; knip-ratchet 0/0/0 ; couleurs en dur 0 ; imports croisés 7 ≤ 7 ; vitest complet 810 fichiers / 8 665 tests verts (5 fichiers / 23 tests ignorés préexistants). La garde `contract-surface.guard.test.ts` rougissait sur le retrait ASSUMÉ de `SquadWeaponBar` / `SquadWeaponKills` (remplacés par `SquadWeaponToolLine` / `SquadWeaponTools`). Snapshot régénéré par la procédure documentée (`UPDATE_CONTRACT_SURFACE=1`). Ce sont ses deux seules disparitions ; le reste du diff du snapshot n'est que des ajouts d'autres chantiers qu'il n'avait pas encore enregistrés, et les ajouts sont tolérés par la garde.

Décisions du superviseur sur le rapport L2 (2026-09-27) :
- périmètre étendu (port `KillSourceCategorizer`, lecture par catégorie, chargeur) : ACCEPTÉ ;
- libellé « Grenade frag » du registre : ACCEPTÉ (le registre fait foi) ;
- mêlée lue sur la feuille (D8) et écart d'un frag chez Chocoboflor : ACCEPTÉS, tracés en debug ;
- ligne bidon à la couleur « Non attribué » : ACCEPTÉ (cohérent avec la Répartition).

Compléments L2 (2026-09-27, exécuteur Opus, `wt/emprise`) :
- **L2.6** : `SquadFragSection` est montée par `SquadContributionsPage`, section « Frags et armes » (`t.sections.fragsArmes`), entre la rangée Stats par minute / Radar synergie et « Performance ». « Répartition des frags » est pleine largeur, « Outils de destruction » juste en dessous, pleine largeur. Sur Halo 5, « Précision par rôle » reste à côté de la Répartition comme avant ; la maquette la dit « carte à part, inchangée ». Les cartes sont toujours montées, comme les autres graphes de la page : chaque carte gère son état vide. L'ordre des joueurs reprend le prédicat d'Usages (classes de frags OU série de performance). « Mécaniques de frag » (Halo 5) n'a pas bougé. Sur `SquadUsagesPage`, le montage est retiré avec `hasFrags`, `fragClassesByPlayer`, `playerColors`, `playerOrder` et les imports devenus morts : l'onglet ne compte plus que l'équipement et les formes pour son état vide. Tests : Contributions monte les deux cartes dans l'ordre et avant Performance, y compris sans données ; Usages ne les monte plus.
- **Grenades sans film** : le détail par type ne vient que des lignes MESURÉES au film (`FromDamageSource`). Une ligne de grenade typée d'une autre provenance (table native de Halo 5) n'est pas posée. Un joueur sans grenade typée au film reçoit une ligne `grenade` (nouvelle nature `domain.SquadToolKindGrenade`, classe grenade, nommée côté web par `frags.class.grenade`) au total de la feuille. Avec film, le détail remplace la ligne, joueur par joueur. Tests : `TestBuildSquadWeaponTools_Halo5GrenadesSansFilm` (typées natives 8 + 4, feuille 14 → une ligne « Grenade » 14, aucun « Non attribué ») et `TestBuildSquadWeaponTools_GrenadesParJoueur`. La découverte L2 « Halo 5, grenades non typées en Non attribué » est donc close ; « Environmental Explosives » nommé sur Halo 5 reste vrai.
- Gate des compléments :
  - `go test ./...` vert sauf `internal/sync/skill` (`TestLUSRV2Shadow_RafalesBornees_300Candidats`, seuil de 2 s dépassé de 5 à 17 ms sous charge), vert rejoué isolé ; `make go-api-lint` 0 issue ;
  - contrat : `openapi-gen`, `generate-types` et `openapi-check` OK, aucun fichier modifié (`kind` est une chaîne) ;
  - web : typecheck OK ; lint 0 erreur (26 avertissements préexistants) ; knip 0/0/0 ; couleurs 0 ; imports croisés 7 ≤ 7 ; vitest complet 810 fichiers / 8 668 tests verts (5 / 23 ignorés préexistants).

### L3 — Contributions : objectif · moyen

Périmètre Go : `teammates_service.go` / `teammates_service_usage.go` (périmètre D2), nouveau calcul d'historique
d'objectif (`internal/analysis/` pur + type `domain`), `domain/squad_v2.go:119-123`,
`service/squad_service_v2.go:31-35,43-48,130-143`, `api/wire/registry_pages_home.go:155-159`, contrat.
Périmètre web : `formes/cards/ObjectiveCards.tsx`, `formes/model/objectives.ts`, nouveaux
composants de carte, coquille de fiche extraite de `MedalDigest.tsx`, `SquadContributionsPage.tsx`,
`SquadLayout.tsx:35,392-397`, `SquadObjectiveStatsPanel.tsx`, `v2/types.ts:73`, `features/squad/i18n.ts`.

- [x] L3.1 Périmètre D2 pour les blocs d'usage ; mettre à jour `teammates_service_usage_test.go` (…_PublieLeBlocEquipementSurLeScopeFiltre, renommé …_SurLePerimetreEscouade, + repli sans sélection) ; ratchet `TestUsageScopeReadsSquadPopulation` ajouté à `no_raw_squad_intersection_test.go`.
- [x] L3.2 Résultat, score et dominance par match : jointure côté web sur `match_history` par `match_id` (`outcome`, `score_label`, `dominance_flag`), valide une fois le périmètre D2 posé ; aucun champ Go ajouté pour ça. Test : un match du bloc sans ligne `match_history` → case sans résultat, pas d'erreur.
- [x] L3.3 Historique d'objectif D6/D7 (Go, `analysis` pur + `domain`), exclusion du drapeau neutre, tests purs.
- [x] L3.4 Extraire la coquille de fiche de `MedalDigest.tsx` (S7) ; `MedalDigest` l'utilise ; `MedalDigest.test.tsx` vert.
- [x] L3.5 Rapport de force par famille de mode (cadres par famille, rôles, barre camp contre camp, durées).
- [x] L3.6 Rapport de force au fil de la session (masquée sous 3 matchs à objectif).
- [x] L3.7 Répartition de l'objectif dans l'escouade (fiches, « Rôle dominant », barres par ligne).
- [x] L3.8 Rapport de force, soirée après soirée.
- [x] L3.9 Supprimer `ObjectivesLobbyTrackCard`, `ObjectivesGapSquadCard`, `roleLobbyParts`, le panneau et le champ `objective_stats_by_xuid` (Go, contrat, web, i18n).
- [x] L3.10 Chiffres témoins retrouvés par test : 07/09 Prendre 39,0 / Défendre 36,8 / Tenir 43,8 % ; fiches du 22/09 (JGtm 4 drapeaux capturés, 3 volés).
- Gate : tests cités, `objective_roles_test.go`, `no_objective_submode_list_test.go`, `no_raw_squad_intersection_test.go` + gate commun + contrat.

Journal L3 (2026-09-27, exécuteur Opus, `wt/emprise`) :
- **Périmètre D2 (L3.1)** : `loadUsageBlocks` reçoit une `porteeUsage` (matchs filtrés, `allSquadRows` APRÈS `filterExactComposition`, historique de la composition, équipe alliée par match, historique de matchs). `perimetreEscouade` garde les matchs filtrés présents dans `allSquadRows`, dans l'ordre des matchs filtrés ; sans coéquipier sélectionné, les matchs filtrés tels quels. S'applique aux trois blocs (`equipment_usage`, `formes_retenues`, historique d'objectif). Ratchet `TestUsageScopeReadsSquadPopulation` (le câblage de `GetPage` et l'appel de `perimetreEscouade`). Test renommé `…_PublieLeBlocEquipementSurLePerimetreEscouade` (m2 joué sans Ally1 sort du bloc, `TotalMatches` reste 2) et nouveau `…_SansSelectionLePerimetreEstLeScopeFiltre`. Commentaires de périmètre mis à jour (domain, service, `types.ts`).
- **Résultat par match (L3.2)** : `buildSessionFil` joint `match_history` par `match_id` (issue, score, dominance). Un match sans ligne garde sa case, sans résultat ni erreur (test du modèle, d2 du 07/09).
- **Historique d'objectif (L3.3)** : type `domain.SquadObjectiveHistory` (soirée affichée, soirées précédentes, soirées avec objectif, soirées sous le minimum, seuil publié) ; calcul pur `squadformes.BuildObjectiveHistory` (dans `squadformes` pour réutiliser `familyColumnsOfRole` : pas de 3e copie du vocabulaire par famille) ; orchestration `teammates_service_objective_history.go`, UNE lecture `LoadObjectiveColumnRows` sur l'union historique + périmètre ; camp = équipe alliée que `GetPage` charge déjà (`mainTeamByMatch`). Part d'un rôle par match = notre camp / lobby sur les colonnes du rôle de la famille, facultatives exclues ; soirée = moyenne des parts (chaque match pèse pareil) ; un rôle à 0 sur 0 ne pèse pas (choix documenté, identique côté web). Soirée = libellé de session ; précédente = jouée avant le premier match du périmètre et sans match du périmètre ; ≥ 3 matchs à objectif ; les 10 dernières. Drapeau neutre : `skillchain.IsNeutralFlagSubMode` ajouté à la source unique (`objective_family.go`, constante nommée dans LA liste, ratchet vert), injecté par `WithObjectiveHistory` au câblage sous `CapMatchObjectiveStats`. Logs : `teammates_objective_history` (debug), échecs en warn. Publié sans coéquipier sélectionné : non (nil).
- **Coquille (L3.4)** : `SquadPlayerSheet` / `SquadSheetAvatar` / `SquadSheetSection` extraits de `MedalDigest` ; `MedalDigest` les utilise (le dépliage des médailles passe par `afterFooter`, après le pied, comme avant). Fond de pastille des médailles gardé à l'identique (voir Découvertes). `MedalDigest.test.tsx` vert.
- **Quatre cartes (L3.5 à L3.8)**, dossier `features/squad/objectif/` : modèle pur `objectif.logic.ts`, graphes `objectifCharts.ts` (ECharts, deux grilles alignées, rendu SVG), cartes `ObjectiveBalanceCard`, `ObjectiveSessionFilCard`, `ObjectiveSheetsCard`, `ObjectiveEveningsCard`, cadre / légende / note `ObjectifFrame`, textes `objectifStrings.ts` (FR / EN, ceux de la maquette), montage `SquadObjectiveSection` sur Contributions après « Frags et armes » : famille | fil (même rangée `lg:grid-cols-2`, même hauteur), fiches pleine largeur, soirée après soirée pleine largeur ; section retirée sans match à objectif.
- **Suppressions (L3.9)** : `ObjectivesGapSquadCard`, `ObjectivesLobbyTrackCard` (+ leurs textes de `cardsI18n`), `roleLobbyParts` (+ ses deux tests), le bloc Objectifs du contexte escouade de `FormesRetenuesSection` (solo seul désormais ; test « dix-sept cartes »), `SquadObjectiveStatsPanel` et son montage, `objective_stats_by_xuid` (domain, `SquadServiceV2.WithObjectiveStatsRepo` et son alimentation, câblage `SquadV2Ctx`, contrat, `v2/types.ts`), chaînes `objectives` de l'i18n Escouade.
- **Témoins (L3.10)** : Go `TestObjectiveHistory_Temoin0709` (39,0 / 36,8 / 43,8 %, 1 sur 7, 4 Bases puis 3 Drapeau) ; web `objectif.logic.test.ts` (même cumul par le fil ; fiches du 22/09 : JGtm 4 capturés, 3 volés ; rôle dominant) et `SquadObjectiveSection.test.tsx` (rendu des fiches). Fixtures figées reprises de la maquette (`objectif.fixtures.ts`), aucune base lue.
- Gate :
  - Go : `go test ./...` vert (code de sortie 0, aucun paquet en échec) ; `make go-api-lint` 0 issue ; ciblés verts : `TestObjectiveRoles_*` (`objective_roles_test.go`), `TestNoDuplicateObjectiveSubModeList` + `TestObjectiveListMarkersStillInSource`, `TestExactCompositionWiringPresent` + `TestNoRawIntersectionConsumption` + `TestUsageScopeReadsSquadPopulation`, `TestObjectiveHistory_*`, `TestIsNeutralFlagSubMode`, tests de page (périmètre D2, historique, dégradé) ;
  - contrat : `make openapi-gen && make generate-types && make openapi-check` OK ; nouveau schéma `SquadObjectiveHistory` / `SquadObjectiveEvening` / `SquadObjectiveFamilyCount`, champ `squad_objective_history` ; `objective_stats_by_xuid` retiré de `SquadHeader` ; garde de surface verte sans régénération du snapshot (aucun type retiré) ;
  - web : typecheck OK ; lint 0 erreur (26 avertissements préexistants) ; knip 0/0/0 ; couleurs en dur 0 ; imports croisés 7 ≤ 7 ; vitest complet 818 fichiers, 8 687 tests verts, 10 tests en délai dépassé (5 s) dans 8 fichiers de garde-rails qui parcourent `src/` (machine chargée par d'autres processus) ; rejoués : 6 fichiers verts au délai normal, les 2 derniers (`useCopyToClipboard.guard`, `encountersFormat.guard`) dépassent encore 5 s même seuls (6 à 9 s) et passent avec `--testTimeout=60000` : les 8 fichiers, 30 tests verts. Aucun test désactivé.
- **Corrections superviseur (2026-09-27)**. Décisions : légende en pied de carte, ECharts, encoche de dominance, rôle 0 sur 0 hors moyenne et abréviations ACCEPTÉS ; découvertes 1, 3 et 6 restent consignées ; 2 et 5 corrigées :
  - **Drapeau neutre hors du fil** : `modesEcartes` (service) est l'unique prédicat « mode écarté » du fil ET de l'historique, sur une seule source : `pairNamesOf` = `pair_name` brut des lignes canoniques du joueur, complété par les lignes escouade. Le bloc formes le publie par match (`SquadFormesObjective.excluded_from_balance`, via `MatchMeta.ObjectiveExcluded`) et `buildSessionFil` ignore ces matchs. Tests : Go `TestTeammatesService_GetPage_FilEtHistoriqueEcartentLeMemeMatch` (le même match marqué dans le bloc et retiré du point « ce soir ») ; web « drapeau neutre (excluded_from_balance) : le fil l'ignore, sa fin égale le point ce soir » (07/09 + un 8e match neutre : fin du fil = 39,0 / 36,8 / 43,8 %, 7 matchs).
  - **Taille de `teammates_service.go`** : 501 lignes (522 avant le lot). `filterCanonicalByMatchIDsSet` déplacée telle quelle dans `teammates_service_briefing.go` (420 lignes), comportement inchangé.
  - Gate : `go test` des paquets touchés verts (teammates, squadformes, domain, skillchain, archlint, wire, service, narrative) ; `make go-api-lint` 0 issue ; contrat régénéré et vérifié (`excluded_from_balance` ajouté) ; typecheck `tsc -b --force` OK ; lint 0 erreur ; knip 0/0/0, couleurs 0, imports croisés 7 ≤ 7 ; vitest des fichiers touchés 11 fichiers / 85 tests verts ; les 8 fichiers de garde-rails en délai lors de la suite complète, rejoués chacun SEUL au délai standard (5 s) : tous verts (1, 3, 1, 4, 3, 15, 2, 1 tests).

### L4 — Emprise : données (Go) · lourd

Périmètre : `sync/replayartifacts/derivations.go`, `padtiers.go` (+ test par `Deriver`),
`api/wire/registry_build_queue.go`, `analysis/sessionusage/*` (publication des comptes),
`platform/duckdb/session_usage_repo.go` (colonnes camo/overshield), nouveau bloc
`SquadEmpriseBlock` (`domain`) calculé dans `internal/analysis/` (pur) et orchestré dans
`service/teammates/`, contrat. **Revue adversariale obligatoire** (sync / persist).

- [ ] L4.1 Panne des niveaux de socle : `WithRead` ajouté à `DerivationsDeps` et fourni par les 3 appelants ; erreur de lecture journalisée (`slog.ErrorContext`) ; échec de lot → marque NON posée ; test qui passe par `Deriver`.
- [ ] L4.2 Publier D11 (comptes par camp et par ressource, par match, par joueur, bonus perdus, temps et frags d'effet, socles vidés) sans casser `usage_outcomes_guard_test.go` ni le golden.
- [ ] L4.3 Bloc `emprise` : contrôle par ressource et par camp ; série par match (résultat et dominance joints côté web comme L3.2) ; fiches par joueur ; grille match × objet (niveau de socle, sans film) ; production (frags d'effet, `power_weapon_kills` par camp) et exposition ; rendement ; habitude D5.
- [ ] L4.4 Capability D10 (Halo 5) ; `ErrCapabilityNotSupported` → réponse partielle propre ; test du chemin dégradé.
- [ ] L4.5 Logs `slog.*Context` sur les étapes ; aucune erreur avalée.
- [ ] L4.6 Chiffres témoins du 22/09 retrouvés par test d'intégration : bonus 12 / 8, armes spéciales 23 / 29, frags d'effet 8 / 5, temps d'effet 2 min 39 / 1 min 53, frags aux armes spéciales 47 / 54.
- Gate : `go test -tags=integration ./...` (serial `-p 1`), `padtiers_integration_test.go`, `pad_tiers_wiring_test.go`, `no_art_patterns_test.go`, `no_analysis_type_in_http_body_test.go`, `no_title_package_in_analysis_test.go` + gate commun + contrat.

### L5 — Emprise : onglet (web) · lourd

Périmètre : route `squad/emprise.tsx` + redirection `squad/usages.tsx`, `SquadLayout.tsx`,
`lib/pageTitle.ts`, `features/squad/i18n.ts`, nouvelle page de l'onglet et ses cartes, formes
réutilisées (`Piste100Form` revue, `GrilleForm`, `OutcomeSequenceTape`, `sessionBarsTrendChart`,
coquille de fiche L3.4), suppressions D12.

- [ ] L5.1 Onglet « Emprise » (D1), route et redirection, titre de page, tests de navigation (`SquadLayout.nav.test.tsx`, `pageTitle.labels.guard.test.ts`, `shellNavigation.test.ts`).
- [ ] L5.2 Les 7 cartes du §3 (Emprise), dans l'ordre et les blocs de la maquette, spec S1-S13.
- [ ] L5.3 `Piste100Form` : adversaire en `team-enemy` plein, « compte · part » dans les segments, repli S3, trait 50 % (S5).
- [ ] L5.4 Supprimer : `EquipmentRegularityCard`, `EquipmentLobbyTrackCard`, `EquipmentSquadGridCard`, `EquipmentSquadTrackCard`, `PadsGapSquadCard`, `PadsTwoFriezesCard`, `PadsSquadByMatchCard`, `PadsSquadWeaponGridCard`, la branche `contexte === 'squad'` et `FormesContexte`, `lobbyParts`, `teamShareOfMatch`, le mode `'squad'` d'`EquipmentUsageSection`, `SquadUsagesPage.tsx` et ses tests / fixtures, les clés i18n orphelines ; `equipment_usage` retiré de `/pages/teammates` s'il n'a plus de lecteur.
- [ ] L5.5 Abaisser les plafonds `knip-ratchet` et `lint-cross-feature-imports` si la suppression les fait baisser.
- [ ] L5.6 Halo 5 (D10) : test du rendu dégradé et de l'onglet masqué.
- Gate : gate commun + `singleCountSource.guard.test.ts`, `noLocalUsageCopies.guard.test.ts`, `usageEmptyStateCanonical.guard.test.ts`.

### L6 — Clôture (superviseur)

- [ ] L6.1 Revue adversariale du diff cumulé (skill `adversarial-review`), correctifs par lot rouvert.
- [ ] L6.2 Rattrapage local (serveur arrêté) : `levelup backfill-pad-tiers` puis `backfill-usage-summary`.
- [ ] L6.3 Fusion `wt/emprise` → `feat/v75` (sur accord), CI verte au niveau job.
- [ ] L6.4 Gate visuel par l'utilisateur, soirée du 22/09, maquettes à côté ; il nomme les témoins.
- [ ] L6.5 Prod (sur accord, prévenir avant) : mêmes rattrapages après déploiement.
- [ ] L6.6 Mettre à jour `REFERENCE_CANAUX_EQUIPEMENT_2026-09-09.md` §4 (périmé), ce plan, le journal.

### L7 — Véhicules (données du film + lignes des cartes) · lourd, après L6

Pourquoi après : nouvelle projection du rejeu (montées, pertes, temps à bord), nouvelle table
append-only (ADR 0026, INSERT via `persist.BatchBuilder`), rattrapage de l'historique, frags
depuis un véhicule côté adversaire (lecture de `match_kill_events_latest` sans filtre de tueur) —
le lot le plus risqué, qui ne doit pas retarder l'onglet. Plan détaillé à écrire à la clôture de
L6 (périmètre, décisions, gates), soumis à l'utilisateur avant exécution.

- [ ] L7.0 Plan détaillé écrit et validé par l'utilisateur.

## 6. Reprise de session

Lire ce fichier (statuts des items), puis `git -C ../LevelUp-wt-emprise log --oneline -15`, puis
l'entrée la plus récente du journal. Reprendre au premier item non statué du premier lot non clos.

## 7. Découvertes (à consigner ici, pas à traiter)

- Route `/pages/squad/v2` (hors `/engagement`) sans appelant web : `SquadPageV2Response` et son
  service entier sont morts (relevé du 2026-09-26) — chantier séparé.
- `teammates_waiting` du contexte de mort : 8 morts sur 220 avec un coéquipier en attente, contre
  74 d'après le journal des morts (22/09) ; détection à diagnostiquer avant tout usage.
- `formes/` deviendra solo seul mais reste sous `features/squad/` (import croisé compté par
  `lint-cross-feature-imports`) — déplacement sous `_shared/` à décider plus tard.
- (L0) `lib/accessibility/scales/fragClass.ts:76-77` : les commentaires « indigo profond » / « orange brûlé » décrivent les anciennes valeurs de `frag-vehicle` / `frag-turret` (échangées par L0) ; fichier hors périmètre L0 (mapping intouchable), commentaires à corriger par un lot qui y touche.
- (L0) Palette Okabe-Ito : aucun jeu de 4 teintes de la palette ne passe le validateur dataviz en entier (ressources) ; écart résiduel accepté par l'exécuteur (bande de clarté sombre dépassée de 0,009) — **CONFIRMÉ par le superviseur le 2026-09-26** : écart marginal, chaque ressource a sa pastille et son nom (codage secondaire).
- (L1) Infobulle de « Écart cumulé au FDA attendu » : la maquette C3EW propose « La FDA de chaque match moins celle que le modèle attendait de ce joueur dans ce match, cumulée depuis le premier match. Au-dessus de zéro : mieux qu'attendu. » ; la carte garde le texte PARTAGÉ `common.charts.fda_gap_tooltip` (`FdaGapTooltipText`, source unique des trois instances Séries temporelles / Sessions / Escouade, 2 phrases). Aligner sur la maquette = modifier `lib/i18n/manifests/common.toml` pour les trois pages (hors périmètre L1) ; noter aussi « le FDA » (manifeste) contre « la FDA » (maquette).
- Écart feuille / film sur les grenades (Madina97294 : 4 au film, 3 sur la feuille le 22/09) et un
  frag de la feuille sans ligne au film.
- (L2) ~~Aucun lot ne monte les cartes frags sur Contributions~~ — traité par L2.6 (superviseur,
  2026-09-27).
- (L2) Libellé FR du registre `hinf_frag_grenade` = « Grenade frag » (`weapon_names.toml`), la
  maquette écrit « Grenade à fragmentation » ; aligner = modifier le manifeste du titre (hors lot).
- (L2) Maquette C3EW : la mêlée y est lue au film (6 / 12 / 15), D8 la veut sur la feuille
  (6 / 13 / 16) — appliqué D8. Conséquence : Chocoboflor a une ligne de plus que sa feuille (64
  contre 63) et Madina97294 aucun reliquat (la ligne « Sans ligne au film » de la maquette
  disparaît ce soir-là) ; l'écart est tracé (`players_above_sheet`), pas corrigé.
- (L2) « Objet explosif (bidon) » prend la couleur « Non attribué » (maquette), alors que les
  bobines AVEC clé sont de classe `environmental` dans la Répartition des frags : la pastille ne
  reflète que les bidons sans clé. ACCEPTÉ par le superviseur le 2026-09-27.
- (L2) Halo 5 : « Outils de destruction » passe par le même builder, sur ses lignes natives
  (`weapon_kills`). Changements : une ligne par clé, dont « Environmental Explosives »
  (`h5_environmental`) qui y apparaît désormais nommée. (Grenades : corrigé par le complément
  L2 du 2026-09-27, une ligne « Grenade » au total de la feuille.) La Répartition H5 (golden)
  est inchangée.
- (L3) `MedalDigest` : le fond de la pastille « … dominante » vaut `${color}22` avec `color` = `var(--ac-squad-player-N)` — CSS invalide, aucun fond n'est peint. Gardé tel quel à l'extraction (brief : rendu des médailles inchangé) ; la coquille partagée peint par défaut `color-mix(… 14 %)` comme la maquette (fiches d'objectif). Aligner les médailles = une ligne dans `MedalDigest.tsx`.
- (L3) ~~Le drapeau neutre est écarté de l'historique (D6) mais PAS du « fil de la session » (« quel que soit le mode ») : une soirée avec du drapeau neutre aurait une valeur de fin du fil différente du point « ce soir ».~~ CORRIGÉ (correction superviseur L3, 2026-09-27).
- (L3) « Ce soir » = le périmètre D2 entier : un filtre qui couvre plusieurs sessions en fait UN point « ce soir » (libellé de session vide côté contrat).
- (L3) Maquette : un rôle à 0 sur 0 compte 0 % ; implémenté : il ne pèse pas (Go et web). Aucune soirée témoin n'est touchée.
- (L3) Abréviations de familles absentes de la maquette choisies par l'exécuteur : FR RC (Roi de la colline), C, R, E, V ; EN CTF, SH, KH, OB, SP, EX, VIP.
- (L3) ~~`teammates_service.go` passe de 522 à 527 lignes (déjà au-dessus de 500) : champ `objectiveModeEcarte` et appel de `loadUsageBlocks` à cinq champs.~~ CORRIGÉ : 501 lignes (correction superviseur L3, 2026-09-27).
- (L3) Le prédicat du drapeau neutre est injecté sous `CapMatchObjectiveStats` (même gate que les colonnes qu'il accompagne) : un 3e titre avec stats d'objectif recevrait le prédicat de Halo Infinite (inoffensif : il ne reconnaît que « neutral flag ctf »).
