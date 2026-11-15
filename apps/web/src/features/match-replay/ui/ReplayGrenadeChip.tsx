/**
 * ReplayGrenadeChip — la puce d'UN type de grenade porté.
 *
 * La puce est la seule à connaître la composition vignette + stock ; la rangée
 * (`ReplayInventoryRow.tsx`) ne fait que la placer dans la boîte de grenades.
 */
import { WeaponIcon } from '@/components/ui/WeaponIcon'
import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'

import type { REPLAY_TEXT, ReplayLocale } from '../i18n/i18n'
import type { GrenadeIconRef } from './grenadeMask'
import { ReplayIconCount } from './ReplayIconCount'

type ReplayText = (typeof REPLAY_TEXT)[ReplayLocale]

/**
 * GrenadeChip — UN type de grenade porté : sa vignette, le stock en surimpression sur son coin
 * bas droit (`ReplayIconCount`, le rendu du compte de charges de la capacité), et l'encre
 * `warning` du type ÉQUIPÉ quand c'est lui qui partira au prochain lancer.
 *
 * LA VIGNETTE PORTE L'IDENTITÉ, LE NOM PASSE EN INFOBULLE (planche du 16/08 : « pas de
 * texte sauf pour le compteur »). Le chiffre est dans la boîte de la vignette : la puce ne
 * mesure que la vignette, et la boîte de grenades garde sa largeur quel que soit le nombre de
 * types portés. Sans AUCUNE image, le libellé revient, suivi du compte à plat : mieux vaut un
 * mot qu'un trou, et une surimpression sur un texte le rendrait illisible.
 */
export function GrenadeChip({
  carried,
  icon,
  iconPx,
  selected,
  t,
}: {
  carried: { rank: number; name: string; count: number }
  icon: GrenadeIconRef | null
  /** Côté de la vignette, du gabarit de la fiche. */
  iconPx: number
  /** Non nul = c'est CE type qui est équipé ; `read` dit si la lecture ou la déduction l'établit. */
  selected: { rank: number; read: boolean } | null
  t: ReplayText
}) {
  return (
    <span
      className={`inline-flex shrink-0 items-center gap-0.5 ${selected ? 'font-semibold' : ''}`}
      // LE TYPE ÉQUIPÉ SE DÉSIGNE PAR LA SEULE ENCRE `warning` : la vignette (masque teint par
      // currentColor) la prend, ni fond ni liseré. Le chiffre garde son rendu de pastille.
      style={selected ? { color: tokenCssVar('warning') } : undefined}
      title={
        selected
          ? `${carried.name} — ${selected.read ? t.grenadeSelectedRead : t.grenadeSelected}`
          : carried.name
      }
    >
      {icon ? (
        <span className="relative inline-flex">
          <WeaponIcon
            imageUrl={icon.url}
            tinted={icon.tinted}
            label={carried.name}
            width={iconPx}
            height={iconPx}
          />
          <ReplayIconCount count={carried.count} />
        </span>
      ) : (
        <>
          {carried.name}
          <span className="tabular-nums text-foreground">{carried.count}</span>
        </>
      )}
    </span>
  )
}
