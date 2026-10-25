/* 晟帆国旅主页（第四版）· 首屏之后各章：红色教育、团建、会务研学、北京导览图、资质与保障、项目档案、名片。
   内容数据都在本文件里，改字只改这里；首屏见 hero4.js。
   各章的画是烘好的图片（assets/img/site/，tools/bake-site.swift 生成），导览图上的 39 枚小画用 assets/js/jh/ 现画。 */
(function () {
  'use strict';
  const $ = (s, r = document) => r.querySelector(s), $$ = (s, r = document) => [...r.querySelectorAll(s)];
  const inWeChat = /MicroMessenger/i.test(navigator.userAgent);
  const desktop = matchMedia('(min-width: 900px)');
  const reduce = matchMedia('(prefers-reduced-motion: reduce)').matches;
  const esc = s => String(s).replace(/[&<>"]/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' })[c]);
  const yin = (ch, kind = '') => `<i class="yin${ch.length > 1 ? ' yin-2' : ''}${kind ? ' ' + kind : ''}" aria-hidden="true">${ch}</i>`;
  if (reduce) document.documentElement.classList.add('reduce');

  /* ================= 内容 ================= */
  const TEAM = {
    gov: {
      note: '主题鲜明、规范节俭，可与主题党日、工会活动结合安排。',
      items: [
        ['dingxiang', '寻', '红色定向', '以红色地标为点位分组寻访，边走边学。', ['半天', '30–200 人']],
        ['gonghui', '聚', '主题党日 + 工会活动', '上午集中学习，下午集体活动，学习与凝聚兼顾。', ['一天', '30–150 人']],
        ['laodong', '耕', '劳动教育', '走进京郊农场参与农事，体会劳动的价值。', ['半天至一天', '春夏秋']],
        ['feiyi', '艺', '非遗文化体验', '风筝、剪纸、景泰蓝等手作，了解北京传统文化。', ['半天', '20–80 人']],
        ['dengshan', '登', '健步走 · 登山', '奥森、香山、长城等线路，强健体魄。', ['半天', '春秋季']],
        ['dushu', '读', '读书分享会', '选定书目，组织分享研讨，形成学习成果。', ['半天', '20–60 人']],
      ],
    },
    ent: {
      note: '重在凝聚与体验，可按人数、预算和季节灵活组合。',
      items: [
        ['tuozhan', '拓', '拓展训练', '专业教练带队，破冰、协作、挑战层层递进。', ['一天', '20–300 人']],
        ['tubu', '行', '徒步穿越', '京西古道、长城、京郊山野线路，全程保障。', ['一天', '春秋季']],
        ['yundong', '赛', '趣味运动会', '场地、器材、裁判、奖品一站安排。', ['半天', '50–500 人']],
        ['gongchuang', '创', '团队共创', '引导师带领目标共识与问题共创，团建也出成果。', ['半天', '20–80 人']],
        ['nianhui', '庆', '年会庆典', '策划、舞美、节目、主持、摄像全流程执行。', ['半天至一天', '100 人以上']],
        ['yanxue', '研', '研学团建', '走进标杆企业与科创园区，学习与团建结合。', ['一天', '20–100 人']],
      ],
    },
  };
  const CERTS = [
    ['mice', 'sf-mice', '会务会展服务企业资质证书', '一级', [['证书编号', 'ZDXY-2024182720'], ['有效期', '2024.07.26 – 2027.07.25'], ['评价机构', '北京中鼎环宇国际信用评价有限公司']], ['中国招标投标网（证书公示）', 'http://www.cecbid.org.cn/']],
    ['travel', 'sf-travel', '旅行社业务经营许可证', '行政许可', [['许可机关', '北京市文化和旅游局']], ['文化和旅游部（旅行社名录可查）', 'https://www.mct.gov.cn/']],
    ['iso9001', 'sf-iso9001', '质量管理体系认证', 'ISO 9001', [['证书编号', 'HG24Q0503R0S'], ['有效期', '2024.11.06 – 2027.11.05']], ['全国认证认可信息公共服务平台', 'http://cx.cnca.cn/']],
    ['iso14001', 'sf-iso14001', '环境管理体系认证', 'ISO 14001', [['证书编号', 'HG24E0280R0S'], ['有效期', '2024.11.06 – 2027.11.05']], ['全国认证认可信息公共服务平台', 'http://cx.cnca.cn/']],
    ['iso45001', 'sf-iso45001', '职业健康安全管理体系认证', 'ISO 45001', [['证书编号', 'HG24S0243R0S'], ['有效期', '2024.11.06 – 2027.11.05']], ['全国认证认可信息公共服务平台', 'http://cx.cnca.cn/']],
  ];
  const GUARDS = [
    ['保', '旅行社责任险', '人保旅行社责任保险统保示范项目'],
    ['险', '意外与专项险', '旅游意外、高原、自驾、航班延误险按需投保'],
    ['车', '车辆', '50 座大巴、16 座考斯特到商务车、礼宾车'],
    ['急', '应急预案', '每个项目单独制定，现场有人负责'],
    ['专', '专人对接', '从方案到结算，一位负责人跟到底'],
    ['票', '合同票据', '合同、发票、结算单据齐全'],
  ];
  const GC_STRIP = ['gaoxin', 'zjtx', 'iso9001', 'iso14001', 'iso45001', 'iso27001', 'iso22000', 'iso28000', 'haccp', 'shipin', 'yiliao', 'icp', 'dengbao', 'trace', 'shouhou', 'chuji', 'shengxian'];
  // 项目档案：真实项目到位后，把 placeholder 改成 false、填好字段，photo 填照片路径
  const CASES = [
    { kind: 'red', tag: '红色教育', year: '待补充', title: '项目名称待补充', when: '待补充', where: '待补充', size: '待补充', what: '待补充', placeholder: true },
    { kind: 'lv', tag: '团建', year: '待补充', title: '项目名称待补充', when: '待补充', where: '待补充', size: '待补充', what: '待补充', placeholder: true },
    { kind: 'qing', tag: '会务', year: '待补充', title: '项目名称待补充', when: '待补充', where: '待补充', size: '待补充', what: '待补充', placeholder: true },
  ];


  /* ================= 底子：纸纹 ================= */
  (function paper() {
    try {
      const n = 220, c = document.createElement('canvas'); c.width = c.height = n;
      const g = c.getContext('2d'), img = g.createImageData(n, n);
      for (let y = 0; y < n; y++) { const row = Math.sin(y * 1.9) * 0.16 + (Math.random() - 0.5) * 0.12; for (let x = 0; x < n; x++) {
        const i = (y * n + x) * 4, v = 238 + (Math.random() * 0.55 + row + Math.sin(x * 0.7 + y * 0.05) * 0.04) * 24;
        img.data[i] = v; img.data[i + 1] = v - 5; img.data[i + 2] = v - 16; img.data[i + 3] = 255; } }
      g.putImageData(img, 0, 0);
      document.body.style.setProperty('--silk-img', `url(${c.toDataURL('image/png')})`);
    } catch (e) { /* 没有画布就用纯色底 */ }
  })();
  // 导览图上的小画要用到的渐变（页面里放一次）
  if (window.JH && JH.ARCH) document.body.insertAdjacentHTML('afterbegin', JH.ARCH.defs());

  /* ================= 烘好的画：清单、图片 ================= */
  // 各章的主画面、场景小图、导览图底图是 tools/bake-site.swift 烘好的图片；清单里记着尺寸、画里可点的地方和题签位置
  const ART = 'assets/img/site/';
  let MAN = { version: 0, items: {} };
  const art = (name, o = {}) => {
    const it = MAN.items[name] || { w: 1, h: 1, alpha: false }, v = MAN.version;
    return `<picture class="${o.cls || 'art'}"><source type="image/avif" srcset="${ART}${name}.avif?v=${v}"><img src="${ART}${name}.${it.alpha ? 'png' : 'jpg'}?v=${v}" width="${it.w}" height="${it.h}" alt="${esc(o.alt || '')}"${o.eager ? '' : ' loading="lazy"'} decoding="async"${o.style ? ` style="${o.style}"` : ''}></picture>`;
  };
  const isMobile = () => innerWidth <= 760;
  const pct = (v, of) => `${(v / of * 100).toFixed(3)}%`;

  // 一章的主画面：图片、可点的地方、竖写题签、右上角的图签
  function chapterStage(sec, key, names, title, onPick) {
    const stage = $('.stage', sec), lay = isMobile() ? 'mobile' : 'desk', it = MAN.items[`${key}-${lay}`];
    if (!it) return null;
    const m = it.meta;
    stage.innerHTML = art(`${key}-${lay}`, { alt: title[0], eager: false }) + `<div class="title-slip"><b>${title[0]}</b><i>${title[1]}</i></div>`;
    const hitEls = {}, slipEls = {};
    for (const h of m.hits) {
      const b = document.createElement('button'); b.className = 'hit'; b.type = 'button'; b.setAttribute('aria-label', names[h.id]);
      Object.assign(b.style, { left: pct(h.x, m.W), top: pct(h.y, m.H), width: pct(h.w, m.W), height: pct(h.h, m.H) });
      b.onclick = e => { e.stopPropagation(); onPick(h.id); }; stage.appendChild(b); hitEls[h.id] = b;
      const [sx, sy] = m.slips[h.id], s = document.createElement('button'); s.className = 'slip'; s.type = 'button'; s.textContent = names[h.id];
      s.style.left = pct(sx, m.W); s.style.top = pct(sy, m.H);
      s.onclick = e => { e.stopPropagation(); onPick(h.id); }; stage.appendChild(s); slipEls[h.id] = s;
    }
    return { stage, hitEls, slipEls, mark: id => { for (const k in hitEls) { hitEls[k].classList.toggle('on', k === id); slipEls[k].classList.toggle('on', k === id); } } };
  }
  // 桌面：介绍卡放在画里的空处（多半是天空），不盖住任何一处地标和图签；同样空的位置里挑离这一处近的
  function placePop(sec, pop, S, id) {
    if (isMobile()) return;
    const fig = $('.leaf', sec).getBoundingClientRect(), st = S.stage.getBoundingClientRect(), cw = pop.offsetWidth, ch = pop.offsetHeight, pad = 18;
    const rel = r => ({ l: r.left - fig.left, t: r.top - fig.top, r: r.right - fig.left, b: r.bottom - fig.top });
    const obst = [...Object.values(S.hitEls).map(e => rel(e.getBoundingClientRect())), rel($('.title-slip', S.stage).getBoundingClientRect())];
    const me = rel(S.hitEls[id].getBoundingClientRect()), cx = (me.l + me.r) / 2, cy = (me.t + me.b) / 2;
    const L0 = st.left - fig.left + pad, T0 = st.top - fig.top + pad, L1 = st.right - fig.left - cw - pad, T1 = st.bottom - fig.top - ch - pad;
    const cands = [];
    for (let i = 0; i <= 8; i++) for (let j = 0; j <= 4; j++) cands.push([L0 + (L1 - L0) * i / 8, T0 + (T1 - T0) * j / 4]);
    cands.push([me.r + 16, me.t], [me.l - cw - 16, me.t], [cx - cw / 2, me.t - ch - 16]);
    const over = (l, t) => obst.reduce((a, k) => a + Math.max(0, Math.min(l + cw, k.r) - Math.max(l, k.l)) * Math.max(0, Math.min(t + ch, k.b) - Math.max(t, k.t)), 0);
    let best = cands[0], bs = Infinity;
    for (let [l, t] of cands) { l = Math.max(L0, Math.min(L1, l)); t = Math.max(T0, Math.min(T1, t)); const sc = over(l, t) / 40 + Math.hypot(l + cw / 2 - cx, t + ch / 2 - cy); if (sc < bs) { bs = sc; best = [l, t]; } }
    pop.style.left = best[0] + 'px'; pop.style.top = best[1] + 'px';
  }
  // 弹层（完整行程、服务流程）：点外面或 × 关
  function modal(el) { el.addEventListener('click', e => { if (e.target === el || e.target.closest('.x')) el.hidden = true; }); return () => { el.hidden = false; }; }
  addEventListener('keydown', e => { if (e.key === 'Escape') $$('.modal').forEach(m => { m.hidden = true; }); });

  /* ================= 壹 红色教育 ================= */
  function initRed() {
    const sec = $('#red'); if (!sec) return;
    const SPOT = Object.fromEntries(JH.MAP.SPOTS.map(s => [s[0], { name: s[1], area: s[2], themes: s[4], desc: s[5], fit: s[6] }]));
    const NAMES = { honglou: '北大红楼', lidazhao: '李大钊故居', lugou: '卢沟桥', xiangshan: '香山 · 双清别墅', dangshi: '党史展览馆' };
    const pop = $('.pop', sec);
    let S;
    const open = id => {
      const sp = SPOT[id]; pop.hidden = false;
      pop.innerHTML = `<div class="strip"><b>${sp.name.replace(' · ', '·')}</b><i>红色<br>教育</i></div><div class="body"><button class="x" type="button" aria-label="关闭">×</button><p class="meta">${sp.area}${sp.themes.length ? ' · ' + sp.themes.join('、') : ''}</p><p>${sp.desc}</p><div class="fit">${sp.fit.map(f => `<span>${f}</span>`).join('')}</div><button class="b1" type="button" data-act="inquiry" data-pre="红色教育 / 主题党日" data-note="想去：${sp.name}">按这里定制方案</button></div>`;
      $('.x', pop).onclick = () => { pop.hidden = true; S.mark(''); };
      S.mark(id); placePop(sec, pop, S, id);
    };
    const build = () => { pop.hidden = true; S = chapterStage(sec, 'red', NAMES, ['红色地标群像', '朱墨<br>界画'], open); };
    build();
    const FORMS = [['学', '现场教学', '走进旧址和纪念馆，讲解员带着看、带着讲。', 'vField'], ['日', '主题党日', '在纪念地敬献花篮、重温入党誓词，流程按支部需要安排。', 'vDay'], ['课', '专题党课', '邀请讲师结合主题授课，也可配合单位自己讲。', 'vLecture'], ['景', '情景教学', '情景再现、诵读和访谈，让历史在现场“活”起来。', 'vScene']];
    $('.forms', sec).innerHTML = FORMS.map(([s, t, d, k]) => `<figure class="form"><div class="pic">${art('red-' + k, { alt: t })}</div><figcaption class="txt"><span class="seal">${s}</span><h4>${t}</h4><p>${d}</p></figcaption></figure>`).join('');
    const TL = [['08:30', '集合出发', '统一乘车，负责人随车说明当天安排', 'bus'], ['09:30', '北大红楼', '讲解员带领参观，现场教学', 'honglou'], ['11:00', '主题交流', '结合本单位工作开展研讨', 'meeting'], ['13:30', '香山革命纪念地', '现场教学，重温入党誓词', 'xiangshan'], ['16:30', '返程', '安全送达各集合点', 'bus']];
    $('.tl', sec).innerHTML = TL.map(([t, h, d, k], i) => `<div class="node"><div class="ic">${art('tl-' + k, { cls: 'icon' })}</div><span class="dot">${'一二三四五'[i]}</span><time>${t}</time><h4>${h}</h4><p>${d}</p></div>`).join('');
    const ROWS = [['08:30', '集合出发', '统一乘车，负责人随车说明当天安排'], ['09:30', '北大红楼', '专业讲解，参观主题展览'], ['11:00', '主题交流', '结合本单位工作开展研讨'], ['12:00', '午餐', '按标准安排工作餐'], ['13:30', '香山革命纪念地', '现场教学，重温入党誓词'], ['16:00', '学习小结', '整理学习材料和影像资料'], ['16:30', '返程', '安全送达各集合点']];
    const md = $('.modal', sec);
    $('.panel', md).innerHTML = `<button class="x" type="button" aria-label="关闭">×</button><h3 id="redItinH">红色足迹 · 党性教育</h3><p class="d">现场教学一日 · 示例行程，可按单位学习主题定制</p><ol>${ROWS.map(([t, b, s]) => `<li><time>${t}</time><b>${b}</b><span>${s}</span></li>`).join('')}</ol><div class="inc">一并安排：${['讲解', '车辆', '保险', '餐食', '随团负责人'].map(x => `<span>${x}</span>`).join('')}</div><button class="b1" type="button" data-act="inquiry" data-pre="红色教育 / 主题党日" data-note="参考：红色足迹 · 党性教育一日行程">按这个行程定制方案</button>`;
    $('[data-open="itin"]', sec).onclick = modal(md);
    const WALL = [['f-a', '北大红楼 · 现场教学', 'red-vField'], ['f-b', '纪念地 · 敬献花篮', 'red-vDay'], ['f-c', '香山 · 双清别墅', 'red-shuangqing'], ['f-d', '专题党课', 'red-vLecture'], ['f-e', '情景教学', 'red-vScene']];
    $('.frames', sec).innerHTML = WALL.map(([c, t, k]) => `<figure class="frame ${c}"><div class="fr"><div class="mat"><div class="ph">${art(k)}<em>照片待补</em></div></div></div><figcaption>${t}</figcaption></figure>`).join('');
    return build;
  }

  /* ================= 贰 团建 ================= */
  function initTeam() {
    const sec = $('#team'); if (!sec) return;
    // 画里七处，各处能办的活动（对应 TEAM 里的条目）
    const SPOTS = {
      wall: ['长城登山', '长城 · 登山', '八达岭、慕田峪等段落登城健步走，团队登山。', ['dengshan'], ['tubu']],
      trail: ['京西古道', '京西古道 · 徒步', '古商道上的石板路和关城，适合团队徒步、红色定向。', ['dingxiang'], ['tubu']],
      farm: ['京郊农场', '京郊农场 · 农事', '梯田和果园里参加农事，体会劳动的价值。', ['laodong'], []],
      meadow: ['拓展营地', '草地 · 拓展营地', '高空拓展、团队挑战、趣味运动会，专业教练带队。', [], ['tuozhan', 'yundong']],
      kite: ['沙燕风筝', '沙燕风筝 · 非遗', '扎、糊、绘、放北京沙燕风筝，了解传统手艺。', ['feiyi'], []],
      hall: ['湖边厅堂', '湖边厅堂', '安静的厅堂和水边，适合主题党日、读书会、团队共创和年会。', ['gonghui', 'dushu'], ['gongchuang', 'nianhui']],
      park: ['科技园区', '标杆企业 · 研学', '走进科创园区与标杆企业参访交流。', [], ['yanxue']],
    };
    const NAMES = Object.fromEntries(Object.entries(SPOTS).map(([k, v]) => [k, v[0]]));
    const PRE = { gov: '党政团建', ent: '企业团建' };
    const pop = $('.pop', sec), menu = $('.menu', sec), note = $('.tab-note', sec);
    let S, kind = 'gov';
    const items = k => TEAM[k].items;
    const open = id => {
      const [, name, desc, gov, ent] = SPOTS[id];
      let k = kind, ids = k === 'gov' ? gov : ent;
      if (!ids.length) { k = k === 'gov' ? 'ent' : 'gov'; ids = k === 'gov' ? gov : ent; }
      const list = items(k).filter(it => ids.includes(it[0]));
      pop.hidden = false;
      pop.innerHTML = `<div class="strip"><b>${name.replace(' · ', '·')}</b><i>${k === 'gov' ? '党政<br>团建' : '企业<br>团建'}</i></div><div class="body"><button class="x" type="button" aria-label="关闭">×</button><p class="meta">${desc}</p><ul>${list.map(([, c, t, d]) => `<li>${yin(c)}<div><b>${t}</b><span>${d}</span></div></li>`).join('')}</ul><button class="b1" type="button" data-act="inquiry" data-pre="${PRE[k]}" data-note="想去：${name}">按这里定制方案</button></div>`;
      $('.x', pop).onclick = () => { pop.hidden = true; S.mark(''); $$('li', menu).forEach(li => li.classList.remove('sel')); };
      S.mark(id); placePop(sec, pop, S, id);
      if (k !== kind) setKind(k, true);
      $$('li', menu).forEach(li => li.classList.toggle('sel', ids.includes(li.dataset.item)));
    };
    function setKind(k, keep) {
      kind = k;
      $$('.tab', sec).forEach(b => { const on = b.dataset.tab === k; b.classList.toggle('on', on); b.setAttribute('aria-selected', on); });
      note.textContent = TEAM[k].note;
      menu.innerHTML = items(k).map(([id, c, t, d, meta]) => `<li data-item="${id}" tabindex="0">${yin(c)}<b>${t}</b><p>${d}</p><span class="meta">${meta.map(m => `<i>${m}</i>`).join('')}</span></li>`).join('');
      $('.menu-act .b1', sec).dataset.pre = PRE[k];
      // 画上只亮这一类活动发生的地方
      if (S) for (const [id, v] of Object.entries(SPOTS)) S.slipEls[id].classList.toggle('dim', !(k === 'gov' ? v[3] : v[4]).length);
      if (!keep) { pop.hidden = true; if (S) S.mark(''); }
    }
    // 点下面的活动：画上亮出能办它的地方
    const pickItem = li => {
      const id = li.dataset.item;
      $$('li', menu).forEach(x => x.classList.toggle('sel', x === li));
      for (const [sid, v] of Object.entries(SPOTS)) { const has = (kind === 'gov' ? v[3] : v[4]).includes(id); S.slipEls[sid].classList.toggle('on', has); S.hitEls[sid].classList.toggle('on', has); }
      pop.hidden = true;
      const r = S.stage.getBoundingClientRect(); if (r.bottom < 80 || r.top > innerHeight - 120) S.stage.scrollIntoView({ behavior: reduce ? 'auto' : 'smooth', block: 'center' });
    };
    menu.addEventListener('click', e => { const li = e.target.closest('li[data-item]'); if (li) pickItem(li); });
    menu.addEventListener('keydown', e => { const li = e.target.closest('li[data-item]'); if (li && (e.key === 'Enter' || e.key === ' ')) { e.preventDefault(); pickItem(li); } });
    $$('.tab', sec).forEach(b => b.addEventListener('click', () => setKind(b.dataset.tab)));
    // 照片版位：先用画里的几处局部占位（灰度），换上现场照片即可
    const strip = () => {
      const lay = isMobile() ? 'mobile' : 'desk', it = MAN.items[`team-${lay}`]; if (!it) return;
      const { W, H } = it.meta;
      const CROPS = lay === 'desk' ? [['拓展营地', [800, 430, 1240, 760]], ['京郊农场', [460, 400, 860, 700]], ['湖边厅堂', [1180, 450, 1600, 765]]]
        : [['拓展营地', [10, 450, 250, 630]], ['京郊农场', [170, 280, 400, 452]], ['湖边厅堂', [10, 680, 300, 898]]];
      $('.strip', sec).innerHTML = CROPS.map(([t, [x0, y0, x1, y1]]) => `<figure class="frame"><div class="fr"><div class="mat"><div class="crop" style="aspect-ratio:${x1 - x0} / ${y1 - y0}">${art(`team-${lay}`, { cls: 'pic', style: `width:${(W / (x1 - x0) * 100).toFixed(2)}%;left:${(-x0 / (x1 - x0) * 100).toFixed(2)}%;top:${(-y0 / (y1 - y0) * 100).toFixed(2)}%` })}<em>照片待补</em></div></div></div><figcaption>${t}</figcaption></figure>`).join('');
    };
    const build = () => { pop.hidden = true; S = chapterStage(sec, 'team', NAMES, ['京郊山野图', '团建'], open); setKind(kind); strip(); };
    build();
    return build;
  }

  /* ================= 叁 会务研学 ================= */
  function initMice() {
    const sec = $('#mice'); if (!sec) return;
    const VENUE = {
      yanqi: ['雁栖湖国际会都', '怀柔区', '湖畔的国际会议区，2014 年 APEC 领导人非正式会议在这里举行。', ['高规格会议', '论坛', '会议加团建']],
      richu: ['日出东方酒店', '怀柔区', '雁栖湖畔的酒店，外形像一轮升起的太阳，会议与住宿可一并安排。', ['会议住宿一体', '中小型会议']],
      shougang: ['首钢园', '石景山区', '工业遗存改造的园区，北京冬奥会滑雪大跳台就在这里。', ['论坛', '发布会', '主题活动']],
      cncc: ['国家会议中心', '朝阳区', '奥林匹克公园里的大型会议展览场馆。', ['大型会议', '展览', '论坛']],
      zgc: ['中关村', '海淀区', '科创企业集中，适合科技主题会议和产业参访。', ['科技会议', '产业对标']],
      lize: ['丽泽金融商务区', '丰台区', '新兴金融商务区，离公司近，适合中小型会议和培训。', ['会议', '培训']],
    };
    const NAMES = Object.fromEntries(Object.entries(VENUE).map(([k, v]) => [k, v[0]]));
    const pop = $('.pop', sec);
    let S, opened = false;
    const open = id => {
      const [name, area, desc, fit] = VENUE[id]; pop.hidden = false;
      pop.innerHTML = `<div class="strip"><b>${name}</b><i>会议<br>场地</i></div><div class="body"><button class="x" type="button" aria-label="关闭">×</button><p class="meta">${area} · 场地示例</p><p>${desc}</p><div class="fit">${fit.map(f => `<span>${f}</span>`).join('')}</div><button class="b1" type="button" data-act="inquiry" data-pre="会议会展" data-note="想在：${name}">按这里定制会议方案</button></div>`;
      $('.x', pop).onclick = () => { pop.hidden = true; S.mark(''); };
      S.mark(id); placePop(sec, pop, S, id);
    };
    const build = () => {
      pop.hidden = true;
      S = chapterStage(sec, 'mice', NAMES, ['京华会客厅', '新楼<br>旧笔'], open);
      if (!S) return;
      // 推门见客：两扇隔扇门，滑到这里时向两边打开
      const door = `door-${isMobile() ? 'mobile' : 'desk'}`;
      S.stage.insertAdjacentHTML('beforeend', `<div class="doors" aria-hidden="true"><div class="door l">${art(door, { cls: 'dpic' })}</div><div class="door r">${art(door, { cls: 'dpic' })}</div></div>`);
      if (opened || reduce || !('IntersectionObserver' in window)) { S.stage.classList.add('open'); opened = true; return; }
      const io = new IntersectionObserver(es => { if (es.some(e => e.isIntersecting)) { S.stage.classList.add('open'); opened = true; io.disconnect(); } }, { threshold: 0.45 });
      io.observe(S.stage);
    };
    build();
    const MEETS = [['vWork', '工作会议', '年度工作会 · 座谈会', '会场布置规范简朴，议程、材料、签到一并安排。'], ['vTrain', '培训班', '干部培训 · 业务培训', '课程、教室、食宿统筹，可与现场教学结合。'], ['vForum', '论坛研讨', '行业论坛 · 专题研讨', '嘉宾邀请与接待、议程把控、会场设备按需配置。'], ['vGala', '年会庆典', '年会 · 表彰 · 庆典', '舞台、节目、灯光音响统一统筹。'], ['vLaunch', '发布会', '成果发布 · 新闻发布', '背景板、媒体区、现场直播按需准备。'], ['vExpo', '展览展示', '展会 · 成果展', '展位设计搭建与现场讲解。']];
    $('.meets', sec).innerHTML = MEETS.map(([k, t, s, d]) => `<figure class="meet"><div class="pic">${art('mice-' + k, { alt: t })}</div><figcaption><h4>${t}<small>${s}</small></h4><p>${d}</p></figcaption></figure>`).join('');
    const CAPS = [['方', '方案策划', '按主题、人数和预算出方案与报价。'], ['场', '场地比选', '会场、酒店多家比选，实地踏勘后预订。'], ['接', '接待服务', '报名、接送站、住宿餐饮统一安排。'], ['执', '现场执行', '签到、会场搭建、设备调试与议程把控。'], ['保', '安全保障', '医疗应急、保险和安全预案。']];
    $('.caps', sec).innerHTML = CAPS.map(([s, t, d]) => `<li><span class="seal">${s}</span><b>${t}</b><span>${d}</span></li>`).join('');
    const STUDY = [['修', '干部研修', '课程、师资、地点按单位需求定制，课堂学习与现场教学结合。', '可结合：红色教育现场、专题党课', 'vCadre'], ['标', '产业对标', '走进标杆企业与科创园区实地参访，安排交流座谈。', '常去：中关村、亦庄、首钢园', 'vBench'], ['察', '专题考察', '围绕城市更新、数字化转型等主题组织考察调研。', '常去：城市副中心、首钢园、冬奥场馆', 'vSurvey']];
    $('.studies', sec).innerHTML = STUDY.map(([s, t, d, go, k]) => `<figure class="study"><div class="pic">${art('mice-' + k, { alt: t })}</div><figcaption class="txt"><span class="seal">${s}</span><h4>${t}</h4><p>${d}</p><span class="go">${go}</span></figcaption></figure>`).join('');
    const FLOW = [['前', '会前', ['需求沟通，出方案和预算', '场地、酒店比选与预订', '报名统计与参会通知', '物料设计制作', '交通与接站安排']], ['中', '会中', ['签到与现场引导', '会场搭建与设备调试', '议程与时间把控', '餐饮与茶歇', '摄影摄像', '医疗与应急保障']], ['后', '会后', ['返程送站', '影像资料整理交付', '费用结算与票据', '总结回访']]];
    const md = $('.modal', sec);
    $('.panel', md).innerHTML = `<button class="x" type="button" aria-label="关闭">×</button><h3 id="miceFlowH">服务流程</h3><p class="d">会前、会中、会后，每一项都有专人负责</p><div class="flow">${FLOW.map(([i, h, list]) => `<section><h5><i>${i}</i>${h}</h5><ul>${list.map(t => `<li>${t}</li>`).join('')}</ul></section>`).join('')}</div>`;
    $('[data-open="flow"]', sec).onclick = modal(md);
    return build;
  }

  /* ================= 肆 北京导览图 ================= */
  function initBj() {
    const sec = $('#bj'); if (!sec || !window.JH || !JH.MAP) return;
    const M = JH.MAP, IC = JH.ICONS, mapEl = $('.map', sec), scroll = $('.scroll', sec), pop = $('.pop', sec), itin = $('.itin', sec), tray = $('.tray', sec);
    const S = Object.fromEntries(M.SPOTS.map(s => [s[0], { id: s[0], name: s[1], area: s[2], cat: s[3], themes: s[4], desc: s[5], fit: s[6], at: s[7], ins: s[8] }]));
    const PRE = { red: '红色教育 / 主题党日', gold: '研学考察', green: '企业团建', blue: '会议会展' };
    const ROUTE_PRE = { axis: '研学考察', red: '红色教育 / 主题党日', wall: '研学考察', tech: '研学考察', hills: '企业团建' };
    const state = { cat: 'all', route: '', card: '', picks: new Set() };
    let pins = {}, k = 1, mobile = false;
    function build() {
      mobile = isMobile();
      k = (mobile ? 1100 : mapEl.clientWidth || 1200) / M.W;
      mapEl.innerHTML = art('map-base', { cls: 'map-img', alt: '京华研学导览图（示意图）' }) +
        `<svg class="map-over" viewBox="0 0 ${M.W} ${M.H}" aria-hidden="true">${M.labels(k)}${M.inset(k)}</svg><svg class="map-route" viewBox="0 0 ${M.W} ${M.H}" aria-hidden="true"></svg>` +
        `<div class="cartouche"><b>京华研学导览图</b><small>示意图 · 不按比例</small></div>` +
        `<svg class="compass" viewBox="-32 -32 64 64" aria-hidden="true"><circle r="22" fill="#f6efe0" stroke="#6b5d4b" stroke-width="1.2"/><circle r="16" fill="none" stroke="#6b5d4b" stroke-width=".6"/><path d="M0 -29L5 0L0 26L-5 0Z" fill="#2a2621"/><path d="M0 -29L5 0H-5Z" fill="#b3362b"/><path d="M-26 0L0 4L22 0L0 -4Z" fill="#6b5d4b"/><text x="0" y="-21" text-anchor="middle" font-size="9" fill="#f6efe0" font-family="STKaiti, serif">北</text></svg>`;
      pins = {};
      const place = (sp, x, y, inset) => {
        const L = IC[sp.id](), sc = inset ? 0.42 : 0.72, wScene = (L.box[1] - L.box[0] + 12) * sc;
        const b = document.createElement('button');
        b.className = 'pin'; b.type = 'button'; b.dataset.id = sp.id; b.style.left = pct(x, M.W); b.style.top = pct(y, M.H); b.style.width = pct(wScene, M.W);
        b.innerHTML = JH.svg(L, sc * k) + `<span class="tag"><i style="background:${M.CATS[sp.cat].color}"></i>${sp.name}</span>`;
        b.setAttribute('aria-label', sp.name);
        b.addEventListener('click', e => { e.stopPropagation(); openCard(sp.id); });
        mapEl.appendChild(b); pins[sp.id] = b;
      };
      Object.values(S).sort((a, b) => (!!a.ins - !!b.ins) || ((a.at || a.ins)[1] - (b.at || b.ins)[1])).forEach(sp => sp.at ? place(sp, sp.at[0], sp.at[1], false) : place(sp, sp.ins[0], sp.ins[1], true));
      const home = document.createElement('div'); home.className = 'home'; home.style.left = pct(M.HOME[0], M.W); home.style.top = pct(M.HOME[1], M.H);
      home.innerHTML = JH.svg(IC.home(), 0.62) + '<span class="tag">晟帆 · 从这里出发</span>'; mapEl.appendChild(home);
      const ot = document.createElement('div'); ot.className = 'oldtag'; ot.textContent = '老城 · 见右下放大'; ot.style.left = pct(M.OLD.cx, M.W); ot.style.top = pct(M.OLD.cy - 66, M.H); mapEl.appendChild(ot);
      pop.hidden = true; state.card = '';
      render();
      if (mobile) requestAnimationFrame(() => { const mw = mapEl.clientWidth, mh = mapEl.clientHeight; scroll.scrollLeft = 700 / M.W * mw - scroll.clientWidth / 2; scroll.scrollTop = 620 / M.H * mh - scroll.clientHeight / 2; });
    }
    $('.cats', sec).innerHTML = '<span>类别</span>' + [['all', '全部', '#2a2621'], ...Object.entries(M.CATS).map(([k2, c]) => [k2, c.name, c.color])].map(([id, n, c]) => `<button class="chip${id === 'all' ? ' on' : ''}" type="button" data-cat="${id}"><i style="background:${c}"></i>${n}</button>`).join('');
    $('.routes', sec).innerHTML = '<span>研学线路</span>' + M.ROUTES.map(r => `<button class="chip" type="button" data-route="${r.id}">${r.name}<small>${r.days}</small></button>`).join('');
    $('.cats', sec).addEventListener('click', e => { const b = e.target.closest('[data-cat]'); if (!b) return; state.cat = b.dataset.cat; state.route = ''; render(); });
    $('.routes', sec).addEventListener('click', e => { const b = e.target.closest('[data-route]'); if (!b) return; state.route = state.route === b.dataset.route ? '' : b.dataset.route; render(); if (mobile && state.route) focusRoute(true); });
    function focusRoute(smooth) {
      const R = M.ROUTES.find(r => r.id === state.route); if (!R) return;
      const pts = R.stops.map(id => S[id].at || S[id].ins), xs = pts.map(p => p[0]), ys = pts.map(p => p[1]);
      const cx = (Math.min(...xs) + Math.max(...xs)) / 2 / M.W * mapEl.clientWidth, cy = (Math.min(...ys) + Math.max(...ys)) / 2 / M.H * mapEl.clientHeight - 20;
      scroll.scrollTo({ left: cx - scroll.clientWidth / 2, top: cy - scroll.clientHeight / 2, behavior: smooth ? 'smooth' : 'auto' });
    }
    const ptOf = id => S[id].at || null, insOf = id => S[id].ins || null;
    function drawRoute() {
      const svg = $('.map-route', mapEl), R = M.ROUTES.find(r => r.id === state.route);
      for (const [id, b] of Object.entries(pins)) { const i = R ? R.stops.indexOf(id) : -1, dot = $('.tag i', b); b.classList.toggle('stop', i >= 0); if (i >= 0) { dot.className = 'n'; dot.textContent = '一二三四五六'[i]; } else { dot.className = ''; dot.textContent = ''; } }
      if (!R) { svg.innerHTML = ''; itin.hidden = true; return; }
      const main = R.stops.map(id => ptOf(id) || [M.OLD.cx, M.OLD.cy]), sub = R.stops.map(insOf).filter(Boolean);
      const curve = pts => pts.map((p, i) => i ? `Q${(pts[i - 1][0] + p[0]) / 2 + (p[1] - pts[i - 1][1]) * 0.18} ${(pts[i - 1][1] + p[1]) / 2 - (p[0] - pts[i - 1][0]) * 0.18} ${p[0]} ${p[1]}` : `M${p[0]} ${p[1]}`).join('');
      const sw = 2.6 / k, line = pts => `<path d="${curve(pts)}" fill="none" stroke="#f6efe0" stroke-width="${sw * 2.6}" stroke-linecap="round" opacity=".8"/><path d="${curve(pts)}" fill="none" stroke="#b3362b" stroke-width="${sw}" stroke-dasharray="${sw * 3} ${sw * 2}" stroke-linecap="round"/>`;
      const uniq = []; for (const p of main) if (!uniq.length || uniq[uniq.length - 1] !== p) uniq.push(p);
      svg.innerHTML = (uniq.length > 1 ? line(uniq) : '') + (sub.length > 1 ? line(sub) : '');
      itin.hidden = false;
      itin.innerHTML = `<div><h4>${R.name}</h4><span class="d">${R.days} · ${R.stops.length} 站 · 示例线路，可按需增减</span></div><ol>${R.stops.map((id, i) => `<li><b>${'一二三四五六'[i]}</b>${S[id].name}</li>`).join('')}</ol><button class="b1" type="button" data-act="inquiry" data-pre="${ROUTE_PRE[R.id] || '研学考察'}" data-note="线路：${R.name}（${R.stops.map(id => S[id].name).join('、')}）">按这条线路定制方案</button>`;
    }
    let opening = false;
    function openCard(id) {
      state.card = id; const sp = S[id]; pop.hidden = false;
      pop.innerHTML = `<button class="x" type="button" aria-label="关闭">×</button><div class="pic">${JH.svg(IC[id](), 1.2)}</div><h3>${sp.name}</h3><div class="meta"><i style="background:${M.CATS[sp.cat].color}"></i>${M.CATS[sp.cat].name} · ${sp.area}${sp.themes.length ? ' · ' + sp.themes.join('、') : ''}</div><p>${sp.desc}</p><div class="fit">${sp.fit.map(f => `<span>${f}</span>`).join('')}</div><div class="act"><button class="b2" type="button" data-pick="${id}">${state.picks.has(id) ? '已加入行程' : '加入行程'}</button><button class="b1" type="button" data-act="inquiry" data-pre="${PRE[sp.cat]}" data-note="想去：${sp.name}">按这里定制方案</button></div>`;
      $('.x', pop).onclick = () => { pop.hidden = true; state.card = ''; render(); };
      $('[data-pick]', pop).onclick = () => { state.picks.has(id) ? state.picks.delete(id) : state.picks.add(id); openCard(id); render(); };
      render(); placeCard(id);
      if (mobile && !opening) {
        // 手机：卡片从底部升起，把地图挪到卡片上方，点开的小画留在看得见的地方
        const top0 = $('.leaf', sec).getBoundingClientRect().top;
        if (top0 < 0 || top0 > 104) scrollBy(0, top0 - 64);
        const sr = scroll.getBoundingClientRect(), cr = pop.getBoundingClientRect(), pr = pins[id].getBoundingClientRect();
        const visTop = Math.max(sr.top, 0), visBot = Math.min(sr.bottom, cr.top) - 8;
        scroll.scrollBy({ left: (pr.left + pr.right) / 2 - (sr.left + sr.right) / 2, top: (pr.top + pr.bottom) / 2 - (visTop + visBot) / 2, behavior: 'smooth' });
      }
    }
    // 桌面：卡片放在小画旁边；有线路时，挑挡住线路站点最少的一侧
    function placeCard(id) {
      if (mobile) return;
      const fig = $('.leaf', sec).getBoundingClientRect(), pr = pins[id].getBoundingClientRect(), cw = pop.offsetWidth, ch = pop.offsetHeight, gap = 16, pad = 12;
      const R = M.ROUTES.find(r => r.id === state.route), rel = r => ({ l: r.left - fig.left, t: r.top - fig.top, r: r.right - fig.left, b: r.bottom - fig.top + 26 });
      const keep = [rel(pr), ...(R ? R.stops.filter(s => s !== id).map(s => rel(pins[s].getBoundingClientRect())) : [])];
      const cx = pr.left - fig.left + pr.width / 2, top0 = pr.top - fig.top;
      const clampL = l => Math.max(pad, Math.min(fig.width - cw - pad, l)), clampT = t => Math.max(pad, Math.min(fig.height - ch - pad, t));
      const cands = [[pr.right - fig.left + gap, top0 - 40], [pr.left - fig.left - cw - gap, top0 - 40], [cx - cw / 2, pr.bottom - fig.top + 34], [cx - cw / 2, top0 - ch - gap],
        [pad, pad], [fig.width - cw - pad, pad], [pad, fig.height - ch - pad], [fig.width - cw - pad, fig.height - ch - pad]].map(([l, t]) => [clampL(l), clampT(t)]);
      const over = (l, t) => keep.reduce((a, q) => a + Math.max(0, Math.min(l + cw, q.r) - Math.max(l, q.l)) * Math.max(0, Math.min(t + ch, q.b) - Math.max(t, q.t)), 0);
      let best = cands[0], bs = Infinity;
      cands.forEach(([l, t], i) => { const sc = over(l, t) * 20 + Math.hypot(l + cw / 2 - cx, t + ch / 2 - top0) + i * 4; if (sc < bs) { bs = sc; best = [l, t]; } });
      pop.style.left = best[0] + 'px'; pop.style.top = best[1] + 'px';
    }
    function render() {
      $$('.cats .chip', sec).forEach(b => b.classList.toggle('on', !state.route && b.dataset.cat === state.cat));
      $$('.routes .chip', sec).forEach(b => b.classList.toggle('on', b.dataset.route === state.route));
      const R = M.ROUTES.find(r => r.id === state.route);
      for (const [id, b] of Object.entries(pins)) {
        b.classList.toggle('dim', R ? !R.stops.includes(id) : (state.cat !== 'all' && S[id].cat !== state.cat));
        b.classList.toggle('picked', state.picks.has(id)); b.classList.toggle('on', state.card === id);
      }
      drawRoute();
      const ids = [...state.picks], list = ids.map(id => S[id].name);
      tray.innerHTML = `<b>我的行程</b><span class="list">${list.length ? list.join('、') : '在图上点小画，把想去的地方加进来'}</span>${list.length ? `<button class="b1" type="button" data-act="inquiry" data-pre="${PRE[S[ids[0]].cat]}" data-note="想去：${list.join('、')}">按这 ${list.length} 处定制方案</button>` : ''}`;
    }
    build();
    let rt; addEventListener('resize', () => { clearTimeout(rt); rt = setTimeout(() => { if (isMobile() !== mobile) build(); else { drawRoute(); if (state.card) placeCard(state.card); } }, 150); });
  }

  /* ================= 启动：先读清单，再铺各章的画 ================= */
  const rebuilds = [];
  fetch(ART + 'manifest.json', { cache: 'no-cache' }).then(r => r.json()).catch(() => ({ version: 0, items: {} })).then(m => {
    MAN = m;
    for (const f of [initRed, initTeam, initMice, initCases]) { try { const b = f(); if (b) rebuilds.push(b); } catch (e) { console.error(e); } }
    try { initBj(); } catch (e) { console.error(e); }
    let wasMobile = isMobile(), rt;
    addEventListener('resize', () => { clearTimeout(rt); rt = setTimeout(() => { if (isMobile() !== wasMobile) { wasMobile = isMobile(); rebuilds.forEach(b => b()); } }, 200); });
  });


  /* ================= 伍 资质：装裱成册页，滑到时展开 ================= */
  $('#certAlbum').innerHTML = `<ol class="album-row">${CERTS.map(([id, img, name, level, kv], i) => {
    const exp = (kv.find(([k]) => k === '有效期') || [])[1];
    return `<li style="--i:${i}"><button type="button" class="mount" data-cert="${id}" aria-label="查看${esc(name)}原件">
      <span class="mount-slip">${yin(({ mice: '一级', travel: '许可' })[id] || '认证', 'yin-s')}<b>${name}</b></span>
      <span class="mount-img"><img src="assets/img/cert/s/${img}.jpg" alt="" loading="lazy"></span>
      <span class="mount-foot">${exp ? '有效期至 ' + exp.split('–')[1].trim() : kv[0][0] + ' · ' + kv[0][1]}</span>
    </button></li>`;
  }).join('')}</ol>`;
  $('#guards').innerHTML = GUARDS.map(([c, t, d]) => `<li>${yin(c)}<div><b>${t}</b><span>${d}</span></div></li>`).join('');
  (function strip() {
    const one = GC_STRIP.map(k => `<span class="gc-cert"><img src="assets/img/cert/s/gc-${k}.jpg" alt="" loading="lazy"></span>`).join('');
    $('#gcStrip').innerHTML = `<div class="gc-track">${one}${one}</div>`;
  })();

  /* ================= 陆 项目档案：卷宗 ================= */
  // 现场照片到位前，照片位先用各章画里的场景（灰度）占位；填了 photo 就换成照片
  function initCases() {
    const build = () => {
      const lay = isMobile() ? 'mobile' : 'desk', tm = MAN.items[`team-${lay}`];
      const [x0, y0, x1, y1] = lay === 'desk' ? [770, 455, 1270, 768] : [4, 455, 256, 613];   // 团建画里的拓展营地，16:10
      const PH = {
        red: art('red-vField'),
        lv: tm ? art(`team-${lay}`, { cls: 'pic', style: `width:${(tm.meta.W / (x1 - x0) * 100).toFixed(2)}%;left:${(-x0 / (x1 - x0) * 100).toFixed(2)}%;top:${(-y0 / (y1 - y0) * 100).toFixed(2)}%` }) : '',
        qing: art('mice-vWork'),
      };
      $('#casesList').innerHTML = CASES.map(c => `<li class="dos${c.placeholder ? ' ph' : ''}" data-ac="${c.kind}">
    <div class="dos-tab"><span>${c.tag}</span><b>${c.year}</b></div>
    <div class="dos-photo">${c.photo ? `<img src="${c.photo}" alt="${esc(c.title)}" loading="lazy">` : `<div class="ph">${PH[c.kind] || ''}<em>照片待补</em></div>`}</div>
    <div class="dos-body"><h3>${esc(c.title)}</h3><dl>
      <div><dt>时间</dt><dd>${esc(c.when)}</dd></div><div><dt>地点</dt><dd>${esc(c.where)}</dd></div>
      <div><dt>规模</dt><dd>${esc(c.size)}</dd></div><div><dt>服务</dt><dd>${esc(c.what)}</dd></div></dl></div>
    ${yin('档', 'yin-r')}</li>`).join('');
    };
    build();
    return build;
  }

  /* ================= 柒 名帖 ================= */
  const cardEl = $('#card'), card = { ...cardEl.dataset };
  const fmt = m => m.replace(/^(\d{3})(\d{4})(\d{4})$/, '$1 $2 $3');
  cardEl.innerHTML = `
    <div class="tie">
      <div class="tie-brand"><img src="assets/img/sf-mark.png" alt="" width="22" height="31"><div><b>北京晟帆国际旅行社有限公司</b><span>贵昌集团旗下 · 会务文旅板块</span></div></div>
      <div class="tie-name"><div class="tie-photo" aria-hidden="true"><span>照片</span></div><div><h3>${esc(card.name)}</h3><p>${esc(card.title)}</p></div>${yin('晟帆', 'yin-n')}</div>
      <ul class="tie-lines">
        <li><svg class="i"><use href="#i-phone"/></svg><div><b>${card.mobile ? fmt(card.mobile) : '1XX XXXX XXXX'}</b><span>手机 · 微信同号${card.mobile ? '' : '（待提供）'}</span></div></li>
        <li><svg class="i"><use href="#i-tel"/></svg><div><b>${card.tel}</b><span>公司座机</span></div></li>
        <li><svg class="i"><use href="#i-mail"/></svg><div><b>${card.email}</b><span>邮箱</span></div></li>
        <li><svg class="i"><use href="#i-pin"/></svg><div><b>丰台区花乡六圈西路 8 号院</b><span>新华双创园 B 座 3 层 307 室</span></div></li>
      </ul>
      <div class="tie-cta">
        <button class="b-primary" type="button" data-act="call"><svg class="i"><use href="#i-phone"/></svg>打电话</button>
        <button class="b-ghost" type="button" data-act="wechat"><svg class="i"><use href="#i-wechat"/></svg>加微信</button>
        <button class="b-ghost" type="button" data-act="vcard"><svg class="i"><use href="#i-card"/></svg>存名片</button>
      </div>
    </div>
    <div class="tie-side">
      <div class="qr ph" aria-label="微信二维码（待提供）"><span>微信二维码<br>待提供</span></div>
      <p>长按识别，加我微信</p>
      <button class="b-ghost sm" type="button" data-act="inquiry">获取活动方案</button>
    </div>`;

  /* ================= 弹层 / 提示 / 动作 ================= */
  const sheet = $('#sheet'), sheetBody = $('#sheetBody');
  function openSheet(html) {
    sheetBody.innerHTML = html;
    if (typeof sheet.showModal === 'function') { if (!sheet.open) sheet.showModal(); } else sheet.setAttribute('open', '');
    sheet.scrollTop = 0;
  }
  sheet.addEventListener('click', e => { if (e.target === sheet) sheet.close(); });
  let toastT;
  function toast(msg) {
    const t = $('#toast'); t.textContent = msg; t.classList.add('show');
    clearTimeout(toastT); toastT = setTimeout(() => t.classList.remove('show'), 2400);
  }
  async function copy(text, label) {
    if (!text) { toast(label + '待提供'); return; }
    try { await navigator.clipboard.writeText(text); }
    catch {
      const ta = document.createElement('textarea'); ta.value = text; ta.style.position = 'fixed'; ta.style.opacity = '0';
      document.body.appendChild(ta); ta.select(); document.execCommand('copy'); ta.remove();
    }
    toast('已复制' + label);
  }
  function certSheet(id) {
    const c = CERTS.find(x => x[0] === id); if (!c) return;
    const [, img, name, level, kv, verify] = c;
    openSheet(`<h3>${name}</h3><p class="sub">${level} · 北京晟帆国际旅行社有限公司</p>
      <figure class="cert-big"><img src="assets/img/cert/${img}.jpg" alt="${esc(name)}原件"></figure>
      <dl class="kv">${kv.map(([k, v]) => `<div><dt>${k}</dt><dd>${v}</dd></div>`).join('')}</dl>
      ${verify ? `<a class="b-ghost" href="${verify[1]}" target="_blank" rel="noopener">去官方平台核验 · ${verify[0]}</a>` : ''}
      <p class="hint">图片来自公司宣传册，正式上线前换成清晰扫描件。</p>`);
  }
  function vcard() {
    const n = card.name || '';
    return ['BEGIN:VCARD', 'VERSION:3.0', `N:${n.slice(0, 1)};${n.slice(1)};;;`, `FN:${n}`, 'ORG:北京晟帆国际旅行社有限公司', `TITLE:${card.title || ''}`,
      card.mobile && `TEL;TYPE=CELL:${card.mobile}`, `TEL;TYPE=WORK:${card.tel}`, `EMAIL;TYPE=INTERNET:${card.email}`,
      `ADR;TYPE=WORK:;;${card.address};;;;`, 'NOTE:贵昌集团旗下 · 红色教育 · 党政团建 · 企业团建 · 会务研学',
      `URL:${location.href.split('#')[0]}`, 'END:VCARD'].filter(Boolean).join('\r\n');
  }
  const ACTIONS = {
    call() { location.href = 'tel:' + (card.mobile || card.tel.replace(/-/g, '')); },
    wechat() {
      openSheet(`<h3>加我微信</h3><p class="sub">长按识别二维码，或复制微信号搜索添加</p>
        <div class="qr ph big"><span>微信二维码<br>待提供</span></div>
        <dl class="kv"><div><dt>微信号</dt><dd>${card.wechat || '待提供'}</dd></div><div><dt>座机</dt><dd>${card.tel}</dd></div></dl>
        <button class="b-ghost" type="button" data-act="copy-wechat">复制微信号</button>`);
    },
    'copy-wechat'() { copy(card.wechat, '微信号'); },
    'copy-email'() { copy(card.email, '邮箱'); },
    'copy-link'() { copy(location.href.split('#')[0], '链接'); },
    vcard() {
      if (inWeChat) {
        openSheet(`<h3>保存到通讯录</h3><p class="sub">微信里不能直接保存文件</p><p class="txt">请点右上角「···」，选「在浏览器打开」，再点一次「存名片」。</p><button class="b-ghost" type="button" data-act="copy-link">复制本页链接</button>`);
        return;
      }
      const a = document.createElement('a');
      a.href = URL.createObjectURL(new Blob([vcard()], { type: 'text/vcard;charset=utf-8' }));
      a.download = (card.name || '名片') + '.vcf';
      document.body.appendChild(a); a.click(); a.remove();
      setTimeout(() => URL.revokeObjectURL(a.href), 1000);
      toast('已生成名片文件');
    },
    docs() {
      openSheet(`<h3>索取资质文件</h3><p class="sub">比价、供应商入库、合同签订时常用</p>
        <ul class="doc-list">${['营业执照（副本）', '旅行社业务经营许可证', '会务会展服务企业资质证书（一级）', 'ISO 9001 / 14001 / 45001 认证证书', '旅行社责任保险保单'].map(t => `<li>${yin('件', 'yin-s')}${t}</li>`).join('')}</ul>
        <p class="txt">可提供加盖公章的复印件或电子版，请联系对接人，说明单位名称和用途。</p>
        <div class="row2"><button class="b-primary" type="button" data-act="call"><svg class="i"><use href="#i-phone"/></svg>打电话</button><button class="b-ghost" type="button" data-act="copy-email"><svg class="i"><use href="#i-mail"/></svg>复制邮箱</button></div>`);
    },
  };
  window.SF = { $, $$, esc, openSheet, toast, copy, inWeChat, desktop, reduce, ACTIONS, yin };
  document.addEventListener('click', e => {
    const c = e.target.closest('[data-cert]'); if (c) { certSheet(c.dataset.cert); return; }
    const act = e.target.closest('[data-act]');
    if (act && ACTIONS[act.dataset.act]) { e.preventDefault(); ACTIONS[act.dataset.act](act); }
  });

  /* ================= 进场 / 导航 / 联系条 ================= */
  const io = new IntersectionObserver(es => es.forEach(e => { if (e.isIntersecting) { e.target.classList.add('in'); io.unobserve(e.target); } }), { threshold: 0.14, rootMargin: '0px 0px -6% 0px' });
  $$('[data-rv]').forEach(el => io.observe(el));
  const links = $$('.nav-links a');
  const spy = new IntersectionObserver(es => es.forEach(e => {
    if (!e.isIntersecting) return;
    links.forEach(a => a.classList.toggle('on', a.getAttribute('href') === '#' + e.target.id));
  }), { rootMargin: '-45% 0px -50% 0px' });
  $$('main > section[id]').forEach(s => spy.observe(s));
  const nav = $('#nav'), dock = $('#dock'), juan = $('.jhero');
  let ticking = false;
  function onScroll() {
    if (ticking) return; ticking = true;
    requestAnimationFrame(() => {
      const past = !juan || scrollY > juan.offsetTop + juan.offsetHeight - innerHeight * 0.6;
      nav.classList.toggle('solid', past);
      dock.classList.toggle('show', past);
      ticking = false;
    });
  }
  addEventListener('scroll', onScroll, { passive: true });
  onScroll();

})();
