document.addEventListener("DOMContentLoaded", () => {
  document.querySelectorAll(".portal-search>svg,.portal-toolbar button[type='submit'] svg").forEach((icon) => {
    Object.entries({ viewBox: "0 0 24 24", fill: "none", stroke: "currentColor", "stroke-width": "2", "stroke-linecap": "round" }).forEach(([key, value]) => icon.setAttribute(key, value));
    icon.innerHTML = '<circle cx="11" cy="11" r="7"></circle><path d="m20 20-4-4"></path>';
  });
  setupPublicBrowser();
});

function setupPublicBrowser() {
  const heading = document.querySelector(".portal-section-heading");
  const list = document.querySelector(".portal-files");
  const searchToolbar = document.querySelector(".portal-toolbar");
  const sourceSort = searchToolbar?.querySelector('select[name="sort"]');
  if (!heading || !list) return;
  if (searchToolbar) {
    searchToolbar.classList.add("portal-heading-toolbar");
    heading.appendChild(searchToolbar);
  }
  if (sourceSort) {
    sourceSort.hidden = true;
    sourceSort.closest(".portal-toolbar")?.classList.add("public-toolbar-browser-ready");
  }
  ["/static/css/my-drive.css", "/static/css/google-grid.css"].forEach((href) => {
    if (document.querySelector(`link[href="${href}"]`)) return;
    const link = document.createElement("link"); link.rel = "stylesheet"; link.href = href; document.head.appendChild(link);
  });
  const items = [...list.querySelectorAll(":scope>article")];
  list.className = "drive-list public-portal-drive-list";
  items.forEach(convertPublicItem);

  const toolbar = document.createElement("div");
  toolbar.className = "drive-toolbar public-drive-toolbar";
  toolbar.innerHTML = `<span>${items.length} item</span><div class="view-controls"><div class="view-segmented" role="group" aria-label="Pilih tampilan"><button type="button" data-public-view="details" aria-label="Tampilan list" title="List"><svg viewBox="0 0 24 24"><path d="M4 5h2v2H4V5Zm4 0h12v2H8V5ZM4 11h2v2H4v-2Zm4 0h12v2H8v-2ZM4 17h2v2H4v-2Zm4 0h12v2H8v-2Z"/></svg></button><button type="button" data-public-view="grid-md" aria-label="Tampilan grid" title="Grid"><svg viewBox="0 0 24 24"><path d="M4 4h6v6H4V4Zm10 0h6v6h-6V4ZM4 14h6v6H4v-6Zm10 0h6v6h-6v-6Z"/></svg></button></div></div><form><select name="sort" aria-label="Urutkan"><option value="name">Nama</option><option value="date">Terbaru</option><option value="size">Ukuran</option><option value="type">Jenis File</option></select></form>`;
  list.parentNode.insertBefore(toolbar, list);
  const breadcrumb = heading.querySelector(".portal-breadcrumb");
  if (breadcrumb) toolbar.parentNode.insertBefore(breadcrumb, toolbar);
  const sort = toolbar.querySelector("select");
  sort.value = new URLSearchParams(location.search).get("sort") || "name";
  sort.addEventListener("change", () => { const url = new URL(location.href); url.searchParams.set("sort", sort.value); location.href = url.toString(); });
  let view = localStorage.getItem("publicPortalView") === "grid-md" ? "grid-md" : "details";
  const applyView = () => {
    list.dataset.view = view;
    toolbar.querySelectorAll("[data-public-view]").forEach((button) => { const active = button.dataset.publicView === view; button.classList.toggle("checked", active); button.setAttribute("aria-pressed", String(active)); });
  };
  toolbar.querySelectorAll("[data-public-view]").forEach((button) => button.addEventListener("click", () => { view = button.dataset.publicView; localStorage.setItem("publicPortalView", view); applyView(); }));
  heading.classList.add("public-drive-heading");
  applyView();
}

function convertPublicItem(article) {
  const name = article.querySelector(".portal-file-info strong")?.textContent.trim() || "Item";
  const metadata = (article.querySelector(".portal-file-info small")?.textContent || "").split("•").map((value) => value.trim());
  const folderLink = article.querySelector(':scope>a[href*="folder="]');
  const previewLink = article.querySelector('.portal-file-actions a[href*="/preview"]');
  const downloadLink = article.querySelector('.portal-file-actions a[href*="/download"]');
  const folder = Boolean(folderLink);
  const extension = name.includes(".") ? name.split(".").pop().toLowerCase() : "";
  const bytes = Number.parseInt(metadata[1], 10) || 0;
  const size = folder ? "—" : `${(bytes / 1048576).toFixed(2)} MB`;
  const mime = publicMime(extension, folder);
  const mainURL = folder ? folderLink.href : previewLink.href;
  article.className = folder ? "is-folder" : "is-file";
  article.dataset.nodeName = name;
  article.innerHTML = `<a class="node-main" href="${safe(mainURL)}"${folder ? "" : ' target="_blank" rel="noopener"'}><span class="node-icon ${folder ? "folder" : "file"}"><svg viewBox="0 0 24 24"><path d="${folder ? "M10 4H2v16h20V6H12l-2-2Z" : "M14 2H6a2 2 0 0 0-2 2v16h14a2 2 0 0 0 2-2V8l-6-6Zm1 7V5.5L18.5 9H15Z"}"/></svg></span><div><strong>${safe(name)}</strong><small>${safe(mime)}</small></div></a><span class="node-size">${safe(size)}</span><time>${safe(metadata[2] || "-")}</time><div class="node-actions"></div>`;
  setupPublicMenu(article, { folder, folderLink, previewLink, downloadLink, name });
  if (!folder) article.appendChild(makePreview(name, extension, previewLink.href));
}

function setupPublicMenu(article, item) {
  const actions = article.querySelector(".node-actions");
  const more = document.createElement("button"); more.type = "button"; more.className = "node-more-button"; more.textContent = "⋮"; more.setAttribute("aria-label", `Menu ${item.name}`); more.setAttribute("aria-expanded", "false");
  const menu = document.createElement("div"); menu.className = "node-action-menu public-node-menu"; menu.hidden = true;
  menu.innerHTML = item.folder ? '<button type="button" data-open><svg viewBox="0 0 24 24"><path d="M10 4H2v16h20V6H12l-2-2Z"/></svg><span>Buka folder</span></button>' : '<button type="button" data-preview><svg viewBox="0 0 24 24"><path d="M12 5C6.5 5 2.5 10.2 2 11l-.6 1 .6 1c.5.8 4.5 6 10 6s9.5-5.2 10-6l.6-1-.6-1c-.5-.8-4.5-6-10-6Zm0 12c-3.7 0-6.8-3.1-8-5 1.2-1.9 4.3-5 8-5s6.8 3.1 8 5c-1.2 1.9-4.3 5-8 5Zm0-8a3 3 0 1 0 0 6 3 3 0 0 0 0-6Z"/></svg><span>Preview</span></button><button type="button" data-download><svg viewBox="0 0 24 24"><path d="M19 9h-4V3H9v6H5l7 7 7-7ZM5 18v2h14v-2H5Z"/></svg><span>Download</span></button>';
  more.addEventListener("click", (event) => { event.stopPropagation(); document.querySelectorAll(".public-node-menu:not([hidden])").forEach((open) => { if (open !== menu) open.hidden = true; }); menu.hidden = !menu.hidden; more.setAttribute("aria-expanded", String(!menu.hidden)); });
  menu.addEventListener("click", (event) => event.stopPropagation());
  menu.querySelector("[data-open]")?.addEventListener("click", () => { location.href = item.folderLink.href; });
  menu.querySelector("[data-preview]")?.addEventListener("click", () => window.open(item.previewLink.href, "_blank", "noopener"));
  menu.querySelector("[data-download]")?.addEventListener("click", () => { location.href = item.downloadLink.href; });
  document.addEventListener("click", () => { menu.hidden = true; more.setAttribute("aria-expanded", "false"); });
  actions.append(more, menu);
}

function makePreview(name, extension, url) {
  const preview = document.createElement("a"); preview.className = "grid-file-preview"; preview.href = url; preview.target = "_blank"; preview.rel = "noopener"; preview.setAttribute("aria-label", `Preview ${name}`);
  if (["jpg", "jpeg", "png", "gif", "webp"].includes(extension)) { preview.classList.add("image-preview"); const image = document.createElement("img"); image.src = url; image.alt = `Preview ${name}`; image.loading = "lazy"; preview.appendChild(image); }
  else if (extension === "pdf") { preview.classList.add("pdf-preview"); preview.innerHTML = `<iframe src="${safe(url)}#page=1&zoom=page-fit&toolbar=0&navpanes=0&scrollbar=0" title="Preview PDF" loading="lazy" tabindex="-1" scrolling="no"></iframe><span class="preview-type-badge pdf">PDF</span>`; }
  else { preview.classList.add("generic-preview"); preview.innerHTML = `<span class="generic-file-icon">${safe(extension.slice(0, 5).toUpperCase() || "FILE")}</span><small>Preview tidak tersedia</small>`; }
  return preview;
}

function publicMime(extension, folder) { if (folder) return "folder"; return ({ pdf: "application/pdf", jpg: "image/jpeg", jpeg: "image/jpeg", png: "image/png", gif: "image/gif", webp: "image/webp", mp4: "video/mp4", txt: "text/plain" })[extension] || `file/${extension || "unknown"}`; }
function safe(value) { const element = document.createElement("div"); element.textContent = value; return element.innerHTML.replaceAll('"', "&quot;"); }
