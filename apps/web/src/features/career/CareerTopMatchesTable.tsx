/**
 * CareerTopMatchesTable — tableau des meilleurs/pires matchs.
 */
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import type { CareerTopMatch } from '@/lib/api/types'

interface Props {
  items: CareerTopMatch[]
}

export function CareerTopMatchesTable({ items }: Props) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">Meilleurs matchs récents</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-100 text-xs font-medium text-gray-500">
                <th className="pb-2 text-left">Date</th>
                <th className="pb-2 text-left">Carte / Mode</th>
                <th className="pb-2 text-right">Score</th>
                <th className="pb-2 text-right">Résultat</th>
                <th className="pb-2 text-right">Perf.</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-50">
              {items.map((m) => (
                <tr key={m.match_id} className="hover:bg-gray-50">
                  <td className="py-1.5 text-gray-400">
                    {m.start_time ? new Date(m.start_time).toLocaleDateString('fr-FR') : '—'}
                  </td>
                  <td className="py-1.5">
                    <span className="font-medium text-gray-800">{m.map_ui ?? '—'}</span>
                    {m.mode_ui && (
                      <span className="ml-1 text-xs text-gray-400">· {m.mode_ui}</span>
                    )}
                  </td>
                  <td className="py-1.5 text-right text-gray-700">{m.score_label}</td>
                  <td className="py-1.5 text-right">
                    <Badge
                      variant={
                        m.outcome_label?.toLowerCase().includes('victoire')
                          ? 'success'
                          : m.outcome_label?.toLowerCase().includes('défaite')
                          ? 'destructive'
                          : 'secondary'
                      }
                    >
                      {m.outcome_label}
                    </Badge>
                  </td>
                  <td className="py-1.5 text-right font-mono text-purple-700">
                    {m.performance_score != null ? m.performance_score.toFixed(0) : '—'}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </CardContent>
    </Card>
  )
}
