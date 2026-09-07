/**
 * MatchPadControlSection — LE CONTRÔLE DES ARMES SPÉCIALES D'UN MATCH, une arme par ligne.
 *
 * CE QUE LA PAGE NE SAVAIT PAS DIRE, ET CE QUE LE BILAN D'ÉQUIPEMENT REFUSAIT DE DIRE. La
 * section voisine (`MatchEquipmentUsageSection`) compte les socles VIDÉS, au niveau du match et
 * sans ramasseur : c'était la seule chose vraie tant que `padPickups[].xuid` valait `null`
 * partout. L'événement natif de ramassage l'a levée (SCHÉMA 30) : ce bloc-ci nomme le
 * ramasseur socle par socle, et c'est la stat de domination tactique demandée — qui a tenu le
 * fusil de précision, qui a raflé l'épée.
 *
 * LA FORME EST UN GRAPHE, PLUS UN TABLEAU (2026-09-03, retours utilisateur). Une ligne par
 * arme : son nom à gauche, deux bâtons superposés — le camp du joueur de la page au-dessus,
 * l'adverse en dessous — et dans chaque bâton un segment par joueur du camp. L'ÉCHELLE EST
 * COMMUNE À TOUTES LES LIGNES (toutes comptent la même chose : des prises), avec un axe entier
 * en pied. La question « qui a tenu le lance-roquettes » se lit alors en une ligne au lieu d'un
 * balayage de colonne.
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
 * CE QUE L'ÉCRAN DIT DE SA PROPRE MESURE, et il doit le dire : le graphe ne montre QUE les
 * occupations dont l'événement natif nomme le ramasseur. Les autres sont réelles — celles d'un
 * socle affiché sont annotées à DROITE de sa ligne (« + N sans nom »), jamais versées à un camp,
 * et la note de pied les compte toutes par CAUSE : ambiguës, non couvertes, datées sans nom,
 * socles de bonus hors jointure. Sans elle, le lecteur croirait tenir la totalité des socles.
 *
 * COULEURS. Les camps prennent `teamTokenCssVar` — les jetons `team-ally` / `team-enemy` que les
 * réglages d'accessibilité surchargent, et NON la cascade d'identité de `teamColor.ts` (cf.
 * l'en-tête de `match-view/teamSeriesColor.ts`). Les joueurs d'un camp s'en distinguent par un
 * éclaircissement, calculé par `padControlChart`.
 *
 * Aucun calcul ici : tout vient de `padControlLogic` (les mesures) et `padControlChart` (la
 * projection).
 */
import { useCallback, useMemo, useState } from 'react'

import { ChartLegend } from '@/components/charts/ChartLegend'
import { CollapsedItemsToggle } from '@/components/ui/collapsed-items-toggle'
import { SectionCard } from '@/components/ui/section-card'
import { Tooltip } from '@/components/ui/tooltip'
import { teamTokenCssVar } from '@/features/match-view/teamSeriesColor'
import type { MatchScoreboardRow } from '@/lib/api/types'
import { resolveTeamLabel } from '@/lib/halo/teamLabel'
import { HeaderLabelTooltip } from '@/lib/table/columnMeta'

import { REPLAY_TEXT, type ReplayLocale } from './i18n/i18n'
import type { ReplayText } from './i18n/i18nContract'
import { buildPadControlBars, type PadBarModel, type PadBarRow } from './model/padControlChart'
import {
  buildPadControl,
  padControlGaps,
  padControlMissing,
  type PadControl,
} from './model/padControlLogic'
import { useMatchReplay } from '../../lib/replay/queries'
import { padNameFor } from './layers/useReplayWeaponPads'

/** Largeur de la colonne des noms d'arme, et de l'annotation de droite (px). */
const NAME_WIDTH = 148
const NOTE_WIDTH = 78

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
  // REPLIÉ PAR DÉFAUT (plan 2026-09-05, décision D3) : état posé AU MONTAGE, jamais persisté.
  // Le repli ne touche QUE les lignes du graphe : totaux, dénominateurs et ventilation des
  // manques viennent de `control`, calculé sur TOUTES les armes — le total ne ment pas.
  const [expanded, setExpanded] = useState(false)
  const visibleControl = useMemo(
    () =>
      control && !expanded && control.collapsedWeapons.length > 0
        ? { ...control, weapons: control.forwardWeapons }
        : control,
    [control, expanded],
  )
  const meSide = useMemo(() => board.find((r) => r.is_me)?.team_side ?? null, [board])

  const teamLabel = useCallback(
    (side: string | null) =>
      resolveTeamLabel(side ? board.filter((r) => (r.team_side ?? '') === side) : [], side, t),
    [board, t],
  )
  // « Allié » = du côté du joueur de la page ; camp inconnu -> encre neutre (cf. teamSeriesColor).
  const allyOf = useCallback(
    (side: string | null) => (side == null || meSide == null ? null : side === meSide),
    [meSide],
  )
  const bars = useMemo(
    () =>
      visibleControl && data
        ? buildPadControlBars({
            control: visibleControl,
            weaponLabel: (weapon) => padNameFor(weapon, data.weaponLabels, t, locale),
            teamLabel,
            teamColor: (side) => teamTokenCssVar(allyOf(side)),
            // Le camp du joueur de la page passe EN HAUT du bâton : c'est sa page. Le camp
            // inconnu ferme la marche — on ne le glisse pas entre les deux camps nommés.
            teamRank: (side) => (allyOf(side) === true ? 0 : side == null ? 2 : 1),
          })
        : null,
    [visibleControl, data, t, locale, teamLabel, allyOf],
  )

  // Double porte : pas d'artefact, ou aucune prise attribuée -> rien du tout.
  if (!control?.hasData || !bars) return null

  return (
    <SectionCard
      title={t.padControl.title}
      label={t.padControl.title}
      titleAdornment={(label) => (
        // Le bouton du repli vit dans l'EN-TÊTE de la carte, à côté du titre et de son
        // infobulle (plan 2026-09-05, G2.2). Zéro socle replié = pas de bouton.
        <span className="flex items-center justify-between gap-2">
          <HeaderLabelTooltip text={t.padControl.titleHint} focusable>
            <span>{label}</span>
          </HeaderLabelTooltip>
          <CollapsedItemsToggle
            expanded={expanded}
            count={control.collapsedWeapons.length}
            onToggle={() => setExpanded((v) => !v)}
            showLabelFmt={t.collapsedColumnsShowFmt}
            hideLabel={t.collapsedColumnsHide}
            hint={t.collapsedColumnsHint}
          />
        </span>
      )}
      footer={<PadControlFootnotes control={control} t={t} />}
    >
      <PadControlBody bars={bars} allyOf={allyOf} t={t} />
    </SectionCard>
  )
}

/**
 * PadControlBody — le corps de la carte : le graphe et sa légende, gardés par leur contenu.
 *
 * Extrait du composant le 2026-09-05 (plafond de taille de fonction du dépôt). Replié avec
 * ZÉRO socle élu (toutes les prises sont hors vote) : rien à dessiner — le bouton de l'en-tête
 * reste la porte, et la note de pied garde tous les comptes.
 */
function PadControlBody({
  bars,
  allyOf,
  t,
}: {
  bars: PadBarModel
  allyOf: (side: string | null) => boolean | null
  t: ReplayText
}) {
  return (
    <div className="px-3 pb-3 pt-3">
      {bars.rows.length > 0 && (
        <>
          <PadControlBars model={bars} t={t} />
          <ChartLegend
            className="pt-3"
            items={(bars.rows[0]?.sticks ?? []).map((stick) => ({
              key: stick.side ?? 'sans-equipe',
              label: stick.label,
              color: teamTokenCssVar(allyOf(stick.side)),
            }))}
          />
        </>
      )}
    </div>
  )
}

/** La grille du graphe : nom d'arme | bâtons | annotation. Reprise à l'identique par l'axe. */
const ROW_GRID = { gridTemplateColumns: `${NAME_WIDTH}px 1fr ${NOTE_WIDTH}px`, gap: 12 }

/**
 * PadControlBars — les lignes d'arme, puis l'axe commun.
 *
 * L'AXE EST HORS DES LIGNES et posé sur la MÊME grille : c'est ce qui garantit que sa
 * graduation « 3 » tombe à l'aplomb du bout d'un bâton de trois prises.
 */
function PadControlBars({ model, t }: { model: PadBarModel; t: ReplayText }) {
  return (
    <div className="overflow-x-auto">
      <div className="min-w-[560px]">
        {model.rows.map((row) => (
          <PadWeaponRow key={row.weapon} row={row} t={t} />
        ))}
        <div className="grid" style={ROW_GRID}>
          <div />
          <div className="relative h-4 border-t border-border text-3xs text-muted-foreground tabular-nums">
            {model.ticks.map((tick) => (
              <span
                key={tick}
                className="absolute top-0.5 -translate-x-1/2"
                style={{ left: `${(tick / model.bound) * 100}%` }}
              >
                {tick}
              </span>
            ))}
          </div>
          <div />
        </div>
        <div className="grid" style={ROW_GRID}>
          <div />
          <div className="text-center text-3xs text-muted-foreground">{t.padControl.axisPickups}</div>
          <div />
        </div>
      </div>
    </div>
  )
}

/** Une arme : son nom, ses deux bâtons superposés, et ce qui n'a pas de ramasseur nommé. */
function PadWeaponRow({ row, t }: { row: PadBarRow; t: ReplayText }) {
  return (
    <div className="mb-2.5 grid items-center" style={ROW_GRID}>
      <div className="truncate text-right text-xs" title={row.label}>
        {row.label}
      </div>
      <div className="flex flex-col gap-[3px]">
        {row.sticks.map((stick) => (
          <div
            key={stick.side ?? 'sans-equipe'}
            className="flex h-[13px] overflow-hidden bg-muted"
          >
            {stick.segments.map((seg) => {
              const tip = t.padControl.barTipFmt(seg.name, stick.label, row.label, seg.count)
              return (
                // Le segment porte son nombre quand il est assez large : sous ~11 % du rail le
                // chiffre se fait rogner et vaut moins que la place qu'il prend. Le retrait de
                // 2 px ouvre la saignée qui sépare deux joueurs du même camp.
                // `text-white` : libellé posé SUR l'aplat du camp, quelle que soit la palette
                // réglée — un contraste de texte dans un aplat, pas une couleur sémantique.
                <div
                  key={seg.xuid}
                  className="mr-[2px] h-full"
                  style={{ width: `calc(${seg.fraction * 100}% - 2px)` }}
                >
                  <Tooltip content={tip} className="h-full w-full">
                    <div
                      className="flex h-full w-full items-center justify-center overflow-hidden whitespace-nowrap text-3xs font-semibold text-white"
                      style={{ backgroundColor: seg.color }}
                      tabIndex={0}
                      role="img"
                      aria-label={tip}
                    >
                      {seg.fraction * 100 > 11 ? seg.count : ''}
                    </div>
                  </Tooltip>
                </div>
              )
            })}
          </div>
        ))}
      </div>
      <div className="text-3xs text-muted-foreground tabular-nums">
        {row.unnamed > 0 ? t.padControl.unnamedFmt(row.unnamed) : ''}
      </div>
    </div>
  )
}

/**
 * PadControlFootnotes — le dénominateur, et TOUT ce que le graphe ne montre pas.
 *
 * LA SOMME DOIT SE VÉRIFIER À L'ŒIL : prises attribuées + occupations hors graphe = occupations
 * mesurées, ventilées par cause.
 *
 * Le document sans bloc de datation n'existe pas en production (cf. `PadControl.coverage`) : la
 * note est alors simplement omise, sans phrase dédiée — le graphe, lui, reste rendu.
 */
function PadControlFootnotes({ control, t }: { control: PadControl; t: ReplayText }) {
  const p = t.padControl
  const missing = padControlMissing(control)
  const gaps = padControlGaps(control)
  if (!control.coverage) return null
  return (
    <div className="space-y-1 border-t border-border px-3 pb-2 pt-2 text-[11px] text-muted-foreground">
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
