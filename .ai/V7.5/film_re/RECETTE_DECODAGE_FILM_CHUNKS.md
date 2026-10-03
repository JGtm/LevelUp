# RECETTE — Décodage des films Halo Infinite (chunks → world)

> **LA TABLE DE REFERENCE EST `apps/go-api/internal/analysis/filmdec/testdata/ecs_table.tsv`.**
> Archetype x composant, statut de portage, niveau, source `fichier:ligne`, adresse du deser :
> tout cela se lit LA, tenu par les garde-rails G1-G3 (`filmdec/ecs_table_guard_test.go`).
> Cette recette porte la METHODE et les grammaires ; les tables qu’elle cite sont des
> illustrations datees, la table `.tsv` fait foi sur les statuts.

> Consolidation de la RE du décodeur de replication ECS. Objectif : reconstruire le
> « world » (entités + positions) d'un film **hors-ligne**, à partir des chunks.
> Dernière mise à jour : 2026-07-01 (extraction live de la table ECS via Cheat Engine).

---

## 0. Le pipeline en 6 étapes (résumé)

```
film 000d5950/  ──►  chunks zlib  ──►  registre + records ECS  ──►  world (slot→archétype + valeurs)
```

1. **Décompresser** chaque `chunk_XX.bin` (zlib, magic 0x78).
2. **Parser le registre** (`chunk_00`) : liste ordonnée des composants par archétype + **niveau de précision par composant** (le champ `flags`).
3. **Découper les frames** : header 16 octets `[u16 type][u16 ?][u32 size][u64 timestamp]` + payload. `type==0` = frame delta, `type==2` = keyframe.
4. **Boucle de records** sur chaque payload (bit-stream MSB-first) : header type + id, puis NEW / DEL / DELTA.
5. **Désérialiser chaque composant présent** via le deser propre au composant (largeurs quantifiées, cf. table universelle §5).
6. **Maintenir le world** : slot(id)→archétype, appliquer les deltas, déquantifier les positions (§7).

Le mur historique = étape 4/5 sur le **keyframe** (records NEW full-state) : chaque NEW consomme un
default-state par archétype dont la grammaire vit dans les desers de composant du **build du jeu**
(pas dans le film). §5 le résout définitivement.

---

## 1. Structure des fichiers

| Élément | Contenu |
|---|---|
| `data/cache/film_chunks/<matchid>/chunk_00.bin` | **Registre** ECS (zlib → ~1.97 Mo) : schéma des archétypes |
| `chunk_02.bin` | **Keyframe** (type-2) : état complet initial, 100% records NEW |
| `chunk_01.bin`, `chunk_03..26.bin` | **Frames** (type-0) : records DELTA (+ NEW/DEL au spawn/despawn) |
| `film_manifests/*.json` | juste les URLs/types de chunks (métadonnée, pas de data ECS) |

Header de frame (dans un chunk décompressé, boucle sur `off`) :
`type=u16@off`, `size=u32@off+4`, `timestamp_us=u64@off+8`, payload=`[off+16 : off+16+size]`, avancer `off += 16+size`.

BitReader : **MSB-first, big-endian**. Chaque lecture avance un compteur de bits. (Go : `internal/analysis/filmdec/BitReader`.)

---

## 2. Registre (chunk_00) — schéma + précision  ✅ RÉSOLU (PISTE 1)

Le registre inflaté = suite de **blocs d'archétype** de taille fixe.
- bloc = `archetypeBlockSlots(64) × slot(260 o)` = 16640 o ; **174 archétypes** valides.
- slot (260 o) = `[u32 kind LE][u32 flags LE][name ASCII NUL-paddé]`.
- Les slots à nom non-vide en tête = la **liste ordonnée des composants** de l'archétype (index i = bit du masque de présence).

**`flags` = niveau de précision (L) du composant** (0..4). Largeur d'axe d'un vec3 quantifié = **`6 + L`**
(formule `FUN_140be9b88` : `width=min(26, bitLen(ceil(axisRange/(2·step(L)))))`, `step(L)=2^(16-L)/120`).
Ex biped : i0 position L0→6/6/6, i1 vélocité L1→7, i11 dead-state L3, i21 visée L4.

Go : `internal/analysis/filmdec/registry.go` (`Archetype.Flags[]`, `Level(i)`).

---

## 3. Boucle de records (`FUN_1406cd128`)  ✅ RÉSOLU

**Identique** pour type-0 (frames) et type-2 (keyframe). Par record :

1. (optionnel `HasExtraFields` : `R(32)` préfixe — **absent** sur 000d5950)
2. **type** : `R(1)` ; si 1 → DELTA (type 3) ; si 0 → `R(2)` → type ∈ {0=FIN, 1=NEW, 2=?, 3=DELTA}
3. **id** (`FUN_1406d3140`) : slot = `id & 0x3fffffff`, bits 30-31 = tag génération
4. dispatch (`FUN_1406cbaa0`) :
   - **NEW** (type 1) → §4
   - **DEL** (type 2) → `R(32)` inconditionnel, puis unbind du slot
   - **DELTA** (type 3) → masque de présence + desers des composants présents (PAS de default-state)

`type==0` (lead 0 + `00`) = fin de frame. Go : `frame_records.go`.

---

## 4. Record NEW — default-state + boucle composants (`FUN_1408f1aa4`)  ✅ MODÈLE COMPLET

```
R(6) = index d'archétype  ──►  descripteur runtime (vtable)
 ├─ vtable[0x60](reader)         = deser default-state  (biped: FUN_140f44c38 = representation/identité ~120-160 b)
 ├─ vtable[0x88] / [0x30]        = post-process, 0 bit
 └─ FUN_14076cb60(descripteur, reader) = BOUCLE DE COMPOSANTS
      ├─ masque de présence sparse (FUN_1406d7610) : R(1) gate + R(3) count + count×R(6) index
      │    → OU'é avec le DEFAULT-MASK (vtable[0xa0]) : les composants du default-mask sont TOUJOURS lus
      └─ pour chaque bit du masque effectif : deser du composant via son vtable[0x28]
```

- **DEFAULT-MASK biped = {i0}** (position). Getter `vtable[0xa0]=FUN_14076ca20` construit littéralement le
  masque `{i0}` (désassemblé live : zéro un local, met le bit 0). Donc au spawn, **i0 (position) est TOUJOURS lu**
  même s'il est absent du masque de présence sparse (qui, lui, liste i5,i7,i8…).
- Le default-state (vtable[0x60]) ne lit PAS la position : il lit l'identité (representation-name, variant,
  ids, flags). La position vient du composant i0 dans la boucle, via le default-mask.

**Conséquence décodeur** : pour un record NEW biped, après le default-state, la boucle de composants doit
lire i0 **inconditionnellement** (default-mask) + les composants du masque de présence.

---

## 5. LA TABLE UNIVERSELLE — registre ECS runtime  ✅ RÉSOLU (extraction live 2026-07-01)

Le mapping `typeIndex → descripteur → deser par composant` **N'EST PAS dans le film** : il est construit à
l'init du jeu dans `DAT_144e61d88` (0 statiquement). C'est une **constante du BUILD** (identique pour tous les
films/maps). On le lit **une seule fois** sur un jeu lancé (n'importe quel film en Théâtre) via Cheat Engine.

### Structure (universelle, valable pour tout archétype)

```
REG          = *(DAT_144e61d88)                    ; base = 0x140000000 (statique) + reloc ASLR
descripteur  = *(REG + 8 + typeIndex*8)            ; par typeIndex (35=biped, 40, 2, …)  — 174 valides
composants[] = descripteur + 8                     ; tableau de pointeurs de descripteur de composant
count        = *(int*)(descripteur + 8 + 0x4320)   ; nb de composants (biped=64, #40=48, #2=18)
compDesc[i]  = *(composants + i*8)
deser[i]     = *( *compDesc[i] + 0x28 )             ; vtable[0x28] ; SI == thunk FUN_14076ce9c → prendre vtable[0x30]
```

Le thunk `FUN_14076ce9c` = `jmp vtable[0x30]` → le vrai deser est en `vtable[0x30]` quand `[0x28]` pointe le thunk.

### Procédure de lecture (Cheat Engine MCP, jeu lancé)

1. `base = getAddress("HaloInfinite.exe")` ; `DAT = base + (0x144e61d88 - 0x140000000)`.
2. `REG = readQword(DAT)` (doit être ≠ 0 = ECS peuplé).
3. Pour chaque `typeIndex` voulu : suivre la structure ci-dessus.
4. Reconvertir une adresse runtime en adresse Ghidra statique : `static = runtime - base + 0x140000000`.
5. Décompiler chaque `deser[i]` dans Ghidra (statique) et le porter en Go.

**Validation** : biped vtable[0x60] live = `0x140F44C38` = `FUN_140f44c38` connu (= mon port). Mapping confirmé.

---

## 6. Table des desers — archétype BIPED #35 (64 composants)  ✅ EXTRAITE LIVE

Adresses **statiques** (Ghidra). Composants de même nom partagent le même deser (ex 4 slots d'arme).

| i | composant | deser (statique) |
|--|--|--|
| 0 | object-position-dynamic-precision | **FUN_1406cfe44** (= deser Go actuel ✓) |
| 1 | object-translational-velocity-dyn-precision | FUN_14076d45c |
| 2 | object-forward-and-up | FUN_14076e278 |
| 3 | object-angular-velocity | **FUN_140d70998** (⚠ mon mapping Go était FUN_140d87740 = FAUX) |
| 4 | object-body-vitality | FUN_140fb8978 (R(8)+3×R(1)) |
| 5 | object-shield-vitality | FUN_140d50cbc |
| 6 | object-region-state | FUN_140e1bfa0 |
| 7 | object-damage-sections | FUN_142f03c80 |
| 8 | object-constraint | FUN_142f039cc |
| 9 | object-multiplayer-properties | FUN_140f53308 |
| 10 | object-parent-state | FUN_140c1e4d0 |
| 11 | object-dead-state | FUN_140c1dce0 |
| 12 | object-scale | FUN_1407dc6e4 |
| 13 | object-maximum-vitalities | FUN_1407ee054 |
| 14 | object-dissolver | FUN_140dd9f9c |
| 15 | object-low-frequency | FUN_1407ef088 |
| 16 | object-physics-flags | FUN_1407ee070 |
| 17 | object-frame-configuration | FUN_1407f0534 |
| 18 | unit-control | FUN_141017084 |
| 19 | unit-actor-control | FUN_1408f0778 |
| 20 | unit-actor-state | FUN_14058bcf4 |
| 21 | unit-desired-aiming-vector | FUN_14076df7c |
| 22 | unit-grenade-counts | FUN_140f0de00 |
| 23 | unit-malleable-property | FUN_140f68cc0 |
| 24 | unit-low-frequency | FUN_140f72e48 |
| 25 | unit-command-tick | FUN_1406cfb28 |
| 26 | unit-equipment | FUN_1409685d8 |
| 27 | unit-stun | FUN_142ed75fc |
| 28 | unit-active-camo-state | FUN_142ed3ae0 |
| 29 | unit-crouch | FUN_142ed42a8 |
| 30/33/36/39 | weapon-state-ammo | FUN_140ea1018 |
| 31/34/37/40 | weapon-state-rounds-inventory | FUN_140fe4e88 |
| 32/35/38/41 | weapon-state-overheated | FUN_142f04c6c |
| 42 | biped-desired-weapon-set | FUN_14109d298 |
| 43-46 | weapon-state-type-info (arme tenue) | FUN_1407f06bc |
| 47 | biped-desired-grenade-set | FUN_140c6a628 |
| 48 | biped-desired-ability-set | FUN_1410f8fcc |
| 49 | biped-control-context | FUN_14107166c |
| 50 | biped-map-editor-flag | FUN_142f02854 |
| 51 | biped-emp-timer | FUN_142f02830 |
| 52 | biped-low-frequency-data | FUN_140fc91c8 |
| 53 | biped-malleable-property | FUN_140ff6764 |
| 54 | biped-mobility-action | FUN_1408f0264 |
| 55 | biped-posture-physics | FUN_142f0293c |
| 56 | biped-spartan-ability-energy | FUN_140fc1410 |
| 57 | biped-spartan-ability | FUN_142f02810 |
| 58 | biped-spartan-ability-malleable-property | FUN_140fea4c0 |
| 59 | biped-spartan-ability-non-predicted-state | FUN_142f02994 |
| 60 | simulation-state | FUN_142f02434 |
| 61 | simulation-state-playback | FUN_142f02454 |
| 62 | biped-slide | FUN_142f02978 |
| 63 | biped-action | FUN_142f027f4 |

Deser default-state (vtable[0x60]) = **FUN_140f44c38** (identité/representation, ~120-160 b, version-gated).

---

## 7. Déquantification (positions monde)  ✅ RÉSOLU

- Position i0 = vec3 quantifié absolu (chemin `FUN_14076e524`) : `R(1)` gate + `R(idxW)` + 3×`R(axisW)`.
  `idxW=1`, `axisW=6` (registre flags L0).
- Plage monde = `QuantRangeWorld100` = ±100 par axe (`DAT_143b8c6f0` precision-2).
- Déquant : `world = min + step·(q + 0.5)`, `step = (max-min)/2^axisW`. Go : `position_capture.go::dequantWorldAxis`.
- 3 encodages i0 : keep-baseline (`bUsePred=1`, position INCHANGÉE — pas une coord), absolu quantifié, delta prédit.

---

## 8. État du décodeur Go + prochaines étapes  (MAJ 2026-07-03)

**Fait** : chunks zlib, registre+flags, boucle de records, ~65 desers de composant, déquant plage-map réelle,
capture position i0, **inférence de chaînes récursive** (`frame_chain_infer.go`) + **resync validé** (`validatedResync`)
→ **les 8 bipèdes sont ATTEINTS** (desync 27%→18.5%), rendu trajectoires PNG (`cmd/tmp_traj INFER=2 INFERRESYNC=1`).

**Le mur restant n'est PAS l'atteinte, c'est le desync séquentiel** (positions bit-exactes) :
1. **Porter la liste FINIE de composants** qui bloquent le décodage séquentiel (mesurée `tmp_traj INFER=2`, desync-cause) :
   `ti=37 i29/i30` (equipment-command-tick), `ti=35 i55/i57/i60/i63` (biped posture/ability/sim/action),
   `ti=14 i1`, `ti=5 i7/i22/i24`. Chaque port réduit le desync → le décodage séquentiel atteint plus de bipèdes
   à leur VRAI alignement (donc VRAIE position). Méthode §5 (descripteur runtime → vtable[0x28]/[0x30] → décompiler → porter).
2. Chaînes de transitoires restantes (`inférence échouée`, ~3700) : inférence récursive plus profonde OU keyframe complet.

**Reconstruction de trajectoire (modèle position, IMPORTANT)** : la position d'un joueur est **absolue au spawn/respawn**
puis **progresse en DELTAS** (le composant i0 a 3 encodages : keep-baseline = non réémis, absolu quantifié = spawn,
delta prédit). ⇒ NE PAS relier les absolus par des lignes (ils sautent entre points de spawn dispersés = faux « bruit »).
Reconstruire = **ancrer à chaque absolu, accumuler les deltas suivants, SEGMENTER à chaque respawn** (`cmd/tmp_traj` mode
delta-accum). Le POV (dense en deltas) donne une trajectoire propre ; les distants sont sparse en deltas (atteinte limitée),
pas absents.

### ⚠ MUR PROUVÉ (2026-07-03) : validité STRUCTURELLE ≠ correction de POSITION (offline)

Un delta biped décode « proprement » (structurellement) à **plusieurs alignements de bits** ; seul le VRAI alignement
séquentiel (fixé par la chaîne de records depuis le début de frame) donne la VRAIE position. Le scan exhaustif
(`ScanFrameTargets`, `cmd/tmp_bipedharvest`) ATTEINT la queue mais, aux faux alignements, décode des positions FAUSSES.
⇒ pas de raccourci par « scan de coordonnées » : le film est bit-packé, la position n'est pas repérable indépendamment du
décodage séquentiel complet. **Diagnostic `cmd/tmp_bipedreach`** : la donnée des affamés EST dans le flux (~680-920
deltas-position/slot), non atteinte car coupée par desync amont — pas absente (réfute « per-client »).

**Méthode pour tout autre archétype/composant** : §5 (lire le descripteur runtime, prendre vtable[0x28]/[0x30],
décompiler statiquement, porter). Le mapping est une constante de build, à lire une seule fois.

### ⚠ Le VRAI mur restant = la VALIDATION (pas la RE)

La table §5 donne la bonne *fonction* deser par composant, mais :
- Les **largeurs exactes** de certains desers viennent d'args passés en registre (non tracés par Ghidra),
  et diffèrent par appelant (ex FUN_14076d528 : dir=arg7, magnitude=arg6 ; i3 biped passe (…,8,0x13)).
- **La métrique offline `clean-frame` est dominée par des FAUX-PROPRES** : un deser PLUS correct peut FAIRE
  BAISSER le nombre de frames « propres » (il casse des réalignements fortuits). Test vécu : router i3 vers la
  bonne fonction a fait chuter clean-frames 18060→9701. **Ne jamais valider un fix de deser au clean-frame count.**
- **Validation fiable = oracle live** : CE breakpoint sur le deser (`FUN_140dXXXXX`), lire `reader+0x2c` (position
  bit) avant/après → largeur exacte consommée. OU charger le film cible en Théâtre et comparer les positions
  décodées de l'entity store avec le décodage offline. Sans ça, chaque port de largeur reste une hypothèse.

Recommandation : une prochaine session CE avec **breakpoints sur les desers** (pas juste lecture de structure)
donne les largeurs exactes + valide bit-à-bit → débloque le portage fiable → keyframe → 8 trajectoires.
