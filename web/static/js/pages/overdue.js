// overdue.js —— 超时工单列表页逻辑（单一职责：展示超时工单 + 接收轮询更新）
if (!Auth.requireAuth()) throw new Error('stop');

document.getElementById('nav').innerHTML = UI.renderNavbar('overdue');
UI.bindNavbar();

const PAGE_SIZE = 50;

async function load() {
  try {
    const res = await API.get('/tickets/overdue', { page: 1, page_size: PAGE_SIZE });
    document.getElementById('overdueCount').textContent = `（共 ${res.total} 条）`;
    const wrap = document.getElementById('listWrap');
    wrap.innerHTML = UI.renderTicketTable(res.items);
  } catch (err) {
    UI.toast(err.message, 'error');
  }
}

// 接收 auth.js 轮询触发的超时更新事件，刷新页面
window.addEventListener('overdue-updated', () => load());

load();
Auth.startOverduePolling();
