import { useState, useEffect } from "react";
import { cn, getExtension } from "../lib/utils";
import { UI } from "../constants";
import type { Post } from "../types";
import { DownloadButton, type DownloadFile } from "./DownloadButton";

export function ViewerActions({ post, canAccess }: { post: Post; canAccess: boolean }) {
    const [toast, setToast] = useState<string | null>(null);
    useEffect(() => {
        if (!toast) return;
        const t = setTimeout(() => setToast(null), 1800);
        return () => clearTimeout(t);
    }, [toast]);

    const blob = post.image?.blobs?.[0];
    const token = localStorage.getItem("jwt");
    const fullUrl = blob ? `/images/${blob.ID}${token ? `?token=${token}` : ""}` : "";
    const downloadUrl = fullUrl ? fullUrl + (fullUrl.includes("?") ? "&" : "?") + "download=1" : "";
    const ext = getExtension(blob?.contentType);

    const file: DownloadFile = {
        url: downloadUrl,
        name: blob?.filename || `${post.title || "image"}.${ext}`,
        mime: blob?.contentType || "image/png",
        size: blob?.size || 0,
    };

    return (
        <div className="space-y-3">
            <DownloadButton file={file} enabled={canAccess} />

            <div className="grid grid-cols-3 gap-2">
                <a
                    href={canAccess ? fullUrl : undefined}
                    target={canAccess ? "_blank" : undefined}
                    rel="noreferrer"
                    onClick={(e) => {
                        if (!canAccess) e.preventDefault();
                    }}
                    className={cn(UI.button, "px-3 py-2 text-xs", canAccess ? UI.btnGreen : cn(UI.btnYellow, UI.btnDisabled))}
                    title={canAccess ? "Open full in new tab" : "No access"}
                >
                    Open
                </a>
                <button
                    type="button"
                    onClick={() => setToast(canAccess ? "Pretend: sent to DMs" : "No access")}
                    disabled={!canAccess}
                    className={cn(UI.button, "px-3 py-2 text-xs", canAccess ? UI.btnRed : cn(UI.btnYellow, UI.btnDisabled))}
                    title={canAccess ? "Send to DMs (mock)" : "No access"}
                >
                    DM
                </button>
                <button type="button" onClick={() => setToast("Copied post link (mock)")} className={cn(UI.button, "px-3 py-2 text-xs", UI.btnBlue)}>
                    Share
                </button>
            </div>

            {toast ? (
                <div className="rounded-2xl border-4 border-zinc-200 bg-white px-3 py-2 text-xs font-bold text-zinc-600 shadow-[3px_3px_0px_rgba(0,0,0,0.12)]">
                    {toast}
                </div>
            ) : null}
        </div>
    );
}
