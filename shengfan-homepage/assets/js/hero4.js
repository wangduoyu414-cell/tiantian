/* 首屏《京华胜景图》v4：读 assets/img/hero/manifest.json，按版式（电脑 / 手机）和季节铺上烘好的各层图片，
   再加上会动的小块（云、鹤、晟帆、雁阵）和交互：
   - 云开见景：图片到齐后，一屏云雾向两边散开，画面从近处推到全景；
   - 滚动分层：电脑上首屏多停半屏，各层按纵深推开，云向两边散；手机上远处的层走得慢；
   - 点地标出题签：竖写小签展开、钤印，四周蒙一层纸色只亮这一处，旁边出介绍卡；
   - 四时：右下角一方印，点一下换季（换整套图片，落花、荷叶、落叶与雁阵、落雪与积雪）；
   - 晟帆：湖上小船，点它出公司介绍。
   页面里的结构见 design/v4/hero.html；数据在 manifest.json，图片由 tools/bake-hero.swift 生成。 */
(function () {
  'use strict';
  const root = document.querySelector('.jhero');
  if (!root) return;
  const $ = s => root.querySelector(s);
  const reduce = matchMedia('(prefers-reduced-motion: reduce)').matches;
  const fine = matchMedia('(hover: hover) and (pointer: fine)').matches;
  const q = new URLSearchParams(location.search);
  const BASE = root.dataset.base || 'assets/img/hero/';
  const clamp = (v, a, b) => Math.min(b, Math.max(a, v));
  if (reduce) root.classList.add('reduce');

  /* ── 节气、季节、干支 ── */
  const JQ = [[1, 6, '小寒'], [1, 20, '大寒'], [2, 4, '立春'], [2, 19, '雨水'], [3, 6, '惊蛰'], [3, 21, '春分'], [4, 5, '清明'], [4, 20, '谷雨'], [5, 6, '立夏'], [5, 21, '小满'], [6, 6, '芒种'], [6, 21, '夏至'],
    [7, 7, '小暑'], [7, 23, '大暑'], [8, 8, '立秋'], [8, 23, '处暑'], [9, 8, '白露'], [9, 23, '秋分'], [10, 8, '寒露'], [10, 23, '霜降'], [11, 7, '立冬'], [11, 22, '小雪'], [12, 7, '大雪'], [12, 22, '冬至']];
  const jqIndex = d => { const m = d.getMonth() + 1, day = d.getDate(); let k = 23; JQ.forEach(([mm, dd], i) => { if (m > mm || (m === mm && day >= dd)) k = i; }); return k; };
  const seasonOf = d => { const i = jqIndex(d); return i >= 2 && i < 8 ? 'spring' : i >= 8 && i < 14 ? 'summer' : i >= 14 && i < 20 ? 'autumn' : 'winter'; };
  const ganzhi = d => { const y = d.getFullYear() - (d.getMonth() === 0 || (d.getMonth() === 1 && d.getDate() < 4) ? 1 : 0); return '甲乙丙丁戊己庚辛壬癸'[(y - 4) % 10] + '子丑寅卯辰巳午未申酉戌亥'[(y - 4) % 12]; };
  const SEASONS = ['spring', 'summer', 'autumn', 'winter'], SN = { spring: '春', summer: '夏', autumn: '秋', winter: '冬' };

  /* ── 地标介绍：名称、年代、一段话、相关线路、时长、印文 ── */
  const PICKS = {
    changcheng: ['万里长城', '明长城 · 八达岭、慕田峪等段', '燕山一线的明长城。八达岭、慕田峪等段落适合文化研学和团队登山，可与冬奥场馆排进同一段行程。', '长城与冬奥', '两日', '长城'],
    yiheyuan: ['颐和园', '清 · 乾隆十五年始建', '清代皇家园林，以昆明湖、万寿山为基址。佛香阁居中，排云殿一路登高，长廊沿湖。', '皇家园林文化研学', '半天至一天', '颐和'],
    shiqikong: ['十七孔桥', '颐和园 · 昆明湖', '昆明湖上的十七孔石桥，连接东堤与南湖岛，是园中最大的一座石桥。', '皇家园林文化研学', '半天至一天', '颐和'],
    beihai: ['北海 · 白塔', '清 · 顺治八年建白塔', '琼华岛上的藏式白塔，是北海的标志。', '中轴线文化研学', '一天', '北海'],
    gugong: ['故宫 · 紫禁城', '明 · 永乐十八年建成', '明清两代皇宫，北京中轴线的核心。角楼立在宫城四隅，九梁十八柱七十二条脊。', '中轴线文化研学', '一天', '故宫'],
    tiantan: ['天坛', '明 · 永乐十八年始建', '明清两代皇帝祭天、祈谷之所。祈年殿三重檐覆蓝琉璃，象征天。', '中轴线文化研学', '一天', '天坛'],
    yongding: ['永定门', '明 · 嘉靖三十二年始建', '明清北京外城的正门，北京中轴线的南端起点。今天的城楼是 2004 年复建的。', '中轴线文化研学', '一天', '永定'],
    guobo: ['中国国家博物馆', '《复兴之路》基本陈列', '在《复兴之路》展览里重温近代以来的奋斗历程，是党性教育常去的地方。', '红色教育 · 现场教学', '半天', '国博'],
    boat: ['晟帆国旅 · 一路随行', '北京晟帆国际旅行社 · 贵昌集团旗下', '红色教育、党政团建、企业团建、会务研学，从方案到落地，一站办妥。', '', '', '晟帆'],
  };

  const SLIPNAME = { changcheng: '万里长城', yiheyuan: '颐和园', shiqikong: '十七孔桥', beihai: '北海白塔', gugong: '紫禁城', tiantan: '天坛', yongding: '永定门', guobo: '国家博物馆' };
  const stick = $('.jh-stick'), stage = $('.jh-stage'), zoom = $('.jh-zoom'), card = $('.jh-card'), sBtn = $('.jh-season'), veil = $('.jh-veil'), slip = $('.jh-slip'), tt = $('.jh-title .tt');
  let M = null, lay = null, layoutKey = '', W = 1, H = 1, OY = 1, season = '', items = [], planeEls = {}, hitEls = {}, fxEl = null, stH = 0, picked = '', opened = false;

  fetch(BASE + 'manifest.json', { cache: 'no-cache' }).then(r => r.json()).then(m => { M = m; setup(); })
    .catch(() => { root.classList.add('open', 'opened'); });

  function setup() {
    if (!document.getElementById('jh-hero-defs')) document.body.insertAdjacentHTML('afterbegin', `<svg id="jh-hero-defs" width="0" height="0" style="position:absolute" aria-hidden="true"><defs>${M.defs}</defs></svg>`);
    const now = new Date();
    const colo = $('.jh-colo');
    if (colo) colo.innerHTML = `<span>京华胜景图</span><span>${ganzhi(now)} · ${JQ[jqIndex(now)][2]}</span><span>晟帆敬制</span><i>晟<br>帆</i>`;
    build(innerWidth <= 760 ? 'mobile' : 'desk');
    setSeason(SEASONS.includes(q.get('season')) ? q.get('season') : seasonOf(now), true);
    if (reduce || q.get('open') === '1') { open(true); }
    else if (q.get('hold') === '1') { /* 出图用：停在开场前，等外面调 JHERO.open() */ }
    else {
      const t0 = performance.now(), imgs = [...zoom.querySelectorAll('.jh-pl img')];
      Promise.all(imgs.map(im => im.complete ? 0 : new Promise(r => { im.addEventListener('load', r, { once: true }); im.addEventListener('error', r, { once: true }); })))
        .then(() => setTimeout(open, Math.max(0, 600 - (performance.now() - t0))));
      setTimeout(open, 4500);          // 兜底：图片迟迟不到也要开
    }
    addEventListener('scroll', schedule, { passive: true });
    let rt; addEventListener('resize', () => { clearTimeout(rt); rt = setTimeout(() => { const k = innerWidth <= 760 ? 'mobile' : 'desk'; if (k !== layoutKey) { build(k); setSeason(season, true); } else size(); }, 160); });
    if (fine && !reduce) root.addEventListener('pointermove', e => { const b = stick.getBoundingClientRect(); tx = ((e.clientX - b.left) / b.width - 0.5) * 2; ty = ((e.clientY - b.top) / b.height - 0.5) * 2; schedule(); });
    sBtn.addEventListener('click', () => setSeason(SEASONS[(SEASONS.indexOf(season) + 1) % 4]));
    stick.addEventListener('click', e => { if (picked && !e.target.closest('.jh-hit, .k-boat, .jh-card, .jh-season, .jh-title .btns')) unpick(); });
    card.addEventListener('click', e => { if (e.target.closest('.x')) unpick(); });
    addEventListener('keydown', e => { if (e.key === 'Escape' && picked) unpick(); });
    // 看不见首屏时停掉动画
    if ('IntersectionObserver' in window) new IntersectionObserver(es => { const v = es[0].isIntersecting; root.classList.toggle('paused', !v); visible = v; if (v) schedule(); }, { rootMargin: '80px' }).observe(root);
    if (q.get('pick')) setTimeout(() => pick(q.get('pick')), 50);
  }

  function open(now) {
    if (opened) return; opened = true;
    root.classList.add('open');
    setTimeout(() => root.classList.add('opened'), now === true ? 0 : 3200);
  }

  /* ── 铺画面：各层、会动的小块、可点的地标 ── */
  const pct = r => `left:${r[0] / W * 100}%;top:${r[1] / H * 100}%;width:${(r[2] - r[0]) / W * 100}%;height:${(r[3] - r[1]) / H * 100}%`;
  function build(k) {
    // 推近的基点放在画面底边：近层放得比远层多，底边对齐时近层只会往上多盖住远层，层与层之间不会拉出缝
    layoutKey = k; lay = M.layouts[k]; W = lay.W; H = lay.H; OY = H;
    root.classList.toggle('m', k === 'mobile');
    zoom.innerHTML = '<div class="jh-sky"></div>';
    items = []; planeEls = {}; hitEls = {};
    // 天和最远的山一起动：远山的雾脚压在天上的位置不变，错层时不露边
    const skyEl = zoom.firstElementChild, d0 = Math.min(...lay.planes.map(p => p.depth));
    skyEl.style.transformOrigin = `50% ${((OY / H * 100 + 14) / 128 * 100).toFixed(2)}%`;   // 天比画面上下各多出 14%
    items.push({ el: skyEl, d: d0, side: 0, kind: 'sky' });
    const list = [...lay.planes.map(o => ({ t: 'pl', d: o.depth, o })), ...lay.sprites.map(o => ({ t: 'spr', d: o.depth, o }))]
      .sort((a, b) => a.d - b.d || (a.t === 'pl' ? -1 : 1));
    for (const { t, d, o } of list) {
      const el = document.createElement('div'), r = o.rect;
      el.style.cssText = `${pct(r)};transform-origin:${(W / 2 - r[0]) / (r[2] - r[0]) * 100}% ${(OY - r[1]) / (r[3] - r[1]) * 100}%`;
      if (t === 'pl') { el.className = 'jh-pl'; el.dataset.id = o.id; planeEls[o.id] = el; }
      else {
        el.className = `jh-spr k-${o.kind}`;
        const vb = `${r[0]} ${r[1]} ${r[2] - r[0]} ${r[3] - r[1]}`, sv = (inner, cls = '') => `<svg class="${cls}" viewBox="${vb}" preserveAspectRatio="none" aria-hidden="true">${inner}</svg>`;
        if (o.kind === 'crane') { el.innerHTML = `<div class="in">${o.poses.map((p, i) => sv(p, 'p' + i)).join('')}</div>`; el.style.setProperty('--ph', `${(-o.phase * 1.3).toFixed(2)}s`); }
        else if (o.kind === 'boat') {
          el.innerHTML = `<div class="in"><div class="bob">${sv(o.svg)}</div></div>`;
          el.tabIndex = 0; el.setAttribute('role', 'button'); el.setAttribute('aria-label', '晟帆国旅，一路随行');
          el.addEventListener('click', e => { e.stopPropagation(); pick('boat'); });
          el.addEventListener('keydown', e => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); pick('boat'); } });
          hitEls.boat = el;
        } else el.innerHTML = `<div class="in">${sv(o.svg)}</div>`;
      }
      zoom.appendChild(el);
      items.push({ el, d, side: o.side || 0, kind: t === 'pl' ? 'pl' : o.kind, id: o.id });
    }
    // 地标按钮放进所在的那一层，随层一起动
    for (const h of lay.hits) {
      const pl = planeEls[h.plane], pr = lay.planes.find(p => p.id === h.plane).rect, r = h.rect, b = document.createElement('button');
      b.type = 'button'; b.className = 'jh-hit'; b.setAttribute('aria-label', PICKS[h.id] ? PICKS[h.id][0] : h.id);
      b.style.cssText = `left:${(r[0] - pr[0]) / (pr[2] - pr[0]) * 100}%;top:${(r[1] - pr[1]) / (pr[3] - pr[1]) * 100}%;width:${(r[2] - r[0]) / (pr[2] - pr[0]) * 100}%;height:${(r[3] - r[1]) / (pr[3] - pr[1]) * 100}%`;
      b.addEventListener('click', e => { e.stopPropagation(); picked === h.id ? unpick() : pick(h.id); });
      pl.appendChild(b); hitEls[h.id] = b;
    }
    fxEl = document.createElement('div'); fxEl.className = 'jh-fx'; zoom.appendChild(fxEl);
    cover();
    size();
  }

  // 开场的云：取画里的勾云，放大铺满，左边的往左散、右边的往右散
  function cover() {
    const cv = $('.jh-cover'); if (!cv || opened) return;
    const clouds = lay.sprites.filter(s => s.kind === 'cloud');
    const spots = layoutKey === 'desk'
      ? [[-18, -6, 70, 'l'], [6, 10, 52, 'l'], [-10, 22, 64, 'l'], [-24, 38, 74, 'l'], [2, 48, 56, 'l'], [-14, 62, 70, 'l'], [-4, 78, 62, 'l'], [-22, 88, 72, 'l'],
         [46, -4, 68, 'r'], [60, 12, 56, 'r'], [40, 24, 62, 'r'], [56, 40, 66, 'r'], [36, 52, 58, 'r'], [52, 66, 70, 'r'], [42, 80, 60, 'r'], [58, 90, 66, 'r']]
      : [[-34, 0, 116, 'l'], [-14, 14, 96, 'l'], [-40, 28, 120, 'l'], [-18, 44, 100, 'l'], [-38, 60, 118, 'l'], [-16, 76, 104, 'l'], [-36, 90, 116, 'l'],
         [20, 6, 106, 'r'], [34, 20, 112, 'r'], [16, 36, 104, 'r'], [30, 52, 116, 'r'], [18, 68, 108, 'r'], [32, 84, 112, 'r']];
    cv.innerHTML = '<div class="cv-m"></div>' + spots.map(([l, t, w, side], i) => {
      const c = clouds[i % clouds.length], r = c.rect;
      return `<div class="cv-c ${side}" style="left:${l}%;top:${t}%;width:${w}%;transition-delay:${(i % 5) * 0.07}s,${1.05 + (i % 5) * 0.07}s"><svg viewBox="${r[0]} ${r[1]} ${r[2] - r[0]} ${r[3] - r[1]}" aria-hidden="true">${c.svg}</svg></div>`;
    }).join('');
  }

  function size() {
    const vw = root.clientWidth;
    if (layoutKey === 'mobile') {
      stH = H * vw / W;
      root.style.height = stick.style.height = `${stH}px`;
      stage.style.cssText = `left:0;top:0;width:${vw}px;height:${stH}px`;
    } else {
      root.style.height = stick.style.height = '';
      const vh = stick.clientHeight, s = Math.max(vw / W, vh / H);
      stH = H * s;
      stage.style.cssText = `left:${(vw - W * s) / 2}px;top:${vh - stH}px;width:${W * s}px;height:${stH}px`;
    }
    fxEl.style.setProperty('--fall', `${(stH * 1.06).toFixed(0)}px`);
    lastKey = ''; tick();
  }

  /* ── 四时：换整套图片（淡入）、换落物 ── */
  const src = (id, s, ext) => `${BASE}${layoutKey}-${s}-${id}.${ext}?v=${M.version}`;
  const pic = (id, s) => `<picture><source type="image/avif" srcset="${src(id, s, 'avif')}"><img src="${src(id, s, 'png')}" alt="" decoding="async"></picture>`;
  function setSeason(s, initial) {
    season = s; root.dataset.season = s;
    sBtn.querySelector('b').textContent = SN[s]; sBtn.setAttribute('aria-label', `四时：当前${SN[s]}，点按换季`);
    for (const p of lay.planes) {
      const el = planeEls[p.id], hits = [...el.querySelectorAll('.jh-hit')];
      if (initial) { el.innerHTML = pic(p.id, s); hits.forEach(h => el.appendChild(h)); continue; }
      const sw = document.createElement('div'); sw.className = 'swap'; sw.innerHTML = pic(p.id, s);
      el.insertBefore(sw, hits[0] || null);
      const img = sw.querySelector('img');
      const done = () => requestAnimationFrame(() => { sw.classList.add('in'); setTimeout(() => { [...el.children].forEach(c => { if (c !== sw && !c.classList.contains('jh-hit')) c.remove(); }); sw.replaceWith(sw.firstElementChild); }, 950); });
      if (img.complete) done(); else { img.addEventListener('load', done, { once: true }); img.addEventListener('error', done, { once: true }); }
    }
    fx(s);
    if (!initial) { sBtn.classList.remove('stamp'); void sBtn.offsetWidth; sBtn.classList.add('stamp'); }
  }
  function fx(s) {
    fxEl.className = `jh-fx fx-${s}`;
    if (reduce || s === 'summer') { fxEl.innerHTML = ''; return; }
    const n = layoutKey === 'desk' ? (s === 'winter' ? 46 : 30) : (s === 'winter' ? 26 : 18), R = Math.random;
    let h = '';
    for (let i = 0; i < n; i++) {
      const sz = s === 'winter' ? 2.5 + R() * 4 : 6 + R() * 7;
      h += `<i style="left:${(R() * 104 - 2).toFixed(1)}%;--s:${sz.toFixed(1)}px;--d:${(s === 'winter' ? 9 : 12) + R() * 10}s;--dl:-${(R() * 20).toFixed(1)}s;--dx:${((R() - 0.5) * 160).toFixed(0)}px;--r:${((R() - 0.5) * 720).toFixed(0)}deg;--o:${(0.55 + R() * 0.4).toFixed(2)}"></i>`;
    }
    fxEl.innerHTML = h;
  }

  /* ── 滚动分层、鼠标视差 ── */
  let tx = 0, ty = 0, mx = 0, my = 0, raf = 0, lastKey = '', visible = true, pickP = 0;
  function schedule() { if (!raf) raf = requestAnimationFrame(tick); }
  function tick() {
    raf = 0;
    const r = root.getBoundingClientRect();
    let p = 0, sy = 0;
    if (layoutKey === 'desk') { const span = root.offsetHeight - stick.offsetHeight; p = span > 0 ? clamp(-r.top / span, 0, 1) : 0; }
    else sy = clamp(-r.top, 0, stH);
    if (reduce) p = 0, sy = 0;
    mx += (tx - mx) * 0.08; my += (ty - my) * 0.08;
    const key = `${p.toFixed(4)}|${sy | 0}|${mx.toFixed(3)}|${my.toFixed(3)}`;
    if (key !== lastKey) {
      lastKey = key;
      for (const it of items) {
        const d = it.d;
        let x = mx * d * 18, y = my * d * 9, s = 1;
        if (layoutKey === 'desk') s = 1 + p * d * d * 0.24;
        else y += sy * (1 - d) * 0.3;
        if (it.kind === 'cloud') x += it.side * p * 320 * (0.6 + d);
        if (it.id === 'bough') it.el.style.opacity = Math.max(0, 1 - p * 1.4).toFixed(3);   // 推近时从松枝下穿过，别压到顶栏
        it.el.style.transform = `translate3d(${x.toFixed(2)}px,${y.toFixed(2)}px,0)${s !== 1 ? ` scale(${s.toFixed(4)})` : ''}`;
      }
      if (tt && layoutKey === 'desk') { tt.style.opacity = (1 - p * 1.8).toFixed(3); tt.style.transform = `translateY(${(-p * 50).toFixed(1)}px)`; }
      if (picked) { if (Math.abs(p - pickP) > 0.12) unpick(); else place(); }
    }
    if (visible && (Math.abs(tx - mx) > 0.003 || Math.abs(ty - my) > 0.003)) schedule();
  }

  /* ── 点地标：题签、聚光、介绍卡 ── */
  function pick(id) {
    const P = PICKS[id]; if (!P || !hitEls[id]) return;
    if (picked && hitEls[picked]) hitEls[picked].classList.remove('on');
    picked = id; pickP = layoutKey === 'desk' ? clamp(-root.getBoundingClientRect().top / Math.max(1, root.offsetHeight - stick.offsetHeight), 0, 1) : 0;
    hitEls[id].classList.add('on');
    if (id === 'boat') slip.hidden = true;
    else { slip.hidden = false; slip.innerHTML = `<b>${SLIPNAME[id] || P[0]}</b><i>${[...P[5]].join('<br>')}</i>`; slip.classList.remove('on'); void slip.offsetWidth; slip.classList.add('on'); }
    card.innerHTML = `<button class="x" type="button" aria-label="关闭">×</button><h3>${P[0]}</h3><p class="era">${P[1]}</p><p class="txt">${P[2]}</p>` +
      (P[3] ? `<div class="route"><span>相关线路：<b>${P[3]}</b> · ${P[4]}</span><button class="go" type="button" data-act="inquiry" data-pre="${id === 'guobo' ? '红色教育 / 主题党日' : '研学考察'}" data-note="想去：${P[0]}">定制方案</button></div>` : `<div class="route"><span>立足首都 · 服务政企</span><button class="go" type="button" data-act="inquiry">获取活动方案</button></div>`);
    card.hidden = false;
    root.classList.add('picking');
    place();
    // 手机：卡片从底部升起，把这一处挪到卡片上方看得见的地方
    if (innerWidth <= 760) {
      const hr = hitEls[id].getBoundingClientRect(), top = 70, bottom = innerHeight - card.offsetHeight - 16;
      if (hr.top < top || hr.bottom > bottom) scrollBy({ top: (hr.top + hr.bottom) / 2 - (top + bottom) / 2, behavior: reduce ? 'auto' : 'smooth' });
    }
  }
  function unpick() {
    if (picked && hitEls[picked]) hitEls[picked].classList.remove('on');
    picked = ''; card.hidden = true; slip.classList.remove('on'); root.classList.remove('picking');
    setTimeout(() => { if (!picked) slip.hidden = true; }, 400);
  }
  // 题签、聚光、卡片的位置：按地标按钮此刻在屏幕上的位置换算（层在动，跟着它走）
  function place() {
    const id = picked; if (!id) return;
    const sr = stick.getBoundingClientRect(), hr = hitEls[id].getBoundingClientRect();
    const cx = hr.left + hr.width / 2 - sr.left, cy = hr.top + hr.height / 2 - sr.top;
    veil.style.setProperty('--vx', `${cx.toFixed(0)}px`); veil.style.setProperty('--vy', `${cy.toFixed(0)}px`);
    veil.style.setProperty('--vr', `${(hr.width * 0.95 + 70).toFixed(0)}px`); veil.style.setProperty('--vry', `${(hr.height * 0.95 + 60).toFixed(0)}px`);
    let anchor = hr;
    if (id !== 'boat') {
      const h = lay.hits.find(x => x.id === id), r = h.rect, sx = (h.slip[0] - r[0]) / (r[2] - r[0]), sy = (h.slip[1] - r[1]) / (r[3] - r[1]);
      const lx = hr.left + sx * hr.width - sr.left, ly = hr.top + sy * hr.height - sr.top;
      const sw = slip.offsetWidth, sh = slip.offsetHeight;
      slip.style.left = `${clamp(lx, 8, sr.width - sw - 8).toFixed(1)}px`; slip.style.top = `${clamp(ly, 70, sr.height - sh - 84).toFixed(1)}px`;
      anchor = slip.getBoundingClientRect();
    }
    if (innerWidth > 760) {
      const cw = card.offsetWidth, ch = card.offsetHeight, pad = 16, gap = 16;
      const rel = b => ({ l: b.left - sr.left, t: b.top - sr.top, r: b.right - sr.left, b: b.bottom - sr.top });
      const A = rel(anchor), Hh = rel(hr), T = tt ? rel(tt.getBoundingClientRect()) : null;
      const avoid = [[Hh, 6], [A, 6]]; if (T && +getComputedStyle(tt).opacity > 0.2) avoid.push([T, 3]);
      const cands = [[A.r + gap, A.t], [Math.min(A.l, Hh.l) - cw - gap, A.t], [A.r - cw, A.b + gap], [A.l, A.b + gap], [Hh.r + gap, Hh.b - ch], [Hh.l - cw - gap, Hh.b - ch], [A.r + gap, A.b - ch]];
      const over = (l, t) => avoid.reduce((a, [k, w]) => a + w * Math.max(0, Math.min(l + cw, k.r) - Math.max(l, k.l)) * Math.max(0, Math.min(t + ch, k.b) - Math.max(t, k.t)), 0);
      let best = null, bs = Infinity;
      for (let [l, t] of cands) { l = clamp(l, pad, sr.width - cw - pad); t = clamp(t, 70, sr.height - ch - pad); const sc = over(l, t) / 20 + Math.hypot(l - A.r, t - A.t) * 0.5; if (sc < bs) { bs = sc; best = [l, t]; } }
      card.style.left = `${best[0].toFixed(0)}px`; card.style.top = `${best[1].toFixed(0)}px`;
    }
  }

  window.JHERO = { pick, unpick, setSeason, open };
})();
