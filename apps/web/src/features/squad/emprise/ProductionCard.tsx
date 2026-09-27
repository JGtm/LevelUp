/**
 * ProductionCard — « Frags obtenus avec les ressources » (Escouade › Emprise, bloc « Prendre, et
 * s'en servir », à gauche ; lot L5.2 du plan PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26,
 * maquette de l'onglet, `renderProductivite`).
 *
 * Par ressource produite (pastille de la ressource devant son nom, S8) :
 *   - la barre ÉPAISSE : les frags de notre camp contre ceux de l'adversaire, « compte · part »
 *     dans chaque segment, repli S3 au-dessus, trait 50 % (forme `PisteCampsForm`) ;
 *   - juste dessous, la barre FINE : la part de l'exposition (temps d'effet pour les bonus,
 *     prises pour les armes spéciales), puis la ligne d'exposition (« temps d'effet : 2 min 39 ·
 *     58,5 % … 1 min 53 »).
 * Axe 0-100 % sous les pistes ; légende en pied de carte, centrée (S2). Sans exposition mesurée
 * (Halo 5, sans film : D10), la barre épaisse seule.
 */
import { useMemo } from 'react'

import { tokenCssVar } from '@/lib/accessibility'

import { ObjectifFrame, ObjectifLegend } from '../objectif/ObjectifFrame'
import type { EmpriseText } from './empriseStrings'
import { PisteCampsForm, ThinCampTrack, type PisteCampsRow } from './PisteCampsForm'
import type { ProductionRow } from './production.logic'
import { resourceInk } from './resourceColors'

export function ProductionCard({ rows, t }: { rows: ProductionRow[]; t: EmpriseText }) {
  const legend = useMemo(
    () => (
      <ObjectifLegend
        ariaLabel={t.production.title}
        items={[
          { kind: 'square', label: t.ourSide, color: tokenCssVar('team-ally') },
          { kind: 'square', label: t.opponent, color: tokenCssVar('team-enemy') },
          { kind: 'parity', label: t.parity, color: tokenCssVar('warning') },
          ...(rows.some((r) => r.exposure)
            ? [{ kind: 'thin' as const, label: t.production.thinLegend, color: tokenCssVar('team-ally') }]
            : []),
        ]}
      />
    ),
    [rows, t],
  )
  const pistes = useMemo<PisteCampsRow[]>(() => rows.map((r) => pisteOf(r, t)), [rows, t])
  return (
    <ObjectifFrame title={t.production.title} info={t.production.info} legend={legend} testId="emprise-production">
      <div className="mt-2" aria-label={t.production.ariaLabel} role="group">
        <PisteCampsForm rows={pistes} pctFmt={t.pctFmt} axisMaxLabel={t.pctIntFmt(100)} />
      </div>
    </ObjectifFrame>
  )
}

function pisteOf(r: ProductionRow, t: EmpriseText): PisteCampsRow {
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
    below: r.exposure ? <ExposureLines exposure={r.exposure} resource={r.resource} t={t} /> : undefined,
  }
}

/** La barre fine de l'exposition et sa ligne de valeurs (maquette : `.track.thin` puis `.pexpo`). */
function ExposureLines({
  exposure,
  resource,
  t,
}: {
  exposure: NonNullable<ProductionRow['exposure']>
  resource: string
  t: EmpriseText
}) {
  const ex = t.production.exposure[exposure.kind]
  if (!ex) return null
  const { us, them } = exposure.value
  const share = (us / (us + them)) * 100
  const line = t.production.exposureLine(ex.name, ex.fmt(us), t.pctFmt(share))
  return (
    <>
      <ThinCampTrack
        us={us}
        them={them}
        label={`${t.resources[resource].label} · ${line} / ${ex.fmt(them)}`}
        usTip={t.production.thinTip(t.ourSide, ex.name, ex.fmt(us), t.pctFmt(share))}
        themTip={t.production.thinTip(t.opponent, ex.name, ex.fmt(them), t.pctFmt(100 - share))}
      />
      <div
        className="flex justify-between gap-2 text-[11px] tabular-nums text-muted-foreground"
        data-testid={`emprise-production-exposure-${resource}`}
      >
        <span>{line}</span>
        <span>{ex.fmt(them)}</span>
      </div>
    </>
  )
}
