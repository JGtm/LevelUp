/**
 * sequenceCause.logic — ce que la page dit d'une séquence : toujours un état pour une contribution
 * choisie (séquence, chargement, cause d'absence, « se lit sur les morts »), sauf fonction fermée.
 */
import { describe, expect, it } from 'vitest'

import { FRAG, MORT_COMPLETE } from './mort.fixtures'
import { causeDeLErreur, CODE_SEQUENCE, etatDeLaSequence, type RequeteDeSequence } from './sequenceCause.logic'

const erreur = (code: string) => ({ code, message: 'x', retryable: false, status: 404 })
const REQUETE: RequeteDeSequence = { demandee: true, enCours: false, servie: false, erreur: null }
const etat = (requete: Partial<RequeteDeSequence>, extra: { rejeuDuTitre?: boolean; frag?: boolean } = {}) =>
  etatDeLaSequence({
    contribution: extra.frag ? FRAG : MORT_COMPLETE,
    rejeuDuTitre: extra.rejeuDuTitre ?? true,
    requete: { ...REQUETE, ...requete },
  })

describe('cause d’une requête en échec', () => {
  it('chaque code servi par la route donne sa cause', () => {
    expect(causeDeLErreur(erreur(CODE_SEQUENCE.analyseEnCours))).toBe('analyse_en_cours')
    expect(causeDeLErreur(erreur(CODE_SEQUENCE.filmNonAnalyse))).toBe('film_non_analyse')
    expect(causeDeLErreur(erreur(CODE_SEQUENCE.filmSansHorloge))).toBe('film_sans_horloge')
    expect(causeDeLErreur(erreur(CODE_SEQUENCE.carteSansRayon))).toBe('carte_sans_rayon')
    expect(causeDeLErreur(erreur(CODE_SEQUENCE.sansPosition))).toBe('sans_position')
  })

  it('le refus du garde du rejeu est la fonction fermée, pas une cause', () => {
    expect(causeDeLErreur(erreur('replay_not_available'))).toBe('refus')
  })

  it('une erreur réseau ou un code inconnu : la cause générique, jamais rien', () => {
    expect(causeDeLErreur(new Error('Failed to fetch'))).toBe('inconnue')
    expect(causeDeLErreur(erreur('autre_chose'))).toBe('inconnue')
    expect(causeDeLErreur(undefined)).toBe('inconnue')
  })
})

describe('état de la zone « séquence »', () => {
  it('rien de choisi, ou un titre sans rejeu 2D : aucune zone', () => {
    expect(etatDeLaSequence({ contribution: undefined, rejeuDuTitre: true, requete: REQUETE })).toBeNull()
    expect(etat({ servie: true }, { rejeuDuTitre: false })).toBeNull()
    expect(etat({}, { rejeuDuTitre: false, frag: true })).toBeNull()
  })

  it('séquence servie, ou en chargement', () => {
    expect(etat({ servie: true })).toEqual({ kind: 'servie' })
    expect(etat({ enCours: true })).toEqual({ kind: 'chargement' })
  })

  it('une cause du serveur : l’état vide qui dit pourquoi', () => {
    expect(etat({ erreur: erreur(CODE_SEQUENCE.filmNonAnalyse) })).toEqual({ kind: 'absente', cause: 'film_non_analyse' })
    expect(etat({ erreur: erreur(CODE_SEQUENCE.analyseEnCours) })).toEqual({ kind: 'absente', cause: 'analyse_en_cours' })
  })

  it('fonction fermée par le garde : aucune zone, comme le lien de rejeu', () => {
    expect(etat({ erreur: erreur(CODE_SEQUENCE.refus) })).toBeNull()
  })

  it('erreur inconnue, ou mort sans victime nommée : l’état générique', () => {
    expect(etat({ erreur: new Error('boum') })).toEqual({ kind: 'absente', cause: 'inconnue' })
    expect(etat({ demandee: false })).toEqual({ kind: 'absente', cause: 'inconnue' })
  })

  it('un frag, une entrée, une réapparition : « la séquence se lit sur les morts »', () => {
    expect(etat({ demandee: false }, { frag: true })).toEqual({ kind: 'sur_les_morts' })
    expect(
      etatDeLaSequence({ contribution: { ...FRAG, face: 'entree' }, rejeuDuTitre: true, requete: { ...REQUETE, demandee: false } }),
    ).toEqual({ kind: 'sur_les_morts' })
  })
})
