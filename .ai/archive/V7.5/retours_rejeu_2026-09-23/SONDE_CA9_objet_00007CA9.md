# Sonde CA9 — identité de l'objet `00007CA9` (2026-09-23)

Plan : `.ai/V7.5/PLAN_RETOURS_REJEU_2026-09-23.md` §3.0 (décision du 23/09 nuit : « `00007CA9` inconnu
de l'utilisateur, non affiché, recherche de son identité ») et M3.2. Point de départ :
`SONDE_P3_armes_naissance.md` §4 (branche `feat/rr-sondes`).
Légende : **MESURÉ** (sortie chiffrée d'un instrument ou octets relus), **DÉDUIT**, **HYPOTHÈSE**.

## Verdict

**`00007CA9` est l'arme « mains nues » du jeu : le tag `weap` que le script Lua global nomme
`WeaponTags.unarmed`.** Ce n'est ni un objet d'un autre espace d'identifiants, ni une bobine, ni un
emplacement vide codé : c'est un vrai GlobalID de tag d'arme, de forme basse.

## Preuves

Instrument : `apps/go-api/internal/himodule/ca9_unarmed_research_test.go` (`//go:build research`,
lecture seule des modules installés, aucun film ouvert) :
`CA9_DEPLOY=<bibliotheque>/<jeux>/Halo Infinite/deploy go test -tags=research -count=1 -v -run '^TestCA9Unarmed$' ./internal/himodule/`
— PASS en 0,16 s.

1. **Un GlobalID de tag `weap` (MESURÉ).** `0x00007CA9` est une entrée du module
   `any/globals/globals-rtx-new.module` : groupe `weap`, 8 796 octets (le fusil d'assaut
   `0x48C19D2D`, témoin, fait 27 132 octets dans le même module). Absent de `multiplayer`,
   `multiplayer_r1`, `multiplayer_r3` et `common`. Dépendances déclarées : `hlmt` `6adc7aa3`,
   `mode` `df63faaa`, `jmad` `c7052bbd`, `foot` `6dd82702`, `effe` `a09156e8` — et rien d'autre :
   ni `proj`, ni `jpt!`, ni son de tir, ni `wcfg` (le fusil d'assaut en déclare 10), ni `bitm`.
   Dans le corps du tag, la liste des variantes porte `0x00007CA9` suivi de `42C9679F` et
   `9B555AD2`, comme les armes du catalogue (le fusil d'assaut porte aussi `42C9679F`).
2. **Le script Lua le nomme (MESURÉ, octets).** Le tag `hsc*` `A35C6CE9` (module
   `any/globals/common-rtx-new.module`, seul référent de `0x7CA9` sur les cinq modules globaux :
   outil `weapon-sounds -mode qui`) est du Lua compilé. Sa table `WeaponTags` s'écrit dans le pool
   de constantes `clé` puis entier `TAG(<GlobalID>)` (chaîne = type `0x04` + taille u64 gros-boutiste
   + caractères + NUL ; entier = type `0x02` + u64 gros-boutiste). Octets relus :
   `08 'unarmed' 00 | 02 00 00 00 00 00 00 7c a9`. Relecture complète de la table : **38 clés, 38
   entiers qui sont tous une dépendance `weap` déclarée du script** (sur 63). Étalonnage par les
   armes connues : `assault_rifle = 48C19D2D`, `battle_rifle = 2B1824D5`, `commando_rifle =
   FD98554C`, `energy_sword = 4FF3937E`, `hydra = 767DB96D`, `rocket_launcher = 71AB0A2C`,
   `sidearm_pistol = F408190F`, `sniper_rifle = 0A1992BC`, `mutilator = D7915565`…
   (`.ai/REFERENCE_WEAPON_IDS.md`, `weapons/labels.go`). **`unarmed = 0x00007CA9`**, 26e clé
   sur 38, entre `bandit_vip` et `infection_sword`.
3. **Ce que le Lua en fait (MESURÉ sur le vidage des chaînes du 2026-08-30 ; enchaînement DÉDUIT de
   l'ordre des constantes, le bytecode n'est pas décodé)** :
   - tutoriel (Académie) : `CreateAndPlaceUnarmedWeapon` → `Object_CreateFromTag` `WeaponTags`
     `unarmed` → `Unit_GiveWeapon` `WEAPON_ADDITION_METHOD` `PrimaryWeapon`, à côté de
     `Unit_EmptyAmmo`, `Unit_EmptyGrenadeInventory`, `player_disable_weapon_pickup` : le joueur est
     mis « mains nues » en lui DONNANT cet objet ;
   - Forge, traits de joueur (`hsc*` `0FF80E8C`) : `weaponOverrides` … `WeaponTags` `unarmed` …
     `RESPAWN_WEAPON_SLOT` `Primary` `Backpack` : « mains nues » est la valeur d'un emplacement
     d'arme de réapparition ;
   - Forge, dons d'arme (`hsc*` `87878DB2`) : `ForgeUnitIsValidForWeaponGrant` …
     `Unit_CanUseWeaponDefinition` … `WeaponTags` `unarmed`.
4. **Ghidra (lecture seule)** : le binaire connaît la notion — chaîne « Unarmed Weapon Pickup
   Impulse » (`0x143BB8930`, liée par `FUN_14019FFD0`) dans l'arbre de comportement de l'IA, et
   `FUN_1401E99D0` qui interne « unarmed » (`0x1436867A8`) dans `DAT_1450C420C`. Aucun champ de
   tag `globals` ne référence `0x7CA9` (`qui` sur `globals-rtx-new.module` : 0 référent, 78 174
   entrées) : l'identité vient du tag et du Lua, pas du code.
5. **Cohérence avec le film (MESURÉ par P3, relu ici sans rouvrir de film)** : au coup d'envoi,
   i46 porte `0x7CA9` sur 107 bits contre 162 pour une arme à feu — le composant
   `weapon-state-type-info` n'y a pas de liste de chargeurs (DÉDUIT de la longueur ; `consumeWeaponMagazineList`), ce que
   dit aussi l'absence de `wcfg`/`proj` au point 1 ; 0/15 entrée dans i43/i44 (note
   `NOTE_ORIGINE_POSITIONS_2026-09-01.md`) ; les 15 ramassages natifs de classe ARME à t=0,
   un par joueur, avant toute position répliquée, sont la remise de cet objet à chaque bipède.

## Ce qui est corrigé au passage

- **La prémisse « autre espace d'identifiants » est fausse (MESURÉ).** Les GlobalID de forme basse
  sont de vrais GlobalID de tag : `0x7CA9` est dans la même table d'entrées de module que
  `0x48C19D2D`, et son `AssetID` est lui aussi de forme basse (`0x3119`, contre un AssetID haché
  dont les 32 bits bas sont le GlobalID pour les tags de forme haute). Le dépôt lit déjà des ids
  bas dans les films (`vehicle_families.go` : Warthog `0x00002705`, Mongoose `0x000025AA` ;
  `damagetag/labels.tsv` : `00015438`, `0001535B`). **Pour M4b** : l'absence des `weap` de
  véhicules (`00015435`, `0000AA68`, `00015CFA`…) dans les films n'est PAS une question d'espace
  d'identifiants — ils sont à attendre tels quels quand le tir continu sera décodé.
- Première lecture de ce lot, fausse et retirée : « `hlmt` sans modèle de rendu » (vue des seules
  références inline). La table des dépendances déclare un `mode` (`df63faaa`, hors du module
  `globals`). L'apparence de l'objet n'est donc pas jugée ici.

## Non tranché (HYPOTHÈSES, aucune mesure)

- Pourquoi l'objet occupe le 3e emplacement AU COUP D'ENVOI seulement (à la réapparition, P3 lit un
  3e emplacement vide de 17 bits). Hypothèse : remise pendant l'introduction du match (le script
  déclare `stateAMapIntro`, `stateBPlayerIntro`, `stateCTeamIntro`,
  `stateDTransitionToFirstPerson`, `stateEGameplayStart`), puis retrait ; non mesuré.
- Si un joueur peut TENIR `0x7CA9` en arme active dans un match du parc (Forge avec emplacement
  « mains nues », Infection…) : aucune mesure ; le canal `weaponChanges` le dirait.

## Impact sur le plan (proposition — l'utilisateur décide)

1. **M3.2 (armes de naissance)** : `0x7CA9` n'est pas une arme de dotation. Proposition : l'écarter
   de la dotation affichée et des `loadouts` par une règle NOMMÉE (famille « mains nues », constante
   unique, pas un littéral répété), jamais par un filtre anonyme ; l'emplacement reste compté comme
   lu (couverture), pas comme vide.
2. **Ramassages (`pickups`)** : les ramassages de classe ARME à t=0 portant `00007ca9` (15 sur les deux films de la note du 2026-09-01) ne sont
   pas des prises. Proposition : ne pas les publier comme ramassages (ou les classer « remise
   mains nues », compteur dédié) et les sortir de `coverage.pickups.unknownFamilies`, qui cesse
   alors de compter un identifiant NOMMÉ comme inconnu.
3. **Catalogue** : ajouter l'entrée `0x00007CA9` « Mains nues » / « Unarmed » (FR + EN) avec un
   drapeau « non affichée en dotation », pour qu'un joueur réellement mains nues en cours de match
   (point non tranché ci-dessus) soit nommé au lieu d'être anonyme.
4. **M4b** : lire la correction de prémisse ci-dessus (ids bas = GlobalID valides).
5. Aucun redécodage, aucune révision de grammaire : c'est un nommage.

## Découvertes hors périmètre (non traitées)

- Codes internes du Lua contre le catalogue : `hotrod = 2AC9C2FF` (catalogue « Heatwave »),
  `proto_heatwave = 230447B1` (catalogue « Cindershot »). Un nom de code n'est pas un nom de
  vitrine : aucune conclusion, à vérifier par qui tient le catalogue.
- `skewer = 611BBAE4` dans `WeaponTags` (le catalogue porte `0D20C469`) ; `bandit_vip = F8A0DFE4`,
  `gravity_hammer_escharum = F1409BB1`, `ranked_bulldog = CC7FE33F`, `ranked_stalker_rifle =
  ADB78225`, `ranked_heatwave = 5AC6CFB2` : familles nommables par le Lua.
- Une première passe, qui débordait `WeaponTags` sur les tables suivantes du même pool, a relu
  `forge_fusion_coil_mp = E9E7FF79` : l'AUTRE identifiant muet de la note du 2026-09-01 (« une
  arme ordinaire, 4/4 », absente de `weapon_names.toml`) est la bobine de fusion de Forge (MP).
  L'hypothèse « bobine » de l'utilisateur visait donc juste pour `E9E7FF79`, pas pour `7CA9`.
  Même passe : `m247_hmg = 003F5824`, `plasma_turret = 000026B6`, `gatling_mortar = 3D30B955`,
  `scorpion_tail = B2A1D52F`, `detached_chaingun_mp = D3963939` (appartenance de table non vérifiée).
