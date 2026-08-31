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
 * 2. UNE INFOBULLE MUETTE. Le typage `Record<Locale, ReplayText>` garantit que les DEUX langues
 *    ont les mêmes clés, mais pas qu'elles portent un texte : une chaîne vide passerait la
 *    compilation. Depuis le 2026-08-28 l'infobulle ne dit plus que le NOM et, s'il existe, le
 *    COMPTE À REBOURS (« je ne veux pas de blabla dedans ») — ce sont donc ces libellés-là que
 *    ce fichier surveille, et l'état n'a plus de mot à l'écran.
 *
 * 3. LE RAMASSEUR AFFICHÉ. `padPickups[].xuid` est PUBLIÉ depuis le schéma 29 (2026-08-31) —
 *    l'événement natif le porte — mais AUCUN écran ne l'affiche : ce garde-rail reste, et sa
 *    raison change. Il ne dit plus « la donnée n'existe pas » mais « la donnée existe et son
 *    affichage n'a pas été conçu ». (Justification d'origine, désormais caduque : oracle à 79,7 % contre 90 %
 *    exigés) et le champ EXISTE au contrat : un rendu peut le lire sans voir qu'il est vide.
 *    C'est la clause la plus facile à violer par inadvertance — ce test interdit la lecture.
 */
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

import { REPLAY_TEXT } from './i18n'
import { PAD_EQUIPMENT_FAMILIES } from './weaponPadFamilies'

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

describe('garde-rail : ce que dit l’infobulle est traduit, en FR et en EN', () => {
  it('le calque et ses deux libellés d’infobulle sont traduits — jamais un mot en dur', () => {
    for (const locale of ['fr', 'en'] as const) {
      const t = REPLAY_TEXT[locale]
      expect(t.layerWeaponPads, `titre ${locale}`).toBeTruthy()
      expect(t.layerWeaponPadsHint, `aide ${locale}`).toBeTruthy()
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
    expect(REPLAY_TEXT.fr.padRespawnMeasuredFmt(12)).not.toBe(
      REPLAY_TEXT.en.padRespawnMeasuredFmt(12),
    )
  })

  /**
   * LE BLABLA NE REVIENT PAS (retour utilisateur du 2026-08-28). L'infobulle portait l'état du
   * socle et une note de lecture de deux lignes ; le retour demande « juste le nom, et à la
   * rigueur le timer ». Une parenthèse d'explication rajoutée au compte à rebours serait
   * exactement la dérive qui a été retirée — d'où ce test sur la LONGUEUR du libellé.
   */
  it('le compte à rebours reste court : un chiffre, une unité, pas une phrase', () => {
    for (const locale of ['fr', 'en'] as const) {
      const t = REPLAY_TEXT[locale]
      expect(t.padRespawnMeasuredFmt(12.2).length, `mesuré ${locale}`).toBeLessThanOrEqual(30)
      expect(t.padRespawnExpectedFmt(12.2).length, `attendu ${locale}`).toBeLessThanOrEqual(30)
      expect(t.padRespawnMeasuredFmt(12.2), `mesuré ${locale} porte une parenthèse`).not.toContain('(')
      expect(t.padRespawnExpectedFmt(12.2), `attendu ${locale} porte une parenthèse`).not.toContain('(')
    }
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

  it('les deux langues DIFFÈRENT là où elles le doivent', () => {
    expect(REPLAY_TEXT.fr.padEquipmentFamily.powerup_overshield).not.toBe(
      REPLAY_TEXT.en.padEquipmentFamily.powerup_overshield,
    )
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
