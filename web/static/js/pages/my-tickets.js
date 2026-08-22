// my-tickets.js —— 我提交的工单列表页逻辑
if (!Auth.requireAuth()) throw new Error('stop');

document.getElementById('nav').innerHTML = UI.renderNavbar('my-tickets');
UI.bindNavbar();

let page = 1;
const PAGE_SIZE = 10;

async function load() {
  const query = {
    page, page_size: PAGE_SIZE,
    status: document.getElementById('status').value,
    urgency: document.getElementById('urgency').value,
    search: document.getElementById('search').value.trim(),
  };
  try {
    const res = await API.get('/tickets/my', query);
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

load();
Auth.startOverduePolling();
