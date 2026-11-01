document.addEventListener("DOMContentLoaded",()=>{const root=document.querySelector("[data-workspace]");if(!root)return;const close=()=>root.classList.remove("sidebar-open");document.querySelector("[data-sidebar-open]")?.addEventListener("click",()=>root.classList.add("sidebar-open"));document.querySelector("[data-sidebar-close]")?.addEventListener("click",close);document.addEventListener("keydown",event=>{if(event.key==="Escape")close()});const dashboardLink=document.querySelector(".workspace-topbar .back-dashboard");if(dashboardLink)dashboardLink.innerHTML=`<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M3 13h8V3H3v10Zm0 8h8v-6H3v6Zm10 0h8V11h-8v10Zm0-18v6h8V3h-8Z"/></svg><span>Dashboard</span>`;document.querySelectorAll(".workspace-search>svg,.drive-search>svg,.trash-search>svg,.portal-search>svg").forEach(icon=>{icon.setAttribute("viewBox","0 0 24 24");icon.innerHTML=`<path d="M9.5 3a6.5 6.5 0 1 0 3.98 11.64L19.85 21 21 19.85l-6.36-6.37A6.5 6.5 0 0 0 9.5 3Zm0 2a4.5 4.5 0 1 1 0 9 4.5 4.5 0 0 1 0-9Z"/>`})});

document.addEventListener("DOMContentLoaded",()=>{const sidebar=document.querySelector(".workspace-sidebar");if(!sidebar)return;let form=document.querySelector('form[action="/logout"]');if(form){form.remove()}else{form=document.createElement("form");form.method="post";form.action="/logout";const token=document.querySelector('meta[name="csrf-token"]')?.content||"";const input=document.createElement("input");input.type="hidden";input.name="csrf_token";input.value=token;form.appendChild(input)}form.className="sidebar-logout-form";const button=form.querySelector("button")||document.createElement("button");button.type="submit";button.className="sidebar-logout-button";button.innerHTML=`<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M10 17v2H5V5h5v2h2V3H3v18h9v-4h-2Zm7-8-1.4 1.4 1.6 1.6H9v2h8.2l-1.6 1.6L17 17l4-4-4-4Z"/></svg><span>Keluar</span>`;if(!button.parentElement)form.appendChild(button);sidebar.appendChild(form)});

document.addEventListener("DOMContentLoaded",()=>{const navigation=document.querySelector(".workspace-nav");const logoutForm=document.querySelector(".sidebar-logout-form");if(navigation&&logoutForm)navigation.appendChild(logoutForm)});

document.addEventListener("DOMContentLoaded",()=>{
  const fileLinks=[...document.querySelectorAll('.drive-list article .node-main')].filter(link=>!link.querySelector('.node-icon.folder'));
  if(!fileLinks.length)return;
  const dialog=document.createElement("dialog");
  dialog.className="file-preview-dialog";
  dialog.innerHTML=`<div class="preview-dialog-head"><div><span class="preview-dialog-icon"><svg viewBox="0 0 24 24"><path d="M12 5c-5 0-9.27 3.11-11 7 1.73 3.89 6 7 11 7s9.27-3.11 11-7c-1.73-3.89-6-7-11-7Zm0 11.5a4.5 4.5 0 1 1 0-9 4.5 4.5 0 0 1 0 9Z"/></svg></span><div><strong data-preview-title>Preview file</strong><small data-preview-mime></small></div></div><div><a data-preview-download><svg viewBox="0 0 24 24"><path d="M19 9h-4V3H9v6H5l7 7 7-7ZM5 18v2h14v-2H5Z"/></svg>Unduh</a><button type="button" data-preview-close aria-label="Tutup preview"><svg viewBox="0 0 24 24"><path d="m6.4 5 5.6 5.6L17.6 5 19 6.4 13.4 12l5.6 5.6-1.4 1.4-5.6-5.6L6.4 19 5 17.6l5.6-5.6L5 6.4 6.4 5Z"/></svg></button></div></div><div class="preview-dialog-body" data-preview-body></div>`;
  document.body.appendChild(dialog);
  const title=dialog.querySelector("[data-preview-title]");
  const mimeLabel=dialog.querySelector("[data-preview-mime]");
  const body=dialog.querySelector("[data-preview-body]");
  const download=dialog.querySelector("[data-preview-download]");
  const close=()=>{dialog.close();body.replaceChildren()};
  dialog.querySelector("[data-preview-close]").addEventListener("click",close);
  dialog.addEventListener("click",event=>{if(event.target===dialog)close()});
  fileLinks.forEach(link=>link.addEventListener("click",event=>{
    event.preventDefault();
    const article=link.closest("article");
    const name=article?.dataset.nodeName||"File";
    const mime=link.querySelector("small")?.textContent.trim()||"application/octet-stream";
    const downloadURL=link.href;
    const previewURL=downloadURL.replace(/\/download(?:\?.*)?$/,"/preview");
    title.textContent=name;mimeLabel.textContent=mime;download.href=downloadURL;download.setAttribute("download",name);body.replaceChildren();
    let content;
    if(mime.startsWith("image/")){content=document.createElement("img");content.src=previewURL;content.alt=`Preview ${name}`}
    else if(mime==="application/pdf"){content=document.createElement("iframe");content.src=`${previewURL}#toolbar=1&navpanes=0`;content.title=`Preview ${name}`}
    else if(mime.startsWith("video/")){content=document.createElement("video");content.src=previewURL;content.controls=true;content.autoplay=false}
    else if(mime.startsWith("audio/")){content=document.createElement("audio");content.src=previewURL;content.controls=true}
    else if(mime.startsWith("text/")){content=document.createElement("iframe");content.src=previewURL;content.title=`Preview ${name}`}
    else{content=document.createElement("div");content.className="preview-unavailable";content.innerHTML=`<span><svg viewBox="0 0 24 24"><path d="M14 2H6a2 2 0 0 0-2 2v16h14a2 2 0 0 0 2-2V8l-6-6Zm1 7V5.5L18.5 9H15Z"/></svg></span><h3>Preview belum tersedia</h3><p>Format file ini belum dapat ditampilkan langsung. Gunakan tombol Unduh untuk membukanya.</p>`}
    body.appendChild(content);dialog.showModal()
  }))
});
