/**
 * MatchResourceControlCard — « Contrôle des ressources, par match » (Vue match, carte D ; plan
 * PLAN_MATCHVIEW_EMPRISE_2026-10-06, D5, D6, D13).
 *
 * Par ressource du match (bonus, armes spéciales, véhicules, armes de râtelier) : la piste de l'équipe
 * contre l'adversaire (forme `PisteCampsForm` : « compte · part » dans chaque segment, trait 50 %),
 * puis une piste en retrait par objet pris ; les armes de râtelier sont repliées derrière un bouton.
 * Sous les pistes, la ligne des prises sur un emplacement non identifié quand il y en a. Le modèle
 * vient de `buildMatchControl` ; ce composant ne fait que nommer et tracer.
 */
import { useMemo, useState } from 'react'

import { ObjectifFrame, ObjectifLegend } from '@/features/squad/objectif/ObjectifFrame'
import { RESOURCE_POWERUP, RESOURCE_RACK } from '@/features/squad/emprise/emprise.logic'
import type { EmpriseText } from '@/features/squad/emprise/empriseStrings'
import { PisteCampsForm, type PisteCampsRow } from '@/features/squad/emprise/PisteCampsForm'
import { resourceInk } from '@/features/squad/emprise/resourceColors'
import type { SquadEmpriseObject } from '@/lib/api/types'
import { tokenCssVar } from '@/lib/accessibility'

import type { MatchControl, MatchControlRow } from './matchEmprise.logic'
import type { MatchOwnText } from './matchEmpriseText'

interface Props {
  control: MatchControl
  objectName: (o: SquadEmpriseObject) => string
  t: EmpriseText
  own: MatchOwnText
}

export function MatchResourceControlCard({ control, objectName, t, own }: Props) {
  const [racksOpen, setRacksOpen] = useState(false)
  const legend = useMemo(
    () => (
      <ObjectifLegend
        ariaLabel={t.control.title}
        items={[
          { kind: 'square', label: t.ourSide, color: tokenCssVar('team-ally') },
          { kind: 'square', label: t.opponent, color: tokenCssVar('team-enemy') },
          { kind: 'parity', label: t.parity, color: tokenCssVar('warning') },
        ]}
      />
    ),
    [t],
  )
  const rows = useMemo<PisteCampsRow[]>(
    () =>
      control.rows
        .filter((r) => !(r.object && r.resource === RESOURCE_RACK && !racksOpen))
        .map((r) => pisteOf(r, { objectName, t, own, racks: control.racks, racksOpen, toggle: () => setRacksOpen((v) => !v) })),
    [control, objectName, t, own, racksOpen],
  )
  const u = control.unclassified
  return (
    <ObjectifFrame title={t.control.title} info={t.control.info} legend={legend} testId="match-emprise-control">
      <div className="mt-2" aria-label={t.control.ariaLabel} role="group">
        <PisteCampsForm rows={rows} pctFmt={t.pctFmt} axisMaxLabel={t.pctIntFmt(100)} />
        {u && (
          <p className="mt-2 text-[11px] text-muted-foreground" data-testid="match-emprise-unclassified">
            {own.unclassified(u.us + u.them, u.us, u.them)}
          </p>
        )}
      </div>
    </ObjectifFrame>
  )
}

interface PisteContext {
  objectName: (o: SquadEmpriseObject) => string
  t: EmpriseText
  own: MatchOwnText
  racks: number
  racksOpen: boolean
  toggle: () => void
}

function pisteOf(r: MatchControlRow, x: PisteContext): PisteCampsRow {
  const res = x.t.resources[r.resource]
  const label = r.object ? x.objectName(r.object) : res.label
  const sub = r.object ? undefined : r.resource === RESOURCE_POWERUP && (r.padsEmptied ?? 0) > 0 ? x.own.powerupSub(r.padsEmptied ?? 0) : res.gridSub
  const n = r.us + r.them
  const share = n > 0 ? (r.us / n) * 100 : 0
  const tipSub = sub ?? res.gridSub
  return {
    key: r.key,
    label,
    sublabel: sub,
    dot: r.object ? undefined : resourceInk(r.resource),
    us: r.us,
    them: r.them,
    usTip: x.t.control.segmentTip(x.t.ourSide, label, tipSub, r.us, n, x.t.pctFmt(share)),
    themTip: x.t.control.segmentTip(x.t.opponent, label, tipSub, r.them, n, x.t.pctFmt(100 - share)),
    indent: !!r.object,
    labelNode: !r.object && r.resource === RESOURCE_RACK && x.racks > 0 ? <RacksToggle label={label} x={x} /> : undefined,
  }
}

function RacksToggle({ label, x }: { label: string; x: PisteContext }) {
  return (
    <button
      type="button"
      className="inline-flex items-center text-left hover:underline"
      aria-expanded={x.racksOpen}
      onClick={x.toggle}
      data-testid="match-emprise-racks-toggle"
    >
      <span className="mr-1" aria-hidden>
        {x.racksOpen ? '▾' : '▸'}
      </span>
      <span className="mr-1.5 inline-block h-[9px] w-[9px] shrink-0 rounded-[2px]" style={{ backgroundColor: resourceInk(RESOURCE_RACK) }} aria-hidden />
      {label}
      <small className="ml-1 text-[11px] text-muted-foreground">{x.racksOpen ? x.own.racksUnfolded(x.racks) : x.own.racksFolded(x.racks)}</small>
    </button>
  )
}
