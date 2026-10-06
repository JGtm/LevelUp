# Plan : page Sessions aux formes de l'Emprise — 2026-10-06

> Sources, à lire avant tout lot, qui FONT FOI pour le rendu :
> - maquette validée par l'utilisateur le 2026-10-06, position « Après », vues « Pleine page » ET
>   « Comparaison » : https://claude.ai/artifact/23K7pYangJFArbkCBLGaTu (v2), copie
>   `.ai/V7.5/MAQUETTE_SESSIONS_2026-10-06.html` (script lisible : liste `APRES` l. 846-869,
>   `renderApres` l. 874, `renderCompare` l. 884, `makeFragbar` l. 903, `makeTools` l. 933,
>   `makeControl` l. 958, `makeFil` l. 989, `makeGrid` l. 999, `makeMine` l. 1009, `makeProd`
>   l. 1019, `makeYield` l. 1051, `makeLife` l. 1085, `makeBalance` l. 1118, `makeSheets` l. 1163,
>   `makeEquip` l. 1197, `renderEquip` l. 1221, `renderMine` l. 1246, `meRestTrack` l. 1278,
>   `renderMineCompact` l. 1292, `renderFil` l. 1311, `renderGridCompact` l. 1349, `renderGrid`
>   l. 1379) ; le drapeau `cp` de chaque `make*` sépare la pleine page (`false`) de la vue compacte
>   (`true`). Ne se portent pas : lignes « remplace : … », encarts « Maquette. », pastilles
>   numérotées, entrée `riposte-off` (bandeau « Riposte retirée ») ;
> - relevés : `.ai/V7.5/MESURES_SESSIONS_2026-10-06.md` (§0 les trois sessions témoins, §1 relevés
>   SQL, §3 agrégats de la vue Comparaison) ;
> - plan frère, dont ce lot réutilise les briques : `.ai/PLAN_TIMESERIES_USAGES_EMPRISE_2026-10-05.md`
>   (décisions D1-D15, spécification S1-S10, journaux L1-L5) ;
> - `.ai/thought_log.md`, entrées du 2026-10-05 (relevé des rendus v1) et L1-L5 du 2026-10-06.
>
> Contrat d'exécution : skill `plan-execution` (ordre strict, un lot à la fois, gate passé avant le
> suivant, aucun item sans statut, zéro fix hors périmètre, découvertes consignées §8). Statuts :
> `[x]` fait et vérifié, `[~]` couvert ailleurs (référence), `[!]` non fait (justification écrite).
> Aucune case vide à la clôture d'un lot. « Clos » = les 5 actions de la règle 6 du skill.
>
> Statut du plan : **ACCEPTÉ par le superviseur le 2026-10-06 (D1-D18 fermes, réponses §9) ;
> rebasé sur `262e36b2e` (tête finale de `feat/ts-usages-emprise`, L6 `651bbe972`, L7 `bbe42dbc8`,
> L8 `262e36b2e`) le 2026-10-06, sans conflit ; GO phase 2, lot par lot, compte rendu et
> « continue » du superviseur à chaque clôture.** S6 est fondu dans S5 (§4.F rejoué sur la nouvelle
> base). Second rebase, demandé par le superviseur après S2, sur `554457c31` (L9 `545f765e8` + L9.2
> `554457c31`, corrections de revue du lot TS) : un conflit, `timeseries_service_lives.go` (corps
> déplacé par S1.2, ligne de journal ajoutée par L9) — version déplacée gardée, champ
> `ecartees_journal_non_publiable` reporté dans `solo_lives_block.go` ; généré régénéré sans écart.
> Branche : `feat/sessions-emprise`, créée sur `cd3145ec2`, rebasée sur `554457c31` ; worktree
> `C:\Users\Guillaume\Downloads\Scripts\LevelUp-wt-sessions`.
> **Clôture (2026-10-06) : S1 à S5 et S7 faits, S6 fondu dans S5 ; revue adversariale (S7.5)
> demandée au superviseur ; intégration dans `feat/v75` sur son accord.**

## 0. Objectif, critère de succès, hors périmètre

**Objectif.** La page Sessions (`features/session-detail/`) affiche ses statistiques du film aux
formes de l'Emprise, pour LE JOUEUR ACTIF seul (moi / mon camp / adversaire, jamais un segment par
coéquipier suivi, en contexte escouade comme en solo), dans l'ordre et les formes de la maquette
« Après », en pleine page ET dans le tiroir de comparaison (vue compacte normalisée des deux
colonnes, rangées partagées). « Riposte » quitte la page. Tout ce qui perd son dernier lecteur est
supprimé (règle n° 7), Go et web, contrat compris.

**Critère de succès.** (1) Les cartes A-L du §3 rendues dans l'ordre, conformes à la maquette en
pleine page ET en comparaison (S1-S14) ; (2) chaque carte existe pour la session comparée, dans sa
rangée partagée, avec le marqueur « Sans équivalent dans cette session » du côté qui ne l'a pas ;
(3) chaque carte se retire seule sans donnée et sur Halo 5 (jamais un 500, jamais un zéro inventé) ;
(4) tous les gates des lots verts, dernière exécution dans la session ; (5) inventaire §4 supprimé,
chaque preuve grep à 0 ; (6) docs du lot clôture à jour.

**Hors périmètre** (consigné, non traité) :
- Séries temporelles : le lot TS est clos (L8) et intégré à la base ; ce lot ne change ni son rendu
  ni ses tests de page (§5.1). `equipment_usage`, le nuage d'élévation, les champs non-objectif de
  `formes_retenues` et `LoadUsageFilmPads` n'existent plus (L6, L7).
- Escouade › Synergies (riposte, temps de riposte, morts ripostées, rôles de hauteur) : lot à part de
  l'utilisateur, non lancé. Son bloc `squad_echange` est distinct du bloc `coordination`.
- Vue match : son bloc `combat_tab.riposte` (`domain.MatchRiposteBlock`) est distinct.
- « Appui reçu » et « Portée des engagements » : inchangés (formes, données, compact existant).
- Pas d'habitude sur Sessions (décision utilisateur) ; pas de grille par carte (`maps`) sur Sessions.

## 1. Décisions

### 1.1 Validées par l'utilisateur (2026-10-06) — fermes

- **V1** Joueur actif seul : aucune barre ni segment par coéquipier suivi (`squad_players` et les
  lignes `squad[]` du bloc d'usage perdent tout lecteur).
- **V2** Tiroir : les deux colonnes en vue compacte (`SessionColumnBody`, prop `compact`, rangées
  partagées, marqueur quand une carte manque d'un côté) ; chaque carte neuve a sa variante compacte
  NORMALISÉE (parts en %, cadences par match, niveau ressource, demi-largeur sans défilement), celle
  de la maquette en « Comparaison ».
- **V3** Riposte retirée ; « Appui reçu » et « Portée des engagements » inchangées ; aucune notion de
  hauteur sur la page.
- **V4** Pas d'habitude.
- **V5** Ordre et contenu : §3 (cartes A-L, groupes « Match par match » puis « Frags et usages »).
- **V6** « Mon camp », jamais « Notre camp » ; « de la soirée » pour le périmètre d'une session.

### 1.2 Tranchées par le planificateur — FERMES (plan accepté par le superviseur le 2026-10-06 : D1-D18 confirmées ; Q1 oui — fichiers de L1-L5 du §5.1 modifiables, ajouts `compact` à défaut `false`, corps de `attachEmprise` / `attachLives` remplacés par un appel à signature inchangée, `empriseObjectName` centralisé, Escouade et Séries temporelles identiques à l'écran et tests de page rejoués sans modification ; Q2 option (a) — phase 2 après la clôture de L8, rebase de `feat/sessions-emprise` sur la tête finale de `feat/ts-usages-emprise` donnée par le superviseur, S6 fondu dans S5 à ce moment-là ; Q3 « Précision par arme » gardée pour Halo 5, D14)

- **D1 — Réutiliser les briques des lots L1-L5, en les étendant d'une prop `compact`.** La règle
  « ne recopie rien : importe » et l'exigence d'une variante compacte de CHAQUE carte (V2) imposent
  d'ajouter une variante aux composants livrés par L4-L5 (`squad/emprise/*`, `squad/objectif/*`,
  `timeseries/usages/*`) et à la Squad (`SquadFragBreakdownCard`, `SquadWeaponKillsChart`). Ajout
  rétro-compatible : `compact` vaut `false` par défaut, le rendu actuel de l'Escouade et des Séries
  temporelles ne change pas (leurs tests de page rejoués SANS modification). Aucun de ces fichiers
  n'est dans le périmètre de L6, L7, L8 (§5.1). *Contredit la consigne « ne toucher à aucun de ses
  fichiers » : question Q1.*
- **D2 — Import `session-detail => timeseries` déclaré.** Vérifié : la paire n'existe pas dans
  `tools/lint-cross-feature-imports.mjs` (`ALLOWED_CROSS_IMPORTS` l. 51-225 ; `session-detail=>explorer` l. 203,
  `session-detail=>squad` l. 224) et le ratchet est à 7 ≤ 7 (l. 386). Les quatre cartes propres au
  solo (`MinePickupsCard`, `EquipmentOutcomesCard`, `LivesNearTeammateCard`, et leurs modèles
  `usages.logic.ts`) vivent sous `features/timeseries/usages/` ; les déplacer toucherait 15 fichiers
  du lot TS. La paire est ajoutée à `ALLOWED_CROSS_IMPORTS` avec son commentaire (dépendance durable :
  Sessions et Séries temporelles montent la même Emprise solo), comme `session-detail=>squad`.
- **D3 — Un seul bloc Emprise par session, le type des Séries temporelles.** `domain.SoloEmpriseBlock`
  (`domain/solo_emprise.go:17-26`) réutilisé tel quel : `SquadEmpriseBlock` embarqué + `Equipment` ;
  `Maps` n'est pas calculé sur Sessions (champ nul : la grille par carte n'existe que sur les Séries
  temporelles). Champs de réponse `SessionPageResponse.Emprise` / `CompareEmprise`
  (`json:"emprise,omitempty"`, `json:"compare_emprise,omitempty"`). Le web lit le même type généré.
- **D4 — Assemblage de l'Emprise solo factorisé (2e appelant).** Le corps de
  `TimeseriesService.attachEmprise` (`service/timeseries_service_emprise.go:61-96` : trois lectures
  `squadagg.EmpriseLecteur`, joueur seul, `WithoutTimeScale`, `Build`, `BuildEquipment`) devient
  `buildSoloEmpriseBlock(ctx, soloEmpriseQuery) *domain.SoloEmpriseBlock` dans
  `service/solo_emprise_block.go` (NEUF, paquet `service`, les deux pages y sont) ; `attachEmprise`
  l'appelle avec `WithMaps: true` (signature et tests de `attachEmprise` inchangés). Sessions l'appelle
  avec `WithMaps: false`. Les projections existantes `timeseriesEmpriseMatches`
  (`timeseries_service_emprise.go:110-126`) et `timeseriesFormesMetas`
  (`timeseries_service_sections.go:275-291`) prennent des lignes canoniques : Sessions les appelle sur
  les lignes canoniques de la session (même paquet, aucune copie ; nom préfixé `timeseries`, voir §8).
- **D5 — « Mes vies » factorisée (2e appelant).** Le corps de `TimeseriesService.attachLives`
  (`service/timeseries_service_lives.go:41-76`) devient `lireViesPresOuSeul(ctx, viesQuery)
  *domain.TimeseriesLivesNearTeammate` dans `service/solo_lives_block.go` (NEUF) ; journaux
  `vies_*` avec l'attribut `page` (`timeseries`, `sessions`) — aucun test n'asserte les noms
  `timeseries_vies_*` (grep des `_test.go` : 0). Champs `SessionPageResponse.LivesNearTeammate` /
  `CompareLivesNearTeammate`. Câblage Sessions sous `CapFilmKillPositions`, comme
  `registry_pages_timeseries.go:27-29`.
- **D6 — « Outils de destruction » : le builder de l'Escouade partagé.** Vérifié :
  `buildSquadWeaponTools` (`service/teammates/teammates_squad_weapon_tools.go:87-231`) est PUR et
  privé au paquet `teammates` ; il n'est pas « du film » seul (film, table native, feuille de match).
  Déplacé dans `service/squadagg/weapon_tools.go` (`BuildWeaponTools(WeaponToolInputs)`,
  `PlayersAboveSheet`), tests déplacés avec lui ; `teammates` garde ses lectures
  (`loadSquadToolCategories`, `squadSheet`, `buildSquadToolsSection`) et appelle le builder. Sessions :
  lignes d'arme = `loadSessionWeaponKillRows` déjà lu (`session_page_frag_distribution.go:102-122`),
  catégories de source par l'interface optionnelle `port.KillSourceCategoryRepository`
  (`port/kill_source.go:74-80`) du même repo d'armes, feuille = les compteurs déjà calculés
  (`session_page_frag_distribution.go:83-91`), un seul joueur. Champ
  `SessionCompareEntry.WeaponTools` (`json:"weapon_tools,omitempty"`, type `domain.SquadWeaponTools`).
- **D7 — Objectif : la partie objectif de `formes_retenues`, forme réduite de L7.**
  `squadagg.BuildSquadFormesBlock` (`squadagg/squad_formes.go:55`, requête réduite par L7 :
  `Repo port.SessionUsageRepository`, `Objectives`, `PlayerXUID`, `MainGamertag`, `Metas`,
  `SelectedGamertags`, `Lectures`) sur les matchs de la session,
  champs `SessionPageResponse.FormesRetenues` / `CompareFormesRetenues`
  (`json:"formes_retenues,omitempty"`, `json:"compare_formes_retenues,omitempty"`). Le web ne lit QUE
  les champs que L7 garde (`available`, `unavailable_reason`, `matches_total`, `matches_measured`,
  `main_xuid`, `squad`, `matches[].{match_id, start_time, mode_label, map_label, player_team,
  objective}`) ; garde-rail web (S4.15) qui l'interdit de lire un autre champ. Vérifié : les prises
  nettes de drapeau sont DÉJÀ une colonne optionnelle `flag_grabs_net` de l'objectif des formes
  (`squadagg/squad_formes.go:159-182`), sous le rôle « prendre » — `SessionObjectivesBlock` et son
  `FlagGrabsNet` sont donc remplacés en entier, rien à déplacer. `SelectedGamertags` vide (V1).
  Le golden `analysis/squadformes/testdata/objectif_publie.golden.json` (L7) fixe la projection lue.
- **D8 — Une lecture du résumé d'usage par session (ADR 0036 I4).** `squadagg.LireUsage` une fois par
  scope (session affichée, session comparée), partagée par l'Emprise, les formes, l'effectif de camp
  de la coordination (`sessionusage.BuildTeamContext` sur ses participants) et — tant qu'il vit
  (jusqu'à S5) — l'ancien bloc d'usage. Section de durée `usage_summary`.
- **D9 — Coordination recâblée sans le bloc d'usage.** Vérifié : l'effectif de camp vient aujourd'hui
  de `buildSessionUsage` (`session_page_usage.go:84-90`) et le xuid de la coordination de
  `usageXUID`, posé par `WithSessionUsage` (`session_page_coordination.go:90-97`,
  `registry_pages.go:343-345`, câblé seulement sous `film.usage_summary`). Après le lot : effectif de
  camp depuis D8 ; xuid depuis `WithSessionEmprise(repo, xuid)`, câblé INCONDITIONNELLEMENT (comme
  `WithEmprise` des Séries temporelles) — la coordination garde sa porte propre
  (`JournalDesMortsFiable`). Sans résumé d'usage (Halo 5) : pas d'effectif, coordination sans parité,
  comme aujourd'hui.
- **D10 — Riposte retirée du contrat.** Lecteurs web du bloc `coordination` après le lot : la carte
  « Appui reçu » de Sessions (`appui`, `per_match` versant appui, `matches_*`, `available`) et la
  frise des Séries temporelles (`sessions[].appui`). Sortent du contrat et du calcul :
  `CoordinationRiposte` (`domain/coordination_block.go:86-114`), `Riposte` de `CoordinationBlock`
  (l. 198) et de `CoordinationSessionPoint` (l. 179), les champs riposte de `CoordinationMatchPoint`
  (l. 148-156), `FenetreMs` (l. 191, seul lecteur web `fenetreSeconds` de la carte Riposte,
  `coordinationModel.ts:207-210`), `compterRipostes` / `agregerRiposte` / `medianeMs` et les champs
  riposte de `cumulMatch` (`analysis/coordination/bloc.go:34-44,128-198,219-235`), l'habituel de
  « je suis couvert » (`session_page_coordination.go:130-137`). `coordination.Echanges` et
  `FenetreEchangeMs` RESTENT (Escouade, vue match). Preuves §4.D.
- **D11 — Sections en clés par CARTE.** Le modèle `_sections.ts` (clés stables, groupes titrés,
  `groupSessionSections`, `mergeSessionSectionKeys`) passe de deux clés (`frags`, `usage`) à une clé
  par carte : `frag_bar` (A), `tools` (B), `weapon_accuracy` (D14), `control` (C), `fil` (D), `grid`
  (E), `mine` (F), `production` (G), `yield` (H), `lives` (I), `objective_balance` (J),
  `objective_sheet` (K), `equipment` (L) ; et gagne un SOUS-GROUPE titré (intertitre `h3`) :
  `resources` (C-F, « Ressources de la soirée » + couverture en pleine page), `prendre` (G-H),
  `lives` (I), `objectif` (J-K), `equipment` (L). En comparaison chaque clé est UNE rangée partagée ;
  l'intertitre de groupe et de sous-groupe se pose dans la rangée de sa première clé présente, des
  deux côtés (même règle que les titres actuels, `SessionColumnBody.tsx:240-267`). En pleine page,
  les paires A|B, C|D, G|H partagent une rangée (`pairGridClass`, `_chartSections.tsx:90`) — une
  seule colonne en `compact` (garde `sessionShrink.guard.test.ts`).
- **D12 — Textes de la page.** Fichier `features/session-detail/sessionEmpriseText.ts` (NEUF) :
  surcharges `Record<Locale, …>` de `EMPRISE_TEXT_SOLO`, `OBJECTIF_TEXT_SOLO`, `USAGES_TEXT`
  (`timeseries/usages/usagesText.ts:252-254`) avec « la soirée » au lieu de « le périmètre », FR mot
  pour mot de la maquette, EN traduits ; DEUX jeux par carte quand la maquette distingue les ⓘ
  (pleine page / comparaison : A, B, C, E, F, J, K, L) — la carte reçoit le jeu de sa vue, aucun type
  de texte existant ne change. Intertitres des sous-groupes : clés du manifeste `session`
  (`lib/i18n/manifests/session.toml`, à côté de `section_kills_usage` l. 164), manifeste régénéré.
- **D13 — Lignes « en attente » de la maquette non portées** (même décision que D13 du plan TS) : une
  ressource sans prise n'a pas de ligne (`emprise.logic.ts:65-72`), une carte sans ligne se retire ;
  « 0 prise dans les deux camps », « Non mesuré sur les N matchs » ne se portent pas. Les phrases
  conditionnelles d'ⓘ qui décrivent les données d'illustration (maquette l. 1024 : « Dans cette
  soirée, aucun bonus n'est pris au sol… ») ne se portent pas.
- **D14 — Halo 5 garde « Précision par arme ».** Vérifié : la seule surface de
  `SessionCompareEntry.weapon_accuracy` est `SessionFragCard` (`SessionFragCard.tsx:60,78-87`), qui
  disparaît ; la maquette (Halo Infinite) ne la montre pas, le brief n'en dit rien. Décision : clé
  `weapon_accuracy` après B, gatée par la donnée (comme l'Escouade, `SquadFragSection.tsx:97-112`),
  composant existant `WeaponAccuracyChart` inchangé ; `top_weapon_kills` reste (il le lit). *Question
  Q3 : si l'utilisateur la retire, la chaîne Go `WeaponAccuracy` de Sessions sort en S5.*
- **D15 — Score et dominance des lignes de match.** La carte D (bande de résultats, encoche de
  dominance) et les en-têtes de E lisent score et dominance ; `SessionDetailMatchRow`
  (`domain/session_page.go:35-102`) n'a ni l'un ni l'autre. Ajout `score_label` (via
  `analysis.buildScoreLabelCanonical`, `analysis/home_canonical_recent.go:240`, EXPORTÉ — source
  unique du libellé, ADR 0032 ; table `roundsDecideFor(pdb)` injectée par `WithRoundsDecide`) et
  `dominance_flag` (`StatsMatchRow.DominanceFlag`, `legacymatch/types.go:137`), posés par une passe
  dans un fichier neuf (`session_page_match_scores.go`) pour ne pas agrandir
  `session_page_service.go` (894 L, au-delà du seuil, dette gelée).
- **D16 — Vue compacte de « Mes vies » : celle de la maquette.** Le brief écrit « identique » ; la
  maquette (`makeLife` l. 1095-1108) écrit les parts seules (`pctI`) en compact et « n · % » en pleine
  page. La maquette fait foi.
- **D17 — Nom des objets : 3e copie → helper.** La fonction « nom d'un objet de l'Emprise » (bonus par
  l'i18n d'usage, véhicule par `vehicleFamilyName`, arme par son libellé) existe en deux copies
  (`squad/emprise/useEmpriseModels.ts:29-36`, `timeseries/usages/useUsagesModels.ts:29-36`) ; Sessions
  en ferait une 3e. Règle n° 6 : `empriseObjectName(o, usageText, unknownVehicle)` dans
  `squad/emprise/emprise.logic.ts`, les deux copies migrées, garde-rail grep (S3.1).
- **D18 — Attribution des commits** : ligne système de cette session
  (`Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`), comme D15 du plan TS.

## 2. Spécification de rendu (non négociable)

Reprend S1-S10 du plan TS (§2) ; en cas de doute, la maquette fait foi, puis ce §2.

- **S1** Titre factuel, ⓘ si besoin (texte de la maquette), graphique, légende centrée en bas — rien
  d'autre (aucune phrase de lecture, aucun pied de carte ; les `mnote` de la maquette ne se portent pas).
- **S2** Valeurs DANS les segments si elles tiennent (`components/charts/segmentLabelFit.ts`), repli
  au-dessus sinon, jamais seulement en infobulle.
- **S3** `team-ally` / `team-enemy` à la place des mots ; hachure réservée à « sans film ».
- **S4** Trait 50 % pointillé `warning`, « 50 % : autant que l'adversaire ».
- **S5** Aucun vert / rouge sur une répartition interne (moi `squad-player-1` / reste de mon camp
  `TEAM_REST_INK`, le jeton `team-rest` n'existant pas — découverte L5 du plan TS).
- **S6** Couleurs de ressource `resource-*`, pastille devant chaque nom ; couleurs de classe de frag
  pour A et la pastille de B.
- **S7** FR + EN (`Record<Locale, …>`), « FDA » jamais « KDA », aucun anglicisme
  (`lib/i18n/no-anglicisms.guard.test.ts`), aucun emoji.
- **S8** Aucune couleur en dur ni classe Tailwind de couleur (`tools/lint-no-hardcoded-colors.mjs`).
- **S9** « Mon camp », « de la soirée ».
- **S10** Chaque carte se retire seule sans donnée ; groupe ou sous-groupe sans carte → pas
  d'intertitre (règle actuelle, `sessionSectionVisibility.ts:1-19`).
- **S11 (compact)** Vue compacte = celle de la maquette avec `cp = true` : parts en % dans les
  segments, comptes au survol ; niveau ressource (F, J) ; aucune ligne d'objet dépliable ; graphe D
  plus bas (170 contre 246) avec étiquettes de fin seules et sans dates sous l'axe ; grille E
  réduite aux lignes de ressource (bonus, armes spéciales prises, armes spéciales frags, râteliers,
  véhicules) avec la part de mon camp dans la case (« 71 % »), « — » hachuré sans film, « ? » sans
  niveaux, en-tête heure + carte + V / D, colonnes `minmax(0, 1fr)` (aucun défilement horizontal).
- **S12 (compact)** La rangée partagée d'une carte absente d'un côté porte le marqueur existant
  (`SessionSectionPlaceholder`, `SessionColumnBody.tsx:195-208`).
- **S13** Pleine page : paires A|B, C|D, G|H en deux colonnes ; E, F, I, J, K, L pleine largeur ;
  « Appui reçu » seule dans sa rangée (demi-largeur, comme D11 du plan TS).
- **S14** Intertitre « Ressources de la soirée » suivi en pleine page de « n matchs filmés sur N ·
  frags de la feuille de match sur les N » (maquette l. 854) ; aucun petit texte en comparaison.

## 3. Cartes cibles (ordre à l'écran)

| # | Carte | Source Go (contrat) | Composant web (pleine page / compact) |
|---|---|---|---|
| — | « Match par match » : « Appui reçu » | `coordination` / `compare_coordination` sans riposte (D10) | `SessionCoordinationSection` réduite à l'Appui |
| — | « Portée des engagements » | INCHANGÉ (`range_profiles`, `range_reference`) | `SessionRangeCard` |
| A | Répartition des frags | `current_session.frag_distribution` (existant) | `squad/SquadFragBreakdownCard` un joueur ; compact : parts, total en sous-libellé |
| B | Outils de destruction | `current_session.weapon_tools` (D6, NEUF) | `squad/SquadWeaponKillsChart` + `buildSquadToolRows` ; compact : 6 premiers, part de mes frags |
| B' | Précision par arme (Halo 5) | `current_session.weapon_accuracy` (existant) | `WeaponAccuracyChart` inchangé (D14) |
| C | Contrôle des ressources | `emprise.resources` | `squad/emprise/ResourceControlCard` ; compact : %, râteliers en une ligne |
| D | … au fil de la session | `emprise.matches` + `matches[].{outcome, score_label, dominance_flag}` (D15) | `squad/emprise/ResourceFilCard` axe `match` ; compact : plus bas, étiquettes de fin |
| E | … match par match | `emprise.matches` | `squad/emprise/ResourceMatchGridCard` ; compact : grille réduite (S11) |
| F | Mes prises dans mon camp | `emprise.objects[].squad` + `resources[powerup].outcomes` | `timeseries/usages/MinePickupsCard` ; compact : une barre par ressource, bonus perdus en % |
| G | Frags obtenus avec les ressources | `emprise.production` | `squad/emprise/ProductionCard` ; compact : % seuls |
| H | Rendement face à l'adversaire | `emprise.production` | `squad/emprise/YieldCard` ; compact identique |
| I | Mes vies : près d'un coéquipier ou seul | `lives_near_teammate` (D5, NEUF sur Sessions) | `timeseries/usages/LivesNearTeammateCard` ; compact : parts (D16) |
| J | Rapport de force par famille de mode | `formes_retenues` objectif (D7, NEUF sur Sessions) | `squad/objectif/ObjectiveBalanceCard` ; compact : une barre par rôle et par famille, % |
| K | Ma part à l'objectif | `formes_retenues` + `player_emblem_url` | `squad/objectif/ObjectiveSoloSheetCard` ; compact : le nombre devient ma part en % |
| L | Équipement pris, et ce que j'en ai fait | `emprise.equipment` | `timeseries/usages/EquipmentOutcomesCard` ; compact : barre 100 % en %, fine gardée, « n objets » |

Contrat ajouté (Go, `internal/domain/`) — forme figée, noms définitifs au lot :

```go
// domain/session_page.go — SessionPageResponse
Emprise                  *SoloEmpriseBlock            `json:"emprise,omitempty"`
CompareEmprise           *SoloEmpriseBlock            `json:"compare_emprise,omitempty"`
LivesNearTeammate        *TimeseriesLivesNearTeammate `json:"lives_near_teammate,omitempty"`
CompareLivesNearTeammate *TimeseriesLivesNearTeammate `json:"compare_lives_near_teammate,omitempty"`
FormesRetenues           *SquadFormesBlock            `json:"formes_retenues,omitempty"`
CompareFormesRetenues    *SquadFormesBlock            `json:"compare_formes_retenues,omitempty"`
PlayerEmblemURL          string                       `json:"player_emblem_url,omitempty"`
// domain/session_compare.go — SessionCompareEntry
WeaponTools *SquadWeaponTools `json:"weapon_tools,omitempty"`
// domain/session_page.go — SessionDetailMatchRow
ScoreLabel    string `json:"score_label,omitempty"`
DominanceFlag int    `json:"dominance_flag,omitempty"`
```
Retiré (S5) : `usage`, `compare_usage` et tout type qu'ils sont seuls à atteindre (§4.E, §4.F, tous
deux en S5 depuis le rebase) ;
riposte et `fenetre_ms` du bloc `coordination` (§4.D).

## 4. Inventaire des suppressions — preuves par grep (relevées le 2026-10-06, à REJOUER avant de supprimer)

Chaque preuve se rejoue par `Grep` (outil) sur `apps/web/src` (hors `lib/api/generated.ts`) ou
`apps/go-api/internal`, fichiers de test exclus sauf mention. Attendu APRÈS suppression : 0
occurrence hors fichiers supprimés et hors commentaires historiques (eux corrigés s'ils deviennent
faux, règle 17). Juge de paix web : `node tools/knip-ratchet.mjs` à 0 / 0 / 0.

- **A. Cartes d'usage de Sessions (web).** `SessionUsageSection` n'est monté que par
  `SessionColumnBody.tsx:55,151`. Sortent : `SessionUsageSection.tsx` (+ `.gate.test.tsx`),
  `SessionUsageEquipmentCards.tsx`, `SessionPadControlCards.tsx`, `SessionUsageShared.tsx`,
  `SessionUsageFlagGrabsNet.test.tsx`, la partie usage de `sessionSectionVisibility.ts`
  (`sessionUsageCardsShown`, `sessionUsageShowsSomething`, l. 27-74) et ses lecteurs
  (`SessionDetailPage.tsx:44-46,264,270`).
- **B. `SessionFragCard` (web).** Monté par `SessionColumnBody.tsx:51,145` seulement. Sort avec ses
  imports devenus orphelins si knip le dit (`FragSunburst`, `FragWeaponBreakdown`,
  `fragDetailBreakdown` : lecteurs hors Sessions à vérifier — `FragSunburst` est lu par
  `match-view/MatchFragCard.tsx`, `synthesis`, `timeseries` ; il RESTE). `sessionFragCardHasContent`
  (`sessionSectionVisibility.ts:93-100`) est remplacé par le prédicat de A / B.
- **C. Carte Riposte (web).** `SessionCoordinationSection.tsx:172-190` (carte Riposte),
  `coordinationModel.ts` : `buildRiposteGaugeRows` (l. 88-108), `buildRiposteBand` (l. 168-174),
  `formatDelaiMedian` (l. 195-205), `fenetreSeconds` (l. 207-210) ; `coordinationI18n.ts` : clés
  riposte (`cardRiposte`, `gaugeCovered`, `gaugeIRiposte`, `bandRiposte`, `delaiMedian`,
  `infoRiposte1-3` — liste exacte relue au lot) FR et EN ; cas de `SessionCoordination.test.tsx` et
  `SessionCompareRows.test.tsx:213-232` qui l'exercent (adaptés à l'Appui seul).
- **D. Riposte du contrat (Go + web).** Grep relevé : lecteurs web de `.riposte` / `je_suis_couvert` /
  `je_riposte` / `delai_median_ms` / `team_deaths_avenged` / `covered_share_pct` /
  `riposte_share_pct` / `my_ripostes` : `session-detail/coordinationModel.ts`,
  `SessionCoordinationSection.tsx`, `SessionCoordination.test.tsx`, `SessionCompareRows.test.tsx`,
  `timeseries/TimeseriesCoordinationSection.test.tsx:42-88` (fixtures et un test de
  `serieDeSoirees` sur `p.riposte.je_suis_couvert`, réécrit sur `p.appui.on_me_prepare`) ;
  `match-view/*` et `squad/squadRiposte*` lisent d'AUTRES blocs (`MatchRiposteBlock`,
  `squad_echange`) et RESTENT ; `timeseries/timeseriesCoordination.logic.ts:81` ne cite la riposte
  qu'en commentaire (corrigé). Go : D10.
- **E. Bloc `usage` / `compare_usage` (Go, S5).**
  `service/session_page_usage.go` (318 L) et son test, `service/session_page_usage_labels.go` (+ test),
  `service/session_page_flag_grabs_net_test.go`, `service/pad_tiers_wiring_test.go` (appelle
  `WithSessionUsage`, l. 43), champs `Usage` / `CompareUsage` (`domain/session_page.go:148-149`),
  `WithSessionUsage` et le résolveur d'amis de Sessions (`session_page_service.go:65-71`,
  `registry_pages.go:343-345`, factory `SessionPage` l. 318-366), `analysis/sessionusage/objectives.go` (+ test : lecteurs code
  `session_page_usage.go` seul, `objective_role_rows_repo.go:10` n'est qu'un commentaire),
  `ComputeFlagGrabsNet` et `FlagGrabsNetInput` (`sessionusage/flag_grabs_net.go:61-145`, + cas de test ;
  `FlagGrabsNetRow` RESTE, lu par `duckdb/squad_formes_repo.go` et `port/session_usage.go:65`),
  `ResolveTrackedSquad` (`sessionusage/squad.go:31-92`, + cas de test ; `ResolveScopeFriends` RESTE),
  `platform/duckdb/objective_role_rows_repo.go` (+ test : `LoadObjectiveRoleRows`, seul lecteur
  l'interface `objectiveRoleRowsLoader` de `session_page_usage.go:31-33`), les types de domaine que
  ces seuls fichiers atteignent : `SessionObjectivesBlock`, `SessionObjectiveFamilyBlock`,
  `SessionObjectiveRoleMetric`, `SessionFlagGrabsNetBlock`, `SessionUsageSquadShare` si `squad.go`
  n'en a plus besoin, le champ `Objectives` de `SessionUsageBlock`.
- **F. Anciens orphelins d'intersection — supprimés en S5 (preuves REJOUÉES sur `262e36b2e`).** L6 a
  supprimé `domain/equipment_usage.go`, `sessionusage/usage_overview.go` et réduit
  `squadagg/equipment_usage.go` à `squadagg/lectures_usage.go` (`LireUsage`, `LecturesUsage`) ; L7 a
  supprimé `port.SquadFormesUsageRepository`. Lecteurs hors tests relevés sur la nouvelle base :
  `sessionusage.ComputeUsage` → `sessionusage/usage.go`, `session_page_usage.go` (+ commentaire
  `port/session_usage.go`) ; `ComputePadTiers`, `PadTiersInput` → `pad_tiers.go`,
  `session_page_usage.go` ; `squadagg.NommerArmesDesNiveaux` → `squadagg/pad_tier_labels.go`,
  `session_page_usage_labels.go` ; `domain.SessionUsageBlock` → `usage.go`, `session_page_usage.go`,
  `session_page_usage_labels.go` ; `SessionUsageMetric` → `sessionusage/{squad,usage,usage_outcomes}.go`
  (+ commentaire `domain/equipmentusage/families.go:18`) ; `SessionUsageOutcomes` →
  `usage.go`, `usage_outcomes.go` ; `SessionUsageShares`, `SessionUsagePadFamily`,
  `SessionUsagePowerup` → `usage_families.go` ; `SessionUsageMatchPoint` → `team_context.go`
  (`newMatchPoint`) ; `SessionUsagePadTiersBlock` / `PadTier` / `PadTierWeapon` → `pad_tiers.go`,
  `squadagg/pad_tier_labels.go`. Tous ne sont donc lus que par la chaîne de Sessions et sortent en S5 :
  `ComputeUsage`, `Input`, `metricKeys` et le reste de `usage.go` hors `PlayerRow` / `FilmRow` /
  `ParticipantRow` / `MatchInput` / `BuildMatchInputs` (lus par `squademprise`, `squadformes`,
  `squadagg`) ; `usage_families.go` ; `newMatchPoint` de `team_context.go` (`TeamContext`,
  `BuildTeamContext` RESTENT) ; `usage_test.go` ; les cas `ComputeUsage` de `usage_outcomes_test.go`
  (et `TestBilan_MetricKeysSurLeSeulSujet`, ajouté par L6 sur `metricKeys`) ; `computeOutcomes` /
  `attachOutcomes` / `subjectBilanFamilies` de `usage_outcomes.go` (`equipmentOutcomeOf` /
  `equipmentUsedOf` / `outcomeCountsOf` RESTENT, lus par `PlayerOutcomeCounts`, et le golden
  `usage_outcomes_golden_test.go` est relu : il reste s'il ne teste que ce qui reste) ;
  `ComputePadTiers` + `PadTiersInput` (`PadTierRow` RESTE) ; `squadagg/pad_tier_labels.go` (+ test) ;
  types `SessionUsageBlock`, `SessionUsageMetric`, `SessionUsageOutcomes`, `SessionUsageShares`,
  `SessionUsageMatchPoint`, `SessionUsagePadFamily`, `SessionUsagePowerup`, `SessionUsagePadTier*`
  (`domain/session_usage.go`) — `SessionUsageSquadPlayer`, `SessionUsageUnsupported`,
  `SessionUsageLoadFailed`, `PadTier*`, `PadTierOrder` RESTENT ; alias web de
  `lib/api/types.ts:2299-2312` ; `_shared/usage/usagePadTiersModel.ts` (lu par la seule page Sessions,
  note L8.2 du plan TS).
- **G. `_shared/usage/*` (web, S4).** Restent, lus par la carte « Appui reçu »
  (`SessionCoordinationSection.tsx:33-40`, `coordinationModel.ts`) et ailleurs : `UsageForms`
  (`UsageGaugeGrid`), `UsageRegularityBand`, `UsageBandLegend`, `UsageEmptyNotice`, `usageCardTitle`,
  `usageI18n` (aussi `squad/emprise/useEmpriseModels.ts`, `timeseries/usages/useUsagesModels.ts`),
  `usageGaugeModel`, `usageRegularityBandModel`, `usageFormat` (aussi `squad/formes/format.ts`),
  `usageMetricKinds` (aussi `squad/objectif/*`). Candidats à la suppression dès que A sort (knip
  juge) : `usageAvailability` (+ test), `usageGrids` (+ test), `UsageLobbyTrack`,
  `usageLobbyTrackModel` (+ test), `usageObjectives`, `usageParity` (+ test), `usagePadTiersModel`
  (+ test), `UsageHatchLegend`, `UsageA2Ajustements.test.tsx`, les exports et clés de `usageI18n` /
  `usageMetricKinds` sans lecteur. Gardes à adapter (jamais désactiver) :
  `usageEmptyStateCanonical.guard.test.ts`, `noLocalUsageCopies.guard.test.ts`.
- **H. Bouts devenus morts.** Clés de manifeste `session.detail.*` et `frags.*` sans lecteur (manifest
  régénéré par `node apps/web/scripts/build_i18n_manifests.mjs`) ; types TS retirés du snapshot
  `lib/api/contract-surface.snapshot.json` par la procédure (`UPDATE_CONTRACT_SURFACE=1`),
  disparitions listées au journal du lot ; commentaires de `REFERENCE_CANAUX_EQUIPEMENT` (S7).

## 5. Organisation et gates communs

### 5.1 Fichiers du lot TS : ce qui se touche, ce qui ne se touche pas

- Le lot TS est clos (L8, `262e36b2e`). Ses fichiers ne se touchent que pour les items de ce plan qui
  les nomment : S5 supprime dans `sessionusage/usage_outcomes_test.go` (cas `ComputeUsage`,
  `TestBilan_MetricKeysSurLeSeulSujet`) et dans `port/session_usage.go` (commentaire de
  `ComputeUsage`) ; aucun autre fichier de L6-L8 (`timeseries_service.go`,
  `timeseries_service_sections.go`, `squadagg/lectures_usage.go`, `squadagg/squad_formes.go`,
  `domain/squad_formes.go`, `analysis/squadformes/*`, `domain/timeseries.go`, docs hors lignes
  propres à Sessions — S7.1) n'est modifié. Un second rebase après la revue adverse du lot TS se fait
  sur demande du superviseur seulement.
- Touchés par D1 / D4 / D5 / D17 (fichiers livrés par L1-L5 — Q1 accordée le 2026-10-06 : ajouts
  rétro-compatibles, rendu Escouade et Séries temporelles identique, tests de page rejoués sans
  modification) :
  `service/timeseries_service_emprise.go`, `service/timeseries_service_lives.go` (corps remplacés par
  un appel, signatures inchangées) ; `squad/emprise/{ResourceControlCard, ResourceFilCard,
  empriseCharts, ResourceMatchGridCard, ResourceGridTable, ProductionCard, PisteCampsForm,
  emprise.logic, useEmpriseModels}` ; `squad/objectif/{ObjectiveBalanceCard, ObjectiveSoloSheetCard,
  objectif.logic}` ; `timeseries/usages/{MinePickupsCard, EquipmentOutcomesCard,
  LivesNearTeammateCard, usages.logic, useUsagesModels}` ; `timeseries/TimeseriesCoordinationSection.test.tsx`
  (fixtures riposte, S5).
- Partagés, conflit éventuel (second rebase) résolu par RÉGÉNÉRATION : `apps/go-api/openapi.yaml`,
  `apps/web/src/lib/api/generated.ts`, `lib/api/contract-surface.snapshot.json` ; partagés à hunks
  disjoints : `api/wire/registry_pages.go` (factory `SessionPage` l. 318-366 seulement),
  `lib/api/types.ts` (alias Sessions), `.ai/thought_log.md` (ajout en fin).

### 5.2 Règles d'exécution et gates

- Exécuteur seul, dans le worktree, lots SÉQUENTIELS. Aucun sous-agent, aucun push, aucun merge,
  aucun `git stash`, aucun `git add -A` (stager fichier par fichier), aucun `--no-verify`, aucun
  Python, aucune base de `data/` ouverte, aucun serveur arrêté ou relancé, une commande `go` à la
  fois, aucune commande longue sans sortie (le chien de garde coupe à 600 s : tests Go par lots de
  paquets, comme les journaux L1-L3 du plan TS).
- Environnement Go, à chaque appel PowerShell :
  `$env:Path = "C:\msys64\ucrt64\bin;$env:Path"; $env:CGO_ENABLED = "1"; $env:CC = "C:\msys64\ucrt64\bin\gcc.exe"`.
  Web : `npm ci` dans `apps/web` du worktree (node_modules réel) avant le premier gate web.
- **Gate Go** (depuis `apps/go-api`) : `go build ./...` ; `gofmt -l internal` muet ; `go vet` des
  paquets touchés ; `go test -count=1` des paquets touchés puis du module en lots couvrant tout
  `go list ./...` (cmd + contracttest + analysis + api + domain + port + archlint ; games ;
  platform + service ; sync + persist + migration ; reste de internal ; pkg + scripts + tests) ;
  `go test -tags=integration -p 1 ./internal/platform/duckdb/...` dès que `platform/duckdb` bouge
  (S5) ; garde-rails nommés : `TestAucunTypeAnalysisEnCorpsHuma`, `TestNoNewSlugComparison`,
  `TestAnalysisImporteAucunPaquetDeTitre`, `TestNoLocalRadarRangeLookup`, `TestCampaignExclusionGuard`,
  `TestAucunTauxNu`, `TestNoRawAppendOnlyReads`, `TestLecturesDeLaVueDesNoms_Ratchet` ;
  `make go-api-lint` (Git Bash) 0 issue ; contrat : `go run ./cmd/openapi-gen`, puis
  `go run ./cmd/openapi-gen -check`, `npm run generate-types` (dans `apps/web`) et
  `node tools/check-generated-types-fresh.mjs` (racine).
- **Gate web** (depuis `apps/web`, vitest hors sandbox) : purge `node_modules\.tmp` ;
  `npx tsc -b --force` ; `npm run lint` (0 erreur) ; `npx vitest run --pool=forks` ; depuis la
  racine : `node tools/knip-ratchet.mjs` (0 / 0 / 0), `node tools/lint-no-hardcoded-colors.mjs`
  (0), `node tools/lint-cross-feature-imports.mjs` (≤ 7, aucune dérogation morte),
  `npx lefthook run pre-push` avec `C:\msys64\ucrt64\bin` et `C:\Program Files (x86)\GnuWin32\bin`
  au PATH (découverte L4 du plan TS).
- **Seuils** (CLAUDE.md règle 5) : fichier ≤ 500 L, fonction ≤ 80 L, ≤ 5 paramètres, complexité ≤ 12
  pour tout fichier créé ou modifié ; mesure jointe au journal du lot. Fichiers déjà au-delà, à NE PAS
  agrandir (taille avant / après au journal) : `service/session_page_service.go` 894 L,
  `api/wire/registry_pages.go` 620 L, `features/session-detail/SessionDetailPage.tsx` 551 L,
  `squad/emprise/empriseStrings.ts` 500 L, `lib/api/types.ts`.
- **TDD** : chaque règle neuve a son test ROUGE écrit et vu rouge AVANT le code (compilation ET
  comportement contre un bouchon) ; puis vert ; puis au moins UNE MUTATION par règle, annulée ensuite,
  consignée au journal du lot (« mutation : … → rouge »). Refactorisation sans changement de
  comportement : les tests existants restent verts SANS modification d'assertion, et une mutation
  prouve qu'ils mordent sur le code déplacé.
- **Clôture de lot** = gate vert + items statués + section du lot mise à jour ici + entrée en FIN de
  `.ai/thought_log.md` + commit local `feat(sessions-emprise/<lot>): …` (fichiers stagés un par un) +
  point d'étape au superviseur.

## 6. Lots

### S1 — Go : factorisations préalables, sans changement de comportement · moyen

- [x] S1.1 `service/solo_emprise_block.go` (NEUF) : `soloEmpriseQuery{Page, Player, PlayerXUID,
  RepoRoot, TitleSlug, Locale, Current []squademprise.Match, Lectures *squadagg.LecturesUsage,
  UsageRepo, EmpriseRepo, VehicleRepo, WithMaps bool}` et `buildSoloEmpriseBlock` = corps actuel de
  `attachEmprise` (`timeseries_service_emprise.go:69-95`) ; `attachEmprise` garde sa garde
  (l. 65-67), sa section de durée et sa signature, et appelle le helper (`WithMaps: true`). Tests
  `timeseries_service_emprise_test.go` verts SANS modification ; mutation : `WithMaps` ignoré →
  rouge (`TestTimeseries…` grille), journal `emprise` sans attribut `page` → aucune assertion (noter).
- [x] S1.2 `service/solo_lives_block.go` (NEUF) : `viesQuery{Page, Player, PlayerXUID, Repo, Radar,
  MatchIDs}` et `lireViesPresOuSeul` = corps de `attachLives` (`timeseries_service_lives.go:47-75`),
  journaux `vies_*` + `page` ; `attachLives` l'appelle. Tests `timeseries_service_lives_test.go` verts
  sans modification ; mutation : portée résolue sans `PorteesDuRadarParMatch` → rouge.
- [x] S1.3 `service/squadagg/weapon_tools.go` (NEUF) : `WeaponToolInputs` (champs exportés de
  `squadToolInputs`, `teammates_squad_weapon_tools.go:41-52`), `BuildWeaponTools`, `PlayersAboveSheet`
  et les aides privées (l. 54-231, 274-288) déplacés ; `teammates` appelle
  `squadagg.BuildWeaponTools` / `squadagg.PlayersAboveSheet` ; tests purs du builder déplacés dans
  `squadagg/weapon_tools_test.go` (les cas de `teammates_squad_weapon_tools_test.go` qui exercent la
  section et les lectures restent côté `teammates`). Mutation : « Non attribué » non trié en dernier →
  rouge.
- [x] S1.4 `analysis/home_canonical_recent.go:240` : `buildScoreLabelCanonical` exporté
  (`ScoreLabelCanonical`), appelant de l'accueil migré (l. 91) ; tests de l'accueil verts sans
  modification.
- Gate : gate Go (sans contrat : aucun type public de contrat ne change) ; preuves : `Grep
  "buildSquadWeaponTools|squadToolInputs" apps/go-api/internal` → 0 ; `Grep "LoadLivesNearTeammate\("
  apps/go-api/internal --glob !*_test.go` → `solo_lives_block.go` + repo + port seulement.

Journal S1 (2026-10-06, exécuteur, `feat/sessions-emprise` sur `262e36b2e`) — refactorisations sans changement de comportement :
- **S1.1** `service/solo_emprise_block.go` : `soloEmpriseQuery` + `buildSoloEmpriseBlock` (corps de l'ancien `attachEmprise` : trois lectures `squadagg.EmpriseLecteur`, joueur seul, `WithoutTimeScale`, `Build`, `BuildEquipment`, `BuildMaps` sous `WithMaps`) ; `attachEmprise` garde sa garde, sa section de durée `emprise` et sa signature, et l'appelle (`Page: "timeseries"`, `WithMaps: true`). `timeseries_service_emprise_test.go` vert SANS modification.
- **S1.2** `service/solo_lives_block.go` : `viesQuery` + `lireViesPresOuSeul` (corps de l'ancien `attachLives`, section `lives` après la garde du repo comme avant) ; journaux renommés `vies_*` avec l'attribut `page` (aucun test ne les assertait) ; `attachLives` réduit à sa garde et à l'appel. `timeseries_service_lives_test.go` vert SANS modification.
- **S1.3** `service/squadagg/weapon_tools.go` : `WeaponToolInputs` (champs exportés), `BuildWeaponTools`, `PlayersAboveSheet` et leurs aides privées, déplacés tels quels de `teammates` ; `teammates_squad_weapon_tools.go` (331 → 102 L) garde ses lectures et appelle le builder. Tests purs déplacés dans `squadagg/weapon_tools_test.go` (sept cas, renommés `TestBuildWeaponTools_*` ; seuls les appels et les noms de champs changent ; l'aide `wantKills` y compte deux joueurs au lieu de trois — le troisième valait 0 dans ces cas, assertion identique) ; le témoin de bout en bout `TestSquadWeaponTools_Soiree2209` reste côté `teammates`, inchangé.
- **S1.4** `analysis.ScoreLabelCanonical` (ex-`buildScoreLabelCanonical`, commentaire de contrat réécrit), appelant de l'accueil et tests renommés (`TestScoreLabelCanonical_*`), assertions inchangées.
- **Mutations** (script `mut_s1.ps1` du scratchpad, restauration garantie puis vérifiée par grep) : grille par carte jamais calculée → ROUGE (`TestAttachEmprise_FenetreUnJoueurCartesEtEquipement`) ; lectures partagées non transmises → ROUGE (`TestAttachMigratedSections_UneLectureDuResumeDUsage`) ; portée du radar non résolue → ROUGE (`TestAttachLives_LectureBorneeEtPorteeCourante`) ; « Non attribué » non trié en dernier → ROUGE (`TestBuildWeaponTools_ReliquatEnDernier`) ; manches ignorées → ROUGE (`TestScoreLabelCanonical_VarianteADecideeEnManches`).
- **Gate** (CGO, une commande `go` à la fois, avant-plan) : `go build ./...` 0 ; `gofmt -l internal` muet ; `go vet` de `service`, `squadagg`, `teammates`, `analysis` 0 ; `go test -count=1` des paquets touchés + `archlint` : 5 ok ; module en lots couvrant tout `go list ./...` : cmd + analysis + api + archlint + domain + port 66 ok, contracttest + domain + port 7 ok, games 39 ok, platform + service 28 ok, sync + persist + migration 13 ok, reste de internal + pkg + scripts + tests 48 ok (57 paquets dont sans test) — 0 FAIL ; `make go-api-lint` (cache isolé) 0 issues ; `openapi-gen -check` à jour (aucun type de contrat touché). Preuves : `buildSquadWeaponTools|squadToolInputs|playersAboveSheet` → 0 ; `LoadLivesNearTeammate(` hors tests → `solo_lives_block.go`, repo DuckDB, port.
- Seuils : fichiers neufs ≤ 240 L ; plus longue fonction neuve `buildSoloEmpriseBlock` ~30 L, un paramètre de requête ; fichiers touchés en baisse (`timeseries_service_emprise.go` 126 → 105, `timeseries_service_lives.go` 77 → 42, `teammates_squad_weapon_tools.go` 331 → 102).

### S2 — Go : les blocs neufs de la page Sessions (contrat additif) · lourd

Périmètre : `service/session_page_{emprise,lives,objectif,tools,match_scores,blocks}.go` (NEUFS),
`session_page_service.go` (embarquement d'une struct de dépendances, appel des nouvelles passes),
`session_page_coordination.go`, `session_page_usage.go` (lectures partagées, vit jusqu'à S5),
`session_page_frag_distribution.go`, `domain/session_page.go`, `domain/session_compare.go`,
`api/wire/registry_pages_sessions.go` (NEUF) + factory `SessionPage`, contrat.

- [x] S2.1 `session_page_blocks.go` : `sessionBlocksDeps{xuid, usageRepo, usageFriends, repoRoot,
  empriseRepo, vehicleRepo, emblemLoader, livesRepo, radarRange, formesObjectives, roundsDecide}`
  embarquée par UNE ligne dans `SessionPageService` à la place des champs `sessionUsageRepo`,
  `usageXUID`, `usageFriends`, `repoRoot` (`session_page_service.go:65-71,78-81` : taille du fichier
  en baisse) ; `With*` dans le même fichier : `WithSessionEmprise(repo, xuid)` (inconditionnel),
  `WithSessionVehicleUsage`, `WithSessionEmblemLoader`, `WithSessionLives`, `WithSessionRadarRange`,
  `WithSessionObjectives`, `WithRoundsDecide` ; `WithSessionUsage` garde sa signature jusqu'à S5.
- [x] S2.2 Lecture partagée D8 : `lireUsageDeSession(ctx, ids) *squadagg.LecturesUsage` (section
  `usage_summary`) ; `buildSessionUsage` (`session_page_usage.go:111-113`) consomme ces lectures au
  lieu de relire ; effectif de camp de la coordination depuis `BuildTeamContext` sur ces participants
  (D9) ; `lecteursDeCoordination` lit `s.blocks.xuid` (`session_page_coordination.go:95`). Test :
  une lecture du résumé d'usage par session pour usage + Emprise + formes + coordination (compteur de
  mock) ; mutation : seconde lecture → rouge.
- [x] S2.3 `session_page_emprise.go` : `attachSessionEmprise(ctx, resp, sc, lus)` — courant et
  comparé, `buildSoloEmpriseBlock` (S1.1, `WithMaps: false`, `Page: "sessions"`) sur
  `timeseriesEmpriseMatches(lignes canoniques de la session, locale)` ; section de durée `emprise`.
  Tests (mocks de port, `session_page_emprise_test.go`) écrits ROUGES d'abord : un seul joueur
  (V1, même en contexte escouade : coéquipier suivi présent dans les participants → absent de
  `players`), pas d'habitude ni de placement, `maps` nul, équipement présent ; session comparée servie
  seulement drawer ouvert ; Halo 5 (repo d'usage nil → `film_unavailable = film_unsupported`, feuille
  seule) ; `ErrCapabilityNotSupported` de la feuille → `sheet_unsupported` ; lecture en échec →
  `*_load_failed`, jamais d'erreur de page ; session vide → nil. Mutations : coéquipiers sélectionnés,
  bloc comparé posé drawer fermé, Maps calculées → rouges.
- [x] S2.4 `session_page_lives.go` : `attachSessionLives` — courant et comparé par
  `lireViesPresOuSeul` (S1.2, `Page: "sessions"`). Tests : bornage aux matchs de CHAQUE session et au
  xuid (mock enregistreur), capability absente → nil, échec → nil + ErrorContext. Mutation : lecture
  des matchs des deux sessions dans un seul appel → rouge.
- [x] S2.5 `session_page_objectif.go` : `attachSessionFormes` — courant et comparé,
  `squadagg.BuildSquadFormesBlock` avec `Metas = timeseriesFormesMetas(lignes canoniques de la
  session, locale)`, `Lectures` = D8, `SelectedGamertags` vide (D7) ; `attachSessionEmblem` (même
  patron que `attachEmblem`, `timeseries_service_emprise.go:99-105`). Tests : objectif présent avec
  la colonne `flag_grabs_net` sous `take` (fixture), sans repo d'objectif → bloc sans objectif, Halo 5
  → bloc indisponible ; emblème absent → champ vide. Mutation : coéquipier passé en sélection → rouge.
- [x] S2.6 `session_page_tools.go` : `sessionWeaponTools(ctx, rows, counts, xuid, gamertag)` —
  `squadagg.BuildWeaponTools` (S1.3), catégories par assertion `port.KillSourceCategoryRepository`
  sur `s.weaponKillsRepo` (absente ou `ErrCapabilityNotSupported` → nil, Debug ; autre erreur → Warn),
  appelé par `attachSessionFragDistribution` (`session_page_frag_distribution.go:31-48`) qui renseigne
  `entry.WeaponTools` sur les deux entrées. Tests ROUGES d'abord : une ligne d'arme, mêlée de la
  feuille, objet explosif du film retiré de l'arme qu'il recouvre, reliquat « Non attribué » ; sans
  catégories → reliquat. Mutation : catégories ignorées → rouge.
- [x] S2.7 `session_page_match_scores.go` : `appliquerScoresEtDominance(rows, canonRows,
  roundsDecide)` (D15) sur `resp.Matches` et `resp.CompareMatches`, appelé par une ligne de `GetPage`.
  Tests : score en manches pour une variante déclarée, en points sinon, dominance recopiée, match
  sans canonique → champs vides. Mutation : `RoundsDecide` ignoré → rouge.
- [x] S2.8 `domain/session_page.go`, `domain/session_compare.go` : champs du §3 (commentaire de contrat
  court, au présent).
- [x] S2.9 Câblage `api/wire/registry_pages_sessions.go` (NEUF) : `cablerBlocsSessions(svc, pdb)`
  appelé par UNE ligne de la factory `SessionPage` (`registry_pages.go:318-366`, taille non accrue) —
  `WithSessionEmprise(duckdb.NewSquadEmpriseRepo(pdb), pdb.XUID)` et l'emblème inconditionnels,
  `WithSessionRadarRange(r.radarRangeFor(pdb))`, `WithRoundsDecide(r.roundsDecideFor(pdb))`, vies sous
  `CapFilmKillPositions`, véhicules sous `CapFilmVehicleUsage`, objectifs des formes sous
  `CapMatchObjectiveStats` (même gate que `registry_pages.go:337-339`). Garde-rail
  `registry_pages_sessions_wiring_test.go` (NEUF, patron `registry_pages_timeseries_wiring_test.go`,
  lecteur `appelsDansFactory`) ; mutations : feuille sous condition, vies hors porte → rouges.
- [x] S2.10 Contrat régénéré, diff ADDITIF (0 retrait dans `openapi.yaml` et `generated.ts`) ;
  `contract-surface.guard.test.ts` vert sans régénérer le snapshot.
- Gate : gate Go + contrat ; web : `npm ci`, `generate-types`, `tsc -b --force` 0, vitest
  `src/lib/api` vert.

Journal S2 (2026-10-06, exécuteur, `feat/sessions-emprise`) — blocs neufs de Sessions, contrat additif. Tests écrits AVANT le code (rouge de compilation vu : symboles absents), puis code, puis mutations (dont un bouchon) :
- **S2.1** `session_page_blocks.go` : `sessionBlocksDeps` embarqué par UNE ligne dans `SessionPageService` à la place de `sessionUsageRepo` / `usageXUID` / `usageFriends` / `repoRoot` (champs gardés DANS la struct sous leur nom, promus : le bloc d'usage historique, ses tests et la garde textuelle de `pad_tiers_wiring_test.go` lisent toujours `s.sessionUsageRepo`) + `sessionXUID`, `empriseRepo`, `vehicleRepo`, `emblemLoader`, `livesRepo`, `radarRange`, `formesObjectives`, `roundsDecide` ; `WithSessionEmprise(repo, xuid)`, `WithSessionVehicleUsage`, `WithSessionEmblemLoader`, `WithSessionLives`, `WithSessionRadarRange`, `WithSessionObjectives`, `WithRoundsDecide` ; `WithSessionUsage` inchangé. Constante `pageSessions` (goconst).
- **S2.2** `attachSessionBlocks` remplace l'appel `attachSessionUsage` de `GetPage` (même ligne) : `lecturesDesSessions{courant, compare}` lues UNE fois par session (`lireUsageDeSession`, section `usage_summary`) et partagées par le bloc d'usage historique (`buildSessionUsage` ne relit plus), l'Emprise, l'objectif et `effectifsDeCamp` (D9 : `BuildTeamContext` sur les participants partagés, même définition que l'ancien bloc d'usage) ; `lecteursDeCoordination` lit `sessionXUID`. Les tests du bloc d'usage historique appellent désormais `attachSessionBlocks(…, nil)` (sites d'appel seuls, assertions inchangées) ; les deux doublures de coordination câblent `WithSessionEmprise(nil, "P")` (D9). Changement de comportement (conséquence de D9, signalé au superviseur, §8) : un titre qui nomme ses tueurs SANS résumé d'usage — Halo 5 — reçoit désormais sa coordination sur Sessions, sans parité (elle y était indisponible).
- **S2.3 / S2.4** `session_page_emprise.go` (un seul fichier pour l'Emprise et les vies, au lieu de deux : 62 L) : `buildSoloEmpriseBlock` sans grille par carte, `Page: pageSessions`, sur `timeseriesEmpriseMatches(lignesCanoniquesDe(canon, session), locale)` ; `lireViesPresOuSeul` bornée aux matchs de CHAQUE session et au joueur.
- **S2.5** `session_page_objectif.go` : `squadagg.BuildSquadFormesBlock` (requête réduite L7, `SelectedGamertags` vide), `timeseriesFormesMetas` des lignes canoniques de la session ; `attachSessionEmblem`.
- **S2.6** `session_page_tools.go` : `sessionWeaponTools` (builder `squadagg.BuildWeaponTools`, un joueur, feuille = compteurs canoniques déjà agrégés, catégories par l'interface optionnelle `port.KillSourceCategoryRepository` du lecteur d'armes : absente / non supportée → Debug, échec → Warn) ; `sessionFragDistribution` rend aussi les outils, posés sur les deux entrées.
- **S2.7** `session_page_match_scores.go` : `appliquerScoresEtDominance` (une ligne dans `GetPage`). Écart de source : la dominance est lue sur la ligne canonique (`Enrichment.DominanceFlag`), d'où `StatsMatchRow.DominanceFlag` est elle-même projetée — même valeur, une seule passe.
- **S2.8** Champs de contrat : `SessionPageResponse.{Emprise, CompareEmprise, LivesNearTeammate, CompareLivesNearTeammate, FormesRetenues, CompareFormesRetenues, PlayerEmblemURL}`, `SessionCompareEntry.WeaponTools`, `SessionDetailMatchRow.{ScoreLabel, DominanceFlag}`.
- **S2.9** `api/wire/registry_pages_sessions.go` : `cablerBlocsSessions` (feuille + joueur, emblème, portées du radar, manches inconditionnels ; vies sous `CapFilmKillPositions`, véhicules sous `CapFilmVehicleUsage`, objectifs sous `CapMatchObjectiveStats`) appelé par une ligne de `SessionPage` ; garde-rail `registry_pages_sessions_wiring_test.go` (5 tests, lecteur `appelsDansFactory`).
- **S2.10** Contrat : `openapi.yaml` +21 / −0, `generated.ts` +11 / −0 ; `check-generated-types-fresh` OK ; `contract-surface.guard.test.ts` vert SANS régénération (snapshot non modifié).
- **Mutations** (script `mut_s2.ps1`, restauration garantie puis vérifiée par grep ; 13, toutes ROUGES, rejouées après le regroupement des paramètres) : lecture partagée non transmise à l'Emprise (2 lectures au lieu d'1) ; effectif de camp perdu (Appui ≠ ancien chemin) ; joueur de la coordination tiré du bloc d'usage ; Emprise comparée calculée sur la session affichée ; grille par carte calculée ; bouchon « Emprise non posée » (trois tests rouges) ; vies des deux sessions en une lecture ; coéquipier sélectionné dans l'objectif ; catégories de source ignorées ; manches ignorées ; câblage des blocs sous condition ; feuille et joueur sous condition ; vies hors de leur porte.
- **Gate** : `go build ./...` 0 ; `gofmt -l internal` muet ; `go vet` de `service`, `api/wire`, `domain`, `analysis` 0 ; `go test -count=1` du module en lots couvrant tout `go list ./...` (cmd + contracttest + analysis + api + domain + port + archlint 67 ok, games 39 ok, platform + service 28 ok, sync + persist + migration 13 ok, reste 48 ok) — 0 FAIL, puis `service` + `api/wire` + `archlint` rejoués après la correction du lint et le regroupement des paramètres ; `make go-api-lint` 0 issue (un `goconst` corrigé en cours de gate) ; `openapi-gen -check` à jour ; web : `npm ci` (node_modules réel, 508 paquets), `generate-types`, purge `node_modules\.tmp`, `npx tsc -b --force` 0, vitest `src/lib/api` 5 fichiers / 36 tests verts. `-tags=integration` non requis (aucun paquet `platform/duckdb`, `sync`, `persist`, `migration` modifié).
- Seuils : `session_page_service.go` 894 → 886, `registry_pages.go` 620 → 619 ; fichiers neufs ≤ 152 L ; fonctions ≤ 5 paramètres (`attachSessionEmprise` / `attachSessionFormes` ramenés de 6 à 5 par `lecturesDesSessions`) ; plus longue fonction neuve ~25 L.

### S3 — Web : briques étendues (compact), Escouade et Séries temporelles inchangées · lourd

RÈGLE DU LOT : uniquement des ajouts rétro-compatibles (`compact` défaut `false`) et des
centralisations ; chaque export neuf a un lecteur dans le lot (knip 0 / 0 / 0) ; les tests de page
de l'Escouade et des Séries temporelles sont rejoués nommément SANS modification
(`SquadEmprisePage.test.tsx`, `SquadContributionsPage.test.tsx`, `SquadObjectiveSection.test.tsx`,
`SquadFragSection.test.tsx`, `TimeseriesPage.sections.test.tsx`, `TimeseriesPage.usages.test.tsx`).
BASE DEPUIS LE SECOND REBASE (`554457c31`) : `compact` s'ajoute aux versions L9 / L9.2 de
`EquipmentOutcomesCard.tsx` (repli au pixel aligné sur son segment, `partSegments`,
`repliOffsetPct`), `LivesNearTeammateCard.tsx` (repli au pixel, 3e cause d'exclusion
`excluded_unpublishable` dans l'ⓘ) et `empriseCharts.ts` (rayon réduit des points sur l'axe période
seul) ; leurs tests L9 restent verts sans modification. KNIP EST AVEUGLE SUR CE POSTE (découverte du
lot TS) : la preuve « chaque export neuf a un lecteur » se fait par grep, le ratchet local ne fait
pas foi (le step CI, oui).

- [x] S3.1 D17 : `empriseObjectName` dans `emprise.logic.ts` ; `useEmpriseModels.ts:29-36` et
  `useUsagesModels.ts:29-36` migrés ; garde-rail `squad/emprise/emprise.objectName.guard.test.ts`
  (balayage `import.meta.glob` : aucun fichier hors `emprise.logic.ts` ne combine
  `RESOURCE_VEHICLE` et `vehicleFamilyName` dans une fonction de nommage) ; auto-test sur l'ancien
  littéral ; mutation : copie réintroduite → rouge.
- [x] S3.2 `ResourceControlCard` + `PisteCampsForm` : `compact` → parts seules dans les segments,
  comptes au survol, râteliers en une ligne de ressource (sans dépliage). Tests (rouges d'abord) :
  libellé « 60 % » au lieu de « 12 », infobulle avec les comptes, aucun bouton de dépliage.
- [x] S3.3 `ProductionCard` : `compact` → parts seules (épaisse et fine), ligne d'exposition en % ;
  `YieldCard` inchangée (vérifier qu'aucun compte n'y est écrit hors rendements bruts, comme la
  maquette `makeYield`).
- [x] S3.4 `empriseCharts.ts` / `ResourceFilCard` : option `compact` du mode `match` — hauteur 170,
  graduations 0 / 50 / 100, pas d'étiquette sous la bande, étiquettes de fin seules (maquette
  `renderFil` l. 1312-1316, 1344). Tests sur l'option ECharts ; mutation : dates sous l'axe en compact
  → rouge.
- [x] S3.5 Grille compacte : `compactGridRows(grid)` dans `emprise.logic.ts` (lignes de ressource
  seulement : bonus, armes spéciales prises, frags aux armes spéciales, râteliers, véhicules ; case =
  part de mon camp ou état « sans film » / « non classé » / « rien à prendre ») ; `ResourceGridTable`
  et `ResourceMatchGridCard` : `compact` → ces lignes, case « 71 % » colorée par l'écart (même
  encre), en-tête heure + carte + V / D, gabarit `minmax(0, 1fr)` (S11). Tests : lignes et ordre,
  « ? » sans niveaux, « — » hachuré sans film, aucune largeur minimale de colonne ; mutation : ligne
  d'objet en compact → rouge.
- [x] S3.6 `MinePickupsCard` : `compact` → une barre par ressource (ma part des prises de mon camp en
  %, reste en %), bonus perdus des deux camps en % (comptes au survol) ; modèle `buildMineCompact` dans
  `usages.logic.ts` (depuis `resources[].taken.us`, `objects[].squad[0].taken`, `outcomes`). Tests
  rouges d'abord (témoin MESURES §3 s2209 : bonus 3 / 12 = 25 %, armes spéciales 7 / 13 = 53,8 %,
  râteliers 3 / 24 = 12,5 %, bonus perdus 16,7 % / 25 %) ; mutation : part sur le total des deux
  camps → rouge.
- [x] S3.7 `EquipmentOutcomesCard` : `compact` → segments en %, barre fine gardée, sous-libellé
  « n objets », ligne du reste « p % servis ». Tests ; mutation : sous-libellé avec les prises en
  compact → rouge.
- [x] S3.8 `LivesNearTeammateCard` : `compact` → parts seules dans la barre épaisse et sur la ligne
  des frags (D16). Tests ; mutation : comptes en compact → rouge.
- [x] S3.9 `ObjectiveBalanceCard` : `compact` → une barre par RÔLE et par famille (somme des actions
  du rôle, Tenir en durée) plus la ligne des prises nettes, % seuls ; modèle `buildBalanceByRole` dans
  `objectif.logic.ts`. Tests (témoin MESURES §3 s0709 : Bases prendre 93-109 = 46 %, défendre 30-60 =
  33,3 %, tenir 53,6 % ; Drapeau prises nettes 29-41 = 41,4 %) ; mutation : Tenir sommé avec les
  comptes → rouge.
- [x] S3.10 `ObjectiveSoloSheetCard` : `compact` → la valeur affichée devient la part du camp (« 32 % »,
  « — » si le camp n'a rien), pied en parts (témoin s2209 : Prendre 31,8 %, Défendre 30,3 %, Tenir
  47,2 %). Tests ; mutation : compte affiché en compact → rouge.
- [x] S3.11 `SquadFragBreakdownCard` : `compact` → parts dans les segments, total en sous-libellé du
  nom, pas de total au bout ; `SquadWeaponKillsChart` / `buildSquadToolRows` : option `top` (6) et
  valeur en part des frags du joueur, « Non attribué » exclu en compact (maquette `makeTools`
  l. 942). Tests ; mutation : « Non attribué » gardé en compact → rouge.
- Gate : gate web (aucun Go) ; tests de page Escouade / Séries temporelles nommés ci-dessus rejoués,
  `git diff` vide sur ces fichiers de test.

Journal S3 (2026-10-06, exécuteur, `feat/sessions-emprise` sur `554457c31`) — briques étendues d'une vue compacte, Escouade et Séries temporelles inchangées. Chaque test écrit et vu ROUGE avant son code (symbole absent ou assertion), puis vert, puis mutation :
- **Patron retenu** pour les textes propres à la vue compacte : chaque carte qui en a besoin reçoit un objet `compact` qui PORTE ses formateurs (`ProductionCard` : `exposureLine` ; `MinePickupsCard` : `resourceSub` ; `EquipmentOutcomesCard` : `sub`, `unmeasured`, `restUsed` ; `LivesNearTeammateCard` : `killsLine`, `killsLineAlone` ; `SquadFragBreakdownCard` : `totalSub`, `pctFmt`) ; les autres prennent `compact?: boolean`. Aucun type de texte existant ne change (`empriseStrings.ts` à 500 L, textes des Séries temporelles intacts) ; la page Sessions fournira ces formateurs (S4.3). Types d'options non exportés tant qu'ils n'ont pas de lecteur hors de leur fichier.
- **S3.1** `squad/emprise/objectName.ts` (NEUF : `empriseObjectName`) — écart au plan : un fichier à part plutôt que `emprise.logic.ts`, qui se déclare sans chaîne de langue alors que le nommage lit l'i18n d'usage ; les deux copies (`useEmpriseModels.ts`, `useUsagesModels.ts`) migrées ; garde-rail `objectName.guard.test.ts` (empreinte : un fichier de production qui appelle `equipmentFamilyLabel(` ET `vehicleFamilyName(`, auto-test sur l'ancien littéral) vu rouge sur les deux copies avant la migration.
- **S3.2** `PisteCampsForm` : `pctOnly` (segment et repli en part seule, compte en infobulle) ; `ResourceControlCard` : `compact` (part entière). La carte de l'Escouade ne dépliait déjà aucun râtelier (une ligne par ressource) : rien d'autre à faire pour « râteliers en une ligne ».
- **S3.3** `ProductionCard` : `compact` (parts entières, ligne d'exposition en parts). `YieldCard` : identique dans les deux vues (maquette `makeYield`) — vérifié, aucun changement.
- **S3.4** `empriseCharts.ts` : `FilAxe` match `compact` (rien sous l'axe, graduations tous les 50 %, pied réduit à la bande, points 1,6 + √n ; bande et encoche gardées) via `isCompact` / `radiusFor` ; `ResourceFilCard` : `compact` (170 px). `ResourceFilCard.test.tsx` (NEUF) double `ChartCard`.
- **S3.5** `ResourceGridTable` : `compact` (lignes de ressource seules, râteliers sans bouton, « 71 % », « — » hachuré sans film / véhicules non mesurés, « ? » sans niveaux et camp inconnu, `86px repeat(n, minmax(0, 1fr))` sans largeur minimale ni défilement) ; `ResourceMatchGridCard` : `compact` (en-tête heure, carte tronquée, initiale du résultat dans sa couleur d'issue — l'initiale du libellé du titre). Écart au plan : pas de `compactGridRows` exporté — retirer les lignes d'objet est un choix de rendu de la table, aucune règle métier.
- **S3.6** `usages.logic.ts` : `mineByResource` (le `buildMineCompact` du plan, renommé ; sommes moi / camp des objets de chaque ressource, dans l'ordre du bilan) ; `MinePickupsCard` : `compact` (une barre par ressource, parts entières — la seconde = 100 − la première arrondie, maquette —, bonus perdus en part, compte au survol). Témoin fixture : bonus 31 / 111 → 28 % / 72 %, armes spéciales 30 / 229 → 13 % / 87 %, pertes 11 % / 8 %.
- **S3.7** `EquipmentOutcomesCard` (version L9.2, repli aligné conservé) : `compact` (parts entières dans la barre et au repli, sous-libellé « n objets », « Non mesuré » court, ligne du reste « p % servis »). Mur : 52 · 0 · 32 → 62 % / 38 %, reste 48 % servis. Prop interne renommée `textOf` (`valueOf` heurtait `Object.prototype` au typage).
- **S3.8** `LivesNearTeammateCard` (version L9) : `compact` (parts entières, ligne des frags en parts) — 84 % / 16 %, « frags : 83 % · 0,8 par vie … 0,9 par vie ».
- **S3.9** `objectif.logic.ts` : `BalanceLine.optional` (posé par `buildObjectiveBalance`) et `buildBalanceByRole` ; `ObjectiveBalanceCard` : `compact`. Témoin MESURES §3 retrouvé sur la fixture du 07/09 : Bases prendre 93-109 (46 %), défendre 30-60 (33,3 %), tenir 53,6 % ; Drapeau prises nettes 29-41 (41,4 %).
- **S3.10** `SoloObjectiveSheet.campRoleTotals` ; `ObjectiveSoloSheetCard` : `compact` (part du camp au bout de chaque action et au pied, « — » quand le camp n'a rien fait) ; le pied porte `data-testid="objective-solo-foot"` (enveloppe `contents`). Témoin MESURES §3 retrouvé sur la fixture du 22/09 : Prendre 7 / 22, Défendre 10 / 33.
- **S3.11** `buildSquadToolRows` : option `top` (les premiers outils, « Non attribué » exclu) ; `buildSquadWeaponKillsOption` : `shareTotals` (dénominateur fourni) et `minLabelShare` ; `SquadFragBreakdownCard` : `compact` (parts entières, total en sous-libellé, pas de total au bout).
- **Mutations** (script `mut_s3.ps1`, restauration garantie puis vérifiée) : 17, toutes ROUGES au final — nommage recopié ; parts seules ignorées ; ligne d'exposition compacte ignorée ; étiquettes sous l'axe en compact ; rayon de la maquette en compact ; lignes d'objet en compact (VERTE au premier passage : l'objet du test était dans la section des râteliers, déjà repliée — test renforcé d'un objet de bonus, ROUGE ensuite) ; comptes dans la case ; libellé entier dans l'en-tête ; camp = moi (mes prises) ; pertes en comptes ; sous-libellé complet ; comptes des vies ; colonnes facultatives sommées ; camp = moi (fiche) ; reliquat gardé ; dénominateur ignoré ; comptes dans la répartition.
- **Gate** : purge `node_modules\.tmp` ; `npx tsc -b --force` 0 ; `npm run lint` 0 erreur (26 avertissements, le compte d'avant le lot) ; `npx vitest run --pool=forks` complet : 844 fichiers / 8 904 tests verts, 23 ignorés (160 s) — au premier passage, `useCopyToClipboard.guard.test.ts` a dépassé son délai de 5 s sous la charge de la suite (vert seul en 2,6 s, vert au second passage complet ; fichier non touché, §8) ; `node tools/knip-ratchet.mjs` 0 / 0 / 0 (aveugle sur ce poste : exports neufs vérifiés par grep — `mineByResource`, `MineResource`, `buildBalanceByRole`, `empriseObjectName` ont leur lecteur de production) ; couleurs 0 ; imports croisés 7 ≤ 7 ; `npx lefthook run pre-push` (PATH complet) sortie 0. Tests de page `SquadEmprisePage`, `SquadContributionsPage`, `SquadObjectiveSection`, `SquadFragSection`, `TimeseriesPage.sections`, `TimeseriesPage.usages` : verts, NON modifiés (absents du diff).
- Seuils : plus gros fichiers touchés `empriseCharts.ts` 451 L, `objectif.logic.ts` 422 L, `PisteCampsForm.tsx` 360 L (tous < 500) ; aucune fonction au-delà des seuils (lint 0 erreur).

### S4 — Web : la page Sessions reconstruite, suppressions web · lourd

Périmètre : `features/session-detail/*`, `tools/lint-cross-feature-imports.mjs` (D2),
`lib/i18n/manifests/session.toml` (+ manifeste généré), `lib/api/types.ts` (alias). Aucune requête
neuve, aucune clé de requête neuve (`lib/query/keys.ts` non touché) : tout arrive avec
`useSessionDetailPage`.

- [x] S4.1 `_sections.ts` (D11) : clés par carte, sous-groupes et leur clé de manifeste, paires de la
  pleine page, `groupSessionSections` à deux niveaux, `sessionSectionKeys` sur un objet de présence par
  carte ; `sessionRowKeys(data)` (calcul de `rowKeys` sorti de `SessionDetailPage.tsx:261-276`, fichier
  en baisse). Tests ROUGES d'abord (`_sections.test.ts`) : ordre, intertitre de sous-groupe sur la
  première clé présente, sous-groupe absent sans intertitre, union des deux colonnes, paires.
  Mutations : intertitre posé sur une clé absente, ordre de l'union pris sur la gauche seule → rouges.
- [x] S4.2 `sessionEmprise.logic.ts` (NEUF) : index des matchs depuis `SessionDetailMatchRow`
  (heure, `map_name`, `mode_ui`, résultat `outcomeCodeToValue`, `score_label`, `dominance_flag`),
  couverture, modèles par carte (composition des builders existants : `buildControlRows`,
  `buildResourceFil`, `buildMatchGrid`, `buildMinePickups`, `buildProductionRows`, `buildYieldRows`,
  `buildVehicleCoverage`, `buildLivesModel`, `buildEquipmentRows`, `objectiveMatches`,
  `buildObjectiveBalance`, `buildSoloObjectiveSheet`), et LE prédicat de présence par carte
  (`sessionCardsPresence(column)`) lu par `_sections.ts`, `SessionColumnBody` et chaque carte. Tests
  ROUGES d'abord, une mutation par règle (index : dominance ignorée ; présence : carte objectif sans
  match à objectif, Halo 5 sans film → seules A, B, B' et la barre épaisse des armes spéciales de G si
  la feuille la porte).
- [x] S4.3 `sessionEmpriseText.ts` (D12) : surcharges FR / EN, deux jeux par carte (pleine page /
  comparaison) ; test : titres, ⓘ et légendes FR copiés de la maquette pour chaque `make*` et chaque
  valeur de `cp`, aucun « Notre camp » / « Our side » / « périmètre » / « scope ».
- [x] S4.4 `SessionColumnBody.tsx` : props `usage` → `blocks: SessionColumnBlocks` (emprise, vies,
  formes, emblème, lignes de match, coordination, portée) ; montage des clés A-L et B' ; paires de la
  pleine page ; intertitres de sous-groupe et couverture S14 ; `compact` transmis à chaque carte avec
  le jeu de textes de sa vue. `SessionDetailPage.tsx` : passe les blocs courants et comparés (taille
  non accrue).
- [x] S4.5 `SessionCoordinationSection.tsx` : carte Riposte retirée (§4.C), « Appui reçu » seule,
  demi-largeur en pleine page, empilée en compact ; tests adaptés.
- [x] S4.6 Cartes A et B : `SessionFragBarCard.tsx` et `SessionToolsCard.tsx` (NEUFS, minces : cadre,
  titre, ⓘ, légende) sur `SquadFragBreakdownCard` / `SquadWeaponKillsChart` (S3.11), un joueur
  (couleur `squad-player-1`), libellés de nature par les manifestes `frags` (patron
  `SquadFragSection.tsx:61-74`). B' : `WeaponAccuracyChart` monté tel quel (D14).
- [x] S4.7 Cartes C-L : montage des briques (S3) avec les modèles S4.2 et les textes S4.3 ; D en
  axe `match` avec l'index des matchs de la session (encoche de dominance, bande de résultats) ; K avec
  `player_emblem_url` et le gamertag du joueur.
- [x] S4.8 Tests de page : `SessionColumnBody.test.tsx` réécrit (ordre des cartes, intertitres,
  couverture, retrait par carte, Halo 5, anglais) ; `SessionCompareRows.test.tsx` étendu (une rangée
  partagée par carte, même liste des deux côtés, marqueur du côté sans la carte — session solo sans
  objectif face à une session à objectif, témoin MESURES §0 —, intertitres dans la même rangée des
  deux côtés, cartes en variante compacte des DEUX côtés, aucun intertitre ni rangée en pleine page) ;
  `sessionShrink.guard.test.ts` étendu aux paires de cartes (une colonne en `compact`) ; fixture
  `sessionEmprise.fixtures.ts` tirée des MESURES (s2209, s0709, solo).
- [x] S4.9 D2 : paire `session-detail=>timeseries` ajoutée à `ALLOWED_CROSS_IMPORTS` avec son
  commentaire ; ratchet ≤ 7 ; aucune dérogation morte.
- [x] S4.10 Rejouer CHAQUE preuve grep de §4.A, B, C, G avant de supprimer ; écart → §8, arrêt propre
  si un lecteur inattendu existe.
- [x] S4.11 Web §4.A : fichiers et tests supprimés, `sessionSectionVisibility.ts` réduit ou supprimé.
- [x] S4.12 Web §4.B : `SessionFragCard.tsx` supprimé.
- [x] S4.13 Web §4.C : riposte de `coordinationModel.ts` et `coordinationI18n.ts`.
- [x] S4.14 Web §4.G et §4.H (web) : `_shared/usage/*` orphelins, exports et clés devenus privés ou
  supprimés, manifestes régénérés, gardes d'usage adaptées.
- [x] S4.15 Garde-rail D7 : `features/session-detail/formesFields.guard.test.ts` — aucun fichier de
  `session-detail` ne lit un champ de `formes_retenues` hors de la liste de L7 (balayage des accès
  `.lobby`, `.weapons`, `.weapon_pads`, `.pad_named`, `.pad_unnamed`, `.duration_seconds`,
  `.team_size`, `.lobby_size`, `.measured` sur les objets de formes) ; auto-test ; mutation → rouge.
- [x] S4.16 Ratchets : knip 0 / 0 / 0, imports croisés ≤ 7 ; si un plafond baisse, l'abaisser.
- Gate : gate web ; preuves §4.A, B, C, G rejouées → 0 (côté web).

Journal S4 (2026-10-06, exécuteur, `feat/sessions-emprise` sur `f0a6c1b3c`) — la page Sessions reconstruite aux formes de l'Emprise, suppressions web :
- **S4.1** `_sections.ts` : une clé par carte (`frag_bar`, `tools`, `weapon_accuracy`, `control`, `fil`, `grid`, `mine`, `production`, `yield`, `lives`, `objective_balance`, `objective_sheet`, `equipment`), sous-groupes (`resources`, `prendre`, `lives`, `objectif`, `equipment`, clés de manifeste `session.detail.subsection_*` FR / EN, manifeste régénéré), paires A|B, C|D, G|H (`pairSessionKeys`), `groupSessionSections` à deux niveaux (`subruns`), `sessionRowOpenings` (titre de groupe et intertitre de sous-groupe dans la rangée de leur première clé présente), `sessionSectionKeys` sur `SessionSectionPresence` (présence des cartes + coordination + portée, champs obligatoires : une carte neuve sans présence ne compile pas), `sessionRowKeys(data)`. `_sections.test.ts` (9 cas) vu ROUGE (symboles absents) avant le code. Écart d'ordre : `sessionRowKeys` lit la présence de S4.2 — posé avec S4.2. Le type de présence vit dans `sessionEmprise.logic.ts` (aucun cycle d'import : `_sections` l'importe, pas l'inverse).
- **S4.2** `sessionEmprise.logic.ts` : `SessionColumnBlocks` (la session, ses matchs, son Emprise, ses vies, ses formes, l'emblème, la coordination, la portée — `entry` et `matches` y entrent aussi), `sessionColumnBlocks(data, side)`, `sessionMatchIndex` (score et dominance, D15), `buildSessionEmpriseModels`, `sessionCardsPresence` / `sessionColumnPresence` (le prédicat unique : page, colonne, cartes), `sessionPlayerName`. ÉCART TDD : le code de ce fichier a été écrit avant son test ; compensé par cinq mutations, toutes ROUGES (dominance ignorée, objectif sans match à objectif, contrôle sans film sur Halo 5, précision par arme ignorée, « Non attribué » seul compté comme une carte). Présence de B : au moins un outil NOMMÉ (le « Non attribué » seul ne se dessine pas en vue compacte : la carte ne s'ouvre nulle part, un seul prédicat pour les deux vues).
- **S4.3** `sessionEmpriseText.ts` : `SESSION_CARD_TEXT[locale].{full, compact, compactCards}` — surcharges de `EMPRISE_TEXT_SOLO` (contrôle, fil « de la session » et cumul « de la soirée », production, grille « match par match » et sa légende compacte), `OBJECTIF_TEXT_SOLO` (rapport de force), `USAGES_TEXT` (Mes prises, Équipement, Ma part — part entière en compact), `getSquadText` (A, B) ; formateurs compacts typés par les props des cartes (`ComponentProps`), aucun type d'option exporté. `sessionEmpriseText.test.ts` (9 cas) : chaque ⓘ FR mot pour mot de la maquette, pleine page et `cp`, légendes, formateurs, vocabulaire (balayage des seules parties montées par Sessions : les dictionnaires de l'Escouade portent toute leur page).
- **S4.4 / S4.7** `SessionColumnBody.tsx` réécrit (prop `blocks`) ; montage des cartes dans `useSessionEmpriseCards.tsx` (NEUF, hors plan : la colonne reste à 248 L ; une table de rendus par clé, appelée seulement pour une carte présente) ; pleine page : `DetailSection` par groupe, intertitre `h4` par sous-groupe (`data-session-subgroup`), couverture « n matchs filmés sur N · … » sous « Ressources de la soirée » seulement, paires par `pairGridClass` ; comparaison : une rangée par clé, titre de groupe puis intertitre dans la rangée d'ouverture, marqueur existant du côté sans la carte. `SessionDetailPage.tsx` : blocs mémoïsés par colonne, `rowKeys = sessionRowKeys(data)` ; 551 → 525 L.
- **S4.5** `SessionCoordinationSection.tsx` : « Appui reçu » seule, rangée par `pairGridClass(compact)` (demi-largeur en pleine page, toute la colonne en compact ; écart : gouttière `gap-6` du gabarit au lieu de `gap-3`). `SessionCoordination.test.tsx` réécrit sur l'Appui (fixture typée par cast, sans riposte : S5 n'aura rien à y changer).
- **S4.6** `SessionFragBarCard.tsx` (A, une barre, `getSquadPlayerColors`), `SessionToolsCard.tsx` (B : pleine page tous les outils au compte ; compact six premiers sans « Non attribué », part de TOUS mes frags — dénominateur = somme des lignes du serveur, reliquat compris —, `minLabelShare = 0`) ; B' : `WeaponAccuracyChart` (hauteur 320) monté tel quel. `SessionToolsCard.test.tsx` (NEUF, graphe doublé : 8 lignes au compte ; 6 lignes, part, dénominateur 65 — témoin s2209).
- **S4.8** `SessionColumnBody.test.tsx` réécrit (11 cas : titres, intertitres et couverture, ordre des cartes, paires, retrait par carte, solo sans objectif, aucune carte, Halo 5, compact — sous-libellé « 65 frags » et légende compacte de la grille —, 07/09 compact par rôle 46 % / 54 % et pleine page par action, anglais) ; `SessionCompareRows.test.tsx` réécrit sur s2209 face au solo (6 cas : même liste des deux côtés et rangées A-L, marqueur à droite pour J, K, C, F, intertitres dans la même rangée des deux côtés et sans couverture, compact des deux côtés « 65 frags » / « 72 frags », Appui sans Riposte des deux côtés, aucune rangée en pleine page) ; `sessionShrink.guard.test.ts` + 1 cas (paires et Appui par `pairGridClass(compact)`) ; `sessionEmprise.fixtures.ts` (s2209 sur `EMPRISE_2209` / `block2209`, s0709 sur `block0709`, solo du relevé §1-§3 ; cartes du solo anonymisées : `lint-no-hardcoded-fields` refusait leurs noms).
- **S4.9** `session-detail=>timeseries` déclarée avec son commentaire ; 7 ≤ 7 ; les trois paires `session-detail=>*` ont un lecteur.
- **S4.10** Preuves rejouées avant suppression : §4.A (lecteurs de `SessionUsageSection` et compagnie : les seuls fichiers supprimés + commentaires), §4.B (`SessionFragCard` : commentaires seuls ; `FragWeaponBreakdown`, `buildFragDetailBreakdown`, `FragSunburst` gardent des lecteurs), §4.C (riposte : `coordinationModel.ts`, `coordinationI18n.ts` seuls ; `squadRiposte*` lit d'autres symboles), §4.G (graphe d'import de `_shared/usage` relevé). Aucun lecteur inattendu.
- **S4.11 / S4.12** supprimés : `SessionUsageSection.tsx` (+ `.gate.test.tsx`), `SessionUsageEquipmentCards.tsx`, `SessionPadControlCards.tsx`, `SessionUsageShared.tsx`, `SessionUsageFlagGrabsNet.test.tsx`, `sessionSectionVisibility.ts` (entier : tous ses lecteurs sortent), `SessionFragCard.tsx`.
- **S4.13** `coordinationModel.ts` : `buildRiposteGaugeRows`, `RIPOSTE_BAND`, `buildRiposteBand`, `formatDelaiMedian`, `fenetreSeconds` ; `coordinationI18n.ts` : `cardRiposte`, `gaugeCovered`, `gaugeIRiposte`, `bandRiposte`, `delaiMedian`, `delaiFmt`, `infoRiposte1-3` FR et EN ; en-têtes corrigés.
- **S4.14** `_shared/usage` : supprimés `UsageLobbyTrack.tsx`, `usageLobbyTrackModel.ts` (+ test), `usageGrids.ts` (+ test), `usageObjectives.ts`, `usageParity.ts` (+ test), `usagePadTiersModel.ts` (+ test — §4.F le plaçait en S5 : mort dès S4, il part ici), `UsageHatchLegend.tsx`, `usageGaugeModel.test.ts`, `usageRegularityBandModel.test.ts`, `usageAvailability.test.ts`. Réduits à ce que lisent encore « Appui reçu », les cartes d'objectif de l'Escouade et le nommage des objets : `UsageForms.tsx` 399 → 160 L (colonnes nommées par l'appelant ; plus de repli de lignes, de pile d'issues, de repères de taux, de hachure « lobby », de ligne de total ni d'indice : leur seul producteur, `buildGaugeRow`, n'avait plus de lecteur), `usageGaugeModel.ts` (types seuls), `usageRegularityBandModel.ts` (type seul), `usageAvailability.ts` (causes, titres, phrases), `usageFormat.ts` (`formatUsagePct`), `usageInks.ts`, `usageMetricKinds.ts` (`roleToken`), `usageI18n.ts` 518 → 124 L (états vides, légende de bande, familles d'équipement ; `roleLabel`, `familyLabel`, `deployedFamilyLabel` sans lecteur). Tests réécrits : `UsageForms.test.tsx`, `UsageA2Ajustements.test.tsx` (états vides seuls), `usageFormat.test.ts`, `usageMetricKinds.test.ts` ; garde `noLocalUsageCopies.guard.test.ts` adaptée (motifs des définitions qui restent), `usageEmptyStateCanonical.guard.test.ts` inchangée. Manifestes : aucune clé `session.*` / `frags.*` n'a perdu son dernier lecteur (relevé sur les clés citées par les fichiers supprimés ou modifiés). Commentaires devenus faux corrigés (règle 17) : `_chartSections.tsx`, `sessionShrink.guard.test.ts`, `components/charts/{FragWeaponBreakdown.tsx, StackedTrack.tsx, README.md}`, `timeseries/TimeseriesPage.summary.tsx` (commentaire seul, fichier du lot TS hors liste §5.1 : rendu inchangé).
- **S4.15** `formesFields.guard.test.ts` : balayage des sources de la page (hors tests et fixtures), auto-test sur cinq chaînes (trois lectures interdites, deux permises).
- **Mutations** (script `mut_s4.ps1`, restauration garantie puis vérifiée par grep) : 22, toutes ROUGES — intertitre sur la dernière clé ; union ordonnée sur la gauche ; paire G|H perdue ; dominance ignorée ; objectif sans match à objectif ; contrôle sans film ; précision ignorée ; « Non attribué » seul ; ⓘ compacte du contrôle = pleine page ; jeu de textes pleine page en compact ; formateurs compacts non transmis ; `compact` non transmis au rapport de force ; intertitre de comparaison absent ; couverture perdue ; colonne comparée en pleine page ; Appui hors gabarit ; paire écrite à la main ; garde D7 (champ retiré lu) ; garde de source unique (copie de `roleToken`) ; trait de jauge sans repère ; dénominateur des outils compacts ; seuil des parts par défaut.
- **Gate** : purge `node_modules\.tmp` ; `npx tsc -b --force` 0 ; `npm run lint` 0 erreur (26 avertissements, le compte d'avant) ; `npx vitest run --pool=forks` complet : 840 fichiers / 8 871 tests verts, 23 ignorés (151 s) ; `node tools/knip-ratchet.mjs` 0 / 0 / 0 (aveugle : exports neufs vérifiés par grep — les quatre types internes de `_sections.ts` et `SessionEmpriseCards` désexportés, `session0709` lu par un test ; restent sans lecteur deux types exportés PRÉ-EXISTANTS hors périmètre, §8) ; couleurs 0 ; imports croisés 7 ≤ 7 ; `lint-no-hardcoded-fields` 0 (après anonymisation des cartes de la fixture) ; `npx lefthook run pre-push` (PATH complet) sortie 0. Preuves §4.A, B, C, G rejouées après suppression : 0 occurrence dans `apps/web/src` ; lecteurs web de la riposte du bloc `coordination` : `timeseries/TimeseriesCoordinationSection.test.tsx` seul (S5).
- Seuils : plus gros fichiers touchés `SessionDetailPage.tsx` 525 L (gelé, en baisse), `_sections.ts` 281 L, `sessionEmpriseText.ts` 281 L, `SessionColumnBody.tsx` 248 L ; fonctions ≤ 80 L (`useSessionEmpriseCards` découpé : rendus par clé dans `cardRenderers`), ≤ 5 paramètres (`useSessionEmpriseCards` : 4).

### S5 — Go : suppressions (bloc d'usage entier) et riposte, contrat · lourd

Le web ne lit plus `usage`, `compare_usage` ni la riposte depuis S4. Depuis le rebase sur
`262e36b2e`, §4.F (anciens orphelins d'intersection) est dans ce lot.

- [x] S5.1 Rejouer les preuves §4.D, §4.E et §4.F (producteurs et lecteurs Go et web).
- [x] S5.2 §4.E : chaîne Go du bloc d'usage de Sessions (fichiers, champs, `WithSessionUsage`,
  résolveur d'amis, câblage `registry_pages.go:343-345` remplacé par le résumé d'usage seul dans
  `cablerBlocsSessions`), `objectives.go`, `ComputeFlagGrabsNet`, `ResolveTrackedSquad`,
  `objective_role_rows_repo.go`, types de domaine de §4.E ; tests supprimés avec leur code.
- [x] S5.3 §4.F : `ComputeUsage` et le reste de `usage.go` hors types de lecture, `usage_families.go`,
  `newMatchPoint`, `computeOutcomes` / `attachOutcomes` / `subjectBilanFamilies`, `ComputePadTiers` /
  `PadTiersInput`, `squadagg/pad_tier_labels.go`, types de `domain/session_usage.go` listés ; tests
  suivent (`usage_test.go`, cas `ComputeUsage` et `TestBilan_MetricKeysSurLeSeulSujet` de
  `usage_outcomes_test.go`, `pad_tiers_test.go` si son sujet sort) ; `PlayerOutcomeCounts` et sa
  garde / son golden verts sans modification d'assertion ; web : `usagePadTiersModel.ts` (+ test)
  [~] supprimé en S4.14 (mort dès S4), et ce que knip désigne alors, alias de
  `lib/api/types.ts:2299-2312` (plus aucun lecteur web de `SessionUsageOutcomes` /
  `SessionUsageMetric` / `SessionUsageMatchPoint` depuis S4.14).
- [x] S5.4 §4.D / D10 : riposte et `FenetreMs` du contrat et du calcul ; `bloc_test.go` et les tests
  du service adaptés (Appui identique avant / après : test de non-régression écrit AVANT la coupe sur
  la fixture existante) ; fixtures de `TimeseriesCoordinationSection.test.tsx` et de Sessions suivent.
- [x] S5.5 Contrat régénéré ; snapshot `contract-surface` régénéré par la procédure, disparitions
  listées.
- Gate : gate Go + `-tags=integration -p 1 ./internal/platform/duckdb/...` + contrat + gate web ;
  preuves §4.A-F rejouées → 0.

Journal S5 (2026-10-06, exécuteur, `feat/sessions-emprise` sur `548a0b30d` ; session coupée une fois par la limite de quota, reprise sur l'état réel de l'arbre) — suppressions Go, riposte retirée du contrat :
- **S5.4, test d'abord** : `analysis/coordination/bloc_appui_golden_test.go` (NEUF) fige octet pour octet le versant appui du bloc (couverture, deux couvertures, parité pondérée, versant appui de chaque case) sur trois scénarios ; golden `testdata/bloc_appui.golden.json` PRODUIT par le bloc d'avant la coupe (aucun fichier de coordination encore modifié), puis rejoué VERT sur l'ancien code (`bloc.go`, `bloc_appui.go`, `domain/coordination_block.go` reconstitués par `git show 548a0b30d:…`, restaurés ensuite) ET sur le nouveau : l'appui est identique avant / après.
- **S5.1** Preuves rejouées avant chaque suppression : §4.D (lecteurs Go de la riposte : `analysis/coordination/bloc*.go`, `service/coordination_block.go`, `service/session_page_coordination.go` ; `squad_echange`, `MatchRiposteBlock`, `Echanges`, `FenetreEchangeMs`, `Ripostes` lus ailleurs et GARDÉS), §4.E (`WithSessionUsage`, objectifs, prises nettes, escouade suivie : lecteurs = fichiers supprimés + commentaires), §4.F (lecteurs qualifiés `sessionusage.*` hors paquet : `squademprise` — `PlayerOutcomeCounts`, `OutcomeCounts`, `PowerupFamilies`, `PowerupEffect`, `PadTierRow`, `BuildTeamContext`, `TeamContext` —, `squadformes` / `squadagg` — `MatchInput`, `BuildMatchInputs`, `ResolveScopeFriends` — et le repo ; tous gardés).
- **S5.2** supprimés : `service/session_page_usage{,_labels}.go` (+ tests), `session_page_flag_grabs_net_test.go`, `pad_tiers_wiring_test.go`, `analysis/sessionusage/objectives.go` (+ test), `platform/duckdb/objective_role_rows_repo.go` (+ test), `ComputeFlagGrabsNet` / `FlagGrabsNetInput` (+ test ; `FlagGrabsNetRow` garde), `ResolveTrackedSquad` (+ cas de test), champs `Usage` / `CompareUsage`, `usageXUID` / `usageFriends`. `WithSessionUsage` remplacé par `WithSessionUsageSummary(repo, repoRoot)`, câblé sous `film.usage_summary` dans `cablerBlocsSessions` (garde-rail de câblage réécrit sur ce nom) ; `registry_pages.go` 619 → 615 L. Helpers de test partagés déplacés dans `service/session_usage_mock_test.go`.
- **S5.3** `usage.go` réduit aux lignes et à `BuildMatchInputs` (`MatchInput` sans durée ni compteurs de grain match ; `FilmRow.PadUnnamed` sans lecteur : retiré, et sa colonne du `SELECT` du repo) ; supprimés `usage_families.go`, `newMatchPoint`, `computeOutcomes` / `attachOutcomes` / `subjectBilanFamilies`, `ComputePadTiers` / `PadTiersInput`, `squadagg/pad_tier_labels.go`, tests `usage_test.go`, `usage_outcomes_test.go` (tous ses cas passaient par `ComputeUsage`), `pad_tiers_test.go` ; golden et garde de la bascule « utilisé » inchangés et verts ; `team_context_test.go` (NEUF) reprend la couverture directe de `BuildTeamContext` / `BuildMatchInputs` que portaient les tests supprimés. `domain/session_usage.go` réduit aux raisons machine, à `SessionUsageSquadPlayer` et aux constantes de niveau ; `PadTierOrder` sans lecteur : supprimé avec `pad_tiers_vocabulary_test.go` et `pad_tiers_web_parity_test.go` (ce dernier lisait `usagePadTiersModel.ts`, supprimé en S4 — S4 l'avait rendu ROUGE, §8). Test d'intégration `session_usage_aggregate_integration_test.go` réécrit sur ce qui reste (persister réel, vue `_latest`, assemblage).
- **S5.4** `CoordinationRiposte`, `Riposte` du bloc et des soirées, `FenetreMs` du bloc, champs riposte des cases, `compterRipostes`, `agregerRiposte`, `medianeMs`, champs riposte de `cumulMatch`, habituel de « je suis couvert » ; `CoordinationEntree.Kills` n'avait plus de lecteur dans le bloc : retiré (et son remplissage par le service) — le golden d'appui, privé de ses événements, reste identique. Tests adaptés : `bloc_test.go` (appui seul ; parité mixte pondérée par les appuis), `coordination_block_test.go`, `session_page_coordination_test.go`, `timeseries_service_equipes_test.go` ; web : fixtures de `TimeseriesCoordinationSection.test.tsx` réécrites sur `appui.on_me_prepare`, commentaire de `timeseriesCoordination.logic.ts`.
- **S5.5** Contrat : `openapi.yaml` −554 / +0, `generated.ts` −264 / +0 ; `openapi-gen -check` à jour ; `check-generated-types-fresh` OK ; snapshot `contract-surface` régénéré par la procédure (`UPDATE_CONTRACT_SURFACE=1`), 15 schémas disparus : `CoordinationRiposte`, `SessionFlagGrabsNetBlock`, `SessionObjectiveFamilyBlock`, `SessionObjectiveRoleMetric`, `SessionObjectivesBlock`, `SessionUsageBlock`, `SessionUsageMatchPoint`, `SessionUsageMetric`, `SessionUsageOutcomes`, `SessionUsagePadFamily`, `SessionUsagePadTier`, `SessionUsagePadTierWeapon`, `SessionUsagePadTiersBlock`, `SessionUsagePowerup`, `SessionUsageSquadShare` ; alias morts de `lib/api/types.ts` retirés (dont `SessionUsageSquadPlayer`, sans lecteur web).
- Commentaires devenus faux corrigés (règle 17) : `squademprise/{input.go, build_test.go}`, `domain/equipmentusage/families.go`, `duckdb/squad_formes_repo.go`, `port/{match_range.go, session_usage.go}`, `service/{session_page_range.go, session_page_service.go, coordination_block.go}`. Laissés : récits datés (`migration/steps_shared_flag_grabs_net.go`, correctif C1 de `usage_outcomes.go`) et `squadagg/squad_formes.go:13` (fichier interdit par §5.1, §8).
- **Mutations** (script `mut_s5.ps1`, restauration vérifiée) : 5, toutes ROUGES — appui reçu compté hors camp et parité non pondérée (golden), résumé d'usage câblé hors de sa porte (garde-rail de câblage), partant compté dans l'effectif (team_context), habituel non posé (coordination de Sessions).
- **Gate** : `go build ./...` 0 ; `gofmt -l` muet ; `go vet ./internal/...` 0 ; `go test -count=1` du module en lots couvrant les 348 paquets de `go list ./...` (cœur 67 ok / 157 s, games 39 ok, platform + service 28 ok / 267 s, sync + persist + migration + hors internal 16 ok, reste 45 ok) — 0 FAIL ; `go test -tags=integration -p 1 ./internal/platform/duckdb/...` 4 ok (436 s) ; `make go-api-lint` 0 issue (le balayage `unused` des paquets touchés ne relève que des aides de test PRÉ-EXISTANTES hors périmètre, §8) ; web : purge `.tmp`, `tsc -b --force` 0, lint 0 erreur (26 avertissements), vitest complet 840 fichiers / 8 871 tests verts, knip 0 / 0 / 0 (aveugle), couleurs 0, imports croisés 7 ≤ 7, `lefthook run pre-push` sortie 0. Baseline de présence : aucun test supprimé ou renommé n'y figure (relevé par différence des noms).
- Seuils : `session_page_service.go` 886 (inchangé), `registry_pages.go` 615 ; aucun fichier créé ou modifié au-delà de 500 L ; diff du lot 58 fichiers, +274 / −6 087.

### S6 — fondu dans S5

- [~] S6 Orphelins d'intersection avec le lot TS : rebase sur `262e36b2e` (L6, L7 intégrés), items
  portés par S5.1 et S5.3.

### S7 — Clôture · rapide

- [x] S7.1 Docs : `docs/CHANGELOG.md` + `docs/FR/CHANGELOG.md` (bloc `[7.5.0]` : phrases propres à
  Sessions — numéros relevés AVANT L8, qui a retouché ces fichiers : à relire — EN l. 37 (prises nettes « on Squad and Sessions »), 39 (usage sur trois pages),
  44 (riposte « on the Sessions »), 54 (Sessions en quatre sections) — corrigées, entrée ajoutée) ;
  `docs/RELEASE_NOTES.md` + `docs/FR/RELEASE_NOTES.md` (bloc 7.5 : EN l. 32, 39, 58) ; lignes
  re-vérifiées au moment d'écrire, FR aux lignes homologues ; aucune phrase propre aux Séries
  temporelles touchée (L8.1 du plan TS). Plus une ligne CHANGELOG FR / EN « Halo 5 : Appui reçu
  disponible sur Sessions » (décision du superviseur après S2, §8).
- [x] S7.2 `.ai/V7.5/REFERENCE_CANAUX_EQUIPEMENT_2026-09-09.md` §4, lecteurs de Sessions (l. 293-296)
  et tableau l. 482.
- [x] S7.3 ADR 0036 : vérifier qu'aucun invariant n'est touché (lectures bornées existantes, une
  lecture du résumé d'usage par scope) ; aucune ADR neuve.
- [x] S7.4 Statut de chaque item du plan ; §8 Découvertes relues ; entrée finale du journal.
- [~] S7.5 Revue adversariale du diff cumulé : à demander au SUPERVISEUR (l'exécuteur n'a pas de
  sous-agent) — lots à risque : S2 (lectures partagées, recâblage de la coordination), S5 (contrat).
  DEMANDÉE dans le compte rendu de clôture ; le superviseur la lance (consigne du 2026-10-06).
- Gate : gate Go complet + gate web complet + contrat, rejoués après les docs.

Journal S7 (2026-10-06) :
- **S7.1** CHANGELOG EN / FR (bloc `[7.5.0]`), lignes relues au moment d'écrire (L8 les avait
  déplacées) : ligne d'équipement (lue dans la carte « Équipement ramassé » de l'onglet « Usage » des
  Séries temporelles ET de la page Sessions, plus les trois pages d'usage) ; ligne de coordination
  (Sessions = Appui reçu seul, des deux côtés du tiroir, la Riposte et sa part du contrat ont quitté
  la page) ; deux entrées neuves après celle de l'onglet « Usage » : « The Sessions page reads the Map
  control of its own matches » (cartes, tiroir compact, champs Go ajoutés, retraits) et « Halo 5:
  Support received available on Sessions ». RELEASE_NOTES EN / FR : l. 32 (appui sur les Sessions et
  sur la durée aux Séries temporelles), l. 39 (« On the Sessions and the Timeseries »), entrée neuve
  « The Sessions page becomes the Map control of your evening » après « Sessions in four sections ».
  La ligne des prises nettes « published on Squad and Sessions » (EN / FR l. 37) reste VRAIE et n'est
  pas touchée : Sessions les publie toujours, par la feuille d'objectif partagée (`formes_retenues`,
  colonne du rôle « prendre », `squad/objectif/objectif.logic.ts`). Aucune phrase propre aux Séries temporelles touchée. Les chaînes partagées de
  l'Emprise (`squad/emprise/`, `_shared/`) ne sont pas touchées (lot sémantique en cours, §8).
- **S7.2** Référence équipement : colonnes du film lues (`duration_ms`, `powerup_pickups_json`),
  puce « Page Sessions » réécrite (blocs `emprise` / `compare_emprise`, lecteurs Go et web, ancien bloc
  `usage` retiré en S5), compteur des départs aléatoires parti avec le bloc d'usage, tableau : ligne
  « lue par l'Emprise » (`pad_tiers.go`, `squademprise/`), ligne `usagePadTiersModel.ts` retirée.
- **S7.3** ADR 0036 : I1 / I2 (aucune lecture `v_gamertag_lookup` ni `_latest` non bornée : toutes
  les lectures neuves sont bornées aux matchs de la session et au joueur), I3 (cache invalidé au sync :
  inchangé, aucun cache neuf), I4 (un chargement par requête : résumé d'usage lu UNE fois par session,
  partagé par l'Emprise, l'objectif et l'appui via `LecturesUsage`), I5, I7 non touchés. **I6
  ÉCART TROUVÉ ET CORRIGÉ** : l'Emprise et l'emblème de Sessions (code de S2) ne déclaraient pas de
  section de durée. Test `TestAttachSessionBlocks_SectionsDeDuree` (vu ROUGE : `emprise`, `emblem`
  absentes), puis `Section("emprise")` dans `sessionEmprise` et `Section("emblem")` dans
  `attachSessionEmblem` ; mutations (retrait de chacune) ROUGES. Aucune ADR neuve.
- **S7.4** Cases : S1 à S5 `[x]`, S6 `[~]` (fondu dans S5, S5.1 / S5.3), S5 « `usagePadTiersModel.ts` »
  `[~]` (S4.14), S7.5 `[~]` (demandée au superviseur) ; aucun `[!]`. §8 relu : entrées complétées
  (leçon de S5 retenue par le superviseur, item d'intégration des textes, écart I6 des Séries
  temporelles).
- **Gate** (rejoué APRÈS les docs) : `go build ./...` 0 ; `gofmt -l` muet ; `go vet ./internal/...`
  0 ; `go test -count=1` des 348 paquets en lots (195 ok, 153 sans test, 0 FAIL) ;
  `go test -tags=integration -p 1 ./internal/platform/duckdb/...` 4 ok (353 s) ; `make go-api-lint`
  0 issue ; contrat : `openapi-gen -check` à jour, `check-generated-types-fresh` OK ; web : purge
  `.tmp`, `tsc -b --force` 0, lint 0 erreur (26 avertissements), vitest 840 fichiers / 8 871 tests
  verts, knip 0 / 0 / 0, couleurs 0, imports croisés 7 ≤ 7, `lefthook run pre-push` sortie 0.

## 7. Reprise de session

Relire le skill `plan-execution`, puis ce fichier (cases, journaux de lot), puis les dernières
entrées de `.ai/thought_log.md` du worktree et `git -C <worktree> log --oneline -10`. Reprendre à la
première case non statuée du lot courant. Les décisions du §1 sont fermes une fois le « go » donné.

## 7 bis. Relecture plan-review (2026-10-06, phase 1)

Grille `.claude/skills/plan-review/SKILL.md`, passée sur ce fichier :
- §1 structure : objectif et critère (§0), lots ordonnés du refactor sans effet (S1) au contrat (S5)
  puis aux orphelins dépendants (S6), effort par lot, branche nommée, bloqueurs documentés (§9) — OK.
- §2 couches Go : calculs existants dans `analysis/` (`squademprise`, `coordination`), aucun calcul
  neuf hors du score déjà centralisé ; types dans `domain/` ; orchestration dans `service/` (fichiers
  neufs, aucun handler modifié) ; aucun port neuf (les ports `SoloLivesRepository`, `EmblemURLLoader`,
  `SquadEmpriseRepository`, `KillSourceCategoryRepository` existent) ; aucun SQL hors
  `platform/duckdb` ; aucune requête neuve — OK.
- §3 multi-titre : capabilities `film.usage_summary`, `film.vehicle_usage`, `film.kill_positions`,
  `match.objective.stats`, journal des morts ; feuille et emblème inconditionnels ; Halo 5 testé
  (S2.3, S4.2) ; aucun `slug ==` (`TestNoNewSlugComparison` au gate) — OK.
- §4 adapters : lectures par repos de port, comme les blocs frères (pas de `TitleDataAdapter`) ;
  écart conforme à l'existant — OK.
- §5 tests : service avec mocks (S2), câblage (S2.9), logique et composants web (S3, S4), rangées
  partagées et tiroir (S4.8), garde-rails neufs (S3.1, S4.15) ; `:memory:` : aucune requête neuve ;
  intégration DuckDB en S5 (suppression d'un repo) — OK.
- §6 logs : dégradations journalisées (Debug capability absente, Error lecture en échec, Info bilan)
  dans les helpers partagés (S1) et les passes neuves (S2) — OK.
- §7 front : aucune route, aucune clé de requête ; chaînes FR + EN `Record<Locale>` et manifeste ;
  libellés de nature par les manifestes `frags` ; jetons seulement — OK.
- §8 livraison : gates par lot, journal par lot, dépendance externe documentée (S6 ↔ L6 / L7) — OK.
- §9 exécutabilité : périmètres fermés (listes, preuves grep, knip comme juge des morts web), gates à
  commandes exactes, statuts et règle « aucune case vide », ordre strict, Découvertes, reprise,
  renvoi au skill — OK.
Ajustement après rebase sur `262e36b2e` (2026-10-06) : §4.F rejoué (chaque symbole n'est plus lu que
par la chaîne de Sessions), S6 fondu dans S5 (S5.3), §5.1 réécrit (lot TS clos), lignes citées
déplacées par L6 / L7 recalées (D4, D5, D7, S1.2, S2.9, §4.E), requête réduite de
`BuildSquadFormesBlock` (L7) reportée dans D7.
Défauts trouvés et CORRIGÉS à la relecture : (1) la première version supprimait en S5 tout
`domain/session_usage.go` et `sessionusage/usage.go` — impossible tant que L6 n'est pas intégré
(`usage_outcomes_test.go`, `ComputePadTiers`, `NommerArmesDesNiveaux` y ont encore un lecteur, et L6
modifie `usage_outcomes_test.go`) : séparés en §4.E (S5) et §4.F (S6, dépendant). (2) La coordination
perdait son xuid et son effectif de camp avec le bloc d'usage (D9 ajouté, S2.2). (3) Les briques
compactes sans lecteur auraient rougi knip : elles sont des props de composants existants (S3), les
modèles neufs ont leur lecteur dans le lot.

## 8. Découvertes (à consigner ici, pas à traiter)

- (phase 1) `domain.TimeseriesLivesNearTeammate`, `timeseriesEmpriseMatches`, `timeseriesFormesMetas`
  portent « timeseries » dans leur nom et serviront aussi Sessions (D4, D5) : renommage non fait
  (fichiers du lot TS, bruit de contrat).
- (phase 1) Le tiroir de comparaison relit le résumé d'usage, les vies et l'objectif de la session
  comparée par des lectures séparées de celles de la session affichée (comme le bloc d'usage
  aujourd'hui) ; une lecture commune découpée (patron `lectureCoordination`) n'est pas visée.
- (phase 1) `BuildSquadFormesBlock` lit encore `LoadUsageFilmPads` jusqu'à L7 (lecture inutile pour
  l'objectif seul) : sort avec L7.
- (phase 1) `SessionDetailPage.tsx` (551 L) et `session_page_service.go` (894 L) sont au-delà du seuil
  de taille ; le lot les fait baisser sans viser le seuil.
- (S2) Les niveaux de socle (`LoadPadTiers`) sont lus deux fois par session tant que le bloc d'usage
  historique vit (lui et la lecture du film de l'Emprise) ; la seconde lecture disparaît avec lui en S5.
- (S2, SIGNALÉ AU SUPERVISEUR) Avant S2, la page Sessions d'un titre au journal des morts fiable mais
  sans résumé d'usage recevait une coordination INDISPONIBLE (raison `no_measured_match` : son joueur
  venait du câblage du résumé, absent). HALO 5 EST DANS CE CAS (`match.killfeed.per_kill` = supported,
  `config/titles/halo_5/mappings/capabilities.toml:48`, pas de `film.usage_summary`). Depuis S2 (D9 :
  joueur posé par `WithSessionEmprise`, inconditionnel), Halo 5 reçoit sa coordination sur Sessions,
  sans parité — comme les Séries temporelles, dont le joueur vient de `WithHighlightEventsRepo`. D9
  écrivait « comme aujourd'hui » : c'était inexact. DÉCIDÉ par le superviseur (2026-10-06, option
  a) : comportement gardé — la carte dépend du journal des morts, l'ancienne porte était un accident
  de câblage ; ligne de changelog FR / EN en S7.1.
- (S3) `lib/clipboard/useCopyToClipboard.guard.test.ts` (« l'allowlist ne garde aucune entrée morte »)
  a dépassé son délai par défaut de 5 s sous la charge de la suite complète (5 078 ms) ; vert seul
  (2,6 s) et au second passage complet. Fichier non touché par le lot ; délai du test non traité.
- (S3) Les comptes « 0 prise » / « Non mesuré » de la maquette pour les ressources absentes (pistes en
  attente) restent non portés (D13) dans les vues compactes aussi : une ressource sans prise n'a pas de
  ligne, comme en pleine page.
- (S4) §4.G est allé plus loin que la liste des candidats : le seul producteur des piles d'issues, des
  repères de taux, du repli de lignes, de la hachure « lobby » et des lignes de total de
  `UsageGaugeGrid` était `buildGaugeRow`, lu par les seules cartes d'usage de Sessions ; avec elles,
  ces branches de `UsageForms.tsx`, `UsageHatchLegend.tsx` et la majeure partie de `usageI18n.ts`
  (518 → 124 L) sont sorties (règle 7, conséquence directe du lot). `usagePadTiersModel.ts`, prévu en
  S5 par §4.F, est sorti en S4 (mort dès S4).
- (S4) Deux types exportés sans importeur, PRÉ-EXISTANTS, dans des fichiers dont le lot n'a touché qu'un
  commentaire : `FragWeaponBreakdownProps` (`components/charts/FragWeaponBreakdown.tsx`),
  `TimeseriesSummaryTabProps` (`timeseries/TimeseriesPage.summary.tsx`). Non traités (hors périmètre ;
  le step knip de la CI les juge).
- (S4) `lib/capabilities/FeatureGate.tsx:25` cite encore `usageAvailability` dans un récit daté (une
  prop retirée le 2026-09-06) : historique, non une affirmation sur le code actuel — laissé tel quel.
- (S4) `lint-no-hardcoded-fields` balaie les fixtures (hors `squad/emprise/emprise.fixtures.ts`,
  allowlisté) : les noms de cartes du relevé solo ont été anonymisés dans `sessionEmprise.fixtures.ts`
  plutôt que d'allonger l'allowlist.
- (S5) S4 avait rendu ROUGE un test Go : `domain/pad_tiers_web_parity_test.go` lisait
  `_shared/usage/usagePadTiersModel.ts`, supprimé en S4 (le gate de S4 est web seul). Corrigé en S5 :
  le test et `PadTierOrder`, sans lecteur après la coupe, sont supprimés. **LEÇON (retenue par le
  superviseur, 2026-10-06) : toute suppression web se double d'un grep des tests Go qui lisent
  `apps/web/src`** — un gate web seul ne voit pas ces tests.
- (S5) `service/squadagg/squad_formes.go:13` cite encore `service/session_page_usage_labels.go`
  (supprimé) comme précédent d'arbitrage : fichier interdit par §5.1, commentaire non corrigé.
- (S5) Aides de test sans lecteur, PRÉ-EXISTANTES et hors périmètre, relevées par `unused` :
  `mockSessionCompareSessionsRepo` / `mockSessionCompareStatsRepo` (`service/sessions_service_test.go`),
  `float64Ptr` (`service/teammates/testhelpers_test.go`). Non traitées.
- (S5) Incident d'exécution : une commande de collage de journal s'est terminée par un `python -`
  tapé par erreur (interdit) ; aucun code Python n'a été exécuté (le processus attendait l'entrée
  standard), il a été arrêté par `TaskStop`. Aucun effet sur l'arbre (vérifié par `git status`).
- (S7, ITEM D'INTÉGRATION) Un lot sémantique est en cours sur `feat/ts-usages-fix-equipement` : titres
  factuels sans personne, FR / EN, sur les chaînes PARTAGÉES de l'Emprise. Le lot Sessions n'y touche
  pas et l'héritera à l'intégration. À ce moment-là, les textes PROPRES à Sessions
  (`features/session-detail/sessionEmpriseText.ts`) s'alignent sur la même règle : aucun possessif ni
  pronom de personne (ma / mes / mon camp / moi / notre / nous / ta), joueur = gamertag, « Camp »,
  « Adversaire », « Reste du camp ». Non fait dans ce lot (consigne du superviseur, 2026-10-06).
- (S7) Écart à l'invariant I6 de l'ADR 0036 sur les Séries temporelles : le chargement de l'emblème
  (`service/timeseries_service_emprise.go:83`, `LoadEmblemURLs`) ne déclare pas de section de durée
  (celui de Sessions, même patron, est corrigé dans ce lot). Fichier du lot TS : non traité.

## 9. Questions au superviseur — RÉPONDUES le 2026-10-06

Réponses : Q1 OUI (périmètre §5.1, ajouts rétro-compatibles, pages Escouade et Séries temporelles
identiques, tests de page rejoués sans modification) ; Q2 option (a) ; Q3 garder (D14). Questions
d'origine :

- **Q1** D1 / D4 / D5 / D17 modifient des fichiers livrés par L1-L5 du lot TS (liste §5.1, aucun
  dans le périmètre de L6-L8). Accord ? Sans accord : D4 / D5 deviennent des secondes copies
  (admises par la règle 6, contraires au brief) et les cartes C-L n'ont pas de variante compacte
  sans copie — le plan ne tient plus.
- **Q2** Base de la phase 2 : (a) attendre la clôture de L8 et rebaser `feat/sessions-emprise` sur la
  tête finale de `feat/ts-usages-emprise` (recommandé : §4.F tombe dans S5, S6 disparaît, aucun
  conflit de contrat) ; (b) démarrer sur `cd3145ec2`, S6 restant dépendant d'une intégration faite
  par le superviseur.
- **Q3** D14 : garder « Précision par arme » pour Halo 5 (décision du plan) ou la retirer avec
  `SessionFragCard` (la chaîne Go `WeaponAccuracy` de Sessions sort alors en S5).
