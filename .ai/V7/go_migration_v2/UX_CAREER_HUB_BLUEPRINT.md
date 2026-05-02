# UX_CAREER_HUB_BLUEPRINT.md — Blueprint détaillé du hub Carrière

## Périmètre

- Ce document détaille le **lot 2** défini dans `UX_CAREER_SYNTHESIS_BOUNDARY.md`.
- Il couvre uniquement le projet go-migration.
- Il décrit la cible React/Go pour la page canonique `/players/$playerSlug/career`.
- Il ne couvre ni le renommage global de navigation (lot 1), ni le contrat détaillé de Synthèse (traité séparément).

## Référence amont

Le cadrage produit de base reste `UX_CAREER_SYNTHESIS_BOUNDARY.md`.

Ce blueprint répond à une question plus concrète :

> à quoi doit ressembler la page Carrière une fois qu'elle devient un vrai hub `Progression + Citations` ?

## Snapshot courant

### Frontend React

- `apps/web/src/routes/players/$playerSlug/career.tsx` expose aujourd'hui une page unique `CareerPage`.
- `apps/web/src/features/career/CareerPage.tsx` assemble encore trois registres différents :
  1. progression durable (`summary`, `hero_progress`, `charts`, `lusr`) ;
  2. performances marquantes via `top_matches_preview` ;
  3. rivalités/rencontres via `encounters_preview`.
- `apps/web/src/routes/players/$playerSlug/profile/citations.tsx` garde `Citations` comme route secondaire indépendante.
- `apps/web/src/features/citations/CitationsPage.tsx` dépend du `globalFilterStore` et affiche encore un delta filtré vs complet, ce qui renforce une lecture de scope plutôt qu'une capitalisation durable.

### Backend Go

- `apps/go-api/internal/api/handlers/career.go` expose encore trois endpoints distincts :
  1. `GET /pages/career`
  2. `GET /pages/career/top-matches`
  3. `GET /pages/career/encounters`
- `apps/go-api/internal/domain/career.go` encode encore `top_matches_preview` et `encounters_preview` comme prolongements naturels de la page Carrière.

## Problème précis à résoudre

Le hub Carrière doit devenir une page de **capital long terme**. Or l'assemblage courant raconte encore en partie le périmètre sélectionné et les extrêmes de performance, donc une logique de Synthèse.

Le redesign doit produire une page dont la lecture est immédiatement stable :

1. le joueur comprend qu'il est sur sa trajectoire durable ;
2. les citations y sont traitées comme un capital de maîtrise ;
3. rien dans la page ne ressemble à une récap analytique de période.

## Décision de structure

La page Carrière devient un **hub mono-route** avec deux vues internes :

1. `Progression`
2. `Citations`

Le mot `Carrière` ne doit plus être réutilisé comme nom d'onglet interne.

## Canonical routing

### Route canonique

- Route canonique conservée : `/players/$playerSlug/career`

### Stratégie recommandée pour le tab state

Recommandation prioritaire : représenter l'onglet via un search param.

Exemples :

- `/players/$playerSlug/career?tab=progression`
- `/players/$playerSlug/career?tab=citations`

### Pourquoi ce choix

1. il garde `Carrière` comme destination unique dans la navigation du shell ;
2. il évite de créer une pseudo-hiérarchie produit où `Citations` redeviendrait une destination autonome ;
3. il simplifie le redirect depuis la route legacy `/players/$playerSlug/profile/citations`.

### Redirect legacy

Le redirect cible recommandé est :

- `/players/$playerSlug/profile/citations` → `/players/$playerSlug/career?tab=citations`

## Chrome de page cible

### Header

Titre : `Carrière`

Sous-titre cible :

- `Progression durable, maîtrise et capital joueur`

Le header ne doit plus promettre des `statistiques globales`, formulation trop proche d'une page de synthèse.

### Tab strip

Les deux tabs sont visibles juste sous le header :

1. `Progression`
2. `Citations`

### Règle de comportement

- l'onglet par défaut est `Progression` ;
- le tab state est deep-linkable ;
- un changement d'onglet ne doit pas faire disparaître le header ni changer l'identité de la page ;
- aucun onglet interne ne doit apparaître dans la navigation secondaire globale du shell.

## Vue `Progression`

### Intention

Montrer où en est le joueur dans sa carrière et où il va.

### Blocs cibles

#### Bloc 1 — Snapshot carrière

Contenu attendu :

1. rang actuel ;
2. label de rang ;
3. XP courant ;
4. progression vers le prochain rang ;
5. progression vers Héros ;
6. saison courante si disponible (`current_season`).

### Bloc 2 — Projection et trajectoire

Contenu attendu :

1. vitesse de progression ;
2. projections de date ;
3. indicateur de fiabilité si projection fallback.

### Bloc 3 — Historique XP

Contenu attendu :

1. courbe XP ;
2. checkpoints notables ;
3. éventuels marqueurs de saison si le backend les expose.

### Bloc 4 — Rating compétitif / niveau durable

Contenu attendu :

1. état LUSR actuel ;
2. tendance ;
3. checkpoints historiques ;
4. emplacement CSR si la donnée devient durablement disponible dans cette surface.

### Contenu explicitement retiré de `Progression`

1. top matchs ;
2. pires matchs ;
3. rencontres fréquentes ;
4. antagonistes ;
5. heatmap temporelle ;
6. top semaines ;
7. comparaison solo vs escouade.

## Vue `Citations`

### Intention

Montrer le capital de maîtrise du joueur, pas une photo de période filtrée.

### Règle structurante

La vue `Citations` ne dépend pas du `globalFilterStore` dans sa première version de hub.

Si un filtrage local est ajouté plus tard, il doit être explicitement présenté comme un **filtre local de la vue Citations**, jamais comme le scope analytique global des pages match-oriented.

### Blocs cibles

#### Bloc 1 — Résumé de maîtrise

Contenu attendu :

1. total de commendations ;
2. total de médailles ;
3. nombre ou ratio d'entrées maîtrisées si disponible ;
4. éventuels jalons remarquables.

### Bloc 2 — Commendations

Contenu attendu :

1. liste ou grille par catégorie ;
2. tier label ;
3. valeur courante ;
4. progression de maîtrise.

### Bloc 3 — Médailles

Contenu attendu :

1. medal cabinet trié par importance d'usage ;
2. distribution ;
3. possibilité future de drilldown.

### Bloc explicitement retiré de la vue actuelle

Les KPIs `Total filtré / Total complet / Delta` ne doivent pas être repris tels quels dans le hub Carrière.

Pourquoi :

1. ils racontent une comparaison de scope ;
2. ils renforcent un usage analytique de période ;
3. ils brouillent la distinction avec Synthèse.

## Décomposition React recommandée

### Container

- `CareerHubPage.tsx` : route container, lecture du search param `tab`, header, tab strip, orchestration data.

### Sous-composants cible

- `CareerHubTabs.tsx`
- `CareerProgressionTab.tsx`
- `CareerCitationsTab.tsx`
- `CareerSeasonBadge.tsx`
- `CareerLUSRCard.tsx`
- `CitationsMasterySummary.tsx`
- `CitationsCommendationsSection.tsx`
- `CitationsMedalsSection.tsx`

### Réutilisation de l'existant

À conserver ou extraire :

1. `CareerSummaryCard`
2. `CareerChartsSection`
3. une partie de la logique actuelle de rendu LUSR
4. des sections de `CitationsPage` pour les cartes commendations et la grille de médailles

À sortir du hub :

1. `CareerTopMatchesTable`
2. `CareerEncountersSection`

## Stratégie de requêtes recommandée

### Version de transition minimale

Réutiliser les endpoints existants au maximum :

1. `GET /players/{slug}/pages/career` pour l'onglet `Progression`
2. `POST /players/{slug}/pages/citations` pour l'onglet `Citations`

### Règles de transition

1. l'onglet `Progression` ne charge plus `top-matches` ni `encounters` ;
2. l'onglet `Citations` est appelé avec une requête neutre, sans recyclage automatique des filtres globaux ;
3. le frontend ne garde pas la route `/profile/citations` dans la navigation.

### Cible endpoint plus propre

À moyen terme, la cible recommandée est :

1. `GET /players/{slug}/pages/career` → `CareerProgressionTabResponse`
2. `POST /players/{slug}/pages/career/citations` → `CareerCitationsTabResponse`

Cette cible clarifie l'appartenance de `Citations` au hub sans imposer immédiatement une refonte complète des données côté Go.

## Contrat cible minimal pour `Progression`

Le contrat de `Progression` doit rester concentré sur :

1. `summary`
2. `hero_progress`
3. `projections`
4. `charts`
5. `xp_history`
6. `lusr`
7. `current_season`

Les champs suivants doivent sortir de la payload :

1. `top_matches_preview`
2. `encounters_preview`

## Contrat cible minimal pour `Citations`

Le contrat de `Citations` peut rester proche de l'existant, avec recentrage sémantique :

1. `commendations`
2. `medals_summary`
3. `distribution_chart`
4. éventuellement un `mastery_summary` plus durable

Le bloc `deltas` ne doit plus être le premier niveau de lecture du hub.

## Migration frontend recommandée

### Étape A

1. créer `CareerHubPage.tsx` ;
2. déplacer `CareerPage.tsx` vers `CareerProgressionTab.tsx` ;
3. déplacer `CitationsPage.tsx` vers `CareerCitationsTab.tsx` après retrait du filtre global implicite.

### Étape B

1. remplacer l'ancienne route citations par un redirect ;
2. retirer l'entrée secondaire `Citations` du shell ;
3. nettoyer les query keys et libellés associés.

### Étape C

1. supprimer du hub toutes les références à `top matches` et `encounters` ;
2. brancher ces contenus dans Synthèse.

## Critères d'acceptation

1. La page `Carrière` affiche `Progression` par défaut.
2. La vue `Citations` est accessible sans devenir une destination shell autonome.
3. `top matches` et `encounters` n'apparaissent plus dans Carrière.
4. La vue `Citations` ne varie pas silencieusement avec les filtres globaux de lecture match.
5. Le redirect depuis `/profile/citations` mène vers l'onglet `Citations` du hub.
6. Le header et le wording de page décrivent une capitalisation durable, pas une synthèse de période.

## Décision opérationnelle

Le hub Carrière doit être implémenté comme une **page unique à tabs deep-linkables**, pas comme deux destinations de navigation concurrentes.

La cible à construire est donc :

- `Carrière` = conteneur unique
- `Progression` = trajectoire durable
- `Citations` = capital de maîtrise

Tout ce qui ressemble à un récit de scope courant doit être retiré de ce hub et basculé dans Synthèse.
