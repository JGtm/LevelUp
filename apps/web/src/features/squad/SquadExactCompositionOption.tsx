/**
 * SquadExactCompositionOption — la case « composition stricte » de la barre de filtres Escouade.
 *
 * Cochée par défaut (cf. `exactComposition.ts`) : seuls les matchs joués avec exactement les
 * coéquipiers cochés sont gardés. L'explication s'ouvre au survol et au focus, avec le même
 * composant d'infobulle que le reste de la page (`Tooltip`) ; une infobulle native `title` ne
 * s'ouvrait qu'après un délai, sans mise en forme, et pas au clavier.
 */
import { Tooltip } from '@/components/ui/tooltip'
import type { Locale } from '@/lib/i18n/locale'

import { getSquadText } from './i18n'

export interface SquadExactCompositionOptionProps {
  locale: Locale
  checked: boolean
  onChange: (value: boolean) => void
}

export function SquadExactCompositionOption({ locale, checked, onChange }: SquadExactCompositionOptionProps) {
  const t = getSquadText(locale)
  return (
    <Tooltip content={t.filter.exactCompositionTitle} className="shrink-0">
      <label className="inline-flex cursor-pointer items-center gap-1.5 rounded-md border border-input bg-background px-2.5 py-1 text-xs font-medium text-foreground transition-colors hover:bg-muted">
        <input
          type="checkbox"
          className="h-3 w-3 rounded border-input text-primary focus:ring-1 focus:ring-ring"
          checked={checked}
          onChange={(e) => onChange(e.target.checked)}
        />
        {t.filter.exactComposition}
      </label>
    </Tooltip>
  )
}
