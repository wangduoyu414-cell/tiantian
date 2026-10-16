// 在浏览器控制台运行：列出页面上用到思源宋体（Noto Serif SC）的字符，按字重分组。
// 弹层里的字（证书名、感谢信标题等）由 site.js 放在 window.__serifExtra 里，一并收进来。
// 把输出的 700 / 900 两串字填进 tools/font-text.json，再运行 python3 tools/fonts.py。
(() => {
  const sets = { 700: new Set(), 900: new Set() };
  const w = document.createTreeWalker(document.body, NodeFilter.SHOW_TEXT);
  while (w.nextNode()) {
    const n = w.currentNode, el = n.parentElement;
    if (!el) continue;
    const cs = getComputedStyle(el);
    if (!/Noto Serif SC/.test(cs.fontFamily)) continue;
    const k = +cs.fontWeight >= 800 ? 900 : 700;
    for (const ch of n.textContent) if (ch.trim()) sets[k].add(ch);
  }
  const tpl = document.getElementById('tpl-roster');
  if (tpl) tpl.content.querySelectorAll('.rc-head b').forEach(b => { for (const ch of b.textContent) sets[900].add(ch); });
  for (const ch of (window.__serifExtra || '')) if (ch.trim()) sets[700].add(ch);
  const out = { 700: [...sets[700]].sort().join(''), 900: [...sets[900]].sort().join('') };
  console.log(JSON.stringify(out, null, 2));
  return out;
})();
