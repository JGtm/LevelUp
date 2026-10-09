/**
 * PickupSheetsCard — « Répartition des prises dans l'escouade » (Escouade › Emprise, bloc « Rôles
 * dans l'escouade » ; lot L5.2 du plan PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26, maquette de
 * l'onglet, `renderRoleCards`).
 *
 * Au-dessus des fiches, la ligne « Bonus perdus » : pour chaque camp, sa pastille d'équipe
 * (`team-ally` / `team-enemy`, S4 : pas de « nous » / « eux ») puis « 2 sur 12 (17 %) ».
 * Une fiche par joueur de l'escouade (jamais de fiche du reste du camp : les joueurs inconnus
 * n'ont pas de fiche, leurs prises restent comptées dans celles de l'équipe), dans la coquille des
 * fiches de médailles (`SquadPlayerSheet`, S7) : « Ressource dominante » en haut à droite (celle où le
 * joueur pèse le plus dans les prises de notre camp) ; une section par ressource, pastille de sa
 * couleur devant le nom (S8) ; les MÊMES objets dans le même ordre sur toutes les fiches, un
 * zéro reste une ligne atténuée avec un trait pointillé ; UNE PASTILLE PAR PRISE à la couleur du
 * joueur, pastille VIDE = bonus perdu (gardé sans l'activer ou lâché en mourant) ; au pied,
 * « 3 bonus · 9 armes spéciales ». Rôles sans jugement (S10) : aucun classement, aucune couleur
 * de valeur.
 */
import { useMemo, type CSSProperties } from 'react'

import { Tooltip } from '@/components/ui/tooltip'
import { tokenCssVar } from '@/lib/accessibility'

import { ObjectifFrame, ObjectifLegend } from '../objectif/ObjectifFrame'
import { SquadPlayerSheet, SquadSheetAvatar, SquadSheetSection } from '../SquadPlayerSheet'
import type { PickupLine, PickupLosses, PickupSheets } from './emprise.logic'
import type { EmpriseText } from './empriseStrings'
import { resourceInk } from './resourceColors'
import { TipText } from './TipText'

/** Ce que la carte sait de chaque fiche, dans l'ordre des fiches du modèle. */
export interface PickupIdentity {
  label: string
  color: string
  emblemUrl?: string
}

interface Props {
  sheets: PickupSheets
  identities: PickupIdentity[]
  /** Le nom d'un objet (bonus nommé par le web, arme par le titre). */
  itemName: (line: PickupLine) => string
  /** Escouade › Emprise : les fiches à même la section (titre et aide sur son intertitre). */
  bare?: boolean
  t: EmpriseText
}

export function PickupSheetsCard({ sheets, identities, itemName, bare = false, t }: Props) {
  const legend = useMemo(() => {
    const fg = 'var(--foreground)' // color-allow: encre neutre des marques de légende (maquette)
    return (
      <ObjectifLegend
        ariaLabel={t.sheets.title}
        items={[
          { kind: 'dot', label: t.sheets.legendTaken, color: fg },
          { kind: 'ring', label: t.sheets.legendLost, color: fg },
        ]}
      />
    )
  }, [t])

  return (
    <ObjectifFrame title={t.sheets.title} info={t.sheets.info} legend={legend} testId="emprise-sheets" bare={bare}>
      {sheets.losses && <LossesLine losses={sheets.losses} t={t} />}
      <div
        className="grid grid-cols-1 gap-4 min-[561px]:grid-cols-2 min-[1001px]:[grid-template-columns:repeat(var(--sheets),minmax(0,1fr))]"
        style={{ '--sheets': Math.min(Math.max(sheets.owners.length, 2), 4) } as CSSProperties}
      >
        {sheets.owners.map((owner, si) => {
          const who = identities[si]
          const dom = sheets.dominant[si]
          const id = owner.xuid ?? 'rest'
          return (
            <SquadPlayerSheet
              key={id}
              testId={`emprise-sheet-${id}`}
              className="min-w-0"
              color={who.color}
              name={who.label}
              avatar={<SquadSheetAvatar label={who.label} color={who.color} emblemUrl={who.emblemUrl} />}
              dominant={dom ? { caption: t.sheets.dominant, label: t.resources[dom].label } : undefined}
              footer={sheets.sections.map((s) => (
                <span key={s.resource} data-testid={`emprise-sheet-foot-${id}-${s.resource}`}>
                  <span className="font-medium text-foreground">{s.totals[si]}</span> {t.resources[s.resource].footer}
                </span>
              ))}
            >
              {sheets.sections.map((s) => (
                <SquadSheetSection
                  key={s.resource}
                  label={
                    <span className="inline-flex items-center">
                      <span
                        className="mr-1.5 inline-block h-[9px] w-[9px] rounded-[2px]"
                        style={{ backgroundColor: resourceInk(s.resource) }}
                        aria-hidden
                      />
                      {t.resources[s.resource].label}
                    </span>
                  }
                >
                  <div className="flex flex-col">
                    {s.lines.map((line) => (
                      <SheetLine key={line.object.key} line={line} si={si} ownerId={id} who={who} name={itemName(line)} t={t} />
                    ))}
                  </div>
                </SquadSheetSection>
              ))}
            </SquadPlayerSheet>
          )
        })}
      </div>
    </ObjectifFrame>
  )
}

function SheetLine({
  line,
  si,
  ownerId,
  who,
  name,
  t,
}: {
  line: PickupLine
  si: number
  ownerId: string
  who: PickupIdentity
  name: string
  t: EmpriseText
}) {
  const n = line.taken[si]
  const lost = Math.min(n, line.kept[si] + line.dropped[si])
  const zero = n === 0
  const lostText = t.sheets.lostFmt(line.kept[si], line.dropped[si])
  return (
    <Tooltip className="w-full" content={<TipText text={t.sheets.lineTip(who.label, name, n, line.camp, lostText)} />}>
      <div
        className={`grid min-h-5 w-full grid-cols-[minmax(0,1fr)_auto_18px] items-center gap-2 text-[12.5px] ${
          zero ? 'text-muted-foreground' : ''
        }`}
        data-testid={`emprise-sheet-line-${ownerId}-${line.object.key}`}
        data-zero={zero ? 'true' : undefined}
      >
        <span className="truncate">{name}</span>
        <span className="flex flex-nowrap items-center justify-end gap-[3px]">
          {zero ? (
            <span className="inline-block w-[11px] border-t border-dashed border-muted-foreground" aria-hidden />
          ) : (
            Array.from({ length: n }, (_, i) => {
              const hollow = i >= n - lost
              return (
                <span
                  key={i}
                  className="inline-block h-[11px] w-[11px] rounded-full"
                  style={hollow ? { boxShadow: `inset 0 0 0 2px ${who.color}` } : { backgroundColor: who.color }}
                  data-dot={hollow ? 'lost' : 'taken'}
                  aria-hidden
                />
              )
            })
          )}
        </span>
        <span className={`text-right tabular-nums ${zero ? 'font-normal' : 'font-semibold'}`}>{n}</span>
      </div>
    </Tooltip>
  )
}

function LossesLine({ losses, t }: { losses: PickupLosses; t: EmpriseText }) {
  const camps = [
    { key: 'us', color: tokenCssVar('team-ally'), label: t.ourSide, ...losses.us },
    { key: 'them', color: tokenCssVar('team-enemy'), label: t.opponent, ...losses.them },
  ]
  return (
    <div className="flex flex-wrap gap-3.5 rounded-lg bg-muted px-2.5 py-1.5 text-xs text-muted-foreground" data-testid="emprise-losses">
      <span className="font-semibold text-foreground">{t.sheets.lossesTitle}</span>
      {camps.map((c) => (
        <span key={c.key} className="inline-flex items-center" data-testid={`emprise-losses-${c.key}`}>
          <span className="mr-[5px] inline-block h-[9px] w-[9px] rounded-[2px]" style={{ backgroundColor: c.color }} role="img" aria-label={c.label} />
          <span className="font-semibold text-foreground">{t.sheets.lossesFmt(c.lost, c.taken)}</span>
          {c.taken > 0 && <span>&nbsp;({t.pctIntFmt((c.lost / c.taken) * 100)})</span>}
        </span>
      ))}
    </div>
  )
}
