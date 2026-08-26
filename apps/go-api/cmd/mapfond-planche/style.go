package main

// feuilleDeStyle — L'HABILLAGE DE LA PLANCHE.
//
// PARTI PRIS. Le sujet est une planche de releve : des plaques orthographiques grises sur
// fond transparent. Les neutres tirent donc vers l'ardoise, la seule couleur d'accent est une
// encre de geometre (ambre), et les couleurs semantiques (valide / refuse / en attente) ne
// servent QUE le statut — elles ne decorent rien.
//
// AUCUNE POLICE DISTANTE : la CSP d'un artefact publie bloque les CDN, et une police
// embarquee en data URI pesterait plus lourd que les vignettes. Les piles systeme sont donc
// choisies, pas subies — grotesque pour le texte, chasse fixe a chiffres tabulaires pour tout
// ce qui s'aligne en colonne.
//
// LE DAMIER N'EST PAS UN ORNEMENT : sans lui, le vide transparent d'un fond se confond avec
// le fond de la page et le defaut de cadrage devient invisible. C'est l'objet meme de la
// planche.
const feuilleDeStyle = `
:root {
  --sol: #e9ecf0;
  --plaque: #f8fafb;
  --damier: #b9c4d0;
  --encre: #111820;
  --sourd: #5a6675;
  --trait: #ccd3db;
  --accent: #a86a12;
  --ok: #2f7d5c;
  --ko: #a83f38;
  --attente: #a86a12;
  --grotesque: ui-sans-serif, system-ui, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
  --chasse: ui-monospace, "SF Mono", "Cascadia Mono", Consolas, "Liberation Mono", monospace;
}
@media (prefers-color-scheme: dark) {
  :root:not([data-theme="light"]) {
    --sol: #0b0f14;
    --plaque: #141a21;
    --damier: #313f4c;
    --encre: #dfe6ee;
    --sourd: #7d8b9b;
    --trait: #232c36;
    --accent: #e0a64a;
    --ok: #54b08a;
    --ko: #e0716a;
    --attente: #e0a64a;
  }
}
:root[data-theme="dark"] {
  --sol: #0b0f14;
  --plaque: #141a21;
  --damier: #313f4c;
  --encre: #dfe6ee;
  --sourd: #7d8b9b;
  --trait: #232c36;
  --accent: #e0a64a;
  --ok: #54b08a;
  --ko: #e0716a;
  --attente: #e0a64a;
}

body {
  background: var(--sol);
  color: var(--encre);
  font-family: var(--grotesque);
  line-height: 1.5;
  margin: 0;
  padding: clamp(1.5rem, 4vw, 3.5rem) clamp(1rem, 4vw, 3rem) 5rem;
}

.tete { max-width: 62ch; display: flex; flex-direction: column; gap: 0.75rem; margin-bottom: 2.5rem; }
.surtitre {
  font-family: var(--chasse); font-size: 0.7rem; letter-spacing: 0.14em;
  text-transform: uppercase; color: var(--accent); margin: 0;
}
h1 { font-size: clamp(1.6rem, 3.6vw, 2.4rem); line-height: 1.15; letter-spacing: -0.02em; margin: 0; text-wrap: balance; }
.intro { margin: 0; color: var(--sourd); }
.legende {
  margin: 0.5rem 0 0; padding-top: 0.85rem; border-top: 1px solid var(--trait);
  font-size: 0.85rem; color: var(--sourd);
}
.legende .ex {
  display: inline-block; width: 1.6rem; height: 0.9rem; vertical-align: -1px; margin-right: 0.55rem;
  border: 1px solid var(--sourd); outline: 1px dashed var(--accent); outline-offset: -4px;
}

.planche { display: grid; gap: 1.25rem; grid-template-columns: repeat(auto-fill, minmax(290px, 1fr)); }
.fiche {
  background: var(--plaque); border: 1px solid var(--trait); border-radius: 3px;
  padding: 0.9rem 0.9rem 1rem; display: flex; flex-direction: column; gap: 0.6rem;
}
/* Une fiche de COMPARAISON prend toute la rangee : deux etats se jugent cote a cote ou pas du tout. */
.fiche:has(figure + figure) { grid-column: 1 / -1; }

.entete { display: flex; align-items: baseline; justify-content: space-between; gap: 0.75rem; }
h2 { font-size: 1rem; letter-spacing: -0.01em; margin: 0; }
.sous { margin: -0.35rem 0 0; font-family: var(--chasse); font-size: 0.72rem; color: var(--sourd); font-variant-numeric: tabular-nums; }
.statut {
  font-family: var(--chasse); font-size: 0.62rem; letter-spacing: 0.08em; text-transform: uppercase;
  white-space: nowrap; padding: 0.18rem 0.45rem; border: 1px solid currentColor; border-radius: 2px;
}
.statut.ok { color: var(--ok); }
.statut.ko { color: var(--ko); }
.statut.attente { color: var(--attente); }

.colonnes { display: grid; gap: 0.9rem; grid-template-columns: repeat(auto-fit, minmax(240px, 1fr)); }
figure { margin: 0; display: flex; flex-direction: column; gap: 0.45rem; }
.cadre {
  border: 1px solid var(--sourd);
  background-color: var(--damier);
  background-image:
    linear-gradient(45deg, var(--plaque) 25%, transparent 25%, transparent 75%, var(--plaque) 75%),
    linear-gradient(45deg, var(--plaque) 25%, transparent 25%, transparent 75%, var(--plaque) 75%);
  background-size: 16px 16px;
  background-position: 0 0, 8px 8px;
  display: flex; align-items: center; justify-content: center;
}
.cadre img { display: block; max-width: 100%; height: auto; }
figcaption { display: flex; flex-direction: column; gap: 0.12rem; }
.col { font-size: 0.8rem; font-weight: 600; }
.chiffres { font-family: var(--chasse); font-size: 0.7rem; color: var(--sourd); font-variant-numeric: tabular-nums; }
.chiffres b { color: var(--encre); font-weight: 600; }

@media (prefers-reduced-motion: no-preference) {
  .fiche { transition: border-color 120ms ease; }
}
.fiche:hover { border-color: var(--sourd); }
`
