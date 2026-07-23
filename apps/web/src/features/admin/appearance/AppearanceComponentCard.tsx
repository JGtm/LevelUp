/**
 * AppearanceComponentCard — diagnostic d'UN composant du Spartan ID (bannière /
 * emblème / arrière-plan / indicatif de service). Vignette de la valeur servie
 * (image ou texte), badge de verdict (StatusBadge canonique), et explication
 * DÉPLIABLE : le POURQUOI (detail technique traduit) + « quoi faire » (rien /
 * attendre / réauthentifier). Aucune couleur en dur — tokens via StatusBadge.
 */
import { useState } from 'react'

import { Card } from '@/components/ui/card'
import { API_BASE_URL } from '@/lib/api/client'
import type { AppearanceComponentDiagnosis } from '@/lib/api/types'
import { StatusBadge } from '../components/StatusBadge'
import { useAdminT } from '../useAdminText'
import {
  componentLabelKey,
  detailExplanationKey,
  isImageComponent,
  servedFromKey,
  verdictActionKey,
  verdictActionKind,
  verdictBadgeStatus,
  verdictLabelKey,
} from './appearanceDiagDisplay'

// Entrée SSO existante (RedirectFlowPanel.handleClick) : redirect plein-page vers
// le backend qui génère l'état CSRF puis redirige vers Microsoft /authorize.
const SSO_LOGIN_HREF = `${API_BASE_URL}/auth/xbox/login`

export function AppearanceComponentCard({ diag }: { diag: AppearanceComponentDiagnosis }) {
  const tA = useAdminT()
  const label = tA(componentLabelKey(diag.component))
  const actionKind = verdictActionKind(diag.verdict)

  return (
    <Card className="flex flex-col gap-3 p-4">
      <div className="flex items-start justify-between gap-2">
        <h4 className="text-sm font-semibold text-foreground">{label}</h4>
        <StatusBadge
          status={verdictBadgeStatus(diag.verdict)}
          label={tA(verdictLabelKey(diag.verdict))}
          title={tA(verdictActionKey(diag.verdict))}
        />
      </div>

      <ServedValueThumbnail diag={diag} label={label} />

      <p className="text-xs text-muted-foreground">{tA(servedFromKey(diag.served_from))}</p>

      <details className="group text-sm">
        <summary className="cursor-pointer select-none text-xs font-medium uppercase tracking-wide text-muted-foreground marker:content-none">
          {tA('admin.appearance.why_toggle')}
        </summary>
        <div className="mt-2 space-y-3">
          <div>
            <p className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
              {tA('admin.appearance.why_label')}
            </p>
            <p className="mt-0.5 text-sm text-foreground">
              {tA(detailExplanationKey(diag.verdict, diag.detail))}
            </p>
            {diag.detail && (
              <p className="mt-0.5 font-mono text-xs text-muted-foreground">{diag.detail}</p>
            )}
          </div>
          <div>
            <p className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
              {tA('admin.appearance.what_to_do')}
            </p>
            <p className="mt-0.5 text-sm text-foreground">
              {tA(verdictActionKey(diag.verdict))}
            </p>
            {actionKind === 'reauth' && (
              <a
                href={SSO_LOGIN_HREF}
                className="mt-2 inline-block text-sm font-medium text-primary underline underline-offset-2 hover:text-foreground"
              >
                {tA('admin.appearance.reauth_cta')}
              </a>
            )}
          </div>
        </div>
      </details>
    </Card>
  )
}

/**
 * Vignette de la valeur servie : image bornée (bannière/emblème/arrière-plan) ou
 * texte (indicatif de service). Fallback propre quand l'URL est vide (composant
 * absent → état vide explicite) ou que l'image ne charge pas (onError).
 */
function ServedValueThumbnail({
  diag,
  label,
}: {
  diag: AppearanceComponentDiagnosis
  label: string
}) {
  const tA = useAdminT()
  const [imgFailed, setImgFailed] = useState(false)
  const value = diag.served_value.trim()

  if (!value) return <EmptyThumbnail message={tA('admin.appearance.no_served_value')} />

  if (isImageComponent(diag.component)) {
    if (imgFailed) return <EmptyThumbnail message={tA('admin.appearance.no_served_value')} />
    return (
      <div className="flex items-center justify-center rounded-md border bg-muted/30 p-2">
        <img
          src={value}
          alt={label}
          loading="lazy"
          onError={() => setImgFailed(true)}
          className="max-h-24 max-w-full object-contain"
        />
      </div>
    )
  }

  // Indicatif de service : texte.
  return (
    <div className="rounded-md border bg-muted/30 px-3 py-4 text-center">
      <span className="font-mono text-lg font-semibold tracking-widest text-foreground">
        {value}
      </span>
    </div>
  )
}

function EmptyThumbnail({ message }: { message: string }) {
  return (
    <div className="rounded-md border border-dashed border-border bg-muted/40 px-3 py-6 text-center text-xs text-muted-foreground">
      {message}
    </div>
  )
}
