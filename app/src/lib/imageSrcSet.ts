/**
 * Responsive image srcSet generator for gallery views.
 * Uses the /images/:id/resize endpoint with width descriptors.
 */

const SRCSET_WIDTHS = [256, 500, 1000, 1080, 1920] as const;
const DEFAULT_QUALITY = 90;

/**
 * Builds a srcSet string for pixel-width breakpoints.
 * Includes q (quality) param — defaults to 90.
 * Example: "/images/42/resize?w=256&q=90&token=abc 256w, ..."
 */
export function buildSrcSet(blobId: number, token: string | null, quality: number = DEFAULT_QUALITY): string {
    return SRCSET_WIDTHS.map((w) => {
        let url = `/images/${blobId}/resize?w=${w}&q=${quality}`;
        if (token) url += `&token=${token}`;
        return `${url} ${w}w`;
    }).join(", ");
}

/**
 * Builds a srcSet string for percentage-based breakpoints (25%, 50%, 75%).
 * These are resolved server-side to absolute pixel widths.
 */
export function buildPercentageSrcSet(blobId: number, token: string | null, quality: number = DEFAULT_QUALITY): string {
    const percentages = [25, 50, 75] as const;
    return percentages.map((p) => {
        let url = `/images/${blobId}/resize?p=${p}&q=${quality}`;
        if (token) url += `&token=${token}`;
        return url;
    }).join(", ");
}

/** Gallery card sizes attribute for responsive images */
export const GALLERY_CARD_SIZES = "(max-width: 640px) 100vw, (max-width: 1024px) 50vw, 33vw";

/** Panel thumbnail sizes attribute (96px / w-24) */
export const PANEL_THUMB_SIZES = "96px";
