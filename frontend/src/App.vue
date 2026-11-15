<template>
  <div
    v-if="apiError"
    class="fixed left-4 right-4 top-3 z-[100] rounded-xl bg-clay-soft p-4 text-clay shadow-lg"
    role="alert"
  >
    {{ apiError }}
    <button class="float-right" @click="apiError = ''" aria-label="Закрыть">
      ×
    </button>
  </div>
  <div v-if="loading" class="p-10 text-center">
    Загружаем данные мастерской…
  </div>
  <main
    v-else-if="
      !user ||
      (parentRoute && user.role !== 'parent') ||
      (!parentRoute && user.role !== 'master')
    "
    class="mx-auto my-16 max-w-md card p-8"
  >
    <h1 class="page-title">Ясна</h1>
    <p v-if="parentRoute" class="my-4">
      Откройте персональную ссылку, которую выдал мастер.
    </p>
    <form v-else-if="!user" @submit.prevent="login" class="space-y-4 mt-5">
      <label class="block"
        >Логин<input
          class="input w-full"
          v-model="loginName"
          autocomplete="username"
          required
      /></label>
      <label class="block"
        >Пароль<input
          class="input w-full"
          v-model="password"
          type="password"
          autocomplete="current-password"
          required
      /></label>
      <button class="btn-primary" :disabled="busy">Войти мастером</button>
    </form>
    <button v-if="user && !parentRoute" class="btn-primary mt-4" @click="logout">
      Выйти из текущего кабинета
    </button>
    <a href="/yasna" class="block mt-6">Кабинет мастера</a>
  </main>
  <template v-else
    ><div class="mx-auto flex max-w-7xl justify-end gap-4 px-5 pt-2 text-xs">
      <button @click="reload">Обновить данные</button
      ><button v-if="!parentRoute" @click="logout">Выйти</button>
    </div>
    <component :is="page" v-bind="props" :key="routeLocation"
  /></template>
</template>
<script setup lang="ts">
import {
  ref,
  computed,
  onMounted,
  onUnmounted,
  defineAsyncComponent,
} from "vue";
import {
  api,
  session,
  user,
  logout,
  apiError,
  busy,
  report,
} from "./shared/api";
import { refresh } from "./shared/use-demo";
const routeLocation = ref(location.pathname + location.search);
const loading = ref(true),
  loginName = ref(""),
  password = ref("");
const parentRoute = computed(
  () =>
    routeLocation.value.includes("parent") ||
    ["/access", "/payment-result"].includes(routeLocation.value.split("?")[0]),
);
const routes: Record<string, string> = {
  "/": "SchedulePage",
  "/yasna": "SchedulePage",
  "/yasna/day": "DayPage",
  "/yasna/journal": "JournalPage",
  "/yasna/clients": "ClientsPage",
  "/yasna/subscriptions": "SubscriptionsPage",
  "/yasna/finance": "FinancePage",
  "/yasna/requests": "RequestsPage",
  "/yasna/parent": "ParentSchedulePage",
  "/access": "ParentSchedulePage",
  "/yasna/parent-day": "ParentDayPage",
  "/yasna/parent-history": "ParentHistoryPage",
  "/yasna/parent-payments": "ParentPaymentsPage",
  "/yasna/parent-tariffs": "ParentTariffsPage",
  "/yasna/parent-checkout": "ParentCheckoutPage",
  "/payment-result": "PaymentResultPage",
};
const screens = import.meta.glob("./screens/*.vue");
const components = Object.fromEntries(
  Object.entries(screens).map(([path, load]) => [
    path,
    defineAsyncComponent(load as any),
  ]),
);
const page = computed(
  () =>
    components[
      "./screens/" +
        (routes[routeLocation.value.split("?")[0]] || "SchedulePage") +
        ".vue"
    ],
);
const props = computed(() =>
  Object.fromEntries(
    new URLSearchParams(routeLocation.value.split("?")[1] || ""),
  ),
);
function syncRoute() {
  routeLocation.value = location.pathname + location.search;
  apiError.value = "";
}
function navigate(event: MouseEvent) {
  if (
    event.defaultPrevented ||
    event.button !== 0 ||
    event.ctrlKey ||
    event.metaKey ||
    event.shiftKey ||
    event.altKey
  )
    return;
  const anchor = (event.target as Element)?.closest("a");
  if (!anchor || anchor.target || anchor.hasAttribute("download")) return;
  const url = new URL(anchor.href);
  if (
    url.origin !== location.origin ||
    !routes[url.pathname] ||
    url.pathname === "/access" ||
    url.hash
  )
    return;
  event.preventDefault();
  history.pushState(null, "", url.pathname + url.search);
  syncRoute();
  window.scrollTo(0, 0);
}
let syncTimer:ReturnType<typeof setInterval>;
onMounted(() => {
  syncTimer=setInterval(()=>{if(document.visibilityState==='visible'&&!document.querySelector('[role="dialog"]')&&document.activeElement?.tagName!=='INPUT'&&document.activeElement?.tagName!=='SELECT')void reload()},60000);
  window.addEventListener("focus",reload);
  document.addEventListener("click", navigate);
  window.addEventListener("popstate", syncRoute);
});
onUnmounted(() => {
  clearInterval(syncTimer);window.removeEventListener("focus",reload);
  document.removeEventListener("click", navigate);
  window.removeEventListener("popstate", syncRoute);
});

let refreshing=false;
async function reload() {
  if(refreshing||!user.value)return;refreshing=true;
  try {
    await refresh();
  } catch (e) {
    report(e);
  } finally {refreshing=false;}
}
async function login() {
  busy.value = true;
  try {
    await api("/auth/master/login", "POST", {
      login: loginName.value,
      password: password.value,
    });
    password.value = "";
    await session();
    await refresh();
  } catch (e) {
    report(e);
  } finally {
    busy.value = false;
  }
}
onMounted(async () => {
  try {
    const token = new URLSearchParams(location.hash.slice(1)).get("token");
    if (token) {
      history.replaceState(null, "", location.pathname + location.search);
      await api("/auth/logout", "POST");
      await api("/auth/access-link/exchange", "POST", { token });
    }
    await session();
    await refresh();
  } catch (e) {
    if (tokenPresentError(e)) report(e);
  } finally {
    loading.value = false;
  }
});
function tokenPresentError(e: unknown) {
  return (e as Error).message !== "Войдите в кабинет.";
}
</script>
