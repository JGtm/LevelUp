import { useEffect, useMemo, useRef, useState } from 'react'
import { useAppShellStore } from '@/stores/appShellStore'
import { DEFAULT_TITLE_SLUG } from '@/lib/staticAssets'

/**
 * Images headers par titre (slug → liste de chemins dans /public/titles/{slug}/).
 * Pour ajouter un nouveau jeu, créer public/titles/{slug}/ et y lister ses images.
 */
const HEADER_IMAGES_BY_TITLE: Record<string, string[]> = {
  halo_infinite: [
    '/titles/halo_infinite/echoes-within-header.webp',
    '/titles/halo_infinite/Infinite.png',
    '/titles/halo_infinite/LoneWolves.png',
    '/titles/halo_infinite/HINF-S2_Fracture_Entrenched.png',
    '/titles/halo_infinite/Halo-infinite-dlc-reach.png',
    '/titles/halo_infinite/halo-infinite-dlc-combined-arms.png',
    '/titles/halo_infinite/infinite_fractures_tenrai.png',
  ],
  halo_5: ['/titles/halo_5/wallpaper_halo_5_guardians_01.png'],
}

const ROTATION_INTERVAL_MS = 45_000
const FADE_DURATION_MS = 900

function pickOther(images: string[], current: string): string {
  const others = images.filter((img) => img !== current)
  return others[Math.floor(Math.random() * others.length)]
}

/**
 * HomeHeroBanner — bandeau visuel décoratif de l'accueil.
 * Effectue une rotation aléatoire entre les images du titre courant toutes les 45 secondes.
 * Utilise deux couches superposées pour un vrai cross-fade simultané (pas de noir entre images).
 */
export function HomeHeroBanner() {
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  const images = useMemo(
    () => HEADER_IMAGES_BY_TITLE[titleSlug] ?? HEADER_IMAGES_BY_TITLE[DEFAULT_TITLE_SLUG] ?? [],
    [titleSlug],
  )

  // Deux couches (A=0, B=1) qui alternent. La couche inactive reçoit la prochaine
  // image pendant qu'elle est à opacity:0, puis les deux transitionnent simultanément.
  const [srcs, setSrcs] = useState<[string, string]>(() => [
    images[Math.floor(Math.random() * images.length)] ?? '',
    '',
  ])
  const [activeIdx, setActiveIdx] = useState(0)

  // Refs pour lire l'état courant dans setInterval sans stale closure
  const srcsRef = useRef(srcs)
  srcsRef.current = srcs
  const activeIdxRef = useRef(activeIdx)
  activeIdxRef.current = activeIdx

  useEffect(() => {
    if (images.length === 0) return
    const initial = images[Math.floor(Math.random() * images.length)] ?? ''
    setSrcs([initial, ''])
    setActiveIdx(0)

    // Une seule image : on l'affiche fixe, pas de rotation (pickOther renverrait
    // undefined → couche vide → bannière noire toutes les 45s).
    if (images.length < 2) return

    const timer = setInterval(() => {
      const currentActive = activeIdxRef.current
      const currentSrc = srcsRef.current[currentActive]
      const nextSrc = pickOther(images, currentSrc)
      const nextIdx = currentActive === 0 ? 1 : 0

      // 1. Charger la nouvelle image dans la couche inactive (encore à opacity:0)
      setSrcs((prev) => {
        const next = [...prev] as [string, string]
        next[nextIdx] = nextSrc
        return next
      })

      // 2. Double rAF : laisser le navigateur peindre le background-image sur la couche
      //    inactive avant de déclencher le cross-fade (évite tout flash)
      requestAnimationFrame(() => {
        requestAnimationFrame(() => {
          setActiveIdx(nextIdx)
        })
      })
    }, ROTATION_INTERVAL_MS)

    return () => clearInterval(timer)
  }, [images])

  return (
    <div
      aria-hidden="true"
      data-testid="home-hero-banner"
      className="relative overflow-hidden rounded-lg border border-border/80 bg-card/70 shadow-[0_30px_80px_-48px_rgba(15,23,42,0.95)]"
    >
      <div className="pointer-events-none absolute inset-x-0 top-0 h-px bg-gradient-to-r from-transparent via-primary/70 to-transparent" />
      <div className="pointer-events-none absolute inset-0 bg-gradient-to-r from-background/20 via-transparent to-background/28" />
      <div className="pointer-events-none absolute inset-x-0 bottom-0 h-16 bg-gradient-to-t from-background/45 to-transparent" />

      <div className="relative h-36 w-full sm:h-48 lg:h-56">
        {srcs.map((src, i) => (
          <div
            key={i}
            style={{
              position: 'absolute',
              inset: 0,
              backgroundImage: src ? `url(${src})` : undefined,
              backgroundSize: 'cover',
              backgroundPosition: 'center',
              transition: `opacity ${FADE_DURATION_MS}ms ease`,
              opacity: activeIdx === i ? 1 : 0,
            }}
          />
        ))}
      </div>
    </div>
  )
}
