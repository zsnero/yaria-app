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

  // --- TMDB key ---
  let saveTimeout;
  page.querySelector('#tmdb-key').addEventListener('input', (e) => {
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
