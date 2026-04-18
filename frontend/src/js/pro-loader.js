// Pro module loader - loads Mantorex styles and scripts if available in the build
(function() {
  window._proModulesLoaded = false;
  window._proModulesError = null;

  // Load Pro CSS
  var link = document.createElement('link');
  link.rel = 'stylesheet';
  link.href = 'css/pro.css';
  document.head.appendChild(link);

  // Pro scripts to load in order
  var proScripts = [
    'js/components/card.js',
    'js/components/row.js',
    'js/components/torrentlist.js',
    'js/pages/home.js',
    'js/pages/search.js',
    'js/pages/detail.js',
    'js/pages/player.js',
    'js/pages/library.js'
  ];

  // Use synchronous XHR + eval to load scripts (works with Wails asset server)
  var allLoaded = true;
  for (var i = 0; i < proScripts.length; i++) {
    try {
      var xhr = new XMLHttpRequest();
      xhr.open('GET', proScripts[i], false); // synchronous
      xhr.send();
      if (xhr.status === 200 && xhr.responseText) {
        eval(xhr.responseText);
      } else {
        allLoaded = false;
        break;
      }
    } catch(e) {
      allLoaded = false;
      window._proModulesError = e.message;
      break;
    }
  }
  window._proModulesLoaded = allLoaded;
})();
