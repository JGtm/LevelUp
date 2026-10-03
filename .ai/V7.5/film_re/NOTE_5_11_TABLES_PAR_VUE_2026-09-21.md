# Note 5.11 — Les tables d entites par vue, et la fin de trame (2026-09-21)

> Lot 5.11.6 / 5.11.7, branche `feat/decfilm-63`. Cette note est le DOUBLE de la passation du
> plan (`.ai/PLAN_DECODEUR_FILM_2026-09-13.md`, section « Passation 5.11 ») : le plan fait foi
> sur l avancement, cette note sur la GRAMMAIRE.

## 1. Ce que le lot a corrige, et c est d abord une erreur de mesure

Le lot 5.11 a conclu « le saut n a pas de champ replique dans le film ». Sa mesure de fermeture
etait fausse : l instrument calculait `len(payload)*8 - EndBit` et **ne sommait que les restes
POSITIFS**. Il publiait « 6,3 bits non lus par paquet, la marche est complete, le film ne cache
rien » alors que la marche **lisait 57 bits AU-DELA de la fin du paquet sur 95,7 % des paquets**
de `dad793c7`, et fabriquait un record fantome (`ti=6` slot 26) a partir de zeros.

Un compte de records ne ferme pas une trame. Le seul gate est le CURSEUR du lecteur, et il
n existait pas : `DecodeFrameViewsCurseur` le rend, `TestMouvement5116Gate` l exige dans
`[0 ; 7]` (le bourrage d octet).

## 2. La grammaire de la fin de trame

`FUN_142987460` (frame-processeur) tient trois objets de vue, appelle `FUN_1406cd128` sur chacun
AVEC LE MEME LECTEUR, puis applique les records. **Il n ecrit rien apres les trois vues.**

`FUN_1406cd128` (la boucle de records, une par vue) :

```
cVar3 = *(char*)(vue + 0x12)          // mode ; != 0 -> sortie au premier tour
cVar1 = FUN_14076cea8()               // drapeau runtime = HasExtraFields ; R(32) par tour
boucle {
  [si cVar1] R(32)
  prefixe = R(1) ; si 0 -> type = R(2)          // type 0 = FIN DE LISTE -> break
  FUN_1406d3140(0, reader, 7, &eid)             // idLow + tag(2)
  si type == 3 (DELTA) :
      lVar11 = (eid & 0x3fffffff) * 0xa0
      si (vue[0x38][lVar11 + 8] == eid && vue[0x38][lVar11 + 2] == (short)type)
            FUN_141f86b58(...)                  // le corps
      sinon uVar14 = 2                          // REJET
      si uVar14 != 0 -> break                   // LA VUE SORT, zero bit de corps
}
```

La table d une vue est un **VECTEUR INDEXE PAR SLOT** : cardinal `(vue[0x40] - vue[0x38]) / 0xa0`,
agrandi par `FUN_1411b3c84(vue+0x38, max(0x1fff, slot))`, entrees construites une a une par
`FUN_1408f15c8`. Une entree jamais posee porte donc `eid = 0`.

**Le PIED DE TRAME est la consequence** : `[records de la vue 1][recEnd 3 bits][en-tete rejete de
la vue 2 : 1+idLow+2][en-tete rejete de la vue 3 : 1+idLow+2][bourrage d octet]`. Sur `dad793c7`
(`idLow` 13) cela fait `3 + 16 + 16` bits.

## 3. La transcription hors ligne

`slotState.Vue` retient la vue ou la liaison a ete posee ; `World.PoserVueCourante` annonce la
vue marchee ; `World.VuePossede` rend la garde ; `rejetDeVue` la pose dans `decodeInferLoop`, la
ou `DecodeFrameRecords` portait deja `GenerationMatches`. Bascule de profil :
`GrammaireBalayage.TablesParVue`, **defaut leve** depuis le 5.11.7.

Deux attributions, et ce sont des lectures : un record NEW lie a la VUE COURANTE ; un slot inconnu
de la vue est rejete comme un slot d une autre vue. **Les liaisons d IMAGE-CLE vont en VUE 0** —
seule attribution que le film permette, confirmee par la fermeture des paquets.

AVANT / APRES :

| mesure | avant | apres |
|---|---:|---:|
| `dad793c7` paquets fermes au curseur (reste 0..7) | **0 %** | **99,50 %** (5 068 a reste 0) |
| `dad793c7` records fantomes `ti=6` slot 26 | 5 202 | **0** |
| `dad793c7` records `ti=0` desynchronises | 15 | **0** |
| `dad793c7` records `ti=35` | 75 | 75 |
| `bfecd02b` records `ti=35` | 97 343 | **114 458** (+17 115) |
| `bfecd02b` desyncs `ti=35` | 6 | **5** |
| `bfecd02b` etalon `i21` | 65,3 % | 65,2 % |
| `bfecd02b` paquets non localises | 1 924 | **1 189** |
| `bcb6d393` `movementStates` (`replay-equiv`) | 1 364 | **1 489** (+125) |

`grammar-2026-09-21.9` / `killsource-2026-09-21.9`. La sortie des FAITS change reellement (le
digest `killsource` bouge) : le parc de lignes de kill devient candidat au backlog D6, sur signal
utilisateur. `SchemaVersion` 67 inchangee.

## 4. Ce que la garde supersede

L inference d archetype sur slot non lie (lot 5.3.3-b) ne s applique plus par defaut :
**l ecrivain ne devine pas**. Le mecanisme reste joignable (`InferenceChaine`) et
`frame_chain_infer_test.go` le met dans l etat ou il travaille (`withChain` abaisse
`TablesParVue`, raison ecrite sur place).

## 5. Le pied, relu — et le « candidat » du 5.11.6 demenu

Les 32 bits se decoupent en deux en-tetes `[prefixe 1][idLow 13][tag 2]`. Les bits qui basculaient
au decollage sont donc des CHAMPS :

| bit du mot | champ |
|---|---|
| **27** | un bit d `idLow` de la VUE 3 — slot **7136 -> 7140** |
| **30** | le premier bit du TAG DE GENERATION de la VUE 3 — **0 -> 2** |

Controle de la decoupe : les slots de la VUE 2 sont LIES (26 et 19, `ti=6` statborg) — c est un
vrai en-tete. Ceux de la VUE 3 ne le sont jamais : le monde hors ligne ne modelise aucune entite
de vue 3 (toutes les liaisons d image-cle vont en vue 0). **Et sur `bfecd02b` ces 32 bits prennent
2 778 et 2 947 valeurs DISTINCTES sur 17 562 paquets** : ce sont des references d entite, pas des
drapeaux. Le candidat est resolu, et il n en est pas un.

**La piste qui reste** : au decollage la vue 3 nomme une ENTITE DIFFERENTE (slot 7140, generation
2). La nommer demande de ventiler les images-cles PAR VUE.

## 6. `i54` et le mantling — lecture d ecrivain seule

Voir le point (4) de la passation du plan : la queue d `i54`, que le depot consomme et jette,
porte — sous la garde runtime `bloc[0x9d]` — un `R(10)` (`bloc+8`, sentinelle `-1`), **trois
vec3** (`bloc+0x18`, `+0x24`, `+0x30`), `bloc+0x3c`, un `R(1)` (`bloc+0xa1`), un **`R(7)`**
(`bloc+0x98`), un **`R(2)`** (`bloc+0x9c` = `etat+0x1294`, QUATRE valeurs) et un `R(1)`
(`bloc+0x9f`). Serialiseur : `FUN_1407ea38c` ; lecteur : `FUN_1408f02c8`.

Il manque UN maillon pour nommer les quatre valeurs : qui ecrit `etat+0x1294` dans l objet
vivant — a chercher depuis le consommateur, jamais depuis l offset.
