/* 首屏《京华胜景图》：三远分层，所有地标同框
   远：燕山一线，长城顺山脊横贯；中远：西山玉峰塔、颐和园佛香阁（万寿山）、北海白塔（琼华岛），昆明湖从左边一路铺到近处；
   中：故宫宫城（金顶一片、城墙、午门、两角角楼、筒子河）居中，天坛祈年殿在右侧高耸、四周柏林；
   近：昆明湖水面与十七孔桥（带倒影）、右下一块坡岸，中轴南端永定门、国博立在岸上；最前：松枝、松石框景。
   会动的东西单独成块（云、鹤、晟帆、秋天的雁阵），不画进层里。

   compose(layout, px, { season }) 返回：
   - planes：静止的几层，各带纵深 depth（0 最远、1 最近）和所在矩形 rect；上线时每层烘成一张图片。
   - sprites：会动的小块（云、鹤的三个姿势、船、雁阵），各带 rect 和 depth。
   - hits：可点的地标（id、rect、题签位置 slip）。
   四时：SEASON_CSS 换树色、山色、水色；冬天屋面积雪、湖面结冰；夏天湖里加荷叶（画进近景层）。 */
(function () {
  'use strict';
  const J = window.JH, { N, P2, Pen, place, rng, samp, smooth } = J, Nt = J.NATURE, LM = J.LM, X2 = J.LMX2;

  /* ── 四时的设色：烘图和现场都用同一套 ── */
  const lin = (id, stops, x1 = 0, y1 = 0, x2 = 0, y2 = 1) => `<linearGradient id="${id}" x1="${x1}" y1="${y1}" x2="${x2}" y2="${y2}">${stops.map(([o, c, a = 1]) => `<stop offset="${o}" stop-color="${c}"${a < 1 ? ` stop-opacity="${a}"` : ''}/>`).join('')}</linearGradient>`;
  function defs() {
    // 翻转坐标里画的屋面、山：offset 0 在视觉的下沿
    return lin('jh-hazev', [[0, '#efe6d3', 0], [0.3, '#efe6d3', 1], [0.72, '#efe6d3', 1], [1, '#efe6d3', 0]]) +   // 上下都淡出：滚动错层时不露硬边
      lin('jh-snow-yellow', [[0, '#c98a2a'], [0.16, '#e6c584'], [0.3, '#f8f6f1'], [1, '#ffffff']]) +
      lin('jh-snow-gray', [[0, '#5d6462'], [0.16, '#aeb2ae'], [0.3, '#f5f6f4'], [1, '#ffffff']]) +
      lin('jh-snow-blue', [[0, '#1a3a77'], [0.18, '#9bb0d4'], [0.34, '#f3f6fa'], [1, '#ffffff']]) +
      lin('jh-snow-green', [[0, '#2b6a4c'], [0.16, '#a7c7b4'], [0.3, '#f4f8f5'], [1, '#ffffff']]) +
      lin('jh-snow-shan', [[0, '#a59f92'], [0.35, '#c8cbc6'], [0.65, '#eef1f1'], [1, '#ffffff']]) +
      lin('jh-snow-far', [[0, '#d2cfc6'], [0.5, '#e8ebea'], [1, '#f7f8f8']]) +
      lin('jh-snow-mid', [[0, '#b8b2a2'], [0.45, '#dcdfdb'], [1, '#f8f9f8']]) +
      lin('jh-ice', [[0, '#e9eff1'], [1, '#d6e2e6']]) +
      lin('jh-autumn-shan', [[0, '#c99a5a'], [0.35, '#c9a066'], [0.62, '#a19b6a'], [1, '#6f8c86']]) +
      lin('jh-autumn-mid', [[0, '#cfac7c'], [0.4, '#c3a874'], [0.7, '#9fa37c'], [1, '#7a958c']]) +
      lin('jh-spring-shan', [[0, '#c7a56f'], [0.3, '#a9b07a'], [0.55, '#78ad86'], [0.8, '#5a9890'], [1, '#41758f']]);
  }
  const SEASON_CSS = {
    spring: `.s-spring .jh .c-tree-l{fill:#f2c3cb}.s-spring .jh .c-tree{fill:#9fc57f}.s-spring .jh .c-tree-d{fill:#6f9e5d}
.s-spring .jh .c-willow{fill:#b9dc79}.s-spring .jh .c-willow-l{fill:#dff0a6}.s-spring .jh .c-willow-d{fill:#8db35c}.s-spring .jh .l-wash-g{stroke:#cfe3a4}
.s-spring .jh .g-shan{fill:url(#jh-spring-shan)}`,
    summer: '',
    autumn: `.s-autumn .jh .c-tree-l{fill:#e8b04a}.s-autumn .jh .c-tree{fill:#cd7a3b}.s-autumn .jh .c-tree-d{fill:#8f5a33}
.s-autumn .jh .c-willow{fill:#d1bf62}.s-autumn .jh .c-willow-l{fill:#e6d68a}.s-autumn .jh .c-willow-d{fill:#a8933f}.s-autumn .jh .l-wash-g{stroke:#e2cf8a}
.s-autumn .jh .g-shan{fill:url(#jh-autumn-shan)}.s-autumn .jh .g-shan-mid{fill:url(#jh-autumn-mid)}`,
    winter: `.s-winter .jh .c-tree-l{fill:#dedbd2}.s-winter .jh .c-tree{fill:#a89e8e}.s-winter .jh .c-tree-d{fill:#7c7264}
.s-winter .jh .c-willow{fill:#cbc4aa}.s-winter .jh .c-willow-l{fill:#e6e1d0}.s-winter .jh .c-willow-d{fill:#a9a089}.s-winter .jh .l-wash-g{stroke:#e9e6dc}
.s-winter .jh .c-pine{fill:#456456}.s-winter .jh .c-pine-d{fill:#2f4a3e}.s-winter .jh .c-bai{fill:#3f5a4c}.s-winter .jh .c-bai-l{fill:#5d7568}.s-winter .jh .c-bai-d{fill:#2c4337}
.s-winter .jh .lw{stroke:#fff;stroke-opacity:.95;stroke-width:calc(var(--w2,.95px)*1.8)}
.s-winter .jh .g-yellow{fill:url(#jh-snow-yellow)}.s-winter .jh .g-gray{fill:url(#jh-snow-gray)}.s-winter .jh .g-blue{fill:url(#jh-snow-blue)}.s-winter .jh .g-green{fill:url(#jh-snow-green)}
.s-winter .jh .g-shan{fill:url(#jh-snow-shan)}.s-winter .jh .g-shan-far{fill:url(#jh-snow-far)}.s-winter .jh .g-shan-mid{fill:url(#jh-snow-mid)}
.s-winter .jh .g-water,.s-winter .jh .g-lake{fill:url(#jh-ice)}.s-winter .jh .c-water{fill:#e6eef0}.s-winter .jh .l-wave{stroke:#9fb6bf;stroke-opacity:.5}.s-winter .jh .l-wave2{stroke:#fff;stroke-opacity:1}
.s-winter .jh .c-bank-t{fill:#eceee9}.s-winter .jh .c-tree-l,.s-winter .jh .c-tree{stroke-opacity:.9}`,
  };

  /* ── 荷叶（夏天）：一片片圆叶，叶脉放射，间几枝粉荷 ── */
  function lotus(o = {}) {
    const P = Pen(), r = rng(o.seed || 31), w = o.w || 160, n = o.n || 9;
    const pads = Array.from({ length: n }, () => [(r() - 0.5) * w, r() * w * 0.12, w * (0.07 + r() * 0.05)]).sort((a, b) => b[1] - a[1]);
    for (const [x, y, R] of pads) {
      const ry = R * 0.36, a0 = 0.3 + r() * 0.5;
      P.p(`M${P2(x, y)}L${P2(x + Math.cos(a0) * R, y + Math.sin(a0) * ry)}A${N(R)} ${N(ry)} 0 1 1 ${P2(x + Math.cos(a0 + 0.35) * R, y + Math.sin(a0 + 0.35) * ry)}Z`, 'c-lotus l3');
      let v = ''; for (let k = 0; k < 6; k++) { const a = a0 + 0.5 + k * 0.95; v += `M${P2(x, y)}L${P2(x + Math.cos(a) * R * 0.85, y + Math.sin(a) * ry * 0.85)}`; }
      P.p(v, 'l-lotus');
    }
    for (let k = 0; k < (o.flowers ?? 3); k++) {
      const [x, y, R] = pads[Math.floor(r() * pads.length)], fx = x + R * 0.3, fy = y + R * 0.2, h = R * 1.4, s = R * 0.42;
      P.p(`M${P2(fx, fy)}Q${P2(fx + s * 0.3, fy + h * 0.5)} ${P2(fx, fy + h)}`, 'l-stem');
      const cy = fy + h;
      P.p(`M${P2(fx, cy)}C${P2(fx - s * 1.1, cy + s * 0.4)} ${P2(fx - s * 0.7, cy + s * 1.5)} ${P2(fx - s * 0.1, cy + s * 1.9)}C${P2(fx, cy + s * 1.2)} ${P2(fx, cy + s * 0.6)} ${P2(fx, cy)}ZM${P2(fx, cy)}C${P2(fx + s * 1.1, cy + s * 0.4)} ${P2(fx + s * 0.7, cy + s * 1.5)} ${P2(fx + s * 0.1, cy + s * 1.9)}C${P2(fx, cy + s * 1.2)} ${P2(fx, cy + s * 0.6)} ${P2(fx, cy)}Z`, 'c-lotus-f l3');
      P.p(`M${P2(fx - s * 0.35, cy + s * 0.2)}C${P2(fx - s * 0.4, cy + s * 1.2)} ${P2(fx + s * 0.4, cy + s * 1.2)} ${P2(fx + s * 0.35, cy + s * 0.2)}C${P2(fx + s * 0.2, cy + s * 0.9)} ${P2(fx - s * 0.2, cy + s * 0.9)} ${P2(fx - s * 0.35, cy + s * 0.2)}Z`, 'c-lotus-f2 l3');
    }
    return { g: P.s(), box: [-w * 0.62, w * 0.62, -w * 0.08, w * 0.26] };
  }

  /* ── 雁阵（秋天）：一行人字，远近大小不一 ── */
  function geese(n = 7, seed = 3) {
    const P = Pen(), r = rng(seed); let d = '';
    for (let i = 0; i < n; i++) { const k = i - (n - 1) / 2, x = k * 22 + i * 6, y = -Math.abs(k) * 9 + (r() - 0.5) * 3, s = 4.6 + r() * 1.4; d += `M${P2(x - s, y + s * 0.5)}Q${P2(x - s * 0.4, y + s * 0.1)} ${P2(x, y - s * 0.12)}Q${P2(x + s * 0.4, y + s * 0.1)} ${P2(x + s, y + s * 0.5)}`; }
    P.p(d, 'l-goose');
    return { g: P.s(), box: [-90, 110, -40, 6] };
  }

  /* ── 构图 ── */
  function compose(layout = 'desk', px = 1, opt = {}) {
    const D = layout === 'desk', W = D ? 1600 : 400, H = D ? 1000 : 1120, season = opt.season || 'summer';
    const L = {}, ext = {}, sprites = [], hits = [];
    const grow = (id, x0, y0, x1, y1) => { const e = ext[id] || (ext[id] = [Infinity, Infinity, -Infinity, -Infinity]); e[0] = Math.min(e[0], x0); e[1] = Math.min(e[1], y0); e[2] = Math.max(e[2], x1); e[3] = Math.max(e[3], y1); };
    const add = (id, s, r) => { L[id] = (L[id] || '') + s; if (r) grow(id, ...r); };
    // 按底边放：bottom 是画面里元素底边的 y，h 是画面高度；返回比例和所占矩形
    const rectOf = (el, cx, bottom, h) => { const s = h / (el.box[3] - el.box[2]); return [s, [cx + el.box[0] * s, bottom - h, cx + el.box[1] * s, bottom]]; };
    const put = (id, el, cx, bottom, h, cls = '', hit) => {
      const [s, r] = rectOf(el, cx, bottom, h);
      add(id, place(el, cx, bottom + el.box[2] * s, h, px, cls), r);
      if (hit) hits.push({ id: hit, rect: r, plane: id });
      return s;
    };
    const sprite = (o, el, cx, bottom, h, cls = '') => { const [s, r] = rectOf(el, cx, bottom, h); sprites.push({ ...o, rect: r, svg: place(el, cx, bottom + el.box[2] * s, h, px, cls) }); return s; };
    const reflect = (id, el, cx, y0, s, a = 0.26) => add(id, `<g class="jh reflect" opacity="${a}" style="${J.lineStyle(s * px)}" transform="translate(${N(cx)} ${N(y0)}) scale(${N(s)} ${N(s)})">${el.g}</g>`);
    const haze = (id, y0, y1, a) => add(id, `<rect x="-20" y="${N(y0)}" width="${W + 40}" height="${N(y1 - y0)}" fill="url(#jh-hazev)" opacity="${a}"/>`, [0, y0, W, y1]);
    const mist = (id, cx, bottom, w, h) => put(id, Nt.mistBlob({ w, h }), cx, bottom, h);
    const cloud = (o, cx, bottom, h) => sprite({ kind: 'cloud', ...o }, Nt.cloud2(o.c), cx, bottom, h);
    const crane = (cx, bottom, h, dir, phase, depth) => { const poses = ['up', 'flat', 'down'].map(pose => { const el = Nt.crane({ pose, dir }), [s] = rectOf(el, cx, bottom, h); return place(el, cx, bottom + el.box[2] * s, h, px); }); const [, r] = rectOf(Nt.crane({ dir }), cx, bottom, h); sprites.push({ kind: 'crane', id: `crane${sprites.length}`, rect: r, poses, phase, dir, depth }); };
    // 长城可点的一段：取山脊上 x 在 [a, b] 之间的那段城墙
    const wallHit = (pts, cx, bottom, h, box, a, b) => { const s = h / (box[3] - box[2]); const xs = pts.map(([x, y]) => [cx + x * s, bottom + box[2] * s - y * s]).filter(([x]) => x >= a && x <= b); const ys = xs.map(p => p[1]); hits.push({ id: 'changcheng', rect: [a, Math.min(...ys) - 22, b, Math.max(...ys) + 14], plane: 'far' }); };

    if (D) {
      /* ── 电脑：1600 × 1000 ── */
      const yan = Nt.range({ w: 1760, h: 300, seed: 7, fill: 'g-shan-far', peaks: [[-0.95, 0.82, 0.07], [-0.78, 1, 0.07], [-0.58, 0.74, 0.08], [-0.34, 0.42, 0.1], [-0.04, 0.3, 0.12], [0.3, 0.44, 0.1], [0.54, 0.76, 0.08], [0.74, 0.98, 0.07], [0.93, 0.84, 0.07]] });
      const gw = X2.greatwall(yan.pts.slice(4, -4), { h: 7, towers: [0.05, 0.16, 0.27, 0.4, 0.55, 0.68, 0.8, 0.92], huts: [0.16, 0.8] });
      put('far', { g: yan.g + gw.g, box: yan.box }, 800, 566, 312, 'far');
      wallHit(yan.pts.slice(4, -4), 800, 566, 312, yan.box, 1180, 1520);
      haze('far', 180, 640, 0.27);
      mist('far', 800, 640, 1900, 130); mist('far', 260, 600, 700, 80); mist('far', 1340, 610, 760, 90);
      cloud({ c: { w: 380, h: 44, seed: 3 }, depth: 0.25, side: -1 }, 300, 556, 64);
      cloud({ c: { w: 440, h: 46, seed: 5, dir: -1 }, depth: 0.25, side: 1 }, 1320, 566, 66);
      // 中远：西山、玉峰塔、万寿山与佛香阁、北海琼华岛与白塔；昆明湖从左边铺下来
      put('midfar', Nt.lake({ x0: -20, x1: 780, h: 180, seed: 3 }), 0, 866, 180);
      put('midfar', Nt.range({ w: 720, h: 190, seed: 11, fill: 'g-shan-mid', peaks: [[-0.84, 0.86, 0.12], [-0.5, 1, 0.12], [-0.14, 0.66, 0.14], [0.26, 0.5, 0.14], [0.7, 0.34, 0.16]] }), 230, 664, 196, 'far');
      put('midfar', Nt.hill({ w: 140, h: 44, seed: 3, density: 0.6 }), 456, 628, 46);
      put('midfar', LM.yufeng(), 456, 616, 62, 'far');
      put('midfar', Nt.hill({ w: 470, h: 120, seed: 9, fill: 'g-shan-mid' }), 300, 700, 128);
      put('midfar', LM.foxiang(), 300, 708, 212, '', 'yiheyuan');
      put('midfar', Nt.grove({ w: 150, h: 60, seed: 6, kind: 'mixed', trunks: false }), 180, 690, 56); put('midfar', Nt.grove({ w: 150, h: 60, seed: 16, kind: 'mixed', trunks: false }), 420, 690, 56);
      put('midfar', Nt.bank({ w: 560, h: 18, seed: 2, end: 0.2 }), 300, 718, 18);
      put('midfar', Nt.grove({ w: 380, h: 56, seed: 5, kind: 'mixed', density: 1.1, trunks: false }), 300, 712, 52);
      put('midfar', Nt.hill({ w: 280, h: 80, seed: 13, fill: 'g-shan-mid', density: 0.9 }), 586, 712, 86);
      put('midfar', LM.baita(), 586, 652, 106, '', 'beihai');
      put('midfar', Nt.grove({ w: 200, h: 40, seed: 8, kind: 'mixed', trunks: false }), 586, 716, 40);
      put('midfar', Nt.waves({ w: 560, h: 120, s: 0.75, seed: 9 }), 270, 860, 120);
      if (season === 'summer') put('midfar', lotus({ w: 150, n: 8, seed: 5, flowers: 2 }), 470, 792, 22);
      haze('midfar', 380, 720, 0.1);
      cloud({ c: { w: 300, h: 34, seed: 9 }, depth: 0.45, side: -1 }, 150, 716, 50);
      cloud({ c: { w: 360, h: 36, seed: 11, dir: -1 }, depth: 0.45, side: 1 }, 1030, 644, 52);
      cloud({ c: { w: 280, h: 30, seed: 13 }, depth: 0.45, side: 0 }, 700, 662, 44);
      // 中：宫城、天坛
      put('mid', LM.gugong(), 830, 822, 206, '', 'gugong');
      put('mid', Nt.bank({ w: 540, h: 30, seed: 4, end: 0.18 }), 1392, 836, 30);
      put('mid', Nt.grove({ w: 430, h: 110, seed: 12, kind: 'bai', density: 1.1, trunks: false }), 1412, 812, 112);
      put('mid', LM.qinian(), 1394, 826, 232, '', 'tiantan');
      put('mid', Nt.cypress({ h: 200, seed: 9 }), 1590, 838, 112);
      mist('mid', 820, 900, 1100, 46); mist('mid', 1360, 884, 700, 40);
      // 鹤群：过宫城上空
      for (const [x, y, s, dir, ph] of [[690, 590, 34, 1, 0], [752, 616, 28, 1, 0.33], [820, 576, 26, 1, 0.66], [628, 640, 26, 1, 0.5]]) crane(x, y, s, dir, ph, 0.6);
      // 近：昆明湖（十七孔桥）、右下坡岸（永定门、国博、树）
      put('near', Nt.lake({ x0: -20, x1: 1620, h: 156, seed: 5, cls: 'g-water' }), 0, 1006, 156);
      put('near', Nt.waves({ w: 760, h: 150, s: 1.1 }), 300, 1004, 150);
      const br = LM.shiqikong(), bs = put('near', br, 330, 912, 48, '', 'shiqikong');
      reflect('near', br, 330, 912 + br.box[2] * bs, bs);
      if (season === 'summer') { put('near', lotus({ w: 220, n: 12, seed: 7 }), 150, 968, 34); put('near', lotus({ w: 160, n: 8, seed: 9, flowers: 2 }), 640, 994, 26); }
      sprite({ kind: 'boat', id: 'boat', depth: 0.8, range: [470, 640] }, Nt.boat({}), 548, 984, 66);
      put('near', Nt.bank({ w: 1120, h: 92, seed: 6, end: 0.12 }), 1130, 1006, 92);
      add('near', `<path d="M690 1004C760 990 802 974 812 946L822 946C826 972 858 990 930 1004Z" fill="#e7dcc3" stroke="#8b7a5c" stroke-width="${N(0.7 / px)}"/><path d="M846 986C930 978 1090 972 1250 962L1252 966C1100 978 950 988 872 996Z" fill="#e7dcc3" stroke="#8b7a5c" stroke-width="${N(0.6 / px)}"/>`);
      put('near', Nt.grove({ w: 300, h: 44, seed: 14, kind: 'mixed', trunks: false, density: 0.9 }), 1010, 922, 40);
      for (const [x, b, h, sd] of [[948, 946, 84, 11], [1078, 944, 74, 23], [1450, 960, 92, 31], [1512, 966, 100, 7]]) put('near', Nt.cypress({ h: 200, seed: sd }), x, b, h);
      const team = (n, flag) => { const P = Pen(); const cs = ['b', 'r', 'o', 'g', 'w', 'b', 'r']; for (let i = 0; i < n; i++) { const Lp = Nt.person({ h: 40, c: cs[i % cs.length], pose: i === 0 && flag ? 'flag' : 'walk', dir: -1, hair: i % 3 === 1 ? 'long' : 'short', bag: i % 2 === 0 }); P.g(Lp.g, ` transform="translate(${N(i * 22)} ${N((i % 2) * 3)})"`); } return { g: P.s(), box: [-16, n * 22, 0, flag ? 52 : 42] }; };
      put('near', team(6, true), 836, 980, 15);
      put('near', team(4, true), 1150, 972, 13);
      put('near', LM.yongding(), 816, 944, 120, '', 'yongding');
      put('near', LM.guobo(), 1246, 958, 76, '', 'guobo');
      put('near', Nt.willow({ h: 200, seed: 7, density: 0.5 }), 660, 960, 128);
      // 最前：松枝（左上）、松石（右下），分两块以便裁小
      put('bough', Nt.pineBough({ len: 420, seed: 3, droop: 0.12 }), -110, 250, 150);
      put('pine', Nt.rock({ w: 200, h: 120, seed: 21 }), 1556, 1012, 80);
      put('pine', Nt.pine({ h: 260, seed: 4, lean: -1 }), 1612, 1012, 220);
      sprite({ kind: 'geese', id: 'geese', depth: 0.2, only: 'autumn' }, geese(7, 3), 1180, 214, 34, 'far');
    } else {
      /* ── 手机：400 × 1120（一屏到一屏半） ── */
      const yan = Nt.range({ w: 560, h: 150, seed: 7, fill: 'g-shan-far', peaks: [[-0.9, 0.86, 0.12], [-0.56, 1, 0.12], [-0.2, 0.6, 0.14], [0.16, 0.68, 0.14], [0.5, 1, 0.12], [0.86, 0.86, 0.12]] });
      const gw = X2.greatwall(yan.pts.slice(3, -3), { h: 6, towers: [0.08, 0.26, 0.45, 0.62, 0.82], huts: [0.26] });
      put('far', { g: yan.g + gw.g, box: yan.box }, 200, 494, 152, 'far');
      wallHit(yan.pts.slice(3, -3), 200, 494, 152, yan.box, 228, 392);
      haze('far', 300, 520, 0.18);
      mist('far', 200, 552, 620, 132);
      cloud({ c: { w: 170, h: 26, seed: 3 }, depth: 0.25, side: -1 }, 82, 498, 38);
      cloud({ c: { w: 160, h: 24, seed: 5, dir: -1 }, depth: 0.25, side: 1 }, 330, 508, 36);
      // 颐和园（左）、天坛（右）
      put('midfar', Nt.hill({ w: 250, h: 70, seed: 9, fill: 'g-shan-mid' }), 110, 664, 74);
      put('midfar', LM.foxiang(), 110, 676, 196, '', 'yiheyuan');
      put('midfar', Nt.grove({ w: 90, h: 46, seed: 6, kind: 'mixed', trunks: false }), 44, 650, 44); put('midfar', Nt.grove({ w: 90, h: 46, seed: 16, kind: 'mixed', trunks: false }), 178, 650, 44);
      put('midfar', Nt.bank({ w: 240, h: 12, seed: 2, end: 0.2 }), 110, 680, 12);
      put('midfar', Nt.grove({ w: 210, h: 46, seed: 5, kind: 'mixed', density: 1.3, trunks: false }), 110, 674, 48);
      put('midfar', Nt.grove({ w: 230, h: 56, seed: 12, kind: 'bai', trunks: false }), 302, 668, 58);
      put('midfar', LM.qinian(), 300, 678, 138, '', 'tiantan');
      put('midfar', Nt.bank({ w: 250, h: 14, seed: 4, end: 0.18 }), 300, 690, 14);
      cloud({ c: { w: 160, h: 22, seed: 9 }, depth: 0.45, side: -1 }, 80, 704, 32);
      cloud({ c: { w: 150, h: 22, seed: 11, dir: -1 }, depth: 0.45, side: 1 }, 330, 712, 32);
      // 故宫
      put('mid', LM.gugong({ half: 1000 }), 200, 884, 150, '', 'gugong');
      mist('mid', 200, 930, 520, 34);
      for (const [x, y, s, dir, ph] of [[150, 736, 24, 1, 0], [194, 754, 20, 1, 0.4], [236, 728, 18, 1, 0.75]]) crane(x, y, s, dir, ph, 0.6);
      // 水面、十七孔桥、坡岸上的永定门
      put('near', Nt.lake({ x0: -20, x1: 420, h: 230, seed: 5, cls: 'g-water' }), 0, 1126, 230);
      put('near', Nt.waves({ w: 440, h: 220, s: 0.7, seed: 4 }), 200, 1120, 220);
      const br = LM.shiqikong(), bs = put('near', br, 200, 970, 30, '', 'shiqikong');
      reflect('near', br, 200, 970 + br.box[2] * bs, bs);
      if (season === 'summer') put('near', lotus({ w: 120, n: 8, seed: 7 }), 168, 1034, 18);
      put('near', Nt.bank({ w: 300, h: 80, seed: 6, end: 0.14 }), 300, 1124, 80);
      put('near', Nt.cypress({ h: 200, seed: 11 }), 364, 1070, 62);
      put('near', LM.yongding(), 292, 1074, 88, '', 'yongding');
      sprite({ kind: 'boat', id: 'boat', depth: 0.8, range: [96, 170] }, Nt.boat({}), 110, 1056, 56);
      put('pine', Nt.rock({ w: 160, h: 110, seed: 21 }), 20, 1124, 64);
      put('pine', Nt.pine({ h: 260, seed: 4 }), -6, 1124, 190);
      sprite({ kind: 'geese', id: 'geese', depth: 0.2, only: 'autumn' }, geese(5, 3), 290, 226, 22, 'far');
    }
    // 静止层：纵深、所在矩形（裁到画面内，四周留一点边）
    const DEPTH = { far: 0.12, midfar: 0.34, mid: 0.56, near: 0.8, bough: 1, pine: 1 };
    const planes = Object.keys(DEPTH).filter(id => L[id]).map(id => {
      const e = ext[id], pad = 6;
      const rect = [Math.max(0, Math.floor(e[0] - pad)), Math.max(0, Math.floor(e[1] - pad)), Math.min(W, Math.ceil(e[2] + pad)), Math.min(H, Math.ceil(e[3] + pad))];
      return { id, depth: DEPTH[id], rect, svg: L[id] };
    });
    // 题签放在地标旁边的空处（画面坐标，竖排小签的左上角）
    const SLIP = D ? { changcheng: [1530, 232], yiheyuan: [430, 470], beihai: [652, 520], gugong: [1112, 560], tiantan: [1250, 560], shiqikong: [190, 830], yongding: [880, 790], guobo: [1478, 760] }
      : { changcheng: [20, 300], yiheyuan: [196, 470], tiantan: [218, 520], gugong: [12, 690], shiqikong: [300, 880], yongding: [206, 960] };
    for (const h of hits) h.slip = SLIP[h.id] || [h.rect[2] + 6, h.rect[1]];
    return { W, H, season, planes, sprites, hits };
  }

  window.JH.JINGHUA = { compose, defs, SEASON_CSS, lotus, geese };
})();
