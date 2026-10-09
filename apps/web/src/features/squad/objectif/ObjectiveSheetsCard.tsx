/**
 * ObjectiveSheetsCard — « Répartition de l'objectif dans l'escouade » (Escouade › Emprise, sa
 * propre section ; lot L3 du plan PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26, maquette C3EW).
 *
 * Une fiche par joueur de l'escouade, posées à même la section comme les fiches de médailles (le
 * titre et l'aide sont sur l'intertitre de la section) et dans leur coquille (`SquadPlayerSheet`,
 * S7) : « Rôle dominant » en haut à droite (le rôle où le joueur pèse le plus dans notre camp) ;
 * par famille de mode (étiquette de section), une ligne par action avec une barre à l'échelle de
 * la MÊME ligne sur toutes les fiches et la valeur au bout (les temps en durée) ; un zéro reste
 * une ligne, atténuée ; au pied, « n Prendre · n Défendre · durée Tenir ». Rôles sans jugement
 * (S10) : aucun classement, aucune couleur de valeur — la barre prend la couleur de la fiche.
 */
import type { CSSProperties } from 'react'

import { Tooltip } from '@/components/ui/tooltip'

import type { FormesText } from '../formes/i18n'
import { OBJECTIVE_ROLES } from '../formes/model/objectives'
import { SquadPlayerSheet, SquadSheetAvatar, SquadSheetSection } from '../SquadPlayerSheet'
import type { ObjectiveSheets } from './objectif.logic'
import type { ObjectifText } from './objectifStrings'

/** Ce que la carte sait de chaque fiche, dans l'ordre des fiches du modèle. */
export interface SheetIdentity {
  label: string
  color: string
  emblemUrl?: string
}

interface Props {
  sheets: ObjectiveSheets
  identities: SheetIdentity[]
  familyLabel: (family: string) => string
  columns: FormesText['columns']
  t: ObjectifText
}

export function ObjectiveSheetsCard({ sheets, identities, familyLabel, columns, t }: Props) {
  const fmt = (v: number, duration: boolean) => (duration ? t.durationFmt(v) : String(Math.round(v * 10) / 10))
  return (
    <div
      className="grid grid-cols-1 gap-4 min-[561px]:grid-cols-2 min-[1001px]:[grid-template-columns:repeat(var(--sheets),minmax(0,1fr))]"
      style={{ '--sheets': Math.min(Math.max(sheets.owners.length, 2), 4) } as CSSProperties}
      data-testid="objective-sheets"
    >
      {sheets.owners.map((owner, si) => {
        const who = identities[si]
        const dom = sheets.dominant[si]
        const totals = sheets.roleTotals[si]
        return (
          <SquadPlayerSheet
            key={owner.xuid}
            testId={`objective-sheet-${owner.xuid}`}
            className="min-w-0"
            color={who.color}
            name={who.label}
            avatar={<SquadSheetAvatar label={who.label} color={who.color} emblemUrl={who.emblemUrl} />}
            dominant={dom ? { caption: t.sheets.dominantRole, label: t.roles[dom] } : undefined}
            footer={OBJECTIVE_ROLES.map((role, ri) => (
              <span key={role}>
                <span className="font-medium text-foreground">{fmt(totals[ri], role === 'hold')}</span> {t.roles[role]}
              </span>
            ))}
          >
            {sheets.families.map((f) => (
              <SquadSheetSection key={f.family} label={familyLabel(f.family)}>
                <div className="flex flex-col gap-1">
                  {f.lines.map((line) => {
                    const v = line.values[si]
                    const label = columns[line.key] ?? line.key
                    const text = fmt(v, line.duration)
                    const zero = v === 0
                    return (
                      <Tooltip
                        key={line.key}
                        className="w-full"
                        content={t.sheets.lineTip(who.label, familyLabel(f.family), label, text)}
                      >
                        <div
                          className={`grid min-h-5 w-full grid-cols-[minmax(0,1fr)_56px_50px] items-center gap-2 text-[12.5px] ${
                            zero ? 'text-muted-foreground' : ''
                          }`}
                          data-testid={`objective-sheet-line-${owner.xuid}-${line.key}`}
                          data-zero={zero ? 'true' : undefined}
                        >
                          <span className="truncate">{label}</span>
                          <span className="relative block h-2 overflow-hidden rounded-[2px] bg-muted">
                            {!zero && line.max > 0 && (
                              <span
                                className="absolute inset-y-0 left-0 rounded-[2px]"
                                style={{ width: `${(v / line.max) * 100}%`, backgroundColor: who.color }}
                              />
                            )}
                          </span>
                          <span
                            className={`text-right tabular-nums ${
                              line.duration ? 'text-[11.5px] font-medium' : zero ? 'font-normal' : 'font-semibold'
                            }`}
                          >
                            {text}
                          </span>
                        </div>
                      </Tooltip>
                    )
                  })}
                </div>
              </SquadSheetSection>
            ))}
          </SquadPlayerSheet>
        )
      })}
    </div>
  )
}
