# T3 — Fin de vue B et cadrage des vues (campagne grammaire, phase 1, 2026-10-01)

> Piste T3 du plan `.ai/V7.5/PLAN_CAMPAGNE_GRAMMAIRE_2026-10-01.md` (étape 2). Lecture seule : Ghidra
> (`HaloInfinite.exe`, serveur HTTP 127.0.0.1:8089, aucune écriture), code Go à la tête du worktree
> `feat/campagne-grammaire` (base `69564ef7d`). Aucune commande `go`, aucune base, aucun film lu.
> Adresses : image base `0x140000000`.

## 0. Les trois questions, et les réponses courtes

| question | réponse | confiance |
|---|---|---|
| (a) que fait `FUN_142987460` après un code 2/3 de la vue B ? | il l'**ignore** : la vue B s'arrête, la vue C est lue **depuis le curseur laissé à la fin de l'en-tête rejeté**, aucun drapeau d'erreur n'est posé. Le chemin RÉSEAU du même format (`FUN_14076b47c`) fait l'inverse : il cesse de lire les vues suivantes, n'applique AUCUN record du paquet et rend le code à l'appelant | établi (jeu lu) |
| (b) une vue porte-t-elle une longueur ou une borne ? | **non**. L'écrivain concatène des sous-flux de bits sans préfixe (`FUN_1406d5d14`), la seule taille est celle du paquet en octets ; rien ne permet de reprendre la vue C au bon bit après un rejet. Mais l'écrivain impose un **ORDRE** que le parseur ignore : `NEW*` puis `DELTA*` puis `DEL*`, slots croissants dans chaque groupe | établi (jeu lu) |
| (c) une sortie par rejet est-elle jamais légitime ? | **non**. L'écrivain n'émet un `DELTA` que pour une entité dont l'état de vue vaut 3, et cet état n'est atteint qu'après l'écriture ET l'acquittement (immédiat pour le film) d'un `NEW`. Un lecteur qui a lu l'image-clé et tous les `NEW` ne rejette jamais. Un rejet est donc toujours un `NEW` perdu, ou un en-tête lu à une position fausse | établi (jeu lu) |

## 1. L'écrivain du paquet delta du film — grammaire

### 1.1 Qui écrit le paquet de type 0

| maillon | adresse | ce qu'il fait |
|---|---|---|
| enregistreur par tick | `FUN_142f2c3b0` | trois écrivains de bits (`0xd8` o chacun) ; écrivain 0 = `FUN_142f2c050` puis `FUN_1406d49c4(w, 0)` ; écrivain 1 = `FUN_142f2cc78` ; écrivain 2 = concaténation des tampons par joueur `DAT_145178b58 + 0x480 + k*0x4c8` (état 1..2, non vides) puis `FUN_1406d49c4(w, 0)` (`142f2c557: XOR R8D,R8D` / `142f2c57d: XOR R8D,R8D`) ; remis à `FUN_1428e339c(&DAT_144c23178, ...)` |
| copie dans la session film | `FUN_1428e339c` | `memcpy` des trois écrivains dans 3 x `0x9008` o : `+0` taille en octets, `+4` nombre de BITS (`writer+0x2c`), `+8` données |
| sérialisation du paquet | `FUN_14299d2c8` (appelé par `FUN_1428dfd30`, même garde `session+0x120 / +0x1e1d18 == 1`) | `14299d333: MOV R8B,[0x144706104]` puis `CALL 0x1406d49c4` = **le bit de configuration** ; puis trois fois `FUN_1406d5d14(paquet, tampon_i)` (copie de BITS, aucun préfixe) ; en-tête de paquet : type `0` (`MOV word ptr [RBX],DI`, `EDI = 0`), taille `(bits + 7) / 8` octets, horodatage |

D'où, côté écrivain :

```
paquet delta  := cfg:1  vueA  vueB  vueC  bourrage(0..7 bits)
vueA          := { message }  0                         FUN_142f2c050 ; terminateur 1 bit
vueB          := NEW*  DELTA*  DEL*  000                 FUN_142f2cc78 -> vtable de vue B
vueC          := { tampon de contrôle d'un joueur }  0   tampons DAT_145178b58 ; terminateur 1 bit
```

Le lecteur `FUN_142987460` lit exactement cela : `DAT_144706104 = FUN_1406cf008(lecteur)`, puis
trois fois `vtable[0x60]` (zéro bit) et `vtable[0x40]` sur `*(conteneur + 0x228 + rang*8)`.

### 1.2 La vue B du film est un objet de vue B, et sa vtable est celle que le lecteur emprunte

`FUN_140373bf0` construit `DAT_145178bf0` par `FUN_140b87eec`, qui pose `*obj = 0x1436a87e0` et
construit **trois écrivains de bits** en `obj + 0x3612*8 = +0x1b090` (puis `+0x1b168`, `+0x1b240`).
La vtable `0x1436a87e0` (lue en mémoire) :

| slot | fonction | rôle |
|---|---|---|
| `+0x10` | `FUN_142f2e174` | liste des entités de la vue (mot `vue<<30 \| genre<<23 \| slot`) |
| `+0x18` | `FUN_142f24a78` -> `FUN_142f2cee0` | écrit UN record dans le sous-écrivain `obj + 0x1afb8 + genre*0xd8` |
| `+0x20` | `FUN_14076b9c8` | concatène les sous-écrivains dans le paquet |
| `+0x30` | `FUN_1409c98c0` | taille du terminateur : `3` (ou `3 + 32` avec `HasExtraFields`) |
| `+0x38` | `FUN_14076b010` -> `FUN_14076c75c` | écrit le terminateur `[magic si extra] 0 00` |
| `+0x40` | `FUN_1406cd128` | LA boucle de records du lecteur (xref DATA `1436a8820`) |
| `+0x58` | `FUN_140862664` -> `FUN_14086268c` | acquittement d'un paquet |

### 1.3 L'ORDRE des records dans la vue B — la grammaire que le parseur n'emploie pas

`FUN_142f24a78` :

```
lVar3 = param_1 + 0x1afb8 + (longlong)*(int *)((longlong)param_2 + 4) * 0xd8;   // genre -> sous-écrivain
...
FUN_142f2cee0(param_1,&local_38);                                              // écrit dans lVar3
```

Le champ `+4` de la requête est le GENRE (`local_54 = uVar1 >> 0x17 & 0x7f` dans `FUN_142f2cc78`) :
genre 1 -> `+0x1b090`, genre 2 -> `+0x1b168`, genre 3 -> `+0x1b240`. Et `FUN_14076b9c8` :

```
FUN_1406d5d14(param_2,param_1 + 0x1b090);   // genre 1 : NEW
FUN_1406d5d14(param_2,param_1 + 0x1b240);   // genre 3 : DELTA
FUN_1406d5d14(param_2,param_1 + 0x1b168);   // genre 2 : DEL
```

Les genres sont les types de record : `FUN_142f303bc` écrit `FUN_142f2c754(w, 1, ...)` (NEW),
`FUN_142f304a8` `FUN_142f2c754(w, 2, ...)` (DEL), `FUN_142f30610` `FUN_142f2c754(w, 3, ...)` (DELTA).

La liste vient de `FUN_142f2e174`, qui parcourt la table de vue `vue+0x38` par index CROISSANT (mots du
bitmap `vue+0x58`, `uVar9` de 0 vers le haut), et `FUN_142f2cc78` écrit les entrées dans l'ordre de la
liste ; chaque record est AJOUTÉ à son sous-écrivain (et retiré par `FUN_14076a148(w, 1)` s'il ne tient
pas : écriture transactionnelle, jamais de record partiel). Donc :

> **vue B du film = tous les `NEW` (slots croissants), puis tous les `DELTA` (slots croissants), puis
> tous les `DEL` (slots croissants), puis `000`.** Une entité a un seul genre par paquet (la liste
> est calculée une fois, avant toute écriture).

Bornes de l'écrivain, pour mémoire : budget `0x48000 - vtable[0x30]()` bits, arrêt de la liste quand il
reste moins de `0x1000` bits, liste plafonnée à `0x200` entrées si `DAT_145121140 == 1`.

### 1.4 La règle qui décide NEW ou DELTA, et pourquoi un rejet n'est jamais légitime

`FUN_142f2e174` (genre par entrée de vue) :

```
si (*(short *)(vue[0x38] + 2 + slot*0xa0) == 3) :          // l'entité est CRÉÉE chez le lecteur
     ... bitmap de la vue ou entité disparue -> genre 2 (DEL : 0x1000000)
     sinon                                  -> genre 3 (DELTA : 0x1800000)
sinon, si l'entité existe :                  -> genre 1 (NEW : 0x800000)
```

L'état 3 n'est posé que par l'écriture d'un `NEW` puis son acquittement :

- `FUN_142f2cee0`, record de genre 1 écrit et tenu : `FUN_142f2f8f0(view, eid)` -> `*entrée |= 1`,
  `FUN_1408f1358(view, eid, 3 - (view+0x14 != 0x20))`, et `datum+8 |= 2` ;
- `FUN_14086268c` (acquittement) : nœud de genre 1 acquitté -> `FUN_1408f1358(view, eid, 3)` ; nœud de
  genre 2 acquitté -> état 0 ; non acquitté -> état 1 ou 3 (renvoi) ;
- `FUN_142f2cc78` appelle `vtable[0x58](obj, id, 1, 0)` sur chaque paquet qu'il vient d'écrire : **la vue
  du film s'acquitte elle-même, immédiatement**. Aucun renvoi, aucune perte.

Côté lecteur, le miroir : un `NEW` lu passe par `FUN_1408f1314` -> `FUN_1408f1618` (entrée dans la
table de datums du décodeur partagé, pas de 200) puis `FUN_1408f1358(vue, eid, 3)`. Et
`FUN_1408f18d0`, le prédicat de `FUN_1408f1314`, **rend toujours 1** : le code 2 du `NEW` est
inatteignable (s'il y a un occupant, il est ÉVINCÉ, cf. §5).

**Conclusion (c)** : dans un film écrit par le jeu, l'ensemble des eid que la vue B peut écrire en
`DELTA` est inclus dans « déclarés par l'image-clé du chunk » ∪ « `NEW` écrit dans un paquet
antérieur ». Un lecteur qui a lu ces deux sources ne rencontre jamais la garde de `FUN_1406cbaa0`
(`*(uint *)(slot * 200 + t) != eid`). Le code 3 existe pour le réseau (paquets perdus, connexion
désynchronisée), pas pour le film. **Toute sortie de vue B par rejet, hors ligne, est un défaut du
lecteur** : un `NEW` non lu ou non lié (R1), ou un en-tête lu à une position fausse (R2).

## 2. Le lecteur : ce qui suit un code 2/3

### 2.1 Où naissent les codes

`FUN_1406cbaa0` (branche vive, `DAT_14474cd78 == 1` dans l'image), cas `param_1 == 3` (DELTA) :

```
if (eid == 0xffffffff || n <= slot || *(uint *)(slot*200 + t) != eid || ...)
    iVar13 = 3 - (uint)(*(int *)(DAT_144c1cfa8 + 4) != 2);     // 2 ou 3, AUCUN bit lu
```

Le sélecteur de baseline (`FUN_1406cdc04`) n'est lu QU'APRÈS cette garde. Autres sources de code non
nul, après lecture partielle : `FUN_1408f1aa4` (corps de `NEW`) rend 3 après `R(6)` si le descripteur
d'archétype manque ; `FUN_1406caad8` rend 3 après le sélecteur si l'historique désigné manque, et 2
d'entrée si `*(int *)(DAT_144c1cfa8 + 4) != 2` (prédicat de mode global, lu aussi par
`FUN_14053a7c8`).

`FUN_1406cd128` sort de sa boucle dès que le code est non nul (`while (uVar14 == 0)`) ; l'épilogue
`1406cd3a8` écrit `*param_6` et rend le code. Rien d'autre : ni drapeau d'erreur sur le lecteur, ni
remise du curseur.

### 2.2 Le chemin du film : le code est ignoré

`FUN_142987460` (décompilé) :

```
DAT_144706104 = FUN_1406cf008(param_2);
do {
   (**(code **)(*plVar2 + 0x60))(plVar2, 0xa00 - iVar5, ..., local_res18);
   (**(code **)(*plVar2 + 0x40))(plVar2, &uStack_78050, param_2, 0xa00 - iVar5, ..., local_res20);
   ...
} while (uVar7 < 3);           // la valeur de retour de vtable[0x40] n'est lue par personne
```

Puis `FUN_14298816c` ne juge le paquet que sur `lecteur[0x24] != 0 || lecteur[0x18]*8 < lecteur[0x2c]`
(débordement), et `FUN_1428e27c0` (la pompe) n'examine ce verdict que pour le PREMIER paquet de
chaque appel. Le lecteur de bits rend des zéros au-delà du tampon (`FUN_1406d6c7c`) sans poser
`+0x24`.

**Donc, dans le film : après un rejet, la vue B s'arrête, la vue C commence au bit qui suit l'en-tête
rejeté — c'est-à-dire sur le sélecteur de baseline `R(1)` du record rejeté — et le paquet n'est pas
déclaré en erreur.** Le Go fait EXACTEMENT la même chose (`frame_infer.go`, `rejetDeVue` :
`br.SetBitPos(finEntete)` puis `return out, inferred, true`).

### 2.3 Le chemin réseau du même format : le code est une erreur

`FUN_14076b47c` (slot `0x1436a86e8` d'une autre vtable ; le jumeau réseau de `FUN_142987460`, même bit de
configuration, mêmes trois vues sur `param_1 + 0x128` ou `+0x140`) :

```
if ((plVar7 != 0) && (local_c084 == 0)) {
    local_c084 = (**(code **)(*plVar7 + 0x40))(plVar7, ...);      // une vue
}
...
if ((local_c084 == 0) && (*(int *)(param_3 + 0x20) != 4)) {
    ... (**(code **)(*plVar7 + 0x48))(plVar7, puVar15);            // APPLICATION des records
}
return local_c084;
```

Le premier code non nul **arrête la lecture des vues suivantes ET supprime l'application de tout le
paquet**, et il est rendu à l'appelant. C'est la sémantique du format : un code 2/3 dit « ce paquet
n'est pas lisible dans l'état présent ». Le chemin du film l'ignore parce que, pour le film, il ne
peut pas se produire (§1.4) ; ce n'est pas une tolérance du format.

### 2.4 L'écrivain du chemin réseau confirme le cadrage

`FUN_14076aca4` (slot `0x1436a86e0`, voisin de `FUN_14076b47c`) : écrit le bit `DAT_144706104`, appelle
`vtable[0x28]` sur les trois vues, écrit chaque entité par `vtable[0x18]` avec la requête
`{slot = mot & 0x1fff, genre = mot >> 0x17 & 0x7f}`, puis pour chaque vue `vtable[0x20]` (concaténation)
et `vtable[0x38]` (terminateur). Même grammaire que le film.

## 3. (b) Aucune longueur, aucune borne par vue

Preuves, toutes côté écrivain : `FUN_1406d5d14` copie `src+0x28` bits de la source vers la
destination et ne pose rien d'autre (il cumule seulement `+0xd0` / `+0xd4`, des compteurs) ;
`FUN_14076c75c` n'écrit que `[0xf0c3a57e si extra] 0 00` ; `FUN_14299d2c8` n'écrit que le bit de
configuration et les trois tampons ; la vue C (`FUN_1406cf548`) lit `R(1)` puis `kind = R(2)` en boucle,
sans compte. La seule taille du format est celle du paquet (`(bits + 7) / 8` octets dans l'en-tête de
16 octets), et la vue C est le dernier élément : c'est exactement l'oracle `vueCFermee`, et il n'y en a
pas d'autre.

Ce qui existe en revanche, et que le parseur ignore, ce sont les contraintes d'ORDRE du §1.3 :

1. aucun `NEW` après un `DELTA` ou un `DEL` dans la même vue B ; aucun `DELTA` après un `DEL` ;
2. slots strictement croissants dans chaque groupe ;
3. corollaire : l'en-tête qui suit un `DELTA` de slot `s` est un `DELTA` de slot `> s`, un `DEL`, ou
   `000`.

Elles ne donnent pas la largeur du corps rejeté (l'archétype reste inconnu), donc pas de reprise
directe. Elles donnent un ORACLE D'ALIGNEMENT local, indépendant de la fermeture, et elles
contraignent fortement une recherche de reprise (§6, C4).

## 4. Comparaison au Go, et les écarts

| point | jeu | Go | écart |
|---|---|---|---|
| curseur après rejet | fin de l'en-tête rejeté, vue C lue de là (film) | idem (`rejetDeVue`, `SetBitPos(finEntete)`) | **aucun** sur le curseur |
| sens du rejet | erreur (chemin réseau) ; jamais produit par l'écrivain du film | `hitEnd = true`, la vue B est « terminée » (`decodeInferLoop`) ; la carte classe le paquet « vue C : terminateur hors cadre » (`bloquantDuPaquet`, dernier cas) | **la cause est mal nommée** : la première faute est le rejet, pas la vue C |
| ce que lit la vue C après un rejet | le sélecteur `R(1)` du record rejeté, fermé dans 99,992 % des DELTA (`TestBaseline516Selecteur`, `bfecd02b` 174 592 / 174 606) -> terminateur | idem : `consumeVueC` lit ce 0, vue C « vide », 1 bit | prédiction : vue C vide sur ~100 % des sorties par rejet (témoin : les douze paquets de 96 bits de `dad793c7`, bit 53) |
| ordre des records de vue B | NEW* DELTA* DEL*, slots croissants | aucun contrôle ; `decodeInferLoop` accepte tout ordre | grammaire non portée (oracle manquant) |
| `NEW` sur un slot occupé | `FUN_1408f18d0` évince l'occupant (`FUN_1408f1358(vue, ancien, 0)`, `FUN_1408f12c4(dec, ancien, 1)`) puis crée | `contreditUneEntiteVivante` REFUSE la liaison si le slot est lié en dur à un autre archétype | divergence de modèle (§5) |
| après un rejet, lire la vue C ? | réseau : non ; film : oui, mais n'arrive jamais | oui, et le verdict `Atteinte = true` nourrit `tir_continu.go` (`Reached`, `OpenViewB`) | la vue C lue derrière un rejet est une lecture fabriquée |

## 5. Le `NEW` sur un slot occupé (lu en passant, pèse sur R1)

`FUN_1408f18d0` :

```
lVar2 = slot * 0xa0;
if (*(short *)(vue[0x38] + 2 + lVar2) != 0) {
    iVar1 = *(int *)(vue[0x38] + 8 + lVar2);
    if (iVar1 != -1) { FUN_1408f1358(vue, iVar1, 0); FUN_1408f12c4(*(vue+0x20), iVar1, 1); }
}
return 1;
```

Le lecteur du jeu ne refuse jamais un `NEW` : il libère l'ancien occupant du slot. Le refus Go
(`contreditUneEntiteVivante`, lot D-fix du 2026-09-24, mesuré utile sur `0797ce72`) visait des `NEW`
mal alignés ; il a un second effet : quand le `NEW` est vrai (slot recyclé sans `DEL` lu), l'ancienne
liaison survit, et chaque `DELTA` de la nouvelle entité est lu sous l'archétype de l'ancienne. Un `NEW`
mal aligné se détecte désormais AUTREMENT, par l'ordre du §1.3 (un `NEW` après un `DELTA` est
impossible). Non mesuré ici.

## 6. Constats, correctifs, vecteurs, mesures

### C1 — Un rejet n'est jamais légitime dans un film (établi)

- Jeu : §1.4 (`FUN_142f2e174`, `FUN_142f2cee0`, `FUN_142f2f8f0`, `FUN_14086268c`, `FUN_142f2cc78`
  acquittement `vtable[0x58](obj, id, 1, 0)`, `FUN_1408f1314`, `FUN_1408f18d0`).
- Go : `frame_infer.go` `rejetDeVue` (compte et sort), `LierParRepliDAnticipation` (repli 5.23).
- Écart : le Go traite le rejet comme une fin de vue ordinaire ; c'est un symptôme.
- Correctif (instrument, phase 1) : chaque rejet est classé R1 / R2 (C4). Correctif (phase 2) :
  lire les `NEW` perdus à leur source (piste T1) ; le repli d'anticipation reste un repli compté.
- Mesure : pour chaque eid rejeté (premier rejet par chunk), chercher dans les paquets delta
  ANTÉRIEURS du même chunk un en-tête `NEW` de même slot et même tête (`enteteNeufEn`, toutes
  positions), et ventiler : (a) dans la tête d'un paquet à événements, avant le début localisé ;
  (b) dans un paquet « liste non localisée » ; (c) lu mais désynchronisé ; (d) refusé
  (`NeufsContreUnVivant`) ; (e) introuvable. Le jeu dit que (e) ne contient que des R2 ou des
  entités de l'image-clé non lues (moitié de payload non marchée, 5.19 §5.4).

### C2 — Après un code 2/3 : le film continue sur la vue C, le réseau annule le paquet (établi)

- Jeu : `FUN_142987460` ignore le retour de `vtable[0x40]` ; `FUN_14076b47c` arrête la lecture et
  l'application au premier code non nul ; `FUN_1406cd128` épilogue `1406cd3a8`.
- Go : curseur conforme. Classement non conforme : `frame_closure_classement.go` `bloquantDuPaquet`
  rend `causeTerminateurHorsCadre` pour un paquet dont la vue B est sortie par rejet.
- Correctif (instrument) : faire porter la sortie de vue B au classement et nommer la cause
  « vue B : rejet hors datum » (ou « de vue ») AVANT tout examen de la vue C ; la carte v2
  (`frame_closure_detail.go`, `SortieDeVueB`) en a déjà la donnée. Correctif (phase 2, change des
  statistiques publiées) : sur rejet, ne pas lire la vue C (verdict `Atteinte = false`, cause
  « rejet »), comme le chemin réseau.
- Vecteur (écrivain, 96 bits, `dad793c7`, NOTE 5.16 §1) :
  `10100000011110110100001000000000 01000100101000100100100001000001 00000100110000011001110000000000`
  = `cfg 1 | vueA 0 | DELTA slot 123 tag 1 sel 0 masque 0x1 R(8) | DELTA slot 1298 tag 1 sel 0
  masque 0x2 R(24) | 000 | vueC 0 | 0000`. Monde SANS le slot 1298 : sortie « rejet hors datum » au
  bit 37, curseur 53, vue C vide (bit 53 = le sélecteur), curseur 54, reste 42 bits non nuls ; cause
  attendue après correctif « vue B : rejet hors datum », pas « vue C : terminateur hors cadre ». Monde
  AVEC `1298 -> ti=47` : ferme à reste nul (4 bits).
- Mesure : dans la carte v2, croiser `Cause = hors cadre` x `Sortie` x `VueC.Vide`. Prédiction :
  (i) la grande majorité du « hors cadre » est `Sortie = rejet` (5.15 : 23 452 rejets contre 400
  terminateurs sur `bfecd02b`) ; (ii) parmi les sorties par rejet, vue C vide >= 99,9 %.

### C3 — Pas de longueur, pas de borne : la reprise n'est pas dans le format (établi, constat négatif)

- Jeu : §3 (`FUN_1406d5d14`, `FUN_14076c75c`, `FUN_14299d2c8`, `FUN_1406cf548`).
- Go : `vueCFermee` est déjà la seule borne du format ; rien à ajouter.
- Écart : aucun. La recherche d'un champ de taille à porter est close.
- Mesure : aucune ; la fermeture au bourrage reste l'oracle unique de cadrage de fin.

### C4 — L'ordre de l'écrivain : NEW* puis DELTA* puis DEL*, slots croissants (établi)

- Jeu : `FUN_142f24a78` (sous-écrivain = genre), `FUN_14076b9c8` (concaténation 1, 3, 2),
  `FUN_142f2e174` (liste par slot croissant), `FUN_142f2cc78` (écriture dans l'ordre de la liste),
  `FUN_140b87eec` (la vue du film a trois sous-écrivains en `+0x1b090/+0x1b168/+0x1b240`).
- Go : `decodeInferLoop` n'emploie aucun ordre ; `debut_de_liste.go` sait que les `NEW` sont en
  tête (lot M4b) mais pas que les `DELTA` sont croissants ni que le premier `DELTA` est le plus petit
  slot modifié (pas forcément 123).
- Correctif (instrument, phase 1) : un ORACLE D'ORDRE sur les records rendus par la vue B. Pour un
  rejet : R1 « en-tête plausible » si le slot rejeté est strictement supérieur au dernier `DELTA` lu
  et qu'aucun `DEL` n'a été lu ; R2 « en-tête impossible » sinon. Pour un paquet sorti par
  terminateur et non fermé : la première violation d'ordre désigne le record d'avant comme premier
  suspect de largeur. Correctif (phase 2) : une violation d'ordre est une désynchronisation (arrêt de
  la vue, record compté) ; la reprise après un rejet R1 peut être cherchée sous contrainte d'ordre
  (en-tête suivant = `DELTA` de slot > rejeté sur un slot lié, chaîne croissante jusqu'à `000`, vue C
  fermée), ce que la recherche du 5.15 §7 (a), sans ordre, ne pouvait pas trancher (10,4 % et
  322 classes d'écart).
- Vecteurs : (V1) le paquet de 96 bits ci-dessus, monde sans 1298 : rejet R1 (1298 > 123). (V2) le
  même paquet, les deux `DELTA` permutés (slot 1298 puis 123), monde complet : décode et ferme, mais
  l'oracle doit lever « ordre violé » au second record — l'écrivain ne produit jamais ce flux.
  (V3) un `DELTA` de slot 123 suivi d'un en-tête `NEW` (`0 01`) : « NEW après DELTA ».
- Mesure : sur les 20 films, part R1 / R2 des sorties par rejet, par build ; part des paquets FERMÉS
  qui violent l'ordre (attendu : 0 — c'est le test de l'oracle lui-même ; un paquet fermé qui viole
  l'ordre réfuterait §1.3 ou désignerait une fermeture fortuite).

### C5 — Le jeu évince sur `NEW`, le Go refuse (établi pour le jeu, effet non mesuré)

- Jeu : `FUN_1408f18d0` (§5) ; `FUN_1408f1314` rend toujours vrai.
- Go : `frame_infer.go` `corpsDeRecordNeuf` -> `contreditUneEntiteVivante` -> `refuserUnNeuf`.
- Écart : un `NEW` vrai sur un slot recyclé sans `DEL` lu laisse l'ancienne liaison en place.
- Correctif (phase 2) : remplacer le refus par l'éviction du jeu, la protection contre les `NEW` mal
  alignés passant à l'oracle d'ordre (C4) ; à mesurer avant tout changement.
- Mesure : `NeufsContreUnVivant` par film ; pour chaque refus, le `NEW` est-il en tête de vue B (ordre
  respecté) ? si oui, les `DELTA` suivants de ce slot se lisent-ils (taux de désynchronisation) sous
  l'archétype refusé contre l'archétype gardé ?

## 7. Ce que T3 ne tranche pas

- POURQUOI les `NEW` sont perdus (piste T1). T3 dit seulement qu'ils EXISTENT, en tête de la vue B
  d'un paquet antérieur du chunk, slots croissants.
- La valeur de `view+0x14` pour la vue du film (état 2 ou 3 posé à l'écriture d'un `NEW`) : sans
  effet sur la conclusion, l'acquittement immédiat pose 3.
- Le prédicat `*(int *)(DAT_144c1cfa8 + 4) == 2` : objet global de `0x113618` o (`FUN_1410ffa34`) ;
  son sens exact n'est pas lu. Il vaut 2 en lecture de film, sans quoi aucun `DELTA` ne serait lu.

## 8. Découvertes hors périmètre (consignées, non traitées)

1. `marchViews = 8` (`object_deaths_march.go`) et `killsource.Options.Views` : l'écrivain du film
   (`FUN_14299d2c8`, `FUN_142f2c3b0`) prouve TROIS vues ; toute boucle de records au-delà de la vue B
   lit la vue C (et le bourrage) comme des records d'entités. Déjà mesuré perdant au lot 5.3.3-b
   (`movement_states.go`, `MovementStateViews`), désormais prouvé chez l'écrivain.
2. Localisateur des paquets à événements (`marchLocateStrict`) : la signature « premier record = DELTA
   du slot 123 » n'est vraie que si aucun slot < 123 n'est modifié dans le tick ; sinon les `DELTA`
   de petits slots sont sautés comme les `NEW`. `debutParChaine` ne part que d'un candidat `NEW` :
   un paquet sans `NEW` mais avec un `DELTA` de slot < 123 perd ce record.
3. `debutDeLaListe` : les candidats `NEW` sont restreints à la bande d'archétype des images-clés
   (`SlotDeLArchetype`) ; une entité née et morte entre deux images-clés n'y figure pas forcément,
   et la chaîne exige que TOUS les `NEW` de tête se traversent sans désynchronisation. Les deux
   conditions recoupent exactement la population des 801 eid de la NOTE 5.26.
