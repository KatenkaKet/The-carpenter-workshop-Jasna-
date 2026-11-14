// @shared
import { computed, onMounted, ref } from "vue";
import { useDemo } from "./use-demo";

const SESSION_KEY = "yasna.parent.session.v1";

function extractToken(raw: string): string {
  const match = raw.match(/[#?&]token=([^&]+)/);
  if (match?.[1]) return decodeURIComponent(match[1]);
  return raw.trim().replace(/^#?token=/, "");
}

export function useParent() {
  const { state, familyById, familyByToken } = useDemo();
  const familyId = ref("");
  const ready = ref(false);
  const error = ref("");

  onMounted(() => {
    if (typeof window === "undefined") return;

    const hash = window.location.hash;
    if (hash.includes("token=")) {
      const token = extractToken(hash);
      window.history.replaceState(
        null,
        "",
        window.location.pathname + window.location.search,
      );
      const family = familyByToken(token);
      if (family) {
        familyId.value = family.id;
        window.sessionStorage.setItem(SESSION_KEY, family.id);
      } else {
        error.value =
          "Ссылка недействительна или больше не работает. Попросите мастера новую.";
      }
    } else {
      const saved = window.sessionStorage.getItem(SESSION_KEY);
      if (saved && familyById(saved)) familyId.value = saved;
    }
    ready.value = true;
  });

  const family = computed(() =>
    familyId.value ? familyById(familyId.value) : undefined,
  );

  const children = computed(() =>
    family.value
      ? family.value.childIds
          .map((id) => state.children.find((child) => child.id === id))
          .filter((child): child is NonNullable<typeof child> => Boolean(child))
      : [],
  );

  function loginAs(id: string): void {
    error.value = "";
    if (!familyById(id)) return;
    familyId.value = id;
    if (typeof window !== "undefined") {
      window.sessionStorage.setItem(SESSION_KEY, id);
    }
  }

  function logout(): void {
    familyId.value = "";
    if (typeof window !== "undefined") {
      window.sessionStorage.removeItem(SESSION_KEY);
    }
  }

  return {
    familyId,
    family,
    children,
    ready,
    error,
    loginAs,
    logout,
  };
}
