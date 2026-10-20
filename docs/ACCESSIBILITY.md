# Accessibility

A manual checklist for the shell, forms, and overlays. Run it at least before a
major release and after navigation or form changes.

## Keyboard

- [ ] Every primary action is reachable with `Tab`, `Enter`, and `Space`.
- [ ] Focus is clearly visible in both light and dark mode.
- [ ] `Escape` closes dialogs, slideovers, dropdowns, and the command palette.
- [ ] Focus returns to the trigger after an overlay closes.
- [ ] No focus trap leaks to content behind an overlay.
- [ ] Tab order follows visual order.

## Landmarks and labels

- [ ] Every page has one main heading.
- [ ] Primary navigation, content, and footer use appropriate landmarks.
- [ ] Icon buttons have a descriptive `aria-label`.
- [ ] Decorative images use `aria-hidden` or an empty alt.
- [ ] Every form field has a connected label.
- [ ] Error messages are readable by screen readers and connected to the field.

## Visual

- [ ] Text contrast is at least 4.5:1 for normal text.
- [ ] Focus and status are not distinguished by color alone.
- [ ] Touch targets are at least 44×44 px on mobile.
- [ ] The layout remains readable at 200% zoom.
- [ ] `prefers-reduced-motion` does not trigger distracting animation.

## Mobile

- [ ] The sidebar can be opened and closed with the keyboard and a button.
- [ ] Tables are replaced by card lists on narrow viewports.
- [ ] There is no horizontal overflow at 320px and 390px.
- [ ] Row actions remain available through a context menu.

## Current status

- Primary icon labels and sidebar buttons have text labels.
- There is no comprehensive screen reader audit yet.
- There is no skip link in the dashboard shell yet.
- There is no automated contrast test in CI yet.
