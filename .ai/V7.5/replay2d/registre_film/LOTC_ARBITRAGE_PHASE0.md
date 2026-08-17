# Lot C — arbitrage superviseur apres la phase 0 (2026-08-17, soir)

> Lu sur pieces : `LOTC_PHASE0.md`, `LOTC_gates.log`, `lotC/LOTC_table_C03.tsv` (385 lignes),
> commit `3c2502b2a` sur `wt/zones-film`.

## Ce que la phase 0 etablit (acquis, ne pas re-mesurer)

1. **ti=23 est ABSENT** en image-cle ET en delta sur 11 films et 5 modes (0 slot / 117 460
   records d'image-cle, 0 record / 376 080 paquets delta) : l'ecart n°2 du plan est confirme et
   etendu ; le deser `consumeSelectableZoneData` n'a plus de condition de branchement — sa
   condition de RETRAIT est remplie (a traiter au lot 0 ou a la cloture de ce lot : suppression,
   ligne de table `deser_non_cable` -> retiree, `//nolint` avec elle).
2. **ti=10 et ti=12 sont la machinerie du mode** : presents sur les 10 films a objectif (30-157
   slots ti=10, 7-58 slots ti=12), ABSENTS du temoin Slayer ; leurs numeros de slot ne sont PAS
   stables d'un film a l'autre (10/81, 5/16 en commun) — aucune phase ne cablera un slot.
3. **`ti=12 i14 radial-progress` est LE canal** : 74,7 % et 93,1 % de tous les records ti=12 des
   deux Strongholds, 140,5x et 868,5x le plancher de bruit ; les deux autres progressions (i13,
   i15) sont muettes, `timers`/`formatted-text` faibles. `ti=10 i1 boundary-color` : 10 347
   annonces sur 10 films (6,5x). `ti=13 managed-object-property` (i1 88,9x ; i13/i21
   `player-masked-property` 37x, slots STABLES 26/26 entre les Strongholds) : piste au moins aussi
   bonne que ti=10, non inventoriee par le plan.
4. `ti=47` : le splash DYNAMIQUE (i1, porte) est anti-correle aux captures de zone (0,01x) et suit
   les evenements de drapeau en CTF (2-3x) ; `i2 personal-ai-data` (non porte) domine en zones
   (77-81 % des records, 1 214x le plancher) — c'est la voix de l'IA personnelle (annonces).
5. `ti=4 high-frequency` (1 slot, porte) : 30 000-50 000 annonces par film, TOUS modes, Slayer
   compris (`LOTC_table_C03.tsv` ligne 1) — un compteur haute frequence de simulation ; sonde F3
   a lire par le hook du lot 0 (candidat horloge/tick pour le lot B).
6. Cout : 112 s cumulees, pic 15-18 Mo — le risque D17 ne se materialise pas pour cet instrument.

## Verdict tenu, critere requalifie

Le GATE 0 est NON ATTEINT sur sa clause 2 (concentration ± 3 s autour des captures >= 3x : max
2,37x). Mais la clause 2 a ete ecrite pour un canal d'EVENEMENT ; `radial-progress` et les
`rtpc` sont des ETATS CONTINUS, reemis pendant tout le remplissage d'une zone, c'est-a-dire AVANT
l'instant de capture — une fenetre centree sur la capture ne peut pas les concentrer. Le critere
est montre non discriminant pour l'objet mesure (meme situation que le gate 0 de l'item 6, arbitre
le 17/08). L'enonce de repli du plan (« les zones ne parlent pas en delta par ces archetypes »)
serait FAUX : elles parlent, en etat, massivement.

## Decision : phase 1 ouverte, en deux temps, avec un gate d'ETAT ecrit avant

**Phase 1a — RE bornee, LECTURE SEULE, sans code Go** (tout de suite, dans `wt/zones-film`) :
- C.1a.0 Instrument : la largeur du slot dans l'ancrage prend `FrameConfig.IDLowBits` du film
  (11 ou 14, `frame_records.go:39-44`) au lieu de 13 fixes ; re-compter sur 3 films ; part
  « hors grammaire » attendue en baisse (chiffre avant/apres). Fichiers `_test.go` seulement.
- C.1a.1 Pour, dans l'ordre : `ti=12 i14 managed-navpoint-radial-progress`, `ti=10 i1
  managed-object-boundary-color-component`, `ti=13 i1 managed-object-property-component`,
  `ti=13 i13/i21 managed-object-player-masked-property-component` (et, si le temps reste,
  `ti=10 i26/i27 rtpc`) : retrouver le deserialiseur (recette R7-d : chaine du nom dans `.rdata`
  -> descripteur de composant -> vtable ; `getName()` = +0x18 ; le lecteur est la methode sœur —
  `PLAN_R7D_ECRIVAIN_VTABLE.md` et `killweapon/WALK_PORT_NOTES.md`), le decompiler, ECRIRE la
  grammaire bit a bit (largeurs, portes, dependance au niveau du registre ou a un etat runtime),
  et sortir 2-3 vecteurs de test depuis des octets de film (position d'un record connu par
  l'ancrage + masque). Ghidra : instance de l'utilisateur, programme `HaloInfinite.exe`, outils
  MCP `mcp__ghidra__*` en LECTURE SEULE (`decompile_function`, `disassemble_function`,
  `search_strings`/`list_strings`, `get_xrefs_to`, `read_memory`, `list_data_items`) — AUCUN
  rename, AUCUN commentaire, AUCUN script, AUCUNE analyse (c'est le programme de l'utilisateur).
  Borne : 1 jour executeur par composant ; au-dela, STOP sur ce composant, ecrit, suivant.
- C.1a.2 Journal `LOTC_PHASE1A.md` : par composant, adresse, decompile resume, grammaire, vecteurs,
  confiance ; ce qui n'a pas ete trouve et pourquoi.

**Phase 1b — port Go** (APRES fusion du lot 0 dans `wt/registre-film` et rebase de `wt/zones-film`
— les `case` s'ajoutent alors dans un nouveau fichier `components_managed_object.go` avec hooks,
table ECS editee dans le meme commit, G1 vert) puis **C.0.2** (32 drapeaux de ti=10 i0 par le hook
du lot 0) et la MESURE d'etat :
- G-C1 (ecrit avant, jamais rebaisse) : `radial-progress` : rampes monotones 0 -> max puis remise a
  zero ; sur les 2 Strongholds, chaque `ZoneCapture` (statborg, ms) est precedee, sur UN slot ti=12,
  d'une rampe qui atteint son max dans [t-2 s ; t+2 s] pour >= 80 % des captures ; temoin : memes
  captures decalees de +20 s -> <= 20 % ; KOTH : une seule rampe active a la fois sur >= 90 % du
  temps ou une rampe existe. `boundary-color` : <= 8 valeurs distinctes ; un changement dans
  [t-2 s ; t+2 s] de chaque capture sur UN slot ti=10 pour >= 80 % ; l'appariement slot -> zone du
  catalogue (par la position du capteur, `AttributeZones`, `Attributed` seulement) est coherent sur
  tout le match a >= 90 %. `player-masked-property` : enumerable (<= 16 valeurs), transitions datees
  vs captures (taux ecrit).
- Si G-C1 tient sur >= 1 canal : phase 2 (publication `zoneStates`) ; sinon negatif ecrit par
  canal, lot clos `[!]`.

`DesyncAt` sur les 11 films ne doit pas s'aggraver d'un portage (<= 1 %). Le plan item 4 garde sa
phase 2 tant que G-C1 n'est pas tenu.

## Decouvertes recues, a porter par le superviseur

- plancher de bruit de `matchWorldObjectRecord` non chiffre alors que les scanners de PRODUCTION
  ti=37/41/42 l'emploient (`equipment_state.go:246` exige `Idx[0]==0`, pas les autres) — ligne de
  registre a ouvrir (hors perimetre de ce lot) ; `IDLowBits` 11/14 vs 13 cable ; `ti=13` non
  inventorie ; `ti=47 i2` sans ligne ; KOTH/Oddball sans oracle nomme (`ScoreCurve` en repli) ;
  `000d5950` = 27 chunks utiles (§1.2 du plan a corriger) ; `ecs_table.tsv` exacte sur les 7
  archetypes recenses.
