/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail — LA GÉOMÉTRIE DE LA FRISE NE SE RECOPIE PAS DANS SES COMPOSANTS.
 *
 * # LE DÉFAUT QU'IL PRÉVIENT (2026-09-06, lot du trait de lecture)
 *
 * Un curseur natif réserve sa demi-largeur à chaque bout de sa course : la position d'un ratio
 * n'est donc pas `left: r%` mais `calc(8px + (100% - 16px) * r)` — c'est ce que rend `trackLeft`,
 * et c'est ce qui aligne une marque de kill sur la pastille au moment où elle passe dessus.
 *
 * La bulle de temps, elle, avait été posée à `left: var(--played)` — le pourcentage BRUT du
 * remplissage. Elle portait donc jusqu'à 8 px d'écart avec l'instant qu'elle annonce, pendant
 * plusieurs semaines, sans que personne le voie : aucun repère ne passait par là pour trahir le
 * décalage. Le trait de lecture, lui, le trahirait immédiatement — il traverse les marques.
 *
 * C'est le mode de défaillance à retenir : DEUX géométries qui cohabitent, dont l'une est
 * plausible (« un pourcentage de la largeur, quoi de plus naturel ») et fausse. Une relecture
 * future réintroduira la mauvaise de bonne foi. Ce test refuse donc la FORME — les littéraux de
 * la formule — et pas seulement la valeur.
 *
 * # CE QU'IL EXIGE
 *
 * La formule vit UNE fois, dans `model/replayTimelineTracksLogic.ts`, sous deux formes :
 * `trackLeft(ratio)` pour une position connue au rendu, `trackLeftVar(cssVar)` pour une position
 * qui vit dans une variable CSS. Les composants de la frise les APPELLENT et n'écrivent jamais
 * les nombres eux-mêmes.
 *
 * # CE QU'IL N'INTERDIT PAS
 *
 * Les longueurs qui ne sont pas de la géométrie de piste : la marge que la bulle de temps garde
 * aux deux bouts (`BUBBLE_EDGE`) est une demi-largeur de bulle, pas une demi-largeur de curseur.
 * Elle est exemptée NOMMÉMENT ci-dessous, avec sa raison — une exemption anonyme rouvrirait la
 * porte à n'importe quel `100% - …`.
 */
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

import { describe, expect, it } from 'vitest'

import { featureRoot } from '../test/featureFiles'

/** Les composants qui posent quelque chose sur les pistes. Tout nouveau va dans cette liste. */
const COMPOSANTS = ['ui/ReplayTimelineTracks.tsx', 'ui/ReplayPlayhead.tsx']

/**
 * Les littéraux de la géométrie de curseur. `THUMB_PX` compris : l'importer pour recomposer la
 * formule à la main est exactement la dérive que l'on refuse — le module de logique l'exporte
 * pour ses propres tests, pas pour que les composants le recalculent.
 *
 * POURQUOI DES MOTIFS ET PAS DES CHAÎNES pour les deux longueurs. « 8px » se trouve dans
 * `h-[18px]`, « 16px » se trouverait dans `w-[116px]` : une recherche de sous-chaîne rendrait
 * ce garde rouge sur des hauteurs de piste qui n'ont rien à voir avec la formule. Le regard
 * arrière écarte donc ce qui est précédé d'un CHIFFRE (une autre longueur) ou d'un CROCHET (une
 * valeur arbitraire Tailwind, jamais un `calc()`). Ce qui reste, c'est la formule recopiée :
 * `calc(8px + (100% - 16px) * …)`, où les deux nombres suivent une parenthèse ou une espace.
 */
const INTERDITS: readonly { readonly motif: RegExp; readonly quoi: string }[] = [
  { motif: /THUMB_PX/, quoi: 'THUMB_PX' },
  { motif: /100%\s*-/, quoi: '100% -' },
  { motif: /(?<![\d.[])8px/, quoi: '8px' },
  { motif: /(?<![\d.[])16px/, quoi: '16px' },
]

/**
 * Exemptions nommées, datées, avec leur raison. Retirées de la source AVANT le balayage.
 *
 *  - `calc(100% - ${BUBBLE_EDGE})` (2026-09-06) : la borne haute du `clamp` qui retient la bulle
 *    de temps dans la frise. Elle se mesure à la demi-largeur du TEXTE de la bulle, pas à la
 *    demi-largeur du curseur — c'est une autre grandeur, qui n'a rien à partager avec les pistes.
 */
const EXEMPTIONS = ['calc(100% - ${BUBBLE_EDGE})']

function source(chemin: string): string {
  return readFileSync(resolve(featureRoot(), chemin), 'utf8')
}

/** La source sans ses exemptions : ce qui reste doit être exempt de tout littéral de géométrie. */
function sourceBalayable(chemin: string): string {
  return EXEMPTIONS.reduce((src, exempt) => src.split(exempt).join(''), source(chemin))
}

describe('garde-rail : la géométrie de la frise ne vit qu’une fois', () => {
  for (const composant of COMPOSANTS) {
    for (const { motif, quoi } of INTERDITS) {
      it(`${composant} n’écrit pas « ${quoi} » : il appelle trackLeft / trackLeftVar`, () => {
        expect(
          sourceBalayable(composant),
          `${composant} recopie « ${quoi} », un littéral de la géométrie de curseur. ` +
            `Cette formule vit dans model/replayTimelineTracksLogic.ts : appeler trackLeft(ratio) ` +
            `pour une position connue au rendu, trackLeftVar(cssVar) pour une position qui suit ` +
            `la lecture. Recopiée ici, elle divergera — et le trait de lecture passera à côté du ` +
            `kill qu’il désigne sans qu’aucun autre test ne rougisse.`,
        ).not.toMatch(motif)
      })
    }
  }

  /**
   * SANS CE CAS, LE GARDE SERAIT VERT ET INERTE : un composant qui cesserait d'employer les
   * helpers passerait les cas ci-dessus haut la main — il n'y aurait plus rien à recopier.
   * C'est le mode de défaillance des gardes en « ne contient pas ».
   */
  it('les deux composants passent bien par les helpers — sans quoi ce garde ne garde rien', () => {
    const total = COMPOSANTS.map(source).join('\n')
    expect(total).toContain('trackLeft(')
    expect(total).toContain('trackLeftVar(')
  })

  /**
   * L'EXEMPTION DOIT RESTER VRAIE. Si la bulle change de mise en forme et que sa borne
   * disparaît, l'exemption devient une porte ouverte sur rien — et la prochaine `100% - …`
   * écrite ici passerait sans être vue. Une allowlist qui ne correspond plus à rien se retire.
   */
  it('chaque exemption correspond encore à une écriture réelle', () => {
    const total = COMPOSANTS.map(source).join('\n')
    for (const exempt of EXEMPTIONS) {
      expect(total, `l’exemption « ${exempt} » ne correspond plus à rien : la retirer`).toContain(
        exempt,
      )
    }
  })
})
