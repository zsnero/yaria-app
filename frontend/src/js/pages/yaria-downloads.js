// Yaria Downloads Page - manages active and completed downloads
let _yariaDownloadsCleanups = [];
let _yariaDownloadsInterval = null;

function renderYariaDownloads(container) {
  container.innerHTML = '';

  const page = document.createElement('div');
  page.className = 'yaria-downloads-page';

  const header = document.createElement('div');
  header.className = 'yaria-downloads-header';
  header.innerHTML = `
    <h2 class="section-heading">Downloads</h2>
    <p class="section-subtitle">Manage your video downloads</p>
  `;
  page.appendChild(header);

  const listEl = document.createElement('div');
  listEl.className = 'downloads-list';
  listEl.id = 'yd-list';
  listEl.innerHTML = '<div class="no-results" style="padding:32px 0;">No downloads yet</div>';
  page.appendChild(listEl);

  container.appendChild(page);

  // Subscribe to Wails events for real-time progress
  if (window.runtime && window.runtime.EventsOn) {
    const off = window.runtime.EventsOn('download-progress', (data) => {
      updateDownloadItemUI(data, listEl);
    });
    _yariaDownloadsCleanups.push(off);
  }

  // Refresh list immediately + periodically
  refreshDownloadList(listEl);
  _yariaDownloadsInterval = setInterval(() => {
    refreshDownloadList(listEl);
  }, 3000);
}

function cleanupYariaDownloads() {
  // Remove Wails event listeners
  _yariaDownloadsCleanups.forEach(fn => {
    if (typeof fn === 'function') fn();
  });
  _yariaDownloadsCleanups = [];

  // Clear refresh interval
  if (_yariaDownloadsInterval) {
    clearInterval(_yariaDownloadsInterval);
    _yariaDownloadsInterval = null;
  }
}

// --- Shared download list functions (used by both yaria-home.js and yaria-downloads.js) ---

async function refreshDownloadList(listEl) {
  if (!listEl) return;
  try {
    const data = await API.getDownloads();
    const downloads = data.downloads || data || [];
    if (!Array.isArray(downloads) || downloads.length === 0) {
      listEl.innerHTML = '<div class="no-results" style="padding:32px 0;">No downloads yet</div>';
      return;
    }
    renderDownloadList(listEl, downloads);
  } catch (e) {
    // Silently fail on refresh
  }
}

function renderDownloadList(listEl, downloads) {
  let html = '';
  downloads.forEach(dl => {
    const statusClass = getStatusClass(dl.status);
    const progress = dl.percent || dl.progress || 0;
    const showProgress = dl.status === 'downloading' || dl.status === 'queued' || dl.status === 'metadata';
    const showCancel = dl.status === 'downloading' || dl.status === 'queued' || dl.status === 'metadata';
    const showRemove = dl.status === 'complete' || dl.status === 'error' || dl.status === 'cancelled';

    html += `<div class="download-item" data-id="${esc(dl.id)}">
      <div class="download-item-info">
        <div class="download-item-title">${esc(dl.title || dl.url || 'Download')}</div>
        <div class="download-item-details">
          <span class="download-status ${statusClass}">${esc(dl.status || 'unknown').toUpperCase()}</span>
          ${progress > 0 ? `<span class="download-percent">${progress.toFixed(1)}%</span>` : ''}
          ${dl.speed ? `<span class="download-speed">${esc(dl.speed)}</span>` : ''}
          ${dl.eta ? `<span class="download-eta">ETA: ${esc(dl.eta)}</span>` : ''}
        </div>
        ${showProgress ? `<div class="download-progress-bar"><div class="download-progress-fill" style="width:${Math.min(progress, 100)}%"></div></div>` : ''}
        ${dl.error ? `<div class="download-error-msg">${esc(dl.error)}</div>` : ''}
      </div>
      <div class="download-item-actions">
        ${showCancel ? `<button class="btn btn-ghost btn-sm dl-cancel-btn" data-id="${esc(dl.id)}">Cancel</button>` : ''}
        ${showRemove ? `<button class="btn btn-ghost btn-sm dl-remove-btn" data-id="${esc(dl.id)}">Remove</button>` : ''}
      </div>
    </div>`;
  });

  listEl.innerHTML = html;

  // Bind cancel buttons
  listEl.querySelectorAll('.dl-cancel-btn').forEach(btn => {
    btn.addEventListener('click', async () => {
      try {
        await API.cancelDownload(btn.dataset.id);
        refreshDownloadList(listEl);
      } catch (e) { console.error('Cancel failed:', e); }
    });
  });

  // Bind remove buttons
  listEl.querySelectorAll('.dl-remove-btn').forEach(btn => {
    btn.addEventListener('click', async () => {
      try {
        await API.removeDownload(btn.dataset.id);
        refreshDownloadList(listEl);
      } catch (e) { console.error('Remove failed:', e); }
    });
  });
}

function updateDownloadItemUI(data, listEl) {
  if (!listEl || !data || !data.id) return;
  const item = listEl.querySelector(`.download-item[data-id="${data.id}"]`);
  if (!item) {
    // New item -- just refresh the full list
    refreshDownloadList(listEl);
    return;
  }

  // Update progress bar
  const fill = item.querySelector('.download-progress-fill');
  const pct = data.percent || data.progress || 0;
  if (fill && pct > 0) {
    fill.style.width = Math.min(pct, 100) + '%';
  }

  // Update percent display
  const percentEl = item.querySelector('.download-percent');
  if (percentEl && pct > 0) {
    percentEl.textContent = pct.toFixed(1) + '%';
  } else if (!percentEl && pct > 0) {
    const details = item.querySelector('.download-item-details');
    if (details) {
      const s = document.createElement('span');
      s.className = 'download-percent';
      s.textContent = pct.toFixed(1) + '%';
      const badge = details.querySelector('.download-status');
      if (badge && badge.nextSibling) details.insertBefore(s, badge.nextSibling);
      else details.appendChild(s);
    }
  }

  // Update status badge
  const badge = item.querySelector('.download-status');
  if (badge && data.status) {
    badge.textContent = data.status;
    badge.className = 'download-status ' + getStatusClass(data.status);
  }

  // Update speed/eta
  const details = item.querySelector('.download-item-details');
  if (details) {
    const speedEl = details.querySelector('.download-speed');
    const etaEl = details.querySelector('.download-eta');
    if (speedEl && data.speed) speedEl.textContent = data.speed;
    else if (data.speed && !speedEl) {
      const s = document.createElement('span');
      s.className = 'download-speed';
      s.textContent = data.speed;
      details.appendChild(s);
    }
    if (etaEl && data.eta) etaEl.textContent = 'ETA: ' + data.eta;
    else if (data.eta && !etaEl) {
      const s = document.createElement('span');
      s.className = 'download-eta';
      s.textContent = 'ETA: ' + data.eta;
      details.appendChild(s);
    }
  }

  // If status changed to complete/error/cancelled, refresh full list for button updates
  if (data.status === 'complete' || data.status === 'error' || data.status === 'cancelled') {
    setTimeout(() => refreshDownloadList(listEl), 500);
  }
}

function getStatusClass(status) {
  switch (status) {
    case 'downloading': return 'status-downloading';
    case 'complete': return 'status-complete';
    case 'error': return 'status-error';
    case 'cancelled': return 'status-cancelled';
    case 'queued': return 'status-queued';
    default: return '';
  }
}
