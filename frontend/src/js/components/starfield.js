// Cosmic starfield - optimized with will-change and reduced count
(function() {
  const container = document.getElementById('starfield');
  if (!container) return;

  // Use a single canvas-like approach: batch create all stars as one innerHTML
  // This is much faster than creating 100+ individual DOM elements
  let html = '';

  // Twinkling stars (reduced from 200 to 120 for performance)
  for (let i = 0; i < 120; i++) {
    const size = Math.random() * 2 + 0.5;
    const x = Math.random() * 100;
    const y = Math.random() * 100;
    const dur = (Math.random() * 5 + 4).toFixed(1);
    const delay = (Math.random() * 8).toFixed(1);
    const peak = (Math.random() * 0.5 + 0.15).toFixed(2);

    html += `<div class="star" style="width:${size}px;height:${size}px;left:${x}%;top:${y}%;--dur:${dur}s;--delay:${delay}s;--peak:${peak}"></div>`;
  }

  // Bright accent stars (fewer, larger)
  for (let i = 0; i < 8; i++) {
    const size = Math.random() * 2 + 2.5;
    const x = Math.random() * 100;
    const y = Math.random() * 100;
    const dur = (Math.random() * 6 + 5).toFixed(1);
    const delay = (Math.random() * 10).toFixed(1);

    html += `<div class="star" style="width:${size}px;height:${size}px;left:${x}%;top:${y}%;--dur:${dur}s;--delay:${delay}s;--peak:0.8;background:radial-gradient(circle,#fff 0%,rgba(167,139,250,0.4) 50%,transparent 70%)"></div>`;
  }

  container.innerHTML = html;

  // Shooting stars at random intervals
  function shootingStar() {
    const star = document.createElement('div');
    star.className = 'shooting-star';
    const x = Math.random() * 80 + 10;
    const y = Math.random() * 40;
    const angle = Math.random() * 30 + 15;
    const dist = Math.random() * 250 + 150;
    const dx = Math.cos(angle * Math.PI / 180) * dist;
    const dy = Math.sin(angle * Math.PI / 180) * dist;
    const dur = (Math.random() * 0.8 + 0.6).toFixed(2);
    const w = Math.random() * 50 + 30;

    star.style.cssText = `left:${x}%;top:${y}%;width:${w}px;--dx:${dx}px;--dy:${dy}px;--shoot-dur:${dur}s;--shoot-delay:0s`;
    container.appendChild(star);
    setTimeout(() => star.remove(), parseFloat(dur) * 1000 + 200);
  }

  // Schedule shooting stars less frequently for performance
  function schedule() {
    setTimeout(() => {
      shootingStar();
      schedule();
    }, Math.random() * 10000 + 6000); // every 6-16 seconds
  }
  setTimeout(schedule, 3000);
})();
