/* 京华胜景 · 青绿长卷。画面坐标 4400×1000（y 向下），水岸线 WY。
   分四层，滚动时各层速度不同：远山（far）、中景（mid）、主景（main）、近景框景（fore）。 */
(function () {
  'use strict';
  const A = window.ART2, { N, pt, poly, rng, Pen, sub } = A;
  const W = 4400, H = 1000, WY = 770;

  // Catmull-Rom 平滑
  function smooth(pts) {
    let d = `M${pt(pts[0])}`;
    for (let i = 0; i < pts.length - 1; i++) {
      const p0 = pts[i - 1] || pts[i], p1 = pts[i], p2 = pts[i + 1], p3 = pts[i + 2] || p2;
      d += `C${N(p1[0] + (p2[0] - p0[0]) / 6)} ${N(p1[1] + (p2[1] - p0[1]) / 6)} ${N(p2[0] - (p3[0] - p1[0]) / 6)} ${N(p2[1] - (p3[1] - p1[1]) / 6)} ${pt(p2)}`;
    }
    return d;
  }
  const bell = (t, c, w) => { const x = (t - c) / w; return Math.abs(x) >= 1 ? 0 : Math.pow(1 - x * x, 1.35); };

  /* ── 青绿山 ───────────────────────────────────────
     一座山 = 主峰 + 两侧肩峰；石青顶、石绿腰、赭石脚；披麻皴、点苔、坡上小树。 */
  function mountain(P, o) {
    const { x, base, h, w, seed = 1, grad = 'g-mtnMid', cun = true, moss = true, trees = 0, cls = '' } = o;
    const r = rng(seed), n = 46, pts = [];
    const sh1 = 0.5 + r() * 0.2, sh2 = 0.45 + r() * 0.2, c1 = -0.42 - r() * 0.12, c2 = 0.4 + r() * 0.15;
    for (let i = 0; i <= n; i++) {
      const t = -1 + 2 * i / n;
      let f = Math.max(bell(t, 0, 0.62), sh1 * bell(t, c1, 0.42), sh2 * bell(t, c2, 0.45), 0.18 * bell(t, 0, 1));
      f += (r() - 0.5) * 0.045 * (1 - Math.abs(t));
      pts.push([x + t * w, base - Math.max(0, f) * h]);
    }
    pts[0][1] = base; pts[n][1] = base;
    const d = smooth(pts) + 'Z';
    P.p(d, `${cls} mtn ${grad}`, ` fill="url(#${grad})"`);
    if (cun) {
      // 披麻皴：沿轮廓取点，每处两三笔短弧，顺坡往下，越往山脚越淡越短
      let cd = '';
      const peakX = [x, x + c1 * w, x + c2 * w];
      for (let i = 2; i < n - 1; i += 2) {
        const [px, py] = pts[i]; if (py > base - h * 0.22) continue;
        const near = peakX.reduce((a, b) => Math.abs(px - b) < Math.abs(px - a) ? b : a), dir = px < near ? -1 : 1;
        const len = (base - py) * (0.18 + r() * 0.16);
        for (let k = 0; k < 3; k++) {
          const sx = px + dir * k * 2.2 - dir * 2, sy = py + 3 + k * 2.6;
          cd += `M${N(sx)} ${N(sy)}q${N(dir * len * 0.18)} ${N(len * 0.45)} ${N(dir * len * (0.28 + r() * 0.12))} ${N(len * (0.9 - k * 0.18))}`;
        }
      }
      // 山脚横向矾头小皴
      for (let k = 0; k < 7; k++) { const yy = base - h * (0.05 + r() * 0.12), xx = x + (r() - 0.5) * w * 1.3, ll = w * (0.05 + r() * 0.08); cd += `M${N(xx - ll)} ${N(yy)}q${N(ll)} ${N(-2.4)} ${N(ll * 2)} 0`; }
      P.p(cd, `${cls} cun`);
    }
    if (moss) {
      let md = '';
      pts.forEach(([px, py], i) => { if (i % 2 || py > base - h * 0.35) return; for (let k = 0; k < 2; k++) { const dx = (r() - 0.5) * 8, dy = 2 + r() * 6, rr = 1 + r() * 1.4; md += `M${N(px + dx - rr)} ${N(py + dy)}a${N(rr)} ${N(rr)} 0 1 0 ${N(rr * 2)} 0a${N(rr)} ${N(rr)} 0 1 0 ${N(-rr * 2)} 0Z`; } });
      P.p(md, `${cls} moss`);
    }
    if (trees) {
      // 坡上的树按丛长：每丛三五棵，同一树种
      const B = window.FLORA.Bucket(), list = [];
      for (let i = 0; i < trees; i++) {
        const t = -0.82 + r() * 1.64, top = pts[Math.round((t + 1) / 2 * n)][1], cy = top + (base - top) * (0.22 + r() * 0.55);
        const kind = 'pepgay'[Math.floor(r() * 6)], k = 2 + Math.floor(r() * 3);
        for (let j = 0; j < k; j++) list.push([x + t * w * 0.9 + (j - k / 2) * 5 + r() * 3, cy + (r() - 0.5) * 6, 9 + r() * 7, kind]);
      }
      list.sort((a, b) => a[1] - b[1]).forEach(([px, py, h, kind]) => window.FLORA.tree(B, px, py, h, kind, r));
      B.flush(P);
    }
    return pts;
  }
  function domeHill(cx, base, hw, hh, seed) {
    const r = rng(seed), pts = [];
    for (let i = 0; i <= 64; i++) {
      const t = -1 + 2 * i / 64;
      let f = Math.pow(Math.max(0, 1 - Math.pow(Math.abs(t), 2.3)), 0.85);
      f *= 1 + 0.05 * Math.sin(t * 8.5 + 1.3) * (1 - t * t) - 0.07 * Math.max(0, t);
      pts.push([cx + t * hw, base - f * hh + (r() - 0.5) * 2]);
    }
    pts[0][1] = base; pts[64][1] = base;
    return pts;
  }
  function tinyTree(P, x, y, h, r) {
    if (r() < 0.55) P.p(`M${N(x)} ${N(y)}V${N(y - h * 0.3)}M${N(x - h * 0.32)} ${N(y - h * 0.3)}L${N(x)} ${N(y - h)}L${N(x + h * 0.32)} ${N(y - h * 0.3)}Z`, 'tree-s');
    else P.p(`M${N(x)} ${N(y)}V${N(y - h * 0.35)}M${N(x - h * 0.35)} ${N(y - h * 0.5)}a${N(h * 0.35)} ${N(h * 0.3)} 0 1 0 ${N(h * 0.7)} 0a${N(h * 0.35)} ${N(h * 0.3)} 0 1 0 ${N(-h * 0.7)} 0Z`, 'tree-s t-dot');
  }

  /* ── 树 ─────────────────────────────────────────── */
  // 一团树叶：边缘是小弧组成的花边，里面撒点叶，下半部压暗
  function foliage(cx, cy, rx, ry, r, bumps = 14) {
    const pts = [];
    for (let i = 0; i < bumps; i++) { const a = i / bumps * Math.PI * 2, k = 1 + (r() - 0.5) * 0.22; pts.push([cx + Math.cos(a) * rx * k, cy + Math.sin(a) * ry * k]); }
    let d = `M${pt(pts[0])}`;
    for (let i = 0; i < bumps; i++) {
      const a = pts[i], b = pts[(i + 1) % bumps], mx = (a[0] + b[0]) / 2, my = (a[1] + b[1]) / 2;
      const ox = (mx - cx) * 0.22, oy = (my - cy) * 0.22;
      d += `Q${N(mx + ox)} ${N(my + oy)} ${pt(b)}`;
    }
    return d + 'Z';
  }
  function dots(cx, cy, rx, ry, n, r) {
    let d = '';
    for (let i = 0; i < n; i++) { const a = r() * Math.PI * 2, t = Math.sqrt(r()); d += `M${N(cx + Math.cos(a) * rx * t)} ${N(cy + Math.sin(a) * ry * t)}h.01`; }
    return d;
  }
  // 落叶树：两三团树冠叠成一棵，颜色随季节（kind: a 枫 / b 银杏 / g 常绿）
  function crown(P, x, y, rr, kind, seed) {
    const r = rng(seed);
    P.p(`M${N(x)} ${N(y)}Q${N(x - 1)} ${N(y - rr * 0.5)} ${N(x + (r() - 0.5) * 3)} ${N(y - rr * 0.95)}M${N(x)} ${N(y - rr * 0.5)}l${N(rr * 0.35)} ${N(-rr * 0.35)}`, 'branch');
    const parts = [[0, -1.45, 1, 0.82], [-0.62, -1.1, 0.7, 0.6], [0.66, -1.12, 0.7, 0.6], [0.05, -1.95, 0.68, 0.55]];
    let d = '', dk = '';
    for (const [dx, dy, sx, sy] of parts) { d += foliage(x + dx * rr, y + dy * rr, sx * rr, sy * rr, r, 12); }
    P.p(d, `crown t-${kind}`);
    for (const [dx, dy, sx, sy] of parts) dk += `M${N(x + dx * rr - sx * rr * 0.8)} ${N(y + dy * rr + sy * rr * 0.15)}q${N(sx * rr * 0.8)} ${N(sy * rr * 0.7)} ${N(sx * rr * 1.6)} 0`;
    P.p(dk, 'crown-sh');
    P.p(dots(x, y - rr * 1.5, rr * 1.25, rr * 0.95, 26, r), 'leafdot');
  }
  // 树林：一片前后叠压的点叶树冠，颜色压低饱和度，下缘压暗，用来把建筑"种"进山水里
  function grove(P, x0, x1, y, rr, kinds, seed, rows = 2) {
    const r = rng(seed), items = [];
    for (let row = 0; row < rows; row++) for (let x = x0 + r() * rr; x < x1; x += rr * (0.8 + r() * 0.6)) items.push([x, y + row * rr * 0.42 + (r() - 0.5) * rr * 0.4, rr * (0.6 + r() * 0.6), kinds[Math.floor(r() * kinds.length)]]);
    items.sort((a, b) => a[1] - b[1]);
    // 按前后分两层合并：后排先画，同色一条路径
    const half = Math.ceil(items.length / 2);
    for (const part of [items.slice(0, half), items.slice(half)]) {
      const by = {}; let br = '', sh = '', dt = '';
      for (const [x, yy, R, k] of part) {
        if (r() < 0.35) br += `M${N(x)} ${N(yy)}V${N(yy - R * 0.9)}`;
        by[k] = (by[k] || '') + foliage(x, yy - R * 1.05, R, R * 0.78, r, 11);
        sh += `M${N(x - R * 0.85)} ${N(yy - R * 0.9)}q${N(R * 0.85)} ${N(R * 0.62)} ${N(R * 1.7)} 0`;
        dt += dots(x, yy - R * 1.08, R * 0.85, R * 0.6, 9, r);
      }
      P.p(br, 'branch');
      for (const k of Object.keys(by)) P.p(by[k], `crown t-${k}`);
      P.p(sh, 'crown-sh'); P.p(dt, 'leafdot');
    }
  }
  // 松：虬曲的干（鳞状树皮），横出的枝，枝头一簇簇扇形松针
  function needleFan(cx, cy, R, r, n = 11) {
    let fill = `M${N(cx - R)} ${N(cy)}A${N(R)} ${N(R * 0.5)} 0 0 1 ${N(cx + R)} ${N(cy)}Z`, ln = '';
    for (let k = 0; k <= n; k++) { const a = Math.PI + Math.PI * k / n + (r() - 0.5) * 0.08; ln += `M${N(cx)} ${N(cy + 1)}L${N(cx + Math.cos(a) * R)} ${N(cy + Math.sin(a) * R * 0.52)}`; }
    return [fill, ln];
  }
  function pine(P, x, y, h, seed, lean = 1) {
    const r = rng(seed), L = lean;
    const spine = [[x, y], [x + L * h * 0.07, y - h * 0.22], [x - L * h * 0.05, y - h * 0.46], [x + L * h * 0.1, y - h * 0.7], [x + L * h * 0.04, y - h * 0.9], [x + L * h * 0.09, y - h]];
    const tw = i => h * 0.04 * (1 - i / spine.length * 0.75);
    const left = spine.map(([px, py], i) => [px - tw(i), py]), right = spine.map(([px, py], i) => [px + tw(i), py]);
    P.p(smooth(left) + 'L' + smooth(right.slice().reverse()).slice(1) + 'Z', 'trunk-g');
    let scales = '';
    for (let i = 0; i < 26; i++) { const t = r() * 0.92, idx = Math.min(spine.length - 2, Math.floor(t * (spine.length - 1))), f = t * (spine.length - 1) - idx, a = spine[idx], b = spine[idx + 1], px = a[0] + (b[0] - a[0]) * f + (r() - 0.5) * tw(idx) * 1.2, py = a[1] + (b[1] - a[1]) * f, w = tw(idx) * 0.55; scales += `M${N(px - w)} ${N(py)}q${N(w)} ${N(-w * 0.9)} ${N(w * 2)} 0`; }
    P.p(scales, 'bark');
    const limbs = [[0.34, -1, 0.5, -0.04], [0.46, 1, 0.44, 0.03], [0.58, -1, 0.4, -0.05], [0.7, 1, 0.36, -0.02], [0.82, -1, 0.28, -0.06], [0.93, 1, 0.22, -0.04]];
    let fills = '', lines = '', lite = '';
    for (const [t, side, len, rise] of limbs) {
      const idx = Math.min(spine.length - 1, Math.round(t * (spine.length - 1))), sp = spine[idx];
      const by = y - h * t, ex = sp[0] + side * L * h * len * (side === L ? 1 : 0.85), ey = by + h * rise;
      P.p(`M${N(sp[0])} ${N(by)}C${N(sp[0] + (ex - sp[0]) * 0.35)} ${N(by - h * 0.03)} ${N(sp[0] + (ex - sp[0]) * 0.7)} ${N(ey + h * 0.04)} ${N(ex)} ${N(ey)}`, 'limb');
      const span = Math.abs(ex - sp[0]), nf = 3 + Math.round(span / (h * 0.09));
      for (let k = 0; k < nf; k++) {
        const f = (k + 0.5) / nf, cx = sp[0] + (ex - sp[0]) * (0.25 + f * 0.8), cy = by + (ey - by) * f - h * (0.01 + r() * 0.02), R = h * (0.05 + r() * 0.025);
        const [fl, ln] = needleFan(cx, cy, R, r); fills += fl; lines += ln;
        if (r() < 0.6) { const [fl2, ln2] = needleFan(cx + (r() - 0.5) * R, cy - R * 0.32, R * 0.75, r, 9); fills += fl2; lite += ln2; }
      }
    }
    P.p(fills, 'pine'); P.p(lines, 'needle'); P.p(lite, 'needle-l');
  }

  // 古柏：几团深绿树叶上下叠成一株，不对称
  function cypress(P, x, y, h, seed) {
    const r = rng(seed);
    P.p(`M${N(x - 1.5)} ${N(y)}Q${N(x + 2)} ${N(y - h * 0.15)} ${N(x - 1)} ${N(y - h * 0.3)}`, 'trunkL');
    let d = '';
    for (let i = 0; i < 6; i++) {
      const t = i / 5, cy = y - h * (0.28 + t * 0.6), w = h * (0.2 - t * 0.09) * (0.85 + r() * 0.3), cx = x + (r() - 0.5) * h * 0.08;
      d += foliage(cx, cy, w, h * 0.13, r, 12);
    }
    P.p(d, 'cypress');
    P.p(dots(x, y - h * 0.6, h * 0.15, h * 0.36, 30, r), 'leafdot');
  }
  function willow(P, x, y, h, seed) {
    const r = rng(seed);
    P.p(`M${N(x - 2)} ${N(y)}C${N(x + 4)} ${N(y - h * 0.3)} ${N(x - 6)} ${N(y - h * 0.55)} ${N(x + 2)} ${N(y - h * 0.8)}L${N(x + 5)} ${N(y - h * 0.8)}C${N(x - 2)} ${N(y - h * 0.55)} ${N(x + 8)} ${N(y - h * 0.3)} ${N(x + 3)} ${N(y)}Z`, 'trunk');
    P.p(foliage(x + 2, y - h * 0.82, h * 0.32, h * 0.14, r, 12), 'willow-c');
    let sd = '';
    for (let i = 0; i < 34; i++) {
      const sx = x + 2 + (r() - 0.5) * h * 0.62, sy = y - h * (0.76 + r() * 0.16), len = h * (0.32 + r() * 0.42), sw = (r() - 0.5) * 10;
      sd += `M${N(sx)} ${N(sy)}Q${N(sx + sw)} ${N(sy + len * 0.5)} ${N(sx + sw * 0.6)} ${N(sy + len)}`;
    }
    P.p(sd, 'willow');
  }
  // 垂柳框景：一根老枝从画外斜伸进来，分出几股细枝；柳丝成簇垂下，长短参差，梢头微微后飘；丝上对生细叶
  function hangingWillow(P, x, y, span, len, seed) {
    const r = rng(seed);
    const A0 = [x - span * 0.75, y - 46], C0 = [x - span * 0.15, y - 6], B0 = [x + span * 0.5, y + 26];
    const q = (t, A = A0, C = C0, B = B0) => [(1 - t) ** 2 * A[0] + 2 * (1 - t) * t * C[0] + t * t * B[0], (1 - t) ** 2 * A[1] + 2 * (1 - t) * t * C[1] + t * t * B[1]];
    // 老枝：有粗细
    const L = [], R2 = [];
    for (let i = 0; i <= 16; i++) { const t = i / 16, [px, py] = q(t), w = 5.2 * (1 - t * 0.8); L.push([px, py - w]); R2.push([px, py + w * 0.6]); }
    P.p('M' + L.map(pt).join('L') + 'L' + R2.reverse().map(pt).join('L') + 'Z', 'willow-limbF');
    // 几股细枝
    const twigs = [];
    let tw = '';
    for (const t of [0.32, 0.5, 0.66, 0.8, 0.94]) {
      const [sx, sy] = q(t), ex = sx + 18 + r() * 34, ey = sy + 8 + r() * 16;
      tw += `M${N(sx)} ${N(sy)}Q${N((sx + ex) / 2)} ${N(sy - 4)} ${N(ex)} ${N(ey)}`;
      twigs.push([[sx, sy], [(sx + ex) / 2, sy - 4], [ex, ey]]);
    }
    P.p(tw, 'willow-twig');
    let d = '', lf = '', lf2 = '';
    // 柳丝：一簇一簇挂在细枝和老枝上
    const hang = (sx, sy, k) => {
      for (let i = 0; i < k; i++) {
        const ox = sx + (r() - 0.5) * 16, L2 = len * (0.25 + Math.pow(r(), 0.7) * 0.85), drift = 6 + r() * 16;
        const c1 = [ox + 2, sy + L2 * 0.32], c2 = [ox + drift * 0.4, sy + L2 * 0.72], e = [ox + drift, sy + L2 + r() * 6];
        d += `M${N(ox)} ${N(sy)}C${pt(c1)} ${pt(c2)} ${pt(e)}`;
        const nl = Math.round(L2 / 7.5);
        for (let j = 1; j <= nl; j++) {
          const u = j / (nl + 1), v = 1 - u, bx = v ** 3 * ox + 3 * v * v * u * c1[0] + 3 * v * u * u * c2[0] + u ** 3 * e[0], by = v ** 3 * sy + 3 * v * v * u * c1[1] + 3 * v * u * u * c2[1] + u ** 3 * e[1];
          const side = j % 2 ? 1 : -1, ll = (4.2 - u * 1.6) * (0.8 + r() * 0.4);
          const seg = `M${N(bx)} ${N(by)}q${N(side * ll * 0.5)} ${N(ll * 0.35)} ${N(side * ll * 0.4)} ${N(ll)}`;
          if (r() < 0.32) lf2 += seg; else lf += seg;
        }
      }
    };
    for (const [a1, c1, e1] of twigs) for (let k = 0; k < 3; k++) { const t = 0.35 + k * 0.3, px = (1 - t) ** 2 * a1[0] + 2 * (1 - t) * t * c1[0] + t * t * e1[0], py = (1 - t) ** 2 * a1[1] + 2 * (1 - t) * t * c1[1] + t * t * e1[1]; hang(px, py, 3 + Math.floor(r() * 3)); }
    for (const t of [0.2, 0.42, 0.58, 0.74, 0.88, 1]) { const [px, py] = q(t); hang(px, py + 2, 2 + Math.floor(r() * 3)); }
    P.p(d, 'willow-s'); P.p(lf, 'willow-lf'); P.p(lf2, 'willow-lf2');
  }
  // 荷（夏）：浮叶、立叶、荷花、花苞
  function lotus(P, x, y, s, seed) {
    const r = rng(seed); let fl = '', vein = '', st = '', up = '', pet = '', pet2 = '';
    for (let i = 0; i < 7; i++) {
      const lx = x + (r() - 0.5) * 120 * s, ly = y + (r() - 0.3) * 16 * s, rx = (10 + r() * 9) * s, ry = rx * 0.3;
      fl += `M${N(lx - rx)} ${N(ly)}a${N(rx)} ${N(ry)} 0 1 0 ${N(rx * 2)} 0a${N(rx)} ${N(ry)} 0 1 0 ${N(-rx * 2)} 0Z`;
      for (let k = 0; k < 5; k++) { const a = Math.PI * (0.15 + 0.7 * k / 4); vein += `M${N(lx)} ${N(ly)}l${N(Math.cos(a) * rx * 0.9)} ${N(-Math.sin(a) * ry * 0.9)}`; }
    }
    for (let i = 0; i < 3; i++) {
      const sx = x + (r() - 0.5) * 90 * s, h = (24 + r() * 22) * s, w = (11 + r() * 5) * s;
      st += `M${N(sx)} ${N(y + 2 * s)}q${N((r() - 0.5) * 6 * s)} ${N(-h * 0.5)} 0 ${N(-h)}`;
      up += `M${N(sx - w)} ${N(y - h - w * 0.35)}Q${N(sx)} ${N(y - h + w * 0.5)} ${N(sx + w)} ${N(y - h - w * 0.35)}Q${N(sx)} ${N(y - h - w * 0.15)} ${N(sx - w)} ${N(y - h - w * 0.35)}Z`;
    }
    for (let i = 0; i < 2; i++) {
      const fx = x + (r() - 0.5) * 70 * s, h = (30 + r() * 18) * s, fy = y - h, R = 7 * s;
      st += `M${N(fx)} ${N(y + 2 * s)}q${N((r() - 0.5) * 5 * s)} ${N(-h * 0.5)} 0 ${N(-h)}`;
      for (const [a, k] of [[-0.9, 0.85], [0.9, 0.85], [-0.4, 1], [0.4, 1], [0, 1.08]]) { const tx = fx + Math.sin(a) * R * 0.9, ty = fy - Math.cos(a) * R * k * 0.9; const seg = `M${N(fx)} ${N(fy + R * 0.2)}Q${N(fx + Math.sin(a) * R * 0.9 - R * 0.35)} ${N(fy - R * 0.3)} ${N(tx)} ${N(ty)}Q${N(fx + Math.sin(a) * R * 0.9 + R * 0.35)} ${N(fy - R * 0.3)} ${N(fx)} ${N(fy + R * 0.2)}Z`; if (Math.abs(a) > 0.5) pet2 += seg; else pet += seg; }
    }
    const bx = x + (r() - 0.5) * 60 * s, bh = 40 * s;
    st += `M${N(bx)} ${N(y + 2 * s)}q${N(3 * s)} ${N(-bh * 0.5)} 0 ${N(-bh)}`;
    pet += `M${N(bx)} ${N(y - bh + 1)}Q${N(bx - 4 * s)} ${N(y - bh - 5 * s)} ${N(bx)} ${N(y - bh - 11 * s)}Q${N(bx + 4 * s)} ${N(y - bh - 5 * s)} ${N(bx)} ${N(y - bh + 1)}Z`;
    P.p(fl, 'lt-leaf'); P.p(vein, 'lt-vein'); P.p(st, 'lt-stem'); P.p(up, 'lt-up'); P.p(pet2, 'lt-pet2'); P.p(pet, 'lt-pet');
  }
  // 雁阵（秋）：人字形，翅膀起落
  function geese(x, y, n, s, seed) {
    const r = rng(seed); let g = '';
    for (let i = 0; i < n; i++) {
      const k = Math.ceil(i / 2), side = i % 2 ? 1 : -1, bx = x - k * 26 * s, by = y + side * k * 13 * s, sc = s * (0.9 + r() * 0.2);
      const up = `M${N(bx - 10 * sc)} ${N(by - 3 * sc)}Q${N(bx - 4 * sc)} ${N(by - 5 * sc)} ${N(bx)} ${N(by)}Q${N(bx - 3 * sc)} ${N(by + 5 * sc)} ${N(bx - 9 * sc)} ${N(by + 4 * sc)}`;
      const dn = `M${N(bx - 10 * sc)} ${N(by + 1 * sc)}Q${N(bx - 4 * sc)} ${N(by - 1 * sc)} ${N(bx)} ${N(by)}Q${N(bx - 3 * sc)} ${N(by + 1 * sc)} ${N(bx - 9 * sc)} ${N(by + 2 * sc)}`;
      g += `<path class="goose" d="${up}"><animate attributeName="d" dur="${(1.3 + r() * 0.3).toFixed(2)}s" begin="-${r().toFixed(2)}s" repeatCount="indefinite" values="${up};${dn};${up}"/></path><path class="goose-b" d="M${N(bx - 2 * sc)} ${N(by)}L${N(bx + 6 * sc)} ${N(by - 1 * sc)}"/>`;
    }
    return `<g class="geese">${g}</g>`;
  }
  // 太湖石
  function rock(P, x, y, w, h, seed) {
    const r = rng(seed), pts = [];
    for (let i = 0; i <= 10; i++) { const t = i / 10; pts.push([x - w / 2 + w * t, y - h * Math.sin(Math.PI * t) * (0.75 + r() * 0.4)]); }
    P.p(smooth(pts) + `L${N(x + w / 2)} ${N(y)}Z`, 'rock');
    let c = ''; for (let k = 0; k < 4; k++) { const cx = x + (r() - 0.5) * w * 0.6, cy = y - h * (0.25 + r() * 0.4); c += `M${N(cx - w * 0.12)} ${N(cy)}q${N(w * 0.1)} ${N(-h * 0.12)} ${N(w * 0.22)} ${N(h * 0.04)}`; }
    P.p(c, 'cun');
    P.e(x + (r() - 0.5) * w * 0.3, y - h * 0.45, w * 0.06, h * 0.08, 'rock-hole');
  }
  function reeds(P, x, y, n, seed) {
    const r = rng(seed); let d = '';
    for (let i = 0; i < n; i++) { const bx = x + i * 5 + r() * 3, hh = 22 + r() * 22, bend = (r() - 0.3) * 10; d += `M${N(bx)} ${N(y)}q${N(bend * 0.3)} ${N(-hh * 0.6)} ${N(bend)} ${N(-hh)}`; }
    P.p(d, 'reed');
  }
  // 岸：一块赭石色的土坡，上沿起伏
  function bank(P, x0, x1, top, bottom, seed) {
    const r = rng(seed), pts = [];
    for (let x = x0; x <= x1; x += (x1 - x0) / 16) pts.push([x, top + (r() - 0.5) * 10 + (Math.abs(x - (x0 + x1) / 2) / ((x1 - x0) / 2)) ** 3 * (bottom - top) * 0.7]);
    P.p(smooth(pts) + `L${N(x1)} ${N(bottom)}Q${N((x0 + x1) / 2)} ${N(bottom + 12)} ${N(x0)} ${N(bottom)}Z`, 'bank');
    let g = ''; for (let i = 0; i < 40; i++) { const gx = x0 + r() * (x1 - x0), gy = top + 6 + r() * (bottom - top - 8); g += `M${N(gx)} ${N(gy)}q1.6 -4 3.2 0`; }
    P.p(g, 'grass');
    P.p(smooth(pts), 'bank-l');
  }

  /* ── 水、云、雾、鸟、舟 ───────────────────────── */
  function waves(x0, x1, y0, y1, seed, cls = 'wave') {
    const r = rng(seed); let d = '';
    for (let y = y0 + 9; y < y1 - 4; y += 9 + r() * 4) {
      let x = x0 + r() * 30;
      while (x < x1) {
        const seg = 40 + r() * 90; let xx = x; d += `M${N(xx)} ${N(y)}`;
        while (xx < Math.min(x + seg, x1)) { const w = 12 + r() * 8, h = 1.6 + r() * 1.4; d += `q${N(w / 2)} ${N(-h)} ${N(w)} 0`; xx += w; }
        x = xx + 20 + r() * 60;
      }
    }
    return `<path class="${cls}" d="${d}"/>`;
  }
  function xiangyun(x, y, s, flip = 1) {
    return `<g class="cloud" transform="translate(${x} ${y}) scale(${N(s * flip)} ${N(s)})">
      <path class="cloud-f" d="M-46 10C-52 10-53 2-46-1C-48-10-37-16-28-11C-23-22-6-23 1-14C7-21 22-21 26-11C36-13 44-4 39 4C46 5 47 11 41 12Z"/>
      <path class="cloud-l" d="M-28-11C-28-5-22-2-17-5C-14-8-17-12-21-10M1-14C1-8 6-5 11-7C14-10 11-14 8-12M26-11C25-6 28-2 33-3M-46-1C-42 2-37 1-36-2"/>
      <path class="cloud-l" d="M41 12C58 12 76 9 96 3M-46 10C-60 10-72 12-86 16"/></g>`;
  }
  function mist(x, y, w, h, cls = 'mist') { return `<ellipse class="${cls}" cx="${x}" cy="${y}" rx="${w}" ry="${h}" fill="url(#g-mistR)"/>`; }
  function birds(x, y, n, s, seed) {
    const r = rng(seed); let g = '';
    for (let i = 0; i < n; i++) {
      const bx = x + i * 26 * s + r() * 14, by = y + (i % 2) * 12 * s + r() * 10, k = s * (0.8 + r() * 0.4), dur = (0.9 + r() * 0.5).toFixed(2);
      const up = `M${N(bx - 9 * k)} ${N(by - 4 * k)}Q${N(bx - 4 * k)} ${N(by - 5 * k)} ${N(bx)} ${N(by)}Q${N(bx + 4 * k)} ${N(by - 5 * k)} ${N(bx + 9 * k)} ${N(by - 4 * k)}`;
      const dn = `M${N(bx - 9 * k)} ${N(by + 3 * k)}Q${N(bx - 4 * k)} ${N(by - 1 * k)} ${N(bx)} ${N(by)}Q${N(bx + 4 * k)} ${N(by - 1 * k)} ${N(bx + 9 * k)} ${N(by + 3 * k)}`;
      g += `<path class="bird" d="${up}"><animate attributeName="d" dur="${dur}s" repeatCount="indefinite" values="${up};${dn};${up}" begin="${(r() * 1).toFixed(2)}s"/></path>`;
    }
    return `<g class="birds">${g}</g>`;
  }
  // 仙鹤（取宋徽宗《瑞鹤图》之意）：白身、黑颈、丹顶，次级飞羽黑，长腿后伸；双翅扇动
  function crane(x, y, s, flip, phase, dur) {
    const wUp = 'M3-1C1-9-5-18-15-26C-12-19-11-15-11-11C-9-8-7-4-6 0Z', wDn = 'M3 0C2 6-2 14-9 22C-9 16-10 12-11 9C-9 6-7 3-6 1Z';
    const eUp = 'M-15-26C-12-19-11-15-11-11C-9-8-7-4-6 0', eDn = 'M-9 22C-9 16-10 12-11 9C-9 6-7 3-6 1';
    const an = (a, b) => `<animate attributeName="d" values="${a};${b};${a}" keyTimes="0;.5;1" calcMode="spline" keySplines=".45 0 .55 1;.45 0 .55 1" dur="${dur}s" begin="-${phase}s" repeatCount="indefinite"/>`;
    return `<g class="crane" transform="translate(${N(x)} ${N(y)}) scale(${N(s * flip)} ${N(s)})">` +
      `<g transform="translate(3 -1.5)"><path class="crane-wf" d="${wUp}">${an(wUp, wDn)}</path><path class="crane-e" d="${eUp}">${an(eUp, eDn)}</path></g>` +
      `<path class="crane-leg" d="M-12 2L-28 4.4M-12 2.8L-28 5.8"/>` +
      `<path class="crane-t" d="M-14 1C-17 0-20 0-22 1C-19 2.2-16 3-13 3Z"/>` +
      `<path class="crane-b" d="M-14 1C-9-4 3-5 9-2C11-1 10 2 6 3C0 4-8 4-14 1Z"/>` +
      `<path class="crane-n" d="M7-2Q15-4.4 22-7.2"/><circle class="crane-h" cx="23" cy="-7.5" r="1.8"/><circle class="crane-c" cx="23.3" cy="-8.8" r=".9"/><path class="crane-k" d="M24.5-8L31-6.7L24.5-6.6Z"/>` +
      `<path class="crane-w" d="${wUp}">${an(wUp, wDn)}</path><path class="crane-e" d="${eUp}">${an(eUp, eDn)}</path></g>`;
  }
  function boat(x, y, s) {
    return `<g class="boat" transform="translate(${x} ${y}) scale(${s})">
      <path class="boat-wake" d="M30 6C60 7 90 5 120 7M36 10C60 11 80 10 104 11"/>
      <path class="hull" d="M-44 0H44C40 6 30 9 20 10H-22C-34 9-41 5-44 0Z"/>
      <path class="hull-l" d="M-40 3H40"/>
      <rect class="cabin" x="-24" y="-16" width="40" height="16"/>
      <path class="cabin-w" d="M-20-13h7v10h-7ZM-9-13h7v10h-7ZM2-13h7v10h-7Z"/>
      <path class="boat-roof" d="M-32-15Q-6-19 24-15L18-25H-26Z"/>
      <path class="boat-roofl" d="M-30-16Q-6-20 22-16"/>
      <path class="pole" d="M30 0V-22"/></g>`;
  }

  /* ── 宫墙与宫殿屋顶群（故宫） ──────────────────── */
  function hall(ew, rh, rw, bodyH, tone = 'Y', double = true) {
    return sub(Q => {
      A.body(Q, 0, bodyH, ew * 0.72, { cols: 9, lat: 0.66, lintel: bodyH * 0.16 });
      A.dougong(Q, bodyH, bodyH * 0.12, ew * 0.72, 2);
      if (double) {
        A.roof(Q, bodyH + bodyH * 0.12, ew * 1.02, rh * 0.32, ew * 0.7, { tone, noRidge: true, band: 1.6, tile: 2, lite: false });
        A.body(Q, bodyH * 1.12 + rh * 0.32, bodyH * 1.12 + rh * 0.32 + bodyH * 0.32, ew * 0.66, { cols: 9, lattice: false, lintel: bodyH * 0.12 });
        A.roof(Q, bodyH * 1.44 + rh * 0.32, ew * 0.95, rh, rw, { tone, band: 1.7, tile: 2, lite: false });
      } else A.roof(Q, bodyH * 1.12, ew, rh, rw, { tone, band: 1.6, tile: 2, lite: false });
    });
  }
  function palaceWall(P, x0, x1, y, h) {
    P.p(`M${x0} ${y}H${x1}V${y - h}H${x0}Z`, 'pwall');
    P.p(`M${x0} ${y - h * 0.35}H${x1}`, 'pwall-l');
    P.p(`M${x0 - 4} ${y - h}H${x1 + 4}L${x1} ${y - h - 7}H${x0}Z`, 'pwall-cap');
    let t = ''; for (let x = x0; x < x1; x += 5) t += `M${x} ${y - h - 6.5}v5`; P.p(t, 'pwall-tile');
  }

  // 紫禁城城墙：栗红墙身、条石墙根、垛口
  function cityWall(P, x0, x1, y, h) {
    P.p(`M${x0} ${y}H${x1}V${y - h}H${x0}Z`, 'cwall');
    P.p(`M${x0} ${y - 4}H${x1}V${y}H${x0}Z`, 'cwall-base');
    P.p(`M${x0} ${y - h * 0.55}H${x1}`, 'pwall-l');
    let c = `M${x0} ${y - h}H${x1}V${y - h - 3.6}H${x0}Z`; for (let x = x0; x < x1 - 6; x += 15.2) c += `M${x} ${y - h - 3.4}h9.6v-6.4h-9.6Z`;
    P.p(c, 'cwall-c');
  }

  /* ── 长城（沿山脊） ──────────────────────────────── */
  function greatWall(P, ridge, from, to) {
    const seg = ridge.filter(([x]) => x >= from && x <= to);
    if (seg.length < 2) return;
    const top = seg.map(([x, y]) => [x, y - 6]);
    P.p(smooth(seg) + 'L' + top.slice().reverse().map(pt).join('L') + 'Z', 'gw');
    let cr = ''; for (let i = 0; i < top.length; i++) cr += `M${N(top[i][0])} ${N(top[i][1])}v-2.4`; P.p(cr, 'gw-c');
    // 敌楼：放在局部高点
    for (let i = 2; i < seg.length - 2; i++) {
      const [x, y] = seg[i];
      if (y < seg[i - 1][1] && y < seg[i + 1][1] && y < seg[i - 2][1] && y < seg[i + 2][1]) {
        P.p(`M${N(x - 9)} ${N(y)}V${N(y - 15)}H${N(x + 9)}V${N(y)}Z`, 'gw-t');
        let c = `M${N(x - 9)} ${N(y - 15)}`; for (let k = 0; k < 5; k++) c += `h2v-2.6h1.6v2.6`; P.p(c, 'gw-tc');
        P.p(`M${N(x - 5)} ${N(y - 4)}v-5a1.4 1.4 0 0 1 2.8 0v5ZM${N(x + 2.2)} ${N(y - 4)}v-5a1.4 1.4 0 0 1 2.8 0v5Z`, 'gw-w');
      }
    }
  }

  // 地标：带 id 的内层组，倒影用 <use> 引用，不重复整份图形
  const placed = {};
  function placeId(name, x, y, h, id, cls = '', attrs = '') {
    const L = A.LANDMARKS[name](), sc = h / (L.box[3] - L.box[2]);
    placed[id] = { x, y, sc };
    return `<g class="lm ${cls}" transform="translate(${N(x)} ${N(y)}) scale(${N(sc)} ${N(-sc)})"${attrs}><g id="${id}">${L.g}</g></g>`;
  }
  let refl = [];
  function reflectUse(id) {
    const { x, y, sc } = placed[id];
    refl.push(`<g transform="translate(${N(x)} ${N(y)}) scale(${N(sc)} ${N(sc)})"><use href="#${id}"/></g>`);
    return '';
  }
  // 水面：倒影越远越淡，被一道道细波纹截断。
  // 不用蒙版、也不给倒影组设透明度（两者都要离屏缓冲，滚动时很费）：倒影照常画，
  // 上面盖一层和水同色、越往下越不透明的渐变，再在倒影范围里压几道浅色波纹。
  const hex = c => [1, 3, 5].map(i => parseInt(c.slice(i, i + 2), 16));
  const mixc = (a, b, t) => '#' + hex(a).map((v, i) => Math.round(v + (hex(b)[i] - v) * t).toString(16).padStart(2, '0')).join('');
  function coverGrad(id, c0, c1, c2) {
    const ra = [[0, 0.34], [0.08, 0.3], [0.16, 0.24], [0.26, 0.19], [0.38, 0.13], [0.5, 0.07], [0.65, 0.03], [0.8, 0.005], [1, 0]];
    const col = t => t < 0.5 ? mixc(c0, c1, t / 0.5) : mixc(c1, c2, (t - 0.5) / 0.5);
    return `<linearGradient id="${id}" gradientUnits="userSpaceOnUse" x1="0" y1="${WY - 8}" x2="0" y2="${H + 40}">${ra.map(([t, al]) => `<stop offset="${t}" stop-color="${col(t)}" stop-opacity="${(1 - al).toFixed(3)}"/>`).join('')}</linearGradient>`;
  }
  function waterDefs() {
    return `<defs>${coverGrad('g-rcover', '#c7d6d0', '#d7e1d9', '#e6e5d6')}${coverGrad('g-rcoverW', '#dfe7ea', '#ebf0ef', '#f1efe6')}` +
      `<pattern id="p-ripple" width="230" height="13" patternUnits="userSpaceOnUse"><path class="rip" d="M0 2H26M64 2H98M150 2H196M30 6.8H52M112 6.8H160M196 6.8H230M8 10.6H40M84 10.6H128M170 10.6H184"/></pattern>` +
      `<pattern id="p-waves" width="430" height="56" patternUnits="userSpaceOnUse">${waves(0, 430, 0, 56, 9, 'wave')}</pattern>` +
      `<pattern id="p-waves2" width="610" height="74" patternUnits="userSpaceOnUse" patternTransform="translate(137 23)">${waves(0, 610, 0, 74, 31, 'wave wave2')}</pattern></defs>`;
  }

  function trees(P, list, seed) {
    const B = window.FLORA.Bucket(), r = rng(seed);
    list.slice().sort((a, b) => a[1] - b[1]).forEach(([x, y, h, k]) => window.FLORA.tree(B, x, y, h, k, r));
    B.flush(P);
  }
  const at = (x, y, s, inner, cls = '', attrs = '') => `<g${cls ? ` class="${cls}"` : ''} transform="translate(${N(x)} ${N(y)}) scale(${N(s)} ${N(-s)})"${attrs}>${inner}</g>`;

  /* ── 颐和园 ─────────────────────────────────────────
     万寿山满山林木；中轴自湖岸牌楼起，排云门、排云殿、德辉殿，登石台到佛香阁；两侧转轮藏、宝云阁；长廊沿湖。 */
  function yiheyuan(main) {
    const YH = 1150, HB = WY + 6, hill = domeHill(YH, HB, 430, 214, 61), ridge = x => A.yAt(hill, x);
    const platY = 680, Y = Pen();
    Y.p(smooth(hill) + 'Z', 'mtn wsh', ' fill="url(#g-hill)"');
    Y.raw(`<clipPath id="cp-wsh"><rect x="${YH - 520}" y="0" width="1040" height="${HB}"/></clipPath><g clip-path="url(#cp-wsh)">`);
    window.FLORA.canopy(Y, { x0: YH - 426, x1: YH + 426, top: ridge, base: HB - 2, rows: 18, rFar: 7, rNear: 13, emerge: 0.025,
      mix: { p: 3, c: 0.7, e: 3.4, g: 2, a: 1.3, y: 0.8 }, seed: 5,
      skip: (x, y) => { const d = Math.abs(x - YH); return y > platY - 4 && (y < 702 ? d < 46 : y < 742 ? d < 58 + (y - 702) * 1.7 : d < 130); } });
    Y.raw('</g>');
    // 石台与佛香阁
    Y.raw(placeId('foxiangge', YH, platY, 228, 'lm-fx', 'pick', ' data-pick="yhy"'));
    // 转轮藏（东）、宝云阁（西）：石台两侧的小重檐阁
    for (const sx of [-1, 1]) Y.raw(at(YH + sx * 114, platY - 12, 0.5, sub(Q => { Q.g(hall(44, 16, 20, 13, 'Y', true), ' transform="translate(0 7)"'); A.terrace(Q, 0, 7, 34, { rail: 3, post: 4.4, bat: 1 }); })));
    window.FLORA.clump(Y, { x0: YH - 150, x1: YH - 80, y: platY - 2, h: 22, mix: { e: 2, p: 1, a: 1 }, seed: 81, rows: 1, dense: 0.34 });
    window.FLORA.clump(Y, { x0: YH + 80, x1: YH + 150, y: platY - 2, h: 22, mix: { e: 2, p: 1, y: 1 }, seed: 83, rows: 1, dense: 0.34 });
    // 德辉殿：贴着石台脚
    Y.raw(at(YH, platY + 14, 0.5, hall(78, 24, 50, 18, 'Y', false)));
    // 排云殿：汉白玉台上重檐，东西配殿
    Y.raw(at(YH, 744, 0.52, sub(Q => {
      Q.g(hall(100, 28, 62, 22, 'Y', true), ' transform="translate(0 14)"');
      A.terrace(Q, 0, 14, 124, { rail: 4.4, post: 6, bat: 2, chishou: 8 });
      A.stairs(Q, 0, 14, 20, 15, 9, { ramp: 7 });
    })));
    for (const sx of [-1, 1]) Y.raw(at(YH + sx * 98, 746, 0.44, hall(64, 20, 40, 16, 'Y', false)));
    // 院墙与排云门
    palaceWall(Y, YH - 132, YH - 40, 760, 11); palaceWall(Y, YH + 40, YH + 132, 760, 11);
    Y.raw(at(YH, 761, 0.6, hall(72, 22, 46, 18, 'Y', false)));
    // 长廊：沿湖一线绿瓦红柱，四座八角重檐亭
    const lx0 = YH - 392, lx1 = YH + 392;
    Y.p(`M${lx0} ${WY - 1}V${WY - 9}H${lx1}V${WY - 1}Z`, 'lang-b');
    let lc = ''; for (let xx = lx0 + 3; xx < lx1; xx += 7) lc += `M${xx} ${WY - 1}v-7`; Y.p(lc, 'lang-c');
    Y.p(`M${lx0 - 3} ${WY - 9}H${lx1 + 3}L${lx1} ${WY - 14}H${lx0}Z`, 'lang-r');
    for (const u of [-300, -160, 160, 300]) Y.raw(at(YH + u, WY - 1, 0.62, sub(Q => {
      for (const c of [-9, -3, 3, 9]) Q.r(c - 0.9, 0, 1.8, 13, 'f-col kf');
      A.roof(Q, 13, 15, 4, 7, { tone: 'G', noRidge: true, band: 1, tile: 1.3, soffit: false, lite: false });
      Q.r(-8, 17, 16, 3.4, 'f-paintG kf');
      A.roof(Q, 20.4, 12, 9, 0, { tone: 'G', band: 1, tile: 1.2, soffit: false, lite: false });
      A.baoding(Q, 29.4, 0.6);
    })));
    // 湖岸汉白玉栏杆
    let posts = ''; for (let xx = lx0 - 10; xx <= lx1 + 10; xx += 5.5) posts += `M${N(xx)} ${WY + 1.6}v-4.6`;
    Y.p(`M${lx0 - 12} ${WY - 1.4}H${lx1 + 12}V${WY + 2.4}H${lx0 - 12}Z`, 'shore-rail');
    Y.p(posts, 'shore-post');
    // 码头石阶
    Y.p(`M${YH - 30} ${WY - 1.4}H${YH + 30}L${YH + 36} ${WY + 6}H${YH - 36}Z`, 'axis-steps');
    Y.p(`M${YH - 32} ${WY + 1.2}H${YH + 32}M${YH - 34} ${WY + 3.6}H${YH + 34}`, 'kt-s');
    // 云辉玉宇牌楼：三间四柱七楼，立在湖岸
    Y.raw(at(YH, WY - 1, 0.52, sub(Q => {
      for (const u of [-31, -11, 11, 31]) Q.r(u - 1.5, 0, 3, 27, 'f-col kf');
      Q.r(-34, 20, 68, 3, 'f-paintG kf'); Q.r(-34, 23, 68, 2.4, 'f-paintB kf');
      A.caihua(Q, 20, 3, 34);
      for (const [u, y, ew, rh] of [[0, 31, 13, 6.5], [-21, 28, 10, 5], [21, 28, 10, 5], [-11, 29.5, 4.5, 4], [11, 29.5, 4.5, 4], [-33, 26.5, 5, 4], [33, 26.5, 5, 4]]) {
        if (Math.abs(u) === 11 || Math.abs(u) === 33) Q.r(u - 4, y - 4, 8, 4, 'f-paintG kt');
        else Q.r(u - ew * 0.75, y - 5.5, ew * 1.5, 5.5, 'f-paintB kt');
        Q.g(sub(R => A.roof(R, y, ew, rh, ew * 0.55, { band: 1, tile: 1.2, soffit: false, lite: false })), ` transform="translate(${u} 0)"`);
      }
    })));
    main.raw(`<g id="yhy-g">${Y.s()}</g>`);
    refl.push(`<use href="#yhy-g" transform="translate(0 ${2 * (HB - 2)}) scale(1 -1)"/>`);
  }

  /* ── 合成 ───────────────────────────────────────── */
  function compose() {
    const far = Pen(), mid = Pen(), main = Pen();
    refl = [];
    // 独立小块：自己平移、自己动画，不牵动整层重绘。
    // layer：插在哪层之后；d：相对主景的速度；anchor：数字＝这处居中时与主景对齐，'layer'＝与所在层对齐，'start'/'end'＝贴着卷首/卷尾的屏幕边
    const spr = [];
    const sprite = (o, svg) => spr.push({ layer: 'main', d: 1, anchor: 'layer', ...o, svg });
    const mistSprite = (layer, d, x, y, w, h, cls = 'mist') => sprite({ layer, d, x0: x - w, x1: x + w, y0: y - h, y1: y + h }, mist(x, y, w, h, cls));

    // ── 远景：西山、燕山，玉峰塔，CBD 没在雾里
    const ridgeA = mountain(far, { x: 380, base: 650, h: 290, w: 560, seed: 7, grad: 'g-mtnFar', moss: false, cls: 'far1' });
    mountain(far, { x: -160, base: 650, h: 230, w: 400, seed: 3, grad: 'g-mtnFar', moss: false, cls: 'far1' });
    mountain(far, { x: 1060, base: 655, h: 210, w: 430, seed: 11, grad: 'g-mtnFar', moss: false, cls: 'far1' });
    mountain(far, { x: 1880, base: 660, h: 165, w: 520, seed: 19, grad: 'g-mtnFar', moss: false, cls: 'far1' });
    mountain(far, { x: 2950, base: 670, h: 150, w: 560, seed: 23, grad: 'g-mtnFar', moss: false, cls: 'far1' });
    mountain(far, { x: 4150, base: 670, h: 165, w: 660, seed: 29, grad: 'g-mtnFar', moss: false, cls: 'far1' });
    greatWall(far, ridgeA, -140, 930);
    far.raw(A.place('yufeng', 930, 486, 74, 'far-lm'));
    far.raw(`<g class="cbd" transform="translate(4210 650)">${[[-96, 150, 22], [-60, 220, 18], [-26, 296, 22], [12, 250, 16], [46, 182, 20], [82, 132, 18]].map(([dx, h, w]) => `<path d="M${dx - w} 0V${-h}H${dx + w}V0Z"/>`).join('')}<path d="M-30-296h8v-16h-8Z"/></g>`);
    [[700, 628, 560, 60], [2000, 648, 760, 70], [3300, 658, 760, 70], [4320, 640, 440, 70]].forEach(([x, y, w, h]) => mistSprite('far', 0.72, x, y, w, h));

    // ── 中景：长城下的青绿近山；故宫后面的景山与万春亭
    const ridgeB = mountain(mid, { x: 210, base: 740, h: 350, w: 340, seed: 41, grad: 'g-mtnNear', trees: 18 });
    mountain(mid, { x: 640, base: 748, h: 250, w: 300, seed: 43, grad: 'g-mtnMid', trees: 14 });
    greatWall(mid, ridgeB, -60, 470);
    mountain(mid, { x: 1900, base: 752, h: 120, w: 260, seed: 47, grad: 'g-mtnMid', trees: 8 });
    const jing = mountain(mid, { x: 2660, base: 718, h: 108, w: 230, seed: 49, grad: 'g-hill', cun: false, trees: 10 });
    mid.raw(`<g transform="translate(2660 ${N(Math.min(...jing.map(p => p[1])) + 6)}) scale(.62 -.62)">${sub(Q => { Q.r(-12, 0, 24, 2.2, 'f-stone kf'); Q.p('M-9 2.2v10M-3 2.2v10M3 2.2v10M9 2.2v10', 'k'); A.roof(Q, 12.2, 17, 7, 0, { band: 1.1, tile: 1.4 }); A.roof(Q, 18.6, 12, 9, 0, { band: 1, tile: 1.3, soffit: false }); A.baoding(Q, 27.6, 0.7); })}</g>`);
    mountain(mid, { x: 3760, base: 755, h: 110, w: 330, seed: 59, grad: 'g-mtnMid', trees: 8 });
    [[380, 730, 460, 46], [1250, 752, 560, 36], [2400, 735, 560, 40], [3420, 756, 640, 36]].forEach(([x, y, w, h]) => mistSprite('mid', 0.86, x, y, w, h));

    // ── 主景：一条贯穿全卷的水
    main.raw(waterDefs());
    main.p(`M-400 ${WY - 8}H${W + 400}V${H + 40}H-400Z`, 'water');
    main.raw('<!--REFL-->');
    main.raw(`<rect class="wave-p" x="-400" y="${WY - 2}" width="${W + 800}" height="${H - WY + 40}" fill="url(#p-waves)"/><rect class="wave-p" x="-400" y="${WY + 4}" width="${W + 800}" height="${H - WY + 34}" fill="url(#p-waves2)"/>`);

    // 左岸：长城脚下
    bank(main, -400, 780, WY - 12, WY + 70, 1);
    window.FLORA.clump(main, { x0: -360, x1: 700, y: WY - 6, h: 40, mix: { e: 2, g: 2, a: 1.2, y: 1, p: 0.8 }, seed: 11, rows: 3, under: true });
    rock(main, 300, WY + 64, 56, 30, 3); reeds(main, 330, WY + 70, 9, 4);

    // 颐和园
    yiheyuan(main);
    main.raw(placeId('shiqikong', 1650, WY + 64, 34, 'lm-sqk', 'pick', ' data-pick="yhy"'));
    main.raw(reflectUse('lm-sqk'));
    bank(main, 1836, 1960, WY + 40, WY + 74, 5);
    window.FLORA.clump(main, { x0: 1846, x1: 1950, y: WY + 48, h: 28, mix: { e: 1, g: 1, a: 1, y: 0.6 }, seed: 13, rows: 2 });
    willow(main, 724, WY + 8, 104, 91); willow(main, 1592, WY + 10, 96, 93);

    // 故宫：景山前的宫殿屋顶群，栗红城墙临筒子河，角楼立在墙角，城门上起重檐门楼
    main.raw(`<g class="palace-far" transform="translate(2470 ${WY - 58}) scale(.62 -.62)">${hall(92, 28, 56, 24, 'Y', false)}</g>`);
    main.raw(`<g class="palace-far" transform="translate(2700 ${WY - 92}) scale(.5 -.5)">${hall(92, 28, 56, 24, 'Y', false)}</g>`);
    main.raw(`<g class="palace-far" transform="translate(2590 ${WY - 64}) scale(1 -1)">${hall(118, 34, 74, 30)}</g>`);
    main.raw(`<g class="palace-far" transform="translate(2935 ${WY - 58}) scale(.72 -.72)">${hall(96, 30, 60, 26, 'Y', false)}</g>`);
    main.raw(mist(2600, WY - 70, 380, 34, 'mist-s') + mist(2660, WY - 98, 260, 30, 'mist-s'));
    cityWall(main, 2280, 3010, WY + 2, 62);
    main.raw(`<g class="pick" data-pick="gugong" transform="translate(2812 ${WY + 2}) scale(.95 -.95)">${sub(Q => {
      Q.p(poly([[-58, 0], [58, 0], [54, 74], [-54, 74]]), 'f-cwall k');
      for (const u of [-28, 0, 28]) Q.p(`M${u - 8} 0V20A8 8 0 0 0 ${u + 8} 20V0Z`, 'f-dark kf');
      Q.r(-56, 74, 112, 3, 'f-stone kf');
      A.body(Q, 77, 95, 44, { cols: 9, lat: 0.64, lintel: 2.4 });
      A.dougong(Q, 95, 2.2, 44, 2.4);
      A.roof(Q, 97.6, 60, 8, 40, { noRidge: true, band: 1.6, tile: 2 });
      A.body(Q, 105.6, 112, 38, { cols: 9, lattice: false, lintel: 1.8 });
      A.roof(Q, 112, 56, 22, 30, { band: 1.8, tile: 2 });
    })}</g>`);
    main.raw(placeId('jiaolou', 2250, WY + 2, 252, 'lm-jl', 'pick', ' data-pick="gugong"'));
    main.raw(reflectUse('lm-jl'));

    // 天坛：古柏分列两侧，祈年殿立在三层祈谷坛上
    bank(main, 3010, 3660, WY - 6, WY + 112, 9);
    trees(main, [[3050, WY + 6, 196, 'oc'], [3112, WY + 2, 172, 'oc'], [3440, WY + 2, 178, 'oc'], [3502, WY + 8, 200, 'oc'], [3150, WY - 2, 150, 'oc'], [3400, WY - 2, 156, 'oc']], 400);
    main.raw(placeId('qinian', 3275, WY + 40, 330, 'lm-qn', 'pick', ' data-pick="tiantan"'));
    trees(main, [[2990, WY + 70, 168, 'oc'], [3044, WY + 92, 150, 'oc'], [3096, WY + 104, 118, 'oc'], [3456, WY + 104, 122, 'oc'], [3512, WY + 90, 156, 'oc'], [3570, WY + 72, 172, 'oc']], 450);
    rock(main, 3620, WY + 104, 50, 26, 21); reeds(main, 3000, WY + 104, 7, 22);

    // 国家博物馆
    bank(main, 3640, 4700, WY - 2, WY + 92, 11);
    window.FLORA.clump(main, { x0: 3600, x1: 3700, y: WY + 26, h: 40, mix: { e: 2, g: 1, y: 1, a: 0.6 }, seed: 17, rows: 2, under: true });
    main.raw(placeId('guobo', 3930, WY + 30, 138, 'lm-gb', 'pick', ' data-pick="guobo"'));
    window.FLORA.clump(main, { x0: 4170, x1: 4360, y: WY + 34, h: 44, mix: { a: 1, g: 1.4, y: 1, e: 1.6 }, seed: 19, rows: 3, under: true });

    // 夏：湖上荷（只在夏天显示）
    main.raw(`<g class="summer-only">${sub(Q => { lotus(Q, 800, WY + 30, 0.8, 5); lotus(Q, 1500, WY + 40, 0.75, 6); lotus(Q, 1960, WY + 92, 0.8, 7); lotus(Q, 3700, WY + 118, 0.8, 8); })}</g>`);
    sprite({ layer: 'fore', d: 1.16, anchor: 1150, x0: 1040, x1: 1300, y0: H - 120, y1: H + 20 }, `<g class="summer-only">${sub(Q => lotus(Q, 1170, H - 8, 1.7, 9))}</g>`);
    sprite({ layer: 'fore', d: 1.16, anchor: 3275, x0: 3360, x1: 3620, y0: H - 120, y1: H + 20 }, `<g class="summer-only">${sub(Q => lotus(Q, 3490, H - 6, 1.6, 10))}</g>`);

    // 画舫（独立小块，来回慢行）
    sprite({ x0: 1470, x1: 1600, y0: WY + 76, y1: WY + 112 }, boat(1520, WY + 98, 0.82));
    sprite({ x0: 3520, x1: 3640, y0: WY + 146, y1: WY + 186 }, boat(3560, WY + 168, 0.8));

    // 瑞鹤：一群仙鹤在宫城上空盘旋
    sprite({ x0: 2240, x1: 2900, y0: 260, y1: 500 }, `<g class="cranes">${[[2290, 392, 0.86, 1, 0.1], [2372, 356, 0.7, 1, 0.9], [2446, 418, 0.96, 1, 0.5], [2520, 372, 0.74, 1, 1.3], [2598, 436, 0.8, 1, 0.3], [2668, 396, 0.64, 1, 1.1], [2416, 300, 0.6, 1, 0.7], [2566, 320, 0.62, 1, 1.5], [2742, 350, 0.72, -1, 0.2], [2346, 452, 0.58, -1, 1.2]].map(([x, y, sc, f, ph], i) => crane(x, y, sc, f, ph, 1.5 + (i % 3) * 0.2)).join('')}</g>`);

    // ── 近景框景
    // 卷首迎客松：电脑上大，手机上小一号、只露半边
    sprite({ layer: 'fore', d: 1.16, anchor: 'start', off: -260, x0: -260, x1: 600, y0: 300, y1: H + 40, only: 'wide' }, sub(Q => window.FLORA.bigPine(Q, 40, H + 30, 600, 601, 1)));
    sprite({ layer: 'fore', d: 1.16, anchor: 'start', off: -240, x0: -240, x1: 420, y0: 420, y1: H + 40, only: 'narrow' }, sub(Q => window.FLORA.bigPine(Q, -10, H + 30, 470, 611, 1, { limbs: [[0.4, -1, 0.3, 0.04], [0.55, 1, 0.62, 0.02], [0.7, -1, 0.26, -0.02], [0.82, 1, 0.34, -0.03], [0.94, -1, 0.16, -0.03]] })));
    // 卷尾：一株斜出的松，只在电脑上
    sprite({ layer: 'fore', d: 1.16, anchor: 'end', off: -360, x0: 4100, x1: 4700, y0: 520, y1: H + 40, only: 'wide' }, sub(Q => window.FLORA.bigPine(Q, 4590, H + 30, 420, 603, -1, { limbs: [[0.4, -1, 0.3, 0.04], [0.52, 1, 0.5, 0.0], [0.66, -1, 0.22, -0.02], [0.78, 1, 0.3, -0.03], [0.92, -1, 0.15, -0.03]] })));
    // 角楼前的垂柳
    sprite({ layer: 'fore', d: 1.12, anchor: 2250, x0: 1740, x1: 2110, y0: -70, y1: 480, only: 'wide' }, `<g class="sway">${sub(Q => hangingWillow(Q, 1930, -10, 240, 380, 77))}</g>`);
    // 水边石与芦苇
    sprite({ layer: 'fore', d: 1.16, anchor: 1500, x0: 1400, x1: 1580, y0: H - 70, y1: H + 20 }, sub(Q => { rock(Q, 1500, H + 6, 120, 52, 31); reeds(Q, 1430, H + 4, 14, 32); }));
    sprite({ layer: 'fore', d: 1.16, anchor: 2830, x0: 2760, x1: 2940, y0: H - 60, y1: H + 20 }, sub(Q => { rock(Q, 2820, H + 8, 90, 40, 33); reeds(Q, 2860, H + 6, 12, 34); }));
    mistSprite('fore', 1.16, 1200, H - 30, 700, 60, 'mist mist-near'); mistSprite('fore', 1.16, 3000, H - 20, 800, 60, 'mist mist-near');

    // ── 天空：日、祥云、飞鸟
    const sky = `<rect class="sky" x="-400" y="-20" width="${W + 800}" height="${H + 40}" fill="url(#g-sky)"/>` +
      `<circle cx="3275" cy="292" r="200" fill="url(#g-sun)"/><circle class="sun" cx="3275" cy="292" r="44"/>`;
    [[560, 168, 1.4, 1], [1640, 116, 1.1, -1], [2580, 186, 1.2, 1], [3720, 132, 1.3, -1], [4260, 226, 0.9, 1]].forEach(([x, y, k, f], i) => sprite({ layer: 'sky', d: 0.55, x0: x - 130 * k, x1: x + 130 * k, y0: y - 40 * k, y1: y + 30 * k }, `<g class="clouds${i % 2 ? ' alt' : ''}">${xiangyun(x, y, k, f)}</g>`));
    sprite({ layer: 'sky', d: 0.55, x0: 820, x1: 980, y0: 230, y1: 290 }, `<g class="not-autumn">${birds(840, 252, 4, 0.9, 9)}</g>`);
    sprite({ layer: 'sky', d: 0.55, x0: 1420, x1: 1680, y0: 320, y1: 470 }, `<g class="autumn-only">${geese(1660, 395, 9, 1.1, 12)}</g>`);
    sprite({ layer: 'sky', d: 0.55, x0: 3700, x1: 3900, y0: 300, y1: 420 }, `<g class="autumn-only">${geese(3880, 360, 5, 0.9, 13)}</g>`);

    const ripple = [[720, 1600], [1380, 1920], [2000, 2500]].map(([x0, x1]) => `<rect x="${x0}" y="${WY}" width="${x1 - x0}" height="230" fill="url(#p-ripple)"/>`).join('');
    const mainS = main.s().replace('<!--REFL-->', `<g class="reflect">${refl.join('')}</g><rect class="rcover" x="-400" y="${WY - 8}" width="${W + 800}" height="${H - WY + 48}"/>${ripple}`);
    return { W, H, WY, sky, far: far.s(), mid: mid.s(), main: mainS, sprites: spr };
  }

  // 章节插图（vignettes.js）复用这些画山水、树、水、云的小工具
  const HELP = { smooth, bell, mountain, domeHill, rock, reeds, bank, willow, hangingWillow, lotus, waves, xiangyun, mist, birds, geese, boat, crane, hall, palaceWall, cityWall, greatWall, trees, at };
  window.SCENE = { compose, W, H, WY, HELP };
})();
