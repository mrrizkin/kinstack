import { ref } from "vue";

// Shell-level UI state shared by the layout and the components rendered inside
// it. A module singleton keeps the dashboard shell free of extra providers.
export const notificationsOpen = ref(false);
export const unreadCount = ref(0);
