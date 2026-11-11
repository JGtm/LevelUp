/**
 * MatchLeadShare — LA MAINMISE DU MATCH : la part de la durée jouée passée en tête du score par
 * l'équipe du joueur, et le nombre de retournements. Deux montages : le bloc de l'en-tête (entre
 * Performance et Rang) et le rappel sous la carte « Écart de score », qui en est la lecture
 * graphique.
 *
 * La mesure n'est montrée que si le match a un rejeu (elle vient du score du film) ou porte déjà
 * un chiffre. Un match sans mesure (mode à peu de points : drapeau, colline, bombe) garde le
 * bloc avec un TIRET et l'infobulle dit pourquoi — jamais « 0 % ».
 */
import { InfoTooltip } from '@/components/ui/info-tooltip'
import { formatLeadChanges, formatLeadShare } from '@/lib/charts/leadShare/model'
import { LEAD_SHARE_TEXT } from '@/lib/charts/leadShare/text'

import type { MatchViewLocale } from './i18n'
import type { LeadShareData } from './matchLeadShare.logic'

interface LeadShareBlockProps extends LeadShareData {
  locale: MatchViewLocale
}

/** Le bloc de l'en-tête : nom + aide, grand chiffre ou tiret, retournements. */
export function LeadShareBlock({ leadShare, leadChanges, locale }: LeadShareBlockProps) {
  const text = LEAD_SHARE_TEXT[locale]
  const changes = formatLeadChanges(leadChanges, text)
  return (
    <div className="flex flex-col items-center" data-testid="match-lead-share">
      <span className="inline-flex items-center gap-1 text-2xs font-semibold uppercase tracking-wider text-muted-foreground">
        {text.name}
        <InfoTooltip content={text.help} iconClass="h-3 w-3" />
      </span>
      <span
        className="text-4xl font-black leading-none tabular-nums"
        title={leadShare != null ? undefined : text.notMeasured}
        data-testid="match-lead-share-value"
      >
        {formatLeadShare(leadShare)}
      </span>
      {changes ? <span className="mt-1 text-xs text-muted-foreground">{changes}</span> : null}
    </div>
  )
}

/** Le rappel sous « Écart de score » : « Mainmise : 62 % · 3 retournements », ou le tiret. */
export function LeadShareRecall({ leadShare, leadChanges, locale }: LeadShareBlockProps) {
  const text = LEAD_SHARE_TEXT[locale]
  const changes = formatLeadChanges(leadChanges, text)
  return (
    <p
      className="px-1 text-xs text-muted-foreground tabular-nums"
      title={leadShare != null ? undefined : text.notMeasured}
      data-testid="match-lead-share-recall"
    >
      <span className="font-semibold">{text.name}</span> : {formatLeadShare(leadShare)}
      {changes ? ` · ${changes}` : ''}
    </p>
  )
}
