// Yaria Desktop - SPA Router with Tab Navigation
const app = document.getElementById('app');
const navRight = document.getElementById('nav-right');
const tabYaria = document.getElementById('tab-yaria');
const tabMantorex = document.getElementById('tab-mantorex');

let activeTab = 'yaria';
let _proChecked = false;
let _isPro = false;

window.addEventListener('scroll', () => {
  const navbar = document.getElementById('navbar');
  navbar.style.background = window.scrollY > 10 ? 'rgba(10,10,15,0.95)' : 'rgba(10,10,15,0.8)';
});

function getActiveTab(path) {
  if (path.startsWith('/yaria')) return 'yaria';
  if (path === '/settings') return activeTab;
  return 'mantorex';
}

function updateNavRight(tab) {
  if (tab === 'yaria') {
    navRight.innerHTML = `
      <a href="#/yaria/downloads" class="nav-link">Downloads</a>
      <a href="#/settings" class="nav-link">Settings</a>
    `;
  } else {
    navRight.innerHTML = `
      <div class="search-box">
        <input type="text" id="search-input" placeholder="Search..." autocomplete="off">
      </div>
      <a href="#/torrent-downloads" class="nav-link">Downloads</a>
      <a href="#/library" class="nav-link">Library</a>
      <a href="#/settings" class="nav-link">Settings</a>
    `;
    const searchInput = document.getElementById('search-input');
    if (searchInput) {
      searchInput.addEventListener('keydown', (e) => {
        if (e.key === 'Enter') {
          const q = searchInput.value.trim();
          if (q) window.location.hash = `#/search?q=${encodeURIComponent(q)}`;
        }
      });
    }
  }
}

function updateTabHighlight(tab) {
  tabYaria.classList.toggle('active', tab === 'yaria');
  tabMantorex.classList.toggle('active', tab === 'mantorex');
}

async function checkPro() {
  if (_proChecked) return _isPro;
  // Wait for Wails runtime to be ready
  let retries = 0;
  while ((!window.go || !window.go.main || !window.go.main.LicenseService) && retries < 50) {
    await new Promise(r => setTimeout(r, 100));
    retries++;
  }
  try {
    _isPro = await API.isPro();
  } catch (e) {
    _isPro = false;
  }
  _proChecked = true;
  return _isPro;
}

function resetProCache() { _proChecked = false; _isPro = false; }

function isMantorexRoute(path) {
  return path === '/mantorex' || path === '/mantorex/' ||
         path === '/search' || path === '/detail' ||
         path === '/play' || path === '/library' ||
         path === '/torrent-downloads';
}

// Activation gate shown when Pro is not available
async function renderActivationGate(container) {
  container.innerHTML = '';

  // Check if this is a free build (no Pro module compiled) vs just no license
  let licenseInfo = null;
  try { licenseInfo = await API.checkLicense(); } catch(e) {}
  const proAvailable = licenseInfo && licenseInfo.pro_available;
  const hasValidKey = licenseInfo && licenseInfo.valid && licenseInfo.plan === 'pro';

  const page = document.createElement('div');
  page.style.cssText = 'max-width:500px;margin:0 auto;padding:80px 20px;text-align:center;';

  if (!proAvailable) {
    // Free build -- Mantorex code not compiled in
    page.innerHTML = `
      <h1 style="font-size:48px;font-weight:800;background:linear-gradient(135deg,#c4b5fd,#8b5cf6,#ec4899);-webkit-background-clip:text;-webkit-text-fill-color:transparent;background-clip:text;margin-bottom:12px;">Mantorex</h1>
      <p style="color:var(--text-dim);font-size:16px;margin-bottom:8px;">Available with Yaria Pro</p>
      <p style="color:var(--text-muted);font-size:13px;margin-bottom:40px;">This is the free version of Yaria. Mantorex features require the Pro build.</p>
      <div style="background:rgba(255,255,255,0.03);border:1px solid rgba(255,255,255,0.06);border-radius:var(--radius);padding:28px;">
        <p style="color:var(--text-dim);font-size:14px;margin-bottom:16px;">Download Yaria Pro to unlock Mantorex</p>
        <a href="https://yaria.app" target="_blank" class="btn btn-primary" style="display:inline-block;text-decoration:none;">Get Yaria Pro</a>
      </div>
      ${hasValidKey ? '<p style="color:var(--green);font-size:12px;margin-top:16px;">You have a valid Pro license. Download the Pro build to use it.</p>' : ''}
      <div id="gate-device" style="color:var(--text-muted);font-size:11px;margin-top:12px;"></div>
    `;
  } else {
    // Pro build but no valid license
    page.innerHTML = `
      <h1 style="font-size:48px;font-weight:800;background:linear-gradient(135deg,#c4b5fd,#8b5cf6,#ec4899);-webkit-background-clip:text;-webkit-text-fill-color:transparent;background-clip:text;margin-bottom:12px;">Mantorex</h1>
      <p style="color:var(--text-dim);font-size:16px;margin-bottom:8px;">Available with Yaria Pro</p>
      <p style="color:var(--text-muted);font-size:13px;margin-bottom:40px;">Unlock all features with a Pro license</p>
      <div style="background:rgba(255,255,255,0.03);border:1px solid rgba(255,255,255,0.06);border-radius:var(--radius);padding:28px;text-align:left;">
        <label style="display:block;font-size:13px;font-weight:600;color:var(--text-dim);margin-bottom:8px;">Enter your Pro license key</label>
        <input type="text" id="gate-key-input" placeholder="YARIA-XXXX-XXXX-XXXX" style="width:100%;padding:12px 16px;background:rgba(255,255,255,0.04);border:1px solid rgba(255,255,255,0.08);border-radius:var(--radius-sm);color:var(--text);font-size:14px;outline:none;margin-bottom:16px;">
        <button class="btn btn-primary" id="gate-activate-btn" style="width:100%;padding:12px;font-size:14px;">Activate Pro</button>
        <div id="gate-error" style="display:none;margin-top:12px;font-size:13px;color:var(--red);"></div>
        <div id="gate-success" style="display:none;margin-top:12px;font-size:13px;color:var(--green);"></div>
      </div>
      <p style="color:var(--text-muted);font-size:12px;margin-top:20px;">
        Get a license at <a href="https://yaria.app" target="_blank" style="color:var(--accent);">yaria.app</a>
      </p>
      <div id="gate-device" style="color:var(--text-muted);font-size:11px;margin-top:12px;"></div>
    `;

    const btn = page.querySelector('#gate-activate-btn');
    const input = page.querySelector('#gate-key-input');
    const errEl = page.querySelector('#gate-error');
    const okEl = page.querySelector('#gate-success');

    btn.addEventListener('click', async () => {
      const key = input.value.trim();
      if (!key) { errEl.textContent = 'Please enter a license key'; errEl.style.display = 'block'; return; }
      btn.disabled = true; btn.textContent = 'Activating...';
      errEl.style.display = 'none'; okEl.style.display = 'none';
      try {
        const result = await API.activateKey(key);
        if (result.error) {
          errEl.textContent = result.error; errEl.style.display = 'block';
        } else if (result.valid) {
          okEl.textContent = 'Pro activated! Loading Mantorex...'; okEl.style.display = 'block';
          resetProCache();
          setTimeout(() => { window.location.hash = '#/mantorex'; route(); }, 1000);
        } else {
          errEl.textContent = 'Invalid license key'; errEl.style.display = 'block';
        }
      } catch (e) {
        errEl.textContent = e.message || 'Activation failed'; errEl.style.display = 'block';
      }
      btn.disabled = false; btn.textContent = 'Activate Pro';
    });
    input.addEventListener('keydown', (e) => { if (e.key === 'Enter') btn.click(); });
  }

  container.appendChild(page);

  API.getDeviceInfo().then(info => {
    const el = document.getElementById('gate-device');
    if (el) el.textContent = `Device: ${info.device_name} (${info.device_id})`;
  }).catch(() => {});
}

// Route Pro pages
function routePro(path, params) {
  if (path === '/mantorex' || path === '/mantorex/') {
    if (typeof renderHome === 'function') renderHome(app);
    else renderActivationGate(app);
  } else if (path === '/search') {
    const q = params.get('q') || '';
    if (q && typeof renderSearch === 'function') {
      const si = document.getElementById('search-input');
      if (si) si.value = q;
      renderSearch(app, q);
    } else if (typeof renderHome === 'function') {
      renderHome(app);
    }
  } else if (path === '/detail' && typeof renderDetail === 'function') {
    renderDetail(app, params);
  } else if (path === '/play' && typeof renderPlayer === 'function') {
    renderPlayer(app, params.get('magnet')||'', params.get('title')||'', params.get('poster')||'');
  } else if (path === '/library' && typeof renderLibrary === 'function') {
    renderLibrary(app);
  } else if (path === '/torrent-downloads') {
    renderTorrentDownloads(app);
  } else {
    renderActivationGate(app);
  }
}

// Main router
async function route() {
  const hash = window.location.hash || '#/yaria';
  const [path, queryStr] = hash.substring(1).split('?');
  const params = new URLSearchParams(queryStr || '');

  // Save detail page scroll position before navigating away
  if (typeof _detailCache !== 'undefined' && _detailCache.key) {
    _detailCache.scroll = window.scrollY;
  }

  // Cleanup
  if (typeof cleanupPlayer === 'function' && !path.startsWith('/play')) cleanupPlayer();
  if (typeof cleanupYariaHome === 'function' && !path.startsWith('/yaria')) cleanupYariaHome();
  if (typeof cleanupYariaDownloads === 'function' && path !== '/yaria/downloads') cleanupYariaDownloads();
  if (typeof cleanupTorrentDownloads === 'function' && path !== '/torrent-downloads') cleanupTorrentDownloads();

  // Don't reset scroll for cached pages (handled by the page itself)
  if (path !== '/detail') {
    window.scrollTo(0, 0);
  }

  // Trigger page transition animation
  app.style.animation = 'none';
  app.offsetHeight; // force reflow
  app.style.animation = 'pageIn 0.25s ease-out';

  activeTab = getActiveTab(path);
  updateTabHighlight(activeTab);
  updateNavRight(activeTab);

  // Hide nav search on Mantorex home
  const navSearchBox = document.querySelector('#nav-right .search-box');
  if (navSearchBox) navSearchBox.classList.toggle('hidden', path === '/mantorex' || path === '/mantorex/');

  // Yaria routes (always available)
  if (path === '/yaria' || path === '/yaria/') {
    renderYariaHome(app); return;
  }
  if (path === '/yaria/downloads') {
    renderYariaDownloads(app); return;
  }
  if (path === '/settings') {
    renderSettings(app); return;
  }
  if (path === '/' || path === '') {
    window.location.hash = '#/yaria'; return;
  }

  // Mantorex routes (require Pro)
  if (isMantorexRoute(path)) {
    const pro = await checkPro();
    if (!pro) { await renderActivationGate(app); return; }
    routePro(path, params);
    return;
  }

  // Unknown route
  window.location.hash = activeTab === 'yaria' ? '#/yaria' : '#/mantorex';
}

window.addEventListener('hashchange', route);
route();
