/* 获取活动方案：一屏一题的小问卷。提交到 /api/inquiry（部署说明见 README），
   接口不通时给出电话、微信和"复制需求"，线索不会丢。 */
(function () {
  'use strict';
  const { $, $$, esc, openSheet, toast, copy, ACTIONS } = window.SF;
  const ENDPOINT = '/api/inquiry';
  const STEPS = [
    { key: 'type', q: '想办什么活动？', hint: '可多选', multi: true,
      opts: [['red', '红色教育 / 主题党日'], ['gov', '党政团建'], ['ent', '企业团建'], ['mice', '会议会展'], ['yanxiu', '研学考察'], ['tubu', '商务出行'], ['jlyt', '其他']] },
    { key: 'org', q: '您的单位属于哪一类？',
      opts: [['red', '党政机关'], ['gonghui', '事业单位'], ['duibiao', '国有企业'], ['yanxue', '其他企业'], ['yanxiu', '高校 / 医院'], ['jlyt', '其他']] },
    { key: 'size', q: '大概多少人参加？', opts: [[null, '30 人以内'], [null, '30–100 人'], [null, '100–300 人'], [null, '300 人以上']] },
    { key: 'when', q: '计划什么时候？', group2: { key: 'days', q: '活动时长', opts: [[null, '半天'], [null, '一天'], [null, '两天及以上']] },
      opts: [[null, '一个月内'], [null, '1–3 个月'], [null, '3 个月以后'], [null, '时间待定']] },
    { key: 'contact', q: '怎么联系您？', form: true },
  ];
  let st = {}, idx = 0;

  function chips(step, sel) {
    return `<div class="iq-opts${step.opts[0][0] ? ' ic-on' : ''}" role="${step.multi ? 'group' : 'radiogroup'}">${step.opts.map(([ic, t]) => {
      const on = step.multi ? (sel || []).includes(t) : sel === t;
      return `<button type="button" class="iq-opt${on ? ' on' : ''}" data-v="${esc(t)}" aria-pressed="${on}">${ic ? ICON(ic) : ''}<span>${t}</span></button>`;
    }).join('')}</div>`;
  }
  function render() {
    const s = STEPS[idx], pct = Math.round((idx + 1) / STEPS.length * 100);
    let body;
    if (s.form) {
      body = `<div class="iq-form">
        <label><span>称呼 <i>*</i></span><input name="name" autocomplete="name" placeholder="例如：王老师、李主任" value="${esc(st.name || '')}"></label>
        <label><span>手机或微信 <i>*</i></span><input name="phone" inputmode="tel" autocomplete="tel" placeholder="方便我们联系您" value="${esc(st.phone || '')}"></label>
        <label><span>单位名称</span><input name="company" autocomplete="organization" placeholder="选填" value="${esc(st.company || '')}"></label>
        <label><span>补充说明</span><textarea name="note" rows="3" placeholder="选填，例如：想去香山、需要讲解、预算范围">${esc(st.note || '')}</textarea></label>
        <label class="iq-hp" aria-hidden="true">请勿填写<input name="website" tabindex="-1" autocomplete="off"></label>
        <p class="iq-privacy">信息仅用于联系您、准备活动方案。</p>
      </div>`;
    } else {
      body = chips(s, st[s.key]) + (s.group2 ? `<p class="iq-q2">${s.group2.q}</p>` + chips(s.group2, st[s.group2.key]).replace('iq-opts', 'iq-opts g2') : '');
    }
    openSheet(`<div class="iq" data-step="${idx}">
      <div class="iq-bar"><i style="width:${pct}%"></i></div>
      <p class="iq-n">第 ${idx + 1} / ${STEPS.length} 步</p>
      <h3>${s.q}</h3>${s.hint ? `<p class="sub">${s.hint}</p>` : ''}
      ${body}
      <div class="iq-nav">${idx ? '<button type="button" class="btn ghost" data-iq="back">上一步</button>' : '<span></span>'}
        <button type="button" class="btn primary" data-iq="next">${s.form ? '提交' : '下一步'}</button></div>
      <p class="iq-err" role="alert"></p></div>`);
  }
  function err(msg) { const e = $('.iq-err'); if (e) e.textContent = msg; }
  function readForm() {
    for (const k of ['name', 'phone', 'company', 'note', 'website']) { const el = $(`.iq [name="${k}"]`); if (el) st[k] = el.value.trim(); }
  }
  function summary() {
    return ['【活动需求】', `类型：${(st.type || []).join('、') || '未选'}`, `单位：${st.org || '未选'}`, `人数：${st.size || '未选'}`,
      `时间：${st.when || '未选'}　时长：${st.days || '未选'}`, `称呼：${st.name}`, `联系方式：${st.phone}`,
      st.company && `单位名称：${st.company}`, st.note && `说明：${st.note}`].filter(Boolean).join('\n');
  }
  async function submit() {
    readForm();
    if (!st.name) return err('请填写称呼');
    if (!/^(1[3-9]\d{9}|[A-Za-z][-_A-Za-z0-9]{5,19}|0\d{2,3}-?\d{7,8})$/.test(st.phone.replace(/\s/g, ''))) return err('请填写正确的手机号或微信号');
    if (st.website) return done(true);
    const btn = $('.iq [data-iq="next"]'); btn.disabled = true; btn.textContent = '提交中…';
    const ctrl = new AbortController(), t = setTimeout(() => ctrl.abort(), 8000);
    try {
      const r = await fetch(ENDPOINT, { method: 'POST', headers: { 'Content-Type': 'application/json' }, signal: ctrl.signal,
        body: JSON.stringify({ ...st, website: undefined, page: location.href.split('#')[0], at: new Date().toISOString() }) });
      done(r.ok);
    } catch { done(false); } finally { clearTimeout(t); }
  }
  function done(ok) {
    const tel = $('#card').dataset.tel;
    openSheet(ok
      ? `<div class="iq-done"><div class="iq-ok">${ICON('baoxian')}</div><h3>收到，我们会尽快联系您</h3><p class="txt">一般一个工作日内给您初步方案。着急的话，可以直接打电话。</p>
         <button class="btn primary" type="button" data-act="call"><svg class="i"><use href="#i-phone"/></svg>拨打 ${tel}</button></div>`
      : `<div class="iq-done"><h3>暂时没能提交成功</h3><p class="txt">您填写的需求已整理好，可以复制后发给我们，或直接打电话。</p>
         <pre class="iq-sum">${esc(summary())}</pre>
         <div class="row2"><button class="btn primary" type="button" data-iq="copy">复制需求</button><button class="btn ghost" type="button" data-act="call"><svg class="i"><use href="#i-phone"/></svg>打电话</button></div></div>`);
    if (ok) st = {};
  }
  document.addEventListener('click', e => {
    const opt = e.target.closest('.iq-opt');
    if (opt) {
      const s = STEPS[idx], g2 = opt.closest('.g2'), step = g2 ? s.group2 : s, v = opt.dataset.v;
      if (step.multi) {
        const a = st[step.key] = st[step.key] || [];
        const i = a.indexOf(v); if (i >= 0) a.splice(i, 1); else a.push(v);
        opt.classList.toggle('on'); opt.setAttribute('aria-pressed', opt.classList.contains('on'));
      } else {
        st[step.key] = v;
        $$('.iq-opt', opt.parentElement).forEach(b => { const on = b === opt; b.classList.toggle('on', on); b.setAttribute('aria-pressed', on); });
        if (!s.group2 || (st[s.key] && st[s.group2.key])) setTimeout(() => { if (STEPS[idx] === s) { idx++; render(); } }, 260);
      }
      err('');
      return;
    }
    const b = e.target.closest('[data-iq]'); if (!b) return;
    const k = b.dataset.iq, s = STEPS[idx];
    if (k === 'back') { if (s.form) readForm(); idx = Math.max(0, idx - 1); render(); }
    if (k === 'next') {
      if (s.form) return submit();
      if (s.multi ? !(st[s.key] || []).length : !st[s.key]) return err('请先选一项');
      if (s.group2 && !st[s.group2.key]) return err('请选择活动时长');
      idx++; render();
    }
    if (k === 'copy') copy(summary(), '需求');
  });
  ACTIONS.inquiry = el => {
    idx = 0;
    const pre = el && el.dataset && el.dataset.pre, note = el && el.dataset && el.dataset.note;
    if (pre) { st.type = [pre]; }
    if (note && !(st.note || '').includes(note)) st.note = st.note ? `${st.note}；${note}` : note;   // 从哪处、哪条线路点进来的，带进补充说明
    render();
  };
})();
