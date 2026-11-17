/**
 * ReplayRotatedButton — LE BOUTON « PLAN TOURNÉ » de la barre d'une ligne (Cinéma et Plein cadre).
 *
 * Il n'existe QUE quand l'Auto a tourné le plan (`autoRotation` de l'objet de commandes) : jamais pour un
 * choix retenu, jamais quand le plan est droit. Son clic retient 0 quart de tour pour la carte — le plan
 * revient droit et y reste. La barre complète du Standard ne le porte pas : le retour y passe par la section
 * d'orientation du panneau Vue.
 *
 * LA FORME est celle de la maquette (capsule d'une ligne, icône de rotation, libellé) ; ses cotes sont celles
 * de `COTES_BARRE_COMPACTE`. Bordure et texte prennent la couleur de suivi (`--ring`), la bordure à moitié.
 */
import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'
import { COTES_BARRE_COMPACTE as C } from '../model/cotesBarreCompacte'

/** La flèche qui reboucle : le geste de rotation. Décorative (le bouton porte le nom). */
function RotationIcon() {
  return (
    <svg
      width={C.planTourneIconePx}
      height={C.planTourneIconePx}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      aria-hidden="true"
    >
      <path d="M21 12a9 9 0 1 1-3-6.7L21 8" />
      <path d="M21 3v5h-5" />
    </svg>
  )
}

export function ReplayRotatedButton({
  onKeepUpright, locale,
}: {
  /** Retient 0 quart de tour pour la carte : le plan revient droit. */
  onKeepUpright: () => void
  locale: ReplayLocale
}) {
  const t = REPLAY_TEXT[locale].frames
  return (
    <button
      type="button"
      onClick={onKeepUpright}
      title={t.rotatedPlanTip}
      className="inline-flex shrink-0 cursor-pointer items-center whitespace-nowrap text-[12.5px] transition-colors hover:bg-accent focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
      style={{
        height: C.planTourneHauteurPx,
        padding: `0 ${C.planTourneMargeXPx}px`,
        columnGap: C.planTourneEcartPx,
        borderRadius: C.planTourneRayonPx,
        border: `${C.planTourneBordurePx}px solid color-mix(in srgb, var(--ring) ${C.planTourneBordurePct}%, transparent)`,
        color: 'var(--ring)',
      }}
    >
      <RotationIcon />
      {t.rotatedPlan}
    </button>
  )
}
