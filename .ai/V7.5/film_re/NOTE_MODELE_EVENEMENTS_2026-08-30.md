# LE MODELE DE PAQUET EST PERCE : [configuration][liste d'evenements][trame de records]

Date : 2026-08-30, fin de soiree. Lot 1 du plan « percer la trame ». Cette note etablit LA
RECONCILIATION des lots D et E et ses consequences — dont la reouverture de la lunette.

## Le modele M, etabli par deux chaines sans etape commune

Un paquet delta de film s'ecrit :

```
[1 bit  drapeau de configuration]            (toujours 1 sur le corpus)
[liste d'evenements :
   ( 1  [R(7) type]  [3 references gardees]  [charge du type] )*  0 ]
[trame de records ECS, jusqu'a la fin du paquet]
```

La grammaire de la liste est EXACTEMENT celle prouvee au desassemblage par le lot E
(FUN_14076a1c4 + FUN_14080a9d4) : continuation, R(7) type < 123, trois references
[R(1) porte ; si 1 : R(w) index + R(2) generation] avec w par domaine (vtable+0x58), puis
la charge utile du type (vtable+0x68). LE POINT QUE TOUT LE MONDE AVAIT MANQUE : cette
liste vit APRES le bit de configuration et AVANT la trame de records — les deux lots
regardaient le meme flux avec un decalage d'un bit, et chacun voyait sa moitie.

**Chaine 1 (arithmetique, corpus entier)** : un evenement en tete de paquet donne
`octet0 = 0xC0 | (type >> 1)` et `bit8 = type & 1`. La table tombe juste sur TOUTES les
familles observees ; une liste vide donne `octet0 ∈ 0x80..0xBF` (bit 1 = 0) — 0xA0/0x80/
0x89 et leur cadrage k=2 prouve par l'oracle Rosette.

**Chaine 2 (decodage de bout en bout, 2 films)** : la famille 0xCA decodee ENTIEREMENT avec
des largeurs 100 % sourcees de l'exe (type 21 : refs domaines 4/8/7 -> R(9)/R(13)/R(13),
charge R(2)) — voir le verdict ci-dessous.

## La table famille -> evenement (annexe A du lot E appliquee)

| Octet | Volume corpus | Type(s) = bits 2..8 | Nom |
|---|---|---|---|
| `0xA0`, `0x80`, `0x89`, `0x88`… | ~35 M | — (bit 1 = 0) | liste vide : trame de records pure |
| `0xC0` | 983 883 | 0 / 1 | **damage_aftermath / damage_section_response** — LES DEGATS |
| `0xC2` | 458 938 | 4 / 5 | item_detonate_countdown / **projectile_detonate** |
| `0xC3` | 245 358 | 6 / 7 | projectile_impact_effect / projectile_object_impact_effect |
| `0xC7` | 1 023 286 | 14 / 15 | PlayEffectOnObject / Script |
| `0xCA` | 399 988 | 20 / **21** | incident / **unit_zoom — LA LUNETTE** (mesure : 100 % type 21 sur 2 films) |
| `0xD2` | 2 535 816 | **36** | **action_weapon_fire** — le record de tir/degat (bit8=0 constant mesure) |
| `0xD3` | 528 262 | 38 / 39 | **weapon_reload / biped_throw_initiate** (grenades) — le « gisement » du plan |
| `0xE5` | 195 824 | 74 / 75 | AISetMotorProgram / AIDialog |
| `0xE6` | 195 107 | 76 / 77 | Dialogue2D / DebugSendCameraPosition |
| `0xE9` | 922 724 | 82 / 83 | PlayerGameEventSmall / TeamGameEvent |
| `0xF3` | 31 228 | 102/103 | NetworkedActionRequest / EquipmentSpawnedObject |

(bit8 departage les deux types d'un octet ; a mesurer famille par famille — fait pour 0xCA
et 0xD2.)

## Le verdict 0xCA (unit_zoom) — bout en bout, deux films

| mesure | 000d5950 (12 chunks) | 00502e52 (12 chunks) |
|---|---|---|
| type lu = 21 | **97 / 97** | **86 / 86** |
| ref0 (l'unite, domaine 4) presente | 97 / 97 (17 index distincts) | 86 / 86 (22 index) |
| charge R(2) = niveau + 1 | **1 x50 · 0 x47** (50 entrees / 47 sorties) | **1 x48 · 0 x38** |
| listes multiples (2e evenement) | 15 (types 38, 36, 5, 82, 1) | 16 (types 38, 0, 36, 6, 5) |
| trame apres l'evenement : fermeture | 37,8 % (temoin 0xA0 : 36,3 %) | 20,0 % (seuil 18 %) |
| masques 1..7 des deltas aboutis | **99,4 %** (n=168) | **99,3 %** (n=136) |
| verdict M3 (ecrit avant mesure) | **TENU** | **TENU** |

Les charges ne prennent QUE les valeurs 0 et 1 (= niveaux −1 et 0) : **des paires
entree/sortie de lunette**, quasi equilibrees — la semantique attendue, mesuree.

## Consequences

1. **LA LUNETTE EST DANS LA BOBINE.** La conclusion « aucun evenement de zoom »
   (phases 3-10 du chantier visee, negatif « triple-verrouille ») est REFUTEE : les trois
   chaines du negatif partageaient le meme decalage d'un bit (le bit de configuration
   ignore : type lu = octet & 0x7F au lieu de bits 2..8 ; octets attendus 0x95/0xA4 au lieu
   de 0xCA/0xD2). C'est exactement le piege que la methode du plan citait : « deux chaines
   concordantes compatibles avec une explication plus simple que personne n'avait
   cherchee ». ~400 000 evenements unit_zoom sur le corpus. RESTE pour le produit : associer
   ref0 (index domaine 4) aux joueurs, et croiser avec la verite terrain du chantier visee.
2. **0xD2 = action_weapon_fire (type 36), pas PlayerGameEventSmall.** L'affirmation
   inverse du lot E (E7, « PROUVE cote binaire ») portait sur le cadrage errone. La charge
   du type 36 (lecteur FUN_14080C1F8, variable, refs var-int internes) est la cible pour la
   visee complete et la victime — fire_events.go y lit deja des morceaux stables aux
   offsets 36..142, qui se relisent maintenant comme [3 refs de l'evenement + debut de
   charge].
3. **Le « gisement 0xD3 » = recharges d'arme (38) + amorces de lancer de grenade (39)** —
   moins glorieux que des kills, mais deux signaux produit neufs (economie de munitions,
   grenades par joueur) si la charge se decode (type 38 : 8 octets, lecteur 0x1407f0ff8).
4. **0xC0 = damage_aftermath / damage_section_response (983 k)** : le VRAI gisement de
   degats par coup, a instruire.
5. Les « k gagnants » des balayages etaient la longueur MODALE de l'evenement de tete
   (2 + 7 + refs + charge, variable par paquet) — le mystere de l'« en-tete par famille »
   est dissous : il n'y a pas d'en-tete, il y a un evenement.
6. Les recensements passes par premier octet restent des mesures valides du COUPLE
   (type>>1) ; leurs etiquettes se corrigent par la table ci-dessus.

## Ce qui reste ouvert

- La charge des types a grammaire variable (36 en tete) : largeurs R(n) sur pile a
  resoudre (Ghidra, lot E incertitude n4) avant de decoder tir par tir.
- bit8 par famille sur le corpus entier (0/1 par type) — une passe.
- Le pont index domaine 4 -> joueur (ref0 de unit_zoom) : mesurable par correlation avec
  les fils de vie ; gate produit final = verite terrain lunette du chantier visee.
- Les listes multiples (15/16 % des 0xCA) : decoder l'evenement suivant exige sa charge.
