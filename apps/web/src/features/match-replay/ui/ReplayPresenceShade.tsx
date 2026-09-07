/**
 * ReplayPresenceShade — LE TEMPS QU'UN JOUEUR N'A PAS JOUÉ, ombré sur sa piste (2026-09-07, L4).
 *
 * # CE QUE L'OMBRE DIT, ET QUE LE VIDE NE DISAIT PAS
 *
 * Une piste sans marque a deux causes opposées : le joueur n'était pas là, ou il n'a rien fait.
 * Jusqu'ici les deux se dessinaient pareil — une bande grise. L'ombre les sépare, et c'est elle
 * qui explique une dominance qui s'effondre à la huitième minute : l'effectif a baissé.
 *
 * # LE BORD DIT LA CONFIANCE, ET C'EST LA RÈGLE LA PLUS IMPORTANTE DE CE FICHIER
 *
 * L'API date une arrivée à la seconde : le bord est FRANC. Le repli film la DÉDUIT d'une borne de
 * vie à 10-20 s près, et ne distingue pas un départ d'une élimination définitive : le bord se
 * DÉGRADE vers le transparent sur une douzaine de pixels, et l'infobulle porte la réserve. Une
 * frontière au pixel sur une déduction serait un mensonge de précision — le dessin doit avouer ce
 * que la donnée ne sait pas. (`source` vient de `presenceFeed.ts` et voyage jusqu'ici intact.)
 * Les MOTS de cette réserve, eux, viennent de `model/presenceWording.ts` — le même foyer que la
 * ligne du fil, pour que les deux ne se contredisent jamais sur un même événement.
 *
 * # LE GLYPHE EST LE BOUTON, ET IL VIT DU CÔTÉ OMBRÉ
 *
 * Décisions 2 et 2 bis du plan « frise, point de vue » : un seul objet, pas un repère plus une
 * cible. La porte se pose sur la FRONTIÈRE entre l'ombre et le jeu, à l'intérieur de l'ombre —
 * bord droit de l'ombre de tête pour un arrivant, bord gauche de l'ombre de queue pour un
 * partant. Ce côté-là n'est pas un détail d'esthétique : c'est ce qui garantit que le glyphe ne
 * recouvre JAMAIS une marque de kill, puisqu'un kill ne peut exister que dans la zone jouée.
 *
 * Le clic pose le curseur à l'image de l'événement. IL NE MET PAS EN PAUSE et ne change pas le
 * point de vue (décision 1 : le curseur appartient à l'utilisateur, la sélection ne le déplace
 * jamais — l'inverse vaut aussi, ce bouton-ci ne déplace que lui).
 *
 * # LA PISTE DES COÉQUIPIERS N'A PAS DE GLYPHE
 *
 * Elle a des PALIERS (`ReplayAbsenceShade`) : l'opacité y dit la part de l'effectif absente. À
 * quatre coéquipiers, quatre portes empilées sur quatorze pixels ne se liraient pas, et la
 * question de cette piste n'est pas « qui est parti » mais « combien manquaient ». Le fil, lui,
 * nomme chacun.
 *
 * # COULEURS
 *
 * L'ombre est une couleur STRUCTURELLE (`--muted-foreground`, l'encre neutre du dépôt), exception
 * assumée du skill `color-tokens` et de même nature que le fond de piste `bg-muted/40` : les deux
 * encres sémantiques disponibles ici désignent un CAMP, et une absence n'en désigne aucun. Le
 * GLYPHE, lui, est sémantique — camp du joueur quand il est connu, encre courante sinon.
 */
import { Fragment } from 'react'

import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'

import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'
import type { AbsenceStep, PresenceShade } from '../model/presenceTrackLogic'
import { trackLeft, trackWidth } from '../model/replayTimelineTracksLogic'
import { presenceWording } from '../model/presenceWording'
import { PresenceGlyph } from './PresenceGlyph'

/**
 * L'ENCRE DE L'OMBRE et sa force. `--muted-foreground` est le gris neutre du thème : il
 * s'assombrit sur fond clair et s'éclaircit sur fond sombre, donc l'ombre reste VISIBLE dans les
 * deux thèmes là où un `--background` ou un `--foreground` en aurait disparu dans l'un des deux.
 * L'alpha maximal est celui de la piste du joueur regardé (un absent sur un) ; les paliers de la
 * piste des coéquipiers le multiplient par la part manquante.
 */
const SHADE_INK = 'var(--muted-foreground)'
const SHADE_ALPHA = 0.5

/** La longueur sur laquelle un bord DÉDUIT s'efface. Une douzaine de pixels : assez pour qu'on
 *  voie que la frontière n'est pas nette, trop peu pour qu'on la croie ailleurs. */
const FILM_FADE = '12px'

/** Le glyphe de la frise est plus petit que celui du fil : une piste fait dix-huit pixels. */
const TRACK_GLYPH_W = 11
const TRACK_GLYPH_H = 10

/**
 * ReplayPresenceShade — les ombres de la piste du joueur REGARDÉ, et leurs portes cliquables.
 *
 * SANS HORLOGE ÉTABLIE, `shades` EST VIDE ET C'EST VOULU : `presenceEntries` ne produit aucune
 * ligne de présence quand la fenêtre de gameplay ou l'horloge du film manquent (même porte que
 * les lignes du fil). L'ombrage est alors ABSENT plutôt que FAUX — une ombre posée sur un axe non
 * recalé se tromperait de 3,6 à 50,8 s. Rien à corriger ici : la dégradation a lieu en amont.
 */
export function ReplayPresenceShade({
  shades,
  identity,
  onSeekFrame,
  locale,
}: {
  shades: readonly PresenceShade[]
  /** Camp de chaque xuid, relatif au point de vue : il teinte le glyphe, jamais l'ombre. */
  identity: ReadonlyMap<string, { ally: boolean }>
  /** Poser le curseur à cette image. Ne met pas en pause (cf. l'en-tête). */
  onSeekFrame: (frame: number) => void
  locale: ReplayLocale
}) {
  const t = REPLAY_TEXT[locale]
  return (
    <>
      {shades.map((s) => {
        const { label, hint } = presenceWording(s, t)
        const entering = s.kind === 'joined'
        return (
          <Fragment key={s.key}>
            <span
              aria-hidden="true"
              className={`pointer-events-none absolute inset-y-0 ${entering ? 'rounded-l-full' : 'rounded-r-full'}`}
              style={{
                left: trackLeft(s.from),
                width: trackWidth(s.from, s.to),
                background: shadeBackground(s.source, entering),
                opacity: SHADE_ALPHA,
              }}
            />
            <button
              type="button"
              onClick={() => onSeekFrame(s.frame)}
              // LE GLYPHE SE POSE DU CÔTÉ OMBRÉ DE SA FRONTIÈRE (cf. l'en-tête) : l'arrivant
              // recule de sa propre largeur pour tenir dans l'ombre de tête, le partant part du
              // bord et déborde vers l'ombre de queue.
              className={`absolute top-0 flex h-full items-center rounded-[2px] text-muted-foreground transition-colors hover:bg-accent focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring ${entering ? '-translate-x-full' : ''}`}
              style={{ left: trackLeft(s.edge) }}
              aria-label={`${s.clock} — ${label}`}
              title={hint}
            >
              <PresenceGlyph
                kind={s.kind}
                color={glyphInk(identity.get(s.xuid)?.ally)}
                width={TRACK_GLYPH_W}
                height={TRACK_GLYPH_H}
              />
            </button>
          </Fragment>
        )
      })}
    </>
  )
}

/**
 * ReplayAbsenceShade — la piste des COÉQUIPIERS, ombrée par PALIERS.
 *
 * L'opacité d'un palier est la part de l'effectif manquante (un sur quatre = un quart de
 * l'alpha) : la piste s'assombrit à mesure que l'équipe se vide, et se rallume quand un
 * remplaçant arrive. Aucun glyphe (cf. l'en-tête), aucun bord dégradé — un palier agrégé mêle
 * déjà des sources différentes, et prétendre y dater une frontière au pixel n'aurait pas de sens.
 */
export function ReplayAbsenceShade({ steps }: { steps: readonly AbsenceStep[] }) {
  return (
    <>
      {steps.map((s) => (
        <span
          key={s.key}
          aria-hidden="true"
          className="pointer-events-none absolute inset-y-0"
          style={{
            left: trackLeft(s.from),
            width: trackWidth(s.from, s.to),
            background: SHADE_INK,
            opacity: (SHADE_ALPHA * s.absent) / s.total,
          }}
        />
      ))}
    </>
  )
}

/**
 * LE FOND D'UNE OMBRE : plat quand la source AFFIRME, dégradé vers la zone jouée quand elle
 * DÉDUIT. Le dégradé part toujours de la frontière — `to left` pour une ombre de tête (dont la
 * frontière est à droite), `to right` pour une ombre de queue.
 */
function shadeBackground(source: PresenceShade['source'], entering: boolean): string {
  if (source === 'api') return SHADE_INK
  const sens = entering ? 'to left' : 'to right'
  return `linear-gradient(${sens}, transparent, ${SHADE_INK} ${FILM_FADE})`
}

/**
 * L'ENCRE DU GLYPHE — les deux encres d'équipe du rejeu, et l'encre COURANTE quand le camp n'est
 * pas connu (bot sans ligne de tableau de score, joueur hors scoreboard). Jamais l'une des deux
 * par défaut : ce serait désigner un camp au hasard, sur le seul repère de la frise qui nomme
 * quelqu'un.
 */
function glyphInk(ally: boolean | undefined): string {
  if (ally === undefined) return 'currentColor'
  return tokenCssVar(ally ? 'team-ally' : 'team-enemy')
}
