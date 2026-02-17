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

export function resolveImageSrc(data: string | undefined, contentType?: string): string | undefined {
    if (!data) return undefined;
    if (data.startsWith("http") || data.startsWith("blob:") || data.startsWith("data:")) return data;

    // Guess type if missing
    let mime = contentType;
    if (!mime) {
        if (data.startsWith("iVBORw")) mime = "image/png";
        else if (data.startsWith("/9j/")) mime = "image/jpeg";
        else if (data.startsWith("R0lGOD")) mime = "image/gif";
        else if (data.startsWith("UklGR")) mime = "image/webp";
        else mime = "image/png"; // Default fallback
    }
    return `data:${mime};base64,${data}`;
}

export function base64ToBlob(base64: string, type = "application/octet-stream") {
    const binStr = atob(base64);
    const len = binStr.length;
    const arr = new Uint8Array(len);
    for (let i = 0; i < len; i++) {
        arr[i] = binStr.charCodeAt(i);
    }
    return new Blob([arr], { type });
}

export function getExtension(mime?: string): string {
    if (!mime) return "png";
    const parts = mime.toLowerCase().split("/");
    if (parts.length < 2) return "png";
    const ext = parts[1].split(";")[0].split("+")[0];
    if (ext === "jpeg") return "jpg";
    if (ext === "plain") return "txt";
    if (ext === "octet-stream") return "png"; // Fallback for generic stream
    if (ext === "x-icon") return "ico";
    if (ext === "svg+xml") return "svg";

    // Video types
    if (ext === "mp4") return "mp4";
    if (ext === "webm") return "webm";
    if (ext === "quicktime") return "mov";
    if (ext === "x-matroska") return "mkv";
    if (ext === "x-msvideo") return "avi";

    return ext || "png";
}
