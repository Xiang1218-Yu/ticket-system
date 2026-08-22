// auth.js —— 认证状态层（单一职责：token 存取、登录态、路由守卫、登出、超时通知）
const Auth = (function () {
  function token() { return localStorage.getItem('ts_token'); }
  function isLoggedIn() { return !!token(); }

  function saveToken(t) { localStorage.setItem('ts_token', t); }
  function user() {
    try { return JSON.parse(localStorage.getItem('ts_user')); } catch { return null; }
  }
  function saveUser(u) { localStorage.setItem('ts_user', JSON.stringify(u)); }

  function logout() {
    localStorage.removeItem('ts_token');
    localStorage.removeItem('ts_user');
    location.href = '/login';
  }

  // requireAuth：页面需要登录；否则跳登录
  function requireAuth() {
    if (!isLoggedIn()) { location.href = '/login'; return false; }
    return true;
  }
  // redirectIfAuthed：已登录则跳首页（登录/注册页用）
  function redirectIfAuthed() {
    if (isLoggedIn()) { location.href = '/tickets'; return true; }
    return false;
  }

  function isRole(...roles) {
    const u = user();
    return u && roles.includes(u.role);
  }

  // ---- 超时通知（浏览器通知模拟）----
  let notifiedSet = new Set(JSON.parse(localStorage.getItem('ts_notified') || '[]'));
  function saveNotified() { localStorage.setItem('ts_notified', JSON.stringify([...notifiedSet])); }

  async function ensureNotifyPermission() {
    if (!('Notification' in window)) return false;
    if (Notification.permission === 'granted') return true;
    if (Notification.permission !== 'denied') {
      await Notification.requestPermission();
    }
    return Notification.permission === 'granted';
  }

  // pollOverdue：轮询超时工单，对新增项弹通知
  let timer = null;
  function startOverduePolling() {
    if (timer) return;
    ensureNotifyPermission();
    timer = setInterval(checkOverdue, 30000);
    checkOverdue();
  }
  function stopOverduePolling() { if (timer) { clearInterval(timer); timer = null; } }

  async function checkOverdue() {
    if (!isLoggedIn()) return;
    try {
      const res = await API.get('/tickets/overdue');
      const items = res.items || [];
      const newOnes = items.filter(t => !notifiedSet.has(t.id));
      if (newOnes.length > 0 && 'Notification' in window && Notification.permission === 'granted') {
        newOnes.forEach(t => {
          new Notification('工单超时提醒', {
            body: `工单 ${t.ticket_no}「${t.title}」已超时，请尽快处理`,
          });
          notifiedSet.add(t.id);
        });
        saveNotified();
      }
      // 触发自定义事件，供页面刷新超时清单
      window.dispatchEvent(new CustomEvent('overdue-updated', { detail: items }));
    } catch (e) { /* 静默 */ }
  }

  return {
    token, isLoggedIn, saveToken, user, saveUser, logout,
    requireAuth, redirectIfAuthed, isRole,
    startOverduePolling, stopOverduePolling, ensureNotifyPermission,
  };
})();
