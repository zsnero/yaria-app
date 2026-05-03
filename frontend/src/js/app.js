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
  if (_proChecked && _isPro) return true; // only cache positive results
  // Wait for Wails runtime to be ready
  let retries = 0;
  while ((!window.go || !window.go.main || !window.go.main.LicenseService) && retries < 100) {
    await new Promise(r => setTimeout(r, 100));
    retries++;
  }
  try {
    _isPro = await API.isPro();
    _proChecked = true;
  } catch (e) {
    _isPro = false;
    _proChecked = false; // don't cache failures -- retry next time
  }
  return _isPro;
}

function resetProCache() { _proChecked = false; _isPro = false; }

function isMantorexRoute(path) {
  return path === '/mantorex' || path === '/mantorex/' ||
         path === '/search' || path === '/detail' ||
         path === '/play' || path === '/library' ||
         path === '/torrent-downloads' ||
         path === '/local' || path === '/remote';
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
        <a href="https://yaria.live" target="_blank" class="btn btn-primary" style="display:inline-block;text-decoration:none;">Get Yaria Pro</a>
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
      <div style="background:rgba(255,255,255,0.03);border:1px solid rgba(255,255,255,0.06);border-radius:var(--radius);padding:28px;max-width:380px;margin:0 auto;">
        <label style="display:block;font-size:13px;font-weight:600;color:var(--text-dim);margin-bottom:8px;text-align:left;">Enter your Pro license key</label>
        <input type="text" id="gate-key-input" placeholder="YARIA-XXXX-XXXX-XXXX" style="width:100%;padding:12px 16px;background:rgba(255,255,255,0.04);border:1px solid rgba(255,255,255,0.08);border-radius:var(--radius-sm);color:var(--text);font-size:14px;outline:none;margin-bottom:16px;text-align:center;letter-spacing:1px;">
        <button class="btn btn-primary" id="gate-activate-btn" style="width:100%;padding:12px;font-size:14px;text-align:center;">Activate Pro</button>
        <div id="gate-error" style="display:none;margin-top:12px;font-size:13px;color:var(--red);text-align:center;"></div>
        <div id="gate-success" style="display:none;margin-top:12px;font-size:13px;color:var(--green);text-align:center;"></div>
      </div>
      <p style="color:var(--text-muted);font-size:12px;margin-top:20px;">
        Get a license at <a href="https://yaria.live" target="_blank" style="color:var(--accent);">yaria.live</a>
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

// Disclaimer gate -- shown once before first Mantorex use
function renderDisclaimer(container) {
  container.innerHTML = '';
  const page = document.createElement('div');
  page.style.cssText = 'max-width:600px;margin:0 auto;padding:60px 24px;';
  page.innerHTML = `
    <div style="text-align:center;margin-bottom:32px;">
      <h1 style="font-size:36px;font-weight:800;background:linear-gradient(135deg,#c4b5fd,#8b5cf6,#ec4899);-webkit-background-clip:text;-webkit-text-fill-color:transparent;background-clip:text;margin-bottom:8px;">Mantorex</h1>
      <p style="color:var(--text-dim);font-size:14px;">Torrent Search & Streaming</p>
    </div>
    <div style="background:rgba(255,255,255,0.03);border:1px solid rgba(255,255,255,0.06);border-radius:var(--radius);padding:28px;">
      <h2 style="font-size:18px;font-weight:700;margin-bottom:16px;color:var(--yellow);">Legal Disclaimer</h2>
      <div style="font-size:13px;color:var(--text-dim);line-height:1.7;">
        <p style="margin-bottom:12px;">Mantorex is a <strong style="color:var(--text);">torrent search engine and BitTorrent client</strong>. It does not host, upload, or distribute any content. All search results are fetched from third-party indexing websites.</p>
        <p style="margin-bottom:12px;">By using Mantorex, you acknowledge and agree that:</p>
        <ul style="padding-left:20px;margin-bottom:12px;">
          <li style="margin-bottom:8px;"><strong style="color:var(--text);">You are solely responsible</strong> for ensuring that your use of BitTorrent and any content you download or stream complies with all applicable laws in your jurisdiction.</li>
          <li style="margin-bottom:8px;">Downloading or streaming <strong style="color:var(--text);">copyrighted material without authorization</strong> from the copyright holder may be illegal in your country.</li>
          <li style="margin-bottom:8px;">The developers of Yaria <strong style="color:var(--text);">do not condone, encourage, or promote</strong> the downloading or streaming of copyrighted content without proper authorization.</li>
          <li style="margin-bottom:8px;">The developers assume <strong style="color:var(--text);">no liability</strong> for how you choose to use this software or the content you access through it.</li>
          <li style="margin-bottom:8px;">Torrent search results are provided by <strong style="color:var(--text);">third-party websites</strong> over which we have no control. We make no guarantees about the accuracy, legality, or safety of listed content.</li>
        </ul>
        <p style="margin-bottom:0;">Use this feature at your own risk and discretion. If you do not agree with these terms, click "Decline" to return to the Yaria downloader.</p>
      </div>
    </div>
    <div style="display:flex;gap:12px;margin-top:24px;justify-content:center;">
      <button class="btn btn-ghost" id="disc-decline" style="padding:12px 32px;">Decline</button>
      <button class="btn btn-primary" id="disc-accept" style="padding:12px 32px;">I Understand & Accept</button>
    </div>
  `;
  container.appendChild(page);

  page.querySelector('#disc-accept').addEventListener('click', () => {
    localStorage.setItem('mantorex_disclaimer_accepted', Date.now().toString());
    route();
  });
  page.querySelector('#disc-decline').addEventListener('click', () => {
    window.location.hash = '#/yaria';
  });
}

// Route Pro pages
function routePro(path, params) {
  // Show mode switcher on Mantorex pages (except player)
  if (path !== '/play' && typeof renderModeSwitcher === 'function') {
    renderModeSwitcher();
  } else if (typeof removeModeSwitcher === 'function') {
    removeModeSwitcher();
  }

  if (path === '/local') {
    if (typeof setMantorexMode === 'function') setMantorexMode('local');
    if (typeof renderLocalHome === 'function') renderLocalHome(app);
    else renderActivationGate(app);
  } else if (path === '/remote') {
    if (typeof setMantorexMode === 'function') setMantorexMode('remote');
    if (typeof renderRemoteHome === 'function') renderRemoteHome(app);
    else renderActivationGate(app);
  } else if (path === '/mantorex' || path === '/mantorex/') {
    if (typeof setMantorexMode === 'function') setMantorexMode('torrents');
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
    const localId = params.get('local') || '';
    const remoteStream = params.get('remoteStream') || '';
    if (localId) {
      renderLocalPlayer(app, localId, params.get('title')||'', params.get('poster')||'');
    } else if (remoteStream) {
      renderRemotePlayer(app, remoteStream, params.get('title')||'');
    } else {
      renderPlayer(app, params.get('magnet')||'', params.get('title')||'', params.get('poster')||'');
    }
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

  // Yaria routes (always available) -- hide mode switcher
  if (path === '/yaria' || path === '/yaria/' || path === '/yaria/downloads' || path === '/settings') {
    if (typeof removeModeSwitcher === 'function') removeModeSwitcher();
  }
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
    // First-time disclaimer gate
    if (!localStorage.getItem('mantorex_disclaimer_accepted')) {
      renderDisclaimer(app); return;
    }
    routePro(path, params);
    return;
  }

  // Unknown route
  window.location.hash = activeTab === 'yaria' ? '#/yaria' : '#/mantorex';
}

window.addEventListener('hashchange', route);
route();
