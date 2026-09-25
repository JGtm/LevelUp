// Pour un WeaponID lu aux offsets fixes, cherche le decalage d (bits) qui fait apparaitre la
// moitie basse d'arme personnelle 0x42C9679F ou une moitie basse nulle d'arme de vehicule.
const PERSO = 0x42C9679Fn, M32 = 0xFFFFFFFFn;
export function decale(hex) {
  const W = BigInt(hex);
  const out = [];
  for (let d = 1; d <= 31; d++) { // champs reels PLUS TOT de d bits : variante = (W >> d) & M32
    const v = (W >> BigInt(d)) & M32;
    if (v === PERSO) out.push({ d: -d, kind: 'perso', tagBas: (W >> BigInt(32 + d)).toString(16), bits: 32 - d });
  }
  for (let s = 1; s <= 31; s++) { // champs reels PLUS TARD de s bits : haut de variante = W & mask
    const mask = (1n << BigInt(32 - s)) - 1n;
    if ((W & mask) === (PERSO >> BigInt(s))) out.push({ d: +s, kind: 'perso', tag: ((W >> BigInt(32 - s)) & M32).toString(16) });
  }
  return out;
}
const known = { '48c19d2d':'MA40', '2b1824d5':'BR75', 'f408190f':'Sidekick', '2fb21c87':'Bandit', 'b533957e':'Needler', '71ab0a2c':'SPNKr', '767db96d':'Hydra', 'b619d84a':'Bulldog', 'd7915565':'Mutilator', 'fd98554c':'Commando', '0a1992bc':'S7' };
for (const h of process.argv.slice(2)) {
  const r = decale(h);
  console.log(h, JSON.stringify(r.map(x => ({...x, nom: Object.entries(known).find(([k]) => x.tag ? k === x.tag.padStart(8,'0') : k.endsWith(x.tagBas))?.[1]}))));
}
