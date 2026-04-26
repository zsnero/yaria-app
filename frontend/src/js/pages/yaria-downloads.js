// Yaria Downloads Page - active and completed downloads with progress
let _yariaDownloadsCleanups = [];
let _yariaDownloadsInterval = null;

function renderYariaDownloads(container) {
  container.innerHTML = '';

  const page = document.createElement('div');
  page.className = 'yaria-downloads-page';

  page.innerHTML = `
    <div class="yaria-downloads-header">
      <div>
        <h2 class="section-heading">Downloads</h2>
        <p class="section-subtitle">Manage your video downloads</p>
      </div>
      <button class="btn btn-ghost btn-sm" id="yd-clear-all" style="color:var(--text-muted);">Clear Completed</button>
    </div>
    <div id="yd-active-section"></div>
    <div id="yd-completed-section"></div>
    <div id="yd-empty" class="no-results" style="padding:48px 0;display:none;">No downloads yet. Go to the Yaria tab to start downloading.</div>
  `;

  container.appendChild(page);

  // Clear completed button
  page.querySelector('#yd-clear-all').addEventListener('click', async () => {
    try {
      const downloads = await API.getDownloads();
      const arr = Array.isArray(downloads) ? downloads : [];
      for (const dl of arr) {
        if (dl.status === 'complete' || dl.status === 'error' || dl.status === 'cancelled') {
          await API.removeDownload(dl.id);
        }
      }
      refreshDownloadList();
    } catch(e) {}
  });

  // Subscribe to progress events
  if (window.runtime && window.runtime.EventsOn) {
    const off = window.runtime.EventsOn('download-progress', (data) => {
      updateDownloadItemUI(data);
    });
    _yariaDownloadsCleanups.push(off);
  }

  refreshDownloadList();
  _yariaDownloadsInterval = setInterval(refreshDownloadList, 3000);
}

function cleanupYariaDownloads() {
  _yariaDownloadsCleanups.forEach(fn => { if (typeof fn === 'function') fn(); });
  _yariaDownloadsCleanups = [];
  if (_yariaDownloadsInterval) { clearInterval(_yariaDownloadsInterval); _yariaDownloadsInterval = null; }
}

async function refreshDownloadList() {
  const activeSection = document.getElementById('yd-active-section');
  const completedSection = document.getElementById('yd-completed-section');
  const emptyEl = document.getElementById('yd-empty');
  if (!activeSection || !completedSection) return;

  try {
    const downloads = await API.getDownloads();
    const arr = Array.isArray(downloads) ? downloads : [];

    if (arr.length === 0) {
      activeSection.innerHTML = '';
      completedSection.innerHTML = '';
      if (emptyEl) emptyEl.style.display = 'block';
      return;
    }
    if (emptyEl) emptyEl.style.display = 'none';

    const active = arr.filter(d => d.status === 'downloading' || d.status === 'queued' || d.status === 'metadata');
    const completed = arr.filter(d => d.status === 'complete' || d.status === 'error' || d.status === 'cancelled');

    // Active section
    if (active.length > 0) {
      let html = '<h3 class="yd-section-title">Active</h3>';
      active.forEach(dl => { html += renderDownloadItem(dl, true); });
      activeSection.innerHTML = html;
      bindButtons(activeSection);
    } else {
      activeSection.innerHTML = '';
    }

    // Completed section
    if (completed.length > 0) {
      let html = '<h3 class="yd-section-title" style="margin-top:24px;">History</h3>';
      completed.forEach(dl => { html += renderDownloadItem(dl, false); });
      completedSection.innerHTML = html;
      bindButtons(completedSection);
    } else {
      completedSection.innerHTML = '';
    }
  } catch(e) {}
}

function renderDownloadItem(dl, isActive) {
  const pct = dl.percent || 0;
  const sc = getStatusClass(dl.status);

  return `<div class="download-item" data-id="${esc(dl.id)}">
    ${dl.thumbnail ? `<img class="dl-item-thumb" src="${esc(dl.thumbnail)}" alt="" onerror="this.style.display='none'">` : ''}
    <div class="download-item-info">
      <div class="download-item-title">${esc(dl.title || dl.url || 'Download')}</div>
      <div class="download-item-details">
        <span class="download-status ${sc}">${esc(dl.status || '').toUpperCase()}</span>
        ${pct > 0 ? `<span class="dl-item-pct">${pct.toFixed(1)}%</span>` : ''}
        ${dl.speed ? `<span class="dl-item-speed">${esc(dl.speed)}</span>` : ''}
        ${dl.eta ? `<span class="dl-item-eta">ETA: ${esc(dl.eta)}</span>` : ''}
        ${dl.started_at ? `<span class="dl-item-time">${esc(dl.started_at)}</span>` : ''}
      </div>
      ${isActive ? `<div class="download-progress-bar"><div class="download-progress-fill" style="width:${Math.min(pct,100)}%"></div></div>` : ''}
      ${dl.error ? `<div class="dl-item-error">${esc(dl.error)}</div>` : ''}
    </div>
    <div class="download-item-actions">
      ${isActive ? `<button class="btn btn-ghost btn-sm dl-cancel-btn" data-id="${esc(dl.id)}">Cancel</button>` : ''}
      ${!isActive ? `<button class="btn btn-ghost btn-sm dl-remove-btn" data-id="${esc(dl.id)}">Remove</button>` : ''}
    </div>
  </div>`;
}

function bindButtons(container) {
  container.querySelectorAll('.dl-cancel-btn').forEach(btn => {
    btn.addEventListener('click', async () => {
      await API.cancelDownload(btn.dataset.id);
      refreshDownloadList();
    });
  });
  container.querySelectorAll('.dl-remove-btn').forEach(btn => {
    btn.addEventListener('click', async () => {
      await API.removeDownload(btn.dataset.id);
      refreshDownloadList();
    });
  });
}

function updateDownloadItemUI(data) {
  if (!data || !data.id) return;
  const item = document.querySelector(`.download-item[data-id="${data.id}"]`);
  if (!item) { refreshDownloadList(); return; }

  const bar = item.querySelector('.download-progress-fill');
  const status = item.querySelector('.download-status');
  const pctEl = item.querySelector('.dl-item-pct');
  const speedEl = item.querySelector('.dl-item-speed');
  const etaEl = item.querySelector('.dl-item-eta');

  if (bar && data.percent != null) bar.style.width = Math.min(data.percent, 100) + '%';
  if (status && data.status) { status.textContent = data.status.toUpperCase(); status.className = 'download-status ' + getStatusClass(data.status); }
  if (pctEl && data.percent != null) pctEl.textContent = data.percent.toFixed(1) + '%';
  if (speedEl && data.speed) speedEl.textContent = data.speed;
  if (etaEl && data.eta) etaEl.textContent = 'ETA: ' + data.eta;

  if (data.status === 'complete' || data.status === 'error' || data.status === 'cancelled') {
    setTimeout(refreshDownloadList, 500);
  }
}

function getStatusClass(status) {
  switch (status) {
    case 'downloading': return 'status-downloading';
    case 'complete': return 'status-complete';
    case 'error': return 'status-error';
    case 'cancelled': return 'status-cancelled';
    case 'queued': case 'metadata': return 'status-queued';
    default: return '';
  }
}
