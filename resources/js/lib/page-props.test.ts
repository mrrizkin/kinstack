import { describe, expect, mock, test } from "bun:test";

import { ref } from "vue";
import { z } from "zod";

const page = ref<{ props: Record<string, unknown> }>({
  props: { users: [{ id: "user-1" }] },
});

mock.module("@inertiajs/vue3", () => ({
  usePage: () => ({
    get props() {
      return page.value.props;
    },
  }),
}));

const { usePageSchema } = await import("./page-props");

const pageSchema = z.looseObject({ users: z.array(z.looseObject({ id: z.string() })) });

describe("usePageSchema", () => {
  test("re-reads props after a state-preserving Inertia mutation", () => {
    const props = usePageSchema(pageSchema);
    expect(props.users[0]?.id).toBe("user-1");

    page.value = { props: { users: [{ id: "user-2" }] } };
    expect(props.users[0]?.id).toBe("user-2");
  });

  test("parses through zod instead of casting the prop bag", () => {
    page.value = { props: { users: [{ id: 42 }] } };
    expect(() => usePageSchema(pageSchema)).toThrow();
  });
});
