/**
 * ReplayAutoPlayToggle — L'INTERRUPTEUR DE LA LECTURE AUTOMATIQUE : il ne commande pas le lecteur ouvert,
 * il décide de son état de départ (cf. `ReplayTransportCommands.autoPlay`). Partagé par la barre complète
 * et la barre d'une ligne.
 */
import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'
import { COMPACT_BUTTON_SQUARE } from './compactButton'
import { AutoPlayIcon } from './ReplayTransportIcons'

const compactClass = 'inline-flex shrink-0 cursor-pointer items-center justify-center transition-colors'

export function ReplayAutoPlayToggle({
  autoPlay, onToggle, locale, compact = false,
}: {
  autoPlay: boolean
  onToggle: () => void
  locale: ReplayLocale
  /** Le format de la barre d'une ligne : carré ; actif, il garde son fond. */
  compact?: boolean
}) {
  const t = REPLAY_TEXT[locale]
  const state = autoPlay
    ? 'bg-secondary text-secondary-foreground hover:bg-secondary/80'
    : 'text-muted-foreground hover:bg-accent hover:text-accent-foreground'
  return (
    <button
      type="button"
      onClick={onToggle}
      role="switch"
      aria-checked={autoPlay}
      aria-label={t.autoPlay}
      title={`${t.autoPlay} — ${t.autoPlayHint}`}
      style={compact ? COMPACT_BUTTON_SQUARE : undefined}
      className={compact ? `${compactClass} ${state}` : `inline-flex h-[30px] cursor-pointer items-center gap-1.5 rounded-full px-2 text-[10.5px] font-medium transition-colors ${state}`}
    >
      <AutoPlayIcon on={autoPlay} />
    </button>
  )
}
