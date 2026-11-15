<template>
  <section class="card mx-auto my-6 max-w-5xl p-5">
    <h2 class="section-title">Неоплаченные счета</h2>
    <div
      v-for="i in state.invoices.filter((x) => x.status === 'open')"
      :key="i.id"
      class="border-b border-line py-3"
    >
      <p>{{ childById(i.childId)?.name }} · {{ i.title }} · {{ i.amount }} ₽</p>
      <button class="btn-quiet" :disabled="busy" @click="pay(i.id, 'cash')">
        Внести наличные</button
      ><button
        class="btn-quiet"
        :disabled="busy"
        @click="pay(i.id, 'transfer')"
      >
        Внести перевод
      </button>
    </div>
  </section>
</template>
<script setup lang="ts">
import { ref } from "vue";
import { api, key } from "../shared/api";
import { useDemo, refresh } from "../shared/use-demo";
const { state, childById } = useDemo();
const busy = ref(false);
const keys = new Map<string, string>();
async function pay(id: string, method: string) {
  if (!confirm("Подтвердить получение оплаты?")) return;
  busy.value = true;
  try {
    const k = keys.get(id) || key();
    keys.set(id, k);
    await api("/master/payments", "POST", { invoice_id: id, method }, k);
    await refresh();
  } finally {
    busy.value = false;
  }
}
</script>
