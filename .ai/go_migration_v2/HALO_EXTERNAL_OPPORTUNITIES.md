# HALO_EXTERNAL_OPPORTUNITIES.md — Opportunites externes Halo et plans d'implementation

> Document de cadrage issu de la revue de trois repos externes :
>
> 1. SpartanRecord
> 2. halo-infinite-api
> 3. HaloInfiniteGetter
>
> Objectif : transformer les idees utiles en opportunites concretes pour LevelUp-go-migration,
> avec un plan d'implementation exploitable par lot.

## Role du document

Ce document sert a trois choses :

1. lister les opportunites produit, data et outillage issues de projets externes ;
2. prioriser ce qui a le meilleur ratio valeur / effort pour LevelUp ;
3. proposer, pour chaque opportunite, un plan d'implementation adapte a l'architecture Go + React du repo.

## Hypothese de travail

Les opportunites ci-dessous sont pensees pour la cible runtime actuelle :

1. backend Go dans `apps/go-api/` ;
2. frontend React dans `apps/web/` ;
3. DuckDB comme source de verite locale ;
4. contrats API internes orientes parcours utilisateur, pas orientes endpoints Waypoint.

## Mise a jour apres arbitrage produit

Suite au retour produit du 2026-04-18, ce document est reserre autour de trois decisions :

1. O4 Match count endpoint est ecarte : la valeur est trop faible avec vos filtres et votre systeme d'exclusion.
2. O6 Export service record / profil est ecarte : faible valeur percue pour LevelUp.
3. O10 Store tracker et O11 Social layer passent en parking : hors coeur stats tant que le positionnement produit n'evolue pas.

Les opportunites encore actives sont donc jugees selon deux criteres :

1. renforcer les surfaces existantes sans ajouter de menu ;
2. fiabiliser la couche metadata / provider qui alimente deja le produit.

## Principes d'atterrissage UI

Pour eviter de surcharger la navigation, les opportunites retenues doivent suivre ces regles :

1. O2, O3 et O8 ne meritent aucun menu : ce sont des briques invisibles qui alimentent `career`, `match-history`, `match-view` et les futures pages derivees.
2. O5 Compare doit vivre comme un mode contextuel declenche depuis une surface existante du scope joueur, par exemple `career`, `home`, `squad` ou `profile/citations`, pas comme une rubrique globale supplementaire.
3. O7 Leaderboards doit commencer comme un bloc ou une vue secondaire dans `career` ou `home`, pas comme une section autonome.
4. O9 Year in Review peut exister comme route partageable ou campagne saisonniere, sans entree permanente dans le shell principal.
5. Toute nouvelle surface produit doit d'abord prouver sa valeur comme sous-parcours dans une page existante avant de gagner un point d'entree dedie.

## Resume executif

### Valeur immediate retenue

1. exposer la privacy des matchs ;
2. fiabiliser les metadonnees saisons / CSR seasons ;
3. enrichir les metadonnees medailles ;
4. lancer un Compare MVP sans nouveau menu.

### Valeur transverse utile a moyen terme

1. asset discovery + versioning ;
2. Waypoint Explorer interne ;
3. cache ETag et snapshots de ressources.

### Idees a garder en parking produit

1. leaderboards CSR, seulement si un atterrissage discret est valide ;
2. Year in Review, sous forme de campagne ou route partageable ;
3. store tracker, seulement si LevelUp elargit son scope au-dela des stats ;
4. social layer type Spartan Company, seulement si une vraie proposition groupe emerge.

### Valeur outillage interne

1. Waypoint Explorer interne ;
2. cache ETag et snapshots de ressources ;
3. asset discovery/versioning pour audits metadata ;
4. diagnostics de bans et limitations provider.

## Vagues recommandees

| Vague | Horizon | Opportunites | But principal |
|-------|---------|--------------|---------------|
| V1 | court terme | privacy, calendars, medals metadata, compare MVP | gains rapides cote UX et fiabilite metadata |
| V2 | court / moyen terme | compare etendu, asset discovery, Waypoint Explorer, ETag snapshots | renforcer les surfaces existantes et l'outillage |
| V3 | moyen terme | CSR leaderboards, Year in Review, ban diagnostics | ajouter des experiences secondaires sans nouveau menu |
| V4 | conditionnel | store tracker, social layer | a n'ouvrir que si le scope produit evolue |

## Inventaire priorise

| ID | Opportunite | Source externe | Type | Statut apres revue | Commentaire |
|----|-------------|----------------|------|--------------------|-------------|
| O1 | Match privacy | halo-infinite-api, SpartanRecord | produit + data | retenu V1 | visible dans bootstrap + pages match/history |
| O2 | Season calendars + CSR calendars | halo-infinite-api | metadata | retenu V1 | infra invisible qui nourrit les ecrans existants |
| O3 | Medals metadata Waypoint | halo-infinite-api | metadata | retenu V1 | enrichissement de metadata, pas de menu |
| O4 | Match count endpoint | halo-infinite-api | perf + UX | ecarte | peu utile avec filtres + exclusions |
| O5 | Compare joueur vs joueur | SpartanRecord | produit | retenu V1 | mode contextuel dans un parcours joueur existant |
| O6 | Export service record / profil | SpartanRecord | produit | ecarte | faible valeur percue |
| O7 | CSR leaderboards | SpartanRecord | produit | parking V3 | a tester comme bloc secondaire, pas comme menu |
| O8 | Asset discovery + versioning | halo-infinite-api | metadata + tooling | retenu V2 | socle metadata transverse |
| O9 | Year in Review | SpartanRecord | produit | parking V3 | route partageable ou campagne, pas navigation fixe |
| O10 | Store / economy tracker | SpartanRecord | produit | parking lointain | hors coeur stats aujourd'hui |
| O11 | Spartan Company / social layer | SpartanRecord | produit | parking lointain | a rabattre sur squad si un jour retenu |
| O12 | Ban diagnostics | halo-infinite-api | support + ops | option admin | utile hors UI publique |
| O13 | Waypoint Explorer interne | HaloInfiniteGetter | dev tooling | retenu V2 | accelerateur de spikes et metadata |
| O14 | ETag cache + snapshots | HaloInfiniteGetter | dev tooling | retenu V2 | veille reproducible des ressources critiques |

---

## O1 — Match privacy

### Pourquoi c'est interessant

1. permet d'expliquer proprement pourquoi certaines donnees match sont absentes ;
2. aligne LevelUp avec un besoin UX deja visible chez SpartanRecord ;
3. evite de traiter des trous de donnees comme des erreurs produit vagues.

### Signal externe utile

1. `GET /hi/players/{xuid}/matches-privacy` ;
2. `PUT /hi/players/{xuid}/matches-privacy` dans `halo-infinite-api`.

### Plan d'implementation

1. Ajouter un provider interne Go pour lire la privacy des matchs depuis le backend Halo.
2. Etendre le modele canonique ou le read model bootstrap avec un champ de visibilite match.
3. Exposer une information lisible dans `GET /api/v1/bootstrap` et dans les pages dependantes de l'historique.
4. Afficher un bandeau explicite dans le frontend React quand l'historique est prive ou partiellement exploitable.
5. Ajouter un warning structure dans les payloads `match-history`, `match-view` et `explorer`.

### Chantiers techniques

1. `apps/go-api/internal/service/bootstrap_service.go`
2. `apps/go-api/internal/api/handlers/`
3. `apps/web/src/features/*`
4. eventuelle persistance DuckDB dans une table metadata joueur si on veut memoriser le dernier etat observe.

### Validation

1. cas compte public ;
2. cas compte prive ;
3. cas transition public -> prive ;
4. golden values de warning au lieu d'une erreur generique.

---

## O2 — Season calendars + CSR calendars

### Pourquoi c'est interessant

1. fiabilise les saisons et periodes CSR ;
2. reduit le hardcode metier ;
3. aide Compare, Year in Review, historique, rankings et projections.

### Signal externe utile

1. `SeasonCalendar.json` ;
2. `CsrSeasonCalendar.json` exposes par `halo-infinite-api`.

### Plan d'implementation

1. Creer un fetcher metadata dedie cote Go pour les calendars.
2. Persister les resultats dans `metadata.duckdb` avec version, fetched_at et content_hash.
3. Ajouter un job de refresh manuel ou planifie, isole de l'API publique.
4. Brancher les services career, stats et compare sur ces tables plutot que sur des hypotheses statiques.
5. Ajouter une politique de fallback sur les donnees locales si le provider est indisponible.

### Materialisation UI recommandee

1. aucune entree de menu dediee ;
2. utiliser ces donnees pour fiabiliser les filtres et libelles deja presents dans `career`, `match-history` et les futures vues compare ;
3. si une exposition explicite devient necessaire, la limiter a un selecteur de saison dans un ecran existant.

### Chantiers techniques

1. migration metadata DuckDB ;
2. repository Go metadata ;
3. service de refresh ;
4. tests de mapping `provider -> metadata.duckdb`.

### Validation

1. table de seasons cohherente avec les saisons visibles dans l'UI ;
2. regression tests sur current season ;
3. tests sur plages annuelles 2024/2025 pour Year in Review.

---

## O3 — Medals metadata Waypoint

### Pourquoi c'est interessant

1. enrichit les labels, categories et assets medailles ;
2. peut ameliorer citations, commendations et Year in Review ;
3. permet de verifier ou completer la metadata locale actuelle.

### Signal externe utile

1. `Waypoint/file/medals/metadata.json` expose via `halo-infinite-api`.

### Plan d'implementation

1. Ajouter un importeur de metadata medailles vers `metadata.duckdb`.
2. Definir une table versionnee ou un enrichissement de la table existante des medailles.
3. Mapper icones, rarete, groupes et labels si disponibles.
4. Rebrancher citations/commendations sur cette source enrichie.
5. Prevoir un fallback local si une cle manque ou si le JSON externe change.

### Validation

1. comparaison de cardinalite entre metadata actuelle et metadata Waypoint ;
2. tests de rendu frontend sur medaille connue, medaille inconnue, icone absente.

---

## O4 — Match count endpoint

### Decision produit actuelle

Cette opportunite est ecartee pour l'instant.

### Pourquoi on l'ecarte

1. vos filtres et exclusions rendent le compteur brut peu exploitable ;
2. un total provider risque d'etre moins utile qu'un total calcule dans le scope reel LevelUp ;
3. la complexite de reconciliation ne se justifie pas a ce stade.

### Si le sujet revient plus tard

1. partir d'un besoin produit explicite sur un total filtre stable ;
2. definir d'abord la semantique LevelUp avant de reconnecter un endpoint provider externe.

---

## O5 — Compare joueur vs joueur

### Pourquoi c'est interessant

1. forte valeur produit immediate ;
2. surface differenciante par rapport aux pages deja portees ;
3. faible dependance au runtime legacy si on reste sur des KPIs lisibles.

### Signal externe utile

1. CompareView dans SpartanRecord ;
2. comparaison de rank, matches, winrate, KDA, KDR, CSR, damage, accuracy.

### Plan d'implementation

1. Definir un contrat `POST /api/v1/players/{player_slug}/pages/compare` ou une route dediee symetrique a confirmer.
2. Construire un read model compare a partir de deux joueurs et d'un contexte de filtres commun.
3. Commencer par un MVP sur 10 a 12 KPIs stables : matches, winrate, KDA, KDR, kills/game, deaths/game, assists/game, CSR current, CSR best, accuracy, damage/game, career rank.
4. Ajouter ensuite les variantes par periode, playlist et session.
5. Cote React, creer une feature `compare` avec selection du second joueur, recents, et etat vide.

### Materialisation UI recommandee

1. ajouter un CTA `Comparer` dans l'en-tete joueur de `career` ou `profile`, ouvrant un drawer, un sheet ou un mode secondaire ;
2. ajouter le meme point d'entree depuis `squad` et plus tard depuis une ligne de leaderboard ;
3. eviter tout item de navigation global tant que la feature n'a pas prouve son usage.

### Chantiers techniques

1. contrat OpenAPI ;
2. service Go `CompareService` ;
3. repository pour charger les deux joueurs avec un filtre commun ;
4. page React et composants de comparaison ;
5. gestion des joueurs prives et des donnees partielles.

### Validation

1. golden values sur un duo de joueurs de reference ;
2. tests UI sur joueur absent, joueur prive, joueur identique, donnees asymetriques.

---

## O6 — Export service record / profil

### Decision produit actuelle

Cette opportunite est ecartee pour l'instant.

### Pourquoi on l'ecarte

1. la valeur utilisateur parait faible par rapport au reste du backlog ;
2. l'export d'historique couvre deja le besoin d'export principal.

---

## O7 — CSR leaderboards

### Pourquoi c'est interessant

1. ouvre une surface communautaire ;
2. donne une valeur reutilisable pour home, squad et compare ;
3. exploite naturellement les seasons CSR.

### Signal externe utile

1. pages Leaderboards de SpartanRecord ;
2. CSR leaderboard consomme depuis leur couche data.

### Plan d'implementation

1. Cadrer un leaderboard limite au perimetre LevelUp ou a un corpus de joueurs connus, pas a la population Halo complete au debut.
2. Definir des classements par playlist, saison CSR et region logique si applicable.
3. Exposer d'abord un bloc secondaire dans `home` ou `career`, avant toute page dediee.
4. Ajouter filtres de saison, playlist et delta recent.

### Materialisation UI recommandee

1. commencer par un module compact `Top CSR de ta cohorte` dans `home` ou `career` ;
2. ajouter un lien `Voir plus` vers une route secondaire seulement si le bloc est vraiment consulte ;
3. reutiliser le meme parcours comme point d'entree vers Compare.

### Validation

1. tests sur tri, egalites, pagination et saison vide ;
2. verification metadonnees CSR via O2.

---

## O8 — Asset discovery + versioning

### Pourquoi c'est interessant

1. utile pour maps, playlists, map-mode pairs et UGC ;
2. accelere la maintenance metadata ;
3. facilite des pages plus riches sans hardcode fragile.

### Signal externe utile

1. `getAsset` ;
2. `getSpecificAssetVersion` ;
3. asset kinds exposes par `halo-infinite-api`.

### Plan d'implementation

1. Ajouter une couche interne de discovery metadata non exposee directement aux utilisateurs.
2. Persister asset_id, version_id, kind, labels et dates utiles dans `metadata.duckdb`.
3. Construire un pipeline de refresh incremental par type d'asset.
4. Reutiliser ces assets dans match view, playlists et store tracker.

### Validation

1. tests de mapping par `AssetKind` ;
2. policy de cache et invalidation documentee.

---

## O9 — Year in Review

### Pourquoi c'est interessant

1. forte valeur narrative et partageable ;
2. reutilise beaucoup de briques deja presentes ;
3. donne un produit signature si le rendu est soigne.

### Signal externe utile

1. page `YearInReview.tsx` de SpartanRecord ;
2. callouts pour playtime, matches, career rank, medals, kill breakdown.

### Plan d'implementation

1. Cadrer un MVP annuel avec une seule annee supportee au debut.
2. Definir un endpoint `GET /api/v1/players/{player_slug}/pages/year-in-review?year=YYYY`.
3. Reutiliser O2 pour reconstituer les saisons couvrant une annee donnee.
4. Agreger : matchs joues, temps de jeu, winrate, KDA, armes ou medailles marquantes, progression de rang, meilleurs matchs.
5. Cote React, privilegier une page partageable plutot qu'un simple dashboard dense.

### Materialisation UI recommandee

1. ne pas ajouter ce parcours dans le menu principal ;
2. le lancer depuis une carte temporaire sur `home` ou `career` quand l'annee est disponible ;
3. assumer une route deep-linkable partageable, mais decouverte contextuelle.

### Validation

1. snapshots de payload pour une annee connue ;
2. tests sur annee vide ;
3. tests sur annee partiellement couverte.

---

## O10 — Store / economy tracker

### Decision produit actuelle

Cette opportunite passe en parking lointain.

### Pourquoi la mettre en parking

1. le sujet s'eloigne du coeur analytics de LevelUp ;
2. aucun atterrissage UI propre n'apparait sans diluer la promesse produit.

### Si le sujet revient plus tard

1. le traiter comme un module optionnel dans `home`, jamais comme un menu prioritaire ;
2. n'ouvrir le chantier que si une proposition valeur claire est confirmee.

---

## O11 — Spartan Company / social layer

### Decision produit actuelle

Cette opportunite passe en parking lointain.

### Pourquoi la mettre en parking

1. la place naturelle dans l'UI n'est pas claire aujourd'hui ;
2. la feature risque de recreer un menu ou un sous-produit social sans validation utilisateur.

### Si le sujet revient plus tard

1. le rabattre d'abord sur `squad` sous forme de groupes favoris ou cohortes sauvegardees ;
2. ne jamais commencer par une rubrique sociale autonome.

---

## O12 — Ban diagnostics

### Pourquoi c'est interessant

1. utile pour support et diagnostic ;
2. peut expliquer des comportements anormaux cote provider ;
3. faible priorite produit mais forte utilite ops.

### Signal externe utile

1. `bansummary` ;
2. `banning/file/{banPath}` exposes par `halo-infinite-api`.

### Plan d'implementation

1. Garder cela hors UI publique dans un premier temps.
2. Ajouter un outil admin ou debug route derriere feature flag.
3. Exposer seulement un diagnostic lisible, pas les payloads bruts par defaut.

### Validation

1. tests sur 404 / 403 / message absent ;
2. audit de ce qui est acceptable d'afficher en UI admin.

---

## O13 — Waypoint Explorer interne

### Pourquoi c'est interessant

1. reduit le cout de decouverte de nouvelles ressources ;
2. accelere les spikes sans bricoler des scripts ad hoc ;
3. se marie tres bien avec votre couche `src/ai/` et vos docs .ai.

### Signal externe utile

1. mode GET / SCAN de HaloInfiniteGetter ;
2. navigation ressource -> sous-ressources.

### Plan d'implementation

1. Creer un outil interne, pas une feature utilisateur finale.
2. Le cadrer comme un panneau dev ou une route admin protegee.
3. Fournir deux operations : fetch d'une ressource exacte et scan recursif d'un JSON.
4. Stocker les ressources dans un cache local versionne et navigable.
5. Permettre l'export d'un snapshot pour fixtures et audits.

### Chantiers techniques

1. route admin Go ou script CLI ;
2. stockage local sous `data/cache/waypoint_explorer/` ou equivalent ;
3. UI minimale React optionnelle, ou simple CLI au debut.

### Validation

1. scan d'un calendar ;
2. detection des ressources deja vues ;
3. export d'un snapshot reproductible.

---

## O14 — ETag cache + snapshots de ressources

### Pourquoi c'est interessant

1. suit l'evolution des ressources Waypoint ;
2. aide a detecter les regressions provider ;
3. transforme la veille manuelle en pipeline reproductible.

### Signal externe utile

1. `old_files`, `etags.json`, update cached resources dans HaloInfiniteGetter.

### Plan d'implementation

1. Ajouter un cache HTTP interne qui memorise ETag, fetched_at, source_url et content_hash.
2. Stocker les anciennes versions seulement pour les ressources a haute valeur : seasons, CSR calendars, medals metadata, playlists, map-mode pairs.
3. Ajouter un diff simple entre version N et N+1.
4. Eventuellement brancher une notification Discord si une ressource critique change.

### Validation

1. test sur 304 Not Modified ;
2. test sur changement d'ETag ;
3. test de restauration d'un snapshot local.

---

## Ordre d'implementation recommande

### Lot A — 2 a 3 sprints

1. O2 Season calendars + CSR calendars
2. O3 Medals metadata
3. O1 Match privacy
4. O5 Compare MVP

### Lot B — 2 sprints

1. O8 Asset discovery
2. O13 Waypoint Explorer
3. O14 ETag snapshots

### Lot C — 2 a 3 sprints

1. O7 CSR leaderboards si le bloc home/career prouve sa valeur
2. O9 Year in Review si la campagne shareable est jugee prioritaire
3. O12 Ban diagnostics

### Lot D — optionnel

1. O10 Store tracker
2. O11 Social layer

## Recommendation finale

Si l'objectif est de maximiser la valeur rapidement sans disperser le chantier, l'ordre ideal est :

1. fiabiliser les metadonnees externes ;
2. exposer proprement la privacy ;
3. lancer Compare comme mode contextuel depuis `career`, `squad` ou `profile/citations` plutot que comme destination autonome ;
4. monter le socle outillage metadata en parallele ;
5. n'ouvrir leaderboards et Year in Review qu'une fois leur atterrissage UI discret valide.

## Livrables conseilles pour la suite

1. une ADR courte pour valider les opportunites retenues en V1 ;
2. un mini sprint plan dedie a Compare ;
3. un mini sprint plan dedie au pipeline metadata externe ;
4. une note UX courte listant les points d'entree contextuels retenus pour Compare, leaderboards et Year in Review ;
5. un lot outillage interne Waypoint Explorer derriere feature flag admin.

## Note UX et direction design

Cette section ne propose pas de code ni de backlog supplementaire. Elle fixe une direction de design pour que les opportunites retenues renforcent le produit sans degrader la navigation.

Important : a ce stade, l'application ne semble pas avoir une page `joueur` autonome au sens strict. Elle a surtout un scope joueur avec plusieurs destinations dediees comme `home`, `career`, `stats/history`, `squad`, `media` ou `profile/citations`. Les recommandations ci-dessous parlent donc d'abord de points d'entree dans ces surfaces existantes.

### Principes directeurs

1. moins de destinations, plus de profondeur : une page doit ressembler a un lieu fort, pas a un simple support de widgets ;
2. une page = une intention dominante : on evite les surfaces qui racontent trois histoires en meme temps ;
3. densite oui, bruit non : l'information doit etre structuree par rythme visuel, tailles, contrastes et respirations ;
4. les parcours secondaires doivent etre contextuels : drawer, sheet, tabs, panneaux lies a la page courante avant de meriter une route dediee ;
5. le ton visuel doit etre premium et assume : interface nette, technique, precise, avec une personnalite Halo implicite mais jamais cosplay ;
6. toute nouvelle feature doit renforcer le sentiment de lecture, de maitrise et de progression, pas seulement ajouter des chiffres.

### Langage visuel recommande

1. privilegier une base claire et tendue, avec contraste net et surfaces bien delimitees ;
2. utiliser une palette sobre : fonds lumineux ou graphite clair, accents acier, olive, sable, cyan technique ou orange signal selon les cas ;
3. eviter les couleurs gadgets ou la surabondance d'etats colores concurrents ;
4. travailler une typographie plus expressive que le simple stack neutre, avec une hiérarchie marquee entre titres, metriques et texte de contexte ;
5. utiliser les animations avec parcimonie : reveal a l'arrivee, transitions de panneaux, skeletons soignes, jamais de micro-animations gratuites ;
6. faire sentir la notion de mission, de theatre d'operations et de session sans verser dans le skeuomorphisme militaire caricatural.

### Hierarchie de navigation recommandee

1. la navigation primaire doit rester courte et memorisable ;
2. les parcours `Compare`, `Leaderboards` et `Year in Review` ne doivent pas entrer dans la navigation principale au premier niveau ;
3. `Compare` doit etre invoque depuis une surface existante du scope joueur, idealement `Career`, `Squad` ou `Profile/Citations` ;
4. `Leaderboards` doit emerger comme vue secondaire depuis `home` ou `career` ;
5. `Year in Review` doit etre decouvert via une campagne ou une carte saisonniere, puis vivre comme destination partageable ;
6. si la navigation principale parait deja trop dense, il faut envisager a terme une consolidation de certaines rubriques avant toute expansion.

### Page par page

#### Home

Role : page d'entree, de situation et d'orientation.

1. la page doit raconter l'etat du joueur en quelques secondes : forme recente, objectifs, signaux importants, points d'entree prioritaires ;
2. le hero ne doit pas etre un simple bandeau decoratif, mais un resume clair de la situation du moment ;
3. les modules doivent etre peu nombreux mais forts : recent performance, dernier match important, alertes ou opportunites, progression notable ;
4. `Leaderboards` et `Year in Review` peuvent apparaitre ici sous forme de cartes editoriales ou de modules temporaires ;
5. la home doit donner envie d'ouvrir une analyse plus profonde, pas essayer d'etre toute l'application a elle seule.

Direction visuelle : grande respiration, hero fort, cartes plus editoriales que tabulaires, forte sensation de cockpit ou de briefing.

#### Career

Role : page de reference du joueur, lecture stable de sa progression et de son niveau.

1. `Career` doit etre la destination la plus solide et la plus comprehensible du produit ;
2. les KPIs principaux doivent etre presents en haut, avec une hierarchie tres claire entre rang, volume de jeu, efficacite et dynamique ;
3. le CTA `Comparer` doit vivre ici de facon naturelle, dans l'en-tete ou a cote du bloc identitaire joueur ;
4. `Leaderboards` peut commencer ici comme une vue secondaire ou un bloc `Position dans ta cohorte` ;
5. la page doit eviter l'effet sapin de Noel : mieux vaut peu de sections mais tres lisibles que dix cartes egales visuellement.

Direction visuelle : plus institutionnelle que Home, plus stable, avec une composition en strates nettes et des sections plus analytiques.

#### Match History

Role : page d'exploration chronologique et d'acces au detail.

1. l'historique doit etre percu comme un flux maitrisable, pas comme une grille illisible ;
2. les filtres doivent etre tres visibles mais visuellement plus calmes que le contenu ;
3. la privacy doit apparaitre ici sous forme de warning elegant, explicite et non alarmiste ;
4. les exclusions manuelles doivent etre claires, irreversibles en apparence courte, mais jamais envahissantes ;
5. les lignes ou cartes de match doivent mettre en avant le signal utile avant la masse de statistiques.

Direction visuelle : flux, cadence, lisibilite, forte qualite de tableau ou de liste dense, avec excellents etats vides et etats limites.

#### Match View

Role : theatre d'operations du match individuel.

1. la page doit assumer un niveau de detail superieur, mais avec une entree narrative immediate : contexte, issue, moment cle, lecture rapide ;
2. les warnings de privacy ou de donnees manquantes doivent etre integres dans la lecture, pas jetes comme erreurs systeme ;
3. les statistiques detaillees doivent venir apres une couche de synthese ;
4. si des assets plus riches arrivent via O8, ils doivent augmenter la comprehension du match, pas seulement decorer.

Direction visuelle : immersive, plus dramatique, plus cinematographique que les autres pages, mais toujours precise et tres propre.

#### Squad

Role : page relationnelle, sociale au sens analytique, pas au sens reseau social.

1. la page doit montrer avec qui le joueur performe, perd, progresse ou se stabilise ;
2. `Compare` peut naturellement partir d'ici, en comparant le joueur avec un coequipier recurrent ;
3. si un jour O11 revient, c'est ici qu'il devra etre rattache, jamais comme produit parallele autonome ;
4. la page doit privilegier les cohortes, binomes, habitudes, complementarites et tensions de jeu.

Direction visuelle : plus relationnelle, plus matricielle, mais en gardant un fort niveau de clarte et de tri visuel.

#### Compare

Role : mode d'analyse laterale, focalise sur la lecture d'ecarts et de profils.

1. `Compare` ne doit pas commencer comme une page de menu ;
2. le meilleur format initial est un drawer large, un sheet ou une split view qui garde le contexte du joueur de depart ;
3. la lecture doit reposer sur des duels clairs : domination, proximite, complementarite, asymetrie de volume ;
4. les KPI doivent etre peu nombreux et impeccablement racontes ;
5. si la feature prouve une forte recurrence, elle peut ensuite gagner une route dediee, mais seulement apres validation d'usage.

Direction visuelle : structure comparative tres lisible, dualite gauche/droite claire, emphasis forte sur les deltas et les points de bascule.

#### Leaderboards

Role : preuve sociale et positionnement relatif.

1. au debut, ce n'est pas une destination primaire ;
2. la premiere incarnation doit etre un module compact dans `Home` ou `Career` ;
3. la cle UX n'est pas d'afficher beaucoup de noms, mais de montrer ou se situe le joueur par rapport a sa cohorte ;
4. un lien `Voir plus` peut ouvrir une route secondaire si le module est frequemment utilise ;
5. ce parcours doit rester utile meme avec un perimetre LevelUp limite, sans singer un leaderboard mondial impossible a tenir.

Direction visuelle : plus editorial que brut, avec mise en scene des positions, deltas et seuils plutot qu'une simple table generique.

#### Year in Review

Role : grande page narrative, partageable, emotionnelle et retrospective.

1. cette surface doit etre pensee comme une experience a part ;
2. elle ne doit pas polluer la navigation courante toute l'annee ;
3. sa decouverte peut venir d'une carte home, d'un bandeau de saison ou d'un lien partage ;
4. la hierarchie doit alterner moments forts, chiffres clefs, progression et signatures de jeu ;
5. la page doit assumer un rythme plus editorial et moins dashboard.

Direction visuelle : forte ambition visuelle, sections pleine largeur, respirations larges, transitions marquees, storytelling premium.

### Regles de qualite pour le frontend

1. une action importante doit toujours etre visible dans la zone haute de la page ;
2. une page dense doit toujours proposer une lecture rapide avant la lecture experte ;
3. un etat vide ou partiel doit rester beau, clair et volontaire ;
4. les warning states doivent etre traites comme une partie du produit, pas comme des erreurs de dev ;
5. toute nouvelle composante doit justifier son existence par sa valeur de lecture, pas uniquement par la disponibilite d'un endpoint.

### Consequence sur les opportunites ouvertes

1. O1 doit prioriser la qualite de mise en scene des limites de donnees ;
2. O2 et O3 doivent rester largement invisibles mais hausser la qualite percue des ecrans existants ;
3. O5 doit viser une experience comparative premium avant d'aspirer a une page autonome ;
4. O7 et O9 ne doivent etre ouverts qu'avec une intention editoriale forte, pas comme des annexes de menu.

## Schemas d'atterrissage UI

Cette section illustre schematiquement comment l'utilisateur decouvre chaque opportunite visible dans l'UI.

Hypothese cible : ces schemas supposent un shell sans sidebar obligatoire. Les points d'entree se font donc via `Home`, `Career`, `Squad`, des cartes editoriales, des CTA d'en-tete, des tabs contextuels ou une navigation principale compacte.

Ne sont pas inclus ici :

1. les opportunites metadata et socle invisibles cote utilisateur : O2, O3, O8 ;
2. les opportunites admin / support / outillage : O12, O13, O14.

Convention de lecture :

1. `Shell` = point d'entree principal dans l'application, sans presumer une barre laterale ;
2. `Module` = bloc dans une page existante ;
3. `CTA` = point d'entree actionnable ;
4. `Route secondaire` = destination possible, mais pas menu de premier niveau.

### O1 - Match privacy

```text
Shell
	-> Match History
		-> Bandeau "Historique limite / prive / partiel"
			-> Lignes de match avec etats de donnees clairs
				-> Match View
					-> Warning integre + sections degradees elegantement

Et en parallele :
Bootstrap / Home
	-> signal discret "certaines donnees sont limitees"
		-> renvoi vers Match History
```

But UX : expliquer la limite de donnees sans transformer le produit en ecran d'erreur.

### O4 - Match count endpoint

```text
Pas d'atterrissage UI prevu
	-> opportunite ecartee
	-> aucun module dedie
	-> aucun point d'entree recommande
```

But UX : eviter d'introduire un compteur seduisant mais peu fiable dans le scope reel LevelUp.

### O5 - Compare joueur vs joueur

```text
Career
	-> CTA "Comparer"
		-> Drawer / Sheet "Choisir un joueur"
			-> Vue comparee laterale
				-> option future : "Ouvrir en grand"
					-> Route secondaire si l'usage le justifie

ou

Squad
	-> Ligne coequipier recurrent
		-> CTA "Comparer"
			-> meme drawer / meme vue comparee
```

But UX : garder le contexte du joueur de depart et eviter une destination primaire vide de type `Compare`.

### O6 - Export service record / profil

```text
Pas d'atterrissage UI prevu
	-> opportunite ecartee
	-> pas de bouton dedie recommande a ce stade
```

But UX : ne pas ajouter une action secondaire faible dans des en-tetes deja charges.

### O7 - CSR leaderboards

```text
Home
	-> Module "Top CSR de ta cohorte"
		-> lecture rapide de la position du joueur
			-> CTA "Voir plus"
				-> Route secondaire Leaderboards si le module est consulte

ou

Career
	-> Bloc "Position dans ta cohorte"
		-> meme CTA "Voir plus"
			-> meme route secondaire

Puis depuis Leaderboards :
	-> clic sur un joueur
		-> CTA "Comparer"
			-> ouverture de Compare
```

But UX : commencer comme preuve sociale compacte, pas comme destination primaire.

### O9 - Year in Review

```text
Home
	-> Carte saisonniere / annuelle
		-> CTA "Voir ton annee"
			-> Route dediee shareable Year in Review
				-> lecture narrative par chapitres
					-> retour vers Career / Home

ou

Career
	-> Carte editoriale ponctuelle "Ton recap annuel"
		-> meme route shareable
```

But UX : en faire une experience evenementielle et memorisable, pas un onglet permanent.

### O10 - Store / economy tracker

```text
Pas d'atterrissage UI maintenant
	-> opportunite en parking lointain

Si le sujet revient un jour :
Home
	-> Module optionnel "Rotation du moment"
		-> detail en overlay ou route secondaire
```

But UX : ne pas diluer LevelUp hors de son coeur analytics sans validation produit explicite.

### O11 - Spartan Company / social layer

```text
Pas d'atterrissage UI maintenant
	-> opportunite en parking lointain

Si le sujet revient un jour :
Squad
	-> Groupes / cohortes sauvegardees
		-> vue de groupe
			-> details de performance collective
```

But UX : si la dimension groupe revient, elle doit naitre dans `Squad`, pas ouvrir un sous-produit social parallele.

### Lecture d'ensemble

```text
Home
	-> Leaderboards (module)
	-> Year in Review (carte ponctuelle)
	-> signal privacy (si necessaire)

Career
	-> Compare (CTA principal)
	-> Leaderboards (bloc secondaire)
	-> Year in Review (carte ponctuelle)

Match History / Match View
	-> Privacy (warning integre)

Squad
	-> Compare (point d'entree relationnel)
	-> Social layer un jour, si retenu
```

Synthese :

1. O1 s'ancre dans `Match History` et `Match View` ;
2. O5 s'ancre dans `Career`, `Squad` et eventuellement `Profile/Citations` ;
3. O7 commence dans `Home` ou `Career`, puis peut gagner une route secondaire ;
4. O9 commence dans `Home` ou `Career`, puis vit comme experience shareable ;
5. O4 et O6 n'ont pas d'atterrissage UI ;
6. O10 et O11 restent sans landing tant qu'ils sont en parking.
