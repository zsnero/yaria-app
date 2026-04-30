// Yaria Home Page - Video & Audio Downloader
let _yariaHomeEventCleanups = [];
let _yariaHomeDepsReady = false;

async function renderYariaHome(container) {
  container.innerHTML = '';

  const page = document.createElement('div');
  page.className = 'yaria-home-page';

  // Hero
  const hero = document.createElement('div');
  hero.className = 'yaria-hero';
  hero.innerHTML = `
    <div class="yaria-logo-wrap">
      <img src="img/yaria-logo.svg" alt="Yaria" class="yaria-logo-img">
    </div>
    <p class="yaria-subtitle">Video & Audio Downloader</p>
    <!-- <p class="yaria-desc">Download from YouTube, Vimeo, Twitter, TikTok, and 1000+ sites</p> -->
  `;
  page.appendChild(hero);

  // URL input
  const urlSection = document.createElement('div');
  urlSection.className = 'download-url-section';
  urlSection.innerHTML = `
    <div class="download-input-wrap">
      <input type="text" class="download-input" id="dl-url" placeholder="Paste a video URL..." autocomplete="off">
      <button class="btn btn-primary" id="dl-fetch-btn">Fetch</button>
    </div>
    <div id="dl-deps-status" class="download-deps-status">
      <div class="spinner" style="width:18px;height:18px;border-width:2px;margin-bottom:0;display:inline-block;vertical-align:middle;"></div>
      <span style="margin-left:8px;color:var(--text-muted);font-size:13px;">Preparing download tools...</span>
    </div>
  `;
  page.appendChild(urlSection);

  // Content area: holds the current fetch/format/download flow
  const contentArea = document.createElement('div');
  contentArea.id = 'dl-content';
  page.appendChild(contentArea);

  // No active downloads section here -- use Downloads page instead

  container.appendChild(page);

  // State
  let currentURL = '';
  let currentMeta = null;
  let selectedFormat = null;
  let audioOnly = false;
  let audioFormat = 'mp3';
  let formats = { video: [], audio: [] };
  let downloadDir = '';

  const urlInput = page.querySelector('#dl-url');
  const fetchBtn = page.querySelector('#dl-fetch-btn');
  const depsStatus = page.querySelector('#dl-deps-status');

  // Init deps
  initDeps(depsStatus, urlInput, fetchBtn);
  loadDownloadDir();

  // Fetch on click/Enter
  const doFetch = () => {
    let url = urlInput.value.trim();
    if (!url) return;
    // Clean shell-escaped URLs (e.g. from terminal paste)
    url = url.replace(/\\([?=&#])/g, '$1');
    urlInput.value = url;
    currentURL = url;
    currentMeta = null;
    selectedFormat = null;
    fetchAndShow(url);
  };
  fetchBtn.addEventListener('click', doFetch);
  urlInput.addEventListener('keydown', (e) => { if (e.key === 'Enter') doFetch(); });

  // Subscribe to events
  if (window.runtime && window.runtime.EventsOn) {
    const offReady = window.runtime.EventsOn('deps-ready', () => {
      _yariaHomeDepsReady = true;
      depsStatus.style.display = 'none';
      urlInput.disabled = false;
      fetchBtn.disabled = false;
    });
    _yariaHomeEventCleanups.push(offReady);

    const offProgress = window.runtime.EventsOn('deps-progress', (data) => {
      if (data && data.message && depsStatus) {
        depsStatus.innerHTML = `
          <div class="spinner" style="width:18px;height:18px;border-width:2px;margin-bottom:0;display:inline-block;vertical-align:middle;"></div>
          <span style="margin-left:8px;color:var(--text-muted);font-size:13px;">${esc(data.message)}</span>`;
      }
    });
    _yariaHomeEventCleanups.push(offProgress);

    const offError = window.runtime.EventsOn('deps-error', (data) => {
      if (depsStatus) {
        depsStatus.innerHTML = `<span style="color:var(--red);font-size:13px;">Setup failed: ${esc(data.error||'')}</span>
          <button class="btn btn-ghost btn-sm" id="retry-deps" style="margin-left:8px;">Retry</button>`;
        const r = depsStatus.querySelector('#retry-deps');
        if (r) r.addEventListener('click', () => initDeps(depsStatus, urlInput, fetchBtn));
      }
    });
    _yariaHomeEventCleanups.push(offError);

    // Download progress -- update inline progress items
    const offDlProgress = window.runtime.EventsOn('download-progress', (data) => {
      updateActiveDownload(data);
    });
    _yariaHomeEventCleanups.push(offDlProgress);
  }

  // Downloads are managed in the Downloads page

  setTimeout(() => urlInput.focus(), 100);

  // ---- Inner functions ----

  async function initDeps(statusEl, inputEl, btnEl) {
    inputEl.disabled = true;
    btnEl.disabled = true;
    try {
      const check = await API.checkDeps();
      if (check && check.ready) {
        _yariaHomeDepsReady = true;
        statusEl.style.display = 'none';
        inputEl.disabled = false;
        btnEl.disabled = false;
        return;
      }
    } catch (e) {}
    try {
      await API.initDeps();
      _yariaHomeDepsReady = true;
      statusEl.style.display = 'none';
      inputEl.disabled = false;
      btnEl.disabled = false;
    } catch (err) {
      statusEl.innerHTML = `<span style="color:var(--red);font-size:13px;">Failed: ${esc(err.message)}</span>`;
    }
  }

  async function loadDownloadDir() {
    try { downloadDir = await API.getDownloadDir() || ''; } catch(e) {}
  }

  async function fetchAndShow(url) {
    const content = document.getElementById('dl-content');
    content.innerHTML = `<div class="dl-card">
      <div class="dl-card-loading"><div class="spinner" style="margin:0 auto 12px;"></div><p style="color:var(--text-muted);font-size:13px;text-align:center;">Fetching video info...</p></div>
    </div>`;

    try {
      const meta = await API.fetchMetadata(url);
      currentMeta = meta;

      // Build the unified card: thumbnail + info + formats + download button
      let html = `<div class="dl-card">`;

      // Top row: thumbnail + title
      html += `<div class="dl-card-header">`;
      if (meta.thumbnail) {
        html += `<img class="dl-card-thumb" src="${esc(meta.thumbnail)}" alt="" onerror="this.style.display='none'">`;
      }
      html += `<div class="dl-card-info">
        <h3 class="dl-card-title">${esc(meta.title || 'Unknown Title')}</h3>`;
      if (meta.uploader) html += `<p class="dl-card-uploader">${esc(meta.uploader)}</p>`;
      if (meta.duration) {
        const m = Math.floor(meta.duration / 60), s = meta.duration % 60;
        html += `<p class="dl-card-duration">${m}:${String(s).padStart(2, '0')}</p>`;
      }
      html += `</div></div>`;

      // Format selection inline
      html += `<div class="dl-card-formats">
        <div class="format-header">
          <span class="format-section-title" style="font-size:13px;">Quality</span>
          <div class="audio-toggle-wrap">
            <label class="audio-toggle"><input type="checkbox" id="dl-audio-toggle" ${audioOnly ? 'checked' : ''}><span class="audio-toggle-slider"></span></label>
            <span class="audio-toggle-label">Audio Only</span>
          </div>
        </div>
        <div class="format-grid" id="dl-format-grid"></div>
      </div>`;

      // Download dir + button
      html += `<div class="dl-card-actions">
        <div class="dl-card-dir">
          <input type="text" class="setting-input dir-input" id="dl-dir" value="${esc(downloadDir)}" readonly placeholder="Download directory">
          <button class="btn btn-ghost btn-sm" id="dl-browse-btn">Browse</button>
        </div>
        <button class="btn btn-primary" id="dl-start-btn" style="padding:12px 48px;font-size:14px;align-self:center;">Download</button>
      </div>`;

      html += `</div>`;
      content.innerHTML = html;

      // Render format options
      renderFormatGrid();

      // Bind audio toggle
      const audioToggle = content.querySelector('#dl-audio-toggle');
      if (audioToggle) {
        audioToggle.addEventListener('change', (e) => {
          audioOnly = e.target.checked;
          selectedFormat = null;
          renderFormatGrid();
        });
      }

      // Browse dir
      const browseBtn = content.querySelector('#dl-browse-btn');
      if (browseBtn) {
        browseBtn.addEventListener('click', async () => {
          try {
            const dir = await API.selectDownloadDir();
            if (dir) {
              downloadDir = dir;
              const inp = content.querySelector('#dl-dir');
              if (inp) inp.value = dir;
            }
          } catch(e) {}
        });
      }

      // Start download
      const startBtn = content.querySelector('#dl-start-btn');
      startBtn.addEventListener('click', () => startDownload(url, startBtn));

      // Fetch formats in background
      loadFormats(url);

    } catch (err) {
      content.innerHTML = `<div class="dl-card"><div class="no-results" style="padding:20px;">Failed: ${esc(err.message)}</div></div>`;
    }
  }

  function renderFormatGrid() {
    const grid = document.getElementById('dl-format-grid');
    if (!grid) return;
    let html = '';

    if (audioOnly) {
      const fmts = ['mp3', 'm4a', 'opus', 'wav', 'flac'];
      if (!selectedFormat) { selectedFormat = 'mp3'; audioFormat = 'mp3'; }
      fmts.forEach(f => {
        const sel = audioFormat === f ? ' selected' : '';
        html += `<div class="format-card audio-fmt-card${sel}" data-audio-fmt="${f}"><div class="format-card-label">${f.toUpperCase()}</div></div>`;
      });
    } else {
      if (!selectedFormat) selectedFormat = 'best';

      // "Best" quality option - let yt-dlp decide
      const bestSel = selectedFormat === 'best' ? ' selected' : '';
      html += `<div class="format-card${bestSel}" data-format="best">
        <div class="format-card-label">Best</div>
        <div class="format-card-ext" style="font-size:10px;color:var(--text-muted);">Auto</div>
      </div>`;

      if (formats.video.length > 0) {
        formats.video.forEach((f, i) => {
          const key = f.resolution || f.format_id || `video-${i}`;
          const sel = selectedFormat === key ? ' selected' : '';
          html += `<div class="format-card${sel}" data-format="${esc(key)}">
            <div class="format-card-label">${esc(f.resolution || f.format_note || key)}</div>
            ${f.ext ? `<div class="format-card-ext">${esc(f.ext)}</div>` : ''}
          </div>`;
        });
      } else {
        ['2160p', '1440p', '1080p', '720p', '480p', '360p'].forEach(res => {
          const sel = selectedFormat === res ? ' selected' : '';
          html += `<div class="format-card${sel}" data-format="${res}"><div class="format-card-label">${res}</div></div>`;
        });
      }
    }

    grid.innerHTML = html;

    // Bind clicks
    grid.querySelectorAll('.audio-fmt-card').forEach(card => {
      card.addEventListener('click', () => {
        grid.querySelectorAll('.audio-fmt-card').forEach(c => c.classList.remove('selected'));
        card.classList.add('selected');
        audioFormat = card.dataset.audioFmt;
        selectedFormat = audioFormat;
      });
    });
    grid.querySelectorAll('.format-card:not(.audio-fmt-card)').forEach(card => {
      card.addEventListener('click', () => {
        grid.querySelectorAll('.format-card:not(.audio-fmt-card)').forEach(c => c.classList.remove('selected'));
        card.classList.add('selected');
        selectedFormat = card.dataset.format;
      });
    });
  }

  async function loadFormats(url) {
    try {
      const result = await API.listFormats(url);
      formats.video = result.video || [];
      formats.audio = result.audio || [];
      renderFormatGrid();
    } catch(e) {}
  }

  async function startDownload(url, btn) {
    if (!selectedFormat) return;
    btn.disabled = true;
    btn.textContent = 'Starting...';

    try {
      // Check for existing/in-progress download of the same URL
      const existing = await API.checkExistingDownload(url, downloadDir);
      if (existing && existing.exists) {
        if (!confirm(existing.message + '\n\nDownload anyway?')) {
          btn.textContent = 'Download';
          btn.disabled = false;
          return;
        }
      }

      const result = await API.startDownload(url, selectedFormat, downloadDir, audioOnly, audioFormat);
      const dlId = result.id;

      // Replace the card content with inline progress
      const content = document.getElementById('dl-content');
      const title = currentMeta ? currentMeta.title : url;
      const thumb = currentMeta ? currentMeta.thumbnail : '';

      content.innerHTML = `<div class="dl-card">
        <div class="dl-card-header">
          ${thumb ? `<img class="dl-card-thumb" src="${esc(thumb)}" alt="" onerror="this.style.display='none'">` : ''}
          <div class="dl-card-info">
            <h3 class="dl-card-title">${esc(title)}</h3>
            <div class="dl-inline-progress">
              <div class="dl-inline-status">
                <span class="download-status status-downloading" id="ip-status-${esc(dlId)}">DOWNLOADING</span>
                <span class="dl-inline-pct" id="ip-pct-${esc(dlId)}">0%</span>
                <span class="dl-inline-speed" id="ip-speed-${esc(dlId)}"></span>
                <span class="dl-inline-eta" id="ip-eta-${esc(dlId)}"></span>
              </div>
              <div class="download-progress-bar"><div class="download-progress-fill" id="ip-bar-${esc(dlId)}" style="width:0%"></div></div>
            </div>
          </div>
        </div>
        <div class="dl-card-actions" style="justify-content:space-between;align-items:center;">
          <span class="dl-inline-msg" id="ip-msg-${esc(dlId)}" style="font-size:12px;color:var(--text-muted);"></span>
          <div style="display:flex;gap:8px;">
            <button class="btn btn-ghost btn-sm" id="ip-cancel-${esc(dlId)}">Cancel</button>
            <button class="btn btn-primary btn-sm" id="ip-new">Download Another</button>
          </div>
        </div>
      </div>`;

      // Cancel button
      const cancelBtn = content.querySelector(`#ip-cancel-${dlId}`);
      if (cancelBtn) {
        cancelBtn.addEventListener('click', async () => {
          await API.cancelDownload(dlId);
          cancelBtn.textContent = 'Cancelled';
          cancelBtn.disabled = true;
        });
      }

      // Download Another button -- reset the page for a new URL
      const newBtn = content.querySelector('#ip-new');
      if (newBtn) {
        newBtn.addEventListener('click', () => {
          currentMeta = null;
          selectedFormat = null;
          formats = { video: [], audio: [] };
          content.innerHTML = '';
          urlInput.value = '';
          urlInput.focus();
          refreshActiveDownloads();
        });
      }

    } catch (err) {
      btn.textContent = 'Download';
      btn.disabled = false;
    }
  }

  // Update inline progress for a specific download
  function updateActiveDownload(data) {
    if (!data || !data.id) return;

    // Update inline progress card if visible
    const bar = document.getElementById(`ip-bar-${data.id}`);
    const pct = document.getElementById(`ip-pct-${data.id}`);
    const speed = document.getElementById(`ip-speed-${data.id}`);
    const eta = document.getElementById(`ip-eta-${data.id}`);
    const status = document.getElementById(`ip-status-${data.id}`);
    const msg = document.getElementById(`ip-msg-${data.id}`);

    if (bar && data.percent != null) bar.style.width = Math.min(data.percent, 100) + '%';
    if (pct && data.percent != null) pct.textContent = data.percent.toFixed(1) + '%';
    if (speed && data.speed) speed.textContent = data.speed;
    if (eta && data.eta) eta.textContent = 'ETA: ' + data.eta;
    if (status && data.status) {
      status.textContent = data.status.toUpperCase();
      status.className = 'download-status ' + getStatusClass(data.status);
    }
    if (msg) {
      if (data.status === 'complete') msg.textContent = 'Download complete!';
      else if (data.status === 'error') { msg.textContent = data.error || 'Download failed'; msg.style.color = 'var(--red)'; }
      else if (data.status === 'metadata') msg.textContent = 'Fetching metadata...';
    }

    // Update message on completion/error
    if (data.status === 'complete' && msg) {
      msg.style.color = 'var(--green)';
    }
  }

  function formatFilesize(bytes) {
    if (!bytes) return '';
    if (bytes > 1073741824) return (bytes / 1073741824).toFixed(1) + ' GB';
    if (bytes > 1048576) return (bytes / 1048576).toFixed(1) + ' MB';
    if (bytes > 1024) return (bytes / 1024).toFixed(1) + ' KB';
    return bytes + ' B';
  }
}

function cleanupYariaHome() {
  _yariaHomeEventCleanups.forEach(fn => { if (typeof fn === 'function') fn(); });
  _yariaHomeEventCleanups = [];
  _yariaHomeDepsReady = false;
}
