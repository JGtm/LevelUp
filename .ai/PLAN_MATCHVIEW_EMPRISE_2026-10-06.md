# Plan : Vue match aux formes de l'Emprise — 2026-10-06

> Sources, à lire avant tout lot, qui FONT FOI pour le rendu :
> - maquette validée par l'utilisateur le 2026-10-06 (« ok validé pour la vue match »), position
>   « Après » : https://claude.ai/artifact/Pk9WDrhLgm6yY2wECPWw3H (v2), copie
>   `.ai/V7.5/MAQUETTE_MATCHVIEW_2026-10-06.html` (script lisible : `renderApres` l. 1160-1184,
>   `makeTools` l. 947, `renderDistance` l. 652 (filtre de la mêlée l. 653), `makeControl` l. 968,
>   `makeSheets` l. 1007, `makeEquipGrid` l. 1045, `makeProd` l. 1055, `makeYield` l. 1088,
>   `makeLives` l. 1124, `renderAssists` l. 824 (en-tête « a assisté… » l. 834)). Ne se portent
>   pas : lignes « remplace : … », encarts « Maquette. » (`mnote`), bandeau « À soumettre »
>   (l. 1013), pastilles numérotées, phrases `line1` « … retirée » (l. 1167, 1183) ;
> - relevés : `.ai/V7.5/MESURES_MATCHVIEW_2026-10-06.md` (§3 chiffres dérivés par carte des deux
>   matchs témoins `ab526724` Starboard et `4f77afc1` Flood Gulch BTB) ;
> - brief `BRIEF_IMPLEM_MATCHVIEW.md` (décisions utilisateur du 2026-10-06, §1-§2) ;
> - plans frères dont ce lot réutilise les briques : `.ai/PLAN_TIMESERIES_USAGES_EMPRISE_2026-10-05.md`
>   (lot TS, D1-D15, S1-S10, journaux L1-L6) et `.ai/PLAN_SESSIONS_EMPRISE_2026-10-06.md` (lot
>   Sessions, dans `LevelUp-wt-sessions`, D1-D18, lots S1-S7) ;
> - `.ai/thought_log.md`, entrées du 2026-10-05 (relevé des rendus v1) et compléments du 2026-10-06
>   (maquette Vue match rendue, décisions utilisateur, validation).
>
> Contrat d'exécution : skill `plan-execution` (ordre strict, un lot à la fois, gate passé avant le
> suivant, aucun item sans statut, zéro fix hors périmètre, découvertes consignées §8). Statuts :
> `[x]` fait et vérifié, `[~]` couvert ailleurs (référence), `[!]` non fait (justification écrite).
> Aucune case vide à la clôture d'un lot. « Clos » = les 5 actions de la règle 6 du skill.
>
> Statut du plan : **ACCEPTÉ par le superviseur le 2026-10-06 (D1-D22 fermes, réponses §9)** ;
> **phase 2 exécutée** le 2026-10-07 (M0-M5 clos, revue adversariale demandée au superviseur) (signal du superviseur : rebase sur la tête de `feat/v75`
> `2668848b1`, qui porte les lots Séries temporelles et Sessions, puis « go » M0-M5 ; push à la fin,
> CI ; ni merge ni autre rebase). Amendements de M0 : §1.4.
> Branche : `feat/matchview-emprise`, créée sur `651bbe972` (tête L6 de `feat/ts-usages-emprise`),
> worktree `C:\Users\Guillaume\Downloads\Scripts\LevelUp-wt-matchview`.

## 0. Objectif, critère de succès, hors périmètre

**Objectif.** La Vue match (`features/match-view/`) montre ses statistiques du film aux formes de
l'Emprise, pour UN match, en COMPTES exhaustifs (D21 du projet), dans l'ordre et les formes de la
maquette « Après » : onglet « Armes et terrain » A-J (§3), Chronologie › Faits marquants inchangés,
Joueurs › Assistances avec l'axe « a assisté… ». « Riposte » et « Hauteur d'engagement » quittent la
page ; « Part de chaque équipe » et « Contrôle des armes spéciales » sont remplacées. Tout ce qui
perd son dernier lecteur est supprimé (règle n° 7), Go et web, contrat compris.

**Critère de succès.** (1) Les cartes A-J du §3 rendues dans l'ordre, conformes à la maquette
(S1-S12) ; (2) chaque carte se retire seule sans donnée, sans film et sur Halo 5 (jamais un 500,
jamais un zéro inventé) ; (3) tous les gates des lots verts, dernière exécution dans la session ;
(4) inventaire §4 supprimé, chaque preuve grep à 0 ; (5) docs du lot clôture à jour.

**Hors périmètre** (consigné, non traité) :
- Chronologie › Faits marquants (`MatchImpactBadgesBar`) : décision utilisateur, on n'y touche pas.
- « Occupation du terrain » (`MatchPositionsHeatmap`) : inchangée.
- La grille d'équipement par joueur (`ValueGrid` de `MatchEquipmentUsageSection`) garde sa forme et
  sa source (l'artefact de rejeu) : seuls son titre, son ⓘ, sa légende et les mots de ses
  infobulles changent (carte F).
- Pas de nouvelle carte d'objectif (le tableau des scores en a déjà une, décision utilisateur).
- Escouade › Synergies (riposte, hauteur) : lot à part de l'utilisateur, non lancé.
- Q20 (`GetMatchKVPairs`) et son lecteur `kvPairs` : RESTENT (voir P3 du §9) ; seul le bloc Riposte
  qui les relisait sort.
- `FragWeaponBreakdown` et `buildFragDetailBreakdown` : RESTENT (lus par la Synthèse et les Séries
  temporelles, P4 du §9).
- Fichiers des lots TS L7/L8 et du périmètre Sessions (`features/session-detail/`,
  `service/session_page*`, `domain/session_*`) : aucun n'est modifié (§5.1).

## 1. Décisions

### 1.1 Validées par l'utilisateur (2026-10-06) — fermes

- **V1** La riposte et la hauteur sortent de toutes les pages ; l'appui (assistances) reste.
- **V2** Un match = des COMPTES exhaustifs, jamais un taux ; les cartes par joueur de MON camp ont un
  sens ici (les deux camps et tout le roster sont connus).
- **V3** « Qui a pris quoi chez nous » (fiches par joueur de mon camp) : GARDÉE.
- **V4** « Répartition des frags » : l'ANNEAU est gardé tel quel.
- **V5** « Distance par arme » : gardée, sans la MÊLÉE ; libellés de joueur à la taille des autres
  axes de la page.
- **V6** « Faits marquants » : inchangés.
- **V7** « Assistances » : graphe gardé, axe « Assistances par patron » → « a assisté… » (FR) /
  « assisted… » (EN).
- **V8** Pas de nouvelle carte d'objectif.
- **V9** Ordre et contenu de l'onglet « Armes et terrain » : §3 (A-J).

### 1.2 Tranchées par le planificateur — FERMES (plan accepté par le superviseur le 2026-10-06 : D1-D22 confirmées ; Q1 ordre du brief — joueur de la page, puis profils suivis, puis le reste de mon camp dans l'ordre du tableau des scores, bots et partis exclus des fiches et comptés dans les infobulles, même ordre pour la carte I ; Q2 oui — D sans lignes « en attente », une ressource sans prise n'a pas de ligne, les véhicules suivent leur couverture, G et H gardent leurs lignes « non mesuré » de la liste fermée, les trois textes absents de la maquette écrits par l'exécuteur en FR et EN ; Q3 oui — helpers S1.1 / S1.2 et `solo_lives_repo.go` étendus à signature existante inchangée, tests existants verts sans modification, une mutation prouve que chaque nouveau paramètre est lu ; attribution : ligne de la session de l'exécuteur)

- **D1 — Dépendance au lot Sessions, nommée.** La phase 2 part de la tête finale de
  `feat/sessions-emprise`. Symboles attendus (plan Sessions §6, lot S1 ; noms définitifs relus en
  M0) : `service.buildSoloEmpriseBlock` et `service.soloEmpriseQuery` (`service/solo_emprise_block.go`,
  S1.1) ; `service.lireViesPresOuSeul` et `service.viesQuery` (`service/solo_lives_block.go`, S1.2) ;
  `squadagg.BuildWeaponTools`, `squadagg.WeaponToolInputs`, `squadagg.PlayersAboveSheet`
  (`service/squadagg/weapon_tools.go`, S1.3) ; web : `empriseObjectName` (`squad/emprise/emprise.logic.ts`,
  S3.1) et la prop `compact` des briques (S3, défaut `false`, non utilisée ici). Un symbole absent ou
  renommé en M0 : la table §1.3 est mise à jour (renommage seulement) ; un symbole dont la FORME
  diffère (paramètres, retour) : arrêt et compte rendu.
- **D2 — Le bloc Emprise du match réutilise l'assemblage solo de Sessions (S1.1), étendu d'un champ.**
  Trois assemblages existeraient sinon (Escouade, solo, match) : règle n° 6. `soloEmpriseQuery` gagne
  `Players []domain.SessionUsageSquadPlayer` (nil = comportement actuel : `squadagg.SquadPlayers`, le
  joueur seul) ; la Vue match l'appelle avec `WithMaps: false` et `Players` = les joueurs de mon camp
  (D3). Vérifié sur pièces : `squadagg.SquadPlayers` (`squadagg/squad_formes.go:129-145`) passe par
  `sessionusage.ResolveScopeFriends` (`analysis/sessionusage/squad.go:94-129`), qui TRIE par matchs
  partagés puis gamertag et TRONQUE à `MaxTrackedSquadPlayers` : il ne peut pas rendre onze joueurs
  dans l'ordre du tableau des scores, d'où la liste passée toute faite. Le bloc publié est
  `domain.MatchEmpriseBlock` (§3) : le `SquadEmpriseBlock` du résultat (la carte Équipement que
  calcule l'assemblage solo n'est pas publiée : la carte F reste sur l'artefact) plus
  `kill_journal_publishable` (D7).
- **D3 — Les joueurs de mon camp, et leur ordre.** Lignes de `d.scoreboard` (Q12) dont `TeamID` vaut
  celui du joueur de la page, `IsBot` faux, `LeftInProgress` différent de `true` (présents à la fin ;
  partis et bots exclus — ils restent dans le « reste du camp » des comptes, jamais une fiche). Ordre :
  le joueur de la page, puis les profils suivis (clés de `friendsExtras`, `match_view_data_loaders.go:442-457`)
  dans l'ordre du tableau des scores, puis les autres dans l'ordre du tableau des scores. *Écart à la
  maquette*, qui suit l'ordre du tableau des scores sans remonter les profils suivis (m2407 :
  Madina97294, suivie, en 5e fiche) : le brief §2 E dit « joueur de la page en premier, profils suivis
  ensuite » — le brief est retenu (Q1). Fonction pure `matchCampPlayers` (service), testée.
- **D4 — Les fiches E sans fiche « reste du camp ».** `buildPickupSheets` (`squad/emprise/emprise.logic.ts:241-272`)
  ajoute toujours une fiche « reste du camp » (l. 246). Brief : bots et partis exclus des fiches. La
  Vue match retire cette dernière fiche dans son modèle (`matchEmprise.logic.ts`, fonction pure) ; les
  infobulles gardent « n des m prises de mon camp » avec m = le camp ENTIER (comptes exhaustifs, V2).
  La légende de `PickupSheetsCard` (l. 49-61) perd son 3e item quand `restColor` est absent (prop
  rendue optionnelle, défaut inchangé pour l'Escouade).
- **D5 — Les râteliers dans D et E.** Vérifié : le bilan `resources` du bloc ne porte PAS les râteliers
  (`analysis/squademprise/build.go:94-116`, `bilan` : bonus, armes spéciales, véhicules), alors que
  les objets du match les portent (`match.go:176-185`, `tallyTiers` ; `match.go:231-249`,
  `publierMatch` : une `SquadEmpriseMatchResource` par ressource tracée, râtelier compris). D (une
  piste par ressource puis par objet) se construit donc sur `emprise.matches[0].resources` ; E reçoit
  la liste de ses sections : `buildPickupSheets` gagne un 3e paramètre optionnel `resources?: string[]`
  (défaut : `buildControlRows(block)`, comportement actuel ; la Vue match passe les ressources de
  `matches[0]`, râtelier compris).
- **D6 — Prises sur un emplacement non identifié, au grain du match.** Non publiées aujourd'hui
  (`tierResource`, `input.go:146-156`, écarte `non_classe`). Ajout additif au contrat : 
  `SquadEmpriseMatch.UnclassifiedPickups *SquadEmpriseCount` (`json:"unclassified_pickups,omitempty"`),
  calculé dans `tallyMatch` (`match.go:100-121`) sur les lignes `domain.PadTierUnclassified`
  (`domain/session_usage.go:361`) d'un match mesuré (film ET camp connu), quel que soit l'état des
  niveaux, mon camp / adversaire par la même règle de camp (`camp.side`, `match.go:59-67`) ; nil sans
  ligne non classée. Lu par la seule Vue match (ligne sous D) ; les autres pages le reçoivent sans le
  lire (champ de contrat, pas code mort).
- **D7 — « Journal des morts publiable » : un fait de la page.** Les frags pendant l'effet d'un bonus
  (G, H) et les frags par vie (I) ne se lisent que si le journal du match est publiable ligne à ligne
  (constat MESURES §3 : `4f77afc1` a 0 ligne publiable sur 294 ; ses frags sous effet valent 0/0 et ne
  sont pas une mesure). Q21d (`platform/duckdb/match_view_repo_assist_pairs.go:73-80`, CTE `scope`,
  déjà chargée par la page, `match_view_data_loaders.go:173-177`) gagne
  `COUNT(*) FILTER (WHERE publishable) AS publishable_deaths` ;
  `domain.MatchAssistScopeRaw` (`domain/assist_pairs.go:70-73`) gagne `PublishableDeaths int`.
  Publié : `MatchEmpriseBlock.KillJournalPublishable` et `MatchLivesNearTeammate.FragsMeasured`
  (= `PublishableDeaths > 0`). Aucune requête de plus.
- **D8 — « Vies » : une lecture pour tout le camp (ADR 0036 I4), calcul existant par joueur.** Nouveau
  port étroit `port.CampLivesRepository.LoadLivesNearTeammateForPlayers(ctx, matchIDs []string,
  xuids []string) (map[string]domain.ViesLues, error)`, mis en œuvre par `duckdb.SoloLivesRepo`
  (`platform/duckdb/solo_lives_repo.go`) : les quatre requêtes actuelles (l. 38-65) filtrent
  `list_contains(?, xuid)` au lieu de `= ?` et rendent le xuid ; `LoadLivesNearTeammate` (l. 70-109)
  devient un appel de la nouvelle méthode avec un seul xuid (une seule copie des requêtes ; signature
  et port `SoloLivesRepository` inchangés, les mocks des Séries temporelles et de Sessions restent
  valides). Liste des matchs toujours liée en constante sur le `match_id` de chaque vue (I2). Le
  service découpe par joueur et appelle `coordination.ViesPresOuSeul`
  (`analysis/coordination/vies_pres_ou_seul.go:33`) pour chacun : aucun type ni fonction neuve dans
  `coordination` (garde `TestAucunTauxNu` inchangée). L'assemblage S1.2 (`lireViesPresOuSeul`) est
  généralisé : `lireViesDuCamp(ctx, viesCampQuery)` porte les journaux et la portée du radar, et
  `lireViesPresOuSeul` l'appelle avec un seul xuid (une seule copie).
- **D9 — « Outils de destruction » (B) par `squadagg.BuildWeaponTools` (S1.3).** Lignes d'arme :
  `d.bulkWeapons` (déjà chargé, Q bulk weapons, `match_view_data_loaders.go:222-226`) du joueur de la
  page, projetées en `port.WeaponKillRow` (`port/weapon_kills.go:77-130`) — `Label` et `LabelEN` =
  `WeaponLabel` (déjà dans la langue de la requête, `domain/match_view_raw.go:300-304`), `Class`,
  `WeaponKey`, `MechanicKills`, `FromDamageSource` recopiés (champ à champ, les deux types ont la même
  forme) ; catégories de source : `port.KillSourceCategoryRepository` (`port/kill_source.go:74-80`)
  par assertion sur le repo d'armes câblé (`r.weaponKillsRepoFor(pdb)`, `registry_pages.go:510-518`),
  filtres `{MatchIDs: [match], XUIDs: [moi]}` ; absente ou `ErrCapabilityNotSupported` → nil (Debug),
  autre erreur → Warn, les deux lignes manquent (reliquat) ; feuille : la ligne du joueur au tableau
  des scores (frags, mêlée, grenades, mécaniques natives sous `titleHasNativeKillMechanics`). Publié :
  `MatchCombatTab.WeaponTools *domain.SquadWeaponTools`. Web : `squad/SquadWeaponKillsChart` +
  `buildSquadToolRows` (`squad/charts/squadFragTools.ts:49-64`), un joueur, couleur du joueur de la
  page (paire d'imports `match-view=>squad` déjà déclarée, `tools/lint-cross-feature-imports.mjs:166`).
- **D10 — A et B côte à côte, survol lié retiré.** `MatchFragCard` (`match-view/MatchFragCard.tsx:41-87`)
  garde l'anneau (`FragSunburst`, inchangé) et remplace `FragWeaponBreakdown` par la carte B ; grille
  `lg:grid-cols-2` à colonnes égales (maquette `grid2`, l. 1163), anneau seul en pleine largeur quand
  B n'a pas de ligne. Le survol lié anneau ↔ détail par arme (l. 43-44, 69-70, 80-81) disparaît : B
  (graphe ECharts de l'Escouade) ne l'offre pas. Prédicat `hasMatchFragData`
  (`blockPredicates.ts:64-74`) : anneau OU au moins une ligne d'outil.
- **D11 — Mêlée exclue de « Distance par arme » dans le LECTEUR.** `KillDistanceRepo.resolveRows`
  (`platform/duckdb/kill_distance_repo.go:152-211`) écarte, avant agrégation, une clé dont la classe du
  registre (`resolveWeaponKeyLabelsAny` rend déjà `class`, `weapon_resolver.go:227-264`) vaut
  `domain.FragClassMelee` (`domain/frag_distribution.go:35`). Halo Infinite : la clé de mêlée générique
  sort, l'épée à énergie (classe `heavy`, migration `metadata_reclass_sword_hammer`) reste — comme la
  maquette (`renderDistance` l. 653). Seul lecteur : la Vue match (`match_view_data_loaders.go:235-240`).
  La résolution des libellés passe donc AVANT l'agrégation (une requête de métadonnées, comme avant).
- **D12 — Libellés de joueur de « Distance par arme ».** `_killDistanceChart.ts:213-222` : le `rich`
  (`arme`, `joueur`) ne fixe pas de `fontSize` et hérite de l'axe partagé. Pour ne pas dépendre de
  l'héritage ECharts, les deux styles reprennent explicitement `axis.axisLabel.fontSize` ; test qui
  l'asserte. Aucune autre retouche de la carte.
- **D13 — Lignes « en attente » de la maquette : portées en G et H, pas en D.** Brief : G et H disent
  « non mesuré » avec la raison ; D n'en dit rien. D suit la règle des pages sœurs (D13 du plan TS) :
  une ressource sans prise n'a pas de ligne (les lignes « 0 prise… », « Non mesuré : aucune prise
  publiée… », « Véhicules : non mesuré » de `makeControl` l. 981-991 ne se portent pas) ; la ligne
  « n prises sur un emplacement non identifié » (D6), elle, se porte. G et H : liste FERMÉE de raisons
  (§3, carte G / H), calculée par `matchEmprise.logic.ts` depuis le bloc (aucune règle dans un
  composant), affichée par une prop optionnelle `pending` de `ProductionCard` / `YieldCard` et une
  ligne `pending` de `PisteCampsForm` (défaut : aucune, Escouade / Séries temporelles / Sessions
  inchangées). Trois raisons n'ont pas de texte dans la maquette (aucun témoin n'y tombe) et sont
  écrites par le plan : « Aucun frag pendant l'effet d'un bonus », « Non mesuré : feuille de match
  illisible », « frags non mesurés : journal des morts non publiable » (I).
- **D14 — « Vies » : une ligne par joueur, la ligne de la carte solo réutilisée.** La rangée de
  `timeseries/usages/LivesNearTeammateCard.tsx:56-76` sort en `LivesNearTeammateRow` (même fichier
  de dossier, `timeseries/usages/LivesNearTeammateRow.tsx`), paramétrée (libellé, sous-libellé, préfixe
  d'identifiants de test, frags mesurés ou non) ; `LivesNearTeammateCard` l'utilise (rendu inchangé,
  ses tests rejoués sans modification). La Vue match l'importe : paire `match-view=>timeseries`
  ajoutée à `ALLOWED_CROSS_IMPORTS` avec son commentaire (même décision que D2 du plan Sessions pour
  `session-detail=>timeseries` ; la paire ne compte pas dans le plafond 7). Modèle par joueur :
  `buildLivesModel` (`timeseries/usages/usages.logic.ts`) appliqué à chaque joueur. Journal non
  publiable (D7) : la barre fine et la ligne des frags sont remplacées par « frags non mesurés :
  journal des morts non publiable » (jamais 0 frag inventé).
- **D15 — Carte F : ⓘ de la maquette et réserve de couverture.** Titre « Équipement : servi, gardé,
  lâché, par joueur » ; ⓘ = texte de `makeEquipGrid` (l. 1048) suivi, quand elle est non nulle, de la
  phrase de réserve actuelle (`coverageReserveFmt`, `MatchEquipmentUsageSection.tsx:181-182` : la
  réserve ne se cache pas, décision utilisateur du 2026-09-09 citée l. 320-322) ; légende = les trois
  issues (« Servi », « Gardé sans servir », « Lâché ») + « Tractions de grappin » quand la colonne
  existe ; les trois formateurs d'issue des infobulles (`outcomeUsedFmt` / `outcomeKeptFmt` /
  `outcomeDroppedFmt`, `match-replay/model/equipmentUsageColumns.ts:219-224`) prennent les mêmes mots
  (même carte, même vocabulaire).
- **D16 — Intertitre « Équipement et terrain » et sa couverture.** Comme la maquette (l. 1168) et comme
  le bilan des Séries temporelles (L5.2 du plan TS) : « film décodé · N joueurs présents à la fin » ou
  « sans film », calculé par `matchEmprise.logic.ts` (`emprise.matches[0].has_film` ; N = lignes du
  tableau des scores sans `left_in_progress`, bots compris — maquette : 8 sur Starboard, 24 sur le
  BTB) ; aucune autre phrase.
- **D17 — Textes.** Nouveau fichier `features/match-view/matchEmpriseText.ts` (`Record<Locale, …>`) :
  surcharge de `EMPRISE_TEXT` pour D, E, G, H (« Mon camp », « chez nous » dans le titre de E comme la
  maquette), textes de B et I, raisons G / H, intertitre D16 ; FR mot pour mot de la maquette, EN
  traduits. `match-view/i18n.ts` (1 158 L, au-delà du seuil, dette gelée) n'est pas agrandi : seules
  des clés y sont retirées (§4.B) et une valeur y change (V7).
- **D18 — Câblage dans un fichier neuf.** `api/wire/registry_pages.go` (620 L, au-delà du seuil) : les
  l. 131-133 (repo de distance) deviennent UN appel `svc = r.cablerFilmMatchView(svc, pdb)` dans
  `api/wire/registry_pages_matchview.go` (NEUF) — distance (même porte qu'aujourd'hui), feuille de
  l'Emprise (inconditionnelle), résumé d'usage sous `CapFilmUsageSummary`, véhicules sous
  `CapFilmVehicleUsage`, vies sous `CapFilmKillPositions` + portée du radar (`r.radarRangeFor`),
  catégories de source (repo d'armes). Taille de `registry_pages.go` en baisse.
- **D19 — Dépendances du service dans une struct embarquée.** `service/match_view_service.go` (568 L,
  au-delà du seuil) : les dépendances neuves vivent dans `matchViewEmpriseDeps` déclarée dans
  `service/match_view_emprise.go` (NEUF), embarquée par UNE ligne ; leurs `With*` dans le même
  fichier (patron `usagesDeps`, `timeseries_service_emprise.go:24-57`).
- **D20 — Une seule ligne d'embarquement dans `domain/match_view.go`.** Le fichier (878 L) reçoit
  `MatchViewEmpriseFields` embarqué dans `MatchViewResponse` (une ligne ; struct déclarée dans
  `domain/match_emprise.go`, NEUF, champs `emprise` et `lives_near_teammate` aplatis dans le JSON —
  même mécanisme que `SoloEmpriseBlock`) et `WeaponTools` dans `MatchCombatTab` (deux lignes) ; M4 en
  retire les blocs Riposte (l. 485-491) et Elevation (l. 521-527) : solde négatif.
- **D21 — Attribution des commits** : ligne système de cette session
  (`Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`), comme D15 du plan TS et D18 du plan
  Sessions (le brief cadre §0 en citait une autre).
- **D22 — Section de durée et journaux.** `attachMatchEmprise` et `attachMatchLives` déclarent chacune
  sa section (`match_emprise`, `match_lives`, ADR 0036 I6) ; journaux `emprise_*` / `vies_*` avec
  l'attribut `page` = `match_view` (les helpers S1 le portent déjà).

### 1.3 Table des dépendances S1 (relue en M0)

| Attendu (plan Sessions) | Fichier | Usage ici |
|---|---|---|
| `buildSoloEmpriseBlock(ctx, soloEmpriseQuery) *domain.SoloEmpriseBlock` | `service/solo_emprise_block.go` (S1.1) | M1.5 (champ `Players`), M2.3 |
| `lireViesPresOuSeul(ctx, viesQuery)` | `service/solo_lives_block.go` (S1.2) | M1.4 (généralisé), M2.4 |
| `squadagg.BuildWeaponTools(WeaponToolInputs) *domain.SquadWeaponTools`, `squadagg.PlayersAboveSheet` | `service/squadagg/weapon_tools.go` (S1.3) | M2.5 |
| `empriseObjectName(o, usageText, unknownVehicle)` | `squad/emprise/objectName.ts` (S3.1 ; relu en M0 : fichier propre, pas `emprise.logic.ts`) | M3 (noms d'objets de D et E) |

Relu en M0 (base `2668848b1`) : les quatre symboles existent avec la forme attendue —
`soloEmpriseQuery{Page, Player, PlayerXUID, RepoRoot, TitleSlug, Locale, Current, Lectures,
UsageRepo, EmpriseRepo, VehicleRepo, WithMaps}` (`service/solo_emprise_block.go:21-41`) ;
`viesQuery{Page, Player, PlayerXUID, Repo, Radar, MatchIDs}` et `lireViesPresOuSeul`
(`service/solo_lives_block.go:26-69`) ; `squadagg.WeaponToolInputs{Rows, Categories, PlayersOrdered,
GtByXUID, Sheet, HasMechanics}`, `BuildWeaponTools`, `PlayersAboveSheet`
(`service/squadagg/weapon_tools.go:34-45,80,228`).

### 1.4 Amendements de M0 (2026-10-07, sur pièces et sur consigne du superviseur)

- **A1 (D7, D14) — Journal non publiable : la règle existe déjà pour les vies.** Depuis les correctifs
  du lot TS, `SoloLivesRepo` nomme les matchs au journal non publiable (`qSoloJournalNonPubliable`,
  `solo_lives_repo.go:61-67,123-138`) et `coordination.ViesPresOuSeul` ÉCARTE leurs vies et les COMPTE
  (`ExcludedUnpublishable`, `vies_pres_ou_seul.go:64-67`). La carte I suit cette règle : pas de champ
  `FragsMeasured`, pas de texte « frags non mesurés » ; un joueur dont toutes les vies sont écartées
  n'a pas de barre, la carte se retire quand aucun joueur n'a de vie rangée, et l'ⓘ compte les trois
  causes (texte de « Isolement », `usagesCardsText.ts:129-135`). D7 reste pour G et H seulement
  (`publishable_deaths` de Q21d) ; la lecture multi-joueurs (D8) garde la lecture du journal, commune à
  tous les joueurs.
- **A2 (D17) — Règles de texte du 2026-10-06 (validées par l'utilisateur, consigne du superviseur du
  2026-10-07).** Titres = noms de domaine courts, déclinés par un complément factuel, jamais de verbe
  ni de phrase ; AUCUN mot de personne (ma, mes, mon, moi, notre, nous, ta, tes, ton, me, je, « chez
  nous ») ; le joueur = son gamertag ; groupes « Équipe », « Adversaire », « Reste de l'équipe » ;
  « équipe », jamais « camp ». Aides ⓘ : une ou deux phrases, mesure et périmètre. Même carte qu'une
  page sœur = même titre, même aide (portée « le match »). Titres retenus : A « Répartition des frags »,
  B « Outils de destruction », C « Distance par arme », D « Contrôle des ressources, par match »,
  E « Prises par joueur » (sens : qui a pris quoi dans l'équipe), F « Usage d'équipements, par
  joueur », G « Frags par ressource », H « Rendement par ressource », I « Isolement, par joueur ».
  Les textes de la maquette ne se portent plus mot pour mot : D17 et M3.3 suivent ces règles.
- **A3 — Dette « Mes matchs ».** `MatchViewPage.tsx:146,177` écrit « Mes matchs » / « My matches » en
  dur → « Matchs » / « Matches » (M3.13). La garde `timeseries/usages/textesSansPersonne.test.ts` est
  ÉTENDUE à `features/match-view/` (une seule garde, pas de copie) : textes `MATCH_VIEW_TEXT`,
  `MATCH_EMPRISE_TEXT` et littéraux de phrase des sources de la feature ; vue rouge avant, verte après
  (M3.13).
- **A4 — Bloc `combat_tab` et coordination.** La riposte du bloc `coordination` est sortie avec
  Sessions D10 ; les types `MatchRiposte*` de la Vue match sont désormais `domain/coordination_block.go:161-211`
  (en-tête l. 10). §4.F suit ces numéros.

## 2. Spécification de rendu (non négociable)

Reprend S1-S10 du plan TS (§2) ; en cas de doute, la maquette fait foi, puis ce §2.

- **S1** Titre factuel, ⓘ si besoin (texte de la maquette), graphique, légende centrée en bas — rien
  d'autre (aucune phrase de lecture, aucun pied de carte ; les `mnote` ne se portent pas).
- **S2** Valeurs DANS les segments si elles tiennent (`components/charts/segmentLabelFit.ts`), repli
  au-dessus sinon, jamais seulement en infobulle.
- **S3** `team-ally` / `team-enemy` à la place des mots ; hachure réservée à « sans film ».
- **S4** Trait 50 % pointillé `warning`, « 50 % : autant que l'adversaire ».
- **S5** Aucun vert / rouge sur une répartition interne ; les fiches E et les lignes I prennent la
  palette de joueurs du match (`match-view/colors.ts:91`, `buildMatchPlayerColors` : moi, profils
  suivis, reste de mon camp) ; I garde « près » `squad-player-1` / « seul » `extreme` (maquette
  `NEAR_INK` / `ALONE_INK`, l. 471).
- **S6** Couleurs de ressource `resource-*`, pastille devant chaque nom ; couleurs de classe de frag
  pour la pastille de B.
- **S7** FR + EN (`Record<Locale, …>`), « FDA » jamais « KDA », aucun anglicisme
  (`lib/i18n/no-anglicisms.guard.test.ts`), aucun emoji.
- **S8** Aucune couleur en dur ni classe Tailwind de couleur (`tools/lint-no-hardcoded-colors.mjs`).
- **S9** « Mon camp » ; légende de D : noms d'équipe résolus (`resolveTeamLabel`) avec « (mon camp) »
  pour le mien (maquette l. 973).
- **S10** Chaque carte se retire seule sans donnée ; section sans carte → pas d'intertitre (règle
  `blockPredicates.ts:1-19`) ; onglet sans rien → l'état vide existant (`arsenalEmptyTitle`).
- **S11** Disposition (maquette `renderApres`) : A | B en deux colonnes égales ; C pleine largeur ;
  D, E, F pleine largeur ; G | H en deux colonnes ; I, J pleine largeur.
- **S12** Valeurs de la maquette pour les témoins (MESURES §3) : D m2209 bonus 5–2, armes spéciales
  0–2, râteliers 4–3 (Fusil électrique 2–2, VK78 Commando 2–0, CQS48 Bulldog 0–1) ; m2407 armes
  spéciales 19–11, râteliers 29–40, 31 prises non identifiées (19 / 12) ; I m2209 (portée 18 m) JGtm
  11 près / 2 seul, 8 / 0 frags.

## 3. Cartes cibles (ordre à l'écran)

| # | Carte | Source Go (contrat) | Composant web |
|---|---|---|---|
| A | Répartition des frags | `combat_tab.frag_distribution` (inchangé) | `FragSunburst` dans `MatchFragCard` (D10) |
| B | Outils de destruction | `combat_tab.weapon_tools` (D9, NEUF) | `match-view/MatchToolsCard.tsx` (NEUF, mince) sur `squad/SquadWeaponKillsChart` |
| C | Distance par arme | `combat_tab.kill_distance_by_weapon` sans mêlée (D11) | `MatchKillDistanceSection` (D12) |
| D | Contrôle des ressources du match | `emprise.matches[0].resources` + `unclassified_pickups` (D5, D6) | `match-view/MatchResourceControlCard.tsx` (NEUF) sur `PisteCampsForm` (lignes indentées, râteliers repliés) |
| E | Qui a pris quoi chez nous | `emprise.objects[].squad`, `emprise.players`, `resources[powerup].outcomes` | `squad/emprise/PickupSheetsCard` (D4, D5) |
| F | Équipement : servi, gardé, lâché, par joueur | artefact de rejeu (inchangé) | `MatchEquipmentUsageSection` réduit à la grille (D15) |
| G | Frags obtenus avec les ressources | `emprise.production`, `kill_journal_publishable` | `squad/emprise/ProductionCard` + prop `pending` (D13) |
| H | Rendement face à l'adversaire | `emprise.production` | `squad/emprise/YieldCard` + prop `pending` (D13) |
| I | Vies : près d'un coéquipier ou seul | `lives_near_teammate` (D8, NEUF) | `match-view/MatchLivesCard.tsx` (NEUF) sur `LivesNearTeammateRow` (D14) |
| J | Occupation du terrain | positions (inchangé) | `MatchPositionsHeatmap` |
| — | Joueurs › Assistances | `combat_tab.assist_pairs` (inchangé) | `MatchAssistChart`, axe « a assisté… » (V7) |

Raisons fermées (D13), dans l'ordre des ressources, seulement sur un match mesuré
(`has_film && team_known`) sauf mention :
- G bonus : effet mesuré mais journal non publiable → ligne « Frags pendant l'effet non mesurés :
  journal des morts non publiable » + barre fine du temps d'effet ; aucun temps d'effet ni frag →
  « Aucun temps d'effet mesuré » ; temps d'effet et journal publiable mais 0 frag des deux côtés →
  « Aucun frag pendant l'effet d'un bonus ».
- G armes spéciales : feuille lue, 0 frag des deux côtés → « 0 frag aux armes spéciales dans les deux
  camps » (y compris sans film) ; feuille en échec (`sheet_load_failed`) → « Non mesuré : feuille de
  match illisible » (y compris sans film) ; frags présents sans prise mesurée → barre épaisse et ligne
  « aucune prise d'arme spéciale mesurée » sous elle (maquette l. 1078).
- G / H véhicules : titre qui mesure les véhicules (`emprise.vehicles` présent) et match non mesuré
  (`matches[0].vehicles` ≠ `measured`) → « Non mesuré » ; sinon aucune ligne sans frag.
- H bonus : journal non publiable → « Non mesuré : frags pendant l'effet non publiés » ; un camp sans
  temps d'effet → « Non mesurable : <camp> n'a eu aucun temps d'effet (mon camp <durée>, <n> frags) ».
- H armes spéciales : un camp sans prise → « Non mesurable : <camp> n'a pris aucune arme spéciale
  (<a> contre <b>) ».

Contrats neufs (Go, `internal/domain/match_emprise.go`) — forme figée, noms définitifs au lot :

```go
// MatchViewEmpriseFields — embarqué dans MatchViewResponse (D20), champs aplatis.
type MatchViewEmpriseFields struct {
    Emprise           *MatchEmpriseBlock      `json:"emprise,omitempty"`
    LivesNearTeammate *MatchLivesNearTeammate `json:"lives_near_teammate,omitempty"`
}
// MatchEmpriseBlock — l'Emprise d'UN match, composition = mon camp (D2, D3).
type MatchEmpriseBlock struct {
    SquadEmpriseBlock                       // Matches a un seul élément
    KillJournalPublishable bool `json:"kill_journal_publishable"` // D7
}
// MatchLivesNearTeammate — « Isolement », par joueur de l'équipe (D8 ; journal non publiable : A1).
type MatchLivesNearTeammate struct {
    Players []MatchLivesPlayer `json:"players"` // ordre de D3
}
type MatchLivesPlayer struct {
    XUID string `json:"xuid"`
    TimeseriesLivesNearTeammate      // Near, Alone, ExcludedUnlocated, ExcludedNoRadar, …
}
// domain/squad_emprise.go — SquadEmpriseMatch (D6)
UnclassifiedPickups *SquadEmpriseCount `json:"unclassified_pickups,omitempty"`
// domain/match_view.go — MatchCombatTab (D9)
WeaponTools *SquadWeaponTools `json:"weapon_tools,omitempty"`
```

## 4. Inventaire des suppressions — preuves par grep (relevées le 2026-10-06, à REJOUER avant de supprimer)

Chaque preuve se rejoue par `Grep` (outil) sur `apps/web/src` (hors `lib/api/generated.ts`) ou
`apps/go-api/internal`. Attendu APRÈS suppression : 0 occurrence hors fichiers supprimés et hors
commentaires historiques (corrigés s'ils deviennent faux, règle 17). Juge de paix web :
`node tools/knip-ratchet.mjs` 0 / 0 / 0.

- **A. Hauteur d'engagement (web).** `MatchElevationSection.tsx` (219 L), `_elevation.ts` (+ test) :
  seul lecteur `MatchViewTabArsenal.tsx:56,170-175` ; prop `elevation` (`MatchViewTabArsenal.tsx:76,119`,
  `MatchViewPage.tsx:424`) ; clés `elevation*` de `MatchViewText` (`match-view/i18n.ts:120-143`
  type, 489-507 FR, 850-867 EN) ; alias `MatchElevationBlock` / `MatchElevationKill`
  (`lib/api/types.ts:1975,1983,1990`, retirés en M4 avec le contrat).
- **B. Riposte (web).** `MatchRiposteSection.tsx` (308 L, + test), `_riposte.ts` (+ test) : seul
  lecteur `MatchViewTabPlayers.tsx:16,29,42-43,61,91-100` ; prop `riposte` (`MatchViewPage.tsx:442`,
  `MatchViewTabPlayers.test.tsx:49-50`) ; clés `riposte*` (`match-view/i18n.ts:245-261` type, 573-586
  FR, 932-945 EN) ; alias `MatchRiposteBlock`, `MatchRiposteDeath`, `MatchRiposteePlayer`
  (`lib/api/types.ts:1965-1969,1993-2004`, M4). `squad/squadRiposte*` lit un AUTRE bloc
  (`squad_echange`) : RESTE.
- **C. Contrôle des armes spéciales (web, A7).** `match-replay/MatchPadControlSection.tsx` (383 L,
  + test 471 L) : seuls lecteurs `MatchViewTabArsenal.tsx:40,42,104-106,188-194` ;
  `model/padControlLogic.ts` (+ test), `model/padControlChart.ts` (+ test),
  `model/padControlColumns.ts` (+ test) : lus par la section et entre eux seulement ;
  `model/weaponTier.ts` (+ test) : lu par `padControlLogic.ts:45` et par le TYPE `PadTier` de
  `i18n/i18nContract.ts:14,155,164` (bloc `PadControlText`) — sort si knip le dit après le retrait de
  `PadControlText` ; bloc `padControl` de `REPLAY_TEXT` (`match-replay/i18n/i18n.ts:401`, 852) et son
  type (`i18nContract.ts:1033`) ; cas `goFixtures.contract.test.ts:67,213` (le contrat des fixtures Go
  ne doit plus exercer un modèle supprimé) ; dérogations d'import devenues mortes
  `tools/lint-cross-feature-imports.mjs:108` (`match-view=>match-replay/MatchPadControlSection`) et
  `:124` (`…/model/padControlLogic`) ; mentions de `match-replay/README.md`.
- **D. « Part de chaque équipe » (web, A6).** `UsageFamilyTracks` (`MatchEquipmentUsageSection.tsx:234-298`),
  son montage (l. 160-170, 205-215) et la grille à deux colonnes (l. 185) ; `buildUsageFamilyBars`,
  `orderedTeams` et leurs types (`model/equipmentUsageChart.ts:147-256`) et leur cas de test
  (`equipmentUsageChart.test.ts:170-230`) ; clés `viewTeamShare`, `shareTipFmt`
  (`i18n/i18n.ts:378-380`, 829-831 ; `i18nContract.ts:58-73`) et cas de test qui les citent
  (`MatchEquipmentUsageSection.test.tsx:164,261-264,418`). `StackedTrack` RESTE (lu par l'Explorateur).
- **E. Détail des frags dans `MatchFragCard` (web).** Imports `FragWeaponBreakdown`,
  `buildFragDetailBreakdown`, `normalizeFragWeapons` de `MatchFragCard.tsx:27-28,34,53-57` et de
  `blockPredicates.ts:21,31-41,52-74` ; `normalizeFragWeapons` et `COUNT_ONLY_LABELS` sortent s'ils
  n'ont plus de lecteur ; `FragWeaponBreakdown` et `buildFragDetailBreakdown` RESTENT (Synthèse,
  Séries temporelles) ; mock de `MatchFragCard.test.tsx:33-37` adapté.
- **F. Riposte (Go + contrat).** `service/match_view_builders_riposte.go` (191 L, + test) : appel
  unique `match_view_data_loaders.go:434-438` ; `campsDuScoreboard`, `nomsDuScoreboard`, `nomOuRepli`,
  `riposteMatchKey` n'ont pas d'autre lecteur (grep : ce fichier seul) ; champ `MatchCombatTab.Riposte`
  (`domain/match_view.go:485-491`) ; types `MatchRiposteDeath`, `MatchRiposteePlayer`,
  `MatchRiposteBlock` (`domain/coordination_block.go:211-262`) et la ligne d'en-tête qui les cite
  (l. 14). `coordination.Echanges` et `FenetreEchangeMs` RESTENT (Escouade). Q20 RESTE (P3).
- **G. Hauteur d'engagement (Go + contrat).** `domain/match_elevation.go` (100 L),
  `analysis/match_elevation.go` (+ test), `service/match_view_elevation_test.go` et `viewerKillCount`
  (`match_view_builders_combat.go:457-468`, seul lecteur la hauteur) ; `LoadMatchElevation`
  (`port/repository_data.go:188-195`, `platform/duckdb/kill_distance_repo_elevation.go` + test) ;
  chargement et champ (`match_view_data_loaders.go:83-86,241-245,529-534`) ; champ
  `MatchCombatTab.Elevation` (`domain/match_view.go:521-527`). `analysis.signedElevation`,
  `MeasuredKill`, `Side`, `percentileLinear` RESTENT (`weapon_range.go`, profils de portée).
- **H. Bouts devenus morts.** Snapshot `lib/api/contract-surface.snapshot.json` régénéré par la
  procédure (`UPDATE_CONTRACT_SURFACE=1`), disparitions listées au journal du lot ;
  `REFERENCE_CANAUX_EQUIPEMENT` §4 (l. 280-281 socles de la vue match, l. 481 `weaponTier.ts`).

## 5. Organisation et gates communs

### 5.1 Frontières de fichiers

- Interdits : fichiers propres aux lots TS L7 / L8 (plan TS §6) et au périmètre Sessions
  (`features/session-detail/`, `service/session_page*`, `domain/session_*`). Exception assumée et
  DÉPENDANTE du rebase (D1) : `domain/session_usage.go` n'est que LU (constante `PadTierUnclassified`).
- Fichiers livrés par TS L1-L6 ou Sessions S1-S3 et modifiés ici (ajouts rétro-compatibles, défauts
  inchangés, tests existants rejoués SANS modification) : `service/solo_emprise_block.go` (D2),
  `service/solo_lives_block.go` (D8), `platform/duckdb/solo_lives_repo.go` (D8),
  `squad/emprise/{emprise.logic.ts (D5), PickupSheetsCard.tsx (D4), PisteCampsForm.tsx,
  ProductionCard.tsx, YieldCard.tsx (D13)}`, `timeseries/usages/LivesNearTeammateCard.tsx` (D14),
  `analysis/squademprise/match.go` (D6), `domain/squad_emprise.go` (D6).
- Partagés, conflit attendu au rebase et résolu par RÉGÉNÉRATION : `apps/go-api/api/openapi.yaml`,
  `apps/web/src/lib/api/generated.ts`, `lib/api/contract-surface.snapshot.json` ; à hunks disjoints :
  `api/wire/registry_pages.go` (factory `MatchView`), `lib/api/types.ts`, `tools/lint-cross-feature-imports.mjs`,
  `.ai/thought_log.md` (ajout en fin), docs du lot clôture (lignes propres à la Vue match).

### 5.2 Règles d'exécution et gates

- Exécuteur seul, dans le worktree, lots SÉQUENTIELS. Aucun sous-agent, aucun push, aucun merge,
  aucun `git stash`, aucun `git add -A` (stager fichier par fichier), aucun `--no-verify`, aucun
  Python, aucune base de `data/` ouverte, aucun serveur arrêté ou relancé, une commande `go` à la
  fois, aucune commande longue sans sortie (chien de garde à 600 s : tests Go par lots de paquets).
- Environnement Go, à chaque appel PowerShell :
  `$env:Path = "C:\msys64\ucrt64\bin;$env:Path"; $env:CGO_ENABLED = "1"; $env:CC = "C:\msys64\ucrt64\bin\gcc.exe"`.
  Web : `npm ci` dans `apps/web` du worktree (node_modules réel) avant le premier gate web.
- **Gate Go** (depuis `apps/go-api`) : `go build ./...` ; `gofmt -l internal` muet ; `go vet` des
  paquets touchés ; `go test -count=1` des paquets touchés puis du module en lots couvrant tout
  `go list ./...` (cmd + contracttest + analysis + api + domain + port + archlint ; games ;
  platform + service ; sync + persist + migration ; reste de internal ; pkg + scripts + tests) ;
  `go test -tags=integration -p 1 ./internal/platform/duckdb/...` dès que `platform/duckdb` bouge
  (M1, M4) ; garde-rails nommés : `TestAucunTypeAnalysisEnCorpsHuma`, `TestNoNewSlugComparison`,
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
  agrandir au solde du plan (taille avant / après au journal de chaque lot) : `domain/match_view.go`
  878 L (+3 en M2, −14 en M4, D20), `service/match_view_service.go` 568 L (+1 en M2, D19),
  `service/match_view_data_loaders.go` 714 L (+2 à +4 en M2 pour les appels, −13 en M4),
  `api/wire/registry_pages.go` 620 L (−2, D18), `port/repository_data.go` 640 L (−8 en M4),
  `match-view/i18n.ts` 1 158 L, `match-replay/i18n/i18n.ts` 925 L et `i18nContract.ts` 1 139 L (en
  baisse en M3), `lib/api/types.ts` 3 449 L (alias : + en M3, − en M4).
- **TDD** : chaque règle neuve a son test ROUGE écrit et vu rouge AVANT le code (compilation ET
  comportement contre un bouchon) ; puis vert ; puis au moins UNE MUTATION par règle, annulée ensuite,
  consignée au journal du lot (« mutation : … → rouge »). Refactorisation sans changement de
  comportement : tests existants verts SANS modification d'assertion, et une mutation prouve qu'ils
  mordent sur le code déplacé.
- **Clôture de lot** = gate vert + items statués + section du lot mise à jour ici + entrée en FIN de
  `.ai/thought_log.md` + commit local `feat(matchview/<lot>): …` (fichiers stagés un par un) + point
  d'étape au superviseur.

## 6. Lots

### M0 — Préalable : rebase et relecture des dépendances · rapide (aucun code)

- [x] M0.1 Sur « rebase sur <sha> » : `git -C <worktree> rebase <sha>` ; conflits attendus seulement
  sur `.ai/thought_log.md` (ajouts en fin, les deux gardés) ; aucun fichier de code ne diverge (la
  branche ne porte que ce plan, la maquette, les mesures et le journal).
- [x] M0.2 Relire sur pièces chaque symbole de la table §1.3 (fichier, signature) ; mettre la table et
  les items M1-M3 à jour si un NOM a changé ; arrêt et compte rendu si une FORME diffère (D1).
- [x] M0.3 Rejouer les lignes citées par M1-M2 (le code a bougé avec L7, L8, S1-S7) et corriger les
  numéros dans ce plan ; rejouer les preuves §4 (relevé de départ, consigné au journal).
- Gate : aucun code ; `git -C <worktree> log --oneline -5` et `git status --short` propres ; le plan
  mis à jour commité (`docs(matchview): plan relu sur la base <sha>`).

Journal M0 (2026-10-07, exécuteur) :
- **M0.1** Rebase sur `2668848b1` (tête de `feat/v75`, signal du superviseur, au lieu de la tête de
  `feat/sessions-emprise` : elle la contient) sans conflit, y compris sur le journal. `go build ./...`
  vert (au second passage avec `-p 2` : le premier a échoué sur « fichier de pagination insuffisant »,
  mémoire de la machine, pas le code) ; `npm ci` (508 paquets, node_modules réel) ; `npx tsc -b --force` 0.
- **M0.2** Symboles relus : §1.3 (forme identique ; `empriseObjectName` vit dans `objectName.ts`).
- **M0.3** Écarts sur pièces consignés en §1.4 (A1 : règle des vies au journal non publiable déjà en
  place ; A4 : numéros des types `MatchRiposte*`). Les autres lignes citées par M1-M2 sont relues à
  l'ouverture de chaque item (règle 4 du contrat) : les fichiers Go de la Vue match n'ont pas bougé
  depuis la base du plan hormis `registry_pages.go` (12 lignes, factory Sessions) et
  `coordination_block.go` (riposte de la coordination retirée par Sessions). Côté web, le lot du rejeu a
  modifié `MatchEquipmentUsageSection.tsx`, `MatchPadControlSection.tsx`, les modèles d'équipement et de
  socles et `match-view/{teamColor, teamSeriesColor, xuidMeta, _momentum, MatchScoreCurveChart,
  MatchScoreEventsChart}` (allégeance lue dans le film) : §4 est rejoué en M3.9 sur ces versions.
- Consignes du superviseur intégrées : A2 (règles de texte), A3 (dette « Mes matchs » et garde étendue).

### M1 — Go : lectures et calculs (aucun contrat neuf) · moyen

Périmètre : `platform/duckdb/{kill_distance_repo.go, match_view_repo_assist_pairs.go,
solo_lives_repo.go}` (+ tests), `domain/assist_pairs.go`, `port/timeseries_lives.go`,
`service/{solo_lives_block.go, solo_emprise_block.go}` (+ tests).

- [x] M1.1 D11 : `resolveRows` écarte les clés de classe `domain.FragClassMelee`. Test `:memory:`
  écrit d'abord (`kill_distance_repo_test.go`, patron existant) : une clé de mêlée, une épée de classe
  `heavy`, un fusil — la mêlée sort, les deux autres restent, comptes et min / moy / max intacts.
  Mutation : comparaison sur `Role` au lieu de `Class` → rouge.
- [x] M1.2 D7 : Q21d `publishable_deaths` + `MatchAssistScopeRaw.PublishableDeaths` + `scanAssistPairs`.
  Test `:memory:` (patron `match_view_repo_assist_pairs_test.go`) : 3 lignes dont 1 publiable sans
  assistance connue → `MatchDeaths` 3, `MeasuredDeaths` 0, `PublishableDeaths` 1 ; match sans ligne
  → 0 partout. Mutation : `FILTER (WHERE publishable AND assist_known)` recopié → rouge.
- [x] M1.3 D8 : `port.CampLivesRepository` (`port/timeseries_lives.go`) ; `SoloLivesRepo.LoadLivesNearTeammateForPlayers`
  (quatre requêtes paramétrées par une liste de xuids, xuid rendu par les trois premières) ;
  `LoadLivesNearTeammate` délègue. Tests `:memory:` écrits d'abord (`solo_lives_repo_test.go`) :
  deux joueurs du même match séparés, joueur non demandé absent, liste vide sans requête,
  `exigerFenetresBornees` sur les trois vues `_latest` (nouveau `TestSoloLivesRepo_ParJoueurs_BorneEtDernierePasse`) ;
  `TestSoloLivesRepo_BorneEtDernierePasse` vert SANS modification. Mutations : liste des matchs en
  sous-requête → rouge ; filtre joueur `= ?` sur le premier xuid seulement → rouge.
- [x] M1.4 D8 : `lireViesDuCamp(ctx, viesCampQuery) (map[string]domain.TimeseriesLivesNearTeammate, bool)`
  (second retour : au moins une vie lue) — une lecture `LoadLivesNearTeammateForPlayers`, portée par
  `mappings.PorteesDuRadarParMatch`, `coordination.ViesPresOuSeul` par joueur, journaux `vies_*` +
  `page` (Debug capability absente, Error lecture en échec, Info bilan avec les frags écartés) ;
  `lireViesPresOuSeul` l'appelle avec un xuid et rend son bloc comme aujourd'hui. Tests des Séries
  temporelles (`timeseries_service_lives_test.go`) et de Sessions (tests de S2.4) verts SANS
  modification ; test neuf (mock enregistreur) : une lecture pour deux joueurs, un bilan par joueur.
  Mutations : un appel au repo par joueur → rouge (compteur) ; liste des joueurs ignorée (seul le
  joueur de la page lu) → rouge.
- [x] M1.5 D2 : `soloEmpriseQuery.Players` ; nil → `squadagg.SquadPlayers` comme aujourd'hui. Tests
  `timeseries_service_emprise_test.go` et ceux de Sessions S2.3 verts SANS modification ;
  test neuf : `Players` fourni → `block.Players` identique, dans l'ordre, et parts `squad` par joueur.
  Mutation : `Players` ignoré → rouge.
- Gate : gate Go (sans contrat : aucun type de réponse ne change ; `MatchAssistScopeRaw` est interne)
  + `go test -tags=integration -p 1 ./internal/platform/duckdb/...` ; ADR 0036 (EN) : nouveau test I2
  cité (`docs/adr/0036-page-reads-are-scoped.md:155-163` et tableau l. ~421).
Journal M1 (2026-10-07, exécuteur) — test rouge AVANT chaque code (compilation puis comportement) :
- **M1.1** `KillDistanceRepo.resolveRows` résout libellés et classe AVANT l'agrégation et écarte les clés de classe `domain.FragClassMelee` (une seule requête de métadonnées, comme avant). Test `TestKillDistance_MeleeExclue` (classificateur local `kdMeleeClassifier` : `hinf_unarmed` classe `melee`, `hinf_energy_sword` classe `heavy` au registre réel) vu rouge (mêlée publiée), puis vert ; les neuf tests existants verts sans modification.
- **M1.2** Q21d : `publishable_deaths` (`COUNT(*) FILTER (WHERE publishable)`), `MatchAssistScopeRaw.PublishableDeaths`, scan ; doctrine du fichier mise à jour. `TestQ21dAssistPairs_MortsPubliables` rouge de compilation puis de comportement (champ jamais rempli), puis vert.
- **M1.3** `port.CampLivesRepository` ; `SoloLivesRepo.LoadLivesNearTeammateForPlayers` (vies, morts et frags filtrés par une liste de joueurs APRÈS la fenêtre, xuid en tête de ligne ; journal non publiable et variantes communs) ; `LoadLivesNearTeammate` l'appelle avec un joueur (une seule copie des requêtes). `TestSoloLivesRepo_ParJoueurs_BorneEtDernierePasse` rouge de compilation puis vert, `exigerFenetresBornees(…, 11, 4)` ; les cinq tests existants verts SANS modification.
- **M1.4** `lireViesDuCamp` / `viesCampQuery` ; la lecture d'un joueur et celle de l'équipe partagent `viesLuesOK`, `viesCapabilityAbsente` et `bilansDesVies` (une seule copie de la portée du radar, du calcul et du journal ; le journal `vies` porte désormais `xuid` au lieu du gamertag, aucun test ne l'assertait). `solo_lives_block_test.go` (une lecture pour trois joueurs, bilan à zéro pour un joueur sans vie, dégradations) vu rouge contre un bouchon, puis vert ; tests des Séries temporelles et de Sessions verts sans modification.
- **M1.5** `soloEmpriseQuery.Players` (nil = joueur seul comme avant) ; `TestBuildSoloEmpriseBlock_JoueursFournis` (fiches dans l'ordre fourni, parts P 1 / A 2 / reste) rouge (une seule fiche), puis vert ; tests `TestAttachEmprise*` et de Sessions verts sans modification.
- **Mutations** (toutes ROUGES, restauration vérifiée octet à octet) : exclusion de la mêlée retirée ; épée lourde écartée aussi ; `publishable_deaths` filtré comme `measured_deaths` ; liste des matchs liée en semi-jointure (fenêtres non bornées) ; liste des joueurs réduite au premier ; `lireViesDuCamp` réduite au premier joueur ; « au moins une vie lue » jamais vrai ; `Players` ignoré.
- **ADR 0036** (EN) : `TestSoloLivesRepo_ParJoueurs_BorneEtDernierePasse` cité en I2 (liste et tableau).
- **Gate** : `go build ./...` 0 ; `gofmt -l internal` muet ; `go vet` de `platform/duckdb`, `domain`, `port`, `service` 0 ; tests des paquets touchés 4 ok ; module en six lots couvrant `go list ./...` : cœur 67 ok, games 39 ok, platform + service 28 ok, sync + persist + migration 12 ok + `sync/skill` rejoué seul ok (`TestLUSRV2Shadow_RafalesBornees_300Candidats` a dépassé son budget sous charge dans le lot, paquet non touché, vert seul), reste 45 ok, hors internal 4 ok — 0 FAIL ; `go test -tags=integration -p 1 ./internal/platform/duckdb/...` 4 ok (373 s) ; garde-rails nommés PASS ; `make go-api-lint` 0 issues. Contrat inchangé (`MatchAssistScopeRaw` est interne).
- Seuils : `solo_lives_repo.go` 241 L, `solo_lives_block.go` 131 L, `kill_distance_repo.go` 233 L ; plus longue fonction neuve `LoadLivesNearTeammateForPlayers` ~45 L ; `bilansDesVies` 5 paramètres (ctx compris).

### M2 — Go : les blocs de la Vue match (contrat additif) · lourd

Périmètre : `domain/{match_emprise.go (NEUF), match_view.go, squad_emprise.go}`,
`analysis/squademprise/match.go` (+ test), `service/{match_view_emprise.go, match_view_lives.go,
match_view_tools.go (NEUFS), match_view_service.go, match_view_data_loaders.go}` (+ tests),
`api/wire/{registry_pages.go, registry_pages_matchview.go (NEUF)}` (+ test de câblage), contrat.

- [x] M2.1 Types §3 (`domain/match_emprise.go`, `SquadEmpriseMatch.UnclassifiedPickups`,
  `MatchCombatTab.WeaponTools`, ligne d'embarquement D20) ; commentaires de contrat courts, au présent.
- [x] M2.2 D6 : `tallyMatch` / `publierMatch` posent `UnclassifiedPickups`. Tests ROUGES d'abord
  (`analysis/squademprise/build_test.go` ou fichier neuf `unclassified_test.go`) : témoin m2407 réduit
  (lignes `non_classe` de mon camp 19, adversaire 12 → `{19, 12}`), niveaux non établis (lignes
  `non_classe` comptées quand même), match sans film → nil, camp inconnu → nil, aucune ligne non
  classée → nil ; `build_test.go` existant vert sans modification. Mutation : lignes de l'adversaire
  comptées chez nous → rouge.
- [x] M2.3 D2, D3, D22 : `attachMatchEmprise(ctx, &resp, matchID, d, friendsExtras)` — `matchCampPlayers`
  (D3, pure) ; `buildSoloEmpriseBlock` avec `Page: "match_view"`, `Current` = un `squademprise.Match`
  (identifiant, heure de début de la méta), `WithMaps: false`, `Players` ; publié en
  `MatchEmpriseBlock{SquadEmpriseBlock, KillJournalPublishable}` ; section `match_emprise`. Appel par
  une ligne dans `buildMatchViewFromData` (après le tableau des scores et `friendsExtras`). Tests
  (mocks de port, `match_view_emprise_test.go`) écrits ROUGES d'abord : ordre D3 (moi, suivis, autres ;
  bot et parti exclus), un seul match publié, objets par joueur, journal publiable vrai / faux, Halo 5
  (repo d'usage nil → `film_unavailable = film_unsupported`, feuille seule), feuille non supportée /
  en échec, film en échec → `*_load_failed`, jamais d'erreur de page ; témoin m2209 (MESURES §3 D et E :
  bonus 5–2, râteliers 4–3, fiches JGtm / XL JACOB / Madina97294 / Chocoboflor dans l'ordre D3).
  Mutations : bot gardé en fiche → rouge ; profils suivis non remontés → rouge ; journal publiable lu
  sur `MeasuredDeaths` → rouge.
- [x] M2.4 D8, D22 : `attachMatchLives` — repo nil → Debug « capability absente », bloc absent ; une
  lecture `lireViesDuCamp` pour les joueurs de D3 ; bloc absent si aucun joueur n'a de vie lue ;
  section `match_lives` (A1 : pas de `FragsMeasured`). Tests : ordre des joueurs = D3, bilan par joueur
  (témoin m2209 à la main, portée 18 m : JGtm 11 / 2 vies, 8 / 0 frags), échec de lecture → bloc
  absent + ErrorContext, journal non publiable → vies écartées et comptées pour chaque joueur.
  Mutations : ordre du repo au lieu de D3 → rouge ; un joueur sans vie lue retiré de la liste → rouge.
- [x] M2.5 D9 : `matchWeaponTools(ctx, d, scoreboardRow)` → `squadagg.BuildWeaponTools` ; trace
  `PlayersAboveSheet` (Debug, patron `teammates_squad_weapon_tools.go:323-329`). Tests ROUGES d'abord
  (`match_view_tools_test.go`) : témoin m2209 (MESURES §3 B : MK50 Sidekick 7, Mêlée 2, Grenade frag 1,
  VK78 Commando 1) ; mêlée depuis la feuille ; objet explosif du film retiré de l'arme qu'il recouvre ;
  reliquat « Non attribué » (m2407 : 10 non attribués) ; catégories absentes → reliquat ; lignes des
  autres joueurs ignorées. Mutation : `bulkWeapons` non filtré sur le joueur → rouge.
- [x] M2.6 D18, D19 : `matchViewEmpriseDeps` (`With*` : `WithEmpriseSheet`, `WithEmpriseUsageSummary(repo,
  repoRoot)`, `WithEmpriseVehicles`, `WithCampLives`, `WithRadarRange`, `WithKillSourceCategories`) ;
  `cablerFilmMatchView` ; garde-rail `registry_pages_matchview_wiring_test.go` (patron
  `registry_pages_timeseries_wiring_test.go`, lecteur `appelsDansFactory`) : feuille et catégories
  inconditionnelles, résumé d'usage / véhicules / vies sous leur seule porte. Mutations : feuille sous
  condition → rouge ; vies hors porte (`if true`) → rouge.
- [x] M2.7 Contrat régénéré, diff ADDITIF (0 retrait dans `openapi.yaml` et `generated.ts`) ;
  `contract-surface.guard.test.ts` vert sans régénérer le snapshot ; schémas `MatchEmpriseBlock`,
  `MatchLivesNearTeammate`, `MatchLivesPlayer` aplatis comme attendu (vérifié dans `openapi.yaml`).
- Gate : gate Go + contrat ; web : `npm ci`, `generate-types`, `tsc -b --force` 0, vitest `src/lib/api`
  vert.
Journal M2 (2026-10-07, exécuteur) :
- **M2.1** `domain/match_emprise.go` (`MatchViewEmpriseFields` embarqué dans `MatchViewResponse` par une ligne, `MatchEmpriseBlock`, `MatchLivesNearTeammate` sans `FragsMeasured` — A1 —, `MatchLivesPlayer`) ; `SquadEmpriseMatch.UnclassifiedPickups` ; `MatchCombatTab.WeaponTools`.
- **M2.2** `unclassifiedOf` (`match.go`), posé par `tallyMatch` sur un match filmé au camp connu, publié par `publierMatch`. `unclassified_test.go` (témoin BTB réduit 19 / 12, niveaux non établis comptés, nil sans ligne, sans camp, sans film) vu rouge (nil) avant le code ; `build_test.go` vert inchangé.
- **M2.3** `match_view_emprise.go` : `matchViewEmpriseDeps` embarquée par une ligne dans `MatchViewService` et ses six `With*` ; `matchCampPlayers` (Q1 : le joueur de la page, les profils suivis — clés de `friendsExtras` —, le reste, chaque groupe dans l'ordre du tableau ; bots et partis sans fiche ; sans équipe : le joueur seul) ; `matchEmpriseBlock` par `buildSoloEmpriseBlock` (`Players`, `Page: match_view`, section `match_emprise`) ; posé par une ligne du littéral de réponse de `buildMatchViewFromData`.
- **M2.4** `matchLives` par `lireViesDuCamp` (section `match_lives`), un bilan par joueur dans l'ordre des fiches, à zéro sans vie lue.
- **M2.5** `match_view_tools.go` : `squadagg.BuildWeaponTools` sur les frags par arme du match déjà chargés (libellé de la requête = libellé et libellé anglais), catégories par `port.KillSourceCategoryRepository` (Debug non supportées, Warn échec), feuille = la ligne du joueur au tableau, mécaniques natives par capability ; trace `PlayersAboveSheet`. Le filtre « lignes du joueur » prévu dans la projection n'a pas été gardé : le builder ne garde déjà que les joueurs qu'on lui nomme (mutation de ce filtre VERTE, donc code mort) — la projection passe toutes les lignes.
- **M2.6** `registry_pages_matchview.go` (`cablerFilmMatchView` : distance comme avant, feuille et portée du radar sans condition, catégories par assertion sur le lecteur d'armes, résumé d'usage / vies / véhicules chacun sous sa porte) appelé par UNE ligne de la factory ; `registry_pages_matchview_wiring_test.go`.
- **M2.7** Contrat : `openapi.yaml` +115 lignes, `generated.ts` +39, 0 retrait ; schémas `MatchEmpriseBlock`, `MatchLivesNearTeammate`, `MatchLivesPlayer` et champs `emprise` / `lives_near_teammate` aplatis dans la réponse, `weapon_tools`, `unclassified_pickups` vérifiés ; `check-generated-types-fresh` OK ; `tsc -b --force` 0, vitest `src/lib/api` 5 / 36 verts (snapshot de surface intact).
- **Écart de méthode** (consigné comme en L2 du plan TS) : pour `matchEmpriseFields` et `matchWeaponTools`, le code a été écrit avant les tests ; le rouge a été obtenu en rejouant les tests contre un bouchon (bloc vide, outils nil). Les témoins chiffrés de MESURES §3 sont portés pour les outils (22/09 : Sidekick 7, mêlée 2, grenade 1, VK78 1 ; 24/07 : 10 non attribués) ; pour l'Emprise et les vies, les fixtures de service sont réduites (le témoin du 22/09 sert la fixture web de M3).
- **Mutations** (ROUGES) : adversaire compté chez nous (non classées) ; bloc vide ; outils nil ; bot gardé en fiche ; profils suivis non remontés ; journal lu sur `MeasuredDeaths` ; joueur sans vie retiré ; autre joueur nommé au builder ; catégories ignorées ; feuille sous condition ; vies hors porte.
- **Gate** : `go build ./...` 0 ; `gofmt -l internal` muet ; `go vet` des paquets touchés 0 ; tests des paquets touchés 5 ok ; module en six lots : 67 + 39 + 28 + 13 + 45 + 4 ok, 0 FAIL ; `make go-api-lint` 0 issue (un `prealloc` corrigé en cours de gate) ; garde-rails nommés PASS ; `openapi-gen -check` à jour. `-tags=integration` non requis (aucun paquet `platform/duckdb` modifié).
- Seuils : `domain/match_view.go` 878 → 881 (D20), `match_view_service.go` 568 → 569, `match_view_data_loaders.go` 714 → 716, `registry_pages.go` 614 → 612 ; fichiers neufs ≤ 174 L ; plus longue fonction neuve `matchCampPlayers` ~32 L ; `matchEmpriseFields` 5 paramètres (ctx compris).

### M3 — Web : l'onglet reconstruit, briques étendues, suppressions web · lourd

Périmètre : `features/match-view/*`, `features/match-replay/{MatchEquipmentUsageSection.tsx,
MatchPadControlSection.tsx, model/*, i18n/*, test/goFixtures.contract.test.ts, README.md}`,
`squad/emprise/{emprise.logic.ts, PickupSheetsCard.tsx, PisteCampsForm.tsx, ProductionCard.tsx,
YieldCard.tsx}`, `timeseries/usages/{LivesNearTeammateCard.tsx, LivesNearTeammateRow.tsx (NEUF)}`,
`tools/lint-cross-feature-imports.mjs`, `lib/api/types.ts` (alias). Aucune requête neuve, aucune clé
de requête neuve (`lib/query/keys.ts` non touché) : tout arrive avec la réponse de la Vue match.

RÈGLE DU LOT : les ajouts aux briques partagées sont rétro-compatibles (props optionnelles, défaut =
rendu actuel) ; les tests de page de l'Escouade, des Séries temporelles et de Sessions sont rejoués
nommément SANS modification (`SquadEmprisePage.test.tsx`, `SquadContributionsPage.test.tsx`,
`SquadObjectiveSection.test.tsx`, `SquadFragSection.test.tsx`, `TimeseriesPage.sections.test.tsx`,
`TimeseriesPage.usages.test.tsx`, `SessionColumnBody.test.tsx`, `SessionCompareRows.test.tsx`).

- [x] M3.1 Briques partagées (tests rouges d'abord pour chaque prop) : `PisteCampsRow.indent`,
  `.labelNode` (bouton de repli des râteliers) et `.pending` (texte dans une piste atténuée, `below`
  conservé) ; `ProductionCard` prop `pending` (lignes fusionnées dans l'ordre `RESOURCE_ORDER`) ;
  `YieldCard` prop `pending` ; `PickupSheetsCard` `restColor` optionnel (D4) ; `buildPickupSheets`
  paramètre `resources` (D5) ; `LivesNearTeammateRow` extraite (D14), `LivesNearTeammateCard` sur elle.
  Mutations : `pending` ignoré → rouge ; ligne indentée rendue sans retrait → rouge ; section râtelier
  absente malgré `resources` → rouge.
- [x] M3.2 `match-view/matchEmprise.logic.ts` (+ test) : modèle D (ressources de `matches[0]`, objets
  dépliés, râteliers repliés par défaut, sous-libellé « prises · N socles vidés » pour les bonus, ligne
  non identifiée D6) ; modèle E (D4, D5, sections du match, fiches dans l'ordre `emprise.players`) ;
  raisons G / H (liste fermée §3) ; modèle I (`buildLivesModel` par joueur, ordre D3, état « frags non
  mesurés ») ; couverture D16 ; LE prédicat `matchEmpriseCards(data)` (une entrée par carte B, D, E, G,
  H, I) lu par l'onglet et par chaque carte. Fixture `matchEmprise.fixtures.ts` tirée des MESURES
  (m2209, m2407). Tests ROUGES d'abord, une mutation par règle (au moins : raisons G bonus inversées,
  « 0 frag » affiché journal non publiable, ligne non identifiée oubliée, fiche du reste gardée,
  ordre des lignes I différent de E).
- [x] M3.3 `matchEmpriseText.ts` (D17) + test : titres, ⓘ, légendes et raisons FR copiés de la
  maquette (`makeTools` l. 948-951, `makeControl` l. 971-974, `makeSheets` l. 1010-1012,
  `makeEquipGrid` l. 1047-1049, `makeProd` l. 1058-1060, `makeYield` l. 1091-1093, `makeLives`
  l. 1127-1129), parité FR / EN par le typage, aucun « Notre camp » / « Our side » / « KDA ».
- [x] M3.4 Cartes : `MatchToolsCard.tsx` (B, D9) ; `MatchFragCard.tsx` (D10) ; `MatchResourceControlCard.tsx`
  (D, légende S9) ; E par `PickupSheetsCard` (identités et couleurs de `buildMatchPlayerColors`,
  initiales) ; G / H par `ProductionCard` / `YieldCard` avec `pending` ; `MatchLivesCard.tsx` (I, ⓘ
  avec le compte des vies écartées des deux causes, comme la maquette l. 1128) ; chaque carte se
  retire par le prédicat M3.2.
- [x] M3.5 `MatchViewTabArsenal.tsx` : ordre §3 et S11 ; intertitre D16 ; `MatchElevationSection`,
  `MatchPadControlSection` et leurs prédicats retirés ; props neuves (`emprise`, `livesNearTeammate`,
  `weaponTools`) passées par `MatchViewPage.tsx` (l. 414-431), `elevation` retirée. Section
  « Équipement et terrain » posée par `equipment || matchEmpriseCards… || hasPositions`.
- [x] M3.6 F (D15) : `MatchEquipmentUsageSection` réduit à la grille, titre, ⓘ, légende, mots des
  infobulles ; §4.D supprimé.
- [x] M3.7 C (D12) : `fontSize` explicite des deux styles `rich` ; test sur l'option ECharts.
- [x] M3.8 Joueurs : `MatchViewTabPlayers.tsx` sans Riposte (§4.B) ; `assistValueAxis` FR « a
  assisté… », EN « assisted… » (`match-view/i18n.ts:587`, 946) ; test de l'axe.
- [x] M3.9 Rejouer CHAQUE preuve grep de §4.A-E avant de supprimer ; écart → §8, arrêt propre si un
  lecteur inattendu existe.
- [x] M3.10 Suppressions web §4.A, §4.B, §4.C, §4.D, §4.E (fichiers, tests, clés, dérogations
  d'import mortes, cas de `goFixtures.contract.test.ts`), commentaires devenus faux corrigés.
- [x] M3.11 Paire `match-view=>timeseries` (D14) dans `ALLOWED_CROSS_IMPORTS` avec son commentaire ;
  ratchets : knip 0 / 0 / 0, imports croisés ≤ 7, aucune dérogation morte ; si un plafond baisse,
  l'abaisser.
- [x] M3.13 A3 : la garde `textesSansPersonne.test.ts` étendue à `features/match-view/` (textes et
  littéraux de phrase des sources), vue ROUGE sur « Mes matchs » / « My matches », puis
  `MatchViewPage.tsx:146,177` → « Matchs » / « Matches » (test `MatchViewPage.test.tsx:103-109` suit) ;
  verte.
- [x] M3.12 Tests de page : `MatchViewTabs.test.tsx` et `MatchViewTabPlayers.test.tsx` adaptés
  (ordre des cartes A-J, intertitres, retrait par carte, sans film, Halo 5 — seule la barre épaisse
  des armes spéciales de G si la feuille la porte —, anglais, état vide de l'onglet) ; test de page
  NEUF `MatchViewTabArsenal.test.tsx` (le fichier n'existe pas au 2026-10-06 : l'onglet n'est testé
  que par `MatchViewTabs.test.tsx`).
- Gate : gate web ; preuves §4.A-E rejouées → 0 (côté web).
Journal M3 (2026-10-07, exécuteur) :
- **M3.1** Briques : `PisteCampsRow.indent` / `.labelNode` / `.pending` (texte atténué à la place de la barre, `below` gardé) ; `ProductionCard` props `pending` (rangées dans `RESOURCE_ORDER`, barre fine de l'exposition gardée) et `notes` (ligne sous la barre : « aucune prise d'arme spéciale mesurée ») ; `YieldCard` prop `pending` ; `PickupSheetsCard.restColor` optionnel ; `buildPickupSheets(…, resources?)` ; `LivesNearTeammateRow` extraite (ligne, `idPrefix` unique par ligne) avec `livesLayout.ts` (encres et grille hors du fichier de composant, aucun avertissement `react-refresh` de plus) ; `LivesNearTeammateCard` sur elle, exporte `LivesLegend` et `LivesAxis`. Libellés des natures des outils : TROISIÈME copie évitée (`toolKindLabels` dans `squad/charts/squadFragTools.ts`, Escouade et Sessions migrées, garde `squadFragTools.labels.guard.test.ts`). Tests neufs ajoutés à côté des anciens (aucune assertion existante modifiée) : `PisteCampsForm.test.tsx`, `ProductionCard.test.tsx`, `emprise.logic.test.ts` (cas ajoutés), `YieldCard.test.tsx`, `PickupSheetsCard.test.tsx` (neufs) ; tests de page de l'Escouade, des Séries temporelles et de Sessions rejoués sans modification (suite complète verte).
- **M3.2** `matchEmprise.logic.ts` : D (`buildMatchControl` : ressource puis objets pris, bonus avec socles vidés, râteliers comptés pour le repli, ligne non identifiée sur match mesuré), E (`buildMatchSheets` : fiches de `players`, SANS fiche du reste, sections = ressources prises par l'équipe, râteliers compris), G et H (raisons fermées en clés), I (`buildMatchLives`, ordre des fiches), couverture, `matchEmpriseCards` + `hasEquipmentEmpriseCard`. Écart sur le plan : la carte B garde SON prédicat (`blockPredicates.hasWeaponTools`), retiré de `matchEmpriseCards` (deux prédicats pour la même carte divergeraient). Fixtures `matchEmprise.fixtures.ts` (22/09 Starboard, 24/07 Flood Gulch ; production recalée sur la sémantique Go : frags de la feuille sur les matchs aux niveaux mesurés). 23 tests.
- **M3.3** `matchEmpriseText.ts` : même carte → même titre (contrôle = titre « par match » de l'Escouade, fiches, frags, rendement, outils), aides récrites « sur le match » / « in the match » en une ou deux phrases, « Isolement, par joueur » ; `squad` réduit à `weaponKills` + `empty` (les seules parties lues). Test `matchEmpriseText.test.ts`.
- **M3.4** Cartes `MatchToolsCard` (B), `MatchFragCard` (A | B, survol lié retiré), `MatchResourceControlCard` (D : légende Équipe / Adversaire / 50 %, râteliers repliés, ligne non identifiée), E par `PickupSheetsCard` (palette du match, sans fiche du reste), G / H avec `pending`, `MatchLivesCard` (I) ; assemblage `useMatchEmprise.tsx`. Tests `MatchToolsCard.test.tsx`, `MatchLivesCard.test.tsx`, `MatchFragCard.test.tsx` (récrit : le survol lié et le détail par arme ont disparu, D10).
- **M3.5** Onglet : « Frags et armes » (A | B, C) puis « Équipement et terrain » avec sa couverture ; D, E, F, G | H, I, J ; `MatchElevationSection`, `MatchPadControlSection` et leurs prédicats retirés ; props `weaponTools`, `emprise`, `livesNearTeammate` passées par `MatchViewPage`.
- **M3.6** F : une seule carte « Usage d'équipements, par joueur », aide (mesure, portée, répulseur ; réserve quand elle n'est pas nulle), légende Servi / Gardé sans servir / Lâché / Tractions de grappin ; infobulles des segments sur les mêmes mots. Devenus morts avec la vue des parts et retirés : `uniqueUsageGroups`, `UsageColumnGroup.label/hint`, textes `groupEquipment`, `groupGrappleHint`, `groupEquipmentHint`. Définition de « servi » corrigée sur le code (`usageUsedOf` : activé pour un bonus, posé pour le mur, charge consommée pour le reste).
- **M3.7** `_killDistanceChart.ts` : les deux styles `rich` reprennent `axis.axisLabel.fontSize` ; test rouge (`undefined`) puis vert.
- **M3.8** Joueurs sans Riposte ; axe « a assisté… » / « assisted… » ; test de l'axe (mock du graphe qui expose `valueAxisName`).
- **M3.9** Preuves §4.A-E rejouées avant suppression : aucun lecteur inattendu (les occurrences hors liste sont des commentaires, dont deux historiques gardés : `section-card.guard.test.ts`, `noIsMeOutsideViewpoint.guard.test.ts`). Après : 0 occurrence de code.
- **M3.10** Supprimés : `MatchElevationSection.tsx`, `_elevation.ts` (+ test), `MatchRiposteSection.tsx` (+ test), `_riposte.ts` (+ test), `MatchPadControlSection.tsx` (+ test), `model/padControlLogic.ts`, `padControlChart.ts`, `padControlColumns.ts`, `weaponTier.ts` (+ tests), `PadControlText` et le bloc `padControl` (FR, EN), clés `elevation*` et `riposte*`, `buildUsageFamilyBars` / `orderedTeams` et leurs types (+ cas de test), `normalizeFragWeapons` / `COUNT_ONLY_LABELS`, cas `buildPadControl` de `goFixtures.contract.test.ts`, dérogations d'import mortes ; commentaires devenus faux corrigés (`README` du rejeu, `equipmentUsageLogic.ts`, `xuidMeta.test.ts`, `blockPredicates.ts`, `lint-cross-feature-imports.mjs`).
- **M3.11** `match-view=>timeseries` ajoutée avec son commentaire ; deux dérogations mortes retirées ; imports croisés 7 / plafond 7 (inchangé) ; knip 0 / 0 / 0.
- **M3.13** Garde `textesSansPersonne.test.ts` étendue (UNE garde) : `MATCH_VIEW_TEXT`, `MATCH_EMPRISE_TEXT`, textes de la carte F, manifeste `match_view.toml`, littéraux de phrase des sources de `features/match-view/` (hors tests, listes de classes et fragments de code), et `m'` / `t'` ajoutés aux mots refusés. Vue ROUGE (6 cas : « Mes matchs » / « My matches », « Mon équipe » / « My team », « T'a martyrisé » / « Martyred you », « You victimized them », « camp » dans quatre textes, « vous / vos » dans deux aides du manifeste, deux messages d'indisponibilité au tutoiement). Corrigés : « Matchs » / « Matches » (clé `navMatches` du dictionnaire : le lint des libellés de champ refuse le littéral dans la page), « Équipe » / « Team », « A fragué le joueur n fois » / « Fragué par le joueur n fois » (EN « Killed the player n times » / « Killed by the player n times »), « équipe » au lieu de « camp » (score à manches, score décodé, porteurs de bombe, positions « Équipe A / B »), aides Rendement et Résistance récrites en mesure et portée, messages d'indisponibilité au joueur de la page / compte connecté. Verte.
- **M3.12** `MatchViewTabs.test.tsx` adapté (cartes de l'Emprise mockées par leur présence, ordre D-J) ; `MatchViewTabPlayers.test.tsx` (sans Riposte, axe) ; NEUF `MatchViewTabArsenal.test.tsx` sur les vrais modèles : ordre des sept cartes, couverture, rangée G | H, repli des râteliers, objets en retrait, ligne non identifiée, raisons du 24/07, ordre d'« Isolement », retrait par carte, Halo 5 (feuille seule : barre des armes spéciales seule, « sans film »), anglais, état vide.
- **Écart de méthode** (comme en M2) : le code de M3 a été écrit avant ses tests ; le rouge a été obtenu par mutation sur le code livré. **Mutations** (toutes ROUGES, restauration vérifiée octet à octet) : raisons G du bonus inversées ; « aucun frag » affiché journal non publiable ; ligne non identifiée oubliée ; fiche du reste gardée ; ordre des lignes I = ordre du repo ; `pending` ignoré (piste, frags) ; retrait de l'objet supprimé ; `resources` ignoré ; `pending` du rendement ignoré ; entrée « reste » toujours posée ; `notes` ignorées ; barre fine des frags retirée de la ligne extraite (tests des Séries temporelles) ; axe des assistances revenu à l'ancien texte ; râteliers jamais repliés ; couverture absente ; copie des libellés des outils réintroduite (garde) ; « Mes matchs » réintroduit (garde). Rouge vu avant code : `fontSize` (D12), garde étendue.
- **Gate** : `tsc -b --force` 0 ; `npm run lint` 0 erreur, 26 avertissements (aucun neuf) ; vitest complet 862 fichiers / 9 096 tests verts ; knip 0 / 0 / 0 ; couleurs 0 ; imports croisés 7 ≤ 7, aucune dérogation morte ; `lefthook run pre-push` vert (dont `lint-no-hardcoded-fields`, rouge au premier passage sur « Matchs » littéral, corrigé par `navMatches`).
- Seuils : fichiers neufs ≤ 299 L (`matchEmprise.logic.ts`) ; `match-view/i18n.ts` 1 150 → 1 042, `match-replay/i18n/i18n.ts` 918 → 852, `i18nContract.ts` 1 140 → 1 031 (lignes non vides) ; `lib/api/types.ts` + 9 (alias, retirés en M4 pour la riposte et la hauteur).

### M4 — Go : suppressions et contrat · moyen

Le web ne lit plus `combat_tab.riposte` ni `combat_tab.elevation` depuis M3.

- [x] M4.1 Rejouer les preuves §4.F et §4.G (producteurs et lecteurs Go et web).
- [x] M4.2 §4.F : chaîne Riposte de la Vue match, types et champ ; tests supprimés avec leur code.
- [x] M4.3 §4.G : chaîne de la hauteur (domaine, calcul, port, lecteur DuckDB, chargement, champ) ;
  tests supprimés avec leur code ; `viewerKillCount` si plus lu.
- [x] M4.4 Contrat régénéré ; snapshot `contract-surface` régénéré par la procédure, disparitions
  listées (attendu : `MatchElevationBlock`, `MatchElevationKill`, `MatchRiposteBlock`,
  `MatchRiposteDeath`, `MatchRiposteePlayer`, et ce que la régénération révèle en plus) ; alias de
  `lib/api/types.ts` (§4.A, §4.B) retirés.
- Gate : gate Go + `-tags=integration -p 1 ./internal/platform/duckdb/...` + contrat + gate web ;
  preuves §4.A-G rejouées → 0.
Journal M4 (2026-10-07, exécuteur) :
- **M4.1** Preuves §4.F et §4.G rejouées : la riposte de la Vue match n'avait qu'un appel (`match_view_data_loaders.go`) et ses aides (`campsDuScoreboard`, `nomsDuScoreboard`, `nomOuRepli`, `riposteMatchKey`, `mortsRiposte`, `joueursRiposte`) aucun autre lecteur ; la hauteur : un chargement, un calcul, un lecteur DuckDB, `viewerKillCount` lu par elle seule ; aucun double de test n'implémentait `LoadMatchElevation`. `coordination.Echanges` et `FenetreEchangeMs` restent (Escouade, Tactique), `analysis.signedElevation` et la portée restent (`weapon_range.go`).
- **M4.2** Supprimés : `service/match_view_builders_riposte.go` (+ test), le champ `MatchCombatTab.Riposte`, les types `MatchRiposteDeath`, `MatchRiposteePlayer`, `MatchRiposteBlock` et la ligne d'en-tête qui les citait (`domain/coordination_block.go`).
- **M4.3** Supprimés : `domain/match_elevation.go`, `analysis/match_elevation.go` (+ test), `platform/duckdb/kill_distance_repo_elevation.go` (+ test), `port.KillDistanceRepository.LoadMatchElevation`, le chargement `kill_elevation` et le champ `elevationKills`, le champ `MatchCombatTab.Elevation`, `viewerKillCount` (+ `service/match_view_elevation_test.go`). Le commentaire de `KillDistanceByWeapon` dit désormais la mêlée exclue (M1, D11).
- **M4.4** Contrat : `openapi.yaml` −138 lignes, `generated.ts` −57, 0 ajout ; `openapi-gen -check` à jour, `check-generated-types-fresh` OK. Snapshot `contract-surface` régénéré par la procédure : disparitions `MatchElevationBlock`, `MatchElevationKill`, `MatchRiposteBlock`, `MatchRiposteDeath`, `MatchRiposteePlayer` (les cinq attendues) ; la régénération ajoute les trois schémas de M2 (`MatchEmpriseBlock`, `MatchLivesNearTeammate`, `MatchLivesPlayer`) et, venus de la base `feat/v75` non encore photographiés, la route `/players/{player_slug}/pages/trends`, `postTrendsPage` et quinze schémas `Trends*`. Alias web retirés (`MatchElevationBlock`, `MatchElevationKill`, `MatchRiposteBlock`, `MatchRiposteDeath`, `MatchRiposteePlayer`, champs `riposte` / `elevation` de `MatchCombatTab`).
- **Gate** : `go build ./...` 0 ; `gofmt -l internal` muet (un alignement de champ corrigé) ; `go vet` des paquets touchés 0 ; tests des paquets touchés 6 ok ; module en six lots : 67 + 39 + 28 + 13 + 45 + 4 ok, 0 FAIL ; `go test -tags=integration -p 1 ./internal/platform/duckdb/...` 4 ok ; garde-rails nommés PASS ; `make go-api-lint` 0 issue ; contrat à jour ; web : `tsc -b --force` 0, vitest complet 862 fichiers / 9 096 tests verts, knip 0 / 0 / 0. Preuves §4.A-G rejouées : 0 occurrence de code (restent les commentaires historiques notés en M3, le contrat ancien des CHANGELOG traité en M5).
- Seuils : `domain/match_view.go` 881 → 866, `match_view_data_loaders.go` 716 → 695, `port/repository_data.go` 640 → 631, `coordination_block.go` 212 → 154, `match_view_builders_combat.go` 468 → 455.
- Écart d'ordre : les phrases de `docs/RELEASE_NOTES.md` (EN, FR) de M5.1 ont été préparées pendant l'attente du test d'intégration, laissées hors du commit de M4.

### M5 — Clôture · rapide

- [x] M5.1 Docs : `docs/CHANGELOG.md` + `docs/FR/CHANGELOG.md` (bloc `[7.5.0]` : phrases propres à la
  Vue match relevées EN l. 53 — onglet « Weapons and terrain » avec hauteur, contrôle des socles, part
  de chaque équipe — et les lignes riposte / hauteur l. 44-45 si elles citent la Vue match, corrigées ;
  entrée ajoutée) ; `docs/RELEASE_NOTES.md` + `docs/FR/RELEASE_NOTES.md` (bloc 7.5 : EN l. 32, 33,
  69 ; FR l. 32, 70) ; lignes re-vérifiées au moment d'écrire (L8 et S7 les auront touchées), FR aux
  lignes homologues ; aucune phrase propre aux Séries temporelles ni à Sessions touchée.
- [x] M5.2 `.ai/V7.5/REFERENCE_CANAUX_EQUIPEMENT_2026-09-09.md` §4 (lecteurs de la Vue match
  l. 263-281 : socles par l'Emprise du match, `MatchPadControlSection` supprimé) et tableau l. 481
  (`weaponTier.ts` s'il est supprimé).
- [x] M5.3 ADR 0036 (fait en M1, vérifié ici) ; aucune ADR neuve.
- [x] M5.4 Statut de chaque item du plan ; §8 Découvertes relues ; entrée finale du journal.
- [~] M5.5 (demandée au superviseur dans le compte rendu de clôture) Revue adversariale du diff cumulé : à demander au SUPERVISEUR (l'exécuteur n'a pas de
  sous-agent) — lots à risque : M1 (lecture bornée multi-joueurs, Q21d), M2 (agrégats, ordre D3),
  M4 (contrat).
- Gate : gate Go complet + gate web complet + contrat, rejoués après les docs.
Journal M5 (2026-10-07, exécuteur) :
- **M5.1** `docs/CHANGELOG.md` + `docs/FR/CHANGELOG.md` (bloc `[7.5.0]`) : entrée « The match view reads the Map control of its match » / « La fiche du match lit l'Emprise de son match » ajoutée après celle de Sessions ; phrases devenues fausses corrigées (niveaux d'arme : `weaponTier.ts` web supprimé ; coordination : plus de riposte sur la fiche du match ; hauteur d'engagement : plus sur la fiche du match ; « Match view reorganised » sans hauteur, socles ni part de chaque équipe). `docs/RELEASE_NOTES.md` + FR (bloc 7.5) : riposte « lue sur l'Escouade », hauteur sans la fiche du match, rubrique « Fiche du match » récrite (distance par arme sans mêlée, onglet qui lit l'Emprise du match), lignes « contrôle des armes spéciales par niveau », « les tournants d'abord » (repli retiré le 2026-09-19) et « la riposte par joueur » retirées. Aucune phrase propre aux Séries temporelles ni à Sessions touchée.
- **M5.2** `REFERENCE_CANAUX_EQUIPEMENT` : en-tête daté, lecteurs de la Vue match (onglet renommé, socles lus dans l'Emprise du match, `weaponTiers` sans lecteur web), ligne `weaponTier.ts` du tableau retirée.
- **M5.3** ADR 0036 : `TestSoloLivesRepo_ParJoueurs_BorneEtDernierePasse` cité (liste I2 et tableau), vérifié ; aucune ADR neuve.
- **M5.4** Statuts : M0-M5 statués, M5.5 `[~]` ; §8 relue (six découvertes de M3 ajoutées, aucune traitée).
- **M5.5** Revue adversariale du diff cumulé (`833ec8e15..` tête) demandée au superviseur : lots à risque M1 (lecture bornée multi-joueurs, Q21d), M2 (agrégats, ordre des fiches), M4 (contrat).
- **Gate** : aucun fichier de code n'a changé depuis le gate de M4 (documents seulement) ; rejoués après les docs : `openapi-gen -check` à jour, `check-generated-types-fresh` OK, `lefthook run pre-push` vert ; suites Go (six lots, intégration DuckDB) et web (862 fichiers) vertes sur l'arbre de code identique de M4.

### M6 — Constats de la revue adversariale, ronde 1 (2668848b1..815d99657) · rapide

Contrainte machine (re-cuisson du parc en cours) : ni `go build ./...` ni `go test ./...` sur le module ;
vet et tests des paquets touchés, vitest ciblé.

- [x] M6.1 R1 (règle 7) : `combat_tab.weapon_kills` sans lecteur depuis M3 — construction retirée de
  `buildCombatTabFull` (paramètre `bulkWeapons` retiré, trois appelants), type `MatchWeaponKill` et champ
  `MatchCombatTab.WeaponKills`, test `TestBuildCombatTabFull_ExcludesNonCombatFromWeaponBreakdown`, clé
  de la fixture dorée `match_view_slayer.json`, interface et champ web (`lib/api/types.ts`), commentaire
  de `match_view_raw.go`, entrée `MatchWeaponKill` de `BASELINE_COLLISIONS` (`tools/lint-contract-ratchet.mjs`, refusée par le crochet de pré-envoi au premier push). Grep : aucun symbole restant (`PlayerWeaponKillRow.weapon_kills` du tableau des
  scores est une autre donnée, lue par `PlayerDetailPanel`).
- [x] M6.2 R2 (règles 7 et 17) : `kill_measured.go` — champs `victimGT` / `killerGT`, colonnes
  `victim_gamertag` / `feed_killer_gamertag` et affectation retirés ; `victimXUID` gardé (profils de
  portée côté victime), commentaire récrit.
- [x] M6.3 R3 (résultat faux sur chemin dégradé) : le booléen `kill_journal_publishable` devient l'état
  `kill_journal` à trois valeurs (`publishable` / `not_publishable` / `unavailable`, constantes
  `domain.MatchKillJournal*`, enum au contrat). `unavailable` quand la lecture Q21d échoue
  (`matchViewData.assistScopeFailed`, posé par le chargeur), décidé en un point (`killJournalState`).
  Web : « Frags par ressource » et « Rendement par ressource » écrivent « Non mesuré : lecture
  indisponible » / « Not measured: reading unavailable », jamais la cause « non publiable ».
- Tests : Go `TestMatchEmprise_JournalIndisponibleSurEchecDuChargeur` (chargeur en échec → indisponible ;
  lecture réussie sans mort publiable → non publiable), tests existants adaptés au nouvel état ; web
  `matchEmprise.logic.test.ts` (deux cas) et `MatchViewTabArsenal.test.tsx` (chaque état → son texte),
  vus ROUGES avant le code. Mutations ROUGES : drapeau d'échec jamais posé ; « publiable » dès zéro mort ;
  `unavailable` rangé en « non publiable » (frags) ; raison du rendement ignorée.
- Gate : `go vet` de `domain`, `service`, `platform/duckdb`, `tests/golden` 0 ; tests de ces paquets ok ;
  `-tags=integration -run 'KillDistance|KillMeasured|WeaponRange' ./internal/platform/duckdb/` ok ;
  golangci sur les paquets touchés 0 issue ; contrat régénéré (`openapi-gen -check` à jour, types frais),
  snapshot de surface régénéré par la procédure (`MatchWeaponKill` retiré, enum de `kill_journal`
  ajouté) ; `tsc -b --force` 0 ; `npm run lint` 0 erreur (26 avertissements, aucun neuf) ; vitest
  `match-view`, `match-replay`, `lib/api` et la garde des textes : 278 fichiers verts ; knip 0 / 0 / 0 ;
  libellés de champ, couleurs, imports croisés 7 ≤ 7.
### M6.1 — Revue adversariale, ronde 2 (b9b81fc81) : R3 jusqu'au lecteur réel · rapide

- [x] M6.1.1 `platform/duckdb/match_view_repo_assist_pairs.go` : `GetMatchAssistPairs` rendait une portée
  nulle SANS erreur sur échec du lecteur partagé ou de Q21d (WARN avalé) : l'état `unavailable` de M6 ne
  pouvait donc jamais naître en production (seul le mock renvoyait l'erreur). Les deux branches rendent
  désormais `fmt.Errorf("MatchViewRepo.GetMatchAssistPairs: %w", err)`, sans WARN local (le seul
  appelant, `goLoad`, journalise et dégrade) ; commentaire de tête récrit (un échec est une erreur, une
  portée nulle d'un match sans film est un résultat) ; doc du port alignée.
- [x] M6.1.2 Test d'intégration du lecteur réel `match_view_repo_assist_pairs_errors_test.go` : contexte
  annulé → erreur enveloppant `context.Canceled`, portée nulle ; base de test au journal sans colonne
  `publishable` (schéma non migré) → erreur, portée nulle. Vu ROUGE avant la correction ; mutation
  (branche Q21d revenue à `nil`) → ROUGE sur les deux.
- [x] M6.1.3 Appelants : le service (`goLoad`, déjà journalisant) ; `noopMatchViewRepo` n'est qu'un
  contrôle de compilation (`port/repository.go:78`), son contrat (portée nulle, pas d'erreur) est gardé.
- Gate : `go vet` (dont `-tags=integration`) de `platform/duckdb`, `port`, `service`, `api/wire` 0 ;
  `-tags=integration -p 1 -run 'Assist|MatchView' ./internal/platform/duckdb/` ok ; tests unitaires de
  `platform/duckdb`, `service`, `api/wire`, `port` ok ; golangci des paquets touchés 0 issue ;
  `openapi-gen -check` à jour (contrat inchangé) ; web non touché.
## 7. Reprise de session

Relire le skill `plan-execution`, puis ce fichier (cases, journaux de lot), puis les dernières
entrées de `.ai/thought_log.md` du worktree et `git -C <worktree> log --oneline -10`. Reprendre à la
première case non statuée du lot courant. Les décisions du §1 sont fermes une fois le « go » donné.

## 7 bis. Relecture plan-review (2026-10-06, phase 1)

Grille `.claude/skills/plan-review/SKILL.md`, passée sur ce fichier :
- §1 structure : objectif et critère (§0), lots ordonnés du préalable sans code (M0) aux lectures (M1),
  aux blocs et au contrat additif (M2), au web (M3), aux suppressions de contrat (M4) ; effort par lot ;
  branche nommée ; bloqueurs documentés (dépendance Sessions D1, questions §9) — OK.
- §2 couches Go : calculs dans `analysis/` (squademprise D6, coordination réutilisé D8), types dans
  `domain/`, orchestration dans `service/` (fichiers neufs), port neuf `CampLivesRepository`, aucun
  SQL hors `platform/duckdb`, aucun handler modifié (la réponse existante porte les blocs) — OK.
- §3 multi-titre : capabilities `film.usage_summary`, `film.vehicle_usage`, `film.kill_positions`,
  porte de la distance inchangée ; feuille inconditionnelle ; Halo 5 testé (M2.3, M3.12) ; aucun
  `slug ==` (`TestNoNewSlugComparison` au gate) ; mécaniques natives par `titleHasNativeKillMechanics`
  (capability existante) — OK.
- §4 adapters : lectures par repos de port, comme les blocs frères (pas de `TitleDataAdapter`) ; écart
  conforme à l'existant — OK.
- §5 tests : purs (M2.2), DuckDB `:memory:` avec fenêtres bornées (M1.1-M1.3), service avec mocks
  (M1.4, M1.5, M2.3-M2.5), câblage (M2.6), web logique et composants (M3) ; mutation par règle — OK.
  Handlers : aucun changement, aucun test httptest ajouté.
- §6 logs : dégradations journalisées (Debug capability absente, Warn catégories, Error lecture en
  échec, Info bilan des vies) — OK.
- §7 front : aucune route, aucune clé de requête ; chaînes FR + EN `Record<Locale>` ; libellés
  d'arme du Go, natures par les libellés de `squadFragTools` ; jetons seulement — OK.
- §8 livraison : gates par lot, journal par lot, dépendance externe documentée (D1, M0) — OK.
- §9 exécutabilité : périmètres fermés (listes, preuves grep, knip juge des morts web), gates à
  commandes exactes, statuts et règle « aucune case vide », ordre strict, Découvertes, reprise,
  renvoi au skill — OK.
Défauts trouvés et CORRIGÉS à la relecture : (1) la première version calculait les prises non
classées dans la Vue match à partir de l'Input de l'Emprise, que l'assemblage S1.1 ne rend pas — la
grandeur est passée au grain du match dans le bloc (D6) ; (2) la première version lisait les vies
joueur par joueur — contraire à ADR 0036 I4 — : une lecture pour le camp (D8) ; (3) la première
version laissait les fiches E sans râteliers (le bilan ne les porte pas) : D5 ; (4) les ajouts aux
briques partagées sans lecteur dans leur lot auraient fait rougir knip : ils naissent en M3 avec leur
lecteur.

## 8. Découvertes (à consigner ici, pas à traiter)

- (phase 1) Les « Vies » des Séries temporelles (et de Sessions) comptent 0 frag pendant les vies d'un
  match dont le journal des morts n'est pas publiable (`qSoloFrags` ne lit que `publishable`) : zéro
  silencieux sur ces matchs. La Vue match le dit (D7, D14) ; les deux autres pages ne sont pas
  touchées (hors périmètre).
- (phase 1) Q20 (`GetMatchKVPairs`, `platform/duckdb/match_view_repo_extras.go:130-180`) lit le journal
  sans filtre `publishable` (constat MESURES §3 / thought_log : 32 « morts vengées » sur 294 lignes non
  publiables) ; ses lecteurs restants (tug-of-war, courbe FDA, victime du fil, antagonistes) héritent
  du même défaut. Non traité.
- (phase 1) Le bilan `resources` de l'Emprise ne porte pas les râteliers (`build.go:94-116`) : la
  carte « Contrôle des ressources » de l'Escouade et des Séries temporelles n'a donc pas de piste
  râtelier, alors que le brief TS parlait de « râteliers repliés » sur cette carte. Non traité.
- (phase 1) `domain.TimeseriesLivesNearTeammate`, `soloEmpriseQuery` portent « timeseries » / « solo »
  dans leur nom et servent aussi la Vue match (un camp) : renommage non fait (bruit de contrat).
- (phase 1) La feuille de match (`LoadPowerWeaponKills`) et le tableau des scores (Q12) relisent la
  même colonne `power_weapon_kills` dans une requête de la Vue match ; deux lectures distinctes,
  bornées au match, conservées (fusion hors périmètre).
- (M3) Après le retrait du contrôle des armes spéciales, plus aucun lecteur web du bloc `weaponTiers`
  du document de rejeu (seul `replayDocumentSchema.ts` le déclare ; le Go le produit). Non traité :
  le contrat du rejeu n'est pas dans le périmètre (à statuer avec le lot du rejeu).
- (M3) `combat_tab.weapon_kills` n'a plus de lecteur web (le détail par arme de `MatchFragCard` est
  parti, D10) ; ni §4.F ni §4.G ne le citent, le Go le produit toujours. Non traité.
- (M3) Textes du rejeu sans lecteur, antérieurs au lot : `equipmentUsage.title`,
  `equipmentUsage.outcomeTotalTakenFmt`. Non traités.
- (M3) `MatchCadenceChart.tsx:99,192` écrit « MA » (moyenne mobile, sigle anglais) devant
  `combatTeamLabel` : avec « Équipe », la légende se lit « MA Équipe ». Non traité (carte hors lot).
- (M3) `notSyncedDescription` (Vue match) tutoie à l'impératif (« reviens », « Vérifie ») sans pronom :
  la garde ne le voit pas. Non traité.
- (M3) Le jeu de textes de l'Escouade (`getSquadText`) porte des phrases à la personne et « camp »
  (choix des coéquipiers « your synergies », aide du score « au camp qui a perdu », aides Rendement /
  Résistance « vous ») : vues par la garde tant que la Vue match embarquait tout le jeu, hors Vue match
  depuis qu'elle n'en garde que `weaponKills` et `empty`. Non traité (page Escouade).

## 9. Questions au superviseur — RÉPONDUES le 2026-10-06

Réponses : Q1 ordre du brief (moi, profils suivis, reste de mon camp dans l'ordre du tableau des
scores ; même ordre pour I) ; Q2 oui ; Q3 oui (signatures existantes inchangées, tests existants
verts sans modification, une mutation par nouveau paramètre). Questions d'origine :

- **Q1** D3 : ordre des fiches E et des lignes I — le brief (« joueur de la page, profils suivis
  ensuite ») ou la maquette (ordre du tableau des scores, Madina97294 en 5e fiche sur le BTB) ? Le plan
  retient le brief.
- **Q2** D13 : D ne porte pas les lignes « en attente » de la maquette (bonus non publiés sur le BTB,
  « 0 prise », véhicules non mesurés), par cohérence avec les pages sœurs ; G et H les portent (brief).
  Accord ?
- **Q3** D2 / D8 : modification des helpers livrés par Sessions S1.1 / S1.2 (`Players` optionnel,
  lecture du camp) et de `solo_lives_repo.go` (TS L3), signatures existantes inchangées. Accord ?

Points tranchés par le code (pour mémoire, aucune décision attendue) :
- **P1** « Le bloc publie les fiches (`sheets`) » : non — les fiches sont un modèle WEB
  (`buildPickupSheets`) construit depuis `objects[].squad` ; le bloc publie bien les objets par match
  (`matches[]` à un élément, `publierMatch` avec les parts par joueur, `match.go:231-249`).
- **P2** La ligne des prises non classées n'a aucune donnée publiée aujourd'hui : D6.
- **P3** Q20 n'est pas sans lecteur une fois la Riposte retirée : elle RESTE.
- **P4** `FragWeaponBreakdown` reste lu par la Synthèse (`SynthesisPage.tsx:19,449`) et les Séries
  temporelles (`TimeseriesPage.summary.tsx:22,267`) : il RESTE.
- **P5** Les frags par vie de la maquette pour le BTB viennent de `highlight_events` (MESURES §3 I) ;
  l'app lit le journal publiable (source du brief) et dit « non mesurés » sur ce match (D7, D14).
