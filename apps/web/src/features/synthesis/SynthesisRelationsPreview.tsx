import { Link } from '@tanstack/react-router'

import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import type { SynthesisEncounterPreview, SynthesisRivalriesPreview } from '@/lib/api/types'

function PreviewList({
  title,
  items,
  variant,
}: {
  title: string
  items: SynthesisEncounterPreview[]
  variant: 'secondary' | 'destructive'
}) {
  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between gap-3">
        <h3 className="text-sm font-semibold text-foreground">{title}</h3>
        <Badge variant="outline">{items.length}</Badge>
      </div>
      {items.length === 0 ? (
        <EmptyStateNotice
          title="Aucune entrée"
          description="Le scope courant n'a pas encore révélé de profil récurrent sur ce versant."
        />
      ) : (
        items.map((entry) => (
          <div key={`${title}-${entry.xuid}`} className="rounded-2xl border border-border/70 bg-background/70 p-4">
            <div className="flex items-start justify-between gap-3">
              <div>
                <p className="text-sm font-semibold text-foreground">{entry.gamertag}</p>
                <p className="mt-1 text-sm text-muted-foreground">
                  {entry.match_count} matchs · avec {entry.as_teammate} · contre {entry.as_enemy}
                </p>
              </div>
              <Badge variant={variant}>{variant === 'secondary' ? `Avec ${entry.as_teammate}` : `Contre ${entry.as_enemy}`}</Badge>
            </div>
          </div>
        ))
      )}
    </div>
  )
}

export function SynthesisRelationsPreview({
  playerSlug,
  preview,
}: {
  playerSlug: string
  preview?: SynthesisRivalriesPreview | null
}) {
  return (
    <Card data-testid="synthesis-relations-preview">
      <CardHeader className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <CardTitle>Relations de jeu</CardTitle>
          <p className="mt-1 text-sm text-muted-foreground">
            Les visages qui reviennent le plus souvent dans le scope courant, à tes côtés comme en face.
          </p>
        </div>
        <Link
          to="/players/$playerSlug/palmares/relations"
          params={{ playerSlug }}
          className="inline-flex rounded-xl border border-border px-3 py-2 text-sm font-medium text-foreground transition-colors hover:bg-muted"
        >
          Ouvrir la vue complète
        </Link>
      </CardHeader>
      <CardContent>
        {!preview || preview.total === 0 ? (
          <EmptyStateNotice
            title="Aucune relation détectée"
            description="Le scope courant ne contient pas encore assez de rencontres répétées pour afficher ce bloc."
          />
        ) : (
          <div className="grid gap-4 xl:grid-cols-2">
            <PreviewList title="Alliés fréquents" items={preview.top_teammates} variant="secondary" />
            <PreviewList title="Antagonistes" items={preview.top_enemies} variant="destructive" />
          </div>
        )}
      </CardContent>
    </Card>
  )
}
