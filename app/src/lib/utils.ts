export function cn(...parts: Array<string | false | null | undefined>) {
    return parts.filter(Boolean).join(" ");
}

export function intersect(a: string[], b: string[]) {
    const bs = new Set(b);
    for (const x of a) if (bs.has(x)) return true;
    return false;
}

export function formatBytes(bytes: number) {
    const thresh = 1024;
    if (bytes < thresh) return `${bytes} B`;
    const units = ["KB", "MB", "GB", "TB"];
    let u = -1;
    let v = bytes;
    do {
        v /= thresh;
        u++;
    } while (v >= thresh && u < units.length - 1);
    return `${v.toFixed(v >= 10 || u === 0 ? 0 : 1)} ${units[u]}`;
}

export function safeRevoke(url?: string) {
    if (!url) return;
    try {
        URL.revokeObjectURL(url);
    } catch {
        // ignore
    }
}

export function clamp(n: number, a: number, b: number) {
    return Math.max(a, Math.min(b, n));
}

export const uid = () => Math.random().toString(16).slice(2) + "_" + Date.now().toString(16);
