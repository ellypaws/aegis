import { useMemo, useState } from "react";
import { cn } from "../lib/utils";
import { UI } from "../constants";
import type { Post } from "../types";
import { LockedOverlay } from "./LockedOverlay";
import { ImageWithSpinner } from "./ImageWithSpinner";
import { ChevronLeft, ChevronRight } from "lucide-react";

export function PostDetailView({
    selected,
    onBack,
    canAccessPost,
}: {
    selected: Post | null;
    onBack: () => void;
    canAccessPost: (p: Post) => boolean;
}) {
    // Use the hydrated post data directly from App.tsx — no extra fetch needed
    const activePost = selected;
    const images = useMemo(() => activePost?.images || [], [activePost]);
    const [currentIndex, setCurrentIndex] = useState(0);

    const canAccess = useMemo(() => {
        if (!activePost) return false;
        return canAccessPost(activePost);
    }, [activePost, canAccessPost]);

    const currentAccessLabel = useMemo(() => {
        if (!activePost) return "";
        if (canAccess) return "Public"; // Simplified label if accessible
        return activePost.allowedRoles.length ? `Requires: ${activePost.allowedRoles.map(r => r.name).join(", ")}` : "Public";
    }, [activePost, canAccess]);

    const currentImage = images[currentIndex];

    // Helper to get URL for main display or thumbnail
    const getUrl = (blobId: number | undefined, access: boolean, type: "full" | "thumb" = "full") => {
        if (!blobId) return undefined;
        const token = localStorage.getItem("jwt");
        if (access) {
            if (type === "thumb") {
                return `/images/${blobId}/resize?w=256${token ? `&token=${token}` : ""}`;
            }
            return `/images/${blobId}${token ? `?token=${token}` : ""}`;
        }
        return `/thumb/${blobId}`;
    };

    const displayUrl = currentImage ? getUrl(currentImage.blobs?.[0]?.ID, canAccess, "full") : undefined;

    const handleNext = () => {
        setCurrentIndex((prev) => (prev + 1) % images.length);
    };

    const handlePrev = () => {
        setCurrentIndex((prev) => (prev - 1 + images.length) % images.length);
    };

    return (
        <div className={cn("p-4", UI.card, "backdrop-blur-sm")}>
            <div className="flex items-center justify-between gap-3">
                <div className="min-w-0">
                    <div className="truncate text-lg font-black uppercase tracking-wide text-zinc-900 dark:text-zinc-100">
                        {activePost?.title ?? "Untitled"}
                    </div>
                </div>
                <button type="button" onClick={onBack} className={cn(UI.button, UI.btnYellow)}>
                    Back
                </button>
            </div>

            <div className="mt-4 relative overflow-hidden rounded-3xl border-4 border-zinc-200 dark:border-zinc-700 bg-white dark:bg-zinc-900 shadow-[6px_6px_0px_rgba(0,0,0,0.14)] dark:shadow-[6px_6px_0px_rgba(0,0,0,0.5)]">
                <div className="w-full min-h-[50vh] max-h-[85vh] flex items-center justify-center relative bg-black/5">
                    {activePost ? (
                        (() => {
                            if (displayUrl) {
                                const contentType = currentImage?.blobs?.[0]?.contentType || "";
                                const filename = currentImage?.blobs?.[0]?.filename || "";
                                const isVideo = contentType.startsWith("video/") || /\.(mp4|webm|mov|mkv|avi)$/i.test(filename);

                                return (
                                    <>
                                        {isVideo && canAccess ? (
                                            <video
                                                src={displayUrl}
                                                controls
                                                autoPlay
                                                loop
                                                className="max-h-[85vh] w-full h-auto object-contain bg-black mx-auto"
                                            />
                                        ) : (
                                            <ImageWithSpinner
                                                src={displayUrl}
                                                alt={activePost.title ?? ""}
                                                className="max-h-[85vh] w-full h-auto object-contain bg-white dark:bg-zinc-900 mx-auto"
                                                draggable={false}
                                            />
                                        )}

                                        {/* Carousel Controls */}
                                        {images.length > 1 && (
                                            <>
                                                <button
                                                    onClick={handlePrev}
                                                    className="absolute left-4 top-1/2 -translate-y-1/2 p-3 rounded-full bg-black/40 text-white/80 hover:bg-black/60 hover:text-white transition-all backdrop-blur-sm border-2 border-white/10"
                                                >
                                                    <ChevronLeft className="h-8 w-8" />
                                                </button>
                                                <button
                                                    onClick={handleNext}
                                                    className="absolute right-4 top-1/2 -translate-y-1/2 p-3 rounded-full bg-black/40 text-white/80 hover:bg-black/60 hover:text-white transition-all backdrop-blur-sm border-2 border-white/10"
                                                >
                                                    <ChevronRight className="h-8 w-8" />
                                                </button>
                                            </>
                                        )}
                                    </>
                                );
                            }
                            if (!canAccess) {
                                return (
                                    <div className="flex h-full w-full items-center justify-center py-20">
                                        <div className="text-center">
                                            <div className="text-sm font-black uppercase text-zinc-500 dark:text-zinc-400">🔒 Locked</div>
                                            <div className="mt-1 text-sm font-bold text-zinc-400 dark:text-zinc-500">{currentAccessLabel}</div>
                                        </div>
                                    </div>
                                );
                            }
                            return (
                                <div className="flex h-full w-full items-center justify-center text-sm font-bold text-zinc-400 dark:text-zinc-500 py-20">
                                    No media
                                </div>
                            );
                        })()
                    ) : (
                        <div className="flex h-full w-full items-center justify-center text-sm font-bold text-zinc-400 dark:text-zinc-500 py-20">
                            No posts
                        </div>
                    )}
                </div>

                {!canAccess && currentImage?.blobs?.[0]?.ID ? (
                    <div className="absolute inset-0 pointer-events-none">
                        <LockedOverlay label={currentAccessLabel} />
                    </div>
                ) : null}
            </div>

            {/* Film Strip */}
            {images.length > 1 && (
                <div className="mt-4 overflow-x-auto pb-4 [&::-webkit-scrollbar]:h-3 [&::-webkit-scrollbar-track]:bg-transparent [&::-webkit-scrollbar-thumb]:rounded-full [&::-webkit-scrollbar-thumb]:bg-zinc-300 dark:[&::-webkit-scrollbar-thumb]:bg-zinc-600">
                    <div className="flex gap-3 px-1 w-fit mx-auto">
                        {images.map((img, idx) => {
                            const blob = img.blobs?.[0];
                            const thumbUrl = getUrl(blob?.ID, canAccess, "thumb");
                            const isSelected = idx === currentIndex;
                            const contentType = blob?.contentType || "";
                            const isVideo = contentType.startsWith("video/");

                            return (
                                <button
                                    key={img.ID || idx}
                                    onClick={() => setCurrentIndex(idx)}
                                    className={cn(
                                        "relative h-20 w-20 shrink-0 overflow-hidden rounded-xl border-4 transition-all duration-200",
                                        isSelected
                                            ? "border-zinc-900 dark:border-zinc-100 shadow-[0_0_0_2px_rgba(255,255,255,1)] dark:shadow-[0_0_0_2px_rgba(0,0,0,1)] scale-105 z-10"
                                            : "border-transparent opacity-70 hover:opacity-100 hover:border-zinc-300 dark:hover:border-zinc-600"
                                    )}
                                >
                                    {thumbUrl ? (
                                        isVideo ? (
                                            <video
                                                src={thumbUrl}
                                                muted
                                                className="h-full w-full object-cover"
                                                onMouseOver={e => e.currentTarget.play().catch(() => { })}
                                                onMouseOut={e => e.currentTarget.pause()}
                                            />
                                        ) : (
                                            <img
                                                src={thumbUrl}
                                                alt={`Thumbnail ${idx + 1}`}
                                                className="h-full w-full object-cover"
                                                draggable={false}
                                            />
                                        )
                                    ) : (
                                        <div className="flex h-full w-full items-center justify-center bg-zinc-100 dark:bg-zinc-800">
                                            <span className="text-xs text-zinc-400">No img</span>
                                        </div>
                                    )}
                                    <div className="absolute bottom-0 right-0 bg-black/60 px-1.5 py-0.5 text-[9px] font-bold text-white rounded-tl-lg">
                                        {idx + 1}
                                    </div>
                                    {isVideo && (
                                        <div className="absolute inset-0 flex items-center justify-center pointer-events-none">
                                            <div className="w-6 h-6 rounded-full bg-black/50 flex items-center justify-center">
                                                <div className="w-0 h-0 border-t-4 border-t-transparent border-l-6 border-l-white border-b-4 border-b-transparent ml-0.5" />
                                            </div>
                                        </div>
                                    )}
                                </button>
                            );
                        })}
                    </div>
                </div>
            )}

            {activePost?.description ? (
                <div className={cn("mt-4 p-4", UI.card)}>
                    <div className="text-sm text-zinc-600 dark:text-zinc-300 font-medium whitespace-pre-wrap">
                        {activePost.description}
                    </div>
                </div>
            ) : null}
        </div>
    );
}
