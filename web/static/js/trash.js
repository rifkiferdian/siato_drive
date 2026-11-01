document.addEventListener("DOMContentLoaded", async () => {
  const dashboardLink = document.querySelector(".workspace-topbar .back-dashboard");
  if (dashboardLink) dashboardLink.innerHTML = `<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M3 13h8V3H3v10Zm0 8h8v-6H3v6Zm10 0h8V11h-8v10Zm0-18v6h8V3h-8Z"/></svg><span>Dashboard</span>`;
  let Swal;
  try { Swal = await loadSweetAlert(); } catch (error) { console.error(error); return; }

  const csrf = document.querySelector('meta[name="csrf-token"]')?.content || "";
  document.querySelector("[data-trash-filter]")?.addEventListener("change", (event) => event.currentTarget.form.requestSubmit());
  const post = async (url) => {
    const response = await fetch(url, { method: "POST", headers: { "X-CSRF-Token": csrf } });
    const data = await response.json().catch(() => ({}));
    if (!response.ok) throw new Error(data.error || "Operasi gagal");
    return data;
  };
  const fail = (message) => Swal.fire({ icon: "error", title: "Operasi gagal", text: message, confirmButtonText: "Tutup", confirmButtonColor: "#7c3aed" });
  const selectAll = document.querySelector("[data-select-all]");
  const restoreSelected = document.querySelector("[data-restore-selected]");
  const deleteSelected = document.querySelector("[data-delete-selected]");
  const selectionCount = document.querySelector("[data-selection-count]");
  const itemChecks = () => [...document.querySelectorAll("[data-item-check]")];
  const selectedItems = () => itemChecks().filter((checkbox) => checkbox.checked).map((checkbox) => checkbox.closest("[data-trash-item]"));
  const updateSelection = () => {
    const checks = itemChecks();
    const count = checks.filter((checkbox) => checkbox.checked).length;
    if (selectionCount) selectionCount.textContent = `${count} item dipilih`;
    if (restoreSelected) restoreSelected.disabled = count === 0;
    if (deleteSelected) deleteSelected.disabled = count === 0;
    if (selectAll) { selectAll.checked = checks.length > 0 && count === checks.length; selectAll.indeterminate = count > 0 && count < checks.length; }
  };
  selectAll?.addEventListener("change", () => { itemChecks().forEach((checkbox) => { checkbox.checked = selectAll.checked; }); updateSelection(); });
  itemChecks().forEach((checkbox) => checkbox.addEventListener("change", updateSelection));

  const removeItem = (item, reloadWhenEmpty = true) => {
    item.classList.add("is-removing");
    window.setTimeout(() => { item.remove(); updateSelection(); if (reloadWhenEmpty && !document.querySelector("[data-trash-item]")) location.reload(); }, 220);
  };

  restoreSelected?.addEventListener("click", async () => {
    const items = selectedItems();
    if (!items.length) return;
    const result = await Swal.fire({ icon: "question", title: `Pulihkan ${items.length} item?`, text: "Semua item yang dipilih akan dikembalikan ke lokasi sebelumnya.", showCancelButton: true, confirmButtonText: "Pulihkan yang Dipilih", cancelButtonText: "Batal", confirmButtonColor: "#7c3aed", reverseButtons: true });
    if (!result.isConfirmed) return;
    restoreSelected.disabled = true;
    deleteSelected.disabled = true;
    let restored = 0;
    const failures = [];
    for (const item of items) {
      try { await post(`/api/v1/trash/nodes/${item.dataset.nodeId}/restore`); removeItem(item, false); restored += 1; }
      catch (error) { failures.push(`${item.dataset.nodeName}: ${error.message}`); }
    }
    if (failures.length) await Swal.fire({ icon: "warning", title: `${restored} item berhasil dipulihkan`, text: `${failures.length} item gagal dipulihkan. ${failures[0]}`, confirmButtonColor: "#7c3aed" });
    else await Swal.fire({ icon: "success", title: `${restored} item berhasil dipulihkan`, timer: 1500, showConfirmButton: false });
    if (restored === items.length && itemChecks().length === items.length) location.reload(); else updateSelection();
  });

  deleteSelected?.addEventListener("click", async () => {
    const items = selectedItems();
    if (!items.length) return;
    const result = await Swal.fire({ icon: "warning", title: `Hapus ${items.length} item permanen?`, text: "Semua item yang dipilih beserta isinya tidak dapat dipulihkan kembali.", showCancelButton: true, confirmButtonText: "Hapus yang Dipilih", cancelButtonText: "Batal", confirmButtonColor: "#dc2626", reverseButtons: true, focusCancel: true });
    if (!result.isConfirmed) return;
    deleteSelected.disabled = true;
    let deleted = 0;
    const failures = [];
    for (const item of items) {
      try { await post(`/api/v1/trash/nodes/${item.dataset.nodeId}/permanent-delete`); removeItem(item, false); deleted += 1; }
      catch (error) { failures.push(`${item.dataset.nodeName}: ${error.message}`); }
    }
    if (failures.length) await Swal.fire({ icon: "warning", title: `${deleted} item berhasil dihapus`, text: `${failures.length} item gagal dihapus. ${failures[0]}`, confirmButtonColor: "#7c3aed" });
    else await Swal.fire({ icon: "success", title: `${deleted} item dihapus permanen`, timer: 1500, showConfirmButton: false });
    if (deleted === items.length && itemChecks().length === items.length) location.reload(); else updateSelection();
  });

  document.querySelectorAll("[data-restore]").forEach((button) => button.addEventListener("click", async () => {
    const item = button.closest("[data-trash-item]");
    const result = await Swal.fire({ icon: "question", title: "Pulihkan item?", html: `<strong>${escapeHTML(item.dataset.nodeName)}</strong> akan dikembalikan ke lokasi sebelumnya.`, showCancelButton: true, confirmButtonText: "Pulihkan", cancelButtonText: "Batal", confirmButtonColor: "#7c3aed", reverseButtons: true });
    if (!result.isConfirmed) return;
    try { await post(`/api/v1/trash/nodes/${item.dataset.nodeId}/restore`); removeItem(item); await Swal.fire({ toast: true, position: "top-end", icon: "success", title: "Item berhasil dipulihkan", timer: 1800, showConfirmButton: false }); } catch (error) { await fail(error.message); }
  }));

  document.querySelectorAll("[data-delete]").forEach((button) => button.addEventListener("click", async () => {
    const item = button.closest("[data-trash-item]");
    const result = await Swal.fire({ icon: "warning", title: "Hapus permanen?", html: `<strong>${escapeHTML(item.dataset.nodeName)}</strong> beserta isinya tidak dapat dipulihkan kembali.`, showCancelButton: true, confirmButtonText: "Hapus Permanen", cancelButtonText: "Batal", confirmButtonColor: "#dc2626", reverseButtons: true, focusCancel: true });
    if (!result.isConfirmed) return;
    try { await post(`/api/v1/trash/nodes/${item.dataset.nodeId}/permanent-delete`); removeItem(item); await Swal.fire({ toast: true, position: "top-end", icon: "success", title: "Item dihapus permanen", timer: 1800, showConfirmButton: false }); } catch (error) { await fail(error.message); }
  }));
});

function loadSweetAlert() {
  const stylesheet = document.createElement("link"); stylesheet.rel = "stylesheet"; stylesheet.href = "/static/vendor/sweetalert2/sweetalert2.min.css"; document.head.appendChild(stylesheet);
  if (window.Swal) return Promise.resolve(window.Swal);
  return new Promise((resolve, reject) => { const script = document.createElement("script"); script.src = "/static/vendor/sweetalert2/sweetalert2.min.js"; script.onload = () => resolve(window.Swal); script.onerror = () => reject(new Error("SweetAlert2 gagal dimuat")); document.head.appendChild(script); });
}

function escapeHTML(value) { const element = document.createElement("div"); element.textContent = value; return element.innerHTML; }
