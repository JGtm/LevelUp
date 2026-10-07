/**
 * PisteCampsForm — LA PISTE CAMP CONTRE CAMP de l'onglet Emprise (lot L5.3 du plan
 * PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26 ; maquette de l'onglet, `renderPistes`).
 *
 * La barre entière vaut les prises des deux camps. Notre camp en `team-ally` à gauche,
 * l'adversaire en `team-enemy` PLEIN à droite (S4 : la couleur d'équipe remplace les mots, la
 * hachure reste réservée à « sans film »). Chaque segment porte son « compte · part » : le nôtre
 * collé à gauche (« 12 · 60 % »), celui de l'adversaire collé à droite (« 40 % · 8 »). Une
 * valeur qui ne tient pas dans son segment avec 6 px de marge de chaque côté (mesure au pixel,
 * `segmentLabelFit`) part sur la ligne de repli AU-DESSUS de la barre, pastille d'équipe devant,
 * du côté de son camp (S3 : jamais tronquée, jamais seulement en infobulle). Le trait 50 % en
 * pointillé `warning` (S5). Axe 0-100 % sous les pistes.
 *
 * Une ligne peut porter, sous sa barre, un contenu de plus (`below`) : « Frags obtenus avec
 * les ressources » y pose la barre fine de l'exposition (`ThinCampTrack`) et sa ligne de
 * valeurs. L'ancienne piste du lobby découpée par joueur (`Piste100Form`, adversaire hachuré)
 * est supprimée avec ses quatre cartes (lot L5.4).
 */
import { useRef, type ReactNode } from 'react'

import { useSegmentLabelFit } from '@/components/charts/segmentLabelFit'
import { Tooltip } from '@/components/ui/tooltip'
import { tokenCssVar } from '@/lib/accessibility'

import { pisteColumns } from './pisteLayout'
import { TipText } from './TipText'

const ALLY = tokenCssVar('team-ally')
const ENEMY = tokenCssVar('team-enemy')
const PARITY = tokenCssVar('warning')

/** Graduations de l'axe (maquette : 0, 25, 50, 75, 100 %). */
const TICKS = [0, 25, 50, 75, 100] as const

export interface PisteCampsRow {
  key: string
  label: string
  sublabel?: string
  /** Pastille devant le nom (couleur de la ressource, S8). */
  dot?: string
  us: number
  them: number
  usTip: string
  themTip: string
  /** Sous la barre, dans sa colonne (« Frags obtenus avec les ressources » : la barre fine et la ligne d'exposition). */
  below?: ReactNode
}

export interface PisteCampsFormProps {
  rows: PisteCampsRow[]
  /** Une part en pourcentage (« 60 % »). */
  pctFmt: (v: number) => string
  /** Le libellé de la dernière graduation (« 100 % »). */
  axisMaxLabel: string
  /** Largeur de la colonne des noms (px) ; maquette : 118. */
  labelWidth?: number
  /**
   * Parts seules (vue compacte du tiroir de comparaison de Sessions) : chaque segment et le repli
   * n'écrivent que la part ; le compte reste dans l'infobulle. Défaut : « compte · part ».
   */
  pctOnly?: boolean
}

const fitKey = (row: string, side: 'us' | 'them') => `${row}|${side}`

export function PisteCampsForm({ rows, pctFmt, axisMaxLabel, labelWidth = 118, pctOnly = false }: PisteCampsFormProps) {
  const ref = useRef<HTMLDivElement | null>(null)
  const hidden = useSegmentLabelFit(ref, rows)
  const columns = pisteColumns(labelWidth)
  return (
    <div ref={ref} className="flex flex-col gap-3.5">
      {rows.map((row) => (
        <PisteRow key={row.key} row={row} columns={columns} hidden={hidden} pctFmt={pctFmt} pctOnly={pctOnly} />
      ))}
      <TrackAxis columns={columns} ticks={TICKS.map((v) => ({ at: v, label: v === 100 ? axisMaxLabel : String(v) }))} />
    </div>
  )
}

/**
 * TrackAxis — l'axe sous les pistes, dans la colonne des barres ; à 10 px des pistes (maquette :
 * écart du corps de carte, pas celui des pistes). `at` en pourcentage de la piste.
 */
export function TrackAxis({ columns, ticks }: { columns: string; ticks: { at: number; label: string }[] }) {
  return (
    <div className="-mt-1 grid gap-3" style={{ gridTemplateColumns: columns }} aria-hidden>
      <span />
      <div className="relative h-3.5 text-[10.5px] tabular-nums text-muted-foreground">
        {ticks.map((tick) => (
          <span key={tick.at} className="absolute -translate-x-1/2 whitespace-nowrap" style={{ left: `${tick.at}%` }}>
            {tick.label}
          </span>
        ))}
      </div>
    </div>
  )
}

function PisteRow({
  row,
  columns,
  hidden,
  pctFmt,
  pctOnly,
}: {
  row: PisteCampsRow
  columns: string
  hidden: ReadonlySet<string>
  pctFmt: (v: number) => string
  pctOnly: boolean
}) {
  const n = row.us + row.them
  const share = n > 0 ? (row.us / n) * 100 : 0
  const usPct = pctFmt(share)
  const themPct = pctFmt(100 - share)
  // Un segment vide n'a pas de place : sa valeur part au repli, comme celle qui ne tient pas.
  const usHidden = row.us <= 0 || hidden.has(fitKey(row.key, 'us'))
  const themHidden = row.them <= 0 || hidden.has(fitKey(row.key, 'them'))
  return (
    <div
      className="grid items-center gap-3"
      style={{ gridTemplateColumns: columns }}
      data-testid={`piste-camps-row-${row.key}`}
    >
      <div className="min-w-0 text-[12.5px] leading-tight">
        <span className="inline-flex items-center">
          {row.dot && (
            <span className="mr-1.5 inline-block h-[9px] w-[9px] shrink-0 rounded-[2px]" style={{ backgroundColor: row.dot }} aria-hidden />
          )}
          {row.label}
        </span>
        {row.sublabel && <small className="block text-[11px] text-muted-foreground">{row.sublabel}</small>}
      </div>
      <div className="flex min-w-0 flex-col gap-1">
        {(usHidden || themHidden) && (
          <div
            className="flex justify-between gap-2 text-xs tabular-nums text-muted-foreground"
            data-testid={`piste-camps-repli-${row.key}`}
          >
            <span>{usHidden && <RepliValue color={ALLY} count={pctOnly ? null : row.us} pct={usPct} />}</span>
            <span>{themHidden && <RepliValue color={ENEMY} count={pctOnly ? null : row.them} pct={themPct} />}</span>
          </div>
        )}
        <CampTrack
          row={row}
          share={share}
          labels={{ us: usPct, them: themPct, pctOnly }}
          usHidden={usHidden}
          themHidden={themHidden}
        />
        {row.below}
      </div>
    </div>
  )
}

/** La barre des deux camps : segments, valeurs dedans (masquées quand elles ne tiennent pas), trait 50 %. */
function CampTrack({
  row,
  share,
  labels,
  usHidden,
  themHidden,
}: {
  row: PisteCampsRow
  share: number
  /** Les parts de chaque camp, et `pctOnly` : la part seule dans le segment (vue compacte). */
  labels: { us: string; them: string; pctOnly: boolean }
  usHidden: boolean
  themHidden: boolean
}) {
  const { us: usPct, them: themPct, pctOnly } = labels
  return (
    <div className="relative h-[22px] rounded-[3px] bg-muted" role="img" aria-label={`${row.label} : ${row.us} · ${usPct} / ${themPct} · ${row.them}`}>
      {row.us > 0 && (
        <Segment
          left={0}
          width={share}
          color={ALLY}
          rounded={row.them > 0 ? 'rounded-l-[3px]' : 'rounded-[3px]'}
          fitKey={fitKey(row.key, 'us')}
          hidden={usHidden}
          align="start"
          tip={row.usTip}
        >
          {pctOnly ? (
            <b className="font-extrabold">{usPct}</b>
          ) : (
            <>
              <b className="font-extrabold">{row.us}</b> · {usPct}
            </>
          )}
        </Segment>
      )}
      {row.them > 0 && (
        <Segment
          left={share}
          width={100 - share}
          color={ENEMY}
          rounded={row.us > 0 ? 'rounded-r-[3px]' : 'rounded-[3px]'}
          // Le filet de 2 px à la couleur de la carte entre les deux camps (maquette :
          // `.seg + .seg { box-shadow: -2px 0 0 var(--card) }`).
          separated={row.us > 0}
          fitKey={fitKey(row.key, 'them')}
          hidden={themHidden}
          align="end"
          tip={row.themTip}
        >
          {pctOnly ? (
            <b className="font-extrabold">{themPct}</b>
          ) : (
            <>
              {themPct} · <b className="font-extrabold">{row.them}</b>
            </>
          )}
        </Segment>
      )}
      <ParityMark />
    </div>
  )
}

/** Le trait 50 % en pointillé `warning` (S5), 3 px au-delà de la piste. */
function ParityMark() {
  return (
    <div
      className="pointer-events-none absolute -bottom-[3px] -top-[3px] left-1/2 w-0 border-l-2 border-dashed"
      style={{ borderColor: PARITY }}
      aria-hidden
    />
  )
}

/**
 * ThinCampTrack — la BARRE FINE (8 px) des deux camps, sous une piste : notre part en
 * `team-ally`, celle de l'adversaire en `team-enemy`, filet de 2 px entre les deux, trait 50 %.
 * Aucune valeur dedans (maquette : la ligne d'exposition dessous les écrit) ; infobulle par camp.
 */
export function ThinCampTrack({ us, them, usTip, themTip, label }: { us: number; them: number; usTip: string; themTip: string; label: string }) {
  const n = us + them
  const share = n > 0 ? (us / n) * 100 : 0
  return (
    <div className="relative h-2 rounded-[3px] bg-muted" role="img" aria-label={label}>
      {us > 0 && (
        <ThinSegment left={0} width={share} color={ALLY} rounded={them > 0 ? 'rounded-l-[3px]' : 'rounded-[3px]'} tip={usTip} />
      )}
      {them > 0 && (
        <ThinSegment
          left={share}
          width={100 - share}
          color={ENEMY}
          rounded={us > 0 ? 'rounded-r-[3px]' : 'rounded-[3px]'}
          separated={us > 0}
          tip={themTip}
        />
      )}
      <ParityMark />
    </div>
  )
}

function ThinSegment({
  left,
  width,
  color,
  rounded,
  separated,
  tip,
}: {
  left: number
  width: number
  color: string
  rounded: string
  separated?: boolean
  tip: string
}) {
  return (
    <div
      className={`absolute inset-y-0 ${rounded}`}
      style={{ left: `${left}%`, width: `${width}%`, backgroundColor: color, boxShadow: separated ? '-2px 0 0 var(--card)' : undefined }}
    >
      <Tooltip content={<TipText text={tip} />} className="h-full w-full">
        <div className="h-full w-full cursor-help" />
      </Tooltip>
    </div>
  )
}

/** La valeur repliée au-dessus de la barre ; `count` nul = la part seule (vue compacte). */
function RepliValue({ color, count, pct }: { color: string; count: number | null; pct: string }) {
  return (
    <span className="inline-flex items-center">
      <span className="mr-[5px] inline-block h-[9px] w-[9px] rounded-[2px]" style={{ backgroundColor: color }} aria-hidden />
      {count == null ? (
        <b className="text-[13px] font-bold text-foreground">{pct}</b>
      ) : (
        <>
          <b className="text-[13px] font-bold text-foreground">{count}</b>
          <span className="whitespace-pre">{` · ${pct}`}</span>
        </>
      )}
    </span>
  )
}

function Segment({
  left,
  width,
  color,
  rounded,
  separated,
  fitKey: key,
  hidden,
  align,
  tip,
  children,
}: {
  left: number
  width: number
  color: string
  rounded: string
  separated?: boolean
  fitKey: string
  hidden: boolean
  align: 'start' | 'end'
  tip: string
  children: ReactNode
}) {
  return (
    <div
      className={`absolute inset-y-0 ${rounded}`}
      style={{
        left: `${left}%`,
        width: `${width}%`,
        backgroundColor: color,
        boxShadow: separated ? '-2px 0 0 var(--card)' : undefined,
      }}
      data-fit-key={key}
    >
      <Tooltip content={<TipText text={tip} />} className="h-full w-full">
        <div
          className={`flex h-full w-full cursor-help items-center overflow-hidden ${
            align === 'start' ? 'justify-start pl-[7px]' : 'justify-end pr-[7px]'
          }`}
        >
          {/* `text-white` : écriture posée SUR l'aplat d'équipe, question de contraste dans le
              segment et non couleur sémantique (même usage que « Rapport de force par famille
              de mode » et la Répartition des frags). */}
          <span
            data-fit-label
            className="whitespace-nowrap text-xs font-medium tabular-nums leading-none text-white"
            style={{ visibility: hidden ? 'hidden' : 'visible' }}
          >
            {children}
          </span>
        </div>
      </Tooltip>
    </div>
  )
}
