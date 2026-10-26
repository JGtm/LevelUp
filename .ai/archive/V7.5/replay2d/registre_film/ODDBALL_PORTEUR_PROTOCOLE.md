# PROTOCOLE — PORTEUR DU CRANE D'ODDBALL + IDENTITE PAR MANCHE

> Ecrit et COMMITE AVANT tout refactor ou toute mesure (regle du lot). Il fixe les criteres
> d'acceptation ; le code qui suit les applique. Aucun seuil n'est deplace apres coup.

Deux parties : (A) une correction d'INFRA de l'identite slot->joueur qui touche du code LIVRE
(couronne VIP schema 22, drapeau CTF schema 15) ; (B) la publication du PORTEUR du crane
d'Oddball, au patron de la couronne VIP.

Acquis figes (ne pas re-mesurer) : `ODDBALL_VERITE_TERRAIN_d9781168.md` + `TERRAIN_*.log`.
- PORTEUR = tics de score de mode par joueur = **comp 0 cote A** (`skull_scoring_ticks`).
- PRISES = **comp 21 cote B** (`skull_grabs`).
- Gate oracle : porteur PRINCIPAL correct **7/7 films**. Gate terrain manche 1 : prises 9/9,
  porteurs 8/9 (seuil 8/9 tenu). Signal universel (identifie par l'oracle sur 7 films).
- Emplacements identifies par l'oracle films confondus, PAS ajustes a d9781168.

## PARTIE A — IDENTITE PAR MANCHE (infra, code LIVRE : neutralite obligatoire)

Constat mesure sur pieces : en multi-manche le SLOT d'entite est REATTRIBUE d'une manche a
l'autre (d9781168 : slot 22 = scuderiasven en manche 0 puis LadyJezz en manches 1-2). Or
`objectiveevents.SlotIdentityByDeaths` rend UN SEUL `map[slot]->xuid` pour tout le match, et il
est appele par `build_objectives_live.go:137` (drapeau) ET `vip_crown.go:221` (couronne). La
couronne VIP livree est donc FAUSSE en VIP multi-manche.

Livrables A :
1. `objectiveevents.SlotIdentityByRound(recs, deaths) map[int]map[int]string` : identite
   slot->xuid PAR MANCHE. Chaque manche est resolue par LE MEME algorithme prudent que le pont
   plat (`slotIdentityFromDeaths` : marge + rejet des xuid contestes), sur les progressions du
   compteur de morts (`comp 2 B`) RESTREINTES a la manche, appariees au fil des morts COMPLET.
2. Un resolveur `RoundIdentity` (fenetres de manche en ms du match) expose aux calques :
   `At(slot, timeMS)` (evenements VIP/drapeau) et `AtRound(round, slot)` (porteur, qui connait
   deja sa manche). Migration de `build_objectives_live.go:137` et `vip_crown.go:221`.

CRITERE DE NEUTRALITE (NON NEGOCIABLE) : un film a **au plus une manche reelle** rend
EXACTEMENT le resultat d'avant. Garanti par CONSTRUCTION : pour <= 1 manche, la resolution
retourne `{manche: SlotIdentityByDeaths(recs, deaths)}` — LA MEME fonction, indexee de la meme
facon (`At` ignore le temps quand il n'y a qu'une manche). Preuve :
- unites `objectiveevents` + `replay` + `contracttest` verts ;
- tests golden/temoins VIP & CTF mono-manche INCHANGES (le golden d'assemblage regenere ne
  bouge QUE sur la montee de schema et le calque neuf vide, jamais sur les comptes de drapeau) ;
- un test NOUVEAU prouve la DIFFERENCE sur un cas multi-manche synthetique (slot reattribue :
  le pont plat attribue au mauvais joueur apres la bascule, le pont par manche corrige).

## PARTIE B — PUBLICATION DU PORTEUR (schema 23, au patron `vip_crown`)

Reconstruction : porteur = trains de tics (`comp 0 A`) par manche, chaque train (increments,
trou de fermeture `tickGapMS`) attribue via `RoundIdentity.AtRound(manche, slot)`. Fenetres PAR
MANCHE (reset a la bascule ; un passage de manche NE FERME PAS un portage par une fausse prise).
Prises (`comp 21 B`) publiees en couverture (denominateur), pas comme bornes de portage.

Gate B = celui de l'outil, DEJA figé : porteur PRINCIPAL correct 7/7 (seuil >= 3/5). La
reconstruction Go reproduit le canal des tics de `cmd/oddball-terrain` ; aucun seuil nouveau.

Publication :
- `replay.SkullCarry` (xuid, t0, t1 frames, closed) + `SkullCarriesCoverage` ; champ document
  `skullCarries` ; garde de mode chez `replaybuild` (`isSkullVariant`, jamais devinee dans le
  film — comme la couronne et la colline). Le crane LIBRE (couche position, schema 21) reste
  publie : le porteur est la couche VIVANTE par-dessus.
- Web : `skullCarrierLayer.ts` + `useReplaySkullCarrier.ts` (patron `vipCrownLayer` /
  `useReplayVipCrown`), glyphe crane sur le porteur courant, toggle, i18n FR+EN.
- Triplet de schema : `replay.SchemaVersion` 22->23, contrat OpenAPI regenere,
  `EXPECTED_REPLAY_SCHEMA_VERSION` web 22->23, chronique du document + contracttest a jour.
- MULTI-MANCHE au rendu : les vies de portage sont datees sur l'horloge continue et s'affichent
  a travers les bascules sans se melanger (l'identite change de manche, la frame reste continue).

Gate B / livraison : `go test` paquets touches + `contracttest` ; `tsc -b` (cache purge) ;
`vitest match-replay` ; lint web ; parite de schema. Re-cuisson des TEMOINS (>= d9781168 + 1
autre film Oddball) avec verification de CONTENU (porteur principal nomme, comptes coherents).

## RISQUE & REVUE

Ce lot touche du CODE LIVRE (couronne VIP schema 22, drapeau CTF schema 15). Fichiers a risque :
`internal/analysis/replay/vip_crown.go`, `build_objectives_live.go`, `flag_carries.go`,
`internal/analysis/objectiveevents/slotidentity_deaths.go` (+ le nouveau `slotidentity_rounds.go`),
`internal/replaybuild/matchfacts.go`. Une ADVERSARIAL-REVIEW est REQUISE avant merge.

STOP si la regression VIP/CTF n'est pas parfaitement neutre en mono-manche : signaler, ne pas
livrer.
