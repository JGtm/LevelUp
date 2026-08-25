# PLAN — Compteur « amis en jeu » : les amis sont les joueurs INSCRITS DANS L'APP

> Écrit le 2026-08-25. Branche : `wt/amis-app` (worktree
> `C:/Users/Guillaume/Downloads/Scripts/LevelUp-wt-amis-app`, base `3cafdfbe8` =
> origin/feat/v75). Contrat : skill `plan-execution` — statuts `[x]`/`[~]`/`[!]`,
> aucune case vide, zéro fix hors périmètre (→ Découvertes), pas de `git add -A`,
> pas de push, ni thought_log ni REGISTRE (superviseur).

## Décision produit (utilisateur, 2026-08-25 — CORRIGE D-P4 du chantier notion5)

La pastille « N amis en jeu » ne compte PLUS la liste `friend_gamertags` des
Réglages. **« Mes amis » = les joueurs inscrits dans l'app qui ne sont pas les
miens** : pour l'utilisateur courant, le compte = joueurs de l'instance dont il
n'est PAS propriétaire (ADR 0029), actuellement EN JEU sur un titre supporté.
Chaque utilisateur voit donc SON compte (personnalisé par la propriété), et la
donnée vient du watcher déjà en place — **zéro appel Xbox supplémentaire**.
Conséquences : la mécanique « présence des amis par lot Xbox » livrée la veille
(batch + résolution gamertag→xuid + cache TTL + singleflight) n'a plus AUCUN
consommateur → suppression complète (règle 0 code mort). Le constat de revue
« liste globale d'admin servie à tout authentifié » disparaît par construction.

### Précision utilisateur post-lancement (2026-08-25) — RÈGLE QUI FAIT FOI

Une instance héberge des utilisateurs qui ne sont PAS amis entre eux : « mes
amis » ne sont donc PAS « tous les autres joueurs inscrits », mais **les joueurs
de MON CERCLE au sens ADR 0029** — exactement ceux que je vois déjà dans mon
sélecteur (les miens + ceux de mes co-membres de groupe, chokepoint
`BootstrapService.OwnedPlayers` / `resolveCoMembers`).

**`friends_in_game` = joueurs VISIBLES pour l'utilisateur courant selon ce
chokepoint, MOINS ceux dont il est directement propriétaire, actuellement EN JEU
(même prédicat de titre et de fraîcheur que la manette).** Conséquences :

- un utilisateur sans groupe ne voit que ses joueurs → compte 0 ;
- un étranger à mon groupe n'entre jamais dans mon compte, ni moi dans le sien ;
- aucune identité hors cercle ne transite (le compte est un entier ; la liste
  `players` reste filtrée comme avant) ;
- la distinction « possédé en propre » vs « visible via co-membre » n'existant
  pas sur la liste servie, elle est prise au chokepoint authz existant
  (`authz.Enforced` + `authz.CurrentUser`, via `BootstrapService.OwnsPlayerDirectly`)
  — la logique de groupe n'est PAS dupliquée.

Cette règle remplace la formulation « état GLOBAL du watcher moins les possédés »
du lancement, qui compterait les étrangers au groupe.

### Correctif du 2026-08-25 (lot H, décision superviseur GRAVÉE) — les deux régimes

La règle ci-dessus suppose des identités. Sans elles (`LEVELUP_AUTH_MODE=none`,
la valeur PAR DÉFAUT, ou mode démo, ou aucun user store) il n'existe PAS de
« possédé en propre » : `OwnsPlayerDirectly` rend donc FAUX pour tout profil, et
le compteur vaut **tous les joueurs visibles en jeu** — l'instance entière EST le
cercle de son opérateur. Le comportement d'origine (« sans enforcement, tous les
profils sont les siens ») mettait la pastille à zéro EN PERMANENCE sur la
configuration par défaut : une fonctionnalité livrée éteinte (règle n°11).

| Régime | `OwnsPlayerDirectly` | `friends_in_game` |
|---|---|---|
| Propriété appliquée (`password` / `xbox` + user store) | vrai pour le xuid lié de l'utilisateur | cercle visible MOINS mes profils |
| Propriété non appliquée (`none`, démo, pas de store) | toujours faux | tous les joueurs visibles en jeu |

Dans les deux cas la borne reste la LISTE VISIBLE (`OwnedPlayers`, scopée par
`X-LevelUp-Title`) : le compteur ne voit jamais plus loin que la liste `players`
servie à côté.

## Items

- [x] G1 — Go service : remplacer le calcul de `friends_in_game` dans le service
  de présence (`internal/service/presence_service.go` + `server_presence.go`) :
  compte = joueurs suivis par le watcher, EN JEU (mêmes règles de fraîcheur et de
  titre que la manette — réutiliser exactement le même prédicat), dont le
  propriétaire n'est PAS l'utilisateur courant. La notion « pas les miens »
  s'appuie sur le même chokepoint de propriété que la liste `players` servie
  (BootstrapService.OwnedPlayers / authz) — PAS de logique de slug, PAS de
  duplication du prédicat de propriété : dériver « amis » = joueurs présents dans
  l'état watcher MOINS ceux que l'utilisateur possède. Joueurs démo/auth-only :
  déjà absents du watcher (SyncablePlayers) — le vérifier sur pièces et l'écrire
  en commentaire, pas en re-filtre.
- [x] G2 — Go suppression : retirer la chaîne morte `friend_gamertags` → présence :
  `internal/presence/batch_client.go` (+ son entrée de surface netguard si elle
  devient orpheline), `internal/service/presence_friends.go` (compteur, cache,
  singleflight, backoff), le résolveur gamertag→xuid de `server_presence.go`
  (`friendXUIDResolverFrom`, `friendPresenceFrom`) et leur câblage wire, ainsi
  que TOUS leurs tests. `friend_gamertags` lui-même (Réglages, « avec amis »)
  reste intouché — seuls ses usages PRÉSENCE meurent. Grep final :
  aucun symbole orphelin, aucun import mort.
- [x] G3 — contrat : `friends_in_game` garde son nom et son type (int) mais sa
  sémantique change → mettre à jour le godoc du DTO (`domain/presence.go`) et le
  commentaire OpenAPI ; régénérer `openapi.yaml` + `make generate-types` ; gate
  openapi vert. Pas de renommage de champ (le front est déjà branché dessus).
- [x] G4 — web : rien à changer fonctionnellement (le hook et la pastille lisent
  `friends_in_game`). Vérifier les libellés : « N amis en jeu » / « N friends in
  game » restent justes ; ajuster l'infobulle SEULEMENT si elle nomme la liste
  des Réglages (vérifier sur pièces).
- [x] G5 — tests : Go — deux utilisateurs différents obtiennent des comptes
  différents sur le même état watcher (A possède p1 : voit p2+p3 en jeu = 2 ;
  B possède p2 : voit p1 [hors jeu] + p3 = 1) ; utilisateur propriétaire de tout
  → 0 ; watcher éteint → 0 ; la fraîcheur (borne 3 min) s'applique au compte
  comme à la manette. Adapter/retirer les tests des chemins supprimés. Web —
  vitest existants du PlayerSwitcher verts (adaptés si un libellé bouge).
- [x] G6 — gates du lot :
  `cd apps/go-api ; go vet ./... ; go test ./internal/presence/... ./internal/watcher/... ./internal/service/... ./internal/api/... ./internal/platform/netguard/... ./contracttest/...`
  puis `cd apps/web ; npx tsc -b --force ; npx eslint <touchés> ; npx vitest run <PlayerSwitcher + shell>`.
  0 erreur, 0 nouveau warning. Codes retour capturés SANS pipe.

## Journal d'exécution

**G1 (2026-08-25)** — le compteur sort désormais de la MÊME boucle que la liste
`players` : pour chaque joueur visible en jeu (même prédicat `TitleSlug != ""`,
même borne de fraîcheur `presenceFreshnessWindow`), +1 si l'utilisateur n'en est
pas le propriétaire direct. Prédicat de propriété injecté au composition root
(`BootstrapService.OwnsPlayerDirectly`, nouveau fichier `bootstrap_ownership.go`
— `bootstrap_service.go` est à 614 L, dette gelée, on ne l'agrandit pas), pas de
logique de groupe dupliquée dans le service. `PresenceService.WithFriends` prend
maintenant ce prédicat au lieu du compteur Xbox ; budget de 3 s, goroutine et
`friendsBudget` supprimés (plus aucun appel sortant sur ce chemin). Vérifié sur
pièces : le daemon watcher ne suit que `domain.SyncablePlayers`
(`cmd/server/main.go:2124`) et `OwnedPlayers` retire déjà les profils auth-only —
écrit en commentaire, aucun re-filtre ajouté. Deux tests du chemin supprimé
(`FriendsCountIncluded`, `BlockingFriendsSource`) retirés ici par nécessité de
compilation ; la couverture neuve arrive en G5.
Gates locaux : `go build ./...` EXIT=0, `go vet ./internal/service/... ./internal/api/...` EXIT=0.

**G2 (2026-08-25)** — chaîne `friend_gamertags` → présence supprimée : 5 fichiers
effacés (`service/presence_friends.go` + test, `presence/batch_client.go` + test,
`watcher/daemon_presence_batch.go`), 3 symboles retirés de fichiers vivants
(`GamertagRepo.ResolveXUIDsByGamertags` + son test d'intégration + l'import
`analysis` devenu mort, `ServiceRegistry.FriendGamertags`, le test
`TestPresenceBatch_NoClient_ReturnsSentinelError`), et `buildPresenceService`
réduit de 5 à 2 paramètres. `friend_gamertags` lui-même est intact : le résolveur
interne `friendGamertagsResolver` garde ses 5 appelants (Escouade, carrière).
Entrée netguard : SANS OBJET — `batch_client.go` portait un `netguard.Check`, il
n'a jamais eu d'entrée d'allowlist (vérifié dans `netguard_coverage_test.go`) ;
le ratchet reste vert. Grep final : aucun symbole orphelin de la chaîne (les
occurrences restantes de `WithFriendXUIDResolver` appartiennent à la carrière,
chaîne distincte, hors périmètre).
Gates locaux : `go build ./...` EXIT=0 ; `go test` des 5 paquets du lot EXIT=0.

**G3 (2026-08-25)** — `friends_in_game` garde son nom et son type (`int`, requis
au schéma) ; seule sa SÉMANTIQUE change. Godoc de `domain.PresenceSnapshot`
réécrit, godoc du handler aligné, et le champ porte désormais un tag `doc:` — le
schéma n'avait aucune description, il en a une maintenant (diff openapi.yaml : 1
ligne ajoutée, aucune autre). `make openapi-gen` EXIT=0, `make generate-types`
EXIT=0 (4 lignes dans generated.ts), `make openapi-check` EXIT=0 (contrat à jour
+ generated.ts dérivé).

**G4 (2026-08-25)** — vérifié sur pièces : le libellé `common.shell.friends_in_game`
(« # ami/amis en jeu » / « # friend(s) in game ») et l'infobulle du badge (même
chaîne, portée par `title` et `aria-label`) ne nomment PAS la liste des Réglages
— rien à changer, aucun libellé touché, aucun test vitest à adapter. Un seul
écart trouvé : le godoc de `usePresence.ts` disait « le nombre d'amis (liste des
Réglages) » et décrivait un TTL Xbox de 45 s ; corrigé (commentaire uniquement,
zéro changement fonctionnel).

**G5 (2026-08-25)** — `internal/service/presence_friends_test.go` réécrit de zéro
(8 tests) : compte PERSONNEL (même état watcher, A possède p1 → 2, B possède p2 →
1) ; propriétaire de tout → 0 ; watcher éteint sous 3 formes → 0 avec `players`
non nil ; borne de fraîcheur appliquée au compte comme à la manette ; prédicat de
propriété absent → 0 ; profil sans xuid non compté ; session absente → 0 et liste
vide (fail-closed) ; ÉTRANGER HORS GROUPE jamais compté — ce dernier joué à
travers le VRAI `BootstrapService` (db_profiles temporaire, user store, résolveur
de co-membres), le seul endroit où la frontière du cercle est décidée : alice et
bob (même groupe) comptent 1 chacun, carol (hors groupe) compte 0 et n'apparaît
dans aucune des deux listes. Ajouté aussi `TestOwnsPlayerDirectly_...` :
propriété ≠ visibilité (co-membre, admin, xuid vide, session nil, enforcement
désactivé). Tests des chemins supprimés : retirés en G1/G2 (compilation).
Gate : `go test ./internal/service/ -run "Friends|OwnsPlayerDirectly"` EXIT=0.

**G6 (2026-08-25)** — gates du lot, codes retour capturés sans pipe (sorties
redirigées vers fichier, `echo $?` immédiat) :

| Gate | Commande | Résultat |
|---|---|---|
| Go vet | `go vet ./...` | EXIT_VET=0, sortie vide |
| Go tests | `go test ./internal/presence/... ./internal/watcher/... ./internal/service/... ./internal/api/... ./internal/platform/netguard/... ./contracttest/...` | EXIT_GOTEST=0 — 12 paquets `ok`, 0 FAIL, 2 sans test |
| Types web | `npx tsc -b --force` | EXIT_TSC=0, sortie vide |
| Lint web | `npx eslint src/components/shell/usePresence.ts src/lib/api/generated.ts` | EXIT_ESLINT=0, 0 warning |
| Vitest | `npx vitest run src/components/shell/` | EXIT_VITEST=0 — 16 fichiers, 153 tests passés |
| Contrat | `make openapi-check` (G3) | EXIT=0 — openapi.yaml à jour + generated.ts dérivé |

Seul message non vert des sorties : l'avertissement Vite `configLoader: 'native'`
(`__dirname` dans vite.config.ts), préexistant et hors périmètre.

## LOT H — correctifs de revue (2026-08-25)

Issus de deux relectures adversariales du lot G. Chaque constat a été VÉRIFIÉ SUR
PIÈCES avant correction ; rien d'autre n'a été introduit.

- [x] H1 (P1) — `bootstrap_ownership.go` : `OwnsPlayerDirectly` rend FAUX quand
  la propriété n'est pas appliquée (cf. « Correctif du 2026-08-25 » ci-dessus).
- [ ] H2 (P2) — justification écrite du `return false` (utilisateur sans xuid).
- [ ] H3 (P2) — contrat : périmètre de titre explicite.
- [ ] H4 (P2) — identité résolue UNE fois par requête.
- [ ] H5 — test de jonction sur `buildPresenceService`.

### Journal du lot H

**H1 (2026-08-25)** — constat vérifié : `authz.Enforced` est faux dès
`AuthMode="none"` (`config.go:241` — c'est le DÉFAUT), et
`OwnsPlayerDirectly` rendait alors `true` pour tout profil ⇒ `isFriend` toujours
faux ⇒ pastille à zéro en permanence sur la configuration par défaut. Corrigé en
`false`, avec le commentaire daté des deux régimes. Le test
`TestOwnsPlayerDirectly_...` ne verrouille plus l'ancien comportement (l'assertion
« sans enforcement, tous les profils sont les siens » est retirée) ; deux tests
neufs le remplacent : `TestOwnsPlayerDirectly_NotEnforced_OwnsNothing` (la
propriété) et `TestFriendsInGame_NotEnforced_CountsEveryVisiblePlayerInGame`
(la conséquence : 2 joueurs visibles en jeu → compteur 2, via un
`BootstrapService` réel en mode `none`). Section décision du plan mise à jour.
Gate : `go test ./internal/service/ -run "Friends|OwnsPlayerDirectly|FilterOwnedPlayers|Presence"` EXIT=0.

## Découvertes (à consigner, ne pas traiter)

- **Cas admin (G1)** : un administrateur voit TOUT le parc dans son sélecteur
  (`CanAccessPlayer` accorde l'accès sur le rôle) ; par la règle « visibles moins
  possédés en propre », son compteur inclut donc des joueurs hors de son cercle
  social. Conforme à la règle gravée (« ce que je vois dans mon sélecteur, moins
  les miens ») et sans fuite d'identité (le compte est un entier), mais c'est un
  cas à trancher si le produit veut un compte « social » strict pour les admins.
