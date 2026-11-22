// node_modules/esm-env/false.js
var false_default = !1;

// node_modules/svelte/src/internal/client/warnings.js
var bold = "font-weight: bold", normal = "font-weight: normal";
function await_reactivity_loss(name) {
  false_default ? console.warn(`%c[svelte] await_reactivity_loss
%cDetected reactivity loss when reading \`${name}\`. This happens when state is read in an async function after an earlier \`await\`
https://svelte.dev/e/await_reactivity_loss`, bold, normal) : console.warn("https://svelte.dev/e/await_reactivity_loss");
}
function derived_inert() {
  false_default ? console.warn(`%c[svelte] derived_inert
%cReading a derived belonging to a now-destroyed effect may result in stale values
https://svelte.dev/e/derived_inert`, bold, normal) : console.warn("https://svelte.dev/e/derived_inert");
}
function hydration_mismatch(location) {
  false_default ? console.warn(
    `%c[svelte] hydration_mismatch
%c${location ? `Hydration failed because the initial UI does not match what was rendered on the server. The error occurred near ${location}` : "Hydration failed because the initial UI does not match what was rendered on the server"}
https://svelte.dev/e/hydration_mismatch`,
    bold,
    normal
  ) : console.warn("https://svelte.dev/e/hydration_mismatch");
}
function lifecycle_double_unmount() {
  false_default ? console.warn(`%c[svelte] lifecycle_double_unmount
%cTried to unmount a component that was not mounted
https://svelte.dev/e/lifecycle_double_unmount`, bold, normal) : console.warn("https://svelte.dev/e/lifecycle_double_unmount");
}
function svelte_boundary_reset_noop() {
  false_default ? console.warn("%c[svelte] svelte_boundary_reset_noop\n%cA `<svelte:boundary>` `reset` function only resets the boundary the first time it is called\nhttps://svelte.dev/e/svelte_boundary_reset_noop", bold, normal) : console.warn("https://svelte.dev/e/svelte_boundary_reset_noop");
}
function transition_slide_display(value) {
  false_default ? console.warn(`%c[svelte] transition_slide_display
%cThe \`slide\` transition does not work correctly for elements with \`display: ${value}\`
https://svelte.dev/e/transition_slide_display`, bold, normal) : console.warn("https://svelte.dev/e/transition_slide_display");
}

export {
  false_default,
  await_reactivity_loss,
  derived_inert,
  hydration_mismatch,
  lifecycle_double_unmount,
  svelte_boundary_reset_noop,
  transition_slide_display
};
