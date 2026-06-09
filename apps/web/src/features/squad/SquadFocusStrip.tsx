/**
 * SquadFocusStrip — « Cap d'escouade » (Phase C).
 *
 * Bandeau prospectif en tête de l'onglet Synergies : si la composition courante
 * correspond à une escouade enregistrée, affiche ses objectifs (défis) et permet
 * de les gérer (rejoindre / réévaluer). Sinon, propose d'enregistrer la compo
 * comme escouade. Élément forward-looking, isolé du contenu rétrospectif.
 *
 * Identité d'escouade : roster clé xuid (cf. backend Phase C). La sélection de
 * coéquipiers (hors joueur courant) est matchée au roster exact d'une escouade.
 */
import { useMemo, useState } from 'react'
import { Button } from '@/components/ui/button'
import { useAppShellStore } from '@/stores/appShellStore'
import { useSquadContext } from './SquadContext'
import {
  useMySquads,
  useSquadChallenges,
  useCreateSquad,
  useEvaluateSquadChallenge,
} from '@/features/prestige/hooks/useSquads'
import { useJoinSquadChallenge } from '@/features/prestige/hooks'
import type { SquadWithMembers } from '@/lib/prestige'

const STRINGS = {
  fr: {
    title: "Cap d'escouade",
    saved: (name: string) => `Escouade « ${name} »`,
    objectives: (n: number) => `${n} objectif${n > 1 ? 's' : ''}`,
    manage: 'Gérer les objectifs',
    hide: 'Masquer',
    saveCta: 'Enregistrer cette compo comme escouade',
    saving: 'Enregistrement…',
    noObjectives: 'Aucun objectif actif pour cette escouade.',
    join: 'Rejoindre',
    joined: 'Rejoint',
    evaluate: 'Réévaluer',
    challenge: 'Défi',
  },
  en: {
    title: 'Squad focus',
    saved: (name: string) => `Squad “${name}”`,
    objectives: (n: number) => `${n} objective${n > 1 ? 's' : ''}`,
    manage: 'Manage objectives',
    hide: 'Hide',
    saveCta: 'Save this lineup as a squad',
    saving: 'Saving…',
    noObjectives: 'No active objective for this squad.',
    join: 'Join',
    joined: 'Joined',
    evaluate: 'Re-evaluate',
    challenge: 'Challenge',
  },
}

export function SquadFocusStrip() {
  const { selectedRows, playerSlug } = useSquadContext()
  const locale = useAppShellStore((s) => s.locale)
  const t = STRINGS[locale === 'en' ? 'en' : 'fr']
  const [open, setOpen] = useState(false)

  const selectionXuids = useMemo(
    () => selectedRows.map((r) => r.xuid).filter((x): x is string => !!x),
    [selectedRows],
  )

  const { data: mySquads } = useMySquads(playerSlug)
  const createSquad = useCreateSquad(playerSlug)

  // Match : escouade dont le roster hors joueur courant == sélection (exact).
  const matched = useMemo<SquadWithMembers | null>(() => {
    if (!mySquads?.squads || selectionXuids.length === 0) return null
    const sel = new Set(selectionXuids)
    return (
      mySquads.squads.find((sm) => {
        const others = sm.members.filter((m) => m.user_id !== playerSlug).map((m) => m.xuid)
        return others.length === sel.size && others.every((x) => sel.has(x))
      }) ?? null
    )
  }, [mySquads, selectionXuids, playerSlug])

  // Pas de composition exploitable → rien à afficher.
  if (selectionXuids.length === 0) return null

  const onSave = () => {
    const members = selectedRows
      .filter((r) => r.xuid)
      .map((r) => ({ xuid: r.xuid as string, gamertag: r.gamertag }))
    const name = selectedRows.map((r) => r.gamertag).join(' + ')
    createSquad.mutate({ name, created_by: playerSlug, members })
  }

  return (
    <section className="rounded-lg border border-border bg-card p-3">
      <div className="flex items-center justify-between gap-3">
        <div className="min-w-0">
          <p className="text-2xs uppercase tracking-wide text-muted-foreground">{t.title}</p>
          {matched ? (
            <p className="truncate text-sm font-semibold text-foreground">
              {t.saved(matched.squad.name)}
            </p>
          ) : (
            <p className="text-sm text-muted-foreground">{t.saveCta}</p>
          )}
        </div>

        {matched ? (
          <Button variant="outline" size="sm" onClick={() => setOpen((v) => !v)}>
            {open ? t.hide : t.manage}
          </Button>
        ) : (
          <Button size="sm" onClick={onSave} disabled={createSquad.isPending}>
            {createSquad.isPending ? t.saving : t.saveCta}
          </Button>
        )}
      </div>

      {matched && open ? (
        <SquadObjectivesPanel squadId={matched.squad.id} playerSlug={playerSlug} t={t} />
      ) : null}
    </section>
  )
}

function SquadObjectivesPanel({
  squadId,
  playerSlug,
  t,
}: {
  squadId: string
  playerSlug: string
  t: (typeof STRINGS)['fr']
}) {
  const { data } = useSquadChallenges(squadId)
  const join = useJoinSquadChallenge()
  const evaluate = useEvaluateSquadChallenge(squadId)
  const challenges = data?.squad_challenges ?? []

  if (challenges.length === 0) {
    return <p className="mt-3 text-xs text-muted-foreground">{t.noObjectives}</p>
  }

  return (
    <ul className="mt-3 space-y-2">
      {challenges.map((c) => (
        <li
          key={c.id}
          className="flex items-center justify-between gap-2 rounded-md border border-border bg-background px-3 py-2"
        >
          <span className="truncate text-sm">{c.template_id || t.challenge}</span>
          <span className="flex shrink-0 gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => evaluate.mutate({ id: c.id, requestedBy: playerSlug })}
              disabled={evaluate.isPending}
            >
              {t.evaluate}
            </Button>
            <Button
              size="sm"
              onClick={() => join.mutate({ challengeId: c.id, userId: playerSlug })}
              disabled={join.isPending}
            >
              {t.join}
            </Button>
          </span>
        </li>
      ))}
    </ul>
  )
}
