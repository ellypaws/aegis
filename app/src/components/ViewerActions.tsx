import { useState, useEffect } from "react";
import { cn, getExtension } from "../lib/utils";
import type { Post } from "../types";
import { DownloadButton, type DownloadFile } from "./DownloadButton";
import { Share2, MessageCircle, ExternalLink } from "lucide-react";

export function ViewerActions({ post, canAccess }: { post: Post; canAccess: boolean }) {
    const [toast, setToast] = useState<string | null>(null);
    const [sendingDM, setSendingDM] = useState(false);

    useEffect(() => {
        if (!toast) return;
        const t = setTimeout(() => setToast(null), 3000);
        return () => clearTimeout(t);
    }, [toast]);

    const token = localStorage.getItem("jwt");

    const files: DownloadFile[] = (post.images || []).flatMap((img, i) => {
        const blob = img.blobs?.[0];
        if (!blob) return [];

        const fullUrl = `/images/${blob.ID}${token ? `?token=${token}` : ""}`;
        const downloadUrl = fullUrl + (fullUrl.includes("?") ? "&" : "?") + "download=1";
        const ext = getExtension(blob.contentType);

        return [{
            url: downloadUrl,
            name: blob.filename || `${post.title || "image"}-${i + 1}.${ext}`,
            mime: (blob.contentType === "application/octet-stream" ? "image/png" : blob.contentType) || "image/png",
            size: blob.size || 0,
        }];
    });

    const blob = post.images?.[0]?.blobs?.[0];
    const fullUrl = blob ? `/images/${blob.ID}${token ? `?token=${token}` : ""}` : "";

    const handleShare = () => {
        const url = `${window.location.origin}/posts/${post.postKey}`;
        navigator.clipboard.writeText(url).then(() => {
            setToast("Link copied to clipboard!");
        }).catch(() => {
            setToast("Failed to copy link.");
        });
    };

    const handleDM = async () => {
        if (!canAccess || sendingDM) return;
        setSendingDM(true);
        try {
            const res = await fetch(`/posts/${post.postKey}/dm`, {
                method: 'POST',
                headers: {
                    'Authorization': `Bearer ${token}`
                }
            });
            if (res.ok) {
                setToast("Sent to your DMs!");
            } else {
                setToast("Failed to send DM. Make sure your DMs are open.");
            }
        } catch (e) {
            setToast("Error sending DM.");
        } finally {
            setSendingDM(false);
        }
    };

    return (
        <div className="space-y-3">
            <DownloadButton files={files} enabled={canAccess} />

            <div className="grid grid-cols-3 gap-2">
                <a
                    href={canAccess ? fullUrl : undefined}
                    target={canAccess ? "_blank" : undefined}
                    rel="noreferrer"
                    onClick={(e) => {
                        if (!canAccess) e.preventDefault();
                    }}
                    className={cn(
                        "flex items-center justify-center gap-2 rounded-xl border-2 px-3 py-2 text-xs font-bold transition-all",
                        canAccess
                            ? "border-green-300 bg-white text-green-700 hover:bg-green-50 dark:border-green-800 dark:bg-zinc-900 dark:text-green-400 dark:hover:bg-green-900/20"
                            : "cursor-not-allowed border-zinc-200 bg-zinc-50 text-zinc-400 dark:border-zinc-800 dark:bg-zinc-900/50 dark:text-zinc-600"
                    )}
                    title={canAccess ? "Open full in new tab" : "No access"}
                >
                    <ExternalLink className="w-4 h-4" />
                    Open
                </a>

                <button
                    type="button"
                    onClick={handleDM}
                    disabled={!canAccess || sendingDM}
                    className={cn(
                        "flex items-center justify-center gap-2 rounded-xl border-2 px-3 py-2 text-xs font-bold transition-all",
                        canAccess
                            ? "border-purple-300 bg-white text-purple-700 hover:bg-purple-50 dark:border-purple-800 dark:bg-zinc-900 dark:text-purple-400 dark:hover:bg-purple-900/20"
                            : "cursor-not-allowed border-zinc-200 bg-zinc-50 text-zinc-400 dark:border-zinc-800 dark:bg-zinc-900/50 dark:text-zinc-600",
                        sendingDM && "opacity-70 cursor-wait"
                    )}
                    title={canAccess ? "Send to DMs" : "No access"}
                >
                    <MessageCircle className="w-4 h-4" />
                    {sendingDM ? "..." : "DM"}
                </button>

                <button
                    type="button"
                    onClick={handleShare}
                    className={cn(
                        "flex items-center justify-center gap-2 rounded-xl border-2 px-3 py-2 text-xs font-bold transition-all",
                        "border-blue-300 bg-white text-blue-700 hover:bg-blue-50 dark:border-blue-800 dark:bg-zinc-900 dark:text-blue-400 dark:hover:bg-blue-900/20"
                    )}
                    title="Copy link"
                >
                    <Share2 className="w-4 h-4" />
                    Share
                </button>
            </div>

            {toast ? (
                <div className="rounded-xl border-2 border-zinc-200 bg-white dark:bg-zinc-900 dark:border-zinc-700 px-3 py-2 text-xs font-bold text-zinc-600 dark:text-zinc-300 shadow-sm text-center animate-in fade-in slide-in-from-bottom-2">
                    {toast}
                </div>
            ) : null}
        </div>
    );
}
