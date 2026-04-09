import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  Routes,
  Route,
  useNavigate,
  useLocation,
  matchPath,
} from "react-router-dom";
import { intersect, cn } from "./lib/utils";
import { UI } from "./constants";
import type {
  CachedRoleName,
  ChannelPayload,
  DiscordUser,
  JwtPayload,
  NameCache,
  Post,
  RolePayload,
  SessionPayload,
  SettingsGuildPayload,
  SortMode,
  UploadConfigPayload,
  ViewMode,
} from "./types";
import { DiagonalSlitHeader } from "./components/DiagonalSlitHeader";
import { LoginModal } from "./components/LoginModal";
import {
  AuthorPanel,
  type AuthorPanelPostInput,
} from "./components/AuthorPanel";
import { TopBar } from "./components/TopBar";
import { MainGalleryView } from "./components/MainGalleryView";
import { PostDetailView } from "./components/PostDetailView";
import { RightSidebar } from "./components/RightSidebar";
import { ProfileSidebar } from "./components/ProfileSidebar";
import { NotFound } from "./pages/NotFound";
import { SettingsModal } from "./pages/Settings";
import { useSettings } from "./contexts/SettingsContext";
import { MembershipModal } from "./components/MembershipModal.tsx";

const NAME_CACHE_KEY = "drigo_role_channel_name_cache_v1";
const PAGE_SIZE = 9;

function readNameCache(): NameCache {
  try {
    const raw = localStorage.getItem(NAME_CACHE_KEY);
    if (!raw) return { roles: {}, channels: {} };
    const parsed = JSON.parse(raw) as Partial<NameCache>;
    return {
      roles: parsed.roles ?? {},
      channels: parsed.channels ?? {},
      guildName: parsed.guildName,
    };
  } catch {
    return { roles: {}, channels: {} };
  }
}

function mapDiscordRoles(
  rawRoles: readonly RolePayload[] | undefined,
): NonNullable<DiscordUser["roles"]> {
  if (!Array.isArray(rawRoles)) return [];

  return rawRoles
    .filter((role): role is Required<Pick<RolePayload, "id">> & RolePayload =>
      typeof role.id === "string" && role.id.trim().length > 0,
    )
    .map((role) => ({
      id: role.id,
      name: role.name ?? "",
      color: role.color ?? 0,
      managed: !!role.managed,
      mentionable: !!role.mentionable,
      hoist: !!role.hoist,
      position: role.position ?? 0,
      permissions: role.permissions ?? 0,
      icon: role.icon ?? "",
      unicodeEmoji: role.unicodeEmoji ?? "",
      flags: role.flags ?? 0,
    }));
}

function mapRoleCache(
  rawRoles: readonly RolePayload[] | undefined,
): Record<string, CachedRoleName> {
  const roles: Record<string, CachedRoleName> = {};
  if (!Array.isArray(rawRoles)) return roles;

  rawRoles.forEach((role) => {
    if (!role.id || !role.name) return;
    roles[role.id] = {
      name: role.name,
      color: role.color,
      managed: role.managed,
    };
  });

  return roles;
}

function mapChannelCache(
  rawChannels: readonly ChannelPayload[] | undefined,
): Record<string, string> {
  const channels: Record<string, string> = {};
  if (!Array.isArray(rawChannels)) return channels;

  rawChannels.forEach((channel) => {
    if (!channel.id || !channel.name) return;
    channels[channel.id] = channel.name;
  });

  return channels;
}

function toDiscordUserFromClaims(claims: JwtPayload): DiscordUser | null {
  if (!claims.uid || !claims.sub) return null;

  return {
    userId: claims.uid,
    username: claims.sub,
    globalName: claims.sub,
    avatar: claims.avt || "",
    banner: claims.ban || "",
    bot: false,
    roles: mapDiscordRoles(claims.roles),
    isAdmin: !!claims.adm,
  };
}

function toDiscordUserFromSession(session: SessionPayload): DiscordUser | null {
  if (!session.userId || !session.username) return null;

  return {
    userId: session.userId,
    username: session.username,
    globalName: session.globalName || session.username,
    avatar: session.avatar || "",
    banner: session.banner || "",
    bot: !!session.bot,
    roles: mapDiscordRoles(session.roles),
    isAdmin: !!session.isAdmin,
  };
}

function App() {
  const [user, setUser] = useState<DiscordUser | null>(null);
  const [nameCache, setNameCache] = useState<NameCache>(() => readNameCache());
  const [loginOpen, setLoginOpen] = useState(false);
  const [posts, setPosts] = useState<Post[]>([]);
  const [loading, setLoading] = useState(false);
  const [initialLoading, setInitialLoading] = useState(true);

  const navigate = useNavigate();
  const location = useLocation();

  const [tagFilter, setTagFilter] = useState<string | null>(null);
  const [q, setQ] = useState("");
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [membershipOpen, setMembershipOpen] = useState(false);
  const [editingPost, setEditingPost] = useState<Post | null>(null);

  const [sortMode, setSortMode] = useState<SortMode>(() => {
    const saved = localStorage.getItem("gallery_sort");
    return saved === "date" ? "date" : "id";
  });

  const sortCacheRef = useRef<Record<SortMode, Post[]>>({ id: [], date: [] });
  const postsRef = useRef<Post[]>([]);
  const sessionRefreshInFlightRef = useRef<Promise<void> | null>(null);

  useEffect(() => {
    localStorage.setItem("gallery_sort", sortMode);
  }, [sortMode]);

  const postMatch = matchPath("/post/:postId", location.pathname);
  const selectedId = postMatch ? postMatch.params.postId || null : null;

  const view: ViewMode = useMemo(() => {
    if (selectedId) return "post";
    if (location.pathname === "/") return "gallery";
    return "not-found";
  }, [location.pathname, selectedId]);

  const mergeNameCache = useCallback(
    (incoming: {
      roles?: Record<string, CachedRoleName>;
      channels?: Record<string, string>;
      guildName?: string;
    }) => {
      setNameCache((previous) => {
        const next: NameCache = {
          roles: { ...previous.roles, ...(incoming.roles ?? {}) },
          channels: { ...previous.channels, ...(incoming.channels ?? {}) },
          guildName: incoming.guildName ?? previous.guildName,
        };
        localStorage.setItem(NAME_CACHE_KEY, JSON.stringify(next));
        return next;
      });
    },
    [],
  );

  const refreshSession = useCallback(
    async (token: string) => {
      if (sessionRefreshInFlightRef.current) {
        await sessionRefreshInFlightRef.current;
        return;
      }

      const refreshPromise = (async () => {
        try {
          const res = await fetch("/auth/session", {
            headers: {
              Authorization: `Bearer ${token}`,
            },
          });

          if (res.status === 401) {
            localStorage.removeItem("jwt");
            setUser(null);
            return;
          }

          if (!res.ok) return;

          const data = (await res.json()) as SessionPayload;
          const liveUser = toDiscordUserFromSession(data);
          if (!liveUser) return;

          const roleCache = mapRoleCache(data.roles);
          if (Object.keys(roleCache).length > 0) {
            mergeNameCache({ roles: roleCache });
          }

          setUser(liveUser);
        } catch (err) {
          console.error("Failed to refresh authenticated session", err);
        }
      })();

      sessionRefreshInFlightRef.current = refreshPromise;
      try {
        await refreshPromise;
      } finally {
        sessionRefreshInFlightRef.current = null;
      }
    },
    [mergeNameCache],
  );

  useEffect(() => {
    const params = new URLSearchParams(location.search);
    const tokenParam = params.get("token");

    if (tokenParam) {
      localStorage.setItem("jwt", tokenParam);
      navigate(location.pathname, { replace: true });
    }

    const token = localStorage.getItem("jwt");
    if (!token) return;

    try {
      const base64Url = token.split(".")[1];
      const base64 = base64Url.replace(/-/g, "+").replace(/_/g, "/");
      const jsonPayload = decodeURIComponent(
        window
          .atob(base64)
          .split("")
          .map((char) => "%" + ("00" + char.charCodeAt(0).toString(16)).slice(-2))
          .join(""),
      );

      const claims = JSON.parse(jsonPayload) as JwtPayload;
      if (claims.exp && claims.exp * 1000 < Date.now()) {
        console.warn("JWT token expired, logging out.");
        localStorage.removeItem("jwt");
        return;
      }

      const claimUser = toDiscordUserFromClaims(claims);
      if (claimUser) {
        setUser(claimUser);
      }

      const rolesFromClaims = mapRoleCache(claims.roles);
      if (Object.keys(rolesFromClaims).length > 0) {
        mergeNameCache({ roles: rolesFromClaims });
      }

      void refreshSession(token);
    } catch (err) {
      console.error("Failed to parse JWT", err);
      localStorage.removeItem("jwt");
    }
  }, [
    location.pathname,
    location.search,
    mergeNameCache,
    navigate,
    refreshSession,
  ]);

  useEffect(() => {
    if (!user?.userId) return;

    const handleRefresh = () => {
      const token = localStorage.getItem("jwt");
      if (!token) return;
      void refreshSession(token);
    };

    const handleVisibilityChange = () => {
      if (document.visibilityState === "visible") {
        handleRefresh();
      }
    };

    window.addEventListener("focus", handleRefresh);
    document.addEventListener("visibilitychange", handleVisibilityChange);
    return () => {
      window.removeEventListener("focus", handleRefresh);
      document.removeEventListener("visibilitychange", handleVisibilityChange);
    };
  }, [refreshSession, user?.userId]);

  useEffect(() => {
    if (!user?.isAdmin) return;

    const token = localStorage.getItem("jwt");
    if (!token) return;

    fetch("/upload", {
      headers: {
        Authorization: `Bearer ${token}`,
      },
    })
      .then((res) => (res.ok ? (res.json() as Promise<UploadConfigPayload>) : null))
      .then((data) => {
        if (!data) return;

        mergeNameCache({
          roles: mapRoleCache(data.roles),
          channels: mapChannelCache(data.channels),
          guildName:
            typeof data.guild_name === "string" ? data.guild_name : undefined,
        });
      })
      .catch((err) => {
        console.error("Failed to refresh role/channel cache", err);
      });
  }, [mergeNameCache, user?.isAdmin]);

  const { settings } = useSettings();
  const settingsGuildData = settings as SettingsGuildPayload;

  const effectiveNameCache = useMemo<NameCache>(() => {
    const settingsRoles = mapRoleCache(settingsGuildData.roles);
    const settingsChannels = mapChannelCache(settingsGuildData.channels);

    return {
      roles: { ...nameCache.roles, ...settingsRoles },
      channels: { ...nameCache.channels, ...settingsChannels },
      guildName:
        typeof settingsGuildData.guild_name === "string"
          ? settingsGuildData.guild_name
          : nameCache.guildName,
    };
  }, [
    nameCache.channels,
    nameCache.guildName,
    nameCache.roles,
    settingsGuildData.channels,
    settingsGuildData.guild_name,
    settingsGuildData.roles,
  ]);

  const hydratedPosts = useMemo(() => {
    return posts.map((post) => ({
      ...post,
      allowedRoles: (post.allowedRoles ?? []).map((role) => {
        const cached = effectiveNameCache.roles[role.id];
        if (!cached) return role;
        return {
          ...role,
          name: cached.name,
          color: cached.color ?? role.color,
        };
      }),
    }));
  }, [effectiveNameCache.roles, posts]);

  const roleOptions = useMemo(() => {
    return Object.entries(effectiveNameCache.roles)
      .map(([id, role]) => ({
        id,
        name: role.name,
        color: role.color,
        managed: role.managed,
      }))
      .sort((a, b) => a.name.localeCompare(b.name));
  }, [effectiveNameCache.roles]);

  const channelOptions = useMemo(() => {
    return Object.entries(effectiveNameCache.channels)
      .map(([id, name]) => ({ id, name }))
      .sort((a, b) => a.name.localeCompare(b.name));
  }, [effectiveNameCache.channels]);

  const resolveChannelName = useCallback(
    (id: string) => effectiveNameCache.channels[id] || id,
    [effectiveNameCache.channels],
  );

  const [page, setPage] = useState(1);
  const [hasMore, setHasMore] = useState(true);

  const loadPosts = useCallback(
    (pageNum: number, reset = false, sort: SortMode = sortMode) => {
      setLoading(true);

      const token = localStorage.getItem("jwt");
      const headers: Record<string, string> = {};
      if (token) {
        headers.Authorization = `Bearer ${token}`;
      }

      const fetchLimit =
        reset && pageNum === 1
          ? Math.max(PAGE_SIZE, postsRef.current.length)
          : PAGE_SIZE;

      fetch(`/posts?page=${pageNum}&limit=${fetchLimit}&sort=${sort}`, {
        headers,
      })
        .then((res) => res.json() as Promise<unknown>)
        .then((data) => {
          const items: Post[] = Array.isArray(data) ? (data as Post[]) : [];
          const rolesFromPosts: Record<string, CachedRoleName> = {};

          items.forEach((post) => {
            (post.allowedRoles ?? []).forEach((role) => {
              if (!role?.id || !role?.name) return;
              rolesFromPosts[role.id] = {
                name: role.name,
                color: role.color,
              };
            });
          });

          if (Object.keys(rolesFromPosts).length > 0) {
            mergeNameCache({ roles: rolesFromPosts });
          }

          const newPosts = reset ? items : [...postsRef.current, ...items];
          postsRef.current = newPosts;
          setPosts(newPosts);
          sortCacheRef.current[sort] = newPosts;
          setHasMore(items.length === fetchLimit);
          setLoading(false);
          setInitialLoading(false);
        })
        .catch((err) => {
          console.error("Failed to fetch posts", err);
          setLoading(false);
          setInitialLoading(false);
        });
    },
    [mergeNameCache, sortMode],
  );

  const resetAndLoadPosts = useCallback(
    (sort: SortMode = sortMode) => {
      postsRef.current = [];
      sortCacheRef.current = { id: [], date: [] };
      setPosts([]);
      setPage(1);
      setHasMore(true);
      setInitialLoading(true);
      loadPosts(1, true, sort);
    },
    [loadPosts, sortMode],
  );

  useEffect(() => {
    const cached = sortCacheRef.current[sortMode];
    if (cached.length > 0) {
      postsRef.current = cached;
      setPosts(cached);
      setHasMore(cached.length % PAGE_SIZE === 0);
      return;
    }

    resetAndLoadPosts(sortMode);
  }, [resetAndLoadPosts, sortMode, user]);

  const handleLoadMore = useCallback(() => {
    if (!hasMore || loading) return;
    sortCacheRef.current = { id: [], date: [] };
    const nextPage = page + 1;
    setPage(nextPage);
    loadPosts(nextPage, false, sortMode);
  }, [hasMore, loadPosts, loading, page, sortMode]);

  const [selectedPost, setSelectedPost] = useState<Post | null>(null);

  const hydratedSelectedPost = useMemo(() => {
    if (!selectedPost) return null;
    return {
      ...selectedPost,
      allowedRoles: (selectedPost.allowedRoles ?? []).map((role) => {
        const cached = effectiveNameCache.roles[role.id];
        if (!cached) return role;
        return {
          ...role,
          name: cached.name,
          color: cached.color ?? role.color,
        };
      }),
    };
  }, [effectiveNameCache.roles, selectedPost]);

  const selected = useMemo(() => {
    if (!selectedId) return null;
    return hydratedPosts.find((post) => post.postKey === selectedId) ?? hydratedSelectedPost;
  }, [hydratedPosts, hydratedSelectedPost, selectedId]);

  useEffect(() => {
    if (!selectedId || selected) return;
    if (selectedPost && selectedPost.postKey === selectedId) return;

    const token = localStorage.getItem("jwt");
    const headers: Record<string, string> = {};
    if (token) {
      headers.Authorization = `Bearer ${token}`;
    }

    fetch(`/posts/${selectedId}`, { headers })
      .then((res) => (res.ok ? (res.json() as Promise<Post>) : null))
      .then((post) => {
        if (post) {
          setSelectedPost(post);
        }
      })
      .catch((err) => {
        console.error("Failed to fetch individual post", err);
      });
  }, [selected, selectedId, selectedPost]);

  const viewerRoleIds = useMemo(
    () => (user?.roles ?? []).map((role) => role.id),
    [user],
  );

  const filteredPosts = useMemo(() => {
    const search = q.trim().toLowerCase();
    if (!search) return hydratedPosts;

    return hydratedPosts.filter((post) => {
      const haystack =
        `${post.title ?? ""} ${(post.description ?? "").slice(0, 120)}`.toLowerCase();
      return haystack.includes(search);
    });
  }, [hydratedPosts, q]);

  const defaultDocumentTitleRef = useRef(document.title);

  useEffect(() => {
    const heroTitle = settings.hero_title?.trim();
    document.title =
      heroTitle && heroTitle.length > 0
        ? heroTitle
        : defaultDocumentTitleRef.current;
  }, [settings.hero_title]);

  const canAccessPost = useMemo(() => {
    return (post: Post) => {
      if (user?.isAdmin) return true;
      if (settings.public_access) return true;

      const postRoleIds = post.allowedRoles.map((role) => role.id);
      if (postRoleIds.length === 0) return true;
      return intersect(postRoleIds, viewerRoleIds);
    };
  }, [settings.public_access, user, viewerRoleIds]);

  async function handleCreate(postInput: AuthorPanelPostInput) {
    const formData = new FormData();
    formData.append("title", postInput.title);
    formData.append("description", postInput.description);
    formData.append("roles", postInput.allowedRoleIds.join(","));
    formData.append("channels", postInput.channelIds.join(","));
    formData.append("postDate", postInput.postDate);
    formData.append("focusX", String(postInput.focusX ?? 50));
    formData.append("focusY", String(postInput.focusY ?? 50));

    if (postInput.images && postInput.images.length > 0) {
      postInput.images.forEach((file) => {
        formData.append("images", file);
      });
    }

    if (postInput.thumbnail) {
      formData.append("thumbnail", postInput.thumbnail);
    }

    const token = localStorage.getItem("jwt");
    const headers: Record<string, string> = {};
    if (token) headers.Authorization = `Bearer ${token}`;

    try {
      const res = await fetch("/posts", {
        method: "POST",
        headers,
        body: formData,
      });

      if (!res.ok) {
        const err = await res.json().catch(() => ({ error: res.statusText }));
        alert(`Upload failed: ${err.error || res.statusText}`);
        return false;
      }

      await res.json();
      resetAndLoadPosts();
      return true;
    } catch (err) {
      console.error("Upload error:", err);
      alert("Upload failed. Check the console for details.");
      return false;
    }
  }

  async function handleUpdate(
    postKey: string,
    postInput: Omit<AuthorPanelPostInput, "images"> & {
      images?: File[];
      removeImageIds?: number[];
      mediaOrder?: string[];
      clearThumbnail?: boolean;
    },
  ) {
    const formData = new FormData();
    formData.append("title", postInput.title);
    formData.append("description", postInput.description);
    formData.append("roles", postInput.allowedRoleIds.join(","));
    formData.append("channels", postInput.channelIds.join(","));
    formData.append("postDate", postInput.postDate);
    formData.append("focusX", String(postInput.focusX ?? 50));
    formData.append("focusY", String(postInput.focusY ?? 50));

    if (postInput.images && postInput.images.length > 0) {
      postInput.images.forEach((file) => {
        formData.append("images", file);
      });
    }

    if (postInput.thumbnail) {
      formData.append("thumbnail", postInput.thumbnail);
    }
    if (postInput.clearThumbnail) {
      formData.append("clearThumbnail", "1");
    }
    if (postInput.removeImageIds && postInput.removeImageIds.length > 0) {
      formData.append("removeImageIds", postInput.removeImageIds.join(","));
    }
    if (postInput.mediaOrder && postInput.mediaOrder.length > 0) {
      formData.append("mediaOrder", postInput.mediaOrder.join(","));
    }

    const token = localStorage.getItem("jwt");
    const headers: Record<string, string> = {};
    if (token) headers.Authorization = `Bearer ${token}`;

    try {
      const res = await fetch(`/posts/${postKey}`, {
        method: "PATCH",
        headers,
        body: formData,
      });

      if (!res.ok) {
        const err = await res.json().catch(() => ({ error: res.statusText }));
        alert(`Edit failed: ${err.error || res.statusText}`);
        return false;
      }

      setEditingPost(null);
      resetAndLoadPosts();
      return true;
    } catch (err) {
      console.error("Edit error:", err);
      alert("Edit failed. Check the console for details.");
      return false;
    }
  }

  async function handleDelete(postKey: string) {
    const token = localStorage.getItem("jwt");
    const headers: Record<string, string> = {};
    if (token) headers.Authorization = `Bearer ${token}`;

    try {
      const res = await fetch(`/posts/${postKey}`, {
        method: "DELETE",
        headers,
      });

      if (!res.ok) {
        const err = await res.json().catch(() => ({ error: res.statusText }));
        alert(`Delete failed: ${err.error || res.statusText}`);
        return;
      }

      if (editingPost && editingPost.postKey === postKey) {
        setEditingPost(null);
      }
      navigate("/");
      resetAndLoadPosts();
    } catch (err) {
      console.error("Delete error:", err);
      alert("Delete failed. Check the console for details.");
    }
  }

  return (
    <>
      <SettingsModal open={settingsOpen} onClose={() => setSettingsOpen(false)} />
      {membershipOpen && <MembershipModal onClose={() => setMembershipOpen(false)} />}

      <div className={UI.page}>
        <LoginModal
          open={loginOpen}
          onClose={() => setLoginOpen(false)}
          onLogin={(nextUser) => {
            setUser(nextUser);
          }}
        />

        <div className={UI.max}>
          <DiagonalSlitHeader
            posts={posts}
            onClickRandom={() => {
              const pick =
                filteredPosts[
                  Math.floor(Math.random() * Math.max(1, filteredPosts.length))
                ];
              if (pick) {
                navigate(`/post/${pick.postKey}`);
              }
            }}
          />

          {view !== "not-found" && (
            <TopBar
              guildName={effectiveNameCache.guildName}
              view={view === "post" ? "post" : "gallery"}
              setView={(nextView) => {
                if (nextView === "gallery") navigate("/");
              }}
              tagFilter={tagFilter}
              setTagFilter={setTagFilter}
              user={user}
              setLoginOpen={setLoginOpen}
              setUser={setUser}
              selectedId={selectedId}
              setSettingsOpen={setSettingsOpen}
              setMembershipOpen={setMembershipOpen}
            />
          )}

          {user?.isAdmin ? (
            <div className="mt-6">
              <AuthorPanel
                key={editingPost?.postKey ?? "create"}
                user={user}
                onCreate={handleCreate}
                editingPost={editingPost}
                onUpdate={handleUpdate}
                onCancelEdit={() => setEditingPost(null)}
                availableRoles={roleOptions}
                availableChannels={channelOptions}
                guildName={effectiveNameCache.guildName}
              />
            </div>
          ) : null}

          <div className="mt-6 flex flex-col gap-4 lg:flex-row relative items-start">
            <div
              className={cn(
                "transition-all duration-500 w-full lg:w-[calc(100%-22rem)]",
              )}
            >
              <Routes>
                <Route
                  path="/"
                  element={
                    <MainGalleryView
                      posts={filteredPosts}
                      selectedId={selectedId}
                      canAccessPost={canAccessPost}
                      onOpenPost={(id) => {
                        navigate(`/post/${id}`);
                      }}
                      q={q}
                      setQ={setQ}
                      onLoadMore={handleLoadMore}
                      hasMore={hasMore}
                      loading={loading}
                      initialLoading={initialLoading}
                      sortMode={sortMode}
                      onSortChange={setSortMode}
                    />
                  }
                />
                <Route
                  path="/post/:postId"
                  element={
                    <PostDetailView
                      selected={selected}
                      onBack={() => {
                        navigate("/");
                      }}
                      canAccessPost={canAccessPost}
                    />
                  }
                />
                <Route path="*" element={<NotFound />} />
              </Routes>
            </div>

            {view !== "not-found" && (
              <div className={cn("w-full lg:w-80")}>
                <div className="sticky top-4">
                  {view === "gallery" ? (
                    <ProfileSidebar user={user} onLogin={() => setLoginOpen(true)} />
                  ) : (
                    <RightSidebar
                      posts={filteredPosts}
                      selectedId={selectedId}
                      canAccessPost={canAccessPost}
                      onSelect={(id) => {
                        navigate(`/post/${id}`);
                      }}
                      selected={selected}
                      user={user}
                      resolveChannelName={resolveChannelName}
                      onEditPost={(post) => {
                        setEditingPost({
                          ...post,
                          allowedRoles: [...(post.allowedRoles ?? [])],
                          images: [...(post.images ?? [])],
                        });
                        window.scrollTo({ top: 0, behavior: "smooth" });
                      }}
                      onDeletePost={handleDelete}
                    />
                  )}
                </div>
              </div>
            )}
          </div>
        </div>
      </div>
    </>
  );
}

export default App;
