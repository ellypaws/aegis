import React, { useState, useEffect, useMemo } from "react";
import { cn, safeRevoke } from "../lib/utils";
import { UI } from "../constants";
import { DiscordUser, Post, FileRef } from "../types";
import { MOCK_GUILD } from "../data/mock";
import { Patterns } from "./Patterns";
import { DropdownAddToList } from "./DropdownAddToList";
import { RolePill, ChannelPill } from "./Pills";

export function AuthorPanel({ user, onCreate }: { user: DiscordUser; onCreate: (post: Omit<Post, "id" | "createdAt">) => void }) {
    const [title, setTitle] = useState("");
    const [desc, setDesc] = useState("");
    const [tagsText, setTagsText] = useState("");
    const [allowedRoleIds, setAllowedRoleIds] = useState<string[]>(["r_tier1"]);
    const [channelIds, setChannelIds] = useState<string[]>(["c_art"]);

    const [fullFile, setFullFile] = useState<File | null>(null);
    const [thumbFile, setThumbFile] = useState<File | null>(null);

    const [previewFull, setPreviewFull] = useState<string | null>(null);
    const [previewThumb, setPreviewThumb] = useState<string | null>(null);

    useEffect(() => {
        if (previewFull) safeRevoke(previewFull);
        if (!fullFile) {
            setPreviewFull(null);
            return;
        }
        const u = URL.createObjectURL(fullFile);
        setPreviewFull(u);
        return () => safeRevoke(u);
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [fullFile]);

    useEffect(() => {
        if (previewThumb) safeRevoke(previewThumb);
        if (!thumbFile) {
            setPreviewThumb(null);
            return;
        }
        const u = URL.createObjectURL(thumbFile);
        setPreviewThumb(u);
        return () => safeRevoke(u);
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [thumbFile]);

    const tags = useMemo(() => {
        return tagsText
            .split(",")
            .map((s) => s.trim())
            .filter(Boolean)
            .slice(0, 12);
    }, [tagsText]);

    const canSubmit = !!fullFile && allowedRoleIds.length > 0;

    return (
        <div className={cn("relative overflow-hidden", UI.card)}>
            <Patterns.Polka color="rgba(255,0,0,0.08)" />
            <div className="pointer-events-none absolute top-[-16px] left-[-16px] h-24 w-24 rounded-full border-4 border-white bg-red-400 shadow-lg" />
            <div className="pointer-events-none absolute bottom-[-10px] right-[-10px] h-20 w-20 rotate-12 border-4 border-white bg-blue-400 shadow-lg" />

            <div className="relative z-10 p-5">
                <div className="flex items-start justify-between gap-4">
                    <div>
                        <div className="inline-block rounded-xl border-4 border-red-400 bg-white px-4 py-2 rotate-[2deg] shadow-[4px_4px_0px_rgba(248,113,113,1)]">
                            <div className="text-base font-black uppercase tracking-tight text-red-500">Author Upload</div>
                        </div>
                        <div className="mt-3 text-sm font-bold text-zinc-600">
                            Pick a <span className="text-zinc-900">full image</span> + optional <span className="text-zinc-900">thumbnail</span>. These file pickers are real.
                        </div>
                    </div>

                    <div className="rounded-2xl border-4 border-green-300 bg-white px-3 py-2 shadow-[4px_4px_0px_rgba(74,222,128,1)]">
                        <div className="text-[10px] font-black uppercase tracking-wide text-green-300">Logged in</div>
                        <div className="text-sm font-black text-green-700">{user.username}</div>
                    </div>
                </div>

                <div className="mt-5 grid gap-4 lg:grid-cols-2">
                    <div className="space-y-3">
                        <div className="grid gap-3 md:grid-cols-2">
                            <label className="space-y-1">
                                <div className={UI.label}>Full image *</div>
                                <input
                                    type="file"
                                    accept="image/*"
                                    onChange={(e) => setFullFile(e.target.files?.[0] ?? null)}
                                    className={cn(
                                        UI.input,
                                        "file:mr-3 file:rounded-xl file:border-4 file:border-blue-300 file:bg-blue-200 file:px-3 file:py-1.5 file:text-xs file:font-black file:uppercase file:tracking-wide file:text-blue-900"
                                    )}
                                />
                            </label>
                            <label className="space-y-1">
                                <div className={UI.label}>Thumbnail (optional)</div>
                                <input
                                    type="file"
                                    accept="image/*"
                                    onChange={(e) => setThumbFile(e.target.files?.[0] ?? null)}
                                    className={cn(
                                        UI.input,
                                        "file:mr-3 file:rounded-xl file:border-4 file:border-green-300 file:bg-green-200 file:px-3 file:py-1.5 file:text-xs file:font-black file:uppercase file:tracking-wide file:text-green-900"
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
                                <div className={UI.label}>Tags (comma-separated)</div>
                                <input value={tagsText} onChange={(e) => setTagsText(e.target.value)} placeholder="kemono, pink, sketch" className={UI.input} />
                            </label>
                        </div>

                        <label className="space-y-1">
                            <div className={UI.label}>Description (optional)</div>
                            <textarea value={desc} onChange={(e) => setDesc(e.target.value)} placeholder="Add context, credits, notes…" rows={4} className={cn(UI.input, "resize-none")} />
                        </label>

                        <DropdownAddToList
                            label="Allowed roles (tiers) *"
                            placeholder="Click to open role picker"
                            options={MOCK_GUILD.roles.filter((r) => r.id !== "r_author")}
                            selectedIds={allowedRoleIds}
                            onAdd={(id) => setAllowedRoleIds((s) => (s.includes(id) ? s : [...s, id]))}
                            onRemove={(id) => setAllowedRoleIds((s) => s.filter((x) => x !== id))}
                            renderSelected={(id) => <RolePill roleId={id} />}
                        />

                        <DropdownAddToList
                            label="Channels to post in"
                            placeholder="Click to open channel picker"
                            options={MOCK_GUILD.channels}
                            selectedIds={channelIds}
                            onAdd={(id) => setChannelIds((s) => (s.includes(id) ? s : [...s, id]))}
                            onRemove={(id) => setChannelIds((s) => s.filter((x) => x !== id))}
                            renderSelected={(id) => <ChannelPill channelId={id} />}
                        />

                        <div className="flex items-center justify-between gap-3 pt-2">
                            <div className="text-xs font-bold text-zinc-500">Full image required. Roles required. Thumbnail optional.</div>
                            <button
                                type="button"
                                disabled={!canSubmit}
                                onClick={() => {
                                    if (!fullFile) return;

                                    // Functional: store object URLs in the post itself.
                                    const full: FileRef = {
                                        name: fullFile.name,
                                        mime: fullFile.type || "image/*",
                                        url: URL.createObjectURL(fullFile),
                                        size: fullFile.size,
                                    };
                                    const thumb: FileRef | undefined = thumbFile
                                        ? {
                                            name: thumbFile.name,
                                            mime: thumbFile.type || "image/*",
                                            url: URL.createObjectURL(thumbFile),
                                            size: thumbFile.size,
                                        }
                                        : undefined;

                                    onCreate({
                                        title: title.trim() || undefined,
                                        description: desc.trim() || undefined,
                                        tags,
                                        allowedRoleIds,
                                        channelIds,
                                        full,
                                        thumb,
                                        authorId: user.id,
                                    });

                                    // Clear form. (previews will revoke via effects)
                                    setTitle("");
                                    setDesc("");
                                    setTagsText("");
                                    setAllowedRoleIds(["r_tier1"]);
                                    setChannelIds(["c_art"]);
                                    setFullFile(null);
                                    setThumbFile(null);
                                }}
                                className={cn(UI.button, UI.btnRed, !canSubmit && UI.btnDisabled)}
                            >
                                Create Post
                            </button>
                        </div>
                    </div>

                    <div className="space-y-3">
                        <div className="grid gap-3 md:grid-cols-2">
                            <div className={cn("p-3", UI.cardBlue)}>
                                <div className={UI.label}>Full Preview</div>
                                <div className="mt-2 aspect-square overflow-hidden rounded-2xl border-4 border-zinc-200 bg-zinc-50">
                                    {previewFull ? <img src={previewFull} className="h-full w-full object-cover" alt="" /> : <div className="flex h-full w-full items-center justify-center text-xs font-bold text-zinc-400">Pick a file</div>}
                                </div>
                                {fullFile ? <div className="mt-2 text-xs font-bold text-zinc-500">{fullFile.name}</div> : null}
                            </div>
                            <div className={cn("p-3", UI.cardGreen)}>
                                <div className={UI.label}>Thumbnail Preview</div>
                                <div className="mt-2 aspect-square overflow-hidden rounded-2xl border-4 border-zinc-200 bg-zinc-50">
                                    {previewThumb ? <img src={previewThumb} className="h-full w-full object-cover" alt="" /> : <div className="flex h-full w-full items-center justify-center text-xs font-bold text-zinc-400">Optional</div>}
                                </div>
                                {thumbFile ? <div className="mt-2 text-xs font-bold text-zinc-500">{thumbFile.name}</div> : null}
                            </div>
                        </div>

                        <div className={cn("p-4", UI.card)}>
                            <div className={UI.sectionTitle}>What gets posted (mock)</div>
                            <div className="mt-2 space-y-2 text-sm font-bold text-zinc-700">
                                <div>
                                    <span className="text-zinc-400">Guild:</span> {MOCK_GUILD.name}
                                </div>
                                <div className="flex flex-wrap gap-2">
                                    <span className="text-zinc-400">Channels:</span>
                                    {channelIds.length ? channelIds.map((c) => <ChannelPill key={c} channelId={c} />) : <span className="text-zinc-400">None</span>}
                                </div>
                                <div className="flex flex-wrap gap-2">
                                    <span className="text-zinc-400">Allowed roles:</span>
                                    {allowedRoleIds.length ? allowedRoleIds.map((r) => <RolePill key={r} roleId={r} />) : <span className="text-zinc-400">None</span>}
                                </div>
                            </div>
                            <div className="mt-3 text-xs font-bold text-zinc-400">Real app: post embeds + links per channel, and optionally DM purchasers.</div>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    );
}
