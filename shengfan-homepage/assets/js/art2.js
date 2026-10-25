/* 晟帆主页 · 青绿长卷图形库（第二版）
   画法：界画勾线 + 青绿设色。统一光源在左上；琉璃、宫墙、立柱、汉白玉都用渐变出体积；
   线分三档（主线、细部、发丝），用暖墨色。建筑在局部坐标里画（u 向右、v 向上，原点在底部中线），
   放置时套 translate(x y) scale(s -s)。山水、云、水、树直接在画面坐标里画（y 向下）。 */
(function () {
  'use strict';
  const N = v => Math.round(v * 100) / 100;
  const pt = p => N(p[0]) + ' ' + N(p[1]);
  const bez = (c, t) => { const u = 1 - t; return [0, 1].map(k => u * u * u * c[0][k] + 3 * u * u * t * c[1][k] + 3 * u * t * t * c[2][k] + t * t * t * c[3][k]); };
  const samp = (c, n = 40) => Array.from({ length: n + 1 }, (_, i) => bez(c, i / n));
  const yAt = (poly, x) => {
    for (let i = 1; i < poly.length; i++) {
      const a = poly[i - 1], b = poly[i];
      if ((x - a[0]) * (x - b[0]) <= 0 && a[0] !== b[0]) return a[1] + (b[1] - a[1]) * (x - a[0]) / (b[0] - a[0]);
    }
    return null;
  };
  const C = c => `C${pt(c[1])} ${pt(c[2])} ${pt(c[3])}`;
  const rev = c => [c[3], c[2], c[1], c[0]];
  const mc = c => c.map(p => [-p[0], p[1]]);
  const poly = (pts, close = true) => 'M' + pts.map(pt).join('L') + (close ? 'Z' : '');
  // 可复现的随机数
  function rng(seed) { let s = seed >>> 0 || 1; return () => (s = (s * 1664525 + 1013904223) >>> 0) / 4294967296; }

  function Pen() {
    const o = [];
    const api = {
      p(d, c, extra = '') { if (d) o.push(`<path class="${c}" d="${d}"${extra}/>`); return api; },
      r(x, y, w, h, c, extra = '') { o.push(`<rect class="${c}" x="${N(x)}" y="${N(y)}" width="${N(w)}" height="${N(h)}"${extra}/>`); return api; },
      c(x, y, r, c, extra = '') { o.push(`<circle class="${c}" cx="${N(x)}" cy="${N(y)}" r="${N(r)}"${extra}/>`); return api; },
      e(x, y, rx, ry, c, extra = '') { o.push(`<ellipse class="${c}" cx="${N(x)}" cy="${N(y)}" rx="${N(rx)}" ry="${N(ry)}"${extra}/>`); return api; },
      g(inner, attrs = '') { o.push(`<g${attrs}>${inner}</g>`); return api; },
      raw(s) { o.push(s); return api; },
      s() { return o.join(''); },
    };
    return api;
  }
  const sub = fn => { const Q = Pen(); fn(Q); return Q.s(); };

  /* ── 渐变与图案（页面里放一次） ──────────────────────
     建筑在翻转坐标里画，所以建筑用的竖向渐变 offset 0 是视觉上的"下"。 */
  function lin(id, stops, x1 = 0, y1 = 0, x2 = 0, y2 = 1) {
    return `<linearGradient id="${id}" x1="${x1}" y1="${y1}" x2="${x2}" y2="${y2}">${stops.map(([o, c, a = 1]) => `<stop offset="${o}" stop-color="${c}"${a < 1 ? ` stop-opacity="${a}"` : ''}/>`).join('')}</linearGradient>`;
  }
  function rad(id, stops, cx = .5, cy = .5, r = .5) {
    return `<radialGradient id="${id}" cx="${cx}" cy="${cy}" r="${r}">${stops.map(([o, c, a = 1]) => `<stop offset="${o}" stop-color="${c}"${a < 1 ? ` stop-opacity="${a}"` : ''}/>`).join('')}</radialGradient>`;
  }
  function defs() {
    return '<svg width="0" height="0" style="position:absolute" aria-hidden="true"><defs>' +
      // 建筑（翻转坐标：offset 0 在下）
      lin('g-roofY', [[0, '#b48231'], [.18, '#d7a443'], [.7, '#ecc368'], [1, '#f5d88a']]) +
      lin('g-roofB', [[0, '#1f3f78'], [.2, '#2f5ea6'], [.75, '#5e88c2'], [1, '#86a9d6']]) +
      lin('g-roofG', [[0, '#6f7672'], [.25, '#8c928d'], [1, '#c4c7c1']]) +
      lin('g-roofGr', [[0, '#2f6e52'], [1, '#6aa585']]) +
      // 冬：屋面积雪（翻转坐标，下沿露出一点琉璃色）
      lin('g-roofYW', [[0, '#c4964a'], [.16, '#e6d9bd'], [.42, '#f6f3ec'], [1, '#fdfcf8']]) +
      lin('g-roofBW', [[0, '#3a5f96'], [.16, '#d6dde6'], [.42, '#f3f5f6'], [1, '#fdfdfc']]) +
      lin('g-roofGW', [[0, '#7c827e'], [.18, '#dcddd8'], [1, '#fbfbf8']]) +
      lin('g-eaveY', [[0, '#9c6c25'], [1, '#e2b252']]) +
      lin('g-wall', [[0, '#8f3427'], [.5, '#b0473a'], [1, '#bf5644']]) +
      lin('g-cwall', [[0, '#73291f'], [.45, '#8e382b'], [1, '#9c4232']]) +
      lin('g-cwallY', [[0, '#9c4232'], [.55, '#8e382b'], [1, '#73291f']]) +
      lin('g-col', [[0, '#d66b55'], [.35, '#b2432f'], [1, '#86281c']], 0, 0, 1, 0) +
      lin('g-door', [[0, '#7e2a1f'], [1, '#a13a2b']]) +
      lin('g-stone', [[0, '#e1dacb'], [.35, '#f3efe5'], [1, '#fffcf4']]) +
      lin('g-brick', [[0, '#8f8a80'], [1, '#bdb6a8']]) +
      lin('g-paintB', [[0, '#1f4f74'], [1, '#3b7598']]) +
      lin('g-paintG', [[0, '#2a6a52'], [1, '#4d917a']]) +
      lin('g-shade', [[0, '#2e2b26', .0], [1, '#2e2b26', .32]]) +
      lin('g-glass', [[0, '#7f9bb0'], [1, '#c3d3dc']]) +
      lin('g-ashlar', [[0, '#968e7f'], [.55, '#b3ab9a'], [1, '#c8c1b0']]) +
      lin('g-facade', [[0, '#d3c29d'], [.6, '#e6d8b8'], [1, '#efe4c9']]) +
      lin('g-recess', [[0, '#4d4237'], [1, '#6e6151']]) +
      lin('g-gold', [[0, '#9a7428'], [.5, '#d8b25a'], [1, '#f4dc98']], 0, 0, 1, 0) +
      // 圆屋面的体积：左亮右暗
      lin('g-vol', [[0, '#fff', .26], [.28, '#fff', .05], [.55, '#000', 0], [1, '#14203a', .3]], 0, 0, 1, 0) +
      // 山水（画面坐标：offset 0 在上）
      lin('g-sky', [[0, '#efe6d3'], [.55, '#f1e7d2'], [.86, '#efdfc1'], [1, '#ebdcbd']]) +
      rad('g-sun', [[0, '#f6d48c', .9], [.35, '#f3c873', .45], [1, '#f3c873', 0]]) +
      lin('g-mtnFar', [[0, '#8fb0bd'], [.55, '#b3c6c3'], [1, '#e6dcc6', .1]]) +
      lin('g-mtnMid', [[0, '#3f7c96'], [.32, '#5f9a8d'], [.7, '#9cb98e'], [1, '#d4b989']]) +
      lin('g-mtnNear', [[0, '#2f6b86'], [.28, '#4f8f78'], [.62, '#86ad78'], [1, '#c99e68']]) +
      lin('g-hill', [[0, '#6f9a7c'], [.6, '#a9bf8b'], [1, '#d7c391']]) +
      lin('g-hillW', [[0, '#e9ece8'], [.6, '#e2e2d8'], [1, '#d8cfbd']]) +
      lin('g-mtnFarW', [[0, '#e8ebea'], [.5, '#c9d3d4'], [1, '#e6dcc6', .1]]) +
      lin('g-mtnMidW', [[0, '#f3f2ee'], [.3, '#d9e0df'], [.72, '#a9b9b3'], [1, '#d4c7ad']]) +
      lin('g-mtnNearW', [[0, '#f5f4f0'], [.28, '#dde3e1'], [.66, '#9fb2aa'], [1, '#cdbc9c']]) +
      lin('g-water', [[0, '#c7d6d0'], [.5, '#d7e1d9'], [1, '#e6e5d6']]) +
      lin('g-waterW', [[0, '#dfe7ea'], [.5, '#ebf0ef'], [1, '#f1efe6']]) +
      lin('g-skyW', [[0, '#e6e5df'], [.6, '#ece8de'], [1, '#e9e3d4']]) +
      lin('g-mist', [[0, '#f4ecdb', 0], [.5, '#f4ecdb', .85], [1, '#f4ecdb', 0]]) +
      rad('g-mistR', [[0, '#f6efe0', .9], [.6, '#f6efe0', .4], [1, '#f6efe0', 0]]) +
      lin('g-ground', [[0, '#ddcfae'], [1, '#e9dfc6']]) +
      lin('g-trunk', [[0, '#7a5a40'], [.45, '#5b412f'], [1, '#3a2a1f']], 0, 0, 1, 0) +
      lin('g-reflect', [[0, '#fff', .55], [1, '#fff', 0]]) +
      '<mask id="m-reflect" maskContentUnits="objectBoundingBox"><rect width="1" height="1" fill="url(#g-reflect)"/></mask>' +
      // 菱花隔扇
      '<pattern id="p-lattice" width="1.8" height="1.8" patternUnits="userSpaceOnUse"><rect width="1.8" height="1.8" fill="#7a2a1f"/><path d="M.9 0 1.8 .9 .9 1.8 0 .9Z" fill="none" stroke="#c9944d" stroke-width=".22" opacity=".75"/></pattern>' +
      // 石台：错缝条石
      '<pattern id="p-ashlar" width="9" height="3.6" patternUnits="userSpaceOnUse"><path d="M0 0H9M0 1.8H9M0 0V1.8M4.5 1.8V3.6" fill="none" stroke="#625a4e" stroke-width=".2" opacity=".5"/></pattern>' +
      // 冬天落叶林：灰褐底上的细枝
      '<pattern id="p-twigs" width="16" height="12" patternUnits="userSpaceOnUse"><rect width="16" height="12" fill="#ad9f90"/><path d="M2 12 3.6 7 2.6 3M3.6 7 6.4 4.2 7 1.4M10 12 11.4 8 14.6 5.4M11.4 8 10.6 4.4 11.6 1.6M14.6 5.4 15.4 2.6M6.4 4.2 8.6 3" fill="none" stroke="#6e5c4d" stroke-width=".55" stroke-linecap="round" opacity=".75"/></pattern>' +
      // 砖缝
      '<pattern id="p-brick" width="6" height="2.4" patternUnits="userSpaceOnUse"><path d="M0 0H6M0 1.2H6M0 0V1.2M3 1.2V2.4" fill="none" stroke="#5d564b" stroke-width=".18" opacity=".55"/></pattern>' +
      '</defs></svg>';
  }

  /* ── 建筑构件 ─────────────────────────────────── */

  // 屋顶（正面或圆形攒尖）。ey 檐口中点高，ew 檐口半宽，rh 屋面高，rw 正脊半宽（0 为攒尖）
  // o.tone: 'Y' 黄琉璃 / 'B' 蓝琉璃 / 'G' 灰瓦；o.round 圆屋面；o.noRidge 上沿被遮；o.trim 绿剪边
  function roof(P, ey, ew, rh, rw, o = {}) {
    const round = !!o.round, tone = o.tone || 'Y';
    const lift = o.lift ?? ew * (round ? 0.05 : 0.075), band = o.band ?? Math.max(1.2, rh * 0.13), top = ey + rh;
    const Bt = [[-ew, ey + lift], [-ew * 0.5, ey - lift * 0.12], [ew * 0.5, ey - lift * 0.12], [ew, ey + lift]];
    const Tt = [[-ew * 0.985, ey + lift + band * 0.6], [-ew * 0.5, ey + band * 0.95], [ew * 0.5, ey + band * 0.95], [ew * 0.985, ey + lift + band * 0.6]];
    const R0 = Tt[3], dx = R0[0] - rw, dy = top - R0[1];
    const k = round ? [0.34, 0.12, 0.14, 0.5] : [0.5, 0.06, 0.1, 0.52];
    const RS = [R0, [R0[0] - dx * k[0], R0[1] + dy * k[1]], [rw + dx * k[2], R0[1] + dy * k[3]], [rw, top]], LS = mc(RS);
    // 檐下：深色檐底 + 一排椽头
    if (o.soffit !== false) {
      const sd = band * 1.5, bpts = samp(Bt, 30);
      const low = bpts.slice().reverse().map(([x, y]) => [x, y - sd * (1 - (Math.abs(x) / ew) ** 3)]);
      P.p(poly(bpts.concat(low)), 'f-soffit');
      let rf = ''; const n = Math.round(ew * 2 / (band * 1.1));
      const bs = samp(Bt);
      for (let i = 1; i < n; i++) { const u = -ew + i * 2 * ew / n; if (Math.abs(u) > ew * 0.94) continue; const y = yAt(bs, u); if (y != null) rf += `M${N(u)} ${N(y)}v${N(-band * 0.42)}`; }
      P.p(rf, 'f-rafter', ` stroke-width="${N(band * 0.36)}"`);
    }
    const shape = `M${pt(Bt[0])}${C(Bt)}L${pt(R0)}${C(RS)}L${pt(LS[3])}${C(rev(LS))}Z`;
    P.p(shape, `f-roof${tone} k`);
    if (round) P.p(shape, 'f-vol');
    // 檐口带（亮边，绿剪边时为绿色）
    P.p(`M${pt(Bt[0])}${C(Bt)}L${pt(Tt[3])}${C(rev(Tt))}Z`, o.trim ? 'f-roofGr kf' : `f-eave${tone} kf`);
    // 瓦垄与勾头
    const tp = samp(Tt), rp = samp(RS), lp = samp(LS), bp = samp(Bt);
    let dark = '', lite = '', heads = '';
    const headR = Math.min(band * 0.34, 0.9);
    if (round) {
      const n = o.tiles ?? Math.round(ew / 1.45);
      for (let i = 1; i < n; i++) {
        const f = Math.sin(-Math.PI / 2 + Math.PI * i / n);
        if (Math.abs(f) > 0.97) continue;
        const pts = rp.map(([x, y]) => [f * x, y]); const yb = yAt(tp, pts[0][0]); if (yb != null) pts[0] = [pts[0][0], yb];
        const d = 'M' + pts.filter((_, j) => j % 2 === 0 || j === pts.length - 1).map(pt).join('L');
        dark += d; lite += 'M' + pts.filter((_, j) => j % 2 === 0 || j === pts.length - 1).map(([x, y]) => pt([x + 0.35, y])).join('L');
        const hb = yAt(bp, pts[0][0]); if (hb != null && Math.abs(f) < 0.93) heads += `M${N(pts[0][0])} ${N(hb + headR * 0.2)}h.01`;
      }
    } else {
      const step = o.tile ?? Math.max(1.5, ew / 15);
      for (let u = -ew * 0.93 + step * 0.5; u < ew * 0.93; u += step) {
        const yb = yAt(tp, u); if (yb == null) continue;
        const yt = Math.abs(u) <= rw ? top : yAt(u > 0 ? rp : lp, u);
        if (yt == null || yt - yb < 1) continue;
        dark += `M${N(u)} ${N(yb)}L${N(u)} ${N(yt)}`; lite += `M${N(u + 0.4)} ${N(yb)}L${N(u + 0.4)} ${N(yt - 0.4)}`;
        const hb = yAt(bp, u); if (hb != null) heads += `M${N(u)} ${N(hb + headR * 0.2)}h.01`;
      }
    }
    P.p(dark, 'kt'); if (o.lite !== false) P.p(lite, 'kl'); P.p(heads, `f-head${tone}`, ` stroke-width="${N(headR * 2)}"`);
    // 翼角起翘
    const f = band * (o.flick ?? 1.7);
    if (!round) {
      for (const s of [1, -1]) {
        const x = s * ew;
        P.p(`M${N(x)} ${N(ey + lift)}Q${N(x + s * ew * 0.045)} ${N(ey + lift + f * 0.12)} ${N(x + s * (ew * 0.07 + 0.6))} ${N(ey + lift + f)}L${N(x + s * (ew * 0.055))} ${N(ey + lift + f * 0.75)}Q${N(x + s * ew * 0.02)} ${N(ey + lift + band * 0.5)} ${N(x - s * band * 0.6)} ${N(ey + lift + band * 0.7)}Z`, `f-eave${tone} k`);
      }
    }
    // 正脊、吻兽、垂脊上的脊兽
    if (rw > 0 && !o.noRidge) {
      const b = band * 0.95;
      P.r(-rw, top - b * 0.1, rw * 2, b, `f-eave${tone} k`);
      P.p(`M${N(-rw)} ${N(top + b * 0.45)}H${N(rw)}`, 'kl');
      if (o.ridge === 'xie') {
        for (const s of [1, -1]) P.p(`M${N(s * rw)} ${N(top + b * 0.6)}Q${N(s * (rw + b * 1.6))} ${N(top + b * 0.8)} ${N(s * (rw + b * 2.1))} ${N(top + b * 2.9)}`, 'k');
      } else if (o.ridge !== 'none') {
        const h = b * 2.8;
        for (const s of [1, -1]) {
          const x = s * rw;
          P.p(`M${N(x - s * b * 0.1)} ${N(top + b * 0.8)}C${N(x + s * b * 0.6)} ${N(top + b + h * 0.55)} ${N(x - s * b * 0.5)} ${N(top + b + h)} ${N(x - s * b * 1.7)} ${N(top + b + h * 0.72)}C${N(x - s * b * 1.1)} ${N(top + b + h * 0.5)} ${N(x - s * b * 1.2)} ${N(top + b * 0.9)} ${N(x - s * b * 0.6)} ${N(top + b * 0.8)}Z`, `f-eave${tone} k`);
        }
      }
      // 垂脊：从正脊端头斜下到翼角，脊兽是一串小圆点
      if (o.hips !== false) {
        for (const s of [1, -1]) {
          const a = [s * rw, top], z = [s * (ew * 0.86), ey + lift + band * 0.9];
          P.p(`M${pt(a)}Q${N(s * (rw + (ew - rw) * 0.35))} ${N(top - rh * 0.18)} ${pt(z)}`, 'kf');
          for (let i = 1; i <= 4; i++) { const t = 0.68 + i * 0.065, x = a[0] + (z[0] - a[0]) * t, y = a[1] + (z[1] - a[1]) * t + rh * 0.05; P.c(x, y + band * 0.35, band * 0.28, `f-eave${tone} kf`); }
        }
      }
    }
    return { top, band };
  }

  // 斗拱带：檐下一排斗拱，青绿彩画
  function dougong(P, y0, h, hw, step) {
    const n = Math.max(4, Math.round(hw * 2 / step));
    let a = '', b = '';
    for (let i = 0; i <= n; i++) {
      const u = -hw + i * 2 * hw / n, w = step * 0.42;
      a += `M${N(u - w * 0.35)} ${N(y0)}h${N(w * 0.7)}v${N(h * 0.35)}h${N(-w * 0.7)}Z`;          // 坐斗
      b += `M${N(u - w)} ${N(y0 + h * 0.35)}h${N(w * 2)}v${N(h * 0.22)}h${N(-w * 2)}Z`;             // 横拱
      a += `M${N(u - w * 0.3)} ${N(y0 + h * 0.57)}h${N(w * 0.6)}v${N(h * 0.43)}h${N(-w * 0.6)}Z`;    // 升
    }
    P.r(-hw, y0, hw * 2, h, 'f-paintB');
    P.p(b, 'f-paintG kt'); P.p(a, 'f-gold kt');
  }

  // 额枋彩画：青绿相间的箍头、方心，金线勾边
  function caihua(P, y0, h, hw) {
    P.r(-hw, y0, hw * 2, h, 'f-paintG kf');
    const n = Math.max(3, Math.round(hw / (h * 3.2)));
    let blue = '', gold = '';
    for (let i = 0; i < n; i++) {
      const a = -hw + (2 * hw) * i / n, b = -hw + (2 * hw) * (i + 1) / n, m = (a + b) / 2, w = (b - a) * 0.3;
      blue += `M${N(m - w)} ${N(y0 + h * 0.18)}L${N(m - w * 0.7)} ${N(y0 + h * 0.5)}L${N(m - w)} ${N(y0 + h * 0.82)}H${N(m + w)}L${N(m + w * 0.7)} ${N(y0 + h * 0.5)}L${N(m + w)} ${N(y0 + h * 0.18)}Z`;
      gold += `M${N(a + 0.4)} ${N(y0)}V${N(y0 + h)}`;
    }
    P.p(blue, 'f-paintB'); P.p(gold, 'kg'); P.p(`M${N(-hw)} ${N(y0 + h * 0.5)}H${N(hw)}`, 'kgt');
  }

  // 屋身：红柱（明暗）、菱花隔扇、门钉、额枋彩画
  function body(P, y0, y1, hw, o = {}) {
    const cols = o.cols ?? 6, cw = o.cw ?? Math.max(1.2, hw * 0.045), h = y1 - y0;
    P.r(-hw, y0, hw * 2, h, `${o.wall || 'f-wallIn'} k`);
    if (o.lattice !== false) {
      for (let i = 0; i < cols - 1; i++) {
        const a = -hw + (2 * hw) * i / (cols - 1) + cw * 0.7, b = -hw + (2 * hw) * (i + 1) / (cols - 1) - cw * 0.7;
        const lt = y0 + h * 0.06, lh = h * (o.lat ?? 0.7);
        P.r(a, lt, b - a, lh, 'f-lattice kf');
        P.p(`M${N(a)} ${N(lt + lh * 0.3)}H${N(b)}M${N((a + b) / 2)} ${N(lt)}V${N(lt + lh)}`, 'kt');
      }
    }
    let cd = '';
    for (let i = 0; i < cols; i++) { const u = -hw + (2 * hw) * i / (cols - 1); cd += `M${N(u - cw / 2)} ${N(y0)}h${N(cw)}v${N(h)}h${N(-cw)}Z`; }
    P.p(cd, 'f-col kf');
    const lb = o.lintel ?? Math.min(3, h * 0.17);
    if (o.paint !== false) caihua(P, y1 - lb, lb, hw); else P.r(-hw, y1 - lb, hw * 2, lb, 'f-paintG kf');
    if (o.dougong) dougong(P, y1, o.dgH ?? lb * 0.9, hw, o.dougong);
  }

  // 汉白玉台基：须弥座线脚、栏板、望柱、螭首
  function terrace(P, y0, h, hw, o = {}) {
    const bat = o.bat ?? 0, top = y0 + h, w = hw - bat;
    P.p(poly([[-hw, y0], [hw, y0], [w, top], [-w, top]]), 'f-stone k');
    // 须弥座：上下枭混
    P.p(poly([[-hw, y0], [hw, y0], [hw - bat * 0.2, y0 + h * 0.22], [-hw + bat * 0.2, y0 + h * 0.22]]), 'f-stoneD kt');
    P.p(`M${N(-w)} ${N(top - h * 0.2)}H${N(w)}`, 'kt');
    if (o.chishou) { let d = ''; const n = Math.round(w * 2 / o.chishou); for (let i = 0; i <= n; i++) { const u = -w + i * 2 * w / n; d += `M${N(u - 0.6)} ${N(top - h * 0.12)}h1.2l-.2 -1.1h-.8Z`; } P.p(d, 'f-stoneD kt'); }
    if (o.rail) {
      const rh = o.rail, sp = o.post ?? 5, n = Math.max(2, Math.round(w * 2 / sp));
      let panels = '', posts = '', caps = '';
      for (let i = 0; i < n; i++) {
        const a = -w + i * 2 * w / n, b = -w + (i + 1) * 2 * w / n;
        panels += `M${N(a + 0.5)} ${N(top + rh * 0.12)}H${N(b - 0.5)}V${N(top + rh * 0.7)}H${N(a + 0.5)}Z`;
      }
      for (let i = 0; i <= n; i++) {
        const u = -w + i * 2 * w / n;
        posts += `M${N(u - 0.45)} ${N(top)}h0.9v${N(rh)}h-0.9Z`;
        caps += `M${N(u - 0.6)} ${N(top + rh)}Q${N(u - 0.7)} ${N(top + rh + 1.2)} ${N(u)} ${N(top + rh + 1.5)}Q${N(u + 0.7)} ${N(top + rh + 1.2)} ${N(u + 0.6)} ${N(top + rh)}Z`;
      }
      P.r(-w, top, w * 2, rh * 0.78, 'f-stone kt');
      P.p(panels, 'f-stoneD kt');
      P.p(`M${N(-w)} ${N(top + rh * 0.78)}H${N(w)}`, 'kt');
      P.p(posts, 'f-stone kt'); P.p(caps, 'f-stone kt');
    }
  }

  // 台阶，中间丹陛石刻云纹
  function stairs(P, y0, y1, w0, w1, n, o = {}) {
    P.p(poly([[-w0, y0], [w0, y0], [w1, y1], [-w1, y1]]), 'f-stone k');
    let d = '';
    for (let i = 1; i < n; i++) { const t = i / n, yy = y0 + (y1 - y0) * t, ww = w0 + (w1 - w0) * t; d += `M${N(-ww)} ${N(yy)}H${N(ww)}`; }
    P.p(d, 'kt');
    if (o.ramp) {
      const r = o.ramp;
      P.p(poly([[-r, y0], [r, y0], [r * 0.85, y1], [-r * 0.85, y1]]), 'f-stoneD kt');
      let c = ''; for (let i = 0; i < 4; i++) { const y = y0 + (y1 - y0) * (0.15 + i * 0.22); c += `M${N(-r * 0.6)} ${N(y)}c${N(r * 0.3)} ${N(1.4)} ${N(r * 0.5)} ${N(-1.2)} ${N(r * 0.6)} 0s${N(r * 0.4)} ${N(1.2)} ${N(r * 0.6)} 0`; }
      P.p(c, 'kt');
    }
  }

  // 鎏金宝顶
  function baoding(P, y0, s) {
    P.p(poly([[-s * 1.6, y0], [s * 1.6, y0], [s * 1.1, y0 + s * 0.8], [s * 1.35, y0 + s * 1.15], [-s * 1.35, y0 + s * 1.15], [-s * 1.1, y0 + s * 0.8]]), 'f-gold k');
    P.p(`M${N(-s * 0.55)} ${N(y0 + s * 1.15)}C${N(-s * 1.4)} ${N(y0 + s * 1.9)} ${N(-s * 1.25)} ${N(y0 + s * 3.4)} 0 ${N(y0 + s * 3.75)}C${N(s * 1.25)} ${N(y0 + s * 3.4)} ${N(s * 1.4)} ${N(y0 + s * 1.9)} ${N(s * 0.55)} ${N(y0 + s * 1.15)}Z`, 'f-gold k');
    P.p(`M${N(-s * 0.35)} ${N(y0 + s * 3.65)}Q0 ${N(y0 + s * 4.3)} ${N(s * 0.35)} ${N(y0 + s * 3.65)}L0 ${N(y0 + s * 6.2)}Z`, 'f-gold k');
    P.p(`M${N(-s * 0.5)} ${N(y0 + s * 2.3)}Q${N(-s * 0.2)} ${N(y0 + s * 3)} ${N(s * 0.2)} ${N(y0 + s * 3.2)}`, 'kl');
  }

  // 歇山山面（角楼用）
  function xsGable(P, ey, ew, h, o = {}) {
    const lift = o.lift ?? ew * 0.13, band = o.band ?? Math.max(0.9, h * 0.06), apex = ey + h;
    const Bt = [[-ew, ey + lift], [-ew * 0.45, ey - lift * 0.12], [ew * 0.45, ey - lift * 0.12], [ew, ey + lift]];
    const Tt = [[-ew * 0.97, ey + lift + band * 0.6], [-ew * 0.45, ey + band * 0.95], [ew * 0.45, ey + band * 0.95], [ew * 0.97, ey + lift + band * 0.6]];
    const R0 = Tt[3], RS = [R0, [ew * 0.58, ey + lift + h * 0.1], [ew * 0.2, ey + h * 0.6], [0, apex]], LS = mc(RS);
    if (o.soffit !== false) { const bpts = samp(Bt, 24), sd = band * 1.6; P.p(poly(bpts.concat(bpts.slice().reverse().map(([x, y]) => [x, y - sd * (1 - (Math.abs(x) / ew) ** 3)]))), 'f-soffit'); }
    P.p(`M${pt(Bt[0])}${C(Bt)}L${pt(R0)}${C(RS)}${C(rev(LS))}Z`, 'f-roofY k');
    P.p(`M${pt(Bt[0])}${C(Bt)}L${pt(Tt[3])}${C(rev(Tt))}Z`, 'f-eaveY kf');
    const gb = ey + h * (o.gy ?? 0.44), gw = ew * (o.gw ?? 0.34), ga = apex - h * 0.07, bw = band * 1.1;
    const tp = samp(Tt), rp = samp(RS), lp = samp(LS), bp = samp(Bt), step = o.tile ?? Math.max(1.3, ew / 12);
    let td = '', tl = '', heads = '';
    for (let u = -ew * 0.9 + step / 2; u < ew * 0.9; u += step) {
      const yb = yAt(tp, u); let yt = yAt(u > 0 ? rp : lp, u);
      if (yb == null || yt == null) continue;
      if (Math.abs(u) < gw + bw) yt = Math.min(yt, gb);
      if (yt - yb > 0.8) { td += `M${N(u)} ${N(yb)}L${N(u)} ${N(yt)}`; tl += `M${N(u + 0.35)} ${N(yb)}L${N(u + 0.35)} ${N(yt - 0.3)}`; }
      const hb = yAt(bp, u); if (hb != null) heads += `M${N(u)} ${N(hb + 0.1)}h.01`;
    }
    P.p(td, 'kt'); P.p(tl, 'kl'); P.p(heads, 'f-headY', ' stroke-width="1"');
    P.p(poly([[-gw, gb], [0, ga], [gw, gb]]), 'f-gable kf');
    // 山花里的金色绶带纹
    P.p(`M${N(-gw * 0.6)} ${N(gb + (ga - gb) * 0.25)}Q0 ${N(gb + (ga - gb) * 0.55)} ${N(gw * 0.6)} ${N(gb + (ga - gb) * 0.25)}`, 'kg');
    P.p(`M${N(-gw - bw)} ${N(gb - bw * 0.25)}L0 ${N(ga + bw * 1.35)}L${N(gw + bw)} ${N(gb - bw * 0.25)}L${N(gw)} ${N(gb)}L0 ${N(ga)}L${N(-gw)} ${N(gb)}Z`, 'f-eaveY k');
    P.p(`M0 ${N(ga - 0.2)}C${N(bw * 0.9)} ${N(ga - bw * 1.4)} ${N(bw * 0.5)} ${N(ga - bw * 2.6)} 0 ${N(ga - bw * 3)}C${N(-bw * 0.5)} ${N(ga - bw * 2.6)} ${N(-bw * 0.9)} ${N(ga - bw * 1.4)} 0 ${N(ga - 0.2)}Z`, 'f-gold kf');
    P.p(`M${N(-gw - bw * 1.6)} ${N(gb)}H${N(gw + bw * 1.6)}`, 'kf');
    const f = band * (o.flick ?? 2.6);
    for (const s of [1, -1]) {
      const x = s * ew;
      P.p(`M${N(x)} ${N(ey + lift)}Q${N(x + s * ew * 0.05)} ${N(ey + lift + f * 0.15)} ${N(x + s * (ew * 0.09 + 0.5))} ${N(ey + lift + f)}L${N(x + s * ew * 0.07)} ${N(ey + lift + f * 0.72)}Q${N(x + s * ew * 0.02)} ${N(ey + lift + band * 0.5)} ${N(x - s * band * 0.6)} ${N(ey + lift + band * 0.7)}Z`, 'f-eaveY k');
    }
    return { apex };
  }

  /* ── 地标（局部坐标）─────────────────────────────── */

  // 天坛祈年殿：三重檐蓝琉璃圆攒尖，三层汉白玉祈谷坛
  function qinian() {
    const P = Pen();
    body(P, 21, 39.2, 33, { cols: 9, lat: 0.74, lintel: 2.4 });
    dougong(P, 39.2, 2.8, 33.8, 2.3);
    roof(P, 42, 51, 13.5, 28, { tone: 'B', round: true, noRidge: true, lift: 3, band: 1.8, tiles: 46, shadeW: 34 });
    P.r(-28, 55.5, 56, 4.8, 'f-wallIn k'); caihua(P, 55.5, 2.4, 28); dougong(P, 57.9, 2.4, 28, 2.1);
    roof(P, 60.3, 40.5, 12, 20.5, { tone: 'B', round: true, noRidge: true, lift: 2.6, band: 1.6, tiles: 38, shadeW: 28 });
    P.r(-20.5, 72.3, 41, 4.3, 'f-wallIn k'); caihua(P, 72.3, 2, 20.5); dougong(P, 74.3, 2.3, 20.5, 2);
    P.r(-2.6, 71.6, 5.2, 7.4, 'f-paintB k'); P.r(-1.8, 72.4, 3.6, 5.8, 'kg');
    P.p('M-.9 73.4v3.6M.9 73.4v3.6', 'kgt');
    roof(P, 76.6, 30, 19.5, 0, { tone: 'B', round: true, lift: 2.2, band: 1.5, tiles: 30, shadeW: 20.5 });
    baoding(P, 96, 1.4);
    terrace(P, 14, 7, 60, { rail: 3.2, post: 4.2, bat: 1, chishou: 7 });
    terrace(P, 7, 7, 72, { rail: 3.2, post: 4.2, bat: 1, chishou: 7 });
    terrace(P, 0, 7, 84, { rail: 3.2, post: 4.2, bat: 1, chishou: 7 });
    stairs(P, 0, 21, 10, 7.6, 15, { ramp: 3.6 });
    return { g: P.s(), box: [-86, 86, 0, 106] };
  }

  // 故宫角楼：转角居中、两面对称展开；上层两山花加十字脊宝顶，中层两大山花，下层四个抱厦
  function jiaolou(o = {}) {
    const P = Pen(), b = o.wallH ?? 24;
    const at = (x, fn) => P.g(sub(fn), ` transform="translate(${x} 0)"`);
    body(P, b + 42, b + 49, 21, { cols: 7, lattice: false, lintel: 1.4 });
    P.r(-10.5, b + 49, 21, 21, 'f-roofY k');
    let vt = ''; for (let u = -9.8; u <= 9.8; u += 1.4) vt += `M${N(u)} ${N(b + 49)}V${N(b + 69.6)}`; P.p(vt, 'kt');
    for (const x of [-10.5, 10.5]) at(x, Q => xsGable(Q, b + 48, 17, 22, { gw: 0.38, tile: 1.2, flick: 2.8 }));
    P.r(-11, b + 69.4, 22, 1.5, 'f-eaveY k');
    baoding(P, b + 70.9, 1.15);
    body(P, b + 26, b + 34, 37, { cols: 9, lat: 0.62, lintel: 1.6 });
    dougong(P, b + 34, 1.4, 37, 1.9);
    for (const x of [-19.5, 19.5]) at(x, Q => xsGable(Q, b + 23.5, 29, 24.5, { tile: 1.4, flick: 2.8 }));
    roof(P, b + 23, 13, 4.4, 0, { noRidge: true, lift: 2.6, band: 1.1, soffit: false, tile: 1.4 });
    body(P, b, b + 11, 47, { cols: 13, lat: 0.64, lintel: 1.6 });
    dougong(P, b + 11, 1.3, 47, 1.9);
    roof(P, b + 12, 63, 6, 52, { lift: 3.8, band: 1.4, tile: 1.8, noRidge: true, shadeW: 47 });
    for (const x of [-42, -14, 14, 42]) at(x, Q => xsGable(Q, b + 13.6, 13.5, 14.5, { tile: 1.2, flick: 2.3 }));
    // 城墙：栗红色，墙根一道条石，顶上垛口；墙身略收分
    P.p(poly([[-96, 0], [96, 0], [92, b], [-92, b]]), 'f-cwall k');
    P.p(poly([[-96, 0], [96, 0], [95.6, 2.2], [-95.6, 2.2]]), 'f-stoneD kt');
    P.p(poly([[60, 0], [96, 0], [92, b], [60, b]]), 'f-shadeL');
    P.p(`M-93.6 ${N(b * 0.55)}H93.6`, 'kt');
    let cr = `M-92 ${b}H92V${b + 1.6}H-92Z`;
    for (let u = -92; u < 90; u += 6.6) cr += `M${N(u)} ${b + 1.6}h4.2v2.8h-4.2Z`;
    P.p(cr, 'f-cwall kf');
    P.r(-92, b + 1.2, 184, 0.6, 'f-stoneD');
    return { g: P.s(), box: [-98, 98, 0, b + 78] };
  }

  // 颐和园佛香阁：石台 + 八角三层四重檐（黄琉璃绿剪边）
  function foxiangge() {
    const P = Pen();
    P.p(poly([[-41, 0], [41, 0], [37.6, 31], [-37.6, 31]]), 'f-ashlar k');
    P.p(poly([[-41, 0], [41, 0], [37.6, 31], [-37.6, 31]]), 'f-ashlarP');
    P.p(poly([[16, 0], [41, 0], [37.6, 31], [16, 31]]), 'f-shadeL');
    // 台面两道收分线脚
    P.p('M-40.2 7H40.2M-39.2 16H39.2M-38.4 24H38.4', 'kt');
    P.r(-38.6, 31, 77.2, 3, 'f-stone k');
    P.p('M-38.6 32.5H38.6', 'kt');
    terrace(P, 34, 1.8, 38, { rail: 2.8, post: 3.4 });
    body(P, 38.6, 49.4, 24, { cols: 7, lat: 0.62, lintel: 1.7 });
    dougong(P, 49.4, 1.5, 24, 1.8);
    roof(P, 50.6, 37, 6.4, 20, { noRidge: true, lift: 2.8, band: 1.3, tile: 1.6, trim: true, shadeW: 24 });
    body(P, 57, 63.2, 20.5, { cols: 7, lattice: false, lintel: 1.4 });
    roof(P, 63.2, 31.5, 6, 17, { noRidge: true, lift: 2.5, band: 1.2, tile: 1.5, trim: true, shadeW: 20.5 });
    terrace(P, 69.2, 1, 17.4, { rail: 2.2, post: 3 });
    body(P, 72.4, 79.6, 17, { cols: 5, lat: 0.6, lintel: 1.3 });
    dougong(P, 79.6, 1.2, 17, 1.6);
    roof(P, 80.6, 27, 5.6, 13.6, { noRidge: true, lift: 2.2, band: 1.1, tile: 1.4, trim: true, shadeW: 17 });
    body(P, 86.2, 93.6, 13.6, { cols: 5, lat: 0.6, lintel: 1.2 });
    roof(P, 93.6, 22.5, 20, 0, { lift: 2, band: 1.1, tile: 1.3, trim: true, shadeW: 13.6 });
    baoding(P, 113.6, 1.15);
    return { g: P.s(), box: [-44, 44, 0, 121] };
  }

  // 十七孔桥：长桥微拱，十七孔由两端向中间渐大，汉白玉栏杆望柱上蹲石狮
  function shiqikong() {
    const P = Pen(), L = 200, n = 17;
    const deck = u => 10 + 9 * (1 - (u / L) ** 2);
    const spans = Array.from({ length: n }, (_, i) => 7 + (8 - Math.abs(i - 8)) * 0.55), pier = 3.2;
    const total = spans.reduce((a, b) => a + b, 0) + pier * (n + 1);
    let x = -total / 2, holes = '';
    for (const s of spans) {
      x += pier;
      const c = x + s / 2, rise = deck(c) - 3.2 - s / 2;
      holes += `M${N(x)} 0V${N(Math.max(0.5, rise))}A${N(s / 2)} ${N(s / 2)} 0 0 0 ${N(x + s)} ${N(Math.max(0.5, rise))}V0Z`;
      x += s;
    }
    let top = ''; for (let u = -L; u <= L; u += 8) top += (u === -L ? 'M' : 'L') + `${N(u)} ${N(deck(u))}`;
    P.p(`M${-L} 0` + top.replace(/^M/, 'L') + `L${L} 0Z` + holes, 'f-stone k', ' fill-rule="evenodd"');
    let arcs = ''; x = -total / 2;
    for (const s of spans) { x += pier; const c = x + s / 2, rise = deck(c) - 3.2 - s / 2; arcs += `M${N(x - 0.8)} ${N(Math.max(0.5, rise))}A${N(s / 2 + 0.8)} ${N(s / 2 + 0.8)} 0 0 0 ${N(x + s + 0.8)} ${N(Math.max(0.5, rise))}`; x += s; }
    P.p(arcs, 'kt');
    // 栏杆：栏板 + 望柱石狮
    let rail = '', posts = '', lions = '';
    for (let u = -L + 2; u <= L - 2; u += 4) {
      const y = deck(u);
      rail += (u === -L + 2 ? 'M' : 'L') + `${N(u)} ${N(y + 2.6)}`;
      posts += `M${N(u)} ${N(y)}V${N(y + 3.4)}`;
      lions += `M${N(u - 0.8)} ${N(y + 3.4)}C${N(u - 1)} ${N(y + 4.8)} ${N(u + 1.1)} ${N(y + 5.2)} ${N(u + 0.9)} ${N(y + 3.4)}Z`;
    }
    P.p(rail, 'kf'); P.p(posts, 'kt'); P.p(lions, 'f-stone kt');
    let deckL = ''; for (let u = -L; u <= L; u += 8) deckL += (u === -L ? 'M' : 'L') + `${N(u)} ${N(deck(u) - 1.4)}`;
    P.p(deckL, 'kt');
    return { g: P.s(), box: [-L - 2, L + 2, 0, 26] };
  }

  // 玉泉山玉峰塔：远景小塔
  function yufeng() {
    const P = Pen();
    for (let i = 0; i < 7; i++) {
      const y = 4 + i * 6.2, w = 6.2 - i * 0.45;
      P.r(-w * 0.62, y, w * 1.24, 4.4, 'f-stone kt');
      P.p(`M${N(-w - 1.2)} ${N(y + 4.4)}Q0 ${N(y + 5.6)} ${N(w + 1.2)} ${N(y + 4.4)}L${N(w * 0.7)} ${N(y + 6.2)}H${N(-w * 0.7)}Z`, 'f-roofG kt');
    }
    P.r(-7, 0, 14, 4, 'f-stone kt');
    P.p('M0 48V54', 'kf'); P.c(0, 54.6, 0.9, 'f-gold kt');
    return { g: P.s(), box: [-9, 9, 0, 56] };
  }

  // 北海白塔：须弥座、塔肚、眼光门、十三天、华盖、日月宝顶
  function baita() {
    const P = Pen();
    const step = (y0, h, w, n) => P.p(poly([[-w, y0], [w, y0], [w, y0 + h * 0.5], [w - n, y0 + h * 0.5], [w - n, y0 + h], [-w + n, y0 + h], [-w + n, y0 + h * 0.5], [-w, y0 + h * 0.5]]), 'f-stone k');
    step(0, 6, 31, 2); step(6, 5, 26.5, 2); step(11, 4, 22.5, 1.5);
    P.p(poly([[-20, 15], [20, 15], [18.5, 18.4], [-18.5, 18.4]]), 'f-stone kf');
    const belly = 'M-18.5 18.4C-27.5 26-27 45-16.5 54.4C-9 60.4 9 60.4 16.5 54.4C27 45 27.5 26 18.5 18.4Z';
    P.p(belly, 'f-white k');
    P.p('M5 19.2C17 25.6 20 43 14.6 52.6C21 46 21.4 27 5 19.2Z', 'f-shadeL');
    P.p('M-6.4 25.6H6.4V36.4C6.4 41.4 3.2 44.6 0 46.6C-3.2 44.6-6.4 41.4-6.4 36.4Z', 'f-gable k');
    P.p('M-3.8 28H3.8V36.4C3.8 39.6 1.9 41.6 0 42.8C-1.9 41.6-3.8 39.6-3.8 36.4Z', 'f-gold kf');
    P.r(-9.5, 58.4, 19, 3.2, 'f-stone k');
    P.p(poly([[-8, 61.6], [8, 61.6], [4.2, 90], [-4.2, 90]]), 'f-stone k');
    let rings = ''; for (let i = 1; i < 13; i++) { const y = 61.6 + 28.4 * i / 13, h = 8 - 3.8 * i / 13; rings += `M${N(-h)} ${N(y)}H${N(h)}`; }
    P.p(rings, 'kf');
    P.p('M-13 91.6L13 91.6L9.6 95.6L-9.6 95.6Z', 'f-gold k');
    for (const u of [-12, -7, 7, 12]) { P.p(`M${u} 91.6v-2.2`, 'kt'); P.c(u, 89, 0.7, 'f-gold kt'); }
    P.c(0, 98.6, 2.2, 'f-gold k');
    P.p('M-3 103Q0 99.6 3 103Q0 101.6-3 103Z', 'f-gold k');
    P.c(0, 105.4, 1.3, 'f-gold k');
    return { g: P.s(), box: [-33, 33, 0, 108] };
  }

  // 中国国家博物馆（西立面）：中部高柱廊凹进，柱后是深深的阴影；两翼略低，一排排竖向长窗；
  // 檐口是黄琉璃绿剪边——远远一看就是这座楼
  function guobo() {
    const P = Pen(), cw = 84, ww = 158;
    // 基座与中部大台阶
    P.p(poly([[-ww - 8, 0], [ww + 8, 0], [ww + 5, 3], [-ww - 5, 3]]), 'f-stone k');
    P.p(poly([[-cw - 10, 0], [cw + 10, 0], [cw + 2, 8], [-cw - 2, 8]]), 'f-stone k');
    let st = ''; for (let y = 1.6; y < 8; y += 1.6) { const w = cw + 10 - 8 * y / 8; st += `M${N(-w)} ${N(y)}H${N(w)}`; } P.p(st, 'kt');
    // 两翼
    for (const sgn of [-1, 1]) {
      const a = sgn * cw, b = sgn * ww, lo = Math.min(a, b), hi = Math.max(a, b);
      P.p(poly([[lo, 3], [hi, 3], [hi, 54], [lo, 54]]), 'f-facade k');
      let win = '';
      const n = 7;
      for (let i = 0; i < n; i++) { const u = lo + 7 + i * ((hi - lo - 14) / (n - 1)); for (const [y0, h] of [[8, 11], [22, 11], [36, 11]]) win += `M${N(u - 2.1)} ${N(y0)}h4.2v${h}h-4.2Z`; }
      P.p(win, 'f-win kt');
      P.p(`M${N(lo)} 20H${N(hi)}M${N(lo)} 34H${N(hi)}`, 'kt');
      P.r(lo - (sgn < 0 ? 2 : 0), 54, hi - lo + 2, 2.6, 'f-facade k');
      P.g(sub(Q => roof(Q, 56.6, (hi - lo) / 2 + 3, 3.6, (hi - lo) / 2 - 1, { tone: 'Y', noRidge: true, trim: true, soffit: false, tile: 2.2, band: 1.1, lift: 0.6, flick: 0.6 })), ` transform="translate(${N((lo + hi) / 2)} 0)"`);
      if (sgn > 0) P.p(poly([[hi - 10, 3], [hi, 3], [hi, 54], [hi - 10, 54]]), 'f-shadeL');
    }
    // 中部柱廊：后墙深影，十二根方柱，左面受光
    P.r(-cw, 8, cw * 2, 58, 'f-recess k');
    let dr = ''; for (let i = 0; i < 11; i++) { const u = -cw + 7 + 6.2 + i * ((cw * 2 - 14) / 11); dr += `M${N(u - 2.4)} 8h4.8v14h-4.8Z`; } P.p(dr, 'f-dark');
    let cols = '', colS = ''; const nc = 12;
    for (let i = 0; i < nc; i++) { const u = -cw + 6 + i * ((cw * 2 - 12) / (nc - 1)); cols += `M${N(u - 3.4)} 8h6.8v58h-6.8Z`; colS += `M${N(u + 1)} 8h2.4v58h-2.4Z`; }
    P.p(cols, 'f-facade kf'); P.p(colS, 'f-shadeL');
    // 额枋、金色匾额带
    P.r(-cw - 2, 66, cw * 2 + 4, 7, 'f-facade k');
    P.r(-26, 67.6, 52, 3.8, 'f-gold kf');
    let dent = ''; for (let u = -cw; u < cw; u += 3.6) dent += `M${N(u)} 73v-1.3h1.8v1.3`; P.p(dent, 'kt');
    // 黄琉璃绿剪边挑檐 + 女儿墙
    roof(P, 73, cw + 10, 6.4, cw - 2, { tone: 'Y', noRidge: true, trim: true, tile: 2.4, band: 1.5, lift: 1.2, flick: 1, soffit: false });
    P.r(-cw + 16, 79.4, cw * 2 - 32, 7, 'f-facade k');
    P.r(-cw + 14, 86.4, cw * 2 - 28, 1.6, 'f-facade kf');
    return { g: P.s(), box: [-ww - 10, ww + 10, 0, 88] };
  }

  const LANDMARKS = { qinian, jiaolou, foxiangge, shiqikong, yufeng, baita, guobo };
  function place(name, x, y, h, cls = '', attrs = '') {
    const L = LANDMARKS[name]();
    const s = h / (L.box[3] - L.box[2]);
    return `<g class="lm lm-${name} ${cls}" transform="translate(${N(x)} ${N(y)}) scale(${N(s)} ${N(-s)})"${attrs}>${L.g}</g>`;
  }

  const ART2 = { N, pt, poly, rng, Pen, sub, defs, roof, dougong, caihua, body, terrace, stairs, baoding, xsGable, samp, yAt, C, rev, mc, bez, LANDMARKS, place };
  window.ART2 = ART2;
})();
