/* 会务研学 · 京华会客厅：新建筑，老画法
   同一套墨线和设色画北京的会议场地；天色换成清晨的青白，和首屏的暖纸色、红色教育的朝霞区分开。
   场地外观依据公开资料：
   - 雁栖湖国际会议中心："鸿雁展翼，汉唐飞扬"，大屋顶两端檐角高挑如展翅的鸿雁。
   - 日出东方酒店：外形像一轮升起的太阳，侧看如扇贝。
   - 首钢园：四座 70 米高的双曲线冷却塔，前面是滑雪大跳台（造型取敦煌飞天的飘带，高约 60 米，比冷却塔低），群明湖、三高炉。
   - 国家会议中心：弧形屋顶借鉴传统飞檐，正面一道约 400 米长的通高玻璃幕墙。
   - 中关村、丽泽：现代楼群。
   画面里不画旗帜、徽记、标语和任何文字。 */
(function () {
  'use strict';
  const J = window.JH, { N, P2, Pen, poly, smooth, samp, rng } = J, A = J.ARCH, Nt = J.NATURE;
  const rect = (x, y, w, h) => `M${P2(x, y)}h${N(w)}v${N(h)}h${N(-w)}Z`;

  function defs() {
    const lin = (id, stops, x1 = 0, y1 = 0, x2 = 0, y2 = 1) => `<linearGradient id="${id}" x1="${x1}" y1="${y1}" x2="${x2}" y2="${y2}">${stops.map(([o, c, a = 1]) => `<stop offset="${o}" stop-color="${c}"${a < 1 ? ` stop-opacity="${a}"` : ''}/>`).join('')}</linearGradient>`;
    return '<svg width="0" height="0" style="position:absolute" aria-hidden="true"><defs>' +
      lin('mc-sky', [[0, '#dfe8ea'], [.55, '#eaeeea'], [1, '#f3eee2']]) +
      lin('mc-water', [[0, '#cddcda'], [1, '#e3e9e2']]) +
      lin('mc-glass', [[0, '#6f97a2'], [.55, '#8fb3b8'], [1, '#bcd3d0']]) +
      lin('mc-glass-h', [[0, '#a9c6c6'], [.5, '#86aab2'], [1, '#62899a']], 0, 0, 1, 0) +
      lin('mc-sun', [[0, '#c99a4a'], [.45, '#e2bf75'], [1, '#f4e1ad']]) +
      lin('mc-tower', [[0, '#e6e2d8'], [.35, '#d4d0c5'], [.75, '#aaa69c'], [1, '#8e8a81']], 0, 0, 1, 0) +
      lin('mc-ribbon', [[0, '#f7f8f4'], [1, '#d7dfe0']]) +
      lin('mc-rust', [[0, '#9a5a3c'], [.5, '#7f4530'], [1, '#5f3324']], 0, 0, 1, 0) +
      lin('mc-ground', [[0, '#e9e6da', 0], [.4, '#e7e2d3', .85], [1, '#e1dac8']]) +
      lin('mc-screen', [[0, '#2f4650'], [1, '#4c6772']]) +
      lin('mc-warm', [[0, '#f6d9a2', 0], [1, '#f6d9a2', .55]]) +
      '<pattern id="mc-lattice" width="26" height="45" patternUnits="userSpaceOnUse"><rect width="26" height="45" fill="#5e1c13"/><path d="M0 0L26 45M26 0L0 45M0 22.5H26" fill="none" stroke="#d8a85a" stroke-width="1.7"/><circle cx="13" cy="22.5" r="4.4" fill="#7a2a1c" stroke="#d8a85a" stroke-width="1.2"/><circle cx="0" cy="0" r="4.4" fill="#7a2a1c" stroke="#d8a85a" stroke-width="1.2"/><circle cx="26" cy="0" r="4.4" fill="#7a2a1c" stroke="#d8a85a" stroke-width="1.2"/><circle cx="0" cy="45" r="4.4" fill="#7a2a1c" stroke="#d8a85a" stroke-width="1.2"/><circle cx="26" cy="45" r="4.4" fill="#7a2a1c" stroke="#d8a85a" stroke-width="1.2"/><circle cx="13" cy="22.5" r="1.5" fill="#d8a85a"/></pattern>' +
      '<radialGradient id="mc-spot"><stop offset="0" stop-color="#fff2cf" stop-opacity=".85"/><stop offset="1" stop-color="#fff2cf" stop-opacity="0"/></radialGradient>' +
      '</defs></svg>';
  }

  /* ── 雁栖湖国际会议中心：白石台基、方柱与通高玻璃，上覆一张灰色大屋顶，四角高挑如鸿雁展翅 ── */
  function yanqiHall(o = {}) {
    const P = Pen(), W = 440, hw = W / 2;
    P.r(-hw - 46, 0, W + 92, 12, 'c-marble l2'); P.p(`M${P2(-hw - 46, 7)}H${N(hw + 46)}`, 'l3');
    P.r(-hw - 28, 12, W + 56, 10, 'c-marble l2');
    A.stairs(P, 0, 22, 62, 18, 6);
    const y0 = 22, H = 64;
    P.r(-hw, y0, W, H, 'g-glassv l2');
    let mu = ''; for (let x = -hw + 8; x < hw; x += 8) mu += `M${P2(x, y0)}V${N(y0 + H)}`; mu += `M${P2(-hw, y0 + H * 0.42)}H${N(hw)}M${P2(-hw, y0 + H * 0.78)}H${N(hw)}`;
    P.p(mu, 'l-grid');
    let pier = ''; for (let x = -hw; x <= hw + 0.1; x += W / 8) pier += rect(x - 5, y0, 10, H); P.p(pier, 'c-gb-l l2');
    P.r(-hw - 8, y0 + H, W + 16, 9, 'c-gb l2'); P.p(`M${P2(-hw - 8, y0 + H + 4.5)}H${N(hw + 8)}`, 'lg');
    let br = ''; for (let x = -hw; x < hw; x += 11) br += rect(x + 2, y0 + H + 9, 6, 5); P.p(br, 'c-gb-d l3');
    J.withLod(0.6, () => A.roof(P, { type: 'wd', ey: y0 + H + 14, ew: hw + 50, rh: 82, rw: hw * 0.46, tone: 'gray', tile: 7, beasts: 0, wen: false, lift: 18, fx: 30, fy: 22, t0: 0.4, ridgeH: 7, sd: 12 }));
    return { g: P.s(), box: [-hw - 96, hw + 96, -2, y0 + H + 14 + 82 + 12] };
  }

  /* ── 日出东方酒店：半圆的楼身像升起的太阳，自底边正中放射出一道道肋，横向是一圈圈楼层线 ── */
  function richu(o = {}) {
    const P = Pen(), a = 160, b = 150, y0 = 10;
    P.r(-a - 26, 0, 2 * a + 52, y0, 'c-gb l2');
    const arc = (s, n = 64) => samp(t => { const th = Math.PI * (1 - t); return [a * s * Math.cos(th), y0 + b * s * Math.sin(th)]; }, n);
    P.p(poly(arc(1)), 'g-sun l1');
    let rib = ''; for (let k = 1; k < 18; k++) { const th = Math.PI * k / 18; rib += `M0 ${N(y0)}L${P2(a * Math.cos(th), y0 + b * Math.sin(th))}`; }
    P.p(rib, 'l-rib');
    let fl = ''; for (let k = 2; k < 10; k++) fl += 'M' + arc(k / 10, 40).map(q => P2(...q)).join('L');
    P.p(fl, 'l-rib2');
    P.p(poly(samp(t => { const th = Math.PI * (0.5 + 0.5 * t); return [a * Math.cos(th), y0 + b * Math.sin(th)]; }, 30).concat([[0, y0]])), 'c-light');
    P.p(`M${P2(-34, y0)}Q0 ${N(y0 + 30)} ${P2(34, y0)}Z`, 'c-pane l2');
    return { g: P.s(), box: [-a - 30, a + 30, -2, y0 + b + 6] };
  }

  /* ── 首钢园：四座双曲线冷却塔（两前两后），三高炉的钢架与管道，前面一条飘带似的滑雪大跳台 ── */
  function coolingTower(P, cx, base, h, rb) {
    const yt = h * 0.76, r0 = rb * 0.58, c = h * 0.3, R = y => r0 * Math.sqrt(1 + ((y - yt) / c) ** 2) * (y > yt ? 0.98 : 1);
    const rBot = R(0), sc = rb / rBot, L = samp(t => { const y = h * t; return [cx - R(y) * sc, base + y]; }, 28), Rr = L.map(([x, y]) => [2 * cx - x, y]);
    P.p(poly([...L, ...Rr.reverse()]), 'g-tower l2');
    let st = ''; for (const f of [-0.6, -0.2, 0.25, 0.62]) st += 'M' + samp(t => { const y = h * (0.06 + 0.9 * t); return P2(cx + R(y) * sc * f, base + y); }, 12).join('L');
    P.p(st, 'l-stain');
    const rt = R(h) * sc; P.e(cx, base + h, rt, rt * 0.12, 'c-tower-in l3');
    let leg = ''; for (let k = -4; k <= 4; k++) { const x = cx + k * rb * 0.22; leg += `M${P2(x - rb * 0.1, base)}L${P2(x + rb * 0.04, base + h * 0.07)}M${P2(x + rb * 0.1, base)}L${P2(x - rb * 0.04, base + h * 0.07)}`; }
    P.p(leg, 'l2');
    P.p(`M${P2(cx - rBot * sc * 0.98, base + h * 0.07)}H${N(cx + rBot * sc * 0.98)}`, 'l2');
  }
  function shougang(o = {}) {
    const P = Pen(), r = rng(5);
    // 三高炉：炉体、钢架、斜管
    const fx = -250;
    P.r(fx - 26, 0, 52, 112, 'g-rust l2');
    let fr = ''; for (let y = 0; y < 112; y += 16) fr += `M${P2(fx - 26, y)}L${P2(fx + 26, y + 16)}M${P2(fx + 26, y)}L${P2(fx - 26, y + 16)}M${P2(fx - 26, y)}H${N(fx + 26)}`; P.p(fr, 'l3');
    P.p(`M${P2(fx - 14, 112)}L${P2(fx - 10, 150)}H${N(fx + 10)}L${P2(fx + 14, 112)}Z`, 'g-rust l2');
    P.r(fx - 4, 150, 8, 26, 'c-rust-d l2');
    P.p(`M${P2(fx + 10, 140)}C${P2(fx + 40, 140)} ${P2(fx + 52, 120)} ${P2(fx + 56, 60)}L${P2(fx + 64, 60)}C${P2(fx + 60, 124)} ${P2(fx + 44, 148)} ${P2(fx + 10, 148)}Z`, 'c-rust-d l2');
    P.r(fx + 52, 0, 16, 62, 'g-rust l2');
    P.r(fx - 60, 0, 24, 196, 'c-concrete l2'); P.r(fx - 62, 190, 28, 6, 'c-concrete-d l3');
    // 冷却塔：后两座、前两座
    for (const [cx, h, rb] of [[-130, 128, 46], [-30, 132, 47]]) coolingTower(P, cx, 14, h, rb);
    for (const [cx, h, rb] of [[-176, 120, 44], [-80, 124, 45]]) coolingTower(P, cx, 0, h, rb);
    // 滑雪大跳台：一整条宽飘带，从冷却塔前的出发区落下——助滑道上陡下缓，跳台口微微上挑，落地坡先陡后平；
    //             飘带在跳台口附近翻转（上白下青对调）；下面一排 75 度斜撑
    const dx = -150;
    const yTop = x => x >= 168 ? 52 + 62 * ((x - 168) / 94) ** 1.5 : x >= 150 ? 52 + 1.2 * (168 - x) / 18 : 53.2 * ((x + 22) / 172) ** 1.6;
    const xs = samp(t => 262 - 284 * t, 140), topL = xs.map(x => [x + dx, yTop(x)]);
    const thick = x => 5 + 13 * Math.max(0, Math.min(1, (x + 22) / 284)) ** 1.2, tw = x => Math.max(0.1, Math.min(0.9, 0.5 + 0.45 * Math.sin((x - 150) / 110 * Math.PI)));
    const midL = xs.map(x => [x + dx, yTop(x) - thick(x) * tw(x)]), botL = xs.map(x => [x + dx, yTop(x) - thick(x)]);
    let sup = ''; for (const x of [246, 226, 206, 188, 170, 128, 96, 64]) { const y = yTop(x) - thick(x); sup += `M${P2(x + dx - 2, y)}L${P2(x + dx - 2 - y * 0.27, 0)}L${P2(x + dx + 3 - y * 0.27, 0)}L${P2(x + dx + 3, y)}Z`; }
    P.p(sup, 'c-steel l3');
    P.p(smooth(topL) + 'L' + smooth(midL.slice().reverse()).slice(1) + 'Z', 'g-ribbon');
    P.p(smooth(midL) + 'L' + smooth(botL.slice().reverse()).slice(1) + 'Z', 'c-ribbon-b');
    P.p(smooth(topL) + 'L' + smooth(botL.slice().reverse()).slice(1) + 'Z', 'l2');
    P.p(smooth(midL), 'l-ribbon');
    P.p(smooth(topL.map(([x, y]) => [x, y - 1.6]).filter(([x]) => x > 168 + dx || x < 150 + dx)), 'l-ribbon');
    // 出发区：坡顶一座小方楼
    P.r(250 + dx, 108, 28, 18, 'g-ribbon l2'); P.p(`M${P2(247 + dx, 126)}h34l-3 5h-28Z`, 'c-steel l3'); P.r(255 + dx, 114, 18, 7, 'c-pane l3');
    // 筒仓
    for (const [x, w, h] of [[150, 26, 70], [180, 26, 70]]) { P.r(x - w / 2, 0, w, h, 'c-concrete l2'); P.e(x, h, w / 2, 3, 'c-concrete-d l3'); }
    return { g: P.s(), box: [-320, 200, -4, 202] };
  }

  /* ── 国家会议中心：长长一道通高玻璃幕墙，竖向白鳍，屋檐是一道起伏的长弧，两端微翘 ── */
  function cncc(o = {}) {
    const P = Pen(), W = o.w ?? 720, hw = W / 2;
    P.r(-hw - 40, 0, W + 80, 6, 'c-stone l2'); P.r(-hw - 24, 6, W + 48, 5, 'c-stone l3');
    const y0 = 11, H = 74;
    P.r(-hw, y0, W, H, 'g-glassv l2');
    let mu = ''; for (let x = -hw + 6; x < hw; x += 6) mu += `M${P2(x, y0)}V${N(y0 + H)}`; mu += `M${P2(-hw, y0 + 26)}H${N(hw)}M${P2(-hw, y0 + 50)}H${N(hw)}`;
    P.p(mu, 'l-grid');
    let fin = ''; for (let x = -hw; x <= hw + 0.1; x += 30) fin += rect(x - 2, y0, 4, H); P.p(fin, 'c-gb-l l3');
    P.p(rect(-hw, y0, W, 10), 'c-shade');
    // 屋檐：一道长长的薄檐，轻轻起伏，两端像飞檐一样上翘；檐下一道阴影
    const ov = 26, Y = x => { const u = x / (hw + ov); return y0 + H + 5 + 3 * Math.cos(u * Math.PI * 2) + 16 * Math.abs(u) ** 5; };
    const xs = samp(t => -hw - ov + 2 * (hw + ov) * t, 120);
    const topE = xs.map(x => [x, Y(x) + 8]), botE = xs.map(x => [x, Y(x)]);
    P.p(smooth(botE.filter(([x]) => Math.abs(x) <= hw)) + 'L' + smooth(botE.filter(([x]) => Math.abs(x) <= hw).map(([x, y]) => [x, y - 7]).reverse()).slice(1) + 'Z', 'c-soffit');
    P.p(smooth(topE) + 'L' + smooth(botE.slice().reverse()).slice(1) + 'Z', 'c-gb-l l1');
    P.p(smooth(topE.map(([x, y]) => [x, y - 3.2])), 'l3');
    return { g: P.s(), box: [-hw - ov - 16, hw + ov + 16, -2, y0 + H + 34] };
  }

  /* ── 现代楼：一栋玻璃楼，可选顶部样式 ── */
  function glassTower(P, x, w, h, o = {}) {
    const top = o.top || 'flat';
    if (top === 'curve') P.p(`M${P2(x - w / 2, 0)}V${N(h)}Q${P2(x, h + w * 0.5)} ${P2(x + w / 2, h)}V0Z`, `${o.cls || 'g-glassv'} l2`);
    else if (top === 'slant') P.p(`M${P2(x - w / 2, 0)}V${N(h)}L${P2(x + w / 2, h + w * 0.4)}V0Z`, `${o.cls || 'g-glassv'} l2`);
    else P.r(x - w / 2, 0, w, h, `${o.cls || 'g-glassv'} l2`);
    let g = ''; for (let y = 6; y < h; y += 6) g += `M${P2(x - w / 2 + 0.8, y)}h${N(w - 1.6)}`; for (let k = 1; k < Math.round(w / 9); k++) g += `M${P2(x - w / 2 + w * k / Math.round(w / 9), 2)}V${N(h - 2)}`;
    P.p(g, 'l-grid');
    P.r(x + w * 0.16, 0, w * 0.34, h, 'c-shade');
    if (top === 'flat') { P.r(x - w / 2 - 1.5, h, w + 3, 4, 'c-gb-l l3'); P.r(x - w * 0.18, h + 4, w * 0.36, 6, 'c-gb l3'); }
  }
  function zgc(o = {}) {
    const P = Pen();
    glassTower(P, -64, 46, 150, { top: 'slant', cls: 'g-glassh' }); glassTower(P, 60, 52, 168, { top: 'flat' });
    glassTower(P, -4, 58, 214, { top: 'curve' }); glassTower(P, -110, 40, 96, { cls: 'g-glassh' }); glassTower(P, 112, 44, 112, { cls: 'g-glassh' });
    P.r(-140, 0, 280, 8, 'c-stone l2');
    return { g: P.s(), box: [-142, 142, -2, 248] };
  }
  /* ── 丽泽：一栋扭转上升的塔楼（两半围着中庭旋转），旁边两栋方楼 ── */
  function lize(o = {}) {
    const P = Pen(), H = 250;
    glassTower(P, -70, 44, 130, { cls: 'g-glassh' }); glassTower(P, 76, 46, 150);
    const L = samp(t => [-30 + 10 * Math.sin(t * Math.PI * 1.2), H * t], 24), R = samp(t => [30 - 10 * Math.sin(t * Math.PI * 1.2 + 0.6), H * t], 24), M1 = samp(t => [-4 + 8 * Math.sin(t * Math.PI * 1.2 + 0.3), H * t], 24), M2 = M1.map(([x, y]) => [x + 8, y]);
    P.p(smooth(L) + 'L' + smooth(M1.slice().reverse()).slice(1) + 'Z', 'g-glassv l2');
    P.p(smooth(M2) + 'L' + smooth(R.slice().reverse()).slice(1) + 'Z', 'g-glassh l2');
    P.p(smooth(M1) + 'L' + smooth(M2.slice().reverse()).slice(1) + 'Z', 'c-pane');
    let fl = ''; for (let k = 1; k < 40; k++) { const i = Math.round(k / 40 * 24); fl += `M${P2(L[i][0] + 1, L[i][1])}L${P2(M1[i][0], M1[i][1] + 1.6)}M${P2(M2[i][0], M2[i][1] - 1.6)}L${P2(R[i][0] - 1, R[i][1])}`; }
    P.p(fl, 'l-grid');
    P.p(`M${P2(-30, H)}Q0 ${N(H + 8)} ${P2(30, H)}`, 'l2');
    P.r(-110, 0, 220, 8, 'c-stone l2');
    return { g: P.s(), box: [-112, 112, -2, H + 12] };
  }

  /* ── 大巴、人群 ── */
  function coach(o = {}) {
    const P = Pen(), dir = o.dir || 1, X = x => x * dir;
    P.p(`M${P2(X(-60), 6)}V30C${P2(X(-60), 40)} ${P2(X(-54), 44)} ${P2(X(-44), 44)}H${N(X(52))}C${P2(X(58), 44)} ${P2(X(62), 40)} ${P2(X(62), 32)}V14L${P2(X(56), 6)}Z`, 'c-bus l2');
    P.p(`M${P2(X(-54), 22)}H${N(X(48))}V37H${N(X(-54))}Z`, 'c-pane l3');
    let m = ''; for (let x = -38; x < 48; x += 16) m += `M${P2(X(x), 22)}V37`; P.p(m, 'l-mull');
    P.p(`M${P2(X(50), 22)}H${N(X(60))}V37H${N(X(54))}Z`, 'c-pane l3'); P.p(`M${P2(X(-60), 15)}H${N(X(62))}V18H${N(X(-60))}Z`, 'c-qing');
    for (const x of [-38, 38]) { P.c(X(x), 6, 7, 'c-ink'); P.c(X(x), 6, 3, 'c-gray'); }
    return { g: P.s(), box: [-64, 66, -2, 46] };
  }
  function crowd(n, o = {}) {
    const P = Pen(), r = rng(o.seed || 3), cs = ['k', 'b', 'w', 'k', 'b', 'k', 'w', 'g'];
    for (let i = 0; i < n; i++) { const L = Nt.person({ h: 40, c: cs[i % cs.length], pose: o.pose || (r() < 0.6 ? 'walk' : 'stand'), dir: o.dir || 1, hair: r() < 0.3 ? 'long' : 'short', bag: r() < 0.2 }); P.g(L.g, ` transform="translate(${N(i * (o.gap || 13) + r() * 4)} ${N((i % 3) * 2)})"`); }
    return { g: P.s(), box: [-14, n * (o.gap || 13) + 14, 0, 42] };
  }

  /* ── 场景里的小人：正面坐着、背影坐着 ── */
  function seatedFront(P, x, y, s, c) {
    P.p(`M${P2(x - s * 0.34, y)}V${N(y + s * 0.5)}Q${P2(x - s * 0.34, y + s * 0.62)} ${P2(x - s * 0.22, y + s * 0.62)}H${N(x + s * 0.22)}Q${P2(x + s * 0.34, y + s * 0.62)} ${P2(x + s * 0.34, y + s * 0.5)}V${N(y)}Z`, 'c-chair l3');
    P.p(`M${P2(x - s * 0.12, y + s * 0.26)}L${P2(x - s * 0.14, y)}M${P2(x + s * 0.12, y + s * 0.26)}L${P2(x + s * 0.14, y)}`, 'l-leg');
    P.p(`M${P2(x - s * 0.2, y + s * 0.26)}C${P2(x - s * 0.23, y + s * 0.6)} ${P2(x - s * 0.15, y + s * 0.73)} ${P2(x, y + s * 0.74)}C${P2(x + s * 0.15, y + s * 0.73)} ${P2(x + s * 0.23, y + s * 0.6)} ${P2(x + s * 0.2, y + s * 0.26)}Z`, `c-robe-${c} l3`);
    P.p(`M${P2(x - s * 0.06, y + s * 0.73)}L${P2(x, y + s * 0.6)}L${P2(x + s * 0.06, y + s * 0.73)}Z`, 'c-robe-w');
    P.p(`M${P2(x - s * 0.02, y + s * 0.74)}h${N(s * 0.04)}v${N(s * 0.04)}h${N(-s * 0.04)}Z`, 'c-skin');
    P.e(x, y + s * 0.88, s * 0.1, s * 0.12, 'c-skin l3');
    P.p(`M${P2(x - s * 0.105, y + s * 0.9)}C${P2(x - s * 0.11, y + s * 1.03)} ${P2(x + s * 0.11, y + s * 1.03)} ${P2(x + s * 0.105, y + s * 0.9)}Q${P2(x, y + s * 0.95)} ${P2(x - s * 0.105, y + s * 0.9)}Z`, 'c-hair');
  }
  function seatedBack(P, x, y, s, c) {
    P.p(`M${P2(x - s * 0.5, y)}C${P2(x - s * 0.52, y + s * 0.5)} ${P2(x - s * 0.36, y + s * 0.68)} ${P2(x, y + s * 0.7)}C${P2(x + s * 0.36, y + s * 0.68)} ${P2(x + s * 0.52, y + s * 0.5)} ${P2(x + s * 0.5, y)}Z`, `c-robe-${c} l3`);
    P.r(x - s * 0.08, y + s * 0.66, s * 0.16, s * 0.1, 'c-skin');
    P.e(x, y + s * 0.9, s * 0.19, s * 0.23, 'c-hair l3');
  }
  const putP = (P, L, x, y, k = 1) => P.g(L.g, ` transform="translate(${N(x)} ${N(y)})${k !== 1 ? ` scale(${N(k)})` : ''}"`);
  // 一幅山水小画（墙上的画、背景板、屏幕上的画面）
  function landscape(P, x, y, w, h, o = {}) {
    P.r(x - w / 2, y, w, h, `${o.frame || 'c-paper'} l2`);
    const mt = Nt.mountain({ w: w * 0.86, h: h * 0.66, seed: o.seed || 5, peaks: [[-0.5, 0.7, 0.26], [0, 1, 0.3], [0.55, 0.6, 0.26]], fans: false, trees: false, fill: o.fill || 'g-shan' });
    P.g(mt.g, ` transform="translate(${N(x)} ${N(y + h * 0.1)})"`);
  }
  const wallBg = (P, y0 = -40, y1 = 170, cls = 'c-wall-in') => P.r(-170, y0, 340, y1 - y0, cls);

  /* ── 六类会议 ── */
  // 工作会议：长条会议桌，对面一排正面坐着，近处一排背影；桌上名牌、茶杯、话筒，中间一溜绿植；墙上一幅山水
  function vWork() {
    const P = Pen();
    wallBg(P); P.r(-170, -40, 340, 74, 'c-floor');
    landscape(P, 0, 92, 150, 62, { seed: 3 });
    for (let i = 0; i < 5; i++) seatedFront(P, -112 + i * 56, 30, 54, ['k', 'b', 'k', 'w', 'k'][i]);
    P.p('M-160 40H160L150 26H-150Z', 'c-table l2'); P.p('M-160 40H160V34H-160Z', 'c-cloth l3');
    let tents = '', cups = '', mic = '';
    for (let i = 0; i < 5; i++) { const x = -112 + i * 56; tents += `M${P2(x - 12, 33)}l3 7h18l3 -7Z`; cups += rect(x + 15, 33, 5, 6) + rect(x + 14, 39, 7, 1.4); mic += `M${P2(x - 18, 34)}v3q0 9 8 12`; }
    P.p(tents, 'c-tent l3'); P.p(cups, 'c-trim l3'); P.p(mic, 'l2');
    for (const x of [-84, -28, 28, 84]) P.p(`M${P2(x - 4, 30)}h8l-1 4h-6Z M${P2(x - 7, 34)}c-2 6 4 8 7 4c3 4 9 2 7 -4c-3 2 -11 2 -14 0Z`, 'c-leaf l3');
    for (let i = 0; i < 6; i++) seatedBack(P, -140 + i * 56, -30, 46, ['b', 'k', 'w', 'k', 'b', 'k'][i]);
    return { g: P.s(), box: [-170, 170, -42, 170] };
  }
  // 培训班：投影幕布上一幅山水，讲师在一旁讲解；前面一排排课桌与背影
  function vTrain() {
    const P = Pen();
    wallBg(P);
    P.r(-130, 64, 170, 96, 'c-screen-f l2'); P.r(-122, 70, 154, 84, 'c-paper l3');
    P.r(-122, 140, 154, 14, 'c-qing'); P.r(-114, 145, 70, 4, 'c-paper');
    let bl = ''; for (let k = 0; k < 4; k++) bl += `M${P2(-112, 128 - k * 14)}h4M${P2(-104, 128 - k * 14)}h${N(52 - (k % 2) * 14)}`; P.p(bl, 'l2');
    let bar = ''; for (const [x, h] of [[-36, 22], [-24, 34], [-12, 28], [0, 46], [12, 40]]) bar += rect(x, 80, 8, h); P.p(bar, 'c-lu l3'); P.p('M-40 80H24', 'l2');
    putP(P, Nt.person({ h: 78, c: 'k', pose: 'point', dir: -1 }), 108, 36);
    P.r(-170, 34, 340, 4, 'c-wood-l l3');
    for (const [y, s] of [[10, 44], [-30, 52]]) { P.r(-170, y, 340, 6, 'c-wood l2'); for (let i = 0; i < 6; i++) seatedBack(P, -140 + i * 56 + (y > 0 ? 18 : 0), y - 4, s, ['k', 'b', 'w', 'k', 'b', 'k'][(i + (y > 0 ? 2 : 0)) % 6]); }
    return { g: P.s(), box: [-170, 170, -42, 170] };
  }
  // 论坛研讨：台上四位嘉宾坐在扶手椅上，主持人在讲台边；背景板一幅青绿山水；前排观众背影
  function vForum() {
    const P = Pen();
    wallBg(P, -40, 170, 'c-hall');
    P.r(-160, 40, 320, 126, 'c-board l2'); landscape(P, 0, 70, 250, 84, { seed: 9, frame: 'c-board-in' });
    P.r(-170, 22, 340, 18, 'c-stage l2'); P.p('M-170 22H170', 'l2');
    for (let i = 0; i < 4; i++) seatedFront(P, -96 + i * 54, 40, 50, ['k', 'w', 'k', 'b'][i]);
    for (const x of [-69, -15, 39]) P.p(`M${P2(x - 8, 40)}h16v10h-16Z`, 'c-wood l3');
    P.p('M120 40L124 82H146L150 40Z', 'c-wood l2'); putP(P, Nt.person({ h: 74, c: 'k', pose: 'stand', dir: -1 }), 154, 40);
    for (let i = 0; i < 7; i++) seatedBack(P, -150 + i * 50, -36, 48, ['k', 'b', 'k', 'w', 'k', 'b', 'k'][i]);
    return { g: P.s(), box: [-170, 170, -42, 170] };
  }
  // 年会庆典：舞台两侧一对宫灯，中间主持人，暖光；台下圆桌与宾客
  function vGala() {
    const P = Pen();
    wallBg(P, -40, 170, 'c-hall-d');
    P.r(-170, 30, 340, 140, 'c-stage-bg');
    for (const s of [-1, 1]) { const x0 = s * 170; P.p(`M${P2(x0, 170)}L${P2(x0 - s * 40, 170)}C${P2(x0 - s * 34, 120)} ${P2(x0 - s * 52, 70)} ${P2(x0 - s * 30, 30)}L${P2(x0, 30)}Z`, 'c-curtain l2'); }
    let val = 'M-170 170H170V156'; for (let x = 170; x > -170; x -= 20) val += `Q${P2(x - 10, 146)} ${P2(x - 20, 156)}`; P.p(val + 'Z', 'c-curtain l2');
    P.e(0, 70, 120, 60, 'g-warm');
    for (const x of [-90, 90]) { P.p(`M${P2(x, 156)}V${N(132)}`, 'l2'); P.p(`M${P2(x - 13, 132)}h26l-3 -4h-20Z M${P2(x - 16, 128)}h32v-26h-32Z M${P2(x - 13, 102)}h26l-3 -4h-20Z`, 'c-lantern l2'); P.p(`M${P2(x - 16, 124)}h32M${P2(x - 16, 106)}h32M${P2(x - 8, 128)}v-26M${P2(x + 8, 128)}v-26`, 'lg'); P.p(`M${P2(x - 4, 98)}v-14M${P2(x, 98)}v-16M${P2(x + 4, 98)}v-14`, 'l-tassel'); }
    P.r(-170, 22, 340, 10, 'c-stage l2');
    putP(P, Nt.person({ h: 70, c: 'w', pose: 'stand', dir: 1, hair: 'long' }), -12, 32); putP(P, Nt.person({ h: 74, c: 'k', pose: 'stand', dir: -1 }), 18, 32);
    for (const [x, y] of [[-110, -18], [0, -24], [110, -18]]) { P.e(x, y, 46, 9, 'c-cloth-w l2'); P.p(`M${P2(x - 46, y)}V${N(y - 12)}Q${P2(x, y - 22)} ${P2(x + 46, y - 12)}V${N(y)}`, 'c-cloth-w l2'); for (const dx of [-34, -12, 12, 34]) seatedBack(P, x + dx, y - 26, 30, ['k', 'b', 'w', 'k'][(dx + 34) / 22 | 0]); P.p(`M${P2(x - 3, y + 2)}c-6 -8 6 -12 3 -2c6 -6 10 2 3 4Z`, 'c-flower-r'); }
    return { g: P.s(), box: [-170, 170, -50, 170] };
  }
  // 发布会：背景板、讲台与发言人，一侧展台上的模型在灯下；前面一排三脚架与记者
  function vLaunch() {
    const P = Pen();
    wallBg(P, -40, 170, 'c-hall');
    P.r(-160, 34, 320, 132, 'c-board-l l2'); P.r(-160, 86, 320, 26, 'c-qing'); P.p('M-160 86H160M-160 112H160', 'lg');
    landscape(P, 60, 118, 120, 40, { seed: 4, frame: 'c-board-l' });
    P.r(-170, 24, 340, 12, 'c-stage l2');
    P.p('M-40 36L-36 84H-4L0 36Z', 'c-wood l2'); P.p('M-20 84L-26 96', 'l2'); putP(P, Nt.person({ h: 80, c: 'k', pose: 'stand', dir: 1 }), -54, 36);
    P.e(100, 70, 40, 46, 'g-spot'); P.r(84, 36, 32, 30, 'c-gb-l l2'); P.p('M88 66L100 92L112 66Z', 'c-glass l2'); P.p('M94 66L100 80L106 66', 'l3');
    for (let i = 0; i < 5; i++) { const x = -136 + i * 66; P.p(`M${P2(x, -40)}L${P2(x + 8, -6)}L${P2(x + 16, -40)}M${P2(x + 8, -6)}V${N(-40)}`, 'l2'); P.r(x - 2, -6, 20, 12, 'c-ink'); P.r(x + 16, -3, 8, 6, 'c-gray-d'); putP(P, Nt.person({ h: 52, c: ['b', 'k', 'w', 'k', 'b'][i], pose: 'photo', back: true }), x + 30, -40); }
    return { g: P.s(), box: [-170, 170, -42, 170] };
  }
  // 展览展示：三个展位，展板上是山水与图表色块，中间台子上一座城市模型；观众与讲解员
  function vExpo() {
    const P = Pen();
    wallBg(P, -40, 170, 'c-hall');
    for (const [x, w, c] of [[-112, 100, 'c-qing'], [0, 108, 'c-lu'], [112, 100, 'c-qing']]) {
      P.r(x - w / 2, 40, w, 112, 'c-board-l l2'); P.r(x - w / 2, 140, w, 12, c);
      landscape(P, x, 82, w * 0.8, 50, { seed: Math.abs(x) + 3, frame: 'c-paper' });
      P.r(x - w * 0.4, 50, w * 0.22, 24, 'c-qing-l l3'); P.r(x - w * 0.14, 50, w * 0.22, 16, 'c-gold l3'); P.r(x + w * 0.12, 50, w * 0.22, 30, 'c-lu-l l3');
    }
    P.r(-50, 0, 100, 26, 'c-gb-l l2'); let blk = ''; for (const [x, w, h] of [[-38, 10, 18], [-24, 12, 30], [-8, 9, 22], [6, 14, 36], [24, 10, 16], [36, 8, 26]]) blk += rect(x, 26, w, h); P.p(blk, 'c-glass2 l3'); P.p('M-50 30C-20 34 10 26 50 32L50 26H-50Z', 'c-water l3');
    putP(P, Nt.person({ h: 74, c: 'o', pose: 'point', dir: 1, hair: 'long' }), -92, -30);
    [['k', 'clasp'], ['b', 'stand'], ['w', 'photo'], ['k', 'stand']].forEach(([c, pose], i) => putP(P, Nt.person({ h: 70, c, pose, dir: -1, hair: i === 2 ? 'long' : 'short' }), -40 + i * 46, -34));
    return { g: P.s(), box: [-170, 170, -42, 170] };
  }

  /* ── 研学三张卡 ── */
  // 干部研修：研讨室里围坐交流，讲师在板前画结构图；窗外西山
  function vCadre() {
    const P = Pen();
    wallBg(P);
    for (const x of [-120, 120]) { P.r(x - 34, 70, 68, 84, 'c-wood l2'); P.r(x - 30, 74, 60, 76, 'c-sky-w'); const mt = Nt.mountain({ w: 56, h: 30, seed: 6, peaks: [[-0.4, 0.8, 0.3], [0.3, 1, 0.3]], fans: false, trees: false, fill: 'g-shan-far' }); P.g(mt.g, ` transform="translate(${N(x)} 74)"`); P.p(`M${P2(x, 74)}V150M${P2(x - 30, 112)}H${N(x + 30)}`, 'l2'); }
    P.r(-56, 70, 112, 78, 'c-paper l2'); P.p('M-30 128h20v-12h-20ZM10 128h20v-12h-20ZM-10 98h20v-12h-20ZM-20 116L0 98M20 116L0 98', 'l2');
    putP(P, Nt.person({ h: 80, c: 'k', pose: 'point', dir: 1 }), -72, 36);
    P.e(0, 14, 130, 22, 'c-table l2'); P.p('M-130 14V8Q0 -12 130 8V14', 'c-wood l2');
    for (let i = 0; i < 4; i++) seatedFront(P, -66 + i * 44, 22, 40, ['b', 'k', 'w', 'k'][i]);
    let bk = ''; for (const x of [-60, -18, 26, 70]) bk += rect(x - 8, 26, 16, 10); P.p(bk, 'c-paper l3');
    for (let i = 0; i < 5; i++) seatedBack(P, -116 + i * 58, -30, 44, ['k', 'b', 'k', 'w', 'b'][i]);
    return { g: P.s(), box: [-170, 170, -42, 170] };
  }
  // 产业对标：智能产线旁参观，机械臂、传送带、大屏数据；讲解员带一队人
  function vBench() {
    const P = Pen();
    wallBg(P, -40, 170, 'c-hall');
    P.r(-150, 92, 150, 70, 'g-screen l2');
    P.p('M-136 104h12v24h-12ZM-118 104h12v38h-12ZM-100 104h12v30h-12ZM-82 104h12v46h-12Z', 'c-qing-l'); P.p('M-60 110L-44 128L-30 120L-16 146', 'lg2'); P.c(-16, 146, 2.4, 'c-gold');
    P.r(-170, 40, 340, 12, 'c-steel l2'); let roll = ''; for (let x = -164; x < 170; x += 10) roll += `M${P2(x, 46)}a2.6 2.6 0 1 0 5.2 0a2.6 2.6 0 1 0 -5.2 0Z`; P.p(roll, 'c-gray l3');
    for (const x of [-120, -40, 40]) P.r(x, 52, 30, 16, 'c-gb-l l3');
    const ax = 96; P.r(ax - 16, 52, 32, 10, 'c-gray-d l2'); P.p(`M${P2(ax - 6, 62)}L${P2(ax - 10, 112)}L${P2(ax + 2, 114)}L${P2(ax + 6, 62)}Z`, 'c-arm l2'); P.p(`M${P2(ax - 10, 110)}L${P2(ax + 40, 132)}L${P2(ax + 44, 122)}L${P2(ax + 2, 102)}Z`, 'c-arm l2'); P.p(`M${P2(ax + 40, 132)}L${P2(ax + 50, 104)}L${P2(ax + 58, 106)}L${P2(ax + 46, 130)}Z`, 'c-arm l2'); P.c(ax - 4, 112, 6, 'c-gray-d l2'); P.c(ax + 42, 128, 5, 'c-gray-d l2'); P.p(`M${P2(ax + 50, 104)}l-4 -8m12 10l2 -9`, 'l2');
    putP(P, Nt.person({ h: 76, c: 'o', pose: 'point', dir: 1, hair: 'long' }), -110, -32);
    [['k', 'clasp'], ['b', 'stand'], ['k', 'photo'], ['w', 'stand'], ['k', 'clasp']].forEach(([c, pose], i) => putP(P, Nt.person({ h: 70, c, pose, dir: -1, hair: i === 3 ? 'long' : 'short' }), -60 + i * 40, -36));
    return { g: P.s(), box: [-170, 170, -42, 170] };
  }
  // 专题考察：城市规划沙盘前，讲解员指着模型讲；墙上一幅城市全景
  function vSurvey() {
    const P = Pen();
    wallBg(P, -40, 170, 'c-hall');
    P.r(-160, 96, 320, 66, 'c-board-l l2');
    let sky = ''; const r = rng(8); for (let x = -150; x < 150; x += 12) { const h = 10 + r() * 34; sky += rect(x, 104, 10, h); } P.p(sky, 'c-glass2 l3'); P.p('M-160 104H160', 'l2');
    const mt = Nt.mountain({ w: 300, h: 28, seed: 9, peaks: [[-0.6, 0.8, 0.2], [-0.1, 1, 0.24], [0.5, 0.7, 0.22]], fans: false, trees: false, fill: 'g-shan-far' }); P.g(mt.g, ' transform="translate(0 128) scale(1 .6)"');
    P.p('M-150 22L-120 64H120L150 22Z', 'c-sand l2'); P.p('M-150 22H150V12H-150Z', 'c-wood l2');
    P.p('M-120 62C-60 50 -40 36 0 40C40 44 80 30 136 26L140 30C84 36 44 50 0 46C-40 42 -60 56 -116 64Z', 'c-water l3');
    let b = ''; for (const [x, y, w, h] of [[-96, 34, 10, 14], [-80, 30, 12, 20], [-60, 36, 8, 10], [-30, 26, 14, 24], [-10, 30, 10, 16], [20, 28, 12, 30], [44, 34, 9, 12], [64, 26, 12, 22], [90, 32, 10, 14], [-110, 46, 9, 8], [100, 44, 8, 10]]) b += rect(x, y, w, h); P.p(b, 'c-gb-l l3');
    let gr = ''; for (const [x, y, R2] of [[-50, 52, 7], [6, 52, 6], [76, 48, 7], [116, 38, 6], [-126, 30, 6]]) gr += `M${P2(x - R2, y)}a${N(R2)} ${N(R2 * 0.6)} 0 1 0 ${N(R2 * 2)} 0a${N(R2)} ${N(R2 * 0.6)} 0 1 0 ${N(-R2 * 2)} 0Z`; P.p(gr, 'c-tree l3');
    putP(P, Nt.person({ h: 76, c: 'o', pose: 'point', dir: 1, hair: 'long' }), -140, -36);
    [['k', 'clasp'], ['b', 'stand'], ['w', 'stand'], ['k', 'photo'], ['b', 'clasp']].forEach(([c, pose], i) => putP(P, Nt.person({ h: 64, c, pose, back: true, hair: i === 2 ? 'long' : 'short' }), -70 + i * 50, -40));
    return { g: P.s(), box: [-170, 170, -42, 170] };
  }

  /* ── 水面：上沿微起伏、两头圆收的一片湖 ── */
  function pond(o = {}) {
    const r = rng(o.seed || 7), w = o.w || 800, h = o.h || 60, e = o.end ?? Math.min(w * 0.18, h * 2.4), hw = w / 2;
    const top = samp(t => { const x = -hw + w * t, k = Math.min(1, (hw - Math.abs(x)) / e); return [x, h * Math.sqrt(Math.max(0, 1 - (1 - k) ** 2)) * (1 - 0.03 * Math.sin(t * 9) - 0.01 * r())]; }, 80);
    const d = smooth(top) + 'Z';
    return { g: `<path class="${o.cls || 'g-mcwater'} l3" d="${d}"/>`, box: [-hw, hw, 0, h] };
  }
  /* ── 城市带：一溜低矮的楼和树，远处的城（场地之间的过渡） ── */
  function cityBand(o = {}) {
    const P = Pen(), r = rng(o.seed || 21), w = o.w || 1400, hw = w / 2;
    let x = -hw;
    const blocks = [];
    while (x < hw) { const bw = 24 + r() * 46, bh = 14 + r() * 34; blocks.push([x + bw / 2, bw, bh, r()]); x += bw + 4 + r() * 18; }
    for (const [cx, bw, bh, k] of blocks) {
      P.r(cx - bw / 2, 0, bw, bh, `${k < 0.5 ? 'c-gb-l' : 'g-glassv'} l3`);
      let g = ''; for (let y = 5; y < bh - 2; y += 5) g += `M${P2(cx - bw / 2 + 2, y)}h${N(bw - 4)}`; P.p(g, 'l-grid');
      if (k > 0.7) P.r(cx - bw * 0.2, bh, bw * 0.4, 4, 'c-gb l3');
    }
    let tr = ''; for (let i = 0; i < w / 40; i++) { const tx = -hw + r() * w, R = 9 + r() * 7; tr += Nt.clumpPath(tx, R * 0.5, R, R * 0.7, r, 8); }
    P.p(tr, 'c-tree l3');
    return { g: P.s(), box: [-hw, hw, -2, 56] };
  }

  /* ── 隔扇门（推门见客用）：一扇五抹头隔扇，按门的实际大小画——格心是大号菱花，绦环板、裙板金线，转角包金看叶 ── */
  function doorLeaf(P, x0, x1, y0, y1) {
    const w = x1 - x0, h = y1 - y0, f = Math.max(10, w * 0.07), Y = t => y0 + h * t;
    P.r(x0, y0, w, h, 'c-door-red l1');
    const cell = (a, b, cls) => P.r(x0 + f, Y(a), w - 2 * f, Y(b) - Y(a), cls);
    const gold = (a, b, inset) => P.p(`M${P2(x0 + f + inset, Y(a) + inset)}h${N(w - 2 * f - 2 * inset)}v${N(Y(b) - Y(a) - 2 * inset)}h${N(-(w - 2 * f - 2 * inset))}Z`, 'lg2');
    // 裙板：暗红底，金线框，正中一朵如意云
    cell(0.05, 0.27, 'c-door-red-d l2'); gold(0.05, 0.27, f * 0.45);
    const cx = (x0 + x1) / 2, cy = Y(0.16), r = Math.min(w * 0.16, h * 0.05);
    P.p(`M${P2(cx - r * 2.2, cy)}C${P2(cx - r * 2.2, cy + r * 1.2)} ${P2(cx - r * 0.8, cy + r * 1.3)} ${P2(cx - r * 0.5, cy + r * 0.5)}C${P2(cx - r * 0.3, cy + r * 1.5)} ${P2(cx + r * 0.3, cy + r * 1.5)} ${P2(cx + r * 0.5, cy + r * 0.5)}C${P2(cx + r * 0.8, cy + r * 1.3)} ${P2(cx + r * 2.2, cy + r * 1.2)} ${P2(cx + r * 2.2, cy)}C${P2(cx + r * 2.2, cy - r * 1.2)} ${P2(cx + r * 0.8, cy - r * 1.3)} ${P2(cx + r * 0.5, cy - r * 0.5)}C${P2(cx + r * 0.3, cy - r * 1.5)} ${P2(cx - r * 0.3, cy - r * 1.5)} ${P2(cx - r * 0.5, cy - r * 0.5)}C${P2(cx - r * 0.8, cy - r * 1.3)} ${P2(cx - r * 2.2, cy - r * 1.2)} ${P2(cx - r * 2.2, cy)}Z`, 'lg2');
    // 绦环板（下、上）
    for (const [a, b] of [[0.3, 0.36], [0.92, 0.965]]) { cell(a, b, 'c-door-red-d l2'); gold(a, b, f * 0.3); }
    // 格心：大号菱花
    cell(0.39, 0.89, 'p-mclattice l2'); gold(0.39, 0.89, -f * 0.18);
    // 抹头线
    let rails = ''; for (const t of [0.05, 0.27, 0.3, 0.36, 0.39, 0.89, 0.92, 0.965]) rails += `M${P2(x0, Y(t))}H${N(x1)}`; P.p(rails, 'l2');
    // 看叶：边梃与抹头交接处包金
    let kan = ''; for (const t of [0.05, 0.27, 0.39, 0.89]) for (const [sx, xx] of [[1, x0], [-1, x1]]) kan += `M${P2(xx, Y(t) - f * 0.7)}h${N(sx * f * 1.25)}v${N(f * 1.4)}h${N(-sx * f * 1.25)}Z`;
    P.p(kan, 'g-gold l3');
    // 一明一暗：门扇左上受光
    P.p(`M${P2(x0, y1)}h${N(w)}l${N(-f * 0.5)} ${N(-f * 0.5)}h${N(-(w - f))}v${N(-(h - f))}l${N(-f * 0.5)} ${N(-f * 0.5)}Z`, 'c-light');
  }

  /* ── 主视觉《京华会客厅》 ── */
  function compose(layout = 'desk', px = 1) {
    const D = layout === 'desk', W = D ? 1600 : 400, H = D ? 760 : 860, L = {}, hits = [];
    const add = (id, s) => { L[id] = (L[id] || '') + s; };
    const put = (id, el, cx, bottom, h, cls = '', hit) => { const s = h / (el.box[3] - el.box[2]); add(id, J.place(el, cx, bottom + el.box[2] * s, h, px, cls)); if (hit) hits.push({ id: hit, x: cx + el.box[0] * s, y: bottom - h, w: (el.box[1] - el.box[0]) * s, h }); return s; };
    const mist = (id, cx, bottom, w, h) => put(id, Nt.mistBlob({ w, h }), cx, bottom, h);
    let slips;
    if (D) {
      add('sky', `<rect width="${W}" height="${H}" fill="url(#mc-sky)"/>`);
      put('far', Nt.range({ w: 1700, h: 210, seed: 17, fill: 'g-shan-far', peaks: [[-0.9, 0.86, 0.08], [-0.66, 1, 0.09], [-0.4, 0.74, 0.09], [-0.12, 0.5, 0.1], [0.2, 0.4, 0.12], [0.52, 0.58, 0.1], [0.8, 0.8, 0.09]] }), 800, 400, 210, 'far');
      put('far', Nt.range({ w: 980, h: 170, seed: 19, fill: 'g-shan-mid', peaks: [[-0.8, 0.8, 0.12], [-0.4, 1, 0.12], [0.05, 0.7, 0.14], [0.5, 0.5, 0.14], [0.86, 0.3, 0.14]] }), 430, 410, 170, 'far');
      mist('far', 800, 430, 1800, 80);
      // 雁栖湖：一片湖，岸上日出东方，湖心岛上会议中心
      put('back', pond({ w: 1020, h: 74, seed: 3 }), 470, 470, 74);
      put('back', Nt.waves({ w: 900, h: 60, s: 0.7, seed: 5 }), 470, 466, 60);
      put('back', Nt.bank({ w: 420, h: 16, seed: 2, end: 0.2 }), 186, 412, 16);
      put('back', richu(), 186, 404, 104, '', 'richu');
      put('back', Nt.bank({ w: 560, h: 18, seed: 4, end: 0.18 }), 560, 420, 18);
      put('back', yanqiHall(), 560, 412, 120, '', 'yanqi');
      put('cloud1', Nt.cloud2({ w: 300, h: 34, seed: 3 }), 980, 220, 48);
      put('cloud1', Nt.cloud2({ w: 240, h: 30, seed: 7, dir: -1 }), 330, 160, 42);
      mist('mid', 800, 500, 1800, 60);
      // 首钢园（右中）：群明湖在前
      put('mid', pond({ w: 700, h: 74, seed: 7 }), 1290, 560, 74);
      put('mid', Nt.waves({ w: 600, h: 60, s: 0.7, seed: 9 }), 1290, 556, 60);
      put('mid', Nt.bank({ w: 620, h: 18, seed: 6, end: 0.16 }), 1250, 534, 18);
      put('mid', shougang(), 1270, 528, 196, '', 'shougang');
      // 城：场地之间一溜低楼和树
      put('mid', cityBand({ w: 1160, seed: 21 }), 580, 602, 66);
      mist('mid', 800, 588, 1800, 40);
      add('mid', `<rect x="-20" y="590" width="${W + 40}" height="${H - 590}" fill="url(#mc-ground)"/>`);
      put('mid', cityBand({ w: 900, seed: 33 }), 1040, 660, 60);
      // 国家会议中心（居中在前）、门前大巴与人群
      put('front', cncc(), 700, 692, 104, '', 'cncc');
      put('front', coach({ dir: 1 }), 470, 712, 34); put('front', coach({ dir: -1 }), 990, 712, 34);
      put('front', crowd(9, { seed: 4, dir: 1 }), 660, 716, 22); put('front', crowd(6, { seed: 7, dir: -1, pose: 'stand' }), 800, 716, 22);
      for (const x of [300, 1100]) put('front', Nt.cypress({ h: 200, seed: x % 13 }), x, 716, 132);
      // 两翼：左中关村、右丽泽
      put('fore', zgc(), 130, 770, 300, '', 'zgc');
      put('fore', lize(), 1478, 770, 300, '', 'lize');
      put('fore', Nt.cypress({ h: 200, seed: 23 }), 1352, 776, 150);
      slips = { richu: [28, 290], yanqi: [318, 286], shougang: [1086, 330], cncc: [372, 438], zgc: [282, 470], lize: [1528, 440] };
    } else {
      // 手机：四排自上而下，题签各放在所属场地上方或旁边的空处
      add('sky', `<rect width="${W}" height="${H}" fill="url(#mc-sky)"/>`);
      put('far', Nt.range({ w: 520, h: 110, seed: 17, fill: 'g-shan-far', peaks: [[-0.8, 0.9, 0.16], [-0.3, 1, 0.18], [0.3, 0.7, 0.18], [0.8, 0.86, 0.16]] }), 200, 250, 110, 'far');
      mist('far', 200, 268, 560, 46);
      put('cloud1', Nt.cloud2({ w: 150, h: 22, seed: 3 }), 110, 70, 30);
      // 雁栖湖
      put('back', pond({ w: 460, h: 48, seed: 3 }), 200, 300, 48);
      put('back', Nt.bank({ w: 170, h: 10, seed: 2, end: 0.2 }), 76, 270, 10);
      put('back', richu(), 76, 266, 52, '', 'richu');
      put('back', Nt.bank({ w: 260, h: 10, seed: 4, end: 0.18 }), 268, 274, 10);
      put('back', yanqiHall(), 268, 270, 72, '', 'yanqi');
      mist('mid', 200, 322, 520, 36);
      // 首钢园
      put('mid', pond({ w: 460, h: 50, seed: 7 }), 200, 452, 50);
      put('mid', Nt.bank({ w: 420, h: 10, seed: 6, end: 0.16 }), 190, 432, 10);
      put('mid', shougang(), 200, 428, 122, '', 'shougang');
      mist('mid', 200, 474, 520, 34);
      add('mid', `<rect x="-20" y="466" width="${W + 40}" height="${H - 466}" fill="url(#mc-ground)"/>`);
      put('mid', cityBand({ w: 420, seed: 21 }), 200, 500, 30, 'far');
      // 国家会议中心
      put('front', cncc({ w: 480 }), 200, 560, 66, '', 'cncc');
      put('front', crowd(8, { seed: 4 }), 150, 580, 14);
      // 中关村、丽泽
      // 两组楼之间：薄雾、一排柏、门前一辆大巴，让中间这块地不空
      mist('front', 200, 720, 520, 60);
      put('front', cityBand({ w: 420, seed: 33 }), 200, 704, 34, 'far');
      put('front', coach({ dir: -1 }), 330, 594, 20);
      put('fore', Nt.cypress({ h: 200, seed: 9 }), 212, 868, 104);
      put('fore', zgc(), 92, 866, 166, '', 'zgc');
      put('fore', lize(), 312, 866, 172, '', 'lize');
      slips = { richu: [62, 104], yanqi: [196, 92], shougang: [336, 312], cncc: [362, 440], zgc: [4, 598], lize: [252, 600] };
    }
    const order = ['sky', 'far', 'back', 'cloud1', 'mid', 'front', 'fore'];
    return { W, H, hits, slips, layers: order.map(id => ({ id, svg: `<svg class="jh-layer mc" viewBox="0 0 ${W} ${H}" preserveAspectRatio="xMidYMax slice" aria-hidden="true">${L[id] || ''}</svg>` })) };
  }

  window.JH.MICE = { defs, doorLeaf, yanqiHall, richu, shougang, cncc, zgc, lize, coach, crowd, pond, cityBand, vWork, vTrain, vForum, vGala, vLaunch, vExpo, vCadre, vBench, vSurvey, compose };
})();
