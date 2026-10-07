/**
 * ProductionCard — « Frags obtenus avec les ressources » (Escouade › Emprise, bloc « Prendre, et
 * s'en servir », à gauche ; lot L5.2 du plan PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26,
 * maquette de l'onglet, `renderProductivite`).
 *
 * Par ressource produite (pastille de la ressource devant son nom, S8) :
 *   - la barre ÉPAISSE : les frags de notre camp contre ceux de l'adversaire, sur la MÊME
 *     population que la barre fine (`production.logic.ts`), « compte · part » dans chaque
 *     segment, repli S3 au-dessus, trait 50 % (forme `PisteCampsForm`) ;
 *   - juste dessous, la barre FINE : la part de l'exposition (temps d'effet pour les bonus,
 *     prises pour les armes spéciales), puis la ligne d'exposition (« temps d'effet : 2 min 39 ·
 *     58,5 % … 1 min 53 »).
 * Axe 0-100 % sous les pistes ; légende en pied de carte, centrée (S2). Sans exposition mesurée
 * (Halo 5, sans film : D10), la barre épaisse seule.
 */
import { useMemo } from 'react'

import { tokenCssVar } from '@/lib/accessibility'

import { ObjectifFrame, ObjectifLegend } from '../objectif/ObjectifFrame'
import { RESOURCE_ORDER } from './emprise.logic'
import type { EmpriseText } from './empriseStrings'
import { PisteCampsForm, ThinCampTrack, type PisteCampsRow } from './PisteCampsForm'
import type { ProductionRow } from './production.logic'
import { resourceInk } from './resourceColors'

/**
 * Vue compacte du tiroir de comparaison de Sessions : les parts entières seules dans les segments, et
 * la ligne d'exposition écrite en parts par `exposureLine` (« prises sur les socles : 38 % »).
 */
interface ProductionCompact {
  exposureLine: (name: string, pct: string) => string
}

/**
 * Une ressource sans mesure d'un côté (Vue match) : la raison à la place de la barre épaisse, et la
 * barre fine de l'exposition quand elle existe.
 */
export interface ProductionPending {
  resource: string
  text: string
  exposure?: ProductionRow['exposure']
}

interface Props {
  rows: ProductionRow[]
  t: EmpriseText
  compact?: ProductionCompact
  /** Lignes « non mesuré » (Vue match), rangées avec les autres dans l'ordre des ressources. */
  pending?: ProductionPending[]
  /** Une ligne atténuée sous la barre d'une ressource (Vue match : « aucune prise d'arme spéciale mesurée »). */
  notes?: Partial<Record<string, string>>
}

export function ProductionCard({ rows, t, compact, pending, notes }: Props) {
  const legend = useMemo(
    () => (
      <ObjectifLegend
        ariaLabel={t.production.title}
        items={[
          { kind: 'square', label: t.ourSide, color: tokenCssVar('team-ally') },
          { kind: 'square', label: t.opponent, color: tokenCssVar('team-enemy') },
          { kind: 'parity', label: t.parity, color: tokenCssVar('warning') },
          ...(rows.some((r) => r.exposure) || (pending ?? []).some((p) => p.exposure)
            ? [{ kind: 'thin' as const, label: t.production.thinLegend, color: tokenCssVar('team-ally') }]
            : []),
        ]}
      />
    ),
    [rows, pending, t],
  )
  const pistes = useMemo<PisteCampsRow[]>(() => {
    const out = [
      ...rows.map((r) => withNote(pisteOf(r, t, compact), notes?.[r.resource])),
      ...(pending ?? []).map((p) => pendingOf(p, t, compact)),
    ]
    return out.sort((a, b) => RESOURCE_ORDER.indexOf(a.key) - RESOURCE_ORDER.indexOf(b.key))
  }, [rows, pending, notes, t, compact])
  return (
    <ObjectifFrame title={t.production.title} info={t.production.info} legend={legend} testId="emprise-production">
      <div className="mt-2" aria-label={t.production.ariaLabel} role="group">
        <PisteCampsForm rows={pistes} pctFmt={compact ? t.pctIntFmt : t.pctFmt} axisMaxLabel={t.pctIntFmt(100)} pctOnly={!!compact} />
      </div>
    </ObjectifFrame>
  )
}

function pisteOf(r: ProductionRow, t: EmpriseText, compact: ProductionCompact | undefined): PisteCampsRow {
  const res = t.resources[r.resource]
  const sub = res.productionSub
  const n = r.kills.us + r.kills.them
  const share = (r.kills.us / n) * 100
  return {
    key: r.resource,
    label: res.label,
    sublabel: sub,
    dot: resourceInk(r.resource),
    us: r.kills.us,
    them: r.kills.them,
    usTip: t.production.segmentTip(t.ourSide, sub, r.kills.us, n, t.pctFmt(share)),
    themTip: t.production.segmentTip(t.opponent, sub, r.kills.them, n, t.pctFmt(100 - share)),
    below: r.exposure ? <ExposureLines exposure={r.exposure} resource={r.resource} t={t} compact={compact} /> : undefined,
  }
}

function pendingOf(p: ProductionPending, t: EmpriseText, compact: ProductionCompact | undefined): PisteCampsRow {
  const res = t.resources[p.resource]
  return {
    key: p.resource,
    label: res.label,
    sublabel: res.productionSub,
    dot: resourceInk(p.resource),
    us: 0,
    them: 0,
    usTip: '',
    themTip: '',
    pending: p.text,
    below: p.exposure ? <ExposureLines exposure={p.exposure} resource={p.resource} t={t} compact={compact} /> : undefined,
  }
}

function withNote(row: PisteCampsRow, note: string | undefined): PisteCampsRow {
  if (!note) return row
  return {
    ...row,
    below: (
      <>
        {row.below}
        <div className="text-[11px] text-muted-foreground" data-testid={`emprise-production-note-${row.key}`}>
          {note}
        </div>
      </>
    ),
  }
}

/** La barre fine de l'exposition et sa ligne de valeurs (maquette : `.track.thin` puis `.pexpo`). */
function ExposureLines({
  exposure,
  resource,
  t,
  compact,
}: {
  exposure: NonNullable<ProductionRow['exposure']>
  resource: string
  t: EmpriseText
  compact: ProductionCompact | undefined
}) {
  const ex = t.production.exposure[exposure.kind]
  if (!ex) return null
  const { us, them } = exposure.value
  const share = (us / (us + them)) * 100
  const line = compact
    ? compact.exposureLine(ex.name, t.pctIntFmt(share))
    : t.production.exposureLine(ex.name, ex.fmt(us), t.pctFmt(share))
  const right = compact ? t.pctIntFmt(100 - share) : ex.fmt(them)
  return (
    <>
      <ThinCampTrack
        us={us}
        them={them}
        label={`${t.resources[resource].label} · ${line} / ${right}`}
        usTip={t.production.thinTip(t.ourSide, ex.name, ex.fmt(us), t.pctFmt(share))}
        themTip={t.production.thinTip(t.opponent, ex.name, ex.fmt(them), t.pctFmt(100 - share))}
      />
      <div
        className="flex justify-between gap-2 text-[11px] tabular-nums text-muted-foreground"
        data-testid={`emprise-production-exposure-${resource}`}
      >
        <span>{line}</span>
        <span>{right}</span>
      </div>
    </>
  )
}
