document.addEventListener("DOMContentLoaded",()=>{
  if(!document.querySelector('link[href="/static/css/public-embed-controls.css"]')){const style=document.createElement("link");style.rel="stylesheet";style.href="/static/css/public-embed-controls.css";document.head.appendChild(style)}
  const list=document.querySelector("[data-shared-list]");if(!list)return;
  document.querySelectorAll(".shared-size").forEach(element=>{const bytes=Number.parseInt(element.textContent,10)||0;element.textContent=`${(bytes/1048576).toFixed(2)} MB`});
  let view=localStorage.getItem("sharedFolderView")==="grid"?"grid":"list";
  const buttons=[...document.querySelectorAll("[data-view]")];
  const apply=()=>{list.classList.toggle("view-grid",view==="grid");buttons.forEach(button=>{const active=button.dataset.view===view;button.classList.toggle("active",active);button.setAttribute("aria-pressed",String(active))})};
  buttons.forEach(button=>button.addEventListener("click",()=>{view=button.dataset.view;localStorage.setItem("sharedFolderView",view);apply()}));
  document.querySelector("[data-sort]")?.addEventListener("change",event=>event.currentTarget.form.requestSubmit());
  document.addEventListener("click",event=>document.querySelectorAll(".shared-list details[open]").forEach(menu=>{if(!menu.contains(event.target))menu.removeAttribute("open")}));
  apply();
});
