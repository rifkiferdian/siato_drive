document.addEventListener("DOMContentLoaded", () => {
  document.querySelectorAll(".portal-search > svg, .portal-toolbar button[type='submit'] svg").forEach((icon) => {
    icon.setAttribute("viewBox", "0 0 24 24");
    icon.setAttribute("fill", "none");
    icon.setAttribute("stroke", "currentColor");
    icon.setAttribute("stroke-width", "2");
    icon.setAttribute("stroke-linecap", "round");
    icon.setAttribute("stroke-linejoin", "round");
    icon.innerHTML = '<circle cx="11" cy="11" r="7"></circle><path d="m20 20-4-4"></path>';
  });
  const toMB = (bytes) => `${(Number(bytes) / (1024 * 1024)).toFixed(2)} MB`;
  document.querySelectorAll(".portal-file-info small").forEach((element) => {
    element.childNodes.forEach((node) => {
      if (node.nodeType !== Node.TEXT_NODE) return;
      node.textContent = node.textContent.replace(/\b(\d+)\s+byte\b/i, (_, bytes) => toMB(bytes));
    });
  });
});
