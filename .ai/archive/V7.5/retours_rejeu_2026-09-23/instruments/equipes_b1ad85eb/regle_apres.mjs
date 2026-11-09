// regle_apres.mjs — LA REGLE PROPOSEE, SIMULEE : « une place par siege occupe a l'instant T,
// depart = sortie du panneau, remplacant = meme place ».
//
// CE QUE CETTE SIMULATION PREND COMME PRESENCE, ET POURQUOI CE N'EST PAS LA SOURCE CIBLE. La cible
// (option 1 du rapport) est la presence lue dans le FILM (entite ti=9 par occupant, image-cle par
// image-cle, affinee par les vies). Elle n'est pas dans les documents cuits : la simulation prend
// donc un SUBSTITUT fait de ce qui existe sans decodage —
//   presence = [min(premiere vie, arrivee API), fin]   fin = fin du film si l'API ne dit pas « parti »,
//              sinon max(derniere vie, depart API), puis BORNEE par l'arrivee du successeur sur la
//              meme place (une place n'a qu'un occupant) ;
//   equipe   = equipe du film (roster/pistes) quand elle est publiee, sinon equipe API (substitut du
//              designateur par ENTITE que le film porte — cf. rapport §2) ;
//   place    = les occupants du debut prennent chacun une place ; un arrivant prend la place liberee
//              le plus recemment dans son equipe ; faute de place libre, celle d'un occupant que
//              l'API dit PARTI et dont la derniere vie est finie (son depart est alors borne a
//              l'arrivee) ; sinon debordement COMPTE.
import { trackWindow, playerLabel } from './panneaux_lib.mjs';

export function groupes(doc, players, parts, { toF }) {
  const parCle = new Map(players.map((p) => [p.xuid, p]));
  const roster = new Map((doc.roster ?? []).map((e) => [e.xuid || (e.bot && e.name ? `bot:${e.name}` : ''), e]));
  const occ = [];
  const vus = new Set();
  const ajouter = (cle, nom, equipeAPI, api) => {
    const p = parCle.get(cle);
    const e = roster.get(cle);
    const vies = p?.lives ?? [];
    const L0 = vies.length ? Math.min(...vies.map((l) => trackWindow(l).start)) : Infinity;
    const L1 = vies.length ? Math.max(...vies.map((l) => trackWindow(l).end)) : -Infinity;
    let equipe = e?.team ?? null;
    let source = 'film';
    if (equipe === null || equipe === undefined) {
      const t = vies.find((l) => l.team !== undefined && l.team >= 0)?.team;
      if (t !== undefined) equipe = t;
      else { equipe = equipeAPI; source = 'api'; }
    }
    if (equipe === null || equipe === undefined) return;
    // Present au coup d'envoi selon l'API : present des la premiere image du film.
    const A0 = api ? (api.debut ? 0 : Math.max(0, api.a)) : Infinity;
    const debut = Math.min(L0, A0);
    if (!Number.isFinite(debut)) return;
    const parti = api ? api.left : false;
    const fin = parti ? Math.max(L1, api.b) : Infinity;
    occ.push({ cle, nom, equipe, source, debut, fin, parti, L1, idx: e?.filmIndex ?? 99, player: p });
    vus.add(cle);
  };
  for (const r of parts) {
    const cle = r.xuid.startsWith('bid(') ? `bot:${r.gamertag} [bot]` : r.xuid;
    const api = {
      a: r.fj == null ? -Infinity : toF(r.fj),
      b: r.ll == null ? Infinity : toF(r.ll),
      left: r.left_in_progress === 'true',
      debut: r.present_at_beginning === 'true',
    };
    ajouter(cle, r.gamertag, r.team_id == null ? null : Number(r.team_id), api);
  }
  // Les occupants que le film nomme et que l'API ne porte pas (rare) : presence = leurs vies.
  for (const p of players) if (!vus.has(p.xuid) && p.lives.length) ajouter(p.xuid, playerLabel(p), null, null);
  const parEquipe = new Map();
  for (const o of occ) {
    if (!parEquipe.has(o.equipe)) parEquipe.set(o.equipe, []);
    parEquipe.get(o.equipe).push(o);
  }
  const out = [];
  for (const [equipe, liste] of [...parEquipe.entries()].sort((a, b) => a[0] - b[0])) {
    liste.sort((a, b) => a.debut - b.debut || a.idx - b.idx);
    const places = [];
    for (const o of liste) {
      // place libre : son dernier occupant est sorti avant l'arrivee — la plus recemment liberee.
      let choix = -1;
      let finChoix = -Infinity;
      places.forEach((pl, i) => {
        const der = pl[pl.length - 1];
        if (der.fin <= o.debut && der.fin > finChoix) { choix = i; finChoix = der.fin; }
      });
      if (choix < 0) {
        // pas de place libre : celle d'un occupant PARTI (API) dont la derniere vie est finie.
        let meilleur = Infinity;
        places.forEach((pl, i) => {
          const der = pl[pl.length - 1];
          if (der.parti && der.L1 <= o.debut && der.fin < meilleur) { choix = i; meilleur = der.fin; }
        });
        if (choix >= 0) places[choix][places[choix].length - 1].fin = o.debut;
      }
      if (choix < 0) places.push([o]);
      else places[choix].push(o);
    }
    out.push({ key: `f${equipe}`, side: `t${equipe}`, places });
  }
  return out;
}

export function panneaux(groups, f) {
  return groups.map((g) => ({
    key: g.key,
    side: g.side,
    tuiles: g.places.flatMap((pl, i) => {
      const o = pl.find((x) => x.debut <= f && f < x.fin);
      return o ? [{ seat: i, kind: 'present', fantome: false, name: o.nom, xuid: o.cle, source: o.source }] : [];
    }),
  }));
}
