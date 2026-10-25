/* 界画库 · 多边形楼阁（八角、四角）：佛香阁、玉峰塔、亭子
   和祈年殿同一套视法：视平线在 eye，高于视平线的檐口前沿往上拱（看得见檐底），低于的台面往下垂。
   平面是正 n 边形，正前方一个面；θ = 3π/2 是正前方，看得见的是前半圈（-sinθ > 0）。
   屋面：每个面按举架做凹曲面，瓦垄顺着该面的坡向走、碰到戗脊就止；檐口每个面中段平、两角起翘出冲，
         每个角上翼角上挑、挂套兽，戗脊上几只走兽；攒尖收到顶上安宝顶。 */
(function () {
  'use strict';
  const { N, P2, Pen, sub, cub, samp, poly, Round } = window.JH, A = window.JH.ARCH;

  function Tower(o = {}) {
    const P = o.P || Pen(), n = o.n || 8, R = Round(o.eye ?? 0, o.top ?? 400, o.upK ?? 0.16, o.dnK ?? 0.08, o.dnZ ?? -100);
    const fc = Math.PI * 1.5, step = Math.PI * 2 / n, cs = Math.cos(step / 2);
    const faces = Array.from({ length: n }, (_, k) => fc + k * step).filter(th => -Math.sin(th) > 0.05);
    faces.sort((a, b) => -Math.sin(a) - -Math.sin(b));                       // 先画两侧，正面最后
    // 面上一点：外接圆半径 r 的多边形，面心角 th、面内参数 u ∈ [-1, 1]
    const plan = (r, th, u) => { const ax = Math.cos(th), ay = Math.sin(th), tx = -ay, ty = ax, a = r * cs, h = r * Math.sin(step / 2); return [ax * a + tx * h * u, ay * a + ty * h * u]; };
    const proj = ([x, y], z) => { const r = Math.hypot(x, y), th = Math.atan2(y, x); return R.p(r, th, z); };
    const at = (r, th, u, z) => proj(plan(r, th, u), z);
    const vis = th => -Math.sin(th) > 0.05;
    const shadeFace = th => Math.cos(th) > 0.2 ? 'c-shade' : Math.cos(th) < -0.2 ? 'c-light' : '';

    /* 楼身：每个看得见的面——角柱、柱间隔扇或槛窗（按面的朝向缩窄）、额枋彩画、斗拱一带 */
    function body(r, z0, z1, b = {}) {
      const bH = b.bH ?? (z1 - z0) * 0.12, dH = b.dH ?? (z1 - z0) * 0.14, zc = z1 - bH - dH, cw = b.cw ?? Math.max(3, r * 0.06);
      for (const th of faces) {
        const k = b.cols ?? 2, us = Array.from({ length: k + 1 }, (_, i) => -1 + 2 * i / k);
        for (let i = 0; i < k; i++) {
          const a = us[i], c = us[i + 1], q = [at(r, th, a, z0), at(r, th, c, z0), at(r, th, c, zc), at(r, th, a, zc)];
          P.p(poly(q), b.fill === 'wall' ? 'c-red l3' : b.fill === 'stone' ? 'c-marble l3' : b.fill === 'win' ? 'p-lattice2 l3' : 'p-lattice l3');
          if (b.fill === 'stone') { const [x0, y0] = at(r, th, (a + c) / 2, z0 + (zc - z0) * 0.25), [x1] = at(r, th, (a + c) / 2 + 0.22, z0), w = Math.abs(x1 - x0), hh = (zc - z0) * 0.42; if (w > 0.4) P.p(`M${P2(x0 - w, y0)}v${N(hh * 0.6)}a${N(w)} ${N(w)} 0 0 0 ${N(w * 2)} 0v${N(-hh * 0.6)}Z`, 'c-ink'); }
          if (b.fill !== 'wall' && b.fill !== 'stone') { let d = ''; for (const f of [0.08, 0.3, 0.92]) { const z = z0 + (zc - z0) * f; d += `M${P2(...at(r, th, a, z))}L${P2(...at(r, th, c, z))}`; } P.p(d, 'l3'); P.p(poly([at(r, th, a, z0), at(r, th, c, z0), at(r, th, c, z0 + (zc - z0) * 0.3), at(r, th, a, z0 + (zc - z0) * 0.3)]), 'c-red l3'); }
        }
        // 额枋：青绿两道，金线；斗拱一带
        P.p(poly([at(r + 0.5, th, -1, zc), at(r + 0.5, th, 1, zc), at(r + 0.5, th, 1, zc + bH), at(r + 0.5, th, -1, zc + bH)]), 'c-qing l3');
        P.p(poly([at(r + 0.5, th, -0.55, zc + bH * 0.2), at(r + 0.5, th, 0.55, zc + bH * 0.2), at(r + 0.5, th, 0.55, zc + bH * 0.8), at(r + 0.5, th, -0.55, zc + bH * 0.8)]), 'c-lu lg');
        P.p(poly([at(r + 1, th, -1, zc + bH), at(r + 1, th, 1, zc + bH), at(r + 1, th, 1, z1), at(r + 1, th, -1, z1)]), 'c-red l3');
        let dg = ''; const m = b.dg ?? 4;
        for (let i = 0; i <= m; i++) { const u = -1 + 2 * i / m, [x, y0] = at(r + 1.5, th, u, zc + bH), [x1] = at(r + 1.5, th, Math.min(1, u + 0.12), zc + bH), w = Math.max(0.6, Math.abs(x1 - x) * 0.9), hh = (z1 - zc - bH); dg += `M${P2(x - w, y0)}h${N(w * 2)}v${N(hh * 0.4)}h${N(w * 0.6)}v${N(hh * 0.3)}h${N(-w * 3.2)}v${N(-hh * 0.3)}h${N(w * 0.6)}Z`; }
        P.p(dg, 'c-lu l3');
        const sh = shadeFace(th); if (sh) P.p(poly([at(r + 1, th, -1, z0), at(r + 1, th, 1, z0), at(r + 1, th, 1, z1), at(r + 1, th, -1, z1)]), sh);
      }
      // 角柱：圆柱不缩窄
      let cd = '';
      for (let k = 0; k < n; k++) { const th = fc + step / 2 + k * step; if (-Math.sin(th) < -0.05) continue; const [x, yb] = R.p(r, th, z0), [, yt] = R.p(r, th, zc); cd += `M${P2(x - cw / 2, yb)}h${N(cw)}V${N(yt)}h${N(-cw)}Z`; }
      if (!b.noCols) P.p(cd, 'g-col l2');
      return z1;
    }

    /* 平座：楼层外一圈挑台，台边一道枋，上立寻杖栏杆 */
    function balcony(r, z, h, b = {}) {
      for (const th of faces) {
        P.p(poly([at(r, th, -1, z - h * 0.35), at(r, th, 1, z - h * 0.35), at(r, th, 1, z), at(r, th, -1, z)]), 'c-lu l3');
        const k = b.posts ?? 5; let posts = '', rail = '';
        for (let i = 0; i <= k; i++) { const u = -1 + 2 * i / k, [x, y] = at(r - 0.5, th, u, z); posts += `M${P2(x - 0.5, y)}h1v${N(h)}h-1Z`; }
        for (const f of [0.35, 0.95]) rail += `M${P2(...at(r - 0.5, th, -1, z + h * f))}L${P2(...at(r - 0.5, th, 1, z + h * f))}`;
        P.p(poly([at(r - 0.5, th, -1, z), at(r - 0.5, th, 1, z), at(r - 0.5, th, 1, z + h * 0.35), at(r - 0.5, th, -1, z + h * 0.35)]), `${b.cls || 'c-red'} l3`);
        P.p(posts, `${b.cls || 'c-red'} l3`); P.p(rail, 'l2');
      }
    }

    /* 台基：多边形须弥座，面上一道束腰 */
    function base(r, z0, z1, b = {}) {
      for (const th of faces) {
        P.p(poly([at(r, th, -1, z0), at(r, th, 1, z0), at(r, th, 1, z1), at(r, th, -1, z1)]), `${b.cls || 'c-marble'} l2`);
        P.p(poly([at(r - 1.5, th, -1, z0 + (z1 - z0) * 0.35), at(r - 1.5, th, 1, z0 + (z1 - z0) * 0.35), at(r - 1.5, th, 1, z0 + (z1 - z0) * 0.65), at(r - 1.5, th, -1, z0 + (z1 - z0) * 0.65)]), `${b.cls ? b.cls + '-d' : 'c-marble-d'} l3`);
        const sh = shadeFace(th); if (sh) P.p(poly([at(r, th, -1, z0), at(r, th, 1, z0), at(r, th, 1, z1), at(r, th, -1, z1)]), sh);
      }
    }

    /* 屋面：re/ze 檐口外接圆半径与高，rt/zt 屋顶（攒尖 rt = 0）；r 参数 o：
       tone 琉璃色、trim 剪边色、tb 檐口带厚、lift 角上起翘、chu 角上出冲（比例）、rIn/zIn 檐下内沿、tiles 每面瓦垄数、beasts 每条戗脊走兽数 */
    function roof(re, ze, rt, zt, r = {}) {
      const tone = r.tone || 'yellow', trim = r.trim || tone, tb = r.tb ?? (zt - ze) * 0.06, lift = r.lift ?? (zt - ze) * 0.12, chu = r.chu ?? 0.05, p = 2.4;
      const rIn = r.rIn ?? rt + (re - rt) * 0.55, zIn = r.zIn ?? ze - tb * 0.2;
      // 坡面剖面（按外接圆半径）：檐口平缓、近顶陡
      const prof = t => cub([re, ze], [re - (re - rt) * 0.56, ze + (zt - ze) * 0.08], [rt + (re - rt) * 0.1, ze + (zt - ze) * 0.56], [rt, zt], t);
      const PA = samp(prof, 48).map(([rr, z]) => [rr * cs, z]);
      const Z = a => { if (a >= PA[0][0]) return PA[0][1]; for (let i = 1; i < PA.length; i++) { const [a0, z0] = PA[i - 1], [a1, z1] = PA[i]; if (a <= a0 && a >= a1) return z0 + (z1 - z0) * (a - a0) / ((a1 - a0) || 1); } return PA[PA.length - 1][1]; };
      // 檐口：面上参数 u，平面点按外接圆方向外推（角上出冲），高度角上起翘
      const eaveP = (th, u, dz = 0) => { const [x, y] = plan(re, th, u), k = Math.pow(Math.abs(u), p); return proj([x * (1 + chu * k), y * (1 + chu * k)], ze + lift * k + dz); };
      const hip = th => samp(t => { const [rr, z] = prof(t); const k = 1 - Math.min(1, t * 2.4); return R.p(rr * cs / Math.cos(step / 2) * (1 + chu * k * k), th, z + lift * k * k); }, 18);
      const vtx = (th, s) => th + s * step / 2;
      const us = samp(t => -1 + 2 * t, 24);
      for (const th of faces) {
        const eave = us.map(u => eaveP(th, u)), band = us.map(u => eaveP(th, u, tb));
        const hipL = hip(vtx(th, -1)), hipR = hip(vtx(th, 1));
        // 檐下：檐口到斗拱外沿，深色，两排椽头
        const inner = us.map(u => at(rIn, th, u, zIn));
        P.p(poly([...eave, ...inner.slice().reverse()]), 'c-soffit');
        let fly = '', rnd = '';
        for (let i = 1; i < us.length - 1; i++) { const [x, y] = eave[i], [x2] = eave[i + 1], w = Math.max(0.35, Math.abs(x2 - x) * 0.32); fly += `M${P2(x - w, y - 0.3)}h${N(w * 2)}v${N(-tb * 0.5)}h${N(-w * 2)}Z`; rnd += `M${P2(x - w, y - tb * 0.95)}a${N(w)} ${N(w)} 0 1 0 ${N(w * 2)} 0a${N(w)} ${N(w)} 0 1 0 ${N(-w * 2)} 0Z`; }
        P.p(fly, 'c-lu l3'); P.p(rnd, 'c-qing l3');
        // 坡面
        const topL = rt > 0.5 ? at(rt, th, -1, zt) : [0, zt], topR = rt > 0.5 ? at(rt, th, 1, zt) : [0, zt];
        const surf = [...band, ...hipR.slice(1), topR, topL, ...hipL.slice(1).reverse()];
        P.p(poly(surf), `g-${tone} l1`);
        // 瓦垄：从檐口带上沿顺着该面的坡向往里、往上，出了本面（碰到戗脊）就止
        const nT = r.tiles ?? 9, nx = Math.cos(th), ny = Math.sin(th);
        let dk = '', lt = '';
        for (let i = 1; i < nT; i++) {
          const u = -1 + 2 * i / nT, [x0, y0] = plan(re, th, u), k = Math.pow(Math.abs(u), p), q0 = [x0 * (1 + chu * k), y0 * (1 + chu * k)];
          const z0 = ze + lift * k + tb, a0 = q0[0] * nx + q0[1] * ny, off = z0 - Z(a0), run = (a0 - rt * cs) * 0.35, pts = [];
          for (let d = 0; ; d += (a0 - rt * cs) / 30) {
            const q = [q0[0] - nx * d, q0[1] - ny * d], a = a0 - d; if (a <= rt * cs + 0.2) break;
            let ang = Math.atan2(q[1], q[0]) - th; ang = Math.atan2(Math.sin(ang), Math.cos(ang)); if (Math.abs(ang) > step / 2 - 0.004) break;
            pts.push(proj(q, Z(a) + off * Math.max(0, 1 - d / (run || 1))));
          }
          if (pts.length > 2) { dk += 'M' + pts.map(q => P2(...q)).join('L'); lt += 'M' + pts.slice(1, -1).map(([x, y]) => P2(x + 0.35, y)).join('L'); }
        }
        P.p(dk, 'l3'); P.p(lt, 'lw');
        const sh = shadeFace(th); if (sh) P.p(poly(surf), sh);
        // 檐口带与勾头
        P.p(poly([...eave, ...band.slice().reverse()]), `c-${trim}-d l2`);
        let heads = ''; for (let i = 1; i < us.length - 1; i += 1) { const [x, y] = us[i] !== undefined ? eaveP(th, us[i], tb * 0.5) : [0, 0], [x2] = eaveP(th, us[Math.min(us.length - 1, i + 1)], tb * 0.5), rr = Math.max(0.3, Math.min(tb * 0.36, Math.abs(x2 - x) * 0.4)); heads += `M${P2(x - rr, y)}a${N(rr)} ${N(rr)} 0 1 0 ${N(rr * 2)} 0a${N(rr)} ${N(rr)} 0 1 0 ${N(-rr * 2)} 0Z`; }
        P.p(heads, `c-${trim}-l l3`);
      }
      // 戗脊、翼角、套兽、走兽：每个看得见的角
      const hb = r.ridge ?? tb * 1.1;
      const corners = [];
      for (let k = 0; k < n; k++) { const v = fc + step / 2 + k * step; if (-Math.sin(v) > -0.02) corners.push(v); }
      corners.sort((a, b) => -Math.sin(a) - -Math.sin(b));
      for (const v of corners) {
        const pts = hip(v), s = Math.cos(v) >= 0 ? 1 : -1, out = Math.abs(Math.cos(v));
        const up = pts.map(([x, y], i) => { const a = pts[Math.max(0, i - 1)], b = pts[Math.min(pts.length - 1, i + 1)]; let qx = -(b[1] - a[1]), qy = b[0] - a[0]; const l = Math.hypot(qx, qy) || 1; qx /= l; qy /= l; if (qy < 0) { qx = -qx; qy = -qy; } return [x + qx * hb, y + qy * hb]; });
        P.p(poly([...pts, ...up.reverse()]), `c-${tone} l2`);
        // 翼角：角上一截上挑的仔角梁头，套兽挂在下面
        const [cx, cy] = pts[0], L = (re * 0.032 + tb) * (0.45 + out * 0.6), tip = [cx + s * L * out, cy + L * 0.6 + tb * 0.35];
        P.p(`M${P2(cx - s * tb * 0.8, cy)}Q${P2(cx + s * L * out * 0.6, cy - tb * 0.1)} ${P2(...tip)}L${P2(tip[0] - s * tb * 0.25, tip[1] + tb * 0.55)}Q${P2(cx + s * L * out * 0.3, cy + tb * 0.9)} ${P2(cx - s * tb * 0.8, cy + tb)}Z`, `c-${trim}-d l2`);
        A.taoshou(P, tip[0] - s * tb * 0.2, tip[1] - tb * 0.55, tb * 1.25, s, tone);
        const nb = r.beasts ?? 2;
        for (let i = 0; i < nb; i++) { const [x, y] = pts[2 + i * 2] || pts[pts.length - 1]; A.beast(P, x, y + hb * 0.85, hb * 1.45, s, tone, i ? 'sit' : 'lead'); }
      }
      return { prof };
    }

    return { P, R, faces, plan, proj, at, body, balcony, base, roof, n };
  }

  window.JH.TOWER = Tower;
})();
