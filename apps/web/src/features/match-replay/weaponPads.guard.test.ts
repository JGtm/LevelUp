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
import { PAD_EQUIPMENT_FAMILIES } from './weaponPadFamilies'
import type { PadState } from './weaponPadTime'

/**
 * Les fichiers de ce lot : le calque, la lecture temporelle, la liste des tailles, le hook,
 * l'infobulle. `weaponPadTime.ts` a rejoint la liste le 2026-08-27, quand la lecture des états
 * et du compte à rebours a été extraite du calque — un fichier sorti du périmètre du garde-rail
 * en sortirait aussi les règles qu'il porte.
 */
const FICHIERS = [
  'weaponPadsLayer.ts',
  'weaponPadTime.ts',
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
      expect(t.padRespawnMeasuredFmt(12.2), `réapparition mesurée ${locale}`).toContain('13')
      expect(t.padRespawnExpectedFmt(12.2), `réapparition attendue ${locale}`).toContain('13')
    }
  })

  /**
   * LA RÉSERVE EST DANS LE SIGNE (D3, 2026-08-27) : le compte MESURÉ vise une apparition vue
   * dans le film, il est exact et ne doit PAS porter le « ≈ » ; celui du CYCLE le garde. Deux
   * libellés identiques feraient lire une prédiction comme une mesure — c'est précisément la
   * confusion que la décision supprime.
   */
  it('le compte MESURÉ et le compte ATTENDU ne se disent pas de la même façon', () => {
    for (const locale of ['fr', 'en'] as const) {
      const t = REPLAY_TEXT[locale]
      expect(t.padRespawnMeasuredFmt(12.2), `mesuré ${locale} porte la réserve`).not.toContain('≈')
      expect(t.padRespawnExpectedFmt(12.2), `attendu ${locale} sans réserve`).toContain('≈')
      expect(t.padRespawnMeasuredFmt(12.2)).not.toBe(t.padRespawnExpectedFmt(12.2))
    }
    expect(REPLAY_TEXT.fr.padRespawnMeasuredFmt(12)).not.toBe(
      REPLAY_TEXT.en.padRespawnMeasuredFmt(12),
    )
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

describe('garde-rail : une famille NON-ARME de socle est nommée, jamais servie brute', () => {
  const cles = Object.keys(PAD_EQUIPMENT_FAMILIES)

  it('chaque famille de la table a son libellé dans les DEUX langues', () => {
    for (const locale of ['fr', 'en'] as const) {
      const labels = REPLAY_TEXT[locale].padEquipmentFamily
      expect(Object.keys(labels).sort(), `familles ${locale} désynchronisées`).toEqual(
        [...cles].sort(),
      )
      for (const key of cles) {
        expect(labels[key as keyof typeof labels], `${key} sans libellé ${locale}`).toBeTruthy()
      }
    }
  })

  it('aucun libellé ne recopie la clé du document — c’est le défaut qu’on corrige', () => {
    for (const locale of ['fr', 'en'] as const) {
      for (const value of Object.values(REPLAY_TEXT[locale].padEquipmentFamily)) {
        expect(value, `libellé ${locale} = clé brute`).not.toMatch(/_/)
      }
    }
  })

  it('les deux langues DIFFÈRENT là où elles le doivent, et la réserve existe', () => {
    expect(REPLAY_TEXT.fr.padEquipmentFamily.powerup_overshield).not.toBe(
      REPLAY_TEXT.en.padEquipmentFamily.powerup_overshield,
    )
    for (const locale of ['fr', 'en'] as const) {
      expect(REPLAY_TEXT[locale].padPlacementNotePowerUp, `réserve ${locale}`).toBeTruthy()
    }
    expect(REPLAY_TEXT.fr.padPlacementNotePowerUp).not.toBe(REPLAY_TEXT.en.padPlacementNotePowerUp)
  })
})

/**
 * GARDE-RAIL DU LISERÉ (2026-08-27, retour utilisateur « icône AVEC contour »).
 *
 * La cuisson refusait le liseré aux images FINIES du jeu (`outline: null`) au motif qu'on ne
 * peut pas les reteindre — vrai pour leur corps, faux pour leur contour : cerner ne demande que
 * la SILHOUETTE. Le type `PadIcon.outline` n'est plus nullable, donc le compilateur tient déjà
 * l'invariant ; ce cas garde la TRACE de la règle à l'endroit où on l'a violée, pour qu'un
 * `outline: null` réintroduit avec un `as` ou un `?? null` ne passe pas en silence.
 */
describe('garde-rail : toute vignette de socle est CERNÉE, image finie comprise', () => {
  it('la cuisson ne sert jamais un liseré absent', () => {
    expect(source('useReplayWeaponPads.ts')).not.toMatch(/outline:\s*null/)
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
