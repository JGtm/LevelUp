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
import { useState } from 'react'
import { useNavigate } from '@tanstack/react-router'

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { OpenSpartanImportCard } from './OpenSpartanImportCard'

export function OnboardingOpenSpartanPage() {
  const navigate = useNavigate()
  const [showAdvanced, setShowAdvanced] = useState(false)

  return (
    <div className="min-h-screen bg-background flex items-center justify-center p-4">
      <div className="w-full max-w-2xl space-y-6">
        <div className="flex flex-col items-center gap-2">
          <img src="/logo-full-inline.png" alt="LevelUp" className="h-16 shrink-0 object-contain" />
        </div>

        <Card>
          <CardHeader>
            <CardTitle>Bienvenue sur LevelUp</CardTitle>
            <CardDescription>
              On synchronise tes derniers matchs depuis Halo Waypoint. Tu peux continuer
              vers le dashboard, la sync tourne en arrière-plan et tes données
              apparaîtront au fur et à mesure.
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

        {/* HTML5 <details> for native accessible disclosure. No extra deps. */}
        <details
          className="rounded-lg border border-border bg-card text-card-foreground"
          onToggle={(e) => setShowAdvanced((e.target as HTMLDetailsElement).open)}
        >
          <summary className="cursor-pointer select-none p-4 text-sm font-medium hover:bg-muted/50 transition-colors">
            Options avancées →
          </summary>
          <div className="border-t border-border p-4">
            <p className="mb-4 text-sm text-muted-foreground">
              Tu as déjà utilisé un autre client Halo qui stocke ses données en local
              (OpenSpartan) ? Tu peux importer son fichier <code>.db</code> ici pour
              récupérer des matchs plus anciens que ce que l'API Halo expose
              aujourd'hui.
            </p>
            {showAdvanced && <OpenSpartanImportCard />}
          </div>
        </details>
      </div>
    </div>
  )
}
