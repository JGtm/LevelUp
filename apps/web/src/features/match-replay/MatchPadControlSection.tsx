/**
 * MatchPadControlSection — LE CONTRÔLE DES ARMES SPÉCIALES D'UN MATCH, une colonne par socle.
 *
 * CE QUE LA PAGE NE SAVAIT PAS DIRE, ET CE QUE LE BILAN D'ÉQUIPEMENT REFUSAIT DE DIRE. La
 * section voisine (`MatchEquipmentUsageSection`) compte les socles VIDÉS, au niveau du match et
 * sans ramasseur : c'était la seule chose vraie tant que `padPickups[].xuid` valait `null`
 * partout. L'événement natif de ramassage l'a levée (SCHÉMA 30) : ce bloc-ci nomme le
 * ramasseur socle par socle, et c'est la stat de domination tactique demandée — qui a tenu le
 * fusil de précision, qui a raflé l'épée.
 *
 * LA FORME EST UN GRAPHE, PLUS UN TABLEAU (2026-09-03, retours utilisateur). UNE COLONNE PAR
 * SOCLE (2026-09-21, décision D18, proposition 4.A de la maquette) : la hauteur d'une colonne
 * est le nombre de prises NOMMÉES de ce socle, sur une ÉCHELLE COMMUNE à tous les socles, et
 * la colonne est empilée par joueur — couleur du camp, éclaircissement par joueur. Le rail
 * 100 % qui précédait donnait à tous les socles la MÊME longueur : un socle pris 3 fois
 * occupait autant de largeur qu'un socle pris 12, et le total, écrit en petit à gauche, était
 * la seule chose qui détrompait. Les groupes de niveau ne sont plus des bandeaux : un TRAIT
 * VERTICAL discret sépare la puissance du terrain, et le nom de chaque groupe avec son
 * sous-total s'écrit DANS le graphe, centré en haut de sa moitié.
 *
 * DÉPLIÉ, ET IL N'Y A PLUS DE REPLI (2026-09-13). La décision D3 du 2026-09-05 (replié par
 * défaut, seuls les socles « game changers » en avant) est RÉVOQUÉE par l'utilisateur :
 * « pourquoi il est constamment replié et n'affiche jamais rien par défaut ? ». Toutes les
 * armes s'affichent au chargement ; l'ordre du vote survit en ORDRE d'affichage (les socles
 * décisifs en tête), il ne cache plus rien.
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
 * socle affiché sont annotées SOUS sa colonne (« + N sans nom »), jamais versées à un camp.
 * LA NOTE DE PIED QUI LES VENTILAIT PAR CAUSE A ÉTÉ RETIRÉE le 2026-09-13 (« virer le texte
 * "34 prises attribuées sur 65 occupations…" ») : l'annotation de colonne reste le seul endroit
 * où l'écran avoue ce qu'il ne montre pas, et elle est à l'aplomb du socle concerné.
 *
 * COULEURS. Les camps prennent les jetons `team-ally` / `team-enemy` que les réglages
 * d'accessibilité surchargent, et NON la cascade d'identité de `teamColor.ts` (cf. l'en-tête de
 * `match-view/teamSeriesColor.ts`). Les joueurs d'un camp s'en distinguent par un
 * éclaircissement calculé par `padControlChart` — rendu en `color-mix` dans la légende DOM, en
 * OPACITÉ dans le canvas, qui n'accepte ni variable CSS ni `color-mix`.
 *
 * Aucun calcul ici : tout vient de `padControlLogic` (les mesures), `padControlChart` (la
 * projection) et `padControlColumns` (le pivot en colonnes).
 */
import { useCallback, useMemo, useState } from 'react'

import { BarStackedChart } from '@/components/charts/BarStackedChart'
import { ChartLegend } from '@/components/charts/ChartLegend'
import { getEChartsThemeColors } from '@/components/charts/_utils'
import { titleWithInfo } from '@/components/ui/title-with-info'
import { SectionCard } from '@/components/ui/section-card'
import { teamSeriesColor, teamTokenCssVar } from '@/features/match-view/teamSeriesColor'
import { useColorPaletteVersion } from '@/lib/accessibility/useColorPaletteVersion'
import type { MatchScoreboardRow } from '@/lib/api/types'
import { useThemeVersion } from '@/lib/echarts/useThemeVersion'
import { resolveTeamLabel } from '@/lib/halo/teamLabel'

import { REPLAY_TEXT, type ReplayLocale } from './i18n/i18n'
import type { ReplayText } from './i18n/i18nContract'
import { buildPadControlBars, type PadBarModel, type PadBarRow } from './model/padControlChart'
import { buildPadColumns, type PadColumnGroupInput } from './model/padControlColumns'
import { buildPadControl, hasPadControl, type PadControl } from './model/padControlLogic'
import { PAD_TIER_ORDER, type PadTier } from './model/weaponTier'
import { useMatchReplay } from '../../lib/replay/queries'
import { padNameFor } from './layers/useReplayWeaponPads'

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
  const control = useMemo(
    () => (data ? buildPadControl(data, board) : null),
    [data, board],
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
      control && data
        ? buildPadControlBars({
            control,
            weaponLabel: (weapon) => padNameFor(weapon, data.weaponLabels, t, locale),
            teamLabel,
            teamColor: (side) => teamTokenCssVar(allyOf(side)),
            // Le camp du joueur de la page ouvre la barre : c'est sa page. Le camp inconnu
            // ferme la marche — on ne le glisse pas entre les deux camps nommés.
            teamRank: (side) => (allyOf(side) === true ? 0 : side == null ? 2 : 1),
          })
        : null,
    [control, data, t, locale, teamLabel, allyOf],
  )

  // Double porte : pas d'artefact, ou aucune prise attribuée -> rien du tout. MÊME prédicat
  // que celui lu par le parent pour poser (ou non) son titre de section (`bars` ne vaut null
  // que dans les mêmes cas — il se construit du même `control`).
  if (!hasPadControl(control) || !bars) return null

  // Ce que le bloc ne montre PLUS : les prises d'un socle qu'aucun emplacement de carte ne
  // confirme. Compté sur les LIGNES affichées, comme les sous-totaux de niveau.
  // NIVEAUX NON ÉTABLIS = TOUT EST « non classé » : le compte n'aurait alors aucun sens (il
  // vaudrait le match entier) et `tiersUnmeasuredNote` le dit déjà. Il ne se calcule donc que
  // lorsque la carte est dans la référence.
  const unclassified = control.tiersMeasured
    ? bars.rows
        .filter((row) => (control.tierOfWeapon[row.weapon] ?? 'unclassified') === 'unclassified')
        .reduce((sum, row) => sum + row.total, 0)
    : 0

  return (
    <SectionCard
      title={t.padControl.title}
      label={t.padControl.title}
      // UNE SEULE INFOBULLE (i) SUR LE TITRE, 2026-09-21 (lot D). Elle a remplacé le
      // `HeaderLabelTooltip` invisible du libellé ET les deux notes qui s'écrivaient
      // au-dessus du graphe (niveaux non établis, départs aléatoires) : trois réserves sur
      // la même carte, à trois endroits, dont une seule se voyait.
      titleAdornment={titleWithInfo(
        <div className="space-y-2">
          <p>{t.padControl.titleHint}</p>
          {!control.tiersMeasured && <p>{t.padControl.tiersUnmeasuredNote}</p>}
          {control.randomStarts && <p>{t.padControl.randomStartsNote}</p>}
          {/* LE GROUPE « NON IDENTIFIÉ » NE SE REND PLUS (D2) : son compte se dit ICI,
              et nulle part ailleurs. */}
          {unclassified > 0 && <p>{t.padControl.unclassifiedHintFmt(unclassified)}</p>}
        </div>,
      )}
    >
      <PadControlBody bars={bars} control={control} allyOf={allyOf} t={t} />
    </SectionCard>
  )
}

/**
 * PadControlBody — le corps de la carte : le graphe des socles décisifs, sa légende, et le
 * dépliable des armes de base.
 *
 * Extrait du composant le 2026-09-05 (plafond de taille de fonction du dépôt).
 */
function PadControlBody({
  bars,
  control,
  allyOf,
  t,
}: {
  bars: PadBarModel
  control: PadControl
  allyOf: (side: string | null) => boolean | null
  t: ReplayText
}) {
  const groupes = groupRowsByTier(bars.rows, control)
  const base = groupes.find((groupe) => groupe.tier === 'base')
  // LES ARMES DE BASE FERMENT LA MARCHE, DANS UN DÉPLIABLE FERMÉ (D2) : reprendre son fusil
  // d'assaut n'est pas contrôler la carte, et leurs colonnes écraseraient l'échelle commune
  // des socles décisifs — une base prise trente fois contre un sniper pris quatre.
  const decisifs = groupes.filter((groupe) => groupe.tier !== 'base')
  return (
    <div className="px-3 pb-3 pt-3">
      <PadColumnsChart
        // LE TITRE DE GROUPE NE S'ÉCRIT QUE SI LES NIVEAUX SONT ÉTABLIS : sans référence de
        // carte, tout est « non classé » et un titre ferait lire une absence de mesure comme
        // un résultat de mesure. L'infobulle du titre le dit déjà.
        groups={decisifs.map((groupe) => ({
          label: control.tiersMeasured
            ? `${t.padControl.tierShortLabels[groupe.tier]} · ${t.padControl.tierSubtotalFmt(groupe.total)}`
            : '',
          rows: groupe.rows,
        }))}
        allyOf={allyOf}
        t={t}
        emptyMessage={t.padControl.chartEmpty}
        legendLabel={t.padControl.title}
      />
      {base && <PadTierFold groupe={base} allyOf={allyOf} t={t} />}
    </div>
  )
}

/** Hauteur du graphe : assez haute pour que les comptes tiennent dans les segments. */
const CHART_HEIGHT = 300

/**
 * PadColumnsChart — UNE COLONNE PAR SOCLE, hauteur = ses prises nommées, échelle commune.
 *
 * LES ENCRES DU CANVAS SE RÉSOLVENT ICI, et pas dans `padControlChart` : ECharts peint un
 * bitmap et n'accepte ni `var(--token)` ni `color-mix()`. Le camp donne la couleur
 * (`teamSeriesColor`, donc les jetons `team-ally` / `team-enemy` de la palette réglée), le rang
 * du joueur dans son camp donne l'OPACITÉ — l'équivalent canvas de l'éclaircissement
 * `color-mix` que la légende DOM, elle, garde tel quel. Les deux se re-résolvent au changement
 * de thème comme de palette.
 */
function PadColumnsChart({
  groups,
  allyOf,
  t,
  emptyMessage,
  legendLabel,
}: {
  groups: PadColumnGroupInput[]
  allyOf: (side: string | null) => boolean | null
  t: ReplayText
  emptyMessage: string
  legendLabel: string
}) {
  const themeVersion = useThemeVersion()
  const paletteVersion = useColorPaletteVersion()
  const model = useMemo(
    () => buildPadColumns({ groups, unnamedFmt: t.padControl.unnamedFmt }),
    [groups, t],
  )
  const encres = useMemo(() => {
    void themeVersion
    void paletteVersion
    const tc = getEChartsThemeColors()
    const couleurs: Record<string, string> = {}
    const opacites: Record<string, number> = {}
    for (const joueur of model.players) {
      couleurs[joueur.key] = teamSeriesColor(allyOf(joueur.side), tc)
      opacites[joueur.key] = joueur.tint / 100
    }
    return { couleurs, opacites }
  }, [model, allyOf, themeVersion, paletteVersion])
  const note = useCallback((categorie: string) => model.notes[categorie], [model])

  return (
    <>
      <BarStackedChart
        frameless
        height={CHART_HEIGHT}
        emptyMessage={emptyMessage}
        series={
          model.datapoints.length > 0
            ? [{ key: 'pad-control', datapoints: model.datapoints }]
            : []
        }
        componentOrder={model.componentOrder}
        componentHexColors={encres.couleurs}
        componentOpacity={encres.opacites}
        categoryGroups={model.groups}
        valueLabels={{ segments: true, totals: model.totals }}
        categoryNote={note}
        // Un joueur n'a pris qu'une poignée de socles : sans ce filtre, l'infobulle d'une
        // colonne listerait les huit joueurs du lobby dont six à zéro.
        tooltipHideZero
        // UNE SEULE LÉGENDE, celle du DOM (ci-dessous) : elle porte l'éclaircissement exact
        // des segments, que la légende ECharts ne saurait pas reproduire.
        showLegend={false}
      />
      <ChartLegend
        className="pt-2"
        ariaLabel={legendLabel}
        items={model.players.map((joueur) => ({
          key: joueur.key,
          label: joueur.key,
          color: joueur.cssColor,
        }))}
      />
    </>
  )
}

/**
 * groupRowsByTier range les lignes par niveau, dans l'ORDRE ÉCRIT `PAD_TIER_ORDER`
 * (puissance, terrain, base) et SANS toucher à l'ordre interne : celui-ci reste celui de
 * `padControlLogic` — les socles décisifs d'abord, puis du plus disputé au moins disputé. Un
 * niveau sans ligne n'existe pas : un groupe vide ne se sépare de rien.
 *
 * LE SOUS-TOTAL AFFICHÉ EST CELUI DES LIGNES DU GROUPE, pas `control.tierTotals` : les deux ne
 * coïncident que si aucune arme ne s'est trouvée à deux niveaux dans le match (0,17 % des
 * prises mesurées). C'est le total des lignes affichées qui doit s'additionner sous les yeux du
 * lecteur, sinon les chiffres de l'écran ne se recomposent pas.
 *
 * DEUX NIVEAUX NE SONT PLUS RENDUS (2026-09-21, décision utilisateur). « NON CLASSÉ » : une
 * ligne « emplacement non identifié » faisait lire une absence de mesure comme un niveau de
 * jeu — son compte passe dans l'infobulle (i) du titre, il ne disparaît pas. « BONUS » : un
 * socle de camouflage ou de surbouclier est un ÉQUIPEMENT, pas une arme ; il est DÉJÀ compté
 * par « Usages d'équipement » (`equipmentUsageLogic.ts`, `EPISODE_FAMILIES` ligne 71, épisodes
 * agrégés ligne 339, versés au côté « utilisé » de la colonne du power-up — cf.
 * `equipmentUsageColumns.ts` lignes 124-128). Rien n'est perdu, la mesure change de carte.
 */
function groupRowsByTier(
  rows: readonly PadBarRow[],
  control: PadControl,
): { tier: PadTier; rows: PadBarRow[]; total: number }[] {
  // « NON CLASSÉ » SURVIT AU SEUL CAS OÙ IL N'EST PAS UN VERDICT : carte hors référence, où
  // AUCUNE ligne n'a de niveau. Le retirer là viderait le bloc entier d'un match pourtant
  // mesuré. Les titres de groupe ne s'écrivent alors pas (cf. `PadControlBody`), donc le
  // lecteur ne voit pas non plus le mot « non identifié ».
  return PAD_TIER_ORDER.filter(
    (tier) => tier !== 'powerup' && (tier !== 'unclassified' || !control.tiersMeasured),
  )
    .map((tier) => {
      const lignes = rows.filter(
        (row) => (control.tierOfWeapon[row.weapon] ?? 'unclassified') === tier,
      )
      return { tier, rows: lignes, total: lignes.reduce((sum, row) => sum + row.total, 0) }
    })
    .filter((groupe) => groupe.rows.length > 0)
}

/**
 * PadTierFold — le niveau « base », derrière un bouton fermé par défaut.
 *
 * Le COMPTE EST DANS LE LIBELLÉ (`baseToggleFmt`) : un dépliable qui ne dit pas ce qu'il cache
 * ne s'ouvre jamais. Il s'affiche même quand les niveaux ne sont pas établis — dans ce cas
 * aucune ligne n'est classée « base » et le groupe n'existe pas, la question ne se pose pas.
 *
 * OUVERT, IL REND LE MÊME GRAPHE (2026-09-21, D18) : colonnes empilées par socle, une échelle
 * qui n'est QUE la sienne — les armes de base se comparent entre elles, pas aux socles de
 * puissance, sinon l'échelle commune de ces derniers serait écrasée.
 */
function PadTierFold({
  groupe,
  allyOf,
  t,
}: {
  groupe: { tier: PadTier; rows: PadBarRow[]; total: number }
  allyOf: (side: string | null) => boolean | null
  t: ReplayText
}) {
  const [open, setOpen] = useState(false)
  const groups = useMemo(() => [{ label: '', rows: groupe.rows }], [groupe.rows])
  return (
    <section className="mt-2">
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        aria-expanded={open}
        className="mb-2 flex w-full items-baseline gap-2 border-b pb-1 text-3xs uppercase tracking-wide text-muted-foreground hover:text-foreground"
      >
        <span aria-hidden="true">{open ? '▾' : '▸'}</span>
        <span>{t.padControl.baseToggleFmt(groupe.rows.length)}</span>
        <span className="tabular-nums normal-case tracking-normal">
          {t.padControl.tierSubtotalFmt(groupe.total)}
        </span>
      </button>
      {open && (
        <PadColumnsChart
          groups={groups}
          allyOf={allyOf}
          t={t}
          emptyMessage={t.padControl.chartEmpty}
          legendLabel={t.padControl.tierLabels.base}
        />
      )}
    </section>
  )
}
