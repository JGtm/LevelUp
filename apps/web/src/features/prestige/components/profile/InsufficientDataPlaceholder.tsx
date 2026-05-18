/**
 * État "données insuffisantes" — < 30 matchs sur la fenêtre.
 *
 * Affiché à la place du PlayerProfileCard complet quand HasEnoughData=false.
 * Garde-fou G2 du plan §4.5.4 — pas de fausse précision.
 */
interface InsufficientDataPlaceholderProps {
  matchesAnalyzed: number
  required: number
}

export function InsufficientDataPlaceholder({
  matchesAnalyzed,
  required,
}: InsufficientDataPlaceholderProps) {
  const missing = Math.max(0, required - matchesAnalyzed)
  return (
    <section
      className="rounded-lg border border-dashed border-border bg-card p-6 text-center"
      aria-label="profile.insufficient_data"
    >
      <h2 className="mb-1 text-lg font-semibold">Profil en construction</h2>
      <p className="text-sm text-muted-foreground">
        {matchesAnalyzed} match{matchesAnalyzed > 1 ? 's' : ''} analysé
        {matchesAnalyzed > 1 ? 's' : ''} sur cette fenêtre.
      </p>
      <p className="mt-2 text-sm">
        Joue encore <span className="font-semibold">{missing}</span> match
        {missing > 1 ? 's' : ''} pour débloquer ton profil complet (style,
        engagement, leviers d&apos;amélioration).
      </p>
    </section>
  )
}
