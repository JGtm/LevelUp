/**
 * ObjectiveBalanceCard — « Rapport de force par famille de mode » (Escouade › Contributions,
 * section Objectif ; lot L3 du plan PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26, maquette C3EW).
 *
 * Un CADRE par famille de mode, bandeau « Bases · 4 matchs » ; dedans, les actions rangées sous
 * les intertitres Prendre / Défendre / Tenir ; pour chaque action, la barre camp contre camp :
 * notre camp en `team-ally` avec « compte · part » dans son segment, l'adversaire en
 * `team-enemy` avec son compte (S4 : les couleurs d'équipe remplacent les mots). Une valeur qui
 * ne tient pas dans son segment (6 px de marge de chaque côté) part sur la ligne de repli
 * au-dessus de la barre, pastille d'équipe devant (S3). Le trait 50 % en pointillé `warning`
 * (S5). Les temps s'écrivent en durée (« 17 min 48 »).
 */
import { useMemo, useRef } from 'react'

import { useSegmentLabelFit } from '@/components/charts/segmentLabelFit'
import { Tooltip } from '@/components/ui/tooltip'
import { tokenCssVar } from '@/lib/accessibility'

import type { FormesText } from '../formes/i18n'
import { buildBalanceByRole, type BalanceFamily, type BalanceLine } from './objectif.logic'
import { ObjectifFrame, ObjectifLegend } from './ObjectifFrame'
import type { ObjectifText } from './objectifStrings'

const ALLY = tokenCssVar('team-ally')
const ENEMY = tokenCssVar('team-enemy')
const PARITY = tokenCssVar('warning')

interface Props {
  families: BalanceFamily[]
  familyLabel: (family: string) => string
  columns: FormesText['columns']
  t: ObjectifText
  /**
   * Vue compacte du tiroir de comparaison de Sessions (maquette `makeBalance` avec `cp`) : une barre
   * par RÔLE (somme de ses actions, `buildBalanceByRole`) puis les colonnes facultatives, la part
   * entière seule dans les segments ; les comptes restent dans l'infobulle.
   */
  compact?: boolean
}

const fitKey = (family: string, key: string, side: 'us' | 'them') => `${family}|${key}|${side}`

/** Les lignes d'une famille en vue compacte, à la forme des lignes d'action (nom = rôle ou colonne). */
function compactFamilies(families: BalanceFamily[]): BalanceFamily[] {
  return buildBalanceByRole(families).map((f) => ({
    family: f.family,
    matches: f.matches,
    roles: [{ role: 'take' as const, lines: f.lines.map((l) => ({ key: l.key, duration: l.duration, optional: l.role == null, us: l.us, them: l.them, share: l.share })) }],
  }))
}

export function ObjectiveBalanceCard({ families: actions, familyLabel, columns, t, compact = false }: Props) {
  const families = useMemo(() => (compact ? compactFamilies(actions) : actions), [actions, compact])
  const bodyRef = useRef<HTMLDivElement | null>(null)
  const hidden = useSegmentLabelFit(bodyRef, families)
  const lineLabel = (key: string) => (compact && key in t.roles ? t.roles[key as keyof typeof t.roles] : (columns[key] ?? key))
  const legend = useMemo(
    () => (
      <ObjectifLegend
        ariaLabel={t.balance.title}
        items={[
          { kind: 'square', label: t.ourSide, color: ALLY },
          { kind: 'square', label: t.opponent, color: ENEMY },
          { kind: 'parity', label: t.parity, color: PARITY },
        ]}
      />
    ),
    [t],
  )
  return (
    <ObjectifFrame title={t.balance.title} info={t.balance.info} legend={legend} testId="objective-balance">
      <div ref={bodyRef} className="flex flex-col gap-3">
        {families.map((f) => (
          <div key={f.family} className="overflow-hidden rounded-lg border border-border" data-testid={`objective-balance-family-${f.family}`}>
            <div className="flex items-baseline gap-2 border-b border-border bg-muted px-3 py-[7px] text-sm font-semibold">
              {familyLabel(f.family)}
              <span className="text-[11.5px] font-normal text-muted-foreground">{t.matchesFmt(f.matches)}</span>
            </div>
            <div className="flex flex-col gap-[5px] px-3 pb-2.5 pt-2">
              {f.roles.map((r) => (
                <div key={r.role} className="flex flex-col gap-[5px]">
                  {!compact && <div className="mt-[3px] text-[10.5px] uppercase tracking-[.04em] text-muted-foreground">{t.roles[r.role]}</div>}
                  {r.lines.map((l) => (
                    <BalanceBar
                      key={l.key}
                      family={f.family}
                      line={l}
                      label={lineLabel(l.key)}
                      hidden={hidden}
                      t={t}
                      pctOnly={compact}
                    />
                  ))}
                </div>
              ))}
            </div>
          </div>
        ))}
      </div>
    </ObjectifFrame>
  )
}

function BalanceBar({
  family,
  line,
  label,
  hidden,
  t,
  pctOnly,
}: {
  family: string
  line: BalanceLine
  label: string
  hidden: ReadonlySet<string>
  t: ObjectifText
  /** Vue compacte : la part entière seule dans chaque segment. */
  pctOnly: boolean
}) {
  const fmt = (v: number) => (line.duration ? t.durationFmt(v) : String(Math.round(v * 10) / 10))
  const sharePct = line.share * 100
  const usText = pctOnly ? t.pctFmt(sharePct) : `${fmt(line.us)} · ${t.pctFmt(sharePct)}`
  const themText = pctOnly ? t.pctFmt(100 - sharePct) : fmt(line.them)
  // Un segment vide n'a pas de place : sa valeur part au repli, comme celle qui ne tient pas.
  const usHidden = line.us <= 0 || hidden.has(fitKey(family, line.key, 'us'))
  const themHidden = line.them <= 0 || hidden.has(fitKey(family, line.key, 'them'))
  const repliOffset = usHidden ? 0 : themHidden ? sharePct : null
  return (
    <div className="grid grid-cols-[150px_minmax(0,1fr)] items-center gap-2.5 text-xs" data-testid={`objective-balance-row-${family}-${line.key}`}>
      <div className="truncate leading-tight" title={label}>
        {label}
      </div>
      <div className="flex min-w-0 flex-col gap-[3px]">
        {repliOffset != null && (
          <div
            className="flex gap-2 text-2xs font-semibold tabular-nums"
            style={{ paddingLeft: `${repliOffset}%` }}
            data-testid={`objective-balance-repli-${family}-${line.key}`}
          >
            {usHidden && <RepliValue color={ALLY} text={usText} />}
            {themHidden && <RepliValue color={ENEMY} text={themText} />}
          </div>
        )}
        <div className="relative h-5 rounded-[3px] bg-muted" role="img" aria-label={`${label} : ${usText} / ${themText}`}>
          {line.us > 0 && (
            <Segment
              left={0}
              width={sharePct}
              color={ALLY}
              rounded={line.them > 0 ? 'rounded-l-[3px]' : 'rounded-[3px]'}
              fitKey={fitKey(family, line.key, 'us')}
              text={usText}
              hidden={usHidden}
              tip={t.balance.segmentTip(t.ourSide, label, fmt(line.us), t.pctFmt(sharePct))}
            />
          )}
          {line.them > 0 && (
            <Segment
              left={sharePct}
              width={100 - sharePct}
              color={ENEMY}
              rounded={line.us > 0 ? 'rounded-r-[3px] border-l-2 border-card' : 'rounded-[3px]'}
              fitKey={fitKey(family, line.key, 'them')}
              text={themText}
              hidden={themHidden}
              tip={t.balance.segmentTip(t.opponent, label, fmt(line.them), t.pctFmt(100 - sharePct))}
            />
          )}
          <div
            className="pointer-events-none absolute -bottom-[3px] -top-[3px] left-1/2 w-0 border-l-2 border-dashed"
            style={{ borderColor: PARITY }}
            aria-hidden
          />
        </div>
      </div>
    </div>
  )
}

function RepliValue({ color, text }: { color: string; text: string }) {
  return (
    <span className="inline-flex items-center gap-[3px]">
      <span className="inline-block h-2 w-2 rounded-[2px]" style={{ backgroundColor: color }} aria-hidden />
      {text}
    </span>
  )
}

function Segment({
  left,
  width,
  color,
  rounded,
  fitKey: key,
  text,
  hidden,
  tip,
}: {
  left: number
  width: number
  color: string
  rounded: string
  fitKey: string
  text: string
  hidden: boolean
  tip: string
}) {
  return (
    <div
      className={`absolute inset-y-0 ${rounded}`}
      style={{ left: `${left}%`, width: `${width}%`, backgroundColor: color }}
      data-fit-key={key}
    >
      <Tooltip content={tip} className="h-full w-full">
        <div className="flex h-full w-full cursor-help items-center justify-center overflow-hidden">
          {/* `text-white` : une écriture posée SUR l'aplat d'équipe, question de contraste dans
              le segment et non couleur sémantique (même usage que la Répartition des frags). */}
          <span
            data-fit-label
            className="whitespace-nowrap text-[11.5px] font-semibold tabular-nums text-white"
            style={{ visibility: hidden ? 'hidden' : 'visible' }}
          >
            {text}
          </span>
        </div>
      </Tooltip>
    </div>
  )
}
