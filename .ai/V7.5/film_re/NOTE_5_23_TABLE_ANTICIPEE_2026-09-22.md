# NOTE 5.23 — LA TABLE ANTICIPEE DES ARCHETYPES

Lot 5.23, branche `feat/decfilm-73`, base `f0bd32b0a` (lots 5.1 a 5.21, grammar-2026-09-22.10,
schema 67). Sur la mesure du lot 5.20.2 : **74,7 % des slots rejetes sont declares, avec leur
archetype, par l image-cle du chunk SUIVANT**.

Ce lot ne cherche pas l ecrivain de la naissance — sept lots l ont instruit (5.15 a 5.21) et la
question est close. Il construit un REPLI : une table `(slot, tete) -> archetype` batie sur les
images-cles de TOUT le film, consultee AU POINT DE REJET, qui lie une entite nee en milieu de
chunk sur la foi d une image-cle ULTERIEURE. **Le record de naissance reste non lu.** C est dit,
c est date, c est compte.

---

## 1. LA CLE, ET ELLE EST CELLE QUE LE JEU COMPARE

Une seule lecture de Ghidra dans tout le lot, et c est celle-la : `FUN_1406caad8`, la porte que
tout corps de delta franchit.

```
uVar21 = param_2 & 0x3fffffff                            ; le SLOT — 30 bits bas de l eid
si param_2 == 0xffffffff                     -> return 3 ; sentinelle
lVar19 = *(longlong *)(param_1 + 0x20)                   ; base de la table, pas 200
si (fin - base) / 200 <= uVar21              -> return 3 ; slot hors cardinal
si *(uint *)(uVar21 * 200 + lVar19) != param_2 -> return 3   ; <- LA CLE
puVar18 = uVar21 * 200 + lVar19 ; puVar18[1]             ; l ARCHETYPE, en +0x04
```

La table est INDEXEE par le slot et son entree porte l eid **ENTIER** : les deux bits de tete
comptent, et un delta dont la tete ne vaut pas celle de l entree ne rend AUCUN bit de corps.
**La cle est donc le mot de 32 bits lui-meme**, `(slot, tete)`.

**ET LES DEUX BITS DE TETE D UNE IMAGE-CLE SONT CEUX QU UN DELTA DOIT PRESENTER**, parce que
c est la MEME table : l entree de 200 octets que cette porte teste est celle que `FUN_142e2bfd0`
remplit depuis le payload d image-cle (`e[0x00] = R(32)` l eid, `e[0x04] = R(32)` l archetype —
lot 5.20.1, pas de 0xC8 = 200 confirme par `FUN_142e2bb9c`).

Le brief demandait de le DIRE si le tag de 2 bits d un en-tete de delta et les bits 30-31 d un
eid d image-cle n etaient pas le meme champ. **Au sens de la comparaison, ils le sont** — le jeu
les confronte mot a mot. Ce qu ils SIGNIFIENT reste ce que le lot 5.13.1 a etabli (le rang de la
vue chez `FUN_142f2e174`, la generation du datum chez `FUN_1408f1730`), et les deux films temoins
ne les departagent pas : une seule valeur, `1`, du cote des images-cles. La cle, elle, n est pas
ambigue, et ce lot cle dessus.

---

## 2. LA TABLE

`keyframe_anticipe.go` (207 lignes, couche `grammar`). Une passe sur les images-cles de TOUS les
chunks, par la lecture que le monde emprunte deja (`WalkKeyframeWorld` rend `Slot`, `TI`, `Gen` —
le mot de 32 bits decompose) ; aucune seconde lecture d image-cle n est ecrite.

Chaque cle porte la suite DATEE de ses declarations. `ArchetypeApres(id, chunk)` rend la PREMIERE
declaration STRICTEMENT POSTERIEURE au chunk du rejet : **anticiper, c est lire l avenir d un
slot, jamais son passe.** La passe entiere coute **1,8 s** sur `bfecd02b`, sans aucun decodage de
trame.

### Ce que la table pese (`TestTable523`, `bfecd02b`, carte `snowbound`)

| | valeur |
|---|---:|
| declarations d image-cle versees | **12 688** |
| cles `(slot, tete)` distinctes | **1 015** |
| cles portees par PLUS D UN archetype | **0** |
| tetes rencontrees cote image-cle | **`1` seule**, 12 688 fois |

**ZERO CONFLIT** : aucun slot n est reutilise sous la meme tete sur ce film. La datation par
chunk n arbitre donc rien ici — et elle reste, parce qu elle est la garde qui empechera un slot
recycle de rendre l archetype de son occupant PRECEDENT le jour ou un film en portera un.

### Ce que la table couvre

| ou le rejet trouve-t-il son archetype ? | rejets | part |
|---|---:|---:|
| **declare par une image-cle POSTERIEURE** | **17 432** | **74,7 %** |
| declare seulement par un chunk anterieur ou courant | 0 | 0,0 % |
| **aucune image-cle du film, jamais** | **5 893** | **25,3 %** |

Le chiffre reproduit celui du 5.20.2 au rejet pres. Declarant a **+1 chunk : 17 430** ; a +8 : 1 ;
a +13 : 1. Par archetype anticipe : `ti=35` **16 932** (les reapparitions de bipedes), `ti=42`
346, `ti=41` 85, `ti=40` 53, `ti=10` 9, `ti=37` 7.

### Et la tete discrimine

Les en-tetes rejetes portent la tete `1` 22 687 fois, mais aussi `0` (98), `2` (392) et `3`
(148) : **638 en-tetes presentent une tete qu AUCUNE image-cle du film n emploie**, donc un eid
que le jeu ne pourrait pas apparier. Une cle reduite au seul slot ne resoudrait que **19 rejets
de plus** et lierait ces 638-la. Le prix de la cle juste est de 19 liaisons ; ce qu elle ecarte
est 638 lectures prises a une position fausse.
