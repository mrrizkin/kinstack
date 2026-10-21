/**
 * Icon registry: the only place an icon name is written.
 *
 * One collection, so migrating is an edit to this file. Values stay literal
 * because the Nuxt UI Vite plugin bundles icons by scanning source.
 */
export const icons = {
  // Shell navigation
  dashboard: "i-material-symbols:dashboard-outline-rounded",
  users: "i-material-symbols:group-outline-rounded",
  access: "i-material-symbols:shield-lock-outline-rounded",
  audit: "i-material-symbols:history-rounded",
  media: "i-material-symbols:image-outline-rounded",
  notifications: "i-material-symbols:notifications-outline-rounded",
  settings: "i-material-symbols:settings-outline-rounded",
  help: "i-material-symbols:help-outline-rounded",

  // Settings sections
  account: "i-material-symbols:person-outline-rounded",
  security: "i-material-symbols:shield-outline-rounded",
  preferences: "i-material-symbols:tune-rounded",
  passkey: "i-material-symbols:passkey-rounded",
  token: "i-material-symbols:key-outline-rounded",

  // Appearance
  light: "i-material-symbols:light-mode-outline-rounded",
  dark: "i-material-symbols:dark-mode-outline-rounded",
  system: "i-material-symbols:desktop-windows-outline-rounded",

  // Actions
  add: "i-material-symbols:add-rounded",
  minus: "i-material-symbols:remove-rounded",
  check: "i-material-symbols:check-rounded",
  close: "i-material-symbols:close-rounded",
  copied: "i-material-symbols:inventory-rounded",
  copy: "i-material-symbols:content-copy-outline-rounded",
  delete: "i-material-symbols:delete-outline-rounded",
  download: "i-material-symbols:download-rounded",
  drag: "i-material-symbols:drag-indicator",
  edit: "i-material-symbols:edit-square-outline-rounded",
  filter: "i-material-symbols:filter-alt-outline-rounded",
  invite: "i-material-symbols:mail-outline-rounded",
  key: "i-material-symbols:key-outline-rounded",
  lock: "i-material-symbols:lock-outline-rounded",
  login: "i-material-symbols:login-rounded",
  logout: "i-material-symbols:logout-rounded",
  mail: "i-material-symbols:mail-outline-rounded",
  palette: "i-material-symbols:palette-outline-rounded",
  reload: "i-material-symbols:refresh-rounded",
  reset: "i-material-symbols:restart-alt-rounded",
  search: "i-material-symbols:search-rounded",
  sliders: "i-material-symbols:tune-rounded",
  stop: "i-material-symbols:stop-outline-rounded",
  upload: "i-material-symbols:upload-rounded",
  cloudUpload: "i-material-symbols:cloud-upload-outline-rounded",
  unfoldMore: "i-material-symbols:unfold-more-rounded",
  external: "i-material-symbols:arrow-outward-rounded",
  more: "i-material-symbols:more-vert",
  eye: "i-material-symbols:visibility-outline-rounded",
  eyeOff: "i-material-symbols:visibility-off-outline-rounded",

  // Panels and menus
  menu: "i-material-symbols:menu-rounded",
  panelClose: "i-material-symbols:left-panel-close-outline-rounded",
  panelOpen: "i-material-symbols:left-panel-open-outline-rounded",

  // Navigation
  arrowLeft: "i-material-symbols:arrow-back-rounded",
  arrowRight: "i-material-symbols:arrow-forward-rounded",
  arrowUp: "i-material-symbols:arrow-upward-rounded",
  arrowDown: "i-material-symbols:arrow-downward-rounded",
  chevronLeft: "i-material-symbols:chevron-left-rounded",
  chevronRight: "i-material-symbols:chevron-right-rounded",
  chevronUp: "i-material-symbols:keyboard-arrow-up-rounded",
  chevronDown: "i-material-symbols:keyboard-arrow-down-rounded",
  chevronDoubleLeft: "i-material-symbols:keyboard-double-arrow-left-rounded",
  chevronDoubleRight: "i-material-symbols:keyboard-double-arrow-right-rounded",

  // Feedback
  success: "i-material-symbols:check-circle-outline-rounded",
  error: "i-material-symbols:cancel-outline-rounded",
  info: "i-material-symbols:info-outline-rounded",
  warning: "i-material-symbols:warning-outline-rounded",
  alert: "i-material-symbols:error-outline-rounded",
  tip: "i-material-symbols:lightbulb-outline-rounded",
  star: "i-material-symbols:star-outline-rounded",
  loading: "i-material-symbols:progress-activity",

  // Records and identity
  file: "i-material-symbols:description-outline-rounded",
  folder: "i-material-symbols:folder-outline-rounded",
  folderOpen: "i-material-symbols:folder-open-outline-rounded",
  tag: "i-material-symbols:tag-rounded",
  calendar: "i-material-symbols:calendar-month-outline-rounded",
  markRead: "i-material-symbols:mark-email-read-outline-rounded",
  markAllRead: "i-material-symbols:done-all-rounded",
  bellOff: "i-material-symbols:notifications-off-outline-rounded",
  shieldCheck: "i-material-symbols:verified-user-outline-rounded",
  shieldOff: "i-material-symbols:gpp-bad-outline-rounded",
  deny: "i-material-symbols:block-outline-rounded",
  qrCode: "i-material-symbols:qr-code-2-rounded",
  userAdd: "i-material-symbols:person-add-outline-rounded",
  userActivate: "i-material-symbols:person-check-outline-rounded",
  userDeactivate: "i-material-symbols:person-off-outline-rounded",
  userPermissions: "i-material-symbols:manage-accounts-outline-rounded",
  domain: "i-material-symbols:domain-rounded",
  docs: "i-material-symbols:menu-book-outline-rounded",
  support: "i-material-symbols:support-agent-outline-rounded",
} as const;
