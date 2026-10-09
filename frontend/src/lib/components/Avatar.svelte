<script lang="ts">
    import { getInitials } from '$lib/utils/getInitials';

    /*
     * A person's initials in a neutral circle, the same for every role
     * (docs/design/style-guide.md → Avatar). Role is shown by a badge, never by colour.
     */
    type Size = 'sm' | 'md' | 'lg';

    interface Props {
        displayName?: string;
        size?: Size;
    }

    let { displayName = '', size = 'sm' }: Props = $props();

    const sizeClasses: Record<Size, string> = {
        sm: 'w-8 text-xs',
        md: 'w-10 text-sm',
        lg: 'w-12 text-base'
    };
</script>

<div class="avatar avatar-placeholder" aria-hidden="true">
    <div class="rounded-full border border-base-300 bg-base-200 text-base-content {sizeClasses[size]}">
        <span>{getInitials(displayName)}</span>
    </div>
</div>
