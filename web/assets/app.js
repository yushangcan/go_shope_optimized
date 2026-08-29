(() => {
  const state = { token: localStorage.getItem('go_shope_token') || '', user: null, products: [], adminProducts: [], activities: [] };
  const $ = (selector) => document.querySelector(selector);
  const $$ = (selector) => [...document.querySelectorAll(selector)];
  const api = async (path, options = {}) => {
    const headers = { ...(options.headers || {}) };
    if (state.token) headers.Authorization = `Bearer ${state.token}`;
    if (options.body) headers['Content-Type'] = 'application/json';
    const response = await fetch(path, { ...options, headers });
    const data = response.status === 204 ? null : await response.json().catch(() => ({}));
    if (!response.ok) throw new Error(data.error || '请求暂时未成功，请稍后再试');
    return data;
  };
  const money = (cents = 0) => `￥${(Number(cents) / 100).toFixed(2)}`;
  const escapeHtml = (text = '') => String(text).replace(/[&<>"]/g, char => ({ '&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;' }[char]));
  const iconFor = (id) => ['⌁', '⌂', '✦', '♡', '◌'][Number(id || 0) % 5];
  const notify = (message) => { const toast = $('#toast'); if (!toast) return; toast.textContent = message; toast.classList.add('show'); clearTimeout(notify.timer); notify.timer = setTimeout(() => toast.classList.remove('show'), 2400); };
  const modal = (id, open) => { const node = $(`#${id}`); if (node) { node.classList.toggle('open', open); node.setAttribute('aria-hidden', String(!open)); } };
  const isoFromLocal = (value) => value ? new Date(value).toISOString() : '';
  const localInputValue = (value) => { if (!value) return ''; const date = new Date(value); const offset = date.getTimezoneOffset() * 60000; return new Date(date.getTime() - offset).toISOString().slice(0,16); };
  const localFromISO = (value) => value ? new Date(value).toLocaleString('zh-CN', { month:'numeric', day:'numeric', hour:'2-digit', minute:'2-digit' }) : '-';

  async function refreshIdentity() {
    if (!state.token) return null;
    try { state.user = await api('/api/users/me'); return state.user; }
    catch { localStorage.removeItem('go_shope_token'); state.token = ''; state.user = null; return null; }
  }
  function updateIdentityUI() {
    const account = $('#account-button'); const orderCount = $('#order-count'); const adminName = $('#admin-name'); const adminWelcome = $('#admin-welcome');
    if (state.user) { if (account) account.textContent = `${state.user.username} · 退出`; if (orderCount) orderCount.textContent = '查看订单'; if (adminName) adminName.textContent = state.user.username; if (adminWelcome) adminWelcome.textContent = state.user.username; }
    else { if (account) account.textContent = '登录 / 注册'; if (orderCount) orderCount.textContent = '未登录'; }
  }
  async function loadPublicData() {
    [state.products, state.activities] = await Promise.all([api('/api/products'), api('/api/seckill/activities')]);
    renderProducts(); renderActivities(); renderAdmin();
  }
  async function loadAdminData() {
    if (state.user?.role !== 'ADMIN') return;
    state.adminProducts = await api('/api/admin/products');
    renderAdmin();
  }
  function renderProducts() {
    const root = $('#product-list'); if (!root) return;
    const query = ($('#product-search')?.value || '').trim().toLowerCase();
    const products = state.products.filter(p => !query || `${p.name} ${p.description}`.toLowerCase().includes(query));
    root.innerHTML = products.length ? products.map(p => `<article class="product-card"><div class="product-image">${iconFor(p.id)}</div><div class="product-info"><h3>${escapeHtml(p.name)}</h3><p>${escapeHtml(p.description || '为日常准备的一份小心意')}</p><div class="product-foot"><div><b>${money(p.price)}</b><small>库存 ${p.stock}</small></div><button data-product-id="${p.id}" ${p.stock <= 0 ? 'disabled' : ''}>${p.stock <= 0 ? '暂时售罄' : '立即购买'}</button></div></div></article>`).join('') : '<p class="loading-text">暂时没有匹配的商品。</p>';
  }
  function renderActivities() {
    const root = $('#seckill-list'); if (!root) return;
    const now = Date.now(); const active = state.activities.filter(a => new Date(a.end_time).getTime() > now && a.status === 'ACTIVE');
    root.innerHTML = active.length ? active.slice(0,3).map(a => `<article class="sale-item"><div class="sale-image">${iconFor(a.product_id)}</div><div class="sale-info"><h3>${escapeHtml(a.product?.name || `秒杀商品 #${a.product_id}`)}</h3><p>仅剩 ${a.available_stock} 件 · 限量心动</p><div class="price-row"><b>${money(a.seckill_price)}</b><del>${money(a.product?.price || a.seckill_price)}</del></div><button data-seckill-id="${a.id}" ${a.available_stock <= 0 ? 'disabled' : ''}>${a.available_stock <= 0 ? '已抢完' : '立即抢购'}</button></div></article>`).join('') : '<p class="loading-text">今日秒杀正在准备中，稍后再来看看吧。</p>';
    const nearest = active.sort((a,b) => new Date(a.end_time)-new Date(b.end_time))[0]; updateCountdown(nearest?.end_time);
  }
  function updateCountdown(endTime) {
    const node = $('#sale-countdown'); if (!node) return;
    clearInterval(updateCountdown.timer); if (!endTime) { node.textContent = '敬请期待'; return; }
    const tick = () => { const diff = new Date(endTime).getTime() - Date.now(); if (diff <= 0) { node.textContent = '已结束'; clearInterval(updateCountdown.timer); return; } const h = String(Math.floor(diff / 3600000)).padStart(2,'0'); const m = String(Math.floor(diff % 3600000 / 60000)).padStart(2,'0'); const s = String(Math.floor(diff % 60000 / 1000)).padStart(2,'0'); node.textContent = `${h}:${m}:${s}`; }; tick(); updateCountdown.timer = setInterval(tick,1000);
  }
  function renderAdmin() {
    const productRoot = $('#admin-product-list'); const activityRoot = $('#admin-activity-list'); const select = $('#activity-product');
    const products = state.adminProducts.length ? state.adminProducts : state.products;
    if (productRoot) productRoot.innerHTML = products.length ? products.map(p => `<div class="admin-row"><div><b>${escapeHtml(p.name)}</b><small>${money(p.price)} · 库存 ${p.stock} · ${escapeHtml(p.description || '暂无描述')}</small></div><div class="row-right"><span class="status-badge">${p.status === 'ON_SALE' ? '已上架' : '已下架'}</span><button class="row-action" data-edit-product="${p.id}">编辑</button><button class="row-action danger" data-delete-product="${p.id}">删除</button></div></div>`).join('') : '<p class="muted">暂无商品，请先创建商品。</p>';
    if (activityRoot) activityRoot.innerHTML = state.activities.length ? state.activities.map(a => `<div class="admin-row"><div><b>${escapeHtml(a.product?.name || `商品 #${a.product_id}`)}</b><small>${money(a.seckill_price)} · 剩余 ${a.available_stock}/${a.total_stock} · ${localFromISO(a.end_time)} 截止</small></div><div class="row-right"><span class="status-badge ${a.status !== 'ACTIVE' ? 'pending' : ''}">${a.status}</span><button class="row-action" data-edit-activity="${a.id}">编辑</button><button class="row-action danger" data-delete-activity="${a.id}">删除</button></div></div>`).join('') : '<p class="muted">还没有秒杀活动。</p>';
    if (select) select.innerHTML = '<option value="">选择商品</option>' + products.map(p => `<option value="${p.id}">${escapeHtml(p.name)}（库存 ${p.stock}）</option>`).join('');
    const metricProducts = $('#metric-products'); const metricActivities = $('#metric-activities'); if (metricProducts) metricProducts.textContent = products.filter(p => p.status === 'ON_SALE').length; if (metricActivities) metricActivities.textContent = state.activities.length;
  }
  async function loadOrders() {
    const adminRoot = $('#admin-order-list'); const storeRoot = $('#store-order-list'); if (!adminRoot && !storeRoot) return;
    const root = adminRoot || storeRoot;
    if (!state.user) { root.innerHTML = '<p class="muted">登录后查看订单。</p>'; return; }
    try { const isAdminView = Boolean(adminRoot); const orders = await api(isAdminView ? '/api/admin/orders' : '/api/orders'); root.innerHTML = orders.length ? orders.map(o => `<div class="order-line"><div><b>${escapeHtml(o.product_name)}</b><small>${escapeHtml(o.order_no)}</small></div><span>${money(o.total_amount)}</span><span class="status-badge ${o.status === 'PENDING' ? 'pending' : o.status === 'CANCELLED' ? 'cancelled' : ''}">${o.status}</span>${!isAdminView && o.status === 'PENDING' ? `<span><button data-pay-order="${o.id}">支付</button> <button data-cancel-order="${o.id}">取消</button></span>` : '<span></span>'}</div>`).join('') : '<p class="muted">暂时没有订单。</p>'; const sales = orders.filter(o => o.status === 'PAID').reduce((total,o) => total + Number(o.total_amount),0); const salesNode = $('#metric-sales'); const ordersNode = $('#metric-orders'); if (salesNode) salesNode.textContent = money(sales); if (ordersNode) ordersNode.textContent = orders.length; }
    catch (error) { root.innerHTML = `<p class="muted">${escapeHtml(error.message)}</p>`; }
  }
  async function submitLogin(form, admin = false) { const username = form.querySelector('input[type="text"], input:not([type])')?.value.trim(); const password = form.querySelector('input[type="password"]')?.value; const result = await api('/api/auth/login', { method:'POST', body: JSON.stringify({ username, password }) }); state.token = result.token; localStorage.setItem('go_shope_token', state.token); await refreshIdentity(); if (admin && state.user?.role !== 'ADMIN') { localStorage.removeItem('go_shope_token'); state.token = ''; state.user = null; throw new Error('该账号不是管理员'); } notify('登录成功，欢迎回来'); updateIdentityUI(); modal(admin ? 'admin-auth-modal' : 'auth-modal', false); await loadAdminData(); await loadOrders(); }
  async function createSeckillOrder(activityID) { if (!state.token) { modal('auth-modal', true); return; } try { const order = await api(`/api/seckill/activities/${activityID}/orders`, { method:'POST', body: JSON.stringify({ request_id: `${Date.now()}-${crypto.randomUUID?.() || Math.random()}` }) }); notify(`秒杀下单成功，订单号：${order.order_no}`); await loadPublicData(); await loadOrders(); } catch (error) { notify(error.message); } }
  async function createProductOrder(productID) { if (!state.token) { modal('auth-modal', true); return; } try { const order = await api(`/api/products/${productID}/orders`, { method:'POST', body: JSON.stringify({ quantity: 1, request_id: `${Date.now()}-${crypto.randomUUID?.() || Math.random()}` }) }); notify(`普通商品下单成功，订单号：${order.order_no}`); await loadPublicData(); await loadOrders(); } catch (error) { notify(error.message); } }
  async function createProduct(form) { const data = Object.fromEntries(new FormData(form)); const id = data.id; delete data.id; data.price = Number(data.price); data.stock = Number(data.stock); try { await api(id ? `/api/admin/products/${id}` : '/api/admin/products', { method:id ? 'PUT' : 'POST', body:JSON.stringify(data) }); notify(id ? '商品已更新' : '商品已创建'); form.reset(); form.classList.add('hidden'); await loadPublicData(); await loadAdminData(); } catch (error) { notify(error.message); } }
  async function createActivity(form) { const data = Object.fromEntries(new FormData(form)); const id = data.id; delete data.id; const existing = id ? state.activities.find(item => item.id === Number(id)) : null; data.product_id = existing ? existing.product_id : Number(data.product_id); data.seckill_price = Number(data.seckill_price); data.total_stock = Number(data.total_stock); data.start_time = isoFromLocal(data.start_time); data.end_time = isoFromLocal(data.end_time); try { await api(id ? `/api/admin/seckill/activities/${id}` : '/api/admin/seckill/activities', { method:id ? 'PUT' : 'POST', body:JSON.stringify(data) }); notify(id ? '秒杀活动已更新' : '秒杀活动已创建'); form.reset(); form.elements.product_id.disabled = false; form.classList.add('hidden'); await loadPublicData(); await loadAdminData(); } catch (error) { notify(error.message); } }
  function editProduct(id) { const product = (state.adminProducts.length ? state.adminProducts : state.products).find(item => item.id === Number(id)); const form = $('#product-form'); if (!product || !form) return; form.elements.id.value = product.id; form.elements.name.value = product.name; form.elements.price.value = product.price; form.elements.stock.value = product.stock; form.elements.description.value = product.description || ''; form.elements.status.value = product.status; form.classList.remove('hidden'); form.scrollIntoView({behavior:'smooth', block:'center'}); }
  function editActivity(id) { const activity = state.activities.find(item => item.id === Number(id)); const form = $('#activity-form'); if (!activity || !form) return; form.elements.id.value = activity.id; form.elements.product_id.value = activity.product_id; form.elements.product_id.disabled = true; form.elements.seckill_price.value = activity.seckill_price; form.elements.total_stock.value = activity.total_stock; form.elements.start_time.value = localInputValue(activity.start_time); form.elements.end_time.value = localInputValue(activity.end_time); form.elements.status.value = activity.status; form.classList.remove('hidden'); form.scrollIntoView({behavior:'smooth', block:'center'}); }
  async function deleteAdminResource(path, message) { if (!window.confirm('确认删除这条记录吗？')) return; try { await api(path, {method:'DELETE'}); notify(message); await loadPublicData(); await loadAdminData(); } catch (error) { notify(error.message); } }
  function bindEvents() {
    $('#account-button')?.addEventListener('click', () => state.user ? (localStorage.removeItem('go_shope_token'), state.token='',state.user=null,state.adminProducts=[],updateIdentityUI(),loadOrders(),notify('已退出登录')) : modal('auth-modal', true));
    $('#orders-button')?.addEventListener('click', async () => { if (!state.user) { modal('auth-modal', true); return; } await loadOrders(); modal('orders-modal', true); });
    $('#admin-login-button')?.addEventListener('click', () => modal('admin-auth-modal', true));
    $$('.close-modal').forEach(button => button.addEventListener('click', () => modal(button.dataset.close, false)));
    $('#auth-form')?.addEventListener('submit', async event => { event.preventDefault(); try { await submitLogin(event.currentTarget); } catch (error) { notify(error.message); } });
    $('#admin-auth-form')?.addEventListener('submit', async event => { event.preventDefault(); try { await submitLogin(event.currentTarget, true); } catch (error) { notify(error.message); } });
    $('#register-button')?.addEventListener('click', async () => { const username = $('#auth-username').value.trim(); const password = $('#auth-password').value; try { await api('/api/auth/register',{method:'POST',body:JSON.stringify({username,password})}); notify('账号已创建，请点击登录'); } catch(error) { notify(error.message); } });
    $('#search-button')?.addEventListener('click', renderProducts); $('#product-search')?.addEventListener('input', renderProducts);
    $('#show-product-form')?.addEventListener('click', () => { const form = $('#product-form'); form.reset(); form.classList.toggle('hidden'); }); $('#show-activity-form')?.addEventListener('click', () => { const form = $('#activity-form'); form.reset(); form.elements.product_id.disabled = false; form.classList.toggle('hidden'); });
    $('#product-form')?.addEventListener('submit', event => { event.preventDefault(); createProduct(event.currentTarget); }); $('#activity-form')?.addEventListener('submit', event => { event.preventDefault(); createActivity(event.currentTarget); }); $('#refresh-orders')?.addEventListener('click', loadOrders);
    $$('[data-scroll]').forEach(button => button.addEventListener('click', () => $(`#${button.dataset.scroll}`)?.scrollIntoView({ behavior:'smooth' })));
    document.addEventListener('click', event => { const saleButton = event.target.closest('[data-seckill-id]'); if (saleButton) createSeckillOrder(saleButton.dataset.seckillId); const productButton = event.target.closest('[data-product-id]'); if (productButton) createProductOrder(productButton.dataset.productId); const payButton = event.target.closest('[data-pay-order]'); if (payButton) api(`/api/orders/${payButton.dataset.payOrder}/pay`, {method:'POST'}).then(() => {notify('订单支付成功');return loadOrders();}).catch(error=>notify(error.message)); const cancelButton = event.target.closest('[data-cancel-order]'); if (cancelButton) api(`/api/orders/${cancelButton.dataset.cancelOrder}/cancel`, {method:'POST'}).then(() => {notify('订单已取消，库存已恢复');return loadOrders();}).then(loadPublicData).catch(error=>notify(error.message)); const editProductButton = event.target.closest('[data-edit-product]'); if (editProductButton) editProduct(editProductButton.dataset.editProduct); const deleteProductButton = event.target.closest('[data-delete-product]'); if (deleteProductButton) deleteAdminResource(`/api/admin/products/${deleteProductButton.dataset.deleteProduct}`, '商品已删除'); const editActivityButton = event.target.closest('[data-edit-activity]'); if (editActivityButton) editActivity(editActivityButton.dataset.editActivity); const deleteActivityButton = event.target.closest('[data-delete-activity]'); if (deleteActivityButton) deleteAdminResource(`/api/admin/seckill/activities/${deleteActivityButton.dataset.deleteActivity}`, '秒杀活动已删除'); });
  }
  async function init() { bindEvents(); $('#today-date') && ($('#today-date').textContent = new Date().toLocaleDateString('zh-CN',{year:'numeric',month:'long',day:'numeric',weekday:'long'})); await refreshIdentity(); updateIdentityUI(); try { await loadPublicData(); await loadAdminData(); await loadOrders(); } catch (error) { notify(`页面数据加载失败：${error.message}`); } }
  init();
})();
