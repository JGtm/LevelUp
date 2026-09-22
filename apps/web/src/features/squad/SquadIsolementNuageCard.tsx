/**
 * SquadIsolementNuageCard — « Frags non ripostés » (section Coordination).
 *
 * UN PETIT POINT PAR MORT, UN GROS POINT PAR JOUEUR (contrat refondu le 2026-09-19,
 * PLAN_AJUSTEMENTS_PRE_V75 décision 4). En abscisse la distance au coéquipier visible le
 * plus proche RAPPORTÉE à la portée du radar du match (1,0 = à la portée, repère vertical
 * tracé) ; en ordonnée le délai avant la riposte, en secondes. Deux bandes nommées portent
 * ce qui n'a pas de valeur : « hors de vue » à droite (aucun coéquipier visible), « jamais
 * ripostée » en haut. Le gros point d'un joueur est la médiane de ses deux axes, sa taille
 * dit combien de morts.
 *
 * LA RÈGLE DES 5 s SE TRACE, ELLE NE COUPE PAS (décision utilisateur du 2026-09-22).
 * Une mort dont le tueur est tombé à 30 s se voit à 30 s, et la fenêtre de riposte est
 * posée en repère TIRETÉ horizontal, même grammaire que le `windowMark` de
 * `HistogramChart`. Au-dessus du repère, la mort se dessine en LOSANGE PLEIN — même encre,
 * même taille : la forme dit « le tueur est tombé, mais trop tard pour que ce soit une
 * riposte ». Les trois états (`etatMort`) viennent du CONTRAT, ce composant ne les
 * recalcule pas.
 *
 * ENCRE PLEINE PARTOUT (décision utilisateur du 2026-09-22) : plus aucune opacité sous 1,
 * ni sur les points, ni sur les repères médians, ni sur les bandes, ni sur les pastilles de
 * cette légende. Ce que l'atténuation codait — « ceci est un petit point, cela un repère »
 * — est porté par la TAILLE et par l'ordre de superposition.
 *
 * L'AXE DES DÉLAIS EST LOGARITHMIQUE ET BORNÉ PAR LE CONTRAT (même décision). Il court du
 * plancher d'affichage (0,2 s) au plafond `plafond_ms` (60 s) — au-delà le serveur ne
 * publie plus de délai du tout —, et ses graduations nomment des durées reconnaissables
 * (0,2 · 0,5 · 1 · 2 · 5 · 10 · 30 · 60 s). En linéaire, tout ce qui porte le sens — les
 * ripostes sous la fenêtre de 5 s — s'écrasait au ras de l'axe.
 *
 * LA FIGURE SE CONSTRUIT AILLEURS : `squadIsolementNuageOption.ts` (logique pure, testable
 * sans monter le composant). Ici, le rendu React et la légende DOM, rien d'autre.
 *
 * La carte n'est pas montée quand la section est absente du contrat (cf.
 * SquadSynergiesPage) : une section omise n'est pas une section à zéro.
 */
import { useMemo } from 'react'

import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import { TooltipParagraphs } from '@/components/ui/info-tooltip'
import { titleWithInfo } from '@/components/ui/title-with-info'
import { SectionCard } from '@/components/ui/section-card'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { intlLocale } from '@/lib/formatters'
import type {
  SquadEchangeJoueur,
  SquadIsolementMort,
  SquadIsolementRepere,
  SquadNuageIsolement,
} from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'

import { getSquadPlayerColors } from './colors'
import {
  echellesNuage,
  fenetreSecondes,
  plafondSecondes,
  TAILLE_MORT,
} from './squadIsolement.logic'
import { buildNuageOption, seriesParJoueur } from './squadIsolementNuageOption'
import { getSquadIsolementText } from './squadIsolementStrings'

export interface SquadIsolementNuageCardProps {
  nuage: SquadNuageIsolement
  joueurs: SquadEchangeJoueur[]
}

/** Diamètres (px) des pastilles de la légende de taille — les deux bouts de l'échelle. */
const TAILLE_MIN_LEGENDE = 12
const TAILLE_MAX_LEGENDE = 22

export function SquadIsolementNuageCard({ nuage, joueurs }: SquadIsolementNuageCardProps) {
  const locale = useAppShellStore((s) => s.locale)
  const t = getSquadIsolementText(locale)
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
  const numFmt = useMemo(
    () => new Intl.NumberFormat(numLoc, { minimumFractionDigits: 1, maximumFractionDigits: 1 }),
    [numLoc],
  )
  // Les graduations de l'axe log vont de 0,2 s à 60 s : une décimale AU PLUS, et aucune
  // imposée — « 1,0 s » et « 60,0 s » alourdiraient huit étiquettes pour rien.
  const secFmt = useMemo(
    () => new Intl.NumberFormat(numLoc, { minimumFractionDigits: 0, maximumFractionDigits: 1 }),
    [numLoc],
  )

  const morts = useMemo(() => nuage.morts ?? [], [nuage.morts])
  const reperes = useMemo(() => nuage.reperes ?? [], [nuage.reperes])
  const vide = morts.length === 0

  const playerColors = useMemo(() => {
    const [main, ...teammates] = joueurs
    return getSquadPlayerColors(main?.gamertag ?? '', teammates.map((j) => j.gamertag))
  }, [joueurs])

  const series = useMemo(() => seriesParJoueur(morts, joueurs), [morts, joueurs])
  // Le PLAFOND vient du contrat (`plafond_ms`) : c'est le haut de la zone mesurée de l'axe
  // des délais, jamais un 60 codé en dur de ce côté-ci.
  const echelles = useMemo(() => echellesNuage(morts, plafondSecondes(nuage)), [morts, nuage])
  const repereParXUID = useMemo(() => {
    const map = new Map<string, SquadIsolementRepere>()
    for (const r of reperes) map.set(r.xuid, r)
    return map
  }, [reperes])

  // Les extrêmes RÉELS du roster : la taille des gros points s'y projette, et la légende
  // de taille les nomme. Une échelle absolue ne dirait plus rien au-delà de son plafond.
  // La fenêtre vient du CONTRAT : c'est la règle du jeu, elle ne se recopie pas ici.
  const fenetreS = fenetreSecondes(nuage)

  const volumes = reperes.map((r) => r.nb_morts).filter((n) => n > 0)
  const mortsMin = volumes.length > 0 ? Math.min(...volumes) : 0
  const mortsMax = volumes.length > 0 ? Math.max(...volumes) : 0

  const buildOption = useMemo(
    () => (s: ChartSeries<SquadIsolementMort>[]) =>
      buildNuageOption(s, {
        playerColors,
        echelles,
        repereParXUID,
        t,
        pctFmt,
        numFmt,
        secFmt,
        mortsMin,
        mortsMax,
        fenetreS,
      }),
    [
      playerColors,
      echelles,
      repereParXUID,
      t,
      pctFmt,
      numFmt,
      secFmt,
      mortsMin,
      mortsMax,
      fenetreS,
    ],
  )

  return (
    <SectionCard
      title={t.cardTitle}
      label={t.sectionLabel}
      titleAdornment={titleWithInfo(<TooltipParagraphs items={[t.help, t.figure]} />)}
    >
      <div className="space-y-2 px-3 py-2" data-testid="squad-isolement-nuage">
        {vide ? (
          <EmptyStateNotice title={t.emptyTitle} description={t.emptyDescription} />
        ) : (
          <>
            <ChartCard series={series} buildOption={buildOption} height={380} frameless />
            {/* LÉGENDE DE TAILLE, en DOM : ECharts n'en a pas pour un encodage de taille.
                Elle porte les VRAIES valeurs extrêmes du roster — un encodage qu'on ne
                nomme pas ne se lit pas. Pastilles à ENCRE PLEINE : l'ancien `/50` ne
                codait rien, il atténuait. */}
            <div
              className="flex flex-wrap items-center gap-4 text-2xs text-muted-foreground"
              data-testid="squad-isolement-legende-taille"
            >
              <span className="flex items-center gap-1.5">
                <span
                  className="inline-block rounded-full bg-muted-foreground"
                  style={{ width: TAILLE_MIN_LEGENDE, height: TAILLE_MIN_LEGENDE }}
                />
                {t.legendDeaths(mortsMin)}
              </span>
              <span className="flex items-center gap-1.5">
                <span
                  className="inline-block rounded-full bg-muted-foreground"
                  style={{ width: TAILLE_MAX_LEGENDE, height: TAILLE_MAX_LEGENDE }}
                />
                {t.legendDeaths(mortsMax)}
              </span>
              <span className="flex items-center gap-1.5">
                <span
                  className="inline-block rounded-full bg-muted-foreground"
                  style={{ width: TAILLE_MORT, height: TAILLE_MORT }}
                />
                {t.legendDeath}
              </span>
              <span className="flex items-center gap-1.5">
                {/* Le LOSANGE de la légende : plein, même encre, même gabarit que le
                    disque voisin — un carré tourné d'un quart de tour, la silhouette
                    exacte du symbole `diamond` d'ECharts. */}
                <span
                  className="inline-block bg-muted-foreground"
                  style={{
                    width: TAILLE_MORT,
                    height: TAILLE_MORT,
                    transform: 'rotate(45deg)',
                  }}
                  data-testid="squad-isolement-legende-hors-fenetre"
                />
                {t.legendOutOfWindow}
              </span>
              <span className="flex items-center gap-1.5">
                <span
                  className="inline-block rounded-full border border-dashed border-muted-foreground"
                  style={{ width: TAILLE_MIN_LEGENDE, height: TAILLE_MIN_LEGENDE }}
                />
                {t.legendLowSample(nuage.plancher_echantillon_faible)}
              </span>
            </div>
          </>
        )}
      </div>
    </SectionCard>
  )
}
