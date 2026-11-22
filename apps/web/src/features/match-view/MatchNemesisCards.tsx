/**
 * MatchNemesisCards — match_view.12 (Némésis et Souffre-douleur).
 *
 * Deux cartes côte à côte sur fond bg-card (aligné sur les autres blocs) :
 *   - Némésis : adversaire qui m'a le plus éliminé.
 *   - Souffre-douleur : adversaire que j'ai le plus éliminé.
 *
 * Source : les éliminations de `combat_tab.highlight_events` (déjà chargées par la page),
 * lues au dernier instant par la fonction commune des antagonistes (`lib/antagonists`),
 * la même que le rejeu : le tenant du titre garde la carte sur une égalité. Seuls les
 * adversaires comptent (camp différent selon le tableau de score ; les bots sont exclus,
 * cf. `MatchNemesisCards.logic.ts`).
 */
import { tokenCssVar, type SemanticToken } from '@/lib/accessibility'
import type { MatchHighlightEvent, MatchScoreboardRow } from '@/lib/api/types'
import { themedIconSrc } from '@/lib/themedIcon'
import { useSettingsDraftStore } from '@/stores/settingsDraftStore'
import type { MatchViewText } from './i18n'
import { readMatchAntagonists, type MatchAntagonist } from './MatchNemesisCards.logic'

interface Props {
  events: readonly MatchHighlightEvent[]
  scoreboard: MatchScoreboardRow[] | null | undefined
  meXUID: string | null
  t: MatchViewText
}

export function MatchNemesisCards({ events, scoreboard, meXUID, t }: Props) {
  const theme = useSettingsDraftStore((state) => state.localUiPrefs.theme)

  // Pas de `return null` quand il n'y a aucun duel : on garde les deux cartes
  // titrées (Némésis / Souffre-douleur) qui affichent leur état vide
  // (`combatNoNemesis`) au lieu de faire disparaître la section.
  const { nemesis: nemesisRow, bully: bullyRow } = readMatchAntagonists(events, scoreboard ?? [], meXUID)

  return (
    <div className="grid grid-cols-2 gap-4">
      <NemesisCard
        title={t.combatNemesisTitle}
        accentToken="outcome-loss"
        row={nemesisRow}
        statLine={nemesisRow ? t.combatKilledMeFmt(nemesisRow.count) : null}
        logoSrc={themedIconSrc('nemesis', theme)}
        t={t}
      />
      <NemesisCard
        title={t.combatBullyTitle}
        accentToken="outcome-win"
        row={bullyRow}
        statLine={bullyRow ? t.combatIKilledFmt(bullyRow.count) : null}
        logoSrc={themedIconSrc('victim', theme)}
        t={t}
      />
    </div>
  )
}

interface CardProps {
  title: string
  accentToken: SemanticToken
  row: MatchAntagonist | null
  /** null = etat vide (aucun duel) : la ligne de stat est masquee plutot que d afficher un 0 absurde. */
  statLine: string | null
  logoSrc: string
  t: MatchViewText
}

function NemesisCard({ title, accentToken, row, statLine, logoSrc, t }: CardProps) {
  const accent = tokenCssVar(accentToken)
  return (
    <div
      className="relative overflow-hidden rounded-lg border border-border/60 bg-card p-5"
      style={{
        boxShadow: `inset 0 0 0 1px ${accent}33`,
      }}
    >
      <div
        className="absolute inset-y-0 left-0 w-1"
        style={{ background: accent }}
        aria-hidden="true"
      />
      <div className="flex items-center gap-3 pl-2">
        <img
          src={logoSrc}
          alt=""
          aria-hidden
          className="h-12 w-12 shrink-0 object-contain opacity-75"
        />
        <div className="min-w-0 flex-1">
          <div
            className="text-xs font-semibold uppercase tracking-wider"
            style={{ color: accent }}
          >
            {title}
          </div>
          <div className="mt-1 truncate text-2xl font-bold text-on-scrim">
            {row?.gamertag ?? t.combatNoNemesis}
          </div>
          {statLine !== null && (
            <div className="mt-3 text-sm text-on-scrim/80">
              <div className="tabular-nums">{statLine}</div>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
