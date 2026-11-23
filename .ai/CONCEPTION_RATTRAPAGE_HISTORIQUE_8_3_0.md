# Conception : refonte du rattrapage de l'historique pour la 8.3.0

Date : 2026-11-22. Statut : proposition, à faire trancher (section 5) puis à transformer en plan.
Auteur : la revue adversariale de la 8.2.2 (une ronde, lecture seule).

Ce document se lit sans la conversation qui l'a produit. Il décrit l'état livré par la 8.2.2, ses
limites chiffrées, la conception cible, ce qu'elle supprime, les décisions à prendre, les risques et un
découpage en étapes.

**Base de lecture.** Le code a été lu sur `release/v8.2.2` au commit `de25cc2063`, donc AVANT les
correctifs de revue de la 8.2.2 (voir 1.4). Toute liste de fichiers ou de lignes de ce document est à
revérifier sur la branche au moment d'ouvrir le chantier. Les chemins Go sont relatifs à `apps/go-api/`.

**Vocabulaire, un mot par chose.**

| Mot | Sens |
|---|---|
| liste de l'API | L'historique des matchs d'un joueur tel que l'API le rend, du plus récent au plus ancien, par pages de 25, adressées par décalage. |
| ensemble connu | Les matchs déjà rangés pour un joueur (règle unique du paquet `internal/sync/knownset`). |
| cycle | Le cycle de synchronisation de l'ordonnanceur (`internal/scheduler`), toutes les 6 heures par défaut. |
| delta | La partie du cycle qui prend les nouveaux matchs du haut de la liste. |
| rattrapage | La récupération des matchs anciens jamais rangés. |
| relevé | NOUVEAU. La liste complète et ordonnée des identifiants de matchs d'un joueur, lue une fois. |
| lot | NOUVEAU. 25 matchs du relevé traités ensemble : récupérés, écrits, prouvés rangés. |
| session | NOUVEAU. Une suite de lots pour un joueur, bornée dans le temps. |
| boucle de rattrapage | NOUVEAU. Le travailleur de fond qui enchaîne les sessions, hors du cycle. |
| ouvrier de films | NOUVEAU. Le travailleur de fond qui résorbe l'arriéré de films, hors du cycle. |

---

## 1. Le problème et ce que la 8.2.2 a livré

### 1.1 Le problème d'origine

Issue GitHub #213 : un joueur sur l'application Windows 8.2.1, environ 7 000 matchs. La synchronisation
automatique ne prenait que les nouveaux matchs. L'historique ancien n'était jamais rattrapé.

Cause : le cycle automatique passe par le pipeline V2 (`internal/sync/v2`). Sa découverte s'arrêtait au
premier match connu. Un rattrapage existait depuis la 8.1.2, mais seulement dans l'ancien moteur V1
(`internal/sync/engine.go`), que l'ordonnanceur n'appelle pas. Les tests ne passaient que par V1.

### 1.2 Ce que fait la 8.2.2

- **Un seul moteur de rattrapage** : le cycle de l'ordonnanceur. Pipeline V2 pour Halo Infinite, runner
  `livesync` pour Halo 5. Le moteur V1 (suivi en direct, synchronisation manuelle d'un joueur,
  synchronisation initiale) est un delta pur : il ne lit ni n'écrit l'état de l'historique.
- **Passes bornées** : tant que la fin de l'historique d'un joueur n'a pas été vue, chaque cycle range
  au plus 200 matchs anciens pour lui (`historystate.PassBudget()`).
- **Cadence** : tant qu'un joueur rattrape, le cycle suivant vient après 10 minutes au lieu de
  l'intervalle réglé (`catchUpCycleInterval`, `internal/scheduler/auto_sync_loop.go`). Sur le bureau, la
  cadence de 10 minutes tient aussi tant qu'il reste des films à traiter.
- **Marque-page** : la passe range sa position dans la liste de l'API, pour ne pas relire la liste
  depuis le début. La position est recalée par une ancre (l'ancienne tête de liste) et contrôlée par un
  identifiant.
- **Delta** : il s'arrête après 25 matchs connus d'affilée (`domain.DeltaKnownRun`), plus au premier.
- **Écran** : bouton « Synchroniser maintenant » et état de l'historique par titre dans Réglages ›
  Données & Médias (`apps/web/src/features/settings/SyncNowCard.tsx`, `HistoryState.tsx`).
- **Films** : sur le bureau (capacité `FilmCatchUp` du profil de déploiement), les deux étapes film du
  post-sync ne sont bornées que par leur budget de 5 minutes. Sur le site hébergé : 8 films de frags et
  5 rejeux par cycle, comme avant.

### 1.3 État rangé par la 8.2.2

Tout l'état vit dans la table `sync_meta` de la base joueur, sans changement de schéma. Seul le paquet
`internal/sync/historystate` nomme ces clés (garde-rail
`internal/archlint/history_key_single_source_test.go`).

| Clé `sync_meta` | Contenu | Fichier |
|---|---|---|
| `history_complete_at` | Instant où la fin de l'historique a été vue. Absente : rattrapage à faire. | `historystate/historystate.go` |
| `history_catchup_cursor` | JSON : marque-page (`offset`, `last_match_id`, `head_match_id`) et matchs en échec avec leur nombre de passes (`failed`, 200 au plus). | `historystate/cursor.go`, `anchor.go` |
| `history_failure_streak` | JSON : échecs consécutifs d'une page (Halo 5 seulement). | `historystate/streak.go` |
| `history_skip_set` | JSON : matchs à ne plus demander, `not_found` (404/410) ou `abandoned`, 2 000 au plus. | `historystate/skipset.go` |

Lecture et écriture de `sync_meta` : `internal/sync/syncmeta` (SELECT puis UPDATE ou INSERT, jamais de
DELETE ni de `ON CONFLICT`). Un effacement range une valeur vide.

Fichiers principaux de la 8.2.2 (lignes au commit lu) :

| Rôle | Fichiers | Lignes |
|---|---|---|
| État | `internal/sync/historystate/{historystate,cursor,anchor,skipset,streak}.go` | 661 |
| Découverte V2 en rattrapage | `internal/sync/v2/fetchers_history.go`, `fetchers_history_anchor.go` | 378 |
| Règlement après persistance (V2) | `internal/sync/v2/history.go`, `history_settle.go`, `history_skip.go` | 553 |
| Preuve de rangement, file d'écriture | `internal/sync/v2/cycle_stored.go`, `persist_pending.go`, `persist_drain.go` | 307 |
| Rattrapage Halo 5 | `internal/games/halo_5/livesync/catchup.go`, `catchup_anchor.go`, `backfill_walk.go`, `history_store.go` | 733 |
| Cadence et bouton | `internal/scheduler/auto_sync_loop.go`, `auto_sync_manual.go`, `auto_sync_passes.go` | 593 |
| Arriéré de films | `internal/sync/filmbacklog/filmbacklog.go`, `killcollector/postsync_arriere.go`, `postsync_bornes.go`, `replayartifacts/bornes.go` | environ 220 |
| Lecture pour l'écran | `internal/platform/duckdb/historyrepo/repo.go`, `internal/service/history_sync_service.go`, `internal/port/history_state.go` | environ 330 |

### 1.4 Correctifs de revue annoncés pour la 8.2.2

Le pilote a annoncé ces correctifs avant la sortie de la 8.2.2. Ils n'étaient pas dans le code lu.

- Abandon d'un match : aucune tentative comptée quand l'échec est systémique ; nouvelle tentative après
  24 heures.
- Budget global par cycle, et plus seulement par joueur.
- Garde de progrès sur la cadence de 10 minutes.
- Branche « marque-page sans ancre » du pipeline Infinite supprimée.
- Fin d'historique confirmée par une page vide ; silence Discord pendant un rattrapage ; relecture de
  l'état du cycle à l'écran ; textes de l'écran ; fichiers repassés sous 500 lignes.

Ces correctifs traitent les symptômes. La refonte retire leur cause (section 2).

### 1.5 Mesures réelles de la 8.2.2

Mesures sur une copie des données du développeur, profil `dev`, avant les correctifs de revue.

| Mesure | Valeur |
|---|---|
| Parcours complet d'une liste de 1 250 matchs | 50 pages, 12 s |
| Récupération de 200 matchs (parallélisme 8) | environ 235 s |
| Par match, en parallèle | stats 3,4 s, skill 1,6 s, extrait de film 4,2 s |
| Persistance | environ 1 s par match, en série |
| Films, bureau | 10,5 s par film de frags, environ 9 s par rejeu |
| Pic mémoire du processus pendant une passe de 200 | 937 Mo |
| Réponses 429 | aucune, avec un seul joueur |

---

## 2. Les limites de la 8.2.2 que la refonte lève

Chaque limite vient du même choix : le rattrapage vit DANS le cycle de synchronisation de tous les
joueurs, et il avance par position dans une liste qui bouge.

### 2.1 Mémoire et durée proportionnelles au nombre de joueurs

- Fait : le cycle récupère tous les matchs listés avant d'écrire le premier
  (`internal/sync/v2/fetch_shared.go`, `res.Matches`). Une passe de 200 matchs pour UN joueur donne un
  pic de 937 Mo.
- Fait : au commit lu, le budget est par joueur. Aucun test n'a deux joueurs en rattrapage en même
  temps.
- Hypothèse : avec deux ou trois joueurs en rattrapage, le site hébergé (2 Go) est dépassé.
- Le correctif 8.2.2 (budget global) borne la mémoire, mais divise le débit par le nombre de joueurs.
- Fait : pendant une passe, le verrou « un cycle tourne » est tenu 15 à 25 minutes. Le bouton
  « Synchroniser maintenant » est refusé pendant ce temps, et les nouveaux matchs des autres joueurs
  attendent.

### 2.2 Débit bas, à cause de l'attente

- Fait : une passe coûte environ 12 s de liste, 235 s de récupération, 220 s d'écriture, puis jusqu'à
  10 minutes de films sur le bureau, puis 10 minutes d'attente.
- Calcul : 200 matchs toutes les 18 à 28 minutes, soit 430 à 670 matchs par heure.
- Calcul : 7 000 matchs demandent 35 passes, 12 à 16 heures d'application ouverte. 20 000 matchs
  demandent 100 passes, 35 à 45 heures.

### 2.3 Un état par position, fragile et coûteux à maintenir

- Fait : trois façons de situer le marque-page dans Infinite (ancre, sans ancre, sans marque-page) et
  une quatrième, différente, dans Halo 5 (page de recouvrement, compte d'échecs par page).
- Fait : quatre clés d'état, environ 2 300 lignes hors tests pour l'état, la découverte, le règlement
  et Halo 5.
- Fait : toute incohérence de position (match arrivé pendant la lecture, ancre perdue au-delà de 200
  arrivées) fait repartir la lecture du début. À 20 000 matchs, une reprise depuis le début coûte
  800 pages, environ 3 minutes.
- Fait : un joueur déjà complet mais sans clé relit toute sa liste sans rien noter. Une seule page en
  erreur fait tout recommencer.

### 2.4 Un abandon qui rend définitif un verdict provisoire

- Fait : au commit lu, un match en échec à 3 passes est abandonné, puis rangé dans l'ensemble des
  matchs à ne plus demander. Seul le 404/410 est distingué.
- Fait : une panne de l'API de 30 minutes (liste lisible, stats en erreur) fait abandonner 200 matchs
  sains. Pour Halo 5, une page de 25 matchs par tranche de trois passes.
- Le correctif 8.2.2 atténue (échec systémique ignoré, nouvelle tentative après 24 heures). Le
  mécanisme reste : un marque-page qui avance exige de décider qu'un match « ne bloque plus ».

### 2.5 Des films qui n'avancent que dans un cycle de synchronisation

- Fait : les deux étapes film tournent dans le post-sync du cycle. Pour décoder des films, le bureau
  rejoue toutes les 10 minutes la découverte et le post-sync de tous les joueurs.
- Fait : sur le site hébergé, la borne de 8 films de frags vaut « par cycle ». Pendant un rattrapage, le
  cycle revient toutes les 10 minutes au lieu de 6 heures : l'étape tourne jusqu'à 36 fois plus
  souvent.
- Calcul : 5 minutes de budget pour 10,5 s par film donnent environ 28 films de frags par passe. Avec
  un décodeur deux fois plus rapide, 57. Le frein devient l'attente de 10 minutes, pas le décodeur.

### 2.6 Aucun dénominateur à l'écran

- Fait : la 8.2.2 ne connaît le nombre total de matchs qu'à la fin du rattrapage. L'écran dit
  « Rattrapage en cours » et « Matchs enregistrés : N », jamais « N sur M ».

### 2.7 Aucun moyen de revérifier un historique dit complet

- Fait : les joueurs marqués complets par l'ancien moteur V1 le restent. V1 rangeait la fin même avec
  des matchs en échec.
- Fait : après la fin, seul le delta regarde la liste, sur ses premières pages. Un trou plus profond
  n'est jamais revu.

---

## 3. La conception cible

### 3.1 Principe

Deux idées remplacent le marque-page.

1. **Le relevé.** On lit une fois toute la liste de l'API et on range les identifiants. Le passé d'une
   liste ne change pas : le relevé reste vrai.
2. **Le reste à faire se calcule.** À tout moment : reste = relevé, moins l'ensemble connu, moins les
   matchs introuvables. Il n'y a plus de position à ranger, à recaler ni à contrôler.

Trois conséquences.

- Un match en échec ne bloque plus rien. Il reste dans le reste à faire, avec une date de prochaine
  tentative.
- Le rattrapage sort du cycle. Une boucle à part traite un joueur à la fois, par lots de 25.
- Le total est connu dès le relevé : l'écran affiche « X sur Y ».

### 3.2 Le relevé

**Où.** Dans `sync_meta` de la base joueur, sous une nouvelle clé `history_id_list`, lue et écrite par
`internal/sync/historystate` (recommandation, décision D-1). Aucun changement de schéma. Le relevé suit
la base joueur dans les sauvegardes et dans la purge d'un joueur.

**Format.** Un document JSON, écrit une fois (décision D-2) :

```json
{
  "v": 1,
  "taken_at": "2026-12-01T10:00:00Z",
  "ids": ["<id le plus récent>", "...", "<id le plus ancien>"],
  "excluded": [12, 13, 40]
}
```

- `ids` : tous les identifiants de la liste, dans l'ordre de l'API, sans doublon.
- `excluded` : positions des entrées que le titre ne récupère jamais (Halo 5 : Warzone et Campagne).
  Elles restent dans `ids` parce que Halo 5 a besoin de la position d'une entrée dans la liste (3.4).
  Vide pour Halo Infinite.
- `v` : version du format. Une version inconnue vaut « pas de relevé ».

**Taille.** Un identifiant est un UUID de 36 caractères, soit 39 octets en JSON.

| Matchs | Taille du relevé | Durée de lecture de la liste (12 s pour 50 pages) |
|---|---|---|
| 7 000 | 273 Ko | environ 67 s |
| 20 000 | 780 Ko | environ 3 min 12 |
| 50 000 (borne `historystate.MaxDepth`) | 1,95 Mo | environ 8 min |

**Prise du relevé.**

1. La boucle lit la liste page par page, du décalage 0 jusqu'à une page VIDE (une page partielle ne
   suffit pas).
2. Les identifiants sont gardés en mémoire, dédoublonnés.
3. Le relevé est écrit en une fois, sous le bail d'écriture du joueur, seulement quand la page vide
   est atteinte.
4. Une erreur de page arrête la lecture. Rien n'est écrit. La boucle réessaie avec un délai qui
   double (5 minutes, puis 10, jusqu'à 1 heure).

Un match qui arrive en haut de liste PENDANT la lecture décale les pages d'un cran vers le bas. La page
suivante répète alors la dernière entrée de la page précédente. L'effet est un doublon, jamais un
oubli : le dédoublonnage suffit.

**Quand un relevé est pris.**

- Le joueur n'a ni clé de fin (`history_complete_at`) ni relevé valide : premier rattrapage, ou
  rattrapage commencé sous la 8.2.2 (4.3).
- Un ré-audit est demandé (3.7).

**Renouvellement.** Un relevé n'est jamais mis à jour. Les matchs arrivés après lui sont au-dessus de
sa tête : c'est le travail du delta. À la fin du rattrapage, la clé du relevé reçoit une valeur vide.

**Matchs qui disparaissent de la liste.** Halo Infinite récupère un match par son identifiant, pas par
la liste. Un match du relevé que l'API ne sert plus répond 404 ou 410 : il est rangé comme introuvable
(`history_skip_set`, raison `not_found`), comme aujourd'hui. Pour Halo 5, voir 3.4.

**Matchs qui apparaissent tard dans le passé de la liste.** Hypothèse : le cas est rare. Le relevé ne
les voit pas. Le ré-audit (3.7) les retrouve.

### 3.3 Les échecs, sans abandon définitif

Une nouvelle clé `history_failures` remplace la liste `failed` du marque-page :

```json
{"failed": [{"id": "...", "attempts": 2, "first_at": "...", "last_at": "...", "next_at": "..."}]}
```

Règles (décision D-4) :

- Un 404 ou 410 n'est jamais un échec : le match est introuvable, rangé une fois pour toutes.
- Un échec systémique ne compte pas : erreur 429, 5xx ou réseau, ou plus de 10 % d'un lot en échec. La
  session s'arrête et la boucle attend (3.5).
- Un échec propre au match compte une tentative et fixe `next_at` : 1 heure, puis 24 heures, puis
  7 jours.
- Un match à 3 échecs étalés sur au moins 24 heures est « en échec durable ». Il n'empêche plus de
  ranger la fin. Il reste dans la liste, retenté tous les 7 jours, et compté à l'écran.
- La liste est bornée à 2 000 entrées. Au-delà, la boucle journalise et n'ajoute plus : le reste à
  faire se recalcule de toute façon à chaque session.

**Fin de l'historique.** `history_complete_at` est rangée quand chaque identifiant du relevé est soit
dans l'ensemble connu, soit introuvable, soit en échec durable. Un changement nécessaire dans
`historystate.Complete` : une clé présente mais VIDE vaut « incomplet » (aujourd'hui, seule la présence
compte). Sans ce changement, un ré-audit ne peut pas rouvrir un historique, puisque `sync_meta` ne
supporte pas de DELETE.

### 3.4 La boucle de rattrapage

Un nouveau paquet `internal/sync/historycatchup` porte la boucle. Il ne connaît aucun titre.

**Une session, pour un joueur.**

1. Prendre le verrou croisé du joueur (`sync.SyncGate.TryClaimT`), déjà partagé par le cycle, le suivi
   en direct et la synchronisation manuelle. Occupé : passer au joueur suivant.
2. Lire l'état : clé de fin, relevé, introuvables, échecs. Sans relevé : le prendre (3.2).
3. Lire l'ensemble connu une fois (`knownset.Load`).
4. Calculer le reste à faire, dans l'ordre du relevé : ni connu, ni introuvable, ni exclu, ni en
   attente de sa prochaine tentative.
5. Tant qu'il reste du travail et que la session peut continuer :
   1. prendre les 25 premiers : un lot ;
   2. demander à la source du titre de les récupérer et de les écrire ;
   3. prouver le rangement : relire, pour ces 25 identifiants seulement, ce que la base partagée a
      rangé ;
   4. mettre à jour introuvables et échecs sous le bail d'écriture du joueur ;
   5. publier la progression (3.8) ;
   6. attendre la pause entre lots.
6. Reste à faire vide (ou seulement des échecs durables) : ranger la fin, vider le relevé.
7. Lancer le post-sync du joueur pour les matchs rangés (grain : décision D-10).

**Quand une session s'arrête.**

- Durée maximale atteinte (décision D-7).
- Un cycle de synchronisation démarre : la session finit son lot, puis cède.
- Arrêt du serveur : le contexte est annulé. Le lot en cours n'est pas compté comme un échec.
- Échec systémique : la boucle attend, avec un délai qui double (5 minutes à 1 heure).
- Bureau, une partie tourne : pause (décision D-6).
- Synchronisation automatique désactivée : pas de session (décision D-5).

**Un joueur à la fois.** La boucle prend les joueurs à historique incomplet à tour de rôle, une session
chacun. Jamais deux sessions en même temps dans un processus. La mémoire est celle d'un lot, quel que
soit le nombre de joueurs.

**La source du titre.** Un port, câblé par `cmd/server` (aucun import d'un paquet de titre dans le
moteur commun) :

```go
// Dans internal/port (noms indicatifs).
type HistoryCatchUpSource interface {
    // ListPage rend une page de la liste de l'API : identifiants, et lesquels sont exclus.
    ListPage(ctx context.Context, p Player, offset, count int) (HistoryListPage, error)
    // FetchAndStore récupère et écrit un lot. Index est la position du premier identifiant
    // dans le relevé. Rend, par identifiant, l'issue : rangé, introuvable, échec propre,
    // échec systémique.
    FetchAndStore(ctx context.Context, p Player, lot HistoryLot) (map[string]LotOutcome, error)
}
```

- **Halo Infinite** (adaptateur dans `internal/sync/v2`). `FetchAndStore` réutilise les briques du
  cycle : `SharedMatchFetcher` pour la récupération, `CycleBatchPersister` pour l'écriture, donc
  `persist.BatchQueue` puis les `Persister`. Aucun nouveau chemin d'écriture. La position du lot n'est
  pas utilisée.
- **Halo 5** (adaptateur dans `internal/games/halo_5/livesync`). Fait vérifié : la capture Halo 5
  construit un match à partir de son entrée de liste (`halo_5/capture.go`, `capturePage`). Le mode de
  jeu de l'entrée décide de l'adresse du rapport. Un identifiant seul ne suffit pas. `FetchAndStore`
  relit donc la page de liste où le lot doit se trouver : position du lot dans le relevé, plus le
  nombre de matchs arrivés depuis le relevé (la position actuelle de la tête du relevé). Il ne traite
  que les entrées retrouvées PAR IDENTIFIANT. Une entrée non retrouvée dans la page ni dans ses deux
  voisines est un échec systémique. Trois échecs systémiques de cette nature font reprendre le relevé.
  La position n'est plus un état : c'est un indice, vérifié entrée par entrée.

**Preuve de rangement.** La règle « connu » a une source unique (`internal/sync/knownset`, garde-rail
`archlint/known_set_single_source_test.go`). La lecture ciblée d'un lot s'ajoute DANS ce paquet (par
exemple `knownset.Contains`), pas dans un nouveau dépôt de lecture.

**Débit attendu.** Hypothèse, à mesurer (étape de mesure, section 7) : un lot de 25 coûte environ 30 s
de récupération et 25 à 28 s d'écriture, soit une minute. Sans pause : environ 1 500 matchs par heure,
contre 430 à 670 en 8.2.2. 7 000 matchs en 5 heures environ, 20 000 en 13 heures environ.

**Ce que le cycle garde.** Le cycle redevient un delta pour tous les joueurs, à l'intervalle réglé. Il
ne rattrape plus, ne lit plus l'état de l'historique, ne change plus de cadence.

### 3.5 Site hébergé, plusieurs joueurs, 429

- Un joueur à la fois, un lot de 25 à la fois : la mémoire ne dépend plus du nombre de joueurs.
- Une pause entre lots, réglée par une capacité du profil de déploiement (ADR 0044), jamais par le
  profil lui-même. Valeurs fixées après la mesure (décision D-7).
- La prise des relevés est sérialisée comme le reste : un seul joueur lit sa liste à la fois. La 8.2.2
  lance une lecture par joueur, toutes en parallèle.
- Un 429 arrête la session, sans tentative comptée. La boucle attend, délai doublé de 5 minutes à
  1 heure. Un compteur et une ligne de journal par arrêt.
- Le rattrapage n'émet aucune notification de « nouveaux matchs » (décision D-11).

### 3.6 L'ouvrier de films

Un travailleur de fond par processus, hors du cycle.

- Il appelle les deux étapes film existantes par leurs points d'entrée actuels : la source des frags
  (`killcollector.RunPostSync`) puis les rejeux et leurs dérivés (`replayartifacts.Run`). Budgets de
  temps, verrou de décodage, report quand une partie tourne, marqueurs terminaux : inchangés.
- Il se réveille quand un cycle ou un lot de rattrapage a rangé des matchs, au démarrage, puis de
  lui-même tant qu'il reste de l'arriéré.
- **Garde de progrès intégrée** : une passe qui n'écrit rien double le délai avant la suivante
  (5 minutes à 6 heures). Une passe qui écrit le remet au minimum.
- Le post-sync du cycle n'appelle plus les deux étapes film. Il réveille l'ouvrier, en lui passant les
  matchs du cycle : ils restent traités en premier.
- **Site hébergé** : le rythme devient explicite, par heure et non par cycle (décision D-8). Les
  rejeux y restent mis en file pour l'ouvrier de rejeux existant (`replaybuild`), sans changement.
- Le paquet `internal/sync/filmbacklog` reste la publication de l'arriéré. Son lecteur est l'écran.
  L'ordonnanceur ne le lit plus.

Effet d'un décodeur deux fois plus rapide : deux fois plus de films par passe, et plus aucune attente
de 10 minutes entre deux passes.

### 3.7 Le ré-audit

Reprendre un relevé pour un joueur dit complet et le comparer à l'ensemble connu (décision D-3).

- Reste à faire vide : rien ne change.
- Reste à faire non vide : la clé de fin reçoit une valeur vide, le relevé est rangé, la boucle
  rattrape.
- Coût : une lecture de liste, 67 s pour 7 000 matchs.
- Déclencheurs proposés : une commande `levelup history audit <joueur>`, une action dans
  l'administration, et une fois automatiquement pour les joueurs dont la clé de fin est antérieure à la
  8.2.2 (rangée par l'ancien moteur V1).

### 3.8 La progression à l'écran

La boucle publie, par titre et par joueur, après chaque lot : total du relevé hors exclus, rangés,
introuvables, en échec. Publication en mémoire, comme `filmbacklog`, lue par un port.

- `GET /players/{player_slug}/sync/history` sert quatre champs de plus par titre. Chaque champ a son
  lecteur : l'écran (règle 18).
- Écran : « 3 400 sur 7 012 », et, s'il y en a, « 12 introuvables » et « 2 en échec ».
- Après la fin : « Complet », avec le nombre de matchs enregistrés, comme aujourd'hui.
- Avant le premier lot d'un processus neuf : l'état sans les nombres. Aucun compteur n'est rangé en
  base : pas de valeur périmée.
- Textes dans chaque langue de `config/locales.toml` (dictionnaire `syncNowText.ts`, catalogue
  `internal/i18n/messages/sync.toml`).

### 3.9 Le delta par ancre : non retenu

Idée : ranger la tête de liste à chaque cycle, et prendre au cycle suivant tout ce qui est au-dessus.
Le delta devient exact et lit une page de moins quand il y a des nouveaux matchs.

Recommandation : ne pas le faire en 8.3.0 (décision D-9). Raisons :

- La règle des 25 connus d'affilée resterait comme repli (ancre absente ou perdue) et pour le moteur
  V1, qui n'a pas d'état. Le code augmenterait au lieu de diminuer.
- Le trou que la règle des 25 laisse demande plus de 25 matchs connus au-dessus d'un inconnu. Il faut
  pour cela une synchronisation arrêtée pendant qu'un coéquipier suivi en direct range les matchs
  communs. Le ré-audit retrouve ce cas.
- Le delta est le chemin de tous les joueurs à chaque cycle. Le toucher sans mesure d'un défaut réel
  est un risque sans gain démontré.

---

## 4. Ce qui est supprimé de la 8.2.2

Règle 7 du projet : ce qui est débranché est supprimé, avec ses tests. Liste établie au commit lu, à
revérifier.

### 4.1 Code supprimé

| Fichier | Sort |
|---|---|
| `internal/sync/historystate/cursor.go`, `anchor.go`, `streak.go` et leurs tests | Supprimés. |
| `internal/sync/historystate/historystate.go` | Gardé. `PassBudget` supprimé. `Complete` : valeur vide = incomplet. `MarkComplete` vide le relevé au lieu du marque-page. |
| `internal/sync/historystate/skipset.go` | Gardé pour `not_found`. La raison `abandoned` disparaît. |
| `internal/sync/v2/fetchers_history.go`, `fetchers_history_anchor.go` et leurs tests | Supprimés. `ListUnknownMatches` redevient un delta seul ; `Listing.EndSeen` et `Listing.Proposed` disparaissent. |
| `internal/sync/v2/history.go`, `history_settle.go` | Supprimés. `HistoryState`, `PlayerHistory`, `historyTracker`, `WithHistoryWriter`, `settleHistory`, `trackFailures` disparaissent. |
| `internal/sync/v2/history_skip.go` | Réduit : le delta range encore ses introuvables. |
| `internal/sync/v2/known_loader.go` | `readHistoryState` réduit à la lecture des introuvables. |
| `internal/sync/v2/types.go` | Champ `PlayerOutcome.HistoryPending` supprimé. |
| `internal/sync/v2/cycle_stored.go`, `persist_pending.go`, `persist_drain.go` | Gardés : le compte « inséré » sur preuve et l'attente de la file servent au delta. |
| `internal/games/halo_5/livesync/catchup.go`, `catchup_anchor.go` et leurs tests | Supprimés. `Runner.RunDelta` redevient un delta seul. |
| `internal/games/halo_5/livesync/backfill_walk.go`, `backfill.go` | `BackfillResume`, le budget et l'abandon de page supprimés. La commande `h5-backfill` garde son parcours. |
| `internal/games/halo_5/livesync/history_store.go` | Supprimé : l'état passe par le moteur commun. |
| `internal/games/halo_5/capture.go` | `CaptureStats.TopSpan`, `TopIDs`, `CaptureOptions.TopLimit` supprimés. `ListMatchIDs` gardé (prise du relevé). |
| `internal/domain/sync.go` | Champ `SyncResult.HistoryPending` supprimé. `DeltaKnownRun` gardé. |
| `internal/scheduler/auto_sync_loop.go` | `catchUpCycleInterval`, `cycleDebt`, `historyReader`, `filmBacklog`, `pendingHistoryOf`, `pendingHistoryAfterV2`, `filmBacklogOf`, garde de progrès de la 8.2.2 : supprimés. La boucle revient à un intervalle unique. |
| `internal/scheduler/auto_sync_manual.go`, `auto_sync_passes.go` | Gardés (bouton, dernière passe). |
| `internal/port/history_state.go` | `HistoryCompletionReader` supprimé si plus aucun appelant. `SyncActivity.FilmsPending` déplacé vers le port de l'ouvrier de films. |
| `internal/platform/duckdb/historyrepo/repo.go` | Gardé pour l'écran. |
| Budget global par cycle (correctif 8.2.2) | Supprimé : le cycle ne rattrape plus. |
| Tests de bout en bout `cmd/server/sync_v2_convergence_*`, `*difficile*`, `*interruption*` | Gardés comme témoins : mêmes scénarios, joués par la boucle (section 6). |

Ordre de grandeur : environ 1 900 lignes de production supprimées, pour un moteur commun estimé à
500 à 700 lignes et deux adaptateurs de 100 à 150 lignes. Estimation, pas une mesure.

### 4.2 Clés `sync_meta`

| Clé | Sort |
|---|---|
| `history_complete_at` | Gardée, même sens. Une valeur vide vaut désormais « incomplet ». |
| `history_skip_set` | Gardée, pour les introuvables. |
| `history_id_list` | Nouvelle : le relevé. |
| `history_failures` | Nouvelle : les échecs par identifiant. |
| `history_catchup_cursor` | Plus lue. Vidée une fois par la migration. |
| `history_failure_streak` | Plus lue. Vidée une fois par la migration. |

Le garde-rail `archlint/history_key_single_source_test.go` suit : deux littéraux retirés, deux ajoutés.
Il interdit aussi les deux anciens littéraux partout, paquet `historystate` compris, une fois la
migration retirée (4.3).

### 4.3 Migration des états existants

Elle se fait à la première session de la boucle pour un joueur, sous le bail d'écriture du joueur.
Aucune migration de schéma.

| État trouvé | Action |
|---|---|
| Clé de fin présente et non vide | Rien. Le joueur est complet. Le marque-page est déjà vide. |
| Clé de fin absente, marque-page présent (rattrapage 8.2.2 en cours) | Prendre un relevé. Le reste à faire se calcule depuis l'ensemble connu : aucune information du marque-page n'est nécessaire. Vider `history_catchup_cursor`. |
| Liste `failed` du marque-page | Non reprise. Ces matchs sont dans le reste à faire et seront retentés. Coût : une tentative de plus par match. |
| `history_skip_set`, raison `not_found` | Gardée telle quelle. |
| `history_skip_set`, raison `abandoned` | Retirée de l'ensemble et versée dans `history_failures` avec 3 tentatives et une prochaine tentative immédiate. Le match est donc redemandé une fois, puis suit la règle 3.3. |
| `history_failure_streak` (Halo 5) | Vidée. Les pages abandonnées reviennent dans le reste à faire, puisqu'elles ne sont pas dans l'ensemble connu. |

Le code de migration porte, en commentaire, sa date de retrait et son critère mesurable (modèle :
`scheduler/auto_sync_convergence.go`). Critère proposé : un compteur « états 8.2.2 migrés » à zéro à
chaque démarrage pendant une version.

---

## 5. Décisions à faire trancher par l'utilisateur

| N° | Décision | Recommandation |
|---|---|---|
| D-1 | Où ranger le relevé : clé `sync_meta` de la base joueur, ou fichier voisin dans le dossier du joueur. | Clé `sync_meta`. Aucun nouveau type de fichier, suit la sauvegarde et la purge, même mécanique que les clés existantes. Le fichier n'apporte qu'une écriture sans bail. |
| D-2 | Format du relevé : JSON lisible (780 Ko pour 20 000 matchs) ou binaire compact (320 Ko). | JSON lisible. La taille est négligeable devant la base, et le relevé se lit avec `duckdb` pour un diagnostic. |
| D-3 | Ré-audit : (a) commande et action d'administration ; (b) une fois automatiquement pour les joueurs marqués complets avant la 8.2.2. | (a) et (b). Le (b) répare les trous laissés par l'ancien moteur, pour une lecture de liste par joueur, sérialisée. |
| D-4 | Matchs en échec : plus d'abandon définitif. « Complet » malgré des échecs durables, affichés et retentés tous les 7 jours. | Oui. Seuils proposés : 3 échecs étalés sur 24 heures ; délais 1 heure, 24 heures, 7 jours. |
| D-5 | Synchronisation automatique désactivée : la boucle tourne-t-elle ? | Non. Elle suit le réglage. Le bouton « Synchroniser maintenant » lance alors une session bornée, et l'écran le dit. |
| D-6 | Bureau : rattrapage pendant qu'une partie de Halo tourne. | Pause, comme les films. Le réseau et le disque restent au jeu. |
| D-7 | Rythme de la boucle : durée d'une session, pause entre lots, par profil. | Fixer après la mesure. Point de départ : bureau, session de 15 minutes, pas de pause ; site hébergé, session de 10 minutes, 60 s entre lots. |
| D-8 | Films sur le site hébergé : rythme par heure au lieu de « par cycle ». | Oui. Même quantité qu'avant la 8.2.2 : 8 films de frags et 5 rejeux par intervalle de synchronisation réglé, quel que soit le rattrapage. |
| D-9 | Delta par ancre à la place de la règle des 25 connus d'affilée. | Non en 8.3.0 (raisons en 3.9). À revoir si le ré-audit trouve des trous. |
| D-10 | Post-sync pendant un rattrapage : à chaque lot, tous les 200 matchs, ou en fin de session. | Tous les 200 matchs et en fin de session, à confirmer par la mesure du coût du recalcul des notes. |
| D-11 | Matchs rattrapés : aucune notification de nouveaux matchs (Discord, sociales). Records et défis : calculés, sans notification. | Oui. Un match de 2022 n'est pas une nouvelle. |
| D-12 | Écran : afficher « X sur Y », le nombre d'introuvables et le nombre de matchs en échec. | Oui, pour tout joueur, sur tous les profils sauf la démo. |
| D-13 | Bouton « Synchroniser maintenant » : lance le cycle delta de tous les joueurs (comme en 8.2.2) et réveille la boucle. | Oui, sans autre changement. Le délai minimal de 2 minutes reste. |

Réservé à l'utilisateur dans tous les cas : fusion dans `main`, tag, publication.

---

## 6. Risques et tests

### 6.1 Risques

| Risque | Parade |
|---|---|
| **Règles d'écriture DuckDB (anti-corruption).** Un nouveau chemin d'écriture par match contournerait `BatchBuilder`. | Les adaptateurs réutilisent `CycleBatchPersister` (Infinite) et `persistBatches` (Halo 5). Le moteur commun n'écrit que `sync_meta`, par `syncmeta.Set`, sous le bail du joueur. Aucun DELETE, aucun `ON CONFLICT`. Gate : `internal/sync/no_art_patterns_test.go` et les gardes `archlint` inchangés, sans nouvelle entrée d'allowlist. |
| **Valeur de 780 Ko dans `sync_meta`.** Comportement de DuckDB sur une grande valeur mise à jour. | Le relevé est écrit une fois et vidé une fois. Les clés qui changent souvent (`history_failures`) sont petites. À vérifier dans la mesure : taille du fichier et du journal après relevé, puis après un CHECKPOINT. |
| **Un seul écrivain par base.** La boucle lit et écrit pendant que la file d'écriture travaille. | Lectures par `duckdb.OpenPlayerReadForQuery` (handle partagé avec l'écrivain). Écritures par l'ouverture sous bail du post-sync (`postSyncPlayerOpener`). Aucun nouveau site d'ouverture en écriture. |
| **Boucle et cycle en même temps sur un joueur.** | Verrou croisé `SyncGate` par joueur et par titre. La file d'écriture est unique : les écritures restent en série. Un match récupéré deux fois est écarté à l'écriture (registre). |
| **Site hébergé, 2 vCPU et 2 Go.** | Un joueur, un lot. Test de mémoire avec trois joueurs (6.2). Mesure sous une limite de 2 Go. |
| **Plusieurs joueurs : famine.** Un joueur à 20 000 matchs retarde les autres. | Tour de rôle par session. Un nouveau joueur attend au plus (nombre de joueurs en rattrapage) sessions. |
| **429.** | Arrêt de session sans tentative comptée, attente doublée de 5 minutes à 1 heure, compteur. Relevés sérialisés. |
| **Halo 5 : entrée non retrouvée à sa position.** | Recherche par identifiant dans la page et ses deux voisines ; sinon échec systémique ; au troisième, nouveau relevé. |
| **Relevé faux ou incomplet.** Page courte au milieu, insertion tardive. | Fin sur page vide seulement. Ré-audit. |
| **Coût du post-sync sur des matchs anciens.** Le recalcul des notes repart-il de loin à chaque fois ? Non mesuré. | Mesure avant de fixer D-10. |
| **Arrêt du serveur ou de l'application en plein lot.** | Le lot en cours n'est pas compté. Au redémarrage, le reste à faire se recalcule. Les tests d'interruption de la 8.2.2 sont rejoués. |
| **Ouvrier de films et verrous existants.** Verrou de décodage, report en partie, passe unique par titre. | L'ouvrier appelle les points d'entrée existants, qui prennent déjà ces verrous. Une seule passe à la fois, par construction. |
| **Frontières de couches.** | Moteur dans `internal/sync/historycatchup`, ports dans `internal/port`, câblage dans `cmd/server`, aucune lecture SQL hors `platform/duckdb` et `knownset`. Gate : `go test ./internal/archlint/...`. |
| **Deux moteurs pendant la transition.** | Une seule version livre la bascule : la boucle arrive et le rattrapage du cycle part dans la même version. Aucun interrupteur (règle 11). |

### 6.2 Tests

**Témoins avant et après.** Les scénarios de bout en bout de la 8.2.2 sont les témoins : même état
final attendu (tous les matchs de la fausse liste au registre, clé de fin rangée, journal d'écriture
vide, aucun doublon). Capturer leurs sorties sur la 8.2.2 avant de toucher au code, puis comparer.

- `cmd/server/sync_v2_convergence_integration_test.go` : 1 000 matchs, arrivées en haut de liste.
- `cmd/server/sync_v2_convergence_interruption_integration_test.go` : arrêts pendant la récupération,
  pendant l'écriture, entre deux lots, redémarrage avec un journal en attente.
- `cmd/server/sync_v2_convergence_difficile_*` : deux joueurs, introuvables, historique déjà complet.
- `internal/games/halo_5/livesync/catchup_convergence_integration_test.go`.

**Tests neufs.**

1. Relevé : aller-retour du format, version inconnue, arrivée pendant la lecture (doublon, pas
   d'oubli), page courte au milieu (pas de fin), page en erreur (rien d'écrit).
2. Reste à faire : relevé moins connu moins introuvables, ordre conservé, exclus ignorés.
3. Échecs : 404 jamais compté ; échec systémique jamais compté ; trois échecs étalés donnent un échec
   durable ; la fin se range avec des échecs durables ; une nouvelle tentative réussie retire l'entrée.
4. **Panne de l'API** : toutes les stats en erreur 500 pendant dix lots, puis API saine. Attendu :
   aucun match perdu, tous rangés.
5. **Trois joueurs en rattrapage** : une seule session à la fois, pic mémoire borné (seuil fixé par la
   mesure), tour de rôle respecté.
6. 429 : la session s'arrête, le délai double, aucune tentative comptée.
7. Contrat de la source du titre : une même suite de tests jouée contre l'adaptateur Infinite et
   l'adaptateur Halo 5.
8. Halo 5 : liste décalée par des arrivées, entrée retrouvée dans une page voisine, entrée disparue.
9. Migration : jeux d'états 8.2.2 (marque-page, échecs, introuvables, abandonnés, compte d'échecs
   Halo 5) ; après migration, les anciennes clés sont vides et aucun match n'est perdu.
10. Ré-audit : un joueur complet avec un trou ; la clé de fin est vidée, le trou est rattrapé.
11. Ouvrier de films : passe sans progrès, le délai double ; passe avec progrès, il revient au
    minimum ; le post-sync du cycle n'appelle plus les étapes film.
12. Ordonnanceur : l'intervalle ne change plus, quel que soit l'état de l'historique.
13. Écran : « X sur Y », introuvables, en échec, état sans nombres, fin, dans chaque langue.

### 6.3 Mesure réelle

Avant de fixer les valeurs (D-7, D-10), comme l'étape de mesure de la 8.2.2.

Règle du projet : un serveur lancé sur une copie des données a d'abord ses chemins de réglages
neutralisés.

1. Copie des données du développeur, clé de fin retirée pour un joueur de plus de 1 000 matchs.
2. Mesurer la prise du relevé : durée, appels de liste, taille de la base avant et après.
3. Mesurer une session de 200 matchs : durée par lot, appels par type, pic mémoire, réponses 429.
4. Mesurer le post-sync après 200 matchs anciens : durée, dont le recalcul des notes.
5. Rejouer avec trois joueurs en rattrapage, processus limité à 2 Go et 2 processeurs.
6. Mesurer l'ouvrier de films : films traités par heure, comparés aux 28 par passe de la 8.2.2.
7. Écrire les résultats dans le plan, à côté du tableau 1.5.

Critères de succès proposés : pic mémoire sous 600 Mo quel que soit le nombre de joueurs ; au moins
1 000 matchs par heure sur le bureau ; aucun 429 au rythme retenu pour le site hébergé.

---

## 7. Proposition de découpage

Ce n'est pas le plan définitif. Le plan se rédige après les décisions de la section 5, sous
`.ai/PLAN_RATTRAPAGE_HISTORIQUE_8_3_0_<date>.md`, et passe la grille `plan-review` avec `levelup-plan`.
Contrat d'exécution : skill `plan-execution`.

**Objectif en une phrase.** Le rattrapage de l'historique passe par un relevé et une boucle à part,
commune à Halo Infinite et Halo 5. Critère observable : les témoins de la 8.2.2 donnent le même état
final, trois joueurs en rattrapage tiennent sous le seuil de mémoire mesuré, et l'écran affiche
« X sur Y ».

**Branche d'intégration** : `release/v8.3.0`. Une branche et un worktree par piste
(`feat/830-rattrapage-<piste>`). Caches : `GOCACHE=%LOCALAPPDATA%\go-build-830<piste>`, clôture par
`powershell -File scripts/disk-hygiene.ps1 -Lot 830<piste> -Apply`.

**Chemin critique** : étapes 0, 1, 2, puis les pistes A et B, puis 7. La piste C (films) ne dépend que
de l'étape 0 : elle démarre en même temps que l'étape 1.

| Étape | Piste | Contenu | Modèle | Effort | Dépend de |
|---|---|---|---|---|---|
| 0 | cadrage | Décisions tranchées, plan rédigé et relu (`plan-review`), témoins de la 8.2.2 capturés, ADR ébauchée | Fable | moyen | section 5 |
| 1 | socle | `historystate` : relevé, échecs, `Complete` sur valeur vide, migration des états 8.2.2 | Sonnet | moyen | 0 |
| 2 | socle | `historycatchup` : boucle, port de la source, arrêts, 429, progression publiée ; tests avec une source simulée | Sonnet | lourd | 1 |
| 3 | A | Adaptateur Infinite, câblage, retrait du rattrapage du cycle V2 | Sonnet | moyen | 2 |
| 4 | B | Adaptateur Halo 5, retrait du rattrapage de `livesync` | Sonnet | moyen | 2 |
| 5 | C | Ouvrier de films, retrait des étapes film du post-sync du cycle | Sonnet | moyen | 0 |
| 6 | D | API et écran « X sur Y », textes | Sonnet | moyen | 2 (port) |
| 7 | intégration | Ordonnanceur : retrait de la cadence de rattrapage et des champs `HistoryPending` | Sonnet | rapide | 3, 4, 5 |
| 8 | E | Ré-audit : commande, action d'administration, passe automatique (si D-3) | Sonnet | moyen | 3, 4 |
| 9 | mesure | Mesure réelle (6.3), valeurs de D-7 et D-10 fixées | Sonnet | moyen | 7 |
| 10 | clôture | Suppressions restantes, garde-rails, aide et `docs/COMMANDS.md` (FR et EN), ADR | Haiku pour le mécanique, Sonnet pour l'ADR | rapide | 9 |
| 11 | revue | Revue adversariale du diff cumulé ; revue par lot pour les étapes 1, 3 et 4 (écritures et synchronisation) | Fable | moyen | 10 |
| 12 | livraison | Gates complets, journal, PR vers `release/v8.3.0` | pilote | moyen | 11 |

### Gates proposés, par étape

Tous depuis `apps/go-api`, sauf mention.

- **Étape 0.** Les témoins sont rangés dans le plan : sorties de
  `go test -tags=integration -run 'Convergence|Difficile|Interruption' -v -timeout 40m ./cmd/server/`
  et de `go test -tags=integration -run Convergence -v ./internal/games/halo_5/livesync/`.
- **Étape 1.** `go build ./... && go test ./internal/sync/historystate/... ./internal/sync/syncmeta/... ./internal/archlint/...`.
  Preuve : test de migration vert sur les cinq jeux d'états de 4.3.
- **Étape 2.** `go test ./internal/sync/historycatchup/... ./internal/port/... ./internal/archlint/...`.
  Preuve : tests 4, 5 et 6 de 6.2 verts avec la source simulée.
- **Étape 3.** `go test ./internal/sync/v2/... ./cmd/server/...` puis
  `go test -tags=integration ./internal/sync/... ./internal/persist/... ./cmd/server/...`.
  Preuve : témoins Infinite au même état final ; `grep -rn "settleHistory\|trackFailures\|Proposed" internal/sync/v2 --include=*.go`
  ne rend rien.
- **Étape 4.** `go test ./internal/games/halo_5/...` puis
  `go test -tags=integration ./internal/games/halo_5/...`.
  Preuve : témoin Halo 5 au même état final ; `grep -rn "BackfillResume\|LoadStreak\|topDelta" internal --include=*.go` ne rend rien.
- **Étape 5.** `go test ./internal/sync/killcollector/... ./internal/sync/replayartifacts/... ./internal/sync/filmbacklog/... ./cmd/server/...`
  (et `-tags=integration`). Preuve : test 11 de 6.2 ; les tests de bornes par profil de la 8.2.2
  restent verts ou sont remplacés un pour un.
- **Étape 6.** `go test ./internal/api/... ./internal/service/...`, `make generate-types` sans diff
  résiduel ; depuis `apps/web` : `npx vitest run src/features/settings --maxWorkers=2` et
  `npx tsc --noEmit -p .`.
- **Étape 7.** `go test ./internal/scheduler/... ./internal/archlint/...`. Preuve :
  `grep -rn "catchUpCycleInterval\|HistoryPending\|cycleDebt" internal cmd --include=*.go` ne rend rien.
- **Étape 8.** `go test ./cmd/levelup/... ./internal/sync/historycatchup/...`. Preuve : test 10 de 6.2.
- **Étape 9.** Tableau de mesures écrit dans le plan ; critères de 6.3 tenus, ou valeurs ajustées et
  décision notée.
- **Étape 10.** `go test ./internal/archlint/...` ; le garde-rail des clés d'historique refuse les deux
  anciens littéraux ; depuis la racine, `make go-api-lint` (aucun fichier touché ne dépasse 500 lignes
  sans exemption) ; depuis `apps/web`, `node tools/lint-cross-feature-imports.mjs`.
- **Étape 12.** `make gate-push`, puis CI verte :
  `gh run list --branch release/v8.3.0 --workflow CI --limit 1`. Entrée dans `.ai/thought_log.md`.

### Points que le plan devra fermer

- Périmètre de chaque étape en liste fermée, statuts `[x]`, `[~]`, `[!]`, section « Découvertes ».
- Le numéro de l'ADR (le suivant libre sur `release/v8.3.0` au moment d'ouvrir) et l'amendement de
  l'ADR 0027 (le cycle ne rattrape plus).
- La date de retrait et le critère de la migration des états 8.2.2 (4.3).
- Le protocole de reprise : ce fichier, le plan, `git log --oneline -10` dans chaque worktree.

---

## Ce que ce document ne sait pas

- Le code de la 8.2.2 APRÈS ses correctifs de revue : non lu. Les listes de la section 4 partent du
  commit `de25cc2063`.
- Le comportement réel de l'API : page courte avant la fin, limite de décalage, seuil des 429.
- Le nombre de joueurs suivis sur le site hébergé, et combien auront encore un rattrapage à faire
  quand la 8.3.0 sortira. Si la 8.2.2 a fini le rattrapage de tous, le gain immédiat de la refonte
  porte sur les nouveaux joueurs, le ré-audit, le site hébergé et la réduction du code.
- Le coût du post-sync sur des matchs anciens (recalcul des notes), jamais mesuré.
- Les débits et les tailles de mémoire de la section 3 : calculés depuis les mesures de la 8.2.2, pas
  mesurés.
