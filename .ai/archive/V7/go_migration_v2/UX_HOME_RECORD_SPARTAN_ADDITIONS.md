# UX_HOME_RECORD_SPARTAN_ADDITIONS.md — Ajouts inspirés de Spartan Record

## Périmètre

- Ce document s'applique uniquement au projet go-migration.
- Il complète le cadrage existant `Carrière / Synthèse` sans le remettre en cause.
- Il ne propose pas un remplacement de la page record actuelle copiée depuis Python.
- Il décrit des **ajouts ciblés** inspirés de Spartan Record, réinterprétés pour LevelUp.

## Références amont

Ce document complète :

- `UX_CAREER_SYNTHESIS_BOUNDARY.md`
- `UX_CAREER_HUB_BLUEPRINT.md`
- `SYNTHESIS_TARGET_CONTRACT_AND_UI.md`

Il répond à une question différente :

> comment enrichir la surface d'accueil / record de LevelUp avec certaines idées fortes de Spartan Record, sans perdre l'identité actuelle du produit ni recréer une confusion entre Accueil, Carrière et Synthèse ?

## Position de départ

### Côté produit

La surface la plus proche d'un `service record` d'accueil dans le shell React actuel est :

- `apps/web/src/features/home/HomePage.tsx`

Cette page joue aujourd'hui le rôle de `Mission Control` :

1. KPIs globaux ;
2. Battle Pass / défis ;
3. sessions récentes ;
4. highlights ;
5. matchs récents ;
6. accès rapides ;
7. dernier match ;
8. médias récents.

### Côté frontière produit

Le cadre déjà retenu reste valable :

- `Carrière` = progression durable + citations + capital long terme ;
- `Synthèse` = lecture analytique du scope courant ;
- `Accueil / Record` = point d'entrée joueur rapide, lisible et éditorial.

## Décision de fond

La page record actuelle est **conservée** comme base.

La bonne stratégie n'est pas un remplacement `Home → clone de Spartan Record`, mais une série d'ajouts précis :

1. renforcer l'identité joueur ;
2. enrichir certaines sections de lecture rapide ;
3. améliorer la présentation de quelques blocs clés ;
4. préparer des métadonnées plus dynamiques pour les visuels.

## Clarification vocabulaire

Le terme `hero block` peut prêter à confusion.

Dans ce document, il ne veut pas dire :

1. un énorme bandeau marketing ;
2. une grande carte pleine largeur type landing page ;
3. une seconde page `Carrière` cachée dans l'accueil.

Ce qu'il veut dire ici est plus simple :

1. le **bloc d'ouverture principal** de la home ;
2. la première zone que l'utilisateur lit en arrivant ;
3. la zone qui donne immédiatement l'identité du joueur et 3 à 5 informations majeures.

### Recommandation ajustée

Vu la préférence exprimée pour un `Spartan ID` assez petit, la bonne direction n'est **pas** de fusionner agressivement l'identité et toute la carte `Performance globale` dans un seul gros composant.

La recommandation la plus saine est plutôt :

1. un `Spartan ID` compact en tête ;
2. juste en dessous ou juste à côté, un résumé de performance plus serré ;
3. une relation visuelle claire entre les deux, sans les forcer dans un même bloc massif.

Autrement dit :

on peut parler d'un **bloc d'ouverture record**, mais pas forcément d'une **fusion totale** entre identité et performance.

## Décision mise à jour sur l'ouverture de la home

Le terme à privilégier pour LevelUp est donc :

- `bloc d'ouverture record`

et sa composition recommandée est :

1. `Spartan ID` compact ;
2. KPIs majeurs juste à proximité ;
3. éventuellement une petite carte `Career Rank` ou `Time Played` si elle reste très compacte.

Ce qui n'est pas recommandé à ce stade :

1. une grande fusion visuelle identitaire + 6 KPIs + rang + dataset + médias ;
2. une carte d'ouverture trop haute qui mange toute la ligne de flottaison.

## Diagnostic UX sur l'existant Go

Avant d'ajouter quoi que ce soit, il faut regarder où le shell React est déjà dense ou redondant.

### 1. `HomePage` est déjà riche, mais hiérarchiquement plate

Aujourd'hui, `apps/web/src/features/home/HomePage.tsx` contient déjà :

1. `Performance globale`
2. `Battle Pass`
3. `Défis actifs`
4. `Sessions récentes`
5. `Points saillants`
6. `Matchs récents`
7. `Accès rapide`
8. `Dernier match`
9. `Médias récents`

Le problème principal n'est pas le volume pur, mais l'absence de hiérarchie forte entre ces blocs.

Résultat :

1. tout a presque le même poids ;
2. la page manque d'un vrai point d'entrée identitaire ;
3. plusieurs cartes racontent des choses voisines avec des composants séparés.

### 2. `CareerPage` est déjà trop chargée pour accueillir plus sans retrait

Aujourd'hui, `apps/web/src/features/career/CareerPage.tsx` porte encore à la fois :

1. la progression durable ;
2. le LUSR ;
3. les top matchs ;
4. les rencontres fréquentes.

Donc, avant d'ajouter un `Hero Tracker`, il faut déjà faire de la place.

### 3. `CitationsPage` a beaucoup de contenu, mais une hiérarchie encore brute

Aujourd'hui, `apps/web/src/features/citations/CitationsPage.tsx` ouvre avec :

1. le delta filtré ;
2. la distribution ;
3. les commendations ;
4. la grille de médailles.

Le contenu est utile, mais la lecture premium des médailles n'est pas encore clairement priorisée.

### 4. `SynthesisPage` est analytique, donc son budget d'attention est déjà entamé

Aujourd'hui, `apps/web/src/features/synthesis/SynthesisPage.tsx` affiche déjà :

1. deux cartes KPI `Solo / Escouade` ;
2. un graphique bipolaire ;
3. une table détaillée de comparaison ;
4. une heatmap ;
5. un tableau `Top semaines`.

Cette page peut absorber des enrichissements analytiques, mais pas sans repriorisation.

### 5. `MatchHistoryPage` est la surface la plus saine

`apps/web/src/features/match-history/MatchHistoryPage.tsx` reste focalisée sur une seule tâche : lire et exporter un historique dense.

Ici, l'ajout de cartes visuelles est un complément crédible, pas une réécriture nécessaire.

## Règle de budget d'attention

Le principe à retenir pour ce chantier est simple :

**on n'ajoute pas un gros bloc sans compresser, fusionner ou déclasser au moins un bloc existant sur la même page.**

Sans cette règle, la référence Spartan Record deviendrait un prétexte pour empiler encore plus de surface.

## Ce qui est redondant aujourd'hui

### Accueil

#### `Matchs récents` et `Dernier match`

Ces deux blocs racontent quasiment la même chose.

Décision recommandée :

1. garder un seul bloc `Matchs récents` ;
2. intégrer le CTA `Voir le dernier match` dans la première tuile ou la première ligne ;
3. supprimer la carte dédiée `Dernier match` si des tuiles de matchs enrichies arrivent.

#### `Performance globale` et futur record hero

Si on ajoute un `Spartan ID` + un résumé type service record, garder la carte `Performance globale` telle quelle deviendrait redondant.

Décision recommandée :

1. remplacer la carte `Performance globale` actuelle par une version plus resserrée ;
2. associer visuellement ce résumé au `Spartan ID` ;
3. ne pas imposer une fusion monobloc si le `Spartan ID` reste volontairement compact.

#### `Sessions récentes` et `Points saillants`

Ces deux blocs appartiennent au même registre de lecture : `qu'est-ce qui s'est passé récemment ?`

Décision recommandée :

1. conserver les deux seulement si leur rôle visuel est bien distinct ;
2. sinon, les fusionner dans un bloc `Signaux récents` ou `Activité récente`.

#### `Battle Pass / Défis` face à la logique record

Ces blocs sont utiles, mais ils relèvent du live-service, pas de l'identité record du joueur.

Décision recommandée :

1. les conserver ;
2. mais les faire descendre visuellement sous la partie record ;
3. éviter qu'ils occupent le tout début de page une fois le `Spartan ID` en place.

### Carrière

#### `Top matchs` et `Rencontres fréquentes`

Ces blocs sont déjà identifiés comme du bruit par rapport à la mission durable de `Carrière`.

Décision recommandée :

1. ils doivent s'effacer de `CareerPage` avant toute inspiration Spartan Record supplémentaire ;
2. le `Hero Tracker` ne doit arriver qu'après cette clarification.

### Citations

#### `Delta` en tête de page

Le delta est utile, mais il prend aujourd'hui une place de lecture très haute.

Décision recommandée :

1. si la page reste analytique, compacter le delta ;
2. si la page devient un onglet durable de `Carrière`, retirer ce trio de KPIs du haut de page ;
3. en contrepartie, faire remonter le cabinet `Top Medals` et la hiérarchie des médailles.

### Synthèse

#### Détail de comparaison trop haut dans la page

Le combo `cartes KPI + graphique + table détaillée + heatmap + top semaines` consomme déjà beaucoup d'attention avant même l'ajout d'autres blocs.

Décision recommandée :

1. si `overview`, `highlights` et `rivalries` arrivent, la table détaillée solo / escouade doit descendre ou devenir repliable ;
2. on ne peut pas tout garder à poids égal au-dessus de la ligne de flottaison.

## Ce qu'on reprend

### 1. Un `Spartan ID` compact

Inspiration retenue : la carte d'identité joueur visuellement forte mais de taille contenue.

### Cible LevelUp

Ajouter en tête de `HomePage` un bloc `Spartan ID` compact qui contient :

1. emblème ;
2. backdrop ou bannière ;
3. gamertag ;
4. service tag ou identifiant Halo lisible ;
5. rang carrière actuel ;
6. éventuellement un petit ornement de rang si disponible.

### Taille cible

Le `Spartan ID` doit rester une carte d'identité, pas une fresque.

Donc :

1. hauteur visuelle contenue ;
2. lecture rapide en un coup d'oeil ;
3. pas d'empilement de sous-sections analytiques à l'intérieur.

### Principe de sourcing

Le futur `Spartan ID` de LevelUp ne doit pas copier la contrainte de sourcing de Spartan Record.

La bonne formule ici est :

1. **leur design** ;
2. **nos endpoints**.

Autrement dit :

1. inspiration visuelle externe ;
2. source de vérité applicative LevelUp.

### Règle de priorité des données

L'ordre de priorité recommandé pour alimenter le composant est :

1. données enrichies/authentifiées LevelUp issues de la session Halo active, du cache MSAL ou des échanges de tokens déjà en place ;
2. données locales persistées déjà synchronisées si elles existent ;
3. fallback public minimal seulement en dernier recours.

### Pourquoi c'est important

1. certaines données d'apparence riches comme backdrop, bannière ou ornements peuvent dépasser le simple monde public ;
2. le projet Go a déjà un pipeline `MSAL → exchange Halo → session auth`, donc une base plus riche que ce qu'un composant `public-only` permettrait ;
3. il serait contre-productif de dégrader volontairement le composant au niveau le plus pauvre juste pour imiter une implémentation externe.

### Règles

1. la carte reste plus petite que chez Spartan Record ;
2. elle ne remplace pas le header shell global ;
3. elle ne duplique pas la page Carrière ;
4. elle doit fonctionner même si certains assets manquent ;
5. elle doit utiliser les données d'apparence enrichies de LevelUp quand elles sont disponibles, sans se limiter à une version `public-only`.

### Décision produit

Le `Spartan ID` appartient à `Accueil / Record`, pas à `Carrière`.

Pourquoi :

1. c'est un point d'entrée identitaire ;
2. il renforce l'impression de produit dès l'arrivée ;
3. il n'a pas vocation à devenir une page autonome.

Corollaire :

ce composant doit reprendre le meilleur des deux mondes :

1. le traitement visuel de Spartan Record ;
2. les données plus riches et mieux authentifiées de LevelUp.

## 2. Enrichir la home record sans la remplacer

Inspiration retenue : la page `Service Record` de Spartan Record est claire parce qu'elle aligne quelques sections structurantes toujours lisibles.

### Cible LevelUp

Conserver la structure générale de `HomePage`, mais l'enrichir avec des blocs mieux hiérarchisés autour des thèmes suivants :

1. Matches
2. Kills & Deaths
3. Accuracy
4. Kills
5. Career Rank
6. CSRs
7. Top Medals
8. Data Set
9. Time Played

### Règle de composition

Ces blocs ne doivent pas transformer l'accueil en dashboard analytique complet.

La home doit rester :

1. rapide à lire ;
2. stable ;
3. orientée résumé ;
4. plus éditoriale que `Synthèse`.

### Traduction concrète

La home peut absorber :

1. un meilleur résumé de volumes et d'efficacité ;
2. une carte `Career Rank` compacte ;
3. une carte `CSR` compacte si la donnée est fiable ;
4. un preview `Top Medals` ;
5. une carte `Time Played` ;
6. une carte `Data Set` de confiance.

Elle ne doit pas absorber :

1. les heatmaps ;
2. les rivalités ;
3. les breakdowns détaillés carte / mode ;
4. les comparaisons lourdes solo / escouade.

### Recommandation sur le haut de page

Le haut de page recommandé devient :

1. `Spartan ID` compact ;
2. résumé de performance resserré ;
3. puis seulement ensuite les autres cartes éditoriales.

Il n'est donc pas obligatoire de parler d'une fusion totale entre `Spartan ID` et `Performance globale`.

La bonne décision produit est surtout d'éviter deux blocs d'ouverture qui racontent la même chose sans hiérarchie.

### Ce qui doit s'effacer pour faire la place

Pour que cette évolution reste tenable, les arbitrages recommandés sur `HomePage` sont :

1. **fusionner** `Performance globale` avec le futur bloc hero `Spartan ID` ;
2. **fusionner ou supprimer** la carte `Dernier match` au profit d'un bloc `Matchs récents` plus riche ;
3. **déclasser** `Battle Pass` et `Défis` sous le coeur record de la page ;
4. **réduire ou fusionner** `Sessions récentes` et `Points saillants` si la page devient trop longue.

Autrement dit :

la home peut gagner 2 ou 3 inspirations fortes, mais seulement si 2 ou 3 blocs existants cessent d'exister comme cartes autonomes de premier rang.

## 3. Rejet du toggle `Overall / Per Match`

Décision : **ne pas en faire une direction produit prioritaire**.

### Raisons

1. ce toggle n'est pas naturellement aligné avec la lecture actuelle de LevelUp ;
2. il introduit un second axe de lecture qui brouille facilement les KPIs ;
3. la stratégie actuelle de LevelUp distingue déjà assez clairement volumes, ratios et vues analytiques ;
4. le bénéfice perçu semble inférieur à celui d'un meilleur découpage des blocs.

### Corollaire

Pas de toggle global `Overall / Per Match` à prévoir sur `HomePage` dans ce lot.

Si un besoin réel réapparaît plus tard, il devra être justifié bloc par bloc, pas injecté comme paradigme global de la home.

## 4. Hero Tracker oui, seconde page Carrière non

Décision : `Carrière` existe déjà ; il ne faut pas créer un second parcours qui ferait doublon.

### Ce qui est retenu

Ajouter un module `Hero Tracker` ou une variante simplifiée dans `Carrière`.

### Ce qui n'est pas retenu

1. une deuxième page carrière ;
2. une réplique dense de la fresque Career Rank de Spartan Record ;
3. une visualisation trop lourde en lecture par défaut.

### Traduction concrète

La direction recommandée pour `CareerPage` est :

1. garder la base `summary + hero progress + charts + lusr` ;
2. ajouter un module `Hero Tracker` secondaire, pliable ou contenu dans une section dédiée ;
3. éviter toute surcharge qui ferait concurrence à la progression principale.

### Ce qui doit s'effacer pour faire la place

Avant d'ajouter ce `Hero Tracker`, la page doit perdre :

1. `CareerTopMatchesTable` ;
2. `CareerEncountersSection`.

Sinon, on ajoute encore un bloc à une page qui raconte déjà trop de choses.

## 5. Médailles : améliorer la hiérarchie, pas supprimer le delta

Inspiration retenue : la structure `Top Medals` de Spartan Record est plus lisible et plus propre que la surface actuelle.

### Cible LevelUp long terme

Dans le hub `Carrière > Citations` :

1. mieux hiérarchiser les médailles ;
2. mettre en avant un cabinet `Top Medals` ;
3. conserver la lecture de maîtrise durable.

### Cible LevelUp analytique

Dans la lecture `solo / escouade`, le delta reste pertinent.

La bonne traduction n'est donc pas de supprimer le delta partout, mais de séparer les usages :

1. `Carrière / Citations` = lecture durable, cabinet propre, progression, distribution ;
2. `Synthèse` ou bloc analytique solo / escouade = médailles contextuelles avec delta conservé.

### Décision produit

Le pattern à viser est :

1. une surface `Top Medals` plus premium dans `Citations` ;
2. un mini-bloc `médailles solo vs escouade` avec delta dans `Synthèse`, pas dans `Carrière`.

### Ce qui doit s'effacer pour faire la place

La contrepartie recommandée est :

1. ne plus laisser le trio `Delta` monopoliser l'ouverture de la page si on veut une vraie entrée premium sur les médailles ;
2. placer `Distribution` après les sections de capital principal, pas comme point d'entrée obligatoire ;
3. éviter de créer un second bloc top-level de médailles sans retrait de poids ailleurs.

## 6. Tuiles de matchs : oui en complément, non en remplacement du tableau

Inspiration retenue : les cartes de match de Spartan Record apportent une lecture plus visuelle, notamment avec la vignette de map.

### Cible LevelUp

Deux usages pertinents :

1. preview de matchs récents plus riche sur la home ;
2. vue optionnelle `cards` pour l'historique des matchs.

### Recommandation précise pour la home

Si les tuiles remplacent la logique actuelle `Dernier match` autonome, la home ne doit pas devenir un mini historique paginé.

La bonne cible est :

1. **4 tuiles maximum** sur desktop ;
2. affichage responsive plus court sur mobile ;
3. aucune pagination dans la home ;
4. un CTA clair vers l'historique complet.

Pourquoi 4 :

1. au-delà, le bloc prend trop de hauteur ;
2. la home a déjà beaucoup d'autres responsabilités ;
3. 4 suffisent pour donner une sensation de récence sans concurrence avec `MatchHistory`.

### Pagination recommandée

#### Sur la home

Pas de pagination.

Pas de scroll infini.

Pas de remontée historique longue.

La home montre simplement :

1. les 4 matchs les plus récents ;
2. un accès vers l'historique si l'utilisateur veut aller plus loin.

#### Sur l'historique des matchs

Si une vue cartes est ajoutée à `MatchHistoryPage`, elle doit réutiliser la logique paginée existante, pas inventer un scroll infini parallèle.

Direction recommandée :

1. garder la même pagination serveur que la vue table ;
2. éventuellement abaisser le nombre d'items par page en mode cartes si 50 est trop dense visuellement ;
3. remonter aussi loin que l'historique paginé disponible, mais **jamais** en scroll infini sur la home.

### Décision produit

La distinction doit être nette :

1. `Home` = aperçu récent de 4 matchs ;
2. `MatchHistory` = exploration paginée complète ;
3. pas de pagination infinie dans la home.

### Règle

Le tableau actuel reste la surface canonique pour :

1. tri ;
2. densité ;
3. export ;
4. inspection rapide de gros volumes.

Les tuiles servent à :

1. rendre les previews plus désirables ;
2. mieux exploiter les visuels de map ;
3. créer une alternative de lecture, pas un remplacement imposé.

### Ce qui doit s'effacer pour faire la place

Sur la home, une preview en tuiles de match rend la carte `Dernier match` autonome encore moins nécessaire.

Dans l'historique, en revanche, rien n'a besoin de disparaître : la bonne UX est un switch `table / cartes`, pas un remplacement.

## 7. Carte `Data Set` sur l'accueil

Inspiration retenue : Spartan Record rend plus explicite ce qu'il montre et d'où ça vient.

### Cible LevelUp

Ajouter une carte `Data Set` sur `HomePage`.

### Contenu recommandé

1. nombre de matchs inclus dans le scope d'accueil ;
2. nombre de matchs exclus manuellement s'il existe ;
3. date ou label de dernière sync ;
4. statut des données live si pertinent ;
5. éventuel niveau de couverture des données si certaines briques sont best-effort.

### Règle

Le bloc doit rester une carte de confiance produit, pas une page d'administration déguisée.

Il doit répondre à :

> qu'est-ce que je regarde exactement ?

## 8 bis. `Médias récents` sur la home : rail compact, preview et lightbox

Constat : la home liste déjà des médias récents, mais sous une forme trop pauvre pour jouer un vrai rôle produit.

La bonne direction n'est pas une mini-galerie complète sur l'accueil.

La bonne direction est un **rail compact** de contenus récents qui donne envie d'ouvrir la galerie complète.

### Cible LevelUp

Sur `HomePage`, le bloc `Médias récents` doit devenir :

1. une seule ligne de médias récents ;
2. **4 éléments maximum** ;
3. une preview riche, mais courte ;
4. un CTA vers la page `Médias` complète.

### Pourquoi 4 et pas 6

1. 6 prend trop de place dans une home déjà chargée ;
2. 4 suffit pour donner une sensation de fraîcheur ;
3. au-delà, on recrée une galerie miniature concurrente de la vraie page média.

### Comportement visuel cible

Chaque vignette doit proposer :

1. une miniature statique par défaut ;
2. pour les clips, une preview animée au survol ;
3. un bouton de `like / unlike` visible directement sur la vignette ;
4. un clic qui ouvre le média en grand.

### Viewer cible

Le bon pattern ici reste bien une `lightbox` moderne.

Pas besoin de chercher une techno plus exotique tant que le comportement est net.

Le viewer ouvert doit permettre :

1. lecture automatique du clip à l'ouverture si le navigateur l'autorise ;
2. navigation manuelle `précédent / suivant` s'il existe d'autres médias dans le rail ou la galerie ;
3. `like / unlike` visible aussi dans la vue ouverte ;
4. fermeture simple (`Escape`, clic externe, bouton fermer).

### Règle produit

Le rail home est une **surface d'appel**, pas la galerie de référence.

Donc :

1. pas de pagination dans la home ;
2. pas de deuxième page cachée dans l'accueil ;
3. pas de grille de 12 médias au-dessus de la ligne de flottaison.

### Ce qui doit s'effacer pour faire la place

Le bloc actuel de médias récents en pseudo-grille pauvre doit disparaître au profit d'un rail plus premium.

### Règle de persistance des likes

À court terme, un `like` local navigateur est acceptable pour le confort personnel.

Mais :

1. cela ne doit pas être confondu avec un vrai compteur partagé ;
2. si le produit veut afficher un **nombre réel de likes**, il faudra un stockage backend dédié ;
3. la home peut afficher l'état `aimé / non aimé` avant que ce chantier backend existe.

## 9. Images de maps : cache-aside oneshot via la couche API Go

Décision : c'est une priorité technique raisonnable, parce qu'elle évite de rester bloqué à chaque nouvelle map.

### Constat

Le stockage manuel local a un coût d'entretien inutile.

### Direction cible

**Pattern oneshot cache-aside** : une image de map est fetchée depuis Waypoint la première fois qu'elle est demandée (ou si la qualité enregistrée est insuffisante), stockée localement dans `metadata.duckdb`, puis servie depuis le cache sans re-fetch réseau.

Principes :

1. Le **frontend** appelle simplement `/api/maps/{title_id}/{map_id}/image` — il ne sait pas si l'image vient du disque ou du réseau.
2. Le **handler Go** fait : check registry → fetch Waypoint si absent ou qualité insuffisante → écriture locale → réponse.
3. Le **pipeline sync** ne gère pas les assets visuels — ce n'est pas son rôle.
4. **Multi-titres dès le départ** : `title_id` est une clé de partition dans le registry, pas une constante hardcodée. Un nouveau titre Halo est un nouveau scope, pas un patch.
5. **Singleflight** côté Go : si 4 tuiles de match demandent la même map simultanément au premier accès, une seule requête Waypoint part — les autres attendent le résultat.

### Registry

Table dans `metadata.duckdb` :

```
map_images_registry(title_id, map_id, local_path, source_url, quality_level, fetched_at, content_hash)
```

`quality_level` permet de re-fetcher si une meilleure résolution devient disponible sans tout retélécharger.

### Règles

1. pas de fetch directement dans le navigateur comme source de vérité ;
2. pas de dépendance à des chemins téléchargés à la main ;
3. compatibilité avec l'arrivée de nouvelles maps sans patch manuel obligatoire ;
4. même pattern appliqué aux images de médailles (D1) et aux assets généraux (D2) dans le Sprint 54.

### Lien avec les chantiers existants

Cette direction s'aligne avec `waypoint_assets_raw`, `waypoint_medals_raw` et le pattern de cache-aside déjà retenu pour D1/D2 dans le Sprint 54.

## Surfaces cibles

## Arbitrage final par page

### `HomePage`

Ajouter :

1. `Spartan ID`
2. résumé service record plus net
3. `Data Set`
4. preview matchs plus visuelle

Compresser ou retirer :

1. `Performance globale` telle qu'elle existe aujourd'hui
2. la carte autonome `Dernier match`
3. éventuellement `Sessions récentes` et `Points saillants` comme deux blocs séparés

Précision :

1. le `Spartan ID` peut rester compact et séparé ;
2. ce n'est pas une obligation de fusionner toute l'identité et toute la performance dans un seul bloc massif ;
3. le vrai enjeu est la hiérarchie, pas la fusion pour la fusion.

Déclasser :

1. `Battle Pass`
2. `Défis`
3. `Accès rapide`

### `CareerPage`

Ajouter :

1. `Hero Tracker`

Retirer :

1. `Top matchs`
2. `Rencontres fréquentes`

### `CitationsPage`

Ajouter :

1. hiérarchie `Top Medals`
2. lecture cabinet plus premium

Compresser ou décaler :

1. `Delta`
2. `Distribution`

### `SynthesisPage`

Ajouter :

1. bloc médailles `solo / escouade` avec delta si utile

Compresser ou décaler :

1. la table détaillée solo / escouade si la page gagne `overview + highlights + rivalries`
2. une partie des lectures secondaires au-dessous de la ligne de flottaison

### `MatchHistoryPage`

Ajouter :

1. vue cartes optionnelle

Conserver tel quel :

1. le tableau canonique
2. la logique d'export
3. la densité de lecture existante

### Accueil / Record

Fichier principal : `apps/web/src/features/home/HomePage.tsx`

Ajouts visés :

1. `Spartan ID` compact ;
2. blocs service record plus lisibles ;
3. carte `Data Set` ;
4. preview `Top Medals` ;
5. preview matchs en tuiles enrichies si les vignettes de map sont prêtes.

### Carrière

Fichier principal : `apps/web/src/features/career/CareerPage.tsx`

Ajouts visés :

1. `Hero Tracker` ;
2. éventuel bloc rang plus élégant ;
3. pas de duplication de la home record ;
4. pas de seconde page carrière.

### Citations

Fichier principal : `apps/web/src/features/citations/CitationsPage.tsx`

Ajouts visés :

1. hiérarchie `Top Medals` plus premium ;
2. meilleure structuration des familles et de la distribution ;
3. préservation du rôle long terme de la page.

### Synthèse

Fichier principal : `apps/web/src/features/synthesis/SynthesisPage.tsx`

Ajouts visés :

1. bloc médailles `solo vs escouade` avec delta ;
2. lecture contextuelle, distincte du cabinet durable de `Citations`.

### Historique des matchs

Fichiers principaux :

- `apps/web/src/features/match-history/MatchHistoryPage.tsx`
- `apps/web/src/features/match-history/MatchHistoryTable.tsx`

Ajouts visés :

1. vue cartes optionnelle ;
2. conservation du tableau comme surface canonique.

## Données et contrats à prévoir

### Pour le `Spartan ID`

Le contrat home devra pouvoir exposer proprement :

1. emblem URL ;
2. backdrop ou banner URL ;
3. service tag ou identifiant joueur lisible ;
4. rang carrière et ornement éventuel ;
5. un comportement cohérent si seules certaines données enrichies sont disponibles.

### Règle de contrat backend

Le backend devrait idéalement exposer une structure dédiée de type `spartan_identity` ou équivalent, au lieu d'un assemblage opportuniste de champs frontend.

Cette structure doit rendre possible :

1. un état premium complet si l'auth Halo est disponible ;
2. un état enrichi partiel si seules certaines données existent ;
3. un fallback public minimal sans casser la carte.

### Pour la carte `Data Set`

Le contrat home devra pouvoir exposer :

1. compte de matchs inclus ;
2. compte de matchs exclus ;
3. fraîcheur de sync ;
4. éventuels hints de couverture ou de dégradation.

### Pour les tuiles de match

Il faut une vignette ou image de map disponible dans la payload match, avec fallback clair.

### Pour les visuels dynamiques

Il faut une source de vérité metadata côté Go, pas seulement une convention frontend.

## Ordre de livraison recommandé

### Lot 1 — Ajouts à plus forte valeur visible

1. `Spartan ID` compact sur la home ;
2. carte `Data Set` ;
3. petit enrichissement des blocs service record de la home.

### Lot 2 — Visuels et matchs

1. stratégie images de maps dynamiques ;
2. preview matchs en tuiles sur la home ;
3. éventuelle vue cartes dans l'historique.

### Lot 3 — Capital médailles et progression

1. refonte de la hiérarchie `Top Medals` ;
2. bloc médailles solo / escouade avec delta ;
3. module `Hero Tracker` dans `Carrière`.

## Critères d'acceptation

1. La page d'accueil actuelle reste reconnaissable et n'est pas remplacée par un clone de Spartan Record.
2. Un `Spartan ID` compact améliore immédiatement l'identité joueur au-dessus du contenu.
3. Aucun toggle global `Overall / Per Match` n'est introduit par défaut.
4. `Carrière` n'est pas dupliquée ; seul un `Hero Tracker` ciblé y est ajouté.
5. Les médailles gagnent une meilleure hiérarchie sans perdre le delta dans la lecture analytique solo / escouade.
6. Les tuiles de match restent un complément visuel et ne suppriment pas le tableau canonique.
7. La home comporte une carte `Data Set` expliquant ce que l'utilisateur regarde.
8. Les visuels de map s'appuient sur une stratégie dynamique ou semi-dynamique pilotée par les métadonnées.

## Décision opérationnelle

La ligne retenue est la suivante :

- garder la page record actuelle ;
- l'enrichir avec un `Spartan ID`, une meilleure hiérarchie de blocs et une carte `Data Set` ;
- améliorer les médailles et les cartes de match là où cela renforce réellement la lecture ;
- traiter les images de map comme un sujet metadata produit, pas comme une collection manuelle d'assets.

Pour le `Spartan ID`, la règle est maintenant explicite :

- s'inspirer du design Spartan Record ;
- ne pas hériter de sa limite `public-only` ;
- brancher le composant sur les meilleures données LevelUp disponibles via auth Halo, cache MSAL, session et enrichissements locaux.

Autrement dit :

**Spartan Record sert ici de source d'inspiration ciblée, pas de modèle à recopier.**
