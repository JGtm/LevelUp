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
- [ ] G2 — Go suppression : retirer la chaîne morte `friend_gamertags` → présence :
  `internal/presence/batch_client.go` (+ son entrée de surface netguard si elle
  devient orpheline), `internal/service/presence_friends.go` (compteur, cache,
  singleflight, backoff), le résolveur gamertag→xuid de `server_presence.go`
  (`friendXUIDResolverFrom`, `friendPresenceFrom`) et leur câblage wire, ainsi
  que TOUS leurs tests. `friend_gamertags` lui-même (Réglages, « avec amis »)
  reste intouché — seuls ses usages PRÉSENCE meurent. Grep final :
  aucun symbole orphelin, aucun import mort.
- [ ] G3 — contrat : `friends_in_game` garde son nom et son type (int) mais sa
  sémantique change → mettre à jour le godoc du DTO (`domain/presence.go`) et le
  commentaire OpenAPI ; régénérer `openapi.yaml` + `make generate-types` ; gate
  openapi vert. Pas de renommage de champ (le front est déjà branché dessus).
- [ ] G4 — web : rien à changer fonctionnellement (le hook et la pastille lisent
  `friends_in_game`). Vérifier les libellés : « N amis en jeu » / « N friends in
  game » restent justes ; ajuster l'infobulle SEULEMENT si elle nomme la liste
  des Réglages (vérifier sur pièces).
- [ ] G5 — tests : Go — deux utilisateurs différents obtiennent des comptes
  différents sur le même état watcher (A possède p1 : voit p2+p3 en jeu = 2 ;
  B possède p2 : voit p1 [hors jeu] + p3 = 1) ; utilisateur propriétaire de tout
  → 0 ; watcher éteint → 0 ; la fraîcheur (borne 3 min) s'applique au compte
  comme à la manette. Adapter/retirer les tests des chemins supprimés. Web —
  vitest existants du PlayerSwitcher verts (adaptés si un libellé bouge).
- [ ] G6 — gates du lot :
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

## Découvertes (à consigner, ne pas traiter)

- **Cas admin (G1)** : un administrateur voit TOUT le parc dans son sélecteur
  (`CanAccessPlayer` accorde l'accès sur le rôle) ; par la règle « visibles moins
  possédés en propre », son compteur inclut donc des joueurs hors de son cercle
  social. Conforme à la règle gravée (« ce que je vois dans mon sélecteur, moins
  les miens ») et sans fuite d'identité (le compte est un entier), mais c'est un
  cas à trancher si le produit veut un compte « social » strict pour les admins.
