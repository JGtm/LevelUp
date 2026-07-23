/**
 * AppearanceDiagSection — « Diagnostic apparence Spartan » (onglet Données, Lot G).
 *
 * Le titre de section est porté HORS carte par la page parente (AdminDataPage,
 * règle du Lot A) : ici, un rappel + le sélecteur de joueur suivi + le bouton
 * « Diagnostiquer », puis le résultat par composant. Le diagnostic part
 * UNIQUEMENT au clic (mutation à la demande, aucun refetch auto/au focus).
 */
import { useState } from 'react'

import { Button } from '@/components/ui/button'
import { Spinner } from '@/components/ui/spinner'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { useAppShellStore } from '@/stores/appShellStore'
import type { AppearanceDiagnosisResponse } from '@/lib/api/types'
import { useAdminT, useDateLocale } from '../useAdminText'
import { useAppearanceDiag } from './mutations'
import { AppearanceComponentCard } from './AppearanceComponentCard'
import { fetchStatusKey } from './appearanceDiagDisplay'

export function AppearanceDiagSection() {
  const tA = useAdminT()
  const dateLocale = useDateLocale()
  const availablePlayers = useAppShellStore((s) => s.availablePlayers)
  const [selectedSlug, setSelectedSlug] = useState('')
  const diag = useAppearanceDiag()

  function onSelect(slug: string) {
    setSelectedSlug(slug)
    diag.reset() // changer de joueur efface le résultat précédent.
  }

  function onDiagnose() {
    if (selectedSlug) diag.mutate(selectedSlug)
  }

  return (
    <div className="space-y-4">
      <p className="max-w-3xl text-sm text-muted-foreground">{tA('admin.appearance.hint')}</p>

      <div className="flex flex-wrap items-end gap-3">
        <label className="flex flex-col gap-1 text-xs font-medium uppercase tracking-wide text-muted-foreground">
          {tA('admin.appearance.select_label')}
          <select
            value={selectedSlug}
            onChange={(e) => onSelect(e.target.value)}
            className="min-w-[14rem] rounded-md border border-input bg-background px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-ring"
          >
            <option value="" disabled>
              {tA('admin.appearance.select_placeholder')}
            </option>
            {availablePlayers.map((p) => (
              <option key={p.player_slug} value={p.player_slug}>
                {p.gamertag}
              </option>
            ))}
          </select>
        </label>
        <Button onClick={onDiagnose} disabled={!selectedSlug || diag.isPending}>
          {diag.isPending ? tA('admin.appearance.diagnosing') : tA('admin.appearance.diagnose')}
        </Button>
      </div>

      {diag.isPending ? (
        <div className="py-6">
          <Spinner label={tA('admin.appearance.diagnosing')} />
        </div>
      ) : diag.isError ? (
        <EmptyStateNotice
          title={tA('admin.appearance.error_title')}
          description={tA('admin.appearance.error_desc')}
        />
      ) : diag.data ? (
        <DiagnosisResult data={diag.data} dateLocale={dateLocale} />
      ) : (
        <EmptyStateNotice
          title={tA('admin.appearance.initial_title')}
          description={tA('admin.appearance.initial_desc')}
        />
      )}
    </div>
  )
}

function DiagnosisResult({
  data,
  dateLocale,
}: {
  data: AppearanceDiagnosisResponse
  dateLocale: string
}) {
  const tA = useAdminT()
  const components = data.components ?? []
  const generatedAt = data.generated_at ? new Date(data.generated_at) : null

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-muted-foreground">
        <span className="font-medium text-foreground">{data.gamertag}</span>
        {generatedAt && !Number.isNaN(generatedAt.getTime()) && (
          <span>
            {tA('admin.appearance.generated_at')} {generatedAt.toLocaleString(dateLocale)}
          </span>
        )}
        <span>
          {tA('admin.appearance.last_fetch_label')} : {tA(fetchStatusKey(data.last_fetch_status))}
        </span>
      </div>

      <div className="grid gap-4 sm:grid-cols-2">
        {components.map((c) => (
          <AppearanceComponentCard key={c.component} diag={c} />
        ))}
      </div>
    </div>
  )
}
