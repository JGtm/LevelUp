/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail (CLAUDE.md n° 6, patron « helper + garde-rail ») : LA LISTE DES FAMILLES DONT LE
 * LÂCHER SE DESSINE EST ÉCRITE, ELLE N'EST PAS DÉDUITE — et ce test est ce qui l'empêche de
 * diverger de ce dont elle est déduite en pensée.
 *
 * POURQUOI ÉCRITE. `placementDropped.ts` est lu PAR le calque : déduire sa liste de
 * `PLACEMENT_RENDER` créerait un cycle d'import à l'exécution. Le prix de cette contrainte est
 * une seconde énumération ; ce fichier en est la contrepartie, et il rejoue la correspondance
 * DANS LES DEUX SENS.
 *
 * CE QUI CASSE SANS LUI, ET SILENCIEUSEMENT :
 *  - une famille déployable ajoutée au manifeste et dessinée quand elle est posée, mais absente
 *    de la liste : son lâcher resterait invisible sans qu'aucune erreur ne le dise ;
 *  - une famille retirée de la table de rendu et laissée dans la liste : on dessinerait le
 *    lâcher d'un objet qu'on ne sait plus dessiner déployé ;
 *  - un power-up ajouté au titre côté socles (`PAD_EQUIPMENT_FAMILIES`) sans que ce calque le
 *    suive : le socle serait grand et nommé, le lâcher muet.
 *
 * ET LA GARDE DES COULEURS, qui vaut pour tout ce lot : aucune valeur littérale de couleur ne
 * doit entrer dans les fichiers du lâcher — l'encre arrive du thème par le calque appelant.
 */
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

import { PLACEMENT_RENDER, type PlacementKind } from './equipmentPlacementsLayer'
import {
  DROPPED_EQUIPMENT_FAMILIES,
  PLACEMENT_DROPPED_FAMILIES,
} from './placementDropped'
import { PAD_EQUIPMENT_FAMILIES, POWER_PAD_KEYS } from './weaponPadFamilies'

/**
 * La famille par DÉFAUT du serveur. Elle a une règle de rendu (`unnamed`) mais reste HORS de la
 * liste des lâchers : promouvoir en objet de puissance ce qu'on ne sait pas nommer affirmerait
 * un enjeu que rien n'établit. Nommée ici pour que l'exclusion soit une décision lisible.
 */
const DEFAULT_FAMILY = 'other'
const DEFAULT_KIND: PlacementKind = 'unnamed'

/** Les familles à qui la table donne une forme d'objet ACTIF, le défaut mis à part. */
function famillesDeployables(): string[] {
  return Object.entries(PLACEMENT_RENDER)
    .filter(([, kind]) => kind !== null && kind !== DEFAULT_KIND)
    .map(([family]) => family)
    .sort()
}

describe('garde-rail : la liste des lâchers suit la table de rendu', () => {
  it('toute famille DÉPLOYABLE de la table a son lâcher dessiné', () => {
    for (const f of famillesDeployables()) {
      expect(
        DROPPED_EQUIPMENT_FAMILIES,
        `${f} se dessine posée, mais son lâcher serait muet`,
      ).toContain(f)
    }
  })

  it('et réciproquement : aucun équipement de la liste n’est absent de la table', () => {
    for (const f of DROPPED_EQUIPMENT_FAMILIES) {
      expect(
        PLACEMENT_RENDER[f],
        `${f} se dessinerait lâchée sans qu'on sache la dessiner posée`,
      ).toBeTruthy()
    }
  })

  it('la famille par défaut reste dehors : on ne promeut pas ce qu’on ne sait pas nommer', () => {
    expect(PLACEMENT_RENDER[DEFAULT_FAMILY]).toBe(DEFAULT_KIND)
    expect(PLACEMENT_DROPPED_FAMILIES).not.toContain(DEFAULT_FAMILY)
  })

  it('les POWER-UPS viennent du vocabulaire des socles, jamais d’une 3e copie', () => {
    const powerUps = Object.keys(PAD_EQUIPMENT_FAMILIES)
    for (const key of powerUps) {
      expect(PLACEMENT_DROPPED_FAMILIES, `${key} : socle grand, lâcher muet`).toContain(key)
      // La même clé est l'une des « armes de puissance » explicites de l'utilisateur (18/08).
      expect(POWER_PAD_KEYS, `${key} hors de la liste explicite de puissance`).toContain(key)
    }
    expect(PLACEMENT_DROPPED_FAMILIES).toHaveLength(
      DROPPED_EQUIPMENT_FAMILIES.length + powerUps.length,
    )
  })

  it('les familles PORTÉES (`null`) n’entrent jamais dans la liste', () => {
    const portees = Object.entries(PLACEMENT_RENDER)
      .filter(([, kind]) => kind === null)
      .map(([family]) => family)
    expect(portees.length, 'la table ne porte plus aucune famille à null ?').toBeGreaterThan(0)
    for (const f of portees) {
      expect(PLACEMENT_DROPPED_FAMILIES, `${f} est portée, son lâcher ne se dessine pas`).not.toContain(f)
    }
  })
})

describe('garde-rail : aucune couleur écrite dans les fichiers du lâcher', () => {
  it('l’encre arrive du thème, jamais d’un littéral', () => {
    for (const f of ['placementDropped.ts', 'placementShapes.ts', 'placementHitTest.ts']) {
      const src = readFileSync(resolve(__dirname, f), 'utf8')
      expect(/#[0-9a-fA-F]{6}\b/.test(src), `${f} porte une valeur hex`).toBe(false)
      expect(/oklch\(|rgba?\(/.test(src), `${f} porte une couleur littérale`).toBe(false)
    }
  })

  it('le tiroir et l’infobulle n’emploient que des classes SÉMANTIQUES', () => {
    // Tailwind de couleur brute (`text-red-500`, `bg-blue-900`…) : interdit dans `features/`.
    const brut = /\b(?:text|bg|border|fill|stroke)-(?:red|blue|green|yellow|orange|purple|pink|gray|grey|slate|zinc|neutral|stone|amber|lime|emerald|teal|cyan|sky|indigo|violet|fuchsia|rose)-\d{2,3}\b/
    for (const f of ['./settings/ReplaySettingsDrawer.tsx', 'ReplayPlacementTip.tsx']) {
      const src = readFileSync(resolve(__dirname, f), 'utf8')
      expect(brut.test(src), `${f} porte une classe Tailwind de couleur`).toBe(false)
      expect(/#[0-9a-fA-F]{6}\b/.test(src), `${f} porte une valeur hex`).toBe(false)
    }
  })
})
