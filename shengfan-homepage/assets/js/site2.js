/* 晟帆国旅主页 · 后四章：会务研学、北京资源图、资质与保障、项目档案、名片与咨询 */
(function () {
  'use strict';
  const { $, $$, esc, openSheet, toast, copy, inWeChat, desktop, ACTIONS } = window.SF;
  const N = v => Math.round(v * 10) / 10;

  /* ================= 内容 ================= */
  const PLAN_LEGEND = ['主屏与灯光音响', '主席台与讲台', '嘉宾与代表席', '签到台', '引导岗', '摄影摄像机位', '茶歇区', '安全出口'];
  const CHECKS = [
    ['会前', ['需求沟通，出方案和预算', '场地、酒店比选与预订', '报名统计与参会通知', '物料设计制作', '交通与接站安排']],
    ['会中', ['签到与现场引导', '会场搭建与设备调试', '议程与时间把控', '餐饮与茶歇', '摄影摄像', '医疗与应急保障']],
    ['会后', ['返程送站', '影像资料整理交付', '费用结算与票据', '总结回访']],
  ];
  const STUDY = [
    ['yanxiu', '干部研修', '依托高校资源开展非学历培训，课程、师资、地点按需定制。'],
    ['duibiao', '产业对标', '走进领先企业实地参访，安排交流座谈与案例研讨。'],
    ['kaocha', '专题考察', '围绕数字化转型、专精特新等主题组织调研考察。'],
  ];

  // 北京资源图：示意坐标（720×560），不按实际比例
  const BJ_LINES = {
    red: { name: '红色教育', color: '#b0252c', path: ['xiangshan', 'lidazhao', 'honglou', 'dangshi'], branch: ['lugou', 'lidazhao'] },
    gold: { name: '文化研学', color: '#a87e26', path: ['badaling', 'yiheyuan', 'gugong', 'tiantan'] },
    green: { name: '团建场地', color: '#4a8f1e', path: ['fangshan', 'mentougou', 'badaling', 'yanqi', 'miyun', 'pinggu'] },
    blue: { name: '会议场地', color: '#1f5aa6', path: ['yanqi', 'zgc', 'shougang', 'lize', 'yizhuang', 'cbd'] },
  };
  const BJ = {
    xiangshan: [150, 214, '香山革命纪念地', '海淀区', '1949 年中共中央在此指挥渡江战役、筹建新中国。', ['主题党日', '现场教学'], 'l'],
    lidazhao: [318, 300, '李大钊故居', '西城区', '李大钊在北京居住时间最长的寓所。', ['现场教学', '党史学习'], 'bl'],
    honglou: [404, 262, '北大红楼', '东城区', '中国共产党早期北京革命活动旧址。', ['主题党日', '现场教学'], 'r'],
    dangshi: [470, 168, '中国共产党历史展览馆', '朝阳区', '全面展示党的百年奋斗历程。', ['党史学习', '主题党日'], 'r'],
    lugou: [232, 412, '卢沟桥 · 抗战纪念馆', '丰台区', '全民族抗战爆发地，就在公司所在的丰台。', ['现场教学', '爱国主义教育'], 'b'],
    badaling: [118, 92, '延庆 · 八达岭长城', '延庆区', '明长城代表段落，登城研学、团队登山都合适。', ['文化研学', '登山'], 'tr'],
    yiheyuan: [236, 168, '颐和园', '海淀区', '皇家园林，适合文化研学与健步走。', ['文化研学', '健步走'], 'r'],
    gugong: [378, 318, '故宫', '东城区', '明清皇宫，北京中轴线核心。', ['文化研学'], 'l'],
    tiantan: [420, 400, '天坛', '东城区', '明清皇家祭天之所，北京中轴线南段。', ['文化研学'], 'r'],
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
    ['baoxian', '旅行社责任险', '人保旅行社责任保险统保示范项目'],
    ['baoxian', '意外与专项险', '旅游意外、高原、自驾、航班延误险按需投保'],
    ['cheliang', '车辆', '50 座大巴、16 座考斯特到商务车、礼宾车'],
    ['yingji', '应急预案', '每个项目单独制定，现场有人负责'],
    ['zhuanren', '专人对接', '从方案到结算，一位负责人跟到底'],
    ['piaoju', '合同票据', '合同、发票、结算单据齐全'],
  ];
  const GC_STRIP = ['gaoxin', 'zjtx', 'iso9001', 'iso14001', 'iso45001', 'iso27001', 'iso22000', 'iso28000', 'haccp', 'shipin', 'yiliao', 'icp', 'dengbao', 'trace', 'shouhou', 'chuji', 'shengxian'];

  // 项目档案：真实项目到位后，把 placeholder 改成 false 并填好字段
  const CASES = [
    { kind: 'red', tag: '红色教育', year: '待补充', title: '项目名称待补充', when: '待补充', where: '待补充', size: '待补充', what: '待补充', placeholder: true },
    { kind: 'green', tag: '团建', year: '待补充', title: '项目名称待补充', when: '待补充', where: '待补充', size: '待补充', what: '待补充', placeholder: true },
    { kind: 'blue', tag: '会务', year: '待补充', title: '项目名称待补充', when: '待补充', where: '待补充', size: '待补充', what: '待补充', placeholder: true },
  ];

  /* ================= 03 会务研学 ================= */
  (function plan() {
    const seats = [];
    for (let r = 0; r < 8; r++) for (const [x0, x1] of [[78, 254], [306, 482]]) for (let x = x0; x <= x1; x += 17.6) seats.push(`M${N(x)} ${132 + r * 20}h11.6v7.6h-11.6Z`);
    const mark = (n, x, y) => `<g class="pl-n"><circle cx="${x}" cy="${y}" r="11"/><text x="${x}" y="${y + 4.6}" text-anchor="middle">${n}</text></g>`;
    const svg = `<svg class="pl" viewBox="0 0 560 372" role="img" aria-label="会场平面布置示意">
      <path class="pl-wall" d="M132 332H536V20H24V332H88M536 64V102"/>
      <path class="pl-gap" d="M536 66V100"/>
      <path class="pl-door" d="M88 332A44 44 0 0 1 132 288M132 332V288"/>
      <path class="pl-screen" d="M196 32H364"/>
      <rect class="pl-stage" x="150" y="44" width="260" height="46" rx="3"/>
      <path class="pl-fine" d="M176 66H332M176 72H332"/>
      ${[186, 214, 242, 270, 298, 326].map(x => `<circle class="pl-chair" cx="${x}" cy="58" r="4"/>`).join('')}
      <rect class="pl-desk" x="352" y="56" width="22" height="16" rx="2"/>
      <path class="pl-seat" d="${seats.join('')}"/>
      <path class="pl-fine" d="M280 120V300" stroke-dasharray="2 4"/>
      <rect class="pl-desk" x="38" y="270" width="68" height="18" rx="2"/>
      ${[[150, 312], [168, 312]].map(([x, y]) => `<circle class="pl-p" cx="${x}" cy="${y}" r="4.4"/>`).join('')}
      <path class="pl-cam" d="M268 318h18v12h-18ZM286 321l8-4v14l-8-4Z"/>
      ${[[418, 296], [458, 296], [498, 296]].map(([x, y]) => `<circle class="pl-desk" cx="${x}" cy="${y}" r="11"/>`).join('')}
      <path class="pl-route" pathLength="1" d="M110 368V312C110 300 100 292 84 290M118 300C200 300 252 300 268 300C278 300 280 292 280 282V140"/>
      ${mark(1, 280, 20)}${mark(2, 430, 67)}${mark(3, 66, 120)}${mark(4, 72, 252)}${mark(5, 159, 292)}${mark(6, 312, 324)}${mark(7, 458, 266)}${mark(8, 520, 83)}
    </svg>`;
    $('#micePlan').innerHTML = svg + `<figcaption><ol class="pl-legend">${PLAN_LEGEND.map(t => `<li>${t}</li>`).join('')}</ol><p>示意：剧院式布局，虚线为参会人员动线。</p></figcaption>`;
  })();
  $('#miceChecks').innerHTML = CHECKS.map(([h, items]) => `<section class="ck"><h3><span>${h}</span></h3><ul>${items.map((t, i) => `<li style="--i:${i}"><svg class="ck-i" viewBox="0 0 16 16" aria-hidden="true"><circle cx="8" cy="8" r="7"/><path d="M4.8 8.2l2.2 2.2 4.2-4.4"/></svg>${t}</li>`).join('')}</ul></section>`).join('');
  $('#study').innerHTML = STUDY.map(([ic, t, d]) => `<li>${ICON(ic)}<div><b>${t}</b><p>${d}</p></div></li>`).join('');

  /* ================= 04 北京资源图 ================= */
  const bjMap = $('#bjMap'), bjCard = $('#bjCard'), bjFilter = $('#bjFilter');
  const LABEL = { r: [14, 4.5, 'start'], l: [-14, 4.5, 'end'], t: [0, -15, 'middle'], b: [0, 24, 'middle'], tr: [12, -10, 'start'], bl: [-12, 20, 'end'] };
  const linesOf = k => Object.entries(BJ_LINES).filter(([, L]) => L.path.includes(k) || (L.branch || []).includes(k)).map(([id]) => id);
  function polyline(ids) {
    const pts = ids.map(k => BJ[k]);
    let d = `M${pts[0][0]} ${pts[0][1]}`;
    for (let i = 1; i < pts.length; i++) {
      const [x0, y0] = pts[i - 1], [x1, y1] = pts[i], dx = x1 - x0, dy = y1 - y0;
      // 地铁图式折线：先走 45° 斜线，再走横/竖直线
      const m = Math.min(Math.abs(dx), Math.abs(dy));
      const mx = x0 + Math.sign(dx) * m, my = y0 + Math.sign(dy) * m;
      d += `L${mx} ${my}L${x1} ${y1}`;
    }
    return d;
  }
  (function drawMap() {
    let lines = '', stations = '';
    for (const [id, L] of Object.entries(BJ_LINES)) {
      const d = polyline(L.path) + (L.branch ? ' ' + polyline(L.branch) : '');
      lines += `<g class="bj-line" data-line="${id}" style="--c:${L.color}"><path class="bj-case" d="${d}"/><path class="bj-ink" d="${d}"/></g>`;
    }
    for (const [k, [x, y, name, , , , pos]] of Object.entries(BJ)) {
      const ls = linesOf(k), inter = ls.length > 1, [dx, dy, anchor] = LABEL[pos || 'r'];
      const c = BJ_LINES[ls[0]].color;
      stations += `<g class="bj-st${inter ? ' x' : ''}" data-st="${k}" data-lines="${ls.join(' ')}" tabindex="0" role="button" aria-label="${esc(name)}">
        <circle class="bj-hit" cx="${x}" cy="${y}" r="16"/><circle class="bj-dot" cx="${x}" cy="${y}" r="${inter ? 7 : 5.4}" style="--c:${c}"/>
        <text x="${x + dx}" y="${y + dy}" text-anchor="${anchor}">${esc(name)}</text></g>`;
    }
    const home = `<g class="bj-home" data-st="home" tabindex="0" role="button" aria-label="晟帆国旅所在地">
      <circle class="bj-pulse" cx="${HOME[0]}" cy="${HOME[1]}" r="9"/><circle class="bj-hdot" cx="${HOME[0]}" cy="${HOME[1]}" r="9"/>
      <path class="bj-sail" transform="translate(${HOME[0] - 6.2} ${HOME[1] - 7.4}) scale(.62)" d="M10.6 1.8c3.4 3.6 4.4 8.6 2.2 13.4H5.4C9 11.4 10.8 6.8 10.6 1.8Z"/>
      <text x="${HOME[0] + 16}" y="${HOME[1] + 5}">晟帆 · 丰台花乡</text></g>`;
    bjMap.innerHTML = `<svg class="bj-svg" viewBox="0 0 720 540" role="img" aria-label="北京资源示意线路图">
      <rect class="bj-ring" x="8" y="8" width="704" height="524" rx="22"/>
      <path class="bj-axis" d="M392 120V470"/><text class="bj-axis-t" x="398" y="132">中轴线</text>
      ${lines}${stations}${home}</svg>`;
    bjFilter.innerHTML = [['all', '全部', '#12213f'], ...Object.entries(BJ_LINES).map(([id, L]) => [id, L.name, L.color])]
      .map(([id, t, c], i) => `<button type="button" class="bj-f${i ? '' : ' on'}" data-line="${id}" style="--c:${c}"><i></i>${t}</button>`).join('');
  })();
  function bjShow(k) {
    $$('.bj-st, .bj-home', bjMap).forEach(g => g.classList.toggle('sel', g.dataset.st === k));
    if (k === 'home') {
      bjCard.innerHTML = `<p class="bj-tag" style="--c:#004291">出发地</p><h3>晟帆国旅 · 新华双创园</h3><p class="bj-area">丰台区花乡六圈西路 8 号院</p>
        <p class="bj-desc">市内线路一般当天往返；京郊线路按季节和人数推荐场地，车辆、餐食、保险统一安排。</p>
        <button class="btn sm primary" type="button" data-act="inquiry">按需求定制线路</button>`;
      return;
    }
    const [, , name, area, desc, fit] = BJ[k], ls = linesOf(k);
    bjCard.innerHTML = `<p class="bj-tags">${ls.map(id => `<span class="bj-tag" style="--c:${BJ_LINES[id].color}">${BJ_LINES[id].name}</span>`).join('')}</p>
      <h3>${esc(name)}</h3><p class="bj-area">${esc(area)}</p><p class="bj-desc">${esc(desc)}</p>
      <p class="bj-fit">适合：${fit.map(f => `<em>${f}</em>`).join('')}</p>
      <button class="btn sm primary" type="button" data-act="inquiry" data-pre="${ls.includes('red') ? '红色教育 / 主题党日' : ls.includes('blue') ? '会议会展' : ls.includes('green') ? '企业团建' : '研学考察'}">按这里定制方案</button>`;
  }
  bjMap.addEventListener('click', e => { const g = e.target.closest('[data-st]'); if (g) bjShow(g.dataset.st); });
  bjMap.addEventListener('keydown', e => { if ((e.key === 'Enter' || e.key === ' ') && e.target.dataset.st) { e.preventDefault(); bjShow(e.target.dataset.st); } });
  bjFilter.addEventListener('click', e => {
    const b = e.target.closest('.bj-f'); if (!b) return;
    const id = b.dataset.line;
    $$('.bj-f', bjFilter).forEach(x => x.classList.toggle('on', x === b));
    bjMap.dataset.only = id === 'all' ? '' : id;
    $$('.bj-line', bjMap).forEach(g => g.classList.toggle('dim', id !== 'all' && g.dataset.line !== id));
    $$('.bj-st', bjMap).forEach(g => g.classList.toggle('dim', id !== 'all' && !g.dataset.lines.split(' ').includes(id)));
    if (id !== 'all') { const first = BJ_LINES[id].path[0]; bjShow(first); }
  });
  bjShow('home');
  // 手机上默认把出发地滚到可视区
  requestAnimationFrame(() => { const sc = $('#bjScroll'); if (sc.scrollWidth > sc.clientWidth) sc.scrollLeft = (sc.scrollWidth - sc.clientWidth) * 0.28; });

  /* ================= 05 资质与保障 ================= */
  $('#certFiles').innerHTML = `<ol class="file-row">${CERTS.map(([id, img, name, level, kv], i) =>
    `<li style="--i:${i}"><button type="button" class="file" data-cert="${id}" aria-label="查看${esc(name)}原件">
      <span class="file-img"><img src="assets/img/cert/s/${img}.jpg" alt="" loading="lazy"></span>
      <span class="file-t"><b>${name}</b><em>${level}</em><small>${kv.map(([k, v]) => k === '有效期' ? '有效期至 ' + v.split('–')[1].trim() : '').filter(Boolean).join('') || kv[0][0] + ' ' + kv[0][1]}</small></span>
    </button></li>`).join('')}</ol>`;
  $('#guards').innerHTML = GUARDS.map(([ic, t, d]) => `<li>${ICON(ic)}<div><b>${t}</b><span>${d}</span></div></li>`).join('');
  (function strip() {
    const one = GC_STRIP.map(k => `<img src="assets/img/cert/s/gc-${k}.jpg" alt="" loading="lazy">`).join('');
    $('#gcStrip').innerHTML = `<div class="gc-track">${one}${one}</div>`;
  })();
  function certSheet(id) {
    const c = CERTS.find(x => x[0] === id); if (!c) return;
    const [, img, name, level, kv, verify] = c;
    openSheet(`<h3>${name}</h3><p class="sub">${level} · 北京晟帆国际旅行社有限公司</p>
      <figure class="cert-big"><img src="assets/img/cert/${img}.jpg" alt="${esc(name)}原件"></figure>
      <dl class="kv">${kv.map(([k, v]) => `<div><dt>${k}</dt><dd>${v}</dd></div>`).join('')}</dl>
      ${verify ? `<a class="btn ghost" href="${verify[1]}" target="_blank" rel="noopener">去官方平台核验 · ${verify[0]}</a>` : ''}
      <p class="hint">图片来自公司宣传册，正式上线前换成清晰扫描件。</p>`);
  }
  document.addEventListener('click', e => { const b = e.target.closest('[data-cert]'); if (b) certSheet(b.dataset.cert); });

  /* ================= 06 项目档案 ================= */
  $('#cases-list').innerHTML = CASES.map(c => `<li class="dos${c.placeholder ? ' ph' : ''}" data-theme="${c.kind}">
    <div class="dos-tab"><span>${c.tag}</span><b>${c.year}</b></div>
    <div class="dos-photo slot"><span>项目现场照片</span><small>待补充</small></div>
    <div class="dos-body"><h3>${esc(c.title)}</h3><dl>
      <div><dt>时间</dt><dd>${esc(c.when)}</dd></div><div><dt>地点</dt><dd>${esc(c.where)}</dd></div>
      <div><dt>规模</dt><dd>${esc(c.size)}</dd></div><div><dt>服务</dt><dd>${esc(c.what)}</dd></div></dl></div></li>`).join('');

  /* ================= 名片 ================= */
  const cardEl = $('#card'), card = { ...cardEl.dataset };
  const fmt = m => m.replace(/^(\d{3})(\d{4})(\d{4})$/, '$1 $2 $3');
  cardEl.innerHTML = `
    <div class="card-main">
      <div class="card-brand"><img src="assets/img/sf-mark.png" alt="" width="22" height="31"><div><b>北京晟帆国际旅行社有限公司</b><span>贵昌集团旗下 · 会务文旅板块</span></div></div>
      <div class="card-person"><div class="card-photo" aria-hidden="true"><span>照片</span></div><div><h3>${esc(card.name)}</h3><p>${esc(card.title)}</p></div></div>
      <ul class="card-lines">
        <li><svg class="i"><use href="#i-phone"/></svg><div><b>${card.mobile ? fmt(card.mobile) : '1XX XXXX XXXX'}</b><span>手机 · 微信同号${card.mobile ? '' : '（待提供）'}</span></div></li>
        <li><svg class="i"><use href="#i-tel"/></svg><div><b>${card.tel}</b><span>公司座机</span></div></li>
        <li><svg class="i"><use href="#i-mail"/></svg><div><b>${card.email}</b><span>邮箱</span></div></li>
        <li><svg class="i"><use href="#i-pin"/></svg><div><b>丰台区花乡六圈西路 8 号院</b><span>新华双创园 B 座 3 层 307 室</span></div></li>
      </ul>
      <div class="card-cta">
        <button class="btn primary" type="button" data-act="call"><svg class="i"><use href="#i-phone"/></svg>打电话</button>
        <button class="btn ghost" type="button" data-act="wechat"><svg class="i"><use href="#i-wechat"/></svg>加微信</button>
        <button class="btn ghost" type="button" data-act="vcard"><svg class="i"><use href="#i-card"/></svg>存名片</button>
      </div>
    </div>
    <div class="card-side">
      <div class="qr ph" aria-label="微信二维码（待提供）"><span>微信二维码<br>待提供</span></div>
      <p>长按识别，加我微信</p>
      <button class="btn sm ghost" type="button" data-act="inquiry">获取活动方案<svg class="i"><use href="#i-arrow"/></svg></button>
    </div>`;

  function vcard() {
    const n = card.name || '';
    return ['BEGIN:VCARD', 'VERSION:3.0', `N:${n.slice(0, 1)};${n.slice(1)};;;`, `FN:${n}`, 'ORG:北京晟帆国际旅行社有限公司', `TITLE:${card.title || ''}`,
      card.mobile && `TEL;TYPE=CELL:${card.mobile}`, `TEL;TYPE=WORK:${card.tel}`, `EMAIL;TYPE=INTERNET:${card.email}`,
      `ADR;TYPE=WORK:;;${card.address};;;;`, 'NOTE:贵昌集团旗下 · 红色教育 · 党政团建 · 企业团建 · 会务研学',
      `URL:${location.href.split('#')[0]}`, 'END:VCARD'].filter(Boolean).join('\r\n');
  }
  Object.assign(ACTIONS, {
    call() { if (card.mobile) location.href = 'tel:' + card.mobile; else location.href = 'tel:' + card.tel.replace(/-/g, ''); },
    wechat() {
      openSheet(`<h3>加我微信</h3><p class="sub">长按识别二维码，或复制微信号搜索添加</p>
        <div class="qr ph big"><span>微信二维码<br>待提供</span></div>
        <dl class="kv"><div><dt>微信号</dt><dd>${card.wechat || '待提供'}</dd></div><div><dt>座机</dt><dd>${card.tel}</dd></div></dl>
        <button class="btn" type="button" data-act="copy-wechat">复制微信号</button>`);
    },
    'copy-wechat'() { copy(card.wechat, '微信号'); },
    'copy-email'() { copy(card.email, '邮箱'); },
    'copy-link'() { copy(location.href.split('#')[0], '链接'); },
    vcard() {
      if (inWeChat) {
        openSheet(`<h3>保存到通讯录</h3><p class="sub">微信里不能直接保存文件</p><p class="txt">请点右上角「···」，选「在浏览器打开」，再点一次「存名片」。</p><button class="btn ghost" type="button" data-act="copy-link">复制本页链接</button>`);
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
        <ul class="doc-list">${['营业执照（副本）', '旅行社业务经营许可证', '会务会展服务企业资质证书（一级）', 'ISO 9001 / 14001 / 45001 认证证书', '旅行社责任保险保单'].map(t => `<li><svg class="i"><use href="#i-doc"/></svg>${t}</li>`).join('')}</ul>
        <p class="txt">可提供加盖公章的复印件或电子版，请联系对接人，说明单位名称和用途。</p>
        <div class="row2"><button class="btn primary" type="button" data-act="call"><svg class="i"><use href="#i-phone"/></svg>打电话</button><button class="btn ghost" type="button" data-act="copy-email"><svg class="i"><use href="#i-mail"/></svg>复制邮箱</button></div>`);
    },
  });
})();
