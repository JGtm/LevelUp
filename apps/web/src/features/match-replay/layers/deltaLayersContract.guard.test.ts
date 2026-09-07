/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail (CLAUDE.md n° 6) : LES TYPES ÉCRITS À LA MAIN DES SCHÉMAS 25-27 NE DOIVENT JAMAIS
 * DIVERGER DES STRUCTURES GO QU'ILS DÉCRIVENT.
 *
 * POURQUOI IL EXISTE. `ReplayWeaponChange`, `ReplayEquipmentChange` et `ReplayGroundWeapon`
 * (lib/api/types.ts) sont les SEULS types du document de rejeu qui ne viennent pas du contrat
 * généré : `api/openapi.yaml` n'a pas été régénéré quand les trois calques ont été livrés côté
 * Go, le 2026-08-30. Sans garde-rail, un champ renommé dans le Go passerait inaperçu jusqu'à ce
 * qu'un écran se vide en silence — exactement ce que la génération empêche partout ailleurs.
 *
 * CE QU'IL VÉRIFIE, ET C'EST DÉLIBÉRÉMENT ÉTROIT : les NOMS des champs JSON, les VALEURS des
 * énumérations, la sentinelle de rang, et la présence des trois champs sur le document. Pas les
 * types Go (un `int` et un `float32` se sérialisent tous deux en `number`) — ce n'est pas ce qui
 * casse en silence.
 *
 * IL DISPARAÎT AVEC LES TYPES QU'IL GARDE : le jour où `make openapi-gen && make generate-types`
 * publie ces trois schémas, les déclarations manuelles redeviennent des ré-exports et ce fichier
 * n'a plus d'objet.
 *
 * Même patron que `placementFamily.guard.test.ts` : lire le fichier Go source par `readFileSync`
 * et comparer, plutôt que dupliquer sans lien vérifiable.
 */
import { describe, expect, it } from 'vitest'

import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

import { racineDuDepot } from '../test/featureFiles'
import {
  REPLAY_NO_ABILITY_RANK,
  type ReplayEquipmentChange,
  type ReplayGroundWeapon,
  type ReplayWeaponChange,
} from '@/lib/api/types'

const REPO = racineDuDepot()
const GO = resolve(REPO, 'apps/go-api/internal/analysis/replay')
const FILMDEC = resolve(REPO, 'apps/go-api/internal/analysis/filmdec')

/** Égalité STRICTE de deux types (le double conditionnel différé est ce qui la rend stricte). */
type Equals<A, B> = (<T>() => T extends A ? 1 : 2) extends <T>() => T extends B ? 1 : 2
  ? true
  : false
type Expect<T extends true> = T

/** Les balises `json:` d'une structure Go nommée, dans l'ordre de déclaration. */
function goJSONTags(file: string, struct: string): string[] {
  const src = readFileSync(resolve(GO, file), 'utf8')
  const start = src.indexOf(`type ${struct} struct {`)
  expect(start, `structure ${struct} introuvable dans ${file}`).toBeGreaterThanOrEqual(0)
  const end = src.indexOf('\n}', start)
  const body = src.slice(start, end)
  return [...body.matchAll(/`json:"([A-Za-z0-9]+)(?:,[a-z]+)?"`/g)].map((m) => m[1])
}

/**
 * La valeur littérale d'une constante Go de type chaîne.
 *
 * DEUX ÉCRITURES À COUVRIR, et le Go emploie les deux : typée
 * (`WeaponTaken WeaponChangeKind = "taken"`) et nue (`GroundWeaponEndPickup = "pickup"`).
 * On coupe donc sur le `=` plutôt que d'écrire une expression rationnelle qui doive connaître
 * la forme — ce qui se lit, et ce qui ne casse pas au prochain typage.
 */
function goStringConst(file: string, name: string): string {
  const src = readFileSync(resolve(GO, file), 'utf8')
  const ligne = src.split('\n').find((l) => l.trim().startsWith(`${name} `))
  expect(ligne, `constante ${name} introuvable dans ${file}`).toBeDefined()
  const valeur = (ligne ?? '').split('=').slice(1).join('=').trim()
  expect(valeur.startsWith('"'), `constante ${name} : valeur non littérale`).toBe(true)
  return valeur.slice(1, valeur.lastIndexOf('"'))
}

// --- LES CLÉS DE CHAQUE TYPE, PROUVÉES EXHAUSTIVES PAR LE COMPILATEUR -----------------------
//
// Les trois listes ci-dessous sont confrontées AU TYPE par les assertions `_Cles*` : une clé
// ajoutée côté TS sans être ajoutée ici fait échouer `tsc -b`, et la comparaison au Go qui suit
// fait échouer la CI si le Go ne la porte pas. Les deux verrous sont nécessaires — le premier
// interdit d'oublier, le second interdit d'inventer.

const WEAPON_CHANGE_KEYS = ['t', 'slot', 'kind', 'w', 'from'] as const
type _ClesWeaponChange = Expect<Equals<(typeof WEAPON_CHANGE_KEYS)[number], keyof ReplayWeaponChange>>

// `recovered` et `gap` entrent au schéma 38 (2026-09-03) : la PROVENANCE de l'émission et le
// saut de compteur RÉSIDUEL. Le second est celui qui compte pour le rendu — sous un saut, `from`
// n'est plus une identité (cf. `identityIsUnknown`, placementTeleport.ts).
const EQUIPMENT_CHANGE_KEYS = ['t', 'slot', 'kind', 'r', 'from', 'recovered', 'gap'] as const
type _ClesEquipmentChange = Expect<
  Equals<(typeof EQUIPMENT_CHANGE_KEYS)[number], keyof ReplayEquipmentChange>
>

const GROUND_WEAPON_KEYS = [
  't0',
  't1',
  't1max',
  'x',
  'y',
  'z',
  'w',
  'origin',
  'dropper',
  'end',
  'picker',
] as const
type _ClesGroundWeapon = Expect<
  Equals<(typeof GROUND_WEAPON_KEYS)[number], keyof ReplayGroundWeapon>
>

const GROUND_WEAPON_ENDS = ['pickup', 'seen', 'open'] as const
type _FinsGroundWeapon = Expect<
  Equals<(typeof GROUND_WEAPON_ENDS)[number], ReplayGroundWeapon['end']>
>

const WEAPON_CHANGE_KINDS = ['taken', 'dropped', 'swapped'] as const
type _NaturesWeaponChange = Expect<
  Equals<(typeof WEAPON_CHANGE_KINDS)[number], ReplayWeaponChange['kind']>
>

const EQUIPMENT_CHANGE_KINDS = ['taken', 'spent'] as const
type _NaturesEquipmentChange = Expect<
  Equals<(typeof EQUIPMENT_CHANGE_KINDS)[number], ReplayEquipmentChange['kind']>
>

describe('garde-rail : les types manuels des schémas 25-27 <-> les structures Go', () => {
  it('les assertions de clés tiennent à la compilation', () => {
    const cles: [_ClesWeaponChange, _ClesEquipmentChange, _ClesGroundWeapon] = [true, true, true]
    const natures: [_FinsGroundWeapon, _NaturesWeaponChange, _NaturesEquipmentChange] = [
      true,
      true,
      true,
    ]
    expect(cles.concat(natures)).toEqual([true, true, true, true, true, true])
  })

  it('ReplayWeaponChange porte EXACTEMENT les champs JSON de replay.WeaponChange', () => {
    expect([...WEAPON_CHANGE_KEYS].sort()).toEqual(
      goJSONTags('document_weapon_changes.go', 'WeaponChange').sort(),
    )
  })

  it('ReplayEquipmentChange porte EXACTEMENT les champs JSON de replay.EquipmentChange', () => {
    expect([...EQUIPMENT_CHANGE_KEYS].sort()).toEqual(
      goJSONTags('document_equipment_changes.go', 'EquipmentChange').sort(),
    )
  })

  it('ReplayGroundWeapon porte EXACTEMENT les champs JSON de replay.GroundWeapon', () => {
    expect([...GROUND_WEAPON_KEYS].sort()).toEqual(
      goJSONTags('document_ground_weapon_items.go', 'GroundWeapon').sort(),
    )
  })

  it('les FINS d’affichage sont celles que le Go publie', () => {
    expect(GROUND_WEAPON_ENDS).toEqual([
      goStringConst('document_ground_weapon_items.go', 'GroundWeaponEndPickup'),
      goStringConst('document_ground_weapon_items.go', 'GroundWeaponEndSeen'),
      goStringConst('document_ground_weapon_items.go', 'GroundWeaponEndOpen'),
    ])
  })

  it('les NATURES de changement sont celles que le Go publie', () => {
    expect(WEAPON_CHANGE_KINDS).toEqual([
      goStringConst('document_weapon_changes.go', 'WeaponTaken'),
      goStringConst('document_weapon_changes.go', 'WeaponDropped'),
      goStringConst('document_weapon_changes.go', 'WeaponSwapped'),
    ])
    expect(EQUIPMENT_CHANGE_KINDS).toEqual([
      goStringConst('document_equipment_changes.go', 'EquipmentTaken'),
      goStringConst('document_equipment_changes.go', 'EquipmentSpent'),
    ])
  })

  it('REPLAY_NO_ABILITY_RANK vaut la sentinelle du décodeur', () => {
    // `replay.NoAbilityRank` n'est qu'un alias de `filmdec.AbilitySetNoRank` : c'est la valeur
    // de ce dernier qu'il faut vérifier, sans quoi le garde-rail comparerait un nom à un nom.
    const src = readFileSync(resolve(FILMDEC, 'components_biped_ability.go'), 'utf8')
    const m = src.match(/^const AbilitySetNoRank = (-?\d+)$/m)
    expect(m, 'const AbilitySetNoRank introuvable').not.toBeNull()
    expect(REPLAY_NO_ABILITY_RANK).toBe(Number(m![1]))
    expect(
      readFileSync(resolve(GO, 'document_equipment_changes.go'), 'utf8'),
    ).toContain('const NoAbilityRank = filmdec.AbilitySetNoRank')
  })

  it('le document publie bien les trois calques, sous ces noms-là', () => {
    const doc = readFileSync(resolve(GO, 'document.go'), 'utf8')
    for (const champ of ['weaponChanges', 'equipmentChanges', 'groundWeapons']) {
      expect(doc, `champ ${champ} du ReplayDocument Go`).toContain(`json:"${champ},omitempty"`)
    }
  })
})
