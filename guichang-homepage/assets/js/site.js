(() => {
  const $ = (s, r = document) => r.querySelector(s);
  const $$ = (s, r = document) => [...r.querySelectorAll(s)];
  const reduce = matchMedia('(prefers-reduced-motion: reduce)').matches;
  const desktop = matchMedia('(min-width: 900px)');
  const inWeChat = /MicroMessenger/i.test(navigator.userAgent);
  const wait = ms => new Promise(r => setTimeout(r, ms));
  const clamp = (v, a = 0, b = 1) => Math.min(b, Math.max(a, v));
  const ease = t => 1 - Math.pow(1 - t, 3);
  const smooth = t => t * t * (3 - 2 * t);
  const rng = s => () => (s = (s * 16807) % 2147483647) / 2147483647;
  const card = $('#card').dataset;
  const IMG = 'assets/img/';

  /* ================= 数据 ================= */
  const GC = '贵昌集团有限公司', SF = '北京晟帆国际旅行社有限公司';
  const CNCA = ['全国认证认可信息公共服务平台', 'http://cx.cnca.cn/'];
  const GROUPS = { rec: '企业认定', lic: '准入许可', sys: '体系认证', ip: '知识产权' };
  // [id, 分组, 名称, 等级, 持证方, 图片名, 缩略图宽, 高, 说明行, 官方核验]
  const CERTS = [
    ['gaoxin', 'rec', '高新技术企业证书', '国家级认定', 'gc', 'gc-gaoxin', 300, 191, [['发证', '2024 年 12 月 · 有效期三年']], ['高新技术企业认定管理工作网', 'http://www.innocom.gov.cn/']],
    ['zjtx', 'rec', '北京市“专精特新”中小企业', '省级认定', 'gc', 'gc-zjtx', 300, 198, [['认定机构', '北京市经济和信息化局'], ['有效期', '在有效期内（复核后的新证书待补充）']], ['北京市经济和信息化局', 'https://jxj.beijing.gov.cn/']],
    ['shipin', 'lic', '食品经营许可证', '行政许可', 'gc', 'gc-shipin', 300, 202, [['作用', '食材配送进入机关食堂、医院的前提']]],
    ['yiliao', 'lic', '医疗器械经营许可证', '行政许可', 'gc', 'gc-yiliao', 300, 204, [['作用', '向医院供应医疗器械类物资的前提']]],
    ['yiliao2', 'lic', '第二类医疗器械经营备案凭证', '行政备案', 'gc', 'gc-yiliao-beian', 207, 300],
    ['icp', 'lic', '增值电信业务经营许可证', '行政许可', 'gc', 'gc-icp', 206, 300, [['作用', '合创慧采平台开展在线业务的资质']]],
    ['dengbao', 'lic', '信息系统安全等级保护备案证明', '公安备案', 'gc', 'gc-dengbao', 300, 191, [['备案系统', '合创慧采电商系统']]],
    ['mice', 'lic', '会务会展服务企业资质证书（一级）', '行业最高等级', 'sf', 'sf-mice', 226, 320, [['证书编号', 'ZDXY-2024182720'], ['有效期', '2024.07.26 – 2027.07.25']], ['中国招标投标网（证书公示）', 'http://www.cecbid.org.cn/']],
    ['travel', 'lic', '旅行社业务经营许可证', '行政许可', 'sf', 'sf-travel', 320, 221, [['许可机关', '北京市文化和旅游局']], ['文化和旅游部（旅行社名录可查）', 'https://www.mct.gov.cn/']],
    ['iso9001', 'sys', '质量管理体系认证证书', 'ISO 9001', 'gc', 'gc-iso9001', 206, 300],
    ['iso14001', 'sys', '环境管理体系认证证书', 'ISO 14001', 'gc', 'gc-iso14001', 206, 300],
    ['iso45001', 'sys', '职业健康安全管理体系认证证书', 'ISO 45001', 'gc', 'gc-iso45001', 207, 300],
    ['iso22000', 'sys', '食品安全管理体系认证证书', 'ISO 22000', 'gc', 'gc-iso22000', 226, 300],
    ['haccp', 'sys', 'HACCP 体系认证证书', 'HACCP', 'gc', 'gc-haccp', 207, 300],
    ['iso27001', 'sys', '信息安全管理体系认证证书', 'ISO/IEC 27001', 'gc', 'gc-iso27001', 208, 300],
    ['iso27701', 'sys', '隐私信息管理体系认证证书', 'ISO/IEC 27701', 'gc', 'gc-iso27701', 206, 300],
    ['iso28000', 'sys', '供应链安全管理体系认证证书', 'ISO 28000', 'gc', 'gc-iso28000', 203, 300],
    ['trace', 'sys', '可追溯管理体系认证证书', 'GB/T 38159', 'gc', 'gc-trace', 198, 300],
    ['shengxian', 'sys', '生鲜农产品配送管理体系认证证书', '体系认证', 'gc', 'gc-shengxian', 300, 298],
    ['chuji', 'sys', '初级生鲜食品配送管理体系认证证书', '体系认证', 'gc', 'gc-chuji', 267, 300],
    ['shouhou', 'sys', '售后服务认证证书（五星级）', 'GB/T 27922', 'gc', 'gc-shouhou', 297, 300],
    ['sf9001', 'sys', '质量管理体系认证证书', 'ISO 9001', 'sf', 'sf-iso9001', 225, 320, [['证书编号', 'HG24Q0503R0S'], ['有效期', '2024.11.06 – 2027.11.05']]],
    ['sf14001', 'sys', '环境管理体系认证证书', 'ISO 14001', 'sf', 'sf-iso14001', 225, 320, [['证书编号', 'HG24E0280R0S'], ['有效期', '2024.11.06 – 2027.11.05']]],
    ['sf45001', 'sys', '职业健康安全管理体系认证证书', 'ISO 45001', 'sf', 'sf-iso45001', 224, 320, [['证书编号', 'HG24S0243R0S'], ['有效期', '2024.11.06 – 2027.11.05']]],
    ['patent1', 'ip', '发明专利证书', '知识产权', 'gc', 'gc-patent1', 224, 300],
    ['patent2', 'ip', '发明专利证书', '知识产权', 'gc', 'gc-patent2', 217, 300],
    ['soft1', 'ip', '计算机软件著作权登记证书', '知识产权', 'gc', 'gc-soft1', 221, 300],
    ['soft2', 'ip', '计算机软件著作权登记证书', '知识产权', 'gc', 'gc-soft2', 207, 300],
    ['tm1', 'ip', '注册商标 · 稻草森林', '自有品牌', 'gc', 'gc-tm-daocao', 211, 300, [['用途', '自有食用油等食材品牌']]],
    ['tm2', 'ip', '注册商标 · 菱克', '自有品牌', 'gc', 'gc-tm-lingke', 210, 300, [['用途', '自有打印耗材品牌']]],
    ['tm3', 'ip', '注册商标 · 一叶扁舟', '自有品牌', 'gc', 'gc-tm-yiye', 210, 300, [['用途', '自有复印纸品牌']]],
  ].map(([id, g, name, lv, who, img, w, h, kv = [], verify]) => ({ id, g, name, lv, who, img, w, h, kv, verify: verify || (g === 'sys' ? CNCA : null) }));

  const LETTERS = { l1: '张家口冬奥村（冬残奥村）场馆运行团队 · 感谢信', l3: '北京 2022 年冬奥会和冬残奥会组织委员会 · 感谢信', l4: '中国华油集团有限公司 · 感谢信', l9: '北京大学人民医院 · 表彰信' };
  const letter = id => ({ img: `${IMG}letter/${id}.jpg`, tag: id === 'l9' ? '表彰信' : '感谢信', title: LETTERS[id].split(' · ')[0], sub: '来函原件 · 取自公司介绍材料' });
  const photo = (f, title, sub) => ({ img: `${IMG}photo/${f}.jpg`, tag: '公益', title, sub, photo: true });

  // 已服务地区：[键, 名称, 说明, 标注偏移 x, y, 对齐]
  const SERVED = [
    ['tj', '天津', '已有服务项目', 1.3, .55, 'start'],
    ['he', '河北', '已有服务项目', -.4, 2.6, 'end'],
    ['sx', '山西', '同煤大唐塔山煤矿有限公司等', -1.3, 1.2, 'end'],
    ['ln', '辽宁', '已有服务项目', 1.3, 1.7, 'start'],
    ['jl', '吉林', '吉林工程职业学院等', 1.2, -.8, 'start'],
    ['sh', '上海', '上海滴水湖学校等', 1.4, .6, 'start'],
    ['sn', '陕西', '西安市轨道交通集团有限公司等', -1.4, .6, 'end'],
    ['sc', '四川', '四川大学工会委员会等', -1.4, .6, 'end'],
    ['gd', '广东', '广州水投城市服务有限公司等', 1.4, .6, 'start'],
  ];
  const ORBIT = [
    ['菱克办公', '北京菱克办公用品有限公司', '办公日用品 · 大品牌授权与自有品牌'],
    ['贵昌工业品', '贵昌工业品（北京）有限公司', '五金工业品 · 线下门店，应急供货'],
    ['稻草森林', '北京稻草森林食品供应链科技有限公司', '绿色食材 · 新发地多温区库房与种植基地'],
    ['合创慧采', '合创慧采电商平台', '线上福利兑换 · 自研数字化采购平台'],
    ['慧采物流', '北京慧采物流有限公司', '普通货物运输 · 冷链运输'],
    ['公记餐饮', '北京公记餐饮有限公司', '食堂餐饮管理 · 物业与酒店管理'],
    ['世纪天成', '北京世纪天成咨询有限公司', '工程、货物与服务招标代理'],
    ['晟帆国旅', '北京晟帆国际旅行社有限公司', '会务文旅 · 团建党建、活动定制、商务接待', 'sf'],
  ];
  // 只列可公开的企业、高校、医院；党政机关、部队与涉密单位不上墙
  const CLIENTS = [
    ['中央企业', ['中国人寿北京市分公司', '中国华油集团有限公司', '中电科（北京）物业管理有限公司', '中国黄金集团物业管理有限公司', '中铁世纪传媒广告有限公司', '中国太平洋保险', '中国电能成套设备有限公司'], 70],
    ['高校', ['中国人民大学', '中央财经大学', '北京师范大学', '北京交通大学', '北京航空航天大学', '中国政法大学', '北京邮电大学', '华北电力大学', '北京林业大学', '中国石油大学', '首都师范大学', '北京第二外国语学院'], 84],
    ['三甲医院', ['北京大学人民医院', '北京同仁医院', '北京积水潭医院', '北京大学第三医院', '北京世纪坛医院', '中国中医科学院广安门医院'], 62],
    ['企业 · 金融 · 京外', ['北京印钞有限公司', '北京银行', '日上免税行', '北京市政路桥', '广州水投城市服务有限公司', '四川大学工会委员会', '吉林工程职业学院', '上海滴水湖学校', '西安市轨道交通集团有限公司'], 76],
  ];

  // 字体裁剪用：动态生成的宋体文字（见 tools/collect-serif-chars.js）
  window.__serifExtra = CERTS.map(c => c.name + c.lv).join('') + Object.values(LETTERS).join('') + ORBIT.map(o => o[0] + o[1]).join('')
    + CLIENTS.map(c => c[1].join('')).join('') + SERVED.map(s => s[1]).join('') + '北京总部公益图书捐赠物资捐赠证书加我微信保存到通讯录感谢信表彰信专精特新省级认定南海诸岛';

  /* ================= 纹样与小部件 ================= */
  const MICRO = '贵昌集团·晟帆国旅·GUICHANG GROUP·SHENGFAN TRAVEL·';
  $$('[data-micro]').forEach(el => { el.dataset.t = MICRO.repeat(14); });

  function ringBand(c, R, a, n, layers, pts) {
    let d = '';
    for (let k = 0; k < layers; k++) {
      const ph = k * 2 * Math.PI / layers;
      for (let i = 0; i <= pts; i++) {
        const t = i / pts * 2 * Math.PI, r = R + a * Math.sin(n * t + ph);
        d += (i ? 'L' : 'M') + (c + r * Math.cos(t)).toFixed(1) + ' ' + (c + r * Math.sin(t)).toFixed(1);
      }
    }
    return d;
  }
  function flower(c, R, r, dd, copies, pts) {
    let d = '';
    for (let k = 0; k < copies; k++) {
      const rot = k * 2 * Math.PI / (copies * (R / r));
      for (let i = 0; i <= pts; i++) {
        const t = i / pts * 2 * Math.PI;
        const x = (R + r) * Math.cos(t) - dd * Math.cos((R + r) / r * t);
        const y = (R + r) * Math.sin(t) - dd * Math.sin((R + r) / r * t);
        d += (i ? 'L' : 'M') + (c + x * Math.cos(rot) - y * Math.sin(rot)).toFixed(1) + ' ' + (c + x * Math.sin(rot) + y * Math.cos(rot)).toFixed(1);
      }
    }
    return d;
  }
  function rosette(size, color, sw, bands, fl) {
    const c = size / 2;
    let s = '';
    bands.forEach(b => { s += `<path d="${ringBand(c, b.R, b.a, b.n, b.L, b.p)}" fill="none" stroke="${color}" stroke-opacity="${b.o}" stroke-width="${sw}"/>`; });
    if (fl) s += `<path d="${flower(c, fl.R, fl.r, fl.d, fl.k, fl.p)}" fill="none" stroke="${color}" stroke-opacity="${fl.o}" stroke-width="${sw}"/>`;
    return `<svg viewBox="0 0 ${size} ${size}" aria-hidden="true">${s}</svg>`;
  }
  const ROSETTES = {
    cover: () => rosette(600, '#F1D596', .7, [{ R: 268, a: 8, n: 64, L: 5, p: 1300, o: .42 }, { R: 206, a: 34, n: 22, L: 16, p: 560, o: .3 }, { R: 116, a: 30, n: 12, L: 18, p: 340, o: .36 }], { R: 30, r: 6, d: 22, k: 6, p: 420, o: .42 }),
    cover2: () => rosette(600, '#F1D596', .6, [{ R: 206, a: 30, n: 26, L: 12, p: 640, o: .2 }, { R: 160, a: 16, n: 30, L: 8, p: 640, o: .18 }]),
    card: () => rosette(300, '#C9A35B', .6, [{ R: 134, a: 6, n: 40, L: 4, p: 800, o: .5 }, { R: 98, a: 20, n: 14, L: 12, p: 380, o: .4 }, { R: 52, a: 14, n: 8, L: 12, p: 240, o: .45 }]),
    crown: () => rosette(600, '#F1D596', .7, [{ R: 282, a: 7, n: 72, L: 5, p: 1400, o: .5 }, { R: 236, a: 26, n: 28, L: 14, p: 700, o: .32 }, { R: 170, a: 18, n: 40, L: 8, p: 700, o: .26 }]),
  };
  $$('[data-rosette]').forEach(el => { el.innerHTML = ROSETTES[el.dataset.rosette](); });
  $$('[data-wave]').forEach(el => {
    const w = 1200, h = 34; let d = '';
    for (let k = 0; k < 10; k++) {
      const ph = k * 2 * Math.PI / 10;
      for (let x = 0; x <= w; x += 3) d += (x ? 'L' : 'M') + x + ' ' + (h / 2 + 11 * Math.sin(x / w * 18 * 2 * Math.PI + ph)).toFixed(1);
    }
    el.innerHTML = `<svg viewBox="0 0 ${w} ${h}" preserveAspectRatio="none" aria-hidden="true"><path d="${d}" fill="none" stroke="#C9A35B" stroke-opacity=".38" stroke-width=".7" vector-effect="non-scaling-stroke"/></svg>`;
  });
  function barcodes(root = document) {
    $$('.barcode', root).forEach(el => {
      const r = rng(+el.dataset.seed || 7), n = +el.dataset.n || 44; let h = '';
      for (let i = 0; i < n; i++) h += `<i style="width:${1 + Math.floor(r() * 3)}px;margin-right:${1 + Math.floor(r() * 2)}px"></i>`;
      el.innerHTML = h;
    });
  }
  barcodes();
  function fakeQR(seed = 11) {
    const n = 25, r = rng(seed); let s = '';
    const inF = (x, y) => (x < 8 && y < 8) || (x >= n - 8 && y < 8) || (x < 8 && y >= n - 8);
    for (let y = 0; y < n; y++) for (let x = 0; x < n; x++) if (!inF(x, y) && r() < .48) s += `<rect x="${x}" y="${y}" width="1.02" height="1.02"/>`;
    const fin = (x, y) => `<rect x="${x}" y="${y}" width="7" height="7"/><rect x="${x + 1}" y="${y + 1}" width="5" height="5" fill="#fff"/><rect x="${x + 2}" y="${y + 2}" width="3" height="3"/>`;
    return `<svg viewBox="0 0 ${n} ${n}" shape-rendering="crispEdges" fill="#16213E" role="img" aria-label="微信二维码（占位）">${s}${fin(0, 0)}${fin(n - 7, 0)}${fin(0, n - 7)}</svg>`;
  }
  $$('[data-qr]').forEach(el => { el.outerHTML = fakeQR(); });
  // 专精特新证书角上的金色认定徽章（品牌样式，不仿任何机关印章）；也盖住原图左下角被营业执照压住的部分
  $$('[data-badge]').forEach(el => {
    el.innerHTML = `<svg viewBox="0 0 200 200" aria-hidden="true">
      <defs><linearGradient id="bdg" x1="0" y1="0" x2="1" y2="1"><stop offset="0" stop-color="#FFF1C6"/><stop offset=".35" stop-color="#E9C982"/><stop offset=".7" stop-color="#B8914C"/><stop offset="1" stop-color="#8B6526"/></linearGradient>
      <path id="bdr" d="M100 100 m-70 0 a70 70 0 1 1 140 0 a70 70 0 1 1 -140 0"/></defs>
      <circle cx="100" cy="100" r="96" fill="url(#bdg)"/><circle cx="100" cy="100" r="88" fill="#0B2A71"/>
      <circle cx="100" cy="100" r="84" fill="none" stroke="#E9C982" stroke-width="1.2"/><circle cx="100" cy="100" r="56" fill="none" stroke="#E9C982" stroke-width="1"/>
      <path d="${ringBand(100, 74, 3, 36, 3, 500)}" fill="none" stroke="#E9C982" stroke-opacity=".35" stroke-width=".6"/>
      <text font-family="JetBrains Mono, monospace" font-size="10.5" fill="#F1D596"><textPath href="#bdr" textLength="432" lengthAdjust="spacing">SPECIALIZED · REFINED · DISTINCTIVE · INNOVATIVE ·</textPath></text>
      <text x="100" y="94" text-anchor="middle" font-family="Noto Serif SC, serif" font-weight="900" font-size="25" fill="#F1D596">专精特新</text>
      <text x="100" y="122" text-anchor="middle" font-family="Noto Serif SC, serif" font-weight="700" font-size="15" letter-spacing="4" fill="#E9C982">省级认定</text></svg>`;
  });

  /* ================= 翻牌 ================= */
  function flapEl(ch, cls = '') {
    const s = document.createElement('span');
    s.className = 'flap ' + cls; s.dataset.ch = ch;
    s.setAttribute('aria-hidden', 'true');
    s.innerHTML = `<span class="h t"><b>${ch}</b></span><span class="h b"><b>${ch}</b></span><span class="leaf"><span class="h t f"><b>${ch}</b></span><span class="h b k"><b>${ch}</b></span></span>`;
    return s;
  }
  function flapTo(el, ch, dur = 380) {
    const old = el.dataset.ch;
    if (old === ch) return Promise.resolve();
    const [t, b, f, k] = el.querySelectorAll('b');
    const leaf = el.querySelector('.leaf');
    t.textContent = ch; b.textContent = old; f.textContent = old; k.textContent = ch;
    el.style.setProperty('--dur', dur + 'ms');
    leaf.classList.remove('go'); void leaf.offsetWidth; leaf.classList.add('go');
    el.dataset.ch = ch;
    return new Promise(r => setTimeout(() => { b.textContent = ch; f.textContent = ch; leaf.classList.remove('go'); r(); }, dur + 30));
  }
  const setNow = (el, ch) => { el.dataset.ch = ch; el.querySelectorAll('b').forEach(b => { b.textContent = ch; }); };

  const WORDS = ['办公用品', '五金工业', '绿色食材', '福利兑换', '会议会展', '参访研学', '标杆考察', '商务出行'];
  const POOL = [...WORDS.join('')];
  const heroFlaps = [...WORDS[0]].map(ch => { const e = flapEl(ch); $('#heroFlaps').appendChild(e); return e; });
  let wi = 0, busy = false;
  setInterval(async () => {
    if (busy || scrollY > innerHeight || document.hidden) return;
    busy = true; wi = (wi + 1) % WORDS.length;
    const w = [...WORDS[wi]];
    await Promise.all(heroFlaps.map(async (el, i) => {
      await wait(i * 90);
      if (reduce) { setNow(el, w[i]); return; }
      await flapTo(el, POOL[Math.floor(Math.random() * POOL.length)], 150);
      await flapTo(el, w[i], 260);
    }));
    busy = false;
  }, 2800);

  const DIG = '0123456789';
  $$('.flaps[data-val]').forEach(box => {
    box.setAttribute('aria-label', box.dataset.val);
    box._f = [...box.dataset.val].map(ch => { const e = flapEl(' ', /[一-鿿]/.test(ch) ? 'sm cjk' : 'sm'); box.appendChild(e); return e; });
  });
  function runBoard(board) {
    $$('.flaps[data-val]', board).forEach((box, row) => {
      const v = [...box.dataset.val];
      box._f.forEach(async (el, i) => {
        if (reduce) { setNow(el, v[i]); return; }
        await wait(250 + row * 170 + i * 80);
        const steps = 4 + Math.floor(Math.random() * 4);
        for (let s = 0; s < steps; s++) await flapTo(el, DIG[Math.floor(Math.random() * 10)], 110);
        await flapTo(el, v[i], 170);
      });
    });
  }

  /* ================= 01 服务网络：点阵地图 ================= */
  const mapResize = (function map() {
    const D = window.CN_DOTS, fig = $('#nwMap');
    if (!D || !fig) return () => {};
    const cv = $('canvas', fig), svg = $('svg', fig), ctx = cv.getContext('2d');
    const VB = { x: -1.2, y: -.5, w: 70.6, h: 50 };
    const NS = 'http://www.w3.org/2000/svg';
    const mk = (tag, attrs, parent) => { const e = document.createElementNS(NS, tag); for (const k in attrs) e.setAttribute(k, attrs[k]); if (parent) parent.appendChild(e); return e; };
    const C = D.city, BJ = C.bj;
    const dots = [];
    for (let r = 0; r < D.rows; r++) for (let c = 0; c < D.cols; c++) { const l = +D.lv[r * D.cols + c]; if (l) dots.push({ x: c + .5, y: r + .5, l }); }
    dots.push({ x: C.diaoyu[0], y: C.diaoyu[1], l: 1 }, { x: C.chiwei[0], y: C.chiwei[1], l: 1 });
    const jr = rng(5);
    dots.forEach(d => {
      let h = Math.exp(-((d.x - BJ[0]) ** 2 + (d.y - BJ[1]) ** 2) / (2 * 3.6 ** 2));
      SERVED.forEach(([k]) => { const p = C[k]; h = Math.max(h, .78 * Math.exp(-((d.x - p[0]) ** 2 + (d.y - p[1]) ** 2) / (2 * 2.3 ** 2))); });
      d.h = h; d.t0 = Math.hypot(d.x - BJ[0], d.y - BJ[1]) * 24 + jr() * 140;
    });
    const T_DOTS = Math.max(...dots.map(d => d.t0)) + 480;
    const gArcs = $('.arcs', svg), gComets = $('.comets', svg), gNodes = $('.nodes', svg), gInset = $('.nw-inset', svg);
    const arcs = SERVED.map(([k], i) => {
      const p = C[k], mx = (BJ[0] + p[0]) / 2, my = (BJ[1] + p[1]) / 2, dx = p[0] - BJ[0], dy = p[1] - BJ[1], L = Math.hypot(dx, dy);
      let nx = -dy / L, ny = dx / L; if (ny > 0) { nx = -nx; ny = -ny; }
      const bulge = Math.min(7, L * .3 + .6);
      const path = mk('path', { class: 'arc', d: `M${BJ[0]} ${BJ[1]} Q${(mx + nx * bulge).toFixed(2)} ${(my + ny * bulge).toFixed(2)} ${p[0]} ${p[1]}` }, gArcs);
      const len = path.getTotalLength();
      path.style.strokeDasharray = path.style.strokeDashoffset = len;
      const comet = mk('circle', { class: 'comet', r: .42, cx: BJ[0], cy: BJ[1], opacity: 0 }, gComets);
      return { path, len, comet, ph: i * .137 };
    });
    const labels = [];
    const label = (x, y, txt, anchor, cls) => { const t = mk('text', { x, y, 'text-anchor': anchor, class: 'lbl' + (cls ? ' ' + cls : '') }, gNodes); t.textContent = txt; labels.push(t); };
    SERVED.forEach(([k, name, , dx, dy, anchor]) => {
      const p = C[k], g = mk('g', { class: 'nd', 'data-k': k, transform: `translate(${p[0]} ${p[1]})` }, gNodes);
      mk('circle', { class: 'hit', r: 2.4 }, g); mk('circle', { class: 'ring', r: .9 }, g); mk('circle', { class: 'core', r: .42 }, g);
      label(p[0] + dx, p[1] + dy, name, anchor);
    });
    const hq = mk('g', { class: 'nd hq sel', 'data-k': 'bj', transform: `translate(${BJ[0]} ${BJ[1]})` }, gNodes);
    mk('circle', { class: 'pulse', r: 1 }, hq); mk('circle', { class: 'pulse b', r: 1 }, hq); mk('circle', { class: 'hit', r: 2.6 }, hq);
    const star = Array.from({ length: 10 }, (_, i) => { const a = -Math.PI / 2 + i * Math.PI / 5, r = i % 2 ? .62 : 1.45; return (i ? 'L' : 'M') + (r * Math.cos(a)).toFixed(3) + ' ' + (r * Math.sin(a)).toFixed(3); }).join('') + 'Z';
    mk('path', { class: 'star', d: star }, hq);
    label(BJ[0] - 1.5, BJ[1] - 2.3, '北京 · 总部', 'end', 'hq');

    // 南海诸岛小框
    const IN = D.inset, iw = 8.4, ih = iw / IN.aspect, ix = 68.7 - iw, iy = 49.1 - ih, ipx = IN.pitch * iw;
    mk('rect', { x: ix, y: iy, width: iw, height: ih, rx: .3 }, gInset);
    for (let r = 0; r < IN.rows; r++) for (let c = 0; c < IN.cols; c++) {
      const l = +IN.lv[r * IN.cols + c]; if (!l) continue;
      mk('circle', { cx: (ix + IN.ox * iw + (c + .5) * ipx).toFixed(3), cy: (iy + IN.oy * ih + (r + .5) * ipx).toFixed(3), r: ((l === 2 ? .36 : .24) * ipx).toFixed(3), fill: 'rgba(11,42,113,.55)' }, gInset);
    }
    IN.parts.forEach(([kind, cx, cy, len, ang]) => {
      const x = ix + cx * iw, y = iy + cy * ih;
      if (kind === 'isle') mk('circle', { cx: x, cy: y, r: .13, fill: 'rgba(11,42,113,.7)' }, gInset);
      else { const L = Math.max(len * iw, .32) / 2; mk('line', { class: 'dash', x1: x - Math.cos(ang) * L, y1: y - Math.sin(ang) * L, x2: x + Math.cos(ang) * L, y2: y + Math.sin(ang) * L }, gInset); }
    });
    const insetLbl = mk('text', { x: ix + iw - .5, y: iy + ih - .6, 'text-anchor': 'end' }, gInset); insetLbl.textContent = '南海诸岛';

    let scale = 1, dpr = 1, started = 0, raf = 0;
    const easeBack = t => 1 + 2.6 * Math.pow(t - 1, 3) + 1.6 * Math.pow(t - 1, 2);
    function draw(t) {
      const r = fig.getBoundingClientRect(), ox = (r.width - VB.w * scale) / 2, oy = (r.height - VB.h * scale) / 2;
      ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
      ctx.clearRect(0, 0, r.width, r.height);
      for (const d of dots) {
        const a = clamp((t - d.t0) / 480); if (a <= 0) continue;
        const rad = (d.l === 2 ? .35 : .23) * (1 + .28 * d.h) * scale * easeBack(a);
        if (rad <= 0) continue;
        ctx.fillStyle = `rgba(11,42,113,${((.3 + .62 * d.h) * clamp(a * 2.5)).toFixed(3)})`;
        ctx.beginPath(); ctx.arc(ox + (d.x - VB.x) * scale, oy + (d.y - VB.y) * scale, rad, 0, 6.2832); ctx.fill();
      }
    }
    function loop(now) {
      const t = now - started;
      if (t < T_DOTS + 60) draw(t);
      arcs.forEach((a, i) => {
        const k = clamp((t - 700 - i * 110) / 1100);
        a.path.style.strokeDashoffset = (a.len * (1 - ease(k))).toFixed(2);
        if (k >= 1) {
          const ph = (t / 2600 + a.ph) % 1, pt = a.path.getPointAtLength(a.len * ph);
          a.comet.setAttribute('cx', pt.x.toFixed(2)); a.comet.setAttribute('cy', pt.y.toFixed(2));
          a.comet.setAttribute('opacity', (Math.sin(ph * Math.PI) * .95).toFixed(2));
        }
      });
      raf = requestAnimationFrame(loop);
    }
    function start() {
      if (started) return;
      started = performance.now(); fig.classList.add('on');
      if (reduce) { draw(1e9); arcs.forEach(a => { a.path.style.strokeDashoffset = 0; }); return; }
      raf = requestAnimationFrame(loop);
    }
    new IntersectionObserver((es, io) => es.forEach(e => { if (e.isIntersecting) { start(); io.disconnect(); } }), { threshold: .3 }).observe(fig);
    document.addEventListener('visibilitychange', () => {
      if (document.hidden) { cancelAnimationFrame(raf); raf = 0; } else if (started && !reduce && !raf) raf = requestAnimationFrame(loop);
    });
    const info = $('#nwCard');
    const INFO = Object.fromEntries(SERVED.map(([k, n, s]) => [k, [n, s]]));
    INFO.bj = ['北京 · 总部', '丰台区新华双创园 · 机关、高校、医院与央国企客户集中'];
    gNodes.addEventListener('click', e => {
      const g = e.target.closest('.nd'); if (!g) return;
      $$('.nd', gNodes).forEach(n => n.classList.toggle('sel', n === g));
      const [n, s] = INFO[g.dataset.k];
      $('.pin', info).textContent = n[0]; $('b', info).textContent = n; $('span', info).textContent = s;
      info.classList.remove('flash'); void info.offsetWidth; info.classList.add('flash');
    });
    function resize() {
      const r = fig.getBoundingClientRect();
      dpr = Math.min(2.5, devicePixelRatio || 1);
      cv.width = Math.round(r.width * dpr); cv.height = Math.round(r.height * dpr);
      scale = Math.min(r.width / VB.w, r.height / VB.h) || 1;
      const fs = Math.max(1.4, 11.5 / scale);
      labels.forEach(t => { t.setAttribute('font-size', fs.toFixed(2)); t.setAttribute('stroke-width', (3.2 / scale).toFixed(3)); });
      insetLbl.setAttribute('font-size', Math.max(.9, 8 / scale).toFixed(2));
      arcs.forEach(a => { a.path.setAttribute('stroke-width', (1.7 / scale).toFixed(3)); a.comet.setAttribute('r', Math.max(.42, 2.6 / scale).toFixed(3)); });
      draw(started ? performance.now() - started : (reduce ? 1e9 : 0));
    }
    resize();
    return resize;
  })();

  /* ================= 02 资质：31 张证书砌成门槛 ================= */
  const wall = (function certWall() {
    const sec = $('#credentials'), wallEl = $('#wall'), crown = $('#crown'), dim = $('#qwDim'), foot = $('#qwFoot');
    const num = $('#qwN'), hint = $('#qwHint'), cta = $('#qwCta'), ip = $('.ip', foot);
    const DUR = .13, TOTAL = CERTS.filter(c => c.g !== 'ip').length;
    const ROT = [22, -24, -7]; // 与 CSS 里 .qw-wall 的 rotateX / rotateY / rotateZ 一致
    let tiles = [], lastP = -1;
    function matrix() {
      const [ax, ay, az] = ROT.map(d => d * Math.PI / 180);
      const Rx = [[1, 0, 0], [0, Math.cos(ax), -Math.sin(ax)], [0, Math.sin(ax), Math.cos(ax)]];
      const Ry = [[Math.cos(ay), 0, Math.sin(ay)], [0, 1, 0], [-Math.sin(ay), 0, Math.cos(ay)]];
      const Rz = [[Math.cos(az), -Math.sin(az), 0], [Math.sin(az), Math.cos(az), 0], [0, 0, 1]];
      const mul = (A, B) => A.map((row, i) => B[0].map((_, j) => row.reduce((s, _, k) => s + A[i][k] * B[k][j], 0)));
      return mul(mul(Rx, Ry), Rz);
    }
    function build() {
      const desk = desktop.matches;
      const cfg = desk ? { cols: 7, tw: 150, gap: 20 } : { cols: 5, tw: 104, gap: 13 };
      const r = rng(11), order = CERTS.slice().sort(() => r() - .5);
      const cols = Array.from({ length: cfg.cols }, () => ({ items: [], h: 0 }));
      order.forEach(c => {
        const col = cols.reduce((a, b) => (b.h < a.h ? b : a));
        const th = Math.round(cfg.tw * c.h / c.w);
        col.items.push({ c, y: col.h, th }); col.h += th + cfg.gap;
      });
      const H = Math.max(...cols.map(c => c.h)), W = cfg.cols * cfg.tw + (cfg.cols - 1) * cfg.gap;
      wallEl.style.width = W + 'px'; wallEl.style.height = H + 'px';
      wallEl.innerHTML = ''; tiles = [];
      const rr = rng(29), R = matrix(), vw = innerWidth, vh = innerHeight;
      // 散落位置先在屏幕空间里取（铺满视野、远近不一），再用墙面旋转的逆矩阵换回墙面坐标
      const toLocal = (X, Y, Z) => [R[0][0] * X + R[1][0] * Y + R[2][0] * Z, R[0][1] * X + R[1][1] * Y + R[2][1] * Z, R[0][2] * X + R[1][2] * Y + R[2][2] * Z];
      cols.forEach((col, ci) => {
        const el = document.createElement('div');
        el.className = 'qw-col' + (ci % 2 ? ' down' : '');
        el.style.left = ci * (cfg.tw + cfg.gap) + 'px';
        el.style.width = cfg.tw + 'px';
        el.style.setProperty('--h', col.h + 'px');
        el.style.setProperty('--dur', (col.h / (desk ? 15 : 11)).toFixed(1) + 's');
        el.style.animationDelay = (-rr() * 20).toFixed(1) + 's';
        for (let copy = -1; copy <= 1; copy++) {
          col.items.forEach(it => {
            const t = document.createElement('figure');
            const top = it.y + copy * col.h;
            t.className = 'qc' + (it.c.g === 'rec' ? ' top' : '');
            t.style.cssText = `top:${top}px;width:${cfg.tw}px;height:${it.th}px;--sd:${(rr() * 7).toFixed(1)}s`;
            t.dataset.cert = it.c.id;
            t.innerHTML = `<img src="${IMG}cert/s/${it.c.img}.jpg" alt="" loading="${copy === 0 ? 'eager' : 'lazy'}" decoding="async">`;
            el.appendChild(t);
            // 从下往上砌：越靠下越早落位
            const yc = clamp((top + it.th / 2) / H, -.4, 1.4);
            const s = .03 + .42 * clamp((1.25 - yc) / 1.5) + rr() * .045;
            const Z = -300 - Math.pow(rr(), .8) * 2300, f = (1000 - Z) / 1000;
            const X = (rr() - .5) * vw * 1.15 * f + (desk ? vw * .16 * f : 0), Y = (rr() - .58) * vh * .95 * f;
            const [lx, ly, lz] = toLocal(X, Y, Z);
            const ox = ci * (cfg.tw + cfg.gap) + cfg.tw / 2 - W / 2, oy = top + it.th / 2 - H / 2;
            tiles.push({ el: t, s, sx: lx - ox, sy: ly - oy, sz: lz, rx: (rr() - .5) * 70, ry: (rr() - .5) * 90, rz: (rr() - .5) * 60, g: -1 });
          });
        }
        wallEl.appendChild(el);
      });
      lastP = -1;
    }
    function render(p) {
      for (const t of tiles) {
        const l = clamp((p - t.s) / DUR), k = 1 - ease(l);
        t.el.style.transform = k > .001
          ? `translate3d(${(t.sx * k).toFixed(1)}px,${(t.sy * k).toFixed(1)}px,${(t.sz * k).toFixed(1)}px) rotateX(${(t.rx * k).toFixed(1)}deg) rotateY(${(t.ry * k).toFixed(1)}deg) rotateZ(${(t.rz * k).toFixed(1)}deg)`
          : 'none';
        t.el.style.opacity = (.32 + .68 * clamp(l * 1.8)).toFixed(3);
        const g = Math.round(clamp(1 - Math.abs(p - (t.s + DUR)) / .028) * 20) / 20;
        if (g !== t.g) { t.g = g; t.el.style.setProperty('--g', g); }
      }
      // 计数跟着墙面砌筑的进度走
      num.textContent = Math.round(TOTAL * smooth(clamp((p - .06) / .46)));
      const c = smooth(clamp((p - .6) / .15)), s = smooth(clamp((p - .74) / .14));
      crown.style.setProperty('--c', c.toFixed(3));
      crown.classList.toggle('on', c > .6);
      dim.style.setProperty('--c', c.toFixed(3));
      foot.style.setProperty('--s', s.toFixed(3));
      ip.style.setProperty('--s', smooth(clamp((p - .44) / .1)).toFixed(2));
      cta.classList.toggle('on', s > .6);
      hint.style.opacity = (1 - clamp(p / .12)).toFixed(2);
    }
    function tick() {
      const r = sec.getBoundingClientRect();
      if (r.bottom < -50 || r.top > innerHeight + 50) return;
      const p = reduce ? 1 : clamp(-r.top / (sec.offsetHeight - innerHeight));
      if (Math.abs(p - lastP) < 1e-4) return;
      lastP = p; render(p);
    }
    const setHint = () => { hint.textContent = (desktop.matches ? '向下滚动' : '往上滑') + '，看 31 张证书一张张砌成门槛'; };
    setHint(); build(); render(reduce ? 1 : 0);
    return { tick, rebuild() { setHint(); build(); tick(); } };
  })();

  /* ================= 原件查看器：证书、感谢信、活动照片 ================= */
  const viewer = (function () {
    const vw = $('#vw'), track = $('#vwTrack'), dotsEl = $('#vwDots'), idx = $('#vwIdx'), tabsEl = $('#vwTabs');
    const ORDER = ['rec', 'lic', 'sys', 'ip'];
    const certItem = c => ({
      id: c.id, g: c.g, img: `${IMG}cert/${c.img}.jpg`, tag: c.lv, title: c.name,
      sub: `${GROUPS[c.g]} · 持证单位：${c.who === 'sf' ? SF : GC}`, kv: c.kv, verify: c.verify,
    });
    const SETS = {
      certs: () => ORDER.flatMap(g => CERTS.filter(c => c.g === g)).map(certItem),
      olympics: () => ['l1', 'l3', 'l4'].map(letter),
      covid: () => ['l9'].map(letter),
      charity: () => [photo('ev-charity', '公益童行 · 筑梦林西', '图书捐赠活动现场'), photo('ev-charity2', '物资捐赠', '捐赠物资清点现场'), { ...photo('ev-donate', '捐赠证书', '北京丰宣福台合作发展基金会颁发'), photo: false }],
    };
    let items = [], slides = [], dotEls = [];
    function slideHTML(it) {
      const rows = (it.kv || []).map(([k, v]) => `<div><dt>${k}</dt><dd>${v}</dd></div>`).join('');
      const ver = it.verify ? `<a class="verify" href="${it.verify[1]}" target="_blank" rel="noopener">官方核验 · ${it.verify[0]}<svg><use href="#i-out"/></svg></a>` : '';
      return `<div class="vw-s"><div class="pic${it.photo ? ' photo' : ''}"><img data-src="${it.img}" alt="${it.title}"></div>
        <div class="meta"><em>${it.tag}</em><b>${it.title}</b><span>${it.sub}</span>${rows ? `<dl class="kv">${rows}</dl>` : ''}${ver}</div></div>`;
    }
    function fill(list) {
      items = list;
      track.innerHTML = list.map(slideHTML).join('');
      dotsEl.innerHTML = list.length > 1 ? list.map(() => '<i></i>').join('') : '';
      slides = [...track.children]; dotEls = [...dotsEl.children];
    }
    function load(i) { for (let k = i - 1; k <= i + 1; k++) { const im = slides[k] && $('img[data-src]', slides[k]); if (im) { im.src = im.dataset.src; im.removeAttribute('data-src'); } } }
    function setIdx(i) {
      idx.textContent = String(i + 1).padStart(2, '0') + ' / ' + String(items.length).padStart(2, '0');
      dotEls.forEach((d, k) => d.classList.toggle('on', k === i));
      const g = items[i] && items[i].g;
      $$('button', tabsEl).forEach(b => b.classList.toggle('on', b.dataset.g === g));
      load(i);
    }
    track.addEventListener('scroll', () => setIdx(clamp(Math.round(track.scrollLeft / track.clientWidth), 0, items.length - 1)), { passive: true });
    tabsEl.addEventListener('click', e => {
      const b = e.target.closest('[data-g]'); if (!b) return;
      const i = items.findIndex(it => it.g === b.dataset.g);
      if (i >= 0) track.scrollTo({ left: i * track.clientWidth, behavior: reduce ? 'auto' : 'smooth' });
    });
    function open(set, startId) {
      fill(SETS[set]());
      tabsEl.innerHTML = set === 'certs' ? ORDER.map(g => `<button type="button" data-g="${g}">${GROUPS[g]} ${CERTS.filter(c => c.g === g).length}</button>`).join('') : '';
      const i = Math.max(0, items.findIndex(it => it.id === startId));
      vw.classList.add('on'); document.documentElement.style.overflow = 'hidden';
      requestAnimationFrame(() => { track.scrollLeft = i * track.clientWidth; setIdx(i); });
    }
    function close() { vw.classList.remove('on'); document.documentElement.style.overflow = ''; }
    $('#vwX').addEventListener('click', close);
    addEventListener('keydown', e => { if (e.key === 'Escape' && vw.classList.contains('on')) close(); });
    return { open };
  })();

  /* ================= 03 合创慧采：手机里轮播真实界面 ================= */
  (function phone() {
    const imgs = $$('#phone img'), cap = $('#phoneCap span'), dots = $('#phoneDots');
    if (!imgs.length) return;
    dots.innerHTML = imgs.map(() => '<i></i>').join('');
    const ds = [...dots.children];
    let i = 0, timer = 0;
    const show = k => { i = k; imgs.forEach((im, j) => im.classList.toggle('on', j === k)); ds.forEach((d, j) => d.classList.toggle('on', j === k)); cap.textContent = imgs[k].dataset.cap; };
    show(0);
    if (reduce) return;
    new IntersectionObserver(es => es.forEach(e => {
      clearInterval(timer);
      if (e.isIntersecting) timer = setInterval(() => show((i + 1) % imgs.length), 2600);
    }), { threshold: .3 }).observe($('#phone'));
  })();

  /* ================= 03 集团星图 ================= */
  (function orbit() {
    const box = $('#orbit'), cardEl = $('#orbitCard');
    const R = 38, N = ORBIT.length;
    const pos = ORBIT.map((_, i) => { const a = -Math.PI / 2 + i * 2 * Math.PI / N; return [50 + R * Math.cos(a), 50 + R * Math.sin(a)]; });
    box.innerHTML = `<svg viewBox="0 0 100 100" aria-hidden="true"><circle class="ring" cx="50" cy="50" r="47"/><circle class="ring d" cx="50" cy="50" r="${R}"/>
      ${pos.map(([x, y], i) => `<line class="ray" data-i="${i}" x1="50" y1="50" x2="${x.toFixed(2)}" y2="${y.toFixed(2)}"/>`).join('')}</svg>
      <div class="orbit-core"><span class="mk" role="img" aria-label="贵昌集团"></span><small>GUICHANG GROUP</small></div>
      ${ORBIT.map(([s, , , sf], i) => `<button type="button" class="sat${sf ? ' sf' : ''}" data-i="${i}" style="left:${pos[i][0].toFixed(2)}%;top:${pos[i][1].toFixed(2)}%"><i>${String(i + 1).padStart(2, '0')}</i><b>${s}</b></button>`).join('')}`;
    const show = i => {
      $$('.sat', box).forEach(b => b.classList.toggle('on', +b.dataset.i === i));
      $$('.ray', box).forEach(l => l.classList.toggle('on', +l.dataset.i === i));
      const [, name, desc] = ORBIT[i];
      cardEl.innerHTML = `<b>${name}</b><span>${desc}</span>`;
    };
    box.addEventListener('click', e => { const b = e.target.closest('.sat'); if (b) show(+b.dataset.i); });
    show(0);
  })();

  /* ================= 04 客户：铭牌滚动墙 ================= */
  (function marquee() {
    const box = $('.mq');
    box.innerHTML = CLIENTS.map(([g, names, d], i) => {
      const one = `<span class="g">${g}</span>` + names.map(n => `<span>${n}</span>`).join('');
      return `<div class="mq-row${i % 2 ? ' rev' : ''}" style="--d:${d}s">${one}${one}</div>`;
    }).join('');
    $$('.mq-row', box).forEach((row, i) => { if (i) row.lastElementChild.setAttribute('aria-hidden', 'true'); });
  })();

  /* ================= 弹层 / 提示 ================= */
  const sheet = $('#sheet'), sheetBody = $('#sheetBody');
  function openSheet(html) {
    sheetBody.innerHTML = html;
    const h = $('h3', sheetBody); if (h) h.id = 'sheetTitle';
    if (typeof sheet.showModal === 'function') sheet.showModal(); else sheet.setAttribute('open', '');
    sheet.scrollTop = 0;
  }
  sheet.addEventListener('click', e => { if (e.target === sheet) sheet.close(); });
  sheet.addEventListener('close', () => sheetBody.classList.remove('in'));
  let toastT;
  function toast(msg) {
    if (!msg) return;
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
  function openRoster() {
    openSheet($('#tpl-roster').innerHTML);
    barcodes(sheetBody);
    requestAnimationFrame(() => requestAnimationFrame(() => sheetBody.classList.add('in')));
  }
  function vcard() {
    const n = card.name || '';
    return ['BEGIN:VCARD', 'VERSION:3.0', `N:${n.slice(0, 1)};${n.slice(1)};;;`, `FN:${n}`,
      'ORG:贵昌集团有限公司', `TITLE:${card.titleGc || ''}`,
      card.mobile && `TEL;TYPE=CELL:${card.mobile}`, `TEL;TYPE=WORK:${card.tel}`,
      card.email && `EMAIL;TYPE=INTERNET:${card.email}`,
      `ADR;TYPE=WORK:;;${card.address};;;;`, `NOTE:贵昌集团 · 晟帆国旅 · 集团热线 ${card.hotline}`,
      `URL:${location.href.split('#')[0]}`, 'END:VCARD'].filter(Boolean).join('\r\n');
  }
  const ACTIONS = {
    call() { if (card.mobile) location.href = 'tel:' + card.mobile; else toast('手机号待提供，可先拨座机 010-5339 8618'); },
    wechat() {
      openSheet(`<h3>加我微信</h3><p class="sub">WECHAT · 长按识别二维码</p><div class="qr">${fakeQR()}</div>
        <dl class="kv" style="margin-top:18px"><div><dt>微信号</dt><dd>${card.wechat || '待提供'}</dd></div></dl>
        <button class="btn" type="button" data-act="copy-wechat">复制微信号</button><p class="hint">二维码为占位，拿到真实微信二维码后替换。</p>`);
    },
    'copy-wechat'() { copy(card.wechat, '微信号'); },
    'copy-email'() { copy(card.email, '邮箱'); },
    'copy-link'() { copy(location.href.split('#')[0], '链接'); },
    roster: openRoster,
    vcard() {
      if (inWeChat) {
        openSheet(`<h3>保存到通讯录</h3><p class="sub">SAVE CONTACT</p><p style="margin:0;font-size:14px;line-height:1.8;color:var(--ink2)">微信里不能直接保存文件。请点右上角「···」，选「在浏览器打开」，再点一次「存通讯录」。</p><button class="btn ghost" type="button" data-act="copy-link">复制本页链接</button>`);
        return;
      }
      const a = document.createElement('a');
      a.href = URL.createObjectURL(new Blob([vcard()], { type: 'text/vcard;charset=utf-8' }));
      a.download = (card.name || '名片') + '.vcf';
      document.body.appendChild(a); a.click(); a.remove();
      setTimeout(() => URL.revokeObjectURL(a.href), 1000);
      toast('已生成名片文件');
    },
  };
  document.addEventListener('click', e => {
    const act = e.target.closest('[data-act]');
    if (act && ACTIONS[act.dataset.act]) { e.preventDefault(); ACTIONS[act.dataset.act](); return; }
    const gal = e.target.closest('[data-gallery]');
    if (gal) { viewer.open(gal.dataset.gallery); return; }
    const cert = e.target.closest('[data-cert]');
    if (cert) viewer.open('certs', cert.dataset.cert);
  });

  /* ================= 进场 ================= */
  const io = new IntersectionObserver(es => es.forEach(e => {
    if (!e.isIntersecting) return;
    e.target.classList.add('in'); io.unobserve(e.target);
    if (e.target.matches('.board')) runBoard(e.target);
  }), { threshold: .15, rootMargin: '0px 0px -6% 0px' });
  $$('[data-rv]').forEach(el => io.observe(el));

  /* ================= 导航高亮 ================= */
  const links = $$('.nav-links a');
  const spy = new IntersectionObserver(es => es.forEach(e => {
    if (!e.isIntersecting) return;
    links.forEach(a => a.classList.toggle('on', a.getAttribute('href') === '#' + e.target.id));
  }), { rootMargin: '-45% 0px -50% 0px' });
  $$('main > section[id]').forEach(s => spy.observe(s));

  /* ================= 滚动：封面压暗、证书墙、导航 / 联系条 ================= */
  const cv = $('.cv'), dock = $('#dock'), nav = $('#nav'), contact = $('#contact'), creds = $('#credentials'), clients = $('#clients'), cue = $('.cv-cue span');
  const setCue = () => { cue.textContent = desktop.matches ? '向下翻开' : '上滑翻开'; };
  setCue();
  let ticking = false;
  function onScroll() {
    if (ticking) return; ticking = true;
    requestAnimationFrame(() => {
      const vh = innerHeight, y = scrollY;
      cv.style.setProperty('--p', Math.min(1, y / vh).toFixed(3));
      cv.style.visibility = y > vh * 1.05 ? 'hidden' : '';
      nav.classList.toggle('show', y > vh * .8);
      // 导航压在深色章节上时切成深色
      nav.classList.toggle('dark', [creds, clients].some(el => { const r = el.getBoundingClientRect(); return r.top < 66 && r.bottom > 66; }));
      // 证书墙那一屏是沉浸式的，联系条先收起，免得挡住「查看全部证书原件」
      const q = creds.getBoundingClientRect(), inWall = q.top < vh * .5 && q.bottom > vh * .5;
      dock.classList.toggle('show', y > vh * .55 && !inWall && contact.getBoundingClientRect().top > vh * .55);
      wall.tick();
      ticking = false;
    });
  }
  addEventListener('scroll', onScroll, { passive: true });
  let lastW = innerWidth;
  addEventListener('resize', () => {
    if (Math.abs(innerWidth - lastW) > 40) { lastW = innerWidth; setCue(); wall.rebuild(); }
    mapResize(); onScroll();
  });
  onScroll();
})();
