<script lang="ts">
  import type { Icon } from 'lucide-svelte';
  import Card from './Card.svelte';

  export interface BreadcrumbItem {
    label: string;
    href?: string;
    icon: typeof Icon;
  }

  let { items = [] }: { items: BreadcrumbItem[] } = $props();
</script>

<Card>
  <div class="breadcrumbs p-4 text-sm">
    <ul>
      {#each items as { label, href, icon: IconComponent }}
        <li>
          {#if href}
            <a {href}>
              <IconComponent class="h-4 w-4 stroke-current" />
              {label}
            </a>
          {:else}
            <span class="inline-flex items-center gap-2">
              <IconComponent class="h-4 w-4 stroke-current" />
              {label}
            </span>
          {/if}
        </li>
      {/each}
    </ul>
  </div>
</Card>