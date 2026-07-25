import { EmptyStateCard } from '@/components/ui/empty-state'
import { useAppShellStore } from '@/stores/appShellStore'

import type { TitleCapability } from './capabilities'

/**
 * Libellés FR/EN par capability pour le placeholder « feature indisponible pour
 * ce titre ». Inline (pas de manifest TOML) car ce composant vit dans `lib/` et
 * n'a qu'un seul message court par locale. NO-OP pour `halo_infinite` (déclare
 * toutes les capabilities → ce placeholder n'est jamais rendu en mono-titre).
 */
const FEATURE_LABEL: Record<TitleCapability, { fr: string; en: string }> = {
  matchmaking: { fr: 'les matchs', en: 'matches' },
  firefight: { fr: 'le Firefight (PvE)', en: 'Firefight (PvE)' },
  forge: { fr: 'Forge', en: 'Forge' },
  media: { fr: 'les médias', en: 'media' },
  ranked: { fr: 'le classé (CSR)', en: 'ranked (CSR)' },
  career: { fr: 'la carrière', en: 'career' },
  season_pass: { fr: 'le pass saisonnier', en: 'the season pass' },
  'asset.images': { fr: 'les visuels d assets', en: 'asset images' },
  achievements: { fr: 'les succès', en: 'achievements' },
  engagement: { fr: 'l engagement intra-match', en: 'intra-match engagement' },
  lusr: { fr: 'le rating LUSR', en: 'the LUSR rating' },
  'world.leaderboard': { fr: 'les classements mondiaux', en: 'world leaderboards' },
  native_kill_mechanics: { fr: 'les assassinats et compétences Spartan', en: 'assassinations and Spartan abilities' },
  team_mmr: { fr: 'le MMR par match', en: 'per-match MMR' },
  damage_taken: { fr: 'les dégâts subis', en: 'damage taken' },
  weapon_accuracy: { fr: 'la précision par arme', en: 'accuracy by weapon' },
  spartan_customizer: { fr: 'la personnalisation Spartan', en: 'Spartan customization' },
  expected_stats: { fr: 'l écart au FDA attendu', en: 'expected KDA stats' },
  waypoint_match_url: { fr: 'les liens Halo Waypoint', en: 'Halo Waypoint links' },
  objective_stats: { fr: 'les stats objectifs (CTF/Zones/Oddball)', en: 'objective stats (CTF/Zones/Oddball)' },
}

interface FeatureUnavailableProps {
  capability: TitleCapability
  className?: string
}

/**
 * FeatureUnavailable — placeholder gracieux rendu à la place d'une page entière
 * quand le titre courant ne déclare pas `capability` (cf. {@link RouteCapabilityGate}).
 * Évite une page morte / vide : on annonce explicitement que la fonctionnalité
 * n'existe pas pour ce titre, au lieu d'afficher un squelette sans données.
 */
export function FeatureUnavailable({ capability, className }: FeatureUnavailableProps) {
  const locale = useAppShellStore((s) => s.locale)
  const label = FEATURE_LABEL[capability]
  const feature = locale === 'en' ? label.en : label.fr
  const title = locale === 'en' ? 'Not available for this title' : 'Indisponible pour ce titre'
  const description =
    locale === 'en'
      ? `This title does not provide ${feature}.`
      : `Ce titre ne fournit pas ${feature}.`

  return (
    <div className={className ?? 'p-6'}>
      <EmptyStateCard title={title} description={description} />
    </div>
  )
}
