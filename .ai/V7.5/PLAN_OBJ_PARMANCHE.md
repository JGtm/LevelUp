# PLAN — Objectifs PAR MANCHE + nettoyage porteur (lot wt/obj-parmanche)

> Protocole COMMITE AVANT tout refactor (contrat plan-execution). Base f2d2a87a8.
> Perimetre FERME : correctness du calque Objectives multi-manche + 3 P2 de revue porteur.

## Constat d'architecture (verifie sur pieces)

- `replaybuild/matchfacts.go:identifiedEvents` resout l'identite slot->xuid par
  `objectiveevents.SlotIdentityFrom(recs, lines)` (pont par TOTAUX, exige les lignes de
  match). En MULTI-MANCHE le slot d'entite est reattribue -> attribution fausse apres la
  bascule (meme defaut que couronne/drapeau, deja corriges).
- `ResolveRoundIdentity(recs, deaths)` (pont par MANCHE) exige le FIL DES MORTS
  (`[]DeathInstant`). `port.MatchFacts` ne le porte PAS ; il n'existe que dans
  `BuildFromFilm` (`opt.Deaths`). Donc la resolution DOIT descendre dans le paquet
  `analysis/replay`, EXACTEMENT comme `vip_crown.go` / `build_objectives_live.go` /
  `skull_carries.go` (qui font `ResolveRoundIdentity(in.Records, deathInstantsOf(opt.Deaths))`).
- `doc.Objectives` (actions posees) est aussi consomme par le calque des zones
  (`build_zones.go`) : il reste produit par `attachObjectiveActions`, forme inchangee.
- Forme du document (`ObjectiveAction` t/xuid/stat/timeMs) INCHANGEE -> AUCUN bump de schema.

## Etape 1 — MIGRATION Objectives par manche (touche du code LIVRE)

- [ ] 1a. `objectiveevents` : ajouter `IdentifyNamedEventsByRound(evs, RoundIdentity)`
  (miroir de `IdentifyNamedEvents` mais `.At(slot, timeMS)` par manche) ; factoriser le
  tri commun. Test unitaire : mono-manche == pont plat ; multi-manche attribue la manche 1
  au bon joueur (reutilise `twoRoundReassignedFixture`), et DIFFERE du pont plat.
- [ ] 1b. `replay` : `Options.Objectives` passe de `[]IdentifiedEvent` a un `ObjectiveInput`
  (Records + ObjType). `attachObjectiveActions` resout par manche via `opt.Deaths` — ne
  paie le pont que s'il y a des evenements nommes (garde cout/RAM Slayer).
- [ ] 1c. `replaybuild/matchfacts.go` : `identifiedEvents` -> `objectiveInput(recs, variant)`
  (Records + `ObjectiveTypeOf(variant)`), plus de dependance aux lignes.
- [ ] 1d. Tests : reecrire `matchfacts_test.go` (objectiveInput) ; adapter le cablage.
- [ ] 1e. REGRESSION mono-manche : golden/temoins objectifs inchanges. Prouver la
  neutralite mono-manche par construction (RoundIdentity.At une manche = pont plat) et par
  les tests CI. Golden film reel = corpus-gated (env) : statuer honnetement.

## Etape 2 — P2 #1 : `skullCarrySecondsByXUID` test-only

- [ ] Deplacer `skull_carries.go:skullCarrySecondsByXUID` (aucun appelant de prod ; seul
  `skull_carries_witness_test.go` l'appelle) vers le fichier de test. Zero changement de
  comportement.

## Etape 3 — P2 #2 : doc `SkullTicksComponent`

- [ ] `objectiveevents/score.go:~79` : la reference `[incrementInstants]` pointe un
  identifiant qui n'existe QUE dans `cmd/oddball-terrain`. Corriger vers la vraie fonction
  de dedup portee en prod (`replay.skullTickInstants`), en prose (lien cross-package
  vers un non-exporte ne resout pas). Doc seulement.

## Etape 4 — P2 #3 : test synthetique porteur (si peu couteux)

- [ ] `TestSkullCarrierWitness` skip en CI (pas de corpus). SI peu couteux, ajouter un test
  SYNTHETIQUE en memoire exercant `skullCarrySecondsByXUID`/le chemin porteur et gate en CI.
  Sinon, laisser tel quel et le statuer au CR (pas d'action = valide).

## Gates (chaque etape)

`go build ./...`, `go vet ./...`, `go test` des paquets touches (replaybuild, replay,
objectiveevents) + contracttest (le contrat NE bouge PAS). Pas de web (contrat stable).

## Decouvertes

(a remplir — non traitees dans ce lot)
