// tickets.js —— 所有工单列表页逻辑（单一职责：查询与展示所有工单）
if (!Auth.requireAuth()) throw new Error('stop');
const u = Auth.user();
if (u.role !== 'admin' && !u.is_leader) {
  // 仅管理员/组长可访问
  location.href = '/my-tickets';
  throw new Error('stop');
}

document.getElementById('nav').innerHTML = UI.renderNavbar('tickets');
UI.bindNavbar();

let page = 1;
const PAGE_SIZE = 10;

async function loadCategories() {
  const cats = await API.get('/categories');
  const sel = document.getElementById('category');
  sel.innerHTML = '<option value="">全部类型</option>' +
    cats.map(c => `<option value="${UI.esc(c.name)}">${UI.esc(c.name)}</option>`).join('');
}

async function load() {
  const query = {
    page, page_size: PAGE_SIZE,
    status: document.getElementById('status').value,
    category: document.getElementById('category').value,
    urgency: document.getElementById('urgency').value,
    search: document.getElementById('search').value.trim(),
  };
  try {
    const res = await API.get('/tickets', query);
    const wrap = document.getElementById('listWrap');
    let html = UI.renderTicketTable(res.items);
    html += UI.renderPagination(res.total, page, PAGE_SIZE, (p) => { page = p; load(); });
    wrap.innerHTML = html;
  } catch (err) {
    UI.toast(err.message, 'error');
  }
}

document.getElementById('searchBtn').addEventListener('click', () => { page = 1; load(); });
document.getElementById('search').addEventListener('keydown', (e) => {
  if (e.key === 'Enter') { page = 1; load(); }
});

loadCategories().then(load);
Auth.startOverduePolling();
