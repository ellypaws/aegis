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
import type { DiscordUser, Post, ViewMode } from "./types";
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

type CachedRoleName = {
  name: string;
  color?: number;
  managed?: boolean;
};

type NameCache = {
  roles: Record<string, CachedRoleName>;
  channels: Record<string, string>;
  guildName?: string;
};

const NAME_CACHE_KEY = "drigo_role_channel_name_cache_v1";

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

  // Sort mode — lifted up so it controls the backend fetch
  type SortMode = "id" | "date";
  const [sortMode, setSortMode] = useState<SortMode>(() => {
    const saved = localStorage.getItem("gallery_sort");
    return saved === "date" ? "date" : "id";
  });
  useEffect(() => {
    localStorage.setItem("gallery_sort", sortMode);
  }, [sortMode]);

  // Cache: store fetched posts per sort mode for instant switching
  // Map<SortMode, Post[]>
  const sortCacheRef = useRef<Record<string, Post[]>>({});

  // Derive state from URL
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

  // Parse JWT for user info on mount
  useEffect(() => {
    const params = new URLSearchParams(location.search);
    const tokenParam = params.get("token");

    if (tokenParam) {
      localStorage.setItem("jwt", tokenParam);
      // Remove token from URL
      navigate(location.pathname, { replace: true });
    }

    const token = localStorage.getItem("jwt");
    if (token) {
      try {
        const base64Url = token.split(".")[1];
        const base64 = base64Url.replace(/-/g, "+").replace(/_/g, "/");
        const jsonPayload = decodeURIComponent(
          window
            .atob(base64)
            .split("")
            .map(function (c) {
              return "%" + ("00" + c.charCodeAt(0).toString(16)).slice(-2);
            })
            .join(""),
        );

        const claims = JSON.parse(jsonPayload);
        const rolesFromClaims: Record<string, CachedRoleName> = {};
        if (Array.isArray(claims.roles)) {
          claims.roles.forEach((r: any) => {
            if (!r?.id || !r?.name) return;
            rolesFromClaims[r.id] = {
              name: r.name,
              color: r.color,
              managed: r.managed,
            };
          });
        }
        if (Object.keys(rolesFromClaims).length > 0) {
          mergeNameCache({ roles: rolesFromClaims });
        }

        const u: DiscordUser = {
          userId: claims.uid,
          username: claims.sub,
          globalName: claims.sub,
          discriminator: "0000",
          avatar: claims.avt || "",
          banner: claims.ban || "",
          accentColor: 0,
          bot: false,
          system: false,
          publicFlags: 0,
          roles: claims.roles
            ? claims.roles.map((r: any) => ({
                id: r.id,
                name: r.name,
                color: r.color,
                managed: r.managed,
                mentionable: r.mentionable,
                hoist: r.hoist,
                position: r.position,
                permissions: r.permissions,
                icon: r.icon,
                unicodeEmoji: r.unicodeEmoji,
                flags: r.flags,
              }))
            : [],
          isAdmin: claims.adm,
          isAuthor: claims.adm,
        };
        setUser(u);
      } catch (e) {
        console.error("Failed to parse JWT", e);
        localStorage.removeItem("jwt");
      }
    }
  }, []);

  useEffect(() => {
    if (!user?.isAdmin) return;

    const token = localStorage.getItem("jwt");
    if (!token) return;

    const headers: Record<string, string> = {
      Authorization: `Bearer ${token}`,
    };

    fetch("/upload", { headers })
      .then((res) => {
        if (!res.ok) return null;
        return res.json();
      })
      .then((data) => {
        if (!data) return;

        const roles: Record<string, CachedRoleName> = {};
        const channels: Record<string, string> = {};

        if (Array.isArray(data.roles)) {
          data.roles.forEach((r: any) => {
            const id = String(r?.id ?? "").trim();
            const name = String(r?.name ?? "").trim();
            if (!id || !name) return;
            roles[id] = {
              name,
              color: r.color,
              managed: r.managed,
            };
          });
        }

        if (Array.isArray(data.channels)) {
          data.channels.forEach((c: any) => {
            const id = String(c?.id ?? "").trim();
            const name = String(c?.name ?? "").trim();
            if (!id || !name) return;
            channels[id] = name;
          });
        }

        mergeNameCache({
          roles,
          channels,
          guildName:
            typeof data.guild_name === "string" ? data.guild_name : undefined,
        });
      })
      .catch((err) => {
        console.error("Failed to refresh role/channel cache", err);
      });
  }, [user?.isAdmin, mergeNameCache]);

  const hydratedPosts = useMemo(() => {
    return posts.map((post) => ({
      ...post,
      allowedRoles: (post.allowedRoles ?? []).map((role) => {
        const cached = nameCache.roles[role.id];
        if (!cached) return role;
        return {
          ...role,
          name: cached.name,
          color: cached.color ?? role.color,
        };
      }),
    }));
  }, [posts, nameCache.roles]);

  const roleOptions = useMemo(() => {
    return Object.entries(nameCache.roles)
      .map(([id, role]) => ({
        id,
        name: role.name,
        color: role.color,
        managed: role.managed,
      }))
      .sort((a, b) => a.name.localeCompare(b.name));
  }, [nameCache.roles]);

  const channelOptions = useMemo(() => {
    return Object.entries(nameCache.channels)
      .map(([id, name]) => ({ id, name }))
      .sort((a, b) => a.name.localeCompare(b.name));
  }, [nameCache.channels]);

  const resolveChannelName = useCallback(
    (id: string) => nameCache.channels[id] || id,
    [nameCache.channels],
  );

  // Fetch posts
  const [page, setPage] = useState(1);
  const [hasMore, setHasMore] = useState(true);
  const limit = 9;

  const loadPosts = useCallback(
    (pageNum: number, reset: boolean = false, sort: SortMode = sortMode) => {
      setLoading(true);
      const token = localStorage.getItem("jwt");
      const headers: Record<string, string> = {};
      if (token) {
        headers["Authorization"] = `Bearer ${token}`;
      }

      // If reset and we need all currently-visible items, compute the right limit
      const fetchLimit =
        reset && pageNum === 1 ? Math.max(limit, posts.length) : limit;

      fetch(`/posts?page=${pageNum}&limit=${fetchLimit}&sort=${sort}`, {
        headers,
      })
        .then((res) => res.json())
        .then((data) => {
          const items: Post[] = Array.isArray(data) ? data : [];
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

          const newPosts = reset ? items : [...posts, ...items];
          setPosts(newPosts);
          // Update cache for this sort mode
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
    [sortMode, posts, mergeNameCache],
  );

  // Initial Fetch
  useEffect(() => {
    setInitialLoading(true);
    sortCacheRef.current = {}; // clear cache on user change
    loadPosts(1, true);
    setPage(1);
  }, [user]);

  // When sort mode changes, use cache if available, otherwise fetch
  useEffect(() => {
    const cached = sortCacheRef.current[sortMode];
    if (cached && cached.length > 0) {
      setPosts(cached);
      setHasMore(cached.length % limit === 0); // rough heuristic
    } else {
      loadPosts(1, true, sortMode);
      setPage(1);
    }
  }, [sortMode]);

  const handleLoadMore = () => {
    if (!hasMore || loading) return;
    // Invalidate cache for both sort modes since we're extending the list
    sortCacheRef.current = {};
    const nextPage = page + 1;
    setPage(nextPage);
    loadPosts(nextPage, false, sortMode);
  };

  const [selectedPost, setSelectedPost] = useState<Post | null>(null);

  const hydratedSelectedPost = useMemo(() => {
    if (!selectedPost) return null;
    return {
      ...selectedPost,
      allowedRoles: (selectedPost.allowedRoles ?? []).map((role) => {
        const cached = nameCache.roles[role.id];
        if (!cached) return role;
        return {
          ...role,
          name: cached.name,
          color: cached.color ?? role.color,
        };
      }),
    };
  }, [selectedPost, nameCache.roles]);

  const selected = useMemo(() => {
    if (!selectedId) return null;
    return (
      hydratedPosts?.find((p) => p.postKey === selectedId) ??
      hydratedSelectedPost
    );
  }, [hydratedPosts, selectedId, hydratedSelectedPost]);

  useEffect(() => {
    if (!selectedId) {
      setSelectedPost(null);
      return;
    }
    if (selectedPost && selectedPost.postKey !== selectedId) {
      setSelectedPost(null);
    }
  }, [selectedId, selectedPost]);

  // If we have a selectedId but it's not in the loaded posts list, fetch it individually
  useEffect(() => {
    if (!selectedId || selected) return;
    // Prevent re-fetching if we already have it in selectedPost
    if (selectedPost && selectedPost.postKey === selectedId) return;

    const token = localStorage.getItem("jwt");
    const headers: Record<string, string> = {};
    if (token) headers["Authorization"] = `Bearer ${token}`;

    fetch(`/posts/${selectedId}`, { headers })
      .then((res) => {
        if (!res.ok) return null;
        return res.json();
      })
      .then((post) => {
        if (post) {
          setSelectedPost(post);
        }
      })
      .catch((err) => console.error("Failed to fetch individual post", err));
  }, [selectedId, selected, selectedPost]);

  const viewerRoleIds = useMemo(
    () => (user?.roles ?? []).map((r) => r.id),
    [user],
  );

  const filteredPosts = useMemo(() => {
    const base = hydratedPosts.slice();
    const byTag = base; // Tag filtering logic if tags field exists in future
    const s = q.trim().toLowerCase();
    if (!s) return byTag;
    return byTag.filter((p) => {
      const hay =
        `${p.title ?? ""} ${(p.description ?? "").slice(0, 120)}`.toLowerCase();
      return hay.includes(s);
    });
  }, [hydratedPosts, tagFilter, q]);

  const { settings } = useSettings();
  const defaultDocumentTitleRef = useRef(document.title);

  useEffect(() => {
    const heroTitle = settings.hero_title?.trim();
    document.title =
      heroTitle && heroTitle.length > 0
        ? heroTitle
        : defaultDocumentTitleRef.current;
  }, [settings.hero_title]);

  const canAccessPost = useMemo(() => {
    return (p: Post) => {
      if (user?.isAdmin) return true; // Admin/Author access
      if (settings.public_access) return true; // Global Public Access
      const postRoleIds = p.allowedRoles.map((r) => r.id);
      if (postRoleIds.length === 0) return true;
      return intersect(postRoleIds, viewerRoleIds);
    };
  }, [user, viewerRoleIds, settings.public_access]);

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
    if (token) headers["Authorization"] = `Bearer ${token}`;

    try {
      const res = await fetch("/posts", {
        method: "POST",
        headers,
        body: formData,
      });

      if (!res.ok) {
        const err = await res.json().catch(() => ({ error: res.statusText }));
        alert(`Upload failed: ${err.error || res.statusText}`);
        return;
      }

      const created = await res.json();
      console.log("Post created:", created);
      // Refresh posts to show the new one
      loadPosts(1, true);
      setPage(1);
    } catch (err) {
      console.error("Upload error:", err);
      alert("Upload failed. Check the console for details.");
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
    if (token) headers["Authorization"] = `Bearer ${token}`;

    try {
      const res = await fetch(`/posts/${postKey}`, {
        method: "PATCH",
        headers,
        body: formData,
      });

      if (!res.ok) {
        const err = await res.json().catch(() => ({ error: res.statusText }));
        alert(`Edit failed: ${err.error || res.statusText}`);
        return;
      }

      console.log("Post updated");
      setEditingPost(null);
      loadPosts(1, true);
      setPage(1);
    } catch (err) {
      console.error("Edit error:", err);
      alert("Edit failed. Check the console for details.");
    }
  }

  async function handleDelete(postKey: string) {
    const token = localStorage.getItem("jwt");
    const headers: Record<string, string> = {};
    if (token) headers["Authorization"] = `Bearer ${token}`;

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

      console.log("Post deleted");
      navigate("/");
      loadPosts(1, true);
      setPage(1);
    } catch (err) {
      console.error("Delete error:", err);
      alert("Delete failed. Check the console for details.");
    }
  }

  return (
    <>
      <SettingsModal
        open={settingsOpen}
        onClose={() => setSettingsOpen(false)}
      />
      {membershipOpen && (
        <MembershipModal onClose={() => setMembershipOpen(false)} />
      )}
      <div className={UI.page}>
        <LoginModal
          open={loginOpen}
          onClose={() => setLoginOpen(false)}
          onLogin={(u) => {
            setUser(u);
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
              guildName={nameCache.guildName}
              view={view === "post" ? "post" : "gallery"} // Fallback to gallery styles if 404
              setView={(v) => {
                if (v === "gallery") navigate("/");
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
                guildName={nameCache.guildName}
              />
            </div>
          ) : null}

          <div className="mt-6 flex flex-col gap-4 lg:flex-row relative items-start">
            {/* Left: MAIN */}
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

            {/* Right: SIDEBAR - Do not show on 404? or default? */}
            {view !== "not-found" && (
              <div className={cn("w-full lg:w-80")}>
                <div className="sticky top-4">
                  {view === "gallery" ? (
                    <ProfileSidebar
                      user={user}
                      onLogin={() => setLoginOpen(true)}
                    />
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
