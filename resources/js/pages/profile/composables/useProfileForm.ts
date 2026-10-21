import { useForm } from "@inertiajs/vue3";
import { watch } from "vue";

import { useServerErrors } from "@/lib/errors";

import type { Profile } from "../types";

interface ProfileFormProps {
  profile: Profile;
}

export function useProfileForm(props: ProfileFormProps) {
  const profileForm = useForm({ display_name: props.profile.display_name });
  const { message: fieldError, formError } = useServerErrors();

  watch(
    () => props.profile.display_name,
    (value) => {
      profileForm.display_name = value;
    },
  );

  function submitProfile() {
    profileForm.post("/profile", { preserveScroll: true });
  }

  return { profileForm, fieldError, formError, submitProfile };
}
