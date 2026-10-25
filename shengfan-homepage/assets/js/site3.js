/* 晟帆国旅主页（第三版） · 首屏之后各章：红色教育、团建、会务研学、北京资源、资质与保障、项目档案、名片。
   内容数据都在本文件顶部，改字只改这里。插图见 vignettes.js，首屏长卷见 hero.js。 */
(function () {
  'use strict';
  const $ = (s, r = document) => r.querySelector(s), $$ = (s, r = document) => [...r.querySelectorAll(s)];
  const inWeChat = /MicroMessenger/i.test(navigator.userAgent);
  const desktop = matchMedia('(min-width: 900px)');
  const reduce = matchMedia('(prefers-reduced-motion: reduce)').matches;
  const esc = s => String(s).replace(/[&<>"]/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' })[c]);
  const N = v => Math.round(v * 10) / 10;
  const V = window.VIGNETTE;
  // 印章式图标：一方小印，一个字
  const yin = (ch, kind = '') => `<i class="yin${ch.length > 1 ? ' yin-2' : ''}${kind ? ' ' + kind : ''}" aria-hidden="true">${ch}</i>`;

  /* ================= 内容 ================= */
  const RED_ITIN = {
    tag: '现场教学一日 · 示例', title: '红色足迹 · 党性教育',
    meta: [['1', '天'], ['20–200', '人'], ['', '机关党支部、国企党组织']],
    rows: [
      ['08:30', '集合出发', '统一乘车，负责人随车说明当天安排'],
      ['09:30', '北大红楼', '专业讲解，参观主题展览', 'honglou'],
      ['11:00', '主题交流', '结合本单位工作开展研讨'],
      ['12:00', '午餐', '按标准安排工作餐'],
      ['13:30', '香山革命纪念地', '现场教学，重温入党誓词', 'xiangshan'],
      ['16:00', '学习小结', '整理学习材料和影像资料'],
      ['16:30', '返程', '安全送达各集合点'],
    ],
    foot: ['讲解', '车辆', '保险', '餐食', '随团负责人'],
  };
  const RED_FORMS = [
    ['日', '主题党日', '按月度学习主题策划'],
    ['学', '现场教学', '纪念馆、旧址实地讲解'],
    ['课', '专题党课', '邀请讲师或配合单位自讲'],
    ['景', '情景教学', '情景再现，沉浸体验'],
    ['誓', '重温入党誓词', '仪式规范，流程有序'],
    ['研', '交流研讨', '把学习体会落到工作上'],
  ];
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
  const CHECKS = [
    ['会前', ['需求沟通，出方案和预算', '场地、酒店比选与预订', '报名统计与参会通知', '物料设计制作', '交通与接站安排']],
    ['会中', ['签到与现场引导', '会场搭建与设备调试', '议程与时间把控', '餐饮与茶歇', '摄影摄像', '医疗与应急保障']],
    ['会后', ['返程送站', '影像资料整理交付', '费用结算与票据', '总结回访']],
  ];
  const STUDY = [
    ['修', '干部研修', '依托高校资源开展非学历培训，课程、师资、地点按需定制。'],
    ['标', '产业对标', '走进领先企业实地参访，安排交流座谈与案例研讨。'],
    ['察', '专题考察', '围绕数字化转型、专精特新等主题组织调研考察。'],
  ];
  // 北京资源图：示意坐标（720×540），不按实际比例
  const BJ_LINES = {
    red: { name: '红色教育', color: '#b3362b', path: ['xiangshan', 'lidazhao', 'honglou', 'dangshi'], branch: ['lugou', 'lidazhao'] },
    gold: { name: '文化研学', color: '#b0842e', path: ['badaling', 'yiheyuan', 'gugong', 'tiantan'] },
    green: { name: '团建场地', color: '#4f8a52', path: ['fangshan', 'mentougou', 'badaling', 'yanqi', 'miyun', 'pinggu'] },
    blue: { name: '会议场地', color: '#2f5f86', path: ['yanqi', 'zgc', 'shougang', 'lize', 'yizhuang', 'cbd'] },
  };
  const BJ = {
    xiangshan: [150, 214, '香山革命纪念地', '海淀区', '1949 年中共中央在此指挥渡江战役、筹建新中国。', ['主题党日', '现场教学'], 'l', 'shuangqing'],
    lidazhao: [318, 300, '李大钊故居', '西城区', '李大钊在北京居住时间最长的寓所。', ['现场教学', '党史学习'], 'bl'],
    honglou: [404, 262, '北大红楼', '东城区', '中国共产党早期北京革命活动旧址。', ['主题党日', '现场教学'], 'r', 'honglou'],
    dangshi: [470, 168, '中国共产党历史展览馆', '朝阳区', '全面展示党的百年奋斗历程。', ['党史学习', '主题党日'], 'r'],
    lugou: [232, 412, '卢沟桥 · 抗战纪念馆', '丰台区', '全民族抗战爆发地，就在公司所在的丰台。', ['现场教学', '爱国主义教育'], 'b', 'lugou'],
    badaling: [118, 92, '延庆 · 八达岭长城', '延庆区', '明长城代表段落，登城研学、团队登山都合适。', ['文化研学', '登山'], 'tr'],
    yiheyuan: [236, 168, '颐和园', '海淀区', '皇家园林，适合文化研学与健步走。', ['文化研学', '健步走'], 'r', 'foxiangge'],
    gugong: [378, 318, '故宫', '东城区', '明清皇宫，北京中轴线核心。', ['文化研学'], 'l', 'jiaolou'],
    tiantan: [420, 400, '天坛', '东城区', '明清皇家祭天之所，北京中轴线南段。', ['文化研学'], 'r', 'qinian'],
    fangshan: [112, 488, '房山 · 十渡', '房山区', '山水峡谷，适合拓展与户外活动。', ['拓展', '户外'], 'r'],
    mentougou: [70, 300, '门头沟 · 京西古道', '门头沟区', '古道徒步线路，适合团队徒步。', ['徒步'], 'r'],
    yanqi: [356, 64, '怀柔 · 雁栖湖', '怀柔区', '湖区山林，会议与团建都适合。', ['会议', '团建'], 't'],
    miyun: [530, 78, '密云', '密云区', '山区徒步与农事体验。', ['徒步', '劳动教育'], 't'],
    pinggu: [650, 168, '平谷', '平谷区', '山野线路与果园农事体验。', ['劳动教育', '健步走'], 'b'],
    zgc: [280, 228, '中关村', '海淀区', '科创企业集中，适合产业对标与会议。', ['产业对标', '会议'], 'l'],
    shougang: [176, 330, '首钢园', '石景山区', '工业遗存改造的园区，适合论坛与发布会。', ['论坛', '发布会'], 'l'],
    lize: [300, 386, '丰台 · 丽泽', '丰台区', '金融商务区，离公司近，适合中小型会议。', ['会议', '培训'], 'r'],
    yizhuang: [512, 452, '亦庄', '经开区', '高新产业集中，适合产业参访与会议。', ['产业对标', '会议'], 'r'],
    cbd: [540, 318, 'CBD', '朝阳区', '商务酒店集中，适合各类会议。', ['会议', '年会'], 'r'],
  };
  const HOME = [300, 482];
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
  // 项目档案：真实项目到位后，把 placeholder 改成 false 并填好字段
  const CASES = [
    { kind: 'red', tag: '红色教育', year: '待补充', title: '项目名称待补充', when: '待补充', where: '待补充', size: '待补充', what: '待补充', placeholder: true },
    { kind: 'lv', tag: '团建', year: '待补充', title: '项目名称待补充', when: '待补充', where: '待补充', size: '待补充', what: '待补充', placeholder: true },
    { kind: 'qing', tag: '会务', year: '待补充', title: '项目名称待补充', when: '待补充', where: '待补充', size: '待补充', what: '待补充', placeholder: true },
  ];

  /* ================= 画面底子：插图渐变、绢纹 ================= */
  $('#defs').insertAdjacentHTML('beforeend', V.defs());
  const silkBg = () => { if (window.SILK_URL) document.body.style.setProperty('--silk-img', `url(${window.SILK_URL})`); };
  silkBg();

  // 插图放进一个 .sc 容器，跟首屏同一套四时设色
  const season = () => (window.JUAN && JUAN.season && JUAN.season()) || 'autumn';
  const scWrap = (inner, vb, cls = '') => `<div class="sc v-sc ${cls}" data-season="${season()}"><svg class="a2 v-svg" viewBox="${vb}" preserveAspectRatio="xMidYMid meet" aria-hidden="true">${inner}</svg></div>`;
  const stage = $('#stage');
  if (stage && 'MutationObserver' in window) new MutationObserver(() => $$('.v-sc').forEach(el => { el.dataset.season = stage.dataset.season; })).observe(stage, { attributes: true, attributeFilter: ['data-season'] });

  /* ================= 壹 红色教育 ================= */
  const red = V.redScroll();
  $('#redPan').innerHTML = scWrap(red.svg, `0 0 ${red.W} ${red.H}`) +
    red.spots.map(s => `<button type="button" class="stamp-pin" data-spot="${s.id}" style="left:${N(s.x / red.W * 100)}%;top:${N(s.y / red.H * 100)}%"><i>${s.n}</i><span>${s.name}</span></button>`).join('');
  const redCap = $('#redCap');
  function redShow(id) {
    const s = red.spots.find(x => x.id === id) || red.spots[0];
    $$('#redPan .stamp-pin').forEach(b => b.classList.toggle('on', b.dataset.spot === s.id));
    $$('#redPan [data-spot]').forEach(g => g.classList.toggle('sel', g.dataset.spot === s.id));
    $$('#redItin [data-spot]').forEach(li => li.classList.toggle('sel', li.dataset.spot === s.id));
    redCap.innerHTML = `<b class="cap-n">${s.n}</b><div><h3>${s.name}</h3><p class="cap-area">${s.area}</p><p>${s.note}</p><p class="cap-fit">适合：${s.fit.map(f => `<em>${f}</em>`).join('')}</p></div>`;
  }
  $('#redPan').addEventListener('click', e => { const t = e.target.closest('[data-spot]'); if (t) redShow(t.dataset.spot); });
  redShow('xiangshan');

  function itinHTML(it) {
    return `<header class="itin-head"><p class="itin-tag">${it.tag}</p><h3>${it.title}</h3>
      <ul class="itin-meta">${it.meta.map(([n, u]) => `<li>${n ? `<b>${n}</b>` : ''}${u}</li>`).join('')}</ul>${yin('示例', 'yin-w')}</header>
      <ol class="itin-list">${it.rows.map(([t, b, s, spot]) => `<li${spot ? ` class="key" data-spot="${spot}"` : ''}><time>${t}</time><i></i><div><b>${b}</b><span>${s}</span></div></li>`).join('')}</ol>
      <footer class="itin-foot">含${it.foot.map(f => `<em>${f}</em>`).join('')}</footer>`;
  }
  $('#redItin').innerHTML = itinHTML(RED_ITIN);
  $('#redItin').addEventListener('click', e => { const li = e.target.closest('[data-spot]'); if (li) { redShow(li.dataset.spot); $('#redLeaf').scrollIntoView({ behavior: reduce ? 'auto' : 'smooth', block: 'center' }); } });
  $('#redForms').innerHTML = RED_FORMS.map(([c, t, d]) => `<li>${yin(c)}<div><b>${t}</b><span>${d}</span></div></li>`).join('');

  /* ================= 贰 团建 ================= */
  const team = V.teamScroll();
  $('#teamPan').innerHTML = scWrap(team.svg, `0 0 ${team.W} ${team.H}`) +
    team.spots.map(s => `<button type="button" class="stamp-pin sm" data-spot="${s.id}" style="left:${N(s.x / team.W * 100)}%;top:${N(s.y / team.H * 100)}%"><i></i><span>${s.name}</span></button>`).join('');
  const teamCap = $('#teamCap'), teamMenu = $('#teamMenu'), teamNote = $('#teamNote');
  let teamKind = 'gov';
  function setTeam(k) {
    teamKind = k;
    const T = TEAM[k];
    $$('.tabs .tab').forEach(b => { const on = b.dataset.tab === k; b.classList.toggle('on', on); b.setAttribute('aria-selected', on); });
    teamNote.textContent = T.note;
    teamMenu.innerHTML = T.items.map(([id, c, t, d, meta]) => `<li data-item="${id}">${yin(c)}<div><b>${t}</b><p>${d}</p></div><span class="meta">${meta.map(m => `<i>${m}</i>`).join('')}</span></li>`).join('');
    // 画上只亮这一类活动发生的地方
    $$('#teamPan .stamp-pin').forEach(b => { const s = team.spots.find(x => x.id === b.dataset.spot); b.classList.toggle('dim', !s[k].length); });
    teamCap.innerHTML = `<p class="cap-tip">点画上的印记，看在那里能做什么</p>`;
    $$('#teamPan .spot').forEach(g => g.classList.remove('sel'));
  }
  function teamShow(id) {
    const s = team.spots.find(x => x.id === id); if (!s) return;
    const ids = s[teamKind].length ? s[teamKind] : (s.gov.length ? s.gov : s.ent);
    if (!s[teamKind].length) setTeam(teamKind === 'gov' ? 'ent' : 'gov');
    $$('#teamPan .stamp-pin').forEach(b => b.classList.toggle('on', b.dataset.spot === id));
    $$('#teamPan .spot').forEach(g => g.classList.toggle('sel', g.dataset.spot === id));
    const items = TEAM[teamKind].items.filter(it => ids.includes(it[0]));
    $$('#teamMenu li').forEach(li => li.classList.toggle('sel', ids.includes(li.dataset.item)));
    teamCap.innerHTML = `<div><h3>${s.name}</h3>${items.map(([, c, t, d]) => `<p>${yin(c)}<b>${t}</b>　${d}</p>`).join('')}</div>`;
  }
  $('#teamPan').addEventListener('click', e => { const t = e.target.closest('[data-spot]'); if (t) teamShow(t.dataset.spot); });
  setTeam('gov');
  $$('.tabs .tab').forEach(b => b.addEventListener('click', () => setTeam(b.dataset.tab)));

  /* ================= 叁 会务研学：会场平面（仿样式雷图档） ================= */
  (function plan() {
    const seats = [];
    for (let r = 0; r < 8; r++) for (const [x0, x1] of [[96, 262], [318, 484]]) for (let x = x0; x <= x1; x += 16.6) seats.push(`M${N(x)} ${150 + r * 19}h11v7h-11Z`);
    let cols = ''; for (let x = 60; x <= 520; x += 46) for (const y of [40, 330]) cols += `M${x - 3} ${y - 3}h6v6h-6Z`;
    for (let y = 86; y < 330; y += 61) for (const x of [60, 520]) cols += `M${x - 3} ${y - 3}h6v6h-6Z`;
    // 黄签：竖排小字，贴在图上
    const slip = (x, y, t, k, red) => `<g class="ys-slip${red ? ' red' : ''}" data-k="${k}"><rect x="${x - 9}" y="${y}" width="18" height="${t.length * 13 + 10}" rx="1.5"/>${[...t].map((ch, i) => `<text x="${x}" y="${y + 17 + i * 13}">${ch}</text>`).join('')}</g>`;
    const svg = `<svg class="ys" viewBox="0 0 600 420" role="img" aria-label="会场平面布置示意（仿样式雷图档画法）">
      <rect class="ys-paper" x="0" y="0" width="600" height="420"/>
      <path class="ys-wall" d="M40 20H540V350H150M106 350H40Z"/><path class="ys-wall2" d="M46 26H534V344H150M106 344H46Z"/>
      <path class="ys-door" d="M106 350A44 44 0 0 1 150 306M150 350V306"/>
      <path class="ys-gap" d="M540 70V110"/><path class="ys-exit" d="M540 72V108"/>
      <path class="ys-col" d="${cols}"/>
      <g class="ys-area" data-k="stage"><rect x="160" y="46" width="260" height="50" rx="2"/><path class="ys-screen" d="M210 36H370"/>${[188, 216, 244, 272, 300, 328, 356, 384].map(x => `<circle class="ys-chair" cx="${x}" cy="66" r="4"/>`).join('')}<rect class="ys-desk" x="398" y="60" width="16" height="14"/></g>
      <path class="ys-seat" d="${seats.join('')}"/>
      <path class="ys-aisle" d="M290 140V306"/>
      <g class="ys-area" data-k="sign"><rect x="56" y="282" width="72" height="18" rx="1"/></g>
      <g class="ys-area" data-k="tea">${[[430, 318], [468, 318], [506, 318]].map(([x, y]) => `<circle cx="${x}" cy="${y}" r="11"/>`).join('')}</g>
      <g class="ys-area" data-k="cam"><path d="M276 322h18v12h-18ZM294 325l8-4v14l-8-4Z"/></g>
      <g class="ys-area" data-k="guide">${[[164, 318], [182, 318]].map(([x, y]) => `<circle cx="${x}" cy="${y}" r="4.6"/>`).join('')}</g>
      <path class="ys-route" pathLength="1" d="M128 412V330C128 318 120 312 108 310M136 318C220 318 266 318 282 318C290 318 290 310 290 300V160"/>
      ${slip(140, 100, '主席台', 'stage', true)}${slip(30, 214, '签到', 'sign')}${slip(560, 270, '茶歇', 'tea')}${slip(316, 352, '机位', 'cam')}${slip(204, 352, '引导', 'guide')}${slip(566, 116, '安全出口', 'exit', true)}
      <g class="ys-tblock"><rect x="562" y="20" width="26" height="86"/>${[...'会场图样'].map((ch, i) => `<text x="575" y="${40 + i * 18}">${ch}</text>`).join('')}</g>
    </svg>`;
    $('#micePlan').innerHTML = `<div class="leaf-in">${svg}</div><figcaption><p>剧院式布局示意。<span class="m-hide">图上黄签标出主席台、签到、引导、机位、茶歇和安全出口；</span>朱红虚线是参会人员的动线。</p></figcaption>`;
  })();
  $('#miceChecks').innerHTML = CHECKS.map(([h, items]) => `<section class="ck">${yin(h, 'yin-v')}<ul>${items.map((t, i) => `<li style="--i:${i}"><svg class="ck-i" viewBox="0 0 18 18" aria-hidden="true"><path pathLength="1" d="M2.4 9.6C4 10.6 5.4 12.2 6.6 14.4C8.6 9.6 11.6 6 16 2.6"/></svg>${t}</li>`).join('')}</ul></section>`).join('');
  $('#study').innerHTML = STUDY.map(([c, t, d]) => `<li>${yin(c)}<div><b>${t}</b><p>${d}</p></div></li>`).join('');

  /* ================= 肆 北京资源图 ================= */
  const bjMap = $('#bjMap'), bjCard = $('#bjCard'), bjFilter = $('#bjFilter');
  const LABEL = { r: [14, 4.5, 'start'], l: [-14, 4.5, 'end'], t: [0, -15, 'middle'], b: [0, 24, 'middle'], tr: [12, -10, 'start'], bl: [-12, 20, 'end'] };
  const linesOf = k => Object.entries(BJ_LINES).filter(([, L]) => L.path.includes(k) || (L.branch || []).includes(k)).map(([id]) => id);
  function polyline(ids) {
    const pts = ids.map(k => BJ[k]);
    let d = `M${pts[0][0]} ${pts[0][1]}`;
    for (let i = 1; i < pts.length; i++) {
      const [x0, y0] = pts[i - 1], [x1, y1] = pts[i], dx = x1 - x0, dy = y1 - y0, m = Math.min(Math.abs(dx), Math.abs(dy));
      d += `L${x0 + Math.sign(dx) * m} ${y0 + Math.sign(dy) * m}L${x1} ${y1}`;   // 地铁图式：先 45° 斜线，再横竖直线
    }
    return d;
  }
  // 站点旁的小地标：用首屏和红色足迹里同一份界画，缩成小像
  function mini(name, x, y, h) {
    const L = (A2L[name] || window.VIGNETTE.RED[name])(), sc = h / (L.box[3] - L.box[2]);
    return `<g class="bj-mini" transform="translate(${x} ${y}) scale(${N(sc * 100) / 100} ${-N(sc * 100) / 100})">${L.g}</g>`;
  }
  const A2L = window.ART2.LANDMARKS;
  (function drawMap() {
    let lines = '', stations = '', minis = '';
    for (const [id, L] of Object.entries(BJ_LINES)) {
      const d = polyline(L.path) + (L.branch ? ' ' + polyline(L.branch) : '');
      lines += `<g class="bj-line" data-line="${id}" style="--c:${L.color}"><path class="bj-case" d="${d}"/><path class="bj-ink" d="${d}"/></g>`;
    }
    for (const [k, [x, y, name, , , , pos, lm]] of Object.entries(BJ)) {
      const ls = linesOf(k), inter = ls.length > 1, [dx, dy, anchor] = LABEL[pos || 'r'], c = BJ_LINES[ls[0]].color;
      stations += `<g class="bj-st${inter ? ' x' : ''}" data-st="${k}" data-lines="${ls.join(' ')}" tabindex="0" role="button" aria-label="${esc(name)}">
        <circle class="bj-hit" cx="${x}" cy="${y}" r="16"/><circle class="bj-dot" cx="${x}" cy="${y}" r="${inter ? 7 : 5.4}" style="--c:${c}"/>
        <text x="${x + dx}" y="${y + dy}" text-anchor="${anchor}">${esc(name)}</text></g>`;
    }
    // 几处有代表性的站点，旁边配一枚小地标
    minis = mini('foxiangge', 212, 156, 46) + mini('jiaolou', 352, 306, 30) + mini('qinian', 448, 392, 40) + mini('lugou', 226, 446, 18) + mini('honglou', 428, 252, 20) + mini('shuangqing', 120, 206, 24);
    const home = `<g class="bj-home" data-st="home" tabindex="0" role="button" aria-label="晟帆国旅所在地">
      <circle class="bj-pulse" cx="${HOME[0]}" cy="${HOME[1]}" r="9"/><circle class="bj-hdot" cx="${HOME[0]}" cy="${HOME[1]}" r="9"/>
      <path class="bj-sail" transform="translate(${HOME[0] - 6.2} ${HOME[1] - 7.4}) scale(.62)" d="M10.6 1.8c3.4 3.6 4.4 8.6 2.2 13.4H5.4C9 11.4 10.8 6.8 10.6 1.8Z"/>
      <text x="${HOME[0] + 16}" y="${HOME[1] + 5}">晟帆 · 丰台花乡</text></g>`;
    bjMap.innerHTML = `<div class="sc v-sc" data-season="${season()}"><svg class="a2 bj-svg" viewBox="0 0 720 540" role="img" aria-label="北京资源示意线路图">
      <rect class="bj-ring" x="8" y="8" width="704" height="524" rx="6"/><rect class="bj-ring2" x="16" y="16" width="688" height="508" rx="3"/>
      ${window.SCENE.HELP.xiangyun(600, 70, 0.6)}${window.SCENE.HELP.xiangyun(110, 470, 0.5, -1)}
      <path class="bj-axis" d="M392 120V470"/><text class="bj-axis-t" x="398" y="132">中轴线</text>
      ${minis}${lines}${stations}${home}
      <g class="bj-legend-seal"><rect x="636" y="474" width="52" height="40" rx="2"/><text x="662" y="491">京华</text><text x="662" y="506">导览</text></g></svg></div>`;
    bjFilter.innerHTML = [['all', '全部', '#2f2a24'], ...Object.entries(BJ_LINES).map(([id, L]) => [id, L.name, L.color])]
      .map(([id, t, c], i) => `<button type="button" class="bj-f${i ? '' : ' on'}" data-line="${id}" style="--c:${c}"><i></i>${t}</button>`).join('');
  })();
  function bjShow(k) {
    $$('.bj-st, .bj-home', bjMap).forEach(g => g.classList.toggle('sel', g.dataset.st === k));
    if (k === 'home') {
      bjCard.innerHTML = `<p class="bj-tags"><span class="bj-tag" style="--c:#1f4a66">出发地</span></p><h3>晟帆国旅 · 新华双创园</h3><p class="bj-area">丰台区花乡六圈西路 8 号院</p>
        <p class="bj-desc">市内线路一般当天往返；京郊线路按季节和人数推荐场地，车辆、餐食、保险统一安排。</p>
        <button class="b-primary sm" type="button" data-act="inquiry">按需求定制线路</button>`;
      return;
    }
    const [, , name, area, desc, fit] = BJ[k], ls = linesOf(k);
    bjCard.innerHTML = `<p class="bj-tags">${ls.map(id => `<span class="bj-tag" style="--c:${BJ_LINES[id].color}">${BJ_LINES[id].name}</span>`).join('')}</p>
      <h3>${esc(name)}</h3><p class="bj-area">${esc(area)}</p><p class="bj-desc">${esc(desc)}</p>
      <p class="bj-fit">适合：${fit.map(f => `<em>${f}</em>`).join('')}</p>
      <button class="b-primary sm" type="button" data-act="inquiry" data-pre="${ls.includes('red') ? '红色教育 / 主题党日' : ls.includes('blue') ? '会议会展' : ls.includes('green') ? '企业团建' : '研学考察'}">按这里定制方案</button>`;
  }
  bjMap.addEventListener('click', e => { const g = e.target.closest('[data-st]'); if (g) bjShow(g.dataset.st); });
  bjMap.addEventListener('keydown', e => { if ((e.key === 'Enter' || e.key === ' ') && e.target.dataset && e.target.dataset.st) { e.preventDefault(); bjShow(e.target.dataset.st); } });
  bjFilter.addEventListener('click', e => {
    const b = e.target.closest('.bj-f'); if (!b) return;
    const id = b.dataset.line;
    $$('.bj-f', bjFilter).forEach(x => x.classList.toggle('on', x === b));
    $$('.bj-line', bjMap).forEach(g => g.classList.toggle('dim', id !== 'all' && g.dataset.line !== id));
    $$('.bj-st', bjMap).forEach(g => g.classList.toggle('dim', id !== 'all' && !g.dataset.lines.split(' ').includes(id)));
    if (id !== 'all') bjShow(BJ_LINES[id].path[0]);
  });
  bjShow('home');
  requestAnimationFrame(() => { const sc = $('#bjScroll'); if (sc.scrollWidth > sc.clientWidth) sc.scrollLeft = (sc.scrollWidth - sc.clientWidth) * 0.28; });

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
  $('#casesList').innerHTML = CASES.map(c => `<li class="dos${c.placeholder ? ' ph' : ''}" data-ac="${c.kind}">
    <div class="dos-tab"><span>${c.tag}</span><b>${c.year}</b></div>
    <div class="dos-photo slot"><span>项目现场照片</span><small>待补充</small></div>
    <div class="dos-body"><h3>${esc(c.title)}</h3><dl>
      <div><dt>时间</dt><dd>${esc(c.when)}</dd></div><div><dt>地点</dt><dd>${esc(c.where)}</dd></div>
      <div><dt>规模</dt><dd>${esc(c.size)}</dd></div><div><dt>服务</dt><dd>${esc(c.what)}</dd></div></dl></div>
    ${yin('档', 'yin-r')}</li>`).join('');

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
  window.SF = { $, $$, esc, openSheet, toast, copy, itinHTML, inWeChat, desktop, reduce, ACTIONS, yin };
  document.addEventListener('click', e => {
    const c = e.target.closest('[data-cert]'); if (c) { certSheet(c.dataset.cert); return; }
    const act = e.target.closest('[data-act]');
    if (act && ACTIONS[act.dataset.act]) { e.preventDefault(); const pc = $('#pickCard'); if (pc) pc.hidden = true; ACTIONS[act.dataset.act](act); }
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
  const nav = $('#nav'), dock = $('#dock'), juan = $('.juan');
  let ticking = false;
  function onScroll() {
    if (ticking) return; ticking = true;
    requestAnimationFrame(() => {
      const past = scrollY > juan.offsetTop + juan.offsetHeight - innerHeight * 0.6;
      nav.classList.toggle('solid', past);
      dock.classList.toggle('show', past);
      ticking = false;
    });
  }
  addEventListener('scroll', onScroll, { passive: true });
  onScroll();
  // 首屏的绢纹生成后，全站底子也用上
  setTimeout(silkBg, 0);
})();
