/**
 * FicheLancers — « GRENADES LANCÉES » (une ligne par type) et « RAMASSAGES » (armes, grenades) du joueur jusqu'à
 * l'instant lu. Le film ne montre pas tous les ramassages : leur compte est un minimum, la section le dit par la
 * capsule « estimation » à côté de son titre. Les grenades ramassées ne se relient à aucun nom : une seule ligne,
 * sans type.
 *
 * Un zéro n'a pas de ligne : il n'y a rien à lister. Le rendu ne lit rien.
 */
import { REPLAY_TEXT, type ReplayLocale } from '../../i18n/i18n'
import { formatCount } from '../../model/ficheFormat'
import type { PickupsView } from '../../model/playerInstantView'
import { EstimateCapsule, FicheRow, FicheRows, FicheSection } from './ficheParts'

export function FicheGrenades({ throws, locale }: { throws: readonly { name: string; count: number }[]; locale: ReplayLocale }) {
  const t = REPLAY_TEXT[locale].playerInstant
  return (
    <FicheSection title={t.grenadesThrown}>
      <FicheRows>
        {throws.map((g) => (
          <FicheRow key={g.name} label={g.name}>
            {formatCount(g.count, locale)}
          </FicheRow>
        ))}
      </FicheRows>
    </FicheSection>
  )
}

export function FicheRamassages({ pickups, locale }: { pickups: PickupsView; locale: ReplayLocale }) {
  const t = REPLAY_TEXT[locale].playerInstant
  return (
    <FicheSection title={t.pickups} aside={<EstimateCapsule text={t.estimate} hint={t.pickupsHint} />}>
      <FicheRows>
        {pickups.weapons !== null && <FicheRow label={t.pickupWeapons}>{formatCount(pickups.weapons, locale)}</FicheRow>}
        {pickups.grenades !== null && <FicheRow label={t.pickupGrenades}>{formatCount(pickups.grenades, locale)}</FicheRow>}
      </FicheRows>
    </FicheSection>
  )
}
