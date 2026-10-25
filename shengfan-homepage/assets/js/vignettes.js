/* 晟帆主页 · 各章插图（与首屏同一套界画青绿画法）
   红色足迹小长卷（香山双清别墅、卢沟桥与宛平城、李大钊故居、北大红楼、党史展览馆）、
   团建「百川汇流」京郊山水、会场平面（仿样式雷图档）。
   建筑在局部坐标里画（u 向右、v 向上，原点在底部中线），放置时套 translate(x y) scale(s -s)。 */
(function () {
  'use strict';
  const A = window.ART2, { N, pt, poly, rng, Pen, sub } = A, F = window.FLORA, H = window.SCENE.HELP;
  const at = H.at;

  /* ── 附加渐变与纹理（放进页面 defs） ─────────────── */
  function lin(id, stops, x1 = 0, y1 = 0, x2 = 0, y2 = 1) {
    return `<linearGradient id="${id}" x1="${x1}" y1="${y1}" x2="${x2}" y2="${y2}">${stops.map(([o, c, a = 1]) => `<stop offset="${o}" stop-color="${c}"${a < 1 ? ` stop-opacity="${a}"` : ''}/>`).join('')}</linearGradient>`;
  }
  function defs() {
    return '<svg width="0" height="0" style="position:absolute" aria-hidden="true"><defs>' +
      lin('g-redbrick', [[0, '#8a3326'], [.5, '#a5432f'], [1, '#b4523a']]) +
      lin('g-gbrick', [[0, '#7f8180'], [.6, '#9a9b97'], [1, '#aaaaa5']]) +
      lin('g-plaster', [[0, '#d9d4c8'], [1, '#efebe1']]) +
      lin('g-slate', [[0, '#5d6466'], [1, '#868c8c']]) +
      lin('g-river', [[0, '#bccfcc'], [1, '#dfe4d8']]) +
      lin('g-vsky', [[0, '#efe6d3'], [.7, '#f1e8d6'], [1, '#ece0c6']]) +
      lin('g-ground2', [[0, '#e3d6b6'], [1, '#d8c8a2']]) +
      '<pattern id="p-redbrick" width="5" height="2" patternUnits="userSpaceOnUse"><path d="M0 0H5M0 1H5M0 0V1M2.5 1V2" fill="none" stroke="#f1d2b8" stroke-width=".16" opacity=".45"/></pattern>' +
      '<pattern id="p-gbrick" width="5" height="2" patternUnits="userSpaceOnUse"><path d="M0 0H5M0 1H5M0 0V1M2.5 1V2" fill="none" stroke="#4b4c4a" stroke-width=".16" opacity=".4"/></pattern>' +
      '<pattern id="p-slate" width="3" height="2" patternUnits="userSpaceOnUse"><path d="M0 2H3M1.5 0V1M0 1H3" fill="none" stroke="#3f4546" stroke-width=".14" opacity=".45"/></pattern>' +
      '</defs></svg>';
  }

  /* ── 红色地标（局部坐标） ─────────────────────── */

  // 北大红楼：四层红砖楼，白色腰线和窗框，中段略凸出、正中拱门；石板瓦四坡顶上开老虎窗、两根烟囱
  function honglou() {
    const P = Pen(), hw = 96, top = 48;
    P.r(-hw - 2, 0, hw * 2 + 4, 3, 'f-stone k');
    P.p('M-98 1.5H98', 'kt');
    P.r(-hw, 3, hw * 2, top - 3, 'f-redbrick k');
    P.r(-hw, 3, hw * 2, top - 3, 'f-redbrickP');
    P.r(62, 3, hw - 62, top - 3, 'f-shadeL');
    // 中段凸出：两侧壁柱略深
    P.p(`M-15 3V${top + 4}M15 3V${top + 4}`, 'kf');
    P.r(-15, 3, 30, top - 3, 'f-shadeL');
    // 腰线
    for (const y of [14.6, 25.8, 37]) P.r(-hw, y, hw * 2, 1.2, 'f-plaster kt');
    // 窗：每层十九樘，底层正中让给门
    let fr = '', gl = '', lt = '';
    for (const [y, h] of [[5.4, 7.6], [17, 7], [28.2, 7], [39.4, 6.8]]) for (let i = 0; i < 19; i++) {
      const u = -86 + i * (172 / 18); if (Math.abs(u) < 7 && y < 10) continue;
      fr += `M${N(u - 2.5)} ${N(y)}h5v${h}h-5Z`; gl += `M${N(u - 1.8)} ${N(y + 0.7)}h3.6v${N(h - 1.4)}h-3.6ZM${N(u)} ${N(y + 0.7)}v${N(h - 1.4)}`;
      lt += `M${N(u - 3.1)} ${N(y + h + 0.5)}h6.2`;
    }
    P.p(fr, 'f-plaster kt'); P.p(gl, 'f-win kt'); P.p(lt, 'kf');
    // 正门：石券拱门，台阶
    P.p('M-6 3V10.4A6 6 0 0 0 6 10.4V3Z', 'f-plaster k');
    P.p('M-4.4 3V10.2A4.4 4.4 0 0 0 4.4 10.2V3Z', 'f-dark kf');
    P.p('M-8 0H8L9 -0.1M-9 0', 'kt');
    // 檐口与女儿墙，中段起一道山花
    P.r(-hw - 1, top, hw * 2 + 2, 2.2, 'f-plaster k');
    P.p(poly([[-16, top + 2.2], [16, top + 2.2], [16, top + 5.4], [0, top + 9], [-16, top + 5.4]]), 'f-redbrick k');
    P.p(`M-12 ${top + 5}L0 ${top + 8}L12 ${top + 5}`, 'kt');
    // 石板瓦四坡顶、老虎窗、烟囱
    P.p(poly([[-hw - 2, top + 2.2], [hw + 2, top + 2.2], [hw - 12, top + 12], [-hw + 12, top + 12]]), 'f-slate k');
    P.p(poly([[-hw - 2, top + 2.2], [hw + 2, top + 2.2], [hw - 12, top + 12], [-hw + 12, top + 12]]), 'f-slateP');
    let dm = ''; for (const u of [-64, -38, 38, 64]) dm += `M${u - 3.4} ${top + 3.6}V${top + 7.6}L${u} ${top + 10.2}L${u + 3.4} ${top + 7.6}V${top + 3.6}Z`;
    P.p(dm, 'f-plaster kt');
    let dg = ''; for (const u of [-64, -38, 38, 64]) dg += `M${u - 1.8} ${top + 4.2}h3.6v3h-3.6Z`; P.p(dg, 'f-win');
    for (const u of [-74, 74]) { P.r(u - 2.4, top + 8, 4.8, 7.4, 'f-redbrick k'); P.r(u - 3, top + 15.4, 6, 1.2, 'f-plaster kt'); }
    return { g: P.s(), box: [-100, 100, 0, 66] };
  }

  // 李大钊故居：胡同里的四合院——灰砖院墙、广亮大门（红漆门、门簪、门墩、门匾），院里北房屋脊和老槐树
  function siheyuan() {
    const P = Pen();
    // 院里北房：露出屋脊
    P.g(sub(Q => A.roof(Q, 22, 46, 9, 38, { tone: 'G', band: 1, tile: 1.6, soffit: false, lite: false })), ' transform="translate(0 2)"');
    for (const s of [-1, 1]) {
      const a = s < 0 ? -64 : 16, b = s < 0 ? -16 : 64;
      P.r(a, 0, b - a, 19, 'f-gbrick k'); P.r(a, 0, b - a, 19, 'f-gbrickP');
      P.r(a, 0, b - a, 2.4, 'f-stoneD kt');
      P.p(poly([[a - 1.2, 19], [b + 1.2, 19], [b, 22], [a, 22]]), 'f-roofG k');
      let t = ''; for (let x = a + 1; x < b; x += 1.6) t += `M${N(x)} 19.3v2.4`; P.p(t, 'kt');
      if (s > 0) P.r(16, 0, 48, 19, 'f-shadeL');
    }
    // 门楼
    P.r(-16, 0, 32, 25, 'f-gbrick k'); P.r(-16, 0, 32, 25, 'f-gbrickP');
    P.r(-11.5, 1.4, 23, 21, 'f-stoneD kt');
    P.r(-9.4, 1.4, 18.8, 19.2, 'f-col k');
    P.p('M0 1.4V20.6M-9.4 11H9.4', 'kt');
    // 门钹与门簪
    P.c(-1.8, 10.4, 0.8, 'f-gold kt'); P.c(1.8, 10.4, 0.8, 'f-gold kt');
    P.r(-9.4, 20.6, 18.8, 1.6, 'f-paintG kt');
    P.p('M-5.4 22.2h3v2.4h-3ZM2.4 22.2h3v2.4h-3Z', 'f-gold kt');
    // 门匾
    P.r(-6, 15.4, 12, 3.6, 'f-dark kt'); P.p('M-4.6 17.2h9.2', 'kgt');
    // 门墩（抱鼓石）
    for (const u of [-11.2, 11.2]) { P.r(u - 2.2, 0, 4.4, 2.6, 'f-stone kf'); P.c(u, 5, 2.4, 'f-stone kf'); P.c(u, 5, 1.2, 'kt'); }
    // 台阶
    P.p('M-13 0H13L14 -1.4H-14Z', 'f-stone kt');
    A.roof(P, 25, 23, 10.4, 15, { tone: 'G', band: 1.2, tile: 1.4, lift: 1.2, soffit: true, lite: false });
    return { g: P.s(), box: [-66, 66, -2, 40] };
  }

  // 香山双清别墅：灰瓦小院，前檐红柱廊；右侧六角亭，前面一池双清泉
  function shuangqing() {
    const P = Pen();
    // 池
    P.p('M-30 0C-26-5 30-6 44-2C50 0 46 2 40 2H-26C-31 2-32 1-30 0Z', 'f-pool kt');
    P.p('M-18-2.4h16M8-3.4h22', 'kl');
    // 台基
    P.p(poly([[-40, 0], [40, 0], [39, 3], [-39, 3]]), 'f-stone k');
    // 屋身：白墙、窗、红柱廊
    P.r(-35, 3, 70, 13, 'f-plaster k');
    let w = ''; for (const u of [-26, -14, 14, 26]) w += `M${u - 3.4} 6h6.8v7h-6.8Z`; P.p(w, 'f-lattice kt');
    P.r(-4, 3, 8, 10.6, 'f-col kf');
    let c = ''; for (let i = 0; i < 8; i++) { const u = -33 + i * (66 / 7); c += `M${N(u - 0.9)} 3h1.8v13h-1.8Z`; } P.p(c, 'f-col kf');
    P.r(-36, 16, 72, 1.8, 'f-paintG kt');
    A.roof(P, 17.8, 44, 11.5, 28, { tone: 'G', band: 1.3, tile: 1.6, lift: 2.2, lite: false });
    // 六角亭
    P.g(sub(Q => {
      Q.p(poly([[-11, 0], [11, 0], [10, 2], [-10, 2]]), 'f-stone k');
      Q.p('M-8 2v10M-2.7 2v10M2.7 2v10M8 2v10', 'f-col k');
      Q.p('M-9 9.6H9', 'kt');
      A.roof(Q, 12, 13.5, 12.5, 0, { tone: 'G', band: 1, tile: 1.3, lift: 2.2, lite: false });
      A.baoding(Q, 24.5, 0.7);
    }), ' transform="translate(60 1)"');
    return { g: P.s(), box: [-46, 76, -6, 44] };
  }

  // 卢沟桥：十一孔联拱石桥，桥面平缓，望柱上蹲石狮；桥墩迎水面有分水尖；两端华表
  function lugou() {
    const P = Pen(), spans = [12.6, 13.6, 14.6, 15.6, 16.6, 17.6, 16.6, 15.6, 14.6, 13.6, 12.6], pier = 6;
    const total = spans.reduce((a, b) => a + b, 0) + pier * (spans.length + 1), L = -total / 2, R = total / 2;
    const deck = u => 19 + 1.6 * (1 - (u / R) ** 2);
    let x = L, holes = '', rings = '', cut = '';
    for (const s of spans) {
      cut += `M${N(x + pier * 0.1)} 0L${N(x + pier / 2)} -2.6L${N(x + pier * 0.9)} 0`;
      x += pier;
      const spring = 3.6;
      holes += `M${N(x)} 0V${spring}A${N(s / 2)} ${N(s / 2)} 0 0 0 ${N(x + s)} ${spring}V0Z`;
      rings += `M${N(x - 1.2)} ${spring}A${N(s / 2 + 1.2)} ${N(s / 2 + 1.2)} 0 0 0 ${N(x + s + 1.2)} ${spring}`;
      x += s;
    }
    cut += `M${N(x + pier * 0.1)} 0L${N(x + pier / 2)} -2.6L${N(x + pier * 0.9)} 0`;
    // 桥身（两端顺坡落地）
    let top = `M${N(L - 16)} 0L${N(L)} ${N(deck(L) - 2)}`;
    for (let u = L; u <= R; u += 8) top += `L${N(u)} ${N(deck(u))}`;
    top += `L${N(R)} ${N(deck(R))}L${N(R + 16)} 0Z`;
    P.p(top + holes, 'f-ashlar k', ' fill-rule="evenodd"');
    P.p(top + holes, 'f-ashlarP', ' fill-rule="evenodd"');
    P.p(rings, 'kt'); P.p(cut, 'f-stone kt');
    let deckL = ''; for (let u = L; u <= R; u += 8) deckL += (u === L ? 'M' : 'L') + `${N(u)} ${N(deck(u) - 1.6)}`; P.p(deckL, 'kt');
    // 栏板、望柱、石狮
    let panel = '', posts = '', lions = '';
    for (let u = L - 10; u <= R + 10; u += 4.6) {
      const y = u < L ? (u - L + 16) / 16 * (deck(L) - 2) : u > R ? (R + 16 - u) / 16 * deck(R) : deck(u);
      if (y < 2) continue;
      panel += `M${N(u)} ${N(y)}h4.6v2.4h-4.6Z`;
      posts += `M${N(u - 0.5)} ${N(y)}h1v4h-1Z`;
      lions += `M${N(u - 1)} ${N(y + 4)}C${N(u - 1.3)} ${N(y + 5.6)} ${N(u - 0.2)} ${N(y + 6.6)} ${N(u + 0.4)} ${N(y + 6.2)}C${N(u + 1.3)} ${N(y + 5.8)} ${N(u + 1.2)} ${N(y + 4.6)} ${N(u + 1)} ${N(y + 4)}Z`;
    }
    P.p(panel, 'f-stone kt'); P.p(posts, 'f-stone kt'); P.p(lions, 'f-stoneD kt');
    // 华表
    for (const u of [L - 20, R + 20]) {
      P.r(u - 2.4, 0, 4.8, 1.6, 'f-stone kf');
      P.r(u - 1.3, 1.6, 2.6, 30, 'f-stone k');
      P.p(`M${N(u - 5)} 27.4Q${N(u)} 29.6 ${N(u + 5)} 27.4L${N(u + 4)} 26.4Q${N(u)} 28 ${N(u - 4)} 26.4Z`, 'f-stone kf');
      P.r(u - 2.8, 31.6, 5.6, 1.4, 'f-stone kf');
      P.p(`M${N(u - 1.4)} 33C${N(u - 1.6)} 35.6 ${N(u + 1.8)} 36 ${N(u + 1.4)} 33Z`, 'f-stoneD kf');
    }
    return { g: P.s(), box: [L - 26, R + 26, -3, 38] };
  }

  // 宛平城：灰砖城墙、垛口，券门，城上单檐歇山门楼
  function wanping() {
    const P = Pen();
    P.p(poly([[-74, 0], [74, 0], [71, 26], [-71, 26]]), 'f-gbrick k');
    P.p(poly([[-74, 0], [74, 0], [71, 26], [-71, 26]]), 'f-gbrickP');
    P.p(poly([[40, 0], [74, 0], [71, 26], [40, 26]]), 'f-shadeL');
    P.p('M-7.4 0V11A7.4 7.4 0 0 0 7.4 11V0Z', 'f-dark k');
    P.p('M-9.4 0V11A9.4 9.4 0 0 0 9.4 11V0', 'kf');
    let cr = 'M-71 26H71V27.6H-71Z'; for (let u = -71; u < 69; u += 5.8) cr += `M${N(u)} 27.6h3.6v2.6h-3.6Z`; P.p(cr, 'f-gbrick kf');
    P.r(-26, 26, 52, 2, 'f-stone kt');
    A.body(P, 28, 37, 22, { cols: 7, lat: 0.6, lintel: 1.4 });
    A.dougong(P, 37, 1.3, 22, 1.8);
    A.roof(P, 38.3, 33, 12, 18, { tone: 'G', band: 1.4, tile: 1.6, lift: 2.6, lite: false });
    return { g: P.s(), box: [-76, 76, 0, 54] };
  }

  // 中国共产党历史展览馆：高台阶、通高白石柱廊、正中凸起的入口和金色额带（不画任何旗帜、徽记）
  function dangshi() {
    const P = Pen(), hw = 100;
    P.p(poly([[-hw - 8, 0], [hw + 8, 0], [hw + 4, 4], [-hw - 4, 4]]), 'f-stone k');
    P.p(poly([[-30, 0], [30, 0], [26, 9], [-26, 9]]), 'f-stone k');
    let st = ''; for (let y = 1.5; y < 9; y += 1.5) { const w = 30 - 4 * y / 9; st += `M${N(-w)} ${N(y)}H${N(w)}`; } P.p(st, 'kt');
    P.r(-hw, 4, hw * 2, 40, 'f-facade k');
    P.r(-hw + 4, 9, hw * 2 - 8, 33, 'f-recess');
    let cols = '', sh = ''; const nc = 22;
    for (let i = 0; i < nc; i++) { const u = -hw + 6 + i * ((hw * 2 - 12) / (nc - 1)); cols += `M${N(u - 2.2)} 9h4.4v33h-4.4Z`; sh += `M${N(u + 0.8)} 9h1.4v33h-1.4Z`; }
    P.p(cols, 'f-facade kf'); P.p(sh, 'f-shadeL');
    P.r(-hw - 2, 42, hw * 2 + 4, 4, 'f-facade k');
    P.r(-hw - 2, 44.6, hw * 2 + 4, 0.8, 'f-gold');
    // 正中入口：再高一层
    P.r(-30, 9, 60, 44, 'f-facade k');
    P.r(-22, 9, 44, 30, 'f-glass kt');
    let m = ''; for (let u = -22; u <= 22; u += 5.5) m += `M${N(u)} 9V39`; P.p(m + 'M-22 24H22', 'kt');
    P.r(-32, 50, 64, 4, 'f-facade k'); P.r(-32, 52.6, 64, 0.8, 'f-gold');
    return { g: P.s(), box: [-110, 110, 0, 56] };
  }

  const RED = { honglou, siheyuan, shuangqing, lugou, wanping, dangshi };
  const place = (name, x, y, h, cls = '', attrs = '') => {
    const L = RED[name](), sc = h / (L.box[3] - L.box[2]);
    return `<g class="lm lm-${name} ${cls}" transform="translate(${N(x)} ${N(y)}) scale(${N(sc)} ${N(-sc)})"${attrs}>${L.g}</g>`;
  };

  // 倒影：同一份图形上下翻过来，压淡
  const placeRefl = (name, x, y, h) => {
    const L = RED[name](), sc = h / (L.box[3] - L.box[2]);
    return `<g class="v-refl" transform="translate(${N(x)} ${N(y)}) scale(${N(sc)} ${N(sc)})">${L.g}</g>`;
  };

  /* ── 胡同：一片灰瓦屋顶，几排前后错落 ───────────── */
  function hutong(P, x0, x1, y, rows, seed) {
    const r = rng(seed);
    for (let k = 0; k < rows; k++) {
      const yy = y - (rows - 1 - k) * 14, s = 0.55 + k * 0.12;
      let roofs = '', walls = '', ridges = '';
      for (let x = x0 + r() * 20; x < x1; x += (34 + r() * 26) * s) {
        const w = (22 + r() * 16) * s, h = (6 + r() * 3) * s;
        walls += `M${N(x - w)} ${N(yy)}V${N(yy - h * 0.8)}H${N(x + w)}V${N(yy)}Z`;
        roofs += `M${N(x - w - 2)} ${N(yy - h * 0.8)}L${N(x - w * 0.55)} ${N(yy - h * 1.9)}H${N(x + w * 0.55)}L${N(x + w + 2)} ${N(yy - h * 0.8)}Z`;
        ridges += `M${N(x - w * 0.55)} ${N(yy - h * 1.9)}H${N(x + w * 0.55)}`;
      }
      P.p(walls, `ht-wall${k < rows - 1 ? ' far' : ''}`); P.p(roofs, `ht-roof${k < rows - 1 ? ' far' : ''}`); P.p(ridges, 'ht-ridge');
    }
  }

  /* ── 红色足迹小长卷 ─────────────────────────────
     自西向东：香山（双清别墅、满山红叶）→ 永定河上的卢沟桥与宛平城 → 胡同里的李大钊故居 → 北大红楼 → 党史展览馆。
     一条朱红虚线把五处串起来，像一张行程单上的站点。 */
  const RW = 2000, RH = 600, GY = 452;
  function redScroll() {
    const P = Pen();
    P.p(`M0 0H${RW}V${RH}H0Z`, 'v-sky');
    // 远山：西山一线，城区上空只剩淡淡的山影
    H.mountain(P, { x: 260, base: GY - 20, h: 250, w: 330, seed: 7, grad: 'g-mtnFar', moss: false, cls: 'far1' });
    H.mountain(P, { x: 620, base: GY - 30, h: 170, w: 300, seed: 11, grad: 'g-mtnFar', moss: false, cls: 'far1' });
    H.mountain(P, { x: 1500, base: GY - 40, h: 90, w: 520, seed: 19, grad: 'g-mtnFar', moss: false, cls: 'far1' });
    P.raw(H.mist(700, GY - 40, 520, 40) + H.mist(1500, GY - 50, 640, 46));
    // 香山：满山林木，红叶成片
    const hill = H.domeHill(300, GY + 4, 300, 230, 23), ridge = x => A.yAt(hill, x);
    P.p(H.smooth(hill) + 'Z', 'mtn g-hill', ' fill="url(#g-hill)"');
    P.raw(`<clipPath id="cp-xs"><rect x="-10" y="0" width="640" height="${GY + 4}"/></clipPath><g clip-path="url(#cp-xs)">`);
    F.canopy(P, { x0: 8, x1: 596, top: ridge, base: GY + 2, rows: 13, rFar: 7, rNear: 12, emerge: 0.02, mix: { a: 3.4, y: 1.4, e: 2.4, p: 1.6, g: 1.2 }, seed: 31,
      skip: (x, y) => y > GY - 70 && Math.abs(x - 330) < 120 });
    P.raw('</g>');
    // 双清别墅
    P.raw(place('shuangqing', 320, GY - 6, 96, 'pick', ' data-spot="xiangshan"'));
    F.clump(P, { x0: 190, x1: 250, y: GY + 4, h: 42, mix: { a: 3, e: 1 }, seed: 33, rows: 2 });
    F.clump(P, { x0: 430, x1: 520, y: GY + 4, h: 44, mix: { a: 2, y: 1, e: 1 }, seed: 35, rows: 2, under: true });
    // 永定河、卢沟桥、宛平城
    P.p(`M540 ${GY + 6}C700 ${GY + 2} 900 ${GY + 4} 1080 ${GY + 8}L1110 ${RH}H520Z`, 'v-river');
    P.raw(H.waves(560, 1080, GY + 12, RH - 10, 5, 'wave'));
    P.raw(placeRefl('lugou', 800, GY + 12, 54));
    P.raw(place('lugou', 800, GY + 10, 54, 'pick', ' data-spot="lugou"'));
    H.bank(P, 480, 560, GY + 2, GY + 30, 3); H.reeds(P, 500, GY + 28, 8, 4);
    H.bank(P, 1040, 1180, GY, GY + 34, 5);
    P.raw(place('wanping', 1110, GY + 4, 96, 'pick', ' data-spot="lugou"'));
    H.willow(P, 1188, GY + 10, 96, 41);
    // 城区：地面、胡同屋顶、槐树
    P.p(`M1150 ${GY - 4}H${RW}V${RH}H1150Z`, 'v-ground');
    hutong(P, 1180, RW, GY - 40, 3, 51);
    F.clump(P, { x0: 1190, x1: 1250, y: GY - 6, h: 60, mix: { g: 2, e: 1 }, seed: 53, rows: 1 });
    P.raw(place('siheyuan', 1340, GY + 8, 92, 'pick', ' data-spot="lidazhao"'));
    F.clump(P, { x0: 1420, x1: 1470, y: GY - 4, h: 64, mix: { g: 2, y: 1 }, seed: 55, rows: 1 });
    P.raw(place('honglou', 1630, GY + 6, 132, 'pick', ' data-spot="honglou"'));
    F.clump(P, { x0: 1500, x1: 1760, y: GY + 18, h: 40, mix: { g: 2, e: 1.4, y: 0.6 }, seed: 57, rows: 1, dense: 0.5 });
    P.raw(place('dangshi', 1880, GY + 2, 112, 'pick', ' data-spot="dangshi"'));
    F.clump(P, { x0: 1770, x1: 1800, y: GY + 4, h: 70, mix: { e: 1 }, seed: 59, rows: 1 });
    F.clump(P, { x0: 1960, x1: 2000, y: GY + 4, h: 66, mix: { e: 1, g: 1 }, seed: 61, rows: 1 });
    // 近处：一条小路贯穿全卷，朱红虚线就走在路上
    P.p(`M0 ${GY + 70}C300 ${GY + 52} 520 ${GY + 96} 760 ${GY + 84}C1000 ${GY + 72} 1200 ${GY + 60} 1420 ${GY + 74}C1640 ${GY + 88} 1840 ${GY + 66} ${RW} ${GY + 70}`, 'v-road');
    P.raw(`<path class="v-route" pathLength="1" d="M60 ${GY + 66}C300 ${GY + 50} 520 ${GY + 92} 760 ${GY + 80}C1000 ${GY + 68} 1200 ${GY + 56} 1420 ${GY + 70}C1640 ${GY + 84} 1840 ${GY + 62} 1960 ${GY + 66}"/>`);
    // 近景：左右两角的石与松
    H.rock(P, 40, RH - 6, 100, 46, 71); H.reeds(P, 70, RH - 8, 10, 72);
    F.bigPine(P, 1990, RH + 20, 300, 73, -1, { limbs: [[0.45, -1, 0.42, 0.02], [0.62, 1, 0.16, -0.02], [0.76, -1, 0.3, -0.03], [0.9, -1, 0.18, -0.03]] });
    // 天空：祥云、雁
    P.raw(H.xiangyun(980, 120, 1.1) + H.xiangyun(1700, 90, 0.9, -1));
    P.raw(H.birds(560, 150, 5, 1, 9));
    const spots = [
      { id: 'xiangshan', x: 320, y: GY + 58, n: 1, name: '香山革命纪念地', area: '海淀区', note: '1949 年中共中央在此指挥渡江战役、筹建新中国。', fit: ['主题党日', '现场教学'] },
      { id: 'lugou', x: 800, y: GY + 82, n: 2, name: '卢沟桥 · 抗战纪念馆', area: '丰台区', note: '全民族抗战爆发地，就在公司所在的丰台。', fit: ['现场教学', '爱国主义教育'] },
      { id: 'lidazhao', x: 1340, y: GY + 60, n: 3, name: '李大钊故居', area: '西城区', note: '李大钊在北京居住时间最长的寓所。', fit: ['现场教学', '党史学习'] },
      { id: 'honglou', x: 1630, y: GY + 82, n: 4, name: '北大红楼', area: '东城区', note: '中国共产党早期北京革命活动旧址。', fit: ['主题党日', '现场教学'] },
      { id: 'dangshi', x: 1880, y: GY + 68, n: 5, name: '中国共产党历史展览馆', area: '朝阳区', note: '全面展示党的百年奋斗历程。', fit: ['党史学习', '主题党日'] },
    ];
    return { W: RW, H: RH, svg: P.s(), spots };
  }

  /* ── 团建：百川汇流（京郊山野） ─────────────────
     几道山溪从不同的山谷流下来，汇成一条河，河上一叶晟帆——"来自不同岗位，拧成一股绳"。
     山野里散着几处活动：长城登山、古道徒步、梯田农事、草地拓展、天上沙燕风筝、湖边厅堂。 */
  const TW = 2000, TH = 600;
  function figure(x, y, s, c, pose = 0) {
    // 点景人物：小到只剩身形，颜色区分队伍
    const h = 14 * s, legs = pose ? `M${N(x - 1.2 * s)} ${N(y - 5 * s)}L${N(x - 2.6 * s)} ${N(y)}M${N(x + 1.2 * s)} ${N(y - 5 * s)}L${N(x + 2.4 * s)} ${N(y)}` : `M${N(x - 1 * s)} ${N(y - 5 * s)}V${N(y)}M${N(x + 1 * s)} ${N(y - 5 * s)}V${N(y)}`;
    return `<path class="fg-leg" d="${legs}"/><path class="fg-body fg-${c}" d="M${N(x - 2.4 * s)} ${N(y - 5 * s)}Q${N(x - 2.8 * s)} ${N(y - 10.6 * s)} ${N(x)} ${N(y - 11 * s)}Q${N(x + 2.8 * s)} ${N(y - 10.6 * s)} ${N(x + 2.4 * s)} ${N(y - 5 * s)}Z"/><circle class="fg-head" cx="${N(x)}" cy="${N(y - h + 1.2 * s)}" r="${N(1.9 * s)}"/>`;
  }
  function group(pts, s, colors, seed) {
    const r = rng(seed); let g = '';
    pts.forEach(([x, y], i) => { g += figure(x, y, s * (0.92 + r() * 0.16), colors[i % colors.length], i % 2); });
    return `<g class="figs">${g}</g>`;
  }
  // 沙燕风筝：北京的传统风筝，黑白二色画燕身，翅尖描彩
  function kite(x, y, s) {
    return `<g class="kite" transform="translate(${N(x)} ${N(y)}) scale(${N(s)})">
      <path class="kt-tail" d="M-6 14C-12 30-4 44-16 60M6 14C12 30 4 44 16 60"/>
      <path class="kt-wing" d="M0-6C-10-14-30-14-42-6C-34-2-26 2-14 4C-8 5-3 4 0 2C3 4 8 5 14 4C26 2 34-2 42-6C30-14 10-14 0-6Z"/>
      <path class="kt-body" d="M0-14C-5-10-6-2-4 6L-7 16L0 12L7 16L4 6C6-2 5-10 0-14Z"/>
      <path class="kt-pat" d="M-34-7C-28-10-20-10-14-6M34-7C28-10 20-10 14-6M-2-8h4M-3-3h6"/>
      <circle class="kt-eye" cx="-2" cy="-9" r="1.1"/><circle class="kt-eye" cx="2" cy="-9" r="1.1"/>
    </g>`;
  }
  function teamScroll() {
    const P = Pen(), WY2 = 470;
    P.p(`M0 0H${TW}V${TH}H0Z`, 'v-sky');
    // 远近几重山，长城沿最高的山脊
    const far = H.mountain(P, { x: 420, base: 380, h: 260, w: 460, seed: 3, grad: 'g-mtnFar', moss: false, cls: 'far1' });
    H.mountain(P, { x: 1180, base: 390, h: 200, w: 420, seed: 5, grad: 'g-mtnFar', moss: false, cls: 'far1' });
    H.mountain(P, { x: 1800, base: 392, h: 170, w: 360, seed: 9, grad: 'g-mtnFar', moss: false, cls: 'far1' });
    H.greatWall(P, far, 0, 880);
    P.raw(H.mist(600, 380, 600, 50) + H.mist(1500, 392, 640, 46));
    const m1 = H.mountain(P, { x: 260, base: WY2 - 4, h: 300, w: 300, seed: 41, grad: 'g-mtnNear', trees: 16 });
    H.mountain(P, { x: 760, base: WY2 - 8, h: 210, w: 280, seed: 43, grad: 'g-mtnMid', trees: 12 });
    H.mountain(P, { x: 1240, base: WY2 - 6, h: 180, w: 260, seed: 47, grad: 'g-mtnMid', trees: 10 });
    H.mountain(P, { x: 1700, base: WY2 - 10, h: 140, w: 300, seed: 49, grad: 'g-mtnNear', trees: 10 });
    H.greatWall(P, m1, 60, 470);
    // 长城上登山的队伍
    const wallPts = m1.filter(([x]) => x > 210 && x < 330).filter((_, i) => i % 1 === 0).slice(0, 6).map(([x, y]) => [x, y - 5]);
    P.raw(`<g class="spot" data-spot="wall">${group(wallPts, 0.9, ['r', 'b', 'r', 'o', 'b', 'r'], 1)}</g>`);
    // 山溪：从各山谷流下，在山脚汇成一条河
    const streams = [
      [[150, 300], [180, 360], [230, 410], [300, 446], [420, WY2 + 6]],
      [[520, 330], [540, 380], [560, 420], [600, 452], [700, WY2 + 10]],
      [[980, 330], [960, 380], [930, 420], [880, 452], [800, WY2 + 12]],
      [[1110, 350], [1100, 400], [1080, 440], [1040, 460], [960, WY2 + 14]],
      [[1520, 360], [1480, 410], [1420, 448], [1340, WY2 + 4], [1200, WY2 + 16]],
    ];
    let sd = '';
    for (const s of streams) sd += H.smooth(s);
    P.p(sd, 'v-stream-b'); P.raw(`<path class="v-stream" pathLength="1" d="${sd}"/>`);
    // 汇流的大河，向右流去
    P.p(`M380 ${WY2 + 4}C700 ${WY2 - 4} 1000 ${WY2 + 8} 1300 ${WY2 + 4}C1600 ${WY2} 1800 ${WY2 + 6} ${TW} ${WY2 + 2}V${TH}H380C340 ${TH - 60} 360 ${WY2 + 40} 380 ${WY2 + 4}Z`, 'v-river');
    P.raw(H.waves(420, TW, WY2 + 14, TH - 10, 6, 'wave'));
    P.raw(`<path class="v-flow" pathLength="1" d="M420 ${WY2 + 30}C800 ${WY2 + 20} 1200 ${WY2 + 40} 1600 ${WY2 + 30}C1800 ${WY2 + 26} 1900 ${WY2 + 30} 1990 ${WY2 + 28}"/>`);
    // 左岸：古道（石板小路）与徒步的人
    P.p(`M0 ${WY2 + 8}C120 ${WY2 - 10} 220 ${WY2 + 30} 380 ${WY2 + 6}V${TH}H0Z`, 'v-ground');
    P.raw(`<path class="v-trail" d="M10 ${TH - 30}C80 ${TH - 60} 140 ${WY2 + 70} 230 ${WY2 + 50}C300 ${WY2 + 36} 340 ${WY2 + 24} 380 ${WY2 + 14}"/>`);
    P.raw(`<g class="spot" data-spot="trail">${group([[96, TH - 54], [116, TH - 64], [138, WY2 + 82], [158, WY2 + 72], [180, WY2 + 64]], 1.25, ['b', 'o', 'b', 'r', 'g'], 2)}</g>`);
    // 梯田与农事
    let terr = '';
    for (let k = 0; k < 6; k++) { const y = 380 + k * 14, x0 = 560 + k * 10, x1 = 760 - k * 6; terr += `M${x0} ${y}C${x0 + 60} ${y - 6} ${x1 - 60} ${y - 6} ${x1} ${y}C${x1 - 40} ${y + 8} ${x0 + 40} ${y + 8} ${x0} ${y}Z`; }
    P.p(terr, 'v-terrace'); P.p(terr, 'v-terrace-l');
    P.raw(`<g class="spot" data-spot="farm">${group([[610, 392], [660, 404], [700, 418], [640, 432]], 0.95, ['o', 'g', 'o', 'b'], 3)}</g>`);
    // 草地：拓展（拔河）与沙燕风筝
    P.p(`M820 ${WY2 - 2}C900 ${WY2 - 30} 1060 ${WY2 - 34} 1160 ${WY2 - 4}Z`, 'v-meadow');
    P.raw(`<g class="spot" data-spot="meadow"><path class="v-rope" d="M906 ${WY2 - 12}L1072 ${WY2 - 14}"/>${group([[912, WY2 - 10], [928, WY2 - 11], [944, WY2 - 12], [960, WY2 - 13], [1018, WY2 - 14], [1034, WY2 - 14], [1050, WY2 - 13], [1066, WY2 - 12]], 1, ['r', 'r', 'r', 'r', 'b', 'b', 'b', 'b'], 4)}</g>`);
    P.raw(`<g class="spot" data-spot="kite"><path class="kt-line" d="M1120 ${WY2 - 14}Q1150 300 1180 190"/>${group([[1120, WY2 - 12]], 1, ['o'], 5)}${kite(1182, 180, 0.9)}</g>`);
    // 湖边厅堂：会议、读书、年会都可以在这里
    P.raw(`<g class="spot" data-spot="hall">${at(1500, WY2 - 2, 0.62, H.hall(84, 24, 54, 22, 'G', false))}${group([[1462, WY2 + 2], [1478, WY2 + 2], [1530, WY2 + 2]], 1, ['b', 'r', 'g'], 6)}</g>`);
    F.clump(P, { x0: 1380, x1: 1440, y: WY2, h: 54, mix: { g: 2, y: 1 }, seed: 81, rows: 2 });
    F.clump(P, { x0: 1570, x1: 1640, y: WY2, h: 50, mix: { e: 2, a: 1 }, seed: 83, rows: 2 });
    // 河上一叶晟帆：众流归一
    P.raw(`<g class="spot" data-spot="sail" transform="translate(1800 ${WY2 + 50}) scale(.62)">${sailboat()}</g>`);
    // 近景
    H.rock(P, 1960, TH - 4, 90, 40, 91); H.reeds(P, 1900, TH - 6, 10, 92);
    P.raw(H.xiangyun(900, 110, 1) + H.xiangyun(1600, 80, 0.8, -1));
    P.raw(H.geese(1500, 160, 7, 1, 7));
    const spots = [
      { id: 'wall', x: 290, y: 250, name: '长城 · 登山', gov: ['dengshan'], ent: ['tubu'] },
      { id: 'trail', x: 150, y: WY2 + 40, name: '京西古道 · 徒步', gov: ['dingxiang'], ent: ['tubu'] },
      { id: 'farm', x: 660, y: 370, name: '京郊农场 · 农事', gov: ['laodong'], ent: [] },
      { id: 'meadow', x: 990, y: WY2 - 40, name: '草地 · 拓展', gov: [], ent: ['tuozhan', 'yundong'] },
      { id: 'kite', x: 1182, y: 150, name: '沙燕风筝 · 非遗', gov: ['feiyi'], ent: [] },
      { id: 'hall', x: 1500, y: WY2 - 60, name: '湖边厅堂', gov: ['gonghui', 'dushu'], ent: ['gongchuang', 'nianhui'] },
      { id: 'sail', x: 1800, y: WY2 + 4, name: '标杆企业 · 研学', gov: [], ent: ['yanxue'] },
    ];
    return { W: TW, H: TH, svg: P.s(), spots };
  }
  function sailboat() {
    return `<path class="gb-hull" d="M-62-6H54C61-6 65-10 68-17C64-3 53 7 33 10H-44C-56 8-62 2-62-6Z"/><path class="gb-strake" d="M-59-2H60"/>
      <path class="gb-mast" d="M8-6V-120"/><path class="gb-sail" d="M-28-110C-8-115 16-119 38-117C40-84 40-50 38-20C14-18-10-18-34-20C-36-50-35-82-28-110Z"/>
      <path class="gb-batten" d="M-31-95C-6-99 20-101 39-100M-33-80C-7-83 20-85 39-84M-34-65C-7-67 20-68 39-68M-35-50C-8-51 20-52 39-52M-35-35C-8-35 20-36 38-36"/>
      <rect class="gb-seal" x="0" y="-76" width="17" height="21" rx="1.6"/>`;
  }

  window.VIGNETTE = { defs, redScroll, teamScroll, RED, place, figure, group, kite, sailboat };
})();
