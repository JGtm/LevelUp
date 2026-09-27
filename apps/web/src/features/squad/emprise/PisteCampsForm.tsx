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
 * Remplace, pour l'Emprise, `formes/forms/Piste100Form` (piste du lobby découpée par joueur,
 * adversaire hachuré et anonyme) : une autre question, dont les quatre cartes appelantes sont
 * retirées au lot L5.4.
 */
import { useRef, type ReactNode } from 'react'

import { useSegmentLabelFit } from '@/components/charts/segmentLabelFit'
import { Tooltip } from '@/components/ui/tooltip'
import { tokenCssVar } from '@/lib/accessibility'

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
}

export interface PisteCampsFormProps {
  rows: PisteCampsRow[]
  /** Une part en pourcentage (« 60 % »). */
  pctFmt: (v: number) => string
  /** Le libellé de la dernière graduation (« 100 % »). */
  axisMaxLabel: string
  /** Largeur de la colonne des noms (px) ; maquette : 118. */
  labelWidth?: number
}

const fitKey = (row: string, side: 'us' | 'them') => `${row}|${side}`

export function PisteCampsForm({ rows, pctFmt, axisMaxLabel, labelWidth = 118 }: PisteCampsFormProps) {
  const ref = useRef<HTMLDivElement | null>(null)
  const hidden = useSegmentLabelFit(ref, rows)
  const columns = `${labelWidth}px minmax(0,1fr)`
  return (
    <div ref={ref} className="flex flex-col gap-3.5">
      {rows.map((row) => (
        <PisteRow key={row.key} row={row} columns={columns} hidden={hidden} pctFmt={pctFmt} />
      ))}
      {/* Axe à 10 px des pistes (maquette : écart du corps de carte, pas celui des pistes). */}
      <div className="-mt-1 grid gap-3" style={{ gridTemplateColumns: columns }} aria-hidden>
        <span />
        <div className="relative h-3.5 text-[10.5px] tabular-nums text-muted-foreground">
          {TICKS.map((v) => (
            <span key={v} className="absolute -translate-x-1/2" style={{ left: `${v}%` }}>
              {v === 100 ? axisMaxLabel : v}
            </span>
          ))}
        </div>
      </div>
    </div>
  )
}

function PisteRow({
  row,
  columns,
  hidden,
  pctFmt,
}: {
  row: PisteCampsRow
  columns: string
  hidden: ReadonlySet<string>
  pctFmt: (v: number) => string
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
            <span>{usHidden && <RepliValue color={ALLY} count={row.us} pct={usPct} />}</span>
            <span>{themHidden && <RepliValue color={ENEMY} count={row.them} pct={themPct} />}</span>
          </div>
        )}
        <CampTrack row={row} share={share} usPct={usPct} themPct={themPct} usHidden={usHidden} themHidden={themHidden} />
      </div>
    </div>
  )
}

/** La barre des deux camps : segments, valeurs dedans (masquées quand elles ne tiennent pas), trait 50 %. */
function CampTrack({
  row,
  share,
  usPct,
  themPct,
  usHidden,
  themHidden,
}: {
  row: PisteCampsRow
  share: number
  usPct: string
  themPct: string
  usHidden: boolean
  themHidden: boolean
}) {
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
          <b className="font-extrabold">{row.us}</b> · {usPct}
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
          {themPct} · <b className="font-extrabold">{row.them}</b>
        </Segment>
      )}
      <div
        className="pointer-events-none absolute -bottom-[3px] -top-[3px] left-1/2 w-0 border-l-2 border-dashed"
        style={{ borderColor: PARITY }}
        aria-hidden
      />
    </div>
  )
}

function RepliValue({ color, count, pct }: { color: string; count: number; pct: string }) {
  return (
    <span className="inline-flex items-center">
      <span className="mr-[5px] inline-block h-[9px] w-[9px] rounded-[2px]" style={{ backgroundColor: color }} aria-hidden />
      <b className="text-[13px] font-bold text-foreground">{count}</b>
      <span className="whitespace-pre">{` · ${pct}`}</span>
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
