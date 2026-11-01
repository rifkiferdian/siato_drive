document.addEventListener("DOMContentLoaded", () => {
  const toggle = document.querySelector("[data-password-toggle]");
  const password = document.querySelector("#password");
  if (!toggle || !password) return;

  toggle.addEventListener("click", () => {
    const showing = password.type === "text";
    password.type = showing ? "password" : "text";
    toggle.setAttribute("aria-pressed", String(!showing));
    toggle.setAttribute("aria-label", showing ? "Tampilkan kata sandi" : "Sembunyikan kata sandi");
    password.focus();
  });
});
