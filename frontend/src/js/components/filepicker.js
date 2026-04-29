// In-app file picker for selecting download directories
// Shows quick access locations + browsable directory tree

function showDownloadPicker(onSelect) {
  // Create overlay
  const overlay = document.createElement('div');
  overlay.className = 'fp-overlay';

  const modal = document.createElement('div');
  modal.className = 'fp-modal';

  let currentPath = '';
  let selectedPath = '';

  modal.innerHTML = `
    <div class="fp-header">
      <h3>Choose Download Location</h3>
      <button class="fp-close" title="Cancel">&times;</button>
    </div>
    <div class="fp-body">
      <div class="fp-sidebar">
        <div class="fp-section-title">Quick Access</div>
        <div class="fp-quick" id="fp-quick"></div>
        <div class="fp-section-title" style="margin-top:16px;">Current</div>
        <div class="fp-current" id="fp-current-path" style="font-size:11px;color:var(--text-muted);word-break:break-all;padding:4px 0;"></div>
      </div>
      <div class="fp-main">
        <div class="fp-breadcrumb" id="fp-breadcrumb"></div>
        <div class="fp-list" id="fp-list">
          <div style="padding:20px;text-align:center;color:var(--text-muted);">Loading...</div>
        </div>
      </div>
    </div>
    <div class="fp-footer">
      <div class="fp-new-folder">
        <input type="text" class="fp-new-input" id="fp-new-input" placeholder="New folder name..." style="display:none;">
        <button class="btn btn-ghost btn-sm" id="fp-new-btn">New Folder</button>
      </div>
      <div class="fp-actions">
        <button class="btn btn-ghost btn-sm" id="fp-cancel">Cancel</button>
        <button class="btn btn-primary btn-sm" id="fp-select" disabled>Select This Folder</button>
      </div>
    </div>
  `;

  overlay.appendChild(modal);
  document.body.appendChild(overlay);

  // Close on overlay click or close button
  overlay.addEventListener('click', (e) => { if (e.target === overlay) close(); });
  modal.querySelector('.fp-close').addEventListener('click', close);
  modal.querySelector('#fp-cancel').addEventListener('click', close);

  const selectBtn = modal.querySelector('#fp-select');
  selectBtn.addEventListener('click', () => {
    if (selectedPath) {
      close();
      onSelect(selectedPath);
    }
  });

  // New folder
  const newBtn = modal.querySelector('#fp-new-btn');
  const newInput = modal.querySelector('#fp-new-input');
  newBtn.addEventListener('click', async () => {
    if (newInput.style.display === 'none') {
      newInput.style.display = 'block';
      newInput.focus();
    } else {
      const name = newInput.value.trim();
      if (name && currentPath) {
        // Create folder via a simple mkdir
        try {
          // We don't have a mkdir API, so just select the path with the new name appended
          const newPath = currentPath + '/' + name;
          selectedPath = newPath;
          selectBtn.disabled = false;
          selectBtn.textContent = `Select "${name}"`;
          newInput.style.display = 'none';
          newInput.value = '';
        } catch(e) {}
      }
    }
  });
  newInput.addEventListener('keydown', (e) => { if (e.key === 'Enter') newBtn.click(); if (e.key === 'Escape') { newInput.style.display = 'none'; } });

  // Load quick access locations
  loadQuickAccess();
  // Default to home directory
  navigateTo('~');

  function close() {
    overlay.remove();
  }

  async function loadQuickAccess() {
    const quickEl = modal.querySelector('#fp-quick');
    const locations = [
      { name: 'Downloads', path: '~/Downloads', icon: '📥' },
      { name: 'Mantorex', path: '~/Downloads/Mantorex', icon: '🎬' },
      { name: 'Videos', path: '~/Videos', icon: '🎞' },
      { name: 'Home', path: '~', icon: '🏠' },
      { name: 'Desktop', path: '~/Desktop', icon: '🖥' },
      { name: 'Documents', path: '~/Documents', icon: '📁' },
    ];

    let html = '';
    locations.forEach(loc => {
      html += `<div class="fp-quick-item" data-path="${esc(loc.path)}">
        <span class="fp-quick-icon">${loc.icon}</span>
        <span>${esc(loc.name)}</span>
      </div>`;
    });

    // System dialog fallback
    html += `<div class="fp-quick-item fp-system-btn" style="margin-top:12px;border-top:1px solid rgba(255,255,255,0.04);padding-top:12px;">
      <span class="fp-quick-icon">📂</span>
      <span>System Picker...</span>
    </div>`;

    quickEl.innerHTML = html;

    quickEl.querySelectorAll('.fp-quick-item:not(.fp-system-btn)').forEach(item => {
      item.addEventListener('click', () => {
        quickEl.querySelectorAll('.fp-quick-item').forEach(i => i.classList.remove('active'));
        item.classList.add('active');
        navigateTo(item.dataset.path);
      });
    });

    quickEl.querySelector('.fp-system-btn').addEventListener('click', async () => {
      const dir = await API.selectTorrentDownloadDir();
      if (dir) {
        close();
        onSelect(dir);
      }
    });
  }

  async function navigateTo(path) {
    // Expand ~ to home
    if (path.startsWith('~')) {
      // We'll try to read the directory and see what happens
    }

    currentPath = path;
    selectedPath = path;
    selectBtn.disabled = false;

    const currentEl = modal.querySelector('#fp-current-path');
    currentEl.textContent = path;
    selectBtn.textContent = 'Select This Folder';

    // Update breadcrumb
    const bcEl = modal.querySelector('#fp-breadcrumb');
    const parts = path.split('/').filter(p => p);
    let bcHtml = '';
    let cumPath = '';
    parts.forEach((part, i) => {
      cumPath += '/' + part;
      if (part === '~') cumPath = '~';
      const isLast = i === parts.length - 1;
      bcHtml += `<span class="fp-bc-item${isLast ? ' active' : ''}" data-path="${esc(cumPath)}">${esc(part === '~' ? 'Home' : part)}</span>`;
      if (!isLast) bcHtml += '<span class="fp-bc-sep">/</span>';
    });
    bcEl.innerHTML = bcHtml;
    bcEl.querySelectorAll('.fp-bc-item:not(.active)').forEach(item => {
      item.addEventListener('click', () => navigateTo(item.dataset.path));
    });

    // Load directory contents
    const listEl = modal.querySelector('#fp-list');
    listEl.innerHTML = '<div style="padding:20px;text-align:center;color:var(--text-muted);font-size:13px;">Loading...</div>';

    try {
      // Use Go backend to list directories
      const dirs = await API.listDirectories(path);
      if (!dirs || dirs.length === 0) {
        listEl.innerHTML = '<div style="padding:20px;text-align:center;color:var(--text-muted);font-size:13px;">Empty folder</div>';
        return;
      }

      let html = '';
      dirs.forEach(d => {
        html += `<div class="fp-dir-item" data-path="${esc(d.path)}">
          <span class="fp-dir-icon">📁</span>
          <span class="fp-dir-name">${esc(d.name)}</span>
        </div>`;
      });
      listEl.innerHTML = html;

      listEl.querySelectorAll('.fp-dir-item').forEach(item => {
        item.addEventListener('click', () => navigateTo(item.dataset.path));
      });
    } catch(e) {
      listEl.innerHTML = `<div style="padding:20px;text-align:center;color:var(--red);font-size:13px;">Cannot access this folder</div>`;
    }
  }
}
