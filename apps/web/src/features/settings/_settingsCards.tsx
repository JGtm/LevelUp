/**
 * Cartes de réglages réutilisables, extraites de l'ancien GeneralTab lors du
 * regroupement des onglets Settings (2026-06).
 *
 * - InterfaceCard → onglet « Apparence & Accessibilité »
 * - DiscordCard   → onglet « Notifications » (Discord = canal externe)
 * - MediaCard     → onglet « Données & Médias »
 *
 * Chaque carte partage les props TabProps (merged/handleChange/t/frozen) et le
 * plumbing settings de SettingsPage. L'exemption démo de la langue est conservée :
 * le Select langue n'est jamais figé, tout le reste l'est en mode démo.
 */
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Select } from '@/components/ui/select'
import { useScanMedia } from '@/features/settings/queries'
import { ToggleRow, type TabProps } from './_settingsShared'

export function InterfaceCard({ merged, handleChange, t, frozen }: TabProps) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">{t.interfaceTitle}</CardTitle>
      </CardHeader>
      <CardContent className="space-y-1 divide-y divide-border/50">
        <div className="flex items-center justify-between py-2">
          <span className="text-sm text-foreground">{t.langLabel}</span>
          <Select
            value={merged.lang ?? 'fr'}
            onChange={(e) => handleChange('lang', e.target.value)}
            className="w-auto"
          >
            <option value="fr">{t.langFr}</option>
            <option value="en">{t.langEn}</option>
          </Select>
        </div>
        <div className="flex items-center justify-between py-2">
          <span className="text-sm text-foreground">{t.timezoneLabel}</span>
          <Select
            value={merged.user_timezone ?? 'Europe/Paris'}
            onChange={(e) => handleChange('user_timezone', e.target.value)}
            className="w-auto"
            disabled={frozen}
          >
            {/* Identifiants timezone IANA — pas de traduction (techniques) */}
            {/* eslint-disable @levelup/no-hardcoded-strings */}
            <option value="Europe/Paris">Europe/Paris</option>
            <option value="Europe/London">Europe/London</option>
            <option value="America/New_York">America/New_York</option>
            <option value="America/Los_Angeles">America/Los_Angeles</option>
            <option value="America/Chicago">America/Chicago</option>
            <option value="Asia/Tokyo">Asia/Tokyo</option>
            <option value="UTC">UTC</option>
            {/* eslint-enable @levelup/no-hardcoded-strings */}
          </Select>
        </div>
        <ToggleRow label={t.showRecords} value={merged.show_records ?? false} onChange={(v) => handleChange('show_records', v)} disabled={frozen} />
        <ToggleRow label={t.normalizeModeLabels} value={merged.normalize_mode_labels ?? true} onChange={(v) => handleChange('normalize_mode_labels', v)} disabled={frozen} />
        <ToggleRow label={t.excludeBTB} value={merged.career_top_exclude_btb ?? false} onChange={(v) => handleChange('career_top_exclude_btb', v)} disabled={frozen} />
        <ToggleRow label={t.refreshClearsCaches} value={merged.refresh_clears_caches ?? false} onChange={(v) => handleChange('refresh_clears_caches', v)} disabled={frozen} />
      </CardContent>
    </Card>
  )
}

export function DiscordCard({ merged, handleChange, t, frozen }: TabProps) {
  return (
    <>
      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t.discordTitle}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-1 divide-y divide-border/50">
          <ToggleRow label={t.discordEnabled} value={merged.discord_notifications_enabled ?? false} onChange={(v) => handleChange('discord_notifications_enabled', v)} disabled={frozen} />
          <ToggleRow label={t.discordNotifySync} value={merged.discord_notify_sync ?? false} onChange={(v) => handleChange('discord_notify_sync', v)} disabled={frozen || !merged.discord_notifications_enabled} />
          <ToggleRow label={t.discordNotifyBackfill} value={merged.discord_notify_backfill ?? false} onChange={(v) => handleChange('discord_notify_backfill', v)} disabled={frozen || !merged.discord_notifications_enabled} />
          <ToggleRow label={t.discordNotifyNewVersion} value={merged.discord_notify_new_version ?? false} onChange={(v) => handleChange('discord_notify_new_version', v)} disabled={frozen || !merged.discord_notifications_enabled} />
          <ToggleRow label={t.discordNotifyFriends} value={merged.discord_notify_friends ?? false} onChange={(v) => handleChange('discord_notify_friends', v)} disabled={frozen || !merged.discord_notifications_enabled} />
        </CardContent>
      </Card>

      {merged.discord_notifications_enabled && !merged.discord_webhook_url_present && (
        <div className="rounded-lg border border-warning bg-warning/10 px-4 py-3 text-sm text-warning">
          ⚠️ {t.discordNoWebhook}
        </div>
      )}
    </>
  )
}

export function MediaCard({ merged, handleChange, t, frozen }: TabProps) {
  const scanMedia = useScanMedia()
  return (
    <>
      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t.mediaTitle}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-1 divide-y divide-border/50">
          <ToggleRow label={t.mediaWatcherEnabled} value={merged.media_watcher_enabled ?? false} onChange={(v) => handleChange('media_watcher_enabled', v)} disabled={frozen} />
          <div className="flex flex-col gap-1 py-2">
            <span className="text-sm text-foreground">{t.mediaBaseDirLabel}</span>
            <input
              type="text"
              value={merged.media_captures_base_dir ?? ''}
              onChange={(e) => handleChange('media_captures_base_dir', e.target.value)}
              placeholder={t.mediaBaseDirPlaceholder}
              disabled={frozen}
              className="w-full rounded border border-input bg-background px-2 py-1 text-sm font-mono text-foreground disabled:cursor-not-allowed disabled:opacity-50"
            />
            <span className="text-xs text-muted-foreground">{t.mediaBaseDirHint}</span>
          </div>
          <div>
            <ToggleRow
              label={t.mediaDeleteSource}
              value={merged.media_delete_source_after_transcode ?? false}
              onChange={(v) => handleChange('media_delete_source_after_transcode', v)}
              disabled={frozen}
            />
            <span className="block pb-2 text-xs text-muted-foreground">{t.mediaDeleteSourceHint}</span>
          </div>
          <div className="flex items-center justify-between py-2">
            <span className="text-sm text-foreground">{t.mediaToleranceLabel}</span>
            <input
              type="number"
              value={merged.media_tolerance_minutes ?? 10}
              onChange={(e) => handleChange('media_tolerance_minutes', Number(e.target.value))}
              disabled={frozen}
              className="w-20 rounded border border-input bg-background px-2 py-1 text-right text-sm text-foreground disabled:cursor-not-allowed disabled:opacity-50"
              min={1}
              max={60}
            />
          </div>
          <div className="flex items-center justify-between py-2">
            <Button
              variant="outline"
              size="sm"
              disabled={frozen || scanMedia.isPending}
              onClick={() => scanMedia.mutate()}
            >
              {scanMedia.isPending
                ? t.mediaScanRunning
                : scanMedia.isSuccess
                  ? t.mediaScanDone
                  : scanMedia.isError
                    ? t.mediaScanError
                    : t.mediaScanButton}
            </Button>
          </div>
        </CardContent>
      </Card>

      {merged.media_watcher_enabled && !merged.media_captures_base_dir && (
        <div className="rounded-lg border border-warning bg-warning/10 px-4 py-3 text-sm text-warning">
          ⚠️ {t.mediaNoBaseDir}
        </div>
      )}
    </>
  )
}
