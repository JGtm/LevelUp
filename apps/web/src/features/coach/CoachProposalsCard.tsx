/**
 * CoachProposalsCard — carte affichant les proposals du coach_advisor
 * (ADR 0020 Phase 10).
 *
 * - Affiche le toggle CoachProactiveMode en suggestion si désactivé.
 * - Liste les proposals pending avec boutons Accepter / Ignorer.
 * - Optimistic UI : invalide la query après mutation.
 *
 * À monter sur la page Ascension ou Prestige (decision UI : à la suite du
 * panneau "Séries" ou en pied de page Prestige).
 */

import { useState } from 'react'

import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'

import type { CoachStrings } from './i18n'
import {
  useAcceptCoachProposal,
  useCoachProposals,
  useDismissCoachProposal,
} from './queries'
import type { CoachProposal } from './types'

export interface CoachProposalsCardProps {
  playerSlug: string
  proactiveEnabled: boolean
  t: CoachStrings
}

export function CoachProposalsCard({ playerSlug, proactiveEnabled, t }: CoachProposalsCardProps) {
  const { data, isLoading, isError } = useCoachProposals(playerSlug, 'pending')
  const accept = useAcceptCoachProposal(playerSlug)
  const dismiss = useDismissCoachProposal(playerSlug)
  const [feedback, setFeedback] = useState<string | null>(null)

  // Si le mode n'est pas activé, on affiche un hint d'opt-in plutôt que la liste.
  if (!proactiveEnabled) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t.proposalsTitle}</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-xs text-muted-foreground">{t.proposalsOptInHint}</p>
        </CardContent>
      </Card>
    )
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">{t.proposalsTitle}</CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        {isLoading && <p className="text-xs text-muted-foreground">…</p>}
        {isError && <p className="text-xs text-destructive">{t.proposalsLoadError}</p>}
        {!isLoading && !isError && (data?.items ?? []).length === 0 && (
          <p className="text-xs text-muted-foreground">{t.proposalsEmpty}</p>
        )}
        {feedback && <p className="text-xs text-muted-foreground">{feedback}</p>}
        {(data?.items ?? []).map((p) => (
          <ProposalRow
            key={p.id}
            proposal={p}
            t={t}
            onAccept={async () => {
              try {
                await accept.mutateAsync(p.id)
                setFeedback(t.acceptedSuccess)
              } catch {
                setFeedback(t.acceptError)
              }
            }}
            onDismiss={async () => {
              try {
                await dismiss.mutateAsync(p.id)
                setFeedback(t.dismissedSuccess)
              } catch {
                setFeedback(t.dismissError)
              }
            }}
            accepting={accept.isPending && accept.variables === p.id}
            dismissing={dismiss.isPending && dismiss.variables === p.id}
          />
        ))}
      </CardContent>
    </Card>
  )
}

interface ProposalRowProps {
  proposal: CoachProposal
  t: CoachStrings
  onAccept: () => void
  onDismiss: () => void
  accepting: boolean
  dismissing: boolean
}

function ProposalRow({ proposal, t, onAccept, onDismiss, accepting, dismissing }: ProposalRowProps) {
  const kindLabel = proposal.kind === 'arc' ? t.kindArc : t.kindChallenge
  const originLabel = proposal.origin === 'synthesized' ? t.originSynthesized : t.originCatalog
  const reasonLabel = readableReason(proposal, t)
  return (
    <div className="rounded-md border border-border bg-card/50 p-3">
      <div className="flex items-start justify-between gap-2">
        <div className="space-y-1">
          <p className="text-sm font-medium text-foreground">
            {kindLabel} — {proposal.source_metric ?? proposal.radar_axis ?? proposal.source_signal}
          </p>
          {reasonLabel && (
            <p className="text-xs text-muted-foreground">{reasonLabel}</p>
          )}
          <p className="text-[10px] uppercase tracking-wide text-muted-foreground">
            {t.signal}: {proposal.source_signal} · {t.strength}: {(proposal.strength * 100).toFixed(0)}% · {t.origin}: {originLabel}
            {proposal.suggested_tier && <> · {t.suggestedTier}: {proposal.suggested_tier}</>}
          </p>
        </div>
        <div className="flex shrink-0 gap-2">
          <Button
            size="sm"
            onClick={onAccept}
            disabled={accepting || dismissing}
          >
            {accepting ? t.accepting : t.accept}
          </Button>
          <Button
            variant="outline"
            size="sm"
            onClick={onDismiss}
            disabled={accepting || dismissing}
          >
            {dismissing ? t.dismissing : t.dismiss}
          </Button>
        </div>
      </div>
    </div>
  )
}

/**
 * Tente d'extraire un libellé lisible depuis reason_params (JSON). Le backend
 * y injecte label_fr / label_en selon la locale. Fallback : signal kind.
 */
function readableReason(p: CoachProposal, t: CoachStrings): string {
  if (!p.reason_params) return ''
  try {
    const parsed = JSON.parse(p.reason_params) as Record<string, unknown>
    const labelFR = typeof parsed.label_fr === 'string' ? parsed.label_fr : ''
    const labelEN = typeof parsed.label_en === 'string' ? parsed.label_en : ''
    return labelFR || labelEN || ''
  } catch {
    void t // pas de fallback utilisable depuis t — silently empty
    return ''
  }
}
