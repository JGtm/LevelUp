# Note 5.13 — Le rang de la vue, et ce que la « vue 3 » est vraiment (2026-09-22)

> Lot 5.13.1, branche `feat/decfilm-63`. Le plan (`.ai/PLAN_DECODEUR_FILM_2026-09-13.md`,
> section « Post-chantier — lot 5.13 ») fait foi sur l avancement, cette note sur la GRAMMAIRE.
> Elle CORRIGE deux lectures du journal RE et de la passation 5.11.

## 1. Les deux bits de tete d un identifiant d image-cle sont le RANG DE LA VUE

L encodeur de la liste de REFERENCE d une vue de replication est `FUN_142f2e174`
(`replication_entity_manager_view.cpp`), slot `+0x10` de la vtable de vue `0x1436a87e0`. Il
parcourt la table d entites DE SA VUE (`vue+0x38` a `vue+0x40`, pas de `0xa0`, bitmap de presence
`vue+0x58`, masque d interet par vue `1 << vue[0x14]`) et ecrit, par entite retenue, UN mot de
32 bits :

```
*mot = *(int *)(vue + 8) << 0x1e | *mot & <masque> | slot & 0x1fff | <genre>
FUN_140bbd808(mot, priorite)      // *mot = *mot & 0xff801fff | (priorite & 0x3ff) << 0xd
```

soit `[rang:2 @30][genre:2 @23][priorite:10 @13][slot:13 @0]` — l `id:32` de l en-tete
`[id:32][field:26][ti:6]` que `keyframe_world.go` lit deja.

**Les deux bits de tete viennent de `vue + 8`, pas de l entite**, et c est verifie sur
l instruction : `142f2e2ec MOV ECX, dword ptr [RDI + 0x8]` puis `142f2e304 SHL ECX, 0x1e`, ou
`RDI` est `param_1`, c est-a-dire la vue (meme registre que `vue+0x38`, `vue+0x58`, `vue+0x14`).
Les trois sites de genre (`142f2e304`, `142f2e38a`, `142f2e440`) lisent le meme champ.

**Et `vue + 8` est le RANG de la vue** : le registraire `FUN_1409c9860(conteneur, rang, vue)`
l y ecrit — `*(int *)(param_3 + 1) = param_2` — en rangeant la vue dans le tableau que
`FUN_142987460` parcourt. `FUN_141f855b4` l appelle pour les rangs 0, 1 et 2.

Le journal RE du lot G (2026-08-27, `WALK_PORT_NOTES.md` §1) nommait ce champ `gen` sans avoir
decompile l instruction. Ce n est ni une generation, ni un identifiant d entite.

## 2. Ce que le film porte, mesure

| film | paquets d image-cle | records par paquet | rangs distincts |
|---|---:|---|---|
| `dad793c7` | 5 | 123 a 186 | **{1}** |
| `bfecd02b` | 60 | 424 a 482 | **{1}** |

`TestImageCle513Vues`. L image-cle d un film est donc MONO-RANG : elle enumere la table d UNE
vue, celle du gestionnaire d entites. Et la vue que la marche hors ligne parcourt en PREMIER est
celle qui rend les records (`TestVues513EspaceDeNoms`) : `dad793c7` 5 628 records dont tag 1 sur
**5 628 / 5 628** ; `bfecd02b` **157 250 / 157 554** (99,81 %).

« Toutes les liaisons d image-cle vont en vue 0 » n etait pas une limite du portage — c est ce
que le film porte. Le port (`World.BindImageCle`) le LIT desormais au lieu de l attribuer
d office : premier rang rencontre -> la vue de la marche ; tout autre rang -> vue INCONNUE, qui
ne rejette rien.

## 3. Le flux DELTA n ecrit PAS le meme champ — et le confondre coute des records

`FUN_142f30610` ecrit l identifiant que la table de la vue porte
(`uVar3 = *(uint *)(slot * 0xa0 + 8 + vue[0x38])`, passe a `FUN_142f2c754`), et cet identifiant
est pose par `FUN_1408f1730` a `*(byte *)(datum + 1) << 0x1e | slot` : un champ du DATUM, par
entite. Exiger l egalite des deux dans la garde de table de vue coute **21 records `ti=35`** sur
`bfecd02b` (114 458 -> 114 437). `World.VuePossede` compare donc le SLOT, et le dit sur place.

## 4. « L entite 7140 generation 2 de la vue 3 » n existe pas

`FUN_142987460` appelle `vtable[0x40]` sur ses trois vues. Les trois vtables (`/read_memory`)
n ont PAS la meme fonction a ce slot :

| vtable | `+0x40` | grammaire de la boucle de records |
|---|---|---|
| `0x1436a8700` | `FUN_14076a1c4` | si `vue[0x11]` : **zero bit** ; sinon boucle `R(1)` (0 = fin) puis UN corps (`FUN_14080a9d4`). **Rend toujours zero record** (`*param_6 = 0`) |
| `0x1436a87e0` | `FUN_1406cd128` | le GESTIONNAIRE D ENTITES : `[R(32) film]`, `prefixe R(1)`, `si 0 -> type R(2)`, `idLow + tag 2`, corps selon le type. **La seule grammaire que la marche porte** |
| `0x1436a8770` | `FUN_1406cf548` | `[prologue FUN_142f2539c si drapeau]` puis boucle `R(1)` (0 = fin), `kind R(2)`, trois handlers (`FUN_1406d0388` / `FUN_142f29b38` / `FUN_142f29e54`), `kind == 3` = zero bit |

Les deux « en-tetes de record rejetes » du pied de trame (lot 5.11.7) ne sont donc pas des
en-tetes du gestionnaire d entites : ce sont les flux des DEUX AUTRES classes de vue, lus avec la
grammaire du gestionnaire. `slot 7136 tag 0` et `slot 7140 tag 2` ne designent AUCUNE entite — ce
sont `R(1)` + `kind R(2)` + du corps d une autre grammaire, decoupes en `[prefixe][idLow][tag]`,
et les bits 27 / 30 qui basculaient au decollage sont des bits DE CE CORPS.

Controle sur le film : sur `dad793c7` les vues de rang 1 et 2 ne rendent que **13 records**, TOUS
de type DEL (slots 0, 260, 261, 262) — jamais un NEW, donc jamais un archetype ; et le slot 7140
n apparait dans aucun record d image-cle (elles s arretent au slot 1 345).

**La piste « nommer l entite de la vue 3 » est fermee par la grammaire, pas par une absence** :
l objet a nommer n existe pas.

## 5. Ce que cela ouvre — la route vers 100 % de paquets fermes

Porter les DEUX autres boucles de records (`FUN_14076a1c4`, `FUN_1406cf548`) est la route vers la
fermeture complete des paquets : les 13 records DEL fantomes de `dad793c7` et ses 18
debordements viennent tous de la grammaire du gestionnaire appliquee a ces deux flux. C est un lot
a soi seul (trois handlers a mesurer pour `FUN_1406cf548`, un pour `FUN_14076a1c4`), consigne au
§4 du plan.
