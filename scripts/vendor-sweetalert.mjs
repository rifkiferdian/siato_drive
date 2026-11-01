import { copyFile, mkdir } from "node:fs/promises";

const source = "node_modules/sweetalert2/dist";
const destination = "web/static/vendor/sweetalert2";
await mkdir(destination, { recursive: true });
await Promise.all([
  copyFile(`${source}/sweetalert2.min.js`, `${destination}/sweetalert2.min.js`),
  copyFile(`${source}/sweetalert2.min.css`, `${destination}/sweetalert2.min.css`),
]);
console.log("SweetAlert2 assets vendored locally.");
