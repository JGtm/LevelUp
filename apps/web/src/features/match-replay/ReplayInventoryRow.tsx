/**
 * ReplayInventoryRow — munitions de l'arme en main, grenades portées, capacité d'armure.
 *
 * LA RANGÉE EST UNE GRILLE À CELLULES FIXES depuis le 2026-08-24 (même règle que la rangée
 * d'armes, demande utilisateur) : une cellule de MUNITIONS, une boîte de GRENADES, une
 * cellule de CAPACITÉ — toujours rendues, toujours aux mêmes largeurs. Une donnée absente
 * laisse sa cellule vide plutôt que de décaler les colonnes des fiches voisines.
 *
 * TOUT CE QUI N'EST PAS LU S'AFFICHE COMME LACUNE, jamais comme une valeur par défaut :
 * - une capacité hors table reçoit un GLYPHE NEUTRE (vignette vide en pointillés) et garde
 *   son NUMÉRO en infobulle ;
 * - un emplacement dont le film n'écrit RIEN est PLEIN (flux différentiel : le plein est la
 *   valeur par défaut, jamais transmise) ;
 * - les ARMES À CHARGE (familles plasma / mêlée / énergie : épée, marteau, pistolet à
 *   plasma, Pulse Carbine, Ravager, Sentinel Beam...) n'ont PAS de chargeur : leurs champs
 *   mag/res sont des ancrages parasites du film (mesure 2026-08-24 : 46 cellules « 1/0 »
 *   sur le témoin, toutes sur ces familles). Elles affichent leur charge en POURCENTAGE
 *   écrit (« 87% », demande utilisateur du 2026-08-24 — pas une jauge) : 100% quand rien
 *   n'a été consommé — jamais un « 1/0 ».
 *
 * SEULE L'ARME EN MAIN garde ses munitions (règle de la fiche compacte, devenue LA fiche) :
 * les munitions d'une arme rangée ne se lisent que pour préparer une permutation, ce qui
 * n'est pas ce qu'on regarde dans un rejeu.
 *
 * CHAQUE CELLULE PÂLIT AVEC L'ÂGE DE LA LECTURE QUI LA DÉCRIT, et non avec celui de la rangée :
 * les munitions viennent de l'inventaire (images-clés, ~20 s), la capacité du canal i48, les
 * grenades de l'axe `grenadeReads` (deux canaux, dont les paquets delta). Un estompage unique
 * posé sur le conteneur multipliait l'opacité propre de chaque cellule par celle de l'inventaire
 * et rendait toute cellule plus fraîche PLUS PÂLE que la rangée — l'inverse de ce qu'un
 * estompage dit. C'est la même raison que pour les armes portées, appliquée par lecture.
 *
 * LA CELLULE DE CAPACITÉ VIT DANS `ReplayAbilityCell.tsx` depuis le 2026-09-04 (lot P6) :
 * l'affichage des CHARGES restantes s'y branche (compte lu, « plein » qualitatif, ou rien —
 * cf. `abilityChargeLogic.ts`), et l'extraction paie l'addition — ce fichier était à 4 lignes
 * de son plafond de 500.
 */
import { WeaponIcon } from '@/components/ui/WeaponIcon'
import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'

import type { CatalogLabel } from './i18n/catalogLabel'
import type { EquippedReading } from './equippedLogic'
import { REPLAY_TEXT, type ReplayLocale } from './i18n/i18n'
import {
  grenadeBoxAt,
  grenadeBoxHint,
  grenadesCarriedFrom,
  type InventoryEmptyState,
  inventoryAt,
  inventoryEmptyHint,
  selectedGrenadeFrom,
} from './inventoryReading'
import { ReplayAbilityCell, StateMark } from './ReplayAbilityCell'
import { formatSeconds, frameToMs, freshness, READING_FADE } from './replayLogic'
import type { ReplayDocumentReady } from './replayNormalize'
import { familyOf } from './shotEffects'

/** Vignette d'un TYPE DE GRENADE : 14 px (option 2a — la boîte de 56 px est taillée dessus). */
const GRENADE_ICON_PX = 14
/** Largeurs FIXES des cellules — la grille des fiches en dépend (option 2a : 32 / 56). */
const AMMO_CELL_W = 32
const GRENADES_BOX_W = 56

/** Les familles d'arme À CHARGE : pas de chargeur, une jauge (cf. en-tête). */
const CHARGE_FX = new Set(['plasma', 'melee', 'light'])

export function ReplayInventoryRow({
  doc,
  slot,
  equipped,
  frame,
  readingFull,
  locale,
}: {
  doc: ReplayDocumentReady
  slot: number
  equipped: EquippedReading | null
  frame: number
  readingFull: number
  locale: ReplayLocale
}) {
  const t = REPLAY_TEXT[locale]
  const read = inventoryAt(doc, slot, frame)
  const state = read?.state
  // LES GRENADES ONT LEUR PROPRE LECTURE, et donc leur propre age (schema 20, lot 4.4 du suivi
  // delta). L axe `grenadeReads` porte DEUX canaux sur la meme grandeur : les images-cles
  // (~20 s) et les paquets delta, transmis AU CHANGEMENT — donc la ou l etat bouge. La plus
  // recente gagne, et l age median de ce qui s affiche tombe de 10,00 s a 8,09 s.
  //
  // LE REPLI EST LE POINT : un artefact anterieur au schema 20 ne porte pas cet axe, et la
  // boite retombe alors sur la lecture d inventaire — exactement l affichage d avant. Un
  // artefact ancien ne se vide pas, il est seulement moins frais.
  //
  // LE DEPARTAGE VIT DANS `grenadeBoxAt`, PAS ICI : une lecture A VENIR ne prime jamais une
  // information passee (meme doctrine que la « lecture vide A VENIR »), et le composant ne
  // fait qu afficher ce que la fonction pure a tranche.
  const box = grenadeBoxAt(doc, slot, frame, read)
  const grenades = box ? grenadesCarriedFrom(box.g, doc.grenadeLabels, locale) : []
  const selected = box ? selectedGrenadeFrom(box.g, box.gs) : null
  const ammo = state?.am ?? []
  // L'ÉTAT VIDE DE LA LECTURE COURANTE (schéma 19, lot du 2026-08-25) : présent, il veut dire
  // que la lecture qui couvre l'image ne rend RIEN — `state` porte alors la dernière lecture
  // PLEINE du même slot, et cet état-ci s'affiche À CÔTÉ (cf. `InventoryEmptyMark`).
  //
  // LE `return null` DU LOT « LECTURE VIDE » N'A PLUS LIEU D'ÊTRE : la refonte du 2026-08-24
  // rend la grille à cellules fixes INCONDITIONNELLEMENT (parité de gabarit morte/vivante), donc
  // la ligne ne disparaît plus jamais — ni sur une lecture vide, ni sur une lecture pleine sans
  // contenu. La garantie du lot est tenue a fortiori.
  const empty = read?.empty

  // Âge NÉGATIF = lecture d'une image-clé À VENIR (début de vie, cf. inventoryAt) :
  // l'infobulle le dit — l'estompage porte déjà sur la valeur absolue.
  const ageMs = read ? formatSeconds(frameToMs(Math.abs(read.age), doc)) : ''
  const ageTitle = !read
    ? undefined
    : read.age < 0
      ? `${t.inventoryAhead} ${ageMs}`
      : `${t.inventoryAge} ${ageMs}`

  // La cellule de munitions décrit l'ARME EN MAIN : l'identifiant de famille dit si c'est
  // une arme à charge (jauge) ou à chargeur (compte).
  const drawnId = equipped && equipped.drawn !== null ? equipped.weapons[0]?.id : undefined

  return (
    <div
      className="flex items-center gap-[5px] font-mono text-[9.5px] text-muted-foreground"
      // LA RANGÉE N'IMPOSE PLUS SON ÂGE À TOUTE LA GRILLE. L'estompage vivait ici tant que
      // l'inventaire était la seule lecture de la ligne ; il ne l'est plus (capacité par i48,
      // grenades par l'axe `grenadeReads`). Laissé au conteneur, il MULTIPLIAIT l'opacité
      // propre de chaque cellule par celle de l'inventaire : une lecture de grenades de 8,1 s
      // s'affichait plus PÂLE qu'avant le lot, et le gain de fraîcheur devenait invisible.
      // Chaque cellule porte donc l'âge de la lecture qui la décrit, et rien d'autre.
      title={ageTitle}
    >
      <span
        className="inline-flex shrink-0 items-center"
        style={{ width: AMMO_CELL_W, opacity: read ? freshness(read.age, readingFull, READING_FADE) : 1 }}
      >
        {equipped && ammo.length > 0 && (
          equipped.drawn !== null ? (
            <AmmoCell
              ammo={ammo[equipped.drawn] ?? {}}
              charge={CHARGE_FX.has(familyOf(drawnId ? doc.weaponLabels?.[drawnId]?.fx : undefined))}
              fullLabel={t.ammoFullLabel}
              drawnHint={t.ammoDrawnHint}
              gaugeLabel={t.gaugeLabel}
            />
          ) : !equipped.holstered && !empty ? (
            // Sélecteur non lu : la cellule dit la lacune — armes rangées (D=2), elle,
            // n'affiche RIEN : aucune arme en main, aucune munition à décrire.
            //
            // SAUF SUR UNE LECTURE VIDE : `equippedWeapons` y rend volontairement `drawnUnread`
            // (aucune arme dégainée pour un joueur que l'artefact déclare mort), et le badge
            // d'état en fin de rangée dit POURQUOI. Écrire « dégainée ? » à côté de « Mort »
            // poserait une lacune là où la cause est connue.
            <span className="border-b border-dashed border-border opacity-80">
              {t.drawnUnknown}
            </span>
          ) : null
        )}
      </span>
      <span
        className="inline-flex shrink-0 items-center gap-1 overflow-hidden"
        // ÂGE PROPRE AUX GRENADES, comme pour la capacité et pour la même raison : l'axe
        // `grenadeReads` ne tombe pas sur les images-clés de l'inventaire, l'estompage de
        // l'inventaire ne le décrit donc pas. Un âge négatif est une lecture À VENIR : la
        // valeur absolue estompe, l'infobulle dit « dans X s ».
        style={{
          width: GRENADES_BOX_W,
          opacity: box ? freshness(box.age, readingFull, READING_FADE) : 1,
        }}
        title={box ? grenadeBoxHint(t, box, grenades, doc) : undefined}
      >
        {grenades.map((g) => (
          <GrenadeChip
            key={g.rank}
            carried={g}
            icon={grenadeMaskOf(doc.grenadeLabels?.[g.rank])}
            selected={
              typeof selected === 'object' && selected !== null && g.rank === selected.rank
                ? selected
                : null
            }
            t={t}
          />
        ))}
        {selected === 'indeterminate' && (
          <span className="border-b border-dashed border-border opacity-80">
            {t.grenadeSelUnknown}
          </span>
        )}
      </span>
      {/* LA CELLULE DE CAPACITÉ vit dans son propre fichier depuis le lot P6 (charges) :
          vignette à largeur fixe, puis compte de charges ou « plein » dans l'espace souple —
          elle garde son âge propre, sa lecture ne tombant pas sur les images-clés. */}
      <ReplayAbilityCell
        doc={doc}
        slot={slot}
        frame={frame}
        readingFull={readingFull}
        locale={locale}
      />
      {/* L'ÉTAT VIDE VIENT APRÈS LES CELLULES FIXES, et c'est ce qui concilie les deux règles.
          Le lot « lecture vide » le voulait à côté de l'équipement ; la refonte veut que les
          colonnes des fiches restent alignées. Un badge inséré AVANT une cellule fixe décalerait
          toute la grille des fiches voisines dès qu'un joueur meurt — placé en DERNIER, il occupe
          la place libre de la rangée sans toucher à une seule largeur. Il est souple et tronqué
          (« Inventaire indisponible » est long) : l'infobulle porte le texte entier. */}
      {empty && read && (
        <InventoryEmptyMark
          empty={empty}
          label={empty.kind === 'dead' ? t.inventoryDeadLabel : t.inventoryEmptyLabel}
          hint={inventoryEmptyHint(t, read, empty, doc)}
          // L'ÂGE DU BADGE EST CELUI DE LA LECTURE VIDE, pas celui de l'équipement affiché à
          // côté (`read.age`) : c'est la même distinction que porte déjà son infobulle.
          fade={freshness(empty.age, readingFull, READING_FADE)}
        />
      )}
    </div>
  )
}

/** Ce que la puce a besoin de savoir pour peindre la vignette : l'URL, et son mode. */
interface GrenadeIconRef {
  url: string
  /** Vrai = masque à teindre (currentColor) ; faux = image finie. */
  tinted: boolean
}

/**
 * grenadeMaskOf — la vignette d'un rang : le MASQUE DE HUD cuit dans l'artefact, la
 * version PLEINE de la maquette (option 2a du handoff 2026-08-27 — elle remplace l'image
 * versionnée de la planche du 16/08, dont le module `grenadeIcon.ts` est parti avec elle).
 * Teint par currentColor, le masque suit le thème ET l'encre `warning` du type équipé.
 * Sans vignette : null — l'appelant garde le libellé, jamais l'image d'un autre type.
 */
function grenadeMaskOf(label: CatalogLabel | undefined): GrenadeIconRef | null {
  return label?.img ? { url: label.img, tinted: !!label.tinted } : null
}

/**
 * GrenadeChip — UN type de grenade porté : sa vignette, son compteur, et la marque du type
 * ÉQUIPÉ quand c'est lui qui partira au prochain lancer.
 *
 * LA VIGNETTE PORTE L'IDENTITÉ, LE NOM PASSE EN INFOBULLE (planche du 16/08 : « pas de
 * texte sauf pour le compteur »). Sans AUCUNE image, le libellé revient : mieux vaut un mot
 * qu'un trou.
 */
function GrenadeChip({
  carried,
  icon,
  selected,
  t,
}: {
  carried: { rank: number; name: string; count: number }
  icon: GrenadeIconRef | null
  /** Non nul = c'est CE type qui est équipé ; `read` dit si la lecture ou la déduction l'établit. */
  selected: { rank: number; read: boolean } | null
  t: (typeof REPLAY_TEXT)[ReplayLocale]
}) {
  return (
    <span
      className={`inline-flex shrink-0 items-center gap-0.5 ${selected ? 'rounded-sm px-0.5 font-semibold' : ''}`}
      style={
        selected
          ? {
              color: tokenCssVar('warning'),
              boxShadow: `0 0 0 1px ${tokenCssVar('warning')}`,
              background: `color-mix(in srgb, ${tokenCssVar('warning')} 13%, transparent)`,
            }
          : undefined
      }
      title={
        selected
          ? `${carried.name} — ${selected.read ? t.grenadeSelectedRead : t.grenadeSelected}`
          : carried.name
      }
    >
      {icon ? (
        <WeaponIcon
          imageUrl={icon.url}
          tinted={icon.tinted}
          label={carried.name}
          width={GRENADE_ICON_PX}
          height={GRENADE_ICON_PX}
        />
      ) : (
        carried.name
      )}
      <span className="tabular-nums">×{carried.count}</span>
    </span>
  )
}

/**
 * InventoryEmptyMark — la lecture qui couvre cette image ne rend RIEN, et la ligne le DIT au
 * lieu de disparaître.
 *
 * LES DEUX ÉTATS NE SE CONFONDENT PAS, et c'est tout l'intérêt du marqueur publié par
 * l'artefact : « Mort » est CORROBORÉ par le fil des éliminations (88,3 % des lectures vides
 * mesurées, contre 1,1 % des lectures pleines soumises à la même fenêtre) ; « Inventaire
 * indisponible » dit qu'on ne sait pas. Écrire « Mort » sur le second serait affirmer à l'écran
 * ce qu'aucune pièce n'établit.
 *
 * L'ÉTAT MORT PORTE UN MOT, PAS SEULEMENT UN DESSIN — contrairement aux pictogrammes muets de
 * `StateMark` (chargeur plein). Ceux-là décorent une lecture PRÉSENTE ; celui-ci explique
 * pourquoi l'équipement affiché a vingt secondes d'âge, ce qu'un dessin seul ne dit pas.
 * L'encre vient du token sémantique, jamais d'une valeur.
 *
 * IL EST SOUPLE ET TRONQUÉ, seul de la rangée : les autres cellules ont une largeur FIXE parce
 * que les fiches s'alignent en colonnes, ce badge occupe la place qui reste et ne décale donc
 * rien. « Inventaire indisponible » y tient rarement en entier — l'infobulle, elle, est complète.
 */
function InventoryEmptyMark({
  empty,
  label,
  hint,
  fade,
}: {
  empty: InventoryEmptyState
  label: string
  hint: string
  /** Estompage de l'âge de la LECTURE VIDE — la rangée ne l'impose plus (cf. le conteneur). */
  fade: number
}) {
  return (
    <span
      className={
        empty.kind === 'dead'
          ? 'min-w-0 truncate rounded-sm px-0.5 font-semibold uppercase'
          : 'min-w-0 truncate border-b border-dashed border-border'
      }
      style={
        empty.kind === 'dead'
          ? {
              opacity: fade,
              color: tokenCssVar('destructive'),
              boxShadow: `0 0 0 1px ${tokenCssVar('destructive')}`,
            }
          : { opacity: fade * 0.8 }
      }
      title={hint}
    >
      {label}
    </span>
  )
}

/** AmmoFullMark — munitions pleines : un chargeur rempli. */
function AmmoFullMark({ label }: { label: string }) {
  return (
    <StateMark label={label}>
      <svg width="10" height="10" viewBox="0 0 12 12" aria-hidden="true">
        <rect x="4" y="1.5" width="4" height="9" rx="1" fill="currentColor" />
      </svg>
    </StateMark>
  )
}

/**
 * AmmoCell — les munitions de l'ARME EN MAIN.
 *
 * DEUX NATURES, DEUX RENDUS. Une arme À CHARGE (`charge`) montre son POURCENTAGE écrit
 * (« 87% », demande utilisateur du 2026-08-24) : la donnée du film compte le CONSOMMÉ, le
 * nombre dit le RESTANT (complément) ; rien de consommé = « 100% » — et ses champs
 * mag/res, quand le film en émet, sont des ancrages parasites qu'on n'affiche JAMAIS
 * (cf. l'en-tête du fichier). Une arme à CHARGEUR montre son compte `mag/res` ; un
 * emplacement jamais écrit est PLEIN (pictogramme).
 *
 * PAS DE COMPTEUR « n / N » sur une charge : le quantum est propre à l'arme et le film ne
 * dit pas combien de charges font un plein.
 */
function AmmoCell({
  ammo,
  charge,
  fullLabel,
  drawnHint,
  gaugeLabel,
}: {
  ammo: { mag?: number; res?: number; gauge?: number }
  charge: boolean
  fullLabel: string
  drawnHint: string
  gaugeLabel: string
}) {
  const chargePct = (consumed: number) => (
    <span title={gaugeLabel}>{Math.round(Math.max(0, 1 - consumed) * 100)}%</span>
  )
  return (
    <span className="inline-flex items-center gap-1 tabular-nums text-foreground" title={drawnHint}>
      {charge ? (
        chargePct(ammo.gauge ?? 0)
      ) : ammo.mag !== undefined ? (
        <span>
          {ammo.mag}
          {ammo.res !== undefined && <span className="opacity-60">/{ammo.res}</span>}
        </span>
      ) : ammo.gauge !== undefined ? (
        chargePct(ammo.gauge)
      ) : (
        <AmmoFullMark label={fullLabel} />
      )}
    </span>
  )
}
