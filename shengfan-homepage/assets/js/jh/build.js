/* 界画库 · 组合：按开间模数把构件拼成一座殿（正立面）
   bays：从明间往外的各间面阔（单位约 0.1 米），左右对称；colH 檐柱高；platform 台基高。
   屋顶 type：wd 庑殿 / xs 歇山 / cj 攒尖；double 重檐。 */
(function () {
  'use strict';
  const { N, Pen } = window.JH, A = window.JH.ARCH;

  function columnXs(bays) {
    // 明间居中，左右对称往外
    const xs = [-bays[0] / 2, bays[0] / 2];
    let l = -bays[0] / 2, r = bays[0] / 2;
    for (let i = 1; i < bays.length; i++) { l -= bays[i]; r += bays[i]; xs.unshift(l); xs.push(r); }
    return xs;
  }

  function hall(o) {
    const P = Pen();
    const xs = columnXs(o.bays), W = xs[xs.length - 1], cw = o.colW ?? o.colH * 0.085;
    const tone = o.tone || 'yellow', kind = o.caihua || 'hexi';
    const pH = o.platform ?? 0, pW = W + (o.platOver ?? o.colH * 0.55);
    let y = 0;
    // 台基、栏杆、台阶
    if (pH) y = A.xumizuo(P, 0, pH, pW, { chishou: o.chishou !== false });
    const y0 = y, y1 = y0 + o.colH;
    // 柱间隔扇（梢间、尽间可改槛窗）
    for (let i = 0; i < xs.length - 1; i++) A.geshan(P, xs[i] + cw / 2, xs[i + 1] - cw / 2, y0 + cw * 0.3, y1 - o.colH * 0.02, o.leaves ?? 4);
    A.columns(P, xs, y0, y1, cw);
    // 额枋、斗拱
    const bH = o.colH * 0.2, dH = o.colH * 0.22;
    A.beams(P, xs, y1, bH, kind);
    A.dougong(P, xs, y1 + bH, dH, o.per ?? 4, cw);
    let top = y1 + bH + dH;
    const ew = W + (o.over ?? o.colH * 0.62), rh = o.rh ?? o.colH * 1.05;
    if (o.double) {
      // 下层腰檐，上面是上檐的屋身（矮一截、缩进一圈）
      const wallW = W - (o.inset ?? (o.bays.length > 1 ? o.bays[o.bays.length - 1] : W * 0.3));
      const yao = A.roof(P, { type: 'yao', ey: top - o.colH * 0.04, ew, rh: o.colH * 0.36, wallW, tone, trim: o.trim, tile: o.tile, lift: o.colH * 0.16, sd: o.colH * 0.2, ridgeH: rh * 0.1, beasts: o.beasts });
      const xs2 = o.bays.length > 1 ? xs.slice(1, -1) : [-wallW, wallW], b0 = yao.top - o.colH * 0.02, b1 = b0 + o.colH * 0.36;
      for (let i = 0; i < xs2.length - 1; i++) A.geshan(P, xs2[i] + cw / 2, xs2[i + 1] - cw / 2, b0, b1, 4);
      A.columns(P, xs2, b0, b1, cw * 0.92);
      A.beams(P, xs2, b1, bH * 0.9, kind);
      A.dougong(P, xs2, b1 + bH * 0.9, dH * 0.95, o.per ?? 4, cw);
      top = b1 + bH * 0.9 + dH * 0.95;
      const r = A.roof(P, { type: o.type || 'wd', ey: top - o.colH * 0.04, ew: wallW + (o.over ?? o.colH * 0.62), rh, rw: o.rw ?? (wallW * 0.5), tone, trim: o.trim, tile: o.tile, beasts: o.beasts, lift: o.colH * 0.2 });
      top = r.ridgeTop;
      if (o.type === 'cj') { A.baoding(P, r.top - rh * 0.02, rh * 0.09); top = r.top + rh * 0.5; }
    } else {
      const r = A.roof(P, { type: o.type || 'wd', ey: top - o.colH * 0.04, ew, rh, rw: o.rw ?? W * 0.5, tone, trim: o.trim, tile: o.tile, beasts: o.beasts, sides: o.sides, lift: o.colH * 0.22 });
      top = r.ridgeTop;
      if (o.type === 'cj') { A.baoding(P, r.top - rh * 0.02, rh * 0.09); top = r.top + rh * 0.5; }
    }
    // 台明边上的栏杆挡在殿身前面，台阶画在最前
    if (pH && o.rail) {
      const sw = o.stairW ?? o.bays[0] * 0.5;
      A.balustrade(P, -pW + cw, -sw - cw, y0, o.rail, o.rail * 1.05, o.railHead);
      A.balustrade(P, sw + cw, pW - cw, y0, o.rail, o.rail * 1.05, o.railHead);
    }
    if (pH && o.stairs !== false) A.stairs(P, -pH * 0.05, y0, o.stairW ?? o.bays[0] * 0.5, o.ramp ?? o.bays[0] * 0.16);
    const half = Math.max(pW, ew) + o.colH * 0.3;
    return { g: P.s(), box: [-half, half, -pH * 0.06, top + o.colH * 0.15] };
  }

  // 把一座殿的尺寸参数统一乘 k（远处的殿小一号，线宽不跟着变）
  const DIM = ['bays', 'colH', 'colW', 'platform', 'platOver', 'rail', 'stairW', 'ramp', 'rh', 'rw', 'over', 'inset', 'tile'];
  function scaled(o, k) { const q = { ...o }; for (const key of DIM) if (q[key] != null) q[key] = Array.isArray(q[key]) ? q[key].map(v => v * k) : q[key] * k; return q; }

  window.JH.BUILD = { hall, columnXs, scaled };
})();
