/**
 * État "données insuffisantes" — < 30 matchs sur la fenêtre.
 *
 * Affiché à la place du PlayerProfileCard complet quand HasEnoughData=false.
 * Garde-fou G2 du plan §4.5.4 — pas de fausse précision.
 */
import { useProfileI18n } from '../../hooks/useProfileI18n'

interface InsufficientDataPlaceholderProps {
  matchesAnalyzed: number
  required: number
}

export function InsufficientDataPlaceholder({
  matchesAnalyzed,
  required,
}: InsufficientDataPlaceholderProps) {
  const { t } = useProfileI18n()
  const missing = Math.max(0, required - matchesAnalyzed)
  return (
    <section className="rounded-lg border border-dashed border-border bg-card p-6 text-center">
      <h2 className="mb-1 text-lg font-semibold">{t('profile.insufficient.title')}</h2>
      <p className="text-sm text-muted-foreground">
        {t('profile.insufficient.subtitle', { n: matchesAnalyzed })}
      </p>
      <p className="mt-2 text-sm">{t('profile.insufficient.cta', { missing })}</p>
    </section>
  )
}
