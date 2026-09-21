/**
 * SquadRiposteMatricePanel — repli « Qui riposte pour qui » de la carte « Riposte ».
 *
 * Une RIPOSTE est une mort de notre camp dont le tueur est abattu par un coéquipier dans
 * les 5 s. LIGNE = celui qui riposte, COLONNE = celui pour qui on riposte — la même
 * orientation que le graphe d'appui (Assistant / Bénéficiaire), son voisin de section.
 *
 * CE N'EST PLUS UNE CARTE (D19, 2026-09-21) : elle s'intitulait « Assistances croisées »
 * tout en comptant des ripostes, et son titre disputait le mot « assistance » au seul bloc
 * qui en compte vraiment. Elle est devenue le DÉTAIL par couple de la carte « Riposte »,
 * repliée et fermée par défaut — pas de SectionCard dans une SectionCard.
 *
 * RAMPE 0 → MAX, ET PAS « MIN OBSERVÉ » → MAX (correction 2026-09-13). Le wrapper ajuste
 * par défaut le bas de l'échelle à la plus petite valeur : sur un roster homogène
 * (133 … 190 ripostes), les six cases sortaient toutes dans le dernier quart de la rampe —
 * un aplat, dont on ne lisait plus aucune intensité. L'échelle part donc de ZÉRO : c'est le
 * seul bas qui a un sens pour un COMPTE.
 *
 * PALETTE : rampe de FRÉQUENCE, mono-teinte (`paletteMode="frequency"`). Un nombre de
 * ripostes n'est ni chaud ni froid — la rampe cold→hot lui collerait un jugement, et les
 * couleurs PAR JOUEUR (`squad-player-*`) feraient croire que la teinte désigne quelqu'un.
 *
 * LA DIAGONALE EST UNE CASE IMPOSSIBLE, PAS UNE MESURE QUI MANQUE (`emptyCells="blank"`).
 * Personne ne riposte pour soi-même. Le mode par défaut du wrapper la hachurait et posait
 * sous le graphe une légende « aucune mesure sur cet axe » — qui annonçait un trou de
 * données là où il n'y a qu'une impossibilité.
 *
 * LA LIGNE « REÇU N » ET LES DEUX PUCES SONT EN DOM, SOUS LE GRAPHE, pas dans le canvas.
 * ECharts ne sait pas poser un second rang d'étiquettes sous un axe de catégories ; les
 * fabriquer en `graphic` aurait été une mise en page manuelle en pixels, illisible et
 * fragile. La rangée reprend donc la géométrie de la grille du wrapper (`left: 96px`,
 * `right: 24px`, colonnes égales) et se réaligne d'elle-même à tout redimensionnement.
 */
import { useMemo } from 'react'

import { Heatmap2DChart, type ChartPointHeatmap } from '@/components/charts/Heatmap2DChart'
import { getEChartsThemeColors } from '@/components/charts/_utils'
import { NarrativeBadge } from '@/components/feedback/NarrativeBadge'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { tokenVar } from '@/lib/accessibility'
import { intlLocale } from '@/lib/formatters'
import type { SquadEchange } from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'

import {
  couvertureParJoueur,
  extremesCouverture,
  matriceSeries,
  matriceVide,
} from './squadRiposte.logic'
import { getSquadRiposteText } from './squadRiposteStrings'

/** Géométrie de la grille du wrapper `Heatmap2DChart` — la rangée « reçu N » s'y aligne. */
const GRILLE_GAUCHE_PX = 96
const GRILLE_DROITE_PX = 24

export interface SquadRiposteMatricePanelProps {
  echange: SquadEchange
}

export function SquadRiposteMatricePanel({ echange }: SquadRiposteMatricePanelProps) {
  const locale = useAppShellStore((s) => s.locale)
  const t = getSquadRiposteText(locale)
  const numLoc = intlLocale(locale)

  const perMatchFmt = useMemo(
    () => new Intl.NumberFormat(numLoc, { minimumFractionDigits: 2, maximumFractionDigits: 2 }),
    [numLoc],
  )

  const secondes = echange.fenetre_ms / 1000
  const series = useMemo(() => matriceSeries(echange), [echange])
  const extremes = useMemo(() => extremesCouverture(echange), [echange])
  const recus = useMemo(() => couvertureParJoueur(echange), [echange])

  const maxRipostes = useMemo(
    () => (echange.cellules ?? []).reduce((m, c) => Math.max(m, c.nombre), 0),
    [echange.cellules],
  )

  // Tooltip PROPRE à cette lecture : le libellé par défaut du wrapper parle de taux de
  // victoire et de matchs, ce qu'une case de cette matrice n'est pas.
  const formatTooltip = useMemo(
    () => (p: ChartPointHeatmap) =>
      t.matrixTooltip(p.y, p.x, p.value ?? 0, perMatchFmt.format(Number(p.detail?.perMatch ?? 0))),
    [t, perMatchFmt],
  )

  // ENCRE UNIQUE, ET ELLE SE LIT SUR TOUTE LA RAMPE. Le bas de la rampe de fréquence est
  // un bleu très sombre et le haut un bleu moyen : l'encre CLAIRE du thème est lisible sur
  // les deux, là où l'encre par défaut d'ECharts (grise) disparaît sur le bas.
  const cellLabelColor = getEChartsThemeColors().text

  if (matriceVide(echange)) {
    return <EmptyStateNotice title={t.emptyTitle} description={t.noPairs} />
  }

  return (
    <div className="space-y-2" data-testid="squad-riposte-matrice">
      <p className="text-2xs text-muted-foreground">{t.matrixHelp(secondes)}</p>
      <Heatmap2DChart
        series={series}
        paletteMode="frequency"
        valueRange={[0, maxRipostes]}
        showVisualMap={false}
        emptyCells="blank"
        cellLabelColor={cellLabelColor}
        formatTooltip={formatTooltip}
        height={260}
        frameless
      />
      {/* « reçu N » sous chaque COLONNE, aligné sur la grille du wrapper. */}
      <div
        className="grid gap-1 text-center"
        style={{
          paddingLeft: GRILLE_GAUCHE_PX,
          paddingRight: GRILLE_DROITE_PX,
          gridTemplateColumns: `repeat(${recus.length}, minmax(0, 1fr))`,
        }}
        data-testid="squad-riposte-recus"
      >
        {recus.map((j) => (
          <div key={j.xuid} className="space-y-1">
            <p className="text-2xs text-muted-foreground">{t.matrixReceived(j.ripostes)}</p>
            {extremes?.plusCouvert.xuid === j.xuid && (
              <NarrativeBadge
                size="sm"
                colorVar={tokenVar('outcome-win')}
                label={t.badgeMostCovered(j.gamertag)}
              />
            )}
            {extremes?.moinsCouvert.xuid === j.xuid && (
              <NarrativeBadge
                size="sm"
                colorVar={tokenVar('warning')}
                label={t.badgeLeastCovered(j.gamertag)}
              />
            )}
          </div>
        ))}
      </div>
    </div>
  )
}
