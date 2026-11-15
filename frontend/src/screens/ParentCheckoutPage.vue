<template>
  <main class="mx-auto my-10 max-w-lg card p-6">
    <a href="/yasna/parent-tariffs">← Тарифы</a>
    <h1 class="page-title mt-5">Оплата абонемента</h1>
    <p class="my-4">{{ plan?.title }} · {{ plan?.price }} ₽</p>
    <p>
      Оплата подтверждается платёжным сервисом. После оплаты вернитесь в кабинет
      и проверьте статус.
    </p>
    <button
      class="btn-primary mt-5"
      :disabled="busy || !plan || !child"
      @click="pay"
    >
      {{ busy ? "Открываем оплату…" : "Перейти к оплате через СБП" }}
    </button>
  </main>
</template>
<script setup lang="ts">
import { computed, ref } from "vue";
import { api, key } from "../shared/api";
import { useDemo } from "../shared/use-demo";
import { DEMO_TODAY } from "../shared/demo";
const props = defineProps<{
  tariff?: string;
  child?: string;
  lesson?: string;
}>();
const { state } = useDemo();
const busy = ref(false);
const plan = computed(() =>
  state.settings.tariffs.find((p) => p.id === props.tariff),
);
const child = computed(() => props.child || state.children[0]?.id);
let invoiceId = "";
const invoiceKey = key(),
  paymentKey = key();
async function pay() {
  busy.value = true;
  try {
    if (!invoiceId) {
      const inv = await api(
        "/invoices",
        "POST",
        {
          child_id: child.value,
          kind: "subscription",
          plan_id: plan.value?.id,
          valid_from: DEMO_TODAY,
        },
        invoiceKey,
      );
      invoiceId = inv.id;
    }
    const payment = await api(
      "/payments/intent",
      "POST",
      { invoice_id: invoiceId, method: "sbp" },
      paymentKey,
    );
    sessionStorage.setItem("jasna.payment", payment.id);
    if (!payment.payment_url)
      throw new Error("Платёж обрабатывается. Проверьте его в истории оплат.");
    location.assign(payment.payment_url);
  } finally {
    busy.value = false;
  }
}
</script>
