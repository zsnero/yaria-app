// Settings page
async function renderSettings(container) {
  container.innerHTML = '';

  const page = document.createElement('div');
  page.className = 'settings-page';

  page.innerHTML = `
    <button class="back-btn" id="settings-back">Back</button>
    <h2>Settings</h2>

    <div class="setting-group" id="license-section">
      <div class="setting-label">Yaria Pro License</div>
      <div id="license-status" style="margin-bottom:12px;">
        <div class="spinner" style="width:20px;height:20px;border-width:2px;margin-bottom:0;display:inline-block;vertical-align:middle;"></div>
        <span style="color:var(--text-muted);font-size:13px;margin-left:8px;">Checking license...</span>
      </div>
    </div>

    <div class="setting-group">
      <div class="setting-label">TMDB API Key (Optional)</div>
      <div class="setting-desc">
        Enables trending content, richer metadata, and movie descriptions.
        Get a free key at <a href="https://www.themoviedb.org/settings/api" target="_blank" style="color:var(--accent)">themoviedb.org</a>.
      </div>
      <input type="text" class="setting-input" id="tmdb-key" placeholder="Enter your TMDB API key">
      <div class="setting-saved" id="tmdb-saved">Saved!</div>
    </div>

    <div class="setting-group">
      <div class="setting-label">Clear Cache</div>
      <div class="setting-desc">Remove cached data, partial files, and metadata.</div>
      <div id="cache-stats" style="margin-bottom:12px;">
        <div class="spinner" style="width:20px;height:20px;border-width:2px;margin-bottom:0;display:inline-block;vertical-align:middle;"></div>
        <span style="color:var(--text-muted);font-size:13px;margin-left:8px;">Scanning...</span>
      </div>
      <div id="cache-actions" style="display:none;">
        <div style="display:flex;gap:8px;flex-wrap:wrap;">
          <button class="btn btn-ghost btn-sm" id="clear-partial">Partial Downloads</button>
          <button class="btn btn-ghost btn-sm" id="clear-meta">Metadata</button>
          <button class="btn btn-ghost btn-sm" id="clear-dirs">Data Folders</button>
          <button class="btn btn-primary btn-sm" id="clear-all" style="background:var(--red);box-shadow:none;">Clear Everything</button>
        </div>
      </div>
      <div id="cache-result" style="display:none;margin-top:10px;font-size:13px;color:var(--green);"></div>
    </div>

    <div class="setting-group" id="codec-section">
      <div class="setting-label">Video Codecs</div>
      <div class="setting-desc">Codecs required for playing video streams in the app.</div>
      <div id="codec-status" style="margin-top:12px;">
        <div class="spinner" style="width:20px;height:20px;border-width:2px;margin-bottom:0;display:inline-block;vertical-align:middle;"></div>
        <span style="color:var(--text-muted);font-size:13px;margin-left:8px;">Checking codecs...</span>
      </div>
    </div>

    <div class="setting-group">
      <div class="setting-label">Proxy</div>
      <div class="setting-desc">Route downloads through a proxy server.</div>
      <select id="proxy-type" class="setting-input" style="margin-bottom:8px;">
        <option value="none">No Proxy</option>
        <option value="http">HTTP Proxy</option>
        <option value="socks5">SOCKS5 Proxy</option>
      </select>
      <input type="text" class="setting-input" id="proxy-addr" placeholder="e.g. http://127.0.0.1:8080" style="display:none;">
      <div class="setting-saved" id="proxy-saved">Saved!</div>
    </div>

    <div class="setting-group">
      <div class="setting-label">Download Speed Limit</div>
      <div class="setting-desc">Limit download bandwidth. Applies to video downloads.</div>
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

    <div class="setting-group">
      <div class="setting-label">About</div>
      <div class="setting-desc" style="margin-top:8px">
        Yaria Desktop App v1.0.0<br>
        Video & audio downloader.
      </div>
    </div>
  `;

  container.appendChild(page);

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

  // --- Cache stats ---
  try {
    const stats = await API.getCacheStats();
    const statsDiv = page.querySelector('#cache-stats');
    statsDiv.innerHTML = `
      <div style="font-size:13px;color:var(--text-dim);line-height:1.8;">
        <div>Partial downloads: <span style="color:var(--text)">${stats.partial_files} files</span> (${stats.partial_size})</div>
        <div>Metadata files: <span style="color:var(--text)">${stats.meta_files} files</span> (${stats.meta_size})</div>
        <div>Data folders: <span style="color:var(--text)">${stats.data_dirs} folders</span> (${stats.dir_size})</div>
        <div style="margin-top:6px;font-weight:600;color:var(--text)">Total: ${stats.total_size}</div>
        <div style="font-size:12px;color:var(--text-muted);margin-top:4px">${stats.data_dir}</div>
      </div>
    `;
    page.querySelector('#cache-actions').style.display = 'block';
  } catch (err) {
    page.querySelector('#cache-stats').innerHTML = '<span style="color:var(--text-muted);font-size:13px;">Could not load cache info</span>';
  }

  // --- Clear buttons ---
  const clearBtn = (id, type) => {
    page.querySelector(id).addEventListener('click', async () => {
      const resultDiv = page.querySelector('#cache-result');
      if (type === 'all' && !confirm('This will delete ALL cached data. Continue?')) return;
      try {
        const result = await API.clearCache(type);
        resultDiv.style.display = 'block';
        resultDiv.style.color = 'var(--green)';
        resultDiv.textContent = `Cleared ${result.removed} items, freed ${result.freed}`;
        setTimeout(() => renderSettings(container), 2000);
      } catch (err) {
        resultDiv.style.display = 'block';
        resultDiv.style.color = 'var(--red)';
        resultDiv.textContent = `Failed: ${err.message}`;
      }
    });
  };
  clearBtn('#clear-partial', 'partial');
  clearBtn('#clear-meta', 'meta');
  clearBtn('#clear-dirs', 'dirs');
  clearBtn('#clear-all', 'all');
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
        if (!confirm('Deactivate your Pro license on this device?')) return;
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
          Get a license at <a href="https://yaria.app" target="_blank" style="color:var(--accent);">yaria.app</a>
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
