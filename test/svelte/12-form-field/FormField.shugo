<script lang="ts">
  type Option = { value: string; label: string };
  type Props = {
    id: string;
    name: string;
    label: string;
    type: "text" | "email" | "select" | "textarea";
    value?: string;
    options?: Option[];
    placeholder?: string;
    hint?: string;
    error?: string;
    required?: boolean;
    disabled?: boolean;
  };
  let { id, name, label, type, value, options = [], placeholder, hint, error, required = false, disabled = false }: Props = $props();
  const hintId = $derived(hint ? `${id}-hint` : undefined);
  const errorId = $derived(error ? `${id}-error` : undefined);
  const describedBy = $derived([hintId, errorId].filter(Boolean).join(" ") || undefined);
  const invalid = $derived(error ? "true" : "false");
</script>

<div class={["field", { "field--invalid": error, "field--disabled": disabled }]}>
  <!-- The space stays outside the block: Svelte drops whitespace at the edges of a block. -->
  <label for={id}>{label} {#if required}<span class="required" aria-hidden="true">*</span>{/if}</label>
  <div class="control">
    {#if type === "select"}
      <select {id} {name} {required} {disabled} aria-invalid={invalid} aria-describedby={describedBy} {value}>
        {#each options as option}<option value={option.value}>{option.label}</option>{/each}
      </select>
    {:else if type === "textarea"}
      <textarea {id} {name} rows="4" placeholder={placeholder || undefined} {required} {disabled} aria-invalid={invalid} aria-describedby={describedBy} {value}></textarea>
    {:else}
      <input {type} {id} {name} {value} placeholder={placeholder || undefined} {required} {disabled} aria-invalid={invalid} aria-describedby={describedBy} />
    {/if}
  </div>
  {#if hintId}<p class="hint" id={hintId}>{hint}</p>{/if}
  {#if errorId}<p class="error" id={errorId}>{error}</p>{/if}
</div>
