// Settings page -- sidebar navigation with content panels
async function renderSettings(container) {
  container.innerHTML = '';

  const page = document.createElement('div');
  page.className = 'settings-page';

  // Check if pro features are available
  let _settingsPro = false;
  try { _settingsPro = await API.isPro(); } catch(e) {}

  const sections = [
    { id: 'general', label: 'General', icon: 'gear' },
    { id: 'downloader', label: 'Downloader', icon: 'download' },
    ...(_settingsPro ? [
      { id: 'torrents', label: 'Torrents', icon: 'bolt' },
      { id: 'media', label: 'Local Media', icon: 'folder' },
      { id: 'server', label: 'Media Server', icon: 'server' },
    ] : []),
    { id: 'about', label: 'About', icon: 'info' },
  ];

  const icons = {
    gear: '<svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor"><path d="M19.14 12.94c.04-.3.06-.61.06-.94 0-.32-.02-.64-.07-.94l2.03-1.58a.49.49 0 0 0 .12-.61l-1.92-3.32a.49.49 0 0 0-.59-.22l-2.39.96c-.5-.38-1.03-.7-1.62-.94l-.36-2.54a.48.48 0 0 0-.48-.41h-3.84c-.24 0-.43.17-.47.41l-.36 2.54c-.59.24-1.13.57-1.62.94l-2.39-.96a.49.49 0 0 0-.59.22L2.74 8.87c-.12.21-.08.47.12.61l2.03 1.58c-.05.3-.07.62-.07.94s.02.64.07.94l-2.03 1.58a.49.49 0 0 0-.12.61l1.92 3.32c.12.22.37.29.59.22l2.39-.96c.5.38 1.03.7 1.62.94l.36 2.54c.05.24.24.41.48.41h3.84c.24 0 .44-.17.47-.41l.36-2.54c.59-.24 1.13-.56 1.62-.94l2.39.96c.22.08.47 0 .59-.22l1.92-3.32c.12-.22.07-.47-.12-.61l-2.01-1.58zM12 15.6A3.6 3.6 0 1 1 12 8.4a3.6 3.6 0 0 1 0 7.2z"/></svg>',
    download: '<svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor"><path d="M19 9h-4V3H9v6H5l7 7 7-7zM5 18v2h14v-2H5z"/></svg>',
    bolt: '<svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor"><path d="M11 21h-1l1-7H7.5c-.88 0-.33-.75-.31-.78C8.48 10.94 10.42 7.54 13.01 3h1l-1 7h3.51c.4 0 .62.19.4.66C12.97 17.55 11 21 11 21z"/></svg>',
    folder: '<svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor"><path d="M10 4H4c-1.1 0-2 .9-2 2v12c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V8c0-1.1-.9-2-2-2h-8l-2-2z"/></svg>',
    server: '<svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor"><path d="M4 1h16c1.1 0 2 .9 2 2v4c0 1.1-.9 2-2 2H4c-1.1 0-2-.9-2-2V3c0-1.1.9-2 2-2zm0 8h16c1.1 0 2 .9 2 2v4c0 1.1-.9 2-2 2H4c-1.1 0-2-.9-2-2v-4c0-1.1.9-2 2-2zm0 8h16c1.1 0 2 .9 2 2v2c0 1.1-.9 2-2 2H4c-1.1 0-2-.9-2-2v-2c0-1.1.9-2 2-2zM7 4.5a1.5 1.5 0 1 0 0 3 1.5 1.5 0 0 0 0-3zm0 8a1.5 1.5 0 1 0 0 3 1.5 1.5 0 0 0 0-3z"/></svg>',
    info: '<svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor"><path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-6h2v6zm0-8h-2V7h2v2z"/></svg>',
  };

  page.innerHTML = `
    <div class="stg-layout">
      <div class="stg-sidebar">
        <button class="stg-back" id="settings-back">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor"><path d="M20 11H7.83l5.59-5.59L12 4l-8 8 8 8 1.41-1.41L7.83 13H20v-2z"/></svg>
          Back
        </button>
        <div class="stg-nav">
          ${sections.map((s, i) => `
            <button class="stg-nav-item${i === 0 ? ' active' : ''}" data-section="${s.id}">
              <span class="stg-nav-icon">${icons[s.icon]}</span>
              ${s.label}
            </button>
          `).join('')}
        </div>
        <div class="stg-version" id="stg-version-label">Yaria</div>
      </div>
      <div class="stg-content">
        <!-- General -->
        <div class="stg-panel active" data-panel="general">
          <h3 class="stg-panel-title">General</h3>
          <div class="setting-group" id="license-section">
            <div class="setting-label">Yaria Pro License</div>
            <div id="license-status">
              <div class="spinner" style="width:18px;height:18px;border-width:2px;display:inline-block;vertical-align:middle;"></div>
              <span style="color:var(--text-muted);font-size:13px;margin-left:8px;">Checking...</span>
            </div>
          </div>
          <div class="setting-group">
            <div class="setting-label">TMDB API Key</div>
            <div class="setting-desc">Enables trending content, posters, and metadata. Get a free key at <a href="https://www.themoviedb.org/settings/api" target="_blank" style="color:var(--accent)">themoviedb.org</a>.</div>
            <input type="text" class="setting-input" id="tmdb-key" placeholder="Enter your TMDB API key">
            <div class="setting-saved" id="tmdb-saved">Saved!</div>
          </div>
          <div class="setting-group">
            <div class="setting-label">Proxy</div>
            <div class="setting-desc">Route network traffic through a proxy server.</div>
            <select id="proxy-type" class="setting-input" style="margin-bottom:8px;">
              <option value="none">No Proxy</option>
              <option value="http">HTTP Proxy</option>
              <option value="socks5">SOCKS5 Proxy</option>
            </select>
            <input type="text" class="setting-input" id="proxy-addr" placeholder="e.g. http://127.0.0.1:8080" style="display:none;">
            <div class="setting-saved" id="proxy-saved">Saved!</div>
          </div>
          ${_isLinux ? `
          <div class="setting-group">
            <div class="setting-label">Video Format Filter</div>
            <div class="setting-desc">Linux cannot reliably play HEVC/x265/10-bit video. When enabled, these formats are hidden from torrent listings. Disable to show all formats (for downloading or if you have working codec support).</div>
            <label style="display:flex;align-items:center;gap:10px;margin-top:8px;cursor:pointer;">
              <input type="checkbox" id="format-filter-toggle" ${localStorage.getItem('yaria_show_all_formats') === '1' ? '' : 'checked'} style="width:18px;height:18px;accent-color:var(--accent);cursor:pointer;">
              <span style="font-size:13px;color:var(--text-dim);">Hide unplayable formats (HEVC, x265, 10-bit)</span>
            </label>
          </div>
          ` : ''}
        </div>

        <!-- Downloader -->
        <div class="stg-panel" data-panel="downloader">
          <h3 class="stg-panel-title">Yaria Downloader</h3>
          <div class="setting-group">
            <div class="setting-label">Download Speed Limit</div>
            <div class="setting-desc">Limit bandwidth for video and audio downloads.</div>
            <select id="speed-limit" class="setting-input">
              <option value="0">Unlimited</option>
              <option value="524288">512 KB/s</option>
              <option value="1048576">1 MB/s</option>
              <option value="2097152">2 MB/s</option>
              <option value="5242880">5 MB/s</option>
              <option value="10485760">10 MB/s</option>
            </select>
            <div class="setting-saved" id="speed-saved">Saved!</div>
          </div>
          <div class="setting-group">
            <div class="setting-label">Concurrent Downloads</div>
            <div class="setting-desc">Maximum number of simultaneous downloads.</div>
            <select id="max-concurrent" class="setting-input">
              <option value="1">1</option>
              <option value="2">2</option>
              <option value="3" selected>3</option>
              <option value="5">5</option>
              <option value="10">10</option>
            </select>
            <div class="setting-saved" id="concurrent-saved">Saved!</div>
          </div>
        </div>

        <!-- Torrents -->
        ${_settingsPro ? `
        <div class="stg-panel" data-panel="torrents">
          <h3 class="stg-panel-title">Mantorex / Torrents</h3>
          <div class="setting-group" id="codec-section">
            <div class="setting-label">Video Codecs</div>
            <div class="setting-desc">Codecs required for playing streams in the built-in player.</div>
            <div id="codec-status" style="margin-top:10px;">
              <div class="spinner" style="width:18px;height:18px;border-width:2px;display:inline-block;vertical-align:middle;"></div>
              <span style="color:var(--text-muted);font-size:13px;margin-left:8px;">Checking...</span>
            </div>
          </div>
          <div class="setting-group">
            <div class="setting-label">Torrent Cache</div>
            <div class="setting-desc">Remove cached torrent data, partial files, and metadata.</div>
            <div id="cache-stats" style="margin-bottom:10px;">
              <div class="spinner" style="width:18px;height:18px;border-width:2px;display:inline-block;vertical-align:middle;"></div>
              <span style="color:var(--text-muted);font-size:13px;margin-left:8px;">Scanning...</span>
            </div>
            <div id="cache-actions" style="display:none;">
              <div style="display:flex;gap:8px;flex-wrap:wrap;">
                <button class="btn btn-ghost btn-sm" id="clear-partial">Partial</button>
                <button class="btn btn-ghost btn-sm" id="clear-meta">Metadata</button>
                <button class="btn btn-ghost btn-sm" id="clear-dirs">Data</button>
                <button class="btn btn-primary btn-sm" id="clear-all" style="background:var(--red);box-shadow:none;">Clear All</button>
              </div>
            </div>
            <div id="cache-result" style="display:none;margin-top:8px;font-size:13px;color:var(--green);"></div>
          </div>
        </div>

        <!-- Local Media -->
        <div class="stg-panel" data-panel="media">
          <h3 class="stg-panel-title">Local Media</h3>
          <div class="setting-group" id="media-folders-section">
            <div class="setting-label">Media Folders</div>
            <div class="setting-desc">Directories scanned for your media library with TMDB metadata.</div>
            <div id="media-folders-list" style="margin-top:10px;">
              <div class="spinner" style="width:18px;height:18px;border-width:2px;display:inline-block;vertical-align:middle;"></div>
              <span style="color:var(--text-muted);font-size:13px;margin-left:8px;">Loading...</span>
            </div>
          </div>
          <div class="setting-group" id="hwaccel-section">
            <div class="setting-label">Hardware Acceleration</div>
            <div class="setting-desc">Detected hardware video encoders for transcoding.</div>
            <div id="hwaccel-status" style="margin-top:10px;">
              <div class="spinner" style="width:18px;height:18px;border-width:2px;display:inline-block;vertical-align:middle;"></div>
              <span style="color:var(--text-muted);font-size:13px;margin-left:8px;">Detecting...</span>
            </div>
          </div>
        </div>

        <!-- Media Server -->
        <div class="stg-panel" data-panel="server">
          <h3 class="stg-panel-title">Media Server</h3>
          <div class="setting-group" id="media-server-section">
            <div class="setting-label">LAN Server</div>
            <div class="setting-desc">Stream media to phones, tablets, and browsers on your network.</div>
            <div id="media-server-status" style="margin-top:10px;">
              <div class="spinner" style="width:18px;height:18px;border-width:2px;display:inline-block;vertical-align:middle;"></div>
              <span style="color:var(--text-muted);font-size:13px;margin-left:8px;">Loading...</span>
            </div>
          </div>
          <div class="setting-group" id="dlna-section">
            <div class="setting-label">DLNA / UPnP</div>
            <div class="setting-desc">Advertise to smart TVs, consoles, and DLNA devices.</div>
            <div id="dlna-status" style="margin-top:10px;">
              <div class="spinner" style="width:18px;height:18px;border-width:2px;display:inline-block;vertical-align:middle;"></div>
              <span style="color:var(--text-muted);font-size:13px;margin-left:8px;">Loading...</span>
            </div>
          </div>
        </div>
        ` : ''}

        <!-- About -->
        <div class="stg-panel" data-panel="about">
          <h3 class="stg-panel-title">About & Legal</h3>
          <div class="setting-group">
            <div class="setting-label">Yaria</div>
            <div class="setting-desc" style="margin-top:6px;" id="about-version">
              Loading version...
            </div>
            <div id="update-section" style="margin-top:14px;"></div>
            <div style="display:flex;gap:8px;margin-top:14px;flex-wrap:wrap;">
              <button class="btn btn-ghost btn-sm" id="show-privacy">Privacy Policy</button>
              <button class="btn btn-ghost btn-sm" id="show-terms">Terms of Use</button>
            </div>
          </div>
        </div>
      </div>
    </div>
  `;

  container.appendChild(page);

  // --- Sidebar navigation ---
  page.querySelectorAll('.stg-nav-item').forEach(btn => {
    btn.addEventListener('click', () => {
      page.querySelectorAll('.stg-nav-item').forEach(b => b.classList.remove('active'));
      btn.classList.add('active');
      page.querySelectorAll('.stg-panel').forEach(p => p.classList.remove('active'));
      const panel = page.querySelector(`.stg-panel[data-panel="${btn.dataset.section}"]`);
      if (panel) panel.classList.add('active');
    });
  });

  // --- License section ---
  loadLicenseSection(page);

  // --- Codec section ---
  loadCodecSection(page);

  // --- TMDB key ---
  // Load saved TMDB key
  const tmdbInput = page.querySelector('#tmdb-key');
  try {
    const tmdbInfo = await API.getTMDBKey();
    if (tmdbInfo && tmdbInfo.configured) {
      tmdbInput.value = tmdbInfo.key;
      tmdbInput.placeholder = 'API key saved';
    }
  } catch(e) {}

  let saveTimeout;
  tmdbInput.addEventListener('input', (e) => {
    clearTimeout(saveTimeout);
    saveTimeout = setTimeout(async () => {
      const key = e.target.value.trim();
      try {
        await API.saveTMDBKey(key);
        const saved = page.querySelector('#tmdb-saved');
        saved.textContent = 'Saved!';
        saved.style.color = 'var(--green)';
        saved.style.display = 'block';
        setTimeout(() => { saved.style.display = 'none'; }, 3000);
      } catch (err) {
        const saved = page.querySelector('#tmdb-saved');
        saved.textContent = 'Failed to save: ' + err.message;
        saved.style.color = 'var(--red)';
        saved.style.display = 'block';
      }
    }, 800);
  });

  // --- Proxy settings ---
  const proxyTypeEl = page.querySelector('#proxy-type');
  const proxyAddrEl = page.querySelector('#proxy-addr');
  const proxySaved = page.querySelector('#proxy-saved');

  // Show/hide address input based on proxy type
  proxyTypeEl.addEventListener('change', async () => {
    const type = proxyTypeEl.value;
    proxyAddrEl.style.display = (type === 'none') ? 'none' : 'block';
    try {
      await API.saveProxy(type, proxyAddrEl.value);
      proxySaved.textContent = 'Saved!';
      proxySaved.style.color = 'var(--green)';
      proxySaved.style.display = 'block';
      setTimeout(() => { proxySaved.style.display = 'none'; }, 3000);
    } catch(e) {}
  });

  let proxyAddrTimeout;
  proxyAddrEl.addEventListener('input', () => {
    clearTimeout(proxyAddrTimeout);
    proxyAddrTimeout = setTimeout(async () => {
      try {
        await API.saveProxy(proxyTypeEl.value, proxyAddrEl.value.trim());
        proxySaved.textContent = 'Saved!';
        proxySaved.style.color = 'var(--green)';
        proxySaved.style.display = 'block';
        setTimeout(() => { proxySaved.style.display = 'none'; }, 3000);
      } catch(e) {}
    }, 800);
  });

  // Load saved proxy settings
  try {
    const proxy = await API.getProxy();
    if (proxy.type) proxyTypeEl.value = proxy.type;
    if (proxy.addr) proxyAddrEl.value = proxy.addr;
    if (proxy.type && proxy.type !== 'none') proxyAddrEl.style.display = 'block';
  } catch(e) {}

  // --- Format filter toggle (Linux only) ---
  const formatToggle = page.querySelector('#format-filter-toggle');
  if (formatToggle) {
    formatToggle.addEventListener('change', () => {
      // checked = filter ON (hide unsupported) = remove the key
      // unchecked = filter OFF (show all) = set key to '1'
      if (formatToggle.checked) {
        localStorage.removeItem('yaria_show_all_formats');
      } else {
        localStorage.setItem('yaria_show_all_formats', '1');
      }
    });
  }

  // --- Speed limit ---
  const speedLimitEl = page.querySelector('#speed-limit');
  const speedSaved = page.querySelector('#speed-saved');
  speedLimitEl.addEventListener('change', async () => {
    try {
      await API.saveSpeedLimit(parseInt(speedLimitEl.value, 10));
      speedSaved.textContent = 'Saved!';
      speedSaved.style.color = 'var(--green)';
      speedSaved.style.display = 'block';
      setTimeout(() => { speedSaved.style.display = 'none'; }, 3000);
    } catch(e) {}
  });

  // Load saved speed limit
  try {
    const limit = await API.getSpeedLimit();
    if (limit) speedLimitEl.value = String(limit);
  } catch(e) {}

  // --- Concurrent downloads ---
  const maxConcurrentEl = page.querySelector('#max-concurrent');
  const concurrentSaved = page.querySelector('#concurrent-saved');
  maxConcurrentEl.addEventListener('change', async () => {
    try {
      await API.setMaxConcurrent(parseInt(maxConcurrentEl.value, 10));
      concurrentSaved.textContent = 'Saved!';
      concurrentSaved.style.color = 'var(--green)';
      concurrentSaved.style.display = 'block';
      setTimeout(() => { concurrentSaved.style.display = 'none'; }, 3000);
    } catch(e) {}
  });

  // Load saved max concurrent
  try {
    const maxC = await API.getMaxConcurrent();
    if (maxC) maxConcurrentEl.value = String(maxC);
  } catch(e) {}

  // --- Back ---
  page.querySelector('#settings-back').addEventListener('click', () => window.history.back());

  // --- Media Folders ---
  loadMediaFolders(page);

  // --- Media Server ---
  loadMediaServerSection(page);

  // --- DLNA ---
  loadDLNASection(page);

  // --- HW Accel ---
  loadHWAccelSection(page);

  // --- Version label in sidebar ---
  API.getAppVersion().then(v => {
    const el = page.querySelector('#stg-version-label');
    if (el) el.textContent = 'Yaria v' + v;
  }).catch(() => {});

  // --- Version + Update ---
  loadUpdateSection(page);

  // --- Privacy Policy ---
  page.querySelector('#show-privacy').addEventListener('click', () => showLegalModal('Privacy Policy', `
<p>Last updated: May 2026</p>

<h3>Overview</h3>
<p>Yaria ("the App") is a desktop application. We respect your privacy and are committed to transparency about how the App operates.</p>

<h3>Data We Collect</h3>
<p><strong>We do not collect, transmit, or store any personal data on our servers.</strong> All user data stays on your device.</p>
<ul>
  <li><strong>License keys</strong> are validated against our server (yaria.live) to verify your subscription. The server receives your license key and a device fingerprint (a hash derived from your machine ID). No personal information is sent.</li>
  <li><strong>TMDB API key</strong> (if provided) is stored locally and used to fetch movie/TV metadata directly from The Movie Database API. We do not proxy or log these requests.</li>
  <li><strong>Search queries</strong> in Mantorex are sent directly to third-party torrent indexing websites. We do not log, proxy, or store your searches.</li>
  <li><strong>Download history, library data, watch progress, and settings</strong> are stored locally on your device in a BoltDB database under <code>~/.yaria/</code> and <code>~/.config/</code>.</li>
  <li><strong>Cookies</strong> extracted for video downloads are cached locally at <code>~/.yaria/cookies.txt</code> and are never transmitted to us.</li>
</ul>

<h3>Third-Party Services</h3>
<ul>
  <li><strong>yaria.live</strong> -- License validation only. Receives license key + device ID hash.</li>
  <li><strong>TMDB (The Movie Database)</strong> -- Metadata queries using your own API key.</li>
  <li><strong>Torrent indexing sites</strong> -- Search queries are sent directly from your device to third-party sites. We have no control over their privacy practices.</li>
  <li><strong>GitHub</strong> -- Dependency updates (yt-dlp, FFmpeg) are fetched from GitHub releases.</li>
</ul>

<h3>Data Storage</h3>
<p>All data is stored locally on your machine. You can delete all app data by removing the <code>~/.yaria/</code> and <code>~/.config/yaria/</code> directories.</p>

<h3>Analytics & Telemetry</h3>
<p>The App does not include any analytics, telemetry, crash reporting, or tracking of any kind.</p>

<h3>Changes</h3>
<p>We may update this policy. Changes will be included in future versions of the App.</p>

<h3>Contact</h3>
<p>For questions about this policy, visit <a href="https://yaria.live" target="_blank" style="color:var(--accent);">yaria.live</a>.</p>
  `));

  // --- Terms of Use ---
  page.querySelector('#show-terms').addEventListener('click', () => showLegalModal('Terms of Use', `
<p>Last updated: May 2026</p>

<h3>Acceptance</h3>
<p>By downloading, installing, or using Yaria ("the App"), you agree to these Terms of Use. If you do not agree, do not use the App.</p>

<h3>Description of Service</h3>
<p>Yaria is a video and audio downloader that wraps yt-dlp. The Pro version ("Yaria Pro") includes Mantorex, a torrent search engine and BitTorrent client.</p>

<h3>Lawful Use</h3>
<ul>
  <li>You agree to use the App <strong>only for lawful purposes</strong> and in compliance with all applicable local, national, and international laws.</li>
  <li>The App is a tool. <strong>You are solely responsible</strong> for the content you download, stream, or share using the App.</li>
  <li>Downloading or distributing copyrighted material without authorization from the copyright holder is illegal in most jurisdictions. <strong>We do not condone or encourage copyright infringement.</strong></li>
</ul>

<h3>Mantorex (Torrent Features)</h3>
<ul>
  <li>Mantorex searches third-party torrent indexing websites. We do not host, index, or control any torrent content.</li>
  <li>Search results are fetched directly from third-party sites. We make no warranties about the accuracy, legality, or safety of listed content.</li>
  <li>By using Mantorex, you accept the Legal Disclaimer presented on first use.</li>
  <li>BitTorrent is a peer-to-peer protocol. When downloading or streaming, your IP address is visible to other peers in the swarm.</li>
</ul>

<h3>Intellectual Property</h3>
<p>Yaria, the Yaria logo, and Mantorex are property of their respective owners. Third-party tools (yt-dlp, FFmpeg, aria2) are used under their respective open-source licenses.</p>

<h3>Licensing</h3>
<ul>
  <li>The free version of Yaria is available for personal use.</li>
  <li>Yaria Pro requires a valid license key purchased from yaria.live.</li>
  <li>License keys are bound to a single device and are non-transferable.</li>
  <li>We reserve the right to revoke licenses that violate these terms.</li>
</ul>

<h3>Disclaimer of Warranties</h3>
<p>The App is provided <strong>"as is"</strong> without warranty of any kind. We do not guarantee uninterrupted or error-free operation. Use at your own risk.</p>

<h3>Limitation of Liability</h3>
<p>To the maximum extent permitted by law, the developers shall not be liable for any damages arising from the use of this App, including but not limited to damages from downloaded content, legal actions, data loss, or service interruptions.</p>

<h3>Changes</h3>
<p>We reserve the right to modify these terms. Continued use after changes constitutes acceptance.</p>

<h3>Contact</h3>
<p>For questions about these terms, visit <a href="https://yaria.live" target="_blank" style="color:var(--accent);">yaria.live</a>.</p>
  `));

  // --- Cache stats (pro only) ---
  if (_settingsPro) {
    try {
      const stats = await API.getCacheStats();
      const statsDiv = page.querySelector('#cache-stats');
      if (statsDiv) {
        statsDiv.innerHTML = `
          <div style="font-size:13px;color:var(--text-dim);line-height:1.8;">
            <div>Partial downloads: <span style="color:var(--text)">${stats.partial_files} files</span> (${stats.partial_size})</div>
            <div>Metadata files: <span style="color:var(--text)">${stats.meta_files} files</span> (${stats.meta_size})</div>
            <div>Data folders: <span style="color:var(--text)">${stats.data_dirs} folders</span> (${stats.dir_size})</div>
            <div style="margin-top:6px;font-weight:600;color:var(--text)">Total: ${stats.total_size}</div>
            <div style="font-size:12px;color:var(--text-muted);margin-top:4px">${stats.data_dir}</div>
          </div>
        `;
        const actionsEl = page.querySelector('#cache-actions');
        if (actionsEl) actionsEl.style.display = 'block';
      }
    } catch (err) {
      const statsDiv = page.querySelector('#cache-stats');
      if (statsDiv) statsDiv.innerHTML = '<span style="color:var(--text-muted);font-size:13px;">Could not load cache info</span>';
    }

    // --- Clear buttons ---
    const clearBtn = (id, type) => {
      const btn = page.querySelector(id);
      if (!btn) return;
      btn.addEventListener('click', async () => {
        const resultDiv = page.querySelector('#cache-result');
        if (type === 'all') {
          const proceed = await new Promise(resolve => {
            appConfirm('This will delete ALL cached data. Continue?', () => resolve(true), () => resolve(false));
          });
          if (!proceed) return;
        }
        try {
          const result = await API.clearCache(type);
          if (resultDiv) {
            resultDiv.style.display = 'block';
            resultDiv.style.color = 'var(--green)';
            resultDiv.textContent = `Cleared ${result.removed} items, freed ${result.freed}`;
          }
          setTimeout(() => renderSettings(container), 2000);
        } catch (err) {
          if (resultDiv) {
            resultDiv.style.display = 'block';
            resultDiv.style.color = 'var(--red)';
            resultDiv.textContent = `Failed: ${err.message}`;
          }
        }
      });
    };
    clearBtn('#clear-partial', 'partial');
    clearBtn('#clear-meta', 'meta');
    clearBtn('#clear-dirs', 'dirs');
    clearBtn('#clear-all', 'all');
  }
}

async function loadLicenseSection(page) {
  const statusDiv = page.querySelector('#license-status');
  try {
    const info = await API.checkLicense();
    const device = await API.getDeviceInfo();

    if (info.valid && info.plan === 'pro') {
      // Active pro license
      statusDiv.innerHTML = `
        <div style="font-size:13px;line-height:1.8;">
          <div style="display:flex;align-items:center;gap:8px;margin-bottom:8px;">
            <span style="background:var(--green);color:#000;font-size:11px;font-weight:700;padding:3px 10px;border-radius:99px;">PRO</span>
            <span style="color:var(--text);">Active</span>
          </div>
          <div style="color:var(--text-dim);">Email: <span style="color:var(--text)">${esc(info.email || '-')}</span></div>
          <div style="color:var(--text-dim);">Key: <span style="color:var(--text)">${esc(info.key ? info.key.substring(0, 8) + '...' : '-')}</span></div>
          <div style="color:var(--text-dim);">Device: <span style="color:var(--text)">${esc(device.device_name)}</span></div>
          <div style="color:var(--text-muted);font-size:11px;margin-top:4px;">ID: ${esc(device.device_id)}</div>
        </div>
        <button class="btn btn-ghost btn-sm" id="deactivate-btn" style="margin-top:12px;color:var(--red);">Deactivate License</button>
      `;
      page.querySelector('#deactivate-btn').addEventListener('click', async () => {
        const proceed = await new Promise(resolve => {
          appConfirm('Deactivate your Pro license on this device?', () => resolve(true), () => resolve(false));
        });
        if (!proceed) return;
        await API.deactivate();
        resetProCache();
        renderSettings(page.closest('#app') || page.parentElement);
      });
    } else {
      // No license or free
      statusDiv.innerHTML = `
        <div style="font-size:13px;line-height:1.8;margin-bottom:12px;">
          <div style="display:flex;align-items:center;gap:8px;margin-bottom:8px;">
            <span style="background:var(--text-muted);color:#000;font-size:11px;font-weight:700;padding:3px 10px;border-radius:99px;">FREE</span>
            <span style="color:var(--text-dim);">Mantorex features require Pro</span>
          </div>
          <div style="color:var(--text-muted);font-size:12px;">Device: ${esc(device.device_name)} (${esc(device.device_id)})</div>
        </div>
        <div style="display:flex;gap:8px;align-items:center;">
          <input type="text" id="license-key-input" placeholder="Enter license key" style="flex:1;padding:10px 14px;background:rgba(255,255,255,0.04);border:1px solid rgba(255,255,255,0.08);border-radius:var(--radius-sm);color:var(--text);font-size:13px;outline:none;">
          <button class="btn btn-primary btn-sm" id="activate-btn">Activate</button>
        </div>
        <div id="license-error" style="display:none;margin-top:8px;font-size:13px;color:var(--red);"></div>
        <div id="license-success" style="display:none;margin-top:8px;font-size:13px;color:var(--green);"></div>
        <p style="color:var(--text-muted);font-size:12px;margin-top:12px;">
          Get a license at <a href="https://yaria.live" target="_blank" style="color:var(--accent);">yaria.live</a>
        </p>
      `;

      const btn = page.querySelector('#activate-btn');
      const input = page.querySelector('#license-key-input');
      const errEl = page.querySelector('#license-error');
      const okEl = page.querySelector('#license-success');

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
            okEl.textContent = 'Pro activated! Mantorex is now available.'; okEl.style.display = 'block';
            resetProCache();
            setTimeout(() => renderSettings(page.closest('#app') || page.parentElement), 1500);
          } else {
            errEl.textContent = 'Invalid license key'; errEl.style.display = 'block';
          }
        } catch (e) {
          errEl.textContent = e.message || 'Activation failed'; errEl.style.display = 'block';
        }
        btn.disabled = false; btn.textContent = 'Activate';
      });
      input.addEventListener('keydown', (e) => { if (e.key === 'Enter') btn.click(); });
    }
  } catch (e) {
    statusDiv.innerHTML = '<span style="color:var(--text-muted);font-size:13px;">Could not check license status</span>';
  }
}

async function loadMediaServerSection(page) {
  const el = page.querySelector('#media-server-status');
  if (!el) return;

  try {
    const status = await API.mediaServerStatus();
    if (status.running) {
      el.innerHTML = `
        <div style="font-size:13px;line-height:2;">
          <div style="display:flex;align-items:center;gap:8px;margin-bottom:8px;">
            <span style="background:var(--green);color:#000;font-size:11px;font-weight:700;padding:3px 10px;border-radius:99px;">RUNNING</span>
            <span style="color:var(--text);">Port ${status.port}</span>
          </div>
          <div style="color:var(--text-dim);">URL: <a href="${esc(status.url)}" target="_blank" style="color:var(--accent);">${esc(status.url)}</a></div>
          ${status.pin ? '<div style="color:var(--text-muted);font-size:12px;">PIN protected</div>' : '<div style="color:var(--yellow);font-size:12px;">No PIN set (open access)</div>'}
        </div>
        <div style="display:flex;gap:8px;margin-top:12px;">
          <button class="btn btn-ghost btn-sm" id="ms-stop" style="color:var(--red);">Stop Server</button>
        </div>
      `;
      el.querySelector('#ms-stop').addEventListener('click', async () => {
        await API.stopMediaServer();
        loadMediaServerSection(page);
      });
    } else {
      el.innerHTML = `
        <div style="font-size:13px;line-height:2.2;">
          <div style="display:flex;gap:8px;margin-bottom:8px;align-items:center;">
            <div style="flex:1;">
              <label style="font-size:12px;color:var(--text-dim);">Port</label>
              <input type="number" id="ms-port" class="setting-input" value="${status.port || 8096}" style="width:100px;padding:6px 10px;font-size:13px;">
            </div>
            <div style="flex:1;">
              <label style="font-size:12px;color:var(--text-dim);">PIN (optional)</label>
              <input type="text" id="ms-pin" class="setting-input" placeholder="No PIN" maxlength="8" style="padding:6px 10px;font-size:13px;">
            </div>
          </div>
        </div>
        <button class="btn btn-primary btn-sm" id="ms-start">Start Server</button>
        <span id="ms-error" style="display:none;color:var(--red);font-size:13px;margin-left:10px;"></span>
      `;
      el.querySelector('#ms-start').addEventListener('click', async () => {
        const port = parseInt(el.querySelector('#ms-port').value) || 8096;
        const pin = el.querySelector('#ms-pin').value.trim();
        const errEl = el.querySelector('#ms-error');
        await API.setMediaServerPort(port);
        if (pin) await API.setMediaServerPin(pin);
        const result = await API.startMediaServer();
        if (result.error) {
          errEl.textContent = result.error;
          errEl.style.display = 'inline';
        } else {
          loadMediaServerSection(page);
        }
      });
    }
  } catch(e) {
    el.innerHTML = '<span style="color:var(--text-muted);font-size:13px;">Media server not available</span>';
  }
}

async function loadDLNASection(page) {
  const el = page.querySelector('#dlna-status');
  if (!el) return;
  try {
    const status = await API.dlnaStatus();
    if (status.running) {
      el.innerHTML = `
        <div style="font-size:13px;line-height:2;">
          <div style="display:flex;align-items:center;gap:8px;margin-bottom:4px;">
            <span style="background:var(--green);color:#000;font-size:11px;font-weight:700;padding:3px 10px;border-radius:99px;">ACTIVE</span>
            <span style="color:var(--text);">Port ${status.port}</span>
          </div>
          <div style="color:var(--text-muted);font-size:12px;">Devices on your network can now discover and browse your media.</div>
        </div>
        <button class="btn btn-ghost btn-sm" id="dlna-stop" style="margin-top:8px;color:var(--red);">Stop DLNA</button>
      `;
      el.querySelector('#dlna-stop').addEventListener('click', async () => {
        await API.stopDLNA();
        loadDLNASection(page);
      });
    } else {
      el.innerHTML = `
        <button class="btn btn-primary btn-sm" id="dlna-start">Start DLNA Server</button>
        <span id="dlna-error" style="display:none;color:var(--red);font-size:13px;margin-left:10px;"></span>
      `;
      el.querySelector('#dlna-start').addEventListener('click', async () => {
        const result = await API.startDLNA();
        if (result.error) {
          const errEl = el.querySelector('#dlna-error');
          errEl.textContent = result.error;
          errEl.style.display = 'inline';
        } else {
          loadDLNASection(page);
        }
      });
    }
  } catch(e) {
    el.innerHTML = '<span style="color:var(--text-muted);font-size:13px;">DLNA not available</span>';
  }
}

async function loadHWAccelSection(page) {
  const el = page.querySelector('#hwaccel-status');
  if (!el) return;
  try {
    const info = await API.detectHWAccel();
    if (!info.available) {
      el.innerHTML = '<span style="color:var(--text-muted);font-size:13px;">FFmpeg not found. Install FFmpeg to enable transcoding.</span>';
      return;
    }
    const encoders = info.encoders || [];
    let html = '<div style="font-size:13px;line-height:2;">';
    encoders.forEach(enc => {
      const icon = enc.available ? '<span style="color:var(--green);">Available</span>' : '<span style="color:var(--text-muted);">Not found</span>';
      html += `<div style="display:flex;justify-content:space-between;align-items:center;">
        <span style="color:var(--text-dim);">${esc(enc.name)}</span>
        <span>${icon}</span>
      </div>`;
    });
    html += '</div>';
    el.innerHTML = html;
  } catch(e) {
    el.innerHTML = '<span style="color:var(--text-muted);font-size:13px;">Could not detect hardware acceleration</span>';
  }
}

async function loadUpdateSection(page) {
  const versionEl = page.querySelector('#about-version');
  const updateEl = page.querySelector('#update-section');
  if (!versionEl || !updateEl) return;

  try {
    const version = await API.getAppVersion();
    versionEl.innerHTML = `Yaria Desktop App v${esc(version)}<br>Video & audio downloader + media center.`;
  } catch(e) {
    versionEl.innerHTML = 'Yaria Desktop App<br>Video & audio downloader + media center.';
  }

  updateEl.innerHTML = `<button class="btn btn-ghost btn-sm" id="check-update-btn">Check for Updates</button>
    <span id="update-status" style="font-size:13px;color:var(--text-muted);margin-left:10px;"></span>`;

  updateEl.querySelector('#check-update-btn').addEventListener('click', async () => {
    const btn = updateEl.querySelector('#check-update-btn');
    const statusEl = updateEl.querySelector('#update-status');
    btn.disabled = true;
    btn.textContent = 'Checking...';
    statusEl.textContent = '';

    try {
      const info = await API.checkUpdate();
      if (info.error) {
        statusEl.style.color = 'var(--red)';
        statusEl.textContent = info.error;
        btn.textContent = 'Check for Updates';
        btn.disabled = false;
        return;
      }

      if (info.available) {
        updateEl.innerHTML = `
          <div style="font-size:13px;margin-bottom:12px;">
            <span style="color:var(--green);font-weight:600;">Update available!</span>
            <span style="color:var(--text-muted);">v${esc(info.current)} → v${esc(info.latest)}</span>
          </div>
          <button class="btn btn-primary btn-sm" id="install-update-btn">Update Now</button>
          <div id="update-progress" style="display:none;margin-top:12px;">
            <div class="download-progress-bar" style="margin-bottom:6px;">
              <div class="download-progress-fill" id="update-bar" style="width:0%"></div>
            </div>
            <p id="update-msg" style="font-size:12px;color:var(--text-muted);"></p>
          </div>
        `;

        updateEl.querySelector('#install-update-btn').addEventListener('click', async () => {
          const installBtn = updateEl.querySelector('#install-update-btn');
          const progDiv = updateEl.querySelector('#update-progress');
          installBtn.disabled = true;
          installBtn.textContent = 'Updating...';
          progDiv.style.display = 'block';

          await API.startUpdate();

          const poll = setInterval(async () => {
            try {
              const s = await API.updateStatus();
              const bar = updateEl.querySelector('#update-bar');
              const msg = updateEl.querySelector('#update-msg');
              if (bar) bar.style.width = (s.progress || 0) + '%';
              if (msg) msg.textContent = s.message || '';

              if (!s.updating) {
                clearInterval(poll);
                installBtn.style.display = 'none';
                if (s.progress >= 100) {
                  if (msg) {
                    msg.style.color = 'var(--green)';
                    msg.innerHTML = s.message + '<br><button class="btn btn-primary btn-sm" style="margin-top:10px;" onclick="location.reload()">Restart App</button>';
                  }
                } else if (s.message) {
                  if (msg) msg.style.color = 'var(--red)';
                }
              }
            } catch(e) {
              clearInterval(poll);
            }
          }, 800);
        });

      } else {
        statusEl.style.color = 'var(--green)';
        statusEl.textContent = 'You\'re on the latest version (v' + info.current + ')';
        btn.textContent = 'Check for Updates';
        btn.disabled = false;
      }
    } catch(e) {
      statusEl.style.color = 'var(--red)';
      statusEl.textContent = 'Check failed: ' + (e.message || '');
      btn.textContent = 'Check for Updates';
      btn.disabled = false;
    }
  });
}

async function loadMediaFolders(page) {
  const listEl = page.querySelector('#media-folders-list');
  if (!listEl) return;

  try {
    const folders = await API.getMediaFolders();
    if (folders.error) {
      listEl.innerHTML = '<span style="color:var(--text-muted);font-size:13px;">Media library not available in this build</span>';
      return;
    }

    const movieDirs = folders.movie_dirs || [];
    const tvDirs = folders.tv_dirs || [];
    const videoDirs = folders.video_dirs || [];

    let html = '<div style="font-size:13px;line-height:2.2;">';

    const renderDirList = (dirs, label, type) => {
      html += `<div style="color:var(--text-dim);font-weight:600;margin-top:8px;">${label}</div>`;
      if (dirs.length === 0) {
        html += '<div style="color:var(--text-muted);font-style:italic;">No folders configured</div>';
      }
      dirs.forEach(dir => {
        html += `<div style="display:flex;align-items:center;justify-content:space-between;gap:8px;">
          <span style="color:var(--text);font-size:12px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;" title="${esc(dir)}">${esc(dir)}</span>
          <button class="btn btn-ghost btn-sm media-remove-dir" data-path="${esc(dir)}" data-type="${type}" style="color:var(--red);flex-shrink:0;font-size:11px;">Remove</button>
        </div>`;
      });
      html += `<button class="btn btn-ghost btn-sm media-add-dir" data-type="${type}" style="margin-top:4px;">+ Add Folder</button>`;
    };

    renderDirList(movieDirs, 'Movies', 'movie');
    renderDirList(tvDirs, 'TV Shows', 'tv');
    renderDirList(videoDirs, 'Other Videos', 'video');

    html += '</div>';
    html += '<button class="btn btn-primary btn-sm" id="media-scan-now" style="margin-top:12px;">Scan Now</button>';
    html += '<span id="media-scan-msg" style="font-size:12px;color:var(--text-muted);margin-left:10px;"></span>';

    listEl.innerHTML = html;

    // Add folder buttons -- use in-app file picker
    listEl.querySelectorAll('.media-add-dir').forEach(btn => {
      btn.addEventListener('click', () => {
        if (typeof showDownloadPicker === 'function') {
          showDownloadPicker(async (dir) => {
            if (dir) {
              await API.addMediaFolder(dir, btn.dataset.type);
              loadMediaFolders(page);
            }
          });
        }
      });
    });

    // Remove folder buttons
    listEl.querySelectorAll('.media-remove-dir').forEach(btn => {
      btn.addEventListener('click', async () => {
        await API.removeMediaFolder(btn.dataset.path, btn.dataset.type);
        loadMediaFolders(page);
      });
    });

    // Scan button
    const scanBtn = listEl.querySelector('#media-scan-now');
    if (scanBtn) {
      scanBtn.addEventListener('click', async () => {
        scanBtn.disabled = true;
        scanBtn.textContent = 'Scanning...';
        const msgEl = listEl.querySelector('#media-scan-msg');
        await API.scanMedia();
        const poll = setInterval(async () => {
          const s = await API.scanMediaStatus();
          if (msgEl) msgEl.textContent = s.message || '';
          if (!s.scanning) {
            clearInterval(poll);
            scanBtn.disabled = false;
            scanBtn.textContent = 'Scan Now';
          }
        }, 1000);
      });
    }

  } catch (e) {
    listEl.innerHTML = '<span style="color:var(--text-muted);font-size:13px;">Could not load media folders</span>';
  }
}

function showLegalModal(title, html) {
  // Remove any existing modal
  const existing = document.querySelector('.legal-modal-overlay');
  if (existing) existing.remove();

  const overlay = document.createElement('div');
  overlay.className = 'legal-modal-overlay';
  overlay.style.cssText = 'position:fixed;inset:0;z-index:1000;background:rgba(0,0,0,0.6);backdrop-filter:blur(4px);display:flex;align-items:center;justify-content:center;animation:pageIn 0.2s ease-out;';

  const modal = document.createElement('div');
  modal.style.cssText = 'width:640px;max-width:92vw;max-height:80vh;background:#12121e;border:1px solid rgba(255,255,255,0.08);border-radius:var(--radius);display:flex;flex-direction:column;box-shadow:0 24px 80px rgba(0,0,0,0.6);';

  const header = document.createElement('div');
  header.style.cssText = 'display:flex;justify-content:space-between;align-items:center;padding:18px 24px;border-bottom:1px solid rgba(255,255,255,0.06);';
  header.innerHTML = `<h3 style="font-size:18px;font-weight:700;margin:0;">${title}</h3><button id="legal-close" style="background:none;border:none;color:var(--text-muted);font-size:22px;cursor:pointer;width:28px;height:28px;display:flex;align-items:center;justify-content:center;border-radius:50;">&times;</button>`;

  const body = document.createElement('div');
  body.style.cssText = 'flex:1;overflow-y:auto;padding:24px;font-size:13px;color:var(--text-dim);line-height:1.7;';
  body.innerHTML = html;

  // Style the inner HTML elements
  body.querySelectorAll('h3').forEach(h => h.style.cssText = 'font-size:15px;font-weight:700;color:var(--text);margin:20px 0 8px;');
  body.querySelectorAll('h3')[0].style.marginTop = '0';
  body.querySelectorAll('ul').forEach(u => u.style.cssText = 'padding-left:20px;margin:8px 0;');
  body.querySelectorAll('li').forEach(l => l.style.cssText = 'margin-bottom:6px;');
  body.querySelectorAll('code').forEach(c => c.style.cssText = 'background:rgba(255,255,255,0.06);padding:1px 6px;border-radius:3px;font-size:12px;');
  body.querySelectorAll('p').forEach(p => { if (!p.style.marginBottom) p.style.marginBottom = '10px'; });

  modal.appendChild(header);
  modal.appendChild(body);
  overlay.appendChild(modal);
  document.body.appendChild(overlay);

  const close = () => overlay.remove();
  header.querySelector('#legal-close').addEventListener('click', close);
  overlay.addEventListener('click', (e) => { if (e.target === overlay) close(); });
}

async function loadCodecSection(page) {
  const statusDiv = page.querySelector('#codec-status');
  if (!statusDiv) return;

  try {
    const info = await API.checkCodecs();
    if (!info || info.supported === false) {
      statusDiv.innerHTML = '<span style="color:var(--text-muted);font-size:13px;">Codec check not available on this platform</span>';
      return;
    }

    const codecs = info.codecs || [];
    let html = '<div style="font-size:13px;line-height:2;">';
    codecs.forEach(c => {
      const icon = c.installed ? '<span style="color:var(--green);">Installed</span>' : '<span style="color:var(--red);">Missing</span>';
      html += `<div style="display:flex;justify-content:space-between;align-items:center;">
        <span style="color:var(--text-dim);">${esc(c.name)}</span>
        <span>${icon} <span style="color:var(--text-muted);font-size:11px;margin-left:6px;">${esc(c.package)}</span></span>
      </div>`;
    });
    html += '</div>';

    if (!info.all_installed) {
      html += `<button class="btn btn-primary btn-sm" id="install-codecs-settings" style="margin-top:12px;">Install Missing Codecs</button>`;
      html += `<div id="codec-install-status" style="display:none;margin-top:8px;font-size:13px;"></div>`;
    }

    statusDiv.innerHTML = html;

    const installBtn = page.querySelector('#install-codecs-settings');
    if (installBtn) {
      installBtn.addEventListener('click', async () => {
        installBtn.disabled = true;
        installBtn.textContent = 'Installing...';
        const st = page.querySelector('#codec-install-status');
        if (st) { st.style.display = 'block'; st.style.color = 'var(--text-muted)'; st.textContent = 'A password prompt may appear...'; }
        try {
          const result = await API.installCodecs();
          if (result.error) {
            if (st) { st.style.color = 'var(--red)'; st.textContent = result.error; }
            installBtn.textContent = 'Install Missing Codecs';
            installBtn.disabled = false;
          } else {
            if (st) { st.style.color = 'var(--green)'; st.textContent = 'Installed! Restart the app for full codec support.'; }
            setTimeout(() => loadCodecSection(page), 2000);
          }
        } catch(e) {
          if (st) { st.style.color = 'var(--red)'; st.textContent = 'Failed: ' + (e.message || ''); }
          installBtn.textContent = 'Install Missing Codecs';
          installBtn.disabled = false;
        }
      });
    }
  } catch (e) {
    statusDiv.innerHTML = '<span style="color:var(--text-muted);font-size:13px;">Could not check codec status</span>';
  }
}
