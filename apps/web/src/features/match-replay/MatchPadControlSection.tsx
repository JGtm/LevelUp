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
 * LA FORME EST UN GRAPHE, PLUS UN TABLEAU (2026-09-03, retours utilisateur). UNE ARME, UNE
 * BARRE (2026-09-13) : le nom du socle à gauche avec son total, puis un rail unique valant
 * 100 % des occupations NOMMÉES de ce socle, découpé en un segment par joueur — le camp du
 * joueur de la page d'abord, l'adverse ensuite, séparés par un filet. Le graphe empilait
 * jusqu'ici un bâton PAR CAMP, d'où le constat de l'utilisateur : « pourquoi "Contrôle des
 * armes spéciales" a plusieurs épaisseurs de barres ? [...] Les étiquettes dans les barres sont
 * illisibles, masquées par la hauteur de la barre qui est insuffisante. » La barre fait
 * maintenant 24 px et le compte s'y écrit dès 6 % du rail.
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
 * socle affiché sont annotées à DROITE de sa ligne (« + N sans nom »), jamais versées à un camp.
 * LA NOTE DE PIED QUI LES VENTILAIT PAR CAUSE A ÉTÉ RETIRÉE le 2026-09-13 (« virer le texte
 * "34 prises attribuées sur 65 occupations…" ») : l'annotation de ligne reste le seul endroit
 * où l'écran avoue ce qu'il ne montre pas, et elle est à l'aplomb du socle concerné.
 *
 * COULEURS. Les camps prennent `teamTokenCssVar` — les jetons `team-ally` / `team-enemy` que les
 * réglages d'accessibilité surchargent, et NON la cascade d'identité de `teamColor.ts` (cf.
 * l'en-tête de `match-view/teamSeriesColor.ts`). Les joueurs d'un camp s'en distinguent par un
 * éclaircissement, calculé par `padControlChart`.
 *
 * Aucun calcul ici : tout vient de `padControlLogic` (les mesures) et `padControlChart` (la
 * projection).
 */
import { useCallback, useMemo } from 'react'

import { ChartLegend } from '@/components/charts/ChartLegend'
import { SectionCard } from '@/components/ui/section-card'
import { Tooltip } from '@/components/ui/tooltip'
import { teamTokenCssVar } from '@/features/match-view/teamSeriesColor'
import type { MatchScoreboardRow } from '@/lib/api/types'
import { resolveTeamLabel } from '@/lib/halo/teamLabel'
import { HeaderLabelTooltip } from '@/lib/table/columnMeta'

import { REPLAY_TEXT, type ReplayLocale } from './i18n/i18n'
import type { ReplayText } from './i18n/i18nContract'
import { buildPadControlBars, type PadBarModel, type PadBarRow } from './model/padControlChart'
import { buildPadControl, type PadControl } from './model/padControlLogic'
import { PAD_TIER_ORDER, type PadTier } from './model/weaponTier'
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

/**
 * hasPadControl — LA double porte de la carte, en fonction pure : pas d'artefact de rejeu
 * (`control` null) ou aucune prise attribuée -> pas de carte.
 *
 * Exportée pour que le parent (`match-view/MatchViewTabArsenal`) décide d'afficher ou non
 * le titre de section qui coiffe cette carte — un titre ne se pose jamais au-dessus de
 * rien. Le parent construit son `PadControl` avec le MÊME `buildPadControl` et le même
 * artefact (une seule clé de cache) : le prédicat, lui, ne s'écrit qu'ici.
 */
export function hasPadControl(control: PadControl | null | undefined): control is PadControl {
  return control?.hasData === true
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

  return (
    <SectionCard
      title={t.padControl.title}
      label={t.padControl.title}
      titleAdornment={(label) => (
        <HeaderLabelTooltip text={t.padControl.titleHint} focusable>
          <span>{label}</span>
        </HeaderLabelTooltip>
      )}
    >
      <PadControlBody bars={bars} control={control} allyOf={allyOf} t={t} />
    </SectionCard>
  )
}

/**
 * PadControlBody — le corps de la carte : le graphe et sa légende, gardés par leur contenu.
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
  return (
    <div className="px-3 pb-3 pt-3">
      {bars.rows.length > 0 && (
        <>
          {!control.tiersMeasured && (
            <p className="pb-3 text-3xs text-muted-foreground">{t.padControl.tiersUnmeasuredNote}</p>
          )}
          {control.randomStarts && (
            <p className="pb-3 text-3xs text-muted-foreground">{t.padControl.randomStartsNote}</p>
          )}
          <PadControlBars model={bars} control={control} t={t} />
          <ChartLegend
            className="pt-3"
            items={bars.teams.map((team) => ({
              key: team.side ?? 'sans-equipe',
              label: team.label,
              color: teamTokenCssVar(allyOf(team.side)),
            }))}
          />
        </>
      )}
    </div>
  )
}

/**
 * La grille du graphe : nom d'arme + total | barre | annotation.
 *
 * LES TROIS PISTES SONT EN `minmax(0, …)` DEPUIS LE 2026-09-19 (lot 2, retrait du défilement
 * horizontal) : une piste de largeur fixe refuse de passer sous la taille de son contenu et
 * pousse le rail hors du cadre. Avec le plancher à zéro, le nom d'arme se tronque (son `title`
 * garde le nom entier) et la barre garde sa part de la largeur disponible.
 */
const ROW_GRID = {
  gridTemplateColumns: `minmax(0, ${NAME_WIDTH}px) minmax(0, 1fr) minmax(0, ${NOTE_WIDTH}px)`,
  gap: 12,
}

/**
 * PadControlBars — les lignes d'arme.
 *
 * PLUS D'AXE DE PRISES : chaque rail vaut 100 % des occupations nommées de SON socle, et le
 * dénominateur de la ligne s'écrit à côté du nom de l'arme. Un axe partagé n'aurait plus rien
 * à graduer.
 */
function PadControlBars({
  model,
  control,
  t,
}: {
  model: PadBarModel
  control: PadControl
  t: ReplayText
}) {
  const groupes = groupRowsByTier(model.rows, control)
  return (
    // PLUS DE DÉFILEMENT HORIZONTAL (2026-09-19, lot 2) : le graphe tenait derrière un
    // `overflow-x-auto` et une largeur plancher de 560 px, donc une barre de défilement sous la
    // carte dès qu'il était à l'étroit. Depuis que ce bloc voisine « Usages d'équipement » dans
    // l'onglet « Contrôle », il suit le MÊME modèle : les trois colonnes de la ligne se
    // répartissent la largeur disponible et le nom d'arme se tronque (son `title` garde le nom
    // entier), plutôt que de pousser le rail hors du cadre.
    <div className="min-w-0">
      {groupes.map((groupe) => (
        <section key={groupe.tier} className="mb-1">
          {/* L'INTERTITRE N'APPARAÎT QUE SI LES NIVEAUX SONT ÉTABLIS : sans référence de
              carte, un unique bandeau « Emplacement non identifié » au-dessus de tout le
              bloc ferait lire une absence de mesure comme un résultat de mesure. La note
              au-dessus du graphe le dit déjà, en toutes lettres. */}
          {control.tiersMeasured && (
            <h4 className="mb-2 flex items-baseline gap-2 border-b pb-1 text-3xs uppercase tracking-wide text-muted-foreground">
              <span>{t.padControl.tierLabels[groupe.tier]}</span>
              <span className="tabular-nums normal-case tracking-normal">
                {t.padControl.tierSubtotalFmt(groupe.total)}
              </span>
            </h4>
          )}
          {groupe.rows.map((row) => (
            <PadWeaponRow key={row.weapon} row={row} t={t} />
          ))}
        </section>
      ))}
    </div>
  )
}

/**
 * groupRowsByTier range les lignes par niveau, dans l'ORDRE ÉCRIT `PAD_TIER_ORDER` (base,
 * terrain, puissance, bonus, non classé) et SANS toucher à l'ordre interne : celui-ci reste
 * celui de `padControlLogic` — les socles décisifs d'abord, puis du plus disputé au moins
 * disputé. Un niveau sans ligne n'a pas d'intertitre : une section vide ne dit rien.
 *
 * LE SOUS-TOTAL AFFICHÉ EST CELUI DES LIGNES DU GROUPE, pas `control.tierTotals` : les deux ne
 * coïncident que si aucune arme ne s'est trouvée à deux niveaux dans le match (0,17 % des
 * prises mesurées). C'est le total des lignes affichées qui doit s'additionner sous les yeux du
 * lecteur, sinon les chiffres de l'écran ne se recomposent pas.
 */
function groupRowsByTier(
  rows: readonly PadBarRow[],
  control: PadControl,
): { tier: PadTier; rows: PadBarRow[]; total: number }[] {
  return PAD_TIER_ORDER.map((tier) => {
    const lignes = rows.filter((row) => (control.tierOfWeapon[row.weapon] ?? 'unclassified') === tier)
    return { tier, rows: lignes, total: lignes.reduce((sum, row) => sum + row.total, 0) }
  }).filter((groupe) => groupe.rows.length > 0)
}

/** Hauteur de la barre : assez haute pour que le compte s'y lise en `text-xs` (retour 13/09). */
const BAR_HEIGHT = 24

/** Sous cette part du rail, le chiffre se ferait rogner : l'infobulle garde la valeur. */
const LABEL_MIN_FRACTION = 0.06

/** Une arme : son nom, son total, sa barre unique, et ce qui n'a pas de ramasseur nommé. */
function PadWeaponRow({ row, t }: { row: PadBarRow; t: ReplayText }) {
  return (
    <div className="mb-2 grid items-center" style={ROW_GRID}>
      <div className="flex items-center justify-end gap-2 text-xs">
        <span className="truncate" title={row.label}>
          {row.label}
        </span>
        <span className="tabular-nums text-muted-foreground">{row.total}</span>
      </div>
      <div className="flex overflow-hidden bg-muted" style={{ height: BAR_HEIGHT }}>
        {row.segments.map((seg) => {
          const tip = t.padControl.barTipFmt(seg.name, seg.sideLabel, row.label, seg.count)
          return (
            // La largeur est portée par l'ITEM du flex, en `calc(%)`. Le retrait de 2 px ouvre
            // la saignée entre deux joueurs ; le FILET (bordure gauche) marque, lui, le passage
            // d'un camp à l'autre — une barre unique doit dire où finit un camp.
            // `text-white` : libellé posé SUR l'aplat du camp, quelle que soit la palette
            // réglée — un contraste de texte dans un aplat, pas une couleur sémantique.
            <div
              key={seg.xuid}
              className={`mr-[2px] h-full last:mr-0${seg.startsSide ? ' border-l-2 border-card' : ''}`}
              style={{ width: `calc(${seg.fraction * 100}% - 2px)` }}
            >
              <Tooltip content={tip} className="h-full w-full">
                <div
                  className="flex h-full w-full items-center justify-center overflow-hidden whitespace-nowrap text-xs font-semibold text-white"
                  style={{ backgroundColor: seg.color }}
                  tabIndex={0}
                  role="img"
                  aria-label={tip}
                >
                  {seg.fraction >= LABEL_MIN_FRACTION ? seg.count : ''}
                </div>
              </Tooltip>
            </div>
          )
        })}
      </div>
      <div className="text-3xs text-muted-foreground tabular-nums">
        {row.unnamed > 0 ? t.padControl.unnamedFmt(row.unnamed) : ''}
      </div>
    </div>
  )
}

