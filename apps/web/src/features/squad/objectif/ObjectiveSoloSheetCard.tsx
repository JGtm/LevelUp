/**
 * ObjectiveSoloSheetCard — « Ma part à l'objectif » (Séries temporelles › Usages, bloc « Objectif » ;
 * plan PLAN_TIMESERIES_USAGES_EMPRISE_2026-10-05, maquette v4).
 *
 * LA fiche du joueur affiché (gamertag et emblème, initiale sans emblème), dans la coquille des
 * fiches de l'Escouade (`SquadPlayerSheet`) : « Rôle dominant » en haut à droite ; les familles de
 * mode CÔTE À CÔTE ; par action, une barre = ma part du total de mon camp et la valeur au bout (les
 * temps en durée) ; un zéro reste une ligne, atténuée ; au pied, « n Prendre · n Défendre · durée
 * Tenir ». Aucune fiche « reste de mon camp » : la page est celle d'un joueur.
 */
import { Tooltip } from '@/components/ui/tooltip'

import { squadPlayerInk } from '../formes/colors'
import type { FormesText } from '../formes/i18n'
import { OBJECTIVE_ROLES, type ObjectiveRole } from '../formes/model/objectives'
import { SquadPlayerSheet, SquadSheetAvatar, SquadSheetSection } from '../SquadPlayerSheet'
import type { SoloObjectiveSheet, SoloSheetLine } from './objectif.logic'
import { ObjectifFrame } from './ObjectifFrame'

/** Les textes de la fiche solo (fournis par la page qui la monte). */
export interface SoloSheetText {
  title: string
  info: string
  dominantRole: string
  roles: Record<ObjectiveRole, string>
  pctFmt: (v: number) => string
  durationFmt: (seconds: number) => string
  /** « JGtm · Drapeau\nVols de drapeau : 3 des 14 de mon camp (21,4 %) ». */
  lineTip: (player: string, family: string, column: string, value: string, camp: string, pct: string | null) => string
}

interface Props {
  sheet: SoloObjectiveSheet
  name: string
  emblemUrl?: string
  familyLabel: (family: string) => string
  columns: FormesText['columns']
  t: SoloSheetText
  /**
   * Vue compacte du tiroir de comparaison de Sessions (maquette `makeSheets` avec `cp`) : le nombre au
   * bout d'une action et le pied deviennent ma part du total de mon camp (« — » quand mon camp n'a
   * rien fait) ; les comptes restent dans l'infobulle. `t.pctFmt` écrit la part.
   */
  compact?: boolean
}

const INK = squadPlayerInk(0)
const NO_SHARE = '—'

export function ObjectiveSoloSheetCard({ sheet, name, emblemUrl, familyLabel, columns, t, compact = false }: Props) {
  const fmt = (v: number, duration: boolean) => (duration ? t.durationFmt(v) : String(Math.round(v * 10) / 10))
  const shareText = (part: number, camp: number) => (camp > 0 ? t.pctFmt((part / camp) * 100) : NO_SHARE)
  return (
    <ObjectifFrame title={t.title} info={t.info} testId="objective-solo-sheet">
      <SquadPlayerSheet
        testId={`objective-solo-sheet-${sheet.xuid}`}
        className="min-w-0"
        color={INK}
        name={name}
        avatar={<SquadSheetAvatar label={name} color={INK} emblemUrl={emblemUrl} />}
        dominant={sheet.dominant ? { caption: t.dominantRole, label: t.roles[sheet.dominant] } : undefined}
        footer={
          <span className="contents" data-testid="objective-solo-foot">
            {OBJECTIVE_ROLES.map((role, ri) => (
              <span key={role}>
                <span className="font-medium text-foreground">
                  {compact ? shareText(sheet.roleTotals[ri], sheet.campRoleTotals[ri]) : fmt(sheet.roleTotals[ri], role === 'hold')}
                </span>{' '}
                {t.roles[role]}
              </span>
            ))}
          </span>
        }
      >
        <div className="grid gap-x-7 gap-y-3.5" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(260px, 1fr))' }}>
          {sheet.families.map((f) => (
            <SquadSheetSection key={f.family} label={familyLabel(f.family)}>
              <div className="flex flex-col gap-1">
                {f.lines.map((line) => (
                  <SoloLine
                    key={line.key}
                    line={line}
                    label={columns[line.key] ?? line.key}
                    tip={t.lineTip(
                      name,
                      familyLabel(f.family),
                      columns[line.key] ?? line.key,
                      fmt(line.value, line.duration),
                      fmt(line.camp, line.duration),
                      line.share != null ? t.pctFmt(line.share * 100) : null,
                    )}
                    text={compact ? shareText(line.value, line.camp) : fmt(line.value, line.duration)}
                  />
                ))}
              </div>
            </SquadSheetSection>
          ))}
        </div>
      </SquadPlayerSheet>
    </ObjectifFrame>
  )
}

function SoloLine({ line, label, tip, text }: { line: SoloSheetLine; label: string; tip: string; text: string }) {
  const zero = line.value === 0
  return (
    <Tooltip className="w-full" content={tip}>
      <div
        className={`grid min-h-5 w-full grid-cols-[minmax(0,1fr)_56px_50px] items-center gap-2 text-[12.5px] ${zero ? 'text-muted-foreground' : ''}`}
        data-testid={`objective-solo-line-${line.key}`}
        data-zero={zero ? 'true' : undefined}
      >
        <span className="truncate">{label}</span>
        <span className="relative block h-2 overflow-hidden rounded-[2px] bg-muted">
          {!zero && line.share != null && (
            <span
              className="absolute inset-y-0 left-0 rounded-[2px]"
              style={{ width: `${line.share * 100}%`, backgroundColor: INK }}
              data-testid={`objective-solo-bar-${line.key}`}
            />
          )}
        </span>
        <span className={`text-right tabular-nums ${line.duration ? 'text-[11.5px] font-medium' : zero ? 'font-normal' : 'font-semibold'}`}>
          {text}
        </span>
      </div>
    </Tooltip>
  )
}
