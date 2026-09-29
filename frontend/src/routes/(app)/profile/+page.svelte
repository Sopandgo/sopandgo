<script>
    import Alert from "$lib/components/Alert.svelte";
    import Card from "$lib/components/Card.svelte";
    import CardPageHeading from "$lib/components/CardPageHeading.svelte";
    import ChangePassword from "$lib/components/ChangePassword.svelte";
    import ListUserSignatures from "$lib/components/ListUserSignatures.svelte";
    import {
        CalendarClockIcon,
        IdCardIcon,
        MailIcon,
        StampIcon,
        UserIcon,
    } from "lucide-svelte";
    let { data, form } = $props();

    const user = $derived(data.user);
    const mustChange = $derived(!!user?.must_change_password);
</script>

{#if mustChange}
    <Card>
        <div class="card-body gap-4">
            <CardPageHeading>
                <UserIcon class="w-8 h-8" />
                Change your password
            </CardPageHeading>
            <Alert
                variant="warning"
                message="You are using the default bootstrap password. Choose a new password before continuing."
            />
            <ChangePassword {form} showLogoutWarning={true} />
        </div>
    </Card>
{:else}
    <Card>
        <div class="card-body">
            <CardPageHeading>
                <UserIcon class="w-8 h-8" />
                Your Profile
            </CardPageHeading>

            <div class="flex items-center gap-2 text-base-content/70">
                <span class="text-sm font-medium"
                    >Everything you need to manage your sopandgo profile.</span
                >
            </div>
        </div>
    </Card>

    <div class="flex flex-col lg:flex-row gap-6 mt-6">
        <ul class="list w-full gap-6">
            <Card>
                <li
                    class="p-4 pb-2 text-xs opacity-60 tracking-widest uppercase font-bold border-b border-base-200/50"
                >
                    Information
                </li>

                <li class="list-row items-center">
                    <div>
                        <UserIcon size={24} class="p-1 opacity-70" />
                    </div>
                    <div class="flex-1">
                        <div class="text-sm font-medium">{user.display_name}</div>
                    </div>
                </li>

                <li class="list-row items-center">
                    <div>
                        <MailIcon size={24} class="p-1 opacity-70" />
                    </div>
                    <div class="flex-1">
                        <div class="text-sm font-medium">{user.email}</div>
                    </div>
                </li>

                <li class="list-row items-center">
                    <div>
                        <StampIcon size={24} class="p-1 opacity-70" />
                    </div>
                    <div class="flex-1">
                        <div class="text-sm font-medium">Role: {user.role}</div>
                    </div>
                </li>

                <li class="list-row items-center">
                    <div>
                        <CalendarClockIcon size={24} class="p-1 opacity-70" />
                    </div>
                    <div class="flex-1">
                        <div class="text-sm font-medium">
                            Created: {new Date(user.created_at).toLocaleString()}
                        </div>
                        <div class="text-xs opacity-40 font-mono italic">
                            Timestamp (UTC)
                        </div>
                    </div>
                </li>

                <li class="list-row items-center">
                    <div>
                        <IdCardIcon size={24} class="p-1 opacity-70" />
                    </div>
                    <div class="flex-1">
                        <div class="text-sm font-medium">ID: {user.id}</div>
                    </div>
                </li>
            </Card>

            <ListUserSignatures status={data.signatureStatus || []} />
        </ul>

        <Card>
            <div class="card-body">
                <h2
                    class="card-title text-xs opacity-60 tracking-widest uppercase font-bold mb-4"
                >
                    Update Password
                </h2>
                <ChangePassword {form} showLogoutWarning={true} />
            </div>
        </Card>
    </div>
{/if}
