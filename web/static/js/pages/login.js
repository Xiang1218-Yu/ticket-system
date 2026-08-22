// login.js —— 登录页逻辑（单一职责：登录表单）
Auth.redirectIfAuthed();

document.getElementById('loginForm').addEventListener('submit', async (e) => {
  e.preventDefault();
  const form = e.target;
  const username = form.username.value.trim();
  const password = form.password.value;
  if (!username || !password) return;
  try {
    const data = await API.post('/auth/login', { username, password });
    Auth.saveToken(data.token);
    Auth.saveUser(data.user);
    Auth.startOverduePolling();
    UI.toast('登录成功', 'success');
    setTimeout(() => location.href = '/dashboard', 400);
  } catch (err) {
    UI.toast(err.message, 'error');
  }
});
