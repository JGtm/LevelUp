/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail des SOCLES D'ARME — les trois dérives que le compilateur ne voit pas.
 *
 * 1. UNE COULEUR EN DUR. Les modules du calque reçoivent leurs encres de l'appelant, qui les
 *    tient des variables du thème (règle projet : aucun hex, aucune classe Tailwind de couleur
 *    dans `features/`). Un `#8b97a3` glissé dans un tracé ne suivrait NI le thème clair NI le
 *    sombre, et rien ne le signalerait à l'exécution.
 *
 * 2. UN ÉTAT SANS LIBELLÉ. `PadState` a trois valeurs ; le typage `Record<Locale, ReplayText>`
 *    garantit que les DEUX langues ont les mêmes clés, mais pas que ces clés couvrent les trois
 *    états ni qu'elles portent un texte. Une chaîne vide passerait la compilation et laisserait
 *    l'infobulle muette.
 *
 * 3. LE RAMASSEUR AFFICHÉ. `padPickups[].xuid` vaut `null` partout (oracle à 79,7 % contre 90 %
 *    exigés) et le champ EXISTE au contrat : un rendu peut le lire sans voir qu'il est vide.
 *    C'est la clause la plus facile à violer par inadvertance — ce test interdit la lecture.
 */
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

import { REPLAY_TEXT } from './i18n'
import type { PadState } from './weaponPadsLayer'

/** Les fichiers de ce lot : le calque, la liste des tailles, le hook, l'infobulle. */
const FICHIERS = [
  'weaponPadsLayer.ts',
  'weaponPadFamilies.ts',
  'useReplayWeaponPads.ts',
  'ReplayWeaponPadTip.tsx',
]

const source = (f: string) => readFileSync(resolve(__dirname, f), 'utf8')

/** Les trois états, énumérés ici pour que l'ajout d'un quatrième fasse rougir ce test. */
const ETATS: PadState[] = ['full', 'uncertain', 'empty']

describe('garde-rail : aucune couleur écrite en dur dans le calque des socles', () => {
  it('aucun hexadécimal de couleur', () => {
    for (const f of FICHIERS) {
      expect(source(f), `${f} porte un hex de couleur`).not.toMatch(/#[0-9a-fA-F]{6}\b/)
    }
  })

  it('aucune classe Tailwind de couleur (les encres viennent du thème)', () => {
    for (const f of FICHIERS) {
      expect(source(f), `${f} porte une classe Tailwind de couleur`).not.toMatch(
        /\b(?:text|bg|border|fill|stroke)-(?:red|green|blue|yellow|amber|rose|emerald|sky|violet|orange|lime|teal|cyan|indigo|fuchsia|pink|slate|gray|zinc|neutral|stone)-\d{2,3}\b/,
      )
    }
  })
})

describe('garde-rail : chaque état de socle a son libellé, en FR et en EN', () => {
  it('les trois états sont nommés dans les deux langues', () => {
    for (const locale of ['fr', 'en'] as const) {
      const labels = REPLAY_TEXT[locale].padState
      expect(Object.keys(labels).sort(), `états ${locale} désynchronisés`).toEqual([...ETATS].sort())
      for (const s of ETATS) expect(labels[s], `${s} sans libellé ${locale}`).toBeTruthy()
    }
  })

  it('le calque, son état et sa note de lecture sont traduits — jamais un mot en dur', () => {
    for (const locale of ['fr', 'en'] as const) {
      const t = REPLAY_TEXT[locale]
      expect(t.layerWeaponPads, `titre ${locale}`).toBeTruthy()
      expect(t.layerWeaponPadsHint, `aide ${locale}`).toBeTruthy()
      expect(t.padPlacementNote, `note ${locale}`).toBeTruthy()
      expect(t.padCountdownFmt(12.2), `compte à rebours ${locale}`).toContain('13')
      expect(t.padRespawnFmt(12.2), `réapparition ${locale}`).toContain('13')
      expect(t.padCycleFmt(40.09, 2, 3), `cycle ${locale}`).toMatch(/40[.,]1/)
    }
  })

  it('les deux langues DIFFÈRENT là où elles le doivent (pas de FR recopié en EN)', () => {
    expect(REPLAY_TEXT.fr.layerWeaponPads).not.toBe(REPLAY_TEXT.en.layerWeaponPads)
    for (const s of ETATS) expect(REPLAY_TEXT.fr.padState[s]).not.toBe(REPLAY_TEXT.en.padState[s])
  })

  it('la note de lecture DIT que socle et râtelier ne sont pas distingués', () => {
    expect(REPLAY_TEXT.fr.padPlacementNote.toLowerCase()).toContain('râtelier')
    expect(REPLAY_TEXT.en.padPlacementNote.toLowerCase()).toContain('rack')
  })
})

describe('garde-rail : le RAMASSEUR n’est jamais lu', () => {
  it('aucun fichier du lot ne touche à `padPickups` ni à un `xuid`', () => {
    for (const f of FICHIERS) {
      const src = source(f)
      // Les commentaires ont le droit d'EXPLIQUER pourquoi le champ n'est pas lu ; le code,
      // non — on cherche donc le champ hors ligne de commentaire.
      const code = src
        .split('\n')
        .filter((l) => !/^\s*(\*|\/\/|\/\*)/.test(l))
        .join('\n')
      expect(code, `${f} lit padPickups`).not.toMatch(/padPickups/)
      expect(code, `${f} lit un xuid`).not.toMatch(/\bxuid\b/i)
    }
  })
})
