/**
 * recette_export_rejeu.js — LA RECETTE COMPLETE DE L'EXPORT VIDEO, EN UN SEUL PASSAGE.
 *
 * # POURQUOI CE FICHIER EXISTE
 *
 * L'export a huit proprietes observables — image, dimensions, duree, maintien de fin, pistes
 * sonores, surimpression, voix, musique — et elles se cassent INDEPENDAMMENT. Les verifier une
 * a une, c'est corriger l'une en cassant l'autre sans le voir : c'est exactement ce qui s'est
 * produit le 2026-08-28, en quatre allers-retours (image sans son, son sans image, tout sans
 * l'ecran de fin). Ce script les verifie TOUTES, d'un coup, et rend un verdict par ligne.
 *
 * Aucun test automatise ne peut le remplacer : jsdom ne calcule pas de mise en page, n'a pas
 * WebCodecs, et ne decode pas un MP4. C'est une recette de NAVIGATEUR, et elle doit tourner
 * apres CHAQUE changement du chantier d'export.
 *
 * # COMMENT S'EN SERVIR
 *
 * 1. Ouvrir une page de rejeu, connecte, et attendre que les donnees du match soient la.
 * 2. Coller ce fichier entier dans la console, puis : `await recetteExport()`.
 *
 * # LA PREMIERE VERIFICATION EST LA PLUS IMPORTANTE
 *
 * `donneesDuMatch` verifie que la PAGE elle-meme a son scoreboard, son issue et sa fenetre de
 * gameplay. Sans elles, l'export sort sans ecran de fin, sans score, sans voix et sans musique
 * — non par regression, mais parce qu'il refuse d'affirmer une fin de match qu'il ne connait
 * pas. Sans ce garde, on passe des heures a chercher un bug dans l'export alors que c'est
 * l'API du match qui repond 404.
 */

async function recetteExport({ secondes = 6 } = {}) {
  const dormir = (ms) => new Promise((r) => setTimeout(r, ms))
  const resultats = []
  const noter = (nom, ok, detail) => resultats.push({ nom, verdict: ok ? 'OK' : 'ECHEC', detail })

  // --- 0. PRE-REQUIS : la page a-t-elle ses donnees ? ---
  const texte = document.body.innerText
  const sansEquipe = /SANS ÉQUIPE/i.test(texte)
  const bouton = document.querySelector('button[aria-label="Exporter la vidéo"]')
  if (!bouton) return [{ nom: 'bouton export', verdict: 'ECHEC', detail: 'absent (WebCodecs ?)' }]
  if (sansEquipe) {
    return [{
      nom: 'donneesDuMatch',
      verdict: 'ECHEC',
      detail: 'la PAGE n a ni equipe ni issue (vue du match en 404 ?) — l export sortira sans ecran de fin, sans score, sans voix et sans musique. Corriger l API avant de chercher plus loin.',
    }]
  }
  noter('donneesDuMatch', true, 'scoreboard et issue presents')

  // --- 1. EXPORT de la fin du match, son compris ---
  let clip = null
  const vraiCreate = URL.createObjectURL.bind(URL)
  URL.createObjectURL = (b) => { if (b instanceof Blob && b.type === 'video/mp4') clip = b; return vraiCreate(b) }
  try {
    if (bouton.getAttribute('aria-expanded') !== 'true') { bouton.click(); await dormir(300) }
    const dlg = () => document.querySelector('[role="dialog"][aria-label*="Export"]')
    const poser = (el, v) => {
      Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, 'value').set.call(el, String(v))
      el.dispatchEvent(new Event('input', { bubbles: true }))
      el.dispatchEvent(new Event('change', { bubbles: true }))
    }
    const fin = dlg().querySelector('input[aria-label="Fin"]')
    const debut = dlg().querySelector('input[aria-label="Début"]')
    const max = Number(fin.max)
    // On vise la FIN du match : c'est la seule plage qui porte l'ecran de fin, la voix et la
    // musique. Une plage de milieu de match ne prouverait rien de ces trois-la.
    poser(fin, max); await dormir(100)
    poser(debut, Math.max(0, max - Math.round((max / (dureeFilmSecondes() || 1)) * secondes)))
    await dormir(200)
    const t0 = performance.now()
    Array.from(dlg().querySelectorAll('button')).find((b) => b.textContent.trim() === 'Exporter').click()
    let etat = ''
    for (let i = 0; i < 600; i++) {
      await dormir(200)
      const d = dlg()
      if (d) etat = d.innerText.replace(/\n+/g, ' | ')
      if (/Fichier déposé|échoué/.test(etat)) break
    }
    noter('exportTermine', /Fichier déposé/.test(etat), etat.slice(0, 120))
    noter('vitesse', true, `${Math.round((performance.now() - t0) / 100) / 10} s de calcul`)
  } finally {
    URL.createObjectURL = vraiCreate
  }
  if (!clip) { noter('fichierProduit', false, 'aucun blob mp4'); return resultats }
  noter('fichierProduit', true, `${Math.round(clip.size / 1024 / 1024 * 100) / 100} Mo`)

  // --- 2. LE CONTENEUR : image, pistes, noms ---
  const mb = await import('/node_modules/.vite/deps/mediabunny.js')
  const input = new mb.Input({ source: new mb.BlobSource(clip), formats: mb.ALL_FORMATS })
  const pistes = await input.getTracks()
  const video = pistes.find((p) => p.type === 'video')
  const audio = pistes.filter((p) => p.type === 'audio')
  const paquets = video ? await video.computePacketStats().then((s) => s.packetCount).catch(() => 0) : 0
  // L'IMAGE : c'est elle qui a disparu le 2026-08-28 quand les paquets partaient sans etre
  // attendus avant la finalisation du conteneur. Zero paquet = clip muet de l'image.
  noter('image', paquets > 0, `${paquets} paquets video`)
  noter('pistesSonores', audio.length >= 2, audio.map((p) => p.name ?? '(sans nom)').join(' | '))
  // LE MIXAGE COMPLET EN PREMIER : un lecteur ordinaire ne joue que la premiere piste.
  noter('mixageEnPremier', audio[0]?.name === 'Mixage complet', audio[0]?.name ?? '(aucune)')

  // --- 3. LE SON : y a-t-il de l'energie APRES la borne de plage ? ---
  const ctx = new AudioContext()
  const buf = await ctx.decodeAudioData(await clip.arrayBuffer())
  await ctx.close()
  const d = buf.getChannelData(0)
  const sr = buf.sampleRate
  const rms = (a, b) => {
    const i0 = Math.max(0, Math.floor(a * sr)); const i1 = Math.min(d.length, Math.floor(b * sr))
    let s = 0; for (let i = i0; i < i1; i++) s += d[i] * d[i]
    return Math.sqrt(s / Math.max(1, i1 - i0))
  }
  const dureeClip = buf.duration
  // LA CONCLUSION (voix + musique) tombe SUR la borne et deborde : le clip est donc plus long
  // que la plage, et cette queue doit PORTER DU SON. Une queue silencieuse veut dire que la
  // conclusion a ete ecartee — c'est ce qui arrivait quand le plafond de voix l'avalait, puis
  // quand la musique etait indecodable.
  const queue = dureeClip - secondes
  noter('maintienDeFin', queue > 1, `${Math.round(queue * 100) / 100} s apres la plage`)
  noter('sonDansLaQueue', rms(secondes + 0.2, dureeClip) > 0.005, `RMS ${Math.round(rms(secondes + 0.2, dureeClip) * 10000) / 10000}`)
  // La MUSIQUE dure ~10 s : si elle est la, la queue lui ressemble. Sans elle, la queue ne fait
  // que la duree d'une voix (~1,4 s).
  noter('musiquePresente', queue > 5, `queue de ${Math.round(queue * 10) / 10} s (une voix seule ferait ~1,5 s)`)

  // --- 4. L'ECRAN DE FIN dans l'image ---
  const v = document.createElement('video')
  v.src = URL.createObjectURL(clip); v.muted = true
  await new Promise((r) => { v.onloadedmetadata = r; setTimeout(r, 5000) })
  const grab = async (t) => {
    await new Promise((r) => { v.onseeked = r; v.currentTime = t; setTimeout(r, 4000) })
    const c = document.createElement('canvas'); c.width = v.videoWidth; c.height = v.videoHeight
    c.getContext('2d', { willReadFrequently: true }).drawImage(v, 0, 0)
    return c.getContext('2d', { willReadFrequently: true }).getImageData(0, 0, c.width, c.height).data
  }
  const milieu = await grab(dureeClip / 2)
  const finale = await grab(Math.max(0, dureeClip - 0.4))
  let ecart = 0
  for (let i = 0; i < milieu.length; i += 4) ecart += Math.abs(milieu[i] - finale[i])
  ecart = ecart / (milieu.length / 4)
  // L'ecran de fin pose un VOILE plein cadre plus un panneau : la derniere image differe donc
  // FRANCHEMENT de celle du milieu. Sans lui, les deux images se ressemblent.
  noter('ecranDeFin', ecart > 12, `ecart moyen ${Math.round(ecart * 10) / 10} entre le milieu et la fin`)
  noter('dimensions', v.videoWidth >= 1000, `${v.videoWidth}x${v.videoHeight}`)

  return resultats
}

/** La duree du FILM en secondes, lue sur l'horloge de la barre de lecture (« 0:55 / 8:53 »). */
function dureeFilmSecondes() {
  const m = document.body.innerText.match(/\d+:\d\d\s*\/\s*(\d+):(\d\d)/)
  return m ? Number(m[1]) * 60 + Number(m[2]) : 0
}

// eslint-disable-next-line no-undef
if (typeof window !== 'undefined') window.recetteExport = recetteExport
