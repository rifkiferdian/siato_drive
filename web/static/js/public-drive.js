document.addEventListener("DOMContentLoaded", async () => {
  const driveActions = document.querySelector(".drive-actions");
  if (driveActions) {
    const publicLink = document.createElement("a");
    publicLink.className = "public-page-link";
    publicLink.href = "/public";
    publicLink.target = "_blank";
    publicLink.rel = "noopener";
    publicLink.innerHTML = `<span class="public-link-icon"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="M12 2a10 10 0 1 0 0 20 10 10 0 0 0 0-20Zm6.92 6h-2.95a15.7 15.7 0 0 0-1.38-3.56A8.06 8.06 0 0 1 18.92 8ZM12 4c.83 1.2 1.48 2.55 1.85 4h-3.7A13.7 13.7 0 0 1 12 4ZM4.26 14a7.8 7.8 0 0 1 0-4h3.39a16.5 16.5 0 0 0 0 4H4.26Zm5.5 0a14.4 14.4 0 0 1 0-4h4.48a14.4 14.4 0 0 1 0 4H9.76ZM12 20a13.7 13.7 0 0 1-1.85-4h3.7A13.7 13.7 0 0 1 12 20Zm4.35-6a16.5 16.5 0 0 0 0-4h3.39a7.8 7.8 0 0 1 0 4h-3.39Z"/></svg></span><span>Buka Portal Publik</span><svg class="public-link-arrow" viewBox="0 0 24 24" aria-hidden="true"><path d="M14 3h7v7h-2V6.41l-9.29 9.3-1.42-1.42 9.3-9.29H14V3Z"/></svg>`;
    driveActions.prepend(publicLink);
  }
  let Swal;
  try { Swal = await loadSweetAlert(); } catch (error) { console.error(error); return; }
  const csrf = document.querySelector('meta[name="csrf-token"]')?.content || "";
  const parent = new URLSearchParams(location.search).get("folder") || "";
  setupDriveView(Swal);
  document.querySelector("[data-sort-select]")?.addEventListener("change", (event) => event.currentTarget.form.requestSubmit());
  const post = async (url, body) => {
    const response = await fetch(url, { method: "POST", headers: { "X-CSRF-Token": csrf }, body });
    const data = await response.json().catch(() => ({}));
    if (!response.ok) throw new Error(data.error || "Operasi gagal");
    return data;
  };
  const showError = (message) => Swal.fire({ icon: "error", title: "Operasi gagal", text: message, confirmButtonText: "Tutup", confirmButtonColor: "#7c3aed" });
  const showSuccess = (title, text = "") => Swal.fire({ icon: "success", title, text, timer: 1300, timerProgressBar: true, showConfirmButton: false });
  const showToast = (title, icon = "success") => Swal.fire({ toast: true, position: "top-end", icon, title, timer: 1800, timerProgressBar: true, showConfirmButton: false });

  document.querySelector("[data-open-folder]")?.addEventListener("click", async () => {
    const result = await Swal.fire({ title: "Buat folder baru", text: "Folder akan dibuat di lokasi yang sedang dibuka.", input: "text", inputLabel: "Nama folder", inputPlaceholder: "Contoh: Dokumen Akademik", showCancelButton: true, confirmButtonText: "Buat Folder", cancelButtonText: "Batal", confirmButtonColor: "#7c3aed", inputValidator: (value) => !value.trim() ? "Nama folder wajib diisi" : undefined });
    if (!result.isConfirmed) return;
    const body = new FormData(); body.set("name", result.value.trim()); body.set("parent_id", parent);
    try { await post("/api/v1/public-drive/folders", body); await showSuccess("Folder berhasil dibuat"); location.reload(); } catch (error) { await showError(error.message); }
  });

  const input = document.querySelector("[data-upload-input]");
  input?.addEventListener("change", async () => {
    const files = [...input.files]; if (!files.length) return;
    const box = document.querySelector("[data-upload-progress]"); const bar = document.querySelector("[data-upload-bar]"); const label = document.querySelector("[data-upload-label]"); const percent = document.querySelector("[data-upload-percent]"); box.hidden = false;
    for (let index = 0; index < files.length; index += 1) {
      label.textContent = `Mengunggah ${files[index].name} (${index + 1}/${files.length})`; bar.value = Math.round(index / files.length * 100); percent.textContent = `${bar.value}%`;
      const body = new FormData(); body.set("parent_id", parent); body.set("file", files[index]);
      try { await post("/api/v1/public-drive/upload", body); } catch (error) { box.hidden = true; input.value = ""; await showError(error.message); return; }
    }
    bar.value = 100; percent.textContent = "100%"; await showSuccess("Upload selesai", `${files.length} file berhasil diunggah.`); location.reload();
  });

  document.querySelectorAll("[data-rename]").forEach((button) => button.addEventListener("click", () => {
    const item = button.closest("[data-node-id]");
    const nameElement = item.querySelector(".node-main strong");
    if (!nameElement || item.querySelector(".inline-rename")) return;
    const originalName = item.dataset.nodeName;
    const input = document.createElement("input");
    input.type = "text"; input.className = "inline-rename"; input.value = originalName; input.maxLength = 255; input.setAttribute("aria-label", "Nama baru");
    nameElement.replaceWith(input); input.focus(); input.select();
    let finished = false;
    const restore = (name = originalName) => { const replacement = document.createElement("strong"); replacement.textContent = name; input.replaceWith(replacement); };
    const save = async () => {
      if (finished) return; finished = true; const name = input.value.trim();
      if (!name || name === originalName) { restore(); return; }
      input.disabled = true; const body = new FormData(); body.set("name", name);
      try { await post(`/api/v1/public-drive/nodes/${item.dataset.nodeId}/rename`, body); item.dataset.nodeName = name; item.querySelector(".node-more-button")?.setAttribute("aria-label", `Menu ${name}`); restore(name); await showToast("Nama berhasil diubah"); } catch (error) { restore(); await showError(error.message); }
    };
    input.addEventListener("click", (event) => { event.preventDefault(); event.stopPropagation(); });
    input.addEventListener("keydown", (event) => { if (event.key === "Enter") { event.preventDefault(); save(); } if (event.key === "Escape") { event.preventDefault(); finished = true; restore(); } });
    input.addEventListener("blur", save);
  }));

  document.querySelectorAll("[data-trash]").forEach((button) => button.addEventListener("click", async () => {
    const item = button.closest("[data-node-id]");
    const result = await Swal.fire({ icon: "warning", title: "Pindahkan ke Sampah?", html: `<strong>${escapeHTML(item.dataset.nodeName)}</strong> dapat dipulihkan kembali dari Sampah.`, showCancelButton: true, confirmButtonText: "Ya, pindahkan", cancelButtonText: "Batal", confirmButtonColor: "#dc2626", reverseButtons: true });
    if (!result.isConfirmed) return;
    try { await post(`/api/v1/public-drive/nodes/${item.dataset.nodeId}/trash`, new FormData()); item.remove(); await showSuccess("Dipindahkan ke Sampah"); } catch (error) { await showError(error.message); }
  }));
});

function loadSweetAlert() {
  const stylesheet = document.createElement("link"); stylesheet.rel = "stylesheet"; stylesheet.href = "/static/vendor/sweetalert2/sweetalert2.min.css"; document.head.appendChild(stylesheet);
  if (window.Swal) return Promise.resolve(window.Swal);
  return new Promise((resolve, reject) => { const script = document.createElement("script"); script.src = "/static/vendor/sweetalert2/sweetalert2.min.js"; script.onload = () => resolve(window.Swal); script.onerror = () => reject(new Error("SweetAlert2 gagal dimuat")); document.head.appendChild(script); });
}

function escapeHTML(value) { const element = document.createElement("div"); element.textContent = value; return element.innerHTML; }

function setupDriveView(Swal) {
  if (!document.querySelector('link[href="/static/css/google-grid.css"]')) { const style = document.createElement("link"); style.rel = "stylesheet"; style.href = "/static/css/google-grid.css"; document.head.appendChild(style); }
  const toolbar = document.querySelector(".drive-toolbar");
  const list = document.querySelector(".drive-list");
  if (!toolbar || !list) return;
  setupGridPreviews(list);
  const sortForm = toolbar.querySelector("form");
  const controls = document.createElement("div");
  controls.className = "view-controls";
  controls.innerHTML = `<button type="button" class="view-trigger" aria-haspopup="menu" aria-expanded="false"><span class="view-trigger-icon">☷</span><span>View</span><span>⌄</span></button><div class="view-menu" role="menu" hidden>
    <button type="button" data-view="grid-xl"><span>▣</span>Extra large icons<i></i></button>
    <button type="button" data-view="grid-lg"><span>▣</span>Large icons<i></i></button>
    <button type="button" data-view="grid-md"><span>▦</span>Medium icons<i></i></button>
    <button type="button" data-view="grid-sm"><span>⁙</span>Small icons<i></i></button>
    <button type="button" data-view="list"><span>☷</span>List<i></i></button>
    <button type="button" data-view="details"><span>☷</span>Details<i></i></button>
    <button type="button" data-view="tiles"><span>☷</span>Tiles<i></i></button>
    <button type="button" data-view="content"><span>☷</span>Content<i></i></button>
    <hr><button type="button" data-pane="details"><span>▤</span>Details pane<i></i></button>
    <button type="button" data-pane="preview"><span>▯</span>Preview pane<i></i></button>
  </div>`;
  toolbar.insertBefore(controls, sortForm);
  const trigger = controls.querySelector(".view-trigger");
  const menu = controls.querySelector(".view-menu");
  const browser = document.createElement("div"); browser.className = "drive-browser"; list.parentNode.insertBefore(browser, list); browser.appendChild(list);
  const pane = document.createElement("aside"); pane.className = "drive-side-pane"; pane.hidden = true; browser.appendChild(pane);
  let currentView = localStorage.getItem("publicDriveView") || "details";
  let paneMode = localStorage.getItem("publicDrivePane") || "none";
  let selected = null;
  const applyView = () => { list.dataset.view = currentView; controls.querySelectorAll("[data-view]").forEach((button) => button.classList.toggle("checked", button.dataset.view === currentView)); };
  const renderPane = () => {
    pane.hidden = paneMode === "none"; browser.classList.toggle("has-pane", paneMode !== "none");
    controls.querySelectorAll("[data-pane]").forEach((button) => button.classList.toggle("checked", button.dataset.pane === paneMode));
    if (paneMode === "none") return;
    if (!selected) { pane.innerHTML = `<div class="pane-empty"><span>▤</span><strong>${paneMode === "preview" ? "Preview" : "Details"}</strong><p>Pilih file atau folder untuk melihat informasi.</p></div>`; return; }
    const name = selected.dataset.nodeName; const id = selected.dataset.nodeId; const mime = selected.querySelector(".node-main small")?.textContent.trim() || "folder"; const size = selected.querySelector(".node-size")?.textContent.trim() || "0 byte"; const date = selected.querySelector("time")?.textContent.trim() || "-"; const folder = selected.querySelector(".node-icon")?.classList.contains("folder");
    let preview = `<div class="pane-file-symbol">${folder ? "▰" : "▤"}</div>`;
    if (paneMode === "preview" && !folder) { const url = `/api/v1/public-drive/nodes/${id}/preview`; if (mime.startsWith("image/")) preview = `<img src="${url}" alt="Preview ${escapeHTML(name)}">`; else if (mime === "application/pdf" || mime.startsWith("text/plain")) preview = `<iframe src="${url}" title="Preview ${escapeHTML(name)}"></iframe>`; else if (mime === "video/mp4") preview = `<video src="${url}" controls></video>`; else preview = `<div class="pane-unavailable">Preview tidak tersedia untuk tipe file ini.</div>`; }
    pane.innerHTML = `<header><strong>${paneMode === "preview" ? "Preview" : "Details"}</strong><button type="button" aria-label="Tutup pane">×</button></header><div class="pane-preview">${preview}</div><h3>${escapeHTML(name)}</h3><dl><dt>Type</dt><dd>${escapeHTML(mime)}</dd><dt>Size</dt><dd>${escapeHTML(size)}</dd><dt>Modified</dt><dd>${escapeHTML(date)}</dd></dl>`;
    pane.querySelector("header button").addEventListener("click", () => { paneMode = "none"; localStorage.setItem("publicDrivePane", paneMode); renderPane(); });
  };
  trigger.addEventListener("click", (event) => { event.stopPropagation(); menu.hidden = !menu.hidden; trigger.setAttribute("aria-expanded", String(!menu.hidden)); });
  document.addEventListener("click", () => { menu.hidden = true; trigger.setAttribute("aria-expanded", "false"); }); menu.addEventListener("click", (event) => event.stopPropagation());
  controls.querySelectorAll("[data-view]").forEach((button) => button.addEventListener("click", () => { currentView = button.dataset.view; localStorage.setItem("publicDriveView", currentView); applyView(); menu.hidden = true; }));
  controls.querySelectorAll("[data-pane]").forEach((button) => button.addEventListener("click", () => { paneMode = paneMode === button.dataset.pane ? "none" : button.dataset.pane; localStorage.setItem("publicDrivePane", paneMode); renderPane(); }));
  list.querySelectorAll("article").forEach((article) => article.addEventListener("click", (event) => { if (event.target.closest(".node-actions")) return; selected?.classList.remove("selected"); selected = article; selected.classList.add("selected"); renderPane(); }));
  paneMode = "none";
  localStorage.setItem("publicDrivePane", "none");
  renderPane();
  currentView = currentView.startsWith("grid") ? "grid-md" : "details";
  controls.innerHTML = `<div class="view-segmented" role="group" aria-label="Pilih tampilan">
    <button type="button" data-view="details" aria-label="Tampilan list" title="List"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="M4 5h2v2H4V5Zm4 0h12v2H8V5ZM4 11h2v2H4v-2Zm4 0h12v2H8v-2ZM4 17h2v2H4v-2Zm4 0h12v2H8v-2Z"/></svg></button>
    <button type="button" data-view="grid-md" aria-label="Tampilan grid" title="Grid"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="M4 4h6v6H4V4Zm10 0h6v6h-6V4ZM4 14h6v6H4v-6Zm10 0h6v6h-6v-6Z"/></svg></button>
  </div>`;
  controls.querySelectorAll("[data-view]").forEach((button) => button.addEventListener("click", () => { currentView = button.dataset.view; localStorage.setItem("publicDriveView", currentView); applyView(); }));
  list.querySelectorAll("article").forEach((article) => {
    const actions = article.querySelector(".node-actions");
    const renameButton = actions?.querySelector("[data-rename]");
    const trashButton = actions?.querySelector("[data-trash]");
    if (!actions || !renameButton || !trashButton) return;
    renameButton.classList.add("action-source");
    trashButton.classList.add("action-source");
    const moreButton = document.createElement("button");
    moreButton.type = "button";
    moreButton.className = "node-more-button";
    moreButton.setAttribute("aria-label", `Menu ${article.dataset.nodeName}`);
    moreButton.setAttribute("aria-haspopup", "menu");
    moreButton.setAttribute("aria-expanded", "false");
    moreButton.title = "Tindakan lainnya";
    moreButton.textContent = "⋮";
    const actionMenu = document.createElement("div");
    actionMenu.className = "node-action-menu";
    actionMenu.setAttribute("role", "menu");
    actionMenu.hidden = true;
    actionMenu.innerHTML = `<button type="button" role="menuitem" data-menu-rename><svg viewBox="0 0 24 24" aria-hidden="true"><path d="m3 17.25-.75 4.5 4.5-.75L19.81 7.94l-3.75-3.75L3 17.25Z"/></svg><span>Ubah nama</span></button><button type="button" role="menuitem" class="danger" data-menu-trash><svg viewBox="0 0 24 24" aria-hidden="true"><path d="M6 19a2 2 0 0 0 2 2h8a2 2 0 0 0 2-2V7H6v12ZM15.5 4l-1-1h-5l-1 1H5v2h14V4h-3.5Z"/></svg><span>Pindahkan ke Sampah</span></button>`;
    moreButton.addEventListener("click", (event) => {
      event.stopPropagation();
      document.querySelectorAll(".node-action-menu:not([hidden])").forEach((openMenu) => { if (openMenu !== actionMenu) { openMenu.hidden = true; openMenu.previousElementSibling?.setAttribute("aria-expanded", "false"); } });
      actionMenu.hidden = !actionMenu.hidden;
      moreButton.setAttribute("aria-expanded", String(!actionMenu.hidden));
    });
    actionMenu.addEventListener("click", (event) => event.stopPropagation());
    actionMenu.querySelector("[data-menu-rename]").addEventListener("click", () => { actionMenu.hidden = true; moreButton.setAttribute("aria-expanded", "false"); renameButton.click(); });
    actionMenu.querySelector("[data-menu-trash]").addEventListener("click", () => { actionMenu.hidden = true; moreButton.setAttribute("aria-expanded", "false"); trashButton.click(); });
    document.addEventListener("click", () => { actionMenu.hidden = true; moreButton.setAttribute("aria-expanded", "false"); });
    actions.append(moreButton, actionMenu);
  });
  applyView();
}

function setupGridPreviews(list) {
  list.querySelectorAll("article[data-node-id]").forEach((article) => {
    const icon = article.querySelector(".node-icon");
    if (icon?.classList.contains("folder")) { article.classList.add("is-folder"); return; }
    article.classList.add("is-file");
    const mime = article.querySelector(".node-main small")?.textContent.trim() || "application/octet-stream";
    const name = article.dataset.nodeName;
    const url = `/api/v1/public-drive/nodes/${article.dataset.nodeId}/preview`;
    const preview = document.createElement("div"); preview.className = "grid-file-preview";
    if (mime.startsWith("image/")) { preview.classList.add("image-preview"); preview.innerHTML = `<img src="${url}" alt="Preview ${escapeHTML(name)}" loading="lazy">`; }
    else if (mime === "application/pdf") { preview.classList.add("pdf-preview"); preview.innerHTML = `<iframe src="${url}#page=1&zoom=page-fit&toolbar=0&navpanes=0&scrollbar=0" title="Preview ${escapeHTML(name)}" loading="lazy" tabindex="-1" scrolling="no"></iframe><span class="preview-type-badge pdf">PDF</span>`; }
    else if (mime === "video/mp4") { preview.classList.add("video-preview"); preview.innerHTML = `<video src="${url}" preload="metadata" muted></video><span class="preview-play">▶</span>`; }
    else { preview.classList.add("generic-preview"); const extension = name.includes(".") ? name.split(".").pop().slice(0, 5).toUpperCase() : "FILE"; preview.innerHTML = `<span class="generic-file-icon">${escapeHTML(extension)}</span><small>Preview tidak tersedia</small>`; }
    if (!preview.classList.contains("generic-preview")) { preview.tabIndex = 0; preview.setAttribute("role", "button"); preview.setAttribute("aria-label", `Buka preview ${name}`); const open = () => window.open(url, "_blank", "noopener"); preview.addEventListener("click", open); preview.addEventListener("keydown", (event) => { if (event.key === "Enter" || event.key === " ") { event.preventDefault(); open(); } }); }
    article.appendChild(preview);
  });
}
