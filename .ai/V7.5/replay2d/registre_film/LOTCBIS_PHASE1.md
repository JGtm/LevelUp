# Lot C-bis — phase 1 : le port de `ti=13`, et la mesure d'etat

> Perimetre : CB.1.1 (port) et CB.1.2 (mesure) + Gate 1 du plan `PLAN_EXPLOITATION_REGISTRE_FILM.md`.
> Grammaire lue en phase 0 : `LOTCBIS_PHASE0.md`. Base : `38f78a46a`, branche `wt/zones-ti13`.
> Mesures du 2026-08-18. Gates : `LOTCBIS_gates.log`. Sorties : `lotC/<short8>_ti13_*.tsv`.

## 1. CB.1.1 — le port

| ti | i | composant | statut table | deser_addr | code_source |
|---|---|---|---|---|---|
| 13 | 1 | `managed-object-property-component` | `porte` | `FUN_140ce5554` | `components_managed_property.go:189` |
| 13 | 2..33 | `managed-object-player-masked-property-component` (32 lignes) | `porte` | `FUN_140ce593c` | `components_managed_property.go:208` |

`traverse.go` ne grossit que de six lignes (deux `case` de routage, 867-872) ; le corps est dans le
fichier neuf. Les 33 lignes de table ECS sont editees DANS LE MEME COMMIT (regle G1), et le
garde-rail `TestG1TableSuitLeCode` passe dans les deux sens.

**Statut `porte` et non `partiel`, et c'est motive.** La largeur depend de la donnee (4 a 36 bits
selon le tag), mais le branchement porte sur une valeur LUE DANS LE FLUX, les seize branches sont
integralement consommees, et le cas rend toujours `true` : aucun desync n'est possible. Meme
raisonnement qu'au lot C phase 1b pour les `rtpc`.

**Un seul endroit ecrit la table de largeurs.** `managedPropertyPayloadBits` vit dans le code de
production, et le decodeur des vecteurs de test l'APPELLE au lieu d'en garder une copie — la copie
de la phase 0 a ete supprimee dans ce commit. Les vecteurs figes testent donc la table portee, pas
un double qui pourrait diverger.

**L'index de joueur n'est pas dans le deser, et c'est le bon partage.** `consumeByName` ne recoit
pas l'index du composant ; le jeu le lit dans le descripteur. L'appelant, lui, a le masque :
`ManagedPropertyPlayerIndex(i)` rend `i-2` (i2 porte le joueur 0). Meme partage des roles que pour
les quatre `rtpc` de ti=10.

### Vecteurs : 33 cas sur octets reels, verts du premier coup

| test | cas | ce qu'il contraint |
|---|---|---|
| `TestTi13VecteursFiges` | 33 | tag, largeur de charge utile, quantum, position de fin |
| `TestTi13VecteursPortDeser` | 33 | ce que le HOOK publie et combien de bits le deser de PRODUCTION consomme |
| `TestTi13HookConsommeLesMemesBitsSansHook` | 32 | poser un hook ne change pas la consommation, sur les 16 tags et les 2 modes |
| `TestTi13ConvertisseursFigent` | 12 | dequantification `[-100,+100]`, enumere ou 0 vaut -1, index de joueur |
| `TestTi13RampeTag3` | 3 | la rampe du tag 3 monte a pas constant |
| `TestTi13IdentifiantsPartagesEntreFilms` | 3 | les string-ids du tag 5 sont les memes sur deux films |

Aucun film n'est lu : les octets sont recopies, les tests tournent en CI. 31 des 33 vecteurs sont
CHAINES (leur largeur est confirmee par le flux lui-meme).

### Gate de portage : `DesyncAt` sur les 12 films, avant / apres

Instrument du lot 0 (`delta_walk_witness_test.go`, 12 premiers chunks). **Aucun film ne recule, LES
DOUZE progressent** — le lot C n'en avait fait progresser que sept.

| film | mode | records avant -> apres | traversee ABOUTIE avant -> apres | gain |
|---|---|---|---|---|
| `7344d24f` | Strongholds | 33 029 -> 33 109 | 25 021 -> **25 149** | **+128** |
| `8076f97f` | KOTH | 32 970 -> 33 012 | 24 889 -> **24 943** | **+54** |
| `696a9d7c` | Strongholds | 32 713 -> 32 737 | 24 653 -> **24 693** | **+40** |
| `64e8adfa` | CTF | 39 776 -> 39 806 | 31 935 -> **31 973** | **+38** |
| `24dbb67d` | Oddball | 39 634 -> 39 659 | 30 917 -> **30 954** | **+37** |
| `530820e5` | CTF | 35 542 -> 35 569 | 26 241 -> **26 273** | **+32** |
| `53ce4390` | CTF | 38 008 -> 38 030 | 28 586 -> **28 613** | **+27** |
| `01e1f945` | KOTH | 37 967 -> 37 986 | 29 140 -> **29 162** | **+22** |
| `000d5950` | Slayer | 38 862 -> 38 878 | 30 060 -> **30 080** | **+20** |
| `0a247154` | KOTH | 33 226 -> 33 239 | 24 468 -> **24 484** | **+16** |
| `606d9844` | KOTH | 34 512 -> 34 524 | 26 642 -> **26 657** | **+15** |
| `06dfe6d9` | — | 10 607 -> 10 613 | 8 494 -> **8 502** | **+8** |

La table figee des trois films de reference a ete mise a jour avec sa justification ecrite sur
place, comme son contrat l'exige, et la chronique du lot C a ete conservee a cote de la neuve.

**Le gain reste modeste, et il faut dire pourquoi** : une traversee n'aboutit que si TOUS les
composants annonces sont portes, et le trafic est domine par le bipede. **Ce qui est neuf, c'est
que le gain touche LES DOUZE films, y compris le temoin Slayer ou ti=13 est pourtant quasi muet
(5 records d'i1)** — signe que les records reparees ne sont pas seulement ceux de la bande ti=13,
mais aussi ceux qui la SUIVENT dans le paquet et que la marche sequentielle atteint desormais.

### Gates joues

`EXIT_VET=0` · `EXIT_TEST_FILMDEC=0` (filmdec, replay, objectiveevents) · `EXIT_BUILD_CGO=0`
(`go build ./...`, CGO actif) · `EXIT_LINT=0` (`golangci-lint --new-from-merge-base=origin/main
./...`, 0 issue) · `TestG1TableSuitLeCode` et `TestG3TableSuitLeDocument` verts.
