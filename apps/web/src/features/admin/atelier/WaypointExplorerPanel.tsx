/**
 * WaypointExplorerPanel — onglet « Explorateur d'API » de l'Atelier.
 *
 * Interroge en direct l'API Discovery UGC de Halo Waypoint (résolution d'un
 * asset par segment / identifiant / version). Backend : GET /lab/waypoint
 * (admin-only ; réutilise reg.AnyPlayerTokens + halo.FetchAsset). Les erreurs
 * d'appel (404 / auth / token absent) reviennent dans la réponse (resolved_ok
 * = false) et sont affichées ici — pas une erreur réseau.
 */
import { useState } from 'react'

import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'
import { apiErrorMessage } from '@/lib/api/client'
import type { LabWaypointResponse } from '@/lib/api/types'
import type { AdminManifestKey } from '@/lib/i18n/generated/admin'
import { useLabWaypoint } from '@/features/lab/queries'
import { useAdminT, type TAdmin } from '../useAdminText'

const SEGMENTS: ReadonlyArray<{ value: string; labelKey: AdminManifestKey }> = [
  { value: 'map', labelKey: 'admin.atelier.seg_map' },
  { value: 'playlist', labelKey: 'admin.atelier.seg_playlist' },
  { value: 'pair', labelKey: 'admin.atelier.seg_pair' },
  { value: 'game_variant', labelKey: 'admin.atelier.seg_game_variant' },
]

const INPUT_CLASS = 'w-full rounded-xl border border-input px-3 py-2 text-sm'

export function WaypointExplorerPanel() {
  const tA = useAdminT()
  const explore = useLabWaypoint()
  const [segment, setSegment] = useState('map')
  const [assetID, setAssetID] = useState('')
  const [versionID, setVersionID] = useState('')
  const [lang, setLang] = useState('en-US')

  const canRun = assetID.trim() !== '' && versionID.trim() !== '' && !explore.isPending

  function run() {
    explore.mutate({
      segment,
      assetID: assetID.trim(),
      versionID: versionID.trim(),
      lang: lang.trim() || undefined,
    })
  }

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle className="text-base">{tA('admin.atelier.tab_api')}</CardTitle>
          <CardDescription>{tA('admin.atelier.api_intro')}</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
            <label className="space-y-1">
              <span className="text-xs text-muted-foreground">{tA('admin.atelier.api_segment')}</span>
              <select value={segment} onChange={(e) => setSegment(e.target.value)} className={INPUT_CLASS}>
                {SEGMENTS.map((s) => (
                  <option key={s.value} value={s.value}>
                    {tA(s.labelKey)}
                  </option>
                ))}
              </select>
            </label>
            <label className="space-y-1">
              <span className="text-xs text-muted-foreground">{tA('admin.atelier.api_asset_id')}</span>
              <input value={assetID} onChange={(e) => setAssetID(e.target.value)} className={INPUT_CLASS} />
            </label>
            <label className="space-y-1">
              <span className="text-xs text-muted-foreground">{tA('admin.atelier.api_version_id')}</span>
              <input value={versionID} onChange={(e) => setVersionID(e.target.value)} className={INPUT_CLASS} />
            </label>
            <label className="space-y-1">
              <span className="text-xs text-muted-foreground">{tA('admin.atelier.api_lang')}</span>
              <input value={lang} onChange={(e) => setLang(e.target.value)} className={INPUT_CLASS} />
            </label>
          </div>
          <Button onClick={run} disabled={!canRun}>
            {explore.isPending ? tA('admin.atelier.api_running') : tA('admin.atelier.api_run')}
          </Button>
          {explore.isError && (
            <p className="text-sm" style={{ color: tokenCssVar('destructive') }}>
              {apiErrorMessage(explore.error) ?? tA('admin.atelier.api_failed')}
            </p>
          )}
        </CardContent>
      </Card>

      {explore.data && <WaypointResult result={explore.data} tA={tA} />}
    </div>
  )
}

function WaypointResult({ result, tA }: { result: LabWaypointResponse; tA: TAdmin }) {
  const ok = result.resolved_ok
  const statusLabel = ok ? tA('admin.atelier.api_resolved') : tA('admin.atelier.api_failed')
  return (
    <Card>
      <CardHeader className="pb-3">
        <div className="flex flex-wrap items-center gap-3">
          <span className="text-sm font-semibold" style={{ color: tokenCssVar(ok ? 'success' : 'destructive') }}>
            {statusLabel}
          </span>
          <span className="font-mono text-xs text-muted-foreground">
            {`${result.endpoint}/${result.asset_id} · ${result.latency_ms} ms`}
          </span>
        </div>
      </CardHeader>
      <CardContent className="space-y-3 text-sm">
        {ok ? (
          <div className="grid gap-2 md:grid-cols-2">
            <p>
              {tA('admin.atelier.api_name')}: <span className="font-medium text-foreground">{result.asset_name}</span>
            </p>
            {result.description ? (
              <p>
                {tA('admin.atelier.api_description')}:{' '}
                <span className="font-medium text-foreground">{result.description}</span>
              </p>
            ) : null}
            {result.image_url ? (
              <div className="md:col-span-2">
                <p className="text-xs text-muted-foreground">{tA('admin.atelier.api_image')}</p>
                <img
                  src={result.image_url}
                  alt={result.asset_name ?? ''}
                  className="mt-1 max-h-40 rounded-lg border border-border"
                />
              </div>
            ) : null}
          </div>
        ) : (
          <p style={{ color: tokenCssVar('destructive') }}>{result.error}</p>
        )}
      </CardContent>
    </Card>
  )
}
