import { afterEach, describe, expect, it } from 'vitest'
import {
  addDescribedBy,
  claimTitle,
  findTitleSource,
  isCoveredByTooltip,
  refreshClaim,
  releaseTitle,
  removeDescribedBy,
  STASH_ATTRIBUTE,
  TOOLTIP_OWNER_ATTRIBUTE,
} from './titleTooltip.logic'

function mount(html: string): HTMLElement {
  document.body.innerHTML = html
  return document.body
}

afterEach(() => {
  document.body.innerHTML = ''
})

describe('findTitleSource : choix de l’élément', () => {
  it('prend l’élément le plus proche qui porte un title, enfants compris', () => {
    mount('<div title="Parent"><button id="b" title="Bouton"><span id="s">x</span></button></div>')
    const source = findTitleSource(document.getElementById('s'))
    expect(source?.owner.id).toBe('b')
    expect(source?.text).toBe('Bouton')
  })

  it('remonte jusqu’au parent quand l’enfant n’a pas de title', () => {
    mount('<div id="p" title="Parent"><span id="s">x</span></div>')
    expect(findTitleSource(document.getElementById('s'))?.owner.id).toBe('p')
  })

  it('un title vide plus proche supprime celui des ancêtres, comme le navigateur', () => {
    mount('<div title="Parent"><span id="s" title="">x</span></div>')
    expect(findTitleSource(document.getElementById('s'))).toBeNull()
  })

  it('ne trouve rien sans title', () => {
    mount('<div><span id="s">x</span></div>')
    expect(findTitleSource(document.getElementById('s'))).toBeNull()
    expect(findTitleSource(null)).toBeNull()
  })

  it('lit le <title> enfant d’un élément SVG', () => {
    mount('<svg><g id="g"><path id="p"><title>Arc 42</title></path></g></svg>')
    const source = findTitleSource(document.getElementById('p'))
    expect(source?.owner.id).toBe('p')
    expect(source?.text).toBe('Arc 42')
  })

  it('ignore un <title> SVG vide et remonte', () => {
    mount('<div id="d" title="Haut"><svg><path id="p"><title> </title></path></svg></div>')
    expect(findTitleSource(document.getElementById('p'))?.owner.id).toBe('d')
  })
})

describe('claimTitle / releaseTitle : restauration', () => {
  it('retire le title pendant la prise en charge et le restitue ensuite', () => {
    mount('<button id="b" title="Aide">Ok</button>')
    const b = document.getElementById('b') as HTMLElement
    const source = findTitleSource(b)!
    claimTitle(source)
    expect(b.getAttribute('title')).toBe('')
    expect(b.getAttribute(STASH_ATTRIBUTE)).toBe('Aide')
    // Un élément pris en charge reste reconnu, y compris depuis un de ses enfants.
    expect(findTitleSource(b)?.text).toBe('Aide')
    releaseTitle(b)
    expect(b.getAttribute('title')).toBe('Aide')
    expect(b.hasAttribute(STASH_ATTRIBUTE)).toBe(false)
  })

  it('est idempotent et sans effet sur un élément non pris en charge', () => {
    mount('<button id="b" title="Aide">Ok</button>')
    const b = document.getElementById('b') as HTMLElement
    const source = findTitleSource(b)!
    claimTitle(source)
    claimTitle(source)
    releaseTitle(b)
    releaseTitle(b)
    expect(b.getAttribute('title')).toBe('Aide')
  })

  it('ne remplace pas un title réécrit pendant la prise en charge', () => {
    mount('<button id="b" title="Ancien">Ok</button>')
    const b = document.getElementById('b') as HTMLElement
    claimTitle(findTitleSource(b)!)
    b.setAttribute('title', 'Neuf')
    releaseTitle(b)
    expect(b.getAttribute('title')).toBe('Neuf')
  })

  it('vide le <title> SVG pendant la prise en charge puis le rend', () => {
    mount('<svg><path id="p"><title>Arc 42</title></path></svg>')
    const p = document.getElementById('p') as unknown as Element
    claimTitle(findTitleSource(p)!)
    expect(p.querySelector('title')?.textContent).toBe('')
    releaseTitle(p)
    expect(p.querySelector('title')?.textContent).toBe('Arc 42')
  })
})

describe('nom accessible', () => {
  it('garde le nom d’un bouton icône dont le title est le seul nom, puis le retire', () => {
    mount('<button id="b" title="Fermer"><svg></svg></button>')
    const b = document.getElementById('b') as HTMLElement
    claimTitle(findTitleSource(b)!)
    expect(b.getAttribute('aria-label')).toBe('Fermer')
    releaseTitle(b)
    expect(b.hasAttribute('aria-label')).toBe(false)
  })

  it('ne touche pas un élément qui a déjà un nom (texte, aria-label, étiquette de champ)', () => {
    mount(
      '<button id="a" title="Aide">Texte</button><button id="b" title="Aide" aria-label="Nom">i</button>' +
        '<label for="c">Champ</label><input id="c" title="Aide">',
    )
    for (const id of ['a', 'b', 'c']) {
      const el = document.getElementById(id) as HTMLElement
      const before = el.getAttribute('aria-label')
      claimTitle(findTitleSource(el)!)
      expect(el.getAttribute('aria-label')).toBe(before)
      releaseTitle(el)
    }
  })

  it('donne un nom à un SVG dont le <title> était le seul nom, puis le retire', () => {
    mount('<svg id="s"><title>Tendance</title></svg>')
    const s = document.getElementById('s') as unknown as Element
    claimTitle(findTitleSource(s)!)
    expect(s.getAttribute('aria-label')).toBe('Tendance')
    releaseTitle(s)
    expect(s.hasAttribute('aria-label')).toBe(false)
  })
})

describe('aria-describedby', () => {
  it('ajoute et retire sa référence sans perdre les autres', () => {
    mount('<button id="b" aria-describedby="autre">Ok</button>')
    const b = document.getElementById('b') as HTMLElement
    addDescribedBy(b, 'tip')
    addDescribedBy(b, 'tip')
    expect(b.getAttribute('aria-describedby')).toBe('autre tip')
    removeDescribedBy(b, 'tip')
    expect(b.getAttribute('aria-describedby')).toBe('autre')
    removeDescribedBy(b, 'autre')
    expect(b.hasAttribute('aria-describedby')).toBe(false)
  })
})

describe('isCoveredByTooltip', () => {
  it('vrai sous un conteneur d’infobulle JS, faux ailleurs', () => {
    mount(`<div ${TOOLTIP_OWNER_ATTRIBUTE}=""><button id="in" title="A">x</button></div><button id="out" title="B">y</button>`)
    expect(isCoveredByTooltip(document.getElementById('in') as HTMLElement)).toBe(true)
    expect(isCoveredByTooltip(document.getElementById('out') as HTMLElement)).toBe(false)
  })
})

describe('refreshClaim : titre réécrit ou retiré par l’application', () => {
  it('une réécriture remplace le texte retenu et revide le title', () => {
    mount('<button id="b" title="Ancien"><svg></svg></button>')
    const b = document.getElementById('b') as HTMLElement
    claimTitle(findTitleSource(b)!)
    b.setAttribute('title', 'Neuf')
    expect(refreshClaim(b)).toEqual({ kind: 'text', text: 'Neuf' })
    expect(b.getAttribute('title')).toBe('')
    expect(b.getAttribute(STASH_ATTRIBUTE)).toBe('Neuf')
    // Le nom accessible provisoire suit le texte neuf, puis disparaît.
    expect(b.getAttribute('aria-label')).toBe('Neuf')
    releaseTitle(b)
    expect(b.getAttribute('title')).toBe('Neuf')
    expect(b.hasAttribute('aria-label')).toBe(false)
  })

  it('un retrait oublie la prise en charge : rien n’est restitué', () => {
    mount('<button id="b" title="Ancien">Ok</button>')
    const b = document.getElementById('b') as HTMLElement
    claimTitle(findTitleSource(b)!)
    b.removeAttribute('title')
    expect(refreshClaim(b)).toEqual({ kind: 'gone' })
    expect(b.hasAttribute(STASH_ATTRIBUTE)).toBe(false)
    releaseTitle(b)
    expect(b.hasAttribute('title')).toBe(false)
  })

  it('sans changement : same', () => {
    mount('<button id="b" title="Aide">Ok</button>')
    const b = document.getElementById('b') as HTMLElement
    claimTitle(findTitleSource(b)!)
    expect(refreshClaim(b)).toEqual({ kind: 'same' })
  })

  it('relit le <title> SVG réécrit ou supprimé', () => {
    mount('<svg><path id="p"><title>Arc 1</title></path></svg>')
    const p = document.getElementById('p') as unknown as Element
    claimTitle(findTitleSource(p)!)
    p.querySelector('title')!.textContent = 'Arc 2'
    expect(refreshClaim(p)).toEqual({ kind: 'text', text: 'Arc 2' })
    expect(p.querySelector('title')?.textContent).toBe('')
    p.querySelector('title')!.remove()
    expect(refreshClaim(p)).toEqual({ kind: 'gone' })
    expect(p.hasAttribute(STASH_ATTRIBUTE)).toBe(false)
  })
})
