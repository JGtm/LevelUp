/**
 * ReplayTransportCompact — LA BARRE DE LECTURE D'UNE LIGNE du Cinéma et du Plein cadre. Mêmes commandes
 * que la barre complète (`ReplayTransport`), sur 56 px, de gauche à droite :
 *
 *  - la CAMÉRA (emplacement `camera`, voir ci-dessous) et le son ;
 *  - le transport : lecture / pause (rond plein), recommencer, −10, +10, le temps « instant / durée »,
 *    l'interrupteur de lecture automatique ;
 *  - la FRISE (`ReplayTimelineCompact`), qui prend la largeur restante ;
 *  - la vitesse ; le bouton « Plan tourné » quand l'Auto a tourné le plan (`ReplayRotatedButton`) ; le zoom − et +
 *    (la commande de zoom flottante n'est pas rendue dans ces cadres) ;
 *  - capture et export (`ReplayCaptureButtons`), icônes de cadre et plein écran du système
 *    (`ReplayFrameButtons`).
 *
 * Un séparateur vertical sépare les groupes (`ReplayCompactBarControls`). La barre est PILOTÉE PAR L'OBJET DE
 * COMMANDES de la toile et par lui seul : aucun état de lecture ne vit ici, et les raccourcis clavier sont ceux
 * de la toile.
 *
 * SUR UNE OU DEUX LIGNES (`barLines`) : quand la frise n'aurait plus sa largeur minimale, elle passe seule sur la
 * première ligne et les commandes sur la seconde. La barre mesure, la règle décide.
 *
 * SOUS LE BLOC OU SUR LUI (`barPlacement`) : au Cinéma, et au Plein cadre quand la barre fine est éteinte, la barre est
 * SOUS le bloc arrondi du plan, sur le fond de la page : sans fond propre, à marge latérale étroite (`barMarginX`),
 * que la règle des lignes compte. Dans le bloc ou posée sur le plan, elle garde sa marge de socle.
 *
 * FINE OU PLEINE (`barForm`) : au Plein cadre, avec la préférence « barre fine », la barre est posée sur le plan par
 * la toile (`overlaysTransportBar`) et prend la forme FINE pendant la lecture : la barre pleine réduite à
 * `fineEchelle`, plus basse, plus translucide, floutée derrière. Pause, survol, focus clavier ou geste récent la
 * rendent pleine ; mouvement réduit demandé, le passage est instantané. Une commande qui a le focus n'est donc
 * jamais dans la forme réduite. Hors surimpression, la barre garde sa forme pleine.
 *
 * L'EMPLACEMENT DE LA CAMÉRA : `camera` est un nœud que l'appelant fournit (le bouton « Caméra » et son
 * menu). Absent, rien n'est rendu — ni bouton inerte, ni espace — et le son occupe l'extrémité gauche. Présent,
 * il se pose à gauche du son, dans le même groupe. Le nœud est monté par la fonction de rendu de l'emplacement
 * de la toile, qui ne contient aucun hook : un état du menu vit dans le composant qu'il fournit.
 *
 * PENDANT UN EXPORT, LA LECTURE ET LA FRISE SONT NEUTRALISÉES comme dans la barre complète : la boucle
 * d'export écrit l'image courante, et la lecture, les sauts et le glisser de la frise écrivent le MÊME ref.
 * Les icônes de cadre gardent la règle de `ReplayFrameButtons`.
 */
import type { CSSProperties, ReactNode } from 'react'

import { useReducedMotion } from '../hooks/useReducedMotion'
import { useTransportBarForm } from '../hooks/useTransportBarForm'
import { useTransportBarLines } from '../hooks/useTransportBarLines'
import { COTES_BARRE_COMPACTE as C } from '../model/cotesBarreCompacte'
import { barMarginX, barPlacement, type BarForm, type BarLines, type BarPlacement } from '../model/transportBarLogic'
import { ReplayCompactBarControls } from './ReplayCompactBarControls'
import type { ReplayTransportCommands } from './replayTransportCommands'

export interface ReplayTransportCompactProps extends ReplayTransportCommands {
  /** Le bouton « Caméra » et son menu, à gauche du son ; absent = rien (cf. l'en-tête). */
  camera?: ReactNode
}

/** Le fond de la barre : aucun sous le bloc (le fond de la page), léger dans le bloc, plein en surimpression, ou fin et flouté. */
function backgroundClass(placement: BarPlacement, form: BarForm): string {
  if (placement === 'below') return ''
  if (placement === 'in-block') return 'bg-background/40'
  return form === 'fine' ? 'bg-background/45 backdrop-blur-[6px]' : 'bg-background/70'
}

/** La hauteur de la racine : celle de la barre pleine sur une ligne, celle de la forme fine, ou celle du contenu. */
function barHeight(form: BarForm, lines: BarLines, rowHeight: number): number | undefined {
  if (form === 'fine') {
    if (lines === 1) return C.fineHauteurPx
    return rowHeight > 0 ? Math.round(rowHeight * C.fineEchelle) : undefined
  }
  return lines === 1 ? C.barreHauteurPx : undefined
}

/** La hauteur de la rangée sur une ligne : toute la barre, ou — forme fine — de quoi remplir la hauteur fine une fois réduite. */
function rowHeightOf(form: BarForm, lines: BarLines): number | string | undefined {
  if (lines === 2) return undefined
  return form === 'fine' ? C.fineHauteurPx / C.fineEchelle : '100%'
}

/** La rangée des commandes : la barre pleine, ou — forme fine — la même, plus large d'autant qu'elle est réduite. */
function rowStyle(form: BarForm, lines: BarLines, animated: boolean, marginX: number): CSSProperties {
  const fine = form === 'fine'
  return {
    columnGap: C.ecartPx,
    padding: `0 ${marginX}px`,
    height: rowHeightOf(form, lines),
    width: fine ? `${100 / C.fineEchelle}%` : undefined,
    transform: fine ? `scale(${C.fineEchelle})` : undefined,
    transformOrigin: 'left top',
    transition: animated ? 'transform 150ms ease' : undefined,
    '--bc-marge-y': `${C.deuxLignesCommandeMargeYPx}px`,
  } as CSSProperties
}

/** Une ligne : les commandes centrées. Deux : elles se rangent à la suite, la frise passe en tête (cf. `ReplayCompactBarControls`). */
const ROW_CLASS: Record<BarLines, string> = {
  1: 'flex items-center',
  2: 'flex flex-wrap content-start [&>:not([data-replay-bar-timeline])]:[margin-block:var(--bc-marge-y)]',
}

export function ReplayTransportCompact(props: ReplayTransportCompactProps) {
  const { camera, ...commands } = props
  const { frameControl, thinBar, playing } = commands
  const { form, handlers } = useTransportBarForm({ frame: frameControl.frame, thinPreferred: thinBar, playing })
  const placement = barPlacement(frameControl.frame, thinBar)
  const marginX = barMarginX(placement)
  const { outerRef, rowRef, lines, rowHeight } = useTransportBarLines(marginX)
  const animated = !useReducedMotion()
  return (
    <div
      ref={outerRef}
      className={`relative ${backgroundClass(placement, form)}`}
      style={{ height: barHeight(form, lines, rowHeight), transition: animated ? 'height 150ms ease' : undefined }}
      data-replay-transport="compact"
      data-replay-bar-form={form}
      data-replay-bar-lines={lines}
      {...handlers}
    >
      <div ref={rowRef} className={ROW_CLASS[lines]} style={rowStyle(form, lines, animated, marginX)}>
        <ReplayCompactBarControls {...commands} camera={camera} lines={lines} />
      </div>
    </div>
  )
}
