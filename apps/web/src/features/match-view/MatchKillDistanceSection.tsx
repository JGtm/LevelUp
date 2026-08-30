/**
 * MatchKillDistanceSection — POC (LOT G.3, 2026-08-30, plan
 * .ai/PLAN_RETOURS_UTILISATEUR_2026-08-29.md §3bis DEC-8) : « kills par arme
 * sur la distance » + distance moyenne par arme, PAR JOUEUR, pour ce match.
 *
 * Cadrage utilisateur MOT POUR MOT (2026-08-30) : « mettre le nombre de kills
 * par armes sur la distance et indiquer la distance moyenne pour chaque
 * arme... pour chaque joueur. Pour le moment c'est tout ce qu'on va faire au
 * niveau de la lecture de la distance. » — vue match UNIQUEMENT, par joueur,
 * kills DU TUEUR seulement (arme et distance de l'ASSISTANT explicitement
 * fermés). Rien d'autre : pas d'agrégat multi-matchs, pas de portée par arme.
 *
 * Gabarit structurel : MatchObjectivesSection.tsx (mêmes classes de tableau,
 * mêmes tokens, même repli sur le scoreboard pour gamertag + total de kills —
 * `combat_tab.kill_distance_by_weapon` ne porte QUE le xuid, cf. domain Go).
 *
 * État vide : `combat_tab.kill_distance_by_weapon` absent/vide (titre sans
 * capture positions, backfill non joué, ou couverture du match sous le
 * plancher mesuré 75,8 %) -> RIEN. Pas de cadre vide, pas de « bientôt
 * disponible » — c'est le cas de la quasi-totalité des matchs tant que le
 * backfill de masse n'a pas tourné en prod.
 */
import type { MatchKillDistancePlayer, MatchScoreboardRow } from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'

import type { MatchViewText } from './i18n'

interface Props {
  players: MatchKillDistancePlayer[] | null | undefined
  scoreboard: MatchScoreboardRow[] | null | undefined
  t: MatchViewText
}

/** Nom + total de kills d'un joueur, résolus depuis le scoreboard déjà chargé. */
function playerContext(
  xuid: string,
  scoreboard: MatchScoreboardRow[],
): { gamertag: string; totalKills: number } {
  const row = scoreboard.find((r) => r.xuid === xuid)
  return { gamertag: row?.gamertag ?? xuid, totalKills: row?.kills ?? 0 }
}

export function MatchKillDistanceSection({ players, scoreboard, t }: Props) {
  const locale = useAppShellStore((s) => s.locale)
  const board = scoreboard ?? []
  const rows = players ?? []

  // État vide soigné : rien à mesurer -> aucune carte (pas de cadre vide).
  if (rows.length === 0) return null

  return (
    <section className="rounded-lg border-2 border-border" aria-label={t.killDistanceTitle}>
      <h3 className="flex items-center gap-2 px-3 py-2 text-sm font-bold uppercase tracking-wider text-foreground">
        {t.killDistanceTitle}
        <span className="rounded-full border border-warning/40 bg-warning/10 px-2 py-0.5 text-3xs font-semibold normal-case tracking-normal text-warning">
          {t.killDistancePocBadge}
        </span>
      </h3>
      <div className="overflow-x-auto">
        <table className="w-full border-collapse text-3xs">
          <thead>
            <tr className="text-muted-foreground">
              <th className="border border-border border-b-2 px-2 py-1 text-left">
                {t.killDistanceColWeapon}
              </th>
              <th className="border border-border border-b-2 px-2 py-1 text-right">
                {t.killDistanceColKills}
              </th>
              <th className="border border-border border-b-2 px-2 py-1 text-right">
                {t.killDistanceColAvg}
              </th>
            </tr>
          </thead>
          {rows.map((player) => {
            const { gamertag, totalKills } = playerContext(player.xuid, board)
            const measured = (player.weapons ?? []).reduce((sum, w) => sum + w.measured_kills, 0)
            const isMe = board.find((r) => r.xuid === player.xuid)?.is_me ?? false
            return (
              <tbody key={player.xuid} className={isMe ? 'bg-info/10' : ''}>
                <tr>
                  <th
                    colSpan={3}
                    className="border border-border px-3 py-1 text-left text-xs font-semibold text-foreground"
                  >
                    {t.killDistancePlayerHeaderFmt(gamertag, measured, totalKills)}
                  </th>
                </tr>
                {(player.weapons ?? []).map((w) => (
                  <tr key={w.weapon_key}>
                    <td className="border border-border px-2 py-1 text-left">
                      {(locale === 'en' ? w.label_en : w.label) || w.weapon_key}
                    </td>
                    <td className="border border-border px-2 py-1 text-right tabular-nums">
                      <span className="inline-flex min-w-5 items-center justify-center rounded-full bg-primary/10 px-1.5 py-0.5 font-semibold text-primary">
                        {w.measured_kills}
                      </span>
                    </td>
                    <td className="border border-border px-2 py-1 text-right tabular-nums">
                      {t.killDistanceAvgFmt(w.avg_distance_m)}
                      {w.measured_kills > 1 && (
                        <span className="ml-1 text-muted-foreground">
                          {t.killDistanceRangeFmt(w.min_distance_m, w.max_distance_m)}
                        </span>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            )
          })}
        </table>
      </div>
      <p className="px-3 pb-2 pt-2 text-[11px] text-muted-foreground">{t.killDistanceReserve}</p>
    </section>
  )
}
