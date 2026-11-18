/**
 * ReplayAutoPlayToggle — L'INTERRUPTEUR DE LA LECTURE AUTOMATIQUE : il ne commande pas le lecteur ouvert,
 * il décide de son état de départ (cf. `ReplayTransportCommands.autoPlay`). Partagé par la barre complète
 * et la barre d'une ligne.
 */
import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'
import { AutoPlayIcon } from './ReplayTransportIcons'

export function ReplayAutoPlayToggle({
  autoPlay, onToggle, locale,
}: {
  autoPlay: boolean
  onToggle: () => void
  locale: ReplayLocale
}) {
  const t = REPLAY_TEXT[locale]
  return (
    <button
      type="button"
      onClick={onToggle}
      role="switch"
      aria-checked={autoPlay}
      aria-label={t.autoPlay}
      title={`${t.autoPlay} — ${t.autoPlayHint}`}
      className={`inline-flex h-[30px] cursor-pointer items-center gap-1.5 rounded-full px-2 text-[10.5px] font-medium transition-colors ${
        autoPlay
          ? 'bg-secondary text-secondary-foreground hover:bg-secondary/80'
          : 'text-muted-foreground hover:bg-accent hover:text-accent-foreground'
      }`}
    >
      <AutoPlayIcon on={autoPlay} />
    </button>
  )
}
