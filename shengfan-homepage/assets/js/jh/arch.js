/* 界画库 · 建筑构件（正立面，局部坐标 u 右 v 上）
   屋顶：庑殿（wd）、歇山（xs）、攒尖（cj）、腰檐（yao，重檐的下层，上沿被墙挡住）。
     屋面按"举架"做成凹曲面：檐口平缓、近脊陡峭；檐口线中段平、两端起翘，翼角冲出上挑；
     正脊两端鸱吻，垂脊、戗脊顺屋面曲线走，脊端套兽，脊上仙人走兽。
   斗拱：按"攒"排，柱头科在柱上，平身科在两柱之间，攒与攒之间是红色垫拱板。
   额枋：和玺或旋子彩画，青绿相间，金线勾边。
   柱：朱红圆柱带明暗，古镜柱础。隔扇：菱花格心、绦环板、裙板。
   台基：须弥座线脚（圭角、下枋、下枭、束腰、上枭、上枋），莲瓣、螭首；汉白玉栏杆（望柱、柱头、寻杖、净瓶、华板）；台阶带御路。 */
(function () {
  'use strict';
  const { N, P2, clamp, lerp, Pen, sub, cub, samp, poly, line, yAt } = window.JH;

  /* ── 渐变与纹样（页面里放一次） ───────────────────
     建筑在翻转坐标里画，竖向渐变 offset 0 在视觉的"下"。 */
  const lin = (id, stops, x1 = 0, y1 = 0, x2 = 0, y2 = 1) => `<linearGradient id="${id}" x1="${x1}" y1="${y1}" x2="${x2}" y2="${y2}">${stops.map(([o, c, a = 1]) => `<stop offset="${o}" stop-color="${c}"${a < 1 ? ` stop-opacity="${a}"` : ''}/>`).join('')}</linearGradient>`;
  function defs() {
    return '<svg width="0" height="0" style="position:absolute" aria-hidden="true"><defs>' +
      lin('jh-yellow', [[0, '#c98a2a'], [.35, '#dfa33a'], [1, '#f0c766']]) +
      lin('jh-blue', [[0, '#1a3a77'], [.4, '#2a529a'], [1, '#4f77bb']]) +
      lin('jh-green', [[0, '#2b6a4c'], [.4, '#3b8763'], [1, '#6aac88']]) +
      lin('jh-gray', [[0, '#5d6462'], [.4, '#7b817e'], [1, '#a5aaa5']]) +
      lin('jh-col', [[0, '#d8644c'], [.28, '#bf4632'], [.7, '#9a2e20'], [1, '#6f1c12']], 0, 0, 1, 0) +
      lin('jh-drum', [[0, '#fff6e0', .28], [.3, '#fff6e0', 0], [.62, '#140c06', 0], [1, '#140c06', .38]], 0, 0, 1, 0) +
      lin('jh-shan', [[0, '#c49a6a'], [.28, '#b7a073'], [.5, '#6f9f80'], [.78, '#4f8a8e'], [1, '#3a6a8c']]) +
      lin('jh-bark', [[0, '#9c7a58'], [.45, '#7c5c42'], [1, '#4f3a2a']], 0, 0, 1, 0) +
      lin('jh-sky', [[0, '#f3e9d6'], [.55, '#efe4cf'], [1, '#ebdfc7']]) +
      lin('jh-water', [[0, '#dfe5dc'], [.35, '#d3ded8'], [1, '#c3d3cf']]) +
      lin('jh-lake', [[0, '#d7e0d9'], [1, '#c9d7d2']]) +
      '<radialGradient id="jh-glow"><stop offset="0" stop-color="#f7dcae" stop-opacity=".55"/><stop offset="1" stop-color="#f7dcae" stop-opacity="0"/></radialGradient>' +
      '<radialGradient id="jh-mistr"><stop offset="0" stop-color="#f7f1e3" stop-opacity=".95"/><stop offset=".55" stop-color="#f7f1e3" stop-opacity=".7"/><stop offset="1" stop-color="#f7f1e3" stop-opacity="0"/></radialGradient>' +
      lin('jh-haze', [[0, '#efe6d3', 0], [.1, '#efe6d3', .36], [.9, '#efe6d3', .36], [1, '#efe6d3', 0]], 0, 0, 1, 0) +
      '<radialGradient id="jh-hazer"><stop offset="0" stop-color="#efe6d3" stop-opacity=".4"/><stop offset=".72" stop-color="#efe6d3" stop-opacity=".34"/><stop offset="1" stop-color="#efe6d3" stop-opacity="0"/></radialGradient>' +
      lin('jh-mapbg', [[0, '#f4ecdb'], [1, '#ede2c9']]) +
      lin('jh-hazeb', [[0, '#efe6d3', 0], [.28, '#efe6d3', .85], [1, '#efe6d3', 1]]) +
      lin('jh-lawn', [[0, '#c3d79c'], [.4, '#b2cb8a'], [1, '#9dbb78']]) +
      lin('jh-shan-far', [[0, '#d8cdb3'], [.3, '#c9c6b0'], [.62, '#a8b7a9'], [1, '#8fa6ab']]) +
      lin('jh-shan-mid', [[0, '#c7a879'], [.32, '#b5a77e'], [.6, '#7fa48a'], [1, '#5a8890']]) +
      lin('jh-mist', [[0, '#f3ebdb', 0], [.45, '#f6efe0', .92], [.6, '#f6efe0', .92], [1, '#f3ebdb', 0]]) +
      lin('jh-rocktop', [[0, '#6f9f80', 0], [.35, '#5f987a', .88], [1, '#3f6f8e']]) +
      lin('jh-cloud', [[0, '#ecdab4'], [.5, '#faf3e2'], [1, '#fffdf7']]) +
      lin('jh-gold', [[0, '#8f6418'], [.22, '#e8c46a'], [.42, '#fff0b8'], [.6, '#d7a944'], [1, '#7c5512']], 0, 0, 1, 0) +
      // 三交六椀菱花：暗红底、金色菱花线
      '<pattern id="jh-lattice" width="2.6" height="4.5" patternUnits="userSpaceOnUse"><rect width="2.6" height="4.5" fill="#6a1f15"/><path d="M0 0L2.6 4.5M2.6 0L0 4.5M0 2.25H2.6" fill="none" stroke="#d6a457" stroke-width=".22" opacity=".85"/><circle cx="1.3" cy="2.25" r=".42" fill="none" stroke="#d6a457" stroke-width=".16"/><circle cx="0" cy="0" r=".42" fill="none" stroke="#d6a457" stroke-width=".16"/><circle cx="2.6" cy="4.5" r=".42" fill="none" stroke="#d6a457" stroke-width=".16"/></pattern>' +
      // 直棂窗
      '<pattern id="jh-lattice2" width="1.6" height="4" patternUnits="userSpaceOnUse"><rect width="1.6" height="4" fill="#5e1d14"/><path d="M.8 0V4" stroke="#c99a52" stroke-width=".32"/></pattern>' +
      '<pattern id="jh-brickp" width="5" height="2" patternUnits="userSpaceOnUse"><path d="M0 0H5M0 1H5M0 0V1M2.5 1V2" fill="none" stroke="#4a4844" stroke-width=".14" opacity=".55"/></pattern>' +
      '</defs></svg>';
  }

  const TONE = { yellow: 'yellow', blue: 'blue', green: 'green', gray: 'gray' };

  // 用 (u, v) 写的路径（只用 M L C Q Z 的点对）映射到画面
  const mapPath = (d, f) => d.replace(/(-?\d*\.?\d+)[ ,](-?\d*\.?\d+)/g, (_, a, b) => { const [x, y] = f(+a, +b); return P2(x, y); });


  /* ── 鸱吻（正吻）：正脊两端。龙首在下、张口朝里衔住正脊；身子直立，外沿一排背鳍、中段背兽；
        顶上尾巴向外翻卷再旋回；尾后斜插剑把。x0、y0 是正脊端头的脊顶；dir = 1 在右端（尾朝右），-1 在左端；h 高度。 ── */
  function chiwen(P, x0, y0, h, dir, tone) {
    const w = h * 0.7, f = (u, v) => [x0 + dir * u * w, y0 + v * h], m = d => mapPath(d, f);
    P.p(m('M.5 .97L.62 .97L.7 1.3L.58 1.32Z'), `c-${tone}-d l3`);
    P.p(m('M-.04 -.2C.06 -.24 .3 -.22 .42 -.12C.6 -.04 .8 0 .86 .12L.86 .66C.9 .74 .98 .8 1 .88C1.02 1 .9 1.06 .78 1.02C.62 .98 .42 .96 .3 .97C.16 .98 .06 .92 .04 .82L.04 .36C0 .32 -.06 .3 -.1 .26L-.12 .14C-.06 .12 .02 .1 .08 .08L.04 0L-.02 -.08C-.06 -.1 -.08 -.16 -.04 -.2Z'), `c-${tone} l2`);
    if ((window.JH.lod ?? 1) < 0.7) return;
    P.p(m('M.62 .1C.74 .1 .86 .14 .86 .2L.86 .66C.8 .62 .7 .6 .62 .62Z'), `c-${tone}-d`);
    P.p(m('M.08 .08L.04 0L-.02 -.08C.02 -.04 .06 0 .08 .08Z'), `c-${tone}-d`);
    let fin = '', sc = '';
    for (let v = 0.18; v < 0.6; v += 0.1) fin += m(`M.86 ${N(v)}L.95 ${N(v + 0.05)}L.86 ${N(v + 0.1)}Z`);
    for (let v = 0.44; v < 0.86; v += 0.12) for (let u = 0.14; u < 0.72; u += 0.14) sc += m(`M${N(u)} ${N(v)}Q${N(u + 0.07)} ${N(v + 0.07)} ${N(u + 0.14)} ${N(v)}`);
    P.p(fin, `c-${tone} l3`); P.p(sc, 'l3');
    P.p(m('M.78 1.02C.9 .98 .93 .86 .85 .82C.77 .78 .7 .86 .76 .9'), 'l2');
    P.p(m('M.08 .27C.14 .31 .22 .3 .26 .25M.26 .3C.36 .36 .46 .32 .44 .22M.3 .16C.4 .2 .5 .16 .48 .06M-.1 .2L-.04 .2'), 'l3');
    P.p(m('M.86 .26C.95 .25 1.02 .3 1.02 .37C1.02 .44 .95 .48 .86 .46Z'), `c-${tone} l3`);
    const [ex, ey] = f(0.16, 0.2); P.c(ex, ey, h * 0.032, 'c-ink');
    const [bx, by] = f(0.95, 0.38); P.c(bx, by, h * 0.016, 'c-ink');
  }

  /* ── 走兽：侧影，头朝屋角。kind：'lead' 仙人骑凤、'sit' 坐兽、'head' 垂兽（戗兽，大兽头）
     x、y 是兽座落在脊上的点，s 是高度，dir 朝向（1 朝右）。 */
  function beast(P, x, y, s, dir, tone, kind = 'sit') {
    const f = (u, v) => [x + dir * (u - 0.45) * s, y + v * s], m = d => mapPath(d, f);
    if (kind === 'lead') {
      // 凤：昂首，尾羽后扬；仙人端坐凤背，戴幞头
      P.p(m('M.12 .22C-.04 .26 -.14 .46 -.1 .64C-.04 .52 .06 .38 .2 .32Z'), `c-${tone} l3`);
      P.p(m('M.06 .14C.12 .3 .3 .38 .5 .36C.62 .36 .68 .46 .7 .58C.71 .68 .78 .74 .85 .7L.98 .64L.86 .6C.8 .56 .78 .46 .76 .36C.74 .22 .64 .1 .5 .08L.5 0L.3 0L.3 .06C.2 .06 .1 .08 .06 .14Z'), `c-${tone} l3`);
      P.p(m('M.22 .3C.34 .22 .48 .22 .6 .3M.78 .73C.76 .8 .8 .84 .85 .83'), 'l3');
      P.p(m('M.3 .34C.28 .5 .3 .66 .36 .74C.4 .78 .5 .78 .52 .72C.56 .6 .58 .46 .62 .36Z'), `c-${tone}-l l3`);
      P.p(m('M.48 .66C.54 .6 .6 .58 .64 .61'), 'l3');
      P.p(m('M.37 .78C.36 .88 .4 .93 .44 .93C.49 .93 .52 .88 .51 .78Z'), `c-${tone}-l l3`);
      P.p(m('M.36 .9L.52 .9L.51 .98L.37 .98Z'), `c-${tone}-d l3`);
      const [ex, ey] = f(0.83, 0.665); P.c(ex, ey, s * 0.022, 'c-ink');
      return;
    }
    if (kind === 'head') {
      // 垂兽：大兽头，鬃毛后披，独角，张口
      P.p(m('M.12 0L.12 .4C.02 .5 .04 .7 .16 .76C.12 .88 .2 .98 .32 .96L.34 1.16C.4 1.12 .44 1.04 .46 .96C.58 .98 .7 .94 .76 .86C.86 .84 .96 .8 .97 .72C.97 .66 .9 .62 .84 .64L.7 .6L.86 .5C.86 .44 .8 .4 .7 .42C.64 .3 .64 .14 .7 0Z'), `c-${tone} l3`);
      P.p(m('M.16 .58C.22 .66 .3 .62 .26 .55M.2 .82C.27 .9 .34 .85 .3 .78M.58 .86C.64 .91 .72 .9 .74 .84M.12 .2L.66 .2'), 'l3');
      P.p(m('M.12 0L.12 .2L.66 .2L.7 0Z'), `c-${tone}-d l3`);
      const [ex, ey] = f(0.67, 0.79); P.c(ex, ey, s * 0.03, 'c-ink');
      return;
    }
    // 坐兽：后腿蹲坐、前腿直立、挺胸昂首，尾巴上卷
    P.p(m('M.1 .26C-.06 .28 -.14 .5 -.06 .64C-.02 .7 .08 .7 .08 .6C.06 .66 0 .64 -.02 .58C-.06 .46 .02 .36 .12 .36Z'), `c-${tone} l3`);
    P.p(m('M.08 0C.02 .14 .02 .34 .12 .44C.2 .52 .3 .6 .36 .7C.34 .82 .4 .96 .52 .98C.6 1.04 .7 1.02 .76 .94L.92 .9C.98 .88 .98 .8 .92 .78L.82 .78L.9 .72C.86 .66 .76 .66 .7 .68C.74 .58 .76 .5 .72 .42C.7 .34 .74 .22 .74 .06L.86 .04L.86 0Z'), `c-${tone} l3`);
    P.p(m('M.14 .06C.12 .26 .3 .36 .44 .26C.5 .2 .5 .1 .46 .02M.62 .4L.62 .02M.42 .84C.46 .9 .52 .88 .5 .82M.4 .72C.44 .78 .5 .76 .48 .7'), 'l3');
    const [ex, ey] = f(0.74, 0.875); P.c(ex, ey, s * 0.03, 'c-ink');
  }

  /* ── 套兽：仔角梁头上套的兽头，朝外 ─────────────── */
  function taoshou(P, x, y, s, dir, tone) {
    const f = (u, v) => [x + dir * u * s, y + (v - 0.32) * s], m = d => mapPath(d, f);
    P.p(m('M0 .32C.2 .44 .5 .46 .66 .38C.8 .34 .92 .28 .94 .18C.92 .1 .84 .1 .78 .14L.64 .08L.82 -.02C.76 -.1 .62 -.12 .52 -.06C.36 -.12 .12 -.06 0 0Z'), `c-${tone} l3`);
    P.p(m('M0 0C.12 -.06 .36 -.12 .52 -.06C.4 -.02 .2 .02 0 .08Z'), `c-${tone}-d`);
    P.p(m('M.44 .33C.5 .37 .58 .35 .6 .3M.18 .28C.24 .34 .3 .3 .26 .23'), 'l3');
    const [ex, ey] = f(0.52, 0.25); P.c(ex, ey, s * 0.035, 'c-ink');
  }

  /* ── 屋顶（正立面） ────────────────────────────
     o: ey 檐口线中点高、ew 檐口半宽、rh 屋面高（檐口到正脊上沿）、rw 正脊半宽（攒尖为 0）
        type: 'wd' 庑殿 | 'xs' 歇山（看长边）| 'xsg' 歇山（看山面：山花、博风、悬鱼朝前）| 'cj' 攒尖 | 'yao' 腰檐
        tone 琉璃色；trim 剪边色；lift 起翘；tile 瓦垄间距；ridgeH 脊高；beasts 走兽数；sd 檐下深；wallW 腰檐处的墙半宽；sides 攒尖边数
        ends: { l: 'cut', r: 'cut' } 某一端不做翼角，直接截断（窝角处接别的屋面）
        xsg 另有：gw 山花底半宽、gy 山花底高（占 rh 的比例）、apex: 'wen' | 'baoding' | 'none' */
  function roof(P, o) {
    const tone = o.tone || 'yellow', type = o.type || 'wd', trim = o.trim || tone, lod = window.JH.lod ?? 1;
    const ew = o.ew, ey = o.ey, rh = o.rh, rw = type === 'cj' ? 0 : (o.rw ?? 0);
    const tb = o.band ?? rh * 0.075;                    // 檐口带（连檐、瓦口、勾头一排）
    const lift = o.lift ?? rh * 0.2;                    // 起翘
    const fx = o.fx ?? ew * 0.045, fy = o.fy ?? lift * 0.75;   // 翼角冲出、上挑
    const ridgeH = o.ridgeH ?? rh * 0.11;
    const top = ey + rh;
    const t0 = o.t0 ?? 0.42;
    const cut = { [-1]: o.ends?.l === 'cut', [1]: o.ends?.r === 'cut' };
    // 檐口线：中段平，近角处按幂曲线起翘（截断的一端不起翘）
    const E = u => { const s = u < 0 ? -1 : 1, t = Math.abs(u) / ew; return ey + (cut[s] || t <= t0 ? 0 : lift * Math.pow((t - t0) / (1 - t0), 2.3)); };
    const slope = lift * 2.3 / ((1 - t0) * ew), eE = s => E(s * ew);
    // 翼角：檐口带出了檐端继续上挑，端头收成一截仔角梁头，下挂套兽
    const lowTip = s => cut[s] ? [s * ew, ey] : [s * (ew + fx), eE(s) + fy];
    const upTip = s => cut[s] ? [s * ew, ey + tb] : [s * (ew + fx + tb * 0.12), eE(s) + fy + tb * 0.72];
    const cLo = s => [s * (ew + fx * 0.55), eE(s) + slope * fx * 0.55], cUp = s => [s * (ew + fx * 0.5), eE(s) + tb + slope * fx * 0.5];
    const toUp = s => cut[s] ? '' : `Q${P2(...cUp(s))} ${P2(...upTip(s))}`, fromUp = s => cut[s] ? '' : `Q${P2(...cUp(s))} ${P2(s * ew, eE(s) + tb)}`;
    const toLo = s => cut[s] ? '' : `Q${P2(...cLo(s))} ${P2(...lowTip(s))}`, fromLo = s => cut[s] ? '' : `Q${P2(...cLo(s))} ${P2(s * ew, eE(s))}`;
    // 两侧轮廓：从翼角顺戗脊（凹曲线）到屋顶；截断的一端直上
    const G = type === 'xs' ? [rw + rh * 0.14, ey + rh * 0.42] : type === 'xsg' ? [o.gw ?? ew * 0.6, ey + rh * (o.gy ?? 0.38)] : null;
    const sideOf = s => {
      const T = upTip(s);
      const R = type === 'wd' || type === 'cj' ? [s * rw, top] : G ? [s * G[0], G[1]] : [s * o.wallW, top];
      if (cut[s]) { const yt = type === 'xsg' ? G[1] : top; return [T, [T[0], T[1] + (yt - T[1]) / 3], [T[0], T[1] + (yt - T[1]) * 2 / 3], [T[0], yt]]; }
      const c2 = G ? [R[0] + (T[0] - R[0]) * 0.14, R[1] - rh * 0.1] : type === 'yao' ? [R[0] + (T[0] - R[0]) * 0.12, top - rh * 0.4] : [R[0] + (T[0] - R[0]) * 0.1, top - rh * 0.44];
      return [T, [T[0] - (T[0] - R[0]) * 0.52, T[1] + rh * 0.03], c2, R];
    };
    const sR = sideOf(1), sL = sideOf(-1);
    const sidePts = samp(t => cub(...sR, t), 30), sidePtsL = samp(t => cub(...sL, t), 30);
    // 歇山山面：博风外沿（排山勾滴）从山花底角凹曲而上到山尖
    const bof = G && type === 'xsg' ? samp(t => cub([G[0], G[1]], [G[0] * 0.55, G[1] + (top - G[1]) * 0.35], [G[0] * 0.18, G[1] + (top - G[1]) * 0.78], [0, top], t), 24) : null;
    // 檐口带上沿
    const bandTop = samp(t => { const u = -ew + 2 * ew * t; return [u, E(u) + tb]; }, 60);
    // 屋面外轮廓
    let outline = `M${P2(...upTip(-1))}${fromUp(-1)}` + bandTop.slice(1, -1).map(p => `L${P2(...p)}`).join('') + `L${P2(ew, eE(1) + tb)}${toUp(1)}`;
    outline += `C${P2(...sR[1])} ${P2(...sR[2])} ${P2(...sR[3])}`;
    if (type === 'xs') outline += `L${P2(rw * 0.985, top)}L${P2(-rw * 0.985, top)}L${P2(...sL[3])}`;
    else if (type === 'xsg') outline += bof.slice(1).map(p => `L${P2(...p)}`).join('') + bof.slice().reverse().slice(1).map(([x, y]) => `L${P2(-x, y)}`).join('');
    else if (type === 'wd') outline += `L${P2(-rw, top)}`;
    else if (type === 'yao') outline += `L${P2(...sL[3])}`;
    outline += `C${P2(...sL[2])} ${P2(...sL[1])} ${P2(...sL[0])}Z`;
    // 檐下（看得见的檐底阴影）与椽头
    const sd = o.sd ?? rh * 0.2;
    const under = samp(t => { const u = -ew + 2 * ew * t; return [u, E(u) - sd * (1 - 0.35 * Math.pow(Math.abs(u) / ew, 3))]; }, 60);
    const eave = samp(t => { const u = -ew + 2 * ew * t; return [u, E(u)]; }, 60);
    P.p(poly([lowTip(-1), ...eave, lowTip(1), ...under.slice().reverse()]), 'c-soffit');
    const rs = o.rafter ?? Math.max(1.2, (o.tile ?? ew / 16) * 1.05);
    let fly = '', round = '';
    for (let u = -ew * 0.97 + rs / 2; u < ew * 0.97; u += rs) {
      const y = E(u), k = Math.abs(u) / ew, sz = rs * 0.36;
      fly += `M${P2(u - sz, y - sz * 0.15)}h${N(sz * 2)}v${N(-sz * 1.6)}h${N(-sz * 2)}Z`;
      round += `M${P2(u - sz * 0.95, y - sz * 3.1)}a${N(sz * 0.95)} ${N(sz * 0.95)} 0 1 0 ${N(sz * 1.9)} 0a${N(sz * 0.95)} ${N(sz * 0.95)} 0 1 0 ${N(-sz * 1.9)} 0Z`;
      if (k > 0.97) break;
    }
    if (lod >= 0.7) { P.p(fly, 'c-lu l3'); P.p(round, 'c-qing l3'); }
    // 屋面
    P.p(outline, `g-${tone} l1`);
    // 瓦垄：自檐口带上沿竖直通到正脊、博脊或戗脊，每垄一道暗缝一道亮脊
    const tile = (o.tile ?? ew / 16) / (lod >= 0.7 ? 1 : 1.8);
    let dark = '', lite = '', shade = '';
    const topAt = u => {
      const s = u < 0 ? -1 : 1, a = Math.abs(u), pts = (s > 0 ? sidePts : sidePtsL).map(([x, y]) => [Math.abs(x), y]);
      if (type === 'xsg' && a <= G[0]) return G[1] - ridgeH * 0.25;
      if (cut[s]) return type === 'xsg' ? G[1] : top;
      if (type === 'xs' && a <= G[0]) return a <= rw ? top : yAt(pts, a) ?? top;
      if (type === 'yao' && a <= pts[pts.length - 1][0]) return top;
      if (a <= rw) return top;
      return yAt(pts, a);
    };
    for (let u = -ew * 0.98 + tile / 2; u < ew * 0.98; u += tile) {
      const y0 = E(u) + tb, y1 = topAt(u);
      if (y1 == null || y1 - y0 < tile * 0.6) continue;
      const yy1 = type === 'cj' || type === 'wd' || type === 'xs' ? y1 - ridgeH * (Math.abs(u) <= rw ? 0.1 : 0.35) : y1;
      dark += `M${P2(u - tile * 0.5, y0)}V${N(yy1)}`;
      lite += `M${P2(u - tile * 0.12, y0 + tile * 0.2)}V${N(yy1 - tile * 0.2)}`;
      shade += `M${P2(u + tile * 0.1, y0)}h${N(tile * 0.24)}V${N(yy1)}h${N(-tile * 0.24)}Z`;   // 筒瓦背光的一侧
    }
    if (lod >= 0.7) P.p(shade, 'c-shade'); P.p(dark, 'l3'); if (lod >= 0.7) P.p(lite, 'lw');
    // 檐口带：连檐 + 一排勾头（圆）与滴水（尖）
    P.p(`M${P2(...lowTip(-1))}${fromLo(-1)}` + eave.slice(1, -1).map(p => `L${P2(...p)}`).join('') + `L${P2(ew, eE(1))}${toLo(1)}` +
      `L${P2(...upTip(1))}${fromUp(1)}` + bandTop.slice(1, -1).reverse().map(p => `L${P2(...p)}`).join('') + `L${P2(-ew, eE(-1) + tb)}${toUp(-1)}Z`, `c-${trim}-d l2`);
    let heads = '', drips = '';
    for (let u = -ew * 0.98 + tile / 2; u < ew * 0.98; u += tile) {
      const y = E(u), r = Math.min(tb * 0.38, tile * 0.36);
      heads += `M${P2(u - r, y + tb * 0.52)}a${N(r)} ${N(r)} 0 1 0 ${N(r * 2)} 0a${N(r)} ${N(r)} 0 1 0 ${N(-r * 2)} 0Z`;
      const v = u + tile / 2; if (Math.abs(v) < ew * 0.97) { const yv = E(v); drips += `M${P2(v - tile * 0.3, yv + tb * 0.18)}L${P2(v, yv - tb * 0.38)}L${P2(v + tile * 0.3, yv + tb * 0.18)}Z`; }
    }
    if (lod >= 0.7) { P.p(drips, `c-${trim} l3`); P.p(heads, `c-${trim}-l l3`); }
    // 套兽：挂在翼角端头下
    for (const s of [-1, 1]) if (!cut[s] && lod >= 0.4) { const [x, y] = lowTip(s); taoshou(P, x - s * tb * 0.3, y - tb * 0.02, tb * 1.3, s, tone); }
    // 戗脊（垂脊）：顺着屋面轮廓的一条脊带，脊上仙人走兽
    const hb = ridgeH * 0.62;
    const ridgeBand = pts => {
      const up = pts.map(([x, y], i) => { const a = pts[Math.max(0, i - 1)], b = pts[Math.min(pts.length - 1, i + 1)]; let nx = -(b[1] - a[1]), ny = b[0] - a[0]; const l = Math.hypot(nx, ny) || 1; nx /= l; ny /= l; if (ny < 0) { nx = -nx; ny = -ny; } return [x + nx * hb, y + ny * hb]; });
      return poly([...pts, ...up.reverse()]);
    };
    for (const s of [-1, 1]) {
      if (cut[s]) continue;
      const pts = s > 0 ? sidePts : sidePtsL;
      P.p(ridgeBand(pts.slice(1)), `c-${tone} l2`);
      if (type === 'xs') { const g = s > 0 ? sR[3] : sL[3]; P.p(poly([[g[0], g[1]], [s * rw, top], [s * (rw + hb * 0.9), top], [g[0] + s * hb * 0.9, g[1] + hb * 0.3]]), `c-${tone} l2`); }
      const nb = lod >= 0.7 ? (o.beasts ?? 5) : 0, bs = hb * 1.5;
      const at = L => { let acc = 0; for (let i = 1; i < pts.length; i++) { const a = pts[i - 1], b = pts[i], d = Math.hypot(b[0] - a[0], b[1] - a[1]); if (acc + d >= L) { const f = (L - acc) / d; return [a[0] + (b[0] - a[0]) * f, a[1] + (b[1] - a[1]) * f]; } acc += d; } return pts[pts.length - 1]; };
      let L = bs * 0.55;
      for (let i = 0; i < nb; i++) { const [x, y] = at(L); beast(P, x, y + hb * 0.88, i ? bs : bs * 1.12, s, tone, i ? 'sit' : 'lead'); L += bs * (i ? 1.02 : 1.15); }
      if (nb > 0) { const [x2, y2] = at(L + bs * 0.35); beast(P, x2, y2 + hb * 0.8, bs * 1.6, s, tone, 'head'); }
    }
    // 歇山山面：博脊、山花（红底金绶带）、博风板（红底金钉）、排山勾滴、悬鱼、山尖
    if (type === 'xsg') {
      const off = (pts, d) => pts.map(([x, y], i) => { const a = pts[Math.max(0, i - 1)], b = pts[Math.min(pts.length - 1, i + 1)]; let nx = (b[1] - a[1]), ny = -(b[0] - a[0]); const l = Math.hypot(nx, ny) || 1; return [x + nx / l * d, y + ny / l * d]; });
      const c0 = [...bof.map(([x, y]) => [-x, y]), ...bof.slice().reverse().slice(1)];      // 左底角 → 山尖 → 右底角
      // 往里（往下）收：排山勾滴、博风板；收进去的线在山花底（博脊）处截平
      const clip = pts => { const out = []; for (let i = 0; i < pts.length; i++) { const p = pts[i], q = pts[i + 1]; if (p[1] >= G[1]) out.push(p); if (q && (p[1] - G[1]) * (q[1] - G[1]) < 0) { const f = (G[1] - p[1]) / (q[1] - p[1]); out.push([p[0] + (q[0] - p[0]) * f, G[1]]); } } return out; };
      const c1 = clip(off(c0, tb * 0.85)), c2 = clip(off(off(c0, tb * 0.85), rh * 0.075));
      const yg = G[1];
      P.p(poly([[-G[0] * 0.98, yg], ...c2.filter(([, y]) => y > yg), [G[0] * 0.98, yg]]), 'c-red l2');
      // 绶带：山花中线上一个结，两边飘带
      const hgt = top - yg, cx = 0, cy = yg + hgt * 0.38, s2 = hgt * 0.12;
      P.p(`M${P2(cx - s2, cy)}C${P2(cx - s2, cy + s2 * 0.9)} ${P2(cx + s2, cy + s2 * 0.9)} ${P2(cx + s2, cy)}C${P2(cx + s2, cy - s2 * 0.9)} ${P2(cx - s2, cy - s2 * 0.9)} ${P2(cx - s2, cy)}Z` +
        `M${P2(cx - s2, cy)}C${P2(cx - s2 * 2.6, cy - s2 * 0.2)} ${P2(cx - s2 * 3.4, cy + s2 * 0.9)} ${P2(cx - s2 * 4.6, cy - s2 * 0.3)}M${P2(cx + s2, cy)}C${P2(cx + s2 * 2.6, cy - s2 * 0.2)} ${P2(cx + s2 * 3.4, cy + s2 * 0.9)} ${P2(cx + s2 * 4.6, cy - s2 * 0.3)}`, 'lg2');
      P.p(poly([...c1, ...c2.slice().reverse()]), 'c-red-d l3');
      let studs = ''; for (let i = 2; i < Math.min(c1.length, c2.length) - 2; i += 2) { const [x, y] = c1[i], [x2, y2] = c2[Math.min(c2.length - 1, i)]; studs += `M${P2((x + x2) / 2, (y + y2) / 2)}h.01`; }
      P.p(studs, 'lg2');
      P.p(poly([...c0, ...c1.slice().reverse()]), `c-${trim} l2`);
      const c1f = off(c0, tb * 0.85);
      let ph = ''; for (let i = 1; i < c0.length - 1; i += 1) { const [x, y] = c0[i], [x2, y2] = c1f[i], r = tb * 0.32; if (y2 < G[1]) continue; ph += `M${P2((x + x2) / 2 - r, (y + y2) / 2)}a${N(r)} ${N(r)} 0 1 0 ${N(r * 2)} 0a${N(r)} ${N(r)} 0 1 0 ${N(-r * 2)} 0Z`; }
      P.p(ph, `c-${trim}-l l3`);
      // 悬鱼
      const sy = top - tb * 0.85 - rh * 0.075;
      P.p(`M${P2(-hgt * 0.035, sy)}L${P2(hgt * 0.035, sy)}L${P2(hgt * 0.03, sy - hgt * 0.12)}C${P2(hgt * 0.09, sy - hgt * 0.16)} ${P2(hgt * 0.05, sy - hgt * 0.24)} 0 ${N(sy - hgt * 0.22)}C${P2(-hgt * 0.05, sy - hgt * 0.24)} ${P2(-hgt * 0.09, sy - hgt * 0.16)} ${P2(-hgt * 0.03, sy - hgt * 0.12)}Z`, 'c-gold l3');
      // 博脊：山花底下一道横脊
      P.r(-G[0] - hb * 0.4, yg - ridgeH * 0.32, G[0] * 2 + hb * 0.8, ridgeH * 0.62, `c-${tone} l2`);
      if (o.apex === 'baoding') { P.g(sub(Q => baoding(Q, 0, ridgeH * 0.75)), ` transform="translate(0 ${N(top - tb * 0.3)})"`); }
      else if (o.apex !== 'none') { P.p(poly([[-ridgeH * 0.55, top - tb * 0.4], [ridgeH * 0.55, top - tb * 0.4], [ridgeH * 0.48, top + ridgeH * 1.9], [-ridgeH * 0.48, top + ridgeH * 1.9]]), `c-${tone} l2`); P.p(`M${P2(-ridgeH * 0.5, top + ridgeH * 1.2)}h${N(ridgeH)}`, 'l3'); }
    }
    // 正脊与鸱吻（截断的一端正脊通到截口，不安吻）
    if ((type === 'wd' || type === 'xs') && (rw > 0 || cut[-1] || cut[1])) {
      const xl = cut[-1] ? -ew : -rw - ridgeH * 0.1, xr = cut[1] ? ew : rw + ridgeH * 0.1;
      P.r(xl, top - ridgeH * 0.2, xr - xl, ridgeH * 1.2, `c-${tone} l2`);
      P.p(`M${P2(xl, top + ridgeH * 0.62)}H${N(xr)}`, 'l3');
      let pan = ''; for (let u = xl + ridgeH * 1.3; u < xr - ridgeH * 1.1; u += ridgeH * 1.6) pan += `M${P2(u, top + ridgeH * 0.12)}h${N(ridgeH * 1.1)}v${N(ridgeH * 0.38)}h${N(-ridgeH * 1.1)}Z`;
      P.p(pan, `c-${tone}-l l3`);
      if (o.wen !== false) { if (!cut[1]) chiwen(P, rw, top + ridgeH, ridgeH * 2.7, 1, tone); if (!cut[-1]) chiwen(P, -rw, top + ridgeH, ridgeH * 2.7, -1, tone); }
    }
    // 攒尖：多边形的面在正立面里看得到几道戗脊
    if (type === 'cj' && (o.sides || 4) > 4) {
      const n = o.sides, inner = [];
      for (let i = 1; i < n / 2; i++) { const a = Math.PI * i / (n / 2); const c = -Math.cos(a); if (Math.abs(c) < 0.98) inner.push(c); }
      let d = '';
      for (const c of inner) {
        const x0 = c * ew, y0 = E(x0) + tb;
        d += `M${P2(x0, y0)}C${P2(x0 * 0.6, y0 + rh * 0.04)} ${P2(x0 * 0.1, top - rh * 0.44)} 0 ${N(top)}`;
      }
      P.p(d, 'l2');
    }
    return { top, ridgeTop: top + ridgeH, E, tb, sd, G };
  }

  /* ── 柱：朱红圆柱，左亮右暗；古镜柱础 ─────────── */
  function columns(P, xs, y0, y1, w) {
    let d = '', base = '';
    for (const x of xs) {
      d += `M${P2(x - w / 2, y0)}h${N(w)}v${N(y1 - y0)}h${N(-w)}Z`;
      base += `M${P2(x - w * 0.82, y0)}h${N(w * 1.64)}c0 ${N(w * 0.18)} ${N(-w * 0.14)} ${N(w * 0.32)} ${N(-w * 0.32)} ${N(w * 0.34)}h${N(-w)}c${N(-w * 0.18)} ${N(-w * 0.02)} ${N(-w * 0.32)} ${N(-w * 0.16)} ${N(-w * 0.32)} ${N(-w * 0.34)}Z`;
    }
    P.p(d, 'g-col l2');
    P.p(base, 'c-marble l3');
  }

  /* ── 隔扇：一间里 n 扇。格心菱花、绦环板、裙板（金线框） ── */
  function geshan(P, x0, x1, y0, y1, n) {
    const w = (x1 - x0) / n, H = y1 - y0;
    let frame = '', lat = '', gold = '', lines = '';
    for (let i = 0; i < n; i++) {
      const a = x0 + i * w, m = w * 0.1;
      frame += `M${P2(a, y0)}h${N(w)}v${N(H)}h${N(-w)}Z`;
      const yq0 = y0 + H * 0.36, yq1 = y0 + H * 0.92;
      lat += `M${P2(a + m, yq0)}h${N(w - m * 2)}v${N(yq1 - yq0)}h${N(-w + m * 2)}Z`;
      const yr0 = y0 + H * 0.06, yr1 = y0 + H * 0.27;
      gold += `M${P2(a + m * 1.6, yr0 + H * 0.02)}h${N(w - m * 3.2)}v${N(yr1 - yr0 - H * 0.04)}h${N(-w + m * 3.2)}Z`;
      lines += `M${P2(a, yq0)}h${N(w)}M${P2(a, yr1)}h${N(w)}M${P2(a, yr0)}h${N(w)}M${P2(a, yq1)}h${N(w)}M${P2(a, y0 + H * 0.31)}h${N(w)}`;
    }
    if ((window.JH.lod ?? 1) < 0.7) { P.p(frame, 'c-red'); P.p(lat, 'p-lattice'); return; }
    P.p(frame, 'c-red l3'); P.p(lat, 'p-lattice l3'); P.p(gold, 'lg'); P.p(lines, 'l3');
  }

  /* ── 额枋彩画 ───────────────────────────────
     一间一段：两头箍头，中间枋心（和玺是尖头"<>"圭线光、枋心里金龙），两侧藻头青绿相间。
     flip 让上下两根枋青绿对调。 */
  function caihua(P, a, b, y0, h, kind = 'hexi', flip = false) {
    const L = b - a, m = (a + b) / 2, hl = L * 0.18, gt = h * 0.42;
    const cA = flip ? 'c-qing' : 'c-lu', cB = flip ? 'c-lu' : 'c-qing';
    P.r(a, y0, L, h, `${cB} l3`);
    if ((window.JH.lod ?? 1) < 0.7) { P.r(a + L * 0.3, y0 + h * 0.15, L * 0.4, h * 0.7, cA); return; }
    // 箍头
    P.p(`M${P2(a, y0)}h${N(h * 0.55)}v${N(h)}h${N(-h * 0.55)}ZM${P2(b - h * 0.55, y0)}h${N(h * 0.55)}v${N(h)}h${N(-h * 0.55)}Z`, `${cA} l3`);
    // 枋心
    const fx0 = m - hl, fx1 = m + hl;
    P.p(`M${P2(fx0, y0 + h * 0.5)}L${P2(fx0 + gt, y0 + h * 0.12)}H${N(fx1 - gt)}L${P2(fx1, y0 + h * 0.5)}L${P2(fx1 - gt, y0 + h * 0.88)}H${N(fx0 + gt)}Z`, `${cA} l3`);
    P.p(`M${P2(fx0, y0 + h * 0.5)}L${P2(fx0 + gt, y0 + h * 0.12)}H${N(fx1 - gt)}L${P2(fx1, y0 + h * 0.5)}L${P2(fx1 - gt, y0 + h * 0.88)}H${N(fx0 + gt)}Z`, 'lg');
    if (kind === 'hexi') {
      // 枋心金龙（简笔：一条游动的金线和龙珠）
      P.p(`M${P2(fx0 + gt * 1.4, y0 + h * 0.4)}C${P2(m - hl * 0.3, y0 + h * 0.78)} ${P2(m - hl * 0.1, y0 + h * 0.2)} ${P2(m + hl * 0.2, y0 + h * 0.55)}S${P2(fx1 - gt * 1.6, y0 + h * 0.62)} ${P2(fx1 - gt * 1.2, y0 + h * 0.4)}`, 'lg2');
      P.c(m + hl * 0.1, y0 + h * 0.3, h * 0.08, 'c-gold');
      // 藻头：Σ 形圭线光
      for (const s of [-1, 1]) {
        const x0 = s < 0 ? a + h * 0.55 : b - h * 0.55, x1 = s < 0 ? fx0 : fx1, w = x1 - x0, k = w * 0.16;
        P.p(`M${P2(x0 + k, y0 + h * 0.08)}L${P2(x0 + k * 2.2, y0 + h * 0.5)}L${P2(x0 + k, y0 + h * 0.92)}M${P2(x1 - k, y0 + h * 0.08)}L${P2(x1 - k * 2.2, y0 + h * 0.5)}L${P2(x1 - k, y0 + h * 0.92)}M${P2(x0 + k * 2.2, y0 + h * 0.5)}H${N(x1 - k * 2.2)}`, 'lg');
      }
    } else {
      // 旋子：藻头一整两破旋花
      for (const s of [-1, 1]) {
        const cx = s < 0 ? (a + h * 0.55 + fx0) / 2 : (b - h * 0.55 + fx1) / 2, r = h * 0.34;
        P.c(cx, y0 + h * 0.5, r, `${cA} l3`); P.c(cx, y0 + h * 0.5, r * 0.45, 'c-paper l3');
        P.p(`M${P2(cx - r * 1.9, y0)}a${N(r)} ${N(r)} 0 0 1 0 ${N(h)}M${P2(cx + r * 1.9, y0)}a${N(r)} ${N(r)} 0 0 0 0 ${N(h)}`, 'l3');
      }
    }
    P.p(`M${P2(a, y0)}h${N(L)}M${P2(a, y0 + h)}h${N(L)}`, 'lg');
  }

  /* ── 额枋层：小额枋、由额垫板、大额枋、平板枋 ───── */
  function beams(P, xs, y0, H, kind = 'hexi') {
    const hs = H * 0.3, hp = H * 0.16, hb = H * 0.38, hpb = H - hs - hp - hb;
    const a = xs[0], b = xs[xs.length - 1], cw = H * 0.4;
    for (let i = 0; i < xs.length - 1; i++) caihua(P, xs[i] + cw * 0.3, xs[i + 1] - cw * 0.3, y0, hs, kind, i % 2 === 1);
    P.r(a - cw * 0.5, y0 + hs, b - a + cw, hp, 'c-red l3');
    const lod = window.JH.lod ?? 1;
    let dots = ''; if (lod >= 0.7) for (let x = a; x <= b; x += hp * 2.4) dots += `M${P2(x, y0 + hs + hp * 0.5)}h.01`;
    P.p(dots, 'lg2');
    for (let i = 0; i < xs.length - 1; i++) caihua(P, xs[i] + cw * 0.3, xs[i + 1] - cw * 0.3, y0 + hs + hp, hb, kind, i % 2 === 0);
    // 平板枋：青底金色降魔云
    P.r(a - cw * 0.9, y0 + H - hpb, b - a + cw * 1.8, hpb, 'c-qing l3');
    let cl = ''; if (lod >= 0.7) for (let x = a; x < b; x += hpb * 2.2) cl += `M${P2(x, y0 + H - hpb * 0.3)}q${N(hpb * 0.5)} ${N(-hpb * 0.6)} ${N(hpb)} 0`;
    P.p(cl, 'lg');
  }

  /* ── 斗拱：柱头科在柱上，平身科在两柱之间；攒与攒之间垫拱板 ── */
  function dougong(P, xs, y0, H, per = 4, ext = 0) {
    const a = xs[0] - ext, b = xs[xs.length - 1] + ext;
    P.r(a, y0, b - a, H * 0.82, 'c-red l3');
    const pos = [];
    for (let i = 0; i < xs.length; i++) { pos.push(xs[i]); if (i < xs.length - 1) for (let k = 1; k <= per; k++) pos.push(xs[i] + (xs[i + 1] - xs[i]) * k / (per + 1)); }
    if ((window.JH.lod ?? 1) < 0.7) { let blk = ''; for (const x of pos) { const w = Math.min(...xs.slice(1).map((v, i) => v - xs[i])) / (per + 1) * 0.42; blk += `M${P2(x - w, y0 + H * 0.15)}h${N(w * 2)}v${N(H * 0.6)}h${N(-w * 2)}Z`; } P.p(blk, 'c-lu'); P.r(a - 2, y0 + H * 0.78, b - a + 4, H * 0.22, 'c-lu l3'); return; }
    const minGap = Math.min(...xs.slice(1).map((x, i) => x - xs[i])) / (per + 1), d = minGap / 9.6;
    let dou = '', arm1 = '', arm2 = '', sheng = '', ang = '', gold = '';
    const h1 = H * 0.2, h2 = H * 0.2, h3 = H * 0.2;
    for (const x of pos) {
      // 坐斗
      dou += `M${P2(x - d * 1.6, y0)}h${N(d * 3.2)}l${N(d * 0.3)} ${N(h1)}h${N(-d * 3.8)}Z`;
      // 瓜拱（两端卷杀）
      const w1 = d * 3.3, w2 = d * 4.6, yA = y0 + h1, yB = yA + h2;
      arm1 += `M${P2(x - w1, yA + h2 * 0.35)}q0 ${N(-h2 * 0.35)} ${N(h2 * 0.35)} ${N(-h2 * 0.35)}h${N(w1 * 2 - h2 * 0.7)}q${N(h2 * 0.35)} 0 ${N(h2 * 0.35)} ${N(h2 * 0.35)}v${N(h2 * 0.65)}h${N(-w1 * 2)}Z`;
      arm2 += `M${P2(x - w2, yB + h3 * 0.35)}q0 ${N(-h3 * 0.35)} ${N(h3 * 0.35)} ${N(-h3 * 0.35)}h${N(w2 * 2 - h3 * 0.7)}q${N(h3 * 0.35)} 0 ${N(h3 * 0.35)} ${N(h3 * 0.35)}v${N(h3 * 0.65)}h${N(-w2 * 2)}Z`;
      for (const u of [-w1 + d * 0.7, 0, w1 - d * 0.7]) sheng += `M${P2(x + u - d * 0.75, yA + h2)}h${N(d * 1.5)}v${N(h2 * 0.3)}h${N(-d * 1.5)}Z`;
      // 昂嘴：从正面看是一个朝下的尖
      ang += `M${P2(x - d * 0.9, yB)}L${P2(x, yB - h2 * 0.9)}L${P2(x + d * 0.9, yB)}Z`;
      gold += `M${P2(x - w1 + d * 0.4, yA + h2 * 0.5)}h${N(w1 * 2 - d * 0.8)}M${P2(x - w2 + d * 0.4, yB + h3 * 0.5)}h${N(w2 * 2 - d * 0.8)}`;
    }
    P.p(dou, 'c-qing l3'); P.p(arm1, 'c-lu l3'); P.p(ang, 'c-qing l3'); P.p(sheng, 'c-qing l3'); P.p(arm2, 'c-qing l3'); P.p(gold, 'lg');
    // 挑檐枋
    P.r(a - d * 2, y0 + h1 + h2 + h3, b - a + d * 4, H - h1 - h2 - h3, 'c-lu l3');
    P.p(`M${P2(a - d * 2, y0 + H - (H - h1 - h2 - h3) * 0.5)}H${N(b + d * 2)}`, 'lg');
  }

  /* ── 须弥座：从下到上圭角、下枋、下枭（莲瓣）、束腰、上枭（莲瓣）、上枋；四角螭首 ── */
  function xumizuo(P, y0, h, hw, o = {}) {
    const k = [0, 0.14, 0.28, 0.38, 0.6, 0.72, 1].map(f => y0 + h * f);
    const ins = h * 0.07;
    // 右侧轮廓（向内收的束腰、弧形的枭）
    const R = [[hw + ins * 0.4, k[0]], [hw + ins * 0.4, k[1]], [hw, k[1]], [hw, k[2]]];
    const curve1 = samp(t => [hw - ins * 1.4 * Math.sin(t * Math.PI / 2), k[2] + (k[3] - k[2]) * t], 8).slice(1);
    const mid = [[hw - ins * 1.4, k[4]]];
    const curve2 = samp(t => [hw - ins * 1.4 * Math.cos(t * Math.PI / 2), k[4] + (k[5] - k[4]) * t], 8).slice(1);
    const top = [[hw + ins * 0.2, k[5]], [hw + ins * 0.2, k[6]]];
    const right = [...R, ...curve1, ...mid, ...curve2, ...top];
    const left = right.map(([x, y]) => [-x, y]).reverse();
    P.p(poly([...right, ...left]), 'c-marble l1');
    // 束腰压暗，枭面淡一档
    P.p(poly([[hw - ins * 1.4, k[3]], [hw - ins * 1.4, k[4]], [-hw + ins * 1.4, k[4]], [-hw + ins * 1.4, k[3]]]), 'c-marble-d l3');
    P.p(`M${P2(-hw, k[1])}H${N(hw)}M${P2(-hw, k[2])}H${N(hw)}M${P2(-hw - ins * 0.2, k[5])}H${N(hw + ins * 0.2)}M${P2(-hw - ins * 0.4, k[1])}`, 'l3');
    // 莲瓣：上枭、下枭各一排
    const petals = (ya, yb, up) => { let d = ''; const w = (yb - ya) * 1.5; for (let x = -hw + ins * 1.6; x < hw - ins * 1.6 - w; x += w) d += up ? `M${P2(x, ya)}Q${P2(x, yb)} ${P2(x + w / 2, yb)}Q${P2(x + w, yb)} ${P2(x + w, ya)}` : `M${P2(x, yb)}Q${P2(x, ya)} ${P2(x + w / 2, ya)}Q${P2(x + w, ya)} ${P2(x + w, yb)}`; return d; };
    if ((window.JH.lod ?? 1) < 0.7) return k[6];
    P.p(petals(k[4], k[5], true) + petals(k[2], k[3], false), 'l3');
    // 束腰的椀花结带
    let knot = ''; const step = (hw * 2) / Math.max(2, Math.round(hw / (h * 0.9)));
    for (let x = -hw + step; x < hw - step * 0.5; x += step) knot += `M${P2(x - h * 0.08, k[3] + h * 0.06)}h${N(h * 0.16)}v${N(h * 0.1)}h${N(-h * 0.16)}Z`;
    P.p(knot, 'c-marble l3');
    // 圭角：角上卷云脚
    for (const s of [-1, 1]) P.p(`M${P2(s * (hw + ins * 0.4), k[0])}c${N(-s * h * 0.06)} ${N(h * 0.02)} ${N(-s * h * 0.1)} ${N(h * 0.08)} ${N(-s * h * 0.06)} ${N(h * 0.12)}`, 'l3');
    // 螭首
    if (o.chishou) for (const s of [-1, 1]) {
      const x = s * (hw + ins * 0.2), y = k[5] + h * 0.06;
      P.p(`M${P2(x, y)}C${P2(x + s * h * 0.12, y + h * 0.05)} ${P2(x + s * h * 0.26, y + h * 0.04)} ${P2(x + s * h * 0.34, y - h * 0.01)}L${P2(x + s * h * 0.3, y - h * 0.05)}L${P2(x + s * h * 0.36, y - h * 0.08)}C${P2(x + s * h * 0.24, y - h * 0.1)} ${P2(x + s * h * 0.12, y - h * 0.08)} ${P2(x, y - h * 0.09)}Z`, 'c-marble l3');
      P.c(x + s * h * 0.22, y + h * 0.005, h * 0.012, 'c-ink');
    }
    return k[6];
  }

  /* ── 汉白玉栏杆：地栿、望柱（柱头）、栏板（寻杖、净瓶、华板） ── */
  function balustrade(P, a, b, y0, h, sp, head = 'yun') {
    const n = Math.max(1, Math.round((b - a) / sp)), s = (b - a) / n, pw = h * 0.13;
    P.r(a - pw, y0, b - a + pw * 2, h * 0.08, 'c-marble l3');
    let panel = '', rail = '', vase = '', inset = '', posts = '', heads = '';
    for (let i = 0; i < n; i++) {
      const x0 = a + i * s + pw * 0.5, x1 = a + (i + 1) * s - pw * 0.5;
      panel += `M${P2(x0, y0 + h * 0.08)}H${N(x1)}V${N(y0 + h * 0.4)}H${N(x0)}Z`;
      inset += `M${P2(x0 + s * 0.08, y0 + h * 0.13)}H${N(x1 - s * 0.08)}V${N(y0 + h * 0.35)}H${N(x0 + s * 0.08)}Z`;
      rail += `M${P2(x0, y0 + h * 0.6)}H${N(x1)}V${N(y0 + h * 0.68)}H${N(x0)}Z`;
      for (const f of [0.3, 0.7]) {
        const x = x0 + (x1 - x0) * f, vw = h * 0.07;
        vase += `M${P2(x - vw * 0.6, y0 + h * 0.4)}C${P2(x - vw * 1.4, y0 + h * 0.46)} ${P2(x - vw * 0.4, y0 + h * 0.52)} ${P2(x - vw * 0.5, y0 + h * 0.56)}L${P2(x - vw * 1.2, y0 + h * 0.6)}H${N(x + vw * 1.2)}L${P2(x + vw * 0.5, y0 + h * 0.56)}C${P2(x + vw * 0.4, y0 + h * 0.52)} ${P2(x + vw * 1.4, y0 + h * 0.46)} ${P2(x + vw * 0.6, y0 + h * 0.4)}Z`;
      }
    }
    for (let i = 0; i <= n; i++) {
      const x = a + i * s;
      posts += `M${P2(x - pw / 2, y0 + h * 0.08)}h${N(pw)}v${N(h * 0.74)}h${N(-pw)}Z`;
      const yh = y0 + h * 0.82;
      if (head === 'lian') heads += `M${P2(x - pw * 0.55, yh)}C${P2(x - pw * 0.7, yh + h * 0.12)} ${P2(x - pw * 0.2, yh + h * 0.2)} ${P2(x, yh + h * 0.2)}C${P2(x + pw * 0.2, yh + h * 0.2)} ${P2(x + pw * 0.7, yh + h * 0.12)} ${P2(x + pw * 0.55, yh)}Z`;
      else heads += `M${P2(x - pw * 0.55, yh)}h${N(pw * 1.1)}v${N(h * 0.05)}c0 ${N(h * 0.08)} ${N(-pw * 0.25)} ${N(h * 0.13)} ${N(-pw * 0.55)} ${N(h * 0.13)}s${N(-pw * 0.55)} ${N(-h * 0.05)} ${N(-pw * 0.55)} ${N(-h * 0.13)}Z`;
    }
    if ((window.JH.lod ?? 1) < 0.7) { P.p(panel, 'c-marble l3'); P.p(rail, 'c-marble'); P.p(posts, 'c-marble l3'); return; }
    P.p(panel, 'c-marble l3'); P.p(inset, 'c-marble-d l3'); P.p(vase, 'c-marble l3'); P.p(rail, 'c-marble l3');
    P.p(posts, 'c-marble l3'); P.p(heads, 'c-marble l3');
  }

  /* ── 台阶：两侧垂带，中间御路（云纹） ─────────── */
  function stairs(P, y0, y1, w, ramp = 0, n = 0) {
    const H = y1 - y0, steps = n || Math.max(4, Math.round(H / 1.6));
    P.p(poly([[-w, y0], [w, y0], [w, y1], [-w, y1]]), 'c-marble l2');
    let d = ''; for (let i = 1; i < steps; i++) { const y = y0 + H * i / steps; d += `M${P2(-w, y)}H${N(w)}`; }
    P.p(d, 'l3');
    const bw = w * 0.12;
    P.p(`M${P2(-w, y0)}h${N(bw)}v${N(H)}h${N(-bw)}ZM${P2(w - bw, y0)}h${N(bw)}v${N(H)}h${N(-bw)}Z`, 'c-marble-d l3');
    if (ramp) {
      P.p(poly([[-ramp, y0], [ramp, y0], [ramp, y1], [-ramp, y1]]), 'c-marble l3');
      let c = ''; for (let i = 0; i < 4; i++) { const y = y0 + H * (0.14 + i * 0.22); c += `M${P2(-ramp * 0.6, y)}c${N(ramp * 0.25)} ${N(H * 0.06)} ${N(ramp * 0.5)} ${N(-H * 0.04)} ${N(ramp * 0.6)} 0s${N(ramp * 0.35)} ${N(H * 0.05)} ${N(ramp * 0.6)} 0`; }
      P.p(c, 'l3');
    }
  }

  /* ── 鎏金宝顶：须弥座、宝瓶、宝珠 ─────────────── */
  function baoding(P, y0, s) {
    P.p(poly([[-s * 1.5, y0], [s * 1.5, y0], [s * 1.2, y0 + s * 0.5], [s * 1.35, y0 + s * 0.85], [-s * 1.35, y0 + s * 0.85], [-s * 1.2, y0 + s * 0.5]]), 'g-gold l2');
    P.p(`M${P2(-s * 0.6, y0 + s * 0.85)}C${P2(-s * 1.5, y0 + s * 1.6)} ${P2(-s * 1.3, y0 + s * 3.2)} 0 ${N(y0 + s * 3.4)}C${P2(s * 1.3, y0 + s * 3.2)} ${P2(s * 1.5, y0 + s * 1.6)} ${P2(s * 0.6, y0 + s * 0.85)}Z`, 'g-gold l2');
    P.p(`M${P2(-s * 0.45, y0 + s * 3.3)}h${N(s * 0.9)}v${N(s * 0.35)}h${N(-s * 0.9)}Z`, 'g-gold l3');
    P.c(0, y0 + s * 4.25, s * 0.62, 'g-gold l2');
    P.p(`M${P2(-s * 0.7, y0 + s * 1.6)}C${P2(-s * 0.4, y0 + s * 2.6)} ${P2(-s * 0.1, y0 + s * 2.9)} ${P2(s * 0.1, y0 + s * 3)}`, 'lw');
  }

  window.JH.ARCH = { defs, roof, chiwen, beast, taoshou, columns, geshan, caihua, beams, dougong, xumizuo, balustrade, stairs, baoding };
})();
