// register.js —— 注册页逻辑（单一职责：注册表单）
Auth.redirectIfAuthed();

const roleSel = document.getElementById('role');
const groupField = document.getElementById('groupField');
roleSel.addEventListener('change', () => {
  groupField.style.display = roleSel.value === 'handler' ? '' : 'none';
});

document.getElementById('regForm').addEventListener('submit', async (e) => {
  e.preventDefault();
  const form = e.target;
  const body = {
    username: form.username.value.trim(),
    name: form.name.value.trim(),
    password: form.password.value,
    role: form.role.value,
    group: form.role.value === 'handler' ? form.group.value : '',
    is_leader: false,
  };
  try {
    await API.post('/auth/register', body);
    UI.toast('注册成功，请登录', 'success');
    setTimeout(() => location.href = '/login', 600);
  } catch (err) {
    UI.toast(err.message, 'error');
  }
});
