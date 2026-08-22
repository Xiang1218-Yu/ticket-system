// api.js —— 数据访问层（单一职责：封装 fetch，自动带 JWT，统一错误处理）
const API = (function () {
  const BASE = '/api/v1';

  function getToken() {
    return localStorage.getItem('ts_token') || '';
  }

  async function request(method, path, { body, form, query } = {}) {
    const headers = {};
    const token = getToken();
    if (token) headers['Authorization'] = 'Bearer ' + token;

    let url = BASE + path;
    if (query) {
      const qs = new URLSearchParams();
      Object.entries(query).forEach(([k, v]) => {
        if (v !== undefined && v !== null && v !== '') qs.append(k, v);
      });
      const s = qs.toString();
      if (s) url += '?' + s;
    }

    let opts = { method, headers };
    if (form) {
      opts.body = form; // multipart，不设 Content-Type，浏览器自动加 boundary
    } else if (body !== undefined) {
      headers['Content-Type'] = 'application/json';
      opts.body = JSON.stringify(body);
    }
    opts.headers = headers;

    let resp;
    try {
      resp = await fetch(url, opts);
    } catch (e) {
      throw new Error('网络错误：' + e.message);
    }

    // 文件下载
    if (resp.headers.get('content-disposition')?.includes('attachment')) {
      const blob = await resp.blob();
      return { blob: blob, filename: getFilename(resp) };
    }

    const data = await resp.json().catch(() => ({}));
    if (!resp.ok) {
      const msg = data.message || ('请求失败 (' + resp.status + ')');
      if (resp.status === 401) {
        // 未登录/失效：清理并跳登录
        localStorage.removeItem('ts_token');
        if (location.pathname !== '/login' && location.pathname !== '/register') {
          location.href = '/login';
        }
      }
      throw new Error(msg);
    }
    return data.data !== undefined ? data.data : data;
  }

  function getFilename(resp) {
    const cd = resp.headers.get('content-disposition') || '';
    const m = cd.match(/filename="?([^"]+)"?/);
    return m ? m[1] : 'download';
  }

  return {
    get: (p, q) => request('GET', p, { query: q }),
    post: (p, b) => request('POST', p, { body: b }),
    put: (p, b) => request('PUT', p, { body: b }),
    upload: (p, form) => request('POST', p, { form }),
    download: (p) => request('GET', p),
    request,
    getToken,
  };
})();
