/**
 * TacticalReapparitionsPuce — la puce d'écart en points entre la part d'une zone pendant la tenue et sa part
 * sans condition : ▲ n (au-dessus), ▼ n (en dessous) ou = (identique). Le sens se lit aussi au texte
 * accessible, jamais à la seule couleur.
 */
import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'

import type { TacticalText } from './i18n'
import { puceDEcart } from './reapparitions.logic'

export function PuceDEcart({ t, ecart }: { t: TacticalText; ecart: number }) {
  const { sens, texte } = puceDEcart(ecart)
  const nom = sens === 'haut' ? t.reappDeltaAbove(ecart) : sens === 'bas' ? t.reappDeltaBelow(Math.abs(ecart)) : t.reappDeltaEqual
  const style =
    sens === 'haut'
      ? { color: tokenCssVar('team-enemy'), backgroundColor: `color-mix(in oklch, ${tokenCssVar('team-enemy')} 14%, transparent)` }
      : undefined
  return (
    <span
      className={`rounded px-1 py-px text-[10px] font-semibold leading-none ${sens === 'haut' ? '' : sens === 'bas' ? 'bg-muted text-muted-foreground' : 'text-muted-foreground'}`}
      style={style}
      title={nom}
      aria-label={nom}
      data-sens={sens}
    >
      {texte}
    </span>
  )
}
