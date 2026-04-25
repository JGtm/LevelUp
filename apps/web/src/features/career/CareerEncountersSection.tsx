/**
 * CareerEncountersSection — adversaires et coéquipiers fréquents.
 */
import { useState } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Spinner } from '@/components/ui/spinner'
import type { CareerEncounter } from '@/lib/api/types'
import { useCareerEncounters } from './queries'

interface Props {
  playerSlug: string
  preview: CareerEncounter[] | null | undefined
}

function WinRate({ wins, total }: { wins: number; total: number }) {
  const pct = total > 0 ? Math.round((wins / total) * 100) : 0
  return (
    <span
      className={
        pct >= 55
          ? 'text-success font-medium'
          : pct <= 40
          ? 'text-destructive font-medium'
          : 'text-foreground'
      }
    >
      {pct} %
    </span>
  )
}

function EncounterTable({ items }: { items: CareerEncounter[] }) {
  // Phase D plan multi-titres : libellés métier issus du backend TOML avec
  // fallback gracieux sur les valeurs FR locales si MULTI_TITLE_API_ENABLED off.
  const { data: fieldMappings } = useFieldMappings()
  const labelOf = (key: string, fallback: string): string =>
    fieldMappings?.fields[key]?.label ?? fallback
  return (
    <div className="overflow-x-auto">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-border text-xs font-medium text-muted-foreground">
            <th className="pb-2 text-left">Joueur</th>
            <th className="pb-2 text-right">{labelOf('total_matches_played', 'Matchs')}</th>
            <th className="pb-2 text-right">{labelOf('win_rate', 'Win%')}</th>
            <th className="pb-2 text-right">Victoires</th>
            <th className="pb-2 text-right">Défaites</th>
            <th className="pb-2 text-right">Dernière rencontre</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-border">
          {items.map((enc) => (
            <tr key={enc.encounter_key} className="hover:bg-muted">
              <td className="py-1.5 font-medium text-foreground">{enc.opponent_gamertag}</td>
              <td className="py-1.5 text-right text-muted-foreground">{enc.count_matches}</td>
              <td className="py-1.5 text-right">
                <WinRate wins={enc.wins} total={enc.count_matches} />
              </td>
              <td className="py-1.5 text-right text-success">{enc.wins}</td>
              <td className="py-1.5 text-right text-destructive">{enc.losses}</td>
              <td className="py-1.5 text-right text-xs text-muted-foreground">
                {enc.last_seen_at
                  ? new Date(enc.last_seen_at).toLocaleDateString('fr-FR')
                  : '—'}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

export function CareerEncountersSection({ playerSlug, preview }: Props) {
  const [showAll, setShowAll] = useState(false)
  const { data, isLoading } = useCareerEncounters(playerSlug, showAll)

  const items = showAll && data ? data.items : (preview ?? [])

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <CardTitle className="text-base">Rencontres fréquentes</CardTitle>
          <Badge variant="secondary">{items.length} joueur{items.length !== 1 ? 's' : ''}</Badge>
        </div>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <div className="flex justify-center py-6">
            <Spinner size="sm" label="Chargement…" />
          </div>
        ) : items.length === 0 ? (
          <p className="py-6 text-center text-sm text-muted-foreground">Aucune rencontre enregistrée.</p>
        ) : (
          <>
            <EncounterTable items={items} />
            {!showAll && (preview?.length ?? 0) > 0 && (
              <div className="mt-4 flex justify-center">
                <Button variant="outline" size="sm" onClick={() => setShowAll(true)}>
                  Voir toutes les rencontres
                </Button>
              </div>
            )}
          </>
        )}
      </CardContent>
    </Card>
  )
}
