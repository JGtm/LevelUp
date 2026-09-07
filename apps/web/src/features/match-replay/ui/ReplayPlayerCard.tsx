/**
 * ReplayPlayerCard — LA TUILE d'un siège : ce qu'un joueur est à l'instant lu, vivant ou mort,
 * bouclier, armes portées, temps avant le retour, et les effets qui l'habillent.
 *
 * EXTRAITE DE `ReplayTeams.tsx` LE 2026-09-06 (plan fiches compactes, item 1.3) avec ses deux
 * compagnons — l'incrustation d'effets (`ZoneFxOverlay`) et ses tables d'éclairs et de croix.
 * Les LECTURES (compteurs, état vital, armes, zones, objectif, composition des effets) sont
 * parties le même jour dans `model/playerCardReadings.ts` : cette tuile ne calcule plus rien,
 * elle rend. La colonne (`ReplayTeams.tsx`) ne garde que les camps, la scène des effets et la
 * grille des sièges.
 *
 * LES COTES VIENNENT DU GABARIT (`model/cardGabarit.ts`) — des nombres et des booléens, jamais
 * un mot de densité : la même tuile rend la fiche normale (235 px, corps de 35) et, sur une
 * Grande équipe, la tuile compacte (115 × 62, corps de 31). La MISE EN PAGE se lit dans une
 * table FERMÉE par la hauteur du corps (`TILE_LAYOUT`, 2026-09-07) : un corps de 35 rend la
 * fiche d'aujourd'hui classe pour classe (fixation `__fixtures__/replayTeams.4v4.html`), un
 * corps de 31 la tuile à trois lignes. La profondeur du DOM autour du nom ne bouge dans aucun
 * des deux : dix tests atteignent la tuile par `getByText(nom).parentElement.parentElement`.
 *
 * TROIS RÈGLES QUI NE SE NÉGOCIENT PAS ICI :
 *   1. Une valeur non lue s'affiche comme une lacune, jamais comme un zéro ni une moyenne.
 *   2. Une lecture ancienne PÂLIT et dit son âge — l'inventaire ne se lit qu'aux images-clés,
 *      une toutes les ~20 s, et le faire passer pour l'instant courant était un défaut réel.
 *   3. Aucun littéral de couleur : les rôles passent par des tokens sémantiques.
 */
import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'
import type { ReplayScoreTimelineReady } from '@/lib/replay/scoreTimeline'

import { ReplayObjectiveMark } from './ReplayObjectiveMark'
import { ReplayCountersBadge } from './ReplayCountersBadge'
import { ReplayInventoryRow } from './ReplayInventoryRow'
import { EliminatedBox, VitalityBar } from './ReplayVitality'
import { ReplayWeaponsRow } from './ReplayWeaponsRow'
import type { CardGabarit } from '../model/cardGabarit'
import type { ZonePresence } from '../model/equipmentZones'
import { handCellHint } from '../model/handCellHint'
import { cardChrome, hasUnderLayer } from '../model/playerCardFx'
import { playerCardReadings, type CardFxScene } from '../model/playerCardReadings'
import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import type { ReplayPlayer, VitalityPresence } from '../../../lib/replay/rosterLogic'

/**
 * LA MISE EN PAGE DE LA TUILE, PAR HAUTEUR DE CORPS — une table FERMÉE de littéraux Tailwind
 * (une valeur arbitraire interpolée ne produit AUCUNE règle, en silence : cf. cardGabarit.ts et
 * `rosterHeight.guard.test.ts`). La clé est `CardGabarit['bodyPx']`, une union de littéraux :
 * un corps qui n'est pas dans la table ne compile pas.
 *
 * 35 — LA FICHE D'AUJOURD'HUI (option 2a du handoff 2026-08-27), classe pour classe : deux
 * lignes, le nom et le triplet côte à côte, puis le corps (jauges 5 / 3 px, rangée de 18 px à
 * cellules fixes). Le nom est en `flex-1` : la ligne est à lui et au triplet.
 *
 * 31 — LA TUILE COMPACTE (maquette A2, plan fiches compactes 2026-09-06) : 115 × 62, marge
 * 6 px, rayon 6 px, TROIS lignes. Ligne 1 (14 px) : le nom SEUL, pleine ligne, `leading-[14px]`
 * — sans lui le preflight Tailwind (1,5) donne 17,25 px et la tuile 65 ; le nom n'est PAS en
 * `flex-1`, pour que l'infobulle de la TUILE (`fx.title`) reste atteignable sur le reste de la
 * ligne (décision D6 : le `title` d'une cellule masque celui de la tuile). Ligne 2 (12 px) : les
 * jauges 4 / 2 px en `flex-1` à gauche (≈ 42 px), le triplet à droite (≈ 54 px). Ligne 3
 * (16 px) : arme en main 48, grenade sélectionnée 14, capacité 16, `gap 5` — 88 px, 13 d'air.
 * Interlignes 3 px ; corps FIXE 31 = 12 + 3 + 16 ; tuile 1 + 6 + 14 + 3 + 31 + 6 + 1 = 62.
 */
interface TileLayout {
  /** La tuile : chrome, marges, rayon. */
  tile: string
  /** La ligne du nom. */
  nameLine: string
  /** Le nom, hors sa taille (`NAME_CLASS`) et hors son encre (vivant / mort). */
  name: string
  /** Le corps, SANS sa hauteur (`BODY_CLASS`) ni son `overflow-hidden` (la garantie, ajoutée). */
  body: string
  /** Le triplet F/M/A vit sur la ligne du nom (vrai) ou sur la ligne des jauges, dans le corps. */
  countersOnNameLine: boolean
  /** La pile des deux jauges. */
  gauges: string
  /** La ligne des jauges quand elle porte AUSSI le triplet (corps de 31) ; '' sinon. */
  vitalsLine: string
  /** La rangée armes + inventaire. */
  row: string
  /**
   * Le RAYON des couches d'effets — couche sous le contenu, incrustation (nuage, éclairs,
   * anneau du capteur, fourreau de translocation) et filigrane de porteur : LE MÊME que celui
   * de la tuile, sans quoi une couche `inset-0` à 8 px déborderait les coins d'une tuile à 6
   * (étape 4.6, 2026-09-07). `rounded-lg` sur la fiche normale (la chaîne d'aujourd'hui, à
   * l'octet), `rounded-md` sur la compacte. Un littéral en clair par gabarit, jamais interpolé.
   */
  layerRadius: string
}

const TILE_LAYOUT: Record<CardGabarit['bodyPx'], TileLayout> = {
  35: {
    tile: 'relative flex shrink-0 flex-col rounded-lg border px-2.5 py-2',
    nameLine: 'relative flex items-baseline gap-1.5',
    name: 'min-w-0 flex-1 truncate',
    body: 'relative mt-[7px]',
    countersOnNameLine: true,
    gauges: 'flex flex-col gap-[3px]',
    vitalsLine: '',
    row: 'mt-[6px] flex h-[18px] flex-nowrap items-center gap-x-[5px] overflow-hidden',
    layerRadius: 'rounded-lg',
  },
  31: {
    tile: 'relative flex shrink-0 flex-col rounded-md border px-1.5 py-1.5',
    nameLine: 'relative flex leading-[14px]',
    name: 'min-w-0 truncate',
    body: 'relative mt-[3px]',
    countersOnNameLine: false,
    gauges: 'flex min-w-0 flex-1 flex-col gap-[2px]',
    vitalsLine: 'flex h-[12px] items-center gap-[5px]',
    row: 'mt-[3px] flex h-[16px] flex-nowrap items-center gap-x-[5px] overflow-hidden',
    layerRadius: 'rounded-md',
  },
}

/** La hauteur FIXE du corps, en classe : le littéral en clair, jamais interpolé. */
const BODY_CLASS: Record<CardGabarit['bodyPx'], string> = { 35: 'h-[35px]', 31: 'h-[31px]' }
/** La taille du nom, en classe — une seule valeur, les deux gabarits la partagent. */
const NAME_CLASS: Record<CardGabarit['namePx'], string> = { 11.5: 'text-[11.5px]' }

interface ReplayPlayerCardProps {
  player: ReplayPlayer
  doc: ReplayDocumentReady
  frame: number
  presence: VitalityPresence
  vitalityFade: number
  readingFull: number
  flashFrames: number
  locale: ReplayLocale
  /** Calque de score du film, déjà passé par la garde d'horloge. */
  scoreTimeline?: ReplayScoreTimelineReady
  fxScene: CardFxScene
  gabarit: CardGabarit
}

export function ReplayPlayerCard({
  player, doc, frame, presence, vitalityFade, readingFull, flashFrames, locale, scoreTimeline, fxScene, gabarit,
}: ReplayPlayerCardProps) {
  const t = REPLAY_TEXT[locale]
  const { live, state, name, equipped, filmIndex, zones, objective, fx } = playerCardReadings({
    player, doc, frame, presence, flashFrames, scoreTimeline, fxScene, text: t,
  })
  const L = TILE_LAYOUT[gabarit.bodyPx]
  // LE TRIPLET EST LE MÊME ÉLÉMENT dans les deux gabarits ; seule sa PLACE change (ligne du
  // nom, ou ligne des jauges — et l'encadré « Éliminé » sur une fiche morte compacte). Il reste
  // visible mort ou vif : la mort ne retire pas les compteurs.
  const counters = <ReplayCountersBadge board={player.board} live={live} locale={locale} gabarit={gabarit} />
  // Le bouclier AU-DESSUS de la santé : l'ordre dans lequel le jeu les encaisse, dit aussi par
  // l'épaisseur (5 / 3 en normal, 4 / 2 en compact). Sans `sh`/`hp` dans le document, aucune
  // barre n'existe (`VitalityBar` rend null sur une lecture nulle).
  const gauges = (
    <div className={L.gauges}>
      <VitalityBar reading={state.shield} fade={vitalityFade} name={t.shieldLabel} token="info" heightPx={gabarit.gaugeShieldPx} />
      <VitalityBar reading={state.health} fade={vitalityFade} name={t.healthLabel} token="success" heightPx={gabarit.gaugeHealthPx} />
    </div>
  )
  // SANS CELLULE DE MUNITIONS (`showAmmo: false`), c'est l'ARME EN MAIN qui dit ses munitions
  // et les marques de la lecture d'inventaire, en infobulle (décision D5) : la fiche compose le
  // texte (`model/handCellHint.ts`, une lecture d'inventaire de plus par fiche compacte — le
  // volet JS du 5.2 le mesure) et le confie à la rangée d'armes. Jamais sur la fiche normale.
  const handHint =
    gabarit.showAmmo || !state.life
      ? undefined
      : (handCellHint(t, doc, state.life.slot, frame, equipped) ?? undefined)
  return (
    // LA TUILE (option 2a du handoff 2026-08-27) : chaque fiche porte sa bordure et son
    // fond — dégradé court autour de `card` en vie, `card` teinté destructive en mort
    // (cf. playerCardFx.cardChrome). `shrink-0` : une tuile ne se tasse jamais, la colonne
    // défile.
    <div className={L.tile} style={cardChrome(state.alive)} title={fx.title}>
      {/* LA COUCHE D'EFFETS ÉPOUSE LA TUILE (option 2a) : fonds, voiles, flou et cadres
          vivent sur cette couche `inset-0`, au rayon de la tuile (`L.layerRadius`), SOUS le
          contenu — les rangées sont en `relative` pour peindre au-dessus d'elle. Les éclats
          de mort/réapparition animent SON fond, jamais celui de la fiche. */}
      {hasUnderLayer(fx) && (
        <div
          aria-hidden
          className={`replay-card-fx pointer-events-none absolute inset-0 ${L.layerRadius} ${fx.flashClass}`}
          style={fx.underStyle}
        />
      )}
      {/* LE FILIGRANE DE PORTEUR : la couche du seul canal qui restait libre sur la fiche —
          derrière le contenu, sans toucher ni la bordure (trois cadres d'équipement) ni le
          fond (verre, voile, teinte de mort). Déclarée AVANT les rangées, qui sont
          `relative` : l'ordre de peinture du DOM les met au-dessus, comme la couche d'effets. */}
      {objective && (
        <ReplayObjectiveMark kind={objective} sizePx={gabarit.watermarkPx} radiusClass={L.layerRadius} />
      )}
      {/* AUCUNE MARQUE D'IDENTITÉ SUR LA FICHE (demande utilisateur du 2026-08-25) : le glyphe
          « ami » a été retiré de la colonne. Il reste au FIL des éliminations, où il sert à
          reconnaître un nom au milieu d'événements qui défilent ; sur une fiche, la colonne
          d'équipe et le nom disent déjà tout ce qu'il y a à savoir. */}
      <div className={L.nameLine}>
        <span
          className={`${L.name} ${NAME_CLASS[gabarit.namePx]} font-bold uppercase tracking-[.06em] ${
            state.alive ? 'text-foreground' : 'text-muted-foreground'
          }`}
          title={name}
        >
          {name}
        </span>
        {L.countersOnNameLine && counters}
      </div>
      {/* HAUTEUR CONSTANTE vivant/mort : le CORPS de la fiche est une zone à hauteur FIXE
          (`bodyPx` : 35 = barres 11 + marge 6 + inventaire 18 ; 31 = jauges 12 + 3 + rangée 16)
          dans les DEUX états. La mort remplace son CONTENU — l'encadré « Éliminé » remplit
          toute la zone — jamais la zone : une fiche qui change de hauteur fait sauter toute la
          colonne à chaque mort (retour utilisateur du 2026-08-24). `overflow-hidden` est la
          garantie, pas un ornement. */}
      <div className={`${L.body} ${BODY_CLASS[gabarit.bodyPx]} overflow-hidden`}>
        {state.alive ? (
          <>
            {L.countersOnNameLine ? (
              gauges
            ) : (
              <div className={L.vitalsLine}>
                {gauges}
                {counters}
              </div>
            )}
            {/* ARMES ET INVENTAIRE SUR UNE GRILLE À CELLULES FIXES (demande utilisateur du
                2026-08-24) : chaque rangée émet des cellules à largeur constante — pour que les
                fiches s'alignent en colonnes. `flex-nowrap` + `overflow-hidden` : la rangée ne
                se replie jamais. */}
            <div className={L.row}>
              <ReplayWeaponsRow
                doc={doc}
                state={state}
                read={equipped}
                frame={frame}
                readingFull={readingFull}
                filmIndex={filmIndex}
                locale={locale}
                gabarit={gabarit}
                handHint={handHint}
              />
              {state.life && (
                <ReplayInventoryRow
                  doc={doc}
                  slot={state.life.slot}
                  equipped={equipped}
                  frame={frame}
                  readingFull={readingFull}
                  locale={locale}
                  gabarit={gabarit}
                />
              )}
            </div>
          </>
        ) : (
          <EliminatedBox
            state={state}
            doc={doc}
            frame={frame}
            locale={locale}
            bodyPx={gabarit.bodyPx}
            counters={L.countersOnNameLine ? undefined : counters}
          />
        )}
      </div>
      <ZoneFxOverlay
        zones={zones}
        translocationDelay={fx.translocationDelay}
        boltCount={gabarit.boltCount}
        radiusClass={L.layerRadius}
      />
    </div>
  )
}

/**
 * Décalages des trois croix du champ de réparation : désynchronisés (délais NÉGATIFS, donc
 * déjà en vol au premier rendu) pour qu'elles ne montent pas au pas. Trois, pas plus : la
 * fiche reste une fiche, l'effet un signe. Abscisses en % pur, glyphe de 10 px : sur une tuile
 * de 115 px elles tombent à 18 / 53 / 85 px, sans chevauchement — les trois restent.
 */
const REPAIR_CROSSES = [
  { left: '16%', delay: '-0.3s' },
  { left: '46%', delay: '-1.1s' },
  { left: '74%', delay: '-1.9s' },
] as const

/**
 * Les TROIS ÉCLAIRS de l'écran occultant (option 2a du handoff 2026-08-27) : abscisse,
 * largeur, et décalage de scintillement dans le cycle de 2,6 s (cf. globals.css,
 * `replay-zone-bolt`). Le décalage se SOUSTRAIT à l'horloge de la pose (`shroudSinceMs`) :
 * à la pose, les délais valent exactement 0 / −1,3 / −2,05 s — la composition de la
 * maquette — et après un saut de lecture chaque éclair reprend à son avancement réel,
 * jamais sur un rythme inventé (même contrat que le capteur et les éclats).
 *
 * LES LARGEURS SONT ABSOLUES (42 / 34 / 28 px) sur des abscisses en % : sur une tuile de
 * 115 px, trois éclairs se chevauchent. Le gabarit dit combien en rendre (`boltCount`) — les
 * deux premiers de la table sur la tuile compacte.
 */
const SHROUD_BOLTS = [
  { left: '10%', width: 42, offsetS: 0 },
  { left: '56%', width: 34, offsetS: 1.3 },
  { left: '33%', width: 28, offsetS: 2.05 },
] as const

/**
 * ZoneFxOverlay — l'INCRUSTATION au-dessus du contenu : le nuage noir et les ÉCLAIRS de
 * l'écran occultant (« par-dessus les infos, mais légèrement »), le contour « détecté » du
 * capteur adverse, les mini croix du champ de réparation, et le FOURREAU de translocation —
 * la lumière qui court sur la bordure, en rotation continue jusqu'à son fondu. (Les fonds,
 * voiles et cadres, eux, vivent sur la couche d'effets SOUS le contenu :
 * playerCardFx.underStyle.)
 *
 * SUR SA PROPRE COUCHE, ET C'EST LE POINT : la couche du dessous anime déjà son fond (une
 * seule animation par élément et par propriété) — la pulsation du capteur, les éclairs et
 * le fourreau vivent donc chacun sur leur enfant. MÊME GÉOMÉTRIE que la couche du dessous
 * (`inset-0` au rayon de la tuile, `radiusClass` — option 2a) : les deux épousent la fiche,
 * jamais deux cadres décalés. `pointer-events-none` : l'infobulle et le survol restent
 * ceux de la fiche. `aria-hidden` : tout ce que l'incrustation montre est déjà dit en
 * texte par l'infobulle (title de la fiche).
 *
 * LE DÉLAI NÉGATIF DU CAPTEUR cale la pulsation sur l'horloge des pings du capteur le plus
 * fraîchement pingé (equipmentZones.sensorSincePingMs) : le contour de la fiche bat AVEC
 * l'onde de la carte, à la cadence officielle — jamais un rythme inventé. Ceux des éclairs
 * suivent l'horloge de la POSE de l'écran (shroudSinceMs) ; celui du fourreau, le contrat
 * des éclats (reprise à l'avancement réel après un saut de lecture).
 */
function ZoneFxOverlay({
  zones,
  translocationDelay,
  boltCount,
  radiusClass,
}: {
  zones: ZonePresence
  translocationDelay: string | null
  /** Nombre d'éclairs à rendre, pris en tête de `SHROUD_BOLTS` (3 en normal, 2 en compact). */
  boltCount: number
  /** Le rayon de la tuile, en classe (`TILE_LAYOUT.layerRadius`) : l'incrustation l'épouse. */
  radiusClass: string
}) {
  const rien =
    !zones.repair && zones.shroudSinceMs === null && zones.sensorSincePingMs === null &&
    translocationDelay === null
  if (rien) return null
  return (
    <div aria-hidden className={`pointer-events-none absolute inset-0 overflow-hidden ${radiusClass}`}>
      {zones.shroudSinceMs !== null && (
        <>
          <div className="replay-zone-cloud absolute inset-0" />
          {SHROUD_BOLTS.slice(0, boltCount).map((b) => (
            <span
              key={b.left}
              className="replay-zone-bolt absolute top-[-8%] h-[116%]"
              style={{
                left: b.left,
                width: b.width,
                animationDelay: `${(-((zones.shroudSinceMs ?? 0) / 1000 + b.offsetS)).toFixed(3)}s`,
              }}
            />
          ))}
        </>
      )}
      {zones.sensorSincePingMs !== null && (
        <div
          className={`replay-zone-sensor absolute inset-0 ${radiusClass} border-[1.5px] border-dashed`}
          style={{
            borderColor: tokenCssVar('destructive'),
            animationDelay: `${(-zones.sensorSincePingMs / 1000).toFixed(3)}s`,
          }}
        />
      )}
      {zones.repair &&
        REPAIR_CROSSES.map((c) => (
          <span
            key={c.left}
            className="replay-zone-cross absolute top-[40%] text-[10px] font-bold leading-none"
            style={{ left: c.left, color: tokenCssVar('success'), animationDelay: c.delay }}
          >
            +
          </span>
        ))}
      {translocationDelay !== null && (
        <div
          className={`replay-flash-translocation absolute inset-0 ${radiusClass}`}
          style={{ animationDelay: translocationDelay }}
        />
      )}
    </div>
  )
}
