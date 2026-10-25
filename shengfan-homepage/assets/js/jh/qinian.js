/* 界画库 · 天坛祈年殿（标杆）
   视平线定在坛面：上面三重檐的檐口往上拱（看得见檐底的椽头、斗拱），下面三层圆坛往下垂（看得见坛面和一圈圈汉白玉栏杆）。
   单位约 0.1 米：三层祈谷坛每层约 1.9 米高；殿身三层，蓝琉璃三重檐圆攒尖，鎏金宝顶。
   比例照实物：殿高（坛面到宝顶）与下檐宽大致相当；三重檐逐层收进，中檐约为下檐的八成、上檐约六成，上檐最陡。
   画法：屋面按举架做凹曲面，瓦垄顺着经线走（越近两侧越密、越弯）；檐下飞椽、檐椽两排椽头；
   柱是圆柱不变宽，柱间隔扇按角度透视缩窄；斗拱一攒一攒随圆周缩窄；台基每层栏杆的望柱随圆周疏密变化。 */
(function () {
  'use strict';
  const { N, P2, Pen, sub, cub, samp, poly, Round } = window.JH, A = window.JH.ARCH;

  function qinian() {
    const P = Pen();
    const EYE = 60, TOP = 440;
    const R = Round(EYE, TOP, 0.19, 0.085, 0);
    const front = (r, z, n = 64) => R.arc(r, z, n);                                      // 前半圈：左→右
    const full = (r, z, n = 96) => Array.from({ length: n }, (_, i) => R.p(r, Math.PI * 2 * i / n, z));
    const ang = (n, off = 0.5) => Array.from({ length: n }, (_, i) => Math.PI * 2 * (i + off) / n);
    const vis = th => Math.sin(th) < -0.02;                                                    // 看得见的前半圈

    /* ── 圆柱面带：r 半径，z0→z1 ── */
    const band = (r, z0, z1, cls) => P.p(poly([...front(r, z0), ...front(r, z1).reverse()]), cls);
    const shade = (r, z0, z1) => P.p(poly([...front(r, z0), ...front(r, z1).reverse()]), 'g-drum');

    /* ── 一层坛：坛面（整圈椭圆）、侧面（前半圈），栏杆另画 ── */
    const T = [{ r: 336, z0: 0, z1: 19 }, { r: 296, z0: 19, z1: 38 }, { r: 256, z0: 38, z1: 57 }];
    const railH = 10.5, stairW = 26;
    function tierBody(t) {
      P.p(poly(full(t.r, t.z1)), 'c-marble l2');                                               // 坛面
      // 须弥座式侧面：上枋、束腰（暗）、下枋，前沿一圈莲瓣
      band(t.r, t.z0, t.z1, 'c-marble l2');
      band(t.r - 2.2, t.z0 + 5.5, t.z1 - 5, 'c-marble-d l3');
      let pet = '';
      for (const th of ang(Math.round(t.r * 0.42))) if (vis(th)) {
        const [x, y] = R.p(t.r, th, t.z1 - 4.2), w = 3.4 * Math.abs(Math.sin(th));
        pet += `M${P2(x - w, y)}Q${P2(x - w, y + 2.6)} ${P2(x, y + 2.6)}Q${P2(x + w, y + 2.6)} ${P2(x + w, y)}`;
      }
      P.p(pet, 'l3');
      shade(t.r, t.z0, t.z1);
      P.p('M' + front(t.r, t.z1).map(([x, y]) => P2(x, y)).join('L'), 'l2');
    }
    function rail(t, half) {
      // half: 'front' 前半圈（画在上一层坛身之后）/ 'back' 后半圈
      const r = t.r - 3, z = t.z1, n = Math.round(r * 0.36);
      const list = ang(n).filter(th => half === 'front' ? vis(th) : Math.sin(th) > 0.02);
      const pts = (zz, th0, th1) => samp(k => R.p(r, th0 + (th1 - th0) * k, zz), 6);
      let pan = '', rl = '', post = '', head = '';
      for (let i = 0; i < list.length - 1; i++) {
        const a = list[i], b = list[i + 1];
        const [xa] = R.p(r, a, z), [xb] = R.p(r, b, z);
        if (half === 'front' && Math.abs((xa + xb) / 2) < stairW + 4) continue;              // 正中让给台阶
        pan += poly([...pts(z + 0.8, a, b), ...pts(z + railH * 0.42, a, b).reverse()]);
        rl += poly([...pts(z + railH * 0.6, a, b), ...pts(z + railH * 0.7, a, b).reverse()]);
      }
      for (const th of list) {
        const [x, y] = R.p(r, th, z); if (half === 'front' && Math.abs(x) < stairW + 2) continue;
        const pw = 1.6;
        post += `M${P2(x - pw / 2, y)}h${N(pw)}v${N(railH * 0.82)}h${N(-pw)}Z`;
        const yh = y + railH * 0.82;
        head += `M${P2(x - pw * 0.6, yh)}h${N(pw * 1.2)}c0 ${N(railH * 0.1)} ${N(-pw * 0.3)} ${N(railH * 0.16)} ${N(-pw * 0.6)} ${N(railH * 0.17)}s${N(-pw * 0.6)} ${N(-railH * 0.07)} ${N(-pw * 0.6)} ${N(-railH * 0.17)}Z`;
      }
      P.p(pan, 'c-marble l3'); P.p(rl, 'c-marble l3'); P.p(post, 'c-marble l3'); P.p(head, 'c-marble l3');
    }

    /* ── 殿身一层：柱、柱间隔扇（或槛窗）、额枋彩画、斗拱一圈 ── */
    function drum(r, z0, z1, nCol, fill, beamH, dgH) {
      const cols = ang(nCol, 0.5);
      // 柱间
      for (let i = 0; i < nCol; i++) {
        const a = cols[i], b = cols[(i + 1) % nCol] + (i === nCol - 1 ? Math.PI * 2 : 0);
        const m = (a + b) / 2; if (!vis(m)) continue;
        const k = samp(t => a + (b - a) * t, 8);
        const lo = k.map(th => R.p(r, th, z0 + 1)), hi = k.map(th => R.p(r, th, z1));
        P.p(poly([...lo, ...hi.slice().reverse()]), fill === 'door' ? 'p-lattice l3' : fill === 'win' ? 'p-lattice2 l3' : 'c-red l3');
        // 隔扇的抹头、裙板：几道顺着圆周的横线
        const rows = fill === 'door' ? [0.06, 0.27, 0.32, 0.92] : [0.2, 0.86];
        let d = ''; for (const f of rows) d += 'M' + k.map(th => P2(...R.p(r, th, z0 + (z1 - z0) * f))).join('L');
        P.p(d, 'l3');
        if (fill === 'door') {
          P.p(poly([...k.map(th => R.p(r, th, z0 + 1)), ...k.slice().reverse().map(th => R.p(r, th, z0 + (z1 - z0) * 0.27))]), 'c-red l3');
          // 竖向分扇（四扇）
          let v = ''; for (const f of [0.25, 0.5, 0.75]) { const th = a + (b - a) * f; v += `M${P2(...R.p(r, th, z0 + 1))}L${P2(...R.p(r, th, z1))}`; }
          P.p(v, 'l3');
        }
      }
      // 柱：圆柱不缩窄
      const cw = Math.max(5, r * 0.055);
      let cd = '';
      for (const th of cols) if (vis(th) || Math.abs(Math.sin(th)) < 0.2 && Math.cos(th) !== 0) {
        if (Math.sin(th) > 0.02) continue;
        const [x, yb] = R.p(r, th, z0), [, yt] = R.p(r, th, z1);
        cd += `M${P2(x - cw / 2, yb)}h${N(cw)}V${N(yt)}h${N(-cw)}Z`;
      }
      P.p(cd, 'g-col l2');
      shade(r, z0, z1);
      // 额枋：一圈青绿彩画，金线勾"<>"枋心
      const zb = z1 + beamH;
      band(r + 0.6, z1, zb, 'c-qing l3');
      for (let i = 0; i < nCol; i++) {
        const a = cols[i], b = cols[(i + 1) % nCol] + (i === nCol - 1 ? Math.PI * 2 : 0), m = (a + b) / 2;
        if (!vis(m)) continue;
        const f0 = a + (b - a) * 0.3, f1 = a + (b - a) * 0.7;
        const top = samp(t => R.p(r + 0.7, f0 + (f1 - f0) * t, zb - beamH * 0.16), 6), bot = samp(t => R.p(r + 0.7, f0 + (f1 - f0) * t, z1 + beamH * 0.16), 6);
        const L = R.p(r + 0.7, a + (b - a) * 0.24, z1 + beamH * 0.5), Rr = R.p(r + 0.7, a + (b - a) * 0.76, z1 + beamH * 0.5);
        P.p(poly([L, ...bot, Rr, ...top.slice().reverse()]), 'c-lu l3');
        P.p(poly([L, ...bot, Rr, ...top.slice().reverse()]), 'lg');
        const mid = samp(t => R.p(r + 0.7, a + (b - a) * (0.34 + 0.32 * t), z1 + beamH * (0.4 + 0.2 * Math.sin(t * Math.PI * 3))), 10);
        P.p('M' + mid.map(([x, y]) => P2(x, y)).join('L'), 'lg2');
      }
      P.p('M' + front(r + 0.6, z1).map(([x, y]) => P2(x, y)).join('L') + 'M' + front(r + 0.6, zb).map(([x, y]) => P2(x, y)).join('L'), 'lg');
      shade(r + 0.6, z1, zb);
      // 斗拱：每间三攒平身科 + 柱头科，随圆周缩窄
      const zd = zb, zd1 = zd + dgH;
      band(r + 1, zd, zd1 - dgH * 0.22, 'c-red l3');
      const per = 3, list = [];
      for (let i = 0; i < nCol; i++) for (let k = 0; k <= per; k++) list.push(cols[i] + (Math.PI * 2 / nCol) * k / (per + 1));
      let dou = '', arm = '', arm2 = '', gold = '';
      for (const th of list) {
        if (!vis(th)) continue;
        const s = Math.abs(Math.sin(th)), [x, y0] = R.p(r + 2, th, zd), [, y1] = R.p(r + 2, th, zd + dgH * 0.25), [, y2] = R.p(r + 2, th, zd + dgH * 0.5), [, y3] = R.p(r + 2, th, zd + dgH * 0.75);
        const d = 1.15 * s;
        dou += `M${P2(x - d * 1.4, y0)}h${N(d * 2.8)}l${N(d * 0.3)} ${N(y1 - y0)}h${N(-d * 3.4)}Z`;
        arm += `M${P2(x - d * 3.2, y1)}h${N(d * 6.4)}v${N(y2 - y1)}h${N(-d * 6.4)}Z`;
        arm2 += `M${P2(x - d * 4.4, y2)}h${N(d * 8.8)}v${N(y3 - y2)}h${N(-d * 8.8)}Z`;
        gold += `M${P2(x - d * 3, (y1 + y2) / 2)}h${N(d * 6)}`;
      }
      P.p(dou, 'c-qing l3'); P.p(arm, 'c-lu l3'); P.p(arm2, 'c-qing l3'); P.p(gold, 'lg');
      band(r + 2.4, zd1 - dgH * 0.25, zd1, 'c-lu l3');
      return zd1;
    }

    /* ── 圆攒尖屋面：举架凹曲面，瓦垄沿经线，檐下两排椽头 ── */
    function roof(re, ze, rt, zt, rIn, zIn, nTiles, tb) {
      const prof = t => cub([re, ze], [re - (re - rt) * 0.56, ze + (zt - ze) * 0.08], [rt + (re - rt) * 0.1, ze + (zt - ze) * 0.56], [rt, zt], t);
      const ts = samp(t => t, 20);
      // 檐下：从檐口前沿到斗拱外沿，深色，两排椽头
      P.p(poly([...front(re, ze), ...front(rIn, zIn).reverse()]), 'c-soffit');
      let fly = '', rnd = '';
      for (const th of ang(Math.round(re * 0.72))) {
        if (!vis(th)) continue;
        const s = Math.abs(Math.sin(th)), [x, y] = R.p(re - 1.2, th, ze), [x2, y2] = R.p(re - 4.6, th, ze - 2.4), w = 1.15 * s;
        fly += `M${P2(x - w, y - 0.4)}h${N(w * 2)}v-2.2h${N(-w * 2)}Z`;
        rnd += `M${P2(x2 - w, y2 - 1.6)}a${N(w)} 1.15 0 1 0 ${N(w * 2)} 0a${N(w)} 1.15 0 1 0 ${N(-w * 2)} 0Z`;
      }
      P.p(fly, 'c-lu l3'); P.p(rnd, 'c-qing l3');
      // 屋面
      const left = ts.map(t => { const [r, z] = prof(t); return [-r, z]; }), right = ts.map(t => { const [r, z] = prof(t); return [r, z]; });
      const surf = [...front(re, ze + tb), ...right.slice(1), ...front(rt, zt).reverse(), ...left.slice(1).reverse()];
      P.p(poly(surf), 'g-blue l1');
      // 瓦垄：经线，近两侧越来越密、越来越弯
      let dk = '', lt = '';
      for (const th of ang(nTiles)) {
        if (!vis(th)) continue;
        const pts = ts.map(t => { const [r, z] = prof(t); return R.p(r, th, z + (t === 0 ? tb : 0)); });
        dk += 'M' + pts.map(([x, y]) => P2(x, y)).join('L');
        const th2 = th + Math.PI / nTiles * 0.5, pts2 = ts.slice(1, -2).map(t => { const [r, z] = prof(t); return R.p(r, th2, z); });
        lt += 'M' + pts2.map(([x, y]) => P2(x, y)).join('L');
      }
      P.p(dk, 'l3'); P.p(lt, 'lw');
      // 屋面体积：左亮右暗
      P.p(poly(surf), 'g-drum');
      // 檐口带与勾头
      P.p(poly([...front(re, ze), ...front(re, ze + tb).reverse()]), 'c-blue-d l2');
      let heads = '';
      for (const th of ang(nTiles)) {
        if (!vis(th)) continue;
        const s = Math.abs(Math.sin(th)), [x, y] = R.p(re, th, ze + tb * 0.5), rr = Math.min(tb * 0.36, 1.3);
        heads += `M${P2(x - rr * s, y)}a${N(rr * s)} ${N(rr)} 0 1 0 ${N(rr * s * 2)} 0a${N(rr * s)} ${N(rr)} 0 1 0 ${N(-rr * s * 2)} 0Z`;
      }
      P.p(heads, 'c-blue-l l3');
      return prof;
    }

    // ── 按前后次序画 ─────────────────────────────
    // 坛：底层坛身 → 中层坛身 → 底层前栏 → 上层坛身 → 中层前栏；后栏先画
    tierBody(T[0]); rail(T[0], 'back');
    tierBody(T[1]); rail(T[1], 'back');
    rail(T[0], 'front');
    tierBody(T[2]); rail(T[2], 'back');
    rail(T[1], 'front');
    // 殿基
    P.p(poly(full(140, 61)), 'c-marble l2');
    band(140, 57, 61, 'c-marble l2');
    // 殿身三层、屋面三重，逐层收进；檐口出挑约 3 到 4 米
    const zA = drum(118, 61, 115, 12, 'door', 9, 11);
    roof(158, zA - 3, 98, zA + 41, 123, zA - 0.5, 62, 4.6);
    const zB = drum(96, zA + 41, zA + 52, 12, 'win', 7, 9);
    roof(128, zB - 3, 74, zB + 41, 100, zB - 0.5, 50, 4.2);
    const zC = drum(72, zB + 41, zB + 50, 12, 'win', 6, 8);
    roof(98, zC - 3, 7, zC + 64, 76, zC - 0.5, 40, 3.8);
    // 宝顶
    const zTop = zC + 64;
    P.g(sub(Q => A.baoding(Q, 0, 7.2)), ` transform="translate(0 ${N(zTop - 3)})"`);
    // 上层前栏、正面台阶（三层一路上去，中间御路）
    rail(T[2], 'front');
    const yBot = R.p(T[0].r, Math.PI * 1.5, 0)[1], yTop = R.p(T[2].r, Math.PI * 1.5, 57)[1];
    const w0 = stairW, w1 = stairW * 0.86;
    P.p(poly([[-w0, yBot], [w0, yBot], [w1, yTop], [-w1, yTop]]), 'c-marble l2');
    let st = ''; const ns = 26; for (let i = 1; i < ns; i++) { const f = i / ns, y = yBot + (yTop - yBot) * f, w = w0 + (w1 - w0) * f; st += `M${P2(-w, y)}H${N(w)}`; }
    P.p(st, 'l3');
    const rw0 = w0 * 0.36, rw1 = w1 * 0.36;
    P.p(poly([[-rw0, yBot], [rw0, yBot], [rw1, yTop], [-rw1, yTop]]), 'c-marble l3');
    let cl = ''; for (let i = 0; i < 6; i++) { const f = 0.08 + i * 0.16, y = yBot + (yTop - yBot) * f, w = rw0 + (rw1 - rw0) * f; cl += `M${P2(-w * 0.7, y)}c${N(w * 0.3)} 2.4 ${N(w * 0.6)} -1.6 ${N(w * 0.7)} 0s${N(w * 0.4)} 2 ${N(w * 0.7)} 0`; }
    P.p(cl, 'l3');
    P.p(`M${P2(-w0, yBot)}L${P2(-w1, yTop)}M${P2(w0, yBot)}L${P2(w1, yTop)}`, 'l2');
    return { g: P.s(), box: [-342, 342, yBot - 2, zTop + 38] };
  }

  window.JH.LM = Object.assign(window.JH.LM || {}, { qinian });
})();
