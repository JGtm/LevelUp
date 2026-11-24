# Audit : que deviennent les huit nouvelles actions d'objectif dans LevelUp ?

Lecture seule, version `release/v8.3.0` au commit 0cebc91fef (dossier `LevelUp-wt-int`), décodeur lu : v1.4.1 (celui que LevelUp exige aujourd'hui). La version 1.5.0 en chantier (`dec-v150`) ne contient pas encore ces huit noms : tout ce qui touche leur forme exacte est donc une SUPPOSITION (marquée ainsi).

Les huit noms : `flag_returners_killed`, `flag_kills_as_returner`, `flag_kills_as_carrier`, `zone_offensive_kills`, `zone_defensive_kills`, `vip_kills`, `vip_assists`, `vip_kills_as_vip`.

## 1. Ce qu'il faut comprendre d'abord (PROUVÉ en lisant le code)

1. Il existe DEUX sources d'objectif dans LevelUp, qui ne se touchent pas.
   - La source « feuille de match » : les chiffres officiels de l'API Halo, rangés dans la table `match_objective_stats` (une ligne par joueur et par match, colonnes fixes). Elle contient DÉJÀ des colonnes pour les huit : `flag_returners_killed`, `kills_as_flag_returner`, `kills_as_flag_carrier`, `zone_offensive_kills`, `zone_defensive_kills`, `vip_kills`, `vip_assists`, `kills_as_vip` (migration `internal/games/halo_infinite/migrations/steps_shared_objective_stats.go`). Les noms du film diffèrent seulement pour trois d'entre elles : `flag_kills_as_returner` = `kills_as_flag_returner`, `flag_kills_as_carrier` = `kills_as_flag_carrier`, `vip_kills_as_vip` = `kills_as_vip`.
   - La source « film » : les actions datées du calque `objectives[]` du document de rejeu. Elles ne sont PAS rangées en base. Elles vivent seulement dans le fichier de rejeu cuit. En base on ne range que les enregistrements bruts du film, et les actions nommées sont recalculées à chaque cuisson (`internal/replaybuild/matchfacts.go` ligne 234, `decfilm.NamedEventsFrom`).
2. Donc ces huit actions n'apparaîtront qu'APRÈS une nouvelle cuisson des rejeux, et seulement dans le rejeu. Aucune table, aucune migration, aucun lecteur SQL n'est concerné.
3. Côté serveur Go, personne ne lit `objectives[]` pour calculer une statistique. Le serveur le transmet tel quel (`stat` est une simple chaîne dans le contrat : `api/openapi.yaml` ligne 23089, pas de liste fermée). Les seuls autres lecteurs sont des outils de banc d'essai (voir tableau).
4. Les pages Escouade (Emprise, onglet Objectif), la fiche de match, les citations, le score de participation, les formes d'escouade : toutes lisent la source « feuille de match » par noms de colonnes, avec liste blanche. Elles ne voient jamais le calque du film. Elles sont donc à l'abri du changement du décodeur.
5. Tout ce qui lit le calque du film est dans le rejeu du site (`apps/web/src/features/match-replay`).

## 2. Les pages « Emprise » et « Contrôle » (noms réels)

- « Emprise » = l'onglet Escouade > Emprise, route `/t/{titre}/players/{joueur}/squad/emprise`, composant `features/squad/SquadEmprisePage.tsx` (en anglais « Map control »). Il parle de qui a tenu la carte par les RESSOURCES (bonus, armes spéciales, râteliers) et les placements. Il lit le bloc serveur `squad_emprise`, qui vient des films mais PAS des actions d'objectif nommées. Sa dernière section « Objectif » / « Répartition de l'objectif dans l'escouade » (`features/squad/objectif/`) lit la feuille de match (colonnes `match_objective_stats`).
- « Contrôle » : il n'existe pas de page de ce nom. Le plus proche est « Contrôle des ressources », un graphique de la page Emprise ci-dessus. « Mainmise » n'existe nulle part dans le code. La page Tactique ne lit aucune action d'objectif. Si le propriétaire pensait à une autre page, il faut me donner son nom d'écran.
- Effet des huit actions sur ces pages : aucun. Elles n'entrent pas dans leurs calculs. (PROUVÉ : Go `analysis/narrative/objective_roles.go`, `objective_participation.go`, `squadformes/formes.go`, `platform/duckdb/objective_stats_repo.go` ; web `features/squad/objectif/*`.)
- Information utile : ces pages comptent DÉJÀ les éliminations liées à l'objectif comme des actions d'objectif, par choix écrit, via les colonnes de la feuille de match. Rôle « prendre » : `flag_returners_killed`, `zone_offensive_kills`, `vip_kills`, `vip_assists`. Rôle « défendre » : `flag_carriers_killed`, `zone_defensive_kills`, `kills_as_vip`. Poids du score de participation : `flag_returners_killed` 1,5 ; `zone_*_kills` 1,0 ; `vip_kills` 3,0 ; `vip_assists` 1,0 ; `kills_as_vip` 2,0. Écartées volontairement : `kills_as_flag_carrier` et `kills_as_flag_returner` (commentaire au-dessus de `ObjectiveFamilyActionWeights`, `objective_participation.go`).

## 3. Tableau des lecteurs

Légende : (a) ignore proprement, (b) compterait à tort, (c) casse, (d) devrait afficher et ne le fait pas. « Film » = calque `objectives[]` ; « feuille » = colonnes API.

| Lecteur (fichier:ligne) | Page ou calcul visible | Traitement actuel | Verdict | Ce qu'il faut faire |
|---|---|---|---|---|
| `web/.../match-replay/layers/objectivesLayer.ts:394-401` `buildObjectivePulses` | Rejeu : anneaux qui s'ouvrent sur la zone la plus proche de l'auteur | Prend TOUTE action dont le nom commence par `flag_`, `zone_`, `hill_`, `skull_`, `vip_`, `bomb_` (`objectiveFamilyOf`). Seuls les `flag_` sont retirés si les portages de drapeau existent (`dropFlags`, ligne 399) | **(b)** : chaque `zone_offensive_kills` / `zone_defensive_kills` dessinerait un anneau « capture » sur la zone. Les `vip_*` aussi s'il y a des éléments. Les 3 `flag_*` aussi sur un film CTF sans portages publiés | Remplacer le filtre « tout préfixe de famille » par une liste blanche des actions SUR l'objectif (captures, sécurisations, prises, retours, vols, détonations). Test à ajouter |
| `web/.../model/objectiveFeedLogic.ts:39-93` `OBJECTIVE_FEED_STATS` + `objectiveFeedEntries` | Rejeu : fil d'actions (lignes « a capturé une base », etc.) | Liste fermée de 11 noms. Une action de famille connue mais hors liste est ÉCARTÉE et comptée dans `unlabelled` (ligne 89-91). Ce compteur n'est lu par personne (`useFeedEventEntries.ts:46` ne garde que `.entries`) | (a) à l'exécution : ligne absente, sans erreur. Mais (d) : rien ne s'affiche | Ajouter les 8 noms si on veut les afficher (voir section 6) |
| `web/.../model/objectiveFeedStats.guard.test.ts` + `test/film-source/internal/facts/objectives/named.go.txt` | Test de parité Go/TS | Compare la liste fermée ci-dessus aux constantes `Stat*` du fichier `named.go` du décodeur, copié par `go run ./cmd/film-web-snapshot` | **(c) voulu** : le test devient rouge dès que la copie est régénérée avec les 8 noms, tant qu'ils ne sont pas dans la liste ET traduits | Prévu à la montée : régénérer la copie, puis ajouter les 8 noms (liste + fr + en). Ne pas contourner |
| `web/.../i18n/i18nContract.ts:168` `feedObjectiveAction: Record<ObjectiveFeedStat,string>` ; `i18n/fr.ts:63-75`, `en.ts:63-75` | Libellés du fil du rejeu | Type fermé : ajouter un nom à la liste sans libellé fr ET en ne compile pas | **(c) voulu** (parité par le typage) | Voir section 6 |
| `web/.../model/objectiveMark.ts:87-90` `EVENT_STATS` | Rejeu : marque d'objectif sur la fiche d'un joueur vivant | Liste blanche : `zone_captures`, `zone_secures`, `bomb_detonations` | (a) | Rien |
| `web/.../sound/objectiveSound.ts:78` `OBJECTIVE_SOUND_STEMS` | Rejeu : sons d'objectif | Liste blanche de 6 noms ; tout nom absent = silence (règle écrite en tête du fichier) | (a) | Rien. Ne pas inventer de son pour les éliminations |
| `web/.../layers/flagCaptureFx.ts:42,69`, `bombBlastFx.ts:51,77`, `useReplayBombCarrier.ts:83` | Rejeu : onde de capture, explosion, sol de la bombe | Test d'égalité sur UN nom précis (`flag_captures`, `bomb_detonations`) | (a) | Rien |
| `web/.../model/carrierKillFold.ts:27,35` | Rejeu : badge « porteur éliminé » sur la ligne de frag | Égalité sur `flag_carriers_killed` seul | (a) | Modèle à réutiliser pour les nouvelles éliminations (section 6) |
| `web/.../ui/ReplayObjectiveLine.tsx:95-99` | Rejeu : rendu d'une ligne du fil | Lit `feedObjectiveAction[stat]` ; ne reçoit que des noms de la liste fermée | (a) tant que la liste est fermée | Si on ajoute un nom, il a son libellé par le typage |
| `web/.../model/objectiveFamilies.ts` (6 familles, parité avec le décodeur) | Rejeu : préfixe de famille | Reconnaît par préfixe. Les 8 noms portent tous un préfixe connu (`flag_`, `zone_`, `vip_`) : aucune famille nouvelle, parité intacte | (a) | Rien |
| `go-api/internal/replaybuild/matchfacts.go:231-261` `identifiedEvents` | Cuisson du rejeu | Passe `decfilm.ObjectiveTypeOf(facts.GameVariantName)` au décodeur. Le compte `coverage.objectives` (`CountObjectiveFamily`) est restreint aux noms à préfixe de famille | Famille VIP : voir section 5. Compteurs : (a) mais ils augmenteront | Voir sections 4 et 5 |
| `go-api/internal/domain/replaydoc/objectives.go:9-14` `ObjectiveAction.Stat` + `api/openapi.yaml:23086` | Contrat du rejeu | Simple chaîne, aucune liste fermée | (a) | Rien |
| `go-api/internal/replaydiff/empreinte_axes.go:199-215` | Outil de comparaison d'artefacts | Compte chaque `stat` par joueur sans filtre | (a), mais la comparaison avant/après montrera 8 nouvelles lignes | Attendu, à lire comme une addition |
| `go-api/internal/replayverite/violations_vies.go:159` `acteursNommes` ; `scores.go:156` | Banc de vérité | `violations_vies` prend TOUTES les actions comme « acteur nommé » ; `scores.go` ne lit que `flag_captures` | (a) pour `scores.go`. Pour `violations_vies` : les nouvelles actions entrent dans la vérification « l'acteur est-il présent à cet instant ? » | Surveiller le premier passage du banc sur le corpus (outil de dev, rien de visible) |
| `go-api/cmd/replay-corpus-gate/verdict_metriques.go:136` (famille `coverage.objectives.`) | Gate corpus avant montée | Surveille des métriques de couverture d'objectifs | Les dénominateurs montent (plus d'actions de famille) : le gate signalera un écart | Prévoir la décision d'acceptation écrite dans la configuration du gate (règle de la mémoire projet) |
| `go-api/internal/sync/comeback.go:108-125` | Badge de retournement (SABORDAGE / ABNÉGATION) | Utilise `decfilm.ObjectiveTypeOf` et `objectiveType != ""` pour décider d'appliquer le badge « contraste de frags » aux modes à objectif. N'utilise PAS les actions nommées ; la courbe vient de `match_objective_events` (`event_type='capture'`, ligne 462) | (a) pour les 8 actions. **Risque indirect (b)** si le décodeur choisit de faire renvoyer `"vip"` par `ObjectiveTypeOf` : les matchs VIP recevraient alors ces badges | Voir section 5 |
| `go-api/internal/persist/*`, `migration/steps_shared_objective_stats.go`, `platform/duckdb/objective_stats_repo.go`, `queries_match_objectives.go`, `sync/citations.go:381`, `analysis/narrative/objective_*.go`, `analysis/squadformes/formes.go`, `sessionusage/objective_row.go` | Fiche de match, Escouade (Emprise > Objectif, Formes), citations, score de participation, séries temporelles | Colonnes API explicites ; les colonnes inconnues sont ignorées | (a) : ne dépendent pas du film | Rien (mais voir section 6, cas d) |
| `go-api/internal/platform/duckdb/objective_events_repo.go` et table `match_objective_events` | Badge de retournement, vue match (anciennes captures) | Reçoit n'importe quelle chaîne `event_type`, mais alimentée par l'ancien extracteur (`objectiveevent`, `diag_weapons_v3`) et la bombe, PAS par les actions nommées | (a) | Rien |

## 4. Le rangement en base (question 3)

- La table qui reçoit les actions par joueur est `match_objective_stats` (append-only, vue `_latest`). Elle n'est PAS alimentée par le film : elle l'est par l'API Halo (`cmd/backfill_objective_stats`, persister `internal/persist/shared_persister.go:558-570`). Colonnes fixes : une liste fermée, pas un « n'importe quel nom ».
- Les lecteurs SQL filtrent par nom de colonne, explicitement (`objective_stats_repo.go:50-62`, `queries_match_objectives.go:29-31`, `sync/citations.go:381`). Aucune requête n'additionne « toutes les colonnes ».
- Les actions nommées du film ne sont dans aucune table. Aucune migration à prévoir.
- Les compteurs de couverture `coverage.objectives.*` (dans le document de rejeu) bougeront, puisque les huit noms ont un préfixe de famille. Aucun écran ne les affiche (PROUVÉ : aucune lecture de `noSlot`, `refusedByRoster`, `coverage.objectives` côté web).

## 5. Le VIP (question 4)

- PROUVÉ : aujourd'hui LevelUp ne passe PAS la famille `vip` au décodeur pour nommer les actions. `identifiedEvents` (`matchfacts.go:234`) passe `decfilm.ObjectiveTypeOf(variante)`. Dans le décodeur v1.4.1, cette fonction (`internal/facts/objectives/extract.go:160-184`) ne reconnaît que drapeau, zones, colline, crâne et bombe. Un VIP renvoie `""`, donc `NamedEventsFrom` rend `nil`.
- Conséquence : `vip_selected` n'apparaît JAMAIS dans `objectives[]` aujourd'hui, alors que le site a déjà son libellé (« a été désigné VIP »). Le VIP du rejeu passe par un autre chemin : `replay.GardesDeLaVariante(...).VIP` (la variante contient « vip ») puis le calque `vipCrown` (`matchfacts.go:157`, `replay/vip_crown.go`).
- Confirmation côté chantier : dans `dec-v150`, `ObjectiveTypeVip` n'est pas exporté par la façade `decfilm` (v1.4.1 non plus ; `decfilm.go:355-364` n'exporte que Bomb, Flag, Hill, Skull, Zone).
- Où se décide la famille côté LevelUp : un seul endroit pour les actions nommées, `matchfacts.go:234`. Autres appelants de `ObjectiveTypeOf` : `sync/comeback.go:108`, `replaybuild/zones.go:135`, `cmd/replay-equiv/child.go:268`, `cmd/zone-attribution`.
- Deux façons de donner VIP au décodeur, avec leurs conséquences (SUPPOSITION sur ce que le décodeur choisira) :
  1. Le décodeur exporte `ObjectiveTypeVip` et LevelUp passe `"vip"` quand la variante est un VIP (`GardesDeLaVariante`). Un symbole de plus dans la façade `decfilm` : le garde-rail `internal/archlint/film_facade_surface_test.go` (plafond 182, mesure exacte) rougira jusqu'à son relèvement daté. Les autres appelants ne changent pas.
  2. Le décodeur fait renvoyer `"vip"` par `ObjectiveTypeOf`. Alors `comeback.go:121` (`objectiveType != ""`) appliquerait SABORDAGE / ABNÉGATION aux matchs VIP : effet de bord visible à surveiller. (Le calcul ne touche que les matchs sans drapeau de retournement déjà posé.)
  
  La première est plus sûre pour LevelUp.

## 6. Ce qu'il faut faire

### A. À corriger AVANT de monter le décodeur (cas b et c)

1. (b) `buildObjectivePulses` (`objectivesLayer.ts:394-401`) : passer d'un filtre par préfixe de famille à une liste blanche d'actions sur l'objectif, sinon les éliminations de zone feront des anneaux de capture. Ajouter un test qui donne un `zone_offensive_kills` et attend zéro pulse. C'est le seul vrai « compte à tort » trouvé.
2. (c, voulu) À la régénération de la copie `film-source` (`go run ./cmd/film-web-snapshot`), le test `objectiveFeedStats.guard.test.ts` rougira : prévoir dans le même lot l'ajout des 8 noms à `OBJECTIVE_FEED_STATS`, leurs libellés fr et en, et leur glyphe (`OBJECTIVE_FAMILY_GLYPH` suit déjà la famille). Le typage (`Record<ObjectiveFeedStat,string>`) interdit d'oublier une langue.
3. VIP : trancher avec la session du décodeur COMMENT la famille VIP est passée (section 5), avant de coder `matchfacts.go:234`. Si `decfilm` exporte un symbole, relever le plafond de `film_facade_surface_test.go` avec sa date.
4. Gate corpus : écrire la décision d'acceptation pour le mouvement attendu de `coverage.objectives.*` (plus d'actions de famille).
5. Ne pas oublier : les 8 actions n'existent qu'après recuisson des rejeux, et la recuisson du parc se fait une seule fois, avec accord des sessions (règle projet). Avant, rien ne change à l'écran.

### B. Ce qu'on peut faire pour les AFFICHER (cas d)

Dans le rejeu (film) :
- Ajouter les 8 noms au fil du rejeu. Libellés proposés (forme de la liste existante : passé composé, après le nom du joueur, aucun possessif) :
  - `flag_returners_killed` : « a éliminé un joueur qui rendait le drapeau » / « killed a flag returner »
  - `flag_kills_as_returner` : « a éliminé un joueur en rendant le drapeau » / « scored a kill while returning the flag »
  - `flag_kills_as_carrier` : « a éliminé un joueur en portant le drapeau » / « scored a kill while carrying the flag »
  - `zone_offensive_kills` : « a éliminé un défenseur de base » / « scored an offensive stronghold kill »
  - `zone_defensive_kills` : « a éliminé un attaquant de base » / « scored a defensive stronghold kill »
  - `vip_kills` : « a éliminé le VIP adverse » / « killed the enemy VIP »
  - `vip_assists` : « a aidé à éliminer le VIP adverse » / « assisted in killing the enemy VIP »
  - `vip_kills_as_vip` : « a éliminé un joueur en étant VIP » / « scored a kill as the VIP »
  (Les libellés sont une proposition ; le sens exact de `zone_offensive` / `zone_defensive` est à confirmer par le décodeur avant de figer le français. Se placer dans `features/match-replay/i18n/fr.ts` et `en.ts`, bloc `feedObjectiveAction`.)
- Danger de doublon : ces actions sont des éliminations. Elles redisent la ligne de frag du fil, comme `flag_carriers_killed` que le site replie déjà en badge sur le frag (`carrierKillFold.ts`). Les afficher comme lignes séparées doublerait le fil. Recommandation : reprendre le même principe de repli (badge sur la ligne de frag du même tueur) plutôt que des lignes neuves.
- Aucun son : les éliminations n'ont pas de son d'objectif propre (même règle que `flag_carriers_killed`).

Hors film, déjà possible sans attendre le décodeur (données déjà en base) :
- Fiche de match (`features/match-view/MatchScoreboard.logic.ts:171-230`) : les colonnes affichées sont en liste fermée par mode ; `flag_carriers_killed`, `flag_returners_killed`, `zone_offensive_kills`, `zone_defensive_kills`, `vip_assists` ne sont pas affichées bien que présentes dans le DTO (le commentaire le dit pour Stockpile/Extraction/VIP). Seul `vip_kills` est affiché. À décider produit, pas technique.
- Timeseries (`TimeseriesObjectiveCard.tsx`) : seulement captures, retours, vols, zones, crâne.

### C. Ce qui ne bouge pas (PROUVÉ)

- Toutes les pages Escouade (Emprise et son onglet Objectif, Contributions, Dynamique, Formes), Tactique, Accueil, Sessions, Citations, Carrière, Explorer : indépendantes des actions du film.
- Score de participation / emprise d'objectif (`narrative`) : lit les colonnes API par liste de poids, pas le film.
- Base de données : aucune migration, aucune table, aucune requête.
- Contrat d'API : `stat` reste une chaîne libre ; le client TypeScript généré ne change pas.
- Sons, ondes de capture, explosion de bombe, marque d'objectif de la fiche joueur, portage du drapeau : listes blanches sur des noms précis.
- Badge de retournement : ne lit pas les actions nommées.

## 7. PROUVÉ ou SUPPOSÉ

PROUVÉ en lisant le code (v1.4.1 du décodeur, `release/v8.3.0`) :
- Les actions nommées ne sont pas rangées en base ; la table d'objectifs est alimentée par l'API et à colonnes fixes.
- Tous les lecteurs Go et web listés ci-dessus, leur mode de filtrage (liste blanche, égalité de nom, ou préfixe), et le cas (b) de `buildObjectivePulses`.
- Le VIP n'est pas passé au décodeur pour nommer les actions ; `vip_selected` n'est donc pas publié aujourd'hui dans `objectives[]`.
- Le test de parité web devient rouge à la régénération de la copie `film-source`.
- Aucune page ne s'appelle « Contrôle » ou « Mainmise » ; « Emprise » est l'onglet Escouade décrit plus haut.

SUPPOSÉ (non vérifiable avant la sortie du décodeur 1.5.0) :
- Que les 8 noms soient des constantes `Stat*` de `named.go` (le test de parité les attendrait alors).
- La façon dont le décodeur transmettra la famille VIP (symbole exporté ou `ObjectiveTypeOf` modifié).
- Le sens exact en français de `zone_offensive_kills` / `zone_defensive_kills` et le moment daté de chaque action (libellés proposés à confirmer).
- Qu'aucun film du corpus n'ait un nom de variante VIP contenant « ctf », « flag » ou « bomb » (cela n'a pas été balayé sur les données).
- Le détail du mouvement des métriques `coverage.objectives.*` au gate corpus (non exécuté).
