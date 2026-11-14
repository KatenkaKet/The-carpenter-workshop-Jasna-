// @shared
import { onUnmounted, ref } from "vue";

export function useToast() {
  const message = ref("");
  let timer: ReturnType<typeof setTimeout> | undefined;

  function show(text: string) {
    message.value = text;
    if (timer) clearTimeout(timer);
    timer = setTimeout(() => {
      message.value = "";
    }, 2800);
  }

  onUnmounted(() => {
    if (timer) clearTimeout(timer);
  });

  return { message, show };
}
