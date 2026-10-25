/* 京华研学导览图：舆图式的北京导览底图（示意，不按比例，不画行政界线）
   画法照清代舆图：平面上画河湖、道路、城墙，山和地标用立面小画摆上去。
   - 西山、军都山、燕山从西往北再往东围成半圈，平原往东南敞开；长城顺着北边山脊走。
   - 永定河、潮白河、北运河、拒马河往东南流；官厅水库、密云水库。
   - 城中心画明清北京城"凸"字城墙和中轴线，外面三环到六环几圈淡虚线。
   - 区只写名字，不画界线。老城里的点太密，单开一块"老城放大"。
   坐标：画面 1600 × 1120，x 往东、y 往南（屏幕坐标）；山和图标是局部 v 向上的小画，放上去时翻一下。 */
(function () {
  'use strict';
  const J = window.JH, { N, P2, Pen, smooth, rng } = J, K = J.ICONS._kit;
  const W = 1600, H = 1120;

  const CATS = {
    red: { name: '红色教育', color: '#b3362b' },
    gold: { name: '文化研学', color: '#a77a24' },
    green: { name: '团建拓展', color: '#4f8a52' },
    blue: { name: '会议场地', color: '#2f5f86' },
  };
  // 景点：[id, 名称, 区, 类别, 研学主题, 一句话, 适合, [x, y] 主图位置（老城里的点为 null）, 老城放大图里的位置]
  const SPOTS = [
    ['xiangshan', '香山革命纪念地', '海淀区', 'red', ['党史学习'], '1949 年中共中央进驻香山，在这里指挥渡江战役、筹建新中国，有双清别墅等革命旧址。', ['主题党日', '现场教学'], [500, 612]],
    ['honglou', '北大红楼', '东城区', 'red', ['党史学习'], '新文化运动和中国共产党早期北京革命活动旧址，现为北京新文化运动纪念馆。', ['主题党日', '现场教学'], null, [1458, 860]],
    ['lidazhao', '李大钊故居', '西城区', 'red', ['党史学习'], '李大钊在北京居住时间最长的一处寓所。', ['现场教学', '党史学习'], null, [1270, 926]],
    ['dangshi', '中国共产党历史展览馆', '朝阳区', 'red', ['党史学习'], '系统展示中国共产党百年奋斗历程的展览馆。', ['党史学习', '主题党日'], [1010, 566]],
    ['lugou', '卢沟桥 · 抗战纪念馆', '丰台区', 'red', ['党史学习', '国防教育'], '全民族抗战爆发地，中国人民抗日战争纪念馆在宛平城内，就在公司所在的丰台区。', ['现场教学', '爱国主义教育'], [582, 876]],
    ['pingxi', '平西抗日战争纪念馆', '房山区', 'red', ['党史学习', '国防教育'], '记述平西抗日根据地斗争历史的纪念馆，地处十渡山区。', ['现场教学'], [300, 958]],
    ['jiaozhuanghu', '焦庄户地道战遗址', '顺义区', 'red', ['国防教育'], '保留抗战时期地道遗址的纪念馆，可下地道参观。', ['现场教学', '国防教育'], [1240, 450]],
    ['gugong', '故宫', '东城区', 'gold', ['传统文化'], '明清两代皇宫，北京中轴线的核心。', ['文化研学'], null, [1370, 926]],
    ['tiantan', '天坛', '东城区', 'gold', ['传统文化'], '明清皇帝祭天、祈谷的场所。', ['文化研学'], null, [1452, 1030]],
    ['yiheyuan', '颐和园', '海淀区', 'gold', ['传统文化'], '清代皇家园林，万寿山、昆明湖一山一水。', ['文化研学', '健步走'], [560, 520]],
    ['yuanmingyuan', '圆明园遗址公园', '海淀区', 'gold', ['传统文化', '国防教育'], '清代皇家园林遗址，爱国主义教育基地。', ['爱国主义教育', '文化研学'], [688, 452]],
    ['beihai', '北海', '西城区', 'gold', ['传统文化'], '皇家园林，琼华岛上一座白塔。', ['文化研学'], null, [1282, 856]],
    ['jingshan', '景山', '西城区', 'gold', ['传统文化'], '中轴线上的制高点，在万春亭俯瞰故宫全景。', ['文化研学'], null, [1370, 856]],
    ['zhonggu', '钟鼓楼', '东城区', 'gold', ['传统文化'], '中轴线北端，元明清三代的报时中心。', ['文化研学'], null, [1370, 792]],
    ['yongding', '永定门', '东城区', 'gold', ['传统文化'], '明清北京外城正门，中轴线南端的起点。', ['文化研学'], null, [1370, 1068]],
    ['guozijian', '国子监', '东城区', 'gold', ['传统文化'], '元明清三代最高学府，辟雍是皇帝讲学的地方。', ['传统文化', '研学课程'], null, [1474, 790]],
    ['shoubo', '首都博物馆', '西城区', 'gold', ['传统文化'], '讲北京历史文化的综合性博物馆。', ['文化研学'], null, [1184, 878]],
    ['guobo', '中国国家博物馆', '东城区', 'gold', ['党史学习', '传统文化'], '有《复兴之路》等基本陈列的国家级博物馆。', ['党史学习', '文化研学'], null, [1478, 930]],
    ['badaling', '八达岭长城', '延庆区', 'gold', ['传统文化'], '明长城最有代表性的段落之一。', ['文化研学', '登山'], [442, 300]],
    ['mutianyu', '慕田峪长城', '怀柔区', 'gold', ['传统文化', '生态文明'], '植被茂密、敌楼密集的一段明长城。', ['登山', '文化研学'], [756, 214]],
    ['simatai', '司马台长城 · 古北水镇', '密云区', 'green', ['传统文化'], '险峻的明长城段落，山下是古北水镇。', ['团建', '登山'], [1182, 122]],
    ['shisanling', '明十三陵', '昌平区', 'gold', ['传统文化'], '明代十三位皇帝的陵墓群。', ['文化研学'], [586, 372]],
    ['yunhe', '大运河 · 城市副中心', '通州区', 'gold', ['传统文化', '生态文明'], '京杭大运河北端，燃灯塔立在运河边。', ['文化研学', '健步走'], [1024, 800]],
    ['aoyun', '奥林匹克公园', '朝阳区', 'gold', ['科技创新'], '鸟巢、水立方，双奥之城的标志。', ['研学', '健步走'], [860, 532]],
    ['kejiguan', '中国科学技术馆', '朝阳区', 'gold', ['科技创新'], '国家级综合性科技馆，适合科普研学。', ['科普研学'], [990, 430]],
    ['shidu', '十渡', '房山区', 'green', ['生态文明'], '拒马河两岸喀斯特峰丛，适合户外拓展。', ['拓展', '户外'], [148, 1012]],
    ['jingxi', '京西古道', '门头沟区', 'green', ['传统文化', '生态文明'], '古商道与驼铃故事，适合团队徒步。', ['徒步'], [292, 600]],
    ['lingshan', '灵山', '门头沟区', 'green', ['生态文明'], '北京最高峰，山顶是高山草甸。', ['登山', '户外'], [132, 470]],
    ['yeyahu', '野鸭湖湿地', '延庆区', 'green', ['生态文明'], '湿地公园，观鸟和生态研学。', ['生态研学'], [196, 236]],
    ['yanqi', '雁栖湖国际会都', '怀柔区', 'blue', ['生态文明'], '湖畔的国际会议区，会议与团建都合适。', ['会议', '团建'], [902, 292]],
    ['pinggu', '平谷桃花', '平谷区', 'green', ['劳动教育'], '春天漫山桃花，农事体验。', ['劳动教育', '健步走'], [1388, 420]],
    ['beigong', '北宫国家森林公园', '丰台区', 'green', ['生态文明'], '离公司近的森林公园，适合健步走。', ['健步走', '团建'], [470, 792]],
    ['dongao', '延庆冬奥赛区', '延庆区', 'green', ['科技创新'], '北京冬奥会延庆赛区，冰雪研学。', ['冰雪研学'], [380, 220]],
    ['guohui', '国家会议中心', '朝阳区', 'blue', [], '举办大型会议、展览的场馆。', ['会议', '展览'], [830, 420]],
    ['shougang', '首钢园', '石景山区', 'blue', ['科技创新'], '工业遗存改造的园区，冬奥滑雪大跳台在这里。', ['论坛', '发布会'], [520, 704]],
    ['zgc', '中关村', '海淀区', 'blue', ['科技创新'], '科创企业集中，产业对标与参访。', ['产业对标', '会议'], [690, 590]],
    ['lize', '丽泽金融商务区', '丰台区', 'blue', [], '新兴金融商务区，离公司近，适合中小型会议。', ['会议', '培训'], [660, 790]],
    ['yizhuang', '亦庄', '经开区', 'blue', ['科技创新'], '北京经济技术开发区，高新产业集中。', ['产业参访', '会议'], [930, 884]],
    ['cbd', 'CBD', '朝阳区', 'blue', [], '中央商务区，商务酒店集中。', ['会议', '年会'], [952, 690]],
  ];
  const HOME = [746, 830];
  const OLD = { cx: 790, cy: 713 };           // 主图上老城的位置
  const INSET = { x: 1124, y: 716, w: 466, h: 398, cx: 1370, s: 3.6, top: 744 };   // 老城放大
  const ROUTES = [
    { id: 'axis', name: '中轴线文化研学', days: '一日', stops: ['yongding', 'tiantan', 'gugong', 'jingshan', 'zhonggu'] },
    { id: 'red', name: '红色足迹', days: '一日', stops: ['honglou', 'lidazhao', 'dangshi', 'xiangshan'] },
    { id: 'wall', name: '长城与冬奥', days: '两日', stops: ['shisanling', 'badaling', 'dongao', 'yeyahu'] },
    { id: 'tech', name: '科创产业对标', days: '一日', stops: ['zgc', 'shougang', 'lize', 'yizhuang'] },
    { id: 'hills', name: '京郊山水团建', days: '两日', stops: ['yanqi', 'mutianyu', 'simatai', 'pinggu'] },
  ];

  /* ── 山的分布：北边山前线、西边山前线以外是山，延庆盆地、水库处空出来 ── */
  const frontN = [[0, 380], [420, 372], [560, 400], [700, 392], [860, 352], [1000, 340], [1150, 330], [1300, 352], [1440, 470], [1600, 520]];
  const lerpLine = (pts, x) => { for (let i = 1; i < pts.length; i++) if (x <= pts[i][0]) { const [x0, y0] = pts[i - 1], [x1, y1] = pts[i]; return y0 + (y1 - y0) * (x - x0) / (x1 - x0); } return pts[pts.length - 1][1]; };
  const frontW = [[400, 0], [470, 420], [492, 560], [470, 700], [420, 820], [380, 940], [340, 1120]];
  const lerpY = (pts, y) => { for (let i = 1; i < pts.length; i++) if (y <= pts[i][1]) { const [x0, y0] = pts[i - 1], [x1, y1] = pts[i]; return x0 + (x1 - x0) * (y - y0) / (y1 - y0); } return pts[pts.length - 1][0]; };
  function density(x, y) {
    const dN = (lerpLine(frontN, x) - y) / 200, dW = y > 380 ? (lerpY(frontW, y) - x) / 260 : 0;
    let d = Math.max(0, Math.min(1, Math.max(dN, dW)));
    const holes = [[300, 240, 120, 70], [1060, 195, 110, 60], [110, 300, 120, 40]];   // 延庆盆地、密云水库、官厅水库
    for (const [hx, hy, rx, ry] of holes) { const e = ((x - hx) / rx) ** 2 + ((y - hy) / ry) ** 2; if (e < 1) d *= Math.max(0, e - 0.25) / 0.75; }
    return d;
  }

  // 局部小画（v 向上）放到画面上
  const at = (inner, x, y, s = 1, px = 1, cls = '') => `<g class="jh ${cls}" style="${J.lineStyle(s * px)}" transform="translate(${N(x)} ${N(y)}) scale(${N(s)} ${N(-s)})">${inner}</g>`;
  const curve = pts => smooth(pts);

  // o.labels === false：不画地名（站点上地名另用一层 SVG 文字叠上去，底图烘成图片）
  function base(px = 1, o = {}) {
    const r = rng(42), out = [];
    const add = s => out.push(s);
    // 纸底与平原上的淡绿
    add(`<rect width="${W}" height="${H}" fill="url(#jh-mapbg)"/>`);
    for (const [x, y, rx, ry] of [[980, 660, 360, 230], [700, 960, 260, 150], [1230, 960, 260, 150], [1220, 580, 200, 120]]) add(`<ellipse cx="${x}" cy="${y}" rx="${rx}" ry="${ry}" fill="#b8c99a" opacity=".16"/>`);
    // 环路：三环到六环，淡虚线
    for (const [w, h, rr] of [[230, 220, 60], [340, 330, 90], [470, 450, 130], [700, 650, 200]]) add(`<rect x="${OLD.cx - w / 2}" y="${OLD.cy - h / 2}" width="${w}" height="${h}" rx="${rr}" fill="none" stroke="#b39a6c" stroke-width="${N(1.1 / px)}" stroke-dasharray="${N(6 / px)} ${N(5 / px)}" opacity=".55"/>`);
    // 河、湖
    const rivers = [
      ['永定河', 8, [[30, 300], [130, 302], [215, 334], [300, 420], [380, 520], [450, 600], [516, 688], [556, 760], [582, 842], [620, 934], [680, 1032], [722, 1120]], [692, 984]],
      ['潮白河', 8, [[1052, 246], [1070, 330], [1110, 410], [1160, 500], [1210, 590], [1250, 680], [1280, 780], [1310, 880], [1340, 980], [1360, 1120]], [1124, 398]],
      ['北运河', 6, [[470, 444], [560, 428], [640, 404], [760, 370], [860, 362], [960, 370], [1060, 396], [1100, 460], [1120, 540], [1124, 620], [1100, 700], [1060, 770], [1046, 840], [1060, 940], [1086, 1040], [1104, 1120]], [1134, 640]],
      ['拒马河', 6, [[40, 880], [104, 950], [176, 1030], [220, 1120]], [60, 950]],
    ];
    let edge = '', fill = '', ripple = '';
    for (const [, w, pts] of rivers) { const d = curve(pts); edge += `<path d="${d}" fill="none" stroke="#6e9aa3" stroke-width="${N(w + 2.4 / px)}" stroke-linecap="round" stroke-linejoin="round"/>`; fill += `<path d="${d}" fill="none" stroke="#d3e2dd" stroke-width="${w}" stroke-linecap="round" stroke-linejoin="round"/>`; }
    const lakes = [
      ['官厅水库', [[0, 276], [80, 268], [168, 282], [214, 302], [168, 326], [66, 334], [0, 324]], [92, 302]],
      ['密云水库', [[956, 172], [1004, 140], [1076, 140], [1140, 166], [1156, 208], [1116, 242], [1062, 236], [1018, 254], [972, 228]], [1056, 196]],
      ['怀柔水库', [[818, 322], [850, 312], [878, 324], [866, 342], [830, 344]], null],
      ['金海湖', [[1452, 486], [1484, 476], [1510, 490], [1494, 508], [1460, 506]], null],
    ];
    for (const [, pts] of lakes) { const d = smooth([...pts, pts[0]]) + 'Z'; edge += `<path d="${d}" fill="#6e9aa3"/>`; fill += `<path d="${d}" fill="#d3e2dd" stroke="#6e9aa3" stroke-width="${N(1.2 / px)}"/>`; const [cx, cy] = pts.reduce((a, b) => [a[0] + b[0] / pts.length, a[1] + b[1] / pts.length], [0, 0]); for (let k = 0; k < 4; k++) ripple += `M${N(cx - 30 + k * 18 + r() * 6)} ${N(cy - 8 + (k % 2) * 12)}q4 -3 8 0q4 3 8 0`; }
    add(edge + fill + `<path d="${ripple}" fill="none" stroke="#6e9aa3" stroke-width="${N(0.8 / px)}" opacity=".7"/>`);
    // 平原上星星点点的小树
    let dots = '';
    for (let i = 0; i < 260; i++) { const x = r() * W, y = r() * H; if (density(x, y) > 0.05) continue; if (Math.hypot(x - OLD.cx, y - OLD.cy) < 140) continue; if (x > INSET.x - 20 && y > INSET.y - 20) continue; const s = 2 + r() * 2; dots += `M${N(x - s)} ${N(y)}a${N(s)} ${N(s * 0.8)} 0 1 0 ${N(s * 2)} 0a${N(s)} ${N(s * 0.8)} 0 1 0 ${N(-s * 2)} 0Z`; }
    add(`<path d="${dots}" fill="#8eab7c" opacity=".55"/>`);
    // 老城"凸"字城墙、护城河、中轴线
    const city = `M755 668H825V722H832V758H748V722H755Z`;
    add(`<path d="${city}" fill="#efe1c8" stroke="#7aa6b0" stroke-width="${N(5 / px)}" stroke-linejoin="round" opacity=".9"/><path d="${city}" fill="none" stroke="#8f3d2b" stroke-width="${N(3 / px)}" stroke-linejoin="round"/><path d="${city}" fill="none" stroke="#f3e9d6" stroke-width="${N(0.9 / px)}" stroke-linejoin="round"/>`);
    add(`<path d="M790 758V666" stroke="#b3362b" stroke-width="${N(1.6 / px)}"/>`);
    add(`<circle cx="${OLD.cx}" cy="${OLD.cy}" r="62" fill="none" stroke="#8f3d2b" stroke-width="${N(1.2 / px)}" stroke-dasharray="${N(3 / px)} ${N(3 / px)}"/>`);
    // 山：舆图式立面小山，按山前线以外的密度撒，北边的先画、南边的压在前面
    const hills = [];
    for (let gy = 0; gy < H + 40; gy += 54) for (let gx = -20; gx < W + 40; gx += 78) {
      const x = gx + (r() - 0.5) * 50 + (Math.round(gy / 54) % 2) * 39, y = gy + (r() - 0.5) * 26, d = density(x, y);
      if (d < 0.16 || r() > 0.22 + d * 0.6) continue;
      hills.push([x, y, d]);
    }
    hills.sort((a, b) => a[1] - b[1]);
    let mts = '';
    for (const [x, y, d] of hills) {
      const P = Pen(), w = 56 + d * 62 + r() * 26, h = 22 + d * 40 + r() * 16, n = 2 + Math.floor(r() * 2);
      const peaks = Array.from({ length: n }, (_, k) => [(k - (n - 1) / 2) * 0.32 + (r() - 0.5) * 0.08, 0.55 + r() * 0.45, 0.24 + r() * 0.1]);
      K.peaks(P, 0, 0, w, h, { seed: Math.round(x * 7 + y), peaks, fill: d > 0.75 ? 'g-shan-mid' : 'g-shan-far', sharp: 0.8 + r() * 0.6 });
      mts += at(P.s(), x, y + 10, 1, px, 'far');
    }
    add(mts);
    // 长城：顺北边山脊，一线城墙、隔一段一座敌楼
    const gw = [[0, 360], [80, 345], [160, 338], [250, 330], [330, 345], [420, 318], [500, 300], [570, 280], [640, 250], [720, 226], [800, 200], [880, 188], [960, 150], [1040, 116], [1120, 100], [1200, 110], [1280, 150], [1350, 220], [1420, 290], [1500, 330], [1600, 340]];
    // Catmull-Rom 取点，再按弧长等距重采样
    const cr = (pts, seg = 10) => { const o = []; for (let i = 0; i < pts.length - 1; i++) { const p0 = pts[i - 1] || pts[i], p1 = pts[i], p2 = pts[i + 1], p3 = pts[i + 2] || p2; for (let k = 0; k < seg; k++) { const t = k / seg, t2 = t * t, t3 = t2 * t; o.push([0, 1].map(j => 0.5 * (2 * p1[j] + (-p0[j] + p2[j]) * t + (2 * p0[j] - 5 * p1[j] + 4 * p2[j] - p3[j]) * t2 + (-p0[j] + 3 * p1[j] - 3 * p2[j] + p3[j]) * t3))); } } o.push(pts[pts.length - 1]); return o; };
    const even = (pts, step) => { const o = [pts[0]]; let acc = 0; for (let i = 1; i < pts.length; i++) { let [xa, ya] = pts[i - 1]; const [xb, yb] = pts[i]; let d = Math.hypot(xb - xa, yb - ya); while (acc + d >= step) { const t = (step - acc) / d; xa += (xb - xa) * t; ya += (yb - ya) * t; o.push([xa, ya]); d = Math.hypot(xb - xa, yb - ya); acc = 0; } acc += d; } return o; };
    const wp = even(cr(gw), 3.2);
    const nrm = i => { const a = wp[Math.max(0, i - 1)], b = wp[Math.min(wp.length - 1, i + 1)]; let nx = b[1] - a[1], ny = -(b[0] - a[0]); const l = Math.hypot(nx, ny) || 1; nx /= l; ny /= l; if (ny > 0) { nx = -nx; ny = -ny; } return [nx, ny]; };
    const off = d => wp.map(([x, y], i) => { const [nx, ny] = nrm(i); return [x + nx * d, y + ny * d]; });
    const top = off(2.2), bot = off(-2.2);
    let wallD = 'M' + top.map(p => `${N(p[0])} ${N(p[1])}`).join('L') + 'L' + bot.reverse().map(p => `${N(p[0])} ${N(p[1])}`).join('L') + 'Z';
    let mer = ''; for (let i = 1; i < wp.length - 1; i += 2) { const [x, y] = wp[i], [nx, ny] = nrm(i), tx = -ny, ty = nx, a = [x + nx * 2.2 - tx * 1.1, y + ny * 2.2 - ty * 1.1], b = [x + nx * 2.2 + tx * 1.1, y + ny * 2.2 + ty * 1.1]; mer += `M${N(a[0])} ${N(a[1])}L${N(b[0])} ${N(b[1])}L${N(b[0] + nx * 2.2)} ${N(b[1] + ny * 2.2)}L${N(a[0] + nx * 2.2)} ${N(a[1] + ny * 2.2)}Z`; }
    add(`<path d="${wallD}" fill="#cfc4ac" stroke="#6a604f" stroke-width="${N(0.9 / px)}" stroke-linejoin="round"/><path d="${mer}" fill="#cfc4ac" stroke="#6a604f" stroke-width="${N(0.6 / px)}"/>`);
    let tw = ''; for (let i = 1; i < gw.length - 1; i += 2) { const [x, y] = gw[i]; tw += `<g transform="translate(${N(x)} ${N(y)})"><rect x="-4.6" y="-8" width="9.2" height="10" fill="#cfc4ac" stroke="#5f5546" stroke-width="${N(0.9 / px)}"/><path d="M-4.6 -8h2v-2h1.6v2h2v-2h1.6v2h2v-2h1.6v2" fill="none" stroke="#5f5546" stroke-width="${N(0.7 / px)}"/><rect x="-2.4" y="-5" width="1.6" height="3" fill="#2a2621"/><rect x=".8" y="-5" width="1.6" height="3" fill="#2a2621"/></g>`; }
    add(tw);
    // 地名：区名（不画界线）、河湖名
    const T = (x, y, s, size, cls = 'mt-d') => `<text class="${cls}" x="${x}" y="${y}" font-size="${size}">${s}</text>`;
    let labels = '';
    for (const [s, x, y] of [['延庆', 300, 288], ['怀柔', 858, 124], ['密云', 1262, 262], ['平谷', 1500, 412], ['昌平', 470, 418], ['顺义', 1240, 566], ['门头沟', 212, 534], ['房山', 410, 1004], ['海淀', 452, 540], ['石景山', 392, 700], ['丰台', 626, 938], ['大兴', 846, 1050], ['通州', 1150, 690], ['朝阳', 1030, 668]]) labels += T(x, y, s, 21);
    for (const [s, w, pts, lp] of rivers) labels += T(lp[0], lp[1], s, 13, 'mt-w');
    for (const [s, , lp] of lakes) if (lp) labels += T(lp[0], lp[1], s, 13, 'mt-w');
    labels += T(560, 262, '长城', 15, 'mt-g');
    LABELS = labels;
    if (o.labels !== false) add(labels);
    return out.join('');
  }

  /* ── 老城放大：同一个"凸"字放大三倍，中轴线、护城河，城里的点另放 ── */
  // 只要地名（先调一次 base 才有）
  let LABELS = '';
  function labels(px = 1) { if (!LABELS) base(px); return LABELS; }

  function inset(px = 1) {
    const { x, y, w, h, cx, s } = INSET, ox = cx - OLD.cx * s, oy = INSET.top - 668 * s;
    const city = `M755 668H825V722H832V758H748V722H755Z`;
    return `<rect x="${x}" y="${y}" width="${w}" height="${h}" rx="6" fill="#f4ecdb" stroke="#8f7a55" stroke-width="${N(1.4 / px)}"/>` +
      `<g transform="translate(${N(ox)} ${N(oy)}) scale(${s})"><path d="${city}" fill="#efe1c8" stroke="#7aa6b0" stroke-width="${N(3 / px / s)}" stroke-linejoin="round"/><path d="${city}" fill="none" stroke="#8f3d2b" stroke-width="${N(2.4 / px / s)}" stroke-linejoin="round"/><path d="M790 758V666" stroke="#b3362b" stroke-width="${N(1.4 / px / s)}"/></g>` +
      `<text class="mt-r" x="${x + 14}" y="${y + 28}" font-size="17">老城放大</text><text class="mt-d" x="${x + 14}" y="${y + 48}" font-size="12">东城 · 西城</text>`;
  }

  window.JH.MAP = { W, H, CATS, SPOTS, HOME, OLD, INSET, ROUTES, base, inset, labels };
})();
