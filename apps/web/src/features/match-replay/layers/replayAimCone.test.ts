/**
 * replayAimCone.test.ts — LA LUNETTE (schéma 29) et son effet sur le cône de visée.
 *
 * EXTRAIT DE `replayMarkers.test.ts` : l'ajout de cette suite y faisait franchir le seuil de
 * 500 lignes du dépôt (CLAUDE.md n°5). La découpe tombe sur une frontière nette — ces épreuves
 * ne testent pas le marqueur mais l'OUVERTURE du cône, et elles sont les seules à lire `s`.
 */
/**
 * replayMarkers.test.ts — CE QUE LE MARQUEUR ÉMET, vérifié sans navigateur.
 *
 * Même outil que le reste du rendu (contexte enregistreur, `test/recordingContext.ts`) :
 * on observe la GÉOMÉTRIE ÉMISE — quelles primitives, combien, avec quels réglages — jamais
 * un pixel. Ce fichier garde les décisions d'habillage du 2026-08-16 :
 *   - le NOM sous le point est cerné AVANT d'être rempli (D4 : lisible du blanc au noir) ;
 *   - le BÂTON n'existe plus (D3 : le calque de visée n'émet plus aucun segment) ;
 *   - la FORME dit l'identité (D5 : losange pour un ami, anneau de plus pour « moi ») ;
 *   - le calque des noms s'éteint (D4 : un BTB à 24 joueurs doit pouvoir le taire) ;
 *   - le STYLE DE LA PLANCHE validée le soir même (§1bis du plan) : plus de halo, traînée
 *     dessinée segment par segment à opacité croissante, croix de mort de taille FIXE, cône
 *     de visée à 0,42 rad de demi-ouverture.
 */
import { describe, expect, it } from "vitest";

import { drawTracksLayer, type MarkerStyle } from "./replayMarkers";
import type { ReplayTrackReady } from "../model/replayNormalize";
import { recordingContext, type CanvasOp } from "../test/recordingContext";

const VIEW = {
  bounds: { minX: 0, minY: 0, maxX: 10, maxY: 10 },
  width: 200,
  height: 100,
  pad: 4,
};

/**
 * UNE vie d'UN SEUL point : la traînée a besoin de deux positions pour tracer un segment,
 * donc ce gabarit garantit qu'aucun `lineTo` de traînée ne vient polluer les comptages de
 * forme et de visée. `h` est le regard, la mesure dont le cône se sert.
 */
function singlePointTrack(
  slot: number,
  over: Partial<ReplayTrackReady> = {},
): ReplayTrackReady {
  return {
    slot,
    team: -1,
    xuid: "A",
    startFrame: 0,
    endFrame: 100,
    points: [{ t: 50, x: 5, y: 5, z: 0, h: 90 }],
    ...over,
  };
}

function style(over: Partial<MarkerStyle> = {}): MarkerStyle {
  return {
    colorOfSlot: () => "rgb(1 2 3)",
    ink: "rgb(9 9 9)",
    // Frame LOIN du départ de la vie : l'anneau d'apparition ne doit pas s'ajouter aux
    // comptages d'arcs (il vit 0,8 s après le spawn).
    frame: 50,
    timing: { trail: 30, aimHold: 60, death: 20, spawn: 5 },
    // Amplitude verticale connue et joueur AU SOL : aucun anneau d'étage ne s'ajoute aux
    // comptages de formes (une carte plate rendrait, elle, l'étage médian).
    z: { min: 0, max: 10 },
    k: 1,
    showAim: false,
    markOfSlot: () => undefined,
    nameOfSlot: () => "Spartan",
    showTrail: true,
    selfInk: "rgb(4 4 4)",
    deathInk: "rgb(5 5 5)",
    labelStroke: "rgb(8 12 18)",
    ...over,
  };
}

/** trace rend les primitives émises par le calque pour une vie et un style donnés. */
function trace(
  over: Partial<MarkerStyle> = {},
  track = singlePointTrack(512),
): CanvasOp[] {
  const { ops, ctx } = recordingContext();
  drawTracksLayer(ctx, [track], VIEW, style(over));
  return ops;
}

/**
 * LA LUNETTE (schéma 29) : l'OUVERTURE du cône, et elle seule.
 *
 * Le contrat tient en une phrase : la lunette resserre l'ANGLE, l'élévation change la LONGUEUR,
 * et les deux ne se marchent jamais dessus. Cette suite le vérifie DANS LES DEUX SENS — c'est
 * l'orthogonalité qui est testée, pas seulement le rétrécissement.
 */
describe("lunette (schéma 29)", () => {
  /** epaulee : une vie d'un point, cap plein est, palier de lunette et élévation au choix. */
  const epaulee = (s?: number, p?: number) =>
    singlePointTrack(512, {
      points: [
        {
          t: 50,
          x: 5,
          y: 5,
          z: 0,
          h: 360,
          ...(s === undefined ? {} : { s }),
          ...(p === undefined ? {} : { p }),
        },
      ],
    });

  /** secteur : les bornes angulaires et le rayon du seul `arc` du relevé. */
  const secteur = (s?: number, p?: number) => {
    const ops = trace(
      { showAim: true, markOfSlot: () => "friend" },
      epaulee(s, p),
    );
    const [, , radius, from, to] = ops.find((o) => o.op === "arc")!
      .args as number[];
    return { ouverture: to - from, rayon: radius };
  };

  it("RESSERRE l ouverture au premier cran — 0,84 rad à la hanche", () => {
    expect(secteur(undefined).ouverture).toBeCloseTo(0.84, 6);
    expect(secteur(1).ouverture).toBeCloseTo((2 * 0.42) / 2.3, 6);
  });

  it("RESSERRE ENCORE au SECOND cran — chaque palier divise l ouverture", () => {
    // Le film publie un NIVEAU, pas un booléen : les armes à deux crans (fusil de précision)
    // émettent le palier 2, mesuré sur quatre films. Le rendu doit les distinguer.
    const cran1 = secteur(1).ouverture;
    const cran2 = secteur(2).ouverture;
    expect(cran2).toBeLessThan(cran1);
    expect(cran2).toBeCloseTo((2 * 0.42) / (2.3 * 2.3), 6);
  });

  it("PLANCHE l ouverture aux crans extrêmes — un cône ne devient jamais un trait", () => {
    // Sans plancher, un troisième cran rendrait le secteur plus fin que le contour du marqueur :
    // l information « il est épaulé » se perdrait au moment où elle est la plus forte.
    expect(secteur(3).ouverture).toBeCloseTo(2 * 0.07, 6);
    expect(secteur(9).ouverture).toBeCloseTo(2 * 0.07, 6);
  });

  it("ne touche PAS à la longueur : c est le domaine de l élévation", () => {
    // Même élévation des deux côtés -> même rayon. Sans cet invariant, les deux mécaniques se
    // confondraient et le cône ne dirait plus laquelle des deux a bougé.
    expect(secteur(1).rayon).toBeCloseTo(secteur(undefined).rayon, 6);
    expect(secteur(1, -30).rayon).toBeCloseTo(secteur(undefined, -30).rayon, 6);
  });

  it("se CUMULE avec l élévation sans la contredire — étroit ET court en plongée", () => {
    const plongeeEpaulee = secteur(1, -30);
    const platHanche = secteur(undefined, 0);
    expect(plongeeEpaulee.ouverture).toBeLessThan(platHanche.ouverture);
    expect(plongeeEpaulee.rayon).toBeLessThan(platHanche.rayon);
  });

  it("ABSENT se lit « pas à la lunette », jamais « inconnu »", () => {
    // Contrat du champ : un artefact antérieur au schéma 29 ne porte pas `s`, et il doit
    // dessiner l ouverture large — ce qu il a toujours fait.
    expect(secteur(undefined).ouverture).toBeCloseTo(0.84, 6);
    expect(secteur(0).ouverture).toBeCloseTo(0.84, 6);
  });
});
