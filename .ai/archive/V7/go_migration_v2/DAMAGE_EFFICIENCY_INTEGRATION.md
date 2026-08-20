# Damage Efficiency Integration - Go Migration

Ce document fige les decisions produit et techniques autour de la metrique issue de `.ai/damage-efficiency.md`, puis explore des integrations possibles dans la stack Go/React.

## Objectif

- Eviter de perdre les decisions prises sur cette notion d'"efficacite".
- Distinguer ce qui est exact, ce qui est un proxy, et ce qui ne doit pas etre vendu comme une verite analytique.
- Cadrer les futures integrations UI, Performance Score, LUSR et backfills associes.

## Decision de base

La metrique est viable au niveau joueur et equipe si elle est traitee comme une lecture agregee du match ou d'une fenetre de matchs.

Elle n'est pas viable comme attribution fine par duel, par cible ou par killer/victim, sauf pour un sous-ensemble particulier : les perfect kills si on les rattache de facon fiable a une paire killer/victim.

## Contraintes data a ne pas oublier

Donnees exactes actuellement disponibles :

- `match_participants` : `damage_dealt`, `damage_taken`, `kills`, `deaths`, `shots_fired`, `shots_hit`
- `medals_earned` : compte des medailles par joueur / match, dont `Perfect`
- `highlight_events` : chronologie coarse des evenements, utile pour kills/deaths, et contenant aussi des events de type `medal` avec `time_ms`
- `killer_victim_pairs` : reconstruction des paires de duel a partir des kills/deaths

Donnees non disponibles :

- degats par hit
- degats par attaquant -> victime
- repartition causale des degats dans un duel precis
- victime directement stockee dans `medals_earned`
- timestamp de medaille directement stocke dans `medals_earned`

Consequence :

- les metriques offensives et defensives par joueur sont exactes comme agregats
- les metriques d'equipe sont exactes
- on peut isoler un sous-ensemble exact de degats lies aux perfect kills : `225 * perfect_kills`
- toute repartition des degats d'equipe vers un joueur est une estimation, pas une observation
- l'attribution d'un perfect kill a une victime precise ne passe pas par `medals_earned` seule ; elle doit etre reconstruite via `highlight_events` + `killer_victim_pairs.time_ms`

## Taxonomie recommandee

Le terme `efficacite` est deja surcharge dans le codebase :

- `damage_efficiency` dans le LUSR Go = `damage_dealt / (damage_dealt + damage_taken)`
- `efficiency` dans certains calculs legacy = `(kills + assists) / deaths`
- la formule Spartan Record vise `225 * kills / damage_dealt`

Decision : ne plus utiliser `efficacite` seul comme libelle produit ou nom de champ pour cette nouvelle famille de metriques.

### Recommandation produit

Terme parapluie recommande : `rendement combat`

Sous-metriques recommandees :

- `conversion offensive` = `225 * kills / damage_dealt`
- `resistance defensive` = `damage_taken / (225 * deaths)`

Notes :

- `conversion offensive` garde l'intuition "combien de degats faut-il pour convertir un kill"
- `resistance defensive` est l'inverse, volontairement plus lisible que le raw Spartan Record `225 * deaths / damage_taken`
- les deux vont dans le meme sens de lecture produit : plus haut = mieux

### Recommandation technique

Noms preferes pour les futurs champs / variables :

- `offensive_conversion`
- `defensive_resistance`
- `team_offensive_conversion`
- `team_defensive_resistance`
- `combat_yield_delta`

Renommage recommande si on touche le LUSR :

- ancien `damage_efficiency` -> `damage_balance`
- ou ancien `damage_efficiency` -> `damage_share`

Le champ actuel `damage_efficiency` expose deja une semantique ambiguë dans les contrats Match View. Il faut eviter de le reutiliser silencieusement pour une autre formule.

## Formules retenues

### 1. Metrique offensive exacte

`offensive_conversion = 225 * kills / damage_dealt`

Interpretation :

- 1.00 = 225 degats par kill en moyenne
- > 1.00 = beaucoup de finitions sur cibles deja faibles ou tres bonne conversion
- < 1.00 = davantage de degats depenses par kill

Cas limites :

- si `damage_dealt <= 0` ou `kills <= 0`, retourner `0`

### 1.b Affinage partiel via perfect kills

Si `P = perfect_kills`, on connait une partie exacte de la conversion offensive :

- `perfect_damage_exact = 225 * P`
- `perfect_conversion_share = (225 * P) / damage_dealt`

On peut ensuite isoler le residuel non-perfect :

`offensive_conversion_residual = 225 * (kills - P) / max(damage_dealt - 225 * P, eps)`

Ce que cet affinage apporte :

- on separe la partie exacte de la partie incertaine
- on evite de faire comme si tous les kills etaient egalement "propres"
- on peut comparer les joueurs qui convertissent via beaucoup de perfects a ceux qui convertissent surtout via cleanup / chip damage / assist pressure

Ce que cet affinage ne resout pas :

- le residuel reste un melange de non-perfect kills, assists implicites, damage sans kill et cleanup
- la formule residuelle reste une lecture agregee, pas une attribution causale par duel

Si la jointure temporelle medal -> kill est validee, on peut aussi construire une composante paire exacte :

`exact_pair_damage_ij = 225 * perfect_kills_ij`

Autrement dit, les perfect kills n'annulent pas l'incertitude globale, mais ils fournissent une ancre exacte pour un sous-ensemble des degats et des kills.

### 2. Metrique defensive player-facing exacte

`defensive_resistance = damage_taken / (225 * deaths)`

Interpretation :

- 1.00 = 225 degats encaisses par mort en moyenne
- > 1.00 = il faut plus qu'un perfect kill theorique pour te tuer en moyenne
- < 1.00 = tu meurs avec moins de marge defensive

Cas limites :

- si `damage_taken <= 0` ou `deaths <= 0`, retourner `0`

### 2.b Affinage partiel cote defensif

Si `Q = perfect_deaths_against_player`, alors on connait aussi une composante exacte cote defense :

- `perfect_damage_taken_exact = 225 * Q`
- `defensive_resistance_residual = max(damage_taken - 225 * Q, 0) / max(225 * (deaths - Q), eps)`

La logique est strictement symetrique : on separe les morts parfaitement converties par l'adversaire des autres morts plus ambiguës.

### 3. Variante raw a garder en reference, pas en priorite UI

`enemy_efficiency_raw = 225 * deaths / damage_taken`

Cette variante reste utile pour comparer LevelUp a Spartan Record ou pour debug. Elle est moins lisible en produit car plus bas = mieux pour le joueur.

## Decision d'integration par surface

### Escouade

Surfaces : `apps/web/src/features/squad/SquadPage.tsx`, `apps/go-api/internal/service/squad_service.go`, `apps/go-api/internal/analysis/squad_timeseries.go`

Direction retenue :

- reimaginer le `squad score` autour d'un bloc plus explicite `performance + victoire + rendement combat`
- privilegier des valeurs exactes : rendement d'equipe, rendement par joueur de l'equipe, part des degats de l'equipe
- ne jamais opposer directement `un joueur` a `l'equipe` sur cette page
- ne pas introduire de repartition heuristique des degats entre coequipiers en coeur de page

Bon candidat de graphe :

- barres par joueur de l'equipe selectionnee, avec `conversion offensive`, `resistance defensive`, `perfect_kills` ou `part_damage_team`
- ou serie temporelle team-level par session / bucket pour visualiser comment l'escouade convertit collectivement ses degats

Anti-pattern a eviter ici :

- `solo ref` vs `avec teammate` melange dans le meme graphe d'escouade
- ratio `joueur / equipe` presente comme une comparaison frontale

### Synthese

Surfaces : `apps/web/src/features/synthesis/SynthesisPage.tsx`, `apps/go-api/internal/service/squad_service.go`

Direction retenue :

- ajouter `conversion offensive` et `resistance defensive` dans `comparison_metrics`
- exposer la metrique cote solo et cote escouade
- faire de `Synthese` la page de reference pour la comparaison `solo vs escouade`
- bon endroit pour les metriques d'equipe exactes et les deltas d'usage
- c'est ici que vit le graphe de comparaison `solo vs squad`, pas dans `Escouade`

Bon candidat de graphe :

- reutiliser la logique du graphe de comparaison existant pour y injecter `conversion offensive`, `resistance defensive`, `perfect_kills par match` et eventuellement `part de damage dealt en escouade`

### Forme / Timeseries

Surfaces : `apps/web/src/features/timeseries/TimeseriesPage.tsx`, `apps/go-api/internal/service/timeseries_service.go`, `apps/go-api/internal/service/stats_service.go`

Direction retenue :

- integrer la notion dans la `forme` via des courbes lissees / rolling, pas juste en valeur brute match par match
- afficher de preference une `forme combat` sur fenetre glissante
- traiter `conversion offensive` et `resistance defensive` comme des signaux de tendance, pas comme un seul verdict absolu

Implementations candidates :

- rolling average 10/20 matchs
- percentile vs historique joueur
- delta court terme vs long terme
- indice combine de forme combat construit a partir de percentiles, pas de bruts

### Performance Score

Surfaces : `apps/go-api/internal/analysis/performance_score.go`, `apps/go-api/internal/sync/performance.go`

Direction retenue :

- integration possible, mais pas en remplacement brutal de `dpm_damage`
- risque principal : double comptage avec `kills_vs_expected`, `deaths_vs_expected` et `damage per minute`
- si integration, preferer un signal normalise historiquement plutot qu'un brut

Recommandation :

- phase 1 : garder la metrique hors du score, en lecture analytique
- phase 2 : tester une version `performance v2` avec poids faible et calibration offline
- si adoption : backfill obligatoire des `performance_score` stockes

### LUSR

Surfaces : `apps/go-api/internal/analysis/skill_rating.go`, `apps/go-api/internal/sync/skill_rating.go`, `apps/go-api/internal/sync/skill_config.go`

Direction retenue :

- integration possible, mais plus sensible encore que pour Performance
- le LUSR utilise deja un `damage_efficiency` qui est en fait une balance de degats
- si on ajoute la nouvelle metrique sans nettoyage semantique, on cree une confusion analytique majeure

Recommandation :

- renommer d'abord la composante existante en `damage_balance`
- n'ajouter `offensive_conversion` / `defensive_resistance` qu'apres recalibration globale des poids
- eviter d'injecter deux signaux tres correlés sans etude de stabilite
- si adoption : backfill obligatoire de `match_skill_rank`

### Historique, tableaux et tuiles match

Surfaces : `apps/web/src/features/match-history/MatchHistoryTable.tsx`, `apps/go-api/internal/service/match_history_service.go`, `apps/web/src/features/home/HomePage.tsx`

Direction retenue :

- historique : ajouter soit la metrique joueur, soit la metrique d'equipe selon la page / le scope
- tuiles home / recent matches : eviter le simple badge texte ; preferer une barre composite horizontale compacte
- la barre composite doit resumer le match lui-meme, pas comparer solo vs squad

Regle :

- solo / historique joueur -> metriques joueur
- surfaces escouade / synthese -> metriques equipe + deltas solo/escouade

Direction UI pour les tuiles :

- pattern visuel type barre composite comme le visuel K/A/D fourni par l'utilisateur
- lecture compacte gauche -> droite, avec une separation claire entre charge offensive et charge defensive
- bon candidat : segment offensif, fin segment neutre / support, segment defensif, plus labels numeriques au-dessus
- si on y integre le rendement combat, il doit rester lisible en un coup d'oeil et ne pas transformer la tuile en mini-tableau

### Match View

Surfaces : `apps/web/src/features/match-view/MatchScoreboard.tsx`, `apps/web/src/features/match-view/PlayerDetailPanel.tsx`, `apps/go-api/internal/service/match_view_service.go`, `apps/go-api/api/openapi.yaml`

Direction retenue :

- surface ideale pour montrer la metrique au niveau match joueur par joueur
- mais il faut clarifier le contrat avant tout changement car `damage_efficiency` existe deja
- preferer un contrat explicite `offensive_conversion` / `defensive_resistance` a une reutilisation implicite de `damage_efficiency`
- si on ajoute la composante perfect, l'enrichissement doit venir d'une jointure medal events -> `killer_victim_pairs`, pas d'un simple count par match

### Carriere

Surface : `apps/web/src/features/career/CareerPage.tsx`

Direction retenue :

- exclure cette famille de metriques de `Carriere`
- ne pas lister ce type d'information sur cette page
- `Carriere` reste centree progression, rang, top matches, encounters et LUSR sans detail de rendement combat

## Autres implementations potentielles a explorer

### A. Version exacte simple

- `conversion offensive`
- `resistance defensive`
- `delta rendement combat = percentile(offensive) - percentile(vulnerabilite)` ou equivalent signe

### B. Version historique / relative

- percentile de `conversion offensive` vs historique perso
- percentile de `resistance defensive` vs historique perso
- badge `au-dessus / en-dessous de ta forme habituelle`

Cette variante est tres pertinente pour la `forme`.

### C. Version equipe

- `team_offensive_conversion`
- `team_defensive_resistance`
- `part_damage_dealt_team`
- `part_kills_team`
- `part_damage_taken_team`

Cette variante est la plus propre pour `Escouade` et `Synthese`.

### D. Version synergie duo / escouade

- delta `avec ce teammate` vs `solo ref`
- delta `avec ce teammate` vs `global squad ref`
- bucket par session pour voir si certaines associations ameliorent la conversion ou la resistance

### E. Version scoree / composite

- `combat_form_index`
- `combat_conversion_percentile`
- `combat_stability_index`

Ces variantes sont des candidates pour `Forme`, puis potentiellement pour `Performance` ou `LUSR` apres calibration.

### F. Version ancree par perfect kills

- `perfect_conversion_share = 225 * perfect_kills / damage_dealt`
- `offensive_conversion_residual`
- `defensive_resistance_residual`
- `exact_pair_damage_ij = 225 * perfect_kills_ij`

Cette famille est particulierement interessante car elle separe enfin :

- une part exacte
- une part residualisee
- une part encore incertaine

Elle ne rend pas la metrique miraculeusement causale, mais elle est nettement plus defendable analytiquement que la version totalement plate.

## Ce qu'il ne faut pas faire

- ne pas presenter une repartition player-level des degats d'equipe comme une verite observee
- ne pas renommer silencieusement `damage_efficiency` sans migrer les contrats et les tests
- ne pas injecter la metrique brute dans Performance ou LUSR sans verifier les correlations avec les autres composantes
- ne pas exposer deux metriques de sens inverse sous des labels presque identiques
- ne pas supposer que `medals_earned` suffit a identifier `qui a perfect qui` ; cette information doit etre reconstruite temporellement

## Backfill et migration

Si la metrique reste purement derivee au rendu depuis les payloads existantes :

- pas de migration de schema obligatoire
- pas de backfill obligatoire

Si la metrique entre dans `Performance Score` ou `LUSR` :

- versionner explicitement la formule
- recalculer les scores persistants
- passer par le workflow standard `scripts/backfill_data.py`
- si de nouvelles colonnes sont stockees, appliquer la discipline migrations DuckDB + step dedie

Ordre recommande :

1. UI display-only sur surfaces match / squad / synthese / timeseries
2. validation analytique sur donnees reelles
3. integration eventuelle dans Performance
4. integration eventuelle dans LUSR

## Strategie de tests

### Tests unitaires analyse

Fichiers cibles :

- `apps/go-api/internal/analysis/performance_score_test.go`
- `apps/go-api/internal/analysis/skill_rating_test.go`
- `apps/go-api/internal/analysis/skill_rating_extra_test.go`
- `apps/go-api/internal/analysis/squad_test.go`
- `apps/go-api/internal/analysis/squad_timeseries_test.go`

Cas a couvrir :

- formule offensive nominale
- formule defensive nominale
- cas limites `0 damage`, `0 kills`, `0 deaths`
- invariants de sens (meilleure conversion offensive => score plus haut)
- verification du sens produit de `resistance defensive`

### Tests service / payload

Fichiers cibles :

- `apps/go-api/internal/service/stats_service_test.go`
- `apps/go-api/internal/service/timeseries_service_test.go`
- `apps/go-api/internal/service/squad_service_test.go`
- `apps/go-api/internal/service/match_history_service_test.go`
- `apps/go-api/internal/service/match_view_service_test.go`

Cas a couvrir :

- presence / absence des nouveaux champs
- non regression des pages vides
- compatibilite des contrats si on enrichit des lignes de tableau ou de scoreboard

### Tests contrat

Si de nouveaux champs API sortent :

- mise a jour de `apps/go-api/api/openapi.yaml`
- couverture via les tests contrat OpenAPI/chi deja en place

### Tests React

Fichiers deja en place :

- `apps/web/src/features/home/HomePage.test.tsx`
- `apps/web/src/features/squad/SquadPage.test.tsx`
- `apps/web/src/features/synthesis/SynthesisPage.test.tsx`
- `apps/web/src/features/match-history/MatchHistoryPage.test.tsx`
- `apps/web/src/features/career/CareerPage.test.tsx`

Ajouts probables :

- test de rendu d'un badge / cellule `conversion offensive`
- test de rendu d'une carte / cellule `resistance defensive`
- test de libelle pour eviter toute confusion semantique

## Synthese finale

Statut actuel :

- la metrique est analytiquement viable au niveau agrege joueur/equipe
- les perfect kills permettent un affinage partiel et tres utile, en ancrant une part exacte a `225`
- la famille doit etre renommee pour sortir du piege `efficacite`
- la meilleure premiere cible produit est `Escouade + Synthese + Forme + Match surfaces`, avec des roles distincts selon les pages
- `Performance` et `LUSR` sont des etapes ulterieures, avec recalibration et backfill

Terme recommande a ce stade :

- famille : `rendement combat`
- libelles principaux : `conversion offensive` et `resistance defensive`
