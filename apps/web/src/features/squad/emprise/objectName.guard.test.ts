/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail (CLAUDE.md n° 6, « ≤ 2 copies » ; plan PLAN_SESSIONS_EMPRISE_2026-10-06, D17) : le NOM
 * D'UN OBJET DE L'EMPRISE (bonus par l'i18n d'usage, véhicule par `vehicleFamilyName`, arme par son
 * libellé) vit dans `squad/emprise/objectName.ts` et nulle part ailleurs. Il existait en deux copies
 * (Escouade, Séries temporelles) et la page Sessions en aurait fait une troisième.
 *
 * Empreinte : un fichier de production qui appelle À LA FOIS `equipmentFamilyLabel(` et
 * `vehicleFamilyName(` réécrit ce nommage. Le module canonique et les tests sont exclus.
 */
import { readdirSync, readFileSync } from 'node:fs'
import { join, resolve } from 'node:path'

import { describe, expect, it } from 'vitest'

const FEATURES_ROOT = resolve(process.cwd(), 'src', 'features')
const CANONIQUE = join(FEATURES_ROOT, 'squad', 'emprise', 'objectName.ts')

/** Vrai si le texte réécrit le nommage des objets de l'Emprise. */
function nommeLesObjetsDeLEmprise(text: string): boolean {
  return /\bequipmentFamilyLabel\(/.test(text) && /\bvehicleFamilyName\(/.test(text)
}

function walk(dir: string): string[] {
  const out: string[] = []
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const full = join(dir, entry.name)
    if (entry.isDirectory()) out.push(...walk(full))
    else if (/\.(ts|tsx)$/.test(entry.name) && !/\.test\.(ts|tsx)$/.test(entry.name)) out.push(full)
  }
  return out
}

describe('garde-rail : le nom d’un objet de l’Emprise a une seule source (D17)', () => {
  it('l’empreinte reconnaît l’ancienne copie et ignore un simple appel du helper', () => {
    const copie =
      "if (o.resource === RESOURCE_POWERUP) return equipmentFamilyLabel(o.key, usageText)\n" +
      'if (o.resource === RESOURCE_VEHICLE) return vehicleFamilyName(o.key, o.label, unknownVehicle)'
    expect(nommeLesObjetsDeLEmprise(copie)).toBe(true)
    expect(nommeLesObjetsDeLEmprise('const n = empriseObjectName(o, usageText, unknown)')).toBe(false)
  })

  it('aucun fichier de production hors objectName.ts ne réécrit ce nommage', () => {
    const offenders = walk(FEATURES_ROOT)
      .filter((f) => f !== CANONIQUE)
      .filter((f) => nommeLesObjetsDeLEmprise(readFileSync(f, 'utf8')))
      .map((f) => f.replace(FEATURES_ROOT, 'src/features'))
    expect(offenders, `nommage des objets de l’Emprise recopié : ${offenders.join(', ')} — importer empriseObjectName`).toEqual([])
  })
})
