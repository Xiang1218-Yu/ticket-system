// ticket-detail.js —— 工单详情页逻辑（单一职责：编排详情各区块）
if (!Auth.requireAuth()) throw new Error('stop');

const me = Auth.user() || {};
document.getElementById('nav').innerHTML = UI.renderNavbar('');
UI.bindNavbar();

const ticketId = new URLSearchParams(location.search).get('id');
if (!ticketId) { location.href = '/tickets'; }

async function load() {
  try {
    const t = await API.get('/tickets/' + ticketId);
    render(t);
  } catch (err) {
    UI.toast(err.message, 'error');
    document.getElementById('detailWrap').innerHTML = `<div class="card empty">${UI.esc(err.message)}</div>`;
  }
}

function canChangeStatus(t, to) {
  if (to === 'processing') {
    return me.role === 'admin' || (t.assignee && t.assignee.id === me.id);
  }
  if (to === 'done') {
    return me.role === 'admin' || (t.assignee && t.assignee.id === me.id) ||
      (me.is_leader && me.group === t.group);
  }
  if (to === 'closed') {
    return me.role === 'admin' || t.submitter.id === me.id;
  }
  return false;
}

function canAssign(t) {
  return me.role === 'admin' || (me.is_leader && me.group === t.group);
}

function canComment(t) {
  return me.role === 'admin' ||
    (t.submitter && t.submitter.id === me.id) ||
    (t.assignee && t.assignee.id === me.id) ||
    (me.role === 'handler' && me.group === t.group);
}

function canReview(t) {
  return t.status === 'closed' && !t.review && (me.role === 'admin' || t.submitter.id === me.id);
}

function render(t) {
  const wrap = document.getElementById('detailWrap');
  const urgent = t.urgency === 'urgent' ? '<span class="urgent-tag">[紧急]</span>' : '';
  const overdueTag = t.overdue ? '<span class="badge" style="background:#fee2e2;color:#dc2626;">超时</span>' : '';

  // 状态操作按钮
  let statusBtns = '';
  if (t.status === 'pending' && canChangeStatus(t, 'processing')) {
    statusBtns += `<button class="btn" onclick="setStatus('processing')">接单/开始处理</button>`;
  }
  if (t.status === 'processing' && canChangeStatus(t, 'done')) {
    statusBtns += `<button class="btn" onclick="setStatus('done')">标记完成</button>`;
  }
  if (t.status === 'done' && canChangeStatus(t, 'closed')) {
    statusBtns += `<button class="btn" onclick="setStatus('closed')">确认关闭</button>`;
  }

  // 指派
  let assignHtml = '';
  if (canAssign(t) && t.status !== 'closed') {
    assignHtml = `<div class="card">
      <h3 class="card-title">指派处理人</h3>
      <div class="filter-bar">
        <select class="form-control" id="assigneeSel" style="min-width:200px;">
          <option value="">${t.assignee ? '当前：' + UI.esc(t.assignee.name) : '选择组内处理人'}</option>
        </select>
        <button class="btn" onclick="doAssign()">指派</button>
      </div>
    </div>`;
  }

  // 备注
  let commentHtml = '';
  if (canComment(t) && t.status !== 'closed') {
    commentHtml = `<div class="card">
      <h3 class="card-title">添加处理备注</h3>
      <textarea class="form-control" id="commentText" placeholder="输入备注内容"></textarea>
      <button class="btn" style="margin-top:10px;" onclick="addComment()">提交备注</button>
    </div>`;
  }

  // 评价
  let reviewHtml = '';
  if (canReview(t)) {
    reviewHtml = `<div class="card">
      <h3 class="card-title">评价处理</h3>
      ${starRow('speed', '处理速度')}
      ${starRow('quality', '处理质量')}
      ${starRow('communicate', '沟通体验')}
      <div class="form-group"><label>评语（可选）</label>
        <textarea class="form-control" id="reviewComment"></textarea></div>
      <button class="btn" onclick="submitReview()">提交评价</button>
    </div>`;
  } else if (t.review) {
    const r = t.review;
    reviewHtml = `<div class="card">
      <h3 class="card-title">处理评价</h3>
      <div class="info-row"><span class="label">处理速度</span><span class="stars">${UI.stars(r.speed)}</span></div>
      <div class="info-row"><span class="label">处理质量</span><span class="stars">${UI.stars(r.quality)}</span></div>
      <div class="info-row"><span class="label">沟通体验</span><span class="stars">${UI.stars(r.communicate)}</span></div>
      <div class="info-row"><span class="label">平均</span><span class="stars">${UI.stars(r.average)}</span> ${r.average.toFixed(1)}</div>
      ${r.comment ? `<div class="info-row"><span class="label">评语</span><span>${UI.esc(r.comment)}</span></div>` : ''}
    </div>`;
  }

  // 时间线
  const timeline = (t.comments || []).map(c => `
    <li class="${c.type === 'system' ? 'tl-system' : ''}">
      <div class="tl-time">${UI.fmtDate(c.created_at)}</div>
      <div><span class="tl-user">${c.user ? UI.esc(c.user.name) : '系统'}</span>：${UI.esc(c.content)}</div>
    </li>`).join('');

  // 附件
  const attachments = (t.attachments || []).map(a => `
    <div class="attach-item">
      <span>${UI.esc(a.filename)} <small style="color:var(--muted);">${UI.fmtSize(a.file_size)}</small></span>
      <a href="${a.download_url}" class="btn btn-ghost">下载</a>
    </div>`).join('') || '<div class="empty">无附件</div>';

  wrap.innerHTML = `
    <div class="card">
      <h2 class="card-title">${urgent} ${UI.esc(t.title)} ${overdueTag}
        <small style="font-size:14px; color:var(--muted);">${UI.esc(t.ticket_no)}</small></h2>
      ${statusBtns ? `<div style="margin-bottom:12px;">${statusBtns}</div>` : ''}
      <div class="detail-grid">
        <div>
          <div class="info-row"><span class="label">状态</span><span class="status ${UI.statusClass(t.status)}">${UI.statusLabel(t.status)}</span></div>
          <div class="info-row"><span class="label">类型</span>${t.category ? UI.esc(t.category.name) : '-'}</div>
          <div class="info-row"><span class="label">紧急程度</span>${UI.URGENCY_LABEL[t.urgency] || t.urgency}</div>
          <div class="info-row"><span class="label">处理组</span>${UI.esc(t.group) || '-'}</div>
          <div class="info-row"><span class="label">提交人</span>${t.submitter ? UI.esc(t.submitter.name) : '-'}</div>
          <div class="info-row"><span class="label">处理人</span>${t.assignee ? UI.esc(t.assignee.name) : '<span style="color:var(--muted);">未指派</span>'}</div>
          <div class="info-row"><span class="label">提交时间</span>${UI.fmtDate(t.submitted_at)}</div>
          <div class="info-row"><span class="label">完成时间</span>${UI.fmtDate(t.completed_at)}</div>
          <div class="info-row"><span class="label">关闭时间</span>${UI.fmtDate(t.closed_at)}</div>
          <div class="info-row" style="border-bottom:none;"><span class="label">描述</span><span>${UI.esc(t.description) || '-'}</span></div>
        </div>
        <div class="card" style="margin:0;">
          <h3 class="card-title" style="font-size:15px;">处理进度时间线</h3>
          <ul class="timeline">${timeline || '<li class="tl-system"><div class="tl-time">暂无记录</div></li>'}</ul>
        </div>
      </div>
    </div>

    <div class="card">
      <h3 class="card-title">附件</h3>
      ${attachments}
    </div>

    ${assignHtml}
    ${commentHtml}
    ${reviewHtml}
  `;

  if (assignHtml) loadAssignees(t);
}

function starRow(key, label) {
  return `<div class="form-group"><label>${label}</label>
    <div class="star-input" data-key="${key}">
      <span data-v="1">★</span><span data-v="2">★</span><span data-v="3">★</span>
      <span data-v="4">★</span><span data-v="5">★</span>
      <input type="hidden" id="star_${key}" value="0">
    </div></div>`;
}

async function loadAssignees(t) {
  try {
    const list = await API.get('/users/handlers', { group: t.group });
    const sel = document.getElementById('assigneeSel');
    const cur = t.assignee ? t.assignee.id : null;
    list.forEach(u => {
      if (u.id !== cur) {
        const opt = document.createElement('option');
        opt.value = u.id;
        opt.textContent = u.name + (u.is_leader ? '（组长）' : '');
        sel.appendChild(opt);
      }
    });
  } catch (e) { UI.toast(e.message, 'error'); }
}

// 星级交互
document.addEventListener('click', async (e) => {
  const span = e.target.closest('.star-input span');
  if (!span) {
    // 状态/备注/指派/评价按钮通过 onclick 调用全局函数
    return;
  }
  const v = +span.dataset.v;
  const input = span.closest('.star-input');
  input.querySelectorAll('span').forEach(s => s.classList.toggle('on', +s.dataset.v <= v));
  document.getElementById('star_' + input.dataset.key).value = v;
});

window.setStatus = async function (status) {
  try {
    await API.put('/tickets/' + ticketId + '/status', { status });
    UI.toast('状态已更新', 'success');
    load();
  } catch (err) { UI.toast(err.message, 'error'); }
};

window.doAssign = async function () {
  const id = document.getElementById('assigneeSel').value;
  if (!id) { UI.toast('请选择处理人', 'error'); return; }
  try {
    await API.put('/tickets/' + ticketId + '/assign', { assignee_id: +id });
    UI.toast('已指派', 'success');
    load();
  } catch (err) { UI.toast(err.message, 'error'); }
};

window.addComment = async function () {
  const content = document.getElementById('commentText').value.trim();
  if (!content) { UI.toast('备注不能为空', 'error'); return; }
  try {
    await API.post('/tickets/' + ticketId + '/comments', { content });
    UI.toast('备注已添加', 'success');
    load();
  } catch (err) { UI.toast(err.message, 'error'); }
};

window.submitReview = async function () {
  const body = {
    speed: +document.getElementById('star_speed').value,
    quality: +document.getElementById('star_quality').value,
    communicate: +document.getElementById('star_communicate').value,
    comment: document.getElementById('reviewComment').value,
  };
  if (!body.speed || !body.quality || !body.communicate) {
    UI.toast('请完成全部评分', 'error'); return;
  }
  try {
    await API.post('/tickets/' + ticketId + '/review', body);
    UI.toast('评价已提交', 'success');
    load();
  } catch (err) { UI.toast(err.message, 'error'); }
};

load();
Auth.startOverduePolling();
