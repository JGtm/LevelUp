/**
 * TitlesTab — onglet « Jeux » des réglages (sélection par titre).
 *
 * Liste les jeux suivis par le joueur courant avec un toggle actif/pause et un
 * bouton purge. Invariant « ≥ 1 jeu actif » : le dernier titre actif a son toggle
 * + son bouton purge grisés (filet serveur 409 last_active_title en secours).
 *
 * Onglet AUTONOME (ses propres hooks, pas TabProps) — calqué sur BackupTab.
 */
import { useState } from 'react'

import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import type { ApiError } from '@/lib/api/client'
import { ToggleRow } from './_settingsShared'
import type { getSettingsText } from './i18n'
import { usePlayerTitles, useSetTitleSync, usePurgeTitleData } from './queries'

type SettingsText = ReturnType<typeof getSettingsText>

function errorMessage(err: unknown, t: SettingsText): string {
  if (err && typeof err === 'object' && 'code' in err) {
    if ((err as ApiError).code === 'last_active_title') return t.titleLastActiveHint
  }
  return t.titleActionError
}

function StatusBadge({ active, t }: { active: boolean; t: SettingsText }) {
  return (
    <span className={`text-xs ${active ? 'text-success' : 'text-muted-foreground'}`}>
      {active ? t.titleStatusActive : t.titleStatusPaused}
    </span>
  )
}

export function TitlesTab({ t, frozen }: { t: SettingsText; frozen?: boolean }) {
  const { data: titles = [], isLoading } = usePlayerTitles()
  const setSync = useSetTitleSync()
  const purge = usePurgeTitleData()
  const [error, setError] = useState<string | null>(null)

  const enrolled = titles.filter((x) => x.enrolled)
  const activeCount = enrolled.filter((x) => x.syncEnabled).length
  const busy = setSync.isPending || purge.isPending

  function handleToggle(slug: string, next: boolean) {
    setError(null)
    setSync.mutate({ slug, enabled: next }, { onError: (e) => setError(errorMessage(e, t)) })
  }

  function handlePurge(slug: string) {
    if (!window.confirm(t.titlePurgeConfirm)) return
    setError(null)
    purge.mutate(
      { slug },
      {
        onSuccess: (r) => {
          if (!r.data_removed) setError(t.titlePurgeResidualWarning)
        },
        onError: (e) => setError(errorMessage(e, t)),
      },
    )
  }

  if (isLoading) return null

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">{t.titlesSectionTitle}</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-1">
        <p className="mb-2 text-xs text-muted-foreground">{t.titlesSectionDesc}</p>
        {enrolled.map((row) => {
          // Le dernier titre actif ne peut être ni mis en pause ni purgé (min 1).
          const isLastActive = row.syncEnabled && activeCount <= 1
          const locked = frozen || busy || isLastActive
          return (
            <div key={row.slug} className="flex items-center justify-between gap-3 border-b border-border/40 py-1 last:border-0">
              <div className="flex-1">
                <ToggleRow
                  label={row.name}
                  value={row.syncEnabled}
                  onChange={(v) => handleToggle(row.slug, v)}
                  disabled={locked}
                  accessory={<StatusBadge active={row.syncEnabled} t={t} />}
                />
              </div>
              <Button variant="ghost" size="sm" disabled={locked} onClick={() => handlePurge(row.slug)}>
                {t.titlePurgeButton}
              </Button>
            </div>
          )
        })}
        {activeCount <= 1 && enrolled.length > 0 && (
          <p className="mt-1 text-xs text-muted-foreground">{t.titleLastActiveHint}</p>
        )}
        {error && (
          <p className="mt-2 text-sm text-destructive" role="alert">
            {error}
          </p>
        )}
      </CardContent>
    </Card>
  )
}
