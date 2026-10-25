/* 晟帆主页 · 林木（青绿长卷用）
   树按"丛"成组、高低错落，有干有冠。冠由几瓣不规则叶团合成：下半压暗、上沿提亮，再点墨点（点叶）。
   一排树按类名合并成几条路径，后排先画，遮挡关系不乱。
   四时：落叶树（g 槐榆 / a 枫 / y 银杏）冬天只剩蟹爪枯枝；常绿树（p 松 / c 柏 / e 常绿阔叶）冬天顶上积雪。 */
(function () {
  'use strict';
  const { rng } = window.ART2;
  const N = v => Math.round(v * 10) / 10;

  const ell = (cx, cy, rx, ry) => `M${N(cx - rx)} ${N(cy)}a${N(rx)} ${N(ry)} 0 1 0 ${N(rx * 2)} 0a${N(rx)} ${N(ry)} 0 1 0 ${N(-rx * 2)} 0Z`;
  // 上拱的月牙（积雪、高光用）
  const cap = (cx, cy, w, h, t = 0.45) => `M${N(cx - w)} ${N(cy)}Q${N(cx)} ${N(cy - h * 2)} ${N(cx + w)} ${N(cy)}Q${N(cx)} ${N(cy - h * 2 * t)} ${N(cx - w)} ${N(cy)}Z`;

  // 按类名归并路径
  function Bucket() {
    const m = new Map();
    return {
      add(cls, d) { if (d) m.set(cls, (m.get(cls) || '') + d); },
      flush(P) { [...m.keys()].map((c, i) => [rank(c), i, c]).sort((a, b) => a[0] - b[0] || a[1] - b[1]).forEach(([, , c]) => P.p(m.get(c), c)); m.clear(); },
    };
  }
  const DEC = { g: 1, a: 1, y: 1 };
  // 输出顺序：枯枝、干、枝 → 冠 → 暗部 → 亮部 → 墨点 → 积雪
  const RANK = { 'fl-bare': 0, 'fl-trunk': 1, 'fl-bark': 2, 'fl-limb': 2, 'fl-sh': 4, 'fl-hi': 5, 'fl-hl': 6, 'fl-needle': 7, 'fl-dot': 8, 'fl-dota': 8, 'fl-dt': 8, 'fl-dta': 8, 'fl-snow': 9, 'fl-sn': 9 };
  const rank = c => RANK[c.split(' ')[0]] ?? 3;

  // 干：根部略粗，向上收；h 是整棵树高，to 是干顶占树高的比例
  function trunk(B, x, y, h, lean, to, w) {
    const ty = y - h * to, tx = x + lean, m = (y + ty) / 2;
    B.add('fl-trunk', `M${N(x - w)} ${N(y)}Q${N(x - w * 0.55 + lean * 0.35)} ${N(m)} ${N(tx - w * 0.3)} ${N(ty)}L${N(tx + w * 0.3)} ${N(ty)}Q${N(x + w * 0.55 + lean * 0.35)} ${N(m)} ${N(x + w)} ${N(y)}Z`);
    return [tx, ty];
  }

  // 蟹爪枯枝：三股主枝，各分两三回，枝梢上挑
  function bare(B, x0, y0, len, r, depth) {
    let d = '';
    const grow = (px, py, a, l, k) => {
      const ex = px + Math.cos(a) * l, ey = py + Math.sin(a) * l;
      const bend = (r() - 0.5) * l * 0.35;
      d += `M${N(px)} ${N(py)}Q${N((px + ex) / 2 - Math.sin(a) * bend)} ${N((py + ey) / 2 + Math.cos(a) * bend)} ${N(ex)} ${N(ey)}`;
      if (k > 0) for (const s of [-1, 1]) grow(ex, ey, a + s * (0.32 + r() * 0.4) - 0.08, l * (0.58 + r() * 0.14), k - 1);
    };
    for (const a of [-2.3, -1.6, -0.9]) grow(x0, y0, a + (r() - 0.5) * 0.3, len * (0.85 + r() * 0.3), depth);
    B.add('fl-bare', d);
  }

  // 阔叶树冠（点叶）：几瓣叶团合成，下半压暗，左上提亮，再点墨点
  function broad(B, x, y, h, kind, r) {
    const lean = (r() - 0.5) * h * 0.1, dec = !!DEC[kind], sfx = dec ? ' fl-dec' : '';
    const [tx, ty] = trunk(B, x, y, h, lean, 0.46, Math.max(0.45, h * 0.038));
    const cx = tx + (r() - 0.5) * h * 0.04, cy = y - h * 0.62;
    const R = h * (kind === 'y' ? 0.27 : 0.36) * (0.88 + r() * 0.24), ry = R * (kind === 'y' ? 1.2 : 0.82);
    const tiny = h < 20, n = tiny ? 3 : 4 + Math.floor(r() * 3), lobes = [];
    for (let i = 0; i < n; i++) {
      const a = -Math.PI / 2 + (i / n) * Math.PI * 2 + (r() - 0.5) * 0.8, d = 0.4 + r() * 0.16;
      lobes.push([cx + Math.cos(a) * R * d, cy + Math.sin(a) * ry * d, R * (0.48 + r() * 0.18), ry * (0.5 + r() * 0.16)]);
    }
    lobes.push([cx, cy, R * 0.6, ry * 0.6]);
    let d = '', sh = '', hi = '', dt = '', sn = '';
    for (const [lx, ly, rx, rr] of lobes) {
      d += ell(lx, ly, rx, rr);
      if (ly > cy - ry * 0.05) sh += ell(lx + rx * 0.06, ly + rr * 0.2, rx * 0.8, rr * 0.74);
      if (ly < cy + ry * 0.1) { hi += ell(lx - rx * 0.16, ly - rr * 0.2, rx * 0.56, rr * 0.5); sn += cap(lx, ly - rr * 0.55, rx * 0.8, rr * 0.32); }
    }
    const col = { g: 'fl-g', e: 'fl-e', a: 'fl-a', y: 'fl-y' }[kind];
    // 大树冬天画枯枝；小树冬天冠子换成枯枝纹理（省笔墨）
    const big = h >= 60;
    B.add(col + (dec ? (big ? ' fl-dec' : ' fl-w') : ''), d); B.add('fl-sh' + (dec && big ? sfx : ''), sh);
    if (tiny) { if (!dec) B.add('fl-snow', sn); return; }
    B.add('fl-hi' + sfx, hi);
    const nd = Math.round(4 + h * 0.4), s = Math.max(0.5, h * 0.022);
    for (let i = 0; i < nd; i++) {
      const [lx, ly, rx, rr] = lobes[Math.floor(r() * lobes.length)], a = r() * Math.PI * 2, t = 0.25 + r() * 0.6;
      dt += `M${N(lx + Math.cos(a) * rx * t)} ${N(ly + Math.abs(Math.sin(a)) * rr * t * (r() < 0.75 ? 1 : -0.6))}h${N(s * 1.4)}`;
    }
    B.add((kind === 'a' || kind === 'y' ? 'fl-dta' : 'fl-dt') + sfx, dt);
    if (dec && big) bare(B, tx, ty, h * 0.24, r, 2);
    else if (!dec) B.add('fl-snow', sn);
  }

  // 松：干微曲，三四层扁平的松针盘左右交错，层间有横枝
  function pine(B, x, y, h, r) {
    const lean = (r() - 0.5) * h * 0.12;
    trunk(B, x, y, h, lean, 0.9, Math.max(0.5, h * 0.03));
    const tiers = h > 28 ? 4 : 3;
    let d = '', hi = '', sn = '', nd = '', lb = '';
    for (let i = 0; i < tiers; i++) {
      const t = i / (tiers - 1), side = i % 2 ? 1 : -1;
      const ty = y - h * (0.44 + t * 0.5), w = h * (0.3 - t * 0.13) * (0.85 + r() * 0.3), th = w * 0.34;
      const ax = x + lean * (0.44 + t * 0.5), cx = ax + side * w * (i === tiers - 1 ? 0.05 : 0.3);
      if (i < tiers - 1) lb += `M${N(ax)} ${N(ty + th * 0.6)}Q${N((ax + cx) / 2)} ${N(ty - th * 0.2)} ${N(cx + side * w * 0.3)} ${N(ty + th * 0.15)}`;
      d += `M${N(cx - w)} ${N(ty + th * 0.15)}Q${N(cx - w * 0.62)} ${N(ty - th)} ${N(cx)} ${N(ty - th * 1.05)}Q${N(cx + w * 0.62)} ${N(ty - th)} ${N(cx + w)} ${N(ty + th * 0.15)}Q${N(cx)} ${N(ty + th * 0.6)} ${N(cx - w)} ${N(ty + th * 0.15)}Z`;
      hi += cap(cx - w * 0.08, ty - th * 0.35, w * 0.7, th * 0.32);
      sn += cap(cx, ty - th * 0.5, w * 0.86, th * 0.42);
      if (h > 34) { const k = h > 50 ? 9 : 6; for (let j = 0; j < k; j++) { const a = Math.PI * (1.08 + 0.84 * (j + 0.5) / k); nd += `M${N(cx)} ${N(ty + th * 0.35)}l${N(Math.cos(a) * w * 0.92)} ${N(Math.sin(a) * th * 1.15)}`; } }
    }
    B.add('fl-limb', lb); B.add('fl-p', d); B.add('fl-hi', hi); B.add('fl-needle', nd); B.add('fl-snow', sn);
  }

  // 柏：窄而高，五六团深绿叶团上下相叠，顶端钝圆、略歪
  function cypress(B, x, y, h, r) {
    const lean = (r() - 0.5) * h * 0.08;
    trunk(B, x, y, h, lean, 0.3, Math.max(0.5, h * 0.04));
    const n = 6;
    let d = '', sh = '', hi = '', dt = '', sn = '';
    for (let i = 0; i < n; i++) {
      const t = i / (n - 1), cy = y - h * (0.3 + t * 0.6), cx = x + lean * (0.3 + t * 0.7) + (r() - 0.5) * h * 0.05;
      const w = h * (0.17 - t * 0.08) * (0.82 + r() * 0.36), hh = h * 0.115;
      d += ell(cx, cy, w, hh);
      sh += ell(cx + w * 0.12, cy + hh * 0.28, w * 0.74, hh * 0.6);
      hi += ell(cx - w * 0.3, cy - hh * 0.28, w * 0.42, hh * 0.42);
      sn += cap(cx, cy - hh * 0.62, w * 0.78, hh * 0.32);
      for (let k = 0; k < 3; k++) dt += `M${N(cx + (r() - 0.5) * w * 1.3)} ${N(cy + r() * hh * 0.7)}h${N(Math.max(0.6, h * 0.02))}`;
    }
    B.add('fl-c', d); B.add('fl-sh', sh); B.add('fl-hi', hi); B.add('fl-dt', dt); B.add('fl-snow', sn);
  }

  // 古柏（天坛）：干粗而扭、下段露出；冠是几团"云头"叶团错落挂在枝上，轮廓不规整
  function oldCypress(B, x, y, h, r) {
    const lean = (r() - 0.5) * h * 0.14, w = h * 0.042, L = [], Rt = [];
    for (let i = 0; i <= 6; i++) { const t = i / 6, cx = x + lean * t + Math.sin(t * 5 + r() * 2) * h * 0.012, ww = w * (1 - t * 0.62); L.push([cx - ww, y - h * 0.66 * t]); Rt.push([cx + ww, y - h * 0.66 * t]); }
    B.add('fl-trunk', 'M' + L.concat(Rt.reverse()).map(([a, b]) => `${N(a)} ${N(b)}`).join('L') + 'Z');
    let gr = ''; for (let k = 0; k < 3; k++) { const o = (k - 1) * w * 0.5; gr += `M${N(x + o)} ${N(y)}Q${N(x + o + lean * 0.3 + (r() - 0.5) * w)} ${N(y - h * 0.3)} ${N(x + lean * 0.6 + o * 0.4)} ${N(y - h * 0.56)}`; }
    B.add('fl-bark', gr);
    const n = 6 + Math.floor(r() * 3);
    let d = '', sh = '', hi = '', dt = '', sn = '', lb = '';
    for (let i = 0; i < n; i++) {
      const t = i / (n - 1), side = i % 2 ? 1 : -1;
      const cy = y - h * (0.36 + t * 0.56) + (r() - 0.5) * h * 0.04;
      const cx = x + lean * (0.5 + t * 0.5) + side * (1 - t * 0.6) * h * 0.13 * (0.3 + r() * 0.7);
      const rw = h * (0.13 - t * 0.05) * (0.85 + r() * 0.35), rh = rw * 0.62;
      lb += `M${N(x + lean * (0.4 + t * 0.45))} ${N(cy + rh * 0.9)}Q${N((x + lean + cx) / 2)} ${N(cy + rh * 0.3)} ${N(cx)} ${N(cy + rh * 0.4)}`;
      d += ell(cx - rw * 0.5, cy + rh * 0.12, rw * 0.56, rh * 0.68) + ell(cx, cy - rh * 0.1, rw * 0.7, rh * 0.9) + ell(cx + rw * 0.52, cy + rh * 0.12, rw * 0.54, rh * 0.66);
      sh += ell(cx + rw * 0.06, cy + rh * 0.4, rw * 0.86, rh * 0.38);
      hi += ell(cx - rw * 0.28, cy - rh * 0.42, rw * 0.38, rh * 0.28);
      sn += cap(cx, cy - rh * 0.7, rw * 0.62, rh * 0.3);
      for (let k = 0; k < 5; k++) dt += `M${N(cx + (r() - 0.5) * rw * 1.6)} ${N(cy + (r() - 0.3) * rh * 1.1)}h${N(Math.max(0.6, h * 0.014))}`;
    }
    B.add('fl-limb', lb); B.add('fl-c', d); B.add('fl-sh', sh); B.add('fl-hi', hi); B.add('fl-dt', dt); B.add('fl-snow', sn);
  }

  function tree(B, x, y, h, kind, r) {
    if (kind === 'p') pine(B, x, y, h, r);
    else if (kind === 'c') cypress(B, x, y, h, r);
    else if (kind === 'oc') oldCypress(B, x, y, h, r);
    else broad(B, x, y, h, kind, r);
  }

  // 按权重抽树种
  function picker(mix, r) {
    const ks = Object.entries(mix), tot = ks.reduce((s, [, w]) => s + w, 0);
    return () => { let t = r() * tot; for (const [k, w] of ks) if ((t -= w) < 0) return k; return ks[0][0]; };
  }

  /* 成片的林：沿山坡分几排，排里按"丛"换树种，丛与丛之间偶尔留空；越靠前越大。
     o: { x0, x1, top(x)→山脊 y, base, rows, hFar, hNear, mix, seed, skip(x,y,h), gap, f0, f1, jit } */
  function forest(P, o) {
    const r = rng(o.seed || 1), rows = o.rows || 6, pick = picker(o.mix, r);
    for (let row = 0; row < rows; row++) {
      const f = rows === 1 ? 1 : row / (rows - 1), B = Bucket();
      let x = o.x0 + r() * 12, kind = pick(), left = 2 + Math.floor(r() * 5);
      while (x < o.x1) {
        const ty = o.top(x);
        const h = (o.hFar + (o.hNear - o.hFar) * f) * (0.72 + r() * 0.56);
        if (ty != null) {
          const y = ty + (o.base - ty) * ((o.f0 ?? 0.04) + f * (o.f1 ?? 0.9)) + (r() - 0.5) * (o.jit ?? 8);
          if (y < o.base + 2 && !(o.skip && o.skip(x, y, h))) tree(B, x, y, h, kind, r);
        }
        x += h * (0.3 + r() * 0.32);
        if (--left <= 0) { kind = pick(); left = 2 + Math.floor(r() * 5); if (r() < (o.gap ?? 0.3)) x += h * (0.5 + r() * 1.1); }
      }
      B.flush(P);
    }
  }

  /* 一丛树（岸边、建筑旁）：沿一条线前后两三排，大小错落。
     o: { x0, x1, y, h, mix, seed, rows, dense } */
  function clump(P, o) {
    const r = rng(o.seed || 1), rows = o.rows || 2, pick = picker(o.mix, r), items = [];
    for (let row = 0; row < rows; row++) {
      let kind = pick(), left = 1 + Math.floor(r() * 3);
      const hr = o.h * (1.12 - row * 0.16);
      for (let x = o.x0 + r() * o.h * 0.3; x < o.x1; x += hr * (o.dense ?? 0.34) * (0.7 + r() * 0.6)) {
        if (r() < 0.07) { x += hr * 0.5; continue; }
        items.push([x, o.y + row * o.h * 0.1 + (r() - 0.5) * o.h * 0.08, hr * (0.7 + r() * 0.5), kind, row]);
        if (--left <= 0) { kind = pick(); left = 1 + Math.floor(r() * 3); }
      }
    }
    for (let row = 0; row < rows; row++) {
      const B = Bucket();
      items.filter(it => it[4] === row).forEach(([x, y, h, k]) => tree(B, x, y, h, k, r));
      B.flush(P);
    }
    if (o.under) canopy(P, { x0: o.x0 - o.h * 0.2, x1: o.x1 + o.h * 0.2, top: () => o.y + o.h * 0.16, base: o.y + o.h * 0.16, rows: 1, rFar: o.h * 0.15, rNear: o.h * 0.15, f0: 0, f1: 0, D: 0.35, mix: o.underMix || { e: 2, g: 2, a: 0.6, y: 0.4 }, seed: (o.seed || 1) + 7, patches: 12 });
  }

  /* 成片林冠（山上密林）：每排是一串相叠的树冠穹顶，同色合成一条路径；穹顶下半压暗，
     冠顶左上一笔亮边、点两三点墨。偶尔有高树探出林冠。比一棵棵画省得多，也更像成片的林。
     o: { x0, x1, top(x)→山脊 y, base, rows, rFar, rNear, mix, seed, skip(x,y,R), f0, f1, emerge } */
  const SHAPE = { e: [1, 1, 1.3], g: [1, 0.95, 1.3], a: [0.95, 1.05, 1.25], y: [0.72, 1.4, 1.35], c: [0.48, 1.85, 1.55], p: [1.25, 0.62, 1.2] };
  function canopy(P, o) {
    const r = rng(o.seed || 1), rows = o.rows || 12, pick = picker(o.mix, r);
    // 树种成片：撒若干片心，每个穹顶取最近片心的树种（横向拉长，顺着等高线），偶尔夹一棵别的
    let yMin = Infinity; for (let x = o.x0; x <= o.x1; x += 8) { const t = o.top(x); if (t != null) yMin = Math.min(yMin, t); }
    const seeds = Array.from({ length: o.patches || 90 }, () => [o.x0 + r() * (o.x1 - o.x0), yMin + r() * (o.base - yMin), pick()]);
    const kindAt = (x, y) => {
      if (r() < 0.16) return pick();
      let best = Infinity, k = seeds[0][2];
      for (const [sx, sy, sk] of seeds) { const d = (x - sx) ** 2 + ((y - sy) * 1.8) ** 2; if (d < best) { best = d; k = sk; } }
      return k;
    };
    for (let row = 0; row < rows; row++) {
      const f = rows === 1 ? 1 : row / (rows - 1), R = o.rFar + (o.rNear - o.rFar) * f;
      const B = Bucket();
      let x = o.x0 + r() * R;
      while (x < o.x1) {
        const t = o.top(x);
        const y = t == null ? null : t + (o.base - t) * ((o.f0 ?? 0.03) + f * (o.f1 ?? 0.95)) + (r() - 0.5) * R * 0.45;
        const kind = y == null ? 'e' : kindAt(x, y), [sw, shh, step] = SHAPE[kind];
        const w = R * sw * (0.82 + r() * 0.36), hb = R * shh * (0.82 + r() * 0.36);
        if (y != null && y < o.base + R && !(o.skip && o.skip(x, y, R))) {
          const dec = DEC[kind] ? ' fl-dec' : '', D = R * (o.D ?? 2);
          if (o.emerge && r() < o.emerge) tree(B, x + (r() - 0.5) * w, y - hb * 0.1, R * (3 + r() * 1.4), 'p', r);
          let d = dome(x, y, w, hb, D);
          if (kind === 'p') {
            const w2 = w * 0.62, h2 = hb * 0.9, x2 = x + w * (r() - 0.5) * 0.5, y2 = y - hb * 0.72;
            d += dome(x2, y2, w2, h2, h2);
            B.add('fl-sn', `M${M(x2 - w2 * 0.8)} ${M(y2 - h2 * 0.55)}Q${M(x2)} ${M(y2 - h2 * 1.2)} ${M(x2 + w2 * 0.8)} ${M(y2 - h2 * 0.55)}`);
          }
          B.add('fl-' + kind + ' fl-cnp' + (dec ? ' fl-w' : ''), d);
          B.add('fl-sh', `M${M(x - w)} ${M(y + D)}V${M(y - hb * 0.12)}Q${M(x)} ${M(y + hb * 0.3)} ${M(x + w)} ${M(y - hb * 0.12)}V${M(y + D)}Z`);
          B.add('fl-hl' + dec, `M${M(x - w * 0.62)} ${M(y - hb * 0.5)}Q${M(x - w * 0.42)} ${M(y - hb * 0.93)} ${M(x + w * 0.05)} ${M(y - hb * 0.9)}`);
          let dt = ''; const nd = 1 + Math.floor(r() * 3);
          for (let i = 0; i < nd; i++) dt += `M${M(x + (r() - 0.5) * w * 1.2)} ${M(y - hb * (0.15 + r() * 0.55))}h${M(R * 0.16)}`;
          B.add((kind === 'a' || kind === 'y' ? 'fl-dta' : 'fl-dt') + dec, dt);
          if (!dec) B.add('fl-sn', `M${M(x - w * 0.8)} ${M(y - hb * 0.55)}Q${M(x)} ${M(y - hb * 1.2)} ${M(x + w * 0.8)} ${M(y - hb * 0.55)}`);
        }
        x += w * step * (0.85 + r() * 0.3);
      }
      B.flush(P);
    }
  }
  // 一个树冠穹顶：两侧竖直向下延伸 D（被下一排盖住）
  const M = v => Math.round(v * 10) / 10;
  const dome = (x, y, w, hb, D) => `M${M(x - w)} ${M(y + D)}V${M(y)}A${M(w)} ${M(hb)} 0 0 1 ${M(x + w)} ${M(y)}V${M(y + D)}Z`;

  /* 迎客松（卷首、卷尾框景）：干粗、斜出回转，鱼鳞皴；几条长枝平伸，其中一条特别长、伸向画心，
     枝上一层层松针云——每朵由三四把扇形松针叠成，下沿压暗，几根亮针提神。 */
  function bigPine(P, x, y, h, seed, lean = 1, o = {}) {
    const r = rng(seed), Lr = lean;
    const spine = o.spine ? o.spine.map(([u, v]) => [x + Lr * h * u, y - h * v]) :
      [[x, y], [x + Lr * h * 0.05, y - h * 0.14], [x + Lr * h * 0.06, y - h * 0.28], [x - Lr * h * 0.01, y - h * 0.43], [x + Lr * h * 0.05, y - h * 0.57], [x + Lr * h * 0.1, y - h * 0.7], [x + Lr * h * 0.06, y - h * 0.84], [x + Lr * h * 0.1, y - h]];
    const sm = pts => { let d = `M${N(pts[0][0])} ${N(pts[0][1])}`; for (let i = 0; i < pts.length - 1; i++) { const p0 = pts[i - 1] || pts[i], p1 = pts[i], p2 = pts[i + 1], p3 = pts[i + 2] || p2; d += `C${N(p1[0] + (p2[0] - p0[0]) / 6)} ${N(p1[1] + (p2[1] - p0[1]) / 6)} ${N(p2[0] - (p3[0] - p1[0]) / 6)} ${N(p2[1] - (p3[1] - p1[1]) / 6)} ${N(p2[0])} ${N(p2[1])}`; } return d; };
    const tw = t => h * (0.046 * (1 - t) + 0.01 * t);
    const at = t => { const f = t * (spine.length - 1), i = Math.min(spine.length - 2, Math.floor(f)), k = f - i; return [spine[i][0] + (spine[i + 1][0] - spine[i][0]) * k, spine[i][1] + (spine[i + 1][1] - spine[i][1]) * k]; };
    const left = [], right = [];
    for (let i = 0; i <= 28; i++) { const t = i / 28, [px, py] = at(t), w = tw(t) * (1 + Math.max(0, 0.12 - t) * 5); left.push([px - w, py]); right.push([px + w, py]); }
    const out = Pen2();
    // 枝：先画，根部藏在干里
    const limbs = o.limbs || [[0.33, -1, 0.34, 0.05], [0.46, 1, 0.66, 0.01], [0.58, -1, 0.3, -0.01], [0.7, 1, 0.32, -0.03], [0.82, -1, 0.22, -0.04], [0.93, 1, 0.17, -0.03]];
    let limbF = '', twig = '';
    const back = [], front = [];
    for (const [t, side, len, rise] of limbs) {
      const [sx, sy] = at(t), dir = side * Lr, ex = sx + dir * h * len, ey = sy + h * rise, w0 = tw(t) * 0.62;
      const mx1 = sx + (ex - sx) * 0.33, my1 = sy - h * 0.03, mx2 = sx + (ex - sx) * 0.7, my2 = ey + h * 0.015;
      limbF += `M${N(sx - dir * w0)} ${N(sy - w0)}C${N(mx1)} ${N(my1 - w0 * 0.7)} ${N(mx2)} ${N(my2 - w0 * 0.25)} ${N(ex)} ${N(ey)}C${N(mx2)} ${N(my2 + w0 * 0.25)} ${N(mx1)} ${N(my1 + w0 * 0.7)} ${N(sx - dir * w0)} ${N(sy + w0 * 1.2)}Z`;
      const span = Math.abs(ex - sx), nf = Math.max(2, Math.round(span / (h * 0.085)));
      for (let k = 0; k < nf; k++) {
        const f = 0.28 + 0.78 * (k + 0.5) / nf, cx = sx + (ex - sx) * Math.min(1.05, f), cy = sy + (ey - sy) * f - h * (0.006 + r() * 0.016);
        const R = h * (0.062 + r() * 0.02) * (1 - f * 0.12);
        // 细枝把松针云挂到枝上
        twig += `M${N(sx + (ex - sx) * f * 0.92)} ${N(sy + (ey - sy) * f * 0.92)}Q${N(cx - dir * R * 0.3)} ${N(cy + R * 0.1)} ${N(cx)} ${N(cy - R * 0.05)}`;
        (k % 2 ? back : front).push([cx, cy, R]);
        if (r() < 0.6) back.push([cx + dir * R * 0.5, cy - R * 0.55, R * 0.8]);
      }
    }
    const [tx, ty] = at(1); front.push([tx, ty + h * 0.012, h * 0.058]); back.push([tx - Lr * h * 0.025, ty - h * 0.03, h * 0.045]);
    out.p(limbF, 'limbF'); out.p(twig, 'twig');
    out.p(sm(left) + 'L' + sm(right.slice().reverse()).slice(1) + 'Z', 'trunk-g');
    // 鱼鳞皴：沿干一片片
    let sc = '', scl = '';
    for (let i = 0; i < 70; i++) { const t = 0.02 + r() * 0.86, [px, py] = at(t), w = tw(t); const u = (r() - 0.5) * w * 1.2, a = w * (0.3 + r() * 0.2); sc += `M${N(px + u - a)} ${N(py)}q${N(a)} ${N(-a * 0.9)} ${N(a * 2)} ${N(r() * 0.8)}`; if (r() < 0.3) scl += `M${N(px - w * 0.7)} ${N(py + a)}q${N(w * 0.1)} ${N(-a)} 0 ${N(-a * 2)}`; }
    out.p(sc, 'bark'); out.p(scl, 'bark-l');
    // 松针云：每朵是后排三把、前排四把扇形松针，前排亮、后排暗，下沿压暗，几根亮针
    let fb = '', ff = '', under = '', ink = '', lite = '', hl = '', snow = '';
    const fanAt = (fx, fy, fw, fh, front) => {
      const d = `M${N(fx - fw)} ${N(fy)}A${N(fw)} ${N(fh)} 0 0 1 ${N(fx + fw)} ${N(fy)}Z`;
      if (!front) { fb += d; return; }
      ff += d;
      under += `M${N(fx - fw * 0.95)} ${N(fy)}A${N(fw * 0.95)} ${N(fh * 0.3)} 0 0 1 ${N(fx + fw * 0.95)} ${N(fy)}Z`;
      hl += `M${N(fx - fw * 0.72)} ${N(fy - fh * 0.62)}Q${N(fx - fw * 0.1)} ${N(fy - fh * 1.08)} ${N(fx + fw * 0.6)} ${N(fy - fh * 0.74)}`;
      snow += `M${N(fx - fw * 0.82)} ${N(fy - fh * 0.5)}Q${N(fx)} ${N(fy - fh * 1.3)} ${N(fx + fw * 0.82)} ${N(fy - fh * 0.5)}`;
      const nn = 13;
      for (let q = 0; q <= nn; q++) { const a = Math.PI * (1.04 + 0.92 * q / nn) + (r() - 0.5) * 0.05, L2 = 0.8 + r() * 0.2; const seg = `M${N(fx)} ${N(fy + fh * 0.02)}L${N(fx + Math.cos(a) * fw * L2)} ${N(fy + Math.sin(a) * fh * L2)}`; if (q % 3 === 1 && a > Math.PI * 1.15 && a < Math.PI * 1.85) lite += seg; else ink += seg; }
    };
    const cloud = (cx, cy, R, isFront) => {
      const k = isFront ? 4 : 3;
      for (let j = 0; j < k; j++) { const o2 = j - (k - 1) / 2, fw = R * (0.5 + r() * 0.16) * (1 - Math.abs(o2) * 0.1), fh = fw * (0.74 + r() * 0.12); fanAt(cx + o2 * R * 0.5 + (r() - 0.5) * R * 0.12, cy - (isFront ? 0 : R * 0.28) + Math.abs(o2) * R * 0.06 + (r() - 0.5) * R * 0.08, fw, fh, isFront); }
    };
    back.forEach(([cx, cy, R]) => cloud(cx, cy, R, false));
    front.forEach(([cx, cy, R]) => { cloud(cx, cy - R * 0.24, R * 0.92, false); cloud(cx, cy, R, true); });
    out.p(fb, 'pine2-b'); out.p(ff, 'pine2'); out.p(under, 'pine2-u'); out.p(ink, 'needle'); out.p(lite, 'needle-l'); out.p(hl, 'pine2-hl'); out.p(snow, 'fl-sn');
    P.raw(out.s());
  }
  function Pen2() { const o = []; return { p(d, c) { if (d) o.push(`<path class="${c}" d="${d}"/>`); }, s: () => o.join('') }; }

  window.FLORA = { tree, forest, clump, canopy, bigPine, Bucket, ell, cap };
})();
