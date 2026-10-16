/**
 * AnalysisPendingSlot — emplacement d'attente d'une section de la vue match qui dépend du
 * film (arme de chaque frag, rejeu 2D) tant que son analyse est en cours
 * (`header.analysis_pending`). Il remplace l'état vide habituel de la section : un bloc en
 * pointillés, un petit indicateur tournant et un libellé (« En cours d'analyse », « En
 * préparation »).
 *
 * L'indicateur ne tourne que hors `prefers-reduced-motion` (`motion-safe:`). Jetons
 * sémantiques uniquement.
 */
export function AnalysisPendingSlot({ label, className = '' }: { label: string; className?: string }) {
  return (
    <div
      data-testid="analysis-pending-slot"
      className={`flex items-center justify-center gap-2 rounded-xl border border-dashed border-info/40 bg-muted/40 px-4 py-6 text-sm italic text-muted-foreground ${className}`}
    >
      <span
        aria-hidden="true"
        className="inline-block h-4 w-4 shrink-0 rounded-full border-2 border-muted-foreground/30 border-t-info motion-safe:animate-spin"
      />
      <span>{label}</span>
    </div>
  )
}
