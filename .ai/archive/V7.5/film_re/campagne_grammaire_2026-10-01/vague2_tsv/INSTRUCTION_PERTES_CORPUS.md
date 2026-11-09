# Instruction des deux pertes [FILET] du gate de corpus de la vague 2 (rc 1) — 2026-10-04

Tête `feat/campagne-grammaire` = `bd9193d16`, base de mesure `6fa631df0`. Lecture seule du dépôt :
sources extraites par `git archive` dans `v2-instr/src_head` et `v2-instr/src_base`, variantes en
surcouche `-overlay` (`v2-instr/var/*.json`), GOCACHE `go-build-cg2-instr`, une commande `go` à la fois,
films lus en place (`LevelUp/data/cache/film_chunks`), aucune cuisson. Ghidra : HTTP direct
127.0.0.1:8089, lecture seule.
Convention : **mesuré** = compté par la sonde sur les films ; **établi** = lu dans le jeu ;
**estimé** = inférence écrite.

## 0. Verdicts

| Perte (gate, `--base=6fa631df0`) | Paquet | Part | Verdict |
|---|---|---|---|
| `084a804d` `stances.records` 448 661 -> 448 659 (−2) | `2:164` | **LT (règle du masque)**, −1 | **Lecture fausse retirée, D2 au sens strict.** Avant : une fermeture factice (fermée au bit, masque au-delà de l'archétype). Après : un paquet sain, 20 entrées de vue C publiées. |
| | `47:674` | **Règle d'ordre de la vue B**, −1 | **Lecture fausse retirée.** Avant : une création fantôme d'un bipède vivant, dans une chaîne que l'écrivain ne peut pas écrire. Aucune fermeture d'un côté ni de l'autre, donc hors de la lettre de D2. |
| `a349fea8` `continuousFire.holesNotClosing` 22 852 -> 22 856 (+4) | `10:530`, `10:534`, `10:556`, `10:558` | **LT, 100 %** (effet aval du paquet `10:528`) | **Aucune lecture perdue** : 4 trous changent seulement de cause (`holesKind` 956 -> 952). Le total des trous, les fermés et les entrées restent identiques. L'ancienne cause venait d'une lecture contraire à l'écrivain. Hors de la lettre de D2. |

Aucune des deux pertes n'est une vraie perte. Recommandation pour le rc 1 au §5.

## 1. Instrument et contrôle

Sonde `v2-instr/probe/v2instr_research_test.go` (tag `research`, copiée dans les deux arbres du
scratchpad, jamais versionnée). Elle rejoue le corps de `ScanMarcheDesTrames` (même canal des états,
même collecteur du tir continu, enveloppé pour lire au passage le verdict de vue C). Par paquet, elle
écrit : le début de la vue B (genre de début), sa sortie, le verdict de fermeture, les records et les
records bipèdes, et le verdict de vue C. Sur les paquets désignés, elle ajoute les records (genre, ti,
slot, génération, bit, liaison, masque, nombre de composants de l'archétype) et la liaison d'un slot
dans la table d'entités.

**Contrôle (mesuré)** : les totaux de la sonde égalent ceux des artefacts du gate, pour la base comme
pour la tête (`v2-integ/rev/gate_work{,/cuisson-base}/data/cache/replays/halo_infinite/*.json`) :

| | `084a804d` records | fermés | `holesNotClosing` | `a349fea8` `holesKind` | `holesNotClosing` |
|---|---|---|---|---|---|
| artefact base / sonde base | 448 661 / 448 661 | 4 832 / 4 832 | 23 908 / 23 908 | 956 / 956 | 22 852 / 22 852 |
| artefact tête / sonde tête | 448 659 / 448 659 | 4 837 / 4 837 | 23 903 / 23 903 | 952 / 952 | 22 856 / 22 856 |

Variantes en surcouche sur la tête (`debut_de_liste.go` seul) :
- `noOrder` : LU + LT, sans la règle d'ordre ;
- `noMasque` : LU + règle d'ordre, sans LT ;
- `neither` : LU seul.

Résultats (`v2-instr/out/resume.tsv`, mesuré) :

| Variante | `084a804d` records | fermés | `a349fea8` `holesKind` / `holesNotClosing` |
|---|---|---|---|
| base `6fa631df0` | 448 661 | 4 832 | 956 / 22 852 |
| `neither` (LU) | 448 661 | 4 832 | 956 / 22 852 |
| `noMasque` (règle d'ordre seule) | 448 660 | 4 834 | 956 / 22 852 |
| `noOrder` (LT seul) | 448 660 | 4 837 | 952 / 22 856 |
| tête `bd9193d16` | 448 659 | 4 837 | 952 / 22 856 |

- `neither` égale la base paquet par paquet (TSV identiques à l'octet) : LU ne change rien sur ces deux
  films.
- Sur `a349fea8`, `noOrder` égale la tête à l'octet.
- Les paquets où le nombre de bipèdes change sont exactement deux : `2:164` (`noOrder` et tête) et
  `47:674` (`noMasque` et tête). **Part LT = −1, part de la règle d'ordre = −1, sans interaction.**

## 2. `084a804d 2:164` — LT : une fermeture factice retirée (D2)

Base (mesuré, `out_d/base_084a804d_detail.txt`). La chaîne de tête part du bit 5935 et tombe au bit
près sur la signature (DELTA du slot 123, bit 7046) :

| # | genre | ti | slot | bit | masque | composants de l'archétype |
|---|---|---|---|---|---|---|
| 0 | NEW | 37 | 3315 (gen 3) | 5935 | `0x2000030000000` (bits 36, 37, 49) | **31 : masque au-delà de l'archétype** |
| 1 | NEW | **35 (bipède)** | **6144** | 6184 | `0x110000000049` | 64 |
| 2-3 | DELTA | 5 | 71, 72 | 6758, 6902 | | |
| 4 | DELTA | 4 | 123 | 7046 | | (début localisé) |

- **Avant** : le paquet est fermé au bit (consommés = longueur = 12 104) mais contredit l'écrivain.
  Selon le juge de L0 (carte de base), il porte « masque au-delà de l'archétype + masque épars non
  croissant ». C'est une **fermeture factice**, `Fermee = false`, aucune entrée de vue C publiée.
  L'écrivain du masque (`FUN_142e2da44`, établi au lot LT §1) ne pose aucun bit au-delà de
  `*(desc+0x4320)` : le record 0 n'a pas été écrit là par le jeu tel qu'il est lu. Le record 1 commence
  là où le corps du record 0 finit : sa position n'a aucune autre preuve.
- **Après (LT)** : la chaîne est refusée, la liste part de la signature (7046). Le paquet ferme au bit
  sans règle contredite (`Verdict 1`, `Fermee = true`), et **20 entrées de vue C** sont publiées.
- **Le record bipède perdu est le NEW du slot 6144.** Indices qu'il est faux (mesuré ; la conclusion
  est estimée) :
  - aucun delta sur le slot 6144 dans tout le chunk 2 (`out_h`, paquets 2:0 à 2:1206), alors que les
    bipèdes de ce film occupent les slots 512 à 535 ;
  - la liaison 6144 -> ti 35, posée par ce NEW, est absente dès la première trame du chunk 3 (`3:6`,
    `out_g`) : l'image-clé ne la porte pas ;
  - même chose pour le slot 3315.
- La règle d'ordre ne touche pas ce paquet (NEW 3315 < NEW 6144, puis DELTA 71 < 72 < 123 : ordre
  respecté ; `noMasque` égale la base sur ce paquet).

**Verdict : exception D2 au sens strict.** La baisse s'explique par une fermeture factice retirée, que
remplace un paquet sain.

## 3. `084a804d 47:674` — règle d'ordre : une création fantôme retirée

Base (mesuré) :

| # | genre | ti | slot | bit | liaison |
|---|---|---|---|---|---|
| 0 | **NEW** | **35** | **549 (gen 1)** | 269 | lu NEW |
| 1 | **DEL** | — | 64 | 630 | |
| 2 | DELTA | 4 | 123 | 680 | (début localisé) |
| 4 | **DELTA** | 35 | **549 (gen 2)** | 880 | sous la liaison du NEW |

- **La chaîne contredit l'écrivain deux fois** (établi) :
  - un DEL précède un DELTA. `FUN_14076b9c8` (relu ce jour dans Ghidra) concatène les trois tampons
    `+0x1b090`, `+0x1b240`, `+0x1b168`, soit NEW, puis DELTA, puis DEL (l'attribution des tampons a
    été établie aux lots LS et LT, non re-vérifiée ici ligne à ligne) ;
  - le slot 549 porte un NEW et un DELTA dans la même vue B. Or `FUN_142f2e174` range chaque entrée de
    la table de vue dans un seul sous-écrivain (établi au lot LT §1).
- Le juge de L0 de la carte de base le relève : « vue B : sortie par rejet + **ordre de la vue B** ».
- **Le slot 549 est un bipède vivant** (mesuré) : lié en génération 2 par la table de datums, il reçoit
  un DELTA (masque `0x2200000`) à chaque trame de 47:600 à 47:672, et encore dans 47:674 (bit 880). Le
  NEW de génération 1 au bit 269 est donc une création fantôme d'une entité vivante. Après la règle, ce
  DELTA se lit sous la liaison par datum, comme dans les trames précédentes.
- Ni la base ni la tête ne ferment ce paquet : sortie par rejet au même eid `0x80000244`. Ce n'est donc
  pas une fermeture factice, et D2 ne s'applique pas à la lettre.
- Limite : la tête prend le début de la signature (bit 680), qui n'est pas prouvé non plus (paquet non
  fermé, objection D-REV2-1 non instruite). Ce qui est prouvé, c'est que la lecture retirée est fausse.

**Verdict : lecture fausse retirée, pas une vraie perte.** Le record perdu est une création que
l'écrivain ne peut pas écrire là, sur un bipède dont les deltas continuent d'être lus.

## 4. `a349fea8` +4 `holesNotClosing` — LT : 4 trous reclassés

- Source (mesuré). Paquet `10:528`, base : la chaîne de tête part du **NEW ti 37, slot 2667**, au
  bit 601. Son masque est `0x440106008` (bits 3, 13, 14, 24, **34, 38**) pour **30 composants** : masque
  au-delà de l'archétype, refusé par LT. Le paquet lui-même ne change pas de catégorie : queue opaque,
  vue C arrêtée sur un kind non porté, avant comme après.
- Effet aval (mesuré). En base, la liaison 2667 -> ti 37 (« lu NEW ») fait lire un DELTA du slot 2667
  dans `10:530` à `10:558` (masques `0x3`, `0xf`, `0x2000000f`). En tête, ce slot n'est pas lié :
  l'en-tête du delta est rejeté (eid `0x40000a6b` = slot 2667). La vue B sort par rejet, et la vue C,
  lue d'un bit, ne ferme pas le paquet.

| Paquet | Base : sortie, verdict, vue C | Tête : sortie, verdict, vue C |
|---|---|---|
| `10:530`, `10:534`, `10:556`, `10:558` | terminateur, queue opaque, arrêt « kind non porté » (2 entrées lues), **juge de L0 : « vue C kind non nul »** | rejet, refus, ne ferme pas (`NotClosing`) |
| `10:536` à `10:544` (5 paquets) | terminateur, ne ferme pas | rejet, ne ferme pas (catégorie inchangée) |

- **Pas de lecture perdue** (mesuré). Trous 28 327 = 28 327, fermés 424 = 424, et entrées de vue C,
  rafales et tirs à 0 de part et d'autre (ce film ne publie aucune rafale). Records bipèdes
  identiques (383 483). Seule la cause de 4 trous passe de `holesKind` à `holesNotClosing`.
- **L'ancienne cause venait d'une lecture contraire à l'écrivain.** Deux raisons :
  - la vue C lue en base contient une entrée de kind ≠ 0, que le juge de L0 marque contraire à
    l'écrivain ;
  - elle n'est atteinte qu'après un delta lu sous une liaison dont la seule source est un NEW au
    masque non écrivable (établi : `FUN_142e2da44`).
- La liaison 2667 -> ti 37 n'a aucune autre preuve (mesuré). Elle n'est jamais posée en tête dans tout
  le reste du film. En base, elle disparaît dès la première trame du chunk 11 (`11:6`), car l'image-clé
  ne porte pas le slot 2667. **Non établi** : que l'en-tête du NEW (slot 2667, archétype 37) soit faux.
  Un en-tête juste au corps mal lu, sur un format ancien (`a349fea8` = format 33), est possible : c'est
  la famille D-L0-2 / D-LT-1. Dans les deux cas, aucune donnée publiée ne change.
- Part : `noOrder` égale la tête à l'octet ; `noMasque` égale la base sur ces compteurs. **LT = 100 %,
  règle d'ordre = 0.** Cela confirme l'attribution « mesurée » du plan.

**Verdict : pas une vraie perte.** C'est un reclassement de 4 trous, à total constant ; leur ancienne
cause reposait sur une lecture que l'écrivain contredit. Hors de la lettre de D2 : aucune fermeture,
factice ou non, n'est en jeu.

## 5. Recommandation pour le rc 1

1. **Admettre les deux pertes**, comme lectures fausses retirées :
   - `084a804d 2:164` relève de D2 au sens strict (fermeture factice remplacée par un sain) ;
   - `084a804d 47:674` et `a349fea8` (4 trous) relèvent du principe de D2, pas de sa lettre. Dans les
     deux cas, le juge de L0 marquait déjà la lecture de base comme contraire à l'écrivain (« ordre de
     la vue B », « vue C kind non nul »), et la lecture retirée reposait sur un record que l'écrivain
     n'écrit pas (ordre NEW, DELTA, DEL ; un genre par entité ; masque dans l'archétype).
   Le texte de D2 ne parle que de fermetures factices. L'admission de ces deux cas demande donc une
   **décision datée du pilote ou de l'utilisateur**, avec cette justification (proposition :
   « exception D2 étendue aux lectures que le juge de L0 contredit, paquet par paquet instruit »).
2. **Aucun réglage n'est à faire** : les deux règles qui jouent (masque dans l'archétype,
   `FUN_142e2da44` ; ordre de la vue B, `FUN_14076b9c8` et `FUN_142f2e174`) sont générales et lues dans
   le jeu. Aucun ordre, seuil ni condition n'a été choisi à la mesure.
3. **Découverte notée, non traitée (règle 5)** :
   - le filet `holesNotClosing` compte un reclassement entre causes de trou comme une perte, même quand
     `holes` est constant. Ce n'est un défaut du gate que si le pilote le juge ainsi ; rien n'est
     modifié ;
   - le NEW `ti 37` à masque hors archétype sur `a349fea8` (format 33) et `084a804d` relève peut-être
     de la même famille que les NEW `ti 41` du lot LT §4.2 (corps mal lu sur format ancien). Non
     instruit.

## 6. Pièces (scratchpad `v2-instr/`)

- `probe/v2instr_research_test.go` : la sonde.
- `var/{noOrder,noMasque,neither}/debut_de_liste.go` et `var/*.json` : les variantes en surcouche.
- `out/{base,head,noOrder,noMasque,neither}_{084a804d,a349fea8}.tsv` (un paquet par ligne) et
  `out/resume.tsv`.
- `out_d/*_detail.txt` (records des paquets instruits, base et tête), `out_c10`, `out_e`, `out_f`
  (liaison du slot 2667, chunks 10 et suivants), `out_g` (slots 6144 et 3315, chunks 2 et 3), `out_h`
  (chunk 2 complet, base).
- `carte_{base,tete,lult}.tsv` : lignes des deux films extraites des cartes v2 de la revue
  (`v2-integ/rev/carte_*`).

Colonnes du TSV par paquet : chunk, index, ts, début de vue B (1 en-tête, 2 signature, 3 chaîne,
4 fermeture, 5 fermeture au bit, 6 non localisé), bit de début, bits, sortie de vue B, verdict,
fermé au bit, règle, records, bipèdes, désynchronisés, verdict reçu, vue C atteinte, fermée au bit,
invariant, fermée, arrêt, entrées, tête.
