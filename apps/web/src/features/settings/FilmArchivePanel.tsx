/**
 * FilmArchivePanel — l'archive des films : conserver ou non, où, et ce qu'elle occupe (lot L14,
 * ADR 0045 D-12). La même carte sert les Réglages (onglet Données) et l'assistant de premier
 * lancement (`variant="wizard"`, où elle remplace le titre de carte par une phrase d'introduction).
 *
 * Chaque changement part tout de suite en PATCH /settings, comme le reste des Réglages. Les refus
 * du serveur — combinaison « sans archive + rétention finie », dossier non vide, archive qui
 * contient déjà des films — arrivent rédigés dans la langue de la requête et s'affichent tels
 * quels : la carte ne les reformule pas.
 */
import { useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { formatBytes } from '@/lib/formatters/bytes'
import { apiErrorMessage } from '@/lib/api/client'
import { queryKeys } from '@/lib/query/keys'
import { useAppShellStore } from '@/stores/appShellStore'
import { useDeploymentCapability } from '@/stores/useDeploymentCapability'
import { useSettings, useUpdateSettings } from './queries'
import { useFilmArchiveStatus, type FilmArchiveStatus } from './filmArchiveQueries'
import { getFilmArchiveText, type FilmArchiveText } from './filmArchiveText'
import { ToggleRow } from './_settingsShared'
import type { UpdateSettingsRequest } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'

interface FilmArchivePanelProps {
  variant?: 'card' | 'wizard'
}

export function FilmArchivePanel({ variant = 'card' }: FilmArchivePanelProps) {
  const locale = useAppShellStore((s) => s.locale)
  const t = getFilmArchiveText(locale)
  const frozen = !useDeploymentCapability('can_edit_settings')
  const qc = useQueryClient()
  const { data: settings } = useSettings()
  const { data: status, isError } = useFilmArchiveStatus()
  const update = useUpdateSettings()
  const [dirDraft, setDirDraft] = useState<string | null>(null)
  const [message, setMessage] = useState<{ ok: boolean; text: string } | null>(null)

  const archiveFilms = settings?.archive_films ?? true
  const dir = dirDraft ?? settings?.film_archive_dir ?? ''

  function save(req: UpdateSettingsRequest) {
    setMessage(null)
    update.mutate(req, {
      onSuccess: () => {
        setMessage({ ok: true, text: t.saved })
        setDirDraft(null)
        void qc.invalidateQueries({ queryKey: queryKeys.filmArchive })
      },
      onError: (err) => setMessage({ ok: false, text: apiErrorMessage(err) ?? t.saveFailed }),
    })
  }

  const body = (
    <div className="space-y-3">
      {variant === 'wizard' && <p className="text-sm text-muted-foreground">{t.wizardIntro}</p>}
      <fieldset disabled={frozen} className="m-0 min-w-0 space-y-3 border-0 p-0">
        <div>
          <ToggleRow
            label={t.toggleLabel}
            value={archiveFilms}
            onChange={(v) => save({ archive_films: v })}
          />
          <p className="text-xs text-muted-foreground">
            {archiveFilms ? t.toggleHintOn : t.toggleHintOff}
          </p>
        </div>
        <LocationField
          t={t}
          dir={dir}
          status={status}
          onChange={setDirDraft}
          onSave={() => save({ film_archive_dir: dir.trim() })}
          dirty={dirDraft !== null && dirDraft !== (settings?.film_archive_dir ?? '')}
          saving={update.isPending}
        />
      </fieldset>
      {message && (
        <p
          role={message.ok ? 'status' : 'alert'}
          className={`text-xs ${message.ok ? 'text-success' : 'text-destructive'}`}
        >
          {message.text}
        </p>
      )}
      {variant === 'card' && <Occupancy t={t} locale={locale} status={status} failed={isError} />}
    </div>
  )

  if (variant === 'wizard') return body
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">{t.title}</CardTitle>
      </CardHeader>
      <CardContent>{body}</CardContent>
    </Card>
  )
}

interface LocationFieldProps {
  t: FilmArchiveText
  dir: string
  status: FilmArchiveStatus | undefined
  dirty: boolean
  saving: boolean
  onChange: (v: string) => void
  onSave: () => void
}

function LocationField({ t, dir, status, dirty, saving, onChange, onSave }: LocationFieldProps) {
  const hasFilms = (status?.films ?? 0) > 0
  return (
    <div className="space-y-1">
      <label className="text-sm text-foreground" htmlFor="film-archive-dir">
        {t.locationLabel}
      </label>
      <div className="flex gap-2">
        <input
          id="film-archive-dir"
          type="text"
          className="min-w-0 flex-1 rounded border border-border bg-background px-2 py-1 text-sm text-foreground"
          value={dir}
          placeholder={status?.dir ?? t.locationPlaceholder}
          onChange={(e) => onChange(e.target.value)}
        />
        <Button size="sm" variant="outline" disabled={!dirty || saving} onClick={onSave}>
          {t.locationSave}
        </Button>
      </div>
      <p className="text-xs text-muted-foreground">{t.locationHint}</p>
      {hasFilms && <p className="text-xs text-muted-foreground">{t.moveHint}</p>}
      {status?.restart_required && (
        <p className="text-xs text-warning" role="status">
          {t.restartRequired}
        </p>
      )}
    </div>
  )
}

interface OccupancyProps {
  t: FilmArchiveText
  locale: Locale
  status: FilmArchiveStatus | undefined
  failed: boolean
}

function Occupancy({ t, locale, status, failed }: OccupancyProps) {
  if (failed) return <p className="text-xs text-muted-foreground">{t.unavailable}</p>
  if (!status) return <p className="text-xs text-muted-foreground">{t.loading}</p>
  const films = `${status.films.toLocaleString(locale)} ${status.films === 1 ? t.filmOne : t.filmMany}`
  const volume = status.volume
  return (
    <dl className="space-y-1 border-t border-border/50 pt-3 text-sm">
      <div className="flex justify-between gap-3">
        <dt className="text-muted-foreground">{t.occupancyLabel}</dt>
        <dd className="text-foreground">
          {films} · {formatBytes(status.bytes, locale)}
        </dd>
      </div>
      {volume && (
        <div className="flex justify-between gap-3">
          <dt className="text-muted-foreground">{t.volumeLabel}</dt>
          <dd className="text-foreground">
            {t.volumeFree
              .replace('{free}', formatBytes(volume.free_bytes, locale))
              .replace('{total}', formatBytes(volume.total_bytes, locale))}
            {volume.status === 'warn' && <span className="ml-2 text-warning">{t.volumeWarn}</span>}
            {volume.status === 'critical' && (
              <span className="ml-2 text-destructive">{t.volumeCritical}</span>
            )}
          </dd>
        </div>
      )}
      {status.partial && <p className="text-xs text-muted-foreground">{t.partial}</p>}
    </dl>
  )
}
