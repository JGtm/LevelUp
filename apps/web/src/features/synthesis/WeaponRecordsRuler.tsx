/**
 * WeaponRecordsRuler — « la règle » des records de distance de frag par arme (Synthèse).
 *
 * Plan : .ai/V7.5/PLAN_RECORDS_DISTANCE_2026-09-20.md, rendu A de la maquette
 * MAQUETTE_REGLE_RECORDS_2026-09-20.html (choix utilisateur du 2026-09-20). Une seule ligne
 * graduée du contact vers la longue portée, un losange par arme à son record, couleur par
 * CLASSE d'arme (la même famille `frag-*` que le sunburst voisin), libellés étagés au-dessus,
 * légende EN DESSOUS ET CENTRÉE. La géométrie vit dans `weaponRecords_logic.ts` (pur, testé).
 *
 * POURQUOI DU SVG ET PAS ECHARTS : le graphe est un TEXTE placé (dix-huit libellés étagés
 * par un placement glouton qui dépend de la largeur réelle) autour d'un axe unique. Une série
 * `custom` d'ECharts ne connaît la largeur du repère qu'au moment de dessiner, donc ne sait
 * pas réserver le bon nombre de rangs au-dessus de l'axe ; en SVG, la hauteur est CALCULÉE
 * depuis la largeur mesurée, et le texte est du vrai texte, net au zoom. Le chrome de la carte
 * (bandeau, pied de légende) est celui des autres blocs de la page (`SectionCard`, même pied
 * que `ChartCard`).
 *
 * CHAQUE RECORD OUVRE LE REJEU À L'INSTANT DU FRAG : `?t=<time_ms>&clock=match` — l'instant
 * vient de `match_kill_events.time_ms`, l'horloge du MATCH, et c'est la route du rejeu qui
 * recale sur l'axe du film une fois le document chargé (même contrat que la carte tactique).
 * Navigation par le routeur, jamais un `<a href>` natif (qui rechargerait l'application).
 *
 * Aucun libellé ne dit « portée de l'arme » : c'est un usage mesuré du joueur. Ce qui est
 * écarté (corps à corps, chute et environnement, équipement) est NOMMÉ en pied, jamais tu.
 */
import { useNavigate } from '@tanstack/react-router'
import { useEffect, useMemo, useRef, useState, type KeyboardEvent, type MouseEvent } from 'react'

import { ChartLegend, type ChartLegendItem } from '@/components/charts/ChartLegend'
import { InfoTooltip } from '@/components/ui/info-tooltip'
import { SectionCard } from '@/components/ui/section-card'
import { FRAG_CLASS_ORDER, fragClassToken } from '@/lib/accessibility/scales/fragClass'
import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'
import type { SynthesisWeaponRecords, WeaponDistanceRecordRow } from '@/lib/api/types'
import { formatDate, intlLocale } from '@/lib/formatters'
import { formatMessage, type ManifestLocale } from '@/lib/i18n/format'
import { fragsManifest } from '@/lib/i18n/generated/frags'
import { synthesisManifest } from '@/lib/i18n/generated/synthesis'
import { useTitleSlug } from '@/lib/title-routing'
import { useAppShellStore } from '@/stores/appShellStore'

import {
  RULER_FALLBACK_WIDTH_PX,
  formatMeters,
  labelBaselineY,
  resolveRecordLabel,
  weaponRecordsLayout,
  type RulerItem,
} from './weaponRecords_logic'

/** Demi-diagonale du losange. */
const DIAMOND_PX = 6

type SynthesisKey = keyof typeof synthesisManifest

interface WeaponRecordsRulerProps {
  records: SynthesisWeaponRecords
  playerSlug: string
}

/** Largeur du conteneur, suivie par ResizeObserver ; repli fixe sans observateur (tests). */
function useMeasuredWidth(ref: React.RefObject<HTMLDivElement | null>): number {
  const [width, setWidth] = useState(RULER_FALLBACK_WIDTH_PX)
  useEffect(() => {
    const el = ref.current
    if (!el || typeof ResizeObserver === 'undefined') return
    setWidth(el.getBoundingClientRect().width || RULER_FALLBACK_WIDTH_PX)
    const ro = new ResizeObserver((entries) => {
      for (const entry of entries) {
        if (entry.contentRect.width > 0) setWidth(entry.contentRect.width)
      }
    })
    ro.observe(el)
    return () => ro.disconnect()
  }, [ref])
  return width
}

/** Les classes présentes, dans l'ordre canonique du sunburst, une pastille chacune. */
function legendItems(rows: readonly WeaponDistanceRecordRow[], locale: ManifestLocale): ChartLegendItem[] {
  const present = new Set(rows.map((r) => r.class ?? ''))
  return FRAG_CLASS_ORDER.filter((c) => present.has(c)).map((c) => ({
    key: c,
    label: formatMessage(fragsManifest, `frags.class.${c}` as never, locale),
    color: tokenCssVar(fragClassToken(c)),
  }))
}

export function WeaponRecordsRuler({ records, playerSlug }: WeaponRecordsRulerProps) {
  const locale = useAppShellStore((s) => s.locale) as ManifestLocale
  const t = (key: SynthesisKey, vars?: Record<string, unknown>) =>
    formatMessage(synthesisManifest, key, locale, vars)
  const titleSlug = useTitleSlug()
  const navigate = useNavigate()
  const hostRef = useRef<HTMLDivElement | null>(null)
  const width = useMeasuredWidth(hostRef)
  // Le contrat sérialise toujours `weapons` (jamais `null`), mais le type généré ne le sait
  // pas : le repli vide protège le rendu sans inventer de donnée.
  const weapons = useMemo(() => records.weapons ?? [], [records.weapons])
  const layout = useMemo(() => weaponRecordsLayout(weapons, locale, width), [weapons, locale, width])
  const [hover, setHover] = useState<{ item: RulerItem; x: number; y: number } | null>(null)

  const openReplay = (row: WeaponDistanceRecordRow) => {
    void navigate({
      to: '/{-$lang}/t/$titleSlug/players/$playerSlug/matches/$matchId/replay',
      params: { titleSlug, playerSlug, matchId: row.record.match_id },
      // `t` est une CHAÎNE dans le schéma de la route (cf. TacticalCellCard) ; `clock: match`
      // parce que `time_ms` est l'horloge du match, recalée par la route du rejeu.
      search: { t: String(row.record.time_ms), clock: 'match' },
    })
  }
  const onMove = (item: RulerItem) => (e: MouseEvent<SVGGElement>) => {
    const host = hostRef.current?.getBoundingClientRect()
    setHover({ item, x: e.clientX - (host?.left ?? 0), y: e.clientY - (host?.top ?? 0) })
  }
  const onKey = (row: WeaponDistanceRecordRow) => (e: KeyboardEvent<SVGGElement>) => {
    if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault()
      openReplay(row)
    }
  }

  const count = t('synthesis.weapon_records.count', {
    weapons: weapons.length, measured: records.measured_kills, total: records.total_kills,
  })
  const excludedList = (records.excluded ?? [])
    .map((w) => t('synthesis.weapon_records.excluded_item', { label: resolveRecordLabel(w, locale), measured: w.measured }))
    .join(' · ')
  const legend = legendItems(weapons, locale)
  // Les armes écartées et la réserve de couverture vivent dans le (i) du titre, pas sous
  // le graphe (retrait de la note demandé le 2026-09-20) : ce qui est écarté reste NOMMÉ.
  const help = (
    <span data-testid="weapon-records-help">
      {excludedList ? `${t('synthesis.weapon_records.excluded', { list: excludedList })} ` : ''}
      {t('synthesis.weapon_records.coverage_note')}
    </span>
  )

  return (
    <SectionCard
      title={t('synthesis.weapon_records.title')}
      label={t('synthesis.weapon_records.title')}
      titleAdornment={(label) => (
        <span className="flex flex-wrap items-baseline justify-between gap-2">
          <span className="inline-flex items-center gap-1.5">
            {label}
            <InfoTooltip content={help} iconClass="w-3.5 h-3.5" />
          </span>
          <span className="text-xs font-normal tabular-nums text-muted-foreground">{count}</span>
        </span>
      )}
      footer={
        <>
          {legend.length > 0 && (
            <div className="flex-none border-t border-border px-3 py-2" data-testid="weapon-records-legend">
              <ChartLegend items={legend} align="center" />
            </div>
          )}
        </>
      }
    >
      <p className="px-3 pt-2 text-xs text-muted-foreground">{t('synthesis.weapon_records.subtitle')}</p>
      <div ref={hostRef} className="relative px-3 pb-2 pt-1" onMouseLeave={() => setHover(null)}>
        {layout.items.length === 0 ? (
          <p className="py-6 text-center text-sm text-muted-foreground" data-testid="weapon-records-empty">
            {t('synthesis.weapon_records.empty_all_excluded')}
          </p>
        ) : (
          <RulerSvg layout={layout} onMove={onMove} onOpen={openReplay} onKey={onKey} locale={locale} t={t} />
        )}
        {hover && <RecordTooltip hover={hover} locale={locale} t={t} />}
      </div>
    </SectionCard>
  )
}

interface RulerSvgProps {
  layout: ReturnType<typeof weaponRecordsLayout>
  locale: ManifestLocale
  t: (key: SynthesisKey, vars?: Record<string, unknown>) => string
  onMove: (item: RulerItem) => (e: MouseEvent<SVGGElement>) => void
  onOpen: (row: WeaponDistanceRecordRow) => void
  onKey: (row: WeaponDistanceRecordRow) => (e: KeyboardEvent<SVGGElement>) => void
}

/** Le dessin : graduations, axe, puis un groupe cliquable par arme (rappel, deux textes, losange). */
function RulerSvg({ layout, locale, t, onMove, onOpen, onKey }: RulerSvgProps) {
  const { width, height, axisY, ticks, x, items } = layout
  return (
    <svg
      viewBox={`0 0 ${width} ${height}`}
      width="100%"
      height={height}
      className="block overflow-visible text-[11px]"
      role="img"
      aria-label={t('synthesis.weapon_records.title')}
      data-testid="weapon-records-ruler"
    >
      {ticks.map((m) => (
        <g key={m} className="text-border">
          <line x1={x(m)} x2={x(m)} y1={8} y2={axisY} stroke="currentColor" strokeDasharray="2 4" />
          <text x={x(m)} y={axisY + 18} textAnchor="middle" fill="currentColor" className="text-[10px] text-muted-foreground">
            {formatMeters(m, locale).replace(/[,.]0 m$/, ' m')}
          </text>
        </g>
      ))}
      <line x1={x(0)} x2={x(layout.maxMeters)} y1={axisY} y2={axisY} stroke="currentColor" strokeWidth={1.5} className="text-muted-foreground" />
      {ticks.map((m) => (
        <line key={`tick-${m}`} x1={x(m)} x2={x(m)} y1={axisY - 4} y2={axisY + 4} stroke="currentColor" className="text-muted-foreground" />
      ))}
      {items.map((it) => {
        const ly = labelBaselineY(layout, it)
        const color = tokenCssVar(fragClassToken(it.row.class))
        // Le trait de rappel relie le losange au libellé, vers le haut ou vers le bas.
        const leader =
          it.side === 'top'
            ? { y1: ly + 4, y2: axisY - DIAMOND_PX - 1 }
            : { y1: axisY + DIAMOND_PX + 1, y2: ly - 10 }
        return (
          <g
            key={it.row.weapon_key}
            role="link"
            tabIndex={0}
            className="cursor-pointer outline-none focus-visible:opacity-80"
            aria-label={t('synthesis.weapon_records.open_replay_label', { weapon: it.label, record: it.valueText })}
            onMouseMove={onMove(it)}
            onClick={() => onOpen(it.row)}
            onKeyDown={onKey(it.row)}
            data-testid="weapon-record-item"
            data-side={it.side}
          >
            <line x1={it.cx} x2={it.cx} y1={leader.y1} y2={leader.y2} stroke="currentColor" className="text-border" />
            <text x={it.labelX} y={ly - 1} textAnchor="middle" fill="currentColor" className="text-foreground">
              {it.label}
            </text>
            <text x={it.labelX} y={ly + 11} textAnchor="middle" fill="currentColor" className="text-[10px] tabular-nums text-muted-foreground">
              {it.valueText}
            </text>
            <polygon
              points={`${it.cx},${axisY - DIAMOND_PX} ${it.cx + DIAMOND_PX},${axisY} ${it.cx},${axisY + DIAMOND_PX} ${it.cx - DIAMOND_PX},${axisY}`}
              fill={color}
              // Liseré de la couleur de la carte : deux losanges voisins restent distincts.
              stroke="var(--card)"
              strokeWidth={1}
            />
          </g>
        )
      })}
    </svg>
  )
}

interface RecordTooltipProps {
  hover: { item: RulerItem; x: number; y: number }
  locale: ManifestLocale
  t: (key: SynthesisKey, vars?: Record<string, unknown>) => string
}

/** L'infobulle : l'arme, le record, la médiane et l'effectif, le match, l'invite au clic. */
function RecordTooltip({ hover, locale, t }: RecordTooltipProps) {
  const { row } = hover.item
  const map = (locale === 'en' ? row.record.map_label_en : row.record.map_label) || row.record.map_label_en || ''
  const date = row.record.started_at ? formatDate(row.record.started_at, intlLocale(locale)) : ''
  const where = [map, date].filter(Boolean).join(' · ')
  return (
    <div
      role="tooltip"
      className="pointer-events-none absolute z-10 max-w-[18rem] rounded-md border border-border bg-popover px-2.5 py-2 text-xs text-popover-foreground shadow-lg"
      style={{ left: hover.x + 14, top: hover.y + 14 }}
      data-testid="weapon-records-tooltip"
    >
      <div>
        <span className="font-semibold">{hover.item.label}</span>
        <span className="text-muted-foreground"> — {t('synthesis.weapon_records.tooltip_record')}</span>
      </div>
      <div>
        <span className="font-semibold tabular-nums">{hover.item.valueText}</span>
        <span className="text-muted-foreground">
          {' · '}
          {t('synthesis.weapon_records.tooltip_context', { median: formatMeters(row.median_m, locale), measured: row.measured })}
        </span>
      </div>
      {where && <div className="text-muted-foreground">{where}</div>}
      <div className="text-muted-foreground">{t('synthesis.weapon_records.tooltip_open')}</div>
    </div>
  )
}
