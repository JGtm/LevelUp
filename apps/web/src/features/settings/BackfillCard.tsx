/**
 * BackfillCard — UI de recalcul rétroactif (Settings → Sync tab).
 *
 * P8.4 (revue 2026-04-29) : extrait de SettingsPage.tsx (~195L).
 */
import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { apiErrorMessage } from '@/lib/api/client'
import { useAppShellStore } from '@/stores/appShellStore'
import { useStartBackfill } from '@/features/settings/queries'
import { useJobStatus } from '@/features/setup/queries'
import { useJobToasts } from '@/features/settings/useJobToasts'
import type { getSettingsText } from '@/features/settings/i18n'
import { ToggleRow } from './_settingsShared'

interface BackfillCardProps {
  t: ReturnType<typeof getSettingsText>
}

export function BackfillCard({ t }: BackfillCardProps) {
  const availablePlayers = useAppShellStore((s) => s.availablePlayers)
  const realPlayers = availablePlayers.filter((p) => !p.is_demo)
  const firstSlug = realPlayers[0]?.player_slug ?? ''

  const [scope, setScope] = useState({
    medals: false,
    skill: false,
    aliases: false,
    personal_scores: false,
    performance_scores: false,
    lusr: false,
    csr: false,
    events: false,
    weapons: false,
    engagement_scores: false,
    engagement_coefficients: false,
  })
  const [selectedSlug, setSelectedSlug] = useState<string>(firstSlug)
  const [forceRescan, setForceRescan] = useState(false)
  const [showForceConfirm, setShowForceConfirm] = useState(false)
  const [activeJobId, setActiveJobId] = useState<string | null>(null)

  // Si la liste de joueurs arrive après le premier render, initialiser selectedSlug
  // (ajustement pendant le rendu au lieu d'un effet ; auto-terminant).
  if (!selectedSlug && firstSlug) setSelectedSlug(firstSlug)

  const startBackfill = useStartBackfill()
  const { data: jobStatus } = useJobStatus(activeJobId ?? '', !!activeJobId)

  const running =
    !!activeJobId &&
    jobStatus?.status !== 'succeeded' &&
    jobStatus?.status !== 'failed' &&
    jobStatus?.status !== 'cancelled' &&
    jobStatus?.status !== 'interrupted'

  // Reset activeJobId quand le job atteint un état terminal (réaction à l'état
  // async du polling → arrête la requête useJobStatus ; effet légitime).
  useEffect(() => {
    if (
      activeJobId &&
      (jobStatus?.status === 'succeeded' ||
        jobStatus?.status === 'failed' ||
        jobStatus?.status === 'cancelled' ||
        jobStatus?.status === 'interrupted')
    ) {
      // eslint-disable-next-line react-hooks/set-state-in-effect -- réaction à l'arrivée async d'un état terminal (arrête le polling), pas un dérivé synchrone (2026-07-22)
      setActiveJobId(null)
    }
  }, [jobStatus?.status, activeJobId])

  useJobToasts(jobStatus, {
    succeeded: t.backfillToastSucceeded,
    succeededWithWarnings: t.backfillToastSucceededWithWarnings,
    failed: t.backfillToastFailed,
    cancelled: t.backfillToastCancelled,
  })

  const anyChecked = Object.values(scope).some(Boolean)
  const canRun = anyChecked && !!selectedSlug && !running && !startBackfill.isPending

  const selectedGamertag =
    realPlayers.find((p) => p.player_slug === selectedSlug)?.gamertag ?? selectedSlug

  function runBackfill(force: boolean) {
    startBackfill.mutate(
      { player_slug: selectedSlug, ...scope, force_rescan: force },
      {
        onSuccess: (job) => {
          setActiveJobId(job.job_id)
          toast.info(t.backfillToastStarted, {
            description: `${selectedGamertag}${force ? ' — forcé' : ''}`,
          })
        },
        onError: (err) => {
          toast.error(t.backfillToastStartFailed, {
            description: apiErrorMessage(err),
          })
        },
      },
    )
  }

  function handleRunClick() {
    if (!canRun) return
    if (forceRescan) {
      setShowForceConfirm(true)
    } else {
      runBackfill(false)
    }
  }

  function confirmForce() {
    setShowForceConfirm(false)
    runBackfill(true)
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">{t.backfillTitle}</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        {/* Sélection des types de données */}
        <div className="grid grid-cols-2 gap-x-6 gap-y-1 py-1 sm:grid-cols-3">
          {(
            [
              ['medals', t.backfillMedals],
              ['skill', t.backfillSkill],
              ['aliases', t.backfillAliases],
              ['personal_scores', t.backfillPersonalScores],
              ['performance_scores', t.backfillPerfScores],
              ['lusr', t.backfillLUSR],
              ['csr', t.backfillCSR],
              ['events', t.backfillEvents],
              ['weapons', t.backfillWeapons],
              ['engagement_scores', t.backfillEngagementScores],
              ['engagement_coefficients', t.backfillEngagementCoefficients],
            ] as const
          ).map(([field, label]) => (
            <ToggleRow
              key={field}
              label={label}
              value={scope[field]}
              onChange={(v) => setScope((s) => ({ ...s, [field]: v }))}
              disabled={running}
            />
          ))}
        </div>

        {/* Joueur */}
        <div className="flex items-center justify-between gap-3 border-t border-border/50 pt-3 text-sm">
          <span>{t.backfillPlayerLabel}</span>
          <select
            value={selectedSlug}
            onChange={(e) => setSelectedSlug(e.target.value)}
            disabled={running || realPlayers.length === 0}
            className="rounded-md border border-input bg-background px-2 py-1 text-sm text-foreground disabled:cursor-not-allowed disabled:opacity-50"
          >
            {realPlayers.map((p) => (
              <option key={p.player_slug} value={p.player_slug}>{p.gamertag}</option>
            ))}
          </select>
        </div>

        {/* Forcer */}
        <ToggleRow
          label={t.backfillForceLabel}
          value={forceRescan}
          onChange={setForceRescan}
          disabled={running}
        />

        {/* Bouton + hint */}
        <div className="flex flex-col gap-2">
          <Button
            onClick={handleRunClick}
            disabled={!canRun}
            className="self-start"
          >
            {running ? t.backfillRunningLabel : t.backfillRunButton}
          </Button>
          {!anyChecked && (
            <p className="text-xs text-muted-foreground">{t.backfillNoScopeHint}</p>
          )}
        </div>

        {/* Confirmation forcer */}
        {showForceConfirm && (
          <div className="rounded-md border border-border bg-muted/40 p-3 text-sm">
            <p className="font-medium text-foreground">{t.backfillForceConfirmTitle}</p>
            <p className="mt-1 text-xs text-muted-foreground">{t.backfillForceConfirmBody}</p>
            <div className="mt-3 flex gap-2">
              <Button variant="destructive" size="sm" onClick={confirmForce}>
                {t.backfillForceConfirmOk}
              </Button>
              <Button variant="outline" size="sm" onClick={() => setShowForceConfirm(false)}>
                {t.backfillForceConfirmCancel}
              </Button>
            </div>
          </div>
        )}

        {/* Warnings retournés par le backend après exécution */}
        {jobStatus?.status === 'succeeded' && jobStatus.warnings && jobStatus.warnings.length > 0 && (
          <div className="rounded-md border border-warning/40 bg-warning/5 p-3 text-sm">
            <p className="font-medium text-warning">{t.backfillWarningsHeader}</p>
            <ul className="mt-1 list-disc pl-5 text-xs text-muted-foreground">
              {jobStatus.warnings.map((w, i) => (
                <li key={i}>{w}</li>
              ))}
            </ul>
          </div>
        )}
      </CardContent>
    </Card>
  )
}
