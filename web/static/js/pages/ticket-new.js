// ticket-new.js —— 提交工单页逻辑（单一职责：提交表单）
if (!Auth.requireAuth()) throw new Error('stop');

document.getElementById('nav').innerHTML = UI.renderNavbar('new');
UI.bindNavbar();

async function loadCategories() {
  const cats = await API.get('/categories');
  const sel = document.getElementById('category');
  sel.innerHTML = cats.map(c => `<option value="${c.id}">${UI.esc(c.name)}</option>`).join('');
}

document.getElementById('ticketForm').addEventListener('submit', async (e) => {
  e.preventDefault();
  const form = e.target;
  const fd = new FormData(form);
  try {
    const data = await API.upload('/tickets', fd);
    UI.toast('工单已提交：' + data.ticket_no, 'success');
    setTimeout(() => location.href = '/tickets/detail?id=' + data.id, 600);
  } catch (err) {
    UI.toast(err.message, 'error');
  }
});

loadCategories();
