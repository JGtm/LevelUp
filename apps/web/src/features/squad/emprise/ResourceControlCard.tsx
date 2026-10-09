/**
 * ResourceControlCard — « Contrôle des ressources » (Escouade › Emprise, bloc « Bilan de la
 * soirée », à gauche ; lot L5.2 du plan PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26, maquette
 * de l'onglet).
 *
 * Une piste par ressource du bilan (bonus, armes spéciales ; la liste du bloc), pastille de la
 * couleur de la ressource devant son nom (S8), notre camp contre l'adversaire avec « compte ·
 * part » dans chaque segment, trait 50 % (forme `PisteCampsForm`, L5.3), axe 0-100 % dessous ;
 * légende en pied de carte, centrée (S2).
 */
import { useMemo } from 'react'

import { tokenCssVar } from '@/lib/accessibility'

import { ObjectifFrame, ObjectifLegend } from '../objectif/ObjectifFrame'
import type { ControlRow } from './emprise.logic'
import type { EmpriseText } from './empriseStrings'
import { PisteCampsForm, type PisteCampsRow } from './PisteCampsForm'
import { resourceInk } from './resourceColors'

/**
 * `compact` (vue compacte du tiroir de comparaison de Sessions) : la part entière seule dans chaque
 * segment, le compte au survol ; les textes de la vue viennent de l'appelant (`t`).
 */
export function ResourceControlCard({ rows, t, compact = false }: { rows: ControlRow[]; t: EmpriseText; compact?: boolean }) {
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
  const pistes = useMemo<PisteCampsRow[]>(
    () =>
      rows.map((r) => {
        const res = t.resources[r.resource]
        const n = r.us + r.them
        return {
          key: r.resource,
          label: res.pisteTitle,
          sublabel: res.pisteSub,
          dot: resourceInk(r.resource),
          us: r.us,
          them: r.them,
          usTip: t.control.segmentTip(t.ourSide, res.pisteTitle, res.pisteSub, r.us, n, t.pctFmt(r.share * 100)),
          themTip: t.control.segmentTip(t.opponent, res.pisteTitle, res.pisteSub, r.them, n, t.pctFmt(100 - r.share * 100)),
        }
      }),
    [rows, t],
  )
  return (
    <ObjectifFrame title={t.control.title} info={t.control.info} legend={legend} testId="emprise-control">
      <div className="mt-2" aria-label={t.control.ariaLabel} role="group">
        <PisteCampsForm rows={pistes} pctFmt={compact ? t.pctIntFmt : t.pctFmt} axisMaxLabel={t.pctIntFmt(100)} pctOnly={compact} />
      </div>
    </ObjectifFrame>
  )
}
