import React, { useState, useEffect, useMemo, useCallback } from "react";
import { cn, safeRevoke } from "../lib/utils";
import { UI } from "../constants";
import type { DiscordUser, Post } from "../types";
import { MOCK_GUILD } from "../data/mock";
import { Patterns } from "./Patterns";
import { DropdownAddToList } from "./DropdownAddToList";
import { RolePill, ChannelPill } from "./Pills";
import { ImagePlus } from "lucide-react";

const { useRef } = React;

/** Shared drop-zone overlay shown during drag */
function DropOverlay({ visible }: { visible: boolean }) {
    return (
        <div
            className={cn(
                "absolute inset-0 z-20 flex flex-col items-center justify-center rounded-2xl border-4 border-dashed transition-all duration-200 pointer-events-none",
                visible
                    ? "border-blue-400 bg-blue-50/80 opacity-100 scale-100"
                    : "border-transparent bg-transparent opacity-0 scale-95"
            )}
        >
            <ImagePlus className={cn("h-10 w-10 text-blue-400 transition-transform duration-200", visible && "animate-bounce")} />
            <span className="mt-1 text-sm font-black uppercase tracking-wide text-blue-500">Drop here</span>
        </div>
    );
}

function toDatetimeLocal(date: Date) {
    const pad = (n: number) => n.toString().padStart(2, "0");
    return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`;
}

function isImageFile(file: File): boolean {
    return file.type.startsWith("image/");
}

function isVideoFile(file: File): boolean {
    return file.type.startsWith("video/");
}

function isMediaFile(file: File): boolean {
    return isImageFile(file) || isVideoFile(file);
}

export type AuthorPanelPostInput = {
    title: string;
    description: string;
    allowedRoleIds: string[];
    channelIds: string[];
    image: File;
    thumbnail?: File;
    postDate: string;
    focusX?: number;
    focusY?: number;
};

export function AuthorPanel({ user, onCreate, editingPost, onUpdate, onCancelEdit }: {
    user: DiscordUser;
    onCreate?: (postInput: AuthorPanelPostInput) => void;
    editingPost?: Post | null;
    onUpdate?: (postKey: string, postInput: AuthorPanelPostInput & { image?: File }) => void;
    onCancelEdit?: () => void;
}) {
    const isEditing = !!editingPost;

    const [isOpen, setIsOpen] = useState(() => {
        try {
            const stored = localStorage.getItem("author_panel_open");
            return stored !== null ? JSON.parse(stored) : true;
        } catch {
            return true;
        }
    });

    useEffect(() => {
        localStorage.setItem("author_panel_open", JSON.stringify(isOpen));
    }, [isOpen]);

    const [title, setTitle] = useState("");
    const [desc, setDesc] = useState("");
    const [postDate, setPostDate] = useState(() => toDatetimeLocal(new Date()));
    const [allowedRoleIds, setAllowedRoleIds] = useState<string[]>(() => {
        try { return JSON.parse(localStorage.getItem("author_allowedRoleIds") || "[]"); } catch { return []; }
    });
    const [channelIds, setChannelIds] = useState<string[]>(() => {
        try { return JSON.parse(localStorage.getItem("author_channelIds") || "[]"); } catch { return []; }
    });

    const [fullFile, setFullFile] = useState<File | null>(null);
    const [thumbFile, setThumbFile] = useState<File | null>(null);

    const [previewFull, setPreviewFull] = useState<string | null>(null);
    const [previewThumb, setPreviewThumb] = useState<string | null>(null);
    const [isFullVideo, setIsFullVideo] = useState(false);
    const [isThumbVideo, setIsThumbVideo] = useState(false);

    const fullInputRef = useRef<HTMLInputElement>(null);
    const thumbInputRef = useRef<HTMLInputElement>(null);

    const [dragFull, setDragFull] = useState(false);
    const [dragThumb, setDragThumb] = useState(false);

    const [focusX, setFocusX] = useState(50);
    const [focusY, setFocusY] = useState(50);
    const fullPreviewRef = useRef<HTMLDivElement>(null);

    // Seed form fields when editingPost changes
    useEffect(() => {
        if (editingPost) {
            setIsOpen(true);
            setTitle(editingPost.title || "");
            setDesc(editingPost.description || "");
            setPostDate(editingPost.timestamp ? toDatetimeLocal(new Date(editingPost.timestamp)) : toDatetimeLocal(new Date()));
            setAllowedRoleIds(editingPost.allowedRoles?.map(r => r.id) || []);
            setChannelIds(editingPost.channelId ? editingPost.channelId.split(",").map(s => s.trim()).filter(Boolean) : []);
            setFocusX(editingPost.focusX ?? 50);
            setFocusY(editingPost.focusY ?? 50);
            setFullFile(null);
            setThumbFile(null);

            // Show the current media as preview
            const blob = editingPost.image?.blobs?.[0];
            if (blob) {
                const token = localStorage.getItem("jwt");
                setPreviewFull(`/images/${blob.ID}${token ? `?token=${token}` : ""}`);
                // Check if the blob is a video based on contentType or filename
                const isVideo = blob.contentType?.startsWith("video/") ||
                    blob.filename?.match(/\.(mp4|webm|mov|avi|mkv)$/i);
                setIsFullVideo(!!isVideo);
            } else {
                setPreviewFull(null);
                setIsFullVideo(false);
            }
            setPreviewThumb(null);
            setIsThumbVideo(false);
        }
    }, [editingPost]);

    const handleDrop = useCallback((e: React.DragEvent, setter: (f: File | null) => void, acceptVideo: boolean = false) => {
        e.preventDefault();
        e.stopPropagation();
        setDragFull(false);
        setDragThumb(false);
        const file = e.dataTransfer.files?.[0];
        if (file) {
            if (acceptVideo && isMediaFile(file)) {
                setter(file);
            } else if (!acceptVideo && isImageFile(file)) {
                setter(file);
            }
        }
    }, []);

    const handleDragOver = useCallback((e: React.DragEvent) => {
        e.preventDefault();
        e.stopPropagation();
    }, []);

    const [config, setConfig] = useState<{ roles: any[], channels: any[], guild_name?: string } | null>(null);

    // Persist allowedRoleIds and channelIds to localStorage (only in create mode)
    useEffect(() => {
        if (!isEditing) {
            localStorage.setItem("author_allowedRoleIds", JSON.stringify(allowedRoleIds));
        }
    }, [allowedRoleIds, isEditing]);

    useEffect(() => {
        if (!isEditing) {
            localStorage.setItem("author_channelIds", JSON.stringify(channelIds));
        }
    }, [channelIds, isEditing]);

    useEffect(() => {
        const token = localStorage.getItem("jwt");
        const headers: Record<string, string> = {};
        if (token) headers["Authorization"] = `Bearer ${token}`;

        fetch("/upload", { headers })
            .then(res => {
                if (res.ok) return res.json();
                throw new Error("Unauthorized or failed");
            })
            .then(data => setConfig(data))
            .catch(err => console.error("Failed to load upload config", err));
    }, []);

    useEffect(() => {
        // Don't revoke remote URLs (edit mode current-image preview)
        if (previewFull && previewFull.startsWith("blob:")) safeRevoke(previewFull);
        if (!fullFile) {
            // In edit mode, keep remote preview; in create mode clear
            if (!isEditing) {
                setPreviewFull(null);
                setIsFullVideo(false);
            }
            return;
        }
        const u = URL.createObjectURL(fullFile);
        setPreviewFull(u);
        setIsFullVideo(isVideoFile(fullFile));
        return () => safeRevoke(u);
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [fullFile]);

    useEffect(() => {
        if (previewThumb) safeRevoke(previewThumb);
        if (!thumbFile) {
            setPreviewThumb(null);
            setIsThumbVideo(false);
            return;
        }
        const u = URL.createObjectURL(thumbFile);
        setPreviewThumb(u);
        setIsThumbVideo(isVideoFile(thumbFile));
        return () => safeRevoke(u);
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [thumbFile]);

    // In create mode, full image required. In edit mode, image is optional (keep existing).
    const canSubmit = isEditing
        ? allowedRoleIds.length > 0
        : !!fullFile && allowedRoleIds.length > 0;

    const roleOptions = useMemo(() => {
        if (config?.roles) {
            return config.roles
                .filter((r: any) => r.name !== "@everyone" && !r.managed)
                .map((r: any) => ({ id: r.id, name: r.name, color: r.color }));
        }
        return MOCK_GUILD.roles.filter((r) => r.id !== "r_author").map(r => ({ id: r.id, name: r.name, color: r.color }));
    }, [config]);

    const channelOptions = useMemo(() => {
        if (config?.channels) {
            return config.channels.map((c: any) => ({ id: c.id, name: c.name }));
        }
        return MOCK_GUILD.channels;
    }, [config]);

    const handleSubmit = () => {
        if (isEditing && editingPost && onUpdate) {
            onUpdate(editingPost.postKey, {
                title: title.trim(),
                description: desc.trim(),
                allowedRoleIds,
                channelIds,
                image: fullFile!,
                thumbnail: thumbFile || undefined,
                postDate: postDate ? new Date(postDate).toISOString() : new Date().toISOString(),
                focusX,
                focusY,
            });
        } else if (onCreate && fullFile) {
            onCreate({
                title: title.trim(),
                description: desc.trim(),
                allowedRoleIds,
                channelIds,
                image: fullFile,
                thumbnail: thumbFile || undefined,
                postDate: postDate ? new Date(postDate).toISOString() : new Date().toISOString(),
                focusX,
                focusY,
            });

            // Clear form (keep roles & channels). Previews will revoke via effects.
            setTitle("");
            setDesc("");
            setPostDate(toDatetimeLocal(new Date()));
            setFullFile(null);
            setThumbFile(null);
            setFocusX(50);
            setFocusY(50);
            // Reset file input elements so displayed text clears
            if (fullInputRef.current) fullInputRef.current.value = "";
            if (thumbInputRef.current) thumbInputRef.current.value = "";
        }
    };

    return (
        <div className={cn("relative transition-all duration-300", UI.card, "backdrop-blur-sm")}>
            <div className="absolute inset-0 overflow-hidden rounded-[20px] pointer-events-none">
                <Patterns.Polka color={isEditing ? "rgba(59,130,246,0.08)" : "rgba(255,0,0,0.08)"} />
                <div className={cn("absolute top-[-16px] left-[-16px] h-24 w-24 rounded-full border-4 border-white shadow-lg", isEditing ? "bg-blue-400" : "bg-red-400")} />
                <div className={cn("absolute bottom-[-10px] right-[-10px] h-20 w-20 rotate-12 border-4 border-white shadow-lg", isEditing ? "bg-purple-400" : "bg-blue-400")} />
            </div>

            <div className="relative z-10 p-5">
                <div className="flex items-start justify-between gap-4">
                    <div className="flex items-center gap-3">
                        <div>
                            <div
                                onClick={() => setIsOpen(!isOpen)}
                                className={cn(
                                    "inline-block rounded-xl border-4 bg-white px-4 py-2 rotate-[2deg] cursor-pointer hover:scale-105 active:scale-95 transition-transform select-none",
                                    isEditing
                                        ? "border-blue-400 shadow-[4px_4px_0px_rgba(59,130,246,1)]"
                                        : "border-red-400 shadow-[4px_4px_0px_rgba(248,113,113,1)]"
                                )}>
                                <div className={cn(
                                    "text-base font-black uppercase tracking-tight",
                                    isEditing ? "text-blue-500" : "text-red-500"
                                )}>
                                    {isEditing ? "Edit Post" : "Author Upload"}
                                </div>
                            </div>
                            {isOpen && (
                                <div className="mt-3 text-sm font-bold text-zinc-600 dark:text-zinc-400">
                                    {isEditing ? (
                                        <>Editing <span className="text-zinc-900 dark:text-zinc-100">{editingPost?.title || "Untitled"}</span>. Update fields and save.</>
                                    ) : (
                                        <>Pick a <span className="text-zinc-900 dark:text-zinc-100">full media</span> + optional <span className="text-zinc-900 dark:text-zinc-100">thumbnail</span>.</>
                                    )}
                                </div>
                            )}
                        </div>
                    </div>

                    <div className="flex items-center gap-2">
                        {isEditing && onCancelEdit && (
                            <button
                                type="button"
                                onClick={onCancelEdit}
                                className={cn(UI.button, UI.btnYellow)}
                            >
                                Cancel
                            </button>
                        )}
                        <div className="rounded-2xl border-4 border-green-300 bg-white px-3 py-2 shadow-[4px_4px_0px_rgba(74,222,128,1)]">
                            <div className="text-[10px] font-black uppercase tracking-wide text-green-300">Logged in</div>
                            <div className="text-sm font-black text-green-700">{user.username}</div>
                        </div>
                    </div>
                </div>

                {isOpen && (
                    <div className="mt-5 grid gap-4 lg:grid-cols-2 animate-in fade-in slide-in-from-top-4 duration-300">
                        <div className="space-y-3">
                            <div className="grid gap-3 md:grid-cols-2">
                                <label
                                    className="space-y-1 block cursor-pointer group relative"
                                    onDragEnter={(e) => { e.preventDefault(); setDragFull(true); }}
                                    onDragOver={handleDragOver}
                                    onDragLeave={(e) => { e.preventDefault(); setDragFull(false); }}
                                    onDrop={(e) => handleDrop(e, setFullFile, true)}
                                >
                                    <DropOverlay visible={dragFull} />
                                    <div className={UI.label}>{isEditing ? "Replace media (optional)" : "Full media *"}</div>
                                    <input
                                        ref={fullInputRef}
                                        type="file"
                                        accept="image/*,video/*"
                                        onChange={(e) => setFullFile(e.target.files?.[0] ?? null)}
                                        className={cn(
                                            UI.input,
                                            "file:mr-3 file:rounded-xl file:border-4 file:border-blue-300 file:bg-blue-200 file:px-3 file:py-1.5 file:text-xs file:font-black file:uppercase file:tracking-wide file:text-blue-900",
                                            "file:cursor-pointer file:transition-all file:duration-200 file:ease-out",
                                            "file:hover:scale-105 file:hover:brightness-110 file:hover:shadow-[0_4px_12px_rgba(147,197,253,0.5)]",
                                            "file:active:scale-95",
                                            "cursor-pointer"
                                        )}
                                    />
                                </label>
                                <label
                                    className="space-y-1 block cursor-pointer group relative"
                                    onDragEnter={(e) => { e.preventDefault(); setDragThumb(true); }}
                                    onDragOver={handleDragOver}
                                    onDragLeave={(e) => { e.preventDefault(); setDragThumb(false); }}
                                    onDrop={(e) => handleDrop(e, setThumbFile, false)}
                                >
                                    <DropOverlay visible={dragThumb} />
                                    <div className={UI.label}>Thumbnail (optional)</div>
                                    <input
                                        ref={thumbInputRef}
                                        type="file"
                                        accept="image/*"
                                        onChange={(e) => setThumbFile(e.target.files?.[0] ?? null)}
                                        className={cn(
                                            UI.input,
                                            "file:mr-3 file:rounded-xl file:border-4 file:border-green-300 file:bg-green-200 file:px-3 file:py-1.5 file:text-xs file:font-black file:uppercase file:tracking-wide file:text-green-900",
                                            "file:cursor-pointer file:transition-all file:duration-200 file:ease-out",
                                            "file:hover:scale-105 file:hover:brightness-110 file:hover:shadow-[0_4px_12px_rgba(134,239,172,0.5)]",
                                            "file:active:scale-95",
                                            "cursor-pointer"
                                        )}
                                    />
                                </label>
                            </div>

                            <div className="grid gap-3 md:grid-cols-2">
                                <label className="space-y-1">
                                    <div className={UI.label}>Title (optional)</div>
                                    <input value={title} onChange={(e) => setTitle(e.target.value)} placeholder="e.g. Valentines set" className={UI.input} />
                                </label>

                                <label className="space-y-1">
                                    <div className={UI.label}>Post date</div>
                                    <input type="datetime-local" value={postDate} onChange={(e) => setPostDate(e.target.value)} className={UI.input} />
                                </label>
                            </div>

                            <label className="space-y-1">
                                <div className={UI.label}>Description (optional)</div>
                                <textarea value={desc} onChange={(e) => setDesc(e.target.value)} placeholder="Add context, credits, notes…" rows={4} className={cn(UI.input, "resize-none")} />
                            </label>

                            <DropdownAddToList
                                label="Allowed roles (tiers) *"
                                placeholder="Click to open role picker"
                                options={roleOptions}
                                selectedIds={allowedRoleIds}
                                onAdd={(id) => setAllowedRoleIds((s) => (s.includes(id) ? s : [...s, id]))}
                                onRemove={(id) => setAllowedRoleIds((s) => s.filter((x) => x !== id))}
                                renderSelected={(id) => {
                                    const r = roleOptions.find(o => o.id === id);
                                    return <RolePill name={r?.name || id} color={r?.color} />;
                                }}
                            />

                            <DropdownAddToList
                                label="Channels to post in"
                                placeholder="Click to open channel picker"
                                options={channelOptions}
                                selectedIds={channelIds}
                                onAdd={(id) => setChannelIds((s) => (s.includes(id) ? s : [...s, id]))}
                                onRemove={(id) => setChannelIds((s) => s.filter((x) => x !== id))}
                                renderSelected={(id) => {
                                    const c = channelOptions.find(o => o.id === id);
                                    return <ChannelPill name={c?.name || id} />;
                                }}
                            />

                            <div className="flex items-center justify-between gap-3 pt-2">
                                <div className="text-xs font-bold text-zinc-500">
                                    {isEditing
                                        ? "Roles required. New media optional (keeps current)."
                                        : "Full media required. Roles required. Thumbnail optional."}
                                </div>
                                <button
                                    type="button"
                                    disabled={!canSubmit}
                                    onClick={handleSubmit}
                                    className={cn(
                                        UI.button,
                                        isEditing ? UI.btnBlue : UI.btnRed,
                                        !canSubmit && UI.btnDisabled
                                    )}
                                >
                                    {isEditing ? "Save Changes" : "Create Post"}
                                </button>
                            </div>
                        </div>

                        <div className="space-y-3">
                            <div className="grid gap-3 md:grid-cols-2">
                                <div
                                    className={cn(
                                        "p-3 transition relative",
                                        UI.cardBlue,
                                        previewFull && !isFullVideo ? "cursor-crosshair" : "cursor-pointer hover:opacity-90 active:scale-95"
                                    )}
                                    onClick={() => { if (!previewFull) fullInputRef.current?.click(); }}
                                    onDragEnter={(e) => { if (!previewFull) { e.preventDefault(); setDragFull(true); } }}
                                    onDragOver={(e) => { if (!previewFull) handleDragOver(e); }}
                                    onDragLeave={(e) => { if (!previewFull) { e.preventDefault(); setDragFull(false); } }}
                                    onDrop={(e) => { if (!previewFull) handleDrop(e, setFullFile, true); }}
                                >
                                    {!previewFull && <DropOverlay visible={dragFull} />}
                                    <div className={UI.label}>Full Preview</div>
                                    <div
                                        ref={fullPreviewRef}
                                        className={cn(
                                            "mt-2 aspect-square overflow-hidden rounded-2xl border-4 border-zinc-200 bg-zinc-50 relative select-none",
                                            previewFull && !isFullVideo ? "cursor-crosshair" : "cursor-pointer"
                                        )}
                                        onMouseDown={(e) => {
                                            const rect = fullPreviewRef.current?.getBoundingClientRect();
                                            if (!rect || !previewFull || isFullVideo) return;
                                            e.stopPropagation();
                                            const x = Math.min(100, Math.max(0, ((e.clientX - rect.left) / rect.width) * 100));
                                            const y = Math.min(100, Math.max(0, ((e.clientY - rect.top) / rect.height) * 100));
                                            setFocusX(Math.round(x * 10) / 10);
                                            setFocusY(Math.round(y * 10) / 10);

                                            const onMove = (ev: MouseEvent) => {
                                                const mx = Math.min(100, Math.max(0, ((ev.clientX - rect.left) / rect.width) * 100));
                                                const my = Math.min(100, Math.max(0, ((ev.clientY - rect.top) / rect.height) * 100));
                                                setFocusX(Math.round(mx * 10) / 10);
                                                setFocusY(Math.round(my * 10) / 10);
                                            };
                                            const onUp = () => {
                                                window.removeEventListener("mousemove", onMove);
                                                window.removeEventListener("mouseup", onUp);
                                            };
                                            window.addEventListener("mousemove", onMove);
                                            window.addEventListener("mouseup", onUp);
                                        }}
                                        onClick={(e) => { if (previewFull) e.stopPropagation(); }}
                                    >
                                        {previewFull ? (
                                            <>
                                                {isFullVideo ? (
                                                    <video
                                                        src={previewFull}
                                                        className="h-full w-full object-cover"
                                                        controls
                                                        controlsList="nodownload"
                                                        playsInline
                                                    />
                                                ) : (
                                                    <>
                                                        <img src={previewFull} className="h-full w-full object-cover" style={{ objectPosition: `${focusX}% ${focusY}%` }} alt="" draggable={false} />
                                                        {/* Focus point marker */}
                                                        <div
                                                            className="absolute pointer-events-none"
                                                            style={{ left: `${focusX}%`, top: `${focusY}%`, transform: "translate(-50%, -50%)" }}
                                                        >
                                                            <div className="h-8 w-8 rounded-full bg-zinc-500/30 border-2 border-zinc-400/50 shadow-lg flex items-center justify-center backdrop-blur-[1px]">
                                                                <div className="h-2.5 w-2.5 rounded-full bg-zinc-700/80 shadow" />
                                                            </div>
                                                        </div>
                                                    </>
                                                )}
                                            </>
                                        ) : (
                                            <div className="flex h-full w-full items-center justify-center text-xs font-bold text-zinc-400">
                                                {isEditing ? "Current media" : "Pick a file"}
                                            </div>
                                        )}
                                    </div>
                                    {previewFull && !isFullVideo && (
                                        <div className="mt-1 text-[10px] font-bold text-zinc-400 tabular-nums">
                                            Focus: {focusX.toFixed(1)}% × {focusY.toFixed(1)}%
                                        </div>
                                    )}
                                    {fullFile ? <div className="mt-2 text-xs font-bold text-zinc-500 dark:text-zinc-400">{fullFile.name}</div> : null}
                                </div>
                                <div
                                    className={cn("p-3 cursor-pointer hover:opacity-90 active:scale-95 transition relative", UI.cardGreen)}
                                    onClick={() => thumbInputRef.current?.click()}
                                    onDragEnter={(e) => { e.preventDefault(); setDragThumb(true); }}
                                    onDragOver={handleDragOver}
                                    onDragLeave={(e) => { e.preventDefault(); setDragThumb(false); }}
                                    onDrop={(e) => handleDrop(e, setThumbFile, false)}
                                >
                                    <DropOverlay visible={dragThumb} />
                                    <div className={UI.label}>Thumbnail Preview</div>
                                    <div className="mt-2 aspect-square overflow-hidden rounded-2xl border-4 border-zinc-200 bg-zinc-50">
                                        {previewThumb ? (
                                            isThumbVideo ? (
                                                <video
                                                    src={previewThumb}
                                                    className="h-full w-full object-cover"
                                                    controls
                                                    controlsList="nodownload"
                                                    playsInline
                                                />
                                            ) : (
                                                <img src={previewThumb} className="h-full w-full object-cover" alt="" />
                                            )
                                        ) : (
                                            <div className="flex h-full w-full items-center justify-center text-xs font-bold text-zinc-400">Optional</div>
                                        )}
                                    </div>
                                    {thumbFile ? <div className="mt-2 text-xs font-bold text-zinc-500 dark:text-zinc-400">{thumbFile.name}</div> : null}
                                </div>
                            </div>

                            <div className={cn("p-4", UI.card)}>
                                <div className={UI.sectionTitle}>{isEditing ? "Updated values" : "What gets posted"}</div>
                                <div className="mt-2 space-y-2 text-sm font-bold text-zinc-700 dark:text-zinc-300">
                                    <div>
                                        <span className="text-zinc-400">Guild:</span> {config?.guild_name || MOCK_GUILD.name}
                                    </div>
                                    <div className="flex flex-wrap gap-2">
                                        <span className="text-zinc-400">Channels:</span>
                                        {channelIds.length ? channelIds.map((c) => {
                                            const ch = channelOptions.find(o => o.id === c);
                                            return <ChannelPill key={c} name={ch?.name || c} />;
                                        }) : <span className="text-zinc-400">None</span>}
                                    </div>
                                    <div className="flex flex-wrap gap-2">
                                        <span className="text-zinc-400">Allowed roles:</span>
                                        {allowedRoleIds.length ? allowedRoleIds.map((r) => {
                                            const ro = roleOptions.find(o => o.id === r);
                                            return <RolePill key={r} name={ro?.name || r} color={ro?.color} />;
                                        }) : <span className="text-zinc-400">None</span>}
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>
                )}
            </div>
        </div>
    );
}
