<script lang="ts">
    import { resolve } from '$app/paths';
    import { getInitials } from '#lib/utils/getInitials.js';

    /*
     * A person's picture, or their initials in a neutral circle when they have none
     * (docs/design/style-guide.md → Avatar). Same for every role: role is shown by a
     * badge, never by colour.
     */
    type Size = 'sm' | 'md' | 'lg';

    interface Props {
        displayName?: string;
        size?: Size;
        /** With `hasAvatar`, loads the picture through the same-origin proxy. */
        userId?: string;
        hasAvatar?: boolean;
        /** Cache-busting value: changes whenever the picture changes. */
        avatarContentHash?: string;
    }

    let {
        displayName = '',
        size = 'sm',
        userId = '',
        hasAvatar = false,
        avatarContentHash = ''
    }: Props = $props();

    const sizeClasses: Record<Size, string> = {
        sm: 'w-8 text-xs',
        md: 'w-10 text-sm',
        lg: 'w-12 text-base'
    };

    // Stored renditions are sm and md; the largest Avatar uses md.
    const imageSize = $derived(size === 'lg' ? 'md' : 'sm');

    const src = $derived(hasAvatar && userId
        ? resolve(`api/users/avatar?user_id=${encodeURIComponent(userId)}&size=${imageSize}&v=${encodeURIComponent(avatarContentHash)}`)
        : '');

    // Remember which URL failed so a new picture (new hash) is tried again.
    let failedSrc = $state('');
    const showImage = $derived(src !== '' && src !== failedSrc);
</script>

<div class="avatar {showImage ? '' : 'avatar-placeholder'}" aria-hidden="true">
    {#if showImage}
        <div
            class="rounded-full border border-base-300 bg-base-200 {sizeClasses[size]}"
        ><img src={src} alt="" onerror={() => failedSrc = src} /></div>
    {:else}
        <div class="rounded-full border border-base-300 bg-base-200 text-base-content {sizeClasses[size]}">
            <span>{getInitials(displayName)}</span>
        </div>
    {/if}
</div>
