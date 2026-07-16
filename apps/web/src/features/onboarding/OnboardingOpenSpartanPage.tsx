/**
 * OnboardingOpenSpartanPage — landing page shown right after a successful
 * Xbox SSO login. The default action is informational ("sync running"), with
 * an opt-in disclosure that reveals the OpenSpartan import card.
 *
 * Intentionally not gated by a "first login" flag in v1 — returning users
 * can still reach the import via this route and just click "Continuer".
 * If desired later, a redirect-on-second-visit guard can be added by
 * checking bootstrap.user.created_at.
 */
import { useEffect, useState } from 'react'
import { useNavigate } from '@tanstack/react-router'

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { OpenSpartanImportCard } from './OpenSpartanImportCard'
import { SetPasswordCard } from '@/features/auth/SetPasswordCard'
import { useAppShellStore } from '@/stores/appShellStore'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest, type CommonManifestKey } from '@/lib/i18n/generated/common'

export function OnboardingOpenSpartanPage() {
  const navigate = useNavigate()
  const [showAdvanced, setShowAdvanced] = useState(false)
  const hasPassword = useAppShellStore((s) => s.hasPassword)
  const locale = useAppShellStore((s) => s.locale)
  const t = (key: CommonManifestKey) => formatMessage(commonManifest, key, locale)

  // Garde défense-en-profondeur : un joueur DÉJÀ établi (profil prêt + données
  // synchronisées) ne doit jamais rester coincé sur l'onboarding « on synchronise
  // tes derniers matchs » — il y va parfois par un vieux lien ou le flux redirect
  // SSO. On le renvoie direct au dashboard. Le vrai nouveau joueur (setup_state
  // != 'ready') reste sur cette page. Gaté sur isBootstrapped pour éviter une
  // redirection prématurée pré-hydratation.
  const isBootstrapped = useAppShellStore((s) => s.isBootstrapped)
  const setupState = useAppShellStore((s) => s.setupState)
  const currentPlayer = useAppShellStore((s) => s.currentPlayer)
  useEffect(() => {
    if (isBootstrapped && setupState === 'ready' && currentPlayer) {
      navigate({ to: '/' })
    }
  }, [isBootstrapped, setupState, currentPlayer, navigate])

  return (
    <div className="min-h-screen bg-background flex items-center justify-center p-4">
      <div className="w-full max-w-2xl space-y-6">
        <div className="flex flex-col items-center gap-2">
          <img src="/logo-full-inline.png" alt="LevelUp" className="h-16 shrink-0 object-contain" />
        </div>

        <Card>
          <CardHeader>
            <CardTitle>{t('common.onboarding.welcome')}</CardTitle>
            <CardDescription>
              {t('common.onboarding.sync_running_intro')}
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="flex justify-end">
              <Button
                onClick={() => navigate({ to: '/' })}
                data-testid="onboarding-continue"
              >
                Continuer →
              </Button>
            </div>
          </CardContent>
        </Card>

        {/* PR-C : opt-in mot de passe (re-login rapide). Masqué si déjà défini. */}
        {!hasPassword && <SetPasswordCard />}

        {/* HTML5 <details> for native accessible disclosure. No extra deps. */}
        <details
          className="rounded-lg border border-border bg-card text-card-foreground"
          onToggle={(e) => setShowAdvanced((e.target as HTMLDetailsElement).open)}
        >
          <summary className="cursor-pointer select-none p-4 text-sm font-medium hover:bg-muted/50 transition-colors">
            {t('common.onboarding.advanced_options')}
          </summary>
          <div className="border-t border-border p-4">
            <p className="mb-4 text-sm text-muted-foreground">
              {t('common.onboarding.openspartan_intro')} <code>.db</code> {t('common.onboarding.openspartan_intro_suffix')}
            </p>
            {showAdvanced && <OpenSpartanImportCard />}
          </div>
        </details>
      </div>
    </div>
  )
}
