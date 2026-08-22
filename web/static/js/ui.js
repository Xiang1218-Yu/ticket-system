// ui.js —— 展示工具层（单一职责：格式化、转义、toast、分页、星级、导航栏）
const UI = (function () {
  function esc(s) {
    if (s == null) return '';
    return String(s)
      .replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;').replace(/'/g, '&#39;');
  }

  function fmtDate(s) {
    if (!s) return '-';
    const d = new Date(s);
    if (isNaN(d)) return '-';
    const p = n => String(n).padStart(2, '0');
    return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}`;
  }

  const STATUS_LABEL = { pending: '待处理', processing: '处理中', done: '已完成', closed: '已关闭' };
  const STATUS_CLASS = {
    pending: 'status-pending', processing: 'status-processing',
    done: 'status-done', closed: 'status-closed',
  };
  function statusLabel(s) { return STATUS_LABEL[s] || s; }
  function statusClass(s) { return STATUS_CLASS[s] || ''; }

  const URGENCY_LABEL = { normal: '普通', urgent: '紧急' };

  function stars(n) {
    n = Math.round(n || 0);
    return '★★★★★'.slice(0, n) + '☆☆☆☆☆'.slice(0, 5 - n);
  }

  function toast(msg, type = 'info') {
    const el = document.createElement('div');
    el.className = 'toast toast-' + type;
    el.textContent = msg;
    document.body.appendChild(el);
    setTimeout(() => el.classList.add('show'), 10);
    setTimeout(() => {
      el.classList.remove('show');
      setTimeout(() => el.remove(), 300);
    }, 2500);
  }

  // navbar 渲染（各页面共享）
  function renderNavbar(active) {
    const u = Auth.user() || {};
    const items = [
      { href: '/dashboard', key: 'dashboard', label: '看板', show: true },
      { href: '/tickets', key: 'tickets', label: '所有工单', show: u.role === 'admin' || u.is_leader },
      { href: '/pending', key: 'pending', label: '待我处理', show: u.role === 'handler' },
      { href: '/my-tickets', key: 'my-tickets', label: '我提交的', show: true },
      { href: '/tickets/new', key: 'new', label: '提交工单', show: true },
      { href: '/overdue', key: 'overdue', label: '超时工单', show: true },
    ].filter(i => i.show);

    const links = items.map(i =>
      `<a href="${i.href}" class="nav-link${i.key === active ? ' active' : ''}">${i.label}</a>`
    ).join('');

    const roleLabel = { admin: '管理员', handler: '处理人', employee: '员工' }[u.role] || u.role;
    const leaderTag = u.is_leader ? '（组长）' : '';
    const groupTag = u.group ? `[${u.group}]` : '';

    return `
      <nav class="navbar">
        <div class="nav-brand">工单系统</div>
        <div class="nav-links">${links}</div>
        <div class="nav-user">
          <span class="nav-name">${esc(u.name)} <small>${roleLabel}${leaderTag}${groupTag}</small></span>
          <a href="#" class="nav-logout" id="logoutBtn">退出</a>
        </div>
      </nav>`;
  }

  function bindNavbar(container = document) {
    const btn = container.querySelector('#logoutBtn');
    if (btn) btn.onclick = (e) => { e.preventDefault(); Auth.logout(); };
  }

  // 工单表格行 HTML（列表页共用）
  function renderTicketRow(t) {
    const urgent = t.urgency === 'urgent' ? '<span class="urgent-tag">[紧急]</span> ' : '';
    const overdueMark = t.overdue ? ' row-overdue' : '';
    const overdueTag = t.overdue ? ' <span class="badge" style="background:#fee2e2;color:#dc2626;">超时</span>' : '';
    const submitter = t.submitter ? esc(t.submitter.name) : '-';
    const assignee = t.assignee ? esc(t.assignee.name) : '<span style="color:var(--muted);">未指派</span>';
    const category = t.category ? esc(t.category.name) : '-';
    return `
      <tr class="${overdueMark}">
        <td><a href="/tickets/detail?id=${t.id}">${esc(t.ticket_no)}</a></td>
        <td>${urgent}<a href="/tickets/detail?id=${t.id}">${esc(t.title)}</a>${overdueTag}</td>
        <td>${category}</td>
        <td><span class="status ${statusClass(t.status)}">${statusLabel(t.status)}</span></td>
        <td>${submitter}</td>
        <td>${assignee}</td>
        <td>${fmtDate(t.submitted_at)}</td>
      </tr>`;
  }

  // 分页渲染
  function renderPagination(total, page, pageSize, onChange) {
    const pages = Math.max(1, Math.ceil(total / pageSize));
    if (pages <= 1) return '';
    let html = '<div class="pagination">';
    const btn = (label, p, dis, cur) =>
      `<button class="page-btn${cur ? ' active' : ''}" ${dis ? 'disabled' : ''} data-page="${p}">${label}</button>`;
    html += btn('上一页', page - 1, page <= 1, false);
    html += `<span class="page-info">${page} / ${pages}</span>`;
    html += btn('下一页', page + 1, page >= pages, false);
    html += '</div>';
    setTimeout(() => {
      document.querySelectorAll('.page-btn').forEach(b => {
        b.onclick = () => { const p = +b.dataset.page; if (p && !b.disabled) onChange(p); };
      });
    }, 0);
    return html;
  }

  // 文件大小
  function fmtSize(bytes) {
    if (bytes < 1024) return bytes + ' B';
    if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB';
    return (bytes / 1024 / 1024).toFixed(1) + ' MB';
  }

  // 工单表格（含表头与数据行，列表页共用）
  function renderTicketTable(items) {
    if (!items || items.length === 0) {
      return '<div class="empty">暂无工单</div>';
    }
    const head = `<thead><tr>
      <th>编号</th><th>标题</th><th>类型</th><th>状态</th>
      <th>提交人</th><th>处理人</th><th>提交时间</th>
    </tr></thead>`;
    const rows = items.map(renderTicketRow).join('');
    return `<table class="table">${head}<tbody>${rows}</tbody></table>`;
  }

  return {
    esc, fmtDate, fmtSize, statusLabel, statusClass, URGENCY_LABEL, stars,
    toast, renderNavbar, bindNavbar, renderPagination, renderTicketRow, renderTicketTable,
  };
})();
