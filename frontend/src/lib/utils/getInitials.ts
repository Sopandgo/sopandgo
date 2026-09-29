/**
 * Derive initials from a display name for use in Avatars.
 * Handles single names, multiple names, and edge cases safely.
 */
export function getInitials(name: string | null | undefined): string {
    // 1. Safety check for null, undefined, or empty values
    if (!name || typeof name !== 'string' || name.trim() === '') {
        return '?';
    }

    // 2. Normalize and split by any whitespace
    const parts = name.trim().split(/\s+/);

    // 3. Handle single name case (e.g., "Gemini" -> "G")
    if (parts.length === 1) {
        return parts[0][0].toUpperCase();
    }

    // 4. Handle multiple names (e.g., "John Doe" or "John Michael Doe" -> "JD")
    return (
        parts[0][0].toUpperCase() +
        parts[parts.length - 1][0].toUpperCase()
    );
}