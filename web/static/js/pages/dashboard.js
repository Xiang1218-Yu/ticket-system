// dashboard.js —— 统计看板页逻辑（单一职责：拉取统计并渲染）
if (!Auth.requireAuth()) throw new Error('stop');

document.getElementById('nav').innerHTML = UI.renderNavbar('dashboard');
UI.bindNavbar();

async function load() {
  try {
    const d = await API.get('/stats');
    renderStats(d);
    renderCategories(d.by_category || []);
    renderWorkload(d.workload || []);
  } catch (err) {
    UI.toast(err.message, 'error');
  }
}

function renderStats(d) {
  const grid = document.getElementById('statGrid');
  const cards = [
    { num: d.today_new, label: '今日新增', cls: '' },
    { num: d.week_new, label: '本周新增', cls: '' },
    { num: d.pending, label: '待处理工单', cls: 'warn' },
    { num: d.overdue, label: '超时工单', cls: 'danger' },
    { num: d.avg_text || '-', label: '平均处理时长', cls: '' },
  ];
  grid.innerHTML = cards.map(c =>
    `<div class="stat-card ${c.cls}"><div class="stat-num">${UI.esc(String(c.num))}</div><div class="stat-label">${c.label}</div></div>`
  ).join('');
}

function renderCategories(cats) {
  const wrap = document.getElementById('catChart');
  if (!cats.length) { wrap.innerHTML = '<div class="empty">暂无数据</div>'; return; }
  const max = Math.max(...cats.map(c => c.count), 1);
  wrap.innerHTML = cats.map(c => {
    const pct = Math.round(c.count / max * 100);
    return `<div style="margin:8px 0;">
      <div style="display:flex; justify-content:space-between; font-size:14px;">
        <span>${UI.esc(c.category)}</span><span style="color:var(--muted);">${c.count}</span>
      </div>
      <div style="background:var(--border); border-radius:4px; height:10px; margin-top:4px;">
        <div style="width:${pct}%; height:100%; background:var(--primary); border-radius:4px;"></div>
      </div>
    </div>`;
  }).join('');
}

function renderWorkload(list) {
  const wrap = document.getElementById('workload');
  if (!list.length) { wrap.innerHTML = '<div class="empty">暂无数据</div>'; return; }
  wrap.innerHTML = '<table class="table"><thead><tr><th>处理人</th><th>组</th><th>处理工单数</th></tr></thead><tbody>' +
    list.map(w => `<tr><td>${UI.esc(w.name)}</td><td>${UI.esc(w.group)}</td><td>${w.count}</td></tr>`).join('') +
    '</tbody></table>';
}

load();
Auth.startOverduePolling();
