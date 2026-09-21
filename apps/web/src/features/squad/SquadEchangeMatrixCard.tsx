/**
 * SquadEchangeMatrixCard — « Qui couvre qui » (onglet Synergies, carte 3 de l'échange).
 *
 * Un ÉCHANGE est une mort vengée : un coéquipier abat le tueur dans les 5 s.
 * LIGNE = celui qui venge, COLONNE = celui qui est vengé — la même orientation que le
 * graphe des assistances (Assistant / Bénéficiaire), son voisin immédiat.
 *
 * L'ORIENTATION ET LA DÉFINITION DE L'ÉCHANGE VIVENT DANS L'INFOBULLE ⓘ DU TITRE
 * (2026-09-19) : deux paragraphes de texte gris sous la grille, lus une fois puis jamais.
 * Le bandeau de couverture, la mention « échanges réalisés sur N matchs » et la légende de
 * rampe ont été retirés avec eux — la grille porte déjà ses nombres.
 *
 * RAMPE 0 → MAX, ET PAS « MIN OBSERVÉ » → MAX (correction 2026-09-13, maquette 4c520da6).
 * Le wrapper ajuste par défaut le bas de l'échelle à la plus petite valeur : sur un roster
 * homogène (133 … 190 échanges), les six cases sortaient toutes dans le dernier quart de la
 * rampe — un aplat, dont on ne lisait plus aucune intensité. L'échelle part donc de ZÉRO :
 * c'est le seul bas qui a un sens pour un COMPTE.
 *
 * PALETTE : rampe de FRÉQUENCE, mono-teinte (`paletteMode="frequency"`). Un nombre de
 * vengeances n'est ni chaud ni froid — la rampe cold→hot lui collerait un jugement, et les
 * couleurs PAR JOUEUR (`squad-player-*`) feraient croire que la teinte désigne quelqu'un.
 *
 * LA DIAGONALE EST UNE CASE IMPOSSIBLE, PAS UNE MESURE QUI MANQUE (`emptyCells="blank"`).
 * Personne ne se venge soi-même. Le mode par défaut du wrapper la hachurait et posait sous
 * le graphe une légende « aucune mesure sur cet axe » — qui annonçait un trou de données là
 * où il n'y a qu'une impossibilité. Elle reste donc vide, sans damier ni légende ; les deux
 * axes portent les mêmes gamertags, ce qui dit déjà pourquoi.
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
import { InfoTooltip, TooltipParagraphs } from '@/components/ui/info-tooltip'
import { SectionCard } from '@/components/ui/section-card'
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
  PLANCHER_MORTS,
} from './squadEchange.logic'
import { getSquadEchangeText } from './squadEchangeStrings'

/** Géométrie de la grille du wrapper `Heatmap2DChart` — la rangée « reçu N » s'y aligne. */
const GRILLE_GAUCHE_PX = 96
const GRILLE_DROITE_PX = 24

export interface SquadEchangeMatrixCardProps {
  echange: SquadEchange
}

export function SquadEchangeMatrixCard({ echange }: SquadEchangeMatrixCardProps) {
  const locale = useAppShellStore((s) => s.locale)
  const t = getSquadEchangeText(locale)
  const numLoc = intlLocale(locale)

  const pctFmt = useMemo(
    () =>
      new Intl.NumberFormat(numLoc, {
        style: 'percent',
        minimumFractionDigits: 1,
        maximumFractionDigits: 1,
      }),
    [numLoc],
  )
  const perMatchFmt = useMemo(
    () => new Intl.NumberFormat(numLoc, { minimumFractionDigits: 2, maximumFractionDigits: 2 }),
    [numLoc],
  )

  const secondes = echange.fenetre_ms / 1000
  const series = useMemo(() => matriceSeries(echange), [echange])
  const extremes = useMemo(() => extremesCouverture(echange), [echange])
  const recus = useMemo(() => couvertureParJoueur(echange), [echange])
  const vide = matriceVide(echange)

  const maxEchanges = useMemo(
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

  // Encre du nombre écrit dans la case : claire sur une case sombre (bas de rampe),
  // sombre sur une case claire (haut de rampe). Les deux encres sont celles du THÈME du
  // graphe (fond de carte / texte), pas des couleurs neuves.
  // ENCRE UNIQUE, ET ELLE SE LIT SUR TOUTE LA RAMPE. Le bas de la rampe de fréquence est
  // un bleu très sombre et le haut un bleu moyen : l'encre CLAIRE du thème est lisible sur
  // les deux, là où l'encre par défaut d'ECharts (grise) disparaît sur le bas.
  const cellLabelColor = getEChartsThemeColors().text

  // Le CONSTAT EN MOTS, au-dessus du graphe : « sur N matchs, X des Y morts de votre camp
  // ont été vengées dans les 5 s ». Un lecteur doit pouvoir repartir avec la phrase sans
  // lire la grille.
  const narrative = t.narrative({
    matches: echange.matchs_total,
    brut: echange.couverture.brut,
    n: echange.couverture.n,
    seconds: secondes,
    rate: pctFmt.format(echange.couverture.taux),
  })

  // LA RÉSERVE D'ÉCHANTILLON REJOINT L'INFOBULLE ⓘ DU TITRE (2026-09-21). Elle vivait en
  // pied de carte, en gris, sous la grille : une ligne de méthode posée là où l'œil cherche
  // la donnée. Une carte n'a désormais qu'UNE infobulle, et tout ce qui dit la méthode ou la
  // portée y tient — ici l'orientation de la grille, puis la réserve quand elle s'applique.
  const aide = (
    <TooltipParagraphs
      items={[
        t.matrixHelp(secondes),
        echange.couverture.echantillon_faible
          ? `${t.lowSample} — ${t.lowSampleHint(PLANCHER_MORTS)}`
          : null,
      ]}
    />
  )

  return (
    <SectionCard
      title={t.sectionTitle}
      label={t.sectionLabel}
      titleAdornment={(label) => (
        <span className="flex items-center gap-1.5" data-testid="squad-echange-low-sample">
          {label}
          <InfoTooltip content={aide} />
        </span>
      )}
    >
      <div className="space-y-2 px-3 py-2" data-testid="squad-echange-matrix">
        <p
          className="border-l-2 border-info pl-3 text-sm text-foreground"
          data-testid="squad-echange-narrative"
        >
          {narrative}
        </p>
        {vide ? (
          <EmptyStateNotice title={t.emptyTitle} description={t.noPairs} />
        ) : (
          <>
            <Heatmap2DChart
              series={series}
              paletteMode="frequency"
              valueRange={[0, maxEchanges]}
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
              data-testid="squad-echange-recus"
            >
              {recus.map((j) => (
                <div key={j.xuid} className="space-y-1">
                  <p className="text-2xs text-muted-foreground">{t.matrixReceived(j.vengeances)}</p>
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
          </>
        )}
      </div>
    </SectionCard>
  )
}
