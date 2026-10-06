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
> base). Un second rebase peut suivre la revue adverse du lot TS : sur demande du superviseur
> seulement.
> Branche : `feat/sessions-emprise`, créée sur `cd3145ec2`, rebasée sur `262e36b2e` ; worktree
> `C:\Users\Guillaume\Downloads\Scripts\LevelUp-wt-sessions`.

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

- [ ] S3.1 D17 : `empriseObjectName` dans `emprise.logic.ts` ; `useEmpriseModels.ts:29-36` et
  `useUsagesModels.ts:29-36` migrés ; garde-rail `squad/emprise/emprise.objectName.guard.test.ts`
  (balayage `import.meta.glob` : aucun fichier hors `emprise.logic.ts` ne combine
  `RESOURCE_VEHICLE` et `vehicleFamilyName` dans une fonction de nommage) ; auto-test sur l'ancien
  littéral ; mutation : copie réintroduite → rouge.
- [ ] S3.2 `ResourceControlCard` + `PisteCampsForm` : `compact` → parts seules dans les segments,
  comptes au survol, râteliers en une ligne de ressource (sans dépliage). Tests (rouges d'abord) :
  libellé « 60 % » au lieu de « 12 », infobulle avec les comptes, aucun bouton de dépliage.
- [ ] S3.3 `ProductionCard` : `compact` → parts seules (épaisse et fine), ligne d'exposition en % ;
  `YieldCard` inchangée (vérifier qu'aucun compte n'y est écrit hors rendements bruts, comme la
  maquette `makeYield`).
- [ ] S3.4 `empriseCharts.ts` / `ResourceFilCard` : option `compact` du mode `match` — hauteur 170,
  graduations 0 / 50 / 100, pas d'étiquette sous la bande, étiquettes de fin seules (maquette
  `renderFil` l. 1312-1316, 1344). Tests sur l'option ECharts ; mutation : dates sous l'axe en compact
  → rouge.
- [ ] S3.5 Grille compacte : `compactGridRows(grid)` dans `emprise.logic.ts` (lignes de ressource
  seulement : bonus, armes spéciales prises, frags aux armes spéciales, râteliers, véhicules ; case =
  part de mon camp ou état « sans film » / « non classé » / « rien à prendre ») ; `ResourceGridTable`
  et `ResourceMatchGridCard` : `compact` → ces lignes, case « 71 % » colorée par l'écart (même
  encre), en-tête heure + carte + V / D, gabarit `minmax(0, 1fr)` (S11). Tests : lignes et ordre,
  « ? » sans niveaux, « — » hachuré sans film, aucune largeur minimale de colonne ; mutation : ligne
  d'objet en compact → rouge.
- [ ] S3.6 `MinePickupsCard` : `compact` → une barre par ressource (ma part des prises de mon camp en
  %, reste en %), bonus perdus des deux camps en % (comptes au survol) ; modèle `buildMineCompact` dans
  `usages.logic.ts` (depuis `resources[].taken.us`, `objects[].squad[0].taken`, `outcomes`). Tests
  rouges d'abord (témoin MESURES §3 s2209 : bonus 3 / 12 = 25 %, armes spéciales 7 / 13 = 53,8 %,
  râteliers 3 / 24 = 12,5 %, bonus perdus 16,7 % / 25 %) ; mutation : part sur le total des deux
  camps → rouge.
- [ ] S3.7 `EquipmentOutcomesCard` : `compact` → segments en %, barre fine gardée, sous-libellé
  « n objets », ligne du reste « p % servis ». Tests ; mutation : sous-libellé avec les prises en
  compact → rouge.
- [ ] S3.8 `LivesNearTeammateCard` : `compact` → parts seules dans la barre épaisse et sur la ligne
  des frags (D16). Tests ; mutation : comptes en compact → rouge.
- [ ] S3.9 `ObjectiveBalanceCard` : `compact` → une barre par RÔLE et par famille (somme des actions
  du rôle, Tenir en durée) plus la ligne des prises nettes, % seuls ; modèle `buildBalanceByRole` dans
  `objectif.logic.ts`. Tests (témoin MESURES §3 s0709 : Bases prendre 93-109 = 46 %, défendre 30-60 =
  33,3 %, tenir 53,6 % ; Drapeau prises nettes 29-41 = 41,4 %) ; mutation : Tenir sommé avec les
  comptes → rouge.
- [ ] S3.10 `ObjectiveSoloSheetCard` : `compact` → la valeur affichée devient la part du camp (« 32 % »,
  « — » si le camp n'a rien), pied en parts (témoin s2209 : Prendre 31,8 %, Défendre 30,3 %, Tenir
  47,2 %). Tests ; mutation : compte affiché en compact → rouge.
- [ ] S3.11 `SquadFragBreakdownCard` : `compact` → parts dans les segments, total en sous-libellé du
  nom, pas de total au bout ; `SquadWeaponKillsChart` / `buildSquadToolRows` : option `top` (6) et
  valeur en part des frags du joueur, « Non attribué » exclu en compact (maquette `makeTools`
  l. 942). Tests ; mutation : « Non attribué » gardé en compact → rouge.
- Gate : gate web (aucun Go) ; tests de page Escouade / Séries temporelles nommés ci-dessus rejoués,
  `git diff` vide sur ces fichiers de test.

### S4 — Web : la page Sessions reconstruite, suppressions web · lourd

Périmètre : `features/session-detail/*`, `tools/lint-cross-feature-imports.mjs` (D2),
`lib/i18n/manifests/session.toml` (+ manifeste généré), `lib/api/types.ts` (alias). Aucune requête
neuve, aucune clé de requête neuve (`lib/query/keys.ts` non touché) : tout arrive avec
`useSessionDetailPage`.

- [ ] S4.1 `_sections.ts` (D11) : clés par carte, sous-groupes et leur clé de manifeste, paires de la
  pleine page, `groupSessionSections` à deux niveaux, `sessionSectionKeys` sur un objet de présence par
  carte ; `sessionRowKeys(data)` (calcul de `rowKeys` sorti de `SessionDetailPage.tsx:261-276`, fichier
  en baisse). Tests ROUGES d'abord (`_sections.test.ts`) : ordre, intertitre de sous-groupe sur la
  première clé présente, sous-groupe absent sans intertitre, union des deux colonnes, paires.
  Mutations : intertitre posé sur une clé absente, ordre de l'union pris sur la gauche seule → rouges.
- [ ] S4.2 `sessionEmprise.logic.ts` (NEUF) : index des matchs depuis `SessionDetailMatchRow`
  (heure, `map_name`, `mode_ui`, résultat `outcomeCodeToValue`, `score_label`, `dominance_flag`),
  couverture, modèles par carte (composition des builders existants : `buildControlRows`,
  `buildResourceFil`, `buildMatchGrid`, `buildMinePickups`, `buildProductionRows`, `buildYieldRows`,
  `buildVehicleCoverage`, `buildLivesModel`, `buildEquipmentRows`, `objectiveMatches`,
  `buildObjectiveBalance`, `buildSoloObjectiveSheet`), et LE prédicat de présence par carte
  (`sessionCardsPresence(column)`) lu par `_sections.ts`, `SessionColumnBody` et chaque carte. Tests
  ROUGES d'abord, une mutation par règle (index : dominance ignorée ; présence : carte objectif sans
  match à objectif, Halo 5 sans film → seules A, B, B' et la barre épaisse des armes spéciales de G si
  la feuille la porte).
- [ ] S4.3 `sessionEmpriseText.ts` (D12) : surcharges FR / EN, deux jeux par carte (pleine page /
  comparaison) ; test : titres, ⓘ et légendes FR copiés de la maquette pour chaque `make*` et chaque
  valeur de `cp`, aucun « Notre camp » / « Our side » / « périmètre » / « scope ».
- [ ] S4.4 `SessionColumnBody.tsx` : props `usage` → `blocks: SessionColumnBlocks` (emprise, vies,
  formes, emblème, lignes de match, coordination, portée) ; montage des clés A-L et B' ; paires de la
  pleine page ; intertitres de sous-groupe et couverture S14 ; `compact` transmis à chaque carte avec
  le jeu de textes de sa vue. `SessionDetailPage.tsx` : passe les blocs courants et comparés (taille
  non accrue).
- [ ] S4.5 `SessionCoordinationSection.tsx` : carte Riposte retirée (§4.C), « Appui reçu » seule,
  demi-largeur en pleine page, empilée en compact ; tests adaptés.
- [ ] S4.6 Cartes A et B : `SessionFragBarCard.tsx` et `SessionToolsCard.tsx` (NEUFS, minces : cadre,
  titre, ⓘ, légende) sur `SquadFragBreakdownCard` / `SquadWeaponKillsChart` (S3.11), un joueur
  (couleur `squad-player-1`), libellés de nature par les manifestes `frags` (patron
  `SquadFragSection.tsx:61-74`). B' : `WeaponAccuracyChart` monté tel quel (D14).
- [ ] S4.7 Cartes C-L : montage des briques (S3) avec les modèles S4.2 et les textes S4.3 ; D en
  axe `match` avec l'index des matchs de la session (encoche de dominance, bande de résultats) ; K avec
  `player_emblem_url` et le gamertag du joueur.
- [ ] S4.8 Tests de page : `SessionColumnBody.test.tsx` réécrit (ordre des cartes, intertitres,
  couverture, retrait par carte, Halo 5, anglais) ; `SessionCompareRows.test.tsx` étendu (une rangée
  partagée par carte, même liste des deux côtés, marqueur du côté sans la carte — session solo sans
  objectif face à une session à objectif, témoin MESURES §0 —, intertitres dans la même rangée des
  deux côtés, cartes en variante compacte des DEUX côtés, aucun intertitre ni rangée en pleine page) ;
  `sessionShrink.guard.test.ts` étendu aux paires de cartes (une colonne en `compact`) ; fixture
  `sessionEmprise.fixtures.ts` tirée des MESURES (s2209, s0709, solo).
- [ ] S4.9 D2 : paire `session-detail=>timeseries` ajoutée à `ALLOWED_CROSS_IMPORTS` avec son
  commentaire ; ratchet ≤ 7 ; aucune dérogation morte.
- [ ] S4.10 Rejouer CHAQUE preuve grep de §4.A, B, C, G avant de supprimer ; écart → §8, arrêt propre
  si un lecteur inattendu existe.
- [ ] S4.11 Web §4.A : fichiers et tests supprimés, `sessionSectionVisibility.ts` réduit ou supprimé.
- [ ] S4.12 Web §4.B : `SessionFragCard.tsx` supprimé.
- [ ] S4.13 Web §4.C : riposte de `coordinationModel.ts` et `coordinationI18n.ts`.
- [ ] S4.14 Web §4.G et §4.H (web) : `_shared/usage/*` orphelins, exports et clés devenus privés ou
  supprimés, manifestes régénérés, gardes d'usage adaptées.
- [ ] S4.15 Garde-rail D7 : `features/session-detail/formesFields.guard.test.ts` — aucun fichier de
  `session-detail` ne lit un champ de `formes_retenues` hors de la liste de L7 (balayage des accès
  `.lobby`, `.weapons`, `.weapon_pads`, `.pad_named`, `.pad_unnamed`, `.duration_seconds`,
  `.team_size`, `.lobby_size`, `.measured` sur les objets de formes) ; auto-test ; mutation → rouge.
- [ ] S4.16 Ratchets : knip 0 / 0 / 0, imports croisés ≤ 7 ; si un plafond baisse, l'abaisser.
- Gate : gate web ; preuves §4.A, B, C, G rejouées → 0 (côté web).

### S5 — Go : suppressions (bloc d'usage entier) et riposte, contrat · lourd

Le web ne lit plus `usage`, `compare_usage` ni la riposte depuis S4. Depuis le rebase sur
`262e36b2e`, §4.F (anciens orphelins d'intersection) est dans ce lot.

- [ ] S5.1 Rejouer les preuves §4.D, §4.E et §4.F (producteurs et lecteurs Go et web).
- [ ] S5.2 §4.E : chaîne Go du bloc d'usage de Sessions (fichiers, champs, `WithSessionUsage`,
  résolveur d'amis, câblage `registry_pages.go:343-345` remplacé par le résumé d'usage seul dans
  `cablerBlocsSessions`), `objectives.go`, `ComputeFlagGrabsNet`, `ResolveTrackedSquad`,
  `objective_role_rows_repo.go`, types de domaine de §4.E ; tests supprimés avec leur code.
- [ ] S5.3 §4.F : `ComputeUsage` et le reste de `usage.go` hors types de lecture, `usage_families.go`,
  `newMatchPoint`, `computeOutcomes` / `attachOutcomes` / `subjectBilanFamilies`, `ComputePadTiers` /
  `PadTiersInput`, `squadagg/pad_tier_labels.go`, types de `domain/session_usage.go` listés ; tests
  suivent (`usage_test.go`, cas `ComputeUsage` et `TestBilan_MetricKeysSurLeSeulSujet` de
  `usage_outcomes_test.go`, `pad_tiers_test.go` si son sujet sort) ; `PlayerOutcomeCounts` et sa
  garde / son golden verts sans modification d'assertion ; web : `usagePadTiersModel.ts` (+ test) et
  ce que knip désigne alors, alias de `lib/api/types.ts:2299-2312`.
- [ ] S5.4 §4.D / D10 : riposte et `FenetreMs` du contrat et du calcul ; `bloc_test.go` et les tests
  du service adaptés (Appui identique avant / après : test de non-régression écrit AVANT la coupe sur
  la fixture existante) ; fixtures de `TimeseriesCoordinationSection.test.tsx` et de Sessions suivent.
- [ ] S5.5 Contrat régénéré ; snapshot `contract-surface` régénéré par la procédure, disparitions
  listées.
- Gate : gate Go + `-tags=integration -p 1 ./internal/platform/duckdb/...` + contrat + gate web ;
  preuves §4.A-F rejouées → 0.

### S6 — fondu dans S5

- [~] S6 Orphelins d'intersection avec le lot TS : rebase sur `262e36b2e` (L6, L7 intégrés), items
  portés par S5.1 et S5.3.

### S7 — Clôture · rapide

- [ ] S7.1 Docs : `docs/CHANGELOG.md` + `docs/FR/CHANGELOG.md` (bloc `[7.5.0]` : phrases propres à
  Sessions — numéros relevés AVANT L8, qui a retouché ces fichiers : à relire — EN l. 37 (prises nettes « on Squad and Sessions »), 39 (usage sur trois pages),
  44 (riposte « on the Sessions »), 54 (Sessions en quatre sections) — corrigées, entrée ajoutée) ;
  `docs/RELEASE_NOTES.md` + `docs/FR/RELEASE_NOTES.md` (bloc 7.5 : EN l. 32, 39, 58) ; lignes
  re-vérifiées au moment d'écrire, FR aux lignes homologues ; aucune phrase propre aux Séries
  temporelles touchée (L8.1 du plan TS).
- [ ] S7.2 `.ai/V7.5/REFERENCE_CANAUX_EQUIPEMENT_2026-09-09.md` §4, lecteurs de Sessions (l. 293-296)
  et tableau l. 482.
- [ ] S7.3 ADR 0036 : vérifier qu'aucun invariant n'est touché (lectures bornées existantes, une
  lecture du résumé d'usage par scope) ; aucune ADR neuve.
- [ ] S7.4 Statut de chaque item du plan ; §8 Découvertes relues ; entrée finale du journal.
- [ ] S7.5 Revue adversariale du diff cumulé : à demander au SUPERVISEUR (l'exécuteur n'a pas de
  sous-agent) — lots à risque : S2 (lectures partagées, recâblage de la coordination), S5 (contrat).
- Gate : gate Go complet + gate web complet + contrat, rejoués après les docs.

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
  écrivait « comme aujourd'hui » : c'était inexact. Décision attendue (garder l'alignement sur les
  Séries temporelles, ou rendre au joueur de la coordination la porte du résumé d'usage).

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
