/* 晟帆主页 · 线描图形库
   建筑都在局部坐标里画：u 向右、v 向上，原点在建筑底部中线；输出时套一层
   translate(x y) scale(s -s)。线条用 non-scaling-stroke，缩放后粗细不变。 */
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

  function Pen() {
    const o = [];
    const api = {
      p(d, c, extra = '') { if (d) o.push(`<path class="${c}" d="${d}"${extra}/>`); return api; },
      r(x, y, w, h, c) { o.push(`<rect class="${c}" x="${N(x)}" y="${N(y)}" width="${N(w)}" height="${N(h)}"/>`); return api; },
      c(x, y, r, c) { o.push(`<circle class="${c}" cx="${N(x)}" cy="${N(y)}" r="${N(r)}"/>`); return api; },
      g(inner, attrs = '') { o.push(`<g${attrs}>${inner}</g>`); return api; },
      s() { return o.join(''); },
    };
    return api;
  }
  const poly = (pts, close = true) => 'M' + pts.map(pt).join('L') + (close ? 'Z' : '');
  const sym = half => half.concat(half.slice().reverse().map(p => [-p[0], p[1]])); // 右半边轮廓（自下而上）→ 对称闭合

  /* ── 构件 ───────────────────────────────────────────── */

  // 屋顶立面。ey 檐口中点高度，ew 檐口半宽，rh 屋面高，rw 正脊半宽（0 为攒尖）
  // o.round：圆形攒尖（瓦垄向顶点收）；o.noRidge：上沿被上一层屋身挡住；o.ridge：'wen' 正吻 / 'xie' 蝎子尾
  function roof(P, ey, ew, rh, rw, o = {}) {
    const round = !!o.round;
    const lift = o.lift ?? ew * 0.07, band = o.band ?? Math.max(1.3, rh * 0.14), top = ey + rh;
    const Bt = [[-ew, ey + lift], [-ew * 0.5, ey - lift * 0.1], [ew * 0.5, ey - lift * 0.1], [ew, ey + lift]];
    const Tt = [[-ew * 0.985, ey + lift + band * 0.55], [-ew * 0.5, ey + band * 0.95], [ew * 0.5, ey + band * 0.95], [ew * 0.985, ey + lift + band * 0.55]];
    const R0 = Tt[3], dx = R0[0] - rw, dy = top - R0[1];
    const k = round ? [0.34, 0.12, 0.14, 0.5] : [0.5, 0.06, 0.1, 0.52];
    const RS = [R0, [R0[0] - dx * (o.k1 ?? k[0]), R0[1] + dy * (o.k2 ?? k[1])], [rw + dx * (o.k3 ?? k[2]), R0[1] + dy * (o.k4 ?? k[3])], [rw, top]];
    const LS = mc(RS);
    // 檐下阴影，只落在屋身宽度内
    if (o.shade !== false) { const sw = o.shadeW ?? ew * 0.8; P.p(`M${N(-sw)} ${N(ey)}H${N(sw)}V${N(ey - band * 1.3)}H${N(-sw)}Z`, 'w-shade'); }
    const shape = `M${pt(Bt[0])}${C(Bt)}L${pt(R0)}${C(RS)}L${pt(LS[3])}${C(rev(LS))}Z`;
    P.p(shape, `${o.w || 'w-roof-y'} k`);
    if (round) P.p(shape, 'vol');
    if (o.trim) P.p(`M${pt(Bt[0])}${C(Bt)}L${pt(Tt[3])}${C(rev(Tt))}Z`, `${o.trim} kf`);
    P.p(`M${pt(Tt[0])}${C(Tt)}`, 'kf');
    // 瓦垄
    const tp = samp(Tt), rp = samp(RS), lp = samp(LS);
    let td = '';
    if (round) {
      // 圆形屋面：瓦垄是屋面上的经线，按 sin 分布（两侧密、中间疏），沿屋面轮廓收向顶部
      const n = o.tiles ?? Math.round(ew / 1.6);
      for (let i = 1; i < n; i++) {
        const f = Math.sin(-Math.PI / 2 + Math.PI * i / n);
        if (Math.abs(f) > 0.96) continue;
        const pts = rp.map(([x, y]) => [f * x, y]);
        const yb = yAt(tp, pts[0][0]);
        if (yb != null) pts[0] = [pts[0][0], yb];
        td += 'M' + pts.filter((_, j) => j % 2 === 0 || j === pts.length - 1).map(pt).join('L');
      }
    } else {
      const step = o.tile ?? Math.max(1.8, ew / 13);
      for (let u = -ew * 0.93 + step * 0.5; u < ew * 0.93; u += step) {
        const yb = yAt(tp, u);
        if (yb == null) continue;
        const yt = Math.abs(u) <= rw ? top : yAt(u > 0 ? rp : lp, u);
        if (yt == null || yt - yb < 1) continue;
        td += `M${N(u)} ${N(yb)}L${N(u)} ${N(yt)}`;
      }
    }
    P.p(td, 'kt');
    // 翼角起翘（圆形屋面没有翼角）
    const f = band * (o.flick ?? 1.5);
    if (!round) P.p(`M${N(ew)} ${N(ey + lift)}Q${N(ew * 1.04)} ${N(ey + lift + f * 0.15)} ${N(ew * 1.06 + 0.4)} ${N(ey + lift + f)}M${N(-ew)} ${N(ey + lift)}Q${N(-ew * 1.04)} ${N(ey + lift + f * 0.15)} ${N(-ew * 1.06 - 0.4)} ${N(ey + lift + f)}`, 'k');
    // 正脊与吻兽
    if (rw > 0 && !o.noRidge) {
      const b = band * 0.9;
      P.r(-rw, top, rw * 2, b, `${o.w || 'w-roof-y'} k`);
      if (o.ridge === 'xie') {
        P.p(`M${N(rw)} ${N(top + b)}Q${N(rw + b * 1.4)} ${N(top + b * 1.1)} ${N(rw + b * 1.9)} ${N(top + b * 2.8)}M${N(-rw)} ${N(top + b)}Q${N(-rw - b * 1.4)} ${N(top + b * 1.1)} ${N(-rw - b * 1.9)} ${N(top + b * 2.8)}`, 'k');
      } else if (o.ridge !== 'none') {
        const h = b * 2.6;
        P.p(`M${N(rw - b * 0.2)} ${N(top + b)}C${N(rw + b * 0.5)} ${N(top + b + h * 0.6)} ${N(rw - b * 0.6)} ${N(top + b + h)} ${N(rw - b * 1.5)} ${N(top + b + h * 0.75)}M${N(-rw + b * 0.2)} ${N(top + b)}C${N(-rw - b * 0.5)} ${N(top + b + h * 0.6)} ${N(-rw + b * 0.6)} ${N(top + b + h)} ${N(-rw + b * 1.5)} ${N(top + b + h * 0.75)}`, `${o.w || 'w-roof-y'} k`);
      }
    }
    return { top, band };
  }

  // 歇山山花（正面朝外的三角山墙）：底宽 hw，底高 y0，顶高 y1
  function gable(P, y0, hw, y1, o = {}) {
    const b = o.b ?? Math.max(0.9, hw * 0.09);
    P.p(poly([[-hw, y0], [0, y1], [hw, y0]]), `${o.w || 'w-paint'} k`);
    P.p(`M${N(-hw + b * 1.6)} ${N(y0 + b * 0.6)}L0 ${N(y1 - b * 1.7)}L${N(hw - b * 1.6)} ${N(y0 + b * 0.6)}`, 'kf');
    // 博缝板外缘
    P.p(`M${N(-hw - b * 0.8)} ${N(y0 - b * 0.3)}L0 ${N(y1 + b * 0.9)}L${N(hw + b * 0.8)} ${N(y0 - b * 0.3)}`, 'k');
    // 悬鱼
    P.p(`M0 ${N(y1 - b * 0.4)}C${N(b * 0.9)} ${N(y1 - b * 1.6)} ${N(b * 0.5)} ${N(y1 - b * 2.8)} 0 ${N(y1 - b * 3.2)}C${N(-b * 0.5)} ${N(y1 - b * 2.8)} ${N(-b * 0.9)} ${N(y1 - b * 1.6)} 0 ${N(y1 - b * 0.4)}Z`, 'w-gold kf');
  }

  // 歇山屋顶的山面（三角山花朝外）：ey 檐口中点，ew 檐口半宽，h 到顶点的高度
  function xsGable(P, ey, ew, h, o = {}) {
    const lift = o.lift ?? ew * 0.13, band = o.band ?? Math.max(0.9, h * 0.06), apex = ey + h;
    const Bt = [[-ew, ey + lift], [-ew * 0.45, ey - lift * 0.12], [ew * 0.45, ey - lift * 0.12], [ew, ey + lift]];
    const Tt = [[-ew * 0.97, ey + lift + band * 0.6], [-ew * 0.45, ey + band * 0.95], [ew * 0.45, ey + band * 0.95], [ew * 0.97, ey + lift + band * 0.6]];
    const R0 = Tt[3], RS = [R0, [ew * 0.58, ey + lift + h * 0.1], [ew * 0.2, ey + h * 0.6], [0, apex]], LS = mc(RS);
    if (o.shade !== false) { const sw = o.shadeW ?? ew * 0.7; P.p(`M${N(-sw)} ${N(ey)}H${N(sw)}V${N(ey - band * 1.6)}H${N(-sw)}Z`, 'w-shade'); }
    P.p(`M${pt(Bt[0])}${C(Bt)}L${pt(R0)}${C(RS)}${C(rev(LS))}Z`, `${o.w || 'w-roof-y'} k`);
    P.p(`M${pt(Tt[0])}${C(Tt)}`, 'kf');
    const gb = ey + h * (o.gy ?? 0.44), gw = ew * (o.gw ?? 0.34), ga = apex - h * 0.07, bw = band * 1.1;
    const tp = samp(Tt), rp = samp(RS), lp = samp(LS), step = o.tile ?? Math.max(1.4, ew / 11);
    let td = '';
    for (let u = -ew * 0.9 + step / 2; u < ew * 0.9; u += step) {
      const yb = yAt(tp, u);
      let yt = yAt(u > 0 ? rp : lp, u);
      if (yb == null || yt == null) continue;
      if (Math.abs(u) < gw + bw) yt = Math.min(yt, gb);
      if (yt - yb > 0.8) td += `M${N(u)} ${N(yb)}L${N(u)} ${N(yt)}`;
    }
    P.p(td, 'kt');
    P.p(poly([[-gw, gb], [0, ga], [gw, gb]]), `${o.gwash || 'w-wall'} kf`);
    P.p(`M${N(-gw - bw)} ${N(gb - bw * 0.25)}L0 ${N(ga + bw * 1.35)}L${N(gw + bw)} ${N(gb - bw * 0.25)}L${N(gw)} ${N(gb)}L0 ${N(ga)}L${N(-gw)} ${N(gb)}Z`, `${o.w || 'w-roof-y'} k`);
    P.p(`M0 ${N(ga - 0.2)}C${N(bw * 0.9)} ${N(ga - bw * 1.4)} ${N(bw * 0.5)} ${N(ga - bw * 2.6)} 0 ${N(ga - bw * 3)}C${N(-bw * 0.5)} ${N(ga - bw * 2.6)} ${N(-bw * 0.9)} ${N(ga - bw * 1.4)} 0 ${N(ga - 0.2)}Z`, 'w-gold kf');
    P.p(`M${N(-gw - bw * 1.6)} ${N(gb)}H${N(gw + bw * 1.6)}`, 'kf');
    const f = band * (o.flick ?? 2.4);
    P.p(`M${N(ew)} ${N(ey + lift)}Q${N(ew * 1.05)} ${N(ey + lift + f * 0.2)} ${N(ew * 1.08 + 0.4)} ${N(ey + lift + f)}M${N(-ew)} ${N(ey + lift)}Q${N(-ew * 1.05)} ${N(ey + lift + f * 0.2)} ${N(-ew * 1.08 - 0.4)} ${N(ey + lift + f)}`, 'k');
    if (o.wen !== false) P.p(`M-0.4 ${N(apex + bw * 0.9)}C-0.2 ${N(apex + bw * 3)} ${N(bw * 1.5)} ${N(apex + bw * 3.6)} ${N(bw * 1.3)} ${N(apex + bw * 2.1)}`, 'k');
    return { apex };
  }

  // 屋身：柱、隔扇、额枋
  function body(P, y0, y1, hw, o = {}) {
    const cols = o.cols ?? 6, cw = o.cw ?? Math.max(1.1, hw * 0.04), h = y1 - y0;
    P.r(-hw, y0, hw * 2, h, `${o.w || 'w-red-l'} k`);
    if (o.lattice !== false) {
      let d = '';
      for (let i = 0; i < cols - 1; i++) {
        const a = -hw + (2 * hw) * i / (cols - 1) + cw * 0.8, b = -hw + (2 * hw) * (i + 1) / (cols - 1) - cw * 0.8;
        const lt = y0 + h * 0.08, lh = h * (o.lat ?? 0.66);
        d += `M${N(a)} ${N(lt)}H${N(b)}V${N(lt + lh)}H${N(a)}Z`;
        const n = Math.max(2, Math.round((b - a) / (o.mul ?? 1.6)));
        for (let k = 1; k < n; k++) { const x = a + (b - a) * k / n; d += `M${N(x)} ${N(lt)}V${N(lt + lh)}`; }
        d += `M${N(a)} ${N(lt + lh * 0.62)}H${N(b)}`;
      }
      P.p(d, 'kt');
    }
    let cd = '';
    for (let i = 0; i < cols; i++) { const u = -hw + (2 * hw) * i / (cols - 1); cd += `M${N(u - cw / 2)} ${N(y0)}h${N(cw)}v${N(h)}h${N(-cw)}Z`; }
    P.p(cd, 'w-col kf');
    const lb = o.lintel ?? Math.min(2.6, h * 0.16);
    P.r(-hw, y1 - lb, hw * 2, lb, 'w-paint kf');
    if (o.dougong) {
      const n = Math.max(4, Math.round(hw * 2 / o.dougong));
      let d = '';
      for (let i = 0; i <= n; i++) { const u = -hw + i * (2 * hw) / n; d += `M${N(u - 0.6)} ${N(y1)}h1.2v${N(o.dgH ?? 1.6)}h-1.2Z`; }
      P.p(d, 'w-paint kt');
    }
  }

  // 台基 / 须弥座，带栏杆
  function terrace(P, y0, h, hw, o = {}) {
    const bat = o.bat ?? 0, top = y0 + h, w = hw - bat;
    P.p(poly([[-hw, y0], [hw, y0], [w, top], [-w, top]]), `${o.w || 'w-stone'} k`);
    let d = '';
    for (const fr of (o.mold ?? [0.24, 0.76])) { const yy = y0 + h * fr, ww = hw - bat * fr; d += `M${N(-ww)} ${N(yy)}H${N(ww)}`; }
    P.p(d, 'kf');
    if (o.rail) {
      const rh = o.rail, sp = o.post ?? 5;
      P.r(-w, top, w * 2, rh * 0.76, `${o.w || 'w-stone'} kf`);
      const n = Math.max(2, Math.round(w * 2 / sp));
      let r = `M${N(-w)} ${N(top + rh * 0.22)}H${N(w)}`, caps = '';
      for (let i = 0; i <= n; i++) {
        const u = -w + i * (2 * w) / n;
        r += `M${N(u)} ${N(top)}V${N(top + rh)}`;
        caps += `M${N(u - 0.55)} ${N(top + rh)}Q${N(u)} ${N(top + rh + 1.3)} ${N(u + 0.55)} ${N(top + rh)}Z`;
      }
      P.p(r, 'kf');
      P.p(caps, `${o.w || 'w-stone'} kf`);
    }
  }

  // 台阶（居中）
  function stairs(P, y0, y1, w0, w1, n, o = {}) {
    P.p(poly([[-w0, y0], [w0, y0], [w1, y1], [-w1, y1]]), `${o.w || 'w-stone'} k`);
    let d = '';
    for (let i = 1; i < n; i++) { const t = i / n, yy = y0 + (y1 - y0) * t, ww = w0 + (w1 - w0) * t; d += `M${N(-ww)} ${N(yy)}H${N(ww)}`; }
    P.p(d, 'kt');
    if (o.ramp) { const r = o.ramp; P.p(poly([[-r, y0], [r, y0], [r * 0.85, y1], [-r * 0.85, y1]]), 'w-stone kf'); P.p(`M${N(-r * 0.5)} ${N(y0 + 1)}C${N(r * 0.6)} ${N(y0 + (y1 - y0) * 0.3)} ${N(-r * 0.6)} ${N(y0 + (y1 - y0) * 0.6)} ${N(r * 0.4)} ${N(y1 - 1)}`, 'kt'); }
  }

  // 鎏金宝顶：须弥座、宝珠、尖
  function baoding(P, y0, s) {
    P.p(poly([[-s * 1.5, y0], [s * 1.5, y0], [s * 1.05, y0 + s * 0.8], [s * 1.3, y0 + s * 1.15], [-s * 1.3, y0 + s * 1.15], [-s * 1.05, y0 + s * 0.8]]), 'w-gold k');
    P.p(`M${N(-s * 0.55)} ${N(y0 + s * 1.15)}C${N(-s * 1.35)} ${N(y0 + s * 1.9)} ${N(-s * 1.2)} ${N(y0 + s * 3.4)} 0 ${N(y0 + s * 3.7)}C${N(s * 1.2)} ${N(y0 + s * 3.4)} ${N(s * 1.35)} ${N(y0 + s * 1.9)} ${N(s * 0.55)} ${N(y0 + s * 1.15)}Z`, 'w-gold k');
    P.p(`M${N(-s * 0.35)} ${N(y0 + s * 3.6)}Q0 ${N(y0 + s * 4.3)} ${N(s * 0.35)} ${N(y0 + s * 3.6)}L0 ${N(y0 + s * 6)}Z`, 'w-gold k');
  }

  // 页面里放一次：圆屋面体积光等渐变
  function defs() {
    return '<svg width="0" height="0" style="position:absolute" aria-hidden="true"><defs>' +
      '<linearGradient id="art-vol" x1="0" x2="1" y1="0" y2="0">' +
      '<stop offset="0" stop-color="#1f2f52" stop-opacity=".2"/><stop offset=".3" stop-color="#1f2f52" stop-opacity="0"/>' +
      '<stop offset=".42" stop-color="#fff" stop-opacity=".28"/><stop offset=".6" stop-color="#fff" stop-opacity="0"/>' +
      '<stop offset="1" stop-color="#1f2f52" stop-opacity=".24"/></linearGradient>' +
      '<linearGradient id="art-sky" x1="0" x2="0" y1="0" y2="1"><stop offset="0" stop-color="#f4ecdc" stop-opacity="0"/><stop offset=".5" stop-color="#f4ead6" stop-opacity=".55"/><stop offset=".833" stop-color="#f1e2c6" stop-opacity=".95"/><stop offset=".834" stop-color="#f1e2c6" stop-opacity="0"/></linearGradient>' +
      '<linearGradient id="art-water" x1="0" x2="0" y1="0" y2="1"><stop offset="0" stop-color="#9fb6cf" stop-opacity=".34"/><stop offset=".55" stop-color="#c3d0de" stop-opacity=".14"/><stop offset="1" stop-color="#f6f4ef" stop-opacity="0"/></linearGradient>' +
      '<radialGradient id="art-sun"><stop offset="0" stop-color="#f2cf8a" stop-opacity=".55"/><stop offset=".6" stop-color="#f2d59c" stop-opacity=".22"/><stop offset="1" stop-color="#f2d59c" stop-opacity="0"/></radialGradient>' +
      '</defs></svg>';
  }

  /* ── 北京地标 ───────────────────────────────────────── */
  // 每个函数返回 { g: 局部坐标里的图形, box: [左, 右, 下, 上] }

  // 天坛祈年殿
  function qinian() {
    const P = Pen();
    body(P, 21, 39.4, 33, { cols: 9, lat: 0.7, mul: 1.5, lintel: 1.6 });
    P.r(-33.6, 39.4, 67.2, 2.6, 'w-paint kf');
    let dg = ''; for (let u = -32; u <= 32; u += 2.4) dg += `M${N(u)} 39.8v1.8`; P.p(dg, 'kt');
    roof(P, 42, 51, 13.5, 28, { w: 'w-roof-b', round: true, noRidge: true, lift: 3, band: 1.7, tiles: 40, shadeW: 34 });
    body(P, 55.5, 60.3, 28, { cols: 11, lattice: false, w: 'w-paint', lintel: 1.4, cw: 0.8 });
    roof(P, 60.3, 40.5, 12, 20.5, { w: 'w-roof-b', round: true, noRidge: true, lift: 2.6, band: 1.5, tiles: 34, shadeW: 28 });
    body(P, 72.3, 76.6, 20.5, { cols: 9, lattice: false, w: 'w-paint', lintel: 1.2, cw: 0.75 });
    P.r(-2.4, 72.4, 4.8, 6.2, 'w-plaque kf');
    P.p('M-1.5 73.3H1.5V77.8H-1.5Z', 'kt');
    roof(P, 76.6, 30, 19.5, 0, { w: 'w-roof-b', round: true, lift: 2.2, band: 1.4, tiles: 28, shadeW: 20.5 });
    baoding(P, 96, 1.35);
    // 祈谷坛：自上而下画，前一层栏杆压住后一层台面
    terrace(P, 14, 7, 60, { rail: 3, post: 4.4, bat: 1 });
    terrace(P, 7, 7, 72, { rail: 3, post: 4.4, bat: 1 });
    terrace(P, 0, 7, 84, { rail: 3, post: 4.4, bat: 1 });
    stairs(P, 0, 21, 9.5, 7.2, 14, { ramp: 3.4 });
    return { g: P.s(), box: [-86, 86, 0, 105] };
  }

  // 故宫角楼：取从筒子河外看转角的角度——转角居中，左右两个立面对称展开，
  // 上层两个山花加十字脊宝顶，中层两个大山花，下层腰檐上四个抱厦小山花
  function jiaolou() {
    const P = Pen(), b = 18;
    const at = (x, fn) => P.g(sub(fn), ` transform="translate(${x} 0)"`);
    // 顶层：两个山面之间先铺一块屋面（十字脊的另一条脊从这里伸向后方）
    body(P, b + 42, b + 49, 21, { cols: 7, lattice: false, lintel: 1.2 });
    P.p(poly([[-10.5, b + 70], [10.5, b + 70], [0, b + 57]]), 'w-roof-y k');
    let vt = ''; for (let u = -8; u <= 8; u += 1.6) vt += `M${N(u)} ${N(b + 57 + 13 * Math.abs(u) / 10.5)}V${N(b + 69.6)}`; P.p(vt, 'kt');
    for (const x of [-10.5, 10.5]) at(x, Q => xsGable(Q, b + 48, 17, 22, { gw: 0.38, tile: 1.3, flick: 2.6, wen: false, shade: false }));
    P.r(-10.8, b + 69.6, 21.6, 1.3, 'w-roof-y k');
    baoding(P, b + 70.9, 1.1);
    // 中层
    body(P, b + 26, b + 34, 37, { cols: 9, lat: 0.6, mul: 1.25, lintel: 1.3 });
    for (const x of [-19.5, 19.5]) at(x, Q => xsGable(Q, b + 23.5, 29, 24.5, { tile: 1.5, flick: 2.6, shade: false, wen: false }));
    roof(P, b + 23, 13, 4.4, 0, { noRidge: true, lift: 2.6, band: 1.1, shade: false, tile: 1.5 });
    // 下层
    body(P, b, b + 11, 47, { cols: 13, lat: 0.62, mul: 1.25, lintel: 1.4 });
    roof(P, b + 10.6, 63, 6, 52, { lift: 3.8, band: 1.3, tile: 2, noRidge: true, shadeW: 47 });
    for (const x of [-42, -14, 14, 42]) at(x, Q => xsGable(Q, b + 12.4, 13.5, 14.5, { tile: 1.25, flick: 2.2, shade: false, wen: false }));
    // 城墙与垛口（压住楼身底部）
    P.p(poly([[-96, 0], [96, 0], [92, b], [-92, b]]), 'w-brick k');
    let bd = ''; for (let y = 3; y < b; y += 3) { const w = 96 - 4 * y / b; bd += `M${N(-w)} ${N(y)}H${N(w)}`; }
    P.p(bd, 'kt');
    let cr = `M-92 ${b}H92V${b + 1.8}H-92Z`;
    for (let u = -92; u < 90; u += 6.6) cr += `M${N(u)} ${b + 1.8}h4.2v2.8h-4.2Z`;
    P.p(cr, 'w-brick kf');
    return { g: P.s(), box: [-98, 98, 0, b + 78] };
  }

  function sub(fn) { const Q = Pen(); fn(Q); return Q.s(); }

  // 砖缝横线
  function courses(P, x0, x1, y0, y1, step, bat = 0) {
    let d = '';
    for (let y = y0 + step; y < y1 - 0.2; y += step) { const k = bat * (y - y0) / (y1 - y0); d += `M${N(x0 + k)} ${N(y)}H${N(x1 - k)}`; }
    P.p(d, 'kt');
  }

  // 颐和园佛香阁（立在万寿山石台上，八角三层四重檐，黄琉璃绿剪边）
  function foxiangge() {
    const P = Pen();
    P.p('M-104 0C-76 5-56 30-34 47C-22 56-10 59 0 59C10 59 22 56 34 47C56 30 76 5 104 0Z', 'w-hill k');
    trees(P, [[-78, 9, 4.2], [-66, 16, 4.8], [-54, 25, 4.4], [-43, 34, 5], [56, 24, 4.6], [68, 15, 4.8], [80, 8, 4.2], [46, 33, 4.6], [-88, 4, 3.6], [90, 3.5, 3.6]]);
    P.p(poly([[-41, 52], [41, 52], [37, 86], [-37, 86]]), 'w-brick k');
    courses(P, -41, 41, 52, 86, 3.4, 4);
    P.p('M-6.5 52V62A6.5 6.5 0 0 0 6.5 62V52Z', 'w-dark k');
    P.p('M-9.5 52V62A9.5 9.5 0 0 0 9.5 62V52', 'kf');
    terrace(P, 86, 1.6, 38, { rail: 2.6, post: 3.6, w: 'w-stone', mold: [] });
    body(P, 90.2, 101, 24, { cols: 7, lat: 0.6, mul: 1.2, lintel: 1.3 });
    roof(P, 101, 37, 6.4, 20, { noRidge: true, lift: 2.8, band: 1.2, tile: 1.7, trim: 'w-roof-gr', shadeW: 24 });
    body(P, 107.4, 113.6, 20.5, { cols: 7, lattice: false, lintel: 1.2 });
    roof(P, 113.6, 31.5, 6, 17, { noRidge: true, lift: 2.5, band: 1.1, tile: 1.6, trim: 'w-roof-gr', shadeW: 20.5 });
    terrace(P, 119.6, 1, 17.4, { rail: 2.2, post: 3.2, mold: [] });
    body(P, 122.8, 130, 17, { cols: 5, lat: 0.6, mul: 1.1, lintel: 1.1 });
    roof(P, 130, 27, 5.6, 13.6, { noRidge: true, lift: 2.2, band: 1, tile: 1.5, trim: 'w-roof-gr', shadeW: 17 });
    body(P, 135.6, 143, 13.6, { cols: 5, lat: 0.6, mul: 1.05, lintel: 1 });
    roof(P, 143, 22.5, 20, 0, { lift: 2, band: 1, tile: 1.4, trim: 'w-roof-gr', shadeW: 13.6 });
    baoding(P, 163, 1.15);
    return { g: P.s(), box: [-106, 106, 0, 171] };
  }

  // 北海白塔（琼华岛上的藏式喇嘛塔）
  function baita() {
    const P = Pen();
    P.p('M-62 0C-46 3-36 22-22 30C-14 34 14 34 22 30C36 22 46 3 62 0Z', 'w-hill k');
    trees(P, [[-44, 9, 4], [-34, 17, 4.4], [36, 16, 4.4], [46, 8, 4], [-52, 4, 3.4], [53, 3.6, 3.4]]);
    const zj = (y0, h, w, n) => { const pts = [[w, y0]]; for (let i = 0; i < n; i++) { pts.push([w - i * 1.8, y0 + h * 0.15 + i * 0.01], [w - (i + 1) * 1.8, y0 + h * 0.15 + i * 0.01]); } return pts; };
    P.p(poly([[-31, 30], [31, 30], [31, 33], [29, 33], [29, 36], [-29, 36], [-29, 33], [-31, 33]]), 'w-white k');
    P.p(poly([[-26.5, 36], [26.5, 36], [26.5, 38.6], [24.5, 38.6], [24.5, 41], [-24.5, 41], [-24.5, 38.6], [-26.5, 38.6]]), 'w-white k');
    P.p(poly([[-22.5, 41], [22.5, 41], [22.5, 43], [21, 43], [21, 45], [-21, 45], [-21, 43], [-22.5, 43]]), 'w-white k');
    P.p('M-31 31.5H31M-26.5 37.3H26.5M-22.5 42H22.5', 'kt');
    P.p(poly([[-20, 45], [20, 45], [18.5, 48.4], [-18.5, 48.4]]), 'w-white kf');
    const belly = 'M-18.5 48.4C-27.5 56-27 75-16.5 84.4C-9 90.4 9 90.4 16.5 84.4C27 75 27.5 56 18.5 48.4Z';
    P.p(belly, 'w-white k');
    P.p('M7 49.2C17 55.6 19 73 13.6 82.6C19.6 76 20.4 57 7 49.2Z', 'w-shade');
    P.p('M-6.4 55.6H6.4V66.4C6.4 71.4 3.2 74.6 0 76.6C-3.2 74.6-6.4 71.4-6.4 66.4Z', 'w-col k');
    P.p('M-3.8 58H3.8V66.4C3.8 69.6 1.9 71.6 0 72.8C-1.9 71.6-3.8 69.6-3.8 66.4Z', 'w-gold kf');
    P.r(-9.5, 88.4, 19, 3.2, 'w-white k');
    P.p(poly([[-8, 91.6], [8, 91.6], [4.2, 120], [-4.2, 120]]), 'w-white k');
    let rings = '';
    for (let i = 1; i < 13; i++) { const y = 91.6 + 28.4 * i / 13, h = 8 - 3.8 * i / 13; rings += `M${N(-h)} ${N(y)}H${N(h)}`; }
    P.p(rings, 'kf');
    P.p('M-13 121.6L13 121.6L9.6 125.6L-9.6 125.6Z', 'w-gold k');
    let bells = ''; for (const u of [-12, -7, 7, 12]) bells += `M${u} 121.6v-2.2`; P.p(bells, 'kf');
    for (const u of [-12, -7, 7, 12]) P.c(u, 119, 0.7, 'w-gold kf');
    P.c(0, 128.6, 2.2, 'w-gold k');
    P.p('M-3 133Q0 129.6 3 133Q0 131.6-3 133Z', 'w-gold k');
    P.c(0, 135.4, 1.3, 'w-gold k');
    return { g: P.s(), box: [-64, 64, 0, 138] };
  }

  // 正阳门箭楼（城台门洞 + 四层箭窗，灰瓦绿剪边重檐歇山）
  function jianlou() {
    const P = Pen();
    P.p(poly([[-62, 0], [62, 0], [58, 30], [-58, 30]]), 'w-brick k');
    courses(P, -62, 62, 0, 30, 3.3, 4);
    P.p('M-8.5 0V15A8.5 8.5 0 0 0 8.5 15V0Z', 'w-dark k');
    P.p(poly([[-52, 30], [52, 30], [50, 70], [-50, 70]]), 'w-brick k');
    let win = '', glass = '';
    for (const y of [35.6, 44.6, 53.6, 62.6]) for (let i = 0; i < 13; i++) {
      const x = -43.2 + i * 7.2;
      win += `M${N(x - 2)} ${N(y)}h4v4.8h-4Z`;
      glass += `M${N(x - 1.1)} ${N(y + 0.8)}h2.2v3.2h-2.2Z`;
    }
    P.p(win, 'w-white kf');
    P.p(glass, 'w-dark');
    roof(P, 70, 58, 6.2, 48, { noRidge: true, w: 'w-roof-g', trim: 'w-roof-gr', lift: 2.6, band: 1.3, tile: 2, shadeW: 50 });
    body(P, 76.2, 84, 44, { cols: 11, lattice: false, w: 'w-brick', lintel: 1.3, cw: 1 });
    roof(P, 84, 60, 21, 30, { w: 'w-roof-g', trim: 'w-roof-gr', lift: 3, band: 1.6, tile: 2.2, shadeW: 44 });
    return { g: P.s(), box: [-66, 66, 0, 112] };
  }

  // CBD 天际线：中国尊、国贸三期和几栋塔楼（远景，用浅玻璃色）
  function cbd() {
    const P = Pen();
    const tower = (x, hw, h, top = 'flat') => {
      const d = top === 'slant' ? poly([[x - hw, 0], [x + hw, 0], [x + hw, h - hw * 0.9], [x - hw, h]]) : poly([[x - hw, 0], [x + hw, 0], [x + hw, h], [x - hw, h]]);
      P.p(d, 'w-glass k');
      let g = ''; for (let u = x - hw + 2.6; u < x + hw - 1; u += 2.6) g += `M${N(u)} 0V${N(top === 'slant' ? h - hw * 0.9 * (u - x + hw) / (2 * hw) : h)}`;
      for (let y = 8; y < h - 4; y += 8) g += `M${N(x - hw)} ${N(y)}H${N(x + hw)}`;
      P.p(g, 'kt');
    };
    tower(-58, 11, 118, 'slant');
    tower(54, 10, 132);
    tower(-36, 12, 170);
    // 国贸三期：塔身 + 收分的塔冠
    P.p(poly([[-24, 0], [-4, 0], [-4, 196], [-7, 204], [-21, 204], [-24, 196]]), 'w-glass k');
    let g3 = ''; for (let u = -21.5; u < -5; u += 2.5) g3 += `M${N(u)} 0V196`; for (let y = 9; y < 196; y += 9) g3 += `M-24 ${y}H-4`;
    P.p(g3, 'kt');
    P.p('M-22 204L-19 214H-9L-6 204', 'w-glass k');
    // 中国尊：底宽、腰细、顶宽
    const R = [[34, 0], [25.5, 88], [25.5, 170], [31, 256]], L = R.map(([x, y]) => [24 - x, y]);
    const cz = [[34, 0]].concat([]);
    const shape = `M${pt(L[0])}H${N(R[0][0])}C${pt(R[1])} ${pt(R[2])} ${pt(R[3])}H${N(L[3][0])}C${pt(L[2])} ${pt(L[1])} ${pt(L[0])}Z`;
    P.p(shape, 'w-glass k');
    const rp = samp(R), lp = samp(L);
    let gz = '';
    for (let y = 10; y < 252; y += 10) { const xr = yAt(rp.map(([x, yy]) => [yy, x]), y), xl = 24 - xr; gz += `M${N(xl)} ${N(y)}H${N(xr)}`; }
    for (let f = -0.8; f <= 0.81; f += 0.2) gz += 'M' + rp.map(([x, y]) => [12 + (x - 12) * f, y]).filter((_, i) => i % 3 === 0 || i === rp.length - 1).map(pt).join('L');
    P.p(gz, 'kt');
    P.p(`M${N(L[3][0])} 256L${N(L[3][0] + 3)} 263H${N(R[3][0] - 3)}L${N(R[3][0])} 256`, 'w-glass k');
    return { g: P.s(), box: [-70, 66, 0, 266] };
  }

  // 长城：山脊上的城墙与敌楼。返回宽景，需要指定宽度
  function changcheng(W = 420, seed = 1) {
    const P = Pen();
    const ridge = u => 34 + 14 * Math.sin(u / 41 + seed) + 8 * Math.sin(u / 17 + seed * 2.1) + 4 * Math.sin(u / 7.3 + seed);
    const back = u => 44 + 18 * Math.sin(u / 57 + seed * 0.7) + 9 * Math.sin(u / 23 + seed * 1.3);
    const half = W / 2, pts = [], bpts = [];
    for (let u = -half; u <= half + 0.01; u += 3) { pts.push([u, ridge(u)]); bpts.push([u, back(u)]); }
    P.p('M' + [[-half, 0]].concat(bpts, [[half, 0]]).map(pt).join('L') + 'Z', 'w-far2 k');
    P.p('M' + [[-half, 0]].concat(pts, [[half, 0]]).map(pt).join('L') + 'Z', 'w-hill k');
    // 城墙：沿山脊的一条带
    const wall = pts.map(([u, v]) => [u, v + 4.4]);
    P.p('M' + pts.map(pt).join('L') + 'L' + wall.slice().reverse().map(pt).join('L') + 'Z', 'w-brick k');
    let cr = ''; for (let i = 0; i < wall.length; i++) if (i % 1 === 0) cr += `M${N(wall[i][0])} ${N(wall[i][1])}v1.5`;
    P.p(cr, 'kf');
    // 敌楼：放在局部高点
    for (let i = 2; i < pts.length - 2; i++) {
      const [u, v] = pts[i];
      if (v > pts[i - 1][1] && v > pts[i + 1][1] && v > pts[i - 2][1] && v > pts[i + 2][1] && v > 30) {
        P.r(u - 6, v, 12, 11, 'w-brick k');
        let c2 = `M${N(u - 6)} ${N(v + 11)}`; for (let k = 0; k < 4; k++) c2 += `h1.6v1.5h1.4v-1.5`;
        P.p(c2, 'kf');
        P.p(`M${N(u - 3.4)} ${N(v + 3)}v3.4a1.1 1.1 0 0 0 2.2 0v-3.4ZM${N(u + 1.2)} ${N(v + 3)}v3.4a1.1 1.1 0 0 0 2.2 0v-3.4Z`, 'w-dark kf');
      }
    }
    return { g: P.s(), box: [-half, half, 0, 72] };
  }

  // 树：几团重叠的树冠，trees(P, [[u, v, r], ...], 填色)
  function trees(P, list, cls = 'w-tree') {
    list.forEach(([u, v, r], i) => {
      const k = (i * 37 % 10) / 10;
      P.p(`M${N(u)} ${N(v)}L${N(u + r * 0.05)} ${N(v + r * 1.1)}`, 'kf');
      const blobs = [[0, 1.55, 0.95], [-0.62, 1.2, 0.7 + k * 0.1], [0.66, 1.25, 0.72 - k * 0.1], [0.12, 2.05, 0.62]];
      let d = '';
      for (const [dx, dy, rr] of blobs) { const cx = u + dx * r, cy = v + dy * r, R = rr * r; d += `M${N(cx - R)} ${N(cy)}a${N(R)} ${N(R)} 0 1 0 ${N(R * 2)} 0a${N(R)} ${N(R)} 0 1 0 ${N(-R * 2)} 0Z`; }
      P.p(d, `${typeof cls === 'function' ? cls(i) : cls} kf`);
    });
  }

  // 松：层层平展的松冠
  function pine(P, u, v, h) {
    P.p(`M${N(u)} ${N(v)}C${N(u - h * 0.04)} ${N(v + h * 0.4)} ${N(u + h * 0.06)} ${N(v + h * 0.7)} ${N(u + h * 0.02)} ${N(v + h)}`, 'k');
    for (const [dy, w, dx] of [[0.42, 0.34, -0.06], [0.62, 0.3, 0.08], [0.8, 0.24, -0.02], [0.96, 0.16, 0.02]]) {
      const cx = u + h * dx, cy = v + h * dy, ww = h * w, hh = h * 0.09;
      P.p(`M${N(cx - ww)} ${N(cy)}C${N(cx - ww * 0.7)} ${N(cy + hh * 1.6)} ${N(cx + ww * 0.7)} ${N(cy + hh * 1.6)} ${N(cx + ww)} ${N(cy)}C${N(cx + ww * 0.5)} ${N(cy - hh * 0.6)} ${N(cx - ww * 0.5)} ${N(cy - hh * 0.6)} ${N(cx - ww)} ${N(cy)}Z`, 'w-tree kf');
    }
  }

  /* ── 红色地标 ───────────────────────────────────────── */

  // 卢沟桥：十一孔联拱石桥，望柱上蹲着石狮
  function lugou() {
    const P = Pen();
    P.p('M-128 0H128V-7H-128Z', 'w-water');
    let rip = ''; for (let i = 0; i < 18; i++) { const x = -120 + i * 14 + (i % 2) * 5; rip += `M${x} ${-2.4 - (i % 3)}h${6 + (i % 4)}`; } P.p(rip, 'kt');
    const spans = [15, 16.2, 17.4, 18.6, 19.8, 21, 19.8, 18.6, 17.4, 16.2, 15], pier = 5.2;
    const total = spans.reduce((a, b) => a + b, 0) + pier * (spans.length + 1);
    let x = -total / 2, holes = '';
    const piers = [];
    for (const s of spans) {
      piers.push(x); x += pier;
      const rise = s * 0.62;
      holes += `M${N(x)} 0V${N(rise - s / 2)}A${N(s / 2)} ${N(s / 2)} 0 0 0 ${N(x + s)} ${N(rise - s / 2)}V0Z`;
      x += s;
    }
    piers.push(x);
    const L = -total / 2 - 6, R = total / 2 + 6;
    P.p(`M${N(L)} 0L${N(-total / 2)} 0` + `V18H${N(total / 2)}V0L${N(R)} 0L${N(R - 3)} 21H${N(L + 3)}Z` + holes, 'w-stone k', ' fill-rule="evenodd"');
    let cut = ''; for (const p of piers.slice(1, -1)) cut += `M${N(p + pier * 0.15)} 0L${N(p + pier / 2)} 3.2L${N(p + pier * 0.85)} 0`;
    P.p(cut, 'kt');
    P.p(`M${N(L + 3)} 18H${N(R - 3)}`, 'kf');
    // 栏杆与石狮
    P.r(L + 3, 21, R - L - 6, 4.2, 'w-stone kf');
    let posts = '', lions = '';
    for (let u = L + 4; u <= R - 4; u += 4.6) { posts += `M${N(u)} 21V26.6`; lions += `M${N(u - 1)} 26.6C${N(u - 1.2)} 28.4 ${N(u + 1.4)} 29.2 ${N(u + 1.2)} 26.6Z`; }
    P.p(posts, 'kf');
    P.p(lions, 'w-stone kf');
    // 两端华表
    for (const u of [L - 4, R + 4]) {
      P.r(u - 1.3, 0, 2.6, 38, 'w-stone k');
      P.p(`M${N(u - 5)} 33.5H${N(u + 5)}`, 'k');
      P.p(`M${N(u - 2.6)} 38H${N(u + 2.6)}V39.6H${N(u - 2.6)}Z`, 'w-stone kf');
      P.p(`M${N(u - 1.1)} 39.6C${N(u - 1.4)} 42 ${N(u + 1.4)} 42.4 ${N(u + 1.1)} 39.6Z`, 'w-stone kf');
    }
    return { g: P.s(), box: [L - 10, R + 10, -8, 44] };
  }

  // 北大红楼：四层红砖楼，中间是入口
  function honglou() {
    const P = Pen();
    P.r(-92, 0, 184, 2.6, 'w-brick k');
    P.r(-90, 2.6, 180, 44, 'w-redbrick k');
    P.p('M-90 13.6H90M-90 24.6H90M-90 35.6H90', 'kt');
    let win = '', glass = '';
    for (const y of [5.6, 16.6, 27.6, 38.6]) for (let i = 0; i < 19; i++) {
      const u = -81 + i * 9; if (Math.abs(u) < 5 && y < 10) continue;
      win += `M${N(u - 2.3)} ${N(y)}h4.6v6.2h-4.6Z`; glass += `M${N(u - 1.5)} ${N(y + 0.8)}h3v4.6h-3Z`;
    }
    P.p(win, 'w-white kf');
    P.p(glass, 'w-glass');
    P.p('M-10 2.6V48.6H10V2.6', 'kf');
    P.p('M-4.2 2.6V9.6A4.2 4.2 0 0 0 4.2 9.6V2.6Z', 'w-dark k');
    P.p('M-7 0H7V-1.6H-7Z', 'w-stone kf');
    P.r(-92, 46.6, 184, 2.4, 'w-stone k');
    P.p(poly([[-91, 49], [91, 49], [80, 58], [-80, 58]]), 'w-roof-g k');
    let dm = ''; for (const u of [-54, -27, 27, 54]) dm += `M${u - 3} 50.6V54.6L${u} 57L${u + 3} 54.6V50.6Z`;
    P.p(dm, 'w-white kf');
    P.p('M-12 49V58.6H12V49', 'kf');
    P.p(poly([[-12, 58.6], [12, 58.6], [0, 63]]), 'w-roof-g k');
    return { g: P.s(), box: [-94, 94, -2, 64] };
  }

  // 李大钊故居：胡同里的四合院门楼
  function siheyuan() {
    const P = Pen();
    for (const s of [-1, 1]) {
      const x0 = s < 0 ? -60 : 17;
      P.r(x0, 0, 43, 25, 'w-brick k');
      courses(P, x0, x0 + 43, 0, 25, 2.8);
      P.p(poly([[x0 - 1, 25], [x0 + 44, 25], [x0 + 43, 28.6], [x0, 28.6]]), 'w-roof-g k');
    }
    trees(P, [[-42, 22, 7.4], [-30, 24, 5.2], [44, 22, 6.8]], 'w-tree');
    P.r(-17, 0, 34, 29.5, 'w-brick k');
    courses(P, -17, 17, 0, 29.5, 2.8);
    P.r(-11, 2.6, 22, 22.4, 'w-stone kf');
    P.r(-8.6, 3, 17.2, 20.6, 'w-col k');
    P.p('M0 3V23.6', 'kf');
    P.c(-1.7, 13.4, 0.75, 'w-gold kf'); P.c(1.7, 13.4, 0.75, 'w-gold kf');
    P.p('M-6 24.6h3.4v2.2h-3.4ZM2.6 24.6h3.4v2.2h-3.4Z', 'w-gold kf');
    for (const u of [-10.6, 10.6]) { P.r(u - 2, 0, 4, 2.2, 'w-stone kf'); P.c(u, 4.4, 2.1, 'w-stone kf'); P.c(u, 4.4, 1.1, 'kt'); }
    P.r(-13, 0, 26, 0.2, 'kf');
    roof(P, 29.5, 24, 10.6, 19.5, { w: 'w-roof-g', ridge: 'xie', lift: 1, band: 1.2, tile: 1.5, shadeW: 17 });
    return { g: P.s(), box: [-62, 62, 0, 46] };
  }

  // 香山：山峦、红叶，半山一座亭子
  function xiangshan() {
    const P = Pen();
    P.p('M-88 0C-72 8-60 36-40 50C-30 57-20 56-12 49C-4 42 4 46 12 53C22 61 34 62 44 52C58 38 72 10 88 0Z', 'w-hill k');
    P.p('M-62 8C-52 18-46 30-38 38M40 44C48 36 56 22 62 10', 'kt');
    const crowns = [[-70, 2, 4.6], [-60, 8, 5.4], [-50, 16, 5], [-41, 25, 5.6], [-31, 32, 5], [-20, 36, 4.6], [-48, 2, 4.4], [-36, 9, 5.2], [-24, 16, 5.4], [-12, 22, 5], [-2, 26, 4.6],
      [-28, 1, 4.2], [-14, 4, 4.8], [0, 8, 5.2], [12, 14, 5], [24, 20, 5.4], [36, 26, 5], [48, 22, 5.2], [58, 14, 5.4], [68, 6, 4.8], [10, 1, 4.4], [24, 4, 4.8], [38, 8, 5], [52, 4, 4.6], [76, 1, 4]];
    const tone = i => ['w-maple', 'w-maple', 'w-roof-y', 'w-maple', 'w-tree', 'w-maple', 'w-roof-y'][i % 7];
    trees(P, crowns, tone);
    // 半山亭
    P.g(sub(Q => {
      Q.r(-8, 0, 16, 1.8, 'w-stone kf');
      Q.p('M-6 1.8v8.4M-2 1.8v8.4M2 1.8v8.4M6 1.8v8.4', 'k');
      roof(Q, 10.2, 11, 7.4, 0, { lift: 1.4, band: 0.9, tile: 1.3, shade: false });
      baoding(Q, 17.6, 0.6);
    }), ' transform="translate(26 53)"');
    return { g: P.s(), box: [-90, 90, 0, 76] };
  }

  // 中国共产党历史展览馆：高台阶上的柱廊
  function dangshi() {
    const P = Pen();
    P.p(poly([[-104, 0], [104, 0], [100, 7.2], [-100, 7.2]]), 'w-stone k');
    P.p('M-103 2.4H103M-101.6 4.8H101.6', 'kt');
    P.p(poly([[-26, 0], [26, 0], [24, 7.2], [-24, 7.2]]), 'w-stone kf');
    let st = ''; for (let y = 1; y < 7.2; y += 1.2) st += `M-25.6 ${N(y)}H25.6`; P.p(st, 'kt');
    P.r(-94, 7.2, 188, 40, 'w-stone k');
    P.r(-88, 7.2, 176, 40, 'w-shade');
    let cols = '';
    for (let i = 0; i < 18; i++) { const u = -86 + i * (172 / 17); cols += `M${N(u - 2.3)} 7.2h4.6v40h-4.6Z`; }
    P.p(cols, 'w-white kf');
    P.r(-97, 47.2, 194, 5, 'w-stone k');
    P.p('M-97 49.4H97', 'kt');
    P.p(poly([[-101, 52.2], [101, 52.2], [97, 57.2], [-97, 57.2]]), 'w-stone k');
    P.r(-60, 57.2, 120, 4.4, 'w-stone k');
    return { g: P.s(), box: [-106, 106, 0, 62] };
  }

  // 大兴机场航站楼：舒展的波浪形屋面 + 玻璃幕墙（南中轴延长线上的"新国门"）
  function daxing() {
    const P = Pen();
    P.p('M-112 0H112V3H-112Z', 'w-stone kf');
    P.p('M-104 3V17C-70 24-36 28 0 28C36 28 70 24 104 17V3Z', 'w-glass k');
    let m = ''; for (let u = -100; u <= 100; u += 4) { const h = 17 + 11 * (1 - (u / 104) ** 2); m += `M${u} 3V${N(h - 1)}`; }
    P.p(m, 'kt');
    P.p('M-104 10.5C-70 16-36 19 0 19C36 19 70 16 104 10.5', 'kt');
    P.p('M-118 15C-80 25-40 31.6 0 31.6C40 31.6 80 25 118 15L114 13C78 22 40 28.2 0 28.2C-40 28.2-78 22-114 13Z', 'w-white k');
    let r = ''; for (let u = -96; u <= 96; u += 12) { const y = 28.2 + 3.4 * (1 - (u / 118) ** 2); r += `M${u} ${N(y - 3.2)}L${u} ${N(y)}`; }
    P.p(r, 'kt');
    for (const u of [-40, 40]) P.p(`M${u - 4} 3C${u - 3} 12 ${u - 1} 18 ${u} 24C${u + 1} 18 ${u + 3} 12 ${u + 4} 3`, 'kf');
    return { g: P.s(), box: [-120, 120, 0, 34] };
  }

  // 柏：天坛古柏那样的窄长树冠
  function cypress(P, u, v, h) {
    P.p(`M${N(u)} ${N(v)}V${N(v + h * 0.18)}`, 'kf');
    P.p(`M${N(u)} ${N(v + h)}C${N(u + h * 0.2)} ${N(v + h * 0.78)} ${N(u + h * 0.2)} ${N(v + h * 0.3)} ${N(u + h * 0.12)} ${N(v + h * 0.16)}C${N(u + h * 0.04)} ${N(v + h * 0.1)} ${N(u - h * 0.04)} ${N(v + h * 0.1)} ${N(u - h * 0.12)} ${N(v + h * 0.16)}C${N(u - h * 0.2)} ${N(v + h * 0.3)} ${N(u - h * 0.2)} ${N(v + h * 0.78)} ${N(u)} ${N(v + h)}Z`, 'w-tree kf');
    P.p(`M${N(u)} ${N(v + h * 0.9)}V${N(v + h * 0.22)}`, 'kt');
  }

  const LANDMARKS = { qinian, jiaolou, foxiangge, baita, jianlou, cbd, changcheng, daxing, lugou, honglou, siheyuan, xiangshan, dangshi };

  /* ── 组合：首屏北京长卷 ─────────────────────────────── */
  const W = 2400, H = 600, G = 500;
  // 祥云（屏幕坐标）
  const cloud = (x, y, s, flip = 1) => `<g class="cloud" transform="translate(${x} ${y}) scale(${N(s * flip)} ${N(s)})"><path class="w-white kf" d="M-30 8C-34 8-35 2-30 0C-31-6-24-10-18-7C-15-14-4-15 1-9C5-14 15-14 18-7C25-8 30-2 27 3C31 4 32 8 28 9Z"/><path class="kt" d="M-18-7C-18-3-14-1-11-3M1-9C1-5 4-3 7-4M18-7C17-4 19-1 22-1M-30 0C-27 2-24 1-23-1"/><path class="kf" d="M28 9C40 9 52 7 64 3M-30 8C-40 8-48 9-56 12"/></g>`;
  const bird = (x, y, s) => `<path class="kf bird" d="M${x} ${y}q${N(3 * s)} ${N(-2.6 * s)} ${N(6 * s)} 0q${N(3 * s)} ${N(-2.6 * s)} ${N(6 * s)} 0"/>`;
  function plane(x, y, s, rot) {
    return `<g class="plane" transform="translate(${x} ${y}) rotate(${rot}) scale(${s})"><path class="w-white k" d="M-22 0C-22-2.6-18-3.4-14-3.4H14C20-3.4 24-1.4 25 0C24 1.4 20 2.2 14 2.2H-14C-18 2.2-22 1.8-22 0Z"/><path class="w-white k" d="M-2-2.6L-8-13H-4.4L6-2.6ZM-2 1.8L-6 8.4H-3.4L4 1.8Z"/><path class="w-white k" d="M-19-2.8L-23-10H-20.4L-14-3.2Z"/><path class="kt" d="M16-1.6h5"/></g>`;
  }
  function panorama() {
    const far = sub(P => {
      P.g(LANDMARKS.changcheng(1180, 1.2).g, ` transform="translate(560 ${G}) scale(1.5 -1.5)"`);
      P.g(LANDMARKS.changcheng(1400, 3.4).g.replace(/<path class="w-brick k"[\s\S]*$/, ''), ` transform="translate(1720 ${G}) scale(1.35 -1.1)"`);
    });
    const P = Pen();
    P.p(`M0 0H${W}V${H}H0Z`, 'sky', ' fill="url(#art-sky)"');
    P.c(1290, 318, 150, 'sun');
    P.g(far, ' class="far"');
    P.g(place('cbd', 2170, G, 292), ' class="far"');
    P.g(cloud(520, 158, 1.25) + cloud(1600, 112, 1) + cloud(860, 96, 0.8, -1) + cloud(2310, 150, 0.9, -1), ' class="clouds"');
    // 中景地标
    const mid = sub(Q => {
      for (const [u, h] of [[130, 64], [168, 52], [462, 58], [790, 46], [1104, 70], [1140, 56], [1472, 72], [1508, 58], [1832, 60], [1868, 48]]) cypress(Q, u, -G, h);
    });
    P.g(mid, ' transform="scale(1 -1)"');
    let s = P.s();
    s += place('foxiangge', 300, G, 252) + place('baita', 612, G, 188) + place('jiaolou', 932, G, 168) + place('qinian', 1290, G, 258) + place('jianlou', 1650, G, 182) + place('daxing', 1960, G, 52);
    s += `<g class="birds">${bird(858, 226, 1.3)}${bird(890, 212, 1)}${bird(912, 238, 0.85)}</g>`;
    s += `<path class="route" d="M2010 466C2040 400 2080 330 2140 270S2290 170 2400 132"/>` + plane(2140, 268, 1.3, -38);
    // 水岸
    s += `<path d="M0 ${G}H${W}V${H}H0Z" fill="url(#art-water)"/><path class="k" d="M0 ${G}H${W}"/>`;
    let rip = ''; for (let i = 0; i < 60; i++) { const x = (i * 157) % W, y = G + 18 + (i * 37) % 70, w = 14 + (i * 13) % 34; rip += `M${x} ${y}h${w}`; }
    s += `<path class="kt" d="${rip}"/>`;
    return `<svg class="art pano-svg" viewBox="0 0 ${W} ${H}" preserveAspectRatio="xMidYMax slice" role="img" aria-label="北京地标长卷：颐和园佛香阁、北海白塔、故宫角楼、天坛祈年殿、正阳门箭楼、大兴机场与 CBD">${s}</svg>`;
  }

  // 把一个地标放到 (x,y)，按高度 h 缩放
  function place(name, x, y, h, cls = '') {
    const L = LANDMARKS[name]();
    const s = h / (L.box[3] - L.box[2]);
    return `<g class="lm lm-${name} ${cls}" transform="translate(${N(x)} ${N(y)}) scale(${N(s)} ${N(-s)})">${L.g}</g>`;
  }

  window.ART = { LANDMARKS, place, defs, panorama, plane, cloud, Pen, roof, xsGable, body, terrace, stairs, gable, baoding, poly, N, sub, trees, pine, cypress };
})();
