/* 界画库 · 导览图小图标（北京研学导览图用）
   每处景点一枚小界画：同一套墨线和矿物色，比正式地标简化，单枚只有几 KB。
   局部坐标 u 右 v 上，原点在图标底边中点（落地处），图标大致在宽 120、高 100 以内。
   画法：先用几样小构件（屋顶、殿身、台基、城墙、券门、山、水、树），再拼成各处景点。 */
(function () {
  'use strict';
  const { N, P2, Pen, poly, smooth, samp, rng } = window.JH;

  /* ── 小构件 ─────────────────────────────────── */
  // 屋顶：檐口两端起翘，戗脊凹曲到正脊；type：wd 庑殿、xs 歇山（长边）、xsg 歇山山面、cj 攒尖、ys 硬山
  function roof(P, cx, y, w, h, o = {}) {
    const tone = o.tone || 'yellow', type = o.type || 'wd', hw = w / 2, lift = o.lift ?? h * 0.28;
    const rw = type === 'cj' ? 0 : (o.rw ?? hw * 0.42), ov = h * 0.1;
    const tL = [cx - hw - ov, y + lift], tR = [cx + hw + ov, y + lift];
    const top = y + h;
    let d = `M${P2(...tL)}Q${P2(cx - hw * 0.78, y)} ${P2(cx - hw * 0.5, y)}L${P2(cx + hw * 0.5, y)}Q${P2(cx + hw * 0.78, y)} ${P2(...tR)}`;
    const hip = (s, X, Y) => { const T = s > 0 ? tR : tL; return `C${P2(T[0] - (T[0] - X) * 0.55, T[1] + h * 0.04)} ${P2(X + (T[0] - X) * 0.12, Y - (Y - T[1]) * 0.45)} ${P2(X, Y)}`; };
    const hipBack = (s, X, Y) => { const T = s > 0 ? tR : tL; return `C${P2(X + (T[0] - X) * 0.12, Y - (Y - T[1]) * 0.45)} ${P2(T[0] - (T[0] - X) * 0.55, T[1] + h * 0.04)} ${P2(...T)}`; };
    if (type === 'xs') { const G = rw + h * 0.16, gy = y + h * 0.46; d += hip(1, cx + G, gy) + `L${P2(cx + rw, top)}L${P2(cx - rw, top)}L${P2(cx - G, gy)}` + hipBack(-1, cx - G, gy); }
    else if (type === 'xsg') { const gw = o.gw ?? hw * 0.5, gy = y + h * 0.42; d += hip(1, cx + gw, gy) + `Q${P2(cx + gw * 0.3, gy + (top - gy) * 0.5)} ${P2(cx, top)}Q${P2(cx - gw * 0.3, gy + (top - gy) * 0.5)} ${P2(cx - gw, gy)}` + hipBack(-1, cx - gw, gy); }
    else if (type === 'ys') { d += `L${P2(cx + hw * 0.9, top - h * 0.2)}L${P2(cx + rw, top)}L${P2(cx - rw, top)}L${P2(cx - hw * 0.9, top - h * 0.2)}Z`; }
    else d += hip(1, cx + rw, top) + (rw > 0 ? `L${P2(cx - rw, top)}` : '') + hipBack(-1, cx - rw, top);
    P.p(d + 'Z', `g-${tone} l2`);
    // 瓦垄几道
    let t = ''; const n = o.tiles ?? Math.max(3, Math.round(w / 9));
    for (let i = 1; i < n; i++) { const x = cx - hw * 0.86 + hw * 1.72 * i / n, a = Math.abs(x - cx); const yt = type === 'xsg' && a < (o.gw ?? hw * 0.5) ? y + h * 0.4 : a <= rw ? top - h * 0.06 : y + h * (0.9 - 0.75 * (a - rw) / Math.max(1, hw - rw)); if (yt > y + h * 0.2) t += `M${P2(x, y + h * 0.16)}V${N(yt)}`; }
    P.p(t, 'l3');
    // 檐口带
    P.p(`M${P2(...tL)}Q${P2(cx - hw * 0.78, y)} ${P2(cx - hw * 0.5, y)}L${P2(cx + hw * 0.5, y)}Q${P2(cx + hw * 0.78, y)} ${P2(...tR)}L${P2(tR[0] - 1, tR[1] + h * 0.1)}Q${P2(cx + hw * 0.76, y + h * 0.13)} ${P2(cx + hw * 0.5, y + h * 0.13)}L${P2(cx - hw * 0.5, y + h * 0.13)}Q${P2(cx - hw * 0.76, y + h * 0.13)} ${P2(tL[0] + 1, tL[1] + h * 0.1)}Z`, `c-${o.trim || tone}-d l3`);
    if (type === 'xsg') { const gw = o.gw ?? hw * 0.5, gy = y + h * 0.42; P.p(`M${P2(cx - gw * 0.82, gy + h * 0.02)}Q${P2(cx - gw * 0.25, gy + (top - gy) * 0.5)} ${P2(cx, top - h * 0.1)}Q${P2(cx + gw * 0.25, gy + (top - gy) * 0.5)} ${P2(cx + gw * 0.82, gy + h * 0.02)}Z`, 'c-red l3'); }
    // 正脊、吻
    if ((type === 'wd' || type === 'xs') && rw > 0) { P.r(cx - rw, top - h * 0.04, rw * 2, h * 0.12, `c-${tone} l3`); for (const s of [-1, 1]) P.p(`M${P2(cx + s * rw, top)}c${N(s * h * 0.02)} ${N(h * 0.18)} ${N(s * h * 0.16)} ${N(h * 0.28)} ${N(s * h * 0.2)} ${N(h * 0.18)}c${N(-s * h * 0.06)} ${N(-h * 0.02)} ${N(-s * h * 0.08)} ${N(-h * 0.1)} ${N(-s * h * 0.06)} ${N(-h * 0.18)}Z`, `c-${tone} l3`); }
    if (type === 'cj' && o.finial !== false) { P.c(cx, top + h * 0.12, h * 0.1, 'g-gold l3'); P.r(cx - h * 0.04, top - h * 0.02, h * 0.08, h * 0.06, 'g-gold'); }
    return top;
  }
  // 圆攒尖（祈年殿一类）一层：檐口微拱，屋面凹曲收到上一层
  function roundRoof(P, cx, y, re, rt, h, tone = 'blue') {
    const d = `M${P2(cx - re, y)}Q${P2(cx, y + h * 0.2)} ${P2(cx + re, y)}C${P2(cx + re - (re - rt) * 0.55, y + h * 0.08)} ${P2(cx + rt + (re - rt) * 0.1, y + h * 0.6)} ${P2(cx + rt, y + h)}L${P2(cx - rt, y + h)}C${P2(cx - rt - (re - rt) * 0.1, y + h * 0.6)} ${P2(cx - re + (re - rt) * 0.55, y + h * 0.08)} ${P2(cx - re, y)}Z`;
    P.p(d, `g-${tone} l2`);
    let m = ''; for (const f of [-0.66, -0.33, 0, 0.33, 0.66]) m += `M${P2(cx + re * f, y + h * 0.1 * (1 - f * f))}Q${P2(cx + (re + rt) * 0.5 * f, y + h * 0.5)} ${P2(cx + rt * f, y + h)}`;
    P.p(m, 'l3');
    P.p(`M${P2(cx - re, y)}Q${P2(cx, y + h * 0.2)} ${P2(cx + re, y)}L${P2(cx + re - 1, y + h * 0.12)}Q${P2(cx, y + h * 0.3)} ${P2(cx - re + 1, y + h * 0.12)}Z`, `c-${tone}-d l3`);
  }
  // 殿身：红墙或隔扇，几根柱，正中门，顶上一道青绿额枋
  function body(P, cx, y, w, h, o = {}) {
    P.r(cx - w / 2, y, w, h, `${o.cls || 'c-red'} l2`);
    if (o.lattice !== false) P.r(cx - w / 2 + w * 0.06, y + h * 0.12, w * 0.88, h * 0.6, 'p-lattice');
    let c = ''; const n = o.cols ?? 4; for (let i = 0; i <= n; i++) { const x = cx - w / 2 + w * i / n; c += `M${P2(x, y)}V${N(y + h)}`; }
    P.p(c, 'l3');
    P.r(cx - w / 2 - 0.5, y + h * 0.82, w + 1, h * 0.18, 'c-qing l3');
    if (o.door) P.r(cx - w * 0.07, y, w * 0.14, h * 0.6, 'c-red-d l3');
  }
  function base(P, cx, y, w, h, cls = 'c-marble') { P.r(cx - w / 2, y, w, h, `${cls} l2`); P.p(`M${P2(cx - w / 2, y + h * 0.62)}h${N(w)}`, 'l3'); }
  function rail(P, x0, x1, y, h) { let d = `M${P2(x0, y + h)}H${N(x1)}`; for (let x = x0; x <= x1 + 0.1; x += h * 1.1) d += `M${P2(x, y)}V${N(y + h * 1.2)}`; P.p(d, 'l3'); }
  function stairs(P, cx, y, w, h) { P.r(cx - w / 2, y, w, h, 'c-marble l3'); let d = ''; for (let k = 1; k < 4; k++) d += `M${P2(cx - w / 2, y + h * k / 4)}h${N(w)}`; P.p(d, 'l3'); }
  // 城墙：墙身 + 顶上垛口
  function wall(P, x0, x1, y, h, cls = 'c-wallr', mer = true) {
    P.r(x0, y, x1 - x0, h, `${cls} l2`);
    P.r(x0, y, x1 - x0, h * 0.3, 'c-shade');
    if (mer) { const m = h * 0.22; let d = ''; for (let x = x0 + m * 0.3; x < x1 - m; x += m * 1.7) d += `M${P2(x, y + h)}h${N(m)}v${N(m * 0.9)}h${N(-m)}Z`; P.p(d, `${cls} l3`); }
  }
  function arch(P, cx, y, w, h) { P.p(`M${P2(cx - w / 2, y)}V${N(y + h - w / 2)}A${N(w / 2)} ${N(w / 2)} 0 0 0 ${P2(cx + w / 2, y + h - w / 2)}V${N(y)}Z`, 'c-ink'); }
  // 落地的一抹淡影
  function shadow(P, w) { P.e(0, 0.5, w / 2, Math.max(1.4, w * 0.06), 'c-shade'); }
  // 山：两三座青绿小峰
  function peaks(P, cx, y, w, h, o = {}) {
    const r = rng(o.seed || 5), list = o.peaks || [[-0.28, 0.72, 0.3], [0.05, 1, 0.34], [0.34, 0.62, 0.3]];
    for (const [px, ph, pw] of list) {
      const x0 = cx + px * w, hh = h * ph, ww = w * pw;
      const pts = samp(t => { const u = t * 2 - 1; return [x0 + u * ww, y + hh * Math.max(0, 1 - u * u) ** (o.sharp ?? 0.9) * (1 + (r() - 0.5) * 0.06)]; }, 14);
      pts[0][1] = y; pts[14][1] = y;
      P.p(smooth(pts) + 'Z', `${o.fill || 'g-shan'}`); P.p(smooth(pts), 'l2');
      let c = ''; for (const k of [0.35, 0.65]) c += `M${P2(x0 + (k - 0.5) * ww * 0.6, y + hh * 0.8)}q${N(ww * 0.06)} ${N(-hh * 0.3)} ${N(ww * 0.12 * (k < 0.5 ? -1 : 1))} ${N(-hh * 0.62)}`;
      P.p(c, 'l3');
    }
  }
  // 山丘：圆浑，上面几团树冠
  function hill(P, cx, y, w, h, o = {}) {
    const r = rng(o.seed || 9), pts = samp(t => { const u = t * 2 - 1; return [cx + u * w / 2, y + h * Math.max(0, 1 - u * u) ** 0.8]; }, 16);
    pts[0][1] = y; pts[16][1] = y;
    P.p(smooth(pts) + 'Z', `${o.fill || 'g-shan-mid'}`); P.p(smooth(pts), 'l2');
    const n = o.trees ?? Math.round(w / 12);
    const cp = window.JH.NATURE.clumpPath; let cl = '', hi = '';
    for (let i = 0; i < n; i++) { const t = (i + 0.5) / n, x = cx + (t - 0.5) * w * 0.86, yt = y + h * Math.max(0, 1 - ((x - cx) / (w / 2)) ** 2) ** 0.8, yy = y + (yt - y) * (0.4 + r() * 0.5), R = w * 0.065; cl += cp(x, yy, R, R * 0.72, r, 8); hi += `M${P2(x - R * 0.55, yy + R * 0.1)}Q${P2(x - R * 0.1, yy + R * 0.62)} ${P2(x + R * 0.45, yy + R * 0.28)}Q${P2(x, yy + R * 0.32)} ${P2(x - R * 0.55, yy + R * 0.1)}Z`; }
    P.p(cl, `${o.tree || 'c-tree'} l3`); P.p(hi, 'c-light');
  }
  // 水：一片椭圆水面，几道水纹
  function water(P, cx, y, w, h) {
    P.e(cx, y, w / 2, h / 2, 'c-water l3');
    let d = ''; for (const [fx, fy] of [[-0.3, -0.1], [0.15, 0.15], [0.25, -0.2], [-0.05, 0.25]]) { const x = cx + fx * w, yy = y + fy * h; d += `M${P2(x - w * 0.06, yy)}q${N(w * 0.03)} ${N(h * 0.08)} ${N(w * 0.06)} 0q${N(w * 0.03)} ${N(-h * 0.08)} ${N(w * 0.06)} 0`; }
    P.p(d, 'l-water');
  }
  // 小树：pine 松、bai 柏、maple 红叶、peach 桃花、willow 柳、round 杂树
  function tree(P, x, y, h, kind = 'round') {
    const tw = h * 0.05;
    P.p(`M${P2(x - tw, y)}L${P2(x - tw * 0.5, y + h * 0.55)}L${P2(x + tw * 0.5, y + h * 0.55)}L${P2(x + tw, y)}Z`, 'g-bark l3');
    if (kind === 'pine') { for (const [dy, w] of [[0.48, 0.42], [0.66, 0.34], [0.82, 0.24]]) P.p(`M${P2(x - h * w, y + h * dy)}A${N(h * w)} ${N(h * w * 0.45)} 0 0 0 ${P2(x + h * w, y + h * dy)}Z`, 'c-pine l3'); return; }
    if (kind === 'willow') { P.p(`M${P2(x - h * 0.34, y + h * 0.2)}C${P2(x - h * 0.4, y + h * 0.9)} ${P2(x + h * 0.4, y + h * 0.9)} ${P2(x + h * 0.34, y + h * 0.2)}Z`, 'c-willow l3'); let s = ''; for (let k = -2; k <= 2; k++) s += `M${P2(x + k * h * 0.12, y + h * 0.75)}V${N(y + h * 0.22)}`; P.p(s, 'l-willow-s'); return; }
    const cls = kind === 'bai' ? 'c-bai' : kind === 'maple' ? 'c-maple' : kind === 'peach' ? 'c-peach' : 'c-tree';
    P.p(`M${P2(x - h * 0.3, y + h * 0.5)}a${N(h * 0.3)} ${N(h * 0.26)} 0 1 0 ${N(h * 0.6)} 0a${N(h * 0.3)} ${N(h * 0.26)} 0 1 0 ${N(-h * 0.6)} 0Z`, `${cls} l3`);
    if (kind === 'peach' || kind === 'maple') { let d = ''; for (const [fx, fy] of [[-0.12, 0.6], [0.1, 0.48], [0.02, 0.66], [-0.16, 0.44], [0.16, 0.58]]) d += `M${P2(x + fx * h, y + fy * h)}h.01`; P.p(d, kind === 'peach' ? 'l-dot-w' : 'l-dot'); }
  }
  // 现代楼：玻璃幕墙一格一格
  function tower(P, x, y, w, h, o = {}) {
    P.p(o.d || poly([[x - w / 2, y], [x + w / 2, y], [x + w / 2, y + h], [x - w / 2, y + h]]), `${o.cls || 'c-glass'} l2`);
    let g = ''; for (let k = 1; k < Math.round(h / 4); k++) g += `M${P2(x - w / 2 + 0.6, y + k * 4)}h${N(w - 1.2)}`; for (let k = 1; k < 3; k++) g += `M${P2(x - w / 2 + w * k / 3, y + 1)}V${N(y + h - 1)}`;
    P.p(g, 'l-grid');
    P.p(poly([[x + w * 0.1, y], [x + w / 2, y], [x + w / 2, y + h], [x + w * 0.1, y + h]]), 'c-shade');
  }
  // 山脊顶线：按 peaks() 同样的峰形公式取上沿，城墙沿着走
  function ridgeOf(cx, y, w, h, list, sharp, x0, x1, step = 3, sink = 2.5) {
    const out = [];
    for (let x = x0; x <= x1 + 0.01; x += step) { let v = 0; for (const [px, ph, pw] of list) { const u = (x - (cx + px * w)) / (w * pw); v = Math.max(v, h * ph * Math.max(0, 1 - u * u) ** sharp); } out.push([x, y + v - sink]); }
    return out;
  }
  // 长城：墙身一条带子顺山脊起伏，上沿一排垛口，隔一段一座敌楼（方台、两个券窗、台顶垛口）
  function greatWall(P, pts, o = {}) {
    const th = o.h ?? 5, mer = th * 0.48;
    P.p(poly([...pts, ...pts.slice().reverse().map(([x, y]) => [x, y - th])]), 'c-gw l2');
    P.p('M' + pts.map(([x, y]) => P2(x, y - th * 0.45)).join('L'), 'l3');
    let m = ''; for (let i = 0; i < pts.length - 1; i++) { const [xa, ya] = pts[i], [xb, yb] = pts[i + 1], len = Math.hypot(xb - xa, yb - ya); for (let d = mer * 0.6; d < len - mer * 0.4; d += mer * 2) { const t = d / len, x = xa + (xb - xa) * t, y = ya + (yb - ya) * t; m += `M${P2(x - mer * 0.45, y - 0.3)}h${N(mer * 0.9)}v${N(mer)}h${N(-mer * 0.9)}Z`; } }
    P.p(m, 'c-gw l3');
    for (const i of o.towers || []) {
      const [x, y] = pts[Math.min(pts.length - 1, i)], w = th * 2.2, hh = th * 2.1, yb = y - th * 1.2;
      P.r(x - w / 2, yb, w, hh, 'c-gw l2');
      P.r(x + w * 0.1, yb, w * 0.4, hh, 'c-shade');
      P.p(`M${P2(x - w * 0.3, yb + hh * 0.3)}h${N(w * 0.18)}v${N(hh * 0.32)}h${N(-w * 0.18)}ZM${P2(x + w * 0.12, yb + hh * 0.3)}h${N(w * 0.18)}v${N(hh * 0.32)}h${N(-w * 0.18)}Z`, 'c-ink');
      let tm = ''; for (let k = 0; k < 3; k++) tm += `M${P2(x - w / 2 + w * (0.08 + k * 0.33), yb + hh)}h${N(w * 0.2)}v${N(mer)}h${N(-w * 0.2)}Z`;
      P.p(tm, 'c-gw l3');
      if (o.huts && o.huts.includes(i)) roof(P, x, yb + hh + mer, w * 0.9, th * 1.4, { type: 'xs', tone: 'gray', rw: w * 0.2, tiles: 3 });
    }
  }
  const out = (P, box) => ({ g: P.s(), box });

  /* ── 景点图标 ───────────────────────────────── */
  const I = {};
  // 故宫：宫城一角，栗红城墙上一座三重檐角楼
  I.gugong = () => { const P = Pen(); shadow(P, 116); wall(P, -58, 58, 0, 22); body(P, 10, 26, 46, 11, { cols: 4 }); roof(P, 10, 36, 66, 8, { type: 'wd', rw: 18 }); body(P, 10, 43, 34, 8, { cols: 3 }); roof(P, 10, 50, 54, 15, { type: 'xsg', gw: 13 }); roof(P, 10, 62, 42, 18, { type: 'xsg', gw: 10 }); P.c(10, 82, 2.4, 'g-gold l3'); return out(P, [-60, 60, -2, 86]); };
  // 天坛：三层圆坛、三重蓝檐、鎏金宝顶
  I.tiantan = () => { const P = Pen(); shadow(P, 104); base(P, 0, 0, 100, 6); base(P, 0, 6, 86, 6); base(P, 0, 12, 72, 6); rail(P, -34, 34, 18, 2.6); body(P, 0, 18, 40, 15, { cols: 5, door: true }); roundRoof(P, 0, 32, 31, 21, 9); body(P, 0, 41, 32, 5, { cols: 4, lattice: false }); roundRoof(P, 0, 45, 26, 16, 8); body(P, 0, 53, 24, 4, { cols: 3, lattice: false }); roundRoof(P, 0, 56, 21, 1.2, 16); P.c(0, 74, 2.6, 'g-gold l3'); P.r(-0.8, 70, 1.6, 3, 'g-gold'); return out(P, [-52, 52, -2, 78]); };
  // 颐和园：万寿山上佛香阁，山前昆明湖
  I.yiheyuan = () => { const P = Pen(); water(P, 0, 4, 116, 12); hill(P, 0, 8, 108, 34, { seed: 3 }); P.p(poly([[-20, 28], [20, 28], [17, 46], [-17, 46]]), 'c-stone l2'); P.r(-3, 28, 6, 18, 'c-marble l3'); body(P, 0, 46, 22, 8, { cols: 3 }); roof(P, 0, 53, 36, 7, { type: 'wd', rw: 6, trim: 'green' }); body(P, 0, 59, 18, 6, { cols: 3 }); roof(P, 0, 64, 30, 6, { type: 'wd', rw: 5, trim: 'green' }); body(P, 0, 69, 14, 5, { cols: 2 }); roof(P, 0, 73, 24, 13, { type: 'cj', trim: 'green' }); return out(P, [-60, 60, -4, 90]); };
  // 圆明园：大水法残迹，断券、残柱、散石
  I.yuanmingyuan = () => { const P = Pen(); shadow(P, 104); P.p('M-40 0V38L-34 44L-28 38H-14L-8 42L-2 36L4 40V0ZM-30 0V14A10 10 0 0 0 -10 14V0Z', 'c-marble l2', ' fill-rule="evenodd"'); P.p('M-36 6V30M-6 6V30M-20 26V34', 'l3'); P.r(14, 0, 7, 30, 'c-marble l2'); P.r(13, 30, 9, 4, 'c-marble l3'); P.r(30, 0, 6, 16, 'c-marble l2'); P.p('M40 0L45 4L50 0ZM-52 0L-47 5L-42 0Z', 'c-marble-d l3'); P.p('M16 6L18 26', 'l3'); return out(P, [-54, 54, -2, 48]); };
  // 北海：琼华岛上白塔，环水
  I.beihai = () => { const P = Pen(); water(P, 0, 3, 112, 12); hill(P, 0, 7, 92, 28, { seed: 5 }); P.r(-14, 30, 28, 6, 'c-marble l2'); P.p('M-11 36C-16 42 -16 54 -8 58L8 58C16 54 16 42 11 36Z', 'c-marble l2'); P.p('M-3 40V50Q0 54 3 50V40Z', 'c-red l3'); P.p('M-6 58L6 58L4 74L-4 74Z', 'c-marble l2'); let r = ''; for (let k = 1; k < 6; k++) r += `M${P2(-6 + k * 0.35, 58 + k * 2.8)}h${N(12 - k * 0.7)}`; P.p(r, 'l3'); P.p('M-9 74Q0 79 9 74Z', 'g-gold l3'); P.c(0, 79, 2, 'g-gold l3'); return out(P, [-58, 58, -4, 84]); };
  // 景山：山顶万春亭（三重檐方亭）
  I.jingshan = () => { const P = Pen(); shadow(P, 108); hill(P, 0, 0, 108, 38, { seed: 7 }); body(P, 0, 36, 16, 7, { cols: 2 }); roof(P, 0, 42, 32, 6, { type: 'cj', finial: false }); body(P, 0, 46, 12, 4, { cols: 2, lattice: false }); roof(P, 0, 49, 26, 6, { type: 'cj', finial: false }); roof(P, 0, 54, 20, 12, { type: 'cj' }); return out(P, [-56, 56, -2, 70]); };
  // 钟鼓楼：前鼓楼（红城台三券、上重檐歇山），后钟楼（灰石楼）
  I.zhonggu = () => { const P = Pen(); shadow(P, 112); P.r(14, 0, 30, 30, 'c-gw l2'); arch(P, 29, 0, 9, 18); roof(P, 29, 30, 36, 8, { type: 'xs', tone: 'gray', trim: 'green', rw: 8 }); P.r(22, 37, 14, 8, 'c-gw l2'); roof(P, 29, 44, 26, 12, { type: 'xs', tone: 'gray', trim: 'green', rw: 6 }); P.r(-46, 0, 56, 26, 'c-wallr l2'); for (const x of [-32, -18, -4]) arch(P, x, 0, 8, 16); body(P, -18, 26, 44, 12, { cols: 5 }); roof(P, -18, 37, 62, 8, { type: 'wd', tone: 'gray', trim: 'green', rw: 16 }); body(P, -18, 44, 34, 7, { cols: 4 }); roof(P, -18, 50, 48, 13, { type: 'xs', tone: 'gray', trim: 'green', rw: 11 }); return out(P, [-56, 50, -2, 66]); };
  // 永定门：灰砖城台正中券门，台上重檐城楼
  I.yongding = () => { const P = Pen(); shadow(P, 90); P.p(poly([[-42, 0], [42, 0], [38, 28], [-38, 28]]), 'c-gw l2'); P.p(poly([[-42, 0], [42, 0], [38, 28], [-38, 28]]), 'p-brick'); arch(P, 0, 0, 16, 22); wall(P, -38, 38, 28, 3, 'c-gw'); body(P, 0, 31, 50, 12, { cols: 5 }); roof(P, 0, 42, 70, 9, { type: 'wd', tone: 'gray', trim: 'green', rw: 20 }); body(P, 0, 50, 38, 7, { cols: 4 }); roof(P, 0, 56, 54, 14, { type: 'xs', tone: 'gray', trim: 'green', rw: 13 }); return out(P, [-46, 46, -2, 74]); };
  // 国子监：辟雍，方殿重檐攒尖，四面环水、白石栏
  I.guozijian = () => { const P = Pen(); P.e(0, 6, 54, 10, 'c-water l2'); rail(P, -50, 50, 9, 2.4); P.p('M-6 -4h12v8h-12Z', 'c-marble l3'); base(P, 0, 8, 44, 5); body(P, 0, 13, 34, 14, { cols: 4, door: true }); roof(P, 0, 26, 52, 9, { type: 'cj', finial: false }); body(P, 0, 34, 24, 5, { cols: 3, lattice: false }); roof(P, 0, 38, 40, 18, { type: 'cj' }); return out(P, [-56, 56, -6, 62]); };
  // 首都博物馆：方正石墙大楼，一个青铜筒斜伸出来，上面大挑檐
  I.shoubo = () => { const P = Pen(); shadow(P, 112); P.r(-50, 0, 100, 34, 'c-gb l2'); let g = ''; for (let x = -44; x < 50; x += 8) g += `M${P2(x, 2)}V32`; P.p(g, 'l-grid'); P.p('M-58 34H58L54 40H-54Z', 'c-gb-d l2'); P.p('M-30 0L-36 0C-40 14 -38 30 -30 42L-12 46C-16 32 -16 14 -12 0Z', 'c-bronze l2'); P.p('M-33 8C-34 18 -33 28 -28 38M-24 4C-26 16 -25 30 -20 42', 'l3'); P.p(poly([[18, 0], [44, 0], [44, 28], [18, 28]]), 'c-glass l2'); return out(P, [-60, 60, -2, 48]); };
  // 国家博物馆：方柱柱廊，檐口一道黄琉璃
  I.guobo = () => { const P = Pen(); shadow(P, 116); P.r(-56, 0, 112, 6, 'c-stone l2'); P.r(-56, 6, 26, 30, 'c-gb l2'); P.r(30, 6, 26, 30, 'c-gb l2'); P.r(-30, 6, 60, 34, 'c-gb-d l2'); let c = ''; for (let k = 0; k < 8; k++) c += `M${P2(-27 + k * 7.6, 6)}h4v32h-4Z`; P.p(c, 'c-gb-l l3'); P.r(-58, 36, 116, 3, 'c-gb l3'); P.r(-60, 39, 120, 3.5, 'c-yellow l3'); P.r(-34, 40, 68, 4, 'c-yellow l3'); let w = ''; for (const x0 of [-52, 34]) for (let k = 0; k < 3; k++) w += `M${P2(x0 + k * 7, 12)}h3v18h-3Z`; P.p(w, 'c-gbw'); return out(P, [-62, 62, -2, 48]); };
  // 北大红楼：红砖四层楼，一排排白框窗，正中门
  I.honglou = () => { const P = Pen(); shadow(P, 110); P.r(-54, 0, 108, 46, 'c-brick-r l2'); P.p('M-56 46H56L52 52H-52Z', 'c-gray-d l2'); let w = ''; for (let row = 0; row < 4; row++) for (let k = 0; k < 11; k++) { if (row === 0 && k === 5) continue; w += `M${P2(-50 + k * 9.4, 4 + row * 10.6)}h4.6v6.4h-4.6Z`; } P.p(w, 'c-win l3'); P.r(-5, 0, 10, 9, 'c-ink'); P.p('M-54 11H54M-54 22H54M-54 33H54', 'l-brick'); return out(P, [-58, 58, -2, 54]); };
  // 李大钊故居：四合院门楼，灰墙灰瓦，红门，院里一棵树
  I.lidazhao = () => { const P = Pen(); shadow(P, 104); tree(P, 26, 20, 44, 'round'); P.r(-50, 0, 100, 22, 'c-gw l2'); P.r(-50, 22, 100, 3, 'c-gray-d l3'); P.r(-14, 0, 28, 28, 'c-gw l2'); P.r(-8, 0, 16, 20, 'c-red l2'); P.p('M0 0V20', 'l3'); roof(P, 0, 28, 40, 12, { type: 'ys', tone: 'gray', rw: 9 }); return out(P, [-54, 54, -2, 66]); };
  // 中国共产党历史展览馆：方柱高廊，正中大门，宽台阶
  I.dangshi = () => { const P = Pen(); shadow(P, 116); stairs(P, 0, 0, 70, 6); P.r(-56, 6, 112, 36, 'c-gb-d l2'); let c = ''; for (let k = 0; k < 12; k++) c += `M${P2(-52 + k * 9.2, 6)}h4.6v36h-4.6Z`; P.p(c, 'c-gb-l l3'); P.r(-60, 42, 120, 6, 'c-gb l2'); P.r(-56, 48, 112, 3, 'c-gb-l l3'); return out(P, [-62, 62, -2, 54]); };
  // 卢沟桥与宛平城：十一孔石桥，望柱石狮；桥头宛平城门
  I.lugou = () => { const P = Pen(); water(P, -10, 2, 118, 9); let d = 'M-62 0V12H30V0Z'; for (let k = 0; k < 9; k++) { const x = -56 + k * 9.6; d += `M${P2(x, 0)}V4A3.8 3.8 0 0 0 ${P2(x + 7.6, 4)}V0Z`; } P.p(d, 'c-marble l2', ' fill-rule="evenodd"'); let p = ''; for (let x = -60; x <= 28; x += 5) p += `M${P2(x, 12)}v4`; P.p(p + 'M-62 15H30', 'l3'); let lion = ''; for (let x = -60; x <= 28; x += 10) lion += `M${P2(x - 1.2, 16)}a1.2 1.4 0 0 0 2.4 0Z`; P.p(lion, 'c-marble l3'); P.p(poly([[30, 0], [62, 0], [60, 26], [32, 26]]), 'c-gw l2'); arch(P, 46, 0, 9, 16); roof(P, 46, 26, 34, 12, { type: 'xs', tone: 'gray', rw: 7 }); return out(P, [-64, 64, -6, 42]); };
  // 平西抗日战争纪念馆：青山前一座石碑、一排纪念馆，松柏夹道
  I.pingxi = () => { const P = Pen(); peaks(P, 0, 0, 120, 40, { seed: 4 }); P.r(-40, 0, 80, 14, 'c-gb l2'); let w = ''; for (let k = 0; k < 7; k++) w += `M${P2(-34 + k * 10, 3)}h5v8h-5Z`; P.p(w, 'c-gbw'); P.p('M-5 0H5V40L0 46L-5 40Z', 'c-stone l2'); P.p('M-2 14V36', 'l3'); for (const x of [-48, 48]) tree(P, x, 0, 26, 'bai'); return out(P, [-62, 62, -2, 50]); };
  // 焦庄户地道战遗址：村口土坡上的地道口，几间民房，一棵树
  I.jiaozhuanghu = () => { const P = Pen(); shadow(P, 110); P.p('M-56 0C-40 18 -10 22 10 18C24 16 40 8 56 0Z', 'c-zhe l2'); arch(P, -12, 2, 11, 12); P.r(14, 0, 34, 14, 'c-gw l2'); roof(P, 31, 14, 40, 9, { type: 'ys', tone: 'gray', rw: 13 }); P.r(26, 0, 6, 9, 'c-wood l3'); tree(P, -40, 6, 30, 'round'); return out(P, [-58, 58, -2, 40]); };
  // 香山革命纪念地：香山上红叶，掩着一座小院（双清别墅）和八角亭
  I.xiangshan = () => { const P = Pen(); peaks(P, 0, 0, 120, 50, { seed: 6, peaks: [[-0.2, 1, 0.42], [0.3, 0.7, 0.36]] }); for (const [x, y, h] of [[-44, 2, 22], [-30, 14, 20], [40, 4, 20], [26, 16, 18], [-8, 26, 16]]) tree(P, x, y, h, 'maple'); P.r(-14, 2, 30, 11, 'c-gw l2'); roof(P, 1, 13, 38, 8, { type: 'ys', tone: 'gray', rw: 12 }); P.r(-12, 2, 5, 7, 'c-red l3'); body(P, 30, 30, 8, 6, { cols: 2, lattice: false }); roof(P, 30, 35, 16, 9, { type: 'cj' }); return out(P, [-62, 62, -2, 56]); };
  // 八达岭长城：两座山头，城墙顺脊而上，两座敌楼
  I.badaling = () => { const P = Pen(), pk = [[-0.3, 0.9, 0.34], [0.22, 0.66, 0.36]]; peaks(P, 0, 0, 124, 56, { seed: 8, peaks: pk }); const rd = ridgeOf(0, 0, 124, 56, pk, 0.9, -62, 62, 3); greatWall(P, rd, { h: 5.2, towers: [9, 30] }); return out(P, [-64, 64, -2, 66]); };
  // 慕田峪长城：苍翠山岭，敌楼密集
  I.mutianyu = () => { const P = Pen(), pk = [[-0.25, 1, 0.36], [0.28, 0.82, 0.34]]; peaks(P, 0, 0, 124, 52, { seed: 12, fill: 'g-shan-mid', peaks: pk }); const rd = ridgeOf(0, 0, 124, 52, pk, 0.9, -60, 62, 3); greatWall(P, rd, { h: 5, towers: [8, 13, 18, 29, 35], huts: [13] }); return out(P, [-64, 64, -2, 64]); };
  // 司马台长城与古北水镇：陡峭的单脊，城墙骑在刀背一样的山脊上；山下水镇灰瓦
  I.simatai = () => { const P = Pen(), pk = [[-0.1, 1, 0.3], [0.3, 0.7, 0.26]]; P.e(0, 2, 60, 6, 'c-water l3'); peaks(P, 4, 6, 120, 56, { seed: 14, sharp: 1.6, peaks: pk }); const rd = ridgeOf(4, 6, 120, 56, pk, 1.6, -36, 62, 2.5, 2); greatWall(P, rd, { h: 4.4, towers: [6, 13, 22, 30] }); for (const [x, w] of [[-46, 20], [-24, 18], [26, 18], [48, 18]]) { P.r(x - w / 2, 4, w, 8, 'c-marble l3'); P.r(x - w * 0.12, 4, w * 0.24, 5, 'c-wood'); roof(P, x, 12, w + 4, 6, { type: 'ys', tone: 'gray', rw: w * 0.3 }); } return out(P, [-64, 64, -6, 70]); };
  // 明十三陵：明楼（方城上重檐歇山）、宝城一圈，松柏
  I.shisanling = () => { const P = Pen(); shadow(P, 112); P.e(0, 10, 54, 12, 'c-wallr l2'); P.e(0, 14, 48, 9, 'c-tree l3'); P.r(-20, 0, 40, 24, 'c-wallr l2'); arch(P, 0, 0, 10, 16); P.r(-14, 24, 28, 14, 'c-wallr l2'); P.r(-6, 28, 12, 7, 'c-stone l3'); roof(P, 0, 38, 44, 9, { type: 'wd', rw: 12 }); P.r(-11, 46, 22, 5, 'c-wallr l3'); roof(P, 0, 50, 36, 13, { type: 'xs', rw: 9 }); for (const x of [-46, 44]) tree(P, x, 0, 30, 'pine'); return out(P, [-60, 60, -2, 66]); };
  // 大运河与燃灯塔（城市副中心）：运河、帆船、柳，岸上十三层密檐砖塔
  I.yunhe = () => { const P = Pen(); P.p('M-60 4C-20 10 20 -2 60 4L60 -4C20 -10 -20 2 -60 -4Z', 'c-water l2'); P.p('M-30 6l14 0l-2 3l-10 0Z', 'c-wood l3'); P.p('M-23 9V22l7 -10Z', 'c-sail l3'); P.r(18, 4, 18, 6, 'c-stone l2'); let z = 10, w = 13; P.r(27 - 6, z, 12, 12, 'c-gw l2'); z += 12; for (let k = 0; k < 9; k++) { P.p(`M${P2(27 - w / 2 - 2, z)}h${N(w + 4)}l-1.5 1.6h${N(-w - 1)}Z`, 'c-gray-d l3'); P.r(27 - w / 2, z + 1.6, w, 2.4, 'c-gw l3'); z += 4; w -= 0.7; } P.p(`M${P2(27 - 4, z)}L27 ${N(z + 9)}L${P2(27 + 4, z)}Z`, 'c-gray-d l3'); P.c(27, z + 10, 1.4, 'g-gold'); for (const x of [-48, 52]) tree(P, x, 2, 26, 'willow'); return out(P, [-62, 62, -10, z + 14]); };
  // 奥林匹克公园：鸟巢（钢架交错的椭圆碗）与水立方（泡泡蓝方盒）
  I.aoyun = () => { const P = Pen(); shadow(P, 118); P.p('M-56 4C-56 26 6 26 6 4C6 -2 -56 -2 -56 4Z', 'c-steel l2'); let l = ''; for (let k = 0; k < 9; k++) { const x = -54 + k * 7; l += `M${P2(x, 2)}L${P2(x + 9, 20)}M${P2(x + 9, 2)}L${P2(x, 20)}`; } P.p(l, 'l3'); P.p('M-50 14C-40 20 -10 20 0 14', 'l2'); P.r(14, 0, 42, 22, 'c-cube l2'); let b = ''; for (let k = 0; k < 14; k++) { const x = 17 + (k % 7) * 5.6, y = 4 + Math.floor(k / 7) * 9 + (k % 2) * 3; b += `M${P2(x - 1.8, y)}a1.8 1.8 0 1 0 3.6 0a1.8 1.8 0 1 0 -3.6 0Z`; } P.p(b, 'l-wave2'); return out(P, [-60, 60, -2, 30]); };
  // 中国科学技术馆：方正大馆，旁一颗球幕
  I.kejiguan = () => { const P = Pen(); shadow(P, 110); P.r(-50, 0, 70, 38, 'c-gb-l l2'); let g = ''; for (let k = 1; k < 6; k++) g += `M${P2(-50 + k * 11.6, 2)}V36M${P2(-48, k * 6.2)}H18`; P.p(g, 'l-grid'); P.p('M-54 38H24L20 42H-50Z', 'c-gb-d l2'); P.c(36, 16, 16, 'c-steel l2'); P.p('M20 16Q36 24 52 16M23 8Q36 13 49 8M36 0V32', 'l3'); return out(P, [-56, 56, -2, 44]); };
  // 十渡：拒马河两岸喀斯特尖峰，河上竹筏
  I.shidu = () => { const P = Pen(); peaks(P, 0, 4, 124, 60, { seed: 16, sharp: 1.4, fill: 'g-shan-mid', peaks: [[-0.32, 1, 0.22], [-0.08, 0.7, 0.2], [0.2, 0.92, 0.22], [0.4, 0.6, 0.2]] }); P.p('M-62 6C-20 10 20 2 62 6L62 -2C20 -6 -20 2 -62 -2Z', 'c-water l2'); P.p('M-14 2h18l-1.6 2.4h-15Z', 'c-mat l3'); P.p('M-6 4.4V12', 'l-mast'); return out(P, [-64, 64, -4, 68]); };
  // 京西古道：山间一条石板古道，蜿蜒到一座关口
  I.jingxi = () => { const P = Pen(); peaks(P, 0, 0, 124, 50, { seed: 18, fill: 'g-shan-mid' }); P.p('M-44 0C-30 10 -10 6 -2 14C6 22 20 20 24 30L28 30C24 18 10 18 2 10C-6 2 -24 4 -38 0Z', 'c-stone l2'); let s = ''; for (const [x, y] of [[-34, 3], [-20, 6], [-6, 9], [6, 16], [16, 22]]) s += `M${P2(x, y)}l3 1`; P.p(s, 'l3'); P.r(18, 28, 18, 10, 'c-gw l2'); arch(P, 27, 28, 6, 8); roof(P, 27, 38, 22, 6, { type: 'ys', tone: 'gray', rw: 6 }); P.p('M-40 4c0 -3 3 -4 5 -2l3 0c1 -3 4 -3 5 0c2 1 2 3 1 4l-1 3h-1l0 -3h-6l-1 3h-1l1 -4c-2 0 -4 -1 -5 -1Z', 'c-zhe-d l3'); return out(P, [-64, 64, -2, 56]); };
  // 灵山：北京最高峰，山顶高山草甸，云绕山腰
  I.lingshan = () => { const P = Pen(); peaks(P, 0, 0, 124, 66, { seed: 20, fill: 'g-shan-mid', peaks: [[0, 1, 0.42], [-0.32, 0.6, 0.3], [0.34, 0.54, 0.3]] }); P.p('M-14 56C-8 62 8 62 14 56C8 60 -8 60 -14 56Z', 'c-bank-t l3'); P.p('M-40 24C-26 30 -10 22 4 26C14 28 24 24 34 28C24 34 -20 34 -40 24Z', 'c-cloud l-cloud'); return out(P, [-64, 64, -2, 72]); };
  // 野鸭湖：湿地水面、芦苇、几只飞起的野鸭
  I.yeyahu = () => { const P = Pen(); P.e(0, 4, 58, 9, 'c-water l2'); let reed = ''; for (let k = 0; k < 16; k++) { const x = -54 + k * 7 + (k % 3), h = 10 + (k % 4) * 4; reed += `M${P2(x, 2)}q1 ${N(h * 0.6)} ${N((k % 2 ? 2 : -2))} ${N(h)}`; } P.p(reed, 'l-reed'); let duck = ''; for (const [x, y, s] of [[-10, 32, 1], [6, 38, 0.8], [20, 30, 0.9]]) duck += `M${P2(x - 5 * s, y)}q${N(2.5 * s)} ${N(3 * s)} ${N(5 * s)} 0q${N(2.5 * s)} ${N(3 * s)} ${N(5 * s)} 0`; P.p(duck, 'l2'); return out(P, [-62, 62, -6, 44]); };
  // 雁栖湖国际会都：湖、远山，岸上日出东方（半圆形的楼）与会议中心（大屋顶）
  I.yanqi = () => { const P = Pen(); peaks(P, 0, 10, 124, 34, { seed: 22, fill: 'g-shan-far' }); P.e(0, 6, 60, 8, 'c-water l2'); P.p('M-50 8V18A16 22 0 0 0 -18 18V8Z', 'c-glass l2'); let g = ''; for (let k = 1; k < 5; k++) g += `M${P2(-50, 8 + k * 4.6)}H-18`; P.p(g + 'M-34 8V40', 'l-grid'); P.p('M-48 30Q-34 46 -20 30', 'l3'); body(P, 26, 8, 44, 12, { cols: 5 }); roof(P, 26, 19, 60, 12, { type: 'xs', rw: 14, tone: 'gray' }); return out(P, [-64, 64, -2, 48]); };
  // 平谷桃花：山坡上一片粉红桃林
  I.pinggu = () => { const P = Pen(); hill(P, 0, 0, 120, 30, { seed: 24, trees: 0, fill: 'g-shan-mid' }); for (const [x, y, h] of [[-44, 2, 22], [-28, 8, 24], [-10, 14, 24], [10, 12, 24], [28, 8, 24], [46, 2, 20], [-20, 0, 20], [20, 0, 20]]) tree(P, x, y, h, 'peach'); return out(P, [-62, 62, -2, 42]); };
  // 北宫国家森林公园：林木满坡，坡上一座亭
  I.beigong = () => { const P = Pen(); hill(P, 0, 0, 120, 40, { seed: 26, trees: 10 }); body(P, 18, 30, 10, 5, { cols: 2, lattice: false }); roof(P, 18, 34, 20, 10, { type: 'cj' }); return out(P, [-62, 62, -2, 50]); };
  // 延庆冬奥：雪山上几道雪道，山脚一座场馆
  I.dongao = () => { const P = Pen(); peaks(P, 0, 0, 124, 62, { seed: 28, fill: 'g-shan-mid', peaks: [[-0.12, 1, 0.42], [0.34, 0.66, 0.3]] }); P.p('M-30 50C-22 34 -26 20 -18 2M-14 58C-4 40 -12 22 -2 4M4 52C10 36 6 18 14 2', 'l-ski'); P.r(24, 0, 30, 10, 'c-glass l2'); P.p('M22 10H56L52 15H26Z', 'c-steel l3'); return out(P, [-64, 64, -2, 68]); };
  // 国家会议中心：长条大楼，波浪形屋檐，通透玻璃
  I.guohui = () => { const P = Pen(); shadow(P, 120); P.r(-58, 0, 116, 26, 'c-glass l2'); let g = ''; for (let x = -54; x < 58; x += 6) g += `M${P2(x, 2)}V24`; P.p(g + 'M-58 12H58', 'l-grid'); P.p('M-62 26C-48 32 -34 32 -20 28C-6 24 6 24 20 28C34 32 48 32 62 26L62 30C48 36 34 36 20 32C6 28 -6 28 -20 32C-34 36 -48 36 -62 30Z', 'c-steel l2'); return out(P, [-64, 64, -2, 38]); };
  // 首钢园：两座冷却塔、一座高炉，滑雪大跳台像一条飘带
  I.shougang = () => { const P = Pen(); shadow(P, 118); for (const [x, s] of [[-40, 1], [-18, 0.9]]) P.p(`M${P2(x - 14 * s, 0)}C${P2(x - 8 * s, 14 * s)} ${P2(x - 8 * s, 26 * s)} ${P2(x - 11 * s, 40 * s)}H${N(x + 11 * s)}C${P2(x + 8 * s, 26 * s)} ${P2(x + 8 * s, 14 * s)} ${P2(x + 14 * s, 0)}Z`, 'c-steel l2'); P.r(4, 0, 10, 30, 'c-zhe-d l2'); P.r(2, 30, 14, 4, 'c-gray-d l3'); P.p('M20 0C26 10 34 30 44 46L52 46C46 36 40 22 40 10C40 4 44 0 52 0Z', 'c-snow l2'); P.p('M44 46L46 50H54L52 46', 'c-glass l3'); return out(P, [-60, 60, -2, 52]); };
  // 中关村：一簇玻璃高楼
  I.zgc = () => { const P = Pen(); shadow(P, 104); tower(P, -26, 0, 20, 38); tower(P, 0, 0, 24, 56); tower(P, 26, 0, 18, 44, { cls: 'c-glass2' }); return out(P, [-52, 52, -2, 58]); };
  // 丽泽：一座扭转上升的高楼，旁边低一些的楼
  I.lize = () => { const P = Pen(); shadow(P, 104); tower(P, -28, 0, 18, 32, { cls: 'c-glass2' }); P.p('M-8 0C-12 20 -4 40 -10 66H10C4 40 12 20 8 0Z', 'c-glass l2'); P.p('M-6 10C-4 24 4 32 2 50M2 4C4 20 -2 40 0 60', 'l-grid'); tower(P, 28, 0, 20, 40); return out(P, [-52, 52, -2, 68]); };
  // 亦庄：科技园区，弧形研发楼与厂房
  I.yizhuang = () => { const P = Pen(); shadow(P, 116); P.r(-56, 0, 50, 18, 'c-gb-l l2'); P.p('M-56 18L-44 26L-32 18L-20 26L-8 18Z', 'c-steel l2'); P.p('M0 0V30C14 38 40 38 54 30V0Z', 'c-glass l2'); let g = ''; for (let k = 1; k < 6; k++) g += `M${P2(0, k * 5)}C14 ${N(k * 5 + 6)} 40 ${N(k * 5 + 6)} 54 ${N(k * 5)}`; P.p(g, 'l-grid'); return out(P, [-60, 60, -2, 40]); };
  // CBD：中国尊（上宽下窄的瓶形高楼）领着一片楼群
  I.cbd = () => { const P = Pen(); shadow(P, 108); tower(P, -30, 0, 18, 40, { cls: 'c-glass2' }); tower(P, 30, 0, 18, 46); P.p('M-9 0C-6 30 -7 60 -11 86H11C7 60 6 30 9 0Z', 'c-glass l2'); P.p('M-11 86C-6 90 6 90 11 86', 'l2'); let g = ''; for (let k = 1; k < 17; k++) { const y = k * 5, w = 9 - Math.sin(y / 86 * Math.PI) * 2 + y / 86 * 2; g += `M${P2(-w, y)}H${N(w)}`; } P.p(g, 'l-grid'); return out(P, [-52, 52, -2, 92]); };

  /* ── 晟帆出发地：一面小帆旗 ───────────────────── */
  I.home = () => { const P = Pen(); P.p('M0 0V40', 'l-mast'); P.p('M1 40C10 38 18 36 24 32C18 28 10 26 1 24Z', 'c-red l2'); P.p('M1 24V40', 'l2'); return out(P, [-6, 28, -1, 42]); };

  window.JH.ICONS = Object.assign({}, I, { _kit: { roof, roundRoof, body, base, rail, stairs, wall, arch, shadow, peaks, hill, water, tree, tower } });
})();
