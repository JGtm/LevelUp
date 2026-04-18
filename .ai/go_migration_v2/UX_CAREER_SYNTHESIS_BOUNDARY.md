# UX_CAREER_SYNTHESIS_BOUNDARY.md — Cadrage UX Carrière / Synthèse

## Périmètre

- Ce cadrage s'applique uniquement au projet go-migration.
- La cible est le shell React dans apps/web et ses contrats Go dans apps/go-api.
- Le legacy Python / Streamlit est explicitement hors scope.

## Problème produit

L'UX actuelle mélange deux logiques distinctes :

1. une logique de capitalisation long terme du joueur ;
2. une logique de lecture analytique du périmètre actuellement sélectionné.

Aujourd'hui, la page Carrière React porte encore des blocs qui relèvent d'une lecture de synthèse : top/pires performances et rencontres. En parallèle, le corpus historique parle encore d'un hub Profil, alors que le shell React a déjà promu Carrière comme entrée principale.

Le résultat est une frontière floue entre :

- ce qui décrit durablement le joueur ;
- ce qui raconte la période ou les filtres actifs.

## Décisions produit

1. Le terme Profil sort de l'UX cible go-migration.
2. L'entrée de navigation joueur concernée s'appelle Carrière.
3. Dans la page Carrière, l'onglet interne anciennement nommé Carrière devient Progression.
4. Carrière devient le hub de capitalisation long terme du joueur.
5. Synthèse devient la page de lecture transversale du périmètre courant.
6. Tout bloc fortement dépendant de la période, des filtres ou du contexte solo / escouade doit vivre en Synthèse.

## Nomenclature cible

| Concept legacy | Libellé cible | Rôle produit |
|----------------|---------------|--------------|
| Profil | Carrière | Hub joueur long terme |
| Carrière (onglet interne) | Progression | Progression rang / XP / LUSR |
| Citations | Citations | Capitalisation des commendations et médailles |
| Synthèse | Synthèse | Vue récapitulative du scope filtré |

## Règle d'arbitrage

Un bloc appartient à Carrière s'il répond à la question : qui est ce joueur dans la durée ?

Un bloc appartient à Synthèse s'il répond à la question : que raconte le périmètre actuellement sélectionné ?

Corollaire :

- si un bloc change fortement avec la période, les filtres ou le split solo / escouade, il va en Synthèse ;
- si un bloc représente un capital durable ou un état de progression, il va en Carrière.

## Architecture cible

### Carrière

#### Objectif

Décrire l'identité long terme du joueur : progression, rang, trajectoire et capital symbolique.

#### Navigation interne

- Onglet 1 : Progression
- Onglet 2 : Citations

#### Onglet Progression

Contenu cible :

1. Rang actuel et progression XP
2. Progression vers Héros
3. Historique XP et projections
4. LUSR / CSR et leur évolution

Contenu explicitement exclu :

1. Top / pires performances
2. Top antagonistes, némésis, victimes
3. Heatmap temporelle
4. Top semaines
5. Comparaison Solo vs Escouade
6. Breakdown carte / mode à visée analytique

#### Onglet Citations

Contenu cible :

1. Commendations
2. Médailles
3. Distribution de citations et progression de maîtrise

Règle UX : Citations reste un capital du joueur. Cette vue ne doit pas servir de fourre-tout pour des récaps de période qui appartiennent à Synthèse.

### Synthèse

#### Objectif

Donner une lecture compacte, analytique et actionnable du scope courant : filtres globaux, période locale et contexte solo / escouade.

#### Ordre de lecture cible

1. Vue d'ensemble
2. Solo vs Escouade
3. Performances marquantes
4. Rivalités
5. Activité et temporalité
6. Répartition carte / mode

#### Bloc 1 — Vue d'ensemble

Nouveau bloc central de la page.

Principe de présentation :

1. Cumuls pour les volumes
2. Moyennes pour les métriques d'efficacité
3. Maximums pour les pics ou records courts

Découpage recommandé :

- Volumes cumulés : matchs, victoires, défaites, kills, deaths, assists, temps joué
- Moyennes : win rate, K/D, accuracy, kills/min, durée de vie moyenne, performance score
- Pics : meilleur match en kills, meilleur match en performance score, meilleure précision, plus longue série pertinente si disponible

Règle produit : si une métrique n'est pas fiable ou pas disponible côté backend Go, elle doit être omise, pas simulée.

#### Bloc 2 — Solo vs Escouade

La comparaison solo / escouade reste dans Synthèse. C'est une lecture contextuelle du scope, pas un attribut de carrière.

#### Bloc 3 — Performances marquantes

Ce bloc absorbe :

1. top performances
2. pires performances

Règle : les matchs mis en avant doivent être cohérents avec le scope de Synthèse. Si la période vaut all, la page peut montrer l'all-time ; sinon elle montre les extrêmes du périmètre filtré.

#### Bloc 4 — Rivalités

Ce bloc absorbe :

1. top antagonistes
2. némésis
3. victimes
4. top rencontrés si le produit décide de garder cette lecture relationnelle dans la même surface

Règle : dès lors qu'on lit ces tables comme une conséquence du périmètre analysé, elles n'ont plus leur place dans Carrière.

#### Bloc 5 — Activité et temporalité

Ce bloc regroupe :

1. heatmap jour × heure
2. top semaines

#### Bloc 6 — Répartition carte / mode

Le breakdown carte / mode reste en Synthèse comme lecture agrégée du scope courant.

## Règles d'interaction

### Carrière

- Progression est une vue long terme.
- Elle ne doit pas dépendre des filtres globaux de lecture match par match.
- Citations peut conserver un filtrage local plus tard si nécessaire, mais ce filtrage ne doit pas brouiller la séparation Carrière / Synthèse.

### Synthèse

- Synthèse dépend du contexte de filtres global.
- Le sélecteur de période local reste propre à la page.
- Tous les blocs analytiques déplacés depuis Carrière doivent respecter exactement ce même scope.

## Impacts navigation et routing

### Cible UX

1. Le joueur voit Carrière dans la navigation.
2. Citations n'est plus une destination de navigation globale indépendante.
3. Citations devient une vue interne de Carrière.

### Cible technique recommandée

1. Faire de /players/$playerSlug/career le hub visuel canonique.
2. Déprécier /players/$playerSlug/profile/citations au profit d'une destination carrière.
3. Prévoir un redirect propre pendant la migration pour ne pas casser les liens existants.

## Impacts contrats Go / React

### Carrière

Tendance cible : alléger le contrat de page Carrière pour le recentrer sur la progression et les citations.

Les champs suivants n'ont plus vocation à rester au coeur de la page Carrière :

1. top_matches_preview
2. encounters_preview

### Synthèse

La page Synthèse doit devenir la surface réceptrice des blocs analytiques déplacés.

Extension cible recommandée du contrat :

1. overview
2. highlighted_matches_preview
3. rivalries_preview
4. heatmap_data
5. top_weeks
6. comparison_metrics
7. map_mode_breakdowns

Si la payload devient trop lourde, la stratégie recommandée est :

1. garder un preview dans la page principale ;
2. charger les détails lourds via endpoints dédiés ou lazy queries ;
3. éviter une page monolithique qui fetch tout même quand les panneaux sont repliés.

## Ordre de livraison recommandé

### Lot 1 — Terminologie et navigation

1. supprimer Profil des libellés UX côté go-migration ;
2. renommer l'onglet interne Carrière en Progression ;
3. replier Citations sous Carrière.

### Lot 2 — Hub Carrière

1. faire converger la page Carrière vers une vraie structure Progression / Citations ;
2. retirer les blocs analytiques qui relèvent de Synthèse.

### Lot 3 — Enrichissement Synthèse

1. ajouter la Vue d'ensemble ;
2. déplacer top / pires performances ;
3. déplacer rivalités et antagonistes ;
4. harmoniser le scope de tous les blocs avec les filtres et la période.

## Critères d'acceptation UX

1. Un utilisateur comprend immédiatement que Carrière parle de progression long terme.
2. Un utilisateur comprend immédiatement que Synthèse parle du périmètre courant.
3. Le mot Profil ne subsiste plus dans l'UX cible go-migration.
4. Le doublon Carrière / onglet Carrière a disparu au profit de Progression.
5. Les performances marquantes et rivalités ne créent plus de confusion avec la progression de carrière.

## Décision finale

La séparation cible est la suivante :

- Carrière = progression + citations + capital long terme
- Synthèse = overview + comparaison de contexte + performances marquantes + rivalités + patterns d'activité

Ce cadrage doit servir de référence produit pour les prochaines modifications React et Go sur ce périmètre.
