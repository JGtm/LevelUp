/**
 * cotesCorps.ts — LES MARGES DU CORPS DE CHAQUE CADRE, en pixels : ce que le corps (la boîte que mesure
 * `useCardRowLayout`) laisse de chaque côté à son contenu, et sous lui. Une cote PAR CADRE : la règle de tenue de la
 * rangée (`rowLayout`) et celle de l'en-tête (`cotesEnteteScore`) comptent ces marges dans la largeur qu'elles
 * demandent, et le cadre (`ReplayFrameLayout`) les pose en `style` depuis ce même tableau, de sorte que le seuil
 * est exact et ne peut pas diverger de la mise en page.
 *
 * `margeLateralePx` est le remplissage latéral PROPRE du corps (et de l'en-tête, qui lui est aligné) : celui que la
 * largeur mesurée du corps comprend. Le Plein cadre est un cadre d'écran qui pose déjà `REPLAY_FULLSCREEN_FRAME` (12 px
 * de chaque côté) : sa marge propre de 4 px y ajoute 16 px entre le bord de l'écran et le contenu, comme l'en-tête.
 * `margeBassePx` est la marge sous le corps : celle du corps au Cinéma, celle du bas de l'écran au Plein cadre.
 * Le Standard n'a que sa marge latérale : sa boîte est celle de la page bornée (`COTES_PAGE_STANDARD`).
 */
import { COTES_PAGE_STANDARD } from './cotesStandard'
import type { ReplayFrame } from './replayFrame'

export const COTES_CORPS_CADRE = Object.freeze({
  cinema: Object.freeze({ margeLateralePx: 24, margeBassePx: 8 }),
  'plein-cadre': Object.freeze({ margeLateralePx: 4, margeBassePx: 6 }),
})

/** La marge latérale propre du corps d'un cadre, comptée par les règles de largeur. */
export function margeCorpsDe(frame: ReplayFrame): number {
  return frame === 'standard' ? COTES_PAGE_STANDARD.margeLateralePx : COTES_CORPS_CADRE[frame].margeLateralePx
}
