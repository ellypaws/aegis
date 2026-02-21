import React, { useState, useEffect, useMemo, useCallback } from "react";
import { cn, safeRevoke } from "../lib/utils";
import { UI } from "../constants";
import type { DiscordUser, Post } from "../types";
import { Patterns } from "./Patterns";
import { DropdownAddToList } from "./DropdownAddToList";
import { RolePill, ChannelPill } from "./Pills";
import {
  ImagePlus,
  X,
  FileVideo,
  ChevronLeft,
  ChevronRight,
} from "lucide-react";

const { useRef } = React;

/** Shared drop-zone overlay shown during drag */
function DropOverlay({ visible }: { visible: boolean }) {
  return (
    <div
      className={cn(
        "absolute inset-0 z-20 flex flex-col items-center justify-center rounded-2xl border-4 border-dashed transition-all duration-200 pointer-events-none",
        visible
          ? "border-blue-400 bg-blue-50/80 opacity-100 scale-100"
          : "border-transparent bg-transparent opacity-0 scale-95",
      )}
    >
      <ImagePlus
        className={cn(
          "h-10 w-10 text-blue-400 transition-transform duration-200",
          visible && "animate-bounce",
        )}
      />
      <span className="mt-1 text-sm font-black uppercase tracking-wide text-blue-500">
        Drop here
      </span>
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

function normalizeIdList(ids: Array<string | null | undefined>): string[] {
  const seen = new Set<string>();
  const normalized: string[] = [];
  ids.forEach((value) => {
    const id = String(value ?? "").trim();
    if (!id || seen.has(id)) return;
    seen.add(id);
    normalized.push(id);
  });
  return normalized;
}

export type AuthorPanelPostInput = {
  title: string;
  description: string;
  allowedRoleIds: string[];
  channelIds: string[];
  images: File[];
  thumbnail?: File;
  removeImageIds?: number[];
  mediaOrder?: string[];
  clearThumbnail?: boolean;
  postDate: string;
  focusX?: number;
  focusY?: number;
};

export function AuthorPanel({
  user,
  onCreate,
  editingPost,
  onUpdate,
  onCancelEdit,
  availableRoles,
  availableChannels,
  guildName,
}: {
  user: DiscordUser;
  onCreate?: (postInput: AuthorPanelPostInput) => Promise<boolean | void> | boolean | void;
  editingPost?: Post | null;
  onUpdate?: (
    postKey: string,
    postInput: Omit<AuthorPanelPostInput, "images"> & {
      images?: File[];
      removeImageIds?: number[];
      mediaOrder?: string[];
      clearThumbnail?: boolean;
    },
  ) => Promise<boolean | void> | boolean | void;
  onCancelEdit?: () => void;
  availableRoles?: Array<{
    id: string;
    name: string;
    color?: number;
    managed?: boolean;
  }>;
  availableChannels?: Array<{ id: string; name: string }>;
  guildName?: string;
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
    try {
      return JSON.parse(localStorage.getItem("author_allowedRoleIds") || "[]");
    } catch {
      return [];
    }
  });
  const [channelIds, setChannelIds] = useState<string[]>(() => {
    try {
      return JSON.parse(localStorage.getItem("author_channelIds") || "[]");
    } catch {
      return [];
    }
  });

  // Multi-file state
  type NewMediaItem = { key: string; file: File };
  const [newMediaItems, setNewMediaItems] = useState<NewMediaItem[]>([]);
  const [mediaOrder, setMediaOrder] = useState<string[]>([]);
  const [removedRemoteImageIds, setRemovedRemoteImageIds] = useState<number[]>(
    [],
  );

  const [thumbFile, setThumbFile] = useState<File | null>(null);
  const [existingThumbUrl, setExistingThumbUrl] = useState<string | null>(null);
  const [clearThumbnail, setClearThumbnail] = useState(false);

  // Previews
  type PreviewItem = { key: string; url: string; isVideo: boolean; file: File };
  const [filePreviews, setFilePreviews] = useState<PreviewItem[]>([]);

  // Remote previews for editing mode (existing images)
  type RemotePreview = {
    url: string;
    isVideo: boolean;
    id: number;
    blobId?: number;
    hasThumbnail?: boolean;
  };
  const [remotePreviews, setRemotePreviews] = useState<RemotePreview[]>([]);

  const [previewThumb, setPreviewThumb] = useState<string | null>(null);
  const [isThumbVideo, setIsThumbVideo] = useState(false);

  const fullInputRef = useRef<HTMLInputElement>(null);
  const thumbInputRef = useRef<HTMLInputElement>(null);

  const [dragFull, setDragFull] = useState(false);
  const [dragThumb, setDragThumb] = useState(false);

  const [focusX, setFocusX] = useState(50);
  const [focusY, setFocusY] = useState(50);
  const fullPreviewRef = useRef<HTMLDivElement>(null);
  const [previewSize, setPreviewSize] = useState(() => {
    try {
      const stored = localStorage.getItem("author_media_preview_size");
      const parsed = stored ? Number(stored) : NaN;
      if (Number.isFinite(parsed)) {
        return Math.min(160, Math.max(56, parsed));
      }
    } catch {
      // ignore storage parse errors
    }
    return 64;
  });
  const [draggedMediaToken, setDraggedMediaToken] = useState<string | null>(
    null,
  );

  const selectedFiles = useMemo(
    () => newMediaItems.map((m) => m.file),
    [newMediaItems],
  );

  useEffect(() => {
    localStorage.setItem("author_media_preview_size", String(previewSize));
  }, [previewSize]);

  // Seed form fields when editingPost changes
  useEffect(() => {
    if (editingPost) {
      setIsOpen(true);
      setTitle(editingPost.title || "");
      setDesc(editingPost.description || "");
      setPostDate(
        editingPost.timestamp
          ? toDatetimeLocal(new Date(editingPost.timestamp))
          : toDatetimeLocal(new Date()),
      );
      setAllowedRoleIds(
        normalizeIdList(
          (editingPost.allowedRoles || []).map(
            (r) => r?.id || (r as unknown as { roleId?: string })?.roleId,
          ),
        ),
      );
      setChannelIds(
        editingPost.channelId
          ? editingPost.channelId
            .split(",")
            .map((s) => s.trim())
            .filter(Boolean)
          : [],
      );
      setFocusX(editingPost.focusX ?? 50);
      setFocusY(editingPost.focusY ?? 50);
      setNewMediaItems([]);
      setMediaOrder([]);
      setRemovedRemoteImageIds([]);
      setThumbFile(null);
      setClearThumbnail(false);

      // Show current media as remote previews
      const token = localStorage.getItem("jwt");
      const remotes: RemotePreview[] = [];

      if (editingPost.images && editingPost.images.length > 0) {
        editingPost.images.forEach((img) => {
          const blob = img.blobs?.[0];
          if (blob) {
            const isVideo =
              blob.contentType?.startsWith("video/") ||
              blob.filename?.match(/\.(mp4|webm|mov|avi|mkv)$/i);
            remotes.push({
              url: `/images/${blob.ID}${token ? `?token=${token}` : ""}`,
              isVideo: !!isVideo,
              id: img.ID,
              blobId: blob.ID,
              hasThumbnail: !!img.hasThumbnail,
            });
          }
        });
      }
      setRemotePreviews(remotes);
      setMediaOrder(remotes.map((r) => `e:${r.id}`));

      const firstWithThumb = remotes.find((r) => r.hasThumbnail && !!r.blobId);
      if (firstWithThumb?.blobId) {
        setExistingThumbUrl(
          `/thumb/${firstWithThumb.blobId}${token ? `?token=${token}` : ""}`,
        );
      } else {
        setExistingThumbUrl(null);
      }

      setPreviewThumb(null);
      setIsThumbVideo(false);
    }
  }, [editingPost]);

  // Handle file changes -> generate previews
  useEffect(() => {
    const newPreviews: PreviewItem[] = [];
    newMediaItems.forEach((item) => {
      const file = item.file;
      if (isMediaFile(file)) {
        newPreviews.push({
          key: item.key,
          url: URL.createObjectURL(file),
          isVideo: isVideoFile(file),
          file: file,
        });
      }
    });
    setFilePreviews(newPreviews);

    return () => {
      newPreviews.forEach((p) => safeRevoke(p.url));
    };
  }, [newMediaItems]);

  // Cleanup thumb
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

  const handleDrop = useCallback(
    (
      e: React.DragEvent,
      setter: (files: File[]) => void,
      acceptVideo: boolean = false,
    ) => {
      e.preventDefault();
      e.stopPropagation();
      setDragFull(false);
      setDragThumb(false);

      const droppedFiles = Array.from(e.dataTransfer.files);
      if (droppedFiles.length > 0) {
        const validFiles = droppedFiles.filter((f) =>
          acceptVideo ? isMediaFile(f) : isImageFile(f),
        );
        if (validFiles.length > 0) {
          setter(validFiles);
        }
      }
    },
    [],
  );

  // For thumbnail (single file)
  const handleThumbDrop = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setDragThumb(false);
    const file = e.dataTransfer.files?.[0];
    if (file && isImageFile(file)) {
      setThumbFile(file);
      setClearThumbnail(false);
    }
  }, []);

  const handleDragOver = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
  }, []);

  // Persist allowedRoleIds and channelIds to localStorage (only in create mode)
  useEffect(() => {
    if (!isEditing) {
      localStorage.setItem(
        "author_allowedRoleIds",
        JSON.stringify(allowedRoleIds),
      );
    }
  }, [allowedRoleIds, isEditing]);

  useEffect(() => {
    if (!isEditing) {
      localStorage.setItem("author_channelIds", JSON.stringify(channelIds));
    }
  }, [channelIds, isEditing]);

  const availableRemoteIds = useMemo(
    () => new Set(remotePreviews.map((p) => p.id)),
    [remotePreviews],
  );
  const availableNewKeys = useMemo(
    () => new Set(newMediaItems.map((p) => p.key)),
    [newMediaItems],
  );
  const effectiveMediaCount = useMemo(() => {
    let count = 0;
    mediaOrder.forEach((token) => {
      if (token.startsWith("e:")) {
        const parsed = Number(token.slice(2));
        if (Number.isFinite(parsed) && availableRemoteIds.has(parsed))
          count += 1;
        return;
      }
      if (token.startsWith("n:")) {
        const key = token.slice(2);
        if (availableNewKeys.has(key)) count += 1;
      }
    });
    return count;
  }, [mediaOrder, availableRemoteIds, availableNewKeys]);

  // Validation
  const canSubmit = isEditing
    ? effectiveMediaCount > 0
    : selectedFiles.length > 0;

  const roleOptions = useMemo(() => {
    if (availableRoles && availableRoles.length > 0) {
      return availableRoles
        .filter((r: any) => r.name !== "@everyone" && !r.managed)
        .map((r: any) => ({ id: r.id, name: r.name, color: r.color }));
    }
    return [];
  }, [availableRoles]);

  const channelOptions = useMemo(() => {
    if (availableChannels && availableChannels.length > 0) {
      return availableChannels.map((c: any) => ({ id: c.id, name: c.name }));
    }
    return [];
  }, [availableChannels]);

  const handleSubmit = async () => {
    const normalizedRoleIds = normalizeIdList(allowedRoleIds);

    const validOrderedTokens = mediaOrder.filter((token) => {
      if (token.startsWith("e:")) {
        const parsed = Number(token.slice(2));
        return (
          Number.isFinite(parsed) && remotePreviews.some((r) => r.id === parsed)
        );
      }
      if (token.startsWith("n:")) {
        const key = token.slice(2);
        return newMediaItems.some((item) => item.key === key);
      }
      return false;
    });

    const newKeysInOrder: string[] = [];
    validOrderedTokens.forEach((token) => {
      if (!token.startsWith("n:")) return;
      const key = token.slice(2);
      if (!newKeysInOrder.includes(key)) {
        newKeysInOrder.push(key);
      }
    });

    const orderedNewFiles = newKeysInOrder
      .map((key) => newMediaItems.find((item) => item.key === key)?.file)
      .filter((file): file is File => !!file);

    const newIndexByKey = new Map<string, number>();
    newKeysInOrder.forEach((key, idx) => newIndexByKey.set(key, idx));
    const mediaOrderForBackend = validOrderedTokens
      .map((token) => {
        if (token.startsWith("e:")) return token;
        if (token.startsWith("n:")) {
          const idx = newIndexByKey.get(token.slice(2));
          if (idx !== undefined) return `n:${idx}`;
        }
        return "";
      })
      .filter(Boolean);

    const payload: AuthorPanelPostInput = {
      title: title.trim(),
      description: desc.trim(),
      allowedRoleIds: normalizedRoleIds,
      channelIds,
      images: isEditing ? orderedNewFiles : selectedFiles,
      thumbnail: thumbFile || undefined,
      removeImageIds: isEditing ? removedRemoteImageIds : undefined,
      mediaOrder: isEditing ? mediaOrderForBackend : undefined,
      clearThumbnail: isEditing ? clearThumbnail && !thumbFile : undefined,
      postDate: postDate
        ? new Date(postDate).toISOString()
        : new Date().toISOString(),
      focusX,
      focusY,
    };

    if (isEditing && editingPost && onUpdate) {
      // Fix type compatibility: explicit cast or fresh object
      const updatePayload: Omit<AuthorPanelPostInput, "images"> & {
        images?: File[];
      } = {
        ...payload,
        images: payload.images.length > 0 ? payload.images : undefined,
      };
      await onUpdate(editingPost.postKey, updatePayload);
    } else if (onCreate && selectedFiles.length > 0) {
      const success = await onCreate(payload);
      if (success === false) return;
      // Reset form
      setTitle("");
      setDesc("");
      setPostDate(toDatetimeLocal(new Date()));
      setNewMediaItems([]);
      setMediaOrder([]);
      setRemovedRemoteImageIds([]);
      setThumbFile(null);
      setExistingThumbUrl(null);
      setClearThumbnail(false);
      setFocusX(50);
      setFocusY(50);
      setRemotePreviews([]);
      if (fullInputRef.current) fullInputRef.current.value = "";
      if (thumbInputRef.current) thumbInputRef.current.value = "";
    }
  };

  const addFiles = (files: File[]) => {
    const accepted = files.filter(isMediaFile);
    if (accepted.length === 0) return;
    const addedItems = accepted.map((file, idx) => ({
      key: `${Date.now()}-${idx}-${Math.random().toString(36).slice(2, 10)}`,
      file,
    }));
    setNewMediaItems((prev) => [...prev, ...addedItems]);
    setMediaOrder((prev) => [
      ...prev,
      ...addedItems.map((item) => `n:${item.key}`),
    ]);
  };

  const removeMediaToken = (token: string) => {
    if (token.startsWith("e:")) {
      const parsed = Number(token.slice(2));
      if (Number.isFinite(parsed)) {
        const remoteID = parsed as number;
        setRemotePreviews((prev) =>
          prev.filter((item) => item.id !== remoteID),
        );
        setRemovedRemoteImageIds((prev) =>
          prev.includes(remoteID) ? prev : [...prev, remoteID],
        );
      }
    }
    if (token.startsWith("n:")) {
      const key = token.slice(2);
      setNewMediaItems((prev) => prev.filter((item) => item.key !== key));
    }
    setMediaOrder((prev) => prev.filter((item) => item !== token));
  };

  const moveMediaToken = (token: string, direction: -1 | 1) => {
    setMediaOrder((prev) => {
      const idx = prev.indexOf(token);
      if (idx < 0) return prev;
      const target = idx + direction;
      if (target < 0 || target >= prev.length) return prev;
      const next = [...prev];
      const tmp = next[idx];
      next[idx] = next[target];
      next[target] = tmp;
      return next;
    });
  };

  const reorderMediaToken = (sourceToken: string, targetToken: string) => {
    if (!sourceToken || !targetToken || sourceToken === targetToken) return;
    setMediaOrder((prev) => {
      const sourceIndex = prev.indexOf(sourceToken);
      const targetIndex = prev.indexOf(targetToken);
      if (sourceIndex < 0 || targetIndex < 0 || sourceIndex === targetIndex)
        return prev;
      const next = [...prev];
      const [moved] = next.splice(sourceIndex, 1);
      next.splice(targetIndex, 0, moved);
      return next;
    });
  };

  const handleCancelEdit = () => {
    setNewMediaItems([]);
    setMediaOrder([]);
    setRemovedRemoteImageIds([]);
    setThumbFile(null);
    setRemotePreviews([]);
    setExistingThumbUrl(null);
    setClearThumbnail(false);
    if (fullInputRef.current) fullInputRef.current.value = "";
    if (thumbInputRef.current) thumbInputRef.current.value = "";
    onCancelEdit?.();
  };

  // Determine what to show in the main preview area
  // If editing and no new files, show existing remotes.
  // If new files selected, show new files.
  // Actually, maybe show a grid/carousel?
  // To keep it simple, we'll show the FIRST item in the big box for focus, and a strip below for all items.

  const activePreview = useMemo(() => {
    const first = mediaOrder[0];
    if (!first) return null;
    if (first.startsWith("n:")) {
      const key = first.slice(2);
      const found = filePreviews.find((p) => p.key === key);
      if (found) return { ...found, isRemote: false };
      return null;
    }
    if (first.startsWith("e:")) {
      const id = Number(first.slice(2));
      const found = remotePreviews.find((p) => p.id === id);
      if (found) return { ...found, isRemote: true };
    }
    return null;
  }, [filePreviews, remotePreviews, mediaOrder]);

  const focusPreview = useMemo(() => {
    const explicitThumbUrl =
      previewThumb || (!clearThumbnail && existingThumbUrl);
    if (explicitThumbUrl) {
      return {
        url: explicitThumbUrl,
        isVideo: false,
        isRemote: !!existingThumbUrl && !previewThumb,
        isThumbnail: true,
      };
    }
    if (!activePreview) return null;
    return { ...activePreview, isThumbnail: false };
  }, [activePreview, clearThumbnail, existingThumbUrl, previewThumb]);

  // We only support focus picking on the VERY FIRST image for now (to keep UI simple and consistent with backend focus field which is per post, not per image yet?)
  // Wait, backend has focusX/focusY per POST, but we have multiple images now.
  // Does the focus apply to the first image (cover)?
  // Yes, let's assume focus applies to the cover image.

  return (
    <div
      className={cn(
        "relative transition-all duration-300",
        UI.card,
        "backdrop-blur-sm",
      )}
    >
      <div className="absolute inset-0 overflow-hidden rounded-[20px] pointer-events-none">
        <Patterns.Polka
          color={isEditing ? "rgba(59,130,246,0.08)" : "rgba(255,0,0,0.08)"}
        />
        <div
          className={cn(
            "absolute top-[-16px] left-[-16px] h-24 w-24 rounded-full border-4 border-white shadow-lg",
            isEditing ? "bg-blue-400" : "bg-red-400",
          )}
        />
        <div
          className={cn(
            "absolute bottom-[-10px] right-[-10px] h-20 w-20 rotate-12 border-4 border-white shadow-lg",
            isEditing ? "bg-purple-400" : "bg-blue-400",
          )}
        />
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
                    : "border-red-400 shadow-[4px_4px_0px_rgba(248,113,113,1)]",
                )}
              >
                <div
                  className={cn(
                    "text-base font-black uppercase tracking-tight",
                    isEditing ? "text-blue-500" : "text-red-500",
                  )}
                >
                  {isEditing ? "Edit Post" : "Author Upload"}
                </div>
              </div>
              {isOpen && (
                <div className="mt-3 text-sm font-bold text-zinc-600 dark:text-zinc-400">
                  {isEditing ? (
                    <>
                      Editing{" "}
                      <span className="text-zinc-900 dark:text-zinc-100">
                        {editingPost?.title || "Untitled"}
                      </span>
                      . Update fields and save.
                    </>
                  ) : (
                    <>
                      Pick{" "}
                      <span className="text-zinc-900 dark:text-zinc-100">
                        media files
                      </span>{" "}
                      + optional{" "}
                      <span className="text-zinc-900 dark:text-zinc-100">
                        thumbnail
                      </span>
                      .
                    </>
                  )}
                </div>
              )}
            </div>
          </div>

          <div className="flex items-center gap-2">
            {isEditing && onCancelEdit && (
              <button
                type="button"
                onClick={handleCancelEdit}
                className={cn(UI.button, UI.btnYellow)}
              >
                Cancel
              </button>
            )}
            <div className="rounded-2xl border-4 border-green-300 bg-white px-3 py-2 shadow-[4px_4px_0px_rgba(74,222,128,1)]">
              <div className="text-[10px] font-black uppercase tracking-wide text-green-300">
                Logged in
              </div>
              <div className="text-sm font-black text-green-700">
                {user.username}
              </div>
            </div>
          </div>
        </div>

        {isOpen && (
          <div className="mt-5 grid gap-4 lg:grid-cols-2 animate-in fade-in slide-in-from-top-4 duration-300">
            <div className="space-y-3">
              {/* File Inputs */}
              <div className="grid gap-3 md:grid-cols-2">
                <label
                  className="space-y-1 block cursor-pointer group relative"
                  onDragEnter={(e) => {
                    e.preventDefault();
                    setDragFull(true);
                  }}
                  onDragOver={handleDragOver}
                  onDragLeave={(e) => {
                    e.preventDefault();
                    setDragFull(false);
                  }}
                  onDrop={(e) => handleDrop(e, addFiles, true)}
                >
                  <DropOverlay visible={dragFull} />
                  <div className={UI.label}>
                    {isEditing ? "Add more media" : "Media Files *"}
                  </div>
                  <input
                    ref={fullInputRef}
                    type="file"
                    accept="image/*,video/*"
                    multiple
                    onChange={(e) => {
                      if (e.target.files) addFiles(Array.from(e.target.files));
                    }}
                    className={cn(
                      UI.input,
                      "file:mr-3 file:rounded-xl file:border-4 file:border-blue-300 file:bg-blue-200 file:px-3 file:py-1.5 file:text-xs file:font-black file:uppercase file:tracking-wide file:text-blue-900",
                      "file:cursor-pointer file:transition-all file:duration-200 file:ease-out",
                      "file:hover:scale-105 file:hover:brightness-110 file:hover:shadow-[0_4px_12px_rgba(147,197,253,0.5)]",
                      "file:active:scale-95",
                      "cursor-pointer",
                    )}
                  />
                </label>
                <label
                  className="space-y-1 block cursor-pointer group relative"
                  onDragEnter={(e) => {
                    e.preventDefault();
                    setDragThumb(true);
                  }}
                  onDragOver={handleDragOver}
                  onDragLeave={(e) => {
                    e.preventDefault();
                    setDragThumb(false);
                  }}
                  onDrop={handleThumbDrop}
                >
                  <DropOverlay visible={dragThumb} />
                  <div className={UI.label}>Thumbnail (optional)</div>
                  <input
                    ref={thumbInputRef}
                    type="file"
                    accept="image/*"
                    onChange={(e) => {
                      setThumbFile(e.target.files?.[0] ?? null);
                      setClearThumbnail(false);
                    }}
                    className={cn(
                      UI.input,
                      "file:mr-3 file:rounded-xl file:border-4 file:border-green-300 file:bg-green-200 file:px-3 file:py-1.5 file:text-xs file:font-black file:uppercase file:tracking-wide file:text-green-900",
                      "file:cursor-pointer file:transition-all file:duration-200 file:ease-out",
                      "file:hover:scale-105 file:hover:brightness-110 file:hover:shadow-[0_4px_12px_rgba(134,239,172,0.5)]",
                      "file:active:scale-95",
                      "cursor-pointer",
                    )}
                  />
                </label>
              </div>

              {/* Metadata Inputs */}
              <div className="grid gap-3 md:grid-cols-2">
                <label className="space-y-1">
                  <div className={UI.label}>Title (optional)</div>
                  <input
                    value={title}
                    onChange={(e) => setTitle(e.target.value)}
                    placeholder="e.g. Valentines set"
                    className={UI.input}
                  />
                </label>

                <label className="space-y-1">
                  <div className={UI.label}>Post date</div>
                  <input
                    type="datetime-local"
                    value={postDate}
                    onChange={(e) => setPostDate(e.target.value)}
                    className={UI.input}
                  />
                </label>
              </div>

              <label className="space-y-1">
                <div className={UI.label}>Description (optional)</div>
                <textarea
                  value={desc}
                  onChange={(e) => setDesc(e.target.value)}
                  placeholder="Add context, credits, notes…"
                  rows={4}
                  className={cn(UI.input, "resize-none")}
                />
              </label>

              <DropdownAddToList
                label="Allowed roles (tiers)"
                placeholder="Click to open role picker"
                options={roleOptions}
                selectedIds={allowedRoleIds}
                onAdd={(id) =>
                  setAllowedRoleIds((s) => (s.includes(id) ? s : [...s, id]))
                }
                onRemove={(id) =>
                  setAllowedRoleIds((s) => s.filter((x) => x !== id))
                }
                renderSelected={(id) => {
                  const r = roleOptions.find((o) => o.id === id);
                  return <RolePill name={r?.name || id} color={r?.color} />;
                }}
              />

              <DropdownAddToList
                label="Channels to post in"
                placeholder="Click to open channel picker"
                options={channelOptions}
                selectedIds={channelIds}
                onAdd={(id) =>
                  setChannelIds((s) => (s.includes(id) ? s : [...s, id]))
                }
                onRemove={(id) =>
                  setChannelIds((s) => s.filter((x) => x !== id))
                }
                renderSelected={(id) => {
                  const c = channelOptions.find((o) => o.id === id);
                  return <ChannelPill name={c?.name || c} />;
                }}
              />

              <div className="flex items-center justify-between gap-3 pt-2">
                <div className="text-xs font-bold text-zinc-500">
                  {isEditing
                    ? "Reorder, remove, or add media before saving."
                    : "At least one file required."}
                </div>
                <button
                  type="button"
                  disabled={!canSubmit}
                  onClick={handleSubmit}
                  className={cn(
                    UI.button,
                    isEditing ? UI.btnBlue : UI.btnRed,
                    !canSubmit && UI.btnDisabled,
                  )}
                >
                  {isEditing ? "Save Changes" : "Create Post"}
                </button>
              </div>
            </div>

            {/* Previews and file list */}
            <div className="space-y-3">
              <div className="grid gap-3 md:grid-cols-2">
                {/* Main Focus Preview */}
                <div
                  className={cn(
                    "p-3 transition relative",
                    UI.cardBlue,
                    focusPreview && !focusPreview.isVideo
                      ? "cursor-crosshair"
                      : "cursor-pointer hover:opacity-90 active:scale-95",
                  )}
                  onDragEnter={(e) => {
                    e.preventDefault();
                    setDragFull(true);
                  }}
                  onDragOver={(e) => {
                    handleDragOver(e);
                  }}
                  onDragLeave={(e) => {
                    e.preventDefault();
                    setDragFull(false);
                  }}
                  onDrop={(e) => handleDrop(e, addFiles, true)}
                >
                  {!focusPreview && <DropOverlay visible={dragFull} />}
                  <div className={UI.label}>
                    {focusPreview?.isThumbnail
                      ? "Cover Preview (Thumbnail)"
                      : "Cover Preview (First Item)"}
                  </div>
                  <div
                    ref={fullPreviewRef}
                    className={cn(
                      "mt-2 aspect-square overflow-hidden rounded-2xl border-4 border-zinc-200 bg-zinc-50 relative select-none",
                      focusPreview && !focusPreview.isVideo
                        ? "cursor-crosshair"
                        : "cursor-pointer",
                    )}
                    onMouseDown={(e) => {
                      const rect =
                        fullPreviewRef.current?.getBoundingClientRect();
                      if (!rect || !focusPreview || focusPreview.isVideo)
                        return;
                      e.stopPropagation();
                      const x = Math.min(
                        100,
                        Math.max(
                          0,
                          ((e.clientX - rect.left) / rect.width) * 100,
                        ),
                      );
                      const y = Math.min(
                        100,
                        Math.max(
                          0,
                          ((e.clientY - rect.top) / rect.height) * 100,
                        ),
                      );
                      setFocusX(Math.round(x * 10) / 10);
                      setFocusY(Math.round(y * 10) / 10);

                      const onMove = (ev: MouseEvent) => {
                        const mx = Math.min(
                          100,
                          Math.max(
                            0,
                            ((ev.clientX - rect.left) / rect.width) * 100,
                          ),
                        );
                        const my = Math.min(
                          100,
                          Math.max(
                            0,
                            ((ev.clientY - rect.top) / rect.height) * 100,
                          ),
                        );
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
                    onClick={(e) => {
                      if (focusPreview) e.stopPropagation();
                      else fullInputRef.current?.click();
                    }}
                  >
                    {focusPreview ? (
                      <>
                        {focusPreview.isVideo ? (
                          <video
                            src={focusPreview.url}
                            className="h-full w-full object-cover"
                            controls
                            controlsList="nodownload"
                            playsInline
                          />
                        ) : (
                          <>
                            <img
                              src={focusPreview.url}
                              className="h-full w-full object-cover"
                              style={{
                                objectPosition: `${focusX}% ${focusY}%`,
                              }}
                              alt=""
                              draggable={false}
                            />
                            <div
                              className="absolute pointer-events-none"
                              style={{
                                left: `${focusX}%`,
                                top: `${focusY}%`,
                                transform: "translate(-50%, -50%)",
                              }}
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
                        {isEditing ? "No media" : "Pick a file"}
                      </div>
                    )}
                  </div>
                  {focusPreview && !focusPreview.isVideo && (
                    <div className="mt-1 text-[10px] font-bold text-zinc-400 tabular-nums">
                      Focus: {focusX.toFixed(1)}% × {focusY.toFixed(1)}%
                    </div>
                  )}
                </div>

                <div
                  className={cn(
                    "p-3 cursor-pointer hover:opacity-90 active:scale-95 transition relative",
                    UI.cardGreen,
                  )}
                  onClick={() => thumbInputRef.current?.click()}
                  onDragEnter={(e) => {
                    e.preventDefault();
                    setDragThumb(true);
                  }}
                  onDragOver={handleDragOver}
                  onDragLeave={(e) => {
                    e.preventDefault();
                    setDragThumb(false);
                  }}
                  onDrop={handleThumbDrop}
                >
                  <DropOverlay visible={dragThumb} />
                  <div className={UI.label}>Thumbnail Preview</div>
                  <div className="mt-2 aspect-square overflow-hidden rounded-2xl border-4 border-zinc-200 bg-zinc-50">
                    {previewThumb || (!clearThumbnail && existingThumbUrl) ? (
                      isThumbVideo ? (
                        <video
                          src={previewThumb || undefined}
                          className="h-full w-full object-cover"
                          controls
                          controlsList="nodownload"
                          playsInline
                        />
                      ) : (
                        <img
                          src={previewThumb || existingThumbUrl || ""}
                          className="h-full w-full object-cover"
                          alt=""
                        />
                      )
                    ) : (
                      <div className="flex h-full w-full items-center justify-center text-xs font-bold text-zinc-400">
                        Optional
                      </div>
                    )}
                  </div>
                  {thumbFile ? (
                    <div className="mt-2 text-xs font-bold text-zinc-500 dark:text-zinc-400">
                      {thumbFile.name}
                    </div>
                  ) : null}
                  {isEditing && (previewThumb || existingThumbUrl) && (
                    <button
                      type="button"
                      onClick={(e) => {
                        e.stopPropagation();
                        setThumbFile(null);
                        if (thumbInputRef.current)
                          thumbInputRef.current.value = "";
                        setPreviewThumb(null);
                        setIsThumbVideo(false);
                        setClearThumbnail(true);
                      }}
                      className={cn(
                        "mt-2",
                        UI.button,
                        UI.btnYellow,
                        "px-2 py-1 text-[10px]",
                      )}
                    >
                      Remove Thumbnail
                    </button>
                  )}
                </div>
              </div>

              {/* File List / Gallery Strip */}
              {mediaOrder.length > 0 && (
                <div className={cn("p-3", UI.card)}>
                  <div className="flex items-end justify-between gap-3">
                    <div className={UI.label}>Attached Media</div>
                    <label className="flex items-center gap-2 text-[10px] font-bold uppercase tracking-wide text-zinc-500">
                      <span>Preview Size</span>
                      <input
                        type="range"
                        min={56}
                        max={160}
                        step={4}
                        value={previewSize}
                        onChange={(e) => setPreviewSize(Number(e.target.value))}
                        className="w-24"
                      />
                    </label>
                  </div>
                  <div
                    className="mt-2 flex gap-2 overflow-x-auto pb-2 scrollbar-thin scrollbar-thumb-zinc-300 dark:scrollbar-thumb-zinc-600"
                    onDragOver={(e) => {
                      if (!draggedMediaToken) return;
                      e.preventDefault();
                    }}
                  >
                    {mediaOrder.map((token, i) => {
                      const isRemote = token.startsWith("e:");
                      const remoteId = isRemote ? Number(token.slice(2)) : null;
                      const newKey =
                        !isRemote && token.startsWith("n:")
                          ? token.slice(2)
                          : null;
                      const remote =
                        isRemote && Number.isFinite(remoteId)
                          ? remotePreviews.find((p) => p.id === remoteId)
                          : null;
                      const local = newKey
                        ? filePreviews.find((p) => p.key === newKey)
                        : null;

                      if (!remote && !local) return null;

                      const isVideo = remote
                        ? remote.isVideo
                        : !!local?.isVideo;
                      const src = remote ? remote.url : local?.url || "";

                      return (
                        <div
                          key={token}
                          draggable
                          onDragStart={(e) => {
                            setDraggedMediaToken(token);
                            e.dataTransfer.effectAllowed = "move";
                          }}
                          onDragEnd={() => setDraggedMediaToken(null)}
                          onDragOver={(e) => {
                            if (
                              !draggedMediaToken ||
                              draggedMediaToken === token
                            )
                              return;
                            e.preventDefault();
                            e.dataTransfer.dropEffect = "move";
                          }}
                          onDrop={(e) => {
                            e.preventDefault();
                            if (!draggedMediaToken) return;
                            reorderMediaToken(draggedMediaToken, token);
                            setDraggedMediaToken(null);
                          }}
                          className={cn(
                            "relative shrink-0 overflow-hidden rounded-lg border-2 group cursor-grab active:cursor-grabbing",
                            remote ? "border-blue-200" : "border-green-200",
                            draggedMediaToken === token && "opacity-60",
                          )}
                          style={{
                            width: `${previewSize}px`,
                            height: `${previewSize}px`,
                          }}
                        >
                          {isVideo ? (
                            remote ? (
                              <video
                                src={src}
                                className="h-full w-full object-cover"
                              />
                            ) : (
                              <div className="h-full w-full bg-zinc-900 flex items-center justify-center">
                                <FileVideo className="h-6 w-6 text-white" />
                              </div>
                            )
                          ) : (
                            <img
                              src={src}
                              className="h-full w-full object-cover"
                              alt=""
                            />
                          )}
                          {remote ? (
                            <div className="absolute bottom-0 left-0 right-0 flex items-center justify-center bg-black/30 text-[8px] font-bold text-white uppercase">
                              Saved
                            </div>
                          ) : null}
                          <div className="absolute top-0 left-0 right-0 flex justify-between opacity-0 group-hover:opacity-100 transition-opacity">
                            <button
                              type="button"
                              disabled={i === 0}
                              onClick={() => moveMediaToken(token, -1)}
                              className="m-0.5 rounded bg-black/60 p-0.5 text-white disabled:opacity-40"
                            >
                              <ChevronLeft className="h-3 w-3" />
                            </button>
                            <button
                              type="button"
                              disabled={i === mediaOrder.length - 1}
                              onClick={() => moveMediaToken(token, 1)}
                              className="m-0.5 rounded bg-black/60 p-0.5 text-white disabled:opacity-40"
                            >
                              <ChevronRight className="h-3 w-3" />
                            </button>
                          </div>
                          <button
                            type="button"
                            onClick={() => removeMediaToken(token)}
                            className="absolute bottom-0 right-0 p-0.5 bg-red-500 text-white rounded-tl-lg opacity-0 group-hover:opacity-100 transition-opacity"
                          >
                            <X className="h-3 w-3" />
                          </button>
                        </div>
                      );
                    })}
                  </div>
                </div>
              )}

              <div className={cn("p-4", UI.card)}>
                <div className={UI.sectionTitle}>
                  {isEditing ? "Updated values" : "What gets posted"}
                </div>
                <div className="mt-2 space-y-2 text-sm font-bold text-zinc-700 dark:text-zinc-300">
                  <div>
                    <span className="text-zinc-400">Guild:</span>{" "}
                    {guildName || "Unknown"}
                  </div>
                  <div>
                    <span className="text-zinc-400">Media count:</span>{" "}
                    {effectiveMediaCount}
                  </div>
                  <div className="flex flex-wrap gap-2">
                    <span className="text-zinc-400">Channels:</span>
                    {channelIds.length ? (
                      channelIds.map((c) => {
                        const ch = channelOptions.find((o) => o.id === c);
                        return <ChannelPill key={c} name={ch?.name || c} />;
                      })
                    ) : (
                      <span className="text-zinc-400">None</span>
                    )}
                  </div>
                  <div className="flex flex-wrap gap-2">
                    <span className="text-zinc-400">Allowed roles:</span>
                    {allowedRoleIds.length ? (
                      allowedRoleIds.map((r) => {
                        const ro = roleOptions.find((o) => o.id === r);
                        return (
                          <RolePill
                            key={r}
                            name={ro?.name || r}
                            color={ro?.color}
                          />
                        );
                      })
                    ) : (
                      <span className="text-zinc-400">None</span>
                    )}
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
