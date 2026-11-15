import { createApp } from "vue";
import App from "./App.vue";
import "./app.css";
import "@fortawesome/fontawesome-free/css/all.min.css";
import { report } from "./shared/api";
const app = createApp(App);
app.config.errorHandler = report;
app.mount("#app");
window.addEventListener("unhandledrejection", (event) => {
  event.preventDefault();
  report(event.reason);
});
