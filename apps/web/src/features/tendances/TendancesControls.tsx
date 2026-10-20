/**
 * TendancesControls — la barre de commandes de la page Tendances : la bascule de vue, le filtre
 * « Type de partie » et, en vue Escouade, le sélecteur d'escouade. Elle reste à l'écran pendant
 * un chargement ou une erreur : changer de vue ne doit pas faire disparaître la bascule qui
 * permet de revenir.
 */
import { Select } from '@/components/ui/select'
import type { TrendsMember } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'

import { getTendancesText } from './i18n'
import { gameTypeLabel } from './labels'
import type { TendancesView } from './tendances.logic'
import { TendancesSegmented } from './TendancesSegmented'
import { TendancesSquadPicker } from './TendancesSquadPicker'
import type { TendancesSquadSelection } from './useTendancesSquadSelection'

const GAME_TYPE_SELECT_ID = 'tendances-game-type'

const VIEWS: readonly TendancesView[] = ['solo', 'squad']

export interface TendancesControlsProps {
  playerSlug: string
  locale: Locale
  view: TendancesView
  onViewChange: (view: TendancesView) => void
  gameType: string
  /** Clés des types de partie proposés (celui qui est choisi y figure toujours). */
  gameTypeKeys: readonly string[]
  onGameTypeChange: (gameType: string) => void
  squad: TendancesSquadSelection
  /** Membres de la réponse courante (joueur principal en tête) : de quoi enregistrer l'escouade. */
  members: readonly TrendsMember[]
}

export function TendancesControls({
  playerSlug,
  locale,
  view,
  onViewChange,
  gameType,
  gameTypeKeys,
  onGameTypeChange,
  squad,
  members,
}: TendancesControlsProps) {
  const t = getTendancesText(locale)
  return (
    <div className="flex flex-wrap items-center gap-3">
      <TendancesSegmented
        options={VIEWS.map((v) => ({ value: v, label: t.viewLabel(v) }))}
        value={view}
        onChange={onViewChange}
        ariaLabel={t.viewAria}
      />
      <label htmlFor={GAME_TYPE_SELECT_ID} className="text-sm text-muted-foreground">
        {t.gameTypeLabel}
      </label>
      <div className="w-56">
        <Select
          id={GAME_TYPE_SELECT_ID}
          value={gameType}
          onChange={(event) => onGameTypeChange(event.target.value)}
        >
          <option value="">{t.allGameTypes}</option>
          {gameTypeKeys.map((key) => (
            <option key={key} value={key}>
              {gameTypeLabel(key, locale)}
            </option>
          ))}
        </Select>
      </div>
      {view === 'squad' && (
        <TendancesSquadPicker
          playerSlug={playerSlug}
          locale={locale}
          selected={squad.squadGamertags}
          onChange={squad.setSquadGamertags}
          exactComposition={squad.exactComposition}
          onExactCompositionChange={squad.setExactComposition}
          members={members}
        />
      )}
    </div>
  )
}
