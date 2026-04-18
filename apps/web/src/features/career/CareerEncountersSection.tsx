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
          ? 'text-green-600 font-medium'
          : pct <= 40
          ? 'text-red-500 font-medium'
          : 'text-gray-700'
      }
    >
      {pct} %
    </span>
  )
}

function EncounterTable({ items }: { items: CareerEncounter[] }) {
  return (
    <div className="overflow-x-auto">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-gray-100 text-xs font-medium text-gray-500">
            <th className="pb-2 text-left">Joueur</th>
            <th className="pb-2 text-right">Matchs</th>
            <th className="pb-2 text-right">Win%</th>
            <th className="pb-2 text-right">Victoires</th>
            <th className="pb-2 text-right">Défaites</th>
            <th className="pb-2 text-right">Dernière rencontre</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-gray-50">
          {items.map((enc) => (
            <tr key={enc.encounter_key} className="hover:bg-gray-50">
              <td className="py-1.5 font-medium text-gray-900">{enc.opponent_gamertag}</td>
              <td className="py-1.5 text-right text-gray-600">{enc.count_matches}</td>
              <td className="py-1.5 text-right">
                <WinRate wins={enc.wins} total={enc.count_matches} />
              </td>
              <td className="py-1.5 text-right text-green-600">{enc.wins}</td>
              <td className="py-1.5 text-right text-red-500">{enc.losses}</td>
              <td className="py-1.5 text-right text-xs text-gray-400">
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
          <p className="py-6 text-center text-sm text-gray-400">Aucune rencontre enregistrée.</p>
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
