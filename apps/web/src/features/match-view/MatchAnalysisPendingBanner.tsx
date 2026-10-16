/**
 * MatchAnalysisPendingBanner — bandeau INFO sous l'en-tête de la vue match tant que
 * l'analyse du film est en cours (`header.analysis_pending`).
 *
 * Teinte INFO, pas avertissement : rien n'est cassé, les données arrivent. `role="status"`
 * pour que le lecteur d'écran l'annonce sans interrompre. À droite, « Mise à jour
 * automatique » avec un point qui pulse (hors `prefers-reduced-motion`) : la vue se
 * recharge seule (cf. lib/analysis-pending/analysisPending.logic.ts).
 */
import type { MatchViewText } from './i18n'

export function MatchAnalysisPendingBanner({ t }: { t: MatchViewText }) {
  return (
    <div className="px-6 pt-4" data-testid="match-analysis-pending-banner">
      <div
        role="status"
        className="flex flex-wrap items-center gap-x-4 gap-y-2 rounded-md border border-info/50 bg-info/10 px-4 py-3"
      >
        <svg
          xmlns="http://www.w3.org/2000/svg"
          viewBox="0 0 20 20"
          fill="currentColor"
          className="h-4 w-4 shrink-0 text-info"
          aria-hidden="true"
        >
          <path
            fillRule="evenodd"
            d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a.75.75 0 000 1.5h.253a.25.25 0 01.244.304l-.459 2.066A1.75 1.75 0 0010.747 15H11a.75.75 0 000-1.5h-.253a.25.25 0 01-.244-.304l.459-2.066A1.75 1.75 0 009.253 9H9z"
            clipRule="evenodd"
          />
        </svg>
        <div className="min-w-0 flex-1">
          <p className="text-sm font-semibold text-foreground">{t.analysisPendingTitle}</p>
          <p className="mt-0.5 text-xs text-muted-foreground">{t.analysisPendingBody}</p>
        </div>
        <span className="inline-flex shrink-0 items-center gap-2 text-xs font-medium text-info">
          <span aria-hidden="true" className="relative flex h-2 w-2">
            <span className="absolute inline-flex h-full w-full rounded-full bg-info opacity-75 motion-safe:animate-ping" />
            <span className="relative inline-flex h-2 w-2 rounded-full bg-info" />
          </span>
          {t.analysisPendingAutoUpdate}
        </span>
      </div>
    </div>
  )
}
