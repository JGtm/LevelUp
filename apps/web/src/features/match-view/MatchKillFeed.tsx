/**
 * MatchKillFeed — kill-feed chronologique d'un match (timeline canonique
 * d'events, chargée on-demand via /matches/{id}/events?types=kill).
 *
 * Source serveur : reconstruit depuis les faits marquants pour Halo Infinite
 * (dégradé — sans arme ni positions) ou natif pour Halo 5. La section est
 * AUTONOME (rend son propre titre) et se masque entièrement si la timeline est
 * indisponible (503), en erreur, ou sans kill — pas de section vide titrée.
 *
 * Affichage des noms : EXCLUSIVEMENT via displayPlayerName (chokepoint front,
 * jamais gamertag||xuid). Le backend a déjà résolu les gamertags (v_gamertag_lookup) ;
 * un xuid orphelin reste masqué proprement.
 */
import type { ReactNode } from 'react'
import { useMatchEvents } from './queries'
import { apiErrorCode } from '@/lib/api/client'
import { displayPlayerName } from '@/lib/players/displayName'
import type { MatchEvent } from '@/lib/api/types'
import type { MatchViewText } from './i18n'

function formatEventTime(ms: number): string {
  const totalSec = Math.max(0, Math.floor(ms / 1000))
  const m = Math.floor(totalSec / 60)
  const s = totalSec % 60
  return `${m}:${s.toString().padStart(2, '0')}`
}

interface MatchKillFeedProps {
  playerSlug: string
  matchId: string
  /** xuid du joueur courant — met en avant les kills/morts le concernant. */
  meXUID: string | null
  t: MatchViewText
}

// KillFeedShell — coquille de section commune (titre type-1) pour les 3 états
// (indisponible / aucun event / liste), évite la duplication du markup.
function KillFeedShell({ t, children }: { t: MatchViewText; children: ReactNode }) {
  return (
    <section className="space-y-4">
      <h3 className="text-base font-semibold text-foreground">{t.sectionKillFeed}</h3>
      {children}
    </section>
  )
}

export function MatchKillFeed({ playerSlug, matchId, meXUID, t }: MatchKillFeedProps) {
  // Kill-feed = uniquement les kills (les médailles ont leur propre section).
  const { data, isPending, isError, error } = useMatchEvents(playerSlug, matchId, ['kill'])

  // Chargement → rien (section secondaire, pas de squelette).
  if (isPending) return null

  // Erreur : on distingue le titre qui N'EXPOSE PAS la timeline (503
  // capability_not_supported → message explicite « indisponible pour ce titre »,
  // cf. PLAN_CANONICAL_MATCH_EVENTS §4 Phase 3) d'une erreur réseau transitoire
  // (masquage silencieux pour ne pas polluer la page).
  if (isError) {
    if (apiErrorCode(error) === 'capability_not_supported') {
      return (
        <KillFeedShell t={t}>
          <p className="text-sm text-muted-foreground">{t.killFeedUnsupported}</p>
        </KillFeedShell>
      )
    }
    return null
  }

  const events = data?.events ?? []
  // Titre supporté mais ce match n'a aucun kill exploitable → état vide explicite
  // (distinct de « indisponible pour ce titre »).
  if (events.length === 0) {
    return (
      <KillFeedShell t={t}>
        <p className="text-sm text-muted-foreground">{t.killFeedNoData}</p>
      </KillFeedShell>
    )
  }

  const degraded = (data?.limitations ?? []).length > 0

  return (
    <KillFeedShell t={t}>
      <div className="space-y-2">
        <ol className="divide-y divide-border overflow-hidden rounded-md border border-border bg-card">
          {events.map((ev, i) => (
            <KillFeedRow key={`${ev.time_ms}-${i}`} ev={ev} meXUID={meXUID} t={t} />
          ))}
        </ol>
        {degraded && (
          <p className="text-xs text-muted-foreground">{t.killFeedDegradedNote}</p>
        )}
      </div>
    </KillFeedShell>
  )
}

function KillFeedRow({
  ev,
  meXUID,
  t,
}: {
  ev: MatchEvent
  meXUID: string | null
  t: MatchViewText
}) {
  // PlayerIdentity sérialise en PascalCase (canonical.PlayerIdentity sans json tags).
  const killerName = ev.killer
    ? displayPlayerName(ev.killer.Gamertag, ev.killer.XUID)
    : t.killFeedEnvironment
  const victimName = ev.victim ? displayPlayerName(ev.victim.Gamertag, ev.victim.XUID) : '—'
  const involvesMe = !!meXUID && (ev.killer?.XUID === meXUID || ev.victim?.XUID === meXUID)
  const weaponLabel = ev.weapon?.DefaultLabel?.trim()

  return (
    <li
      className={`flex items-center gap-3 px-3 py-2 text-sm ${
        involvesMe ? 'font-medium text-foreground' : 'text-muted-foreground'
      }`}
    >
      <span className="shrink-0 tabular-nums text-xs text-muted-foreground">
        {formatEventTime(ev.time_ms)}
      </span>
      <span className="min-w-0 flex-1 truncate">
        <span className="text-foreground">{killerName}</span>
        <span className="mx-1.5 text-muted-foreground" aria-hidden="true">
          →
        </span>
        <span>{victimName}</span>
      </span>
      {ev.headshot && (
        <span className="shrink-0 text-xs uppercase tracking-wide text-muted-foreground">HS</span>
      )}
      {weaponLabel && (
        <span className="max-w-[10rem] shrink-0 truncate text-xs text-muted-foreground">
          {weaponLabel}
        </span>
      )}
    </li>
  )
}
