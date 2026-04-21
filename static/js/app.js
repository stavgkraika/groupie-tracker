// app.js — client-side live search, range filters, checkbox filters, and refresh.
// No framework or build step required; runs as a plain IIFE.
(function () {
  const searchInput    = document.getElementById("search-input");
  const artistsGrid    = document.getElementById("artists-grid");
  const resultCount    = document.getElementById("result-count");
  const status         = document.getElementById("search-status");
  const refreshButton  = document.getElementById("refresh-button");
  const resetButton    = document.getElementById("reset-button");
  const creationMin    = document.getElementById("creation-min");
  const creationMax    = document.getElementById("creation-max");
  const albumMin       = document.getElementById("album-min");
  const albumMax       = document.getElementById("album-max");
  const membersMin     = document.getElementById("members-min");
  const membersMax     = document.getElementById("members-max");
  const locationInput  = document.getElementById("location-input");

  if (!searchInput || !artistsGrid || !resultCount || !status) return;

  // controller holds the AbortController for the in-flight request so it can be
  // cancelled when a new input event arrives before the response returns.
  let controller = null;
  // debounceId holds the setTimeout handle used to debounce rapid input events.
  let debounceId = null;

  // ── Event listeners ────────────────────────────────────────────────────────

  // All filter inputs share the same debounced handler.
  const filterInputs = [
    searchInput, creationMin, creationMax,
    albumMin, albumMax, membersMin, membersMax, locationInput,
  ].filter(Boolean);

  filterInputs.forEach(function (el) {
    el.addEventListener("input", scheduleSearch);
  });

  // Checkboxes sync to the min/max number inputs and trigger a search.
  document.querySelectorAll(".member-checkbox").forEach(function (cb) {
    cb.addEventListener("change", function () {
      syncCheckboxesToRange();
      scheduleSearch();
    });
  });

  if (refreshButton) {
    refreshButton.addEventListener("click", async function () {
      refreshButton.disabled = true;
      status.textContent = "Refreshing data from the upstream API...";
      try {
        const response = await fetch("/api/refresh", { method: "POST" });
        if (!response.ok) throw new Error("refresh failed");
        status.textContent = "Data refreshed. Running search again.";
        await runSearch();
      } catch {
        status.textContent = "Refresh failed. The server stayed online and returned a controlled error.";
      } finally {
        refreshButton.disabled = false;
      }
    });
  }

  if (resetButton) {
    resetButton.addEventListener("click", function () {
      filterInputs.forEach(function (el) { el.value = ""; });
      document.querySelectorAll(".member-checkbox").forEach(function (cb) {
        cb.checked = false;
      });
      runSearch();
    });
  }

  // ── Helpers ────────────────────────────────────────────────────────────────

  function scheduleSearch() {
    clearTimeout(debounceId);
    debounceId = window.setTimeout(runSearch, 250);
  }

  // When checkboxes change, set the min/max number inputs to the lowest and
  // highest checked values so the server receives a proper range.
  function syncCheckboxesToRange() {
    const checked = Array.from(
      document.querySelectorAll(".member-checkbox:checked")
    ).map(function (cb) { return parseInt(cb.value, 10); });

    if (checked.length === 0) {
      if (membersMin) membersMin.value = "";
      if (membersMax) membersMax.value = "";
    } else {
      if (membersMin) membersMin.value = Math.min.apply(null, checked);
      if (membersMax) membersMax.value = Math.max.apply(null, checked);
    }
  }

  // buildParams collects every active filter into a URLSearchParams object.
  function buildParams() {
    const p = new URLSearchParams();
    const q = searchInput.value.trim();
    if (q)                              p.set("q",            q);
    if (creationMin && creationMin.value.trim())   p.set("creation_min", creationMin.value.trim());
    if (creationMax && creationMax.value.trim())   p.set("creation_max", creationMax.value.trim());
    if (albumMin    && albumMin.value.trim())      p.set("album_min",    albumMin.value.trim());
    if (albumMax    && albumMax.value.trim())      p.set("album_max",    albumMax.value.trim());
    if (membersMin  && membersMin.value.trim())    p.set("members_min",  membersMin.value.trim());
    if (membersMax  && membersMax.value.trim())    p.set("members_max",  membersMax.value.trim());
    if (locationInput && locationInput.value.trim()) p.set("location",   locationInput.value.trim());
    return p;
  }

  // runSearch sends a GET /api/search request with all active filter params,
  // cancels any previous in-flight request, and re-renders the artist grid.
  async function runSearch() {
    if (controller) controller.abort();
    controller = new AbortController();

    const params = buildParams();
    status.textContent = params.toString()
      ? "Filtering artists..."
      : "Showing all artists.";

    try {
      const response = await fetch("/api/search?" + params.toString(), {
        signal: controller.signal,
      });
      if (!response.ok) throw new Error("search failed");

      const artists = await response.json();
      renderArtists(artists);
      resultCount.textContent = artists.length + " result(s)";
      status.textContent = params.toString()
        ? "Found " + artists.length + " matching artist(s)."
        : "Showing all artists.";
    } catch (error) {
      // Ignore aborted requests — expected when the user types quickly.
      if (error.name === "AbortError") return;
      status.textContent = "Search failed. Try again.";
    }
  }

  // renderArtists replaces the artist grid innerHTML with cards built from the
  // JSON array returned by /api/search. All user-supplied strings are escaped.
  function renderArtists(artists) {
    if (!Array.isArray(artists) || artists.length === 0) {
      artistsGrid.innerHTML = '<p class="empty-state">No artists found.</p>';
      return;
    }
    artistsGrid.innerHTML = artists.map(function (artist) {
      return (
        '<article class="artist-card">' +
          '<img src="' + escapeHTML(artist.image) + '" alt="' + escapeHTML(artist.name) + ' poster" loading="lazy">' +
          '<div class="artist-card-body">' +
            '<div class="title-row">' +
              '<h3>' + escapeHTML(artist.name) + '</h3>' +
              '<span class="badge">' + artist.memberCount + ' members</span>' +
            '</div>' +
            '<ul class="meta-list">' +
              '<li><strong>Created:</strong> ' + artist.creationDate + '</li>' +
              '<li><strong>First album:</strong> ' + escapeHTML(artist.firstAlbum) + '</li>' +
              '<li><strong>Locations:</strong> ' + artist.locationCount + '</li>' +
              '<li><strong>Concert dates:</strong> ' + artist.concertCount + '</li>' +
            '</ul>' +
            '<a class="button-link" href="/artist?id=' + artist.id + '">View details</a>' +
          '</div>' +
        '</article>'
      );
    }).join("");
  }

  // escapeHTML prevents XSS by replacing the five special HTML characters.
  function escapeHTML(value) {
    return String(value)
      .replaceAll("&", "&amp;")
      .replaceAll("<", "&lt;")
      .replaceAll(">", "&gt;")
      .replaceAll('"', "&quot;")
      .replaceAll("'", "&#39;");
  }
})();
