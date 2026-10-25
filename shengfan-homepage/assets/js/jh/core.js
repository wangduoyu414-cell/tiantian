/* 界画库 · 基础
   一、笔：三级墨线（外轮廓 / 结构 / 细部），线宽按放置比例换算，屏幕上约 1.6 / 0.9 / 0.5 像素，远处的物件线再细一点。
   二、色：矿物色平涂，每种色有阳面、阴面、提亮三档（见 jh.css）。光从左上来。
   三、坐标：建筑在局部坐标里画，u 向右、v 向上，单位约合 0.1 米；原点在台基底部中线。
       放进画面时套 translate(x y) scale(s -s)。
   四、规矩：直线一律横平竖直、对齐开间模数；曲线（屋面举架、檐口起翘、券洞、圆形构件的透视）一律用公式算，不手抖。 */
(function () {
  'use strict';
  const N = v => Math.round(v * 100) / 100;
  const P2 = (x, y) => `${N(x)} ${N(y)}`;
  const clamp = (v, a, b) => Math.min(b, Math.max(a, v));
  const lerp = (a, b, t) => a + (b - a) * t;

  function Pen() {
    const o = [];
    const api = {
      p(d, c, extra = '') { if (d) o.push(`<path class="${c}" d="${d}"${extra}/>`); return api; },
      r(x, y, w, h, c, extra = '') { o.push(`<rect class="${c}" x="${N(x)}" y="${N(y)}" width="${N(w)}" height="${N(h)}"${extra}/>`); return api; },
      c(x, y, r, c, extra = '') { o.push(`<circle class="${c}" cx="${N(x)}" cy="${N(y)}" r="${N(r)}"${extra}/>`); return api; },
      e(x, y, rx, ry, c, extra = '') { o.push(`<ellipse class="${c}" cx="${N(x)}" cy="${N(y)}" rx="${N(rx)}" ry="${N(ry)}"${extra}/>`); return api; },
      g(inner, attrs = '') { o.push(`<g${attrs}>${inner}</g>`); return api; },
      raw(s) { o.push(s); return api; },
      s() { return o.join(''); },
    };
    return api;
  }
  const sub = fn => { const Q = Pen(); fn(Q); return Q.s(); };

  /* ── 曲线工具 ───────────────────────────────── */
  const cub = (p0, p1, p2, p3, t) => { const u = 1 - t; return [u * u * u * p0[0] + 3 * u * u * t * p1[0] + 3 * u * t * t * p2[0] + t * t * t * p3[0], u * u * u * p0[1] + 3 * u * u * t * p1[1] + 3 * u * t * t * p2[1] + t * t * t * p3[1]]; };
  const samp = (f, n = 32) => Array.from({ length: n + 1 }, (_, i) => f(i / n));
  const poly = (pts, close = true) => 'M' + pts.map(([x, y]) => P2(x, y)).join('L') + (close ? 'Z' : '');
  const line = pts => poly(pts, false);
  // 折线上按 x 取 y（x 单调）
  function yAt(pts, x) {
    for (let i = 1; i < pts.length; i++) {
      const a = pts[i - 1], b = pts[i];
      if ((x - a[0]) * (x - b[0]) <= 0 && a[0] !== b[0]) return a[1] + (b[1] - a[1]) * (x - a[0]) / (b[0] - a[0]);
    }
    return null;
  }
  // Catmull-Rom 平滑
  function smooth(pts, close = false) {
    let d = `M${P2(...pts[0])}`;
    const n = pts.length;
    for (let i = 0; i < n - (close ? 0 : 1); i++) {
      const p0 = pts[(i - 1 + n) % n] , p1 = pts[i], p2 = pts[(i + 1) % n], p3 = pts[(i + 2) % n];
      const q0 = close ? p0 : (pts[i - 1] || p1), q3 = close ? p3 : (pts[i + 2] || p2);
      d += `C${P2(p1[0] + (p2[0] - q0[0]) / 6, p1[1] + (p2[1] - q0[1]) / 6)} ${P2(p2[0] - (q3[0] - p1[0]) / 6, p2[1] - (q3[1] - p1[1]) / 6)} ${P2(...p2)}`;
    }
    return d + (close ? 'Z' : '');
  }
  // 可复现的随机数（山石、树、水纹用；建筑不用）
  function rng(seed) { let s = seed >>> 0 || 1; return () => (s = (s * 1664525 + 1013904223) >>> 0) / 4294967296; }

  /* ── 线宽：按"局部单位 → 屏幕像素"的比例换算 ──────────
     px：一个局部单位在屏幕上约多少像素。远处小的物件（px 小）线也跟着细一些。 */
  function lineStyle(px) {
    const f = clamp(0.62 + px * 0.32, 0.62, 1);
    return `--w1:${N(1.6 * f / px)}px;--w2:${N(0.95 * f / px)}px;--w3:${N(0.55 * f / px)}px`;
  }
  // 放置：局部坐标的图形 g（含 box = [左, 右, 下, 上]）放到画面里，h 是画面里的高度，px 是画面单位到屏幕的比例
  function place(L, x, y, h, scenePx = 1, cls = '', attrs = '') {
    const s = h / (L.box[3] - L.box[2]);
    return `<g class="jh ${cls}" style="${lineStyle(s * scenePx)}" transform="translate(${N(x)} ${N(y)}) scale(${N(s)} ${N(-s)})"${attrs}>${L.g}</g>`;
  }
  // 单独成图：给样式页、小图标用
  function svg(L, px, pad = 6, cls = '') {
    const [l, r, b, t] = L.box, w = r - l + pad * 2, h = t - b + pad * 2;
    return `<svg class="jh-svg ${cls}" viewBox="${N(l - pad)} ${N(-t - pad)} ${N(w)} ${N(h)}" width="${N(w * px)}" height="${N(h * px)}" aria-hidden="true"><g class="jh" style="${lineStyle(px)}" transform="scale(1 -1)">${L.g}</g></svg>`;
  }

  /* ── 圆形构件的透视（祈年殿、圆亭、圆坛） ────────
     视平线在 eyeZ。高于视平线的圆，前沿往上拱（看得见檐底）；低于视平线的圆，前沿往下垂（看得见台面）。
     θ 从 π（左）经 3π/2（正前）到 2π（右）是看得见的前半圈。 */
  function Round(eyeZ, upZ, upK = 0.13, dnK = 0.13, dnZ = 0) {
    const e = z => z >= eyeZ ? upK * Math.min(1, (z - eyeZ) / Math.max(1, upZ - eyeZ)) : -dnK * Math.min(1, (eyeZ - z) / Math.max(1, eyeZ - dnZ));
    const p = (r, th, z) => [r * Math.cos(th), z + e(z) * r * -Math.sin(th)];
    // 前半圈的弧：从左到右
    const arc = (r, z, n = 48) => Array.from({ length: n + 1 }, (_, i) => p(r, Math.PI + Math.PI * i / n, z));
    return { e, p, arc };
  }

  /* ── 细节档位：远处、小尺寸的建筑少画椽头、斗拱细部、彩画纹样（JH.lod：1 全画，0.4 只留轮廓和大块颜色） ── */
  function withLod(v, fn) { const J = window.JH, old = J.lod; J.lod = v; try { return fn(); } finally { J.lod = old; } }

  window.JH = { N, P2, clamp, lerp, Pen, sub, cub, samp, poly, line, yAt, smooth, rng, lineStyle, place, svg, Round, withLod, lod: 1 };
})();
