import { useEffect, useMemo, useState } from 'react'
import { useAppShellStore } from '@/stores/appShellStore'

/**
 * Images headers par titre (slug → liste de chemins dans /public/titles/{slug}/).
 * Pour ajouter un nouveau jeu, créer public/titles/{slug}/ et y lister ses images.
 */
const HEADER_IMAGES_BY_TITLE: Record<string, string[]> = {
  halo_infinite: [
    '/titles/halo_infinite/echoes-within-header.webp',
    '/titles/halo_infinite/Infinite.png',
    '/titles/halo_infinite/LoneWolves.png',
  ],
}

const ROTATION_INTERVAL_MS = 45_000

function pickOther(images: string[], current: string): string {
  const others = images.filter((img) => img !== current)
  return others[Math.floor(Math.random() * others.length)]
}

/**
 * HomeHeroBanner — bandeau visuel décoratif de l'accueil.
 * Effectue une rotation aléatoire entre les images du titre courant toutes les 45 secondes.
 */
export function HomeHeroBanner() {
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  const images = useMemo(
    () => HEADER_IMAGES_BY_TITLE[titleSlug] ?? HEADER_IMAGES_BY_TITLE['halo_infinite'] ?? [],
    [titleSlug],
  )

  const [src, setSrc] = useState(
    () => images[Math.floor(Math.random() * images.length)] ?? '',
  )
  const [visible, setVisible] = useState(true)

  // Relancer le timer et réinitialiser le src quand le titre change
  useEffect(() => {
    if (images.length === 0) return
    setSrc(images[Math.floor(Math.random() * images.length)] ?? '')
    const timer = setInterval(() => {
      setVisible(false)
      setTimeout(() => {
        setSrc((prev) => pickOther(images, prev))
        setVisible(true)
      }, 600)
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

      <img
        src={src}
        alt=""
        className="block h-36 w-full object-cover object-center sm:h-48 lg:h-56"
        style={{ transition: 'opacity 0.6s ease', opacity: visible ? 1 : 0 }}
      />
    </div>
  )
}
