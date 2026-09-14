# PLAN — Ajustements pré-v7.5 (retours utilisateur du 2026-09-13)

> Créé le 2026-09-13. Base `feat/v75` @ `4acf56aae`. Contrat : skill `plan-execution`.
> Orchestration : superviseur (ce plan, worktrees, fusions, gates, CI) + un exécuteur Opus
> par lot, chacun dans SON worktree `LevelUp-wt-ajust-<lot>` / branche `feat/ajust-<lot>`.
> Briefs : `BRIEF_COMMUN.md` + `BRIEF_<LOT>.md` (scratchpad de session, copiés en fin de
> chantier sous `.ai/V7.5/briefs_ajustements_2026-09-13/`).
>
> Cadrage utilisateur (verbatim, à relire avant chaque décision) : « beaucoup de choses que
> j'ai réexpliqué 15 fois et demandé à suivre des instructions ou des modèles d'artefacts qui
> ne sont JAMAIS respectées ni suivies » ; « tu fais très attention aux rendus, j'en ai marre
> de l'improvisation, hallucination et devinettes ». Règle dérivée : quand un artefact est
> cité, il est porté fond ET forme ; tout rendu est vérifié sur capture par l'exécuteur, puis
> par le superviseur sur données réelles après intégration.

## 0. Artefacts de référence (lus intégralement le 13/09, copies dans le scratchpad)

| Artefact | URL | Sert à |
|---|---|---|
| Les formes retenues (2026-09-04) | https://claude.ai/code/artifact/2ec1b8eb-5b4d-4484-b632-c8ee91569825 | Escouade : usages d'équipement, contrôle des armes spéciales, objectifs (3 blocs, 6 formes) |
| L'échange sur la page Escouade (2026-09-08) | https://claude.ai/code/artifact/4c520da6-775b-4fb8-9d6e-dd6aa8d629b9 | Escouade : Le compte (KPI), Combien et à quelle vitesse, Qui couvre qui, Pourquoi la vengeance ne vient pas, Donné et reçu, Taux par session |
| Onglet Tactique (2026-09-08) | https://claude.ai/code/artifact/034b1915-ea1b-49d9-a4f5-06aa9fbd1ccd | Tactique : grille des cartes, analyse d'une carte, cellule, KPI, coordination d'équipe |

## 1. Lots

| Lot | Branche | Port Vite | Périmètre |
|---|---|---|---|
| A | `feat/ajust-accueil` | 5181 | Accueil + Médias : miniatures, tooltip « j'aime » |
| B | `feat/ajust-synthese-ts` | 5182 | Synthèse + Timeseries : Meilleures stats, migration Portée/Usages vers Timeseries, Activité jour×heure, Premier frag, Écart cumulé |
| C | `feat/ajust-sessions` | 5183 | Sessions : Usages d'équipement en trois blocs, parts dépliées, compact = drawer, couleurs des parts |
| D1 | `feat/ajust-escouade` | 5184 | Escouade : retraits, assistances en barres empilées, Échange (4c520da6), Perf joueur×carte, blocs scindés, Médailles en dernier |
| D2 | `feat/ajust-escouade-formes` | 5185 | Escouade : portage intégral de « Les formes retenues » (2ec1b8eb) |
| E | `feat/ajust-citations-medailles` | 5186 | Citations + Médailles : rangées pleine largeur, hauteurs alignées |
| F | `feat/ajust-tactique` | 5187 | Tactique : rechargement au clic, conformité à 034b1915 |
| G | `feat/ajust-matchview` | 5188 | Match view : Distance par arme, Carte de chaleur, Usages d'équipement, Armes spéciales, Tableau des scores, Outils de destruction |

## 2. Items (statut à la clôture de chaque lot)

### Lot A — Accueil / Médias
- [x] A.1 Les miniatures des médias s'affichent de nouveau sur l'accueil (cause trouvée sur pièces, corrigée, rendu vérifié).
- [x] A.2 Les mentions « j'aime » (qui a aimé) passent en infobulle au survol du cœur-compteur — Accueil ET Médias.

### Lot B — Synthèse / Timeseries
- [x] B.1 « Meilleures stats » : retirer Retours de drapeau, Vols de drapeau, Temps porteur (drapeau), Zones sécurisées, Temps en zone, Temps porteur (crâne).
- [x] B.2 « Portée des engagements » et « Usages d'équipement » quittent Synthèse et vont dans Timeseries (onglet choisi et justifié dans le rapport).
- [x] B.3 Portée : supprimer la phrase « À quelle distance vous fraguez… » ; « Portée par arme — mes frags et mes morts » devient « Portée par arme » ; supprimer « N armes · N frags et N morts mesurés » ; supprimer le sous-titre « Portée (i) », le (i) passe au titre du bloc ; « Dénivelé » a son propre bloc.
- [x] B.4 « Usages d'équipement » et « Ma part de l'équipement du lobby » : deux blocs distincts, même rangée. Idem « Contrôle des armes spéciales » / « Ma part des armes spéciales du lobby ». « Eux (anonyme) » remplacé par « Équipe adverse ».
- [x] B.5 « Activité par jour et heure » retrouve son format d'affichage précédent : Lundi en haut, titres d'axes, échelle verticale (wrapper étendu), ET les cases retrouvent leur couleur — cause réelle trouvée par la suite du lot : depuis `7568a7bd2` le `visualMap` classait sur la dimension `detail` (objet), aucune case n'était teintée sur les 5 consommateurs du wrapper (`dimension: 2` posé) ; le damier venait des `splitArea` croisés des deux axes ; option `emptyCells: 'hidden'` opt-in sur ce seul consommateur.
- [x] B.6 Timeseries : « Premier frag / première mort » centré verticalement dans sa carte.
- [x] B.7 Timeseries : « Écart d'engagement cumulé » à droite de « Engagement », même rangée.

### Lot C — Sessions
- [x] C.1 « Usages d'équipement » divisé en trois blocs : Cadences, Parts, Régularité match par match. **Décision utilisateur 13/09 : cadence PAR MATCH (moyenne par match mesuré), plus de normalisation par 10 minutes** — Go `per10Min` remplacé, DTO `*_per_match`.
- [x] C.2 Parts : en vue étendue, toujours dépliées (les trois jauges visibles, sans bouton).
- [x] C.3 La vue compacte (drawer) montre les mêmes éléments que la vue étendue, en compact.
- [x] C.4 « Ma part dans mon équipe » et « Ma part dans le lobby » : distinguables, forme conservée. Livré par TEXTURE (colonnes rapportées au lobby hachurées, colonne équipe pleine) : sur les grandeurs d'équipement les deux jauges portent la même pile d'issues par construction, et aucun jeton existant ne contraste à la fois avec `team-ally` et le trait de parité sur les 4 palettes hors la famille rouge (réservée à l'adversaire) — mesuré par l'exécuteur.

### Lot D1 — Escouade
- [x] D1.1 Retirer la KPI « Taux d'échange » (SquadEchangeKpi) — remplacée par « Le compte » de 4c520da6 (D1.3).
- [x] D1.2 « Assistances dans l'escouade » : barres verticales empilées (BarStackedChart, comme la match view), le tableau est supprimé.
- [x] D1.3 Échange, selon 4c520da6 : « Le compte » (4 KPI), « Combien, et à quelle vitesse » (histogramme, barres hors fenêtre hachurées, ligne fenêtre 5 s), « Qui couvre qui » (matrice avec reçu + puces), « Pourquoi la vengeance ne vient pas » (nuage : un point PAR SESSION et par joueur, gros point par joueur, quadrants nommés, légende de taille), « Donné et reçu » (barres groupées), « Taux d'échange par session » (ligne).
- [x] D1.4 « Performance par joueur × carte » : légende et étiquettes d'axes visibles.
- [~] D1.5 Usages / Ma part du lobby et Armes spéciales / Ma part du lobby : deux blocs distincts par rangée et « Équipe adverse » = lot B (composant partagé) ; page vérifiée en `mode="squad"` (« Notre part… »). « Matchs mesurés N/M » retiré des titres par I.4 (ligne de pied « Mesuré sur N matchs sur M » une fois par rangée).
- [x] D1.6 « Médailles — Résumé de l'escouade » toujours en dernière section.

### Lot D2 — Escouade, « Les formes retenues »
- [x] D2.1 Six formes (Écart à la parité, Jauge double, Piste du lobby, Grille et bande, Bâton min-max) portées en composants réutilisables, fidèles au CSS de l'artefact.
- [x] D2.2 Bloc 1 Usages d'équipements : 7 cartes (Solo B jauge double ; Solo A grille par match ; Solo B bâton ; Escouade B bande ; Escouade C piste du lobby ; Escouade A grille par coéquipier ; Escouade B piste sans parité), grenades exclues.
- [x] D2.3 Bloc 2 Contrôle des armes spéciales : 7 cartes (écart par famille solo ; jauge + bande ; taux de rafle par arme ; écart par famille escouade ; les deux frises ; emprise par match ; taux de rafle par coéquipier).
- [x] D2.4 Bloc 3 Objectifs : 5 cartes (écart par rôle ; part par famille de mode ; valeurs brutes ; rapport de force ; ce que mon camp prend).
- [x] D2.5 Données Go nécessaires (par match, par joueur du lobby, deux camps ; socles par arme) exposées sur la page Escouade, INSERT-only / lecture `_latest`, title-agnostic.

### Lot E — Citations / Médailles
- [x] E.1 Une rangée occupe toujours toute la largeur (plus de piste fantôme).
- [x] E.2 Blocs d'une même rangée alignés en hauteur.

### Lot F — Tactique
- [x] F.1 Un clic sur le plan ne recharge plus la page ; la cellule se sélectionne et son détail s'affiche (cause prouvée par reproduction).
- [x] F.2 Conformité à 034b1915, élément par élément (écran 1, écran 2, cellule, KPI, coordination d'équipe avec distance médiane et distribution des distances). Écarts assumés : seuil d'isolement = rayon radar par variante (18/24 m, les deux affichés si mélange), pas « 25 m » ; ▼ accompagné de « moins c'est mieux » (aucune période de comparaison servie) ; note « 15 premières secondes… » NON écrite : les routes sont rasterisées en passages par match, pas dessinées vie par vie (autre chantier).

### Lot G — Match view
- [x] G.1 « Distance par arme » : un seul graphe, sections par arme, joueurs sous chaque arme ; joueur actif et amis en couleur distincte.
- [x] G.2 « Carte de chaleur des positions » : refaite lisible, avec un narratif — ou retirée si aucune lecture utile n'est possible (décision justifiée).
- [x] G.3 « Usages d'équipement » : plus de grenades ; plus de distinction épisode/durée ; texte de pied retiré.
- [x] G.4 « Contrôle des armes spéciales » : une seule épaisseur de barre, déplié par défaut, étiquettes lisibles, texte de pied retiré.
- [x] G.5 « Tableau des scores » : en-têtes d'équipe aux couleurs choisies par l'utilisateur.
- [x] G.7 (ajouté 13/09 sur remarque utilisateur « on avait déjà cadré tous les noms d'armes ») « 0xD7915565 » = Mutilateur : l'entrée `hinf_mutilator` posée au registre le 10/09 n'a pas reçu son identifiant film (`0xd791556542c9679f`, connu de `games/weapons/labels.go` depuis avril) → aucune clé par famille, donc aucun libellé ni à la cuisson ni à la requête. Corriger l'entrée film, garde-rail, et compléter les libellés À LA REQUÊTE pour les artefacts déjà cuits.
- [x] G.6 Outils de destruction tronqués ; en-têtes « Rend. » et « Résist. ».

## 3. Intégration (superviseur)
- [ ] I.1 Revue sur pièces de chaque rapport (captures lues), fusion lot par lot dans `feat/v75`, gates cumulés (tsc, eslint, vitest, go test), push, CI verte au niveau job.
- [x] I.2 Passe visuelle finale sur données réelles (:8000 rebâti sur feat/v75) — 2 passes, tous les items A→G.7 + D1/D2 statués CONFORMES sur captures (dossiers FINAL et FINAL2). Défauts hors liste consignés en §4 et traités par I.5.
- [x] I.6 (perf) Le bloc `formes_retenues` pèse 1,01 Mo sur 2,96 Mo (34 %) en publiant 1 147 matchs quand les formes n'en exploitent que ~330 (mesurés ou à objectif) — D2ter fait et fusionné : 1 147 → 327 matchs publiés, bloc −14 % seulement (969 → 834 Ko), payload −4,4 % — l'estimation de 60-70 % était fausse (les matchs retirés étaient vides). Rendu identique à l'octet (13 742,56 px). Le vrai poids = 72 % en lignes d'objectif (2 632 objets répétant les noms de colonnes) → changement de contrat (valeurs alignées sur l'ordre des colonnes, ~500 Ko) consigné ci-dessous.
- [x] I.5 Corrections issues de la passe finale : (D2bis) FAIT et fusionné `b795a371c` : les lignes d'objectif n'avaient pas de camp (recollé depuis les participants, 100 % de couverture mesurée sur la réponse réelle ; camp inconnu = publié sans camp) ; nom du joueur résolu aux deux bouts ; repli des six formes par match (20 derniers mesurés, notes de pied) ; VÉRIFIÉ sur données réelles (passe 3, serveur 23:14) : écarts des deux signes par famille, segments colorés, 0 occurrence du XUID, 20 lignes/colonnes + notes de pied, page de 48 176 px à 21 728 px, 0 erreur réseau, 0 avertissement React ; (Fbis) FAIT et fusionné `6dfafd492` : trois défauts de projection du plan tactique (cellules d'index négatif jetées → 11/54 dessinées ; cadre = bbox des morts au lieu du cadre du fond 53 × 69 m ; Y non inversé) → projection sur le cadre du fond comme « Où ça se joue », 54/54 cellules sur les couloirs, vignettes calées ; « 9,9 m » ; (finitions) FAIT et fusionné `f8d482f2b` : « 1 match », état vide propre de Dénivelé, une étiquette de carte sur K au-delà de 18 cartes (12 cartes inchangé), légende des frags hors canvas.
- [x] I.3 thought_log, mémoire, retrait des worktrees (9 worktrees, 16 branches fusionnées supprimées ; un Vite orphelin du lot E arrêté).
- [x] I.4 Retirer la mention « Matchs mesurés N/M » des titres des blocs d'équipement (Escouade ET Timeseries, composant partagé `EquipmentUsageSection`) — demandé par l'utilisateur pour Escouade ; la couverture reste lisible ailleurs (pied ou infobulle du titre).

## 4. Découvertes (non traitées)

- (A) Le rail « Médias récents » de l'accueil n'apparaît qu'après ~10 s : la requête du rail ne part qu'à 5,7 s (ordonnancement des requêtes de l'accueil), contre 1,4 s sur la page Médias. Chantier accueil, hors périmètre.
- (A) Accueil : une carte de session récente imbrique un bouton dans un bouton (avertissement React).
- (A) Champ « auteur » vide sur certains médias anciens (repli sur le joueur de la page), comportement existant.
- (C) Le titre « Qui ramasse les armes spéciales » est annoncé deux fois aux outils d'accessibilité sur la session comparée.
- (C) `ValueGrid` n'avait aucun mode dense (3 graduations quelle que soit la largeur) ; un mode resserré a été ajouté pour le comparateur, les autres pages étroites qui l'emploient n'ont pas été vérifiées.
- (G) Palette joueur du match (`features/match-view/colors.ts`) : « autres alliés » et « adversaires » se croisent sur deux oranges quasi identiques (Chocoboflor / XRoachX4937) — visible partout où cette palette sert.
- (G) Arme sans nom « 0xD7915565 » dans le contrôle des socles (catalogue du film muet, repli = identifiant brut).
- (G) Chargement direct sur `?tab=chronology` : bandeau « match non chargé en totalité » et graphes vides vus une fois, non reproduit.
- (B) `dimension: 2` du `visualMap` corrige AUSSI les heatmaps Escouade (joueur × carte) et profil d'intensité, grises jusqu'ici : à regarder à la passe finale.
- (D2) Jetons imposés : camouflage et surbouclier partagent la même teinte (idem mur et autres poses) là où l'artefact en avait cinq — lignes nommées et légendées, mais lecture d'un coup d'œil moins nette ; à arbitrer (teinte dédiée au surbouclier ?).
- (D2) Au-dessus / en dessous en vert et rouge (jetons divergents) là où l'artefact voulait deux teintes neutres.
- (D2) « Drapeaux saisis » absent des cartes d'objectif : la table rôle → grandeurs du dépôt (narrative) écarte cette grandeur de « prendre » (captures, vols, aides seulement) ; l'artefact la listait — DÉCISION UTILISATEUR 13/09 : « prises nettes » (jonglage replié) mesurées depuis le film, plan `.ai/PLAN_PRISES_NETTES_DRAPEAU_2026-09-13.md`, étape 0 (mesure) lancée.
- (D2) La page Escouade met près d'une minute à répondre sur le poste (proxy Vite coupe avant) : temps de réponse à surveiller.
- (F) Unité de « Où je gagne » : l'app dit « engagements par match », la maquette « écart V moins D » (clé `tactical.unit.gagne`).
- (F) Détail de cellule : 4,5 s entre le clic et la liste sur données réelles (requête lente ; le cadre de sélection masque l'attente).
- (F) Avertissement React « two children with the same key » sur la page Ascension (hors Tactique) ; 502 intermittents du :8000 sur les lectures tactiques lourdes.
- (F) Routes de spawn rasterisées, pas en polylignes par vie comme la maquette.
- (D1) `aria.decal.show` hachure automatiquement toute série sans motif déclaré (matrice délavée depuis toujours) — neutralisé par un motif transparent dans les deux wrappers.
- (D1) Un `useMemo` à dépendances vides qui résout les couleurs du thème rend des chaînes vides au premier rendu (variables CSS pas encore posées).
- (D1) `label.color` en fonction fait disparaître toutes les étiquettes d'une heatmap ECharts.
- (D1) « Matchs mesurés 104/1147 » sur les blocs d'équipement alors que la page compte 305 matchs : deux dénominateurs sous le même mot.
- (D1) La diagonale de la matrice ne peut pas porter le tiret « — » avec `visualMap.dimension: 2` (ECharts n'étiquette pas une case non classable) : mode `emptyCells: 'blank'`, diagonale vide.
- (FINAL) Synthèse : le donut « Répartition des frags » a ~20 étiquettes de laisse empilées — DÉCISION 14/09 : ne garder sur l'anneau des armes que celles ≥ 5 %, le reste regroupé en « Autres (N armes) » par classe, sans étiquette — FAIT `00724572f`, capture : 3 étiquettes au lieu de ~20.
- (FINAL) Accueil : « Résultats de la dernière session » = une barre rouge pleine largeur sans axe ni légende, suivie d'un vide.
- (FINAL) Médailles : 404 sur `/static/medals/halo_infinite/1053114074.png` = médaille du mode VIP (table `medal_category_table.go`), absente du disque (166 autres présentes) ET du catalogue (« Inconnue ») → médaille postérieure à la dernière synchro du référentiel ; accord utilisateur 14/09 → FAIT et fusionné `f5c16bbb2` : « Clash of Kings » (VIP, nom EN depuis SpartanRecord + table des noms du film, FR vide faute de source officielle — absente du metadata.json gamecms), PNG découpé à la tuile 55 de la sprite sheet officielle (256 px, placement vérifié contre 3 tuiles voisines déjà versionnées), migration idempotente, chaîne `refresh-metadata medal-images` documentée. PNG servi (200).
- (FINAL) Tactique : ~26 avertissements React « two children with the same key » (un par carte de la grille) ; 404 sur le fond de carte `571afb7f-…` = « Cole Protocol », carte FIREFIGHT (PvE non géré, utilisateur 14/09) : matchs Firefight exclus des 4 lectures tactiques par `COALESCE(mr.is_firefight, FALSE) = FALSE` — FAIT `00724572f` (tests Go ; capture après rebuild du serveur à faire) ; calque de chaleur d'Illusion quasi vide avec des cellules hors du plan (à requalifier sur serveur à jour).
- (FINAL) Timeseries : badges orange « À VÉRIFIER » sur 6 cartes (Distribution FDA, Durée de vie moyenne, FDA, Performance solo par mois, Progression LUSR, Rang et performance, Intensité) sur données réelles — DÉCISION 14/09 : retirée — FAIT, fusionné `00724572f` (manifeste vidé, mécanisme inerte). Origine : tournée de revue visuelle « Timeseries & Escouade » du 2026-07-25 (`lib/review/chart-review.ts`, entrées jamais retirées depuis `dc33a1e9b`) ; le badge disparaît en vidant les entrées après verdict de l'utilisateur (DEC-8). Décision utilisateur attendue : clore la tournée.
- (FINAL) Timeseries « Performance solo par mois » : les deux libellés de l'axe droit se chevauchent (« Taux de victoire » / « MMR équipe »).
- (FINAL) Match view G.3 : deux lignes de pied data-driven subsistent sous « Usages d'équipement » (« 1 traction de grappin lue… », « 6 gestes mesurés sans propriétaire… ») — DÉCISION 14/09 : retirées — FAIT `00724572f` (aucun pied ; la réserve de couverture du 09/09 survit en une phrase dans l'infobulle du titre).
- (FINAL2) `/pages/teammates` : 2,2 s sans coéquipiers, 12,7 s / 3,1 Mo avec deux coéquipiers ; un 502 du proxy Vite observé (Dynamique) ; page Synergies à 48 176 px sur 427 matchs. Part du bloc `formes_retenues` à mesurer (D2bis).
- (FINAL2) 6 avertissements ECharts « Can't get DOM width or height » (graphes montés dans un conteneur de taille nulle) sur Escouade et Timeseries.
- (FINAL2) Timeseries : « Portée par arme » et « Dénivelé » en état vide sur une session de 5 matchs (seuil de 8 mesures) — forme vérifiée, rendu peuplé non vérifié ; « Ma part des armes spéciales du lobby » se retire quand 0 prise (vu peuplée sur Escouade).
- (D2ter) 72 % du bloc `formes_retenues` = lignes d'objectif sous forme d'objets répétant les noms de colonnes ; publier des valeurs alignées sur l'ordre des colonnes économiserait ~500 Ko (changement de contrat). Le temps de réponse de `/pages/teammates` (12,5 s) n'est pas expliqué par le poids : à profiler côté serveur (requêtes du bloc, formes, échange).
- (D2ter) Bandeau : « 1147 matchs » sans espace de millier (les pieds écrivent « 1 043 ») ; deux tuiles répètent leur valeur dans leur légende.
- (Fbis) Le fond d'Illusion ne couvre que 39 % de son cadre (`coveredShare: 0,386`) : larges marges vides, le plan paraît petit dans sa carte.
- (FINAL4) Page Escouade : ré-ancrage automatique sur la dernière session de la composition — FONCTION VOULUE (confirmé par l'utilisateur le 14/09), pas un défaut.
- (FINAL4) Rail de sessions : « Précédente » suit l'ordre ALPHABÉTIQUE du libellé (31/10/2025 avant 31/07/2026), pas la chronologie.
- (FINAL4) Deux contrôles portent le même libellé accessible « Session précédente » (rail de période désactivé + rail de session).
- (MÉDAILLES) Le référentiel des médailles N'EST PAS REPRODUCTIBLE : une base neuve n'a que 2 médailles nommées sur 164 (les 167 lignes locales sont un héritage Python) ; le commentaire de `ops/seed.go` (« alimentée par les migrations ») est faux ; 12 définitions ne viennent d'aucune source officielle, au moins une fausse (`2976102155` « Action Hero / Désamorçage ») ; `refresh-metadata medals` n'alimente jamais `medal_definitions` ; le proxy `/api/v1/assets/medals/{slug}/{id}/image` répond 502 pour tout id ; le chemin gamecms codé dans `platform/halo/medal_provider.go` (`Progression/...`) répond 403, le bon est `Waypoint/file/medals/metadata.json`. Chantier à part : rendre le seed reproductible depuis `metadata.json` (151 médailles officielles, fr-FR inclus).
- (CLÔTURE) Le prédicat PvE `COALESCE(x.is_firefight, FALSE) = FALSE` est recopié en 9 endroits (7 + 2) : helper + garde-rail à poser (règle ≤ 2 copies), refactor transverse.
- (CLÔTURE) Quatre champs de `EquipmentUsageCoverage` (`tracksTotal`, `grapplePulls`, `grapplePullLives`, `powerupPads`) n'ont plus de lecteur d'interface ; commentaire d'en-tête de `i18nContract.ts` cite une clé `notMeasured` disparue.
- (A) Quatre garde-fous vitest qui parcourent toute l'arborescence dépassent le délai de 5 s sur ce poste (verts à 60 s) : `lab-removal.guard`, `clockMShort.guard`, `noLocalUsageCopies.guard`, `fragClass.colorSource.guard`.

## 4bis. Contraintes d'environnement relevées par les lots

- L'API :8000 n'accepte les origines que sur les ports 5173/5174 (écritures refusées depuis un autre port) et exige une session (auth_state missing sans cookie) : les captures sur données réelles depuis un port de lot passent par une session existante du poste (lot A) ou par des fixtures (lot E). Procédure de session pour Playwright (lot A) : cookie `levelup_session` = `<session_id>.<hex(HMAC-SHA256(LEVELUP_SESSION_SECRET, session_id))>` avec un `data/sessions/<id>.json` de JGtm (`auth_ready`), injecté par `addCookies` ; origine acceptée = Vite sur 5173/5174 (sinon `page.route` qui réémet avec `Origin: http://localhost:5173`). Lecture seule, jamais de clic d'écriture. Passe finale = superviseur depuis :5174.

## 5. Journal

- 2026-09-13 — Lots A-G lancés (8 exécuteurs Opus, worktrees dédiés). Décision utilisateur reçue en cours de route : « Cadence c'est par match, pas par minutes » — transmise aux lots C et D2 (l'artefact 2ec1b8eb est amendé sur ce seul point : plus de `p10`).
- 2026-09-13 — Lot A rendu et fusionné (`7c9f915ac`). Cause des miniatures prouvée : depuis le 20/05 la vignette téléchargeait l'image hors cache (fetch + createImageBitmap), n'affichait rien avant réception complète (~800 Ko) et abandonnait en silence sur requête interrompue (20/20 interrompues à chaque chargement de l'accueil). Correctif : l'image est toujours montée, le canvas ne fait que figer la première frame. « J'aime » : infobulle portée par le bouton cœur (vignette, grille, visionneuse), libellés dans le manifeste media.toml.
- 2026-09-13 — Lot E rendu et fusionné : piste fantôme retirée de `rowGridTemplate` (une rangée = les pistes des blocs présents), `.block-row` en `align-items: stretch`, cartes `h-full flex flex-col`, contenu calé en haut. Captures 1440/1024/700 lues : rangées au bord, hauteurs égales, une colonne sous 768 px. Vérifié sur fixtures (pas de session sur :8000 depuis le port du lot) — passe données réelles à faire par le superviseur.
- 2026-09-13 — Lot C rendu et fusionné : cadences PAR MATCH (Go `per10Min` remplacé, DTO `*_per_match`, openapi régénéré), trois cartes (Cadences par match / Parts et parités / Régularité match par match), parts toujours visibles (repli supprimé), compact = même contenu resserré (plus rien de masqué en drawer), part lobby hachurée. Captures lues : les trois colonnes de jauges visibles, hachure nette, drawer complet des deux côtés. Suite vitest complète verte (7 389), Go build/vet/test verts, intégration SessionUsage verte.
- 2026-09-13 — Lot G rendu et fusionné (témoin 7fce3219, CTF Takamanohara). G.5 = régression datée : commit `3f116dfe6` du 2026-07-23 (couleur d'identité + logos) avait mis la couleur officielle du jeu avant le jeton utilisateur ; rétabli, test ajouté. G.2 : « Où ça se joue » = fond de plan + calque chaleur (grille 2 m, 100 % des 367 positions sur le plan), camps A/B (attribution spatiale, pas de nom de joueur). G.4 : repli supprimé entièrement. Captures lues : une carte Distance par arme groupée par arme avec couleur par joueur, en-têtes Eagle/Cobra aux couleurs posées dans les préférences, « Rend. »/« Résist. », « Marteau antigra… » tronqué, usages sans grenade ni pavé, socles une barre par arme lisible. Suite vitest complète verte (7 400).
- 2026-09-13 — Lot B rendu et fusionné (sans conflit) : 6 cartes d'objectif retirées (le sous-bloc garde 3 compteurs), Portée → Timeseries/Synthèse et Usages → Timeseries/Progression (DTO Go déplacé, openapi + types régénérés, champs retirés de la réponse Synthèse), Portée par arme + Dénivelé en deux cartes côte à côte, usages en 4 cartes sur 2 rangées, « Équipe adverse », wrapper Heatmap2D étendu (yAxisInverse, axisNames, visualMapOrient), Premier frag centré, Écart cumulé à droite. Captures lues. Gates : tsc 0, vitest 7 391, go build/vet/test, openapi à jour, golangci ratchet 0.
- 2026-09-13 — Gates cumulés sur feat/v75 après A+C+E+G+B : tsc 0 erreur, go build/vet OK, vitest 1er run 1 échec (fichier non capturé, dépassement de délai probable — pattern des garde-fous relevé par les lots A et E), 2e run 700 fichiers / 7 401 tests verts.
- 2026-09-13 — Lot D2 rendu et fusionné (conflit `ValueGrid.tsx` résolu à la main : mode `dense` du lot C + `axisTitle`/`hatchNotMeasured`/`nameWidth` du lot D2, une largeur explicite l'emporte sur dense) : bloc Go `formes_retenues` (lectures `_latest`, deux camps, occupations sans ramasseur branchées), 6 formes, 19 cartes, 2 contextes, cadence par match. Captures lues (19 cartes + section 1440/1024). Vérifié sur fixture « difficile » injectée sur la page réelle ; passe données réelles à faire après rebuild du serveur.
- 2026-09-13 — Suite du lot B fusionnée (`9c09b2c8d`) : heatmap Activité colorée (régression `visualMap.dimension` datée `7568a7bd2`), cases vides invisibles, libellé du donut sous l'anneau. Captures lues : rouge/vert heure par heure, plus de damier ; « 2451 » au centre, unité dessous.
- 2026-09-13 — G.7 fusionné : `hinf_mutilator` reçoit `0xd791556542c9679f` au registre film ; garde-rail `filmshell_covers_labels_test.go` (allowlist à une entrée justifiée : « Sandwich », objet Forge) ; `replay.CompleteWeaponLabels` complète à la requête les artefacts cuits avant l'entrée au registre (test sur l'artefact réel `bc60b4d9` : « Mutilateur », clé `hinf_mutilator`). Deux cliquets voisins rectifiés comme ils le prescrivaient. Rendu à vérifier à la passe finale (serveur à rebâtir).
- 2026-09-13 — Lot F rendu et fusionné (sans conflit). Bug prouvé par reproduction : le clic sur le canvas ne rechargeait PAS, le lien « ouvrir dans le rejeu » (balise native) rechargeait l'application, et le clic n'avait aucun retour visuel (panneau après 4,5 s) — d'où « rien ne change ». Corrigé : Link du routeur, cadre de sélection sur le plan. Conformité 034b1915 : bascule Grille/Analyse, mini-plans des morts sur les vignettes, ordre des questions, rampe de légende avec bornes et unité (divergente pour Où je gagne — le côté défaite était effacé avant), KPI Morts en isolement partout, section Coordination d'équipe (Go + web : médiane, distribution binnée, deux seuils 18/24 m). Captures lues ; les champs Go nouveaux vérifiés par stub + tests Go, rendu réel à la passe finale.
- 2026-09-13 — Lot D1 fusionné après D1bis (l'exécuteur a fusionné feat/v75 dans sa branche : peinture manuelle de la matrice retirée puisque `dimension: 2` la rend inutile ; légende unique sur joueur × carte ; `emptyCells: 'blank'` sur la matrice). Conflit restant résolu par le superviseur sur `HistogramChart.tsx` : union des options D1 (`binHatched`, `showValues`, `windowMark`) et F (`thresholds`), `binAttenuated` renommé `binHatched` chez le consommateur Tactique, deux `markLine` réunis en un (`reunirMarkLines`). tsc 0, vitest charts+tactical+squad 1 007 verts.
- 2026-09-13 — Passe visuelle finale, partie 1 (hors Escouade, 61 captures) : A, B.1, B.5-B.7, C, E, F.1, G.1-G.6 CONFORMES sur données réelles. INCIDENT : le serveur :8000 servait un binaire de 20:07 — air échouait à reconstruire (43 × « exit status 1 », gcc absent de son PATH ; le build passe depuis Bash), donc B.2-B.4, F.2 (Go) et G.7 non vérifiables. Air relancé à 22:31 avec msys64/ucrt64/bin en tête du PATH (Start-Process), binaire reconstruit, port 8000 OK. Passe partie 2 (items Go + Escouade) relancée. Gates locaux cumulés : vitest 704/7 467 verts, go build + tests des paquets touchés verts. CI feat/v75 : Go Linux/Windows, lint, contrat, ADR 0021, pré-déploiement verts ; Frontend et couverture en cours. Push de feat/v75 fait par une autre session (a emporté les fusions) ; coordination établie avec la session décodeur (signal « fenêtre 5 min » avant sa poussée).
- 2026-09-13 — I.4 fusionné et poussé (`7451bc920`) : les quatre cartes d'équipement (Escouade et Timeseries) n'ont plus « Matchs mesurés N/M » en titre ; une ligne de pied par rangée. Captures lues. Suite vitest complète verte (704/7 467).
- 2026-09-13 — Passe visuelle finale, partie 2 (serveur à jour) : B.2-B.4, F.2, G.7 CONFORMES ; page Escouade D1.1-D1.6 et D2.1-D2.5 CONFORMES (19 cartes, ordre des sections, Médailles en dernier). Défauts réels trouvés sur données réelles et renvoyés en correction (I.5) : objectifs des formes retenues à 0 % partout (faux), XUID brut du joueur de la page, emprise à 427 lignes, calque Tactique hors plan, arrondi, accord « 1 matchs », état vide Dénivelé, axe Carte à 56 étiquettes, légende frags Escouade.
- 2026-09-13 — CI verte sur le push I.4 (run 34781204195). Lot finitions visuelles fusionné et poussé (`f8d482f2b`) : captures lues (57 cartes lisibles, 12 cartes inchangé, légende sous le graphe, Dénivelé avec sa phrase, singulier). Restent : D2bis (objectifs à 0 %, XUID brut, emprise 427 lignes), Fbis (calque Tactique hors plan, arrondi), prises nettes (13 constats de revue).
- 2026-09-13 — D2bis fusionné et poussé (`b795a371c`). Cause du 0 % partout : les lignes d'objectif du bloc étaient publiées sans camp ; corrigé (camp des participants), test Go. Nom du joueur : résolu serveur puis filet web. Repli par match sur six formes. Passe 3 lancée sur données réelles ; D2ter (allègement du bloc) lancé.
- 2026-09-13 — Passe 3 (données réelles, serveur reconstruit sur D2bis) : objectifs des formes retenues corrects (Drapeau 101 matchs : volés 52,0 % +2,0, rapatrieurs 40,6 % −9,4… ; Crâne 75,0 % +25,0), plus de XUID brut, replis à 20, page −55 %. `/pages/teammates` toujours 12,5 s / 3,1 Mo (le repli allège le rendu, pas la charge : D2ter en cours).
- 2026-09-13 — D2ter fusionné et poussé : matchs vides retirés du bloc (327 publiés), compteurs de portée conservés, test de propriété Go (1 000 matchs vides n'altèrent que les compteurs), rendu identique.
- 2026-09-14 — Fbis fusionné et poussé (`6dfafd492`) : la chaleur tactique tombe sur le plan (preuve : 54 cellules servies / 54 peintes / 0 hors cadre sur Illusion, 75 vignettes calées). Prises nettes fusionnées (`90b79a96d`). Rattrapage `backfill-flag-grabs-net` lancé serveur arrêté.
- 2026-09-14 — Passe 4 (prises nettes sur données réelles, après rattrapage) : conforme sur Sessions et Escouade ; infobulle de cellule non mesurée corrigée par le superviseur (`GrilleForm`). CI verte sur D2bis, D2ter, prises nettes, Fbis. Worktrees retirés, docs commités. CHANTIER CLOS — restent les verdicts utilisateur du §4.
- 2026-09-14 — Décisions utilisateur : badges « À vérifier » retirés, notes de pied retirées, donut des frags ≥ 5 %, Firefight hors Tactique (Cole Protocol), niveaux d'armes lancés (étape 0 fusionnée : couverture suffisante), médaille VIP comblée et fusionnée.
- 2026-09-14 — Lot clôture fusionné et poussé (`00724572f`) : badges retirés, aucun pied sous les usages, Firefight hors Tactique, anneau des armes ≥ 5 %. Niveaux d'armes : étapes 1-2 livrées (match view par niveau, Fiesta géré) ; étape 3 tranchée par le superviseur = motif prises nettes (table append-only par passe, projection d'artefacts, CLI de rattrapage, prod à la release).
