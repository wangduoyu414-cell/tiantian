/* 团队建设 · 《京郊山野图》：同一套界画墨线、青绿设色，画京郊能办团建的几处地方
   - 长城登山：远处山脊上的城墙，敌楼边一队人。
   - 京西古道：山间石板古道蜿蜒上山，过一座关城，导游举旗带队徒步。
   - 京郊农场：一层层梯田，田边几户灰瓦农舍，一队人在田里劳作。
   - 草地拓展：一片草坪，高空拓展架（木杆、平台、攀网、滑索），一圈人围成团队。
   - 沙燕风筝：天上几只北京沙燕风筝，线牵到草坪上放风筝的人手里。
   - 湖边厅堂：湖边一座歇山厅堂、一座亭子，垂柳，适合主题党日、读书会、团队共创。
   - 标杆企业：远处科技园区的玻璃楼，一辆大巴、一队参访的人。
   画面里不画旗帜上的字、徽记和标语。 */
(function () {
  'use strict';
  const J = window.JH, { N, P2, Pen, poly, smooth, samp, rng } = J, Nt = J.NATURE, A = J.ARCH;
  const rect = (x, y, w, h) => `M${P2(x, y)}h${N(w)}v${N(h)}h${N(-w)}Z`;

  /* ── 沙燕风筝：头、身、一对宽翅、燕尾；翅上彩绘（红、青、黄几块），墨线勾边 ── */
  function kite(o = {}) {
    const P = Pen(), c = o.c || 'r', tilt = o.tilt || 0;
    const R = ([x, y]) => [x * Math.cos(tilt) - y * Math.sin(tilt), x * Math.sin(tilt) + y * Math.cos(tilt)];
    const m = pts => poly(pts.map(R));
    // 翅：上沿弧、下沿内收
    for (const s of [-1, 1]) {
      const wing = samp(t => [s * (6 + 46 * t), 34 + 10 * Math.sin(Math.PI * t) - 30 * t * t], 14).concat(samp(t => [s * (52 - 44 * t), 4 - 18 * Math.sin(Math.PI * t * 0.9) + 12 * t], 14));
      P.p(m(wing), 'c-kite-w l2');
      P.p(m(samp(t => [s * (14 + 30 * t), 26 + 6 * Math.sin(Math.PI * t) - 20 * t * t], 10).concat(samp(t => [s * (44 - 30 * t), 10 - 8 * Math.sin(Math.PI * t) + 6 * t], 10))), `c-kite-${c} l3`);
      P.p(m([[s * 22, 18], [s * 30, 24], [s * 36, 16], [s * 28, 10]]), 'c-kite-y l3');
    }
    // 燕尾
    P.p(m([[-5, -6], [-14, -44], [-6, -40], [0, -18], [6, -40], [14, -44], [5, -6]]), `c-kite-${c} l2`);
    // 身与头
    P.p(m(samp(t => { const a = Math.PI * 2 * t; return [Math.sin(a) * 7, 12 + Math.cos(a) * 22]; }, 24)), 'c-kite-w l2');
    P.p(m(samp(t => { const a = Math.PI * 2 * t; return [Math.sin(a) * 6.5, 38 + Math.cos(a) * 7]; }, 18)), 'c-kite-k l2');
    for (const s of [-1, 1]) { const [x, y] = R([s * 2.6, 39]); P.c(x, y, 1.6, 'c-kite-w'); }
    P.p(m([[-4, 18], [4, 18], [3, 6], [-3, 6]]), `c-kite-${c} l3`);
    return { g: P.s(), box: [-56, 56, -46, 48] };
  }

  /* ── 高空拓展架：两根木杆加斜撑，两层平台，杆间攀网，一条滑索斜拉到地面 ── */
  function ropes(o = {}) {
    const P = Pen(), H = o.h || 200, W = o.w || 120;
    // 滑索
    P.p(`M${P2(W / 2 + 4, H * 0.92)}L${P2(W * 2.2, 4)}`, 'l-cable');
    P.r(W * 2.2 - 6, 0, 12, 10, 'c-wood l3');
    // 攀网
    let net = ''; for (let k = 0; k <= 6; k++) { const x = -W / 2 + W * k / 6; net += `M${P2(x, H * 0.12)}L${P2(x, H * 0.62)}`; } for (let k = 0; k <= 7; k++) { const y = H * 0.12 + H * 0.5 * k / 7; net += `M${P2(-W / 2, y)}L${P2(W / 2, y)}`; }
    P.p(net, 'l-net');
    // 斜撑与木杆
    let br = ''; for (const s of [-1, 1]) br += `M${P2(s * W / 2, H * 0.4)}L${P2(s * (W / 2 + 36), 0)}`; P.p(br, 'l-pole2');
    for (const s of [-1, 1]) { P.r(s * W / 2 - 5, 0, 10, H, 'g-bark l2'); P.r(s * W / 2 - 7, H - 6, 14, 6, 'c-wood-d l3'); }
    // 两层平台、栏杆
    for (const y of [H * 0.62, H * 0.9]) { P.r(-W / 2 - 14, y, W + 28, 6, 'c-wood l2'); let rl = ''; for (let x = -W / 2 - 12; x <= W / 2 + 12; x += 10) rl += `M${P2(x, y + 6)}v10`; P.p(rl + `M${P2(-W / 2 - 14, y + 16)}H${N(W / 2 + 14)}`, 'l3'); }
    // 网上攀爬的人、平台上的人
    const climber = Nt.person({ h: 34, c: 'o', pose: 'point', dir: 1 }); P.g(climber.g, ` transform="translate(${N(-W * 0.12)} ${N(H * 0.3)})"`);
    const top = Nt.person({ h: 32, c: 'b', pose: 'clasp', dir: -1 }); P.g(top.g, ` transform="translate(${N(W * 0.2)} ${N(H * 0.9 + 6)})"`);
    return { g: P.s(), box: [-W / 2 - 40, W * 2.2 + 8, -2, H + 18] };
  }

  /* ── 梯田：顺山坡一层层弧形的田。每层的田面中间高、两头顺坡往下收，前沿一道土坎；田里一垄垄庄稼。 ── */
  function terraces(o = {}) {
    const P = Pen(), r = rng(o.seed || 7), w = o.w || 360, n = o.n || 5, hh = o.h || 140, step = hh / n;
    for (let i = n - 1; i >= 0; i--) {
      const k = i / n, ww = w * (1 - k * 0.62), cx = (r() - 0.5) * w * 0.06 + k * w * 0.06, x0 = cx - ww / 2, x1 = cx + ww / 2;
      const y0 = i * step, sag = step * (0.9 + 0.5 * (1 - k));       // 两头往下收的深度
      const arc = (dy, t) => y0 + dy - sag * Math.pow(Math.abs(2 * t - 1), 2.4);
      const topF = samp(t => [x0 + (x1 - x0) * t, arc(step * 1.05, t)], 30);
      const lip = samp(t => [x1 - (x1 - x0) * t, arc(step * 0.62, 1 - t)], 30);
      const foot = samp(t => [x0 + (x1 - x0) * t, arc(0, t)], 30);
      P.p(smooth(topF) + 'L' + smooth(lip).slice(1) + 'Z', `${i % 2 ? 'c-field2' : 'c-field'} l2`);
      P.p(smooth(lip.slice().reverse()) + 'L' + smooth(foot.slice().reverse()).slice(1) + 'Z', 'c-bank-f l3');
      let rows = ''; const m = Math.max(6, Math.round(ww / 12));
      for (let j = 1; j < m; j++) { const t = j / m, x = x0 + (x1 - x0) * t, ya = arc(step * 0.7, t), yb = arc(step * 1.0, t); rows += `M${P2(x, ya)}L${P2(x + (t - 0.5) * 6, yb)}`; }
      P.p(rows, 'l-crop');
    }
    return { g: P.s(), box: [-w / 2 - 10, w / 2 + 10, -step * 1.6, hh + step * 0.4] };
  }

  /* ── 农舍：几间灰瓦硬山小房，院墙，门窗，房后一棵树 ── */
  function farmhouses(o = {}) {
    const P = Pen(), r = rng(o.seed || 17), n = o.n || 3, Y = J.RED.yingshan;
    let x = 0; const houses = [];
    for (let i = 0; i < n; i++) { const w = 54 + r() * 20, h = 22 + r() * 4; houses.push([x + w / 2, w, h, i % 2 ? 6 : 0]); x += w + 10 + r() * 8; }
    const off = x / 2;
    for (const [cx0, w, h, dy] of houses) {
      const cx = cx0 - off, R2 = 14 + r() * 6;
      if (r() < 0.6) P.p(Nt.clumpPath(cx + w * 0.4, dy + h + 14 + R2 * 0.3, R2, R2 * 0.72, r, 8), 'c-tree l3');
      P.r(cx - w / 2, dy, w, h, 'c-wall-g l2');
      P.r(cx - 5, dy, 10, h * 0.66, 'c-wood l3');
      for (const s of [-1, 1]) P.r(cx + s * w * 0.28 - 5, dy + h * 0.36, 10, 8, 'p-lattice2 l3');
      Y(P, cx, dy + h, w + 6, 12, false);
    }
    return { g: P.s(), box: [-off - 12, off + 12, -2, 60] };
  }

  /* ── 古道与关城：一条石板路从山脚蜿蜒上山，半山一座关城（城台、券门、硬山小楼） ── */
  function trail(o = {}) {
    const P = Pen(), H = o.h || 240;
    const sp = samp(t => [Math.sin(t * Math.PI * 2.2) * 60 * (1 - t * 0.5), t * H], 40);
    const L = [], R = [];
    for (let i = 0; i < sp.length; i++) { const w = 14 * (1 - i / sp.length * 0.6); L.push([sp[i][0] - w, sp[i][1]]); R.push([sp[i][0] + w, sp[i][1]]); }
    P.p(smooth(L) + 'L' + smooth(R.reverse()).slice(1) + 'Z', 'c-stone l2');
    let st = ''; for (let i = 1; i < sp.length - 1; i += 2) { const [x, y] = sp[i], w = 13 * (1 - i / sp.length * 0.6); st += `M${P2(x - w, y)}L${P2(x + w, y + 1)}`; }
    P.p(st, 'l3');
    // 关城
    const gx = sp[26][0], gy = sp[26][1] - 8;
    P.p(poly([[gx - 34, gy], [gx + 34, gy], [gx + 31, gy + 30], [gx - 31, gy + 30]]), 'c-gw l2');
    P.p(`M${P2(gx - 9, gy)}V${N(gy + 15)}A9 9 0 0 0 ${P2(gx + 9, gy + 15)}V${N(gy)}Z`, 'c-ink');
    let mer = ''; for (let x = gx - 30; x < gx + 28; x += 8) mer += rect(x, gy + 30, 5, 5); P.p(mer, 'c-gw l3');
    P.r(gx - 16, gy + 30, 32, 12, 'c-gw l2');
    J.RED.yingshan(P, gx, gy + 42, 40, 11, false);
    return { g: P.s(), box: [-80, 80, -2, H + 10], at: t => sp[Math.round(t * (sp.length - 1))] };
  }

  /* ── 一队人：举旗的导游在前，后面跟几位背包的 ── */
  function team(n, o = {}) {
    const P = Pen(), r = rng(o.seed || 5), cs = o.colors || ['r', 'b', 'o', 'g', 'b', 'r', 'o', 'g'], gap = o.gap || 15;
    for (let i = 0; i < n; i++) {
      const g = i === 0 && o.guide, Lp = Nt.person({ h: 40, c: g ? 'o' : cs[i % cs.length], pose: g ? 'flag' : (o.pose || 'walk'), dir: o.dir || 1, hair: r() < 0.35 ? 'long' : 'short', bag: !g && r() < 0.6 });
      P.g(Lp.g, ` transform="translate(${N(-i * gap * (o.dir || 1))} ${N((i % 2) * 2 + (o.rise || 0) * i)})"`);
    }
    return { g: P.s(), box: (o.dir || 1) > 0 ? [-(n - 1) * gap - 16, 16, 0, 52 + Math.max(0, (o.rise || 0) * n)] : [-16, (n - 1) * gap + 16, 0, 52 + Math.max(0, (o.rise || 0) * n)] };
  }
  // 围成一圈的人（团队共创、破冰）：前排背影、后排正面
  function circle(n, o = {}) {
    const P = Pen(), r = rng(o.seed || 9), cs = ['r', 'b', 'o', 'g', 'w', 'b', 'r', 'o', 'g', 'k'], R = o.r || 60;
    const pos = Array.from({ length: n }, (_, i) => { const a = Math.PI * 2 * i / n + 0.3; return [Math.cos(a) * R, Math.sin(a) * R * 0.32, a]; }).sort((a, b) => b[1] - a[1]);
    for (const [x, y, a] of pos) { const front = Math.sin(a) < 0; const Lp = Nt.person({ h: 40, c: cs[Math.floor(r() * cs.length)], pose: front ? 'stand' : 'clasp', back: front, dir: x > 0 ? -1 : 1, hair: r() < 0.3 ? 'long' : 'short' }); P.g(Lp.g, ` transform="translate(${N(x)} ${N(y)})"`); }
    return { g: P.s(), box: [-R - 16, R + 16, -R * 0.34, R * 0.34 + 42] };
  }

  /* ── 主视觉 ── */
  function compose(layout = 'desk', px = 1) {
    const D = layout === 'desk', W = D ? 1600 : 400, H = D ? 760 : 900, L = {}, hits = [];
    const add = (id, s) => { L[id] = (L[id] || '') + s; };
    const put = (id, el, cx, bottom, h, cls = '', hit) => { const s = h / (el.box[3] - el.box[2]); add(id, J.place(el, cx, bottom + el.box[2] * s, h, px, cls)); if (hit) hits.push({ id: hit, x: cx + el.box[0] * s, y: bottom - h, w: (el.box[1] - el.box[0]) * s, h }); return s; };
    const mist = (id, cx, bottom, w, h) => put(id, Nt.mistBlob({ w, h }), cx, bottom, h);
    const kiteAt = (id, x, y, h, c, tilt, hx, hy) => { add(id, `<g class="jh"><path d="M${N(x)} ${N(y - h * 0.4)}Q${N((x + hx) / 2 + 40)} ${N((y + hy) / 2 + 20)} ${N(hx)} ${N(hy)}" class="l-string" style="stroke-width:${N(0.8 / px)}px"/></g>`); put(id, kite({ c, tilt }), x, y, h); };
    const lawn = (d, y0, path) => {
      const r = rng(31); let tufts = '', flowers = '';
      for (let i = 0; i < (D ? 260 : 120); i++) {
        const x = r() * W, y = y0 + 12 + Math.pow(r(), 0.7) * (H - y0 - 12), k = (y - y0) / (H - y0), s = 2.2 + k * 5;
        tufts += `M${N(x - s)} ${N(y)}l${N(s * 0.6)} ${N(-s * 1.1)}l${N(s * 0.4)} ${N(s * 1.1)}l${N(s * 0.5)} ${N(-s * 0.9)}l${N(s * 0.5)} ${N(s * 0.9)}`;
        if (r() < 0.12) flowers += `M${N(x + s * 2)} ${N(y - s * 0.4)}h.01`;
      }
      add('near', `<g class="jh"><path d="${d}" class="g-lawn"/>${path ? `<path d="${path}" class="c-lawnpath"/>` : ''}<path d="${tufts}" class="l-tuft" style="stroke-width:${N(0.7 / px)}px"/><path d="${flowers}" class="l-flower" style="stroke-width:${N(3.2 / px)}px"/></g>`);
    };
    let slips;
    add('sky', `<rect width="${W}" height="${H}" fill="url(#jh-sky)"/><ellipse cx="${D ? 900 : 220}" cy="${D ? 300 : 260}" rx="${D ? 620 : 240}" ry="${D ? 220 : 180}" fill="url(#jh-glow)"/>`);
    if (D) {
      // 远：燕山与长城
      const yan = Nt.range({ w: 1700, h: 230, seed: 23, fill: 'g-shan-far', peaks: [[-0.92, 0.7, 0.08], [-0.7, 1, 0.08], [-0.48, 0.82, 0.08], [-0.24, 0.5, 0.1], [0.04, 0.42, 0.12], [0.34, 0.56, 0.1], [0.62, 0.48, 0.1], [0.88, 0.62, 0.09]] });
      const gw = J.LMX2.greatwall(yan.pts.slice(3, 40), { h: 7, towers: [0.08, 0.22, 0.38, 0.55, 0.72, 0.9], huts: [0.22, 0.72] });
      put('far', { g: yan.g + gw.g, box: yan.box }, 800, 420, 230, 'far', 'wall');
      put('far', team(5, { seed: 3, gap: 9 }), 286, 262, 11, 'far');
      add('far', `<rect x="-20" y="360" width="${W + 40}" height="300" fill="url(#jh-hazeb)"/>`);
      mist('far', 800, 470, 1800, 110);
      // 天上的沙燕风筝
      kiteAt('sky2', 1060, 150, 62, 'r', -0.18, 1010, 560); kiteAt('sky2', 1186, 108, 50, 'b', 0.12, 1048, 566); kiteAt('sky2', 930, 214, 40, 'g', -0.08, 990, 566);
      hits.push({ id: 'kite', x: 880, y: 60, w: 360, h: 190 });
      put('sky2', Nt.cloud2({ w: 300, h: 34, seed: 3 }), 520, 200, 46);
      put('sky2', Nt.cloud2({ w: 260, h: 30, seed: 7, dir: -1 }), 1420, 238, 40);
      // 中远：西山与京西古道
      put('mid', Nt.mountain({ w: 520, h: 300, seed: 41, peaks: [[-0.3, 1, 0.3], [0.24, 0.76, 0.3], [0.62, 0.5, 0.26]], fill: 'g-shan', fans: false, trees: true }), 250, 600, 320);
      const tr = trail({ h: 260 }), ts = put('mid', tr, 230, 600, 270, '', 'trail');
      put('mid', team(6, { guide: true, seed: 5, gap: 13, dir: 1, rise: 1.5 }), 248, 560, 22);
      // 中：梯田与农舍
      put('mid', Nt.hill({ w: 420, h: 110, seed: 13, fill: 'g-shan-mid', density: 0.7 }), 650, 560, 116);
      put('mid', farmhouses({ n: 3, seed: 17 }), 742, 470, 42);
      put('mid', terraces({ w: 420, n: 5, h: 130, seed: 7 }), 650, 606, 168, '', 'farm');
      put('mid', team(4, { seed: 11, gap: 22, pose: 'stand', dir: -1 }), 640, 548, 16);
      mist('mid', 700, 640, 1500, 60);
      // 近：草坪、拓展架、放风筝的人
      lawn(`M-20 ${N(H)}V610Q300 560 640 590Q900 610 1180 576Q1400 556 1620 580V${N(H)}Z`, 580, 'M560 772C640 720 760 700 880 690C1000 680 1120 676 1210 664L1226 668C1130 684 1010 694 890 704C780 714 680 738 620 772Z');
      put('near', ropes({ h: 220, w: 120 }), 900, 640, 170, '', 'meadow');
      put('near', circle(9, { r: 70, seed: 4 }), 1060, 676, 50);
      put('near', team(3, { seed: 13, gap: 26, pose: 'point', dir: 1 }), 1032, 600, 22);
      // 湖边厅堂、亭子、垂柳
      put('near', J.MICE.pond({ w: 640, h: 96, seed: 5, cls: 'g-water' }), 1470, 704, 96);
      put('near', Nt.waves({ w: 480, h: 80, s: 0.8, seed: 3 }), 1440, 700, 80);
      const hall = J.withLod(0.6, () => J.BUILD.hall({ bays: [64, 56, 50], colH: 62, platform: 14, rail: 8, type: 'xs', tone: 'gray', trim: 'green', tile: 4.4, beasts: 3, per: 2, rw: 80, caihua: 'xuanzi' }));
      put('near', hall, 1340, 606, 104, '', 'hall');
      for (const [x, b, h, sd] of [[1210, 640, 116, 7], [1590, 650, 110, 9]]) put('near', Nt.willow({ h: 200, seed: sd, density: 0.5 }), x, b, h);
      // 远处的科技园区与参访的人
      put('mid', J.MICE.zgc(), 1470, 520, 120, 'far', 'park');
      put('mid', Nt.grove({ w: 260, h: 50, seed: 21, kind: 'mixed', trunks: false }), 1470, 528, 40);
      mist('mid', 1470, 560, 520, 70);
      put('near', J.MICE.coach({ dir: -1 }), 1150, 664, 26);
      // 前景：两侧的松与坡
      put('fore', Nt.pine({ h: 260, seed: 4 }), 40, 774, 230);
      put('fore', Nt.rock({ w: 200, h: 110, seed: 19 }), 92, 774, 56);
      put('fore', Nt.dianye({ h: 190, seed: 11 }), 1570, 774, 150);
      slips = { wall: [430, 186], trail: [40, 330], farm: [828, 452], meadow: [760, 476], kite: [1248, 76], hall: [1236, 446], park: [1560, 330] };
    } else {
      const yan = Nt.range({ w: 520, h: 120, seed: 23, fill: 'g-shan-far', peaks: [[-0.8, 0.9, 0.16], [-0.3, 1, 0.18], [0.3, 0.7, 0.18], [0.8, 0.86, 0.16]] });
      const gw = J.LMX2.greatwall(yan.pts.slice(2, -2), { h: 5, towers: [0.12, 0.34, 0.56, 0.8], huts: [0.34] });
      put('far', { g: yan.g + gw.g, box: yan.box }, 200, 230, 120, 'far', 'wall');
      add('far', `<rect x="-20" y="190" width="${W + 40}" height="300" fill="url(#jh-hazeb)"/>`);
      mist('far', 200, 262, 560, 70);
      kiteAt('sky2', 300, 96, 44, 'r', -0.16, 250, 600); kiteAt('sky2', 360, 150, 32, 'b', 0.1, 262, 604);
      hits.push({ id: 'kite', x: 260, y: 50, w: 130, h: 110 });
      // 古道（左）与梯田（右）
      put('mid', Nt.mountain({ w: 260, h: 200, seed: 41, peaks: [[-0.2, 1, 0.36], [0.36, 0.7, 0.3]], fill: 'g-shan', fans: false, trees: true }), 100, 420, 200);
      put('mid', trail({ h: 260 }), 100, 420, 180, '', 'trail');
      put('mid', team(5, { guide: true, seed: 5, gap: 13, rise: 1.5 }), 112, 394, 16);
      put('mid', farmhouses({ n: 2, seed: 17 }), 318, 330, 30);
      put('mid', terraces({ w: 250, n: 5, h: 120, seed: 7 }), 296, 426, 124, '', 'farm');
      mist('mid', 200, 450, 520, 44);
      // 草坪、拓展架、一圈人
      lawn(`M-20 ${N(H)}V418Q120 400 220 414Q320 430 420 408V${N(H)}Z`, 412, 'M120 760C150 720 200 690 250 660C290 636 330 620 360 606L368 612C340 628 300 646 262 670C214 700 170 730 150 760Z');
      put('near', ropes({ h: 220, w: 120 }), 110, 600, 150, '', 'meadow');
      put('near', circle(8, { r: 60, seed: 4 }), 284, 616, 40);
      put('near', team(2, { seed: 13, gap: 26, pose: 'point' }), 262, 520, 18);
      // 湖边厅堂与园区
      put('near', J.MICE.zgc(), 320, 690, 92, 'far', 'park');
      put('near', Nt.grove({ w: 180, h: 40, seed: 21, kind: 'mixed', trunks: false }), 320, 698, 30);
      put('near', Nt.lake({ x0: -20, x1: 420, h: 150, seed: 5, cls: 'g-water' }), 0, 906, 150);
      put('near', Nt.waves({ w: 420, h: 130, s: 0.6, seed: 3 }), 200, 900, 130);
      const hall = J.withLod(0.6, () => J.BUILD.hall({ bays: [64, 56, 50], colH: 62, platform: 14, rail: 8, type: 'xs', tone: 'gray', trim: 'green', tile: 4.4, beasts: 3, per: 2, rw: 80, caihua: 'xuanzi' }));
      put('near', hall, 150, 772, 76, '', 'hall');
      put('near', Nt.willow({ h: 200, seed: 7, density: 0.5 }), 30, 790, 96);
      put('fore', Nt.dianye({ h: 190, seed: 11 }), 380, 904, 120);
      slips = { wall: [14, 120], trail: [168, 276], farm: [350, 300], meadow: [6, 480], kite: [218, 50], hall: [256, 710], park: [370, 560] };
    }
    const order = ['sky', 'far', 'sky2', 'mid', 'near', 'fore'];
    return { W, H, hits, slips, layers: order.map(id => ({ id, svg: `<svg class="jh-layer" viewBox="0 0 ${W} ${H}" preserveAspectRatio="xMidYMax slice" aria-hidden="true">${L[id] || ''}</svg>` })) };
  }

  window.JH.TEAM = { kite, ropes, terraces, farmhouses, trail, team, circle, compose };
})();
