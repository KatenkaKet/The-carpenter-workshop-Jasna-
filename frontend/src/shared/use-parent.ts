import { computed, ref } from "vue";
import { user, logout } from "./api";
import { useDemo } from "./use-demo";
export function useParent() {
  const { state, ready } = useDemo();
  const familyId = computed(() =>
    user.value?.role === "parent" ? user.value.family_id : "",
  );
  const family = computed(() =>
    state.families.find((f) => f.id === familyId.value),
  );
  return {
    familyId,
    family,
    children: computed(() =>
      state.children.filter((c) => c.familyId === familyId.value),
    ),
    ready,
    error: ref(""),
    loginAs: () => {},
    logout,
  };
}
