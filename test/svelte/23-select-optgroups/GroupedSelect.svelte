<script lang="ts">
  type Option = { value: string; label: string; disabled?: boolean };
  type Group = { label: string; options: Option[] };
  type Props = {
    id: string;
    name: string;
    value: string;
    placeholder: string;
    options: Option[];
    groups: Group[];
    required?: boolean;
  };
  let { id, name, value, placeholder, options, groups, required = false }: Props = $props();
</script>

{#snippet item(option: Option)}
  <option value={option.value} selected={option.value === value} disabled={option.disabled}>{option.label}</option>
{/snippet}

<select {id} {name} {required}>
  <option value="" disabled selected={value === ""}>{placeholder}</option>
  {#each options as option}{@render item(option)}{/each}
  {#each groups as group}
    {#if group.options.length > 0}
      <optgroup label={group.label}>
        {#each group.options as option}{@render item(option)}{/each}
      </optgroup>
    {/if}
  {/each}
</select>
