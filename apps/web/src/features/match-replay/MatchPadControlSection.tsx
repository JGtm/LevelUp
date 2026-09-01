/**
 * MatchPadControlSection — LE CONTRÔLE DES ARMES SPÉCIALES D'UN MATCH, par joueur et par équipe.
 *
 * CE QUE LA PAGE NE SAVAIT PAS DIRE, ET CE QUE LE BILAN D'ÉQUIPEMENT REFUSAIT DE DIRE. La
 * section voisine (`MatchEquipmentUsageSection`) compte les socles VIDÉS, au niveau du match et
 * sans ramasseur : c'était la seule chose vraie tant que `padPickups[].xuid` valait `null`
 * partout. L'événement natif de ramassage l'a levée (SCHÉMA 30) : ce tableau-ci nomme le
 * ramasseur socle par socle, et c'est la stat de domination tactique demandée — qui a tenu le
 * fusil de précision, qui a raflé l'épée.
 *
 * ELLE VIT DANS `match-replay/` ET NON DANS `match-view/`, pour la raison qui vaut déjà pour sa
 * voisine : le VOCABULAIRE. Chaque nom d'arme qu'elle écrit vient du catalogue du document ou de
 * la table des familles de socle (`padNameFor`), tous deux au dictionnaire du rejeu. Le sens de
 * l'import est établi de longue date : `MatchScoreCurveChart` lit `match-replay/queries` depuis
 * `match-view`.
 *
 * DOUBLE PORTE, comme la courbe de score et le bilan d'équipement : pas d'artefact (le cas de la
 * quasi-totalité des matchs) OU aucune prise attribuée -> RIEN. Pas de cadre vide, pas de
 * « bientôt disponible ». Un match sans socle, un match dont aucune occupation n'a pu être datée,
 * un artefact d'avant le schéma 30 : dans les trois cas la section ne s'affiche pas. La MÊME clé
 * de cache que ses voisines (`useMatchReplay`, gaté par `header.replay_available`) : les blocs de
 * l'onglet partagent un seul téléchargement.
 *
 * CE QUE L'ÉCRAN DIT DE SA PROPRE MESURE, et il doit le dire : le tableau ne montre QUE les
 * occupations dont l'événement natif nomme le ramasseur. Toutes les autres sont réelles, et la
 * note de bas de tableau les compte par CAUSE — ambiguës, non couvertes, datées sans nom, socles
 * de bonus hors jointure. Sans elle, le lecteur croirait tenir la totalité des socles du match.
 *
 * Aucun calcul ici : tout vient de `padControlLogic`. Gabarit structurel :
 * `MatchEquipmentUsageSection.tsx` — mêmes classes de tableau, mêmes tokens.
 */
import { useMemo } from 'react'

import type { MatchScoreboardRow } from '@/lib/api/types'
import { resolveTeamLabel } from '@/lib/halo/teamLabel'
import { HeaderLabelTooltip } from '@/lib/table/columnMeta'

import { REPLAY_TEXT, type ReplayLocale } from './i18n'
import type { ReplayText } from './i18nContract'
import {
  buildPadControl,
  padControlGaps,
  padControlMissing,
  type PadControl,
  type PadControlTeam,
} from './padControlLogic'
import { useMatchReplay } from './queries'
import { padNameFor } from './useReplayWeaponPads'

interface Props {
  playerSlug: string
  matchId: string
  /** `header.replay_available` — le même gate que le lien rejeu et que la courbe de score. */
  replayAvailable: boolean
  scoreboard: MatchScoreboardRow[] | null | undefined
  locale: ReplayLocale
}

export function MatchPadControlSection({
  playerSlug,
  matchId,
  replayAvailable,
  scoreboard,
  locale,
}: Props) {
  const t = REPLAY_TEXT[locale]
  const { data } = useMatchReplay(playerSlug, matchId, replayAvailable)
  const board = useMemo(() => scoreboard ?? [], [scoreboard])
  const control = useMemo(() => (data ? buildPadControl(data, board) : null), [data, board])
  // Le NOM d'un socle passe par la cascade unique du rejeu (famille d'équipement, puis catalogue
  // du document, puis identifiant brut) : aucune seconde table de noms n'est écrite ici.
  const weaponNames = useMemo(
    () =>
      data && control
        ? control.weapons.map((weapon) => padNameFor(weapon, data.weaponLabels, t, locale))
        : [],
    [data, control, t, locale],
  )
  const meXUID = useMemo(() => board.find((r) => r.is_me)?.xuid ?? null, [board])

  // Double porte : pas d'artefact, ou aucune prise attribuée -> rien du tout.
  if (!control?.hasData) return null

  return (
    <section className="rounded-lg border-2 border-border" aria-label={t.padControl.title}>
      <h3 className="px-3 py-2 text-sm font-bold uppercase tracking-wider text-foreground">
        <HeaderLabelTooltip text={t.padControl.titleHint} focusable>
          <span>{t.padControl.title}</span>
        </HeaderLabelTooltip>
      </h3>
      <div className="overflow-x-auto">
        <table className="w-full border-collapse text-3xs">
          <thead>
            <tr className="text-muted-foreground">
              <th className="border border-border border-b-2 px-2 py-1 text-left">
                {t.padControl.colPlayer}
              </th>
              <th className="border border-border border-b-2 px-2 py-1 text-right font-semibold">
                {t.padControl.colTotal}
              </th>
              {weaponNames.map((name, i) => (
                <th
                  key={control.weapons[i]}
                  className="border border-border border-b-2 px-2 py-1 text-right"
                >
                  {name}
                </th>
              ))}
            </tr>
          </thead>
          {control.byTeam.map((team) => (
            <PadControlTeamBody
              key={team.side ?? 'sans-equipe'}
              team={team}
              board={board}
              weapons={control.weapons}
              meXUID={meXUID}
              t={t}
            />
          ))}
        </table>
      </div>
      <PadControlFootnotes control={control} t={t} />
    </section>
  )
}

/**
 * PadControlTeamBody — un camp : son en-tête, ses joueurs (du plus preneur au moins preneur), et
 * son total.
 *
 * LE CAMP SANS NOM EST UN CAMP QUAND MÊME (`side` null) : ce sont les joueurs que le film a vus
 * vivre et que le scoreboard ignore. Ils gardent leur ligne sous « Sans équipe » — le trou se
 * montre, il ne se comble pas, et surtout on ne les verse pas dans un camp au hasard.
 */
function PadControlTeamBody({
  team,
  board,
  weapons,
  meXUID,
  t,
}: {
  team: PadControlTeam
  board: MatchScoreboardRow[]
  weapons: string[]
  meXUID: string | null
  t: ReplayText
}) {
  const rows = board.filter((r) => (r.team_side ?? '') === (team.side ?? ''))
  const label = resolveTeamLabel(team.side ? rows : [], team.side, t)
  return (
    <tbody>
      <tr>
        <th
          colSpan={weapons.length + 2}
          className="border border-border px-3 py-1 text-left text-xs font-semibold text-foreground"
        >
          {label}
        </th>
      </tr>
      {team.players.map((p) => (
        // Clé composite : un xuid peut manquer (entrée de film sans identité résolue) et
        // plusieurs lignes se percuteraient sur une clé vide.
        <tr key={`${p.xuid}||${p.name}`} className={p.xuid === meXUID ? 'bg-info/10' : ''}>
          <td className="border border-border px-2 py-1 text-left">{p.name}</td>
          <td className="border border-border px-2 py-1 text-right font-semibold tabular-nums">
            {p.total}
          </td>
          {weapons.map((weapon) => (
            <td key={weapon} className="border border-border px-2 py-1 text-right tabular-nums">
              {p.byWeapon[weapon] ?? 0}
            </td>
          ))}
        </tr>
      ))}
      <tr className="font-semibold text-foreground">
        <td className="border border-border px-2 py-1 text-left">{t.padControl.teamTotal}</td>
        <td className="border border-border px-2 py-1 text-right tabular-nums">
          {team.total.total}
        </td>
        {weapons.map((weapon) => (
          <td key={weapon} className="border border-border px-2 py-1 text-right tabular-nums">
            {team.total.byWeapon[weapon] ?? 0}
          </td>
        ))}
      </tr>
    </tbody>
  )
}

/**
 * PadControlFootnotes — le dénominateur, et TOUT ce que le tableau ne montre pas.
 *
 * LA SOMME DOIT SE VÉRIFIER À L'ŒIL : prises attribuées + occupations hors tableau = occupations
 * mesurées, ventilées par cause.
 *
 * Le document sans bloc de datation n'existe pas en production (cf. `PadControl.coverage`) : la
 * note est alors simplement omise, sans phrase dédiée — le tableau, lui, reste rendu.
 */
function PadControlFootnotes({ control, t }: { control: PadControl; t: ReplayText }) {
  const p = t.padControl
  const missing = padControlMissing(control)
  const gaps = padControlGaps(control)
  if (!control.coverage) return null
  return (
    <div className="space-y-1 px-3 pb-2 pt-2 text-[11px] text-muted-foreground">
      <p>{p.attributedFmt(control.attributed, control.coverage.occupations)}</p>
      {missing > 0 && gaps.length > 0 && (
        <p>
          {p.missingFmt(missing)}
          {` ${gaps.map((g) => p.gapFmt[g.key](g.count)).join(' · ')}.`}
        </p>
      )}
    </div>
  )
}
