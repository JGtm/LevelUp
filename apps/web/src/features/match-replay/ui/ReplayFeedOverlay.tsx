/**
 * ReplayFeedOverlay — LE FIL DES ÉLIMINATIONS POSÉ SUR LE PLAN, en bas à gauche du bloc de la toile (Cinéma et
 * Plein cadre, quand la rangée de fiches est posée : la colonne ne porte alors plus le fil).
 *
 * LES CINQ DERNIÈRES ÉLIMINATIONS à l'instant lu, la plus récente EN BAS, sans en-tête, sans menu, sans fond :
 * des lignes nues (instant, lieu de la victime, tueur, arme, victime ; pas de médailles), à la ligne et aux
 * colonnes du fil de la colonne (`KillLine`, `cotesFeedLine.ts`). Il lit le MÊME fil que la colonne
 * (`entries`) ; `planFeed.ts` n'en garde que les éliminations, sans le filtre du Standard.
 *
 * IL NE CAPTE PAS LE POINTEUR : le zoom et le déplacement de la carte passent à travers lui. Il se place sous
 * les surcouches de la carte (écran de fin, compte à rebours : couche `COTES_FIL_PLAN.couche`, sous leur
 * couche 10). L'ombre portée du texte et des icônes vient du jeton de voile (`--scrim`), pour rester lisible
 * sur n'importe quel fond de carte ; les deux lignes les plus anciennes d'un fil plein s'estompent.
 * Aucune élimination passée : rien n'est rendu.
 */
import { useMemo } from 'react'

import { teamTokenCssVar } from '@/features/match-view/teamSeriesColor'
import type { FilmAllegiance } from '@/lib/replay/filmAllegiance'
import { NO_MARKS, type PlayerMarkKind } from '@/lib/replay/playerMarks'

import type { FeedZoneOf } from '../hooks/useFeedZones'
import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'
import { COTES_FIL_PLAN } from '../model/cotesFilPlan'
import type { ReplayFeedEntry } from '../model/killFeedLogic'
import { dernieresEliminations, opaciteLigneFilPlan } from '../model/planFeed'
import { displayClockMs, type ReplayWindowBounds } from '../model/replayWindow'
import { KillLine, type GamertagIndex, type InkOf } from './ReplayFeedLines'

/**
 * L'ombre portée, par le jeton de voile : le texte hérite de `text-shadow`, l'icône d'arme de la colonne
 * `arme` reçoit un `drop-shadow` (un masque d'image ne porte pas d'ombre de texte). Une classe écrite en
 * entier, que Tailwind lit telle quelle.
 */
const OMBRE_PORTEE =
  '[text-shadow:0_0_2px_var(--scrim),0_0_6px_var(--scrim),0_1px_2px_var(--scrim)] [&_[data-feed-col=arme]]:[filter:drop-shadow(0_0_2px_var(--scrim))]'

interface Props {
  /** Le fil aligné de la page (`model.feed`), trié par instant : le même que celui de la colonne. */
  entries: readonly ReplayFeedEntry[]
  /** L'instant lu, sur l'axe BRUT du film : il décide des éliminations déjà survenues. */
  nowMs: number
  /** Recale l'instant AFFICHÉ de chaque ligne, comme dans la colonne. */
  playWindow: ReplayWindowBounds | null
  /** Les noms de la feuille, par xuid de base. */
  xuidMeta: GamertagIndex
  /** L'allégeance du film vue du joueur regardé : l'encre des noms et de l'icône d'arme. */
  allegiance: FilmAllegiance
  locale: ReplayLocale
  /** Les marques d'identité (« moi », « ami ») ; absentes = aucune. */
  marks?: ReadonlyMap<string, PlayerMarkKind>
  /** La zone de la victime de chaque frag ; absent = la carte n'a aucune zone, pas de colonne de lieu. */
  zoneOf?: FeedZoneOf
  /** De combien le fil monte au-dessus du bas du bloc : la hauteur de la barre pleine quand elle est posée sur le plan. */
  liftPx?: number
}

export function ReplayFeedOverlay({ entries, nowMs, playWindow, xuidMeta, allegiance, locale, marks = NO_MARKS, zoneOf, liftPx = 0 }: Props) {
  const inkOf = useMemo<InkOf>(() => (xuid) => teamTokenCssVar(allegiance.ofXuid(xuid)), [allegiance])
  const lignes = dernieresEliminations(entries, nowMs)
  if (lignes.length === 0) return null
  return (
    <ul
      data-replay-plan-feed=""
      aria-label={REPLAY_TEXT[locale].feedFilterLabel.kills}
      aria-live="off"
      className={`pointer-events-none absolute flex flex-col ${OMBRE_PORTEE}`}
      style={{
        left: COTES_FIL_PLAN.margeGauchePx,
        bottom: COTES_FIL_PLAN.margeBasPx + liftPx,
        width: COTES_FIL_PLAN.largeurPx,
        maxWidth: `calc(100% - ${2 * COTES_FIL_PLAN.margeGauchePx}px)`,
        zIndex: COTES_FIL_PLAN.couche,
      }}
    >
      {lignes.map((entree, index) => (
        <KillLine
          key={entree.key}
          kill={entree.kill}
          replayMs={displayClockMs(entree.replayMs, playWindow)}
          inkOf={inkOf}
          xuidMeta={xuidMeta}
          marks={marks}
          locale={locale}
          zone={zoneOf?.(entree.key)}
          onPlan={{ opacity: opaciteLigneFilPlan(index, lignes.length) }}
        />
      ))}
    </ul>
  )
}
