/**
 * MatchHeader.waypointLink.tsx — le lien externe « Voir sur Halo Waypoint » du header
 * de match view.
 *
 * Aucune reconstruction d'URL ici : `buildWaypointMatchUrl` est le point unique
 * (garde-rail `waypointUrl.guard.test.ts`), et le gating passe par la capability
 * `waypoint_match_url` comme dans ExplorerMatchesTable / SquadSynergyHistoryTable.
 */
import { buildWaypointMatchUrl, waypointLogoSrc } from '@/lib/match-nav/waypointUrl'
import { useCapability } from '@/lib/capabilities/capabilities'
import { useTitleSlug } from '@/lib/title-routing'
import { useSettingsDraftStore } from '@/stores/settingsDraftStore'

import { matchViewManifest } from '@/lib/i18n/generated/match_view'
import type { MatchViewLocale } from './i18n'

interface WaypointLinkProps {
  matchId: string
  playerSlug: string
  locale: MatchViewLocale
}

/** Rien quand le titre courant ne déclare pas de page de match sur Waypoint. */
export function WaypointLink({ matchId, playerSlug, locale }: WaypointLinkProps) {
  const titleSlug = useTitleSlug()
  const available = useCapability('waypoint_match_url')
  // Logo raster à deux variantes : c'est le thème LOCAL déjà tranché par le store qui
  // choisit le fichier, un PNG ne se teinte pas en `currentColor`.
  const theme = useSettingsDraftStore((state) => state.localUiPrefs.theme)
  if (!available) return null
  const label = matchViewManifest['match_view.header.waypoint_link'][locale]
  return (
    <a
      href={buildWaypointMatchUrl(playerSlug, matchId, titleSlug)}
      target="_blank"
      rel="noopener noreferrer"
      title={label}
      aria-label={label}
      className="inline-flex h-9 w-9 items-center justify-center rounded-md border border-border bg-transparent transition-colors hover:bg-accent hover:text-accent-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
    >
      {/* Logotype 360x160 (ratio 2.25) : `object-contain` + taille intrinsèque, sinon
          le `fill` par défaut l'écrase dans un carré (déformation constatée I19). */}
      <img
        src={waypointLogoSrc(theme)}
        alt=""
        aria-hidden
        width={24}
        height={11}
        className="h-auto w-6 object-contain"
      />
    </a>
  )
}
