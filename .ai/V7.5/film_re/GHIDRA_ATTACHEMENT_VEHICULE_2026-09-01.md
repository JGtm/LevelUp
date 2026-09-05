# Attachement/placement d'une arme-tourelle sur un chassis — preuve moteur (Ghidra HTTP)

> Ecrit le 2026-09-01, worktree `LevelUp-wt-vehicules`. RETRO-INGENIERIE PURE, LECTURE SEULE.
> Aucun commit, aucun `.module` ouvert (un autre agent tenait les modules en RAM) : tout via
> Ghidra HTTP `http://127.0.0.1:8089` sur `HaloInfinite.exe` (Steam, PE x64, image_base
> `0x140000000`, 311 104 fonctions, 1 450 919 symboles).
> Repond au retour utilisateur : « la tourelle du Warthog rendue par l'assembleur statique est
> trop grosse et mal placee ». RAFFINE (et refute en partie) `ASSEMBLAGE_ENFANTS_2026-09-01.md`
> §4/§7 (« transformee de marqueur NON necessaire, translation nulle correcte pour TOUS »).

## TL;DR — la reponse

- **Le moteur N'attache PAS l'enfant a translation nulle.** Il resout un MARQUEUR NOMME sur le
  parent (StringId), en obtient une transformee COMPLETE, et pose l'enfant a
  `matrice_marqueur_parent x offset_marqueur_enfant`. Prouve : chaine
  `Object_AttachObjectToMarker` -> `FUN_1407d7a44` -> `FUN_140806968` -> compose `FUN_140474790`.
- **La transformee de marqueur porte une ECHELLE, et la composition la MULTIPLIE.** La structure
  de transformee du pipeline est `{echelle uniforme (1 float), rotation 3x3 (9), translation (3)}`
  = 13 floats / 56 o (= le pas `0x38` du resolveur de marqueur). `FUN_140474790` calcule
  `resultat[0] = a[0] * b[0]` (produit des echelles) et applique cette echelle a la translation.
  Donc si un noeud du parent porte une echelle de repos != 1.0, l'enfant est mis a l'echelle.
- **Le marqueur d'attache EST un NOEUD du squelette du chassis.** Le resolveur teste, pour chaque
  marqueur, si son StringId est un NOEUD (`FUN_1405dd208`) : si oui il prend la matrice
  MODEL-SPACE du noeud (`FUN_1404919b8`), sinon la matrice bakee du marqueur. Ceci CONFIRME
  l'observation « pas de bloc marker-groups, le pivot vit dans les noeuds » d'ASSEMBLAGE_ENFANTS,
  MAIS montre que ce noeud sert AUSSI a la pose de REPOS de l'attache, pas seulement a
  l'animation.
- **Conclusion pour l'assembleur statique** : la translation nulle est un CAS PARTICULIER
  (marqueur/noeud resolvant a l'identite — le Scorpion « co-repere »). Elle n'est PAS universelle.
  Pour un vehicule dont le noeud d'attache de l'arme n'est pas a l'identite (le Warthog decrit
  par l'utilisateur), l'assembleur DOIT composer la transformee model-space du noeud nomme
  (translation + rotation + echelle uniforme) et l'appliquer a l'enfant. « Trop grosse » =
  echelle du noeud ignoree ; « mal placee » = translation/rotation du noeud ignorees.
- **Reserve honnete** : la VALEUR de la transformee du noeud d'attache du `warthog_g` (offset,
  rotation, echelle != 1 ?) est une donnee du `.module`, que je n'ai PAS pu lire (contrainte
  RAM). J'ai prouve le MECANISME (le moteur applique T+R+echelle depuis le noeud) ; je n'ai pas
  mesure la valeur pour le Warthog. Le symptome rapporte est neanmoins exactement coherent avec
  un noeud non-identite.

## 1. Endpoints Ghidra HTTP utilises

`GET /mcp/schema` (liste des ~200 endpoints) · `GET /list_open_programs` · `GET /get_current_program_info`
· `GET /search_strings?search_term=..` · `GET /search_functions?name_pattern=..` ·
`GET /get_xrefs_to?address=0x..` · `GET /decompile_function?address=0x..` ·
`GET /read_memory?address=0x..&length=N` · `POST /disassemble_bytes {start_address,length}`.
Le pont MCP Python est mort : curl HTTP direct uniquement.

## 2. Surface d'API d'attache (chaines, adresses)

Fonctions natives Lua (CamelCase) et HaloScript (snake_case) toutes presentes dans l'EXE :

| chaine | adresse | role |
|---|---|---|
| `Object_AttachObjectToMarker` | `143c53f38` | attache Lua, SANS echelle |
| `Object_AttachScaledObjectToMarker` | `143c53f58` | attache Lua, AVEC echelle (existe -> l'echelle est un parametre de 1re classe) |
| `Object_GetMarkerWorldPosition` | `143c53e40` | position monde d'un marqueur nomme |
| `objects_attach` | `143c2c7f0` | HaloScript : attache enfant->marqueur |
| `objects_physically_attach` | `143c2c868` | attache physique (Havok) |
| `object_at_marker` | `143c2c840` | enfant present au marqueur donne |
| `object_set_scale` | `143c2c8d8` | fixe l'echelle objet (propagee aux enfants) |
| `unit_get_primary_weapon` | `143ca1b38` | arme primaire d'une unite |
| `object_get_turret` / `_count` | `143ca1798` / `143ca17e0` | tourelle(s) d'un objet |
| `vehicle-equipment-turret-parent-component` | `143c99478` | composant ECS : parentage de tourelle |
| `vehicle-auto-turret-{aiming-vector,triggers,target}-component` | `143c994e8`/`143c99558`/`143c995a0` | ECS auto-tourelle |
| `warthog_gunner` / `warthog_gunner_gauss` / `warthog_gunner_rocket` | `143bf8f88`/`143bf9640`/`143bf95f0` | libelles de siege (StringId) |
| `scorpion_gunner` | `143bf9390` | idem |
| `hkQsTransform` | `14379f8e8` | transformee Havok Quaternion+Scale (porte l'echelle) |
| `hkaBoneAttachment` / `boneFromAttachment` | `143982650` / `143982498` | attache Havok a un os (matrice pleine) |

## 3. La chaine d'attache (fonctions, adresses, role)

Enregistrement Lua (factory generique `FUN_141ec0400`, worker C++ passe en 5e arg sur la pile) —
verifie au desassemblage a `140dcc1ce` (nom `Object_AttachObjectToMarker`, worker `142c715f0`) et
`140dcc20a` (nom `Object_AttachScaledObjectToMarker`, worker `142c71608`) :

```
Object_AttachObjectToMarker        worker FUN_142c715f0  -> FUN_1431c1c00 -> FUN_1407d7834 -> FUN_1407d7918
Object_AttachScaledObjectToMarker  worker FUN_142c71608  -> FUN_1407d7918   (avec params echelle param_3,param_4 ; flag=1)
```

`FUN_142c71608` (scaled) : `if (valide) { if (FUN_1431c0ca4(obj,scale)) FUN_1407d7918(obj,marqueur,scale,arg,0,1); }`
— le `1` final est le flag « appliquer l'echelle ». Le non-scaled passe par le meme cUr.

`FUN_1407d7a44` (routine d'attache, appelee par `FUN_1407d7918`) — le coeur lisible :

```c
lVar1 = resolve(child);                    // FUN_140471c88 : handle enfant -> struct
FUN_1407d7c90(parent, marqueur_parent, &out1);   // resout marqueur PARENT -> transformee (out -> local_178)
FUN_1407d7c90(child,  marqueur_child,  &out2);    // resout marqueur ENFANT -> transformee (out -> local_108)
if (marqueur_child valide)
    FUN_140806968(child, out2, tmp, flag_echelle);   // <-- pose + echelle (voir sect. 4)
else
    ... FUN_1407f4790 x2 ; FUN_140806b0c ; FUN_14080682c ...   // idem sans marqueur enfant
FUN_1407d8c60(parent, child, out1[0], arg);        // lie enfant<->parent (liste d'enfants)
*(int*)(lVar1 + 0x14c) = marqueur_parent;          // <-- ENFANT stocke le StringId du marqueur PARENT
*(int*)(lVar1 + 0x34)  = marqueur_child;           // <-- StringId du marqueur ENFANT
```

Le champ `+0x14c` ecrit ici est EXACTEMENT celui que lit `object_at_marker`
(`FUN_1431c16c8` : parcourt les enfants via `+0x28`/`+0x1c` et compare `enfant[+0x14c] == marqueur`).
Double confirmation independante du champ.

## 4. Le calcul de pose + l'ECHELLE — la preuve

`FUN_140806968` (appelee par `FUN_1407d7a44` avec le flag d'echelle en param_4) :

```c
lVar1 = objet();                                  // FUN_1404777f0
uVar2 = FUN_140489064(lVar1, local_58);           // matrice noeud/marqueur du parent (voir sect. 5)
FUN_1405027d8(local_d8, local_58);
FUN_140474790(local_58, param_2 + 0x38, local_a0);  // COMPOSE : mat_marqueur_parent x offset_enfant

if ((flag & 1) != 0 &&
    fabs(objet[+0x74] - 1.0) > 1e-4) {            // si flag scaled ET echelle objet != 1.0
    float s = objet[+0x74];                        // <-- ECHELLE OBJET (champ +0x74)
    local_78 *= s; fStack_74 *= s; local_70 *= s;  // multiplie le VECTEUR POSITION par l'echelle
}
FUN_140474790(param_3, local_58, local_d8);       // nouvelle compose
...
FUN_1406c7bf4(objet, local_68, local_f8, local_e8, 0, 1);  // ecrit la transformee finale de l'enfant
```

Constantes (`read_memory 0x143cd8370`, decodees) : `143cd8374 = 1.0` ; `143cd837c ~= 1.0e-4`
(epsilon) ; `143cd8380 = 0x7fffffff` (masque valeur absolue) ; `143cd8370 = 0.0`. Donc le test
est bien `fabs(echelle - 1.0) > 1e-4`.

`FUN_140474790` (composition de deux transformees `{echelle, R3x3, T}` = 13 floats) :

```c
*param_3      = param_1[0] * param_2[0];   // ECHELLE : produit des deux echelles
param_3[1..9] = R2 * R1;                    // rotation 3x3 composee (sommes de produits)
param_3[10]   = (R2 * T1) * param_1[0] + T2[x];   // translation : rotee, MISE A L'ECHELLE (*param_1[0]), + T2
param_3[11]   = ... * param_1[0] + T2[y];
param_3[12]   = ... * param_1[0] + T2[z];
```

Le fait que `param_3[0] = param_1[0] * param_2[0]` soit un PRODUIT SIMPLE (pas une somme de
produits comme les 9 elements de rotation) prouve que l'element [0] est un SCALAIRE d'echelle
uniforme separe du 3x3 — le type « real_matrix4x3 a echelle en tete » classique du moteur Halo.
L'echelle se propage donc a CHAQUE composition de marqueur/noeud, independamment du chemin
« scaled » gameplay.

Deux sources d'echelle, distinctes :
1. **Echelle de NOEUD / marqueur (donnee du tag)** : portee par la transformee model-space du
   noeud, multipliee a la compose (`FUN_140474790`). Active meme a echelle objet = 1.0.
   -> C'est CELLE qui compte pour un sprite statique.
2. **Echelle d'OBJET runtime (`+0x74`, defaut 1.0)** : appliquee au vecteur position UNIQUEMENT
   si `flag scaled` ET `!= 1.0`. `object_set_scale` (`FUN_1431c1c8c`) l'ecrit et la propage
   RECURSIVEMENT a tous les enfants (`+0x28` premier enfant, `+0x1c` frere suivant).
   -> Non pertinente au spawn par defaut ; c'est un multiplicateur gameplay.

## 5. La resolution du marqueur = matrice model-space du NOEUD

`FUN_1407d7c90` -> `FUN_1407d7ce4` : resout un StringId de marqueur (param_3) sur un objet, sortie
= tableau de transformees (pas `0x38` = 56 o par entree) :

- Recherche des marqueurs correspondants : `FUN_1406d9714(rm, marqueur_id, ...)`.
- Repli si absent : ecrit une transformee IDENTITE — translation 0, et `0x3f800000` (= 1.0) dans
  les slots d'echelle/rotation. La structure porte donc explicitement des champs d'ECHELLE (1.0
  par defaut).
- Boucle finale (le point cle) : pour chaque marqueur,
  `cVar = FUN_1405dd208(objet, StringId, ...)` teste si le StringId est un NOEUD ; si OUI, la
  matrice utilisee est celle du NOEUD (`FUN_1404919b8` -> `local_b8`), sinon la matrice propre du
  marqueur ; puis `FUN_140474790(...)` compose. -> le marqueur d'attache se resout a la matrice
  MODEL-SPACE d'un noeud nomme du squelette.

`FUN_140489064` construit la matrice de l'objet a partir de trois vecteurs 12 o a `+0x1f8`,
`+0x204`, `+0x210` (position + base orientation), en acces double-buffer thread-safe.

## 6. Structure d'objet (champs prouves par decompilation)

| offset | sens | preuve |
|---|---|---|
| `+0x18` | index objet PARENT (lien d'attache) | `FUN_1407d7a44`, `FUN_1407d7918` (`==-1` = non attache) |
| `+0x1c` | index objet FRERE suivant | `FUN_1431c1c8c`, `FUN_1431c16c8` |
| `+0x28` | index objet PREMIER ENFANT | `FUN_1431c1c8c`, `FUN_1431c16c8` |
| `+0x34` | StringId du marqueur de l'ENFANT | `FUN_1407d7a44` |
| `+0x74` | ECHELLE uniforme de l'objet (float, defaut 1.0) | `FUN_1431c1c8c`, `FUN_140806968` |
| `+0x14c` | StringId du marqueur PARENT ou l'objet est attache | `FUN_1407d7a44` (ecriture) + `FUN_1431c16c8` (lecture) |
| `+0x1f8/+0x204/+0x210` | vecteurs de transformee de l'objet | `FUN_140489064` |
| `+0x8e` | type/etat (2 ou 3 dans la branche vehicule/biped) | `FUN_1407d7a44` |

Hierarchie objet -> enfants confirmee par DEUX fonctions independantes (`object_set_scale` et
`object_at_marker`), memes offsets `+0x28`/`+0x1c`.

## 7. Reponse point par point a la question posee

1. **Quelle fonction lit l'attache d'arme d'un vehicule ?** Cote placement/rendu : la chaine
   `Object_Attach[Scaled]ObjectToMarker` -> `FUN_1407d7a44` -> `FUN_140806968` (avec resolution de
   marqueur `FUN_1407d7ce4`). Cote gameplay/ECS : composant
   `vehicle-equipment-turret-parent-component` + `unit_get_primary_weapon` + sieges
   `warthog_gunner*`. Le lien chassis<->arme est par nom (confirme ASSEMBLAGE_ENFANTS) ; le
   POINT d'attache est un StringId de marqueur = noeud du squelette (stocke sur l'enfant a
   `+0x14c` une fois attache).

2. **La pose de l'enfant : marqueur nomme ou origine ? Echelle ou seulement T+R ?** MARQUEUR
   NOMME. La pose est `matrice_model-space_du_noeud_parent x offset_marqueur_enfant`, composee par
   `FUN_140474790`. La composition inclut TRANSLATION + ROTATION (3x3) + ECHELLE uniforme (element
   [0], multiplie). Ce n'est donc PAS l'origine du modele enfant, sauf quand le noeud resout a
   l'identite. L'ECHELLE existe et se propage.

3. **Constante/champ expliquant un facteur d'echelle sur un modele d'arme monte ?** Deux : (a)
   l'echelle uniforme portee par la transformee de noeud/marqueur (element [0] du type
   `{echelle,R,T}`, multipliee a chaque compose) ; (b) l'echelle d'objet runtime `+0x74`
   (defaut 1.0, propagee aux enfants par `object_set_scale`, appliquee a la position si le flag
   « scaled » est mis et l'echelle != 1.0). Constantes de garde : 1.0 `@143cd8374`, epsilon 1e-4
   `@143cd837c`, masque abs `@143cd8380`.

## 8. Ce que ca implique pour l'assembleur top-down statique

- **La translation nulle N'est PAS correcte en general.** Elle l'est quand le noeud d'attache de
  l'enfant resout a (quasi) l'identite : translation ~0, rotation ~identite, echelle ~1. C'est le
  cas mesure du Scorpion (« co-repere »). Ce n'est PAS une regle universelle : le moteur applique
  toujours la transformee du noeud ; le Scorpion ne « marche » que parce que sa valeur est
  ~identite.
- **Pour le Warthog** (symptome « trop grosse et mal placee »), l'assembleur doit :
  1. trouver dans le squelette du chassis le NOEUD dont le StringId est le marqueur d'attache de
     l'arme (candidat le siege/monture, ex. famille `*_gunner` / le noeud portant `warthog_g`) ;
  2. composer sa transformee MODEL-SPACE (remonter la chaine de noeuds depuis la racine, produit
     des transformees locales `{echelle,R,T}` de chaque noeud) ;
  3. appliquer cette transformee a l'enfant : translation (corrige « mal placee ») + rotation +
     ECHELLE uniforme (corrige « trop grosse » si l'echelle de la chaine de noeuds != 1.0).
  Le champ `Translation` deja present dans `RenduAssemblage` (objet_isole.go) est le bon crochet ;
  il faut l'etendre pour porter aussi la rotation et l'echelle uniforme, alimentees depuis le
  noeud nomme.
- **L'echelle d'objet runtime `+0x74` n'est PAS la cause au spawn** (elle vaut 1.0). La cause
  « trop grosse » plausible est l'echelle de la transformee de NOEUD (donnee du tag), ignoree.

## 9. CR honnete — prouve vs suppose

**Prouve (adresse + decompilation) :**
- Le moteur attache un enfant a un MARQUEUR NOMME du parent et compose une transformee complete
  (`FUN_1407d7a44`, `FUN_140806968`, `FUN_140474790`).
- La transformee du pipeline porte une echelle uniforme MULTIPLIEE a la composition
  (`FUN_140474790 : resultat[0] = a[0]*b[0]`).
- Le marqueur se resout a la matrice model-space d'un NOEUD du squelette (`FUN_1407d7ce4` +
  `FUN_1405dd208` + `FUN_1404919b8`).
- Existence d'une API d'attache AVEC echelle distincte de la version sans (`Object_AttachScaled...`
  vs `Object_Attach...`, workers `142c71608` vs `142c715f0`).
- Champs d'objet : parent `+0x18`, frere `+0x1c`, 1er enfant `+0x28`, marqueur enfant `+0x34`,
  echelle `+0x74`, marqueur parent `+0x14c`. Echelle propagee aux enfants (`object_set_scale`).
- Constantes 1.0 / 1e-4 / masque abs (`read_memory`).

**Suppose / non mesurable ici (dit sans le maquiller) :**
- La VALEUR de la transformee du noeud d'attache de `warthog_g` (offset, rotation, echelle != 1 ?)
  : donnee du `.module`, NON lue (contrainte RAM, aucun module ouvert). J'ai prouve que le moteur
  APPLIQUE T+R+echelle depuis ce noeud ; je n'ai pas la valeur numerique pour le Warthog. Le
  symptome utilisateur (« trop grosse et mal placee ») est neanmoins exactement le signe d'un
  noeud non-identite ignore.
- Le NOM exact du noeud/marqueur d'attache de l'arme du Warthog (candidats `warthog_gunner*`,
  `turret_g`) : a extraire du squelette du chassis (lecture de tag, hors Ghidra).
- L'appariement precis « marqueur parent <-> marqueur enfant » pour l'alignement fin
  (`FUN_140806968` compose `mat_parent x offset_enfant`) : le mecanisme est prouve, l'offset exact
  du marqueur enfant du `warthog_g` reste une donnee de module.

## 10. Fonctions clefs (index adresses)

```
FUN_1407d7a44  routine d'attache (ecrit +0x14c / +0x34 ; resout 2 marqueurs ; appelle 140806968)
FUN_1407d7918  coeur d'attache (dispatch scaled/non-scaled)
FUN_1407d7c90 -> FUN_1407d7ce4   resolution marqueur (StringId -> transformee ; noeud si StringId=noeud)
FUN_140806968  pose + echelle : compose mat_marqueur_parent x offset_enfant, *echelle si flag&!=1
FUN_140474790  compose transformee {echelle,R3x3,T} : resultat[0]=a[0]*b[0] (echelle multipliee)
FUN_140489064  matrice noeud/objet depuis +0x1f8/+0x204/+0x210
FUN_1404919b8  matrice model-space d'un noeud (utilisee par la resolution de marqueur)
FUN_1405dd208  teste si un StringId est un NOEUD du squelette
FUN_1406c7bf4  ecrit la transformee finale de l'enfant
FUN_1431c1c8c  worker object_set_scale : ecrit +0x74, recursif sur enfants (+0x28/+0x1c)
FUN_1431c16c8  worker object_at_marker : parcourt enfants, compare +0x14c
FUN_142c715f0  worker Object_AttachObjectToMarker (sans echelle)
FUN_142c71608  worker Object_AttachScaledObjectToMarker (avec echelle)
FUN_141ec0400  factory d'enregistrement Lua (worker C++ en 5e arg)
Constantes @143cd8374=1.0  @143cd837c=1e-4  @143cd8380=0x7fffffff  @143cd8370=0.0
```
