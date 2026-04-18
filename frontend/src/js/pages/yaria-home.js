// Yaria Home Page - Video & Audio Downloader with branding hero
let _yariaHomeEventCleanups = [];
let _yariaHomeDepsReady = false;

async function renderYariaHome(container) {
  container.innerHTML = '';

  const page = document.createElement('div');
  page.className = 'yaria-home-page';

  // Branding hero
  const hero = document.createElement('div');
  hero.className = 'yaria-hero';
  hero.innerHTML = `
    <h1 class="yaria-title">Yaria</h1>
    <p class="yaria-subtitle">Video & Audio Downloader</p>
    <p class="yaria-desc">Download from YouTube, Vimeo, Twitter, TikTok, and 1000+ sites</p>
  `;
  page.appendChild(hero);

  // URL input section
  const urlSection = document.createElement('div');
  urlSection.className = 'download-url-section';
  urlSection.innerHTML = `
    <div class="download-input-wrap">
      <input type="text" class="download-input" id="dl-url" placeholder="Paste a video URL (YouTube, Vimeo, Twitter, etc.)" autocomplete="off">
      <button class="btn btn-primary" id="dl-fetch-btn">Fetch</button>
    </div>
    <div id="dl-deps-status" class="download-deps-status">
      <div class="spinner" style="width:18px;height:18px;border-width:2px;margin-bottom:0;display:inline-block;vertical-align:middle;"></div>
      <span style="margin-left:8px;color:var(--text-muted);font-size:13px;">Preparing download tools...</span>
    </div>
  `;
  page.appendChild(urlSection);

  // Metadata display (hidden initially)
  const metaSection = document.createElement('div');
  metaSection.className = 'download-meta';
  metaSection.id = 'dl-meta';
  metaSection.style.display = 'none';
  page.appendChild(metaSection);

  // Format selection (hidden initially)
  const formatSection = document.createElement('div');
  formatSection.className = 'download-formats';
  formatSection.id = 'dl-formats';
  formatSection.style.display = 'none';
  page.appendChild(formatSection);

  // Download controls (hidden initially)
  const controlsSection = document.createElement('div');
  controlsSection.className = 'download-controls';
  controlsSection.id = 'dl-controls';
  controlsSection.style.display = 'none';
  controlsSection.innerHTML = `
    <div class="dir-picker">
      <label class="setting-label">Download Directory</label>
      <div class="dir-picker-row">
        <input type="text" class="setting-input dir-input" id="dl-dir" readonly placeholder="Select download directory...">
        <button class="btn btn-ghost btn-sm" id="dl-browse-btn">Browse</button>
      </div>
    </div>
    <button class="btn btn-primary download-btn" id="dl-start-btn" disabled>Download</button>
    <div id="dl-toast" class="download-toast" style="display:none;"></div>
  `;
  page.appendChild(controlsSection);

  container.appendChild(page);

  // State
  let currentMeta = null;
  let selectedFormat = null;
  let audioOnly = false;
  let audioFormat = 'mp3';
  let formats = { video: [], audio: [] };

  // Elements
  const urlInput = page.querySelector('#dl-url');
  const fetchBtn = page.querySelector('#dl-fetch-btn');
  const depsStatus = page.querySelector('#dl-deps-status');
  const startBtn = page.querySelector('#dl-start-btn');
  const dirInput = page.querySelector('#dl-dir');
  const browseBtn = page.querySelector('#dl-browse-btn');
  const toastEl = page.querySelector('#dl-toast');

  // Init deps
  initDeps(depsStatus, urlInput, fetchBtn);

  // Load saved download dir
  loadDownloadDir(dirInput);

  // Fetch metadata on click or Enter
  const doFetch = () => {
    const url = urlInput.value.trim();
    if (!url) return;
    fetchMetadata(url, metaSection, formatSection, controlsSection, page);
  };
  fetchBtn.addEventListener('click', doFetch);
  urlInput.addEventListener('keydown', (e) => {
    if (e.key === 'Enter') doFetch();
  });

  // Browse directory
  browseBtn.addEventListener('click', async () => {
    try {
      const dir = await API.selectDownloadDir();
      if (dir) dirInput.value = dir;
    } catch (err) {
      console.error('Browse dir failed:', err);
    }
  });

  // Start download
  startBtn.addEventListener('click', async () => {
    const url = urlInput.value.trim();
    const dir = dirInput.value.trim();
    if (!url || !selectedFormat) return;

    startBtn.disabled = true;
    startBtn.textContent = 'Starting...';
    try {
      await API.startDownload(url, selectedFormat, dir, audioOnly, audioFormat);
      startBtn.textContent = 'Download Started';

      // Show toast notification
      toastEl.style.display = 'block';
      toastEl.innerHTML = `Download started! <a href="#/yaria/downloads" class="toast-link">View Downloads</a>`;
      setTimeout(() => {
        toastEl.style.display = 'none';
      }, 8000);

      setTimeout(() => {
        startBtn.textContent = 'Download';
        startBtn.disabled = false;
      }, 2000);
    } catch (err) {
      startBtn.textContent = 'Download';
      startBtn.disabled = false;
      alert('Download failed: ' + err.message);
    }
  });

  // Subscribe to Wails events
  if (window.runtime && window.runtime.EventsOn) {
    const offDepsReady = window.runtime.EventsOn('deps-ready', () => {
      _yariaHomeDepsReady = true;
      depsStatus.style.display = 'none';
      urlInput.disabled = false;
      fetchBtn.disabled = false;
    });
    _yariaHomeEventCleanups.push(offDepsReady);
  }

  // Focus URL input
  setTimeout(() => urlInput.focus(), 100);

  // --- Inner functions ---

  async function initDeps(statusEl, inputEl, btnEl) {
    inputEl.disabled = true;
    btnEl.disabled = true;
    try {
      // Check if deps are already available
      const check = await API.checkDeps();
      if (check && check.ready) {
        _yariaHomeDepsReady = true;
        statusEl.style.display = 'none';
        inputEl.disabled = false;
        btnEl.disabled = false;
        return;
      }
    } catch (e) { /* proceed to init */ }

    try {
      await API.initDeps();
      _yariaHomeDepsReady = true;
      statusEl.style.display = 'none';
      inputEl.disabled = false;
      btnEl.disabled = false;
    } catch (err) {
      statusEl.innerHTML = `<span style="color:var(--red);font-size:13px;">Failed to initialize download tools: ${esc(err.message)}</span>`;
    }
  }

  async function loadDownloadDir(inputEl) {
    try {
      const dir = await API.getDownloadDir();
      if (dir) inputEl.value = dir;
    } catch (e) { /* ignore */ }
  }

  async function fetchMetadata(url, metaEl, formatEl, controlsEl, pageEl) {
    metaEl.style.display = 'block';
    metaEl.innerHTML = '<div class="loading-screen" style="height:auto;padding:24px;"><div class="spinner"></div><p>Fetching video info...</p></div>';
    formatEl.style.display = 'none';
    controlsEl.style.display = 'none';
    currentMeta = null;
    selectedFormat = null;

    try {
      const meta = await API.fetchMetadata(url);
      currentMeta = meta;

      let metaHTML = `<div class="download-meta-card">`;
      metaHTML += `<div class="download-meta-info">`;
      metaHTML += `<h3 class="download-meta-title">${esc(meta.title || 'Unknown Title')}</h3>`;
      if (meta.playlist && meta.playlist_count > 0) {
        metaHTML += `<div class="download-meta-playlist">Playlist: ${esc(meta.playlist)} (${meta.playlist_count} videos)</div>`;
      }
      if (meta.duration) {
        const mins = Math.floor(meta.duration / 60);
        const secs = meta.duration % 60;
        metaHTML += `<div class="download-meta-duration">Duration: ${mins}:${String(secs).padStart(2, '0')}</div>`;
      }
      if (meta.uploader) {
        metaHTML += `<div class="download-meta-uploader">By: ${esc(meta.uploader)}</div>`;
      }
      metaHTML += `</div>`;
      if (meta.thumbnail) {
        metaHTML += `<img class="download-meta-thumb" src="${esc(meta.thumbnail)}" alt="Thumbnail">`;
      }
      metaHTML += `</div>`;
      metaEl.innerHTML = metaHTML;

      // Now fetch formats
      loadFormats(url, formatEl, controlsEl, pageEl);
    } catch (err) {
      metaEl.innerHTML = `<div class="no-results" style="padding:20px;">Failed to fetch metadata: ${esc(err.message)}</div>`;
    }
  }

  async function loadFormats(url, formatEl, controlsEl, pageEl) {
    formatEl.style.display = 'block';
    formatEl.innerHTML = '<div class="loading-screen" style="height:auto;padding:20px;"><div class="spinner"></div><p>Loading formats...</p></div>';

    try {
      const result = await API.listFormats(url);
      formats.video = (result.video || []);
      formats.audio = (result.audio || []);
      renderFormats(formatEl, controlsEl, pageEl);
    } catch (err) {
      formatEl.innerHTML = `<div class="no-results" style="padding:20px;">Failed to load formats: ${esc(err.message)}</div>`;
    }
  }

  function renderFormats(formatEl, controlsEl, pageEl) {
    let html = `<div class="format-header">
      <h3 class="format-section-title">Select Format</h3>
      <div class="audio-toggle-wrap">
        <label class="audio-toggle">
          <input type="checkbox" id="dl-audio-toggle" ${audioOnly ? 'checked' : ''}>
          <span class="audio-toggle-slider"></span>
        </label>
        <span class="audio-toggle-label">Audio Only</span>
      </div>
    </div>`;

    if (audioOnly) {
      // Audio format selector
      html += `<div class="audio-format-selector">
        <label class="setting-label" style="margin-bottom:8px;">Audio Format</label>
        <div class="format-grid">`;
      const audioFormats = ['mp3', 'm4a', 'opus', 'wav', 'flac'];
      audioFormats.forEach(fmt => {
        const sel = audioFormat === fmt ? ' selected' : '';
        html += `<div class="format-card audio-fmt-card${sel}" data-audio-fmt="${fmt}">
          <div class="format-card-label">${fmt.toUpperCase()}</div>
        </div>`;
      });
      html += `</div></div>`;

      // Also show available audio streams if any
      if (formats.audio.length > 0) {
        html += `<div class="format-grid" style="margin-top:12px;">`;
        formats.audio.forEach((f, i) => {
          const sel = selectedFormat === (f.resolution || f.format_id || `audio-${i}`) ? ' selected' : '';
          html += `<div class="format-card${sel}" data-format="${esc(f.resolution || f.format_id || 'audio-' + i)}">
            <div class="format-card-label">${esc(f.resolution || f.format_note || 'Audio')}</div>
            <div class="format-card-ext">${esc(f.ext || '')}</div>
            ${f.filesize ? `<div class="format-card-size">${formatFilesize(f.filesize)}</div>` : ''}
          </div>`;
        });
        html += `</div>`;
      }
    } else {
      // Video format cards
      html += `<div class="format-grid">`;
      if (formats.video.length > 0) {
        // Auto-select best format: prefer 1080p, fallback to highest
        if (!selectedFormat) {
          const preferred = formats.video.find(f => f.resolution === '1080p');
          if (preferred) selectedFormat = preferred.resolution || preferred.format_id;
          else selectedFormat = formats.video[0].resolution || formats.video[0].format_id || 'video-0';
          updateStartBtn();
        }
        formats.video.forEach((f, i) => {
          const label = f.resolution || f.format_note || f.format_id || `Format ${i + 1}`;
          const fmtKey = f.resolution || f.format_id || `video-${i}`;
          const sel = selectedFormat === fmtKey ? ' selected' : '';
          html += `<div class="format-card${sel}" data-format="${esc(fmtKey)}">
            <div class="format-card-label">${esc(label)}</div>
            <div class="format-card-ext">${esc(f.ext || '')}</div>
            ${f.filesize ? `<div class="format-card-size">${formatFilesize(f.filesize)}</div>` : ''}
          </div>`;
        });
      } else {
        // No formats from yt-dlp, provide default resolution options
        if (!selectedFormat) { selectedFormat = '1080p'; updateStartBtn(); }
        ['2160p', '1440p', '1080p', '720p', '480p', '360p'].forEach(res => {
          const sel = selectedFormat === res ? ' selected' : '';
          html += `<div class="format-card${sel}" data-format="${res}">
            <div class="format-card-label">${res}</div>
          </div>`;
        });
      }
      html += `</div>`;
    }

    formatEl.innerHTML = html;
    controlsEl.style.display = 'block';

    // Bind audio toggle
    const audioToggle = formatEl.querySelector('#dl-audio-toggle');
    if (audioToggle) {
      audioToggle.addEventListener('change', (e) => {
        audioOnly = e.target.checked;
        selectedFormat = null;
        updateStartBtn();
        renderFormats(formatEl, controlsEl, pageEl);
      });
    }

    // Bind audio format cards
    formatEl.querySelectorAll('.audio-fmt-card').forEach(card => {
      card.addEventListener('click', () => {
        formatEl.querySelectorAll('.audio-fmt-card').forEach(c => c.classList.remove('selected'));
        card.classList.add('selected');
        audioFormat = card.dataset.audioFmt;
      });
    });

    // Bind format cards
    formatEl.querySelectorAll('.format-card:not(.audio-fmt-card)').forEach(card => {
      card.addEventListener('click', () => {
        formatEl.querySelectorAll('.format-card:not(.audio-fmt-card)').forEach(c => c.classList.remove('selected'));
        card.classList.add('selected');
        selectedFormat = card.dataset.format;
        updateStartBtn();
      });
    });
  }

  function updateStartBtn() {
    startBtn.disabled = !selectedFormat;
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
  // Remove Wails event listeners
  _yariaHomeEventCleanups.forEach(fn => {
    if (typeof fn === 'function') fn();
  });
  _yariaHomeEventCleanups = [];
  _yariaHomeDepsReady = false;
}
