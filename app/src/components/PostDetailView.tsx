import { useMemo, useState, useRef } from "react";
import { cn } from "../lib/utils";
import { UI } from "../constants";
import type { Post } from "../types";
import { LockedOverlay } from "./LockedOverlay";
import { ImageWithSpinner } from "./ImageWithSpinner";
import { ChevronLeft, ChevronRight, CircleX } from "lucide-react";
import { motion, AnimatePresence } from "framer-motion";
import { createPortal } from "react-dom";

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
    const [direction, setDirection] = useState(0);

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

    const paginate = (newDirection: number) => {
        setDirection(newDirection);
        setCurrentIndex((prev) => (prev + newDirection + images.length) % images.length);
    };

    const setIndex = (index: number) => {
        setDirection(index > currentIndex ? 1 : -1);
        setCurrentIndex(index);
    };

    const variants = {
        enter: (direction: number) => ({
            x: direction > 0 ? 300 : -300,
            opacity: 0,
            scale: 0.95,
        }),
        center: {
            zIndex: 1,
            x: 0,
            opacity: 1,
            scale: 1,
        },
        exit: (direction: number) => ({
            zIndex: 0,
            x: direction < 0 ? 300 : -300,
            opacity: 0,
            scale: 0.95,
        }),
    };

    const [isModalOpen, setIsModalOpen] = useState(false);
    const [showControls, setShowControls] = useState(true);
    const controlsTimeoutRef = useRef<ReturnType<typeof setTimeout>>(null);

    const dragStartX = useRef(0);
    const hasMultipleImages = images.length > 1;

    const handleDragEnd = (_: any, { offset }: any) => {
        const swipe = offset.x; // positive = right, negative = left

        if (swipe < -50) {
            paginate(1);
        } else if (swipe > 50) {
            paginate(-1);
        }
    };

    // Auto-hide controls logic
    const handleMouseMove = () => {
        setShowControls(true);
        if (controlsTimeoutRef.current) clearTimeout(controlsTimeoutRef.current);
        controlsTimeoutRef.current = setTimeout(() => setShowControls(false), 3000);
    };

    // Clean up timeout on unmount or when modal closes
    const closeTimeout = () => {
        if (controlsTimeoutRef.current) clearTimeout(controlsTimeoutRef.current);
    };

    return (
        <div className={cn(
            "relative w-full overflow-hidden dark:bg-black/90",
            "backdrop-blur-sm",
            "p-4 relative rounded-[var(--theme-radius)] bg-[var(--theme-card-bg-light)] dark:bg-[var(--theme-card-bg-dark)] border-[length:var(--theme-border-size)] border-[var(--theme-border-light)] dark:border-[var(--theme-border-dark)] shadow-[6px_6px_0px_rgba(0,0,0,0.18)] dark:shadow-[6px_6px_0px_rgba(255,255,0,0.1)] transition-colors duration-300 backdrop-blur-sm"
        )}>
            {/* Background Blur Layer */}
            <div className="absolute inset-0 z-0 pointer-events-none">
                {activePost && displayUrl && (
                    <motion.img
                        key={currentImage?.blobs?.[0]?.ID || "bg"}
                        src={displayUrl}
                        initial={{ opacity: 0 }}
                        animate={{ opacity: 0.5 }}
                        transition={{ duration: 0.5 }}
                        className="h-full w-full object-cover blur-3xl opacity-10 dark:opacity-50"
                        alt=""
                    />
                )}
                <div className="absolute inset-0 bg-zinc-100/40 dark:bg-black/40" />
            </div>

            {/* Main Content */}
            <div className="relative z-10 flex flex-col p-4 box-border gap-4">
                <div className="flex items-center justify-between gap-3 shrink-0">
                    <div className="min-w-0">
                        <div className="truncate text-lg font-black uppercase tracking-wide text-zinc-800 dark:text-white drop-shadow-md">
                            {activePost?.title ?? "Untitled"}
                        </div>
                    </div>
                    <button type="button" onClick={onBack} className={cn(UI.button, UI.btnYellow)}>
                        Back
                    </button>
                </div>

                <motion.div
                    layout
                    className="relative mx-auto flex w-full max-w-7xl flex-1 flex-col items-center justify-center"
                >
                    <motion.div
                        layout
                        transition={{ layout: { type: "spring", stiffness: 300, damping: 30 } }}
                        className="relative w-full max-w-5xl overflow-hidden isolate transform-gpu transition-colors duration-300 md:p-4 md:rounded-(--theme-radius) md:bg-(--theme-card-bg-light) md:dark:bg-(--theme-card-bg-dark) md:border-(length:--theme-border-size) md:border-zinc-400 md:dark:border-zinc-800 md:shadow-[6px_6px_0px_rgba(0,0,0,0.18)] md:dark:shadow-[6px_6px_0px_rgba(255,255,0,0.1)] md:backdrop-blur-sm"
                        style={{ maskImage: "linear-gradient(white, white)", WebkitMaskImage: "linear-gradient(white, white)" }}
                    >
                        <div className="flex min-h-[50vh] w-full items-center justify-center relative md:bg-zinc-100 md:dark:bg-zinc-900/50 md:rounded-lg overflow-hidden">
                            <AnimatePresence initial={false} custom={direction} mode="popLayout">
                                {activePost ? (
                                    displayUrl ? (
                                        <motion.div
                                            key={currentIndex}
                                            custom={direction}
                                            variants={variants}
                                            initial="enter"
                                            animate="center"
                                            exit="exit"
                                            transition={{
                                                x: { type: "spring", stiffness: 300, damping: 30 },
                                                opacity: { duration: 0.2 }
                                            }}
                                            drag={hasMultipleImages ? "x" : false}
                                            dragConstraints={{ left: 0, right: 0 }}
                                            dragElastic={1}
                                            onDragEnd={handleDragEnd}
                                            className="flex w-full items-center justify-center cursor-pointer"
                                            onPointerDown={(e) => {
                                                dragStartX.current = e.clientX;
                                            }}
                                            onClick={(e) => {
                                                const dist = Math.abs(e.clientX - dragStartX.current);
                                                if (dist > 10) {
                                                    e.stopPropagation();
                                                    return;
                                                }
                                                setIsModalOpen(true);
                                            }}
                                        >
                                            {(() => {
                                                const contentType = currentImage?.blobs?.[0]?.contentType || "";
                                                const filename = currentImage?.blobs?.[0]?.filename || "";
                                                const isVideo = contentType.startsWith("video/") || /\.(mp4|webm|mov|mkv|avi)$/i.test(filename);

                                                return isVideo && canAccess ? (
                                                    <video
                                                        src={displayUrl}
                                                        controls
                                                        autoPlay
                                                        loop
                                                        className="max-h-[75vh] w-auto max-w-full rounded-lg object-contain shadow-lg pointer-events-none"
                                                    />
                                                ) : (
                                                    !canAccess ? (
                                                        <div className="w-full h-full min-h-[50vh] aspect-square overflow-hidden relative">
                                                            <div className="absolute inset-0 bg-zinc-900">
                                                                <ImageWithSpinner
                                                                    src={displayUrl}
                                                                    alt={activePost.title ?? ""}
                                                                    className="w-full h-full object-cover blur-xl scale-110 opacity-70"
                                                                    draggable={false}
                                                                />
                                                            </div>
                                                        </div>
                                                    ) : (
                                                        <ImageWithSpinner
                                                            src={displayUrl}
                                                            alt={activePost.title ?? ""}
                                                            className="rounded-lg shadow-lg max-h-[75vh] w-auto max-w-full object-contain"
                                                            draggable={false}
                                                        />
                                                    )
                                                );
                                            })()}
                                        </motion.div>
                                    ) : (
                                        <motion.div
                                            key="empty"
                                            initial={{ opacity: 0 }}
                                            animate={{ opacity: 1 }}
                                            exit={{ opacity: 0 }}
                                            className="flex h-full w-full items-center justify-center py-20"
                                        >
                                            {!canAccess ? (
                                                <div className="text-center">
                                                    <div className="text-2xl font-black uppercase text-white/50">🔒 Locked</div>
                                                    <div className="mt-2 text-sm font-bold text-white/40">{currentAccessLabel}</div>
                                                </div>
                                            ) : (
                                                <span className="text-white/50 font-bold">No media</span>
                                            )}
                                        </motion.div>
                                    )
                                ) : (
                                    <div className="flex h-full w-full items-center justify-center py-20 text-white/50 font-bold">
                                        No post selected
                                    </div>
                                )}
                            </AnimatePresence>

                            {/* Carousel Controls - Inside the container */}
                            {images.length > 1 && (
                                <>
                                    <button
                                        onClick={(e) => { e.stopPropagation(); paginate(-1); }}
                                        className="absolute left-4 top-1/2 -translate-y-1/2 rounded-full bg-black/50 p-3 text-white/80 backdrop-blur-md transition-all hover:bg-black/70 hover:text-white hover:scale-110 active:scale-95 hidden md:block border border-white/10 z-20"
                                    >
                                        <ChevronLeft className="h-8 w-8" />
                                    </button>
                                    <button
                                        onClick={(e) => { e.stopPropagation(); paginate(1); }}
                                        className="absolute right-4 top-1/2 -translate-y-1/2 rounded-full bg-black/50 p-3 text-white/80 backdrop-blur-md transition-all hover:bg-black/70 hover:text-white hover:scale-110 active:scale-95 hidden md:block border border-white/10 z-20"
                                    >
                                        <ChevronRight className="h-8 w-8" />
                                    </button>
                                </>
                            )}
                        </div>

                        {!canAccess && currentImage?.blobs?.[0]?.ID ? (
                            <div className="absolute inset-0 pointer-events-none z-20">
                                <LockedOverlay label={currentAccessLabel} />
                            </div>
                        ) : null}
                    </motion.div>
                </motion.div>

                {/* Film Strip */}
                {images.length > 1 && (
                    <div className="w-full overflow-hidden shrink-0">
                        <div className="flex w-full justify-center overflow-x-auto py-4 scrollbar-hide">
                            <div className="flex gap-3 px-4 w-fit mx-auto">
                                {images.map((img, idx) => {
                                    const blob = img.blobs?.[0];
                                    const thumbUrl = getUrl(blob?.ID, canAccess, "thumb");
                                    const isSelected = idx === currentIndex;
                                    const contentType = blob?.contentType || "";
                                    const isVideo = contentType.startsWith("video/");

                                    return (
                                        <button
                                            key={img.ID || idx}
                                            onClick={() => setIndex(idx)}
                                            className={cn(
                                                "relative h-16 w-16 shrink-0 overflow-hidden rounded-xl border-2 transition-all duration-300 ease-out",
                                                isSelected
                                                    ? "border-white scale-110 shadow-lg ring-2 ring-white/20 z-10"
                                                    : "border-white/10 opacity-60 hover:opacity-100 hover:scale-105 hover:border-white/50"
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
                                                <div className="flex h-full w-full items-center justify-center bg-zinc-800">
                                                    <span className="text-[10px] text-white/30">N/A</span>
                                                </div>
                                            )}
                                            {isVideo && (
                                                <div className="absolute inset-0 flex items-center justify-center pointer-events-none bg-black/20">
                                                    <div className="w-4 h-4 rounded-full bg-white/20 flex items-center justify-center backdrop-blur-sm">
                                                        <div className="w-0 h-0 border-t-[3px] border-t-transparent border-l-[5px] border-l-white border-b-[3px] border-b-transparent ml-0.5" />
                                                    </div>
                                                </div>
                                            )}
                                        </button>
                                    );
                                })}
                            </div>
                        </div>
                    </div>
                )}

                {activePost?.description ? (
                    <div className="mx-auto w-full max-w-2xl rounded-xl border border-white/10 bg-black/40 p-4 backdrop-blur-md shrink-0">
                        <div className="text-sm font-medium text-white/80 whitespace-pre-wrap">
                            {activePost.description}
                        </div>
                    </div>
                ) : null}
            </div>

            {/* Full Screen Modal */}
            {createPortal(
                <AnimatePresence>
                    {isModalOpen && displayUrl && (
                        <motion.div
                            initial={{ opacity: 0 }}
                            animate={{ opacity: 1 }}
                            exit={{ opacity: 0 }}
                            className="fixed inset-0 z-50 flex items-center justify-center bg-black/95 backdrop-blur-xl"
                            onClick={() => { setIsModalOpen(false); closeTimeout(); }}
                            onMouseMove={handleMouseMove}
                        >
                            <div className="absolute top-4 right-4 z-50">
                                <button
                                    onClick={() => setIsModalOpen(false)}
                                    className="p-2 text-white/70 hover:text-white hover:bg-white/10 rounded-full transition-colors"
                                >
                                    <CircleX className="h-6 w-6" />
                                </button>
                            </div>

                            <div
                                className="relative w-full h-full flex items-center justify-center p-4"
                            >
                                <AnimatePresence initial={false} custom={direction} mode="popLayout">
                                    <motion.div
                                        key={currentIndex}
                                        custom={direction}
                                        variants={variants}
                                        initial="enter"
                                        animate="center"
                                        exit="exit"
                                        transition={{
                                            x: { type: "spring", stiffness: 300, damping: 30 },
                                            opacity: { duration: 0.2 }
                                        }}
                                        drag={hasMultipleImages ? "x" : false}
                                        dragConstraints={{ left: 0, right: 0 }}
                                        dragElastic={1}
                                        onDragEnd={handleDragEnd}
                                        className="absolute inset-0 flex items-center justify-center p-4"
                                    >
                                        {(() => {
                                            const contentType = currentImage?.blobs?.[0]?.contentType || "";
                                            const filename = currentImage?.blobs?.[0]?.filename || "";
                                            const isVideo = contentType.startsWith("video/") || /\.(mp4|webm|mov|mkv|avi)$/i.test(filename);

                                            return isVideo && canAccess ? (
                                                <video
                                                    src={displayUrl}
                                                    controls
                                                    autoPlay
                                                    loop
                                                    className="max-h-full max-w-full rounded-md object-contain shadow-2xl"
                                                    onClick={(e) => e.stopPropagation()} // Prevent closing when clicking video controls
                                                />
                                            ) : (
                                                <img
                                                    src={displayUrl}
                                                    alt={activePost?.title ?? ""}
                                                    className="max-h-full max-w-full rounded-md object-contain shadow-2xl"
                                                    draggable={false}
                                                    onClick={(e) => e.stopPropagation()}
                                                />
                                            );
                                        })()}
                                    </motion.div>
                                </AnimatePresence>

                                {/* Modal Navigation Controls */}
                                {images.length > 1 && (
                                    <AnimatePresence>
                                        {showControls && (
                                            <>
                                                <motion.button
                                                    initial={{ opacity: 0, x: -20 }}
                                                    animate={{ opacity: 1, x: 0 }}
                                                    exit={{ opacity: 0, x: -20 }}
                                                    onClick={(e) => { e.stopPropagation(); paginate(-1); }}
                                                    className="absolute left-4 top-1/2 -translate-y-1/2 rounded-full bg-black/50 p-4 text-white/80 backdrop-blur-md transition-all hover:bg-black/70 hover:text-white hover:scale-110 hidden md:block active:scale-95 border border-white/10 z-50"
                                                >
                                                    <ChevronLeft className="h-8 w-8" />
                                                </motion.button>
                                                <motion.button
                                                    initial={{ opacity: 0, x: 20 }}
                                                    animate={{ opacity: 1, x: 0 }}
                                                    exit={{ opacity: 0, x: 20 }}
                                                    onClick={(e) => { e.stopPropagation(); paginate(1); }}
                                                    className="absolute right-4 top-1/2 -translate-y-1/2 rounded-full bg-black/50 p-4 text-white/80 backdrop-blur-md transition-all hover:bg-black/70 hover:text-white hover:scale-110 hidden md:block active:scale-95 border border-white/10 z-50"
                                                >
                                                    <ChevronRight className="h-8 w-8" />
                                                </motion.button>
                                            </>
                                        )}
                                    </AnimatePresence>
                                )}
                            </div>
                        </motion.div>
                    )}
                </AnimatePresence>,
                document.body
            )}
        </div>
    );
}
