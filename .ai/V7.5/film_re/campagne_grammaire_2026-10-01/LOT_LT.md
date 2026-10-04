# Lot LT — règle de tête de liste : jamais une tête NEW dont le masque contredit l'écrivain (2026-10-04)

> Lot de marche NEUF de la vague 2 du plan `.ai/PLAN_CAMPAGNE_GRAMMAIRE_2026-10-01.md` (§5,
> « Découvertes de la vague 1 », règle de tête de liste ; D-L8-1, D-L8-8, D-L3a-1, D-L2-5, D-L2-12),
> GO de la vague 2 de l'utilisateur du 2026-10-04, sous le contrat `plan-execution`, dans le cadre du
> critère FERME du 2026-10-02 : **corrections générales lues dans le jeu, aucun réglage par film, par
> carte ni par version non vérifiable dans le jeu**.
>
> Worktree `LevelUp-wt-cg2-lt`, branche `feat/cg2-lt` créée depuis `feat/cg2-ls` = `80d2acd20`
> (LU + LS au-dessus de `2393d7db7`). Films lus en place (`LevelUp/data/cache/film_chunks`, lecture
> seule), aucune cuisson du parc. Mesures et scripts : `lt_tsv/` (ce dossier) et
> `scratchpad/v2-LT/` (session `f46f71fc`). GOCACHE `go-build-cg2-lt`, une commande `go` à la fois.
> Convention : **mesuré** = compté par un outil sur les films ; **établi** = lu dans le jeu (Ghidra,
> `HaloInfinite.exe` HI_1_13_0, HTTP direct 127.0.0.1:8089, lecture seule) ; **supposé** = hypothèse
> écrite.

## 0. Statut

| Item | Statut | En une ligne |
|---|---|---|
| LT.1 la preuve par chaîne (`pasDEssai` / `debutParChaine`, localisateur de tête) refuse un NEW ou un DELTA dont le masque contredit `FUN_142e2da44` | [x] | +44 paquets sains, +1 023 records utiles sains, **0 sain perdu sur 20 films** ; les 5 sains perdus de la vague 1 retrouvés (§3) |
| LT.2 second rang de `debutParFermeture` : règle générale qui refuse les têtes contredites sans perdre les 23 sains de `e5adf7b2` | [!] | **impossible sous le critère** : les têtes utiles du second rang sont des NEW `ti=41` des formats anciens dont le corps est mal lu par la largeur MPP non lue dans le jeu (LM, mis de côté le 2026-10-02) ; instruit §4 |
| LT.3 retrait (ou forte réduction) du repli `repli_debut_de_liste_ferme_au_bit` | [!] | non retiré : son retrait seul fait baisser `e5adf7b2` (−21 sains) ; compte inchangé par LT.1 (9 099 → 9 100 carte, 8 069 → 8 068 cuisson) ; retrait sans baisse MESURÉ sous la largeur MPP 8/3 (§4.3) |
| LT.4 L2 rejoué par-dessus LT (surcouche) | [x] mesuré, **L2 ne tient pas le gate 2** | `1c4c63c2` −9 sains / −422 utiles sains, `d9781168` −2 / −33 (§5) ; non intégré |
| Révision | [x] | `grammar-2026-10-03.4` ; `killsource`, `source`, `objectives`, `profile` constantes (§6) |
| Gate 1 (tests, vet, archlint, révision) | [x] | §7 |
| Gate 2 (carte v2, 20 films) | [x] | aucun film en baisse, 0 sain perdu (§3) |
| Gate 3 (killsource, 19 témoins) | [x] | JSON identique à l'octet 19 / 19 (§7) |
| Gate 5 (recopie du pilotage) | [x] | `TestFrameClosureDetailleeRendLaCarteDeFrameClosure` : `ks_000d5950` 446 / 27, `ks_e5adf7b2` 52 / 2 ; la sonde `lt_tete_research_test.go` RECOPIE le localisateur (`ltLocaliser`), la chaîne (`ltChaine`, reprise de `debutParChaine` et `chaineJusqua`), le pas (`ltPas`, copie de `pasDEssai`) et le second rang (`ltFermeture`) ; ces recopies sont contrôlées par la variante `prod`, qui redonne la carte de la base sur `e5adf7b2` (4 147 / 80 067), la variante `chaine` celle du lot (4 149 / 80 107) — rejoué par le contrôle indépendant (phrase « aucune recopie » du lot corrigée) |
| `replay-equiv` (recette L0) | [x] | 4 étapes sur 61 divergent, toutes expliquées (§7) |
| Mutations | [x] | 5 / 5 ROUGES (§7) |
| D-L8-8 (arrêter la traversée de la marche sur un masque non écrit) | [!] | non traité : règle de la marche principale, hors de la règle de tête ; consigné (D-LT-6) |

Verdict : **[x] retenu pour LT.1** (règle de la chaîne de tête) ; LT.2 et LT.3 sont `[!]` avec
instruction, et le repli reste nommé et compté tel quel.

## 1. Ce qui est lu dans le jeu (établi)

- `FUN_142e2da44` (écrivain du masque d'un record NEW ou DELTA, relu dans cette session) : il ne
  parcourt que `i < *(desc + 0x4320)` (le nombre de composants du descripteur) ; s'il compte plus de
  sept composants présents, il écrit `R(1) = 1` puis le masque sur 64 bits (`FUN_1406d6498(..., 0x40)`,
  retour `0x41`) ; sinon `R(1) = 0`, le compte sur 3 bits, puis les index sur 6 bits dans l'ordre
  croissant de `i`. Donc : aucun bit au-delà du dernier composant de l'archétype, jamais un masque
  dense de sept composants ou moins, jamais un masque épars à index non croissants. Ce sont les trois
  règles que `lireMasque` et `traverseComponentLoopFrom` marquent déjà dans `EntityTrace.MasqueNonEcrit`
  (lot L0).
- `FUN_1408efb58` (état par défaut du projectile `ti=41`, relu, `lt_tsv/` → `scratchpad/v2-LT/ghidra/`)
  lit d'abord `FUN_1406cf008`, soit R(1), puis R(8) si ce bit vaut 1 (la version), et ensuite
  seulement le bloc MPP `FUN_14080cfe8` (corrigé au contrôle indépendant, relu dans Ghidra ; c'est
  aussi ce que porte `consumeDefaultStateTI41`) : la largeur de ce bloc décale tout le corps qui suit
  dans le record NEW.
- L'ordre des NEW (`FUN_142f2e174` : slots croissants, un genre par entité ; `FUN_14076b9c8` : NEW,
  puis DELTA, puis DEL ; `FUN_142f2f8f0` : état 3 à l'écriture du NEW) : relu sur les décompilations
  de LS (`scratchpad/v2-LS/g_142f2e174.c`, `g_14076b9c8.c`) ; il est déjà au juge de L0
  (`InvariantOrdreVueB`) et ne départage AUCUNE des têtes du second rang (§4.2 : la tête utile est le
  seul candidat qui ferme au bit près dans son paquet).

Ce qui n'est PAS lu dans le jeu : la largeur du bloc MPP des formats 20 à 25 (aucun exécutable de ces
builds ; `profile.MPPPourFormat` la laisse indéterminée, R_VEH.md §1.4 l'a mesurée à 8/3). Elle sert
ici UNIQUEMENT de mesure de recherche (§4.3), jamais de correctif.

## 2. Ce qui change (production)

`grammar/debut_de_liste.go`, `pasDEssai` (deux lignes, plus le commentaire de contrat) :

```go
case recNew:
	...
	return tr.EndBit, tr.DesyncAt == -1 && tr.MasqueNonEcrit == InvariantAucun
case recDelta:
	rec, fin, ok := TryDeltaAt(pay, pos, w, essai)
	return fin, ok && rec.Trace.MasqueNonEcrit == InvariantAucun
```

Un pas de la chaîne de tête refuse un record dont le masque contredit `FUN_142e2da44`, quelle que soit
la règle contredite. C'est la réparation de L2 (`99046f75a`, retirée avec L2) et le correctif que L8
avait mesuré en surcouche (D-L8-1). Le premier rang de `debutParFermetureRangee` exigeait déjà une
lecture fermée sans règle contredite (L0) ; le second rang est inchangé (§4). Signatures de
`localiserLaListe` et `debutParFermetureRangee` inchangées ; la phrase de l'en-tête qui énumère les
conditions de la chaîne gagne « aucun masque que l'écrivain n'écrit pas ».

Fichiers de la représentation intermédiaire touchés : `debut_de_liste.go` (étape 1, fusionnée) — corps
de `pasDEssai` et commentaires seulement. Aucun fichier de la liste de l'étape 2 (`feat/ri-etape2`)
n'est touché.

Test neuf `debut_de_liste_masque_test.go` (repris de L2, `b12eb7692`, composants d'un bit
`projectile-tether-state` au lieu du composant `device-*` retiré avec L2) : pour chaque règle, sur le
pas NEW et sur le pas DELTA, un masque contredit et son TÉMOIN (même paquet, mêmes composants lus,
masque que l'écrivain écrit) ; le témoin prouve que la chaîne tombe au bit près et que seul le masque
refuse.

Sonde `lt_tete_research_test.go` (tag `research` ligne 1) : compteur du repli par film, marche sous
variantes de tête, recensement des NEW par archétype et règle de masque, détail de paquets, et
découpage MPP imposé comme MESURE.

## 3. Gate 2 — carte v2, 20 films (`-denominateur-fixe`, base `80d2acd20` contre lot)

Paquets sains (fermés au sens de L0) et records utiles sains ; `lt_tsv/gate2_lt.tsv`,
`lt_tsv/paquets_changes_lt.tsv`.

| Film | Sains avant | Sains après | Net | Sains perdus | Utiles sains avant | après | Net |
|---|---|---|---|---|---|---|---|
| `0797ce72` | 19156 | 19156 | +0 | 0 | 159929 | 159929 | +0 |
| `084a804d` | 4832 | 4837 | +5 | 0 | 89616 | 89755 | +139 |
| `111fa685` | 4026 | 4026 | +0 | 0 | 42896 | 42896 | +0 |
| `11de8353` | 5629 | 5631 | +2 | 0 | 65621 | 65658 | +37 |
| `1c4c63c2` | 13887 | 13888 | +1 | 0 | 176877 | 176899 | +22 |
| `396cfc92` | 23131 | 23131 | +0 | 0 | 169502 | 169502 | +0 |
| `4f77afc1` | 24082 | 24104 | +22 | 0 | 607244 | 607913 | +669 |
| `50247b26` | 139 | 139 | +0 | 0 | 274 | 274 | +0 |
| `51ebbc0f` | 25471 | 25472 | +1 | 0 | 185697 | 185712 | +15 |
| `60ae07c4` | 14594 | 14596 | +2 | 0 | 88031 | 88047 | +16 |
| `a349fea8` | 424 | 424 | +0 | 0 | 3654 | 3654 | +0 |
| `a521164d` | 692 | 692 | +0 | 0 | 78 | 78 | +0 |
| `bcb6d393` | 5879 | 5879 | +0 | 0 | 35492 | 35492 | +0 |
| `bf15f7ab` | 28603 | 28603 | +0 | 0 | 216104 | 216104 | +0 |
| `bfecd02b` | 27666 | 27668 | +2 | 0 | 239045 | 239064 | +19 |
| `c75f33b8` | 26148 | 26148 | +0 | 0 | 173096 | 173096 | +0 |
| `d9781168` | 36995 | 36995 | +0 | 0 | 278340 | 278340 | +0 |
| `e5adf7b2` | 4147 | 4149 | +2 | 0 | 80067 | 80107 | +40 |
| `f75e7053` | 23673 | 23673 | +0 | 0 | 162137 | 162137 | +0 |
| `fb1a1a72` | 43450 | 43457 | +7 | 0 | 325749 | 325815 | +66 |
| **corpus** | **332624** | **332668** | **+44** | **0** | **3099449** | **3100472** | **+1023** |

- **Aucun film en baisse ; 0 sain perdu** (ni « devenu contredit », ni « devenu non fermé ») ; 0 paquet
  fermé au bit perdu.
- **Juge sur les GAGNÉS** : les 44 paquets gagnés étaient tous fermés au bit près et contredits par
  une règle de masque (43 « masque au-delà de l'archétype », 1 « masque épars non croissant ») : la
  chaîne prenait pour tête un NEW que l'écrivain ne peut pas écrire. Ils ferment maintenant depuis le
  début de la signature, aucune règle contredite.
- **Les 5 sains perdus en brut de la vague 1 sont retrouvés** : `fb1a1a72` 7:92 (D-L8-1),
  `1c4c63c2` 11:1620, `4f77afc1` 25:874, 37:1188, 59:682 (D-L3a-1), tous « masque au-delà de
  l'archétype » → sains.
- **Factices** : 1 paquet gagné au bit près, contredit (`1c4c63c2` 38:650, sortie par rejet) — part
  factice des gains au bit 1 / 1 ; il n'est pas compté sain.
- Autres changements (non sains des deux côtés) : `a349fea8` 10 paquets dont la première règle
  change (9 au chunk 10 « aucune » → « sortie par rejet », 27:530 l'inverse), non instruits un par un
  (D-LT-5).
- Indicateur D1 (fixe consolidé 7 758 290) : 3 099 449 → 3 100 472 records utiles sains, 39,95 % →
  39,96 % (arithmétique sur le TSV).

## 4. Le second rang de `debutParFermeture` : pourquoi aucune règle lue dans le jeu ne le retire (LT.2, LT.3)

### 4.1 Mesure des variantes (sonde, marche de la carte, 20 films ; `lt_tsv/variante_prod_vers_*.tsv`)

La sonde rejoue la marche de la carte avec des localisateurs de tête de recherche. Contrôle : la
variante `prod` redonne la carte de base à l'unité (paquets sains et utiles sains par film), la
variante `chaine` redonne la carte du lot.

| Variante (contre la base) | Sains corpus | Utiles sains | Sains perdus | Films en baisse |
|---|---|---|---|---|
| `chaine` (= LT.1, production) | +44 | +1 023 | 0 | aucun |
| `chaine+masque` (2e rang refusé si la 1re règle contredite est de masque) | +1 213 | +24 498 | 29 | `e5adf7b2` −21 / −707 |
| `chaine+tete` (2e rang refusé si le NEW de tête contredit le masque) | +1 420 | +28 161 | 27 | `e5adf7b2` −21 / −707 |
| `chaine+aucun` (pas de second rang) | +1 566 | +30 902 | 29 | `e5adf7b2` −21 / −707 |

Le second rang est donc, à cette base, une perte nette partout SAUF sur `e5adf7b2`, où ses têtes
portent 23 paquets sains (pertes brutes des trois variantes : les 23 de `e5adf7b2`, plus 4 à 6 sur
`1c4c63c2` et 0 ou 1 sur `4f77afc1`, films en hausse nette).

### 4.2 Les 23 sains de `e5adf7b2` (mesuré, `lt_tsv/e5adf7b2_*.tsv`, `perdus_prod_vers_chaine+aucun.txt`)

- Les 23 paquets perdus sont 10:960 à 10:1024. Sans second rang, chacun s'arrête sur un DELTA du slot
  1076 ou 1096 (« vue B : sortie par rejet », slot non lié) ; avec lui, ces DELTA se lisent sous
  `ti=41` (liaison « NEW lu », masques propres) et les paquets ferment sans règle contredite.
- Les liaisons viennent de deux listes du second rang : 10:20 (tête NEW 1076 `ti=41`, au bit 3748) et
  10:848 (tête NEW 1096 `ti=41`, au bit 3383). Dans les deux, le NEW de tête contredit l'écrivain
  (masque au-delà des 22 composants de `ti=41`) ; c'est le seul candidat d'où le paquet ferme au bit
  près, et les records qui suivent se lisent proprement. C'est le cas de D-L0-2 (« en-tête juste,
  corps mal lu »).
- Recensement des NEW `ti=41` lus par la marche (`neufs_ti41_mpp_defaut.tsv`) : sur les films
  HI_1_12_0 / HI_1_13_0, ils sont presque tous à masque écrivable et dans des paquets fermés
  (`4f77afc1` 1 620 fermés contre 9 contredits, `fb1a1a72` 569 / 2, `bfecd02b` 210 / 4) ; sur les
  formats anciens, presque tous contredits et presque jamais fermés (`e5adf7b2` 4 NEW dans des paquets
  fermés, 170 contredits ; `1c4c63c2` 14 / 221 ; `111fa685` 4 / 265 ; `084a804d` 1 / 155).

**Cause, établie pour la grammaire et mesurée pour la largeur** : le corps du NEW `ti=41`
(`FUN_1408efb58`) porte, après R(1)[+R(8)], le bloc MPP (`FUN_14080cfe8`), dont la largeur des formats 20 à 25
n'est pas lue dans le jeu (`MPPPourFormat` : indéterminée, le décodeur garde 9/5). Sous le découpage
8/3 mesuré par R-VEH (imposé par la sonde, `LT_MPP=8/3`, MESURE de recherche seulement), les NEW
`ti=41` de `e5adf7b2` se lisent : **4 → 1 368 dans des paquets fermés**, contredits 170 → 15
(`neufs_ti41_mpp83.tsv` ; `084a804d` 1 → 2 154, `1c4c63c2` 14 → 790).

**Pourquoi aucune règle de tête ne les garde** : un NEW dont le masque contredit l'écrivain n'est pas
un record écrit là par le jeu tel qu'il est lu ; le garder comme tête suppose que son en-tête est juste
et son corps mal lu, ce qu'aucune fonction du jeu ne permet de décider ; et la lecture qui le rendrait
juste (la largeur MPP des formats anciens) n'est pas vérifiable dans le jeu — c'est LM, mis de côté par
l'utilisateur le 2026-10-02. L'ordre des NEW (`FUN_142f2e174`, `FUN_14076b9c8`) ne départage rien : à
10:20 et 10:848, le candidat du second rang est le seul qui ferme au bit près. Restreindre le second
rang par une condition « tête d'un archétype à bloc MPP sur un format à largeur MPP indéterminée »
serait une condition par version, hors du critère (proposée en décision, D-LT-2).

### 4.3 Sous la largeur MPP mesurée 8/3 (recherche, 20 films ; `variante_mpp83_chaine_vers_*.tsv`)

| Variante (contre `chaine`, tous deux sous 8/3) | Sains corpus | Utiles sains | Sains perdus | Films en baisse |
|---|---|---|---|---|
| `chaine+tete` | +7 819 | +183 523 | 6 | aucun |
| `chaine+aucun` (repli retiré) | **+9 731** | **+231 532** | 6 (`084a804d` 1, `1c4c63c2` 4, `4f77afc1` 1) | **aucun** |

Sous 8/3, `e5adf7b2` ne perd plus rien (12 267 → 12 267) : 10:848 se localise au PREMIER rang (fermé,
sain) et 10:20 n'est plus localisé sans perte. **Le retrait du repli `repli_debut_de_liste_ferme_au_bit`
tient le gate 2 dès que la largeur MPP des formats anciens est lue** : il dépend d'une décision sur LM,
pas d'une règle de tête.

### 4.4 Compteur du repli, par film (avant / après LT.1)

Carte (marche de la cuisson `ScanMarcheDesTrames` sur les 20 films de la carte, sonde) et cuisson
(`replay-equiv`, 20 films d'équivalence, journal des replis) ; `repli_par_film_carte.tsv`,
`repli_par_film_cuisson.tsv`. « Listes par chaîne » = listes ouvertes par un NEW de tête prouvé par la
chaîne (`EventPacketsNewRecordStart`).

| Film | Repli (carte) avant | après | Listes par chaîne avant | après | Repli (cuisson) avant → après |
|---|---|---|---|---|---|
| `0797ce72` | 20 | 20 | 759 | 759 | - |
| `084a804d` | 472 | 472 | 710 | 657 | 472 → 472 |
| `111fa685` | 141 | 141 | 196 | 193 | 141 → 141 |
| `11de8353` | 128 | 128 | 169 | 161 | 128 → 128 |
| `1c4c63c2` | 6425 | 6426 | 8728 | 8673 | 6427 → 6426 |
| `396cfc92` | 9 | 9 | 464 | 464 | - |
| `4f77afc1` | 1013 | 1013 | 3963 | 3931 | - |
| `50247b26` | 14 | 14 | 43 | 40 | 14 → 14 |
| `51ebbc0f` | 100 | 100 | 660 | 659 | - |
| `60ae07c4` | 108 | 108 | 151 | 144 | 104 → 104 |
| `a349fea8` | 44 | 44 | 110 | 93 | 44 → 44 |
| `a521164d` | 5 | 5 | 31 | 29 | 5 → 5 |
| `bcb6d393` | 5 | 5 | 274 | 271 | 5 → 5 |
| `bf15f7ab` | 10 | 10 | 549 | 549 | - |
| `bfecd02b` | 22 | 22 | 545 | 543 | - |
| `c75f33b8` | 93 | 93 | 483 | 483 | - |
| `d9781168` | 284 | 284 | 1164 | 1161 | 283 → 283 |
| `e5adf7b2` | 159 | 159 | 213 | 205 | 159 → 159 |
| `f75e7053` | 8 | 8 | 442 | 442 | - |
| `fb1a1a72` | 39 | 39 | 1032 | 1027 | 39 → 39 |
| **total** | **9 099** | **9 100** | | | **8 069 → 8 068** (20 films d'équivalence, dont 7 hors carte : `000d5950` 2, `01e1f945` 9, `51101d1d` 1, `53ce4390` 19, `64e8adfa` 60, `696a9d7c` 17, `7344d24f` 27, `9f57c612` 113, inchangés) |

LT.1 ne change pas le compte du repli (±1 sur `1c4c63c2`) ; il retire 202 listes ouvertes par une
chaîne de tête contredite. Retirer le repli (variante `chaine+aucun`) le mettrait à 0 ; sous 8/3, sans
film en baisse.

## 5. L2 rejoué par-dessus LT (LT.4)

Code de L2 (`99046f75a` + `b12eb7692`) fusionné à trois voies sur l'arbre du lot, en surcouche
(`-overlay`, `lt_tsv/l2_overlay.json`) : `components_device_ti43.go` (neuf), `bit_leaf_readers.go`
(lecteur `consume140d580d0` de L2, à côté de celui de L3a), et le maillon `consumeMoteurDePartie` →
`consumeComposantsDispositif` → `consumeComposantsVueBM4b`. Les conflits de `components_player.go`,
`components_walk_batch9.go`, `vitality.go` (lecteur unique de L3a contre celui de L2, mêmes bits) et
`dispatch_object.go` (commentaires) sont résolus du côté de l'arbre du lot ; la réparation de L2
(`pasDEssai`) EST LT.1. Colonne `product_use` de `ecs_table.tsv` identique.

| Configuration (contre LT) | `1c4c63c2` sains / utiles sains | `d9781168` | Corpus | Gate 2 |
|---|---|---|---|---|
| **L2 sur LT (production, carte v2 20 films)** | **−9 / −422** (19 perdus devenus contredits, 10 gagnés) | −2 / −33 (5 perdus) | +18 485 / +162 600 | **en défaut** |
| L2 sur LT, sans second rang (sonde, contre LT sans second rang) | +10 / 0 | +3 / +8 | — (2 films) | tenu sur ces 2 films |
| L2 sur LT, sous 8/3 et sans second rang (sonde, 20 films, contre LT sous 8/3 sans second rang) | +19 / **−50** (2 perdus) | +3 / +8 | +18 535 / +163 339 | **en défaut** (utiles de `1c4c63c2`) |

**L2 ne tient pas le gate 2 sous LT.** Sa perte passe par le second rang (sans lui, elle disparaît sur
`1c4c63c2` et `d9781168`), comme D-L2-12 l'avait établi en C11 ; avec 8/3 et sans second rang, il
reste un défaut de 50 records utiles sains sur `1c4c63c2` (2 sains perdus), non instruit. Marginale
publiée pour le pilote ; L2 n'est pas intégré ici.

## 6. Révisions

- `grammar.Rev` : `grammar-2026-10-03.3` → **`grammar-2026-10-03.4`** (rang suivant, sans trou ;
  valeur portée par aucune autre branche : LS porte `.3`, LN `grammar-2026-10-04`) ; entrée de
  `rev_chronique.go` ; empreinte régénérée (`711b5d12…`) par la commande du dépôt.
- `killsource.Rev` **constante** (`killsource-2026-10-04`) : sortie `cmd/killsource json` identique
  à l'octet sur les 19 témoins (killsource ne passe pas par la chaîne de tête) ; golden régénéré à
  révision constante (empreinte recopiée, D23).
- `source.Rev`, `profile.Rev`, `objectives.Rev` (`objectives-2026-09-27`) constantes : le gate
  `objectives` est resté vert sans régénération (la lecture des signaux ne change pas).
- `shapes.golden` (types) et les 8 fixtures de contrat (schéma 77) régénérés : différences limitées aux
  chaînes de révision (vérifié ligne à ligne après décompression, 0 ligne hors révision).

## 7. Gates (sorties exactes)

Depuis `apps/go-api` du worktree, `GOCACHE=C:/Users/Guillaume/AppData/Local/go-build-cg2-lt`.

| Gate | Sortie |
|---|---|
| `gofmt -l ./internal/ ./cmd/` | vide (rc 0) |
| `go vet ./...` | rc 0 |
| `go vet -tags=research ./internal/games/halo_infinite/film/...` | rc 0 |
| `go test ./internal/archlint/` | `ok` (31,4 s) |
| G-film `go test ./internal/games/halo_infinite/film/... ./internal/replaybuild/... ./internal/sync/killcollector/... -count=1 -timeout 30m` | rc 0, **20 paquets `ok`** (premier passage : 4 rouges attendus, goldens de révision — killsource, revision, types, fixtures de contrat —, régénérés par les commandes du dépôt, §6) |
| `golangci-lint run --new-from-rev=80d2acd20 ./internal/games/halo_infinite/film/internal/grammar/` | `0 issues.` |
| Vecteurs du lot (`TestUneChaineQuiTraverseUnMasqueNonEcritNeProuveRien`) | vert |
| Gate 5 (`TestFrameClosureDetailleeRendLaCarteDeFrameClosure -v`) | `ks_000d5950` 446 localisées / 27 non localisées, `ks_e5adf7b2` 52 / 2 (comme L0.3) |
| Mutations (`-overlay`, `lt_tsv/jouer.sh`, `lt_tsv/mutations/`) | **5 / 5 ROUGES** : M1 pas NEW sans la règle ; M2 pas DELTA sans la règle ; M3 seule la règle « au-delà de l'archétype » (NEW et DELTA) ; M4 idem sur le DELTA seul ; M5 refuser tout masque non vide (témoins rouges) |
| Gate 2 (carte v2) | §3 |
| Gate 3 — killsource json, 19 témoins (`killsource_19_temoins.txt`) | 19 / 19 rc 0 et **JSON identiques à l'octet** : aucune mort, valeur, ni voie `read_path` changée |
| `replay-equiv` (recette L0 : racine factice au scratchpad, `config/` et références identiques à l'octet au worktree ; base et lot contre les mêmes références) | base et lot 20 / 20 « différents » des références (références non re-figées depuis LS) ; **base contre lot : 4 étapes sur 61 divergent** (`replay_equiv_etapes_divergentes.tsv`) : `artifact` 20 (chaîne de révision), `movementStates.stats` 16 (compteurs de la marche, dont les listes par chaîne, §4.4), `continuousFire.stats` 9, `continuousFire` 2 (`084a804d` paquets à vue C fermée 4 832 → 4 837, trous 27 625 → 27 620, suites de trous 549 → 547, tir tenu dans un trou 899 472 → 899 409 ms ; `1c4c63c2` 13 889 → 13 890, 65 661 → 65 660, 1 087 134 → 1 087 116 ms ; rafales inchangées). `movementStates` (intervalles publiés) identique 20 / 20. Replis : seul `repli_debut_de_liste_ferme_au_bit` de `1c4c63c2` (6 427 → 6 426) et `repli_liaison_par_anticipation` sur 8 films (±1 ou ±2) bougent (`replay_equiv_replis_diff.txt`) |
| Gate 7 — pas de cuisson en lot | tenu : aucune cuisson du parc ; cuissons de `replay-equiv` sur la racine factice du scratchpad |

`replay-corpus-gate` (§6.0 point 6) n'est pas joué dans ce lot : la mission demandait les points 1, 2,
3, le gate 5 et `replay-equiv` ; c'est un écart à signaler au pilote (§8).

## 8. Écarts

- Au critère de l'utilisateur : aucun dans le code de production (une règle lue dans `FUN_142e2da44`).
  La largeur MPP 8/3 n'est employée que par la sonde, comme MESURE, et le dit (`LT_MPP`).
- Au plan : `replay-corpus-gate` non joué (ci-dessus) ; le juge des invariants est celui de la carte
  (définition de L0), joué sur les gagnés et les perdus (§3).
- Aux interdits : un `python3 --version` exécuté par mégarde dans une commande de diagnostic (aucun
  code Python écrit ni exécuté au-delà) ; une écriture vide dans `/tmp`, effacée aussitôt ; sans effet
  sur les livrables.

## 9. Découvertes

- **D-LT-1** Les têtes UTILES du second rang de `debutParFermeture` sont, sur le corpus, des NEW `ti=41`
  des formats anciens dont le corps est mal lu par la largeur MPP non lue dans le jeu ; sous 8/3
  (mesuré, R-VEH), les NEW `ti=41` de `e5adf7b2` passent de 4 à 1 368 dans des paquets fermés, et le
  second rang se retire sans film en baisse (+9 731 sains, +231 532 utiles sains, 6 perdus bruts sur
  des films en hausse). Le retrait du repli `repli_debut_de_liste_ferme_au_bit` est donc suspendu à la
  décision LM (mise de côté le 2026-10-02), pas à une règle de tête. Décision demandée au pilote /
  à l'utilisateur.
- **D-LT-2** À cette base, le second rang est une perte nette sur tous les films sauf `e5adf7b2`
  (variante `chaine+aucun` : +1 566 sains corpus, `4f77afc1` +540, `d9781168` +452, `1c4c63c2` +386).
  Une restriction aux formats dont la grammaire est lue dans l'exécutable (format 27) tiendrait le gate
  2 (aucune perte saine sur ces films hors `4f77afc1` 1, net +540), mais c'est une condition par
  version : hors du critère sans décision explicite. Non appliquée.
- **D-LT-3** L2 sous LT : perte entièrement portée par le second rang (`1c4c63c2` −9 → +10 et
  `d9781168` −2 → +3 sans lui) ; sous 8/3 sans second rang, reste `1c4c63c2` −50 records utiles sains
  (2 sains perdus), non instruit. La reprise de L2 suppose donc LM, le retrait du second rang, puis
  l'instruction de ce reste.
- **D-LT-4** Le compteur du repli diffère entre la marche des 20 films de la carte (sonde,
  `ScanMarcheDesTrames` sur un `ContexteDeFilm` de recherche) et la cuisson de `replay-equiv`
  (`1c4c63c2` 6 425 contre 6 427, `60ae07c4` 108 contre 104, `d9781168` 284 contre 283) : contexte de
  film différent (supposé : largeurs de carte du catalogue posées par la cuisson). Non instruit.
- **D-LT-5** `a349fea8` : 10 paquets non sains changent de première règle (chunks 10 et 27), par des
  liaisons qui changent ; aucun sain touché ; non instruits un par un.
- **D-LT-6** D-L8-8 (la traversée de la marche principale ne s'arrête pas sur un masque non écrit) reste
  ouverte : règle de marche distincte de la tête de liste ; à mesurer à part (elle change les liaisons
  des paquets non fermés).
- **D-LT-7** Les références `replay-equiv` du dépôt ne sont pas re-figées depuis LS (base et lot 20 / 20
  différents) ; §6.0 point 6 demande de les re-figer après chaque vague fusionnée.
- **D-LT-8 — Deux conditions de `pasDEssai` ne sont tenues par aucun test (consignée à l'intégration
  de la vague 2, d'après le contrôle indépendant).** MD (retirer `ok &&` du pas DELTA) et ME (retirer
  `tr.DesyncAt == -1 &&` du pas NEW) restent VERTES ; les mutations équivalentes le sont aussi à la base
  `80d2acd20` (rejoué par le contrôle) : le trou est antérieur à LT. Un témoin « NEW qui se
  désynchronise, masque écrivable » et un témoin « DELTA qui se désynchronise, masque écrivable » dans
  `casDeMasques` (`debut_de_liste_masque_test.go`) le fermeraient.
