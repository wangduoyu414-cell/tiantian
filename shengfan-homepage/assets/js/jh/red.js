/* 红色教育 · 朱墨界画
   设色收到朱砂、赭石、墨、泥金；山用浅绛（赭石打底、墨青罩顶），青绿压到最低；背景一抹晨光暖色。不画任何旗帜和徽记。
   地标（外观依据公开资料）：
   - 北大红楼：红砖红瓦，地上四层、半地下一层，坐北朝南，正面是一道长立面。
   - 李大钊故居：文华胡同里灰砖门楼的小院，院里海棠。
   - 卢沟桥：十一孔联拱石桥，望柱上石狮；宛平城在桥东头扼桥而立，城墙上留着炮击的弹痕。
   - 香山双清别墅：院中池旁一座六角红亭，正房展室，山石上"双清"刻石。
   - 中国共产党历史展览馆：传统柱廊式，东西两面各 28 根廊柱。
   画面里一律只画建筑、山水和人，不画旗、徽、标语。 */
(function () {
  'use strict';
  const J = window.JH, { N, P2, Pen, poly, smooth, samp, rng } = J, A = J.ARCH, Nt = J.NATURE;

  /* ── 朱墨设色的渐变（页面里放一次） ── */
  function defs() {
    const lin = (id, stops, x1 = 0, y1 = 0, x2 = 0, y2 = 1) => `<linearGradient id="${id}" x1="${x1}" y1="${y1}" x2="${x2}" y2="${y2}">${stops.map(([o, c, a = 1]) => `<stop offset="${o}" stop-color="${c}"${a < 1 ? ` stop-opacity="${a}"` : ''}/>`).join('')}</linearGradient>`;
    return '<svg width="0" height="0" style="position:absolute" aria-hidden="true"><defs>' +
      lin('zm-shan', [[0, '#dcbc92'], [.45, '#c5a37d'], [1, '#7c7265']]) +
      lin('zm-mid', [[0, '#e6cfac'], [.5, '#cfb391'], [1, '#9d9181']]) +
      lin('zm-far', [[0, '#efdfc6'], [1, '#c9bdaa']]) +
      lin('zm-water', [[0, '#e3d6c1'], [1, '#efe5d4']]) +
      lin('zm-sky', [[0, '#efe4d1'], [.55, '#f4d9b6'], [1, '#f2e6d4']]) +
      lin('zm-tile', [[0, '#8e3a28'], [.55, '#ab4a33'], [1, '#c26248']]) +
      lin('zm-brick', [[0, '#c26c4c'], [.5, '#a7513a'], [1, '#8a3e2b']], 0, 0, 1, 0) +
      lin('zm-rocktop', [[0, '#c9a27a', 0], [.4, '#a68c70', .9], [1, '#6f6a62']]) +
      lin('zm-ground', [[0, '#eadfcb', 0], [.35, '#e8dcc6', .8], [1, '#e2d4bb']]) +
      lin('zm-mist', [[0, '#f4ead9', 0], [.5, '#f6eddd', .92], [1, '#f4ead9', 0]]) +
      '<radialGradient id="zm-glow"><stop offset="0" stop-color="#f4bd84" stop-opacity=".6"/><stop offset=".55" stop-color="#f4c999" stop-opacity=".22"/><stop offset="1" stop-color="#f4c999" stop-opacity="0"/></radialGradient>' +
      '<radialGradient id="zm-lamp"><stop offset="0" stop-color="#ffd98a" stop-opacity=".9"/><stop offset="1" stop-color="#ffd98a" stop-opacity="0"/></radialGradient>' +
      '</defs></svg>';
  }

  const rect = (x, y, w, h) => `M${P2(x, y)}h${N(w)}v${N(h)}h${N(-w)}Z`;

  /* ── 一排西式窗：白框、玻璃、中梃与横档、窗台、窗上砖砌弧拱 ── */
  function windows(P, xs, y0, w, h, o = {}) {
    let fr = '', gl = '', mu = '', si = '', ar = '';
    for (const x of xs) {
      fr += rect(x - w / 2, y0, w, h);
      const m = w * 0.14; gl += rect(x - w / 2 + m, y0 + m, w - 2 * m, h - 2 * m);
      mu += `M${P2(x, y0 + m)}V${N(y0 + h - m)}M${P2(x - w / 2 + m, y0 + h * 0.64)}H${N(x + w / 2 - m)}`;
      si += rect(x - w / 2 - 1.4, y0 - 1.8, w + 2.8, 1.8);
      if (o.arch !== false) ar += `M${P2(x - w / 2 - 1.6, y0 + h)}Q${P2(x, y0 + h + 3.2)} ${P2(x + w / 2 + 1.6, y0 + h)}L${P2(x + w / 2 + 1.6, y0 + h + 2.6)}Q${P2(x, y0 + h + 5.8)} ${P2(x - w / 2 - 1.6, y0 + h + 2.6)}Z`;
    }
    P.p(fr, 'c-trim l3'); P.p(gl, 'c-pane'); P.p(mu, 'l-mull'); P.p(si, 'c-trim l3'); if (ar) P.p(ar, 'c-brick-d l3');
  }

  /* ── 北大红楼：红砖墙、红瓦四坡顶、白色檐口与腰线、墙角隅石；四层窗户一排排对齐，正中入口带石框拱门和台阶 ── */
  function honglou(o = {}) {
    const P = Pen(), W = o.w ?? 700, hw = W / 2, base = 16, fl = 36, nF = 4, top = base + fl * nF, cor = 8, rh = 46, ov = 10, cb = 48, bay = 27;
    const eY = top + cor, rTop = eY + rh, run = rh * 1.05, rx = hw + ov - run;
    // 屋顶：红瓦，四坡直顶；瓦垄竖向，斜脊处变短；左半受光
    P.p(poly([[-hw - ov, eY], [hw + ov, eY], [rx, rTop], [-rx, rTop]]), 'g-tile l1');
    let t = ''; for (let x = -hw - ov + 5; x < hw + ov - 2; x += 5.6) { const a = Math.abs(x), yt = a <= rx ? rTop - 1.5 : eY + (hw + ov - a) * rh / run; if (yt - eY > 4) t += `M${P2(x, eY + 2.6)}V${N(yt - 1)}`; }
    P.p(t, 'l-tile');
    P.p(`M${P2(-hw - ov, eY)}L${P2(-rx, rTop)}M${P2(hw + ov, eY)}L${P2(rx, rTop)}`, 'l2');
    P.r(-rx - 1, rTop - 2.6, rx * 2 + 2, 4.6, 'c-tile-d l3');
    // 檐口：白色，下面一排小齿
    P.r(-hw - 5, top, W + 10, cor, 'c-trim l2'); P.p(`M${P2(-hw - 5, top + cor * 0.55)}H${N(hw + 5)}`, 'l3');
    let dent = ''; for (let x = -hw - 3; x < hw + 3; x += 4.2) dent += rect(x, top + 0.6, 2, cor * 0.4); P.p(dent, 'c-trim-d');
    // 墙身：红砖，细砖缝
    P.r(-hw, base, W, fl * nF, 'g-brick l1');
    let bk = ''; for (let y = base + 3.4; y < top - 1; y += 3.4) bk += `M${P2(-hw, y)}H${N(hw)}`; P.p(bk, 'l-brick');
    // 墙角隅石
    let q = ''; for (let y = base + 3, k = 0; y < top - 4; y += 7, k++) for (const s of [-1, 1]) q += rect(s > 0 ? hw - (k % 2 ? 9 : 14) : -hw, y, k % 2 ? 9 : 14, 4.6);
    P.p(q, 'c-trim l3');
    // 腰线：首层上下两道白石带
    for (const y of [base, base + fl - 1.2]) P.r(-hw - 2, y, W + 4, 3, 'c-trim l3');
    // 中间入口一间略凸出：右边一道阴影
    P.r(-cb / 2, base, cb, fl * nF + 1, 'g-brick l2');
    let bk2 = ''; for (let y = base + 3.4; y < top - 1; y += 3.4) bk2 += `M${P2(-cb / 2, y)}H${N(cb / 2)}`; P.p(bk2, 'l-brick');
    P.r(cb / 2, base, 3.4, fl * nF, 'c-shade');
    for (const y of [base, base + fl - 1.2]) P.r(-cb / 2 - 2, y, cb + 4, 3, 'c-trim l3');
    // 窗：每层每间一扇
    const xs = []; for (let x = cb / 2 + bay / 2 + 2; x < hw - bay * 0.45; x += bay) xs.push(x, -x);
    for (let f = 0; f < nF; f++) windows(P, xs, base + f * fl + 9, 12, 21);
    for (let f = 1; f < nF; f++) windows(P, [-9, 9], base + f * fl + 9, 9, 21);
    // 半地下：石基，一排小窗
    P.r(-hw - 3, 0, W + 6, base, 'c-stone l2');
    let bw = ''; for (const x of xs) bw += rect(x - 4.5, 4, 9, 7); P.p(bw, 'c-pane l3');
    // 入口：石框拱门，门上一块素面石匾，门前台阶
    const dw = 22, dh = 34;
    P.p(`M${P2(-dw / 2 - 4.5, base)}V${N(base + dh - dw / 2)}A${N(dw / 2 + 4.5)} ${N(dw / 2 + 4.5)} 0 0 0 ${P2(dw / 2 + 4.5, base + dh - dw / 2)}V${N(base)}Z`, 'c-trim l2');
    P.p(`M${P2(-dw / 2, base)}V${N(base + dh - dw / 2)}A${N(dw / 2)} ${N(dw / 2)} 0 0 0 ${P2(dw / 2, base + dh - dw / 2)}V${N(base)}Z`, 'c-door l2');
    P.p(`M0 ${N(base)}V${N(base + dh - 3)}M${P2(-dw / 2, base + dh - dw / 2)}H${N(dw / 2)}`, 'l3');
    for (let k = 0; k < 4; k++) P.r(-dw / 2 - 14 + k * 2.5, k * 4, dw + 28 - k * 5, 4, 'c-stone l3');
    return { g: P.s(), box: [-hw - ov - 6, hw + ov + 6, -2, rTop + 4] };
  }

  /* ── 硬山门楼的屋顶：正面看是一道瓦面，两端山墙收住；清水脊两头蝎子尾上翘；檐下墀头 ── */
  function yingshan(P, cx, y, w, h, tails = true) {
    const hw = w / 2, top = y + h;
    P.p(poly([[cx - hw, y], [cx + hw, y], [cx + hw - 2, top], [cx - hw + 2, top]]), 'g-gray l2');
    let t = ''; for (let x = cx - hw + 3; x < cx + hw - 2; x += 3.6) t += `M${P2(x, y + 3)}V${N(top - 0.8)}`; P.p(t, 'l3');
    P.r(cx - hw - 1, y - 1.5, w + 2, 3.6, 'c-gray-d l3');
    let gh = ''; for (let x = cx - hw + 1.8; x < cx + hw; x += 3.6) gh += `M${P2(x - 1.2, y - 0.2)}a1.2 1.2 0 1 0 2.4 0a1.2 1.2 0 1 0 -2.4 0Z`; P.p(gh, 'c-gray-l l3');
    P.r(cx - hw + 3, top, w - 6, 4.6, 'c-gray-d l2');
    if (tails) for (const s of [-1, 1]) P.p(`M${P2(cx + s * (hw - 4), top + 1)}C${P2(cx + s * (hw + 1), top + 2)} ${P2(cx + s * (hw + 4), top + 6)} ${P2(cx + s * (hw + 6), top + 12)}L${P2(cx + s * (hw + 4), top + 12.6)}C${P2(cx + s * (hw + 2), top + 8)} ${P2(cx + s * (hw - 1), top + 5.4)} ${P2(cx + s * (hw - 4), top + 4.6)}Z`, 'c-gray-d l3');
    for (const s of [-1, 1]) P.r(cx + s * (hw - 3) - 3, y - 9, 6, 8, 'c-gw l3');
  }

  /* ── 李大钊故居：胡同里一道灰砖院墙，灰砖门楼、红门、门簪、抱鼓石，门旁挂一块木牌；墙后探出一树海棠 ── */
  function lidazhao(o = {}) {
    const P = Pen(), W = o.w ?? 260, hw = W / 2, wh = 46, gx = o.gx ?? -34, gw = 56, gh = 62, r = rng(9);
    // 海棠
    const tx = gx + 96;
    P.p(Nt.stroke2([[tx, 30], [tx + 3, 62], [tx - 4, 92]], 4.4, 2.2), 'g-bark l2');
    P.p(Nt.stroke2([[tx + 2, 64], [tx + 18, 84], [tx + 30, 92]], 2.2, 1), 'g-bark l3');
    P.p(Nt.stroke2([[tx - 1, 72], [tx - 18, 88], [tx - 30, 94]], 2.2, 1), 'g-bark l3');
    for (const [x, y, R] of [[tx - 30, 98, 24], [tx + 32, 96, 22], [tx + 2, 108, 28], [tx - 12, 88, 20], [tx + 18, 84, 20]]) {
      P.p(Nt.clumpPath(x, y, R, R * 0.72, r, 9), 'c-haitang l2');
      P.p(`M${P2(x - R * 0.55, y + R * 0.12)}Q${P2(x - R * 0.12, y + R * 0.66)} ${P2(x + R * 0.46, y + R * 0.3)}Q${P2(x, y + R * 0.34)} ${P2(x - R * 0.55, y + R * 0.12)}Z`, 'c-light');
    }
    let bl = '', bl2 = ''; for (let i = 0; i < 90; i++) { const x = tx - 50 + r() * 100, y = 72 + r() * 52, s = 1.3 + r() * 0.9; const d = `M${P2(x - s, y)}a${N(s)} ${N(s)} 0 1 0 ${N(s * 2)} 0a${N(s)} ${N(s)} 0 1 0 ${N(-s * 2)} 0Z`; if (r() < 0.6) bl += d; else bl2 += d; }
    P.p(bl, 'c-blossom'); P.p(bl2, 'c-blossom2');
    // 院墙：灰砖、墙帽
    P.r(-hw, 0, W, wh, 'c-gw l1');
    let bk = ''; for (let y = 3.4; y < wh; y += 3.4) bk += `M${P2(-hw, y)}H${N(hw)}`; P.p(bk, 'l-brick-g');
    P.p(poly([[-hw - 3, wh], [hw + 3, wh], [hw + 1, wh + 4.4], [-hw - 1, wh + 4.4]]), 'c-gray-d l2');
    let cp = ''; for (let x = -hw; x < hw; x += 4) cp += `M${P2(x, wh + 4.4)}a2 1.6 0 0 1 4 0`; P.p(cp, 'l3');
    // 门楼
    P.r(gx - gw / 2, 0, gw, gh, 'c-gw l1');
    let bk2 = ''; for (let y = 3.4; y < gh; y += 3.4) bk2 += `M${P2(gx - gw / 2, y)}H${N(gx + gw / 2)}`; P.p(bk2, 'l-brick-g');
    P.r(gx - 16, 2, 32, 41, 'c-ink');
    P.r(gx - 15, 2, 30, 40, 'c-door l2'); P.p(`M${P2(gx, 2)}V42`, 'l2');
    P.p(`M${P2(gx - 13, 6)}V38H${N(gx - 2)}V6ZM${P2(gx + 2, 6)}V38H${N(gx + 13)}V6Z`, 'l3');
    P.c(gx - 4, 22, 1.7, 'g-gold l3'); P.c(gx + 4, 22, 1.7, 'g-gold l3');
    P.r(gx - 17, 42, 34, 9, 'c-door-d l3');
    for (const dx of [-8, 8]) P.p(poly(Array.from({ length: 6 }, (_, k) => [gx + dx + 2.6 * Math.cos(k * Math.PI / 3), 46.5 + 2.6 * Math.sin(k * Math.PI / 3)])), 'c-gold l3');
    for (const s of [-1, 1]) { const x = gx + s * 20; P.r(x - 3.6, 0, 7.2, 9, 'c-stone l3'); P.c(x, 14.2, 5.4, 'c-stone l3'); P.c(x, 14.2, 2.4, 'l3'); }
    P.r(gx - 24, -3, 48, 3, 'c-stone l3');
    yingshan(P, gx, gh, gw + 18, 24);
    // 门旁木牌（不写字）
    P.r(gx + gw / 2 + 10, 12, 10, 26, 'c-wood l2'); P.r(gx + gw / 2 + 11.8, 14, 6.4, 22, 'c-wood-l');
    return { g: P.s(), box: [-hw - 4, hw + 4, -4, 128] };
  }

  /* ── 卢沟桥与宛平城：十一孔联拱石桥，望柱上石狮，桥东头一对华表；宛平城灰砖城墙扼桥而立，城门上城楼，墙上几处弹痕 ── */
  function lugou(o = {}) {
    const P = Pen(), r = rng(13), L = o.L ?? 900, n = 11, deckY = 30, pier = 15, span = (L - pier * (n + 1)) / n, rise = span * 0.3, R = (span * span / 4 + rise * rise) / (2 * rise), spring = 3;
    // 宛平城（在桥东头后面）
    if (o.city !== false) {
      const c0 = L / 2 + 8, cw = o.cityW ?? 320, ch = 64, gx = c0 + 64;
      let tr = ''; for (let i = 0; i < 9; i++) { const x = c0 + 30 + i * 34 + (r() - 0.5) * 10, R2 = 14 + r() * 8; tr += Nt.clumpPath(x, ch + R2 * 0.3, R2, R2 * 0.7, r, 8); }
      P.p(tr, 'c-tree-d l3');
      P.r(c0, -6, cw, ch + 6, 'c-gw l1');
      let bk = ''; for (let y = -2.6; y < ch; y += 3.4) bk += `M${P2(c0, y)}H${N(c0 + cw)}`; P.p(bk, 'l-brick-g');
      P.r(c0, -6, cw, 18, 'c-shade');
      let mer = ''; for (let x = c0 + 2; x < c0 + cw - 6; x += 9) mer += rect(x, ch, 5.4, 5); P.p(mer, 'c-gw l3');
      P.p(`M${P2(gx - 13, -6)}V${N(24)}A13 13 0 0 0 ${P2(gx + 13, 24)}V-6Z`, 'c-ink');
      P.p(`M${P2(gx - 17, -6)}V${N(24)}A17 17 0 0 0 ${P2(gx + 17, 24)}V-6`, 'l2');
      // 弹痕：城门两侧墙面上几处
      let hole = ''; for (const [dx, y] of [[30, 40], [44, 22], [58, 50], [-30, 46], [80, 34], [36, 8]]) { const s = 1.4 + r() * 1.4; hole += `M${P2(gx + dx - s, y)}q${N(s)} ${N(s * 1.2)} ${N(s * 2)} 0q${N(-s * 0.6)} ${N(-s * 1.1)} ${N(-s * 2)} 0Z`; }
      P.p(hole, 'c-ink');
      const tower = J.withLod(o.lod ?? 0.6, () => J.BUILD.hall({ bays: [34, 26], colH: 30, platform: 0, type: 'xs', tone: 'gray', tile: 3.4, beasts: 2, per: 1, rw: 20 }));
      P.g(tower.g, ` transform="translate(${N(gx)} ${N(ch + 4)}) scale(.92)"`);
    }
    // 桥身：券洞挖空
    const ramp = 40;
    let d = `M${P2(-L / 2 - ramp, -6)}L${P2(-L / 2 - ramp, deckY - 10)}L${P2(-L / 2, deckY)}H${N(L / 2)}L${P2(L / 2 + ramp * 0.3, deckY - 3)}V-6Z`;
    const arches = [];
    for (let i = 0; i < n; i++) { const x0 = -L / 2 + pier + i * (span + pier), x1 = x0 + span; arches.push([x0, x1]); d += `M${P2(x0, -6)}V${N(spring)}A${N(R)} ${N(R)} 0 0 0 ${P2(x1, spring)}V-6Z`; }
    P.p(d, 'c-stone l1', ' fill-rule="evenodd"');
    let sh = '', ring = '', jt = '', cw2 = '';
    for (const [x0, x1] of arches) {
      // 券：圆心在起拱线下面（弧券），内弧、外弧同心，券石缝沿半径
      const c = (x0 + x1) / 2, cy = spring - (R - rise), f0 = Math.atan2(R - rise, -span / 2), f1 = Math.atan2(R - rise, span / 2), R2 = R + 3.6;
      const at = (rr, t) => { const f = f0 + (f1 - f0) * t; return [c + rr * Math.cos(f), cy + rr * Math.sin(f)]; };
      sh += 'M' + samp(t => at(R, t), 16).map(q => P2(...q)).join('L') + 'L' + samp(t => { const [x, y] = at(R * 0.96, 1 - t); return [x, y - 2.2]; }, 16).map(q => P2(...q)).join('L') + 'Z';
      ring += 'M' + samp(t => at(R2, t), 16).map(q => P2(...q)).join('L');
      for (let k = 1; k < 10; k++) { const a = at(R, k / 10), b = at(R2, k / 10); jt += `M${P2(...a)}L${P2(...b)}`; }
    }
    for (let i = 0; i <= n; i++) { const x = -L / 2 + i * (span + pier) + pier / 2; cw2 += `M${P2(x - pier * 0.62, -6)}L${P2(x, 9)}L${P2(x + pier * 0.62, -6)}Z`; }
    P.p(sh, 'c-shade2'); P.p(ring, 'l3'); P.p(jt, 'l-cun'); P.p(cw2, 'c-stone-d l3');
    // 仰天石、栏板、望柱与石狮
    P.p(`M${P2(-L / 2 - ramp, deckY - 10)}L${P2(-L / 2, deckY)}H${N(L / 2)}L${P2(L / 2 + ramp * 0.3, deckY - 3)}`, 'l2');
    const rh = 9, ps = 9.4, yAt = x => x < -L / 2 ? deckY - 10 * (-L / 2 - x) / ramp : x > L / 2 ? deckY - 3 * (x - L / 2) / (ramp * 0.3) : deckY;
    let pan = '', post = '', lion = '';
    for (let x = -L / 2 - ramp + 4; x <= L / 2 + ramp * 0.3 - 2; x += ps) {
      const y = yAt(x), y2 = yAt(x + ps);
      if (x + ps <= L / 2 + ramp * 0.3) pan += `M${P2(x + 1, y)}L${P2(x + ps - 1, y2)}L${P2(x + ps - 1, y2 + rh * 0.62)}L${P2(x + 1, y + rh * 0.62)}Z`;
      post += rect(x - 1.1, y, 2.2, rh);
      lion += `M${P2(x - 1.7, y + rh)}c0 ${N(rh * 0.3)} ${N(0.6)} ${N(rh * 0.42)} ${N(1.7)} ${N(rh * 0.42)}c${N(1.1)} 0 ${N(1.7)} ${N(-rh * 0.12)} ${N(1.7)} ${N(-rh * 0.42)}Z`;
    }
    P.p(pan, 'c-stone l3'); P.p(post, 'c-stone l3'); P.p(lion, 'c-stone-d l3');
    // 华表：桥东头一对
    for (const dx of [-6, 8]) { const x = L / 2 + ramp * 0.3 + dx + 4; P.r(x - 2.4, deckY - 3, 4.8, 46, 'c-marble l2'); P.r(x - 4.2, deckY - 3, 8.4, 4, 'c-marble l3'); P.p(`M${P2(x - 7, deckY + 34)}h14l-2 3h-10Z`, 'c-marble l3'); P.c(x, deckY + 46, 3, 'c-marble l3'); }
    return { g: P.s(), box: [-L / 2 - ramp - 4, o.city !== false ? L / 2 + 8 + (o.cityW ?? 320) + 4 : L / 2 + ramp + 20, -8, o.city !== false ? 64 + 4 + 80 : deckY + 54] };
  }

  /* ── 香山双清别墅：山坡上一处小院。院墙、院门；正房展室；池旁六角红亭；山石"双清"刻石；石经幢；几棵松、几处红叶 ── */
  function shuangqing(o = {}) {
    const P = Pen(), r = rng(21);
    // 后山的松
    for (const [x, h, sd] of [[-120, 120, 3], [118, 110, 8], [70, 92, 5]]) { const L = Nt.pine({ h, seed: sd, lean: x > 0 ? -1 : 1 }); P.g(L.g, ` transform="translate(${N(x)} 6)"`); }
    // 山石（双清刻石）
    const rk = Nt.rock({ w: 70, h: 44, seed: 17, blocks: [[-0.1, 0.7, 1], [0.3, 0.5, 0.6]] }); P.g(rk.g, ' transform="translate(-142 4)"');
    P.r(-150, 18, 6, 12, 'c-zhu');
    // 院里地面（山坡上，比墙脚高）
    P.p(poly([[-112, 0], [118, 0], [118, 16], [-112, 16]]), 'c-yard');
    // 正房：五间硬山，红柱隔扇
    const hx = 40, hwid = 120, hb = 16;
    P.r(hx - hwid / 2, hb, hwid, 30, 'c-wall-g l2');
    let col = ''; for (let i = 0; i <= 5; i++) col += rect(hx - hwid / 2 + i * hwid / 5 - 1.4, hb, 2.8, 30); P.p(col, 'c-red l3');
    let ls = ''; for (let i = 0; i < 5; i++) ls += rect(hx - hwid / 2 + i * hwid / 5 + 4, hb + 6, hwid / 5 - 8, 18); P.p(ls, 'p-lattice2 l3');
    yingshan(P, hx, hb + 30, hwid + 14, 20, false);
    // 池与六角红亭
    P.e(-58, 15, 38, 6, 'c-water l2');
    let wv = ''; for (const dx of [-22, -4, 14]) wv += `M${P2(-58 + dx, 15)}q3 1.6 6 0q3 -1.6 6 0`; P.p(wv, 'l-water');
    const px = -62, pb = 18;
    P.r(px - 24, pb, 48, 5, 'c-stone l2');
    let pc = ''; for (const dx of [-19, -7, 7, 19]) pc += rect(px + dx - 1.3, pb + 5, 2.6, 28); P.p(pc, 'c-red l3');
    P.r(px - 21, pb + 31, 42, 4, 'c-qing l3');
    P.g(J.sub(Q => A.roof(Q, { type: 'cj', sides: 6, ey: 0, ew: 30, rh: 30, tone: 'gray', tile: 3, beasts: 0, lift: 5, ridgeH: 2.6 })), ` transform="translate(${N(px)} ${N(pb + 34)})"`);
    P.g(J.sub(Q => A.baoding(Q, 0, 2.6)), ` transform="translate(${N(px)} ${N(pb + 64)})"`);
    // 石经幢
    const sx = 0; P.r(sx - 5, 16, 10, 4, 'c-stone l3'); P.r(sx - 2.4, 20, 4.8, 20, 'c-stone l3'); P.p(`M${P2(sx - 5.5, 40)}h11l-2.2 3.4h-6.6Z`, 'c-stone l3'); P.r(sx - 1.6, 43.4, 3.2, 4, 'c-stone l3');
    // 院墙：矮，压在最前
    P.r(-112, 0, 230, 10, 'c-gw l2'); P.p(poly([[-114, 10], [120, 10], [118, 13], [-112, 13]]), 'c-gray-d l3');
    P.r(84, 0, 16, 16, 'c-gw l2'); P.r(87, 0, 10, 12, 'c-door l3'); P.p(poly([[82, 16], [102, 16], [100, 20], [84, 20]]), 'c-gray-d l3');
    // 红叶：只点几处
    for (const [x, y, h, sd] of [[-96, 4, 62, 13], [150, 4, 56, 27], [-12, 14, 44, 31]]) { const L = Nt.maple({ h, seed: sd }); P.g(L.g, ` transform="translate(${N(x)} ${N(y)})"`); }
    return { g: P.s(), box: [-178, 182, -6, 132] };
  }

  /* ── 中国共产党历史展览馆：台阶基座上一长排方柱柱廊，柱后是暗下去的墙面与高窗；额枋、挑檐，檐下一排简化斗拱，泥金勾线 ── */
  function dangshi(o = {}) {
    const P = Pen(), n = o.n ?? 14, sp = 30, W = n * sp, hw = W / 2, H = 172, cw = 13;
    P.r(-hw - 26, 0, W + 52, 5, 'c-ds l2'); P.r(-hw - 19, 5, W + 38, 5, 'c-ds l2'); P.r(-hw - 12, 10, W + 24, 5, 'c-ds l2');
    P.r(-hw - 6, 15, W + 12, 9, 'c-ds-l l2');
    const y0 = 24;
    P.r(-hw, y0, W, H - y0, 'c-ds-d l2');
    let wn = ''; for (let i = 0; i < n - 1; i++) { const x = -hw + sp + i * sp; wn += rect(x - 5, y0 + 18, 10, H - y0 - 44); } P.p(wn, 'c-pane l3');
    let col = '', sh = '', gr = '', cap = '';
    for (let i = 0; i < n; i++) { const x = -hw + sp / 2 + i * sp; col += rect(x - cw / 2, y0, cw, H - y0 - 4); sh += rect(x + cw / 2 - 4, y0, 4, H - y0 - 4); gr += `M${P2(x - 2.6, y0 + 8)}V${N(H - 14)}M${P2(x + 1.6, y0 + 8)}V${N(H - 14)}`; cap += rect(x - cw / 2 - 2, H - 9, cw + 4, 5) + rect(x - cw / 2 - 1.6, y0, cw + 3.2, 4); }
    P.p(col, 'c-ds-l l2'); P.p(sh, 'c-shade'); P.p(gr, 'l3'); P.p(cap, 'c-ds-l l3');
    P.r(-hw - 4, H - 4, W + 8, 13, 'c-ds l2'); P.p(`M${P2(-hw - 4, H + 2.5)}H${N(hw + 4)}`, 'lg');
    let br = ''; for (let x = -hw + 2; x < hw - 4; x += 10) br += rect(x, H + 9, 6, 5); P.p(br, 'c-ds-l l3');
    P.r(-hw - 15, H + 14, W + 30, 7, 'c-ds-l l2'); P.p(`M${P2(-hw - 15, H + 17.5)}H${N(hw + 15)}`, 'lg');
    P.r(-hw - 3, H + 21, W + 6, 15, 'c-ds l2');
    return { g: P.s(), box: [-hw - 28, hw + 28, -2, H + 38] };
  }

  /* ── 小人：背影的一排（坐着听课的观众） ── */
  function seatedBack(P, x, y, s, c) {
    P.p(`M${P2(x - s * 0.5, y)}C${P2(x - s * 0.52, y + s * 0.5)} ${P2(x - s * 0.36, y + s * 0.68)} ${P2(x, y + s * 0.7)}C${P2(x + s * 0.36, y + s * 0.68)} ${P2(x + s * 0.52, y + s * 0.5)} ${P2(x + s * 0.5, y)}Z`, `c-robe-${c} l3`);
    P.r(x - s * 0.08, y + s * 0.66, s * 0.16, s * 0.1, 'c-skin');
    P.e(x, y + s * 0.9, s * 0.19, s * 0.23, 'c-hair l3');
  }
  const putP = (P, L, x, y, k = 1) => P.g(L.g, ` transform="translate(${N(x)} ${N(y)})${k !== 1 ? ` scale(${N(k)})` : ''}"`);

  /* ── 四种教学形式的小场景（各 320 × 200 左右） ── */
  // 现场教学：红砖楼前，讲解员指着楼讲，一组人听
  function vField() {
    const P = Pen();
    P.r(-160, 0, 320, 160, 'g-brick l1');
    let bk = ''; for (let y = 3.4; y < 160; y += 3.4) bk += `M${P2(-160, y)}H160`; P.p(bk, 'l-brick');
    P.r(-162, 0, 324, 10, 'c-stone l2'); P.r(-162, 82, 324, 4, 'c-trim l3'); P.r(-164, 156, 328, 8, 'c-trim l2');
    windows(P, [-124, -74, 74, 124], 26, 20, 40); windows(P, [-124, -74, 74, 124], 100, 20, 40);
    P.p(`M-17 10V${N(48)}A17 17 0 0 0 17 48V10Z`, 'c-door l2'); P.p('M-22 10V48A22 22 0 0 0 22 48V10', 'l2');
    P.r(-170, -30, 340, 30, 'c-pave'); P.p('M-170 0H170', 'l2');
    const guide = Nt.person({ h: 74, c: 'o', pose: 'point', dir: 1, hair: 'long' }); putP(P, guide, -108, -22);
    [['k', 'clasp'], ['w', 'stand'], ['k', 'stand'], ['b', 'clasp'], ['k', 'stand'], ['w', 'clasp']].forEach(([c, pose], i) => putP(P, Nt.person({ h: 68 + (i % 2) * 4, c, pose, dir: -1, hair: i === 2 ? 'long' : 'short' }), -38 + i * 30, -24 - (i % 2) * 5));
    return { g: P.s(), box: [-170, 170, -32, 166] };
  }
  // 主题党日：纪念碑前敬献花篮，两排人肃立（背影）；两侧松
  function vDay() {
    const P = Pen();
    P.r(-170, -34, 340, 34, 'c-pave'); P.p('M-170 0H170', 'l2');
    for (const [x, s] of [[-128, 1], [130, -1]]) putP(P, Nt.pine({ h: 150, seed: 3, lean: s }), x, 0, 1);
    P.r(-64, 0, 128, 9, 'c-stone l2'); P.r(-50, 9, 100, 9, 'c-stone l2');
    P.p(poly([[-26, 18], [26, 18], [22, 150], [-22, 150]]), 'c-monu l1'); P.p(poly([[-26, 18], [-14, 18], [-12, 150], [-22, 150]]), 'c-light');
    P.p(poly([[-27, 150], [27, 150], [18, 162], [-18, 162]]), 'c-monu l2');
    P.p('M-6 40V136M0 40V136M6 40V136', 'l-ins');
    // 花篮：竹架、篮身、一穹花，两条素红绸带
    P.p('M-12 18L-6 46M12 18L6 46M0 18V46', 'l2');
    P.p('M-14 46H14L10 58H-10Z', 'c-wood l2');
    const fr = rng(7); let fl = '', fl2 = '', fl3 = '';
    for (let i = 0; i < 40; i++) { const a = Math.PI * fr(), rr = 4 + fr() * 12, x = Math.cos(a) * rr * 1.3, y = 58 + Math.sin(a) * rr, s = 2.4 + fr(); const d = `M${P2(x - s, y)}a${N(s)} ${N(s)} 0 1 0 ${N(s * 2)} 0a${N(s)} ${N(s)} 0 1 0 ${N(-s * 2)} 0Z`; const k = fr(); if (k < 0.45) fl += d; else if (k < 0.75) fl2 += d; else fl3 += d; }
    P.p('M-20 58C-20 80 20 80 20 58Z', 'c-leaf l3'); P.p(fl3, 'c-flower-w l3'); P.p(fl2, 'c-flower-y l3'); P.p(fl, 'c-flower-r l3');
    P.p('M-8 46C-12 34 -16 26 -14 14L-10 14C-12 26 -8 34 -5 46Z', 'c-red l3'); P.p('M8 46C12 34 16 26 14 14L10 14C12 26 8 34 5 46Z', 'c-red l3');
    // 两排人，背影，深色正装
    for (let i = 0; i < 6; i++) putP(P, Nt.person({ h: 62, c: i % 3 === 1 ? 'b' : 'k', pose: 'stand', back: true, hair: i === 2 || i === 5 ? 'long' : 'short' }), -95 + i * 38, -12);
    for (let i = 0; i < 5; i++) putP(P, Nt.person({ h: 68, c: i % 2 ? 'k' : 'b', pose: 'stand', back: true, hair: i === 3 ? 'long' : 'short' }), -76 + i * 38, -30);
    return { g: P.s(), box: [-170, 170, -36, 166] };
  }
  // 专题党课：会场里，讲台上主讲，背后一幅浅绛山水；前景两排听课的背影
  function vLecture() {
    const P = Pen();
    P.r(-170, -40, 340, 210, 'c-wall-in'); P.r(-170, -40, 340, 66, 'c-wood-l'); P.p('M-170 26H170', 'l2');
    P.r(-96, 70, 150, 86, 'c-wood l2'); P.r(-90, 76, 138, 74, 'c-paper l3');
    const mt = Nt.mountain({ w: 130, h: 56, seed: 5, peaks: [[-0.4, 0.8, 0.26], [0.1, 1, 0.28], [0.6, 0.6, 0.24]], fans: false, trees: false }); putP(P, mt, -21, 78, 1);
    const sp = Nt.person({ h: 80, c: 'k', pose: 'stand', dir: -1 }); putP(P, sp, 104, 26);
    P.p(poly([[84, 26], [124, 26], [120, 74], [88, 74]]), 'c-wood l2'); P.r(82, 74, 44, 5, 'c-wood-d l3'); P.p('M96 79L92 92', 'l2'); P.c(92, 93, 1.6, 'c-ink');
    P.p('M118 79c0-4 6-4 6 0v6h-6Z', 'c-leaf l3');
    // 前景：两排桌子和背影
    P.r(-170, -6, 250, 7, 'c-wood l2');
    for (let i = 0; i < 7; i++) seatedBack(P, -150 + i * 34, -2, 40, ['k', 'b', 'k', 'w', 'k', 'b', 'k'][i]);
    let cups = ''; for (const x of [-132, -64, 4]) cups += rect(x, 1, 5, 6); P.p(cups, 'c-trim l3');
    P.r(-170, -40, 300, 8, 'c-wood l2');
    for (let i = 0; i < 6; i++) seatedBack(P, -140 + i * 42, -36, 50, ['b', 'k', 'w', 'k', 'b', 'k'][i]);
    return { g: P.s(), box: [-170, 170, -42, 170] };
  }
  // 情景教学：小舞台，红色幕布，旧式木桌、马灯、书；两位演员灰布军装、软帽（无徽）；前景观众背影
  function vScene() {
    const P = Pen();
    P.r(-170, 10, 340, 160, 'c-stage-bg');
    P.r(-60, 70, 80, 60, 'c-wood l2'); P.r(-55, 75, 70, 50, 'p-lattice2 l3');
    P.r(-170, 0, 340, 12, 'c-wood l2'); P.p('M-170 6H170', 'l3');
    for (const s of [-1, 1]) { const x0 = s * 170; P.p(`M${P2(x0, 170)}L${P2(x0 - s * 46, 170)}C${P2(x0 - s * 40, 120)} ${P2(x0 - s * 60, 60)} ${P2(x0 - s * 36, 12)}L${P2(x0, 12)}Z`, 'c-curtain l2'); P.p(`M${P2(x0 - s * 14, 170)}C${P2(x0 - s * 12, 110)} ${P2(x0 - s * 22, 60)} ${P2(x0 - s * 12, 12)}M${P2(x0 - s * 28, 170)}C${P2(x0 - s * 26, 110)} ${P2(x0 - s * 38, 60)} ${P2(x0 - s * 24, 12)}`, 'l-fold'); }
    let val = 'M-170 170H170V154'; for (let x = 170; x > -170; x -= 20) val += `Q${P2(x - 10, 144)} ${P2(x - 20, 154)}`; P.p(val + 'Z', 'c-curtain l2');
    // 演员
    putP(P, Nt.person({ h: 78, c: 'y', pose: 'clasp', dir: 1, cap: true }), -36, 12);
    putP(P, Nt.person({ h: 80, c: 'y', pose: 'point', dir: -1, cap: true }), 72, 12);
    // 旧木桌、马灯、书（桌子挡住坐着那位的腿）
    P.r(-74, 12, 6, 34, 'c-wood l3'); P.r(-6, 12, 6, 34, 'c-wood l3'); P.r(-80, 46, 86, 6, 'c-wood l2');
    P.c(-60, 64, 22, 'g-lamp'); P.p('M-64 52h8v3h-8ZM-63 55h6l1 9h-8ZM-61 64h2v4h-2ZM-64 68h8', 'c-gray-d l3'); P.p('M-62 57h4v5h-4Z', 'c-lampg');
    P.p('M-30 52h22l-2 4h-18Z', 'c-paper l3');
    // 观众背影
    for (let i = 0; i < 8; i++) seatedBack(P, -150 + i * 43, -40, 52, ['k', 'b', 'k', 'w', 'k', 'b', 'w', 'k'][i]);
    return { g: P.s(), box: [-170, 170, -42, 172] };
  }

  /* ── 时间线小图：大巴、研讨桌 ── */
  function bus() {
    const P = Pen();
    P.p('M-56 6V30C-56 38 -50 42 -40 42H50C56 42 60 38 60 30V14L54 6Z', 'c-bus l2');
    P.r(-50, 22, 96, 13, 'c-pane l3'); let m = ''; for (let x = -34; x < 46; x += 16) m += `M${P2(x, 22)}V35`; P.p(m, 'l-mull');
    P.p('M48 22H58V35H52Z', 'c-pane l3'); P.r(-56, 14, 116, 4, 'c-red'); P.r(30, 6, 10, 24, 'c-pane l3');
    for (const x of [-36, 36]) { P.c(x, 6, 7, 'c-ink'); P.c(x, 6, 3, 'c-gray'); }
    return { g: P.s(), box: [-62, 64, -2, 46] };
  }
  function meeting() {
    // 研讨：一张长条会议桌（略带透视），对面一排椅背和三个人，桌上名牌、茶杯、一盆绿植
    const P = Pen();
    for (const x of [-34, 0, 34]) P.p(`M${P2(x - 9, 14)}v22q0 4 4 4h10q4 0 4 -4v-22Z`, 'c-wood l3');
    for (const [x, c] of [[-34, 'k'], [0, 'b'], [34, 'k']]) { P.p(`M${P2(x - 8, 14)}C${P2(x - 8, 26)} ${P2(x + 8, 26)} ${P2(x + 8, 14)}Z`, `c-robe-${c} l3`); P.e(x, 30, 4.4, 5.2, 'c-skin l3'); P.p(`M${P2(x - 4.6, 31)}C${P2(x - 4.6, 37)} ${P2(x + 4.6, 37)} ${P2(x + 4.6, 31)}Q${P2(x, 33)} ${P2(x - 4.6, 31)}Z`, 'c-hair'); }
    P.p('M-56 4L-48 16H48L56 4Z', 'c-trim l2'); P.p('M-56 4H56V0H-56Z', 'c-wood l2');
    for (const x of [-34, 0, 34]) { P.p(`M${P2(x - 6, 9)}h12l-1 4h-10Z`, 'c-red l3'); P.r(x + 9, 8, 3.4, 4.4, 'c-trim l3'); }
    P.p('M-50 0V-10M50 0V-10', 'l2');
    P.p('M44 13c-3 -6 2 -9 4 -5c1 -4 6 -3 4 2c3 -1 4 3 1 4Z', 'c-leaf l3'); P.r(45, 9, 6, 4, 'c-wood l3');
    return { g: P.s(), box: [-58, 58, -12, 42] };
  }

  /* ── 槐：北京街巷里常见的老槐，干粗、略歪，树冠宽圆，一团团叶簇叠成穹顶，前亮后暗 ── */
  function huai(o = {}) {
    const P = Pen(), r = rng(o.seed || 51), h = o.h || 200, lean = o.lean ?? 1;
    const sp = samp(t => [lean * h * (0.04 * Math.sin(t * 2.4) + 0.03 * t), h * 0.46 * t], 10);
    const [tx, ty] = sp[sp.length - 1];
    const limbs = [[-0.9, 0.3], [-0.35, 0.36], [0.2, 0.36], [0.75, 0.3]].map(([a, len]) => { const x1 = tx + Math.sin(a) * h * len, y1 = ty + Math.cos(a) * h * len * 0.8; return [[tx, ty - h * 0.04], [tx + (x1 - tx) * 0.5, ty + (y1 - ty) * 0.55], [x1, y1]]; });
    const cl = [];
    for (let i = 0; i < 16; i++) { const t = i / 15, a = Math.PI * (0.08 + 0.84 * t), R = h * (0.11 + r() * 0.05), rr = h * (0.3 + r() * 0.06); cl.push([tx + Math.cos(a) * rr * 1.25 * -1, ty + h * 0.12 + Math.sin(a) * rr * 0.75, R, 0]); }
    for (let i = 0; i < 9; i++) { const x = tx + (r() - 0.5) * h * 0.55, y = ty + h * (0.12 + r() * 0.24); cl.push([x, y, h * (0.1 + r() * 0.05), 1]); }
    cl.sort((a, b) => b[1] - a[1]);
    for (const [x, y, R, f] of cl.filter(c => !c[3])) P.p(Nt.clumpPath(x, y, R, R * 0.72, r, 9), `${y > ty + h * 0.3 ? 'c-tree-d' : 'c-tree'} l2`);
    for (const L of limbs) P.p(Nt.stroke2(L, h * 0.018, h * 0.008), 'g-bark l2');
    P.p(Nt.stroke2(sp, h * 0.045, h * 0.03), 'g-bark l1');
    let bk = ''; for (let i = 1; i < sp.length - 1; i++) { const [x, y] = sp[i]; bk += `M${P2(x - h * 0.02, y)}q${N(h * 0.012)} ${N(h * 0.012)} ${N(h * 0.03)} ${N(h * 0.004)}`; } P.p(bk, 'l3');
    for (const [x, y, R] of cl.filter(c => c[3])) { P.p(Nt.clumpPath(x, y, R, R * 0.72, r, 9), 'c-tree-l l2'); P.p(`M${P2(x - R * 0.55, y + R * 0.12)}Q${P2(x - R * 0.12, y + R * 0.62)} ${P2(x + R * 0.46, y + R * 0.3)}Q${P2(x, y + R * 0.34)} ${P2(x - R * 0.55, y + R * 0.12)}Z`, 'c-light'); }
    let dots = ''; for (let i = 0; i < 46; i++) { const x = tx + (r() - 0.5) * h * 0.7, y = ty + h * (0.06 + r() * 0.42), q = h * 0.01; dots += `M${P2(x - q, y - q * 0.5)}L${P2(x, y + q * 0.3)}L${P2(x + q, y - q * 0.5)}`; } P.p(dots, 'l-dot');
    P.p(`M${P2(-h * 0.06, 0)}Q${P2(-h * 0.08, h * 0.006)} ${P2(-h * 0.1, 0)}M${P2(h * 0.05, 0)}Q${P2(h * 0.07, h * 0.006)} ${P2(h * 0.09, 0)}`, 'l2');
    return { g: P.s(), box: [-h * 0.46, h * 0.5, 0, h * 0.98] };
  }

  /* ── 胡同：一片灰瓦硬山屋顶，前后两排错落，间几棵槐 ── */
  function hutong(o = {}) {
    const P = Pen(), r = rng(o.seed || 41), w = o.w ?? 600, hw = w / 2;
    for (const [row, y0, k] of [[0, 18, 0.8], [1, 0, 1]]) {
      let x = -hw + (row ? 0 : 30);
      while (x < hw - 20) {
        const ww = (46 + r() * 34) * k, wh = 14 * k, rh = 12 * k, cx = x + ww / 2;
        P.r(cx - ww / 2, y0, ww, wh, `${row ? 'c-gw' : 'c-gw-l'} l2`);
        yingshan(P, cx, y0 + wh, ww + 6, rh, false);
        if (r() < 0.35) { const tx = cx + ww / 2 + 6, R = 16 * k; P.p(Nt.clumpPath(tx, y0 + wh + rh + R * 0.4, R, R * 0.7, r, 8), 'c-tree-d l3'); }
        x += ww + 4 + r() * 10;
      }
    }
    return { g: P.s(), box: [-hw - 8, hw + 8, -2, 70] };
  }

  /* ── 雁阵：一行人字，远近大小不一 ── */
  function geese(n = 7, o = {}) {
    const P = Pen(), r = rng(o.seed || 3); let d = '';
    for (let i = 0; i < n; i++) { const k = i - (n - 1) / 2, x = Math.abs(k) * 22 * (k < 0 ? -1 : 1) + i * 6, y = -Math.abs(k) * 9 + (r() - 0.5) * 3, s = 4.6 + r() * 1.4; d += `M${P2(x - s, y + s * 0.5)}Q${P2(x - s * 0.4, y + s * 0.1)} ${P2(x, y - s * 0.12)}Q${P2(x + s * 0.4, y + s * 0.1)} ${P2(x + s, y + s * 0.5)}`; }
    P.p(d, 'l-goose');
    return { g: P.s(), box: [-90, 110, -40, 6] };
  }

  /* ── 主视觉《红色地标群像》：香山在后，卢沟桥与宛平城横在河上做中景，党史展览馆柱廊在右侧收住，北大红楼居中在前，李大钊故居门楼在左前 ── */
  function compose(layout = 'desk', px = 1) {
    const D = layout === 'desk', W = D ? 1600 : 400, H = D ? 760 : 840, L = {}, hits = [];
    const add = (id, s) => { L[id] = (L[id] || '') + s; };
    const put = (id, el, cx, bottom, h, cls = '', hit) => { const s = h / (el.box[3] - el.box[2]); add(id, J.place(el, cx, bottom + el.box[2] * s, h, px, cls)); if (hit) hits.push({ id: hit, x: cx + el.box[0] * s, y: bottom - h, w: (el.box[1] - el.box[0]) * s, h }); return s; };
    const mist = (id, cx, bottom, w, h) => put(id, Nt.mistBlob({ w, h }), cx, bottom, h);
    const people = (n, guideAt = -1) => { const P = Pen(); const cs = ['k', 'b', 'w', 'k', 'k', 'b', 'w', 'k', 'b']; for (let i = 0; i < n; i++) { const g = i === guideAt; const Lp = Nt.person({ h: 40, c: g ? 'o' : cs[i % cs.length], pose: g ? 'point' : (i % 3 === 1 ? 'clasp' : 'stand'), dir: g ? -1 : 1, hair: i % 4 === 2 ? 'long' : 'short' }); P.g(Lp.g, ` transform="translate(${N(i * 15)} ${N((i % 2) * 3)})"`); } return { g: P.s(), box: [-14, n * 15, 0, 42] }; };
    let slips;
    if (D) {
      add('sky', `<rect width="${W}" height="${H}" fill="url(#zm-sky)"/><ellipse cx="1030" cy="330" rx="640" ry="250" fill="url(#zm-glow)"/><circle cx="1030" cy="262" r="40" fill="#f7d8b3" opacity=".75"/><circle cx="1030" cy="262" r="40" fill="none" stroke="#e8b585" stroke-width="${N(1 / px)}" opacity=".6"/>`);
      put('far', Nt.range({ w: 1700, h: 170, seed: 7, fill: 'g-shan-far', peaks: [[-0.92, 0.9, 0.08], [-0.66, 1, 0.09], [-0.36, 0.6, 0.1], [-0.02, 0.4, 0.12], [0.34, 0.55, 0.1], [0.66, 0.92, 0.09], [0.92, 0.8, 0.08]] }), 800, 410, 170, 'far');
      add('far', `<rect x="-20" y="0" width="${W + 40}" height="430" fill="#f3e6d2" opacity=".22"/>`);
      mist('far', 800, 450, 1800, 90);
      put('far', geese(7), 1240, 196, 36, 'far');
      // 香山与双清别墅（左后）
      put('back', Nt.mountain({ w: 700, h: 300, seed: 31, peaks: [[-0.34, 1, 0.3], [0.2, 0.76, 0.3], [0.62, 0.5, 0.26]], fill: 'g-shan', fans: false, trees: true }), 300, 520, 330);
      put('back', shuangqing(), 318, 430, 178, '', 'xiangshan');
      put('cloud1', Nt.cloud2({ w: 300, h: 36, seed: 3 }), 860, 316, 52);
      put('cloud1', Nt.cloud2({ w: 240, h: 28, seed: 11 }), 250, 150, 40);
      put('cloud1', Nt.cloud2({ w: 260, h: 30, seed: 5, dir: -1 }), 1400, 248, 44);
      // 永定河、卢沟桥与宛平城（中景）
      put('mid', Nt.lake({ x0: -20, x1: 1620, h: 150, seed: 5, cls: 'g-water' }), 0, 600, 150);
      put('mid', Nt.waves({ w: 1600, h: 140, s: 0.8, seed: 3 }), 800, 600, 140);
      mist('mid', 800, 478, 1800, 60);
      put('mid', Nt.bank({ w: 560, h: 30, seed: 8, end: 0.12 }), 40, 506, 30);
      put('mid', lugou({ L: 900, cityW: 760 }), 770, 500, 112, '', 'lugou');
      mist('mid', 760, 590, 1700, 70);
      add('mid', `<rect x="-20" y="600" width="${W + 40}" height="${H - 600}" fill="url(#zm-ground)"/>`);
      // 党史展览馆（右侧收住画面）
      put('right', Nt.cypress({ h: 200, seed: 9 }), 1262, 700, 150);
      put('right', dangshi({ n: 14 }), 1496, 694, 214, '', 'dangshi');
      // 北大红楼（居中在前），门前两组人
      put('front', honglou(), 760, 744, 208, '', 'honglou');
      put('front', huai({ h: 200, seed: 51 }), 360, 752, 210); put('front', huai({ h: 200, seed: 57, lean: -1 }), 1170, 752, 196);
      put('front', people(9, 0), 640, 758, 21);
      put('front', people(6), 900, 760, 20);
      // 李大钊故居门楼（左前）、右前一棵古柏
      put('fore', hutong({ w: 520 }), 110, 676, 76);
      put('fore', lidazhao(), 176, 772, 150, '', 'lidazhao');
      put('fore', Nt.cypress({ h: 200, seed: 5 }), 1566, 772, 224);
      slips = { xiangshan: [128, 286], lugou: [520, 382], dangshi: [1302, 452], honglou: [430, 536], lidazhao: [340, 616] };
    } else {
      add('sky', `<rect width="${W}" height="${H}" fill="url(#zm-sky)"/><ellipse cx="230" cy="250" rx="260" ry="190" fill="url(#zm-glow)"/><circle cx="262" cy="176" r="20" fill="#f7d8b3" opacity=".75"/>`);
      put('far', Nt.range({ w: 500, h: 110, seed: 7, fill: 'g-shan-far', peaks: [[-0.8, 0.9, 0.16], [-0.3, 0.6, 0.18], [0.3, 0.7, 0.18], [0.8, 1, 0.16]] }), 200, 270, 110, 'far');
      mist('far', 200, 292, 560, 60);
      put('far', geese(5), 300, 132, 22, 'far');
      put('back', Nt.mountain({ w: 320, h: 170, seed: 31, peaks: [[-0.3, 1, 0.34], [0.36, 0.7, 0.3]], fill: 'g-shan', fans: false, trees: true }), 120, 380, 200);
      put('back', shuangqing(), 124, 304, 92, '', 'xiangshan');
      put('back', Nt.bank({ w: 260, h: 16, seed: 9, end: 0.16 }), 336, 392, 16);
      put('back', dangshi({ n: 8 }), 336, 382, 118, '', 'dangshi');
      put('cloud1', Nt.cloud2({ w: 160, h: 24, seed: 3 }), 250, 214, 34);
      put('mid', Nt.lake({ x0: -20, x1: 420, h: 96, seed: 5, cls: 'g-water' }), 0, 476, 96);
      put('mid', Nt.waves({ w: 420, h: 86, s: 0.6, seed: 3 }), 200, 476, 86);
      mist('mid', 200, 400, 520, 34);
      put('mid', lugou({ L: 900, cityW: 300 }), 170, 444, 52, '', 'lugou');
      mist('mid', 200, 486, 520, 46);
      put('front', honglou({ w: 560 }), 200, 650, 150, '', 'honglou');
      add('mid', `<rect x="-20" y="490" width="${W + 40}" height="${H - 490}" fill="url(#zm-ground)"/>`);
      put('front', huai({ h: 200, seed: 51 }), 14, 656, 140); put('front', huai({ h: 200, seed: 57, lean: -1 }), 390, 656, 130);
      put('front', people(7, 0), 150, 662, 16);
      put('fore', hutong({ w: 420, seed: 5 }), 200, 752, 60);
      put('fore', lidazhao({ w: 240 }), 122, 844, 132, '', 'lidazhao');
      put('fore', Nt.cypress({ h: 200, seed: 5 }), 342, 846, 176);
      slips = { xiangshan: [30, 150], lugou: [178, 318], dangshi: [372, 236], honglou: [70, 548], lidazhao: [226, 716] };
    }
    const order = ['sky', 'far', 'back', 'cloud1', 'mid', 'right', 'front', 'fore'];
    return { W, H, hits, slips, layers: order.map(id => ({ id, svg: `<svg class="jh-layer zm" viewBox="0 0 ${W} ${H}" preserveAspectRatio="xMidYMax slice" aria-hidden="true">${L[id] || ''}</svg>` })) };
  }

  window.JH.RED = { defs, honglou, lidazhao, lugou, shuangqing, dangshi, huai, hutong, yingshan, vField, vDay, vLecture, vScene, bus, meeting, geese, compose, windows };
})();
