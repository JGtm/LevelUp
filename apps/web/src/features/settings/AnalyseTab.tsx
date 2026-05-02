/**
 * AnalyseTab — onglet "Analyse" de SettingsPage : sessions + badges.
 *
 * P8.4 (revue 2026-04-29) : extrait de SettingsPage.tsx (~165L).
 */
import { useState } from 'react'
import { Link } from '@tanstack/react-router'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { useAppShellStore } from '@/stores/appShellStore'
import { useRecalculateSessions } from '@/features/settings/queries'
import { ToggleRow, BulletHint, type TabProps } from './_settingsShared'

export function AnalyseTab({ merged, handleChange, t }: TabProps) {
  const activeSyncJobId = useAppShellStore((s) => s.activeSyncJobId)
  const syncRunning = !!activeSyncJobId
  const recalculate = useRecalculateSessions()
  const [showRecalcConfirm, setShowRecalcConfirm] = useState(false)

  return (
    <>
      {/* Card : Regroupement de sessions */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t.sessionGroupingTitle}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4 divide-y divide-border/50">
          {/* Délai entre matchs */}
          <div className="flex flex-col gap-1 py-2">
            <div className="flex items-center justify-between">
              <span className="text-sm text-foreground">{t.sessionGapLabel}</span>
              <div className="flex items-center gap-1.5">
                <input
                  type="number"
                  min={0}
                  max={1440}
                  className="w-20 rounded border border-border bg-background px-2 py-1 text-right text-sm"
                  value={merged.session_gap_minutes ?? 120}
                  onChange={(e) =>
                    handleChange('session_gap_minutes', parseInt(e.target.value, 10) || 120)
                  }
                />
                <span className="text-sm text-muted-foreground">{t.sessionGapUnit}</span>
              </div>
            </div>
            <p className="text-xs text-muted-foreground">{t.sessionGapHint}</p>
          </div>

          {/* Composition de l'équipe */}
          <div className="flex flex-col gap-1 py-2">
            <div className="flex items-center justify-between">
              <span className="text-sm text-foreground">{t.sessionTeamChangeLabel}</span>
              <select
                value={merged.session_team_change_mode ?? 'friends'}
                onChange={(e) =>
                  handleChange(
                    'session_team_change_mode',
                    e.target.value as 'ignore' | 'group' | 'friends',
                  )
                }
                className="rounded border border-input px-2 py-1 text-sm"
              >
                <option value="ignore">{t.sessionTeamChangeIgnore}</option>
                <option value="group">{t.sessionTeamChangeGroup}</option>
                <option value="friends">{t.sessionTeamChangeFriends}</option>
              </select>
            </div>
            <BulletHint hint={t.sessionTeamChangeHint} />
          </div>

          {/* Couper si passage ranked ↔ social */}
          <div className="flex flex-col gap-1 py-2">
            <ToggleRow
              label={t.sessionSplitRankedLabel}
              value={merged.session_split_on_ranked_change ?? false}
              onChange={(v) => handleChange('session_split_on_ranked_change', v)}
            />
            <p className="text-xs text-muted-foreground">{t.sessionSplitRankedHint}</p>
          </div>

          {/* Bouton recalcul */}
          <div className="pt-2">
            {!showRecalcConfirm ? (
              <Button
                variant="outline"
                size="sm"
                disabled={syncRunning || recalculate.isPending}
                onClick={() => setShowRecalcConfirm(true)}
                title={syncRunning ? t.sessionRecalcRunning : undefined}
              >
                {recalculate.isPending ? t.sessionRecalcRunning : t.sessionRecalcButton}
              </Button>
            ) : (
              <div className="rounded-md border border-border bg-muted/40 p-3 text-sm">
                <p className="font-medium text-foreground">{t.sessionRecalcConfirmTitle}</p>
                <p className="mt-1 text-xs text-muted-foreground">{t.sessionRecalcConfirmBody}</p>
                <div className="mt-3 flex gap-2">
                  <Button
                    variant="destructive"
                    size="sm"
                    onClick={() => { recalculate.mutate(); setShowRecalcConfirm(false) }}
                  >
                    {t.sessionRecalcConfirmOk}
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => setShowRecalcConfirm(false)}
                  >
                    {t.sessionRecalcConfirmCancel}
                  </Button>
                </div>
              </div>
            )}
            {syncRunning && (
              <p className="mt-1 text-xs text-muted-foreground">{t.sessionRecalcPending}</p>
            )}
          </div>
        </CardContent>
      </Card>

      {/* Card : Badges de performance */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t.badgesTitle}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4 divide-y divide-border/50">
          {/* Sensibilité */}
          <div className="flex flex-col gap-2 py-2">
            <span className="text-sm text-foreground">{t.badgeSensitivityLabel}</span>
            <div className="flex gap-4">
              {(
                [
                  ['relaxed', t.badgeSensitivityRelaxed],
                  ['standard', t.badgeSensitivityStandard],
                  ['strict', t.badgeSensitivityStrict],
                ] as const
              ).map(([val, label]) => (
                <label key={val} className="flex cursor-pointer items-center gap-1.5 text-sm">
                  <input
                    type="radio"
                    name="badge_sensitivity"
                    value={val}
                    checked={(merged.outcome_badge_sensitivity ?? 'standard') === val}
                    onChange={() => handleChange('outcome_badge_sensitivity', val)}
                  />
                  {label}
                </label>
              ))}
            </div>
            <BulletHint hint={t.badgeSensitivityHint} />
          </div>

          {/* Exclure bots des badges */}
          <div className="flex flex-col gap-1 py-2">
            <ToggleRow
              label={t.badgeExcludeBotsFromBadgesLabel}
              value={merged.outcome_exclude_bot_matches_from_badges ?? true}
              onChange={(v) => handleChange('outcome_exclude_bot_matches_from_badges', v)}
            />
            <p className="text-xs text-muted-foreground">{t.badgeExcludeBotsFromBadgesHint}</p>
          </div>

          {/* Exclure bots des records */}
          <div className="flex flex-col gap-1 py-2">
            <ToggleRow
              label={t.badgeExcludeBotsFromRecordsLabel}
              value={merged.outcome_exclude_bot_matches_from_records ?? false}
              onChange={(v) => handleChange('outcome_exclude_bot_matches_from_records', v)}
            />
            <p className="text-xs text-muted-foreground">{t.badgeExcludeBotsFromRecordsHint}</p>
          </div>
        </CardContent>
      </Card>

      {/* Card : Progression long-terme (Objectifs & Prestige) */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t.progressionTitle}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <ToggleRow
            label={t.showProgressionLabel}
            value={merged.show_progression ?? true}
            onChange={(v) => handleChange('show_progression', v)}
          />
          <p className="text-xs text-muted-foreground">{t.progressionHint}</p>
          <Link
            to="/help"
            search={{ tab: 'glossary' }}
            className="inline-flex text-xs font-medium text-sidebar-primary hover:underline"
          >
            {t.progressionGlossaryLink} →
          </Link>
        </CardContent>
      </Card>
    </>
  )
}
