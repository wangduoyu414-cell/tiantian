/* 晟帆国旅主页 · 页面逻辑。内容数据都在本文件顶部，改字只改这里。 */
(function () {
  'use strict';
  const $ = (s, r = document) => r.querySelector(s), $$ = (s, r = document) => [...r.querySelectorAll(s)];
  const inWeChat = /MicroMessenger/i.test(navigator.userAgent);
  const desktop = matchMedia('(min-width: 900px)');
  const reduce = matchMedia('(prefers-reduced-motion: reduce)').matches;
  const esc = s => String(s).replace(/[&<>"]/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' })[c]);

  /* ================= 内容 ================= */
  const HERO_CARDS = [
    ['red', '红色教育', '主题党日 · 现场教学 · 党性教育', '#red'],
    ['gov', '党政团建', '有主题、有意义、规范节俭', '#team', 'gov'],
    ['ent', '企业团建', '拓展 · 徒步 · 年会 · 研学团建', '#team', 'ent'],
    ['mice', '会务研学', '会议会展 · 干部研修 · 标杆考察', '#mice'],
  ];

  const STATIONS = [
    ['honglou', '北大红楼', '东城区', '中国共产党早期北京革命活动旧址'],
    ['siheyuan', '李大钊故居', '西城区', '李大钊在北京居住时间最长的寓所'],
    ['xiangshan', '香山革命纪念地', '海淀区', '1949 年中共中央在此指挥渡江战役、筹建新中国'],
    ['lugou', '卢沟桥 · 抗战纪念馆', '丰台区', '全民族抗战爆发地，就在公司所在的丰台'],
    ['dangshi', '中国共产党历史展览馆', '朝阳区', '全面展示党的百年奋斗历程'],
  ];

  const RED_ITIN = {
    tag: '现场教学一日 · 示例', title: '红色足迹 · 党性教育',
    meta: [['1', '天'], ['20–200', '人'], ['', '机关党支部、国企党组织']],
    rows: [
      ['08:30', '集合出发', '统一乘车，负责人随车说明当天安排', true],
      ['09:30', '北大红楼', '专业讲解，参观主题展览'],
      ['11:00', '主题交流', '结合本单位工作开展研讨'],
      ['12:00', '午餐', '按标准安排工作餐'],
      ['13:30', '香山革命纪念地', '现场教学，重温入党誓词', true],
      ['16:00', '学习小结', '整理学习材料和影像资料'],
      ['16:30', '返程', '安全送达各集合点'],
    ],
    foot: ['讲解', '车辆', '保险', '餐食', '随团负责人'],
  };

  const RED_FORMS = [
    ['ztdr', '主题党日', '按月度学习主题策划'],
    ['xcjx', '现场教学', '纪念馆、旧址实地讲解'],
    ['ztdk', '专题党课', '邀请讲师或配合单位自讲'],
    ['qjjx', '情景教学', '情景再现，沉浸体验'],
    ['cwsy', '重温入党誓词', '仪式规范，流程有序'],
    ['jlyt', '交流研讨', '把学习体会落到工作上'],
  ];

  const TEAM = {
    gov: {
      note: '主题鲜明、规范节俭，可与主题党日、工会活动结合安排。',
      items: [
        ['dingxiang', '红色定向', '以红色地标为点位分组寻访，边走边学。', ['半天', '30–200 人']],
        ['gonghui', '主题党日 + 工会活动', '上午集中学习，下午集体活动，学习与凝聚兼顾。', ['一天', '30–150 人']],
        ['laodong', '劳动教育', '走进京郊农场参与农事，体会劳动的价值。', ['半天至一天', '春夏秋']],
        ['feiyi', '非遗文化体验', '风筝、剪纸、景泰蓝等手作，了解北京传统文化。', ['半天', '20–80 人']],
        ['dengshan', '健步走 · 登山', '奥森、香山、长城等线路，强健体魄。', ['半天', '春秋季']],
        ['dushu', '读书分享会', '选定书目，组织分享研讨，形成学习成果。', ['半天', '20–60 人']],
      ],
    },
    ent: {
      note: '重在凝聚与体验，可按人数、预算和季节灵活组合。',
      items: [
        ['tuozhan', '拓展训练', '专业教练带队，破冰、协作、挑战层层递进。', ['一天', '20–300 人']],
        ['tubu', '徒步穿越', '京西古道、长城、京郊山野线路，全程保障。', ['一天', '春秋季']],
        ['yundong', '趣味运动会', '场地、器材、裁判、奖品一站安排。', ['半天', '50–500 人']],
        ['gongchuang', '团队共创', '引导师带领目标共识与问题共创，团建也出成果。', ['半天', '20–80 人']],
        ['nianhui', '年会庆典', '策划、舞美、节目、主持、摄像全流程执行。', ['半天至一天', '100 人以上']],
        ['yanxue', '研学团建', '走进标杆企业与科创园区，学习与团建结合。', ['一天', '20–100 人']],
      ],
    },
  };

  /* ================= 首屏 ================= */
  $('#artDefs').innerHTML = ART.defs();
  $('#pano').innerHTML = ART.panorama();
  $('#heroCards').innerHTML = HERO_CARDS.map(([ic, t, d, href, tab]) =>
    `<li><a href="${href}"${tab ? ` data-tab-go="${tab}"` : ''}>${ICON(ic)}<b>${t}</b><span>${d}</span></a></li>`).join('');

  const pano = $('.pano');
  function startPano() {
    if (reduce) return;
    pano.classList.add('go');
    // 兜底：页面在后台或系统暂停了动画时，4 秒后直接显示长卷，首屏不会空着
    setTimeout(() => pano.classList.add('done'), 4000);
    setTimeout(() => {
      const track = $('#pano');
      if (track.scrollWidth > pano.clientWidth + 8) pano.classList.add('panning');
    }, 3000);
  }
  if (document.readyState === 'complete') startPano(); else addEventListener('load', startPano, { once: true });
  let panT;
  pano.addEventListener('pointerdown', () => { pano.classList.add('paused'); clearTimeout(panT); panT = setTimeout(() => pano.classList.remove('paused'), 4000); });

  // 单独一处地标的小图
  function solo(name, pad = 4, cls = '') {
    const L = ART.LANDMARKS[name](), [l, r, b, t] = L.box;
    return `<svg class="art ${cls}" viewBox="${l - pad} ${-t - pad} ${r - l + pad * 2} ${t - b + pad * 2}" preserveAspectRatio="xMidYMax meet" aria-hidden="true"><g transform="scale(1 -1)">${L.g}</g></svg>`;
  }

  /* ================= 01 红色教育 ================= */
  $('#redRoute').innerHTML = `<ol class="stations">${STATIONS.map(([k, n, area, note]) =>
    `<li class="st"><figure class="st-vig">${solo(k, 6)}</figure><i class="st-dot" aria-hidden="true"></i><div class="st-txt"><b>${n}</b><em>${area}</em><span>${note}</span></div></li>`).join('')}</ol>`;

  function itinHTML(it) {
    return `<header class="itin-head"><p class="itin-tag">${it.tag}</p><h3>${it.title}</h3>
      <ul class="itin-meta">${it.meta.map(([n, u]) => `<li>${n ? `<b>${n}</b>` : ''}${u}</li>`).join('')}</ul></header>
      <ol class="itin-list">${it.rows.map(([t, b, s, key]) => `<li${key ? ' class="key"' : ''}><time>${t}</time><i></i><div><b>${b}</b><span>${s}</span></div></li>`).join('')}</ol>
      <footer class="itin-foot">含${it.foot.map(f => `<em>${f}</em>`).join('')}</footer>`;
  }
  $('#redItin').innerHTML = itinHTML(RED_ITIN);
  $('#redForms').innerHTML = RED_FORMS.map(([ic, t, d]) => `<li>${ICON(ic)}<div><b>${t}</b><span>${d}</span></div></li>`).join('');

  /* ================= 02 团建 ================= */
  // 汇流图：手机上画得更窄更高，线条和端点才看得清
  function flow() {
    const narrow = !desktop.matches, W = narrow ? 420 : 900, H = narrow ? 150 : 150, cy = H / 2, n = narrow ? 9 : 11, x1 = narrow ? 270 : 600, x2 = W - 52;
    let lines = '', ppl = '';
    for (let i = 0; i < n; i++) {
      const y = 12 + i * (H - 24) / (n - 1), w = Math.sin(i * 1.7) * (narrow ? 8 : 14);
      lines += `<path class="fl" pathLength="1" d="M18 ${y.toFixed(1)}C${(x1 * 0.34).toFixed(0)} ${(y + w).toFixed(1)} ${(x1 * 0.6).toFixed(0)} ${(cy + (y - cy) * 0.32).toFixed(1)} ${x1} ${cy}"/>`;
      ppl += `<circle class="fl-p" cx="14" cy="${y.toFixed(1)}" r="${narrow ? 4.6 : 4.2}"/>`;
    }
    $('#teamFlow').innerHTML = `<svg viewBox="0 0 ${W} ${H}" role="img" aria-label="来自不同岗位的人，拧成一股绳">${lines}<path class="fl-main" pathLength="1" d="M${x1} ${cy}H${x2}"/>${ppl}
      <g transform="translate(${x2 + 4} ${cy - 17}) scale(1.6)"><path class="fl-end" d="M10.6 1.8c3.4 3.6 4.4 8.6 2.2 13.4H5.4C9 11.4 10.8 6.8 10.6 1.8Z"/><path class="fl-end" d="M3 16.6c3.6-1 9.6-1 14 0v1.6c-4.4-1-10.4-1-14 0Z"/></g></svg>`;
  }
  flow();
  desktop.addEventListener('change', flow);

  const teamMenu = $('#teamMenu'), teamNote = $('#teamNote');
  function setTeam(k) {
    const T = TEAM[k];
    $$('.tabs .tab').forEach(b => { const on = b.dataset.tab === k; b.classList.toggle('on', on); b.setAttribute('aria-selected', on); });
    teamNote.textContent = T.note;
    teamMenu.innerHTML = T.items.map(([ic, t, d, meta]) => `<li>${ICON(ic)}<div><b>${t}</b><p>${d}</p></div><span class="meta">${meta.map(m => `<i>${m}</i>`).join('')}</span></li>`).join('');
  }
  setTeam('gov');
  $$('.tabs .tab').forEach(b => b.addEventListener('click', () => setTeam(b.dataset.tab)));

  /* ================= 弹层 / 提示 / 动作 ================= */
  const sheet = $('#sheet'), sheetBody = $('#sheetBody');
  function openSheet(html) {
    sheetBody.innerHTML = html;
    if (typeof sheet.showModal === 'function') sheet.showModal(); else sheet.setAttribute('open', '');
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
  window.SF = { $, $$, esc, solo, openSheet, toast, copy, itinHTML, inWeChat, desktop, reduce, ACTIONS: {} };

  document.addEventListener('click', e => {
    const go = e.target.closest('[data-tab-go]');
    if (go) setTeam(go.dataset.tabGo);
    const act = e.target.closest('[data-act]');
    if (act && SF.ACTIONS[act.dataset.act]) { e.preventDefault(); SF.ACTIONS[act.dataset.act](act); }
  });

  /* ================= 进场 / 导航 / 联系条 ================= */
  function observe() {
    const io = new IntersectionObserver(es => es.forEach(e => {
      if (!e.isIntersecting) return;
      e.target.classList.add('in'); io.unobserve(e.target);
    }), { threshold: .12, rootMargin: '0px 0px -6% 0px' });
    $$('[data-rv]:not(.in)').forEach(el => io.observe(el));
  }
  SF.observe = observe;
  const links = $$('.nav-links a');
  const spy = new IntersectionObserver(es => es.forEach(e => {
    if (!e.isIntersecting) return;
    links.forEach(a => a.classList.toggle('on', a.getAttribute('href') === '#' + e.target.id));
  }), { rootMargin: '-45% 0px -50% 0px' });
  $$('main > section[id]').forEach(s => spy.observe(s));
  const nav = $('#nav'), dock = $('#dock'), hero = $('.hero');
  let ticking = false;
  function onScroll() {
    if (ticking) return; ticking = true;
    requestAnimationFrame(() => {
      const y = scrollY;
      nav.classList.toggle('solid', y > 12);
      dock.classList.toggle('show', y > hero.offsetHeight * 0.7);
      ticking = false;
    });
  }
  addEventListener('scroll', onScroll, { passive: true });
  onScroll();
  // 其余章节由 site2.js 渲染，渲染完再统一挂进场动画
  document.addEventListener('DOMContentLoaded', () => setTimeout(observe, 0));
})();
