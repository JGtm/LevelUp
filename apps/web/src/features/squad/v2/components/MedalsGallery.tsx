/**
 * MedalsGallery — galerie de medailles par match × joueur (chunk S9).
 *
 * 1 carte par match, contenu = grille de medailles par joueur (gamertag).
 * Les medailles sont deja triees count desc cote backend.
 */
import type { MedalsGalleryEntry } from '../types'

export interface MedalsGalleryProps {
  entries: MedalsGalleryEntry[]
  /** Ordre stable des joueurs (main + coequipiers). */
  squadOrder: string[]
  /** Labels deja localises par le caller. */
  labels: {
    emptyMatch: string
  }
}

export function MedalsGallery({ entries, squadOrder, labels }: MedalsGalleryProps) {
  if (entries.length === 0) {
    return null
  }
  return (
    <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3" data-testid="medals-gallery">
      {entries.map((entry) => (
        <div
          key={entry.match_id}
          className="rounded-lg border border-border bg-card p-3"
          data-testid="medals-gallery-card"
        >
          <div className="mb-2 text-xs text-muted-foreground">{entry.match_id}</div>
          <div className="flex flex-col gap-2">
            {squadOrder.map((gt) => {
              const medals = entry.medals_by_xuid[gt]
              if (!medals || medals.length === 0) {
                return null
              }
              return (
                <div key={gt} className="flex flex-wrap items-center gap-2">
                  <span className="text-sm font-medium">{gt}</span>
                  <div className="flex flex-wrap gap-1">
                    {medals.map((m) => (
                      <span
                        key={m.medal_id}
                        className="inline-flex items-center gap-1 rounded bg-muted px-2 py-0.5 text-xs"
                        title={m.label ?? `#${m.medal_id}`}
                      >
                        {m.label ?? `#${m.medal_id}`}
                        {m.count > 1 && (
                          <span className="text-muted-foreground">×{m.count}</span>
                        )}
                      </span>
                    ))}
                  </div>
                </div>
              )
            })}
            {Object.keys(entry.medals_by_xuid).length === 0 && (
              <div className="text-sm text-muted-foreground">{labels.emptyMatch}</div>
            )}
          </div>
        </div>
      ))}
    </div>
  )
}
