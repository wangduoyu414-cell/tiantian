/* 晟帆主页 · 线条图标（32×32，1.6 线宽，圆头）。.ac 是题材色的淡彩块，.acs 是题材色线条。
   颜色跟随所在板块：currentColor 是墨色，--ac 是板块的点缀色。 */
(function () {
  'use strict';
  const I = {
    // 首屏四个入口
    red: '<path class="ac" d="M5 12h22L16 5.5z"/><path d="M4 12.5h24M5.5 12 16 5l10.5 7M8 13.5v9M13 13.5v9M19 13.5v9M24 13.5v9M5 23h22M3.5 26.5h25"/>',
    gov: '<circle cx="16" cy="9.5" r="3.2"/><circle cx="8.2" cy="12" r="2.5"/><circle cx="23.8" cy="12" r="2.5"/><path class="ac" d="M10.4 25c0-4.3 2.5-7.6 5.6-7.6s5.6 3.3 5.6 7.6z"/><path d="M10.4 25c0-4.3 2.5-7.6 5.6-7.6s5.6 3.3 5.6 7.6M3.6 24c0-3.4 2-6 4.6-6 1.2 0 2.3.5 3.1 1.4M28.4 24c0-3.4-2-6-4.6-6-1.2 0-2.3.5-3.1 1.4"/>',
    ent: '<path class="ac" d="M3 26l9-12 4.5 5.5L21 14l8 12z"/><path d="M3 26l9-12 4.5 5.5L21 14l8 12zM21 14V5.5l5 2.2-5 2.2"/>',
    mice: '<rect x="5" y="5" width="22" height="12" rx="1.5"/><path class="ac" d="M12 20h8l-1.2 7h-5.6z"/><path d="M12 20h8l-1.2 7h-5.6zM10 20h12M16 17v3M9 9.5h8M9 12.5h12"/>',
    // 红色教育·活动形式
    ztdr: '<rect x="5" y="7" width="22" height="20" rx="2.5"/><path d="M5 12.5h22M11 4.5v5M21 4.5v5"/><path class="acs" d="M11.5 19.5l3.2 3.2 6-6.4"/>',
    xcjx: '<path d="M14 26V11.5L21 7l7 4.5V26"/><path class="ac" d="M18 26v-5h6v5z"/><path d="M18 26v-5h6v5M17 14h2M23 14h2"/><circle cx="7.5" cy="10.5" r="2.6"/><path d="M3.5 26c0-5 1.6-9.2 4-9.2s4 4.2 4 9.2M10.6 16.4l4-2.6"/>',
    ztdk: '<path class="ac" d="M9.5 17h13l-1.6 10h-9.8z"/><path d="M9.5 17h13l-1.6 10h-9.8zM7.5 17h17M16 17v-5"/><circle cx="16" cy="9" r="2.2"/><path d="M5 27h22"/>',
    qjjx: '<path class="ac" d="M4 7.5c3 3 5 8 5 15H4zM28 7.5c-3 3-5 8-5 15h5z"/><path d="M4 27V7.5h24V27M4 7.5c3 3 5 8 5 15M28 7.5c-3 3-5 8-5 15M2.5 27h27"/><circle cx="16" cy="15.5" r="2.2"/><path d="M12.5 23c0-2.6 1.6-4.4 3.5-4.4s3.5 1.8 3.5 4.4"/>',
    cwsy: '<circle class="ac" cx="16" cy="15.5" r="11.5"/><path d="M12 28v-5.8l-3.4-4.3a1.7 1.7 0 0 1 2.6-2.2l1.6 1.8V8.6a1.7 1.7 0 0 1 3.4 0v6.2V6.8a1.7 1.7 0 0 1 3.4 0v8V8.6a1.7 1.7 0 0 1 3.4 0v11.6c0 4.4-2.6 7.8-7 7.8"/>',
    jlyt: '<path d="M4 6.5h14a2 2 0 0 1 2 2v7a2 2 0 0 1-2 2h-7l-4.5 3.5v-3.5H4a2 2 0 0 1-2-2v-7a2 2 0 0 1 2-2z"/><path class="ac" d="M14 13h14a2 2 0 0 1 2 2v7a2 2 0 0 1-2 2h-2.5v3.5L21 24h-7a2 2 0 0 1-2-2v-7a2 2 0 0 1 2-2z"/><path d="M14 13h14a2 2 0 0 1 2 2v7a2 2 0 0 1-2 2h-2.5v3.5L21 24h-7a2 2 0 0 1-2-2v-7a2 2 0 0 1 2-2zM17 17.5h8M17 20.5h5"/>',
    // 党政团建
    dingxiang: '<circle cx="16" cy="16" r="11.5"/><path class="ac" d="M16 7.5l3.6 8.5h-7.2z"/><path d="M16 7.5l3.6 8.5-3.6 8.5-3.6-8.5zM16 3v2M16 27v2M3 16h2M27 16h2"/>',
    gonghui: '<rect x="4.5" y="5" width="13" height="11" rx="2"/><path d="M4.5 9h13M8.5 3.5v3M13.5 3.5v3"/><circle cx="22.5" cy="15" r="3"/><path class="ac" d="M16 28c0-4.4 2.9-7.6 6.5-7.6s6.5 3.2 6.5 7.6z"/><path d="M16 28c0-4.4 2.9-7.6 6.5-7.6s6.5 3.2 6.5 7.6M8 12.4l1.8 1.6 3.2-3.2"/>',
    laodong: '<path d="M5 25h22M16 25V14.5"/><path class="ac" d="M16 17c-5.4 0-7.8-3-7.8-7.6 5.4 0 7.8 3 7.8 7.6z"/><path d="M16 17c-5.4 0-7.8-3-7.8-7.6 5.4 0 7.8 3 7.8 7.6zM16 14.5c0-4.4 2.6-7 7.6-7 0 4.4-2.6 7-7.6 7zM8 25c.5 1.6 1.5 2.5 3 2.5M24 25c-.5 1.6-1.5 2.5-3 2.5"/>',
    feiyi: '<path class="ac" d="M16 5.5c2 1.5 3 3.5 3 6 4-2 7-1.5 9.5 1-3 .5-6 2-9 4.5L16 25.5l-3.5-8.5c-3-2.5-6-4-9-4.5 2.5-2.5 5.5-3 9.5-1 0-2.5 1-4.5 3-6z"/><path d="M16 5.5c2 1.5 3 3.5 3 6 4-2 7-1.5 9.5 1-3 .5-6 2-9 4.5L16 25.5l-3.5-8.5c-3-2.5-6-4-9-4.5 2.5-2.5 5.5-3 9.5-1 0-2.5 1-4.5 3-6zM16 11.5v8.5M16 25.5c-1.4 1.2-1.4 2.4 0 3.5"/>',
    dengshan: '<circle class="ac" cx="23.5" cy="8.5" r="3"/><path d="M2.5 26l9.5-13 5 6.5 3.5-4.5L29.5 26z"/><path class="acs" d="M10 26c1.6-2.6 4.6-3.2 6.2-6" stroke-dasharray="1.8 2.2"/>',
    dushu: '<path class="ac" d="M16 9.5C13 7.5 9 7 4.5 7.5V23c4.5-.5 8.5 0 11.5 2z"/><path d="M16 9.5C13 7.5 9 7 4.5 7.5V23c4.5-.5 8.5 0 11.5 2 3-2 7-2.5 11.5-2V7.5C23 7 19 7.5 16 9.5zM16 9.5V25"/>',
    // 企业团建
    tuozhan: '<rect x="6" y="4.5" width="20" height="23" rx="2"/><circle class="ac" cx="11.5" cy="10" r="1.9"/><circle class="ac" cx="20" cy="12.5" r="1.9"/><circle class="ac" cx="12.5" cy="17.5" r="1.9"/><circle class="ac" cx="20.5" cy="22" r="1.9"/><path d="M11.5 10 20 12.5 12.5 17.5 20.5 22" stroke-dasharray="1.6 2"/>',
    tubu: '<path d="M9 12a7 7 0 0 1 14 0v12.5a2.5 2.5 0 0 1-2.5 2.5h-9A2.5 2.5 0 0 1 9 24.5zM13 8.5V6.8a3 3 0 0 1 6 0v1.7M9 15.5h14"/><rect class="ac" x="12" y="18" width="8" height="5.5" rx="1.5"/><rect x="12" y="18" width="8" height="5.5" rx="1.5"/>',
    yundong: '<path d="M10.5 3.5l5.5 9 5.5-9"/><circle class="ac" cx="16" cy="20" r="7"/><circle cx="16" cy="20" r="7"/><path d="M16 16.6v6.8M14.2 18.1l1.8-1.5"/>',
    gongchuang: '<rect x="4.5" y="5" width="23" height="16" rx="2"/><path class="ac" d="M8 8.5h5v5H8zM15 8.5h5v5h-5zM8 15.5h5v3H8z"/><path d="M8 8.5h5v5H8zM15 8.5h5v5h-5zM22 9h2.5M22 12h2.5M8 15.5h5v3H8zM11 21l-2.5 6M21 21l2.5 6"/>',
    nianhui: '<path class="ac" d="M8 5.5 13 20h6l5-14.5"/><path d="M4 23.5h24M6 23.5v-3.5h20v3.5M5 27.5h22M6 4.5h4M22 4.5h4M8 5.5 13 20M24 5.5 19 20"/>',
    yanxue: '<path d="M4 26V12l6 3.6V12l6 3.6V12l6 3.6V26z"/><circle class="ac" cx="23.5" cy="20" r="4"/><circle cx="23.5" cy="20" r="4"/><path d="M26.4 22.9 29 25.5M8 21h3M8 24h3"/>',
    // 保障
    baoxian: '<path class="ac" d="M16 3.5l10 4v7.5c0 6.5-4.3 11.2-10 13.5-5.7-2.3-10-7-10-13.5V7.5z"/><path d="M16 3.5l10 4v7.5c0 6.5-4.3 11.2-10 13.5-5.7-2.3-10-7-10-13.5V7.5zM11.5 16l3.2 3.2 6-6.4"/>',
    cheliang: '<rect x="4.5" y="6" width="23" height="16.5" rx="3"/><path class="ac" d="M7 9h18v6H7z"/><path d="M7 9h18v6H7zM16 9v6M4.5 18.5h23"/><circle cx="10" cy="24.5" r="2.2"/><circle cx="22" cy="24.5" r="2.2"/>',
    yingji: '<rect x="4" y="9" width="24" height="17" rx="2.5"/><path d="M12 9V6.5h8V9"/><path class="ac" d="M14 13h4v3h3v4h-3v3h-4v-3h-3v-4h3z"/><path d="M14 13h4v3h3v4h-3v3h-4v-3h-3v-4h3z"/>',
    zhuanren: '<circle cx="16" cy="11" r="5"/><path d="M9.5 11a6.5 6.5 0 0 1 13 0M9.5 11v2.5M22.5 11v2.5a3 3 0 0 1-3 3h-2"/><path class="ac" d="M6 28c0-5.5 4.5-9 10-9s10 3.5 10 9z"/><path d="M6 28c0-5.5 4.5-9 10-9s10 3.5 10 9"/>',
    piaoju: '<path d="M8 3.5h11l5.5 5.5v19.5H8zM19 3.5V9h5.5M11.5 14h9M11.5 18h9M11.5 22h4.5"/><circle class="ac" cx="20.5" cy="23.5" r="3.5"/><path class="acs" d="M19 23.5l1.1 1.1 2-2.2"/>',
    // 研学考察
    yanxiu: '<path class="ac" d="M16 6 3 12.5 16 19l13-6.5z"/><path d="M16 6 3 12.5 16 19l13-6.5zM8 15v6c2.5 2.5 13.5 2.5 16 0v-6M27 13.5v7"/>',
    duibiao: '<path d="M4 27V13l6 4v-4l6 4V8h8v19zM2.5 27h27"/><path class="ac" d="M18 11h4v4h-4z"/><path d="M18 11h4v4h-4zM18 19h4M18 22.5h4M7.5 22h3"/>',
    kaocha: '<path d="M4 26.5h24"/><path class="ac" d="M17.5 13h5v13.5h-5z"/><path d="M5.5 26.5V19h5v7.5M11.5 26.5V15h5v11.5M17.5 26.5V13h5v13.5M23.5 26.5V9h5v17.5"/><path class="acs" d="M5 14l6-4 6 2 9-7"/>',
  };
  window.ICON = (name, cls = '') => `<svg class="ic ${cls}" viewBox="0 0 32 32" aria-hidden="true">${I[name] || ''}</svg>`;
})();
