# T8 — Bourrage au-delà du tampon : ce que fait le jeu (2026-10-01)

> Piste T8 de la campagne de grammaire, phase 1 (`.ai/PLAN_CAMPAGNE_GRAMMAIRE_2026-10-01.md`, étape 2).
> Ghidra en LECTURE SEULE sur `HaloInfinite.exe` (image base `0x140000000`, serveur HTTP
> `127.0.0.1:8089` : `decompile_function`, `disassemble_function`, `get_xrefs_to`,
> `search_strings`, `search_instructions`). Code lu à la tête du worktree
> `feat/campagne-grammaire` (= `69564ef7d`). Aucune commande `go` lancée, aucun film lu.

## 0. Réponse courte

1. **La primitive de lecture du jeu rend des ZÉROS au-delà du tampon**, sans exception ni arrêt :
   `source.Bits` rend les MÊMES valeurs que le moteur. Sur ce point, `bits.go` dit vrai.
2. **Mais le moteur COMPTE les bits consommés, et un paquet qui en consomme plus que
   `8 × taille` est un paquet EN ÉCHEC** : `FUN_14298816c` (paquet de type 0, la trame) rend
   `false` quand `lecteur+0x2c > 8 × lecteur+0x18`. Le même idiome garde le paquet de type 8
   (`FUN_142987bd4`) et la lecture d'en-tête de `FUN_142985b24`. Le bourrage n'est donc PAS
   « la convention du moteur » pour un flux valide : c'est la valeur que rend la primitive
   pendant une lecture que le moteur déclare ensuite fautive. `bits.go` omet cette moitié.
3. **L'écrivain de la trame est lu** (`FUN_14299d2c8`) : paquet de type 0 =
   `[bit de configuration] ‖ vue A ‖ vue B ‖ vue C` (concaténation AU BIT près, nombre de bits
   exact de chaque vue) `‖ 0 à 7 bits nuls`, taille d'en-tête = `ceil(bits / 8)`. Un décodage
   juste finit donc sur le terminateur de la vue C avec un reste de 0 à 7 bits tous nuls :
   c'est EXACTEMENT `vueCFermee` (`frame_vue_controle.go`), qui devient une propriété de
   l'écrivain prouvée, plus seulement un oracle de sonde.
4. **Conséquence négative pour la cause n° 1** : « vue C : terminateur hors cadre » exige que
   la vue C ait lu son terminateur DANS le payload (`placeDisponible`) avec un reste > 7 bits ou
   non nul, c'est-à-dire une SOUS-consommation. Le bourrage, lui, ne produit que des
   SUR-consommations, qui sortent par « vue B : fin de payload » (595 paquets, carte v1) ou
   « vue C : débordement » (145). **T8 n'explique pas la cause n° 1** ; son effet direct sur la
   fermeture est borné à ~740 paquets sur 20 films (0,3 % de la cause n° 1).
5. **Ce que T8 change quand même** : des records de vue B dont la lecture déborde (ou finit trop
   près de la fin pour laisser la place aux deux terminateurs) sont LIÉS (NEW) ou PUBLIÉS (DELTA)
   par `decodeInferLoop`. Le moteur déclarerait ces paquets en échec, et l'écrivain ne peut pas
   les produire. Correctif proposé : borne d'écrivain `frameLen − queueMin` (§5).

## 1. La primitive de lecture du jeu

### 1.1 Disposition du lecteur de bits (établie)

Le constructeur `FUN_1424c7b4c` pose `+0x1c = 1` puis appelle `FUN_1411b149c(lecteur, début,
taille)` :

```c
// FUN_1411b149c
*(longlong *)(p + 8)    = debut;          // +0x08 début du tampon
*(longlong *)(p + 0x10) = taille + debut; // +0x10 FIN du tampon (pointeur)
*(int *)(p + 0x18)      = taille;         // +0x18 taille en OCTETS
FUN_1406d5cc0(p, 0);
// FUN_1406d5cc0(p, mode) : +0x20 = mode ; +0x40 = +0x08 (curseur octet) ; +0x28 = 0 ;
// +0x24 = 0 (octet d erreur) ; +0x30 = 0 (cache 64 bits) ; +0x38 = 0 ; mode 3/4 : pré-remplit.
```

| Champ | Rôle |
|---|---|
| `+0x08` | début du tampon |
| `+0x10` | fin du tampon (`début + taille`) |
| `+0x18` | taille en octets |
| `+0x1c` | alignement de la taille à l'écriture (1 pour tous les flux de film lus ici) |
| `+0x20` | mode (1 = écriture, 2 = écrit/clos, 3 = lecture) |
| `+0x24` | octet d'ERREUR (remis à 0 à l'armement) |
| `+0x28` | bits CHARGÉS depuis le tampon (lecture) / écrits en mémoire (écriture) |
| `+0x2c` | bits CONSOMMÉS (lecture) / bits écrits (écriture) |
| `+0x30` | cache 64 bits, MSB d'abord |
| `+0x38` | bits déjà pris dans le cache |
| `+0x40` | curseur octet dans le tampon |

Recoupement indépendant : le prologue de la vue C `FUN_142f2539c` fait
`memcpy(dst, reader+8, reader+0x18)` (déjà cité dans `frame_vue_controle.go`) — début en `+0x08`,
taille en octets en `+0x18`.

### 1.2 Le rechargement : des zéros, pas d'erreur (établi)

`FUN_1406d6c7c(lecteur, n)` est le chemin lent commun (`R(1)` = `FUN_1406cf008` y tombe quand
le cache est vide) ; le même rechargement est INLINÉ à l'identique dans `FUN_1406d84b4`
(lecture quantifiée), `FUN_1406d3140` (identifiant de record), `FUN_141f86704` (archétype d'un
NEW) et quatre fois dans `FUN_1406cd128` (boucle de records de la vue B) :

```c
// FUN_1406d6c7c (extrait)
puVar4 = *(ulonglong **)(p + 0x40);          // curseur octet
uVar5 = 0; uVar6 = 0;
if (*(ulonglong **)(p + 0x10) < puVar4 + 1) { // MOINS de 8 octets avant la fin
  if (puVar4 < *(ulonglong **)(p + 0x10)) {   //   il en reste 1..7 : on les charge
    do { uVar5 = uVar5 << 8 | *(byte *)puVar4; puVar4++; uVar6 += 8; ...
    } while (puVar4 < *(ulonglong **)(p + 0x10));
    uVar5 = uVar5 << (0x40 - uVar6);          //   cales en tête, ZÉROS derrière
  }                                           //   il n en reste aucun : uVar5 = 0, uVar6 = 0
} else { uVar5 = bswap64(*puVar4); uVar6 = 0x40; *(p + 0x40) = puVar4 + 1; }
*(int *)(p + 0x2c) += n;                      // consommés : TOUJOURS +n
*(int *)(p + 0x28) += uVar6;                  // chargés : 0 au-delà de la fin
...
return ancien_cache >> (64 - n) | uVar5 >> (64 - reste);
```

Désassemblage de `FUN_1406d84b4` au même endroit : `1406d864c CMP RCX,[R11+0x10] ;
1406d8650 JNC 1406d85fd` avec `R10 = 0, EDX = 0` — au-delà de la fin, le mot chargé vaut 0 et
`+0x28` n'avance pas. AUCUN test de borne, aucun appel d'erreur, aucune exception dans aucune des
primitives lues. Les bits au-delà de `début + taille` valent 0 ; un tampon de queue de 1 à 7
octets est chargé octet par octet, cadré MSB, complété de zéros.

**Équivalence avec `source.Bits`** : `BitsAt` / `bitsAtEdge` rendent 0 pour tout octet
d'indice `>= len(d)`, MSB d'abord ; `types.Packet.Payload` est EXACTEMENT les `taille` octets qui
suivent l'en-tête de 16 octets (`source/film.go`, `appendPackets` : `d[off+16 : off+16+size]`),
comme la copie `FUN_142988338(…, *(int *)(entete + 4))` du jeu. Les valeurs lues hors tampon
sont donc identiques au bit près.

## 2. Le verdict du moteur sur un paquet qui déborde

### 2.1 Paquet de type 0 (la trame) — `FUN_14298816c` (établi)

```c
char FUN_14298816c(longlong film, longlong entete, void *etat) {
  memset(etat, 0, 0x1b01c);
  cVar1 = FUN_142988338(film, *(film + 0x270), *(int *)(entete + 4), 0); // copie taille octets
  if (cVar1 != 0) {
    r = FUN_140689b88(0xd8, 0x6c, "...\\saved_games\\SavedFilmChunks.cpp", 0x533);
    r = FUN_1424c7b4c(r, *(film + 0x270), *(int *)(entete + 4));       // lecteur sur le payload
    FUN_1406d5cc0(r, 3);                                                 // mode lecture
    if (DAT_144db4330 == 0) FUN_142987460(film, r);                      // config + 3 vues
    if (*(char *)(r + 0x24) != 0 || *(int *)(r + 0x18) * 8 < *(int *)(r + 0x2c))
      cVar1 = 0;                                                         // <- ÉCHEC
    *(int *)(r + 0x20) = 5;  FUN_1405a3950(r);
  }
  return cVar1;
}
```

Le paquet est en échec si l'octet d'erreur `+0x24` est posé OU si les bits consommés
dépassent `8 × taille`. Consommer EXACTEMENT `8 × taille` bits passe ; lire dans les 0 à 7 bits
de bourrage de l'écrivain passe aussi. Le contrôle est FAIT APRÈS l'application : les records de
la trame ont déjà été appliqués par `FUN_142987460` (`vtable[0x48]` puis `FUN_1406d07b0`).

Même idiome ailleurs :
- `FUN_142987bd4` (paquet de type 8) : `if (err != 0 || taille * 8 < consommés) → échec` ;
- `FUN_142985b24` (lecture d'un bloc d'en-tête de film) :
  `if (!ok || local_d4 != 0 || local_e0 * 8 < local_cc) → 0`.

### 2.2 Ce que l'échec déclenche (établi, portée limitée)

`FUN_1428e22c0` aiguille par type de paquet (`0 → FUN_1428e2778 → FUN_14298816c`, `9 → saut`,
`8 → FUN_142987bd4`, type inconnu → télémétrie `"FilmBlockReadError"` et `0`). `FUN_1428e27c0`
(le pas de lecture) teste le retour du PREMIER paquet (`1428e2952 CALL 1428e22c0 ; TEST AL,AL ;
JZ 1428e28bf`) : sur échec, le pas s'arrête sans traiter les paquets suivants du pas et rend 0
(la boucle d'avance rapide `do { } while (FUN_1428e27c0() != 0)` de `FUN_1428e219c` s'arrête
là). Les paquets « immédiats » de la boucle interne (`1428e2992`) ont leur retour IGNORÉ. Il n'y
a ni exception, ni marque de film corrompu dans ce chemin : l'échec est un verdict, pas un arrêt
de lecture du flux.

### 2.3 Ce que le moteur ne fait PAS (négatifs mesurés sur le code)

- Aucun test de borne PENDANT la lecture : ni dans les primitives (§1.2), ni dans
  `FUN_1406cd128` (la boucle de la vue B n'a pas de condition de longueur : elle tourne jusqu'à
  un type 0 ou une erreur), ni dans `FUN_142987460`.
- L'écrivain de l'octet `+0x24` n'a pas été trouvé (balayage `MOV byte ptr [reg+0x24],0x1` :
  aucune occurrence dans la famille du lecteur de bits ; question ouverte, sans incidence ici).

## 3. L'écrivain de la trame : la grammaire du paquet de type 0 (établi)

### 3.1 `FUN_14299d2c8` — le paquet

```c
FUN_1429907c8(buf, 0x1b01c);
FUN_1424c7b4c(w, *buf, taille_buf);  local_21c = 1;   // w+0x1c = 1 : alignement 1 octet
FUN_1406d5cc0(w, 1);                                   // mode écriture
FUN_1406d49c4(w /*, bit */);                           // le BIT DE CONFIGURATION
puVar5 = param_3 + 4;
for (3 fois) {                                         // les trois vues, dans l ordre
  FUN_14064c350(sub); FUN_1411b149c(sub, puVar5 + 1, puVar5[-1]); // données, taille octets
  FUN_1432fe23c(sub, *puVar5);                         // sub+0x28 = sub+0x2c = NB DE BITS exact
  FUN_1406d5d14(w, sub);                               // APPEND de exactement sub+0x28 bits
  puVar5 += 0x2402;                                    // pas 0x9008 octets
}
FUN_1406d6d94(w);                                      // vidage final
entete->type = 0;
entete->taille = (FUN_14076b9b0(w) + 7) / 8;           // ceil(bits / 8)
entete->horodatage = FUN_1405a7150(...);
```

Appelant : `FUN_1428dfd30` (tâche), alimentée par `FUN_1428e339c` (le pas d'enregistrement), qui
remplit le tableau de trois vues avec, pour chacune, `taille = vue+0x18`, `bits = vue+0x2c`
(`piVar14[5]`, compteur de bits ÉCRITS de l'écrivain de la vue) et les données `vue+0x08`, dans des
cases de `0x9008` octets (`*(dst-8) = taille ; *(dst-4) = bits`) — même disposition que celle que
`FUN_14299d2c8` relit (`puVar5[-1]`, `*puVar5`, `puVar5 + 1`).

### 3.2 L'append est au bit près — `FUN_1406d5d14`

La boucle copie `sub+0x28` bits : mots de 64 tant qu'il en reste ≥ 64, puis le reste ; le
dernier octet partiel n'apporte que ses `uVar9` bits de tête
(`(byte)*puVar6 >> (8 - uVar9)`). AUCUN alignement entre vues.

### 3.3 Le vidage pose des zéros — `FUN_1406d6d94`

```c
uVar4 = (+0x38 == 64) ? cache : cache << (64 - +0x38); // bits utiles en tête, ZÉROS derrière
... écrit les octets restants ...
+0x28 += +0x38;
taille = (+0x28 + 7) / 8;  si taille % +0x1c != 0 : arrondi à +0x1c ;  +0x18 = taille ; mode = 2
```

Avec `+0x1c = 1`, la taille est `ceil(bits/8)` et le bourrage fait 0 à 7 bits, tous NULS. Le même
calcul `(bits + 7) / 8` écrit la taille des paquets de types 1 (`FUN_14299c654`), 6
(`FUN_14299d508`), 9 (`FUN_14299cd3c`), 10 (`FUN_1428e309c`) et de l'image-clé
(`FUN_1428e339c`).

### 3.4 La grammaire

```
paquet_type_0 := config:R(1)
                 vueA  (FUN_14076a1c4)   { R(1)=1 corps }* R(1)=0
                 vueB  (FUN_1406cd128)   { [R(32)] type≠0 record }* [R(32)] type=000
                 vueC  (FUN_1406cf548)   { R(1)=1 kind corps }* R(1)=0
                 bourrage : 0..7 bits, tous 0
taille_entete := ceil(|config + A + B + C| / 8)
```

(`[R(32)]` = champs étendus `FUN_14076cea8`, `FrameConfig.HasExtraFields`, faux hors ligne.)

**Corollaires exacts** (W = bits écrits, T = `8 × taille`) :
- `T − W ∈ [0, 7]` et ces bits sont nuls → `vueCFermee` (`reste ≤ bourrageMaxBits = 7`, tous
  nuls) est la propriété de l'écrivain.
- Après le DERNIER record de la vue B, il reste au moins `3 (+32)` bits de terminateur B et
  `1` bit de terminateur C : **tout record de vue B d'un flux valide finit à
  `≤ T − queueMin`, `queueMin = 4` (36 avec champs étendus)**.
- Un lecteur synchronisé avec l'écrivain consomme exactement W ≤ T : il ne déborde JAMAIS. Toute
  lecture qui dépasse T est une désynchronisation (et le moteur la déclare en échec, §2.1).

## 4. Le Go, comparé

### 4.1 Ce qui est conforme

- Valeurs hors tampon (`source/bits.go`, `ReadBits`, `BitsAt`, `bitsAtEdge`) : zéros, comme
  `FUN_1406d6c7c`.
- Découpage du payload (`source/film.go`, `appendPackets`) : exactement `taille` octets.
- Vues A et C bornées par `placeDisponible` (`frame_vue_messages.go`), oracle de fermeture
  `vueCFermee` = §3.4.
- Le type de record (`readRecordType` : `R(1)=1 → DELTA ; sinon R(2)`) = `FUN_1406cd128`.

### 4.2 Les écarts

**E1 — documentation de `bits.go` (lignes de l'en-tête, « LA SEMANTIQUE, ET C EST CELLE DU
MOTEUR »).** Elle affirme que « les bits au-delà du tampon valent zéro, silencieusement — c est le
bourrage de queue du moteur » et « Le lecteur ne porte AUCUN drapeau d erreur ». Le moteur rend
bien des zéros, mais il porte un compteur de bits consommés (`+0x2c`) et un octet d'erreur
(`+0x24`), et son appelant déclare le paquet en échec dès que `consommés > 8 × taille`. Le
bourrage du moteur est celui de l'ÉCRIVAIN (0 à 7 bits nuls à la fin) ; lire au-delà n'est pas
une convention, c'est une faute que le moteur sanctionne après coup.

**E2 — la vue B lie et publie sur des lectures qui débordent (`frame_infer.go`,
`decodeInferLoop`, `corpsDeRecordNeuf`).**
- La boucle `for br.BitPos() < frameLen` n'empêche que de COMMENCER un record au-delà de la fin ;
  le corps (`TraverseEntity`, `decodeDelta`) n'est pas borné.
- NEW : `corpsDeRecordNeuf` appelle `w.BindFull(rec.ID, rec.TypeIndex)` dès que
  `DesyncAt == -1`, sans regarder `rec.Trace.EndBit` ; sur des zéros, `consumeMask` lit `0` +
  `R(3)=0` (masque vide, 4 bits) et la traversée est propre : un NEW dont l'en-tête tient dans
  le payload et dont le corps est fait de bourrage est LIÉ.
- DELTA : le record est ajouté à `out` (et ses captures publiées via `poserSlotDeCapture`) même
  quand sa fin dépasse `frameLen`.
- Terminateur : à `BitPos ∈ {frameLen−2, frameLen−1}`, `readRecordType` complète avec des zéros
  et peut rendre `recEnd` → `hitEnd = true` alors que le terminateur est hors payload ; la vue C
  sort alors en `ArretVueCDebordement`. La fermeture est juste (non fermé), mais la vue B est
  comptée « terminée » et ses records publiés.
- L'inférence de chaîne (`frame_chain_infer.go` : `t.EndBit > frameLen`,
  `br.BitPos() > c.frameLen`, `deltaBodyTrial`) borne à `frameLen`, pas à `frameLen − queueMin`.

Le moteur ne lit pas mieux ces cas (il n'a aucune borne pendant la lecture), mais il déclare le
paquet en échec ; et l'écrivain ne peut pas les produire. Donc chacun de ces records est une
lecture désynchronisée.

**E3 — le Rust non plus n'est pas « le moteur ».** Le port refuse les valeurs synthétiques
(`Indisponible`) : c'est plus strict que la primitive (zéros) mais plus fidèle au VERDICT (§2.1).
Il lie pourtant un NEW dès que l'archétype est lu, corps tronqué compris (RAPPORT_PORT_RUST,
ligne « Déclaration NEW conservée malgré un corps tronqué ») — même faute que le Go, autrement.

### 4.3 Pourquoi E2 ne peut pas faire la cause n° 1 (négatif, par construction)

`bloquantDuPaquet` (`frame_closure_classement.go`) ne rend `causeTerminateurHorsCadre` que si la
vue B est atteinte et « terminée », la vue C atteinte ET `Arret == ArretVueCAucun` (terminateur C
lu). Or `consumeVueC` teste `placeDisponible` avant chaque lecture : un terminateur C lu l'est
DANS le payload. `vueCFermee` est alors faux parce que le reste est > 7 bits ou non nul : la
lecture a consommé MOINS que l'écrivain n'a écrit (W − consommés > 0). Les lectures dans le
bourrage sont des SUR-consommations ; elles sortent par « vue B : fin de payload » (595 paquets)
ou « vue C : débordement » (145 paquets) dans `CARTE_FERMETURE_2026-09-26.md`, pas par
« hors cadre » (214 536). Le seul chemin de T8 vers la cause n° 1 est INDIRECT : une liaison NEW
fausse posée sur un corps bourré change l'archétype d'un slot pour les paquets suivants (jusqu'à
l'image-clé suivante). Son poids n'est pas mesuré.

## 5. Correctif proposé (phase 2 — change une sortie)

1. **`source.Bits` : garder les zéros, ajouter le compteur du moteur.** Un accesseur
   `Deborde() bool` (= position maximale atteinte > `NbBits()`), collant, sans changer aucune
   valeur rendue ni le coût de `BitsAt` (le test se fait sur le curseur, pas par bit). Corriger la
   doc de l'en-tête : « zéros comme `FUN_1406d6c7c` ; le moteur déclare le paquet en échec quand
   les bits consommés dépassent `8 × taille` (`FUN_14298816c`) ». Ne change aucune sortie.
2. **Vue B bornée par l'écrivain** (`decodeInferLoop`, `corpsDeRecordNeuf`, branche DELTA) :
   `queueMin := 4 (+32 si HasExtraFields)` ; un record (NEW ou DELTA) dont la fin dépasse
   `frameLen − queueMin` est une désynchronisation (`DesyncAt` posé, NON lié, non publié,
   compté « corps hors cadre d'écrivain ») ; un terminateur de vue B dont la lecture finit après
   `frameLen − 1` ne vaut pas `hitEnd` (cause « vue B : terminateur dans le bourrage »).
3. **Inférence de chaîne** : remplacer les bornes `> frameLen` par `> frameLen − queueMin`
   (`frame_chain_infer.go`, `deltaBodyTrial`, `confirmChainAt`, `inferUnboundArchetype`).
4. Montée de `grammar.Rev` (changement de liaison) — donc preuve au gate de corpus, contre la tête
   POST-J12, comme le plan le prévoit pour toute la phase 2.

## 6. Vecteurs de test (construits d'après l'écrivain `FUN_14299d2c8`)

Cadre : `DefaultFrameConfig()` (`IDLowBits = 13`, `HasExtraFields = false`), départ en tête de
paquet (`skipLeadBits == PacketPreambleBits`, donc bit de configuration puis vue A lue). Slot 256
(identifiant bas `0000100000000`, étiquette `00`).

**V3 — contrôle positif, écrit tel que l'écrivain l'écrit** (slot 256 lié dans le monde et, si
`TablesParVue`, possédé par la vue courante — sinon `rejetDeVue` sort avant le corps ; V2 idem) :
`config 0 | A 0 | DELTA : 1, id 0000100000000 00, base 0, masque 0 000 | B fin 000 | C fin 0 |
bourrage 00000` = 27 bits utiles, taille 4 octets.
Payload `21 00 00 00`. Attendu (jeu ET Go, avant et après correctif) : vue B terminée, vue C
vide, `vueCFermee` vrai (reste 5 bits nuls), consommés 27 ≤ 32.

**V2 — terminateur B dans le bourrage** : les MÊMES 24 premiers bits, taille 3 octets.
Payload `21 00 00`. Le DELTA finit au bit 23 ; `T − queueMin = 20` : l'écrivain ne peut pas
produire ce paquet. Jeu : terminateur B lu jusqu'au bit 26, C au bit 27 > 24 → `FUN_14298816c`
rend `false`. Go aujourd'hui : `readRecordType` lit `0` + `00` de bourrage → `hitEnd = true`,
vue C `ArretVueCDebordement`. Go attendu après correctif : DELTA en désynchronisation (fin 23 >
20), `hitEnd = false`, cause « vue B : terminateur dans le bourrage », aucun record publié.

**V1 — NEW lié sur un corps bourré** : `config 0 | A 0 | NEW : 001, id 0000100000000 00,
ti 010101 (21)` = 26 bits, taille 4 octets, bourrage 000000.
Payload `08 40 05 40`. Corps sur zéros : porte `R(1)` + masque vide 4 bits = fin au bit 31
(plus long si l'archétype choisi a un désérialiseur d'état par défaut porté — choisir pour le
test un `ti` présent au registre du profil de test). `31 > T − queueMin = 28`. Jeu : B fin
jusqu'au bit 34, C au bit 35 > 32 → paquet en échec. Go aujourd'hui (à confirmer par le test) :
`BindFull(slot 256, ti 21)`, puis `recEnd` lu dans le bourrage → `hitEnd = true`, vue C en
débordement. Go attendu après correctif : NEW NON lié, compté, `hitEnd = false`.

Garde-rail associé : un test de propriété « tout paquet FERMÉ a consommé ≤ `8 × taille` et tout
record publié finit à `≤ 8 × taille − queueMin` » sur le corpus de test existant.

## 7. Mesure qui trancherait l'effet (carte de fermeture v2, item 1.6)

Par film (les 20 témoins), sur les paquets de type 0 marchés :
- `deborde` : paquets dont la position maximale atteinte dépasse `8 × taille` (verdict du moteur
  : échec), ventilés en (a) terminateur B lu dans le bourrage, (b) corps de record B qui déborde
  (NEW / DELTA), (c) vue C en débordement ;
- `hors_cadre_ecrivain` : records de vue B dont la fin est dans `(8 × taille − queueMin,
  8 × taille]` (ne débordent pas mais sont impossibles chez l'écrivain) ;
- `new_lie_borde` : NEW LIÉS relevant de (b) ou de `hors_cadre_ecrivain`, et pour chacun le nombre
  de paquets suivants du même chunk (jusqu'à l'image-clé) dont le premier record fautif porte ce
  slot (effet indirect) ;
- contrôle de l'écrivain : sur les paquets FERMÉS, distribution du reste `8 × taille − W` (doit
  être dans `[0, 7]` à 100 %, sinon §3 est faux pour ce build).

Effet attendu : direct ≤ 740 paquets (595 + 145 de la carte v1, une borne : ces paquets restent
non fermés, seule leur cause change) ; indirect = au plus `new_lie_borde × paquets jusqu'à
l'image-clé`. Si `new_lie_borde` est de l'ordre de la dizaine par film, T8 se range en correctif
de propreté des données (liaisons et valeurs publiées), pas en levier de fermeture.

## 8. Découverte hors piste (consignée, non traitée)

- **`FUN_142987460` ignore le code de retour de la boucle de chaque vue** (`14298753d CALL
  [R10+0x40]` suivi de `ADD ESI,[RBP+0x77fb8]`, aucun test de `EAX`). Quand `FUN_1406cd128` sort
  sur erreur (code 2 : DELTA dont l'eid ne concorde pas avec l'entrée de la table de datums,
  `*(tbl + slot*0xa0 + 8) != eid`), le curseur reste APRÈS l'en-tête du record, et la vue C est
  lue de là — exactement la convention de `rejetDeVue` du Go. Comme un lecteur synchronisé avec
  l'écrivain ne prend jamais cette sortie, toute sortie de vue B par rejet en Go signifie que le
  monde du Go diffère de celui du jeu à cet instant (datum manquant) : à verser à T1 / T3.
- `decodeInferLoop` ne lit pas la garde `R(1)` [+ `R(8)`] que `FUN_1406cd128` lit pour les types
  1 et 2 quand `FUN_14076cea8()` est vrai (le décodeur `frame_records.go` la lit, l. 272 et 291).
  Sans effet tant que `HasExtraFields` est faux hors ligne.
