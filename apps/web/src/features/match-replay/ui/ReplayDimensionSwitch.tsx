/**
 * ReplayDimensionSwitch — LE SEGMENT « 2D / 3D » posé à gauche du bouton « Vue », dans le coin haut droit de la
 * toile (`ReplayViewMenu`). Un choix exclusif à deux positions (`role="radiogroup"`), au même gabarit que le
 * bouton « Vue » qu'il jouxte.
 *
 * IL NE SE REND QUE SI LA VUE 3D EST OFFERTE (`available`, lu par `useReplayDimension`) : sans elle rien n'est
 * rendu, ni segment inerte, ni espace. Il ne porte aucune logique de vue 3D : il montre `value` et appelle
 * `onChange`. Les libellés « 2D » et « 3D » s'écrivent pareil dans toutes les langues ; seul le nom du groupe
 * est traduit.
 */
import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'
import type { ReplayDimension, ReplayDimensionState } from '../settings/useReplayDimension'

const POSITIONS: readonly { value: ReplayDimension; label: string }[] = [
  { value: '2d', label: '2D' },
  { value: '3d', label: '3D' },
]

export function ReplayDimensionSwitch({
  available, value, onChange, locale,
}: ReplayDimensionState & { locale: ReplayLocale }) {
  if (!available) return null
  return (
    <div
      role="radiogroup"
      aria-label={REPLAY_TEXT[locale].frames.dimensionGroup}
      className="inline-flex h-8 items-center gap-0.5 rounded-md border border-border bg-card/85 p-0.5 backdrop-blur-sm"
    >
      {POSITIONS.map((p) => (
        <button
          key={p.value}
          type="button"
          role="radio"
          aria-checked={value === p.value}
          onClick={() => onChange(p.value)}
          className={`h-full cursor-pointer rounded px-2.5 text-xs font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring ${
            value === p.value
              ? 'bg-primary text-primary-foreground'
              : 'text-muted-foreground hover:bg-accent hover:text-accent-foreground'
          }`}
        >
          {p.label}
        </button>
      ))}
    </div>
  )
}
