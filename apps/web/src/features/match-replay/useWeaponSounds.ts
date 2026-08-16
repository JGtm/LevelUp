/**
 * useWeaponSounds — le pont entre la tête de lecture du rejeu et le lecteur de sons.
 *
 * TROIS RESPONSABILITÉS, pas une de plus :
 *
 * 1. S'INITIALISER AU MONTAGE, S'AMORCER AU GESTE SI BESOIN. La politique anti-autoplay
 *    des navigateurs compte l'activation AU NIVEAU DE LA PAGE : dans une appli monopage,
 *    le clic de navigation qui a mené au rejeu suffit déjà, et le contexte créé au montage
 *    démarre directement. Il ne reste suspendu que si la page n'a JAMAIS reçu de geste
 *    (lien direct, rafraîchissement) — cas où aucun navigateur n'autorisera le son de
 *    toute façon ; `amorcer()`, appelé par le canvas au premier pointeur, le reprend
 *    alors. Chargement du manifeste et préchargement se font au montage, pour les seules
 *    armes que CE document utilise — tout est idempotent.
 * 2. AVANCER avec la lecture. `avancer(avant, courant, maxAvance)` joue les tirs franchis,
 *    bornés par les garde-fous de `weaponSoundTrigger` — un document sans manifeste livré
 *    reste simplement muet (contrat du lecteur : jamais une erreur).
 * 3. SUIVRE les réglages d'admin. Les deux curseurs (variation, distance) sont servis par
 *    `useSettings` ; toute mise à jour s'applique à la lecture suivante.
 */
import { useCallback, useEffect, useMemo, useRef } from 'react'

import { useSettings } from '@/features/settings/queries'

import {
  DEFAULT_WEAPON_SOUND_SETTINGS,
  WeaponSoundPlayer,
  fetchWeaponSoundManifest,
  type WeaponSoundSettings,
} from './weaponSoundPlayer'
import { tirsAJouer, weaponFamilyKey, type TirMinimal } from './weaponSoundTrigger'

interface DocumentSonore {
  shots: readonly TirMinimal[]
  titleSlug: string
}

export interface SonsDuRejeu {
  /** À appeler sur le premier geste utilisateur du canvas. Idempotent. */
  amorcer: () => void
  /** À appeler à chaque pas de lecture avec l'ancienne et la nouvelle frame. */
  avancer: (avant: number, courant: number, maxAvance: number) => void
}

export function useWeaponSounds(doc: DocumentSonore): SonsDuRejeu {
  const { data: settings } = useSettings()
  const ctxRef = useRef<AudioContext | null>(null)
  const playerRef = useRef<WeaponSoundPlayer | null>(null)
  const amorceRef = useRef(false)

  // Les réglages du moment, lisibles depuis l'amorçage sans en faire une dépendance.
  const reglages: WeaponSoundSettings = useMemo(
    () => ({
      variationPercent:
        settings?.replay_sound_variation_percent ?? DEFAULT_WEAPON_SOUND_SETTINGS.variationPercent,
      distancePercent:
        settings?.replay_sound_distance_percent ?? DEFAULT_WEAPON_SOUND_SETTINGS.distancePercent,
    }),
    [settings],
  )
  const reglagesRef = useRef(reglages)
  useEffect(() => {
    reglagesRef.current = reglages
    playerRef.current?.setSettings(reglages)
  }, [reglages])

  // Les familles d'armes que CE document peut jouer : on ne précharge rien d'autre.
  const familles = useMemo(() => {
    const out = new Set<string>()
    for (const s of doc.shots) {
      const cle = weaponFamilyKey(s.w)
      if (cle) out.add(cle)
    }
    return [...out]
  }, [doc.shots])
  const famillesRef = useRef(familles)
  useEffect(() => {
    famillesRef.current = familles
  }, [familles])

  const initialiser = useCallback(() => {
    if (amorceRef.current) return
    amorceRef.current = true
    const Contexte = window.AudioContext
    if (!Contexte) return
    const ctx = new Contexte()
    const player = new WeaponSoundPlayer(ctx, reglagesRef.current)
    ctxRef.current = ctx
    playerRef.current = player
    void (async () => {
      player.setManifest(await fetchWeaponSoundManifest(fetch, doc.titleSlug))
      await player.preload(famillesRef.current, fetch, doc.titleSlug)
    })()
  }, [doc.titleSlug])

  // Au montage : dans le cas nominal (navigation dans l'app), le contexte démarre tout de
  // suite et le rejeu a du son SANS geste supplémentaire.
  useEffect(() => {
    initialiser()
  }, [initialiser])

  const amorcer = useCallback(() => {
    initialiser()
    const ctx = ctxRef.current
    if (ctx && ctx.state === 'suspended') void ctx.resume().catch(() => {})
  }, [initialiser])

  const avancer = useCallback(
    (avant: number, courant: number, maxAvance: number) => {
      const player = playerRef.current
      if (!player) return
      for (const cle of tirsAJouer(doc.shots, avant, courant, maxAvance)) {
        player.play(cle)
      }
    },
    [doc.shots],
  )

  // Le contexte audio est une ressource du navigateur : il se ferme avec le composant.
  useEffect(
    () => () => {
      void ctxRef.current?.close().catch(() => {})
      ctxRef.current = null
      playerRef.current = null
    },
    [],
  )

  return useMemo(() => ({ amorcer, avancer }), [amorcer, avancer])
}
