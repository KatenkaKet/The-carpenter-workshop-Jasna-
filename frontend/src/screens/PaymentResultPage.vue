<template>
  <main class="mx-auto my-12 max-w-lg card p-6">
    <h1 class="page-title">Статус оплаты</h1>
    <p class="my-5">{{ label }}</p>
    <button class="btn-primary" @click="check">Проверить статус</button
    ><a class="block mt-5" href="/yasna/parent-payments">История оплат</a
    ><a class="block mt-3" href="/yasna/parent">Записаться на занятие</a>
  </main>
</template>
<script setup lang="ts">
import { ref, onMounted } from "vue";
import { api } from "../shared/api";
import { refresh } from "../shared/use-demo";
const label = ref("Проверяем подтверждение…");
async function check() {
  const id = sessionStorage.getItem("jasna.payment");
  if (!id) {
    label.value = "Откройте историю оплат для проверки.";
    return;
  }
  const p = await api("/payments/" + id);
  label.value =
    p.status === "succeeded"
      ? (p.method === "test" ? "Тестовая оплата подтверждена. Деньги не списывались. Абонемент доступен в кабинете." : "Оплата подтверждена. Абонемент доступен в кабинете.")
      : p.status === "canceled"
        ? "Платёж отменён."
        : "Ожидаем подтверждение платёжного сервиса.";
  await refresh();
}
onMounted(check);
</script>
