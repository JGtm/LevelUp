import { Tooltip } from '@/components/ui/tooltip'

/**
 * Indicateur discret de fraîcheur d'une donnée — icône (i) avec tooltip au survol.
 *
 * Affiche une phrase complète (passée par le parent pour i18n) datée du dernier
 * snapshot connu. Utilisé sur les panneaux Battle Pass / Défis pour signaler
 * l'âge des données quand le live API est indisponible (proxy d'entreprise,
 * tokens expirés…).
 *
 * Le format de date est confié au parent (locale-dependent) : on reçoit la
 * date déjà formatée dans le message.
 */
const DEFAULT_COLOR_CLASS = 'text-muted-foreground/35 hover:text-muted-foreground/80'

export function DataFreshnessIndicator({
  snapshotAt,
  buildLabel,
  locale,
  className,
}: {
  /** Date ISO 8601 (RFC3339) du snapshot, ou null/undefined si inconnu. */
  snapshotAt: string | null | undefined
  /** Construit le texte du tooltip à partir de la date formatée. */
  buildLabel: (formattedDate: string) => string
  /** Locale BCP-47 pour le formatage de la date (ex. 'fr-FR'). */
  locale: string
  /**
   * Classes de couleur Tailwind appliquées à l'icône (override). Doit inclure
   * la variante hover si on veut un feedback au survol. Par défaut : ton très
   * discret `muted-foreground/35` → `/80` au hover.
   */
  className?: string
}) {
  if (!snapshotAt) return null
  const date = new Date(snapshotAt)
  if (Number.isNaN(date.getTime())) return null

  const formatted = date.toLocaleString(locale, {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
  const label = buildLabel(formatted)
  const colorClass = className ?? DEFAULT_COLOR_CLASS

  return (
    <Tooltip content={label}>
      <span
        role="img"
        aria-label={label}
        data-testid="data-freshness-indicator"
        className={`inline-flex h-3.5 w-3.5 cursor-help items-center justify-center transition-colors ${colorClass}`}
      >
        <svg
          aria-hidden="true"
          viewBox="0 0 16 16"
          fill="none"
          stroke="currentColor"
          strokeWidth="1.4"
          strokeLinecap="round"
          strokeLinejoin="round"
          className="h-full w-full"
        >
          <circle cx="8" cy="8" r="6.5" />
          <line x1="8" y1="7.2" x2="8" y2="11.5" />
          <circle cx="8" cy="5" r="0.6" fill="currentColor" stroke="none" />
        </svg>
      </span>
    </Tooltip>
  )
}
