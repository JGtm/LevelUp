/**
 * SquadObjectivesPanel — panneau « objectifs » du bandeau Cap d'escouade :
 * défis actifs (label localisé, état Rejoint, progression par membre, abandon)
 * + génération/création de défis depuis le pool coach. Extrait de
 * SquadFocusStrip.tsx pour le seuil 500 L (CLAUDE.md règle 5) et la séparation
 * de la logique de composant (règle diagnostic #7).
 */
import { useState } from 'react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { apiErrorMessage } from '@/lib/api/client'
import { useAppShellStore } from '@/stores/appShellStore'
import {
  useSquadChallenges,
  useEvaluateSquadChallenge,
  useRefreshSquadPool,
  useCreateSquadChallenge,
  useAbandonSquadChallenge,
} from '@/features/prestige/hooks/useSquads'
import { useJoinSquadChallenge } from '@/features/prestige/hooks'
import type { SquadMember, SquadParticipantProgress } from '@/lib/prestige'
import type { SquadFocusText } from './squadFocusStrings'

export function SquadObjectivesPanel({
  squadId,
  playerSlug,
  members,
  t,
}: {
  squadId: string
  playerSlug: string
  members: SquadMember[]
  t: SquadFocusText
}) {
  const locale = useAppShellStore((s) => s.locale)
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  const { data } = useSquadChallenges(squadId, playerSlug)
  const join = useJoinSquadChallenge()
  const evaluate = useEvaluateSquadChallenge(squadId)
  const refreshPool = useRefreshSquadPool()
  const createChallenge = useCreateSquadChallenge(squadId)
  const abandon = useAbandonSquadChallenge(squadId)
  const challenges = data?.squad_challenges ?? []
  const pool = refreshPool.data?.pool ?? []

  // Progression par défi renvoyée par « Réévaluer » (transitoire : la réponse
  // d'évaluation n'est pas dans le cache liste — on la garde le temps de l'afficher).
  const [progressByChallenge, setProgressByChallenge] = useState<
    Record<string, SquadParticipantProgress[]>
  >({})
  // Défi en attente de confirmation de suppression (2e clic), pattern du footer squad.
  const [confirmingDelete, setConfirmingDelete] = useState<string | null>(null)

  const onEvaluate = (challengeId: string) =>
    evaluate.mutate(
      { id: challengeId, requestedBy: playerSlug },
      {
        onSuccess: (res) => {
          setProgressByChallenge((prev) => ({ ...prev, [challengeId]: res.progress }))
          toast.success(t.evaluated)
        },
        onError: (err) => toast.error(apiErrorMessage(err) ?? t.evalError),
      },
    )

  const onJoin = (challengeId: string) =>
    join.mutate(
      { challengeId, userId: playerSlug, squadId },
      {
        onSuccess: () => toast.success(t.joinedToast),
        onError: (err) => toast.error(apiErrorMessage(err) ?? t.joinError),
      },
    )

  const onDelete = (challengeId: string) => {
    if (confirmingDelete !== challengeId) {
      setConfirmingDelete(challengeId)
      return
    }
    setConfirmingDelete(null)
    abandon.mutate(
      { id: challengeId, requestedBy: playerSlug },
      {
        onSuccess: () => toast.success(t.challengeDeleted),
        onError: (err) => toast.error(apiErrorMessage(err) ?? t.challengeDeleteError),
      },
    )
  }

  return (
    <div className="mt-3 space-y-3">
      {/* Défis actifs */}
      {challenges.length === 0 ? (
        <p className="text-xs text-muted-foreground">{t.noObjectives}</p>
      ) : (
        <ul className="space-y-2">
          {challenges.map((c) => {
            const label = (locale === 'en' ? c.label_en : c.label_fr) || t.challenge
            const isParticipant = c.participants.some((p) => p.user_id === playerSlug)
            return (
              <li
                key={c.id}
                className="rounded-md border border-border bg-background px-3 py-2"
              >
                <div className="flex items-center justify-between gap-2">
                  <span className="min-w-0">
                    <span className="flex items-center gap-2">
                      <span className="truncate text-sm">{label}</span>
                      {c.expired ? (
                        <span className="shrink-0 text-2xs font-semibold text-muted-foreground">
                          {t.expired}
                        </span>
                      ) : null}
                    </span>
                    {c.target_per_member ? (
                      <span className="text-2xs text-muted-foreground">
                        {t.target(c.target_per_member)}
                      </span>
                    ) : null}
                  </span>
                  <span className="flex shrink-0 gap-2">
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => onEvaluate(c.id)}
                      disabled={evaluate.isPending}
                    >
                      {t.evaluate}
                    </Button>
                    <Button
                      size="sm"
                      variant={isParticipant ? 'outline' : 'default'}
                      onClick={() => onJoin(c.id)}
                      disabled={isParticipant || c.expired || join.isPending}
                    >
                      {isParticipant ? t.joined : t.join}
                    </Button>
                    <Button
                      size="sm"
                      variant="outline"
                      className={confirmingDelete === c.id ? 'text-destructive' : undefined}
                      onClick={() => onDelete(c.id)}
                      disabled={abandon.isPending}
                    >
                      {confirmingDelete === c.id ? t.confirmDelete : t.deleteCta}
                    </Button>
                  </span>
                </div>
                <SquadProgressList
                  progress={progressByChallenge[c.id]}
                  targetPerMember={c.target_per_member}
                  members={members}
                  t={t}
                />
              </li>
            )
          })}
        </ul>
      )}

      {/* Proposer puis créer un défi depuis le pool (biaisé coach). */}
      <div className="border-t border-border pt-3">
        <Button
          variant="outline"
          size="sm"
          onClick={() =>
            refreshPool.mutate(
              { squadId, titleSlug, requestedBy: playerSlug },
              { onError: (err) => toast.error(apiErrorMessage(err) ?? t.poolError) },
            )
          }
          disabled={refreshPool.isPending}
        >
          {refreshPool.isPending ? t.proposing : t.propose}
        </Button>

        {refreshPool.isSuccess && pool.length === 0 ? (
          <p className="mt-2 text-xs text-muted-foreground">{t.poolEmpty}</p>
        ) : null}

        {pool.length > 0 ? (
          <p className="mt-2 text-2xs text-muted-foreground">{t.poolCaption}</p>
        ) : null}

        {pool.length > 0 ? (
          <ul className="mt-2 space-y-2">
            {pool.map((tpl) => (
              <li
                key={tpl.id}
                className="flex items-center justify-between gap-2 rounded-md border border-border bg-background px-3 py-2"
              >
                <span className="truncate text-sm">
                  {locale === 'en' ? tpl.label_en : tpl.label_fr}
                </span>
                <Button
                  size="sm"
                  onClick={() =>
                    createChallenge.mutate(
                      {
                        template_id: tpl.id,
                        title_slug: tpl.title_slug,
                        mode: 'collective',
                        eval_type: tpl.eval_type,
                        window_type: tpl.window_type,
                        window_value: tpl.window_value,
                        target_per_member: tpl.normal_target,
                        created_by: playerSlug,
                      },
                      {
                        onSuccess: () => toast.success(t.created),
                        onError: (err) => toast.error(apiErrorMessage(err) ?? t.createError),
                      },
                    )
                  }
                  disabled={createChallenge.isPending}
                >
                  {createChallenge.isPending ? t.creating : t.create}
                </Button>
              </li>
            ))}
          </ul>
        ) : null}
      </div>
    </div>
  )
}

// SquadProgressList affiche la progression par membre d'un défi, telle que
// renvoyée par « Réévaluer » (valeur cumulée, nb de matchs comptés, atteinte de
// la cible). Le gamertag est résolu depuis le roster (fallback xuid tronqué).
// Rien tant qu'aucune évaluation n'a été lancée pour ce défi.
function SquadProgressList({
  progress,
  targetPerMember,
  members,
  t,
}: {
  progress: SquadParticipantProgress[] | undefined
  targetPerMember?: number
  members: SquadMember[]
  t: SquadFocusText
}) {
  if (!progress || progress.length === 0) return null
  const gamertagByXuid = new Map(members.map((m) => [m.xuid, m.gamertag]))
  return (
    <ul className="mt-2 space-y-1 border-t border-border pt-2">
      {progress.map((p, idx) => {
        const name = gamertagByXuid.get(p.xuid) || p.xuid.slice(0, 8)
        return (
          // Clé composite : le xuid peut être vide (joueurs H5 sans xuid) → collisions
          // key="" entre entrées. L'index désambiguïse.
          <li key={`${p.xuid}||${idx}`} className="flex items-center justify-between gap-2 text-2xs">
            <span className="truncate text-muted-foreground">{name}</span>
            <span className="flex shrink-0 items-center gap-2">
              <span className="tabular-nums">
                {Math.round(p.value)}
                {targetPerMember ? ` / ${targetPerMember}` : ''}
              </span>
              <span className="text-muted-foreground">{t.matchesN(p.matches)}</span>
              {p.completed ? (
                <span className="font-semibold text-success">{t.reached}</span>
              ) : null}
            </span>
          </li>
        )
      })}
    </ul>
  )
}
