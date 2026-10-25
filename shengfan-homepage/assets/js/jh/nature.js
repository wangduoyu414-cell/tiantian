/* 界画库 · 景物程式（按《芥子园画传》的画法，用同一套墨线）
   松：干作鱼鳞皴，枝横出、先垂后扬，枝头一簇簇车轮形松针（扇面淡彩 + 放射墨线）。
   柏：老干扭丝纹、根部外撇，虬枝，枝上一团团鳞叶，后暗前亮，间出枯枝。
   柳：老干微倾，主枝先扬后展，柳丝先挑后垂、随风偏向一侧，丝上狭叶，前后两层。
   点叶树：蟹爪枝，枝头介字点叠出树冠，底下一层淡绿。
   夹叶枫：枝头双勾五角小叶，填朱砂。
   青绿山峦：脊线分面，披麻皴顺脊而下，峰顶矾头，脊上小树。
   青绿山石：两三块叠成一组，顶面石青石绿、立面赭石，唇线、转折线分面，斧劈皴上粗下尖。
   水：鱼鳞纹。云：如意云头攒成一朵，勾线旋进头心，拖云尾。人：现代装束的点景小人，按骨架摆姿势。
   局部坐标 u 右 v 上，原点在根部或底边中点；圆弧在翻转坐标里向上鼓用 sweep = 0。 */
(function () {
  'use strict';
  const { N, P2, Pen, cub, samp, poly, smooth, rng, yAt } = window.JH;

  // 一条脊线左右各偏 w，得到粗细渐变的干或枝
  function stroke2(spine, w0, w1) {
    const n = spine.length, L = [], R = [];
    for (let i = 0; i < n; i++) {
      const a = spine[Math.max(0, i - 1)], b = spine[Math.min(n - 1, i + 1)];
      let nx = -(b[1] - a[1]), ny = b[0] - a[0]; const l = Math.hypot(nx, ny) || 1; nx /= l; ny /= l;
      const w = w0 + (w1 - w0) * i / (n - 1);
      L.push([spine[i][0] + nx * w, spine[i][1] + ny * w]); R.push([spine[i][0] - nx * w, spine[i][1] - ny * w]);
    }
    return smooth(L) + 'L' + smooth(R.slice().reverse()).slice(1) + 'Z';
  }
  // 每个节点各自的半宽（人物四肢、柏干用）
  function taper(spine, ws) {
    const n = spine.length, L = [], R = [];
    for (let i = 0; i < n; i++) {
      const a = spine[Math.max(0, i - 1)], b = spine[Math.min(n - 1, i + 1)];
      let nx = -(b[1] - a[1]), ny = b[0] - a[0]; const l = Math.hypot(nx, ny) || 1; nx /= l; ny /= l;
      L.push([spine[i][0] + nx * ws[i], spine[i][1] + ny * ws[i]]); R.push([spine[i][0] - nx * ws[i], spine[i][1] - ny * ws[i]]);
    }
    return smooth(L) + 'L' + smooth(R.slice().reverse()).slice(1) + 'Z';
  }
  // 折线按段细分，半宽在节点之间线性过渡
  function subdiv(pts, ws, k = 4) {
    const S = [], W = [];
    for (let i = 0; i < pts.length - 1; i++) for (let j = 0; j < k; j++) { const t = j / k; S.push([pts[i][0] + (pts[i + 1][0] - pts[i][0]) * t, pts[i][1] + (pts[i + 1][1] - pts[i][1]) * t]); W.push(ws[i] + (ws[i + 1] - ws[i]) * t); }
    S.push(pts[pts.length - 1]); W.push(ws[ws.length - 1]);
    return [S, W];
  }
  const bend = (x0, y0, x1, y1, k, n = 10) => samp(t => [x0 + (x1 - x0) * t - (y1 - y0) * k * Math.sin(Math.PI * t), y0 + (y1 - y0) * t + (x1 - x0) * k * Math.sin(Math.PI * t)], n);

  /* ── 松 ─────────────────────────────────────── */
  function pine(o = {}) {
    const P = Pen(), r = rng(o.seed || 3), h = o.h || 200, lean = o.lean ?? 1;
    const sp = samp(t => [lean * h * (0.12 * Math.sin(t * 3.4) + 0.16 * t), h * 0.84 * t], 26);
    const limbs = (o.limbs || [[0.4, -1, 0.42, 0.05], [0.53, 1, 0.52, 0.03], [0.66, -1, 0.34, 0.03], [0.77, 1, 0.3, 0.01], [0.88, -1, 0.2, -0.01]]).map(([t, side, len, droop]) => {
      const i = Math.round(t * (sp.length - 1)), [x0, y0] = sp[i], x1 = x0 + side * lean * h * len, y1 = y0 - h * droop;
      return { ls: bend(x0, y0, x1, y1, -0.12 * side * lean), w: h * 0.022 * (1 - t * 0.4), side };
    });
    for (const L of limbs) P.p(stroke2(L.ls, L.w, L.w * 0.3), 'g-bark l2');
    P.p(stroke2(sp, h * 0.056, h * 0.016), 'g-bark l1');
    let sc = '';
    for (let i = 1; i < sp.length - 3; i++) {
      const [x, y] = sp[i], w = h * 0.056 * (1 - i / sp.length * 0.72);
      for (let k = -1; k <= 1; k++) { const cx = x + k * w * 0.5 + (r() - 0.5) * w * 0.3, cy = y + (r() - 0.5) * h * 0.02, a = w * 0.3; sc += `M${P2(cx - a, cy)}q${N(a)} ${N(-a * 0.9)} ${N(a * 2)} 0`; }
    }
    P.p(sc, 'l3');
    P.e(sp[8][0] + h * 0.01, sp[8][1], h * 0.012, h * 0.02, 'c-ink');
    // 松针：枝头成簇的扇形，后排暗、前排亮
    const cl = [];
    for (const L of limbs) { const n = 3 + Math.floor(r() * 2); for (let k = 0; k < n; k++) { const f = 0.3 + 0.7 * (k + 0.5) / n, [x, y] = L.ls[Math.round(f * (L.ls.length - 1))]; cl.push([x + (r() - 0.5) * h * 0.03, y + h * (0.01 + r() * 0.02), h * (0.06 + r() * 0.025), 1]); if (r() < 0.6) cl.push([x + L.side * h * 0.02, y + h * 0.05, h * 0.045, 0]); } }
    const [tx, ty] = sp[sp.length - 1]; cl.push([tx, ty + h * 0.02, h * 0.07, 1], [tx - lean * h * 0.05, ty + h * 0.07, h * 0.05, 0]);
    for (const front of [0, 1]) for (const [cx, cy, R, f] of cl) {
      if (f !== front) continue;
      P.p(`M${P2(cx - R, cy)}A${N(R)} ${N(R * 0.62)} 0 0 0 ${P2(cx + R, cy)}Q${P2(cx, cy - R * 0.16)} ${P2(cx - R, cy)}Z`, front ? 'c-pine l3' : 'c-pine-d l3');
      let nd = ''; for (let k = 0; k <= 18; k++) { const a = Math.PI * k / 18; nd += `M${P2(cx, cy - R * 0.06)}L${P2(cx + Math.cos(a) * R * 0.97, cy + Math.sin(a) * R * 0.6)}`; }
      P.p(nd, 'l-pine');
      if (front) P.p(`M${P2(cx - R * 0.62, cy + R * 0.38)}Q${P2(cx - R * 0.1, cy + R * 0.66)} ${P2(cx + R * 0.46, cy + R * 0.46)}`, 'lw');
    }
    const xs = cl.map(c => c[0]), ys = cl.map(c => c[1]);
    return { g: P.s(), box: [Math.min(-h * 0.12, ...xs) - h * 0.1, Math.max(h * 0.12, ...xs) + h * 0.1, 0, Math.max(...ys) + h * 0.08] };
  }

  // 叶团：波状勾边的一团（柏叶、林冠共用）；底下压扁一点
  function clumpPath(x, y, rx, ry, r, k = 11) {
    const pts = [];
    for (let j = 0; j < k; j++) { const a = Math.PI * 2 * j / k + r() * 0.2, s = 1 + (r() - 0.5) * 0.24; pts.push([x + Math.cos(a) * rx * s, y + Math.sin(a) * ry * s * (Math.sin(a) < 0 ? 0.72 : 1)]); }
    let d = `M${P2(...pts[0])}`;
    for (let j = 0; j < k; j++) { const a = pts[j], b = pts[(j + 1) % k], mx = (a[0] + b[0]) / 2, my = (a[1] + b[1]) / 2; d += `Q${P2(mx + (mx - x) * 0.3, my + (my - y) * 0.3)} ${P2(...b)}`; }
    return d + 'Z';
  }

  /* ── 柏：老干粗壮扭曲，扭丝纹斜绕，根部外撇；三四股虬枝，枝上一团团鳞叶（波状勾边、团内一排排鳞纹），
        后团暗、前团亮，顶上受光；冠外伸出一两枝枯枝 ── */
  function cypress(o = {}) {
    const P = Pen(), r = rng(o.seed || 5), h = o.h || 200, lean = o.lean ?? 1;
    const H0 = h * 0.34;
    const sp = samp(t => [lean * h * (0.045 * Math.sin(t * 3 + 0.3) + 0.02 * t), H0 * t], 12);
    const ws = sp.map((_, i) => { const t = i / (sp.length - 1); return h * (0.072 - 0.024 * t + 0.034 * Math.max(0, 0.18 - t) / 0.18); });
    const [tx, ty] = sp[sp.length - 1];
    const limbs = (o.limbs || [[-0.95, 0.36], [-0.34, 0.5], [0.2, 0.56], [0.8, 0.42]]).map(([a0, lf]) => {
      let x = tx + a0 * h * 0.03, y = ty - h * 0.012, th = (a0 + (r() - 0.5) * 0.2) * lean;
      const n = 9, ds = h * lf / n, pts = [];
      for (let k = 0; k <= n; k++) { pts.push([x, y]); th += (r() - 0.5) * 0.55 - th * 0.08; x += Math.sin(th) * ds; y += Math.cos(th) * ds; }
      return { pts, w: h * 0.034 * (1 - Math.abs(a0) * 0.22) };
    });
    // 叶团：沿枝分布，枝梢一团最大
    const clumps = [];
    for (const L of limbs) {
      const n = L.pts.length - 1;
      for (const k of [Math.round(n * 0.42), Math.round(n * 0.72), n]) {
        const [x, y] = L.pts[k], rx = h * (0.095 + r() * 0.055) * (k === n ? 1.12 : 0.92);
        clumps.push({ x: x + (r() - 0.5) * h * 0.04, y: y + rx * 0.16, rx, ry: rx * (0.5 + r() * 0.12), z: r() });
      }
    }
    clumps.push({ x: tx + lean * h * 0.03, y: h * 0.9, rx: h * 0.13, ry: h * 0.065, z: 0.2 }, { x: tx - lean * h * 0.1, y: h * 0.8, rx: h * 0.11, ry: h * 0.06, z: 0.45 });
    clumps.sort((a, b) => a.z - b.z);
    const clump = ({ x, y, rx, ry, z }) => {
      const k = 11, pts = [];
      for (let j = 0; j < k; j++) { const a = Math.PI * 2 * j / k + r() * 0.2, s = 1 + (r() - 0.5) * 0.24; pts.push([x + Math.cos(a) * rx * s, y + Math.sin(a) * ry * s * (Math.sin(a) < 0 ? 0.72 : 1)]); }
      let d = `M${P2(...pts[0])}`;
      for (let j = 0; j < k; j++) { const a = pts[j], b = pts[(j + 1) % k], mx = (a[0] + b[0]) / 2, my = (a[1] + b[1]) / 2; d += `Q${P2(mx + (mx - x) * 0.3, my + (my - y) * 0.3)} ${P2(...b)}`; }
      P.p(d + 'Z', `${z < 0.35 ? 'c-bai-d' : z < 0.7 ? 'c-bai' : 'c-bai-l'} l2`);
      P.p(`M${P2(x - rx * 0.72, y + ry * 0.18)}Q${P2(x - rx * 0.2, y + ry * 1.02)} ${P2(x + rx * 0.58, y + ry * 0.42)}Q${P2(x - rx * 0.06, y + ry * 0.5)} ${P2(x - rx * 0.72, y + ry * 0.18)}Z`, 'c-light');
      let sc = '', sc2 = '';
      const a = h * 0.011;
      for (let yy = y - ry * 0.62, row = 0; yy < y + ry * 0.78; yy += a * 1.2, row++) for (let xx = x - rx; xx < x + rx; xx += a * 2.3) {
        const xo = xx + (row % 2 ? a * 1.15 : 0) + (r() - 0.5) * a * 0.6;
        if (((xo + a - x) / (rx * 0.84)) ** 2 + ((yy - y) / (ry * 0.78)) ** 2 > 1) continue;
        const seg = `M${P2(xo, yy)}q${N(a)} ${N(a * 0.9)} ${N(a * 2)} 0`;
        if (yy > y + ry * 0.12) sc2 += seg; else sc += seg;
      }
      P.p(sc, 'l-scale'); P.p(sc2, 'l-scale2');
    };
    for (const c of clumps) if (c.z < 0.5) clump(c);
    // 枯枝：从一两根枝梢再伸出去，分叉，不着叶
    let dead = '';
    for (const L of [limbs[0], limbs[limbs.length - 1]]) {
      const [x0, y0] = L.pts[L.pts.length - 1], a = L.pts[L.pts.length - 2], dx = x0 - a[0], dy = y0 - a[1], l = Math.hypot(dx, dy);
      const x1 = x0 + dx / l * h * 0.12, y1 = y0 + dy / l * h * 0.12 + h * 0.03;
      dead += stroke2(bend(x0, y0, x1, y1, 0.15, 5), h * 0.008, h * 0.002);
      dead += stroke2(bend(x0 + (x1 - x0) * 0.5, y0 + (y1 - y0) * 0.5, x1 + (x1 - x0) * 0.1, y1 + h * 0.05, -0.2, 4), h * 0.004, h * 0.0015);
    }
    P.p(dead, 'g-bark l3');
    for (const L of limbs) P.p(stroke2(L.pts, L.w, L.w * 0.3), 'g-bark l2');
    // 干：扭丝纹斜绕，中段一处树洞
    P.p(taper(sp, ws), 'g-bark l1');
    let grain = '';
    for (let k = 0; k < 6; k++) grain += smooth(sp.map(([x, y], i) => [x + ws[i] * 0.86 * Math.sin(k * 1.05 + (i / (sp.length - 1)) * 3.4), y]));
    P.p(grain, 'l3');
    const [hx, hy] = sp[5];
    P.e(hx + ws[5] * 0.15, hy, ws[5] * 0.3, h * 0.03, 'c-ink l3');
    // 根：两侧外撇
    P.p(`M${P2(-ws[0] * 0.9, 0)}Q${P2(-ws[0] * 1.3, h * 0.005)} ${P2(-ws[0] * 1.6, 0)}M${P2(ws[0] * 0.9, 0)}Q${P2(ws[0] * 1.3, h * 0.006)} ${P2(ws[0] * 1.55, 0)}`, 'l2');
    for (const c of clumps) if (c.z >= 0.5) clump(c);
    const xs = clumps.flatMap(c => [c.x - c.rx * 1.12, c.x + c.rx * 1.12]), ys = clumps.map(c => c.y + c.ry * 1.1);
    return { g: P.s(), box: [Math.min(...xs, -h * 0.2), Math.max(...xs, h * 0.2), 0, Math.max(...ys)] };
  }

  /* ── 柳：老干微倾，顶上几根主枝先扬后展、梢头下弯；柳丝自枝上先挑后垂，随风偏向一侧，
        丝上狭叶顺丝斜出；身后一层淡绿，柳丝分前后两层，前亮后暗 ── */
  function willow(o = {}) {
    const P = Pen(), r = rng(o.seed || 7), h = o.h || 200, lean = o.lean ?? 1, wind = (o.wind ?? 0.12) * lean, dens = o.density ?? 1;
    const H0 = h * 0.46;
    const sp = samp(t => [lean * h * (0.06 * Math.sin(t * 2.6) + 0.04 * t), H0 * t], 12);
    // 主枝：从干的上段不同高度分出，先往上扬，再往外弯下来，几根合起来撑出一个穹顶
    const limbs = (o.limbs || [[-1.1, 0.72], [-0.7, 0.84], [-0.3, 1], [0.1, 1], [0.45, 0.9], [0.85, 0.78], [1.2, 0.66]]).map(([a0, f]) => {
      const sgn = a0 >= 0 ? 1 : -1, i0 = Math.round(f * (sp.length - 1)), [bx, by] = sp[i0];
      const len = h * (0.3 + r() * 0.08) * (1 - Math.abs(a0) * 0.12), n = 12, ds = len / n;
      let x = bx, y = by, th = a0 * lean * 0.55 + (r() - 0.5) * 0.1;
      const pts = [];
      for (let k = 0; k <= n; k++) { pts.push([x, y]); th += sgn * lean * (0.06 + 0.13 * (k / n) ** 1.4); x += Math.sin(th) * ds; y += Math.cos(th) * ds; }
      return { pts, w: h * 0.01 * (1 - Math.abs(a0) * 0.15) };
    });
    const [tx, ty] = sp[sp.length - 1];
    // 柳丝：三次曲线，起笔顺着枝势挑出，再垂下，梢头被风带偏
    const strands = [];
    for (const L of limbs) {
      const n = L.pts.length - 1;
      for (let k = 3; k <= n; k += dens >= 1 ? 1 : 2) {
        const m = k < n ? 1 + (r() < 0.7 * dens ? 1 : 0) : 3;
        for (let j = 0; j < m; j++) {
          const p0 = L.pts[k], a = L.pts[k - 1], b = L.pts[Math.min(n, k + 1)];
          let dx = b[0] - a[0], dy = b[1] - a[1]; const l = Math.hypot(dx, dy) || 1; dx /= l; dy /= l;
          const Ls = Math.min(p0[1] - h * 0.12, h * (0.3 + 0.3 * r()) * (0.5 + 0.5 * k / n));
          const q1 = [p0[0] + dx * Ls * 0.12, p0[1] + Ls * (0.04 + 0.06 * r())];
          const q3 = [q1[0] + wind * Ls * 0.55 + (r() - 0.5) * h * 0.03, Math.max(h * 0.03, p0[1] - Ls)];
          const q2 = [q1[0] + wind * Ls * 0.2 + dx * Ls * 0.08, p0[1] - (p0[1] - q3[1]) * 0.38];
          strands.push({ c: [p0, q1, q2, q3], z: r() });
        }
      }
    }
    strands.sort((a, b) => a.z - b.z);
    const curve = s => `M${P2(...s.c[0])}C${P2(...s.c[1])} ${P2(...s.c[2])} ${P2(...s.c[3])}`;
    const leaves = s => {
      const [a, b, c, d] = s.c; let out = '';
      const nl = Math.max(4, Math.round(Math.hypot(d[0] - a[0], d[1] - a[1]) / (h * 0.016) * dens));
      for (let q = 1; q <= nl; q++) {
        const t = 0.1 + 0.88 * q / nl, [x, y] = cub(a, b, c, d, t), [x2, y2] = cub(a, b, c, d, Math.min(1, t + 0.02));
        let ux = x2 - x, uy = y2 - y; const l = Math.hypot(ux, uy) || 1; ux /= l; uy /= l;
        const ang = (q % 2 ? 1 : -1) * 0.55, ca = Math.cos(ang), sa = Math.sin(ang), vx = ux * ca - uy * sa, vy = ux * sa + uy * ca;
        const lf = h * 0.024 * (1 - 0.3 * t), wd = lf * 0.27, nx = -vy * wd, ny = vx * wd;
        out += `M${P2(x, y)}q${N(vx * lf * 0.5 + nx)} ${N(vy * lf * 0.5 + ny)} ${N(vx * lf)} ${N(vy * lf)}q${N(-vx * lf * 0.5 - nx)} ${N(-vy * lf * 0.5 - ny)} ${N(-vx * lf)} ${N(-vy * lf)}Z`;
      }
      return out;
    };
    const layer = (list, leafCls) => { P.p(list.map(curve).join(''), 'l-willow-s'); P.p(list.map(leaves).join(''), leafCls); };
    P.p(strands.map(curve).join(''), 'l-wash-g', ` style="stroke-width:${N(h * 0.05)}px"`);
    const back = strands.filter(s => s.z < 0.3), mid = strands.filter(s => s.z >= 0.3 && s.z < 0.6), front = strands.filter(s => s.z >= 0.6);
    layer(back, 'c-willow-d');
    P.p(stroke2(sp, h * 0.05, h * 0.03), 'g-bark l1');
    for (const L of limbs) P.p(stroke2(L.pts, L.w, L.w * 0.25), 'g-bark l2');
    let bark = '';
    for (let i = 1; i < sp.length - 1; i++) { const [x, y] = sp[i], w = h * (0.046 - 0.018 * i / sp.length); if (r() < 0.65) bark += `M${P2(x - w * 0.7, y)}q${N(w * 0.5)} ${N(-h * 0.008)} ${N(w * 1.1)} ${N(h * 0.004)}`; }
    P.p(bark, 'l3');
    P.e(sp[4][0] + h * 0.008 * lean, sp[4][1], h * 0.012, h * 0.022, 'c-ink');
    P.p(`M${P2(-h * 0.055, 0)}Q${P2(-h * 0.075, h * 0.004)} ${P2(-h * 0.095, 0)}M${P2(h * 0.055, 0)}Q${P2(h * 0.07, h * 0.005)} ${P2(h * 0.09, 0)}`, 'l2');
    layer(mid, 'c-willow'); layer(front, 'c-willow-l');
    const xs = strands.flatMap(s => [s.c[0][0], s.c[3][0]]).concat(limbs.flatMap(L => L.pts.map(p => p[0]))), ys = limbs.flatMap(L => L.pts.map(p => p[1]));
    return { g: P.s(), box: [Math.min(...xs) - h * 0.04, Math.max(...xs) + h * 0.04, 0, Math.max(...ys) + h * 0.04] };
  }

  /* ── 点叶树：蟹爪枝 + 介字点 ─────────────────── */
  function dianye(o = {}) {
    const P = Pen(), r = rng(o.seed || 11), h = o.h || 160;
    const sp = samp(t => [h * 0.05 * Math.sin(t * 3), h * 0.55 * t], 10);
    const tips = [];
    const grow = (x0, y0, a, len, w, d) => {
      const x1 = x0 + Math.cos(a) * len, y1 = y0 + Math.sin(a) * len;
      P.p(stroke2(bend(x0, y0, x1, y1, (r() - 0.5) * 0.3, 6), w, w * 0.5), 'g-bark l3');
      if (d > 0) for (const s of [-1, 1]) grow(x1, y1, a + s * (0.35 + r() * 0.3), len * 0.62, w * 0.55, d - 1); else tips.push([x1, y1]);
    };
    const [tx, ty] = sp[sp.length - 1];
    for (const a of [1.15, 1.6, 2.05]) grow(tx, ty, a + (r() - 0.5) * 0.2, h * 0.22, h * 0.018, 2);
    P.p(stroke2(sp, h * 0.045, h * 0.024), 'g-bark l1');
    // 底色淡绿，再点介字点
    let wash = '';
    for (const [x, y] of tips) wash += `M${P2(x - h * 0.09, y)}a${N(h * 0.09)} ${N(h * 0.07)} 0 1 0 ${N(h * 0.18)} 0a${N(h * 0.09)} ${N(h * 0.07)} 0 1 0 ${N(-h * 0.18)} 0Z`;
    P.p(wash, 'c-wash-g');
    let dots = '';
    for (const [x, y] of tips) for (let j = 0; j < 9; j++) {
      const u = x + (r() - 0.5) * h * 0.16, v = y + (r() - 0.45) * h * 0.12, s = h * 0.014;
      dots += `M${P2(u - s, v - s * 0.6)}L${P2(u, v + s * 0.3)}L${P2(u + s, v - s * 0.6)}M${P2(u, v + s * 0.3)}L${P2(u, v - s * 0.9)}`;
    }
    P.p(dots, 'l-dot');
    return { g: P.s(), box: [-h * 0.42, h * 0.42, 0, h * 1.02] };
  }

  /* ── 夹叶枫：双勾五角小叶，填朱砂 ─────────────── */
  function maple(o = {}) {
    const P = Pen(), r = rng(o.seed || 13), h = o.h || 150;
    const sp = samp(t => [h * 0.04 * Math.sin(t * 3 + 1), h * 0.5 * t], 10);
    const tips = [];
    const grow = (x0, y0, a, len, w, d) => {
      const x1 = x0 + Math.cos(a) * len, y1 = y0 + Math.sin(a) * len;
      P.p(stroke2(bend(x0, y0, x1, y1, (r() - 0.5) * 0.3, 6), w, w * 0.5), 'g-bark l3');
      tips.push([x1, y1]);
      if (d > 0) for (const s of [-1, 1]) grow(x1, y1, a + s * (0.4 + r() * 0.3), len * 0.6, w * 0.55, d - 1);
    };
    const [tx, ty] = sp[sp.length - 1];
    for (const a of [1.2, 1.9]) grow(tx, ty, a, h * 0.24, h * 0.018, 2);
    P.p(stroke2(sp, h * 0.04, h * 0.022), 'g-bark l1');
    const leaf = (x, y, s, rot) => {
      const pts = [];
      for (let k = 0; k < 10; k++) { const a = rot + Math.PI * 2 * k / 10, rr = k % 2 ? s * 0.42 : s; pts.push([x + Math.cos(a) * rr, y + Math.sin(a) * rr]); }
      return poly(pts);
    };
    let red = '', org = '';
    for (const [x, y] of tips) for (let j = 0; j < 4; j++) { const d = leaf(x + (r() - 0.5) * h * 0.1, y + (r() - 0.3) * h * 0.08, h * (0.022 + r() * 0.01), r() * 6); if (r() < 0.7) red += d; else org += d; }
    P.p(red, 'c-maple l3'); P.p(org, 'c-maple-l l3');
    return { g: P.s(), box: [-h * 0.4, h * 0.4, 0, h * 0.95] };
  }

  /* ── 青绿山峦：主峰加肩峰；峰顶分出几道山脊，带转折，把山体分成面；披麻皴贴着脊线顺势而下；
        阳面石青→石绿→赭石晕染，峰顶矾头，脊上点景小树，棱上点苔。
     o: { w, h, seed, peaks: [[x 相对位置, 高度比, 宽度比]], trees } */
  function mountain(o = {}) {
    const P = Pen(), r = rng(o.seed || 21), w = o.w || 420, h = o.h || 240;
    const peaks = o.peaks || [[0.04, 1, 0.2], [-0.5, 0.66, 0.24], [0.56, 0.58, 0.24]];
    const n = 80, pts = [];
    for (let i = 0; i <= n; i++) {
      const t = -0.5 + i / n, x = t * w;
      let y = 0;
      for (const [cx, ph, pw] of peaks) { const d = (t - cx * 0.5) / pw; y = Math.max(y, ph * h * Math.max(0, 1 - d * d) ** 1.25); }
      y += (r() - 0.5) * h * 0.02 * (y > 0 ? 1 : 0);
      pts.push([x, Math.max(0, y)]);
    }
    pts[0][1] = 0; pts[n][1] = 0;
    P.p(smooth(pts) + 'Z', `${o.fill || 'g-shan'}`); P.p(smooth(pts), 'l1');
    const yTop = x => yAt(pts, x) ?? 0;
    // 山脊与披麻皴
    let ridge = '', cun = '', cun2 = '';
    for (const [cx, ph, pw] of peaks) {
      const xc = cx * w * 0.5;
      for (const s of [-1, 1]) for (let j = 0; j < 3; j++) {
        const x0 = xc + s * w * pw * (0.04 + j * 0.07), y0 = yTop(x0) - h * (0.03 + j * 0.02);
        const x1 = xc + s * w * pw * (0.26 + j * 0.2 + r() * 0.06), y1 = Math.max(h * 0.05, y0 - h * ph * (0.38 + r() * 0.16));
        const at = (tt, off = 0, dn = 0) => { const x = x0 + (x1 - x0) * tt + off, y = y0 + (y1 - y0) * tt ** 0.85 + Math.sin(tt * 7 + j * 2) * h * 0.012 - dn; return [x, Math.min(y, yTop(x) - h * 0.02)]; };
        ridge += smooth(samp(t => at(t), 7));
        for (let k = 0; k < 4; k++) {
          const a = 0.06 + r() * 0.3, b = Math.min(1, a + 0.3 + r() * 0.4);
          const seg = smooth(samp(t => at(a + (b - a) * t, (k + 1) * w * 0.008, h * 0.008 * k), 5));
          if (k < 2) cun += seg; else cun2 += seg;
        }
      }
    }
    P.p(cun2, 'l-cun'); P.p(cun, 'l3'); P.p(ridge, 'l2');
    // 矾头：峰顶一簇圆头小石，按所在位置的山脊高度往下放，落在山体里面
    let fan = '';
    if (o.fans !== false) for (const [cx, ph, pw] of peaks) {
      const xc = cx * w * 0.5, R = Math.min(w * 0.018, w * pw * 0.09);
      for (let k = 0; k < 4; k++) { const x = xc + (k - 1.5) * R * 1.7, yt = yTop(x); if (yt < ph * h * 0.82) continue; const y = yt - R * (1.1 + Math.abs(k - 1.5) * 0.25); fan += `M${P2(x - R, y)}a${N(R)} ${N(R * 0.7)} 0 0 0 ${N(R * 2)} 0`; }
    }
    P.p(fan, 'l2');
    // 脊上点景小树：一根竖干，三层横向叶团
    if (o.trees !== false) {
      let tr = '', tc = '';
      for (const [cx, ph, pw] of peaks) {
        const m = 3 + Math.floor(r() * 3);
        for (let k = 0; k < m; k++) {
          const x = cx * w * 0.5 + (r() - 0.5) * w * pw * 0.9, y = yTop(x); if (y < h * 0.25) continue;
          const th = h * (0.035 + r() * 0.025), yb = y - h * 0.012;
          tr += `M${P2(x, yb)}V${N(yb + th)}`;
          for (let q = 0; q < 3; q++) { const yy = yb + th * (0.45 + q * 0.22), ww = th * (0.32 - q * 0.08); tc += `M${P2(x - ww, yy)}a${N(ww)} ${N(ww * 0.5)} 0 1 0 ${N(ww * 2)} 0a${N(ww)} ${N(ww * 0.5)} 0 1 0 ${N(-ww * 2)} 0Z`; }
        }
      }
      P.p(tr, 'l3'); P.p(tc, 'c-pine-d');
    }
    let moss = '';
    for (let i = 4; i < n - 3; i += 3) { const [x, y] = pts[i]; if (y > h * 0.3) moss += `M${P2(x + (r() - 0.5) * 4, y - 2 - r() * 4)}h.01`; }
    P.p(moss, 'l-moss');
    return { g: P.s(), box: [-w / 2, w / 2, 0, h * 1.06], pts };
  }

  /* ── 青绿山石：两三块叠成一组（后高前低）。每块：顶面石青石绿，立面赭石；一道唇线分出顶面，一道转折线分出前面（亮）与侧面（暗）；
        斧劈皴是上粗下尖的斜笔，侧面密、前面疏；棱上点苔。轮廓每段略外鼓，转角保持硬朗。 ── */
  function rock(o = {}) {
    const P = Pen(), r = rng(o.seed || 17), w = o.w || 160, h = o.h || 110;
    const blocks = o.blocks || [[-0.14, 0.64, 1], [0.24, 0.5, 0.62], [-0.3, 0.36, 0.34]];
    const seg = (a, b) => { const mx = (a[0] + b[0]) / 2, my = (a[1] + b[1]) / 2; return `Q${P2(mx - (b[1] - a[1]) * 0.035, my + (b[0] - a[0]) * 0.035)} ${P2(...b)}`; };
    const chain = pts => pts.slice(1).map((b, i) => seg(pts[i], b)).join('');
    for (const [cx, bw, bh] of blocks) {
      const W = bw * w, X0 = cx * w - W / 2, H = bh * h, j = () => (r() - 0.5) * 0.05;
      const pt = (fx, fy) => [X0 + W * (fx + j()), H * (fy + j())];
      const out = [[X0, 0], pt(0.02, 0.38), pt(0.07, 0.7), pt(0.18, 0.9), pt(0.36, 0.98), pt(0.5, 0.93), pt(0.6, 0.99), pt(0.78, 0.88), pt(0.9, 0.7), pt(0.98, 0.36), [X0 + W, 0]];
      const lip = [out[2], pt(0.24, 0.73), pt(0.46, 0.79), pt(0.64, 0.76), pt(0.78, 0.69), out[8]];
      const fold = [lip[3], pt(0.68, 0.42), [X0 + W * 0.65, 0]];
      P.p(`M${P2(...out[0])}` + chain(out) + 'Z', 'c-zhe');
      P.p(`M${P2(...lip[3])}` + chain(lip.slice(3)) + chain(out.slice(8)) + `L${P2(...fold[2])}L${P2(...fold[1])}Z`, 'c-zhe-d');
      P.p(`M${P2(...out[2])}` + chain(out.slice(2, 9)) + chain(lip.slice().reverse()) + 'Z', 'g-rocktop');
      // 斧劈皴
      let cun = '';
      const inside = (x, y) => { let c = false; for (let i = 0, k = out.length - 1; i < out.length; k = i++) { const [xi, yi] = out[i], [xk, yk] = out[k]; if ((yi > y) !== (yk > y) && x < (xk - xi) * (y - yi) / (yk - yi) + xi) c = !c; } return c; };
      const wedge = (x, y, L, wt) => { L = Math.min(L, y - H * 0.04); const m = W * 0.02; return L > H * 0.05 && inside(x - wt - m, y) && inside(x + wt + m, y) && inside(x + L * 0.3 + m, y - L) ? `M${P2(x - wt, y)}L${P2(x + wt, y)}L${P2(x + L * 0.3, y - L)}Z` : ''; };
      for (let k = 0; k < 14; k++) { const f = 0.7 + r() * 0.26, x = X0 + W * f, y = H * (0.6 - r() * 0.2) * (1 - (f - 0.7) * 0.8), L = H * (0.18 + r() * 0.3); cun += wedge(x, y, L, W * 0.012); }
      for (let k = 0; k < 6; k++) { const f = 0.12 + r() * 0.5, x = X0 + W * f, y = H * (0.66 - r() * 0.12), L = H * (0.12 + r() * 0.2); cun += wedge(x, y, L, W * 0.008); }
      P.p(cun, 'c-cun');
      P.p(`M${P2(...lip[0])}` + chain(lip) + `M${P2(...fold[0])}` + chain(fold), 'l2');
      P.p(`M${P2(X0 + W * 0.36, H * 0.74)}q${N(-W * 0.04)} ${N(-H * 0.2)} ${N(W * 0.02)} ${N(-H * 0.42)}M${P2(X0 + W * 0.86, H * 0.62)}q${N(W * 0.03)} ${N(-H * 0.2)} 0 ${N(-H * 0.36)}`, 'l3');
      P.p(`M${P2(...out[0])}` + chain(out), 'l1');
      let moss = '';
      for (const q of out.slice(2, 9)) for (let k = 0; k < 2; k++) moss += `M${P2(q[0] + (r() - 0.5) * W * 0.06, q[1] - r() * H * 0.04)}h.01`;
      P.p(moss, 'l-moss');
    }
    return { g: P.s(), box: [-w * 0.55, w * 0.55, 0, h * 1.06] };
  }

  /* ── 水：鱼鳞纹 ─────────────────────────────── */
  function water(o = {}) {
    const P = Pen(), w = o.w || 300, h = o.h || 80, s = o.s || 9;
    P.r(-w / 2, 0, w, h, 'c-water');
    let d = '';
    for (let row = 0, y = h - s * 0.6; y > 0; row++, y -= s * 0.62) for (let x = -w / 2 + (row % 2) * s * 0.5; x < w / 2 - s; x += s) d += `M${P2(x, y)}a${N(s / 2)} ${N(s * 0.42)} 0 0 0 ${N(s)} 0`;
    P.p(d, 'l-water');
    return { g: P.s(), box: [-w / 2, w / 2, 0, h] };
  }

  /* ── 祥云：几枚如意云头攒成一朵。云头从后往前画，每枚只勾上半圈，与云身连成一体；
        勾线不断，顺势旋进头心成回旋（外圈一道、里圈一道双勾）；底下一道云脚，一侧拖出 S 形渐细的云尾，尾梢回卷。 ── */
  function cloud(o = {}) {
    const P = Pen(), s0 = o.s || 1, dir = o.dir || 1, R = 16 * s0;
    const X = x => x * R * dir, Y = y => y * R;
    // 云尾
    const tailSp = samp(t => [X(1.5 + 5.4 * t), Y(0.62 + 0.36 * Math.sin(Math.PI * t * 1.1) - 0.2 * t)], 24);
    P.p(taper(tailSp, tailSp.map((_, i) => R * (0.46 * (1 - i / 24) ** 0.85 + 0.035))), 'g-cloud l-cloud');
    P.p(smooth(samp(t => { const [x, y] = tailSp[Math.round(2 + t * 18)]; return [x, y + R * 0.12 * (1 - t)]; }, 8)), 'l-cloud2');
    const [ex, ey] = tailSp[24];
    P.p('M' + samp(t => { const th = Math.PI * (0.5 + 1.7 * t), q = R * (0.3 - 0.2 * t); return [ex + dir * Math.cos(th) * q * -1, ey + q + Math.sin(th) * q * -1 - R * 0.3]; }, 18).map(p => P2(...p)).join('L'), 'l-cloudline');
    // 云脚
    P.e(0, Y(0.56), R * 2.75, R * 0.5, 'g-cloud l-cloud');
    // 云头：[x, y, r, 旋向, 外侧多勾一段]
    const heads = o.heads || [[-0.95, 2.25, 0.7, -1, 0], [0.86, 2.32, 0.6, 1, 0], [-1.85, 1.02, 0.92, -1, 1], [1.78, 1.0, 0.88, 1, 1], [0, 1.4, 1.25, 1, 0]];
    for (const [hx0, hy0, hr0, hand0, outer] of heads) {
      const cx = X(hx0), cy = Y(hy0), rr = hr0 * R, hand = hand0 * dir;
      P.c(cx, cy, rr, 'g-cloud');
      // 勾线：顺旋向从一侧经顶到另一侧，接着旋进头心
      const a1 = hand > 0 ? Math.PI + 0.3 : -0.3, span = Math.PI + 0.6 + (outer ? 0.85 : 0), turn = Math.PI * 1.35;
      const th = t => a1 - hand * t;
      const outline = samp(t => [cx + Math.cos(th(t * span)) * rr, cy + Math.sin(th(t * span)) * rr], 30);
      const spiral = samp(t => { const q = rr * (1 - 0.78 * t); return [cx + Math.cos(th(span + t * turn)) * q, cy + Math.sin(th(span + t * turn)) * q]; }, 36);
      P.p('M' + outline.concat(spiral.slice(1)).map(p => P2(...p)).join('L'), 'l-cloudline');
      P.p('M' + samp(t => { const q = rr * (0.84 - 0.62 * t); return [cx + Math.cos(th(span + 0.5 + t * (turn - 0.3))) * q, cy + Math.sin(th(span + 0.5 + t * (turn - 0.3))) * q]; }, 30).map(p => P2(...p)).join('L'), 'l-cloud2');
    }
    return { g: P.s(), box: dir > 0 ? [-R * 3, R * 7.4, -R * 0.1, R * 3.1] : [-R * 7.4, R * 3, -R * 0.1, R * 3.1] };
  }

  /* ── 点景人物：现代装束（夹克、长裤、皮鞋），按骨架摆姿势：stand 站、walk 走、point 指点、flag 举小旗（导游）、photo 拍照、clasp 双手交握。
        画序：远侧手臂、腿、衣身、近侧手臂、手、头发。光从左上来，衣身右侧压暗。
     o: { h, c 衣色 r b o g w, tc 裤色类名, pose, dir 朝向, bag 双肩包, hair 'short' | 'long' } */
  function person(o = {}) {
    const P = Pen(), h = o.h || 40, c = o.c || 'b', pose = o.pose || 'stand', dir = o.dir || 1, lean = pose === 'walk' ? 0.035 : 0.006;
    const J = {
      head: [0.006, 0.926], shF: [0.098, 0.812], shB: [-0.098, 0.812],
      elF: [0.122, 0.645], elB: [-0.122, 0.645], wrF: [0.128, 0.488], wrB: [-0.128, 0.488],
      hipF: [0.056, 0.5], hipB: [-0.056, 0.5], knF: [0.058, 0.272], knB: [-0.05, 0.272], anF: [0.058, 0.038], anB: [-0.05, 0.038],
    };
    Object.assign(J, {
      walk: { knF: [0.085, 0.28], anF: [0.115, 0.04], knB: [-0.045, 0.27], anB: [-0.095, 0.07], elF: [0.14, 0.648], wrF: [0.175, 0.5], elB: [-0.135, 0.646], wrB: [-0.165, 0.5] },
      point: { elF: [0.2, 0.75], wrF: [0.31, 0.83], head: [0.014, 0.928] },
      flag: { elF: [0.19, 0.72], wrF: [0.17, 0.875] },
      photo: { elF: [0.12, 0.66], wrF: [0.045, 0.765], elB: [-0.11, 0.66], wrB: [-0.03, 0.765] },
      clasp: { elF: [0.118, 0.64], wrF: [0.03, 0.53], elB: [-0.118, 0.64], wrB: [-0.02, 0.53] },
      stand: { hipB: [-0.056, 0.506], knB: [-0.046, 0.272] },
    }[pose] || {});
    const pt = ([u, v]) => [(u + lean * (v - 0.04)) * h * dir, v * h];
    const limb = (js, ws, cls) => { const [S, W] = subdiv(js.map(pt), ws.map(x => x * h), 4); P.p(taper(S, W), cls); };
    const shoe = an => { const [x, y] = pt(an), d = dir * h; return `M${P2(x - d * 0.028, y + h * 0.028)}C${P2(x - d * 0.036, y)} ${P2(x - d * 0.022, y - h * 0.038)} ${P2(x, y - h * 0.038)}L${P2(x + d * 0.05, y - h * 0.038)}C${P2(x + d * 0.07, y - h * 0.038)} ${P2(x + d * 0.072, y - h * 0.014)} ${P2(x + d * 0.05, y - h * 0.006)}L${P2(x + d * 0.018, y + h * 0.028)}Z`; };
    const hand = (el, wr) => { const [x0, y0] = pt(el), [x1, y1] = pt(wr), l = Math.hypot(x1 - x0, y1 - y0) || 1, x = x1 + (x1 - x0) / l * h * 0.02, y = y1 + (y1 - y0) / l * h * 0.02; P.e(x, y, h * 0.019, h * 0.024, 'c-skin l3'); };
    const jacket = `c-robe-${c}`;
    if (o.bag) { const [x, y] = pt([-0.118, 0.68]); P.p(`M${P2(x - dir * h * 0.04, y - h * 0.11)}h${N(dir * h * 0.075)}v${N(h * 0.23)}q0 ${N(h * 0.03)} ${N(-dir * h * 0.03)} ${N(h * 0.03)}h${N(-dir * h * 0.02)}q${N(-dir * h * 0.025)} 0 ${N(-dir * h * 0.025)} ${N(-h * 0.03)}Z`, 'c-wood l3'); }
    limb([J.shB, J.elB, J.wrB], [0.033, 0.027, 0.021], `${jacket} l3`);
    hand(J.elB, J.wrB);
    limb([J.hipB, J.knB, J.anB], [0.042, 0.031, 0.022], `${o.tc || 'c-trou'} l3`);
    limb([J.hipF, J.knF, J.anF], [0.042, 0.031, 0.022], `${o.tc || 'c-trou'} l3`);
    P.p(shoe(J.anB) + shoe(J.anF), 'c-ink');
    // 衣身：肩、腰、下摆
    const tor = [[-0.03, 0.848], [-0.096, 0.824], [-0.112, 0.75], [-0.104, 0.62], [-0.108, 0.468], [0, 0.452], [0.108, 0.468], [0.104, 0.62], [0.112, 0.75], [0.096, 0.824], [0.03, 0.848]].map(pt);
    P.p(smooth(tor, true), `${jacket} l2`);
    const sh = dir;   // 画面右侧那一半压暗
    P.p(poly([[0.05 * sh, 0.835], [0.096 * sh, 0.824], [0.112 * sh, 0.75], [0.104 * sh, 0.62], [0.108 * sh, 0.468], [0.055 * sh, 0.458]].map(pt)), 'c-shade');
    if (!o.back) P.p(poly([[-0.03, 0.848], [0, 0.79], [0.03, 0.848]].map(pt)), 'c-robe-w l3');
    P.p(`M${P2(...pt([0, 0.79]))}L${P2(...pt([0, 0.456]))}M${P2(...pt([-0.104, 0.49]))}L${P2(...pt([0.104, 0.49]))}`, 'l3');
    if (o.bag) P.p(`M${P2(...pt([0.055, 0.83]))}L${P2(...pt([0.075, 0.6]))}`, 'l2');
    if (pose === 'photo') { const [x, y] = pt([0.006, 0.772]); P.p(`M${P2(x - h * 0.042, y - h * 0.026)}h${N(h * 0.084)}v${N(h * 0.052)}h${N(-h * 0.084)}Z`, 'c-ink'); P.c(x + dir * h * 0.012, y, h * 0.014, 'c-gray l3'); }
    limb([J.shF, J.elF, J.wrF], [0.033, 0.027, 0.021], `${jacket} l3`);
    if (pose === 'flag') {
      const [x, y] = pt(J.wrF);
      P.p(`M${P2(x, y - h * 0.08)}L${P2(x + dir * h * 0.01, y + h * 0.34)}`, 'l2');
      const tx = x + dir * h * 0.01, ty = y + h * 0.34;
      P.p(`M${P2(tx, ty)}L${P2(tx + dir * h * 0.15, ty - h * 0.045)}L${P2(tx, ty - h * 0.1)}Z`, 'c-flag l3');
    }
    hand(J.elF, J.wrF);
    // 头、发、耳
    const [hx, hy] = pt(J.head), d = dir * h;
    P.p(`M${P2(hx - h * 0.02, hy - h * 0.1)}h${N(h * 0.04)}v${N(h * 0.05)}h${N(-h * 0.04)}Z`, 'c-skin l3');
    P.e(hx, hy, h * 0.054, h * 0.066, 'c-skin l3');
    if (o.back) {   // 背影：整个头是头发，不画脸和耳朵
      P.e(hx, hy, h * 0.056, h * 0.067, 'c-hair l3');
      if (o.hair === 'long') P.p(`M${P2(hx - h * 0.05, hy - h * 0.02)}Q${P2(hx, hy - h * 0.14)} ${P2(hx + h * 0.05, hy - h * 0.02)}Z`, 'c-hair');
      if (o.cap) P.p(`M${P2(hx - h * 0.064, hy + h * 0.02)}C${P2(hx - h * 0.066, hy + h * 0.1)} ${P2(hx + h * 0.066, hy + h * 0.1)} ${P2(hx + h * 0.064, hy + h * 0.02)}Z`, 'c-robe-y l3');
      return { g: P.s(), box: [-h * 0.36, h * 0.36, 0, h] };
    }
    const nape = o.hair === 'long' ? [hx - d * 0.064, hy - h * 0.13] : [hx - d * 0.036, hy - h * 0.056];
    P.p(`M${P2(hx + d * 0.05, hy + h * 0.02)}C${P2(hx + d * 0.05, hy + h * 0.076)} ${P2(hx - d * 0.03, hy + h * 0.09)} ${P2(hx - d * 0.06, hy + h * 0.03)}` +
      `C${P2(hx - d * 0.07, hy - h * 0.02)} ${P2(nape[0] - d * 0.008, nape[1] + h * 0.02)} ${P2(...nape)}` +
      `C${P2(nape[0] + d * 0.02, nape[1] + h * 0.03)} ${P2(hx - d * 0.012, hy + h * 0.01)} ${P2(hx + d * 0.012, hy + h * 0.03)}Q${P2(hx + d * 0.035, hy + h * 0.036)} ${P2(hx + d * 0.05, hy + h * 0.02)}Z`, 'c-hair');
    P.e(hx - d * 0.006, hy - h * 0.004, h * 0.009, h * 0.014, 'c-skin-d');
    // 软帽（灰布，无帽徽）：帽顶盖住头发，帽檐朝前
    if (o.cap) P.p(`M${P2(hx - d * 0.062, hy + h * 0.022)}C${P2(hx - d * 0.064, hy + h * 0.104)} ${P2(hx + d * 0.062, hy + h * 0.104)} ${P2(hx + d * 0.06, hy + h * 0.022)}L${P2(hx + d * 0.092, hy + h * 0.012)}L${P2(hx + d * 0.088, hy + h * 0.004)}L${P2(hx - d * 0.062, hy + h * 0.012)}Z`, 'c-robe-y l3');
    return { g: P.s(), box: [-h * 0.36, h * 0.36, 0, pose === 'flag' ? h * 1.26 : h] };
  }


  /* ── 勾云：一朵长条白云。上沿一枚枚大小不一的云头连成起伏的轮廓，下沿平缓，两头收尖；
        每枚云头露在外面的那段弧里再勾一道内线，最大的一两枚旋进去成回旋；一头拖一缕云尾 ── */
  function cloud2(o = {}) {
    const P = Pen(), r = rng(o.seed || 71), w = o.w || 500, h = o.h || 60, dir = o.dir || 1;
    const nL = o.lobes || Math.max(4, Math.round(w / (h * 1.05)));
    const th = x => { const u = 2 * x / w; return h * Math.max(0, 1 - u * u) ** 0.65; };
    const lobes = [];
    for (let i = 0; i < nL; i++) { const t = (i + 0.5) / nL, x = (t - 0.5) * w * 0.84 + (r() - 0.5) * w * 0.05, T = th(x), R = T * (0.42 + r() * 0.34); lobes.push({ x, R, cy: T * 0.6 - R * 0.42 }); }
    const topY = x => { let y = th(x) * 0.6; for (const L of lobes) { const d = x - L.x; if (Math.abs(d) < L.R) y = Math.max(y, L.cy + Math.sqrt(L.R * L.R - d * d)); } return y; };
    const botY = x => -th(x) * 0.4 + Math.sin(x / w * 8 + 1) * h * 0.035 * (th(x) / h);
    const F = ([x, y]) => [x * dir, y];
    const top = samp(t => [(t - 0.5) * w, topY((t - 0.5) * w)], 160), bot = samp(t => [(0.5 - t) * w, botY((0.5 - t) * w)], 60);
    // 云尾：从一头的尖往外拖一缕，尾梢上卷
    const tx = w / 2 - h * 0.1, tail = samp(t => [tx + t * w * 0.15, -h * 0.02 + Math.sin(t * Math.PI) * h * 0.06 - t * h * 0.03], 16), tw = t => h * 0.07 * (1 - t) ** 1.4 + h * 0.004;
    P.p('M' + [...top, ...tail.slice(1).map(([x, y], i) => [x, y + tw((i + 1) / 16)]), ...tail.slice().reverse().map(([x, y], i) => [x, y - tw(1 - i / 16) * 0.6]), ...bot].map(q => P2(...F(q))).join('L') + 'Z', 'g-cloud l-cloud');
    const [ex, ey] = tail[tail.length - 1];
    P.p('M' + samp(t => { const a = -Math.PI / 2 + t * Math.PI * 1.5, q = h * (0.05 - 0.032 * t); return F([ex + Math.cos(a) * q, ey + h * 0.05 + Math.sin(a) * q]); }, 14).map(q => P2(...q)).join('L'), 'l-cloudline');
    // 内勾
    let inner = '';
    for (const L of lobes) {
      let seg = [];
      const flush = () => { if (seg.length > 3) inner += 'M' + seg.map(q => P2(...F(q))).join('L'); seg = []; };
      for (let k = 0; k <= 30; k++) { const a = Math.PI * (0.06 + 0.88 * k / 30), x = L.x - Math.cos(a) * L.R, yo = L.cy + Math.sin(a) * L.R; if (yo >= topY(x) - h * 0.01) seg.push([L.x - Math.cos(a) * L.R * 0.8, L.cy + Math.sin(a) * L.R * 0.8]); else flush(); }
      flush();
    }
    P.p(inner, 'l-cloud2');
    // 回旋：最大的两枚
    for (const L of lobes.slice().sort((a, b) => b.R - a.R).slice(0, o.curls ?? 2)) {
      const hand = L.x < 0 ? 1 : -1, a0 = hand > 0 ? Math.PI * 0.94 : Math.PI * 0.06;
      P.p('M' + samp(t => { const a = a0 - hand * t * Math.PI * 1.5, q = L.R * (0.8 - 0.62 * t); return F([L.x + Math.cos(a) * q, L.cy + Math.sin(a) * q]); }, 24).map(q => P2(...q)).join('L'), 'l-cloud2');
    }
    return { g: P.s(), box: dir > 0 ? [-w / 2, w / 2 + w * 0.24, -h * 0.5, h * 1.1] : [-w / 2 - w * 0.24, w / 2, -h * 0.5, h * 1.1] };
  }

  /* ── 水面：一块水，上沿岸线微微起伏，可指定左右边界（被别的东西挡住的那边随便伸出去） ── */
  function lake(o = {}) {
    const r = rng(o.seed || 7), x0 = o.x0 ?? -300, x1 = o.x1 ?? 300, h = o.h || 100, n = 30;
    const top = samp(t => [x0 + (x1 - x0) * t, h * (1 - 0.012 * Math.sin(t * 5 + 1) - 0.008 * r())], n);
    const d = `M${P2(x0, 0)}` + smooth(top).replace(/^M/, 'L') + `L${P2(x1, 0)}Z`;
    return { g: `<path class="${o.cls || 'g-lake'}" d="${d}"/>`, box: [x0, x1, 0, h] };
  }

  /* ── 雾团：一团中间浓、四周淡出的白 ── */
  function mistBlob(o = {}) { const w = o.w || 400, h = o.h || 80; return { g: `<ellipse class="g-mistr" cx="0" cy="${N(h / 2)}" rx="${N(w / 2)}" ry="${N(h / 2)}"/>`, box: [-w / 2, w / 2, 0, h] }; }

  /* ── 水纹：一排排细浪，远处细密、近处疏长；有的浓、有的淡 ── */
  function waves(o = {}) {
    const P = Pen(), r = rng(o.seed || 81), w = o.w || 800, h = o.h || 150, s0 = o.s ?? 1;
    let d = '', d2 = '';
    for (let y = h - 3 * s0; y > 2 * s0;) {
      const k = 1 - y / h, len = (9 + k * 30) * s0, gap = (12 + k * 34) * s0, amp = (0.9 + k * 2) * s0;
      for (let x = -w / 2 + r() * gap; x < w / 2 - len; x += len + gap * (0.5 + r() * 0.9)) {
        const m = 2 + (r() < 0.4 ? 1 : 0), sg = len / m; let q = `M${P2(x, y)}`; for (let j = 0; j < m; j++) q += `q${N(sg / 2)} ${N(amp)} ${N(sg)} 0`;
        if (r() < 0.65) d += q; else d2 += q;
      }
      y -= (3.2 + k * 8.5) * s0;
    }
    P.p(d, 'l-wave'); P.p(d2, 'l-wave2');
    return { g: P.s(), box: [-w / 2, w / 2, 0, h] };
  }

  /* ── 坡岸：临水的一块平坡地。顶面平缓起伏（石绿），前沿一道土坡（赭石）落到水边，坡面几道横皴，岸沿苔点，水边几块小石 ── */
  function bank(o = {}) {
    const P = Pen(), r = rng(o.seed || 91), w = o.w || 600, h = o.h || 40, n = 48, end = o.end ?? 0.32;
    const taper = t => Math.min(1, Math.sin(Math.PI * Math.min(1, Math.max(0, t))) / Math.sin(Math.PI * end)) ** 0.5;
    const top = samp(t => [(t - 0.5) * w, h * taper(t) * (0.92 + 0.08 * Math.sin(t * 9 + 1))], n);
    const lip = samp(t => [(t - 0.5) * w, h * taper(t) * (0.42 + 0.1 * Math.sin(t * 6 + 2))], n);
    P.p(smooth(top) + 'L' + smooth(lip.slice().reverse()).slice(1) + 'Z', `${o.top || 'c-bank-t'} l2`);
    P.p(smooth(lip) + `L${P2(w / 2, 0)}L${P2(-w / 2, 0)}Z`, `${o.front || 'c-bank-f'} l2`);
    let cun = ''; for (let i = 0; i < w / 18; i++) { const t = 0.08 + r() * 0.84, x = (t - 0.5) * w, yl = lip[Math.round(t * n)][1], y = yl * (0.25 + r() * 0.6), L = w * (0.02 + r() * 0.04); cun += `M${P2(x, y)}q${N(L * 0.5)} ${N(-h * 0.03)} ${N(L)} ${N(h * 0.01)}`; }
    P.p(cun, 'l-cun');
    let dots = ''; for (let i = 2; i < n - 1; i++) { const [x, y] = lip[i]; if (r() < 0.6) dots += `M${P2(x + (r() - 0.5) * 4, y + 1)}h.01`; }
    P.p(dots, 'l-moss');
    let st = ''; if (o.stones) for (let i = 0; i < w / 60; i++) { const x = (r() - 0.5) * w * 0.9, s = h * (0.06 + r() * 0.06); st += `M${P2(x - s, 0)}q${N(s * 0.2)} ${N(s * 1.4)} ${N(s)} ${N(s * 1.3)}q${N(s * 0.8)} ${N(s * 0.1)} ${N(s)} ${N(-s * 1.3)}Z`; }
    P.p(st, 'c-shilu l3');
    return { g: P.s(), box: [-w / 2, w / 2, 0, h * 1.05], top, lip };
  }

  /* ── 林冠：一片成林的树冠（柏林、杂树）。一团团波状勾边的叶团叠成一带，远的高、暗，近的低、亮；叶团里稀疏鳞纹，底下露几截干 ── */
  function grove(o = {}) {
    const P = Pen(), r = rng(o.seed || 101), w = o.w || 400, h = o.h || 80, kind = o.kind || 'bai';
    const tones = kind === 'bai' ? ['c-bai-d', 'c-bai', 'c-bai-l'] : ['c-tree-d', 'c-tree', 'c-tree-l'];
    const n = Math.round(w / (h * 0.36) * (o.density ?? 1)), cl = [];
    for (let i = 0; i < n; i++) { const x = (r() - 0.5) * w * 0.94, k = 1 - (2 * Math.abs(x) / w) ** 3, rx = h * (0.2 + r() * 0.14), y = h * (0.28 + r() * 0.48) * (0.55 + 0.45 * k); cl.push({ x, y, rx, ry: rx * (0.62 + r() * 0.14), z: r() }); }
    cl.sort((a, b) => b.y - a.y);
    let tr = ''; if (o.trunks !== false) for (const c of cl) if (r() < 0.35) tr += stroke2([[c.x, 0], [c.x + (r() - 0.5) * c.rx * 0.3, c.y - c.ry * 0.3]], c.rx * 0.08, c.rx * 0.05);
    P.p(tr, 'g-bark l3');
    for (const c of cl) {
      const tone = tones[c.y > h * 0.55 ? 0 : c.z < 0.5 ? 1 : 2];
      P.p(clumpPath(c.x, c.y, c.rx, c.ry, r, 9), `${tone} l2`);
      P.p(`M${P2(c.x - c.rx * 0.66, c.y + c.ry * 0.2)}Q${P2(c.x - c.rx * 0.18, c.y + c.ry * 0.95)} ${P2(c.x + c.rx * 0.52, c.y + c.ry * 0.42)}Q${P2(c.x - c.rx * 0.06, c.y + c.ry * 0.5)} ${P2(c.x - c.rx * 0.66, c.y + c.ry * 0.2)}Z`, 'c-light');
      if (o.marks !== false) { let m = ''; const a = c.rx * 0.13; for (let j = 0; j < 5; j++) { const u = c.x + (r() - 0.5) * c.rx * 1.2, v = c.y + (r() - 0.6) * c.ry * 0.9; m += `M${P2(u, v)}q${N(a)} ${N(a * 0.9)} ${N(a * 2)} 0`; } P.p(m, 'l-scale'); }
    }
    return { g: P.s(), box: [-w / 2 - h * 0.3, w / 2 + h * 0.3, 0, h * 1.12] };
  }


  /* ── 连绵山脉：燕山、西山。很多座峰连成一线（峰高峰距随机），按远近换设色；返回山脊线给长城走 ──
     o: { w, h, seed, n 峰数, fill 'g-shan' | 'g-shan-far' | 'g-shan-mid', sharp 峰尖程度, trees } */
  function range(o = {}) {
    const r = rng(o.seed || 31), w = o.w || 1600, h = o.h || 260, n = o.n || 7;
    const peaks = o.peaks || Array.from({ length: n }, (_, i) => { const t = (i + 0.5) / n; return [(t - 0.5) * 2 + (r() - 0.5) * 0.12, 0.45 + r() * 0.55, 0.06 + r() * 0.05]; });
    const L = mountain({ w, h, seed: o.seed || 31, peaks, fill: o.fill || 'g-shan-far', trees: o.trees ?? false });
    return { g: L.g, box: L.box, pts: L.pts };
  }

  /* ── 林木满坡的山丘：万寿山、琼华岛、景山。圆浑的山形，坡上一团团点叶树冠（前亮后暗），间几株松柏 ── */
  function hill(o = {}) {
    const P = Pen(), r = rng(o.seed || 41), w = o.w || 600, h = o.h || 180, n = 60;
    const pts = samp(t => { const x = (t - 0.5) * w, u = (t - 0.5) * 2; return [x, Math.max(0, h * (1 - u * u) ** (o.k ?? 0.9) * (1 + (r() - 0.5) * 0.03))]; }, n);
    pts[0][1] = 0; pts[n][1] = 0;
    P.p(smooth(pts) + 'Z', `${o.fill || 'g-shan-mid'}`); P.p(smooth(pts), 'l1');
    // 树冠：沿山脊一排，坡上散布叶团；远（高）的暗，近（低）的亮
    const crowns = [];
    for (let i = 2; i < n - 1; i += 1) { const [x, y] = pts[i]; if (y < h * 0.12) continue; crowns.push([x + (r() - 0.5) * w * 0.01, y - h * 0.04, w * (0.026 + r() * 0.012)]); }
    const m = Math.round((o.density ?? 1) * w / 8);
    for (let i = 0; i < m; i++) { const x = (r() - 0.5) * w * 0.92, top = yAt(pts, x) ?? 0, y = top * (0.12 + r() * 0.8); if (top < h * 0.1) continue; crowns.push([x, y, w * (0.02 + r() * 0.016)]); }
    crowns.sort((a, b) => b[1] - a[1]);
    for (const [x, y, R] of crowns) {
      const k = y / h, tone = k > 0.62 ? 'c-tree-d' : k > 0.32 ? 'c-tree' : 'c-tree-l';
      P.p(clumpPath(x, y, R, R * 0.7, r, 8), `${tone} l3`);
      P.p(`M${P2(x - R * 0.6, y + R * 0.12)}Q${P2(x - R * 0.15, y + R * 0.7)} ${P2(x + R * 0.48, y + R * 0.3)}Q${P2(x - R * 0.05, y + R * 0.36)} ${P2(x - R * 0.6, y + R * 0.12)}Z`, 'c-light');
    }
    return { g: P.s(), box: [-w / 2, w / 2, 0, h * 1.08], pts };
  }

  /* ── 云带：一长条横云，上沿一枚枚如意云头（中间大、两头小），下沿平，两头回卷；层与层之间用它隔开 ── */
  function cloudBand(o = {}) {
    const P = Pen(), r = rng(o.seed || 51), w = o.w || 600, h = o.h || 40, dir = o.dir || 1;
    const n = o.heads || Math.max(3, Math.round(w / (h * 1.6)));
    // 云身：扁长圆角条
    const body = `M${P2(-w / 2 + h * 0.4, 0)}H${N(w / 2 - h * 0.4)}A${N(h * 0.4)} ${N(h * 0.4)} 0 0 0 ${P2(w / 2 - h * 0.4, h * 0.8)}H${N(-w / 2 + h * 0.4)}A${N(h * 0.4)} ${N(h * 0.4)} 0 0 0 ${P2(-w / 2 + h * 0.4, 0)}Z`;
    P.p(body, 'g-cloud l-cloud');
    // 云头
    const heads = [];
    for (let i = 0; i < n; i++) { const t = (i + 0.5) / n, k = 1 - Math.abs(t - 0.5) * 1.1; heads.push([(t - 0.5) * w * 0.9 + (r() - 0.5) * h * 0.3, h * (0.75 + k * 0.35), h * (0.42 + k * 0.42) * (0.85 + r() * 0.3), t < 0.5 ? -1 : 1]); }
    heads.sort((a, b) => a[2] - b[2]);
    for (const [cx, cy, rr, hand0] of heads) {
      const hand = hand0 * dir;
      P.c(cx, cy, rr, 'g-cloud');
      const a1 = hand > 0 ? Math.PI + 0.35 : -0.35, span = Math.PI + 0.5, turn = Math.PI * 1.3, th = t => a1 - hand * t;
      const outl = samp(t => [cx + Math.cos(th(t * span)) * rr, cy + Math.sin(th(t * span)) * rr], 24), sp = samp(t => { const q = rr * (1 - 0.78 * t); return [cx + Math.cos(th(span + t * turn)) * q, cy + Math.sin(th(span + t * turn)) * q]; }, 28);
      P.p('M' + outl.concat(sp.slice(1)).map(p => P2(...p)).join('L'), 'l-cloudline');
    }
    // 两头回卷
    for (const s of [-1, 1]) { const cx = s * (w / 2 - h * 0.4), cy = h * 0.4; P.p('M' + samp(t => { const a = (s > 0 ? -Math.PI / 2 : -Math.PI / 2) + s * t * Math.PI * 1.6, q = h * (0.4 - 0.26 * t); return [cx + Math.cos(a) * q, cy + Math.sin(a) * q]; }, 20).map(p => P2(...p)).join('L'), 'l-cloudline'); }
    P.p(`M${P2(-w / 2 + h * 0.6, h * 0.22)}H${N(w / 2 - h * 0.6)}`, 'l-cloud2');
    return { g: P.s(), box: [-w / 2, w / 2, 0, h * 1.9] };
  }

  /* ── 雾：一条上下淡出的白带，把山脚、殿基藏进去（层与层的过渡） ── */
  function mist(o = {}) { const w = o.w || 600, h = o.h || 60; return { g: `<rect class="g-mist" x="${N(-w / 2)}" y="0" width="${N(w)}" height="${N(h)}"/>`, box: [-w / 2, w / 2, 0, h] }; }

  // 松针团：几簇扇形松针攒成一片平展的针团，上沿略拱、下沿平；后簇暗、前簇亮
  function needlePad(P, cx, cy, R, r) {
    const n = 4 + Math.floor(R / 6), bunches = [];
    for (let i = 0; i < n; i++) { const t = (i + 0.5) / n - 0.5, k = 1 - (2 * t) ** 2; bunches.push([cx + t * R * 1.8 + (r() - 0.5) * R * 0.2, cy + R * 0.12 * k + (r() - 0.5) * R * 0.08, R * (0.32 + 0.22 * k) * (0.85 + r() * 0.3), r()]); }
    bunches.sort((a, b) => a[3] - b[3]);
    for (const [x, y, q, z] of bunches) {
      const front = z > 0.45;
      P.p(`M${P2(x - q, y)}A${N(q)} ${N(q * 0.62)} 0 0 0 ${P2(x + q, y)}Q${P2(x, y - q * 0.14)} ${P2(x - q, y)}Z`, front ? 'c-pine l3' : 'c-pine-d l3');
      let nd = ''; for (let k = 0; k <= 16; k++) { const a = Math.PI * (0.03 + 0.94 * k / 16); nd += `M${P2(x, y - q * 0.05)}L${P2(x + Math.cos(a) * q * 0.98, y + Math.sin(a) * q * 0.6)}`; }
      P.p(nd, 'l-pine');
      if (front) P.p(`M${P2(x - q * 0.6, y + q * 0.36)}Q${P2(x - q * 0.1, y + q * 0.62)} ${P2(x + q * 0.44, y + q * 0.44)}`, 'lw');
    }
  }

  /* ── 松枝（框景）：一根老枝从画外斜伸进来，S 形、渐细，分出几杈；针团平展地伏在杈上和枝梢 ──
     原点在枝根（画外），dir = 1 往右伸、-1 往左伸；len 枝长 */
  function pineBough(o = {}) {
    const P = Pen(), r = rng(o.seed || 61), L = o.len || 400, dir = o.dir || 1, droop = o.droop ?? 0.18;
    const main = samp(t => [dir * L * t, -L * droop * Math.sin(Math.PI * t * 0.85) + L * 0.1 * t * t * t + L * 0.03 * Math.sin(t * 9)], 26);
    const twigs = [];
    for (const [t, ang, len] of o.twigs || [[0.36, -0.55, 0.2], [0.5, 0.5, 0.18], [0.66, -0.35, 0.16], [0.78, 0.6, 0.14]]) {
      const i = Math.round(t * (main.length - 1)), [x0, y0] = main[i];
      twigs.push(samp(u => [x0 + dir * L * len * u * Math.cos(ang), y0 + L * len * u * Math.sin(ang) + L * 0.03 * Math.sin(u * 3)], 8));
    }
    for (const tw of twigs) P.p(stroke2(tw, L * 0.009, L * 0.003), 'g-bark l2');
    P.p(stroke2(main, L * 0.026, L * 0.006), 'g-bark l1');
    let sc = ''; for (let i = 1; i < main.length - 6; i++) { const [x, y] = main[i], ww = L * 0.022 * (1 - i / main.length); sc += `M${P2(x - ww * 0.6, y + (r() - 0.5) * ww * 0.5)}q${N(ww * 0.5)} ${N(ww * 0.6)} ${N(ww * 1.2)} 0`; }
    P.p(sc, 'l3');
    const pads = [];
    for (const tw of twigs) { const [x, y] = tw[tw.length - 1], [x2, y2] = tw[3]; pads.push([x, y + L * 0.006, L * (0.075 + r() * 0.02)], [x2 + dir * L * 0.01, y2 + L * 0.012, L * (0.05 + r() * 0.015)]); }
    for (const t of [0.86, 1]) { const [x, y] = main[Math.round(t * (main.length - 1))]; pads.push([x, y + L * 0.012, L * (t === 1 ? 0.085 : 0.06)]); }
    pads.sort((a, b) => b[1] - a[1]);
    for (const [cx, cy, R] of pads) needlePad(P, cx, cy, R, r);
    const xs = pads.map(c => c[0]).concat(main.map(p => p[0])), ys = pads.map(c => c[1]).concat(main.map(p => p[1]));
    return { g: P.s(), box: [Math.min(...xs) - L * 0.1, Math.max(...xs) + L * 0.1, Math.min(...ys) - L * 0.06, Math.max(...ys) + L * 0.08] };
  }

  /* ── 晟帆：一叶小舟。船身两头翘，中段篷舱，桅上一面硬帆（帆骨一道道），帆上朱印"晟帆"；船尾一人摇橹 ──
     原点在船底中点，船长约 120 */
  function boat(o = {}) {
    const P = Pen(), dir = o.dir || 1, X = x => x * dir;
    P.p(`M${P2(X(-60), 10)}C${P2(X(-52), 2)} ${P2(X(-30), 0)} 0 0C${P2(X(30), 0)} ${P2(X(52), 2)} ${P2(X(64), 14)}L${P2(X(58), 15)}C${P2(X(40), 10)} ${P2(X(-40), 10)} ${P2(X(-56), 14)}Z`, 'c-wood l2');
    P.p(`M${P2(X(-56), 12)}C${P2(X(-40), 8.5)} ${P2(X(40), 8.5)} ${P2(X(60), 13)}`, 'l-hull');
    // 篷舱
    P.p(`M${P2(X(-40), 10)}V18C${P2(X(-36), 30)} ${P2(X(-12), 30)} ${P2(X(-8), 18)}V10Z`, 'c-mat l2');
    P.p(`M${P2(X(-36), 11)}V24M${P2(X(-30), 11)}V27M${P2(X(-24), 11)}V27M${P2(X(-18), 11)}V26M${P2(X(-12), 11)}V22`, 'l-mat');
    // 桅、帆
    P.p(`M${P2(X(10), 10)}V128`, 'l-mast');
    P.p(`M${P2(X(-14), 120)}C${P2(X(6), 125)} ${P2(X(30), 128)} ${P2(X(50), 126)}C${P2(X(53), 94)} ${P2(X(53), 60)} ${P2(X(50), 26)}C${P2(X(28), 24)} ${P2(X(4), 24)} ${P2(X(-20), 26)}C${P2(X(-22), 56)} ${P2(X(-20), 92)} ${P2(X(-14), 120)}Z`, 'c-sail l2');
    let bt = ''; for (let y = 38; y < 120; y += 15) bt += `M${P2(X(-19 + (y - 26) * 0.06), y)}C${P2(X(4), y + 2)} ${P2(X(30), y + 3)} ${P2(X(51), y + 2)}`;
    P.p(bt, 'l-batten');
    P.r(X(14) - (dir < 0 ? 17 : 0), 62, 17, 21, 'c-seal');
    P.raw(`<g transform="translate(${N(X(22.5))} 0) scale(1 -1)"><text class="t-seal" x="0" y="-75">晟</text><text class="t-seal" x="0" y="-67">帆</text></g>`);
    // 船尾摇橹人
    P.p(`M${P2(X(-50), 14)}L${P2(X(-49), 30)}Q${P2(X(-47), 34)} ${P2(X(-44), 30)}L${P2(X(-44), 14)}Z`, 'c-robe-b l3');
    P.p(`M${P2(X(-53), 31)}L${P2(X(-46.5), 38)}L${P2(X(-40), 31)}Z`, 'c-hat l3');
    P.p(`M${P2(X(-45), 26)}L${P2(X(-74), 2)}`, 'l-oar');
    return { g: P.s(), box: dir > 0 ? [-80, 70, -2, 132] : [-70, 80, -2, 132] };
  }

  /* ── 仙鹤：工笔丹顶鹤飞姿。白身、黑颈、丹顶，翼白而后缘一道黑羽，腿往后拖。pose: 'up' | 'flat' | 'down' ── */
  function crane(o = {}) {
    const P = Pen(), dir = o.dir || 1, X = x => x * dir, pose = o.pose || 'flat';
    const wy = pose === 'up' ? 1 : pose === 'down' ? -0.7 : 0.15;
    // 远侧翅
    const wing = (k, far) => {
      const tipY = 30 * wy * k, tipX = -6 + 14 * (1 - Math.abs(wy)) * k;
      const d = `M${P2(X(4), 2)}C${P2(X(2 + tipX * 0.4), 2 + tipY * 0.6)} ${P2(X(tipX - 6), tipY + 6)} ${P2(X(tipX), tipY + 8)}C${P2(X(tipX - 8), tipY + 4)} ${P2(X(tipX - 18), tipY - 2)} ${P2(X(-14), 0)}Z`;
      const tr = `M${P2(X(tipX), tipY + 8)}C${P2(X(tipX - 8), tipY + 4)} ${P2(X(tipX - 18), tipY - 2)} ${P2(X(-14), 0)}L${P2(X(-8), 1)}C${P2(X(tipX - 12), tipY - 0.5)} ${P2(X(tipX - 5), tipY + 3.5)} ${P2(X(tipX - 1), tipY + 6)}Z`;
      P.p(d, far ? 'c-crane-d l3' : 'c-crane l3'); P.p(tr, 'c-ink');
    };
    wing(0.9, true);
    // 身、颈、头、腿
    P.p(`M${P2(X(-16), -2)}C${P2(X(-10), -6)} ${P2(X(8), -6)} ${P2(X(14), 0)}C${P2(X(8), 4)} ${P2(X(-8), 5)} ${P2(X(-16), -2)}Z`, 'c-crane l3');
    P.p(`M${P2(X(12), 0)}C${P2(X(20), 1)} ${P2(X(28), 3)} ${P2(X(36), 3.5)}`, 'l-neck');
    P.p(`M${P2(X(35), 4.6)}C${P2(X(38), 6)} ${P2(X(40), 5)} ${P2(X(41), 3.6)}L${P2(X(48), 2.4)}L${P2(X(41), 2.2)}C${P2(X(39), 1.6)} ${P2(X(36), 1.8)} ${P2(X(35), 2.4)}Z`, 'c-crane l3');
    P.c(X(38.2), 4.6, 1.1, 'c-red');
    P.p(`M${P2(X(-14), -2)}L${P2(X(-34), -6)}M${P2(X(-12), -1)}L${P2(X(-33), -3.6)}`, 'l-leg');
    P.p(`M${P2(X(-16), -1)}C${P2(X(-20), 0)} ${P2(X(-22), -3)} ${P2(X(-18), -4)}Z`, 'c-ink');
    wing(1, false);
    return { g: P.s(), box: dir > 0 ? [-36, 50, -10, 40] : [-50, 36, -10, 40] };
  }

  window.JH.NATURE = { pine, cypress, willow, dianye, maple, mountain, rock, water, cloud, person, stroke2, taper, range, hill, cloudBand, mist, pineBough, boat, crane, cloud2, mistBlob, waves, bank, grove, clumpPath, lake };
})();
