/* 首屏「京华胜景」长卷：展卷、游卷（滚动横移 + 分层视差，每处地标前放慢驻足）、题跋（逐行显现、钤印）、
   点睛（点地标出册页卡）、四时、晟帆随行帆船、卷轴导览、倒影烘焙。样张见 design/v2/。 */
(function () {
  'use strict';
  const $ = s => document.querySelector(s), $$ = s => [...document.querySelectorAll(s)];
  const reduce = matchMedia('(prefers-reduced-motion: reduce)').matches;
  if (reduce) document.documentElement.classList.add('reduce');
  const clamp = (v, a, b) => Math.min(b, Math.max(a, v));
  const easeS = t => -(Math.cos(Math.PI * t) - 1) / 2;

  /* ── 节气、干支、季节 ───────────────────────────── */
  const JQ = [[1, 6, '小寒'], [1, 20, '大寒'], [2, 4, '立春'], [2, 19, '雨水'], [3, 6, '惊蛰'], [3, 21, '春分'], [4, 5, '清明'], [4, 20, '谷雨'], [5, 6, '立夏'], [5, 21, '小满'], [6, 6, '芒种'], [6, 21, '夏至'],
    [7, 7, '小暑'], [7, 23, '大暑'], [8, 8, '立秋'], [8, 23, '处暑'], [9, 8, '白露'], [9, 23, '秋分'], [10, 8, '寒露'], [10, 23, '霜降'], [11, 7, '立冬'], [11, 22, '小雪'], [12, 7, '大雪'], [12, 22, '冬至']];
  const now = new Date();
  const jqIndex = d => { const m = d.getMonth() + 1, day = d.getDate(); let k = 23; JQ.forEach(([mm, dd], i) => { if (m > mm || (m === mm && day >= dd)) k = i; }); return k; };
  const jq = JQ[jqIndex(now)][2];
  const ganzhi = d => { const y = d.getFullYear() - (d.getMonth() === 0 || (d.getMonth() === 1 && d.getDate() < 4) ? 1 : 0); return '甲乙丙丁戊己庚辛壬癸'[(y - 4) % 10] + '子丑寅卯辰巳午未申酉戌亥'[(y - 4) % 12]; };
  const seasonOf = d => { const i = jqIndex(d); return i >= 2 && i < 8 ? 'spring' : i >= 8 && i < 14 ? 'summer' : i >= 14 && i < 20 ? 'autumn' : 'winter'; };

  /* ── 画面：四层长卷 + 独立小块 ─────────────────── */
  $('#defs').innerHTML = ART2.defs();
  const S = SCENE.compose();
  const VB = `-400 0 ${S.W + 800} ${S.H}`;
  const svg = (inner, vb = VB) => `<svg class="a2" viewBox="${vb}" preserveAspectRatio="xMinYMin meet" aria-hidden="true">${inner}</svg>`;
  const LAY = { sky: $('#laySky'), far: $('#layFar'), mid: $('#layMid'), main: $('#layMain') };
  LAY.sky.innerHTML = svg(S.sky);
  LAY.far.innerHTML = svg(S.far);
  LAY.mid.innerHTML = svg(S.mid);
  LAY.main.insertAdjacentHTML('afterbegin', svg(S.main));
  // 各层的小块紧跟在该层之后；近景压在最上（绢纹之下）
  const after = { sky: LAY.sky, far: LAY.far, mid: LAY.mid, main: LAY.main }, silkEl = $('#stage .silk');
  const sprites = S.sprites.map(o => {
    const el = document.createElement('div');
    el.className = 'spr' + (o.only ? ' only-' + o.only : '');
    el.innerHTML = svg(o.svg, `${o.x0} ${o.y0} ${o.x1 - o.x0} ${o.y1 - o.y0}`);
    if (o.layer === 'fore') silkEl.before(el); else { after[o.layer].after(el); after[o.layer] = el; }
    return { ...o, el, vis: true };
  });
  const layers = [['sky', 0.55], ['far', 0.72], ['mid', 0.86], ['main', 1]].map(([k, d]) => ({ el: LAY[k], d }));

  /* ── 题跋 ─────────────────────────────────────── */
  // a：地标中心（走到附近时浮现）；x、y：题跋位置（画面坐标）
  const CAPS = [
    // side：手机上题在画面哪一侧（避开主体和鹤群）
    { a: 300, x: 470, y: 180, side: 'right', t: '万里长城', s: ['八达岭', '明弘治十八年建关城'], seal: '长城' },
    { a: 1150, x: 1390, y: 150, side: 'right', t: '颐和园', s: ['佛香阁 · 十七孔桥', '清乾隆十五年始建'], seal: '颐和' },
    { a: 2300, x: 2150, y: 150, side: 'left', t: '紫禁城', s: ['角楼 · 宫墙', '明永乐十八年建成'], seal: '故宫' },
    { a: 3275, x: 3470, y: 150, side: 'right', t: '天坛', s: ['祈年殿', '明永乐十八年始建'], seal: '天坛' },
    { a: 3930, x: 3690, y: 170, side: 'right', t: '国家博物馆', s: ['复兴之路', '基本陈列所在'], seal: '国博' },
    { a: 4360, x: 4290, y: 150, side: 'right', t: '京华胜景', s: ['晟帆国旅敬制', `${ganzhi(now)} · ${jq}`], seal: '晟帆', colophon: true },
  ];
  const caps = $('#caps');
  caps.innerHTML = CAPS.map(c => `<div class="cap${c.colophon ? ' colophon' : ''}"><b>${c.t}</b>${c.s.map(x => `<span>${x}</span>`).join('')}<i class="seal${c.colophon ? ' big' : ''}">${[...c.seal].map(ch => `<span>${ch}</span>`).join('')}</i></div>`).join('');
  const capEls = $$('.cap').map((el, i) => ({ el, ...CAPS[i] }));

  /* ── 游卷：驻足点与滚动映射 ──────────────────── */
  const sec = $('.juan'), stick = $('.juan-stick'), stage = $('#stage'), title = $('#title'), hint = $('#hint'), prog = $('#prog');
  const STOPS = [{ n: '卷首' }, { a: 300, n: '长城' }, { a: 1150, n: '颐和园' }, { a: 2300, n: '故宫' }, { a: 3275, n: '天坛' }, { a: 3930, n: '国博' }];
  let scale = 1, Vw = 1, pan = 0, top = 0, W = 1, Hh = 1, narrow = false, keys = [], total = 0, stopAt = [];
  function buildKeys() {
    const xs = [];
    const push = (x, dwell, name) => {
      x = clamp(x, 0, pan);
      const last = xs[xs.length - 1];
      if (last && Math.abs(x - last.x) < Vw * 0.12) { last.dwell = Math.max(last.dwell, dwell); last.names.push(name); return; }
      xs.push({ x, dwell, names: [name] });
    };
    STOPS.forEach((s, i) => push(i === 0 ? 0 : s.a - Vw / 2, i === 0 ? 0.28 : 0.6, s.n));
    push(pan, 0.5, '卷尾');
    keys = []; total = 0; stopAt = [];
    const dr = Vw * 0.022;
    xs.forEach((s, i) => {
      const len = s.dwell * Hh, xa = clamp(s.x - (i ? dr : 0), 0, pan), xb = clamp(s.x + (i < xs.length - 1 ? dr : 0), 0, pan);
      stopAt.push({ names: s.names, px: total + len * 0.45, x: s.x });
      keys.push({ from: xa, to: xb, len, travel: false }); total += len;
      if (i < xs.length - 1) {
        const nx = clamp(xs[i + 1].x - dr, 0, pan), tl = Math.max(0.32 * Hh, Math.abs(nx - xb) * scale * 0.95);
        keys.push({ from: xb, to: nx, len: tl, travel: true }); total += tl;
      }
    });
  }
  function xAt(sp) {
    for (const k of keys) { if (sp <= k.len) { const t = k.len ? sp / k.len : 1; return k.from + (k.to - k.from) * (k.travel ? easeS(t) : t); } sp -= k.len; }
    return pan;
  }

  function layout() {
    Hh = stick.clientHeight; W = stick.clientWidth; narrow = W < 700;
    const zoom = narrow ? 1.2 : 1;               // 手机竖屏：画卷略放大、贴底
    scale = Hh * zoom / S.H;
    top = Hh - S.H * scale;
    Vw = W / scale; pan = Math.max(0, S.W - Vw);
    const fullW = (S.W + 800) * scale;
    layers.forEach(l => { l.el.style.width = fullW + 'px'; l.el.style.height = (S.H * scale) + 'px'; l.el.style.top = top + 'px'; });
    sprites.forEach(o => {
      o.on = !o.only || (o.only === 'narrow') === narrow;
      o.el.style.display = o.on ? '' : 'none';
      o.w = (o.x1 - o.x0) * scale;
      o.el.style.width = o.w + 'px'; o.el.style.height = ((o.y1 - o.y0) * scale) + 'px'; o.el.style.top = (top + o.y0 * scale) + 'px';
    });
    capEls.forEach(c => {
      let x = c.x, yPx = Math.max(76 - top, c.y * scale);
      if (narrow) {
        // 手机：题跋贴在该处画面的左上或右上角
        const vx0 = clamp(c.a - Vw / 2, 0, pan), cw = c.el.offsetWidth / scale, m = 16 / scale;
        x = c.side === 'left' ? vx0 + m : vx0 + Vw - m - cw;
        yPx = 84 - top;
      }
      c.el.style.left = ((x + 400) * scale).toFixed(1) + 'px'; c.el.style.top = yPx.toFixed(1) + 'px';
    });
    buildKeys();
    sec.style.height = Math.round(Hh + total) + 'px';
    layoutBoat();
    buildNav();
    update(true);
  }

  let lastX = 0, lastSp = -1;
  function update(force) {
    const sp = clamp(scrollY - sec.offsetTop, 0, total), p = total ? sp / total : 0;
    if (!force && sp === lastSp) return;      // 已滚出首屏（或没动）就不再算
    lastSp = sp;
    const xMain = xAt(sp), cx = xMain + Vw / 2, center = pan / 2;
    layers.forEach(l => { const x = center + l.d * (xMain - center); l.el.style.transform = `translate3d(${(-(x + 400) * scale).toFixed(1)}px,0,0)`; });
    for (const o of sprites) {
      if (!o.on) continue;
      let X = o.x0, A;
      if (o.anchor === 'start') { X = o.off; A = Vw / 2; }
      else if (o.anchor === 'end') { X = pan + Vw + o.off; A = pan + Vw / 2; }
      else if (o.anchor === 'layer') A = (pan + Vw) / 2;
      else A = o.anchor;
      const tx = (X - xMain + (1 - o.d) * (cx - A)) * scale;
      const vis = tx < W + 60 && tx + o.w > -60;
      if (vis !== o.vis) { o.vis = vis; o.el.style.visibility = vis ? '' : 'hidden'; }
      if (vis) o.el.style.transform = `translate3d(${tx.toFixed(1)}px,0,0)`;
    }
    // 只题离画面中心最近的一处；到卷尾再落款
    const atEnd = sp > total - Hh * 0.35;
    let near = null, nd = Infinity;
    capEls.forEach(c => { if (c.colophon) return; const d = Math.abs(c.a - cx); if (d < nd) { nd = d; near = c; } });
    // 卷尾：若最后一处（国博）还在画面里，就题它再落款（电脑上二者同框）；手机上国博已单独题过，只落款
    if (atEnd) { const last = capEls.filter(c => !c.colophon).pop(); if (Math.abs(last.a - cx) < Vw / 2) { near = last; nd = 0; } else near = null; }
    capEls.forEach(c => c.el.classList.toggle('on', sp > Hh * 0.1 && (c.colophon ? atEnd : c === near && nd < Math.max(Vw * 0.36, 170))));
    title.classList.toggle('hide', sp > Hh * 0.1);
    hint.classList.toggle('hide', sp > Hh * 0.04);
    prog.style.width = (p * 100).toFixed(2) + '%';
    navActive(sp);
    moveBoat(sp, xMain - lastX, force);
    lastX = xMain;
  }
  let ticking = false;
  addEventListener('scroll', () => { if (ticking) return; ticking = true; requestAnimationFrame(() => { update(); ticking = false; }); }, { passive: true });
  // 首屏不在视野里时，暂停里面所有动画（CSS 和 SVG 自带的），省电
  if ('IntersectionObserver' in window) new IntersectionObserver(([e]) => {
    stick.classList.toggle('paused', !e.isIntersecting);
    stick.querySelectorAll('svg').forEach(sv => { try { e.isIntersecting ? sv.unpauseAnimations() : sv.pauseAnimations(); } catch (_) {} });
  }).observe(stick);
  let rz = 0;
  addEventListener('resize', () => { cancelAnimationFrame(rz); rz = requestAnimationFrame(layout); });

  /* ── 卷轴导览：底部一条细线，标出各处地标，点哪去哪 ── */
  const nav = $('#juanNav');
  function buildNav() {
    const items = stopAt.flatMap(s => s.names.filter(n => n !== '卷首' && n !== '卷尾').slice(0, 1).map(n => ({ n, px: s.px })));
    nav.querySelectorAll('button').forEach(b => b.remove());
    items.forEach(it => {
      const b = document.createElement('button'); b.type = 'button'; b.textContent = it.n; b.dataset.px = it.px;
      b.style.left = (it.px / total * 100).toFixed(2) + '%';
      b.addEventListener('click', () => scrollTo({ top: sec.offsetTop + it.px, behavior: reduce ? 'auto' : 'smooth' }));
      nav.appendChild(b);
    });
  }
  function navActive(sp) {
    let best = null, bd = Infinity;
    nav.querySelectorAll('button').forEach(b => { const d = Math.abs(+b.dataset.px - sp); if (d < bd) { bd = d; best = b; } });
    nav.querySelectorAll('button').forEach(b => b.classList.toggle('on', b === best && bd < Hh * 0.6));
  }

  /* ── 晟帆随行：一叶帆船跟着往前走，回滚时掉头 ──── */
  const boat = $('#guide');
  let dir = 1, moveT = 0;
  function layoutBoat() {
    const bw = 150 * scale * (narrow ? 0.84 : 0.92), bh = bw;
    boat.style.width = bw + 'px'; boat.style.height = bh + 'px';
    boat.style.left = (W * (narrow ? 0.3 : 0.6) - bw / 2) + 'px';
    boat.style.top = (top + (S.WY + 150) * scale - bh * 0.86) + 'px';
  }
  function moveBoat(sp, dx, force) {
    boat.classList.toggle('on', sp > Hh * 0.12 && sp < total - Hh * 0.05 || (sp >= total - Hh * 0.05 && total > 0));
    if (force) return;
    if (Math.abs(dx) > 0.3) {
      const d = dx > 0 ? 1 : -1;
      if (d !== dir) { dir = d; boat.classList.toggle('back', d < 0); }
      boat.classList.add('moving'); clearTimeout(moveT); moveT = setTimeout(() => boat.classList.remove('moving'), 280);
    }
  }

  /* ── 四时：一方印，点一下换一季 ───────────────── */
  const SEASONS = ['spring', 'summer', 'autumn', 'winter'], SN = { spring: '春', summer: '夏', autumn: '秋', winter: '冬' };
  const sBtn = $('#seasonBtn');
  const setSeason = s => {
    stage.dataset.season = s; stick.dataset.season = s;
    sBtn.querySelector('b').textContent = SN[s];
    sBtn.setAttribute('aria-label', `四时：当前${SN[s]}，点按换季`);
    sBtn.classList.remove('stamp'); void sBtn.offsetWidth; sBtn.classList.add('stamp');
    if (typeof bakeReflections === 'function' && reflBaked) bakeReflections();
  };
  let reflBaked = false;
  setSeason(new URLSearchParams(location.search).get('season') || seasonOf(now));
  sBtn.addEventListener('click', () => setSeason(SEASONS[(SEASONS.indexOf(stage.dataset.season) + 1) % 4]));
  (function snow() {
    let h = ''; for (let i = 0; i < 70; i++) { const s = 2 + Math.random() * 3.5; h += `<i style="left:${(Math.random() * 100).toFixed(1)}%;width:${s.toFixed(1)}px;height:${s.toFixed(1)}px;animation-duration:${(7 + Math.random() * 9).toFixed(1)}s;animation-delay:-${(Math.random() * 12).toFixed(1)}s;--dx:${((Math.random() - 0.5) * 80).toFixed(0)}px;opacity:${(0.5 + Math.random() * 0.45).toFixed(2)}"></i>`; }
    $('#snow').innerHTML = h;
  })();

  /* ── 点睛：点地标出介绍卡 ─────────────────────── */
  const PICKS = {
    yhy: ['颐和园', '清 · 乾隆十五年始建', '清代皇家园林，以昆明湖、万寿山为基址。佛香阁居中，排云殿一路登高，长廊沿湖，十七孔桥卧波。', '皇家园林文化研学', '半天至一天'],
    gugong: ['故宫 · 紫禁城', '明 · 永乐十八年建成', '明清两代皇宫，北京中轴线的核心。角楼立在宫城四隅，九梁十八柱七十二条脊。', '中轴线文化研学', '一天'],
    tiantan: ['天坛', '明 · 永乐十八年始建', '明清两代皇帝祭天、祈谷之所。祈年殿三重檐覆蓝琉璃，象征天。', '中轴线文化研学', '一天'],
    guobo: ['中国国家博物馆', '《复兴之路》基本陈列', '在《复兴之路》展览里重温近代以来的奋斗历程，是党性教育常去的地方。', '红色教育 · 现场教学', '半天'],
    boat: ['晟帆国旅 · 一路随行', '北京晟帆国际旅行社 · 贵昌集团旗下', '红色教育、党政团建、企业团建、会务研学，从方案到落地，一站办妥。', '获取活动方案', ''],
  };
  const card = $('#pickCard');
  const showCard = key => {
    const [t, era, txt, route, dur] = PICKS[key];
    card.innerHTML = `<button class="x" type="button" aria-label="关闭">×</button><h3>${t}</h3><p class="era">${era}</p><p class="txt">${txt}</p><div class="route">${dur ? `<span>相关线路：<b>${route}</b></span><span>${dur}</span>` : `<button class="b-primary sm" type="button" data-act="inquiry">${route}</button>`}</div>`;
    card.hidden = false;
  };
  LAY.main.addEventListener('click', e => { const g = e.target.closest('[data-pick]'); if (g) showCard(g.dataset.pick); });
  boat.addEventListener('click', () => showCard('boat'));
  boat.addEventListener('keydown', e => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); showCard('boat'); } });
  card.addEventListener('click', e => { if (e.target.closest('.x')) card.hidden = true; });

  /* ── 绢纹 ─────────────────────────────────────── */
  (function silk() {
    const n = 220, c = document.createElement('canvas'); c.width = c.height = n;
    const g = c.getContext('2d'), img = g.createImageData(n, n);
    for (let y = 0; y < n; y++) { const row = Math.sin(y * 1.9) * 0.16 + (Math.random() - 0.5) * 0.12; for (let x = 0; x < n; x++) {
      const i = (y * n + x) * 4, v = 238 + (Math.random() * 0.55 + row + Math.sin(x * 0.7 + y * 0.05) * 0.04) * 24;
      img.data[i] = v; img.data[i + 1] = v - 5; img.data[i + 2] = v - 16; img.data[i + 3] = 255; } }
    g.putImageData(img, 0, 0);
    window.SILK_URL = c.toDataURL('image/png');
    $('.silk').style.backgroundImage = `url(${window.SILK_URL})`;
  })();

  /* ── 倒影烘成位图 ───────────────────────────────
     倒影要把颐和园、角楼、十七孔桥整组再画一遍，滚动时最费（实测拖慢一半帧）。
     开场后在后台把倒影画成一张位图贴回水面，滚动时只是贴图；换季时重烘。失败就用回矢量倒影。 */
  const reflG = LAY.main.querySelector('.reflect'), reflSrc = reflG ? reflG.innerHTML : '';
  const cssText = (() => { try { return [...document.styleSheets].filter(sh => /(art2|scene|flora)\.css/.test(sh.href || '')).map(sh => [...sh.cssRules].map(r => r.cssText).join('\n')).join('\n'); } catch (e) { return ''; } })();
  let bakeToken = 0, bakedURL = null;
  async function bakeReflections() {
    if (!reflG || !cssText || !window.Blob || !HTMLCanvasElement.prototype.toBlob) { if (reflG) reflG.innerHTML = reflSrc; return; }
    const token = ++bakeToken, season = stage.dataset.season;
    const ids = [...new Set([...reflSrc.matchAll(/href="#([^"]+)"/g)].map(m => m[1]))];
    const originals = ids.map(id => { const el = document.getElementById(id); return el ? el.outerHTML : ''; }).join('');
    const defsInner = ART2.defs().replace(/^[\s\S]*?<defs>|<\/defs>[\s\S]*$/g, '') + ((LAY.main.querySelector('svg > defs') || {}).innerHTML || '');
    const x0 = 680, x1 = 2540, y0 = S.WY - 4, y1 = S.H + 10, res = Math.min(2, scale * Math.min(devicePixelRatio || 1, 1.5));
    const w = Math.round((x1 - x0) * res), h = Math.round((y1 - y0) * res);
    const str = `<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" width="${w}" height="${h}" viewBox="${x0} ${y0} ${x1 - x0} ${y1 - y0}"><style>${cssText}</style><defs>${defsInner}</defs><g class="sc" data-season="${season}"><g class="a2"><defs>${originals}</defs>${reflSrc}</g></g></svg>`;
    const svgURL = URL.createObjectURL(new Blob([str], { type: 'image/svg+xml' }));
    let keepSvg = false;
    try {
      const img = new Image(); img.src = svgURL; await img.decode();
      let url;
      try {
        const cv = document.createElement('canvas'); cv.width = w; cv.height = h;
        cv.getContext('2d').drawImage(img, 0, 0, w, h);
        const blob = await new Promise(r => cv.toBlob(r, 'image/png'));
        if (!blob) throw new Error('toBlob');
        url = URL.createObjectURL(blob);
      } catch (e) { url = svgURL; keepSvg = true; }       // 画布不让导出：直接用矢量图片，也比 <use> 省
      if (token !== bakeToken) { if (url !== svgURL) URL.revokeObjectURL(url); return; }
      reflG.innerHTML = `<image href="${url}" x="${x0}" y="${y0}" width="${x1 - x0}" height="${y1 - y0}" preserveAspectRatio="none"/>`;
      if (bakedURL) URL.revokeObjectURL(bakedURL);
      bakedURL = url;
    } catch (e) { if (token === bakeToken) reflG.innerHTML = reflSrc; }
    finally { if (!keepSvg) URL.revokeObjectURL(svgURL); }
  }
  if (reflG) reflG.innerHTML = '';

  layout();
  setTimeout(() => { reflBaked = true; bakeReflections(); }, 60);

  /* ── 展卷 ─────────────────────────────────────── */
  const open = () => { stage.classList.remove('closed'); stick.classList.add('open'); };
  if (reduce) open();
  else {
    stage.classList.add('closed');
    requestAnimationFrame(() => requestAnimationFrame(() => setTimeout(open, 250)));
    // 兜底：页面在后台打开、逐帧回调被系统暂停时，1.5 秒后直接展开，首屏不会一直空着
    setTimeout(open, 1500);
  }
  window.JUAN = { layout, update, open, setSeason, stops: () => stopAt, total: () => total, bake: bakeReflections, season: () => stage.dataset.season };
})();
