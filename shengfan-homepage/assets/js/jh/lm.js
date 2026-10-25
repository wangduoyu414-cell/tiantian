/* 界画库 · 地标（首屏《京华胜景图》用）
   都在局部坐标里画（u 右 v 上，单位约 0.1 米），比例照实物照片定，返回 { g, box }。
   角楼：宫城转角上的三重檐、十字脊。一层四面抱厦外绕一圈腰檐，二层抱厦是朝外的歇山山花，
         顶上十字脊，正面又是一个山花，交脊处鎏金宝顶。墙是栗红色宫城城墙，外沿一道雉堞。 */
(function () {
  'use strict';
  const { N, P2, Pen, sub, poly, samp, Round } = window.JH, A = window.JH.ARCH;

  /* ── 一段殿身：x0..x1 两端立柱，柱间隔扇，上面额枋、斗拱 ── */
  function bodySeg(P, xs, z0, z1, o = {}) {
    const cw = o.cw ?? 4.2, bH = o.bH ?? (z1 - z0) * 0.14, dH = o.dH ?? (z1 - z0) * 0.17, zc = z1 - bH - dH;
    for (let i = 0; i < xs.length - 1; i++) A.geshan(P, xs[i] + cw / 2, xs[i + 1] - cw / 2, z0 + 0.4, zc - 0.4, o.leaves ?? Math.max(2, Math.round((xs[i + 1] - xs[i]) / 9)));
    A.columns(P, xs, z0, zc, cw);
    A.beams(P, xs, zc, bH, o.caihua || 'hexi');
    A.dougong(P, xs, zc + bH, dH, o.per ?? 2, cw * 0.6);
  }

  /* ── 宫城城墙：栗红墙身略有收分，下有石基；墙顶外沿一道雉堞（垛墙、垛口、射眼）──
     wallBody 先画（在塔后面），parapet 最后画（挡在塔基前面）。x0..x1 墙长；z0 墙脚、z1 墙顶；cornerL/R 该端是转角 */
  function wallBody(P, x0, x1, z0, z1, o = {}) {
    const H = z1 - z0, bat = H * 0.05, bl = o.cornerL ? bat : 0, br = o.cornerR ? bat : 0;
    P.p(poly([[x0, z0], [x1, z0], [x1 - br, z1], [x0 + bl, z1]]), 'c-wallr l1');
    P.p(poly([[x0, z0], [x1, z0], [x1 - br * 0.3, z0 + H * 0.3], [x0 + bl * 0.3, z0 + H * 0.3]]), 'c-shade');
    if (o.cornerR) P.p(poly([[x1 - H * 0.22, z0], [x1, z0], [x1 - br, z1], [x1 - br - H * 0.12, z1]]), 'c-shade');
    if (o.cornerL) P.p(poly([[x0, z0], [x0 + H * 0.06, z0], [x0 + bl + H * 0.03, z1], [x0 + bl, z1]]), 'c-light');
    P.p(poly([[x0 + bl, z1 - H * 0.04], [x1 - br, z1 - H * 0.04], [x1 - br, z1], [x0 + bl, z1]]), 'c-shade2');
    const sb = H * 0.07;
    P.p(poly([[x0 - 1, z0], [x1 + 1, z0], [x1 + 1 - br * 0.07, z0 + sb], [x0 - 1 + bl * 0.07, z0 + sb]]), 'c-stone l2');
  }
  function parapet(P, x0, x1, z1, H, o = {}) {
    const bat = H * 0.05, a = x0 + (o.cornerL ? bat : 0), b = x1 - (o.cornerR ? bat : 0);
    const mh = o.parapet ?? H * 0.14, mw = o.merlon ?? mh * 1.05, gap = mw * 0.45, low = mh * 0.5;
    let d = `M${P2(a, z1)}L${P2(a, z1 + mh)}`, holes = '';
    for (let x = a + mw; x + gap + mw * 0.6 < b; x += mw + gap) { d += `L${P2(x, z1 + mh)}L${P2(x, z1 + low)}L${P2(x + gap, z1 + low)}L${P2(x + gap, z1 + mh)}`; holes += `M${P2(x - mw * 0.55, z1 + low * 0.45)}h${N(mw * 0.14)}v${N(low * 0.36)}h${N(-mw * 0.14)}Z`; }
    d += `L${P2(b, z1 + mh)}L${P2(b, z1)}Z`;
    P.p(d, `${o.cls || 'c-wallr'} l2`);
    P.p(holes, 'c-ink');
    P.p(`M${P2(a, z1 + mh * 0.08)}H${N(b)}`, 'l3');
  }

  /* ── 故宫角楼 ───────────────────────────────
     o: wallH 城墙高、wallL 墙往左伸多长（塔在墙的右端转角上）；局部原点在墙顶、塔中线 */
  function jiaolou(o = {}) {
    const P = Pen(), tone = 'yellow', wallH = o.wallH ?? 100, wallL = o.wallL ?? 150, C = 92;
    const g = (x, f) => P.g(sub(f), ` transform="translate(${N(x)} 0)"`);
    if (wallH) wallBody(P, -wallL, C, -wallH, 0, { cornerR: true });
    // 台基（矮，大半藏在雉堞后面）
    A.xumizuo(P, 0, 8, 86);
    // 三层（顶）：正身上段红墙，十字脊（长边歇山 + 正面山花），交脊处宝顶
    bodySeg(P, [-48, -16, 16, 48], 82, 122, { per: 2 });
    A.roof(P, { type: 'xs', ey: 120, ew: 63, rh: 52, rw: 24, tone, tile: 3.4, beasts: 0, lift: 8, ridgeH: 5.4 });
    A.roof(P, { type: 'xsg', ey: 120, ew: 63, rh: 52, gw: 30, gy: 0.4, tone, tile: 3.4, beasts: 3, lift: 8, ridgeH: 5.4, apex: 'baoding' });
    // 二层：殿身，抱厦山花（两侧抱厦看长边，正中抱厦山花朝前）
    bodySeg(P, [-66, -48, -30, 30, 48, 66], 60, 82, { per: 1 });
    g(57, Q => A.roof(Q, { type: 'xs', ey: 80, ew: 24, rh: 28, rw: 7, tone, tile: 3.2, beasts: 2, lift: 6, ridgeH: 4.4, ends: { l: 'cut' } }));
    g(-57, Q => A.roof(Q, { type: 'xs', ey: 80, ew: 24, rh: 28, rw: 7, tone, tile: 3.2, beasts: 2, lift: 6, ridgeH: 4.4, ends: { r: 'cut' } }));
    A.roof(P, { type: 'xsg', ey: 80, ew: 45, rh: 40, gw: 24, gy: 0.4, tone, tile: 3.2, beasts: 2, lift: 7, ridgeH: 4.6, apex: 'none' });
    // 一层：殿身一圈，外绕腰檐（两侧抱厦、正中抱厦各一段，窝角处截断）
    bodySeg(P, [-66, -48, -30, 30, 48, 66], 15, 46, { per: 1 });
    g(57, Q => A.roof(Q, { type: 'yao', ey: 44, ew: 24, rh: 16, wallW: 9, tone, tile: 3.2, lift: 5, ridgeH: 4.4, beasts: 2, ends: { l: 'cut' } }));
    g(-57, Q => A.roof(Q, { type: 'yao', ey: 44, ew: 24, rh: 16, wallW: 9, tone, tile: 3.2, lift: 5, ridgeH: 4.4, beasts: 2, ends: { r: 'cut' } }));
    A.roof(P, { type: 'yao', ey: 44, ew: 45, rh: 16, wallW: 30, tone, tile: 3.2, lift: 5, ridgeH: 4.4, beasts: 2 });
    // 台上汉白玉栏杆，最前是城墙雉堞
    A.balustrade(P, -84, 84, 8, 7, 7.5);
    if (wallH) parapet(P, -wallL, C, 0, wallH, { cornerR: true });
    return { g: P.s(), box: [wallH ? -wallL - 4 : -94, C + 8, wallH ? -wallH - 2 : -2, 198] };
  }


  /* ── 颐和园佛香阁：八角三层四重檐，建在约 21 米高的石台上 ──
     石台正面石阶直上，台顶一圈汉白玉栏杆；阁身黄琉璃绿剪边，二、三层外有平座栏杆；顶上八角攒尖、鎏金宝顶。
     局部原点在石台顶面中线。 */
  function foxiang(o = {}) {
    const T = window.JH.TOWER({ n: 8, eye: 0, top: 400, upK: 0.1, dnK: 0.05, dnZ: -80 }), P = T.P, tone = 'yellow', trim = 'green';
    // 石台
    const pt = 190, pb = 212, H = 180;
    P.p(poly([[-pb, -H], [pb, -H], [pt, 0], [-pt, 0]]), 'c-stone l1');
    // 虎皮石：大小不一的石块，缝走折线
    const R = window.JH.rng(29); let joint = '';
    for (let z = -H + 6; z < -6; z += 10 + R() * 8) { const w = pb + (pt - pb) * (z + H) / H; let x = -w + R() * 12; joint += `M${P2(-w, z)}`; while (x < w) { const nx = Math.min(w, x + 14 + R() * 22); joint += `L${P2(nx, z + (R() - 0.5) * 5)}`; x = nx; } }
    for (let i = 0; i < 70; i++) { const z = -H + R() * H, w = pb + (pt - pb) * (z + H) / H, x = (R() - 0.5) * 2 * w * 0.96; joint += `M${P2(x, z)}l${N((R() - 0.5) * 4)} ${N(7 + R() * 6)}`; }
    P.p(joint, 'l-cun');
    P.p(poly([[pb * 0.3, -H], [pb, -H], [pt, 0], [pt * 0.35, 0]]), 'c-shade');
    P.p(poly([[-pb, -H], [-pb + 22, -H], [-pt + 16, 0], [-pt, 0]]), 'c-light');
    // 石阶：正中直上，两侧垂带
    const sw = 24;
    P.p(poly([[-sw - 4, -H], [sw + 4, -H], [sw + 2, 0], [-sw - 2, 0]]), 'c-marble l2');
    let st = ''; for (let z = -H + 5; z < 0; z += 5) st += `M${P2(-sw, z)}H${N(sw)}`;
    P.p(st, 'l3');
    P.p(`M${P2(-sw, -H)}L${P2(-sw + 1, 0)}M${P2(sw, -H)}L${P2(sw - 1, 0)}`, 'l2');
    // 阁：塔基、一层副阶 + 上段、二层、三层，层层收进
    T.base(118, 0, 8);
    T.body(100, 8, 52, { cols: 3, fill: 'door' });
    T.body(82, 78, 100, { cols: 2, fill: 'win' });
    T.roof(134, 50, 86, 80, { tone, trim, tb: 3.6, lift: 4.5, tiles: 12, beasts: 2 });
    T.body(70, 128, 158, { cols: 2, fill: 'door' });
    T.roof(118, 98, 74, 128, { tone, trim, tb: 3.4, lift: 4.5, tiles: 11, beasts: 2 });
    T.balcony(80, 128, 8);
    T.body(58, 186, 212, { cols: 2, fill: 'door' });
    T.roof(104, 156, 64, 186, { tone, trim, tb: 3.2, lift: 4.5, tiles: 10, beasts: 2 });
    T.balcony(68, 186, 7);
    T.roof(94, 210, 0, 316, { tone, trim, tb: 3.1, lift: 5, tiles: 10, beasts: 2 });
    P.g(sub(Q => A.baoding(Q, 0, 7.4)), ` transform="translate(0 310)"`);
    // 台脚下：德辉殿、排云殿一路往下（只露屋顶和殿身上段）
    if (o.front !== false) {
      const B = window.JH.BUILD;
      for (const [y, k, dbl] of [[-H - 6, 0.7, false], [-H - 70, 0.62, true]]) { const hl = window.JH.withLod(0.45, () => B.hall(B.scaled({ bays: [70, 60, 54], colH: 66, platform: 14, rail: 8, double: dbl, type: 'xs', tile: 6, beasts: 3, per: 2 }, k))); P.g(hl.g, ` transform="translate(0 ${N(y)})"`); }
    }
    // 台顶石栏（让出石阶口）
    A.balustrade(P, -pt + 2, -sw - 4, 0, 9, 9); A.balustrade(P, sw + 4, pt - 2, 0, 9, 9);
    return { g: P.s(), box: [-pb - 6, pb + 6, o.front !== false ? -H - 72 : -H - 2, 360] };
  }

  /* ── 玉泉山玉峰塔：八角七层石塔，层层收分，每层一道灰瓦小檐，塔身灰白，正面一个券龛 ── */
  function yufeng(o = {}) {
    const T = window.JH.TOWER({ n: 8, eye: 0, top: 500, upK: 0.12, dnK: 0.05 }), P = T.P;
    T.base(44, 0, 12, { cls: 'c-marble' });
    let z = 12, r = 36;
    for (let i = 0; i < 7; i++) {
      const h = 46 - i * 2.4, r2 = r - 2.2;
      T.body(r, z, z + h * 0.84, { cols: 1, fill: 'stone', bH: 1.6, dH: 2, cw: 2, noCols: true });
      T.roof(r + 5, z + h * 0.84 - 0.5, r2 + 0.5, z + h, { tone: 'gray', tb: 1.6, lift: 1.6, tiles: 5, beasts: 0, chu: 0.04 });
      z += h; r = r2;
    }
    T.roof(r + 7, z - 1, 0, z + 34, { tone: 'gray', tb: 1.8, lift: 2.4, tiles: 5, beasts: 0, chu: 0.05 });
    P.g(sub(Q => A.baoding(Q, 0, 4.2)), ` transform="translate(0 ${N(z + 31)})"`);
    return { g: P.s(), box: [-56, 56, -4, z + 56] };
  }

  /* ── 北海白塔：藏式喇嘛塔。折角须弥座、三道金刚圈、塔肚（正面眼光门）、塔脖子、十三天相轮、鎏金华盖、宝顶 ── */
  function baita(o = {}) {
    const P = Pen(), R = Round(40, 380, 0.09, 0.05, 0);
    const arc = (r, z) => R.arc(r, z, 36), line = pts => 'M' + pts.map(q => P2(...q)).join('L');
    const band = (r0, z0, r1, z1, cls) => P.p(poly([...arc(r0, z0), ...arc(r1, z1).reverse()]), cls);
    A.xumizuo(P, 0, 58, 100);
    P.p(`M${P2(-62, 9)}V53M${P2(62, 9)}V53M${P2(-82, 9)}V53M${P2(82, 9)}V53`, 'l3');
    let z = 58;
    for (const [r, hh] of [[84, 9], [78, 9], [72, 9]]) { band(r, z, r, z + hh, 'c-marble l2'); band(r, z, r, z + hh, 'g-drum'); z += hh; }
    // 塔肚
    const bulb = `M${P2(-60, z)}C${P2(-80, z + 26)} ${P2(-82, z + 100)} ${P2(-41, z + 122)}L${P2(41, z + 122)}C${P2(82, z + 100)} ${P2(80, z + 26)} ${P2(60, z)}Z`;
    P.p(bulb, 'c-marble l1'); P.p(bulb, 'g-drum'); P.p(line(arc(60, z)), 'l2');
    // 眼光门：金框、红地、中间一枚金饰
    const zb = z + 30, zt = z + 92;
    P.p(`M${P2(-17, zb)}V${N(zt - 14)}C${P2(-17, zt - 4)} ${P2(-8, zt + 2)} 0 ${N(zt + 7)}C${P2(8, zt + 2)} ${P2(17, zt - 4)} ${P2(17, zt - 14)}V${N(zb)}Z`, 'g-gold l2');
    P.p(`M${P2(-12.5, zb + 3.5)}V${N(zt - 15)}C${P2(-12.5, zt - 7)} ${P2(-6, zt - 2)} 0 ${N(zt + 1)}C${P2(6, zt - 2)} ${P2(12.5, zt - 7)} ${P2(12.5, zt - 15)}V${N(zb + 3.5)}Z`, 'c-red l3');
    P.p(`M0 ${N(zb + 14)}L${P2(6, zb + 26)}L0 ${N(zb + 38)}L${P2(-6, zb + 26)}Z`, 'c-gold l3'); P.c(0, zb + 26, 2.6, 'c-red-d');
    z += 122;
    // 塔脖子
    band(41, z, 41, z + 5, 'c-marble l2'); band(35, z + 5, 35, z + 15, 'c-marble-d l3'); band(40, z + 15, 40, z + 21, 'c-marble l2'); band(41, z, 40, z + 21, 'g-drum');
    z += 21;
    // 十三天
    const r0 = 29, r1 = 17, H13 = 74;
    P.p(poly([[-r0, z], [r0, z], [r1, z + H13], [-r1, z + H13]]), 'c-marble l2'); P.p(poly([[-r0, z], [r0, z], [r1, z + H13], [-r1, z + H13]]), 'g-drum');
    let rings = ''; for (let i = 1; i <= 13; i++) { const zz = z + H13 * i / 14, rr = r0 + (r1 - r0) * i / 14; rings += line(arc(rr, zz)); }
    P.p(rings, 'l3');
    z += H13;
    // 华盖、铃
    P.p(`M${P2(-38, z)}C${P2(-30, z + 8)} ${P2(-12, z + 12)} 0 ${N(z + 12)}C${P2(12, z + 12)} ${P2(30, z + 8)} ${P2(38, z)}Z`, 'g-gold l2');
    let bells = ''; for (let x = -34; x <= 34; x += 8.5) bells += `M${P2(x - 1.6, z)}h3.2v-3.4h-3.2Z`;
    P.p(bells, 'c-gold l3');
    z += 12;
    // 宝顶：小座、仰月、宝珠、火焰尖
    P.p(poly([[-9, z], [9, z], [7, z + 6], [-7, z + 6]]), 'g-gold l2');
    P.p(`M${P2(-11, z + 14)}C${P2(-9, z + 6)} ${P2(9, z + 6)} ${P2(11, z + 14)}C${P2(6, z + 9)} ${P2(-6, z + 9)} ${P2(-11, z + 14)}Z`, 'g-gold l2');
    P.c(0, z + 19, 6, 'g-gold l2');
    P.p(`M${P2(-2.5, z + 24.5)}C${P2(-3, z + 30)} 0 ${N(z + 33)} 0 ${N(z + 37)}C0 ${N(z + 33)} ${P2(3, z + 30)} ${P2(2.5, z + 24.5)}Z`, 'g-gold l3');
    return { g: P.s(), box: [-106, 106, -2, z + 40] };
  }

  /* ── 长城：顺山脊的一道城墙（墙身侧面、外侧垛口），隔一段一座敌楼（方台、券窗、台顶垛口，个别上面一间铺房）。
     ridge：山脊折线（左→右，和山同一坐标）；towers：敌楼位置（0..1，按弧长）；huts：带铺房的那几座 ── */
  function greatwall(ridge, o = {}) {
    const P = Pen(), h = o.h ?? 10, sink = h * 0.3, mer = h * 0.5;
    const L = [0]; for (let i = 1; i < ridge.length; i++) L.push(L[i - 1] + Math.hypot(ridge[i][0] - ridge[i - 1][0], ridge[i][1] - ridge[i - 1][1]));
    const total = L[L.length - 1], at = s => { let i = 1; while (i < L.length - 1 && L[i] < s) i++; const f = (s - L[i - 1]) / ((L[i] - L[i - 1]) || 1); return [ridge[i - 1][0] + (ridge[i][0] - ridge[i - 1][0]) * f, ridge[i - 1][1] + (ridge[i][1] - ridge[i - 1][1]) * f]; };
    const n = Math.max(2, Math.round(total / (h * 0.55))), pts = Array.from({ length: n + 1 }, (_, i) => at(total * i / n));
    const base = pts.map(([x, y]) => [x, y - sink]), top = pts.map(([x, y]) => [x, y - sink + h]);
    P.p(poly([...base, ...top.slice().reverse()]), 'c-gw l2');
    P.p(poly([...base, ...base.map(([x, y]) => [x, y + h * 0.45]).reverse()]), 'c-shade');
    let m = ''; for (let i = 1; i < top.length - 1; i += 2) { const [x, y] = top[i]; m += `M${P2(x - mer * 0.34, y - 0.2)}v${N(mer)}h${N(mer * 0.68)}v${N(-mer)}Z`; }
    P.p(m, 'c-gw l3');
    for (const f of o.towers || []) {
      const [x, y] = at(total * f), w = h * 1.5, th = h * 2.3, yb = y - sink * 1.6;
      P.p(poly([[x - w, yb], [x + w, yb], [x + w * 0.95, yb + th], [x - w * 0.95, yb + th]]), 'c-gw l2');
      P.p(poly([[x + w * 0.3, yb], [x + w, yb], [x + w * 0.95, yb + th], [x + w * 0.28, yb + th]]), 'c-shade');
      let win = ''; for (const dx of [-0.55, 0, 0.55]) { const cx = x + dx * w, ww = w * 0.15, wy = yb + th * 0.4, hh = th * 0.24; win += `M${P2(cx - ww, wy)}v${N(hh)}a${N(ww)} ${N(ww)} 0 0 0 ${N(ww * 2)} 0v${N(-hh)}Z`; }
      P.p(win, 'c-ink');
      let tm = ''; for (let k = 0; k < 5; k++) { const cx = x - w * 0.8 + k * w * 0.4; tm += `M${P2(cx - mer * 0.3, yb + th)}v${N(mer * 0.9)}h${N(mer * 0.6)}v${N(-mer * 0.9)}Z`; }
      P.p(tm, 'c-gw l3');
      if ((o.huts || []).includes(f)) {
        const hy = yb + th, hw = w * 0.55, hh = h * 1.1;
        P.r(x - hw, hy, hw * 2, hh, 'c-gw l3');
        P.p(`M${P2(x - hw * 1.45, hy + hh)}Q${P2(x - hw * 0.6, hy + hh + h * 0.2)} ${P2(x - hw * 0.5, hy + hh + h * 0.95)}H${N(x + hw * 0.5)}Q${P2(x + hw * 0.6, hy + hh + h * 0.2)} ${P2(x + hw * 1.45, hy + hh)}Z`, 'c-gray-d l3');
      }
    }
    return { g: P.s() };
  }

  /* ── 十七孔桥：长约 150 米，十七个半圆券洞，正中最大、往两头渐小；桥面中间拱起，两头引桥落到岸上；
        券脸一圈券石，券洞里看得见一抹券底阴影；桥面上汉白玉栏杆，望柱头是小石狮。局部原点在桥中线、水面。 ── */
  function shiqikong(o = {}) {
    const P = Pen(), n = 17, half = (n - 1) / 2, s0 = 76, Hc = 76, He = 36;
    const spans = Array.from({ length: n }, (_, i) => s0 * (1 - 0.05 * Math.abs(i - half)));
    const piers = Array.from({ length: n - 1 }, (_, i) => (spans[i] + spans[i + 1]) * 0.15);
    const W = spans.reduce((a, b) => a + b, 0) + piers.reduce((a, b) => a + b, 0), Lh = W / 2 + 60, ramp = 60;
    const deck = x => Math.abs(x) <= Lh ? He + (Hc - He) * (1 - (x / Lh) ** 2) : Math.max(0, He * (1 - (Math.abs(x) - Lh) / ramp));
    const arches = []; let x = -W / 2;
    for (let i = 0; i < n; i++) { arches.push({ c: x + spans[i] / 2, r: spans[i] / 2, pw: piers[i] || 0, pr: x + spans[i] }); x += spans[i] + (piers[i] || 0); }
    const spring = 5;
    const top = samp(t => { const xx = -Lh - ramp + (2 * (Lh + ramp)) * t; return [xx, deck(xx)]; }, 90);
    let d = poly([[-Lh - ramp, -2], ...top, [Lh + ramp, -2]]);
    for (const a of arches) d += `M${P2(a.c - a.r, -2)}V${N(spring)}A${N(a.r)} ${N(a.r)} 0 0 0 ${P2(a.c + a.r, spring)}V-2Z`;
    P.p(d, 'c-marble l1', ' fill-rule="evenodd"');
    // 券底阴影、券石
    let sh = '', ring = '', jt = '';
    for (const a of arches) {
      sh += `M${P2(a.c - a.r, spring)}A${N(a.r)} ${N(a.r)} 0 0 0 ${P2(a.c + a.r, spring)}L${P2(a.c + a.r * 0.9, spring)}A${N(a.r * 0.9)} ${N(a.r * 0.82)} 0 0 1 ${P2(a.c - a.r * 0.9, spring)}Z`;
      const R2 = a.r + Math.max(3, a.r * 0.14);
      ring += `M${P2(a.c - R2, spring)}A${N(R2)} ${N(R2)} 0 0 0 ${P2(a.c + R2, spring)}`;
      for (let k = 1; k < 12; k++) { const t = Math.PI * k / 12; jt += `M${P2(a.c - Math.cos(t) * a.r, spring + Math.sin(t) * a.r)}L${P2(a.c - Math.cos(t) * R2, spring + Math.sin(t) * R2)}`; }
    }
    P.p(sh, 'c-shade2'); P.p(ring, 'l3'); P.p(jt, 'l-cun');
    // 分水尖：桥墩脚下
    let cw = ''; for (const a of arches.slice(0, -1)) { const c = a.pr + a.pw / 2, w = a.pw * 0.6; cw += `M${P2(c - w, -1)}L${P2(c, a.pw * 0.7)}L${P2(c + w, -1)}Z`; }
    P.p(cw, 'c-marble-d l3');
    // 桥面仰天石、栏杆
    const dk = samp(t => { const xx = -Lh - ramp + 2 * (Lh + ramp) * t; return [xx, deck(xx)]; }, 120);
    P.p('M' + dk.map(([xx, y]) => P2(xx, y - 4.5)).join('L'), 'l2');
    const rh = 10;
    let posts = '', heads = '', rail = '', pan = '';
    for (let xx = -Lh - ramp * 0.6; xx <= Lh + ramp * 0.6; xx += 23) { const y = deck(xx); posts += `M${P2(xx - 1.3, y)}h2.6v${N(rh)}h-2.6Z`; heads += `M${P2(xx - 1.9, y + rh)}a1.9 2.2 0 0 0 3.8 0Z`; }
    for (const f of [0.18, 0.5, 0.86]) rail += 'M' + dk.filter(([xx]) => Math.abs(xx) <= Lh + ramp * 0.6).map(([xx, y]) => P2(xx, y + rh * f)).join('L');
    pan = poly([...dk.filter(([xx]) => Math.abs(xx) <= Lh + ramp * 0.6).map(([xx, y]) => [xx, y]), ...dk.filter(([xx]) => Math.abs(xx) <= Lh + ramp * 0.6).reverse().map(([xx, y]) => [xx, y + rh * 0.5])]);
    P.p(pan, 'c-marble l3'); P.p(rail, 'l3'); P.p(posts, 'c-marble l3'); P.p(heads, 'c-marble l3');
    return { g: P.s(), box: [-Lh - ramp - 4, Lh + ramp + 4, -4, Hc + rh + 6], arches, deck, Lh, ramp };
  }

  /* ── 中国国家博物馆（西立面）：正中方柱柱廊，两翼竖向壁柱、窗带，檐口一道黄琉璃，花岗石基座台阶 ── */
  function guobo(o = {}) {
    const P = Pen(), wing = o.wing ?? 620, pw = 290, H1 = 250, Hp = 300, cols = 10;
    const rect = (x, y, w, h) => `M${P2(x, y)}h${N(w)}v${N(h)}h${N(-w)}Z`;
    for (const s of [-1, 1]) {
      const x0 = s > 0 ? pw : -pw - wing, x1 = x0 + wing;
      P.p(rect(x0, 0, wing, 22), 'c-stone-d l2');
      P.p(rect(x0, 22, wing, H1 - 40), 'c-gb l2');
      let pil = '', win = '';
      for (let xx = x0 + 26; xx < x1 - 30; xx += 52) { pil += rect(xx, 22, 10, H1 - 40); win += rect(xx + 19, 56, 16, 120); }
      P.p(win, 'c-gbw l3'); P.p(pil, 'c-gb-l l3');
      P.p(rect(x0, 22, wing, H1 - 40), s > 0 ? 'c-shade' : 'c-light');
      P.p(rect(x0 - 4, H1 - 18, wing + 8, 12), 'c-gb l2');
      P.p(rect(x0 - 8, H1 - 6, wing + 16, 14), 'c-yellow l2');
      let tl = ''; for (let xx = x0 - 6; xx < x1 + 6; xx += 5) tl += `M${P2(xx, H1 - 6)}v14`; P.p(tl, 'l3');
    }
    // 柱廊：台阶、退进去的门墙（暗）、方柱、檐部、黄琉璃檐口、女儿墙
    P.p(rect(-pw - 10, 0, pw * 2 + 20, 34), 'c-stone l2');
    let st = ''; for (let y = 6; y < 34; y += 6) st += `M${P2(-pw - 10, y)}h${N(pw * 2 + 20)}`; P.p(st, 'l3');
    P.p(rect(-pw + 24, 34, pw * 2 - 48, Hp - 74), 'c-gb-d l2');
    let door = ''; for (let i = 0; i < 5; i++) door += rect(-150 + i * 66, 34, 36, 110); P.p(door, 'c-gbw l3');
    let col = '', csh = ''; const sp = (pw * 2 - 70) / (cols - 1);
    for (let i = 0; i < cols; i++) { const xx = -pw + 35 + i * sp; col += rect(xx - 11, 34, 22, Hp - 74); csh += rect(xx + 3, 34, 8, Hp - 74); }
    P.p(col, 'c-gb-l l2'); P.p(csh, 'c-shade');
    P.p(rect(-pw - 6, Hp - 40, pw * 2 + 12, 24), 'c-gb l2');
    P.p(rect(-pw - 12, Hp - 16, pw * 2 + 24, 16), 'c-yellow l2');
    let tl = ''; for (let xx = -pw - 10; xx < pw + 10; xx += 5) tl += `M${P2(xx, Hp - 16)}v16`; P.p(tl, 'l3');
    P.p(rect(-pw + 30, Hp, pw * 2 - 60, 18), 'c-gb l2');
    return { g: P.s(), box: [-pw - wing - 12, pw + wing + 12, -2, Hp + 22] };
  }

  /* ── 永定门（中轴线南端）：城台正中一个券门，台上城楼重檐歇山，灰筒瓦绿琉璃剪边 ── */
  function yongding(o = {}) {
    const P = Pen(), tw = 150, bw = 158, H = 92;
    P.p(poly([[-bw, 0], [bw, 0], [tw, H], [-tw, H]]), 'c-gw l1');
    P.p(poly([[-bw, 0], [bw, 0], [tw, H], [-tw, H]]), 'p-brick');
    P.p(poly([[bw * 0.35, 0], [bw, 0], [tw, H], [tw * 0.33, H]]), 'c-shade');
    P.p(poly([[-bw - 1, 0], [bw + 1, 0], [bw + 0.6, 8], [-bw - 0.6, 8]]), 'c-stone l2');
    P.p(`M${P2(-30, 0)}V44A30 30 0 0 0 ${P2(30, 44)}V0Z`, 'c-ink');
    P.p(`M${P2(-37, 0)}V44A37 37 0 0 0 ${P2(37, 44)}V0`, 'l2');
    parapet(P, -tw, tw, H, 90, { cls: 'c-gw', parapet: 10, merlon: 11 });
    const hall = window.JH.BUILD.hall({ bays: [58, 48, 40], colH: 54, platform: 0, double: true, type: 'xs', tone: 'gray', trim: 'green', tile: 4.4, beasts: 3, per: 3, caihua: 'xuanzi', rw: 70 });
    P.g(hall.g, ` transform="translate(0 ${H + 4})"`);
    return { g: P.s(), box: [-bw - 8, bw + 8, -2, H + 4 + hall.box[3]] };
  }

  /* ── 宫城金顶：太和门、太和殿（三台）、中和殿、保和殿依次往后，两侧体仁阁、弘义阁，殿间红墙黄瓦；
        远处东西六宫一片屋顶。越往后抬得越高、略小（散点透视）。局部原点在太和门台基中线。 ── */
  function redWall(P, x0, x1, y0, h) {
    P.r(x0, y0, x1 - x0, h, 'c-red l2');
    P.r(x0, y0, x1 - x0, h * 0.12, 'c-stone l3');
    P.p(`M${P2(x0 - 3, y0 + h)}H${N(x1 + 3)}L${P2(x1 + 1, y0 + h + h * 0.16)}H${N(x0 - 1)}Z`, 'c-yellow l2');
    let tl = ''; for (let x = x0; x < x1; x += 4) tl += `M${P2(x, y0 + h)}v${N(h * 0.16)}`; P.p(tl, 'l3');
  }
  function santai(P, hw, h) {
    let y = 0;
    for (let i = 0; i < 3; i++) { const w = hw - i * h * 0.9; y = A.xumizuo(P, y, h, w, { chishou: true }); A.balustrade(P, -w + 2, -16, y, h * 0.42, h * 0.5); A.balustrade(P, 16, w - 2, y, h * 0.42, h * 0.5); }
    A.stairs(P, 0, y, 16, 6);
    return y;
  }
  function gongcheng(o = {}) {
    const P = Pen(), J = window.JH, B = J.BUILD, put = (L, x, y) => P.g(L.g, ` transform="translate(${N(x)} ${N(y)})"`);
    const hall = (k, opt, lod = 0.4) => J.withLod(lod, () => B.hall(B.scaled(opt, k)));
    const small = (type, nb) => ({ bays: Array.from({ length: nb }, (_, i) => i ? 50 : 60), colH: 62, platform: 12, type, tile: 7, beasts: 3, per: 1 });
    // 后两排：东西六宫、乾清宫一线，一座挨一座，只露屋顶和殿身上段；排与排之间红墙黄瓦
    const R = J.rng(17);
    for (const [y, k, n0, step, off] of [[470, 0.34, 13, 205, 0], [410, 0.4, 11, 235, 117]]) {
      for (let i = 0; i < n0; i++) {
        if (R() < 0.16) continue;
        const x = -((n0 - 1) / 2) * step + i * step + off * (i % 2 ? 0.25 : -0.25) + (R() - 0.5) * 40, kk = k * (0.86 + R() * 0.3), two = R() < 0.18;
        put(hall(kk, two ? { bays: [56, 48, 44], colH: 92, platform: 10, double: true, type: R() < 0.5 ? 'xs' : 'wd', tile: 7, beasts: 3, per: 1 } : small(R() < 0.33 ? 'wd' : 'xs', 3 + (i % 2))), x, y + (R() - 0.5) * 16);
      }
      redWall(P, -1320, 1320, y - 10, 15);
    }
    for (const [x, y, w] of [[-760, 452, 260], [520, 452, 220], [-180, 392, 200], [980, 392, 240]]) P.g(window.JH.NATURE.grove({ w, h: 46, seed: Math.round(x + y), kind: 'mixed', marks: false, density: 0.8 }).g, ` transform="translate(${N(x)} ${N(y)})"`);
    P.e(0, 462, 1420, 112, 'g-hazer');   // 后排薄雾：四周淡出，滚动错层时不露硬边
    // 保和殿、中和殿
    for (const s of [-1, 1]) for (const [x, k] of [[420, 0.44], [660, 0.42], [900, 0.42], [1140, 0.4]]) put(hall(k, small('xs', 4)), s * x, 330);
    put(hall(0.8, { bays: [84, 74, 66, 56], colH: 80, platform: 40, rail: 12, double: true, type: 'xs', tile: 6.5, beasts: 7, per: 3 }, 0.5), 0, 330);
    redWall(P, -1320, -330, 320, 20); redWall(P, 330, 1320, 320, 20);
    put(hall(0.62, { bays: [78], colH: 66, platform: 26, rail: 9, type: 'cj', tile: 5, beasts: 3, rh: 74, leaves: 6 }, 0.5), 0, 286);
    // 太和殿与三台，两侧廊庑
    const corr = { bays: [44, 44, 44, 44, 44, 44, 44], colH: 52, platform: 10, type: 'xs', tile: 6, beasts: 2, per: 1, rh: 40, rw: 200 };
    for (const s of [-1, 1]) put(hall(0.62, corr), s * 900, 170);
    const Q = Pen(); const ty = J.withLod(0.6, () => santai(Q, 330, 22)); P.g(Q.s(), ' transform="translate(0 150)"');
    put(hall(1, { bays: [84, 74, 66, 56, 50], colH: 92, platform: 0, double: true, type: 'wd', tile: 6, beasts: 9, per: 4 }, 0.75), 0, 150 + ty);
    // 体仁阁、弘义阁，前排廊庑与红墙
    for (const s of [-1, 1]) put(hall(0.62, { bays: [70, 60, 54], colH: 120, platform: 24, rail: 8, double: true, type: 'wd', tile: 6, beasts: 5, per: 3 }, 0.5), s * 560, 96);
    for (const s of [-1, 1]) put(hall(0.55, corr), s * 1040, 70);
    redWall(P, -1320, -380, 60, 30); redWall(P, 380, 1320, 60, 30);
    // 太和门（最前）
    put(hall(0.78, { bays: [84, 74, 66, 56], colH: 82, platform: 30, rail: 10, double: true, type: 'xs', tile: 6.5, beasts: 7, per: 3 }, 0.6), 0, 0);
    return { g: P.s(), box: [-1330, 1330, -4, 600] };
  }

  /* ── 午门：宫城正门。红色城台正中三个券门（中大边小），台上正楼九间重檐庑殿；
        两翼雁翅楼往前伸，翼端各一座城台、上立重檐方亭（阙亭）。局部原点在城台底边中线。 ── */
  function wumen(o = {}) {
    const P = Pen(), J = window.JH, B = J.BUILD, H = 120, hw = 300, lod = o.lod ?? 0.7;
    const block = (x0, x1, h) => {
      P.p(poly([[x0, 0], [x1, 0], [x1 - 3, h], [x0 + 3, h]]), 'c-wallr l1');
      P.p(poly([[x0, 0], [x1, 0], [x1 - 1, h * 0.28], [x0 + 1, h * 0.28]]), 'c-shade');
      P.p(poly([[x0 - 1, 0], [x1 + 1, 0], [x1 + 0.6, 9], [x0 - 0.6, 9]]), 'c-stone l2');
      parapet(P, x0 + 3, x1 - 3, h, 100, { parapet: 12, merlon: 13 });
    };
    block(-hw, hw, H);
    for (const [x, w, h] of [[0, 27, 80], [-104, 20, 66], [104, 20, 66]]) { P.p(`M${P2(x - w, 0)}V${N(h - w)}A${N(w)} ${N(w)} 0 0 0 ${P2(x + w, h - w)}V0Z`, 'c-ink'); P.p(`M${P2(x - w - 5, 0)}V${N(h - w)}A${N(w + 5)} ${N(w + 5)} 0 0 0 ${P2(x + w + 5, h - w)}V0`, 'l2'); }
    const main = J.withLod(lod, () => B.hall({ bays: [64, 58, 54, 50, 46], colH: 74, platform: 8, rail: 7, double: true, type: 'wd', tile: 6, beasts: 7, per: 3 }));
    P.g(main.g, ` transform="translate(0 ${H + 2})"`);
    for (const s of [-1, 1]) {
      const cx = s * 440;
      block(cx - 112, cx + 112, H * 0.94);
      const pav = J.withLod(lod, () => B.hall({ bays: [58], colH: 54, platform: 6, rail: 6, double: true, type: 'cj', tile: 5, beasts: 3, rh: 58, leaves: 4 }));
      P.g(pav.g, ` transform="translate(${N(cx)} ${N(H * 0.94 + 2)})"`);
    }
    return { g: P.s(), box: [-560, 560, -2, H + 2 + main.box[3]] };
  }

  /* ── 故宫宫城：后面一片金顶（宫城金顶群，往后抬高），前面一道宫城城墙，正中午门，两角各一座角楼（放大一些，好认），墙外筒子河 ──
     局部原点在城墙脚下中线 */
  function gugong(o = {}) {
    const P = Pen(), J = window.JH, half = o.half ?? 1500, wallH = 100, k = o.roofScale ?? (half / 1340), jk = o.jiaoScale ?? 1.4;
    const back = gongcheng();
    P.g(back.g, ` transform="translate(0 ${N(150)}) scale(${N(k)})"`);
    // 城墙（午门两侧）
    for (const s of [-1, 1]) { const x0 = s < 0 ? -half : 300, x1 = s < 0 ? -300 : half; wallBody(P, x0, x1, 0, wallH, { cornerL: s < 0, cornerR: s > 0 }); }
    // 两角角楼（先画塔，再画雉堞挡住塔基）
    for (const s of [-1, 1]) { const jl = J.withLod(o.lod ?? 0.75, () => jiaolou({ wallH: 0 })); P.g(jl.g, ` transform="translate(${N(s * (half - 70))} ${wallH}) scale(${N(jk)})"`); }
    for (const s of [-1, 1]) { const x0 = s < 0 ? -half : 300, x1 = s < 0 ? -300 : half; parapet(P, x0, x1, wallH, wallH, { cornerL: s < 0, cornerR: s > 0 }); }
    // 午门
    P.g(wumen({ lod: o.lod ?? 0.7 }).g, ' transform="translate(0 -2)"');
    // 筒子河：城墙外一道水
    for (const s of [-1, 1]) { const x0 = s < 0 ? -half - 30 : 560, x1 = s < 0 ? -560 : half + 30; P.r(x0, -16, x1 - x0, 15, 'c-water'); let wv = ''; for (let x = x0 + 6; x < x1 - 14; x += 22) wv += `M${P2(x, -8)}q5 2 10 0q5 -2 10 0`; P.p(wv, 'l-water'); }
    return { g: P.s(), box: [-half - 130, half + 130, -18, 150 + back.box[3] * k] };
  }

  window.JH.LM = Object.assign(window.JH.LM || {}, { jiaolou, foxiang, yufeng, baita, shiqikong, guobo, yongding, gongcheng, wumen, gugong });
  window.JH.LMX2 = { greatwall, redWall, santai };
  window.JH.LMX = { wallBody, parapet, bodySeg };
})();
