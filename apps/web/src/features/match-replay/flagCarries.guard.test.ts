/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail des DRAPEAUX DE CTF — les quatre dérives que le compilateur ne voit pas.
 *
 * 1. UNE COULEUR EN DUR. Les modules du calque reçoivent leur encre de l'appelant, qui la tient
 *    des variables du thème (règle projet : aucun hex, aucune classe Tailwind de couleur dans
 *    `features/`). Un `#8b97a3` glissé dans un tracé ne suivrait NI le thème clair NI le sombre,
 *    et rien ne le signalerait à l'exécution.
 *
 * 2. UN ÉTAT SANS LIBELLÉ. `FlagState` a quatre valeurs ; le typage `Record<Locale, ReplayText>`
 *    garantit que les DEUX langues ont les mêmes clés, mais pas que ces clés couvrent les quatre
 *    états ni qu'elles portent un texte. Une chaîne vide passerait la compilation et laisserait
 *    l'infobulle muette.
 *
 * 3. UNE URL D'IMAGE DEVINÉE. Le drapeau n'est PAS un `weap` (phase 0, item 0.3) : il n'a pas de
 *    tag, `weaponLabels` ne le nomme sur AUCUN des trois témoins CTF, et l'atlas des vignettes
 *    est indexé sur cet espace de tags. Écrire un chemin de fichier à la main (`contour-26.png`,
 *    par exemple) marcherait aujourd'hui et se tromperait d'arme à la prochaine régénération de
 *    l'atlas — silencieusement. Le glyphe est donc tracé au canvas, et ce test l'exige.
 *
 * 4. LA RÉSERVE DE `carried_open` PERDUE. Ce que la mesure ne ferme pas doit se dire à l'écran :
 *    l'état porte sa réserve dans son libellé ET dans une note. Les deux sont vérifiées ici,
 *    parce qu'une atténuation seule se lit comme un effet de style.
 */
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

import type { FlagState } from './flagCarriesLayer'
import { REPLAY_TEXT } from './i18n'

/** Les fichiers de ce lot : le calque, le hook, l'infobulle, le regroupement des infobulles. */
const FICHIERS = [
  'flagCarriesLayer.ts',
  'useReplayFlagCarries.ts',
  'ReplayFlagTip.tsx',
  'ReplayCanvasTips.tsx',
]

const source = (f: string) => readFileSync(resolve(__dirname, f), 'utf8')

/** Les quatre états, énumérés ici pour que l'ajout d'un cinquième fasse rougir ce test. */
const ETATS: FlagState[] = ['carried', 'carried_open', 'dropped', 'home']

/** Les trois points de vue de camp — la page ne connaît qu'eux (jamais un nom d'équipe). */
const CAMPS = ['ally', 'enemy', 'unknown'] as const

describe('garde-rail : aucune couleur écrite en dur dans le calque des drapeaux', () => {
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

describe("garde-rail : le glyphe est TRACÉ, jamais chargé d'une URL devinée", () => {
  it("aucun chemin d'asset ni chargement d'image dans le calque", () => {
    for (const f of FICHIERS) {
      const code = source(f)
        .split('\n')
        .filter((l) => !/^\s*(\*|\/\/|\/\*)/.test(l))
        .join('\n')
      expect(code, `${f} construit un chemin d'asset`).not.toMatch(/weapons-assets|\.png|\/static\//)
      expect(code, `${f} charge une image`).not.toMatch(/new Image\(|drawImage\(/)
    }
  })
})

describe('garde-rail : chaque état de drapeau a son libellé, en FR et en EN', () => {
  it('les quatre états sont nommés dans les deux langues', () => {
    for (const locale of ['fr', 'en'] as const) {
      const labels = REPLAY_TEXT[locale].flagState
      expect(Object.keys(labels).sort(), `états ${locale} désynchronisés`).toEqual([...ETATS].sort())
      for (const s of ETATS) expect(labels[s], `${s} sans libellé ${locale}`).toBeTruthy()
    }
  })

  it('les trois camps sont nommés dans les deux langues', () => {
    for (const locale of ['fr', 'en'] as const) {
      const sides = REPLAY_TEXT[locale].flagSide
      expect(Object.keys(sides).sort(), `camps ${locale} désynchronisés`).toEqual([...CAMPS].sort())
      for (const s of CAMPS) expect(sides[s], `${s} sans libellé ${locale}`).toBeTruthy()
    }
  })

  it('le calque, sa durée et son porteur inconnu sont traduits — jamais un mot en dur', () => {
    for (const locale of ['fr', 'en'] as const) {
      const t = REPLAY_TEXT[locale]
      expect(t.layerFlagCarries, `titre ${locale}`).toBeTruthy()
      expect(t.layerFlagCarriesHint, `aide ${locale}`).toBeTruthy()
      expect(t.flagCarrierUnknown, `porteur inconnu ${locale}`).toBeTruthy()
      expect(t.flagSinceFmt(12.4), `durée ${locale}`).toContain('12')
      expect(t.flagOpenNote, `réserve ${locale}`).toBeTruthy()
    }
  })

  it('les deux langues DIFFÈRENT là où elles le doivent (pas de FR recopié en EN)', () => {
    expect(REPLAY_TEXT.fr.layerFlagCarries).not.toBe(REPLAY_TEXT.en.layerFlagCarries)
    expect(REPLAY_TEXT.fr.flagOpenNote).not.toBe(REPLAY_TEXT.en.flagOpenNote)
    for (const s of ETATS) expect(REPLAY_TEXT.fr.flagState[s]).not.toBe(REPLAY_TEXT.en.flagState[s])
    for (const s of CAMPS) expect(REPLAY_TEXT.fr.flagSide[s]).not.toBe(REPLAY_TEXT.en.flagSide[s])
  })
})

describe('garde-rail : la réserve de `carried_open` est DITE, pas seulement dessinée', () => {
  it("le libellé de l'état porte lui-même la réserve", () => {
    expect(REPLAY_TEXT.fr.flagState.carried_open.toLowerCase()).toContain('non datée')
    expect(REPLAY_TEXT.en.flagState.carried_open.toLowerCase()).toContain('undated')
  })

  it('la note dit BORNE HAUTE, et non une durée mesurée', () => {
    expect(REPLAY_TEXT.fr.flagOpenNote.toLowerCase()).toContain('borne haute')
    expect(REPLAY_TEXT.en.flagOpenNote.toLowerCase()).toContain('upper bound')
  })
})

describe('garde-rail : le camp se dit allié / adverse, jamais par une couleur', () => {
  it("aucun nom de couleur d'équipe dans les libellés servis", () => {
    for (const locale of ['fr', 'en'] as const) {
      for (const s of CAMPS) {
        expect(REPLAY_TEXT[locale].flagSide[s]).not.toMatch(
          /\b(rouge|bleu|vert|jaune|red|blue|green|yellow|eagle|cobra)\b/i,
        )
      }
    }
  })
})
